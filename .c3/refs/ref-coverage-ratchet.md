---
id: ref-coverage-ratchet
c3-seal: 37dcab516214fae550f19633d2342d0e33cd1371a7ec6b02e07485237cf8762a
title: Coverage Ratchet
type: ref
parent: c3-0
goal: Standardize how Tinkabot proves code-to-doc coverage without accepting prose-only or single-LLM claims.
---

## Goal

Standardize how Tinkabot proves code-to-doc coverage without accepting prose-only or single-LLM claims.

## Choice

Use C3 eval specs as mutable fact-to-code bindings, C3 lookup as the file-to-fact map, a Reverse Tornado OKR with 100 percent coverage and zero uncovered owned files, and independent Codex plus Claude noninteractive review before final handoff.

## Why

The user explicitly set anti-goals: no missing piece of code, no workaround, and no single LLM truth. Mechanical lookup/eval plus two independent review surfaces is the smallest durable structure that can make those anti-goals inspectable.

## How

Required pattern: every owned source/doc/proof surface appears under a C3 eval `code:` glob, broad globs are reviewed with `c3 lookup`, and the operating loop in `tasks/c3-line-coverage-okr.md` records objective, anti-goals, metric freshness, flags, and cadence.
