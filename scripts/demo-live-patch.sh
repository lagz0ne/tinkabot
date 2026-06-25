#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="${1:-$(mktemp -d /tmp/tinkabot-patch-demo.XXXXXX)}"
case "$dist" in
  /*) ;;
  *) dist="$root/$dist" ;;
esac

log_step() {
  printf '\n[%s] %s\n' "$(date -u +%H:%M:%S)" "$*"
}

fail() {
  echo "demo-live-patch: $*" >&2
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

json_field() {
  local file="$1"
  local name="$2"
  sed -n "s/.*\"$name\":\"\\([^\"]*\\)\".*/\\1/p" "$file" | head -n 1
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

store="$(mktemp -d /tmp/tinkabot-patch-store.XXXXXX)"
cfg="$(mktemp -d /tmp/tinkalet-patch-config.XXXXXX)"
data="$(mktemp -d /tmp/tinkalet-patch-data.XXXXXX)"
home="$(mktemp -d /tmp/tinkalet-patch-home.XXXXXX)"
log="$dist/tinkabot.log"
err="$dist/tinkabot.err"
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

log_step "start packaged Tinkabot with the builder bundle"
(cd "$pkg" && ./tinkabot --store "$store" --shell 127.0.0.1:0 --bundle examples/builder >"$log" 2>"$err") &
pid=$!

for _ in {1..300}; do
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

app_path="/artifacts/bundle/builder/index.html"
asset_path="/artifacts/bundle/builder/assets/index.js"
built_path="/artifacts/bundle/builder/_p/built"
printf 'shell %s\n' "$public_shell_url"
printf 'builder %s%s\n' "$public_shell_url" "$app_path"
printf 'local-shell %s\n' "$local_shell_url"

log_step "setup Tinkalet profile from the running Tinkabot store"
mv "$pkg/libexec/tinkabot/nats" "$pkg/libexec/tinkabot/nats.disabled"
got="$(run_tinkalet profile import local --store "$store" --name local)"
expect_eq "$got" "profile local imported"
got="$(run_tinkalet profile use local)"
expect_eq "$got" "profile local selected"

log_step "wait for the cold build artifact"
before_asset="$dist/builder-main-before.js"
before_built="$dist/builder-built-before.json"
for _ in {1..300}; do
  if curl -fsS "$local_shell_url$asset_path" >"$before_asset" 2>/dev/null &&
    curl -fsS "$local_shell_url$built_path" >"$before_built" 2>/dev/null &&
    grep -q '_p/built' "$before_asset"; then
    break
  fi
  sleep 0.1
done
grep -q '_p/built' "$before_asset" || fail "builder artifact missing live projection watcher"
before_rev="$(json_field "$before_built" artifactRevision)"
[[ -n "$before_rev" ]] || fail "initial built projection missing artifactRevision"

if [[ "${TINKABOT_DEMO_PATCH_DELAY:-0}" != "0" ]]; then
  log_step "waiting ${TINKABOT_DEMO_PATCH_DELAY}s before patch"
  sleep "$TINKABOT_DEMO_PATCH_DELAY"
fi

log_step "open browser and patch running work through Tinkalet"
browser_proof="$dist/browser-refresh.json"
PLAYWRIGHT_MODULE="$root/apps/frontend/node_modules/playwright" \
  TINKABOT_DEMO_APP_URL="$public_shell_url$app_path" \
  TINKABOT_DEMO_TINKALET="$pkg/tinkalet" \
  TINKABOT_DEMO_HOME="$home" \
  TINKABOT_DEMO_CONFIG_DIR="$cfg" \
  TINKABOT_DEMO_DATA_DIR="$data" \
  TINKABOT_DEMO_TRIGGER_OUT="$dist/source-trigger.txt" \
  TINKABOT_DEMO_BROWSER_PROOF="$browser_proof" \
  node <<'JS'
const { execFileSync } = require("node:child_process");
const fs = require("node:fs");
const { chromium } = require(process.env.PLAYWRIGHT_MODULE);

