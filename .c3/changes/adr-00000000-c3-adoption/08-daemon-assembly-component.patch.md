---
target: c3-201
scope: whole
type: component
parent: c3-2
title: Daemon Assembly
category: foundation
---
## Goal

Start the product binary posture.

## Parent Fit

| Field | Value |
| --- | --- |
| Parent | c3-2 Go Product Runtime |
| Role | Assemble and expose the single `tinkabot` server binary |
| Owns | `substrate/go/cmd/tinkabot/**`, core Tinkabot assembly tests, manual binary proof |
| Does not own | Tinkalet CLI grammar or SDK package exports |

## Purpose

Daemon Assembly owns the user-facing `tinkabot` executable: flags, startup posture, local profile materialization, role credential printing, shell URL printing, signal-driven drain, and the server-side wiring entry into the Go product runtime.

## Governance

| Reference | Type | Governs | Precedence | Notes |
| --- | --- | --- | --- | --- |
| ref-shadow-authority-boundaries | ref | Binary startup must mint mediated roles instead of handing raw authority to generated code | Runtime authority outranks convenience | The binary prints role creds for operator/user surfaces, not bundle scripts. |

## Contract

| Surface | Direction | Contract | Boundary | Evidence |
| --- | --- | --- | --- | --- |
| CLI flags | IN | `--store` is required; `--shell`, `--bundle`, and `--bundle-sandbox` configure posture. | Operator to daemon | `substrate/go/cmd/tinkabot/main.go` |
| Runtime posture | OUT | Startup prints NATS URL, shell URL, and role credential file paths. | Daemon to operator/manual | `substrate/go/tinkabot/binary_test.go`, `docs/manual/v1.md` |

## Derived Materials

| Material | Must derive from | Allowed variance | Evidence |
| --- | --- | --- | --- |
| Manual start and package examples | Goal, Parent Fit, Contract, and Derived Materials | Path/version examples only | `docs/manual/v1.md`, `README.md` |
