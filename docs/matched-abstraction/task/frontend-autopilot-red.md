---
layer: task
topic: frontend-autopilot-red
references:
  - ../../tasks/tinkabot-frontend-autopilot-okr.md
  - ../approach/tinkalet-edge.md
  - ../approach/bundle-v1.md
  - ../plan/bundle-v1.md
---

# Frontend Autopilot RED

Status: terminal GREEN reached on 2026-06-25. This file preserves the first RED
task and now records the final closure evidence instead of redefining the OKR.

## Scope

Add the first outside-in proof for the frontend-autopilot OKR. This task does
not implement the handler registry or app creation path; it names the first
missing product command and records that RED in a durable proof file.

## Acceptance

- `bun run demo:frontend-autopilot` builds the release-shaped package.
- The script starts packaged Tinkabot from a clean store.
- The script imports and selects a Tinkalet profile.
- The script disables the packaged raw NATS sidecar before user-level Tinkalet
  commands.
- The initial RED attempts the product grammar:
  `tinkalet app handler register vite --from examples/builder --json`.
- After the handler registration slice, the proof advanced to
  `tinkalet app create frontend options-site --handler vite --json`.
- After the app creation slice, the proof now advances to
  `frontend_autopilot_first_failure: "kv-reaction-journey-not-implemented"`.

## RED Evidence

Current expected command:

```bash
bun run demo:frontend-autopilot
```

Expected initial RED result before the first Tinkalet slice: non-zero exit with
`vite-handler-registration-missing`. After `app handler register` and
`app create frontend` land, the same command should still exit non-zero but
advance the proof to:

```json
{
  "kind": "tinkabot.frontendAutopilotProof.v1",
  "pass": false,
  "clean_install_to_kv_reaction_journey_passing": false,
  "frontend_autopilot_reference_families": 3,
  "frontend_autopilot_first_failure": "kv-reaction-journey-not-implemented"
}
```

The partial family count means the clean install/profile path and handler
registration/app creation reached the next RED point. It does not count
objective success.

## Implementation Notes

The proof intentionally fails before any UI authoring. That keeps the
Claude-Opus UI anti-goal intact: Codex may add the proof harness, but must not
write the eventual frontend UI artifact.

If the handler registration command later passes, the script should continue to
the next observable family rather than being removed. Future GREEN work should
extend this same proof until it reaches the terminal KV-watch LLM reaction.

## GREEN Closure

Final proof:
`/tmp/tinkabot-frontend-autopilot.MZToam/frontend-autopilot-proof.json`.

The same `bun run demo:frontend-autopilot` harness now passes with:

- `frontend_autopilot_reference_families: 7`
- `clean_install_to_kv_reaction_journey_passing: true`
- `post_install_user_command_count: 3`
- `manual_unblock_count: 0`
- `non_nats_product_path_count: 0`
- `authority_leak_count: 0`
- created app record driving `generatedPath` and `resultKey`
- visual state delivery through `trusted-shell.nats-watch.push`
- Claude Opus UI generation/provenance/review hash-bound to the packaged UI
- live `tinkalet watch prefix` piped into Claude Opus for the LLM reaction

Final independent reviews:

- Claude Opus via `claude -p`: `VERDICT: PASS`
- read-only Codex follow-up: `VERDICT: PASS`
