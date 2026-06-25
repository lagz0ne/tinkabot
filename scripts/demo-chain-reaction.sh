#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="${1:-$(mktemp -d /tmp/tinkabot-chain-demo.XXXXXX)}"
case "$dist" in
  /*) ;;
  *) dist="$root/$dist" ;;
esac

log_step() {
  printf '\n[%s] %s\n' "$(date -u +%H:%M:%S)" "$*"
}

fail() {
  echo "demo-chain-reaction: $*" >&2
  if [[ -n "${log:-}" && -f "$log" ]]; then
    sed -n '1,120p' "$log" >&2 || true
  fi
  if [[ -n "${err:-}" && -f "$err" ]]; then
    sed -n '1,120p' "$err" >&2 || true
  fi
  exit 1
}

expect_eq() {
  local got="$1"
  local want="$2"
  if [[ "$got" != "$want" ]]; then
    fail "expected $want, got $got"
  fi
}

expect_prefix() {
  local got="$1"
  local want="$2"
  case "$got" in
    "$want"*) ;;
    *) fail "expected prefix $want, got $got" ;;
  esac
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
for file in tinkabot tinkalet libexec/tinkabot/bwrap libexec/tinkabot/nats; do
  [[ -x "$pkg/$file" ]] || fail "$file is not executable"
done

store="$(mktemp -d /tmp/tinkabot-demo-store.XXXXXX)"
cfg="$(mktemp -d /tmp/tinkalet-demo-config.XXXXXX)"
data="$(mktemp -d /tmp/tinkalet-demo-data.XXXXXX)"
home="$(mktemp -d /tmp/tinkalet-demo-home.XXXXXX)"
log="$dist/tinkabot.log"
err="$dist/tinkabot.err"
pid=""
react_pid=""
forward_pid=""

cleanup() {
  if [[ -n "$forward_pid" ]] && kill -0 "$forward_pid" 2>/dev/null; then
    kill -TERM "$forward_pid" 2>/dev/null || true
    wait "$forward_pid" 2>/dev/null || true
  fi
  if [[ -n "$react_pid" ]] && kill -0 "$react_pid" 2>/dev/null; then
    kill -TERM "$react_pid" 2>/dev/null || true
    wait "$react_pid" 2>/dev/null || true
  fi
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
    kill -TERM "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT

log_step "start packaged Tinkabot with the clock bundle"
(cd "$pkg" && PATH="/usr/bin:/bin" ./tinkabot --store "$store" --shell 127.0.0.1:0 --bundle examples/clock >"$log" 2>"$err") &
pid=$!

for _ in {1..200}; do
  if grep -q '^shell  http://127\.0\.0\.1:' "$log"; then
    break
  fi
  if ! kill -0 "$pid" 2>/dev/null; then
    fail "tinkabot exited before posture"
  fi
  sleep 0.1
done

shell_url="$(awk '$1 == "shell" { print $2; exit }' "$log")"
[[ -n "$shell_url" ]] || fail "shell URL missing"
nats_url="$(awk '$1 == "nats" { print $2; exit }' "$log")"
[[ -n "$nats_url" ]] || fail "NATS URL missing"
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
printf 'clock %s/artifacts/bundle/clock/index.html\n' "$public_shell_url"
printf 'local-shell %s\n' "$local_shell_url"
if [[ "$public_shell_url" != "$local_shell_url" ]]; then
  for _ in {1..50}; do
    if curl -fsS "$public_shell_url/artifacts/bundle/clock/index.html" >/dev/null 2>/dev/null; then
      break
    fi
    sleep 0.1
  done
  curl -fsS "$public_shell_url/artifacts/bundle/clock/index.html" >/dev/null || fail "Tailscale shell URL is not reachable"
fi

fast_every="${TINKABOT_DEMO_FAST_EVERY:-100ms}"
if [[ "$fast_every" != "manifest" ]]; then
  log_step "retune clock bundle to $fast_every for realtime sync probe"
  "$pkg/libexec/tinkabot/nats" --no-context \
    --server "$nats_url" \
    --creds "$store/caller.creds" \
    --timeout 2s \
    kv put config_bucket bundle.clock.tick.every "$fast_every" >/dev/null
fi

run_tinkalet() {
  env -i \
    HOME="$home" \
    TINKALET_CONFIG_DIR="$cfg" \
    TINKALET_DATA_DIR="$data" \
    PATH="/nonexistent" \
    "$pkg/tinkalet" "$@"
}

log_step "disable packaged nats sidecar before Tinkalet commands"
mv "$pkg/libexec/tinkabot/nats" "$pkg/libexec/tinkabot/nats.disabled"

log_step "import and select the local Tinkalet profile"
got="$(run_tinkalet profile import local --store "$store" --name local)"
expect_eq "$got" "profile local imported"
got="$(run_tinkalet profile use local)"
expect_eq "$got" "profile local selected"

log_step "accept bundle.clock.tick while realtime projection keeps moving"
before="$dist/projection-before.json"
after="$dist/projection-after.json"
for _ in {1..150}; do
  if curl -fsS "$local_shell_url/projections/bundle.clock.state" >"$before"; then
    break
  fi
  sleep 0.1
done
[[ -s "$before" ]] || fail "clock projection missing before trigger"

got="$(run_tinkalet trigger bundle.clock.tick --request-id demo-chain-reaction-1)"
expect_eq "$got" "profile local accepted bundle.clock.tick"

for _ in {1..150}; do
  if curl -fsS "$local_shell_url/projections/bundle.clock.state" >"$after" && ! cmp -s "$before" "$after"; then
    break
  fi
  sleep 0.1
done
if cmp -s "$before" "$after"; then
  fail "clock projection did not change after trigger"
fi

if [[ "${TINKABOT_DEMO_BROWSER_SYNC:-0}" == "1" ]]; then
  log_step "measure browser sync age through the generated clock UI"
  browser_sync="$dist/realtime-browser-sync.json"
  PLAYWRIGHT_MODULE="$root/apps/frontend/node_modules/playwright" \
    TINKABOT_DEMO_APP_URL="$public_shell_url/artifacts/bundle/clock/index.html" \
    TINKABOT_DEMO_BROWSER_SYNC_OUT="$browser_sync" \
    TINKABOT_DEMO_BROWSER_ROUTE="$([[ "$public_shell_url" == "$local_shell_url" ]] && printf local || printf tailscale)" \
    TINKABOT_DEMO_BROWSER_SYNC_MS="${TINKABOT_DEMO_BROWSER_SYNC_MS:-6000}" \
    TINKABOT_DEMO_BROWSER_SAMPLE_MS="${TINKABOT_DEMO_BROWSER_SAMPLE_MS:-50}" \
    TINKABOT_DEMO_BROWSER_AGE_P95_MS="${TINKABOT_DEMO_BROWSER_AGE_P95_MS:-250}" \
    TINKABOT_DEMO_BROWSER_AGE_P99_MS="${TINKABOT_DEMO_BROWSER_AGE_P99_MS:-500}" \
    TINKABOT_DEMO_FILTER_P95_MS="${TINKABOT_DEMO_FILTER_P95_MS:-50}" \
    TINKABOT_DEMO_SOURCE_INTERVAL_P95_MS="${TINKABOT_DEMO_SOURCE_INTERVAL_P95_MS:-150}" \
    TINKABOT_DEMO_BROWSER_WARMUP_MS="${TINKABOT_DEMO_BROWSER_WARMUP_MS:-30000}" \
    node <<'JS'
const fs = require("node:fs");
const { chromium } = require(process.env.PLAYWRIGHT_MODULE);

const durationMs = Number(process.env.TINKABOT_DEMO_BROWSER_SYNC_MS);
const sampleMs = Number(process.env.TINKABOT_DEMO_BROWSER_SAMPLE_MS);
const thresholds = {
  browserAgeP95Ms: Number(process.env.TINKABOT_DEMO_BROWSER_AGE_P95_MS),
  browserAgeP99Ms: Number(process.env.TINKABOT_DEMO_BROWSER_AGE_P99_MS),
  filterLatencyP95Ms: Number(process.env.TINKABOT_DEMO_FILTER_P95_MS),
  sourceIntervalP95Ms: Number(process.env.TINKABOT_DEMO_SOURCE_INTERVAL_P95_MS),
};
const warmupMs = Number(process.env.TINKABOT_DEMO_BROWSER_WARMUP_MS);

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function percentile(values, pct) {
  if (values.length === 0) return null;
  const sorted = [...values].sort((a, b) => a - b);
  const idx = Math.min(sorted.length - 1, Math.ceil((pct / 100) * sorted.length) - 1);
  return sorted[idx];
}

function num(v) {
  return typeof v === "number" && Number.isFinite(v) ? v : null;
}

(async () => {
  const browser = await chromium.launch({ headless: true });
  try {
    const page = await browser.newPage({ viewport: { width: 1100, height: 760 } });
    await page.goto(process.env.TINKABOT_DEMO_APP_URL, { waitUntil: "domcontentloaded" });
    await page.waitForSelector("#s", { timeout: 15000 });
    await page.waitForFunction(() => {
      try {
        const doc = JSON.parse(document.querySelector("#s")?.textContent || "");
        return typeof doc.browserAgeMs === "number" && typeof doc.view?.seq === "number";
      } catch {
        return false;
      }
    }, null, { timeout: 15000 });
    await page.waitForFunction((limits) => {
      try {
        const doc = JSON.parse(document.querySelector("#s")?.textContent || "");
        const view = doc.view || {};
        return typeof doc.browserAgeMs === "number" &&
          doc.browserAgeMs <= limits.browserAgeP99Ms &&
          typeof view.filterLatencyMs === "number" &&
          view.filterLatencyMs <= limits.filterLatencyP95Ms &&
          typeof view.sourceIntervalMs === "number" &&
          view.sourceIntervalMs <= limits.sourceIntervalP95Ms;
      } catch {
        return false;
      }
    }, thresholds, { timeout: warmupMs });

    const samples = [];
    const unique = new Map();
    const deadline = Date.now() + durationMs;
    while (Date.now() < deadline) {
      const text = await page.locator("#s").textContent();
      try {
        const doc = JSON.parse(text || "{}");
        const view = doc.view || {};
        const seq = num(view.seq);
        const sample = {
          observedAt: new Date().toISOString(),
          seq,
          browserAgeMs: num(doc.browserAgeMs),
          fetchMs: num(doc.fetchMs),
          sourceIntervalMs: num(view.sourceIntervalMs),
          filterLatencyMs: num(view.filterLatencyMs),
        };
        if (seq !== null && sample.browserAgeMs !== null) {
          samples.push(sample);
          if (!unique.has(seq)) unique.set(seq, sample);
        }
      } catch {
        // Ignore transient render text while the page is refreshing.
      }
      await sleep(sampleMs);
    }

    const uniqueSamples = [...unique.values()];
    const ages = samples.map((s) => s.browserAgeMs).filter((v) => v !== null);
    const fetches = samples.map((s) => s.fetchMs).filter((v) => v !== null);
    const filterLatencies = uniqueSamples.map((s) => s.filterLatencyMs).filter((v) => v !== null);
    const sourceIntervals = uniqueSamples.map((s) => s.sourceIntervalMs).filter((v) => v !== null);
    const seqs = uniqueSamples.map((s) => s.seq).filter((v) => v !== null);
    const largeSeqGaps = [];
    for (let i = 1; i < seqs.length; i++) {
      const gap = seqs[i] - seqs[i - 1];
      if (gap > 175) largeSeqGaps.push(gap);
    }

    const stats = {
      kind: "tinkabot.realtimeBrowserSyncProof.v1",
      route: process.env.TINKABOT_DEMO_BROWSER_ROUTE,
      url: process.env.TINKABOT_DEMO_APP_URL,
      durationMs,
      sampleMs,
      samples: samples.length,
      uniqueSeqs: uniqueSamples.length,
      browserAgeMs: {
        p50: percentile(ages, 50),
        p95: percentile(ages, 95),
        p99: percentile(ages, 99),
        max: ages.length ? Math.max(...ages) : null,
      },
      fetchMs: {
        p95: percentile(fetches, 95),
        max: fetches.length ? Math.max(...fetches) : null,
      },
      filterLatencyMs: {
        p95: percentile(filterLatencies, 95),
        max: filterLatencies.length ? Math.max(...filterLatencies) : null,
      },
      sourceIntervalMs: {
        p95: percentile(sourceIntervals, 95),
        max: sourceIntervals.length ? Math.max(...sourceIntervals) : null,
      },
      largeSeqGapsOver175Ms: largeSeqGaps.length,
      thresholds,
    };
    stats.pass = stats.samples >= 20 &&
      stats.uniqueSeqs >= 20 &&
      stats.browserAgeMs.p95 !== null &&
      stats.browserAgeMs.p95 <= thresholds.browserAgeP95Ms &&
      stats.browserAgeMs.p99 <= thresholds.browserAgeP99Ms &&
      stats.filterLatencyMs.p95 !== null &&
      stats.filterLatencyMs.p95 <= thresholds.filterLatencyP95Ms &&
      stats.sourceIntervalMs.p95 !== null &&
      stats.sourceIntervalMs.p95 <= thresholds.sourceIntervalP95Ms;

    fs.writeFileSync(process.env.TINKABOT_DEMO_BROWSER_SYNC_OUT, `${JSON.stringify(stats, null, 2)}\n`);
    console.log(`browser-sync proof ${stats.route} age p95 ${stats.browserAgeMs.p95} p99 ${stats.browserAgeMs.p99} filter p95 ${stats.filterLatencyMs.p95} interval p95 ${stats.sourceIntervalMs.p95} unique ${stats.uniqueSeqs}`);
    if (!stats.pass) {
      throw new Error(`browser sync thresholds failed: ${JSON.stringify(stats)}`);
    }
  } finally {
    await browser.close();
  }
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
JS
fi

transform="$dist/transform-content.sh"
cat >"$transform" <<'SH'
#!/bin/sh
if [ -n "${TINKALET_DATA_DIR:-}" ] || [ -n "${SECRET_CREDS:-}" ]; then
  exit 9
fi
printf '{"asset":"demo-card","caption":"NATS-native chain reaction","style":"derived by local transformer"}'
SH
chmod +x "$transform"

log_step "register a local transformer reaction"
src="demo/content/source"
result="demo/content/rendered"
got="$(run_tinkalet item create "$src" --value '{"content":"illustrate this content"}')"
expect_prefix "$got" "item $src pending rev "
got="$(run_tinkalet reaction add illustrate --watch item "$src" --for resolved --cmd "$transform" --write "$result")"
expect_eq "$got" "reaction illustrate added"

log_step "run the reaction daemon once and resolve the source item"
reaction_run="$dist/reaction-run.json"
reaction_err="$dist/reaction.err"
run_tinkalet daemon react illustrate --once --timeout 5s --json >"$reaction_run" 2>"$reaction_err" &
react_pid=$!
sleep 0.2
got="$(run_tinkalet item resolve "$src" --value '{"content":"approved for illustration"}')"
expect_prefix "$got" "item $src resolved rev "
if ! wait "$react_pid"; then
  react_pid=""
  sed -n '1,120p' "$reaction_err" >&2 || true
  fail "reaction daemon did not complete"
fi
react_pid=""
grep -q '"reaction":"illustrate"' "$reaction_run" || fail "reaction run JSON missing reaction"
grep -q '"status":"ran"' "$reaction_run" || fail "reaction run JSON missing ran status"

reaction_item="$dist/reaction-result.json"
run_tinkalet item get "$result" --json >"$reaction_item"
grep -q "\"key\":\"$result\"" "$reaction_item" || fail "reaction result item missing"
grep -q '"status":"resolved"' "$reaction_item" || fail "reaction result item was not resolved"
grep -q 'NATS-native chain reaction' "$reaction_item" || fail "transformed stdout missing from result item"

log_step "set a Tinkabot-owned schedule and read the scheduled item"
got="$(run_tinkalet schedule set demoheartbeat --every 200ms --write demo/content/heartbeat --value '{"kind":"heartbeat"}')"
expect_eq "$got" "schedule demoheartbeat active every 200ms -> demo/content/heartbeat"
schedule_item="$dist/scheduled-item.json"
for _ in {1..100}; do
  if run_tinkalet item get demo/content/heartbeat --json >"$schedule_item" 2>/dev/null &&
    grep -q '"key":"demo/content/heartbeat"' "$schedule_item" &&
    grep -q '"status":"resolved"' "$schedule_item" &&
    grep -q '"schedule":"demoheartbeat"' "$schedule_item"; then
    break
  fi
  sleep 0.1
done
grep -q '"schedule":"demoheartbeat"' "$schedule_item" 2>/dev/null || fail "scheduled item missing"
got="$(run_tinkalet schedule off demoheartbeat)"
expect_eq "$got" "schedule demoheartbeat off"

log_step "demo passed"
printf 'package root %s\n' "$pkg"
printf 'artifacts %s\n' "$dist"
printf 'realtime cadence override %s\n' "$fast_every"
if [[ -n "${browser_sync:-}" ]]; then
  printf 'browser sync %s\n' "$browser_sync"
fi
printf 'trigger changed projection %s -> %s\n' "$before" "$after"
printf 'reaction wrote %s\n' "$reaction_item"
printf 'schedule wrote %s\n' "$schedule_item"

if [[ "${TINKABOT_DEMO_HOLD:-}" == "1" ]]; then
  log_step "holding demo shell open"
  printf 'open %s/artifacts/bundle/clock/index.html\n' "$public_shell_url"
  while kill -0 "$pid" 2>/dev/null; do
    sleep 3600 &
    wait $! || true
  done
fi
