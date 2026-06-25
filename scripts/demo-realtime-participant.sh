#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="${1:-$(mktemp -d /tmp/tinkabot-realtime-demo.XXXXXX)}"
case "$dist" in
  /*) ;;
  *) dist="$root/$dist" ;;
esac

log_step() {
  printf '\n[%s] %s\n' "$(date -u +%H:%M:%S)" "$*"
}

redact_log() {
  sed -E \
    -e 's#nats://[^[:space:]]+#nats://<redacted>#g' \
    -e 's#[^[:space:]]+\.creds#<creds-path>#g' \
    -e 's#tb_items#<item-bucket>#g' \
    -e 's#\$KV#<kv>#g' \
    -e 's#BEGIN NATS#BEGIN <redacted>#g' \
    -e 's#PRIVATE KEY#<redacted-key>#g'
}

fail() {
  echo "demo-realtime-participant: $*" >&2
  if [[ -n "${log:-}" && -f "$log" ]]; then
    sed -n '1,140p' "$log" | redact_log >&2 || true
  fi
  if [[ -n "${err:-}" && -f "$err" ]]; then
    sed -n '1,140p' "$err" | redact_log >&2 || true
  fi
  exit 1
}

tailscale_host() {
  if ! command -v tailscale >/dev/null 2>&1; then
    return 1
  fi
  local host
  host="$(tailscale status --json 2>/dev/null | sed -n 's/^[[:space:]]*"DNSName": "\([^"]*\)".*/\1/p' | head -n 1)"
  host="${host%.}"
  if [[ -z "$host" ]]; then
    host="$(tailscale ip -4 2>/dev/null | sed -n '1p')"
  fi
  [[ -n "$host" ]] || return 1
  printf '%s\n' "$host"
}

tailscale_ip() {
  if ! command -v tailscale >/dev/null 2>&1; then
    return 1
  fi
  local ip
  ip="$(tailscale ip -4 2>/dev/null | sed -n '1p')"
  [[ -n "$ip" ]] || return 1
  printf '%s\n' "$ip"
}

url_host() {
  case "$1" in
    *:*) printf '[%s]' "$1" ;;
    *) printf '%s' "$1" ;;
  esac
}

mkdir -p "$dist"
log_step "build release-shaped package in $dist"
bash "$root/scripts/release-package.sh" "$dist" >/dev/null

archive="$(find "$dist" -maxdepth 1 -name 'tinkabot-v*.tar.gz' | sort | tail -n 1)"
[[ -n "$archive" ]] || fail "release archive missing"

tar -xzf "$archive" -C "$dist"
pkg="${archive%.tar.gz}"
for file in tinkabot tinkalet libexec/tinkabot/nats; do
  [[ -x "$pkg/$file" ]] || fail "$file is not executable"
done

store="$(mktemp -d /tmp/tinkabot-realtime-store.XXXXXX)"
work="$dist/realtime-work"
log="$dist/tinkabot.log"
err="$dist/tinkabot.err"
proof="$dist/realtime-reference-proof.json"
pid=""
forward_pid=""

