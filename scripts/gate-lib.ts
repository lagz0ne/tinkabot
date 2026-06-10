// Shared corpus scanning for the four standing quality gates
// (docs/matched-abstraction/plan/quality-v1.md:87-92). Each gate reads the
// committed Go test corpus under substrate/go and reports findings in the
// slice's owned failure families (quality-v1.md:71).

import { execSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { join } from "node:path";

export const root = join(import.meta.dir, "..");
export let goDir = join(root, "substrate/go");

// Tests point the gates at a fixture corpus.
export function setGoDir(dir: string) {
  goDir = dir;
}

export type Finding = { family: string; detail: string };

export function goFiles(): string[] {
  return execSync("git ls-files '*.go'", { cwd: goDir, encoding: "utf8" })
    .split("\n")
    .filter(Boolean);
}

export const read = (rel: string) => readFileSync(join(goDir, rel), "utf8");

export function readJSON(rel: string): unknown | null {
  try {
    return JSON.parse(read(rel));
  } catch {
    return null;
  }
}

export const lineOf = (src: string, index: number) =>
  src.slice(0, index).split("\n").length;

export type TestFn = { file: string; name: string; line: number; body: string };

// Top-level Test funcs per committed _test.go file. Body runs to the next
// top-level func, so subtest t.Parallel() calls count for their parent.
export function testFns(): TestFn[] {
  const out: TestFn[] = [];
  for (const file of goFiles().filter((f) => f.endsWith("_test.go"))) {
    const src = read(file);
    const tops = [...src.matchAll(/^func /gm)].map((m) => m.index);
    for (const m of src.matchAll(/^func (Test\w+)\(/gm)) {
      const next = tops.find((i) => i > m.index) ?? src.length;
      out.push({
        file,
        name: m[1],
        line: lineOf(src, m.index),
        body: src.slice(m.index, next),
      });
    }
  }
  return out;
}

// A citation resolves when its first slash segment exactly names a committed
// Test func (stricter than release-evidence.ts's startsWith rule).
export function resolves(citation: string, names: Set<string>): boolean {
  return names.has(citation.split("/")[0]);
}

export function report(gate: string, findings: Finding[]): never {
  if (findings.length === 0) {
    console.log(`${gate} passed`);
    process.exit(0);
  }
  const families = [...new Set(findings.map((f) => f.family))];
  for (const family of families) {
    const group = findings.filter((f) => f.family === family);
    console.error(`\n${family} (${group.length})`);
    for (const f of group) console.error(`  - ${f.detail}`);
  }
  const counts = families
    .map((f) => `${f}=${findings.filter((x) => x.family === f).length}`)
    .join(", ");
  console.error(`\n${gate} FAILED: ${findings.length} findings (${counts})`);
  process.exit(1);
}
