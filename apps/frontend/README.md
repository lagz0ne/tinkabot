# Frontend Lane

Vite trusted shell code belongs here. Generated browser content runs in an opaque sandboxed iframe, talks through a leased message channel, and remains a receiver and intent emitter. It does not own NATS credentials, substrate cookies, service-worker registration authority, or raw NATS access.

`bun run --cwd apps/frontend build` writes the frontend distribution to `substrate/go/frontend/site` so the Go substrate can embed the shell for distribution.
