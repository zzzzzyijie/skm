package adapter

import (
	"fmt"
	"path/filepath"

	"github.com/zzzzzyijie/skm/internal/domain"
)

func Target(agent domain.Agent, placement domain.Placement, userHome, projectRoot, skillName string) (string, error) {
	if !agent.Valid() {
		return "", fmt.Errorf("unsupported agent %q", agent)
	}
	if !placement.Valid() {
		return "", fmt.Errorf("invalid placement %q", placement)
	}
	var base string
	if placement == domain.PlacementProject {
		if projectRoot == "" {
			return "", fmt.Errorf("project root is required for project deployment")
		}
		switch agent {
		case domain.AgentClaude:
			base = filepath.Join(projectRoot, ".claude", "skills")
		case domain.AgentCodex:
			base = filepath.Join(projectRoot, ".agents", "skills")
		}
	} else {
		switch agent {
		case domain.AgentClaude:
			base = filepath.Join(userHome, ".claude", "skills")
		case domain.AgentCodex:
			base = filepath.Join(userHome, ".agents", "skills")
		}
	}
	return filepath.Join(base, skillName), nil
}

func DisplayName(agent domain.Agent) string {
	switch agent {
	case domain.AgentClaude:
		return "Claude Code"
	case domain.AgentCodex:
		return "Codex"
	default:
		return string(agent)
	}
}
