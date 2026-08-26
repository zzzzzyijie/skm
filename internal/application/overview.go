package application

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

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
	agentCounts := make(map[string]map[string]int)
	for _, activation := range state.Activations {
		if activation.Placement != domain.PlacementProject {
			continue
		}
		root := filepath.Clean(activation.ProjectRoot)
		counts[root]++
		if agentCounts[root] == nil {
			agentCounts[root] = make(map[string]int)
		}
		for _, agent := range activation.Agents {
			agentCounts[root][string(agent)]++
		}
	}
	result := make([]ProjectView, 0, len(projects.Projects))
	for _, project := range projects.Projects {
		info, statErr := os.Stat(project.Path)
		root := filepath.Clean(project.Path)
		agents := agentCounts[root]
		if agents == nil {
			agents = map[string]int{}
		}
		result = append(result, ProjectView{
			Project: project, Exists: statErr == nil && info.IsDir(), ActivationCount: counts[root],
			SkillCount: counts[root], AgentCounts: agents,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
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
	checks := []DoctorCheck{{Name: "skm-home", Status: "ok", Message: s.Store.Paths.Home}}
	if path, err := exec.LookPath("git"); err != nil {
		checks = append(checks, DoctorCheck{Name: "git", Status: "optional", Message: "required for Git sources"})
	} else {
		checks = append(checks, DoctorCheck{Name: "git", Status: "ok", Message: path})
	}
	return checks, nil
}
