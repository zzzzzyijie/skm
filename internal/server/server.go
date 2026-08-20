package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/zzzzzyijie/skm/internal/store"
	"github.com/zzzzzyijie/skm/web"
)

var Version = "dev"

type Server struct {
	store *store.Store
	mu    sync.Mutex
}

func New(s *store.Store) *Server {
	return &Server{store: s}
}

// Handler returns the complete UI and API handler. Keeping route construction
// separate from Run makes the local server straightforward to exercise in tests.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("GET /api/dashboard", s.handleDashboard)
	mux.HandleFunc("GET /api/version", s.handleVersion)
	mux.HandleFunc("GET /api/agents", s.handleListAgents)
	mux.HandleFunc("PUT /api/agents", s.handleUpdateAgents)
	mux.HandleFunc("POST /api/agents/custom", s.handleSaveCustomAgent)
	mux.HandleFunc("DELETE /api/agents/{id}", s.handleDeleteCustomAgent)
	mux.HandleFunc("GET /api/skills", s.handleListSkills)
	mux.HandleFunc("GET /api/skills/{id...}", s.handleShowSkill)
	mux.HandleFunc("POST /api/skills/detach", s.handleDetachSkill)
	mux.HandleFunc("POST /api/skills", s.handleAddSkill)
	mux.HandleFunc("POST /api/dialogs/skill-directory", s.handleChooseSkillDirectory)
	mux.HandleFunc("POST /api/dialogs/project-directory", s.handleChooseProjectDirectory)
	mux.HandleFunc("DELETE /api/skills/{id...}", s.handleRemoveSkill)
	mux.HandleFunc("GET /api/prompts", s.handleListPrompts)
	mux.HandleFunc("POST /api/prompts", s.handleCreatePrompt)
	mux.HandleFunc("POST /api/prompts/validate", s.handleValidatePrompt)
	mux.HandleFunc("POST /api/prompt-render", s.handleRenderPrompt)
	mux.HandleFunc("GET /api/prompts/{id...}", s.handleShowPrompt)
	mux.HandleFunc("PUT /api/prompts/{id...}", s.handleUpdatePrompt)
	mux.HandleFunc("DELETE /api/prompts/{id...}", s.handleRemovePrompt)
	mux.HandleFunc("GET /api/tags", s.handleListTags)
	mux.HandleFunc("POST /api/tags", s.handleCreateTag)
	mux.HandleFunc("DELETE /api/tags/{name}", s.handleDeleteTag)
	mux.HandleFunc("PUT /api/skill-tags", s.handleReplaceSkillTags)
	mux.HandleFunc("POST /api/skill-tags/add", s.handleAddTags)
	mux.HandleFunc("POST /api/skill-tags/remove", s.handleRemoveTag)
	mux.HandleFunc("POST /api/tags/rename", s.handleRenameTag)
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("POST /api/enable", s.handleEnable)
	mux.HandleFunc("POST /api/disable", s.handleDisable)
	mux.HandleFunc("GET /api/plan", s.handlePlan)
	mux.HandleFunc("POST /api/apply", s.handleApply)
	mux.HandleFunc("GET /api/sources", s.handleListSources)
	mux.HandleFunc("POST /api/sources", s.handleAddSource)
	mux.HandleFunc("POST /api/sources/{name}/update", s.handleUpdateSource)
	mux.HandleFunc("DELETE /api/sources/{name}", s.handleDeleteSource)
	mux.HandleFunc("POST /api/sync", s.handleSync)
	mux.HandleFunc("GET /api/workspace", s.handleGetWorkspace)
	mux.HandleFunc("PUT /api/workspace", s.handleConfigureWorkspace)
	mux.HandleFunc("GET /api/workspace/preview", s.handlePreviewWorkspace)
	mux.HandleFunc("POST /api/workspace/sync", s.handleWorkspaceSync)
	mux.HandleFunc("GET /api/doctor", s.handleDoctor)
	mux.HandleFunc("GET /api/projects", s.handleListProjects)
	mux.HandleFunc("POST /api/projects", s.handleAddProject)
	mux.HandleFunc("GET /api/projects/{id}/status", s.handleProjectStatus)
	mux.HandleFunc("GET /api/projects/{id}/skills/{skill}", s.handleShowProjectSkill)
	mux.HandleFunc("POST /api/projects/{id}/skills/{skill}/migrate", s.handleMigrateProjectSkill)
	mux.HandleFunc("DELETE /api/projects/{id}/skills/{skill}", s.handleRemoveProjectSkill)
	mux.HandleFunc("GET /api/projects/{id}", s.handleShowProject)
	mux.HandleFunc("POST /api/projects/{id}/link", s.handleProjectLink)
	mux.HandleFunc("POST /api/projects/{id}/copy", s.handleProjectCopy)
	mux.HandleFunc("POST /api/projects/{id}/unlink", s.handleProjectUnlink)
	mux.HandleFunc("DELETE /api/projects/{id}", s.handleUnregisterProject)

	// Static files
	fileServer := http.FileServer(http.FS(web.FS))
	mux.Handle("GET /", fileServer)
	return withSecurityHeaders(mux)
}

func (s *Server) Run(port int, openBrowser bool) error {
	handler := s.Handler()

	addr := fmt.Sprintf("localhost:%d", port)

	fmt.Printf("SKM UI running at http://%s\n", addr)
	if openBrowser {
		go openURL(fmt.Sprintf("http://%s", addr))
	}
	return http.ListenAndServe(addr, handler)
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; img-src 'self' data:; connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func readJSON(r *http.Request, target any) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func (s *Server) withLock(fn func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	unlock, err := s.store.Lock()
	if err != nil {
		return err
	}
	defer func() {
		if unlockErr := unlock(); unlockErr != nil {
			log.Printf("unlock error: %v", unlockErr)
		}
	}()
	return fn()
}

func openURL(url string) {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	default:
		return
	}
	_ = exec.Command(cmd, url).Start()
}

func splitSkillID(r *http.Request) string {
	id := r.PathValue("id")
	return strings.TrimPrefix(id, "/")
}
