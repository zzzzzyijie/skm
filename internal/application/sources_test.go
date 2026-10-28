package application

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
)

// Source failures must remain isolated while successful updates refresh deployments.
func TestSyncSourcesIsolatesFailuresAndRefreshesDeployments(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	service, root := newTestService(t)
	if _, err := service.SyncSources(); err == nil {
		t.Fatal("expected an error without configured sources")
	}
	gitRun := func(repository string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repository
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=skm-test", "GIT_AUTHOR_EMAIL=skm@example.invalid", "GIT_COMMITTER_NAME=skm-test", "GIT_COMMITTER_EMAIL=skm@example.invalid")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
		}
	}
	imported := make(map[string]domain.Skill)
	for _, name := range []string{"bad", "good"} {
		repository := filepath.Join(root, name)
		skillPath := writeTestSkill(t, repository, name+"-skill", "Initial repository Skill")
		gitRun(repository, "init", "-b", "main")
		gitRun(repository, "add", ".")
		gitRun(repository, "commit", "-m", "initial")
		created, err := service.AddSource(AddSourceInput{Input: repository, Name: name})
		if err != nil {
			t.Fatal(err)
		}
		if len(created.Skills) != 1 {
			t.Fatalf("imported %s skills = %+v", name, created.Skills)
		}
		imported[name] = created.Skills[0]
		if name == "good" {
			content := "---\nname: good-skill\ndescription: Updated repository Skill\n---\n\nVersion two.\n"
			if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			gitRun(repository, "add", ".")
			gitRun(repository, "commit", "-m", "update")
		}
	}
	oldGood := imported["good"]
	if _, err := service.Enable(ActivationInput{Skills: []string{oldGood.ID}, Agents: []string{"codex"}, Mode: "symlink"}); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(service.Store.Paths.UserHome, ".codex", "skills", oldGood.Name)
	assertDeployment := func(skill domain.Skill) {
		t.Helper()
		actual, err := filepath.EvalSymlinks(target)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := filepath.EvalSymlinks(skill.Path)
		if err != nil || actual != expected {
			t.Fatalf("deployment = %q, want %q, err=%v", actual, expected, err)
		}
	}
	assertDeployment(oldGood)
	sources, err := service.Store.LoadSources()
	if err != nil {
		t.Fatal(err)
	}
	for index := range sources.Sources {
		if sources.Sources[index].Name == "bad" {
			sources.Sources[index].URL = filepath.Join(root, "missing-repository")
		}
	}
	if err := service.Store.SaveSources(sources); err != nil {
		t.Fatal(err)
	}
	synced, err := service.SyncSources()
	if err != nil {
		t.Fatal(err)
	}
	if synced.Configured != 2 || synced.Updated != 1 || synced.Failed != 1 || synced.SkillCount != 1 || !synced.Applied || synced.DeploymentError != "" {
		t.Fatalf("sync result = %+v", synced)
	}
	statuses := make(map[string]string)
	for _, item := range synced.Results {
		statuses[item.Name] = item.Status
		if item.Name == "bad" && item.Error == "" {
			t.Fatal("failed source must report its error")
		}
	}
	if statuses["good"] != "updated" || statuses["bad"] != "error" {
		t.Fatalf("sync statuses = %+v", statuses)
	}
	manager := catalog.New(service.Store)
	newGood, err := manager.ResolveLibrary(oldGood.ID)
	if err != nil {
		t.Fatal(err)
	}
	if newGood.Hash == oldGood.Hash || newGood.Description != "Updated repository Skill" {
		t.Fatalf("updated skill = %+v", newGood)
	}
	assertDeployment(newGood)
	removed, err := service.RemoveSource("bad")
	if err != nil {
		t.Fatal(err)
	}
	if !removed.BindingRemoved || !removed.CheckoutRemoved {
		t.Fatalf("source removal = %+v", removed)
	}
	if _, err := manager.ResolveLibrary(imported["bad"].ID); err != nil {
		t.Fatalf("removing source must retain imported snapshot: %v", err)
	}
}
