// gate:parallel — the full Go suite must run green over real embedded NATS
// with -shuffle=on, parallel execution, and per-test isolated servers/stores
// obtained through one shared embednats harness factory seam
// (quality-v1.md:89, quality-v1.md:43).
//
// Structural rules checked before the suite runs:
// 1. Every top-level Test func calls t.Parallel(), or carries a
//    `gate:serial` comment naming why its state forbids it.
// 2. Direct embednats server construction (Start(cfg)) in _test.go files is
//    confined to at most one file — the harness factory seam. Everything
//    else must obtain isolated servers through that seam.
// If the structure is serialized or seamless, the gate fails without running
// the suite: a green serial run proves nothing about isolation.

import { spawnSync } from "node:child_process";
import { goDir, goFiles, lineOf, read, report, testFns, type Finding } from "./gate-lib";

export function check(): Finding[] {
  const findings: Finding[] = [];

  for (const fn of testFns()) {
    const head = read(fn.file).split("\n")[fn.line - 2] ?? "";
    if (!fn.body.includes("t.Parallel()") && !head.includes("gate:serial")) {
      findings.push({
        family: "serialized-execution",
        detail: `${fn.file}:${fn.line} ${fn.name} never calls t.Parallel() and carries no gate:serial justification`,
      });
    }
  }

  const construction = new Map<string, number[]>();
  for (const file of goFiles().filter((f) => f.endsWith("_test.go"))) {
    const src = read(file);
    const direct = file.startsWith("embednats/")
      ? /(?:^|[^.\w])Start\(/gm // bare in-package calls
      : /\bembednats\.Start\(/g;
    const lines = [...src.matchAll(direct)].map((m) => lineOf(src, m.index));
    if (lines.length) construction.set(file, lines);
  }
  if (construction.size > 1) {
    const sites = [...construction]
      .map(([f, lines]) => `${f}:${lines.join(",")}`)
      .join("; ");
    findings.push({
      family: "isolation-violation",
      detail: `no single harness factory seam: embednats.Start constructed directly in ${construction.size} test files (${sites})`,
    });
  }

  if (findings.length) {
    findings.push({
      family: "serialized-execution",
      detail: "shuffled parallel suite run skipped: structural findings above make a green run meaningless",
    });
    return findings;
  }

  const run = spawnSync("go", ["test", "./...", "-count=1", "-shuffle=on"], {
    cwd: goDir,
    encoding: "utf8",
    stdio: ["ignore", "inherit", "inherit"],
  });
  if (run.status !== 0) {
    findings.push({
      family: "isolation-violation",
      detail: `go test ./... -count=1 -shuffle=on exited ${run.status}`,
    });
  }
  return findings;
}

if (import.meta.main) report("gate:parallel", check());
