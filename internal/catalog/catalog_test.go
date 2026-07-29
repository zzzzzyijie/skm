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

func TestResolveUsesScopePriority(t *testing.T) {
	values := []domain.Skill{
		{ID: "team/review", Name: "review", Scope: domain.ScopeGlobal},
		{ID: "local/review", Name: "review", Scope: domain.ScopePersonal},
		{ID: "project/review", Name: "review", Scope: domain.ScopeProject},
	}
	value, err := Resolve(values, "review")
	if err != nil {
		t.Fatal(err)
	}
	if value.ID != "project/review" {
		t.Fatalf("resolved %s", value.ID)
	}
	qualified, err := Resolve(values, "team/review")
	if err != nil {
		t.Fatal(err)
	}
	if qualified.ID != "team/review" {
		t.Fatalf("qualified resolved %s", qualified.ID)
	}
}

func TestResolveRejectsSameScopeAmbiguity(t *testing.T) {
	values := []domain.Skill{
		{ID: "one/review", Name: "review", Scope: domain.ScopeGlobal},
		{ID: "two/review", Name: "review", Scope: domain.ScopeGlobal},
	}
	_, err := Resolve(values, "review")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity, got %v", err)
	}
}

func TestAddLocalAppliesDefaultAndExplicitTags(t *testing.T) {
	storage := newStore(t)
	manager := New(storage)
	defaultSkill := makeSkill(t, "default-skill")
	added, err := manager.AddLocal(defaultSkill, "", domain.ScopePersonal, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(added.Tags, []string{"general"}) {
		t.Fatalf("default tags = %#v", added.Tags)
	}

	explicitSkill := makeSkill(t, "explicit-skill")
	added, err = manager.AddLocal(explicitSkill, "", domain.ScopePersonal, []string{"testing"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(added.Tags, []string{"testing"}) {
		t.Fatalf("explicit tags = %#v", added.Tags)
	}
	if strings.Contains(strings.Join(added.Tags, ","), "general") {
		t.Fatal("explicit tags unexpectedly contain general")
	}
}

func TestProjectSkillIsStoredInProjectManifest(t *testing.T) {
	storage := newStore(t)
	manager := New(storage)
	value, err := manager.AddLocal(makeSkill(t, "project-skill"), "", domain.ScopeProject, nil)
	if err != nil {
		t.Fatal(err)
	}
	if value.ProjectRoot != storage.Paths.ProjectRoot {
		t.Fatalf("project root = %s", value.ProjectRoot)
	}
	if _, err := os.Stat(filepath.Join(storage.Paths.ProjectRoot, ".skm", "project.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(storage.Paths.ProjectRoot, ".skm", "skills", "project-skill", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(storage.Paths.ProjectRoot, ".skm", "project.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), storage.Paths.ProjectRoot) || !strings.Contains(string(raw), ".skm/skills/project-skill") {
		t.Fatalf("project manifest is not portable:\n%s", raw)
	}
}

func newStore(t *testing.T) *store.Store {
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
