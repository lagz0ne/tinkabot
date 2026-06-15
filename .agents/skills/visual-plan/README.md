# /visual-plan

Turn risky implementation plans into scannable visual review surfaces.

`/visual-plan` turns the plan an agent would normally write in chat into a
human-optimized MDX document. Instead of a long wall of prose, reviewers get
custom components built for understanding: architecture diagrams, wireframes,
interactive prototypes, file maps, annotated code, OpenAPI-style API specs,
visual schema maps, open questions, and comments.

## What It Does

- Grounds plans in real repo files, schemas, actions, and symbols.
- Chooses the right visual surface: document-only, wireframe canvas, prototype,
  design direction, or visual intake.
- Uses MDX and custom components for diagrams, UI flows, API specs, schema maps,
  diffs, code annotations, and reviewer questions.
- Publishes the result as an Agent-Native Plan instead of inline chat Markdown.
- Keeps the plan as the approval gate before source edits begin.

## When To Use It

Use it for multi-file, ambiguous, risky, architecture-heavy, data-heavy, or
UI-heavy work where the wrong direction would be expensive. It is also useful
when a pasted text plan needs a richer review surface.

Skip it for trivial fixes, single-line changes, or anything whose diff is easier
to review than a plan.

## What Reviewers Get

Reviewers get a plan link that is built for scanning. Decisions, files,
diagrams, contracts, UI states, prototype behavior, schema shape, API boundaries,
and unresolved questions live in one consumable place.

The point is not just prettier planning. It is a better medium for human review:
visual where visuals help, structured where structure helps, and grounded in the
actual codebase.

## Install

```sh
npx @agent-native/skills add --skill visual-plan
```

The skill expects the Agent-Native Plan MCP connector to be available when it is
used.
