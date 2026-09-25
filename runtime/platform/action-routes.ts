// Guarded action route table: compile, canonical dispatch, canonical URL
// building and fixed rejects for checked `action` declarations. The table
// half of this module is pure (no domain, no I/O); the mount adapter half
// maps ActionRouteIssue to domain failures and serves matches through the
// server lifecycle.
//
// Dispatch is method-first: the request method selects a route group, and
// within the group a static segment beats a capture at the same position.
// Two routes of one method collide when their shapes are equal after
// capture names normalize away (duplicate) or when their shapes can match
// one path without a static-priority winner (ambiguous); both are build
// failures, never silent precedence. Different methods never collide.
//
// Matching decodes each raw path segment strictly. A malformed escape, an
// encoded separator, a control, a backslash or a dot segment is a bad
// request before any route is consulted, so none of them can reach a
// protected handler as data. Bun resolves dot segments before user code
// runs; the decoded-dot rejection below keeps direct callers honest too.
// Integer captures additionally require the raw segment to spell the
// canonical decimal text, so an encoded digit never enters as an int.
import { success, failure, invoke, type Completion, type AssertionContext } from "../completion.ts";
import { denyLiveBoundary } from "../assert/context.ts";
import { record, recordIdentity, dataProperty } from "../data.ts";
import { byteLength } from "../bytes.ts";
import { concreteTypeDigestInput } from "../domain-core.ts";
import type { createDomainRuntime } from "../domain.ts";
import { CodecIssue } from "../codec/budget.ts";
import { decodeJSON, encodeJSON, type Schema } from "../codec/json.ts";
import { graph } from "../codec/project.ts";
import { jsonRequestMedia, mediaType } from "../transport/media.ts";
import { requestSnapshot, snapshotBodyBytes, ownedResponse, ownedJsonResponse } from "./http.ts";
import { decodeActionForm, FormIssue, maxFormRows, type FormSchema } from "./form.ts";
import { renderSafe } from "./html.ts";
export class ActionRouteIssue extends Error {
  readonly code:
    | "invalid-route"
    | "duplicate-route"
    | "ambiguous-route"
    | "unknown-action"
    | "capture-arity"
    | "capture-type"
    | "capture-value";
  readonly detail: string;
  // Structured assembly evidence: duplicate collisions carry the method and
  // decoded shape, ambiguous collisions carry the two templates, so the
  // router maps them to duplicate/ambiguous failures without parsing text.
  readonly data: Readonly<Record<string, string>>;
  constructor(
    code: ActionRouteIssue["code"],
    detail: string,
    data: Readonly<Record<string, string>> = Object.freeze({}),
  ) {
    super(`action route ${detail}`);
    this.name = "ActionRouteIssue";
    this.code = code;
    this.detail = detail;
    this.data = data;
  }
}

export type ActionCaptureType = "str" | "int";

export type ActionRouteSourceCapture = Readonly<{ name: string; type: ActionCaptureType }>;

export type ActionRouteSource = Readonly<{
  identity: string;
  method: string;
  path: string;
  captures: readonly ActionRouteSourceCapture[];
}>;

export type ActionRouteSegment =
  | Readonly<{ kind: "static"; raw: string; decoded: string }>
  | Readonly<{ kind: "capture"; name: string; type: ActionCaptureType }>;

export type CompiledActionRoute = Readonly<{
  identity: string;
  method: "GET" | "POST";
  template: string;
  segments: readonly ActionRouteSegment[];
  statics: number;
  bunKey: string;
}>;

export type ActionRouteTable = Readonly<{ routes: readonly CompiledActionRoute[] }>;

export type ActionRouteCapture = Readonly<{ name: string; value: string | bigint }>;

export type ActionRouteMatch =
  | Readonly<{ kind: "match"; identity: string; captures: readonly ActionRouteCapture[] }>
  | Readonly<{ kind: "bad-request" }>
  | Readonly<{ kind: "not-found" }>
  | Readonly<{ kind: "method-not-allowed"; allow: string }>;

const MIN_INT64 = -(2n ** 63n),
  MAX_INT64 = 2n ** 63n - 1n;

function invalid(detail: string): ActionRouteIssue {
  return new ActionRouteIssue("invalid-route", detail);
}

// Template checks mirror the checker's action route shape plus the shared
// static-path contract: raw controls, space, DEL, backslash and the
// `?#*` metacharacters are rejected before decoding, and decoded statics
// reject controls, DEL and backslash again.
function checkTemplate(path: string): string[] {
  if (typeof path !== "string" || !path.isWellFormed()) throw invalid("path is not a string");
  if (!path.startsWith("/") || path.startsWith("//")) throw invalid(`path ${JSON.stringify(path)}`);
  for (const char of path) {
    const code = char.codePointAt(0)!;
    if (
      code <= 0x20 ||
      code === 0x7f ||
      char === "\\" ||
      char === "?" ||
      char === "#" ||
      char === "*"
    )
      throw invalid(`path ${JSON.stringify(path)}`);
  }
  const segments = path.split("/");
  const seen = new Set<string>();
  for (const segment of segments.slice(1)) {
    if (segment.startsWith(":")) throw invalid(`path ${JSON.stringify(path)}`);
    if (!segment.includes("{") && !segment.includes("}")) continue;
    if (!segment.startsWith("{") || !segment.endsWith("}") || segment.length < 3)
      throw invalid(`captures occupy one whole {name} segment in ${JSON.stringify(path)}`);
    const name = segment.slice(1, -1);
    if (!/^[a-z][a-z0-9_]*$/.test(name)) throw invalid(`invalid path capture {${name}}`);
    if (seen.has(name)) throw invalid(`duplicate path capture {${name}}`);
    seen.add(name);
  }
  return segments;
}

