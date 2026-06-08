import {
  TinkabotRuntimeError,
  errorMessage,
  type RuntimeErrorKind,
} from "../nats-script-runtime/errors";

export interface FrontendChainContext {
  chainId: string;
  rootId: string;
  parentId?: string;
  hop: number;
  maxHops: number;
}

export interface FrontendMediatorContext {
  sessionId: string;
  capabilityId: string;
  artifactId: string;
  artifactRevision: string;
  frameId: string;
  chain: FrontendChainContext;
}

export interface ContentIntentMessage {
  type: "content.intent";
  command: string;
  commandId?: string;
  expectedRevision: string;
  payload?: unknown;
}

export interface FrontendCommandIntent {
  type: "frontend.command_intent";
  commandId: string;
  command: string;
  expectedRevision: string;
  payload?: unknown;
  observedAt: string;
  context: FrontendMediatorContext;
}

export type MediatorCommandStatusValue =
  | "accepted"
  | "rejected"
  | "applied"
  | "failed";

export interface MediatorCommandStatusMessage {
  type: "mediator.command_status";
  commandId: string;
  status: MediatorCommandStatusValue;
  sequence: number;
  observedAt: string;
}

export interface MediatorStateMessage {
  type: "mediator.state";
  projectionId: string;
  revision: string;
  sequence: number;
  value: unknown;
}

export interface MediatorErrorMessage {
  type: "mediator.error";
  error: {
    kind: RuntimeErrorKind;
    message: string;
    details?: Record<string, unknown>;
  };
  observedAt: string;
}

export type MediatorToContentMessage =
  | MediatorCommandStatusMessage
  | MediatorStateMessage
  | MediatorErrorMessage;

export interface FrontendMediatorOptions {
  context: FrontendMediatorContext;
  allowedCommands: readonly string[];
  publishCommand: (intent: FrontendCommandIntent) => void | Promise<void>;
  createCommandId?: () => string;
  now?: () => string;
}

export interface FrontendMediator {
  handleContentMessage(
    message: unknown,
  ): Promise<MediatorCommandStatusMessage>;
}

export function createFrontendMediator(
  options: FrontendMediatorOptions,
): FrontendMediator {
  const allowedCommands = new Set(options.allowedCommands);
  const now = options.now ?? (() => new Date().toISOString());
  const createCommandId =
    options.createCommandId ?? createLocalCommandIdFactory();
  let sequence = 0;

  return {
    async handleContentMessage(
      message: unknown,
    ): Promise<MediatorCommandStatusMessage> {
      try {
        assertNoRawNatsVocabulary(message);
        const intentMessage = parseContentIntentMessage(message);

        if (!allowedCommands.has(intentMessage.command)) {
          throw frontendError(
            "FrontendCapabilityDenied",
            "handleContentMessage",
            `Frontend command is not allowed: ${intentMessage.command}`,
            { command: intentMessage.command },
          );
        }

        const commandId = intentMessage.commandId ?? createCommandId();
        const observedAt = now();
        const commandIntent: FrontendCommandIntent = {
          type: "frontend.command_intent",
          commandId,
          command: intentMessage.command,
          expectedRevision: intentMessage.expectedRevision,
          payload: intentMessage.payload,
          observedAt,
          context: options.context,
        };

        await options.publishCommand(commandIntent);

        sequence += 1;
        return {
          type: "mediator.command_status",
          commandId,
          status: "accepted",
          sequence,
          observedAt,
        };
      } catch (error) {
        if (error instanceof TinkabotRuntimeError) throw error;
        throw frontendCritical("handleContentMessage", error);
      }
    },
  };
}

export interface MaterializerProjectionEntry {
  revision: string;
  sequence: number;
  value: unknown;
}

export interface MaterializerCommandEntry {
  status: MediatorCommandStatusValue;
  sequence: number;
  observedAt: string;
}

export interface MaterializerSnapshot {
  projections: Record<string, MaterializerProjectionEntry>;
  commands: Record<string, MaterializerCommandEntry>;
  errors: MediatorErrorMessage["error"][];
}

export interface MaterializerStore {
  apply(message: MediatorToContentMessage): void;
  snapshot(): MaterializerSnapshot;
}

export function createMaterializerStore(): MaterializerStore {
  const projections: Record<string, MaterializerProjectionEntry> = {};
  const commands: Record<string, MaterializerCommandEntry> = {};
  const errors: MediatorErrorMessage["error"][] = [];

  return {
    apply(message: MediatorToContentMessage): void {
      if (message.type === "mediator.state") {
        const current = projections[message.projectionId];
        if (current && message.sequence <= current.sequence) return;
        projections[message.projectionId] = {
          revision: message.revision,
          sequence: message.sequence,
          value: message.value,
        };
        return;
      }

      if (message.type === "mediator.command_status") {
        const current = commands[message.commandId];
        if (current && message.sequence <= current.sequence) return;
        commands[message.commandId] = {
          status: message.status,
          sequence: message.sequence,
          observedAt: message.observedAt,
        };
        return;
      }

      errors.push(message.error);
    },

    snapshot(): MaterializerSnapshot {
      return {
        projections: { ...projections },
        commands: { ...commands },
        errors: [...errors],
      };
    },
  };
}

