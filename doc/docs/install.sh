#!/usr/bin/env sh
# install.sh — installer for go-crap
#
# Usage:
#   curl -fsSL https://padiazg.github.io/go-crap/install.sh | sh
#   curl -fsSL https://padiazg.github.io/go-crap/install.sh | sh -s -- -v v0.5.0
#
# Env vars:
#   GOPATH or $HOME/bin  — where to put the binary
#
# Installs the pre-built binary with version info stamped via ldflags.
# Prefer this over `go install` which rebuilds from source and loses
# version/commit/buildDate stamping.
# Verify with: go-crap version

set -eu

REPO="padiazg/go-crap"
BINARY="go-crap"
GITHUB="https://github.com/${REPO}"

VERSION=""
INSTALL_DIR=""

# ---- arg parsing ----
while [ $# -gt 0 ]; do
  case "$1" in
    -v|--version) VERSION="$2"; shift 2 ;;
    -d|--dir)     INSTALL_DIR="$2"; shift 2 ;;
    -h|--help)
      grep '^#' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) echo "unknown argument: $1" >&2; exit 1 ;;
  esac
done

err()  { echo "error: $*" >&2; exit 1; }
info() { echo ". $*"; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || err "required command '$1' not found"
}

need_cmd curl
need_cmd tar
need_cmd mktemp

# ---- resolve version ----
if [ -z "$VERSION" ]; then
  info "Looking up latest release..."
  LATEST_URL="$(curl -fsS -o /dev/null -D - "${GITHUB}/releases/latest" \
    | tr -d '\r' | awk 'tolower($1)=="location:"{print $2}')"
  VERSION="${LATEST_URL##*/}"
  [ -n "$VERSION" ] || err "could not determine latest version; pass -v <tag> explicitly"
fi
case "$VERSION" in v*) : ;; *) VERSION="v${VERSION}" ;; esac
VERSION_NUM="${VERSION#v}"

# ---- detect platform ----
OS="$(uname -s)"
case "$OS" in
  Linux)  OS="linux" ;;
  Darwin) OS="darwin" ;;
  *) err "unsupported OS: $OS (only linux and darwin binaries are published)" ;;
esac

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)        ARCH="amd64" ;;
  aarch64|arm64)       ARCH="arm64" ;;
  *) err "unsupported architecture: $ARCH (only amd64 and arm64 are published)" ;;
esac

ARCHIVE="go-crap_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="${GITHUB}/releases/download/${VERSION}/${ARCHIVE}"
CHECKSUMS_URL="${GITHUB}/releases/download/${VERSION}/checksums.txt"

info "Installing ${BINARY} ${VERSION} for ${OS}/${ARCH}"

# ---- resolve install dir ----
if [ -z "$INSTALL_DIR" ]; then
  if command -v go >/dev/null 2>&1 && GOPATH="$(go env GOPATH 2>/dev/null)" && [ -n "$GOPATH" ]; then
    INSTALL_DIR="${GOPATH}/bin"
  elif [ -w "/usr/local/bin" ] 2>/dev/null; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="${HOME}/.local/bin"
  fi
fi
mkdir -p "$INSTALL_DIR" 2>/dev/null || err "cannot create install dir: $INSTALL_DIR"
[ -w "$INSTALL_DIR" ] || err "install dir not writable: $INSTALL_DIR (try: GSA_INSTALL_DIR=\$HOME/.local/bin sh install.sh)"

# ---- download, verify, extract ----
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT INT TERM

info "Downloading ${ARCHIVE}..."
curl -fsSL -o "${TMPDIR}/${ARCHIVE}" "$DOWNLOAD_URL" \
  || err "download failed: $DOWNLOAD_URL"

info "Verifying checksum..."
curl -fsSL -o "${TMPDIR}/checksums.txt" "$CHECKSUMS_URL" \
  || err "could not fetch checksums.txt: $CHECKSUMS_URL"

EXPECTED="$(grep " ${ARCHIVE}\$" "${TMPDIR}/checksums.txt" | awk '{print $1}')"
[ -n "$EXPECTED" ] || err "no checksum entry found for ${ARCHIVE}"

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL="$(shasum -a 256 "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')"
else
  err "need sha256sum or shasum to verify checksum"
fi

[ "$EXPECTED" = "$ACTUAL" ] || err "checksum mismatch for ${ARCHIVE} (expected ${EXPECTED}, got ${ACTUAL})"

info "Extracting..."
tar -xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR" "$BINARY"

install -m 0755 "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}" 2>/dev/null \
  || { cp "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}" && chmod 0755 "${INSTALL_DIR}/${BINARY}"; }

info "Installed to ${INSTALL_DIR}/${BINARY}"

case ":$PATH:" in
  *":${INSTALL_DIR}:"*) : ;;
  *) echo "note: ${INSTALL_DIR} is not on your PATH. Add it, e.g.:"
     echo "  export PATH=\"${INSTALL_DIR}:\$PATH\"" ;;
esac

if command -v "${INSTALL_DIR}/${BINARY}" >/dev/null 2>&1; then
  "${INSTALL_DIR}/${BINARY}" version
fi
