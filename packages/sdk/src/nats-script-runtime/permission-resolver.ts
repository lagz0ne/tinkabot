import {
  MetadataValidator,
  type MetadataValidatorOptions,
  type ScriptActivation,
  type ScriptImport,
  type ScriptMetadata,
} from "./metadata-validator";
import {
  TinkabotRuntimeError,
  type RuntimeErrorOrigin,
} from "./errors";
import { matchSubject } from "./subjects";

export interface PermissionResolverOptions extends MetadataValidatorOptions {
  subjectMatcher?: (pattern: string, subject: string) => boolean;
  validateMetadata?: (
    metadata: ScriptMetadata,
    options: MetadataValidatorOptions,
  ) => ScriptMetadata;
}

export interface ResponsePublishContext {
  replySubject: string;
  usedResponses: number;
  now?: number;
  expiresAt?: number;
}

export class PermissionResolver {
  private constructor(
    private readonly metadata: ScriptMetadata,
    private readonly matcher: (pattern: string, subject: string) => boolean,
  ) {}

  static fromMetadata(
    metadata: ScriptMetadata,
    options: PermissionResolverOptions = {},
  ): PermissionResolver {
    try {
      const validate = options.validateMetadata ?? MetadataValidator.validate;
      const validated = validate(metadata, options);
      return new PermissionResolver(
        validated,
        options.subjectMatcher ?? matchSubject,
      );
    } catch (error) {
      if (
        error instanceof TinkabotRuntimeError &&
        error.origin.layer === "MetadataSchema"
      ) {
        throw error;
      }
      throw critical("fromMetadata", error);
    }
  }

  requireImport(name: string): ScriptImport {
    try {
      const found = this.metadata.nats.imports[name];
      if (!found) {
        throw permissionError(
          "ImportNotDeclared",
          "requireImport",
          `Import is not declared: ${name}`,
          { name },
        );
      }
      if (found.kind === "raw_nats" && !this.metadata.nats.advanced?.rawNats) {
        throw permissionError(
          "AdvancedCapabilityDenied",
          "requireImport",
          "Raw NATS import requires explicit advanced capability",
          { name },
        );
      }
      if (found.kind === "cli" && !this.metadata.nats.advanced?.cli) {
        throw permissionError(
          "AdvancedCapabilityDenied",
          "requireImport",
          "CLI import requires explicit advanced capability",
          { name },
        );
      }
      return found;
    } catch (error) {
      throw mapPermissionBoundary(error, "requireImport");
    }
  }

  assertImportPublish(importName: string, subject: string): void {
    try {
      const item = this.requireImport(importName);
      if (item.kind !== "publish") {
        throw permissionError(
          "PermissionDenied",
          "assertImportPublish",
          `Import is not a publish import: ${importName}`,
          { importName, subject },
        );
      }

      const importSubjects = item.subjects ?? (item.subject ? [item.subject] : []);
      if (!importSubjects.some((pattern) => this.matcher(pattern, subject))) {
        throw permissionError(
          "PermissionDenied",
          "assertImportPublish",
          `Import ${importName} cannot publish ${subject}`,
          { importName, subject },
        );
      }

      this.assertPublish(subject);
    } catch (error) {
      throw mapPermissionBoundary(error, "assertImportPublish");
    }
  }

  assertPublish(subject: string): void {
    try {
      this.assertSubjectPermission(
        this.metadata.nats.permissions.publish,
        subject,
        "assertPublish",
        "Publish",
      );
    } catch (error) {
      throw mapPermissionBoundary(error, "assertPublish");
    }
  }

  assertSubscribe(subject: string): void {
    try {
      this.assertSubjectPermission(
        this.metadata.nats.permissions.subscribe,
        subject,
        "assertSubscribe",
        "Subscribe",
      );
    } catch (error) {
      throw mapPermissionBoundary(error, "assertSubscribe");
    }
  }

  assertActivationSource(activationName: string, subject?: string): void {
    try {
      const activation = this.metadata.nats.activations?.[activationName];
      if (!activation) {
        throw permissionError(
          "PermissionDenied",
          "assertActivationSource",
          `Activation source is not declared: ${activationName}`,
          { activationName },
        );
      }

      const declaredSubject = activationSubjectOf(activation);
      const activationSubject = subject ?? declaredSubject;
      if (!this.matcher(declaredSubject, activationSubject)) {
        throw permissionError(
          "PermissionDenied",
          "assertActivationSource",
          `Activation source subject does not match declaration: ${activationSubject}`,
          { activationName, subject: activationSubject, declaredSubject },
        );
      }
      this.assertSubjectPermission(
        this.metadata.nats.permissions.subscribe,
        activationSubject,
        "assertActivationSource",
        "Subscribe",
        { activationName },
      );
    } catch (error) {
      throw mapPermissionBoundary(error, "assertActivationSource");
    }
  }

  assertResponsePublish(
    subject: string,
    context: ResponsePublishContext,
  ): void {
    try {
      const allowResponses = this.metadata.nats.permissions.allow_responses;
      const now = context.now ?? Date.now();
      const expired =
        context.expiresAt !== undefined && now > context.expiresAt;

      if (
        !allowResponses ||
        subject !== context.replySubject ||
        context.usedResponses >= allowResponses.max ||
        expired
      ) {
        throw permissionError(
          "ResponseAuthorityExceeded",
          "assertResponsePublish",
          `Response publish authority exceeded: ${subject}`,
          {
            subject,
            replySubject: context.replySubject,
            usedResponses: context.usedResponses,
            expiresAt: context.expiresAt,
          },
        );
      }
    } catch (error) {
      throw mapPermissionBoundary(error, "assertResponsePublish");
    }
  }

  private assertSubjectPermission(
    permissions: { allow?: string[]; deny?: string[] } | undefined,
    subject: string,
    operation: string,
    action: string,
    details: Record<string, unknown> = {},
  ): void {
    const denied = permissions?.deny?.some((pattern) =>
      this.matcher(pattern, subject),
    );
    if (denied) {
      throw permissionError(
        "PermissionDeniedByDenyRule",
        operation,
        `${action} denied by deny rule: ${subject}`,
        { ...details, subject },
      );
    }

    const allowed = permissions?.allow?.some((pattern) =>
      this.matcher(pattern, subject),
    );
    if (!allowed) {
      throw permissionError(
        "PermissionDenied",
        operation,
        `${action} not allowed: ${subject}`,
        { ...details, subject },
      );
    }
  }
}

function activationSubjectOf(activation: ScriptActivation): string {
  return activation.subject;
}

function mapPermissionBoundary(
  error: unknown,
  operation: string,
): TinkabotRuntimeError {
  if (error instanceof TinkabotRuntimeError) return error;
  return critical(operation, error);
}

function permissionError(
  kind:
    | "ImportNotDeclared"
    | "PermissionDenied"
    | "PermissionDeniedByDenyRule"
    | "ResponseAuthorityExceeded"
    | "AdvancedCapabilityDenied",
  operation: string,
  message: string,
  details?: Record<string, unknown>,
): TinkabotRuntimeError {
  return new TinkabotRuntimeError(kind, message, {
    origin: permissionOrigin(operation, details),
  });
}

function critical(operation: string, cause: unknown): TinkabotRuntimeError {
  return new TinkabotRuntimeError(
    "PermissionCritical",
    "Permission resolution failed with an unknown error",
    {
      origin: permissionOrigin(operation),
      cause,
    },
  );
}

function permissionOrigin(
  operation: string,
  details?: Record<string, unknown>,
): RuntimeErrorOrigin {
  return {
    layer: "ImportsPermissions",
    operation,
    details,
  };
}
