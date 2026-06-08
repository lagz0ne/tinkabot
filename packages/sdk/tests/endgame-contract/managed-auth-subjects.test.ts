import { describe, expect, test } from "bun:test";
import {
  assertPublish,
  assertSubscribe,
  classifySubject,
  compileAuth,
  parseContract,
} from "../../src/endgame-contract/index";
import {
  TinkabotRuntimeError,
  type RuntimeErrorKind,
  type RuntimeErrorLayer,
} from "../../src/nats-script-runtime/errors";

interface AuthCase {
  caseId: string;
  fixture: string;
  expect: {
    policy: string;
    compiledFixture?: string;
    error?: {
      kind: RuntimeErrorKind;
      layer: RuntimeErrorLayer;
    };
  };
}

const root = new URL("../../../../", import.meta.url);
const dir = new URL("schemas/endgame/v1/", root);

describe("ManagedAuthSubjects", () => {
  test("T-MAUTH-COMPILE preserves provenance while compiling NATS-shaped authority", async () => {
    const item = await authCase("active-policy-compiles");
    const policy = parseAuth(await readJson(item.fixture));
    const expected = await readJson(item.expect.compiledFixture!);

    expect(compileAuth(policy)).toEqual(expected);
  });

  test("T-MAUTH-DENY enforces allow, denied-neighbor, and deny-over-allow", async () => {
    const policy = parseAuth(await readJson("fixtures/valid/auth-policy.json"));
    const auth = compileAuth(policy);

    expect(assertPublish(auth, "tb.proof.out.allowed.exec_success_001")).toBeUndefined();

    const denied = capture(() =>
      assertPublish(auth, "tb.proof.out.denied.exec_denied_publish_001"),
    );
    expect(denied.kind).toBe("PermissionDeniedByDenyRule");
    expect(denied.origin.layer).toBe("ManagedAuth");
    expect(denied.origin.details).toMatchObject({
      principalId: "principal.browser.001",
      leaseId: "lease-001",
      subject: "tb.proof.out.denied.exec_denied_publish_001",
    });

    const neighbor = capture(() =>
      assertPublish(auth, "tb.proof.runtime.execute"),
    );
    expect(neighbor.kind).toBe("PermissionDenied");
    expect(neighbor.origin.layer).toBe("ManagedAuth");

    expect(assertSubscribe(auth, "tb.proof.runtime.execute")).toBeUndefined();
  });

  test("T-SUBJECT-TAXONOMY classifies authority and rejects reserved or overbroad subjects", async () => {
    expect(classifySubject("tb.proof.out.allowed.exec_success_001")).toMatchObject({
      plane: "app",
      owner: "script",
    });
    expect(classifySubject("tb.internal.control.auth.rotate")).toMatchObject({
      plane: "control",
      reserved: true,
    });

    for (const id of ["control-overgrant-denied", "wildcard-overreach-denied"]) {
      const item = await authCase(id);
      const policy = parseAuth(await readJson(item.fixture));
      const error = capture(() => compileAuth(policy));

      expect(error.kind).toBe(item.expect.error!.kind);
      expect(error.origin.layer).toBe(item.expect.error!.layer);
    }
  });

  test("T-MAUTH-LEASE-REVISION rejects revoked leases and stale provenance", async () => {
    for (const id of [
      "revoked-lease-denied",
      "expired-lease-denied",
      "provenance-mismatch-denied",
    ]) {
      const item = await authCase(id);
      const policy = parseAuth(await readJson(item.fixture));
      const error = capture(() => compileAuth(policy));

      expect(error.kind).toBe(item.expect.error!.kind);
      expect(error.origin.layer).toBe(item.expect.error!.layer);
      expect(error.origin.details).toMatchObject({
        principalId: "principal.browser.001",
        capabilityId: "cap-001",
        leaseId: "lease-001",
        leaseStatus: policy.capability.leaseStatus,
      });
    }
  });

  test("T-MAUTH-CAPABILITY denies advanced or unbounded schema-valid authority", async () => {
    for (const id of [
      "advanced-import-denied",
      "advanced-exposure-denied",
      "unbounded-response-denied",
      "exposure-subject-missing-denied",
    ]) {
      const item = await authCase(id);
      const policy = parseAuth(await readJson(item.fixture));
      const error = capture(() => compileAuth(policy));

      expect(error.kind).toBe(item.expect.error!.kind);
      expect(error.origin.layer).toBe(item.expect.error!.layer);
      expect(error.origin.details).toMatchObject({
        principalId: "principal.browser.001",
        capabilityId: "cap-001",
        leaseStatus: "active",
      });
    }
  });

  test("T-MAUTH-EXPORT-EXPOSURE requires paired exported exposure subjects", async () => {
    for (const id of [
      "exposure-missing-export-denied",
      "export-missing-exposure-denied",
    ]) {
      const item = await authCase(id);
      const policy = parseAuth(await readJson(item.fixture));
      const error = capture(() => compileAuth(policy));

      expect(error.kind).toBe(item.expect.error!.kind);
      expect(error.origin.layer).toBe(item.expect.error!.layer);
      expect(error.origin.details).toMatchObject({
        principalId: "principal.browser.001",
        capabilityId: "cap-001",
        leaseStatus: "active",
      });
    }
  });
});

async function authCase(id: string): Promise<AuthCase> {
  const cases = await readJson<AuthCase[]>("managed-auth-subjects.cases.json");
  const item = cases.find((candidate) => candidate.caseId === id);
  if (!item) throw new Error(`missing auth case: ${id}`);
  return item;
}

async function readJson<T = any>(path: string): Promise<T> {
  return Bun.file(new URL(path, dir)).json();
}

function parseAuth(input: unknown) {
  const parsed = parseContract(input);
  if (parsed.kind !== "auth.policy") throw new Error("expected auth policy");
  return parsed;
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