const trigger = [
  "trigger",
  "bundle.builder.source",
  "--request-id",
  `demo-live-patch-${Date.now()}`,
];
const triggerEnv = {
  HOME: process.env.TINKABOT_DEMO_HOME,
  TINKALET_CONFIG_DIR: process.env.TINKABOT_DEMO_CONFIG_DIR,
  TINKALET_DATA_DIR: process.env.TINKABOT_DEMO_DATA_DIR,
  PATH: "/nonexistent",
};

(async () => {
  const browser = await chromium.launch({ headless: true });
  try {
    const page = await browser.newPage({ viewport: { width: 960, height: 640 } });
    await page.goto(process.env.TINKABOT_DEMO_APP_URL, { waitUntil: "domcontentloaded" });
    await page.waitForSelector("h1", { timeout: 10000 });
    await page.waitForFunction(
      () => document.querySelector("p")?.textContent?.includes("app.rev."),
      null,
      { timeout: 10000 },
    );
    const before = await page.locator("h1").textContent();
    const beforeMeta = await page.locator("p").textContent();
    const out = execFileSync(process.env.TINKABOT_DEMO_TINKALET, trigger, { encoding: "utf8", env: triggerEnv }).trim();
    fs.writeFileSync(process.env.TINKABOT_DEMO_TRIGGER_OUT, `${out}\n`);
    if (out !== "profile local accepted bundle.builder.source") {
      throw new Error(`source trigger was not accepted: ${out}`);
    }
    await page.waitForFunction(
      (text) => document.querySelector("h1")?.textContent !== text,
      before,
      { timeout: 15000 },
    );
    const after = await page.locator("h1").textContent();
    const afterMeta = await page.locator("p").textContent();
    fs.writeFileSync(process.env.TINKABOT_DEMO_BROWSER_PROOF, JSON.stringify({
      before,
      beforeMeta,
      trigger: out,
      after,
      afterMeta,
      href: page.url(),
    }, null, 2) + "\n");
    console.log(`browser ${beforeMeta} -> ${afterMeta}`);
  } finally {
    await browser.close();
  }
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
JS
grep -q '^profile local accepted bundle\.builder\.source$' "$dist/source-trigger.txt" || fail "source trigger was not accepted"

after_asset="$dist/builder-main-after.js"
after_built="$dist/builder-built-after.json"
for _ in {1..300}; do
  if curl -fsS "$local_shell_url$asset_path" >"$after_asset" 2>/dev/null &&
    curl -fsS "$local_shell_url$built_path" >"$after_built" 2>/dev/null; then
    after_rev="$(json_field "$after_built" artifactRevision)"
    if [[ -n "$after_rev" && "$after_rev" != "$before_rev" ]] && ! cmp -s "$before_asset" "$after_asset"; then
      break
    fi
  fi
  sleep 0.1
done
after_rev="$(json_field "$after_built" artifactRevision)"
[[ "$after_rev" != "$before_rev" ]] || fail "built projection did not advance"
if cmp -s "$before_asset" "$after_asset"; then
  fail "served JS artifact did not change after source patch"
fi

if [[ "$public_shell_url" != "$local_shell_url" ]]; then
  for _ in {1..50}; do
    if curl -fsS "$public_shell_url$app_path" >/dev/null 2>/dev/null; then
      break
    fi
    sleep 0.1
  done
  curl -fsS "$public_shell_url$app_path" >/dev/null || fail "Tailscale builder URL is not reachable"
fi

log_step "demo passed"
printf 'package root %s\n' "$pkg"
printf 'artifacts %s\n' "$dist"
printf 'built projection %s -> %s\n' "$before_rev" "$after_rev"
printf 'asset changed %s -> %s\n' "$before_asset" "$after_asset"
printf 'browser refresh %s\n' "$browser_proof"
printf 'open %s%s\n' "$public_shell_url" "$app_path"

if [[ "${TINKABOT_DEMO_HOLD:-}" == "1" ]]; then
  log_step "holding demo shell open"
  while kill -0 "$pid" 2>/dev/null; do
    sleep 3600 &
    wait $! || true
  done
fi
