---
target: c3-5
scope: whole
type: container
parent: c3-0
title: Proof Docs and Workflow
boundary: examples, release gates, manual, matched-abstraction evidence, and agent workflow
---
## Goal

Keep the product explainable and releasable by binding docs, examples, gates, and agent workflow back to the runtime surfaces they claim.

## Components

| ID | Name | Category | Status | Goal Contribution |
| --- | --- | --- | --- | --- |

## Responsibilities

This container owns release evidence, package scripts, manual checks, scenario/fake/coverage gates, examples, matched-abstraction Approach/Plan/Task evidence, `tasks/` handoff state, and installed agent skills that govern future work.

## Complexity Assessment

Docs and proof can drift from code silently if treated as prose. This container is responsible for making drift visible through release evidence, C3 eval bindings, and independent review.
