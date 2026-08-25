package tags

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const maxTagRunes = 32

func Normalize(values []string, defaults []string) ([]string, error) {
	if len(values) == 0 {
		values = defaults
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(norm.NFKC.String(strings.TrimSpace(value)))
		if !isValidTag(value) {
			return nil, fmt.Errorf("invalid tag %q: use 1-32 Unicode letters, numbers, or internal hyphens", value)
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

func isValidTag(value string) bool {
	length := utf8.RuneCountInString(value)
	if length == 0 || length > maxTagRunes {
		return false
	}
	for index, r := range []rune(value) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			continue
		}
		if r != '-' || index == 0 || index == length-1 {
			return false
		}
	}
	return true
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
