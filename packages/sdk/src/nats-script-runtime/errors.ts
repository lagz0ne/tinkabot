export type RuntimeErrorLayer =
  | "ContractAuthority"
  | "ManagedAuth"
  | "SubjectTaxonomy"
  | "RuntimeSubstrate"
  | "ScriptRecordStore"
  | "MetadataSchema"
  | "ImportsPermissions"
  | "Activation"
  | "FrontendMediator"
  | "FramedStdioRpc"
  | "ProcessRuntime"
  | "RuntimeMediation"
  | "AttributionEventTrail"
  | "ExecutionExchange";

export type RuntimeErrorKind =
  | "ContractInvalid"
  | "ContractCritical"
  | "RevokedLease"
  | "ExpiredLease"
  | "StaleRevision"
  | "ResponseAuthorityUnbounded"
  | "ManagedAuthCritical"
  | "SubjectReserved"
  | "SubjectWildcardOverreach"
  | "SubjectNamespaceCollision"
  | "ImportExportMismatch"
  | "ExposureSubjectMissing"
  | "SubjectDeniedNeighbor"
  | "SubjectTaxonomyCritical"
  | "SubstrateStartupFailed"
  | "SubstrateUnavailable"
  | "SubstrateCleanupFailed"
  | "SubstrateCritical"
  | "RecordNotFound"
  | "RecordRevisionMismatch"
  | "RecordDeletedOrStale"
  | "RecordWriteConflict"
  | "RecordPersistenceFailed"
  | "RecordCritical"
  | "MetadataInvalid"
  | "PlaceholderSubjectRejected"
  | "WildcardPatternInvalid"
  | "SchemaReferenceMissing"
  | "SchemaMismatch"
  | "SecurityPostureUnsupported"
  | "MetadataCritical"
  | "ImportNotDeclared"
  | "PermissionDenied"
  | "PermissionDeniedByDenyRule"
  | "ResponseAuthorityExceeded"
  | "AdvancedCapabilityDenied"
  | "PermissionCritical"
  | "ActivationConfigInvalid"
  | "ActivationUnauthorized"
  | "ActivationSourceUnavailable"
  | "ActivationCursorFailed"
  | "ActivationLedgerFailed"
  | "ActivationDedupeConflict"
  | "ActivationAckFailed"
  | "ActivationLoopSuppressed"
  | "ActivationScheduleLeaseFailed"
  | "ActivationCritical"
  | "FrontendMessageInvalid"
  | "FrontendCapabilityDenied"
  | "FrontendBridgeUnavailable"
  | "FrontendCritical";

export interface RuntimeErrorOrigin {
  layer: RuntimeErrorLayer;
  operation: string;
  details?: Record<string, unknown>;
}

export interface RuntimeErrorOptions {
  origin: RuntimeErrorOrigin;
  cause?: unknown;
}

export class TinkabotRuntimeError extends Error {
  readonly kind: RuntimeErrorKind;
  readonly origin: RuntimeErrorOrigin;
  readonly causeValue?: unknown;

  constructor(
    kind: RuntimeErrorKind,
    message: string,
    options: RuntimeErrorOptions,
  ) {
    super(message, { cause: options.cause });
    this.name = "TinkabotRuntimeError";
    this.kind = kind;
    this.origin = options.origin;
    this.causeValue = options.cause;
  }
}

export function isRuntimeErrorKind<K extends RuntimeErrorKind>(
  error: unknown,
  kind: K,
): error is TinkabotRuntimeError & { kind: K } {
  return error instanceof TinkabotRuntimeError && error.kind === kind;
}

export function isSubstrateError(error: unknown): error is TinkabotRuntimeError {
  return (
    error instanceof TinkabotRuntimeError &&
    error.origin.layer === "RuntimeSubstrate"
  );
}

export function errorMessage(error: unknown): string {
  if (error instanceof Error && error.message) return error.message;
  if (typeof error === "string") return error;
  return "unknown error";
}
