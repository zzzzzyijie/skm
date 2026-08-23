package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/fsx"
	"github.com/zzzzzyijie/skm/internal/skill"
	"github.com/zzzzzyijie/skm/internal/store"
	"github.com/zzzzzyijie/skm/internal/tags"
)

var validSource = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)

var (
	ErrEditConflict = errors.New("Skill edit conflict")
	ErrNotEditable  = errors.New("Skill is not editable")
)

type Manager struct {
	Store *store.Store
	Now   func() time.Time
}

func New(storage *store.Store) *Manager {
	return &Manager{Store: storage, Now: time.Now}
}

func (m *Manager) AddLocal(path, sourceName string, tagValues []string) (domain.Skill, error) {
	if sourceName == "" {
		sourceName = "local"
	}
	document, err := skill.Validate(path)
	if err != nil {
		return domain.Skill{}, err
	}
	return m.Import(document, sourceName, "", tagValues)
}

func (m *Manager) Snapshot(document skill.Document, sourceName, revision string, tagValues []string) (domain.Skill, error) {
	if !validSource.MatchString(sourceName) {
		return domain.Skill{}, fmt.Errorf("invalid source name %q", sourceName)
	}
	config, err := m.Store.LoadConfig()
	if err != nil {
		return domain.Skill{}, err
	}
	normalizedTags, err := tags.Normalize(tagValues, config.Defaults.Tags)
	if err != nil {
		return domain.Skill{}, err
	}
	destination := m.Store.ObjectPath(document.Hash, document.Name)
	if !sameFilePath(document.Path, destination) {
		if _, err := os.Stat(destination); os.IsNotExist(err) {
			if err := fsx.CopyDirAtomic(document.Path, destination); err != nil {
				return domain.Skill{}, err
			}
		} else if err != nil {
			return domain.Skill{}, err
		}
	}
	return domain.Skill{
		ID:          sourceName + "/" + document.Name,
		Name:        document.Name,
		Description: document.Description,
		Tags:        normalizedTags,
		Source:      sourceName,
		Location:    domain.LocationLibrary,
		Revision:    revision,
		Hash:        document.Hash,
		Path:        destination,
		Metadata:    document.Metadata,
		AddedAt:     m.Now().UTC(),
	}, nil
}

func (m *Manager) Import(document skill.Document, sourceName, revision string, tagValues []string) (domain.Skill, error) {
	value, err := m.Snapshot(document, sourceName, revision, tagValues)
	if err != nil {
		return domain.Skill{}, err
	}
	if err := m.Store.UpsertSkill(value); err != nil {
		return domain.Skill{}, err
	}
	return value, nil
}

// Editability reports whether a Library Skill is an independent local
// snapshot owned by skm. Git-backed and live project-backed Skills remain
// read-only so updates continue to flow from their owning source.
func (m *Manager) Editability(value domain.Skill) (bool, string, error) {
	if value.Location != domain.LocationLibrary {
		return false, "only personal Library Skills can be edited", nil
	}
	if value.Mode == domain.ModeSymlink && value.ProjectRoot != "" {
		return false, "this Skill follows a project; edit the project source or detach it first", nil
	}
	sources, err := m.Store.LoadSources()
	if err != nil {
		return false, "", err
	}
	for _, source := range sources.Sources {
		if source.Name == value.Source {
			return false, "this Skill is managed by a Git source; edit the repository and update the source", nil
		}
	}
	if value.Source != "local" && value.Revision != "" {
		return false, "this Skill was imported from a versioned source and is read-only", nil
	}
	return true, "", nil
}

// ValidateContent validates a proposed SKILL.md in the context of the current
// Skill tree. Auxiliary files are copied into a temporary staging directory
// and remain unchanged.
func (m *Manager) ValidateContent(query, content string) (skill.Document, error) {
	value, err := m.ResolveLibrary(query)
	if err != nil {
		return skill.Document{}, err
	}
	if err := m.requireEditable(value); err != nil {
		return skill.Document{}, err
	}
	document, cleanup, err := m.stageContent(value, content)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return skill.Document{}, err
	}
	if err := checkDocumentName(value, document); err != nil {
		return skill.Document{}, err
	}
	document.Path = ""
	return document, nil
}

