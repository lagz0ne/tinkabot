---
id: adr-00000000-c3-adoption
c3-seal: 87a0a4b1639e6174b5ab18c959241ea1288215505c8db695abd06e621ca44f0e
title: C3 Architecture Documentation Adoption
type: adr
goal: Adopt C3 as the frozen architecture truth for Tinkabot so every product/runtime surface can be traced from code and docs back to a named component, reference, or rule.
status: proposed
affects:
    - c3-0
---

## Goal

Adopt C3 as the frozen architecture truth for Tinkabot so every product/runtime surface can be traced from code and docs back to a named component, reference, or rule.

## Context

The repository already has substantial matched-abstraction Approach, Plan, and Task evidence, plus Go, TypeScript, shell, schema, frontend, example, release, and agent-workflow code. It did not have a C3 model, so there was no CLI-owned topology or eval binding that could answer whether a code file was documented. The current product direction emphasizes NATS-native chain reactions, a reactive UI over NATS material, Tinkalet-registered transforms, sandboxed script execution, bundle delivery of UI plus integration, and shadow authority inheritance instead of generated code handling permissions directly.

## Decision

Create the first C3 rung with five containers: contracts/SDK, Go product runtime, Tinkalet edge, browser shell, and proof/docs/workflow. Create component facts that bind to concrete repo surfaces, plus refs and rules for NATS-native chain reaction, shadow authority, bundles as apps, matched-abstraction evidence, code-coverage ratcheting, raw-authority denial, and sandbox fail-closed behavior. After the flip, bind each fact to code/docs through C3 eval specs and run C3 check, eval, lookup coverage, and independent Codex/Claude noninteractive review.

## Affected Topology

| Entity | Type | Why affected | Evidence | Governance review |
| --- | --- | --- | --- | --- |
| c3-0 | system | The system body was authored and all first-rung containers were created below it. | c3-0#n526@v1:sha256:e48680873b15f5fb66893cf025d46759b00da5544d4599263a865531f9449986 | Verify synthesized membership, system constraints, and eval coverage after apply. |
| first-rung create-patches | N.A - genesis unit | New containers, components, refs, and rules were staged as create-patches in this ADR. | N.A - patch folder was applied as the genesis unit | Verify c3 change view, c3 change apply, c3 check, c3 eval, and lookup coverage. |

## Compliance Refs

| Ref | Why required | Evidence | Action |
| --- | --- | --- | --- |
| ref-bundle-as-app | Components c3-203 and c3-503 use the bundle-as-app model for complete UI plus backend integration. | ref-bundle-as-app#n470@v1:sha256:b5e14bb81a3386939ae1567e0afe7a2be56a944a97f3eb030267a19af35c0c83 | create-ref |
| ref-coverage-ratchet | Components c3-501, c3-502, and c3-504 use the mechanical coverage ratchet to prevent undocumented code. | ref-coverage-ratchet#n486@v1:sha256:d50757c1034f6c7f32d0957295ade8051dc3e94a4fb4f4c35819a50776dfcd81 | create-ref |
| ref-matched-abstraction-layers | Components c3-102, c3-501, c3-502, and c3-504 rely on the layer authority split. | ref-matched-abstraction-layers#n478@v1:sha256:c3c6414210c4b434c974e5601e24903fd2692d42603a622d33d66249b475db8c | create-ref |
| ref-nats-native-chain-reaction | Components c3-101, c3-202, c3-203, c3-302, and c3-503 rely on NATS material as the reaction spine. | ref-nats-native-chain-reaction#n453@v1:sha256:bcf39fdbf725aba45582ded1f660e8565056dafbc3ed4e4544bae9106a1a06a7 | create-ref |
| ref-shadow-authority-boundaries | Components c3-201, c3-203, c3-204, c3-301, c3-302, c3-401, and c3-402 rely on mediated authority instead of generated raw access. | ref-shadow-authority-boundaries#n462@v1:sha256:7945c18d1efc8027bf9b8f5ce82202d6cd9fd6aa5654492934cc57b635cf0693 | create-ref |

## Compliance Rules

| Rule | Why required | Evidence | Action |
| --- | --- | --- | --- |
| rule-bundle-sandbox-default-fail-closed | Components c3-202 and c3-203 depend on bundle processes failing closed unless the operator explicitly opts out. | rule-bundle-sandbox-default-fail-closed#n512@v1:sha256:5a7c98bb43285c8edc63d45c8e252a7f7ba1b8b4b2931feb3985db6fd6d5cf64 | create-rule |
| rule-generated-code-no-raw-authority | Component c3-401 depends on rejecting raw NATS authority vocabulary from generated browser content. | rule-generated-code-no-raw-authority#n496@v1:sha256:f4734ac2ff2bebe23ace5ef434126d0b05218a186ebc13a6d82943bc66e913d4 | create-rule |

