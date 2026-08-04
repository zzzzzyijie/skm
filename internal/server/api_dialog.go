package server

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func (s *Server) handleChooseSkillDirectory(w http.ResponseWriter, r *http.Request) {
	s.chooseDirectory(w, "Select a Skill directory")
}

func (s *Server) handleChooseProjectDirectory(w http.ResponseWriter, r *http.Request) {
	s.chooseDirectory(w, "Select a project folder")
}

func (s *Server) chooseDirectory(w http.ResponseWriter, prompt string) {
	if runtime.GOOS != "darwin" {
		writeError(w, http.StatusNotImplemented, fmt.Errorf("native directory selection is currently available on macOS only"))
		return
	}

	script := fmt.Sprintf(`POSIX path of (choose folder with prompt %q)`, prompt)
	output, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("directory selection was cancelled"))
		return
	}
	path := strings.TrimSpace(string(output))
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		writeError(w, http.StatusBadRequest, fmt.Errorf("selected path is not a directory"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": path})
}
