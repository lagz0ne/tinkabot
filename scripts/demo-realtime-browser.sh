#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="${1:-$(mktemp -d /tmp/tinkabot-realtime-browser-demo.XXXXXX)}"
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
  echo "demo-realtime-browser: $*" >&2
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

store="$(mktemp -d /tmp/tinkabot-realtime-browser-store.XXXXXX)"
cfg="$(mktemp -d /tmp/tinkalet-realtime-browser-config.XXXXXX)"
data="$(mktemp -d /tmp/tinkalet-realtime-browser-data.XXXXXX)"
home="$(mktemp -d /tmp/tinkalet-realtime-browser-home.XXXXXX)"
log="$dist/tinkabot.log"
err="$dist/tinkabot.err"
proof="$dist/realtime-browser-proof.json"
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

run_tinkalet() {
  env -i \
    HOME="$home" \
    TINKALET_CONFIG_DIR="$cfg" \
    TINKALET_DATA_DIR="$data" \
    PATH="/nonexistent" \
    "$pkg/tinkalet" "$@"
}

log_step "start packaged Tinkabot with demo session and scoped participants"
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

log_step "seed app state through packaged Tinkalet owner profile"
mv "$pkg/libexec/tinkabot/nats" "$pkg/libexec/tinkabot/nats.disabled"
got="$(run_tinkalet profile import local --store "$store" --name owner)"
[[ "$got" == "profile owner imported" ]] || fail "owner profile import drift: $got"
got="$(run_tinkalet profile use owner)"
[[ "$got" == "profile owner selected" ]] || fail "owner profile use drift: $got"
run_tinkalet item create apps.demo.state.browser-alice --value '{"seq":0,"participant":"alice"}' --json >/dev/null
run_tinkalet item create apps.demo.state.browser-bob --value '{"seq":0,"participant":"bob"}' --json >/dev/null

if [[ "$public_shell_url" != "$local_shell_url" ]]; then
  for _ in {1..50}; do
    if curl -fsS "$public_shell_url" >/dev/null 2>/dev/null; then
      break
    fi
    sleep 0.1
  done
  curl -fsS "$public_shell_url" >/dev/null || fail "Tailscale shell URL is not reachable"
fi

log_step "drive two browser participants through the trusted shell"
PLAYWRIGHT_MODULE="$root/apps/frontend/node_modules/playwright" \
  TINKABOT_DEMO_SHELL_URL="$public_shell_url" \
  TINKABOT_DEMO_LOCAL_SHELL_URL="$local_shell_url" \
  TINKABOT_DEMO_BROWSER_OUT="$proof" \
  TINKABOT_DEMO_BROWSER_ROUTE="$([[ "$public_shell_url" == "$local_shell_url" ]] && printf local || printf tailscale)" \
  TINKABOT_DEMO_BROWSER_ACTIONS="${TINKABOT_DEMO_BROWSER_ACTIONS:-20}" \
  TINKABOT_DEMO_BROWSER_INTERVAL_MS="${TINKABOT_DEMO_BROWSER_INTERVAL_MS:-20}" \
  TINKABOT_DEMO_BROWSER_TIMEOUT_MS="${TINKABOT_DEMO_BROWSER_TIMEOUT_MS:-60000}" \
  TINKABOT_DEMO_BROWSER_ACTION_P95_MS="${TINKABOT_DEMO_BROWSER_ACTION_P95_MS:-750}" \
  TINKABOT_DEMO_BROWSER_ACTION_P99_MS="${TINKABOT_DEMO_BROWSER_ACTION_P99_MS:-1500}" \
  TINKABOT_DEMO_BROWSER_READBACK_P95_MS="${TINKABOT_DEMO_BROWSER_READBACK_P95_MS:-750}" \
  TINKABOT_DEMO_BROWSER_READBACK_P99_MS="${TINKABOT_DEMO_BROWSER_READBACK_P99_MS:-1500}" \
  node <<'JS'
const fs = require("node:fs");
const { chromium } = require(process.env.PLAYWRIGHT_MODULE);

const participants = ["alice", "bob"];
const actions = Number(process.env.TINKABOT_DEMO_BROWSER_ACTIONS);
const intervalMs = Number(process.env.TINKABOT_DEMO_BROWSER_INTERVAL_MS);
const timeoutMs = Number(process.env.TINKABOT_DEMO_BROWSER_TIMEOUT_MS);
const thresholds = {
  actionP95Ms: Number(process.env.TINKABOT_DEMO_BROWSER_ACTION_P95_MS),
  actionP99Ms: Number(process.env.TINKABOT_DEMO_BROWSER_ACTION_P99_MS),
  readbackP95Ms: Number(process.env.TINKABOT_DEMO_BROWSER_READBACK_P95_MS),
  readbackP99Ms: Number(process.env.TINKABOT_DEMO_BROWSER_READBACK_P99_MS),
};

