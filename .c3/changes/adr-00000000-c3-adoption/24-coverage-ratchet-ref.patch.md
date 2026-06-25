---
target: ref-coverage-ratchet
scope: whole
type: ref
parent: c3-0
title: Coverage Ratchet
---
## Goal

Standardize how Tinkabot proves code-to-doc coverage without accepting prose-only or single-LLM claims.

## Choice

Use C3 eval specs as mutable fact-to-code bindings, C3 lookup as the file-to-fact map, a Reverse Tornado OKR with 100 percent coverage and zero uncovered owned files, and independent Codex plus Claude noninteractive review before final handoff.

## Why

The user explicitly set anti-goals: no missing piece of code, no workaround, and no single LLM truth. Mechanical lookup/eval plus two independent review surfaces is the smallest durable structure that can make those anti-goals inspectable.

## How

Required pattern: every owned source/doc/proof surface appears under a C3 eval `code:` glob, broad globs are reviewed with `c3 lookup`, and the operating loop in `tasks/c3-line-coverage-okr.md` records objective, anti-goals, metric freshness, flags, and cadence.
