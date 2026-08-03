package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashDirChangesWithContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("hash is not deterministic")
	}
	if err := os.WriteFile(path, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if changed == first {
		t.Fatal("hash did not change")
	}
}

func TestHashDirIgnoringFinderMetadata(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	target := filepath.Join(root, "target")
	for _, directory := range []string{source, target} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "SKILL.md"), []byte("skill"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(target, ".DS_Store"), []byte("finder metadata"), 0o644); err != nil {
		t.Fatal(err)
	}

	sourceHash, err := HashDirIgnoringFinderMetadata(source)
	if err != nil {
		t.Fatal(err)
	}
	targetHash, err := HashDirIgnoringFinderMetadata(target)
	if err != nil {
		t.Fatal(err)
	}
	if sourceHash != targetHash {
		t.Fatalf("Finder metadata changed hash: %s != %s", sourceHash, targetHash)
	}

}

func TestCopyDirAtomicReplacesDestination(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "new"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "old"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyDirAtomic(source, destination); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, "new")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, "old")); !os.IsNotExist(err) {
		t.Fatalf("old content was not removed: %v", err)
	}
}
