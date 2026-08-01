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
		{domain.AgentClaude, domain.PlacementProject, filepath.Join("/repo", ".claude", "skills", "review")},
		{domain.AgentCodex, domain.PlacementProject, filepath.Join("/repo", ".codex", "skills", "review")},
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
