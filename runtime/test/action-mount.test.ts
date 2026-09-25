import { test, expect } from "bun:test";
import { connect } from "node:net";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, type FailureShape } from "../domain.ts";
import { success, failure, value, errorType, errorPayload } from "../completion.ts";
import { record, array, dataArray, dataProperty, recordIdentity } from "../data.ts";
import { runOwnedRoot, resourceStatus } from "../owner.ts";
import { createServer } from "../platform/server.ts";
import { createRouter, dispatch } from "../platform/router.ts";
import { snapshotRequest, nativeResponse, ownedResponse } from "../platform/http.ts";
import { createHTML } from "../platform/html.ts";
import type { Schema } from "../codec/json.ts";
import type { FormSchema } from "../platform/form.ts";
import { createActionRoutes } from "../platform/action-routes.ts";

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
const origin = { source: "can:test", start: 0, end: 0, invocation: [] };
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
const fragment = async (text: string) => value(await html.fragment([value(await html.text(text))]));

// Mount-site fixtures mirror the UP08 spliced metadata shape: identity,
// method, :name template, ordered captures, input codec, returns, body
// mode, the exhaustive case table and the JSON or HTML codec contracts.
const keyDecl = "up09::invoice_key";
const keyIdentity = identity("record", keyDecl);
const strNode = { identity: "up09::str", kind: "primitive", name: "str" };
const intNode = { identity: "up09::int", kind: "primitive", name: "int" };
const loadResponse: Schema = {
  root: "up09::grid_load_outcome",
  nodes: [
    {
      identity: "up09::grid_load_outcome",
      kind: "variant",
      name: "grid_load_outcome",
      leaves: ["up09::grid_loaded", "up09::grid_denied"],
    },
    {
      identity: "up09::grid_loaded",
      kind: "record",
      name: "grid_loaded",
      fields: [{ name: "label", type: "up09::str" }],
    },
    {
      identity: "up09::grid_denied",
      kind: "record",
      name: "grid_denied",
      fields: [{ name: "reason", type: "up09::str" }],
    },
    strNode,
  ],
};
const saveRequest: Schema = {
  root: "up09::grid_edit_input",
  nodes: [
    {
      identity: "up09::grid_edit_input",
      kind: "record",
      name: "grid_edit_input",
      fields: [
        { name: "operation_id", type: "up09::str" },
        { name: "label", type: "up09::str" },
      ],
    },
    strNode,
  ],
};
const saveResponse: Schema = {
  root: "up09::grid_edit_outcome",
  nodes: [
    {
      identity: "up09::grid_edit_outcome",
      kind: "variant",
      name: "grid_edit_outcome",
      leaves: ["up09::grid_saved", "up09::grid_failed"],
    },
    {
      identity: "up09::grid_saved",
      kind: "record",
      name: "grid_saved",
      fields: [{ name: "operation_id", type: "up09::str" }],
    },
    {
      identity: "up09::grid_failed",
      kind: "record",
      name: "grid_failed",
      fields: [
        { name: "operation_id", type: "up09::str" },
        { name: "reason", type: "up09::str" },
      ],
    },
    strNode,
  ],
};
const formSchema: FormSchema = {
  root: "up09::invoice_form",
  fields: [
    { name: "customer", kind: "str" },
    {
      name: "lines",
      kind: "rows",
      rows: {
        row: "up09::line_wire",
        order: "lines_order",
        collection: "up09::line_rows",
        item: "up09::line_item",
        fields: [{ name: "sku", kind: "str" }],
      },
    },
  ],
};
const captures = [
  { name: "tenant_id", type: "int" },
  { name: "invoice_id", type: "int" },
];
const loadSite = {
  action: "up09::load_grid",
  method: "GET",
  path: "/api/tenants/:tenant_id/invoices/:invoice_id",
  capturesType: keyDecl,
  captures,
  input: { mode: "none" },
  returns: "up09::grid_load_outcome",
  body: "json",
  cases: [
    { leaf: "up09::grid_loaded", status: 200 },
    { leaf: "up09::grid_denied", status: 403 },
  ],
  responseSchema: loadResponse,
};
const saveSite = {
  action: "up09::save_grid",
  method: "POST",
  path: "/api/tenants/:tenant_id/invoices/:invoice_id",
  capturesType: keyDecl,
  captures,
  input: { mode: "json", type: "up09::grid_edit_input", limit: 256, schema: saveRequest },
  returns: "up09::grid_edit_outcome",
  body: "json",
  cases: [
    { leaf: "up09::grid_saved", status: 200 },
    { leaf: "up09::grid_failed", status: 422 },
  ],
  responseSchema: saveResponse,
};
const formSite = {
  action: "up09::save_html",
  method: "POST",
  path: "/tenants/:tenant_id/invoices/:invoice_id",
  capturesType: keyDecl,
  captures,
  input: { mode: "form", type: "up09::invoice_form", limit: 2048, rowsLimit: 2, schema: formSchema },
  returns: "up09::edit_outcome",
  body: "html",
  cases: [
    { leaf: "up09::saved", status: 200, swap: "inner" },
    { leaf: "up09::failed", status: 409, swap: "inner" },
  ],
  rejected: "up09::rejected",
  rawEntry: "up09::raw_entry",
  issue: "up09::issue",
};
const staticSite = {
  action: "up09::load_new",
  method: "GET",
  path: "/invoices/new",
  captures: [],
  input: { mode: "none" },
  returns: "up09::grid_load_outcome",
  body: "json",
  cases: [
    { leaf: "up09::grid_loaded", status: 200 },
    { leaf: "up09::grid_denied", status: 403 },
  ],
  responseSchema: loadResponse,
};

