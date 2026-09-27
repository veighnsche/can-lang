import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { mkdtempSync, writeFileSync, mkdirSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, type FailureShape } from "../domain.ts";
import { success, value, type Completion } from "../completion.ts";
import { record, dataProperty } from "../data.ts";
import { runOwnedRoot } from "../owner.ts";
import { createServer } from "../platform/server.ts";
import { createRouter } from "../platform/router.ts";
import { createResponses } from "../platform/http.ts";
import { createHTML } from "../platform/html.ts";
import type { Schema } from "../codec/json.ts";
import type { FormSchema } from "../platform/form.ts";
import {
  compileActionRoutes,
  matchActionRoute,
  createActionRoutes,
  readActionRouteBinding,
  generationMismatchResponse,
  readGenerationMismatch,
  readGenerationSlot,
  type ActionRouteSource,
} from "../platform/action-routes.ts";
import { createAssets, type AssetTable, type ServedAsset } from "../platform/assets.ts";

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
  "http::invalid_request": ["reason"],
  "http::body_limit": ["limit"],
  "codec::invalid_data": ["path", "reason"],
  "stream::read_failed": ["reason"],
  "stream::write_failed": ["reason"],
  "stream::cancelled": ["reason"],
  "stream::close_failed": ["reason"],
  "action::invalid_path": ["reason"],
  "html::invalid_structure": ["reason"],
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
const router = createRouter(domain, {
  invalid: id("can.std.http@1::invalid_route"),
  duplicate: id("can.std.http@1::duplicate_route"),
  ambiguous: id("can.std.http@1::ambiguous_route"),
});
const mounts = createActionRoutes(domain, {
  invalidRoute: id("can.std.http@1::invalid_route"),
  invalidPath: id("can.std.action@1::invalid_path"),
});
const responses = createResponses(domain, {
  invalid: id("can.std.http@1::invalid_request"),
  invalidData: id("can.std.codec@1::invalid_data"),
  close: id("can.std.stream@1::close_failed"),
  writeFailed: id("can.std.stream@1::write_failed"),
  limit: id("can.std.http@1::body_limit"),
});
const html = createHTML(domain, {
  structure: id("can.std.html@1::invalid_structure"),
  url: "unused",
  target: "unused",
  interval: "unused",
});
const fragment = async (text: string) => value(await html.fragment([value(await html.text(text))]));

const PIN = "a".repeat(64);
const OTHER = "b".repeat(64);
const pairedScript = `/__can/assets/${"c".repeat(64)}.js`;

function pinnedServer(pinned: string | undefined, script: string | undefined) {
  return createServer(
    domain,
    {
      invalidConfig: id("can.std.http@1::invalid_server_config"),
      bindFailed: id("can.std.http@1::bind_failed"),
      shutdownFailed: id("can.std.http@1::shutdown_failed"),
    },
    {
      serve: async () => undefined,
      ...(script === undefined ? {} : { browserScript: script }),
      ...(pinned === undefined ? {} : { generation: Promise.resolve(pinned) }),
    },
  );
}

async function documentResult(): Promise<Completion<unknown>> {
  const text = value(await html.text("hi"));
  const safe = value(await html.document("Paired", [], [text]));
  return responses.html(value(await responses.ok()), value(await responses.emptyHeaders()), safe);
}

