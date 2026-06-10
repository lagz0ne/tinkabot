// gate:coverage — inside-out coverage measured per substrate layer against
// declared thresholds in substrate/go/coverage-thresholds.json
// (quality-v1.md:91). A layer is a top-level package directory under
// substrate/go; every layer must declare a threshold and meet it.

import { execSync } from "node:child_process";
import { goDir, goFiles, readJSON, report, type Finding } from "./gate-lib";

const THRESHOLDS = "coverage-thresholds.json";

const layers = () =>
  [...new Set(goFiles().filter((f) => f.includes("/")).map((f) => f.split("/")[0]))].sort();

function measure(layer: string): number | null {
  const out = execSync(`go test ./${layer}/... -count=1 -cover`, {
    cwd: goDir,
    encoding: "utf8",
  });
  const pcts = [...out.matchAll(/coverage: (\d+(?:\.\d+)?)% of statements/g)].map((m) =>
    Number(m[1]),
  );
  return pcts.length ? Math.min(...pcts) : null;
}

export function check(): Finding[] {
  const findings: Finding[] = [];
  const raw = readJSON(THRESHOLDS);
  const declared =
    raw && typeof raw === "object" && !Array.isArray(raw)
      ? (raw as Record<string, number>)
      : {};

  if (raw === null) {
    findings.push({
      family: "coverage-gap",
      detail: `absent per-layer measurement: no declared thresholds at substrate/go/${THRESHOLDS}`,
    });
    for (const layer of layers()) {
      findings.push({
        family: "coverage-gap",
        detail: `layer ${layer} has no declared coverage threshold`,
      });
    }
    return findings;
  }

  for (const layer of layers()) {
    const min = declared[layer];
    if (typeof min !== "number") {
      findings.push({
        family: "coverage-gap",
        detail: `layer ${layer} has no declared coverage threshold`,
      });
      continue;
    }
    const got = measure(layer);
    if (got === null) {
      findings.push({
        family: "measurement-stale",
        detail: `layer ${layer}: go test -cover emitted no coverage figure`,
      });
    } else if (got < min) {
      findings.push({
        family: "coverage-gap",
        detail: `layer ${layer}: coverage ${got}% below declared threshold ${min}%`,
      });
    } else {
      console.log(`layer ${layer}: ${got}% >= ${min}%`);
    }
  }

  for (const key of Object.keys(declared)) {
    if (!layers().includes(key)) {
      findings.push({
        family: "measurement-stale",
        detail: `threshold declared for unknown layer ${key}`,
      });
    }
  }
  return findings;
}

if (import.meta.main) report("gate:coverage", check());
