package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/fsx"
	"gopkg.in/yaml.v3"
)

type Paths struct {
	Home        string
	UserHome    string
	ProjectRoot string
}

type Store struct {
	Paths Paths
}

func New(paths Paths) (*Store, error) {
	var err error
	paths.Home, err = filepath.Abs(paths.Home)
	if err != nil {
		return nil, err
	}
	paths.UserHome, err = filepath.Abs(paths.UserHome)
	if err != nil {
		return nil, err
	}
	paths.ProjectRoot, err = filepath.Abs(paths.ProjectRoot)
	if err != nil {
		return nil, err
	}
	return &Store{Paths: paths}, nil
}

func DefaultPaths(homeOverride, projectOverride string) (Paths, error) {
	return DefaultPathsWithUserHome(homeOverride, "", projectOverride)
}

// DefaultPathsWithUserHome resolves the three independent filesystem roots
// used by skm. The user-home override is intended for integration tests and
// isolated development, where Agent deployment targets must not use the real
// user home directory.
func DefaultPathsWithUserHome(homeOverride, userHomeOverride, projectOverride string) (Paths, error) {
	userHome := userHomeOverride
	if userHome == "" {
		var err error
		userHome, err = os.UserHomeDir()
		if err != nil {
			return Paths{}, err
		}
	}
	home := homeOverride
	if home == "" {
		home = os.Getenv("SKM_HOME")
	}
	if home == "" {
		home = filepath.Join(userHome, ".skm")
	}
	project := projectOverride
	if project == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return Paths{}, err
		}
		project = FindProjectRoot(cwd)
	}
	return Paths{Home: home, UserHome: userHome, ProjectRoot: project}, nil
}

func FindProjectRoot(start string) string {
	current, err := filepath.Abs(start)
	if err != nil {
		return start
	}
	for {
		if exists(filepath.Join(current, ".git")) || exists(filepath.Join(current, ".skm")) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return start
		}
		current = parent
	}
}

