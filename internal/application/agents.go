package application

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/zzzzzyijie/skm/internal/adapter"
	"github.com/zzzzzyijie/skm/internal/domain"
)

type AgentDescriptor struct {
	ID         domain.Agent `json:"id"`
	Name       string       `json:"name"`
	Path       string       `json:"path,omitempty"`
	Format     string       `json:"format,omitempty"`
	Configured bool         `json:"configured"`
	Detected   bool         `json:"detected"`
	Supported  bool         `json:"supported"`
	Note       string       `json:"note,omitempty"`
	Icon       string       `json:"icon,omitempty"`
	Custom     bool         `json:"custom"`
}

type ConfigureAgentsInput struct {
	Agents []string `json:"agents"`
}

type CustomAgentInput struct {
	ID         domain.Agent `json:"id"`
	Name       string       `json:"name"`
	SkillsPath string       `json:"skillsPath"`
	Icon       string       `json:"icon,omitempty"`
}

var managedAgents = []domain.Agent{
	domain.AgentClaude, domain.AgentCodex, domain.AgentCursor, domain.AgentCopilot,
	domain.AgentGemini, domain.AgentWindsurf, domain.AgentKiro, domain.AgentCline,
	domain.AgentOpenCode, domain.AgentTrae, domain.AgentHermes, domain.AgentOpenClaw,
}

var customAgentIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`)

func (s *Service) ListAgents() ([]AgentDescriptor, error) {
	config, err := s.Store.LoadConfig()
	if err != nil {
		return nil, err
	}
	configured := agentSet(configuredAgents(config.Defaults.Agents, config.Agents))
	customRoots := adapter.CustomRoots(config.Agents)
	result := make([]AgentDescriptor, 0, len(managedAgents)+len(config.Agents))
	for _, agent := range managedAgents {
		descriptor := AgentDescriptor{
			ID: agent, Name: adapter.DisplayName(agent), Configured: configured[agent],
			Detected: s.agentDetected(agent, customRoots, false), Supported: true,
		}
		target, _ := adapter.Target(agent, domain.PlacementUser, "~", "", "<skill-name>")
		descriptor.Path = target
		descriptor.Format = "<skill-name>/SKILL.md"
		result = append(result, descriptor)
	}
	for _, definition := range config.Agents {
		result = append(result, AgentDescriptor{
			ID: definition.ID, Name: definition.Name, Path: definition.SkillsPath + "/<skill-name>",
			Format: "<skill-name>/SKILL.md", Configured: configured[definition.ID], Supported: true,
			Detected: s.agentDetected(definition.ID, customRoots, true), Icon: definition.Icon, Custom: true,
		})
	}
	return result, nil
}

func (s *Service) ConfigureAgents(values []string) ([]AgentDescriptor, error) {
	err := s.withLock(func() error {
		config, err := s.Store.LoadConfig()
		if err != nil {
			return err
		}
		custom := customAgentSet(config.Agents)
		selected := make(map[domain.Agent]bool)
		for _, raw := range values {
			agent := domain.Agent(raw)
			if !agent.Valid() && !custom[agent] {
				return fmt.Errorf("unsupported agent %q", raw)
			}
			selected[agent] = true
		}
		state, err := s.Store.LoadState()
		if err != nil {
			return err
		}
		for _, activation := range state.Activations {
			if activation.Placement != domain.PlacementUser {
				continue
			}
			for _, agent := range activation.Agents {
				if !selected[agent] {
					return fmt.Errorf("agent %s is still enabled for %s; disable it first", adapter.DisplayName(agent), activation.SkillID)
				}
			}
		}
		config.Defaults.Agents = orderedSelectedAgents(selected, config.Agents)
		return s.Store.SaveConfig(config)
	})
	if err != nil {
		return nil, err
	}
	return s.ListAgents()
}

func (s *Service) SaveCustomAgent(input CustomAgentInput) ([]AgentDescriptor, error) {
	definition := domain.AgentDefinition{
		ID:   domain.Agent(strings.ToLower(strings.TrimSpace(string(input.ID)))),
		Name: strings.TrimSpace(input.Name), SkillsPath: strings.TrimSuffix(strings.TrimSpace(input.SkillsPath), "/"),
		Icon: input.Icon,
	}
	if err := validateCustomAgent(definition); err != nil {
		return nil, err
	}
	err := s.withLock(func() error {
		config, err := s.Store.LoadConfig()
		if err != nil {
			return err
		}
		for _, builtIn := range managedAgents {
			if definition.ID == builtIn {
				return fmt.Errorf("agent ID %q is reserved", definition.ID)
			}
		}
		for index, existing := range config.Agents {
			if existing.ID != definition.ID {
				continue
			}
			if existing.SkillsPath != definition.SkillsPath {
				active, activeErr := s.agentIsActive(definition.ID)
				if activeErr != nil {
					return activeErr
				}
				if active {
					return fmt.Errorf("disable all Skills for %s before changing its path", existing.Name)
				}
			}
			config.Agents[index] = definition
			config.Defaults.Agents = configuredAgents(config.Defaults.Agents, config.Agents)
			return s.Store.SaveConfig(config)
		}
		config.Agents = append(config.Agents, definition)
		sort.Slice(config.Agents, func(i, j int) bool { return config.Agents[i].ID < config.Agents[j].ID })
		config.Defaults.Agents = configuredAgents(append(config.Defaults.Agents, definition.ID), config.Agents)
		return s.Store.SaveConfig(config)
	})
	if err != nil {
		return nil, err
	}
	return s.ListAgents()
}

func (s *Service) DeleteCustomAgent(id string) (map[string]string, error) {
	agent := domain.Agent(id)
	err := s.withLock(func() error {
		config, err := s.Store.LoadConfig()
		if err != nil {
			return err
		}
		found := false
		kept := config.Agents[:0]
		for _, definition := range config.Agents {
			if definition.ID == agent {
				found = true
				continue
			}
			kept = append(kept, definition)
		}
		if !found {
			return fmt.Errorf("custom agent %q not found", agent)
		}
		active, err := s.agentIsActive(agent)
		if err != nil {
			return err
		}
		if active {
			return fmt.Errorf("disable all Skills for %s before deleting it", agent)
		}
		config.Agents = kept
		config.Defaults.Agents = configuredAgents(config.Defaults.Agents, config.Agents)
		return s.Store.SaveConfig(config)
	})
	return map[string]string{"status": "ok"}, err
}

func (s *Service) agentIsActive(agent domain.Agent) (bool, error) {
	state, err := s.Store.LoadState()
	if err != nil {
		return false, err
	}
	for _, activation := range state.Activations {
		if activation.Placement != domain.PlacementUser {
			continue
		}
		for _, activeAgent := range activation.Agents {
			if activeAgent == agent {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *Service) agentDetected(agent domain.Agent, customRoots map[domain.Agent]string, custom bool) bool {
	target, err := adapter.Target(agent, domain.PlacementUser, s.Store.Paths.UserHome, "", "probe", customRoots)
	if err != nil {
		return false
	}
	skillsRoot := filepath.Dir(target)
	candidates := []string{skillsRoot}
	if !custom {
		candidates = append(candidates, filepath.Dir(skillsRoot))
	}
	for _, candidate := range candidates {
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return true
		}
	}
	return false
}

func validateCustomAgent(definition domain.AgentDefinition) error {
	if !customAgentIDPattern.MatchString(string(definition.ID)) {
		return fmt.Errorf("agent ID must be 2-32 lowercase letters, numbers, or hyphens")
	}
	if definition.Name == "" || len(definition.Name) > 64 {
		return fmt.Errorf("agent name must be 1-64 characters")
	}
	if !strings.HasPrefix(definition.SkillsPath, "~/") || strings.Contains(definition.SkillsPath, "..") || strings.Contains(definition.SkillsPath, "<skill-name>") {
		return fmt.Errorf("Skill path must be a ~/ directory without .. or <skill-name>")
	}
	if definition.Icon == "" {
		return nil
	}
	parts := strings.SplitN(definition.Icon, ",", 2)
	if len(parts) != 2 || !strings.HasSuffix(parts[0], ";base64") || (!strings.HasPrefix(parts[0], "data:image/png") && !strings.HasPrefix(parts[0], "data:image/jpeg") && !strings.HasPrefix(parts[0], "data:image/webp") && !strings.HasPrefix(parts[0], "data:image/svg+xml")) {
		return fmt.Errorf("icon must be a PNG, JPEG, WebP, or SVG data URL")
	}
	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil || len(decoded) > 256*1024 {
		return fmt.Errorf("icon must be valid and no larger than 256 KiB")
	}
	return nil
}

func agentSet(values []domain.Agent) map[domain.Agent]bool {
	result := make(map[domain.Agent]bool, len(values))
	for _, agent := range values {
		result[agent] = true
	}
	return result
}

func customAgentSet(definitions []domain.AgentDefinition) map[domain.Agent]bool {
	result := make(map[domain.Agent]bool, len(definitions))
	for _, definition := range definitions {
		result[definition.ID] = true
	}
	return result
}

func orderedSelectedAgents(selected map[domain.Agent]bool, definitions []domain.AgentDefinition) []domain.Agent {
	result := make([]domain.Agent, 0, len(selected))
	for _, agent := range managedAgents {
		if selected[agent] {
			result = append(result, agent)
		}
	}
	for _, definition := range definitions {
		if selected[definition.ID] {
			result = append(result, definition.ID)
		}
	}
	return result
}

func configuredAgents(values []domain.Agent, definitions []domain.AgentDefinition) []domain.Agent {
	selected := make(map[domain.Agent]bool)
	custom := customAgentSet(definitions)
	for _, agent := range values {
		if agent.Valid() || custom[agent] {
			selected[agent] = true
		}
	}
	return orderedSelectedAgents(selected, definitions)
}
