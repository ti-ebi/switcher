#!/bin/sh

set -eu

REPO="${SWITCHER_REPO:-ti-ebi/switcher}"
INSTALL_DIR="${SWITCHER_INSTALL_DIR:-$HOME/.local/bin}"
REQUESTED_VERSION="${1:-latest}"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command not found: $1" >&2
    exit 1
  fi
}

detect_os() {
  os="$(uname -s)"
  case "$os" in
    Darwin) echo "darwin" ;;
    Linux) echo "linux" ;;
    *)
      echo "error: unsupported OS: $os (expected Darwin or Linux)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "error: unsupported architecture: $arch (expected x86_64/amd64 or arm64/aarch64)" >&2
      exit 1
      ;;
  esac
}

sha256_file() {
  file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
    return
  fi

  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
    return
  fi

  echo "error: sha256sum or shasum is required for checksum verification" >&2
  exit 1
}

resolve_version() {
  version="$REQUESTED_VERSION"
  if [ "$version" != "latest" ]; then
    case "$version" in
      v*) echo "$version" ;;
      *) echo "v$version" ;;
    esac
    return
  fi

  api_url="https://api.github.com/repos/$REPO/releases/latest"
  if ! response="$(curl -fsSL "$api_url" 2>/dev/null)"; then
    echo "error: could not resolve latest release from $api_url" >&2
    echo "hint: no GitHub Release may exist yet. Create and publish a tag like v0.1.0 first." >&2
    exit 1
  fi

  version="$(printf "%s" "$response" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  if [ -z "$version" ]; then
    echo "error: could not resolve latest release version from $api_url" >&2
    exit 1
  fi

  echo "$version"
}

need_cmd curl
need_cmd tar

OS="$(detect_os)"
ARCH="$(detect_arch)"
VERSION="$(resolve_version)"
ARCHIVE="switcher_${VERSION}_${OS}_${ARCH}.tar.gz"
CHECKSUMS_FILE="checksums.txt"
BASE_URL="https://github.com/$REPO/releases/download/$VERSION"
ARCHIVE_URL="$BASE_URL/$ARCHIVE"
CHECKSUMS_URL="$BASE_URL/$CHECKSUMS_FILE"

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT INT TERM

echo "Installing switcher $VERSION for $OS/$ARCH..."
curl -fsSL "$ARCHIVE_URL" -o "$tmp_dir/$ARCHIVE"
curl -fsSL "$CHECKSUMS_URL" -o "$tmp_dir/$CHECKSUMS_FILE"

expected_checksum="$(grep " $ARCHIVE\$" "$tmp_dir/$CHECKSUMS_FILE" | awk '{print $1}')"
if [ -z "$expected_checksum" ]; then
  echo "error: checksum entry not found for $ARCHIVE" >&2
  exit 1
fi

actual_checksum="$(sha256_file "$tmp_dir/$ARCHIVE")"
if [ "$expected_checksum" != "$actual_checksum" ]; then
  echo "error: checksum verification failed" >&2
  echo "expected: $expected_checksum" >&2
  echo "actual:   $actual_checksum" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
tar -xzf "$tmp_dir/$ARCHIVE" -C "$tmp_dir"

if [ ! -f "$tmp_dir/switcher" ]; then
  echo "error: extracted archive does not contain switcher binary" >&2
  exit 1
fi

install -m 0755 "$tmp_dir/switcher" "$INSTALL_DIR/switcher"

echo "Installed: $INSTALL_DIR/switcher"
if ! command -v tmux >/dev/null 2>&1; then
  echo "warning: tmux is not installed; switcher requires tmux at runtime" >&2
fi

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo "note: add $INSTALL_DIR to your PATH to run 'switcher' directly"
    ;;
esac
