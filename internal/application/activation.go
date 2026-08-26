package application

import (
	"fmt"
	"strings"

	"github.com/zzzzzyijie/skm/internal/adapter"
	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/planner"
)

type ActivationInput struct {
	Skills []string `json:"skills"`
	Agents []string `json:"agents"`
	Mode   string   `json:"mode,omitempty"`
}

func (s *Service) ActivationStatus() (domain.Plan, error) {
	state, err := s.Store.LoadState()
	if err != nil {
		return domain.Plan{}, err
	}
	skills, err := s.Store.LoadAllSkills()
	if err != nil {
		return domain.Plan{}, err
	}
	plan, err := planner.New(s.Store).Build(skills, state)
	if plan.Operations == nil {
		plan.Operations = []domain.Operation{}
	}
	return plan, err
}

func (s *Service) Enable(input ActivationInput) (domain.Plan, error) {
	if len(input.Skills) == 0 {
		return domain.Plan{}, fmt.Errorf("at least one skill is required")
	}
	mode := domain.LinkMode(input.Mode)
	if mode == "" {
		mode = domain.ModeAuto
	}
	if !mode.Valid() {
		return domain.Plan{}, fmt.Errorf("invalid mode %q", input.Mode)
	}
	config, err := s.Store.LoadConfig()
	if err != nil {
		return domain.Plan{}, err
	}
	agents, err := parseAgents(input.Agents, customAgentSet(config.Agents))
	if err != nil {
		return domain.Plan{}, err
	}
	availableAgents := configuredAgents(config.Defaults.Agents, config.Agents)
	if len(input.Agents) == 0 {
		agents = availableAgents
	}
	if len(agents) == 0 {
		return domain.Plan{}, fmt.Errorf("at least one managed agent is required")
	}
	configured := agentSet(availableAgents)
	for _, agent := range agents {
		if !configured[agent] {
			return domain.Plan{}, fmt.Errorf("agent %s has not been added", adapter.DisplayName(agent))
		}
	}
	var plan domain.Plan
	err = s.withLock(func() error {
		manager := catalog.New(s.Store)
		selected := make([]domain.Skill, 0, len(input.Skills))
		for _, query := range input.Skills {
			value, resolveErr := manager.ResolveLibrary(query)
			if resolveErr != nil {
				return resolveErr
			}
			selected = append(selected, value)
		}
		state, stateErr := s.Store.LoadState()
		if stateErr != nil {
			return stateErr
		}
		engine := planner.New(s.Store)
		engine.AddActivations(&state, selected, domain.PlacementUser, "", agents, mode)
		skills, loadErr := s.Store.LoadAllSkills()
		if loadErr != nil {
			return loadErr
		}
		plan, err = engine.BuildScoped(skills, state, domain.PlacementUser, "")
		if err != nil {
			return err
		}
		return engine.Apply(plan, &state)
	})
	if plan.Operations == nil {
		plan.Operations = []domain.Operation{}
	}
	return plan, err
}

func (s *Service) Disable(input ActivationInput) (map[string]string, error) {
	if len(input.Skills) == 0 {
		return nil, fmt.Errorf("at least one skill is required")
	}
	err := s.withLock(func() error {
		ids := make(map[string]struct{}, len(input.Skills))
		manager := catalog.New(s.Store)
		for _, query := range input.Skills {
			value, resolveErr := manager.ResolveLibrary(query)
			if resolveErr != nil {
				return resolveErr
			}
			ids[value.ID] = struct{}{}
		}
		agentMap := make(map[domain.Agent]struct{})
		if len(input.Agents) > 0 {
			config, configErr := s.Store.LoadConfig()
			if configErr != nil {
				return configErr
			}
			parsed, parseErr := parseAgents(input.Agents, customAgentSet(config.Agents))
			if parseErr != nil {
				return parseErr
			}
			for _, agent := range parsed {
				agentMap[agent] = struct{}{}
			}
		}
		state, stateErr := s.Store.LoadState()
		if stateErr != nil {
			return stateErr
		}
		return planner.New(s.Store).Disable(&state, ids, domain.PlacementUser, "", agentMap, false)
	})
	return map[string]string{"status": "ok"}, err
}

func parseAgents(values []string, custom map[domain.Agent]bool) ([]domain.Agent, error) {
	result := make([]domain.Agent, 0, len(values))
	seen := make(map[domain.Agent]struct{})
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			agent := domain.Agent(strings.ToLower(strings.TrimSpace(part)))
			if !agent.Valid() && !custom[agent] {
				return nil, fmt.Errorf("unsupported agent %q", part)
			}
			if _, ok := seen[agent]; ok {
				continue
			}
			seen[agent] = struct{}{}
			result = append(result, agent)
		}
	}
	return result, nil
}
