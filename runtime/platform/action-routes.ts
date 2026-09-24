// Guarded action route table: compile, canonical dispatch, canonical URL
// building and fixed rejects for checked `action` declarations. This module
// is pure (no domain, no I/O): adapters map ActionRouteIssue to domain
// failures and serve matches through the server lifecycle.
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
  constructor(code: ActionRouteIssue["code"], detail: string) {
    super(`action route ${detail}`);
    this.name = "ActionRouteIssue";
    this.code = code;
    this.detail = detail;
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
        );
      if (hit === "ambiguous")
        throw new ActionRouteIssue(
          "ambiguous-route",
          `${routes[j]!.identity} ambiguously overlaps the ${routes[j]!.method} ${shapeOf(routes[j]!)} route of ${routes[i]!.identity}`,
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

function decodeSegments(pathname: string): readonly string[] | undefined {
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
  return decoded;
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
  decoded: readonly string[],
): ActionRouteCapture[] | undefined {
  if (!structural(route, decoded)) return undefined;
  const captures: ActionRouteCapture[] = [];
  for (let i = 0; i < route.segments.length; i++) {
    const segment = route.segments[i]!;
    if (segment.kind === "static") continue;
    if (segment.type === "int") {
      const value = strictInt(decoded[i]!);
      if (value === undefined) return undefined;
      captures.push(Object.freeze({ name: segment.name, value }));
    } else captures.push(Object.freeze({ name: segment.name, value: decoded[i]! }));
  }
  return captures;
}

export function matchActionRoute(
  table: ActionRouteTable,
  method: string,
  pathname: string,
): ActionRouteMatch {
  const decoded = typeof pathname === "string" ? decodeSegments(pathname) : undefined;
  if (decoded === undefined) return Object.freeze({ kind: "bad-request" });
  let best: { route: CompiledActionRoute; captures: readonly ActionRouteCapture[] } | undefined;
  let groupStructural = false;
  const foreign = new Set<string>();
  for (const route of table.routes) {
    const captures = typed(route, decoded);
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
