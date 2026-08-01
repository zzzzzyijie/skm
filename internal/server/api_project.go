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
	"github.com/zzzzzyijie/skm/internal/store"
)

type projectRow struct {
	domain.Project
	Exists          bool `json:"exists"`
	ActivationCount int  `json:"activationCount"`
}

type projectDetails struct {
	Project     domain.Project      `json:"project"`
	Activations []domain.Activation `json:"activations"`
	Plan        domain.Plan         `json:"plan"`
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
		_, statErr := os.Stat(project.Path)
		rows = append(rows, projectRow{Project: project, Exists: statErr == nil, ActivationCount: counts[filepath.Clean(project.Path)]})
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
	agents, err := parseAgents(body.Agents)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var project domain.Project
	var value domain.Skill
	var plan domain.Plan
	err = s.withLock(func() error {
		project, err = s.resolveProject(r.PathValue("id"))
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
		plan, err = engine.Build(skills, state)
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
			agents, parseErr := parseAgents(body.Agents)
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
	plan, err := s.projectPlan(project.Path)
	if err != nil {
		return projectDetails{}, err
	}
	return projectDetails{Project: project, Activations: projectActivations(state, project.Path), Plan: plan}, nil
}

func (s *Server) projectPlan(projectRoot string) (domain.Plan, error) {
	plan, err := s.buildCurrentPlan()
	if err != nil {
		return domain.Plan{}, err
	}
	filtered := domain.Plan{Digest: plan.Digest}
	for _, operation := range plan.Operations {
		if operation.Placement == domain.PlacementProject && filepath.Clean(operation.ProjectRoot) == filepath.Clean(projectRoot) {
			filtered.Operations = append(filtered.Operations, operation)
		}
	}
	return filtered, nil
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
