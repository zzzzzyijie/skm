#!/bin/sh

# Build and exercise skm against temporary Library, Agent, and project roots.
# It never writes to the caller's real ~/.skm, ~/.claude, ~/.codex, or project.

set -eu

usage() {
  cat <<'EOF'
usage: sh scripts/dev_smoke_test.sh [--full] [--race] [--keep]

Builds a temporary skm binary and runs an isolated end-to-end smoke test.

  --full  Also run go test, go vet, installer, and Formula generator tests.
  --race  Also run go test -race ./... (implies --full).
  --keep  Preserve the temporary test directory and print its location.
EOF
}

run_full=false
run_race=false
keep=false

for arg in "$@"; do
  case "$arg" in
    --full) run_full=true ;;
    --race) run_full=true; run_race=true ;;
    --keep) keep=true ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown option: $arg" >&2
      usage >&2
      exit 2
      ;;
  esac
done

repository_root=$(CDPATH= cd "$(dirname "$0")/.." && pwd)
test_root=$(mktemp -d "${TMPDIR:-/tmp}/skm-dev-smoke.XXXXXX")
test_root=$(CDPATH= cd "$test_root" && pwd)
binary="$test_root/bin/skm"
user_home="$test_root/user"
skm_home="$user_home/.skm"
project_root="$test_root/project"
skill_root="$test_root/smoke-skill"
prompt_file="$test_root/PROMPT.md"
go_cache="${GOCACHE:-/tmp/skm-go-cache}"

cleanup() {
  if [ "$keep" = true ]; then
    echo "Temporary test directory retained: $test_root"
    echo "Run this UI manually:"
    echo "  $binary --home $skm_home --user-home $user_home --project $project_root ui --port 9527 --no-browser"
    return
  fi
  rm -rf "$test_root"
}
trap cleanup EXIT HUP INT TERM

step() {
  printf '\n==> %s\n' "$1"
}

run_skm() {
  "$binary" \
    --home "$skm_home" \
    --user-home "$user_home" \
    --project "$project_root" \
    "$@"
}

step "Preparing isolated test roots"
mkdir -p "$user_home" "$project_root" "$skill_root" "$(dirname "$binary")"
printf '%s\n' \
  '---' \
  'name: smoke-skill' \
  'description: Temporary Skill for the skm development smoke test.' \
  '---' \
  'Use only to exercise a local skm build.' \
  > "$skill_root/SKILL.md"
printf '%s\n' \
  '---' \
  'name: smoke-prompt' \
  'description: Temporary Prompt for the skm development smoke test.' \
  'tags: [smoke]' \
  'variables:' \
  '  - name: topic' \
  '    required: true' \
  '---' \
  'Explain {{topic}} clearly.' \
  > "$prompt_file"

step "Building current source"
(
  cd "$repository_root"
  GOCACHE="$go_cache" go build -trimpath -o "$binary" ./cmd/skm
)
"$binary" version

step "Running isolated Library and Activation flow"
run_skm init
run_skm add "$skill_root" --tag smoke
run_skm --json list | grep -F 'local/smoke-skill' >/dev/null
run_skm enable local/smoke-skill --agent claude,codex

claude_target="$user_home/.claude/skills/smoke-skill"
codex_target="$user_home/.codex/skills/smoke-skill"
[ -L "$claude_target" ]
[ -L "$codex_target" ]
run_skm --json plan | grep -F '"status":"unchanged"' >/dev/null
run_skm --json status | grep -F '"status":"unchanged"' >/dev/null
run_skm doctor | grep -F "$skm_home" >/dev/null

run_skm disable local/smoke-skill
[ ! -e "$claude_target" ] && [ ! -L "$claude_target" ]
[ ! -e "$codex_target" ] && [ ! -L "$codex_target" ]

step "Running isolated Prompt flow"
run_skm prompt validate "$prompt_file"
run_skm prompt add "$prompt_file"
run_skm --json prompt list --tag smoke | grep -F 'local/smoke-prompt' >/dev/null
run_skm prompt render local/smoke-prompt --var topic=testing | grep -F 'Explain testing clearly.' >/dev/null
run_skm prompt remove local/smoke-prompt

if [ "$run_full" = true ]; then
  step "Running repository verification suite"
  (
    cd "$repository_root"
    GOCACHE="$go_cache" go test ./...
    GOCACHE="$go_cache" go vet ./...
    sh scripts/install_test.sh
    sh scripts/generate_homebrew_formula_test.sh
    git diff --check
  )
fi

if [ "$run_race" = true ]; then
  step "Running race detector"
  (
    cd "$repository_root"
    GOCACHE="$go_cache" go test -race ./...
  )
fi

printf '\nPASS: isolated skm smoke test completed.\n'
