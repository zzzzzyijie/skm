package application

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/fsx"
	"github.com/zzzzzyijie/skm/internal/planner"
	"github.com/zzzzzyijie/skm/internal/skill"
	gitSource "github.com/zzzzzyijie/skm/internal/source"
	"github.com/zzzzzyijie/skm/internal/store"
)

type ProjectRequireInput struct {
	Project string          `json:"project"`
	Skill   string          `json:"skill"`
	Agents  []string        `json:"agents"`
	Mode    domain.LinkMode `json:"mode"`
	Apply   bool            `json:"apply"`
}

type ProjectVendorInput struct {
	Project string          `json:"project"`
	Skill   string          `json:"skill"`
	Agents  []string        `json:"agents"`
	Mode    domain.LinkMode `json:"mode"`
	Tags    []string        `json:"tags"`
	Apply   bool            `json:"apply"`
}

type ProjectApplyInput struct {
	Project string `json:"project"`
	Force   bool   `json:"force"`
}

type ProjectEntryRemoveInput struct {
	Project string `json:"project"`
	Entry   string `json:"entry"`
	Force   bool   `json:"force"`
}

type ProjectAdvancedResult struct {
	Project         domain.Project            `json:"project"`
	Manifest        domain.Catalog            `json:"manifest"`
	Dependency      *domain.ProjectDependency `json:"dependency,omitempty"`
	Skill           *domain.Skill             `json:"skill,omitempty"`
	Plan            domain.Plan               `json:"plan"`
	SatisfiedByUser []string                  `json:"satisfiedByUser"`
	Applied         bool                      `json:"applied"`
	RemovedID       string                    `json:"removedId,omitempty"`
}

func (s *Service) RequireProjectSkill(input ProjectRequireInput) (ProjectAdvancedResult, error) {
	if strings.TrimSpace(input.Skill) == "" {
		return ProjectAdvancedResult{}, fmt.Errorf("skill is required")
	}
	mode := input.Mode
	if mode == "" {
		mode = domain.ModeAuto
	}
	if !mode.Valid() {
		return ProjectAdvancedResult{}, fmt.Errorf("invalid mode %q", input.Mode)
	}
	var result ProjectAdvancedResult
	err := s.withLock(func() error {
		project, err := s.resolveProject(input.Project)
		if err != nil {
			return err
		}
		projectStore, err := s.storeForProject(project)
		if err != nil {
			return err
		}
		agents, err := s.parseProjectAgents(project.Path, input.Agents)
		if err != nil {
			return err
		}
		value, err := catalog.New(projectStore).ResolveLibrary(input.Skill)
		if err != nil {
			return err
		}
		source, err := findAdvancedSource(projectStore, value.Source)
		if err != nil || !shareableAdvancedGitURL(source.URL) {
			return fmt.Errorf("skill %s has no shareable Git source; vendor it instead", value.ID)
		}
		if value.Revision == "" {
			return fmt.Errorf("skill %s has no Git revision; update its source before requiring it", value.ID)
		}
		dependency := domain.ProjectDependency{
			ID: value.ID, Name: value.Name, Source: value.Source, URL: source.URL,
			Ref: source.Ref, SourcePath: value.SourcePath, Revision: value.Revision,
			Hash: value.Hash, Tags: append([]string(nil), value.Tags...), Agents: agents, Mode: mode,
		}
		manifest, err := projectStore.LoadProjectCatalog()
		if err != nil {
			return err
		}
		if err := ensureAdvancedProjectNameAvailable(manifest, dependency.Name, dependency.ID); err != nil {
			return err
		}
		manifest.Dependencies = upsertAdvancedDependency(manifest.Dependencies, dependency)
		if err := projectStore.SaveProjectCatalog(manifest); err != nil {
			return err
		}
		if err := projectStore.SaveProjectLock(manifest); err != nil {
			return err
		}
		result = ProjectAdvancedResult{Project: project, Manifest: manifest, Dependency: &dependency, Applied: input.Apply}
		if input.Apply {
			applied, err := applyAdvancedProjectLocked(projectStore, false)
			if err != nil {
				return err
			}
			result.Manifest, result.Plan, result.SatisfiedByUser = applied.Manifest, applied.Plan, applied.SatisfiedByUser
		}
		normalizeAdvancedResult(&result)
		return nil
	})
	return result, err
}

