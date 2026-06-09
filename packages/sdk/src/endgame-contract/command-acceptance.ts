import {
  TinkabotRuntimeError,
  type RuntimeErrorKind,
  type RuntimeErrorLayer,
  type RuntimeErrorOrigin,
} from "../nats-script-runtime/errors";
import { parseContract, type Contract } from "./index";

type MaybePromise<T> = T | Promise<T>;

export type BrowserCommandIntent = Extract<
  Contract,
  { kind: "browser.command_intent" }
>;
export type CommandAcceptanceStatus = Extract<
  Contract,
  { kind: "command.acceptance" }
>;
export type CommandActivationIntent = Extract<
  Contract,
  { kind: "activation.intent" }
>;
export type CommandCapability = CommandAcceptanceStatus["capability"];
export type CommandProvenance = CommandAcceptanceStatus["provenance"];

export interface CommandAcceptanceRoute {
  activationName: string;
  scriptKey: string;
  scriptRevision?: number;
  subject: string;
}

export interface CommandAcceptanceClaim {
  status: CommandAcceptanceStatus;
  created: boolean;
}

export interface CommandAcceptanceStore {
  claim(
    commandId: string,
    create: () => MaybePromise<CommandAcceptanceStatus>,
  ): MaybePromise<CommandAcceptanceClaim>;
  nextSequence?(): number;
}

export interface CommandAcceptanceStoreSnapshot {
  statuses: Record<string, CommandAcceptanceStatus>;
}

export interface MemoryCommandAcceptanceStore extends CommandAcceptanceStore {
  snapshot(): CommandAcceptanceStoreSnapshot;
}

export interface CommandAcceptanceOptions {
  provenance: CommandProvenance;
  capability: CommandCapability;
  currentArtifactRevision: string;
  routes: Record<string, CommandAcceptanceRoute>;
  store: CommandAcceptanceStore;
  now?: () => string;
}

export interface CommandAcceptanceResult {
  status: CommandAcceptanceStatus;
  activation?: CommandActivationIntent;
  duplicate: boolean;
}

export interface CommandAcceptance {
  accept(input: unknown | Promise<unknown>): Promise<CommandAcceptanceResult>;
}

export function createMemoryCommandAcceptanceStore(): MemoryCommandAcceptanceStore {
  const statuses: Record<string, CommandAcceptanceStatus> = {};
  const claims = new Map<string, Promise<CommandAcceptanceStatus>>();
  let sequence = 0;

  return {
    async claim(commandId, create) {
      const existing = statuses[commandId];
      if (existing) return { status: existing, created: false };

      const inflight = claims.get(commandId);
      if (inflight) return { status: await inflight, created: false };

      const claim = Promise.resolve()
        .then(create)
        .then((status) => {
          statuses[commandId] = status;
          return status;
        })
        .finally(() => claims.delete(commandId));
      claims.set(commandId, claim);
      return { status: await claim, created: true };
    },

    nextSequence() {
      sequence += 1;
      return sequence;
    },

    snapshot() {
      return {
        statuses: { ...statuses },
      };
    },
  };
}

