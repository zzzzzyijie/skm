package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/fsx"
	promptpkg "github.com/zzzzzyijie/skm/internal/prompt"
	"github.com/zzzzzyijie/skm/internal/skill"
	"github.com/zzzzzyijie/skm/internal/store"
	"github.com/zzzzzyijie/skm/internal/tags"
	"gopkg.in/yaml.v3"
)

var ErrConflict = errors.New("workspace sync has conflicts")

type Change struct {
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	Action     string `json:"action"`
	Reason     string `json:"reason,omitempty"`
	LocalHash  string `json:"localHash,omitempty"`
	RemoteHash string `json:"remoteHash,omitempty"`
	BaseHash   string `json:"baseHash,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

type Preview struct {
	Configured      bool                    `json:"configured"`
	Config          *domain.WorkspaceConfig `json:"config,omitempty"`
	BaseRevision    string                  `json:"baseRevision,omitempty"`
	RemoteRevision  string                  `json:"remoteRevision,omitempty"`
	Skills          int                     `json:"skills"`
	Prompts         int                     `json:"prompts"`
	Uploads         int                     `json:"uploads"`
	Downloads       int                     `json:"downloads"`
	Deletes         int                     `json:"deletes"`
	Conflicts       int                     `json:"conflicts"`
	Changes         []Change                `json:"changes"`
	LastSyncedAt    time.Time               `json:"lastSyncedAt,omitempty"`
	RemoteAvailable bool                    `json:"remoteAvailable"`
}

type Result struct {
	Preview         Preview      `json:"preview"`
	Revision        string       `json:"revision"`
	Committed       bool         `json:"committed"`
	Applied         bool         `json:"applied"`
	DeploymentError string       `json:"deploymentError,omitempty"`
	SyncedAt        time.Time    `json:"syncedAt"`
	Plan            *domain.Plan `json:"plan,omitempty"`
}

type Manager struct {
	Store   *store.Store
	GitPath string
	Now     func() time.Time
}

type contentItem struct {
	Entry domain.WorkspaceEntry
	Path  string
}

type prepared struct {
	preview       Preview
	config        domain.WorkspaceConfig
	state         domain.WorkspaceState
	checkout      string
	tempRoot      string
	workspaceRoot string
	manifestPath  string
	manifestFound bool
	localSkills   map[string]contentItem
	localPrompts  map[string]contentItem
	remoteSkills  map[string]contentItem
	remotePrompts map[string]contentItem
}

func New(storage *store.Store) *Manager {
	return &Manager{Store: storage, GitPath: "git", Now: time.Now}
}

func (m *Manager) Configure(value domain.WorkspaceConfig) (domain.WorkspaceConfig, error) {
	if err := validateConfig(&value); err != nil {
		return domain.WorkspaceConfig{}, err
	}
	checkout, tempRoot, _, _, err := m.clone(value)
	if tempRoot != "" {
		defer os.RemoveAll(tempRoot)
	}
	if err != nil {
		return domain.WorkspaceConfig{}, err
	}
	workspaceRoot, _, manifest, _, err := inspectCheckout(checkout, value)
	if err != nil {
		return domain.WorkspaceConfig{}, err
	}
	if _, _, err := loadRemote(workspaceRoot, manifest); err != nil {
		return domain.WorkspaceConfig{}, err
	}

	previous, err := m.Store.LoadWorkspaceConfig()
	if err != nil {
		return domain.WorkspaceConfig{}, err
	}
	previousState, err := m.Store.LoadWorkspaceState()
	if err != nil {
		return domain.WorkspaceConfig{}, err
	}
	value.Version = domain.WorkspaceSchemaVersion
	value.UpdatedAt = m.Now().UTC()
	if err := m.Store.SaveWorkspaceConfig(value); err != nil {
		return domain.WorkspaceConfig{}, err
	}
	if previous.URL != value.URL || previous.Ref != value.Ref || previous.Root != value.Root {
		if err := m.Store.SaveWorkspaceState(domain.WorkspaceState{
			Version: domain.WorkspaceSchemaVersion, SkillBases: map[string]string{}, PromptBases: map[string]string{},
		}); err != nil {
			return domain.WorkspaceConfig{}, errors.Join(err, m.Store.SaveWorkspaceConfig(previous), m.Store.SaveWorkspaceState(previousState))
		}
	}
	return value, nil
}

func (m *Manager) Config() (domain.WorkspaceConfig, error) {
	return m.Store.LoadWorkspaceConfig()
}

func (m *Manager) Preview() (Preview, error) {
	value, err := m.prepare()
	if err != nil {
		return Preview{}, err
	}
	defer os.RemoveAll(value.tempRoot)
	return value.preview, nil
}

func (m *Manager) Apply() (Result, error) {
	return m.ApplyResolved(nil)
}

func (m *Manager) ApplyResolved(resolutions map[string]string) (Result, error) {
	value, err := m.prepare()
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(value.tempRoot)
	resolveConflicts(&value, resolutions)
	if value.preview.Conflicts > 0 {
		return Result{Preview: value.preview}, fmt.Errorf("%w: resolve %d item(s) before syncing", ErrConflict, value.preview.Conflicts)
	}

	finalSkills := cloneItems(value.remoteSkills)
	finalPrompts := cloneItems(value.remotePrompts)
	for _, change := range value.preview.Changes {
		var local map[string]contentItem
		var final map[string]contentItem
		if change.Kind == "skill" {
			local, final = value.localSkills, finalSkills
		} else {
			local, final = value.localPrompts, finalPrompts
		}
		switch change.Action {
		case "upload":
			item := local[change.ID]
			if previous, ok := final[change.ID]; ok && previous.Entry.Path != "" {
				item.Entry.Path = previous.Entry.Path
			}
			if err := m.writeItem(value.workspaceRoot, change.Kind, item); err != nil {
				return Result{Preview: value.preview}, err
			}
			item.Path = workspaceItemPath(value.workspaceRoot, change.Kind, item.Entry.Path)
			final[change.ID] = item
		case "delete-remote":
			if item, ok := final[change.ID]; ok {
				if err := removeWorkspaceItem(value.workspaceRoot, change.Kind, item.Entry.Path); err != nil {
					return Result{Preview: value.preview}, err
				}
			}
			delete(final, change.ID)
		}
	}

	manifest := domain.WorkspaceManifest{
		Version: domain.WorkspaceSchemaVersion,
		Skills:  entriesFromItems(finalSkills),
		Prompts: entriesFromItems(finalPrompts),
	}
	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		return Result{Preview: value.preview}, err
	}
	if err := fsx.AtomicWriteFile(value.manifestPath, manifestData, 0o644); err != nil {
		return Result{Preview: value.preview}, err
	}
	if err := m.git(value.config.URL, "-C", value.checkout, "add", "-A", "--", "."); err != nil {
		return Result{Preview: value.preview}, err
	}
	committed, err := m.commitIfNeeded(value)
	if err != nil {
		return Result{Preview: value.preview}, err
	}
	revision, err := gitOutput(m.GitPath, value.checkout, "rev-parse", "HEAD")
	if err != nil {
		return Result{Preview: value.preview}, err
	}
	if committed {
		refspec := "HEAD:refs/heads/" + value.config.Ref
		if err := m.git(value.config.URL, "-C", value.checkout, "push", "origin", refspec); err != nil {
			return Result{Preview: value.preview}, fmt.Errorf("publish workspace (remote changed; preview again before retrying): %w", err)
		}
	}

	if err := m.applyLocal(value.workspaceRoot, finalSkills, finalPrompts, revision); err != nil {
		return Result{Preview: value.preview, Revision: revision, Committed: committed}, fmt.Errorf("workspace was published but local apply failed; retry sync to recover: %w", err)
	}
	if err := fsx.ReplacePath(value.checkout, m.Store.WorkspaceCheckoutPath()); err != nil {
		return Result{Preview: value.preview, Revision: revision, Committed: committed, Applied: true}, fmt.Errorf("workspace applied but checkout cache failed: %w", err)
	}
	return Result{
		Preview: value.preview, Revision: revision, Committed: committed, Applied: true, SyncedAt: m.Now().UTC(),
	}, nil
}

func resolveConflicts(value *prepared, resolutions map[string]string) {
	if value.preview.Conflicts == 0 {
		return
	}
	for index := range value.preview.Changes {
		change := &value.preview.Changes[index]
		if change.Action != "conflict" {
			continue
		}
		choice := resolutions[change.Kind+":"+change.ID]
		if choice != "local" && choice != "remote" {
			continue
		}
		if change.Reason == "enabled-skill-delete" && choice == "remote" {
			continue
		}
		var local, remote map[string]contentItem
		if change.Kind == "skill" {
			local, remote = value.localSkills, value.remoteSkills
		} else {
			local, remote = value.localPrompts, value.remotePrompts
		}
		_, hasLocal := local[change.ID]
		_, hasRemote := remote[change.ID]
		if choice == "local" {
			if hasLocal {
				change.Action = "upload"
			} else {
				change.Action = "delete-remote"
			}
		} else if hasRemote {
			change.Action = "download"
		} else {
			change.Action = "delete-local"
		}
		change.Reason = ""
		change.Detail = ""
	}
	recountPreview(&value.preview)
}

func recountPreview(preview *Preview) {
	preview.Uploads, preview.Downloads, preview.Deletes, preview.Conflicts = 0, 0, 0, 0
	for _, change := range preview.Changes {
		switch change.Action {
		case "upload":
			preview.Uploads++
		case "download":
			preview.Downloads++
		case "delete-local", "delete-remote":
			preview.Deletes++
		}
		if change.Action == "conflict" {
			preview.Conflicts++
		}
	}
}

func (m *Manager) prepare() (prepared, error) {
	config, err := m.Store.LoadWorkspaceConfig()
	if err != nil {
		return prepared{}, err
	}
	if strings.TrimSpace(config.URL) == "" {
		return prepared{}, fmt.Errorf("personal workspace is not configured")
	}
	if err := validateConfig(&config); err != nil {
		return prepared{}, err
	}
	state, err := m.Store.LoadWorkspaceState()
	if err != nil {
		return prepared{}, err
	}
	checkout, tempRoot, remoteRevision, remoteAvailable, err := m.clone(config)
	if err != nil {
		return prepared{}, err
	}
	workspaceRoot, manifestPath, manifest, found, err := inspectCheckout(checkout, config)
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		return prepared{}, err
	}
	localSkills, localPrompts, err := m.scanLocal()
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		return prepared{}, err
	}
	remoteSkills, remotePrompts, err := loadRemote(workspaceRoot, manifest)
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		return prepared{}, err
	}
	preview, err := m.buildPreview(config, state, remoteRevision, remoteAvailable, localSkills, localPrompts, remoteSkills, remotePrompts)
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		return prepared{}, err
	}
	return prepared{
		preview: preview, config: config, state: state, checkout: checkout, tempRoot: tempRoot,
		workspaceRoot: workspaceRoot, manifestPath: manifestPath, manifestFound: found,
		localSkills: localSkills, localPrompts: localPrompts, remoteSkills: remoteSkills, remotePrompts: remotePrompts,
	}, nil
}

func inspectCheckout(checkout string, config domain.WorkspaceConfig) (string, string, domain.WorkspaceManifest, bool, error) {
	workspaceRoot := filepath.Join(checkout, filepath.FromSlash(config.Root))
	if !fsx.Within(checkout, workspaceRoot) {
		return "", "", domain.WorkspaceManifest{}, false, fmt.Errorf("workspace root escapes checkout")
	}
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		return "", "", domain.WorkspaceManifest{}, false, err
	}
	if err := ensureSafeWorkspacePath(checkout, workspaceRoot, true); err != nil {
		return "", "", domain.WorkspaceManifest{}, false, err
	}
	manifestPath := filepath.Join(workspaceRoot, "skm-workspace.yaml")
	if _, statErr := os.Lstat(manifestPath); statErr == nil {
		if err := ensureSafeWorkspacePath(workspaceRoot, manifestPath, false); err != nil {
			return "", "", domain.WorkspaceManifest{}, false, err
		}
	} else if !os.IsNotExist(statErr) {
		return "", "", domain.WorkspaceManifest{}, false, statErr
	}
	manifest, found, err := loadManifest(manifestPath)
	if err != nil {
		return "", "", domain.WorkspaceManifest{}, false, err
	}
	return workspaceRoot, manifestPath, manifest, found, nil
}

func (m *Manager) scanLocal() (map[string]contentItem, map[string]contentItem, error) {
	library, err := m.Store.LoadCatalog()
	if err != nil {
		return nil, nil, err
	}
	skills := make(map[string]contentItem)
	for _, value := range library.Skills {
		if value.Location != domain.LocationLibrary || value.Source != "local" || value.ProjectRoot != "" {
			continue
		}
		document, err := skill.Validate(value.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("validate local workspace Skill %s: %w", value.ID, err)
		}
		if value.ID != "local/"+document.Name {
			return nil, nil, fmt.Errorf("local Skill %s does not match its document name", value.ID)
		}
		entry := domain.WorkspaceEntry{ID: value.ID, Path: "skills/" + document.Name, Hash: document.Hash, Tags: append([]string(nil), value.Tags...)}
		skills[value.ID] = contentItem{Entry: entry, Path: document.Path}
	}
	promptCatalog, err := m.Store.LoadPromptCatalog()
	if err != nil {
		return nil, nil, err
	}
	prompts := make(map[string]contentItem)
	for _, value := range promptCatalog.Prompts {
		if value.Source != "local" {
			continue
		}
		document, err := promptpkg.Validate(value.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("validate local workspace Prompt %s: %w", value.ID, err)
		}
		if value.ID != "local/"+document.Name {
			return nil, nil, fmt.Errorf("local Prompt %s does not match its document name", value.ID)
		}
		entry := domain.WorkspaceEntry{ID: value.ID, Path: "prompts/" + document.Name + "/PROMPT.md", Hash: document.Hash, Tags: append([]string(nil), value.Tags...)}
		prompts[value.ID] = contentItem{Entry: entry, Path: document.Path}
	}
	return skills, prompts, nil
}

func (m *Manager) buildPreview(config domain.WorkspaceConfig, state domain.WorkspaceState, revision string, remoteAvailable bool, localSkills, localPrompts, remoteSkills, remotePrompts map[string]contentItem) (Preview, error) {
	preview := Preview{
		Configured: true, Config: &config, BaseRevision: state.Revision, RemoteRevision: revision,
		Skills: len(localSkills), Prompts: len(localPrompts), LastSyncedAt: state.LastSyncedAt, RemoteAvailable: remoteAvailable,
	}
	changes, err := compareKind("skill", state.SkillBases, localSkills, remoteSkills)
	if err != nil {
		return Preview{}, err
	}
	promptChanges, err := compareKind("prompt", state.PromptBases, localPrompts, remotePrompts)
	if err != nil {
		return Preview{}, err
	}
	preview.Changes = append(changes, promptChanges...)
	stateValue, err := m.Store.LoadState()
	if err != nil {
		return Preview{}, err
	}
	enabled := make(map[string]struct{})
	for _, activation := range stateValue.Activations {
		enabled[activation.SkillID] = struct{}{}
	}
	for index := range preview.Changes {
		change := &preview.Changes[index]
		if change.Kind == "skill" && change.Action == "delete-local" {
			if _, ok := enabled[change.ID]; ok {
				change.Action = "conflict"
				change.Reason = "enabled-skill-delete"
				change.Detail = "remote deletion cannot remove an enabled Skill; disable it on this device first"
			}
		}
		switch change.Action {
		case "upload":
			preview.Uploads++
		case "download":
			preview.Downloads++
		case "delete-local", "delete-remote":
			preview.Deletes++
		}
		if change.Action == "conflict" {
			preview.Conflicts++
		}
	}
	return preview, nil
}

func compareKind(kind string, bases map[string]string, local, remote map[string]contentItem) ([]Change, error) {
	ids := make(map[string]struct{}, len(bases)+len(local)+len(remote))
	for id := range bases {
		ids[id] = struct{}{}
	}
	for id := range local {
		ids[id] = struct{}{}
	}
	for id := range remote {
		ids[id] = struct{}{}
	}
	ordered := make([]string, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	result := make([]Change, 0)
	for _, id := range ordered {
		localItem, hasLocal := local[id]
		remoteItem, hasRemote := remote[id]
		base, hasBase := bases[id]
		localValue, remoteValue := "", ""
		if hasLocal {
			localValue = fingerprint(localItem.Entry)
		}
		if hasRemote {
			remoteValue = fingerprint(remoteItem.Entry)
		}
		change := Change{Kind: kind, ID: id, Name: strings.TrimPrefix(id, "local/"), LocalHash: localValue, RemoteHash: remoteValue, BaseHash: base}
		switch {
		case !hasBase && hasLocal && !hasRemote:
			change.Action = "upload"
		case !hasBase && !hasLocal && hasRemote:
			change.Action = "download"
		case !hasBase && hasLocal && hasRemote && localValue == remoteValue:
			continue
		case !hasBase && hasLocal && hasRemote:
			change.Action, change.Reason, change.Detail = "conflict", "created-both", "item was created differently on both devices"
		case hasBase && hasLocal && hasRemote && localValue == remoteValue:
			continue
		case hasBase && hasLocal && hasRemote && localValue != base && remoteValue == base:
			change.Action = "upload"
		case hasBase && hasLocal && hasRemote && localValue == base && remoteValue != base:
			change.Action = "download"
		case hasBase && hasLocal && hasRemote:
			change.Action, change.Reason, change.Detail = "conflict", "changed-both", "item changed differently on this device and the remote workspace"
		case hasBase && !hasLocal && hasRemote && remoteValue == base:
			change.Action = "delete-remote"
		case hasBase && !hasLocal && hasRemote:
			change.Action, change.Reason, change.Detail = "conflict", "deleted-local-changed-remote", "item was deleted locally but changed remotely"
		case hasBase && hasLocal && !hasRemote && localValue == base:
			change.Action = "delete-local"
		case hasBase && hasLocal && !hasRemote:
			change.Action, change.Reason, change.Detail = "conflict", "changed-local-deleted-remote", "item changed locally but was deleted remotely"
		case hasBase && !hasLocal && !hasRemote:
			continue
		default:
			return nil, fmt.Errorf("cannot compare %s %s", kind, id)
		}
		result = append(result, change)
	}
	return result, nil
}

func (m *Manager) clone(config domain.WorkspaceConfig) (string, string, string, bool, error) {
	if _, err := exec.LookPath(m.GitPath); err != nil {
		return "", "", "", false, fmt.Errorf("git executable not found")
	}
	parent := filepath.Join(m.Store.Paths.Home, "workspace")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", "", "", false, err
	}
	tempRoot, err := os.MkdirTemp(parent, ".skm-workspace-")
	if err != nil {
		return "", "", "", false, err
	}
	checkout := filepath.Join(tempRoot, "repo")
	if err := m.git(config.URL, "init", "-b", config.Ref, "--", checkout); err != nil {
		return "", tempRoot, "", false, err
	}
	if err := m.git(config.URL, "-C", checkout, "remote", "add", "origin", config.URL); err != nil {
		return "", tempRoot, "", false, err
	}
	ref := "refs/heads/" + config.Ref
	output, err := m.gitOutput(config.URL, "ls-remote", "--heads", "--", config.URL, ref)
	if err != nil {
		return "", tempRoot, "", false, err
	}
	if strings.TrimSpace(output) == "" {
		return checkout, tempRoot, "", true, nil
	}
	if err := m.git(config.URL, "-C", checkout, "fetch", "--no-tags", "origin", ref); err != nil {
		return "", tempRoot, "", false, err
	}
	if err := m.git(config.URL, "-C", checkout, "checkout", "-B", config.Ref, "FETCH_HEAD"); err != nil {
		return "", tempRoot, "", false, err
	}
	revision, err := gitOutput(m.GitPath, checkout, "rev-parse", "HEAD")
	if err != nil {
		return "", tempRoot, "", false, err
	}
	return checkout, tempRoot, revision, true, nil
}

func (m *Manager) commitIfNeeded(value prepared) (bool, error) {
	command := exec.Command(m.GitPath, "-C", value.checkout, "diff", "--cached", "--quiet", "--exit-code")
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	err := command.Run()
	if err == nil {
		return false, nil
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
		return false, fmt.Errorf("inspect workspace changes: %w", err)
	}
	if err := m.git(value.config.URL, "-C", value.checkout,
		"-c", "user.name=SKM Workspace", "-c", "user.email=workspace@skm.local",
		"commit", "-m", "chore(sync): update SKM workspace"); err != nil {
		return false, err
	}
	return true, nil
}

func (m *Manager) writeItem(root, kind string, item contentItem) error {
	target := workspaceItemPath(root, kind, item.Entry.Path)
	if !fsx.Within(root, target) {
		return fmt.Errorf("workspace item path escapes root: %s", item.Entry.Path)
	}
	if kind == "skill" {
		return fsx.CopyDirAtomic(item.Path, target)
	}
	document, err := promptpkg.Validate(item.Path)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Dir(target)); err != nil {
		return err
	}
	return fsx.AtomicWriteFile(target, []byte(document.Content), 0o644)
}

func (m *Manager) applyLocal(root string, skills, prompts map[string]contentItem, revision string) error {
	beforeSkills, err := m.Store.LoadCatalog()
	if err != nil {
		return err
	}
	beforePrompts, err := m.Store.LoadPromptCatalog()
	if err != nil {
		return err
	}
	now := m.Now().UTC()
	existingSkills := make(map[string]domain.Skill)
	for _, value := range beforeSkills.Skills {
		existingSkills[value.ID] = value
	}
	existingPrompts := make(map[string]domain.Prompt)
	for _, value := range beforePrompts.Prompts {
		existingPrompts[value.ID] = value
	}

	updatedSkills := beforeSkills
	keptSkills := updatedSkills.Skills[:0]
	for _, value := range updatedSkills.Skills {
		if value.Source == "local" && value.ProjectRoot == "" {
			if _, keep := skills[value.ID]; !keep {
				continue
			}
		}
		keptSkills = append(keptSkills, value)
	}
	updatedSkills.Skills = keptSkills
	for id, item := range skills {
		document, err := skill.Validate(workspaceItemPath(root, "skill", item.Entry.Path))
		if err != nil {
			return fmt.Errorf("validate merged Skill %s: %w", id, err)
		}
		value, err := catalog.New(m.Store).Snapshot(document, "local", revision, item.Entry.Tags)
		if err != nil {
			return err
		}
		value.ID = id
		if existing := existingSkills[id]; !existing.AddedAt.IsZero() {
			value.AddedAt = existing.AddedAt
		}
		updatedSkills.Skills = upsertSkill(updatedSkills.Skills, value)
	}

	updatedPrompts := beforePrompts
	keptPrompts := updatedPrompts.Prompts[:0]
	for _, value := range updatedPrompts.Prompts {
		if value.Source == "local" {
			if _, keep := prompts[value.ID]; !keep {
				continue
			}
		}
		keptPrompts = append(keptPrompts, value)
	}
	updatedPrompts.Prompts = keptPrompts
	for id, item := range prompts {
		document, err := promptpkg.Validate(workspaceItemPath(root, "prompt", item.Entry.Path))
		if err != nil {
			return fmt.Errorf("validate merged Prompt %s: %w", id, err)
		}
		normalizedTags, err := tags.Normalize(item.Entry.Tags, document.Tags)
		if err != nil {
			return err
		}
		destination := m.Store.PromptObjectPath(document.Hash, document.Name)
		if err := fsx.AtomicWriteFile(filepath.Join(destination, "PROMPT.md"), []byte(document.Content), 0o644); err != nil {
			return err
		}
		addedAt := now
		if existing := existingPrompts[id]; !existing.AddedAt.IsZero() {
			addedAt = existing.AddedAt
		}
		value := domain.Prompt{
			ID: id, Name: document.Name, Description: document.Description, Tags: normalizedTags, Source: "local",
			Hash: document.Hash, Path: destination, Variables: document.Variables, AddedAt: addedAt, UpdatedAt: now,
		}
		updatedPrompts.Prompts = upsertPrompt(updatedPrompts.Prompts, value)
	}

	if err := m.Store.SaveCatalog(updatedSkills); err != nil {
		return err
	}
	if err := m.Store.SavePromptCatalog(updatedPrompts); err != nil {
		rollbackErr := m.Store.SaveCatalog(beforeSkills)
		return errors.Join(err, rollbackErr)
	}
	state := domain.WorkspaceState{
		Version: domain.WorkspaceSchemaVersion, Revision: revision,
		SkillBases: fingerprints(skills), PromptBases: fingerprints(prompts), LastSyncedAt: now,
	}
	if err := m.Store.SaveWorkspaceState(state); err != nil {
		return errors.Join(err, m.Store.SaveCatalog(beforeSkills), m.Store.SavePromptCatalog(beforePrompts))
	}
	return nil
}

func loadManifest(path string) (domain.WorkspaceManifest, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return domain.WorkspaceManifest{Version: domain.WorkspaceSchemaVersion}, false, nil
	}
	if err != nil {
		return domain.WorkspaceManifest{}, false, err
	}
	var manifest domain.WorkspaceManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return domain.WorkspaceManifest{}, false, fmt.Errorf("read workspace manifest: %w", err)
	}
	if manifest.Version != domain.WorkspaceSchemaVersion {
		return domain.WorkspaceManifest{}, false, fmt.Errorf("unsupported workspace schema version %d", manifest.Version)
	}
	return manifest, true, nil
}

func loadRemote(root string, manifest domain.WorkspaceManifest) (map[string]contentItem, map[string]contentItem, error) {
	skills := make(map[string]contentItem)
	paths := make(map[string]string)
	for _, entry := range manifest.Skills {
		if err := validateEntry(root, "skill", entry, paths); err != nil {
			return nil, nil, err
		}
		path := workspaceItemPath(root, "skill", entry.Path)
		if err := ensureSafeWorkspacePath(root, path, true); err != nil {
			return nil, nil, fmt.Errorf("unsafe workspace Skill %s: %w", entry.ID, err)
		}
		document, err := skill.Validate(path)
		if err != nil {
			return nil, nil, fmt.Errorf("validate workspace Skill %s: %w", entry.ID, err)
		}
		if entry.ID != "local/"+document.Name || entry.Hash != document.Hash {
			return nil, nil, fmt.Errorf("workspace Skill %s manifest does not match its content", entry.ID)
		}
		entry.Tags, err = tags.Normalize(entry.Tags, nil)
		if err != nil {
			return nil, nil, err
		}
		skills[entry.ID] = contentItem{Entry: entry, Path: path}
	}
	prompts := make(map[string]contentItem)
	for _, entry := range manifest.Prompts {
		if err := validateEntry(root, "prompt", entry, paths); err != nil {
			return nil, nil, err
		}
		path := workspaceItemPath(root, "prompt", entry.Path)
		if err := ensureSafeWorkspacePath(root, path, false); err != nil {
			return nil, nil, fmt.Errorf("unsafe workspace Prompt %s: %w", entry.ID, err)
		}
		document, err := promptpkg.Validate(path)
		if err != nil {
			return nil, nil, fmt.Errorf("validate workspace Prompt %s: %w", entry.ID, err)
		}
		if entry.ID != "local/"+document.Name || entry.Hash != document.Hash {
			return nil, nil, fmt.Errorf("workspace Prompt %s manifest does not match its content", entry.ID)
		}
		entry.Tags, err = tags.Normalize(entry.Tags, document.Tags)
		if err != nil {
			return nil, nil, err
		}
		prompts[entry.ID] = contentItem{Entry: entry, Path: path}
	}
	return skills, prompts, nil
}

func validateEntry(root, kind string, entry domain.WorkspaceEntry, paths map[string]string) error {
	if !strings.HasPrefix(entry.ID, "local/") || strings.Count(entry.ID, "/") != 1 {
		return fmt.Errorf("invalid workspace %s ID %q", kind, entry.ID)
	}
	if entry.Hash == "" {
		return fmt.Errorf("workspace %s %s has no hash", kind, entry.ID)
	}
	if filepath.IsAbs(entry.Path) || entry.Path == "" {
		return fmt.Errorf("workspace %s %s has invalid path", kind, entry.ID)
	}
	cleanPath := filepath.ToSlash(filepath.Clean(filepath.FromSlash(entry.Path)))
	name := strings.TrimPrefix(entry.ID, "local/")
	expectedPath := "skills/" + name
	if kind == "prompt" {
		expectedPath = "prompts/" + name + "/PROMPT.md"
	}
	if cleanPath != expectedPath || filepath.ToSlash(entry.Path) != expectedPath {
		return fmt.Errorf("workspace %s %s must use canonical path %s", kind, entry.ID, expectedPath)
	}
	path := workspaceItemPath(root, kind, entry.Path)
	if !fsx.Within(root, path) {
		return fmt.Errorf("workspace %s path escapes root: %s", kind, entry.Path)
	}
	key := filepath.Clean(path)
	if previous := paths[key]; previous != "" {
		return fmt.Errorf("workspace path %s is shared by %s and %s", entry.Path, previous, entry.ID)
	}
	paths[key] = entry.ID
	return nil
}

func validateConfig(value *domain.WorkspaceConfig) error {
	value.URL = strings.TrimSpace(value.URL)
	value.Ref = strings.TrimSpace(value.Ref)
	value.Root = filepath.ToSlash(filepath.Clean(strings.TrimSpace(value.Root)))
	if value.Root == "." {
		value.Root = ""
	}
	if value.URL == "" {
		return fmt.Errorf("workspace Git URL is required")
	}
	if parsed, err := url.Parse(value.URL); err == nil && parsed.User != nil {
		return fmt.Errorf("workspace URL must not contain credentials; use Git credential storage or SSH")
	}
	if value.Ref == "" {
		value.Ref = "main"
	}
	if strings.ContainsAny(value.Ref, "\r\n") || strings.HasPrefix(value.Ref, "-") {
		return fmt.Errorf("invalid workspace branch %q", value.Ref)
	}
	if value.Root != "" && (filepath.IsAbs(value.Root) || value.Root == ".." || strings.HasPrefix(value.Root, "../")) {
		return fmt.Errorf("workspace root must be repository-relative")
	}
	return nil
}

func ensureSafeWorkspacePath(root, path string, wantDirectory bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symbolic link is not allowed: %s", path)
	}
	if wantDirectory && !info.IsDir() {
		return fmt.Errorf("expected directory: %s", path)
	}
	if !wantDirectory && !info.Mode().IsRegular() {
		return fmt.Errorf("expected regular file: %s", path)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	if !fsx.Within(resolvedRoot, resolvedPath) {
		return fmt.Errorf("path escapes workspace root: %s", path)
	}
	return nil
}

func (m *Manager) git(secret string, args ...string) error {
	_, err := m.gitOutput(secret, args...)
	return err
}

func (m *Manager) gitOutput(secret string, args ...string) (string, error) {
	command := exec.Command(m.GitPath, args...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := command.CombinedOutput()
	if err == nil {
		return string(output), nil
	}
	message := strings.TrimSpace(string(output))
	if secret != "" {
		message = strings.ReplaceAll(message, secret, "<redacted-url>")
	}
	if message == "" {
		message = err.Error()
	}
	return "", fmt.Errorf("git command failed: %s", message)
}

func gitOutput(gitPath, directory string, args ...string) (string, error) {
	command := exec.Command(gitPath, append([]string{"-C", directory}, args...)...)
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func workspaceItemPath(root, kind, relative string) string {
	path := filepath.Join(root, filepath.FromSlash(relative))
	if kind == "skill" && filepath.Base(path) == "SKILL.md" {
		path = filepath.Dir(path)
	}
	return path
}

func removeWorkspaceItem(root, kind, relative string) error {
	path := workspaceItemPath(root, kind, relative)
	if !fsx.Within(root, path) {
		return fmt.Errorf("workspace item path escapes root: %s", relative)
	}
	if kind == "prompt" {
		path = filepath.Dir(path)
	}
	return os.RemoveAll(path)
}

func fingerprint(entry domain.WorkspaceEntry) string {
	tagsCopy := append([]string(nil), entry.Tags...)
	sort.Strings(tagsCopy)
	digest := sha256.Sum256([]byte(entry.ID + "\x00" + entry.Hash + "\x00" + strings.Join(tagsCopy, "\x00")))
	return hex.EncodeToString(digest[:])
}

func fingerprints(items map[string]contentItem) map[string]string {
	result := make(map[string]string, len(items))
	for id, item := range items {
		result[id] = fingerprint(item.Entry)
	}
	return result
}

func cloneItems(values map[string]contentItem) map[string]contentItem {
	result := make(map[string]contentItem, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func entriesFromItems(values map[string]contentItem) []domain.WorkspaceEntry {
	result := make([]domain.WorkspaceEntry, 0, len(values))
	for _, value := range values {
		result = append(result, value.Entry)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func upsertSkill(values []domain.Skill, value domain.Skill) []domain.Skill {
	for index := range values {
		if values[index].ID == value.ID {
			values[index] = value
			return values
		}
	}
	return append(values, value)
}

func upsertPrompt(values []domain.Prompt, value domain.Prompt) []domain.Prompt {
	for index := range values {
		if values[index].ID == value.ID {
			values[index] = value
			return values
		}
	}
	return append(values, value)
}
