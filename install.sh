#!/usr/bin/env bash

set -euo pipefail

PROJECT="gokub"
REPOSITORY="${GOKUB_REPOSITORY:-ongyoo/gokub}"
VERSION="${GOKUB_VERSION:-latest}"
INSTALL_DIR="${GOKUB_INSTALL_DIR:-}"

fail() {
  printf 'gokub installer: %s\n' "$1" >&2
  exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"

[[ "$REPOSITORY" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] ||
  fail "repository must use owner/name format"
if [[ "$VERSION" != "latest" ]]; then
  [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]] ||
    fail "version must be a v-prefixed SemVer tag"
fi

case "$(uname -s)" in
  Darwin) OS="Darwin" ;;
  Linux)  OS="Linux" ;;
  *) fail "unsupported operating system: $(uname -s)" ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH="x86_64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) fail "unsupported architecture: $(uname -m)" ;;
esac

if [[ "$VERSION" == "latest" ]]; then
  VERSION="$(
    curl -fsSL --proto '=https' --proto-redir '=https' \
      -H "Accept: application/vnd.github+json" \
      "https://api.github.com/repos/${REPOSITORY}/releases/latest" |
      sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' |
      head -n 1
  )"

  [[ -n "$VERSION" ]] ||
    fail "could not determine latest release for ${REPOSITORY}"
fi

[[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]] ||
  fail "release API returned an invalid version tag"

RELEASE_VERSION="${VERSION#v}"
ARCHIVE="${PROJECT}_${RELEASE_VERSION}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPOSITORY}/releases/download/${VERSION}"

TEMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/gokub-install.XXXXXX")"
trap 'rm -rf "$TEMP_DIR"' EXIT

echo "Installing GOKUB ${VERSION} from ${REPOSITORY}"
echo "Downloading ${ARCHIVE}..."

curl -fL --proto '=https' --proto-redir '=https' \
  "${BASE_URL}/${ARCHIVE}" \
  -o "${TEMP_DIR}/${ARCHIVE}" ||
  fail "release asset not found: ${ARCHIVE}"

curl -fL --proto '=https' --proto-redir '=https' \
  "${BASE_URL}/checksums.txt" \
  -o "${TEMP_DIR}/checksums.txt" ||
  fail "checksums.txt was not found"

EXPECTED="$(
  awk -v file="$ARCHIVE" '$2 == file { print $1 }' \
    "${TEMP_DIR}/checksums.txt"
)"

[[ "$EXPECTED" =~ ^[0-9a-fA-F]{64}$ ]] ||
  fail "checksum for ${ARCHIVE} is missing or invalid"

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(
    sha256sum "${TEMP_DIR}/${ARCHIVE}" |
      awk '{print $1}'
  )"
else
  ACTUAL="$(
    shasum -a 256 "${TEMP_DIR}/${ARCHIVE}" |
      awk '{print $1}'
  )"
fi

[[ "$ACTUAL" == "$EXPECTED" ]] ||
  fail "checksum verification failed"

tar -xzf "${TEMP_DIR}/${ARCHIVE}" -C "$TEMP_DIR" -- "$PROJECT"

[[ -f "${TEMP_DIR}/${PROJECT}" && ! -L "${TEMP_DIR}/${PROJECT}" && -x "${TEMP_DIR}/${PROJECT}" ]] ||
  fail "archive does not contain a regular gokub executable"

if [[ -z "$INSTALL_DIR" ]]; then
  if [[ -d /usr/local/bin && -w /usr/local/bin ]]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="${HOME}/.local/bin"
  fi
fi

mkdir -p "$INSTALL_DIR"
install -m 0755 \
  "${TEMP_DIR}/${PROJECT}" \
  "${INSTALL_DIR}/${PROJECT}"

echo "Installed GOKUB ${VERSION} to ${INSTALL_DIR}/${PROJECT}"

case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo "Add this directory to PATH:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac
