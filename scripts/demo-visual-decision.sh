#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="${1:-$(mktemp -d /tmp/tinkabot-visual-demo.XXXXXX)}"
case "$dist" in
  /*) ;;
  *) dist="$root/$dist" ;;
esac

visual_key="${TINKABOT_DEMO_VISUAL_KEY:-artifacts.artifact-browser.results.choice}"
visual_choice="${TINKABOT_DEMO_VISUAL_CHOICE:-diagram-a}"
visual_session="${TINKABOT_DEMO_VISUAL_SESSION:-visual-001}"

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
  echo "demo-visual-decision: $*" >&2
  if [[ -n "${log:-}" && -f "$log" ]]; then
    sed -n '1,160p' "$log" | redact_log >&2 || true
  fi
  if [[ -n "${err:-}" && -f "$err" ]]; then
    sed -n '1,160p' "$err" | redact_log >&2 || true
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

store="$(mktemp -d /tmp/tinkabot-visual-store.XXXXXX)"
owner_cfg="$(mktemp -d /tmp/tinkalet-visual-owner-config.XXXXXX)"
owner_data="$(mktemp -d /tmp/tinkalet-visual-owner-data.XXXXXX)"
owner_home="$(mktemp -d /tmp/tinkalet-visual-owner-home.XXXXXX)"
watcher_cfg="$(mktemp -d /tmp/tinkalet-visual-watcher-config.XXXXXX)"
watcher_data="$(mktemp -d /tmp/tinkalet-visual-watcher-data.XXXXXX)"
watcher_home="$(mktemp -d /tmp/tinkalet-visual-watcher-home.XXXXXX)"
log="$dist/tinkabot.log"
err="$dist/tinkabot.err"
proof="$dist/visual-decision-proof.json"
watch_event="$dist/watcher-event.json"
watcher_profiles="$dist/watcher-profiles.json"
owner_item="$dist/owner-item.json"
restart_item="$dist/restart-item.json"
before="$dist/projection-before.json"
after="$dist/projection-after.json"
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

run_owner_tinkalet() {
  env -i \
    HOME="$owner_home" \
    TINKALET_CONFIG_DIR="$owner_cfg" \
    TINKALET_DATA_DIR="$owner_data" \
    PATH="/nonexistent" \
    "$pkg/tinkalet" "$@"
}

run_watcher_tinkalet() {
  env -i \
    HOME="$watcher_home" \
    TINKALET_CONFIG_DIR="$watcher_cfg" \
    TINKALET_DATA_DIR="$watcher_data" \
    PATH="/nonexistent" \
    "$pkg/tinkalet" "$@"
}

log_step "start packaged Tinkabot with clock bundle and scoped LLM watcher"
(cd "$pkg" && TB_DEMO_SESSION="$visual_session" PATH="/usr/bin:/bin" ./tinkabot \
  --store "$store" \
  --shell 127.0.0.1:0 \
  --bundle examples/clock \
  --watcher "llm:item:$visual_key" \
  >"$log" 2>"$err") &
pid=$!

for _ in {1..300}; do
  if grep -q '^shell  http://127\.0\.0\.1:' "$log" && grep -q "^watcher llm item $visual_key " "$log"; then
    break
  fi
  if ! kill -0 "$pid" 2>/dev/null; then
    fail "tinkabot exited before watcher was admitted"
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
artifact_url="$public_shell_url/artifacts/bundle/clock/index.html"
visual_url="$public_shell_url/?tb_visual=$visual_key&tb_choice=$visual_choice&tb_session=$visual_session"
printf 'shell %s\n' "$public_shell_url"
printf 'visual %s\n' "$visual_url"
printf 'clock %s\n' "$artifact_url"
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
for _ in {1..150}; do
  if curl -fsS "$artifact_url" >/dev/null 2>/dev/null; then
    break
  fi
  sleep 0.1
done
curl -fsS "$artifact_url" >/dev/null || fail "clock artifact URL is not reachable"

fast_every="${TINKABOT_DEMO_FAST_EVERY:-100ms}"
if [[ "$fast_every" != "manifest" ]]; then
  log_step "retune clock bundle to $fast_every for transform proof"
  "$pkg/libexec/tinkabot/nats" --no-context \
    --server "$nats_url" \
    --creds "$store/caller.creds" \
    --timeout 2s \
    kv put config_bucket bundle.clock.tick.every "$fast_every" >/dev/null
fi

log_step "disable packaged nats sidecar before Tinkalet commands"
mv "$pkg/libexec/tinkabot/nats" "$pkg/libexec/tinkabot/nats.disabled"

log_step "import owner profile in owner Tinkalet environment"
got="$(run_owner_tinkalet profile import local --store "$store" --name owner)"
expect_eq "$got" "profile owner imported"
got="$(run_owner_tinkalet profile use owner)"
expect_eq "$got" "profile owner selected"

log_step "import scoped watcher profile in isolated LLM Tinkalet environment"
got="$(run_watcher_tinkalet profile import local --store "$store/watchers/llm" --name llm)"
expect_eq "$got" "profile llm imported"
got="$(run_watcher_tinkalet profile use llm)"
expect_eq "$got" "profile llm selected"
run_watcher_tinkalet profile list --json >"$watcher_profiles"

WATCHER_PROFILES="$watcher_profiles" node <<'JS'
const fs = require("node:fs");
const list = JSON.parse(fs.readFileSync(process.env.WATCHER_PROFILES, "utf8"));
const names = (list.profiles || []).map((profile) => profile.name);
if (list.default !== "llm") throw new Error(`watcher default is ${list.default}`);
if (names.length !== 1 || names[0] !== "llm") {
  throw new Error(`watcher env is not isolated: ${JSON.stringify(names)}`);
}
JS

log_step "prove clock transform projection changes through Tinkalet trigger"
for _ in {1..150}; do
  if curl -fsS "$local_shell_url/projections/bundle.clock.view" >"$before"; then
    break
  fi
  sleep 0.1
done
[[ -s "$before" ]] || fail "clock view projection missing before trigger"

got="$(run_owner_tinkalet trigger bundle.clock.tick --request-id demo-visual-decision-1)"
expect_eq "$got" "profile owner accepted bundle.clock.tick"

for _ in {1..150}; do
  if curl -fsS "$local_shell_url/projections/bundle.clock.view" >"$after" && ! cmp -s "$before" "$after"; then
    break
  fi
  sleep 0.1
done
if cmp -s "$before" "$after"; then
  fail "clock view projection did not change after trigger"
fi

log_step "drive generated visual UI through the trusted shell"
  PLAYWRIGHT_MODULE="$root/apps/frontend/node_modules/playwright" \
  TINKABOT_DEMO_VISUAL_URL="$visual_url" \
  TINKABOT_DEMO_ARTIFACT_URL="$artifact_url" \
  TINKABOT_DEMO_LOCAL_SHELL_URL="$local_shell_url" \
  TINKABOT_DEMO_VISUAL_OUT="$proof" \
  TINKABOT_DEMO_VISUAL_KEY="$visual_key" \
  TINKABOT_DEMO_VISUAL_CHOICE="$visual_choice" \
  TINKABOT_DEMO_VISUAL_ROUTE="$([[ "$public_shell_url" == "$local_shell_url" ]] && printf local || printf tailscale)" \
  node <<'JS'
const fs = require("node:fs");
const { chromium } = require(process.env.PLAYWRIGHT_MODULE);

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
    const artifactPage = await browser.newPage({ viewport: { width: 980, height: 720 } });
    const artifactStarted = Date.now();
    await artifactPage.goto(process.env.TINKABOT_DEMO_ARTIFACT_URL, { waitUntil: "domcontentloaded" });
    await artifactPage.waitForFunction(() => {
      const text = document.body?.innerText || document.body?.textContent || "";
      return text.includes("tinkabot sequence");
    }, null, { timeout: 10000 });
    const artifact = await artifactPage.evaluate(() => {
      const text = (document.body?.innerText || document.body?.textContent || "").trim();
      const html = document.documentElement.outerHTML || "";
      return {
        url: location.href,
        title: document.title,
        textLength: text.length,
        htmlLength: html.length,
        hasDiagram: html.includes("sequenceDiagram") || text.includes("User or LLM"),
        hasProjectionPanel: Boolean(document.querySelector("#s")),
      };
    });
    artifact.elapsedMs = Date.now() - artifactStarted;
    artifact.nonblank = artifact.textLength > 20 && artifact.hasDiagram && artifact.hasProjectionPanel;
    if (!artifact.nonblank) {
      throw new Error(`artifact render proof failed: ${JSON.stringify(artifact)}`);
    }

    const page = await browser.newPage({ viewport: { width: 980, height: 720 } });
    const started = Date.now();
    await page.goto(process.env.TINKABOT_DEMO_VISUAL_URL, { waitUntil: "domcontentloaded" });
    await page.waitForFunction(() => {
      const proof = window.__tinkabotProof;
      const dispatches = Array.isArray(proof?.dispatched) ? proof.dispatched : [];
      return dispatches.some((d) => d.command === "item_submit" && d.status === "accepted");
    }, null, { timeout: 30000 });

    const frame = page.frames().find((f) => f.url().startsWith("blob:"));
    if (!frame) throw new Error("generated frame missing");
    await frame.waitForFunction(() => document.querySelector("#generated")?.dataset.complete === "true", null, { timeout: 10000 });
    const dom = {
      title: await frame.locator('[data-demo="title"]').textContent(),
      status: await frame.locator('[data-demo="status"]').textContent(),
      selected: await frame.locator('[data-demo="selected"]').textContent(),
      item: await frame.locator('[data-demo="item"]').textContent(),
      denied: Number(await frame.locator('[data-demo="denied"]').textContent()),
      text: await frame.locator("#generated").textContent(),
    };
    const proof = await page.evaluate(() => window.__tinkabotProof);
    const dispatches = Array.isArray(proof?.dispatched) ? proof.dispatched : [];
    const submits = dispatches.filter((d) => d.command === "item_submit");
    const accepted = submits.filter((d) => d.status === "accepted");
    const stats = {
      kind: "tinkabot.visualDecisionProof.v1",
      route: process.env.TINKABOT_DEMO_VISUAL_ROUTE,
      url: process.env.TINKABOT_DEMO_VISUAL_URL,
      artifact,
      localShellUrl: process.env.TINKABOT_DEMO_LOCAL_SHELL_URL,
      key: process.env.TINKABOT_DEMO_VISUAL_KEY,
      choice: process.env.TINKABOT_DEMO_VISUAL_CHOICE,
      elapsedMs: Date.now() - started,
      dom,
      acceptedIntents: Array.isArray(proof?.accepted) ? proof.accepted.length : 0,
      deniedIntents: Array.isArray(proof?.denied) ? proof.denied.length : 0,
      acceptedSubmits: accepted.length,
      deniedDispatches: dispatches.filter((d) => d.status !== "accepted").length,
      itemKey: accepted[0]?.itemKey || null,
      submitLatencyMs: accepted[0]?.latencyMs || null,
      authorityLeakCount: leakCount(JSON.stringify({ proof, dom })),
    };
    stats.pass = stats.acceptedIntents === 1 &&
      stats.artifact.nonblank === true &&
      stats.deniedIntents === 0 &&
      stats.acceptedSubmits === 1 &&
      stats.deniedDispatches === 0 &&
      stats.itemKey === stats.key &&
      stats.dom.status === "complete" &&
      stats.dom.selected === stats.choice &&
      stats.dom.item === stats.key &&
      stats.dom.denied === 0 &&
      stats.authorityLeakCount === 0;
    fs.writeFileSync(process.env.TINKABOT_DEMO_VISUAL_OUT, `${JSON.stringify(stats, null, 2)}\n`);
    console.log(`visual decision proof ${stats.route} ${stats.key}=${stats.choice} latency ${stats.submitLatencyMs}ms`);
    if (!stats.pass) {
      throw new Error(`visual proof failed: ${JSON.stringify(stats)}`);
    }
  } finally {
    await browser.close();
  }
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
JS

log_step "verify isolated scoped watcher can watch only the submitted item"
got="$(run_watcher_tinkalet profile use llm)"
expect_eq "$got" "profile llm selected"
if got="$(run_watcher_tinkalet item get "$visual_key" --json 2>&1)"; then
  fail "watcher direct item get unexpectedly succeeded: $got"
fi
case "$got" in
  *"denied-scope"*) ;;
  *) fail "watcher item get did not fail with denied-scope: $got" ;;
esac
if got="$(run_watcher_tinkalet watch prefix artifacts.artifact-browser.results --limit 1 --timeout 1s --json 2>&1)"; then
  fail "watcher broad prefix unexpectedly succeeded: $got"
fi
case "$got" in
  *"denied-scope"*) ;;
  *) fail "watcher broad prefix did not fail with denied-scope: $got" ;;
esac
run_watcher_tinkalet watch item "$visual_key" --limit 1 --timeout 10s --json >"$watch_event"

WATCH_EVENT="$watch_event" VISUAL_KEY="$visual_key" VISUAL_CHOICE="$visual_choice" node <<'JS'
const fs = require("node:fs");
const ev = JSON.parse(fs.readFileSync(process.env.WATCH_EVENT, "utf8"));
if (ev.key !== process.env.VISUAL_KEY) throw new Error(`wrong watcher key ${ev.key}`);
if (ev.status !== "resolved") throw new Error(`wrong watcher status ${ev.status}`);
if (!ev.value || ev.value.choice !== process.env.VISUAL_CHOICE) {
  throw new Error(`wrong watcher choice ${JSON.stringify(ev.value)}`);
}
JS

log_step "verify owner can read durable submitted item"
got="$(run_owner_tinkalet profile use owner)"
expect_eq "$got" "profile owner selected"
run_owner_tinkalet item get "$visual_key" --json >"$owner_item"

OWNER_ITEM="$owner_item" VISUAL_KEY="$visual_key" VISUAL_CHOICE="$visual_choice" node <<'JS'
const fs = require("node:fs");
const item = JSON.parse(fs.readFileSync(process.env.OWNER_ITEM, "utf8"));
if (item.key !== process.env.VISUAL_KEY) throw new Error(`wrong owner key ${item.key}`);
if (item.status !== "resolved") throw new Error(`wrong owner status ${item.status}`);
if (!item.value || item.value.choice !== process.env.VISUAL_CHOICE) {
  throw new Error(`wrong owner choice ${JSON.stringify(item.value)}`);
}
JS

log_step "restart Tinkabot and verify submitted item survives"
kill -TERM "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
pid=""
restart_log="$dist/tinkabot-restart.log"
restart_err="$dist/tinkabot-restart.err"
(cd "$pkg" && TB_DEMO_SESSION="$visual_session" PATH="/usr/bin:/bin" ./tinkabot \
  --store "$store" \
  --shell 127.0.0.1:0 \
  >"$restart_log" 2>"$restart_err") &
pid=$!
for _ in {1..200}; do
  if grep -q '^shell  http://127\.0\.0\.1:' "$restart_log"; then
    break
  fi
  if ! kill -0 "$pid" 2>/dev/null; then
    log="$restart_log"
    err="$restart_err"
    fail "tinkabot exited before restart posture"
  fi
  sleep 0.1
done
got="$(run_owner_tinkalet profile import local --store "$store" --name owner)"
expect_eq "$got" "profile owner imported"
got="$(run_owner_tinkalet profile use owner)"
expect_eq "$got" "profile owner selected"
run_owner_tinkalet item get "$visual_key" --json >"$restart_item"

RESTART_ITEM="$restart_item" VISUAL_KEY="$visual_key" VISUAL_CHOICE="$visual_choice" node <<'JS'
const fs = require("node:fs");
const item = JSON.parse(fs.readFileSync(process.env.RESTART_ITEM, "utf8"));
if (item.key !== process.env.VISUAL_KEY) throw new Error(`wrong restart key ${item.key}`);
if (item.status !== "resolved") throw new Error(`wrong restart status ${item.status}`);
if (!item.value || item.value.choice !== process.env.VISUAL_CHOICE) {
  throw new Error(`wrong restart choice ${JSON.stringify(item.value)}`);
}
JS

PROOF="$proof" WATCH_EVENT="$watch_event" WATCHER_PROFILES="$watcher_profiles" OWNER_ITEM="$owner_item" RESTART_ITEM="$restart_item" BEFORE="$before" AFTER="$after" node <<'JS'
const fs = require("node:fs");
const proof = JSON.parse(fs.readFileSync(process.env.PROOF, "utf8"));
const list = JSON.parse(fs.readFileSync(process.env.WATCHER_PROFILES, "utf8"));
const profiles = (list.profiles || []).map((profile) => ({
  name: profile.name,
  default: profile.default,
  role: profile.role,
  trust: profile.trust,
  source: profile.source,
  watchScope: profile.watchScope,
  watchTarget: profile.watchTarget,
  credentialRedacted: profile.credentialRedacted,
}));
const names = profiles.map((profile) => profile.name);
proof.watcher = JSON.parse(fs.readFileSync(process.env.WATCH_EVENT, "utf8"));
proof.watcherProfiles = { default: list.default, profiles };
proof.watcherHasOwnerProfile = names.includes("owner");
proof.watcherIsolated = list.default === "llm" && names.length === 1 && names[0] === "llm" && profiles[0]?.watchTarget === proof.key;
proof.ownerItem = JSON.parse(fs.readFileSync(process.env.OWNER_ITEM, "utf8"));
proof.restartItem = JSON.parse(fs.readFileSync(process.env.RESTART_ITEM, "utf8"));
proof.transformChanged = fs.readFileSync(process.env.BEFORE, "utf8") !== fs.readFileSync(process.env.AFTER, "utf8");
proof.restartDurable = proof.restartItem.key === proof.key && proof.restartItem.value?.choice === proof.choice;
proof.artifactRendered = proof.artifact?.nonblank === true;
proof.pass = proof.pass && proof.artifactRendered && proof.transformChanged && proof.watcherIsolated && !proof.watcherHasOwnerProfile && proof.watcher.key === proof.key && proof.ownerItem.key === proof.key && proof.restartDurable;
fs.writeFileSync(process.env.PROOF, `${JSON.stringify(proof, null, 2)}\n`);
if (!proof.pass) throw new Error(`final proof failed: ${JSON.stringify(proof)}`);
JS

log_step "demo passed"
printf 'package root %s\n' "$pkg"
printf 'artifacts %s\n' "$dist"
printf 'visual proof %s\n' "$proof"
printf 'open %s\n' "$public_shell_url"

if [[ "${TINKABOT_DEMO_HOLD:-}" == "1" ]]; then
  log_step "holding demo shell open"
  while kill -0 "$pid" 2>/dev/null; do
    sleep 3600 &
    wait $! || true
  done
fi
