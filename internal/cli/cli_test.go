package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/store"
)

func TestCLIAddTagEnableDisableAndPlan(t *testing.T) {
	root, userHome, project, skmHome := cliPaths(t)
	skillPath := filepath.Join(root, "review")
	writeCLISkill(t, skillPath, "review")
	runCLI(t, "--home", skmHome, "--project", project, "add", skillPath, "--tag", "backend")
	out := runCLI(t, "--home", skmHome, "--project", project, "--json", "list", "--tag", "backend")
	var listEnvelope struct {
		Success bool `json:"success"`
		Data    []struct {
			ID   string   `json:"id"`
			Tags []string `json:"tags"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &listEnvelope); err != nil {
		t.Fatal(err)
	}
	if !listEnvelope.Success || len(listEnvelope.Data) != 1 || listEnvelope.Data[0].ID != "local/review" {
		t.Fatalf("unexpected list response: %s", out)
	}

	runCLI(t, "--home", skmHome, "--project", project, "enable", "--tag", "backend", "--agent", "claude,codex")
	for _, target := range []string{
		filepath.Join(userHome, ".claude", "skills", "review"),
		filepath.Join(userHome, ".codex", "skills", "review"),
	} {
		if info, err := os.Lstat(target); err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("target %s is not a symlink: info=%v err=%v", target, info, err)
		}
	}
	out = runCLI(t, "--home", skmHome, "--project", project, "--json", "plan")
	assertPlanStatuses(t, out, 2, "unchanged")

	runCLI(t, "--home", skmHome, "--project", project, "disable", "review")
	if _, err := os.Lstat(filepath.Join(userHome, ".codex", "skills", "review")); !os.IsNotExist(err) {
		t.Fatalf("disabled target still exists: %v", err)
	}
	out = runCLI(t, "--home", skmHome, "--project", project, "list")
	if !bytes.Contains(out, []byte("local/review")) {
		t.Fatalf("disable removed Library Skill:\n%s", out)
	}
}

func TestCLIUpdateSkillRefreshesUserDeployment(t *testing.T) {
	root, userHome, project, skmHome := cliPaths(t)
	original := filepath.Join(root, "original-review")
	writeCLISkill(t, original, "editable-review")
	runCLI(t, "--home", skmHome, "--user-home", userHome, "--project", project, "add", original)
	runCLI(t, "--home", skmHome, "--user-home", userHome, "--project", project, "enable", "local/editable-review", "--agent", "codex")
	storage := openTestStore(t, skmHome, userHome, project)
	library, err := storage.LoadCatalog()
	if err != nil || len(library.Skills) != 1 {
		t.Fatalf("Library = %#v, %v", library, err)
	}
	before := library.Skills[0]

	replacement := filepath.Join(root, "replacement-review")
	writeCLISkill(t, replacement, "editable-review")
	path := filepath.Join(replacement, "SKILL.md")
	content := strings.Replace(readTestFile(t, path), "CLI integration Skill", "Updated CLI Skill", 1)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(replacement, "helper.txt"), []byte("helper\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := runCLI(t, "--home", skmHome, "--user-home", userHome, "--project", project, "update", "local/editable-review", replacement, "--base-hash", before.Hash)
	if !bytes.Contains(out, []byte("Updated local/editable-review")) {
		t.Fatalf("update output = %s", out)
	}
	library, err = storage.LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	after := library.Skills[0]
	if after.Hash == before.Hash || after.Description != "Updated CLI Skill" {
		t.Fatalf("updated Skill = %#v", after)
	}
	if _, err := os.Stat(filepath.Join(after.Path, "helper.txt")); err != nil {
		t.Fatalf("updated auxiliary file is missing: %v", err)
	}
	target := filepath.Join(userHome, ".codex", "skills", "editable-review")
	linked, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	wantLinked, err := filepath.EvalSymlinks(after.Path)
	if err != nil || linked != wantLinked {
		t.Fatalf("updated target = %q, %v; want %q", linked, err, wantLinked)
	}
	failure := runCLIFailure(t, "--home", skmHome, "--user-home", userHome, "--project", project, "update", "local/editable-review", replacement, "--base-hash", before.Hash)
	if !bytes.Contains(failure, []byte("changed since editing started")) {
		t.Fatalf("stale update error = %s", failure)
	}
}

func TestCLIPromptLifecycleAndRendering(t *testing.T) {
	root, _, project, skmHome := cliPaths(t)
	runCLI(t, "--home", skmHome, "--project", project, "prompt", "create", "summary", "--description", "Summarize a topic", "--variable", "topic", "--body", "Summarize {{topic}}")
	createdOutput := runCLI(t, "--home", skmHome, "--project", project, "prompt", "render", "summary", "--var", "topic=testing")
	if !bytes.Contains(createdOutput, []byte("Summarize testing")) {
		t.Fatalf("created Prompt render: %s", createdOutput)
	}
	runCLI(t, "--home", skmHome, "--project", project, "prompt", "remove", "summary")

	promptPath := filepath.Join(root, "PROMPT.md")
	writeCLIPrompt(t, promptPath, "code-review", "Review carefully")
	runCLI(t, "--home", skmHome, "--project", project, "prompt", "validate", promptPath)
	runCLI(t, "--home", skmHome, "--project", project, "prompt", "add", promptPath)

	out := runCLI(t, "--home", skmHome, "--project", project, "prompt", "list", "--tag", "review")
	if !bytes.Contains(out, []byte("local/code-review")) {
		t.Fatalf("Prompt list: %s", out)
	}
	codePath := filepath.Join(root, "main.go")
	if err := os.WriteFile(codePath, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	out = runCLI(t, "--home", skmHome, "--project", project, "prompt", "render", "code-review", "--var", "language=Go", "--var-file", "code="+codePath)
	if !bytes.Contains(out, []byte("Review this Go code")) || !bytes.Contains(out, []byte("package main")) {
		t.Fatalf("Prompt render: %s", out)
	}

	updated := strings.Replace(readTestFile(t, promptPath), "Review carefully", "Review thoroughly", 1)
	if err := os.WriteFile(promptPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	runCLI(t, "--home", skmHome, "--project", project, "prompt", "update", "code-review", promptPath)
	out = runCLI(t, "--home", skmHome, "--project", project, "--json", "prompt", "show", "code-review")
	if !bytes.Contains(out, []byte("Review thoroughly")) || !bytes.Contains(out, []byte(`"content"`)) {
		t.Fatalf("Prompt show: %s", out)
	}
	exported := filepath.Join(root, "exported", "PROMPT.md")
	runCLI(t, "--home", skmHome, "--project", project, "prompt", "export", "code-review", "--output", exported)
	if !strings.Contains(readTestFile(t, exported), "Review thoroughly") {
		t.Fatal("exported Prompt is stale")
	}
	runCLI(t, "--home", skmHome, "--project", project, "prompt", "remove", "code-review")
	if out := runCLI(t, "--home", skmHome, "--project", project, "prompt", "list"); bytes.Contains(out, []byte("code-review")) {
		t.Fatalf("removed Prompt remained: %s", out)
	}
}

func TestCLIUserHomeOverrideKeepsAgentTargetsIsolated(t *testing.T) {
	root, realUserHome, project, _ := cliPaths(t)
	isolationRoot := filepath.Join(root, "isolated-user")
	isolationSKMHome := filepath.Join(isolationRoot, ".skm")
	skillPath := filepath.Join(root, "isolated-skill")
	writeCLISkill(t, skillPath, "isolated-skill")

	runCLI(t,
		"--home", isolationSKMHome,
		"--user-home", isolationRoot,
		"--project", project,
		"add", skillPath,
	)
	runCLI(t,
		"--home", isolationSKMHome,
		"--user-home", isolationRoot,
		"--project", project,
		"enable", "local/isolated-skill", "--agent", "codex",
	)

	isolationTarget := filepath.Join(isolationRoot, ".codex", "skills", "isolated-skill")
	if info, err := os.Lstat(isolationTarget); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("isolated target %s is not a symlink: info=%v err=%v", isolationTarget, info, err)
	}
	if _, err := os.Lstat(filepath.Join(realUserHome, ".codex", "skills", "isolated-skill")); !os.IsNotExist(err) {
		t.Fatalf("real user target was modified: %v", err)
	}
}

func TestCLIProjectVendorRetainsPersonalLibrary(t *testing.T) {
	root, _, project, skmHome := cliPaths(t)
	skillPath := filepath.Join(root, "release")
	writeCLISkill(t, skillPath, "release")
	runCLI(t, "--home", skmHome, "--project", project, "add", skillPath, "--tag", "release")
	out := runCLI(t, "--home", skmHome, "--project", project, "project", "vendor", "local/release", "--agent", "claude,codex")
	if !bytes.Contains(out, []byte("personal Library copy retained")) {
		t.Fatalf("vendor output:\n%s", out)
	}
	for _, target := range []string{
		filepath.Join(project, ".claude", "skills", "release"),
		filepath.Join(project, ".codex", "skills", "release"),
		filepath.Join(project, ".skm", "skills", "release", "SKILL.md"),
	} {
		if _, err := os.Lstat(target); err != nil {
			t.Fatalf("vendored target %s missing: %v", target, err)
		}
	}
	library := runCLI(t, "--home", skmHome, "--project", project, "list")
	if !bytes.Contains(library, []byte("local/release")) {
		t.Fatalf("personal copy missing:\n%s", library)
	}
	manifest, err := os.ReadFile(filepath.Join(project, ".skm", "project.yaml"))
	if err != nil || !bytes.Contains(manifest, []byte("forkedFrom: local/release")) {
		t.Fatalf("vendor manifest: %v\n%s", err, manifest)
	}
}

func TestCLIRegisteredProjectLinkAndCopy(t *testing.T) {
	root, _, currentProject, skmHome := cliPaths(t)
	registeredProject := filepath.Join(root, "registered-project")
	if err := os.MkdirAll(registeredProject, 0o755); err != nil {
		t.Fatal(err)
	}
	defaultNamedProject := filepath.Join(root, "default-named-project")
	if err := os.MkdirAll(defaultNamedProject, 0o755); err != nil {
		t.Fatal(err)
	}
	linkSkill := filepath.Join(root, "link-skill")
	copySkill := filepath.Join(root, "copy-skill")
	writeCLISkill(t, linkSkill, "link-skill")
	writeCLISkill(t, copySkill, "copy-skill")
	runCLI(t, "--home", skmHome, "--project", currentProject, "add", linkSkill)
	runCLI(t, "--home", skmHome, "--project", currentProject, "add", copySkill)
	runCLI(t, "--home", skmHome, "--project", currentProject, "project", "add", defaultNamedProject)
	out := runCLI(t, "--home", skmHome, "--project", currentProject, "--json", "project", "list")
	if !bytes.Contains(out, []byte(`"id":"default-named-project"`)) {
		t.Fatalf("project add did not default to the root directory name: %s", out)
	}
	runCLI(t, "--home", skmHome, "--project", currentProject, "project", "add", registeredProject, "--name", "A")
	if _, err := os.Stat(filepath.Join(registeredProject, ".skm")); !os.IsNotExist(err) {
		t.Fatalf("project add wrote project metadata: %v", err)
	}

	out = runCLI(t, "--home", skmHome, "--project", currentProject, "--json", "project", "list")
	if !bytes.Contains(out, []byte(`"id":"A"`)) || !bytes.Contains(out, []byte(`"activationCount":0`)) {
		t.Fatalf("registered project list: %s", out)
	}

	runCLI(t, "--home", skmHome, "--project", currentProject, "project", "link", "A", "local/link-skill", "--agent", "claude")
	linkTarget := filepath.Join(registeredProject, ".claude", "skills", "link-skill")
	info, err := os.Lstat(linkTarget)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("project link target is not a symlink: info=%v err=%v", info, err)
	}

	out = runCLI(t, "--home", skmHome, "--project", currentProject, "project", "link", "A", "local/link-skill", "--agent", "claude")
	if !bytes.Contains(out, []byte("unchanged")) {
		t.Fatalf("repeated project link was not idempotent: %s", out)
	}
	stderr := runCLIFailure(t, "--home", skmHome, "--project", currentProject, "project", "copy", "A", "local/link-skill", "--agent", "codex")
	if !bytes.Contains(stderr, []byte("already uses mode symlink")) {
		t.Fatalf("mixed project deployment mode was accepted: %s", stderr)
	}

	runCLI(t, "--home", skmHome, "--project", currentProject, "project", "copy", "A", "local/copy-skill", "--agent", "codex")
	copyTarget := filepath.Join(registeredProject, ".codex", "skills", "copy-skill")
	info, err = os.Lstat(copyTarget)
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("project copy target is a symlink or missing: info=%v err=%v", info, err)
	}
	if _, err := os.Stat(filepath.Join(copyTarget, "SKILL.md")); err != nil {
		t.Fatalf("copied Skill content is missing: %v", err)
	}

	out = runCLI(t, "--home", skmHome, "--project", currentProject, "--json", "project", "status", "A")
	if !bytes.Contains(out, []byte(`"placement":"project"`)) || !bytes.Contains(out, []byte(`"status":"unchanged"`)) {
		t.Fatalf("project status: %s", out)
	}

	runCLI(t, "--home", skmHome, "--project", currentProject, "project", "unlink", "A", "local/link-skill", "--agent", "claude")
	if _, err := os.Lstat(linkTarget); !os.IsNotExist(err) {
		t.Fatalf("project link target still exists after unlink: %v", err)
	}
	runCLI(t, "--home", skmHome, "--project", currentProject, "project", "unlink", "A", "local/copy-skill", "--agent", "codex")
	runCLI(t, "--home", skmHome, "--project", currentProject, "project", "unregister", "A")
	if _, err := os.Stat(registeredProject); err != nil {
		t.Fatalf("unregister removed project files: %v", err)
	}
}

func TestCLIRegisteredProjectRejectsSameNameConflict(t *testing.T) {
	root, _, currentProject, skmHome := cliPaths(t)
	registeredProject := filepath.Join(root, "conflict-project")
	if err := os.MkdirAll(registeredProject, 0o755); err != nil {
		t.Fatal(err)
	}
	firstPath := filepath.Join(root, "first-project-skill")
	secondPath := filepath.Join(root, "second-project-skill")
	writeCLISkill(t, firstPath, "review")
	writeCLISkill(t, secondPath, "review")
	runCLI(t, "--home", skmHome, "--project", currentProject, "add", firstPath, "--source", "team-a")
	runCLI(t, "--home", skmHome, "--project", currentProject, "add", secondPath, "--source", "team-b")
	runCLI(t, "--home", skmHome, "--project", currentProject, "project", "add", registeredProject, "--name", "A")
	runCLI(t, "--home", skmHome, "--project", currentProject, "project", "link", "A", "team-a/review", "--agent", "codex")
	target := filepath.Join(registeredProject, ".codex", "skills", "review")
	before, err := os.Readlink(target)
	if err != nil {
		t.Fatal(err)
	}
	stderr := runCLIFailure(t, "--home", skmHome, "--project", currentProject, "project", "link", "A", "team-b/review", "--agent", "codex")
	if !bytes.Contains(stderr, []byte("multiple Skills target")) {
		t.Fatalf("unexpected project conflict error: %s", stderr)
	}
	after, err := os.Readlink(target)
	if err != nil || after != before {
		t.Fatalf("project conflict changed target: before=%s after=%s err=%v", before, after, err)
	}
}

func TestCLIProjectRequireCanBeSatisfiedByUserActivation(t *testing.T) {
	root, userHome, project, skmHome := cliPaths(t)
	skillPath := filepath.Join(root, "review")
	writeCLISkill(t, skillPath, "review")
	runCLI(t, "--home", skmHome, "--project", project, "add", skillPath, "--source", "team", "--tag", "review")
	storage := openTestStore(t, skmHome, userHome, project)
	library, err := storage.LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	library.Skills[0].Revision = "abc123"
	library.Skills[0].SourcePath = "skills/review"
	if err := storage.SaveCatalog(library); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveSources(domain.Sources{Sources: []domain.Source{{Name: "team", URL: "git@example.invalid:team.git", Ref: "main", Revision: "abc123"}}}); err != nil {
		t.Fatal(err)
	}
	runCLI(t, "--home", skmHome, "--project", project, "enable", "team/review", "--agent", "codex")
	out := runCLI(t, "--home", skmHome, "--project", project, "project", "require", "team/review", "--agent", "codex")
	if !bytes.Contains(out, []byte("Satisfied by user activation: team/review:codex")) {
		t.Fatalf("require was not satisfied by user:\n%s", out)
	}
	if _, err := os.Lstat(filepath.Join(project, ".codex", "skills", "review")); !os.IsNotExist(err) {
		t.Fatalf("redundant project link was created: %v", err)
	}
	manifest, err := os.ReadFile(filepath.Join(project, ".skm", "project.yaml"))
	if err != nil || !bytes.Contains(manifest, []byte("team/review")) || !bytes.Contains(manifest, []byte("revision: abc123")) {
		t.Fatalf("project requirement: %v\n%s", err, manifest)
	}
}

func TestCLIUsesGeneralTagByDefault(t *testing.T) {
	root, _, project, skmHome := cliPaths(t)
	skillPath := filepath.Join(root, "default-skill")
	writeCLISkill(t, skillPath, "default-skill")
	runCLI(t, "--home", skmHome, "--project", project, "add", skillPath)
	out := runCLI(t, "--home", skmHome, "--project", project, "list", "--tag", "general")
	if !bytes.Contains(out, []byte("local/default-skill")) {
		t.Fatalf("default tag query did not return Skill:\n%s", out)
	}
}

func TestCLIInitWithProjectWritesProjectState(t *testing.T) {
	_, _, project, skmHome := cliPaths(t)
	runCLI(t, "--home", skmHome, "--project", project, "init", "--with-project")
	for _, path := range []string{
		filepath.Join(project, ".skm", "project.yaml"),
		filepath.Join(project, ".skm", "lock.yaml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("project state %s missing: %v", path, err)
		}
	}
}

func TestCLIVersionUsesInjectedReleaseVersion(t *testing.T) {
	previous := Version
	Version = "v1.2.3"
	t.Cleanup(func() { Version = previous })

	out := runCLI(t, "version")
	if string(out) != "1.2.3\n" {
		t.Fatalf("version output = %q", out)
	}
}

func TestCLIEnableRejectsSameNameBeforeChangingState(t *testing.T) {
	root, userHome, project, skmHome := cliPaths(t)
	firstPath := filepath.Join(root, "first")
	secondPath := filepath.Join(root, "second")
	writeCLISkill(t, firstPath, "review")
	writeCLISkill(t, secondPath, "review")
	runCLI(t, "--home", skmHome, "--project", project, "add", firstPath, "--source", "team-a")
	runCLI(t, "--home", skmHome, "--project", project, "add", secondPath, "--source", "team-b")
	runCLI(t, "--home", skmHome, "--project", project, "enable", "team-a/review", "--agent", "codex")

	target := filepath.Join(userHome, ".codex", "skills", "review")
	beforeLink, err := os.Readlink(target)
	if err != nil {
		t.Fatal(err)
	}
	stderr := runCLIFailure(t, "--home", skmHome, "--project", project, "enable", "team-b/review", "--agent", "codex")
	if !bytes.Contains(stderr, []byte("multiple Skills target")) {
		t.Fatalf("unexpected conflict error: %s", stderr)
	}
	afterLink, err := os.Readlink(target)
	if err != nil {
		t.Fatal(err)
	}
	if afterLink != beforeLink {
		t.Fatalf("conflicting enable changed target: before=%s after=%s", beforeLink, afterLink)
	}
	storage := openTestStore(t, skmHome, userHome, project)
	state, err := storage.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Activations) != 1 || state.Activations[0].SkillID != "team-a/review" {
		t.Fatalf("conflicting Activation was saved: %#v", state.Activations)
	}
}

func TestCLIRemoveDeletesUnreferencedSnapshot(t *testing.T) {
	root, userHome, project, skmHome := cliPaths(t)
	skillPath := filepath.Join(root, "remove-me")
	writeCLISkill(t, skillPath, "remove-me")
	runCLI(t, "--home", skmHome, "--project", project, "add", skillPath)
	storage := openTestStore(t, skmHome, userHome, project)
	catalog, err := storage.LoadCatalog()
	if err != nil || len(catalog.Skills) != 1 {
		t.Fatalf("catalog = %#v, err=%v", catalog, err)
	}
	objectPath := catalog.Skills[0].Path

	out := runCLI(t, "--home", skmHome, "--project", project, "remove", "local/remove-me")
	if !bytes.Contains(out, []byte("deleted its snapshot")) {
		t.Fatalf("remove output: %s", out)
	}
	if _, err := os.Lstat(objectPath); !os.IsNotExist(err) {
		t.Fatalf("removed snapshot still exists: %v", err)
	}
}

func TestCLIRemoveRetainsSnapshotReferencedByAnotherLibrarySkill(t *testing.T) {
	root, _, project, skmHome := cliPaths(t)
	skillPath := filepath.Join(root, "shared")
	writeCLISkill(t, skillPath, "shared")
	runCLI(t, "--home", skmHome, "--project", project, "add", skillPath, "--source", "team-a")
	runCLI(t, "--home", skmHome, "--project", project, "add", skillPath, "--source", "team-b")

	out := runCLI(t, "--home", skmHome, "--project", project, "remove", "team-a/shared")
	if !bytes.Contains(out, []byte("snapshot retained because it is still referenced")) {
		t.Fatalf("first remove output: %s", out)
	}
	out = runCLI(t, "--home", skmHome, "--project", project, "remove", "team-b/shared")
	if !bytes.Contains(out, []byte("deleted its snapshot")) {
		t.Fatalf("second remove output: %s", out)
	}
}

func TestCLIDisableReportsProjectActivationAndRemoveExplainsHowToRemoveIt(t *testing.T) {
	root, userHome, project, skmHome := cliPaths(t)
	skillPath := filepath.Join(root, "project-review")
	writeCLISkill(t, skillPath, "project-review")
	runCLI(t, "--home", skmHome, "--project", project, "add", skillPath, "--source", "team")

	storage := openTestStore(t, skmHome, userHome, project)
	library, err := storage.LoadCatalog()
	if err != nil || len(library.Skills) != 1 {
		t.Fatalf("library = %#v, err=%v", library, err)
	}
	value := library.Skills[0]
	projectCatalog := domain.Catalog{Dependencies: []domain.ProjectDependency{{
		ID: value.ID, Name: value.Name, Source: value.Source, Hash: value.Hash,
		Agents: []domain.Agent{domain.AgentClaude, domain.AgentCodex}, Mode: domain.ModeAuto,
	}}}
	if err := storage.SaveProjectCatalog(projectCatalog); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveProjectLock(projectCatalog); err != nil {
		t.Fatal(err)
	}
	state := domain.State{Activations: []domain.Activation{{
		SkillID:     "team/project-review",
		Name:        "project-review",
		Placement:   domain.PlacementProject,
		ProjectRoot: project,
		Agents:      []domain.Agent{domain.AgentClaude, domain.AgentCodex},
		Mode:        domain.ModeAuto,
	}}}
	if err := storage.SaveState(state); err != nil {
		t.Fatal(err)
	}

	out := runCLI(t, "--home", skmHome, "--project", project, "disable", "team/project-review")
	if !bytes.Contains(out, []byte("Disabled user Activation(s) for 0 Library Skill(s)")) ||
		!bytes.Contains(out, []byte("remains enabled by project "+project)) ||
		!bytes.Contains(out, []byte("project remove team/project-review")) {
		t.Fatalf("disable output did not explain project Activation:\n%s", out)
	}

	stderr := runCLIFailure(t, "--home", skmHome, "--project", project, "remove", "team/project-review")
	if !bytes.Contains(stderr, []byte("is enabled by project "+project)) ||
		!bytes.Contains(stderr, []byte("project remove team/project-review first")) {
		t.Fatalf("remove error did not explain project Activation:\n%s", stderr)
	}

	runCLI(t, "--home", skmHome, "--project", project, "project", "remove", "team/project-review")
	out = runCLI(t, "--home", skmHome, "--project", project, "remove", "team/project-review")
	if !bytes.Contains(out, []byte("deleted its snapshot")) {
		t.Fatalf("remove output after project remove: %s", out)
	}
	if _, err := os.Lstat(value.Path); !os.IsNotExist(err) {
		t.Fatalf("removed snapshot still exists: %v", err)
	}
}

func TestCLIPruneDryRunThenDeletesOrphan(t *testing.T) {
	root, userHome, project, skmHome := cliPaths(t)
	skillPath := filepath.Join(root, "orphan")
	writeCLISkill(t, skillPath, "orphan")
	runCLI(t, "--home", skmHome, "--project", project, "add", skillPath)
	storage := openTestStore(t, skmHome, userHome, project)
	catalog, err := storage.LoadCatalog()
	if err != nil || len(catalog.Skills) != 1 {
		t.Fatalf("catalog = %#v, err=%v", catalog, err)
	}
	objectPath := catalog.Skills[0].Path
	if err := storage.SaveCatalog(domain.Catalog{}); err != nil {
		t.Fatal(err)
	}

	out := runCLI(t, "--home", skmHome, "--project", project, "prune", "--dry-run")
	if !bytes.Contains(out, []byte("Would prune 1 unreferenced snapshot")) {
		t.Fatalf("prune dry-run output: %s", out)
	}
	if _, err := os.Stat(objectPath); err != nil {
		t.Fatalf("dry-run removed object: %v", err)
	}
	out = runCLI(t, "--home", skmHome, "--project", project, "prune")
	if !bytes.Contains(out, []byte("Pruned 1 unreferenced snapshot")) {
		t.Fatalf("prune output: %s", out)
	}
	if _, err := os.Lstat(objectPath); !os.IsNotExist(err) {
		t.Fatalf("orphan still exists: %v", err)
	}
}

func TestCLISourceRemoveDeletesOrphanedCheckout(t *testing.T) {
	_, _, project, skmHome := cliPaths(t)
	checkoutPath := filepath.Join(skmHome, "sources", "team")
	if err := os.MkdirAll(checkoutPath, 0o755); err != nil {
		t.Fatal(err)
	}

	out := runCLI(t, "--home", skmHome, "--project", project, "source", "remove", "team")
	if !bytes.Contains(out, []byte("Removed orphaned source checkout team")) {
		t.Fatalf("source remove output: %s", out)
	}
	if _, err := os.Lstat(checkoutPath); !os.IsNotExist(err) {
		t.Fatalf("orphaned checkout still exists: %v", err)
	}
}

func assertPlanStatuses(t *testing.T, raw []byte, count int, status string) {
	t.Helper()
	var envelope struct {
		Data struct {
			Operations []struct {
				Status string `json:"status"`
			} `json:"operations"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data.Operations) != count {
		t.Fatalf("unexpected plan: %s", raw)
	}
	for _, operation := range envelope.Data.Operations {
		if operation.Status != status {
			t.Fatalf("plan status is not %s: %s", status, raw)
		}
	}
}

func cliPaths(t *testing.T) (root, userHome, project, skmHome string) {
	t.Helper()
	root = t.TempDir()
	userHome = filepath.Join(root, "user")
	project = filepath.Join(root, "project")
	skmHome = filepath.Join(userHome, ".skm")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", userHome)
	return
}

func openTestStore(t *testing.T, skmHome, userHome, project string) *store.Store {
	t.Helper()
	storage, err := store.New(store.Paths{Home: skmHome, UserHome: userHome, ProjectRoot: project})
	if err != nil {
		t.Fatal(err)
	}
	return storage
}

func runCLI(t *testing.T, args ...string) []byte {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := Execute(args, &stdout, &stderr); code != 0 {
		t.Fatalf("skm %v failed (%d): %s", args, code, stderr.String())
	}
	return stdout.Bytes()
}

func runCLIFailure(t *testing.T, args ...string) []byte {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := Execute(args, &stdout, &stderr); code == 0 {
		t.Fatalf("skm %v unexpectedly succeeded: %s", args, stdout.String())
	}
	return stderr.Bytes()
}

func writeCLISkill(t *testing.T, path, name string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: CLI integration Skill\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeCLIPrompt(t *testing.T, path, name, description string) {
	t.Helper()
	content := "---\nname: " + name + "\ndescription: " + description + "\ntags: [review]\nvariables:\n  - name: language\n    type: select\n    required: true\n    options: [Go, Swift]\n  - name: code\n    type: multiline\n    required: true\n---\nReview this {{language}} code:\n\n{{code}}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
