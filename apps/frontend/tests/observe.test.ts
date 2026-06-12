import { describe, expect, test } from "bun:test";
import { frameLine } from "../src/observe";

describe("observe panel frame rendering", () => {
  test("token frame renders its text", () => {
    const line = frameLine(
      JSON.stringify({
        kind: "session.frame",
        frame: "token",
        origin: "wrapper",
        sessionId: "demo-001",
        text: "tick 7 at 12:00:07\n",
      }),
    );
    expect(line).toBe("tick 7 at 12:00:07\n");
  });

  test("chunk frames are not rendered as text", () => {
    const line = frameLine(
      JSON.stringify({
        kind: "session.frame",
        frame: "chunk",
        origin: "wrapper",
        sessionId: "demo-001",
        body: "{\"type\":\"result\"}",
      }),
    );
    expect(line).toBeNull();
  });

  test("malformed frames are ignored", () => {
    expect(frameLine("{not json")).toBeNull();
    expect(frameLine(JSON.stringify({ frame: "token" }))).toBeNull();
  });
});
