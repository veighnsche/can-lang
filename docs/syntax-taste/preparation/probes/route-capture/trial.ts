// Disposable policy trial against pinned Bun. This is not Can runtime code.
import { strict as assert } from "node:assert";

type Segment = { kind: "literal"; value: string } | { kind: "capture"; name: string; type: "int" | "str" };
type Route = { method: string; pattern: string; segments: Segment[]; staticCount: number };
type Result = { status: number; allow?: string; route?: string; captures?: Record<string, string | bigint> };
const INT_MIN = -(1n << 63n), INT_MAX = (1n << 63n) - 1n;

function parse(pattern: string, method = "GET", types: Record<string, "int" | "str"> = {}): Route {
  if (!pattern.startsWith("/") || pattern.startsWith("//")) throw Error("bad pattern root");
  const names = new Set<string>();
  const segments = pattern.slice(1).split("/").map((part): Segment => {
    if (part.startsWith(":")) {
      const name = part.slice(1);
      if (!/^[a-z][a-z0-9_]*$/.test(name) || names.has(name) || !types[name]) throw Error("bad capture declaration");
      names.add(name);
      return { kind: "capture", name, type: types[name] };
    }
    if (!/^[A-Za-z0-9._~-]+$/.test(part) || part === "." || part === "..") throw Error("bad literal segment");
    return { kind: "literal", value: part };
  });
  if (Object.keys(types).sort().join(",") !== [...names].sort().join(",")) throw Error("record/pattern mismatch");
  return { method, pattern, segments, staticCount: segments.filter((x) => x.kind === "literal").length };
}

function intersect(a: Route, b: Route): boolean {
  return a.segments.length === b.segments.length && a.segments.every((part, i) => {
    const other = b.segments[i];
    return part.kind === "capture" || other.kind === "capture" || part.value === other.value;
  });
}

function assemble(routes: Route[]): Route[] {
  for (let i = 0; i < routes.length; i++) for (let j = 0; j < i; j++) {
    const a = routes[i], b = routes[j];
    if (!intersect(a, b)) continue;
    if (a.pattern === b.pattern && a.method !== b.method) continue;
    if (a.pattern === b.pattern && a.method === b.method) throw Error("duplicate route");
    if (a.staticCount === b.staticCount) throw Error("ambiguous routes");
  }
  return routes;
}

function capture(type: "int" | "str", raw: string): string | bigint {
  const value = decodeURIComponent(raw);
  if (!value.isWellFormed() || value === "" || value === "." || value === ".." || /[\/\\\x00-\x1f\x7f]/.test(value)) throw Error("invalid segment");
  if (type === "str") return value;
  if (raw !== value) throw Error("noncanonical integer encoding");
  if (!/^(?:0|-[1-9][0-9]*|[1-9][0-9]*)$/.test(value)) throw Error("invalid integer spelling");
  const number = BigInt(value);
  if (number < INT_MIN || number > INT_MAX) throw Error("integer overflow");
  return number;
}

function build(route: Route, values: Record<string, string | bigint>): string {
  const expected = route.segments.filter((x) => x.kind === "capture").map((x) => x.name).sort();
  if (Object.keys(values).sort().join(",") !== expected.join(",")) throw Error("wrong capture fields");
  const parts = route.segments.map((part) => {
    if (part.kind === "literal") return part.value;
    const value = values[part.name];
    if (part.type === "int") {
      if (typeof value !== "bigint" || value < INT_MIN || value > INT_MAX) throw Error("invalid int value");
      return value.toString();
    }
    if (typeof value !== "string" || !value.isWellFormed() || value === "" || value === "." || value === ".." || /[\/\\\x00-\x1f\x7f]/.test(value)) throw Error("invalid str value");
    return encodeURIComponent(value);
  });
  return "/" + parts.join("/");
}

function match(routes: Route[], method: string, nativeUrl: string): Result {
  const path = new URL(nativeUrl).pathname;
  const raw = path.slice(1).split("/");
  // Existing ingress treats malformed percent/UTF-8/control anywhere as a bad path.
  try {
    for (const segment of raw) {
      const decoded = decodeURIComponent(segment);
      if (/[\x00-\x1f\x7f]/.test(decoded)) return { status: 400 };
    }
  } catch { return { status: 400 }; }
  const candidates = routes.filter((route) => route.segments.length === raw.length && route.segments.every((part, i) => part.kind === "capture" ? raw[i] !== "" : part.value === raw[i]));
  if (!candidates.length) return { status: 404 };
  const rank = Math.max(...candidates.map((route) => route.staticCount));
  const selected = candidates.filter((route) => route.staticCount === rank);
  const pattern = selected[0].pattern;
  const values: Record<string, string | bigint> = {};
  try {
    selected[0].segments.forEach((part, i) => { if (part.kind === "capture") values[part.name] = capture(part.type, raw[i]); });
  } catch { return { status: 400 }; }
  const allowed = selected.map((route) => route.method).sort();
  const route = selected.find((item) => item.method === method);
  return route ? { status: 200, route: route.pattern, captures: values } : { status: 405, allow: allowed.join(", ") };
}

