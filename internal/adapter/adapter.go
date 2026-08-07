package adapter

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zzzzzyijie/skm/internal/domain"
)

func Target(agent domain.Agent, placement domain.Placement, userHome, projectRoot, skillName string, customRoots ...map[domain.Agent]string) (string, error) {
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
			if len(customRoots) == 0 || customRoots[0][agent] == "" {
				return "", fmt.Errorf("unsupported agent %q", agent)
			}
			var err error
			base, err = customUserRoot(customRoots[0][agent], userHome)
			if err != nil {
				return "", err
			}
		}
		switch agent {
		case domain.AgentClaude:
			base = filepath.Join(userHome, ".claude", "skills")
		case domain.AgentCodex:
			base = filepath.Join(userHome, ".codex", "skills")
		case domain.AgentCursor:
			base = filepath.Join(userHome, ".cursor", "skills")
		case domain.AgentCopilot:
			base = filepath.Join(userHome, ".copilot", "skills")
		case domain.AgentGemini:
			base = filepath.Join(userHome, ".gemini", "skills")
		case domain.AgentWindsurf:
			base = filepath.Join(userHome, ".codeium", "windsurf", "skills")
		case domain.AgentKiro:
			base = filepath.Join(userHome, ".kiro", "skills")
		case domain.AgentCline:
			base = filepath.Join(userHome, ".agents", "skills")
		case domain.AgentOpenCode:
			base = filepath.Join(userHome, ".config", "opencode", "skills")
		case domain.AgentTrae:
			base = filepath.Join(userHome, ".trae", "skills")
		case domain.AgentHermes:
			base = filepath.Join(userHome, ".hermes", "skills")
		case domain.AgentOpenClaw:
			base = filepath.Join(userHome, ".openclaw", "skills")
		}
	}
	return filepath.Join(base, skillName), nil
}

func CustomRoots(definitions []domain.AgentDefinition) map[domain.Agent]string {
	result := make(map[domain.Agent]string, len(definitions))
	for _, definition := range definitions {
		result[definition.ID] = definition.SkillsPath
	}
	return result
}

func customUserRoot(configuredPath, userHome string) (string, error) {
	if !strings.HasPrefix(configuredPath, "~/") {
		return "", fmt.Errorf("custom agent path must start with ~/")
	}
	relative := strings.TrimPrefix(configuredPath, "~/")
	if relative == "" || filepath.Clean(relative) == "." || strings.HasPrefix(filepath.Clean(relative), "..") {
		return "", fmt.Errorf("invalid custom agent path %q", configuredPath)
	}
	return filepath.Join(userHome, filepath.FromSlash(relative)), nil
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
	case domain.AgentCopilot:
		return "GitHub Copilot"
	case domain.AgentGemini:
		return "Gemini CLI"
	case domain.AgentKiro:
		return "Kiro"
	case domain.AgentOpenCode:
		return "OpenCode"
	case domain.AgentHermes:
		return "Hermes"
	case domain.AgentOpenClaw:
		return "OpenClaw"
	default:
		name := string(agent)
		return strings.ToUpper(name[:1]) + name[1:]
	}
}
