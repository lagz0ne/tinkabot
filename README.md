# Tinkabot

Tinkabot runs generated automation as software without handing raw authority to
the generated code. A generated app is a bundle: plain backend processes emit
framed effects on stdout, frontend files are served as sandboxed artifacts, and
the trusted substrate decides what can trigger, write, read, and observe.

The substrate is a single Go binary with embedded NATS, operator/JWT auth, a
trusted browser shell, mediated script execution, and bundle sandboxing through
Bubblewrap. Generated code never receives NATS credentials or store handles.

## Quick Start

Current release channel: GitHub Release archive:

```bash
curl -LO https://github.com/lagz0ne/tinkabot/releases/download/v0.1.0/tinkabot-v0.1.0-linux-amd64.tar.gz
curl -LO https://github.com/lagz0ne/tinkabot/releases/download/v0.1.0/tinkabot-v0.1.0-linux-amd64.tar.gz.sha256

sha256sum -c tinkabot-v0.1.0-linux-amd64.tar.gz.sha256
tar -xzf tinkabot-v0.1.0-linux-amd64.tar.gz
cd tinkabot-v0.1.0-linux-amd64
./tinkabot --version
```

Run directly from the unpacked package:

```bash
./tinkabot --store /tmp/tb-clock --shell 127.0.0.1:8419 --bundle examples/clock
```

Open:

```text
http://127.0.0.1:8419/artifacts/bundle/clock/index.html
```

The package includes `libexec/tinkabot/bwrap` for sandboxing and
`libexec/tinkabot/nats` for operator commands.

To put `tinkabot` on `PATH`, keep the package directory intact and symlink the
binary:

```bash
mkdir -p ~/.local/opt ~/.local/bin
mv tinkabot-v0.1.0-linux-amd64 ~/.local/opt/
ln -sfn ~/.local/opt/tinkabot-v0.1.0-linux-amd64/tinkabot ~/.local/bin/tinkabot
tinkabot --version
```

This layout matters: the binary discovers bundled sidecars relative to itself
or its symlink target. A plain `go install` only installs the Go executable and
does not install the sandbox, NATS CLI sidecar, examples, or release metadata.

## Source Checkout

Prereqs for a source checkout:

- Go matching `substrate/go/go.mod`
- Bun
- Bubblewrap (`bwrap`) on Linux for sandboxed bundles

Build a local release-shaped package:

```bash
bun install
bun run pack:tinkabot /tmp/tinkabot
```

Run the clock bundle:

```bash
/tmp/tinkabot/tinkabot \
  --store /tmp/tb-clock \
  --shell 127.0.0.1:8419 \
  --bundle examples/clock
```

Open:

```text
http://127.0.0.1:8419/artifacts/bundle/clock/index.html
```

Runtime lookup order for Bubblewrap is `TB_BWRAP`, then the bundled sidecar,
then `PATH`; sandbox preflight still fails closed if the host cannot run
Bubblewrap namespaces. For a trusted local demo on a host without working
Bubblewrap, add `--bundle-sandbox none`.

Build a GitHub-Release-shaped archive:

```bash
bun run release:package dist/release
```

## Drive It

The binary prints its NATS client URL and role creds on startup. Use the bundled
NATS CLI sidecar instead of installing a global CLI:

```bash
NATS=./libexec/tinkabot/nats # release package root
# NATS=/tmp/tinkabot/libexec/tinkabot/nats # local package from source
CLIENT_URL=nats://127.0.0.1:4222 # replace with the printed "nats" URL

"$NATS" --no-context --server "$CLIENT_URL" \
  --creds /tmp/tb-clock/caller.creds \
  --timeout 2s \
  request --raw -H Tinkabot-Request-Id:req-clock-1 \
  tb.bundle.clock.tick go
```

The clock also ticks itself every five seconds. The page polls the derived
projection and updates without receiving credentials.

## What Is Proven

- **Authority is derived.** A bundle manifest names local entries and local
  projections; the substrate derives script keys, trigger subjects, projection
  ids, and artifact paths.
- **Effects are mediated.** Scripts emit framed JSON effects; the substrate
  validates policy before writing projections or artifacts.
- **Generated UI is sandboxed.** Bundle pages are served under the trusted shell
  as untrusted artifacts.
- **Bundles are isolated.** Bundle state lives in its own runtime-minted NATS
  account; the app account observes only through explicit exports/imports.
- **Sandboxing is fail-closed.** The default bundle tier requires Bubblewrap
  preflight before generated processes run.
- **Operator proof is reproducible.** Packages include the pinned NATS CLI
  sidecar used by the source checkout's `tools/natscli` proof path.

## Checks

```bash
bun run gate:manual
bun run release:evidence
bun run release:package dist/release
bun run validate:layers
cd substrate/go && go test ./... -count=1
```

`gate:manual` runs the manual's documented command/outcome pairs against a real
binary using the pinned NATS CLI tool. `release:evidence` validates the release
manifest and gate results.

## Examples

- [examples/clock](examples/clock/README.md): smallest complete bundle. A shell
  script emits state and a page; a long-lived filter derives the view consumed
  by the frontend.
- [examples/builder](examples/builder/README.md): advanced bundle. A source
  script emits a tiny Vite app; a Bun/Vite filter rebuilds artifacts when the
  source projection changes.

## Repository Map

- `substrate/go`: Go binary, embedded NATS, auth, bundle runtime, sandboxing
- `examples`: release-shaped bundle examples
- `docs/manual/v1.md`: operator manual and command surface
- `release/v1.json`: release evidence manifest
- `tools/natscli`: pinned NATS CLI Go tool used by proofs
- `packages/sdk`, `schemas/base/v1`: shared contract surface
- `apps/frontend`: trusted browser shell
- `docs/matched-abstraction`: design, plan, and task evidence

## License

MIT. See [LICENSE](LICENSE).

## Current Limits

- No npm wrapper yet; `bun run release:package` creates the GitHub Release
  archive locally.
- Default bundle sandboxing depends on host support for Bubblewrap namespaces.
- The browser shell is functional proof infrastructure, not a polished product
  UI.
- HA and multi-node operation are contract-shaped, not production-operated.
