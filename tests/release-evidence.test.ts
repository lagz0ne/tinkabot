import { describe, expect, test } from "bun:test";
import { check, type Entry, type Gates, type Manifest, type Repo } from "../scripts/release-evidence";

// The checker's seam is committed document text and committed Go test names,
// not NATS behavior. An in-memory repo is the honest unit fixture here; the
// real-corpus seam proof is `bun run release:evidence` against the repository.

const TASK = "docs/matched-abstraction/task/m1.md";

const goodDoc = `---
layer: task
topic: m1
---

# M1 Task

## RED Artifact

Planned failing proof.

## Verification Evidence

RED:

- \`bun run m1:test\` -> failed with missing \`M1Symbol\`.

GREEN:

- \`bun run m1:test\` -> \`3 pass\`, \`0 fail\`.
- \`go test ./m1 -run 'TestM1' -count=1\` -> denied neighbor subject before effect.
- duplicate command resolved through idempotency without a second effect.

## Wrap-Up

The m1 milestone is complete and verified.
`;

const gates: Gates = {
  milestones: ["m1"],
  spine: ["s1"],
  deferred: ["d1"],
  requiredGuards: { m1: ["g1"] },
  requiredCases: ["denied_neighbor", "duplicate"],
};

const entry: Entry = {
  milestone: "m1",
  taskDoc: TASK,
  spineSteps: ["s1"],
  red: { doc: TASK, command: "bun run m1:test", result: "failed with missing `M1Symbol`" },
  insideOut: ["bun run m1:test"],
  outsideIn: ["go test ./m1 -run 'TestM1' -count=1"],
  negativeCases: [
    { case: "denied_neighbor", doc: TASK, quote: "denied neighbor subject before effect" },
    { case: "duplicate", doc: TASK, quote: "duplicate command resolved through idempotency" },
  ],
  scopeGuards: ["g1"],
};

const manifest: Manifest = {
  milestones: [entry],
  deferredScope: ["d1"],
  docAuthority: [{ domain: "m1", docs: [TASK] }],
};

const repo = (files: Record<string, string> = { [TASK]: goodDoc }): Repo => ({
  read: (rel) => files[rel] ?? null,
  goTestNames: () => ["TestM1DeniesNeighbor", "TestM1AcceptsCommand"],
});

const mutate = (fn: (m: Manifest) => void) => {
  const m = structuredClone(manifest);
  fn(m);
  return m;
};

const families = (m: Manifest, r: Repo = repo()) => check(m, r, gates).map((f) => f.family);

test("passes a complete, resolved, in-scope corpus", () => {
  expect(check(manifest, repo(), gates)).toEqual([]);
});

describe("manifest-incomplete", () => {
  test("missing required milestone", () => {
    expect(families(mutate((m) => (m.milestones = [])))).toContain("manifest-incomplete");
  });

  test("uncovered spine step", () => {
    const m = mutate((x) => (x.milestones[0]!.spineSteps = []));
    const found = check(m, repo(), gates);
    expect(found.some((f) => f.family === "manifest-incomplete" && f.detail.includes("spine step s1"))).toBe(true);
  });

  test("missing executed RED citation", () => {
    const m = mutate((x) => (x.milestones[0]!.red = null));
    const found = check(m, repo(), gates);
    expect(found).toEqual([
      { family: "manifest-incomplete", milestone: "m1", detail: "entry lacks an executed RED command/result citation" },
    ]);
  });

  test("no outside-in commands and no not-applicable reason", () => {
    const m = mutate((x) => (x.milestones[0]!.outsideIn = []));
    expect(families(m)).toContain("manifest-incomplete");
  });

  test("dropping every case for a pinned family is caught", () => {
    const m = mutate((x) => {
      x.milestones[0]!.negativeCases = x.milestones[0]!.negativeCases.filter((nc) => nc.case !== "denied_neighbor");
    });
    const found = check(m, repo(), gates);
    expect(
      found.some((f) => f.family === "manifest-incomplete" && f.detail.includes("pinned case family denied_neighbor")),
    ).toBe(true);
  });

  test("missing doc authority map", () => {
    const m = mutate((x) => (x.docAuthority = []));
    expect(families(m)).toContain("manifest-incomplete");
  });
});

