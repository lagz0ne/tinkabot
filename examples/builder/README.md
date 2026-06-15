# builder — a tinkabot bundle example

One folder, one app, built by a chain reaction. A source script emits the app
as a projection; a long-lived Vite filter watches that projection and rebuilds
the app whenever it changes; the built files land as artifacts. Nothing is
installed into the platform — the bundle is served ephemerally for the lifetime
of the run; restart without `--bundle` and it is gone.

## Setup

Vite must resolve before the build filter can run. Install it first — this is
required:

```bash
cd examples/builder
bun install
```

## Run

```bash
cd substrate/go
go run ./cmd/tinkabot --store /tmp/tb-builder --shell 127.0.0.1:8419 --bundle ../../examples/builder
```

Open http://127.0.0.1:8419/artifacts/bundle/builder/index.html — the boot run
of `scripts/source.sh` emits the source projection, the build filter does a cold
Vite build, and the page is served from the produced artifacts. The page shows
the timestamp it was built from on a colored background derived from that time.

## Poke the source

Re-emit the source projection with a fresh timestamp; the filter does a warm
rebuild (~35ms) and overwrites the artifacts in place:

```bash
nats request --creds /tmp/tb-builder/caller.creds -H Tinkabot-Request-Id:req-1 tb.bundle.builder.source go
# -> accepted; the artifacts are rebuilt
```

Then RELOAD THE TAB — the built app does not poll, so the new time and hue only
appear on a fresh load. Build status (files emitted, build ms) shows at
http://127.0.0.1:8419/projections/bundle.builder.built.

## Anatomy

- `bundle.json` — strictly decoded manifest. Authority is derived, never
  declared: entry `source` in bundle `builder` gets trigger
  `tb.bundle.builder.source`, projection ids under `bundle.builder.`, and
  artifacts under `bundle/builder/`. `boot: true` fires `source` once at
  startup so the app exists immediately. The bundle uses LOCAL refs only:
  scripts emit short projection ids (`src`, `built`) and relative artifact
  names (Vite builds with `base: "./"`, so asset URLs need no bundle name);
  the substrate resolves each to the derived global name (`bundle.builder.src`,
  `bundle/builder/<relpath>`).
- `scripts/source.sh` — a plain process emitting a length-framed JSON effect on
  stdout; it never sees NATS, credentials, or store handles. Writes the app
  source map to the short id `src` (resolved to `bundle.builder.src`).
- `scripts/build.ts` — a long-lived filter run with `bun`: the platform pipes
  one JSON line per `src` change into its stdin, it runs a programmatic Vite
  build, emits one artifact frame per output file (stable names, so each
  rebuild overwrites in place), then a `bundle.builder.built` projection frame.
  Chain-reaction: source projection -> watch -> warm Vite rebuild -> artifacts
  -> `/artifacts/bundle/builder/index.html`. The first build is cold; later
  rebuilds run warm at roughly 35ms.
