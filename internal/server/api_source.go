package server

import (
	"net/http"

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
	var result struct {
		Source domain.Source  `json:"source"`
		Skills []domain.Skill `json:"skills"`
	}
	err := s.withLock(func() error {
		var err error
		result.Source, result.Skills, err = gitSource.NewGitManager(s.store, catalog.New(s.store)).Add(domain.Source{
			Name: body.Name, URL: body.URL, Ref: body.Ref, Paths: body.Paths, Tags: body.Tags,
		})
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

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	var result struct {
		Sources []any `json:"sources"`
		Skills  []any `json:"skills"`
		Plan    any   `json:"plan"`
	}
	err := s.withLock(func() error {
		manager := gitSource.NewGitManager(s.store, catalog.New(s.store))
		sources, skills, err := manager.Update(nil)
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
			return err
		}
		if err := engine.Apply(plan, &state); err != nil {
			return err
		}
		result.Plan = plan
		return nil
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
