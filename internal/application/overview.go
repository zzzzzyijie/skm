package application

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zzzzzyijie/skm/internal/adapter"
	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	gitSource "github.com/zzzzzyijie/skm/internal/source"
)

type Dashboard struct {
	SkillCount     int            `json:"skillCount"`
	ActivatedCount int            `json:"activatedCount"`
	SourceCount    int            `json:"sourceCount"`
	RecentSkills   []domain.Skill `json:"recentSkills"`
}

type AddSourceInput struct {
	Input string   `json:"input"`
	Name  string   `json:"name"`
	URL   string   `json:"url"`
	Ref   string   `json:"ref"`
	Paths []string `json:"paths"`
	Tags  []string `json:"tags"`
}

type AddSourceResult struct {
	Source domain.Source  `json:"source"`
	Skills []domain.Skill `json:"skills"`
}

type ProjectView struct {
	domain.Project
	Exists          bool           `json:"exists"`
	Access          string         `json:"access"`
	AccessMessage   string         `json:"accessMessage,omitempty"`
	ActivationCount int            `json:"activationCount"`
	SkillCount      int            `json:"skillCount"`
	AgentCounts     map[string]int `json:"agentCounts"`
}

type WorkspaceView struct {
	Configured bool                    `json:"configured"`
	Config     *domain.WorkspaceConfig `json:"config,omitempty"`
	State      *domain.WorkspaceState  `json:"state,omitempty"`
}

type DoctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (s *Service) Dashboard() (Dashboard, error) {
	catalogValue, err := s.Store.LoadCatalog()
	if err != nil {
		return Dashboard{}, err
	}
	state, err := s.Store.LoadState()
	if err != nil {
		return Dashboard{}, err
	}
	sources, err := s.Store.LoadSources()
	if err != nil {
		return Dashboard{}, err
	}
	activated := make(map[string]struct{})
	for _, activation := range state.Activations {
		if activation.Placement == domain.PlacementUser {
			activated[activation.SkillID] = struct{}{}
		}
	}
	skills := append([]domain.Skill(nil), catalogValue.Skills...)
	sort.Slice(skills, func(i, j int) bool { return skills[i].AddedAt.After(skills[j].AddedAt) })
	if len(skills) > 5 {
		skills = skills[:5]
	}
	return Dashboard{SkillCount: len(catalogValue.Skills), ActivatedCount: len(activated), SourceCount: len(sources.Sources), RecentSkills: skills}, nil
}

func (s *Service) ListSources() ([]domain.Source, error) {
	values, err := s.Store.LoadSources()
	if values.Sources == nil {
		values.Sources = []domain.Source{}
	}
	return values.Sources, err
}

func (s *Service) AddSource(input AddSourceInput) (AddSourceResult, error) {
	rawInput := strings.TrimSpace(input.Input)
	if rawInput == "" {
		rawInput = strings.TrimSpace(input.URL)
	}
	parsed, err := gitSource.ParseInstallInput(rawInput)
	if err != nil {
		return AddSourceResult{}, err
	}
	if len(parsed.RequestedSkills) > 0 && len(input.Paths) > 0 {
		return AddSourceResult{}, fmt.Errorf("paths cannot be combined with --skill selections")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = parsed.SuggestedName
	}
	var result AddSourceResult
	err = s.withLock(func() error {
		var addErr error
		result.Source, result.Skills, addErr = gitSource.NewGitManager(s.Store, catalog.New(s.Store)).AddSelected(
			domain.Source{Name: name, URL: parsed.URL, Ref: input.Ref, Paths: input.Paths, Tags: input.Tags},
			parsed.RequestedSkills,
		)
		return addErr
	})
	return result, err
}

