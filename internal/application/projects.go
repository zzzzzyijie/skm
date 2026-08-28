package application

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zzzzzyijie/skm/internal/adapter"
	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/planner"
	"github.com/zzzzzyijie/skm/internal/skill"
)

type AddProjectInput struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type ProjectScanAgent struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	SkillCount int    `json:"skillCount"`
	Available  bool   `json:"available"`
}

type ProjectScanSkill struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	Agents         []string          `json:"agents"`
	Paths          map[string]string `json:"paths"`
	Hash           string            `json:"hash,omitempty"`
	LibrarySkillID string            `json:"librarySkillId,omitempty"`
	Status         string            `json:"status"`
	Issues         []string          `json:"issues,omitempty"`
}

type ProjectScan struct {
	ScannedAt   time.Time          `json:"scannedAt"`
	SkillCount  int                `json:"skillCount"`
	AgentCounts map[string]int     `json:"agentCounts"`
	Agents      []ProjectScanAgent `json:"agents"`
	Skills      []ProjectScanSkill `json:"skills"`
	Errors      []string           `json:"errors,omitempty"`
}

type ProjectDetails struct {
	Project     domain.Project      `json:"project"`
	Exists      bool                `json:"exists"`
	Activations []domain.Activation `json:"activations"`
	Scan        ProjectScan         `json:"scan"`
	Plan        domain.Plan         `json:"plan"`
	Manifest    domain.Catalog      `json:"manifest"`
}

type ProjectDeployInput struct {
	Project string          `json:"project"`
	Skill   string          `json:"skill"`
	Agents  []string        `json:"agents"`
	Mode    domain.LinkMode `json:"mode"`
	DryRun  bool            `json:"dryRun"`
}

type ProjectDeployResult struct {
	Project domain.Project `json:"project"`
	Skill   domain.Skill   `json:"skill"`
	Plan    domain.Plan    `json:"plan"`
	Applied bool           `json:"applied"`
}

type ProjectUnlinkInput struct {
	Project string   `json:"project"`
	Skill   string   `json:"skill"`
	Agents  []string `json:"agents"`
	Force   bool     `json:"force"`
}

type ProjectMigrateInput struct {
	Project      string          `json:"project"`
	Skill        string          `json:"skill"`
	Agent        string          `json:"agent"`
	Mode         domain.LinkMode `json:"mode"`
	RemoveSource bool            `json:"removeSource"`
	Tags         []string        `json:"tags"`
}

type ProjectMigrateResult struct {
	Project      domain.Project  `json:"project"`
	Skill        domain.Skill    `json:"skill"`
	Mode         domain.LinkMode `json:"mode"`
	RemovedPaths []string        `json:"removedPaths"`
}

type projectSkillRoot struct {
	ID, Label, Path string
}

type projectSkillSource struct {
	Agent, Path string
	Document    skill.Document
}

