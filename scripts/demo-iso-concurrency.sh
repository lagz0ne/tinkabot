#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="${1:-$(mktemp -d /tmp/tinkabot-iso-concurrency.XXXXXX)}"
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
  echo "demo-iso-concurrency: $*" >&2
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

stop_forward() {
  if [[ -n "${forward_pid:-}" ]] && kill -0 "$forward_pid" 2>/dev/null; then
    kill -TERM "$forward_pid" 2>/dev/null || true
    wait "$forward_pid" 2>/dev/null || true
  fi
  forward_pid=""
}

publish_shell() {
  local shell_url="$1"
  local shell_port="${shell_url##*:}"
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
    stop_forward
    socat "TCP-LISTEN:$shell_port,bind=$public_ip,fork,reuseaddr" "TCP:127.0.0.1:$shell_port" >/dev/null 2>&1 &
    forward_pid=$!
    public_shell_url="http://$(url_host "$public_host"):$shell_port"
  else
    public_shell_url="$local_shell_url"
  fi
}

wait_for_shell() {
  local file="$1"
  for _ in {1..300}; do
    if grep -q '^shell  http://127\.0\.0\.1:' "$file"; then
      return 0
    fi
    if ! kill -0 "$pid" 2>/dev/null; then
      fail "tinkabot exited before shell posture"
    fi
    sleep 0.1
  done
  fail "shell URL missing"
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

store="$(mktemp -d /tmp/tinkabot-iso-concurrency-store.XXXXXX)"
work="$dist/iso-concurrency-work"
log="$dist/tinkabot.log"
err="$dist/tinkabot.err"
phase1="$dist/iso-concurrency-phase1.json"
proof="$dist/iso-concurrency-proof.json"
pid=""
forward_pid=""

cleanup() {
  stop_forward
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
    kill -TERM "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT

log_step "start packaged Tinkabot with two app scopes and four participants"
(cd "$pkg" && ./tinkabot \
  --store "$store" \
  --shell 127.0.0.1:0 \
  --participant demo:alice \
  --participant demo:bob \
  --participant other:alice \
  --participant other:bob \
  >"$log" 2>"$err") &
pid=$!

for _ in {1..300}; do
  if grep -q '^shell  http://127\.0\.0\.1:' "$log" && grep -q '^participant other bob ' "$log"; then
    break
  fi
  if ! kill -0 "$pid" 2>/dev/null; then
    fail "tinkabot exited before participants were admitted"
  fi
  sleep 0.1
done

shell_url="$(awk '$1 == "shell" { print $2; exit }' "$log")"
[[ -n "$shell_url" ]] || fail "shell URL missing"
publish_shell "$shell_url"
initial_public_shell_url="$public_shell_url"
printf 'shell %s\n' "$initial_public_shell_url"
printf 'local-shell %s\n' "$local_shell_url"

log_step "disable packaged NATS sidecar before profile-driven commands"
mv "$pkg/libexec/tinkabot/nats" "$pkg/libexec/tinkabot/nats.disabled"

log_step "drive pre-restart concurrent activity through packaged Tinkalet"
ISO_PKG="$pkg" \
ISO_STORE="$store" \
ISO_WORK="$work" \
ISO_PHASE1="$phase1" \
ISO_ACTIONS_PRE="${ISO_ACTIONS_PRE:-3}" \
ISO_INTERVAL_MS="${ISO_INTERVAL_MS:-10}" \
ISO_SHELL_URL="$initial_public_shell_url" \
node --input-type=module <<'JS'
const { execFileSync, spawn } = await import("node:child_process");
const fs = await import("node:fs");
const path = await import("node:path");

const pkg = process.env.ISO_PKG;
const store = process.env.ISO_STORE;
const work = process.env.ISO_WORK;
const phasePath = process.env.ISO_PHASE1;
const preCount = Number(process.env.ISO_ACTIONS_PRE);
const intervalMs = Number(process.env.ISO_INTERVAL_MS);
const tinkalet = path.join(pkg, "tinkalet");
const parts = [
  { key: "demo-alice", app: "demo", id: "alice" },
  { key: "demo-bob", app: "demo", id: "bob" },
  { key: "other-alice", app: "other", id: "alice" },
  { key: "other-bob", app: "other", id: "bob" },
];

if (!Number.isInteger(preCount) || preCount <= 0) throw new Error(`invalid ISO_ACTIONS_PRE=${process.env.ISO_ACTIONS_PRE}`);
if (!Number.isInteger(intervalMs) || intervalMs <= 0) throw new Error(`invalid ISO_INTERVAL_MS=${process.env.ISO_INTERVAL_MS}`);

for (const who of ["owner", ...parts.map((p) => p.key)]) {
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

function runAsync(who, args, launchedAt) {
  return new Promise((resolve, reject) => {
    const child = spawn(tinkalet, args, { env: envFor(who), stdio: ["ignore", "pipe", "pipe"] });
    let stdout = "";
    let stderr = "";
    child.stdout.setEncoding("utf8");
    child.stderr.setEncoding("utf8");
    child.stdout.on("data", (chunk) => { stdout += chunk; });
    child.stderr.on("data", (chunk) => { stderr += chunk; });
    child.on("error", reject);
    child.on("close", (code) => {
      const completedAt = Date.now();
      if (code !== 0 || stderr !== "") {
        reject(new Error(`${who} tinkalet ${args.join(" ")} failed ${code}\nstdout=${stdout}\nstderr=${stderr}`));
        return;
      }
      resolve({ stdout, latencyMs: completedAt - launchedAt, completedAt });
    });
  });
}

function deny(who, args, want) {
  try {
    const out = execFileSync(tinkalet, args, { encoding: "utf8", env: envFor(who) });
    throw new Error(`${who} tinkalet ${args.join(" ")} unexpectedly passed: ${out}`);
  } catch (err) {
    if (err.status === undefined) throw err;
    const stderr = err.stderr || "";
    const stdout = err.stdout || "";
    if (err.status !== 1 || stdout !== "" || stderr !== want) {
      throw new Error(`${who} tinkalet ${args.join(" ")} denial drift ${err.status}\nstdout=${stdout}\nstderr=${stderr}\nwant=${want}`);
    }
    assertNoLeak(`${who} denial`, stdout + stderr);
  }
}

function jsonRun(who, args) {
  const out = run(who, args);
  assertNoLeak(`${who} ${args.join(" ")}`, out);
  return JSON.parse(out);
}

function expect(label, got, want) {
  if (got !== want) throw new Error(`${label}: got ${JSON.stringify(got)}, want ${JSON.stringify(want)}`);
}

function setup(who, source, name) {
  expect(`${who} import`, run(who, ["profile", "import", "local", "--store", source, "--name", name]), `profile ${name} imported\n`);
  expect(`${who} use`, run(who, ["profile", "use", name]), `profile ${name} selected\n`);
}

function leakCount(text) {
  const lower = text.toLowerCase();
  return ["tb_items", "$kv", "$js.api", "begin nats", "private key", ".creds", "credential", "jwt", "nkey", "bearer", "token", "nats://", "tb.app.", "tb.bundle."].filter((token) => lower.includes(token)).length;
}

function assertNoLeak(label, text) {
  const count = leakCount(text);
  if (count !== 0) throw new Error(`${label} leaked raw authority details: ${text}`);
}

function actionID(p, i) {
  return `iso-${p.app}-${p.id}-${i}`;
}

function stateKey(p) {
  return `apps.${p.app}.state.rate-${p.id}`;
}

function actionPrefix(p) {
  return `apps.${p.app}.participants.${p.id}.actions`;
}

function actionKey(p, i) {
  return `${actionPrefix(p)}.${actionID(p, i)}`;
}

function payload(p, i) {
  return JSON.stringify({ app: p.app, participant: p.id, seq: i });
}

function cursor(p) {
  return `iso-${p.app}-${p.id}`;
}

function jsonLines(out) {
  return out.trim().split(/\n+/).filter(Boolean).map((line) => JSON.parse(line));
}

function quantiles(values) {
  const nums = [...values].sort((a, b) => a - b);
  const pick = (q) => nums.length === 0 ? 0 : nums[Math.min(nums.length - 1, Math.ceil(q * nums.length) - 1)];
  return { min: nums[0] || 0, p50: pick(0.5), p95: pick(0.95), p99: pick(0.99), max: nums[nums.length - 1] || 0 };
}

function assertActionItem(item, p, i, state) {
  expect(`${p.key} key ${i}`, item.key, actionKey(p, i));
  expect(`${p.key} status ${i}`, item.status, "pending");
  expect(`${p.key} kind ${i}`, item.value.kind, "tinkabot.appAction.v1");
  expect(`${p.key} app ${i}`, item.value.appId, p.app);
  expect(`${p.key} participant ${i}`, item.value.participantId, p.id);
  expect(`${p.key} action ${i}`, item.value.actionId, actionID(p, i));
  expect(`${p.key} state ${i}`, item.value.stateKey, state.key);
  expect(`${p.key} base ${i}`, item.value.baseRevision, state.revision);
  expect(`${p.key} payload ${i}`, JSON.stringify(item.value.payload), payload(p, i));
  assertNoLeak(`${p.key} action item ${i}`, JSON.stringify(item));
}

async function submitBatch(states, start, count) {
  const jobs = [];
  const latencies = [];
  const launchedAtStart = Date.now();
  let completedAt = launchedAtStart;
  for (let i = start; i < start + count; i++) {
    const target = launchedAtStart + (i - start) * intervalMs;
    await new Promise((resolve) => setTimeout(resolve, Math.max(0, target - Date.now())));
    const launchedAt = Date.now();
    for (const p of parts) {
      const state = states[p.key];
      jobs.push(runAsync(p.key, [
        "action", "submit", actionID(p, i),
        "--state", state.key,
        "--base-revision", String(state.revision),
        "--value", payload(p, i),
        "--json",
      ], launchedAt).then((res) => {
        completedAt = Math.max(completedAt, res.completedAt);
        latencies.push(res.latencyMs);
        const item = JSON.parse(res.stdout);
        assertActionItem(item, p, i, state);
        return { scope: p.key, action: actionID(p, i), revision: item.revision, latencyMs: res.latencyMs };
      }));
    }
  }
  const submitted = await Promise.all(jobs);
  return {
    submitted,
    count_per_participant: count,
    wall_ms: Math.max(1, completedAt - launchedAtStart),
    participant_rate_hz: Number((count / (Math.max(1, completedAt - launchedAtStart) / 1000)).toFixed(2)),
    latency_ms: quantiles(latencies),
    latencies,
  };
}

function watchReplay(p, start, count) {
  const startedAt = Date.now();
  const out = run(p.key, [
    "watch", "prefix", actionPrefix(p),
    "--cursor", cursor(p),
    "--limit", String(count),
    "--timeout", "5s",
    "--json",
  ]);
  const elapsed = Date.now() - startedAt;
  assertNoLeak(`${p.key} watch`, out);
  const events = jsonLines(out);
  let gaps = events.length === count ? 0 : Math.abs(count - events.length);
  const prefix = `${actionPrefix(p)}.`;
  const wantIds = new Set(Array.from({ length: count }, (_, i) => actionID(p, start + i)));
  const seen = new Set();
  let prev = 0;
  for (const event of events) {
    if (!event.key.startsWith(prefix) || event.status !== "pending" || event.source !== "replay") gaps++;
    const id = event.key.startsWith(prefix) ? event.key.slice(prefix.length) : "";
    if (!wantIds.has(id) || seen.has(id)) {
      gaps++;
    } else {
      seen.add(id);
    }
    if (event.revision <= prev) gaps++;
    prev = event.revision;
    assertNoLeak(`${p.key} event`, JSON.stringify(event));
  }
  for (const id of wantIds) {
    if (!seen.has(id)) gaps++;
  }
  return { scope: p.key, elapsed_ms: elapsed, gaps, revisions: events.map((event) => event.revision), ids: events.map((event) => event.key.split(".").pop()) };
}

function crossDenials(states) {
  let denials = 0;
  for (const p of parts) {
    const otherApp = p.app === "demo" ? "other" : "demo";
    const other = parts.find((candidate) => candidate.app === otherApp && candidate.id === p.id);
    const neighbor = p.id === "alice" ? "bob" : "alice";
    deny(p.key, ["item", "get", states[other.key].key], `item ${states[other.key].key} denied get: denied-scope\n`);
    denials++;
    deny(p.key, ["watch", "prefix", `apps.${otherApp}.state`, "--limit", "1", "--timeout", "200ms", "--json"], `watch apps.${otherApp}.state denied prefix: denied-scope\n`);
    denials++;
    deny(p.key, ["watch", "prefix", `apps.${p.app}.participants.${neighbor}.actions`, "--limit", "1", "--timeout", "200ms", "--json"], `watch apps.${p.app}.participants.${neighbor}.actions denied prefix: denied-scope\n`);
    denials++;
    deny(p.key, ["action", "submit", `iso-cross-${p.key}`, "--state", states[other.key].key, "--base-revision", String(states[other.key].revision), "--value", JSON.stringify({ cross: true })], `action iso-cross-${p.key} denied submit: malformed-action\n`);
    denials++;
    deny(p.key, ["item", "create", `${actionPrefix(p)}.raw`, "--value", JSON.stringify({ bypass: true })], `item ${actionPrefix(p)}.raw denied create: denied-scope\n`);
    denials++;
    deny(p.key, ["trigger", "bundle.clock.tick", "--request-id", `iso-${p.key}`], `profile ${p.key} denied bundle.clock.tick: denied-scope\n`);
    denials++;
  }
  return denials;
}

setup("owner", store, "owner");
for (const p of parts) {
  setup(p.key, path.join(store, "participants", p.app, p.id), p.key);
}

const states = {};
for (const p of parts) {
  states[p.key] = jsonRun("owner", ["item", "create", stateKey(p), "--value", JSON.stringify({ app: p.app, participant: p.id, seq: 0 }), "--json"]);
}

const pre = await submitBatch(states, 0, preCount);
const seed = parts.map((p) => watchReplay(p, 0, preCount));
const authorityDenials = crossDenials(states);
const revisionGapCount = seed.reduce((sum, result) => sum + result.gaps, 0);
if (revisionGapCount !== 0) throw new Error(`seed replay gaps: ${JSON.stringify(seed)}`);

const body = {
  shell_url: process.env.ISO_SHELL_URL,
  apps: ["demo", "other"],
  participants: parts,
  states,
  pre,
  seed,
  authority_denials: authorityDenials,
  raw_authority_leak_count: leakCount(JSON.stringify({ pre, seed })),
};
fs.writeFileSync(phasePath, `${JSON.stringify(body, null, 2)}\n`);
console.log(`iso pre-restart actions ${pre.submitted.length} rate ${pre.participant_rate_hz}Hz per participant`);
console.log(`iso pre-restart denials ${authorityDenials}`);
JS

[[ -s "$phase1" ]] || fail "phase1 proof was not written"

log_step "restart Tinkabot on the same store"
kill -TERM "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
pid=""
restart_log="$dist/tinkabot-restart.log"
restart_err="$dist/tinkabot-restart.err"
log="$restart_log"
err="$restart_err"
(cd "$pkg" && PATH="/usr/bin:/bin" ./tinkabot \
  --store "$store" \
  --shell 127.0.0.1:0 \
  >"$restart_log" 2>"$restart_err") &
pid=$!
wait_for_shell "$restart_log"

restart_shell_url="$(awk '$1 == "shell" { print $2; exit }' "$restart_log")"
[[ -n "$restart_shell_url" ]] || fail "restart shell URL missing"
publish_shell "$restart_shell_url"
restart_public_shell_url="$public_shell_url"
printf 'restart-shell %s\n' "$restart_public_shell_url"
printf 'restart-local-shell %s\n' "$local_shell_url"

log_step "drive post-restart reconnect/catch-up through packaged Tinkalet"
ISO_PKG="$pkg" \
ISO_STORE="$store" \
ISO_WORK="$work" \
ISO_PHASE1="$phase1" \
ISO_PROOF="$proof" \
ISO_ACTIONS_POST="${ISO_ACTIONS_POST:-3}" \
ISO_INTERVAL_MS="${ISO_INTERVAL_MS:-10}" \
ISO_RESTART_SHELL_URL="$restart_public_shell_url" \
node --input-type=module <<'JS'
const { execFileSync, spawn } = await import("node:child_process");
const fs = await import("node:fs");
const path = await import("node:path");

const pkg = process.env.ISO_PKG;
const store = process.env.ISO_STORE;
const work = process.env.ISO_WORK;
const phase = JSON.parse(fs.readFileSync(process.env.ISO_PHASE1, "utf8"));
const proofPath = process.env.ISO_PROOF;
const postCount = Number(process.env.ISO_ACTIONS_POST);
const intervalMs = Number(process.env.ISO_INTERVAL_MS);
const tinkalet = path.join(pkg, "tinkalet");
const parts = phase.participants;
if (!Number.isInteger(postCount) || postCount <= 0) throw new Error(`invalid ISO_ACTIONS_POST=${process.env.ISO_ACTIONS_POST}`);

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

function runAsync(who, args, launchedAt) {
  return new Promise((resolve, reject) => {
    const child = spawn(tinkalet, args, { env: envFor(who), stdio: ["ignore", "pipe", "pipe"] });
    let stdout = "";
    let stderr = "";
    child.stdout.setEncoding("utf8");
    child.stderr.setEncoding("utf8");
    child.stdout.on("data", (chunk) => { stdout += chunk; });
    child.stderr.on("data", (chunk) => { stderr += chunk; });
    child.on("error", reject);
    child.on("close", (code) => {
      const completedAt = Date.now();
      if (code !== 0 || stderr !== "") {
        reject(new Error(`${who} tinkalet ${args.join(" ")} failed ${code}\nstdout=${stdout}\nstderr=${stderr}`));
        return;
      }
      resolve({ stdout, latencyMs: completedAt - launchedAt, completedAt });
    });
  });
}

function deny(who, args, want) {
  try {
    const out = execFileSync(tinkalet, args, { encoding: "utf8", env: envFor(who) });
    throw new Error(`${who} tinkalet ${args.join(" ")} unexpectedly passed: ${out}`);
  } catch (err) {
    if (err.status === undefined) throw err;
    const stderr = err.stderr || "";
    const stdout = err.stdout || "";
    if (err.status !== 1 || stdout !== "" || stderr !== want) {
      throw new Error(`${who} tinkalet ${args.join(" ")} denial drift ${err.status}\nstdout=${stdout}\nstderr=${stderr}\nwant=${want}`);
    }
    assertNoLeak(`${who} denial`, stdout + stderr);
  }
}

function expect(label, got, want) {
  if (got !== want) throw new Error(`${label}: got ${JSON.stringify(got)}, want ${JSON.stringify(want)}`);
}

function setup(who, source, name) {
  expect(`${who} import`, run(who, ["profile", "import", "local", "--store", source, "--name", name]), `profile ${name} imported\n`);
  expect(`${who} use`, run(who, ["profile", "use", name]), `profile ${name} selected\n`);
}

function leakCount(text) {
  const lower = text.toLowerCase();
  return ["tb_items", "$kv", "$js.api", "begin nats", "private key", ".creds", "credential", "jwt", "nkey", "bearer", "token", "nats://", "tb.app.", "tb.bundle."].filter((token) => lower.includes(token)).length;
}

function assertNoLeak(label, text) {
  const count = leakCount(text);
  if (count !== 0) throw new Error(`${label} leaked raw authority details: ${text}`);
}

function actionID(p, i) {
  return `iso-${p.app}-${p.id}-${i}`;
}

function actionPrefix(p) {
  return `apps.${p.app}.participants.${p.id}.actions`;
}

function actionKey(p, i) {
  return `${actionPrefix(p)}.${actionID(p, i)}`;
}

function payload(p, i) {
  return JSON.stringify({ app: p.app, participant: p.id, seq: i });
}

function cursor(p) {
  return `iso-${p.app}-${p.id}`;
}

function jsonLines(out) {
  return out.trim().split(/\n+/).filter(Boolean).map((line) => JSON.parse(line));
}

function quantiles(values) {
  const nums = [...values].sort((a, b) => a - b);
  const pick = (q) => nums.length === 0 ? 0 : nums[Math.min(nums.length - 1, Math.ceil(q * nums.length) - 1)];
  return { min: nums[0] || 0, p50: pick(0.5), p95: pick(0.95), p99: pick(0.99), max: nums[nums.length - 1] || 0 };
}

function assertActionItem(item, p, i, state) {
  expect(`${p.key} key ${i}`, item.key, actionKey(p, i));
  expect(`${p.key} status ${i}`, item.status, "pending");
  expect(`${p.key} kind ${i}`, item.value.kind, "tinkabot.appAction.v1");
  expect(`${p.key} app ${i}`, item.value.appId, p.app);
  expect(`${p.key} participant ${i}`, item.value.participantId, p.id);
  expect(`${p.key} action ${i}`, item.value.actionId, actionID(p, i));
  expect(`${p.key} state ${i}`, item.value.stateKey, state.key);
  expect(`${p.key} base ${i}`, item.value.baseRevision, state.revision);
  expect(`${p.key} payload ${i}`, JSON.stringify(item.value.payload), payload(p, i));
  assertNoLeak(`${p.key} action item ${i}`, JSON.stringify(item));
}

async function submitBatch(states, start, count) {
  const jobs = [];
  const latencies = [];
  const launchedAtStart = Date.now();
  let completedAt = launchedAtStart;
  for (let i = start; i < start + count; i++) {
    const target = launchedAtStart + (i - start) * intervalMs;
    await new Promise((resolve) => setTimeout(resolve, Math.max(0, target - Date.now())));
    const launchedAt = Date.now();
    for (const p of parts) {
      const state = states[p.key];
      jobs.push(runAsync(p.key, [
        "action", "submit", actionID(p, i),
        "--state", state.key,
        "--base-revision", String(state.revision),
        "--value", payload(p, i),
        "--json",
      ], launchedAt).then((res) => {
        completedAt = Math.max(completedAt, res.completedAt);
        latencies.push(res.latencyMs);
        const item = JSON.parse(res.stdout);
        assertActionItem(item, p, i, state);
        return { scope: p.key, action: actionID(p, i), revision: item.revision, latencyMs: res.latencyMs };
      }));
    }
  }
  const submitted = await Promise.all(jobs);
  return {
    submitted,
    count_per_participant: count,
    wall_ms: Math.max(1, completedAt - launchedAtStart),
    participant_rate_hz: Number((count / (Math.max(1, completedAt - launchedAtStart) / 1000)).toFixed(2)),
    latency_ms: quantiles(latencies),
    latencies,
  };
}

function watchReplay(p, start, count) {
  const startedAt = Date.now();
  const out = run(p.key, [
    "watch", "prefix", actionPrefix(p),
    "--cursor", cursor(p),
    "--limit", String(count),
    "--timeout", "5s",
    "--json",
  ]);
  const elapsed = Date.now() - startedAt;
  assertNoLeak(`${p.key} watch`, out);
  const events = jsonLines(out);
  let gaps = events.length === count ? 0 : Math.abs(count - events.length);
  const prefix = `${actionPrefix(p)}.`;
  const wantIds = new Set(Array.from({ length: count }, (_, i) => actionID(p, start + i)));
  const seen = new Set();
  let prev = 0;
  for (const event of events) {
    if (!event.key.startsWith(prefix) || event.status !== "pending" || event.source !== "replay") gaps++;
    const id = event.key.startsWith(prefix) ? event.key.slice(prefix.length) : "";
    if (!wantIds.has(id) || seen.has(id)) {
      gaps++;
    } else {
      seen.add(id);
    }
    if (event.revision <= prev) gaps++;
    prev = event.revision;
    assertNoLeak(`${p.key} event`, JSON.stringify(event));
  }
  for (const id of wantIds) {
    if (!seen.has(id)) gaps++;
  }
  return { scope: p.key, elapsed_ms: elapsed, gaps, revisions: events.map((event) => event.revision), ids: events.map((event) => event.key.split(".").pop()) };
}

function duplicateReplayCheck(p) {
  try {
    const out = execFileSync(tinkalet, ["watch", "prefix", actionPrefix(p), "--cursor", cursor(p), "--limit", "1", "--timeout", "200ms", "--json"], { encoding: "utf8", env: envFor(p.key) });
    throw new Error(`${p.key} duplicate replay unexpectedly passed: ${out}`);
  } catch (err) {
    if (err.status === undefined) throw err;
    const want = `watch ${actionPrefix(p)} denied prefix: watch-timeout\n`;
    const stdout = err.stdout || "";
    const stderr = err.stderr || "";
    if (err.status !== 1 || stdout !== "" || stderr !== want) {
      throw new Error(`${p.key} duplicate replay drift ${err.status}\nstdout=${stdout}\nstderr=${stderr}\nwant=${want}`);
    }
    assertNoLeak(`${p.key} duplicate replay`, stdout + stderr);
    return 0;
  }
}

function crossDenials(states) {
  let denials = 0;
  for (const p of parts) {
    const otherApp = p.app === "demo" ? "other" : "demo";
    const other = parts.find((candidate) => candidate.app === otherApp && candidate.id === p.id);
    const neighbor = p.id === "alice" ? "bob" : "alice";
    deny(p.key, ["item", "get", states[other.key].key], `item ${states[other.key].key} denied get: denied-scope\n`);
    denials++;
    deny(p.key, ["watch", "prefix", `apps.${otherApp}.state`, "--limit", "1", "--timeout", "200ms", "--json"], `watch apps.${otherApp}.state denied prefix: denied-scope\n`);
    denials++;
    deny(p.key, ["watch", "prefix", `apps.${p.app}.participants.${neighbor}.actions`, "--limit", "1", "--timeout", "200ms", "--json"], `watch apps.${p.app}.participants.${neighbor}.actions denied prefix: denied-scope\n`);
    denials++;
    deny(p.key, ["action", "submit", `iso-restart-cross-${p.key}`, "--state", states[other.key].key, "--base-revision", String(states[other.key].revision), "--value", JSON.stringify({ cross: true })], `action iso-restart-cross-${p.key} denied submit: malformed-action\n`);
    denials++;
    deny(p.key, ["item", "create", `${actionPrefix(p)}.raw-restart`, "--value", JSON.stringify({ bypass: true })], `item ${actionPrefix(p)}.raw-restart denied create: denied-scope\n`);
    denials++;
    deny(p.key, ["trigger", "bundle.clock.tick", "--request-id", `iso-restart-${p.key}`], `profile ${p.key} denied bundle.clock.tick: denied-scope\n`);
    denials++;
  }
  return denials;
}

setup("owner", store, "owner");
for (const p of parts) {
  setup(p.key, path.join(store, "participants", p.app, p.id), p.key);
}

const start = phase.pre.count_per_participant;
const post = await submitBatch(phase.states, start, postCount);
const catchup = parts.map((p) => watchReplay(p, start, postCount));
const duplicateReplayCount = parts.reduce((sum, p) => sum + duplicateReplayCheck(p), 0);
const postRestartDenials = crossDenials(phase.states);
const seedEvents = phase.seed.reduce((sum, result) => sum + result.ids.length, 0);
const catchupEvents = catchup.reduce((sum, result) => sum + result.ids.length, 0);
const expectedActions = (phase.pre.count_per_participant + postCount) * parts.length;
const observedActions = seedEvents + catchupEvents;
const revisionGapCount = phase.seed.reduce((sum, result) => sum + result.gaps, 0) + catchup.reduce((sum, result) => sum + result.gaps, 0);
const allLatencies = [...phase.pre.latencies, ...post.latencies];
const watchLatencies = [...phase.seed.map((result) => result.elapsed_ms), ...catchup.map((result) => result.elapsed_ms)];
const totalWallMs = phase.pre.wall_ms + post.wall_ms;
const participantRateHz = Number(((phase.pre.count_per_participant + postCount) / (Math.max(1, totalWallMs) / 1000)).toFixed(2));
const rawAuthorityLeakCount = phase.raw_authority_leak_count + leakCount(JSON.stringify({ post, catchup }));
const authorityDenials = phase.authority_denials + postRestartDenials;
const expectedAuthorityDenials = parts.length * 12;
const authorityViolationCount = 0;
const apps = [...new Set(parts.map((p) => p.app))];
const body = {
  kind: "tinkabot.isoConcurrencyProof.v1",
  shell_url: phase.shell_url,
  restart_shell_url: process.env.ISO_RESTART_SHELL_URL,
  apps_started: apps.length,
  participants_started: parts.length,
  participants: parts.map((p) => p.key),
  action_count_per_participant: phase.pre.count_per_participant + postCount,
  expected_actions: expectedActions,
  observed_actions: observedActions,
  revision_gap_count: revisionGapCount,
  duplicate_replay_count: duplicateReplayCount,
  authority_denials: authorityDenials,
  expected_authority_denials: expectedAuthorityDenials,
  authority_violation_count: authorityViolationCount,
  raw_authority_leak_count: rawAuthorityLeakCount,
  participant_rate_hz_per_participant_observed: participantRateHz,
  action_latency_ms: quantiles(allLatencies),
  watch_replay_latency_ms: quantiles(watchLatencies),
  pre_restart: phase.pre,
  post_restart: post,
  seed_replay: phase.seed,
  catchup_replay: catchup,
  restart_reconnect_pass: observedActions === expectedActions && revisionGapCount === 0 && duplicateReplayCount === 0,
  capacity_claim: "observed-only",
};
if (body.observed_actions !== body.expected_actions ||
  body.revision_gap_count !== 0 ||
  body.duplicate_replay_count !== 0 ||
  body.authority_denials !== body.expected_authority_denials ||
  body.authority_violation_count !== 0 ||
  body.raw_authority_leak_count !== 0 ||
  !body.restart_reconnect_pass) {
  throw new Error(`iso concurrency proof failed: ${JSON.stringify(body, null, 2)}`);
}
fs.writeFileSync(proofPath, `${JSON.stringify(body, null, 2)}\n`);
console.log(`iso proof actions ${observedActions}/${expectedActions}`);
console.log(`iso observed per-participant rate ${participantRateHz}Hz`);
console.log(`iso action latency p95 ${body.action_latency_ms.p95}ms watch p95 ${body.watch_replay_latency_ms.p95}ms`);
JS

[[ -s "$proof" ]] || fail "isolation concurrency proof was not written"
grep -q '"apps_started": 2' "$proof" || fail "proof missing two app scopes"
grep -q '"participants_started": 4' "$proof" || fail "proof missing four participants"
grep -q '"revision_gap_count": 0' "$proof" || fail "proof recorded revision gaps"
grep -q '"duplicate_replay_count": 0' "$proof" || fail "proof recorded duplicate replay"
grep -q '"authority_denials": 48' "$proof" || fail "proof recorded wrong authority denial count"
grep -q '"authority_violation_count": 0' "$proof" || fail "proof recorded authority violation"
grep -q '"raw_authority_leak_count": 0' "$proof" || fail "proof recorded raw authority leak"
grep -q '"restart_reconnect_pass": true' "$proof" || fail "proof did not pass restart/reconnect"
grep -q '"capacity_claim": "observed-only"' "$proof" || fail "proof made no observed-only capacity claim"

log_step "isolation concurrency proof written to $proof"
printf 'proof %s\n' "$proof"
