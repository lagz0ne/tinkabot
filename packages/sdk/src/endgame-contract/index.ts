import { z } from "zod";
import {
  TinkabotRuntimeError,
  type RuntimeErrorOrigin,
} from "../nats-script-runtime/errors";

export const contractSchemaId = "tb.schema.endgame.contract_authority.v1";

const rawKeys = new Set([
  "subject",
  "reply",
  "token",
  "credential",
  "credentials",
  "permission",
  "permissions",
  "publish",
  "subscribe",
  "nats",
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

const activation = z.strictObject({
  kind: z.literal("activation.intent"),
  activationId: text,
  triggerId: text,
  scriptKey: text,
  scriptRevision: z.number().int().min(0).optional(),
  source: z.discriminatedUnion("kind", [
    requestReplySource,
    commandAcceptanceSource,
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

export const Contract = z.discriminatedUnion("kind", [
  authPolicy,
  browserCommand,
  acceptance,
  activation,
  artifact,
  projection,
  event,
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
