// Disposable Bun.routes + strict Can-capture adapter trial, not runtime code.
import { strict as assert } from "node:assert";
const min = -(1n << 63n), max = (1n << 63n) - 1n;
const seen: { route: string; values?: Record<string, unknown> }[] = [];
function error(status: number, allow?: string) { return new Response("rejected", { status, headers: allow ? { Allow: allow } : {} }); }
function rawParts(req: Request): string[] | undefined {
  const path = new URL(req.url).pathname;
  const raw = path.slice(1).split("/");
  try {
    for (const part of raw) if (/[\x00-\x1f\x7f]/.test(decodeURIComponent(part))) return undefined;
  } catch { return undefined; }
  return raw;
}
function str(value: string): boolean {
  return value.isWellFormed() && value !== "" && value !== "." && value !== ".." && !/[\/\\\x00-\x1f\x7f]/.test(value);
}
function int(value: string): bigint | undefined {
  if (!/^(?:0|-[1-9][0-9]*|[1-9][0-9]*)$/.test(value)) return undefined;
  const parsed = BigInt(value);
  return parsed < min || parsed > max ? undefined : parsed;
}
function invoice(req: Bun.BunRequest<"/tenants/:tenant_id/invoices/:invoice_id">) {
  const raw = rawParts(req), a = int(req.params.tenant_id), b = int(req.params.invoice_id);
  if (!raw || a === undefined || b === undefined || raw.length !== 4 || raw[0] !== "tenants" || raw[2] !== "invoices" || raw[1] !== a.toString() || raw[3] !== b.toString()) return error(400);
  seen.push({ route: "invoice", values: { tenant_id: a.toString(), invoice_id: b.toString() } });
  return new Response("invoice");
}
function slug(req: Bun.BunRequest<"/invoices/:slug">) {
  const raw = rawParts(req), value = req.params.slug;
  if (!raw || !str(value) || raw.length !== 2 || raw[0] !== "invoices") return error(400);
  try { if (decodeURIComponent(raw[1]) !== value) return error(400); } catch { return error(400); }
  seen.push({ route: "slug", values: { slug: value } });
  return new Response("slug");
}
function fallback(req: Request) {
  const raw = rawParts(req);
  if (!raw) return error(400);
  if (raw.length === 4 && raw[0] === "tenants" && raw[2] === "invoices" && raw[1] && raw[3]) {
    if (raw[1] !== decodeURIComponent(raw[1]) || raw[3] !== decodeURIComponent(raw[3]) || int(raw[1]) === undefined || int(raw[3]) === undefined) return error(400);
    return error(405, "POST");
  }
  if (raw.length === 2 && raw[0] === "invoices" && raw[1]) {
    const value = decodeURIComponent(raw[1]);
    if (!str(value)) return error(400);
    return error(405, raw[1] === "new" ? "GET, POST, PUT" : "POST, PUT");
  }
  return error(404);
}
const server = Bun.serve({ hostname: "127.0.0.1", port: 0,
  routes: {
    "/tenants/:tenant_id/invoices/:invoice_id": { POST: invoice },
    "/invoices/new": { GET: () => { seen.push({ route: "static" }); return new Response("static"); }, PUT: () => { seen.push({ route: "static" }); return new Response("static"); } },
    "/invoices/:slug": { POST: slug, PUT: slug },
  },
  fetch: fallback,
});
const cases: [string, string, number, string?][] = [
  ["POST", "/tenants/1/invoices/7", 200],
  ["POST", "/tenants/0/invoices/-1", 200],
  ["POST", "/tenants/-9223372036854775808/invoices/9223372036854775807", 200],
  ["POST", "/tenants/9223372036854775808/invoices/7", 400],
  ["POST", "/tenants/1/invoices/-9223372036854775809", 400],
  ["PATCH", "/tenants/1/invoices/7", 405, "POST"],
  ["PATCH", "/tenants/01/invoices/7", 400],
  ["POST", "/tenants/01/invoices/7", 400],
  ["POST", "/tenants/%31/invoices/7", 400],
  ["POST", "/tenants/%ZZ/invoices/7", 400],
  ["POST", "/tenants/%ED%A0%80/invoices/7", 400],
  ["POST", "/tenants/%2F/invoices/7", 400],
  ["POST", "/tenants/%5C/invoices/7", 400],
  ["POST", "/tenants/%2E/invoices/7", 400],
  ["POST", "/tenants/%2E%2E/invoices/7", 400],
  ["POST", "/tenants/a\\b/invoices/7", 400],
  ["POST", "/tenants/1/invoices/", 404],
  ["POST", "/tenants/1/invoices/7/", 404],
  ["POST", "/tenants/1/invoices/7/extra", 404],
  ["POST", "/invoices/new", 200],
  ["GET", "/invoices/new", 200],
  ["PUT", "/invoices/new", 200],
  ["PATCH", "/invoices/new", 405, "GET, POST, PUT"],
  ["POST", "/invoices/A%20B%25%E2%82%AC", 200],
  ["POST", "/invoices/a%252Fb", 200],
  ["POST", "/invoices/%2F", 400],
  ["POST", "/invoices/%2E", 400],
];
const results: { method: string; path: string; status: number; allow: string | null; body: string }[] = [];
try {
  for (const [method, path, expected, allow] of cases) {
    const child = Bun.spawn(["curl", "--silent", "--show-error", "--path-as-is", "--globoff", "--max-time", "2", "--include", "--request", method, `http://127.0.0.1:${server.port}${path}`], { stdout: "pipe", stderr: "pipe" });
    const output = await new Response(child.stdout).text();
    assert.equal(await child.exited, 0);
    const status = Number(output.match(/^HTTP\/1\.1 (\d+)/)?.[1]);
    const receivedAllow = output.match(/^allow: ([^\r\n]+)/im)?.[1] ?? null;
    const body = output.split("\r\n\r\n")[1] ?? "";
    assert.equal(status, expected, `${method} ${path}`);
    assert.equal(receivedAllow, allow ?? null, `${method} ${path} Allow`);
    results.push({ method, path, status, allow: receivedAllow, body });
  }
} finally { server.stop(true); }
assert.equal(seen.length, cases.filter((x) => x[2] === 200).length);
console.log(JSON.stringify({ bun: Bun.version, revision: Bun.revision, assertions: cases.length * 3 + 1, cases: results, callbacks: seen }, null, 2));
