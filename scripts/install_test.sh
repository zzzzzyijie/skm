#!/bin/sh

set -eu

root="$(mktemp -d "${TMPDIR:-/tmp}/skm-installer-test.XXXXXX")"
trap 'rm -rf "$root"' EXIT HUP INT TERM

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *) exit 0 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) exit 0 ;;
esac

mkdir -p "$root/payload" "$root/release" "$root/bin"
cat > "$root/payload/skm" <<'EOF'
#!/bin/sh
echo test-version
EOF
chmod +x "$root/payload/skm"

archive="skm_${os}_${arch}.tar.gz"
tar -czf "$root/release/$archive" -C "$root/payload" skm

if command -v shasum >/dev/null 2>&1; then
  checksum="$(shasum -a 256 "$root/release/$archive" | awk '{ print $1 }')"
else
  checksum="$(sha256sum "$root/release/$archive" | awk '{ print $1 }')"
fi
printf '%s  %s\n' "$checksum" "$archive" > "$root/release/checksums.txt"

SKM_RELEASE_BASE_URL="file://$root/release" \
  sh scripts/install.sh --install-dir "$root/bin"

[ "$("$root/bin/skm")" = "test-version" ]

mkdir -p "$root/symlink-bin"
ln -s "$root/bin/skm" "$root/symlink-bin/skm"
if SKM_RELEASE_BASE_URL="file://$root/release" \
  sh scripts/install.sh --install-dir "$root/symlink-bin" >/dev/null 2>&1; then
  echo "installer unexpectedly overwrote a managed symlink" >&2
  exit 1
fi

if sh scripts/install.sh --version '../invalid' >/dev/null 2>&1; then
  echo "installer unexpectedly accepted an invalid version" >&2
  exit 1
fi
