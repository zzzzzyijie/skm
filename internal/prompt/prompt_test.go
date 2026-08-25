package prompt

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zzzzzyijie/skm/internal/store"
)

const validPrompt = `---
name: code-review
description: Review code safely
tags: [review]
variables:
  - name: language
    type: select
    required: true
    options: [Go, Swift]
  - name: code
    type: multiline
    required: true
---
Review this {{language}} code:

{{code}}
`

func TestParseAndRender(t *testing.T) {
	document, err := Parse([]byte(validPrompt))
	if err != nil {
		t.Fatal(err)
	}
	if document.Name != "code-review" || len(document.Variables) != 2 || document.Hash == "" {
		t.Fatalf("document = %#v", document)
	}
	result, err := Render(document, map[string]string{"language": "Go", "code": "package main"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.MissingVariables) != 0 || !strings.Contains(result.Content, "Review this Go code") || !strings.Contains(result.Content, "package main") {
		t.Fatalf("render result = %#v", result)
	}
	missing, err := Render(document, map[string]string{"language": "Swift"})
	if err != nil || !reflect.DeepEqual(missing.MissingVariables, []string{"code"}) {
		t.Fatalf("missing result = %#v, err=%v", missing, err)
	}
}

func TestParseSupportsUnicodeTags(t *testing.T) {
	content := strings.Replace(validPrompt, "tags: [review]", "tags: [简小知]", 1)
	document, err := Parse([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(document.Tags, []string{"简小知"}) {
		t.Fatalf("tags = %#v", document.Tags)
	}
}

func TestParseRejectsUnsafeOrInvalidTemplates(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"missing frontmatter", "plain text"},
		{"invalid tag", "---\nname: bad\ndescription: bad\ntags: [Not Valid]\n---\nbody\n"},
		{"undeclared", "---\nname: bad\ndescription: bad\n---\n{{value}}\n"},
		{"secret default", "---\nname: bad\ndescription: bad\nvariables:\n  - name: token\n    type: secret\n    default: nope\n---\n{{token}}\n"},
		{"select options", "---\nname: bad\ndescription: bad\nvariables:\n  - name: mode\n    type: select\n---\n{{mode}}\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Parse([]byte(test.content)); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestManagerLifecycleAndHashConflict(t *testing.T) {
	root := t.TempDir()
	storage, err := store.New(store.Paths{Home: filepath.Join(root, ".skm"), UserHome: root, ProjectRoot: filepath.Join(root, "project")})
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Ensure(); err != nil {
		t.Fatal(err)
	}
	manager := New(storage)
	value, err := manager.Create(validPrompt, "local", nil)
	if err != nil {
		t.Fatal(err)
	}
	if value.ID != "local/code-review" || value.Tags[0] != "review" {
		t.Fatalf("created Prompt = %#v", value)
	}
	if _, err := os.Stat(filepath.Join(value.Path, "PROMPT.md")); err != nil {
		t.Fatal(err)
	}
	updatedContent := strings.Replace(validPrompt, "Review code safely", "Review code thoroughly", 1)
	updated, err := manager.Update(value.ID, updatedContent, value.Hash, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Hash == value.Hash || updated.Description != "Review code thoroughly" {
		t.Fatalf("updated Prompt = %#v", updated)
	}
	if _, err := manager.Update(value.ID, validPrompt, value.Hash, nil); err == nil {
		t.Fatal("stale Prompt update succeeded")
	}
	if _, err := manager.Remove(value.ID); err != nil {
		t.Fatal(err)
	}
	if prompts, err := manager.List(nil); err != nil || len(prompts) != 0 {
		t.Fatalf("Prompts after removal = %#v, err=%v", prompts, err)
	}
}

func TestManagerUsesPromptDefaultTag(t *testing.T) {
	root := t.TempDir()
	storage, err := store.New(store.Paths{Home: filepath.Join(root, ".skm"), UserHome: root, ProjectRoot: filepath.Join(root, "project")})
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Ensure(); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: untagged\ndescription: Uses the Prompt default tag\n---\nBody\n"
	value, err := New(storage).Create(content, "local", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(value.Tags, []string{"general"}) {
		t.Fatalf("tags = %#v, want Prompt defaults", value.Tags)
	}
}
