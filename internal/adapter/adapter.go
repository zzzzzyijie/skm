package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zzzzzyijie/skm/internal/domain"
)

func Target(agent domain.Agent, placement domain.Placement, userHome, projectRoot, skillName string) (string, error) {
	for _, definition := range domain.DefaultAgentDirectories().Agents {
		if definition.ID == agent {
			return TargetDirectory(definition, placement, userHome, projectRoot, skillName)
		}
	}
	return "", fmt.Errorf("unsupported agent %q", agent)
}

func TargetDirectory(definition domain.AgentDirectory, placement domain.Placement, userHome, projectRoot, skillName string) (string, error) {
	if !definition.ID.Valid() {
		return "", fmt.Errorf("unsupported agent %q", definition.ID)
	}
	if !placement.Valid() {
		return "", fmt.Errorf("invalid placement %q", placement)
	}
	base := definition.UserPath
	if placement == domain.PlacementProject {
		if projectRoot == "" {
			return "", fmt.Errorf("project root is required for project deployment")
		}
		base = definition.ProjectPath
	}
	base, err := ResolvePath(base, userHome, projectRoot, placement)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, skillName), nil
}

func ResolvePath(raw, userHome, projectRoot string, placement domain.Placement) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("agent path is required")
	}
	if strings.HasPrefix(raw, "~") {
		raw = filepath.Join(userHome, strings.TrimPrefix(raw, "~/"))
	}
	configHome := os.Getenv("XDG_CONFIG")
	if configHome == "" {
		configHome = os.Getenv("XDG_CONFIG_HOME")
	}
	if configHome == "" {
		configHome = filepath.Join(userHome, ".config")
	}
	raw = strings.ReplaceAll(raw, "$XDG_CONFIG", configHome)
	raw = strings.ReplaceAll(raw, "${XDG_CONFIG}", configHome)
	raw = strings.ReplaceAll(raw, "$HOME", userHome)
	raw = strings.ReplaceAll(raw, "${HOME}", userHome)
	if placement == domain.PlacementProject && !filepath.IsAbs(raw) {
		raw = filepath.Join(projectRoot, raw)
	}
	return filepath.Clean(raw), nil
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
