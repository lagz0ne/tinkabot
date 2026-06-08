import { describe, expect, test } from "bun:test";
import {
  TinkabotRuntimeError,
  bindDedicatedWorkerMediator,
  createFrontendMediator,
  createMaterializerStore,
  type DedicatedWorkerScopeLike,
  type FrontendCommandIntent,
  type MediatorToContentMessage,
} from "../../src/index";

describe("BrowserFrontendDedicatedWorkerMediator", () => {
  test("T12-FE-INTENT stamps trusted context and publishes only mediated command intents", async () => {
    const published: FrontendCommandIntent[] = [];
    const mediator = createFrontendMediator({
      context: trustedContext(),
      allowedCommands: ["select_artifact"],
      createCommandId: () => "cmd-001",
      now: () => "2026-06-08T00:00:00.000Z",
      publishCommand: async (intent) => {
        published.push(intent);
      },
    });

    const accepted = await mediator.handleContentMessage({
      type: "content.intent",
      command: "select_artifact",
      expectedRevision: "artifact.rev.7",
      payload: { artifactKey: "preview.main" },
    });

    expect(accepted.type).toBe("mediator.command_status");
    expect(accepted.commandId).toBe("cmd-001");
    expect(accepted.status).toBe("accepted");
    expect(published).toHaveLength(1);
    expect(published[0]).toEqual({
      type: "frontend.command_intent",
      commandId: "cmd-001",
      command: "select_artifact",
      expectedRevision: "artifact.rev.7",
      payload: { artifactKey: "preview.main" },
      observedAt: "2026-06-08T00:00:00.000Z",
      context: trustedContext(),
    });
  });

  test("T12-FE-DENY rejects raw NATS vocabulary and disallowed commands before transport", async () => {
    const published: FrontendCommandIntent[] = [];
    const mediator = createFrontendMediator({
      context: trustedContext(),
      allowedCommands: ["select_artifact"],
      publishCommand: async (intent) => {
        published.push(intent);
      },
    });

    const rawSubject = await captureRuntimeError(() =>
      mediator.handleContentMessage({
        type: "content.intent",
        command: "select_artifact",
        expectedRevision: "artifact.rev.7",
        subject: "tb.internal.admin.delete",
        payload: {
          subject: "tb.internal.admin.delete",
        },
      }),
    );
    expect(rawSubject.kind).toBe("FrontendCapabilityDenied");
    expect(rawSubject.origin.layer).toBe("FrontendMediator");

    const disallowed = await captureRuntimeError(() =>
      mediator.handleContentMessage({
        type: "content.intent",
        command: "publish_raw",
        expectedRevision: "artifact.rev.7",
        payload: {},
      }),
    );
    expect(disallowed.kind).toBe("FrontendCapabilityDenied");
    expect(published).toHaveLength(0);
  });

  test("T12-FE-MATERIALIZER applies monotonic projection and command status messages", () => {
    const store = createMaterializerStore();

    store.apply({
      type: "mediator.state",
      projectionId: "main",
      revision: "artifact.rev.7",
      sequence: 7,
      value: { title: "current" },
    });
    store.apply({
      type: "mediator.state",
      projectionId: "main",
      revision: "artifact.rev.6",
      sequence: 6,
      value: { title: "stale" },
    });
    store.apply({
      type: "mediator.command_status",
      commandId: "cmd-001",
      status: "applied",
      sequence: 9,
      observedAt: "2026-06-08T00:00:01.000Z",
    });

    expect(store.snapshot()).toEqual({
      projections: {
        main: {
          revision: "artifact.rev.7",
          sequence: 7,
          value: { title: "current" },
        },
      },
      commands: {
        "cmd-001": {
          status: "applied",
          sequence: 9,
          observedAt: "2026-06-08T00:00:01.000Z",
        },
      },
      errors: [],
    });
  });

  test("T12-FE-WORKER bridges worker messages to mediator status and error outputs", async () => {
    const scope = new FakeWorkerScope();
    const mediator = createFrontendMediator({
      context: trustedContext(),
      allowedCommands: ["select_artifact"],
      createCommandId: () => "cmd-worker",
      now: () => "2026-06-08T00:00:00.000Z",
      publishCommand: async () => {},
    });
    bindDedicatedWorkerMediator(scope, mediator);

    await scope.dispatch({
      type: "content.intent",
      command: "select_artifact",
      expectedRevision: "artifact.rev.7",
      payload: {},
    });
    await scope.dispatch({
      type: "content.intent",
      command: "select_artifact",
      expectedRevision: "artifact.rev.7",
      payload: { token: "secret" },
    });

    expect(scope.posted[0]).toEqual({
      type: "mediator.command_status",
      commandId: "cmd-worker",
      status: "accepted",
      sequence: 1,
      observedAt: "2026-06-08T00:00:00.000Z",
    });
    expect(scope.posted[1]).toMatchObject({
      type: "mediator.error",
      error: {
        kind: "FrontendCapabilityDenied",
      },
    });
  });
});

class FakeWorkerScope implements DedicatedWorkerScopeLike {
  posted: MediatorToContentMessage[] = [];
  private listener?: (event: { data: unknown }) => void | Promise<void>;

  addEventListener(
    type: "message",
    listener: (event: { data: unknown }) => void | Promise<void>,
  ): void {
    if (type === "message") {
      this.listener = listener;
    }
  }

  postMessage(message: MediatorToContentMessage): void {
    this.posted.push(message);
  }

  async dispatch(message: unknown): Promise<void> {
    if (!this.listener) throw new Error("worker listener not bound");
    await this.listener({ data: message });
  }
}

function trustedContext() {
  return {
    sessionId: "session-001",
    capabilityId: "cap-001",
    artifactId: "artifact-001",
    artifactRevision: "artifact.rev.7",
    frameId: "frame-001",
    chain: {
      chainId: "chain-001",
      rootId: "root-001",
      parentId: "parent-001",
      hop: 1,
      maxHops: 5,
    },
  };
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