function decodeStatic(segment: string, path: string): string {
  let decoded: string;
  try {
    decoded = decodeURIComponent(segment);
  } catch {
    throw invalid(`path ${JSON.stringify(path)}`);
  }
  // oxlint-disable-next-line no-control-regex -- Decoded static segments reject C0 controls, DEL, and backslash.
  if (/[\x00-\x1f\x7f\\]/.test(decoded)) throw invalid(`path ${JSON.stringify(path)}`);
  return decoded;
}

function compileRoute(entry: unknown): CompiledActionRoute {
  if (typeof entry !== "object" || entry === null) throw invalid("entry is not an object");
  const source = entry as Partial<ActionRouteSource>;
  if (typeof source.identity !== "string" || source.identity === "")
    throw invalid("identity is not a string");
  if (source.method !== "GET" && source.method !== "POST")
    throw invalid(`method ${JSON.stringify(source.method)}`);
  if (typeof source.path !== "string") throw invalid("path is not a string");
  const raw = checkTemplate(source.path);
  if (!Array.isArray(source.captures)) throw invalid("captures are not rows");
  const rows = new Map<string, ActionCaptureType>();
  for (const row of source.captures) {
    if (typeof row !== "object" || row === null) throw invalid("capture row is not an object");
    const name = (row as Partial<ActionRouteSourceCapture>).name,
      type = (row as Partial<ActionRouteSourceCapture>).type;
    if (typeof name !== "string" || (type !== "str" && type !== "int"))
      throw invalid("capture row needs a name with str or int");
    if (rows.has(name)) throw invalid(`duplicate capture row ${name}`);
    rows.set(name, type);
  }
  const segments: ActionRouteSegment[] = [];
  let statics = 0;
  for (const segment of raw.slice(1)) {
    if (segment.startsWith("{")) {
      const name = segment.slice(1, -1),
        type = rows.get(name);
      if (type === undefined) throw invalid(`path capture {${name}} has no captures row`);
      segments.push(Object.freeze({ kind: "capture", name, type }));
      continue;
    }
    const decoded = decodeStatic(segment, source.path);
    statics++;
    segments.push(Object.freeze({ kind: "static", raw: segment, decoded }));
  }
  for (const name of rows.keys())
    if (!raw.slice(1).includes(`{${name}}`))
      throw invalid(`capture ${name} does not appear in the action path`);
  if (segments.length > 0 && segments[0]!.kind === "static" && segments[0]!.decoded === "__can")
    throw invalid(`path ${JSON.stringify(source.path)}`);
  const bunKey =
    "/" +
    raw
      .slice(1)
      .map((segment) => (segment.startsWith("{") ? `:${segment.slice(1, -1)}` : segment))
      .join("/");
  return Object.freeze({
    identity: source.identity,
    method: source.method,
    template: source.path,
    segments: Object.freeze(segments),
    statics,
    bunKey,
  });
}

function shapeOf(route: CompiledActionRoute): string {
  return (
    "/" +
    route.segments.map((segment) => (segment.kind === "capture" ? "{}" : segment.decoded)).join("/")
  );
}

// Same-method shapes unify when every position agrees or captures. The
// more specific shape wins every shared path when the other's static
// positions are a strict subset of its own; anything else that unifies
// without equal shapes is an ambiguous collision.
function collides(
  first: CompiledActionRoute,
  second: CompiledActionRoute,
): "duplicate" | "ambiguous" | undefined {
  if (first.method !== second.method || first.segments.length !== second.segments.length)
    return undefined;
  let same = true;
  for (let i = 0; i < first.segments.length; i++) {
    const a = first.segments[i]!,
      b = second.segments[i]!;
    if (a.kind === "static" && b.kind === "static") {
      if (a.decoded !== b.decoded) return undefined;
      continue;
    }
    // Captures share one shape position: names normalize away.
    if (a.kind !== b.kind) same = false;
  }
  if (same) return "duplicate";
  const covers = (outer: CompiledActionRoute, inner: CompiledActionRoute): boolean =>
    outer.statics > inner.statics &&
    outer.segments.every(
      (segment, i) => segment.kind === "static" || inner.segments[i]!.kind === "capture",
    );
  return covers(first, second) || covers(second, first) ? undefined : "ambiguous";
}

