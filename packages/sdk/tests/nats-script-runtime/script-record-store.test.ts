import { afterEach, describe, expect, test } from "bun:test";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  RuntimeSubstrate,
  ScriptRecordStore,
  TinkabotRuntimeError,
  type ScriptRecord,
} from "../../src/nats-script-runtime/index";

const started: RuntimeSubstrate[] = [];
const tempDirs: string[] = [];

async function tempStoreDir(): Promise<string> {
  const dir = await mkdtemp(join(tmpdir(), "tinkabot-records-"));
  tempDirs.push(dir);
  return dir;
}

afterEach(async () => {
  while (started.length > 0) {
    const substrate = started.pop();
    await substrate?.stop().catch(() => undefined);
  }
  while (tempDirs.length > 0) {
    const dir = tempDirs.pop();
    if (dir) await rm(dir, { recursive: true, force: true });
  }
});

describe("ScriptRecordStore", () => {
  test("T03 stores and reads exact KV revisions", async () => {
    const substrate = await RuntimeSubstrate.start({
      storeDir: await tempStoreDir(),
    });
    started.push(substrate);
    const store = await ScriptRecordStore.open(substrate, {
      bucket: "TB_SCRIPT_RECORDS_UNIT_REVISIONS",
      history: 3,
    });

    const revision1 = await store.create("scripts.proof.echo", record("rev1"));
    const revision2 = await store.update(
      "scripts.proof.echo",
      record("rev2"),
      { previousRevision: revision1 },
    );

    const first = await store.get("scripts.proof.echo", {
      revision: revision1,
    });
    const second = await store.get("scripts.proof.echo", {
      revision: revision2,
    });

    expect(first.revision).toBe(revision1);
    expect(first.record.source).toBe("export default 'rev1';");
    expect(second.revision).toBe(revision2);
    expect(second.record.source).toBe("export default 'rev2';");
  });

  test("T03 maps missing, deleted, revision mismatch, and write conflict", async () => {
    const substrate = await RuntimeSubstrate.start({
      storeDir: await tempStoreDir(),
    });
    started.push(substrate);
    const store = await ScriptRecordStore.open(substrate, {
      bucket: "TB_SCRIPT_RECORDS_UNIT_STATES",
      history: 3,
    });

    const missing = await captureRuntimeError(() =>
      store.get("scripts.proof.missing"),
    );
    expect(missing.kind).toBe("RecordNotFound");
    expect(missing.origin).toMatchObject({
      layer: "ScriptRecordStore",
      operation: "get",
    });

    const revision1 = await store.create(
      "scripts.proof.deleted",
      record("rev1"),
    );
    await store.delete("scripts.proof.deleted", {
      previousRevision: revision1,
    });

    const deleted = await captureRuntimeError(() =>
      store.get("scripts.proof.deleted"),
    );
    expect(deleted.kind).toBe("RecordDeletedOrStale");

    await store.create("scripts.proof.revision", record("rev1"));
    const mismatch = await captureRuntimeError(() =>
      store.get("scripts.proof.revision", { revision: 999_999 }),
    );
    expect(mismatch.kind).toBe("RecordRevisionMismatch");

    const conflictRev1 = await store.create(
      "scripts.proof.conflict",
      record("rev1"),
    );
    await store.update("scripts.proof.conflict", record("rev2"), {
      previousRevision: conflictRev1,
    });

    const conflict = await captureRuntimeError(() =>
      store.update("scripts.proof.conflict", record("rev3"), {
        previousRevision: conflictRev1,
      }),
    );
    expect(conflict.kind).toBe("RecordWriteConflict");
  });

  test("T04 transforms substrate failures during KV access", async () => {
    const substrateFailure = new TinkabotRuntimeError(
      "SubstrateUnavailable",
      "jetstream unavailable",
      {
        origin: {
          layer: "RuntimeSubstrate",
          operation: "openKvBucket",
        },
      },
    );

    const openFailure = await captureRuntimeError(() =>
      ScriptRecordStore.open(
        {
          openKvBucket: async () => {
            throw substrateFailure;
          },
        },
        {
          bucket: "TB_SCRIPT_RECORDS_UNIT_FAILURES",
          history: 2,
        },
      ),
    );

    expect(openFailure.kind).toBe("RecordPersistenceFailed");
    expect(openFailure.origin).toMatchObject({
      layer: "ScriptRecordStore",
      operation: "open",
    });
    expect(openFailure.cause).toBe(substrateFailure);
  });
});

function record(label: string): ScriptRecord {
  return {
    source: `export default '${label}';`,
    metadata: {
      id: `script-${label}`,
      desc: `Record fixture ${label}`,
      runtime: {
        language: "typescript",
        sandbox: "none",
      },
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
