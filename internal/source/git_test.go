package source

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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
		Name: "team", URL: repository, Paths: []string{"skills/one"}, Tags: []string{"backend"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if bound.Revision == "" || len(imported) != 1 || imported[0].ID != "team/one" {
		t.Fatalf("unexpected import: source=%#v skills=%#v", bound, imported)
	}
	if imported[0].SourcePath != "skills/one" {
		t.Fatalf("source path = %q", imported[0].SourcePath)
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

func TestGitSourceSelectsSkillsByNameAndPersistsPaths(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	repository := t.TempDir()
	run(t, repository, "git", "init", "-b", "main")
	writeRepoSkill(t, repository, "skills/one", "one", "version one")
	writeRepoSkill(t, repository, "catalog/design/two", "two", "version two")
	commitAll(t, repository, "initial")

	storage := sourceStore(t)
	manager := NewGitManager(storage, catalog.New(storage))
	bound, imported, err := manager.AddSelected(domain.Source{
		Name: "market", URL: repository,
	}, []string{"two"})
	if err != nil {
		t.Fatal(err)
	}
	if len(imported) != 1 || imported[0].ID != "market/two" {
		t.Fatalf("imported Skills = %#v", imported)
	}
	if !reflect.DeepEqual(bound.Paths, []string{"catalog/design/two"}) {
		t.Fatalf("bound paths = %#v", bound.Paths)
	}
	sources, err := storage.LoadSources()
	if err != nil {
		t.Fatal(err)
	}
	if len(sources.Sources) != 1 || !reflect.DeepEqual(sources.Sources[0].Paths, bound.Paths) {
		t.Fatalf("stored sources = %#v", sources.Sources)
	}
	if _, err := catalog.New(storage).ResolveLibrary("market/one"); err == nil {
		t.Fatal("unselected Skill was imported")
	}
	if _, _, err := manager.AddSelected(domain.Source{Name: "missing", URL: repository}, []string{"unknown"}); err == nil {
		t.Fatal("missing requested Skill should fail")
	}
}

func TestGitSourceRejectsCredentialURL(t *testing.T) {
	storage := sourceStore(t)
	_, _, err := NewGitManager(storage, catalog.New(storage)).Add(domain.Source{
		Name: "secret", URL: "https://token@example.com/repo.git",
	})
	if err == nil {
		t.Fatal("expected credential URL to be rejected")
	}
}

func TestGitSourceRemoveDeletesCheckoutAndRetainsImportedSkill(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	repository := t.TempDir()
	run(t, repository, "git", "init", "-b", "main")
	writeRepoSkill(t, repository, "skills/one", "one", "version one")
	commitAll(t, repository, "initial")

	storage := sourceStore(t)
	manager := NewGitManager(storage, catalog.New(storage))
	_, imported, err := manager.Add(domain.Source{
		Name: "team", URL: repository, Paths: []string{"skills/one"},
	})
	if err != nil {
		t.Fatal(err)
	}
	checkoutPath := storage.SourcePath("team")
	if _, err := os.Stat(checkoutPath); err != nil {
		t.Fatalf("source checkout missing before remove: %v", err)
	}

	removed, err := manager.Remove("team")
	if err != nil {
		t.Fatal(err)
	}
	if !removed.BindingRemoved || !removed.CheckoutRemoved || removed.Source == nil {
		t.Fatalf("unexpected removal result: %#v", removed)
	}
	if _, err := os.Lstat(checkoutPath); !os.IsNotExist(err) {
		t.Fatalf("source checkout still exists: %v", err)
	}
	sources, err := storage.LoadSources()
	if err != nil || len(sources.Sources) != 0 {
		t.Fatalf("sources = %#v, err=%v", sources, err)
	}
	if _, err := catalog.New(storage).ResolveLibrary("team/one"); err != nil {
		t.Fatalf("imported Library Skill was removed: %v", err)
	}
	if _, err := os.Stat(imported[0].Path); err != nil {
		t.Fatalf("imported snapshot was removed: %v", err)
	}
}

func TestGitSourceRemoveDeletesOrphanedCheckout(t *testing.T) {
	storage := sourceStore(t)
	checkoutPath := storage.SourcePath("team")
	if err := os.MkdirAll(checkoutPath, 0o755); err != nil {
		t.Fatal(err)
	}

	removed, err := NewGitManager(storage, catalog.New(storage)).Remove("team")
	if err != nil {
		t.Fatal(err)
	}
	if removed.BindingRemoved || !removed.CheckoutRemoved || removed.Source != nil {
		t.Fatalf("unexpected orphan removal result: %#v", removed)
	}
	if _, err := os.Lstat(checkoutPath); !os.IsNotExist(err) {
		t.Fatalf("orphaned checkout still exists: %v", err)
	}
	if _, err := NewGitManager(storage, catalog.New(storage)).Remove("team"); err == nil {
		t.Fatal("missing source and checkout should not be removed twice")
	}
}

func TestGitSourceRemoveRefusesSymlinkedCheckout(t *testing.T) {
	storage := sourceStore(t)
	external := t.TempDir()
	marker := filepath.Join(external, "keep")
	if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	checkoutPath := storage.SourcePath("team")
	if err := os.Symlink(external, checkoutPath); err != nil {
		t.Fatal(err)
	}

	if _, err := NewGitManager(storage, catalog.New(storage)).Remove("team"); err == nil {
		t.Fatal("symlinked source checkout should be rejected")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("symlink target was modified: %v", err)
	}
	if info, err := os.Lstat(checkoutPath); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("source symlink was modified: info=%v err=%v", info, err)
	}
}

func TestFetchPinnedDoesNotChangeLibraryOrSourceBinding(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	repository := t.TempDir()
	run(t, repository, "git", "init", "-b", "main")
	writeRepoSkill(t, repository, "skills/one", "one", "version one")
	commitAll(t, repository, "initial")

	storage := sourceStore(t)
	manager := NewGitManager(storage, catalog.New(storage))
	sourceValue, _, err := manager.Add(domain.Source{
		Name: "team", URL: repository, Ref: "main", Paths: []string{"skills/one"}, Tags: []string{"backend"},
	})
	if err != nil {
		t.Fatal(err)
	}
	beforeCatalog, err := storage.LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	beforeSources, err := storage.LoadSources()
	if err != nil {
		t.Fatal(err)
	}

	writeRepoSkill(t, repository, "skills/one", "one", "version two")
	commitAll(t, repository, "update")
	updatedRevision := gitOutput(t, repository, "rev-parse", "HEAD")
	fetched, err := manager.FetchPinned(domain.Source{
		Name: "team", URL: repository, Ref: updatedRevision, Paths: []string{"skills/one"}, Tags: []string{"project"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fetched) != 1 || fetched[0].Revision != updatedRevision || fetched[0].Hash == beforeCatalog.Skills[0].Hash {
		t.Fatalf("unexpected pinned snapshot: %#v", fetched)
	}
	afterCatalog, err := storage.LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	afterSources, err := storage.LoadSources()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(afterCatalog, beforeCatalog) {
		t.Fatalf("FetchPinned changed Library catalog:\nbefore=%#v\nafter=%#v", beforeCatalog, afterCatalog)
	}
	if !reflect.DeepEqual(afterSources, beforeSources) {
		t.Fatalf("FetchPinned changed source binding:\nbefore=%#v\nafter=%#v", beforeSources, afterSources)
	}
	if sourceValue.Revision != beforeSources.Sources[0].Revision {
		t.Fatalf("unexpected initial source revision: %#v %#v", sourceValue, beforeSources.Sources[0])
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

func gitOutput(t *testing.T, directory string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return string(bytes.TrimSpace(output))
}
