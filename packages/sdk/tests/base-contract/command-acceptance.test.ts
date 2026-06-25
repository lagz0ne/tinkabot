import { describe, expect, test } from "bun:test";
import {
  createCommandAcceptance,
  createMemoryCommandAcceptanceStore,
  parseContract,
  type Contract,
} from "../../src/base-contract/index";
import { TinkabotRuntimeError } from "../../src/nats-script-runtime/errors";

type BrowserCommand = Extract<Contract, { kind: "browser.command_intent" }>;

const root = new URL("../../../../", import.meta.url);
const dir = new URL("schemas/base/v1/", root);

describe("CommandAcceptance", () => {
  test("T-CMD-ACCEPT accepts schema-valid browser intent, materializes status, and emits activation handoff", async () => {
    const store = createMemoryCommandAcceptanceStore();
    const acceptance = createAcceptance({ store });
    const intent = await browserCommand();

    const result = await acceptance.accept(intent);

    expect(result.duplicate).toBe(false);
    expect(result.status).toMatchObject({
      kind: "command.acceptance",
      type: "command.acceptance",
      commandId: "cmd-001",
      status: "accepted",
      sequence: 1,
      observedAt: "2026-06-08T00:00:10.000Z",
      provenance: provenance(),
      capability: capability(),
      chain: intent.context.chain,
    });
    expect(parseContract(result.status).kind).toBe("command.acceptance");
    expect(store.snapshot().statuses["cmd-001"]).toEqual(result.status);

    expect(result.activation).toMatchObject({
      kind: "activation.intent",
      activationId: "act:scripts.proof.select_artifact:browser_command:cmd-001",
      triggerId: "cmd-001",
      scriptKey: "scripts.proof.select_artifact",
      scriptRevision: 7,
      source: {
        kind: "command_acceptance",
        activationName: "browser_command",
        subject: "tb.proof.runtime.execute",
        commandId: "cmd-001",
        command: "select_artifact",
        artifactId: "artifact-001",
        artifactRevision: "artifact.rev.7",
        frameId: "frame-001",
      },
      payload: { artifactKey: "preview.main" },
      headers: {
        "tb-chain-id": "chain-001",
        "tb-command-id": "cmd-001",
      },
      observedAt: "2026-06-08T00:00:10.000Z",
      chain: {
        chainId: "chain-001",
        rootId: "root-001",
        parentId: "cmd-001",
        hop: 2,
        maxHops: 5,
      },
      dedupeKey: "command_acceptance:scripts.proof.select_artifact:browser_command:cmd-001",
      provenance: provenance("command-acceptance"),
      capability: capability(),
    });
    expect(parseContract(result.activation).kind).toBe("activation.intent");
  });

  test("T-CMD-IDEMPOTENCY resolves duplicate command without a second activation handoff", async () => {
    const store = createMemoryCommandAcceptanceStore();
    const acceptance = createAcceptance({ store });
    const intent = await browserCommand();

    const first = await acceptance.accept(intent);
    const second = await acceptance.accept(intent);

    expect(second.duplicate).toBe(true);
    expect(second.status).toEqual(first.status);
    expect(second.activation).toBeUndefined();
    expect(Object.keys(store.snapshot().statuses)).toEqual(["cmd-001"]);
  });

  test("T-CMD-IDEMPOTENCY-CURRENT resolves concurrent duplicates atomically", async () => {
    const store = createMemoryCommandAcceptanceStore();
    const acceptance = createAcceptance({ store });
    const intent = await browserCommand();

    const results = await Promise.all([
      acceptance.accept(intent),
      acceptance.accept(intent),
    ]);

    const created = results.filter((item) => !item.duplicate);
    const duplicates = results.filter((item) => item.duplicate);
    expect(created).toHaveLength(1);
    expect(duplicates).toHaveLength(1);
    expect(created[0]?.activation).toBeDefined();
    expect(duplicates[0]?.activation).toBeUndefined();
    expect(duplicates[0]?.status).toEqual(created[0]?.status);
    expect(Object.keys(store.snapshot().statuses)).toEqual(["cmd-001"]);
  });

  test("T-CMD-DENY rejects stale revision and unknown commands as command-acceptance statuses", async () => {
    const store = createMemoryCommandAcceptanceStore();
    const acceptance = createAcceptance({ store });

    const stale = await acceptance.accept({
      ...(await browserCommand()),
      commandId: "cmd-stale-001",
      expectedRevision: "artifact.rev.6",
    });
    expect(stale.activation).toBeUndefined();
    expect(stale.status).toMatchObject({
      commandId: "cmd-stale-001",
      status: "rejected",
      sequence: 1,
      error: {
        kind: "StaleRevision",
        message: "Expected artifact.rev.6 but current artifact.rev.7",
        origin: {
          layer: "CommandAcceptance",
          operation: "accept",
        },
      },
    });
    expect(parseContract(stale.status).kind).toBe("command.acceptance");

    const denied = await acceptance.accept({
      ...(await browserCommand()),
      commandId: "cmd-denied-001",
      command: "delete_artifact",
    });
    expect(denied.activation).toBeUndefined();
    expect(denied.status).toMatchObject({
      commandId: "cmd-denied-001",
      status: "rejected",
      sequence: 2,
      error: {
        kind: "AcceptanceDenied",
        origin: {
          layer: "CommandAcceptance",
          operation: "accept",
        },
      },
    });
    expect(parseContract(denied.status).kind).toBe("command.acceptance");
    expect(store.snapshot().statuses["cmd-stale-001"]).toEqual(stale.status);
    expect(store.snapshot().statuses["cmd-denied-001"]).toEqual(denied.status);
  });

  test("T-CMD-CHAIN rejects activation handoff when chain budget is exhausted", async () => {
    const store = createMemoryCommandAcceptanceStore();
    const acceptance = createAcceptance({ store });
    const intent = await browserCommand();
    const result = await acceptance.accept({
      ...intent,
      commandId: "cmd-loop-001",
      context: {
        ...intent.context,
        chain: {
          ...intent.context.chain,
          hop: 5,
          maxHops: 5,
        },
      },
    });

    expect(result.activation).toBeUndefined();
    expect(result.status).toMatchObject({
      commandId: "cmd-loop-001",
      status: "rejected",
      error: {
        kind: "ActivationLoopSuppressed",
        message: "Command chain hop limit is exhausted",
        origin: {
          layer: "CommandAcceptance",
          operation: "accept",
        },
      },
    });
    expect(parseContract(result.status).kind).toBe("command.acceptance");
    expect(store.snapshot().statuses["cmd-loop-001"]).toEqual(result.status);
  });

  test("T-CMD-CAPABILITY materializes revoked and expired capability denial without activation", async () => {
    for (const [leaseStatus, kind] of [
      ["revoked", "RevokedLease"],
      ["expired", "ExpiredLease"],
    ] as const) {
      const store = createMemoryCommandAcceptanceStore();
      const acceptance = createAcceptance({
        store,
        capability: capability(leaseStatus),
      });
      const result = await acceptance.accept({
        ...(await browserCommand()),
        commandId: `cmd-${leaseStatus}-001`,
      });

      expect(result.activation).toBeUndefined();
      expect(result.status).toMatchObject({
        status: "rejected",
        capability: capability(leaseStatus),
        error: {
          kind,
          origin: {
            layer: "ManagedAuth",
            operation: "authorizeCommand",
          },
        },
      });
      expect(parseContract(result.status).kind).toBe("command.acceptance");
      expect(store.snapshot().statuses[`cmd-${leaseStatus}-001`]).toEqual(
        result.status,
      );
    }
  });

  test("T-CMD-CAPABILITY-CONTEXT binds command context to the active capability", async () => {
    for (const [field, value] of [
      ["sessionId", "session-other"],
      ["capabilityId", "cap-other"],
    ] as const) {
      const store = createMemoryCommandAcceptanceStore();
      const acceptance = createAcceptance({ store });
      const intent = await browserCommand();
      const result = await acceptance.accept({
        ...intent,
        commandId: `cmd-${field}-mismatch`,
        context: {
          ...intent.context,
          [field]: value,
        },
      });

      expect(result.activation).toBeUndefined();
      expect(result.status).toMatchObject({
        commandId: `cmd-${field}-mismatch`,
        status: "rejected",
        error: {
          kind: "CapabilityMismatch",
          message: "Command capability context does not match active lease",
          origin: {
            layer: "ManagedAuth",
            operation: "authorizeCommand",
          },
        },
      });
      expect(parseContract(result.status).kind).toBe("command.acceptance");
      expect(store.snapshot().statuses[`cmd-${field}-mismatch`]).toEqual(
        result.status,
      );
    }
  });

  test("T-CMD-PARTICIPANT-CONTEXT preserves trusted-shell app and participant context", async () => {
    const intent = await browserCommand({
      command: "participant_action",
      commandId: "cmd-participant-001",
      payload: {
        actionId: "browser-1",
        stateKey: "apps.demo.state.round",
        baseRevision: 7,
        value: { answer: "blue" },
      },
      context: {
        appId: "demo",
        participantId: "alice",
      },
    });

    expect(intent.context.appId).toBe("demo");
    expect(intent.context.participantId).toBe("alice");
    expect(parseContract(intent).kind).toBe("browser.command_intent");
  });

  test("T-CMD-CONTRACT rejects raw-authority intent before status materialization", async () => {
    const store = createMemoryCommandAcceptanceStore();
    const acceptance = createAcceptance({ store });

    const error = await captureRuntimeError(() =>
      acceptance.accept(readJson("fixtures/invalid/browser-command-raw-nats.json")),
    );

    expect(error.kind).toBe("ContractInvalid");
    expect(error.origin.layer).toBe("ContractAuthority");
    expect(store.snapshot().statuses).toEqual({});

    const missingId = await captureRuntimeError(() =>
      acceptance.accept(readJson("fixtures/invalid/browser-command-missing-command-id.json")),
    );
    expect(missingId.kind).toBe("ContractInvalid");
    expect(missingId.origin.layer).toBe("ContractAuthority");
    expect(store.snapshot().statuses).toEqual({});
  });

  test("T-CMD-STATUS reports materialization failure as command-acceptance failure", async () => {
    const acceptance = createAcceptance({
      store: {
        async claim(_commandId, create) {
          await create();
          throw new Error("store unavailable");
        },
      },
    });

    const error = await captureRuntimeError(() =>
      acceptance.accept(browserCommand()),
    );

    expect(error.kind).toBe("StatusMaterializationFailed");
    expect(error.origin.layer).toBe("CommandAcceptance");
    expect(error.origin.details).toMatchObject({
      commandId: "cmd-001",
      status: "accepted",
    });
  });
});

