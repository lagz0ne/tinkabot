---
layer: task
topic: web-session-surface
status: complete
references:
  - ../approach/session-v2.md
  - ../plan/session-v2.md
---

# Web Session Surface Task

## Brief

Session-v2 slice 7/7, including program release closure. Owns the direct-NATS
browser observation surface in two hops, per Plan slice 7.

Hop one (substrate + binary): the shell HTTP server establishes an HttpOnly,
SameSite=Strict cookie session on first contact (`tb_shell`, backed by a
JetStream KV store); the mint endpoint `POST /session/viewer` exchanges a
valid cookie for an ephemeral viewer credential — a **bearer** JWT
(`BearerToken=true`, no signing seed ever exists browser-side), short-TTL
(10m), source-pinned to loopback CIDRs, leaf-scoped to subscribe on the
viewer's own deliver subject (`tb.session.<id>.deliver.<nonce>`) plus
`_INBOX.>`, and publish on `tb.app.browser.command` (command acceptance)
only — never the steering subject, never JetStream API authority. The deliver
subject is fed by a substrate-bound push consumer over the session output
stream (`BindViewerDeliver`, DeliverAll = the slice-3 snapshot-plus-tail
attach shape). The WS route `GET /session/ws` is cookie-gated (401 without a
valid cookie) and pipes the upgraded connection to the embedded NATS loopback
WebSocket listener, replaying only the WebSocket handshake headers — the
cookie never crosses into NATS; nats-server's `jwt_cookie` option is not the
mechanism. Renewal is a re-mint through the cookie path (the slice-4 renewal
semantic); per-session authorization is NATS denied-neighbor enforcement.

Hop two (trusted shell): the frame lease in `apps/frontend/src/isolation.ts`
gains `sessions: readonly string[]` (the session observation scope alongside
the command allowlist); `accept()` rejects content intents naming a session
outside the scope with the typed `FrameScopeEscape` error, and `mayObserve`
gates outbound frame forwarding. Untrusted app content never holds any
credential.

Closure: `scenario-matrix.json` gains the `web-session-surface` surface with
all seven pinned families citing committed Go tests; `release/v1.json` gains
the `web-session-surface` milestone over a new `session-v2` spine step;
`scripts/release-evidence.ts` MILESTONES/SPINE grow accordingly and
`direct-browser-nats-websocket` is removed from DEFERRED — the deferral is
retired at the loopback posture by Approach invariant 8's condition citations
(live reload, post-connection revocation, denied-neighbor, stale-access
proven by operator-jwt-authority; confidentiality satisfied at loopback);
external/TLS forms stay deferred.

Carry-note resolution (from slice 6): viewers trust **subject/stream
identity** — the deliver consumer is bound per session stream at mint time —
never the frame body's `sessionId` field, which remains unvalidated data at
the mediator.

## Acceptance Contract

- `go test ./embednats -run TestWebSessionSurface -count=1` passes: a bearer
  viewer credential connects over the embedded NATS WebSocket listener and
  observes a mediated session through its deliver subject including frames
  published before attach (ViewerObserves); a viewer for session A is denied
  session B's deliver and out subjects and denied steer publish
  (CrossSessionLeak); expiry, live revocation, reconnect denial, re-mint
  renewal, and cookie revocation hold (StaleViewerCred); mint misuse is typed
  (ViewerMintFailed); the scenario matrix and release manifest closure checks
  hold (GateMatrixVacuous, UngrownManifest).
- `go test ./tinkabot -run TestWebSessionShell -count=1` passes: the shell
  issues the HttpOnly SameSite=Strict cookie (CookieIssued); a WS upgrade
  without or with a forged cookie is 401 and a cookied upgrade is 101 with
  the NATS INFO banner arriving through the proxy, twice with the same cookie
  (UngatedUpgrade, CookieGatedUpgrade); the mint endpoint denies missing
  cookies (401) and malformed bodies (400), and its grant observes a mediated
  session over WS and lands a steer intent on `tb.app.browser.command`
  (MintEndpoint).
