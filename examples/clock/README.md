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

```bash
nats request --creds /tmp/tb-clock/caller.creds -H Tinkabot-Request-Id:req-1 tb.bundle.clock.tick go
# -> accepted; the page picks up the new renderedAt/unix within 2s
```

## Anatomy

- `bundle.json` — strictly decoded manifest: each entry binds one script file
  to one wired trigger with its grants (projection ids, artifact prefix) and
  an optional `boot` flag fired once at startup.
- `scripts/tick.sh` — a plain process emitting length-framed JSON effects on
  stdout; it never sees NATS, credentials, or store handles.