describe("citation-unresolved", () => {
  test("owning task doc does not exist", () => {
    expect(families(manifest, repo({}))).toContain("citation-unresolved");
  });

  test("RED command string is not in the owning doc", () => {
    const m = mutate((x) => (x.milestones[0]!.red!.command = "bun run never-ran"));
    expect(families(m)).toContain("citation-unresolved");
  });

  test("inside-out command string is not in the owning doc", () => {
    const m = mutate((x) => (x.milestones[0]!.insideOut = ["bun run phantom"]));
    expect(families(m)).toContain("citation-unresolved");
  });

  test("go test -run prefix resolves against committed test names only", () => {
    const m = mutate((x) => (x.milestones[0]!.outsideIn = ["go test ./m1 -run 'TestM1|TestInvented' -count=1"]));
    const doc = goodDoc.replace("go test ./m1 -run 'TestM1' -count=1", "go test ./m1 -run 'TestM1|TestInvented' -count=1");
    const found = check(m, repo({ [TASK]: doc }), gates);
    expect(found.some((f) => f.family === "citation-unresolved" && f.detail.includes("TestInvented"))).toBe(true);
  });

  test("negative-case quote is not in the cited doc", () => {
    const m = mutate((x) => (x.milestones[0]!.negativeCases[0]!.quote = "never written down"));
    expect(families(m)).toContain("citation-unresolved");
  });
});

describe("scope-overclaim", () => {
  test("required scope guard is omitted", () => {
    const m = mutate((x) => (x.milestones[0]!.scopeGuards = []));
    const found = check(m, repo(), gates);
    expect(found).toEqual([{ family: "scope-overclaim", milestone: "m1", detail: "entry omits required scope guard g1" }]);
  });

  test("deferred scope item is not named", () => {
    const m = mutate((x) => (x.deferredScope = []));
    expect(families(m)).toContain("scope-overclaim");
  });

  test("entry text claims deferred work as done", () => {
    const m = mutate((x) => (x.milestones[0]!.notes = "live multi-node behavior is covered"));
    expect(families(m)).toContain("scope-overclaim");
  });

  test("denial evidence citing an exit-code oracle is rejected", () => {
    const doc = goodDoc.replace(
      "denied neighbor subject before effect",
      "denied neighbor subject before effect, exit code 0 observed",
    );
    const m = mutate(
      (x) => (x.milestones[0]!.negativeCases[0]!.quote = "denied neighbor subject before effect, exit code 0 observed"),
    );
    expect(families(m, repo({ [TASK]: doc }))).toContain("scope-overclaim");
  });
});

describe("evidence-stale", () => {
  test("template wrap-up instead of completion record", () => {
    const doc = goodDoc.replace(
      "The m1 milestone is complete and verified.",
      "When complete, announce that m1 owns its boundary.",
    );
    const found = check(manifest, repo({ [TASK]: doc }), gates);
    expect(found).toEqual([
      {
        family: "evidence-stale",
        milestone: "m1",
        detail: `${TASK} wrap-up is a template announcement, not a completion record`,
      },
    ]);
  });

  test("conditional 'is complete when' wrap-up is a template", () => {
    const doc = goodDoc.replace(
      "The m1 milestone is complete and verified.",
      "The m1 milestone is complete when its boundary is proven.",
    );
    expect(families(manifest, repo({ [TASK]: doc }))).toContain("evidence-stale");
  });

  test("negative-case evidence hidden behind an aggregate pass result", () => {
    const m = mutate((x) => (x.milestones[0]!.negativeCases[0]!.quote = "`bun run m1:test` -> `3 pass`, `0 fail`."));
    const found = check(m, repo(), gates);
    expect(
      found.some((f) => f.family === "evidence-stale" && f.detail.includes("hidden behind an aggregate pass result")),
    ).toBe(true);
  });

  test("negative-case quote cited from claim text outside verification evidence", () => {
    const m = mutate((x) => (x.milestones[0]!.negativeCases[0]!.quote = "Planned failing proof."));
    const found = check(m, repo(), gates);
    expect(
      found.some((f) => f.family === "evidence-stale" && f.detail.includes("outside executed verification evidence")),
    ).toBe(true);
  });

  test("superseded authority doc without a supersession marker", () => {
    const old = "docs/matched-abstraction/plan/old.md";
    const m = mutate((x) => (x.docAuthority[0]!.superseded = [old]));
    const found = check(m, repo({ [TASK]: goodDoc, [old]: "# Old Plan\n\nStill reads as authority." }), gates);
    expect(found.some((f) => f.family === "evidence-stale" && f.detail.includes("no supersession marker"))).toBe(true);
  });
});
