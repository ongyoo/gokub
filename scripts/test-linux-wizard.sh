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

# The Command Center loops until the user leaves. Number keys select a menu item
# directly, which is robust to reordering: outside a project Help is item 7 and
# Exit is item 8 (see commandCenterActions(false) in internal/cli). Pick Help to
# render the command list, then Exit so the program terminates on its own.
printf '78' |
  timeout 15s script -qec "cd '$TEMP' && '$TEMP/gokub'" /dev/null > "$TEMP/output.txt"

grep -Fq 'Choose workflow' "$TEMP/output.txt"
grep -Fq 'Commands' "$TEMP/output.txt"
grep -Fq 'start the step-by-step project wizard' "$TEMP/output.txt"

printf 'Linux arrow-key wizard smoke test passed.\n'
