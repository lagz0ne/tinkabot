#!/usr/bin/env bash
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
cd "$ROOT"

C3X_VERSION=${C3X_VERSION:-11.3.0}
if [ -z "${C3X_BIN:-}" ]; then
  cached="$HOME/.cache/c3x/$C3X_VERSION/c3x-$C3X_VERSION-linux-amd64"
  if [ -x "$cached" ]; then
    C3X_BIN=$cached
  else
    C3X_BIN=c3x
  fi
fi

C3=(env C3X_MODE=agent C3X_VERSION="$C3X_VERSION" "$C3X_BIN")
OUT=${C3_COVERAGE_OUT:-/tmp/tinkabot-c3-harness-results.txt}
OWNED=${C3_COVERAGE_OWNED:-/tmp/tinkabot-owned-files.txt}
UNCOVERED=${C3_COVERAGE_UNCOVERED:-/tmp/tinkabot-c3-uncovered.txt}
ERRORS=${C3_COVERAGE_ERRORS:-/tmp/tinkabot-c3-lookup-errors.txt}
ERROR_LOG=${C3_COVERAGE_ERROR_LOG:-/tmp/tinkabot-c3-lookup-error-output.txt}

: > "$OUT"
: > "$UNCOVERED"
: > "$ERRORS"
: > "$ERROR_LOG"

{
  printf '## c3 check --include-adr\n'
  "${C3[@]}" check --include-adr

  printf '\n## c3 eval\n'
  "${C3[@]}" eval

  printf '\n## representative lookups\n'
  for f in \
    substrate/go/tinkabot/bundle.go \
    apps/frontend/src/isolation.ts \
    schemas/base/v1/contract.schema.json \
    scripts/release-evidence.ts \
    scripts/c3-line-coverage-harness.sh \
    .agents/skills/c3/SKILL.md \
    .c3/README.md \
    CLAUDE.md \
    .gitignore \
    tools/natscli/go.mod; do
    printf '\n### %s\n' "$f"
    "${C3[@]}" lookup --json "$f"
  done
} >> "$OUT" 2>&1

{
  git ls-files -z
  git ls-files --others --exclude-standard -z
} | while IFS= read -r -d '' f; do
  case "$f" in
    .git/*|node_modules/*|*/node_modules/*|dist/*|*/dist/*|build/*|*/build/*|coverage/*|*/coverage/*|tmp/*|*/tmp/*|vendor/*|*/vendor/*|.c3/c3.db)
      continue
      ;;
  esac

  if [ -e "$f" ] || [ -L "$f" ]; then
    printf '%s\0' "$f"
  fi
done | sort -zu | tr '\0' '\n' > "$OWNED"

while IFS= read -r f; do
  if ! lookup=$("${C3[@]}" lookup --json "$f" 2>&1); then
    printf '%s\n' "$f" >> "$ERRORS"
    {
      printf '\n### %s\n' "$f"
      printf '%s\n' "$lookup"
    } >> "$ERROR_LOG"
    continue
  fi
  if printf '%s\n' "$lookup" | grep -q 'matches\[0\]'; then
    printf '%s\n' "$f" >> "$UNCOVERED"
  fi
done < "$OWNED"

{
  printf '\n## owned-file lookup coverage\n'
  printf 'owned_files=%s\n' "$(wc -l < "$OWNED")"
  printf 'lookup_errors=%s\n' "$(wc -l < "$ERRORS")"
  printf 'uncovered=%s\n' "$(wc -l < "$UNCOVERED")"
  if [ -s "$ERRORS" ]; then
    printf '\n### lookup error files\n'
    sed -n '1,200p' "$ERRORS"
    printf '\n### lookup error output\n'
    sed -n '1,200p' "$ERROR_LOG"
    exit 1
  fi
  if [ -s "$UNCOVERED" ]; then
    printf '\n### uncovered files\n'
    sed -n '1,200p' "$UNCOVERED"
    exit 1
  fi
} >> "$OUT"

printf 'wrote %s\n' "$OUT"