const strNode = { identity: "c06::str", kind: "primitive", name: "str" };
const keyDecl = "c06::invoice_key";
const loadResponse: Schema = {
  root: "c06::grid_load_outcome",
  nodes: [
    {
      identity: "c06::grid_load_outcome",
      kind: "variant",
      name: "grid_load_outcome",
      leaves: ["c06::grid_loaded", "c06::grid_denied"],
    },
    {
      identity: "c06::grid_loaded",
      kind: "record",
      name: "grid_loaded",
      fields: [{ name: "label", type: "c06::str" }],
    },
    {
      identity: "c06::grid_denied",
      kind: "record",
      name: "grid_denied",
      fields: [{ name: "reason", type: "c06::str" }],
    },
    strNode,
  ],
};
const captures = [
  { name: "tenant_id", type: "int" },
  { name: "invoice_id", type: "int" },
];
const loadSite = {
  action: "c06::load_grid",
  method: "GET",
  path: "/api/tenants/:tenant_id/invoices/:invoice_id",
  capturesType: keyDecl,
  captures,
  input: { mode: "none" },
  returns: "c06::grid_load_outcome",
  body: "json",
  cases: [
    { leaf: "c06::grid_loaded", status: 200 },
    { leaf: "c06::grid_denied", status: 403 },
  ],
  responseSchema: loadResponse,
};
const formSchema: FormSchema = {
  root: "c06::invoice_form",
  fields: [{ name: "customer", kind: "str" }],
};
const formSite = {
  action: "c06::save_html",
  method: "POST",
  path: "/tenants/:tenant_id/invoices/:invoice_id",
  capturesType: keyDecl,
  captures,
  input: { mode: "form", type: "c06::invoice_form", limit: 2048, schema: formSchema },
  returns: "c06::edit_outcome",
  body: "html",
  cases: [
    { leaf: "c06::saved", status: 200, swap: "inner" },
    { leaf: "c06::failed", status: 409, swap: "inner" },
  ],
  rejected: "c06::rejected",
  rawEntry: "c06::raw_entry",
  issue: "c06::issue",
};
const pageSite = {
  action: "c06::read_page",
  method: "GET",
  path: "/tenants/:tenant_id/invoices/:invoice_id/page",
  capturesType: keyDecl,
  captures,
  input: { mode: "none" },
  returns: "c06::page_outcome",
  body: "html",
  cases: [
    { leaf: "c06::page_loaded", status: 200 },
    { leaf: "c06::page_denied", status: 403 },
  ],
};

const calls: string[] = [];
const loadHandler = async (request: unknown, key: unknown) => {
  const tenant = dataProperty(key, "tenant_id") as bigint;
  const invoice = dataProperty(key, "invoice_id") as bigint;
  calls.push(`load:${tenant},${invoice}`);
  expect(typeof request).toBe("object");
  return success(record("c06::grid_loaded", [["label", `t${tenant}i${invoice}`]]));
};
const formHandler = async (request: unknown, key: unknown, form: unknown) => {
  const tenant = dataProperty(key, "tenant_id") as bigint;
  const customer = dataProperty(form, "customer") as string;
  calls.push(`form:${tenant},${customer}`);
  return success(record("c06::saved", []));
};
const outcomeRenderer = async (outcome: unknown) => {
  expect(typeof outcome).toBe("object");
  return success(await fragment("ok:c06::saved"));
};
const structuralRenderer = async (bad: unknown) => {
  expect(typeof bad).toBe("object");
  return success(await fragment("bad:c06::rejected"));
};
const pageHandler = async (request: unknown, key: unknown) => {
  const tenant = dataProperty(key, "tenant_id") as bigint;
  calls.push(`page:${tenant}`);
  return success(record("c06::page_loaded", []));
};
const pageRenderer = async (outcome: unknown) => {
  expect(typeof outcome).toBe("object");
  return success(await fragment("page:c06::loaded"));
};