func (s *Service) VendorProjectSkill(input ProjectVendorInput) (ProjectAdvancedResult, error) {
	if strings.TrimSpace(input.Skill) == "" {
		return ProjectAdvancedResult{}, fmt.Errorf("skill is required")
	}
	mode := input.Mode
	if mode == "" {
		mode = domain.ModeAuto
	}
	if !mode.Valid() {
		return ProjectAdvancedResult{}, fmt.Errorf("invalid mode %q", input.Mode)
	}
	var result ProjectAdvancedResult
	err := s.withLock(func() error {
		project, err := s.resolveProject(input.Project)
		if err != nil {
			return err
		}
		projectStore, err := s.storeForProject(project)
		if err != nil {
			return err
		}
		agents, err := s.parseProjectAgents(project.Path, input.Agents)
		if err != nil {
			return err
		}
		value, err := catalog.New(projectStore).ResolveLibrary(input.Skill)
		if err != nil {
			return err
		}
		if err := checkAdvancedUserNameConflict(projectStore, value.Name, "project/"+value.Name, value.Hash, agents); err != nil {
			return err
		}
		manifest, err := projectStore.LoadProjectCatalog()
		if err != nil {
			return err
		}
		if err := ensureAdvancedProjectNameAvailable(manifest, value.Name, "project/"+value.Name); err != nil {
			return err
		}
		vendored, err := catalog.New(projectStore).Vendor(value, agents, mode, input.Tags)
		if err != nil {
			return err
		}
		manifest, err = projectStore.LoadProjectCatalog()
		if err != nil {
			return err
		}
		if err := projectStore.SaveProjectLock(manifest); err != nil {
			return err
		}
		result = ProjectAdvancedResult{Project: project, Manifest: manifest, Skill: &vendored, Applied: input.Apply}
		if input.Apply {
			applied, err := applyAdvancedProjectLocked(projectStore, false)
			if err != nil {
				return err
			}
			result.Manifest, result.Plan, result.SatisfiedByUser = applied.Manifest, applied.Plan, applied.SatisfiedByUser
		}
		normalizeAdvancedResult(&result)
		return nil
	})
	return result, err
}

func (s *Service) ApplyProject(input ProjectApplyInput) (ProjectAdvancedResult, error) {
	var result ProjectAdvancedResult
	err := s.withLock(func() error {
		project, err := s.resolveProject(input.Project)
		if err != nil {
			return err
		}
		projectStore, err := s.storeForProject(project)
		if err != nil {
			return err
		}
		result, err = applyAdvancedProjectLocked(projectStore, input.Force)
		result.Project = project
		result.Applied = err == nil
		normalizeAdvancedResult(&result)
		return err
	})
	return result, err
}

func (s *Service) RemoveProjectEntry(input ProjectEntryRemoveInput) (ProjectAdvancedResult, error) {
	if strings.TrimSpace(input.Entry) == "" {
		return ProjectAdvancedResult{}, fmt.Errorf("entry is required")
	}
	var result ProjectAdvancedResult
	err := s.withLock(func() error {
		project, err := s.resolveProject(input.Project)
		if err != nil {
			return err
		}
		projectStore, err := s.storeForProject(project)
		if err != nil {
			return err
		}
		manifest, err := projectStore.LoadProjectCatalog()
		if err != nil {
			return err
		}
		updated, vendoredPath, removedID, err := removeAdvancedProjectEntry(manifest, input.Entry)
		if err != nil {
			return err
		}
		if err := projectStore.SaveProjectCatalog(updated); err != nil {
			return err
		}
		if err := projectStore.SaveProjectLock(updated); err != nil {
			return err
		}
		result, err = applyAdvancedProjectLocked(projectStore, input.Force)
		if err != nil {
			return err
		}
		if vendoredPath != "" {
			root := filepath.Join(project.Path, ".skm", "skills")
			if !fsx.Within(root, vendoredPath) {
				return fmt.Errorf("refusing to remove vendored path outside %s", root)
			}
			if err := os.RemoveAll(vendoredPath); err != nil {
				return err
			}
		}
		result.Project, result.RemovedID, result.Applied = project, removedID, true
		normalizeAdvancedResult(&result)
		return nil
	})
	return result, err
}

func (s *Service) storeForProject(project domain.Project) (*store.Store, error) {
	return store.New(store.Paths{Home: s.Store.Paths.Home, UserHome: s.Store.Paths.UserHome, ProjectRoot: project.Path})
}

