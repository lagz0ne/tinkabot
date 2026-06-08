#!/usr/bin/env bun

import { createHash } from "node:crypto";
import { mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { dirname, join } from "node:path";
import { spawn } from "node:child_process";

export type Status = "DONE" | "NEXT" | "TODO";
export type Sandbox = "read-only" | "workspace-write";
export type Role = "approach" | "tests" | "risk" | "worker" | "review" | "fix";

export interface Milestone {
  n: number;
  status: Status;
  topic: string;
  desc: string;
  line: string;
}

export interface Agent {
  role: Role;
  cwd: string;
  sandbox: Sandbox;
  out: string;
  log: string;
  prompt: string;
  lease: string[];
}

export interface Round {
  topic: string;
  scouts: Agent[];
  worker: Agent;
  review: Agent;
  verify: string[];
}

export interface Args {
  root: string;
  run?: string;
  jobs: number;
  maxRounds: number;
  maxFix: number;
  timeoutMin: number;
  allowDirty: boolean;
  dryRun: boolean;
  keep: boolean;
  model?: string;
  codex: string;
}

export const verify = [
  "bun run schema:parity",
  "bun run typecheck",
  "bun run test",
  "bun run build",
  "bun run pack:dry",
  "bun run validate:layers",
  "bun run test:layers",
  "git diff --check",
];

export function parseMilestones(md: string): Milestone[] {
  const lines = md.split(/\r?\n/);
  return lines.flatMap((line) => {
    const match = line.match(/^\s*(\d+)\.\s+(?:(DONE|NEXT):\s+)?(.+)$/);
    if (!match) return [];

    const topic = match[3]?.match(/`([^`]+)`/)?.[1];
    if (!topic) return [];

    return [
      {
        n: Number(match[1]),
        status: (match[2] ?? "TODO") as Status,
        topic,
        desc: match[3].replace(/`[^`]+`:\s*/, "").trim(),
        line,
      },
    ];
  });
}

export function nextMilestone(ms: readonly Milestone[]): Milestone | undefined {
  return ms.find((m) => m.status === "NEXT") ?? ms.find((m) => m.status !== "DONE");
}

export function endgameDone(ms: readonly Milestone[]): boolean {
  return ms.length > 0 && ms.every((m) => m.status === "DONE");
}

export function buildCodexArgs(opts: {
  cwd: string;
  out: string;
  sandbox: Sandbox;
  json?: boolean;
  model?: string;
}): string[] {
  const args = ["--ask-for-approval", "never"];
  if (opts.model) args.push("--model", opts.model);
  args.push(
    "exec",
    "--cd",
    opts.cwd,
    "--sandbox",
    opts.sandbox,
    "--output-last-message",
    opts.out,
    "--color",
    "never",
    "--ephemeral",
  );
  if (opts.json) args.push("--json");
  args.push("-");
  return args;
}

export function planRound(m: Milestone, cfg: Pick<Args, "root" | "run" | "jobs">): Round {
  const run = cfg.run ?? join(cfg.root, ".codex-runs", "endgame", "dry-run");
  const worker = join(run, "worktrees", `${m.n}-${m.topic}-worker`);
  const logs = join(run, "logs");
  const lease = [
    "apps/**",
    "docs/matched-abstraction/**",
    "package.json",
    "bun.lock",
    "packages/**",
    "schemas/**",
    "scripts/**",
    "substrate/**",
    "tasks/todo.md",
    "tests/**",
  ];

  const mk = (role: Role, cwd: string, sandbox: Sandbox): Agent => ({
    role,
    cwd,
    sandbox,
    out: join(logs, `${role}.last.md`),
    log: join(logs, `${role}.jsonl`),
    prompt: prompt(role, m),
    lease: role === "worker" || role === "fix" ? lease : [],
  });

  return {
    topic: m.topic,
    scouts: [
      mk("approach", cfg.root, "read-only"),
      mk("tests", cfg.root, "read-only"),
      mk("risk", cfg.root, "read-only"),
    ],
    worker: mk("worker", worker, "workspace-write"),
    review: mk("review", cfg.root, "read-only"),
    verify,
  };
}