test("pinned servers splice the generation slot into paired documents", async () => {
  const paired = pinnedServer(PIN, pairedScript);
  const owned = await runOwnedRoot(async () => {
    const config = value(await paired.makeConfig("127.0.0.1", 18761n, 1048576n, 5000n));
    const route = value(await router.get("/x", async () => documentResult()));
    const table = value(await router.make([route]));
    const token = value(await paired.start(config, table));
    const body = await (await fetch("http://127.0.0.1:18761/x")).text();
    const tag = `<script type="module" src="${pairedScript}" data-can-generation="${PIN}"></script>`;
    expect(body).toContain(`${tag}</body>`);
    expect(body.split("<script").length - 1).toBe(1);
    expect((await paired.stop(token)).kind).toBe("ok");
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("missing, malformed, and unequal headers refuse before action logic", async () => {
  calls.length = 0;
  const paired = pinnedServer(PIN, pairedScript);
  const owned = await runOwnedRoot(async () => {
    const load = value(await mounts.mount(loadHandler, loadSite));
    const config = value(await paired.makeConfig("127.0.0.1", 18762n, 1048576n, 5000n));
    const table = value(await router.make([load]));
    const token = value(await paired.start(config, table));
    const target = "http://127.0.0.1:18762/api/tenants/1/invoices/7";
    const exact = `{"schemaVersion":1,"kind":"can.generation-mismatch","serverGeneration":"${PIN}"}`;
    const refusals: Record<string, string>[] = [
      {},
      { "can-generation": "nope" },
      { "can-generation": "a".repeat(63) },
      { "can-generation": OTHER },
    ];
    for (const headers of refusals) {
      const refused = await fetch(target, { headers });
      expect(refused.status).toBe(409);
      expect(refused.headers.get("content-type")).toBe("application/json");
      expect(await refused.text()).toBe(exact);
    }
    expect(calls).toEqual([]);
    expect((await paired.stop(token)).kind).toBe("ok");
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("matching headers reach the handler on pinned servers", async () => {
  calls.length = 0;
  const paired = pinnedServer(PIN, pairedScript);
  const owned = await runOwnedRoot(async () => {
    const load = value(await mounts.mount(loadHandler, loadSite));
    const config = value(await paired.makeConfig("127.0.0.1", 18763n, 1048576n, 5000n));
    const table = value(await router.make([load]));
    const token = value(await paired.start(config, table));
    const target = "http://127.0.0.1:18763/api/tenants/1/invoices/7";
    const ok = await fetch(target, { headers: { "can-generation": PIN } });
    expect(ok.status).toBe(200);
    expect(await ok.text()).toContain("grid_loaded");
    expect(calls).toEqual(["load:1,7"]);
    expect((await paired.stop(token)).kind).toBe("ok");
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("form mounts enforce while document mounts serve headerless navigations", async () => {
  calls.length = 0;
  const paired = pinnedServer(PIN, pairedScript);
  const owned = await runOwnedRoot(async () => {
    const form = value(
      await mounts.mountForm(formHandler, outcomeRenderer, structuralRenderer, formSite),
    );
    const page = value(await mounts.mountDocument(pageHandler, pageRenderer, pageSite));
    const config = value(await paired.makeConfig("127.0.0.1", 18764n, 1048576n, 5000n));
    const table = value(await router.make([form, page]));
    const token = value(await paired.start(config, table));
    const base = "http://127.0.0.1:18764";
    const encoded = { "content-type": "application/x-www-form-urlencoded" };
    const refused = await fetch(base + "/tenants/1/invoices/7", {
      method: "POST",
      headers: encoded,
      body: "customer=ann",
    });
    expect(refused.status).toBe(409);
    expect(await refused.text()).toContain("can.generation-mismatch");
    const admitted = await fetch(base + "/tenants/1/invoices/7", {
      method: "POST",
      headers: { ...encoded, "can-generation": PIN },
      body: "customer=ann",
    });
    expect(admitted.status).toBe(200);
    expect(await admitted.text()).toContain("ok:c06::saved");
    const navigation = await fetch(base + "/tenants/1/invoices/7/page");
    expect(navigation.status).toBe(200);
    expect(await navigation.text()).toContain("page:c06::loaded");
    expect(calls).toEqual(["form:1,ann", "page:1"]);
    expect((await paired.stop(token)).kind).toBe("ok");
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("unpinned servers serve headerless actions and splice slotless tags", async () => {
  calls.length = 0;
  const paired = pinnedServer(undefined, pairedScript);
  const owned = await runOwnedRoot(async () => {
    const load = value(await mounts.mount(loadHandler, loadSite));
    const doc = value(await router.get("/x", async () => documentResult()));
    const config = value(await paired.makeConfig("127.0.0.1", 18765n, 1048576n, 5000n));
    const table = value(await router.make([load, doc]));
    const token = value(await paired.start(config, table));
    const base = "http://127.0.0.1:18765";
    const ok = await fetch(base + "/api/tenants/1/invoices/7");
    expect(ok.status).toBe(200);
    expect(calls).toEqual(["load:1,7"]);
    const body = await (await fetch(base + "/x")).text();
    expect(body).toContain(`<script type="module" src="${pairedScript}"></script></body>`);
    expect(body).not.toContain("data-can-generation");
    expect((await paired.stop(token)).kind).toBe("ok");
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("plain routes and asset bytes skip the handshake on pinned servers", async () => {
  const assetBytes = new TextEncoder().encode("asset");
  const assets = {
    serve: async (native: Request): Promise<Response | undefined> => {
      if (new URL(native.url).pathname !== "/__can/assets/x.js") return undefined;
      return new Response(assetBytes, { status: 200 });
    },
    browserScript: pairedScript,
    generation: Promise.resolve(PIN),
  };
  const paired = createServer(
    domain,
    {
      invalidConfig: id("can.std.http@1::invalid_server_config"),
      bindFailed: id("can.std.http@1::bind_failed"),
      shutdownFailed: id("can.std.http@1::shutdown_failed"),
    },
    assets,
  );
  const owned = await runOwnedRoot(async () => {
    const plain = value(
      await router.get("/plain", async () =>
        responses.text(value(await responses.ok()), value(await responses.emptyHeaders()), "plain"),
      ),
    );
    const config = value(await paired.makeConfig("127.0.0.1", 18766n, 1048576n, 5000n));
    const table = value(await router.make([plain]));
    const token = value(await paired.start(config, table));
    const base = "http://127.0.0.1:18766";
    const route = await fetch(base + "/plain");
    expect(route.status).toBe(200);
    expect(await route.text()).toBe("plain");
    const asset = await fetch(base + "/__can/assets/x.js");
    expect(asset.status).toBe(200);
    expect(await asset.text()).toBe("asset");
    expect((await paired.stop(token)).kind).toBe("ok");
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("explicit action tables refuse before invocation", async () => {
  const invoked: string[] = [];
  const table = compileActionRoutes([
    {
      identity: "c06::echo",
      method: "GET",
      path: "/echo/{name}",
      captures: [{ name: "name", type: "str" }],
    },
  ]);
  const paired = createServer(
    domain,
    {
      invalidConfig: id("can.std.http@1::invalid_server_config"),
      bindFailed: id("can.std.http@1::bind_failed"),
      shutdownFailed: id("can.std.http@1::shutdown_failed"),
    },
    {
      serve: async () => undefined,
      browserScript: pairedScript,
      generation: Promise.resolve(PIN),
    },
    {
      table,
      invoke: async (identity) => {
        invoked.push(identity);
        return responses.text(
          value(await responses.ok()),
          value(await responses.emptyHeaders()),
          `echo:${identity}`,
        );
      },
    },
  );
  const owned = await runOwnedRoot(async () => {
    const config = value(await paired.makeConfig("127.0.0.1", 18767n, 1048576n, 5000n));
    const routes = value(await router.make([]));
    const token = value(await paired.start(config, routes));
    const base = "http://127.0.0.1:18767";
    const refused = await fetch(base + "/echo/a");
    expect(refused.status).toBe(409);
    expect(await refused.text()).toContain("can.generation-mismatch");
    expect(invoked).toEqual([]);
    const ok = await fetch(base + "/echo/a", { headers: { "can-generation": PIN } });
    expect(ok.status).toBe(200);
    expect(await ok.text()).toBe("echo:c06::echo");
    expect(invoked).toEqual(["c06::echo"]);
    expect((await paired.stop(token)).kind).toBe("ok");
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("route modes thread from mount to match with a JSON default", async () => {
  const owned = await runOwnedRoot(async () => {
    const load = value(await mounts.mount(loadHandler, loadSite));
    const form = value(
      await mounts.mountForm(formHandler, outcomeRenderer, structuralRenderer, formSite),
    );
    const page = value(await mounts.mountDocument(pageHandler, pageRenderer, pageSite));
    expect(readActionRouteBinding(load).entry.mode).toBe("json");
    expect(readActionRouteBinding(form).entry.mode).toBe("form");
    expect(readActionRouteBinding(page).entry.mode).toBe("document");
    const table = compileActionRoutes([
      readActionRouteBinding(load).entry,
      readActionRouteBinding(form).entry,
      readActionRouteBinding(page).entry,
    ]);
    const json = matchActionRoute(table, "GET", "/api/tenants/1/invoices/7");
    expect(json.kind).toBe("match");
    if (json.kind !== "match") throw new Error("json missed");
    expect(json.mode).toBe("json");
    const posted = matchActionRoute(table, "POST", "/tenants/1/invoices/7");
    expect(posted.kind).toBe("match");
    if (posted.kind !== "match") throw new Error("form missed");
    expect(posted.mode).toBe("form");
    const read = matchActionRoute(table, "GET", "/tenants/1/invoices/7/page");
    expect(read.kind).toBe("match");
    if (read.kind !== "match") throw new Error("document missed");
    expect(read.mode).toBe("document");
    const bare = compileActionRoutes([
      { identity: "c06::bare", method: "GET", path: "/bare", captures: [] },
    ]);
    const fallback = matchActionRoute(bare, "GET", "/bare");
    expect(fallback.kind).toBe("match");
    if (fallback.kind !== "match") throw new Error("bare missed");
    expect(fallback.mode).toBe("json");
    expect(() =>
      compileActionRoutes([
        {
          identity: "c06::bad",
          method: "GET",
          path: "/bad",
          captures: [],
          mode: "swap",
        } as unknown as ActionRouteSource,
      ]),
    ).toThrow("mode");
    return success(undefined);
  });
  expect(owned.completion.kind).toBe("ok");
});

test("mismatch answers stay byte-exact", async () => {
  const response = generationMismatchResponse(PIN);
  expect(response.status).toBe(409);
  expect(response.headers.get("content-type")).toBe("application/json");
  expect(await response.text()).toBe(
    `{"schemaVersion":1,"kind":"can.generation-mismatch","serverGeneration":"${PIN}"}`,
  );
});

test("mismatch shape admits only the exact response", () => {
  expect(
    readGenerationMismatch({
      schemaVersion: 1,
      kind: "can.generation-mismatch",
      serverGeneration: PIN,
    }),
  ).toBe(PIN);
  for (const shape of [
    null,
    [],
    "x",
    {},
    { schemaVersion: 1, kind: "can.generation-mismatch" },
    { schemaVersion: "1", kind: "can.generation-mismatch", serverGeneration: PIN },
    { schemaVersion: 1, kind: "can.generation-mismatch", serverGeneration: "short" },
    { schemaVersion: 1, kind: "grid_conflict", serverGeneration: PIN },
    {
      schemaVersion: 1,
      kind: "can.generation-mismatch",
      serverGeneration: PIN,
      extra: 1,
    },
  ]) {
    expect(readGenerationMismatch(shape)).toBeUndefined();
  }
});

test("slot reads resolve paired scripts and nothing else", () => {
  const script = (slot: string | null) => ({
    querySelector: (selectors: string) => {
      expect(selectors).toBe("script[data-can-generation]");
      return slot === null
        ? null
        : { getAttribute: (name: string) => (name === "data-can-generation" ? slot : null) };
    },
  });
  expect(readGenerationSlot(script(PIN))).toBe(PIN);
  expect(readGenerationSlot(script("nope"))).toBeUndefined();
  expect(readGenerationSlot(script(null))).toBeUndefined();
  expect(readGenerationSlot(undefined)).toBeUndefined();
  expect(readGenerationSlot(null)).toBeUndefined();
  expect(readGenerationSlot({})).toBeUndefined();
  expect(
    readGenerationSlot({
      querySelector: () => {
        throw new Error("boom");
      },
    }),
  ).toBeUndefined();
  expect(
    readGenerationSlot({
      querySelector: () => ({
        getAttribute: () => {
          throw new Error("boom");
        },
      }),
    }),
  ).toBeUndefined();
});

test("created assets pin paired manifests and ignore the rest", async () => {
  const script = await Bun.file(
    new URL("../../distribution/assets/htmx-4.0.0.min.js", import.meta.url),
  ).bytes();
  const guard = await Bun.file(
    new URL("../../distribution/assets/htmx-guard.js", import.meta.url),
  ).bytes();
  const htmxDigest = createHash("sha256").update(script).digest("hex");
  const guardDigest = createHash("sha256").update(guard).digest("hex");
  const entry = new TextEncoder().encode("entry");
  const entryDigest = createHash("sha256").update(entry).digest("hex");
  const entryRoute = `/__can/assets/${entryDigest}.js`;
  const tableBytes = new TextEncoder().encode("{}");
  const tableDigest = createHash("sha256").update(tableBytes).digest("hex");
  const tableRoute = `/__can/assets/${tableDigest}.json`;
  const entryFile = (
    route: string,
    content: Uint8Array,
    mediaType: string,
    name: string,
  ): ServedAsset => ({
    route,
    digest: createHash("sha256").update(content).digest("hex"),
    mediaType,
    file: `assets/${createHash("sha256").update(content).digest("hex")}/${name}`,
    integrity: "",
  });
  const setup = (manifest: string | undefined): { table: AssetTable; root: URL } => {
    const dist = mkdtempSync(join(tmpdir(), "can-generation-"));
    const generation = join(dist, "builds", PIN);
    mkdirSync(join(generation, "assets", htmxDigest), { recursive: true });
    mkdirSync(join(generation, "assets", guardDigest), { recursive: true });
    mkdirSync(join(generation, "assets", entryDigest), { recursive: true });
    mkdirSync(join(generation, "assets", tableDigest), { recursive: true });
    writeFileSync(join(generation, "assets", htmxDigest, "htmx-4.0.0.min.js"), script);
    writeFileSync(join(generation, "assets", guardDigest, "htmx-guard.js"), guard);
    writeFileSync(join(generation, "assets", entryDigest, "browser.js"), entry);
    writeFileSync(join(generation, "assets", tableDigest, "table.json"), tableBytes);
    if (manifest !== undefined) writeFileSync(join(generation, "manifest.json"), manifest);
    const table: AssetTable = {
      htmx: {
        route: "/__can/assets/htmx-4.0.0.min.js",
        digest: htmxDigest,
        mediaType: "text/javascript",
        file: `assets/${htmxDigest}/htmx-4.0.0.min.js`,
        integrity: "sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc",
      },
      guard: {
        route: "/__can/assets/htmx-guard.js",
        digest: guardDigest,
        mediaType: "text/javascript",
        file: `assets/${guardDigest}/htmx-guard.js`,
        integrity: "sha384-C5A6uJ5FDnfm4n0j9TRlnVP+OIz//4VRs7OcvEYgUzCRevIwXmktURVNyPxYvOpQ",
      },
      project: [],
      browser: {
        buildId: OTHER,
        entry: entryRoute,
        table: tableRoute,
        files: [
          entryFile(entryRoute, entry, "text/javascript", "browser.js"),
          entryFile(tableRoute, tableBytes, "application/json", "table.json"),
        ],
      },
    };
    return { table, root: pathToFileURL(generation + "/") };
  };
  const manifest = JSON.stringify({
    schemaVersion: 1,
    kind: "can-output-generation-v1",
    buildID: PIN,
  });
  const paired = setup(manifest);
  expect(await createAssets(paired.table, paired.root).generation).toBe(PIN);
  const unpaired = setup(manifest);
  expect(
    await createAssets({ ...unpaired.table, browser: undefined }, unpaired.root).generation,
  ).toBeUndefined();
  const missing = setup(undefined);
  expect(await createAssets(missing.table, missing.root).generation).toBeUndefined();
  const broken = setup(JSON.stringify({ buildID: "short" }));
  expect(await createAssets(broken.table, broken.root).generation).toBeUndefined();
  const garbage = setup("{oops");
  expect(await createAssets(garbage.table, garbage.root).generation).toBeUndefined();
});
