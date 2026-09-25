import { invoke, success, failure, type Completion, type AssertionContext } from "../completion.ts";
import { array, record, recordIdentity } from "../data.ts";
import { ownBytes, copyBytes, byteLength } from "../bytes.ts";
import { createDomainRuntime } from "../domain.ts";
import { CodecIssue } from "../codec/budget.ts";
import { decodeJSON, encodeJSON, type Schema } from "../codec/json.ts";
import { graph } from "../codec/project.ts";
import { jsonRequestMedia, responseMedia } from "../transport/media.ts";
import { requestSnapshot, createResponses } from "./http.ts";
import {
  compileActionRoutes,
  buildActionURL,
  actionTemplate,
  ActionRouteIssue,
} from "./action-routes.ts";
import type { MountedCallback } from "./router.ts";

const origin = Object.freeze({
  source: "can:action-json",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

// POST JSON wire cap shared by the server adapter and the fetch consumer.
// Oversize requests receive a fixed 413; the consumer refuses to send them.
export const ACTION_JSON_BODY_LIMIT = 8192;

// EmittedActionEntry is one $canActions row as frozen by the compiler: the
// typed action::request metadata (captures plus the json body wire contract)
// and the typed action::response metadata (result plus the case table and
// the shared JSON wire schema of the result variant).
export type EmittedActionEntry = Readonly<{
  identity: string;
  method: string;
  path: string;
  captures: readonly Readonly<{ name: string; type: string }>[];
  body?: Readonly<{ mode: string; type: string; schema?: unknown; form?: unknown }>;
  handler: string;
  result: string;
  cases: readonly Readonly<{ leaf: string; status: number }>[];
  responseSchema?: unknown;
}>;

export type ActionJsonCase = Readonly<{ leaf: string; status: number }>;
export type ActionJsonHandler = (
  inputs: readonly unknown[],
  context?: AssertionContext,
) => Promise<Completion<unknown>>;
export type ActionJsonRouter = Readonly<{
  get: (path: string, callback: MountedCallback) => Promise<Completion<unknown>>;
  post: (path: string, callback: MountedCallback) => Promise<Completion<unknown>>;
}>;
type CheckedEntry = Readonly<{
  identity: string;
  method: "GET" | "POST";
  path: string;
  request: Schema | null;
  response: Schema;
  cases: readonly ActionJsonCase[];
}>;

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object";
}

function checkedSchema(entry: string, slot: string, value: unknown): Schema {
  if (!isRecord(value) || typeof value.root !== "string" || !Array.isArray(value.nodes))
    throw new TypeError(`action ${entry} carries no ${slot} JSON schema`);
  for (const node of value.nodes) {
    if (
      !isRecord(node) ||
      typeof node.identity !== "string" ||
      typeof node.kind !== "string" ||
      typeof node.name !== "string"
    )
      throw new TypeError(`action ${entry} carries a malformed ${slot} JSON schema`);
  }
  const schema = value as unknown as Schema;
  // Fail fast on duplicate nodes and an unresolvable root; the shared codec
  // would reject the same shape later with a less actionable diagnostic.
  graph(schema)(schema.root);
  return schema;
}

function checkedCases(entry: string, cases: unknown): readonly ActionJsonCase[] {
  if (!Array.isArray(cases) || cases.length === 0)
    throw new TypeError(`action ${entry} carries no response cases`);
  const seen = new Set<string>();
  return cases.map((kase) => {
    if (
      !isRecord(kase) ||
      typeof kase.leaf !== "string" ||
      kase.leaf === "" ||
      typeof kase.status !== "number" ||
      !Number.isInteger(kase.status) ||
      kase.status < 200 ||
      kase.status > 599 ||
      kase.status === 204 ||
      kase.status === 205 ||
      kase.status === 304
    )
      throw new TypeError(`action ${entry} carries a malformed response case`);
    if (seen.has(kase.leaf)) throw new TypeError(`action ${entry} repeats case leaf ${kase.leaf}`);
    seen.add(kase.leaf);
    return { leaf: kase.leaf, status: kase.status };
  });
}

function checkedEntry(entry: EmittedActionEntry): CheckedEntry {
  if (!isRecord(entry) || typeof entry.identity !== "string" || entry.identity === "")
    throw new TypeError("action table carries a malformed entry");
  const identity = entry.identity;
  if (entry.method !== "GET" && entry.method !== "POST")
    throw new TypeError(`action ${identity} names an unknown method`);
  if (typeof entry.path !== "string" || entry.path === "")
    throw new TypeError(`action ${identity} names no path`);
  if (!Array.isArray(entry.captures))
    throw new TypeError(`action ${identity} carries malformed captures`);
  if (entry.captures.length !== 0)
    throw new TypeError(
      `action ${identity} declares path captures; mount serves exact JSON paths only`,
    );
  const cases = checkedCases(identity, entry.cases);
  const response = checkedSchema(identity, "response", entry.responseSchema);
  if (entry.method === "GET") {
    if (entry.body !== undefined && entry.body !== null)
      throw new TypeError(`action ${identity} is a bodyless GET action with a body contract`);
    return { identity, method: entry.method, path: entry.path, request: null, response, cases };
  }
  if (!isRecord(entry.body))
    throw new TypeError(`action ${identity} is a POST action with no body`);
  if (entry.body.mode !== "json")
    throw new TypeError(`action ${identity} is not a JSON action; mount serves JSON bodies only`);
  if (typeof entry.body.type !== "string" || entry.body.type === "")
    throw new TypeError(`action ${identity} carries no request wire type`);
  const request = checkedSchema(identity, "request", entry.body.schema);
  return { identity, method: entry.method, path: entry.path, request, response, cases };
}

export function createJsonActions(
  domain: ReturnType<typeof createDomainRuntime>,
  types: Readonly<{
    invalid: string;
    invalidData: string;
    close: string;
    writeFailed: string;
    limit: string;
  }>,
) {
  const responses = createResponses(domain, types);
  async function fixed(status: number, text: string): Promise<Completion<unknown>> {
    const code = await responses.makeBodyStatus(BigInt(status));
    if (code.kind !== "ok") return code;
    const headers = await responses.emptyHeaders();
    if (headers.kind !== "ok") return headers;
    return responses.text(code.value, headers.value, text);
  }
  function buildCallback(entry: CheckedEntry, handler: ActionJsonHandler): MountedCallback {
    return async (request: unknown, context?: AssertionContext): Promise<Completion<unknown>> => {
      const snapshot = requestSnapshot(request);
      let inputs: readonly unknown[];
      if (entry.method === "GET") {
        // GET-load is bodyless: any actual body bytes are a client violation,
        // even though standard clients cannot easily send them.
        if (byteLength(snapshot.body) !== 0n) return fixed(400, "Bad Request");
        inputs = [];
      } else {
        const media = snapshot.headers.find(([name]) => name === "content-type")?.[1];
        if (media === undefined || !jsonRequestMedia(media))
          return fixed(415, "Unsupported Media Type");
        const length = byteLength(snapshot.body);
        if (length > BigInt(ACTION_JSON_BODY_LIMIT)) return fixed(413, "Payload Too Large");
        if (length === 0n) return fixed(400, "Bad Request");
        try {
          inputs = [decodeJSON(entry.request!, snapshot.body, ACTION_JSON_BODY_LIMIT)];
        } catch (cause) {
          if (!(cause instanceof CodecIssue)) throw cause;
          // Malformed, duplicate, over-cap or mistyped JSON never reaches the
          // protected handler; the fixed 400 carries no wire detail.
          return fixed(400, "Bad Request");
        }
      }
      const completed = await invoke(() => handler(inputs, context), origin);
      if (completed.kind !== "ok") return fixed(500, "Internal Server Error");
      const leaf = recordIdentity(completed.value);
      const kase = entry.cases.find((row) => row.leaf === leaf);
      if (leaf === undefined || kase === undefined) return fixed(500, "Internal Server Error");
      // Every domain outcome renders its declared finite status with a JSON
      // representation; the adapter never invents a status or serves HTML.
      const code = await responses.makeBodyStatus(BigInt(kase.status));
      if (code.kind !== "ok") return fixed(500, "Internal Server Error");
      const headers = await responses.emptyHeaders();
      if (headers.kind !== "ok") return fixed(500, "Internal Server Error");
      const rendered = await responses.json(
        entry.response,
        code.value,
        headers.value,
        completed.value,
      );
      if (rendered.kind !== "ok") return fixed(500, "Internal Server Error");
      return rendered;
    };
  }
  function checkedHandler(identity: string, handler: ActionJsonHandler): ActionJsonHandler {
    if (typeof handler !== "function")
      throw new TypeError(`action ${identity} has no callable handler`);
    return handler;
  }
  return Object.freeze({
    // Callback builds the mounted handler for one exact-path JSON entry
    // without pairing a route, for direct dispatch and route-table embedding.
    callback(entry: EmittedActionEntry, handler: ActionJsonHandler): MountedCallback {
      const checked = checkedEntry(entry);
      return buildCallback(checked, checkedHandler(checked.identity, handler));
    },
    // Mount pairs one checked JSON entry with its total handler on the exact
    // static path, using the verb the entry declares. Router failures (an
    // invalid path) propagate as catalogue completions; malformed metadata,
    // capture routes and form bodies are mount-time TypeErrors.
    async mount(
      router: ActionJsonRouter,
      entry: EmittedActionEntry,
      handler: ActionJsonHandler,
    ): Promise<Completion<unknown>> {
      const checked = checkedEntry(entry);
      const run = buildCallback(checked, checkedHandler(checked.identity, handler));
      return checked.method === "GET"
        ? router.get(checked.path, run)
        : router.post(checked.path, run);
    },
  });
}

// ActionFetchResult is the fetch consumer's outcome vocabulary. Transport,
// abort, codec and unexpected-status failures stay distinct from finite
// domain cases; createJsonActionFetch lowers these names into the checked
// per-action failure bound.
export type ActionFetchResult =
  | Readonly<{ kind: "ok"; status: number; leaf: string; value: unknown }>
  | Readonly<{ kind: "transport"; phase: "connect" | "body" | "protocol" }>
  | Readonly<{ kind: "aborted" }>
  | Readonly<{ kind: "codec"; path: string; reason: string }>
  | Readonly<{ kind: "unexpected_status"; status: number }>;

export type ActionFetchInput = Readonly<{
  url: string;
  method: "GET" | "POST";
  request?: Schema;
  body?: unknown;
  response: Schema;
  cases: readonly ActionJsonCase[];
  signal?: AbortSignal;
  requestLimit?: number;
  responseLimit?: number;
}>;

function fetchFailed(cause: unknown): boolean {
  return (
    cause instanceof TypeError ||
    cause instanceof DOMException ||
    (cause instanceof Error && "code" in cause)
  );
}

function fetchAborted(cause: unknown, signal?: AbortSignal): boolean {
  if (signal?.aborted) return true;
  return (
    typeof cause === "object" &&
    cause !== null &&
    (cause as { name?: unknown }).name === "AbortError"
  );
}

// fetchJsonAction consumes one JSON action over native fetch, which runs
// unchanged on Bun and in browsers. The finite case table decides statuses:
// any final status outside the table is unexpected_status without decoding
// the body, and a decodable body under a finite status that still fails the
// response contract is codec. Redirects follow fetch defaults; only the
// final status is classified.
export async function fetchJsonAction(input: ActionFetchInput): Promise<ActionFetchResult> {
  const cases = checkedCases("fetch", input.cases);
  const response = checkedSchema("fetch", "response", input.response);
  const limit = input.responseLimit ?? ACTION_JSON_BODY_LIMIT;
  if (!Number.isSafeInteger(limit) || limit < 1)
    throw new TypeError("action fetch needs a positive response limit");
  let encoded: Uint8Array<ArrayBuffer> | undefined;
  if (input.method === "POST") {
    if (input.body === undefined) throw new TypeError("action fetch POST needs a body value");
    if (input.request === undefined)
      throw new TypeError("action fetch POST needs a request schema");
    const schema = checkedSchema("fetch", "request", input.request);
    let bytes;
    try {
      bytes = encodeJSON(schema, input.body);
    } catch (cause) {
      if (!(cause instanceof CodecIssue)) throw cause;
      throw new TypeError("action fetch POST body is not request-admissible");
    }
    encoded = new Uint8Array(copyBytes(bytes, origin));
    const requestLimit = input.requestLimit ?? ACTION_JSON_BODY_LIMIT;
    if (!Number.isSafeInteger(requestLimit) || requestLimit < 1)
      throw new TypeError("action fetch needs a positive request limit");
    if (encoded.byteLength > requestLimit)
      throw new TypeError("action fetch POST body exceeds the wire limit");
  } else {
    if (input.body !== undefined) throw new TypeError("action fetch GET carries no body");
    if (input.request !== undefined)
      throw new TypeError("action fetch GET carries no request schema");
  }
  let received: Response;
  try {
    received = await fetch(input.url, {
      method: input.method,
      headers:
        input.method === "POST"
          ? { "content-type": "application/json", accept: "application/json" }
          : { accept: "application/json" },
      body: encoded,
      // Same-origin credentials: the emitted browser asset runs on the
      // serving origin, and protected actions authenticate through the
      // session cookie. Cross-origin requests still omit credentials.
      credentials: "same-origin",
      signal: input.signal,
    });
  } catch (cause) {
    if (fetchAborted(cause, input.signal)) return { kind: "aborted" };
    if (fetchFailed(cause)) return { kind: "transport", phase: "connect" };
    throw cause;
  }
  if (!Number.isInteger(received.status) || received.status < 200 || received.status > 599) {
    try {
      await received.body?.cancel();
    } catch {}
    return { kind: "transport", phase: "protocol" };
  }
  if (!cases.some((kase) => kase.status === received.status)) {
    try {
      await received.body?.cancel();
    } catch {}
    return { kind: "unexpected_status", status: received.status };
  }
  const chunks: Uint8Array[] = [];
  let size = 0;
  if (received.body !== null) {
    const reader = received.body.getReader();
    try {
      for (;;) {
        let next;
        try {
          next = await reader.read();
        } catch (cause) {
          if (fetchAborted(cause, input.signal)) return { kind: "aborted" };
          if (fetchFailed(cause)) return { kind: "transport", phase: "body" };
          throw cause;
        }
        if (next.done) break;
        if (next.value.byteLength > limit - size) {
          try {
            await reader.cancel();
          } catch {}
          return { kind: "codec", path: "", reason: "byte_limit" };
        }
        size += next.value.byteLength;
        if (next.value.byteLength !== 0) chunks.push(new Uint8Array(next.value));
      }
    } finally {
      reader.releaseLock();
    }
  }
  const raw = new Uint8Array(size);
  let offset = 0;
  for (const chunk of chunks) {
    raw.set(chunk, offset);
    offset += chunk.byteLength;
  }
  try {
    responseMedia(received.headers.get("content-type") ?? undefined, true);
  } catch (cause) {
    if (!(cause instanceof CodecIssue)) throw cause;
    return { kind: "codec", path: cause.path, reason: cause.reason };
  }
  let value: unknown;
  try {
    value = decodeJSON(response, ownBytes(raw), limit);
  } catch (cause) {
    if (!(cause instanceof CodecIssue)) throw cause;
    return { kind: "codec", path: cause.path, reason: cause.reason };
  }
  const leaf = recordIdentity(value);
  if (
    leaf === undefined ||
    !cases.some((kase) => kase.leaf === leaf && kase.status === received.status)
  )
    return { kind: "codec", path: "", reason: "variant_tag" };
  return { kind: "ok", status: received.status, leaf, value };
}

// JsonFetchSite is the frozen per-action client contract the compiler
// splices into one fetch_json_get/fetch_json_post invocation: the resolved
// operation identity, the route template with its captures, the POST
// request schema, the shared response schema, and the finite case table.
// The checker owns the spelling-to-identity mapping; the adapter
// revalidates every shape it consumes.
export type JsonFetchSite = Readonly<{
  action: string;
  method: "GET" | "POST";
  path: string;
  captures: readonly Readonly<{ name: string; type: string }>[];
  request?: unknown;
  response: unknown;
  cases: readonly ActionJsonCase[];
  limit?: unknown;
}>;

export type JsonFetchTypes = Readonly<{
  transport: string;
  invalidRequest: string;
  bodyLimit: string;
  statusError: string;
  invalidData: string;
  header: string;
}>;

type CheckedFetchSite = Readonly<{
  action: string;
  method: "GET" | "POST";
  path: string;
  captures: readonly Readonly<{ name: string; type: string }>[];
  request: Schema | null;
  response: Schema;
  cases: readonly ActionJsonCase[];
  // Declared JSON request budget in bytes, or null when the site
  // carries none and the default wire cap applies.
  limit: number | null;
}>;

function checkedFetchSite(site: unknown): CheckedFetchSite {
  if (!isRecord(site) || typeof site.action !== "string" || site.action === "")
    throw new TypeError("fetch site carries no action identity");
  const action = site.action;
  if (site.method !== "GET" && site.method !== "POST")
    throw new TypeError(`fetch site ${action} names an unknown method`);
  if (typeof site.path !== "string" || site.path === "")
    throw new TypeError(`fetch site ${action} names no path`);
  if (!Array.isArray(site.captures))
    throw new TypeError(`fetch site ${action} carries malformed captures`);
  for (const row of site.captures) {
    if (
      !isRecord(row) ||
      typeof row.name !== "string" ||
      row.name === "" ||
      (row.type !== "str" && row.type !== "int")
    )
      throw new TypeError(`fetch site ${action} carries a malformed capture`);
  }
  const captures = site.captures as CheckedFetchSite["captures"];
  const cases = checkedCases(action, site.cases);
  const response = checkedSchema(action, "response", site.response);
  let limit: number | null = null;
  if (site.limit !== undefined && site.limit !== null) {
    if (!Number.isSafeInteger(site.limit) || (site.limit as number) < 1)
      throw new TypeError(`fetch site ${action} carries a malformed request budget`);
    limit = site.limit as number;
  }
  if (site.method === "GET") {
    if (site.request !== undefined && site.request !== null)
      throw new TypeError(`fetch site ${action} is a bodyless GET site with a request contract`);
    return {
      action,
      method: site.method,
      path: site.path,
      captures,
      request: null,
      response,
      cases,
      limit,
    };
  }
  if (site.request === undefined || site.request === null)
    throw new TypeError(`fetch site ${action} is a POST site with no request contract`);
  const request = checkedSchema(action, "request", site.request);
  return {
    action,
    method: site.method,
    path: site.path,
    captures,
    request,
    response,
    cases,
    limit,
  };
}

// createJsonActionFetch lowers checked JSON action calls to canonical URL
// building plus native fetch. Emitted invocations pass the authored inputs
// (action name, captures in path order, POST body), then the spliced site,
// then the assertion context; the context travels positionally and is
// otherwise unused. Method, operation identity, codecs, sizes and the
// actual status all validate here: transport, abort, codec and
// unexpected-status outcomes map into the declared failure bound and never
// surface as domain cases.
export function createJsonActionFetch(
  domain: ReturnType<typeof createDomainRuntime>,
  types: JsonFetchTypes,
) {
  const fail = (
    identity: string,
    fields: [string, unknown][],
    operation: string,
  ): Completion<never> =>
    failure(
      domain.create(identity, record(identity, fields), origin, undefined, {
        boundary: "native",
        operation,
      }),
    );
  async function run(
    method: "GET" | "POST",
    name: unknown,
    tail: readonly unknown[],
  ): Promise<Completion<unknown>> {
    if (typeof name !== "string" || name === "")
      throw new TypeError("fetch action name is not a string");
    const rest = [...tail];
    rest.pop();
    const site = checkedFetchSite(rest.pop());
    if (site.method !== method)
      throw new TypeError(`fetch site ${site.action} disagrees with its verb`);
    const want = site.captures.length + (method === "POST" ? 1 : 0);
    if (rest.length !== want) throw new TypeError(`fetch ${site.action} takes ${want} inputs`);
    const captureValues = rest.slice(0, site.captures.length);
    const body = method === "POST" ? rest[rest.length - 1] : undefined;
    let url: string;
    try {
      const table = compileActionRoutes([
        {
          identity: site.action,
          method: site.method,
          path: actionTemplate(site.action, site.path),
          captures: site.captures,
        },
      ]);
      url = buildActionURL(table, site.action, captureValues);
    } catch (cause) {
      // Capture arity and types are checked statically; only runtime
      // capture data can still fail the builder. Template faults are
      // compiler bugs and keep throwing.
      if (
        cause instanceof ActionRouteIssue &&
        (cause.code === "unknown-action" ||
          cause.code === "capture-arity" ||
          cause.code === "capture-type" ||
          cause.code === "capture-value")
      )
        return fail(types.invalidRequest, [["reason", cause.code]], site.action);
      throw cause;
    }
    // Same-origin by construction: the canonical builder emits a relative
    // path, which the served content-security-policy (connect-src 'self')
    // admits. Anything else fails closed before any byte is sent.
    if (!url.startsWith("/") || url.startsWith("//"))
      return fail(types.invalidRequest, [["reason", "origin"]], site.action);
    if (method === "POST") {
      let encoded;
      try {
        encoded = encodeJSON(site.request!, body);
      } catch (cause) {
        if (!(cause instanceof CodecIssue)) throw cause;
        return fail(types.invalidRequest, [["reason", "request"]], site.action);
      }
      const budget = site.limit ?? ACTION_JSON_BODY_LIMIT;
      if (byteLength(encoded) > BigInt(budget))
        return fail(types.bodyLimit, [["limit", BigInt(budget)]], site.action);
    }
    const outcome = await fetchJsonAction({
      url,
      method,
      request: site.request ?? undefined,
      body,
      response: site.response,
      cases: site.cases,
      requestLimit: site.limit ?? undefined,
    });
    switch (outcome.kind) {
      case "ok":
        return success(outcome.value);
      case "transport":
        return fail(types.transport, [["phase", outcome.phase]], site.action);
      case "aborted":
        // A cancelled fetch carries no commit knowledge: it is a
        // transport failure, never a domain case, and must not be read
        // as server rollback. Reconciliation rereads instead of assuming.
        return fail(types.transport, [["phase", "cancelled"]], site.action);
      case "codec":
        return fail(
          types.invalidData,
          [
            ["path", outcome.path],
            ["reason", outcome.reason],
          ],
          site.action,
        );
      case "unexpected_status":
        return fail(
          types.statusError,
          [
            ["status", BigInt(outcome.status)],
            ["headers", array([])],
          ],
          site.action,
        );
    }
  }
  return Object.freeze({
    get(name: unknown, ...tail: unknown[]): Promise<Completion<unknown>> {
      return run("GET", name, tail);
    },
    post(name: unknown, ...tail: unknown[]): Promise<Completion<unknown>> {
      return run("POST", name, tail);
    },
  });
}
