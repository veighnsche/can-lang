import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { mkdtempSync, writeFileSync, mkdirSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createHTML, isHTMLValue, renderSafe } from "../platform/html.ts";
import {
  createAssets,
  browserPolicy,
  assetRetained,
  type AssetTable,
  type ServedAsset,
} from "../platform/assets.ts";
import { createRouter } from "../platform/router.ts";
import { createResponses } from "../platform/http.ts";
import { value, type Completion } from "../completion.ts";
import { record, array } from "../data.ts";

const hash = (kind: string, name: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, name]))
    .digest("hex");
const shape = (
  kind: string,
  declaration: string,
  fields: { name: string; type: string }[] = [],
): FailureShape => ({
  identity: hash(kind, declaration),
  kind,
  declaration,
  fields,
  arguments: [],
  leaves: [],
  inputs: [],
  errors: [],
});
const str = shape("primitive", "str"),
  int = shape("primitive", "int");
const declarations = catalogue.errors.filter((e) =>
  ["html::invalid_structure", "html::invalid_url", "http::invalid_route"].includes(e.name),
);
const errors = declarations.map((e) =>
  shape(
    "error",
    e.identity,
    e.fields.map((f) => ({ name: f.name, type: f.type === "int" ? int.identity : str.identity })),
  ),
);
const domain = createDomainRuntime({
  declarations: declarations.map((e) => ({ ...e, parameters: 0 })),
  shapes: [str, int, ...errors],
});
const urlID = errors[1]!.identity;
const root = mkdtempSync(join(tmpdir(), "can-assets-"));
const script = await Bun.file(
  new URL("../../distribution/assets/htmx-4.0.0.min.js", import.meta.url),
).bytes();
const guard = await Bun.file(
  new URL("../../distribution/assets/htmx-guard.js", import.meta.url),
).bytes();
const css = new TextEncoder().encode("body{color:black}\n");
const digest = (bytes: Uint8Array) => createHash("sha256").update(bytes).digest("hex");
const htmxDigest = digest(script),
  guardDigest = digest(guard),
  cssDigest = digest(css);
const guardEntry: ServedAsset = {
  route: "/__can/assets/htmx-guard.js",
  digest: guardDigest,
  mediaType: "text/javascript",
  file: `assets/${guardDigest}/htmx-guard.js`,
  integrity: "sha384-mr/IRfJgLjok38ftBi21o/T8c9cnFZrvEKtiwVjIOFAlo3Z7h1rGMYsWvebDJ8kG",
};
mkdirSync(join(root, "assets", htmxDigest), { recursive: true });
mkdirSync(join(root, "assets", guardDigest), { recursive: true });
mkdirSync(join(root, "assets", cssDigest), { recursive: true });
writeFileSync(join(root, "assets", htmxDigest, "htmx-4.0.0.min.js"), script);
writeFileSync(join(root, "assets", guardDigest, "htmx-guard.js"), guard);
writeFileSync(join(root, "assets", cssDigest, "site.css"), css);
const cssRoute = `/__can/project/${cssDigest}/site.css`;
const table: AssetTable = {
  htmx: {
    route: "/__can/assets/htmx-4.0.0.min.js",
    digest: htmxDigest,
    mediaType: "text/javascript",
    file: `assets/${htmxDigest}/htmx-4.0.0.min.js`,
    integrity: "sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc",
  },
  guard: guardEntry,
  project: [
    {
      route: cssRoute,
      digest: cssDigest,
      mediaType: "text/css; charset=utf-8",
      file: `assets/${cssDigest}/site.css`,
      integrity: "",
    },
  ],
};
const assets = createAssets(table, pathToFileURL(root + "/"));
const ask = (path: string, headers?: Record<string, string>, method = "GET") =>
  assets.serve(new Request("http://127.0.0.1" + path, { method, headers }));