export function parseArgs(argv = process.argv.slice(2)): Args {
  const root = process.cwd();
  const args: Args = {
    root,
    jobs: num(process.env.CODEX_ORCH_JOBS, 3),
    maxRounds: num(process.env.CODEX_ORCH_MAX_ROUNDS, 12),
    maxFix: num(process.env.CODEX_ORCH_MAX_FIX, 2),
    timeoutMin: num(process.env.CODEX_ORCH_TIMEOUT_MIN, 90),
    allowDirty: false,
    dryRun: false,
    keep: true,
    codex: process.env.CODEX_BIN ?? "codex",
    model: process.env.CODEX_ORCH_MODEL,
  };

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === "--allow-dirty") args.allowDirty = true;
    else if (arg === "--dry-run") args.dryRun = true;
    else if (arg === "--no-keep") args.keep = false;
    else if (arg === "--root") args.root = need(argv[++i], arg);
    else if (arg === "--run-dir") args.run = need(argv[++i], arg);
    else if (arg === "--jobs") args.jobs = Number(need(argv[++i], arg));
    else if (arg === "--max-rounds") args.maxRounds = Number(need(argv[++i], arg));
    else if (arg === "--max-fix") args.maxFix = Number(need(argv[++i], arg));
    else if (arg === "--timeout-min") args.timeoutMin = Number(need(argv[++i], arg));
    else if (arg === "--model") args.model = need(argv[++i], arg);
    else if (arg === "--codex") args.codex = need(argv[++i], arg);
    else if (arg === "-h" || arg === "--help") {
      usage();
      process.exit(0);
    } else {
      throw new Error(`unknown arg: ${arg}`);
    }
  }

  return args;
}

export async function main(args = parseArgs()): Promise<void> {
  const todo = await readFile(join(args.root, "tasks", "todo.md"), "utf8");
  const ms = parseMilestones(todo);
  const id = runId();
  const run = args.run ?? join(args.root, ".codex-runs", "endgame", id);
  const lock = join(args.root, ".codex-runs", "endgame.lock");

  if (endgameDone(ms)) {
    await runVerify(args.root, verify, args);
    console.log("Endgame already DONE; verification passed.");
    return;
  }

  const dirty = (await sh(args.root, "git status --porcelain")).stdout.trim();
  if (!args.allowDirty && dirty) {
    throw new Error("worktree is dirty; commit/stash first or rerun with --allow-dirty");
  }
  if (args.allowDirty && dirty && !args.dryRun) {
    console.warn("Root worktree is dirty; generated worktrees start from HEAD, not uncommitted edits.");
  }

  if (args.dryRun) {
    const m = nextMilestone(ms);
    if (!m) throw new Error("no milestone found");
    console.log(JSON.stringify(planRound(m, { ...args, run }), null, 2));
    return;
  }

  await mkdir(lock, { recursive: false });
  try {
    await mkdir(run, { recursive: true });
    await writeManifest(args.root, run, args, ms);

    const integration = join(run, "integration");
    await worktree(args.root, integration, `codex/endgame-${id}`, "HEAD");

    for (let round = 1; round <= args.maxRounds; round++) {
      const todo = await readFile(join(integration, "tasks", "todo.md"), "utf8");
      const ms = parseMilestones(todo);
      if (endgameDone(ms)) {
        await runVerify(integration, verify, args);
        console.log(`Endgame achieved in ${integration}`);
        return;
      }

      const m = nextMilestone(ms);
      if (!m) throw new Error("no next milestone found");

      const rdir = join(run, `round-${String(round).padStart(3, "0")}-${m.topic}`);
      const rp = planRound(m, { ...args, root: integration, run: rdir });
      await mkdir(join(rdir, "logs"), { recursive: true });

      await fanout(rp.scouts, args);
      await worktree(integration, rp.worker.cwd, `codex/${id}/${round}-${m.topic}`, "HEAD");
      await runCodex(args.codex, rp.worker, args);
      await exportPatch(rp.worker.cwd, join(rdir, "worker.patch"));
      await applyPatch(integration, join(rdir, "worker.patch"));
      await runVerify(integration, rp.verify, args);
      await reviewAndFix(integration, rp, rdir, args);
      await commit(integration, `orchestrate: complete ${m.topic}`);
    }

    throw new Error(`endgame not achieved after ${args.maxRounds} rounds`);
  } finally {
    await rm(lock, { recursive: true, force: true });
  }
}

