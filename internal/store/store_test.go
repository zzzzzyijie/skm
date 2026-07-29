package store

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/zzzzzyijie/skm/internal/domain"
)

func TestEnsureAndYAMLRoundTrips(t *testing.T) {
	root := t.TempDir()
	storage, err := New(Paths{
		Home:        filepath.Join(root, "user", ".skm"),
		UserHome:    filepath.Join(root, "user"),
		ProjectRoot: filepath.Join(root, "project"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Ensure(); err != nil {
		t.Fatal(err)
	}
	config, err := storage.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(config.Defaults.Tags, []string{"general"}) {
		t.Fatalf("default config = %#v", config)
	}

	central := domain.Catalog{Skills: []domain.Skill{{ID: "local/one", Name: "one", Scope: domain.ScopePersonal}}}
	if err := storage.SaveCatalog(central); err != nil {
		t.Fatal(err)
	}
	project := domain.Catalog{Skills: []domain.Skill{{ID: "project/two", Name: "two", Scope: domain.ScopeProject}}}
	if err := storage.SaveProjectCatalog(project); err != nil {
		t.Fatal(err)
	}
	all, err := storage.LoadAllSkills()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("all skills = %#v", all)
	}

	sources := domain.Sources{Sources: []domain.Source{{Name: "team", URL: "git@example.invalid:team.git", Scope: domain.ScopeGlobal}}}
	if err := storage.SaveSources(sources); err != nil {
		t.Fatal(err)
	}
	loadedSources, err := storage.LoadSources()
	if err != nil {
		t.Fatal(err)
	}
	if len(loadedSources.Sources) != 1 || loadedSources.Sources[0].Name != "team" {
		t.Fatalf("sources = %#v", loadedSources)
	}

	state := domain.State{Installations: []domain.Installation{{SkillID: "local/one", Scope: domain.ScopePersonal}}}
	if err := storage.SaveState(state); err != nil {
		t.Fatal(err)
	}
	loadedState, err := storage.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	if len(loadedState.Installations) != 1 || loadedState.Installations[0].SkillID != "local/one" {
		t.Fatalf("state = %#v", loadedState)
	}
}

func TestUpsertAndRemoveSkill(t *testing.T) {
	storage := testStore(t)
	value := domain.Skill{ID: "local/one", Name: "one", Scope: domain.ScopePersonal, Description: "first"}
	if err := storage.UpsertSkill(value); err != nil {
		t.Fatal(err)
	}
	value.Description = "updated"
	if err := storage.UpsertSkill(value); err != nil {
		t.Fatal(err)
	}
	catalog, err := storage.LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Skills) != 1 || catalog.Skills[0].Description != "updated" {
		t.Fatalf("catalog = %#v", catalog)
	}
	if err := storage.RemoveSkill(value.ID, value.Scope); err != nil {
		t.Fatal(err)
	}
	catalog, err = storage.LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Skills) != 0 {
		t.Fatalf("skill was not removed: %#v", catalog)
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

func TestSyncProjectLock(t *testing.T) {
	storage := testStore(t)
	value := domain.Skill{
		ID: "team/review", Name: "review", Source: "team", Scope: domain.ScopeGlobal,
		Revision: "abc123", Hash: "deadbeef", Tags: []string{"review"}, Path: filepath.Join(storage.Paths.Home, "objects", "deadbeef", "review"),
	}
	state := domain.State{Installations: []domain.Installation{{
		SkillID: value.ID, Name: value.Name, Scope: domain.ScopeProject, ProjectRoot: storage.Paths.ProjectRoot,
		Agents: []domain.Agent{domain.AgentClaude, domain.AgentCodex}, Mode: domain.ModeSymlink,
	}}}
	if err := storage.SyncProjectLock(state, []domain.Skill{value}); err != nil {
		t.Fatal(err)
	}
	project, err := storage.LoadProjectCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Dependencies) != 1 || project.Dependencies[0].ID != value.ID {
		t.Fatalf("dependencies = %#v", project.Dependencies)
	}
	var lock domain.LockFile
	if err := loadYAML(filepath.Join(storage.Paths.ProjectRoot, ".skm", "lock.yaml"), &lock); err != nil {
		t.Fatal(err)
	}
	if len(lock.Skills) != 1 || lock.Skills[0].Revision != "abc123" || lock.Skills[0].Hash != "deadbeef" {
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
