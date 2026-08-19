package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/planner"
	"github.com/zzzzzyijie/skm/internal/skill"
	"github.com/zzzzzyijie/skm/internal/store"
)

type projectRow struct {
	domain.Project
	Exists          bool           `json:"exists"`
	ActivationCount int            `json:"activationCount"`
	SkillCount      int            `json:"skillCount"`
	AgentCounts     map[string]int `json:"agentCounts"`
}

type projectDetails struct {
	Project     domain.Project      `json:"project"`
	Exists      bool                `json:"exists"`
	Activations []domain.Activation `json:"activations"`
	Scan        projectScan         `json:"scan"`
	Plan        domain.Plan         `json:"plan"`
}

type projectScan struct {
	ScannedAt   time.Time          `json:"scannedAt"`
	SkillCount  int                `json:"skillCount"`
	AgentCounts map[string]int     `json:"agentCounts"`
	Agents      []projectScanAgent `json:"agents"`
	Skills      []projectScanSkill `json:"skills"`
	Errors      []string           `json:"errors,omitempty"`
}

type projectScanAgent struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	SkillCount int    `json:"skillCount"`
}

type projectScanSkill struct {
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

type projectSkillDetails struct {
	ID        string                 `json:"id"`
	Documents []projectSkillDocument `json:"documents"`
}

type projectSkillDocument struct {
	Agent       string         `json:"agent"`
	Path        string         `json:"path"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata"`
	Body        string         `json:"body"`
	Hash        string         `json:"hash"`
}

type projectSkillSource struct {
	Agent    string
	Path     string
	Document skill.Document
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.LoadProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	state, err := s.store.LoadState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	rows := make([]projectRow, 0, len(projects.Projects))
	counts := make(map[string]int)
	for _, activation := range state.Activations {
		if activation.Placement == domain.PlacementProject {
			counts[filepath.Clean(activation.ProjectRoot)]++
		}
	}
	for _, project := range projects.Projects {
		info, statErr := os.Stat(project.Path)
		scan, scanErr := scanProjectSkills(project.Path)
		if scanErr != nil {
			scan = projectScan{AgentCounts: map[string]int{}, Agents: []projectScanAgent{}}
		}
		rows = append(rows, projectRow{
			Project:         project,
			Exists:          statErr == nil && info.IsDir(),
			ActivationCount: counts[filepath.Clean(project.Path)],
			SkillCount:      scan.SkillCount,
			AgentCounts:     scan.AgentCounts,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) handleAddProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(body.Path) == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("path is required"))
		return
	}
	path, err := canonicalProjectPath(body.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = filepath.Base(path)
	}
	if err := validateProjectID(name); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	project := domain.Project{ID: name, Path: path, AddedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	err = s.withLock(func() error {
		projects, loadErr := s.store.LoadProjects()
		if loadErr != nil {
			return loadErr
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
		return s.store.SaveProjects(projects)
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (s *Server) handleShowProject(w http.ResponseWriter, r *http.Request) {
	project, err := s.resolveProject(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	details, err := s.projectDetails(project)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, details)
}

func (s *Server) handleShowProjectSkill(w http.ResponseWriter, r *http.Request) {
	project, err := s.resolveProject(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	skillID := r.PathValue("skill")
	if skillID == "" || filepath.Base(skillID) != skillID || skillID == "." || skillID == ".." {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid project skill %q", skillID))
		return
	}

	roots, rootsErr := projectSkillRoots(project.Path)
	if rootsErr != nil {
		writeError(w, http.StatusInternalServerError, rootsErr)
		return
	}
	details := projectSkillDetails{ID: skillID, Documents: make([]projectSkillDocument, 0, len(roots))}
	selectedAgent := strings.TrimSpace(r.URL.Query().Get("agent"))
	for _, agent := range roots {
		if selectedAgent != "" && agent.ID != selectedAgent {
			continue
		}
		resolved, candidate, candidateErr := projectSkillDirectory(filepath.Join(agent.Path, skillID))
		if !candidate || candidateErr != nil {
			continue
		}
		document, validateErr := skill.Validate(resolved)
		if validateErr != nil {
			continue
		}
		details.Documents = append(details.Documents, projectSkillDocument{
			Agent:       agent.ID,
			Path:        filepath.Join(agent.Path, skillID),
			Name:        document.Name,
			Description: document.Description,
			Metadata:    document.Metadata,
			Body:        document.Body,
			Hash:        document.Hash,
		})
	}
	if len(details.Documents) == 0 {
		if selectedAgent != "" {
			writeError(w, http.StatusNotFound, fmt.Errorf("readable project Skill %q not found for Agent %q", skillID, selectedAgent))
			return
		}
		writeError(w, http.StatusNotFound, fmt.Errorf("readable project Skill %q not found", skillID))
		return
	}
	writeJSON(w, http.StatusOK, details)
}

func (s *Server) handleMigrateProjectSkill(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Agent        string          `json:"agent"`
		Mode         domain.LinkMode `json:"mode"`
		RemoveSource bool            `json:"removeSource"`
		Tags         []string        `json:"tags"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if body.Mode != domain.ModeSymlink && body.Mode != domain.ModeCopy {
		writeError(w, http.StatusBadRequest, fmt.Errorf("mode must be symlink or copy"))
		return
	}
	if body.RemoveSource && body.Mode != domain.ModeCopy {
		writeError(w, http.StatusBadRequest, fmt.Errorf("project source can only be removed after a copy"))
		return
	}

	var project domain.Project
	var imported domain.Skill
	removedPaths := make([]string, 0)
	err := s.withLock(func() error {
		var err error
		project, err = s.resolveProject(r.PathValue("id"))
		if err != nil {
			return err
		}
		sources, err := loadProjectSkillSources(project.Path, r.PathValue("skill"))
		if err != nil {
			return err
		}
		selected, err := selectProjectSkillSource(sources, body.Agent)
		if err != nil {
			return err
		}
		if body.RemoveSource {
			if err := s.validateProjectSkillRemoval(project.Path, sources, selected.Document.Hash); err != nil {
				return err
			}
		}
		imported, err = catalog.New(s.store).ImportProject(selected.Document, project.Path, domain.Agent(selected.Agent), body.Mode, body.Tags)
		if err != nil {
			return err
		}
		if body.RemoveSource {
			for _, source := range sources {
				if err := os.RemoveAll(source.Path); err != nil {
					return fmt.Errorf("Skill was copied to %s, but removing project source %s failed: %w", imported.ID, source.Path, err)
				}
				removedPaths = append(removedPaths, source.Path)
			}
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"project": project, "skill": imported, "mode": body.Mode,
		"removedPaths": removedPaths,
	})
}

func loadProjectSkillSources(projectRoot, skillID string) ([]projectSkillSource, error) {
	if skillID == "" || filepath.Base(skillID) != skillID || skillID == "." || skillID == ".." {
		return nil, fmt.Errorf("invalid project Skill %q", skillID)
	}
	roots, err := projectSkillRoots(projectRoot)
	if err != nil {
		return nil, err
	}
	result := make([]projectSkillSource, 0, len(roots))
	for _, root := range roots {
		path := filepath.Join(root.Path, skillID)
		resolved, candidate, candidateErr := projectSkillDirectory(path)
		if !candidate {
			continue
		}
		if candidateErr != nil {
			return nil, fmt.Errorf("read %s Skill %s: %w", root.Label, skillID, candidateErr)
		}
		document, validateErr := skill.Validate(resolved)
		if validateErr != nil {
			return nil, fmt.Errorf("validate %s Skill %s: %w", root.Label, skillID, validateErr)
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
	if agent == "" {
		if len(sources) == 1 {
			return sources[0], nil
		}
		return projectSkillSource{}, fmt.Errorf("agent is required because the Skill exists in multiple Agent directories")
	}
	for _, source := range sources {
		if source.Agent == agent {
			return source, nil
		}
	}
	return projectSkillSource{}, fmt.Errorf("project Skill was not found for Agent %q", agent)
}

func (s *Server) validateProjectSkillRemoval(projectRoot string, sources []projectSkillSource, selectedHash string) error {
	for _, source := range sources {
		if source.Document.Hash != selectedHash {
			return fmt.Errorf("cannot move Skill: Agent copies have different content; copy one source without removing the project originals")
		}
	}
	state, err := s.store.LoadState()
	if err != nil {
		return err
	}
	for _, deployment := range state.Deployments {
		if deployment.Placement != domain.PlacementProject || filepath.Clean(deployment.ProjectRoot) != filepath.Clean(projectRoot) {
			continue
		}
		for _, source := range sources {
			if filepath.Clean(deployment.Target) == filepath.Clean(source.Path) {
				return fmt.Errorf("cannot move an SKM-managed project Skill; unlink it first or copy it without removing the project source")
			}
		}
	}
	return nil
}

func (s *Server) handleProjectStatus(w http.ResponseWriter, r *http.Request) {
	project, err := s.resolveProject(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	plan, err := s.projectPlan(project.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": project, "plan": plan})
}

func (s *Server) handleProjectLink(w http.ResponseWriter, r *http.Request) {
	s.handleProjectDeploy(w, r, domain.ModeSymlink)
}

func (s *Server) handleProjectCopy(w http.ResponseWriter, r *http.Request) {
	s.handleProjectDeploy(w, r, domain.ModeCopy)
}

func (s *Server) handleProjectDeploy(w http.ResponseWriter, r *http.Request, mode domain.LinkMode) {
	var body struct {
		Skill  string   `json:"skill"`
		Agents []string `json:"agents"`
		DryRun bool     `json:"dryRun"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(body.Skill) == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("skill is required"))
		return
	}
	var project domain.Project
	var value domain.Skill
	var plan domain.Plan
	var agents []domain.Agent
	var err error
	err = s.withLock(func() error {
		project, err = s.resolveProject(r.PathValue("id"))
		if err != nil {
			return err
		}
		agents, err = parseProjectAgents(project.Path, body.Agents)
		if err != nil {
			return err
		}
		value, err = catalog.New(s.store).ResolveLibrary(body.Skill)
		if err != nil {
			return err
		}
		state, stateErr := s.store.LoadState()
		if stateErr != nil {
			return stateErr
		}
		if err := ensureProjectMode(state, project.Path, value.ID, mode); err != nil {
			return err
		}
		engine := planner.New(s.store)
		engine.AddActivations(&state, []domain.Skill{value}, domain.PlacementProject, project.Path, agents, mode)
		skills, skillsErr := s.store.LoadAllSkills()
		if skillsErr != nil {
			return skillsErr
		}
		plan, err = engine.BuildScoped(skills, state, domain.PlacementProject, project.Path)
		if err != nil {
			return err
		}
		if body.DryRun {
			return nil
		}
		return engine.Apply(plan, &state)
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": project, "skill": value, "plan": plan, "applied": !body.DryRun})
}

func (s *Server) handleProjectUnlink(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Skill  string   `json:"skill"`
		Agents []string `json:"agents"`
		Force  bool     `json:"force"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(body.Skill) == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("skill is required"))
		return
	}
	var project domain.Project
	err := s.withLock(func() error {
		var err error
		project, err = s.resolveProject(r.PathValue("id"))
		if err != nil {
			return err
		}
		state, err := s.store.LoadState()
		if err != nil {
			return err
		}
		skillID, err := resolveProjectSkill(s.store, state, project.Path, body.Skill)
		if err != nil {
			return err
		}
		agentMap := make(map[domain.Agent]struct{})
		if len(body.Agents) > 0 {
			agents, parseErr := parseProjectAgents(project.Path, body.Agents)
			if parseErr != nil {
				return parseErr
			}
			for _, agent := range agents {
				agentMap[agent] = struct{}{}
			}
		}
		return planner.New(s.store).Disable(&state, map[string]struct{}{skillID: {}}, domain.PlacementProject, project.Path, agentMap, body.Force)
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": project, "skill": body.Skill, "status": "ok"})
}

func parseProjectAgents(projectRoot string, values []string) ([]domain.Agent, error) {
	roots, err := projectSkillRoots(projectRoot)
	if err != nil {
		return nil, err
	}
	available := make(map[domain.Agent]struct{}, len(roots))
	defaults := make([]domain.Agent, 0, len(roots))
	for _, root := range roots {
		agent := domain.Agent(root.ID)
		if !agent.ProjectValid() {
			continue
		}
		available[agent] = struct{}{}
		defaults = append(defaults, agent)
	}
	if len(values) == 0 {
		if len(defaults) == 0 {
			return nil, fmt.Errorf("no Agent directories were found in the project")
		}
		return defaults, nil
	}
	result := make([]domain.Agent, 0, len(values))
	seen := make(map[domain.Agent]struct{})
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			agent := domain.Agent(strings.TrimSpace(part))
			if _, ok := available[agent]; !ok {
				return nil, fmt.Errorf("Agent directory %q was not found in the project scan", part)
			}
			if _, ok := seen[agent]; ok {
				continue
			}
			seen[agent] = struct{}{}
			result = append(result, agent)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("at least one project Agent is required")
	}
	return result, nil
}

func (s *Server) handleUnregisterProject(w http.ResponseWriter, r *http.Request) {
	var removed domain.Project
	err := s.withLock(func() error {
		projects, err := s.store.LoadProjects()
		if err != nil {
			return err
		}
		removed, err = resolveProjectFromList(projects.Projects, r.PathValue("id"))
		if err != nil {
			return err
		}
		state, err := s.store.LoadState()
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
		return s.store.SaveProjects(projects)
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, removed)
}

func (s *Server) resolveProject(query string) (domain.Project, error) {
	projects, err := s.store.LoadProjects()
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

func (s *Server) projectDetails(project domain.Project) (projectDetails, error) {
	state, err := s.store.LoadState()
	if err != nil {
		return projectDetails{}, err
	}
	scan, err := scanProjectSkills(project.Path)
	if err != nil {
		return projectDetails{}, err
	}
	library, err := s.store.LoadCatalog()
	if err != nil {
		return projectDetails{}, err
	}
	markProjectLibrarySkills(&scan, library.Skills)
	plan, err := s.projectPlan(project.Path)
	if err != nil {
		return projectDetails{}, err
	}
	info, statErr := os.Stat(project.Path)
	activations := projectActivations(state, project.Path)
	if activations == nil {
		activations = []domain.Activation{}
	}
	if plan.Operations == nil {
		plan.Operations = []domain.Operation{}
	}
	return projectDetails{
		Project:     project,
		Exists:      statErr == nil && info.IsDir(),
		Activations: activations,
		Scan:        scan,
		Plan:        plan,
	}, nil
}

func markProjectLibrarySkills(scan *projectScan, library []domain.Skill) {
	for index := range scan.Skills {
		item := &scan.Skills[index]
		for _, value := range library {
			if value.Name == item.Name || value.Name == item.ID {
				item.LibrarySkillID = value.ID
				break
			}
		}
	}
}

func (s *Server) handleRemoveProjectSkill(w http.ResponseWriter, r *http.Request) {
	var project domain.Project
	removedPaths := make([]string, 0)
	err := s.withLock(func() error {
		var err error
		project, err = s.resolveProject(r.PathValue("id"))
		if err != nil {
			return err
		}
		skillID := r.PathValue("skill")
		paths, err := projectSkillPaths(project.Path, skillID)
		if err != nil {
			return err
		}
		state, err := s.store.LoadState()
		if err != nil {
			return err
		}
		for _, activation := range projectActivations(state, project.Path) {
			if activation.Name == skillID {
				return fmt.Errorf("project Skill %s is managed by SKM; unlink it instead", skillID)
			}
		}
		for _, deployment := range state.Deployments {
			if deployment.Placement != domain.PlacementProject || filepath.Clean(deployment.ProjectRoot) != filepath.Clean(project.Path) {
				continue
			}
			for _, path := range paths {
				if filepath.Clean(deployment.Target) == filepath.Clean(path) {
					return fmt.Errorf("project Skill %s is managed by SKM; unlink it instead", skillID)
				}
			}
		}
		for _, path := range paths {
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf("remove project Skill %s: %w", path, err)
			}
			removedPaths = append(removedPaths, path)
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project": project, "skill": r.PathValue("skill"), "removedPaths": removedPaths,
	})
}

func projectSkillPaths(projectRoot, skillID string) ([]string, error) {
	if skillID == "" || filepath.Base(skillID) != skillID || skillID == "." || skillID == ".." {
		return nil, fmt.Errorf("invalid project Skill %q", skillID)
	}
	roots, err := projectSkillRoots(projectRoot)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(roots))
	for _, root := range roots {
		path := filepath.Join(root.Path, skillID)
		_, candidate, _ := projectSkillDirectory(path)
		if candidate {
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("project Skill %q not found", skillID)
	}
	return paths, nil
}

func scanProjectSkills(projectRoot string) (projectScan, error) {
	result := projectScan{
		ScannedAt:   time.Now().UTC(),
		AgentCounts: make(map[string]int),
		Agents:      make([]projectScanAgent, 0),
		Skills:      make([]projectScanSkill, 0),
	}
	byID := make(map[string]int)
	roots, rootsErr := projectSkillRoots(projectRoot)
	if rootsErr != nil {
		result.Errors = append(result.Errors, rootsErr.Error())
		return result, nil
	}
	for _, agent := range roots {
		result.AgentCounts[agent.ID] = 0
		result.Agents = append(result.Agents, projectScanAgent{ID: agent.ID, Label: agent.Label})
		entries, err := os.ReadDir(agent.Path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", agent.Label, err))
			continue
		}
		for _, entry := range entries {
			path := filepath.Join(agent.Path, entry.Name())
			resolved, candidate, candidateErr := projectSkillDirectory(path)
			if !candidate {
				continue
			}
			index, ok := byID[entry.Name()]
			if !ok {
				result.Skills = append(result.Skills, projectScanSkill{
					ID:     entry.Name(),
					Name:   entry.Name(),
					Status: "ok",
					Agents: make([]string, 0, 2),
					Paths:  make(map[string]string),
				})
				index = len(result.Skills) - 1
				byID[entry.Name()] = index
			}
			item := &result.Skills[index]
			item.Agents = append(item.Agents, agent.ID)
			item.Paths[agent.ID] = path
			result.AgentCounts[agent.ID]++

			if candidateErr != nil {
				addScanIssue(item, agent.Label, "error", candidateErr.Error())
				continue
			}
			document, validateErr := skill.Validate(resolved)
			if validateErr != nil {
				addScanIssue(item, agent.Label, "warning", validateErr.Error())
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

type projectSkillRoot struct {
	ID    string
	Label string
	Path  string
}

// projectSkillRoots discovers Agent directories using the shared convention
// <project>/.<agent>/skills. This keeps scanning independent of deploy support.
func projectSkillRoots(projectRoot string) ([]projectSkillRoot, error) {
	entries, err := os.ReadDir(projectRoot)
	if os.IsNotExist(err) {
		return []projectSkillRoot{}, nil
	}
	if err != nil {
		return nil, err
	}
	roots := make([]projectSkillRoot, 0)
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, ".") || len(name) == 1 {
			continue
		}
		agentRoot := filepath.Join(projectRoot, name)
		info, statErr := os.Stat(agentRoot)
		if statErr != nil || !info.IsDir() {
			continue
		}
		skillsPath := filepath.Join(agentRoot, "skills")
		info, statErr = os.Stat(skillsPath)
		if statErr != nil || !info.IsDir() {
			continue
		}
		id := strings.TrimPrefix(name, ".")
		roots = append(roots, projectSkillRoot{ID: id, Label: projectAgentLabel(id), Path: skillsPath})
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].ID < roots[j].ID })
	return roots, nil
}

func projectAgentLabel(id string) string {
	switch id {
	case "claude":
		return "Claude Code"
	case "codex":
		return "Codex"
	case "cursor":
		return "Cursor"
	case "agent":
		return "Agent"
	case "agents":
		return "Agents"
	default:
		return "." + id
	}
}

// projectSkillDirectory identifies a top-level Agent Skill directory. Files and
// folders without SKILL.md are not Skills and must not appear in scan results.
func projectSkillDirectory(path string) (string, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", false, nil
	}
	if !info.IsDir() {
		return "", false, nil
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", false, nil
	}
	_, err = os.Stat(filepath.Join(resolved, "SKILL.md"))
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return resolved, true, err
	}
	return resolved, true, nil
}

func addScanIssue(item *projectScanSkill, agentLabel, status, message string) {
	item.Issues = append(item.Issues, fmt.Sprintf("%s: %s", agentLabel, message))
	if item.Status == "error" || (item.Status == "warning" && status != "error") {
		return
	}
	item.Status = status
}

func (s *Server) projectPlan(projectRoot string) (domain.Plan, error) {
	state, err := s.store.LoadState()
	if err != nil {
		return domain.Plan{}, err
	}
	skills, err := s.store.LoadAllSkills()
	if err != nil {
		return domain.Plan{}, err
	}
	return planner.New(s.store).BuildScoped(skills, state, domain.PlacementProject, projectRoot)
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

func resolveProjectSkill(storage *store.Store, state domain.State, projectRoot, query string) (string, error) {
	if value, err := catalog.New(storage).ResolveLibrary(query); err == nil {
		return value.ID, nil
	}
	var matches []string
	for _, activation := range projectActivations(state, projectRoot) {
		if activation.SkillID == query || activation.Name == query {
			matches = append(matches, activation.SkillID)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		sort.Strings(matches)
		return "", fmt.Errorf("skill %q is ambiguous; use one of: %s", query, strings.Join(matches, ", "))
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
