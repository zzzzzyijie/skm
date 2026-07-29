package tags

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var validTag = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,30}[a-z0-9])?$`)

func Normalize(values []string, defaults []string) ([]string, error) {
	if len(values) == 0 {
		values = defaults
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if !validTag.MatchString(value) {
			return nil, fmt.Errorf("invalid tag %q: use 1-32 lowercase letters, numbers, or hyphens", value)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("at least one tag is required")
	}
	sort.Strings(result)
	return result, nil
}

func MatchAll(actual, required []string) bool {
	if len(required) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(actual))
	for _, tag := range actual {
		set[tag] = struct{}{}
	}
	for _, tag := range required {
		if _, ok := set[strings.ToLower(tag)]; !ok {
			return false
		}
	}
	return true
}
