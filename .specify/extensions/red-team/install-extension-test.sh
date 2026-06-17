#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
INSTALLER="$SCRIPT_DIR/install-extension.sh"

output="$("$INSTALLER" --dry-run 2>&1)" && {
  printf 'expected installer to fail without repo paths\n' >&2
  exit 1
}

if [[ "$output" != *"No repo paths supplied. Pass repo paths, use --repo PATH, or add entries to REPOS."* ]]; then
  printf 'unexpected installer output:\n%s\n' "$output" >&2
  exit 1
fi

if [[ "$output" == *"unbound variable"* ]]; then
  printf 'installer leaked nounset failure:\n%s\n' "$output" >&2
  exit 1
fi

for local_artifact in "$SCRIPT_DIR/AGENTS.md" "$SCRIPT_DIR/CLAUDE.md" "$SCRIPT_DIR/.claude"; do
  if [[ -e "$local_artifact" ]]; then
    printf 'local development artifact must not ship with extension: %s\n' "$local_artifact" >&2
    exit 1
  fi
done
