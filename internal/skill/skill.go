package skill

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zzzzzyijie/skm/internal/fsx"
	"gopkg.in/yaml.v3"
)

const MaxSkillMDSize = int64(1 << 20)

var validName = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)

type Document struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata"`
	Body        string         `json:"body,omitempty"`
	Hash        string         `json:"hash"`
	Path        string         `json:"path"`
	Files       int            `json:"files"`
	TotalSize   int64          `json:"totalSize"`
}

// Manifest is the lightweight metadata needed to describe a Skill before its
// complete tree has been validated. Repository previews use it to identify an
// invalid candidate without treating that candidate as importable.
type Manifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func ReadManifest(root string) (Manifest, error) {
	document, err := readDocument(root)
	if err != nil {
		return Manifest{}, err
	}
	return Manifest{Name: document.Name, Description: document.Description}, nil
}

func Validate(root string) (Document, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Document{}, err
	}
	tree, err := fsx.ValidateTree(root)
	if err != nil {
		return Document{}, err
	}
	document, err := readDocument(root)
	if err != nil {
		return Document{}, err
	}
	hash, err := fsx.HashDir(root)
	if err != nil {
		return Document{}, err
	}
	document.Hash = hash
	document.Files = tree.Files
	document.TotalSize = tree.TotalSize
	return document, nil
}

func readDocument(root string) (Document, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Document{}, err
	}
	path := filepath.Join(root, "SKILL.md")
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Document{}, fmt.Errorf("SKILL.md not found in %s", root)
		}
		return Document{}, err
	}
	if info.Size() > MaxSkillMDSize {
		return Document{}, fmt.Errorf("SKILL.md exceeds %d bytes", MaxSkillMDSize)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	frontmatter, body, err := splitFrontmatter(data)
	if err != nil {
		return Document{}, err
	}
	metadata := make(map[string]any)
	if err := yaml.Unmarshal(frontmatter, &metadata); err != nil {
		return Document{}, fmt.Errorf("invalid SKILL.md frontmatter: %w", err)
	}
	name, ok := metadata["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		return Document{}, fmt.Errorf("SKILL.md frontmatter requires a string name")
	}
	name = strings.TrimSpace(name)
	if !validName.MatchString(name) {
		return Document{}, fmt.Errorf("invalid skill name %q", name)
	}
	description, ok := metadata["description"].(string)
	if !ok || strings.TrimSpace(description) == "" {
		return Document{}, fmt.Errorf("SKILL.md frontmatter requires a string description")
	}
	description = strings.TrimSpace(description)
	if len(description) > 1024 {
		return Document{}, fmt.Errorf("skill description exceeds 1024 bytes")
	}
	return Document{
		Name:        name,
		Description: description,
		Metadata:    metadata,
		Body:        string(body),
		Path:        root,
	}, nil
}

func splitFrontmatter(data []byte) ([]byte, []byte, error) {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return nil, nil, fmt.Errorf("SKILL.md must start with YAML frontmatter")
	}
	rest := data[4:]
	index := bytes.Index(rest, []byte("\n---\n"))
	if index < 0 {
		if bytes.HasSuffix(rest, []byte("\n---")) {
			index = len(rest) - 4
		} else {
			return nil, nil, fmt.Errorf("SKILL.md frontmatter is not closed")
		}
	}
	frontmatter := rest[:index]
	bodyStart := index + len("\n---\n")
	if bodyStart > len(rest) {
		bodyStart = len(rest)
	}
	return frontmatter, rest[bodyStart:], nil
}
