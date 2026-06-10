# Tinkabot

Tinkabot runs scripts and generated web UI without trusting them. Scripts are plain processes that talk to the platform over stdin/stdout; generated UI runs in a locked-down iframe. Neither ever holds credentials, picks message subjects, or writes data directly. The platform sits in the middle: it checks who asked, runs the work, and stores the results durably. Everything rides on an embedded NATS server, so you can watch and drive the whole system with the standard `nats` CLI.

**Start here:** `docs/manual/v1.md` — how to define a script, trigger it, and read the results. Every command in it is copied from a test that actually ran.

## What's here today

This is a verified source checkout, not an installable tool yet. There is no `tinkabot` binary — the platform currently runs embedded inside Go, and its behavior is proven by tests that drive it end to end with the real `nats` CLI. The single-binary version is the next program of work, and the manual is the contract it has to satisfy.

Flow diagram: https://diashort.apps.quickable.co/d/bb63b165

## How it works

- **Scripts are records.** A script is a JSON document in a key-value store: which command to run, with what limits and cleanup. The process emits results as framed JSON on stdout; the platform decides what gets stored.
- **Triggers are explicit.** Work starts from a request, a published message, a key-value change, a file/object upload, a stream message, or a schedule tick. Each trigger is checked against its credentials, deduplicated, and recorded durably before anything runs.
- **Results are durable.** Accepted outputs become projections and artifacts in NATS-backed stores, with full attribution: who triggered it, with which credential, in which causal chain. A script's raw output is never the source of truth — only what the platform accepts.
- **Credentials are leases.** Every actor (caller, script, browser session, observer) gets a scoped, revocable lease, not a permanent key. Denials beat grants, and being one subject away from your grant means rejection.
- **The browser is split in two.** A trusted shell talks to the server; generated content is sandboxed, sees no credentials, and can only propose typed commands the shell forwards. The server accepts or rejects each command durably.

## Try it

You need Bun, Go, and the `nats` CLI.

```bash
bun install

# Check that every claim in this repo is backed by a test that ran
bun run release:evidence
# -> release evidence check passed: 16 milestones over 11 spine steps

# Watch the whole loop: CLI request -> script runs -> stored projection,
# artifact, and status -- plus blocked writes and duplicate rejection
cd substrate/go
go test ./embednats -run TestScriptMaterializerLoopFromNATSCLI -count=1 -v
```

More checks:

```bash
bun run test          # TypeScript + frontend tests
bun run schema:parity # same contracts validate identically in TS and Go
cd substrate/go && go test ./...   # full Go suite over real embedded NATS
```

Note: `nats` CLI v0.3.0 prints permission errors but still exits 0, so denial checks read the output text, not the exit code.

## Not done yet

- No installable binary or published package (this checkout is the product)
- No real product UI (the browser shell is a proof page)
- Browsers don't connect to NATS directly yet
- Scripts run as trusted local processes — no Docker/sandbox isolation yet
- Credentials load at server start — no live reload or mid-session revocation push
- Schedules need the host to feed clock ticks — no built-in wall-clock loop
- Single node only — no clustering
- No script management UI

## Where things live

- `docs/manual/v1.md` — user manual
- `release/v1.json` — the evidence map: every feature claim linked to the test that proves it (`bun run release:evidence` checks it)
- `schemas/base/v1` — the JSON contracts everything validates against, plus fixtures
- `substrate/go` — the Go platform: embedded NATS, stores, auth, script runner, browser gateway
- `packages/sdk` — TypeScript side of the contracts
- `apps/frontend` — the trusted browser shell (Vite)
- `docs/matched-abstraction` — design and evidence documents (some use the project's old internal name "endgame"; live code was renamed to base/v1)
- `tasks/todo.md` — current working state and next steps
