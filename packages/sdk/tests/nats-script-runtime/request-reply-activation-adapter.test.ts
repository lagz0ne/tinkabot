import { describe, expect, test } from "bun:test";
import {
  PermissionResolver,
  TinkabotRuntimeError,
  activateRequestReply,
  type ActivateRequestReplyOptions,
  type ScriptMetadata,
} from "../../src/nats-script-runtime/index";

describe("RequestReplyActivationAdapter", () => {
  test("T11-RR-AUTH creates authorized intent and preserves request envelope fields", () => {
    const metadata = validMetadata();
    const resolver = PermissionResolver.fromMetadata(metadata, schemaOptions());

    const intent = activateRequestReply({
      metadata,
      resolver,
      envelope: {
        activationName: "request",
        scriptKey: "scripts.proof.echo",
        scriptRevision: 3,
        subject: "tb.proof.runtime.execute",
        requestId: "req-101",
        payload: { text: "activate" },
        headers: {
          "tb.trace": "trace-101",
        },
        replySubject: "tb.proof.reply.req_101",
        observedAt: "2026-06-05T01:00:00.000Z",
        chain: {
          chainId: "chain-101",
          rootId: "root-101",
          hop: 2,
          maxHops: 5,
        },
      },
    });

    expect(intent.source.kind).toBe("request_reply");
    expect(intent.source.activationName).toBe("request");
    expect(intent.source.subject).toBe("tb.proof.runtime.execute");
    expect(intent.source.requestId).toBe("req-101");
    expect(intent.reply?.subject).toBe("tb.proof.reply.req_101");
    expect(intent.payload).toEqual({ text: "activate" });
    expect(intent.headers).toEqual({ "tb.trace": "trace-101" });
    expect(intent.chain.hop).toBe(2);
    expect(intent.chain.maxHops).toBe(5);
    expect(intent.dedupeKey).toBe(
      "request_reply:scripts.proof.echo:request:tb.proof.runtime.execute:req-101",
    );
  });

  test("T11-RR-AUTH maps missing exposure, subject mismatch, and subscribe denial to ActivationUnauthorized", async () => {
    const metadata = validMetadata();

    const missingExposure = await captureRuntimeError(() =>
      Promise.resolve(
        activateRequestReply({
          metadata,
          resolver: PermissionResolver.fromMetadata(metadata, schemaOptions()),
          envelope: baseEnvelope({ activationName: "missing" }),
        }),
      ),
    );
    expect(missingExposure.kind).toBe("ActivationUnauthorized");
    expect(missingExposure.origin.layer).toBe("Activation");

    const mismatch = await captureRuntimeError(() =>
      Promise.resolve(
        activateRequestReply({
          metadata,
          resolver: PermissionResolver.fromMetadata(metadata, schemaOptions()),
          envelope: baseEnvelope({ subject: "tb.proof.runtime.other" }),
        }),
      ),
    );
    expect(mismatch.kind).toBe("ActivationUnauthorized");

    const deniedMetadata = validMetadata({
      nats: {
        permissions: {
          subscribe: {
            allow: ["tb.proof.runtime.>"],
            deny: ["tb.proof.runtime.execute"],
          },
        },
      },
    });
    const denied = await captureRuntimeError(() =>
      Promise.resolve(
        activateRequestReply({
          metadata: deniedMetadata,
          resolver: PermissionResolver.fromMetadata(deniedMetadata, schemaOptions()),
          envelope: baseEnvelope(),
        }),
      ),
    );
    expect(denied.kind).toBe("ActivationUnauthorized");

    const notAllowedMetadata = validMetadata({
      nats: {
        permissions: {
          subscribe: {
            allow: ["tb.proof.other.>"],
          },
        },
      },
    });
    const notAllowed = await captureRuntimeError(() =>
      Promise.resolve(
        activateRequestReply({
          metadata: notAllowedMetadata,
          resolver: PermissionResolver.fromMetadata(
            notAllowedMetadata,
            schemaOptions(),
          ),
          envelope: baseEnvelope(),
        }),
      ),
    );
    expect(notAllowed.kind).toBe("ActivationUnauthorized");
  });

  test("T11-RR-ERR maps invalid input, propagates lower owners, and wraps unknowns", async () => {
    const metadata = validMetadata();

    const invalid = await captureRuntimeError(() =>
      Promise.resolve(
        activateRequestReply({
          metadata,
          resolver: PermissionResolver.fromMetadata(metadata, schemaOptions()),
          envelope: baseEnvelope({ activationName: "" }),
        }),
      ),
    );
    expect(invalid.kind).toBe("ActivationConfigInvalid");

    const loop = await captureRuntimeError(() =>
      Promise.resolve(
        activateRequestReply({
          metadata,
          resolver: PermissionResolver.fromMetadata(metadata, schemaOptions()),
          envelope: baseEnvelope({
            chain: {
              chainId: "chain-loop",
              rootId: "root-loop",
              hop: 9,
              maxHops: 8,
            },
          }),
        }),
      ),
    );
    expect(loop.kind).toBe("ActivationLoopSuppressed");

    const metadataError = new TinkabotRuntimeError(
      "MetadataCritical",
      "metadata failed",
      {
        origin: {
          layer: "MetadataSchema",
          operation: "validate",
        },
      },
    );
    const lowerMetadata = await captureRuntimeError(() =>
      Promise.resolve(
        activateRequestReply({
          metadata,
          resolver: throwingResolver(metadataError),
          envelope: baseEnvelope(),
        }),
      ),
    );
    expect(lowerMetadata).toBe(metadataError);

    const permissionCritical = new TinkabotRuntimeError(
      "PermissionCritical",
      "permission failed",
      {
        origin: {
          layer: "ImportsPermissions",
          operation: "assertActivationSource",
        },
      },
    );
    const lowerPermission = await captureRuntimeError(() =>
      Promise.resolve(
        activateRequestReply({
          metadata,
          resolver: throwingResolver(permissionCritical),
          envelope: baseEnvelope(),
        }),
      ),
    );
    expect(lowerPermission).toBe(permissionCritical);

    const unknown = await captureRuntimeError(() =>
      Promise.resolve(
        activateRequestReply({
          metadata,
          resolver: throwingResolver("boom"),
          envelope: baseEnvelope(),
        }),
      ),
    );
    expect(unknown.kind).toBe("ActivationCritical");
    expect(unknown.origin.layer).toBe("Activation");
    expect(unknown.causeValue).toBe("boom");
  });
});

