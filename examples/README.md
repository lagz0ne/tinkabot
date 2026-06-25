# Tinkabot Examples

Examples are bundle directories: one folder contains a strict `bundle.json`,
backend scripts, and any frontend assets those scripts emit at runtime.

Start with [clock](clock/README.md). It is the smallest release-shaped app:
state script -> derived view filter -> sandboxed page.

Use [builder](builder/README.md) when you want to see a warmer chain reaction:
a source projection feeds a long-lived Bun/Vite filter, which rebuilds frontend
artifacts in place.

Use the packaged demos when you want participant proofs rather than a bundle
example: `bun run demo:turn` for the turn-based flow and `bun run demo:realtime`
for the fast scoped-participant action stream plus terminal result proof.
Use `bun run demo:iso-concurrency` when you want the multitenant proof: one
daemon, two app scopes, four Tinkalet participant profiles, restart/reconnect,
and observed-only rate/latency metrics.

Use Tinkalet for product commands:

```bash
TINKALET_CONFIG_DIR=/tmp/tinkalet-config \
TINKALET_DATA_DIR=/tmp/tinkalet-data \
  ./tinkalet profile import local --store /tmp/tb-clock --name local

TINKALET_CONFIG_DIR=/tmp/tinkalet-config \
TINKALET_DATA_DIR=/tmp/tinkalet-data \
  ./tinkalet trigger bundle.clock.tick
```

Use `tinkalet schedule set <name> --every <duration> --write <item-key>` for
product item schedules. The packaged NATS CLI sidecar is not part of the
normal user, LLM, or transform integration path; keep it for owner/operator
diagnostics and low-level config inspection only:

```bash
NATS=./libexec/tinkabot/nats # release package root
# NATS=$(cd tools/natscli && go tool -n nats) # source checkout
```
