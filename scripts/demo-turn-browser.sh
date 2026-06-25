#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="${1:-$(mktemp -d /tmp/tinkabot-turn-browser-demo.XXXXXX)}"
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
  echo "demo-turn-browser: $*" >&2
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

store="$(mktemp -d /tmp/tinkabot-turn-browser-store.XXXXXX)"
work="$dist/turn-browser-work"
log="$dist/tinkabot.log"
err="$dist/tinkabot.err"
proof="$dist/browser-turn-board-proof.json"
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
(cd "$pkg" && TB_DEMO_SESSION=demo-001 ./tinkabot \
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

log_step "drive browser board and reducer through packaged Tinkalet"
PLAYWRIGHT_MODULE="$root/apps/frontend/node_modules/playwright" \
TURN_BROWSER_PKG="$pkg" \
TURN_BROWSER_STORE="$store" \
TURN_BROWSER_WORK="$work" \
TURN_BROWSER_PROOF="$proof" \
TURN_BROWSER_SHELL_URL="$public_shell_url" \
TURN_BROWSER_LOCAL_SHELL_URL="$local_shell_url" \
TURN_BROWSER_ROUTE="$([[ "$public_shell_url" == "$local_shell_url" ]] && printf local || printf tailscale)" \
TURN_BROWSER_TIMEOUT_MS="${TINKABOT_DEMO_BROWSER_TIMEOUT_MS:-60000}" \
node <<'JS'
const { execFileSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");
const { chromium } = require(process.env.PLAYWRIGHT_MODULE);

const pkg = process.env.TURN_BROWSER_PKG;
const store = process.env.TURN_BROWSER_STORE;
const work = process.env.TURN_BROWSER_WORK;
const proofPath = process.env.TURN_BROWSER_PROOF;
const shellUrl = process.env.TURN_BROWSER_SHELL_URL;
const timeoutMs = Number(process.env.TURN_BROWSER_TIMEOUT_MS);
const tinkalet = path.join(pkg, "tinkalet");
const stateKey = "apps.demo.state.board";
const lines = [
  ["a1", "a2", "a3"],
  ["b1", "b2", "b3"],
  ["c1", "c2", "c3"],
  ["a1", "b1", "c1"],
  ["a2", "b2", "c2"],
  ["a3", "b3", "c3"],
  ["a1", "b2", "c3"],
  ["a3", "b2", "c1"],
];

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

function deny(args, want) {
  try {
    const out = execFileSync(tinkalet, args, { encoding: "utf8", env: envFor("owner") });
    throw new Error(`owner tinkalet ${args.join(" ")} unexpectedly passed: ${out}`);
  } catch (err) {
    if (err.status === undefined) throw err;
    const stdout = err.stdout || "";
    const stderr = err.stderr || "";
    if (err.status !== 1 || stdout !== "" || stderr !== want) {
      throw new Error(`owner tinkalet ${args.join(" ")} denial drift ${err.status}\nstdout=${stdout}\nstderr=${stderr}\nwant=${want}`);
    }
  }
}

function jsonRun(args) {
  return JSON.parse(run(args));
}

function setup() {
  expect("owner import", run(["profile", "import", "local", "--store", store, "--name", "owner"]), "profile owner imported\n");
  expect("owner use", run(["profile", "use", "owner"]), "profile owner selected\n");
  return jsonRun(["item", "create", stateKey, "--value", JSON.stringify({ turn: "alice", cells: {} }), "--json"]);
}

function expect(label, got, want) {
  if (got !== want) throw new Error(`${label}: got ${JSON.stringify(got)}, want ${JSON.stringify(want)}`);
}

function state() {
  return jsonRun(["item", "get", stateKey, "--json"]);
}

function pageUrl(participant) {
  const url = new URL(shellUrl);
  url.searchParams.set("tb_app", "demo");
  url.searchParams.set("tb_participant", participant);
  url.searchParams.set("tb_state", stateKey);
  url.searchParams.set("tb_session", "demo-001");
  url.searchParams.set("tb_board", "1");
  return url.toString();
}

async function openBoard(browser, participant) {
  const page = await browser.newPage({ viewport: { width: 900, height: 720 } });
  const url = pageUrl(participant);
  await page.goto(url, { waitUntil: "domcontentloaded" });
  await page.waitForSelector("iframe", { timeout: timeoutMs });
  const frame = page.frames().find((f) => f.url().startsWith("blob:"));
  if (!frame) throw new Error(`generated frame missing for ${participant}`);
  await frame.waitForFunction(() => Boolean(window.__tinkabotDemo), null, { timeout: timeoutMs });
  await frame.waitForFunction(() => document.querySelector("#generated")?.dataset.boardReady === "true", null, { timeout: timeoutMs });
  return { participant, page, frame, url };
}

async function submit(run, cell, options = {}) {
  const result = await run.frame.evaluate(
    ([nextCell, opts]) => window.__tinkabotDemo.submit(nextCell, opts),
    [cell, options],
  );
  return {
    participant: run.participant,
    cell,
    action: result.action,
    snapshot: result.snapshot,
  };
}

async function refreshAll(runs) {
  return Promise.all(runs.map((run) => run.frame.evaluate(() => window.__tinkabotDemo.refresh())));
}

async function snapshots(runs) {
  return Promise.all(runs.map(async (run) => ({
    participant: run.participant,
    dom: await domSnapshot(run.frame),
    board: await run.frame.evaluate(() => window.__tinkabotDemo.snapshot()),
    shellProof: await run.page.evaluate(() => window.__tinkabotProof),
  })));
}

async function domSnapshot(frame) {
  return {
    title: await frame.locator('[data-demo="title"]').textContent(),
    status: await frame.locator('[data-demo="status"]').textContent(),
    turn: await frame.locator('[data-demo="turn"]').textContent(),
    winner: await frame.locator('[data-demo="winner"]').textContent(),
    cells: await frame.locator('[data-demo="cells"]').textContent(),
    actions: Number(await frame.locator('[data-demo="actions"]').textContent()),
    receipts: Number(await frame.locator('[data-demo="readbacks"]').textContent()),
    denied: Number(await frame.locator('[data-demo="denied"]').textContent()),
    text: await frame.locator("#generated").textContent(),
  };
}

async function waitWinner(runs) {
  await Promise.all(runs.map((run) =>
    run.frame.waitForFunction(() => window.__tinkabotDemo.snapshot().winner === "alice", null, { timeout: timeoutMs })
  ));
}

function acceptedAction(submitted) {
  if (submitted.action?.status !== "accepted" || !submitted.action.item?.key) {
    throw new Error(`expected accepted browser action: ${JSON.stringify(submitted)}`);
  }
  return submitted.action.item.key;
}

function deniedAction(submitted, reason) {
  if (submitted.action?.status !== "denied" || submitted.action.reason !== reason) {
    throw new Error(`expected ${reason} browser denial: ${JSON.stringify(submitted)}`);
  }
  return submitted.action;
}

function resolve(actionKey) {
  const action = jsonRun(["item", "get", actionKey, "--json"]);
  const act = action.value;
  const move = act.payload || {};
  const before = state();
  const board = clone(before.value);
  let receipt;
  let outcome = "applied";
  let reason = "";
  if (board.winner) {
    reason = "game-over";
  } else if (board.turn !== act.participantId) {
    reason = "wrong-turn";
  } else if (board.cells && board.cells[move.cell]) {
    reason = "occupied-cell";
  }
  if (reason) {
    receipt = jsonRun(["action", "reject", actionKey, "--reason", reason, "--json"]);
    outcome = "rejected";
  } else {
    board.cells = board.cells || {};
    board.cells[move.cell] = act.participantId;
    if (winner(board, act.participantId)) {
      board.winner = act.participantId;
    } else {
      board.turn = act.participantId === "alice" ? "bob" : "alice";
    }
    receipt = jsonRun(["action", "apply", actionKey, "--value", JSON.stringify(board), "--json"]);
  }
  const after = state();
  return {
    actionKey,
    participant: act.participantId,
    cell: move.cell,
    beforeRevision: before.revision,
    afterRevision: after.revision,
    actionRevision: action.revision,
    outcome,
    reason,
    receipt: {
      key: receipt.key,
      status: receipt.status,
      revision: receipt.revision,
      value: receipt.value,
    },
  };
}

function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

function winner(board, participant) {
  return lines.some((line) => line.every((cell) => board.cells?.[cell] === participant));
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

(async () => {
  const initial = setup();
  const moveLog = [];
  const denialLog = [];
  const browser = await chromium.launch({ headless: true });
  try {
    const alice = await openBoard(browser, "alice");
    const bob = await openBoard(browser, "bob");
    const runs = [alice, bob];

    await alice.frame.evaluate(() => window.__tinkabotDemo.escape());
    await alice.page.waitForFunction(() => (window.__tinkabotProof?.denied || []).some((msg) => String(msg).includes("Participant")), null, { timeout: timeoutMs });
    const escapeProof = await alice.page.evaluate(() => window.__tinkabotProof.denied.at(-1));
    denialLog.push({ kind: "participant-escape", status: "denied", reason: escapeProof });

    let submitted = await submit(bob, "b1", { actionId: "b-wrong-turn" });
    let actionKey = acceptedAction(submitted);
    let resolved = resolve(actionKey);
    denialLog.push({ kind: "wrong-turn", ...resolved });
    await refreshAll(runs);

    submitted = await submit(alice, "a1", { actionId: "a1" });
    actionKey = acceptedAction(submitted);
    resolved = resolve(actionKey);
    moveLog.push(resolved);
    await refreshAll(runs);

    const current = state();
    submitted = await submit(alice, "c1", { actionId: "a1", baseRevision: current.revision });
    denialLog.push({ kind: "duplicate-action", participant: "alice", cell: "c1", status: "denied", reason: deniedAction(submitted, "duplicate-action").reason });

    submitted = await submit(bob, "b1", { actionId: "b-stale", baseRevision: initial.revision });
    denialLog.push({ kind: "stale-revision", participant: "bob", cell: "b1", status: "denied", reason: deniedAction(submitted, "stale-revision").reason });
    deny(["item", "get", "apps.demo.participants.bob.actions.b-stale"], "item apps.demo.participants.bob.actions.b-stale denied get: item-not-found\n");

    submitted = await submit(bob, "a1", { actionId: "b-occupied" });
    actionKey = acceptedAction(submitted);
    resolved = resolve(actionKey);
    denialLog.push({ kind: "occupied-cell", ...resolved });
    await refreshAll(runs);

    for (const [run, cell, id] of [
      [bob, "b1", "b1"],
      [alice, "a2", "a2"],
      [bob, "b2", "b2"],
      [alice, "a3", "a3"],
    ]) {
      submitted = await submit(run, cell, { actionId: id });
      actionKey = acceptedAction(submitted);
      resolved = resolve(actionKey);
      moveLog.push(resolved);
      await refreshAll(runs);
    }

    await waitWinner(runs);
    const participants = await snapshots(runs);
    const final = state();
    const text = JSON.stringify({ participants, moveLog, denialLog, final });
    const leaks = leakTerms(text);
    const proof = {
      kind: "tinkabot.browserTurnBoardProof.v1",
      route: process.env.TURN_BROWSER_ROUTE,
      shellUrl,
      localShellUrl: process.env.TURN_BROWSER_LOCAL_SHELL_URL,
      participants: participants.map((part) => ({
        participant: part.participant,
        url: pageUrl(part.participant),
        dom: part.dom,
        board: part.board,
      })),
      moveLog,
      denialLog,
      finalBoard: final.value,
      authorityLeakCount: leaks.length,
      authorityLeakTerms: leaks,
    };
    proof.pass = proof.route === "tailscale" &&
      proof.participants.length === 2 &&
      proof.moveLog.filter((m) => m.outcome === "applied").length >= 5 &&
      proof.denialLog.some((d) => d.kind === "wrong-turn" && d.outcome === "rejected") &&
      proof.denialLog.some((d) => d.kind === "occupied-cell" && d.outcome === "rejected") &&
      proof.denialLog.some((d) => d.kind === "duplicate-action" && d.reason === "duplicate-action") &&
      proof.denialLog.some((d) => d.kind === "stale-revision" && d.reason === "stale-revision") &&
      proof.denialLog.some((d) => d.kind === "participant-escape" && d.status === "denied") &&
      proof.finalBoard.winner === "alice" &&
      proof.participants.every((p) => p.board.winner === "alice" && p.dom.winner === "alice") &&
      proof.authorityLeakCount === 0;
    fs.writeFileSync(proofPath, `${JSON.stringify(proof, null, 2)}\n`);
    console.log(`browser turn board proof ${proof.route} moves ${proof.moveLog.length} denials ${proof.denialLog.length} winner ${proof.finalBoard.winner}`);
    if (!proof.pass) throw new Error(`browser turn board proof failed: ${JSON.stringify(proof)}`);
  } finally {
    await browser.close();
  }
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
JS

grep -q '"kind": "tinkabot.browserTurnBoardProof.v1"' "$proof" || fail "browser turn proof kind missing"
grep -q '"winner": "alice"' "$proof" || fail "browser turn proof did not record Alice winner"
grep -q '"authorityLeakCount": 0' "$proof" || fail "browser turn proof recorded authority leaks"

log_step "demo passed"
printf 'package root %s\n' "$pkg"
printf 'artifacts %s\n' "$dist"
printf 'browser turn proof %s\n' "$proof"
printf 'open %s\n' "$public_shell_url"

if [[ "${TINKABOT_DEMO_HOLD:-}" == "1" ]]; then
  log_step "holding demo shell open"
  while kill -0 "$pid" 2>/dev/null; do
    sleep 3600 &
    wait $! || true
  done
fi
