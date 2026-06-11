import { z } from "zod";
import {
  TinkabotRuntimeError,
  type RuntimeErrorOrigin,
} from "../nats-script-runtime/errors";

export const contractSchemaId = "tb.schema.base.contract_authority.v1";

const rawKeys = new Set([
  "subject",
  "reply",
  "token",
  "cred",
  "credential",
  "credentials",
  "permission",
  "permissions",
  "publish",
  "subscribe",
  "nats",
  "nkey",
  "jwt",
  "seed",
  "secret",
  "password",
  "bearer",
]);

const subject = z
  .string()
  .regex(
    /^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+(?:\.(?:[A-Za-z0-9_-]+|\*))*?(?:\.>)?$/,
    "invalid authoritative NATS subject pattern",
  );

const text = z.string().min(1);
const iso = text;

const chain = z.strictObject({
  chainId: text,
  rootId: text,
  parentId: text.optional(),
  hop: z.number().int().min(0),
  maxHops: z.number().int().min(1),
});

const provenance = z.strictObject({
  schemaId: z.literal(contractSchemaId),
  schemaVersion: z.literal("v1"),
  appRevision: text,
  createdAt: iso,
  producer: text,
  digest: text.optional(),
});

const capability = z.strictObject({
  principalId: text,
  sessionId: text,
  capabilityId: text,
  leaseId: text,
  leaseStatus: z.enum(["active", "revoked", "expired"]),
  appRevision: text,
  schemaVersion: z.literal("v1"),
});

const sourceKind = z.enum([
  "request_reply",
  "command_acceptance",
  "subject",
  "kv",
  "object",
  "stream",
  "schedule",
]);

const sourcePrincipal = z.strictObject({
  principalId: text,
  sourceId: text,
  sourceKind,
  authorityRef: text,
});

const sourceLease = z.strictObject({
  leaseId: text,
  leaseStatus: z.enum(["active", "revoked", "expired"]),
  appRevision: text,
  schemaVersion: z.literal("v1"),
  scriptRevision: z.number().int().min(0).optional(),
});

const safeValue = z.unknown().superRefine((value, ctx) => {
  scanRaw(value, ctx, []);
});

const err = z.strictObject({
  kind: text,
  message: text,
  origin: z.strictObject({
    layer: text,
    operation: text,
  }),
});

const permList = z.strictObject({
  allow: z.array(subject).optional(),
  deny: z.array(subject).optional(),
});

const authPolicy = z.strictObject({
  kind: z.literal("auth.policy"),
  provenance,
  capability,
  permissions: z.strictObject({
    publish: permList.optional(),
    subscribe: permList.optional(),
    allow_responses: z
      .strictObject({
        max: z.number().int().min(1),
        expiresMs: z.number().int().min(1).optional(),
      })
      .optional(),
  }),
  imports: z
    .record(
      text,
      z.strictObject({
        kind: z.enum(["publish", "subscribe", "raw_nats", "cli"]),
        subjects: z.array(subject).optional(),
        desc: text.optional(),
      }),
    )
    .optional(),
  exports: z.array(subject).optional(),
  exposure: z
    .record(
      text,
      z.strictObject({
        kind: z.enum(["request_reply", "subject", "kv_watch", "object_change", "stream"]),
        subject: subject.optional(),
        desc: text.optional(),
      }),
    )
    .optional(),
});

const browserCtx = z.strictObject({
  sessionId: text,
  capabilityId: text,
  artifactId: text,
  artifactRevision: text,
  frameId: text,
  chain,
});

const browserCommand = z.strictObject({
  kind: z.literal("browser.command_intent"),
  type: z.literal("content.intent"),
  command: text,
  commandId: text,
  expectedRevision: text,
  payload: safeValue.optional(),
  context: browserCtx,
});

const acceptance = z.strictObject({
  kind: z.literal("command.acceptance"),
  type: z.literal("command.acceptance"),
  commandId: text,
  status: z.enum(["accepted", "rejected", "applied", "failed"]),
  sequence: z.number().int().min(0),
  observedAt: iso,
  provenance,
  capability,
  chain,
  error: err.optional(),
});

const requestReplySource = z.strictObject({
  kind: z.literal("request_reply"),
  activationName: text,
  subject,
  requestId: text,
});

const commandAcceptanceSource = z.strictObject({
  kind: z.literal("command_acceptance"),
  activationName: text,
  subject,
  commandId: text,
  command: text,
  artifactId: text,
  artifactRevision: text,
  frameId: text,
});

