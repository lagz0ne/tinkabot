---
layer: task
topic: nats-script-runtime-distribution-bdd
references:
  - ../approach/platform-structure.md
  - ../plan/nats-script-runtime-traced-tdd.md
  - ../plan/nats-script-runtime.md
  - ../approach/nats-script-runtime.md
---

# NATS Script Runtime Distribution BDD Task

## Platform Reset Supersession

`docs/matched-abstraction/approach/platform-structure.md` supersedes this task wherever substrate ownership or final platform structure is concerned. This task remains historical distribution evidence for the earlier Bun package slice.

## Task Brief

Create a final-form distribution build for the current NATS script runtime slice and verify it with an end-to-end BDD scenario. This task stays at the distribution boundary: it packages the existing substrate and record-store contract, then proves a consumer can use the built artifacts.

Scope includes package distribution metadata, build configuration, root exports, generated `dist` files, and the BDD scenario under `tests/e2e/`.

## Acceptance Contract

The historical task was accepted when `bun run build` emitted ESM, CommonJS, and declaration artifacts under `dist`; package metadata pointed consumers at those artifacts; a pack dry-run included only package metadata and `dist`; and an executable BDD scenario imported from `dist`, started embedded JetStream NATS, wrote and read exact script record revisions, observed a deleted-record error, and stopped cleanly.

The BDD scenario must not import runtime source at execution time. Source type imports are allowed for test authoring only.

## RED Artifact

- `bun test tests/e2e/nats-script-runtime-distribution.bdd.test.ts` -> failed with `Script not found "build"` before distribution metadata and build config existed.

## Execution Notes

The GREEN pass added:

- `src/index.ts` as the package root export.
- `tsdown.config.ts` using the same ESM, CommonJS, declarations, Node target, and clean build pattern as the local embedded-NATS package.
- package `version`, `main`, `module`, `types`, `exports`, `files`, `build`, and `test:e2e` fields.
- `tests/e2e/nats-script-runtime-distribution.feature.md` as the BDD scenario capture.
- `tests/e2e/nats-script-runtime-distribution.bdd.test.ts` as the executable scenario.
- generated `dist/index.mjs`, `dist/index.cjs`, `dist/index.d.mts`, and `dist/index.d.cts`.

Review kept the scenario at the distribution boundary. It does not add metadata/schema, permission, IPC, process runtime, event trail, execution exchange, or vertical script execution behavior.

## Verification Evidence

- `bun run build` -> emitted `dist/index.mjs`, `dist/index.cjs`, `dist/index.d.mts`, and `dist/index.d.cts`.
- `bun test tests/e2e/nats-script-runtime-distribution.bdd.test.ts` -> `1 pass, 0 fail`.
- `bun test` -> `8 pass, 0 fail`.
- `bun run typecheck` -> `bunx @typescript/native-preview --noEmit` completed successfully.
- `bun pm pack --dry-run` -> `Total files: 5`.
- `python3 -B .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction` -> `Layer validation passed: docs/matched-abstraction`.
- `python3 -B -m unittest tests/test_validate_layers.py` -> `Ran 10 tests ... OK`.
- `find . -type d -name __pycache__ -print` -> no matches.

## Wrap-Up Announcement

The final response must state how to build the distribution, how to run the BDD scenario, and that the next traced-TDD runtime slice remains metadata/schema and imports/permissions.
