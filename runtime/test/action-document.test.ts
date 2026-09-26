import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, type FailureShape } from "../domain.ts";
import { success, failure, value, errorType, errorPayload } from "../completion.ts";
import { record, dataProperty, recordIdentity } from "../data.ts";
import { runOwnedRoot, resourceStatus } from "../owner.ts";
import { createServer } from "../platform/server.ts";
import { createRouter } from "../platform/router.ts";
import {
  snapshotRequest,
  snapshotRequestLazy,
  nativeResponse,
  isRequest,
} from "../platform/http.ts";
import { createHTML, renderSafe } from "../platform/html.ts";
import { checkResponseHeaders } from "../platform/htmx-guard.ts";
import { createActionRoutes, readActionRouteBinding } from "../platform/action-routes.ts";

const identity = (kind: string, declaration: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration]))
    .digest("hex");
const textShape: FailureShape = {
  identity: identity("primitive", "str"),
  kind: "primitive",
  declaration: "str",
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
};
const intShape: FailureShape = {
  identity: identity("primitive", "int"),
  kind: "primitive",
  declaration: "int",
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
};
const fieldNames: Record<string, string[]> = {
  "http::invalid_server_config": ["reason"],
  "http::bind_failed": ["address"],
  "http::shutdown_failed": ["phase"],
  "http::invalid_route": ["reason"],
  "http::duplicate_route": ["method", "path"],
  "http::ambiguous_route": ["first", "second"],
  "action::invalid_path": ["reason"],
};
const declarations = catalogue.errors
  .filter((e) => fieldNames[e.name] !== undefined)
  .map((e) => ({ identity: e.identity, name: e.name, parameters: 0 }));
const fieldKinds = new Map(
  catalogue.errors.flatMap((e) => e.fields.map((f) => [e.name + ":" + f.name, f.type])),
);
const errorShapes: FailureShape[] = declarations.map((e) => ({
  identity: identity("error", e.identity),
  kind: "error",
  declaration: e.identity,
  arguments: [],
  fields: fieldNames[e.name]!.map((name) => ({
    name,
    type: fieldKinds.get(e.name + ":" + name) === "int" ? intShape.identity : textShape.identity,
  })),
  leaves: [],
  inputs: [],
  errors: [],
}));
const domain = createDomainRuntime({ declarations, shapes: [textShape, intShape, ...errorShapes] });
const id = (declaration: string) => identity("error", declaration);
const server = createServer(domain, {
  invalidConfig: id("can.std.http@1::invalid_server_config"),
  bindFailed: id("can.std.http@1::bind_failed"),
  shutdownFailed: id("can.std.http@1::shutdown_failed"),
});
const router = createRouter(domain, {
  invalid: id("can.std.http@1::invalid_route"),
  duplicate: id("can.std.http@1::duplicate_route"),
  ambiguous: id("can.std.http@1::ambiguous_route"),
});
const mounts = createActionRoutes(domain, {
  invalidRoute: id("can.std.http@1::invalid_route"),
  invalidPath: id("can.std.action@1::invalid_path"),
});
const htmlDecls = catalogue.errors.filter((e) => e.name === "html::invalid_structure");
const htmlShapes = htmlDecls.map((e) => ({
  identity: identity("error", e.identity),
  kind: "error",
  declaration: e.identity,
  fields: e.fields.map((f) => ({ name: f.name, type: textShape.identity })),
  arguments: [],
  leaves: [],
  inputs: [],
  errors: [],
}));
const htmlDomain = createDomainRuntime({
  declarations: htmlDecls.map((e) => ({ ...e, parameters: 0 })),
  shapes: [textShape, ...htmlShapes],
});
const html = createHTML(htmlDomain, {
  structure: htmlShapes[0]!.identity,
  url: "unused",
  target: "unused",
  interval: "unused",
});

