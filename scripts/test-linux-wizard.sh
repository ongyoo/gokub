#!/usr/bin/env bash

set -euo pipefail

if [[ "$(uname -s)" != "Linux" ]]; then
  printf 'Linux wizard smoke test skipped on %s.\n' "$(uname -s)"
  exit 0
fi

command -v script >/dev/null 2>&1 || {
  printf 'Linux wizard smoke test requires util-linux script.\n' >&2
  exit 1
}
command -v timeout >/dev/null 2>&1 || {
  printf 'Linux wizard smoke test requires timeout.\n' >&2
  exit 1
}

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEMP="$(mktemp -d "${TMPDIR:-/tmp}/gokub-linux-wizard.XXXXXX")"
trap 'rm -rf "$TEMP"' EXIT

cd "$ROOT"
go build -o "$TEMP/gokub" ./cmd/gokub

# Outside a project, Help is the sixth Command Center item.
printf '\033[B\033[B\033[B\033[B\033[B\r' |
  timeout 10s script -qec "cd '$TEMP' && '$TEMP/gokub'" /dev/null > "$TEMP/output.txt"

grep -Fq 'Choose workflow' "$TEMP/output.txt"
grep -Fq 'Commands' "$TEMP/output.txt"
grep -Fq 'start the step-by-step project wizard' "$TEMP/output.txt"

printf 'Linux arrow-key wizard smoke test passed.\n'
