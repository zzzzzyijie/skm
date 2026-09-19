package tags

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeUsesDefaultsOnlyWhenUnspecified(t *testing.T) {
	defaults, err := Normalize(nil, []string{"general"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(defaults, []string{"general"}) {
		t.Fatalf("defaults = %#v", defaults)
	}

	explicit, err := Normalize([]string{"Testing", "code-review", "testing"}, []string{"general"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"code-review", "Testing"}
	if !reflect.DeepEqual(explicit, want) {
		t.Fatalf("explicit = %#v, want %#v", explicit, want)
	}
}

func TestNormalizeExplicitPreservesEmptySelection(t *testing.T) {
	actual, err := NormalizeExplicit([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if actual == nil || len(actual) != 0 {
		t.Fatalf("explicit empty tags = %#v, want non-nil empty slice", actual)
	}
}

func TestNormalizeRejectsInvalidTag(t *testing.T) {
	for _, value := range []string{
		"", "UP PER", "-start", "end-", "a_b", "path/name", "emoji-🎉", "zero\u200bwidth", strings.Repeat("知", 33),
	} {
		if _, err := Normalize([]string{value}, nil); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func TestNormalizeSupportsAndCanonicalizesUnicodeTags(t *testing.T) {
	actual, err := Normalize([]string{" 简小知 ", "ＣＯＤＥ", "Cafe\u0301", "CAFÉ", "简小知"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Café", "CODE", "简小知"}
	if !reflect.DeepEqual(actual, want) {
		t.Fatalf("normalized tags = %#v, want %#v", actual, want)
	}
}

func TestMatchAll(t *testing.T) {
	if !MatchAll([]string{"Backend", "Testing"}, []string{"backend", "TESTING"}) {
		t.Fatal("expected AND match")
	}
	if MatchAll([]string{"backend"}, []string{"backend", "testing"}) {
		t.Fatal("expected missing tag to fail")
	}
}

func TestNormalizePreservesCapitalizationAndDeduplicatesCaseInsensitively(t *testing.T) {
	actual, err := Normalize([]string{"AI", "ai", "iOS", "IOS"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"AI", "iOS"}
	if !reflect.DeepEqual(actual, want) {
		t.Fatalf("normalized tags = %#v, want %#v", actual, want)
	}
	if !Equal("ＡＩ", "ai") {
		t.Fatal("expected compatibility-equivalent tag names to match")
	}
}
