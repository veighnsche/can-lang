import { success, failure, invoke, type Completion, type AssertionContext } from "../completion.ts";
import { denyLiveBoundary } from "../assert/context.ts";
import { record } from "../data.ts";
import { copyBytes } from "../bytes.ts";
import { createDomainRuntime } from "../domain.ts";
import { resourceStateFailure } from "../failure.ts";
import {
  registerResource,
  useResource,
  closeResource,
  guardCallback,
  withScope,
  type Resource,
  type Scope,
} from "../owner.ts";
import {
  snapshotRequest,
  snapshotRequestLazy,
  normalizedPath,
  abandonRequest,
  nativeResponse,
  bindRequestServer,
  isUpgradedResponse,
  type UpgradeServer,
} from "./http.ts";
import {
  matchActionRoute,
  actionReject,
  bunRouteKeys,
  type ActionRouteTable,
  type ActionRouteCapture,
} from "./action-routes.ts";
import { dispatch, isRouterValue, routeKind } from "./router.ts";
import {
  serverSocketOpen,
  serverSocketMessage,
  serverSocketClose,
  serverSocketDrain,
  closeServerSessions,
} from "./websocket.ts";
import { browserPolicy, type AssetServer } from "./assets.ts";
const origin = Object.freeze({
  source: "can:server",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
type Config = Readonly<{ host: string; port: bigint; bodyLimit: number; shutdownMs: number }>;
type TlsMaterial = Readonly<{ cert: Uint8Array; key: Uint8Array }>;
type Native = Readonly<{
  server: Readonly<{ stop: (closeActiveConnections?: boolean) => void } & UpgradeServer>;
  scope: Scope;
  router: unknown;
  bodyLimit: number;
  shutdownMs: number;
  settled: Promise<Completion<void>>;
  settle: (completion: Completion<void>) => void;
}>;
const configs = new WeakMap<object, Config>(),
  material = new WeakMap<object, TlsMaterial>(),
  servers = new WeakMap<object, Native>();
const object = (value: unknown): value is object =>
  value !== null && (typeof value === "object" || typeof value === "function");
export function isServerValue(kind: string | undefined, value: unknown): boolean {
  if (kind === "server_config") return object(value) && configs.has(value);
  if (kind === "tls_config") return object(value) && material.has(value);
  if (kind === "server") return object(value) && servers.has(value);
  return false;
}
// Structural PEM check only: block markers must tile the whole input and the
// payload must carry base64. Bun parses the DER at serve time; anything it
// rejects surfaces as a bind failure. Accepted labels are documented.
const pemBlock = /-----BEGIN ([A-Z0-9 ]+)-----([A-Za-z0-9+/=\r\n \t]+)-----END ([A-Z0-9 ]+)-----/g;
function pemLabels(text: string): string[] | undefined {
  const labels: string[] = [];
  let pos = 0;
  for (;;) {
    while (pos < text.length && /\s/.test(text[pos]!)) pos++;
    if (pos >= text.length) return labels;
    pemBlock.lastIndex = pos;
    const match = pemBlock.exec(text);
    if (
      !match ||
      match.index !== pos ||
      match[1] !== match[3] ||
      match[2]!.replace(/\s/g, "") === ""
    )
      return undefined;
    labels.push(match[1]!);
    pos = pemBlock.lastIndex;
  }
}
function pemText(input: unknown): string | undefined {
  let bytes: Uint8Array;
  try {
    bytes = copyBytes(input, origin);
  } catch {
    return undefined;
  }
  if (bytes.byteLength === 0 || bytes.byteLength > 1048576) return undefined;
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch {
    return undefined;
  }
}
function fixed(status: 400 | 413 | 500): Response {
  const headers = new Headers({
    "content-type": "text/plain; charset=utf-8",
    "x-content-type-options": "nosniff",
  });
  return new Response(
    status === 400 ? "Bad Request" : status === 413 ? "Payload Too Large" : "Internal Server Error",
    { status, headers },
  );
}
const idle: AssetServer = {
  async serve(): Promise<Response | undefined> {
    return undefined;
  },
};
function withPolicy(response: Response): Response {
  const headers = new Headers(response.headers);
  headers.set("content-security-policy", browserPolicy);
  headers.set("x-content-type-options", "nosniff");
  const status = response.status;
  const body = status === 204 || status === 205 || status === 304 ? null : response.body;
  return new Response(body, { status, statusText: response.statusText, headers });
}
export type ActionInvoke = (
  identity: string,
  captures: readonly ActionRouteCapture[],
  request: unknown,
  context?: AssertionContext,
) => Promise<Completion<unknown>>;
export type ActionAdapter = Readonly<{ table: ActionRouteTable; invoke: ActionInvoke }>;
export function createServer(
  domain: ReturnType<typeof createDomainRuntime>,
  types: Readonly<{ invalidConfig: string; bindFailed: string; shutdownFailed: string }>,
  assets: AssetServer = idle,
  actions?: ActionAdapter,
) {
  const invalid = (reason: string) =>
    failure(
      domain.create(types.invalidConfig, record(types.invalidConfig, [["reason", reason]]), origin),
    );
  const shutdown = (phase: string) =>
    failure(
      domain.create(types.shutdownFailed, record(types.shutdownFailed, [["phase", phase]]), origin),
    );
  function readConfig(value: unknown): Config {
    if (!object(value) || !configs.has(value)) throw resourceStateFailure(undefined, origin);
    return configs.get(value)!;
  }
  function readServer(value: unknown): Native {
    if (!object(value) || !servers.has(value)) throw resourceStateFailure(undefined, origin);
    return servers.get(value)!;
  }
  async function launch(
    config: Config,
    router: unknown,
    context: AssertionContext | undefined,
    tls?: TlsMaterial,
  ): Promise<Completion<unknown>> {
    const { host, port, bodyLimit, shutdownMs } = config;
    if (!isRouterValue("router", router)) throw resourceStateFailure(undefined, origin);
    const address = host + ":" + port.toString();
    const bind = () =>
      failure(
        domain.create(types.bindFailed, record(types.bindFailed, [["address", address]]), origin),
      );
    let release!: () => void;
    const lifetime = new Promise<void>((resolve) => {
      release = resolve;
    });
    // The live server resource belongs to the caller's scope so stop and wait
    // validate from user code; only fetch tasks live in the drainable child.
    let token!: Resource;
    let ready!: (
      value: Readonly<{ server?: Native["server"]; scope?: Scope; state?: boolean }>,
    ) => void;
    const decided = new Promise<
      Readonly<{ server?: Native["server"]; scope?: Scope; state?: boolean }>
    >((resolve) => {
      ready = resolve;
    });
    const scoped = withScope(async (scope): Promise<Completion<undefined>> => {
      // Canonical action dispatch runs after ingress on the raw pathname, so
      // method-first/static-within-method matching and 400/404/405
      // classification see the escapes the decoded snapshot already resolved.
      // Anything actions do not claim falls through to the legacy router.
      async function serveSnapshot(
        native: Request,
        snapshot: unknown,
      ): Promise<Completion<Response>> {
        if (actions !== undefined) {
          const match = matchActionRoute(
            actions.table,
            native.method,
            new URL(native.url).pathname,
          );
          if (match.kind === "match") {
            try {
              const completed = await invoke(
                () => actions.invoke(match.identity, match.captures, snapshot, context),
                origin,
              );
              if (completed.kind !== "ok") return completed;
              return success(nativeResponse(completed.value));
            } finally {
              await abandonRequest(snapshot);
            }
          }
          if (match.kind === "bad-request") return success(actionReject(400));
          if (match.kind === "method-not-allowed") return success(actionReject(405, match.allow));
        }
        return dispatch(router, snapshot, context);
      }
      const serveNative =
        (peek: boolean) =>
        async (native: Request): Promise<Completion<Response>> => {
          return useResource(token, "server", async () => {
            // Stream-marked routes skip the eager pre-read; anything the peek
            // cannot prove keeps buffered ingress with its pre-dispatch 413.
            // Native action callbacks always buffer: actions never stream.
            let lazy = false;
            if (peek) {
              try {
                lazy =
                  routeKind(router, native.method, normalizedPath(new URL(native.url))) ===
                  "stream";
              } catch {
                lazy = false;
              }
            }
            const snapshot = lazy
              ? await snapshotRequestLazy(native, bodyLimit)
              : await snapshotRequest(native, bodyLimit);
            if (snapshot.kind === "rejected") return success(fixed(snapshot.status));
            bindRequestServer(snapshot.value, server);
            // Body readers live and die in this per-request scope; dispatch
            // abandons an unread live body before the scope drains.
            return withScope(async () => serveSnapshot(native, snapshot.value));
          });
        };
      const guarded = guardCallback(scope, serveNative(true)),
        guardedAction = guardCallback(scope, serveNative(false));
      // Native route callbacks reenter the same asset, ingress, scope and
      // header lifecycle as fetch; fetch stays the canonical fallback.
      const serveOuter =
        (guardedFetch: (native: Request) => Promise<Completion<Response>>) =>
        async (native: Request): Promise<Response> => {
          try {
            const reserved = await assets.serve(native);
            if (reserved) return withPolicy(reserved);
            const completed = await guardedFetch(native);
            if (completed.kind === "ok" && isUpgradedResponse(completed.value))
              return undefined as unknown as Response;
            return withPolicy(completed.kind === "ok" ? completed.value : fixed(500));
          } catch {
            return withPolicy(fixed(500));
          }
        };
      const actionCallback = serveOuter(guardedAction);
      try {
        // Unknown socket data keeps upgrade honest: the adapter passes its own
        // session tag, which the default Server<undefined> type would forbid.
        const server = Bun.serve<unknown>({
          hostname: host,
          port: Number(port),
          ...(tls === undefined ? {} : { tls: { cert: tls.cert, key: tls.key } }),
          ...(actions === undefined
            ? {}
            : {
                routes: Object.fromEntries(
                  bunRouteKeys(actions.table).map((key) => [key, actionCallback]),
                ),
              }),
          websocket: {
            open: (socket: unknown) => serverSocketOpen(socket),
            message: (socket: unknown, message: unknown) => serverSocketMessage(socket, message),
            close: (socket: unknown, code: number, reason: string) =>
              serverSocketClose(socket, code, reason),
            drain: (socket: unknown) => serverSocketDrain(socket),
          },
          fetch: serveOuter(guarded),
          error: () => withPolicy(fixed(500)),
        });
        ready({ server, scope });
        await lifetime;
      } catch {
        ready({});
        return success(undefined);
      }
      return success(undefined);
    });
    // withScope resolves its boxed outcome; a rejection here only reports an
    // entry violation (no owner context or a closing scope) without hanging.
    void scoped.then(
      () => {},
      () => {
        ready({ state: true });
      },
    );
    const outcome = await decided;
    if (outcome.state) throw resourceStateFailure(undefined, origin);
    if (outcome.server === undefined || outcome.scope === undefined) return bind();
    const server = outcome.server;
    let settle!: (completion: Completion<void>) => void;
    const settled = new Promise<Completion<void>>((resolve) => {
      settle = resolve;
    });
    const native: Native = Object.freeze({
      server,
      scope: outcome.scope,
      router,
      bodyLimit,
      shutdownMs,
      settled,
      settle,
    });
    token = registerResource(
      "server",
      native,
      async (): Promise<Completion<void>> => {
        let completion: Completion<void>;
        // Session prompts precede closeResource at every entry (stop, signal):
        // the closer waits for handler leases, so failing pumps inside it
        // would deadlock against handlers parked in read.
        try {
          server.stop(false);
          completion = success(undefined);
        } catch {
          completion = shutdown("stop");
        }
        settle(completion);
        release();
        return completion;
      },
      { scopeManaged: true, shutdownMilliseconds: shutdownMs },
    );
    servers.set(token, native);
    return success(token);
  }
  return Object.freeze({
    async makeConfig(
      host: unknown,
      port: unknown,
      bodyLimit: unknown,
      shutdownMs: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      if (typeof host !== "string") throw new TypeError("invalid compiler host");
      if (
        typeof port !== "bigint" ||
        typeof bodyLimit !== "bigint" ||
        typeof shutdownMs !== "bigint"
      )
        throw new TypeError("invalid compiler config integer");
      if (bodyLimit < 1n || bodyLimit > 67108864n) return invalid("body_limit");
      if (shutdownMs < 0n || shutdownMs > 2147483647n) return invalid("shutdown_ms");
      const token = Object.freeze(Object.create(null));
      configs.set(
        token,
        Object.freeze({ host, port, bodyLimit: Number(bodyLimit), shutdownMs: Number(shutdownMs) }),
      );
      return success(token);
    },
    async makeTlsConfig(
      cert: unknown,
      key: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const text = pemText(cert),
        secret = pemText(key);
      const labels = text === undefined ? undefined : pemLabels(text),
        secrets = secret === undefined ? undefined : pemLabels(secret);
      if (
        labels === undefined ||
        labels.length === 0 ||
        !labels.every((label) => label === "CERTIFICATE")
      )
        return invalid("tls_cert");
      if (
        secrets === undefined ||
        secrets.length !== 1 ||
        (secrets[0] !== "PRIVATE KEY" &&
          secrets[0] !== "RSA PRIVATE KEY" &&
          secrets[0] !== "EC PRIVATE KEY")
      )
        return invalid("tls_key");
      const token = Object.freeze(Object.create(null));
      material.set(
        token,
        Object.freeze({
          cert: new TextEncoder().encode(text),
          key: new TextEncoder().encode(secret),
        }),
      );
      return success(token);
    },
    async start(
      config: unknown,
      router: unknown,
      context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      denyLiveBoundary(context, origin);
      return launch(readConfig(config), router, context);
    },
    async startTls(
      config: unknown,
      router: unknown,
      tls: unknown,
      context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      denyLiveBoundary(context, origin);
      if (!object(tls) || !material.has(tls)) throw resourceStateFailure(undefined, origin);
      const held = material.get(tls)!;
      return launch(readConfig(config), router, context, {
        cert: new Uint8Array(held.cert),
        key: new Uint8Array(held.key),
      });
    },
    async stop(server: unknown, context?: AssertionContext): Promise<Completion<undefined>> {
      denyLiveBoundary(context, origin);
      const native = readServer(server);
      closeServerSessions(native.server);
      const completion = await closeResource(server, "server", {
        milliseconds: native.shutdownMs,
        failure: () => shutdown("deadline"),
      });
      return completion.kind === "ok" ? success(undefined) : completion;
    },
    async wait(server: unknown, context?: AssertionContext): Promise<Completion<undefined>> {
      denyLiveBoundary(context, origin);
      const native = readServer(server);
      // Signal delivery is context-free, so each firing re-enters ownership
      // through the server scope; a closed scope means the close already ran.
      const onSignal = () => {
        try {
          closeServerSessions(native.server);
          const initiate = guardCallback(native.scope, async () => closeResource(server, "server"));
          void initiate().then(
            () => {},
            () => {},
          );
        } catch {}
      };
      process.on("SIGINT", onSignal);
      process.on("SIGTERM", onSignal);
      try {
        const completion = await native.settled;
        return completion.kind === "ok" ? success(undefined) : completion;
      } finally {
        process.off("SIGINT", onSignal);
        process.off("SIGTERM", onSignal);
      }
    },
  });
}