async function reviewAndFix(root: string, rp: Round, rdir: string, args: Args) {
  const review = { ...rp.review, cwd: root, prompt: prompt("review", { topic: rp.topic } as Milestone) };
  await runCodex(args.codex, review, args);
  let text = await safeRead(review.out);
  for (let i = 1; /^BLOCKING:\s*yes/im.test(text) && i <= args.maxFix; i++) {
    const cwd = join(rdir, "worktrees", `fix-${i}`);
    await worktree(root, cwd, `codex/fix-${runId()}-${i}`, "HEAD");
    const fix = {
      ...rp.worker,
      role: "fix" as Role,
      cwd,
      out: join(rdir, "logs", `fix-${i}.last.md`),
      log: join(rdir, "logs", `fix-${i}.jsonl`),
      prompt: `${prompt("fix", { topic: rp.topic } as Milestone)}\n\nReview findings:\n${text}`,
    };
    await runCodex(args.codex, fix, args);
    await exportPatch(cwd, join(rdir, `fix-${i}.patch`));
    await applyPatch(root, join(rdir, `fix-${i}.patch`));
    await runVerify(root, rp.verify, args);
    await runCodex(args.codex, review, args);
    text = await safeRead(review.out);
  }
  if (/^BLOCKING:\s*yes/im.test(text)) throw new Error("review blockers remain");
}

async function fanout(agents: Agent[], args: Args) {
  const queue = [...agents];
  const workers = Array.from({ length: Math.max(1, args.jobs) }, async () => {
    for (;;) {
      const agent = queue.shift();
      if (!agent) return;
      await mkdir(dirname(agent.out), { recursive: true });
      await runCodex(args.codex, agent, args);
    }
  });
  await Promise.all(workers);
}

async function runCodex(bin: string, agent: Agent, args: Args) {
  await mkdir(dirname(agent.out), { recursive: true });
  await mkdir(dirname(agent.log), { recursive: true });
  await writeFile(agent.log, "");

  const argv = buildCodexArgs({
    cwd: agent.cwd,
    out: agent.out,
    sandbox: agent.sandbox,
    json: true,
    model: args.model,
  });
  await writeFile(`${agent.out}.prompt.md`, agent.prompt);
  const res = await proc(bin, argv, {
    cwd: agent.cwd,
    input: agent.prompt,
    timeoutMin: args.timeoutMin,
  });
  await writeFile(agent.log, res.stdout);
  await writeFile(`${agent.log}.stderr`, res.stderr);
  if (res.code !== 0 || /"type":"(?:turn\.failed|error)"/.test(res.stdout)) {
    throw new Error(`${agent.role} failed with exit ${res.code}`);
  }
}

async function runVerify(cwd: string, cmds: readonly string[], args: Args) {
  for (const cmd of cmds) {
    const res = await proc(cmd, [], { cwd, shell: true, timeoutMin: args.timeoutMin });
    if (res.code !== 0) {
      throw new Error(`verification failed: ${cmd}\n${res.stderr || res.stdout}`);
    }
  }
}

async function worktree(root: string, dir: string, branch: string, base: string) {
  await mkdir(dirname(dir), { recursive: true });
  await sh(root, `git worktree add -B ${q(branch)} ${q(dir)} ${q(base)}`);
}

async function exportPatch(cwd: string, file: string) {
  await sh(cwd, "git add -N .");
  const diff = await sh(cwd, "git diff --binary HEAD");
  if (!diff.stdout.trim()) throw new Error("worker produced no patch");
  await writeFile(file, diff.stdout);
}

async function applyPatch(cwd: string, file: string) {
  await sh(cwd, `git apply --3way ${q(file)}`);
}

async function commit(cwd: string, msg: string) {
  const status = (await sh(cwd, "git status --porcelain")).stdout.trim();
  if (!status) return;
  await sh(cwd, "git add -A");
  await sh(cwd, `git commit -m ${q(msg)}`);
}

async function writeManifest(root: string, run: string, args: Args, ms: Milestone[]) {
  const [sha, branch, status, agents] = await Promise.all([
    sh(root, "git rev-parse HEAD"),
    sh(root, "git branch --show-current"),
    sh(root, "git status --porcelain"),
    safeRead(join(root, "AGENTS.md")).then(hash),
  ]);
  await writeFile(
    join(run, "manifest.json"),
    JSON.stringify(
      {
        runId: run.split("/").at(-1),
        startedAt: new Date().toISOString(),
        root,
        baseSha: sha.stdout.trim(),
        branch: branch.stdout.trim(),
        dirty: Boolean(status.stdout.trim()),
        agentsSha256: agents,
        maxRounds: args.maxRounds,
        maxFix: args.maxFix,
        jobs: args.jobs,
        verify,
        milestones: ms,
      },
      null,
      2,
    ),
  );
}

