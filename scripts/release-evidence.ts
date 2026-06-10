// Centralized release authority checker for release/endgame-v1.json.
// Gate rules come from docs/matched-abstraction/plan/endgame-app.md
// (Release-Spine Decomposition, lines 165-181): sixteen milestones over the
// eleven Release Verification Spine steps, resolvable citations, Plan-owned
// deferred scope, four scope guards, doc authority map, and four owned
// failure families.

import { readFileSync } from "node:fs";
import { join } from "node:path";

export type Family =
  | "manifest-incomplete"
  | "citation-unresolved"
  | "scope-overclaim"
  | "evidence-stale";

export type Finding = { family: Family; milestone?: string; detail: string };

export type Citation = { doc: string; command: string; result: string };
export type NegCase = { case: string; doc: string; quote: string };

export type Entry = {
  milestone: string;
  taskDoc: string;
  spineSteps: string[];
  red: Citation | null;
  insideOut: string[];
  outsideIn: string[];
  outsideInNA?: string;
  negativeCases: NegCase[];
  scopeGuards: string[];
  notes?: string;
};

export type Authority = { domain: string; docs: string[]; superseded?: string[] };

export type Manifest = {
  milestones: Entry[];
  deferredScope: string[];
  docAuthority: Authority[];
};

export type Repo = {
  read: (rel: string) => string | null;
  goTestNames: () => string[];
};

export type Gates = {
  milestones: string[];
  spine: string[];
  deferred: string[];
  requiredGuards: Record<string, string[]>;
  requiredCases: string[];
};

// endgame-app.md:173 — gate list is all sixteen endgame milestones.
export const MILESTONES = [
  "endgame-contract-authority",
  "managed-auth-subjects",
  "command-acceptance",
  "substrate-edge-bootstrap",
  "go-substrate-core",
  "embedded-nats-adapter",
  "frontend-isolation-layer",
  "browser-isolation-proof",
  "activation-contract-authority",
  "activation-ledger-durability",
  "activation-source-authority",
  "activation-router-live-sources",
  "activation-schedule-engine",
  "activation-release-proof",
  "script-materializer-loop",
  "release-spine",
];

// endgame-app.md:133-149 — the eleven Release Verification Spine steps.
export const SPINE = [
  "contract",
  "auth-subjects",
  "browser-edge",
  "browser-isolation",
  "service-worker-setup",
  "command-acceptance",
  "activation",
  "script-runtime",
  "materializer-artifact",
  "frontend-rendering",
  "release-ops",
];

// endgame-app.md:175 — Plan-owned deferred scope, never presented as proven.
export const DEFERRED = [
  "direct-browser-nats-websocket",
  "docker-sandboxing",
  "product-ui-rendering",
  "live-auth-reload",
  "wall-clock-scheduler-loops",
  "broad-script-crud-ui",
  "live-multi-node-ha-scale",
  "package-publication",
];

// endgame-app.md:177 — the four scope guards, pinned to the milestones whose
// evidence they bound.
export const REQUIRED_GUARDS: Record<string, string[]> = {
  "go-substrate-core": ["ha-scale-contract-shape-only"],
  "managed-auth-subjects": ["managed-auth-policy-compile-only"],
  "activation-schedule-engine": ["schedule-engine-proof-no-live-tick-source"],
  "activation-source-authority": ["nats-cli-denial-output-parsed-oracle"],
  "activation-release-proof": [
    "nats-cli-denial-output-parsed-oracle",
    "schedule-engine-proof-no-live-tick-source",
  ],
  "script-materializer-loop": ["nats-cli-denial-output-parsed-oracle"],
};

// todo.md:235 — release gates must include these case families over
// NATS-mediated behavior. Pinned like REQUIRED_GUARDS so the manifest
// cannot weaken its own gates by dropping a family's last covering case.
export const REQUIRED_CASES = [
  "allowed",
  "denied_neighbor",
  "malformed",
  "duplicate",
  "stale_revision",
  "revoked_lease",
  "attributed_failure",
];

export const PLAN_GATES: Gates = {
  milestones: MILESTONES,
  spine: SPINE,
  deferred: DEFERRED,
  requiredGuards: REQUIRED_GUARDS,
  requiredCases: REQUIRED_CASES,
};

// Capability Proof Matrix (endgame-app.md:183-198): a negative-case citation
// must name its case; an aggregate pass line proves nothing case-specific.
const CASE_WORDS: Record<string, RegExp> = {
  allowed: /allow|accept|grant/i,
  denied_neighbor: /denie[ds]|deny|denial/i,
  malformed: /malformed|invalid|reject/i,
  duplicate: /duplicate|dedupe|idempot|no-rerun|replay/i,
  stale_revision: /stale/i,
  revoked_lease: /revok|expired/i,
  session_scope_mismatch: /scope|session/i,
  frame_lease_mismatch: /leased|frame lease|nonce|source window/i,
  loop_suppressed: /loop|suppress/i,
  attributed_failure: /attribut/i,
};

