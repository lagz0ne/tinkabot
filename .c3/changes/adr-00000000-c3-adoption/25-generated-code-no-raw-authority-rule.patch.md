---
target: rule-generated-code-no-raw-authority
scope: whole
type: rule
parent: c3-0
title: Generated Code No Raw Authority
---
## Goal

Enforce that generated browser content and generated backend scripts do not receive raw NATS authority, credentials, permission material, or store handles.

## Rule

Generated code must use mediated Tinkabot surfaces, never raw NATS authority.

## Golden Example

Literal code from `apps/frontend/src/isolation.ts`:

```ts
const raw = new Set([
  "allow", // REQUIRED: raw permission vocabulary is denied.
  "credential", // REQUIRED: generated content cannot smuggle credentials.
  "nats", // REQUIRED: raw NATS authority is not a frame message field.
  "permission", // REQUIRED: permission material stays outside generated content.
  "publish", // REQUIRED: publishing is mediated by accepted command intent.
  "subject", // REQUIRED: raw subject selection is not delegated to content.
  "subscribe", // REQUIRED: subscription authority is shell/substrate-owned.
  "token", // REQUIRED: tokens never cross into generated frame messages.
]);
```

## Not This

| Anti-Pattern | Correct | Why Wrong Here |
| --- | --- | --- |
| Generated iframe message includes `subject` or `token`. | Send a typed content intent that the trusted shell validates. | It bypasses lease, revision, and capability checks. |
| Bundle manifest declares publish/subscribe permissions. | Derive bundle subjects/projections/artifacts from manifest names. | It makes generated app content an authority source. |

## Scope

Applies to generated browser content, bundle scripts, local reactions, and any future LLM-built app integration surface. It does not forbid operator diagnostics with role credentials or Tinkalet profile-managed credentials.

## Override

Only a higher-layer Approach change may introduce a raw-authority path, and it must include a denial matrix, credential lifecycle, and release evidence before implementation.
