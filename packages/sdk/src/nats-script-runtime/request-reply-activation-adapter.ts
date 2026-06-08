import {
  createRequestReplyActivationIntent,
  type ActivationChainContext,
  type ActivationIntent,
} from "./activation-intent";
import {
  TinkabotRuntimeError,
  type RuntimeErrorKind,
  type RuntimeErrorOrigin,
} from "./errors";
import { type ScriptMetadata } from "./metadata-validator";
import { type PermissionResolver } from "./permission-resolver";

export interface RequestReplyActivationEnvelope {
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

export interface ActivateRequestReplyOptions {
  metadata: ScriptMetadata;
  resolver: Pick<PermissionResolver, "assertActivationSource">;
  envelope: RequestReplyActivationEnvelope;
}

export function activateRequestReply(
  options: ActivateRequestReplyOptions,
): ActivationIntent {
  try {
    validateEnvelope(options.envelope);
    const activation =
      options.metadata.nats.activations?.[options.envelope.activationName];

    if (!activation) {
      throw activationError(
        "ActivationUnauthorized",
        "activateRequestReply",
        "Activation source is not declared",
        { activationName: options.envelope.activationName },
      );
    }
    if (activation.kind !== "request_reply") {
      throw activationError(
        "ActivationConfigInvalid",
        "activateRequestReply",
        "Activation source is not request/reply",
        { activationName: options.envelope.activationName },
      );
    }

    try {
      options.resolver.assertActivationSource(
        options.envelope.activationName,
        options.envelope.subject,
      );
    } catch (error) {
      throw mapAuthorityError(error);
    }

    return createRequestReplyActivationIntent({
      activationName: options.envelope.activationName,
      scriptKey: options.envelope.scriptKey,
      scriptRevision: options.envelope.scriptRevision,
      subject: options.envelope.subject,
      requestId: options.envelope.requestId,
      payload: options.envelope.payload,
      headers: options.envelope.headers,
      replySubject: options.envelope.replySubject,
      observedAt: options.envelope.observedAt,
      chain: options.envelope.chain,
      dedupeKey: options.envelope.dedupeKey,
    });
  } catch (error) {
    if (error instanceof TinkabotRuntimeError) throw error;
    throw activationCritical("activateRequestReply", error);
  }
}

function validateEnvelope(envelope: RequestReplyActivationEnvelope): void {
  if (!envelope || typeof envelope !== "object") {
    throw activationError(
      "ActivationConfigInvalid",
      "activateRequestReply",
      "Request/reply activation envelope is required",
    );
  }
  for (const field of ["activationName", "scriptKey", "subject", "requestId"]) {
    const value = envelope[field as keyof RequestReplyActivationEnvelope];
    if (!value || typeof value !== "string") {
      throw activationError(
        "ActivationConfigInvalid",
        "activateRequestReply",
        `Request/reply activation ${field} is required`,
        { field },
      );
    }
  }
}

function mapAuthorityError(error: unknown): TinkabotRuntimeError {
  if (
    error instanceof TinkabotRuntimeError &&
    isAuthorizationDenial(error.kind)
  ) {
    return activationError(
      "ActivationUnauthorized",
      "activateRequestReply",
      "Request/reply activation source is unauthorized",
      {
        lowerKind: error.kind,
        lowerLayer: error.origin.layer,
        lowerOperation: error.origin.operation,
      },
      error,
    );
  }
  throw error;
}

function isAuthorizationDenial(kind: RuntimeErrorKind): boolean {
  return kind === "PermissionDenied" || kind === "PermissionDeniedByDenyRule";
}

function activationError(
  kind:
    | "ActivationConfigInvalid"
    | "ActivationUnauthorized",
  operation: string,
  message: string,
  details?: Record<string, unknown>,
  cause?: unknown,
): TinkabotRuntimeError {
  return new TinkabotRuntimeError(kind, message, {
    origin: activationOrigin(operation, details),
    cause,
  });
}

function activationCritical(
  operation: string,
  cause: unknown,
): TinkabotRuntimeError {
  return new TinkabotRuntimeError(
    "ActivationCritical",
    "Request/reply activation failed with an unknown error",
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