func (s *Service) AddProject(input AddProjectInput) (domain.Project, error) {
	path, err := canonicalProjectPath(input.Path)
	if err != nil {
		return domain.Project{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = filepath.Base(path)
	}
	if err := validateProjectID(name); err != nil {
		return domain.Project{}, err
	}
	now := time.Now().UTC()
	project := domain.Project{ID: name, Path: path, AddedAt: now, UpdatedAt: now}
	err = s.withLock(func() error {
		projects, err := s.Store.LoadProjects()
		if err != nil {
			return err
		}
		for _, existing := range projects.Projects {
			if existing.ID == project.ID {
				return fmt.Errorf("project %q is already registered", project.ID)
			}
			if filepath.Clean(existing.Path) == filepath.Clean(project.Path) {
				return fmt.Errorf("project path is already registered as %s", existing.ID)
			}
		}
		projects.Projects = append(projects.Projects, project)
		return s.Store.SaveProjects(projects)
	})
	return project, err
}

func (s *Service) GetProject(query string) (ProjectDetails, error) {
	project, err := s.resolveProject(query)
	if err != nil {
		return ProjectDetails{}, err
	}
	state, err := s.Store.LoadState()
	if err != nil {
		return ProjectDetails{}, err
	}
	scan, err := scanProjectSkills(project.Path)
	if err != nil {
		return ProjectDetails{}, err
	}
	library, err := s.Store.LoadCatalog()
	if err != nil {
		return ProjectDetails{}, err
	}
	markProjectLibrarySkills(&scan, library.Skills)
	plan, err := s.projectPlan(project.Path)
	if err != nil {
		return ProjectDetails{}, err
	}
	projectStore, err := s.storeForProject(project)
	if err != nil {
		return ProjectDetails{}, err
	}
	manifest, err := projectStore.LoadProjectCatalog()
	if err != nil {
		return ProjectDetails{}, err
	}
	if manifest.Skills == nil {
		manifest.Skills = []domain.Skill{}
	}
	if manifest.Dependencies == nil {
		manifest.Dependencies = []domain.ProjectDependency{}
	}
	info, statErr := os.Stat(project.Path)
	activations := projectActivations(state, project.Path)
	if activations == nil {
		activations = []domain.Activation{}
	}
	if plan.Operations == nil {
		plan.Operations = []domain.Operation{}
	}
	return ProjectDetails{Project: project, Exists: statErr == nil && info.IsDir(), Activations: activations, Scan: scan, Plan: plan, Manifest: manifest}, nil
}

func (s *Service) DeployProject(input ProjectDeployInput) (ProjectDeployResult, error) {
	if strings.TrimSpace(input.Skill) == "" {
		return ProjectDeployResult{}, fmt.Errorf("skill is required")
	}
	mode := input.Mode.Effective()
	if mode != domain.ModeSymlink && mode != domain.ModeCopy {
		return ProjectDeployResult{}, fmt.Errorf("mode must be symlink or copy")
	}
	var result ProjectDeployResult
	err := s.withLock(func() error {
		project, err := s.resolveProject(input.Project)
		if err != nil {
			return err
		}
		agents, err := s.parseProjectAgents(project.Path, input.Agents)
		if err != nil {
			return err
		}
		value, err := catalog.New(s.Store).ResolveLibrary(input.Skill)
		if err != nil {
			return err
		}
		state, err := s.Store.LoadState()
		if err != nil {
			return err
		}
		if err := ensureProjectMode(state, project.Path, value.ID, mode); err != nil {
			return err
		}
		engine := planner.New(s.Store)
		engine.AddActivations(&state, []domain.Skill{value}, domain.PlacementProject, project.Path, agents, mode)
		skills, err := s.Store.LoadAllSkills()
		if err != nil {
			return err
		}
		plan, err := engine.BuildScoped(skills, state, domain.PlacementProject, project.Path)
		if err != nil {
			return err
		}
		result = ProjectDeployResult{Project: project, Skill: value, Plan: plan, Applied: !input.DryRun}
		if input.DryRun {
			return nil
		}
		return engine.Apply(plan, &state)
	})
	return result, err
}

func (s *Service) UnlinkProject(input ProjectUnlinkInput) (map[string]any, error) {
	if strings.TrimSpace(input.Skill) == "" {
		return nil, fmt.Errorf("skill is required")
	}
	var project domain.Project
	err := s.withLock(func() error {
		var err error
		project, err = s.resolveProject(input.Project)
		if err != nil {
			return err
		}
		state, err := s.Store.LoadState()
		if err != nil {
			return err
		}
		skillID, err := resolveProjectSkill(s.Store, state, project.Path, input.Skill)
		if err != nil {
			return err
		}
		agentMap := make(map[domain.Agent]struct{})
		if len(input.Agents) > 0 {
			agents, err := s.parseProjectAgents(project.Path, input.Agents)
			if err != nil {
				return err
			}
			for _, agent := range agents {
				agentMap[agent] = struct{}{}
			}
		}
		return planner.New(s.Store).Disable(&state, map[string]struct{}{skillID: {}}, domain.PlacementProject, project.Path, agentMap, input.Force)
	})
	return map[string]any{"project": project, "skill": input.Skill, "status": "ok"}, err
}

func (s *Service) UnregisterProject(query string) (domain.Project, error) {
	var removed domain.Project
	err := s.withLock(func() error {
		projects, err := s.Store.LoadProjects()
		if err != nil {
			return err
		}
		removed, err = resolveProjectFromList(projects.Projects, query)
		if err != nil {
			return err
		}
		state, err := s.Store.LoadState()
		if err != nil {
			return err
		}
		if len(projectActivations(state, removed.Path)) > 0 {
			return fmt.Errorf("project %s still has managed Activations; unlink its Skills first", removed.ID)
		}
		for _, deployment := range state.Deployments {
			if deployment.Placement == domain.PlacementProject && filepath.Clean(deployment.ProjectRoot) == filepath.Clean(removed.Path) {
				return fmt.Errorf("project %s still has managed Deployments; unlink its Skills first", removed.ID)
			}
		}
		remaining := projects.Projects[:0]
		for _, project := range projects.Projects {
			if project.ID != removed.ID {
				remaining = append(remaining, project)
			}
		}
		projects.Projects = remaining
		return s.Store.SaveProjects(projects)
	})
	return removed, err
}

func (s *Service) MigrateProjectSkill(input ProjectMigrateInput) (ProjectMigrateResult, error) {
	if input.Mode != domain.ModeSymlink && input.Mode != domain.ModeCopy {
		return ProjectMigrateResult{}, fmt.Errorf("mode must be symlink or copy")
	}
	if input.RemoveSource && input.Mode != domain.ModeCopy {
		return ProjectMigrateResult{}, fmt.Errorf("project source can only be removed after a copy")
	}
	result := ProjectMigrateResult{Mode: input.Mode, RemovedPaths: []string{}}
	err := s.withLock(func() error {
		project, err := s.resolveProject(input.Project)
		if err != nil {
			return err
		}
		sources, err := loadProjectSkillSources(project.Path, input.Skill)
		if err != nil {
			return err
		}
		selected, err := selectProjectSkillSource(sources, input.Agent)
		if err != nil {
			return err
		}
		if input.RemoveSource {
			if err := s.validateProjectSkillRemoval(project.Path, sources, selected.Document.Hash); err != nil {
				return err
			}
		}
		imported, err := catalog.New(s.Store).ImportProject(selected.Document, project.Path, domain.Agent(selected.Agent), input.Mode, input.Tags)
		if err != nil {
			return err
		}
		result.Project, result.Skill = project, imported
		if input.RemoveSource {
			for _, source := range sources {
				if err := os.RemoveAll(source.Path); err != nil {
					return fmt.Errorf("Skill was copied to %s, but removing project source %s failed: %w", imported.ID, source.Path, err)
				}
				result.RemovedPaths = append(result.RemovedPaths, source.Path)
			}
		}
		return nil
	})
	return result, err
}

func (s *Service) parseProjectAgents(projectRoot string, values []string) ([]domain.Agent, error) {
	if len(values) == 0 {
		roots, err := projectSkillRoots(projectRoot)
		if err != nil {
			return nil, err
		}
		for _, root := range roots {
			values = append(values, root.ID)
		}
		if len(values) == 0 {
			config, err := s.Store.LoadConfig()
			if err != nil {
				return nil, err
			}
			for _, agent := range config.Defaults.Agents {
				values = append(values, string(agent))
			}
		}
		if len(values) == 0 {
			values = []string{string(domain.AgentCodex)}
		}
	}
	seen := make(map[domain.Agent]bool)
	result := make([]domain.Agent, 0, len(values))
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			agent := domain.Agent(strings.TrimSpace(part))
			if !agent.ProjectValid() {
				return nil, fmt.Errorf("invalid project Agent %q", part)
			}
			if agent == "" || seen[agent] {
				continue
			}
			seen[agent] = true
			result = append(result, agent)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("at least one project Agent is required")
	}
	return result, nil
}

