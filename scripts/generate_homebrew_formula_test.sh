#!/bin/sh

set -eu

repository_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
test_root=$(mktemp -d "${TMPDIR:-/tmp}/skm-formula-test.XXXXXX")
trap 'rm -rf "$test_root"' EXIT HUP INT TERM

checksums="$test_root/checksums.txt"
formula="$test_root/Formula/skm.rb"

cat >"$checksums" <<'EOF'
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  skm_darwin_amd64.tar.gz
bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  skm_darwin_arm64.tar.gz
cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc  skm_linux_amd64.tar.gz
dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd  skm_linux_arm64.tar.gz
EOF

sh "$repository_root/scripts/generate_homebrew_formula.sh" v1.2.3 "$checksums" "$formula"

grep -Fq 'version "1.2.3"' "$formula"
grep -Fq 'sha256 "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"' "$formula"
grep -Fq 'releases/download/v#{version}/skm_linux_arm64.tar.gz' "$formula"
grep -Fq 'system "/usr/bin/xattr", "-c", "skm"' "$formula"
ruby -c "$formula" >/dev/null

if sh "$repository_root/scripts/generate_homebrew_formula.sh" invalid "$checksums" "$formula" >/dev/null 2>&1; then
  echo "error: invalid tag was accepted" >&2
  exit 1
fi

sed '/skm_linux_arm64.tar.gz/d' "$checksums" >"$test_root/incomplete.txt"
if sh "$repository_root/scripts/generate_homebrew_formula.sh" v1.2.3 "$test_root/incomplete.txt" "$formula" >/dev/null 2>&1; then
  echo "error: incomplete checksums were accepted" >&2
  exit 1
fi

echo "Homebrew Formula generator tests passed"
