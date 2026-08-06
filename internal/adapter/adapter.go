package adapter

import (
	"fmt"
	"path/filepath"

	"github.com/zzzzzyijie/skm/internal/domain"
)

func Target(agent domain.Agent, placement domain.Placement, userHome, projectRoot, skillName string) (string, error) {
	if !placement.Valid() {
		return "", fmt.Errorf("invalid placement %q", placement)
	}
	var base string
	if placement == domain.PlacementProject {
		if projectRoot == "" {
			return "", fmt.Errorf("project root is required for project deployment")
		}
		if !agent.ProjectValid() {
			return "", fmt.Errorf("unsupported project agent %q", agent)
		}
		switch agent {
		case domain.AgentClaude:
			base = filepath.Join(projectRoot, ".claude", "skills")
		case domain.AgentCodex:
			base = filepath.Join(projectRoot, ".codex", "skills")
		default:
			base = filepath.Join(projectRoot, "."+string(agent), "skills")
		}
	} else {
		if !agent.Valid() {
			return "", fmt.Errorf("unsupported agent %q", agent)
		}
		switch agent {
		case domain.AgentClaude:
			base = filepath.Join(userHome, ".claude", "skills")
		case domain.AgentCodex:
			base = filepath.Join(userHome, ".codex", "skills")
		}
	}
	return filepath.Join(base, skillName), nil
}

// LegacyTarget returns the pre-.codex target for Codex deployments. It is
// used only to safely clean up deployments created before the path change.
func LegacyTarget(agent domain.Agent, placement domain.Placement, userHome, projectRoot, skillName string) (string, error) {
	if !agent.Valid() {
		return "", fmt.Errorf("unsupported agent %q", agent)
	}
	if !placement.Valid() {
		return "", fmt.Errorf("invalid placement %q", placement)
	}
	if agent != domain.AgentCodex {
		return "", nil
	}
	if placement == domain.PlacementProject {
		if projectRoot == "" {
			return "", fmt.Errorf("project root is required for project deployment")
		}
		return filepath.Join(projectRoot, ".agents", "skills", skillName), nil
	}
	return filepath.Join(userHome, ".agents", "skills", skillName), nil
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