if (!Number.isInteger(actions) || actions <= 0) {
  throw new Error(`invalid TINKABOT_DEMO_BROWSER_ACTIONS=${process.env.TINKABOT_DEMO_BROWSER_ACTIONS}`);
}
if (!Number.isInteger(intervalMs) || intervalMs <= 0) {
  throw new Error(`invalid TINKABOT_DEMO_BROWSER_INTERVAL_MS=${process.env.TINKABOT_DEMO_BROWSER_INTERVAL_MS}`);
}

function percentile(values, pct) {
  if (values.length === 0) return null;
  const sorted = [...values].sort((a, b) => a - b);
  const idx = Math.min(sorted.length - 1, Math.ceil((pct / 100) * sorted.length) - 1);
  return sorted[idx];
}

function pageUrl(participant) {
  const url = new URL(process.env.TINKABOT_DEMO_SHELL_URL);
  url.searchParams.set("tb_app", "demo");
  url.searchParams.set("tb_participant", participant);
  url.searchParams.set("tb_state", `apps.demo.state.browser-${participant}`);
  url.searchParams.set("tb_session", "demo-001");
  url.searchParams.set("tb_auto", String(actions));
  url.searchParams.set("tb_interval_ms", String(intervalMs));
  return url.toString();
}

function proofCounts(proof, participant) {
  const dispatches = Array.isArray(proof?.dispatched) ? proof.dispatched : [];
  const acceptedActions = dispatches.filter((d) => d.command === "participant_action" && d.status === "accepted").length;
  const stateReads = dispatches.filter((d) =>
    d.command === "participant_read" &&
    d.status === "accepted" &&
    d.itemKey === `apps.demo.state.browser-${participant}`
  ).length;
  const readbacks = dispatches.filter((d) =>
    d.command === "participant_read" &&
    d.status === "accepted" &&
    typeof d.itemKey === "string" &&
    d.itemKey.includes(`apps.demo.participants.${participant}.actions.`)
  ).length;
  const denied = dispatches.filter((d) => d.status !== "accepted").length;
  return { acceptedActions, stateReads, readbacks, denied };
}

async function runParticipant(browser, participant) {
  const page = await browser.newPage({ viewport: { width: 980, height: 720 } });
  const url = pageUrl(participant);
  const started = Date.now();
  await page.goto(url, { waitUntil: "domcontentloaded" });
  await page.waitForSelector("iframe", { timeout: timeoutMs });
  await page.waitForFunction((input) => {
    const proof = window.__tinkabotProof;
    if (!proof) return false;
    const dispatches = Array.isArray(proof.dispatched) ? proof.dispatched : [];
    const actions = dispatches.filter((d) => d.command === "participant_action" && d.status === "accepted").length;
    const readbacks = dispatches.filter((d) =>
      d.command === "participant_read" &&
      d.status === "accepted" &&
      typeof d.itemKey === "string" &&
      d.itemKey.includes(`apps.demo.participants.${input.participant}.actions.`)
    ).length;
    return actions >= input.actions && readbacks >= input.actions;
  }, { participant, actions }, { timeout: timeoutMs });

  const frame = page.frames().find((f) => f.url().startsWith("blob:"));
  if (!frame) throw new Error(`generated frame missing for ${participant}`);
  await frame.waitForFunction(() => document.querySelector("#generated")?.dataset.complete === "true", null, { timeout: 10000 });
  const dom = {
    title: await frame.locator('[data-demo="title"]').textContent(),
    status: await frame.locator('[data-demo="status"]').textContent(),
    actions: await numberText(frame, '[data-demo="actions"]'),
    readbacks: await numberText(frame, '[data-demo="readbacks"]'),
    denied: await numberText(frame, '[data-demo="denied"]'),
    text: await frame.locator("#generated").textContent(),
  };
  const proof = await page.evaluate(() => window.__tinkabotProof);
  return {
    participant,
    url,
    elapsedMs: Date.now() - started,
    dom,
    counts: proofCounts(proof, participant),
    proof,
  };
}

async function numberText(frame, selector) {
  const text = await frame.locator(selector).textContent();
  const value = Number(text);
  if (!Number.isFinite(value)) throw new Error(`expected numeric ${selector}, got ${JSON.stringify(text)}`);
  return value;
}

async function assertShellEscapesSession(browser) {
  const payload = `"><img src=x onerror="window.__tinkabotXss=1">`;
  const url = new URL(process.env.TINKABOT_DEMO_SHELL_URL);
  url.searchParams.set("tb_session", payload);
  const page = await browser.newPage({ viewport: { width: 800, height: 520 } });
  try {
    await page.goto(url.toString(), { waitUntil: "domcontentloaded" });
    await page.waitForSelector('[data-obs="sid"]', { timeout: timeoutMs });
    const value = await page.locator('[data-obs="sid"]').inputValue();
    const injectedImages = await page.locator("img").count();
    const xssFlag = await page.evaluate(() => Boolean(window.__tinkabotXss));
    if (value !== payload || injectedImages !== 0 || xssFlag) {
      throw new Error(`session query escaped incorrectly: value=${JSON.stringify(value)} images=${injectedImages} xss=${xssFlag}`);
    }
    return { valueRoundTrip: true, injectedImages, xssFlag };
  } finally {
    await page.close();
  }
}

