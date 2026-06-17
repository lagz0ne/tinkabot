# clock - a Tinkabot Bundle Example

One folder, one app: a backend script wired to a trigger, a long-lived filter
that derives a display view, and a frontend page emitted as an artifact.
Nothing is installed into the platform. The bundle is served ephemerally for
the lifetime of the run; restart without `--bundle` and it is gone.

## Run From Source

```bash
cd substrate/go
go run ./cmd/tinkabot --store /tmp/tb-clock --shell 127.0.0.1:8419 --bundle ../../examples/clock
```

Or run through a local package from the repo root:

```bash
bun run pack:tinkabot /tmp/tinkabot
/tmp/tinkabot/tinkabot --store /tmp/tb-clock --shell 127.0.0.1:8419 --bundle examples/clock
```

Open:

```text
http://127.0.0.1:8419/artifacts/bundle/clock/index.html
```

The page is emitted by the boot run of `scripts/tick.sh` and polls the derived
view every two seconds.

## Poke The Backend

The bundle ticks itself every five seconds (`"every": "5s"` in the manifest).
From the release package root in another terminal, import the local profile and
fire a tick manually:

```bash
TINKALET_CONFIG_DIR=/tmp/tinkalet-config \
TINKALET_DATA_DIR=/tmp/tinkalet-data \
  ./tinkalet profile import local --store /tmp/tb-clock --name local

TINKALET_CONFIG_DIR=/tmp/tinkalet-config \
TINKALET_DATA_DIR=/tmp/tinkalet-data \
  ./tinkalet profile use local

TINKALET_CONFIG_DIR=/tmp/tinkalet-config \
TINKALET_DATA_DIR=/tmp/tinkalet-data \
  ./tinkalet trigger bundle.clock.tick --request-id req-clock-1
# -> profile local accepted bundle.clock.tick
```

The page picks up the new renderedAt/unix within 2s. Control the schedule
through NATS settings for now, using plain caller authority on the config
bucket:

```bash
NATS=./libexec/tinkabot/nats # release package root
# NATS=/tmp/tinkabot/libexec/tinkabot/nats # local package from source
CLIENT_URL=nats://127.0.0.1:4222 # replace with the printed "nats" URL

"$NATS" --no-context --server "$CLIENT_URL" \
  --creds /tmp/tb-clock/caller.creds \
  --timeout 2s \
  kv put config_bucket bundle.clock.tick.every off

"$NATS" --no-context --server "$CLIENT_URL" \
  --creds /tmp/tb-clock/caller.creds \
  --timeout 2s \
  kv put config_bucket bundle.clock.tick.every 1s

"$NATS" --no-context --server "$CLIENT_URL" \
  --creds /tmp/tb-clock/caller.creds \
  --timeout 2s \
  kv del config_bucket bundle.clock.tick.every
```

## Anatomy

- `bundle.json`: strictly decoded manifest. Authority is derived, never
  declared. Entry `tick` in bundle `clock` gets trigger
  `tb.bundle.clock.tick`, script key `scripts.bundle.clock.tick`, projection
  ids under `bundle.clock.`, and artifacts under `bundle/clock/`.
- `scripts/tick.sh`: a plain process emitting length-framed JSON effects on
  stdout. It never sees NATS, credentials, or store handles. It writes raw state
  to the short id `state`, resolved by the substrate to `bundle.clock.state`.
- `scripts/present.sh`: a long-lived filter. The platform pipes one JSON line
  per state change into stdin; the filter emits the short `view`, resolved to
  `bundle.clock.view`. The page consumes only the view, not raw state.
