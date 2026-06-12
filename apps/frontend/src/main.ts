import { accept, frameAttrs, makeLease, type BrowserCommandIntent } from "./isolation";
import { generatedUrl } from "./fixture";
import "./style.css";

declare global {
  interface Window {
    __tinkabotProof?: Proof;
  }
}

interface Proof {
  sandbox: string;
  accepted: BrowserCommandIntent[];
  denied: string[];
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

const attrs = frameAttrs("generated artifact proof");

const proof: Proof = {
  sandbox: attrs.sandbox,
  accepted: [],
  denied: [],
};
window.__tinkabotProof = proof;

app.innerHTML = `
  <main class="shell">
    <section>
      <p class="eyebrow">trusted shell</p>
      <h1>Tinkabot</h1>
      <p>Opaque generated content is isolated behind a leased message channel.</p>
      <dl>
        <div><dt>Sandbox</dt><dd data-proof="sandbox"></dd></div>
        <div><dt>Accepted</dt><dd data-proof="accepted">0</dd></div>
        <div><dt>Denied</dt><dd data-proof="denied">0</dd></div>
        <div><dt>Cookie Probe</dt><dd data-proof="cookie">pending</dd></div>
      </dl>
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

const lease = makeLease({
  frameId: "frame-001",
  sessionId: "session-001",
  capabilityId: "cap-001",
  artifactId: "artifact-001",
  artifactRevision: "artifact.rev.7",
  schemaRevision: "schema.rev.1",
  commands: ["select_artifact"],
  sessions: [],
  chain: {
    chainId: "chain-001",
    rootId: "root-001",
    hop: 0,
    maxHops: 5,
  },
});

window.addEventListener("message", (event) => {
  if (event.source !== frame.contentWindow) return;
  proof.ready = {
    origin: event.origin,
    source: true,
  };
  render();

  const msg = event.data;
  if (msg?.type === "content.ready") {
    frame.contentWindow?.postMessage({ type: "tinkabot.lease", lease }, "*");
    return;
  }
  if (msg?.type === "content.probe") {
    proof.probe = { cookie: msg.cookie, storage: msg.storage };
    render();
    return;
  }

  try {
    const intent = accept(lease, event.source, frame.contentWindow, msg);
    proof.accepted.push(intent);
  } catch (error) {
    proof.denied.push(error instanceof Error ? error.message : String(error));
  }
  render();
});

frame.src = generatedUrl();

render();

function render() {
  app.querySelector('[data-proof="sandbox"]')!.textContent = proof.sandbox;
  app.querySelector('[data-proof="accepted"]')!.textContent = String(
    proof.accepted.length,
  );
  app.querySelector('[data-proof="denied"]')!.textContent = String(proof.denied.length);
  app.querySelector('[data-proof="cookie"]')!.textContent =
    proof.probe?.cookie || "empty";
}
