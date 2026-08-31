package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/skill"
	"github.com/zzzzzyijie/skm/internal/store"
)

func TestHandlerServesEmbeddedUIWithSecurityHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()
	New(testStore(t)).Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	body := recorder.Body.String()
	if recorder.Code != http.StatusOK || !strings.Contains(body, "AI Skill Manager") {
		t.Fatalf("GET / = %d, body=%s", recorder.Code, body)
	}
	if strings.Contains(body, `data-page="dashboard"`) || strings.Contains(body, `components/dashboard.js`) {
		t.Fatal("embedded UI still exposes the removed dashboard page")
	}
	if !strings.Contains(body, `data-page="library"`) || !strings.Contains(body, `data-page="prompts"`) || !strings.Contains(body, `data-page="projects"`) {
		t.Fatal("embedded UI is missing a core navigation item")
	}
	if !strings.Contains(body, `components/prompts.js`) {
		t.Fatal("embedded UI is missing the Prompt component")
	}
	if got := recorder.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q", got)
	}
	if got := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'self'") {
		t.Fatalf("Content-Security-Policy = %q", got)
	}
}

func TestEmbeddedUIProvidesPersistentSettingsAndThemeChoices(t *testing.T) {
	handler := New(testStore(t)).Handler()

	checks := []struct {
		path    string
		markers []string
	}{
		{path: "/", markers: []string{`id="settings-toggle"`, `id="mobile-settings-toggle"`, `src="/theme-init.js"`}},
		{path: "/theme-init.js", markers: []string{`localStorage.getItem('skm-theme')`, `dataset.theme`}},
		{path: "/app.js", markers: []string{"openSettings", "settings-dark-mode", "data-settings-lang", "setTheme", "skm-lang", "skm-theme", `'nav.library': '技能'`, `'nav.prompts': '提示词'`, `'nav.projects': '项目'`}},
		{path: "/app.css", markers: []string{`[data-theme="light"]`, ".settings-segmented", ".settings-switch"}},
	}

	for _, check := range checks {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, check.path, nil))
		body := recorder.Body.String()
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, body=%s", check.path, recorder.Code, body)
		}
		for _, marker := range check.markers {
			if !strings.Contains(body, marker) {
				t.Fatalf("GET %s is missing %q", check.path, marker)
			}
		}
	}
	app := fetchEmbeddedAsset(t, handler, "/app.js")
	if strings.Contains(app, "App.navigate(App.currentPage, false)") {
		t.Fatal("language switching still navigates through a loading state")
	}
	for _, marker := range []string{"suppressTransitionsForSwap", "repaintCurrentPageLanguage", "refreshOpenSettingsLanguage"} {
		if !strings.Contains(app, marker) {
			t.Fatalf("language switching is missing %q", marker)
		}
	}
}

func TestEmbeddedUIProvidesGitSettingsAndBrandSync(t *testing.T) {
	handler := New(testStore(t)).Handler()

	checks := []struct {
		path    string
		markers []string
	}{
		{path: "/", markers: []string{`id="sync-toggle"`, `id="mobile-sync-toggle"`, "sidebar-brand-row"}},
		{path: "/app.js", markers: []string{"runGitSync", "openSettings('git')", "/api/workspace/preview", "/api/workspace/sync", "/api/sources", "data-remove-git-source", "workspaceConflictChoices"}},
		{path: "/app.css", markers: []string{".brand-sync-button", ".workspace-card", ".workspace-change-row", ".git-source-list", ".sync-result-row", ".settings-layout"}},
	}

	for _, check := range checks {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, check.path, nil))
		body := recorder.Body.String()
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, body=%s", check.path, recorder.Code, body)
		}
		for _, marker := range check.markers {
			if !strings.Contains(body, marker) {
				t.Fatalf("GET %s is missing %q", check.path, marker)
			}
		}
	}
	app := fetchEmbeddedAsset(t, handler, "/app.js")
	if strings.Contains(app, "btn-save-git-source") || strings.Contains(app, "function saveGitSource") {
		t.Fatal("Git settings still exposes the removed add-source form")
	}
}