test("pinned htmx and project assets are immutable exact-path responses", async () => {
  const scriptResponse = await ask("/__can/assets/htmx-4.0.0.min.js");
  expect(scriptResponse?.status).toBe(200);
  expect(scriptResponse?.headers.get("content-type")).toBe("text/javascript");
  expect(scriptResponse?.headers.get("etag")).toBe(`"${htmxDigest}"`);
  expect(scriptResponse?.headers.get("cache-control")).toBe("public, max-age=31536000, immutable");
  expect(scriptResponse?.headers.get("x-content-type-options")).toBe("nosniff");
  expect(new Uint8Array(await scriptResponse!.arrayBuffer())).toEqual(script);
  const cached = await ask("/__can/assets/htmx-4.0.0.min.js", {
    "if-none-match": `"${htmxDigest}"`,
  });
  expect(cached?.status).toBe(304);
  expect(await cached!.text()).toBe("");
  const head = await ask("/__can/assets/htmx-4.0.0.min.js", undefined, "HEAD");
  expect(head?.status).toBe(200);
  expect(await head!.text()).toBe("");
  const style = await ask(cssRoute);
  expect(style?.status).toBe(200);
  expect(style?.headers.get("content-type")).toBe("text/css; charset=utf-8");
  expect(await style!.text()).toBe("body{color:black}\n");
  expect(await ask("/app")).toBeUndefined();
});

test("path escapes, unknown routes, wrong bytes and foreign methods are refused", async () => {
  for (const path of [
    `/__can/project/${cssDigest}/../${htmxDigest}/htmx-4.0.0.min.js`,
    `/__can/project/${htmxDigest}/htmx-4.0.0.min.js`,
    `/__can/assets/`,
    `/__can/assets/htmx-4.0.0.min.js.map`,
    `/__can`,
  ]) {
    expect((await ask(path))?.status).toBe(404);
  }
  expect((await ask("/__can/assets/htmx-4.0.0.min.js", undefined, "POST"))?.status).toBe(405);
  writeFileSync(join(root, "assets", cssDigest, "site.css"), "tampered\n");
  expect((await ask(cssRoute))?.status).toBe(404);
  writeFileSync(join(root, "assets", cssDigest, "site.css"), css);
});

test("pinned guard serves exact bytes while guard-ish routes and tables fail closed", async () => {
  const served = await ask("/__can/assets/htmx-guard.js");
  expect(served?.status).toBe(200);
  expect(served?.headers.get("content-type")).toBe("text/javascript");
  expect(served?.headers.get("etag")).toBe(`"${guardDigest}"`);
  expect(served?.headers.get("cache-control")).toBe("public, max-age=31536000, immutable");
  expect(served?.headers.get("x-content-type-options")).toBe("nosniff");
  expect(new Uint8Array(await served!.arrayBuffer())).toEqual(guard);
  const cached = await ask("/__can/assets/htmx-guard.js", {
    "if-none-match": `"${guardDigest}"`,
  });
  expect(cached?.status).toBe(304);
  for (const path of [
    "/__can/assets/htmx-guard.js.map",
    "/__can/assets/htmx-guard2.js",
    "/__can/assets/htmx-guard",
    `/__can/assets/${guardDigest}.js`,
  ]) {
    expect((await ask(path))?.status).toBe(404);
  }
  expect((await ask("/__can/assets/htmx-guard.js", undefined, "POST"))?.status).toBe(405);
  writeFileSync(join(root, "assets", guardDigest, "htmx-guard.js"), "tampered\n");
  expect((await ask("/__can/assets/htmx-guard.js"))?.status).toBe(404);
  writeFileSync(join(root, "assets", guardDigest, "htmx-guard.js"), guard);
  expect((await ask("/__can/assets/htmx-guard.js"))?.status).toBe(200);
  const base = pathToFileURL(root + "/");
  for (const broken of [
    { ...guardEntry, route: "/__can/assets/other-guard.js" },
    {
      ...guardEntry,
      integrity:
        "sha384-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
    },
    { ...guardEntry, integrity: "" },
    { ...guardEntry, mediaType: "application/json" },
    { ...guardEntry, file: `assets/${guardDigest}/renamed.js` },
    { ...guardEntry, digest: "c".repeat(64) },
  ]) {
    expect(() => createAssets({ ...table, guard: broken }, base)).toThrow(TypeError);
  }
});

