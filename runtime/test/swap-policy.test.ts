import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, type FailureShape } from "../domain.ts";
import { createHTML, renderSafe } from "../platform/html.ts";
import { value } from "../completion.ts";
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
  [
    "html::invalid_structure",
    "html::invalid_url",
    "htmx::invalid_target",
    "htmx::invalid_interval",
  ].includes(e.name),
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
const contracts = {
  structure: errors[0]!.identity,
  url: errors[1]!.identity,
  target: errors[2]!.identity,
  interval: errors[3]!.identity,
};
const invoiceTable = [
  {
    method: "POST",
    segments: ["tenants", "{}", "invoices", "{}"],
    cases: [
      { status: 200, swap: "inner" },
      { status: 422, swap: "inner" },
      { status: 409, swap: "inner" },
      { status: 403, swap: "inner" },
      { status: 503, swap: "inner" },
    ],
  },
];
const html = createHTML(domain, contracts, [], invoiceTable);
const tag = async (name: string) => value(await html.makeTag(name));
const text = async (s: string) => value(await html.text(s));
const url = async (s: string) => value(await html.parseURL(s));
const render = async (nodes: unknown[]) => renderSafe(value(await html.fragment(nodes)));
const inner = "{&quot;swap&quot;:&quot;innerHTML&quot;}";

test("hx-post to a checked HTML action gains sorted hx-status exceptions", async () => {
  const form = value(
    await html.element(
      await tag("form"),
      [value(await html.post(await url("/tenants/1/invoices/7")))],
      [await text("body")],
    ),
  );
  expect(await render([form])).toBe(
    `<form hx-post="/tenants/1/invoices/7" hx-status:403="${inner}" hx-status:409="${inner}" hx-status:422="${inner}" hx-status:503="${inner}">body</form>`,
  );
});

test("plain URLs, wrong verbs, and empty tables serialize unchanged", async () => {
  const plain = value(
    await html.element(
      await tag("form"),
      [value(await html.post(await url("/validate")))],
      [await text("body")],
    ),
  );
  expect(await render([plain])).toBe(`<form hx-post="/validate">body</form>`);
  const getter = value(
    await html.element(
      await tag("button"),
      [value(await html.get(await url("/tenants/1/invoices/7")))],
      [await text("go")],
    ),
  );
  expect(await render([getter])).toBe(`<button hx-get="/tenants/1/invoices/7">go</button>`);
  const bare = createHTML(domain, contracts);
  const bareForm = value(
    await bare.element(
      value(await bare.makeTag("form")),
      [value(await bare.post(value(await bare.parseURL("/tenants/1/invoices/7"))))],
      [value(await bare.text("body"))],
    ),
  );
  expect(renderSafe(value(await bare.fragment([bareForm])))).toBe(
    `<form hx-post="/tenants/1/invoices/7">body</form>`,
  );
});

test("matching strips query strings, decodes captures, and rejects misshapen paths", async () => {
  const factory = createHTML(
    domain,
    contracts,
    [],
    [
      {
        method: "POST",
        segments: ["files", "{}"],
        cases: [{ status: 422, swap: "outer" }],
      },
    ],
  );
  const outer = "{&quot;swap&quot;:&quot;outerHTML&quot;}";
  const build = async (path: string) =>
    renderSafe(
      value(
        await factory.fragment([
          value(
            await factory.element(
              value(await factory.makeTag("form")),
              [value(await factory.post(value(await factory.parseURL(path))))],
              [value(await factory.text("b"))],
            ),
          ),
        ]),
      ),
    );
  expect(await build("/files/a%20b?x=1#h")).toBe(
    `<form hx-post="/files/a%20b?x=1#h" hx-status:422="${outer}">b</form>`,
  );
  expect(await build("/files/a/b")).toBe(`<form hx-post="/files/a/b">b</form>`);
  expect(await build("/files/")).toBe(`<form hx-post="/files/">b</form>`);
  expect(await build("/other/a%20b")).toBe(`<form hx-post="/other/a%20b">b</form>`);
});

test("overlapping entries union with first style winning per status", async () => {
  const factory = createHTML(
    domain,
    contracts,
    [],
    [
      {
        method: "POST",
        segments: ["a", "{}"],
        cases: [
          { status: 422, swap: "inner" },
          { status: 409, swap: "outer" },
        ],
      },
      {
        method: "POST",
        segments: ["a", "b"],
        cases: [
          { status: 422, swap: "outer" },
          { status: 403, swap: "inner" },
        ],
      },
    ],
  );
  const node = value(
    await factory.element(
      value(await factory.makeTag("form")),
      [value(await factory.post(value(await factory.parseURL("/a/b"))))],
      [value(await factory.text("b"))],
    ),
  );
  expect(renderSafe(value(await factory.fragment([node])))).toBe(
    `<form hx-post="/a/b" hx-status:403="${inner}" hx-status:409="{&quot;swap&quot;:&quot;outerHTML&quot;}" hx-status:422="${inner}">b</form>`,
  );
});

test("malformed swap tables fail fast and author validation still applies", async () => {
  for (const table of [
    [{ method: "PUT", segments: ["a"], cases: [] }],
    [{ method: "POST", segments: [], cases: [] }],
    [{ method: "POST", segments: ["a/b"], cases: [] }],
    [{ method: "POST", segments: ["a"], cases: [{ status: 99, swap: "inner" }] }],
    [{ method: "POST", segments: ["a"], cases: [{ status: 422, swap: "replace" }] }],
    ["nope"],
  ]) {
    expect(() => createHTML(domain, contracts, [], table)).toThrow(TypeError);
  }
  const dup = await html.element(
    await tag("form"),
    [
      value(await html.post(await url("/tenants/1/invoices/7"))),
      value(await html.post(await url("/tenants/1/invoices/7"))),
    ],
    [await text("b")],
  );
  expect(dup.kind).toBe("domain");
});