const subjectSource = z.strictObject({
  kind: z.literal("subject"),
  activationName: text,
  pattern: subject,
  observedSubject: subject,
  messageId: text,
});

const kvSource = z.strictObject({
  kind: z.literal("kv"),
  activationName: text,
  bucket: text,
  key: text,
  operation: z.enum(["put", "delete", "purge"]),
  revision: z.number().int().min(0),
  watchRevision: z.number().int().min(0),
  resume: text,
});

const objectSource = z.strictObject({
  kind: z.literal("object"),
  activationName: text,
  bucket: text,
  name: text,
  digest: text,
  revision: z.number().int().min(0).optional(),
  objectMetaSequence: z.number().int().min(0),
  watchPosition: text,
});

const streamSource = z.strictObject({
  kind: z.literal("stream"),
  activationName: text,
  stream: text,
  consumer: text,
  streamSequence: z.number().int().min(0),
  consumerSequence: z.number().int().min(0),
  subject,
  deliveryAttempt: z.number().int().min(1),
});

const scheduleSource = z.strictObject({
  kind: z.literal("schedule"),
  activationName: text,
  scheduleId: text,
  tickId: text,
  dueAt: iso,
  ownerPrincipalId: text,
  leaderEpoch: z.number().int().min(0),
  fencingToken: text,
  acquiredAt: iso,
  expiresAt: iso,
  clockId: text,
  clock: text,
});

const processResource = z.strictObject({
  cpuMillis: z.number().int().min(1),
  memoryMB: z.number().int().min(1),
});

const scriptEnv = z.record(text, text).superRefine((env, ctx) => {
  for (const key of Object.keys(env)) {
    if (hasRawKey(key)) {
      ctx.addIssue({
        code: "custom",
        path: [key],
        message: `raw NATS vocabulary is not allowed: ${key}`,
      });
    }
  }
});

const scriptProcess = z.strictObject({
  command: text,
  args: z.array(text),
  cwd: text,
  env: scriptEnv.optional(),
  rpc: z.literal("framed_stdio"),
  timeoutMs: z.number().int().min(1),
  resource: processResource,
  kill: text,
  cleanup: text,
  identity: text,
});

const scriptRecord = z.strictObject({
  kind: z.literal("script.record"),
  scriptKey: text,
  scriptRevision: z.number().int().min(0),
  desc: text.optional(),
  process: scriptProcess,
});

const projectionEffect = z.strictObject({
  kind: z.literal("script.effect"),
  effectType: z.literal("projection"),
  projectionId: text,
  snapshotRevision: text,
  artifactRevision: text,
  sequence: z.number().int().min(0),
  value: safeValue,
});

const artifactEffect = z.strictObject({
  kind: z.literal("script.effect"),
  effectType: z.literal("artifact"),
  artifactName: text,
  artifactRevision: text,
  mediaType: text,
  body: text,
});

const publishEffect = z.strictObject({
  kind: z.literal("script.effect"),
  effectType: z.literal("publish"),
  subject,
  body: safeValue.optional(),
});

const scriptEffect = z.discriminatedUnion("effectType", [
  projectionEffect,
  artifactEffect,
  publishEffect,
]);

const activation = z.strictObject({
  kind: z.literal("activation.intent"),
  activationId: text,
  triggerId: text,
  scriptKey: text,
  scriptRevision: z.number().int().min(0).optional(),
  sourcePrincipal,
  sourceLease,
  source: z.discriminatedUnion("kind", [
    requestReplySource,
    commandAcceptanceSource,
    subjectSource,
    kvSource,
    objectSource,
    streamSource,
    scheduleSource,
  ]),
  payload: safeValue.optional(),
  headers: z.record(text, text),
  observedAt: iso,
  reply: z.strictObject({ subject }).optional(),
  chain,
  dedupeKey: text,
  provenance,
  capability,
});

const artifact = z.strictObject({
  kind: z.literal("artifact.manifest"),
  artifactId: text,
  artifactRevision: text,
  digest: text,
  mediaType: text,
  objectRef: text,
  sandboxPolicy: text,
  framePolicy: text.optional(),
  cspPolicy: text.optional(),
  createdAt: iso,
  provenance,
});

const projection = z.strictObject({
  kind: z.literal("material.projection"),
  projectionId: text,
  snapshotRevision: text,
  artifactRevision: text,
  sequence: z.number().int().min(0),
  value: safeValue,
  observedAt: iso,
  provenance,
});

