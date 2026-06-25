---
layer: task
topic: browser-frontend-dedicated-worker
references:
  - ../approach/browser-frontend-mediator.md
  - ../plan/browser-frontend-mediator.md
---

# Browser Frontend Dedicated Worker Task

## Task Brief

Implement the first managed frontend mediator contract. The unit owns pure TypeScript message validation, mediator intent stamping, materializer state, and a fakeable dedicated-worker bridge.

## Acceptance Contract

- Generated content command messages are validated before transport.
- Mediated command intents include trusted session, capability, artifact, frame, revision, and chain context.
- Raw NATS vocabulary from generated content is rejected.
- Disallowed command names are rejected.
- Materializer state ignores stale projection sequence updates.
- Worker bridge posts accepted status for valid commands and error messages for invalid commands.

## RED Artifact

`tests/browser-frontend/dedicated-worker-mediator.test.ts` is added before implementation and initially fails because `src/browser-frontend/index.ts` does not exist.

## Execution Notes

Keep the slice browser-runtime agnostic. Fake the worker scope and command transport in tests. Do not import the NATS client or start a browser.

## Verification Evidence

- RED: `bun test tests/browser-frontend/dedicated-worker-mediator.test.ts` failed before implementation with `Export named 'createFrontendMediator' not found in module '/home/lagz0ne/dev/tinkabot/src/index.ts'`.
- GREEN: `bun test tests/browser-frontend/dedicated-worker-mediator.test.ts` -> `4 pass, 0 fail`.
- Full test: `bun test` -> `27 pass, 0 fail`.
- Typecheck: `bun run typecheck` -> `bunx @typescript/native-preview --noEmit`.
- Build: `bun run build` -> emitted ESM, CommonJS, and declaration artifacts.
- Pack dry-run: `bun pm pack --dry-run` -> `Total files: 5`.
- Cleanup check: `find . -type d -name __pycache__ -print` -> no output.

## Wrap-Up Announcement

The first managed frontend dedicated-worker mediator slice is implemented as a pure TypeScript contract with fakeable transport and worker scope. It does not include a real browser app, live NATS WebSocket, credential issuer, artifact gateway, or generated iframe.
