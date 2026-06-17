// gate:manual (docs/matched-abstraction/plan/quality-v1.md:93): the manual's
// behavior commands with documented outcomes run verbatim against the running
// binary and must produce those outcomes. Outcome oracles are output text,
// never exit codes — nats CLI v0.3.0 exits 0 on permission errors
// (docs/matched-abstraction/plan/endgame-app.md:177).

import { execFileSync } from "node:child_process";
import { mkdirSync, mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { root, report, type Finding } from "./gate-lib";

export type Pair = { cmd: string; expected: string };

// Command/outcome pairs: inside a bash block, a command line immediately
// followed by a `# -> outcome` line, continued by plain comment lines until a
// blank line. Commands without a documented outcome carry no verbatim claim
// here; their creds-mode proof is TestBinaryManual (consumed slice-4 input).
export function pairs(manual: string): Pair[] {
  const out: Pair[] = [];
  for (const [, block] of manual.matchAll(/```bash\n([\s\S]*?)```/g)) {
    const lines = block!.split("\n").map((l) => l.trim());
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i]!;
      if (!line || line.startsWith("#") || !lines[i + 1]?.startsWith("# ->")) continue;
      let expected = lines[i + 1]!.slice(4).trim();
      let j = i + 2;
      while (lines[j]?.startsWith("#") && !lines[j]!.startsWith("# ->")) {
        expected += " " + lines[j]!.slice(1).trim();
        j++;
      }
      out.push({ cmd: line, expected });
      i = j - 1;
    }
  }
  return out;
}

const strip = (s: string) => s.replace(/\s+/g, "");

// Documented outcomes elide volatile values with `...`; the stable anchors
// between elisions (split again at commas, so anchors need not be contiguous
// around elided fields) must all appear in the live output.
export function anchors(expected: string): string[] {
  return strip(expected)
    .split("...")
    .flatMap((part) => part.split(","))
    .filter((a) => a.length > 1);
}

export function check(manual: string, run: (cmd: string, expected?: string) => string): Finding[] {
  const ps = pairs(manual);
  if (ps.length === 0) {
    return [{ family: "measurement-stale", detail: "manual documents no command/outcome pairs to verify" }];
  }
  const out: Finding[] = [];
  for (const { cmd, expected } of ps) {
    const live = run(cmd, expected);
    const missing = anchors(expected).filter((a) => !strip(live).includes(a));
    if (missing.length) {
      out.push({
        family: "manual-divergence",
        detail: `\`${cmd}\` live output diverges from the documented outcome: missing ${missing
          .map((m) => JSON.stringify(m))
          .join(", ")} (live: ${live.trim() || "<empty>"})`,
      });
    }
  }
  return out;
}

// Served wiring quoted from substrate/go/tinkabot/tinkabot.go wiring(); drift
// surfaces as a loud divergence finding, never a silent pass.
const MATERIAL_BUCKET = "tb_material";
const ARTIFACT_BUCKET = "tb_artifacts";
const SCRIPT_BUCKET = "tb_scripts";
const SCRIPT_KEY = "scripts.app.main";
const SCRIPT_REVISION = 1;
const goDir = join(root, "substrate/go");
const toolDir = join(root, "tools/natscli");
const sh = (s: string) => `'${s.replaceAll("'", "'\\''")}'`;

const frame = (body: string) => `Content-Length: ${body.length}\r\n\r\n${body}`;

// The script behind the manual's documented observations ("Observing
// results"): projection `main` at sequence 9 with artifact.rev.7 and
// {"title":"from-script"}, plus the artifact/main.js manifest and body.
const effects =
  frame(
    JSON.stringify({
      kind: "script.effect",
      effectType: "projection",
      projectionId: "main",
      snapshotRevision: "snap-rel-001",
      artifactRevision: "artifact.rev.7",
      sequence: 9,
      value: { title: "from-script" },
    }),
  ) +
  frame(
    JSON.stringify({
      kind: "script.effect",
      effectType: "artifact",
      artifactName: "artifact/main.js",
      artifactRevision: "artifact.rev.7",
      mediaType: "application/javascript",
      body: "export default 1",
    }),
  );

