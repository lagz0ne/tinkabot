---
name: be-lazy
description: Use when writing or refactoring code, choosing names, reducing boilerplate, deciding whether explicit types are needed, or reviewing code for human terseness. Prefer compiler-backed inference, short clear names, and less ceremony while keeping verification, public contracts, and safety boundaries explicit.
---

# Be Lazy

Write code like a sharp human who saves strokes for things that matter.

## Rule

Use the shortest code that remains obvious to the next reader and provable by the compiler or tests.

Lazy means:

- lean on inference when the language can prove the shape.
- name things as short as possible while still carrying the idea.
- delete ceremony, wrappers, aliases, suffixes, and annotations that do not buy clarity or safety.
- let local context do work.
- prefer standard language features over custom helpers.

Lazy does not mean:

- skipped tests.
- hidden unsafe edges.
- clever dense code.
- implicit `any` or equivalent unverified dynamic shape.
- vague one-letter names outside tiny local scopes.
- leaving public contracts to guesswork.

## Types

Omit explicit types for local values when inference is exact enough.

Keep explicit types at boundaries:

- exported APIs.
- wire formats and schemas.
- storage records.
- security or auth decisions.
- error contracts.
- overloaded, generic, or higher-order code where inference hides intent.
- tests where the asserted shape is the point.

Prefer deriving types from validators, schemas, literals, or existing APIs over restating them by hand.

## Names

Start short. Expand only to remove real ambiguity.

Prefer:

- `id`, `rev`, `src`, `ctx`, `cfg`, `msg`, `err`, `sub`, `pub`, `req`, `res` when the scope makes them clear.
- domain words over role words.
- verbs for actions and nouns for values.

Avoid redundant suffixes and prefixes:

- `Data`, `Info`, `Object`, `Manager`, `Helper`, `Util`, `Impl`, `Type`, `Interface`, `I`.
- repeated module names inside the same module.
- names that describe the type system instead of the domain.

Use longer names when they prevent collisions between similar domain concepts.

## Shape

Prefer direct code until repetition or complexity earns an abstraction.

Avoid one-call wrappers, pass-through functions, config layers for one option, and types that mirror another type without changing meaning.

Use small scopes, early returns, and local variables when they make inference and reading easier.

## Verification

Compiler and tests are part of being lazy: they do the checking so humans do not have to.

Before finishing, run the narrowest meaningful compiler/test check that proves the reduced code still holds.