function prompt(role: Role, m: Pick<Milestone, "topic" | "desc">): string {
  const base = `
You are a Codex ${role} subagent for Tinkabot.
Read AGENTS.md and tasks/todo.md first. Do not spawn more Codex agents.
Current milestone: ${m.topic}
Milestone note: ${m.desc ?? ""}
Respect matched-abstraction, triage-three, traced-TDD, and be-lazy.
`;

  if (role === "worker" || role === "fix") {
    return `${base}
You are the only writer. Use RED-GREEN-TDD. Implement only this milestone boundary.
Update docs/matched-abstraction/task and tasks/todo.md with concrete evidence.
Run the relevant verification before final. Return:
STATUS: passed|failed|blocked
CHANGED: paths
VERIFY: commands/results
BLOCKERS: remaining blockers
`;
  }

  if (role === "review") {
    return `${base}
Review the current diff only. Do not edit files.
Return exactly one leading line: BLOCKING: yes or BLOCKING: no.
Then list confirmed blockers with file references and required fixes.
`;
  }

  return `${base}
Read-only sidecar. Do not edit files.
Return STATUS: passed|blocked, then concise findings for your role:
approach = layer/invariant risks; tests = RED/GREEN verification gaps; risk = safety/security/collision risks.
`;
}

async function sh(cwd: string, cmd: string) {
  const res = await proc(cmd, [], { cwd, shell: true, timeoutMin: 30 });
  if (res.code !== 0) throw new Error(`${cmd}\n${res.stderr || res.stdout}`);
  return res;
}

async function proc(
  file: string,
  args: string[],
  opts: { cwd: string; input?: string; timeoutMin: number; shell?: boolean },
) {
  const p = spawn(file, args, {
    cwd: opts.cwd,
    shell: opts.shell,
    detached: true,
    stdio: ["pipe", "pipe", "pipe"],
  });
  let killer: ReturnType<typeof setTimeout> | undefined;
  const timer = setTimeout(() => {
    try {
      process.kill(-p.pid!, "SIGTERM");
    } catch {}
    killer = setTimeout(() => {
      try {
        process.kill(-p.pid!, "SIGKILL");
      } catch {}
    }, 10_000);
  }, opts.timeoutMin * 60_000);

  let stdout = "";
  let stderr = "";
  p.stdout.on("data", (data) => (stdout += data));
  p.stderr.on("data", (data) => (stderr += data));
  if (opts.input) p.stdin.end(opts.input);
  else p.stdin.end();

  const code = await new Promise<number>((resolve) => {
    p.on("close", (code) => resolve(code ?? 1));
  });
  clearTimeout(timer);
  if (killer) clearTimeout(killer);
  return { code, stdout, stderr };
}

async function safeRead(file: string) {
  try {
    return await readFile(file, "utf8");
  } catch {
    return "";
  }
}

function hash(text: string) {
  return createHash("sha256").update(text).digest("hex");
}

function runId() {
  return new Date().toISOString().replace(/[:.]/g, "-");
}

function num(value: string | undefined, fallback: number) {
  const n = Number(value);
  return Number.isFinite(n) && n > 0 ? n : fallback;
}

function need(value: string | undefined, flag: string) {
  if (!value) throw new Error(`${flag} requires a value`);
  return value;
}

function q(value: string) {
  return `'${value.replaceAll("'", "'\\''")}'`;
}

function usage() {
  console.log(`
Usage: bun run scripts/codex-orchestrate.ts [options]

Options:
  --dry-run              Print the next round plan without launching Codex
  --allow-dirty          Allow starting while the root worktree is dirty
  --root DIR             Repository root (default: cwd)
  --run-dir DIR          Run directory (default: .codex-runs/endgame/<timestamp>)
  --jobs N               Read-only scout concurrency (default: 3)
  --max-rounds N         Endgame loop cap (default: 12)
  --max-fix N            Review/fix loop cap per round (default: 2)
  --timeout-min N        Per command timeout (default: 90)
  --model MODEL          Pass a model to codex
  --codex BIN            Codex binary (default: codex)
`);
}

if (import.meta.main) {
  main().catch((err) => {
    console.error(err instanceof Error ? err.message : err);
    process.exitCode = 1;
  });
}
