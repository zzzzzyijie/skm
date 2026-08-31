package source

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/fsx"
	"github.com/zzzzzyijie/skm/internal/skill"
	"github.com/zzzzzyijie/skm/internal/store"
	"github.com/zzzzzyijie/skm/internal/tags"
)

var validSourceName = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)

type GitManager struct {
	Store   *store.Store
	Catalog *catalog.Manager
	GitPath string
	Now     func() time.Time
}

type RemovalResult struct {
	Name            string         `json:"name"`
	Source          *domain.Source `json:"source,omitempty"`
	BindingRemoved  bool           `json:"bindingRemoved"`
	CheckoutPath    string         `json:"checkoutPath,omitempty"`
	CheckoutRemoved bool           `json:"checkoutRemoved"`
}

type SkillCandidate struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Path        string `json:"path"`
	Valid       bool   `json:"valid"`
	Error       string `json:"error,omitempty"`
}

type Inspection struct {
	Revision string           `json:"revision"`
	Skills   []SkillCandidate `json:"skills"`
}

type sourceCheckout struct {
	TempRoot string
	Path     string
	Revision string
}

func NewGitManager(storage *store.Store, catalogManager *catalog.Manager) *GitManager {
	return &GitManager{Store: storage, Catalog: catalogManager, GitPath: "git", Now: time.Now}
}

func (m *GitManager) Add(value domain.Source) (domain.Source, []domain.Skill, error) {
	return m.AddSelected(value, nil)
}

func (m *GitManager) AddSelected(value domain.Source, skillNames []string) (domain.Source, []domain.Skill, error) {
	if err := validateSource(&value); err != nil {
		return domain.Source{}, nil, err
	}
	config, err := m.Store.LoadConfig()
	if err != nil {
		return domain.Source{}, nil, err
	}
	value.Tags, err = tags.Normalize(value.Tags, config.Defaults.Tags)
	if err != nil {
		return domain.Source{}, nil, err
	}
	sources, err := m.Store.LoadSources()
	if err != nil {
		return domain.Source{}, nil, err
	}
	for _, existing := range sources.Sources {
		if existing.Name == value.Name {
			return domain.Source{}, nil, fmt.Errorf("source %q already exists", value.Name)
		}
	}
	updated, imported, err := m.fetchSelected(value, true, skillNames)
	if err != nil {
		return domain.Source{}, nil, err
	}
	sources.Sources = append(sources.Sources, updated)
	if err := m.Store.SaveSources(sources); err != nil {
		return domain.Source{}, nil, err
	}
	return updated, imported, nil
}

// Inspect clones a source into temporary storage and reports every discovered
// Skill independently. It never changes source bindings, checkouts, or Library
// snapshots, and invalid candidates remain visible for repository previews.
func (m *GitManager) Inspect(value domain.Source) (Inspection, error) {
	if err := validateSource(&value); err != nil {
		return Inspection{}, err
	}
	checkout, err := m.checkout(value)
	if err != nil {
		return Inspection{}, err
	}
	defer os.RemoveAll(checkout.TempRoot)

	directories, err := discoverSkills(checkout.Path, value.Paths)
	if err != nil {
		return Inspection{}, err
	}
	result := Inspection{Revision: checkout.Revision, Skills: make([]SkillCandidate, 0, len(directories))}
	for _, directory := range directories {
		relative, err := filepath.Rel(checkout.Path, directory)
		if err != nil {
			return Inspection{}, err
		}
		candidate := SkillCandidate{
			Name: filepath.Base(directory),
			Path: filepath.ToSlash(relative),
		}
		if manifest, manifestErr := skill.ReadManifest(directory); manifestErr == nil {
			candidate.Name = manifest.Name
			candidate.Description = manifest.Description
		}
		if document, validateErr := skill.Validate(directory); validateErr != nil {
			candidate.Error = relativeSkillError(directory, validateErr)
		} else {
			candidate.Name = document.Name
			candidate.Description = document.Description
			candidate.Valid = true
		}
		result.Skills = append(result.Skills, candidate)
	}
	return result, nil
}

