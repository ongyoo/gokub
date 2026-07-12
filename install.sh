#!/usr/bin/env bash
set -euo pipefail

PROJECT="gokub"
REPOSITORY="${GOKUB_REPOSITORY:-gokub/gokub}"
VERSION="${GOKUB_VERSION:-latest}"
INSTALL_DIR="${GOKUB_INSTALL_DIR:-}"

fail() {
  printf 'gokub installer: %s\n' "$1" >&2
  exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required"

case "$(uname -s)" in
  Darwin) os="Darwin" ;;
  Linux) os="Linux" ;;
  *) fail "unsupported operating system: $(uname -s)" ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="x86_64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) fail "unsupported architecture: $(uname -m)" ;;
esac

if [[ "$VERSION" == "latest" ]]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPOSITORY}/releases/latest" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  [[ -n "$VERSION" ]] || fail "could not determine the latest release for ${REPOSITORY}"
fi

release_version="${VERSION#v}"
archive="${PROJECT}_${release_version}_${os}_${arch}.tar.gz"
base_url="https://github.com/${REPOSITORY}/releases/download/${VERSION}"
temporary="$(mktemp -d "${TMPDIR:-/tmp}/gokub-install.XXXXXX")"
trap 'rm -rf "$temporary"' EXIT

printf 'Downloading GOKUB %s for %s/%s...\n' "$VERSION" "$os" "$arch"
curl -fsSL "${base_url}/${archive}" -o "${temporary}/${archive}"
curl -fsSL "${base_url}/checksums.txt" -o "${temporary}/checksums.txt"

expected="$(awk -v file="$archive" '$2 == file { print $1 }' "${temporary}/checksums.txt")"
[[ -n "$expected" ]] || fail "release checksum for ${archive} was not found"
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "${temporary}/${archive}" | awk '{print $1}')"
else
  actual="$(shasum -a 256 "${temporary}/${archive}" | awk '{print $1}')"
fi
[[ "$actual" == "$expected" ]] || fail "checksum verification failed"

tar -xzf "${temporary}/${archive}" -C "$temporary"
[[ -x "${temporary}/${PROJECT}" ]] || fail "release archive does not contain the gokub executable"

if [[ -z "$INSTALL_DIR" ]]; then
  if [[ -d /usr/local/bin && -w /usr/local/bin ]]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="${HOME}/.local/bin"
  fi
fi
mkdir -p "$INSTALL_DIR"
install -m 0755 "${temporary}/${PROJECT}" "${INSTALL_DIR}/${PROJECT}"

printf 'Installed GOKUB %s to %s\n' "$VERSION" "${INSTALL_DIR}/${PROJECT}"
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *) printf 'Add %s to PATH before running gokub.\n' "$INSTALL_DIR" ;;
esac