export interface DedicatedWorkerScopeLike {
  addEventListener(
    type: "message",
    listener: (event: { data: unknown }) => void | Promise<void>,
  ): void;
  postMessage(message: MediatorToContentMessage): void;
}

export function bindDedicatedWorkerMediator(
  scope: DedicatedWorkerScopeLike,
  mediator: FrontendMediator,
  options: { now?: () => string } = {},
): void {
  const now = options.now ?? (() => new Date().toISOString());

  scope.addEventListener("message", async (event) => {
    try {
      const response = await mediator.handleContentMessage(event.data);
      scope.postMessage(response);
    } catch (error) {
      scope.postMessage(toMediatorErrorMessage(error, now()));
    }
  });
}

function parseContentIntentMessage(message: unknown): ContentIntentMessage {
  if (!isRecord(message)) {
    throw frontendError(
      "FrontendMessageInvalid",
      "parseContentIntentMessage",
      "Frontend message must be an object",
    );
  }

  if (message.type !== "content.intent") {
    throw frontendError(
      "FrontendMessageInvalid",
      "parseContentIntentMessage",
      "Frontend message type is not supported",
      { type: message.type },
    );
  }

  if (typeof message.command !== "string" || message.command.length === 0) {
    throw frontendError(
      "FrontendMessageInvalid",
      "parseContentIntentMessage",
      "Frontend command is required",
    );
  }

  if (
    typeof message.expectedRevision !== "string" ||
    message.expectedRevision.length === 0
  ) {
    throw frontendError(
      "FrontendMessageInvalid",
      "parseContentIntentMessage",
      "Expected revision is required",
      { command: message.command },
    );
  }

  if (
    message.commandId !== undefined &&
    (typeof message.commandId !== "string" || message.commandId.length === 0)
  ) {
    throw frontendError(
      "FrontendMessageInvalid",
      "parseContentIntentMessage",
      "Command id must be a non-empty string when provided",
    );
  }

  return {
    type: "content.intent",
    command: message.command,
    commandId: message.commandId,
    expectedRevision: message.expectedRevision,
    payload: message.payload,
  };
}

const rawNatsFieldNames = new Set([
  "allow",
  "allowresponses",
  "credential",
  "credentials",
  "deny",
  "headers",
  "nats",
  "permissions",
  "publish",
  "reply",
  "replysubject",
  "subject",
  "subjects",
  "subscribe",
  "token",
  "tokens",
]);

function assertNoRawNatsVocabulary(value: unknown, path: string[] = []): void {
  if (Array.isArray(value)) {
    value.forEach((item, index) =>
      assertNoRawNatsVocabulary(item, [...path, String(index)]),
    );
    return;
  }

  if (!isRecord(value)) return;

  for (const [key, nested] of Object.entries(value)) {
    const normalized = key.toLowerCase().replace(/[_-]/g, "");
    if (rawNatsFieldNames.has(normalized)) {
      throw frontendError(
        "FrontendCapabilityDenied",
        "assertNoRawNatsVocabulary",
        `Generated content cannot provide raw NATS field: ${key}`,
        { path: [...path, key].join(".") },
      );
    }
    assertNoRawNatsVocabulary(nested, [...path, key]);
  }
}

function toMediatorErrorMessage(
  error: unknown,
  observedAt: string,
): MediatorErrorMessage {
  if (error instanceof TinkabotRuntimeError) {
    return {
      type: "mediator.error",
      error: {
        kind: error.kind,
        message: error.message,
        details: error.origin.details,
      },
      observedAt,
    };
  }

  return {
    type: "mediator.error",
    error: {
      kind: "FrontendCritical",
      message: errorMessage(error),
    },
    observedAt,
  };
}

function frontendError(
  kind:
    | "FrontendMessageInvalid"
    | "FrontendCapabilityDenied"
    | "FrontendBridgeUnavailable",
  operation: string,
  message: string,
  details?: Record<string, unknown>,
): TinkabotRuntimeError {
  return new TinkabotRuntimeError(kind, message, {
    origin: {
      layer: "FrontendMediator",
      operation,
      details,
    },
  });
}

function frontendCritical(
  operation: string,
  cause: unknown,
): TinkabotRuntimeError {
  return new TinkabotRuntimeError(
    "FrontendCritical",
    "Frontend mediation failed with an unknown error",
    {
      origin: {
        layer: "FrontendMediator",
        operation,
      },
      cause,
    },
  );
}

function createLocalCommandIdFactory(): () => string {
  let counter = 0;
  return () => {
    counter += 1;
    return `frontend-command-${Date.now()}-${counter}`;
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
