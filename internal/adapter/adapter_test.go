package adapter

import (
	"path/filepath"
	"testing"

	"github.com/zzzzzyijie/skm/internal/domain"
)

func TestTargetPaths(t *testing.T) {
	tests := []struct {
		agent domain.Agent
		scope domain.Scope
		want  string
	}{
		{domain.AgentClaude, domain.ScopePersonal, filepath.Join("/users/test", ".claude", "skills", "review")},
		{domain.AgentCodex, domain.ScopeGlobal, filepath.Join("/users/test", ".agents", "skills", "review")},
		{domain.AgentClaude, domain.ScopeProject, filepath.Join("/repo", ".claude", "skills", "review")},
		{domain.AgentCodex, domain.ScopeProject, filepath.Join("/repo", ".agents", "skills", "review")},
	}
	for _, test := range tests {
		got, err := Target(test.agent, test.scope, "/users/test", "/repo", "review")
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Fatalf("Target(%s, %s) = %s, want %s", test.agent, test.scope, got, test.want)
		}
	}
}
