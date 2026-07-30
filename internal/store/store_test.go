package store

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zzzzzyijie/skm/internal/domain"
)

func TestEnsureAndYAMLRoundTrips(t *testing.T) {
	storage := testStore(t)
	config, err := storage.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(config.Defaults.Tags, []string{"general"}) {
		t.Fatalf("default config = %#v", config)
	}
	if err := storage.SaveCatalog(domain.Catalog{Skills: []domain.Skill{{ID: "local/one", Name: "one", Location: domain.LocationLibrary}}}); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveProjectCatalog(domain.Catalog{Skills: []domain.Skill{{ID: "project/two", Name: "two", Location: domain.LocationProject, Path: filepath.Join(storage.Paths.ProjectRoot, ".skm", "skills", "two")}}}); err != nil {
		t.Fatal(err)
	}
	all, err := storage.LoadAllSkills()
	if err != nil || len(all) != 2 {
		t.Fatalf("all skills = %#v, err=%v", all, err)
	}
	sources := domain.Sources{Sources: []domain.Source{{Name: "team", URL: "git@example.invalid:team.git"}}}
	if err := storage.SaveSources(sources); err != nil {
		t.Fatal(err)
	}
	loadedSources, err := storage.LoadSources()
	if err != nil || len(loadedSources.Sources) != 1 {
		t.Fatalf("sources = %#v, err=%v", loadedSources, err)
	}
	state := domain.State{Activations: []domain.Activation{{SkillID: "local/one", Placement: domain.PlacementUser}}}
	if err := storage.SaveState(state); err != nil {
		t.Fatal(err)
	}
	loadedState, err := storage.LoadState()
	if err != nil || len(loadedState.Activations) != 1 || loadedState.Activations[0].SkillID != "local/one" {
		t.Fatalf("state = %#v, err=%v", loadedState, err)
	}
}

func TestUpsertAndRemoveSkill(t *testing.T) {
	storage := testStore(t)
	value := domain.Skill{ID: "local/one", Name: "one", Location: domain.LocationLibrary, Description: "first"}
	if err := storage.UpsertSkill(value); err != nil {
		t.Fatal(err)
	}
	value.Description = "updated"
	if err := storage.UpsertSkill(value); err != nil {
		t.Fatal(err)
	}
	catalog, err := storage.LoadCatalog()
	if err != nil || len(catalog.Skills) != 1 || catalog.Skills[0].Description != "updated" {
		t.Fatalf("catalog = %#v, err=%v", catalog, err)
	}
	if err := storage.RemoveSkill(value.ID, value.Location); err != nil {
		t.Fatal(err)
	}
	catalog, _ = storage.LoadCatalog()
	if len(catalog.Skills) != 0 {
		t.Fatalf("skill was not removed: %#v", catalog)
	}
}

func TestSchemaV1StateAndCatalogAreMigrated(t *testing.T) {
	storage := testStore(t)
	oldCatalog := "version: 1\nskills:\n  - id: local/one\n    name: one\n    scope: personal\n"
	if err := os.WriteFile(filepath.Join(storage.Paths.Home, "catalog.yaml"), []byte(oldCatalog), 0o644); err != nil {
		t.Fatal(err)
	}
	oldState := "version: 1\ninstallations:\n  - skillId: local/one\n    name: one\n    scope: personal\n    agents: [codex]\n    mode: symlink\n"
	if err := os.WriteFile(filepath.Join(storage.Paths.Home, "state", "state.yaml"), []byte(oldState), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := storage.LoadCatalog()
	if err != nil || catalog.Skills[0].Location != domain.LocationLibrary {
		t.Fatalf("catalog migration = %#v, err=%v", catalog, err)
	}
	state, err := storage.LoadState()
	if err != nil || len(state.Activations) != 1 || state.Activations[0].Placement != domain.PlacementUser {
		t.Fatalf("state migration = %#v, err=%v", state, err)
	}
	if err := storage.SaveState(state); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(storage.Paths.Home, "state", "state.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "installations:") || !strings.Contains(string(raw), "activations:") {
		t.Fatalf("state was not written as v2:\n%s", raw)
	}
}

func TestFindProjectRoot(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "repo")
	nested := filepath.Join(project, "packages", "app")
	if err := os.MkdirAll(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := FindProjectRoot(nested); got != project {
		t.Fatalf("FindProjectRoot = %s, want %s", got, project)
	}
}

func TestLockCanBeAcquiredAndReleased(t *testing.T) {
	storage := testStore(t)
	unlock, err := storage.Lock()
	if err != nil {
		t.Fatal(err)
	}
	if err := unlock(); err != nil {
		t.Fatal(err)
	}
	secondUnlock, err := storage.Lock()
	if err != nil {
		t.Fatal(err)
	}
	if err := secondUnlock(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveProjectLockIncludesRequirementsAndVendoredSkills(t *testing.T) {
	storage := testStore(t)
	project := domain.Catalog{
		Dependencies: []domain.ProjectDependency{{ID: "team/review", Name: "review", Source: "team", Revision: "abc123", Hash: "deadbeef", Tags: []string{"review"}}},
		Skills:       []domain.Skill{{ID: "project/release", Name: "release", Source: "project", Hash: "cafe", Tags: []string{"release"}}},
	}
	if err := storage.SaveProjectLock(project); err != nil {
		t.Fatal(err)
	}
	var lock domain.LockFile
	if err := loadYAML(filepath.Join(storage.Paths.ProjectRoot, ".skm", "lock.yaml"), &lock); err != nil {
		t.Fatal(err)
	}
	if len(lock.Skills) != 2 || lock.Skills[1].ID != "team/review" {
		t.Fatalf("lock = %#v", lock)
	}
}

func testStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	storage, err := New(Paths{Home: filepath.Join(root, ".skm"), UserHome: root, ProjectRoot: filepath.Join(root, "project")})
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Ensure(); err != nil {
		t.Fatal(err)
	}
	return storage
}