if (import.meta.main) {
  const manual = readFileSync(join(root, "docs/manual/v1.md"), "utf8");
  const tmp = mkdtempSync(join(tmpdir(), "gate-manual-"));
  let proc: Bun.Subprocess<"ignore", "pipe", "pipe"> | null = null;
  let findings: Finding[];
  try {
    // Known wart (docs/matched-abstraction/task/tinkabot-binary.md:73): the
    // verbatim `go build ./cmd/tinkabot` output name collides with the
    // package directory, so the gate builds with -o.
    const bin = join(tmp, "tinkabot-bin");
    execFileSync("go", ["build", "-o", bin, "./cmd/tinkabot"], { cwd: goDir });
    const nats = execFileSync("go", ["tool", "-n", "nats"], { cwd: toolDir, encoding: "utf8" }).trim();

    proc = Bun.spawn([bin, "--store", join(tmp, "store"), "--shell", "127.0.0.1:0"], {
      stdout: "pipe",
      stderr: "pipe",
    });
    const stdout = proc.stdout.getReader();
    const deadline = Date.now() + 15000;
    let posture = "";
    while ((posture.match(/^creds {2}/gm) ?? []).length < 3) {
      if (Date.now() > deadline) throw new Error(`binary never printed its posture:\n${posture}`);
      const { value, done } = await stdout.read();
      if (done) throw new Error(`binary exited before printing its posture:\n${posture}`);
      posture += new TextDecoder().decode(value);
    }
    const clientURL = posture.match(/^nats {3}(\S+)/m)![1]!;
    const creds = (role: string) => posture.match(new RegExp(`^creds {2}(\\S*${role}\\.creds)`, "m"))![1]!;

    // Author flow ("Defining a script"): land the script record the manual's
    // trigger activates, emitting the documented effects. The work dir is a
    // dedicated subdir because cleanup "workdir.delete" removes the cwd.
    const work = join(tmp, "work");
    mkdirSync(work);
    const record = {
      kind: "script.record",
      scriptKey: SCRIPT_KEY,
      scriptRevision: SCRIPT_REVISION,
      desc: "Render the main material projection.",
      process: {
        command: "/bin/sh",
        args: ["-c", `printf '%s' '${effects}'`],
        cwd: work,
        rpc: "framed_stdio",
        timeoutMs: 2000,
        resource: { cpuMillis: 100, memoryMB: 64 },
        kill: "process.kill",
        cleanup: "workdir.delete",
        identity: "principal.script.001",
      },
    };
    const kvKey = "s." + Buffer.from(SCRIPT_KEY).toString("base64url");
    execFileSync(nats, [
      "--no-context",
      "--server",
      clientURL,
      "--creds",
      creds("author"),
      "--timeout",
      "2s",
      "kv",
      "put",
      SCRIPT_BUCKET,
      kvKey,
      JSON.stringify(record),
    ]);

    // Each documented pair runs verbatim under the manual's connection
    // preamble ("Connecting with the nats CLI"). Role follows the touched
    // surface: script bucket -> author, material/artifact buckets -> observer,
    // anything else -> caller. Pairs whose anchors are not yet all present
    // retry briefly because materialization is asynchronous behind the
    // accepted reply.
    const run = (cmd: string, expected = "") => {
      const observes = /\$MATERIAL_BUCKET|\$ARTIFACT_BUCKET/.test(cmd);
      const authors = /\$SCRIPT_BUCKET/.test(cmd);
      const line = cmd.replace(/^nats /, `${sh(nats)} --no-context --server "$CLIENT_URL" --creds "$CREDS" --timeout 2s `);
      const env = {
        ...process.env,
        CLIENT_URL: clientURL,
        CREDS: creds(authors ? "author" : observes ? "observer" : "caller"),
        MATERIAL_BUCKET,
        ARTIFACT_BUCKET,
        SCRIPT_BUCKET,
      };
      const want = anchors(expected);
      const until = Date.now() + 5000;
      while (true) {
        const p = Bun.spawnSync(["sh", "-c", line], { env, cwd: root });
        const out = p.stdout.toString() + p.stderr.toString();
        if (want.every((a) => strip(out).includes(a)) || Date.now() > until) return out;
        Bun.sleepSync(100);
      }
    };
    findings = check(manual, run);
  } finally {
    proc?.kill();
    rmSync(tmp, { recursive: true, force: true });
  }
  report("gate:manual", findings);
}
