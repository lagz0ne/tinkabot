import { NatsServer, type NatsServerOptions } from "@lagz0ne/nats-embedded";
import {
  connect,
  type ConnectionOptions,
  type KV,
  type KvOptions,
} from "nats";
import {
  TinkabotRuntimeError,
  errorMessage,
  type RuntimeErrorOrigin,
} from "./errors";

export interface EmbeddedNatsServer {
  readonly url: string;
  readonly port: number;
  stop(): Promise<void>;
}

export interface RuntimeNatsConnection {
  jetstream(): {
    views: {
      kv(name: string, opts?: Partial<KvOptions>): Promise<KV>;
    };
  };
  drain?(): Promise<void>;
  close?(): Promise<void>;
}

export type EmbeddedServerFactory = (
  options: NatsServerOptions,
) => Promise<EmbeddedNatsServer>;

export type NatsConnectionFactory = (
  options: ConnectionOptions,
) => Promise<RuntimeNatsConnection>;

export interface RuntimeSubstrateOptions
  extends Omit<NatsServerOptions, "jetstream" | "storeDir"> {
  storeDir: string;
  serverFactory?: EmbeddedServerFactory;
  connectFactory?: NatsConnectionFactory;
}

export class RuntimeSubstrate {
  readonly url: string;
  readonly port: number;

  private stopped = false;

  private constructor(
    private readonly server: EmbeddedNatsServer,
    private readonly connection: RuntimeNatsConnection,
  ) {
    this.url = server.url;
    this.port = server.port;
  }

  static async start(options: RuntimeSubstrateOptions): Promise<RuntimeSubstrate> {
    const {
      serverFactory = (serverOptions) => NatsServer.start(serverOptions),
      connectFactory = (connectOptions) => connect(connectOptions),
      ...serverOptions
    } = options;

    let server: EmbeddedNatsServer;
    try {
      server = await serverFactory({
        ...serverOptions,
        jetstream: true,
        storeDir: options.storeDir,
      });
    } catch (error) {
      throw mapSubstrateError(error, "start", "SubstrateStartupFailed");
    }

    try {
      const connection = await connectFactory({ servers: server.url });
      return new RuntimeSubstrate(server, connection);
    } catch (error) {
      await server.stop().catch(() => undefined);
      throw mapSubstrateError(error, "connect", "SubstrateUnavailable");
    }
  }

  async openKvBucket(
    name: string,
    options?: Partial<KvOptions>,
  ): Promise<KV> {
    try {
      return await this.connection.jetstream().views.kv(name, options);
    } catch (error) {
      throw mapSubstrateError(error, "openKvBucket", "SubstrateUnavailable");
    }
  }

  async stop(): Promise<void> {
    if (this.stopped) return;

    let cleanupError: unknown;
    try {
      if (this.connection.drain) {
        await this.connection.drain();
      } else {
        await this.connection.close?.();
      }
    } catch (error) {
      cleanupError = error;
    }

    try {
      await this.server.stop();
    } catch (error) {
      cleanupError ??= error;
    }

    if (cleanupError !== undefined) {
      throw mapSubstrateError(
        cleanupError,
        "stop",
        "SubstrateCleanupFailed",
      );
    }

    this.stopped = true;
  }
}

function mapSubstrateError(
  error: unknown,
  operation: string,
  fallbackKind:
    | "SubstrateStartupFailed"
    | "SubstrateUnavailable"
    | "SubstrateCleanupFailed",
): TinkabotRuntimeError {
  const origin: RuntimeErrorOrigin = {
    layer: "RuntimeSubstrate",
    operation,
  };

  if (error instanceof TinkabotRuntimeError) return error;

  if (!(error instanceof Error)) {
    return new TinkabotRuntimeError(
      "SubstrateCritical",
      `Runtime substrate ${operation} failed with a non-error value`,
      { origin, cause: error },
    );
  }

  return new TinkabotRuntimeError(
    fallbackKind,
    `Runtime substrate ${operation} failed: ${errorMessage(error)}`,
    { origin, cause: error },
  );
}
