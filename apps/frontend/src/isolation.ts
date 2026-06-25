export interface Chain {
  chainId: string;
  rootId: string;
  parentId?: string;
  hop: number;
  maxHops: number;
}

export interface Lease {
  frameId: string;
  nonce: string;
  sessionId: string;
  capabilityId: string;
  artifactId: string;
  artifactRevision: string;
  schemaRevision: string;
  appId?: string;
  participantId?: string;
  chain: Chain;
  commands: readonly string[];
  sessions: readonly string[];
}

export interface ContentIntent {
  type: "content.intent";
  command: string;
  commandId: string;
  expectedRevision: string;
  nonce: string;
  frameId: string;
  artifactRevision: string;
  schemaRevision: string;
  appId?: string;
  participantId?: string;
  sessionId?: string;
  payload?: unknown;
}

export interface BrowserCommandIntent {
  kind: "browser.command_intent";
  type: "content.intent";
  command: string;
  commandId: string;
  expectedRevision: string;
  payload?: unknown;
  context: {
    sessionId: string;
    capabilityId: string;
    artifactId: string;
    artifactRevision: string;
    frameId: string;
    appId?: string;
    participantId?: string;
    chain: Chain;
  };
}

export type ErrKind =
  | "FrameSandboxDenied"
  | "FrameMessageInvalid"
  | "FrameLeaseDenied"
  | "FrameCapabilityDenied"
  | "FrameScopeEscape";

export class FrameError extends Error {
  layer = "FrontendIsolation" as const;

  constructor(
    public kind: ErrKind,
    message: string,
    public details: Record<string, unknown> = {},
  ) {
    super(message);
  }
}

export const sandbox = "allow-scripts";

const raw = new Set([
  "allow",
  "allowresponses",
  "bearer",
  "cred",
  "credential",
  "credentials",
  "deny",
  "headers",
  "jwt",
  "nats",
  "nkey",
  "permission",
  "permissions",
  "publish",
  "reply",
  "replysubject",
  "secret",
  "seed",
  "subject",
  "subjects",
  "subscribe",
  "password",
  "token",
  "tokens",
]);

export function frameAttrs(title = "generated artifact") {
  return {
    title,
    sandbox: checkSandbox(sandbox),
    referrerPolicy: "no-referrer" as const,
  };
}

export function checkSandbox(value: string) {
  const tokens = new Set(value.split(/\s+/).filter(Boolean));
  if (tokens.size !== 1 || !tokens.has("allow-scripts")) {
    throw err("FrameSandboxDenied", "Generated content requires script-only sandbox", {
      value,
    });
  }
  return [...tokens].sort().join(" ");
}

export function makeLease(input: Omit<Lease, "nonce"> & { nonce?: string }): Lease {
  return {
    ...input,
    nonce: input.nonce ?? nonce(),
  };
}

export function mayObserve(lease: Lease, sessionId: string): boolean {
  return lease.sessions.includes(sessionId);
}