test("declared asset urls mint html provenance and foreign names stay reasons", async () => {
  const declared = createHTML(
    domain,
    { structure: errors[0]!.identity, url: urlID, target: "unused", interval: "unused" },
    [cssRoute],
  );
  const url = value(await declared.declareAsset(cssRoute));
  expect(isHTMLValue("url", url)).toBe(true);
  const node = value(await declared.stylesheet(url));
  expect(renderSafe(value(await declared.document("Assets", [node], [])))).toContain(cssRoute);
  const missing = await declared.rejectAsset("missing");
  expect(missing.kind).toBe("domain");
  if (missing.kind === "domain")
    expect((domainFailureDiagnostics(missing.value).payload as { reason: string }).reason).toBe(
      "missing",
    );
  const unowned = await declared.rejectAsset("unowned");
  if (unowned.kind === "domain")
    expect((domainFailureDiagnostics(unowned.value).payload as { reason: string }).reason).toBe(
      "unowned",
    );
  await expect(declared.declareAsset("https://cdn.example/htmx.js")).rejects.toThrow();
  await expect(declared.rejectAsset("syntax")).rejects.toThrow();
});

test("reserved routes and htmx navigation headers are not application-controlled", async () => {
  const routing = createRouter(domain, {
    invalid: errors[2]!.identity,
    duplicate: "unused",
    ambiguous: "unused",
  });
  const route = await routing.get(
    "/__can/assets/htmx-4.0.0.min.js",
    async () => ({ kind: "ok", value: undefined }) as Completion<unknown>,
  );
  expect(route.kind).toBe("domain");
  const responses = createResponses(domain, {
    invalid: errors[0]!.identity,
    invalidData: "unused",
    close: "unused",
    writeFailed: "unused",
    limit: "unused",
  });
  const header = array([
    record("header", [
      ["name", "HX-Redirect"],
      ["value", "https://evil.test"],
    ]),
  ]);
  const rejected = await responses.makeHeaders(header);
  expect(rejected.kind).toBe("domain");
  expect(browserPolicy).toContain("script-src 'self'");
  expect(browserPolicy).not.toContain("unsafe-eval");
  expect(browserPolicy).toContain("navigate-to 'self'");
  expect(browserPolicy).toContain("form-action 'self'");
});

const retentionMs = 7 * 24 * 60 * 60 * 1000;

test("retention keeps routes through the bound and drops them after", () => {
  const now = 1790314240936;
  expect(assetRetained(now, now - retentionMs + 1)).toBe(true);
  expect(assetRetained(now, now - retentionMs)).toBe(true);
  expect(assetRetained(now, now - retentionMs - 1)).toBe(false);
});

function pairedTree() {
  const dist = mkdtempSync(join(tmpdir(), "can-paired-"));
  const generation = join(dist, "builds", "current");
  const entry = new TextEncoder().encode(
    'console.log("paired");\n//# sourceMappingURL=browser.js.map\n',
  );
  const map = new TextEncoder().encode('{"version":3,"file":"browser.js"}\n');
  const tableBytes = new TextEncoder().encode(
    '{"schemaVersion":1,"kind":"can.diagnostic-table"}\n',
  );
  const entryDigest = digest(entry),
    mapDigest = digest(map),
    tableDigest = digest(tableBytes);
  for (const [name, bytes] of [
    ["htmx-4.0.0.min.js", script],
    ["htmx-guard.js", guard],
    ["browser.js", entry],
    ["browser.js.map", map],
    ["table.json", tableBytes],
  ] as const) {
    const folder =
      name === "htmx-4.0.0.min.js"
        ? htmxDigest
        : name === "htmx-guard.js"
          ? guardDigest
          : digest(bytes);
    mkdirSync(join(generation, "assets", folder), { recursive: true });
    writeFileSync(join(generation, "assets", folder, name), bytes);
  }
  const entryRoute = `/__can/assets/${entryDigest}.js`;
  const mapRoute = `/__can/assets/${mapDigest}.js.map`;
  const tableRoute = `/__can/assets/${tableDigest}.json`;
  const file = (
    route: string,
    content: Uint8Array,
    mediaType: string,
    name: string,
  ): ServedAsset => ({
    route,
    digest: digest(content),
    mediaType,
    file: `assets/${digest(content)}/${name}`,
    integrity: "",
  });
  const paired: AssetTable = {
    htmx: {
      route: "/__can/assets/htmx-4.0.0.min.js",
      digest: htmxDigest,
      mediaType: "text/javascript",
      file: `assets/${htmxDigest}/htmx-4.0.0.min.js`,
      integrity: "sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc",
    },
    guard: guardEntry,
    project: [],
    browser: {
      buildId: "b".repeat(64),
      entry: entryRoute,
      table: tableRoute,
      files: [
        file(entryRoute, entry, "text/javascript", "browser.js"),
        file(mapRoute, map, "application/json", "browser.js.map"),
        file(tableRoute, tableBytes, "application/json", "table.json"),
      ],
    },
  };
  return { dist, generation, paired, entry, map, tableBytes, entryRoute, mapRoute, tableRoute };
}

