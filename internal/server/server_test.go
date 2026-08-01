package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/store"
)

func TestHandlerServesEmbeddedUIWithSecurityHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()
	New(testStore(t)).Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	body := recorder.Body.String()
	if recorder.Code != http.StatusOK || !strings.Contains(body, "AI Skill Manager") {
		t.Fatalf("GET / = %d, body=%s", recorder.Code, body)
	}
	if got := recorder.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q", got)
	}
	if got := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'self'") {
		t.Fatalf("Content-Security-Policy = %q", got)
	}
}

func TestLibraryTagAndActivationLifecycle(t *testing.T) {
	handler := New(testStore(t)).Handler()
	requestJSON(t, handler, http.MethodPost, "/api/sources", map[string]any{}, http.StatusBadRequest, nil)
	skillPath := makeSkill(t, "review")

	var created domain.Skill
	requestJSON(t, handler, http.MethodPost, "/api/skills", map[string]any{
		"path": skillPath, "tags": []string{"review"}, "source": "local",
	}, http.StatusCreated, &created)
	if created.ID != "local/review" {
		t.Fatalf("created Skill ID = %q", created.ID)
	}

	requestJSON(t, handler, http.MethodPost, "/api/skill-tags/add", map[string]any{
		"skill": created.ID, "tags": []string{"quality"},
	}, http.StatusOK, &created)
	requestJSON(t, handler, http.MethodPost, "/api/tags/rename", map[string]any{
		"old": "quality", "new": "engineering",
	}, http.StatusOK, nil)

	var tags []tagCount
	requestJSON(t, handler, http.MethodGet, "/api/tags", nil, http.StatusOK, &tags)
	if !hasTag(tags, "engineering") || !hasTag(tags, "review") {
		t.Fatalf("tags = %#v", tags)
	}

	var plan domain.Plan
	requestJSON(t, handler, http.MethodPost, "/api/enable", map[string]any{
		"skills": []string{created.ID}, "agents": []string{"codex"}, "mode": "auto",
	}, http.StatusOK, &plan)
	if len(plan.Operations) != 1 || plan.Operations[0].SkillID != created.ID {
		t.Fatalf("enable plan = %#v", plan)
	}

	requestJSON(t, handler, http.MethodPost, "/api/disable", map[string]any{
		"skills": []string{created.ID}, "agents": []string{"codex"},
	}, http.StatusOK, nil)
	requestJSON(t, handler, http.MethodDelete, "/api/skills/local/review", nil, http.StatusOK, nil)

	var dashboard dashboardData
	requestJSON(t, handler, http.MethodGet, "/api/dashboard", nil, http.StatusOK, &dashboard)
	if dashboard.SkillCount != 0 || dashboard.ActivatedCount != 0 {
		t.Fatalf("dashboard after removal = %#v", dashboard)
	}
}

func TestProjectLifecycleAPI(t *testing.T) {
	storage := testStore(t)
	handler := New(storage).Handler()
	projectPath := filepath.Join(t.TempDir(), "web-project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatal(err)
	}
	linkSkill := makeSkill(t, "web-link")
	copySkill := makeSkill(t, "web-copy")
	requestJSON(t, handler, http.MethodPost, "/api/skills", map[string]any{"path": linkSkill}, http.StatusCreated, nil)
	requestJSON(t, handler, http.MethodPost, "/api/skills", map[string]any{"path": copySkill}, http.StatusCreated, nil)

	var project domain.Project
	requestJSON(t, handler, http.MethodPost, "/api/projects", map[string]any{"path": projectPath}, http.StatusCreated, &project)
	canonicalProjectPath, err := filepath.EvalSymlinks(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	if project.ID != "web-project" || project.Path != canonicalProjectPath {
		t.Fatalf("project default name = %#v", project)
	}
	var projects []struct {
		ID              string `json:"id"`
		ActivationCount int    `json:"activationCount"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/projects", nil, http.StatusOK, &projects)
	if len(projects) != 1 || projects[0].ID != project.ID || projects[0].ActivationCount != 0 {
		t.Fatalf("project list = %#v", projects)
	}

	var deployment struct {
		Plan domain.Plan `json:"plan"`
	}
	requestJSON(t, handler, http.MethodPost, "/api/projects/web-project/link", map[string]any{
		"skill": "local/web-link", "agents": []string{"claude"},
	}, http.StatusOK, &deployment)
	if len(deployment.Plan.Operations) != 1 || deployment.Plan.Operations[0].Status != domain.StatusCreate {
		t.Fatalf("link plan = %#v", deployment.Plan)
	}
	linkTarget := filepath.Join(projectPath, ".claude", "skills", "web-link")
	if info, err := os.Lstat(linkTarget); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("project link target = %v, err=%v", info, err)
	}

	requestJSON(t, handler, http.MethodPost, "/api/projects/web-project/link", map[string]any{
		"skill": "local/web-link", "agents": []string{"claude"},
	}, http.StatusOK, &deployment)
	if len(deployment.Plan.Operations) != 1 || deployment.Plan.Operations[0].Status != domain.StatusUnchanged {
		t.Fatalf("repeated link plan = %#v", deployment.Plan)
	}
	requestJSON(t, handler, http.MethodPost, "/api/projects/web-project/copy", map[string]any{
		"skill": "local/web-copy", "agents": []string{"codex"},
	}, http.StatusOK, &deployment)
	copyTarget := filepath.Join(projectPath, ".codex", "skills", "web-copy")
	if info, err := os.Lstat(copyTarget); err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("project copy target = %v, err=%v", info, err)
	}

	requestJSON(t, handler, http.MethodDelete, "/api/projects/web-project", nil, http.StatusBadRequest, nil)
	requestJSON(t, handler, http.MethodPost, "/api/projects/web-project/unlink", map[string]any{"skill": "web-link"}, http.StatusOK, nil)
	requestJSON(t, handler, http.MethodPost, "/api/projects/web-project/unlink", map[string]any{"skill": "local/web-copy"}, http.StatusOK, nil)
	requestJSON(t, handler, http.MethodDelete, "/api/projects/web-project", nil, http.StatusOK, nil)
}

func testStore(t *testing.T) *store.Store {
	t.Helper()
	root := t.TempDir()
	storage, err := store.New(store.Paths{
		Home:        filepath.Join(root, "home", ".skm"),
		UserHome:    filepath.Join(root, "home"),
		ProjectRoot: filepath.Join(root, "project"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Ensure(); err != nil {
		t.Fatal(err)
	}
	return storage
}

func makeSkill(t *testing.T, name string) string {
	t.Helper()
	directory := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: Review changes safely\n---\n"
	if err := os.WriteFile(filepath.Join(directory, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return directory
}

func requestJSON(t *testing.T, handler http.Handler, method, url string, body any, wantStatus int, target any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, url, reader)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	defer response.Body.Close()
	if recorder.Code != wantStatus {
		t.Fatalf("%s %s = %d, want %d: %s", method, url, recorder.Code, wantStatus, recorder.Body.String())
	}
	if target != nil {
		if err := json.NewDecoder(response.Body).Decode(target); err != nil {
			t.Fatal(err)
		}
	}
}

func hasTag(values []tagCount, name string) bool {
	for _, value := range values {
		if value.Name == name {
			return true
		}
	}
	return false
}
