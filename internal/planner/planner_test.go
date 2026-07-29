package planner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/fsx"
	"github.com/zzzzzyijie/skm/internal/store"
)

func TestApplySymlinkIsIdempotentAndUpdatesManagedTarget(t *testing.T) {
	storage := plannerStore(t)
	first := plannerSkill(t, "review", "first", domain.ScopePersonal, "local/review")
	state := domain.State{Installations: []domain.Installation{{
		SkillID: first.ID, Name: first.Name, Scope: domain.ScopePersonal,
		Agents: []domain.Agent{domain.AgentClaude}, Mode: domain.ModeSymlink,
	}}}
	engine := New(storage)
	plan, err := engine.Build([]domain.Skill{first}, state)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleStatus(t, plan, domain.StatusCreate)
	if err := engine.Apply(plan, &state); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(storage.Paths.UserHome, ".claude", "skills", "review")
	link, err := os.Readlink(target)
	if err != nil {
		t.Fatal(err)
	}
	if link != first.Path {
		t.Fatalf("link = %s, want %s", link, first.Path)
	}

	plan, err = engine.Build([]domain.Skill{first}, state)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleStatus(t, plan, domain.StatusUnchanged)

	second := plannerSkill(t, "review", "second", domain.ScopePersonal, "local/review")
	plan, err = engine.Build([]domain.Skill{second}, state)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleStatus(t, plan, domain.StatusReplaceManaged)
	if err := engine.Apply(plan, &state); err != nil {
		t.Fatal(err)
	}
	link, err = os.Readlink(target)
	if err != nil {
		t.Fatal(err)
	}
	if link != second.Path {
		t.Fatalf("updated link = %s, want %s", link, second.Path)
	}
}

func TestBuildRefusesUnmanagedTarget(t *testing.T) {
	storage := plannerStore(t)
	value := plannerSkill(t, "review", "body", domain.ScopeProject, "project/review")
	target := filepath.Join(storage.Paths.ProjectRoot, ".agents", "skills", "review")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	state := domain.State{Installations: []domain.Installation{{
		SkillID: value.ID, Name: value.Name, Scope: domain.ScopeProject, ProjectRoot: storage.Paths.ProjectRoot,
		Agents: []domain.Agent{domain.AgentCodex}, Mode: domain.ModeSymlink,
	}}}
	plan, err := New(storage).Build([]domain.Skill{value}, state)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleStatus(t, plan, domain.StatusConflictUnmanaged)
	if err := New(storage).Apply(plan, &state); err == nil {
		t.Fatal("expected apply to refuse unmanaged target")
	}
}

func TestCopyModificationBecomesConflict(t *testing.T) {
	storage := plannerStore(t)
	value := plannerSkill(t, "review", "body", domain.ScopePersonal, "local/review")
	state := domain.State{Installations: []domain.Installation{{
		SkillID: value.ID, Name: value.Name, Scope: domain.ScopePersonal,
		Agents: []domain.Agent{domain.AgentCodex}, Mode: domain.ModeCopy,
	}}}
	engine := New(storage)
	plan, err := engine.Build([]domain.Skill{value}, state)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Apply(plan, &state); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(storage.Paths.UserHome, ".agents", "skills", "review", "SKILL.md")
	if err := os.WriteFile(target, []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err = engine.Build([]domain.Skill{value}, state)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleStatus(t, plan, domain.StatusConflictUnmanaged)
}

func TestPersonalTargetShadowsGlobalTarget(t *testing.T) {
	storage := plannerStore(t)
	global := plannerSkill(t, "review", "global", domain.ScopeGlobal, "team/review")
	personal := plannerSkill(t, "review", "personal", domain.ScopePersonal, "local/review")
	state := domain.State{Installations: []domain.Installation{
		{SkillID: global.ID, Name: global.Name, Scope: domain.ScopeGlobal, Agents: []domain.Agent{domain.AgentClaude}, Mode: domain.ModeSymlink},
		{SkillID: personal.ID, Name: personal.Name, Scope: domain.ScopePersonal, Agents: []domain.Agent{domain.AgentClaude}, Mode: domain.ModeSymlink},
	}}
	plan, err := New(storage).Build([]domain.Skill{global, personal}, state)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Operations) != 1 {
		t.Fatalf("operations = %d, want 1", len(plan.Operations))
	}
	if plan.Operations[0].SkillID != personal.ID {
		t.Fatalf("selected %s, want %s", plan.Operations[0].SkillID, personal.ID)
	}
}

func assertSingleStatus(t *testing.T, plan domain.Plan, want domain.OperationStatus) {
	t.Helper()
	if len(plan.Operations) != 1 || plan.Operations[0].Status != want {
		t.Fatalf("plan = %#v, want one %s operation", plan, want)
	}
}

func plannerStore(t *testing.T) *store.Store {
	t.Helper()
	root := t.TempDir()
	paths := store.Paths{
		Home:        filepath.Join(root, "user", ".skm"),
		UserHome:    filepath.Join(root, "user"),
		ProjectRoot: filepath.Join(root, "project"),
	}
	storage, err := store.New(paths)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Ensure(); err != nil {
		t.Fatal(err)
	}
	return storage
}

func plannerSkill(t *testing.T, name, body string, scope domain.Scope, id string) domain.Skill {
	t.Helper()
	dir := filepath.Join(t.TempDir(), body)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: test\n---\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, err := fsx.HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return domain.Skill{ID: id, Name: name, Scope: scope, Path: dir, Hash: hash}
}