function leakCount(text) {
  const lower = text.toLowerCase();
  const tokens = [
    "tb_items",
    "$kv",
    "begin nats",
    "private key",
    "nats://",
    ".creds",
    "jwt",
    "nkey",
    "seed",
    "bearer",
    "credential",
    "credentials",
    "token",
  ];
  return tokens.filter((token) => lower.includes(token)).length;
}

(async () => {
  const browser = await chromium.launch({ headless: true });
  try {
    const shellInjection = await assertShellEscapesSession(browser);
    const runs = await Promise.all(participants.map((participant) => runParticipant(browser, participant)));
    const dispatches = runs.flatMap((run) => run.proof.dispatched || []);
    const actionLatencies = dispatches
      .filter((d) => d.command === "participant_action" && d.status === "accepted")
      .map((d) => d.latencyMs)
      .filter((v) => Number.isFinite(v));
    const readbackLatencies = dispatches
      .filter((d) => d.command === "participant_read" && d.status === "accepted" && typeof d.itemKey === "string" && d.itemKey.includes(".actions."))
      .map((d) => d.latencyMs)
      .filter((v) => Number.isFinite(v));
    const text = JSON.stringify(runs);
    const stats = {
      kind: "tinkabot.realtimeBrowserParticipantProof.v1",
      route: process.env.TINKABOT_DEMO_BROWSER_ROUTE,
      shellUrl: process.env.TINKABOT_DEMO_SHELL_URL,
      localShellUrl: process.env.TINKABOT_DEMO_LOCAL_SHELL_URL,
      actionsPerParticipant: actions,
      intervalMs,
      participants: runs.map((run) => ({
        participant: run.participant,
        url: run.url,
        elapsedMs: run.elapsedMs,
        dom: run.dom,
        counts: run.counts,
      })),
      shellInjection,
      browserPages: runs.length,
      acceptedActions: actionLatencies.length,
      readbacks: readbackLatencies.length,
      deniedDispatches: dispatches.filter((d) => d.status !== "accepted").length,
      latencyMs: {
        action: {
          p95: percentile(actionLatencies, 95),
          p99: percentile(actionLatencies, 99),
          max: actionLatencies.length ? Math.max(...actionLatencies) : null,
        },
        readback: {
          p95: percentile(readbackLatencies, 95),
          p99: percentile(readbackLatencies, 99),
          max: readbackLatencies.length ? Math.max(...readbackLatencies) : null,
        },
      },
      authorityLeakCount: leakCount(text),
      thresholds,
    };
    stats.pass = stats.browserPages === 2 &&
      stats.acceptedActions === actions * 2 &&
      stats.readbacks === actions * 2 &&
      stats.deniedDispatches === 0 &&
      stats.authorityLeakCount === 0 &&
      stats.latencyMs.action.p95 !== null &&
      stats.latencyMs.action.p95 <= thresholds.actionP95Ms &&
      stats.latencyMs.action.p99 <= thresholds.actionP99Ms &&
      stats.latencyMs.readback.p95 !== null &&
      stats.latencyMs.readback.p95 <= thresholds.readbackP95Ms &&
      stats.latencyMs.readback.p99 <= thresholds.readbackP99Ms &&
      stats.shellInjection.valueRoundTrip === true &&
      stats.shellInjection.injectedImages === 0 &&
      stats.shellInjection.xssFlag === false &&
      runs.every((run) =>
        run.counts.acceptedActions === actions &&
        run.counts.stateReads >= 1 &&
        run.counts.readbacks === actions &&
        run.counts.denied === 0 &&
        run.dom.status === "complete" &&
        run.dom.actions === actions &&
        run.dom.readbacks === actions &&
        run.dom.denied === 0
      );

    fs.writeFileSync(process.env.TINKABOT_DEMO_BROWSER_OUT, `${JSON.stringify(stats, null, 2)}\n`);
    console.log(`browser participant proof ${stats.route} actions ${stats.acceptedActions} readbacks ${stats.readbacks} action p95 ${stats.latencyMs.action.p95} readback p95 ${stats.latencyMs.readback.p95}`);
    if (!stats.pass) {
      throw new Error(`browser participant proof failed: ${JSON.stringify(stats)}`);
    }
  } finally {
    await browser.close();
  }
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
JS

log_step "demo passed"
printf 'package root %s\n' "$pkg"
printf 'artifacts %s\n' "$dist"
printf 'browser proof %s\n' "$proof"
printf 'open %s\n' "$public_shell_url"

if [[ "${TINKABOT_DEMO_HOLD:-}" == "1" ]]; then
  log_step "holding demo shell open"
  while kill -0 "$pid" 2>/dev/null; do
    sleep 3600 &
    wait $! || true
  done
fi
