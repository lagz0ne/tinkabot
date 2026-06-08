import { describe, expect, test } from "bun:test";
import {
  buildCodexArgs,
  endgameDone,
  nextMilestone,
  parseMilestones,
  planRound,
} from "../scripts/codex-orchestrate";

const todo = `
## Milestone Workflow

1. DONE: \`endgame-contract-authority\`: neutral schemas.
2. DONE: \`managed-auth-subjects\`: auth compile proof.
3. NEXT: \`command-acceptance\`: durable intent acceptance.
4. \`substrate-edge-bootstrap\`: Go substrate boundary.
`;

describe("CodexEndgameOrchestrator", () => {
  test("T-ORCH-PARSE finds the next unfinished endgame milestone", () => {
    const ms = parseMilestones(todo);

    expect(ms.map(({ status, topic }) => ({ status, topic }))).toEqual([
      { status: "DONE", topic: "endgame-contract-authority" },
      { status: "DONE", topic: "managed-auth-subjects" },
      { status: "NEXT", topic: "command-acceptance" },
      { status: "TODO", topic: "substrate-edge-bootstrap" },
    ]);
    expect(nextMilestone(ms)?.topic).toBe("command-acceptance");
    expect(endgameDone(ms)).toBe(false);
  });

  test("T-ORCH-DONE requires every endgame milestone to be DONE", () => {
    const done = todo
      .replace("NEXT: `command-acceptance`", "DONE: `command-acceptance`")
      .replace("4. `substrate-edge-bootstrap`", "4. DONE: `substrate-edge-bootstrap`");

    expect(endgameDone(parseMilestones(done))).toBe(true);
  });

  test("T-ORCH-CODEX builds noninteractive safe Codex calls", () => {
    const args = buildCodexArgs({
      cwd: "/repo",
      out: "/tmp/last.md",
      sandbox: "read-only",
      json: true,
    });

    expect(args).toContain("exec");
    expect(args).toContain("--cd");
    expect(args).toContain("/repo");
    expect(args).toContain("--ask-for-approval");
    expect(args).toContain("never");
    expect(args).toContain("--sandbox");
    expect(args).toContain("read-only");
    expect(args).toContain("--output-last-message");
    expect(args).toContain("/tmp/last.md");
    expect(args).toContain("--json");
    expect(args).not.toContain("--dangerously-bypass-approvals-and-sandbox");
  });

  test("T-ORCH-ROUND fans out read-only scouts and keeps one writer", () => {
    const round = planRound(nextMilestone(parseMilestones(todo))!, {
      root: "/repo",
      run: "/repo/.codex-runs/endgame/round-001",
      jobs: 3,
    });

    expect(round.scouts.map(({ role }) => role)).toEqual([
      "approach",
      "tests",
      "risk",
    ]);
    expect(round.scouts.every(({ cwd }) => cwd === "/repo")).toBe(true);
    expect(round.scouts.every(({ sandbox }) => sandbox === "read-only")).toBe(true);
    expect(round.worker.role).toBe("worker");
    expect(round.worker.sandbox).toBe("workspace-write");
    expect(round.worker.cwd).toBe(
      "/repo-round-001-3-command-acceptance-worker",
    );
    expect(round.verify).toContain("bun run validate:layers");
    expect(round.verify).toContain("git diff --check");
  });
});
