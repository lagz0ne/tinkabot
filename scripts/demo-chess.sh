#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="${1:-$(mktemp -d /tmp/tinkabot-chess-demo.XXXXXX)}"
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
  echo "demo-chess: $*" >&2
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

store="$(mktemp -d /tmp/tinkabot-chess-store.XXXXXX)"
work="$dist/chess-work"
log="$dist/tinkabot.log"
err="$dist/tinkabot.err"
proof="$dist/chess-proof.json"
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

log_step "start packaged Tinkabot with chess participants"
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

log_step "drive real chess app proof through browser and Tinkalet reducer"
PLAYWRIGHT_MODULE="$root/apps/frontend/node_modules/playwright" \
CHESS_JS="$root/apps/frontend/node_modules/chess.js" \
CHESS_PKG="$pkg" \
CHESS_STORE="$store" \
CHESS_WORK="$work" \
CHESS_PROOF="$proof" \
CHESS_SHELL_URL="$public_shell_url" \
CHESS_LOCAL_SHELL_URL="$local_shell_url" \
CHESS_ROUTE="$([[ "$public_shell_url" == "$local_shell_url" ]] && printf local || printf tailscale)" \
CHESS_TIMEOUT_MS="${TINKABOT_DEMO_BROWSER_TIMEOUT_MS:-60000}" \
node <<'JS'
const { execFileSync, spawn } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");
const { chromium } = require(process.env.PLAYWRIGHT_MODULE);
const { Chess } = require(process.env.CHESS_JS);

const pkg = process.env.CHESS_PKG;
const store = process.env.CHESS_STORE;
const work = process.env.CHESS_WORK;
const proofPath = process.env.CHESS_PROOF;
const shellUrl = process.env.CHESS_SHELL_URL;
const timeoutMs = Number(process.env.CHESS_TIMEOUT_MS);
const route = process.env.CHESS_ROUTE;
const localShellUrl = process.env.CHESS_LOCAL_SHELL_URL;
const tinkalet = path.join(pkg, "tinkalet");
const board = `board-${Date.now()}`;
const stateKey = `apps.demo.state.chess.${board}`;
const manualBoard = `${board}-manual`;
const manualStateKey = `apps.demo.state.chess.${manualBoard}`;
const proofCursor = `chessproof${Date.now()}`;
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

function newGame(nextBoard) {
  const c = new Chess();
  return {
    kind: "tinkabot.chessGame.v1",
    board: nextBoard,
    status: "waiting",
    fen: c.fen(),
    turn: "white",
    players: {},
    history: [],
    result: null,
  };
}

function setup() {
  expect("owner import", run(["profile", "import", "local", "--store", store, "--name", "owner"]), "profile owner imported\n");
  expect("owner use", run(["profile", "use", "owner"]), "profile owner selected\n");
  jsonRun(["item", "create", stateKey, "--value", JSON.stringify(newGame(board)), "--json"]);
  jsonRun(["item", "create", manualStateKey, "--value", JSON.stringify(newGame(manualBoard)), "--json"]);
  fs.writeFileSync(path.join(work, "manual-board"), `${manualBoard}\n`);
  fs.writeFileSync(path.join(work, "manual-state-key"), `${manualStateKey}\n`);
}

function state(key = stateKey) {
  return jsonRun(["item", "get", key, "--json"]);
}

function pageUrl(participant, name, nextBoard = board, key = stateKey) {
  const url = new URL(shellUrl);
  url.searchParams.set("tb_app", "demo");
  url.searchParams.set("tb_participant", participant);
  url.searchParams.set("tb_state", key);
  url.searchParams.set("tb_session", "demo-001");
  url.searchParams.set("tb_chess", "1");
  url.searchParams.set("tb_board_no", nextBoard);
  if (name) url.searchParams.set("tb_name", name);
  return url.toString();
}

async function openChess(browser, participant, name) {
  const page = await browser.newPage({ viewport: { width: 1120, height: 760 } });
  await page.goto(pageUrl(participant, name), { waitUntil: "domcontentloaded" });
  await page.waitForSelector("iframe", { timeout: timeoutMs });
  const frame = page.frames().find((f) => f.url().startsWith("blob:"));
  if (!frame) throw new Error(`generated chess frame missing for ${participant}`);
  await frame.waitForFunction(() => Boolean(window.__tinkabotChess), null, { timeout: timeoutMs });
  await frame.waitForFunction(() => document.querySelector("#generated")?.dataset.chessReady === "true", null, { timeout: timeoutMs });
  return { participant, name, page, frame };
}