const event = z.strictObject({
  kind: z.literal("event.envelope"),
  eventId: text,
  eventType: text,
  status: z.enum(["success", "failed", "denied", "stale", "revoked"]),
  principalId: text,
  capabilityId: text,
  chain,
  observedAt: iso,
  provenance,
  subject: subject.optional(),
  store: text.optional(),
  revisions: z.record(text, text),
  error: err.optional(),
});

const trustTier = z.strictObject({
  tier: z.enum(["untrusted", "trusted"]),
  ownerLayer: z.literal("trusted-wrapper-authority"),
});

const sessionState = z.enum(["starting", "running", "stopping", "stopped", "failed"]);

const sessionRecord = z.strictObject({
  kind: z.literal("session.record"),
  sessionId: text,
  runnerId: text,
  state: sessionState,
  trust: trustTier,
  provenance,
});

const tokenFrame = z.strictObject({
  kind: z.literal("session.frame"),
  frame: z.literal("token"),
  origin: z.literal("wrapper"),
  sessionId: text,
  text,
});

const chunkFrame = z.strictObject({
  kind: z.literal("session.frame"),
  frame: z.literal("chunk"),
  origin: z.literal("wrapper"),
  sessionId: text,
  body: safeValue,
});

const statusFrame = z.strictObject({
  kind: z.literal("session.frame"),
  frame: z.literal("status"),
  origin: z.literal("runner"),
  sessionId: text,
  state: sessionState,
  detail: text.optional(),
});

const sessionFrame = z.discriminatedUnion("frame", [tokenFrame, chunkFrame, statusFrame]);

const steerIntent = z.discriminatedUnion("intent", [
  z.strictObject({
    kind: z.literal("session.steer_intent"),
    intent: z.literal("steer"),
    sessionId: text,
    text,
  }),
  z.strictObject({
    kind: z.literal("session.steer_intent"),
    intent: z.literal("stop"),
    sessionId: text,
  }),
]);

export const Contract = z.union([
  authPolicy,
  browserCommand,
  acceptance,
  activation,
  scriptRecord,
  scriptEffect,
  artifact,
  projection,
  event,
  sessionRecord,
  sessionFrame,
  steerIntent,
]);

export type Contract = z.infer<typeof Contract>;

export function parseContract(input: unknown): Contract {
  try {
    return Contract.parse(input);
  } catch (error) {
    if (error instanceof TinkabotRuntimeError) throw error;
    if (error instanceof z.ZodError) {
      throw contractError("parseContract", "Contract input is invalid", {
        issues: error.issues.map((issue) => ({
          path: issue.path.join("."),
          message: issue.message,
        })),
      });
    }
    throw contractCritical("parseContract", error);
  }
}

function scanRaw(value: unknown, ctx: z.RefinementCtx, path: PropertyKey[]): void {
  if (!value || typeof value !== "object") return;

  if (Array.isArray(value)) {
    value.forEach((item, index) => scanRaw(item, ctx, [...path, index]));
    return;
  }

  for (const [key, item] of Object.entries(value)) {
    const next = [...path, key];
    if (hasRawKey(key)) {
      ctx.addIssue({
        code: "custom",
        path: next,
        message: `raw NATS vocabulary is not allowed: ${key}`,
      });
    }
    scanRaw(item, ctx, next);
  }
}

function hasRawKey(key: string): boolean {
  const normalized = key.replace(/[-_]/g, "").toLowerCase();
  return [...rawKeys].some((raw) => normalized.includes(raw));
}

function contractError(
  operation: string,
  message: string,
  details?: Record<string, unknown>,
): TinkabotRuntimeError {
  return new TinkabotRuntimeError("ContractInvalid", message, {
    origin: contractOrigin(operation, details),
  });
}

export * from "./managed-auth-subjects";
export * from "./command-acceptance";
export * from "./substrate-edge-bootstrap";

function contractCritical(
  operation: string,
  cause: unknown,
): TinkabotRuntimeError {
  return new TinkabotRuntimeError(
    "ContractCritical",
    "Contract validation failed with an unknown error",
    {
      origin: contractOrigin(operation),
      cause,
    },
  );
}

function contractOrigin(
  operation: string,
  details?: Record<string, unknown>,
): RuntimeErrorOrigin {
  return {
    layer: "ContractAuthority",
    operation,
    details,
  };
}
