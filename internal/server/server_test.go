package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
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
	var details librarySkillDetails
	requestJSON(t, handler, http.MethodGet, "/api/skills/local/review", nil, http.StatusOK, &details)
	if details.ID != created.ID || !strings.Contains(details.Body, "Follow the review workflow.") {
		t.Fatalf("Skill details = %#v", details)
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

func TestAddSourceFromSkillsCommand(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	repository := filepath.Join(t.TempDir(), "market repo")
	if err := os.MkdirAll(repository, 0o755); err != nil {
		t.Fatal(err)
	}
	runGitTest(t, repository, "init", "-b", "main")
	makeRepositorySkill(t, repository, "skills/better-ui", "better-ui")
	makeRepositorySkill(t, repository, "skills/better-colors", "better-colors")
	runGitTest(t, repository, "add", ".")
	runGitTest(t, repository, "-c", "user.name=skm-test", "-c", "user.email=skm@example.invalid", "commit", "-m", "initial")

	storage := testStore(t)
	handler := New(storage).Handler()
	var result struct {
		Source domain.Source  `json:"source"`
		Skills []domain.Skill `json:"skills"`
	}
	requestJSON(t, handler, http.MethodPost, "/api/sources", map[string]any{
		"input": `npx skills@latest add "` + repository + `" --skill better-ui`,
		"tags":  []string{"design"},
	}, http.StatusCreated, &result)
	if result.Source.Name != "market-repo" || result.Source.URL != repository {
		t.Fatalf("source = %#v", result.Source)
	}
	if len(result.Source.Paths) != 1 || result.Source.Paths[0] != "skills/better-ui" {
		t.Fatalf("source paths = %#v", result.Source.Paths)
	}
	if len(result.Skills) != 1 || result.Skills[0].ID != "market-repo/better-ui" {
		t.Fatalf("imported Skills = %#v", result.Skills)
	}
	requestJSON(t, handler, http.MethodPost, "/api/sources", map[string]any{
		"input": "npx skills add owner/repo --agent codex",
	}, http.StatusBadRequest, nil)
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
	cursorSkill := makeSkill(t, "web-cursor")
	requestJSON(t, handler, http.MethodPost, "/api/skills", map[string]any{"path": linkSkill}, http.StatusCreated, nil)
	requestJSON(t, handler, http.MethodPost, "/api/skills", map[string]any{"path": copySkill}, http.StatusCreated, nil)
	requestJSON(t, handler, http.MethodPost, "/api/skills", map[string]any{"path": cursorSkill}, http.StatusCreated, nil)

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

	makeProjectSkill(t, projectPath, "claude", "claude-only")
	makeProjectSkill(t, projectPath, "codex", "codex-only")
	makeProjectSkill(t, projectPath, "claude", "shared")
	makeProjectSkill(t, projectPath, "codex", "shared")
	makeProjectSkill(t, projectPath, "cursor", "cursor-only")
	makeProjectSkill(t, projectPath, "agent", "agent-only")
	for _, agent := range []string{"claude", "codex"} {
		skillsRoot := filepath.Join(projectPath, "."+agent, "skills")
		if err := os.WriteFile(filepath.Join(skillsRoot, ".DS_Store"), []byte("finder metadata"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillsRoot, "README.md"), []byte("Skill directory notes"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(skillsRoot, "notes"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var details struct {
		Exists bool `json:"exists"`
		Scan   struct {
			SkillCount  int            `json:"skillCount"`
			AgentCounts map[string]int `json:"agentCounts"`
			Agents      []struct {
				ID         string `json:"id"`
				Label      string `json:"label"`
				SkillCount int    `json:"skillCount"`
			} `json:"agents"`
			Skills []struct {
				ID     string   `json:"id"`
				Agents []string `json:"agents"`
			} `json:"skills"`
		} `json:"scan"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/projects/web-project", nil, http.StatusOK, &details)
	if !details.Exists || details.Scan.SkillCount != 5 || details.Scan.AgentCounts["claude"] != 2 || details.Scan.AgentCounts["codex"] != 2 || details.Scan.AgentCounts["cursor"] != 1 || details.Scan.AgentCounts["agent"] != 1 {
		t.Fatalf("project scan summary = %#v", details)
	}
	agents := make(map[string]string)
	for _, agent := range details.Scan.Agents {
		agents[agent.ID] = agent.Label
	}
	if agents["cursor"] != "Cursor" || agents["agent"] != "Agent" {
		t.Fatalf("project scan Agents = %#v", details.Scan.Agents)
	}
	for _, scanned := range details.Scan.Skills {
		if scanned.ID == "shared" && (len(scanned.Agents) != 2 || scanned.Agents[0] != "claude" || scanned.Agents[1] != "codex") {
			t.Fatalf("shared Skill scan = %#v", scanned)
		}
		if scanned.ID == "cursor-only" && (len(scanned.Agents) != 1 || scanned.Agents[0] != "cursor") {
			t.Fatalf("Cursor Skill scan = %#v", scanned)
		}
	}
	var projectSkillDetail struct {
		ID        string `json:"id"`
		Documents []struct {
			Agent       string         `json:"agent"`
			Name        string         `json:"name"`
			Description string         `json:"description"`
			Body        string         `json:"body"`
			Metadata    map[string]any `json:"metadata"`
		} `json:"documents"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/projects/web-project/skills/shared", nil, http.StatusOK, &projectSkillDetail)
	if projectSkillDetail.ID != "shared" || len(projectSkillDetail.Documents) != 2 {
		t.Fatalf("project Skill details = %#v", projectSkillDetail)
	}
	for _, document := range projectSkillDetail.Documents {
		if document.Name != "shared" || document.Description == "" || document.Body == "" || document.Metadata["name"] != "shared" {
			t.Fatalf("project Skill document = %#v", document)
		}
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
		"skill": "local/web-cursor", "agents": []string{"cursor"},
	}, http.StatusOK, &deployment)
	cursorTarget := filepath.Join(projectPath, ".cursor", "skills", "web-cursor")
	if info, err := os.Lstat(cursorTarget); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("project Cursor target = %v, err=%v", info, err)
	}
	requestJSON(t, handler, http.MethodPost, "/api/projects/web-project/link", map[string]any{
		"skill": "local/web-cursor", "agents": []string{"windsurf"},
	}, http.StatusBadRequest, nil)

	requestJSON(t, handler, http.MethodPost, "/api/projects/web-project/link", map[string]any{
		"skill": "local/web-link", "agents": []string{"claude"},
	}, http.StatusOK, &deployment)
	if len(deployment.Plan.Operations) != 2 {
		t.Fatalf("repeated link plan = %#v", deployment.Plan)
	}
	for _, operation := range deployment.Plan.Operations {
		if operation.Status != domain.StatusUnchanged {
			t.Fatalf("repeated link operation = %#v", operation)
		}
	}
	requestJSON(t, handler, http.MethodPost, "/api/projects/web-project/copy", map[string]any{
		"skill": "local/web-copy", "agents": []string{"codex"},
	}, http.StatusOK, &deployment)
	copyTarget := filepath.Join(projectPath, ".codex", "skills", "web-copy")
	if info, err := os.Lstat(copyTarget); err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("project copy target = %v, err=%v", info, err)
	}
	if err := os.WriteFile(filepath.Join(copyTarget, ".DS_Store"), []byte("finder metadata"), 0o644); err != nil {
		t.Fatal(err)
	}
	requestJSON(t, handler, http.MethodPost, "/api/enable", map[string]any{
		"skills": []string{"local/web-link"}, "agents": []string{"codex"}, "mode": "auto",
	}, http.StatusOK, nil)
	if err := os.WriteFile(filepath.Join(copyTarget, "SKILL.md"), []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}
	requestJSON(t, handler, http.MethodPost, "/api/enable", map[string]any{
		"skills": []string{"local/web-link"}, "agents": []string{"codex"}, "mode": "auto",
	}, http.StatusOK, nil)

	requestJSON(t, handler, http.MethodDelete, "/api/projects/web-project", nil, http.StatusBadRequest, nil)
	requestJSON(t, handler, http.MethodPost, "/api/projects/web-project/unlink", map[string]any{"skill": "web-link"}, http.StatusOK, nil)
	requestJSON(t, handler, http.MethodPost, "/api/projects/web-project/unlink", map[string]any{"skill": "web-cursor"}, http.StatusOK, nil)
	requestJSON(t, handler, http.MethodPost, "/api/projects/web-project/unlink", map[string]any{"skill": "local/web-copy"}, http.StatusBadRequest, nil)
	requestJSON(t, handler, http.MethodPost, "/api/projects/web-project/unlink", map[string]any{"skill": "local/web-copy", "force": true}, http.StatusOK, nil)
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
	content := "---\nname: " + name + "\ndescription: Review changes safely\n---\n\nFollow the review workflow.\n"
	if err := os.WriteFile(filepath.Join(directory, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return directory
}

func makeProjectSkill(t *testing.T, projectPath, agent, name string) {
	t.Helper()
	directory := filepath.Join(projectPath, "."+agent, "skills", name)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: Project scan test Skill\n---\n\n# " + name + "\n"
	if err := os.WriteFile(filepath.Join(directory, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func makeRepositorySkill(t *testing.T, root, relative, name string) {
	t.Helper()
	directory := filepath.Join(root, relative)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: Repository Skill\n---\n\n# " + name + "\n"
	if err := os.WriteFile(filepath.Join(directory, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGitTest(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
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
