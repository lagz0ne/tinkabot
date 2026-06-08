import { describe, expect, test } from "bun:test";
import {
  createBrowserCommandIntent,
  createBrowserEdgeBootstrap,
  createCommandAcceptance,
  createMemoryCommandAcceptanceStore,
  parseContract,
  toBrowserEdgeContentStatus,
  type Contract,
} from "../../src/endgame-contract/index";
import { TinkabotRuntimeError } from "../../src/nats-script-runtime/errors";

type AuthPolicy = Extract<Contract, { kind: "auth.policy" }>;
type ArtifactManifest = Extract<Contract, { kind: "artifact.manifest" }>;

const root = new URL("../../../../", import.meta.url);
const dir = new URL("schemas/endgame/v1/", root);

describe("SubstrateEdgeBootstrap", () => {
  test("T-EDGE-CREDENTIAL-SPLIT keeps worker credentials out of generated content bootstrap", async () => {
    const edge = await bootstrap();

    expect(edge.worker).toMatchObject({
      principalId: "principal.browser.001",
      sessionId: "session-001",
      capabilityId: "cap-001",
      leaseId: "lease-001",
      credentialDescriptor: {
        kind: "browser.worker_nats",
        ref: "credential.browser.worker.001",
        schemaVersion: "v1",
        appRevision: "app.rev.1",
        artifactRevision: "artifact.rev.7",
        publishAllow: ["tb.proof.out.allowed.>"],
        subscribeAllow: ["tb.proof.runtime.>"],
      },
    });

    expect(edge.content).toEqual({
      sessionId: "session-001",
      capabilityId: "cap-001",
      artifactId: "artifact-001",
      artifactRevision: "artifact.rev.7",
      frameId: "frame-001",
      artifact: {
        digest: "sha256:5a1e",
        mediaType: "application/javascript",
        objectRef: "obj://frontend/artifact-001/rev-7/bundle.js",
        cspPolicy: "csp.subapp.v1",
        framePolicy: "frame.subapp.v1",
        sandboxPolicy: "sandbox.browser.subapp.v1",
      },
      chain: chain(),
    });
    expect(rawAuthorityHits(edge.content)).toEqual([]);

    const authPolicy = await readJson("fixtures/valid/auth-policy.json");
    const artifactManifest = await readJson("fixtures/valid/artifact-manifest.json");
    const missingRef = capture(() =>
      createBrowserEdgeBootstrap({
        authPolicy,
        artifactManifest,
        credentialRef: "",
        frameId: "frame-001",
        chain: chain(),
      }),
    );
    expect(missingRef.kind).toBe("BrowserEdgeInvalid");
    expect(missingRef.origin.details).toMatchObject({
      field: "credentialRef",
    });
  });

  test("T-EDGE-LEASE-DENY denies revoked bootstrap before worker credentials are returned", async () => {
    const authPolicy = await readJson("fixtures/valid/auth-policy-revoked-lease.json");
    const artifactManifest = await readJson("fixtures/valid/artifact-manifest.json");
    const error = capture(() =>
      createBrowserEdgeBootstrap({
        authPolicy,
        artifactManifest,
        credentialRef: "credential.browser.worker.001",
        frameId: "frame-001",
        chain: chain(),
      }),
    );

    expect(error.kind).toBe("RevokedLease");
    expect(error.origin.layer).toBe("ManagedAuth");
  });

  test("T-EDGE-COMMAND-CANONICAL bridges content intent to browser.command_intent only", async () => {
    const edge = await bootstrap();
    const intent = createBrowserCommandIntent(edge.content, {
      type: "content.intent",
      command: "select_artifact",
      commandId: "cmd-edge-001",
      expectedRevision: "artifact.rev.7",
      payload: { artifactKey: "preview.main" },
    });

    expect(intent).toMatchObject({
      kind: "browser.command_intent",
      type: "content.intent",
      command: "select_artifact",
      commandId: "cmd-edge-001",
      expectedRevision: "artifact.rev.7",
      context: {
        sessionId: "session-001",
        capabilityId: "cap-001",
        artifactId: "artifact-001",
        artifactRevision: "artifact.rev.7",
        frameId: "frame-001",
      },
    });
    expect(parseContract(intent).kind).toBe("browser.command_intent");

    const result = await createAcceptance(await auth()).accept(intent);
    expect(result.status.status).toBe("accepted");
    expect(result.activation?.source.kind).toBe("command_acceptance");

    const localFrontendIntent = capture(() =>
      createBrowserCommandIntent(edge.content, {
        type: "frontend.command_intent",
        command: "select_artifact",
        commandId: "cmd-local-001",
        expectedRevision: "artifact.rev.7",
      }),
    );
    expect(localFrontendIntent.kind).toBe("BrowserEdgeInvalid");
    expect(localFrontendIntent.origin.layer).toBe("BrowserEdge");
  });

  test("T-EDGE-REVOKE returns sanitized revoked denial to generated content", async () => {
    const edge = await bootstrap();
    const revoked = await auth("fixtures/valid/auth-policy-revoked-lease.json");
    const acceptance = createAcceptance(revoked);
    const status = await acceptance.accept(
      createBrowserCommandIntent(edge.content, {
        type: "content.intent",
        command: "select_artifact",
        commandId: "cmd-revoked-edge-001",
        expectedRevision: "artifact.rev.7",
        payload: { artifactKey: "preview.main" },
      }),
    );

    expect(status.activation).toBeUndefined();
    expect(status.status).toMatchObject({
      status: "rejected",
      error: {
        kind: "RevokedLease",
        origin: {
          layer: "ManagedAuth",
          operation: "authorizeCommand",
        },
      },
    });

    const content = toBrowserEdgeContentStatus(status.status);
    expect(content).toEqual({
      type: "browser_edge.command_status",
      commandId: "cmd-revoked-edge-001",
      status: "rejected",
      sequence: 1,
      observedAt: "2026-06-08T00:00:10.000Z",
      error: {
        kind: "RevokedLease",
        message: "Capability lease has been revoked",
        layer: "ManagedAuth",
      },
    });
    expect(rawAuthorityHits(content)).toEqual([]);
  });
});

