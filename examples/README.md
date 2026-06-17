# Tinkabot Examples

Examples are bundle directories: one folder contains a strict `bundle.json`,
backend scripts, and any frontend assets those scripts emit at runtime.

Start with [clock](clock/README.md). It is the smallest release-shaped app:
state script -> derived view filter -> sandboxed page.

Use [builder](builder/README.md) when you want to see a warmer chain reaction:
a source projection feeds a long-lived Bun/Vite filter, which rebuilds frontend
artifacts in place.

Use the packaged NATS CLI sidecar for operator commands:

```bash
NATS=./libexec/tinkabot/nats # release package root
# NATS=$(cd tools/natscli && go tool -n nats) # source checkout
```

Copy the `nats` URL printed by the running binary into `CLIENT_URL`.