const calls: string[] = [];
const loadHandler = async (request: unknown, key: unknown) => {
  const tenant = dataProperty(key, "tenant_id") as bigint;
  const invoice = dataProperty(key, "invoice_id") as bigint;
  calls.push(`load:${tenant},${invoice}`);
  // Adapter-built captures carry the true hashed record identity, so
  // nominal matches observe exactly what emitted construction produces.
  expect(recordIdentity(key)).toBe(keyIdentity);
  expect(typeof request).toBe("object");
  return tenant < 0n
    ? success(record("up09::grid_denied", [["reason", "foreign"]]))
    : success(record("up09::grid_loaded", [["label", `t${tenant}i${invoice}`]]));
};
const saveHandler = async (request: unknown, key: unknown, body: unknown) => {
  const tenant = dataProperty(key, "tenant_id") as bigint;
  const invoice = dataProperty(key, "invoice_id") as bigint;
  const operation = dataProperty(body, "operation_id") as string;
  calls.push(`save:${tenant},${invoice},${operation}`);
  expect(recordIdentity(key)).toBe(keyIdentity);
  return operation === ""
    ? success(
        record("up09::grid_failed", [
          ["operation_id", operation],
          ["reason", "empty"],
        ]),
      )
    : success(record("up09::grid_saved", [["operation_id", operation]]));
};
const staticHandler = async () => {
  calls.push("static");
  return success(record("up09::grid_loaded", [["label", "new"]]));
};
const slugSite = {
  action: "up09::save_slug",
  method: "POST",
  path: "/invoices/:slug",
  capturesType: "up09::slug_key",
  captures: [{ name: "slug", type: "str" }],
  input: { mode: "json", type: "up09::grid_edit_input", limit: 256, schema: saveRequest },
  returns: "up09::grid_edit_outcome",
  body: "json",
  cases: [
    { leaf: "up09::grid_saved", status: 200 },
    { leaf: "up09::grid_failed", status: 422 },
  ],
  responseSchema: saveResponse,
};
const slugHandler = async (request: unknown, key: unknown, body: unknown) => {
  const slug = dataProperty(key, "slug") as string;
  const operation = dataProperty(body, "operation_id") as string;
  calls.push(`slug:${slug},${operation}`);
  expect(recordIdentity(key)).toBe(identity("record", "up09::slug_key"));
  return success(record("up09::grid_saved", [["operation_id", operation]]));
};
const formHandler = async (request: unknown, key: unknown, form: unknown) => {
  const tenant = dataProperty(key, "tenant_id") as bigint;
  const customer = dataProperty(form, "customer") as string;
  calls.push(`form:${tenant},${customer}`);
  expect(recordIdentity(key)).toBe(keyIdentity);
  return customer === "stale"
    ? success(record("up09::failed", []))
    : success(record("up09::saved", []));
};
const outcomeRenderer = async (result: unknown) => {
  calls.push(`outcome:${recordIdentity(result)}`);
  return success(await fragment(`ok:${recordIdentity(result)}`));
};
const structuralRenderer = async (bad: unknown) => {
  const raw = dataArray(dataProperty(bad, "raw"))
    .map((entry) => dataProperty(entry, "value") as string)
    .join("|");
  calls.push(`structural:${raw}`);
  return success(await fragment(`bad:${raw}`));
};

async function serve(port: number, routes: unknown[], bodyLimit = 65536n): Promise<unknown> {
  const config = value(await server.makeConfig("127.0.0.1", BigInt(port), bodyLimit, 5000n));
  const table = value(await router.make(routes));
  return value(await server.start(config, table));
}

async function stop(token: unknown): Promise<void> {
  expect(resourceStatus(token)).toMatchObject({ kind: "server", state: "open", leases: 0 });
  expect((await server.stop(token)).kind).toBe("ok");
  expect(resourceStatus(token)).toMatchObject({ kind: "server", state: "closed", leases: 0 });
}

function rawRequest(
  port: number,
  target: string,
  method = "GET",
  body = "",
  headers: Record<string, string> = {},
): Promise<{ status: number; allow: string | null; body: string }> {
  return new Promise((resolve, reject) => {
    const sock = connect(port, "127.0.0.1", () => {
      const head = [`${method} ${target} HTTP/1.1`, "Host: x", "Connection: close"];
      for (const [name, text] of Object.entries(headers)) head.push(`${name}: ${text}`);
      if (body !== "") head.push(`Content-Length: ${Buffer.byteLength(body)}`);
      sock.write(head.join("\r\n") + "\r\n\r\n" + body);
    });
    let data = "";
    sock.on("data", (chunk) => {
      data += chunk.toString("latin1");
    });
    sock.on("close", () => {
      const head = data.split("\r\n\r\n")[0] ?? "";
      const text = data.split("\r\n\r\n").slice(1).join("\r\n\r\n");
      const allow = head.match(/^allow: ([^\r\n]+)/im)?.[1] ?? null;
      resolve({ status: Number(head.split(" ")[1]), allow, body: text });
    });
    sock.on("error", reject);
  });
}

