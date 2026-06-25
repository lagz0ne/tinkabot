---
target: ref-bundle-as-app
scope: whole
type: ref
parent: c3-0
title: Bundle As App
---
## Goal

Standardize the one-folder app model that delivers both UI experience and Tinkabot/Tinkalet integration.

## Choice

A bundle is a strict manifest plus local scripts and emitted artifacts. Loading a bundle makes an ephemeral app for that process: scripts wire to NATS-native triggers/material, UI is served as sandboxed artifacts, and examples/package smoke prove the complete path.

## Why

The user wants content illustration apps to be assembled along the line by LLMs/users. A bundle keeps backend transforms and frontend artifacts together while letting the substrate own authority, materialization, and sandboxing.

## How

Required pattern from `examples/clock/bundle.json` and `examples/builder/bundle.json`: one manifest names scripts, commands, projections, boot/every/watches. Required docs are `docs/matched-abstraction/approach/bundle-v1.md`, `docs/matched-abstraction/plan/bundle-v1.md`, and bundle task evidence.
