# Agent Instructions

This file is the shared source of truth for this workspace. `CLAUDE.md` is a symlink to this file; update `AGENTS.md` only.

## Workspace Flow

- At session start, read `tasks/todo.md` and continue from the current handoff.
- For non-trivial work, announce a RED-GREEN-TDD plan before editing.
- Update `tasks/todo.md` as work progresses: current goal, debts, blockers, next steps.
- If a `.c3/` directory exists, use the `c3` skill before architecture, code-change, or exploration work.
- Before presenting non-trivial concepts, architecture options, product direction, or implementation strategy, use `triage-three` more often as a decision-support pass. Make the user's decision easier by stress-testing the idea first, narrowing weak branches, surfacing tradeoffs, and presenting a recommended path with evidence.
- Keep `triage-three` lightweight for focused concepts and deeper for high-stakes architecture/security/release decisions. Skip it for obvious fixes, status reports, direct command execution, or when the user asks to move immediately without analysis.
- When writing or refactoring code, use the `be-lazy` skill: prefer short clear names, compiler-backed inference, direct code, and less ceremony. Keep explicit contracts at public APIs, schemas, storage, auth/security boundaries, error contracts, and places where inference hides intent.
- Prefer proactive execution over clarification loops. Ask only when guessing would create real risk.
- For TypeScript checks, use `bunx @typescript/native-preview` instead of `tsc`.
- For frontend or URL work, use `agent-browser` for smoke tests and visual verification.
- For local dev services, use `zerobased`.

## Karpathy-Inspired Coding Guardrails

Based on the Karpathy Guidelines project: https://github.com/multica-ai/andrej-karpathy-skills

These rules bias toward caution over speed. Use judgment for obvious one-line edits.

### 1. Think Before Coding

- State the interpretation you are using before non-trivial edits.
- Surface assumptions that affect implementation.
- Name meaningful tradeoffs when more than one path is reasonable.
- Push back when a smaller path satisfies the request.
- Ask a concise question when guessing would create real risk.

### 2. Keep It Simple

- Implement the smallest thing that satisfies the current request.
- Do not add unrequested features, configurability, dependencies, or abstractions.
- Do not create an abstraction for one caller.
- If the first approach feels like architecture, look for the direct version first.
- If 200 lines could be 50, simplify before calling it done.
- Prefer inference over explicit declarations when the compiler can prove the shape; add annotation only where it protects a boundary or clarifies a real ambiguity.
- Use the shortest names that keep meaning intact. Avoid redundant suffixes and prefixes unless they distinguish real domain concepts.

### 3. Make Surgical Changes

- Touch only the files needed for the task.
- Match local style even when another style is preferable.
- Do not reformat, rename, reorganize, or refactor adjacent code as a side effect.
- Clean up imports, variables, helpers, or files made unused by your own change.
- Mention unrelated dead code or design problems separately instead of fixing them in the patch.
- Every changed line should trace to the user's request.

### 4. Define The Goal And Verify It

- Bug fix: identify the failing case and expected behavior.
- Feature: identify the observable behavior the user should get.
- Refactor: identify the behavior that must remain unchanged.
- Review: identify concrete risks, regressions, and missing tests.
- Use the narrowest meaningful verification available.
- Do not claim completion until the work is verified.
- If a check cannot be run, say exactly why and what risk remains.

## Response Pattern

For non-trivial coding work, keep the user oriented with:

```text
Assumption:
Changed:
Verified:
Remaining risk:
```

Use this lightly; do not add ceremony to obvious fixes.
