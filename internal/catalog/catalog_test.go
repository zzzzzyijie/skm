package catalog

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zzzzzyijie/skm/internal/domain"
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
