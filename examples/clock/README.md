# clock — a tinkabot bundle example

One folder, one app: a backend script wired to a trigger, and a frontend page
the script emits as an artifact. Nothing is installed — the bundle is served
ephemerally for the lifetime of the run; restart without `--bundle` and it is
gone.

## Run

```bash
cd substrate/go
go run ./cmd/tinkabot --store /tmp/tb-clock --shell 127.0.0.1:8419 --bundle ../../examples/clock
```

Open http://127.0.0.1:8419/artifacts/bundle/clock/index.html — the page is
emitted by the boot run of `scripts/tick.sh` and polls the backend's
projection every 2 seconds.

## Poke the backend

The bundle ticks itself every 5s (`"every": "5s"` in the manifest); the page
shows the clock advancing on its own. Fire a tick manually any time:

```bash
nats request --creds /tmp/tb-clock/caller.creds -H Tinkabot-Request-Id:req-1 tb.bundle.clock.tick go
# -> accepted; the page picks up the new renderedAt/unix within 2s
```

Control the schedule through NATS settings — plain caller authority on the
config bucket:

```bash
nats kv put config_bucket bundle.clock.tick.every off --creds /tmp/tb-clock/caller.creds   # pause
nats kv put config_bucket bundle.clock.tick.every 1s --creds /tmp/tb-clock/caller.creds    # retune
nats kv del config_bucket bundle.clock.tick.every --creds /tmp/tb-clock/caller.creds       # back to manifest
```

## Anatomy

- `bundle.json` — strictly decoded manifest. Authority is derived, never
  declared: entry `tick` in bundle `clock` gets trigger
  `tb.bundle.clock.tick`, script key `scripts.bundle.clock.tick`, projection
  ids under `bundle.clock.`, and artifacts under `bundle/clock/` — a
  manifest cannot even spell a collision with durable claims. `boot: true`
  fires the entry once at startup.
- `scripts/tick.sh` — a plain process emitting length-framed JSON effects on
  stdout; it never sees NATS, credentials, or store handles.
