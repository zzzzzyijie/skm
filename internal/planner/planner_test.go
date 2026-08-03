package planner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/fsx"
	"github.com/zzzzzyijie/skm/internal/store"
)

func TestApplyUserSymlinkIsIdempotentAndUpdatesManagedTarget(t *testing.T) {
	storage := plannerStore(t)
	first := plannerSkill(t, "review", "first", domain.LocationLibrary, "local/review")
	state := domain.State{Activations: []domain.Activation{{
		SkillID: first.ID, Name: first.Name, Placement: domain.PlacementUser,
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
	if link, err := os.Readlink(target); err != nil || link != first.Path {
		t.Fatalf("link = %s, err=%v, want %s", link, err, first.Path)
	}
	plan, err = engine.Build([]domain.Skill{first}, state)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleStatus(t, plan, domain.StatusUnchanged)

	second := plannerSkill(t, "review", "second", domain.LocationLibrary, "local/review")
	plan, err = engine.Build([]domain.Skill{second}, state)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleStatus(t, plan, domain.StatusReplaceManaged)
}

func TestBuildRefusesUnmanagedProjectTarget(t *testing.T) {
	storage := plannerStore(t)
	value := plannerSkill(t, "review", "body", domain.LocationProject, "project/review")
	target := filepath.Join(storage.Paths.ProjectRoot, ".codex", "skills", "review")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	state := domain.State{Activations: []domain.Activation{{
		SkillID: value.ID, Name: value.Name, Placement: domain.PlacementProject, ProjectRoot: storage.Paths.ProjectRoot,
		Agents: []domain.Agent{domain.AgentCodex}, Mode: domain.ModeSymlink, PinnedHash: value.Hash, PinnedPath: value.Path,
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
	value := plannerSkill(t, "review", "body", domain.LocationLibrary, "local/review")
	state := domain.State{Activations: []domain.Activation{{
		SkillID: value.ID, Name: value.Name, Placement: domain.PlacementUser,
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
	target := filepath.Join(storage.Paths.UserHome, ".codex", "skills", "review", "SKILL.md")
	if err := os.WriteFile(target, []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err = engine.Build([]domain.Skill{value}, state)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleStatus(t, plan, domain.StatusConflictUnmanaged)
}

func TestProjectCopyIgnoresFinderMetadata(t *testing.T) {
	storage := plannerStore(t)
	value := plannerSkill(t, "review", "body", domain.LocationLibrary, "local/review")
	state := domain.State{Activations: []domain.Activation{{
		SkillID: value.ID, Name: value.Name, Placement: domain.PlacementProject, ProjectRoot: storage.Paths.ProjectRoot,
		Agents: []domain.Agent{domain.AgentClaude}, Mode: domain.ModeCopy, PinnedHash: value.Hash, PinnedPath: value.Path,
	}}}
	engine := New(storage)
	plan, err := engine.BuildScoped([]domain.Skill{value}, state, domain.PlacementProject, storage.Paths.ProjectRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Apply(plan, &state); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(storage.Paths.ProjectRoot, ".claude", "skills", "review")
	if err := os.WriteFile(filepath.Join(target, ".DS_Store"), []byte("finder metadata"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err = engine.BuildScoped([]domain.Skill{value}, state, domain.PlacementProject, storage.Paths.ProjectRoot)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleStatus(t, plan, domain.StatusUnchanged)
	if err := engine.Disable(&state, map[string]struct{}{value.ID: {}}, domain.PlacementProject, storage.Paths.ProjectRoot, map[domain.Agent]struct{}{domain.AgentClaude: {}}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("project target still exists: %v", err)
	}
}

func TestBuildScopedDoesNotIncludeOtherPlacementConflicts(t *testing.T) {
	storage := plannerStore(t)
	projectSkill := plannerSkill(t, "project-review", "project", domain.LocationLibrary, "local/project-review")
	userSkill := plannerSkill(t, "user-review", "user", domain.LocationLibrary, "local/user-review")
	state := domain.State{Activations: []domain.Activation{
		{SkillID: projectSkill.ID, Name: projectSkill.Name, Placement: domain.PlacementProject, ProjectRoot: storage.Paths.ProjectRoot, Agents: []domain.Agent{domain.AgentClaude}, Mode: domain.ModeCopy, PinnedHash: projectSkill.Hash, PinnedPath: projectSkill.Path},
		{SkillID: userSkill.ID, Name: userSkill.Name, Placement: domain.PlacementUser, Agents: []domain.Agent{domain.AgentCodex}, Mode: domain.ModeSymlink},
	}}
	engine := New(storage)
	projectPlan, err := engine.BuildScoped([]domain.Skill{projectSkill, userSkill}, state, domain.PlacementProject, storage.Paths.ProjectRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Apply(projectPlan, &state); err != nil {
		t.Fatal(err)
	}
	projectTarget := filepath.Join(storage.Paths.ProjectRoot, ".claude", "skills", projectSkill.Name, "SKILL.md")
	if err := os.WriteFile(projectTarget, []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}
	userPlan, err := engine.BuildScoped([]domain.Skill{projectSkill, userSkill}, state, domain.PlacementUser, "")
	if err != nil {
		t.Fatal(err)
	}
	assertSingleStatus(t, userPlan, domain.StatusCreate)
	if err := engine.Apply(userPlan, &state); err != nil {
		t.Fatal(err)
	}
}

func TestDisableRemovesLegacyCodexDeployment(t *testing.T) {
	storage := plannerStore(t)
	value := plannerSkill(t, "review", "body", domain.LocationLibrary, "local/review")
	legacyTarget := filepath.Join(storage.Paths.UserHome, ".agents", "skills", "review")
	if err := os.MkdirAll(filepath.Dir(legacyTarget), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(value.Path, legacyTarget); err != nil {
		t.Fatal(err)
	}
	state := domain.State{
		Activations: []domain.Activation{{
			SkillID: value.ID, Name: value.Name, Placement: domain.PlacementUser,
			Agents: []domain.Agent{domain.AgentCodex}, Mode: domain.ModeSymlink,
		}},
		Deployments: []domain.Deployment{{
			SkillID: value.ID, Name: value.Name, Agent: domain.AgentCodex,
			Placement: domain.PlacementUser, Target: legacyTarget,
			SourcePath: value.Path, Mode: domain.ModeSymlink, Hash: value.Hash,
		}},
	}
	if err := New(storage).Disable(&state, map[string]struct{}{value.ID: {}}, domain.PlacementUser, "", map[domain.Agent]struct{}{domain.AgentCodex: {}}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(legacyTarget); !os.IsNotExist(err) {
		t.Fatalf("legacy target still exists: %v", err)
	}
	if len(state.Deployments) != 0 || len(state.Activations) != 0 {
		t.Fatalf("legacy deployment state remains: %#v", state)
	}
}

func TestSameNameUserActivationsAreRejected(t *testing.T) {
	storage := plannerStore(t)
	team := plannerSkill(t, "review", "team", domain.LocationLibrary, "team/review")
	local := plannerSkill(t, "review", "local", domain.LocationLibrary, "local/review")
	state := domain.State{Activations: []domain.Activation{
		{SkillID: team.ID, Name: team.Name, Placement: domain.PlacementUser, Agents: []domain.Agent{domain.AgentClaude}, Mode: domain.ModeSymlink},
		{SkillID: local.ID, Name: local.Name, Placement: domain.PlacementUser, Agents: []domain.Agent{domain.AgentClaude}, Mode: domain.ModeSymlink},
	}}
	_, err := New(storage).Build([]domain.Skill{team, local}, state)
	if err == nil || !strings.Contains(err.Error(), "multiple Skills target") {
		t.Fatalf("expected same-name conflict, got %v", err)
	}
}

func TestPinnedActivationUsesExactSnapshot(t *testing.T) {
	storage := plannerStore(t)
	current := plannerSkill(t, "review", "current", domain.LocationLibrary, "team/review")
	pinned := plannerSkill(t, "review", "pinned", domain.LocationLibrary, "team/review")
	state := domain.State{Activations: []domain.Activation{{
		SkillID: pinned.ID, Name: pinned.Name, Placement: domain.PlacementProject, ProjectRoot: storage.Paths.ProjectRoot,
		Agents: []domain.Agent{domain.AgentCodex}, Mode: domain.ModeSymlink, PinnedHash: pinned.Hash, PinnedPath: pinned.Path,
	}}}
	plan, err := New(storage).Build([]domain.Skill{current}, state)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Operations) != 1 || plan.Operations[0].SourcePath != pinned.Path {
		t.Fatalf("plan did not use pinned snapshot: %#v", plan)
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
	storage, err := store.New(store.Paths{
		Home: filepath.Join(root, "user", ".skm"), UserHome: filepath.Join(root, "user"), ProjectRoot: filepath.Join(root, "project"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Ensure(); err != nil {
		t.Fatal(err)
	}
	return storage
}

func plannerSkill(t *testing.T, name, body string, location domain.SkillLocation, id string) domain.Skill {
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
	return domain.Skill{ID: id, Name: name, Location: location, Path: dir, Hash: hash}
}
