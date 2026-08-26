package buildinfo

import (
	"runtime/debug"
	"strings"
)

// Version is populated by release builds. Development builds intentionally
// keep the value as "dev" so callers can distinguish them from releases.
var Version = "dev"

// Current returns a display-safe version without a leading v.
func Current() string {
	if Version != "" && Version != "dev" {
		return strings.TrimPrefix(Version, "v")
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "dev"
}
