#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: harness/bin/build-agent-prompt.sh <topic>

Build the generation prompt packet for one harness topic.
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -ne 1 ]]; then
  usage >&2
  exit 2
fi

topic="$1"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
topic_prompt="$repo_root/harness/topics/$topic/prompt.md"

if [[ ! -f "$topic_prompt" ]]; then
  echo "Unknown topic: $topic" >&2
  echo "Available topics:" >&2
  find "$repo_root/harness/topics" -mindepth 1 -maxdepth 1 -type d -printf '  %f\n' | sort >&2
  exit 2
fi

emit_file() {
  local label="$1"
  local path="$2"

  printf '\n===== BEGIN FILE: %s =====\n' "$label"
  sed -n '1,$p' "$path"
  printf '\n===== END FILE: %s =====\n' "$label"
}

cat <<'HEADER'
# Blindbox Agent Packet

Use only the context in this packet. Do not rely on ambient repository files,
global skills, plugins, memories, custom agents, AGENTS.md, CLAUDE.md, or any
other local instruction files.

HEADER

emit_file "SKILL.md" "$repo_root/SKILL.md"
emit_file "references/layer-contract.md" "$repo_root/references/layer-contract.md"
emit_file "references/layer-prompts.md" "$repo_root/references/layer-prompts.md"
emit_file "harness/prompts/agent-run.md" "$repo_root/harness/prompts/agent-run.md"
emit_file "harness/topics/$topic/prompt.md" "$topic_prompt"
