// gate:fakes — every fake in the Go corpus must appear in the explicit
// allowlist at substrate/go/fakes-allowlist.json with a written narrow
// impossible-to-force-branch justification and the real-NATS proof test that
// validates it; anything un-allowlisted fails (quality-v1.md:90, quality-v1.md:71,
// tasks/todo.md:231).

import { goFiles, lineOf, read, readJSON, report, resolves, testFns, type Finding } from "./gate-lib";

const ALLOWLIST = "fakes-allowlist.json";
const MIN_JUSTIFICATION = 60; // a written justification, not a tag

type Entry = { fake: string; definedIn: string; justification: string; realProof: string };

type Fake = { name: string; definedIn: string; usedIn: string[] };

// A fake is any committed type whose name declares it substitutes a real
// dependency. The real path is embedded NATS (JetStream KV/Object/streams);
// these types exist only to force narrow branches.
function detectFakes(): Fake[] {
  const fakes: Fake[] = [];
  for (const file of goFiles()) {
    const src = read(file);
    for (const m of src.matchAll(/^type ((?:Memory|InMemory|Fake|Mock|Stub)\w*) struct\b/gm)) {
      fakes.push({ name: m[1], definedIn: `${file}:${lineOf(src, m.index)}`, usedIn: [] });
    }
  }
  for (const file of goFiles().filter((f) => f.endsWith("_test.go"))) {
    const src = read(file);
    for (const fake of fakes) {
      for (const m of src.matchAll(new RegExp(String.raw`\b(?:New)?${fake.name}\b`, "g"))) {
        fake.usedIn.push(`${file}:${lineOf(src, m.index)}`);
      }
    }
  }
  return fakes;
}

export function check(): Finding[] {
  const findings: Finding[] = [];
  const fakes = detectFakes();
  const raw = readJSON(ALLOWLIST);
  const entries: Entry[] = Array.isArray(raw) ? raw : [];

  if (raw === null) {
    findings.push({
      family: "fakes-violation",
      detail: `no fakes allowlist at substrate/go/${ALLOWLIST}; every fake below is un-allowlisted`,
    });
  }

  const testNames = new Set(testFns().map((t) => t.name));
  for (const fake of fakes) {
    const entry = entries.find((e) => e.fake === fake.name);
    const used = fake.usedIn.length ? `used in tests at ${fake.usedIn.join(", ")}` : "unused in tests";
    if (!entry) {
      findings.push({
        family: "fakes-violation",
        detail: `un-allowlisted fake ${fake.name} defined at ${fake.definedIn}; ${used}`,
      });
      continue;
    }
    if ((entry.justification ?? "").trim().length < MIN_JUSTIFICATION) {
      findings.push({
        family: "fakes-violation",
        detail: `${fake.name}: allowlist justification missing or too thin to name the impossible-to-force branch`,
      });
    }
    if (!entry.realProof || !resolves(entry.realProof, testNames)) {
      findings.push({
        family: "fakes-violation",
        detail: `${fake.name}: realProof "${entry.realProof ?? ""}" does not resolve to a committed Go test`,
      });
    }
  }

  for (const entry of entries) {
    if (!fakes.some((f) => f.name === entry.fake)) {
      findings.push({
        family: "measurement-stale",
        detail: `allowlist entry ${entry.fake} matches no fake in the corpus`,
      });
    }
  }
  return findings;
}

if (import.meta.main) report("gate:fakes", check());
