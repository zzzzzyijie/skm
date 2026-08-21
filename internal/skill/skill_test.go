package skill

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractZIPFindsWrappedSkill(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "skill.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entries := map[string]string{
		"wrapped/SKILL.md":     "---\nname: archived-skill\ndescription: Imported from ZIP\n---\nbody\n",
		"wrapped/reference.md": "reference",
		"__MACOSX/._wrapped":   "ignored",
	}
	for name, content := range entries {
		entry, createErr := writer.Create(name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := entry.Write([]byte(content)); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	root, err := ExtractZIP(archivePath, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	document, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if document.Name != "archived-skill" || document.Files != 2 {
		t.Fatalf("unexpected extracted document: %#v", document)
	}
}

func TestExtractZIPRejectsUnsafeAndAmbiguousArchives(t *testing.T) {
	tests := []struct {
		name    string
		entries map[string]string
		want    string
	}{
		{name: "path traversal", entries: map[string]string{"../SKILL.md": "unsafe"}, want: "escapes"},
		{name: "multiple Skills", entries: map[string]string{
			"one/SKILL.md": "---\nname: one\ndescription: one\n---\n",
			"two/SKILL.md": "---\nname: two\ndescription: two\n---\n",
		}, want: "exactly one Skill"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archivePath := filepath.Join(t.TempDir(), "skill.zip")
			file, err := os.Create(archivePath)
			if err != nil {
				t.Fatal(err)
			}
			writer := zip.NewWriter(file)
			for name, content := range test.entries {
				entry, createErr := writer.Create(name)
				if createErr != nil {
					t.Fatal(createErr)
				}
				if _, writeErr := entry.Write([]byte(content)); writeErr != nil {
					t.Fatal(writeErr)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			_, err = ExtractZIP(archivePath, t.TempDir())
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ExtractZIP() error = %v, want %q", err, test.want)
			}
		})
	}
}

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