func (s *Service) ListProjects() ([]ProjectView, error) {
	projects, err := s.Store.LoadProjects()
	if err != nil {
		return nil, err
	}
	state, err := s.Store.LoadState()
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for _, activation := range state.Activations {
		if activation.Placement != domain.PlacementProject {
			continue
		}
		root := filepath.Clean(activation.ProjectRoot)
		counts[root]++
	}
	result := make([]ProjectView, 0, len(projects.Projects))
	for _, project := range projects.Projects {
		exists, access, accessMessage := inspectProjectAccess(project.Path)
		root := filepath.Clean(project.Path)
		scan := ProjectScan{AgentCounts: map[string]int{}, Agents: []ProjectScanAgent{}, Skills: []ProjectScanSkill{}}
		if access == "available" {
			var scanErr error
			scan, scanErr = scanProjectSkills(project.Path)
			if scanErr != nil {
				scan = ProjectScan{AgentCounts: map[string]int{}, Agents: []ProjectScanAgent{}, Skills: []ProjectScanSkill{}}
			}
		}
		result = append(result, ProjectView{
			Project: project, Exists: exists, Access: access, AccessMessage: accessMessage, ActivationCount: counts[root],
			SkillCount: scan.SkillCount, AgentCounts: scan.AgentCounts,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func inspectProjectAccess(path string) (bool, string, string) {
	info, err := os.Stat(path)
	if err != nil {
		return false, projectAccessForError(err), err.Error()
	}
	if !info.IsDir() {
		return false, "unavailable", "registered project path is not a directory"
	}
	directory, err := os.Open(path)
	if err != nil {
		return true, projectAccessForError(err), err.Error()
	}
	defer directory.Close()
	if _, err := directory.Readdirnames(1); err != nil && err != io.EOF {
		return true, projectAccessForError(err), err.Error()
	}
	return true, "available", ""
}

func projectAccessForError(err error) string {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "missing"
	case errors.Is(err, fs.ErrPermission):
		return "permission-denied"
	default:
		return "unavailable"
	}
}

func (s *Service) GetWorkspace() (WorkspaceView, error) {
	config, err := s.Store.LoadWorkspaceConfig()
	if err != nil {
		return WorkspaceView{}, err
	}
	if strings.TrimSpace(config.URL) == "" {
		return WorkspaceView{}, nil
	}
	state, err := s.Store.LoadWorkspaceState()
	if err != nil {
		return WorkspaceView{}, err
	}
	return WorkspaceView{Configured: true, Config: &config, State: &state}, nil
}

func (s *Service) Doctor() ([]DoctorCheck, error) {
	checks := []DoctorCheck{}
	if info, err := os.Stat(s.Store.Paths.Home); err != nil || !info.IsDir() {
		checks = append(checks, DoctorCheck{Name: "skm-home", Status: "error", Message: fmt.Sprintf("not accessible: %v", err)})
	} else {
		checks = append(checks, DoctorCheck{Name: "skm-home", Status: "ok", Message: s.Store.Paths.Home})
	}
	if path, err := exec.LookPath("git"); err != nil {
		checks = append(checks, DoctorCheck{Name: "git", Status: "optional", Message: "required for Git sources"})
	} else {
		checks = append(checks, DoctorCheck{Name: "git", Status: "ok", Message: path})
	}
	config, err := s.Store.LoadConfig()
	if err != nil {
		return nil, err
	}
	customRoots := make(map[domain.Agent]string, len(config.Agents))
	for _, definition := range config.Agents {
		customRoots[definition.ID] = definition.SkillsPath
	}
	for _, agent := range configuredAgents(config.Defaults.Agents, config.Agents) {
		target, targetErr := adapter.Target(agent, domain.PlacementUser, s.Store.Paths.UserHome, "", "<skill>", customRoots)
		if targetErr != nil {
			checks = append(checks, DoctorCheck{Name: "agent-" + string(agent), Status: "error", Message: targetErr.Error()})
			continue
		}
		checks = append(checks, DoctorCheck{Name: "agent-" + string(agent), Status: "ok", Message: filepath.Dir(target)})
	}
	skills, err := s.ListSkills(nil)
	if err != nil {
		return nil, err
	}
	unhealthy := 0
	for _, value := range skills {
		if value.Health != "available" {
			unhealthy++
		}
	}
	status := "ok"
	if unhealthy > 0 {
		status = "warning"
	}
	checks = append(checks, DoctorCheck{Name: "library", Status: status, Message: fmt.Sprintf("%d Skills, %d require attention", len(skills), unhealthy)})
	projects, err := s.ListProjects()
	if err != nil {
		return nil, err
	}
	missing := 0
	for _, project := range projects {
		if !project.Exists {
			missing++
		}
	}
	projectStatus := "ok"
	if missing > 0 {
		projectStatus = "warning"
	}
	checks = append(checks, DoctorCheck{Name: "projects", Status: projectStatus, Message: fmt.Sprintf("%d registered, %d missing", len(projects), missing)})
	sources, err := s.ListSources()
	if err != nil {
		return nil, err
	}
	checks = append(checks, DoctorCheck{Name: "sources", Status: "ok", Message: fmt.Sprintf("%d configured", len(sources))})
	return checks, nil
}
