package adapter

import (
	"path/filepath"
	"testing"

	"github.com/zzzzzyijie/skm/internal/domain"
)

func TestTargetPaths(t *testing.T) {
	tests := []struct {
		agent     domain.Agent
		placement domain.Placement
		want      string
	}{
		{domain.AgentClaude, domain.PlacementUser, filepath.Join("/users/test", ".claude", "skills", "review")},
		{domain.AgentCodex, domain.PlacementUser, filepath.Join("/users/test", ".codex", "skills", "review")},
		{domain.AgentCursor, domain.PlacementUser, filepath.Join("/users/test", ".cursor", "skills", "review")},
		{domain.AgentCopilot, domain.PlacementUser, filepath.Join("/users/test", ".copilot", "skills", "review")},
		{domain.AgentGemini, domain.PlacementUser, filepath.Join("/users/test", ".gemini", "skills", "review")},
		{domain.AgentWindsurf, domain.PlacementUser, filepath.Join("/users/test", ".codeium", "windsurf", "skills", "review")},
		{domain.AgentKiro, domain.PlacementUser, filepath.Join("/users/test", ".kiro", "skills", "review")},
		{domain.AgentCline, domain.PlacementUser, filepath.Join("/users/test", ".agents", "skills", "review")},
		{domain.AgentOpenCode, domain.PlacementUser, filepath.Join("/users/test", ".config", "opencode", "skills", "review")},
		{domain.AgentTrae, domain.PlacementUser, filepath.Join("/users/test", ".trae", "skills", "review")},
		{domain.AgentHermes, domain.PlacementUser, filepath.Join("/users/test", ".hermes", "skills", "review")},
		{domain.AgentOpenClaw, domain.PlacementUser, filepath.Join("/users/test", ".openclaw", "skills", "review")},
		{domain.AgentClaude, domain.PlacementProject, filepath.Join("/repo", ".claude", "skills", "review")},
		{domain.AgentCodex, domain.PlacementProject, filepath.Join("/repo", ".codex", "skills", "review")},
		{domain.Agent("cursor"), domain.PlacementProject, filepath.Join("/repo", ".cursor", "skills", "review")},
	}
	for _, test := range tests {
		got, err := Target(test.agent, test.placement, "/users/test", "/repo", "review")
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Fatalf("Target(%s, %s) = %s, want %s", test.agent, test.placement, got, test.want)
		}
	}
}

func TestLegacyCodexTargetPaths(t *testing.T) {
	user, err := LegacyTarget(domain.AgentCodex, domain.PlacementUser, "/users/test", "/repo", "review")
	if err != nil || user != filepath.Join("/users/test", ".agents", "skills", "review") {
		t.Fatalf("legacy user target = %s, err=%v", user, err)
	}
	project, err := LegacyTarget(domain.AgentCodex, domain.PlacementProject, "/users/test", "/repo", "review")
	if err != nil || project != filepath.Join("/repo", ".agents", "skills", "review") {
		t.Fatalf("legacy project target = %s, err=%v", project, err)
	}
}

func TestCustomUserTarget(t *testing.T) {
	roots := map[domain.Agent]string{domain.Agent("my-agent"): "~/.my-agent/skills"}
	got, err := Target(domain.Agent("my-agent"), domain.PlacementUser, "/users/test", "/repo", "review", roots)
	if err != nil || got != filepath.Join("/users/test", ".my-agent", "skills", "review") {
		t.Fatalf("custom target = %s, err=%v", got, err)
	}
	_, err = Target(domain.Agent("my-agent"), domain.PlacementUser, "/users/test", "/repo", "review", map[domain.Agent]string{domain.Agent("my-agent"): "/tmp/skills"})
	if err == nil {
		t.Fatal("custom target outside user home should be rejected")
	}
}
