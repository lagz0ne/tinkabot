#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="${1:-$(mktemp -d /tmp/tinkabot-frontend-autopilot.XXXXXX)}"
case "$dist" in
  /*) ;;
  *) dist="$root/$dist" ;;
esac
app_name="options-site"
option_key="artifacts.${app_name}.results.plan"
option_prefix="artifacts.${app_name}.results."
visual_session="frontend-autopilot-001"

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
  echo "demo-frontend-autopilot: $*" >&2
  if [[ -n "${log:-}" && -f "$log" ]]; then
    sed -n '1,120p' "$log" | redact_log >&2 || true
  fi
  if [[ -n "${err:-}" && -f "$err" ]]; then
    sed -n '1,120p' "$err" | redact_log >&2 || true
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

write_proof() {
  PROOF="$proof" \
  FIRST_FAILURE="${1:-}" \
  REGISTER_EXIT="${2:-0}" \
  REGISTER_STDOUT="${3:-}" \
  REGISTER_STDERR="${4:-}" \
  CREATE_EXIT="${5:-0}" \
  CREATE_STDOUT="${6:-}" \
  CREATE_STDERR="${7:-}" \
  HANDLER_FROM="${handler_from:-}" \
  SHELL_URL="${public_shell_url:-}" \
  USER_COMMANDS="${user_commands:-}" \
  WORKER_COMMANDS="${worker_commands:-}" \
  PROOF_COMMANDS="${proof_commands:-}" \
  MANUAL_UNBLOCKS="${manual_unblocks:-}" \
  node <<'JS'
const fs = require("node:fs");
const registerExit = Number(process.env.REGISTER_EXIT || 0);
const createExit = Number(process.env.CREATE_EXIT || 0);
function jsonl(path) {
  if (!path || !fs.existsSync(path)) return [];
  return fs.readFileSync(path, "utf8").trim().split(/\n+/).filter(Boolean).map((line) => JSON.parse(line));
}
const userCommands = jsonl(process.env.USER_COMMANDS);
const workerCommands = jsonl(process.env.WORKER_COMMANDS);
const proofCommands = jsonl(process.env.PROOF_COMMANDS);
const manualUnblocks = jsonl(process.env.MANUAL_UNBLOCKS);
const proof = {
  kind: "tinkabot.frontendAutopilotProof.v1",
  pass: false,
  clean_install_to_kv_reaction_journey_passing: false,
  frontend_autopilot_reference_families: createExit === 0 ? 3 : (registerExit === 0 ? 2 : 1),
  frontend_autopilot_target_families: 7,
  frontend_autopilot_first_failure: process.env.FIRST_FAILURE || null,
  post_install_user_command_count: userCommands.length,
  post_install_user_commands: userCommands,
  worker_command_count: workerCommands.length,
  worker_commands: workerCommands,
  proof_command_count: proofCommands.length,
  proof_commands: proofCommands,
  manual_unblock_count: manualUnblocks.length,
  manual_unblocks: manualUnblocks,
  non_nats_product_path_count: 0,
  codex_direct_ui_changed_lines: 0,
  non_opus_authored_ui_artifact_count: 0,
  claude_opus_ui_artifact_count: 0,
  authority_leak_count: 0,
  shell_url: process.env.SHELL_URL || null,
  register_handler: {
    command: `tinkalet app handler register vite --from ${process.env.HANDLER_FROM || "<missing>"} --json`,
    exit_code: registerExit,
    stdout: process.env.REGISTER_STDOUT || "",
    stderr: process.env.REGISTER_STDERR || "",
  },
  create_frontend_app: {
    command: "tinkalet app create frontend options-site --handler vite --json",
    exit_code: createExit,
    stdout: process.env.CREATE_STDOUT || "",
    stderr: process.env.CREATE_STDERR || "",
  },
};
fs.writeFileSync(process.env.PROOF, `${JSON.stringify(proof, null, 2)}\n`);
JS
}

mkdir -p "$dist"
log_step "build release-shaped package in $dist"
bash "$root/scripts/release-package.sh" "$dist" >/dev/null
bun_bin="$(command -v bun || true)"
[[ -n "$bun_bin" ]] || fail "bun is required to install the frontend-autopilot Vite bundle"
bun_path="$(dirname "$bun_bin"):/usr/bin:/bin"

archive="$(find "$dist" -maxdepth 1 -name 'tinkabot-v*.tar.gz' | sort | tail -n 1)"
[[ -n "$archive" ]] || fail "release archive missing"

tar -xzf "$archive" -C "$dist"
pkg="${archive%.tar.gz}"
for file in tinkabot tinkalet libexec/tinkabot/bwrap libexec/tinkabot/nats; do
  [[ -x "$pkg/$file" ]] || fail "$file is not executable"
done

store="$(mktemp -d /tmp/tinkabot-frontend-autopilot-store.XXXXXX)"
cfg="$(mktemp -d /tmp/tinkalet-frontend-autopilot-config.XXXXXX)"
data="$(mktemp -d /tmp/tinkalet-frontend-autopilot-data.XXXXXX)"
home="$(mktemp -d /tmp/tinkalet-frontend-autopilot-home.XXXXXX)"
watcher_cfg="$(mktemp -d /tmp/tinkalet-frontend-autopilot-llm-config.XXXXXX)"
watcher_data="$(mktemp -d /tmp/tinkalet-frontend-autopilot-llm-data.XXXXXX)"
watcher_home="$(mktemp -d /tmp/tinkalet-frontend-autopilot-llm-home.XXXXXX)"
log="$dist/tinkabot.log"
err="$dist/tinkabot.err"
proof="$dist/frontend-autopilot-proof.json"
browser_proof="$dist/browser-proof.json"
watch_event="$dist/llm-watch-event.json"
watch_err="$dist/llm-watch-event.err"
llm_reaction="$dist/llm-reaction.json"
opus_ui_generation="$dist/opus-ui-generation.json"
opus_ui_review="$dist/opus-ui-review.json"
pid=""
watch_pid=""
llm_pid=""

cleanup() {
  if [[ -n "$llm_pid" ]] && kill -0 "$llm_pid" 2>/dev/null; then
    kill -TERM "$llm_pid" 2>/dev/null || true
    wait "$llm_pid" 2>/dev/null || true
  fi
  if [[ -n "$watch_pid" ]] && kill -0 "$watch_pid" 2>/dev/null; then
    kill -TERM "$watch_pid" 2>/dev/null || true
    wait "$watch_pid" 2>/dev/null || true
  fi
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
    kill -TERM "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT

user_commands="$dist/user-commands.jsonl"
worker_commands="$dist/worker-commands.jsonl"
proof_commands="$dist/proof-commands.jsonl"
manual_unblocks="$dist/manual-unblocks.jsonl"
: >"$user_commands"
: >"$worker_commands"
: >"$proof_commands"
: >"$manual_unblocks"

record_command() {
  local file="$1"
  local actor="$2"
  local command="$3"
  COMMAND_FILE="$file" COMMAND_ACTOR="$actor" COMMAND_TEXT="$command" node <<'JS'
const fs = require("node:fs");
const rec = {
  actor: process.env.COMMAND_ACTOR,
  command: process.env.COMMAND_TEXT,
  recordedAt: new Date().toISOString(),
};
fs.appendFileSync(process.env.COMMAND_FILE, `${JSON.stringify(rec)}\n`);
JS
}

record_user_command() {
  record_command "$user_commands" "user" "$1"
}

record_worker_command() {
  record_command "$worker_commands" "worker" "$1"
}

record_proof_command() {
  record_command "$proof_commands" "proof" "$1"
}

run_tinkalet() {
  env -i \
    HOME="$home" \
    TINKALET_CONFIG_DIR="$cfg" \
    TINKALET_DATA_DIR="$data" \
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

if ! command -v claude >/dev/null 2>&1; then
  fail "claude CLI is required for the Opus UI and LLM reaction proof"
fi

log_step "ask Claude Opus to generate the packaged UI artifact"
record_worker_command "claude -p --model opus --allowedTools=Read"
ui_file="$pkg/examples/frontend-autopilot/ui/options-site.html"
ui_provenance="$pkg/examples/frontend-autopilot/ui/options-site.provenance.json"
ui_generation_raw="$dist/opus-ui-generation.raw"
ui_generation_err="$dist/opus-ui-generation.err"
set +e
claude -p --model opus --allowedTools=Read >"$ui_generation_raw" 2>"$ui_generation_err" <<'PROMPT'
Generate a complete, self-contained HTML file for a Tinkabot generated iframe that collects website build options.
Do not run tools. Do not use C3. Return JSON plus one fenced html block. The JSON must have:
kind "tinkabot.claudeOpusUiGeneration.v1",
model "opus",
htmlBase64 containing the UTF-8 HTML document if you can produce it reliably; otherwise set htmlBase64 to "",
notes string.
Then include exactly one ```html fenced block containing the full HTML document. Do not include markdown besides the JSON fence and HTML fence.

The HTML must implement this exact browser contract:
- It is a full HTML document with inline CSS and JS only.
- It sends parent.postMessage({ type: "content.ready" }, "*") after setup.
- It listens for a message with type "tinkabot.lease", stores data.lease and data.demo, and marks the UI linked.
- It exposes window.__frontendAutopilot = { submit, snapshot }.
- snapshot() returns { hasLease, submitted, complete, pendingCommandId, value }.
- submit() posts a content.intent message to parent with command "item_submit"; top-level fields must include expectedRevision: lease.artifactRevision, nonce: lease.nonce, frameId: lease.frameId, artifactRevision: lease.artifactRevision, and schemaRevision: lease.schemaRevision; payload.key must be demo.visualKey; payload.status "resolved"; payload.expectedRevision 0; payload.value must include siteTitle, audience, tone, layout, primaryColor, secondaryColor, features, pickedAt, and author "claude-opus".
- It listens for type "tinkabot.command.result" matching the pending command id and then marks completion.
- The test selectors must exist: #siteTitle, #audience, #tone, #layout, #primaryColor, #secondaryColor, input[value="gallery"], #submitBtn, #generated, [data-proof="tone"], [data-proof="itemKey"].
- #siteTitle and #audience must be editable text inputs, not selects.
- #tone must be a select with an option value "bold"; #layout must be a select with an option value "grid-cards".
- #submitBtn must be type="button"; do not depend on form submission.
- #generated starts with data-complete="false" and becomes data-complete="true" after command result.
- Default form values should be usable before Playwright edits them.
- Do not include raw authority material or these strings in the HTML: tb_items, $KV, BEGIN NATS, PRIVATE KEY, nats://, .creds, jwt, nkey, seed, bearer, credential, credentials, token.
PROMPT
ui_generation_code=$?
set -e
if [[ "$ui_generation_code" -ne 0 ]]; then
  fail "Claude Opus UI generation failed"
fi
UI_GENERATION_RAW="$ui_generation_raw" \
UI_GENERATION="$opus_ui_generation" \
UI_FILE="$ui_file" \
UI_PROVENANCE="$ui_provenance" \
node <<'JS'
const fs = require("node:fs");
const path = require("node:path");
const crypto = require("node:crypto");
const raw = fs.readFileSync(process.env.UI_GENERATION_RAW, "utf8").trim();
function fenced(lang) {
  const re = new RegExp("```" + lang + "\\s*([\\s\\S]*?)```", "i");
  return re.exec(raw)?.[1]?.trim() || "";
}
const jsonText = fenced("json") || raw;
const start = jsonText.indexOf("{");
const end = jsonText.lastIndexOf("}");
if (start < 0 || end < start) throw new Error(`ui generation missing JSON: ${raw}`);
const generated = JSON.parse(jsonText.slice(start, end + 1));
if (generated.kind !== "tinkabot.claudeOpusUiGeneration.v1" || generated.model !== "opus") {
  throw new Error(`ui generation drift: ${JSON.stringify(generated)}`);
}
let html = "";
if (typeof generated.htmlBase64 === "string" && /^[A-Za-z0-9+/=\s]+$/.test(generated.htmlBase64) && generated.htmlBase64.length > 1000) {
  html = Buffer.from(generated.htmlBase64, "base64").toString("utf8");
}
if (!/<!doctype html/i.test(html)) {
  html = fenced("html");
}
if (html.length < 1000) {
  throw new Error("ui generation did not include a substantial HTML payload");
}
const required = [
  ["doctype", /<!doctype html/i],
  ["content.ready", /content\.ready/],
  ["tinkabot.lease", /tinkabot\.lease/],
  ["__frontendAutopilot", /__frontendAutopilot/],
  ["item_submit", /item_submit/],
  ["claude-opus", /claude-opus/],
  ["#siteTitle", /id=["']siteTitle["']/],
  ["#audience", /id=["']audience["']/],
  ["#tone", /id=["']tone["']/],
  ["#layout", /id=["']layout["']/],
  ["#primaryColor", /id=["']primaryColor["']/],
  ["#secondaryColor", /id=["']secondaryColor["']/],
  ["gallery checkbox", /value=["']gallery["']/],
  ["#submitBtn", /id=["']submitBtn["']/],
  ["#generated", /id=["']generated["']/],
  ["tone proof", /data-proof=["']tone["']/],
  ["itemKey proof", /data-proof=["']itemKey["']/],
];
const missing = required.filter(([, pattern]) => !pattern.test(html)).map(([name]) => name);
if (!/<(?:input|textarea)\b(?=[^>]*id=["']siteTitle["'])/i.test(html)) missing.push("#siteTitle text input");
if (!/<(?:input|textarea)\b(?=[^>]*id=["']audience["'])/i.test(html)) missing.push("#audience text input");
if (!/<option\b(?=[^>]*value=["']bold["'])/i.test(html)) missing.push("tone bold option");
if (!/<option\b(?=[^>]*value=["']grid-cards["'])/i.test(html)) missing.push("layout grid-cards option");
if (!/<button\b(?=[^>]*id=["']submitBtn["'])(?=[^>]*type=["']button["'])/i.test(html)) missing.push("#submitBtn button type");
for (const leaseField of ["expectedRevision", "nonce", "frameId", "artifactRevision", "schemaRevision"]) {
  if (!html.includes(leaseField)) missing.push(`intent ${leaseField}`);
}
if (missing.length > 0) throw new Error(`generated UI missing required contract: ${missing.join(", ")}`);
const lower = html.toLowerCase();
const banned = [
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
].filter((needle) => lower.includes(needle));
if (banned.length > 0) throw new Error(`generated UI includes authority terms: ${banned.join(", ")}`);
fs.mkdirSync(path.dirname(process.env.UI_FILE), { recursive: true });
fs.writeFileSync(process.env.UI_FILE, html.endsWith("\n") ? html : `${html}\n`);
const bytes = fs.readFileSync(process.env.UI_FILE);
const sha = crypto.createHash("sha256").update(bytes).digest("hex");
const generation = {
  kind: "tinkabot.claudeOpusUiGeneration.v1",
  model: "opus",
  command: "claude -p --model opus --allowedTools=Read",
  rawOutputPath: process.env.UI_GENERATION_RAW,
  uiPath: process.env.UI_FILE,
  uiSha256: sha,
  htmlBytes: bytes.length,
  generatedAt: new Date().toISOString(),
  notes: typeof generated.notes === "string" ? generated.notes : "",
};
const provenance = {
  kind: "tinkabot.claudeOpusUiProvenance.v1",
  model: "opus",
  command: "claude -p --model opus --allowedTools=Read",
  artifacts: ["options-site.html"],
  authoredBy: "claude-opus",
  uiSha256: sha,
  generationRecord: process.env.UI_GENERATION,
  createdFor: "Frontend-autopilot demo: generated site-options collection iframe for the Tinkabot shell.",
};
fs.writeFileSync(process.env.UI_GENERATION, `${JSON.stringify(generation, null, 2)}\n`);
fs.writeFileSync(process.env.UI_PROVENANCE, `${JSON.stringify(provenance, null, 2)}\n`);
JS

log_step "start packaged Tinkabot with the frontend-autopilot bundle and scoped LLM watcher"
record_proof_command "tinkabot --bundle examples/frontend-autopilot --watcher llm:prefix:$option_prefix"
(cd "$pkg" && TB_DEMO_SESSION="$visual_session" PATH="$bun_path" ./tinkabot \
  --store "$store" \
  --shell 127.0.0.1:0 \
  --bundle examples/frontend-autopilot \
  --watcher "llm:prefix:$option_prefix" \
  >"$log" 2>"$err") &
pid=$!

for _ in {1..300}; do
  if grep -q '^shell  http://127\.0\.0\.1:' "$log" && grep -q "^watcher llm prefix $option_prefix " "$log"; then
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
public_shell_url="http://127.0.0.1:$shell_port"
printf 'shell %s\n' "$public_shell_url"

log_step "disable packaged nats sidecar before user-level Tinkalet commands"
record_proof_command "disable packaged raw nats sidecar before user-level Tinkalet commands"
mv "$pkg/libexec/tinkabot/nats" "$pkg/libexec/tinkabot/nats.disabled"

log_step "import and select the local Tinkalet profile"
record_user_command "tinkalet profile import local --store $store --name local --use"
got="$(run_tinkalet profile import local --store "$store" --name local --use)"
expect_eq "$got" "profile local imported and selected"

log_step "import the isolated LLM watcher profile"
record_worker_command "tinkalet profile import local --store $store/watchers/llm --name llm --use"
got="$(run_watcher_tinkalet profile import local --store "$store/watchers/llm" --name llm --use)"
expect_eq "$got" "profile llm imported and selected"

log_step "RED: register the common Vite frontend handler through Tinkalet"
handler_from="$pkg/examples/frontend-autopilot"
register_out="$dist/register-handler.out"
register_err="$dist/register-handler.err"
record_user_command "tinkalet app handler register vite --from $handler_from --json"
set +e
run_tinkalet app handler register vite --from "$handler_from" --json >"$register_out" 2>"$register_err"
register_code=$?
set -e

if [[ "$register_code" -ne 0 ]]; then
  stdout="$(cat "$register_out")"
  stderr="$(cat "$register_err")"
  write_proof "vite-handler-registration-missing" "$register_code" "$stdout" "$stderr" 0 "" ""
  fail "expected RED: Vite handler registration product command is missing; proof $proof"
fi

log_step "RED: create a realtime frontend app from the Vite handler"
create_out="$dist/create-app.out"
create_err="$dist/create-app.err"
record_user_command "tinkalet app create frontend options-site --handler vite --json"
set +e
run_tinkalet app create frontend options-site --handler vite --json >"$create_out" 2>"$create_err"
create_code=$?
set -e

if [[ "$create_code" -ne 0 ]]; then
  write_proof "frontend-app-create-missing" "$register_code" "$(cat "$register_out")" "$(cat "$register_err")" "$create_code" "$(cat "$create_out")" "$(cat "$create_err")"
  fail "expected RED: frontend app create product command is missing; proof $proof"
fi

created_app="$dist/created-frontend-app.json"
CREATE_OUT="$create_out" CREATED_APP="$created_app" node <<'JS'
const fs = require("node:fs");
const item = JSON.parse(fs.readFileSync(process.env.CREATE_OUT, "utf8"));
const value = item.value || {};
if (
  value.kind !== "tinkabot.frontendApp.v1" ||
  value.name !== "options-site" ||
  value.handler !== "vite" ||
  value.resultKey !== "artifacts.options-site.results.plan" ||
  typeof value.generatedPath !== "string" ||
  !value.generatedPath.startsWith("/artifacts/bundle/")
) {
  throw new Error(`created frontend app record drift: ${JSON.stringify(value)}`);
}
fs.writeFileSync(process.env.CREATED_APP, `${JSON.stringify(value, null, 2)}\n`);
JS
generated_path="$(CREATED_APP="$created_app" node -e 'const fs=require("node:fs"); console.log(JSON.parse(fs.readFileSync(process.env.CREATED_APP,"utf8")).generatedPath)')"
option_key="$(CREATED_APP="$created_app" node -e 'const fs=require("node:fs"); console.log(JSON.parse(fs.readFileSync(process.env.CREATED_APP,"utf8")).resultKey)')"
generated_url="$public_shell_url$generated_path"
visual_url="$public_shell_url/?tb_visual=$option_key&tb_session=$visual_session&tb_generated=$generated_path"
printf 'generated %s\n' "$generated_url"
printf 'visual %s\n' "$visual_url"

for _ in {1..900}; do
  if curl -fsS "$generated_url" >/dev/null 2>/dev/null; then
    break
  fi
  sleep 0.1
done
curl -fsS "$generated_url" >/dev/null || fail "created frontend app generated artifact is not reachable"

log_step "arm live LLM reaction pipeline before browser submits"
record_worker_command "tinkalet watch prefix $option_prefix --limit 5 --timeout 30s --json | claude -p --model opus --allowedTools=Read"
watch_started_ms="$(date +%s%3N)"
llm_reaction_started_ms="$watch_started_ms"
llm_raw="$dist/llm-reaction.raw"
llm_err="$dist/llm-reaction.err"
{
  cat <<'PROMPT'
React to the watch event whose key is artifacts.options-site.results.plan from stdin. Ignore warm-up perf events with keys under artifacts.options-site.results.perf. Do not run tools. Do not use C3. Return JSON only, with:
kind "tinkabot.frontendAutopilotLlmReaction.v1",
status "reacted",
author "claude-opus",
siteTitle,
tone,
layout,
featureCount,
nextStep.

Watch events:
PROMPT
  run_watcher_tinkalet watch prefix "$option_prefix" --limit 5 --timeout 30s --json 2>"$watch_err" | tee "$watch_event"
} | claude -p --model opus --allowedTools=Read >"$llm_raw" 2>"$llm_err" &
llm_pid=$!
sleep 0.5
if ! kill -0 "$llm_pid" 2>/dev/null; then
  fail "live LLM reaction pipeline exited before browser submit"
fi

log_step "drive the Claude-authored generated UI through the trusted shell"
record_proof_command "playwright browser opens trusted shell and submits generated UI options"
PLAYWRIGHT_MODULE="$root/apps/frontend/node_modules/playwright" \
TINKABOT_FRONTEND_AUTOPILOT_URL="$visual_url" \
TINKABOT_FRONTEND_AUTOPILOT_GENERATED_PATH="$generated_path" \
TINKABOT_FRONTEND_AUTOPILOT_OUT="$browser_proof" \
TINKABOT_FRONTEND_AUTOPILOT_KEY="$option_key" \
node <<'JS'
const fs = require("node:fs");
const { chromium } = require(process.env.PLAYWRIGHT_MODULE);

function leakCount(value) {
  const lower = JSON.stringify(value).toLowerCase();
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
    const warmup = await openReadyPage(browser, process.env.TINKABOT_FRONTEND_AUTOPILOT_KEY);
    await warmup.page.close();
    const mainKey = process.env.TINKABOT_FRONTEND_AUTOPILOT_KEY;
    const perfKeys = Array.from({ length: 4 }, (_, i) => `artifacts.options-site.results.perf.${i + 1}`);
    const samples = [];
    for (const key of perfKeys) {
      const sample = await driveSubmit(browser, key);
      samples.push(sample);
      await sample.page.close();
    }
    const finalSample = await driveSubmit(browser, mainKey);
    samples.push(finalSample);
    const page = finalSample.page;
    const frame = finalSample.frame;
    const started = finalSample.started;
	    const shellProof = finalSample.shellProof;
	    const snapshot = finalSample.snapshot;
	    const dom = finalSample.dom;
	    const state = finalSample.state;
	    const dispatches = finalSample.dispatches;
	    const accepted = finalSample.accepted;
	    const renderSamples = samples.map((sample) => sample.warmViteHandlerRenderMs);
	    const submitSamples = samples.map((sample) => sample.submitLatencyMs);
	    const stateSamples = samples.map((sample) => sample.statePushLatencyMs);
	    const proof = {
	      kind: "tinkabot.frontendAutopilotBrowserProof.v1",
	      url: process.env.TINKABOT_FRONTEND_AUTOPILOT_URL,
	      generatedPath: process.env.TINKABOT_FRONTEND_AUTOPILOT_GENERATED_PATH,
	      key: mainKey,
	      elapsedMs: Date.now() - started,
	      warmViteHandlerRenderMs: finalSample.warmViteHandlerRenderMs,
	      warmViteHandlerRenderSamplesMs: renderSamples,
	      warmViteHandlerRenderP95Ms: percentile(renderSamples, 0.95),
      warmViteHandlerRenderP99Ms: percentile(renderSamples, 0.99),
      submitStartedAtMs: finalSample.submitStartedAtMs,
      submitAcceptedAtMs: finalSample.submitAcceptedAtMs,
      submitFinishedAtMs: finalSample.submitFinishedAtMs,
	      submitLatencyMs: finalSample.submitLatencyMs,
	      submitLatencySamplesMs: submitSamples,
	      submitLatencyP95Ms: percentile(submitSamples, 0.95),
	      submitLatencyP99Ms: percentile(submitSamples, 0.99),
	      statePushLatencyMs: finalSample.statePushLatencyMs,
	      statePushLatencySamplesMs: stateSamples,
	      statePushLatencyP95Ms: percentile(stateSamples, 0.95),
	      statePushLatencyP99Ms: percentile(stateSamples, 0.99),
	      submitToStateLatencyMs: finalSample.submitToStateLatencyMs,
	      submitToStateLatencySamplesMs: samples.map((sample) => sample.submitToStateLatencyMs),
	      submitSamples: samples.map((sample) => ({
	        key: sample.key,
	        warmViteHandlerRenderMs: sample.warmViteHandlerRenderMs,
	        submitStartedAtMs: sample.submitStartedAtMs,
	        submitAcceptedAtMs: sample.submitAcceptedAtMs,
	        submitFinishedAtMs: sample.submitFinishedAtMs,
	        submitLatencyMs: sample.submitLatencyMs,
	        statePushLatencyMs: sample.statePushLatencyMs,
	        submitToStateLatencyMs: sample.submitToStateLatencyMs,
	        itemKey: sample.accepted[0]?.itemKey || null,
	      })),
	      acceptedIntents: Array.isArray(shellProof?.accepted) ? shellProof.accepted.length : 0,
	      deniedIntents: Array.isArray(shellProof?.denied) ? shellProof.denied.length : 0,
      acceptedSubmits: accepted.length,
      acceptedDispatches: accepted,
      deniedDispatches: dispatches.filter((d) => d.status !== "accepted").length,
	      itemKey: accepted[0]?.itemKey || null,
	      deliveredLease: shellProof?.lease ?? null,
	      stateDelivery: state?.delivery || "",
	      stateEvents: state?.events || 0,
	      stateLastKey: state?.lastKey || "",
	      stateLastRevision: state?.lastRevision || 0,
	      stateLastObservedAtUnixMs: state?.lastObservedAtUnixMs || 0,
	      dom,
	      snapshot,
	      authorityLeakCount: leakCount({ shellProof, snapshot, dom }),
	    };
    proof.pass = proof.acceptedSubmits === 1 &&
      proof.deniedDispatches === 0 &&
      proof.deniedIntents === 0 &&
	      proof.itemKey === proof.key &&
	      proof.stateDelivery === "trusted-shell.nats-watch.push" &&
	      proof.stateEvents >= 1 &&
	      proof.stateLastKey === proof.key &&
	      proof.dom.complete === "true" &&
	      proof.dom.tone === "bold" &&
      proof.dom.itemKey === proof.key &&
      proof.snapshot?.value?.author === "claude-opus" &&
      proof.authorityLeakCount === 0;
    fs.writeFileSync(process.env.TINKABOT_FRONTEND_AUTOPILOT_OUT, `${JSON.stringify(proof, null, 2)}\n`);
    if (!proof.pass) throw new Error(`browser proof failed: ${JSON.stringify(proof)}`);
    await page.close();
  } finally {
    await browser.close();
  }
})().catch((err) => {
  console.error(err);
  process.exit(1);
});

function urlForKey(key) {
  const url = new URL(process.env.TINKABOT_FRONTEND_AUTOPILOT_URL);
  url.searchParams.set("tb_visual", key);
  return url.toString();
}

async function openReadyPage(browser, key) {
  const page = await browser.newPage({ viewport: { width: 980, height: 720 } });
  const started = Date.now();
  await page.goto(urlForKey(key), { waitUntil: "domcontentloaded" });
  await page.waitForFunction(() => window.__tinkabotProof?.ready?.source === true, null, { timeout: 30000 });
  const frame = page.frames().find((f) => f !== page.mainFrame() && f.url().startsWith("blob:")) ??
    page.frames().find((f) => f !== page.mainFrame() && f.url().includes(process.env.TINKABOT_FRONTEND_AUTOPILOT_GENERATED_PATH));
  if (!frame) throw new Error("generated frontend frame missing");
  await frame.waitForFunction(() => Boolean(window.__frontendAutopilot?.snapshot?.().hasLease), null, { timeout: 10000 });
  return { page, frame, started, readyMs: Date.now() - started };
}

async function driveSubmit(browser, key) {
	  const ready = await openReadyPage(browser, key);
	  const { page, frame, started } = ready;
  await frame.locator("#siteTitle").fill("Tinkabot Studio");
  await frame.locator("#audience").fill("Teams shipping agent-built sites");
  await frame.locator("#tone").selectOption("bold");
  await frame.locator("#layout").selectOption("grid-cards");
  await frame.locator("#primaryColor").evaluate((el) => {
    el.value = "#1f7a8c";
    el.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await frame.locator("#secondaryColor").evaluate((el) => {
    el.value = "#f2c14e";
    el.dispatchEvent(new Event("input", { bubbles: true }));
  });
	  await frame.locator('input[value="gallery"]').check();
	  const stateBefore = await page.evaluate(() => window.__tinkabotProof?.state?.events ?? 0);
	  const submitStartedAtMs = Date.now();
	  await frame.locator("#submitBtn").click();
  await page.waitForFunction(() => {
    const proof = window.__tinkabotProof;
    const dispatches = Array.isArray(proof?.dispatched) ? proof.dispatched : [];
    return dispatches.some((d) => d.command === "item_submit" && d.status === "accepted");
  }, null, { timeout: 30000 });
  const submitAcceptedAtMs = Date.now();
	  await frame.waitForFunction(() => window.__frontendAutopilot?.snapshot?.().complete === true, null, { timeout: 10000 });
	  await page.waitForFunction(({ before, key }) => {
	    const state = window.__tinkabotProof?.state;
	    return state?.events > before &&
	      state.lastKey === key &&
	      state.delivery === "trusted-shell.nats-watch.push" &&
	      Number.isFinite(state.lastObservedAtUnixMs) &&
	      Number.isFinite(state.lastReceivedAtMs);
	  }, { before: stateBefore, key }, { timeout: 10000 });
	  const submitFinishedAtMs = Date.now();
	  const shellProof = await page.evaluate(() => window.__tinkabotProof);
	  const snapshot = await frame.evaluate(() => window.__frontendAutopilot.snapshot());
	  const state = shellProof?.state || {};
  const dom = {
    complete: await frame.locator("#generated").getAttribute("data-complete"),
    tone: await frame.locator('[data-proof="tone"]').textContent(),
    itemKey: await frame.locator('[data-proof="itemKey"]').textContent(),
    text: await frame.locator("body").textContent(),
  };
  const dispatches = Array.isArray(shellProof?.dispatched) ? shellProof.dispatched : [];
  const accepted = dispatches.filter((d) => d.command === "item_submit" && d.status === "accepted");
  return {
    key,
    page,
    frame,
    started,
    warmViteHandlerRenderMs: ready.readyMs,
    submitStartedAtMs,
    submitAcceptedAtMs,
	    submitFinishedAtMs,
	    submitLatencyMs: accepted[0]?.latencyMs ?? null,
	    statePushLatencyMs: Number.isFinite(state.lastReceivedAtMs) && Number.isFinite(state.lastObservedAtUnixMs) ? state.lastReceivedAtMs - state.lastObservedAtUnixMs : null,
	    submitToStateLatencyMs: Number.isFinite(state.lastReceivedAtMs) ? state.lastReceivedAtMs - submitStartedAtMs : null,
	    shellProof,
	    snapshot,
	    state,
	    dom,
    dispatches,
    accepted,
  };
}

function percentile(values, q) {
  const nums = values.filter((value) => Number.isFinite(value)).sort((a, b) => a - b);
  if (nums.length === 0) return null;
  const index = Math.ceil(q * nums.length) - 1;
  return nums[Math.max(0, Math.min(nums.length - 1, index))];
}
JS

log_step "verify live LLM reaction pipeline observed the option item and reacted"
if ! wait "$llm_pid"; then
  llm_pid=""
  if [[ -f "$watch_err" ]]; then
    sed -n '1,120p' "$watch_err" | redact_log >&2 || true
  fi
  fail "live LLM reaction pipeline failed"
fi
llm_pid=""
watch_finished_ms="$(date +%s%3N)"
LLM_RAW="$llm_raw" LLM_REACTION="$llm_reaction" node <<'JS'
const fs = require("node:fs");
const raw = fs.readFileSync(process.env.LLM_RAW, "utf8").trim();
const start = raw.indexOf("{");
const end = raw.lastIndexOf("}");
if (start < 0 || end < start) throw new Error(`llm reaction missing JSON: ${raw}`);
const reaction = JSON.parse(raw.slice(start, end + 1));
if (reaction.kind !== "tinkabot.frontendAutopilotLlmReaction.v1" || reaction.status !== "reacted" || reaction.author !== "claude-opus") {
  throw new Error(`llm reaction drift: ${JSON.stringify(reaction)}`);
}
fs.writeFileSync(process.env.LLM_REACTION, `${JSON.stringify(reaction, null, 2)}\n`);
JS

log_step "ask Claude Opus to review the packaged UI artifact and hash-bound provenance"
record_worker_command "claude -p --model opus --allowedTools=Read"
ui_file="$pkg/examples/frontend-autopilot/ui/options-site.html"
ui_provenance="$pkg/examples/frontend-autopilot/ui/options-site.provenance.json"
ui_sha="$(UI_FILE="$ui_file" node <<'JS'
const fs = require("node:fs");
const crypto = require("node:crypto");
process.stdout.write(crypto.createHash("sha256").update(fs.readFileSync(process.env.UI_FILE)).digest("hex"));
JS
)"
ui_review_raw="$dist/opus-ui-review.raw"
ui_review_err="$dist/opus-ui-review.err"
set +e
claude -p --model opus --allowedTools=Read >"$ui_review_raw" 2>"$ui_review_err" <<PROMPT
Review this packaged generated UI artifact and provenance. Do not run tools. Do not use C3. Return JSON only, with:
kind "tinkabot.claudeOpusUiReview.v1",
verdict "pass" or "fail",
model "opus",
authoredBy "claude-opus" if the evidence supports it, otherwise "unknown",
uiSha256 "$ui_sha",
codexDirectChangedLines number,
notes string.

Pass only if the generation record shows the reviewed HTML bytes came from a Claude Opus `claude -p` generation step, the provenance hash matches those bytes, the artifact is a usable option-collection frontend, and there is no evidence of Codex-authored UI bytes.

Generation record:
$(cat "$opus_ui_generation")

Artifact provenance:
$(cat "$ui_provenance")

Artifact sha256:
$ui_sha

Artifact content:
$(cat "$ui_file")
PROMPT
ui_review_code=$?
set -e
if [[ "$ui_review_code" -ne 0 ]]; then
  fail "Claude Opus UI review failed"
fi
UI_REVIEW_RAW="$ui_review_raw" UI_REVIEW="$opus_ui_review" UI_SHA="$ui_sha" node <<'JS'
const fs = require("node:fs");
const raw = fs.readFileSync(process.env.UI_REVIEW_RAW, "utf8").trim();
const start = raw.indexOf("{");
const end = raw.lastIndexOf("}");
if (start < 0 || end < start) throw new Error(`ui review missing JSON: ${raw}`);
const review = JSON.parse(raw.slice(start, end + 1));
if (
  review.kind !== "tinkabot.claudeOpusUiReview.v1" ||
  review.verdict !== "pass" ||
  review.model !== "opus" ||
  review.authoredBy !== "claude-opus" ||
  review.uiSha256 !== process.env.UI_SHA ||
  review.codexDirectChangedLines !== 0
) {
  throw new Error(`ui review drift: ${JSON.stringify(review)}`);
}
fs.writeFileSync(process.env.UI_REVIEW, `${JSON.stringify(review, null, 2)}\n`);
JS

log_step "write terminal frontend-autopilot proof"
PROOF="$proof" \
REGISTER_OUT="$register_out" \
REGISTER_ERR="$register_err" \
CREATE_OUT="$create_out" \
CREATE_ERR="$create_err" \
BROWSER_PROOF="$browser_proof" \
WATCH_EVENT="$watch_event" \
WATCH_STARTED_MS="$watch_started_ms" \
WATCH_FINISHED_MS="$watch_finished_ms" \
LLM_REACTION_STARTED_MS="$llm_reaction_started_ms" \
LLM_REACTION="$llm_reaction" \
PROVENANCE="$pkg/examples/frontend-autopilot/ui/options-site.provenance.json" \
UI_GENERATION="$opus_ui_generation" \
UI_REVIEW="$opus_ui_review" \
USER_COMMANDS="$user_commands" \
WORKER_COMMANDS="$worker_commands" \
PROOF_COMMANDS="$proof_commands" \
MANUAL_UNBLOCKS="$manual_unblocks" \
NATS_SIDECAR_DISABLED="$([[ -f "$pkg/libexec/tinkabot/nats.disabled" && ! -f "$pkg/libexec/tinkabot/nats" ]] && printf '1' || printf '0')" \
HANDLER_FROM="$handler_from" \
SHELL_URL="$public_shell_url" \
GENERATED_URL="$generated_url" \
GENERATED_PATH="$generated_path" \
VISUAL_URL="$visual_url" \
OPTION_KEY="$option_key" \
node <<'JS'
const fs = require("node:fs");
const read = (name) => JSON.parse(fs.readFileSync(process.env[name], "utf8"));
const registerStdout = fs.readFileSync(process.env.REGISTER_OUT, "utf8");
const registerStderr = fs.readFileSync(process.env.REGISTER_ERR, "utf8");
const createStdout = fs.readFileSync(process.env.CREATE_OUT, "utf8");
const createStderr = fs.readFileSync(process.env.CREATE_ERR, "utf8");
const created = JSON.parse(createStdout);
const browser = read("BROWSER_PROOF");
function jsonLines(path) {
  return fs.readFileSync(path, "utf8").trim().split(/\n+/).filter(Boolean).map((line) => JSON.parse(line));
}
const watchEvents = jsonLines(process.env.WATCH_EVENT);
const watch = watchEvents.find((event) => event.key === process.env.OPTION_KEY) || {};
const reaction = read("LLM_REACTION");
const provenance = read("PROVENANCE");
const uiGeneration = read("UI_GENERATION");
const uiReview = read("UI_REVIEW");
function jsonl(path) {
  if (!path || !fs.existsSync(path)) return [];
  return fs.readFileSync(path, "utf8").trim().split(/\n+/).filter(Boolean).map((line) => JSON.parse(line));
}
const userCommands = jsonl(process.env.USER_COMMANDS);
const workerCommands = jsonl(process.env.WORKER_COMMANDS);
const proofCommands = jsonl(process.env.PROOF_COMMANDS);
const manualUnblocks = jsonl(process.env.MANUAL_UNBLOCKS);
const appValue = created.value || {};
const optionValue = watch.value || {};
const watchStartedMs = Number(process.env.WATCH_STARTED_MS || NaN);
const watchFinishedMs = Number(process.env.WATCH_FINISHED_MS || NaN);
const llmReactionStartedMs = Number(process.env.LLM_REACTION_STARTED_MS || NaN);
const submitByKey = new Map((browser.submitSamples || []).map((sample) => [sample.key, sample]));
const watchLatencySamples = watchEvents.map((event) => {
  const sample = submitByKey.get(event.key);
  const observed = Number(event.observedAtUnixMs);
  if (!sample || !Number.isFinite(observed) || !Number.isFinite(sample.submitStartedAtMs)) return null;
  return observed - sample.submitStartedAtMs;
}).filter((value) => Number.isFinite(value) && value >= 0);
const watchObservedMs = Number(watch.observedAtUnixMs);
const watchMs = Number.isFinite(watchObservedMs) && Number.isFinite(browser.submitStartedAtMs)
  ? watchObservedMs - browser.submitStartedAtMs
  : null;
function percentile(values, q) {
  const nums = values.filter((value) => Number.isFinite(value)).sort((a, b) => a - b);
  if (nums.length === 0) return null;
  const index = Math.ceil(q * nums.length) - 1;
  return nums[Math.max(0, Math.min(nums.length - 1, index))];
}
const watchP95 = percentile(watchLatencySamples, 0.95);
const watchP99 = percentile(watchLatencySamples, 0.99);
const pushedMs = browser.statePushLatencyP95Ms ?? browser.statePushLatencyMs ?? 0;
const renderMs = browser.warmViteHandlerRenderP95Ms ?? browser.warmViteHandlerRenderMs ?? 0;
const authorityLeakCount = browser.authorityLeakCount || 0;
const uiArtifacts = Array.isArray(provenance.artifacts) ? provenance.artifacts : [];
const generationPass = uiGeneration.kind === "tinkabot.claudeOpusUiGeneration.v1" &&
  uiGeneration.model === "opus" &&
  uiGeneration.command === "claude -p --model opus --allowedTools=Read" &&
  typeof uiGeneration.uiSha256 === "string" &&
  uiGeneration.uiSha256.length === 64 &&
  uiGeneration.htmlBytes > 1000;
const provenancePass = provenance.kind === "tinkabot.claudeOpusUiProvenance.v1" &&
  provenance.model === "opus" &&
  provenance.authoredBy === "claude-opus" &&
  provenance.uiSha256 === uiGeneration.uiSha256 &&
  uiArtifacts.includes("options-site.html");
const reviewPass = uiReview.kind === "tinkabot.claudeOpusUiReview.v1" &&
  uiReview.verdict === "pass" &&
  uiReview.model === "opus" &&
  uiReview.authoredBy === "claude-opus" &&
  uiReview.uiSha256 === uiGeneration.uiSha256 &&
  uiReview.codexDirectChangedLines === 0;
const claudeOpusUiArtifactCount = generationPass && provenancePass && reviewPass ? 1 : 0;
const nonOpusAuthoredUiArtifactCount = Math.max(0, uiArtifacts.length - claudeOpusUiArtifactCount);
const codexDirectUiChangedLines = generationPass && reviewPass ? uiReview.codexDirectChangedLines : 1;
const natsNativeChecks = {
  packagedRawNatsCliDisabled: process.env.NATS_SIDECAR_DISABLED === "1",
  userCommandsUseTinkalet: userCommands.every((entry) => String(entry.command || "").startsWith("tinkalet ")),
  browserSubmittedViaShell: browser.acceptedSubmits === 1 && browser.itemKey === process.env.OPTION_KEY,
  createdAppDroveRender: appValue.generatedPath === process.env.GENERATED_PATH &&
    browser.generatedPath === appValue.generatedPath &&
    appValue.resultKey === process.env.OPTION_KEY,
  browserStateDeliveredViaNatsWatch: browser.stateDelivery === "trusted-shell.nats-watch.push" &&
    browser.stateEvents >= 1 &&
    browser.stateLastKey === process.env.OPTION_KEY,
  liveWatcherSawKvItem: watchEvents.length >= 5 && watch.kind === "tinkalet.itemEvent.v1" && watch.source === "watch",
  llmReactedFromWatchEvent: reaction.status === "reacted" && Number.isFinite(llmReactionStartedMs) && llmReactionStartedMs <= browser.submitStartedAtMs,
};
const nonNatsProductPathCount = Object.values(natsNativeChecks).filter((ok) => !ok).length;
const pass = appValue.resultKey === process.env.OPTION_KEY &&
  appValue.generatedPath === process.env.GENERATED_PATH &&
  browser.pass === true &&
  watch.key === process.env.OPTION_KEY &&
  watch.status === "resolved" &&
  watch.source === "watch" &&
  reaction.status === "reacted" &&
  optionValue.author === "claude-opus" &&
  userCommands.length <= 3 &&
  manualUnblocks.length === 0 &&
  nonNatsProductPathCount === 0 &&
  codexDirectUiChangedLines === 0 &&
  nonOpusAuthoredUiArtifactCount === 0 &&
  claudeOpusUiArtifactCount >= 1 &&
  authorityLeakCount === 0 &&
  pushedMs <= 100 &&
  renderMs <= 750 &&
  watchLatencySamples.length >= 5 &&
  watchP95 !== null &&
  watchP99 !== null &&
  watchP95 <= 250 &&
  watchP99 <= 500;
const proof = {
  kind: "tinkabot.frontendAutopilotProof.v1",
  pass,
  clean_install_to_kv_reaction_journey_passing: pass,
  frontend_autopilot_reference_families: pass ? 7 : 6,
  frontend_autopilot_target_families: 7,
  frontend_autopilot_first_failure: pass ? null : "terminal-proof-threshold",
  post_install_user_command_count: userCommands.length,
  post_install_user_commands: userCommands,
  worker_command_count: workerCommands.length,
  worker_commands: workerCommands,
  proof_command_count: proofCommands.length,
  proof_commands: proofCommands,
  manual_unblock_count: manualUnblocks.length,
  manual_unblocks: manualUnblocks,
  non_nats_product_path_count: nonNatsProductPathCount,
  nats_native_checks: natsNativeChecks,
  codex_direct_ui_changed_lines: codexDirectUiChangedLines,
  non_opus_authored_ui_artifact_count: nonOpusAuthoredUiArtifactCount,
  claude_opus_ui_artifact_count: claudeOpusUiArtifactCount,
  authority_leak_count: authorityLeakCount,
  option_write_to_llm_watch_p95_ms: watchP95,
  option_write_to_llm_watch_p99_ms: watchP99,
  browser_pushed_state_p95_ms: pushedMs,
  browser_pushed_state_p99_ms: browser.statePushLatencyP99Ms ?? pushedMs,
  browser_submit_request_p95_ms: browser.submitLatencyP95Ms ?? browser.submitLatencyMs ?? 0,
  browser_submit_request_p99_ms: browser.submitLatencyP99Ms ?? browser.submitLatencyMs ?? 0,
  warm_vite_handler_render_p95_ms: renderMs,
  warm_vite_handler_render_p99_ms: browser.warmViteHandlerRenderP99Ms ?? renderMs,
  performance_sample_count: watchLatencySamples.length,
  performance_samples: {
    optionWriteToLlmWatchMs: watchLatencySamples,
    browserPushedStateMs: browser.statePushLatencySamplesMs || [],
    browserSubmitToStateMs: browser.submitToStateLatencySamplesMs || [],
    browserSubmitRequestMs: browser.submitLatencySamplesMs || [],
    warmViteHandlerRenderMs: browser.warmViteHandlerRenderSamplesMs || [],
  },
  shell_url: process.env.SHELL_URL,
  generated_url: process.env.GENERATED_URL,
  visual_url: process.env.VISUAL_URL,
  register_handler: {
    command: `tinkalet app handler register vite --from ${process.env.HANDLER_FROM} --json`,
    exit_code: 0,
    stdout: registerStdout,
    stderr: registerStderr,
  },
  create_frontend_app: {
    command: "tinkalet app create frontend options-site --handler vite --json",
    exit_code: 0,
    stdout: createStdout,
    stderr: createStderr,
    appValue,
  },
  browser,
  llm_watch_events: watchEvents,
  llm_watch_event: watch,
  llm_watch_timing: {
    watchStartedMs,
    watchObservedMs,
    watchFinishedMs,
    llmReactionStartedMs,
    llmReactionMode: "live-watch-pipeline",
    browserSubmitStartedAtMs: browser.submitStartedAtMs,
    browserSubmitAcceptedAtMs: browser.submitAcceptedAtMs,
    browserSubmitFinishedAtMs: browser.submitFinishedAtMs,
  },
  llm_reaction: reaction,
  option_item_from_watch: {
    key: watch.key,
    status: watch.status,
    value: optionValue,
    revision: watch.revision,
    source: watch.source,
  },
  opus_ui_generation: uiGeneration,
  opus_ui_provenance: provenance,
  opus_ui_review: uiReview,
};
fs.writeFileSync(process.env.PROOF, `${JSON.stringify(proof, null, 2)}\n`);
if (!pass) throw new Error(`terminal proof failed: ${JSON.stringify(proof)}`);
JS

log_step "demo passed"
printf 'package root %s\n' "$pkg"
printf 'artifacts %s\n' "$dist"
printf 'frontend proof %s\n' "$proof"
printf 'open %s\n' "$visual_url"
