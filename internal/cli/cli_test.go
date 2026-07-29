package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCLIAddListLinkAndPlan(t *testing.T) {
	root := t.TempDir()
	userHome := filepath.Join(root, "user")
	project := filepath.Join(root, "project")
	skmHome := filepath.Join(userHome, ".skm")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", userHome)
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

	runCLI(t, "--home", skmHome, "--project", project, "link", "review", "--scope", "project", "--agent", "claude,codex")
	for _, target := range []string{
		filepath.Join(project, ".claude", "skills", "review"),
		filepath.Join(project, ".agents", "skills", "review"),
	} {
		if info, err := os.Lstat(target); err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("target %s is not a symlink: info=%v err=%v", target, info, err)
		}
	}
	manifest, err := os.ReadFile(filepath.Join(project, ".skm", "project.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(manifest, []byte("dependencies:")) || !bytes.Contains(manifest, []byte("local/review")) {
		t.Fatalf("project dependency was not recorded:\n%s", manifest)
	}
	lockFile, err := os.ReadFile(filepath.Join(project, ".skm", "lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(lockFile, []byte("local/review")) || !bytes.Contains(lockFile, []byte("hash:")) {
		t.Fatalf("project lock was not recorded:\n%s", lockFile)
	}

	out = runCLI(t, "--home", skmHome, "--project", project, "--json", "plan")
	var planEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Operations []struct {
				Status string `json:"status"`
			} `json:"operations"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &planEnvelope); err != nil {
		t.Fatal(err)
	}
	if len(planEnvelope.Data.Operations) != 2 {
		t.Fatalf("unexpected plan: %s", out)
	}
	for _, operation := range planEnvelope.Data.Operations {
		if operation.Status != "unchanged" {
			t.Fatalf("plan is not idempotent: %s", out)
		}
	}
}

func TestCLIUsesGeneralTagByDefault(t *testing.T) {
	root := t.TempDir()
	userHome := filepath.Join(root, "user")
	t.Setenv("HOME", userHome)
	skillPath := filepath.Join(root, "default-skill")
	writeCLISkill(t, skillPath, "default-skill")
	runCLI(t, "--home", filepath.Join(userHome, ".skm"), "--project", root, "add", skillPath)
	out := runCLI(t, "--home", filepath.Join(userHome, ".skm"), "--project", root, "list", "--tag", "general")
	if !bytes.Contains(out, []byte("local/default-skill")) {
		t.Fatalf("default tag query did not return Skill:\n%s", out)
	}
}

func runCLI(t *testing.T, args ...string) []byte {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := Execute(args, &stdout, &stderr); code != 0 {
		t.Fatalf("skm %v failed (%d): %s", args, code, stderr.String())
	}
	return stdout.Bytes()
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