- `bun test apps/frontend/tests` passes: `FrameScopeEscape` denial and
  in-scope acceptance, `mayObserve` allowed/denied (hop two owner).
- `bun run release:evidence` passes over the extended manifest (17
  milestones, 12 spine steps) with the deferral retirement recorded.
- All five standing gates pass with the surface entry resolving to committed
  Go tests.

## RED Artifact

Executed 2026-06-12 (strengthened RED after a first GREEN under-delivered the
Plan — see Drift Notes):

`cd substrate/go && go test ./embednats -run TestWebSessionSurface -count=1`
-> compile failure: `too many arguments in call to MintViewerCredential`,
`viewer.DeliverSubject undefined`, `viewer.JWT undefined`, `undefined:
BindViewerDeliver` — the bearer/deliver-consumer surface did not exist.

`cd substrate/go && go test ./tinkabot -run TestWebSessionShell -count=1` ->
`no tb_shell cookie session issued`, `upgrade without cookie session must be
401, got "HTTP/1.1 404 Not Found"` — no cookie session, no WS route, no mint
endpoint on the shell server.

## Drift Notes

A first implementation pass (workflow main run) passed its own six subtests
but under-delivered the Plan; the slice was re-driven under a strengthened
RED. Defects fixed:

1. No WebSocket surface existed anywhere — the "upgrade" proof tested a bare
   function returning `(bool, int)`; the binary served no WS route and never
   enabled the embednats WebSocket listener.
