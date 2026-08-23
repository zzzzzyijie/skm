package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/skill"
	"github.com/zzzzzyijie/skm/internal/store"
)

func TestResolveRequiresFullIDForSameName(t *testing.T) {
	values := []domain.Skill{
		{ID: "team/review", Name: "review", Location: domain.LocationLibrary},
		{ID: "local/review", Name: "review", Location: domain.LocationLibrary},
	}
	if _, err := Resolve(values, "review"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity, got %v", err)
	}
	qualified, err := Resolve(values, "team/review")
	if err != nil || qualified.ID != "team/review" {
		t.Fatalf("qualified resolve = %#v, %v", qualified, err)
	}
}

func TestAddLocalAppliesDefaultAndExplicitTags(t *testing.T) {
	storage := newStore(t)
	manager := New(storage)
	added, err := manager.AddLocal(makeSkill(t, "default-skill"), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if added.Location != domain.LocationLibrary || !reflect.DeepEqual(added.Tags, []string{"general"}) {
		t.Fatalf("default Skill = %#v", added)
	}
	added, err = manager.AddLocal(makeSkill(t, "explicit-skill"), "", []string{"testing"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(added.Tags, []string{"testing"}) {
		t.Fatalf("explicit tags = %#v", added.Tags)
	}
}

func TestUpdateContentCreatesSnapshotAndPreservesAuxiliaryFiles(t *testing.T) {
	storage := newStore(t)
	manager := New(storage)
	source := makeSkill(t, "editable-skill")
	if err := os.MkdirAll(filepath.Join(source, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "references", "guide.md"), []byte("keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	added, err := manager.AddLocal(source, "", []string{"testing"})
	if err != nil {
		t.Fatal(err)
	}
	updatedContent := "---\nname: editable-skill\ndescription: updated description\n---\nupdated body\n"
	validated, err := manager.ValidateContent(added.ID, updatedContent)
	if err != nil {
		t.Fatal(err)
	}
	if validated.Description != "updated description" || validated.Files != 2 {
		t.Fatalf("validated document = %#v", validated)
	}
	updated, err := manager.UpdateContent(added.ID, updatedContent, added.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Hash == added.Hash || updated.Path == added.Path || updated.Description != "updated description" {
		t.Fatalf("updated Skill = %#v", updated)
	}
	if data, err := os.ReadFile(filepath.Join(updated.Path, "references", "guide.md")); err != nil || string(data) != "keep me\n" {
		t.Fatalf("preserved reference = %q, %v", data, err)
	}
	if data, err := os.ReadFile(filepath.Join(added.Path, "SKILL.md")); err != nil || strings.Contains(string(data), "updated body") {
		t.Fatalf("old immutable snapshot changed: %q, %v", data, err)
	}
	if _, err := manager.UpdateContent(added.ID, updatedContent, added.Hash); !errors.Is(err, ErrEditConflict) {
		t.Fatalf("stale update error = %v", err)
	}
	rename := strings.Replace(updatedContent, "name: editable-skill", "name: renamed-skill", 1)
	if _, err := manager.ValidateContent(added.ID, rename); err == nil || !strings.Contains(err.Error(), "cannot change") {
		t.Fatalf("rename validation error = %v", err)
	}
	if _, err := manager.UpdateContent(added.ID, rename, updated.Hash); err == nil || !strings.Contains(err.Error(), "cannot change") {
		t.Fatalf("rename error = %v", err)
	}
}

func TestUpdateRejectsGitAndProjectFollowedSkills(t *testing.T) {
	storage := newStore(t)
	manager := New(storage)
	gitSkill, err := manager.AddLocal(makeSkill(t, "git-skill"), "team", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveSources(domain.Sources{Sources: []domain.Source{{Name: "team", URL: "https://example.invalid/team.git"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.UpdateDirectory(gitSkill.ID, makeSkill(t, "git-skill"), gitSkill.Hash); !errors.Is(err, ErrNotEditable) {
		t.Fatalf("Git update error = %v", err)
	}

	projectSource := makeSkill(t, "followed-skill")
	document, err := skill.Validate(projectSource)
	if err != nil {
		t.Fatal(err)
	}
	followed, err := manager.ImportProject(document, filepath.Dir(projectSource), domain.AgentCodex, domain.ModeSymlink, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.UpdateDirectory(followed.ID, projectSource, followed.Hash); !errors.Is(err, ErrNotEditable) {
		t.Fatalf("followed update error = %v", err)
	}
}

func TestVendorCopiesProjectSkillAndRetainsLibrary(t *testing.T) {
	storage := newStore(t)
	manager := New(storage)
	original, err := manager.AddLocal(makeSkill(t, "project-skill"), "", []string{"personal"})
	if err != nil {
		t.Fatal(err)
	}
	vendored, err := manager.Vendor(original, []domain.Agent{domain.AgentCodex}, domain.ModeSymlink, nil)
	if err != nil {
		t.Fatal(err)
	}
	if vendored.Location != domain.LocationProject || vendored.ForkedFrom != original.ID {
		t.Fatalf("vendored = %#v", vendored)
	}
	if _, err := os.Stat(filepath.Join(storage.Paths.ProjectRoot, ".skm", "skills", "project-skill", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ResolveLibrary(original.ID); err != nil {
		t.Fatalf("personal Library copy was not retained: %v", err)
	}
}

func newStore(t *testing.T) *store.Store {
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

func makeSkill(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: test skill\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
