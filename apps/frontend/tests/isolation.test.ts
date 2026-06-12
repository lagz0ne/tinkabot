import { describe, expect, test } from "bun:test";
import {
  FrameError,
  accept,
  checkSandbox,
  denyRaw,
  frameAttrs,
  makeLease,
  mayObserve,
} from "../src/isolation";

describe("frontend isolation", () => {
  test("keeps generated content in script-only opaque sandbox", () => {
    expect(frameAttrs()).toEqual({
      title: "generated artifact",
      sandbox: "allow-scripts",
      referrerPolicy: "no-referrer",
    });

    const unsafe = capture(() => checkSandbox("allow-scripts allow-same-origin"));
    expect(unsafe.kind).toBe("FrameSandboxDenied");

    expect(capture(() => checkSandbox("allow-scripts allow-popups")).kind).toBe(
      "FrameSandboxDenied",
    );
  });

  test("accepts only leased source, nonce, revision, schema, and capability", () => {
    const lease = testLease();
    const src = {};
    const intent = accept(lease, src, src, msg());

    expect(intent).toMatchObject({
      kind: "browser.command_intent",
      type: "content.intent",
      command: "select_artifact",
      commandId: "cmd-001",
      expectedRevision: "artifact.rev.7",
      context: {
        sessionId: "session-001",
        capabilityId: "cap-001",
        artifactId: "artifact-001",
        artifactRevision: "artifact.rev.7",
        frameId: "frame-001",
      },
    });

    expect(capture(() => accept(lease, {}, src, msg())).kind).toBe(
      "FrameLeaseDenied",
    );
    expect(capture(() => accept(lease, src, src, msg({ nonce: "bad" }))).kind).toBe(
      "FrameLeaseDenied",
    );
    expect(
      capture(() => accept(lease, src, src, msg({ artifactRevision: "artifact.rev.6" })))
        .kind,
    ).toBe("FrameLeaseDenied");
    expect(
      capture(() => accept(lease, src, src, msg({ expectedRevision: "artifact.rev.6" })))
        .kind,
    ).toBe("FrameLeaseDenied");
    expect(capture(() => accept(lease, src, src, msg({ command: "publish_raw" }))).kind).toBe(
      "FrameCapabilityDenied",
    );
  });

  test("FrameScopeEscape: steer for session outside observation scope is denied", () => {
    const lease = testLease({ sessions: ["session-001"] });
    const src = {};

    // session in scope — accepted
    const intent = accept(lease, src, src, msg({ command: "steer_session", sessionId: "session-001" }));
    expect(intent.context.sessionId).toBe("session-001");

    // session NOT in scope — denied with FrameScopeEscape
    const denied = capture(() =>
      accept(lease, src, src, msg({ command: "steer_session", sessionId: "session-999" })),
    );
    expect(denied.kind).toBe("FrameScopeEscape");
  });

  test("FrameScopeEscape: content intent with no sessionId passes scope check unchanged", () => {
    const lease = testLease({ sessions: ["session-001"] });
    const src = {};
    // existing content.intent with no sessionId field — unaffected
    const intent = accept(lease, src, src, msg());
    expect(intent.kind).toBe("browser.command_intent");
  });

  test("mayObserve: returns true iff sessionId is in lease.sessions", () => {
    const lease = testLease({ sessions: ["session-001", "session-002"] });
    expect(mayObserve(lease, "session-001")).toBe(true);
    expect(mayObserve(lease, "session-002")).toBe(true);
    expect(mayObserve(lease, "session-999")).toBe(false);

    const emptyLease = testLease({ sessions: [] });
    expect(mayObserve(emptyLease, "session-001")).toBe(false);
  });

  test("denies raw NATS authority anywhere in generated messages", () => {
    const err = capture(() =>
      denyRaw({
        type: "content.intent",
        payload: {
          subject: "tb.internal.admin.delete",
          nested: { token: "secret" },
        },
      }),
    );

    expect(err.kind).toBe("FrameCapabilityDenied");
    expect(err.details.path).toBe("payload.subject");

    const map = capture(() =>
      denyRaw({
        type: "content.intent",
        payload: new Map([["subject", "tb.internal.admin.delete"]]),
      }),
    );
    expect(map.kind).toBe("FrameCapabilityDenied");
    expect(map.details.path).toBe("payload.subject");

    const set = capture(() =>
      denyRaw({
        type: "content.intent",
        payload: new Set([{ token: "secret" }]),
      }),
    );
    expect(set.kind).toBe("FrameCapabilityDenied");
    expect(set.details.path).toBe("payload.0.token");
  });
});

function testLease(extra: Record<string, unknown> = {}) {
  return makeLease({
    frameId: "frame-001",
    nonce: "nonce-001",
    sessionId: "session-001",
    capabilityId: "cap-001",
    artifactId: "artifact-001",
    artifactRevision: "artifact.rev.7",
    schemaRevision: "schema.rev.1",
    commands: ["select_artifact", "steer_session"],
    sessions: ["session-001", "session-002"],
    chain: {
      chainId: "chain-001",
      rootId: "root-001",
      hop: 0,
      maxHops: 5,
    },
    ...extra,
  } as Parameters<typeof makeLease>[0]);
}

function msg(overrides: Record<string, unknown> = {}) {
  return {
    type: "content.intent",
    command: "select_artifact",
    commandId: "cmd-001",
    expectedRevision: "artifact.rev.7",
    nonce: "nonce-001",
    frameId: "frame-001",
    artifactRevision: "artifact.rev.7",
    schemaRevision: "schema.rev.1",
    payload: { artifactKey: "preview.main" },
    ...overrides,
  };
}

function capture(fn: () => unknown): FrameError {
  try {
    fn();
  } catch (error) {
    if (error instanceof FrameError) return error;
    throw error;
  }
  throw new Error("expected FrameError");
}
