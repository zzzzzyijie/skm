#!/bin/sh

set -eu

repository_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
test_root=$(mktemp -d "${TMPDIR:-/tmp}/skm-cask-test.XXXXXX")
trap 'rm -rf "$test_root"' EXIT HUP INT TERM

checksums="$test_root/checksums.txt"
cask="$test_root/Casks/skm-app.rb"

printf '%s  %s\n' \
  aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  SKM-1.2.3-universal.dmg >"$checksums"

sh "$repository_root/scripts/generate_homebrew_cask.sh" v1.2.3 "$checksums" "$cask"

grep -Fq 'cask "skm-app" do' "$cask"
grep -Fq 'version "1.2.3"' "$cask"
grep -Fq 'sha256 "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"' "$cask"
grep -Fq 'SKM-#{version}-universal.dmg' "$cask"
grep -Fq 'depends_on macos: ">= :sonoma"' "$cask"
grep -Fq 'app "SKM.app"' "$cask"
if grep -Fq 'binary "skm"' "$cask"; then
  echo "error: Cask must not install the CLI binary managed by the Formula" >&2
  exit 1
fi
ruby -c "$cask" >/dev/null

if sh "$repository_root/scripts/generate_homebrew_cask.sh" invalid "$checksums" "$cask" >/dev/null 2>&1; then
  echo "error: invalid tag was accepted" >&2
  exit 1
fi

printf '%s  %s\n' \
  invalid \
  SKM-1.2.3-universal.dmg >"$test_root/invalid-checksums.txt"
if sh "$repository_root/scripts/generate_homebrew_cask.sh" v1.2.3 "$test_root/invalid-checksums.txt" "$cask" >/dev/null 2>&1; then
  echo "error: invalid checksum was accepted" >&2
  exit 1
fi

echo "Homebrew Cask generator tests passed"
