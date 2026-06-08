import {
  hasPlaceholderSubject,
  isValidSubjectPattern,
  matchSubject,
} from "../nats-script-runtime/subjects";
import {
  TinkabotRuntimeError,
  type RuntimeErrorKind,
  type RuntimeErrorLayer,
  type RuntimeErrorOrigin,
} from "../nats-script-runtime/errors";
import type { Contract } from "./index";

export type AuthPolicy = Extract<Contract, { kind: "auth.policy" }>;

export interface SubjectRule {
  prefix: string;
  plane: "app" | "control";
  owner:
    | "browser"
    | "script"
    | "activation"
    | "materializer"
    | "artifact"
    | "managed-auth"
    | "system";
  reserved?: boolean;
}

export interface CompiledAuth {
  provenance: AuthPolicy["provenance"];
  capability: AuthPolicy["capability"];
  permissions: AuthPolicy["permissions"];
  imports: NonNullable<AuthPolicy["imports"]>;
  exports: NonNullable<AuthPolicy["exports"]>;
  exposure: NonNullable<AuthPolicy["exposure"]>;
  subjects: {
    publish: string[];
    subscribe: string[];
    imports: string[];
    exports: string[];
    exposure: string[];
  };
}

export interface CompileAuthOptions {
  rules?: readonly SubjectRule[];
}

export const endgameSubjects = [
  {
    prefix: "tb.internal.",
    plane: "control",
    owner: "system",
    reserved: true,
  },
  {
    prefix: "tb.control.",
    plane: "control",
    owner: "managed-auth",
    reserved: true,
  },
  { prefix: "tb.proof.out.", plane: "app", owner: "script" },
  { prefix: "tb.proof.runtime.", plane: "app", owner: "activation" },
  { prefix: "tb.proof.reply.", plane: "app", owner: "activation" },
  { prefix: "tb.proof.exec.", plane: "app", owner: "script" },
  { prefix: "tb.proof.material.", plane: "app", owner: "materializer" },
  { prefix: "tb.proof.artifact.", plane: "app", owner: "artifact" },
] as const satisfies readonly SubjectRule[];

export function compileAuth(
  policy: AuthPolicy,
  opts: CompileAuthOptions = {},
): CompiledAuth {
  try {
    const rules = opts.rules ?? endgameSubjects;
    assertProvenance(policy);

    if (policy.capability.leaseStatus === "revoked") {
      throw authError(
        "RevokedLease",
        "compileAuth",
        "Capability lease is revoked",
        ctx(policy),
      );
    }
    if (policy.capability.leaseStatus === "expired") {
      throw authError(
        "ExpiredLease",
        "compileAuth",
        "Capability lease is expired",
        ctx(policy),
      );
    }

    const responses = policy.permissions.allow_responses;
    if (responses && responses.expiresMs == null) {
      throw authError(
        "ResponseAuthorityUnbounded",
        "compileAuth",
        "Response authority requires an expiration bound",
        ctx(policy),
      );
    }

    const imports = policy.imports ?? {};
    const exports = policy.exports ?? [];
    const exposure = policy.exposure ?? {};
    const publish = [
      ...(policy.permissions.publish?.allow ?? []),
      ...(policy.permissions.publish?.deny ?? []),
    ];
    const subscribe = [
      ...(policy.permissions.subscribe?.allow ?? []),
      ...(policy.permissions.subscribe?.deny ?? []),
    ];
    const importSubjects = Object.values(imports).flatMap((item) => {
      if (item.kind === "raw_nats" || item.kind === "cli") {
        throw authError(
          "AdvancedCapabilityDenied",
          "compileAuth",
          `Advanced import is denied by default: ${item.kind}`,
          ctx(policy),
        );
      }
      return item.subjects ?? [];
    });
    const exposureSubjects = Object.entries(exposure).map(([name, item]) => {
      if (item.kind !== "request_reply") {
        throw authError(
          "AdvancedCapabilityDenied",
          "compileAuth",
          `Advanced exposure is denied by default: ${item.kind}`,
          { ...ctx(policy), exposure: name, exposureKind: item.kind },
        );
      }
      if (!item.subject) {
        throw subjectError(
          "ExposureSubjectMissing",
          "compileAuth",
          `Exposure subject is required: ${name}`,
          { ...ctx(policy), exposure: name },
        );
      }
      return item.subject;
    });

    checkSubjects(policy.permissions.publish?.allow ?? [], "allow", rules);
    checkSubjects(policy.permissions.subscribe?.allow ?? [], "allow", rules);
    checkSubjects(importSubjects, "allow", rules);
    checkSubjects(exports, "allow", rules);
    checkSubjects(exposureSubjects, "allow", rules);
    checkSubjects(policy.permissions.publish?.deny ?? [], "deny", rules);
    checkSubjects(policy.permissions.subscribe?.deny ?? [], "deny", rules);
    checkExportExposurePair(policy, exports, exposureSubjects);

    for (const item of Object.values(imports)) {
      if (item.kind === "publish") {
        for (const subject of item.subjects ?? []) {
          assertAllowed(policy, "publish", subject, "compileAuth");
        }
      }
      if (item.kind === "subscribe") {
        for (const subject of item.subjects ?? []) {
          assertAllowed(policy, "subscribe", subject, "compileAuth");
        }
      }
    }
    for (const subject of [...exports, ...exposureSubjects]) {
      assertAllowed(policy, "subscribe", subject, "compileAuth");
    }

    return {
      provenance: policy.provenance,
      capability: policy.capability,
      permissions: policy.permissions,
      imports,
      exports,
      exposure,
      subjects: {
        publish,
        subscribe,
        imports: importSubjects,
        exports,
        exposure: exposureSubjects,
      },
    };
  } catch (error) {
    if (error instanceof TinkabotRuntimeError) throw error;
    throw authCritical("compileAuth", error);
  }
}