func applyAdvancedProjectLocked(storage *store.Store, force bool) (ProjectAdvancedResult, error) {
	manifest, err := storage.LoadProjectCatalog()
	if err != nil {
		return ProjectAdvancedResult{}, err
	}
	if err := refreshAdvancedVendoredSkills(&manifest); err != nil {
		return ProjectAdvancedResult{}, err
	}
	pinned, err := ensureAdvancedDependencySnapshots(storage, manifest.Dependencies)
	if err != nil {
		return ProjectAdvancedResult{}, err
	}
	state, err := storage.LoadState()
	if err != nil {
		return ProjectAdvancedResult{}, err
	}
	desired, satisfied, err := advancedProjectActivations(storage, manifest, pinned, state)
	if err != nil {
		return ProjectAdvancedResult{}, err
	}
	engine := planner.New(storage)
	if err := engine.SetProjectActivations(&state, storage.Paths.ProjectRoot, desired, force); err != nil {
		return ProjectAdvancedResult{}, err
	}
	if err := storage.SaveProjectCatalog(manifest); err != nil {
		return ProjectAdvancedResult{}, err
	}
	if err := storage.SaveProjectLock(manifest); err != nil {
		return ProjectAdvancedResult{}, err
	}
	skills, err := storage.LoadAllSkills()
	if err != nil {
		return ProjectAdvancedResult{}, err
	}
	plan, err := engine.BuildScoped(skills, state, domain.PlacementProject, storage.Paths.ProjectRoot)
	if err != nil {
		return ProjectAdvancedResult{}, err
	}
	if err := engine.Apply(plan, &state); err != nil {
		return ProjectAdvancedResult{}, err
	}
	result := ProjectAdvancedResult{Manifest: manifest, Plan: plan, SatisfiedByUser: satisfied, Applied: true}
	normalizeAdvancedResult(&result)
	return result, nil
}

func refreshAdvancedVendoredSkills(manifest *domain.Catalog) error {
	for index := range manifest.Skills {
		document, err := skill.Validate(manifest.Skills[index].Path)
		if err != nil {
			return fmt.Errorf("validate vendored Skill %s: %w", manifest.Skills[index].ID, err)
		}
		manifest.Skills[index].Name = document.Name
		manifest.Skills[index].Description = document.Description
		manifest.Skills[index].Hash = document.Hash
		manifest.Skills[index].Metadata = document.Metadata
		manifest.Skills[index].Location = domain.LocationProject
	}
	return nil
}

func ensureAdvancedDependencySnapshots(storage *store.Store, dependencies []domain.ProjectDependency) (map[string]domain.Skill, error) {
	result := make(map[string]domain.Skill, len(dependencies))
	for _, dependency := range dependencies {
		path := storage.ObjectPath(dependency.Hash, dependency.Name)
		if hash, err := fsx.HashDir(path); err == nil && hash == dependency.Hash {
			result[dependency.ID] = domain.Skill{ID: dependency.ID, Name: dependency.Name, Hash: dependency.Hash, Path: path}
			continue
		}
		if dependency.URL == "" || dependency.Revision == "" {
			return nil, fmt.Errorf("project dependency %s cannot be restored; Git URL or revision is missing", dependency.ID)
		}
		var paths []string
		if dependency.SourcePath != "" {
			paths = []string{dependency.SourcePath}
		}
		fetched, err := gitSource.NewGitManager(storage, catalog.New(storage)).FetchPinned(domain.Source{
			Name: dependency.Source, URL: dependency.URL, Ref: dependency.Revision, Paths: paths, Tags: dependency.Tags,
		})
		if err != nil {
			return nil, fmt.Errorf("restore project dependency %s: %w", dependency.ID, err)
		}
		for _, value := range fetched {
			if value.ID == dependency.ID && value.Hash == dependency.Hash {
				result[dependency.ID] = value
				break
			}
		}
		if _, ok := result[dependency.ID]; !ok {
			return nil, fmt.Errorf("restored project dependency %s does not match locked hash %s", dependency.ID, dependency.Hash)
		}
	}
	return result, nil
}

func advancedProjectActivations(storage *store.Store, manifest domain.Catalog, pinned map[string]domain.Skill, state domain.State) ([]domain.Activation, []string, error) {
	library, err := storage.LoadCatalog()
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[string]domain.Skill, len(library.Skills))
	for _, value := range library.Skills {
		byID[value.ID] = value
	}
	var desired []domain.Activation
	var satisfied []string
	for _, dependency := range manifest.Dependencies {
		value, ok := pinned[dependency.ID]
		if !ok {
			return nil, nil, fmt.Errorf("missing pinned snapshot for %s", dependency.ID)
		}
		remaining, matched, err := remainingAdvancedProjectAgents(dependency.ID, dependency.Name, dependency.Hash, dependency.Agents, state.Activations, byID)
		if err != nil {
			return nil, nil, err
		}
		for _, agent := range matched {
			satisfied = append(satisfied, dependency.ID+":"+string(agent))
		}
		if len(remaining) > 0 {
			desired = append(desired, domain.Activation{
				SkillID: dependency.ID, Name: dependency.Name, Placement: domain.PlacementProject,
				ProjectRoot: storage.Paths.ProjectRoot, Agents: remaining, Mode: dependency.Mode,
				PinnedHash: dependency.Hash, PinnedPath: value.Path,
			})
		}
	}
	for _, value := range manifest.Skills {
		remaining, _, err := remainingAdvancedProjectAgents(value.ID, value.Name, value.Hash, value.Agents, state.Activations, byID)
		if err != nil {
			return nil, nil, err
		}
		if len(remaining) > 0 {
			desired = append(desired, domain.Activation{
				SkillID: value.ID, Name: value.Name, Placement: domain.PlacementProject,
				ProjectRoot: storage.Paths.ProjectRoot, Agents: remaining, Mode: value.Mode,
				PinnedHash: value.Hash, PinnedPath: value.Path,
			})
		}
	}
	sort.Strings(satisfied)
	return desired, satisfied, nil
}

