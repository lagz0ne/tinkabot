import { readFileSync } from "fs";
import { dirname, join } from "path";
import { fileURLToPath } from "url";

const dir = dirname(fileURLToPath(import.meta.url));
const html = readFileSync(join(dir, "..", "ui", "options-site.html"), "utf8");
const provenance = JSON.parse(readFileSync(join(dir, "..", "ui", "options-site.provenance.json"), "utf8"));
const now = Date.now();

const body = JSON.stringify({
  kind: "script.effect",
  effectType: "projection",
  projectionId: "src",
  snapshotRevision: `snap-${now}`,
  artifactRevision: `src.rev.${now}`,
  sequence: now,
  value: {
    files: {
      "index.html": html,
    },
    provenance: {
      ui: "options-site.html",
      author: provenance.authoredBy || "unknown",
    },
  },
});

process.stdout.write(`Content-Length: ${Buffer.byteLength(body, "utf8")}\r\n\r\n${body}`);
