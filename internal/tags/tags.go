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
	return normalize(values, false)
}

// NormalizeExplicit validates an explicitly supplied tag list. Unlike
// Normalize, an empty list remains empty instead of restoring defaults.
func NormalizeExplicit(values []string) ([]string, error) {
	return normalize(values, true)
}

func normalize(values []string, allowEmpty bool) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = norm.NFKC.String(strings.TrimSpace(value))
		if !isValidTag(value) {
			return nil, fmt.Errorf("invalid tag %q: use 1-32 Unicode letters, numbers, or internal hyphens", value)
		}
		key := comparisonKey(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 && !allowEmpty {
		return nil, fmt.Errorf("at least one tag is required")
	}
	sort.Slice(result, func(i, j int) bool {
		left := comparisonKey(result[i])
		right := comparisonKey(result[j])
		if left == right {
			return result[i] < result[j]
		}
		return left < right
	})
	return result, nil
}

// Equal compares tag identities without changing their display capitalization.
func Equal(left, right string) bool {
	return comparisonKey(left) == comparisonKey(right)
}

func comparisonKey(value string) string {
	return strings.ToLower(norm.NFKC.String(strings.TrimSpace(value)))
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
		set[comparisonKey(tag)] = struct{}{}
	}
	for _, tag := range required {
		if _, ok := set[comparisonKey(tag)]; !ok {
			return false
		}
	}
	return true
}
