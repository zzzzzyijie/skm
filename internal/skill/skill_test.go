package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSkill(t *testing.T) {
	dir := t.TempDir()
	content := "---\r\nname: code-review\r\ndescription: Review code safely\r\ncustom-field: keep-me\r\n---\r\n\r\n# Instructions\r\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "reference.md"), []byte("reference"), 0o644); err != nil {
		t.Fatal(err)
	}

	document, err := Validate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if document.Name != "code-review" || document.Description != "Review code safely" {
		t.Fatalf("unexpected document: %#v", document)
	}
	if document.Metadata["custom-field"] != "keep-me" {
		t.Fatalf("custom metadata not preserved: %#v", document.Metadata)
	}
	if document.Hash == "" || document.Files != 2 {
		t.Fatalf("hash/files not populated: %#v", document)
	}
}

func TestValidateRejectsInvalidFrontmatter(t *testing.T) {
	tests := map[string]string{
		"missing frontmatter": "# Skill\n",
		"missing description": "---\nname: valid\n---\nbody\n",
		"invalid name":        "---\nname: Invalid_Name\ndescription: test\n---\nbody\n",
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Validate(dir); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateRejectsEscapingSymlink(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "safe")
	if err := os.Symlink("../outside", filepath.Join(dir, "escape")); err != nil {
		t.Fatal(err)
	}
	_, err := Validate(dir)
	if err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("expected escaping symlink error, got %v", err)
	}
}

func writeSkill(t *testing.T, dir, name string) {
	t.Helper()
	content := "---\nname: " + name + "\ndescription: test skill\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