// Phrases that promote completed milestones beyond their evidence
// (endgame-app.md:175,177; todo.md:220,232-233).
const OVERCLAIM = [
  "live multi-node",
  "failover proven",
  "clustering proven",
  "jetstream replica quorum proven",
  "live scheduler",
  "wall-clock loop proven",
  "package published",
  "publish to npm",
  "product ui proven",
  "docker sandbox proven",
  "live auth reload proven",
  "direct browser nats websocket proven",
];

const TEMPLATE_WRAP = [/When complete, announce/i, /is complete\b[^.\n]*\bwhen\b/i];

const runPattern = (cmd: string) => {
  if (!/\bgo test\b/.test(cmd)) return null;
  const m = cmd.match(/-run\s+'([^']+)'/) ?? cmd.match(/-run\s+"([^"]+)"/) ?? cmd.match(/-run\s+(\S+)/);
  return m?.[1] ?? null;
};

export function check(manifest: Manifest, repo: Repo, gates: Gates = PLAN_GATES): Finding[] {
  const out: Finding[] = [];
  const add = (family: Family, detail: string, milestone?: string) =>
    out.push(milestone ? { family, milestone, detail } : { family, detail });

  if (!Array.isArray(manifest?.milestones)) {
    add("manifest-incomplete", "manifest has no milestones array");
    return out;
  }

  const names = new Set(manifest.milestones.map((e) => e.milestone));
  for (const m of gates.milestones) {
    if (!names.has(m)) add("manifest-incomplete", "required milestone has no manifest entry", m);
  }

  const covered = new Set(manifest.milestones.flatMap((e) => e.spineSteps ?? []));
  for (const s of gates.spine) {
    if (!covered.has(s)) add("manifest-incomplete", `spine step ${s} has no covering milestone entry`);
  }

  const goTests = repo.goTestNames();
  const checkRun = (cmd: string, milestone: string) => {
    const pat = runPattern(cmd);
    if (!pat) return;
    for (const alt of pat.split("|")) {
      if (!goTests.some((n) => n.startsWith(alt))) {
        add("citation-unresolved", `go test -run prefix ${alt} matches no committed Go test`, milestone);
      }
    }
  };

  const wrapChecked = new Set<string>();

  for (const e of manifest.milestones) {
    const m = e.milestone;
    if (!gates.milestones.includes(m)) {
      add("manifest-incomplete", "milestone is not in the Plan gate list", m);
      continue;
    }
    if (!e.spineSteps?.length) add("manifest-incomplete", "entry covers no spine step", m);
    for (const s of e.spineSteps ?? []) {
      if (!gates.spine.includes(s)) add("manifest-incomplete", `unknown spine step ${s}`, m);
    }

    const doc = repo.read(e.taskDoc);
    if (doc === null) {
      add("citation-unresolved", `owning task doc ${e.taskDoc} not found`, m);
      continue;
    }

    if (!wrapChecked.has(e.taskDoc)) {
      wrapChecked.add(e.taskDoc);
      const wrapAt = doc.search(/^## Wrap-Up/m);
      const wrap = wrapAt >= 0 ? doc.slice(wrapAt) : "";
      if (TEMPLATE_WRAP.some((re) => re.test(wrap))) {
        add("evidence-stale", `${e.taskDoc} wrap-up is a template announcement, not a completion record`, m);
      }
    }

    const evidenceAt = doc.search(/^## Verification/m);
    const inDoc = (what: string) => {
      if (!doc.includes(what)) add("citation-unresolved", `${what} not found in ${e.taskDoc}`, m);
    };

    if (!e.red) {
      add("manifest-incomplete", "entry lacks an executed RED command/result citation", m);
    } else {
      const rdoc = e.red.doc === e.taskDoc ? doc : repo.read(e.red.doc);
      if (rdoc === null) {
        add("citation-unresolved", `RED citation doc ${e.red.doc} not found`, m);
      } else {
        if (!e.red.command || !rdoc.includes(e.red.command)) {
          add("citation-unresolved", `RED command not found in ${e.red.doc}`, m);
        }
        if (!e.red.result) {
          add("manifest-incomplete", "RED citation has no recorded failure result", m);
        } else if (!rdoc.includes(e.red.result)) {
          add("citation-unresolved", `RED result not found in ${e.red.doc}`, m);
        }
        if (e.red.command) checkRun(e.red.command, m);
      }
    }

    if (!e.insideOut?.length) add("manifest-incomplete", "entry names no inside-out proof commands", m);
    for (const cmd of e.insideOut ?? []) {
      inDoc(cmd);
      checkRun(cmd, m);
    }

    if (!e.outsideIn?.length && !e.outsideInNA) {
      add("manifest-incomplete", "entry needs outside-in proof commands or an explicit not-applicable reason", m);
    }
    for (const cmd of e.outsideIn ?? []) {
      inDoc(cmd);
      checkRun(cmd, m);
    }

    if (!e.negativeCases?.length) {
      add("manifest-incomplete", "entry names no negative-case coverage", m);
    }
    for (const nc of e.negativeCases ?? []) {
      const word = CASE_WORDS[nc.case];
      if (!word) {
        add("manifest-incomplete", `unknown negative case ${nc.case}`, m);
        continue;
      }
      const ndoc = nc.doc === e.taskDoc ? doc : repo.read(nc.doc);
      if (ndoc === null) {
        add("citation-unresolved", `negative case ${nc.case} cites missing doc ${nc.doc}`, m);
        continue;
      }
      const at = ndoc.indexOf(nc.quote);
      const nEvidenceAt = nc.doc === e.taskDoc ? evidenceAt : ndoc.search(/^## Verification/m);
      if (at < 0) {
        add("citation-unresolved", `negative case ${nc.case} quote not found in ${nc.doc}`, m);
        continue;
      }
      if (nEvidenceAt < 0 || at < nEvidenceAt) {
        add("evidence-stale", `negative case ${nc.case} cites claim text outside executed verification evidence`, m);
      } else if (!word.test(nc.quote)) {
        add("evidence-stale", `negative case ${nc.case} is hidden behind an aggregate pass result that never names the case`, m);
      }
      // nats CLI v0.3.0 reports permission errors in output while exiting 0,
      // so denial evidence must be output-parsed (endgame-app.md:177).
      if (/exit (code|status)/i.test(nc.quote) && /denied|revoked/.test(nc.case)) {
        add("scope-overclaim", `negative case ${nc.case} relies on an exit-code oracle instead of output-parsed denial`, m);
      }
    }

    for (const g of gates.requiredGuards[m] ?? []) {
      if (!e.scopeGuards?.includes(g)) {
        add("scope-overclaim", `entry omits required scope guard ${g}`, m);
      }
    }

    const blob = JSON.stringify(e).toLowerCase();
    for (const p of OVERCLAIM) {
      if (blob.includes(p)) add("scope-overclaim", `entry claims beyond its evidence: "${p}"`, m);
    }
  }

  const cited = new Set(manifest.milestones.flatMap((e) => (e.negativeCases ?? []).map((nc) => nc.case)));
  for (const c of gates.requiredCases) {
    if (!cited.has(c)) add("manifest-incomplete", `pinned case family ${c} has no covering case in any milestone`);
  }

  for (const d of gates.deferred) {
    if (!manifest.deferredScope?.includes(d)) {
      add("scope-overclaim", `deferred scope item ${d} is not named in the manifest`);
    }
  }

  if (!manifest.docAuthority?.length) {
    add("manifest-incomplete", "manifest records no doc authority map");
  }
  for (const a of manifest.docAuthority ?? []) {
    for (const p of a.docs ?? []) {
      if (repo.read(p) === null) add("citation-unresolved", `authority doc ${p} not found for domain ${a.domain}`);
    }
    for (const p of a.superseded ?? []) {
      const s = repo.read(p);
      if (s === null) add("citation-unresolved", `superseded doc ${p} not found for domain ${a.domain}`);
      else if (!/supersession|supersede/i.test(s)) {
        add("evidence-stale", `superseded doc ${p} carries no supersession marker for domain ${a.domain}`);
      }
    }
  }

  return out;
}

export function fsRepo(root: string): Repo {
  let goNames: string[] | null = null;
  return {
    read: (rel) => {
      try {
        return readFileSync(join(root, rel), "utf8");
      } catch {
        return null;
      }
    },
    goTestNames: () => {
      if (goNames) return goNames;
      const ls = Bun.spawnSync(["git", "ls-files", "substrate/go/*_test.go"], { cwd: root });
      const files = ls.stdout.toString().split("\n").filter(Boolean);
      goNames = files.flatMap((f) => {
        const src = readFileSync(join(root, f), "utf8");
        return [...src.matchAll(/^func (Test\w+)\(/gm)].map((m) => m[1]!);
      });
      return goNames;
    },
  };
}

if (import.meta.main) {
  const root = join(import.meta.dir, "..");
  const repo = fsRepo(root);
  const raw = repo.read("release/endgame-v1.json");
  let findings: Finding[];
  if (raw === null) {
    findings = [{ family: "manifest-incomplete", detail: "release/endgame-v1.json does not exist" }];
  } else {
    try {
      findings = check(JSON.parse(raw), repo);
    } catch (err) {
      findings = [{ family: "manifest-incomplete", detail: `release/endgame-v1.json is not valid JSON: ${err}` }];
    }
  }

  if (findings.length === 0) {
    console.log(`release evidence check passed: ${MILESTONES.length} milestones over ${SPINE.length} spine steps`);
    process.exit(0);
  }

  const order: Family[] = ["manifest-incomplete", "citation-unresolved", "scope-overclaim", "evidence-stale"];
  for (const family of order) {
    const group = findings.filter((f) => f.family === family);
    if (!group.length) continue;
    console.error(`\n${family} (${group.length})`);
    for (const f of group) console.error(`  - ${f.milestone ? `[${f.milestone}] ` : ""}${f.detail}`);
  }
  const counts = order
    .map((f) => [f, findings.filter((x) => x.family === f).length] as const)
    .filter(([, n]) => n > 0)
    .map(([f, n]) => `${f}=${n}`)
    .join(", ");
  console.error(`\nrelease evidence check FAILED: ${findings.length} findings (${counts})`);
  process.exit(1);
}