function failedField(completion: unknown, name: string): unknown {
  return dataProperty(errorPayload(completion as never), name);
}

test("mounts serve captured JSON actions over live HTTP", async () => {
  calls.length = 0;
  const owned = await runOwnedRoot(async () => {
    const load = value(await mounts.mount(loadHandler, loadSite));
    const save = value(await mounts.mount(saveHandler, saveSite));
    const fresh = value(await mounts.mount(staticHandler, staticSite));
    const token = await serve(18611, [load, save, fresh]);
    const base = "http://127.0.0.1:18611";
    const loaded = await fetch(base + "/api/tenants/1/invoices/7");
    expect(loaded.status).toBe(200);
    expect(loaded.headers.get("content-type")).toBe("application/json; charset=utf-8");
    expect(await loaded.text()).toBe('{"case":"grid_loaded","value":{"label":"t1i7"}}');
    const denied = await fetch(base + "/api/tenants/-1/invoices/7");
    expect(denied.status).toBe(403);
    expect(await denied.text()).toBe('{"case":"grid_denied","value":{"reason":"foreign"}}');
    const saved = await fetch(base + "/api/tenants/1/invoices/7", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ operation_id: "op-1", label: "r1" }),
    });
    expect(saved.status).toBe(200);
    expect(await saved.text()).toBe('{"case":"grid_saved","value":{"operation_id":"op-1"}}');
    const failed = await fetch(base + "/api/tenants/1/invoices/7", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ operation_id: "", label: "r1" }),
    });
    expect(failed.status).toBe(422);
    expect(await failed.text()).toBe(
      '{"case":"grid_failed","value":{"operation_id":"","reason":"empty"}}',
    );
    const created = await fetch(base + "/invoices/new");
    expect(created.status).toBe(200);
    expect(await created.text()).toBe('{"case":"grid_loaded","value":{"label":"new"}}');
    // Canonical builder round-trips through dispatch for the same action.
    const key = record(keyIdentity, [
      ["tenant_id", 3n],
      ["invoice_id", 9n],
    ]);
    const built = value(
      await mounts.url(
        key,
        {
          action: "up09::load_grid",
          path: "/api/tenants/:tenant_id/invoices/:invoice_id",
          captures,
        },
        undefined,
      ),
    );
    expect(built).toBe("/api/tenants/3/invoices/9");
    const round = await fetch(base + (built as string));
    expect(round.status).toBe(200);
    expect(await round.text()).toBe('{"case":"grid_loaded","value":{"label":"t3i9"}}');
    const bare = value(
      await mounts.url(
        {
          action: "up09::load_new",
          path: "/invoices/new",
          captures: [],
        },
        undefined,
      ),
    );
    expect(bare).toBe("/invoices/new");
    expect(calls).toEqual([
      "load:1,7",
      "load:-1,7",
      "save:1,7,op-1",
      "save:1,7,",
      "static",
      "load:3,9",
    ]);
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("mount rejects malformed sites, callables and templates", async () => {
  const bad: [string, unknown, unknown][] = [
    ["null site", loadHandler, null],
    ["missing action", loadHandler, { ...loadSite, action: "" }],
    ["bad method", loadHandler, { ...loadSite, method: "PUT" }],
    ["missing path", loadHandler, { ...loadSite, path: "" }],
    ["bad captures", loadHandler, { ...loadSite, captures: [{ name: "tenant_id", type: "bool" }] }],
    ["missing captures record", loadHandler, { ...loadSite, capturesType: undefined }],
    ["stray captures record", loadHandler, { ...staticSite, capturesType: keyDecl }],
    ["GET with body", loadHandler, { ...loadSite, input: { mode: "json" } }],
    ["POST without body", loadHandler, { ...saveSite, input: { mode: "none" } }],
    ["unknown input mode", loadHandler, { ...saveSite, input: { mode: "xml" } }],
    ["missing wire type", loadHandler, { ...saveSite, input: { ...saveSite.input, type: "" } }],
    ["bad byte limit", loadHandler, { ...saveSite, input: { ...saveSite.input, limit: 0 } }],
    ["missing schema", loadHandler, { ...saveSite, input: { ...saveSite.input, schema: null } }],
    ["bad rows limit", loadHandler, { ...formSite, input: { ...formSite.input, rowsLimit: 99 } }],
    ["missing returns", loadHandler, { ...loadSite, returns: "" }],
    ["bad body mode", loadHandler, { ...loadSite, body: "xml" }],
    ["body disagreement", loadHandler, { ...loadSite, body: "html" }],
    ["JSON at form entry", loadHandler, { ...loadSite, body: "json" }],
    ["missing cases", loadHandler, { ...loadSite, cases: [] }],
    ["bad case status", loadHandler, { ...loadSite, cases: [{ leaf: "x", status: 204 }] }],
    ["missing response schema", loadHandler, { ...loadSite, responseSchema: null }],
    ["non-function handler", "nope", loadSite],
  ];
  expect(bad.length).toBe(22);
  for (const [name, handler, site] of bad) {
    if (name === "JSON at form entry") {
      await expect(mounts.mountForm(loadHandler, outcomeRenderer, structuralRenderer, site)).rejects.toThrow(
        TypeError,
      );
      continue;
    }
    await expect(mounts.mount(handler, site)).rejects.toThrow(TypeError);
  }
  await expect(
    mounts.mountForm(loadHandler, outcomeRenderer, "nope", formSite),
  ).rejects.toThrow(TypeError);
  await expect(mounts.mountForm(loadHandler, outcomeRenderer, structuralRenderer, {
    ...formSite,
    rejected: "",
  })).rejects.toThrow(TypeError);
  // Template faults fail as the declared invalid_route, never a throw.
  const templates = [
    "api/tenants/:tenant_id",
    "/api//:tenant_id",
    "/api/tenants/:Tenant_id",
    "/api/tenants/:tenant_id.json",
    "/api/tenants/:other",
    "/api/tenants/{tenant_id}",
    "/__can/x",
    "/api/tenants/:tenant_id/invoices/:tenant_id",
  ];
  expect(templates.length).toBe(8);
  for (const path of templates) {
    const completed = await mounts.mount(loadHandler, { ...loadSite, path });
    expect(completed.kind).toBe("domain");
    expect(errorType(completed as never)).toBe(id("can.std.http@1::invalid_route"));
    expect(failedField(completed, "reason")).toBe("path");
  }
});

