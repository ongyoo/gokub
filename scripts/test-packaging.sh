#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEMP="$(mktemp -d "${TMPDIR:-/tmp}/gokub-packaging.XXXXXX")"
trap 'rm -rf "$TEMP"' EXIT

DIGEST="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
for asset in \
  gokub_1.2.3_Darwin_arm64.tar.gz \
  gokub_1.2.3_Darwin_x86_64.tar.gz \
  gokub_1.2.3_Linux_arm64.tar.gz \
  gokub_1.2.3_Linux_x86_64.tar.gz; do
  printf '%s  %s\n' "$DIGEST" "$asset" >> "$TEMP/checksums.txt"
done

cd "$ROOT"
./scripts/render-homebrew-formula.sh v1.2.3 "$TEMP/checksums.txt" "$TEMP/gokub.rb" >/dev/null
grep -Fq 'version "1.2.3"' "$TEMP/gokub.rb"
grep -Fq "$DIGEST" "$TEMP/gokub.rb"
grep -Fq '/releases/download/v1.2.3/gokub_1.2.3_Darwin_arm64.tar.gz' "$TEMP/gokub.rb"
ruby -c "$TEMP/gokub.rb" >/dev/null

head -n 3 "$TEMP/checksums.txt" > "$TEMP/incomplete.txt"
if ./scripts/render-homebrew-formula.sh v1.2.3 "$TEMP/incomplete.txt" "$TEMP/invalid.rb" >/dev/null 2>&1; then
  printf 'renderer accepted an incomplete checksum file\n' >&2
  exit 1
fi

mkdir -p "$TEMP/release/input" "$TEMP/release/dist"
go build -o "$TEMP/release/input/gokub" ./cmd/gokub
for asset in \
  gokub_0.1.0_Darwin_arm64.tar.gz \
  gokub_0.1.0_Darwin_x86_64.tar.gz \
  gokub_0.1.0_Linux_arm64.tar.gz \
  gokub_0.1.0_Linux_x86_64.tar.gz; do
  tar -czf "$TEMP/release/dist/$asset" -C "$TEMP/release/input" gokub
  if command -v sha256sum >/dev/null 2>&1; then
    digest="$(sha256sum "$TEMP/release/dist/$asset" | awk '{print $1}')"
  else
    digest="$(shasum -a 256 "$TEMP/release/dist/$asset" | awk '{print $1}')"
  fi
  printf '%s  %s\n' "$digest" "$asset" >> "$TEMP/release/dist/checksums.txt"
done
GOKUB_ALLOW_DEV_PROVENANCE=1 ./scripts/verify-release-artifacts.sh v0.1.0 "$TEMP/release/dist" >/dev/null
if ./scripts/verify-release-artifacts.sh v0.1.0 "$TEMP/release/dist" >/dev/null 2>&1; then
  printf 'release verifier accepted development provenance\n' >&2
  exit 1
fi

printf 'corrupt' >> "$TEMP/release/dist/gokub_0.1.0_Linux_arm64.tar.gz"
if GOKUB_ALLOW_DEV_PROVENANCE=1 ./scripts/verify-release-artifacts.sh v0.1.0 "$TEMP/release/dist" >/dev/null 2>&1; then
  printf 'release verifier accepted a corrupted archive\n' >&2
  exit 1
fi

printf 'Packaging checks passed.\n'
