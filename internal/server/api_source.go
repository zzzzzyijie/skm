package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/planner"
	gitSource "github.com/zzzzzyijie/skm/internal/source"
)

func (s *Server) handleListSources(w http.ResponseWriter, r *http.Request) {
	sources, err := s.store.LoadSources()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, sources.Sources)
}

func (s *Server) handleAddSource(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Input string   `json:"input"`
		Name  string   `json:"name"`
		URL   string   `json:"url"`
		Ref   string   `json:"ref"`
		Paths []string `json:"paths"`
		Tags  []string `json:"tags"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rawInput := strings.TrimSpace(body.Input)
	if rawInput == "" {
		rawInput = strings.TrimSpace(body.URL)
	}
	parsed, err := gitSource.ParseInstallInput(rawInput)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(parsed.RequestedSkills) > 0 && len(body.Paths) > 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("paths cannot be combined with --skill selections"))
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = parsed.SuggestedName
	}
	var result struct {
		Source domain.Source  `json:"source"`
		Skills []domain.Skill `json:"skills"`
	}
	err = s.withLock(func() error {
		var err error
		result.Source, result.Skills, err = gitSource.NewGitManager(s.store, catalog.New(s.store)).AddSelected(
			domain.Source{Name: name, URL: parsed.URL, Ref: body.Ref, Paths: body.Paths, Tags: body.Tags},
			parsed.RequestedSkills,
		)
		return err
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) handleUpdateSource(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var result struct {
		Sources []any `json:"sources"`
		Skills  []any `json:"skills"`
	}
	err := s.withLock(func() error {
		manager := gitSource.NewGitManager(s.store, catalog.New(s.store))
		sources, skills, err := manager.Update([]string{name})
		if err != nil {
			return err
		}
		result.Sources = make([]any, len(sources))
		for i, v := range sources {
			result.Sources[i] = v
		}
		result.Skills = make([]any, len(skills))
		for i, v := range skills {
			result.Skills[i] = v
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleDeleteSource(w http.ResponseWriter, r *http.Request) {
	var result gitSource.RemovalResult
	err := s.withLock(func() error {
		var err error
		result, err = gitSource.NewGitManager(s.store, catalog.New(s.store)).Remove(r.PathValue("name"))
		return err
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type sourceSyncItem struct {
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	Source     *domain.Source `json:"source,omitempty"`
	SkillCount int            `json:"skillCount"`
	Error      string         `json:"error,omitempty"`
}

type sourceSyncResult struct {
	Configured      int              `json:"configured"`
	Updated         int              `json:"updated"`
	Failed          int              `json:"failed"`
	SkillCount      int              `json:"skillCount"`
	Results         []sourceSyncItem `json:"results"`
	Plan            *domain.Plan     `json:"plan,omitempty"`
	Applied         bool             `json:"applied"`
	DeploymentError string           `json:"deploymentError,omitempty"`
	SyncedAt        time.Time        `json:"syncedAt"`
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	var result sourceSyncResult
	err := s.withLock(func() error {
		configured, err := s.store.LoadSources()
		if err != nil {
			return err
		}
		result.Configured = len(configured.Sources)
		result.SyncedAt = time.Now().UTC()
		if result.Configured == 0 {
			return nil
		}

		manager := gitSource.NewGitManager(s.store, catalog.New(s.store))
		for _, configuredSource := range configured.Sources {
			updated, skills, updateErr := manager.Update([]string{configuredSource.Name})
			item := sourceSyncItem{Name: configuredSource.Name}
			if updateErr != nil {
				item.Status = "error"
				item.Error = updateErr.Error()
				result.Failed++
				result.Results = append(result.Results, item)
				continue
			}
			item.Status = "updated"
			item.SkillCount = len(skills)
			if len(updated) > 0 {
				item.Source = &updated[0]
			}
			result.Updated++
			result.SkillCount += len(skills)
			result.Results = append(result.Results, item)
		}
		if result.Updated == 0 {
			return nil
		}

		state, err := s.store.LoadState()
		if err != nil {
			return err
		}
		allSkills, err := s.store.LoadAllSkills()
		if err != nil {
			return err
		}
		engine := planner.New(s.store)
		plan, err := engine.Build(allSkills, state)
		if err != nil {
			result.DeploymentError = err.Error()
			return nil
		}
		result.Plan = &plan
		if applyErr := engine.Apply(plan, &state); applyErr != nil {
			result.DeploymentError = applyErr.Error()
			return nil
		}
		result.Applied = true
		return nil
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if result.Configured == 0 {
		writeError(w, http.StatusConflict, fmt.Errorf("no Git sources configured"))
		return
	}
	writeJSON(w, http.StatusOK, result)
}
