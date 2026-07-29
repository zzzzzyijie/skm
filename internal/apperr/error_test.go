package apperr

import (
	"errors"
	"testing"
)

func TestCodeFindsWrappedApplicationError(t *testing.T) {
	err := errors.Join(errors.New("other"), Wrap("invalid_skill", "invalid Skill", errors.New("bad frontmatter")))
	if got := Code(err); got != "invalid_skill" {
		t.Fatalf("unexpected joined code %q", got)
	}
	wrapped := Wrap("invalid_skill", "invalid Skill", errors.New("bad frontmatter"))
	if got := Code(wrapped); got != "invalid_skill" {
		t.Fatalf("Code = %q", got)
	}
}

func TestNewAndWrapMessages(t *testing.T) {
	if got := New("not_found", "missing").Error(); got != "missing" {
		t.Fatalf("New message = %q", got)
	}
	if got := Wrap("read_failed", "read", errors.New("denied")).Error(); got != "read: denied" {
		t.Fatalf("Wrap message = %q", got)
	}
}
