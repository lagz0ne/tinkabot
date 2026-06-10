export const meta = {
  name: 'endgame-slice',
  description: 'Drive one endgame slice through RED-GREEN-TDD with real-NATS, parallel-test, dual-coverage, be-lazy, no-slop, and security gates',
  whenToUse: 'Run per endgame milestone with args {topic: "<slice>"} (e.g. release-spine). args {dryRun: true} stops after the slice contract for a cheap mechanics check.',
  phases: [
    { title: 'Orient', detail: 'read handoff + owning plan, produce slice contract' },
    { title: 'RED', detail: 'author failing proof and verify it fails' },
    { title: 'GREEN', detail: 'smallest complete implementation' },
    { title: 'Gates', detail: 'six parallel quality gates over the diff' },
    { title: 'Fix', detail: 'resolve blockers, re-run failed gates' },
    { title: 'Verify', detail: 'full verification suite' },
    { title: 'Wrap-up', detail: 'task doc evidence + todo.md handoff' },
  ],
}

const a = typeof args === 'string'
  ? (args.trim().startsWith('{') ? JSON.parse(args) : { topic: args.trim() })
  : (args ?? {})
const topic = a.topic
const dryRun = !!a.dryRun
if (!topic) throw new Error(`args.topic required, e.g. {topic: "release-spine"} — got: ${JSON.stringify(args)}`)

const CONTRACT = {
  type: 'object',
  required: ['topic', 'goal', 'planCitations', 'redDefinition', 'targetedCommands', 'fullVerifyCommands', 'scopeGuards', 'nonGoals'],
  properties: {
    topic: { type: 'string' },
    goal: { type: 'string', description: 'observable outcome of the slice' },
    planCitations: { type: 'array', items: { type: 'string' }, description: 'doc:line citations of the Plan handoff this slice consumes' },
    redDefinition: { type: 'string', description: 'what the failing artifact is and what failure proves' },
    targetedCommands: { type: 'array', items: { type: 'string' }, description: 'narrow commands that own this slice RED/GREEN' },
    fullVerifyCommands: { type: 'array', items: { type: 'string' }, description: 'the full verification suite from tasks/todo.md' },
    scopeGuards: { type: 'array', items: { type: 'string' }, description: 'claims this slice must NOT make' },
    nonGoals: { type: 'array', items: { type: 'string' } },
  },
}

const EVIDENCE = {
  type: 'object',
  required: ['commands', 'summary', 'ok'],
  properties: {
    commands: {
      type: 'array',
      items: {
        type: 'object',
        required: ['cmd', 'result'],
        properties: { cmd: { type: 'string' }, result: { type: 'string', description: 'concrete result line: pass/fail counts or key output' } },
      },
    },
    summary: { type: 'string' },
    ok: { type: 'boolean', description: 'phase succeeded (for RED: the artifact fails as required)' },
  },
}

const VERDICT = {
  type: 'object',
  required: ['gate', 'pass', 'blockers'],
  properties: {
    gate: { type: 'string' },
    pass: { type: 'boolean' },
    blockers: {
      type: 'array',
      items: {
        type: 'object',
        required: ['title', 'detail', 'fix'],
        properties: {
          title: { type: 'string' },
          detail: { type: 'string', description: 'file:line evidence' },
          fix: { type: 'string', description: 'smallest concrete fix' },
        },
      },
    },
  },
}

const rules = `Hard rules for this repo:
- Tests use REAL embedded NATS (substrate/go/embednats runtime, JetStream KV/Object/streams). In-memory fakes only for narrow impossible-to-force branches, each justified in a code comment.
- New Go tests must be parallel-safe: isolated servers/ports/buckets per test, t.Parallel() where state allows.
- Code style follows .codex/skills/be-lazy/SKILL.md: inference over annotation locally, explicit types only at exported APIs, wire formats, storage, auth/security, error contracts. Short names, no ceremony, no one-call wrappers.
- TypeScript checks via bunx @typescript/native-preview, never tsc.
- NATS is the system seam: outside-in proof goes through real NATS-visible surfaces (request/reply, KV, Object Store, streams, statuses), nats CLI where the caller is external.
- Working directory: /home/lagz0ne/dev/tinkabot. Go commands run from substrate/go.`