function createAcceptance(
  overrides: Partial<Parameters<typeof createCommandAcceptance>[0]> = {},
) {
  return createCommandAcceptance({
    provenance: provenance(),
    capability: capability(),
    currentArtifactRevision: "artifact.rev.7",
    store: createMemoryCommandAcceptanceStore(),
    now: () => "2026-06-08T00:00:10.000Z",
    routes: {
      select_artifact: {
        activationName: "browser_command",
        scriptKey: "scripts.proof.select_artifact",
        scriptRevision: 7,
        subject: "tb.proof.runtime.execute",
      },
    },
    ...overrides,
  });
}

async function browserCommand(overrides: Record<string, any> = {}): Promise<BrowserCommand> {
  const base = await readJson("fixtures/valid/browser-command.json");
  return parseBrowserCommand({
    ...base,
    ...overrides,
    context: {
      ...base.context,
      ...(overrides.context ?? {}),
    },
  });
}

function parseBrowserCommand(input: unknown): BrowserCommand {
  const parsed = parseContract(input);
  if (parsed.kind !== "browser.command_intent") {
    throw new Error("expected browser command");
  }
  return parsed;
}

function provenance(producer = "command-acceptance") {
  return {
    schemaId: "tb.schema.base.contract_authority.v1" as const,
    schemaVersion: "v1" as const,
    appRevision: "app.rev.1",
    createdAt: "2026-06-08T00:00:00.000Z",
    producer,
  };
}

function capability(leaseStatus: "active" | "revoked" | "expired" = "active") {
  return {
    principalId: "principal.browser.001",
    sessionId: "session-001",
    capabilityId: "cap-001",
    leaseId: "lease-001",
    leaseStatus,
    appRevision: "app.rev.1",
    schemaVersion: "v1" as const,
  };
}

async function readJson<T = any>(path: string): Promise<T> {
  return Bun.file(new URL(path, dir)).json();
}

async function captureRuntimeError(
  action: () => Promise<unknown>,
): Promise<TinkabotRuntimeError> {
  try {
    await action();
  } catch (error) {
    if (error instanceof TinkabotRuntimeError) return error;
    throw error;
  }
  throw new Error("expected action to fail");
}
