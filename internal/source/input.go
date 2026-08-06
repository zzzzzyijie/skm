package source

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var (
	githubShorthand  = regexp.MustCompile(`^([A-Za-z0-9](?:[A-Za-z0-9-]{0,38}[A-Za-z0-9])?)/([A-Za-z0-9][A-Za-z0-9._-]*)$`)
	skillsPackageRef = regexp.MustCompile(`^skills(?:@[A-Za-z0-9][A-Za-z0-9._-]*)?$`)
)

type InstallInput struct {
	URL             string
	SuggestedName   string
	RequestedSkills []string
}

// ParseInstallInput accepts a Git locator or the copy-paste form of a
// "npx skills add" command. It parses known arguments but never executes them.
func ParseInstallInput(raw string) (InstallInput, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return InstallInput{}, fmt.Errorf("source input is required")
	}

	sourceValue := value
	var requested []string
	fromCommand := looksLikeSkillsCommand(value)
	if fromCommand {
		tokens, err := splitCommand(value)
		if err != nil {
			return InstallInput{}, err
		}
		sourceValue, requested, err = parseSkillsCommand(tokens)
		if err != nil {
			return InstallInput{}, err
		}
	}

	canonicalURL, nameSeed, err := normalizeSourceLocator(sourceValue, fromCommand)
	if err != nil {
		return InstallInput{}, err
	}
	return InstallInput{
		URL:             canonicalURL,
		SuggestedName:   normalizeSuggestedName(nameSeed, canonicalURL),
		RequestedSkills: requested,
	}, nil
}

func looksLikeSkillsCommand(value string) bool {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "$") {
		value = strings.TrimSpace(strings.TrimPrefix(value, "$"))
	}
	return value == "npx" || strings.HasPrefix(value, "npx ") || strings.HasPrefix(value, "npx\t")
}

func parseSkillsCommand(tokens []string) (string, []string, error) {
	if len(tokens) > 0 && tokens[0] == "$" {
		tokens = tokens[1:]
	}
	if len(tokens) == 0 || tokens[0] != "npx" {
		return "", nil, fmt.Errorf("expected an npx skills add command")
	}

	index := 1
	for index < len(tokens) && (tokens[index] == "-y" || tokens[index] == "--yes") {
		index++
	}
	if index >= len(tokens) || !skillsPackageRef.MatchString(tokens[index]) {
		return "", nil, fmt.Errorf("expected skills or skills@<version> after npx")
	}
	index++
	if index >= len(tokens) || tokens[index] != "add" {
		return "", nil, fmt.Errorf("only npx skills add commands are supported")
	}
	index++

	var sourceValue string
	var requested []string
	allSkills := false
	for index < len(tokens) {
		token := tokens[index]
		switch {
		case token == "-y" || token == "--yes":
			index++
		case token == "--all":
			allSkills = true
			index++
		case token == "-s" || token == "--skill":
			if index+1 >= len(tokens) {
				return "", nil, fmt.Errorf("%s requires a Skill name", token)
			}
			requested = append(requested, tokens[index+1])
			index += 2
		case strings.HasPrefix(token, "--skill="):
			requested = append(requested, strings.TrimPrefix(token, "--skill="))
			index++
		case strings.HasPrefix(token, "-s="):
			requested = append(requested, strings.TrimPrefix(token, "-s="))
			index++
		case strings.HasPrefix(token, "-"):
			return "", nil, fmt.Errorf("option %s is not supported for Library import", token)
		case sourceValue == "":
			sourceValue = token
			index++
		default:
			return "", nil, fmt.Errorf("unexpected argument %q in skills add command", token)
		}
	}
	if sourceValue == "" {
		return "", nil, fmt.Errorf("npx skills add requires a source")
	}

	seen := make(map[string]struct{})
	filtered := requested[:0]
	for _, name := range requested {
		name = strings.TrimSpace(name)
		if name == "*" {
			allSkills = true
			continue
		}
		if name == "" {
			return "", nil, fmt.Errorf("Skill name cannot be empty")
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		filtered = append(filtered, name)
	}
	if allSkills {
		filtered = nil
	}
	return sourceValue, filtered, nil
}

func normalizeSourceLocator(value string, fromCommand bool) (string, string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", fmt.Errorf("source input is required")
	}
	if match := githubShorthand.FindStringSubmatch(value); match != nil {
		owner := match[1]
		repository := strings.TrimSuffix(match[2], ".git")
		if repository == "" || repository == "." || repository == ".." {
			return "", "", fmt.Errorf("invalid GitHub repository shorthand %q", value)
		}
		return "https://github.com/" + owner + "/" + repository + ".git", owner + "-" + repository, nil
	}
	if fromCommand && strings.ContainsAny(value, "\r\n") {
		return "", "", fmt.Errorf("source must be a single value")
	}
	return value, sourceNameSeed(value), nil
}

