import { afterEach, describe, expect, test } from "bun:test";
import { connect } from "nats";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  RuntimeSubstrate,
  TinkabotRuntimeError,
  isRuntimeErrorKind,
  type RuntimeSubstrateOptions,
} from "../../src/nats-script-runtime/index";

const started: RuntimeSubstrate[] = [];
const tempDirs: string[] = [];

async function tempStoreDir(): Promise<string> {
  const dir = await mkdtemp(join(tmpdir(), "tinkabot-substrate-"));
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

describe("RuntimeSubstrate", () => {
  test("T01 starts embedded JetStream, exposes KV, and stops rerun-safely", async () => {
    const substrate = await RuntimeSubstrate.start({
      storeDir: await tempStoreDir(),
    });
    started.push(substrate);

    expect(substrate.url).toMatch(/^nats:\/\/127\.0\.0\.1:\d+$/);
    expect(substrate.port).toBeGreaterThan(0);

    const kv = await substrate.openKvBucket("TB_SUBSTRATE_PROOF", {
      history: 2,
    });
    const revision = await kv.put(
      "runtime.ready",
      new TextEncoder().encode("ok"),
    );
    const entry = await kv.get("runtime.ready", { revision });

    expect(entry?.string()).toBe("ok");
    expect(entry?.revision).toBe(revision);

    const url = substrate.url;
    await substrate.stop();
    await substrate.stop();
    started.pop();

    await expect(connect({ servers: url, timeout: 100 })).rejects.toThrow();
  });

  test("T01 maps startup, JetStream availability, and cleanup failures", async () => {
    const startupFailure = await captureRuntimeError(() =>
      RuntimeSubstrate.start({
        storeDir: "/tmp/not-used",
        serverFactory: async () => {
          throw new Error("server refused to start");
        },
      }),
    );

    expect(startupFailure.kind).toBe("SubstrateStartupFailed");
    expect(startupFailure.origin).toMatchObject({
      layer: "RuntimeSubstrate",
      operation: "start",
    });

    const unavailable = await RuntimeSubstrate.start(
      fakeSubstrateOptions({
        kvError: new Error("jetstream unavailable"),
      }),
    );

    const kvFailure = await captureRuntimeError(() =>
      unavailable.openKvBucket("TB_UNAVAILABLE"),
    );

    expect(kvFailure.kind).toBe("SubstrateUnavailable");
    expect(kvFailure.origin).toMatchObject({
      layer: "RuntimeSubstrate",
      operation: "openKvBucket",
    });

    const cleanupFailure = await RuntimeSubstrate.start(
      fakeSubstrateOptions({
        stopError: new Error("stop failed"),
      }),
    );

    const stopFailure = await captureRuntimeError(() => cleanupFailure.stop());

    expect(stopFailure.kind).toBe("SubstrateCleanupFailed");
    expect(stopFailure.origin).toMatchObject({
      layer: "RuntimeSubstrate",
      operation: "stop",
    });
  });

  test("T02 wraps unknown lifecycle exceptions as SubstrateCritical", async () => {
    const error = await captureRuntimeError(() =>
      RuntimeSubstrate.start({
        storeDir: "/tmp/not-used",
        serverFactory: async () => {
          throw "non-error startup failure";
        },
      }),
    );

    expect(error.kind).toBe("SubstrateCritical");
    expect(error.origin).toMatchObject({
      layer: "RuntimeSubstrate",
      operation: "start",
    });
    expect(error.causeValue).toBe("non-error startup failure");
  });
});

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

function fakeSubstrateOptions(args: {
  kvError?: Error;
  stopError?: Error;
}): RuntimeSubstrateOptions {
  return {
    storeDir: "/tmp/not-used",
    serverFactory: async () => ({
      url: "nats://127.0.0.1:1",
      port: 1,
      stop: async () => {
        if (args.stopError) throw args.stopError;
      },
    }),
    connectFactory: async () => ({
      jetstream: () => ({
        views: {
          kv: async () => {
            if (args.kvError) throw args.kvError;
            throw new Error("fake kv should not be used for success");
          },
        },
      }),
      close: async () => undefined,
      drain: async () => undefined,
    }),
  };
}

test("runtime error type guard narrows by kind", () => {
  const error = new TinkabotRuntimeError("SubstrateUnavailable", "unavailable", {
    origin: { layer: "RuntimeSubstrate", operation: "openKvBucket" },
  });

  expect(isRuntimeErrorKind(error, "SubstrateUnavailable")).toBe(true);
  expect(isRuntimeErrorKind(error, "SubstrateStartupFailed")).toBe(false);
});
