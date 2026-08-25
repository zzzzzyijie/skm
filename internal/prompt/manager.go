package prompt

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
	"github.com/zzzzzyijie/skm/internal/store"
	"github.com/zzzzzyijie/skm/internal/tags"
)

var validSource = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)

var ErrEditConflict = errors.New("Prompt edit conflict")

type Manager struct {
	Store *store.Store
	Now   func() time.Time
}

func New(storage *store.Store) *Manager {
	return &Manager{Store: storage, Now: time.Now}
}

func (m *Manager) Add(path, sourceName string, tagValues []string) (domain.Prompt, error) {
	document, err := Validate(path)
	if err != nil {
		return domain.Prompt{}, err
	}
	return m.CreateDocument(document, sourceName, tagValues)
}

func (m *Manager) Create(content, sourceName string, tagValues []string) (domain.Prompt, error) {
	document, err := Parse([]byte(content))
	if err != nil {
		return domain.Prompt{}, err
	}
	return m.CreateDocument(document, sourceName, tagValues)
}

func (m *Manager) CreateDocument(document Document, sourceName string, tagValues []string) (domain.Prompt, error) {
	if sourceName == "" {
		sourceName = "local"
	}
	if !validSource.MatchString(sourceName) {
		return domain.Prompt{}, fmt.Errorf("invalid source name %q", sourceName)
	}
	id := sourceName + "/" + document.Name
	catalog, err := m.Store.LoadPromptCatalog()
	if err != nil {
		return domain.Prompt{}, err
	}
	for _, existing := range catalog.Prompts {
		if existing.ID == id {
			return domain.Prompt{}, fmt.Errorf("Prompt %s already exists; update or rename it first", id)
		}
	}
	normalizedTags, err := m.normalizeTags(tagValues, document.Tags, nil)
	if err != nil {
		return domain.Prompt{}, err
	}
	destination, err := m.snapshot(document)
	if err != nil {
		return domain.Prompt{}, err
	}
	now := m.Now().UTC()
	value := domain.Prompt{
		ID: id, Name: document.Name, Description: document.Description, Tags: normalizedTags,
		Source: sourceName, Hash: document.Hash, Path: destination, Variables: document.Variables,
		AddedAt: now, UpdatedAt: now,
	}
	if err := m.Store.UpsertPrompt(value); err != nil {
		return domain.Prompt{}, err
	}
	return value, nil
}

func (m *Manager) Update(query, content, baseHash string, tagValues []string) (domain.Prompt, error) {
	value, err := m.Resolve(query)
	if err != nil {
		return domain.Prompt{}, err
	}
	if baseHash != "" && baseHash != value.Hash {
		return domain.Prompt{}, fmt.Errorf("%w: %s changed since editing started", ErrEditConflict, value.ID)
	}
	document, err := Parse([]byte(content))
	if err != nil {
		return domain.Prompt{}, err
	}
	if document.Name != value.Name {
		return domain.Prompt{}, fmt.Errorf("Prompt name cannot change from %q to %q; duplicate it instead", value.Name, document.Name)
	}
	normalizedTags, err := m.normalizeTags(tagValues, document.Tags, value.Tags)
	if err != nil {
		return domain.Prompt{}, err
	}
	destination, err := m.snapshot(document)
	if err != nil {
		return domain.Prompt{}, err
	}
	oldHash := value.Hash
	value.Description = document.Description
	value.Tags = normalizedTags
	value.Hash = document.Hash
	value.Path = destination
	value.Variables = document.Variables
	value.UpdatedAt = m.Now().UTC()
	if err := m.Store.UpsertPrompt(value); err != nil {
		return domain.Prompt{}, err
	}
	if oldHash != value.Hash {
		_ = m.removeObjectIfUnreferenced(oldHash, value.Name)
	}
	return value, nil
}

func (m *Manager) List(tagValues []string) ([]domain.Prompt, error) {
	catalog, err := m.Store.LoadPromptCatalog()
	if err != nil {
		return nil, err
	}
	if len(tagValues) > 0 {
		tagValues, err = tags.Normalize(tagValues, nil)
		if err != nil {
			return nil, err
		}
	}
	result := make([]domain.Prompt, 0, len(catalog.Prompts))
	for _, value := range catalog.Prompts {
		if tags.MatchAll(value.Tags, tagValues) {
			result = append(result, value)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			return result[i].ID < result[j].ID
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func (m *Manager) Resolve(query string) (domain.Prompt, error) {
	catalog, err := m.Store.LoadPromptCatalog()
	if err != nil {
		return domain.Prompt{}, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return domain.Prompt{}, fmt.Errorf("prompt name is required")
	}
	if strings.Contains(query, "/") {
		for _, value := range catalog.Prompts {
			if value.ID == query {
				return value, nil
			}
		}
		return domain.Prompt{}, fmt.Errorf("prompt %q not found", query)
	}
	var matches []domain.Prompt
	for _, value := range catalog.Prompts {
		if value.Name == query {
			matches = append(matches, value)
		}
	}
	if len(matches) == 0 {
		return domain.Prompt{}, fmt.Errorf("prompt %q not found", query)
	}
	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for _, value := range matches {
			ids = append(ids, value.ID)
		}
		sort.Strings(ids)
		return domain.Prompt{}, fmt.Errorf("prompt %q is ambiguous; use one of: %s", query, strings.Join(ids, ", "))
	}
	return matches[0], nil
}

func (m *Manager) Read(query string) (domain.Prompt, Document, error) {
	value, err := m.Resolve(query)
	if err != nil {
		return domain.Prompt{}, Document{}, err
	}
	document, err := Validate(value.Path)
	if err != nil {
		return domain.Prompt{}, Document{}, err
	}
	return value, document, nil
}

func (m *Manager) Remove(query string) (domain.Prompt, error) {
	value, err := m.Resolve(query)
	if err != nil {
		return domain.Prompt{}, err
	}
	if err := m.Store.RemovePrompt(value.ID); err != nil {
		return domain.Prompt{}, err
	}
	if err := m.removeObjectIfUnreferenced(value.Hash, value.Name); err != nil {
		return domain.Prompt{}, fmt.Errorf("removed %s but failed to clean its snapshot: %w", value.ID, err)
	}
	return value, nil
}

func (m *Manager) snapshot(document Document) (string, error) {
	destination := m.Store.PromptObjectPath(document.Hash, document.Name)
	path := filepath.Join(destination, "PROMPT.md")
	if data, err := os.ReadFile(path); err == nil && string(data) == document.Content {
		return destination, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err := fsx.AtomicWriteFile(path, []byte(document.Content), 0o644); err != nil {
		return "", err
	}
	return destination, nil
}

func (m *Manager) normalizeTags(explicit, documentTags, fallback []string) ([]string, error) {
	values := explicit
	if values == nil {
		values = documentTags
	}
	if values == nil {
		values = fallback
	}
	config, err := m.Store.LoadConfig()
	if err != nil {
		return nil, err
	}
	return tags.Normalize(values, config.Defaults.PromptTags)
}

func (m *Manager) removeObjectIfUnreferenced(hash, name string) error {
	catalog, err := m.Store.LoadPromptCatalog()
	if err != nil {
		return err
	}
	for _, value := range catalog.Prompts {
		if value.Hash == hash && value.Name == name {
			return nil
		}
	}
	destination := m.Store.PromptObjectPath(hash, name)
	if err := os.RemoveAll(destination); err != nil {
		return err
	}
	parent := filepath.Dir(destination)
	entries, err := os.ReadDir(parent)
	if err == nil && len(entries) == 0 {
		return os.Remove(parent)
	}
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
