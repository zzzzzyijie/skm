package server

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/zzzzzyijie/skm/internal/adapter"
	"github.com/zzzzzyijie/skm/internal/buildinfo"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/fsx"
)

type dashboardData struct {
	SkillCount     int            `json:"skillCount"`
	ActivatedCount int            `json:"activatedCount"`
	SourceCount    int            `json:"sourceCount"`
	RecentSkills   []domain.Skill `json:"recentSkills"`
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	catalog, err := s.store.LoadCatalog()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	state, err := s.store.LoadState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	sources, err := s.store.LoadSources()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	activatedIDs := make(map[string]struct{})
	for _, a := range state.Activations {
		if a.Placement == domain.PlacementUser {
			activatedIDs[a.SkillID] = struct{}{}
		}
	}

	skills := catalog.Skills
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].AddedAt.After(skills[j].AddedAt)
	})
	recent := skills
	if len(recent) > 5 {
		recent = recent[:5]
	}

	data := dashboardData{
		SkillCount:     len(catalog.Skills),
		ActivatedCount: len(activatedIDs),
		SourceCount:    len(sources.Sources),
		RecentSkills:   recent,
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	version := Version
	if version == "" || version == "dev" {
		version = buildinfo.Current()
	}
	writeJSON(w, http.StatusOK, map[string]string{"version": version})
}

type doctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (s *Server) handleDoctor(w http.ResponseWriter, r *http.Request) {
	checks := []doctorCheck{{Name: "skm-home", Status: "ok", Message: s.store.Paths.Home}}

	sources, err := s.store.LoadSources()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if path, err := exec.LookPath("git"); err != nil {
		status := "optional"
		if len(sources.Sources) > 0 {
			status = "error"
		}
		checks = append(checks, doctorCheck{Name: "git", Status: status, Message: "required only for configured Git sources"})
	} else {
		checks = append(checks, doctorCheck{Name: "git", Status: "ok", Message: path})
	}

	for _, agentName := range []domain.Agent{domain.AgentClaude, domain.AgentCodex} {
		target, _ := adapter.Target(agentName, domain.PlacementUser, s.store.Paths.UserHome, s.store.Paths.ProjectRoot, "probe")
		directory := filepath.Dir(target)
		status := "not-created"
		if _, err := os.Stat(directory); err == nil {
			status = "ok"
		}
		checks = append(checks, doctorCheck{Name: string(agentName), Status: status, Message: directory})
	}

	values, err := s.store.LoadAllSkills()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	for _, value := range values {
		hash, hashErr := fsx.HashDir(value.Path)
		if hashErr != nil || hash != value.Hash {
			checks = append(checks, doctorCheck{Name: value.ID, Status: "error", Message: "Skill content is missing or modified"})
		}
	}

	writeJSON(w, http.StatusOK, checks)
}