function baseEnvelope(
  override: Partial<Parameters<typeof activateRequestReply>[0]["envelope"]> = {},
): Parameters<typeof activateRequestReply>[0]["envelope"] {
  return {
    activationName: "request",
    scriptKey: "scripts.proof.echo",
    scriptRevision: 3,
    subject: "tb.proof.runtime.execute",
    requestId: "req-101",
    payload: { text: "activate" },
    headers: {
      "tb.trace": "trace-101",
    },
    observedAt: "2026-06-05T01:00:00.000Z",
    ...override,
  };
}

function validMetadata(
  override: Partial<Omit<ScriptMetadata, "runtime" | "security" | "schemas" | "nats">> & {
    nats?: Partial<Omit<ScriptMetadata["nats"], "permissions">> & {
      permissions?: Partial<ScriptMetadata["nats"]["permissions"]>;
    };
  } = {},
): ScriptMetadata {
  return {
    id: "script.proof.echo",
    desc: "Echoes a proof input through the mediated runtime facade.",
    runtime: {
      language: "typescript",
      runner: "bun",
      sandbox: "none",
    },
    security: {
      trust: "trusted",
      directNats: false,
    },
    schemas: {
      input: "tb.schema.proof.input.echo.v1",
      output: "tb.schema.proof.output.echo.v1",
      ipc: {
        progress: "tb.schema.proof.ipc.progress.v1",
        publishRequest: "tb.schema.proof.ipc.publish_request.v1",
      },
      event: "tb.schema.proof.event.execution.v1",
    },
    nats: {
      io: {
        input: {
          subject: "tb.proof.runtime.execute",
        },
      },
      activations: {
        request: {
          kind: "request_reply",
          subject: "tb.proof.runtime.execute",
          desc: "Request/reply runtime activation.",
        },
        ...override.nats?.activations,
      },
      imports: {
        outbox: {
          kind: "publish",
          subjects: ["tb.proof.out.allowed.>"],
          desc: "Allowed proof output subjects.",
        },
        ...override.nats?.imports,
      },
      permissions: {
        publish: {
          allow: ["tb.proof.out.allowed.>"],
        },
        subscribe: {
          allow: ["tb.proof.runtime.>"],
        },
        allow_responses: {
          max: 1,
          expiresMs: 500,
        },
        ...override.nats?.permissions,
      },
      advanced: {
        ...override.nats?.advanced,
      },
    },
  };
}

function throwingResolver(
  error: unknown,
): ActivateRequestReplyOptions["resolver"] {
  return {
    assertActivationSource: () => {
      throw error;
    },
  };
}

function schemaOptions() {
  return {
    schemaIds: new Set([
      "tb.schema.proof.input.echo.v1",
      "tb.schema.proof.output.echo.v1",
      "tb.schema.proof.ipc.progress.v1",
      "tb.schema.proof.ipc.publish_request.v1",
      "tb.schema.proof.event.execution.v1",
    ]),
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
