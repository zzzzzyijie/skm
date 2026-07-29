package source

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/store"
)

func TestGitSourceCustomPathAndUpdate(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	repository := t.TempDir()
	run(t, repository, "git", "init", "-b", "main")
	writeRepoSkill(t, repository, "skills/one", "one", "version one")
	writeRepoSkill(t, repository, "skills/two", "two", "not bound")
	commitAll(t, repository, "initial")

	storage := sourceStore(t)
	manager := NewGitManager(storage, catalog.New(storage))
	bound, imported, err := manager.Add(domain.Source{
		Name: "team", URL: repository, Paths: []string{"skills/one"}, Tags: []string{"backend"}, Scope: domain.ScopeGlobal,
	})
	if err != nil {
		t.Fatal(err)
	}
	if bound.Revision == "" || len(imported) != 1 || imported[0].ID != "team/one" {
		t.Fatalf("unexpected import: source=%#v skills=%#v", bound, imported)
	}
	if _, err := catalog.New(storage).Resolve("team/two"); err == nil {
		t.Fatal("unbound Skill was imported")
	}
	firstHash := imported[0].Hash

	writeRepoSkill(t, repository, "skills/one", "one", "version two")
	commitAll(t, repository, "update")
	updated, imported, err := manager.Update([]string{"team"})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated) != 1 || len(imported) != 1 {
		t.Fatalf("unexpected update: %#v %#v", updated, imported)
	}
	if imported[0].Hash == firstHash {
		t.Fatal("updated source did not create a new snapshot")
	}
	if len(imported[0].Tags) != 1 || imported[0].Tags[0] != "backend" {
		t.Fatalf("tags were not retained: %#v", imported[0].Tags)
	}
}

func TestGitSourceRejectsCredentialURL(t *testing.T) {
	storage := sourceStore(t)
	_, _, err := NewGitManager(storage, catalog.New(storage)).Add(domain.Source{
		Name: "secret", URL: "https://token@example.com/repo.git", Scope: domain.ScopeGlobal,
	})
	if err == nil {
		t.Fatal("expected credential URL to be rejected")
	}
}

func sourceStore(t *testing.T) *store.Store {
	t.Helper()
	root := t.TempDir()
	storage, err := store.New(store.Paths{
		Home:        filepath.Join(root, "home", ".skm"),
		UserHome:    filepath.Join(root, "home"),
		ProjectRoot: filepath.Join(root, "project"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Ensure(); err != nil {
		t.Fatal(err)
	}
	return storage
}

func writeRepoSkill(t *testing.T, root, relative, name, body string) {
	t.Helper()
	dir := filepath.Join(root, relative)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: Git test Skill\n---\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commitAll(t *testing.T, repository, message string) {
	t.Helper()
	run(t, repository, "git", "add", ".")
	run(t, repository, "git", "-c", "user.name=skm-test", "-c", "user.email=skm@example.invalid", "commit", "-m", message)
}

func run(t *testing.T, directory, name string, args ...string) {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, output)
	}
}
