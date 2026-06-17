---
layer: task
topic: tinkalet-authority-denials
status: complete
references:
  - ../approach/tinkalet-edge.md
  - ../plan/tinkalet-edge.md
  - ./profile-trigger-tour.md
  - ./tinkalet-cli-profile.md
  - ./tinkalet-local-profile-source.md
  - ./tinkalet-trigger-live.md
---

# Tinkalet Authority Denials Task

## Objective

Complete Slice 1 sub-Task 4: Tinkalet's trigger path must fail in product
language for the most important authority problems, without leaking raw NATS
subjects, inboxes, credential material, permission dumps, or stronger-role
fallback behavior.

## Scope

Owns:

- Revoked caller credential denial.
- Denied-neighbor/tampered profile denial.
- Stale managed credential denial even when stronger credentials are present.
- Normal-output privacy for denial paths.

Does not own full remote pairing, arbitrary profile trust models, schedule/item
denials, package archive smoke, or release promotion.

## Acceptance Contract

- `go test ./tinkabot -run TestTinkaletAuthorityDenials -count=1` from
  `substrate/go` passes.
- Revoked caller credentials fail as
  `profile local denied bundle.clock.tick: revoked-credentials\n`.
- A profile whose stored server endpoint no longer matches its local profile
  source fails as `denied-neighbor` before Tinkalet tries ambient authority.
- Removing the managed caller credential fails as `stale-credentials` even when
  `author.creds` is available beside it.
- Normal denial output contains no credential contents, raw `tb.` subjects,
  `_INBOX`, operator key paths, or NATS permission dumps.

## RED Artifact

RED tests were added in `substrate/go/tinkabot/tinkalet_denial_test.go`.

Executed 2026-06-17 from `substrate/go`:

- `go test ./tinkabot -run TestTinkaletAuthorityDenials -count=1` -> expected
  failure before implementation because revoked/neighbor auth errors still
  collapse to `connection-failed`.

## Verification Evidence

- `go test ./tinkabot -run TestTinkaletAuthorityDenials -count=1` -> RED:
  revoked and denied-neighbor cases collapsed to `connection-failed`.
- `go test ./tinkabot -run TestTinkaletAuthorityDenials -count=1` -> `ok`
  after Tinkalet classified local-source auth failures as revoked credentials,
  rejected tampered local-source server mismatches as denied-neighbor, and kept
  stale managed credential behavior from falling back to stronger creds.

## Execution Notes

Implemented 2026-06-17 in `substrate/go/tinkalet`: local profile server
mismatch against the source descriptor returns `denied-neighbor`; auth failures
for matching local profiles return `revoked-credentials`; missing managed
credential copies return `stale-credentials`; and normal denial output renders
only profile, product intent, and reason code.

## Residual Risk

This Task does not prove malformed responder bodies or every possible NATS
permission error. Package/archive and release-gate evidence remain separate.