// UpdateContent creates a new immutable snapshot containing the proposed
// SKILL.md and repoints the Library entry without mutating the old object.
func (m *Manager) UpdateContent(query, content, baseHash string) (domain.Skill, error) {
	value, err := m.ResolveLibrary(query)
	if err != nil {
		return domain.Skill{}, err
	}
	if err := m.requireEditable(value); err != nil {
		return domain.Skill{}, err
	}
	if err := checkBaseHash(value, baseHash); err != nil {
		return domain.Skill{}, err
	}
	document, cleanup, err := m.stageContent(value, content)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return domain.Skill{}, err
	}
	return m.updateDocument(value, document)
}

// UpdateDirectory replaces an editable Skill from a complete validated
// directory. It is used by the CLI so scripts, references, and assets can be
// updated together with SKILL.md.
func (m *Manager) UpdateDirectory(query, path, baseHash string) (domain.Skill, error) {
	value, err := m.ResolveLibrary(query)
	if err != nil {
		return domain.Skill{}, err
	}
	if err := m.requireEditable(value); err != nil {
		return domain.Skill{}, err
	}
	if err := checkBaseHash(value, baseHash); err != nil {
		return domain.Skill{}, err
	}
	document, err := skill.Validate(path)
	if err != nil {
		return domain.Skill{}, err
	}
	return m.updateDocument(value, document)
}

func (m *Manager) requireEditable(value domain.Skill) error {
	editable, reason, err := m.Editability(value)
	if err != nil {
		return err
	}
	if !editable {
		return fmt.Errorf("%w: %s", ErrNotEditable, reason)
	}
	return nil
}

func checkBaseHash(value domain.Skill, baseHash string) error {
	if baseHash != "" && baseHash != value.Hash {
		return fmt.Errorf("%w: %s changed since editing started", ErrEditConflict, value.ID)
	}
	return nil
}

func (m *Manager) stageContent(value domain.Skill, content string) (skill.Document, func(), error) {
	if int64(len(content)) > skill.MaxSkillMDSize {
		return skill.Document{}, nil, fmt.Errorf("SKILL.md exceeds %d bytes", skill.MaxSkillMDSize)
	}
	if err := m.Store.Ensure(); err != nil {
		return skill.Document{}, nil, err
	}
	temporary, err := os.MkdirTemp(m.Store.Paths.Home, ".skill-edit-")
	if err != nil {
		return skill.Document{}, nil, err
	}
	cleanup := func() { _ = os.RemoveAll(temporary) }
	staged := filepath.Join(temporary, "skill")
	if err := fsx.CopyDirAtomic(value.Path, staged); err != nil {
		return skill.Document{}, cleanup, err
	}
	skillPath := filepath.Join(staged, "SKILL.md")
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(skillPath); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := fsx.AtomicWriteFile(skillPath, []byte(content), mode); err != nil {
		return skill.Document{}, cleanup, err
	}
	document, err := skill.Validate(staged)
	return document, cleanup, err
}

func (m *Manager) updateDocument(value domain.Skill, document skill.Document) (domain.Skill, error) {
	if err := checkDocumentName(value, document); err != nil {
		return domain.Skill{}, err
	}
	snapshot, err := m.Snapshot(document, value.Source, value.Revision, value.Tags)
	if err != nil {
		return domain.Skill{}, err
	}
	changed := snapshot.Hash != value.Hash
	value.Description = snapshot.Description
	value.Hash = snapshot.Hash
	value.Path = snapshot.Path
	value.Metadata = snapshot.Metadata
	if changed {
		value.Revision = ""
	}
	if err := m.Store.UpsertSkill(value); err != nil {
		return domain.Skill{}, err
	}
	return value, nil
}

func checkDocumentName(value domain.Skill, document skill.Document) error {
	if document.Name != value.Name {
		return fmt.Errorf("Skill name cannot change from %q to %q; add it as a new Skill instead", value.Name, document.Name)
	}
	return nil
}

