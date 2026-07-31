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
