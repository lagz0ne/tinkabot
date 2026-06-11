import { describe, expect, test } from "bun:test";
import {
  contractSchemaId,
  parseContract,
} from "../../src/base-contract/index";
import {
  TinkabotRuntimeError,
  type RuntimeErrorKind,
} from "../../src/nats-script-runtime/errors";

interface Case {
  caseId: string;
  fixture: string;
  expect: {
    valid: boolean;
    errorKind?: RuntimeErrorKind;
    ownerLayer?: string;
  };
}

const root = new URL("../../../../", import.meta.url);
const dir = new URL("schemas/base/v1/", root);

const requiredCases = [
  "session-record",
  "session-frame-token",
  "session-frame-chunk",
  "session-frame-status",
  "session-steer-intent",
  "session-stop-intent",
  "session-frame-malformed",
  "session-frame-unknown-kind",
  "session-record-missing-provenance",
];

// Mirror of the facade scan vocabulary in substrate/go/core/script_materializer.go:24-39.
// The scan matches field NAMES only (normalized substring), never values, so the frame
// kind VALUE "token" does not collide; session field NAMES must avoid these substrings.
const rawWords = [
  "subject",
  "reply",
  "token",
  "cred",
  "permission",
  "publish",
  "subscribe",
  "nats",
  "nkey",
  "jwt",
  "seed",
  "secret",
  "password",
  "bearer",
];

describe("SessionContractAuthority", () => {
  test("T-SESSION-PARITY session shapes are canonical in schemas/base/v1 and TS/Zod agrees with session.cases.json", async () => {
    const schema = await readJson("contract.schema.json");
    expect(schema.$id).toBe(contractSchemaId);
    expect(schema.$defs.sessionRecord).toBeDefined();
    expect(schema.$defs.sessionFrame).toBeDefined();
    expect(schema.$defs.steerIntent).toBeDefined();
    expect(schema.$defs.trustTier).toBeDefined();

    const cases = await readJson<Case[]>("session.cases.json");
    const ids = new Set(cases.map((item) => item.caseId));
    for (const id of requiredCases) expect(ids).toContain(id);

    for (const item of cases) {
      expect(item.expect.ownerLayer).toBeString();

      const fixture = await readJson(item.fixture);
      if (item.expect.valid) {
        expect(parseContract(fixture).kind).toBeString();
        continue;
      }

      const error = capture(() => parseContract(fixture));
      expect(error).toBeInstanceOf(TinkabotRuntimeError);
      expect(error.kind).toBe(item.expect.errorKind ?? "ContractInvalid");
      expect(error.origin.layer).toBe("ContractAuthority");
    }

    const byId = new Map(cases.map((item) => [item.caseId, item]));
    const status = await readJson(byId.get("session-frame-status")!.fixture);
    expect(status.origin).toBe("runner");
    for (const id of ["session-frame-token", "session-frame-chunk"]) {
      const frame = await readJson(byId.get(id)!.fixture);
      expect(frame.origin).toBe("wrapper");
    }
  });

  test("T-SESSION-UNKNOWN-FRAME a session frame with an unknown frame kind is denied at ContractAuthority", async () => {
    const fixture = await readJson("fixtures/invalid/session-frame-unknown-kind.json");

    const error = capture(() => parseContract(fixture));
    expect(error).toBeInstanceOf(TinkabotRuntimeError);
    expect(error.kind).toBe("ContractInvalid");
    expect(error.origin.layer).toBe("ContractAuthority");
  });

  test("T-SESSION-MISSING-PROVENANCE a session record without provenance is denied at ContractAuthority", async () => {
    const fixture = await readJson("fixtures/invalid/session-record-missing-provenance.json");

    const error = capture(() => parseContract(fixture));
    expect(error).toBeInstanceOf(TinkabotRuntimeError);
    expect(error.kind).toBe("ContractInvalid");
    expect(error.origin.layer).toBe("ContractAuthority");
  });

  test("T-SESSION-RESERVED-VOCAB session field names avoid the facade rawWords vocabulary", async () => {
    const cases = await readJson<Case[]>("session.cases.json");
    const valid = cases.filter((item) => item.expect.valid);
    expect(valid.length).toBeGreaterThan(0);

    for (const item of valid) {
      const fixture = await readJson(item.fixture);
      expect(collidingNames(fixture)).toEqual([]);
    }
  });
});

function collidingNames(value: unknown, path = ""): string[] {
  if (!value || typeof value !== "object") return [];
  if (Array.isArray(value)) {
    return value.flatMap((item, index) => collidingNames(item, `${path}[${index}]`));
  }
  return Object.entries(value).flatMap(([key, item]) => {
    const at = path ? `${path}.${key}` : key;
    const normalized = key.replace(/[-_]/g, "").toLowerCase();
    const hit = rawWords.some((word) => normalized.includes(word));
    return [...(hit ? [at] : []), ...collidingNames(item, at)];
  });
}

async function readJson<T = any>(path: string): Promise<T> {
  return Bun.file(new URL(path, dir)).json();
}

function capture(action: () => unknown): TinkabotRuntimeError {
  try {
    action();
  } catch (error) {
    if (error instanceof TinkabotRuntimeError) return error;
    throw error;
  }
  throw new Error("expected action to fail");
}
