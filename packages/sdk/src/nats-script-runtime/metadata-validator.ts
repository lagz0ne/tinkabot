import {
  TinkabotRuntimeError,
  type RuntimeErrorOrigin,
} from "./errors";
import {
  hasPlaceholderSubject,
  isValidSubjectPattern,
} from "./subjects";

export type ScriptTrust = "trusted" | "untrusted";

export interface ScriptMetadata {
  id: string;
  desc?: string;
  runtime: {
    language: "typescript";
    runner?: "bun";
    sandbox: "none" | string;
  };
  security?: {
    trust: ScriptTrust;
    directNats?: boolean;
  };
  schemas: {
    input: string;
    output: string;
    event?: string;
    ipc?: {
      progress?: string;
      publishRequest?: string;
      publish?: string;
    };
  };
  nats: {
    io?: {
      input?: {
        subject: string;
      };
      output?: {
        subject: string;
      };
    };
    activations?: Record<string, ScriptActivation>;
    imports: Record<string, ScriptImport>;
    permissions: NatsPermissions;
    advanced?: {
      rawNats?: boolean;
      cli?: boolean;
    };
  };
}

export type ScriptActivation =
  | {
      kind: "request_reply";
      subject: string;
      desc?: string;
      exposure?: {
        desc?: string;
      };
    };

export type ScriptImport =
  | {
      kind: "publish";
      subject?: string;
      subjects?: string[];
      desc?: string;
    }
  | {
      kind: "subscribe";
      subject?: string;
      subjects?: string[];
      desc?: string;
    }
  | {
      kind: "raw_nats";
      desc?: string;
    }
  | {
      kind: "cli";
      desc?: string;
    };

export interface NatsPermissions {
  publish?: NatsPermissionList;
  subscribe?: NatsPermissionList;
  allow_responses?: {
    max: number;
    expiresMs?: number;
    ttl?: string;
  };
}

export interface NatsPermissionList {
  allow?: string[];
  deny?: string[];
}

export interface MetadataValidatorOptions {
  schemaIds?: ReadonlySet<string>;
  inspectSchemaRef?: (
    ref: string,
    surface: string,
    metadata: ScriptMetadata,
  ) => void;
}

export class MetadataValidator {
  static validate(
    metadata: ScriptMetadata,
    options: MetadataValidatorOptions = {},
  ): ScriptMetadata {
    try {
      validateShape(metadata);
      validateSecurity(metadata);
      validateSchemas(metadata, options);
      validateSubjects(metadata);
      validateActivations(metadata);
      return metadata;
    } catch (error) {
      if (error instanceof TinkabotRuntimeError) throw error;
      throw critical("validate", error);
    }
  }
}

function validateShape(metadata: ScriptMetadata): void {
  if (!metadata || typeof metadata !== "object") {
    throw metadataError("MetadataInvalid", "Script metadata must be an object");
  }
  if (!metadata.id || typeof metadata.id !== "string") {
    throw metadataError("MetadataInvalid", "Script metadata id is required");
  }
  if (!metadata.runtime || metadata.runtime.language !== "typescript") {
    throw metadataError("MetadataInvalid", "TypeScript runtime metadata is required");
  }
  if (!metadata.schemas || typeof metadata.schemas !== "object") {
    throw metadataError("MetadataInvalid", "Schema metadata is required");
  }
  if (!metadata.nats?.permissions || !metadata.nats.imports) {
    throw metadataError("MetadataInvalid", "NATS permissions and imports are required");
  }
}

function validateSecurity(metadata: ScriptMetadata): void {
  if (metadata.runtime.sandbox !== "none") {
    throw metadataError(
      "SecurityPostureUnsupported",
      "Only trusted sandbox:none execution is currently supported",
    );
  }
  if (metadata.security?.trust && metadata.security.trust !== "trusted") {
    throw metadataError(
      "SecurityPostureUnsupported",
      "Untrusted scripts require sandbox enforcement before execution",
    );
  }
}

