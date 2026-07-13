#!/usr/bin/env bash

set -euo pipefail

VERSION_INPUT="${1:-}"
CHECKSUMS="${2:-}"
OUTPUT="${3:-}"
TEMPLATE="${GOKUB_FORMULA_TEMPLATE:-packaging/homebrew/Formula/gokub.rb.tmpl}"

fail() {
  printf 'formula renderer: %s\n' "$1" >&2
  exit 1
}

[[ "$VERSION_INPUT" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]] ||
  fail "version must be SemVer, for example v0.2.0"
[[ -f "$CHECKSUMS" ]] || fail "checksums file not found: $CHECKSUMS"
[[ -f "$TEMPLATE" ]] || fail "formula template not found: $TEMPLATE"
[[ -n "$OUTPUT" ]] || fail "usage: render-homebrew-formula.sh <version> <checksums.txt> <output>"

VERSION="${VERSION_INPUT#v}"

checksum_for() {
  local filename="$1"
  local value
  value="$(awk -v file="$filename" '$2 == file { print $1 }' "$CHECKSUMS")"
  [[ "$value" =~ ^[0-9a-fA-F]{64}$ ]] || fail "missing or invalid checksum for $filename"
  printf '%s' "$value"
}

DARWIN_ARM64="$(checksum_for "gokub_${VERSION}_Darwin_arm64.tar.gz")"
DARWIN_X86_64="$(checksum_for "gokub_${VERSION}_Darwin_x86_64.tar.gz")"
LINUX_ARM64="$(checksum_for "gokub_${VERSION}_Linux_arm64.tar.gz")"
LINUX_X86_64="$(checksum_for "gokub_${VERSION}_Linux_x86_64.tar.gz")"

mkdir -p "$(dirname "$OUTPUT")"
sed \
  -e "s/VERSION/${VERSION}/g" \
  -e "s/DARWIN_ARM64_SHA256/${DARWIN_ARM64}/g" \
  -e "s/DARWIN_X86_64_SHA256/${DARWIN_X86_64}/g" \
  -e "s/LINUX_ARM64_SHA256/${LINUX_ARM64}/g" \
  -e "s/LINUX_X86_64_SHA256/${LINUX_X86_64}/g" \
  "$TEMPLATE" > "$OUTPUT"

if grep -Eq 'VERSION|[A-Z_]+_SHA256' "$OUTPUT"; then
  fail "formula still contains unresolved placeholders"
fi

printf 'Rendered Homebrew formula: %s\n' "$OUTPUT"
