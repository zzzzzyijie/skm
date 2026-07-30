package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/store"
)

func TestCLIAddTagEnableDisableAndPlan(t *testing.T) {
	root, userHome, project, skmHome := cliPaths(t)
	skillPath := filepath.Join(root, "review")
	writeCLISkill(t, skillPath, "review")
	runCLI(t, "--home", skmHome, "--project", project, "add", skillPath, "--tag", "backend")
	out := runCLI(t, "--home", skmHome, "--project", project, "--json", "list", "--tag", "backend")
	var listEnvelope struct {
		Success bool `json:"success"`
		Data    []struct {
			ID   string   `json:"id"`
			Tags []string `json:"tags"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &listEnvelope); err != nil {
		t.Fatal(err)
	}
	if !listEnvelope.Success || len(listEnvelope.Data) != 1 || listEnvelope.Data[0].ID != "local/review" {
		t.Fatalf("unexpected list response: %s", out)
	}

	runCLI(t, "--home", skmHome, "--project", project, "enable", "--tag", "backend", "--agent", "claude,codex")
	for _, target := range []string{
		filepath.Join(userHome, ".claude", "skills", "review"),
		filepath.Join(userHome, ".agents", "skills", "review"),
	} {
		if info, err := os.Lstat(target); err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("target %s is not a symlink: info=%v err=%v", target, info, err)
		}
	}
	out = runCLI(t, "--home", skmHome, "--project", project, "--json", "plan")
	assertPlanStatuses(t, out, 2, "unchanged")

	runCLI(t, "--home", skmHome, "--project", project, "disable", "review")
	if _, err := os.Lstat(filepath.Join(userHome, ".agents", "skills", "review")); !os.IsNotExist(err) {
		t.Fatalf("disabled target still exists: %v", err)
	}
	out = runCLI(t, "--home", skmHome, "--project", project, "list")
	if !bytes.Contains(out, []byte("local/review")) {
		t.Fatalf("disable removed Library Skill:\n%s", out)
	}
}

func TestCLIProjectVendorRetainsPersonalLibrary(t *testing.T) {
	root, _, project, skmHome := cliPaths(t)
	skillPath := filepath.Join(root, "release")
	writeCLISkill(t, skillPath, "release")
	runCLI(t, "--home", skmHome, "--project", project, "add", skillPath, "--tag", "release")
	out := runCLI(t, "--home", skmHome, "--project", project, "project", "vendor", "local/release", "--agent", "claude,codex")
	if !bytes.Contains(out, []byte("personal Library copy retained")) {
		t.Fatalf("vendor output:\n%s", out)
	}
	for _, target := range []string{
		filepath.Join(project, ".claude", "skills", "release"),
		filepath.Join(project, ".agents", "skills", "release"),
		filepath.Join(project, ".skm", "skills", "release", "SKILL.md"),
	} {
		if _, err := os.Lstat(target); err != nil {
			t.Fatalf("vendored target %s missing: %v", target, err)
		}
	}
	library := runCLI(t, "--home", skmHome, "--project", project, "list")
	if !bytes.Contains(library, []byte("local/release")) {
		t.Fatalf("personal copy missing:\n%s", library)
	}
	manifest, err := os.ReadFile(filepath.Join(project, ".skm", "project.yaml"))
	if err != nil || !bytes.Contains(manifest, []byte("forkedFrom: local/release")) {
		t.Fatalf("vendor manifest: %v\n%s", err, manifest)
	}
}

func TestCLIProjectRequireCanBeSatisfiedByUserActivation(t *testing.T) {
	root, userHome, project, skmHome := cliPaths(t)
	skillPath := filepath.Join(root, "review")
	writeCLISkill(t, skillPath, "review")
	runCLI(t, "--home", skmHome, "--project", project, "add", skillPath, "--source", "team", "--tag", "review")
	storage := openTestStore(t, skmHome, userHome, project)
	library, err := storage.LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	library.Skills[0].Revision = "abc123"
	library.Skills[0].SourcePath = "skills/review"
	if err := storage.SaveCatalog(library); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveSources(domain.Sources{Sources: []domain.Source{{Name: "team", URL: "git@example.invalid:team.git", Ref: "main", Revision: "abc123"}}}); err != nil {
		t.Fatal(err)
	}
	runCLI(t, "--home", skmHome, "--project", project, "enable", "team/review", "--agent", "codex")
	out := runCLI(t, "--home", skmHome, "--project", project, "project", "require", "team/review", "--agent", "codex")
	if !bytes.Contains(out, []byte("Satisfied by user activation: team/review:codex")) {
		t.Fatalf("require was not satisfied by user:\n%s", out)
	}
	if _, err := os.Lstat(filepath.Join(project, ".agents", "skills", "review")); !os.IsNotExist(err) {
		t.Fatalf("redundant project link was created: %v", err)
	}
	manifest, err := os.ReadFile(filepath.Join(project, ".skm", "project.yaml"))
	if err != nil || !bytes.Contains(manifest, []byte("team/review")) || !bytes.Contains(manifest, []byte("revision: abc123")) {
		t.Fatalf("project requirement: %v\n%s", err, manifest)
	}
}

func TestCLIUsesGeneralTagByDefault(t *testing.T) {
	root, _, project, skmHome := cliPaths(t)
	skillPath := filepath.Join(root, "default-skill")
	writeCLISkill(t, skillPath, "default-skill")
	runCLI(t, "--home", skmHome, "--project", project, "add", skillPath)
	out := runCLI(t, "--home", skmHome, "--project", project, "list", "--tag", "general")
	if !bytes.Contains(out, []byte("local/default-skill")) {
		t.Fatalf("default tag query did not return Skill:\n%s", out)
	}
}

func TestCLIInitWithProjectWritesProjectState(t *testing.T) {
	_, _, project, skmHome := cliPaths(t)
	runCLI(t, "--home", skmHome, "--project", project, "init", "--with-project")
	for _, path := range []string{
		filepath.Join(project, ".skm", "project.yaml"),
		filepath.Join(project, ".skm", "lock.yaml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("project state %s missing: %v", path, err)
		}
	}
}

func TestCLIVersionUsesInjectedReleaseVersion(t *testing.T) {
	previous := Version
	Version = "v1.2.3"
	t.Cleanup(func() { Version = previous })

	out := runCLI(t, "version")
	if string(out) != "1.2.3\n" {
		t.Fatalf("version output = %q", out)
	}
}

func TestCLIEnableRejectsSameNameBeforeChangingState(t *testing.T) {
	root, userHome, project, skmHome := cliPaths(t)
	firstPath := filepath.Join(root, "first")
	secondPath := filepath.Join(root, "second")
	writeCLISkill(t, firstPath, "review")
	writeCLISkill(t, secondPath, "review")
	runCLI(t, "--home", skmHome, "--project", project, "add", firstPath, "--source", "team-a")
	runCLI(t, "--home", skmHome, "--project", project, "add", secondPath, "--source", "team-b")
	runCLI(t, "--home", skmHome, "--project", project, "enable", "team-a/review", "--agent", "codex")

	target := filepath.Join(userHome, ".agents", "skills", "review")
	beforeLink, err := os.Readlink(target)
	if err != nil {
		t.Fatal(err)
	}
	stderr := runCLIFailure(t, "--home", skmHome, "--project", project, "enable", "team-b/review", "--agent", "codex")
	if !bytes.Contains(stderr, []byte("multiple Skills target")) {
		t.Fatalf("unexpected conflict error: %s", stderr)
	}
	afterLink, err := os.Readlink(target)
	if err != nil {
		t.Fatal(err)
	}
	if afterLink != beforeLink {
		t.Fatalf("conflicting enable changed target: before=%s after=%s", beforeLink, afterLink)
	}
	storage := openTestStore(t, skmHome, userHome, project)
	state, err := storage.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Activations) != 1 || state.Activations[0].SkillID != "team-a/review" {
		t.Fatalf("conflicting Activation was saved: %#v", state.Activations)
	}
}

func assertPlanStatuses(t *testing.T, raw []byte, count int, status string) {
	t.Helper()
	var envelope struct {
		Data struct {
			Operations []struct {
				Status string `json:"status"`
			} `json:"operations"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data.Operations) != count {
		t.Fatalf("unexpected plan: %s", raw)
	}
	for _, operation := range envelope.Data.Operations {
		if operation.Status != status {
			t.Fatalf("plan status is not %s: %s", status, raw)
		}
	}
}

func cliPaths(t *testing.T) (root, userHome, project, skmHome string) {
	t.Helper()
	root = t.TempDir()
	userHome = filepath.Join(root, "user")
	project = filepath.Join(root, "project")
	skmHome = filepath.Join(userHome, ".skm")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", userHome)
	return
}

func openTestStore(t *testing.T, skmHome, userHome, project string) *store.Store {
	t.Helper()
	storage, err := store.New(store.Paths{Home: skmHome, UserHome: userHome, ProjectRoot: project})
	if err != nil {
		t.Fatal(err)
	}
	return storage
}

func runCLI(t *testing.T, args ...string) []byte {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := Execute(args, &stdout, &stderr); code != 0 {
		t.Fatalf("skm %v failed (%d): %s", args, code, stderr.String())
	}
	return stdout.Bytes()
}

func runCLIFailure(t *testing.T, args ...string) []byte {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := Execute(args, &stdout, &stderr); code == 0 {
		t.Fatalf("skm %v unexpectedly succeeded: %s", args, stdout.String())
	}
	return stderr.Bytes()
}

func writeCLISkill(t *testing.T, path, name string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: CLI integration Skill\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