export function classifySubject(
  subject: string,
  rules: readonly SubjectRule[] = endgameSubjects,
): SubjectRule {
  try {
    validateSubject(subject);
    const found = [...rules]
      .sort((a, b) => b.prefix.length - a.prefix.length)
      .find((rule) => subject.startsWith(rule.prefix));
    if (!found) {
      throw subjectError(
        "SubjectNamespaceCollision",
        "classifySubject",
        `Subject does not belong to a known namespace: ${subject}`,
        { subject },
      );
    }
    return found;
  } catch (error) {
    if (error instanceof TinkabotRuntimeError) throw error;
    throw subjectCritical("classifySubject", error);
  }
}

export function assertPublish(auth: CompiledAuth, subject: string): void {
  assertAuthSubject(auth, "publish", subject, "assertPublish");
}

export function assertSubscribe(auth: CompiledAuth, subject: string): void {
  assertAuthSubject(auth, "subscribe", subject, "assertSubscribe");
}

function assertAuthSubject(
  auth: CompiledAuth,
  kind: "publish" | "subscribe",
  subject: string,
  operation: string,
): void {
  const permissions = auth.permissions[kind];
  const details = { ...ctx(auth), subject };
  if (permissions?.deny?.some((pattern) => matchSubject(pattern, subject))) {
    throw authError(
      "PermissionDeniedByDenyRule",
      operation,
      `${kind} denied by deny rule: ${subject}`,
      details,
    );
  }
  if (!permissions?.allow?.some((pattern) => matchSubject(pattern, subject))) {
    throw authError(
      "PermissionDenied",
      operation,
      `${kind} not allowed: ${subject}`,
      details,
    );
  }
}

function assertAllowed(
  policy: AuthPolicy,
  kind: "publish" | "subscribe",
  subject: string,
  operation: string,
): void {
  const auth = {
    provenance: policy.provenance,
    capability: policy.capability,
    permissions: policy.permissions,
  } as CompiledAuth;
  assertAuthSubject(auth, kind, subject, operation);
}

