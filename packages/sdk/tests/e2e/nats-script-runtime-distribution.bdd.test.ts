import { afterEach, describe, expect, test } from "bun:test";
import { createRequire } from "node:module";
import { mkdtemp, rm, stat } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { pathToFileURL } from "node:url";
import type {
  RuntimeSubstrate,
  ScriptRecord,
  ScriptRecordStore,
  TinkabotRuntimeError,
} from "../../src/nats-script-runtime/index";

interface DistributionApi {
  RuntimeSubstrate: typeof RuntimeSubstrate;
  ScriptRecordStore: typeof ScriptRecordStore;
  TinkabotRuntimeError: typeof TinkabotRuntimeError;
}

const projectRoot = resolve(import.meta.dir, "../..");
const distDir = join(projectRoot, "dist");
const tempDirs: string[] = [];

afterEach(async () => {
  while (tempDirs.length > 0) {
    const dir = tempDirs.pop();
    if (dir) await rm(dir, { recursive: true, force: true });
  }
});

describe("Feature: NATS script runtime distribution", () => {
  test(
    "Scenario: built package runs substrate and record-store contract end to end",
    async () => {
      await givenCleanDistributionBuild();
      const api = await whenConsumerImportsEsmDistribution();
      const cjs = whenConsumerRequiresCjsDistribution();

      thenDistributionExportsRuntimeApi(api);
      thenDistributionExportsRuntimeApi(cjs);

      const substrate = await api.RuntimeSubstrate.start({
        storeDir: await tempStoreDir(),
      });

      try {
        const store = await api.ScriptRecordStore.open(substrate, {
          bucket: "TB_SCRIPT_RECORDS_BDD_DIST",
          history: 3,
        });

        const firstRevision = await store.create(
          "scripts.dist.echo",
          record("first"),
        );
        const secondRevision = await store.update(
          "scripts.dist.echo",
          record("second"),
          { previousRevision: firstRevision },
        );

        const first = await store.get("scripts.dist.echo", {
          revision: firstRevision,
        });
        const second = await store.get("scripts.dist.echo", {
          revision: secondRevision,
        });

        expect(first.record.source).toBe("export default 'first';");
        expect(first.revision).toBe(firstRevision);
        expect(second.record.source).toBe("export default 'second';");
        expect(second.revision).toBe(secondRevision);

        const deletedRevision = await store.create(
          "scripts.dist.deleted",
          record("deleted"),
        );
        await store.delete("scripts.dist.deleted", {
          previousRevision: deletedRevision,
        });

        const deletedError = await captureRuntimeError(api, () =>
          store.get("scripts.dist.deleted"),
        );
        expect(deletedError.kind).toBe("RecordDeletedOrStale");
        expect(deletedError.origin.layer).toBe("ScriptRecordStore");
      } finally {
        await substrate.stop();
      }
    },
    15_000,
  );
});

async function givenCleanDistributionBuild(): Promise<void> {
  await rm(distDir, { recursive: true, force: true });

  const build = Bun.spawn(["bun", "run", "build"], {
    cwd: projectRoot,
    stdout: "pipe",
    stderr: "pipe",
  });
  const [stdout, stderr, exitCode] = await Promise.all([
    new Response(build.stdout).text(),
    new Response(build.stderr).text(),
    build.exited,
  ]);

  if (exitCode !== 0) {
    throw new Error(`distribution build failed\n${stdout}\n${stderr}`);
  }

  for (const file of ["index.mjs", "index.cjs", "index.d.mts", "index.d.cts"]) {
    expect(await exists(join(distDir, file))).toBe(true);
  }
}

async function whenConsumerImportsEsmDistribution(): Promise<DistributionApi> {
  const url = `${pathToFileURL(join(distDir, "index.mjs")).href}?${Date.now()}`;
  return (await import(url)) as DistributionApi;
}

function whenConsumerRequiresCjsDistribution(): DistributionApi {
  const require = createRequire(import.meta.url);
  return require(join(distDir, "index.cjs")) as DistributionApi;
}

function thenDistributionExportsRuntimeApi(api: DistributionApi): void {
  expect(typeof api.RuntimeSubstrate?.start).toBe("function");
  expect(typeof api.ScriptRecordStore?.open).toBe("function");
  expect(typeof api.TinkabotRuntimeError).toBe("function");
}

async function tempStoreDir(): Promise<string> {
  const dir = await mkdtemp(join(tmpdir(), "tinkabot-dist-"));
  tempDirs.push(dir);
  return dir;
}

function record(label: string): ScriptRecord {
  return {
    source: `export default '${label}';`,
    metadata: {
      id: `script-${label}`,
      desc: `Distribution BDD fixture ${label}`,
      runtime: {
        language: "typescript",
        sandbox: "none",
      },
    },
  };
}

async function captureRuntimeError(
  api: DistributionApi,
  action: () => Promise<unknown>,
): Promise<InstanceType<typeof TinkabotRuntimeError>> {
  try {
    await action();
  } catch (error) {
    if (error instanceof api.TinkabotRuntimeError) {
      return error as InstanceType<typeof TinkabotRuntimeError>;
    }
    throw error;
  }
  throw new Error("expected action to fail");
}

async function exists(path: string): Promise<boolean> {
  try {
    await stat(path);
    return true;
  } catch {
    return false;
  }
}
