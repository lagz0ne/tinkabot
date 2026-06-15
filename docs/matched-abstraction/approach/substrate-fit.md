---
layer: approach
topic: substrate-fit
references:
  - ./nats-script-runtime.md
  - ./go-substrate.md
  - ./bundle-v1.md
---

# Substrate Fit Approach

## Purpose

Make the substrate serve real generated frontends end to end by routing each
kind of data through the primitive built for it, instead of forcing one
channel to carry everything. Three capabilities, one principle: large built
assets must not ride the stdout frame channel, bundle build processes must be
isolated, and served artifacts must cache cheaply.

## Invariants

1. **Right primitive for the job.** Small structured truth → KV projections;
   opaque bulk bodies → Object Store; control/effects → framed stdio;
   process isolation → OS namespaces. No primitive is overloaded to cover
   another's job. (Generalizes the NATS-native instinct beyond NATS.)
2. **Scripts stay NATS-blind.** A script never holds credentials or a store
   handle. Bulk output therefore leaves a script by the filesystem, not by
   NATS and not by an oversized stdout frame; the materializer — which holds
   creds and is the gate — performs the privileged Object Store write.
3. **The materializer stays the only gate.** An artifact referenced by path
   is still validated (name/prefix policy, revision) and recorded by the
   materializer before it is truth. Bytes crossing by filesystem do not
   bypass the gate; they bypass only the frame transport.
4. **Isolation is fail-closed.** A bundle process that cannot be sandboxed
   does not run. Build steps that need the network run before the jail is
   sealed; the sandboxed runtime has no network.
5. **Reference resolution is the substrate's.** Lower layers declare local
   intent — short projection ids, relative artifact names, relative frontend
   fetch paths — and the substrate resolves them to the derived global
   namespace at its boundaries: the gate on emit, the scoped server on
   serve. A layer never re-derives or hardcodes a name the substrate already
   controls. (The symmetric completion of derive-by-construction: we derive
   authority from intent, and we hand the resolved reference back.)

## Non-Goals

- Sandboxing the wired app slot or non-bundle scripts (bundles only, this
  pass).
- Multi-node, remote, or rootless-container isolation (bwrap, local, single
  node).
- A general CDN/cache tier; HTTP conditional requests against the digest the
  Object Store already computes, nothing more.
- Changing the projection (KV) effect path — it is already the right tool.

## Scope

In scope: the script process boundary (`LocalScriptRunner`/`FilterLoop`), the
artifact effect contract and how its body is transported and stored, and the
artifact HTTP serving headers. Out of scope: everything in Non-Goals, and any
bundle-v1 surface beyond consuming these (bundle-v1's only delta is the
load-time install step that vendors deps before the jail is sealed).

## Layer Contract

This Approach constrains thinking only. It is a sibling evolution of the Go
substrate and NATS-script-runtime Approaches and cites them; it does not
redefine the bundle-v1 Approach, which consumes the result. Decomposition,
sequencing, and verification belong to the Plan; file work to Tasks.

## Plan-Readiness Gate

Ready when the Plan decomposes into slices that each end in a verifiable
substrate surface, names the seams each touches, keeps the materializer the
sole gate, and proves isolation fail-closed and the filesystem bulk path
without a script ever touching NATS.
