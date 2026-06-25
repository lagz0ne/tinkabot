# builder - a Tinkabot Bundle Example

One folder, one app, built by a chain reaction. A source script emits a tiny
Vite app as a projection; a long-lived Bun/Vite filter watches that projection
and rebuilds artifacts whenever it changes. The generated browser app watches
its own built projection and refreshes the tab when the running bundle rebuilds,
so the browser tracks the NATS reaction chain like a tiny Vite-style loop.

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
Keep the tab open for the next step; it should refresh itself after the patch.

## Poke The Source

Re-emit the source projection with a fresh timestamp; the filter does a warm
rebuild and overwrites the artifacts in place. From the release package root in
another terminal, drive the bundle through Tinkalet instead of a raw NATS
subject:

```bash
export TINKALET_CONFIG_DIR=/tmp/tinkalet-builder-config
export TINKALET_DATA_DIR=/tmp/tinkalet-builder-data

./tinkalet profile import local --store /tmp/tb-builder --name local
./tinkalet profile use local
./tinkalet trigger bundle.builder.source --request-id req-builder-1
# -> profile local accepted bundle.builder.source
```

From a source checkout without a package root, run the same `tinkalet`
subcommands as `go run ./cmd/tinkalet ...` from `substrate/go`.

The open tab polls its scoped `_p/built` projection, sees the new
`artifactRevision`, and reloads against the updated artifact. Build status is
visible at:

```text
http://127.0.0.1:8419/projections/bundle.builder.built
```

## Anatomy

- `bundle.json`: strictly decoded manifest. Entry `source` in bundle `builder`
  gets trigger `tb.bundle.builder.source`, projection ids under
  `bundle.builder.`, and artifacts under `bundle/builder/`.
- `scripts/source.sh`: emits a length-framed JSON effect containing an app
  source map under the short projection id `src`. The source includes the
  browser-side `_p/built` watcher that refreshes the already-open app after a
  rebuild.
- `scripts/build.ts`: long-lived filter run with `bun`. The platform pipes one
  JSON line per `src` change into stdin; the filter runs programmatic Vite,
  emits artifact frames for output files, then emits `bundle.builder.built`.

## Packaged Live Patch Demo

From the repo root:

```bash
bun run demo:patch
```

The harness builds the release-shaped package, starts packaged `tinkabot` with
the builder bundle, imports and selects the emitted local profile with packaged
`tinkalet`, disables the packaged NATS CLI sidecar, waits for the cold Vite
build, opens Chromium against the shown URL, triggers
`bundle.builder.source` through Tinkalet, and verifies that the already-loaded
browser tab refreshes while the served JS artifact and `bundle.builder.built`
projection advance in place. When Tailscale is available it prints and tests
the MagicDNS URL. Use
`TINKABOT_DEMO_PATCH_DELAY=5 TINKABOT_DEMO_HOLD=1 bun run demo:patch` to open
the page before the patch and watch the tab refresh.

## Three Viewpoints

- Chain setup: `tinkabot --bundle examples/builder` owns embedded NATS,
  materializes the bundle profile, runs `source`, and keeps the `build` filter
  attached to `bundle.builder.src`.
- Tinkalet setup: `tinkalet profile import local --store ...` copies the caller
  credential into its managed data dir, then `profile use local` selects it for
  later product commands.
- Use: `tinkalet trigger bundle.builder.source` asks the chain to re-emit
  source. The hidden substrate subject is derived as `tb.bundle.builder.source`;
  callers do not need server URLs, credential paths, or NATS request syntax.
