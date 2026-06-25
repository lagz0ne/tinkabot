import {
  accept,
  denyRaw,
  frameAttrs,
  makeLease,
  type BrowserCommandIntent,
} from "./isolation";
import { generatedUrl } from "./fixture";
import { commandClient, observe, type CommandClient, type StateEvent } from "./observe";
import "./style.css";

declare global {
  interface Window {
    __tinkabotProof?: Proof;
  }
}

interface Dispatch {
  command: string;
  commandId: string;
  status: string;
  reason?: string;
  latencyMs: number;
  itemKey?: string;
  payloadKey?: string;
}

interface Proof {
  sandbox: string;
  accepted: BrowserCommandIntent[];
  denied: string[];
  dispatched: Dispatch[];
  state: {
    delivery: string;
    events: number;
    lastRevision: number;
    errors: string[];
  };
  ready?: {
    origin: string;
    source: boolean;
  };
  probe?: {
    cookie: string;
    storage: string;
  };
}

const root = document.querySelector("#app");
if (!root) throw new Error("missing app root");
const app = root;
const params = new URLSearchParams(location.search);
const appId = param("tb_app");
const participantId = param("tb_participant");
const participantMode = appId !== "" && participantId !== "";
const boardMode = param("tb_board") === "1";
const chessMode = param("tb_chess") === "1";
const chessBoard = param("tb_board_no") || "board-001";
const chessName = param("tb_name");
const typeMode = param("tb_type") === "1";
const typeRace = param("tb_race_no") || "race-001";
const typeAlias = param("tb_alias");
const visualKey = param("tb_visual");
const visualMode = visualKey !== "";
const visualChoice = param("tb_choice") || "diagram-a";
const visualArtifactId = artifactFromVisualKey(visualKey) || "artifact-001";
const sessionId =
  param("tb_session") || (participantMode ? "demo-001" : visualMode ? "visual-001" : "session-001");
const stateKey =
  param("tb_state") ||
  (participantMode
    ? `apps.${appId}.state.${
        chessMode
          ? `chess.${chessBoard}`
          : typeMode
            ? `typerace.${typeRace}`
            : boardMode
              ? "board"
              : "browser"
      }`
    : "");
const autoActions = Number.parseInt(param("tb_auto") || "0", 10);
const intervalMs = Number.parseInt(param("tb_interval_ms") || "25", 10);
const attrs = frameAttrs("generated artifact proof");
const fullAppMode = chessMode || typeMode;

const proof: Proof = {
  sandbox: attrs.sandbox,
  accepted: [],
  denied: [],
  dispatched: [],
  state: {
    delivery: "",
    events: 0,
    lastRevision: 0,
    errors: [],
  },
};
window.__tinkabotProof = proof;

app.innerHTML = fullAppMode
  ? `
    <main class="${chessMode ? "chess-shell" : "app-shell"}">
      <iframe
        data-proof="frame"
        title="${chessMode ? "Chess" : "Typeracing"}"
        sandbox="${attrs.sandbox}"
        referrerpolicy="${attrs.referrerPolicy}"
      ></iframe>
    </main>
  `
  : `
    <main class="shell">
      <section>
        <p class="eyebrow">trusted shell</p>
        <h1>Tinkabot</h1>
        <p>Opaque generated content is isolated behind a leased message channel.</p>
        <dl>
          <div><dt>Sandbox</dt><dd data-proof="sandbox"></dd></div>
          <div><dt>Accepted</dt><dd data-proof="accepted">0</dd></div>
          <div><dt>Dispatched</dt><dd data-proof="dispatched">0</dd></div>
          <div><dt>Denied</dt><dd data-proof="denied">0</dd></div>
          <div><dt>Cookie Probe</dt><dd data-proof="cookie">pending</dd></div>
          <div><dt>Participant</dt><dd data-proof="participant">none</dd></div>
        </dl>
      </section>
      <section class="observe">
        <p class="eyebrow">session observation</p>
        <label>Session <input data-obs="sid" /></label>
        <button data-obs="go">Observe</button>
        <pre data-obs="log"></pre>
      </section>
      <iframe
        data-proof="frame"
        title="${attrs.title}"
        sandbox="${attrs.sandbox}"
        referrerpolicy="${attrs.referrerPolicy}"
      ></iframe>
    </main>
  `;

const frame = app.querySelector<HTMLIFrameElement>("iframe");
if (!frame) throw new Error("missing generated frame");
const generatedFrame = frame;