test("make refuses duplicate, ambiguous and cross-static collisions", async () => {
  const renamed = (
    action: string,
    path: string,
    rows: readonly { name: string; type: string }[],
  ) => ({ ...loadSite, action, path, captures: rows });
  const first = value(await mounts.mount(loadHandler, loadSite));
  const duped = value(
    await mounts.mount(
      loadHandler,
      renamed("up09::load_again", "/api/tenants/:a/invoices/:b", [
        { name: "a", type: "int" },
        { name: "b", type: "int" },
      ]),
    ),
  );
  const doubled = await router.make([first, duped]);
  expect(doubled.kind).toBe("domain");
  expect(errorType(doubled as never)).toBe(id("can.std.http@1::duplicate_route"));
  expect(failedField(doubled, "method")).toBe("GET");
  expect(failedField(doubled, "path")).toBe("/api/tenants/{}/invoices/{}");
  const vague = value(
    await mounts.mount(
      loadHandler,
      renamed("up09::load_vague", "/api/:u/invoices/:v/x", [
        { name: "u", type: "int" },
        { name: "v", type: "int" },
      ]),
    ),
  );
  const crossed = await router.make([first, vague]);
  expect(crossed.kind).toBe("domain");
  expect(errorType(crossed as never)).toBe(id("can.std.http@1::ambiguous_route"));
  expect(failedField(crossed, "first")).toBe("/api/tenants/{tenant_id}/invoices/{invoice_id}");
  expect(failedField(crossed, "second")).toBe("/api/{u}/invoices/{v}/x");
  // Same shapes on different methods never collide.
  const split = value(await router.make([first, value(await mounts.mount(saveHandler, saveSite))]));
  expect(split).toBeDefined();
  // A legacy exact route and an all-static mount on one method and decoded
  // path are the same route.
  const legacy = value(await router.get("/invoices/new", staticHandler));
  const clash = await router.make([legacy, value(await mounts.mount(staticHandler, staticSite))]);
  expect(clash.kind).toBe("domain");
  expect(errorType(clash as never)).toBe(id("can.std.http@1::duplicate_route"));
  expect(failedField(clash, "method")).toBe("GET");
  expect(failedField(clash, "path")).toBe("/invoices/new");
});

