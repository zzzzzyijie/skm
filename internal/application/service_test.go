package application

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zzzzzyijie/skm/internal/store"
)

func TestSkillActivationAndPromptUseCases(t *testing.T) {
	service, root := newTestService(t)
	skillPath := filepath.Join(root, "fixture", "sample")
	if err := os.MkdirAll(skillPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("---\nname: sample\ndescription: Sample Skill\n---\n\nInstructions.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	added, err := service.AddSkill(AddSkillInput{Path: skillPath, Source: "local", Tags: []string{"desktop"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Enable(ActivationInput{Skills: []string{added.ID}, Agents: []string{"codex"}, Mode: "auto"}); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "user", ".codex", "skills", "sample")
	if info, err := os.Lstat(target); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected managed symlink at %s: info=%v err=%v", target, info, err)
	}
	if _, err := service.Disable(ActivationInput{Skills: []string{added.ID}, Agents: []string{"codex"}}); err != nil {
		t.Fatal(err)
	}

	promptValue, err := service.CreatePrompt(PromptWriteInput{
		Name: "review", Description: "Review code", Body: "Review this code", Source: "local", Tags: []string{"desktop"},
	})
	if err != nil {
		t.Fatal(err)
	}
	details, err := service.GetPrompt(promptValue.ID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(details.Body) != "Review this code" {
		t.Fatalf("unexpected Prompt body %q", details.Body)
	}
}

func TestInvokeReturnsStableErrorKind(t *testing.T) {
	service, _ := newTestService(t)
	_, err := service.Invoke(context.Background(), "skills.get", json.RawMessage(`{"id":"missing"}`))
	var appError *Error
	if !errors.As(err, &appError) {
		t.Fatalf("expected application Error, got %v", err)
	}
	if appError.Kind != "not_found" {
		t.Fatalf("expected not_found, got %q", appError.Kind)
	}
}

func TestEmptyCollectionsUseArraysInRPCContract(t *testing.T) {
	service, _ := newTestService(t)
	plan, err := service.ActivationStatus()
	if err != nil {
		t.Fatal(err)
	}
	if plan.Operations == nil {
		t.Fatal("expected empty operations array, got nil")
	}
	sources, err := service.ListSources()
	if err != nil {
		t.Fatal(err)
	}
	if sources == nil {
		t.Fatal("expected empty sources array, got nil")
	}
}

func TestEmptyProjectCanDeployFirstSkill(t *testing.T) {
	service, root := newTestService(t)
	skillPath := filepath.Join(root, "fixture", "sample")
	if err := os.MkdirAll(skillPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("---\nname: sample\ndescription: Sample Skill\n---\n\nInstructions.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	added, err := service.AddSkill(AddSkillInput{Path: skillPath, Source: "local"})
	if err != nil {
		t.Fatal(err)
	}
	projectRoot := filepath.Join(root, "empty-project")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	project, err := service.AddProject(AddProjectInput{Path: projectRoot, Name: "empty"})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := service.DeployProject(ProjectDeployInput{Project: project.ID, Skill: added.ID, Agents: []string{"codex"}, Mode: "symlink", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Applied || len(preview.Plan.Operations) != 1 || preview.Plan.Operations[0].Status != "create" {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	applied, err := service.DeployProject(ProjectDeployInput{Project: project.ID, Skill: added.ID, Agents: []string{"codex"}, Mode: "symlink"})
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Applied {
		t.Fatal("expected project deployment to be applied")
	}
	target := filepath.Join(projectRoot, ".codex", "skills", "sample")
	if info, err := os.Lstat(target); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected first project deployment at %s: info=%v err=%v", target, info, err)
	}
	details, err := service.GetProject(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if details.Scan.SkillCount != 1 || details.Scan.AgentCounts["codex"] != 1 {
		t.Fatalf("unexpected project scan: %+v", details.Scan)
	}
}

func TestSharedStoreRejectsStaleEditAndRecovers(t *testing.T) {
	first, root := newTestService(t)
	skillPath := filepath.Join(root, "fixture", "shared")
	if err := os.MkdirAll(skillPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("---\nname: shared\ndescription: Initial\n---\n\nInitial.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	added, err := first.AddSkill(AddSkillInput{Path: skillPath, Source: "local"})
	if err != nil {
		t.Fatal(err)
	}
	stale, err := first.GetSkill(added.ID)
	if err != nil {
		t.Fatal(err)
	}
	second := New(first.Store)
	newContent := "---\nname: shared\ndescription: Changed by CLI\n---\n\nCLI version.\n"
	if _, err := second.UpdateSkill(UpdateSkillInput{ID: added.ID, Content: newContent, BaseHash: stale.Hash}); err != nil {
		t.Fatal(err)
	}
	_, err = first.UpdateSkill(UpdateSkillInput{ID: added.ID, Content: stale.Content + "\nApp draft.\n", BaseHash: stale.Hash})
	if err == nil {
		t.Fatal("expected stale edit conflict")
	}
	var appError *Error
	if !errors.As(wrap(err), &appError) || appError.Kind != "conflict" {
		t.Fatalf("expected stale edit conflict, got %v", err)
	}
	latest, err := first.GetSkill(added.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(latest.Content, "CLI version") {
		t.Fatalf("external update was lost: %q", latest.Content)
	}
	if _, err := first.UpdateSkill(UpdateSkillInput{ID: added.ID, Content: stale.Content + "\nRecovered draft.\n", BaseHash: latest.Hash}); err != nil {
		t.Fatalf("explicit recovery failed: %v", err)
	}
}

func newTestService(t *testing.T) (*Service, string) {
	t.Helper()
	root := t.TempDir()
	storage, err := store.New(store.Paths{
		Home: filepath.Join(root, "state"), UserHome: filepath.Join(root, "user"), ProjectRoot: filepath.Join(root, "project"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Ensure(); err != nil {
		t.Fatal(err)
	}
	return New(storage), root
}
