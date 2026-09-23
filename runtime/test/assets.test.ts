import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { mkdtempSync, writeFileSync, mkdirSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createHTML, isHTMLValue, renderSafe } from "../platform/html.ts";
import { createAssets, browserPolicy, type AssetTable } from "../platform/assets.ts";
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
const declarations = catalogue.errors.filter((e) => [1220, 1221, 1230].includes(e.id));
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
const css = new TextEncoder().encode("body{color:black}\n");
const digest = (bytes: Uint8Array) => createHash("sha256").update(bytes).digest("hex");
const htmxDigest = digest(script),
  cssDigest = digest(css);
mkdirSync(join(root, "assets", htmxDigest), { recursive: true });
mkdirSync(join(root, "assets", cssDigest), { recursive: true });
writeFileSync(join(root, "assets", htmxDigest, "htmx-4.0.0.min.js"), script);
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
