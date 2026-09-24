import { invoke, type Completion, type AssertionContext } from "../completion.ts";
import { recordIdentity } from "../data.ts";
import { ownBytes, copyBytes, byteLength } from "../bytes.ts";
import { createDomainRuntime } from "../domain.ts";
import { CodecIssue } from "../codec/budget.ts";
import { decodeJSON, encodeJSON, type Schema } from "../codec/json.ts";
import { graph } from "../codec/project.ts";
import { jsonRequestMedia, responseMedia } from "../transport/media.ts";
import { requestSnapshot, createResponses } from "./http.ts";
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
        if (media === undefined || !jsonRequestMedia(media)) return fixed(400, "Bad Request");
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
// domain cases; T23 lowers these names into browser-callable failures.
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
    if (encoded.byteLength > ACTION_JSON_BODY_LIMIT)
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
      credentials: "omit",
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