func (s *Service) resolveProject(query string) (domain.Project, error) {
	projects, err := s.Store.LoadProjects()
	if err != nil {
		return domain.Project{}, err
	}
	return resolveProjectFromList(projects.Projects, query)
}

func resolveProjectFromList(projects []domain.Project, query string) (domain.Project, error) {
	query = strings.TrimSpace(query)
	for _, project := range projects {
		if project.ID == query || filepath.Clean(project.Path) == filepath.Clean(query) {
			return project, nil
		}
	}
	return domain.Project{}, fmt.Errorf("registered project %q not found", query)
}

func scanProjectSkills(projectRoot string) (ProjectScan, error) {
	result := ProjectScan{ScannedAt: time.Now().UTC(), AgentCounts: map[string]int{}, Agents: []ProjectScanAgent{}, Skills: []ProjectScanSkill{}}
	roots, err := projectSkillRoots(projectRoot)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result, nil
	}
	byID := make(map[string]int)
	for _, root := range roots {
		result.AgentCounts[root.ID] = 0
		result.Agents = append(result.Agents, ProjectScanAgent{ID: root.ID, Label: root.Label, Available: true})
		entries, err := os.ReadDir(root.Path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", root.Label, err))
			continue
		}
		for _, entry := range entries {
			resolved, candidate, candidateErr := projectSkillDirectory(filepath.Join(root.Path, entry.Name()))
			if !candidate {
				continue
			}
			index, ok := byID[entry.Name()]
			if !ok {
				result.Skills = append(result.Skills, ProjectScanSkill{ID: entry.Name(), Name: entry.Name(), Status: "ok", Agents: []string{}, Paths: map[string]string{}})
				index = len(result.Skills) - 1
				byID[entry.Name()] = index
			}
			item := &result.Skills[index]
			item.Agents = append(item.Agents, root.ID)
			item.Paths[root.ID] = filepath.Join(root.Path, entry.Name())
			result.AgentCounts[root.ID]++
			if candidateErr != nil {
				addProjectScanIssue(item, root.Label, "error", candidateErr.Error())
				continue
			}
			document, err := skill.Validate(resolved)
			if err != nil {
				addProjectScanIssue(item, root.Label, "warning", err.Error())
				continue
			}
			if item.Name == item.ID {
				item.Name = document.Name
			}
			if item.Description == "" {
				item.Description = document.Description
			}
			if item.Hash == "" {
				item.Hash = document.Hash
			}
		}
	}
	for index := range result.Agents {
		result.Agents[index].SkillCount = result.AgentCounts[result.Agents[index].ID]
	}
	sort.Slice(result.Skills, func(i, j int) bool {
		return strings.ToLower(result.Skills[i].Name) < strings.ToLower(result.Skills[j].Name)
	})
	result.SkillCount = len(result.Skills)
	return result, nil
}

