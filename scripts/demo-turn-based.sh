#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="${1:-$(mktemp -d /tmp/tinkabot-turn-demo.XXXXXX)}"
case "$dist" in
  /*) ;;
  *) dist="$root/$dist" ;;
esac

log_step() {
  printf '\n[%s] %s\n' "$(date -u +%H:%M:%S)" "$*"
}

fail() {
  echo "demo-turn-based: $*" >&2
  if [[ -n "${log:-}" && -f "$log" ]]; then
    sed -n '1,120p' "$log" >&2 || true
  fi
  if [[ -n "${err:-}" && -f "$err" ]]; then
    sed -n '1,120p' "$err" >&2 || true
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

store="$(mktemp -d /tmp/tinkabot-turn-store.XXXXXX)"
work="$dist/turn-work"
log="$dist/tinkabot.log"
err="$dist/tinkabot.err"
proof="$dist/turn-proof.json"
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

log_step "drive turn sequence through packaged Tinkalet"
TURN_PKG="$pkg" \
TURN_STORE="$store" \
TURN_WORK="$work" \
TURN_PROOF="$proof" \
node <<'JS'
const { execFileSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const pkg = process.env.TURN_PKG;
const store = process.env.TURN_STORE;
const work = process.env.TURN_WORK;
const proofPath = process.env.TURN_PROOF;
const tinkalet = path.join(pkg, "tinkalet");
const stateKey = "apps.demo.state.board";
const proof = [];

for (const who of ["owner", "alice", "bob"]) {
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
    return execFileSync(tinkalet, args, { encoding: "utf8", env: envFor(who) });
  } catch (err) {
    throw new Error(`${who} tinkalet ${args.join(" ")} failed ${err.status}\nstdout=${err.stdout || ""}\nstderr=${err.stderr || ""}`);
  }
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

function expect(line, got, want) {
  if (got !== want) {
    throw new Error(`${line}: got ${got}, want ${want}`);
  }
}

function setup(who, source, name) {
  expect(`${who} import`, run(who, ["profile", "import", "local", "--store", source, "--name", name]), `profile ${name} imported\n`);
  expect(`${who} use`, run(who, ["profile", "use", name]), `profile ${name} selected\n`);
}

function submit(who, id, base, cell) {
  return jsonRun(who, ["action", "submit", id, "--state", stateKey, "--base-revision", String(base), "--value", JSON.stringify({ cell }), "--json"]);
}

function state() {
  return jsonRun("owner", ["item", "get", stateKey, "--json"]);
}

function apply(action, before, participant, cell) {
  const board = before.value;
  if (board.winner || board.turn !== participant || board.cells[cell]) {
    throw new Error(`illegal apply attempted for ${action.key}: ${JSON.stringify(board)}`);
  }
  board.cells[cell] = participant;
  board.winner = winner(board, participant) ? participant : undefined;
  if (!board.winner) {
    board.turn = participant === "alice" ? "bob" : "alice";
  }
  const receipt = jsonRun("owner", ["action", "apply", action.key, "--value", JSON.stringify(board), "--json"]);
  expect("apply receipt status", receipt.status, "resolved");
  expect("apply receipt outcome", receipt.value.outcome, "applied");
  const after = state();
  if (after.revision <= before.revision || JSON.stringify(after.value) !== JSON.stringify(board)) {
    throw new Error(`state did not apply ${action.key}: ${JSON.stringify({ before, after, board })}`);
  }
  proof.push({ step: "apply", action: action.key, participant, cell, stateRevision: after.revision });
  return after;
}

function reject(action, before, reason) {
  const receipt = jsonRun("owner", ["action", "reject", action.key, "--reason", reason, "--json"]);
  expect("reject receipt status", receipt.status, "denied");
  expect("reject receipt outcome", receipt.value.outcome, "rejected");
  expect("reject receipt reason", receipt.value.reason, reason);
  const after = state();
  if (after.revision !== before.revision || JSON.stringify(after.value) !== JSON.stringify(before.value)) {
    throw new Error(`reject mutated state ${action.key}: ${JSON.stringify({ before, after })}`);
  }
  proof.push({ step: "reject", action: action.key, reason, stateRevision: after.revision });
  return after;
}

function winner(board, participant) {
  return [
    ["a1", "a2", "a3"],
    ["b1", "b2", "b3"],
    ["c1", "c2", "c3"],
    ["a1", "b1", "c1"],
    ["a2", "b2", "c2"],
    ["a3", "b3", "c3"],
    ["a1", "b2", "c3"],
    ["a3", "b2", "c1"],
  ].some((line) => line.every((cell) => board.cells[cell] === participant));
}

setup("owner", store, "owner");
setup("alice", path.join(store, "participants", "demo", "alice"), "alice");
setup("bob", path.join(store, "participants", "demo", "bob"), "bob");

let current = jsonRun("owner", ["item", "create", stateKey, "--value", JSON.stringify({ turn: "alice", cells: {} }), "--json"]);
const initialRevision = current.revision;
proof.push({ step: "create-state", revision: current.revision });

current = reject(submit("bob", "b-wrong-turn", current.revision, "b1"), current, "wrong-turn");
const aliceA1 = submit("alice", "a1", current.revision, "a1");
current = apply(aliceA1, current, "alice", "a1");

deny("alice", ["action", "submit", "a1", "--state", stateKey, "--base-revision", String(current.revision), "--value", JSON.stringify({ cell: "a1" })], "action a1 denied submit: duplicate-action\n");
deny("bob", ["action", "submit", "b-stale", "--state", stateKey, "--base-revision", String(initialRevision), "--value", JSON.stringify({ cell: "b1" })], "action b-stale denied submit: stale-revision\n");
deny("owner", ["item", "get", "apps.demo.participants.bob.actions.b-stale"], "item apps.demo.participants.bob.actions.b-stale denied get: item-not-found\n");
proof.push({ step: "substrate-denials", duplicate: "a1", stale: "b-stale" });

current = reject(submit("bob", "b-occupied", current.revision, "a1"), current, "occupied-cell");
current = apply(submit("bob", "b1", current.revision, "b1"), current, "bob", "b1");
current = apply(submit("alice", "a2", current.revision, "a2"), current, "alice", "a2");
current = apply(submit("bob", "b2", current.revision, "b2"), current, "bob", "b2");
current = apply(submit("alice", "a3", current.revision, "a3"), current, "alice", "a3");

if (current.value.winner !== "alice") {
  throw new Error(`winner drift: ${JSON.stringify(current.value)}`);
}
proof.push({ step: "complete", winner: current.value.winner, stateRevision: current.revision, board: current.value });
fs.writeFileSync(proofPath, `${JSON.stringify({ winner: current.value.winner, proof }, null, 2)}\n`);

console.log(`turn wrong-turn rejected at rev ${proof.find((p) => p.reason === "wrong-turn").stateRevision}`);
console.log("turn duplicate-action denied");
console.log("turn stale-revision denied");
console.log(`turn occupied-cell rejected at rev ${proof.find((p) => p.reason === "occupied-cell").stateRevision}`);
console.log(`turn complete winner ${current.value.winner} rev ${current.revision}`);
JS

grep -q '"winner": "alice"' "$proof" || fail "turn proof did not record winner"
grep -q '"reason": "wrong-turn"' "$proof" || fail "turn proof missing wrong-turn denial"
grep -q '"reason": "occupied-cell"' "$proof" || fail "turn proof missing occupied-cell denial"
grep -q '"stale": "b-stale"' "$proof" || fail "turn proof missing stale denial"

log_step "turn proof written to $proof"
printf 'proof %s\n' "$proof"
