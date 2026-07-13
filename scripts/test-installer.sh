#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

bash -n install.sh

if GOKUB_REPOSITORY='owner/repo/extra' ./install.sh >/dev/null 2>&1; then
  printf 'installer accepted an invalid repository\n' >&2
  exit 1
fi

if GOKUB_VERSION='1.2.3' ./install.sh >/dev/null 2>&1; then
  printf 'installer accepted a version without a v prefix\n' >&2
  exit 1
fi

grep -Fq -- "--proto '=https' --proto-redir '=https'" install.sh
grep -Fq -- 'tar -xzf "${TEMP_DIR}/${ARCHIVE}" -C "$TEMP_DIR" -- "$PROJECT"' install.sh
grep -Fq -- '! -L "${TEMP_DIR}/${PROJECT}"' install.sh
grep -Fq -- '^[0-9a-fA-F]{64}$' install.sh

printf 'Installer security checks passed.\n'