## Work Breakdown

| Area | Detail | Evidence |
| --- | --- | --- |
| System body | Author c3-0 goal, containers table header, and abstract constraints. | c3 read c3-0 --full |
| Genesis facts | Create container/component/ref/rule patches under the genesis ADR. | c3 change view adr-00000000-c3-adoption |
| Eval bindings | Bind component/ref/rule facts to code/docs globs in .c3/eval. | c3 eval, c3 lookup <file> |
| OKR coverage loop | Persist the Reverse Tornado objective, anti-goals, CKRs/DKRs/PKRs, flags, and operating loop for line-to-doc coverage. | tasks/c3-line-coverage-okr.md |
| Independent review | Run Codex and Claude noninteractive YOLO review over the coverage/model evidence. | /tmp/tinkabot-c3-codex-review.txt, /tmp/tinkabot-c3-claude-review.txt |

## Enforcement Surfaces

| Surface | Behavior | Evidence |
| --- | --- | --- |
| C3 check | Validates the first-rung facts against their canvas and membership model. | c3 check --include-adr |
| C3 eval | Verifies every declared fact-to-code binding resolves and selected claims hold. | c3 eval |
| C3 lookup | Maps concrete files back to owning facts via eval code globs. | c3 lookup substrate/go/tinkabot/bundle.go |
| Coverage OKR | Names 100% file-to-fact coverage and 0 uncovered owned source files as the objective metric. | tasks/c3-line-coverage-okr.md |
| Dual LLM review | Prevents single-model truth by requiring Codex and Claude noninteractive checks before final handoff. | review artifacts in /tmp |

## Alternatives Considered

| Alternative | Rejected because |
| --- | --- |
| Rely only on existing matched-abstraction docs. | Those docs are useful layer authority, but they do not provide C3 lookup/eval bindings for every code surface. |
| Create a massive one-fact-per-file C3 model. | It would look complete while making the architecture harder to maintain; eval bindings can measure exact file coverage without exploding the first-rung concept graph. |
| Accept a prose-only coverage claim. | The user's anti-goal explicitly rejects missing code, workarounds, and single-LLM truth, so coverage must be mechanical and independently reviewed. |

## Risks

| Risk | Mitigation | Verification |
| --- | --- | --- |
| A coarse component fact hides an unowned file. | Eval specs and lookup coverage require every owned source/doc/proof path to map back to a fact. | c3 lookup --json '<glob>', coverage script/check output |
| A C3 claim drifts from code after future edits. | Treat c3 eval as a recurring direct read and keep eval specs mutable when code moves. | c3 eval |
| Runtime smoke stays blocked by embedded NATS readiness in this environment. | Separate docs/model correctness from runtime startup evidence; do not claim live runtime smoke if it remains blocked. | Record any blocked runtime check explicitly in tasks/todo.md. |
| Independent review CLIs cannot both produce model verdicts. | Treat the dual-review anti-goal as unsatisfied until both Codex and Claude artifacts exist. | tasks/c3-line-coverage-okr.md and tasks/todo.md blocker notes. |

## Verification

| Check | Result |
| --- | --- |
| c3 change view adr-00000000-c3-adoption | Passed: 26 create-patches staged as new. |
| c3 change apply adr-00000000-c3-adoption | Passed after tightening component Derived Materials grounding. |
| c3 eval | Passed: 21 holds, 0 drift, 0 needs-judgement. |
| owned-file c3 lookup loop | Passed: scripts/c3-line-coverage-harness.sh verifies 469 current tracked/non-ignored owned files, excluding only dependency/build/generated/cache classes and .c3/c3.db, with 0 lookup errors and 0 uncovered files. |
| c3 lookup substrate/go/tinkabot/bundle.go | Passed: maps to c3-203 plus bundle/NATS/shadow refs and sandbox rule. |
| c3 check --include-adr | Passed after blocker-note update: issues[0], ok true. |
| codex exec --dangerously-bypass-approvals-and-sandbox ... | Passed on final rerun: Codex verified the durable harness reports owned_files=469, lookup_errors=0, uncovered=0; independently replayed all 469 lookups with 0 command errors and 0 matches[0]; found only allowed .c3/c3.db outside the owned set; and returned VERDICT: PASS, FINDINGS: none, GAPS: none. |
| claude -p --dangerously-skip-permissions ... | Passed on final rerun: Claude verified the durable harness reports owned_files=469, lookup_errors=0, uncovered=0; independently replayed all 469 lookups; reconciled 475 git-visible paths to 469 current owned paths plus 5 deleted tracked paths and allowed .c3/c3.db; confirmed no-match lookups exit 0 and are correctly detected through matches[0]; and returned VERDICT: PASS, GAPS: none. |
