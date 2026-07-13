#!/usr/bin/env bash

set -euo pipefail

TAG="${1:-}"
DIST="${2:-dist}"

fail() {
  printf 'release verifier: %s\n' "$1" >&2
  exit 1
}

[[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]] ||
  fail "tag must be a v-prefixed SemVer"
[[ -f "$DIST/checksums.txt" ]] || fail "checksums.txt is missing"

VERSION="${TAG#v}"
assets=(
  "gokub_${VERSION}_Darwin_arm64.tar.gz"
  "gokub_${VERSION}_Darwin_x86_64.tar.gz"
  "gokub_${VERSION}_Linux_arm64.tar.gz"
  "gokub_${VERSION}_Linux_x86_64.tar.gz"
)

digest_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

for asset in "${assets[@]}"; do
  path="$DIST/$asset"
  [[ -f "$path" ]] || fail "release asset is missing: $asset"
  expected="$(awk -v file="$asset" '$2 == file { print $1 }' "$DIST/checksums.txt")"
  [[ "$expected" =~ ^[0-9a-fA-F]{64}$ ]] || fail "checksum is missing or invalid: $asset"
  actual="$(digest_file "$path")"
  [[ "$actual" == "$expected" ]] || fail "checksum mismatch: $asset"
done

TEMP="$(mktemp -d "${TMPDIR:-/tmp}/gokub-release-verify.XXXXXX")"
trap 'rm -rf "$TEMP"' EXIT
LINUX_ARCHIVE="$DIST/gokub_${VERSION}_Linux_x86_64.tar.gz"
tar -xzf "$LINUX_ARCHIVE" -C "$TEMP" -- gokub
[[ -f "$TEMP/gokub" && ! -L "$TEMP/gokub" && -x "$TEMP/gokub" ]] ||
  fail "Linux archive does not contain a regular executable"

metadata="$($TEMP/gokub version --json)"
printf '%s' "$metadata" | grep -Fq '"version":"'"$VERSION"'"' || fail "binary version does not match $TAG"
printf '%s' "$metadata" | grep -Fq '"repository":"ongyoo/gokub"' || fail "binary repository metadata is incorrect"

if [[ "${GOKUB_ALLOW_DEV_PROVENANCE:-0}" != "1" ]]; then
  printf '%s' "$metadata" | grep -Fq '"commit":"dev"' && fail "release binary contains a development commit"
  printf '%s' "$metadata" | grep -Fq '"build_date":"unknown"' && fail "release binary has no build date"
fi

printf 'Verified %s release artifacts for %s.\n' "${#assets[@]}" "$TAG"