async function chessAction(run, fn, args) {
  return run.frame.evaluate(([name, xs]) => window.__tinkabotChess[name](...xs), [fn, args]);
}

async function proveNameTyping(run, value) {
  const input = run.frame.locator('[data-chess="name"]');
  await input.fill("");
  await input.focus();
  await run.page.keyboard.type(value, { delay: 160 });
  await run.page.waitForTimeout(800);
  const got = await input.inputValue();
  if (got !== value) throw new Error(`name input refresh lost typing for ${run.participant}: ${JSON.stringify(got)}`);
  return true;
}

async function snapshots(runs) {
  return Promise.all(runs.map(async (run) => ({
    participant: run.participant,
    url: pageUrl(run.participant, run.name),
    dom: await domSnapshot(run.page, run.frame),
    chess: await run.frame.evaluate(() => window.__tinkabotChess.snapshot()),
    shellProof: await run.page.evaluate(() => window.__tinkabotProof),
  })));
}

async function waitVisible(runs, revision) {
  const start = Date.now();
  await Promise.all(runs.map((run) =>
    run.frame.waitForFunction((rev) => window.__tinkabotChess.snapshot().revision >= rev, revision, { timeout: timeoutMs })
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

async function touchProbe(run) {
  const square = run.frame.locator('[data-square="e2"]');
  const box = await square.boundingBox();
  await square.click();
  const selected = await square.evaluate((el) => el.className.includes("chess-selected"));
  await square.click();
  return {
    pass: Boolean(selected && box && box.width >= 44 && box.height >= 44),
    square: "e2",
    width: Math.round(box?.width || 0),
    height: Math.round(box?.height || 0),
    selected,
  };
}

async function domSnapshot(page, frame) {
  return {
    title: await frame.locator('[data-chess="title"]').textContent(),
    status: await frame.locator('[data-chess="status"]').textContent(),
    white: await frame.locator('[data-demo="white"]').textContent(),
    black: await frame.locator('[data-demo="black"]').textContent(),
    color: await frame.locator('[data-demo="color"]').textContent(),
    turn: await frame.locator('[data-demo="turn"]').textContent(),
    result: await frame.locator('[data-demo="result"]').textContent(),
    fen: await frame.locator('[data-demo="fen"]').textContent(),
    boardSquares: await frame.locator("[data-square]").count(),
    chessShell: await page.locator(".chess-shell").count(),
    parentGenericShellText: (await page.locator("body").textContent()).includes("trusted shell"),
  };
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
  const submitted = await chessAction(run, fn, args);
  const key = accepted(submitted);
  const ev = await watched;
  if (ev.key !== key) throw new Error(`watched ${ev.key}, expected ${key}`);
  return resolve(ev);
}

async function deniedRead(run, key, reason) {
  try {
    await chessAction(run, "read", [key]);
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    if (message.includes(reason)) return reason;
    throw new Error(`expected read denial ${reason}, got ${message}`);
  }
  throw new Error(`expected read denial ${reason}`);
}

function resolve(ev, key = stateKey, nextBoard = board) {
  const actionKey = ev.key;
  const act = ev.value;
  const payload = act.payload || {};
  const before = state(key);
  const game = JSON.parse(JSON.stringify(before.value));
  let reason = "";
  if (payload.board !== nextBoard) reason = "wrong-board";
  else if (payload.type === "join") reason = join(game, act.participantId, payload.name);
  else if (payload.type === "move") reason = move(game, act.participantId, payload);
  else if (payload.type === "resign") reason = resign(game, act.participantId);
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

function join(game, participantId, name) {
  const clean = String(name || "").trim();
  if (!clean) return "missing-name";
  if (game.players.white?.participantId === participantId || game.players.black?.participantId === participantId) return "already-joined";
  if (!game.players.white) {
    game.players.white = { participantId, name: clean };
    return "";
  }
  if (!game.players.black) {
    game.players.black = { participantId, name: clean };
    game.status = "active";
    return "";
  }
  return "board-full";
}

function colorFor(game, participantId) {
  if (game.players.white?.participantId === participantId) return "white";
  if (game.players.black?.participantId === participantId) return "black";
  return "";
}

function move(game, participantId, payload) {
  if (game.status !== "active") return "game-not-active";
  const color = colorFor(game, participantId);
  if (!color) return "not-player";
  if (game.turn !== color) return "wrong-turn";
  const c = new Chess(game.fen);
  let mv;
  try {
    mv = c.move({ from: payload.from, to: payload.to, promotion: payload.promotion || "q" });
  } catch {
    return "illegal-move";
  }
  if (!mv) return "illegal-move";
  game.fen = c.fen();
  game.turn = c.turn() === "w" ? "white" : "black";
  game.history.push({ from: mv.from, to: mv.to, san: mv.san, color, participantId, fen: game.fen });
  if (c.isCheckmate()) {
    game.status = "checkmate";
    game.result = { status: "checkmate", winner: color, loser: color === "white" ? "black" : "white", san: mv.san };
  } else if (c.isStalemate()) {
    game.status = "draw";
    game.result = { status: "stalemate", winner: "" };
  } else if (c.isDraw()) {
    game.status = "draw";
    game.result = { status: "draw", winner: "" };
  }
  return "";
}

function resign(game, participantId) {
  const color = colorFor(game, participantId);
  if (!color) return "not-player";
  if (game.result?.status) return "game-over";
  const winner = color === "white" ? "black" : "white";
  game.status = "resigned";
  game.result = { status: "resigned", winner, loser: color };
  return "";
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
  setup();
  actionWatcher = startActionWatcher();
  const actionLog = [];
  const denialLog = [];
  const browser = await chromium.launch({ headless: true });
	  try {
	    const alice = await openChess(browser, "alice", "Alice");
	    const bob = await openChess(browser, "bob", "Bob");
	    const runs = [alice, bob];
	    const nameTypingStable = [
	      await proveNameTyping(alice, "Alice"),
	      await proveNameTyping(bob, "Bob"),
	    ];
	    const initialRevision = state().revision;

    actionLog.push(await submitReactVisible(runs, alice, "join", ["Alice", board]));

    denialLog.push({ kind: "duplicate-join", ...(await submitReactVisible(runs, alice, "join", ["Alice Again", board])) });

    actionLog.push(await submitReactVisible(runs, bob, "join", ["Bob", board]));
    const touchMoveProof = await touchProbe(alice);

    denialLog.push({ kind: "wrong-board-read", reason: await deniedRead(alice, `${stateKey}-wrong`, "chess board read denied") });

    denialLog.push({ kind: "wrong-board", ...(await submitReactVisible(runs, bob, "join", ["Bob Elsewhere", `${board}-wrong`])) });

    denialLog.push({ kind: "wrong-turn", ...(await submitReactVisible(runs, bob, "move", ["e7", "e5"])) });

    denialLog.push({ kind: "illegal-move", ...(await submitReactVisible(runs, alice, "move", ["e2", "e5"])) });

    actionLog.push(await submitReactVisible(runs, alice, "move", ["f2", "f3", "q", { actionId: "white-f3" }]));

    denialLog.push({
      kind: "duplicate-action",
      reason: denied(await chessAction(alice, "move", ["g2", "g3", "q", { actionId: "white-f3" }]), "duplicate-action"),
    });

    denialLog.push({
      kind: "stale-revision",
      reason: denied(await chessAction(bob, "move", ["a7", "a6", "q", { actionId: "black-stale", baseRevision: initialRevision }]), "stale-revision"),
    });

    for (const [run, from, to, id] of [
      [bob, "e7", "e5", "black-e5"],
      [alice, "g2", "g4", "white-g4"],
      [bob, "d8", "h4", "black-qh4"],
    ]) {
      actionLog.push(await submitReactVisible(runs, run, "move", [from, to, "q", { actionId: id }]));
    }

    await Promise.all(runs.map((run) =>
      run.frame.waitForFunction(() => window.__tinkabotChess.snapshot().result?.status === "checkmate", null, { timeout: timeoutMs })
    ));
    const participants = await snapshots(runs);
    const final = state();
    const visibleLatencies = [...actionLog, ...denialLog].map((x) => x.stateVisibleMs).filter((x) => Number.isFinite(x));
    const stateVisibleP95Ms = pct(visibleLatencies, 95);
    const stateVisibleP99Ms = pct(visibleLatencies, 99);
    const generatedStateReads = stateReadDispatches(participants, stateKey);
    const generatedIframePollingCount = generatedStateReads.length;
    const stateDelivery = participants.every((part) => part.chess.delivery === "trusted-shell.nats-watch.push")
      ? "trusted-shell.nats-watch.push"
      : "missing";
    const visualSmoke = {
      pass: participants.every((p) => p.dom.boardSquares === 64 && p.dom.chessShell === 1 && p.dom.parentGenericShellText === false),
      desktopPages: participants.length,
    };
    const text = JSON.stringify({ participants, actionLog, denialLog, final });
    const leaks = leakTerms(text);
    const proof = {
      kind: "tinkabot.realChessAppProof.v1",
      route,
      shellUrl,
      localShellUrl,
      board,
      stateKey,
      manualBoard,
      manualStateKey,
      players: {
        white: final.value.players.white?.name || "",
        black: final.value.players.black?.name || "",
      },
      terminal: final.value.result,
      participants: participants.map((part) => ({
        participant: part.participant,
        url: part.url,
        dom: part.dom,
        chess: part.chess,
        shellState: part.shellProof?.state,
      })),
	      actionLog,
	      denialLog,
	      reactionMode: "tinkalet.watch.prefix",
		      stateDelivery,
		      generatedIframePollingCount,
		      generatedIframePollingProof: {
		        source: "trusted-shell.dispatched",
		        stateReadCommands: generatedStateReads,
		      },
		      stateVisibleP95Ms,
	      stateVisibleP99Ms,
	      touchMoveProof,
	      visualSmoke,
	      nameTypingStable: nameTypingStable.every(Boolean),
	      finalBoard: final.value,
      authorityLeakCount: leaks.length,
      authorityLeakTerms: leaks,
      platformChessAPIAdditions: 0,
      legalityEngine: "chess.js@1.4.0",
    };
    proof.pass = proof.route === "tailscale" &&
      proof.players.white === "Alice" &&
      proof.players.black === "Bob" &&
      proof.terminal?.status === "checkmate" &&
      proof.terminal?.winner === "black" &&
      proof.denialLog.some((d) => d.kind === "wrong-turn" && d.reason === "wrong-turn") &&
      proof.denialLog.some((d) => d.kind === "illegal-move" && d.reason === "illegal-move") &&
      proof.denialLog.some((d) => d.kind === "wrong-board" && d.reason === "wrong-board") &&
      proof.denialLog.some((d) => d.kind === "wrong-board-read" && d.reason === "chess board read denied") &&
      proof.denialLog.some((d) => d.kind === "duplicate-action" && d.reason === "duplicate-action") &&
	      proof.denialLog.some((d) => d.kind === "stale-revision" && d.reason === "stale-revision") &&
	      [...proof.actionLog, ...proof.denialLog.filter((d) => d.actionKey)].every((a) => a.watchKey === a.actionKey && a.watchKind === "tinkabot.appAction.v1") &&
	      proof.participants.every((p) => p.chess.result?.status === "checkmate" && p.dom.boardSquares === 64 && p.dom.chessShell === 1 && p.dom.parentGenericShellText === false) &&
	      proof.stateDelivery === "trusted-shell.nats-watch.push" &&
	      proof.generatedIframePollingCount === 0 &&
	      proof.stateVisibleP95Ms <= 100 &&
	      proof.stateVisibleP99Ms <= 250 &&
	      proof.touchMoveProof.pass === true &&
	      proof.visualSmoke.pass === true &&
	      proof.nameTypingStable === true &&
	      proof.authorityLeakCount === 0 &&
      proof.platformChessAPIAdditions === 0;
    fs.writeFileSync(proofPath, `${JSON.stringify(proof, null, 2)}\n`);
    console.log(`real chess proof ${proof.route} board ${proof.board} winner ${proof.terminal?.winner} denials ${proof.denialLog.length}`);
    if (!proof.pass) throw new Error(`real chess proof failed: ${JSON.stringify(proof)}`);
  } finally {
    actionWatcher?.stop();
    await browser.close();
  }
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
JS

grep -q '"kind": "tinkabot.realChessAppProof.v1"' "$proof" || fail "chess proof kind missing"
grep -q '"pass": true' "$proof" || fail "chess proof did not pass"
grep -q '"authorityLeakCount": 0' "$proof" || fail "chess proof recorded authority leaks"

log_step "demo passed"
printf 'package root %s\n' "$pkg"
printf 'artifacts %s\n' "$dist"
printf 'chess proof %s\n' "$proof"

if [[ "${TINKABOT_DEMO_HOLD:-}" == "1" ]]; then
  manual_board="$(cat "$work/manual-board")"
  manual_state_key="$(cat "$work/manual-state-key")"
  manual_log="$dist/chess-manual-reducer.log"
  log_step "start manual chess reducer for $manual_board"
  env -i \
    HOME="$work/owner/home" \
    TINKALET_CONFIG_DIR="$work/owner/cfg" \
    TINKALET_DATA_DIR="$work/owner/data" \
    PATH="/nonexistent" \
    TINKALET="$pkg/tinkalet" \
    STATE_KEY="$manual_state_key" \
    BOARD_NO="$manual_board" \
    CHESS_JS="$root/apps/frontend/node_modules/chess.js" \
    CURSOR="chessmanual$(date -u +%H%M%S)" \
    /home/lagz0ne/.local/share/mise/installs/node/latest/bin/node >"$manual_log" 2>&1 <<'JS' &
const { execFileSync } = require("node:child_process");
const { Chess } = require(process.env.CHESS_JS);

const tinkalet = process.env.TINKALET;
const stateKey = process.env.STATE_KEY;
const boardNo = process.env.BOARD_NO;
const cursor = process.env.CURSOR;
const seen = new Set();

function run(args) {
  return execFileSync(tinkalet, args, { encoding: "utf8", env: process.env });
}
function json(args) { return JSON.parse(run(args)); }
function state() { return json(["item", "get", stateKey, "--json"]); }
function colorFor(game, participantId) {
  if (game.players.white?.participantId === participantId) return "white";
  if (game.players.black?.participantId === participantId) return "black";
  return "";
}
function join(game, participantId, name) {
  const clean = String(name || "").trim();
  if (!clean) return "missing-name";
  if (game.players.white?.participantId === participantId || game.players.black?.participantId === participantId) return "already-joined";
  if (!game.players.white) {
    game.players.white = { participantId, name: clean };
    return "";
  }
  if (!game.players.black) {
    game.players.black = { participantId, name: clean };
    game.status = "active";
    return "";
  }
  return "board-full";
}
function move(game, participantId, payload) {
  if (game.status !== "active") return "game-not-active";
  const color = colorFor(game, participantId);
  if (!color) return "not-player";
  if (game.turn !== color) return "wrong-turn";
  const c = new Chess(game.fen);
  let mv;
  try {
    mv = c.move({ from: payload.from, to: payload.to, promotion: payload.promotion || "q" });
  } catch {
    return "illegal-move";
  }
  if (!mv) return "illegal-move";
  game.fen = c.fen();
  game.turn = c.turn() === "w" ? "white" : "black";
  game.history.push({ from: mv.from, to: mv.to, san: mv.san, color, participantId, fen: game.fen });
  if (c.isCheckmate()) {
    game.status = "checkmate";
    game.result = { status: "checkmate", winner: color, loser: color === "white" ? "black" : "white", san: mv.san };
  } else if (c.isStalemate()) {
    game.status = "draw";
    game.result = { status: "stalemate", winner: "" };
  } else if (c.isDraw()) {
    game.status = "draw";
    game.result = { status: "draw", winner: "" };
  }
  return "";
}
function resign(game, participantId) {
  const color = colorFor(game, participantId);
  if (!color) return "not-player";
  if (game.result?.status) return "game-over";
  const winner = color === "white" ? "black" : "white";
  game.status = "resigned";
  game.result = { status: "resigned", winner, loser: color };
  return "";
}
function resolve(key, act) {
  if (seen.has(key)) return;
  seen.add(key);
  const before = state();
  const game = JSON.parse(JSON.stringify(before.value));
  const payload = act.payload || {};
  let reason = "";
  if (payload.board !== boardNo) reason = "wrong-board";
  else if (payload.type === "join") reason = join(game, act.participantId, payload.name);
  else if (payload.type === "move") reason = move(game, act.participantId, payload);
  else if (payload.type === "resign") reason = resign(game, act.participantId);
  else reason = "unknown-action";
  if (reason) {
    const receipt = json(["action", "reject", key, "--reason", reason, "--json"]);
    console.log(JSON.stringify({ at: new Date().toISOString(), action: key, outcome: "rejected", reason, receipt: receipt.key }));
    return;
  }
  const receipt = json(["action", "apply", key, "--value", JSON.stringify(game), "--json"]);
  const after = state();
  console.log(JSON.stringify({ at: new Date().toISOString(), action: key, outcome: "applied", status: after.value.status, turn: after.value.turn, result: after.value.result, receipt: receipt.key }));
}
console.log(JSON.stringify({ at: new Date().toISOString(), reducer: "started", stateKey, boardNo, cursor }));
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
  alice_url="$public_shell_url/?tb_app=demo&tb_participant=alice&tb_state=$manual_state_key&tb_session=demo-001&tb_chess=1&tb_board_no=$manual_board"
  bob_url="$public_shell_url/?tb_app=demo&tb_participant=bob&tb_state=$manual_state_key&tb_session=demo-001&tb_chess=1&tb_board_no=$manual_board"
  manual_smoke="$dist/chess-manual-smoke.json"
  log_step "smoke manual chess links"
  PLAYWRIGHT_MODULE="$root/apps/frontend/node_modules/playwright" \
    ALICE_URL="$alice_url" \
    BOB_URL="$bob_url" \
    MANUAL_SMOKE="$manual_smoke" \
    /home/lagz0ne/.local/share/mise/installs/node/latest/bin/node <<'JS'
const fs = require("node:fs");
const { chromium } = require(process.env.PLAYWRIGHT_MODULE);
const links = [
  ["alice", process.env.ALICE_URL],
  ["bob", process.env.BOB_URL],
];

(async () => {
  const browser = await chromium.launch({ headless: true });
  const out = [];
  try {
    for (const [participant, url] of links) {
      const page = await browser.newPage({ viewport: { width: 1120, height: 760 } });
      await page.goto(url, { waitUntil: "domcontentloaded", timeout: 60000 });
      await page.waitForSelector("iframe", { timeout: 60000 });
      const frame = page.frames().find((f) => f.url().startsWith("blob:"));
      if (!frame) throw new Error(`missing generated frame for ${participant}`);
      await frame.waitForSelector('[data-chess="board"] [data-square]', { timeout: 60000 });
	      const item = {
	        participant,
	        url,
	        title: await frame.locator('[data-chess="title"]').textContent(),
	        status: await frame.locator('[data-chess="status"]').textContent(),
        boardSquares: await frame.locator("[data-square]").count(),
        chessShell: await page.locator(".chess-shell").count(),
	        parentGenericShellText: (await page.locator("body").textContent()).includes("trusted shell"),
	      };
	      const input = frame.locator('[data-chess="name"]');
	      await input.fill("");
	      await input.focus();
	      await page.keyboard.type(participant, { delay: 160 });
	      await page.waitForTimeout(800);
	      item.nameValue = await input.inputValue();
	      out.push(item);
	      await page.close();
    }
  } finally {
    await browser.close();
  }
	  const pass = out.every((p) => p.status === "waiting" && p.boardSquares === 64 && p.chessShell === 1 && p.parentGenericShellText === false && p.nameValue === p.participant);
  fs.writeFileSync(process.env.MANUAL_SMOKE, `${JSON.stringify({ kind: "tinkabot.realChessManualSmoke.v1", pass, pages: out }, null, 2)}\n`);
  if (!pass) throw new Error(`manual chess smoke failed: ${JSON.stringify(out)}`);
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
JS
  printf 'manual-board %s\n' "$manual_board"
  printf 'manual-reducer %s\n' "$manual_log"
  printf 'manual-smoke %s\n' "$manual_smoke"
  printf 'alice %s\n' "$alice_url"
  printf 'bob %s\n' "$bob_url"
  log_step "holding chess app open"
  while kill -0 "$pid" 2>/dev/null; do
    sleep 3600 &
    wait $! || true
  done
fi