test("paired browser bytes serve exact JS, map, and table with the entry selected", async () => {
  const { generation, paired, entry, map, tableBytes, entryRoute, mapRoute, tableRoute } =
    pairedTree();
  const server = createAssets(paired, pathToFileURL(generation + "/"));
  expect(server.browserScript).toBe(entryRoute);
  const served = await server.serve(new Request("http://127.0.0.1" + entryRoute));
  expect(served?.status).toBe(200);
  expect(served?.headers.get("content-type")).toBe("text/javascript");
  expect(served?.headers.get("etag")).toBe(`"${digest(entry)}"`);
  expect(served?.headers.get("cache-control")).toBe("public, max-age=31536000, immutable");
  expect(new Uint8Array(await served!.arrayBuffer())).toEqual(entry);
  const servedMap = await server.serve(new Request("http://127.0.0.1" + mapRoute));
  expect(servedMap?.status).toBe(200);
  expect(servedMap?.headers.get("content-type")).toBe("application/json");
  expect(new Uint8Array(await servedMap!.arrayBuffer())).toEqual(map);
  const servedTable = await server.serve(new Request("http://127.0.0.1" + tableRoute));
  expect(servedTable?.status).toBe(200);
  expect(new Uint8Array(await servedTable!.arrayBuffer())).toEqual(tableBytes);
});

test("paired browser tables reject mismatched identity, routes, and files", async () => {
  const { generation, paired, entryRoute } = pairedTree();
  const root = pathToFileURL(generation + "/");
  const browser = paired.browser!;
  expect(() => createAssets({ ...paired, browser: undefined }, root)).not.toThrow();
  expect(createAssets({ ...paired, browser: undefined }, root).browserScript).toBeUndefined();
  for (const broken of [
    { ...browser, buildId: "short" },
    { ...browser, files: [] },
    { ...browser, entry: browser.files[1]!.route },
    { ...browser, entry: "/__can/assets/" + "c".repeat(64) + ".js" },
    { ...browser, table: browser.entry },
    {
      ...browser,
      files: browser.files.map((file) =>
        file.route === entryRoute ? { ...file, digest: "c".repeat(64) } : file,
      ),
    },
    {
      ...browser,
      files: browser.files.map((file) =>
        file.route === entryRoute ? { ...file, mediaType: "application/json" } : file,
      ),
    },
    {
      ...browser,
      files: browser.files.map((file) =>
        file.route === entryRoute ? { ...file, file: "elsewhere.js" } : file,
      ),
    },
    {
      ...browser,
      files: browser.files.map((file) =>
        file.route === entryRoute ? { ...file, integrity: "sha384-x" } : file,
      ),
    },
    { ...browser, files: [...browser.files, browser.files[0]!] },
  ]) {
    expect(() => createAssets({ ...paired, browser: broken }, root)).toThrow(TypeError);
  }
  // Tampered paired bytes fail closed on serve.
  const tampered = join(
    generation,
    "assets",
    digest(
      new TextEncoder().encode('console.log("paired");\n//# sourceMappingURL=browser.js.map\n'),
    ),
    "browser.js",
  );
  writeFileSync(tampered, "tampered\n");
  const server = createAssets(paired, root);
  expect((await server.serve(new Request("http://127.0.0.1" + entryRoute)))?.status).toBe(404);
});