func sourceNameSeed(value string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(value), "/")
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Host != "" {
		parts := pathParts(parsed.Path)
		if strings.EqualFold(parsed.Hostname(), "github.com") && len(parts) >= 2 {
			return parts[len(parts)-2] + "-" + strings.TrimSuffix(parts[len(parts)-1], ".git")
		}
		if len(parts) > 0 {
			return strings.TrimSuffix(parts[len(parts)-1], ".git")
		}
		return parsed.Hostname()
	}

	if colon := strings.Index(trimmed, ":"); colon > strings.Index(trimmed, "@") && strings.Contains(trimmed[:colon], "@") {
		parts := pathParts(trimmed[colon+1:])
		if len(parts) >= 2 && strings.EqualFold(strings.SplitN(trimmed[:colon], "@", 2)[1], "github.com") {
			return parts[len(parts)-2] + "-" + strings.TrimSuffix(parts[len(parts)-1], ".git")
		}
		if len(parts) > 0 {
			return strings.TrimSuffix(parts[len(parts)-1], ".git")
		}
	}

	return strings.TrimSuffix(filepath.Base(filepath.Clean(trimmed)), ".git")
}

func pathParts(value string) []string {
	values := strings.Split(strings.Trim(value, "/"), "/")
	result := values[:0]
	for _, item := range values {
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func normalizeSuggestedName(seed, identity string) string {
	var result strings.Builder
	lastDash := false
	for _, char := range strings.ToLower(seed) {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' {
			result.WriteRune(char)
			lastDash = false
			continue
		}
		if result.Len() > 0 && !lastDash {
			result.WriteByte('-')
			lastDash = true
		}
	}
	name := strings.Trim(result.String(), "-")
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(identity)))[:8]
	if name == "" {
		return "source-" + digest
	}
	if len(name) > 64 {
		name = strings.TrimRight(name[:55], "-") + "-" + digest
	}
	return name
}

func splitCommand(value string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	quote := rune(0)
	escaped := false
	started := false
	flush := func() {
		if !started {
			return
		}
		tokens = append(tokens, current.String())
		current.Reset()
		started = false
	}

	for _, char := range value {
		if escaped {
			current.WriteRune(char)
			started = true
			escaped = false
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
				continue
			}
			if char == '\\' && quote == '"' {
				escaped = true
				continue
			}
			current.WriteRune(char)
			started = true
			continue
		}

		switch {
		case char == '\n' || char == '\r':
			return nil, fmt.Errorf("skills add command must be on one line")
		case unicode.IsSpace(char):
			flush()
		case char == '\'' || char == '"':
			quote = char
			started = true
		case char == '\\':
			escaped = true
			started = true
		case strings.ContainsRune(";|&<>`", char):
			return nil, fmt.Errorf("shell operators are not allowed in skills add commands")
		default:
			current.WriteRune(char)
			started = true
		}
	}
	if escaped {
		return nil, fmt.Errorf("unfinished escape in skills add command")
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in skills add command")
	}
	flush()
	return tokens, nil
}
