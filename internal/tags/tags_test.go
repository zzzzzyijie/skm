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
	want := []string{"code-review", "testing"}
	if !reflect.DeepEqual(explicit, want) {
		t.Fatalf("explicit = %#v, want %#v", explicit, want)
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
	want := []string{"café", "code", "简小知"}
	if !reflect.DeepEqual(actual, want) {
		t.Fatalf("normalized tags = %#v, want %#v", actual, want)
	}
}

func TestMatchAll(t *testing.T) {
	if !MatchAll([]string{"backend", "testing"}, []string{"backend", "testing"}) {
		t.Fatal("expected AND match")
	}
	if MatchAll([]string{"backend"}, []string{"backend", "testing"}) {
		t.Fatal("expected missing tag to fail")
	}
}