export function createCommandAcceptance(
  opts: CommandAcceptanceOptions,
): CommandAcceptance {
  const now = opts.now ?? (() => new Date().toISOString());
  let sequence = 0;

  return {
    async accept(input): Promise<CommandAcceptanceResult> {
      try {
        const intent = parseBrowserCommand(await input);
        const commandId = intent.commandId;
        let activation: CommandActivationIntent | undefined;
        let draft: CommandAcceptanceStatus | undefined;

        const claim = await materialize(
          commandId,
          () => {
            const observedAt = now();
            const denied = deny(intent);
            activation = denied
              ? undefined
              : activationFor(intent, commandId, observedAt);
            draft = statusFor({
              intent,
              commandId,
              observedAt,
              sequence: nextSequence(),
              state: denied ? "rejected" : "accepted",
              error: denied,
            });
            return draft;
          },
          () => ({
            commandId,
            status: draft?.status,
          }),
        );

        if (!claim.created) {
          return { status: claim.status, duplicate: true };
        }
        return { status: claim.status, activation, duplicate: false };
      } catch (error) {
        if (error instanceof TinkabotRuntimeError) throw error;
        throw new TinkabotRuntimeError(
          "CommandAcceptanceCritical",
          "Command acceptance failed with an unknown error",
          {
            origin: origin("accept"),
            cause: error,
          },
        );
      }
    },
  };

  function nextSequence() {
    if (opts.store.nextSequence) return opts.store.nextSequence();
    sequence += 1;
    return sequence;
  }

  async function materialize(
    commandId: string,
    create: () => MaybePromise<CommandAcceptanceStatus>,
    details: () => Record<string, unknown>,
  ): Promise<CommandAcceptanceClaim> {
    try {
      return await opts.store.claim(commandId, create);
    } catch (error) {
      throw new TinkabotRuntimeError(
        "StatusMaterializationFailed",
        "Command acceptance status materialization failed",
        {
          origin: origin("materializeStatus", details()),
          cause: error,
        },
      );
    }
  }

  function deny(intent: BrowserCommandIntent): CommandStatusError | undefined {
    if (
      intent.context.sessionId !== opts.capability.sessionId ||
      intent.context.capabilityId !== opts.capability.capabilityId
    ) {
      return statusError(
        "CapabilityMismatch",
        "Command capability context does not match active lease",
        managedAuthOrigin(),
      );
    }
    if (opts.capability.leaseStatus === "revoked") {
      return statusError(
        "RevokedLease",
        "Capability lease has been revoked",
        managedAuthOrigin(),
      );
    }
    if (opts.capability.leaseStatus === "expired") {
      return statusError(
        "ExpiredLease",
        "Capability lease has expired",
        managedAuthOrigin(),
      );
    }
    if (
      opts.provenance.appRevision !== opts.capability.appRevision ||
      opts.provenance.schemaVersion !== opts.capability.schemaVersion
    ) {
      return statusError(
        "StaleRevision",
        "Capability revision does not match command provenance",
        managedAuthOrigin(),
      );
    }
    if (intent.expectedRevision !== opts.currentArtifactRevision) {
      return statusError(
        "StaleRevision",
        `Expected ${intent.expectedRevision} but current ${opts.currentArtifactRevision}`,
        origin("accept"),
      );
    }
    if (!opts.routes[intent.command]) {
      return statusError(
        "AcceptanceDenied",
        `Command is not routable: ${intent.command}`,
        origin("accept"),
      );
    }
    if (intent.context.chain.hop + 1 > intent.context.chain.maxHops) {
      return statusError(
        "ActivationLoopSuppressed",
        "Command chain hop limit is exhausted",
        origin("accept"),
      );
    }
  }

  function statusFor(input: {
    intent: BrowserCommandIntent;
    commandId: string;
    observedAt: string;
    sequence: number;
    state: CommandAcceptanceStatus["status"];
    error?: CommandStatusError;
  }): CommandAcceptanceStatus {
    return {
      kind: "command.acceptance",
      type: "command.acceptance",
      commandId: input.commandId,
      status: input.state,
      sequence: input.sequence,
      observedAt: input.observedAt,
      provenance: opts.provenance,
      capability: opts.capability,
      chain: input.intent.context.chain,
      error: input.error,
    };
  }

  function activationFor(
    intent: BrowserCommandIntent,
    commandId: string,
    observedAt: string,
  ): CommandActivationIntent {
    const route = opts.routes[intent.command]!;
    const activationName = route.activationName;
    const chain = {
      ...intent.context.chain,
      parentId: commandId,
      hop: intent.context.chain.hop + 1,
    };
    const activation: CommandActivationIntent = {
      kind: "activation.intent",
      activationId: [
        "act",
        route.scriptKey,
        activationName,
        commandId,
      ].join(":"),
      triggerId: commandId,
      scriptKey: route.scriptKey,
      scriptRevision: route.scriptRevision,
      sourcePrincipal: {
        principalId: "principal.source.command_acceptance",
        sourceId: `src-command-${activationName}`,
        sourceKind: "command_acceptance",
        authorityRef: `auth.source.command_acceptance.${activationName}`,
      },
      sourceLease: {
        leaseId: opts.capability.leaseId,
        leaseStatus: opts.capability.leaseStatus,
        appRevision: opts.provenance.appRevision,
        schemaVersion: opts.provenance.schemaVersion,
        scriptRevision: route.scriptRevision,
      },
      source: {
        kind: "command_acceptance",
        activationName,
        subject: route.subject,
        commandId,
        command: intent.command,
        artifactId: intent.context.artifactId,
        artifactRevision: intent.context.artifactRevision,
        frameId: intent.context.frameId,
      },
      payload: intent.payload,
      headers: {
        "tb-chain-id": intent.context.chain.chainId,
        "tb-command-id": commandId,
      },
      observedAt,
      chain,
      dedupeKey: [
        "command_acceptance",
        route.scriptKey,
        activationName,
        commandId,
      ].join(":"),
      provenance: opts.provenance,
      capability: opts.capability,
    };

    return parseActivation(activation);
  }

  function origin(
    operation: string,
    details?: Record<string, unknown>,
  ): RuntimeErrorOrigin {
    const base: RuntimeErrorOrigin = {
      layer: "CommandAcceptance",
      operation,
    };
    return details ? { ...base, details } : base;
  }
}

interface CommandStatusError {
  kind: RuntimeErrorKind;
  message: string;
  origin: {
    layer: RuntimeErrorLayer;
    operation: string;
  };
}

function parseBrowserCommand(input: unknown): BrowserCommandIntent {
  const parsed = parseContract(input);
  if (parsed.kind !== "browser.command_intent") {
    throw new TinkabotRuntimeError("ContractInvalid", "Expected browser command intent", {
      origin: {
        layer: "ContractAuthority",
        operation: "parseBrowserCommand",
      },
    });
  }
  return parsed;
}

function parseActivation(input: unknown): CommandActivationIntent {
  const parsed = parseContract(input);
  if (parsed.kind !== "activation.intent") {
    throw new TinkabotRuntimeError("ContractInvalid", "Expected activation intent", {
      origin: {
        layer: "ContractAuthority",
        operation: "parseActivation",
      },
    });
  }
  return parsed;
}

function managedAuthOrigin(): CommandStatusError["origin"] {
  return {
    layer: "ManagedAuth",
    operation: "authorizeCommand",
  };
}

function statusError(
  kind: RuntimeErrorKind,
  message: string,
  origin: CommandStatusError["origin"],
): CommandStatusError {
  return { kind, message, origin };
}
