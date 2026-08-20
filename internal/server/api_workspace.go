package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/planner"
	"github.com/zzzzzyijie/skm/internal/workspace"
)

type workspaceView struct {
	Configured bool                    `json:"configured"`
	Config     *domain.WorkspaceConfig `json:"config,omitempty"`
	State      *domain.WorkspaceState  `json:"state,omitempty"`
}

func (s *Server) handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	config, err := s.store.LoadWorkspaceConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if strings.TrimSpace(config.URL) == "" {
		writeJSON(w, http.StatusOK, workspaceView{})
		return
	}
	state, err := s.store.LoadWorkspaceState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, workspaceView{Configured: true, Config: &config, State: &state})
}

func (s *Server) handleConfigureWorkspace(w http.ResponseWriter, r *http.Request) {
	var body domain.WorkspaceConfig
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var value domain.WorkspaceConfig
	err := s.withLock(func() error {
		var configureErr error
		value, configureErr = workspace.New(s.store).Configure(body)
		return configureErr
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, workspaceView{Configured: true, Config: &value})
}

func (s *Server) handlePreviewWorkspace(w http.ResponseWriter, r *http.Request) {
	var value workspace.Preview
	err := s.withLock(func() error {
		var previewErr error
		value, previewErr = workspace.New(s.store).Preview()
		return previewErr
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) handleWorkspaceSync(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Resolutions map[string]string `json:"resolutions"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var result workspace.Result
	err := s.withLock(func() error {
		var syncErr error
		result, syncErr = workspace.New(s.store).ApplyResolved(body.Resolutions)
		if syncErr != nil {
			return syncErr
		}
		state, loadErr := s.store.LoadState()
		if loadErr != nil {
			result.DeploymentError = loadErr.Error()
			return nil
		}
		skills, loadErr := s.store.LoadAllSkills()
		if loadErr != nil {
			result.DeploymentError = loadErr.Error()
			return nil
		}
		engine := planner.New(s.store)
		plan, buildErr := engine.Build(skills, state)
		if buildErr != nil {
			result.DeploymentError = buildErr.Error()
			return nil
		}
		result.Plan = &plan
		if applyErr := engine.Apply(plan, &state); applyErr != nil {
			result.DeploymentError = applyErr.Error()
		}
		return nil
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, workspace.ErrConflict) {
			status = http.StatusConflict
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
