// Owning tests for every gate finding family (quality-v1.md:71). Each case
// drives the gate's check() over a throwaway fixture corpus (git-indexed temp
// dir, pointed at via setGoDir) and asserts the family fires — regression
// signal that the detection logic keeps detecting.

import { execSync } from "node:child_process";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { afterAll, beforeAll, describe, expect, test } from "bun:test";
import { setGoDir, type Finding } from "../scripts/gate-lib";
import { check as fakes } from "../scripts/gate-fakes";
import { check as parallel } from "../scripts/gate-parallel";
import { check as coverage } from "../scripts/gate-coverage";
import { check as scenarios } from "../scripts/gate-scenarios";
import { check as manual } from "../scripts/gate-manual";

const dirs: string[] = [];

// goFiles() shells out to `git ls-files`, so the fixture is a real git index.
function corpus(files: Record<string, string>) {
  const dir = mkdtempSync(join(tmpdir(), "gate-corpus-"));
  dirs.push(dir);
  for (const [rel, src] of Object.entries(files)) {
    mkdirSync(join(dir, dirname(rel)), { recursive: true });
    writeFileSync(join(dir, rel), src);
  }
  execSync("git init -q && git add -A", { cwd: dir });
  setGoDir(dir);
}

afterAll(() => {
  for (const d of dirs) rmSync(d, { recursive: true, force: true });
});

const has = (f: Finding[], family: string, fragment: string) =>
  f.some((x) => x.family === family && x.detail.includes(fragment));

describe("gate:fakes", () => {
  test("fakes-violation: un-allowlisted fake", () => {
    corpus({
      "clock/fake.go": "package clock\n\ntype FakeClock struct{}\n",
      "fakes-allowlist.json": "[]",
    });
    expect(has(fakes(), "fakes-violation", "un-allowlisted fake FakeClock")).toBe(true);
  });

  test("measurement-stale: allowlist entry matching no fake", () => {
    corpus({
      "clock/clock.go": "package clock\n",
      "fakes-allowlist.json": JSON.stringify([
        {
          fake: "FakeGhost",
          definedIn: "clock/gone.go:1",
          justification: "x".repeat(80),
          realProof: "TestGhost",
        },
      ]),
    });
    expect(has(fakes(), "measurement-stale", "FakeGhost matches no fake")).toBe(true);
  });
});

describe("gate:parallel", () => {
  test("serialized-execution: Test func without t.Parallel()", () => {
    corpus({
      "pkg/a_test.go": `package pkg

import "testing"

func TestSerial(t *testing.T) {
	_ = t
}
`,
    });
    expect(has(parallel(), "serialized-execution", "TestSerial never calls t.Parallel()")).toBe(true);
  });

  test("isolation-violation: second embednats.Start call site", () => {
    const startTest = (name: string) => `package pkg

import "testing"

func Test${name}(t *testing.T) {
	t.Parallel()
	embednats.Start(cfg)
}
`;
    corpus({ "a/a_test.go": startTest("A"), "b/b_test.go": startTest("B") });
    expect(has(parallel(), "isolation-violation", "constructed directly in 2 test files")).toBe(true);
  });
});

describe("gate:coverage", () => {
  let findings: Finding[];

  beforeAll(() => {
    corpus({
      "go.mod": "module fixture\n\ngo 1.25\n",
      "calc/calc.go": `package calc

func Covered() int { return 1 }

func Uncovered() int { return 2 }
`,
      "calc/calc_test.go": `package calc

import "testing"

func TestCovered(t *testing.T) {
	t.Parallel()
	if Covered() != 1 {
		t.Fatal("covered")
	}
}
`,
      "coverage-thresholds.json": JSON.stringify({ calc: 90, ghost: 50 }),
    });
    findings = coverage();
  });

  test("coverage-gap: layer below declared threshold", () => {
    expect(has(findings, "coverage-gap", "layer calc: coverage 50% below declared threshold 90%")).toBe(true);
  });

  test("measurement-stale: threshold for unknown layer", () => {
    expect(has(findings, "measurement-stale", "unknown layer ghost")).toBe(true);
  });
});

