package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/planner"
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	plan, err := s.buildCurrentPlan()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *Server) handlePlan(w http.ResponseWriter, r *http.Request) {
	plan, err := s.buildCurrentPlan()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *Server) handleEnable(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Skills []string `json:"skills"`
		Agents []string `json:"agents"`
		Mode   string   `json:"mode"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(body.Skills) == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("at least one skill is required"))
		return
	}
	mode := domain.LinkMode(body.Mode)
	if mode == "" {
		mode = domain.ModeAuto
	}
	if !mode.Valid() {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid mode %q", body.Mode))
		return
	}
	agents, err := parseAgents(body.Agents)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var plan domain.Plan
	err = s.withLock(func() error {
		manager := catalog.New(s.store)
		var selected []domain.Skill
		for _, query := range body.Skills {
			value, resolveErr := manager.ResolveLibrary(query)
			if resolveErr != nil {
				return resolveErr
			}
			selected = append(selected, value)
		}
		state, stateErr := s.store.LoadState()
		if stateErr != nil {
			return stateErr
		}
		engine := planner.New(s.store)
		engine.AddActivations(&state, selected, domain.PlacementUser, "", agents, mode)
		skills, skillsErr := s.store.LoadAllSkills()
		if skillsErr != nil {
			return skillsErr
		}
		plan, err = engine.Build(skills, state)
		if err != nil {
			return err
		}
		return engine.Apply(plan, &state)
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *Server) handleDisable(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Skills []string `json:"skills"`
		Agents []string `json:"agents"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(body.Skills) == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("at least one skill is required"))
		return
	}
	err := s.withLock(func() error {
		ids := make(map[string]struct{}, len(body.Skills))
		manager := catalog.New(s.store)
		for _, query := range body.Skills {
			value, resolveErr := manager.ResolveLibrary(query)
			if resolveErr != nil {
				return resolveErr
			}
			ids[value.ID] = struct{}{}
		}
		agentMap := make(map[domain.Agent]struct{})
		if len(body.Agents) > 0 {
			parsed, agentErr := parseAgents(body.Agents)
			if agentErr != nil {
				return agentErr
			}
			for _, a := range parsed {
				agentMap[a] = struct{}{}
			}
		}
		state, err := s.store.LoadState()
		if err != nil {
			return err
		}
		return planner.New(s.store).Disable(&state, ids, domain.PlacementUser, "", agentMap, false)
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleApply(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Digest string `json:"digest"`
	}
	_ = readJSON(r, &body) // digest is optional

	var plan domain.Plan
	err := s.withLock(func() error {
		state, err := s.store.LoadState()
		if err != nil {
			return err
		}
		skills, err := s.store.LoadAllSkills()
		if err != nil {
			return err
		}
		engine := planner.New(s.store)
		plan, err = engine.Build(skills, state)
		if err != nil {
			return err
		}
		if body.Digest != "" && body.Digest != plan.Digest {
			return fmt.Errorf("plan digest changed: expected %s, got %s", body.Digest, plan.Digest)
		}
		return engine.Apply(plan, &state)
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *Server) buildCurrentPlan() (domain.Plan, error) {
	state, err := s.store.LoadState()
	if err != nil {
		return domain.Plan{}, err
	}
	skills, err := s.store.LoadAllSkills()
	if err != nil {
		return domain.Plan{}, err
	}
	return planner.New(s.store).Build(skills, state)
}

func parseAgents(values []string) ([]domain.Agent, error) {
	if len(values) == 0 {
		return []domain.Agent{domain.AgentClaude, domain.AgentCodex}, nil
	}
	var result []domain.Agent
	seen := make(map[domain.Agent]struct{})
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			a := domain.Agent(strings.ToLower(strings.TrimSpace(part)))
			if !a.Valid() {
				return nil, fmt.Errorf("unsupported agent %q: use claude or codex", part)
			}
			if _, ok := seen[a]; ok {
				continue
			}
			seen[a] = struct{}{}
			result = append(result, a)
		}
	}
	return result, nil
}
