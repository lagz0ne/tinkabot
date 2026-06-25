// Trusted-shell session observation over the cookie-gated WebSocket: mint a
// bearer viewer grant, connect through /session/ws (the cookie rides the
// same-origin upgrade), and stream the session's token frames. The shell
// holds the connection; untrusted generated content never does.
import { connect, jwtAuthenticator, type NatsConnection } from "nats.ws";
import type { BrowserCommandIntent } from "./isolation";

export interface Grant {
  jwt: string;
  deliverSubject: string;
  stateSubject: string;
  wsTicket: string;
}

export interface StateEvent {
  kind: "tinkabot.browserState.v1";
  source: "trusted-shell.nats-watch.push";
  key: string;
  status: string;
  value: unknown;
  revision: number;
  observedAt: string;
}

export interface CommandClient {
  request(intent: BrowserCommandIntent): Promise<unknown>;
  watch(intent: BrowserCommandIntent, onState: (event: StateEvent) => void): Promise<() => void>;
  close(): Promise<void>;
}

// frameLine extracts displayable text from one session frame: token frames
// carry the transcript text; chunk frames and malformed lines render nothing.
export function frameLine(data: string): string | null {
  try {
    const f = JSON.parse(data);
    if (f?.kind === "session.frame" && f.frame === "token" && typeof f.text === "string") {
      return f.text;
    }
  } catch {
    // not a frame line
  }
  return null;
}

export async function mintViewer(sessionId: string): Promise<Grant> {
  const res = await fetch("/session/viewer", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ sessionId }),
  });
  if (!res.ok) throw new Error(`viewer mint failed: ${res.status}`);
  return res.json();
}

export async function observe(
  sessionId: string,
  onText: (text: string) => void,
): Promise<NatsConnection> {
  const grant = await mintViewer(sessionId);
  // Browsers do not reliably attach SameSite cookies to ws:// upgrades; the
  // single-use ticket from the cookie-gated mint carries that authority.
  const proto = location.protocol === "https:" ? "wss" : "ws";
  const nc = await connect({
    servers: `${proto}://${location.host}/session/ws?t=${grant.wsTicket}`,
    authenticator: jwtAuthenticator(grant.jwt),
  });
  void (async () => {
    const decoder = new TextDecoder();
    for await (const m of nc.subscribe(grant.deliverSubject)) {
      const text = frameLine(decoder.decode(m.data));
      if (text !== null) onText(text);
    }
  })();
  return nc;
}

export async function commandClient(sessionId: string): Promise<CommandClient> {
  const grant = await mintViewer(sessionId);
  const proto = location.protocol === "https:" ? "wss" : "ws";
  const nc = await connect({
    servers: `${proto}://${location.host}/session/ws?t=${grant.wsTicket}`,
    authenticator: jwtAuthenticator(grant.jwt),
  });

  return {
    async request(intent) {
      const reply = await nc.request("tb.app.browser.command", JSON.stringify(intent), {
        timeout: 5000,
      });
      return decodeJson(reply.data);
    },
    async watch(intent, onState) {
      const watchIntent = withDelivery(intent, grant.stateSubject);
      const first = await request(nc, watchIntent);
      const subject = deliverySubject(first, grant.stateSubject);
      const sub = nc.subscribe(subject);
      let live = true;
      void (async () => {
        for await (const msg of sub) {
          if (!live) continue;
          onState(decodeJson(msg.data) as StateEvent);
        }
      })();
      await request(nc, { ...watchIntent, commandId: `${watchIntent.commandId}-attach` });
      return () => {
        live = false;
        sub.unsubscribe();
      };
    },
    close() {
      return nc.close();
    },
  };
}

function withDelivery(intent: BrowserCommandIntent, delivery: string): BrowserCommandIntent {
  const payload = isRec(intent.payload) ? { ...intent.payload, delivery } : { delivery };
  return { ...intent, payload };
}

async function request(nc: NatsConnection, intent: BrowserCommandIntent) {
  const reply = await nc.request("tb.app.browser.command", JSON.stringify(intent), {
    timeout: 5000,
  });
  return decodeJson(reply.data);
}

function deliverySubject(value: unknown, prefix: string) {
  if (!isRec(value) || value.status !== "accepted" || typeof value.deliverySubject !== "string") {
    throw new Error("state watch denied");
  }
  if (!value.deliverySubject.startsWith(`${prefix}.`)) {
    throw new Error("state watch escaped viewer grant");
  }
  return value.deliverySubject;
}

export function decodeJson(data: Uint8Array): unknown {
  const text = new TextDecoder().decode(data);
  return JSON.parse(text);
}

function isRec(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
