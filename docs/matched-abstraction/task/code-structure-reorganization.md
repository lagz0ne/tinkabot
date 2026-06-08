---
layer: task
topic: code-structure-reorganization
references:
  - ../approach/code-structure.md
  - ../plan/code-structure.md
  - ../task/platform-structure-reset.md
---

# Code Structure Reorganization Task

## Task Brief

Move the current TypeScript package out of root into SDK package ownership while keeping root commands as orchestration. Add concise ownership notes for the future frontend, Go substrate, and schema lanes.

## Acceptance Contract

- Root package metadata is workspace orchestration, not the published SDK package.
- Existing TypeScript source, TypeScript tests, build config, and distribution output live under SDK package ownership.
- Root no longer has `src`, `dist`, `tsconfig.json`, or `tsdown.config.ts`.
- Root keeps Python layer-validator tests.
- The current package name and public exports are preserved inside the SDK package.
- Future frontend, Go substrate, and schema lanes exist with ownership notes but no fake implementation code.
- Root commands still verify the moved SDK package.
- Workspace `.gitignore` covers dependency folders, build output, dry-pack artifacts, Bun install backups, Python bytecode, local env files, and editor noise.

## RED Artifact

- `find . -maxdepth 3 -type f | sort | sed -n '1,220p'` -> showed root-owned `src`, root-owned TypeScript test folders, root `tsdown.config.ts`, root `tsconfig.json`, and root `dist`.
- `bun test` -> baseline `27 pass`, `0 fail`.
- `bun run typecheck` -> baseline `bunx @typescript/native-preview --noEmit`.
- `bun run build` -> baseline emitted root `dist` artifacts.

## Execution Notes

The move creates workspace lanes and keeps behavior stable. The SDK package keeps the current `tinkabot` package name to avoid a distribution rename in this structure slice.

`packages/sdk` owns the current TypeScript package. Root delegates build, test, typecheck, e2e, and dry-pack commands into that package.

`apps/frontend`, `substrate/go`, and `schemas` contain ownership notes only.

## Verification Evidence

- `bun install` -> `Saved lockfile`.
- `bun test` -> `27 pass`, `0 fail`.
- `bun run typecheck` -> `bunx @typescript/native-preview --noEmit`.
- `bun run build` -> emitted `packages/sdk/dist/index.mjs`, `packages/sdk/dist/index.cjs`, `packages/sdk/dist/index.d.mts`, and `packages/sdk/dist/index.d.cts`.
- `bun run pack:dry` -> `Total files: 6`.
- `find . -maxdepth 1 -type d -name src -print` -> no output.
- `find . -maxdepth 1 -type d -name dist -print` -> no output.
- `find . -maxdepth 1 -type f -name tsconfig.json -print` -> no output.
- `find . -maxdepth 1 -type f -name tsdown.config.ts -print` -> no output.
- `find packages/sdk -maxdepth 1 -type f -name "*.tgz" -print` -> no output.
- `rg -n "node_modules/|dist/|coverage/|\\.tgz|\\.old_modules|__pycache__|\\.pytest_cache|\\.env|\\.DS_Store" .gitignore` -> matched ignore rules for generated workspace artifacts.
- `git rev-parse --is-inside-work-tree` -> `fatal: not a git repository`; direct `git check-ignore` verification is not available in this workspace.
- `bun run test:layers` -> `Ran 10 tests ... OK`.
- `python3 -B .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction` -> `Layer validation passed: docs/matched-abstraction`.

## Wrap-Up Announcement

The root-owned TypeScript package has been moved into SDK package ownership. Root is now workspace orchestration for the current slice.
