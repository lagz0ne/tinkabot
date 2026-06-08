---
layer: approach
topic: code-structure
references:
  - ./platform-structure.md
  - ./nats-script-runtime.md
  - ./browser-frontend-mediator.md
---

# Code Structure Approach

Structure diagram: https://diashort.apps.quickable.co/d/b98307da

## Scope

The repository structure must express the platform reset. Root is orchestration and handoff only. Product code belongs to owned lanes: Go substrate, Vite frontend, SDK/schema, and retained legacy evidence.

The structure must make dependency direction visible. Schema and SDK contracts feed substrate and frontend. Substrate and frontend consume those contracts; they do not redefine them. Generated artifacts follow canonical schemas and validation contracts.

## Layer Contract

Approach owns authority boundaries: root is not product-code authority, Go owns substrate authority, Vite owns trusted frontend shell authority, and SDK/schema owns shared contract shape.

Plan owns lane decomposition, sequencing, and verification strategy.

Task owns one bounded move that preserves existing behavior while removing misleading root-level product-code ownership.

## Decision Hierarchy

Platform authority beats root convenience.

Boundary clarity beats minimum file churn when the existing layout misrepresents ownership.

Neutral schema authority beats language-local types. Runtime validation beats type-only confidence. Generated Zod, Go, TypeScript, and fixture artifacts derive from the schema lane and do not become hand-maintained authority.

Existing Bun and `@lagz0ne/nats-embedded` work remains migration input and regression evidence. It does not define current substrate ownership.

Root remains thin coordination: workspace metadata, scripts, docs, task handoff, and cross-lane verification.

## Plan-Readiness Gate

Plan can proceed when it classifies current root content by owner, states allowed dependency directions, preserves observable behavior, and distinguishes active SDK/browser contract code from legacy substrate proof material.

Plan must return to Approach if TypeScript remains substrate authority, Go or Vite define schema independently, generated artifacts become hand-maintained source, or root keeps product-code ownership as the default.
