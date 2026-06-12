---
layer: approach
topic: bundle-v1
references:
  - ./session-v2.md
---

# Bundle v1 Approach

## Purpose

One folder is one app. Pointing the binary at a bundle — a directory (later a
zip) holding a strictly-decoded manifest plus script sources — serves a
complete application for the lifetime of that process: backend scripts wired
to triggers, and a frontend that is nothing but the artifacts those scripts
emit. Setting up an example, demo, or hand-off app becomes a startup flag,
not a code path in the product.

Diagram: https://diashort.apps.quickable.co/d/4ee5ebeb

## Invariants

1. **The loader is an automated author, never a second truth path.** Bundle
   content becomes real exclusively by landing ordinary script records in a
   script store and firing ordinary activations; every effect still passes
   the materializer's gate. The loader never writes projections, artifacts,
   or events directly.
2. **Disjoint authority by construction.** (Amended 2026-06-12, user
   decision.) A bundle declares no authority: script keys, trigger subjects,
   projection ids, and artifact prefixes are all derived under the bundle's
   own namespace (`scripts.bundle.<name>.*`, `tb.bundle.<name>.*`,
   `bundle.<name>.*`, `bundle/<name>/`), so a manifest cannot even spell a
   collision with durable claims, and the bundle's entire NATS reach is one
   wildcard grant. Loading a bundle is the operator's trust act — the
   operator is assumed to know what is inside; remaining load checks are
   namespace hygiene (well-formed names, intra-bundle duplicates), not
   adversarial defense. Load stays all-or-nothing and fail-fast: a binary
   given a bad bundle refuses to start. (Amended again 2026-06-12, user
   decision: the boundary is realized by NATS account isolation — the bundle
   plane lives in its own minted account, the same names in other accounts
   are unrelated, and the only crossing is an explicit service
   export/import; the derived names remain as the app-facing import-remap
   convention.)
3. **The bundle plane is ephemeral by account lifecycle.** (Amended
   2026-06-12.) Bundle account identity is process-ephemeral, so the entire
   bundle plane — records, wiring, materials, artifacts, ledger — is
   unreachable after restart by construction; nothing durable is mutated by
   loading, and a restart without the bundle restores the exact prior
   surface. The app plane's own truth is untouched throughout.
4. **The trust posture is unchanged.** Bundle frontend content is untrusted
   generated material served read-only under sandbox headers; it never holds
   credentials, never registers workers, and reaches backend state only
   through read-only material surfaces or the already-proven command
   acceptance paths. Running a bundle is the operator's trust act, bounded by
   process lifetime, at the same trust level as v1's trusted local script
   processes.
5. **Strict decode, typed attributed failure.** The manifest and every
   derived record are strictly decoded (unknown fields rejected); every
   denial names its owning layer.

## Non-Goals

- Watching the bundle dir or any live-reload semantics; re-authoring through
  the durable script bucket stays the live path.
- Shadowing or override semantics over durable records.
- Session/agent definitions inside bundles (session-v2 owns that subsystem).
- External or TLS exposure of bundle surfaces; loopback posture only.
- Package registries, signing, or distribution beyond a local dir/zip.

## Scope

In scope: the bundle manifest contract, the loader's authority posture, the
ephemeral state discipline, and the read-only reach that lets bundle frontend
content render bundle backend state. Out of scope: everything the Non-Goals
name, and any behavior change to the non-bundle startup path.

## Layer Contract

This Approach constrains thinking only: purpose, invariants, non-goals, and
decision authority. Decomposition, sequencing, and verification strategy
belong to the Plan; file-level execution and proof belong to Tasks. Lower
layers cite this document and never redefine it.

User decisions recorded from the 2026-06-12 brainstorm: ephemeral run posture
(over durable install) and disjoint namespaces (over overlay/shadow or
sole-source). Within those, this Approach owns the invariants above; the Plan
owns slice decomposition and sequencing; Tasks own file-level execution.
Escalate upward when an invariant blocks a slice — never resolve by weakening
the invariant in place.

## Plan-Readiness Gate

Ready when the Plan decomposes into slices that each end in a verifiable
surface (failing test first, then green, with denial oracles output-parsed),
names which existing seams each slice touches, and keeps zip handling as a
pure front-end to the directory path.
