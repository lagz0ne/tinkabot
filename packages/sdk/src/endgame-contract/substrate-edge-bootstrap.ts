import {
  TinkabotRuntimeError,
  type RuntimeErrorOrigin,
} from "../nats-script-runtime/errors";
import { compileAuth } from "./managed-auth-subjects";
import { parseContract, type Contract } from "./index";

type AuthPolicy = Extract<Contract, { kind: "auth.policy" }>;
type ArtifactManifest = Extract<Contract, { kind: "artifact.manifest" }>;
type BrowserCommandIntent = Extract<Contract, { kind: "browser.command_intent" }>;
type CommandAcceptanceStatus = Extract<Contract, { kind: "command.acceptance" }>;

export interface BrowserEdgeChainContext {
  chainId: string;
  rootId: string;
  parentId?: string;
  hop: number;
  maxHops: number;
}

export interface BrowserEdgeBootstrapOptions {
  authPolicy: unknown;
  artifactManifest: unknown;
  credentialRef: string;
  frameId: string;
  chain: BrowserEdgeChainContext;
  objectNamespace?: string;
  expectedDigest?: string;
}

export interface BrowserEdgeWorkerBootstrap {
  principalId: string;
  sessionId: string;
  capabilityId: string;
  leaseId: string;
  credentialDescriptor: {
    kind: "browser.worker_nats";
    ref: string;
    schemaVersion: "v1";
    appRevision: string;
    artifactRevision: string;
    publishAllow: string[];
    publishDeny: string[];
    subscribeAllow: string[];
    subscribeDeny: string[];
  };
}

export interface BrowserEdgeContentBootstrap {
  sessionId: string;
  capabilityId: string;
  artifactId: string;
  artifactRevision: string;
  frameId: string;
  artifact: {
    digest: string;
    mediaType: string;
    objectRef: string;
    cspPolicy: string;
    framePolicy: string;
    sandboxPolicy: string;
  };
  chain: BrowserEdgeChainContext;
}

export interface BrowserEdgeBootstrap {
  worker: BrowserEdgeWorkerBootstrap;
  content: BrowserEdgeContentBootstrap;
}

export interface BrowserEdgeContentIntent {
  type: "content.intent";
  command: string;
  commandId: string;
  expectedRevision: string;
  payload?: unknown;
}

export interface BrowserEdgeContentStatus {
  type: "browser_edge.command_status";
  commandId: string;
  status: CommandAcceptanceStatus["status"];
  sequence: number;
  observedAt: string;
  error?: {
    kind: string;
    message: string;
    layer: string;
  };
}

export function createBrowserEdgeBootstrap(
  opts: BrowserEdgeBootstrapOptions,
): BrowserEdgeBootstrap {
  try {
    const policy = parseAuth(opts.authPolicy);
    const artifact = parseArtifact(opts.artifactManifest);
    const auth = compileAuth(policy);
    const gateway = gatewayPolicy(artifact, opts);
    const cap = policy.capability;
    const credentialRef = requireText(
      opts.credentialRef,
      "credentialRef",
      "createBrowserEdgeBootstrap",
    );
    const frameId = requireText(opts.frameId, "frameId", "createBrowserEdgeBootstrap");
    const content = {
      sessionId: cap.sessionId,
      capabilityId: cap.capabilityId,
      artifactId: artifact.artifactId,
      artifactRevision: artifact.artifactRevision,
      frameId,
      artifact: gateway,
      chain: opts.chain,
    };

    assertNoRawAuthority(content, "createBrowserEdgeBootstrap");

    return {
      worker: {
        principalId: cap.principalId,
        sessionId: cap.sessionId,
        capabilityId: cap.capabilityId,
        leaseId: cap.leaseId,
        credentialDescriptor: {
          kind: "browser.worker_nats",
          ref: credentialRef,
          schemaVersion: cap.schemaVersion,
          appRevision: cap.appRevision,
          artifactRevision: artifact.artifactRevision,
          publishAllow: auth.permissions.publish?.allow ?? [],
          publishDeny: auth.permissions.publish?.deny ?? [],
          subscribeAllow: auth.permissions.subscribe?.allow ?? [],
          subscribeDeny: auth.permissions.subscribe?.deny ?? [],
        },
      },
      content,
    };
  } catch (error) {
    if (error instanceof TinkabotRuntimeError) throw error;
    throw edgeCritical("createBrowserEdgeBootstrap", error);
  }
}

