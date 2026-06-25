---
id: c3-0
c3-seal: 780bd9bd3e011b143dbceef155e4b6db2c5a2af8b28a356115648e94e0bbd5c0
title: tinkabot
goal: Tinkabot is a NATS-native app substrate for generated chain-reaction applications. A user or agent registers a bundle through Tinkabot; Tinkabot supplies the server authority, embedded NATS, durable coordination state, sandboxed script execution, and trusted shell; Tinkalet supplies profile-aware edge participation and local transforms. UI and backend behavior are reactive to NATS material, especially KV-backed projections and item records, without generated code receiving raw credentials or store handles.
---

## Goal

Tinkabot is a NATS-native app substrate for generated chain-reaction applications. A user or agent registers a bundle through Tinkabot; Tinkabot supplies the server authority, embedded NATS, durable coordination state, sandboxed script execution, and trusted shell; Tinkalet supplies profile-aware edge participation and local transforms. UI and backend behavior are reactive to NATS material, especially KV-backed projections and item records, without generated code receiving raw credentials or store handles.

## Containers

| ID | Name | Boundary | Status | Responsibilities | Goal Contribution |
| --- | --- | --- | --- | --- | --- |
| c3-1 | Contracts and SDK |  | active | Own the neutral contract surface that Go, TypeScript, browser, script, and release proofs derive from. | Own the neutral contract surface that Go, TypeScript, browser, script, and release proofs derive from. |
| c3-2 | Go Product Runtime |  | active | Run Tinkabot as the server authority that turns NATS-native activations, scripts, materials, bundles, and shell serving into one product posture. | Run Tinkabot as the server authority that turns NATS-native activations, scripts, materials, bundles, and shell serving into one product posture. |
| c3-3 | Tinkalet Edge |  | active | Let humans, scripts, CI jobs, and local agents participate in Tinkabot without speaking raw NATS by default. | Let humans, scripts, CI jobs, and local agents participate in Tinkabot without speaking raw NATS by default. |
| c3-4 | Browser Shell |  | active | Provide the trusted browser-side substrate that renders generated artifacts, observes materialized state, and mediates browser intents without exposing raw NATS authority to generated content. | Provide the trusted browser-side substrate that renders generated artifacts, observes materialized state, and mediates browser intents without exposing raw NATS authority to generated content. |
| c3-5 | Proof Docs and Workflow |  | active | Keep the product explainable and releasable by binding docs, examples, gates, and agent workflow back to the runtime surfaces they claim. | Keep the product explainable and releasable by binding docs, examples, gates, and agent workflow back to the runtime surfaces they claim. |

## Abstract Constraints

| Constraint | Rationale | Affected Containers |
| --- | --- | --- |
| NATS material is the integration seam. | Chain reactions must be observable through subjects, KV/Object Store records, streams, schedules, and projections rather than hidden in local function calls. | all |
| Generated scripts and generated UI inherit mediated authority only. | Bundle code should act like an app without explicitly handling NATS credentials, raw subjects, stores, or host secrets. | c3-2, c3-3, c3-4 |
| Documentation coverage is measured by C3 eval bindings and independent review. | The anti-goal is no missing code and no single-LLM truth, so code-to-doc coverage must be mechanical and reviewable. | all |