// Document-site fixtures mirror the spliced metadata shape: identity,
// method, :name template, ordered captures, bodyless input, returns, body
// html and the exhaustive case table. No JSON response schema (HTML
// actions omit it) and no form structural identities (no form decode).
const keyDecl = "up10::invoice_key";
const keyIdentity = identity("record", keyDecl);
const docSite = {
  action: "up10::read_invoice",
  method: "GET",
  path: "/tenants/:tenant_id/invoices/:invoice_id",
  capturesType: keyDecl,
  captures: [
    { name: "tenant_id", type: "int" },
    { name: "invoice_id", type: "int" },
  ],
  input: { mode: "none" },
  returns: "up10::read_outcome",
  body: "html",
  cases: [
    { leaf: "up10::shown", status: 200 },
    { leaf: "up10::denied", status: 404 },
  ],
};
const calls: string[] = [];
const seen: { request?: unknown } = {};
const docHandler = async (request: unknown, key: unknown) => {
  const tenant = dataProperty(key, "tenant_id") as bigint;
  const invoice = dataProperty(key, "invoice_id") as bigint;
  calls.push(`read:${tenant},${invoice}`);
  expect(recordIdentity(key)).toBe(keyIdentity);
  seen.request = request;
  return tenant < 0n
    ? success(record("up10::denied", [["reason", "foreign"]]))
    : success(record("up10::shown", [["label", `t${tenant}i${invoice}`]]));
};
const docRenderer = async (result: unknown) => {
  const label =
    recordIdentity(result) === "up10::shown" ? (dataProperty(result, "label") as string) : "denied";
  calls.push(`document:${recordIdentity(result)}`);
  const body = value(
    await html.element(value(await html.makeTag("main")), [], [value(await html.text(label))]),
  );
  return success(value(await html.document(`invoice ${label}`, [], [body])));
};
const fragSite = {
  action: "up10::save_note",
  method: "POST",
  path: "/tenants/:tenant_id/notes",
  capturesType: "up10::tenant_key",
  captures: [{ name: "tenant_id", type: "int" }],
  input: {
    mode: "form",
    type: "up10::note_form",
    limit: 2048,
    schema: { root: "wire", fields: [{ name: "customer", kind: "str" }] },
  },
  returns: "up10::note_outcome",
  body: "html",
  cases: [
    { leaf: "up10::saved", status: 200, swap: "inner" },
    { leaf: "up10::failed", status: 409, swap: "inner" },
  ],
  rejected: "up10::rejected",
  rawEntry: "up10::raw_entry",
  issue: "up10::issue",
};
const fragHandler = async (request: unknown, key: unknown, form: unknown) => {
  const customer = dataProperty(form, "customer") as string;
  calls.push(`frag:${customer}`);
  expect(recordIdentity(key)).toBe(identity("record", "up10::tenant_key"));
  return customer === "stale"
    ? success(record("up10::failed", []))
    : success(record("up10::saved", []));
};
const fragOutcome = async (result: unknown) => {
  calls.push(`frag-outcome:${recordIdentity(result)}`);
  return success(
    value(await html.fragment([value(await html.text(`ok:${recordIdentity(result)}`))])),
  );
};
const fragStructural = async (_bad: unknown) => {
  calls.push("frag-structural");
  return success(value(await html.fragment([value(await html.text("bad"))])));
};

async function serve(port: number, routes: unknown[]): Promise<unknown> {
  const config = value(await server.makeConfig("127.0.0.1", BigInt(port), 65536n, 5000n));
  const table = value(await router.make(routes));
  return value(await server.start(config, table));
}

async function stop(token: unknown): Promise<void> {
  expect(resourceStatus(token)).toMatchObject({ kind: "server", state: "open", leases: 0 });
  expect((await server.stop(token)).kind).toBe("ok");
  expect(resourceStatus(token)).toMatchObject({ kind: "server", state: "closed", leases: 0 });
}

function failedField(completion: unknown, name: string): unknown {
  return dataProperty(errorPayload(completion as never), name);
}