test("exact legacy routes win over captured mounts within a method", async () => {
  calls.length = 0;
  const owned = await runOwnedRoot(async () => {
    const load = value(await mounts.mount(loadHandler, loadSite));
    const legacy = value(
      await router.get("/api/tenants/1/invoices/7", async () => {
        calls.push("legacy");
        return success(ownedResponse(200, "legacy", false));
      }),
    );
    const token = await serve(18612, [load, legacy]);
    const base = "http://127.0.0.1:18612";
    // The legacy static spelling serves even though the capture matches.
    const exact = await fetch(base + "/api/tenants/1/invoices/7");
    expect(exact.status).toBe(200);
    expect(await exact.text()).toBe("legacy");
    // Other spellings still reach the captured mount.
    const captured = await fetch(base + "/api/tenants/1/invoices/8");
    expect(captured.status).toBe(200);
    expect(await captured.text()).toBe('{"case":"grid_loaded","value":{"label":"t1i8"}}');
    expect(calls).toEqual(["legacy", "load:1,8"]);
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("guarded probe matrix keeps invalid input out of protected handlers", async () => {
  calls.length = 0;
  const owned = await runOwnedRoot(async () => {
    const load = value(await mounts.mount(loadHandler, loadSite));
    const save = value(await mounts.mount(saveHandler, saveSite));
    const slug = value(await mounts.mount(slugHandler, slugSite));
    const fresh = value(await mounts.mount(staticHandler, staticSite));
    const token = await serve(18613, [load, save, slug, fresh]);
    const base = "http://127.0.0.1:18613";
    const invoice = (tenant: string, invoice: string) =>
      `${base}/api/tenants/${tenant}/invoices/${invoice}`;
    const json = { "content-type": "application/json" };
    const edit = JSON.stringify({ operation_id: "op-1", label: "r1" });
    // Canonical int64 endpoints enter with typed bigints, including the
    // negative minimum, which the handler denies with its 403 leaf.
    for (const [tenant, line, status] of [
      ["1", "7", 200],
      ["0", "-1", 200],
      ["-9223372036854775808", "9223372036854775807", 403],
    ]) {
      const response = await fetch(invoice(tenant, line));
      expect(`${tenant}/${line}: ${response.status}`).toBe(`${tenant}/${line}: ${status}`);
      const text = await response.text();
      if (status === 200) expect(text).toContain(`"label":"t${tenant}i${line}"`);
      else expect(text).toContain('"reason":"foreign"');
    }
    // Out-of-range and non-canonical integers are malformed, including an
    // encoded digit, which must not decode into an int capture.
    for (const [tenant, line] of [
      ["9223372036854775808", "7"],
      ["1", "-9223372036854775809"],
      ["01", "7"],
      ["+1", "7"],
      ["-0", "7"],
      ["%31", "7"],
    ]) {
      const response = await fetch(invoice(tenant, line));
      expect(`${tenant}/${line}: ${response.status}`).toBe(`${tenant}/${line}: 400`);
      expect(await response.text()).toBe("Bad Request");
    }
    // Malformed escapes, encoded separators and backslashes never reach
    // a handler; only the 200 cases above entered callbacks. Encoded dots
    // travel below over path-as-is sockets: fetch would normalize them
    // client-side before Bun ever sees the wire bytes.
    for (const [tenant, line] of [["%ZZ", "7"], ["%ED%A0%80", "7"], ["%2F", "7"], ["%5C", "7"]]) {
      const response = await fetch(invoice(tenant, line), { method: "POST", headers: json, body: edit });
      expect(`${tenant}/${line}: ${response.status}`).toBe(`${tenant}/${line}: 400`);
    }
    expect(calls).toEqual(["load:1,7", "load:0,-1", "load:-9223372036854775808,9223372036854775807"]);
    // Method selection with static precedence inside one method.
    const created = await fetch(base + "/invoices/new", { method: "POST", headers: json, body: edit });
    expect(created.status).toBe(200);
    expect(calls.at(-1)).toBe("slug:new,op-1");
    await created.text();
    const freshHit = await fetch(base + "/invoices/new");
    expect(freshHit.status).toBe(200);
    await freshHit.text();
    const put = await fetch(base + "/invoices/new", { method: "PUT" });
    expect(put.status).toBe(405);
    expect(put.headers.get("allow")).toBe("GET, POST");
    expect(await put.text()).toBe("Method Not Allowed");
    const patch = await fetch(base + "/invoices/new", { method: "PATCH" });
    expect(patch.status).toBe(405);
    expect(patch.headers.get("allow")).toBe("GET, POST");
    expect(await patch.text()).toBe("Method Not Allowed");
    const mismatch = await fetch(invoice("1", "7"), { method: "PATCH" });
    expect(mismatch.status).toBe(405);
    expect(mismatch.headers.get("allow")).toBe("GET, POST");
    expect(await mismatch.text()).toBe("Method Not Allowed");
    // Strictly encoded string captures serve; separators and dots do not.
    const encoded = await fetch(base + "/invoices/A%20B%25%E2%82%AC", {
      method: "POST",
      headers: json,
      body: edit,
    });
    expect(encoded.status).toBe(200);
    expect(calls.at(-1)).toBe("slug:A B%€,op-1");
    await encoded.text();
    const double = await fetch(base + "/invoices/a%252Fb", {
      method: "POST",
      headers: json,
      body: edit,
    });
    expect(double.status).toBe(200);
    expect(calls.at(-1)).toBe("slug:a%2Fb,op-1");
    await double.text();
    const separator = await fetch(base + "/invoices/%2F", {
      method: "POST",
      headers: json,
      body: edit,
    });
    expect(separator.status).toBe(400);
    expect(await separator.text()).toBe("Bad Request");
    // Shape misses stay 404 without handler entry.
    for (const target of [
      "/api/tenants/1/invoices/",
      "/api/tenants/1/invoices/7/extra",
      "/api/tenants/1/invoices",
      "/nope",
    ]) {
      const response = await fetch(base + target);
      expect(`${target}: ${response.status}`).toBe(`${target}: 404`);
      expect(await response.text()).toBe("Not Found");
    }
    // Raw dot and separator targets over path-as-is sockets.
    const dotted = await rawRequest(18613, "/api/tenants/./1/invoices/7");
    expect(dotted.status).toBe(200);
    const escaped = await rawRequest(18613, "/%2e%2e/api/tenants/1/invoices/7");
    expect(escaped.status).toBe(200);
    const dotValue = await rawRequest(18613, "/invoices/%2e");
    expect(dotValue.status).toBe(404);
    const dotEscape = await rawRequest(18613, "/invoices/%2e%2e/x");
    expect(dotEscape.status).toBe(404);
    const rawSeparator = await rawRequest(18613, "/invoices/a%2Fb", "POST", edit, json);
    expect(rawSeparator.status).toBe(400);
    const rawMalformed = await rawRequest(18613, "/invoices/%ZZ");
    expect(rawMalformed.status).toBe(400);
    const rawBackslash = await rawRequest(18613, "/api/tenants/a%5Cb/invoices/7");
    expect(rawBackslash.status).toBe(400);
    // Encoded dots resolve before user code runs: a resolved shape that
    // misses every route is a 404, and no dot segment enters as data.
    const encodedDot = await rawRequest(18613, "/api/tenants/%2E/invoices/7");
    expect(encodedDot.status).toBe(404);
    const encodedDotDot = await rawRequest(18613, "/api/tenants/%2E%2E/invoices/7");
    expect(encodedDotDot.status).toBe(404);
    expect(calls).toEqual([
      "load:1,7",
      "load:0,-1",
      "load:-9223372036854775808,9223372036854775807",
      "slug:new,op-1",
      "static",
      "slug:A B%€,op-1",
      "slug:a%2Fb,op-1",
      "load:1,7",
      "load:1,7",
    ]);
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("mounts enforce media, declared budgets and JSON shape", async () => {
  calls.length = 0;
  const owned = await runOwnedRoot(async () => {
    const save = value(await mounts.mount(saveHandler, saveSite));
    const load = value(await mounts.mount(loadHandler, loadSite));
    const token = await serve(18614, [save, load]);
    const base = "http://127.0.0.1:18614";
    const target = base + "/api/tenants/1/invoices/7";
    const edit = JSON.stringify({ operation_id: "op-1", label: "r1" });
    // Wrong media is 415 without handler entry, even with a valid body.
    for (const headers of [
      {},
      { "content-type": "text/plain" },
      { "content-type": "application/json; charset=latin-1" },
      { "content-type": "not-a-media-type" },
    ]) {
      const response = await fetch(target, { method: "POST", headers, body: edit });
      expect(response.status).toBe(415);
      expect(await response.text()).toBe("Unsupported Media Type");
    }
    const json = { "content-type": "application/json" };
    // The declared 256-byte budget 413s; the boundary byte still serves.
    const prefix = '{"operation_id":"op-1","label":"';
    const suffix = '"}';
    const exact = await fetch(target, {
      method: "POST",
      headers: json,
      body: prefix + "p".repeat(256 - prefix.length - suffix.length) + suffix,
    });
    expect(exact.status).toBe(200);
    await exact.text();
    const over = await fetch(target, {
      method: "POST",
      headers: json,
      body: prefix + "p".repeat(257 - prefix.length - suffix.length) + suffix,
    });
    expect(over.status).toBe(413);
    expect(await over.text()).toBe("Payload Too Large");
    // Empty, malformed and mistyped JSON is a fixed 400.
    const empty = await fetch(target, { method: "POST", headers: json, body: "" });
    expect(empty.status).toBe(400);
    expect(await empty.text()).toBe("Bad Request");
    for (const body of [
      "{oops",
      JSON.stringify({ operation_id: "op-1", label: "r1", seats: 2 }),
      JSON.stringify({ operation_id: "op-1" }),
      JSON.stringify({ operation_id: 7, label: "r1" }),
    ]) {
      const response = await fetch(target, { method: "POST", headers: json, body });
      expect(`${body}: ${response.status}`).toBe(`${body}: 400`);
      expect(await response.text()).toBe("Bad Request");
    }
    // A GET body smuggled past native construction is a fixed 400.
    const smuggled = new Request(target, { method: "POST", body: "x" });
    Object.defineProperty(smuggled, "method", { value: "GET", configurable: true });
    const dirty = await snapshotRequest(smuggled, 65536);
    if (dirty.kind !== "request") throw new Error("snapshot rejected");
    const table = value(await router.make([load]));
    const rejected = value(await dispatch(table, dirty.value));
    expect(rejected.status).toBe(400);
    expect(await rejected.text()).toBe("Bad Request");
    expect(calls).toEqual(["save:1,7,op-1"]);
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("server body budget preempts mount entry", async () => {
  calls.length = 0;
  const owned = await runOwnedRoot(async () => {
    const save = value(await mounts.mount(saveHandler, saveSite));
    const token = await serve(18615, [save], 16n);
    const response = await fetch("http://127.0.0.1:18615/api/tenants/1/invoices/7", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ operation_id: "op-1", label: "r1" }),
    });
    expect(response.status).toBe(413);
    expect(await response.text()).toBe("Payload Too Large");
    expect(calls).toEqual([]);
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("form mounts render wire, structural and row-limit outcomes", async () => {
  calls.length = 0;
  const owned = await runOwnedRoot(async () => {
    const form = value(await mounts.mountForm(formHandler, outcomeRenderer, structuralRenderer, formSite));
    const token = await serve(18616, [form]);
    const base = "http://127.0.0.1:18616";
    const target = base + "/tenants/1/invoices/7";
    const encoded = { "content-type": "application/x-www-form-urlencoded" };
    const wire = "customer=ann&lines_order=r1&lines%5Br1%5D%5Bsku%5D=s1";
    const saved = await fetch(target, { method: "POST", headers: encoded, body: wire });
    expect(saved.status).toBe(200);
    expect(saved.headers.get("content-type")).toBe("text/html; charset=utf-8");
    expect(await saved.text()).toContain("ok:up09::saved");
    const stale = await fetch(target, {
      method: "POST",
      headers: encoded,
      body: wire.replace("customer=ann", "customer=stale"),
    });
    expect(stale.status).toBe(409);
    expect(await stale.text()).toContain("ok:up09::failed");
    // A repeated order row is structural: the handler never runs and the
    // 422 renderer redisplay the raw pairs in document order.
    const repeated = await fetch(target, {
      method: "POST",
      headers: encoded,
      body: wire + "&lines_order=r1",
    });
    expect(repeated.status).toBe(422);
    expect(await repeated.text()).toContain("bad:ann|r1|s1|r1");
    // The declared rows_limit of two turns a third row structural too.
    const third =
      "customer=ann&lines_order=r1&lines_order=r2&lines_order=r3" +
      "&lines%5Br1%5D%5Bsku%5D=s1&lines%5Br2%5D%5Bsku%5D=s2&lines%5Br3%5D%5Bsku%5D=s3";
    const limited = await fetch(target, { method: "POST", headers: encoded, body: third });
    expect(limited.status).toBe(422);
    expect(await limited.text()).toContain("bad:");
    // Wrong media is 415 and an over-budget body is 413, both pre-handler.
    const media = await fetch(target, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: wire,
    });
    expect(media.status).toBe(415);
    expect(await media.text()).toBe("Unsupported Media Type");
    const over = await fetch(target, {
      method: "POST",
      headers: encoded,
      body: wire + "&pad=" + "p".repeat(2048),
    });
    expect(over.status).toBe(413);
    expect(await over.text()).toBe("Payload Too Large");
    const malformed = await fetch(target, { method: "POST", headers: encoded, body: "customer=%" });
    expect(malformed.status).toBe(400);
    expect(await malformed.text()).toBe("Bad Request");
    expect(calls).toEqual([
      "form:1,ann",
      "outcome:up09::saved",
      "form:1,stale",
      "outcome:up09::failed",
      "structural:ann|r1|s1|r1",
      "structural:ann|r1|r2|r3|s1|s2|s3",
    ]);
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("handler and renderer faults become 500 without serving domain cases", async () => {
  calls.length = 0;
  const owned = await runOwnedRoot(async () => {
    const fault = failure(
      domain.create(
        id("can.std.http@1::invalid_route"),
        record(id("can.std.http@1::invalid_route"), [["reason", "boom"]]),
        origin,
      ),
    );
    const exploding = async () => {
      calls.push("exploding");
      throw new Error("handler fault");
    };
    const missing = async () => {
      calls.push("missing");
      return success(record("up09::archived", []));
    };
    const unencodable = async () => {
      calls.push("unencodable");
      return success(record("up09::grid_saved", [["operation_id", 7n]]));
    };
    const failingOutcome = async () => {
      calls.push("failingOutcome");
      return fault;
    };
    const failingStructural = async () => {
      calls.push("failingStructural");
      return fault;
    };
    const flatFormHandler = async (request: unknown, form: unknown) => {
      calls.push(`flatForm:${dataProperty(form, "customer") as string}`);
      return success(record("up09::saved", []));
    };
    const flat = <T extends object>(site: T, action: string, path: string) => ({
      ...site,
      action,
      path,
      capturesType: undefined,
      captures: [],
    });
    const crash = value(await mounts.mount(exploding, flat(saveSite, "up09::crash", "/fault/crash")));
    const stray = value(await mounts.mount(missing, flat(saveSite, "up09::stray", "/fault/stray")));
    const broken = value(
      await mounts.mount(unencodable, flat(saveSite, "up09::broken", "/fault/broken")),
    );
    const badOutcome = value(
      await mounts.mountForm(
        flatFormHandler,
        failingOutcome,
        structuralRenderer,
        flat(formSite, "up09::bad_outcome", "/fault/outcome"),
      ),
    );
    const badStructural = value(
      await mounts.mountForm(
        flatFormHandler,
        outcomeRenderer,
        failingStructural,
        flat(formSite, "up09::bad_structural", "/fault/structural"),
      ),
    );
    const token = await serve(18617, [crash, stray, broken, badOutcome, badStructural]);
    const base = "http://127.0.0.1:18617";
    const json = { "content-type": "application/json" };
    const edit = JSON.stringify({ operation_id: "op-1", label: "r1" });
    for (const target of ["/fault/crash", "/fault/stray", "/fault/broken"]) {
      const response = await fetch(base + target, { method: "POST", headers: json, body: edit });
      expect(`${target}: ${response.status}`).toBe(`${target}: 500`);
      expect(await response.text()).toBe("Internal Server Error");
    }
    const encoded = { "content-type": "application/x-www-form-urlencoded" };
    const wire = "customer=ann&lines_order=r1&lines%5Br1%5D%5Bsku%5D=s1";
    const outcome = await fetch(base + "/fault/outcome", {
      method: "POST",
      headers: encoded,
      body: wire,
    });
    expect(outcome.status).toBe(500);
    expect(await outcome.text()).toBe("Internal Server Error");
    const structural = await fetch(base + "/fault/structural", {
      method: "POST",
      headers: encoded,
      body: wire + "&lines_order=r1",
    });
    expect(structural.status).toBe(500);
    expect(await structural.text()).toBe("Internal Server Error");
    // Every fault entered its adapter exactly once; the structural fault
    // never reached the form handler.
    expect(calls).toEqual([
      "exploding",
      "missing",
      "unencodable",
      "flatForm:ann",
      "failingOutcome",
      "failingStructural",
    ]);
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("url fails unbuildable captures as invalid_path", async () => {
  const site = {
    action: "up09::load_grid",
    path: "/api/tenants/:tenant_id/invoices/:invoice_id",
    captures,
  };
  const key = (tenant: unknown, invoice: unknown) =>
    record(keyIdentity, [
      ["tenant_id", tenant],
      ["invoice_id", invoice],
    ]);
  // The int-typed captures record exercises int failures; the str
  // capture failures ride the slug builder below.
  const bad: [string, unknown][] = [
    ["int as str", key("1", 2n)],
    ["str as int", key(1n, "x")],
    ["out of range", key(2n ** 63n, 1n)],
  ];
  expect(bad.length).toBe(3);
  for (const [name, keyValue] of bad) {
    const completed = await mounts.url(keyValue, site, undefined);
    expect(`${name}: ${completed.kind}`).toBe(`${name}: domain`);
    expect(errorType(completed as never)).toBe(id("can.std.action@1::invalid_path"));
    expect(`${name}: ${failedField(completed, "reason")}`).toBe(
      `${name}: ${name === "out of range" ? "capture-value" : "capture-type"}`,
    );
  }
  const slugSite = {
    action: "up09::save_slug",
    path: "/invoices/:slug",
    captures: [{ name: "slug", type: "str" }],
  };
  for (const [name, slug, reason] of [
    ["empty", "", "capture-value"],
    ["separator", "a/b", "capture-value"],
    ["dot", "..", "capture-value"],
    ["mistyped", 7n, "capture-type"],
  ] as const) {
    const completed = await mounts.url(record("up09::slug_key", [["slug", slug]]), slugSite, undefined);
    expect(`${name}: ${completed.kind}`).toBe(`${name}: domain`);
    expect(failedField(completed, "reason")).toBe(reason);
  }
  // Captures-record arity disagreements and malformed sites throw.
  await expect(mounts.url(site, undefined)).rejects.toThrow(TypeError);
  await expect(mounts.url(key(1n, 2n), site)).rejects.toThrow(TypeError);
  await expect(mounts.url(key(1n, 2n), { ...site, path: "" }, undefined)).rejects.toThrow(TypeError);
  await expect(
    mounts.url(key(1n, 2n), { ...site, path: "/api/:Tenant" }, undefined),
  ).rejects.toThrow(TypeError);
});

test("combined dispatch unions 405 claims across both tables", async () => {
  const legacy = value(await router.post("/invoices/new", async () => success(ownedResponse(200, "x", false))));
  const fresh = value(await mounts.mount(staticHandler, staticSite));
  const table = value(await router.make([legacy, fresh]));
  const put = await snapshotRequest(new Request("http://test.local/invoices/new", { method: "PUT" }), 65536);
  if (put.kind !== "request") throw new Error("snapshot rejected");
  const denied = value(await dispatch(table, put.value));
  expect(denied.status).toBe(405);
  expect(denied.headers.get("allow")).toBe("GET, POST");
  expect(await denied.text()).toBe("Method Not Allowed");
  // Either table alone still classifies: the slug mount claims POST under
  // a capture, and unknown paths stay 404.
  const slug = value(await mounts.mount(slugHandler, slugSite));
  const solo = value(await router.make([slug]));
  const other = await snapshotRequest(
    new Request("http://test.local/invoices/abc", { method: "PUT" }),
    65536,
  );
  if (other.kind !== "request") throw new Error("snapshot rejected");
  const foreign = value(await dispatch(solo, other.value));
  expect(foreign.status).toBe(405);
  expect(foreign.headers.get("allow")).toBe("POST");
  await foreign.text();
  const nowhere = await snapshotRequest(new Request("http://test.local/nope", { method: "PUT" }), 65536);
  if (nowhere.kind !== "request") throw new Error("snapshot rejected");
  const missing = value(await dispatch(solo, nowhere.value));
  expect(missing.status).toBe(404);
  expect(await missing.text()).toBe("Not Found");
});
