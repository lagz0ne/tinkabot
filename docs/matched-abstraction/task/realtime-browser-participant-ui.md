---
layer: task
topic: realtime-browser-participant-ui
references:
  - ../../../tasks/tinkabot-objective-okr.md
  - ./browser-participant-action-bridge.md
  - ./realtime-participant-reference-demo.md
  - ./realtime-participant-action-gap.md
---

# Realtime Browser Participant UI

## Scope

CKR-REALTIME needed the browser/UI side of the participant path, not just a
Tinkalet or Go harness. This slice makes the embedded trusted shell dispatch
accepted generated-frame participant intents to the NATS-backed browser command
service, then returns the backend response to the generated iframe so the UI can
render action/readback progress.

This remains a generic substrate proof. It does not add a typeracing service,
score service, game engine, custom realtime channel, or raw NATS credential for
generated UI.

## Contract

```text
packaged Tinkabot
  -> shell URL reachable through Tailscale
  -> trusted shell mints cookie-gated browser command credential
  -> generated iframe receives app/participant lease
  -> generated iframe posts participant_read / participant_action intents
  -> trusted shell accepts and dispatches to tb.app.browser.command
  -> backend materializes participant action items through existing app-action service
  -> generated iframe reads back its own action material and updates DOM counters
```

The proof must fail if the Tailscale shell URL is not reachable, if the
generated iframe receives raw NATS authority, if the shell cannot dispatch
through the cookie-gated browser command path, if either participant misses an
action/readback, if any dispatch is denied, or if product proof output leaks raw
credential, bucket, `$KV`, or NATS URL material.

## RED Boundary

- Public command: `bun run demo:realtime-browser`.
- Expected RED before implementation: the generated iframe could emit accepted
  participant intents, but `apps/frontend/src/main.ts` only counted them; it did
  not request/reply through `tb.app.browser.command` or return command results
  to generated UI.
- First release-shaped RED after wiring the script: Tailscale HTTP origin
  crashed the shell on `crypto.randomUUID()` because that API is unavailable on
  non-secure non-localhost origins.

## GREEN Boundary

The slice is green only when:

- The release archive is built and the packaged binary starts with
  `demo:alice` and `demo:bob` scoped participants.
- The shell URL used by the browser proof is the Tailscale URL when Tailscale is
  available.
- Two browser pages run the generated iframe UI, one leased as Alice and one as
  Bob.
- Each page performs one state read, 20 participant actions, and 20 own-action
  readbacks through the trusted shell.
- The generated UI DOM reports complete status, 20 actions, 20 readbacks, and
  zero denials for both participants.
- A malicious `tb_session` query string is assigned through DOM properties, not
  interpolated into trusted-shell HTML.
- Browser command proof reports `acceptedActions=40`, `readbacks=40`,
  `deniedDispatches=0`, `authorityLeakCount=0`, and latency p95 under the
  configured proof thresholds.

## GREEN Evidence

- `bun test apps/frontend/tests/isolation.test.ts apps/frontend/tests/observe.test.ts`
  -> `11 pass`, proving isolation still enforces leased commands and the command
  reply decoder handles NATS byte payloads.
- `bunx @typescript/native-preview --noEmit -p apps/frontend/tsconfig.json`
  -> pass.
- `bun run build:frontend` -> Vite rebuilt embedded assets under
  `substrate/go/frontend/site`.
- First `TINKABOT_DEMO_BROWSER_ACTIONS=20 TINKABOT_DEMO_BROWSER_INTERVAL_MS=20 bash scripts/demo-realtime-browser.sh /tmp/tinkabot-realtime-browser-green`
  -> RED timeout. Focused inspection over the Tailscale URL showed
  `TypeError: crypto.randomUUID is not a function`; local loopback had already
  proven the action/readback path, so the failure was the Tailscale origin.
- Codex review found three issues: trusted-shell `tb_session` query text was
  interpolated into `innerHTML`, frontend/Go raw-authority filters were weaker
  than the SDK substring rule for composite keys such as `natsSubject`, and the
  proof did not assert generated-iframe counters directly. GREEN fixed all
  three: `tb_session` is assigned as an input property, frontend/Go raw filters
  use substring matching with tests, and the generated iframe exposes/asserts
  `actions`, `readbacks`, and `denied` counters.
- Claude review then found five hardening gaps: a rejected browser command
  connection promise could be cached, observe connections could accumulate,
  leak detection missed credential aliases, generated action errors could deref
  a missing key, and backend command replies were forwarded without the trusted
  shell's raw-authority filter. GREEN fixed all five with client reset,
  connection cleanup, broader leak terms, guarded action keys, and outbound
  response filtering.
- Corrected proof:
  `TINKABOT_DEMO_BROWSER_ACTIONS=20 TINKABOT_DEMO_BROWSER_INTERVAL_MS=20 bash scripts/demo-realtime-browser.sh /tmp/tinkabot-realtime-browser-green5`
  -> pass.
- Latest proof:
  `/tmp/tinkabot-realtime-browser-green5/realtime-browser-proof.json`.
  Metrics: Tailscale route `http://forge.tail6c789a.ts.net:43315`,
  `browserPages=2`, `acceptedActions=40`, `readbacks=40`,
  `deniedDispatches=0`, `authorityLeakCount=0`, action latency p95 `6ms`,
  action p99 `9ms`, readback p95 `7ms`, readback p99 `10ms`,
  `shellInjection.valueRoundTrip=true`, `shellInjection.injectedImages=0`,
  `shellInjection.xssFlag=false`, and Alice/Bob DOM status `complete` with 20
  actions, 20 readbacks, and 0 denials each.
- Final gates after fixes: focused frontend tests, frontend typecheck,
  frontend build, focused browser command Go test, focused realtime/authority Go
  suite, package-level Go suite, full `go test ./...`, script syntax checks,
  `git diff --check`, C3 line coverage `owned_files=491`, `lookup_errors=0`,
  `uncovered=0`, focused C3 evals, and `c3 check --include-adr` all passed.
- Independent review: Codex final review returned `VERDICT: PASS`, findings
  none; Claude final review returned `VERDICT: PASS`, zero new security
  findings, and confirmed the eight prior findings closed. Residuals remain
  non-mission claims: this is a trusted-shell-mediated prerequisite, not an
  arbitrary browser identity proof or full scoped multiplayer completion.

## Non-Goals

- No raw NATS credential for generated content.
- No direct generated-frame WebSocket or NATS connection.
- No example-specific game primitive.
- No max participant or max-rate freeze beyond the measured two-page browser
  proof.
- No complete scoped multiplayer mission claim until the final realtime-heavy
  reference mission combines browser UI, participant sync, terminal result, and
  mission documentation.
