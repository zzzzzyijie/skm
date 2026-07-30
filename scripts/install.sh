#!/bin/sh

set -eu

repository="zzzzzyijie/skm"
version="${SKM_VERSION:-latest}"
install_dir="${SKM_INSTALL_DIR:-}"

usage() {
  cat <<'EOF'
Install skm from GitHub Releases.

Usage:
  install.sh [--version <version>] [--install-dir <directory>]

Environment:
  SKM_VERSION       Release version, for example v0.2.0 (default: latest)
  SKM_INSTALL_DIR   Destination directory override
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      [ "$#" -ge 2 ] || { echo "error: --version requires a value" >&2; exit 1; }
      version="$2"
      shift 2
      ;;
    --install-dir)
      [ "$#" -ge 2 ] || { echo "error: --install-dir requires a value" >&2; exit 1; }
      install_dir="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

case "$version" in
  ""|*[!a-zA-Z0-9._-]*)
    echo "error: invalid version: $version" >&2
    exit 1
    ;;
esac

case "$version" in
  latest) release_path="latest/download" ;;
  v[0-9]*|[0-9]*)
    case "$version" in
      v*) tag="$version" ;;
      *) tag="v$version" ;;
    esac
    release_path="download/$tag"
    ;;
  *)
    echo "error: invalid version: $version" >&2
    exit 1
    ;;
esac

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *)
    echo "error: unsupported operating system: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "error: unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

archive="skm_${os}_${arch}.tar.gz"
release_base_url="${SKM_RELEASE_BASE_URL:-https://github.com/$repository/releases/$release_path}"
temporary_dir="$(mktemp -d "${TMPDIR:-/tmp}/skm-install.XXXXXX")"
trap 'rm -rf "$temporary_dir"' EXIT HUP INT TERM

echo "Downloading $archive..."
curl -fsSL "$release_base_url/$archive" -o "$temporary_dir/$archive"
curl -fsSL "$release_base_url/checksums.txt" -o "$temporary_dir/checksums.txt"

expected="$(awk -v name="$archive" '$2 == name { print $1 }' "$temporary_dir/checksums.txt")"
if [ -z "$expected" ]; then
  echo "error: $archive is missing from checksums.txt" >&2
  exit 1
fi

if command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$temporary_dir/$archive" | awk '{ print $1 }')"
elif command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$temporary_dir/$archive" | awk '{ print $1 }')"
else
  echo "error: shasum or sha256sum is required" >&2
  exit 1
fi

if [ "$actual" != "$expected" ]; then
  echo "error: checksum verification failed for $archive" >&2
  exit 1
fi

tar -xzf "$temporary_dir/$archive" -C "$temporary_dir"
[ -f "$temporary_dir/skm" ] || { echo "error: archive does not contain skm" >&2; exit 1; }

if [ -z "$install_dir" ]; then
  existing="$(command -v skm 2>/dev/null || true)"
  if [ -n "$existing" ] && [ ! -L "$existing" ] && [ -w "$(dirname "$existing")" ]; then
    install_dir="$(dirname "$existing")"
  else
    previous_ifs="$IFS"
    IFS=:
    for directory in $PATH; do
      [ -n "$directory" ] || continue
      [ "$directory" != "." ] || continue
      [ ! -L "$directory/skm" ] || continue
      if [ -d "$directory" ] && [ -w "$directory" ]; then
        install_dir="$directory"
        break
      fi
    done
    IFS="$previous_ifs"
  fi
fi

if [ -z "$install_dir" ]; then
  install_dir="$HOME/.local/bin"
fi

mkdir -p "$install_dir"
[ -w "$install_dir" ] || {
  echo "error: $install_dir is not writable; set SKM_INSTALL_DIR to a writable PATH directory" >&2
  exit 1
}
[ ! -L "$install_dir/skm" ] || {
  echo "error: $install_dir/skm is managed through a symlink; use its package manager or choose another directory" >&2
  exit 1
}

install -m 0755 "$temporary_dir/skm" "$install_dir/skm"
echo "Installed skm to $install_dir/skm"

case ":$PATH:" in
  *":$install_dir:"*)
    "$install_dir/skm" version
    ;;
  *)
    echo "Add this directory to PATH before using skm:" >&2
    echo "  export PATH=\"$install_dir:\$PATH\"" >&2
    ;;
esac
