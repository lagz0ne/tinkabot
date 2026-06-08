import { describe, expect, test } from "bun:test";
import {
  PermissionResolver,
  TinkabotRuntimeError,
  type ScriptImport,
  type ScriptMetadata,
} from "../../src/nats-script-runtime/index";

type MetadataOverride = Partial<
  Omit<ScriptMetadata, "runtime" | "security" | "schemas" | "nats">
> & {
  runtime?: Partial<ScriptMetadata["runtime"]>;
  security?: Partial<NonNullable<ScriptMetadata["security"]>>;
  schemas?: Partial<Omit<ScriptMetadata["schemas"], "ipc">> & {
    ipc?: Partial<NonNullable<ScriptMetadata["schemas"]["ipc"]>>;
  };
  nats?: Partial<Omit<ScriptMetadata["nats"], "io" | "imports" | "permissions">> & {
    io?: Partial<NonNullable<ScriptMetadata["nats"]["io"]>>;
    imports?: Record<string, ScriptImport>;
    permissions?: Partial<ScriptMetadata["nats"]["permissions"]>;
  };
};

describe("PermissionResolver", () => {
  test("T07 propagates metadata/schema validation errors unchanged", async () => {
    const error = await captureRuntimeError(() =>
      Promise.resolve(
        PermissionResolver.fromMetadata(
          validMetadata({
            nats: {
              permissions: {
                publish: {
                  allow: ["tb.proof.out.<scriptId>"],
                },
              },
            },
          }),
          schemaOptions(),
        ),
      ),
    );

    expect(error.kind).toBe("PlaceholderSubjectRejected");
    expect(error.origin.layer).toBe("MetadataSchema");
  });

  test("T08 allows declared imports and concrete NATS wildcard publish patterns", () => {
    const resolver = PermissionResolver.fromMetadata(
      validMetadata(),
      schemaOptions(),
    );

    const imported = resolver.requireImport("outbox");

    expect(imported.kind).toBe("publish");
    if (imported.kind !== "publish") {
      throw new Error("expected publish import");
    }
    expect(imported.subjects).toContain("tb.proof.exec.*.progress");
    expect(() =>
      resolver.assertImportPublish(
        "outbox",
        "tb.proof.exec.exec_success_001.progress",
      ),
    ).not.toThrow();
    expect(() =>
      resolver.assertImportPublish("outbox", "tb.proof.out.allowed.a.b"),
    ).not.toThrow();
    expect(() =>
      resolver.assertResponsePublish("tb.proof.reply.exec_success_001", {
        replySubject: "tb.proof.reply.exec_success_001",
        usedResponses: 0,
        now: 100,
        expiresAt: 200,
      }),
    ).not.toThrow();
  });

  test("T08 rejects undeclared imports, deny-over-allow, response escape, and missing advanced capability", async () => {
    const resolver = PermissionResolver.fromMetadata(
      validMetadata(),
      schemaOptions(),
    );

    const undeclared = await captureRuntimeError(() =>
      Promise.resolve(resolver.requireImport("missing")),
    );
    expect(undeclared.kind).toBe("ImportNotDeclared");
    expect(undeclared.origin.layer).toBe("ImportsPermissions");

    const denied = await captureRuntimeError(() =>
      Promise.resolve(
        resolver.assertPublish("tb.proof.out.denied.exec_denied_publish_001"),
      ),
    );
    expect(denied.kind).toBe("PermissionDeniedByDenyRule");

    const wildcardMiss = await captureRuntimeError(() =>
      Promise.resolve(
        resolver.assertImportPublish(
          "outbox",
          "tb.proof.exec.exec_success_001.detail.progress",
        ),
      ),
    );
    expect(wildcardMiss.kind).toBe("PermissionDenied");

    const terminalWildcardBaseMiss = await captureRuntimeError(() =>
      Promise.resolve(
        resolver.assertImportPublish("outbox", "tb.proof.out.allowed"),
      ),
    );
    expect(terminalWildcardBaseMiss.kind).toBe("PermissionDenied");

    const responseEscape = await captureRuntimeError(() =>
      Promise.resolve(
        resolver.assertResponsePublish("tb.proof.reply.other", {
          replySubject: "tb.proof.reply.exec_success_001",
          usedResponses: 0,
          now: 100,
          expiresAt: 200,
        }),
      ),
    );
    expect(responseEscape.kind).toBe("ResponseAuthorityExceeded");

    const responseOverrun = await captureRuntimeError(() =>
      Promise.resolve(
        resolver.assertResponsePublish("tb.proof.reply.exec_success_001", {
          replySubject: "tb.proof.reply.exec_success_001",
          usedResponses: 1,
          now: 100,
          expiresAt: 200,
        }),
      ),
    );
    expect(responseOverrun.kind).toBe("ResponseAuthorityExceeded");

    const responseExpired = await captureRuntimeError(() =>
      Promise.resolve(
        resolver.assertResponsePublish("tb.proof.reply.exec_success_001", {
          replySubject: "tb.proof.reply.exec_success_001",
          usedResponses: 0,
          now: 250,
          expiresAt: 200,
        }),
      ),
    );
    expect(responseExpired.kind).toBe("ResponseAuthorityExceeded");

    const rawMetadata = validMetadata();
    rawMetadata.nats.imports.raw = {
      kind: "raw_nats",
      desc: "Advanced direct NATS access for a later explicit mode.",
    };
    const rawImportResolver = PermissionResolver.fromMetadata(
      rawMetadata,
      schemaOptions(),
    );

    const advanced = await captureRuntimeError(() =>
      Promise.resolve(rawImportResolver.requireImport("raw")),
    );
    expect(advanced.kind).toBe("AdvancedCapabilityDenied");

    const advancedMetadata = validMetadata();
    advancedMetadata.nats.advanced = {
      rawNats: true,
    };
    advancedMetadata.nats.imports.raw = {
      kind: "raw_nats",
      desc: "Explicit direct NATS access.",
    };
    const advancedResolver = PermissionResolver.fromMetadata(
      advancedMetadata,
      schemaOptions(),
    );

    expect(advancedResolver.requireImport("raw").kind).toBe("raw_nats");
  });

  test("T09 wraps unknown permission exceptions as PermissionCritical", async () => {
    const resolver = PermissionResolver.fromMetadata(validMetadata(), {
      ...schemaOptions(),
      subjectMatcher: () => {
        throw "matcher exploded";
      },
    });

    const error = await captureRuntimeError(() =>
      Promise.resolve(resolver.assertPublish("tb.proof.out.allowed.any")),
    );

    expect(error.kind).toBe("PermissionCritical");
    expect(error.origin).toMatchObject({
      layer: "ImportsPermissions",
      operation: "assertPublish",
    });
    expect(error.causeValue).toBe("matcher exploded");
  });

  test("T07 propagates injected metadata/schema errors unchanged", async () => {
    const metadataError = new TinkabotRuntimeError(
      "WildcardPatternInvalid",
      "bad wildcard",
      {
        origin: {
          layer: "MetadataSchema",
          operation: "validate",
        },
      },
    );

    const error = await captureRuntimeError(() =>
      Promise.resolve(
        PermissionResolver.fromMetadata(validMetadata(), {
          ...schemaOptions(),
          validateMetadata: () => {
            throw metadataError;
          },
        }),
      ),
    );

    expect(error).toBe(metadataError);
  });

  test("T10-ACT-PERM enforces activation source subscribe authority with deny precedence", async () => {
    const resolver = PermissionResolver.fromMetadata(
      validMetadata({
        nats: {
          activations: {
            request: {
              kind: "request_reply",
              subject: "tb.proof.runtime.execute",
              desc: "Allowed runtime request activation.",
            },
            blocked: {
              kind: "request_reply",
              subject: "tb.proof.runtime.blocked",
              desc: "Denied runtime request activation.",
            },
            unlisted: {
              kind: "request_reply",
              subject: "tb.proof.other.execute",
              desc: "Not covered by subscribe allow.",
            },
          },
          permissions: {
            subscribe: {
              allow: ["tb.proof.runtime.>"],
              deny: ["tb.proof.runtime.blocked"],
            },
          },
        } as Partial<ScriptMetadata["nats"]>,
      }),
      schemaOptions(),
    );

    expect(() => resolver.assertSubscribe("tb.proof.runtime.execute")).not.toThrow();
    expect(() => resolver.assertActivationSource("request")).not.toThrow();
    expect(() =>
      resolver.assertActivationSource("request", "tb.proof.runtime.execute"),
    ).not.toThrow();

    const denied = await captureRuntimeError(() =>
      Promise.resolve(resolver.assertActivationSource("blocked")),
    );
    expect(denied.kind).toBe("PermissionDeniedByDenyRule");
    expect(denied.origin.layer).toBe("ImportsPermissions");
    expect(denied.origin.operation).toBe("assertActivationSource");

    const missingAllow = await captureRuntimeError(() =>
      Promise.resolve(resolver.assertActivationSource("unlisted")),
    );
    expect(missingAllow.kind).toBe("PermissionDenied");
    expect(missingAllow.origin.layer).toBe("ImportsPermissions");

    const mismatchedExposure = await captureRuntimeError(() =>
      Promise.resolve(
        resolver.assertActivationSource("request", "tb.proof.runtime.other"),
      ),
    );
    expect(mismatchedExposure.kind).toBe("PermissionDenied");
    expect(mismatchedExposure.origin.operation).toBe("assertActivationSource");
  });
});