let assertions = 0;
function eq(actual: unknown, expected: unknown) { assert.deepEqual(actual, expected); assertions++; }
function rejects(f: () => unknown) { assert.throws(f); assertions++; }

const invoice = parse("/tenants/:tenant_id/invoices/:invoice_id", "POST", { tenant_id: "int", invoice_id: "int" });
const slug = parse("/invoices/:slug", "POST", { slug: "str" });
const literal = parse("/invoices/new", "GET");
const routes = assemble([invoice, slug, literal]);
eq(build(invoice, { tenant_id: 1n, invoice_id: 7n }), "/tenants/1/invoices/7");
eq(build(slug, { slug: "A B%€" }), "/invoices/A%20B%25%E2%82%AC");
for (const value of ["a/b", "a\\b", ".", "..", "", "x\0", "\ud800"]) rejects(() => build(slug, { slug: value }));
for (const value of [0n, -1n, INT_MIN, INT_MAX]) {
  const path = build(invoice, { tenant_id: value, invoice_id: 7n });
  eq(match(routes, "POST", "http://local" + path).captures?.tenant_id, value);
}
for (const input of ["01", "+1", "-0", "1.0", "1e0", "%31", "%2D1", "9223372036854775808", "-9223372036854775809", "%ZZ", "%ED%A0%80", "%2F", "%5C", "%2E%2E", "%00"]) {
  const path = `/tenants/${input}/invoices/7`;
  const result = match(routes, "POST", "http://local" + path);
  // Bun normalizes dotdot into /invoices/7, which matches a different action.
  eq(result.status, input === "%2E%2E" ? 200 : 400);
}
eq(match(routes, "POST", "http://local/tenants/1/invoices/").status, 404);
eq(match(routes, "POST", "http://local/tenants/1/invoices/7/").status, 404);
eq(match(routes, "POST", "http://local/tenants/1/invoices/7/extra").status, 404);
eq(match(routes, "PATCH", "http://local/tenants/1/invoices/7"), { status: 405, allow: "POST" });
eq(match(routes, "POST", "http://local/invoices/new"), { status: 405, allow: "GET" });
eq(match(routes, "GET", "http://local/invoices/new"), { status: 200, route: "/invoices/new", captures: {} });
eq(match(routes, "POST", "http://local/invoices/x").captures?.slug, "x");
eq(match(routes, "POST", "http://local/invoices/A%20B%25%E2%82%AC").captures?.slug, "A B%€");
eq(match(routes, "POST", "http://local/invoices/a%252Fb").captures?.slug, "a%2Fb");
rejects(() => parse("/x/:id/:id", "GET", { id: "str" }));
rejects(() => parse("/x/:id", "GET", { id: "str", other: "str" }));
rejects(() => assemble([invoice, invoice]));
rejects(() => assemble([parse("/a/:x/c", "GET", { x: "str" }), parse("/a/b/:y", "GET", { y: "str" })]));
eq(assemble([parse("/a/:x", "GET", { x: "str" }), parse("/a/:x", "POST", { x: "str" })]).length, 2);

const served: { method: string; url: string; status: number; allow?: string }[] = [];
const server = Bun.serve({ hostname: "127.0.0.1", port: 0, fetch(req) {
  const result = match(routes, req.method, req.url);
  served.push({ method: req.method, url: req.url, status: result.status, allow: result.allow });
  return new Response(result.status === 200 ? "ok" : "rejected", { status: result.status, headers: result.allow ? { Allow: result.allow } : {} });
} });
const liveCases: [string, string, number, string?][] = [
  ["POST", "/tenants/1/invoices/7", 200],
  ["PATCH", "/tenants/1/invoices/7", 405, "POST"],
  ["POST", "/tenants/01/invoices/7", 400],
  ["POST", "/tenants/%ZZ/invoices/7", 400],
  ["POST", "/tenants/%2F/invoices/7", 400],
  ["POST", "/tenants/%2E/invoices/7", 404],
  ["POST", "/tenants/%2E%2E/invoices/7", 200],
  ["POST", "/tenants/1/invoices/7/", 404],
  ["POST", "/invoices/new", 405, "GET"],
  ["GET", "/invoices/new", 200],
  ["POST", "/invoices/A%20B%25%E2%82%AC", 200],
];
try {
  for (const [method, path, status, allow] of liveCases) {
    const child = Bun.spawn(["curl", "--silent", "--show-error", "--path-as-is", "--globoff", "--max-time", "2", "--dump-header", "-", "--output", "/dev/null", "--request", method, `http://127.0.0.1:${server.port}${path}`], { stdout: "pipe", stderr: "pipe" });
    const output = await new Response(child.stdout).text();
    eq(await child.exited, 0);
    assert.match(output, new RegExp(`^HTTP/1\\.1 ${status} `)); assertions++;
    if (allow) { assert.match(output.toLowerCase(), new RegExp(`^allow: ${allow.toLowerCase()}$`, "m")); assertions++; }
  }
} finally { server.stop(true); }

console.log(JSON.stringify({ bun: Bun.version, revision: Bun.revision, assertions, liveCases, served }, null, 2));