test("document mounts serve full-page reads with case statuses over live HTTP", async () => {
  calls.length = 0;
  const owned = await runOwnedRoot(async () => {
    const read = value(await mounts.mountDocument(docHandler, docRenderer, docSite));
    const token = await serve(18731, [read]);
    const base = "http://127.0.0.1:18731";
    const shown = await fetch(base + "/tenants/1/invoices/7");
    expect(shown.status).toBe(200);
    expect(shown.headers.get("content-type")).toBe("text/html; charset=utf-8");
    const page = await shown.text();
    expect(page.startsWith("<!doctype html>")).toBe(true);
    expect(page).toContain("<title>invoice t1i7</title>");
    expect(page).toContain("<main>t1i7</main>");
    // Every case renders a body: the denied leaf serves a full page under
    // its declared 404, not a fragment and not an empty status.
    const denied = await fetch(base + "/tenants/-1/invoices/7");
    expect(denied.status).toBe(404);
    expect(denied.headers.get("content-type")).toBe("text/html; charset=utf-8");
    const deniedPage = await denied.text();
    expect(deniedPage.startsWith("<!doctype html>")).toBe(true);
    expect(deniedPage).toContain("<main>denied</main>");
    // The adapter sets no response-control headers on either mode: no
    // hx-* header, no location, no redirect can ride a document out.
    for (const response of [shown, denied]) {
      for (const name of response.headers.keys()) {
        expect(name.startsWith("hx-")).toBe(false);
      }
      expect(response.headers.get("location")).toBe(null);
    }
    // Canonical builder round-trips through dispatch for the same action.
    const key = record(keyIdentity, [
      ["tenant_id", 3n],
      ["invoice_id", 9n],
    ]);
    const built = value(
      await mounts.url(
        key,
        {
          action: "up10::read_invoice",
          path: "/tenants/:tenant_id/invoices/:invoice_id",
          captures: docSite.captures,
        },
        undefined,
      ),
    );
    expect(built).toBe("/tenants/3/invoices/9");
    const round = await fetch(base + (built as string));
    expect(round.status).toBe(200);
    expect(await round.text()).toContain("<main>t3i9</main>");
    expect(calls).toEqual([
      "read:1,7",
      "document:up10::shown",
      "read:-1,7",
      "document:up10::denied",
      "read:3,9",
      "document:up10::shown",
    ]);
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
  // Request-scoped ownership like every other mount: the retained token
  // is revoked once the response completes.
  expect(isRequest(seen.request)).toBe(false);
});

test("document and fragment modes coexist on one server with unchanged fragments", async () => {
  calls.length = 0;
  const owned = await runOwnedRoot(async () => {
    const read = value(await mounts.mountDocument(docHandler, docRenderer, docSite));
    const save = value(await mounts.mountForm(fragHandler, fragOutcome, fragStructural, fragSite));
    const token = await serve(18732, [read, save]);
    const base = "http://127.0.0.1:18732";
    const page = await fetch(base + "/tenants/1/invoices/7");
    expect(page.status).toBe(200);
    expect(await page.text()).toContain("<!doctype html>");
    // The fragment mount keeps its exact prior behavior: fragment bytes
    // under the declared status, no doctype, same renderer sequence.
    const saved = await fetch(base + "/tenants/1/notes", {
      method: "POST",
      headers: { "content-type": "application/x-www-form-urlencoded" },
      body: "customer=amy",
    });
    expect(saved.status).toBe(200);
    expect(saved.headers.get("content-type")).toBe("text/html; charset=utf-8");
    const fragment = await saved.text();
    expect(fragment).toBe("ok:up10::saved");
    expect(fragment).not.toContain("doctype");
    const failed = await fetch(base + "/tenants/1/notes", {
      method: "POST",
      headers: { "content-type": "application/x-www-form-urlencoded" },
      body: "customer=stale",
    });
    expect(failed.status).toBe(409);
    expect(await failed.text()).toBe("ok:up10::failed");
    expect(calls).toEqual([
      "read:1,7",
      "document:up10::shown",
      "frag:amy",
      "frag-outcome:up10::saved",
      "frag:stale",
      "frag-outcome:up10::failed",
    ]);
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("document dispatch rejects bodies, unknown outcomes and bad renders", async () => {
  const binding = value(await mounts.mountDocument(docHandler, docRenderer, docSite));
  const callback = readActionRouteBinding(binding).callback;
  const captures = [
    { name: "tenant_id", value: 1n },
    { name: "invoice_id", value: 7n },
  ];
  const statusOf = async (request: Request, limit: number, lazy = false) => {
    const snapshot = lazy
      ? await snapshotRequestLazy(request, limit)
      : await snapshotRequest(request, limit);
    if (snapshot.kind !== "request") throw new Error("rejected at ingress");
    const completed = await callback(snapshot.value, captures);
    if (completed.kind !== "ok") return completed;
    return nativeResponse(completed.value);
  };
  // The document callback never consults the wire method; a POST-shaped
  // snapshot exercises the same bodyless probe live GETs pass through.
  const bodied = (await statusOf(
    new Request("http://localhost/tenants/1/invoices/7", { method: "POST", body: "x=1" }),
    65536,
  )) as Response;
  expect(bodied.status).toBe(400);
  expect(await bodied.text()).toBe("Bad Request");
  const flooded = (await statusOf(
    new Request("http://localhost/tenants/1/invoices/7", { method: "POST", body: "x".repeat(64) }),
    8,
    true,
  )) as Response;
  expect(flooded.status).toBe(413);
  const clean = (await statusOf(
    new Request("http://localhost/tenants/1/invoices/7", { method: "POST" }),
    65536,
  )) as Response;
  expect(clean.status).toBe(200);
  expect(clean.headers.get("content-type")).toBe("text/html; charset=utf-8");
  expect(await clean.text()).toContain("<main>t1i7</main>");
  // Handler failures and undeclared leaves are fixed 500s with no detail.
  const failing = readActionRouteBinding(
    value(
      await mounts.mountDocument(
        async () =>
          failure(
            domain.create(id("can.std.http@1::invalid_route"), record("r", []), {
              source: "t",
              start: 0,
              end: 0,
              invocation: [],
            }),
          ),
        docRenderer,
        docSite,
      ),
    ),
  ).callback;
  const empty = await snapshotRequest(new Request("http://localhost/"), 64);
  if (empty.kind !== "request") throw new Error("rejected at ingress");
  const failed = await failing(empty.value, captures);
  expect(failed.kind).toBe("ok");
  expect(nativeResponse(value(failed)).status).toBe(500);
  const stray = readActionRouteBinding(
    value(
      await mounts.mountDocument(
        async () => success(record("up10::stray", [])),
        docRenderer,
        docSite,
      ),
    ),
  ).callback;
  const undeclared = await stray(empty.value, captures);
  expect(undeclared.kind).toBe("ok");
  expect(nativeResponse(value(undeclared)).status).toBe(500);
  // Renderer faults return as-is: the server maps them to an uncertain
  // 500 while the fault detail stays observable to direct dispatch.
  const broken = readActionRouteBinding(
    value(
      await mounts.mountDocument(
        docHandler,
        async () => {
          throw new Error("render-fault");
        },
        docSite,
      ),
    ),
  ).callback;
  const faulted = await broken(empty.value, captures);
  expect(faulted.kind).not.toBe("ok");
});

test("mountDocument rejects malformed sites, callables and templates", async () => {
  const bad: [string, unknown, unknown, unknown][] = [
    ["null site", docHandler, docRenderer, null],
    ["missing action", docHandler, docRenderer, { ...docSite, action: "" }],
    ["bad method", docHandler, docRenderer, { ...docSite, method: "PUT" }],
    ["POST at document entry", docHandler, docRenderer, { ...fragSite }],
    ["JSON body at document entry", docHandler, docRenderer, { ...docSite, body: "json" }],
    [
      "form input at document entry",
      docHandler,
      docRenderer,
      { ...docSite, input: fragSite.input },
    ],
    ["missing path", docHandler, docRenderer, { ...docSite, path: "" }],
    [
      "bad captures",
      docHandler,
      docRenderer,
      { ...docSite, captures: [{ name: "x", type: "bool" }] },
    ],
    ["missing returns", docHandler, docRenderer, { ...docSite, returns: "" }],
    ["bad body mode", docHandler, docRenderer, { ...docSite, body: "xml" }],
    ["missing cases", docHandler, docRenderer, { ...docSite, cases: [] }],
    [
      "bad case status",
      docHandler,
      docRenderer,
      { ...docSite, cases: [{ leaf: "x", status: 204 }] },
    ],
    ["non-function handler", "nope", docRenderer, docSite],
    ["non-function renderer", docHandler, "nope", docSite],
  ];
  expect(bad.length).toBe(14);
  for (const [, handler, renderer, site] of bad) {
    await expect(mounts.mountDocument(handler, renderer, site)).rejects.toThrow(TypeError);
  }
  // Each entry serves exactly one shape: the document site reaches neither
  // the JSON nor the form mount, and neither foreign site reaches this one.
  await expect(mounts.mount(docHandler, docSite)).rejects.toThrow(TypeError);
  await expect(mounts.mountForm(docHandler, docRenderer, fragStructural, docSite)).rejects.toThrow(
    TypeError,
  );
  await expect(mounts.mountDocument(fragHandler, fragOutcome, fragSite)).rejects.toThrow(TypeError);
  // Template faults fail as the declared invalid_route, never a throw.
  const completed = await mounts.mountDocument(docHandler, docRenderer, {
    ...docSite,
    path: "/__can/x",
  });
  expect(completed.kind).toBe("domain");
  expect(errorType(completed as never)).toBe(id("can.std.http@1::invalid_route"));
  expect(failedField(completed, "reason")).toBe("path");
});

test("the HTML guard policy applies to document responses alike", async () => {
  const node = (connected: boolean) => ({ isConnected: connected });
  // A clean document response — full-page status, text/html, no control
  // headers — is admitted exactly like a clean fragment response.
  for (const status of [200, 404]) {
    const verdict = checkResponseHeaders({
      target: node(true),
      response: { status, headers: new Headers({ "content-type": "text/html; charset=utf-8" }) },
    });
    expect(verdict).toEqual({ admit: true });
  }
  // Control headers and fetch-level redirects cancel against document
  // responses with the same finite occurrences as fragments.
  const controlled = checkResponseHeaders({
    target: node(true),
    response: { status: 200, headers: new Headers([["HX-Redirect", "/elsewhere"]]) },
  });
  expect(controlled.admit).toBe(false);
  expect(controlled.occurrence).toEqual({
    kind: "action::protocol",
    phase: "response",
    status: 200,
    effect: "uncertain",
    reason: "control_header",
    header: "redirect",
  });
  const redirected = checkResponseHeaders({
    target: node(true),
    response: { status: 200, headers: new Headers(), redirected: true },
  });
  expect(redirected.admit).toBe(false);
  expect(redirected.occurrence?.reason).toBe("redirect");
});

test("document URLs gain no swap exceptions while fragment policies still derive", async () => {
  // Document actions declare no swap cases, so the emitter table omits
  // them; derivation for a document URL serializes exactly as before.
  const guarded = createHTML(
    htmlDomain,
    {
      structure: htmlShapes[0]!.identity,
      url: "unused",
      target: "unused",
      interval: "unused",
    },
    [],
    [
      {
        method: "POST",
        segments: ["tenants", "{}", "notes"],
        cases: [{ status: 409, swap: "inner" }],
      },
    ],
  );
  const tag = async (name: string) => value(await guarded.makeTag(name));
  const text = async (s: string) => value(await guarded.text(s));
  const url = async (s: string) => value(await guarded.parseURL(s));
  const page = value(
    await guarded.element(
      await tag("button"),
      [value(await guarded.get(await url("/tenants/1/invoices/7")))],
      [await text("open")],
    ),
  );
  expect(renderSafe(value(await guarded.fragment([page])))).toBe(
    '<button hx-get="/tenants/1/invoices/7">open</button>',
  );
  // Positive control: the fragment policy on the same factory still
  // re-admits its declared error status through an exact attribute.
  const inner = "{&quot;swap&quot;:&quot;innerHTML&quot;}";
  const form = value(
    await guarded.element(
      await tag("form"),
      [value(await guarded.post(await url("/tenants/1/notes")))],
      [await text("save")],
    ),
  );
  expect(renderSafe(value(await guarded.fragment([form])))).toBe(
    `<form hx-post="/tenants/1/notes" hx-status:409="${inner}">save</form>`,
  );
});