function validateSchemas(
  metadata: ScriptMetadata,
  options: MetadataValidatorOptions,
): void {
  const refs: Array<[surface: string, ref: string | undefined, token: string]> = [
    ["input", metadata.schemas.input, ".input."],
    ["output", metadata.schemas.output, ".output."],
    ["event", metadata.schemas.event, ".event."],
    ["ipc.progress", metadata.schemas.ipc?.progress, ".ipc."],
    [
      "ipc.publishRequest",
      metadata.schemas.ipc?.publishRequest ?? metadata.schemas.ipc?.publish,
      ".ipc.",
    ],
  ];

  for (const [surface, ref, token] of refs) {
    if (!ref) {
      throw metadataError(
        "SchemaReferenceMissing",
        `Missing schema reference for ${surface}`,
      );
    }
    if (options.schemaIds && !options.schemaIds.has(ref)) {
      throw metadataError(
        "SchemaReferenceMissing",
        `Unknown schema reference for ${surface}: ${ref}`,
      );
    }
    if (!ref.includes(token)) {
      throw metadataError(
        "SchemaMismatch",
        `Schema reference ${ref} does not match ${surface}`,
      );
    }
    options.inspectSchemaRef?.(ref, surface, metadata);
  }
}

function validateActivations(metadata: ScriptMetadata): void {
  const activations = metadata.nats.activations;
  if (activations === undefined) return;
  if (!activations || typeof activations !== "object") {
    throw metadataError("MetadataInvalid", "NATS activations must be an object");
  }

  for (const [name, activation] of Object.entries(activations)) {
    if (!activation || typeof activation !== "object") {
      throw metadataError(
        "MetadataInvalid",
        `Activation declaration must be an object: ${name}`,
      );
    }
    if (activation.kind !== "request_reply") {
      throw metadataError(
        "MetadataInvalid",
        `Unsupported activation kind for ${name}`,
      );
    }
    if (!activation.subject || typeof activation.subject !== "string") {
      throw metadataError(
        "MetadataInvalid",
        `Request/reply activation subject is required: ${name}`,
      );
    }
    if (!isConcreteSubject(activation.subject)) {
      throw metadataError(
        "WildcardPatternInvalid",
        `Request/reply activation subject must be concrete: ${activation.subject}`,
      );
    }
  }
}

function validateSubjects(metadata: ScriptMetadata): void {
  for (const subject of collectSubjects(metadata)) {
    if (hasPlaceholderSubject(subject)) {
      throw metadataError(
        "PlaceholderSubjectRejected",
        `Placeholder subject is not allowed: ${subject}`,
      );
    }
    if (!isValidSubjectPattern(subject)) {
      throw metadataError(
        "WildcardPatternInvalid",
        `Invalid NATS subject pattern: ${subject}`,
      );
    }
  }
}

function collectSubjects(metadata: ScriptMetadata): string[] {
  const subjects: string[] = [];
  const { nats } = metadata;

  if (nats.io?.input?.subject) subjects.push(nats.io.input.subject);
  if (nats.io?.output?.subject) subjects.push(nats.io.output.subject);

  for (const permission of [nats.permissions.publish, nats.permissions.subscribe]) {
    subjects.push(...(permission?.allow ?? []), ...(permission?.deny ?? []));
  }

  for (const item of Object.values(nats.imports)) {
    if ("subject" in item && item.subject) subjects.push(item.subject);
    if ("subjects" in item && item.subjects) subjects.push(...item.subjects);
  }

  for (const item of Object.values(nats.activations ?? {})) {
    if ("subject" in item && item.subject) subjects.push(item.subject);
  }

  return subjects;
}

function isConcreteSubject(subject: string): boolean {
  return (
    isValidSubjectPattern(subject) &&
    !subject.split(".").some((token) => token === "*" || token === ">")
  );
}

function metadataError(
  kind:
    | "MetadataInvalid"
    | "PlaceholderSubjectRejected"
    | "WildcardPatternInvalid"
    | "SchemaReferenceMissing"
    | "SchemaMismatch"
    | "SecurityPostureUnsupported",
  message: string,
): TinkabotRuntimeError {
  return new TinkabotRuntimeError(kind, message, {
    origin: metadataOrigin("validate"),
  });
}

function critical(operation: string, cause: unknown): TinkabotRuntimeError {
  return new TinkabotRuntimeError(
    "MetadataCritical",
    "Metadata validation failed with an unknown error",
    {
      origin: metadataOrigin(operation),
      cause,
    },
  );
}

function metadataOrigin(operation: string): RuntimeErrorOrigin {
  return {
    layer: "MetadataSchema",
    operation,
  };
}