function validMetadata(
  override: MetadataOverride = {},
): ScriptMetadata {
  const base: ScriptMetadata = {
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
          subject: "tb.proof.inbox.echo",
        },
        output: {
          subject: "tb.proof.out.allowed.exec_success_001",
        },
      },
      imports: {
        outbox: {
          kind: "publish",
          subjects: [
            "tb.proof.exec.*.progress",
            "tb.proof.out.allowed.>",
          ],
          desc: "Allowed proof output subjects.",
        },
      },
      permissions: {
        publish: {
          allow: [
            "tb.proof.out.allowed.>",
            "tb.proof.exec.*.progress",
            "tb.proof.exec.*.event",
          ],
          deny: ["tb.proof.out.denied.>"],
        },
        subscribe: {
          allow: ["tb.proof.inbox.echo"],
        },
        allow_responses: {
          max: 1,
          expiresMs: 500,
        },
      },
    },
  };

  return mergeMetadata(base, override);
}

function mergeMetadata(
  base: ScriptMetadata,
  override: MetadataOverride,
): ScriptMetadata {
  return {
    ...base,
    ...override,
    runtime: {
      ...base.runtime,
      ...override.runtime,
    },
    security: {
      trust: override.security?.trust ?? base.security?.trust ?? "trusted",
      directNats:
        override.security?.directNats ?? base.security?.directNats,
    },
    schemas: {
      ...base.schemas,
      ...override.schemas,
      ipc: {
        ...base.schemas.ipc,
        ...override.schemas?.ipc,
      },
    },
    nats: {
      ...base.nats,
      ...override.nats,
      io: {
        ...base.nats.io,
        ...override.nats?.io,
      },
      imports: {
        ...base.nats.imports,
        ...override.nats?.imports,
      },
      permissions: {
        ...base.nats.permissions,
        ...override.nats?.permissions,
        publish: {
          ...base.nats.permissions.publish,
          ...override.nats?.permissions?.publish,
        },
        subscribe: {
          ...base.nats.permissions.subscribe,
          ...override.nats?.permissions?.subscribe,
        },
        allow_responses: {
          max:
            override.nats?.permissions?.allow_responses?.max ??
            base.nats.permissions.allow_responses?.max ??
            1,
          expiresMs:
            override.nats?.permissions?.allow_responses?.expiresMs ??
            base.nats.permissions.allow_responses?.expiresMs,
          ttl:
            override.nats?.permissions?.allow_responses?.ttl ??
            base.nats.permissions.allow_responses?.ttl,
        },
      },
      advanced: {
        ...base.nats.advanced,
        ...override.nats?.advanced,
      },
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