const lease = makeLease({
  frameId: "frame-001",
  sessionId,
  capabilityId: participantMode
    ? `cap-${appId}-${participantId}`
    : visualMode
      ? `cap-${visualArtifactId}-visual`
      : "cap-001",
  artifactId: participantMode
    ? `artifact-${appId}-${participantId}`
    : visualMode
      ? visualArtifactId
      : "artifact-001",
  artifactRevision: "artifact.rev.7",
  schemaRevision: "schema.rev.1",
  appId: participantMode ? appId : undefined,
  participantId: participantMode ? participantId : undefined,
  commands: participantMode
    ? ["participant_read", "participant_action"]
    : visualMode
      ? ["item_submit"]
      : ["select_artifact"],
  sessions: participantMode || visualMode ? [sessionId] : [],
  chain: {
    chainId: participantMode ? `chain-${appId}` : visualMode ? `chain-${visualArtifactId}` : "chain-001",
    rootId: participantMode ? `root-${appId}` : visualMode ? `root-${visualArtifactId}` : "root-001",
    hop: 0,
    maxHops: 5,
  },
});

let client: Promise<CommandClient> | undefined;
let observation: ReturnType<typeof observe> | undefined;
let stateWatch: Promise<() => void> | undefined;

window.addEventListener("message", (event) => {
  if (event.source !== generatedFrame.contentWindow) return;
  proof.ready = {
    origin: event.origin,
    source: true,
  };
  render();

  const msg = event.data;
  if (msg?.type === "content.ready") {
    generatedFrame.contentWindow?.postMessage(
      {
        type: "tinkabot.lease",
        lease,
        demo: {
          stateKey,
          chess: chessMode,
          boardNo: chessBoard,
          playerName: chessName,
          typeRace: typeMode,
          raceNo: typeRace,
          alias: typeAlias,
          board: boardMode,
          autoActions: Number.isFinite(autoActions) ? Math.max(0, autoActions) : 0,
          intervalMs: Number.isFinite(intervalMs) ? Math.max(1, intervalMs) : 25,
          visualKey,
          choice: visualChoice,
        },
      },
      "*",
    );
    startStateWatch();
    return;
  }
  if (msg?.type === "content.probe") {
    proof.probe = { cookie: msg.cookie, storage: msg.storage };
    render();
    return;
  }

  try {
    const intent = accept(lease, event.source, generatedFrame.contentWindow, msg);
    proof.accepted.push(intent);
    if (participantMode || visualMode) void dispatch(intent);
  } catch (error) {
    proof.denied.push(error instanceof Error ? error.message : String(error));
  }
  render();
});

generatedFrame.src = generatedUrl();

const obsLog = app.querySelector<HTMLPreElement>('[data-obs="log"]');
const obsSid = app.querySelector<HTMLInputElement>('[data-obs="sid"]');
const obsGo = app.querySelector<HTMLButtonElement>('[data-obs="go"]');
if (obsLog && obsSid && obsGo) {
  obsSid.value = sessionId;
  obsGo.addEventListener("click", () => {
    obsLog.textContent = "";
    if (observation) void observation.then((nc) => nc.close()).catch(() => undefined);
    observation = observe(obsSid.value, (text) => {
      obsLog.textContent += text;
      obsLog.scrollTop = obsLog.scrollHeight;
    });
    observation.catch((err) => {
      obsLog.textContent = `observe failed: ${err instanceof Error ? err.message : String(err)}`;
    });
  });
}

window.addEventListener("beforeunload", () => {
  if (stateWatch) void stateWatch.then((stop) => stop()).catch(() => undefined);
  if (client) void client.then((c) => c.close()).catch(() => undefined);
  if (observation) void observation.then((nc) => nc.close()).catch(() => undefined);
});

render();

async function dispatch(intent: BrowserCommandIntent) {
  const started = performance.now();
  let response: unknown;
  try {
    enforceAppStateScope(intent);
    response = await (await getClient()).request(intent);
    denyRaw(response);
    const record = recordDispatch(intent, response, performance.now() - started);
    generatedFrame.contentWindow?.postMessage(
      { type: "tinkabot.command.result", commandId: intent.commandId, response: record.response },
      "*",
    );
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    proof.dispatched.push({
      command: intent.command,
      commandId: intent.commandId,
      status: "failed",
      reason: message,
      latencyMs: Math.round(performance.now() - started),
    });
    generatedFrame.contentWindow?.postMessage(
      { type: "tinkabot.command.result", commandId: intent.commandId, error: message },
      "*",
    );
  }
  render();
}

