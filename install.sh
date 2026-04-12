#!/usr/bin/env bash
set -euo pipefail

REPO="Edcko/techne-code"
BINARY="techne"
INSTALL_DIR="/usr/local/bin"
TMP_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

info() {
  printf '\033[1;34m[info]\033[0m %s\n' "$1"
}

error() {
  printf '\033[1;31m[error]\033[0m %s\n' "$1" >&2
  exit 1
}

detect_os() {
  case "$(uname -s)" in
    Darwin) echo "darwin" ;;
    Linux) echo "linux" ;;
    *) error "Unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) error "Unsupported architecture: $(uname -m)" ;;
  esac
}

get_latest_version() {
  local api_url="https://api.github.com/repos/${REPO}/releases/latest"
  local version
  version=$(curl -fsSL "$api_url" 2>/dev/null | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name":\s*"([^"]+)".*/\1/')
  if [ -z "$version" ]; then
    error "Could not determine latest version. Check your internet connection or the repository."
  fi
  echo "$version"
}

check_existing() {
  local current_path
  current_path="$(command -v "$BINARY" 2>/dev/null || true)"
  if [ -n "$current_path" ]; then
    local current_version
    current_version=$("$current_path" version 2>/dev/null || echo "unknown")
    info "Existing installation found at ${current_path} (version: ${current_version})"
  fi
}

download_and_install() {
  local os="$1"
  local arch="$2"
  local version="$3"

  local ext="tar.gz"
  if [ "$os" = "windows" ]; then
    ext="zip"
  fi

  local archive_name="${BINARY}_${version#v}_${os}_${arch}.${ext}"
  local download_url="https://github.com/${REPO}/releases/download/${version}/${archive_name}"
  local checksum_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"

  info "Downloading ${archive_name}..."
  curl -fsSL -o "${TMP_DIR}/${archive_name}" "$download_url" || error "Download failed: ${download_url}"

  info "Verifying checksum..."
  if curl -fsSL -o "${TMP_DIR}/checksums.txt" "$checksum_url" 2>/dev/null; then
    local expected
    expected=$(grep "${archive_name}" "${TMP_DIR}/checksums.txt" | awk '{print $1}')
    if [ -n "$expected" ]; then
      local actual
      if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "${TMP_DIR}/${archive_name}" | awk '{print $1}')
      else
        actual=$(shasum -a 256 "${TMP_DIR}/${archive_name}" | awk '{print $1}')
      fi
      if [ "$expected" != "$actual" ]; then
        error "Checksum mismatch! Expected ${expected}, got ${actual}"
      fi
      info "Checksum verified."
    else
      info "No checksum entry found for ${archive_name}, skipping verification."
    fi
  else
    info "Checksums file not available, skipping verification."
  fi

  info "Extracting..."
  tar -xzf "${TMP_DIR}/${archive_name}" -C "$TMP_DIR" || error "Extraction failed"

  local binary_path="${TMP_DIR}/${BINARY}"
  if [ ! -f "$binary_path" ]; then
    binary_path=$(find "$TMP_DIR" -name "$BINARY" -type f | head -1)
  fi
  if [ -z "$binary_path" ] || [ ! -f "$binary_path" ]; then
    error "Binary not found in archive"
  fi

  chmod +x "$binary_path"

  local target="${INSTALL_DIR}/${BINARY}"
  if [ -w "$INSTALL_DIR" ]; then
    cp -f "$binary_path" "$target"
  else
    info "Elevated permissions required to install to ${INSTALL_DIR}"
    sudo cp -f "$binary_path" "$target"
  fi

  info "Installed ${BINARY} to ${target}"
}

main() {
  info "Installing ${BINARY}..."

  local os arch version
  os=$(detect_os)
  arch=$(detect_arch)
  version=$(get_latest_version)

  info "Platform: ${os}/${arch}"
  info "Version: ${version}"

  check_existing
  download_and_install "$os" "$arch" "$version"

  if command -v "$BINARY" >/dev/null 2>&1; then
    info "Success! Run '${BINARY}' to get started."
  else
    info "Installed but not in PATH. Add ${INSTALL_DIR} to your PATH."
  fi
}

main "$@"