func TestWorkspaceSyncAPIConfiguresPreviewsAndPublishesSkillsAndPrompts(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	remoteRoot := t.TempDir()
	remote := filepath.Join(remoteRoot, "workspace.git")
	runGitTest(t, remoteRoot, "init", "--bare", "-b", "main", remote)
	storage := testStore(t)
	handler := New(storage).Handler()
	skillPath := makeSkill(t, "workspace-skill")
	requestJSON(t, handler, http.MethodPost, "/api/skills", map[string]any{"path": skillPath}, http.StatusCreated, nil)
	requestJSON(t, handler, http.MethodPost, "/api/prompts", map[string]any{
		"name": "workspace-prompt", "description": "Workspace Prompt", "tags": []string{"general"}, "body": "Synchronize this Prompt.",
	}, http.StatusCreated, nil)

	var configured workspaceView
	requestJSON(t, handler, http.MethodPut, "/api/workspace", map[string]any{
		"url": remote, "ref": "main",
	}, http.StatusOK, &configured)
	if !configured.Configured || configured.Config == nil || configured.Config.URL != remote {
		t.Fatalf("configured workspace = %#v", configured)
	}
	var preview struct {
		Uploads   int `json:"uploads"`
		Downloads int `json:"downloads"`
		Conflicts int `json:"conflicts"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/workspace/preview", nil, http.StatusOK, &preview)
	if preview.Uploads != 2 || preview.Downloads != 0 || preview.Conflicts != 0 {
		t.Fatalf("workspace preview = %#v", preview)
	}
	var synced struct {
		Revision string `json:"revision"`
		Applied  bool   `json:"applied"`
	}
	requestJSON(t, handler, http.MethodPost, "/api/workspace/sync", map[string]any{
		"resolutions": map[string]string{},
	}, http.StatusOK, &synced)
	if !synced.Applied || synced.Revision == "" {
		t.Fatalf("workspace sync = %#v", synced)
	}
	requestJSON(t, handler, http.MethodGet, "/api/workspace", nil, http.StatusOK, &configured)
	if configured.State == nil || configured.State.Revision != synced.Revision {
		t.Fatalf("workspace state = %#v", configured.State)
	}
}

func TestEmbeddedProjectUIUsesAgentScopedDetailsWithoutPlanSection(t *testing.T) {
	recorder := httptest.NewRecorder()
	New(testStore(t)).Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/components/projects.js", nil))
	body := recorder.Body.String()
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /components/projects.js = %d, body=%s", recorder.Code, body)
	}
	for _, marker := range []string{"data-project-skill-agent", "?agent=", "projectExistingAgentsForSkill"} {
		if !strings.Contains(body, marker) {
			t.Fatalf("project UI is missing %q", marker)
		}
	}
	if strings.Contains(body, "function projectPlanMarkup") {
		t.Fatal("project UI still renders the deployment plan section")
	}
}

func TestEmbeddedLibraryUIUsesScannedManageableAgents(t *testing.T) {
	handler := New(testStore(t)).Handler()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/components/library.js", nil))
	body := recorder.Body.String()
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /components/library.js = %d, body=%s", recorder.Code, body)
	}
	for _, marker := range []string{"btn-scan-agents", "agent.detected", "btn-new-custom-agent", "tagPickerMarkup", "btn-create-tag", "managedTagRowsMarkup", "beginManagedTagRename", "/api/skill-tags", "detail-new-tag", "createSkillDetailTag", "syncSkillTagState", `data-import-mode="command"`, "btn-choose-skill-zip", "/api/dialogs/skill-", "/api/skills/validate", "baseHash", "skill-content-editor", "showSkillEditor", "/api/sources/preview", "sourceImportState", "renderSourceImportPreview", "data-source-import-candidate", "selectedPaths"} {
		if !strings.Contains(body, marker) {
			t.Fatalf("Library UI is missing %q", marker)
		}
	}
	for _, marker := range []string{
		`id="btn-create-tag" disabled`, "syncManagedTagCreateState",
		"data-create-add-skill-tag", "createAddSkillTag", "syncAddSkillTagCreateState",
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("Library tag creation UI is missing %q", marker)
		}
	}
	if strings.Contains(body, "agent.required") || strings.Contains(body, "lib.fixedAgent") {
		t.Fatal("Agent management UI still contains fixed Agent behavior")
	}
	if strings.Contains(body, "await openSkillDetails(id)") {
		t.Fatal("Skill tag save still reopens the details modal and causes a visible refresh")
	}
	if strings.Contains(body, "await showManageTagsModal()") {
		t.Fatal("Tag mutations still recreate the management modal and cause a visible flash")
	}
	if strings.Contains(body, "promptState.tags = managedTags") {
		t.Fatal("Skill tag mutations still overwrite the independent Prompt tag registry")
	}
	stylesheet := fetchEmbeddedAsset(t, handler, "/app.css")
	for _, marker := range []string{".modal.source-import-modal", ".source-import-candidate.is-invalid", ".source-import-candidate.is-conflict", ".source-import-selection-error"} {
		if !strings.Contains(stylesheet, marker) {
			t.Fatalf("Source import UI stylesheet is missing %q", marker)
		}
	}

	agentIcons := map[string]string{
		"claude": "Claude Code", "codex": "Codex", "cursor": "Cursor", "copilot": "GitHub Copilot",
		"gemini": "Gemini CLI", "windsurf": "Windsurf", "kiro": "Kiro", "cline": "Cline",
		"opencode": "OpenCode", "trae": "Trae", "hermes": "Hermes Agent", "openclaw": "OpenClaw",
	}
	for id, name := range agentIcons {
		icon := fetchEmbeddedAsset(t, handler, "/assets/"+id+".svg")
		for _, marker := range []string{"<svg", "<title>" + name + "</title>"} {
			if !strings.Contains(icon, marker) {
				t.Errorf("Agent icon %s is missing %q", id, marker)
			}
		}
	}
}

func TestAddSkillImportsZIPArchive(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "zip-skill.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("zip-skill/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("---\nname: zip-skill\ndescription: Imported archive\n---\n\nUse it.\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	handler := New(testStore(t)).Handler()
	var imported domain.Skill
	requestJSON(t, handler, http.MethodPost, "/api/skills", map[string]any{
		"path": archivePath, "tags": []string{"general"},
	}, http.StatusCreated, &imported)
	if imported.ID != "local/zip-skill" || !strings.Contains(imported.Path, filepath.Join("objects", imported.Hash)) {
		t.Fatalf("imported Skill = %#v", imported)
	}
	if _, err := os.Stat(filepath.Join(imported.Path, "SKILL.md")); err != nil {
		t.Fatalf("imported snapshot is missing: %v", err)
	}
}

func TestEmbeddedPromptUIUsesStructuredEditorAndClipboard(t *testing.T) {
	recorder := httptest.NewRecorder()
	New(testStore(t)).Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/components/prompts.js", nil))
	body := recorder.Body.String()
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /components/prompts.js = %d, body=%s", recorder.Code, body)
	}
	for _, marker := range []string{"/api/prompts/validate", "/api/tags?scope=prompt", "data-copy-prompt", "copySavedPrompt", "prompt-name", "prompt-description", "prompt-tags", "prompt-new-tag", "btn-create-prompt-tag", "createPromptEditorTag", "tagPickerMarkup", "showManageTagsModal", "baseHash", "setRangeText"} {
		if !strings.Contains(body, marker) {
			t.Fatalf("Prompt UI is missing %q", marker)
		}
	}
	for _, marker := range []string{"id=\"btn-create-prompt-tag\" disabled", "syncPromptEditorTagCreateState", "api.post('/api/tags?scope=prompt'"} {
		if !strings.Contains(body, marker) {
			t.Fatalf("Prompt tag creation UI is missing %q", marker)
		}
	}
	if strings.Contains(body, "data-duplicate-prompt") || strings.Contains(body, "openPromptUse") {
		t.Fatal("Prompt UI still contains duplicate or use-dialog actions")
	}
}

func TestEmbeddedSkillAndPromptUIProvidePersistentViewSwitching(t *testing.T) {
	handler := New(testStore(t)).Handler()
	assets := []struct {
		path    string
		markers []string
	}{
		{path: "/app.js", markers: []string{"collectionViewMode", "collectionViewSwitcherMarkup", "bindCollectionViewSwitcher", "skm-", "data-collection-view", "view.grid", "view.list"}},
		{path: "/components/library.js", markers: []string{"collectionViewMode('library')", "collectionViewSwitcherMarkup('library'", "bindCollectionViewSwitcher('library'", "is-list-view"}},
		{path: "/components/prompts.js", markers: []string{"collectionViewMode('prompts')", "collectionViewSwitcherMarkup('prompts'", "bindCollectionViewSwitcher('prompts'", "is-list-view"}},
		{path: "/app.css", markers: []string{".collection-view-switcher", ".collection-view-option.active", ".card-grid.is-list-view", ".prompt-grid.is-list-view"}},
	}
	for _, asset := range assets {
		body := fetchEmbeddedAsset(t, handler, asset.path)
		for _, marker := range asset.markers {
			if !strings.Contains(body, marker) {
				t.Errorf("%s is missing %q", asset.path, marker)
			}
		}
	}
}

func TestEmbeddedSkillEditorUsesNamedResponsiveGrid(t *testing.T) {
	handler := New(testStore(t)).Handler()
	stylesheet := fetchEmbeddedAsset(t, handler, "/app.css")
	for _, marker := range []string{
		`.skill-editor-shell { display: grid; height: 100%; min-height: 0; grid-template-areas: "content details";`,
		`.prompt-editor-shell, .skill-editor-shell { height: auto; grid-template-areas: "details" "content"; grid-template-columns: 1fr; }`,
	} {
		if !strings.Contains(stylesheet, marker) {
			t.Fatalf("Skill editor stylesheet is missing %q", marker)
		}
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

func TestSkillEditAPIValidatesConflictsAndRefreshesUserActivation(t *testing.T) {
	storage := testStore(t)
	handler := New(storage).Handler()
	skillPath := makeSkill(t, "editable-review")
	if err := os.MkdirAll(filepath.Join(skillPath, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "references", "guide.md"), []byte("preserved\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var created domain.Skill
	requestJSON(t, handler, http.MethodPost, "/api/skills", map[string]any{"path": skillPath}, http.StatusCreated, &created)
	var details librarySkillDetails
	requestJSON(t, handler, http.MethodGet, "/api/skills/local/editable-review", nil, http.StatusOK, &details)
	if !details.Editable || !strings.Contains(details.Content, "name: editable-review") {
		t.Fatalf("editable details = %#v", details)
	}

	requestJSON(t, handler, http.MethodPost, "/api/enable", map[string]any{
		"skills": []string{created.ID}, "agents": []string{"codex"}, "mode": "auto",
	}, http.StatusOK, nil)
	updatedContent := strings.Replace(details.Content, "Review changes safely", "Review edits carefully", 1)
	var validated skill.Document
	requestJSON(t, handler, http.MethodPost, "/api/skills/validate", map[string]any{
		"skill": created.ID, "content": updatedContent,
	}, http.StatusOK, &validated)
	if validated.Description != "Review edits carefully" || validated.Files != 2 {
		t.Fatalf("validated Skill = %#v", validated)
	}
	requestJSON(t, handler, http.MethodPut, "/api/skills/local/editable-review", map[string]any{
		"content": updatedContent, "baseHash": "stale",
	}, http.StatusConflict, nil)

	var result skillUpdateResult
	requestJSON(t, handler, http.MethodPut, "/api/skills/local/editable-review", map[string]any{
		"content": updatedContent, "baseHash": created.Hash,
	}, http.StatusOK, &result)
	if !result.Applied || result.DeploymentError != "" || result.Skill.Hash == created.Hash {
		t.Fatalf("Skill update result = %#v", result)
	}
	if data, err := os.ReadFile(filepath.Join(result.Skill.Path, "references", "guide.md")); err != nil || string(data) != "preserved\n" {
		t.Fatalf("preserved reference = %q, %v", data, err)
	}
	target := filepath.Join(storage.Paths.UserHome, ".codex", "skills", "editable-review")
	linked, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	wantLinked, err := filepath.EvalSymlinks(result.Skill.Path)
	if err != nil {
		t.Fatal(err)
	}
	if linked != wantLinked {
		t.Fatalf("user Activation points to %q, want %q", linked, wantLinked)
	}
	if _, err := os.Stat(created.Path); !os.IsNotExist(err) {
		t.Fatalf("old unreferenced snapshot still exists: %v", err)
	}

	rename := strings.Replace(updatedContent, "name: editable-review", "name: renamed-review", 1)
	requestJSON(t, handler, http.MethodPut, "/api/skills/local/editable-review", map[string]any{
		"content": rename, "baseHash": result.Skill.Hash,
	}, http.StatusBadRequest, nil)
}

func TestPromptAPILifecycleRenderingAndConflict(t *testing.T) {
	handler := New(testStore(t)).Handler()
	content := "---\nname: release-notes\ndescription: Draft release notes\ntags: [release]\nvariables:\n  - name: version\n    required: true\n---\nRelease {{version}}\n"
	var created domain.Prompt
	requestJSON(t, handler, http.MethodPost, "/api/prompts", map[string]any{"content": content}, http.StatusCreated, &created)
	if created.ID != "local/release-notes" || created.Hash == "" || len(created.Variables) != 1 {
		t.Fatalf("created Prompt = %#v", created)
	}

	var values []domain.Prompt
	requestJSON(t, handler, http.MethodGet, "/api/prompts?tag=release", nil, http.StatusOK, &values)
	if len(values) != 1 || values[0].ID != created.ID {
		t.Fatalf("Prompt list = %#v", values)
	}
	var details promptDetails
	requestJSON(t, handler, http.MethodGet, "/api/prompts/local/release-notes", nil, http.StatusOK, &details)
	if details.Content != content || !strings.Contains(details.Body, "Release {{version}}") {
		t.Fatalf("Prompt details = %#v", details)
	}

	var rendered struct {
		Content          string   `json:"content"`
		MissingVariables []string `json:"missingVariables"`
	}
	requestJSON(t, handler, http.MethodPost, "/api/prompt-render", map[string]any{"prompt": created.ID, "variables": map[string]string{}}, http.StatusOK, &rendered)
	if !reflect.DeepEqual(rendered.MissingVariables, []string{"version"}) {
		t.Fatalf("missing variables = %#v", rendered)
	}
	requestJSON(t, handler, http.MethodPost, "/api/prompt-render", map[string]any{"prompt": created.ID, "variables": map[string]string{"version": "v1.0"}}, http.StatusOK, &rendered)
	if rendered.Content != "Release v1.0\n" || len(rendered.MissingVariables) != 0 {
		t.Fatalf("rendered Prompt = %#v", rendered)
	}

	updatedContent := strings.Replace(content, "Draft release notes", "Publish release notes", 1)
	requestJSON(t, handler, http.MethodPut, "/api/prompts/local/release-notes", map[string]any{"content": updatedContent, "baseHash": "stale"}, http.StatusConflict, nil)
	requestJSON(t, handler, http.MethodPut, "/api/prompts/local/release-notes", map[string]any{"content": updatedContent, "baseHash": created.Hash}, http.StatusOK, &created)
	if created.Description != "Publish release notes" {
		t.Fatalf("updated Prompt = %#v", created)
	}
	requestJSON(t, handler, http.MethodDelete, "/api/prompts/local/release-notes", nil, http.StatusOK, nil)
	requestJSON(t, handler, http.MethodGet, "/api/prompts/local/release-notes", nil, http.StatusNotFound, nil)

	structured := map[string]any{
		"name": "meeting-notes", "description": "Summarize a meeting",
		"tags": []string{"work", "summary"}, "body": "Summarize the meeting notes clearly.",
	}
	var validated struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
		Body string   `json:"body"`
	}
	requestJSON(t, handler, http.MethodPost, "/api/prompts/validate", structured, http.StatusOK, &validated)
	if validated.Name != "meeting-notes" || !reflect.DeepEqual(validated.Tags, []string{"summary", "work"}) || validated.Body != "Summarize the meeting notes clearly.\n" {
		t.Fatalf("validated structured Prompt = %#v", validated)
	}
	requestJSON(t, handler, http.MethodPost, "/api/prompts", structured, http.StatusCreated, &created)
	requestJSON(t, handler, http.MethodGet, "/api/prompts/local/meeting-notes", nil, http.StatusOK, &details)
	if !strings.Contains(details.Content, "name: meeting-notes") || details.Body != validated.Body {
		t.Fatalf("structured Prompt details = %#v", details)
	}
	requestJSON(t, handler, http.MethodDelete, "/api/prompts/local/meeting-notes", nil, http.StatusOK, nil)
}

func TestSkillAndPromptTagsAreManagedIndependently(t *testing.T) {
	handler := New(testStore(t)).Handler()
	tagName := "简小知"
	requestJSON(t, handler, http.MethodPost, "/api/tags", map[string]any{"name": tagName}, http.StatusCreated, nil)
	requestJSON(t, handler, http.MethodPost, "/api/tags?scope=prompt", map[string]any{"name": tagName}, http.StatusCreated, nil)

	skillPath := makeSkill(t, "tagged-skill")
	var createdSkill domain.Skill
	requestJSON(t, handler, http.MethodPost, "/api/skills", map[string]any{
		"path": skillPath, "tags": []string{tagName},
	}, http.StatusCreated, &createdSkill)
	var createdPrompt domain.Prompt
	requestJSON(t, handler, http.MethodPost, "/api/prompts", map[string]any{
		"name": "tagged-prompt", "description": "Prompt with a managed tag",
		"tags": []string{tagName}, "body": "Write a concise design review.",
	}, http.StatusCreated, &createdPrompt)

	var values []tagCount
	requestJSON(t, handler, http.MethodGet, "/api/tags", nil, http.StatusOK, &values)
	skillTag := findTagCount(values, tagName)
	if skillTag == nil || skillTag.Count != 1 || skillTag.SkillCount != 1 || skillTag.PromptCount != 0 || skillTag.Default {
		t.Fatalf("managed Skill tag = %#v", skillTag)
	}
	requestJSON(t, handler, http.MethodGet, "/api/tags?scope=prompt", nil, http.StatusOK, &values)
	promptTag := findTagCount(values, tagName)
	if promptTag == nil || promptTag.Count != 1 || promptTag.SkillCount != 0 || promptTag.PromptCount != 1 || promptTag.Default {
		t.Fatalf("managed Prompt tag = %#v", promptTag)
	}

	requestJSON(t, handler, http.MethodDelete, "/api/tags/"+tagName, nil, http.StatusBadRequest, nil)
	requestJSON(t, handler, http.MethodDelete, "/api/tags/"+tagName+"?scope=prompt", nil, http.StatusBadRequest, nil)

	requestJSON(t, handler, http.MethodPost, "/api/tags/rename", map[string]any{
		"old": tagName, "new": "product-design",
	}, http.StatusOK, nil)
	var skillDetails librarySkillDetails
	requestJSON(t, handler, http.MethodGet, "/api/skills/"+createdSkill.ID, nil, http.StatusOK, &skillDetails)
	if !reflect.DeepEqual(skillDetails.Tags, []string{"product-design"}) {
		t.Fatalf("renamed Skill tags = %#v", skillDetails.Tags)
	}
	var details promptDetails
	requestJSON(t, handler, http.MethodGet, "/api/prompts/"+createdPrompt.ID, nil, http.StatusOK, &details)
	if !reflect.DeepEqual(details.Tags, []string{tagName}) || !strings.Contains(details.Content, tagName) {
		t.Fatalf("Skill rename changed Prompt tags = %#v, content=%q", details.Tags, details.Content)
	}

	requestJSON(t, handler, http.MethodPost, "/api/tags/rename?scope=prompt", map[string]any{
		"old": tagName, "new": "prompt-design",
	}, http.StatusOK, nil)
	requestJSON(t, handler, http.MethodGet, "/api/prompts/"+createdPrompt.ID, nil, http.StatusOK, &details)
	if !reflect.DeepEqual(details.Tags, []string{"prompt-design"}) || !strings.Contains(details.Content, "prompt-design") {
		t.Fatalf("renamed Prompt tags = %#v, content=%q", details.Tags, details.Content)
	}
	requestJSON(t, handler, http.MethodGet, "/api/skills/"+createdSkill.ID, nil, http.StatusOK, &skillDetails)
	if !reflect.DeepEqual(skillDetails.Tags, []string{"product-design"}) {
		t.Fatalf("Prompt rename changed Skill tags = %#v", skillDetails.Tags)
	}

	requestJSON(t, handler, http.MethodPut, "/api/skill-tags", map[string]any{
		"skill": createdSkill.ID, "tags": []string{"general"},
	}, http.StatusOK, &createdSkill)
	if !reflect.DeepEqual(createdSkill.Tags, []string{"general"}) {
		t.Fatalf("replaced Skill tags = %#v", createdSkill.Tags)
	}

	requestJSON(t, handler, http.MethodPost, "/api/tags", map[string]any{"name": "unused"}, http.StatusCreated, nil)
	requestJSON(t, handler, http.MethodDelete, "/api/tags/unused", nil, http.StatusOK, nil)
	requestJSON(t, handler, http.MethodGet, "/api/tags", nil, http.StatusOK, &values)
	if findTagCount(values, "unused") != nil {
		t.Fatalf("deleted tag remained: %#v", values)
	}

	requestJSON(t, handler, http.MethodPost, "/api/tags", map[string]any{"name": "shared-unused"}, http.StatusCreated, nil)
	requestJSON(t, handler, http.MethodPost, "/api/tags?scope=prompt", map[string]any{"name": "shared-unused"}, http.StatusCreated, nil)
	requestJSON(t, handler, http.MethodDelete, "/api/tags/shared-unused?scope=prompt", nil, http.StatusOK, nil)
	requestJSON(t, handler, http.MethodGet, "/api/tags", nil, http.StatusOK, &values)
	if findTagCount(values, "shared-unused") == nil {
		t.Fatalf("deleting Prompt tag removed same-named Skill tag: %#v", values)
	}
	requestJSON(t, handler, http.MethodGet, "/api/tags?scope=prompt", nil, http.StatusOK, &values)
	if findTagCount(values, "shared-unused") != nil {
		t.Fatalf("deleted Prompt tag remained: %#v", values)
	}
}

func TestPromptTagScopeHasIndependentRegistryAndDefault(t *testing.T) {
	handler := New(testStore(t)).Handler()
	skillPath := makeSkill(t, "ui-skill")
	requestJSON(t, handler, http.MethodPost, "/api/skills", map[string]any{
		"path": skillPath, "tags": []string{"ui"},
	}, http.StatusCreated, nil)
	requestJSON(t, handler, http.MethodPost, "/api/tags", map[string]any{"name": "skill-ready"}, http.StatusCreated, nil)
	requestJSON(t, handler, http.MethodPost, "/api/tags?scope=prompt", map[string]any{"name": "prompt-ready"}, http.StatusCreated, nil)

	var promptTags []tagCount
	requestJSON(t, handler, http.MethodGet, "/api/tags?scope=prompt", nil, http.StatusOK, &promptTags)
	if findTagCount(promptTags, "ui") != nil || findTagCount(promptTags, "skill-ready") != nil {
		t.Fatalf("Prompt tag scope leaked Skill tags: %#v", promptTags)
	}
	general := findTagCount(promptTags, "general")
	if general == nil || !general.Default {
		t.Fatalf("Prompt default tag missing: %#v", promptTags)
	}
	if findTagCount(promptTags, "prompt-ready") == nil {
		t.Fatalf("Prompt tag scope omitted unused managed tag: %#v", promptTags)
	}

	var created domain.Prompt
	requestJSON(t, handler, http.MethodPost, "/api/prompts", map[string]any{
		"name": "untagged-prompt", "description": "Uses Prompt default", "tags": []string{}, "body": "Body",
	}, http.StatusCreated, &created)
	if !reflect.DeepEqual(created.Tags, []string{"general"}) {
		t.Fatalf("created Prompt tags = %#v, want Prompt default", created.Tags)
	}
}

func TestAgentManagementControlsAvailableActivationTargets(t *testing.T) {
	storage := testStore(t)
	handler := New(storage).Handler()
	for _, path := range []string{
		filepath.Join(storage.Paths.UserHome, ".claude"),
		filepath.Join(storage.Paths.UserHome, ".cursor"),
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	var agents []agentDescriptor
	requestJSON(t, handler, http.MethodGet, "/api/agents", nil, http.StatusOK, &agents)
	configured := make(map[domain.Agent]bool)
	detected := make(map[domain.Agent]bool)
	for _, agent := range agents {
		configured[agent.ID] = agent.Configured
		detected[agent.ID] = agent.Detected
	}
	if !configured[domain.AgentClaude] || !configured[domain.AgentCodex] || configured[domain.AgentCursor] {
		t.Fatalf("default managed agents = %#v", configured)
	}
	if !detected[domain.AgentClaude] || !detected[domain.AgentCursor] || detected[domain.AgentCodex] {
		t.Fatalf("detected agents = %#v", detected)
	}
	requestJSON(t, handler, http.MethodPut, "/api/agents", map[string]any{
		"agents": []string{},
	}, http.StatusOK, &agents)
	for _, agent := range agents {
		if agent.Configured {
			t.Fatalf("Agent %s remained fixed after clearing management", agent.ID)
		}
	}

	skillPath := makeSkill(t, "managed-agent")
	var created domain.Skill
	requestJSON(t, handler, http.MethodPost, "/api/skills", map[string]any{"path": skillPath}, http.StatusCreated, &created)
	requestJSON(t, handler, http.MethodPost, "/api/enable", map[string]any{
		"skills": []string{created.ID}, "mode": "auto",
	}, http.StatusBadRequest, nil)
	requestJSON(t, handler, http.MethodPost, "/api/enable", map[string]any{
		"skills": []string{created.ID}, "agents": []string{"cursor"}, "mode": "auto",
	}, http.StatusBadRequest, nil)

	requestJSON(t, handler, http.MethodPut, "/api/agents", map[string]any{
		"agents": []string{"cursor"},
	}, http.StatusOK, &agents)
	requestJSON(t, handler, http.MethodPost, "/api/enable", map[string]any{
		"skills": []string{created.ID}, "agents": []string{"cursor"}, "mode": "auto",
	}, http.StatusOK, nil)
	requestJSON(t, handler, http.MethodPut, "/api/agents", map[string]any{
		"agents": []string{},
	}, http.StatusBadRequest, nil)
	requestJSON(t, handler, http.MethodPost, "/api/disable", map[string]any{
		"skills": []string{created.ID}, "agents": []string{"cursor"},
	}, http.StatusOK, nil)
	requestJSON(t, handler, http.MethodPut, "/api/agents", map[string]any{
		"agents": []string{},
	}, http.StatusOK, &agents)
	requestJSON(t, handler, http.MethodPost, "/api/agents/custom", map[string]any{
		"id": "local-agent", "name": "Local Agent", "skillsPath": "~/.local-agent/skills",
	}, http.StatusOK, &agents)
	requestJSON(t, handler, http.MethodPost, "/api/enable", map[string]any{
		"skills": []string{created.ID}, "agents": []string{"local-agent"}, "mode": "auto",
	}, http.StatusOK, nil)
	requestJSON(t, handler, http.MethodDelete, "/api/agents/local-agent", nil, http.StatusBadRequest, nil)
	requestJSON(t, handler, http.MethodPost, "/api/disable", map[string]any{
		"skills": []string{created.ID}, "agents": []string{"local-agent"},
	}, http.StatusOK, nil)
	requestJSON(t, handler, http.MethodDelete, "/api/agents/local-agent", nil, http.StatusOK, nil)

	config, err := storage.LoadConfig()
	if err != nil || len(config.Defaults.Agents) != 0 {
		t.Fatalf("saved agent config = %#v, err=%v", config, err)
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

func TestPreviewSourceReportsCandidatesAndImportsSelectedPaths(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	repository := filepath.Join(t.TempDir(), "swiftui source")
	if err := os.MkdirAll(repository, 0o755); err != nil {
		t.Fatal(err)
	}
	runGitTest(t, repository, "init", "-b", "main")
	makeRepositorySkill(t, repository, "swiftui-pro", "swiftui-pro")
	if err := os.MkdirAll(filepath.Join(repository, "swiftui-pro", "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "swiftui-pro", "references", "guide.md"), []byte("guide"), 0o644); err != nil {
		t.Fatal(err)
	}
	makeRepositorySkill(t, repository, "swiftui-pro/skills/swiftui-pro", "swiftui-pro")
	if err := os.Symlink("../../references", filepath.Join(repository, "swiftui-pro", "skills", "swiftui-pro", "references")); err != nil {
		t.Fatal(err)
	}
	runGitTest(t, repository, "add", ".")
	runGitTest(t, repository, "-c", "user.name=skm-test", "-c", "user.email=skm@example.invalid", "commit", "-m", "initial")

	storage := testStore(t)
	handler := New(storage).Handler()
	var preview struct {
		Source          domain.Source `json:"source"`
		RequestedSkills []string      `json:"requestedSkills"`
		Skills          []struct {
			Name  string `json:"name"`
			Path  string `json:"path"`
			Valid bool   `json:"valid"`
			Error string `json:"error"`
		} `json:"skills"`
	}
	requestJSON(t, handler, http.MethodPost, "/api/sources/preview", map[string]any{
		"input": `npx skills add "` + repository + `" --skill swiftui-pro`, "name": "swiftui",
	}, http.StatusOK, &preview)
	if preview.Source.URL != repository || preview.Source.Revision == "" || !reflect.DeepEqual(preview.RequestedSkills, []string{"swiftui-pro"}) {
		t.Fatalf("preview source = %#v, requested=%#v", preview.Source, preview.RequestedSkills)
	}
	if len(preview.Skills) != 2 || !preview.Skills[0].Valid || preview.Skills[0].Path != "swiftui-pro" {
		t.Fatalf("preview skills = %#v", preview.Skills)
	}
	if preview.Skills[1].Valid || !strings.Contains(preview.Skills[1].Error, "symlink escapes skill root: references") {
		t.Fatalf("invalid preview candidate = %#v", preview.Skills[1])
	}
	configured, err := storage.LoadSources()
	if err != nil || len(configured.Sources) != 0 {
		t.Fatalf("preview persisted source: %#v, err=%v", configured, err)
	}

	var imported struct {
		Source domain.Source  `json:"source"`
		Skills []domain.Skill `json:"skills"`
	}
	requestJSON(t, handler, http.MethodPost, "/api/sources", map[string]any{
		"input": preview.Source.URL, "name": preview.Source.Name, "paths": []string{"swiftui-pro"},
	}, http.StatusCreated, &imported)
	if len(imported.Skills) != 1 || imported.Skills[0].Name != "swiftui-pro" || !reflect.DeepEqual(imported.Source.Paths, []string{"swiftui-pro"}) {
		t.Fatalf("selected import = %#v", imported)
	}
}

func TestGitSyncIsolatesSourceFailuresRefreshesDeploymentsAndRemovesBindings(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	storage := testStore(t)
	handler := New(storage).Handler()
	requestJSON(t, handler, http.MethodPost, "/api/sync", map[string]any{}, http.StatusConflict, nil)

	goodRepository := createGitRepository(t, "good-skill")
	badRepository := createGitRepository(t, "bad-skill")
	var goodCreated struct {
		Source domain.Source  `json:"source"`
		Skills []domain.Skill `json:"skills"`
	}
	requestJSON(t, handler, http.MethodPost, "/api/sources", map[string]any{
		"input": goodRepository, "name": "good",
	}, http.StatusCreated, &goodCreated)
	var badCreated struct {
		Source domain.Source  `json:"source"`
		Skills []domain.Skill `json:"skills"`
	}
	requestJSON(t, handler, http.MethodPost, "/api/sources", map[string]any{
		"input": badRepository, "name": "bad",
	}, http.StatusCreated, &badCreated)
	if len(goodCreated.Skills) != 1 || len(badCreated.Skills) != 1 {
		t.Fatalf("created Git Skills: good=%#v bad=%#v", goodCreated.Skills, badCreated.Skills)
	}
	oldGood := goodCreated.Skills[0]
	requestJSON(t, handler, http.MethodPost, "/api/enable", map[string]any{
		"skills": []string{oldGood.ID}, "agents": []string{"codex"}, "mode": "auto",
	}, http.StatusOK, nil)
	deploymentTarget := filepath.Join(storage.Paths.UserHome, ".codex", "skills", oldGood.Name)
	resolvedBefore, err := filepath.EvalSymlinks(deploymentTarget)
	expectedBefore, expectedErr := filepath.EvalSymlinks(oldGood.Path)
	if err != nil || expectedErr != nil || filepath.Clean(resolvedBefore) != filepath.Clean(expectedBefore) {
		t.Fatalf("initial deployment = %q, err=%v, want %q", resolvedBefore, err, oldGood.Path)
	}

	updatedContent := "---\nname: good-skill\ndescription: Updated repository Skill\n---\n\n# good-skill\n\nversion two\n"
	if err := os.WriteFile(filepath.Join(goodRepository, "skills", "good-skill", "SKILL.md"), []byte(updatedContent), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitTest(t, goodRepository, "add", ".")
	runGitTest(t, goodRepository, "-c", "user.name=skm-test", "-c", "user.email=skm@example.invalid", "commit", "-m", "update")

	sources, err := storage.LoadSources()
	if err != nil {
		t.Fatal(err)
	}
	for index := range sources.Sources {
		if sources.Sources[index].Name == "bad" {
			sources.Sources[index].URL = filepath.Join(t.TempDir(), "missing-repository")
		}
	}
	if err := storage.SaveSources(sources); err != nil {
		t.Fatal(err)
	}

	var synced sourceSyncResult
	requestJSON(t, handler, http.MethodPost, "/api/sync", map[string]any{}, http.StatusOK, &synced)
	if synced.Configured != 2 || synced.Updated != 1 || synced.Failed != 1 || synced.SkillCount != 1 || !synced.Applied || synced.DeploymentError != "" {
		t.Fatalf("sync result = %#v", synced)
	}
	statuses := make(map[string]string)
	for _, item := range synced.Results {
		statuses[item.Name] = item.Status
	}
	if statuses["good"] != "updated" || statuses["bad"] != "error" {
		t.Fatalf("sync item statuses = %#v", statuses)
	}
	newGood, err := catalog.New(storage).ResolveLibrary(oldGood.ID)
	if err != nil {
		t.Fatal(err)
	}
	if newGood.Hash == oldGood.Hash || newGood.Description != "Updated repository Skill" {
		t.Fatalf("updated Git Skill = %#v", newGood)
	}
	resolvedAfter, err := filepath.EvalSymlinks(deploymentTarget)
	expectedAfter, expectedErr := filepath.EvalSymlinks(newGood.Path)
	if err != nil || expectedErr != nil || filepath.Clean(resolvedAfter) != filepath.Clean(expectedAfter) {
		t.Fatalf("refreshed deployment = %q, err=%v, want %q", resolvedAfter, err, newGood.Path)
	}

	var removed struct {
		BindingRemoved  bool `json:"bindingRemoved"`
		CheckoutRemoved bool `json:"checkoutRemoved"`
	}
	requestJSON(t, handler, http.MethodDelete, "/api/sources/bad", nil, http.StatusOK, &removed)
	if !removed.BindingRemoved || !removed.CheckoutRemoved {
		t.Fatalf("removed source = %#v", removed)
	}
	if _, err := catalog.New(storage).ResolveLibrary(badCreated.Skills[0].ID); err != nil {
		t.Fatalf("removing Git binding removed imported Library Skill: %v", err)
	}
	requestJSON(t, handler, http.MethodDelete, "/api/sources/bad", nil, http.StatusBadRequest, nil)
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
	requestJSON(t, handler, http.MethodGet, "/api/projects/web-project/skills/shared?agent=codex", nil, http.StatusOK, &projectSkillDetail)
	if len(projectSkillDetail.Documents) != 1 || projectSkillDetail.Documents[0].Agent != "codex" {
		t.Fatalf("filtered project Skill details = %#v", projectSkillDetail)
	}
	requestJSON(t, handler, http.MethodGet, "/api/projects/web-project/skills/shared?agent=cursor", nil, http.StatusNotFound, nil)

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
	requestJSON(t, handler, http.MethodDelete, "/api/projects/web-project/skills/web-link", nil, http.StatusBadRequest, nil)
	if _, err := os.Lstat(linkTarget); err != nil {
		t.Fatalf("managed project Skill was removed directly: %v", err)
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

func TestProjectSkillMigrationAPI(t *testing.T) {
	storage := testStore(t)
	handler := New(storage).Handler()
	projectPath := filepath.Join(t.TempDir(), "migration-project")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatal(err)
	}
	requestJSON(t, handler, http.MethodPost, "/api/projects", map[string]any{"path": projectPath}, http.StatusCreated, nil)

	knownLibraryPath := makeSkill(t, "known-project-skill")
	var knownLibrary domain.Skill
	requestJSON(t, handler, http.MethodPost, "/api/skills", map[string]any{"path": knownLibraryPath}, http.StatusCreated, &knownLibrary)
	makeProjectSkill(t, projectPath, "claude", "known-project-skill")
	makeProjectSkill(t, projectPath, "codex", "known-project-skill")
	var knownProject struct {
		Scan struct {
			Skills []struct {
				ID             string `json:"id"`
				Hash           string `json:"hash"`
				LibrarySkillID string `json:"librarySkillId"`
			} `json:"skills"`
		} `json:"scan"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/projects/migration-project", nil, http.StatusOK, &knownProject)
	if len(knownProject.Scan.Skills) != 1 || knownProject.Scan.Skills[0].LibrarySkillID != knownLibrary.ID || knownProject.Scan.Skills[0].Hash == knownLibrary.Hash {
		t.Fatalf("same-name Library marker = %#v", knownProject.Scan.Skills)
	}
	requestJSON(t, handler, http.MethodDelete, "/api/projects/migration-project/skills/known-project-skill", nil, http.StatusOK, nil)
	for _, agent := range []string{"claude", "codex"} {
		if _, err := os.Lstat(filepath.Join(projectPath, "."+agent, "skills", "known-project-skill")); !os.IsNotExist(err) {
			t.Fatalf("%s project Skill was not removed: %v", agent, err)
		}
	}
	if _, err := catalog.New(storage).ResolveLibrary(knownLibrary.ID); err != nil {
		t.Fatalf("removing project Skill removed Library entry: %v", err)
	}

	makeProjectSkill(t, projectPath, "claude", "followed-project-skill")
	var followed struct {
		Skill domain.Skill `json:"skill"`
	}
	requestJSON(t, handler, http.MethodPost, "/api/projects/migration-project/skills/followed-project-skill/migrate", map[string]any{
		"agent": "claude", "mode": "symlink",
	}, http.StatusCreated, &followed)
	requestJSON(t, handler, http.MethodPost, "/api/enable", map[string]any{
		"skills": []string{followed.Skill.ID}, "agents": []string{"codex"}, "mode": "auto",
	}, http.StatusOK, nil)
	userFollowedLink := filepath.Join(filepath.Dir(storage.Paths.Home), ".codex", "skills", "followed-project-skill")
	requestJSON(t, handler, http.MethodDelete, "/api/projects/migration-project/skills/followed-project-skill", nil, http.StatusOK, nil)
	var preserved struct {
		Health        string          `json:"health"`
		Mode          domain.LinkMode `json:"mode"`
		SnapshotPath  string          `json:"snapshotPath"`
		EffectivePath string          `json:"effectivePath"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/skills/local/followed-project-skill", nil, http.StatusOK, &preserved)
	if preserved.Health != "available" || preserved.Mode != domain.ModeCopy || preserved.EffectivePath == "" {
		t.Fatalf("followed Skill after project removal = %#v", preserved)
	}
	if _, err := os.Stat(preserved.EffectivePath); err != nil {
		t.Fatalf("followed Skill snapshot after project removal: %v", err)
	}
	var statusAfterRemoval domain.Plan
	requestJSON(t, handler, http.MethodGet, "/api/status", nil, http.StatusOK, &statusAfterRemoval)
	var followedUnchanged bool
	for _, operation := range statusAfterRemoval.Operations {
		if operation.SkillID != followed.Skill.ID {
			continue
		}
		if operation.Status == domain.StatusUnchanged || operation.Status == domain.StatusReplaceManaged || operation.Status == domain.StatusCreate {
			followedUnchanged = operation.Status != domain.StatusCreate
		}
	}
	if !followedUnchanged {
		t.Fatalf("followed Skill deployment after project removal = %#v", statusAfterRemoval.Operations)
	}
	resolvedFollowed, err := filepath.EvalSymlinks(userFollowedLink)
	if err != nil {
		t.Fatalf("followed Skill deployment link missing after project removal: %v", err)
	}
	resolvedSnapshot, snapshotErr := filepath.EvalSymlinks(preserved.EffectivePath)
	if snapshotErr != nil || filepath.Clean(resolvedFollowed) != filepath.Clean(resolvedSnapshot) {
		t.Fatalf("followed Skill deployment target = %q, snapshot = %q (err=%v)", resolvedFollowed, resolvedSnapshot, snapshotErr)
	}
	requestJSON(t, handler, http.MethodPost, "/api/disable", map[string]any{
		"skills": []string{followed.Skill.ID}, "agents": []string{"codex"},
	}, http.StatusOK, nil)
	requestJSON(t, handler, http.MethodDelete, "/api/skills/local/followed-project-skill", nil, http.StatusOK, nil)

	externalSkill := makeSkill(t, "external-linked-skill")
	externalLinkRoot := filepath.Join(projectPath, ".cursor", "skills")
	if err := os.MkdirAll(externalLinkRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	externalLink := filepath.Join(externalLinkRoot, "external-linked-skill")
	if err := os.Symlink(externalSkill, externalLink); err != nil {
		t.Fatal(err)
	}
	requestJSON(t, handler, http.MethodDelete, "/api/projects/migration-project/skills/external-linked-skill", nil, http.StatusOK, nil)
	if _, err := os.Lstat(externalLink); !os.IsNotExist(err) {
		t.Fatalf("external project Skill link was not removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(externalSkill, "SKILL.md")); err != nil {
		t.Fatalf("external Skill source was removed with project link: %v", err)
	}

	makeProjectSkill(t, projectPath, "claude", "linked-project-skill")
	var linked struct {
		Skill domain.Skill `json:"skill"`
	}
	requestJSON(t, handler, http.MethodPost, "/api/projects/migration-project/skills/linked-project-skill/migrate", map[string]any{
		"agent": "claude", "mode": "symlink",
	}, http.StatusCreated, &linked)
	linkedSource, err := filepath.EvalSymlinks(filepath.Join(projectPath, ".claude", "skills", "linked-project-skill"))
	if err != nil {
		t.Fatal(err)
	}
	if linked.Skill.ID != "local/linked-project-skill" || linked.Skill.Mode != domain.ModeSymlink || linked.Skill.Path != linkedSource || linked.Skill.SourcePath != linkedSource || linked.Skill.SnapshotPath == "" || linked.Skill.ProjectAgent != domain.AgentClaude {
		t.Fatalf("linked migration = %#v", linked.Skill)
	}
	if _, err := os.Stat(filepath.Join(linked.Skill.SnapshotPath, "SKILL.md")); err != nil {
		t.Fatalf("linked migration fallback snapshot: %v", err)
	}
	var linkedProject struct {
		Scan struct {
			Skills []struct {
				ID             string `json:"id"`
				LibrarySkillID string `json:"librarySkillId"`
			} `json:"skills"`
		} `json:"scan"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/projects/migration-project", nil, http.StatusOK, &linkedProject)
	if len(linkedProject.Scan.Skills) != 1 || linkedProject.Scan.Skills[0].LibrarySkillID != linked.Skill.ID {
		t.Fatalf("linked project Library marker = %#v", linkedProject.Scan.Skills)
	}
	requestJSON(t, handler, http.MethodPost, "/api/projects/migration-project/skills/linked-project-skill/migrate", map[string]any{
		"agent": "claude", "mode": "copy",
	}, http.StatusBadRequest, nil)
	linkedDocument := filepath.Join(linkedSource, "SKILL.md")
	if err := os.WriteFile(linkedDocument, []byte("---\nname: linked-project-skill\ndescription: Updated linked project Skill\n---\n\nUpdated body.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var library []domain.Skill
	requestJSON(t, handler, http.MethodGet, "/api/skills", nil, http.StatusOK, &library)
	var refreshedLinked domain.Skill
	for _, value := range library {
		if value.ID == linked.Skill.ID {
			refreshedLinked = value
			break
		}
	}
	if refreshedLinked.ID == "" || refreshedLinked.Hash == linked.Skill.Hash || refreshedLinked.Description != "Updated linked project Skill" {
		t.Fatalf("refreshed linked Library Skill = %#v", library)
	}
	if err := os.RemoveAll(linkedSource); err != nil {
		t.Fatal(err)
	}
	var fallbackDetails librarySkillDetails
	requestJSON(t, handler, http.MethodGet, "/api/skills/local/linked-project-skill", nil, http.StatusOK, &fallbackDetails)
	if fallbackDetails.Health != "missing" || !fallbackDetails.UsingFallback || fallbackDetails.EffectivePath != linked.Skill.SnapshotPath || !strings.Contains(fallbackDetails.Body, "# linked-project-skill") {
		t.Fatalf("missing linked Skill details = %#v", fallbackDetails)
	}
	requestJSON(t, handler, http.MethodPost, "/api/enable", map[string]any{
		"skills": []string{linked.Skill.ID}, "agents": []string{"codex"}, "mode": "auto",
	}, http.StatusOK, nil)
	userLink := filepath.Join(filepath.Dir(storage.Paths.Home), ".codex", "skills", "linked-project-skill")
	resolvedFallback, err := filepath.EvalSymlinks(userLink)
	expectedFallback, fallbackErr := filepath.EvalSymlinks(linked.Skill.SnapshotPath)
	if err != nil || fallbackErr != nil || filepath.Clean(resolvedFallback) != filepath.Clean(expectedFallback) {
		t.Fatalf("fallback deployment = %q, err=%v", resolvedFallback, err)
	}
	var detached struct {
		Skill domain.Skill `json:"skill"`
	}
	requestJSON(t, handler, http.MethodPost, "/api/skills/detach", map[string]any{"skill": linked.Skill.ID}, http.StatusOK, &detached)
	if detached.Skill.Mode != domain.ModeCopy || detached.Skill.SnapshotPath != "" || detached.Skill.ProjectRoot == "" || detached.Skill.ProjectPath == "" {
		t.Fatalf("detached linked Skill = %#v", detached.Skill)
	}
	resolvedDetached, err := filepath.EvalSymlinks(userLink)
	expectedDetached, detachedErr := filepath.EvalSymlinks(detached.Skill.Path)
	if err != nil || detachedErr != nil || filepath.Clean(resolvedDetached) != filepath.Clean(expectedDetached) {
		t.Fatalf("detached deployment = %q, err=%v", resolvedDetached, err)
	}
	requestJSON(t, handler, http.MethodPost, "/api/disable", map[string]any{
		"skills": []string{linked.Skill.ID}, "agents": []string{"codex"},
	}, http.StatusOK, nil)
	requestJSON(t, handler, http.MethodDelete, "/api/skills/local/linked-project-skill", nil, http.StatusOK, nil)

	makeProjectSkill(t, projectPath, "claude", "moved-project-skill")
	makeProjectSkill(t, projectPath, "codex", "moved-project-skill")
	var moved struct {
		Skill        domain.Skill `json:"skill"`
		RemovedPaths []string     `json:"removedPaths"`
	}
	requestJSON(t, handler, http.MethodPost, "/api/projects/migration-project/skills/moved-project-skill/migrate", map[string]any{
		"agent": "codex", "mode": "copy", "removeSource": true,
	}, http.StatusCreated, &moved)
	if moved.Skill.ID != "local/moved-project-skill" || moved.Skill.Mode != domain.ModeCopy || len(moved.RemovedPaths) != 2 {
		t.Fatalf("moved migration = %#v", moved)
	}
	if _, err := os.Stat(filepath.Join(moved.Skill.Path, "SKILL.md")); err != nil {
		t.Fatalf("copied Library snapshot: %v", err)
	}
	for _, agent := range []string{"claude", "codex"} {
		if _, err := os.Lstat(filepath.Join(projectPath, "."+agent, "skills", "moved-project-skill")); !os.IsNotExist(err) {
			t.Fatalf("%s project source still exists: %v", agent, err)
		}
	}

	makeProjectSkill(t, projectPath, "claude", "diverged-project-skill")
	makeProjectSkill(t, projectPath, "codex", "diverged-project-skill")
	diverged := filepath.Join(projectPath, ".codex", "skills", "diverged-project-skill", "SKILL.md")
	if err := os.WriteFile(diverged, []byte("---\nname: diverged-project-skill\ndescription: Different project copy\n---\n\nDifferent body.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	requestJSON(t, handler, http.MethodPost, "/api/projects/migration-project/skills/diverged-project-skill/migrate", map[string]any{
		"agent": "claude", "mode": "copy", "removeSource": true,
	}, http.StatusBadRequest, nil)
	if _, err := catalog.New(storage).ResolveLibrary("local/diverged-project-skill"); err == nil {
		t.Fatal("diverged Skill was imported despite rejected move")
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
	content := "---\nname: " + name + "\ndescription: Review changes safely\n---\n\nFollow the review workflow.\n"
	if err := os.WriteFile(filepath.Join(directory, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return directory
}

func makeProjectSkill(t *testing.T, projectPath, agent, name string) {
	t.Helper()
	makeProjectSkillAt(t, projectPath, filepath.Join("."+agent, "skills"), name)
}

func makeProjectSkillAt(t *testing.T, projectPath, skillsRelative, name string) {
	t.Helper()
	directory := filepath.Join(projectPath, skillsRelative, name)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: Project scan test Skill\n---\n\n# " + name + "\n"
	if err := os.WriteFile(filepath.Join(directory, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProjectSkillRootsDiscoversNestedAgentDirectories(t *testing.T) {
	projectRoot := t.TempDir()
	makeProjectSkill(t, projectRoot, "claude", "claude-skill")
	makeProjectSkillAt(t, projectRoot, filepath.Join(".codeium", "windsurf", "skills"), "windsurf-skill")
	makeProjectSkillAt(t, projectRoot, filepath.Join(".config", "opencode", "skills"), "opencode-skill")

	roots, err := projectSkillRoots(projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	labels := make(map[string]string)
	paths := make(map[string]string)
	for _, root := range roots {
		labels[root.ID] = root.Label
		paths[root.ID] = root.Path
	}
	if len(roots) != 3 {
		t.Fatalf("project Skill roots = %#v", roots)
	}
	if labels["claude"] != "Claude Code" || labels["windsurf"] != "Windsurf" || labels["opencode"] != "OpenCode" {
		t.Fatalf("project Skill root labels = %#v", labels)
	}
	if paths["windsurf"] != filepath.Join(projectRoot, ".codeium", "windsurf", "skills") {
		t.Fatalf("Windsurf Skill root path = %q", paths["windsurf"])
	}
	if paths["opencode"] != filepath.Join(projectRoot, ".config", "opencode", "skills") {
		t.Fatalf("OpenCode Skill root path = %q", paths["opencode"])
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

func createGitRepository(t *testing.T, name string) string {
	t.Helper()
	repository := t.TempDir()
	runGitTest(t, repository, "init", "-b", "main")
	makeRepositorySkill(t, repository, filepath.Join("skills", name), name)
	runGitTest(t, repository, "add", ".")
	runGitTest(t, repository, "-c", "user.name=skm-test", "-c", "user.email=skm@example.invalid", "commit", "-m", "initial")
	return repository
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

func fetchEmbeddedAsset(t *testing.T, handler http.Handler, path string) string {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET %s = %d, body=%s", path, recorder.Code, recorder.Body.String())
	}
	return recorder.Body.String()
}

func hasTag(values []tagCount, name string) bool {
	return findTagCount(values, name) != nil
}

func findTagCount(values []tagCount, name string) *tagCount {
	for _, value := range values {
		if value.Name == name {
			result := value
			return &result
		}
	}
	return nil
}
