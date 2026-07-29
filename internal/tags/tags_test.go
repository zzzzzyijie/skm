package tags

import (
	"reflect"
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
	for _, value := range []string{"", "UP PER", "-start", "end-", "a_b"} {
		if _, err := Normalize([]string{value}, nil); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
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
