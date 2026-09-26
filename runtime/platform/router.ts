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
  requestNativeRequest,
  abandonRequest,
  revokeRequest,
  nativeResponse,
  snapshotBodyBytes,
  ownedResponse,
  isUpgraded,
  UPGRADED_RESPONSE,
} from "./http.ts";
import { noteRequestSource } from "../transport/request-report.ts";
import { decodeActionForm, FormIssue, type FormSchema } from "./form.ts";
import { renderSafe } from "./html.ts";
import {
  compileActionRoutes,
  matchActionRoute,
  actionTargetSegments,
  bunRouteKeys,
  isActionRouteBinding,
  readActionRouteBinding,
  ActionRouteIssue,
  type ActionRouteTable,
  type ActionRouteSource,
  type ActionDispatchCallback,
} from "./action-routes.ts";
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
// Combined dispatch value: the legacy exact table plus, when the program
// mounts checked actions, the compiled capture table with per-action
// callbacks. Dispatch serves an exact legacy match first (static
// precedence), then a strict captured match, then a union 404/405.
type Router = Readonly<{
  exact: ReadonlyMap<string, ReadonlyMap<string, Route>>;
  actions?: Readonly<{
    table: ActionRouteTable;
    callbacks: ReadonlyMap<string, ActionDispatchCallback>;
  }>;
}>;
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
  if (kind === "route") return object(value) && (routes.has(value) || isActionRouteBinding(value));
  if (kind === "router") return object(value) && routers.has(value);
  return false;
}
function fixed(status: 400 | 404 | 405, allow?: string): Response {
  const headers = new Headers({
    "content-type": "text/plain; charset=utf-8",
    "x-content-type-options": "nosniff",
  });
  if (status === 405 && allow) headers.set("allow", allow);
  const text = status === 404 ? "Not Found" : status === 400 ? "Bad Request" : "Method Not Allowed";
  return new Response(text, { status, headers });
}
async function rejected(response: Response, request: unknown): Promise<Completion<Response>> {
  // A rejected route admits no handler work, so no per-request child can hold
  // the token: abandon any unread body, then end the capability here. The
  // server boundary repeats the revocation idempotently after drainage.
  try {
    return success(response);
  } finally {
    await abandonRequest(request);
    revokeRequest(request);
  }
}
export async function dispatch(
  router: unknown,
  request: unknown,
  context?: AssertionContext,
): Promise<Completion<Response>> {
  const table = read(routers, router),
    snapshot = requestSnapshot(request),
    methods = table.exact.get(snapshot.path);
  const route = methods?.get(snapshot.method);
  // Static precedence: an exact legacy match wins over any captured action
  // before strict target validation can rule the raw spelling out.
  if (route !== undefined) {
    // Authored mount metadata only: the route method and static source
    // come from the validated mount, never the request line.
    noteRequestSource(requestNativeRequest(request), `route:${route.method} ${route.source}`);
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
  const actions = table.actions;
  if (actions === undefined) {
    return rejected(
      methods === undefined ? fixed(404) : fixed(405, [...methods.keys()].sort().join(", ")),
      request,
    );
  }
  // Target-level malformation rules the raw pathname out before the action
  // table is consulted, so no protected handler observes it as data.
  if (actionTargetSegments(snapshot.rawPath) === undefined) return rejected(fixed(400), request);
  const match = matchActionRoute(actions.table, snapshot.method, snapshot.rawPath);
  if (match.kind === "match") {
    const callback = actions.callbacks.get(match.identity);
    if (callback === undefined) throw new TypeError("invalid compiler action route");
    noteRequestSource(requestNativeRequest(request), "action:" + match.identity);
    try {
      const completed = await invoke(() => callback(request, match.captures, context), origin);
      if (completed.kind !== "ok") return completed;
      if (isUpgraded(request)) return success(UPGRADED_RESPONSE);
      return success(nativeResponse(completed.value, snapshot.method === "HEAD"));
    } finally {
      await abandonRequest(request);
    }
  }
  // The target survived strict decoding, so a bad request here is a
  // route-relative typing failure (a structural match with untypable
  // captures): still a 400, still before any handler entry.
  if (match.kind === "bad-request") return rejected(fixed(400), request);
  // Union classification: either table can claim the path under another
  // method. The Allow set merges both claims in sorted order.
  const allow = new Set<string>(methods === undefined ? [] : [...methods.keys()]);
  if (match.kind === "method-not-allowed")
    for (const method of match.allow.split(", ")) allow.add(method);
  if (allow.size === 0) return rejected(fixed(404), request);
  return rejected(fixed(405, [...allow].sort().join(", ")), request);
}
// Ingress peek: the server consults the table before snapshotting so
// stream-marked routes skip the eager bounded pre-read. Anything the
// peek cannot prove falls back to buffered ingress downstream.
// Native action callbacks always buffer: actions never stream.
export function routeKind(
  router: unknown,
  method: string,
  path: string,
): "stream" | "buffered" | undefined {
  const table = read(routers, router),
    methods = table.exact.get(path),
    route = methods?.get(method);
  return route === undefined ? undefined : route.stream ? "stream" : "buffered";
}
// Native registration keys for the router-carried action table, so server
// start can register Bun routes over the same canonical dispatch.
export function routerActionKeys(router: unknown): readonly string[] {
  const table = read(routers, router);
  return table.actions === undefined ? Object.freeze([]) : bunRouteKeys(table.actions.table);
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
      const mounted: { entry: ActionRouteSource; callback: ActionDispatchCallback }[] = [];
      for (const token of dataArray(input)) {
        if (isActionRouteBinding(token)) {
          const binding = readActionRouteBinding(token);
          mounted.push({ entry: binding.entry, callback: binding.callback });
          continue;
        }
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
      if (mounted.length === 0) return success(opaque(routers, { exact: table }));
      let compiled: ActionRouteTable;
      try {
        compiled = compileActionRoutes(mounted.map((binding) => binding.entry));
      } catch (cause) {
        if (!(cause instanceof ActionRouteIssue)) throw cause;
        if (cause.code === "duplicate-route")
          return error(types.duplicate, [
            ["method", cause.data.method ?? ""],
            ["path", cause.data.path ?? ""],
          ]);
        if (cause.code === "ambiguous-route")
          return error(types.ambiguous, [
            ["first", cause.data.first ?? ""],
            ["second", cause.data.second ?? ""],
          ]);
        return error(types.invalid, [["reason", "path"]]);
      }
      // A legacy exact route and an all-static action route on one method
      // and decoded path are the same route: refuse the duplicate. Legacy
      // statics overlapping captured actions stay legal; exact-first
      // dispatch serves the static spelling.
      for (const route of compiled.routes) {
        const statics: string[] = [];
        let allStatic = true;
        for (const segment of route.segments) {
          if (segment.kind !== "static") {
            allStatic = false;
            break;
          }
          statics.push(segment.decoded);
        }
        if (!allStatic) continue;
        const decoded = "/" + statics.join("/");
        if (table.get(decoded)?.has(route.method))
          return error(types.duplicate, [
            ["method", route.method],
            ["path", decoded],
          ]);
      }
      const callbacks = new Map<string, ActionDispatchCallback>();
      for (const binding of mounted) callbacks.set(binding.entry.identity, binding.callback);
      return success(opaque(routers, { exact: table, actions: { table: compiled, callbacks } }));
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
        if (!formMedia(request))
          return success(ownedResponse(415, "Unsupported Media Type", false));
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
