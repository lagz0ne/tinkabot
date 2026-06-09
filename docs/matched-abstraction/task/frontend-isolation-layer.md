---
layer: task
topic: frontend-isolation-layer
references:
  - ../approach/browser-isolation.md
  - ../plan/browser-isolation.md
  - ./browser-isolation-proof.md
---

# Frontend Isolation Layer Task

Diagram: https://diashort.apps.quickable.co/d/2a2abd49

## Objective

Build the frontend-owned isolation layer: a Vite trusted shell, generated artifact fixture, opaque sandboxed iframe, leased source-window message path, raw-authority denial, and Go-embedded frontend distribution.

## Scope

This task owns:

- Vite app structure under `apps/frontend`.
- frontend isolation contract for sandbox policy, frame lease, source-window binding, nonce, schema revision, artifact revision, and capability context.
- generated artifact fixture loaded as an opaque blob document.
- shell proof state for browser automation.
- Go `embed.FS` package for the built frontend distribution.
- workspace scripts for frontend test, typecheck, and build.

## Non-Goals

- No direct browser NATS WebSocket.
- No service-worker implementation.
- No gateway CSRF/origin/fetch-metadata implementation.
- No product UI beyond the proof shell.
- No per-app origin fleet.
- No activation router, script execution, or materializer implementation.

## Acceptance Contract

- The shell renders generated content in an iframe with effective `sandbox="allow-scripts"` and no same-origin token.
- The shell does not hand generated content substrate cookies, tokens, NATS credentials, subjects, or permissions.
- The shell accepts generated messages only from the leased frame source with the expected nonce, frame id, schema revision, artifact revision, and allowed command.
- `expectedRevision` is bound to the leased artifact revision before the shell emits trusted command intent.
- Raw NATS-shaped vocabulary from generated content is denied before any trusted command effect.
- Accepted generated intent becomes canonical `browser.command_intent` shape with trusted context.
- Vite build emits the frontend distribution into the Go frontend package.
- Go can embed and read the built frontend index and assets.

## RED Artifact

Expected failing proof before implementation:

- `T-FRONTEND-SANDBOX`: no Vite shell proves effective script-only opaque iframe sandbox.
- `T-FRONTEND-LEASE`: no frontend contract denies wrong source, bad nonce, stale revision, or disallowed command.
- `T-FRONTEND-RAW-DENY`: generated content can still send raw NATS-shaped fields without frontend-owned denial.
- `T-FRONTEND-BROWSER-SMOKE`: browser automation cannot observe accepted typed intent, raw-authority denial, leased source binding, and empty ambient credential probe.
- `T-GO-FRONTEND-EMBED`: Go cannot embed the built frontend distribution.

## Execution Notes

The proof uses a blob-backed generated artifact fixture so browser automation can observe the shell contract without depending on a finished artifact gateway. The gateway and service-worker parts of `browser-isolation-proof` remain separate work.

The frontend shell does not mint cookies for the proof. HttpOnly, Secure, SameSite cookie issuance and denial behavior belong to the real gateway proof because JavaScript-created cookies are not a valid security oracle.

## Verification Evidence

RED/GREEN:

- `subagent verification` -> NO-GO until structured-clone raw authority, expected revision binding, and cookie-proof overclaim were corrected.
- `bun run test:frontend` -> `3 pass`, `0 fail`, `15 expect() calls`.
- `bun run --cwd apps/frontend typecheck` -> `passed`.
- `bun run build:frontend` -> `vite v7.3.5 built frontend into substrate/go/frontend/site`.
- `go test ./frontend -count=1` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/frontend`; embedded `index.html` references existing built assets.
- `agent-browser open http://127.0.0.1:5173/ && agent-browser eval 'window.__tinkabotProof'` -> sandbox `allow-scripts`, ready source `true`, accepted `1`, denied `1`, no credential material provided by the shell.
- `agent-browser screenshot /tmp/tinkabot-frontend-isolation.png` -> `saved`.
- `bun run test` -> `55 pass`, `0 fail`, `389 expect() calls`.
- `bun run typecheck` -> `passed`.
- `bun run build` -> frontend Vite build and SDK tsdown build passed.
- `bun run schema:parity` -> endgame contract tests `21 pass`, Go packages `contract`, `core`, `edge`, `embednats`, and `frontend` passed.
- `bun run test:e2e` -> `1 pass`, `0 fail`, `16 expect() calls`.
- `bun run pack:dry` -> `6 files`, unpacked size `188.70KB`.
- `bun run validate:layers` -> `Layer validation passed: docs/matched-abstraction`.
- `bun run test:layers` -> `Ran 10 tests ... OK`.
- `git diff --check` -> `clean`.

## Wrap-Up Announcement

The frontend-owned isolation layer is complete when the Vite shell proves opaque generated-content execution, leased source-window messaging, raw-authority denial, typed browser command output, and Go embeddable build output.