phase('Orient')
log(`Slice: ${topic}`)
const contract = await agent(
  `Produce the slice contract for endgame slice "${topic}" in /home/lagz0ne/dev/tinkabot. Read tasks/todo.md (Next Slice, Pinned Decisions, Current Verification Commands), the owning Plan doc(s) under docs/matched-abstraction/plan/ for this topic (for release-spine that is endgame-app.md section "Release-Spine Decomposition"), and any peer Task docs it cites. Do NOT edit anything. Return the contract exactly per schema: every planCitation must be a real doc path with line numbers you verified; targetedCommands are the narrow RED/GREEN commands; fullVerifyCommands come from todo.md; scopeGuards are claims the slice must not make (for release-spine: HA/scale contract-only, managed-auth compile-level, schedule no live tick source, CLI denial output-parsed oracle). ${rules}`,
  { label: `orient:${topic}`, phase: 'Orient', schema: CONTRACT },
)
if (!contract) throw new Error('orient agent returned nothing')
log(`Contract: ${contract.goal}`)

if (dryRun) return { dryRun: true, contract }

phase('RED')
const red = await agent(
  `Execute the RED step for endgame slice "${topic}". Contract: ${JSON.stringify(contract)}.
Write the failing artifact described in redDefinition (failing tests or failing checker — for doc/evidence slices the checker IS the test). Do NOT implement the solution. Run the targeted commands and capture concrete failure output. ok=true means the artifact exists, runs, and FAILS for the contracted reason (not a syntax error). Follow matched-abstraction Task-layer discipline: also draft docs/matched-abstraction/task/${topic}.md with frontmatter (layer: task, topic: ${topic}, status: active, references to the owning plan/approach docs), brief, acceptance contract, and the executed RED citation with real command + failure output. Run bun run validate:layers to keep the doc valid. ${rules}`,
  { label: `red:${topic}`, phase: 'RED', schema: EVIDENCE },
)
if (!red?.ok) return { failed: 'RED', contract, red }
log(`RED proven: ${red.summary}`)

phase('GREEN')
const green = await agent(
  `Execute the GREEN step for endgame slice "${topic}". Contract: ${JSON.stringify(contract)}. RED evidence: ${JSON.stringify(red)}.
Implement the smallest complete change that turns the RED artifact green at its boundary, including denial/failure paths. No new features beyond the contract, no scope expansion, respect nonGoals and scopeGuards. Re-run the targeted commands until they pass; capture concrete results. Update the execution-notes section of docs/matched-abstraction/task/${topic}.md as you go. ${rules}`,
  { label: `green:${topic}`, phase: 'GREEN', schema: EVIDENCE },
)
if (!green?.ok) return { failed: 'GREEN', contract, red, green }
log(`GREEN: ${green.summary}`)

const GATES = [
  {
    key: 'real-nats',
    prompt: `Gate: real embedded NATS only. Review the working-tree diff (git diff; git status --porcelain) in /home/lagz0ne/dev/tinkabot. Every new or changed test that touches substrate behavior must run over the real embedded NATS runtime (embednats) with real JetStream KV/Object/stream state. Flag any new mock/fake/stub/in-memory store usage in tests unless it forces a narrow branch impossible over real NATS AND carries a justifying comment. Flag tests asserting against fakes where an embednats equivalent exists.`,
  },
  {
    key: 'parallel-safety',
    prompt: `Gate: parallel test execution. Review new/changed Go tests in the working-tree diff. Each test must be runnable in parallel: no shared global servers, fixed ports, shared bucket/stream names, or order dependence; prefer t.Parallel() with per-test isolated embedded servers and unique store names. Also flag TS tests with shared mutable module state. Blockers must name the exact test and the shared resource.`,
  },
  {
    key: 'coverage',
    prompt: `Gate: dual coverage. Read the slice contract: ${'<<CONTRACT>>'} and the working-tree diff. Inside-out: every typed failure/error family the slice declares must have one owning test (name the missing ones). Outside-in: check the capability proof matrix cases that apply (allowed, denied-neighbor, malformed, duplicate, stale revision, revoked lease, loop suppression, attributed failure) are each proven over a NATS-visible surface or explicitly N/A with a reason in the task doc. List covered vs missing explicitly.`,
  },
  {
    key: 'be-lazy',
    prompt: `Gate: be-lazy style. Read .codex/skills/be-lazy/SKILL.md, then review ONLY the working-tree diff. Flag: explicit type annotations where inference is exact, redundant suffixes/prefixes (Data, Info, Manager, Helper, Impl), one-call wrappers, pass-through functions, config layers for one option, types mirroring other types, long names where short carries the idea, custom helpers duplicating stdlib. Do NOT flag explicit types at exported APIs, wire/schema/storage shapes, auth decisions, or error contracts — those must stay explicit.`,
  },
  {
    key: 'no-slop',
    prompt: `Gate: no-slop. Review the working-tree diff for AI slop: comments narrating the next line or justifying the change, dead code, unused imports/helpers, placeholder text, defensive checks for impossible states, duplicated logic, over-general abstractions for one caller, README/doc boilerplate that restates code. Every changed line must trace to the slice contract.`,
  },
  {
    key: 'security',
    prompt: `Gate: security/authority. Review the working-tree diff against the repo's authority model (tasks/todo.md Pinned Decisions): no raw NATS authority for scripts or generated content, credentials are scoped leases not ambient, deny wins over allow, subjects concrete (no placeholder/broad wildcards without authoritative prefix), no secrets/credential material in logs/artifacts/test fixtures, no widened permissions, attribution preserved on failures. Flag any authority widening even if tests pass.`,
  },
]