function getClient() {
  client ??= commandClient(lease.sessionId).catch((error) => {
    client = undefined;
    throw error;
  });
  return client;
}

function startStateWatch() {
  if (!participantMode || stateKey === "" || stateWatch) return;
  stateWatch = getClient()
    .then((c) => c.watch(stateWatchIntent(), postState))
    .catch((error) => {
      stateWatch = undefined;
      proof.state.errors.push(error instanceof Error ? error.message : String(error));
      render();
      return () => undefined;
    });
}

function stateWatchIntent(): BrowserCommandIntent {
  return {
    kind: "browser.command_intent",
    type: "content.intent",
    command: "participant_watch",
    commandId: `watch-${lease.frameId}-${Date.now()}`,
    expectedRevision: lease.artifactRevision,
    payload: { key: stateKey },
    context: {
      sessionId: lease.sessionId,
      capabilityId: lease.capabilityId,
      artifactId: lease.artifactId,
      artifactRevision: lease.artifactRevision,
      frameId: lease.frameId,
      appId: lease.appId,
      participantId: lease.participantId,
      chain: lease.chain,
    },
  };
}

function postState(event: StateEvent) {
  try {
    denyRaw(event);
    proof.state.delivery = event.source;
    proof.state.events += 1;
    proof.state.lastRevision = event.revision;
    generatedFrame.contentWindow?.postMessage(
      {
        type: "tinkabot.state",
        source: event.source,
        item: {
          key: event.key,
          status: event.status,
          value: event.value,
          revision: event.revision,
          observedAt: event.observedAt,
        },
      },
      "*",
    );
  } catch (error) {
    proof.state.errors.push(error instanceof Error ? error.message : String(error));
  }
  render();
}

function enforceAppStateScope(intent: BrowserCommandIntent) {
  if (!chessMode && !typeMode) return;
  const payload = asRecord(intent.payload);
  if (intent.command === "participant_read" && payload.key !== stateKey) {
    throw new Error(`${chessMode ? "chess board" : "typerace"} read denied`);
  }
  if (intent.command === "participant_action" && payload.stateKey !== stateKey) {
    throw new Error(`${chessMode ? "chess board" : "typerace"} action denied`);
  }
}

function recordDispatch(
  intent: BrowserCommandIntent,
  response: unknown,
  latency: number,
) {
  const rec = asRecord(response);
  const item = asRecord(rec.item);
  const payload = asRecord(intent.payload);
  const payloadKey = typeof payload.key === "string" ? payload.key : payload.stateKey;
  const entry = {
    command: intent.command,
    commandId: intent.commandId,
    status: typeof rec.status === "string" ? rec.status : "unknown",
    reason: typeof rec.reason === "string" ? rec.reason : undefined,
    latencyMs: Math.round(latency),
    itemKey: typeof item.key === "string" ? item.key : undefined,
    payloadKey: typeof payloadKey === "string" ? payloadKey : undefined,
    response,
  };
  proof.dispatched.push(entry);
  return entry;
}

function render() {
  const sandbox = app.querySelector('[data-proof="sandbox"]');
  if (!sandbox) return;
  sandbox.textContent = proof.sandbox;
  app.querySelector('[data-proof="accepted"]')!.textContent = String(
    proof.accepted.length,
  );
  app.querySelector('[data-proof="dispatched"]')!.textContent = String(
    proof.dispatched.length,
  );
  app.querySelector('[data-proof="denied"]')!.textContent = String(proof.denied.length);
  app.querySelector('[data-proof="cookie"]')!.textContent =
    proof.probe?.cookie || "empty";
  app.querySelector('[data-proof="participant"]')!.textContent = participantMode
    ? `${appId}/${participantId}`
    : visualMode
      ? "visual"
    : "none";
}

function param(name: string) {
  return params.get(name)?.trim() ?? "";
}

function artifactFromVisualKey(key: string) {
  const match = /^artifacts\.([A-Za-z0-9_-]+)\.results\./.exec(key);
  return match?.[1] ?? "";
}

function asRecord(value: unknown): Record<string, unknown> {
  if (typeof value === "object" && value !== null) return value as Record<string, unknown>;
  return {};
}
