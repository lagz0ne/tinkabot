import {
  TinkabotRuntimeError,
  type RuntimeErrorOrigin,
} from "./errors";
import { hasPlaceholderSubject, isValidSubjectPattern } from "./subjects";

export interface ActivationIntent {
  activationId: string;
  triggerId: string;
  scriptKey: string;
  scriptRevision?: number;
  source: RequestReplyActivationSource;
  payload: unknown;
  headers: Record<string, string>;
  observedAt: string;
  reply?: ActivationReplyContext;
  chain: ActivationChainContext;
  dedupeKey: string;
}

export interface RequestReplyActivationSource {
  kind: "request_reply";
  activationName: string;
  subject: string;
  requestId: string;
}

export interface ActivationReplyContext {
  subject: string;
}

export interface ActivationChainContext {
  chainId: string;
  rootId: string;
  parentId?: string;
  hop: number;
  maxHops: number;
}

export interface CreateRequestReplyActivationIntentInput {
  activationName: string;
  scriptKey: string;
  scriptRevision?: number;
  subject: string;
  requestId: string;
  payload?: unknown;
  headers?: Record<string, string>;
  replySubject?: string;
  observedAt?: string;
  chain?: Partial<ActivationChainContext>;
  dedupeKey?: string;
}

export function createRequestReplyActivationIntent(
  input: CreateRequestReplyActivationIntentInput,
): ActivationIntent {
  try {
    validateText(input.activationName, "activationName");
    validateText(input.scriptKey, "scriptKey");
    validateText(input.requestId, "requestId");
    validateConcreteSubject(input.subject, "subject");

    if (input.replySubject) {
      validateConcreteSubject(input.replySubject, "replySubject");
    }

    const chain = normalizeChain(input);
    if (chain.hop > chain.maxHops) {
      throw activationError(
        "ActivationLoopSuppressed",
        "createRequestReplyActivationIntent",
        "Activation hop limit exceeded",
        { hop: chain.hop, maxHops: chain.maxHops },
      );
    }

    const dedupeKey =
      input.dedupeKey ??
      [
        "request_reply",
        input.scriptKey,
        input.activationName,
        input.subject,
        input.requestId,
      ].join(":");
    validateText(dedupeKey, "dedupeKey");

    const activationId = [
      "act",
      input.scriptKey,
      input.activationName,
      input.requestId,
    ].join(":");

    return {
      activationId,
      triggerId: input.requestId,
      scriptKey: input.scriptKey,
      scriptRevision: input.scriptRevision,
      source: {
        kind: "request_reply",
        activationName: input.activationName,
        subject: input.subject,
        requestId: input.requestId,
      },
      payload: input.payload,
      headers: input.headers ?? {},
      observedAt: input.observedAt ?? new Date().toISOString(),
      reply: input.replySubject ? { subject: input.replySubject } : undefined,
      chain,
      dedupeKey,
    };
  } catch (error) {
    if (error instanceof TinkabotRuntimeError) throw error;
    throw activationCritical("createRequestReplyActivationIntent", error);
  }
}

function normalizeChain(
  input: CreateRequestReplyActivationIntentInput,
): ActivationChainContext {
  const chainId =
    input.chain?.chainId ??
    ["chain", input.scriptKey, input.activationName, input.requestId].join(":");
  const rootId = input.chain?.rootId ?? chainId;
  const hop = input.chain?.hop ?? 0;
  const maxHops = input.chain?.maxHops ?? 8;

  validateText(chainId, "chainId");
  validateText(rootId, "rootId");
  if (input.chain?.parentId !== undefined) {
    validateText(input.chain.parentId, "parentId");
  }
  if (!Number.isInteger(hop) || hop < 0) {
    throw activationError(
      "ActivationConfigInvalid",
      "createRequestReplyActivationIntent",
      "Activation hop must be a non-negative integer",
      { hop },
    );
  }
  if (!Number.isInteger(maxHops) || maxHops < 1) {
    throw activationError(
      "ActivationConfigInvalid",
      "createRequestReplyActivationIntent",
      "Activation maxHops must be a positive integer",
      { maxHops },
    );
  }

  return {
    chainId,
    rootId,
    parentId: input.chain?.parentId,
    hop,
    maxHops,
  };
}

function validateText(value: string | undefined, field: string): void {
  if (!value || typeof value !== "string") {
    throw activationError(
      "ActivationConfigInvalid",
      "createRequestReplyActivationIntent",
      `Activation ${field} is required`,
      { field },
    );
  }
}

function validateConcreteSubject(subject: string, field: string): void {
  validateText(subject, field);
  const wildcarded = subject.split(".").some((token) => token === "*" || token === ">");
  if (
    hasPlaceholderSubject(subject) ||
    !isValidSubjectPattern(subject) ||
    wildcarded
  ) {
    throw activationError(
      "ActivationConfigInvalid",
      "createRequestReplyActivationIntent",
      `Activation ${field} must be a concrete NATS subject`,
      { field, subject },
    );
  }
}

function activationError(
  kind:
    | "ActivationConfigInvalid"
    | "ActivationLoopSuppressed",
  operation: string,
  message: string,
  details?: Record<string, unknown>,
): TinkabotRuntimeError {
  return new TinkabotRuntimeError(kind, message, {
    origin: activationOrigin(operation, details),
  });
}

function activationCritical(
  operation: string,
  cause: unknown,
): TinkabotRuntimeError {
  return new TinkabotRuntimeError(
    "ActivationCritical",
    "Activation failed with an unknown error",
    {
      origin: activationOrigin(operation),
      cause,
    },
  );
}

function activationOrigin(
  operation: string,
  details?: Record<string, unknown>,
): RuntimeErrorOrigin {
  return {
    layer: "Activation",
    operation,
    details,
  };
}