describe("gate:scenarios", () => {
  test("scenario-matrix-hole: required Tinkalet trigger surface missing", () => {
    corpus({
      "svc/svc_test.go": `package svc

import "testing"

func TestAllowed(t *testing.T) { t.Parallel() }
`,
      "scenario-matrix.json": JSON.stringify({
        api: {
          allowed: ["TestAllowed"],
          "denied-neighbor": ["TestAllowed"],
          malformed: ["TestAllowed"],
          duplicate: ["TestAllowed"],
          stale: ["TestAllowed"],
          revoked: ["TestAllowed"],
          "attributed-failure": ["TestAllowed"],
        },
      }),
    });
    expect(has(scenarios(), "scenario-matrix-hole", "required outside-in surface tinkalet-trigger is absent")).toBe(true);
  });

  test("scenario-matrix-hole: pinned family missing from present matrix", () => {
    corpus({
      "svc/svc_test.go": `package svc

import "testing"

func TestAllowed(t *testing.T)   { t.Parallel() }
func TestDenied(t *testing.T)    { t.Parallel() }
func TestMalformed(t *testing.T) { t.Parallel() }
func TestDuplicate(t *testing.T) { t.Parallel() }
func TestStale(t *testing.T)     { t.Parallel() }
func TestRevoked(t *testing.T)   { t.Parallel() }
`,
      "scenario-matrix.json": JSON.stringify({
        api: {
          allowed: ["TestAllowed"],
          "denied-neighbor": ["TestDenied"],
          malformed: ["TestMalformed"],
          duplicate: ["TestDuplicate"],
          stale: ["TestStale"],
          revoked: ["TestRevoked"],
          // attributed-failure intentionally absent
        },
      }),
    });
    const f = scenarios();
    expect(has(f, "scenario-matrix-hole", "no case for pinned family attributed-failure")).toBe(true);
    expect(f.filter((x) => x.family === "measurement-stale")).toEqual([]);
  });

  test("measurement-stale: non-resolving citation and unknown family", () => {
    corpus({
      "svc/svc_test.go": `package svc

import "testing"

func TestReal(t *testing.T) { t.Parallel() }
`,
      "scenario-matrix.json": JSON.stringify({
        api: { allowed: ["TestGhost"], bogus: ["TestReal"] },
      }),
    });
    const f = scenarios();
    expect(has(f, "measurement-stale", '"TestGhost" does not resolve')).toBe(true);
    expect(has(f, "measurement-stale", "unknown case family bogus")).toBe(true);
  });
});

// gate:manual (quality-release, plan/quality-v1.md:93): manual commands run
// verbatim against the running binary and produce the documented outcomes.
// The unit seam is manual text plus a live-output transcript — the same
// committed-text seam release-evidence.test.ts uses — while the real-binary
// seam proof is `bun run gate:manual` itself. Denial oracles are output
// text, never exit codes (nats CLI v0.3.0 exits 0 on permission errors).
describe("gate:manual", () => {
  const doc = `# Manual

\`\`\`bash
nats request --raw -H Tinkabot-Request-Id:req-rel-001 tb.proof.runtime.execute ping
# -> Accepted
\`\`\`
`;

  test("measurement-stale: manual yielding no command/outcome pairs", () => {
    expect(has(manual("# no bash blocks", () => ""), "measurement-stale", "no command/outcome pairs")).toBe(true);
  });

  test("manual-divergence: live output diverges from the documented outcome", () => {
    const f = manual(doc, () => "LeaseRevoked");
    expect(has(f, "manual-divergence", "Accepted")).toBe(true);
  });

  test("verbatim manual commands with matching live outcomes pass", () => {
    expect(manual(doc, () => "Accepted")).toEqual([]);
  });
});
