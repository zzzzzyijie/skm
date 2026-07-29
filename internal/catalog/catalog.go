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

func (m *Manager) AddLocal(path, sourceName string, scope domain.Scope, tagValues []string) (domain.Skill, error) {
	if !scope.Valid() {
		return domain.Skill{}, fmt.Errorf("invalid scope %q", scope)
	}
	if sourceName == "" {
		if scope == domain.ScopeProject {
			sourceName = "project"
		} else {
			sourceName = "local"
		}
	}
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
	document, err := skill.Validate(path)
	if err != nil {
		return domain.Skill{}, err
	}
	return m.Import(document, sourceName, scope, "", normalizedTags)
}

func (m *Manager) Import(document skill.Document, sourceName string, scope domain.Scope, revision string, tagValues []string) (domain.Skill, error) {
	if !scope.Valid() {
		return domain.Skill{}, fmt.Errorf("invalid scope %q", scope)
	}
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
	projectRoot := ""
	if scope == domain.ScopeProject {
		if err := m.Store.EnsureProject(); err != nil {
			return domain.Skill{}, err
		}
		destination = filepath.Join(m.Store.Paths.ProjectRoot, ".skm", "skills", document.Name)
		projectRoot = m.Store.Paths.ProjectRoot
	}
	samePath := sameFilePath(document.Path, destination)
	if !samePath {
		if _, err := os.Stat(destination); os.IsNotExist(err) || scope == domain.ScopeProject {
			if err := fsx.CopyDirAtomic(document.Path, destination); err != nil {
				return domain.Skill{}, err
			}
		} else if err != nil {
			return domain.Skill{}, err
		}
	}
	value := domain.Skill{
		ID:          sourceName + "/" + document.Name,
		Name:        document.Name,
		Description: document.Description,
		Tags:        normalizedTags,
		Source:      sourceName,
		Scope:       scope,
		Revision:    revision,
		Hash:        document.Hash,
		Path:        destination,
		ProjectRoot: projectRoot,
		Metadata:    document.Metadata,
		AddedAt:     m.Now().UTC(),
	}
	if err := m.Store.UpsertSkill(value); err != nil {
		return domain.Skill{}, err
	}
	return value, nil
}

func (m *Manager) List(scope domain.Scope, tagValues []string) ([]domain.Skill, error) {
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
		if scope != "" && value.Scope != scope {
			continue
		}
		if !tags.MatchAll(value.Tags, tagValues) {
			continue
		}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			if result[i].Scope.Priority() == result[j].Scope.Priority() {
				return result[i].ID < result[j].ID
			}
			return result[i].Scope.Priority() > result[j].Scope.Priority()
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
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Scope.Priority() > matches[j].Scope.Priority()
	})
	highest := matches[0].Scope.Priority()
	var top []domain.Skill
	for _, value := range matches {
		if value.Scope.Priority() == highest {
			top = append(top, value)
		}
	}
	if len(top) > 1 {
		ids := make([]string, 0, len(top))
		for _, value := range top {
			ids = append(ids, value.ID)
		}
		sort.Strings(ids)
		return domain.Skill{}, fmt.Errorf("skill %q is ambiguous; use one of: %s", query, strings.Join(ids, ", "))
	}
	return top[0], nil
}

func (m *Manager) UpdateTags(query string, mutate func([]string) []string) (domain.Skill, error) {
	value, err := m.Resolve(query)
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
	value, err := m.Resolve(query)
	if err != nil {
		return domain.Skill{}, err
	}
	if err := m.Store.RemoveSkill(value.ID, value.Scope); err != nil {
		return domain.Skill{}, err
	}
	return value, nil
}

func sameFilePath(a, b string) bool {
	a, errA := filepath.Abs(a)
	b, errB := filepath.Abs(b)
	return errA == nil && errB == nil && a == b
}