export function createBrowserCommandIntent(
  ctx: BrowserEdgeContentBootstrap,
  message: unknown,
): BrowserCommandIntent {
  try {
    assertNoRawAuthority(message, "createBrowserCommandIntent");
    const parsed = parseContentIntent(message);
    return parseBrowserCommand({
      kind: "browser.command_intent",
      type: "content.intent",
      command: parsed.command,
      commandId: parsed.commandId,
      expectedRevision: parsed.expectedRevision,
      payload: parsed.payload,
      context: {
        sessionId: ctx.sessionId,
        capabilityId: ctx.capabilityId,
        artifactId: ctx.artifactId,
        artifactRevision: ctx.artifactRevision,
        frameId: ctx.frameId,
        chain: ctx.chain,
      },
    });
  } catch (error) {
    if (error instanceof TinkabotRuntimeError) throw error;
    throw edgeCritical("createBrowserCommandIntent", error);
  }
}

export function toBrowserEdgeContentStatus(
  input: unknown,
): BrowserEdgeContentStatus {
  const status = parseAcceptance(input);
  const message: BrowserEdgeContentStatus = {
    type: "browser_edge.command_status",
    commandId: status.commandId,
    status: status.status,
    sequence: status.sequence,
    observedAt: status.observedAt,
  };

  if (status.error) {
    message.error = {
      kind: status.error.kind,
      message: status.error.message,
      layer: status.error.origin.layer,
    };
  }

  assertNoRawAuthority(message, "toBrowserEdgeContentStatus");
  return message;
}

function gatewayPolicy(
  artifact: ArtifactManifest,
  opts: BrowserEdgeBootstrapOptions,
): BrowserEdgeContentBootstrap["artifact"] {
  const expected = opts.expectedDigest;
  if (expected && artifact.digest !== expected) {
    throw gatewayError(
      "ArtifactDigestMismatch",
      "gatewayPolicy",
      "Artifact digest does not match expected digest",
      { field: "digest", expected, actual: artifact.digest },
    );
  }

  const namespace = opts.objectNamespace ?? "obj://frontend/";
  if (!artifact.objectRef.startsWith(namespace)) {
    throw gatewayError(
      "ArtifactGatewayPolicyInvalid",
      "gatewayPolicy",
      "Artifact object ref is outside the allowed namespace",
      { field: "objectRef", namespace, objectRef: artifact.objectRef },
    );
  }
  if (!artifact.cspPolicy) {
    throw gatewayError(
      "ArtifactGatewayPolicyInvalid",
      "gatewayPolicy",
      "Artifact CSP policy is required",
      { field: "cspPolicy" },
    );
  }
  if (!artifact.framePolicy) {
    throw gatewayError(
      "ArtifactGatewayPolicyInvalid",
      "gatewayPolicy",
      "Artifact frame policy is required",
      { field: "framePolicy" },
    );
  }
  if (!artifact.sandboxPolicy.startsWith("sandbox.browser.")) {
    throw gatewayError(
      "ArtifactGatewayPolicyInvalid",
      "gatewayPolicy",
      "Artifact sandbox policy must be browser scoped",
      { field: "sandboxPolicy", sandboxPolicy: artifact.sandboxPolicy },
    );
  }

  return {
    digest: artifact.digest,
    mediaType: artifact.mediaType,
    objectRef: artifact.objectRef,
    cspPolicy: artifact.cspPolicy,
    framePolicy: artifact.framePolicy,
    sandboxPolicy: artifact.sandboxPolicy,
  };
}

