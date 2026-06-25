#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="${1:-$(mktemp -d /tmp/tinkabot-typerace-demo.XXXXXX)}"
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
  echo "demo-typerace-anon: $*" >&2
  if [[ -n "${log:-}" && -f "$log" ]]; then
    sed -n '1,160p' "$log" | redact_log >&2 || true
  fi
  if [[ -n "${err:-}" && -f "$err" ]]; then
    sed -n '1,160p' "$err" | redact_log >&2 || true
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

run_id="$(date -u +%s%N)"
anon_a="anon-$run_id-a"
anon_b="anon-$run_id-b"

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

store="$(mktemp -d /tmp/tinkabot-typerace-store.XXXXXX)"
work="$dist/typerace-work"
log="$dist/tinkabot.log"
err="$dist/tinkabot.err"
proof="$dist/typerace-anon-proof.json"
pid=""
forward_pid=""
reducer_pid=""

cleanup() {
  if [[ -n "$reducer_pid" ]] && kill -0 "$reducer_pid" 2>/dev/null; then
    kill -TERM "$reducer_pid" 2>/dev/null || true
    wait "$reducer_pid" 2>/dev/null || true
  fi
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

log_step "start packaged Tinkabot with anonymous race participants"
(cd "$pkg" && TB_DEMO_SESSION=demo-001 ./tinkabot \
  --store "$store" \
  --shell 127.0.0.1:0 \
  --participant "demo:$anon_a" \
  --participant "demo:$anon_b" \
  >"$log" 2>"$err") &
pid=$!

for _ in {1..300}; do
  if grep -q '^shell  http://127\.0\.0\.1:' "$log" && grep -q "^participant demo $anon_b " "$log"; then
    break
  fi
  if ! kill -0 "$pid" 2>/dev/null; then
    fail "tinkabot exited before anonymous participants were admitted"
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
if [[ -z "$public_host" && "${TINKABOT_DEMO_ALLOW_LOCAL:-}" != "1" ]]; then
  fail "Tailscale host missing; set TINKABOT_DEMO_PUBLIC_HOST or TINKABOT_DEMO_ALLOW_LOCAL=1 for local-only development"
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

if [[ "$public_shell_url" != "$local_shell_url" ]]; then
  for _ in {1..50}; do
    if curl -fsS "$public_shell_url" >/dev/null 2>/dev/null; then
      break
    fi
    sleep 0.1
  done
  curl -fsS "$public_shell_url" >/dev/null || fail "Tailscale shell URL is not reachable"
fi

log_step "disable packaged NATS sidecar before profile-driven commands"
mv "$pkg/libexec/tinkabot/nats" "$pkg/libexec/tinkabot/nats.disabled"

log_step "drive anonymous typerace proof through browser and Tinkalet reducer"
PLAYWRIGHT_MODULE="$root/apps/frontend/node_modules/playwright" \
TYPE_PKG="$pkg" \
TYPE_STORE="$store" \
TYPE_WORK="$work" \
TYPE_PROOF="$proof" \
TYPE_SHELL_URL="$public_shell_url" \
TYPE_LOCAL_SHELL_URL="$local_shell_url" \
TYPE_ROUTE="$([[ "$public_shell_url" == "$local_shell_url" ]] && printf local || printf tailscale)" \
TYPE_TIMEOUT_MS="${TINKABOT_DEMO_BROWSER_TIMEOUT_MS:-60000}" \
TYPE_PARTICIPANTS="$anon_a,$anon_b" \
node <<'JS'
const { execFileSync, spawn } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");
const { chromium } = require(process.env.PLAYWRIGHT_MODULE);

const pkg = process.env.TYPE_PKG;
const store = process.env.TYPE_STORE;
const work = process.env.TYPE_WORK;
const proofPath = process.env.TYPE_PROOF;
const shellUrl = process.env.TYPE_SHELL_URL;
const localShellUrl = process.env.TYPE_LOCAL_SHELL_URL;
const route = process.env.TYPE_ROUTE;
const timeoutMs = Number(process.env.TYPE_TIMEOUT_MS);
const [anonA, anonB] = process.env.TYPE_PARTICIPANTS.split(",");
const tinkalet = path.join(pkg, "tinkalet");
const race = `race-${Date.now()}`;
const stateKey = `apps.demo.state.typerace.${race}`;
const manualRace = `${race}-manual`;
const manualStateKey = `apps.demo.state.typerace.${manualRace}`;
const prompt = "Quick ideas become visible when every racer shares the same live state.";
const proofCursor = `typeraceproof${Date.now()}`;
let actionWatcher;

for (const dir of ["home", "cfg", "data"]) {
  fs.mkdirSync(path.join(work, "owner", dir), { recursive: true, mode: 0o700 });
}

function envFor(who) {
  return {
    HOME: path.join(work, who, "home"),
    TINKALET_CONFIG_DIR: path.join(work, who, "cfg"),
    TINKALET_DATA_DIR: path.join(work, who, "data"),
    PATH: "/nonexistent",
  };
}

function run(args) {
  try {
    return execFileSync(tinkalet, args, { encoding: "utf8", env: envFor("owner") });
  } catch (err) {
    throw new Error(`owner tinkalet ${args.join(" ")} failed ${err.status}\nstdout=${err.stdout || ""}\nstderr=${err.stderr || ""}`);
  }
}

function jsonRun(args) {
  return JSON.parse(run(args));
}

function expect(label, got, want) {
  if (got !== want) throw new Error(`${label}: got ${JSON.stringify(got)}, want ${JSON.stringify(want)}`);
}

function newRace(nextRace) {
  return {
    kind: "tinkabot.typeRace.v1",
    race: nextRace,
    status: "waiting",
    text: prompt,
    players: {},
    result: null,
    events: [],
  };
}

function setup() {
  expect("owner import", run(["profile", "import", "local", "--store", store, "--name", "owner"]), "profile owner imported\n");
  expect("owner use", run(["profile", "use", "owner"]), "profile owner selected\n");
  jsonRun(["item", "create", stateKey, "--value", JSON.stringify(newRace(race)), "--json"]);
  jsonRun(["item", "create", manualStateKey, "--value", JSON.stringify(newRace(manualRace)), "--json"]);
  fs.writeFileSync(path.join(work, "manual-race"), `${manualRace}\n`);
  fs.writeFileSync(path.join(work, "manual-state-key"), `${manualStateKey}\n`);
  return state();
}

function state(key = stateKey) {
  return jsonRun(["item", "get", key, "--json"]);
}

function pageUrl(participant, alias, nextRace = race, key = stateKey) {
  const url = new URL(shellUrl);
  url.searchParams.set("tb_app", "demo");
  url.searchParams.set("tb_participant", participant);
  url.searchParams.set("tb_state", key);
  url.searchParams.set("tb_session", "demo-001");
  url.searchParams.set("tb_type", "1");
  url.searchParams.set("tb_race_no", nextRace);
  if (alias) url.searchParams.set("tb_alias", alias);
  return url.toString();
}

async function openRace(browser, participant, alias) {
  const page = await browser.newPage({ viewport: { width: 1120, height: 760 } });
  await page.goto(pageUrl(participant, alias), { waitUntil: "domcontentloaded" });
  await page.waitForSelector("iframe", { timeout: timeoutMs });
  const frame = page.frames().find((f) => f.url().startsWith("blob:"));
  if (!frame) throw new Error(`generated typerace frame missing for ${participant}`);
  await frame.waitForFunction(() => Boolean(window.__tinkabotTypeRace), null, { timeout: timeoutMs });
  await frame.waitForFunction(() => document.querySelector("#generated")?.dataset.typeReady === "true", null, { timeout: timeoutMs });
  return { participant, alias, page, frame };
}

async function typeAction(run, fn, args) {
  return run.frame.evaluate(([name, xs]) => window.__tinkabotTypeRace[name](...xs), [fn, args]);
}

async function snapshots(runs) {
  return Promise.all(runs.map(async (run) => ({
    participant: run.participant,
    url: pageUrl(run.participant, run.alias),
    dom: await domSnapshot(run.page, run.frame),
    race: await run.frame.evaluate(() => window.__tinkabotTypeRace.snapshot()),
    shellProof: await run.page.evaluate(() => window.__tinkabotProof),
  })));
}

async function domSnapshot(page, frame) {
  return {
    title: await frame.locator('[data-type="title"]').textContent(),
    status: await frame.locator('[data-type="status"]').textContent(),
    alias: await frame.locator('[data-demo="alias"]').textContent(),
    participant: await frame.locator('[data-demo="participant"]').textContent(),
    progress: await frame.locator('[data-demo="progress"]').textContent(),
    winner: await frame.locator('[data-demo="winner"]').textContent(),
    delivery: await frame.locator('[data-demo="delivery"]').textContent(),
    prompt: await frame.locator('[data-type="prompt"]').textContent(),
    runners: await frame.locator("[data-runner]").count(),
    appShell: await page.locator(".app-shell").count(),
    parentGenericShellText: (await page.locator("body").textContent()).includes("trusted shell"),
  };
}

async function waitVisible(runs, revision) {
  const start = Date.now();
  await Promise.all(runs.map((run) =>
    run.frame.waitForFunction((rev) => window.__tinkabotTypeRace.snapshot().revision >= rev, revision, { timeout: timeoutMs })
  ));
  return Date.now() - start;
}

async function submitReactVisible(runs, run, fn, args) {
  const start = Date.now();
  const out = await submitAndReact(run, fn, args);
  await waitVisible(runs, out.afterRevision);
  out.stateVisibleMs = Date.now() - start;
  return out;
}

function accepted(submitted) {
  if (submitted.action?.status !== "accepted" || !submitted.action.item?.key) {
    throw new Error(`expected accepted browser action: ${JSON.stringify(submitted)}`);
  }
  return submitted.action.item.key;
}

function denied(submitted, reason) {
  if (submitted.action?.status !== "denied" || submitted.action.reason !== reason) {
    throw new Error(`expected ${reason} browser denial: ${JSON.stringify(submitted)}`);
  }
  return submitted.action.reason;
}

function startActionWatcher() {
  const child = spawn(tinkalet, ["watch", "prefix", "apps.demo.participants", "--cursor", proofCursor, "--limit", "64", "--timeout", "60s", "--json"], {
    env: envFor("owner"),
    stdio: ["ignore", "pipe", "pipe"],
  });
  const queue = [];
  const waiters = [];
  let buf = "";
  let stderr = "";
  let stopped = false;

  function next() {
    if (queue.length > 0) return Promise.resolve(queue.shift());
    return new Promise((resolve, reject) => waiters.push({ resolve, reject }));
  }

  function push(ev) {
    const waiter = waiters.shift();
    if (waiter) waiter.resolve(ev);
    else queue.push(ev);
  }

  function fail(err) {
    while (waiters.length > 0) waiters.shift().reject(err);
  }

  child.stdout.on("data", (chunk) => {
    buf += String(chunk);
    const lines = buf.split("\n");
    buf = lines.pop() || "";
    for (const line of lines) {
      if (line.trim() === "") continue;
      try {
        push(JSON.parse(line));
      } catch (err) {
        fail(new Error(`watch returned invalid JSON: ${err instanceof Error ? err.message : String(err)}\n${line}`));
      }
    }
  });
  child.stderr.on("data", (chunk) => { stderr += String(chunk); });
  child.on("error", fail);
  child.on("close", (code) => {
    if (stopped) return;
    fail(new Error(`watch failed ${code}\nstderr=${stderr}`));
  });
  return {
    next,
    stop() {
      stopped = true;
      child.kill("SIGTERM");
    },
  };
}

async function watchNextAction() {
  if (!actionWatcher) throw new Error("action watcher not started");
  for (;;) {
    const ev = await actionWatcher.next();
    const act = ev.value || {};
    if (String(ev.key || "").endsWith(".receipt")) continue;
    if (act.kind !== "tinkabot.appAction.v1") continue;
    if (act.stateKey !== stateKey) continue;
    return ev;
  }
}

async function submitAndReact(run, fn, args) {
  const watched = watchNextAction();
  const submitted = await typeAction(run, fn, args);
  const key = accepted(submitted);
  const ev = await watched;
  if (ev.key !== key) throw new Error(`watched ${ev.key}, expected ${key}`);
  return resolve(ev);
}

function resolve(ev, key = stateKey, nextRace = race) {
  const actionKey = ev.key;
  const act = ev.value;
  const payload = act.payload || {};
  const before = state(key);
  const game = JSON.parse(JSON.stringify(before.value));
  let reason = "";
  if (payload.race !== nextRace) reason = "wrong-race";
  else if (payload.type === "join") reason = join(game, act.participantId, payload.alias);
  else if (payload.type === "progress") reason = progress(game, act.participantId, payload.typed);
  else reason = "unknown-action";

  if (reason) {
    const receipt = jsonRun(["action", "reject", actionKey, "--reason", reason, "--json"]);
    return { actionKey, watchKey: ev.key, watchKind: act.kind, type: payload.type, participant: act.participantId, outcome: "rejected", reason, beforeRevision: before.revision, afterRevision: receiptStateRevision(receipt, before.revision), receipt: receipt.key };
  }
  const receipt = jsonRun(["action", "apply", actionKey, "--value", JSON.stringify(game), "--json"]);
  return { actionKey, watchKey: ev.key, watchKind: act.kind, type: payload.type, participant: act.participantId, outcome: "applied", beforeRevision: before.revision, afterRevision: receiptStateRevision(receipt, before.revision), receipt: receipt.key, state: game };
}

function receiptStateRevision(receipt, fallback) {
  const rev = Number(receipt?.value?.stateRevision || 0);
  return rev > 0 ? rev : fallback;
}

function join(game, participantId, alias) {
  const clean = String(alias || "").trim() || `Anonymous ${participantId.slice(-5).toUpperCase()}`;
  if (game.result?.status) return "race-finished";
  if (game.players[participantId]) return "already-joined";
  game.players[participantId] = {
    participantId,
    name: clean,
    typed: "",
    progress: 0,
    percent: 0,
    mistakes: 0,
    finishedAt: "",
  };
  if (Object.keys(game.players).length >= 2 && game.status === "waiting") {
    game.status = "active";
    game.startedAt = new Date().toISOString();
  }
  event(game, "join", participantId);
  return "";
}

function progress(game, participantId, typed) {
  if (game.result?.status) return "race-finished";
  if (game.status !== "active") return "race-not-active";
  const player = game.players[participantId];
  if (!player) return "not-player";
  const scored = score(game.text, String(typed || ""));
  player.typed = String(typed || "");
  player.progress = scored.progress;
  player.percent = scored.percent;
  player.mistakes = scored.mistakes;
  event(game, "progress", participantId);
  if (scored.done) {
    const now = new Date().toISOString();
    player.finishedAt = now;
    game.status = "finished";
    game.result = { status: "finished", winner: participantId, alias: player.name, finishedAt: now };
    event(game, "finished", participantId);
  }
  return "";
}

function score(text, typed) {
  const src = Array.from(String(text || ""));
  const got = Array.from(String(typed || ""));
  let progress = 0;
  while (progress < src.length && got[progress] === src[progress]) progress += 1;
  return {
    progress,
    percent: src.length ? Math.floor((progress / src.length) * 100) : 0,
    mistakes: Math.max(0, got.length - progress),
    done: progress === src.length && got.length === src.length,
  };
}

function event(game, type, participantId) {
  game.events = [...(game.events || []), { type, participantId, at: new Date().toISOString() }].slice(-24);
}

function leakTerms(text) {
  return [
    ["tb_items", /tb_items/i],
    ["kv-subject", /\$kv/i],
    ["nats-block", /begin nats/i],
    ["private-key", /private key/i],
    ["nats-url", /nats:\/\//i],
    ["creds-path", /\.creds\b/i],
    ["jwt", /\bjwt\b/i],
    ["nkey", /\bnkey\b/i],
    ["seed", /\bseed\b/i],
    ["bearer", /\bbearer\b/i],
    ["credential", /\bcredentials?\b/i],
    ["token", /\btoken\b/i],
  ].filter(([, re]) => re.test(text)).map(([term]) => term);
}

function pct(values, p) {
  if (values.length === 0) return 0;
  const xs = [...values].sort((a, b) => a - b);
  const i = Math.min(xs.length - 1, Math.ceil((p / 100) * xs.length) - 1);
  return xs[i];
}

function stateReadDispatches(participants, key) {
  return participants.flatMap((part) => (part.shellProof?.dispatched || [])
    .filter((hit) => hit.command === "participant_read" && hit.payloadKey === key)
    .map((hit) => ({
      participant: part.participant,
      commandId: hit.commandId,
      status: hit.status,
      payloadKey: hit.payloadKey,
    })));
}

(async () => {
  const initial = setup();
  actionWatcher = startActionWatcher();
  const actionLog = [];
  const denialLog = [];
  const browser = await chromium.launch({ headless: true });
  try {
    const runnerA = await openRace(browser, anonA, "Anonymous A");
    const runnerB = await openRace(browser, anonB, "Anonymous B");
    const runs = [runnerA, runnerB];

    await runnerA.frame.evaluate((other) => window.__tinkabotTypeRace.escape(other), anonB);
    await runnerA.page.waitForFunction(() => (window.__tinkabotProof?.denied || []).some((msg) => String(msg).includes("Participant")), null, { timeout: timeoutMs });
    denialLog.push({ kind: "participant-escape", status: "denied", reason: await runnerA.page.evaluate(() => window.__tinkabotProof.denied.at(-1)) });

    actionLog.push(await submitReactVisible(runs, runnerA, "join", ["Anonymous A"]));
    denialLog.push({ kind: "duplicate-join", ...(await submitReactVisible(runs, runnerA, "join", ["Anonymous A"])) });
    actionLog.push(await submitReactVisible(runs, runnerB, "join", ["Anonymous B"]));

    actionLog.push(await submitReactVisible(runs, runnerA, "progress", [prompt.slice(0, 12), { actionId: "a-progress-1" }]));
    denialLog.push({
      kind: "duplicate-action",
      reason: denied(await typeAction(runnerA, "progress", [prompt.slice(0, 16), { actionId: "a-progress-1" }]), "duplicate-action"),
    });
    denialLog.push({
      kind: "stale-revision",
      reason: denied(await typeAction(runnerB, "progress", [prompt.slice(0, 8), { actionId: "b-stale", baseRevision: initial.revision }]), "stale-revision"),
    });
    actionLog.push(await submitReactVisible(runs, runnerB, "progress", [prompt.slice(0, 10), { actionId: "b-progress-1" }]));
    actionLog.push(await submitReactVisible(runs, runnerA, "progress", [prompt.slice(0, 34), { actionId: "a-progress-2" }]));
    actionLog.push(await submitReactVisible(runs, runnerB, "progress", [prompt.slice(0, 24), { actionId: "b-progress-2" }]));
    actionLog.push(await submitReactVisible(runs, runnerA, "progress", [prompt, { actionId: "a-finish" }]));
    denialLog.push({ kind: "late-progress", ...(await submitReactVisible(runs, runnerB, "progress", [prompt, { actionId: "b-late-finish" }])) });

    await Promise.all(runs.map((run) =>
      run.frame.waitForFunction(() => window.__tinkabotTypeRace.snapshot().result?.status === "finished", null, { timeout: timeoutMs })
    ));
    const participants = await snapshots(runs);
    const final = state();
    const visibleLatencies = [...actionLog, ...denialLog].map((x) => x.stateVisibleMs).filter((x) => Number.isFinite(x));
    const generatedStateReads = stateReadDispatches(participants, stateKey);
    const stateDelivery = participants.every((part) => part.race.delivery === "trusted-shell.nats-watch.push")
      ? "trusted-shell.nats-watch.push"
      : "missing";
    const text = JSON.stringify({ participants, actionLog, denialLog, final });
    const leaks = leakTerms(text);
    const proof = {
      kind: "tinkabot.anonymousTypeRaceProof.v1",
      route,
      shellUrl,
      localShellUrl,
      race,
      stateKey,
      manualRace,
      manualStateKey,
      anonymousUsers: [anonA, anonB],
      prompt,
      participants: participants.map((part) => ({
        participant: part.participant,
        url: part.url,
        dom: part.dom,
        race: part.race,
        shellState: part.shellProof?.state,
      })),
      actionLog,
      denialLog,
      reactionMode: "tinkalet.watch.prefix",
      stateDelivery,
      generatedIframePollingCount: generatedStateReads.length,
      generatedIframePollingProof: {
        source: "trusted-shell.dispatched",
        stateReadCommands: generatedStateReads,
      },
      stateVisibleP95Ms: pct(visibleLatencies, 95),
      stateVisibleP99Ms: pct(visibleLatencies, 99),
      finalRace: final.value,
      authorityLeakCount: leaks.length,
      authorityLeakTerms: leaks,
      platformTypeRaceAPIAdditions: 0,
    };
    proof.pass = proof.route === "tailscale" &&
      proof.anonymousUsers.every((id) => id.startsWith("anon-")) &&
      proof.finalRace.result?.winner === anonA &&
      proof.finalRace.status === "finished" &&
      proof.denialLog.some((d) => d.kind === "participant-escape" && d.status === "denied") &&
      proof.denialLog.some((d) => d.kind === "duplicate-join" && d.reason === "already-joined") &&
      proof.denialLog.some((d) => d.kind === "duplicate-action" && d.reason === "duplicate-action") &&
      proof.denialLog.some((d) => d.kind === "stale-revision" && d.reason === "stale-revision") &&
      proof.denialLog.some((d) => d.kind === "late-progress" && d.reason === "race-finished") &&
      proof.participants.every((p) => p.race.result?.winner === anonA && p.dom.runners === 2 && p.dom.appShell === 1 && p.dom.parentGenericShellText === false) &&
      [...proof.actionLog, ...proof.denialLog.filter((d) => d.actionKey)].every((a) => a.watchKey === a.actionKey && a.watchKind === "tinkabot.appAction.v1") &&
      proof.stateDelivery === "trusted-shell.nats-watch.push" &&
      proof.generatedIframePollingCount === 0 &&
      proof.stateVisibleP95Ms <= 200 &&
      proof.stateVisibleP99Ms <= 300 &&
      proof.authorityLeakCount === 0 &&
      proof.platformTypeRaceAPIAdditions === 0;
    fs.writeFileSync(proofPath, `${JSON.stringify(proof, null, 2)}\n`);
    console.log(`anonymous typerace proof ${proof.route} race ${proof.race} winner ${proof.finalRace.result?.winner} denials ${proof.denialLog.length}`);
    if (!proof.pass) throw new Error(`anonymous typerace proof failed: ${JSON.stringify(proof)}`);
  } finally {
    actionWatcher?.stop();
    await browser.close();
  }
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
JS

grep -q '"kind": "tinkabot.anonymousTypeRaceProof.v1"' "$proof" || fail "anonymous typerace proof kind missing"
grep -q '"pass": true' "$proof" || fail "anonymous typerace proof did not pass"
grep -q '"authorityLeakCount": 0' "$proof" || fail "anonymous typerace proof recorded authority leaks"

log_step "demo passed"
printf 'package root %s\n' "$pkg"
printf 'artifacts %s\n' "$dist"
printf 'typerace proof %s\n' "$proof"

if [[ "${TINKABOT_DEMO_HOLD:-}" == "1" ]]; then
  manual_race="$(cat "$work/manual-race")"
  manual_state_key="$(cat "$work/manual-state-key")"
  manual_log="$dist/typerace-manual-reducer.log"
  log_step "start manual anonymous typerace reducer for $manual_race"
  env -i \
    HOME="$work/owner/home" \
    TINKALET_CONFIG_DIR="$work/owner/cfg" \
    TINKALET_DATA_DIR="$work/owner/data" \
    PATH="/nonexistent" \
    TINKALET="$pkg/tinkalet" \
    STATE_KEY="$manual_state_key" \
    RACE_NO="$manual_race" \
    CURSOR="typeracemanual$(date -u +%H%M%S)" \
    /home/lagz0ne/.local/share/mise/installs/node/latest/bin/node >"$manual_log" 2>&1 <<'JS' &
const { execFileSync } = require("node:child_process");

const tinkalet = process.env.TINKALET;
const stateKey = process.env.STATE_KEY;
const raceNo = process.env.RACE_NO;
const cursor = process.env.CURSOR;
const seen = new Set();

function run(args) {
  return execFileSync(tinkalet, args, { encoding: "utf8", env: process.env });
}
function json(args) { return JSON.parse(run(args)); }
function state() { return json(["item", "get", stateKey, "--json"]); }
function score(text, typed) {
  const src = Array.from(String(text || ""));
  const got = Array.from(String(typed || ""));
  let progress = 0;
  while (progress < src.length && got[progress] === src[progress]) progress += 1;
  return {
    progress,
    percent: src.length ? Math.floor((progress / src.length) * 100) : 0,
    mistakes: Math.max(0, got.length - progress),
    done: progress === src.length && got.length === src.length,
  };
}
function event(game, type, participantId) {
  game.events = [...(game.events || []), { type, participantId, at: new Date().toISOString() }].slice(-24);
}
function join(game, participantId, alias) {
  const clean = String(alias || "").trim() || `Anonymous ${participantId.slice(-5).toUpperCase()}`;
  if (game.result?.status) return "race-finished";
  if (game.players[participantId]) return "already-joined";
  game.players[participantId] = { participantId, name: clean, typed: "", progress: 0, percent: 0, mistakes: 0, finishedAt: "" };
  if (Object.keys(game.players).length >= 2 && game.status === "waiting") {
    game.status = "active";
    game.startedAt = new Date().toISOString();
  }
  event(game, "join", participantId);
  return "";
}
function progress(game, participantId, typed) {
  if (game.result?.status) return "race-finished";
  if (game.status !== "active") return "race-not-active";
  const player = game.players[participantId];
  if (!player) return "not-player";
  const scored = score(game.text, typed);
  player.typed = String(typed || "");
  player.progress = scored.progress;
  player.percent = scored.percent;
  player.mistakes = scored.mistakes;
  event(game, "progress", participantId);
  if (scored.done) {
    const now = new Date().toISOString();
    player.finishedAt = now;
    game.status = "finished";
    game.result = { status: "finished", winner: participantId, alias: player.name, finishedAt: now };
    event(game, "finished", participantId);
  }
  return "";
}
function resolve(key, act) {
  if (seen.has(key)) return;
  seen.add(key);
  const before = state();
  const game = JSON.parse(JSON.stringify(before.value));
  const payload = act.payload || {};
  let reason = "";
  if (payload.race !== raceNo) reason = "wrong-race";
  else if (payload.type === "join") reason = join(game, act.participantId, payload.alias);
  else if (payload.type === "progress") reason = progress(game, act.participantId, payload.typed);
  else reason = "unknown-action";
  if (reason) {
    const receipt = json(["action", "reject", key, "--reason", reason, "--json"]);
    console.log(JSON.stringify({ at: new Date().toISOString(), action: key, outcome: "rejected", reason, receipt: receipt.key }));
    return;
  }
  const receipt = json(["action", "apply", key, "--value", JSON.stringify(game), "--json"]);
  const after = state();
  console.log(JSON.stringify({ at: new Date().toISOString(), action: key, outcome: "applied", status: after.value.status, result: after.value.result, receipt: receipt.key }));
}
console.log(JSON.stringify({ at: new Date().toISOString(), reducer: "started", stateKey, raceNo, cursor }));
for (;;) {
  try {
    const out = run(["watch", "prefix", "apps.demo.participants", "--cursor", cursor, "--limit", "1", "--timeout", "30s", "--json"]);
    const ev = JSON.parse(out);
    const act = ev.value || {};
    if (act.kind !== "tinkabot.appAction.v1" || act.stateKey !== stateKey) continue;
    resolve(ev.key, act);
  } catch (err) {
    const stderr = String(err.stderr || "");
    if (stderr.includes("watch-timeout")) continue;
    console.log(JSON.stringify({ at: new Date().toISOString(), error: String(err.message || err), stderr }));
    Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, 500);
  }
}
JS
  reducer_pid=$!
  runner_a_url="$public_shell_url/?tb_app=demo&tb_participant=$anon_a&tb_state=$manual_state_key&tb_session=demo-001&tb_type=1&tb_race_no=$manual_race&tb_alias=Anonymous%20A"
  runner_b_url="$public_shell_url/?tb_app=demo&tb_participant=$anon_b&tb_state=$manual_state_key&tb_session=demo-001&tb_type=1&tb_race_no=$manual_race&tb_alias=Anonymous%20B"
  manual_smoke="$dist/typerace-manual-smoke.json"
  log_step "smoke manual anonymous typerace links"
  PLAYWRIGHT_MODULE="$root/apps/frontend/node_modules/playwright" \
    RUNNER_A_URL="$runner_a_url" \
    RUNNER_B_URL="$runner_b_url" \
    MANUAL_SMOKE="$manual_smoke" \
    /home/lagz0ne/.local/share/mise/installs/node/latest/bin/node <<'JS'
const fs = require("node:fs");
const { chromium } = require(process.env.PLAYWRIGHT_MODULE);
const links = [
  ["runner-a", process.env.RUNNER_A_URL],
  ["runner-b", process.env.RUNNER_B_URL],
];

(async () => {
  const browser = await chromium.launch({ headless: true });
  const out = [];
  try {
    for (const [label, url] of links) {
      const page = await browser.newPage({ viewport: { width: 1120, height: 760 } });
      await page.goto(url, { waitUntil: "domcontentloaded", timeout: 60000 });
      await page.waitForSelector("iframe", { timeout: 60000 });
      const frame = page.frames().find((f) => f.url().startsWith("blob:"));
      if (!frame) throw new Error(`missing generated frame for ${label}`);
      await frame.waitForSelector('[data-type="prompt"]', { timeout: 60000 });
      await frame.waitForFunction(() => document.querySelector('[data-type="prompt"]')?.textContent.includes("Quick ideas"), null, { timeout: 60000 });
      out.push({
        label,
        url,
        title: await frame.locator('[data-type="title"]').textContent(),
        status: await frame.locator('[data-type="status"]').textContent(),
        prompt: await frame.locator('[data-type="prompt"]').textContent(),
        appShell: await page.locator(".app-shell").count(),
        parentGenericShellText: (await page.locator("body").textContent()).includes("trusted shell"),
      });
      await page.close();
    }
  } finally {
    await browser.close();
  }
  const pass = out.every((p) => p.status === "waiting" && p.prompt.includes("Quick ideas") && p.appShell === 1 && p.parentGenericShellText === false);
  fs.writeFileSync(process.env.MANUAL_SMOKE, `${JSON.stringify({ kind: "tinkabot.anonymousTypeRaceManualSmoke.v1", pass, pages: out }, null, 2)}\n`);
  if (!pass) throw new Error(`manual anonymous typerace smoke failed: ${JSON.stringify(out)}`);
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
JS
  printf 'manual-race %s\n' "$manual_race"
  printf 'manual-reducer %s\n' "$manual_log"
  printf 'manual-smoke %s\n' "$manual_smoke"
  printf 'runner-a %s\n' "$runner_a_url"
  printf 'runner-b %s\n' "$runner_b_url"
  log_step "holding anonymous typerace app open"
  while kill -0 "$pid" 2>/dev/null; do
    sleep 3600 &
    wait $! || true
  done
fi