function checkExportExposurePair(
  policy: AuthPolicy,
  exports: readonly string[],
  exposure: readonly string[],
): void {
  const exported = new Set(exports);
  const exposed = new Set(exposure);
  const missingExports = exposure.filter((subject) => !exported.has(subject));
  const missingExposure = exports.filter((subject) => !exposed.has(subject));
  if (!missingExports.length && !missingExposure.length) return;

  throw subjectError(
    "ImportExportMismatch",
    "compileAuth",
    "Exports and exposure subjects must match",
    { ...ctx(policy), missingExports, missingExposure },
  );
}

function checkSubjects(
  subjects: readonly string[],
  mode: "allow" | "deny",
  rules: readonly SubjectRule[],
): void {
  for (const subject of subjects) {
    const rule = classifySubject(subject, rules);
    if (mode === "allow" && rule.reserved) {
      throw subjectError(
        "SubjectReserved",
        "checkSubjects",
        `Reserved subject cannot be granted: ${subject}`,
        { subject, prefix: rule.prefix },
      );
    }
  }
}

function validateSubject(subject: string): void {
  if (hasPlaceholderSubject(subject) || !isValidSubjectPattern(subject)) {
    throw subjectError(
      "SubjectDeniedNeighbor",
      "validateSubject",
      `Invalid subject pattern: ${subject}`,
      { subject },
    );
  }
  const tokens = subject.split(".");
  if (tokens.at(-1) === ">" && tokens.length < 4) {
    throw subjectError(
      "SubjectWildcardOverreach",
      "validateSubject",
      `Wildcard subject is too broad: ${subject}`,
      { subject },
    );
  }
}

function assertProvenance(policy: AuthPolicy): void {
  if (
    policy.provenance.appRevision !== policy.capability.appRevision ||
    policy.provenance.schemaVersion !== policy.capability.schemaVersion
  ) {
    throw authError(
      "StaleRevision",
      "compileAuth",
      "Policy provenance does not match capability revision",
      ctx(policy),
    );
  }
}

function ctx(policy: Pick<CompiledAuth, "provenance" | "capability">) {
  return {
    principalId: policy.capability.principalId,
    sessionId: policy.capability.sessionId,
    capabilityId: policy.capability.capabilityId,
    leaseId: policy.capability.leaseId,
    leaseStatus: policy.capability.leaseStatus,
    appRevision: policy.capability.appRevision,
    schemaVersion: policy.capability.schemaVersion,
    provenanceAppRevision: policy.provenance.appRevision,
  };
}

function authError(
  kind:
    | "RevokedLease"
    | "ExpiredLease"
    | "StaleRevision"
    | "ResponseAuthorityUnbounded"
    | "AdvancedCapabilityDenied"
    | "PermissionDenied"
    | "PermissionDeniedByDenyRule",
  operation: string,
  message: string,
  details?: Record<string, unknown>,
): TinkabotRuntimeError {
  return new TinkabotRuntimeError(kind, message, {
    origin: origin("ManagedAuth", operation, details),
  });
}

function subjectError(
  kind:
    | "SubjectReserved"
    | "SubjectWildcardOverreach"
    | "SubjectNamespaceCollision"
    | "ImportExportMismatch"
    | "ExposureSubjectMissing"
    | "SubjectDeniedNeighbor",
  operation: string,
  message: string,
  details?: Record<string, unknown>,
): TinkabotRuntimeError {
  return new TinkabotRuntimeError(kind, message, {
    origin: origin("SubjectTaxonomy", operation, details),
  });
}

function authCritical(operation: string, cause: unknown): TinkabotRuntimeError {
  return new TinkabotRuntimeError(
    "ManagedAuthCritical",
    "Managed auth failed with an unknown error",
    { origin: origin("ManagedAuth", operation), cause },
  );
}

function subjectCritical(
  operation: string,
  cause: unknown,
): TinkabotRuntimeError {
  return new TinkabotRuntimeError(
    "SubjectTaxonomyCritical",
    "Subject taxonomy failed with an unknown error",
    { origin: origin("SubjectTaxonomy", operation), cause },
  );
}

function origin(
  layer: RuntimeErrorLayer,
  operation: string,
  details?: Record<string, unknown>,
): RuntimeErrorOrigin {
  return { layer, operation, details };
}
