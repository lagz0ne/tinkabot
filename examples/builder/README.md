# builder - a Tinkabot Bundle Example

One folder, one app, built by a chain reaction. A source script emits a tiny
Vite app as a projection; a long-lived Bun/Vite filter watches that projection
and rebuilds artifacts whenever it changes.

## Setup

Vite must resolve before the build filter can run:

```bash
cd examples/builder
bun install
```

## Run From Source

Then return to the repo root and start the bundle:

```bash
cd substrate/go
go run ./cmd/tinkabot --store /tmp/tb-builder --shell 127.0.0.1:8419 --bundle ../../examples/builder
```

Open:

```text
http://127.0.0.1:8419/artifacts/bundle/builder/index.html
```

The boot run of `scripts/source.sh` emits the source projection, the build
filter does a cold Vite build, and the page is served from produced artifacts.

## Poke The Source

Re-emit the source projection with a fresh timestamp; the filter does a warm
rebuild and overwrites the artifacts in place. From the release package root or
source checkout root in another terminal:

```bash
NATS=./libexec/tinkabot/nats # release package root
# NATS=$(cd tools/natscli && go tool -n nats) # source checkout
CLIENT_URL=nats://127.0.0.1:4222 # replace with the printed "nats" URL

"$NATS" --no-context --server "$CLIENT_URL" \
  --creds /tmp/tb-builder/caller.creds \
  --timeout 2s \
  request --raw -H Tinkabot-Request-Id:req-builder-1 \
  tb.bundle.builder.source go
# -> accepted; the artifacts are rebuilt
```

Reload the tab. The built app does not poll, so the new time and color appear
on a fresh load. Build status is visible at:

```text
http://127.0.0.1:8419/projections/bundle.builder.built
```

## Anatomy

- `bundle.json`: strictly decoded manifest. Entry `source` in bundle `builder`
  gets trigger `tb.bundle.builder.source`, projection ids under
  `bundle.builder.`, and artifacts under `bundle/builder/`.
- `scripts/source.sh`: emits a length-framed JSON effect containing an app
  source map under the short projection id `src`.
- `scripts/build.ts`: long-lived filter run with `bun`. The platform pipes one
  JSON line per `src` change into stdin; the filter runs programmatic Vite,
  emits artifact frames for output files, then emits `bundle.builder.built`.