func projectSkillRoots(projectRoot string) ([]projectSkillRoot, error) {
	entries, err := os.ReadDir(projectRoot)
	if err != nil {
		return nil, err
	}
	result := make([]projectSkillRoot, 0)
	seen := make(map[string]bool)
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, ".") || len(name) == 1 {
			continue
		}
		path := filepath.Join(projectRoot, name, "skills")
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			continue
		}
		id := strings.TrimPrefix(name, ".")
		seen[id] = true
		result = append(result, projectSkillRoot{ID: id, Label: projectAgentLabel(id), Path: path})
	}
	for _, special := range []struct{ id, path string }{{"windsurf", filepath.Join(".codeium", "windsurf", "skills")}, {"opencode", filepath.Join(".config", "opencode", "skills")}} {
		if seen[special.id] {
			continue
		}
		path := filepath.Join(projectRoot, special.path)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			result = append(result, projectSkillRoot{ID: special.id, Label: projectAgentLabel(special.id), Path: path})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func projectSkillDirectory(path string) (string, bool, error) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", false, nil
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", false, nil
	}
	if _, err := os.Stat(filepath.Join(resolved, "SKILL.md")); os.IsNotExist(err) {
		return "", false, nil
	} else if err != nil {
		return resolved, true, err
	}
	return resolved, true, nil
}

func projectAgentLabel(id string) string {
	return adapter.DisplayName(domain.Agent(id))
}

func addProjectScanIssue(item *ProjectScanSkill, agentLabel, status, message string) {
	item.Issues = append(item.Issues, fmt.Sprintf("%s: %s", agentLabel, message))
	if item.Status == "error" || (item.Status == "warning" && status != "error") {
		return
	}
	item.Status = status
}

func markProjectLibrarySkills(scan *ProjectScan, library []domain.Skill) {
	for index := range scan.Skills {
		for _, value := range library {
			if value.Name == scan.Skills[index].Name || value.Name == scan.Skills[index].ID {
				scan.Skills[index].LibrarySkillID = value.ID
				break
			}
		}
	}
}

func (s *Service) projectPlan(projectRoot string) (domain.Plan, error) {
	state, err := s.Store.LoadState()
	if err != nil {
		return domain.Plan{}, err
	}
	skills, err := s.Store.LoadAllSkills()
	if err != nil {
		return domain.Plan{}, err
	}
	return planner.New(s.Store).BuildScoped(skills, state, domain.PlacementProject, projectRoot)
}

