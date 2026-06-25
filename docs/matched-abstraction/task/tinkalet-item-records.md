---
layer: task
topic: tinkalet-item-records
status: complete
references:
  - ../approach/tinkalet-edge.md
  - ../plan/tinkalet-edge.md
  - ./profile-trigger-tour.md
---

# Tinkalet Item Records Task

## Objective

Implement Slice 2 `item-records`: Tinkabot owns durable waitable item records,
and Tinkalet can create, get, resolve, and wait for them through product
commands without exposing raw KV bucket names.

## Artifact Pack

Item lifecycle:

```text
missing
  |
  | item create <key>
  v
pending ---- item resolve <key> ----> resolved
   |                                  ^
   | item wait <key> --for resolved   |
   +----------------------------------+

side statuses allowed for future slices: ready, working, failed, cancelled,
expired. This Task writes pending and resolved only.
```

Storage and revision authority:

```text
tinkalet command
  |
  | selected profile caller creds
  v
Tinkabot app account
  |
  | JetStream KV bucket: tb_items
  | key: product item key (deploy/123)
  | value: tinkabot.item.v1 JSON
  v
durable item truth

KV revision is the item revision. Resolve may pass --revision <n>; if the
stored revision changed, Tinkalet reports stale-revision.
```

Wait sequence:

```text
tinkalet wait deploy/123 --for resolved
    |
    | poll product item through profile
    v
pending ... pending ... resolved
    |
    v
prints resolved item and exits
```

Failure matrix:

| Family            | Product reason         | Proof owner |
| ---               | ---                    | ---         |
| duplicate create  | duplicate-item         | outside-in  |
| stale revision    | stale-revision         | outside-in  |
| denied neighbor   | denied-neighbor        | shared profile guard |
| malformed value   | malformed-value        | CLI unit    |
| expired/revoked   | revoked-credentials    | profile auth path |
| restart recovery  | item survives restart  | outside-in  |
| timeout           | wait-timeout           | item wait   |

Test intent map:

- Tinkalet unit: malformed value and no-profile item denial.
- Real embedded NATS: create/get/resolve/wait over a real Tinkabot profile.
- Revision: duplicate create and stale `--revision` resolve fail product-shaped.
- Restart: item remains readable after Tinkabot restart on the same store.
- Privacy: item JSON/text excludes raw `tb_items`, `$KV`, and credential data.

## Command Contract

```text
tinkalet item create <key> [--status pending] [--value <json>] [--json]
tinkalet item get <key> [--json]
tinkalet item resolve <key> [--value <json>] [--revision <n>] [--json]
tinkalet item wait <key> --for resolved [--timeout <duration>] [--json]
```

Human output is one line: `item <key> <status> rev <n>`. JSON output is the
stored product record with key, status, value, revision, provenance, and
timestamps.

## Acceptance Contract

- `go test ./tinkalet -run TestItemCommandDenials -count=1` passes.
- `go test ./tinkabot -run TestTinkaletItemRecords -count=1` passes.
- `go test ./... -count=1` passes from `substrate/go`.
  `bun run release:evidence` stay green.

## RED Artifact

Added 2026-06-17:

- `substrate/go/tinkalet/tinkalet_test.go` `TestItemCommandDenials`.
- `substrate/go/tinkabot/tinkalet_item_test.go` `TestTinkaletItemRecords`.

Expected RED before implementation:

- `go test ./tinkalet -run TestItemCommandDenials -count=1` fails because
  `tinkalet item` is not recognized.
- `go test ./tinkabot -run TestTinkaletItemRecords -count=1` fails because
  the item command and durable `tb_items` bucket do not exist.

## Verification Evidence

GREEN on 2026-06-17:

- `go test ./tinkalet -run TestItemCommandDenials -count=1` -> pass.
- `go test ./tinkabot -run TestTinkaletItemRecords -count=1` -> pass.
- `go test ./tinkabot -run 'TestTinkaletItemRecords|TestBinaryRestartReloadsWithoutRegeneration|TestLocalProfileDescriptor|TestTinkaletTriggerClock|TestTinkaletAuthorityDenials' -count=1` -> pass.
- `go test ./cmd/tinkalet ./tinkalet -count=1` -> pass.
- `go test ./... -count=1` from `substrate/go` -> pass.
- `bun run gate:scenarios` -> `gate:scenarios passed`.
- `bun run release:evidence` -> `release evidence check passed: 17 milestones over 12 spine steps, 6 gate results`.
- `bun test tests/gate-checkers.test.ts` -> 12 pass.
- `git diff --check` -> pass.

The restart proof exposed a lower-layer durability gap: app-plane JetStream
state was stored under an ephemeral app account public key. The fix persists
the built-in `TB_APP`/`TB_CONTROL` account keys and persists built-in account
revocations, while runtime-minted bundle accounts remain ephemeral. The binary
restart test now proves pre-stop caller creds stay denied after restart and
newly minted caller creds connect.

## Residual Risk

This Task intentionally polls for `item wait`; durable watch cursors belong to
Slice 3. It also grants caller-role item writes for the local profile tour;
finer item roles can land with remote pairing or broader profile work.
