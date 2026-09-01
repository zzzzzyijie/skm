package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zzzzzyijie/skm/internal/domain"
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

func TestProjectAccessStatusDistinguishesRecoveryPaths(t *testing.T) {
	root := t.TempDir()
	exists, access, message := inspectProjectAccess(root)
	if !exists || access != "available" || message != "" {
		t.Fatalf("available directory = exists:%v access:%q message:%q", exists, access, message)
	}

	exists, access, message = inspectProjectAccess(filepath.Join(root, "missing"))
	if exists || access != "missing" || message == "" {
		t.Fatalf("missing directory = exists:%v access:%q message:%q", exists, access, message)
	}

	if access := projectAccessForError(fs.ErrPermission); access != "permission-denied" {
		t.Fatalf("permission error access = %q", access)
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

func TestAdvancedProjectRequireVendorApplyAndRemove(t *testing.T) {
	service, root := newTestService(t)
	requiredPath := writeTestSkill(t, root, "required", "Required Skill")
	required, err := service.AddSkill(AddSkillInput{Path: requiredPath, Source: "team"})
	if err != nil {
		t.Fatal(err)
	}
	library, err := service.Store.LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	library.Skills[0].Revision = "0123456789abcdef"
	library.Skills[0].SourcePath = "."
	if err := service.Store.SaveCatalog(library); err != nil {
		t.Fatal(err)
	}
	if err := service.Store.SaveSources(domain.Sources{Sources: []domain.Source{{Name: "team", URL: "https://example.invalid/team.git", Ref: "main"}}}); err != nil {
		t.Fatal(err)
	}
	vendoredPath := writeTestSkill(t, root, "vendored", "Vendored Skill")
	vendored, err := service.AddSkill(AddSkillInput{Path: vendoredPath, Source: "local"})
	if err != nil {
		t.Fatal(err)
	}
	projectRoot := filepath.Join(root, "advanced-project")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	project, err := service.AddProject(AddProjectInput{Path: projectRoot, Name: "advanced"})
	if err != nil {
		t.Fatal(err)
	}
	requiredResult, err := service.RequireProjectSkill(ProjectRequireInput{
		Project: project.ID, Skill: required.ID, Agents: []string{"codex"}, Mode: domain.ModeSymlink,
	})
	if err != nil {
		t.Fatal(err)
	}
	if requiredResult.Applied || len(requiredResult.Manifest.Dependencies) != 1 {
		t.Fatalf("unexpected require result: %+v", requiredResult)
	}
	vendoredResult, err := service.VendorProjectSkill(ProjectVendorInput{
		Project: project.ID, Skill: vendored.ID, Agents: []string{"claude"}, Mode: domain.ModeCopy,
	})
	if err != nil {
		t.Fatal(err)
	}
	if vendoredResult.Applied || len(vendoredResult.Manifest.Skills) != 1 {
		t.Fatalf("unexpected vendor result: %+v", vendoredResult)
	}
	applied, err := service.ApplyProject(ProjectApplyInput{Project: project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Applied || len(applied.Plan.Operations) != 2 {
		t.Fatalf("unexpected apply result: %+v", applied)
	}
	if info, err := os.Lstat(filepath.Join(projectRoot, ".codex", "skills", "required")); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("required target was not linked: info=%v err=%v", info, err)
	}
	if info, err := os.Stat(filepath.Join(projectRoot, ".claude", "skills", "vendored", "SKILL.md")); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("vendored target was not copied: info=%v err=%v", info, err)
	}
	removed, err := service.RemoveProjectEntry(ProjectEntryRemoveInput{Project: project.ID, Entry: vendoredResult.Skill.ID})
	if err != nil {
		t.Fatal(err)
	}
	if removed.RemovedID != vendoredResult.Skill.ID || len(removed.Manifest.Skills) != 0 {
		t.Fatalf("unexpected remove result: %+v", removed)
	}
}

func TestPromptRenderAndSkillPromptHistoryRollback(t *testing.T) {
	service, root := newTestService(t)
	promptValue, err := service.CreatePrompt(PromptWriteInput{
		Name: "explain", Description: "Explain a topic", Body: "Explain {{topic}} in {{style}}.", Source: "local",
		Variables: []domain.PromptVariable{
			{Name: "topic", Label: "Topic", Type: "text", Required: true},
			{Name: "style", Label: "Style", Type: "select", Default: "plain", Options: []string{"plain", "brief"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := service.RenderPrompt(PromptRenderInput{ID: promptValue.ID, Values: map[string]string{"topic": "history", "style": "brief"}})
	if err != nil || strings.TrimSpace(rendered.Content) != "Explain history in brief." {
		t.Fatalf("render = %+v, err=%v", rendered, err)
	}
	promptDetails, err := service.GetPrompt(promptValue.ID)
	if err != nil {
		t.Fatal(err)
	}
	updatedPrompt, err := service.UpdatePrompt(PromptWriteInput{
		ID: promptValue.ID, Name: promptDetails.Name, Description: promptDetails.Description,
		Body: "Updated {{topic}} in {{style}}.", Variables: promptDetails.Variables, BaseHash: promptDetails.Hash,
	})
	if err != nil {
		t.Fatal(err)
	}
	promptHistory, err := service.ListHistory(HistoryInput{Kind: domain.HistoryPrompt, ItemID: promptValue.ID})
	if err != nil || len(promptHistory) != 2 || promptHistory[0].Hash != updatedPrompt.Hash {
		t.Fatalf("prompt history = %+v, err=%v", promptHistory, err)
	}
	if _, err := service.RollbackHistory(HistoryEntryInput{Kind: domain.HistoryPrompt, ItemID: promptValue.ID, EntryID: promptHistory[1].ID}); err != nil {
		t.Fatal(err)
	}
	restoredPrompt, err := service.GetPrompt(promptValue.ID)
	if err != nil || !strings.Contains(restoredPrompt.Body, "Explain {{topic}}") {
		t.Fatalf("restored Prompt = %+v, err=%v", restoredPrompt, err)
	}

	skillPath := writeTestSkill(t, root, "history-skill", "History Skill")
	skillValue, err := service.AddSkill(AddSkillInput{Path: skillPath, Source: "local"})
	if err != nil {
		t.Fatal(err)
	}
	skillDetails, err := service.GetSkill(skillValue.ID)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(skillDetails.Content, "History Skill", "Changed History", 1)
	if _, err := service.UpdateSkill(UpdateSkillInput{ID: skillValue.ID, Content: changed, BaseHash: skillDetails.Hash, Tags: skillDetails.Tags}); err != nil {
		t.Fatal(err)
	}
	skillHistory, err := service.ListHistory(HistoryInput{Kind: domain.HistorySkill, ItemID: skillValue.ID})
	if err != nil || len(skillHistory) != 2 {
		t.Fatalf("skill history = %+v, err=%v", skillHistory, err)
	}
	diff, err := service.DiffHistory(HistoryDiffInput{Kind: domain.HistorySkill, ItemID: skillValue.ID, From: skillHistory[1].ID, To: "current"})
	if err != nil || !strings.Contains(diff.Diff, "-description: History Skill") || !strings.Contains(diff.Diff, "+description: Changed History") {
		t.Fatalf("skill diff = %+v, err=%v", diff, err)
	}
	if _, err := service.RollbackHistory(HistoryEntryInput{Kind: domain.HistorySkill, ItemID: skillValue.ID, EntryID: skillHistory[1].ID}); err != nil {
		t.Fatal(err)
	}
	restoredSkill, err := service.GetSkill(skillValue.ID)
	if err != nil || !strings.Contains(restoredSkill.Content, "History Skill") {
		t.Fatalf("restored Skill = %+v, err=%v", restoredSkill, err)
	}
}

func writeTestSkill(t *testing.T, root, name, description string) string {
	t.Helper()
	path := filepath.Join(root, "fixture", name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\nInstructions.\n", name, description)
	if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
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