// ImportProject adds a Skill discovered in a registered project to the
// personal Library. Symlink mode keeps the project directory as the live
// source; copy mode creates the usual immutable Library snapshot.
func (m *Manager) ImportProject(document skill.Document, projectRoot string, agent domain.Agent, mode domain.LinkMode, tagValues []string) (domain.Skill, error) {
	if mode != domain.ModeSymlink && mode != domain.ModeCopy {
		return domain.Skill{}, fmt.Errorf("invalid project import mode %q", mode)
	}
	id := "local/" + document.Name
	library, err := m.Store.LoadCatalog()
	if err != nil {
		return domain.Skill{}, err
	}
	for _, existing := range library.Skills {
		if existing.ID == id {
			return domain.Skill{}, fmt.Errorf("Library Skill %s already exists; remove or rename it first", id)
		}
	}

	var value domain.Skill
	if mode == domain.ModeCopy {
		value, err = m.Snapshot(document, "local", "", tagValues)
	} else {
		fallback, snapshotErr := m.Snapshot(document, "local", "", tagValues)
		if snapshotErr != nil {
			return domain.Skill{}, snapshotErr
		}
		config, loadErr := m.Store.LoadConfig()
		if loadErr != nil {
			return domain.Skill{}, loadErr
		}
		normalizedTags, normalizeErr := tags.Normalize(tagValues, config.Defaults.Tags)
		if normalizeErr != nil {
			return domain.Skill{}, normalizeErr
		}
		value = domain.Skill{
			ID:           id,
			Name:         document.Name,
			Description:  document.Description,
			Tags:         normalizedTags,
			Source:       "local",
			Location:     domain.LocationLibrary,
			Hash:         document.Hash,
			Path:         document.Path,
			SnapshotPath: fallback.Path,
			Metadata:     document.Metadata,
			AddedAt:      m.Now().UTC(),
		}
	}
	value.SourcePath = document.Path
	value.ProjectRoot = projectRoot
	value.ProjectAgent = agent
	if relative, relativeErr := filepath.Rel(projectRoot, document.Path); relativeErr == nil && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && relative != ".." {
		value.ProjectPath = filepath.ToSlash(relative)
	}
	value.Mode = mode
	if err := m.Store.UpsertSkill(value); err != nil {
		return domain.Skill{}, err
	}
	return value, nil
}

// DetachProjectLink converts a live project-backed Library entry into an
// independent immutable snapshot while retaining its origin metadata.
func (m *Manager) DetachProjectLink(query string) (domain.Skill, error) {
	value, err := m.ResolveLibrary(query)
	if err != nil {
		return domain.Skill{}, err
	}
	if value.Mode != domain.ModeSymlink || value.ProjectRoot == "" {
		return domain.Skill{}, fmt.Errorf("Library Skill %s is not following a project", value.ID)
	}
	source := value.Path
	document, err := skill.Validate(source)
	if err != nil && value.SnapshotPath != "" {
		source = value.SnapshotPath
		document, err = skill.Validate(source)
	}
	if err != nil {
		return domain.Skill{}, fmt.Errorf("no usable live source or fallback snapshot for %s: %w", value.ID, err)
	}
	snapshot, err := m.Snapshot(document, value.Source, value.Revision, value.Tags)
	if err != nil {
		return domain.Skill{}, err
	}
	value.Path = snapshot.Path
	value.SnapshotPath = ""
	value.Hash = snapshot.Hash
	value.Description = snapshot.Description
	value.Metadata = snapshot.Metadata
	value.Mode = domain.ModeCopy
	if err := m.Store.UpsertSkill(value); err != nil {
		return domain.Skill{}, err
	}
	return value, nil
}

