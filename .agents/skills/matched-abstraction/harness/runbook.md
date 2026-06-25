# Harness Runbook

Use this harness to compare real agent outputs against the matched-abstraction skill. The generation run stops at Plan. Task work must be derivable from the Plan, but no Task documents, code, or execution steps should be produced during the generation run.

The default model set should stay small. Prefer current Kilo frontier/open-weight style models such as:

- `kilo/~moonshotai/kimi-latest` or a concrete Kimi model.
- `kilo/z-ai/glm-5.2`.
- `kilo/qwen/qwen3.7-max`.
- `kilo/deepseek/deepseek-v4-pro`.

Avoid broad free-model sweeps unless specifically requested. Do not include OpenAI or Anthropic generation models in the default comparison set.

## Generate

Prefer the blindbox runner so the agent sees only the prompt packet and an empty home:

```bash
harness/bin/run-blindbox.sh --agent kilo --topic car-parking-system --auth session --model kilo/~moonshotai/kimi-latest --label eval-car-kimi
harness/bin/run-blindbox.sh --agent kilo --topic car-parking-system --auth session --model kilo/z-ai/glm-5.2 --label eval-car-glm
harness/bin/run-blindbox.sh --agent kilo --topic car-parking-system --auth session --model kilo/qwen/qwen3.7-max --label eval-car-qwen
harness/bin/run-blindbox.sh --agent kilo --topic car-parking-system --auth session --model kilo/deepseek/deepseek-v4-pro --label eval-car-deepseek
```

The runner uses `bwrap`, does not mount the repository root, and sets `HOME`, `CODEX_HOME`, Claude config paths, and Kilo data/config paths to sandbox directories. By default, auth comes from whitelisted environment variables such as `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, and Kilo-related env vars. With `--auth session`, the runner copies only the known session credential file for the selected agent into a temporary directory, mounts that temporary copy into the sandbox, and deletes it after the run. Codex session auth stages `~/.codex/auth.json`. Claude session auth stages `~/.claude/.credentials.json`. Kilo session auth stages `~/.local/share/kilo/auth.json` and optional `~/.config/kilo/kilo.jsonc`.

It prevents project and global instruction files such as `AGENTS.md`, `CLAUDE.md`, global skills, plugins, hooks, memories, and host config files from being read. Claude runs through normal Claude Code print mode while keeping `--safe-mode`, disabled slash commands, empty setting sources, no session persistence, and no tools. Kilo runs through `kilo run --format json` with the `ask` agent by default, extracts text events into the final `.md`, and keeps the raw JSON events in `.stdout.txt`. It does not remove the vendor's built-in model or CLI system prompt.

The only persistent writable mount is `harness/runs/`. The prompt packet is mounted read-only as `/prompt.md`; the local Node/Codex runtime, Claude binary, and Kilo package root are mounted read-only under `/opt`.

Use `--agent kilo-free` only when broad free-model spread is explicitly requested. It runs every model currently marked `isFree: true` by `kilo models --verbose`. Use `--refresh-models` when you want Kilo to refresh model metadata before resolving free models. Because free endpoints can be slow or flaky, prefer a per-run timeout and preflight:

```bash
free_regex=$(kilo models --verbose | awk '/^kilo\\// { model=$0 } /"isFree": true/ { gsub(/[][(){}.^$*+?|\\\\]/, "\\\\&", model); print "^" model "$" }' | paste -sd '|' -)
kilo roll-call "$free_regex" --prompt 'Reply with exactly: ok' --timeout 30000 --parallel 5 --quiet --output md
```

If one Kilo free model fails during `--agent kilo-free`, the runner records the failure on stderr, continues through the remaining free models, and exits non-zero after the fan-out completes.

For each agent and topic, provide only:

- `SKILL.md`
- `references/layer-contract.md`
- `references/layer-prompts.md`
- `harness/prompts/agent-run.md`
- One topic prompt from `harness/topics/<topic>/prompt.md`

Do not provide the rubric, topic notes, or review lenses to the generating agent. Those are evaluator inputs.

Save each agent output outside the prompt packet with a stable name such as:

```text
harness/runs/2026-06-17-codex-car-parking-system.md
harness/runs/2026-06-17-claude-car-parking-system.md
harness/runs/2026-06-17-kilo-free-kilo-auto-free-car-parking-system.md
```

## Review

For each output, provide the reviewer:

- The agent output.
- The original topic prompt.
- `harness/rubric.md`
- `harness/review-lenses.md`
- The matching `harness/topics/<topic>/rubric-notes.md`

Use `harness/prompts/reviewer-run.md` when asking an agent to review. Human review can use the same packet.

Use the helper so outputs and sidecars are named consistently. For the default Kilo-only eval path, use a Kilo reviewer:

```bash
harness/bin/review-output.sh --topic car-parking-system --output harness/runs/<label>.md --label <label> --reviewer kilo --model kilo/z-ai/glm-5.2
```

## Compare

Compare Codex, Claude, and Kilo on:

- Total score and any score caps.
- Missing Plan content that would prevent downstream Task delivery.
- Places where the agent descended into task execution despite the Plan-only boundary.
- Places where the agent should have returned upward because Plan details broke an Approach assumption.
- Differences in artifact set fitness, especially use-now/deferred/upstream decisions, depth stops, derive/match hints, diagrams, matrices, and verification plans.
- Kilo free-model spread: which free models complete, which time out, and whether low-cost/free models systematically miss the same layer-contract expectations.

When a failure appears in both agents, revise the skill. When a failure appears in one agent only, keep the output as evidence but avoid overfitting the skill unless the failure repeats on another topic.

## Release

The Claude Code plugin version lives in `.claude-plugin/plugin.json`. Keep the root Codex skill files and the Claude plugin skill copy in sync before release:

```bash
diff -u SKILL.md skills/matched-abstraction/SKILL.md
diff -u references/layer-contract.md skills/matched-abstraction/references/layer-contract.md
diff -u references/layer-prompts.md skills/matched-abstraction/references/layer-prompts.md
```

Validate the Claude plugin:

```bash
claude plugin validate --strict .
claude --plugin-dir "$PWD" plugin details matched-abstraction@inline
```

After committing the release, create the Claude plugin release tag:

```bash
claude plugin tag --dry-run .
claude plugin tag .
```