// RegisterBinding stores a validated source binding without fetching content.
// Workspace sync uses it so a binding published from another device is adopted
// even when the fetch itself fails; a later source sync retries the fetch.
func (m *GitManager) RegisterBinding(value domain.Source) error {
	if err := validateSource(&value); err != nil {
		return err
	}
	sources, err := m.Store.LoadSources()
	if err != nil {
		return err
	}
	replaced := false
	for index := range sources.Sources {
		if sources.Sources[index].Name == value.Name {
			sources.Sources[index] = value
			replaced = true
			break
		}
	}
	if !replaced {
		sources.Sources = append(sources.Sources, value)
	}
	return m.Store.SaveSources(sources)
}

// FetchPinned stores immutable snapshots for a project dependency without
// changing the user's source binding or current Library entries.
func (m *GitManager) FetchPinned(value domain.Source) ([]domain.Skill, error) {
	if err := validateSource(&value); err != nil {
		return nil, err
	}
	_, imported, err := m.fetch(value, false)
	return imported, err
}

func (m *GitManager) Update(names []string) ([]domain.Source, []domain.Skill, error) {
	sources, err := m.Store.LoadSources()
	if err != nil {
		return nil, nil, err
	}
	selected := make(map[string]struct{}, len(names))
	for _, name := range names {
		selected[name] = struct{}{}
	}
	if len(sources.Sources) == 0 {
		return nil, nil, fmt.Errorf("no Git sources configured")
	}
	var updatedSources []domain.Source
	var imported []domain.Skill
	matched := 0
	for i, value := range sources.Sources {
		if len(selected) > 0 {
			if _, ok := selected[value.Name]; !ok {
				continue
			}
		}
		matched++
		updated, skills, err := m.syncOne(value)
		if err != nil {
			return updatedSources, imported, fmt.Errorf("update source %s: %w", value.Name, err)
		}
		sources.Sources[i] = updated
		updatedSources = append(updatedSources, updated)
		imported = append(imported, skills...)
	}
	if matched == 0 {
		return nil, nil, fmt.Errorf("none of the requested sources were found")
	}
	if err := m.Store.SaveSources(sources); err != nil {
		return nil, nil, err
	}
	return updatedSources, imported, nil
}

func (m *GitManager) Remove(name string) (RemovalResult, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if !validSourceName.MatchString(name) {
		return RemovalResult{}, fmt.Errorf("invalid source name %q", name)
	}
	sources, err := m.Store.LoadSources()
	if err != nil {
		return RemovalResult{}, err
	}
	var removed domain.Source
	result := sources.Sources[:0]
	for _, value := range sources.Sources {
		if value.Name == name {
			removed = value
			continue
		}
		result = append(result, value)
	}
	checkoutRoot := filepath.Join(m.Store.Paths.Home, "sources")
	checkoutPath := m.Store.SourcePath(name)
	checkoutExists, err := removableCheckoutExists(checkoutRoot, checkoutPath)
	if err != nil {
		return RemovalResult{}, err
	}
	if removed.Name == "" && !checkoutExists {
		return RemovalResult{}, fmt.Errorf("source %q not found", name)
	}
	removal := RemovalResult{
		Name:           name,
		CheckoutPath:   checkoutPath,
		BindingRemoved: removed.Name != "",
	}
	if removed.Name != "" {
		removal.Source = &removed
		sources.Sources = result
		if err := m.Store.SaveSources(sources); err != nil {
			return RemovalResult{}, err
		}
	}
	if checkoutExists {
		if err := os.RemoveAll(checkoutPath); err != nil {
			return removal, fmt.Errorf("failed to remove source checkout %s: %w", checkoutPath, err)
		}
		removal.CheckoutRemoved = true
	}
	return removal, nil
}

func removableCheckoutExists(root, path string) (bool, error) {
	if !fsx.Within(root, path) {
		return false, fmt.Errorf("source checkout path escapes %s", root)
	}
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return false, err
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("refusing source checkout through non-directory %s", root)
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("refusing to remove non-directory source checkout %s", path)
	}
	return true, nil
}

func (m *GitManager) syncOne(value domain.Source) (domain.Source, []domain.Skill, error) {
	if err := validateSource(&value); err != nil {
		return domain.Source{}, nil, err
	}
	return m.fetch(value, true)
}

func (m *GitManager) fetch(value domain.Source, persist bool) (domain.Source, []domain.Skill, error) {
	return m.fetchSelected(value, persist, nil)
}