test("replaced digest URLs serve from the durable ledger until expiry", async () => {
  const { dist, generation, paired, entry, entryRoute } = pairedTree();
  const stale = new TextEncoder().encode(
    'console.log("stale");\n//# sourceMappingURL=browser.js.map\n',
  );
  const staleDigest = digest(stale);
  const staleRoute = `/__can/assets/${staleDigest}.js`;
  mkdirSync(join(dist, "assets"), { recursive: true });
  writeFileSync(join(dist, "assets", `${staleDigest}.js`), stale);
  const server = createAssets(paired, pathToFileURL(generation + "/"));
  const ask = (path: string, method = "GET") =>
    server.serve(new Request("http://127.0.0.1" + path, { method }));
  // No ledger yet: the unlisted route is unknown.
  expect((await ask(staleRoute))?.status).toBe(404);
  const ledger = (replacedAt: number) => ({
    schemaVersion: 1,
    kind: "can.asset-ledger",
    retained: [
      {
        digest: staleDigest,
        route: staleRoute,
        mediaType: "text/javascript",
        file: `assets/${staleDigest}.js`,
        replacedAt,
      },
    ],
  });
  // Before the bound (with margin; the exact bound is covered by the
  // retention predicate test since file IO would blur a 1ms margin).
  writeFileSync(
    join(dist, "assets.json"),
    JSON.stringify(ledger(Date.now() - retentionMs + 60000)),
  );
  const served = await ask(staleRoute);
  expect(served?.status).toBe(200);
  expect(served?.headers.get("content-type")).toBe("text/javascript");
  expect(served?.headers.get("etag")).toBe(`"${staleDigest}"`);
  expect(new Uint8Array(await served!.arrayBuffer())).toEqual(stale);
  expect((await ask(staleRoute, "POST"))?.status).toBe(405);
  // Tampered durable bytes fail closed, then serve again once restored.
  writeFileSync(join(dist, "assets", `${staleDigest}.js`), "tampered\n");
  expect((await ask(staleRoute))?.status).toBe(404);
  writeFileSync(join(dist, "assets", `${staleDigest}.js`), stale);
  expect((await ask(staleRoute))?.status).toBe(200);
  // A restarted server (a fresh asset layer over the same dist) still
  // serves the retained route from disk, not memory.
  const restarted = createAssets(paired, pathToFileURL(generation + "/"));
  expect((await restarted.serve(new Request("http://127.0.0.1" + staleRoute)))?.status).toBe(200);
  // Past the bound the route expires while the current entry still serves.
  writeFileSync(
    join(dist, "assets.json"),
    JSON.stringify(ledger(Date.now() - retentionMs - 60000)),
  );
  expect((await ask(staleRoute))?.status).toBe(404);
  expect((await ask(entryRoute))?.status).toBe(200);
  expect(new Uint8Array(await (await ask(entryRoute))!.arrayBuffer())).toEqual(entry);
});

test("durable ledger mismatches fail closed", async () => {
  const { dist, generation, paired } = pairedTree();
  const stale = new TextEncoder().encode('console.log("stale");\n');
  const staleDigest = digest(stale);
  const staleRoute = `/__can/assets/${staleDigest}.js`;
  mkdirSync(join(dist, "assets"), { recursive: true });
  writeFileSync(join(dist, "assets", `${staleDigest}.js`), stale);
  const server = createAssets(paired, pathToFileURL(generation + "/"));
  const ask = (path: string) => server.serve(new Request("http://127.0.0.1" + path));
  const good = {
    digest: staleDigest,
    route: staleRoute,
    mediaType: "text/javascript",
    file: `assets/${staleDigest}.js`,
    replacedAt: Date.now(),
  };
  for (const retained of [
    [{ ...good, mediaType: "text/css" }],
    [{ ...good, file: `assets/${staleDigest}.json` }],
    [{ ...good, route: `/__can/assets/${staleDigest}.json` }],
    [{ ...good, digest: "c".repeat(64) }],
    [{ ...good, replacedAt: Number.NaN }],
    ["not-an-entry"],
  ]) {
    writeFileSync(
      join(dist, "assets.json"),
      JSON.stringify({ schemaVersion: 1, kind: "can.asset-ledger", retained }),
    );
    expect((await ask(staleRoute))?.status).toBe(404);
  }
  for (const ledger of ['{"schemaVersion":1,"kind":"wrong","retained":[]}', "not json", "[]"]) {
    writeFileSync(join(dist, "assets.json"), ledger);
    expect((await ask(staleRoute))?.status).toBe(404);
  }
});
