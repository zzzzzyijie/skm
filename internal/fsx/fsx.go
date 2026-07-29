package fsx

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	MaxFileSize  = int64(10 << 20)
	MaxTotalSize = int64(50 << 20)
	MaxFiles     = 1000
)

type TreeInfo struct {
	Files     int   `json:"files"`
	TotalSize int64 `json:"totalSize"`
}

func ValidateTree(root string) (TreeInfo, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return TreeInfo{}, err
	}
	rootInfo, err := os.Stat(root)
	if err != nil {
		return TreeInfo{}, err
	}
	if !rootInfo.IsDir() {
		return TreeInfo{}, fmt.Errorf("%s is not a directory", root)
	}

	var result TreeInfo
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root || entry.IsDir() {
			return nil
		}
		result.Files++
		if result.Files > MaxFiles {
			return fmt.Errorf("skill contains more than %d files", MaxFiles)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if filepath.IsAbs(link) {
				return fmt.Errorf("absolute symlink is not allowed: %s", path)
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(path), link))
			if !Within(root, resolved) {
				return fmt.Errorf("symlink escapes skill root: %s", path)
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported file type: %s", path)
		}
		if info.Size() > MaxFileSize {
			return fmt.Errorf("file exceeds %d bytes: %s", MaxFileSize, path)
		}
		result.TotalSize += info.Size()
		if result.TotalSize > MaxTotalSize {
			return fmt.Errorf("skill exceeds %d total bytes", MaxTotalSize)
		}
		return nil
	})
	return result, err
}

func HashDir(root string) (string, error) {
	if _, err := ValidateTree(root); err != nil {
		return "", err
	}
	root, _ = filepath.Abs(root)
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, path := range paths {
		rel, _ := filepath.Rel(root, path)
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		_, _ = io.WriteString(h, filepath.ToSlash(rel))
		_, _ = io.WriteString(h, "\x00"+info.Mode().String()+"\x00")
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return "", err
			}
			_, _ = io.WriteString(h, link)
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			return "", err
		}
		_, copyErr := io.Copy(h, file)
		closeErr := file.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func CopyDirAtomic(source, destination string) error {
	if _, err := ValidateTree(source); err != nil {
		return err
	}
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	temp, err := os.MkdirTemp(parent, ".skm-copy-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	staged := filepath.Join(temp, "content")
	if err := copyDir(source, staged); err != nil {
		return err
	}
	return ReplacePath(staged, destination)
}

func copyDir(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
		if err != nil {
			_ = in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		inErr := in.Close()
		outErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if inErr != nil {
			return inErr
		}
		return outErr
	})
}

func AtomicWriteFile(path string, data []byte, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".skm-write-")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, path)
}

func ReplacePath(staged, destination string) error {
	if _, err := os.Lstat(destination); os.IsNotExist(err) {
		return os.Rename(staged, destination)
	} else if err != nil {
		return err
	}
	backup := destination + ".skm-backup"
	_ = os.RemoveAll(backup)
	if err := os.Rename(destination, backup); err != nil {
		return err
	}
	if err := os.Rename(staged, destination); err != nil {
		_ = os.Rename(backup, destination)
		return err
	}
	return os.RemoveAll(backup)
}

func Within(root, path string) bool {
	root, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