function parseContentIntent(input: unknown): BrowserEdgeContentIntent {
  if (!isRecord(input)) {
    throw edgeError(
      "BrowserEdgeInvalid",
      "parseContentIntent",
      "Browser Edge content intent must be an object",
    );
  }
  if (input.type !== "content.intent") {
    throw edgeError(
      "BrowserEdgeInvalid",
      "parseContentIntent",
      "Browser Edge only accepts generated content intents",
      { type: input.type },
    );
  }

  const command = input.command;
  const commandId = input.commandId;
  const expectedRevision = input.expectedRevision;
  if (typeof command !== "string" || command.length === 0) {
    throw edgeError(
      "BrowserEdgeInvalid",
      "parseContentIntent",
      "Browser Edge command is required",
    );
  }
  if (typeof commandId !== "string" || commandId.length === 0) {
    throw edgeError(
      "BrowserEdgeInvalid",
      "parseContentIntent",
      "Browser Edge command id is required",
    );
  }
  if (typeof expectedRevision !== "string" || expectedRevision.length === 0) {
    throw edgeError(
      "BrowserEdgeInvalid",
      "parseContentIntent",
      "Browser Edge expected revision is required",
      { command },
    );
  }

  return {
    type: "content.intent",
    command,
    commandId,
    expectedRevision,
    payload: input.payload,
  };
}

function requireText(value: unknown, field: string, operation: string): string {
  if (typeof value === "string" && value.length > 0) return value;
  throw edgeError(
    "BrowserEdgeInvalid",
    operation,
    `Browser Edge ${field} is required`,
    { field },
  );
}

function parseAuth(input: unknown): AuthPolicy {
  const parsed = parseContract(input);
  if (parsed.kind !== "auth.policy") {
    throw edgeError(
      "BrowserEdgeInvalid",
      "parseAuth",
      "Expected auth.policy for Browser Edge bootstrap",
    );
  }
  return parsed;
}

function parseArtifact(input: unknown): ArtifactManifest {
  const parsed = parseContract(input);
  if (parsed.kind !== "artifact.manifest") {
    throw edgeError(
      "BrowserEdgeInvalid",
      "parseArtifact",
      "Expected artifact.manifest for Browser Edge bootstrap",
    );
  }
  return parsed;
}

function parseBrowserCommand(input: unknown): BrowserCommandIntent {
  const parsed = parseContract(input);
  if (parsed.kind !== "browser.command_intent") {
    throw edgeError(
      "BrowserEdgeInvalid",
      "parseBrowserCommand",
      "Expected browser.command_intent",
    );
  }
  return parsed;
}

function parseAcceptance(input: unknown): CommandAcceptanceStatus {
  const parsed = parseContract(input);
  if (parsed.kind !== "command.acceptance") {
    throw edgeError(
      "BrowserEdgeInvalid",
      "parseAcceptance",
      "Expected command.acceptance",
    );
  }
  return parsed;
}

const raw = [
  "nats",
  "subject",
  "token",
  "credential",
  "permission",
  "publish",
  "subscribe",
];

function assertNoRawAuthority(
  value: unknown,
  operation: string,
  path: string[] = [],
): void {
  if (!value || typeof value !== "object") return;

  for (const [key, item] of Object.entries(value)) {
    const normalized = key.toLowerCase().replace(/[-_]/g, "");
    if (raw.some((word) => normalized.includes(word))) {
      throw edgeError(
        "BrowserEdgeCapabilityDenied",
        operation,
        `Generated content cannot receive raw authority field: ${key}`,
        { path: [...path, key].join(".") },
      );
    }
    assertNoRawAuthority(item, operation, [...path, key]);
  }
}

function edgeError(
  kind: "BrowserEdgeInvalid" | "BrowserEdgeCapabilityDenied",
  operation: string,
  message: string,
  details?: Record<string, unknown>,
): TinkabotRuntimeError {
  return new TinkabotRuntimeError(kind, message, {
    origin: origin("BrowserEdge", operation, details),
  });
}

function gatewayError(
  kind: "ArtifactDigestMismatch" | "ArtifactGatewayPolicyInvalid",
  operation: string,
  message: string,
  details?: Record<string, unknown>,
): TinkabotRuntimeError {
  return new TinkabotRuntimeError(kind, message, {
    origin: origin("ArtifactGateway", operation, details),
  });
}

function edgeCritical(operation: string, cause: unknown): TinkabotRuntimeError {
  return new TinkabotRuntimeError(
    "BrowserEdgeCritical",
    "Browser Edge failed with an unknown error",
    {
      origin: origin("BrowserEdge", operation),
      cause,
    },
  );
}

function origin(
  layer: "BrowserEdge" | "ArtifactGateway",
  operation: string,
  details?: Record<string, unknown>,
): RuntimeErrorOrigin {
  return details ? { layer, operation, details } : { layer, operation };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