async function bootstrap() {
  return createBrowserEdgeBootstrap({
    authPolicy: await auth(),
    artifactManifest: await artifact(),
    credentialRef: "credential.browser.worker.001",
    frameId: "frame-001",
    chain: chain(),
  });
}

function createAcceptance(policy: AuthPolicy) {
  return createCommandAcceptance({
    provenance: policy.provenance,
    capability: policy.capability,
    currentArtifactRevision: "artifact.rev.7",
    store: createMemoryCommandAcceptanceStore(),
    now: () => "2026-06-08T00:00:10.000Z",
    routes: {
      select_artifact: {
        activationName: "browser_command",
        scriptKey: "scripts.proof.select_artifact",
        scriptRevision: 7,
        subject: "tb.proof.runtime.execute",
      },
    },
  });
}

async function auth(path = "fixtures/valid/auth-policy.json"): Promise<AuthPolicy> {
  const parsed = parseContract(await readJson(path));
  if (parsed.kind !== "auth.policy") throw new Error("expected auth policy");
  return parsed;
}

async function artifact(): Promise<ArtifactManifest> {
  const parsed = parseContract(await readJson("fixtures/valid/artifact-manifest.json"));
  if (parsed.kind !== "artifact.manifest") {
    throw new Error("expected artifact manifest");
  }
  return parsed;
}

async function readJson<T = any>(path: string): Promise<T> {
  return Bun.file(new URL(path, dir)).json();
}

function chain() {
  return {
    chainId: "chain-001",
    rootId: "root-001",
    parentId: "parent-001",
    hop: 1,
    maxHops: 5,
  };
}

function rawAuthorityHits(value: unknown): string[] {
  const hits: string[] = [];
  scan(value, [], hits);
  return hits;
}

function scan(value: unknown, path: string[], hits: string[]): void {
  if (!value || typeof value !== "object") return;
  for (const [key, item] of Object.entries(value)) {
    const normalized = key.toLowerCase().replace(/[-_]/g, "");
    if (
      [
        "nats",
        "subject",
        "token",
        "credential",
        "permission",
        "publish",
        "subscribe",
      ].some((raw) => normalized.includes(raw))
    ) {
      hits.push([...path, key].join("."));
    }
    scan(item, [...path, key], hits);
  }
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
