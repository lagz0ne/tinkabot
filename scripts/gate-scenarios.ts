// gate:scenarios — outside-in scenario-matrix completeness over the seven
// pinned case families per outside-in surface (quality-v1.md:92,
// tasks/todo.md:237). The matrix lives at substrate/go/scenario-matrix.json
// as { "<surface>": { "<family>": ["TestName", ...] } }; the family list is
// pinned here so the matrix cannot weaken its own gate.

import { readJSON, report, resolves, testFns, type Finding } from "./gate-lib";

const MATRIX = "scenario-matrix.json";

const FAMILIES = [
  "allowed",
  "denied-neighbor",
  "malformed",
  "duplicate",
  "stale",
  "revoked",
  "attributed-failure",
] as const;

const REQUIRED_SURFACES = ["tinkalet-trigger"] as const;

export function check(): Finding[] {
  const findings: Finding[] = [];
  const raw = readJSON(MATRIX);

  if (raw === null || typeof raw !== "object" || Array.isArray(raw)) {
    return [
      {
        family: "scenario-matrix-hole",
        detail: `absent matrix definition: no substrate/go/${MATRIX} declaring the pinned case families (${FAMILIES.join(", ")}) per outside-in surface`,
      },
    ];
  }

  const matrix = raw as Record<string, Record<string, string[]>>;
  const surfaces = Object.keys(matrix);
  if (surfaces.length === 0) {
    findings.push({
      family: "scenario-matrix-hole",
      detail: "matrix declares no outside-in surfaces",
    });
  }
  for (const surface of REQUIRED_SURFACES) {
    if (!(surface in matrix)) {
      findings.push({
        family: "scenario-matrix-hole",
        detail: `required outside-in surface ${surface} is absent`,
      });
    }
  }

  const testNames = new Set(testFns().map((t) => t.name));
  for (const surface of surfaces) {
    const cases = matrix[surface];
    for (const family of FAMILIES) {
      const cited = cases[family] ?? [];
      if (cited.length === 0) {
        findings.push({
          family: "scenario-matrix-hole",
          detail: `surface ${surface}: no case for pinned family ${family}`,
        });
      }
      for (const citation of cited) {
        if (!resolves(citation, testNames)) {
          findings.push({
            family: "measurement-stale",
            detail: `surface ${surface}, family ${family}: "${citation}" does not resolve to a current Go test`,
          });
        }
      }
    }
    for (const key of Object.keys(cases)) {
      if (!(FAMILIES as readonly string[]).includes(key)) {
        findings.push({
          family: "measurement-stale",
          detail: `surface ${surface}: unknown case family ${key}`,
        });
      }
    }
  }
  return findings;
}

if (import.meta.main) report("gate:scenarios", check());
