import type { KV, KvEntry, KvOptions } from "nats";
import {
  TinkabotRuntimeError,
  errorMessage,
  isSubstrateError,
  type RuntimeErrorOrigin,
} from "./errors";

const recordEncoder = new TextEncoder();
const recordDecoder = new TextDecoder();

export interface ScriptRecord {
  source: string;
  metadata: Record<string, unknown>;
}

export interface StoredScriptRecord {
  key: string;
  revision: number;
  record: ScriptRecord;
}

export interface ScriptRecordStoreOptions {
  bucket: string;
  history: number;
}

export interface ScriptRecordGetOptions {
  revision?: number;
}

export interface ScriptRecordWriteOptions {
  previousRevision: number;
}

export interface ScriptRecordSubstrate {
  openKvBucket(name: string, options?: Partial<KvOptions>): Promise<KV>;
}

export class ScriptRecordStore {
  private constructor(private readonly kv: KV) {}

  static async open(
    substrate: ScriptRecordSubstrate,
    options: ScriptRecordStoreOptions,
  ): Promise<ScriptRecordStore> {
    try {
      const kv = await substrate.openKvBucket(options.bucket, {
        history: options.history,
      });
      return new ScriptRecordStore(kv);
    } catch (error) {
      throw mapPersistenceFailure(error, "open");
    }
  }

  async create(key: string, record: ScriptRecord): Promise<number> {
    try {
      return await this.kv.create(key, encodeRecord(record));
    } catch (error) {
      throw mapRecordWriteFailure(error, "create");
    }
  }

  async update(
    key: string,
    record: ScriptRecord,
    options: ScriptRecordWriteOptions,
  ): Promise<number> {
    try {
      return await this.kv.update(
        key,
        encodeRecord(record),
        options.previousRevision,
      );
    } catch (error) {
      throw mapRecordWriteFailure(error, "update");
    }
  }

  async delete(
    key: string,
    options?: ScriptRecordWriteOptions,
  ): Promise<void> {
    try {
      await this.kv.delete(key, {
        previousSeq: options?.previousRevision,
      });
    } catch (error) {
      throw mapRecordWriteFailure(error, "delete");
    }
  }

  async get(
    key: string,
    options: ScriptRecordGetOptions = {},
  ): Promise<StoredScriptRecord> {
    let entry: KvEntry | null;
    try {
      entry = await this.kv.get(
        key,
        options.revision === undefined
          ? undefined
          : { revision: options.revision },
      );
    } catch (error) {
      throw mapPersistenceFailure(error, "get");
    }

    if (!entry) {
      throw new TinkabotRuntimeError(
        options.revision === undefined
          ? "RecordNotFound"
          : "RecordRevisionMismatch",
        options.revision === undefined
          ? `Script record not found: ${key}`
          : `Script record revision not found: ${key}@${options.revision}`,
        { origin: recordOrigin("get", { key, revision: options.revision }) },
      );
    }

    if (entry.operation === "DEL" || entry.operation === "PURGE") {
      throw new TinkabotRuntimeError(
        "RecordDeletedOrStale",
        `Script record is deleted or stale: ${key}@${entry.revision}`,
        { origin: recordOrigin("get", { key, revision: entry.revision }) },
      );
    }

    try {
      return {
        key,
        revision: entry.revision,
        record: decodeRecord(entry),
      };
    } catch (error) {
      throw new TinkabotRuntimeError(
        "RecordCritical",
        `Script record could not be decoded: ${key}@${entry.revision}`,
        {
          origin: recordOrigin("get", { key, revision: entry.revision }),
          cause: error,
        },
      );
    }
  }
}

function encodeRecord(record: ScriptRecord): Uint8Array {
  return recordEncoder.encode(JSON.stringify(record));
}

function decodeRecord(entry: KvEntry): ScriptRecord {
  const decoded = recordDecoder.decode(entry.value);
  return JSON.parse(decoded) as ScriptRecord;
}

function mapPersistenceFailure(
  error: unknown,
  operation: string,
): TinkabotRuntimeError {
  if (error instanceof TinkabotRuntimeError && !isSubstrateError(error)) {
    return error;
  }

  return new TinkabotRuntimeError(
    "RecordPersistenceFailed",
    `Script record persistence failed during ${operation}: ${errorMessage(error)}`,
    { origin: recordOrigin(operation), cause: error },
  );
}

function mapRecordWriteFailure(
  error: unknown,
  operation: string,
): TinkabotRuntimeError {
  if (isSubstrateError(error)) return mapPersistenceFailure(error, operation);
  if (error instanceof TinkabotRuntimeError) return error;

  if (!(error instanceof Error)) {
    return new TinkabotRuntimeError(
      "RecordCritical",
      `Script record ${operation} failed with a non-error value`,
      { origin: recordOrigin(operation), cause: error },
    );
  }

  return new TinkabotRuntimeError(
    "RecordWriteConflict",
    `Script record ${operation} write conflict: ${errorMessage(error)}`,
    { origin: recordOrigin(operation), cause: error },
  );
}

function recordOrigin(
  operation: string,
  details?: Record<string, unknown>,
): RuntimeErrorOrigin {
  return {
    layer: "ScriptRecordStore",
    operation,
    details,
  };
}
