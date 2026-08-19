package prompt

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/tags"
	"gopkg.in/yaml.v3"
)

const MaxPromptSize = int64(1 << 20)

var (
	validName       = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)
	validVariable   = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,62}$`)
	placeholderExpr = regexp.MustCompile(`\{\{\s*([a-z][a-z0-9_-]{0,62})\s*\}\}`)
)

type frontmatter struct {
	Name        string                  `yaml:"name"`
	Description string                  `yaml:"description"`
	Tags        []string                `yaml:"tags,omitempty"`
	Variables   []domain.PromptVariable `yaml:"variables,omitempty"`
}

type Document struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Tags        []string                `json:"tags"`
	Variables   []domain.PromptVariable `json:"variables"`
	Body        string                  `json:"body"`
	Content     string                  `json:"content,omitempty"`
	Hash        string                  `json:"hash"`
	Path        string                  `json:"path,omitempty"`
}

type RenderResult struct {
	Content          string   `json:"content"`
	MissingVariables []string `json:"missingVariables"`
}

func Validate(path string) (Document, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Document{}, err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return Document{}, err
	}
	if info.IsDir() {
		absolute = filepath.Join(absolute, "PROMPT.md")
		info, err = os.Stat(absolute)
		if err != nil {
			if os.IsNotExist(err) {
				return Document{}, fmt.Errorf("PROMPT.md not found in %s", path)
			}
			return Document{}, err
		}
	}
	if !info.Mode().IsRegular() {
		return Document{}, fmt.Errorf("%s is not a regular Prompt file", absolute)
	}
	if info.Size() > MaxPromptSize {
		return Document{}, fmt.Errorf("PROMPT.md exceeds %d bytes", MaxPromptSize)
	}
	data, err := os.ReadFile(absolute)
	if err != nil {
		return Document{}, err
	}
	document, err := Parse(data)
	if err != nil {
		return Document{}, err
	}
	document.Path = absolute
	return document, nil
}

func Parse(data []byte) (Document, error) {
	if int64(len(data)) > MaxPromptSize {
		return Document{}, fmt.Errorf("PROMPT.md exceeds %d bytes", MaxPromptSize)
	}
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	metadata, body, err := splitFrontmatter(data)
	if err != nil {
		return Document{}, err
	}
	var header frontmatter
	if err := yaml.Unmarshal(metadata, &header); err != nil {
		return Document{}, fmt.Errorf("invalid PROMPT.md frontmatter: %w", err)
	}
	header.Name = strings.TrimSpace(header.Name)
	header.Description = strings.TrimSpace(header.Description)
	if !validName.MatchString(header.Name) {
		return Document{}, fmt.Errorf("invalid prompt name %q", header.Name)
	}
	if header.Description == "" {
		return Document{}, fmt.Errorf("PROMPT.md frontmatter requires a string description")
	}
	if len(header.Description) > 1024 {
		return Document{}, fmt.Errorf("prompt description exceeds 1024 bytes")
	}
	if strings.TrimSpace(string(body)) == "" {
		return Document{}, fmt.Errorf("PROMPT.md body cannot be empty")
	}
	if len(header.Tags) > 0 {
		header.Tags, err = tags.Normalize(header.Tags, nil)
		if err != nil {
			return Document{}, err
		}
	}
	if err := validateVariables(header.Variables, string(body)); err != nil {
		return Document{}, err
	}
	hash := sha256.Sum256(data)
	return Document{
		Name: header.Name, Description: header.Description, Tags: header.Tags,
		Variables: normalizeVariables(header.Variables), Body: string(body), Content: string(data),
		Hash: hex.EncodeToString(hash[:]),
	}, nil
}

func Render(document Document, values map[string]string) (RenderResult, error) {
	variables := make(map[string]domain.PromptVariable, len(document.Variables))
	for _, variable := range document.Variables {
		variables[variable.Name] = variable
	}
	for name := range values {
		if _, ok := variables[name]; !ok {
			return RenderResult{}, fmt.Errorf("unknown prompt variable %q", name)
		}
	}
	resolved := make(map[string]string, len(variables))
	var missing []string
	for _, variable := range document.Variables {
		value, provided := values[variable.Name]
		if !provided {
			value = variable.Default
		}
		if value == "" && variable.Required {
			missing = append(missing, variable.Name)
			continue
		}
		if err := validateValue(variable, value); err != nil {
			return RenderResult{}, err
		}
		resolved[variable.Name] = value
	}
	content := placeholderExpr.ReplaceAllStringFunc(document.Body, func(match string) string {
		parts := placeholderExpr.FindStringSubmatch(match)
		if value, ok := resolved[parts[1]]; ok {
			return value
		}
		return match
	})
	return RenderResult{Content: content, MissingVariables: missing}, nil
}

func Build(name, description, body string, tags []string, variables []domain.PromptVariable) ([]byte, error) {
	header, err := yaml.Marshal(frontmatter{Name: name, Description: description, Tags: tags, Variables: variables})
	if err != nil {
		return nil, err
	}
	data := append([]byte("---\n"), header...)
	data = append(data, []byte("---\n")...)
	data = append(data, []byte(strings.TrimSpace(body)+"\n")...)
	if _, err := Parse(data); err != nil {
		return nil, err
	}
	return data, nil
}

func validateVariables(variables []domain.PromptVariable, body string) error {
	seen := make(map[string]struct{}, len(variables))
	for _, variable := range variables {
		variable.Name = strings.TrimSpace(variable.Name)
		if !validVariable.MatchString(variable.Name) {
			return fmt.Errorf("invalid prompt variable name %q", variable.Name)
		}
		if _, ok := seen[variable.Name]; ok {
			return fmt.Errorf("duplicate prompt variable %q", variable.Name)
		}
		seen[variable.Name] = struct{}{}
		variable.Type = normalizedType(variable.Type)
		switch variable.Type {
		case "text", "multiline", "number", "boolean", "select", "secret":
		default:
			return fmt.Errorf("unsupported type %q for prompt variable %s", variable.Type, variable.Name)
		}
		if variable.Type == "select" && len(variable.Options) == 0 {
			return fmt.Errorf("select prompt variable %s requires options", variable.Name)
		}
		if variable.Type == "secret" && variable.Default != "" {
			return fmt.Errorf("secret prompt variable %s cannot define a default", variable.Name)
		}
		if err := validateValue(variable, variable.Default); err != nil {
			return err
		}
	}
	for _, match := range placeholderExpr.FindAllStringSubmatch(body, -1) {
		if _, ok := seen[match[1]]; !ok {
			return fmt.Errorf("prompt body uses undeclared variable %q", match[1])
		}
	}
	return nil
}

func validateValue(variable domain.PromptVariable, value string) error {
	if value == "" {
		return nil
	}
	switch normalizedType(variable.Type) {
	case "number":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("prompt variable %s must be a number", variable.Name)
		}
	case "boolean":
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("prompt variable %s must be true or false", variable.Name)
		}
	case "select":
		for _, option := range variable.Options {
			if value == option {
				return nil
			}
		}
		return fmt.Errorf("prompt variable %s must be one of: %s", variable.Name, strings.Join(variable.Options, ", "))
	}
	return nil
}

func normalizeVariables(variables []domain.PromptVariable) []domain.PromptVariable {
	result := append([]domain.PromptVariable(nil), variables...)
	for index := range result {
		result[index].Name = strings.TrimSpace(result[index].Name)
		result[index].Label = strings.TrimSpace(result[index].Label)
		result[index].Type = normalizedType(result[index].Type)
		if result[index].Label == "" {
			result[index].Label = result[index].Name
		}
	}
	return result
}

func normalizedType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "text"
	}
	return value
}

func splitFrontmatter(data []byte) ([]byte, []byte, error) {
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return nil, nil, fmt.Errorf("PROMPT.md must start with YAML frontmatter")
	}
	rest := data[4:]
	index := bytes.Index(rest, []byte("\n---\n"))
	if index < 0 {
		if bytes.HasSuffix(rest, []byte("\n---")) {
			index = len(rest) - 4
		} else {
			return nil, nil, fmt.Errorf("PROMPT.md frontmatter is not closed")
		}
	}
	bodyStart := index + len("\n---\n")
	if bodyStart > len(rest) {
		bodyStart = len(rest)
	}
	return rest[:index], rest[bodyStart:], nil
}
