import { success, failure, invoke, type Completion, type AssertionContext } from "../completion.ts";
import { denyLiveBoundary } from "../assert/context.ts";
import { record, dataArray, recordIdentity } from "../data.ts";
import { byteLength } from "../bytes.ts";
import { createDomainRuntime } from "../domain.ts";
import { resourceStateFailure } from "../failure.ts";
import { CodecIssue } from "../codec/budget.ts";
import { mediaType } from "../transport/media.ts";
import {
  normalizedPath,
  requestSnapshot,
  abandonRequest,
  revokeRequest,
  nativeResponse,
  snapshotBodyBytes,
  ownedResponse,
  isUpgraded,
  UPGRADED_RESPONSE,
} from "./http.ts";
import { decodeActionForm, FormIssue, type FormSchema } from "./form.ts";
import { renderSafe } from "./html.ts";
const origin = Object.freeze({
  source: "can:router",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
export type MountedCallback = (
  request: unknown,
  context?: AssertionContext,
) => Promise<Completion<unknown>>;
type Method = "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | "OPTIONS" | "HEAD";
type Route = Readonly<{
  method: Method;
  source: string;
  path: string;
  stream: boolean;
  callback: MountedCallback;
}>;
type Router = ReadonlyMap<string, ReadonlyMap<string, Route>>;
const routes = new WeakMap<object, Route>(),
  routers = new WeakMap<object, Router>();
const object = (value: unknown): value is object =>
  value !== null && (typeof value === "object" || typeof value === "function");
function read<T>(map: WeakMap<object, T>, value: unknown): T {
  if (!object(value) || !map.has(value)) throw resourceStateFailure(undefined, origin);
  return map.get(value)!;
}
function opaque<T>(map: WeakMap<object, T>, data: T): unknown {
  const token = Object.freeze(Object.create(null));
  map.set(token, data);
  return token;
}
export function isRouterValue(kind: string | undefined, value: unknown): boolean {
  const map = kind === "route" ? routes : kind === "router" ? routers : undefined;
  return map !== undefined && object(value) && map.has(value);
}
function fixed(status: number, allow?: string): Response {
  const headers = new Headers({
    "content-type": "text/plain; charset=utf-8",
    "x-content-type-options": "nosniff",
  });
  if (allow) headers.set("allow", allow);
  return new Response(status === 404 ? "Not Found" : "Method Not Allowed", { status, headers });
}
export async function dispatch(
  router: unknown,
  request: unknown,
  context?: AssertionContext,
): Promise<Completion<Response>> {
  const table = read(routers, router),
    snapshot = requestSnapshot(request),
    methods = table.get(snapshot.path);
  const route = methods?.get(snapshot.method);
  // A rejected route admits no handler work, so no per-request child can hold
  // the token: abandon any unread body, then end the capability here. The
  // server boundary repeats the revocation idempotently after drainage.
  if (route === undefined) {
    try {
      return success(
        methods === undefined ? fixed(404) : fixed(405, [...methods.keys()].sort().join(", ")),
      );
    } finally {
      await abandonRequest(request);
      revokeRequest(request);
    }
  }
  try {
    const completed = await invoke(() => route.callback(request, context), origin);
    if (completed.kind !== "ok") return completed;
    // Upgraded requests hold a live socket; the server answers no HTTP reply.
    if (isUpgraded(request)) return success(UPGRADED_RESPONSE);
    return success(nativeResponse(completed.value, snapshot.method === "HEAD"));
  } finally {
    await abandonRequest(request);
  }
}
// Ingress peek: the server consults the table before snapshotting so
// stream-marked routes skip the eager bounded pre-read. Anything the
// peek cannot prove falls back to buffered ingress downstream.
export function routeKind(
  router: unknown,
  method: string,
  path: string,
): "stream" | "buffered" | undefined {
  const table = read(routers, router),
    methods = table.get(path),
    route = methods?.get(method);
  return route === undefined ? undefined : route.stream ? "stream" : "buffered";
}
function constructRoute(
  fail: (reason: string) => Completion<never>,
  method: Method,
  source: string,
  callback: MountedCallback,
): Completion<unknown> {
  // Source checking additionally requires a static path and named exact callback.
  // oxlint-disable no-control-regex -- Static paths reject C0 controls, space, DEL, and route metacharacters.
  if (
    !source.isWellFormed() ||
    !source.startsWith("/") ||
    source.startsWith("//") ||
    /[\x00-\x20\x7f\\?#*]/.test(source) ||
    source.split("/").some((segment) => segment.startsWith(":"))
  )
    return fail("path");
  // oxlint-enable no-control-regex
  let path: string;
  try {
    path = normalizedPath(new URL(source, "http://can.invalid"));
  } catch (cause) {
    if (!(cause instanceof URIError) && !(cause instanceof TypeError)) throw cause;
    return fail("path");
  }
  if (path === "/__can" || path.startsWith("/__can/")) return fail("path");
  if (typeof callback !== "function") throw new TypeError("invalid mounted callback");
  return success(opaque(routes, Object.freeze({ method, source, path, stream: false, callback })));
}
export function createRouter(
  domain: ReturnType<typeof createDomainRuntime>,
  types: Readonly<{ invalid: string; duplicate: string; ambiguous: string }>,
) {
  const error = (type: string, fields: readonly (readonly [string, unknown])[]) =>
    failure(domain.create(type, record(type, fields), origin));
  function route(method: Method, source: string, callback: MountedCallback): Completion<unknown> {
    return constructRoute(
      (reason) => error(types.invalid, [["reason", reason]]),
      method,
      source,
      callback,
    );
  }
  return Object.freeze({
    async get(
      path: string,
      callback: MountedCallback,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      return route("GET", path, callback);
    },
    async post(
      path: string,
      callback: MountedCallback,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      return route("POST", path, callback);
    },
    async put(
      path: string,
      callback: MountedCallback,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      return route("PUT", path, callback);
    },
    async patch(
      path: string,
      callback: MountedCallback,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      return route("PATCH", path, callback);
    },
    async delete(
      path: string,
      callback: MountedCallback,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      return route("DELETE", path, callback);
    },
    async options(
      path: string,
      callback: MountedCallback,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      return route("OPTIONS", path, callback);
    },
    async head(
      path: string,
      callback: MountedCallback,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      return route("HEAD", path, callback);
    },
    async stream(input: unknown, _context?: AssertionContext): Promise<Completion<unknown>> {
      const route = read(routes, input);
      return success(opaque(routes, Object.freeze({ ...route, stream: true })));
    },
    async make(input: unknown, _context?: AssertionContext): Promise<Completion<unknown>> {
      const table = new Map<string, Map<string, Route>>();
      for (const token of dataArray(input)) {
        const route = read(routes, token);
        let methods = table.get(route.path);
        if (!methods) {
          methods = new Map();
          table.set(route.path, methods);
        }
        const previous = methods.get(route.method);
        if (previous)
          return previous.source === route.source
            ? error(types.duplicate, [
                ["method", route.method],
                ["path", route.path],
              ])
            : error(types.ambiguous, [
                ["first", previous.source],
                ["second", route.source],
              ]);
        methods.set(route.method, route);
      }
      return success(opaque(routers, table));
    },
  });
}
export type FormActionSite = Readonly<{
  action: string;
  method: string;
  path: string;
  form: FormSchema;
  cases: readonly Readonly<{ leaf: string; status: number }>[];
  rejected: string;
  rawEntry: string;
  issue: string;
}>;
type FormCallable = (value: unknown, context?: AssertionContext) => Promise<Completion<unknown>>;
// HTML action adapter for form-body actions. One serve call binds the three
// server callables of a checked action — the valid-wire handler, the outcome
// renderer and the structural-422 renderer — into a mounted route. Ingress
// failures before parsing (media, body, encoding) receive compiler-owned
// fixed responses; structural violations decode into a rejected value for
// the 422 renderer; valid wire flows through the handler and the outcome
// renderer with the case status of the result leaf.
export function createFormActions(
  domain: ReturnType<typeof createDomainRuntime>,
  types: Readonly<{ invalidRoute: string }>,
) {
  const invalid = (reason: string) =>
    failure(
      domain.create(types.invalidRoute, record(types.invalidRoute, [["reason", reason]]), origin),
    );
  function readSite(site: unknown): FormActionSite {
    const valid =
      site !== null &&
      typeof site === "object" &&
      typeof (site as FormActionSite).action === "string" &&
      (site as FormActionSite).method === "POST" &&
      typeof (site as FormActionSite).path === "string" &&
      (site as FormActionSite).form !== null &&
      typeof (site as FormActionSite).form === "object" &&
      Array.isArray((site as FormActionSite).cases) &&
      (site as FormActionSite).cases.every(
        (entry) =>
          entry !== null &&
          typeof entry === "object" &&
          typeof (entry as { leaf: unknown }).leaf === "string" &&
          Number.isInteger((entry as { status: unknown }).status) &&
          (entry as { status: number }).status >= 200 &&
          (entry as { status: number }).status <= 599,
      ) &&
      typeof (site as FormActionSite).rejected === "string" &&
      typeof (site as FormActionSite).rawEntry === "string" &&
      typeof (site as FormActionSite).issue === "string";
    if (!valid) throw new TypeError("invalid compiler form action");
    return site as FormActionSite;
  }
  function formMedia(request: unknown): boolean {
    const value = requestSnapshot(request).headers.find(([name]) => name === "content-type")?.[1];
    if (value === undefined) return false;
    try {
      const parsed = mediaType(value);
      return (
        parsed.type === "application/x-www-form-urlencoded" &&
        [...parsed.parameters].every(
          ([key, value]) => key === "charset" && value.toLowerCase() === "utf-8",
        )
      );
    } catch (cause) {
      if (cause instanceof CodecIssue) return false;
      throw cause;
    }
  }
  return Object.freeze({
    async serve(
      action: unknown,
      outcome: unknown,
      structural: unknown,
      site: unknown,
      handler: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      if (typeof action !== "string") throw new TypeError("invalid compiler form action");
      const checked = readSite(site);
      // The frozen site carries the fully qualified action identity while
      // the invocation carries the authored spelling (bare or
      // package-qualified), so agreement compares the authored suffix.
      // The checker resolves the spelling before splicing the site, and
      // a wrong action name still disagrees on its tail.
      if (checked.action !== action && !checked.action.endsWith("::" + action))
        throw new TypeError("invalid compiler form action");
      if (
        typeof outcome !== "function" ||
        typeof structural !== "function" ||
        typeof handler !== "function"
      )
        throw new TypeError("invalid compiler form renderer");
      const renderOutcome = outcome as FormCallable,
        renderStructural = structural as FormCallable,
        runHandler = handler as FormCallable;
      const callback: MountedCallback = async (request, context) => {
        denyLiveBoundary(context, origin);
        const body = await snapshotBodyBytes(request);
        if (body.kind === "limited") return success(ownedResponse(413, "Payload Too Large", false));
        if (body.kind !== "bytes") return success(ownedResponse(400, "Bad Request", false));
        if (!formMedia(request)) return success(ownedResponse(400, "Bad Request", false));
        let decoded: ReturnType<typeof decodeActionForm>;
        try {
          decoded = decodeActionForm(checked.form, body.value, Number(byteLength(body.value)), {
            rejected: checked.rejected,
            rawEntry: checked.rawEntry,
            issue: checked.issue,
          });
        } catch (cause) {
          if (cause instanceof CodecIssue || cause instanceof FormIssue)
            return success(ownedResponse(400, "Bad Request", false));
          throw cause;
        }
        if (decoded.kind === "rejected") {
          const rendered = await invoke(() => renderStructural(decoded.value, context), origin);
          if (rendered.kind !== "ok") return rendered;
          return success(ownedResponse(422, renderSafe(rendered.value), true));
        }
        const handled = await invoke(() => runHandler(decoded.value, context), origin);
        if (handled.kind !== "ok") return handled;
        const leaf = recordIdentity(handled.value),
          kase = checked.cases.find((entry) => entry.leaf === leaf);
        if (leaf === undefined || kase === undefined)
          throw new TypeError("invalid compiler action result");
        const rendered = await invoke(() => renderOutcome(handled.value, context), origin);
        if (rendered.kind !== "ok") return rendered;
        return success(ownedResponse(kase.status, renderSafe(rendered.value), true));
      };
      return constructRoute((reason) => invalid(reason), "POST", checked.path, callback);
    },
  });
}