const runGate = (g) =>
  agent(`${g.prompt.replace('<<CONTRACT>>', JSON.stringify(contract))}\nReturn gate="${g.key}". pass=true only with zero blockers. Be adversarial; report only real findings with file:line evidence, not style nitpicks outside this gate's charter. Repo: /home/lagz0ne/dev/tinkabot. Read-only: do NOT edit files.`,
    { label: `gate:${g.key}`, phase: 'Gates', schema: VERDICT })

phase('Gates')
let verdicts = (await parallel(GATES.map((g) => () => runGate(g)))).filter(Boolean)
let failed = verdicts.filter((v) => !v.pass)
log(`Gates round 1: ${verdicts.length - failed.length}/${verdicts.length} pass`)

phase('Fix')
let round = 0
while (failed.length && round < 3) {
  round++
  const blockers = failed.flatMap((v) => v.blockers.map((b) => ({ gate: v.gate, ...b })))
  log(`Fix round ${round}: ${blockers.length} blockers from [${failed.map((v) => v.gate).join(', ')}]`)
  const fix = await agent(
    `Fix these gate blockers in /home/lagz0ne/dev/tinkabot for slice "${topic}": ${JSON.stringify(blockers)}.
Apply the smallest fix per blocker, keep the contract intact (${JSON.stringify(contract.scopeGuards)}), then re-run the slice's targeted commands (${JSON.stringify(contract.targetedCommands)}) to prove nothing regressed. If a blocker is a false positive, do not change code — say why in the summary with file:line evidence. ${rules}`,
    { label: `fix:round${round}`, phase: 'Fix', schema: EVIDENCE },
  )
  if (!fix?.ok) return { failed: 'Fix', contract, red, green, verdicts, fix }
  const rerun = (await parallel(
    failed.map((v) => GATES.find((g) => g.key === v.gate)).filter(Boolean).map((g) => () => runGate(g)),
  )).filter(Boolean)
  verdicts = verdicts.filter((v) => v.pass).concat(rerun)
  failed = rerun.filter((v) => !v.pass)
}
if (failed.length) return { failed: 'Gates', contract, red, green, verdicts }
log('All gates pass')

phase('Verify')
const verify = await agent(
  `Run the FULL verification suite for /home/lagz0ne/dev/tinkabot and report each command with its concrete result (pass/fail counts, ok lines): ${JSON.stringify(contract.fullVerifyCommands)}. Also run git diff --check. Do not fix anything beyond trivial formatting the suite itself flags; ok=false if anything fails. ${rules}`,
  { label: 'verify:full', phase: 'Verify', schema: EVIDENCE },
)
if (!verify?.ok) return { failed: 'Verify', contract, red, green, verdicts, verify }
log(`Full verification: ${verify.summary}`)

phase('Wrap-up')
const wrapup = await agent(
  `Wrap up endgame slice "${topic}" in /home/lagz0ne/dev/tinkabot.
1. Finish docs/matched-abstraction/task/${topic}.md: status: complete, verification-evidence section citing each command with its concrete result (RED: ${JSON.stringify(red.commands)}; GREEN: ${JSON.stringify(green.commands)}; full: ${JSON.stringify(verify.commands)}), gate results (${JSON.stringify(verdicts.map((v) => ({ gate: v.gate, pass: v.pass })))}), and a DECLARATIVE wrap-up announcement (state what IS complete — never "when complete...").
2. Update tasks/todo.md: mark the milestone DONE, add only evidence that changes the current handoff, set the next resume point.
3. Run bun run validate:layers and bun run test:layers; both must pass.
Do not commit. Return summary of what changed. ${rules}`,
  { label: `wrapup:${topic}`, phase: 'Wrap-up', schema: EVIDENCE },
)

return {
  topic,
  contract,
  red: red.summary,
  green: green.summary,
  gates: verdicts.map((v) => ({ gate: v.gate, pass: v.pass })),
  fixRounds: round,
  verify: verify.summary,
  wrapup: wrapup?.summary ?? 'wrap-up agent returned nothing',
}
