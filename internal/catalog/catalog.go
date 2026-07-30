package catalog

import (
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
