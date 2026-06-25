/**
 * build.ts — Vite programmatic bundle filter for the builder bundle.
 *
 * Protocol:
 *   stdin  : one JSON line per source change — the stored projection record
 *            {"kind":"material.projection","projectionId":"bundle.builder.src",...,"value":{"files":{"index.html":"...","src/main.ts":"..."}}}
 *   stdout : frames — "Content-Length: <n>\r\n\r\n<body>" where body is single-line JSON effect
 *
 * Hard constraints:
 *   - stdout contains ONLY frames (all logging -> stderr)
 *   - each frame body <= 262144 bytes
 *
 * Chain: source projection -> long-lived Vite warm-rebuild -> artifact frames -> one built-projection frame.
 */

import { build } from "vite";
import { mkdirSync, writeFileSync, readFileSync, rmSync, readdirSync, statSync } from "fs";
import { join, extname, relative } from "path";
import * as readline from "readline";
import * as os from "os";

const MAX_BODY_BYTES = 262144; // 256 KiB

function mediaType(filename: string): string {
  const ext = extname(filename).toLowerCase();
  switch (ext) {
    case ".html": return "text/html";
    case ".js":
    case ".mjs": return "application/javascript";
    case ".css": return "text/css";
    case ".json": return "application/json";
    case ".svg": return "image/svg+xml";
    case ".png": return "image/png";
    case ".ico": return "image/x-icon";
    default: return "application/octet-stream";
  }
}

function emitFrame(obj: Record<string, unknown>): void {
  const body = JSON.stringify(obj);
  const bodyBytes = Buffer.byteLength(body, "utf8");
  if (bodyBytes > MAX_BODY_BYTES) {
    console.error(`[build] WARN: frame for ${obj.artifactName ?? obj.projectionId} is ${bodyBytes} bytes, exceeds ${MAX_BODY_BYTES} — skipping`);
    return;
  }
  const header = `Content-Length: ${bodyBytes}\r\n\r\n`;
  process.stdout.write(header + body);
}

function collectFiles(dir: string, base: string): { relPath: string; absPath: string }[] {
  const results: { relPath: string; absPath: string }[] = [];
  for (const entry of readdirSync(dir)) {
    const abs = join(dir, entry);
    const rel = relative(base, abs);
    if (statSync(abs).isDirectory()) {
      results.push(...collectFiles(abs, base));
    } else {
      results.push({ relPath: rel, absPath: abs });
    }
  }
  return results;
}

async function processFeed(feedIndex: number, inputLine: string): Promise<void> {
  const start = Date.now();
  let parsed: { value?: { files?: Record<string, string> } };
  try {
    parsed = JSON.parse(inputLine);
  } catch (e) {
    console.error(`[build] feed ${feedIndex}: failed to parse input JSON:`, e);
    return;
  }

  const files = parsed?.value?.files;
  if (!files || typeof files !== "object") {
    console.error(`[build] feed ${feedIndex}: missing value.files`);
    return;
  }

  // Write source files to a temp dir
  const realSrcDir = join(os.tmpdir(), `builder-src-${feedIndex}-${Date.now()}`);
  mkdirSync(realSrcDir, { recursive: true });
  const outDir = join(os.tmpdir(), `builder-out-${feedIndex}-${Date.now()}`);
  mkdirSync(outDir, { recursive: true });

  console.error(`[build] feed ${feedIndex}: writing ${Object.keys(files).length} source files to ${realSrcDir}`);

  for (const [relPath, content] of Object.entries(files)) {
    const absPath = join(realSrcDir, relPath);
    mkdirSync(join(absPath, ".."), { recursive: true });
    writeFileSync(absPath, content, "utf8");
  }

  console.error(`[build] feed ${feedIndex}: starting vite build`);

  const buildStart = Date.now();
  try {
    await build({
      root: realSrcDir,
      base: "./",
      logLevel: "silent",
      build: {
        outDir,
        write: true,
        modulePreload: false,
        rollupOptions: {
          output: {
            entryFileNames: "assets/[name].js",
            chunkFileNames: "assets/[name].js",
            assetFileNames: "assets/[name][extname]",
          },
        },
      },
    });
  } catch (e) {
    console.error(`[build] feed ${feedIndex}: vite build error:`, e);
    rmSync(realSrcDir, { recursive: true, force: true });
    rmSync(outDir, { recursive: true, force: true });
    return;
  }

  const buildWallMs = Date.now() - buildStart;
  console.error(`[build] feed ${feedIndex}: build done in ${buildWallMs}ms, reading output`);

  // Read and emit each output file
  const outputFiles = collectFiles(outDir, outDir);
  console.error(`[build] feed ${feedIndex}: emitting ${outputFiles.length} artifact frames`);

  let emitted = 0;
  for (const { relPath, absPath } of outputFiles) {
    const content = readFileSync(absPath, "utf8");
    const artifactName = relPath.replace(/\\/g, "/");
    const mt = mediaType(relPath);
    const bodyBytes = Buffer.byteLength(content, "utf8");
    console.error(`[build] feed ${feedIndex}: artifact ${artifactName} (${mt}) ${bodyBytes} bytes`);

    emitFrame({
      kind: "script.effect",
      effectType: "artifact",
      artifactName,
      artifactRevision: `app.rev.${feedIndex}`,
      mediaType: mt,
      body: content,
    });
    emitted++;
  }

  // Emit one built-projection frame summarizing this rebuild.
  emitFrame({
    kind: "script.effect",
    effectType: "projection",
    projectionId: "built",
    snapshotRevision: `snap-b-${Date.now()}`,
    artifactRevision: `app.rev.${feedIndex}`,
    sequence: Date.now(),
    value: { emitted, ms: buildWallMs },
  });

  console.error(`[build] feed ${feedIndex}: total wall time ${Date.now() - start}ms`);

  // Cleanup temp dirs
  rmSync(realSrcDir, { recursive: true, force: true });
  rmSync(outDir, { recursive: true, force: true });
}

// Main: read stdin line by line
async function main(): Promise<void> {
  console.error("[build] ready, waiting for source change lines on stdin");
  const rl = readline.createInterface({ input: process.stdin, crlfDelay: Infinity });
  let feedIndex = 0;
  for await (const line of rl) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    feedIndex++;
    await processFeed(feedIndex, trimmed);
  }
  console.error(`[build] stdin closed after ${feedIndex} feeds`);
}

main().catch((e) => {
  console.error("[build] fatal:", e);
  process.exit(1);
});