export function accept(lease: Lease, source: unknown, expectedSource: unknown, msg: unknown) {
  if (source !== expectedSource) {
    throw err("FrameLeaseDenied", "Message source does not match leased frame", {
      frameId: lease.frameId,
    });
  }
  denyRaw(msg);
  const intent = parse(msg);
  if (intent.nonce !== lease.nonce) {
    throw err("FrameLeaseDenied", "Message nonce does not match frame lease", {
      frameId: lease.frameId,
    });
  }
  if (intent.frameId !== lease.frameId) {
    throw err("FrameLeaseDenied", "Message frame id does not match lease", {
      frameId: lease.frameId,
      actual: intent.frameId,
    });
  }
  if (intent.artifactRevision !== lease.artifactRevision) {
    throw err("FrameLeaseDenied", "Message artifact revision is stale", {
      expected: lease.artifactRevision,
      actual: intent.artifactRevision,
    });
  }
  if (intent.expectedRevision !== lease.artifactRevision) {
    throw err("FrameLeaseDenied", "Message expected revision is stale", {
      expected: lease.artifactRevision,
      actual: intent.expectedRevision,
    });
  }
  if (intent.schemaRevision !== lease.schemaRevision) {
    throw err("FrameLeaseDenied", "Message schema revision is stale", {
      expected: lease.schemaRevision,
      actual: intent.schemaRevision,
    });
  }
  if (!lease.commands.includes(intent.command)) {
    throw err("FrameCapabilityDenied", "Command is not allowed for frame lease", {
      command: intent.command,
    });
  }
  if (intent.sessionId !== undefined && !mayObserve(lease, intent.sessionId)) {
    throw err("FrameScopeEscape", "Session is not in frame lease observation scope", {
      sessionId: intent.sessionId,
    });
  }
  if (lease.appId !== undefined && intent.appId !== lease.appId) {
    throw err("FrameScopeEscape", "App is not in frame lease scope", {
      expected: lease.appId,
      actual: intent.appId,
    });
  }
  if (lease.participantId !== undefined && intent.participantId !== lease.participantId) {
    throw err("FrameScopeEscape", "Participant is not in frame lease scope", {
      expected: lease.participantId,
      actual: intent.participantId,
    });
  }

  const out: BrowserCommandIntent = {
    kind: "browser.command_intent",
    type: "content.intent",
    command: intent.command,
    commandId: intent.commandId,
    expectedRevision: intent.expectedRevision,
    payload: intent.payload,
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
  return out;
}

export function denyRaw(value: unknown, path: string[] = [], seen = new WeakSet<object>()) {
  if (!isObj(value)) return;
  if (seen.has(value)) return;
  seen.add(value);

  if (Array.isArray(value)) {
    value.forEach((item, i) => denyRaw(item, [...path, String(i)], seen));
    return;
  }

  if (value instanceof Map) {
    let i = 0;
    for (const [key, item] of value) {
      const label = typeof key === "string" ? key : String(i);
      checkRawKey(label, [...path, label]);
      if (typeof key !== "string") denyRaw(key, [...path, `$key${i}`], seen);
      denyRaw(item, [...path, label], seen);
      i += 1;
    }
    return;
  }

  if (value instanceof Set) {
    let i = 0;
    for (const item of value) {
      denyRaw(item, [...path, String(i)], seen);
      i += 1;
    }
    return;
  }

  for (const [key, item] of Object.entries(value)) {
    checkRawKey(key, [...path, key]);
    denyRaw(item, [...path, key], seen);
  }
}

function parse(msg: unknown): ContentIntent {
  if (!isRec(msg)) throw err("FrameMessageInvalid", "Message must be an object");
  if (msg.type !== "content.intent") {
    throw err("FrameMessageInvalid", "Message type is not supported", {
      type: msg.type,
    });
  }
  for (const key of [
    "command",
    "commandId",
    "expectedRevision",
    "nonce",
    "frameId",
    "artifactRevision",
    "schemaRevision",
  ]) {
    if (typeof msg[key] !== "string" || msg[key].length === 0) {
      throw err("FrameMessageInvalid", "Message field is required", { field: key });
    }
  }
  return msg as unknown as ContentIntent;
}

function err(kind: ErrKind, message: string, details: Record<string, unknown> = {}) {
  return new FrameError(kind, message, details);
}

function nonce() {
  if (typeof crypto.randomUUID === "function") return crypto.randomUUID();
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = [...bytes].map((b) => b.toString(16).padStart(2, "0")).join("");
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

function checkRawKey(key: string, path: string[]) {
  const name = key.toLowerCase().replace(/[-_]/g, "");
  if (![...raw].some((word) => name.includes(word))) return;
  throw err("FrameCapabilityDenied", "Generated content cannot send raw authority", {
    path: path.join("."),
  });
}

function isObj(value: unknown): value is object {
  return typeof value === "object" && value !== null;
}

function isRec(value: unknown): value is Record<string, unknown> {
  return isObj(value);
}