func (m *GitManager) fetchSelected(value domain.Source, persist bool, requestedSkills []string) (domain.Source, []domain.Skill, error) {
	checkout, err := m.checkout(value)
	if err != nil {
		return domain.Source{}, nil, err
	}
	defer os.RemoveAll(checkout.TempRoot)
	directories, err := discoverSkills(checkout.Path, value.Paths)
	if err != nil {
		return domain.Source{}, nil, err
	}
	documents := make([]skill.Document, 0, len(directories))
	for _, directory := range directories {
		document, err := skill.Validate(directory)
		if err != nil {
			return domain.Source{}, nil, fmt.Errorf("validate %s: %w", directory, err)
		}
		documents = append(documents, document)
	}
	if len(requestedSkills) > 0 {
		documents, value.Paths, err = selectDocuments(checkout.Path, documents, requestedSkills)
		if err != nil {
			return domain.Source{}, nil, err
		}
	}
	if err := ensureUniqueDocuments(checkout.Path, documents); err != nil {
		return domain.Source{}, nil, err
	}
	existingCatalog, err := m.Store.LoadCatalog()
	if err != nil {
		return domain.Source{}, nil, err
	}
	existingTags := make(map[string][]string)
	for _, existing := range existingCatalog.Skills {
		if existing.Source == value.Name {
			existingTags[existing.ID] = existing.Tags
		}
	}
	var imported []domain.Skill
	for _, document := range documents {
		tagValues := value.Tags
		if preserved := existingTags[value.Name+"/"+document.Name]; len(preserved) > 0 {
			tagValues = preserved
		}
		importedSkill, err := m.Catalog.Snapshot(document, value.Name, checkout.Revision, tagValues)
		if err != nil {
			return domain.Source{}, imported, err
		}
		relative, relErr := filepath.Rel(checkout.Path, document.Path)
		if relErr != nil {
			return domain.Source{}, imported, relErr
		}
		importedSkill.SourcePath = filepath.ToSlash(relative)
		imported = append(imported, importedSkill)
	}
	if persist {
		if err := fsx.ReplacePath(checkout.Path, m.Store.SourcePath(value.Name)); err != nil {
			return domain.Source{}, imported, err
		}
		updatedCatalog := existingCatalog
		for _, importedSkill := range imported {
			updatedCatalog.Skills = upsertSourceSkill(updatedCatalog.Skills, importedSkill)
		}
		if err := m.Store.SaveCatalog(updatedCatalog); err != nil {
			return domain.Source{}, imported, err
		}
	}
	value.Revision = checkout.Revision
	value.UpdatedAt = m.Now().UTC()
	return value, imported, nil
}

func (m *GitManager) checkout(value domain.Source) (sourceCheckout, error) {
	if _, err := exec.LookPath(m.GitPath); err != nil {
		return sourceCheckout{}, fmt.Errorf("git executable not found")
	}
	parent := filepath.Join(m.Store.Paths.Home, "sources")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return sourceCheckout{}, err
	}
	tempRoot, err := os.MkdirTemp(parent, ".skm-source-")
	if err != nil {
		return sourceCheckout{}, err
	}
	checkout := filepath.Join(tempRoot, "repo")
	if err := m.runGit(value.URL, "clone", "--no-recurse-submodules", "--", value.URL, checkout); err != nil {
		_ = os.RemoveAll(tempRoot)
		return sourceCheckout{}, err
	}
	if value.Ref != "" {
		if err := m.runGit(value.URL, "-C", checkout, "checkout", "--detach", value.Ref); err != nil {
			_ = os.RemoveAll(tempRoot)
			return sourceCheckout{}, fmt.Errorf("checkout ref %q: %w", value.Ref, err)
		}
	}
	revisionBytes, err := exec.Command(m.GitPath, "-C", checkout, "rev-parse", "HEAD").Output()
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		return sourceCheckout{}, fmt.Errorf("resolve Git revision: %w", err)
	}
	return sourceCheckout{TempRoot: tempRoot, Path: checkout, Revision: strings.TrimSpace(string(revisionBytes))}, nil
}

func upsertSourceSkill(values []domain.Skill, value domain.Skill) []domain.Skill {
	for index := range values {
		if values[index].ID == value.ID {
			values[index] = value
			return values
		}
	}
	return append(values, value)
}

