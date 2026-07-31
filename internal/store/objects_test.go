package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zzzzzyijie/skm/internal/domain"
)

func TestDeleteObjectIfUnreferencedHonorsKnownReferences(t *testing.T) {
	storage := testStore(t)
	hash := repeatedHash('a')
	path := writeTestObject(t, storage, hash, "review", "body")

	if err := storage.SaveCatalog(domain.Catalog{Skills: []domain.Skill{{
		ID: "local/review", Name: "review", Hash: hash, Path: path,
	}}}); err != nil {
		t.Fatal(err)
	}
	result, err := storage.DeleteObjectIfUnreferenced(hash, "review")
	if err != nil || result.Deleted || len(result.References) != 1 {
		t.Fatalf("Library reference result = %#v, err=%v", result, err)
	}

	if err := storage.SaveCatalog(domain.Catalog{}); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveState(domain.State{Deployments: []domain.Deployment{{
		SkillID: "local/review", Agent: domain.AgentCodex, SourcePath: path,
	}}}); err != nil {
		t.Fatal(err)
	}
	result, err = storage.DeleteObjectIfUnreferenced(hash, "review")
	if err != nil || result.Deleted || len(result.References) != 1 {
		t.Fatalf("Deployment reference result = %#v, err=%v", result, err)
	}

	if err := storage.SaveState(domain.State{}); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveProjectCatalog(domain.Catalog{Dependencies: []domain.ProjectDependency{{
		ID: "team/review", Name: "review", Hash: hash,
	}}}); err != nil {
		t.Fatal(err)
	}
	result, err = storage.DeleteObjectIfUnreferenced(hash, "review")
	if err != nil || result.Deleted || len(result.References) != 1 {
		t.Fatalf("Project reference result = %#v, err=%v", result, err)
	}

	if err := storage.SaveProjectCatalog(domain.Catalog{}); err != nil {
		t.Fatal(err)
	}
	result, err = storage.DeleteObjectIfUnreferenced(hash, "review")
	if err != nil || !result.Deleted {
		t.Fatalf("unreferenced result = %#v, err=%v", result, err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("object still exists: %v", err)
	}
	if _, err := os.Lstat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatalf("empty hash directory still exists: %v", err)
	}
}

func TestPruneObjectsSupportsDryRunAndPreservesReferences(t *testing.T) {
	storage := testStore(t)
	libraryHash := repeatedHash('a')
	pinnedHash := repeatedHash('b')
	orphanHash := repeatedHash('c')
	libraryPath := writeTestObject(t, storage, libraryHash, "library", "library")
	pinnedPath := writeTestObject(t, storage, pinnedHash, "pinned", "pinned")
	orphanPath := writeTestObject(t, storage, orphanHash, "orphan", "orphan")
	unknownPath := writeTestObject(t, storage, "not-a-hash", "unknown", "unknown")

	if err := storage.SaveCatalog(domain.Catalog{Skills: []domain.Skill{{
		ID: "local/library", Name: "library", Hash: libraryHash, Path: libraryPath,
	}}}); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveState(domain.State{Activations: []domain.Activation{{
		SkillID: "team/pinned", Name: "pinned", PinnedHash: pinnedHash, PinnedPath: pinnedPath,
	}}}); err != nil {
		t.Fatal(err)
	}

	dryRun, err := storage.PruneObjects(true)
	if err != nil {
		t.Fatal(err)
	}
	if dryRun.Candidates != 1 || dryRun.Removed != 0 || dryRun.Retained != 2 {
		t.Fatalf("dry-run result = %#v", dryRun)
	}
	if _, err := os.Stat(orphanPath); err != nil {
		t.Fatalf("dry-run removed orphan: %v", err)
	}

	result, err := storage.PruneObjects(false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Candidates != 1 || result.Removed != 1 || result.Retained != 2 {
		t.Fatalf("prune result = %#v", result)
	}
	for _, path := range []string{libraryPath, pinnedPath, unknownPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("preserved object %s missing: %v", path, err)
		}
	}
	if _, err := os.Lstat(orphanPath); !os.IsNotExist(err) {
		t.Fatalf("orphan still exists: %v", err)
	}
}

func TestDeleteObjectIfUnreferencedRetainsNonStandardMetadata(t *testing.T) {
	storage := testStore(t)
	result, err := storage.DeleteObjectIfUnreferenced("legacy-hash", "review")
	if err != nil {
		t.Fatal(err)
	}
	if result.Deleted || result.RetainedReason == "" {
		t.Fatalf("result = %#v", result)
	}
}

func writeTestObject(t *testing.T, storage *Store, hash, name, body string) string {
	t.Helper()
	path := storage.ObjectPath(hash, name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func repeatedHash(value byte) string {
	result := make([]byte, 64)
	for i := range result {
		result[i] = value
	}
	return string(result)
}