func (s *Store) Ensure() error {
	for _, dir := range []string{
		s.Paths.Home,
		filepath.Join(s.Paths.Home, "objects"),
		filepath.Join(s.Paths.Home, "sources"),
		filepath.Join(s.Paths.Home, "state"),
		filepath.Join(s.Paths.Home, "locks"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	configPath := filepath.Join(s.Paths.Home, "config.yaml")
	if !exists(configPath) {
		return s.SaveConfig(domain.DefaultConfig())
	}
	return nil
}

func (s *Store) EnsureProject() error {
	return os.MkdirAll(filepath.Join(s.Paths.ProjectRoot, ".skm", "skills"), 0o755)
}

func (s *Store) Lock() (func() error, error) {
	if err := s.Ensure(); err != nil {
		return nil, err
	}
	path := filepath.Join(s.Paths.Home, "locks", "state.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return func() error {
		unlockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		closeErr := file.Close()
		return errors.Join(unlockErr, closeErr)
	}, nil
}

func (s *Store) LoadConfig() (domain.Config, error) {
	config := domain.DefaultConfig()
	if err := loadYAML(filepath.Join(s.Paths.Home, "config.yaml"), &config); err != nil {
		return domain.Config{}, err
	}
	return config, nil
}

func (s *Store) SaveConfig(config domain.Config) error {
	config.Version = domain.SchemaVersion
	return saveYAML(filepath.Join(s.Paths.Home, "config.yaml"), config)
}

func (s *Store) LoadCatalog() (domain.Catalog, error) {
	catalog := domain.Catalog{Version: domain.SchemaVersion}
	if err := loadYAML(filepath.Join(s.Paths.Home, "catalog.yaml"), &catalog); err != nil {
		return domain.Catalog{}, err
	}
	for i := range catalog.Skills {
		normalizeSkill(&catalog.Skills[i], domain.LocationLibrary)
	}
	return catalog, nil
}

func (s *Store) SaveCatalog(catalog domain.Catalog) error {
	catalog.Version = domain.SchemaVersion
	for i := range catalog.Skills {
		catalog.Skills[i].Location = domain.LocationLibrary
		catalog.Skills[i].LegacyScope = ""
	}
	sort.Slice(catalog.Skills, func(i, j int) bool { return catalog.Skills[i].ID < catalog.Skills[j].ID })
	return saveYAML(filepath.Join(s.Paths.Home, "catalog.yaml"), catalog)
}

func (s *Store) LoadProjectCatalog() (domain.Catalog, error) {
	catalog := domain.Catalog{Version: domain.SchemaVersion}
	if err := loadYAML(filepath.Join(s.Paths.ProjectRoot, ".skm", "project.yaml"), &catalog); err != nil {
		return domain.Catalog{}, err
	}
	for i := range catalog.Skills {
		normalizeSkill(&catalog.Skills[i], domain.LocationProject)
		if !filepath.IsAbs(catalog.Skills[i].Path) {
			catalog.Skills[i].Path = filepath.Join(s.Paths.ProjectRoot, catalog.Skills[i].Path)
		}
		catalog.Skills[i].ProjectRoot = s.Paths.ProjectRoot
	}
	return catalog, nil
}

func (s *Store) SaveProjectCatalog(catalog domain.Catalog) error {
	if err := s.EnsureProject(); err != nil {
		return err
	}
	catalog.Version = domain.SchemaVersion
	sort.Slice(catalog.Skills, func(i, j int) bool { return catalog.Skills[i].ID < catalog.Skills[j].ID })
	sort.Slice(catalog.Dependencies, func(i, j int) bool { return catalog.Dependencies[i].ID < catalog.Dependencies[j].ID })
	portable := catalog
	portable.Skills = append([]domain.Skill(nil), catalog.Skills...)
	for i := range portable.Skills {
		portable.Skills[i].Location = domain.LocationProject
		portable.Skills[i].LegacyScope = ""
		if fsx.Within(s.Paths.ProjectRoot, portable.Skills[i].Path) {
			if relative, err := filepath.Rel(s.Paths.ProjectRoot, portable.Skills[i].Path); err == nil {
				portable.Skills[i].Path = filepath.ToSlash(relative)
			}
		}
		portable.Skills[i].ProjectRoot = ""
	}
	return saveYAML(filepath.Join(s.Paths.ProjectRoot, ".skm", "project.yaml"), portable)
}

func (s *Store) SaveProjectLock(catalog domain.Catalog) error {
	lockFile := domain.LockFile{Version: domain.SchemaVersion}
	for _, dependency := range catalog.Dependencies {
		lockFile.Skills = append(lockFile.Skills, domain.LockedSkill{
			ID: dependency.ID, Name: dependency.Name, Source: dependency.Source,
			Revision: dependency.Revision, Hash: dependency.Hash, Tags: append([]string(nil), dependency.Tags...),
		})
	}
	for _, value := range catalog.Skills {
		lockFile.Skills = append(lockFile.Skills, domain.LockedSkill{
			ID: value.ID, Name: value.Name, Source: value.Source,
			Revision: value.Revision, Hash: value.Hash, Tags: append([]string(nil), value.Tags...),
		})
	}
	sort.Slice(lockFile.Skills, func(i, j int) bool { return lockFile.Skills[i].ID < lockFile.Skills[j].ID })
	return saveYAML(filepath.Join(s.Paths.ProjectRoot, ".skm", "lock.yaml"), lockFile)
}

func (s *Store) HasProjectState() bool {
	return exists(filepath.Join(s.Paths.ProjectRoot, ".skm", "project.yaml"))
}

func (s *Store) LoadAllSkills() ([]domain.Skill, error) {
	central, err := s.LoadCatalog()
	if err != nil {
		return nil, err
	}
	project, err := s.LoadProjectCatalog()
	if err != nil {
		return nil, err
	}
	return append(central.Skills, project.Skills...), nil
}

func (s *Store) UpsertSkill(value domain.Skill) error {
	if value.Location == domain.LocationProject {
		catalog, err := s.LoadProjectCatalog()
		if err != nil {
			return err
		}
		catalog.Skills = upsertSkill(catalog.Skills, value)
		return s.SaveProjectCatalog(catalog)
	}
	value.Location = domain.LocationLibrary
	catalog, err := s.LoadCatalog()
	if err != nil {
		return err
	}
	catalog.Skills = upsertSkill(catalog.Skills, value)
	return s.SaveCatalog(catalog)
}

func (s *Store) RemoveSkill(id string, location domain.SkillLocation) error {
	if location == domain.LocationProject {
		catalog, err := s.LoadProjectCatalog()
		if err != nil {
			return err
		}
		catalog.Skills = removeSkill(catalog.Skills, id)
		return s.SaveProjectCatalog(catalog)
	}
	catalog, err := s.LoadCatalog()
	if err != nil {
		return err
	}
	catalog.Skills = removeSkill(catalog.Skills, id)
	return s.SaveCatalog(catalog)
}

func (s *Store) LoadSources() (domain.Sources, error) {
	sources := domain.Sources{Version: domain.SchemaVersion}
	if err := loadYAML(filepath.Join(s.Paths.Home, "sources.yaml"), &sources); err != nil {
		return domain.Sources{}, err
	}
	for i := range sources.Sources {
		sources.Sources[i].LegacyScope = ""
	}
	return sources, nil
}

func (s *Store) SaveSources(sources domain.Sources) error {
	sources.Version = domain.SchemaVersion
	for i := range sources.Sources {
		sources.Sources[i].LegacyScope = ""
	}
	sort.Slice(sources.Sources, func(i, j int) bool { return sources.Sources[i].Name < sources.Sources[j].Name })
	return saveYAML(filepath.Join(s.Paths.Home, "sources.yaml"), sources)
}

func (s *Store) LoadProjects() (domain.Projects, error) {
	projects := domain.Projects{Version: domain.SchemaVersion}
	if err := loadYAML(filepath.Join(s.Paths.Home, "projects.yaml"), &projects); err != nil {
		return domain.Projects{}, err
	}
	return projects, nil
}

func (s *Store) SaveProjects(projects domain.Projects) error {
	projects.Version = domain.SchemaVersion
	sort.Slice(projects.Projects, func(i, j int) bool { return projects.Projects[i].ID < projects.Projects[j].ID })
	return saveYAML(filepath.Join(s.Paths.Home, "projects.yaml"), projects)
}

func (s *Store) LoadState() (domain.State, error) {
	state := domain.State{Version: domain.SchemaVersion}
	if err := loadYAML(filepath.Join(s.Paths.Home, "state", "state.yaml"), &state); err != nil {
		return domain.State{}, err
	}
	for _, legacy := range state.LegacyInstallations {
		placement := domain.PlacementUser
		if legacy.Scope == "project" {
			placement = domain.PlacementProject
		}
		state.Activations = append(state.Activations, domain.Activation{
			SkillID: legacy.SkillID, Name: legacy.Name, Placement: placement,
			ProjectRoot: legacy.ProjectRoot, Agents: legacy.Agents, Mode: legacy.Mode, UpdatedAt: legacy.UpdatedAt,
		})
	}
	state.LegacyInstallations = nil
	for i := range state.Deployments {
		if !state.Deployments[i].Placement.Valid() {
			state.Deployments[i].Placement = domain.PlacementUser
			if state.Deployments[i].LegacyScope == "project" {
				state.Deployments[i].Placement = domain.PlacementProject
			}
		}
		state.Deployments[i].LegacyScope = ""
	}
	state.Version = domain.SchemaVersion
	return state, nil
}

func (s *Store) SaveState(state domain.State) error {
	state.Version = domain.SchemaVersion
	state.LegacyInstallations = nil
	for i := range state.Deployments {
		state.Deployments[i].LegacyScope = ""
	}
	sort.Slice(state.Activations, func(i, j int) bool {
		left := string(state.Activations[i].Placement) + state.Activations[i].ProjectRoot + state.Activations[i].SkillID
		right := string(state.Activations[j].Placement) + state.Activations[j].ProjectRoot + state.Activations[j].SkillID
		return left < right
	})
	return saveYAML(filepath.Join(s.Paths.Home, "state", "state.yaml"), state)
}

func (s *Store) ObjectPath(hash, name string) string {
	return filepath.Join(s.Paths.Home, "objects", hash, name)
}

func (s *Store) SourcePath(name string) string {
	return filepath.Join(s.Paths.Home, "sources", name)
}

func normalizeSkill(value *domain.Skill, fallback domain.SkillLocation) {
	if !value.Location.Valid() {
		value.Location = fallback
		if value.LegacyScope == "project" {
			value.Location = domain.LocationProject
		}
	}
	value.LegacyScope = ""
}

func loadYAML(path string, target any) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	if err := yaml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return nil
}

func saveYAML(path string, value any) error {
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	return fsx.AtomicWriteFile(path, data, 0o644)
}

func upsertSkill(skills []domain.Skill, value domain.Skill) []domain.Skill {
	for i := range skills {
		if skills[i].ID == value.ID {
			skills[i] = value
			return skills
		}
	}
	return append(skills, value)
}

func removeSkill(skills []domain.Skill, id string) []domain.Skill {
	result := skills[:0]
	for _, value := range skills {
		if value.ID != id {
			result = append(result, value)
		}
	}
	return result
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