func ensureUniqueDocuments(root string, documents []skill.Document) error {
	seen := make(map[string]string, len(documents))
	for _, document := range documents {
		relative, err := filepath.Rel(root, document.Path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if existing, ok := seen[document.Name]; ok {
			return fmt.Errorf("Skill %q appears more than once in source (%s and %s); select only one path", document.Name, existing, relative)
		}
		seen[document.Name] = relative
	}
	return nil
}

func relativeSkillError(root string, err error) string {
	message := err.Error()
	cleanRoot := filepath.Clean(root)
	message = strings.ReplaceAll(message, cleanRoot+string(os.PathSeparator), "")
	return strings.ReplaceAll(message, cleanRoot, ".")
}

func selectDocuments(root string, documents []skill.Document, requestedSkills []string) ([]skill.Document, []string, error) {
	requested := make(map[string]struct{}, len(requestedSkills))
	for _, name := range requestedSkills {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, nil, fmt.Errorf("requested Skill name cannot be empty")
		}
		requested[name] = struct{}{}
	}

	available := make([]string, 0, len(documents))
	found := make(map[string]string, len(requested))
	selected := make([]skill.Document, 0, len(requested))
	paths := make([]string, 0, len(requested))
	for _, document := range documents {
		available = append(available, document.Name)
		if _, ok := requested[document.Name]; !ok {
			continue
		}
		relative, err := filepath.Rel(root, document.Path)
		if err != nil {
			return nil, nil, err
		}
		relative = filepath.ToSlash(relative)
		if existing, ok := found[document.Name]; ok {
			return nil, nil, fmt.Errorf("Skill %q appears more than once in source (%s and %s)", document.Name, existing, relative)
		}
		found[document.Name] = relative
		selected = append(selected, document)
		paths = append(paths, relative)
	}

	var missing []string
	for name := range requested {
		if _, ok := found[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		sort.Strings(available)
		return nil, nil, fmt.Errorf("requested Skill(s) not found: %s; available: %s", strings.Join(missing, ", "), strings.Join(available, ", "))
	}
	return selected, paths, nil
}

func validateSource(value *domain.Source) error {
	value.Name = strings.ToLower(strings.TrimSpace(value.Name))
	if !ValidName(value.Name) {
		return fmt.Errorf("invalid source name %q", value.Name)
	}
	return ValidateURL(value.URL)
}

// ValidName reports whether name is a usable source binding name.
func ValidName(name string) bool {
	return validSourceName.MatchString(name)
}

// ValidateURL rejects empty URLs and URLs with embedded credentials.
func ValidateURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("source URL is required")
	}
	if parsed, err := url.Parse(raw); err == nil && parsed.User != nil {
		return fmt.Errorf("source URL must not contain credentials; use Git credential storage or SSH")
	}
	return nil
}

func (m *GitManager) runGit(secret string, args ...string) error {
	command := exec.Command(m.GitPath, args...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := command.CombinedOutput()
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(string(output))
	if secret != "" {
		message = strings.ReplaceAll(message, secret, "<redacted-url>")
	}
	if message == "" {
		message = err.Error()
	}
	return fmt.Errorf("git command failed: %s", message)
}

func discoverSkills(root string, boundPaths []string) ([]string, error) {
	if len(boundPaths) > 0 {
		result := make([]string, 0, len(boundPaths))
		seen := make(map[string]struct{})
		for _, value := range boundPaths {
			if filepath.IsAbs(value) {
				return nil, fmt.Errorf("source skill path must be relative: %s", value)
			}
			path := filepath.Clean(filepath.Join(root, value))
			if !fsx.Within(root, path) {
				return nil, fmt.Errorf("source skill path escapes repository: %s", value)
			}
			if filepath.Base(path) == "SKILL.md" {
				path = filepath.Dir(path)
			}
			if _, err := os.Stat(filepath.Join(path, "SKILL.md")); err != nil {
				return nil, fmt.Errorf("bound skill path %s: %w", value, err)
			}
			if _, ok := seen[path]; !ok {
				seen[path] = struct{}{}
				result = append(result, path)
			}
		}
		sort.Strings(result)
		return result, nil
	}
	var result []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}
		if !entry.IsDir() && entry.Name() == "SKILL.md" {
			result = append(result, filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no SKILL.md files found in source")
	}
	sort.Strings(result)
	return result, nil
}