func projectActivations(state domain.State, projectRoot string) []domain.Activation {
	var result []domain.Activation
	for _, activation := range state.Activations {
		if activation.Placement == domain.PlacementProject && filepath.Clean(activation.ProjectRoot) == filepath.Clean(projectRoot) {
			result = append(result, activation)
		}
	}
	return result
}

func resolveProjectSkill(storage interface {
	LoadCatalog() (domain.Catalog, error)
}, state domain.State, projectRoot, query string) (string, error) {
	// Keep resolution local to the registered project. The concrete Store also
	// supports catalog.Manager; the interface makes this helper easy to test.
	for _, activation := range projectActivations(state, projectRoot) {
		if activation.SkillID == query || activation.Name == query {
			return activation.SkillID, nil
		}
	}
	catalogValue, err := storage.LoadCatalog()
	if err != nil {
		return "", err
	}
	for _, value := range catalogValue.Skills {
		if value.ID == query || value.Name == query {
			return value.ID, nil
		}
	}
	return "", fmt.Errorf("project Skill %q not found", query)
}

func ensureProjectMode(state domain.State, projectRoot, skillID string, mode domain.LinkMode) error {
	for _, activation := range projectActivations(state, projectRoot) {
		if activation.SkillID == skillID && activation.Mode.Effective() != mode.Effective() {
			return fmt.Errorf("project Skill %s already uses mode %s; unlink it before switching to %s", skillID, activation.Mode.Effective(), mode.Effective())
		}
	}
	return nil
}

func canonicalProjectPath(value string) (string, error) {
	path, err := filepath.Abs(strings.TrimSpace(value))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("project path %q: %w", value, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project path %q is not a directory", value)
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(canonical), nil
}

func validateProjectID(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("project name is required")
	}
	if strings.ContainsAny(value, `/\\`) {
		return fmt.Errorf("project name %q must not contain path separators", value)
	}
	return nil
}

func loadProjectSkillSources(projectRoot, skillID string) ([]projectSkillSource, error) {
	if skillID == "" || filepath.Base(skillID) != skillID || skillID == "." || skillID == ".." {
		return nil, fmt.Errorf("invalid project Skill %q", skillID)
	}
	roots, err := projectSkillRoots(projectRoot)
	if err != nil {
		return nil, err
	}
	result := make([]projectSkillSource, 0)
	for _, root := range roots {
		path := filepath.Join(root.Path, skillID)
		resolved, candidate, candidateErr := projectSkillDirectory(path)
		if !candidate {
			continue
		}
		if candidateErr != nil {
			return nil, candidateErr
		}
		document, err := skill.Validate(resolved)
		if err != nil {
			return nil, err
		}
		result = append(result, projectSkillSource{Agent: root.ID, Path: path, Document: document})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("readable project Skill %q not found", skillID)
	}
	return result, nil
}

func selectProjectSkillSource(sources []projectSkillSource, agent string) (projectSkillSource, error) {
	agent = strings.TrimSpace(agent)
	if agent == "" && len(sources) == 1 {
		return sources[0], nil
	}
	for _, source := range sources {
		if source.Agent == agent {
			return source, nil
		}
	}
	if agent == "" {
		return projectSkillSource{}, fmt.Errorf("agent is required because the Skill exists in multiple Agent directories")
	}
	return projectSkillSource{}, fmt.Errorf("project Skill was not found for Agent %q", agent)
}

func (s *Service) validateProjectSkillRemoval(projectRoot string, sources []projectSkillSource, selectedHash string) error {
	for _, source := range sources {
		if source.Document.Hash != selectedHash {
			return fmt.Errorf("cannot move Skill: Agent copies have different content")
		}
	}
	state, err := s.Store.LoadState()
	if err != nil {
		return err
	}
	for _, deployment := range state.Deployments {
		if deployment.Placement != domain.PlacementProject || filepath.Clean(deployment.ProjectRoot) != filepath.Clean(projectRoot) {
			continue
		}
		for _, source := range sources {
			if filepath.Clean(deployment.Target) == filepath.Clean(source.Path) {
				return fmt.Errorf("cannot move an SKM-managed project Skill; unlink it first")
			}
		}
	}
	return nil
}
