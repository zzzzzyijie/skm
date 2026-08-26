#!/bin/sh
set -eu

REPOSITORY_ROOT="$(cd "$PROJECT_DIR/.." && pwd)"
RESOURCE_DIRECTORY="$TARGET_BUILD_DIR/$UNLOCALIZED_RESOURCES_FOLDER_PATH"
OUTPUT_PATH="$RESOURCE_DIRECTORY/skm-core"
TEMPORARY_DIRECTORY="$(mktemp -d "${TMPDIR:-/tmp}/skm-core-build.XXXXXX")"

# Xcode launches Run Script phases with a deliberately narrow PATH. Prefer an
# explicit override, then the standard Homebrew locations, before falling back
# to any Go already visible to the process.
if [ -n "${SKM_GO_EXECUTABLE:-}" ] && [ -x "$SKM_GO_EXECUTABLE" ]; then
  GO_EXECUTABLE="$SKM_GO_EXECUTABLE"
elif [ -x /opt/homebrew/bin/go ]; then
  GO_EXECUTABLE=/opt/homebrew/bin/go
elif [ -x /usr/local/bin/go ]; then
  GO_EXECUTABLE=/usr/local/bin/go
elif command -v go >/dev/null 2>&1; then
  GO_EXECUTABLE="$(command -v go)"
else
  echo "error: Go 1.25+ is required. Install Go, or set SKM_GO_EXECUTABLE to its absolute path." >&2
  exit 1
fi

cleanup() {
  rm -rf "$TEMPORARY_DIRECTORY"
}
trap cleanup EXIT

mkdir -p "$RESOURCE_DIRECTORY"
cd "$REPOSITORY_ROOT"

build_core() {
  GOOS=darwin GOARCH="$1" CGO_ENABLED=0 "$GO_EXECUTABLE" build \
    -trimpath \
    -ldflags "-s -w -X github.com/zzzzzyijie/skm/internal/buildinfo.Version=$MARKETING_VERSION" \
    -o "$2" \
    ./cmd/skm
}

build_core arm64 "$TEMPORARY_DIRECTORY/skm-core-arm64"
build_core amd64 "$TEMPORARY_DIRECTORY/skm-core-x86_64"
lipo -create \
  "$TEMPORARY_DIRECTORY/skm-core-arm64" \
  "$TEMPORARY_DIRECTORY/skm-core-x86_64" \
  -output "$OUTPUT_PATH"

chmod 755 "$OUTPUT_PATH"

if [ "${CODE_SIGNING_ALLOWED:-NO}" = "YES" ] && [ -n "${EXPANDED_CODE_SIGN_IDENTITY:-}" ]; then
  codesign --force --options runtime --timestamp=none --sign "$EXPANDED_CODE_SIGN_IDENTITY" "$OUTPUT_PATH"
fi