export function compileActionRoutes(entries: readonly unknown[]): ActionRouteTable {
  if (!Array.isArray(entries)) throw invalid("entries are not an array");
  const routes = entries.map(compileRoute);
  for (let i = 0; i < routes.length; i++)
    for (let j = i + 1; j < routes.length; j++) {
      const hit = collides(routes[i]!, routes[j]!);
      if (hit === "duplicate")
        throw new ActionRouteIssue(
          "duplicate-route",
          `${routes[j]!.identity} duplicates the ${routes[j]!.method} ${shapeOf(routes[j]!)} route of ${routes[i]!.identity}`,
          Object.freeze({ method: routes[j]!.method, path: shapeOf(routes[j]!) }),
        );
      if (hit === "ambiguous")
        throw new ActionRouteIssue(
          "ambiguous-route",
          `${routes[j]!.identity} ambiguously overlaps the ${routes[j]!.method} ${shapeOf(routes[j]!)} route of ${routes[i]!.identity}`,
          Object.freeze({ first: routes[i]!.template, second: routes[j]!.template }),
        );
    }
  return Object.freeze({ routes: Object.freeze(routes) });
}

// Path integers use the language's canonical decimal spelling: no leading
// zeros, no plus sign, no `-0`. Values outside int64 are malformed.
function strictInt(text: string): bigint | undefined {
  const unsigned = text.startsWith("-") ? text.slice(1) : text;
  if (text.startsWith("-") ? !/^[1-9][0-9]*$/.test(unsigned) : !/^(0|[1-9][0-9]*)$/.test(text))
    return undefined;
  if (unsigned.length > 19) return undefined;
  const value = BigInt(text);
  return value < MIN_INT64 || value > MAX_INT64 ? undefined : value;
}

type DecodedTarget = Readonly<{ raw: readonly string[]; decoded: readonly string[] }>;

function decodeSegments(pathname: string): DecodedTarget | undefined {
  const raw = pathname.split("/");
  if (raw[0] !== "") return undefined;
  const decoded: string[] = [];
  for (const segment of raw.slice(1)) {
    let text: string;
    try {
      text = decodeURIComponent(segment);
    } catch {
      return undefined;
    }
    // oxlint-disable-next-line no-control-regex -- Decoded request segments reject C0 controls, DEL, and backslash.
    if (text.includes("/") || /[\x00-\x1f\x7f\\]/.test(text) || text === "." || text === "..")
      return undefined;
    decoded.push(text);
  }
  return { raw: Object.freeze(raw.slice(1)), decoded: Object.freeze(decoded) };
}

// Target-level validation for combined dispatch: decoded segments when the
// raw pathname survives strict per-segment decoding, undefined when a
// malformed escape, encoded separator, control, backslash or dot rules the
// whole target out before any route is consulted.
export function actionTargetSegments(pathname: unknown): readonly string[] | undefined {
  if (typeof pathname !== "string") return undefined;
  return decodeSegments(pathname)?.decoded;
}

function structural(route: CompiledActionRoute, decoded: readonly string[]): boolean {
  return (
    route.segments.length === decoded.length &&
    route.segments.every((segment, i) =>
      segment.kind === "static" ? segment.decoded === decoded[i] : decoded[i] !== "",
    )
  );
}

function typed(
  route: CompiledActionRoute,
  target: DecodedTarget,
): ActionRouteCapture[] | undefined {
  if (!structural(route, target.decoded)) return undefined;
  const captures: ActionRouteCapture[] = [];
  for (let i = 0; i < route.segments.length; i++) {
    const segment = route.segments[i]!;
    if (segment.kind === "static") continue;
    if (segment.type === "int") {
      // Canonical decimal text needs no escapes, so the raw segment must
      // equal the decoded one: an encoded digit is malformed, not an int.
      if (target.raw[i] !== target.decoded[i]) return undefined;
      const value = strictInt(target.decoded[i]!);
      if (value === undefined) return undefined;
      captures.push(Object.freeze({ name: segment.name, value }));
    } else captures.push(Object.freeze({ name: segment.name, value: target.decoded[i]! }));
  }
  return captures;
}

export function matchActionRoute(
  table: ActionRouteTable,
  method: string,
  pathname: string,
): ActionRouteMatch {
  const target = typeof pathname === "string" ? decodeSegments(pathname) : undefined;
  if (target === undefined) return Object.freeze({ kind: "bad-request" });
  const decoded = target.decoded;
  let best: { route: CompiledActionRoute; captures: readonly ActionRouteCapture[] } | undefined;
  let groupStructural = false;
  const foreign = new Set<string>();
  for (const route of table.routes) {
    const captures = typed(route, target);
    if (captures !== undefined) {
      if (route.method === method) {
        if (best === undefined || route.statics > best.route.statics)
          best = { route, captures: Object.freeze(captures) };
      } else foreign.add(route.method);
      continue;
    }
    if (!structural(route, decoded)) continue;
    if (route.method === method) groupStructural = true;
    else foreign.add(route.method);
  }
  if (best !== undefined)
    return Object.freeze({ kind: "match", identity: best.route.identity, captures: best.captures });
  if (groupStructural) return Object.freeze({ kind: "bad-request" });
  if (foreign.size !== 0)
    return Object.freeze({ kind: "method-not-allowed", allow: [...foreign].sort().join(", ") });
  return Object.freeze({ kind: "not-found" });
}

function checkURLValue(text: string): boolean {
  return (
    text !== "" &&
    text !== "." &&
    text !== ".." &&
    !text.includes("/") &&
    // oxlint-disable-next-line no-control-regex -- Builder captures reject C0 controls, DEL, and backslash.
    !/[\x00-\x1f\x7f\\]/.test(text)
  );
}

