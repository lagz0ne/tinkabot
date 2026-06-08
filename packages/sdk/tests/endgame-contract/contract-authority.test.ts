import { describe, expect, test } from "bun:test";
import {
  contractSchemaId,
  parseContract,
} from "../../src/endgame-contract/index";
import {
  TinkabotRuntimeError,
  type RuntimeErrorKind,
} from "../../src/nats-script-runtime/errors";

interface ParityCase {
  caseId: string;
  fixture: string;
  expect: {
    valid: boolean;
    errorKind?: RuntimeErrorKind;
    policy?: string;
  };
}

const root = new URL("../../../../", import.meta.url);
const dir = new URL("schemas/endgame/v1/", root);

describe("EndgameContractAuthority", () => {
  test("T-ENDGAME-CONTRACT loads the canonical schema id", async () => {
    const schema = await readJson("contract.schema.json");

    expect(schema.$id).toBe(contractSchemaId);
  });

  test("T-ENDGAME-CONTRACT keeps TypeScript/Zod parity with shared fixtures", async () => {
    const cases = await readJson<ParityCase[]>("parity.cases.json");

    expect(cases.length).toBeGreaterThan(0);
    expect(new Set(cases.map((item) => item.expect.policy))).toEqual(
      new Set([
        "allowed",
        "no-raw-authority",
        "malformed",
        "stale-revision-denied",
        "revoked-lease-denied",
        "expired-lease-denied",
        "reserved-subject-denied",
        "wildcard-overreach-denied",
        "advanced-capability-denied",
        "response-unbounded-denied",
        "import-export-mismatch",
        "exposure-subject-missing",
        "attributed-failure",
      ]),
    );

    for (const item of cases) {
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
  });
});

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