func (m *Manager) Vendor(value domain.Skill, agents []domain.Agent, mode domain.LinkMode, tagValues []string) (domain.Skill, error) {
	if value.Location != domain.LocationLibrary {
		return domain.Skill{}, fmt.Errorf("only Library Skills can be vendored")
	}
	normalizedTags, err := tags.Normalize(tagValues, value.Tags)
	if err != nil {
		return domain.Skill{}, err
	}
	if err := m.Store.EnsureProject(); err != nil {
		return domain.Skill{}, err
	}
	destination := filepath.Join(m.Store.Paths.ProjectRoot, ".skm", "skills", value.Name)
	if _, err := os.Lstat(destination); err == nil {
		return domain.Skill{}, fmt.Errorf("project Skill %q already exists", value.Name)
	} else if !os.IsNotExist(err) {
		return domain.Skill{}, err
	}
	if err := fsx.CopyDirAtomic(value.Path, destination); err != nil {
		return domain.Skill{}, err
	}
	document, err := skill.Validate(destination)
	if err != nil {
		return domain.Skill{}, err
	}
	forkedAt := value.Revision
	if forkedAt == "" {
		forkedAt = value.Hash
	}
	vendored := domain.Skill{
		ID:          "project/" + document.Name,
		Name:        document.Name,
		Description: document.Description,
		Tags:        normalizedTags,
		Source:      "project",
		Location:    domain.LocationProject,
		Hash:        document.Hash,
		Path:        destination,
		ProjectRoot: m.Store.Paths.ProjectRoot,
		ForkedFrom:  value.ID,
		ForkedAt:    forkedAt,
		Agents:      append([]domain.Agent(nil), agents...),
		Mode:        mode,
		Metadata:    document.Metadata,
		AddedAt:     m.Now().UTC(),
	}
	if err := m.Store.UpsertSkill(vendored); err != nil {
		return domain.Skill{}, err
	}
	return vendored, nil
}

func (m *Manager) List(location domain.SkillLocation, tagValues []string) ([]domain.Skill, error) {
	values, err := m.Store.LoadAllSkills()
	if err != nil {
		return nil, err
	}
	if len(tagValues) > 0 {
		tagValues, err = tags.Normalize(tagValues, nil)
		if err != nil {
			return nil, err
		}
	}
	result := make([]domain.Skill, 0, len(values))
	for _, value := range values {
		if location != "" && value.Location != location {
			continue
		}
		if !tags.MatchAll(value.Tags, tagValues) {
			continue
		}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			return result[i].ID < result[j].ID
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func (m *Manager) Resolve(query string) (domain.Skill, error) {
	values, err := m.Store.LoadAllSkills()
	if err != nil {
		return domain.Skill{}, err
	}
	return Resolve(values, query)
}

func (m *Manager) ResolveLibrary(query string) (domain.Skill, error) {
	catalog, err := m.Store.LoadCatalog()
	if err != nil {
		return domain.Skill{}, err
	}
	return Resolve(catalog.Skills, query)
}

func Resolve(values []domain.Skill, query string) (domain.Skill, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return domain.Skill{}, fmt.Errorf("skill name is required")
	}
	if strings.Contains(query, "/") {
		for _, value := range values {
			if value.ID == query {
				return value, nil
			}
		}
		return domain.Skill{}, fmt.Errorf("skill %q not found", query)
	}
	var matches []domain.Skill
	for _, value := range values {
		if value.Name == query {
			matches = append(matches, value)
		}
	}
	if len(matches) == 0 {
		return domain.Skill{}, fmt.Errorf("skill %q not found", query)
	}
	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for _, value := range matches {
			ids = append(ids, value.ID)
		}
		sort.Strings(ids)
		return domain.Skill{}, fmt.Errorf("skill %q is ambiguous; use one of: %s", query, strings.Join(ids, ", "))
	}
	return matches[0], nil
}

func (m *Manager) UpdateTags(query string, mutate func([]string) []string) (domain.Skill, error) {
	value, err := m.ResolveLibrary(query)
	if err != nil {
		return domain.Skill{}, err
	}
	config, err := m.Store.LoadConfig()
	if err != nil {
		return domain.Skill{}, err
	}
	updated := mutate(append([]string(nil), value.Tags...))
	if len(updated) == 0 {
		updated = config.Defaults.Tags
	}
	value.Tags, err = tags.Normalize(updated, config.Defaults.Tags)
	if err != nil {
		return domain.Skill{}, err
	}
	if err := m.Store.UpsertSkill(value); err != nil {
		return domain.Skill{}, err
	}
	return value, nil
}

func (m *Manager) Remove(query string) (domain.Skill, error) {
	value, err := m.ResolveLibrary(query)
	if err != nil {
		return domain.Skill{}, err
	}
	if err := m.Store.RemoveSkill(value.ID, domain.LocationLibrary); err != nil {
		return domain.Skill{}, err
	}
	return value, nil
}

func sameFilePath(a, b string) bool {
	a, errA := filepath.Abs(a)
	b, errB := filepath.Abs(b)
	return errA == nil && errB == nil && a == b
}