// The canonical URL builder emits the one spelling dispatch accepts: raw
// template statics plus strictly encoded captures in path order, so every
// built URL round-trips to the same action and values.
export function buildActionURL(
  table: ActionRouteTable,
  identity: string,
  values: readonly unknown[],
): string {
  const route = table.routes.find((route) => route.identity === identity);
  if (route === undefined)
    throw new ActionRouteIssue("unknown-action", `unknown action ${identity}`);
  const want = route.segments.filter((segment) => segment.kind === "capture");
  if (!Array.isArray(values) || values.length !== want.length)
    throw new ActionRouteIssue("capture-arity", `${identity} takes ${want.length} captures`);
  let i = 0;
  const segments = route.segments.map((segment) => {
    if (segment.kind === "static") return segment.raw;
    const type = (segment as Extract<ActionRouteSegment, { kind: "capture" }>).type,
      value = values[i++]!;
    if (type === "int") {
      if (typeof value !== "bigint")
        throw new ActionRouteIssue(
          "capture-type",
          `${identity} capture ${segment.name} must be int`,
        );
      if (value < MIN_INT64 || value > MAX_INT64)
        throw new ActionRouteIssue(
          "capture-value",
          `${identity} capture ${segment.name} is out of range`,
        );
      return value.toString();
    }
    if (typeof value !== "string")
      throw new ActionRouteIssue("capture-type", `${identity} capture ${segment.name} must be str`);
    if (!value.isWellFormed() || !checkURLValue(value))
      throw new ActionRouteIssue(
        "capture-value",
        `${identity} capture ${segment.name} is not one segment`,
      );
    return encodeURIComponent(value);
  });
  return "/" + segments.join("/");
}

export function bunRouteKeys(table: ActionRouteTable): readonly string[] {
  return Object.freeze(table.routes.map((route) => route.bunKey));
}

export function actionReject(status: 400 | 405, allow?: string): Response {
  const headers = new Headers({
    "content-type": "text/plain; charset=utf-8",
    "x-content-type-options": "nosniff",
  });
  if (status === 405 && allow !== undefined) headers.set("allow", allow);
  return new Response(status === 400 ? "Bad Request" : "Method Not Allowed", { status, headers });
}

