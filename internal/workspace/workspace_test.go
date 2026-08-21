package workspace

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	promptpkg "github.com/zzzzzyijie/skm/internal/prompt"
	skillpkg "github.com/zzzzzyijie/skm/internal/skill"
	"github.com/zzzzzyijie/skm/internal/store"
)

func TestWorkspaceSyncsProjectMigratedSkill(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	remote := filepath.Join(t.TempDir(), "workspace.git")
	runGit(t, "", "init", "--bare", "-b", "main", remote)

	deviceA := workspaceStore(t)
	projectRoot := filepath.Join(t.TempDir(), "proj")
	projectSkill := filepath.Join(projectRoot, ".claude", "skills", "commit")
	if err := os.MkdirAll(projectSkill, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: commit\ndescription: Project migrated Skill\n---\n\nCommit workflow.\n"
	if err := os.WriteFile(filepath.Join(projectSkill, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	document, err := skillpkg.Validate(projectSkill)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.New(deviceA).ImportProject(document, projectRoot, domain.AgentClaude, domain.ModeCopy, []string{"git"}); err != nil {
		t.Fatal(err)
	}

	managerA := New(deviceA)
	if _, err := managerA.Configure(domain.WorkspaceConfig{URL: remote}); err != nil {
		t.Fatal(err)
	}
	preview, err := managerA.Preview()
	if err != nil {
		t.Fatal(err)
	}
	if preview.Uploads != 1 || preview.Conflicts != 0 {
		t.Fatalf("project-migrated Skill preview = %#v", preview)
	}
	if _, err := managerA.Apply(); err != nil {
		t.Fatal(err)
	}

	deviceB := workspaceStore(t)
	managerB := New(deviceB)
	if _, err := managerB.Configure(domain.WorkspaceConfig{URL: remote}); err != nil {
		t.Fatal(err)
	}
	if _, err := managerB.Apply(); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.New(deviceB).ResolveLibrary("local/commit"); err != nil {
		t.Fatalf("device B restored project-migrated Skill: %v", err)
	}
}

func TestWorkspaceSyncAcrossDevicesAndDetectsConflict(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	remote := filepath.Join(t.TempDir(), "workspace.git")
	runGit(t, "", "init", "--bare", "-b", "main", remote)

	deviceA := workspaceStore(t)
	addSkill(t, deviceA, "review", "version one")
	addPrompt(t, deviceA, "summary", "Summarize version one.")
	managerA := New(deviceA)
	if _, err := managerA.Configure(domain.WorkspaceConfig{URL: remote, Ref: "main"}); err != nil {
		t.Fatal(err)
	}
	previewA, err := managerA.Preview()
	if err != nil {
		t.Fatal(err)
	}
	if previewA.Uploads != 2 || previewA.Downloads != 0 || previewA.Conflicts != 0 {
		t.Fatalf("device A initial preview = %#v", previewA)
	}
	first, err := managerA.Apply()
	if err != nil || !first.Applied || !first.Committed || first.Revision == "" {
		t.Fatalf("device A initial sync = %#v, err=%v", first, err)
	}

	deviceB := workspaceStore(t)
	managerB := New(deviceB)
	if _, err := managerB.Configure(domain.WorkspaceConfig{URL: remote, Ref: "main"}); err != nil {
		t.Fatal(err)
	}
	previewB, err := managerB.Preview()
	if err != nil {
		t.Fatal(err)
	}
	if previewB.Downloads != 2 || previewB.Uploads != 0 || previewB.Conflicts != 0 {
		t.Fatalf("device B restore preview = %#v", previewB)
	}
	if _, err := managerB.Apply(); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.New(deviceB).ResolveLibrary("local/review"); err != nil {
		t.Fatalf("device B restored Skill: %v", err)
	}
	if _, err := promptpkg.New(deviceB).Resolve("local/summary"); err != nil {
		t.Fatalf("device B restored Prompt: %v", err)
	}

	promptB, documentB, err := promptpkg.New(deviceB).Read("local/summary")
	if err != nil {
		t.Fatal(err)
	}
	updatedContent := strings.Replace(documentB.Content, "version one", "version two", 1)
	if _, err := promptpkg.New(deviceB).Update(promptB.ID, updatedContent, promptB.Hash, promptB.Tags); err != nil {
		t.Fatal(err)
	}
	if _, err := managerB.Apply(); err != nil {
		t.Fatal(err)
	}
	if _, err := managerA.Apply(); err != nil {
		t.Fatal(err)
	}
	_, restoredPrompt, err := promptpkg.New(deviceA).Read("local/summary")
	if err != nil || !strings.Contains(restoredPrompt.Content, "version two") {
		t.Fatalf("device A updated Prompt = %#v, err=%v", restoredPrompt, err)
	}

	addSkill(t, deviceA, "review", "device A edit")
	addSkill(t, deviceB, "review", "device B edit")
	if _, err := managerA.Apply(); err != nil {
		t.Fatal(err)
	}
	conflict, err := managerB.Preview()
	if err != nil {
		t.Fatal(err)
	}
	if conflict.Conflicts != 1 || len(conflict.Changes) != 1 || conflict.Changes[0].Action != "conflict" {
		t.Fatalf("device B conflict preview = %#v", conflict)
	}
	if _, err := managerB.Apply(); err == nil || !strings.Contains(err.Error(), "workspace sync has conflicts") {
		t.Fatalf("conflicting sync should stop, got %v", err)
	}
	if _, err := managerB.ApplyResolved(map[string]string{"skill:local/review": "remote"}); err != nil {
		t.Fatalf("resolve conflict with remote version: %v", err)
	}
	resolvedSkill, err := catalog.New(deviceB).ResolveLibrary("local/review")
	if err != nil {
		t.Fatal(err)
	}
	resolvedDocument, err := os.ReadFile(filepath.Join(resolvedSkill.Path, "SKILL.md"))
	if err != nil || !strings.Contains(string(resolvedDocument), "device A edit") {
		t.Fatalf("resolved Skill content = %q, err=%v", resolvedDocument, err)
	}
}

func TestWorkspaceSyncPropagatesPromptDeletion(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	remote := filepath.Join(t.TempDir(), "workspace.git")
	runGit(t, "", "init", "--bare", "-b", "main", remote)
	deviceA := workspaceStore(t)
	addPrompt(t, deviceA, "summary", "Initial prompt.")
	managerA := New(deviceA)
	if _, err := managerA.Configure(domain.WorkspaceConfig{URL: remote}); err != nil {
		t.Fatal(err)
	}
	if _, err := managerA.Apply(); err != nil {
		t.Fatal(err)
	}
	deviceB := workspaceStore(t)
	managerB := New(deviceB)
	if _, err := managerB.Configure(domain.WorkspaceConfig{URL: remote}); err != nil {
		t.Fatal(err)
	}
	if _, err := managerB.Apply(); err != nil {
		t.Fatal(err)
	}
	if _, err := promptpkg.New(deviceA).Remove("local/summary"); err != nil {
		t.Fatal(err)
	}
	preview, err := managerA.Preview()
	if err != nil || preview.Uploads != 0 || preview.Downloads != 0 || preview.Deletes != 1 || preview.Changes[0].Action != "delete-remote" {
		t.Fatalf("deletion publish preview = %#v, err=%v", preview, err)
	}
	if _, err := managerA.Apply(); err != nil {
		t.Fatal(err)
	}
	preview, err = managerB.Preview()
	if err != nil || preview.Changes[0].Action != "delete-local" {
		t.Fatalf("deletion restore preview = %#v, err=%v", preview, err)
	}
	if _, err := managerB.Apply(); err != nil {
		t.Fatal(err)
	}
	if _, err := promptpkg.New(deviceB).Resolve("local/summary"); err == nil {
		t.Fatal("remote Prompt deletion was not applied on device B")
	}
}

func TestWorkspaceDoesNotDeleteEnabledSkill(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	remote := filepath.Join(t.TempDir(), "workspace.git")
	runGit(t, "", "init", "--bare", "-b", "main", remote)
	deviceA := workspaceStore(t)
	addSkill(t, deviceA, "review", "shared")
	managerA := New(deviceA)
	if _, err := managerA.Configure(domain.WorkspaceConfig{URL: remote}); err != nil {
		t.Fatal(err)
	}
	if _, err := managerA.Apply(); err != nil {
		t.Fatal(err)
	}
	deviceB := workspaceStore(t)
	managerB := New(deviceB)
	if _, err := managerB.Configure(domain.WorkspaceConfig{URL: remote}); err != nil {
		t.Fatal(err)
	}
	if _, err := managerB.Apply(); err != nil {
		t.Fatal(err)
	}
	if err := deviceB.SaveState(domain.State{Activations: []domain.Activation{{SkillID: "local/review", Name: "review"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.New(deviceA).Remove("local/review"); err != nil {
		t.Fatal(err)
	}
	if _, err := managerA.Apply(); err != nil {
		t.Fatal(err)
	}
	preview, err := managerB.Preview()
	if err != nil {
		t.Fatal(err)
	}
	if preview.Conflicts != 1 || preview.Changes[0].Reason != "enabled-skill-delete" {
		t.Fatalf("enabled Skill deletion preview = %#v", preview)
	}
	if _, err := managerB.ApplyResolved(map[string]string{"skill:local/review": "remote"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("remote deletion of enabled Skill should remain blocked, got %v", err)
	}
	if _, err := managerB.ApplyResolved(map[string]string{"skill:local/review": "local"}); err != nil {
		t.Fatalf("keeping enabled local Skill should publish it again: %v", err)
	}
}

func TestWorkspaceRejectsUnsafeConfigurationAndManifestPaths(t *testing.T) {
	for _, test := range []struct {
		name   string
		config domain.WorkspaceConfig
	}{
		{name: "credential URL", config: domain.WorkspaceConfig{URL: "https://token@example.com/workspace.git"}},
		{name: "escaping root", config: domain.WorkspaceConfig{URL: "/tmp/workspace.git", Root: "../outside"}},
		{name: "option-like branch", config: domain.WorkspaceConfig{URL: "/tmp/workspace.git", Ref: "-danger"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateConfig(&test.config); err == nil {
				t.Fatalf("unsafe config should fail: %#v", test.config)
			}
		})
	}
	root := t.TempDir()
	for _, test := range []struct {
		kind  string
		entry domain.WorkspaceEntry
	}{
		{kind: "skill", entry: domain.WorkspaceEntry{ID: "local/review", Path: "other/review", Hash: "hash"}},
		{kind: "prompt", entry: domain.WorkspaceEntry{ID: "local/summary", Path: "prompts/summary", Hash: "hash"}},
		{kind: "prompt", entry: domain.WorkspaceEntry{ID: "local/summary", Path: "../PROMPT.md", Hash: "hash"}},
		{kind: "skill", entry: domain.WorkspaceEntry{ID: "local/review", Path: "skills/other", Hash: "hash"}},
		{kind: "prompt", entry: domain.WorkspaceEntry{ID: "local/summary", Path: "prompts/group/summary/PROMPT.md", Hash: "hash"}},
	} {
		if err := validateEntry(root, test.kind, test.entry, map[string]string{}); err == nil {
			t.Fatalf("unsafe %s manifest path should fail: %#v", test.kind, test.entry)
		}
	}
}

func TestWorkspaceRejectsSymbolicLinks(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "SKILL.md")
	if err := os.WriteFile(target, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "SKILL.md")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := ensureSafeWorkspacePath(root, link, false); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("workspace symlink should fail, got %v", err)
	}
}

func workspaceStore(t *testing.T) *store.Store {
	t.Helper()
	root := t.TempDir()
	storage, err := store.New(store.Paths{
		Home: filepath.Join(root, "home", ".skm"), UserHome: filepath.Join(root, "home"), ProjectRoot: filepath.Join(root, "project"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Ensure(); err != nil {
		t.Fatal(err)
	}
	return storage
}

func addSkill(t *testing.T, storage *store.Store, name, body string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: Workspace test Skill\n---\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.New(storage).AddLocal(root, "local", []string{"test"}); err != nil {
		t.Fatal(err)
	}
}

func addPrompt(t *testing.T, storage *store.Store, name, body string) {
	t.Helper()
	content := "---\nname: " + name + "\ndescription: Workspace test Prompt\ntags: [test]\n---\n" + body + "\n"
	if _, err := promptpkg.New(storage).Create(content, "local", []string{"test"}); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	if directory != "" {
		command.Dir = directory
	}
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}