2. The viewer credential granted publish on the session steering subject
   (violating Approach invariant 1's single steering writer); steering now
   rides `tb.app.browser.command` into the proven mediated path, and the
   steer-publish denial is part of CrossSessionLeak.
3. The viewer subscribed `tb.session.<id>.out` directly (live-tail only)
   instead of the Plan's deliver subject fed by a substrate-bound consumer —
   transcript replay on attach was silently lost.
4. No bearer mode: the credential still shipped an nkey seed, which invariant
   8 forbids browser-side.
5. A parallel Go `FrameLease` struct was invented; the real lease is the TS
   `Lease` in `apps/frontend/src/isolation.ts`, now extended in place.

## Verification Evidence

GREEN executed 2026-06-12.

`cd substrate/go && go test ./embednats -run TestWebSessionSurface -count=1`
-> `ok` (all six sub-tests; full embednats suite 15.9s) — ViewerObserves:
bearer JWT (no seed) over the WS listener, deliver subject
`tb.session.<id>.deliver.<nonce>`, frame published before attach delivered
through the substrate-bound DeliverAll consumer, live frame after attach
delivered; denied-neighbor: viewer-A credential denied subscribe on session-B
deliver and out subjects, and denied publish on the steering subject
(permission denials output-parsed from the async error handler);
stale: expired short-TTL viewer credential denied connect; revoked viewer
credential disconnected live and denied reconnect; renewal by re-mint through
a validated cookie connects; revoked cookie no longer validates; attributed
failure: viewer mint misuse returns typed ViewerMintFailed (non-operator
runtime and empty session id).

`cd substrate/go && go test ./tinkabot -run TestWebSessionShell -count=1` ->
`ok 0.28s` — CookieIssued: `tb_shell` Set-Cookie with HttpOnly and
SameSite=Strict; malformed: upgrade without a cookie and with a forged cookie
returns 401, malformed mint body returns 400; CookieGatedUpgrade: 101
Switching Protocols with the NATS INFO banner through the piped backend
connection, duplicate upgrade with the same cookie succeeds; MintEndpoint:
mint without cookie 401, valid grant carries bearer JWT plus deliver subject,
the grant observes a mediated frame over the embedded WebSocket listener, and
its steer intent arrives on `tb.app.browser.command` via request/reply.

`bun test apps/frontend/tests` -> `7 pass, 0 fail` — FrameScopeEscape steer
denial and in-scope acceptance, `mayObserve` true/false; workspace `bun run
test` -> `103 pass`.

`cd substrate/go && go test ./... -count=1` -> all 9 packages ok.

`bun run release:evidence` -> passes: 17 milestones over 12 spine steps, 5
gate results, `direct-browser-nats-websocket` removed from deferred scope
with the retirement record (`retiredDeferrals`) citing the Plan's condition
citations.

Negative cases named in this executed evidence (denial oracles output-parsed,
never exit-code):

- denied-neighbor: viewer-A denied subscribe on session-B subjects, denied steer publish.
- malformed: forged cookie upgrade returns 401, malformed mint body returns 400.
- stale: expired short-TTL viewer credential denied connect.
- revoked: viewer credential disconnected live and denied reconnect.
- attributed: viewer mint misuse returns typed ViewerMintFailed.

## Scope

Owns:

- `substrate/go/embednats/web_session_surface.go` — `ViewerCred`,
  `MintViewerCredential` (bearer re-sign, loopback Src pin, leaf scope),
  `BindViewerDeliver`, cookie session store
  (`IssueSessionCookie`/`ValidateCookieSession`/`RevokeCookieSession`,
  JetStream KV), `ViewerMintFailed` Kind.
- `substrate/go/tinkabot/web_session.go` + shell route dispatch — cookie
  issuance, `POST /session/viewer`, cookie-gated `GET /session/ws` proxy;
  WebSocket listener enabled in the binary's embednats config.
- `apps/frontend/src/isolation.ts` lease `sessions` scope, `mayObserve`,
  `FrameScopeEscape` error kind; frontend tests.
- Closure: `scenario-matrix.json` surface entry, `release/v1.json` milestone
  + `session-v2` spine step + deferral retirement,
  `scripts/release-evidence.ts` MILESTONES/SPINE/DEFERRED.

Does not own:

- Raw terminal/PTY mode, transcript redaction, multi-viewer fanout tuning,
  session resume UX, session list/CRUD UI, external/TLS exposure (all named
  deferred in the Plan).
- The session subsystem, mediation, steering, or wrapper proofs (slices 1-6,
  consumed unchanged).
- Live-agent CI (slice 6's exclusion stands).

## Addendum (2026-06-12, browser demo)

The first real-browser run of this surface (demo-gated observe panel) found
that browsers do not reliably attach SameSite cookies to `ws://` upgrade
handshakes — both Strict and Lax were omitted by the test browser while the
same cookie rode `fetch` requests fine. The cookie gate therefore gained a
derived transport: the cookie-gated mint endpoint also returns a single-use,
30s-TTL upgrade ticket (`wsTicket`), and `GET /session/ws` accepts a valid
cookie OR a valid ticket (`?t=`). The gating authority is unchanged — the
ticket is obtainable only through the cookie session — and the posture is
strictly narrower (single-use, short TTL). Proven by
`TestWebSessionShell/TicketGatedUpgrade` (cookieless upgrade with fresh
ticket succeeds; reuse and forgery are 401) and by the live browser run:
`TB_DEMO_SESSION=demo-001` binary, observe panel streaming the demo session
with full replay-on-reload through the DeliverAll consumer.

## Residual Risk

- The browser composition (automatic cookie on same-origin WS upgrade +
  bearer CONNECT from page script) is proven link-wise in Go — cookie gate,
  proxy pipe to NATS, bearer connect, denied-neighbor — not as a single
  browser run; the trusted shell UI that would exercise it stays deferred
  with product UI rendering.
- `ValidateCookieSession` opens a fresh minted connection per check; fine at
  loopback proof scale, a cache seam if the shell ever serves real traffic.
- Hijacked WS proxy connections are not tracked by `http.Server.Close`; they
  end when either side closes. The binary's process exit bounds them.