const adapterOrigin = Object.freeze({
  source: "can:action-routes",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

// Mount dispatch callback: the strict captures from one canonical match join
// the live request; the callback decodes the declared body, invokes the
// exact checked callables and renders the case status. It runs live only.
export type ActionDispatchCallback = (
  request: unknown,
  captures: readonly ActionRouteCapture[],
  context?: AssertionContext,
) => Promise<Completion<unknown>>;

export type ActionRouteBinding = Readonly<{
  entry: ActionRouteSource;
  callback: ActionDispatchCallback;
}>;

const actionBindings = new WeakMap<object, ActionRouteBinding>();

export function isActionRouteBinding(value: unknown): boolean {
  return (
    value !== null &&
    (typeof value === "object" || typeof value === "function") &&
    actionBindings.has(value)
  );
}

export function readActionRouteBinding(value: unknown): ActionRouteBinding {
  if (!isActionRouteBinding(value)) throw new TypeError("invalid compiler action route");
  return actionBindings.get(value as object)!;
}

type MountCase = Readonly<{ leaf: string; status: number }>;
type MountInput =
  | Readonly<{ mode: "none" }>
  | Readonly<{ mode: "json"; request: Schema; limit: number }>
  | Readonly<{ mode: "form"; form: FormSchema; limit: number; rowsLimit: number }>;
type MountEntry = Readonly<{
  identity: string;
  method: "GET" | "POST";
  path: string;
  body: "json" | "html";
  captures: readonly ActionRouteSourceCapture[];
  capturesType?: string;
  capturesIdentity?: string;
  input: MountInput;
  cases: readonly MountCase[];
  response?: Schema;
  rejected?: string;
  rawEntry?: string;
  issue?: string;
}>;

function isSiteRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object";
}

function mountSchema(identity: string, slot: string, value: unknown): Schema {
  if (!isSiteRecord(value) || typeof value.root !== "string" || !Array.isArray(value.nodes))
    throw new TypeError(`action ${identity} carries no ${slot} JSON schema`);
  for (const node of value.nodes) {
    if (
      !isSiteRecord(node) ||
      typeof node.identity !== "string" ||
      typeof node.kind !== "string" ||
      typeof node.name !== "string"
    )
      throw new TypeError(`action ${identity} carries a malformed ${slot} JSON schema`);
  }
  const schema = value as unknown as Schema;
  // Fail fast on duplicate nodes and an unresolvable root; the shared codec
  // would reject the same shape later with a less actionable diagnostic.
  graph(schema)(schema.root);
  return schema;
}

function mountCases(identity: string, cases: unknown): readonly MountCase[] {
  if (!Array.isArray(cases) || cases.length === 0)
    throw new TypeError(`action ${identity} carries no response cases`);
  const seen = new Set<string>();
  return cases.map((kase) => {
    if (
      !isSiteRecord(kase) ||
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
      throw new TypeError(`action ${identity} carries a malformed response case`);
    if (seen.has(kase.leaf))
      throw new TypeError(`action ${identity} repeats case leaf ${kase.leaf}`);
    seen.add(kase.leaf);
    return { leaf: kase.leaf, status: kase.status };
  });
}

function mountCaptures(identity: string, captures: unknown): readonly ActionRouteSourceCapture[] {
  if (!Array.isArray(captures))
    throw new TypeError(`action ${identity} carries malformed captures`);
  return captures.map((row) => {
    if (
      !isSiteRecord(row) ||
      typeof row.name !== "string" ||
      row.name === "" ||
      (row.type !== "str" && row.type !== "int")
    )
      throw new TypeError(`action ${identity} carries a malformed capture`);
    return { name: row.name, type: row.type };
  });
}

function mountFormField(identity: string, field: unknown, nested: boolean): void {
  if (!isSiteRecord(field) || typeof field.name !== "string" || field.name === "")
    throw new TypeError(`action ${identity} carries a malformed form schema`);
  if (field.kind === "str" || field.kind === "array") return;
  if (field.kind === "optional") {
    if (
      typeof field.some !== "string" ||
      field.some === "" ||
      typeof field.none !== "string" ||
      field.none === ""
    )
      throw new TypeError(`action ${identity} carries a malformed form schema`);
    return;
  }
  if (field.kind !== "rows" || nested)
    throw new TypeError(`action ${identity} carries a malformed form schema`);
  const rows = field.rows;
  if (
    !isSiteRecord(rows) ||
    typeof rows.row !== "string" ||
    rows.row === "" ||
    typeof rows.order !== "string" ||
    rows.order === "" ||
    typeof rows.collection !== "string" ||
    rows.collection === "" ||
    typeof rows.item !== "string" ||
    rows.item === "" ||
    !Array.isArray(rows.fields)
  )
    throw new TypeError(`action ${identity} carries a malformed form schema`);
  for (const child of rows.fields) mountFormField(identity, child, true);
}

function mountFormSchema(identity: string, schema: unknown): FormSchema {
  if (
    !isSiteRecord(schema) ||
    typeof schema.root !== "string" ||
    schema.root === "" ||
    !Array.isArray(schema.fields)
  )
    throw new TypeError(`action ${identity} carries a malformed form schema`);
  for (const field of schema.fields) mountFormField(identity, field, false);
  return schema as unknown as FormSchema;
}

function mountLimit(identity: string, slot: string, value: unknown): number {
  if (typeof value !== "number" || !Number.isSafeInteger(value) || value < 1)
    throw new TypeError(`action ${identity} carries a malformed ${slot} limit`);
  return value;
}

// Checked mount-site validation. Malformed metadata is a compiler bug and
// throws; the route template itself compiles separately so template faults
// surface as the declared invalid_route failure.
function checkedMountSite(site: unknown, form: boolean): MountEntry {
  if (!isSiteRecord(site) || typeof site.action !== "string" || site.action === "")
    throw new TypeError("action mount site carries no action identity");
  const identity = site.action;
  if (site.method !== "GET" && site.method !== "POST")
    throw new TypeError(`action ${identity} names an unknown method`);
  if (typeof site.path !== "string" || site.path === "")
    throw new TypeError(`action ${identity} names no path`);
  const captures = mountCaptures(identity, site.captures);
  let capturesType: string | undefined;
  if (captures.length !== 0) {
    if (typeof site.capturesType !== "string" || site.capturesType === "")
      throw new TypeError(`action ${identity} carries no captures record`);
    capturesType = site.capturesType;
  } else if (site.capturesType !== undefined && site.capturesType !== "") {
    throw new TypeError(`action ${identity} carries captures without a captures record`);
  }
  if (!isSiteRecord(site.input) || typeof site.input.mode !== "string")
    throw new TypeError(`action ${identity} carries no input contract`);
  const mode = site.input.mode;
  if (site.method === "GET" && mode !== "none")
    throw new TypeError(`action ${identity} is a GET action with a body contract`);
  if (site.method === "POST" && mode === "none")
    throw new TypeError(`action ${identity} is a POST action with no body`);
  let input: MountInput;
  if (mode === "none") {
    input = { mode };
  } else if (mode === "json") {
    if (typeof site.input.type !== "string" || site.input.type === "")
      throw new TypeError(`action ${identity} carries no request wire type`);
    input = {
      mode,
      request: mountSchema(identity, "request", site.input.schema),
      limit: mountLimit(identity, "byte", site.input.limit),
    };
  } else if (mode === "form") {
    if (typeof site.input.type !== "string" || site.input.type === "")
      throw new TypeError(`action ${identity} carries no form wire type`);
    const form = mountFormSchema(identity, site.input.schema);
    const rows = form.fields.some((field) => field.kind === "rows");
    const rowsLimit = site.input.rowsLimit;
    if (
      rows &&
      (typeof rowsLimit !== "number" ||
        !Number.isInteger(rowsLimit) ||
        rowsLimit < 1 ||
        rowsLimit > maxFormRows)
    )
      throw new TypeError(`action ${identity} carries a malformed rows limit`);
    if (!rows && rowsLimit !== undefined)
      throw new TypeError(`action ${identity} carries a rows limit without keyed rows`);
    input = {
      mode,
      form,
      limit: mountLimit(identity, "byte", site.input.limit),
      rowsLimit: rows ? (rowsLimit as number) : maxFormRows,
    };
  } else {
    throw new TypeError(`action ${identity} names an unknown input mode`);
  }
  if (typeof site.returns !== "string" || site.returns === "")
    throw new TypeError(`action ${identity} carries no returns type`);
  if (site.body !== "json" && site.body !== "html")
    throw new TypeError(`action ${identity} names an unknown body mode`);
  if ((mode === "form") !== (site.body === "html"))
    throw new TypeError(`action ${identity} disagrees on its body mode`);
  if (form !== (site.body === "html"))
    throw new TypeError(`action ${identity} reached the wrong mount entry`);
  const cases = mountCases(identity, site.cases);
  if (!form) {
    return {
      identity,
      method: site.method,
      path: site.path,
      body: site.body,
      captures,
      capturesType,
      input,
      cases,
      response: mountSchema(identity, "response", site.responseSchema),
    };
  }
  if (
    typeof site.rejected !== "string" ||
    site.rejected === "" ||
    typeof site.rawEntry !== "string" ||
    site.rawEntry === "" ||
    typeof site.issue !== "string" ||
    site.issue === ""
  )
    throw new TypeError(`action ${identity} carries no structural value identities`);
  return {
    identity,
    method: site.method,
    path: site.path,
    body: site.body,
    captures,
    capturesType,
    input,
    cases,
    rejected: site.rejected,
    rawEntry: site.rawEntry,
    issue: site.issue,
  };
}

// Checked `:name` templates normalize to the table's `{name}` spelling, and
// already-normalized whole `{name}` segments pass through unchanged so
// mount, url and fetch share one normalization. Static segments keep the
// shared static-path contract; anything else carrying capture punctuation
// is a template fault, never a silent static.
export function actionTemplate(identity: string, path: string): string {
  const segments = path.split("/");
  if (segments[0] !== "") throw invalid(`path ${JSON.stringify(path)}`);
  const out = segments.map((segment, i) => {
    if (i === 0) return "";
    if (/^:[a-z][a-z0-9_]*$/.test(segment)) return `{${segment.slice(1)}}`;
    if (/^\{[a-z][a-z0-9_]*\}$/.test(segment)) return segment;
    if (segment.includes(":") || segment.includes("{") || segment.includes("}"))
      throw new ActionRouteIssue(
        "invalid-route",
        `${identity} captures occupy one whole :name segment in ${JSON.stringify(path)}`,
      );
    return segment;
  });
  return out.join("/");
}

type MountedHandler = (...args: never[]) => Promise<Completion<unknown>>;

function checkedCallable(identity: string, role: string, handler: unknown): MountedHandler {
  if (typeof handler !== "function")
    throw new TypeError(`action ${identity} has no callable ${role}`);
  return handler as MountedHandler;
}

function hexBytes(digest: ArrayBuffer): string {
  return [...new Uint8Array(digest)].map((byte) => byte.toString(16).padStart(2, "0")).join("");
}

// Captures-record identity for adapter-built values. The site carries the
// declaration only, so mount derives the true hashed identity once over
// the canonical framing shared with the domain digests. Nominal matches
// on the captures value then behave exactly like emitted construction.
// WebCrypto keeps this module free of host-only crypto imports.
async function capturesIdentity(declaration: string): Promise<string> {
  const subtle = globalThis.crypto?.subtle;
  if (!subtle) throw new TypeError("action captures require crypto.subtle");
  const digest = await subtle.digest(
    "SHA-256",
    new TextEncoder().encode(concreteTypeDigestInput(["record", declaration])),
  );
  return hexBytes(digest);
}

function formRequestMedia(value: string): boolean {
  try {
    const parsed = mediaType(value);
    return (
      parsed.type === "application/x-www-form-urlencoded" &&
      [...parsed.parameters].every(
        ([key, param]) => key === "charset" && param.toLowerCase() === "utf-8",
      )
    );
  } catch (cause) {
    if (cause instanceof CodecIssue) return false;
    throw cause;
  }
}

// createActionRoutes binds the exact checked mount callables: one
// request-first handler for JSON actions, plus the distinct normal and
// structural renderers for HTML form actions. Mount returns an http::route
// token for make_router; url builds canonical paths from captures records.
// Template faults fail as invalid_route, unbuildable captures as
// invalid_path; malformed sites and non-callables are compiler bugs.
export function createActionRoutes(
  domain: ReturnType<typeof createDomainRuntime>,
  types: Readonly<{ invalidRoute: string; invalidPath: string }>,
) {
  const invalidRoute = (reason: string) =>
    failure(
      domain.create(
        types.invalidRoute,
        record(types.invalidRoute, [["reason", reason]]),
        adapterOrigin,
      ),
    );
  const invalidPath = (reason: string) =>
    failure(
      domain.create(
        types.invalidPath,
        record(types.invalidPath, [["reason", reason]]),
        adapterOrigin,
      ),
    );
  function buildJsonCallback(entry: MountEntry, handler: MountedHandler): ActionDispatchCallback {
    const input = entry.input;
    if (input.mode !== "none" && input.mode !== "json")
      throw new TypeError("invalid compiler action input");
    return async (request, captures, context) => {
      denyLiveBoundary(context, adapterOrigin);
      const snapshot = requestSnapshot(request);
      const values: unknown[] = [request];
      if (entry.capturesIdentity !== undefined)
        values.push(
          record(
            entry.capturesIdentity,
            entry.captures.map((capture) => [
              capture.name,
              captures.find((row) => row.name === capture.name)!.value,
            ]),
          ),
        );
      if (input.mode === "none") {
        // GET-load is bodyless: any actual body bytes are a client
        // violation, even though standard clients cannot easily send them.
        const probe = await snapshotBodyBytes(request);
        if (probe.kind === "limited")
          return success(ownedResponse(413, "Payload Too Large", false));
        if (probe.kind !== "bytes" || byteLength(probe.value) !== 0n)
          return success(ownedResponse(400, "Bad Request", false));
      } else {
        const media = snapshot.headers.find(([name]) => name === "content-type")?.[1];
        if (media === undefined || !jsonRequestMedia(media))
          return success(ownedResponse(415, "Unsupported Media Type", false));
        const body = await snapshotBodyBytes(request);
        if (body.kind === "limited") return success(ownedResponse(413, "Payload Too Large", false));
        if (body.kind !== "bytes") return success(ownedResponse(400, "Bad Request", false));
        if (byteLength(body.value) > BigInt(input.limit))
          return success(ownedResponse(413, "Payload Too Large", false));
        if (byteLength(body.value) === 0n) return success(ownedResponse(400, "Bad Request", false));
        try {
          values.push(decodeJSON(input.request, body.value, input.limit));
        } catch (cause) {
          if (!(cause instanceof CodecIssue)) throw cause;
          // Malformed, duplicate, over-cap or mistyped JSON never reaches
          // the protected handler; the fixed 400 carries no wire detail.
          return success(ownedResponse(400, "Bad Request", false));
        }
      }
      const completed = await invoke(
        () => (handler as (...args: unknown[]) => Promise<Completion<unknown>>)(...values, context),
        adapterOrigin,
      );
      if (completed.kind !== "ok")
        return success(ownedResponse(500, "Internal Server Error", false));
      const leaf = recordIdentity(completed.value);
      const kase = entry.cases.find((row) => row.leaf === leaf);
      if (leaf === undefined || kase === undefined)
        return success(ownedResponse(500, "Internal Server Error", false));
      // Every domain outcome renders its declared finite status with a JSON
      // representation; the adapter never invents a status or serves HTML.
      let rendered: ReturnType<typeof encodeJSON>;
      try {
        rendered = encodeJSON(entry.response!, completed.value);
      } catch (cause) {
        if (!(cause instanceof CodecIssue)) throw cause;
        return success(ownedResponse(500, "Internal Server Error", false));
      }
      return success(ownedJsonResponse(kase.status, rendered));
    };
  }
  function buildFormCallback(
    entry: MountEntry,
    handler: MountedHandler,
    outcome: MountedHandler,
    structural: MountedHandler,
  ): ActionDispatchCallback {
    const input = entry.input;
    if (input.mode !== "form") throw new TypeError("invalid compiler action input");
    return async (request, captures, context) => {
      denyLiveBoundary(context, adapterOrigin);
      const body = await snapshotBodyBytes(request);
      if (body.kind === "limited") return success(ownedResponse(413, "Payload Too Large", false));
      if (body.kind !== "bytes") return success(ownedResponse(400, "Bad Request", false));
      const media = requestSnapshot(request).headers.find(([name]) => name === "content-type")?.[1];
      if (media === undefined || !formRequestMedia(media))
        return success(ownedResponse(415, "Unsupported Media Type", false));
      if (byteLength(body.value) > BigInt(input.limit))
        return success(ownedResponse(413, "Payload Too Large", false));
      const values: unknown[] = [request];
      if (entry.capturesIdentity !== undefined)
        values.push(
          record(
            entry.capturesIdentity,
            entry.captures.map((capture) => [
              capture.name,
              captures.find((row) => row.name === capture.name)!.value,
            ]),
          ),
        );
      let decoded: ReturnType<typeof decodeActionForm>;
      try {
        decoded = decodeActionForm(
          input.form,
          body.value,
          input.limit,
          {
            rejected: entry.rejected!,
            rawEntry: entry.rawEntry!,
            issue: entry.issue!,
          },
          input.rowsLimit,
        );
      } catch (cause) {
        if (cause instanceof CodecIssue || cause instanceof FormIssue)
          return success(ownedResponse(400, "Bad Request", false));
        throw cause;
      }
      if (decoded.kind === "rejected") {
        const rendered = await invoke(
          () =>
            (structural as (...args: unknown[]) => Promise<Completion<unknown>>)(
              decoded.value,
              context,
            ),
          adapterOrigin,
        );
        if (rendered.kind !== "ok") return rendered;
        return success(ownedResponse(422, renderSafe(rendered.value), true));
      }
      values.push(decoded.value);
      const handled = await invoke(
        () => (handler as (...args: unknown[]) => Promise<Completion<unknown>>)(...values, context),
        adapterOrigin,
      );
      if (handled.kind !== "ok") return success(ownedResponse(500, "Internal Server Error", false));
      const leaf = recordIdentity(handled.value);
      const kase = entry.cases.find((row) => row.leaf === leaf);
      if (leaf === undefined || kase === undefined)
        return success(ownedResponse(500, "Internal Server Error", false));
      const rendered = await invoke(
        () =>
          (outcome as (...args: unknown[]) => Promise<Completion<unknown>>)(handled.value, context),
        adapterOrigin,
      );
      // Renderer faults return as-is: the server maps them to an uncertain
      // 500 while the fault detail stays observable to direct dispatch.
      if (rendered.kind !== "ok") return rendered;
      return success(ownedResponse(kase.status, renderSafe(rendered.value), true));
    };
  }
  async function mountRoute(
    entry: MountEntry,
    callback: ActionDispatchCallback,
  ): Promise<Completion<unknown>> {
    let template: string;
    try {
      template = actionTemplate(entry.identity, entry.path);
      compileActionRoutes([
        {
          identity: entry.identity,
          method: entry.method,
          path: template,
          captures: entry.captures,
        },
      ]);
    } catch (cause) {
      if (!(cause instanceof ActionRouteIssue)) throw cause;
      return invalidRoute("path");
    }
    const token = Object.freeze(Object.create(null));
    actionBindings.set(
      token,
      Object.freeze({
        entry: Object.freeze({
          identity: entry.identity,
          method: entry.method,
          path: template,
          captures: Object.freeze(entry.captures.map((capture) => Object.freeze({ ...capture }))),
        }),
        callback,
      }),
    );
    return success(token);
  }
  async function boundEntry(entry: MountEntry): Promise<MountEntry> {
    if (entry.capturesType === undefined) return entry;
    return Object.freeze({
      ...entry,
      capturesIdentity: await capturesIdentity(entry.capturesType),
    });
  }
  return Object.freeze({
    async mount(
      handler: unknown,
      site: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const entry = checkedMountSite(site, false);
      const run = checkedCallable(entry.identity, "handler", handler);
      const bound = await boundEntry(entry);
      return mountRoute(bound, buildJsonCallback(bound, run));
    },
    async mountForm(
      handler: unknown,
      outcome: unknown,
      structural: unknown,
      site: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const entry = checkedMountSite(site, true);
      const run = checkedCallable(entry.identity, "handler", handler);
      const renderOutcome = checkedCallable(entry.identity, "normal renderer", outcome);
      const renderStructural = checkedCallable(entry.identity, "structural renderer", structural);
      const bound = await boundEntry(entry);
      return mountRoute(bound, buildFormCallback(bound, run, renderOutcome, renderStructural));
    },
    async url(...args: unknown[]): Promise<Completion<unknown>> {
      if (args.length !== 2 && args.length !== 3)
        throw new TypeError("action url takes its captures record and site");
      const site = checkedUrlSite(args.length === 3 ? args[1] : args[0]);
      const found = args.length === 3 ? args[0] : undefined;
      if ((found === undefined) !== (site.captures.length === 0))
        throw new TypeError(`action url for ${site.identity} disagrees on its captures record`);
      const values = site.captures.map((capture) =>
        found === undefined ? undefined : dataProperty(found, capture.name),
      );
      let template: string;
      try {
        template = actionTemplate(site.identity, site.path);
      } catch (cause) {
        if (!(cause instanceof ActionRouteIssue)) throw cause;
        throw new TypeError(`action ${site.identity} carries a malformed route template`);
      }
      let table: ActionRouteTable;
      try {
        table = compileActionRoutes([
          { identity: site.identity, method: site.method, path: template, captures: site.captures },
        ]);
      } catch (cause) {
        if (!(cause instanceof ActionRouteIssue)) throw cause;
        throw new TypeError(`action ${site.identity} carries a malformed route template`);
      }
      try {
        return success(buildActionURL(table, site.identity, values));
      } catch (cause) {
        if (!(cause instanceof ActionRouteIssue)) throw cause;
        return invalidPath(cause.code);
      }
    },
  });
}

type UrlSite = Readonly<{
  identity: string;
  method: "GET" | "POST";
  path: string;
  captures: readonly ActionRouteSourceCapture[];
}>;

// URL-builder site validation: the builder projection carries identity,
// route template and ordered captures only. Template faults are compiler
// bugs and throw; only runtime capture data fails as invalid_path.
function checkedUrlSite(site: unknown): UrlSite {
  if (!isSiteRecord(site) || typeof site.action !== "string" || site.action === "")
    throw new TypeError("action url site carries no action identity");
  const identity = site.action;
  if (site.method !== undefined && site.method !== "GET" && site.method !== "POST")
    throw new TypeError(`action ${identity} names an unknown method`);
  if (typeof site.path !== "string" || site.path === "")
    throw new TypeError(`action ${identity} names no path`);
  return {
    identity,
    method: (site.method ?? "GET") as "GET" | "POST",
    path: site.path,
    captures: mountCaptures(identity, site.captures),
  };
}