func remainingAdvancedProjectAgents(id, name, hash string, requested []domain.Agent, activations []domain.Activation, library map[string]domain.Skill) ([]domain.Agent, []domain.Agent, error) {
	var remaining []domain.Agent
	var satisfied []domain.Agent
	for _, agent := range requested {
		matched := false
		for _, activation := range activations {
			if activation.Placement != domain.PlacementUser || !advancedContainsAgent(activation.Agents, agent) {
				continue
			}
			activeSkill, ok := library[activation.SkillID]
			if !ok {
				return nil, nil, fmt.Errorf("enabled skill %s is missing from Library", activation.SkillID)
			}
			if activeSkill.Name != name {
				continue
			}
			matched = true
			if activeSkill.ID == id && activeSkill.Hash == hash {
				satisfied = append(satisfied, agent)
				break
			}
			return nil, nil, fmt.Errorf("project Skill %s conflicts with enabled user Skill %s for %s; disable the user Skill or align versions", id, activeSkill.ID, agent)
		}
		if !matched {
			remaining = append(remaining, agent)
		}
	}
	return remaining, satisfied, nil
}

func checkAdvancedUserNameConflict(storage *store.Store, name, id, hash string, agents []domain.Agent) error {
	state, err := storage.LoadState()
	if err != nil {
		return err
	}
	library, err := storage.LoadCatalog()
	if err != nil {
		return err
	}
	byID := make(map[string]domain.Skill, len(library.Skills))
	for _, value := range library.Skills {
		byID[value.ID] = value
	}
	_, _, err = remainingAdvancedProjectAgents(id, name, hash, agents, state.Activations, byID)
	return err
}

func findAdvancedSource(storage *store.Store, name string) (domain.Source, error) {
	sources, err := storage.LoadSources()
	if err != nil {
		return domain.Source{}, err
	}
	for _, value := range sources.Sources {
		if value.Name == name {
			return value, nil
		}
	}
	return domain.Source{}, fmt.Errorf("Git source %q is not configured", name)
}

func shareableAdvancedGitURL(value string) bool {
	parsed, err := url.Parse(value)
	if err == nil && parsed.Scheme != "" && parsed.Scheme != "file" {
		return true
	}
	return strings.Contains(value, "@") && strings.Contains(value, ":") && !filepath.IsAbs(value)
}

func ensureAdvancedProjectNameAvailable(manifest domain.Catalog, name, replacingID string) error {
	for _, dependency := range manifest.Dependencies {
		if dependency.Name == name && dependency.ID != replacingID {
			return fmt.Errorf("project already contains same-name Skill %s", dependency.ID)
		}
	}
	for _, value := range manifest.Skills {
		if value.Name == name && value.ID != replacingID {
			return fmt.Errorf("project already contains same-name Skill %s", value.ID)
		}
	}
	return nil
}

func upsertAdvancedDependency(values []domain.ProjectDependency, value domain.ProjectDependency) []domain.ProjectDependency {
	for index := range values {
		if values[index].ID == value.ID {
			values[index] = value
			return values
		}
	}
	return append(values, value)
}

func removeAdvancedProjectEntry(manifest domain.Catalog, query string) (domain.Catalog, string, string, error) {
	query = strings.TrimSpace(query)
	for index, dependency := range manifest.Dependencies {
		if dependency.ID == query || dependency.Name == query {
			manifest.Dependencies = append(manifest.Dependencies[:index], manifest.Dependencies[index+1:]...)
			return manifest, "", dependency.ID, nil
		}
	}
	for index, value := range manifest.Skills {
		if value.ID == query || value.Name == query {
			manifest.Skills = append(manifest.Skills[:index], manifest.Skills[index+1:]...)
			return manifest, value.Path, value.ID, nil
		}
	}
	return domain.Catalog{}, "", "", fmt.Errorf("project entry %q not found", query)
}

func advancedContainsAgent(values []domain.Agent, target domain.Agent) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func normalizeAdvancedResult(result *ProjectAdvancedResult) {
	if result.Manifest.Skills == nil {
		result.Manifest.Skills = []domain.Skill{}
	}
	if result.Manifest.Dependencies == nil {
		result.Manifest.Dependencies = []domain.ProjectDependency{}
	}
	if result.Plan.Operations == nil {
		result.Plan.Operations = []domain.Operation{}
	}
	if result.SatisfiedByUser == nil {
		result.SatisfiedByUser = []string{}
	}
}
