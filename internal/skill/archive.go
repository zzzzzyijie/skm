package skill

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/zzzzzyijie/skm/internal/fsx"
)

// ExtractZIP safely extracts one Skill archive into destination and returns
// the directory containing its SKILL.md. The archive may contain the Skill at
// its root or inside a wrapper directory.
func ExtractZIP(archivePath, destination string) (string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("open Skill ZIP: %w", err)
	}
	defer reader.Close()

	var files int
	var totalSize int64
	for _, entry := range reader.File {
		name := strings.ReplaceAll(entry.Name, "\\", "/")
		name = strings.TrimPrefix(name, "./")
		if name == "" || name == "__MACOSX" || strings.HasPrefix(name, "__MACOSX/") {
			continue
		}
		relative := filepath.Clean(filepath.FromSlash(name))
		if filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("Skill ZIP entry escapes the archive root: %s", entry.Name)
		}
		target := filepath.Join(destination, relative)
		if !fsx.Within(destination, target) {
			return "", fmt.Errorf("Skill ZIP entry escapes the archive root: %s", entry.Name)
		}

		mode := entry.Mode()
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return "", err
			}
			continue
		}
		if !mode.IsRegular() || mode&os.ModeSymlink != 0 {
			return "", fmt.Errorf("unsupported file type in Skill ZIP: %s", entry.Name)
		}
		files++
		if files > fsx.MaxFiles {
			return "", fmt.Errorf("Skill ZIP contains more than %d files", fsx.MaxFiles)
		}
		if entry.UncompressedSize64 > uint64(fsx.MaxFileSize) {
			return "", fmt.Errorf("file exceeds %d bytes in Skill ZIP: %s", fsx.MaxFileSize, entry.Name)
		}
		totalSize += int64(entry.UncompressedSize64)
		if totalSize > fsx.MaxTotalSize {
			return "", fmt.Errorf("Skill ZIP exceeds %d total bytes", fsx.MaxTotalSize)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		permissions := fs.FileMode(0o644)
		if mode.Perm()&0o111 != 0 {
			permissions = 0o755
		}
		if err := extractZIPFile(entry, target, permissions); err != nil {
			return "", err
		}
	}

	root, err := findArchivedSkillRoot(destination)
	if err != nil {
		return "", err
	}
	if _, err := Validate(root); err != nil {
		return "", err
	}
	return root, nil
}

func extractZIPFile(entry *zip.File, target string, permissions fs.FileMode) error {
	input, err := entry.Open()
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, permissions)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(output, io.LimitReader(input, fsx.MaxFileSize+1))
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	if written > fsx.MaxFileSize {
		return fmt.Errorf("file exceeds %d bytes in Skill ZIP: %s", fsx.MaxFileSize, entry.Name)
	}
	return closeErr
}

func findArchivedSkillRoot(destination string) (string, error) {
	var roots []string
	err := filepath.WalkDir(destination, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && entry.Name() == "SKILL.md" {
			roots = append(roots, filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(roots) == 0 {
		return "", fmt.Errorf("SKILL.md not found in Skill ZIP")
	}
	if len(roots) > 1 {
		return "", fmt.Errorf("Skill ZIP must contain exactly one Skill; found %d", len(roots))
	}
	return roots[0], nil
}
