package store

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/zzzzzyijie/skm/internal/fsx"
)

var (
	objectHashPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
	objectNamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)
)

type ObjectRemovalResult struct {
	Path           string   `json:"path"`
	Existed        bool     `json:"existed"`
	Deleted        bool     `json:"deleted"`
	References     []string `json:"references,omitempty"`
	RetainedReason string   `json:"retainedReason,omitempty"`
}

type PruneResult struct {
	DryRun         bool     `json:"dryRun"`
	Scanned        int      `json:"scanned"`
	Candidates     int      `json:"candidates"`
	Removed        int      `json:"removed"`
	Retained       int      `json:"retained"`
	Bytes          int64    `json:"bytes"`
	CandidatePaths []string `json:"candidatePaths"`
}

// DeleteObjectIfUnreferenced removes one immutable snapshot after its Library
// entry has been removed. Referenced and non-standard object paths are retained.
func (s *Store) DeleteObjectIfUnreferenced(hash, name string) (ObjectRemovalResult, error) {
	path, err := s.safeObjectPath(hash, name)
	if err != nil {
		return ObjectRemovalResult{RetainedReason: err.Error()}, nil
	}
	result := ObjectRemovalResult{Path: path}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.Existed = true
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		result.RetainedReason = "snapshot is not a standard directory"
		return result, nil
	}
	references, err := s.objectReferences()
	if err != nil {
		return result, err
	}
	result.References = references[path]
	if len(result.References) > 0 {
		return result, nil
	}
	if err := os.RemoveAll(path); err != nil {
		return result, err
	}
	result.Deleted = true
	if err := removeEmptyDirectory(filepath.Dir(path)); err != nil {
		return result, err
	}
	return result, nil
}

// PruneObjects removes every standard object snapshot that is not referenced
// by the Library, deployment state, pinned Activations, or current project.
func (s *Store) PruneObjects(dryRun bool) (PruneResult, error) {
	result := PruneResult{DryRun: dryRun}
	references, err := s.objectReferences()
	if err != nil {
		return result, err
	}
	root := filepath.Join(s.Paths.Home, "objects")
	hashEntries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	for _, hashEntry := range hashEntries {
		if !hashEntry.IsDir() || hashEntry.Type()&os.ModeSymlink != 0 || !objectHashPattern.MatchString(hashEntry.Name()) {
			continue
		}
		hashPath := filepath.Join(root, hashEntry.Name())
		objects, err := os.ReadDir(hashPath)
		if err != nil {
			return result, err
		}
		for _, object := range objects {
			if !object.IsDir() || object.Type()&os.ModeSymlink != 0 || !validObjectName(object.Name()) {
				continue
			}
			path := filepath.Join(hashPath, object.Name())
			result.Scanned++
			if len(references[path]) > 0 {
				result.Retained++
				continue
			}
			size, err := directorySize(path)
			if err != nil {
				return result, err
			}
			result.Candidates++
			result.Bytes += size
			result.CandidatePaths = append(result.CandidatePaths, path)
			if dryRun {
				continue
			}
			if err := os.RemoveAll(path); err != nil {
				return result, err
			}
			result.Removed++
		}
		if !dryRun {
			if err := removeEmptyDirectory(hashPath); err != nil {
				return result, err
			}
		}
	}
	sort.Strings(result.CandidatePaths)
	return result, nil
}

func (s *Store) objectReferences() (map[string][]string, error) {
	references := make(map[string][]string)
	add := func(path, reference string) {
		if path == "" {
			return
		}
		absolute, err := filepath.Abs(path)
		if err != nil || !fsx.Within(filepath.Join(s.Paths.Home, "objects"), absolute) {
			return
		}
		path = filepath.Clean(absolute)
		for _, existing := range references[path] {
			if existing == reference {
				return
			}
		}
		references[path] = append(references[path], reference)
	}

	catalog, err := s.LoadCatalog()
	if err != nil {
		return nil, err
	}
	for _, value := range catalog.Skills {
		if objectHashPattern.MatchString(value.Hash) && validObjectName(value.Name) {
			add(s.ObjectPath(value.Hash, value.Name), "library:"+value.ID)
		}
		add(value.Path, "library:"+value.ID)
	}

	state, err := s.LoadState()
	if err != nil {
		return nil, err
	}
	for _, activation := range state.Activations {
		if activation.PinnedHash == "" {
			continue
		}
		path := activation.PinnedPath
		if path == "" && objectHashPattern.MatchString(activation.PinnedHash) && validObjectName(activation.Name) {
			path = s.ObjectPath(activation.PinnedHash, activation.Name)
		}
		add(path, "activation:"+activation.SkillID)
	}
	for _, deployment := range state.Deployments {
		add(deployment.SourcePath, "deployment:"+deployment.SkillID+":"+string(deployment.Agent))
	}

	project, err := s.LoadProjectCatalog()
	if err != nil {
		return nil, err
	}
	for _, dependency := range project.Dependencies {
		if objectHashPattern.MatchString(dependency.Hash) && validObjectName(dependency.Name) {
			add(s.ObjectPath(dependency.Hash, dependency.Name), "project:"+dependency.ID)
		}
	}
	for path := range references {
		sort.Strings(references[path])
	}
	return references, nil
}

func (s *Store) safeObjectPath(hash, name string) (string, error) {
	if !objectHashPattern.MatchString(hash) {
		return "", fmt.Errorf("invalid object hash %q", hash)
	}
	if !validObjectName(name) {
		return "", fmt.Errorf("invalid object name %q", name)
	}
	root := filepath.Join(s.Paths.Home, "objects")
	path := s.ObjectPath(hash, name)
	if !fsx.Within(root, path) {
		return "", fmt.Errorf("object path escapes %s", root)
	}
	parent, err := os.Lstat(filepath.Dir(path))
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err == nil && (!parent.IsDir() || parent.Mode()&os.ModeSymlink != 0) {
		return "", fmt.Errorf("refusing object path through non-directory %s", filepath.Dir(path))
	}
	return filepath.Clean(path), nil
}

func validObjectName(name string) bool {
	return objectNamePattern.MatchString(name) && filepath.Base(name) == name
}

func removeEmptyDirectory(path string) error {
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return os.Remove(path)
	}
	return nil
}

func directorySize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
}