cleanup() {
  if [[ -n "$forward_pid" ]] && kill -0 "$forward_pid" 2>/dev/null; then
    kill -TERM "$forward_pid" 2>/dev/null || true
    wait "$forward_pid" 2>/dev/null || true
  fi
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
    kill -TERM "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT

log_step "start packaged Tinkabot with scoped participants"
(cd "$pkg" && ./tinkabot \
  --store "$store" \
  --shell 127.0.0.1:0 \
  --participant demo:alice \
  --participant demo:bob \
  >"$log" 2>"$err") &
pid=$!

for _ in {1..300}; do
  if grep -q '^shell  http://127\.0\.0\.1:' "$log" && grep -q '^participant demo bob ' "$log"; then
    break
  fi
  if ! kill -0 "$pid" 2>/dev/null; then
    fail "tinkabot exited before participants were admitted"
  fi
  sleep 0.1
done

shell_url="$(awk '$1 == "shell" { print $2; exit }' "$log")"
[[ -n "$shell_url" ]] || fail "shell URL missing"
shell_port="${shell_url##*:}"
shell_port="${shell_port%%/*}"
local_shell_url="http://127.0.0.1:$shell_port"
public_host="${TINKABOT_DEMO_PUBLIC_HOST:-}"
if [[ -z "$public_host" ]]; then
  public_host="$(tailscale_host || true)"
fi
if [[ -n "$public_host" ]]; then
  public_ip="$(tailscale_ip || true)"
  [[ -n "$public_ip" ]] || fail "Tailscale IP missing"
  command -v socat >/dev/null 2>&1 || fail "socat is required for the Tailscale shell forwarder"
  socat "TCP-LISTEN:$shell_port,bind=$public_ip,fork,reuseaddr" "TCP:127.0.0.1:$shell_port" >/dev/null 2>&1 &
  forward_pid=$!
  public_shell_url="http://$(url_host "$public_host"):$shell_port"
else
  public_shell_url="$local_shell_url"
fi
printf 'shell %s\n' "$public_shell_url"
printf 'local-shell %s\n' "$local_shell_url"

log_step "disable packaged NATS sidecar before profile-driven commands"
mv "$pkg/libexec/tinkabot/nats" "$pkg/libexec/tinkabot/nats.disabled"

log_step "drive realtime participant proof through packaged Tinkalet"
REALTIME_PKG="$pkg" \
REALTIME_STORE="$store" \
REALTIME_WORK="$work" \
REALTIME_PROOF="$proof" \
REALTIME_ACTIONS="${REALTIME_ACTIONS:-30}" \
REALTIME_INTERVAL_MS="${REALTIME_INTERVAL_MS:-20}" \
node --input-type=module <<'JS'
const { execFileSync, spawn } = await import("node:child_process");
const fs = await import("node:fs");
const path = await import("node:path");

const pkg = process.env.REALTIME_PKG;
const store = process.env.REALTIME_STORE;
const work = process.env.REALTIME_WORK;
const proofPath = process.env.REALTIME_PROOF;
const actionCount = Number(process.env.REALTIME_ACTIONS);
const intervalMs = Number(process.env.REALTIME_INTERVAL_MS);
const tinkalet = path.join(pkg, "tinkalet");
const participants = ["alice", "bob"];
const terminalKey = "apps.demo.state.terminal";
const proof = [];

if (!Number.isInteger(actionCount) || actionCount <= 0) {
  throw new Error(`invalid REALTIME_ACTIONS=${process.env.REALTIME_ACTIONS}`);
}
if (!Number.isInteger(intervalMs) || intervalMs <= 0) {
  throw new Error(`invalid REALTIME_INTERVAL_MS=${process.env.REALTIME_INTERVAL_MS}`);
}

for (const who of ["owner", ...participants]) {
  for (const dir of ["home", "cfg", "data"]) {
    fs.mkdirSync(path.join(work, who, dir), { recursive: true, mode: 0o700 });
  }
}

function envFor(who) {
  return {
    HOME: path.join(work, who, "home"),
    TINKALET_CONFIG_DIR: path.join(work, who, "cfg"),
    TINKALET_DATA_DIR: path.join(work, who, "data"),
    PATH: "/nonexistent",
  };
}

function run(who, args) {
  try {
    return execFileSync(tinkalet, args, { encoding: "utf8", env: envFor(who), maxBuffer: 1024 * 1024 * 10 });
  } catch (err) {
    throw new Error(`${who} tinkalet ${args.join(" ")} failed ${err.status}\nstdout=${err.stdout || ""}\nstderr=${err.stderr || ""}`);
  }
}

function runAsync(who, args) {
  return new Promise((resolve, reject) => {
    const child = spawn(tinkalet, args, { env: envFor(who), stdio: ["ignore", "pipe", "pipe"] });
    let stdout = "";
    let stderr = "";
    child.stdout.setEncoding("utf8");
    child.stderr.setEncoding("utf8");
    child.stdout.on("data", (chunk) => {
      stdout += chunk;
    });
    child.stderr.on("data", (chunk) => {
      stderr += chunk;
    });
    child.on("error", reject);
    child.on("close", (code) => {
      if (code !== 0 || stderr !== "") {
        reject(new Error(`${who} tinkalet ${args.join(" ")} failed ${code}\nstdout=${stdout}\nstderr=${stderr}`));
        return;
      }
      resolve(stdout);
    });
  });
}

function deny(who, args, want) {
  try {
    const out = execFileSync(tinkalet, args, { encoding: "utf8", env: envFor(who) });
    throw new Error(`${who} tinkalet ${args.join(" ")} unexpectedly passed: ${out}`);
  } catch (err) {
    if (err.status === undefined) {
      throw err;
    }
    const stderr = err.stderr || "";
    const stdout = err.stdout || "";
    if (err.status !== 1 || stdout !== "" || stderr !== want) {
      throw new Error(`${who} tinkalet ${args.join(" ")} denial drift ${err.status}\nstdout=${stdout}\nstderr=${stderr}\nwant=${want}`);
    }
  }
}

function jsonRun(who, args) {
  return JSON.parse(run(who, args));
}

function jsonLines(out) {
  return out.trim().split(/\n+/).filter(Boolean).map((line) => JSON.parse(line));
}

function setup(who, source, name) {
  expect(`${who} import`, run(who, ["profile", "import", "local", "--store", source, "--name", name]), `profile ${name} imported\n`);
  expect(`${who} use`, run(who, ["profile", "use", name]), `profile ${name} selected\n`);
}

function expect(label, got, want) {
  if (got !== want) {
    throw new Error(`${label}: got ${JSON.stringify(got)}, want ${JSON.stringify(want)}`);
  }
}

function leakCount(text) {
  return ["tb_items", "$KV", "BEGIN NATS", "PRIVATE KEY", "nats://"].filter((token) => text.includes(token)).length;
}

function assertNoLeak(label, text) {
  const count = leakCount(text);
  if (count !== 0) {
    throw new Error(`${label} leaked raw authority details: ${text}`);
  }
}

function actionKey(who, id) {
  return `apps.demo.participants.${who}.actions.${id}`;
}

function assertActionItem(item, who, id, stateKey, baseRevision) {
  expect(`${id} key`, item.key, actionKey(who, id));
  expect(`${id} status`, item.status, "pending");
  expect(`${id} kind`, item.value.kind, "tinkabot.appAction.v1");
  expect(`${id} app`, item.value.appId, "demo");
  expect(`${id} participant`, item.value.participantId, who);
  expect(`${id} action`, item.value.actionId, id);
  expect(`${id} state`, item.value.stateKey, stateKey);
  expect(`${id} base`, item.value.baseRevision, baseRevision);
  assertNoLeak(id, JSON.stringify(item));
}

function assertReceipt(item, key, outcome, status, reason = "") {
  expect(`${key} receipt key`, item.key, `${key}.receipt`);
  expect(`${key} receipt status`, item.status, status);
  expect(`${key} receipt kind`, item.value.kind, "tinkabot.appActionReceipt.v1");
  expect(`${key} receipt action`, item.value.actionKey, key);
  expect(`${key} receipt state`, item.value.stateKey, terminalKey);
  expect(`${key} receipt outcome`, item.value.outcome, outcome);
  expect(`${key} receipt reason`, item.value.reason || "", reason);
  if (!item.value.actionRevision || !item.value.stateRevision) {
    throw new Error(`${key} receipt missing revisions: ${JSON.stringify(item)}`);
  }
  assertNoLeak(`${key} receipt`, JSON.stringify(item));
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

setup("owner", store, "owner");
setup("alice", path.join(store, "participants", "demo", "alice"), "alice");
setup("bob", path.join(store, "participants", "demo", "bob"), "bob");

const rateStates = new Map();
for (const who of participants) {
  const key = `apps.demo.state.rate-${who}`;
  const item = jsonRun("owner", ["item", "create", key, "--value", JSON.stringify({ participant: who, seq: 0 }), "--json"]);
  rateStates.set(who, item);
  proof.push({ step: "create-rate-state", participant: who, key, revision: item.revision });
}

async function submitRateActions() {
  const jobs = [];
  const startedAt = Date.now();
  let firstLaunch = 0;
  let lastLaunch = 0;
  for (let i = 0; i < actionCount; i++) {
    const target = startedAt + i * intervalMs;
    await sleep(Math.max(0, target - Date.now()));
    const launchedAt = Date.now();
    if (firstLaunch === 0) {
      firstLaunch = launchedAt;
    }
    lastLaunch = launchedAt;
    for (const who of participants) {
      const state = rateStates.get(who);
      const id = `rt-${who}-${i}`;
      const payload = JSON.stringify({ participant: who, seq: i });
      jobs.push(runAsync(who, ["action", "submit", id, "--state", state.key, "--base-revision", String(state.revision), "--value", payload, "--json"]).then((out) => {
        const item = JSON.parse(out);
        assertActionItem(item, who, id, state.key, state.revision);
        return { who, id, revision: item.revision, launchedAt };
      }));
    }
  }
  const submitted = await Promise.all(jobs);
  const completedAt = Date.now();
  return { submitted, firstLaunch, lastLaunch, completedAt };
}

function watchParticipant(who, limit) {
  const out = run(who, ["watch", "prefix", `apps.demo.participants.${who}.actions`, "--limit", String(limit), "--timeout", "10s", "--json"]);
  assertNoLeak(`${who} watch`, out);
  return jsonLines(out);
}

function accountRateEvents(who, events) {
  const prefix = `apps.demo.participants.${who}.actions.`;
  const seen = new Set();
  let revisionOrderViolationCount = 0;
  let prev = 0;
  for (const ev of events) {
    if (!ev.key.startsWith(prefix)) {
      throw new Error(`${who} watch saw out-of-scope key ${ev.key}`);
    }
    if (ev.revision <= prev) {
      revisionOrderViolationCount++;
    }
    prev = ev.revision;
    const id = ev.key.slice(prefix.length);
    if (id.startsWith(`rt-${who}-`)) {
      if (id.includes(".") || ev.status !== "pending") {
        throw new Error(`${who} rate event drift: ${JSON.stringify(ev)}`);
      }
      seen.add(id);
      assertNoLeak(`${who} rate event`, JSON.stringify(ev));
    }
  }
  const missing = [];
  for (let i = 0; i < actionCount; i++) {
    const id = `rt-${who}-${i}`;
    if (!seen.has(id)) {
      missing.push(id);
    }
  }
  const observedIds = [...seen].sort((a, b) => Number(a.split("-").pop()) - Number(b.split("-").pop()));
  return { observed: seen.size, observedIds, missing, revisionOrderViolationCount };
}

function terminalPayload(state) {
  return JSON.stringify(state);
}

function getTerminalState() {
  return jsonRun("owner", ["item", "get", terminalKey, "--json"]);
}

function submitTerminal(who, id, baseRevision, payload) {
  const item = jsonRun(who, ["action", "submit", id, "--state", terminalKey, "--base-revision", String(baseRevision), "--value", JSON.stringify(payload), "--json"]);
  assertActionItem(item, who, id, terminalKey, baseRevision);
  return item;
}

function applyTerminal(action, before, next) {
  const receipt = jsonRun("owner", ["action", "apply", action.key, "--value", terminalPayload(next), "--json"]);
  assertReceipt(receipt, action.key, "applied", "resolved");
  const after = getTerminalState();
  if (after.revision <= before.revision || JSON.stringify(after.value) !== terminalPayload(next)) {
    throw new Error(`terminal apply drift ${action.key}: ${JSON.stringify({ before, after, next })}`);
  }
  proof.push({ step: "apply-terminal", action: action.key, stateRevision: after.revision, state: after.value });
  return after;
}

function rejectTerminal(action, before, reason) {
  const receipt = jsonRun("owner", ["action", "reject", action.key, "--reason", reason, "--json"]);
  assertReceipt(receipt, action.key, "rejected", "denied", reason);
  const after = getTerminalState();
  if (after.revision !== before.revision || JSON.stringify(after.value) !== JSON.stringify(before.value)) {
    throw new Error(`terminal reject mutated state ${action.key}: ${JSON.stringify({ before, after })}`);
  }
  proof.push({ step: "reject-terminal", action: action.key, reason, stateRevision: after.revision });
  return after;
}

function accountTerminalReceipts(who, events, wants) {
  const byAction = new Map();
  for (const ev of events) {
    for (const want of wants) {
      if (ev.key === `${actionKey(who, want.id)}.receipt` && (ev.status === "resolved" || ev.status === "denied")) {
        byAction.set(want.id, ev);
      }
    }
  }
  let missing = 0;
  for (const want of wants) {
    const ev = byAction.get(want.id);
    if (!ev) {
      missing++;
      continue;
    }
    expect(`${want.id} terminal receipt status`, ev.status, want.status);
    expect(`${want.id} terminal receipt outcome`, ev.value.outcome, want.outcome);
    expect(`${want.id} terminal receipt reason`, ev.value.reason || "", want.reason || "");
    assertNoLeak(`${want.id} terminal receipt event`, JSON.stringify(ev));
  }
  return missing;
}

const rate = await submitRateActions();
const submitWallMs = Math.max(1, rate.completedAt - rate.firstLaunch);
const participantRateHz = Number((actionCount / (submitWallMs / 1000)).toFixed(2));
if (participantRateHz < 10) {
  throw new Error(`participant submit rate ${participantRateHz}Hz below 10Hz over ${submitWallMs}ms`);
}

let terminal = jsonRun("owner", ["item", "create", terminalKey, "--value", terminalPayload({ phase: "running", progress: { alice: 0, bob: 0 } }), "--json"]);
proof.push({ step: "create-terminal-state", revision: terminal.revision });

terminal = applyTerminal(
  submitTerminal("alice", "term-a1", terminal.revision, { delta: 1 }),
  terminal,
  { phase: "running", progress: { alice: 1, bob: 0 } },
);
terminal = applyTerminal(
  submitTerminal("bob", "term-b1", terminal.revision, { delta: 1 }),
  terminal,
  { phase: "running", progress: { alice: 1, bob: 1 } },
);
terminal = applyTerminal(
  submitTerminal("alice", "term-finish", terminal.revision, { delta: 1, finish: true }),
  terminal,
  { phase: "finished", progress: { alice: 2, bob: 1 }, winner: "alice" },
);
terminal = rejectTerminal(submitTerminal("bob", "term-late", terminal.revision, { delta: 1 }), terminal, "race-finished");

deny("alice", ["item", "create", "apps.demo.participants.alice.actions.raw", "--value", JSON.stringify({ bypass: true })], "item apps.demo.participants.alice.actions.raw denied create: denied-scope\n");
deny("alice", ["watch", "prefix", "apps.demo.participants.bob.actions", "--limit", "1", "--timeout", "200ms", "--json"], "watch apps.demo.participants.bob.actions denied prefix: denied-scope\n");

// Applied actions produce action + pending receipt + resolved receipt.
// Rejected actions produce action + denied receipt.
const aliceTerminalEvents = 6;
const bobTerminalEvents = 5;
const aliceEvents = watchParticipant("alice", actionCount + aliceTerminalEvents);
const bobEvents = watchParticipant("bob", actionCount + bobTerminalEvents);
const aliceRate = accountRateEvents("alice", aliceEvents);
const bobRate = accountRateEvents("bob", bobEvents);
const terminalMissing =
  accountTerminalReceipts("alice", aliceEvents, [
    { id: "term-a1", status: "resolved", outcome: "applied" },
    { id: "term-finish", status: "resolved", outcome: "applied" },
  ]) +
  accountTerminalReceipts("bob", bobEvents, [
    { id: "term-b1", status: "resolved", outcome: "applied" },
    { id: "term-late", status: "denied", outcome: "rejected", reason: "race-finished" },
  ]);

const expectedActions = actionCount * participants.length;
const observedActions = aliceRate.observed + bobRate.observed;
const revisionGapCount =
  aliceRate.missing.length +
  bobRate.missing.length +
  aliceRate.revisionOrderViolationCount +
  bobRate.revisionOrderViolationCount;
const final = getTerminalState();
const terminalEventLoss = terminalMissing + (final.value.phase === "finished" && final.value.winner === "alice" ? 0 : 1);
const rawAuthorityLeakCount =
  leakCount(JSON.stringify(rate.submitted)) +
  leakCount(JSON.stringify(aliceEvents)) +
  leakCount(JSON.stringify(bobEvents)) +
  leakCount(JSON.stringify(final));

if (observedActions !== expectedActions || revisionGapCount !== 0 || terminalEventLoss !== 0 || rawAuthorityLeakCount !== 0) {
  throw new Error(JSON.stringify({ expectedActions, observedActions, revisionGapCount, terminalEventLoss, rawAuthorityLeakCount }, null, 2));
}

const body = {
  participants_started: participants.length,
  expected_actions: expectedActions,
  observed_actions: observedActions,
  revision_gap_count: revisionGapCount,
  revision_gap_count_kind: "missing expected action ids or non-increasing own-watch revisions",
  observed_action_ids: {
    alice: aliceRate.observedIds,
    bob: bobRate.observedIds,
  },
  participant_rate_hz_per_participant: participantRateHz,
  action_count_per_participant: actionCount,
  interval_ms: intervalMs,
  submit_wall_ms: submitWallMs,
  terminal_event_loss: terminalEventLoss,
  authority_violation_count: 0,
  raw_authority_leak_count: rawAuthorityLeakCount,
  winner: final.value.winner,
  final_state_key: terminalKey,
  final_state_revision: final.revision,
  late_action_rejection: "race-finished",
  proof,
};
fs.writeFileSync(proofPath, `${JSON.stringify(body, null, 2)}\n`);

console.log(`realtime actions ${observedActions}/${expectedActions}`);
console.log(`realtime per-participant rate ${participantRateHz}Hz`);
console.log(`realtime terminal winner ${body.winner} rev ${body.final_state_revision}`);
JS

[[ -s "$proof" ]] || fail "realtime proof was not written"
grep -q '"observed_actions"' "$proof" || fail "realtime proof missing action accounting"
grep -q '"revision_gap_count": 0' "$proof" || fail "realtime proof recorded revision gaps"
grep -q '"terminal_event_loss": 0' "$proof" || fail "realtime proof recorded terminal event loss"
grep -q '"authority_violation_count": 0' "$proof" || fail "realtime proof recorded authority violation"
grep -q '"raw_authority_leak_count": 0' "$proof" || fail "realtime proof recorded raw authority leak"
grep -q '"winner": "alice"' "$proof" || fail "realtime proof did not record winner"

log_step "realtime proof written to $proof"
printf 'proof %s\n' "$proof"
