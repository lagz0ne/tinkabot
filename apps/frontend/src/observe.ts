// Trusted-shell session observation over the cookie-gated WebSocket: mint a
// bearer viewer grant, connect through /session/ws (the cookie rides the
// same-origin upgrade), and stream the session's token frames. The shell
// holds the connection; untrusted generated content never does.
import { connect, jwtAuthenticator, type NatsConnection } from "nats.ws";

export interface Grant {
  jwt: string;
  deliverSubject: string;
  wsTicket: string;
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
