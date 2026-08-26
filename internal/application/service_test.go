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
