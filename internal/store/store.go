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
	userHome, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
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
		if err := s.SaveConfig(domain.DefaultConfig()); err != nil {
			return err
		}
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
	return catalog, nil
}

func (s *Store) SaveCatalog(catalog domain.Catalog) error {
	catalog.Version = domain.SchemaVersion
	sort.Slice(catalog.Skills, func(i, j int) bool { return catalog.Skills[i].ID < catalog.Skills[j].ID })
	return saveYAML(filepath.Join(s.Paths.Home, "catalog.yaml"), catalog)
}

func (s *Store) LoadProjectCatalog() (domain.Catalog, error) {
	catalog := domain.Catalog{Version: domain.SchemaVersion}
	if err := loadYAML(filepath.Join(s.Paths.ProjectRoot, ".skm", "project.yaml"), &catalog); err != nil {
		return domain.Catalog{}, err
	}
	for i := range catalog.Skills {
		if !filepath.IsAbs(catalog.Skills[i].Path) {
			catalog.Skills[i].Path = filepath.Join(s.Paths.ProjectRoot, catalog.Skills[i].Path)
		}
		catalog.Skills[i].ProjectRoot = s.Paths.ProjectRoot
	}
	return catalog, nil
}

func (s *Store) SaveProjectCatalog(catalog domain.Catalog) error {
	catalog.Version = domain.SchemaVersion
	sort.Slice(catalog.Skills, func(i, j int) bool { return catalog.Skills[i].ID < catalog.Skills[j].ID })
	sort.Slice(catalog.Dependencies, func(i, j int) bool { return catalog.Dependencies[i].ID < catalog.Dependencies[j].ID })
	portable := catalog
	portable.Skills = append([]domain.Skill(nil), catalog.Skills...)
	for i := range portable.Skills {
		if fsx.Within(s.Paths.ProjectRoot, portable.Skills[i].Path) {
			if relative, err := filepath.Rel(s.Paths.ProjectRoot, portable.Skills[i].Path); err == nil {
				portable.Skills[i].Path = filepath.ToSlash(relative)
			}
		}
		portable.Skills[i].ProjectRoot = ""
	}
	return saveYAML(filepath.Join(s.Paths.ProjectRoot, ".skm", "project.yaml"), portable)
}

func (s *Store) SyncProjectLock(state domain.State, skills []domain.Skill) error {
	catalog, err := s.LoadProjectCatalog()
	if err != nil {
		return err
	}
	byID := make(map[string]domain.Skill, len(skills))
	for _, value := range skills {
		byID[value.ID] = value
	}
	catalog.Dependencies = nil
	lockFile := domain.LockFile{Version: domain.SchemaVersion}
	for _, installation := range state.Installations {
		if installation.Scope != domain.ScopeProject || installation.ProjectRoot != s.Paths.ProjectRoot {
			continue
		}
		value, ok := byID[installation.SkillID]
		if !ok {
			return fmt.Errorf("cannot lock missing Skill %s", installation.SkillID)
		}
		catalog.Dependencies = append(catalog.Dependencies, domain.ProjectDependency{
			ID: value.ID, Tags: append([]string(nil), value.Tags...), Agents: append([]domain.Agent(nil), installation.Agents...), Mode: installation.Mode,
		})
		lockFile.Skills = append(lockFile.Skills, domain.LockedSkill{
			ID: value.ID, Source: value.Source, Revision: value.Revision, Hash: value.Hash, Tags: append([]string(nil), value.Tags...),
		})
	}
	if err := s.SaveProjectCatalog(catalog); err != nil {
		return err
	}
	sort.Slice(lockFile.Skills, func(i, j int) bool { return lockFile.Skills[i].ID < lockFile.Skills[j].ID })
	return saveYAML(filepath.Join(s.Paths.ProjectRoot, ".skm", "lock.yaml"), lockFile)
}

func (s *Store) HasProjectState(state domain.State) bool {
	for _, installation := range state.Installations {
		if installation.Scope == domain.ScopeProject && installation.ProjectRoot == s.Paths.ProjectRoot {
			return true
		}
	}
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
	if value.Scope == domain.ScopeProject {
		catalog, err := s.LoadProjectCatalog()
		if err != nil {
			return err
		}
		catalog.Skills = upsertSkill(catalog.Skills, value)
		return s.SaveProjectCatalog(catalog)
	}
	catalog, err := s.LoadCatalog()
	if err != nil {
		return err
	}
	catalog.Skills = upsertSkill(catalog.Skills, value)
	return s.SaveCatalog(catalog)
}

func (s *Store) RemoveSkill(id string, scope domain.Scope) error {
	if scope == domain.ScopeProject {
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
	return sources, nil
}

func (s *Store) SaveSources(sources domain.Sources) error {
	sources.Version = domain.SchemaVersion
	sort.Slice(sources.Sources, func(i, j int) bool { return sources.Sources[i].Name < sources.Sources[j].Name })
	return saveYAML(filepath.Join(s.Paths.Home, "sources.yaml"), sources)
}

func (s *Store) LoadState() (domain.State, error) {
	state := domain.State{Version: domain.SchemaVersion}
	if err := loadYAML(filepath.Join(s.Paths.Home, "state", "state.yaml"), &state); err != nil {
		return domain.State{}, err
	}
	return state, nil
}

func (s *Store) SaveState(state domain.State) error {
	state.Version = domain.SchemaVersion
	return saveYAML(filepath.Join(s.Paths.Home, "state", "state.yaml"), state)
}

func (s *Store) ObjectPath(hash, name string) string {
	return filepath.Join(s.Paths.Home, "objects", hash, name)
}

func (s *Store) SourcePath(name string) string {
	return filepath.Join(s.Paths.Home, "sources", name)
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
