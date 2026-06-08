import { describe, expect, test } from "bun:test";
import {
  MetadataValidator,
  TinkabotRuntimeError,
  type RuntimeErrorKind,
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

describe("MetadataValidator", () => {
  test("T05 accepts concrete NATS-focused metadata and treats desc as non-authority", () => {
    const metadata = validMetadata();
    metadata.desc = "Publishes example output containing tb.proof.out.<scriptId> text only.";

    const validated = MetadataValidator.validate(metadata, schemaOptions());

    expect(validated.id).toBe("script.proof.echo");
    expect(validated.desc).toContain("<scriptId>");
    expect(validated.nats.permissions.publish?.allow).toContain(
      "tb.proof.out.allowed.>",
    );
  });

  test("T05 rejects placeholder subjects, invalid wildcards, schema gaps, schema mismatch, and unsupported posture", async () => {
    const cases: Array<{
      name: string;
      metadata: ScriptMetadata;
      kind: RuntimeErrorKind;
    }> = [
      {
        name: "malformed metadata root",
        metadata: validMetadata({
          id: "",
        }),
        kind: "MetadataInvalid",
      },
      {
        name: "placeholder subject",
        metadata: validMetadata({
          nats: {
            permissions: {
              publish: {
                allow: ["tb.proof.out.<scriptId>"],
              },
            },
          },
        }),
        kind: "PlaceholderSubjectRejected",
      },
      {
        name: "terminal wildcard in the middle",
        metadata: validMetadata({
          nats: {
            permissions: {
              publish: {
                allow: ["tb.proof.bad.>.tail"],
              },
            },
          },
        }),
        kind: "WildcardPatternInvalid",
      },
      {
        name: "wildcard before authority prefix",
        metadata: validMetadata({
          nats: {
            permissions: {
              publish: {
                allow: ["*.proof.out"],
              },
            },
          },
        }),
        kind: "WildcardPatternInvalid",
      },
      {
        name: "unregistered schema reference",
        metadata: validMetadata({
          schemas: {
            input: "tb.schema.proof.input.echo.v1",
            output: "tb.schema.proof.output.missing.v1",
          },
        }),
        kind: "SchemaReferenceMissing",
      },
      {
        name: "schema category mismatch",
        metadata: validMetadata({
          schemas: {
            input: "tb.schema.proof.output.echo.v1",
            output: "tb.schema.proof.output.echo.v1",
          },
        }),
        kind: "SchemaMismatch",
      },
      {
        name: "unsupported sandbox posture",
        metadata: validMetadata({
          security: {
            trust: "untrusted",
            directNats: false,
          },
        }),
        kind: "SecurityPostureUnsupported",
      },
    ];

    for (const item of cases) {
      const error = await captureRuntimeError(() =>
        Promise.resolve(MetadataValidator.validate(item.metadata, schemaOptions())),
      );

      expect(error.kind).toBe(item.kind);
      expect(error.origin.layer).toBe("MetadataSchema");
      expect(error.origin.operation).toBe("validate");
    }
  });

  test("T06 wraps unknown metadata validation exceptions as MetadataCritical", async () => {
    const error = await captureRuntimeError(() =>
      Promise.resolve(
        MetadataValidator.validate(validMetadata(), {
          ...schemaOptions(),
          inspectSchemaRef: () => {
            throw "schema inspector exploded";
          },
        }),
      ),
    );

    expect(error.kind).toBe("MetadataCritical");
    expect(error.origin).toMatchObject({
      layer: "MetadataSchema",
      operation: "validate",
    });
    expect(error.causeValue).toBe("schema inspector exploded");
  });

  test("T10-ACT-META accepts request/reply activation exposure and rejects malformed activation", async () => {
    const accepted = MetadataValidator.validate(
      validMetadata({
        nats: {
          activations: {
            request: {
              kind: "request_reply",
              subject: "tb.proof.runtime.execute",
              desc: "Outside-in execution request subject.",
              exposure: {
                desc: "Caller-facing request/reply activation.",
              },
            },
          },
        } as Partial<ScriptMetadata["nats"]>,
      }),
      schemaOptions(),
    );

    expect(accepted.nats.activations?.request?.kind).toBe("request_reply");
    expect(accepted.nats.activations?.request?.exposure?.desc).toContain(
      "request/reply",
    );
    const cases: Array<{
      name: string;
      metadata: ScriptMetadata;
      kind: RuntimeErrorKind;
    }> = [
      {
        name: "missing activation kind",
        metadata: validMetadata({
          nats: {
            activations: {
              request: {
                subject: "tb.proof.runtime.execute",
              },
            },
          } as unknown as Partial<ScriptMetadata["nats"]>,
        }),
        kind: "MetadataInvalid",
      },
      {
        name: "request/reply activation missing subject",
        metadata: validMetadata({
          nats: {
            activations: {
              request: {
                kind: "request_reply",
              },
            },
          } as unknown as Partial<ScriptMetadata["nats"]>,
        }),
        kind: "MetadataInvalid",
      },
      {
        name: "request/reply activation uses wildcard subject",
        metadata: validMetadata({
          nats: {
            activations: {
              request: {
                kind: "request_reply",
                subject: "tb.proof.runtime.*",
              },
            },
          } as Partial<ScriptMetadata["nats"]>,
        }),
        kind: "WildcardPatternInvalid",
      },
    ];

    for (const item of cases) {
      const error = await captureRuntimeError(() =>
        Promise.resolve(MetadataValidator.validate(item.metadata, schemaOptions())),
      );

      expect(error.kind).toBe(item.kind);
      expect(error.origin.layer).toBe("MetadataSchema");
      expect(error.origin.operation).toBe("validate");
    }
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
        out: {
          kind: "publish",
          subject: "tb.proof.out.allowed.exec_success_001",
          desc: "Allowed proof output subject.",
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
