import { describe, expect, test } from "bun:test";
import {
  createRequestReplyActivationIntent,
  TinkabotRuntimeError,
} from "../../src/nats-script-runtime/index";

describe("ActivationIntent", () => {
  test("T10-ACT-INTENT builds request/reply intent with reply context and stable dedupe key", () => {
    const input = {
      activationName: "request",
      scriptKey: "scripts.proof.echo",
      scriptRevision: 7,
      subject: "tb.proof.runtime.execute",
      requestId: "req-001",
      payload: { text: "hello" },
      headers: {
        "tb.trace": "trace-001",
      },
      replySubject: "tb.proof.reply.req_001",
      observedAt: "2026-06-05T00:00:00.000Z",
      chain: {
        chainId: "chain-001",
        rootId: "root-001",
        parentId: "parent-001",
        hop: 1,
        maxHops: 5,
      },
    } as const;

    const first = createRequestReplyActivationIntent(input);
    const second = createRequestReplyActivationIntent(input);

    expect(first.source.kind).toBe("request_reply");
    expect(first.source.subject).toBe("tb.proof.runtime.execute");
    expect(first.source.requestId).toBe("req-001");
    expect(first.reply?.subject).toBe("tb.proof.reply.req_001");
    expect(first.chain).toEqual(input.chain);
    expect(first.dedupeKey).toBe(second.dedupeKey);
    expect(first.activationId).toBe(second.activationId);
    expect(first.payload).toEqual({ text: "hello" });
  });

  test("T10-ACT-ERR rejects invalid activation input with activation-owned errors", async () => {
    const missingIdentity = await captureRuntimeError(() =>
      Promise.resolve(
        createRequestReplyActivationIntent({
          activationName: "request",
          scriptKey: "scripts.proof.echo",
          subject: "tb.proof.runtime.execute",
          requestId: "",
          observedAt: "2026-06-05T00:00:00.000Z",
        }),
      ),
    );

    expect(missingIdentity.kind).toBe("ActivationConfigInvalid");
    expect(missingIdentity.origin.layer).toBe("Activation");

    const loop = await captureRuntimeError(() =>
      Promise.resolve(
        createRequestReplyActivationIntent({
          activationName: "request",
          scriptKey: "scripts.proof.echo",
          subject: "tb.proof.runtime.execute",
          requestId: "req-002",
          observedAt: "2026-06-05T00:00:00.000Z",
          chain: {
            chainId: "chain-002",
            rootId: "root-002",
            hop: 6,
            maxHops: 5,
          },
        }),
      ),
    );

    expect(loop.kind).toBe("ActivationLoopSuppressed");
    expect(loop.origin.layer).toBe("Activation");
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
