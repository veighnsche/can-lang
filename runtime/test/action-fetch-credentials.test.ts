import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, type FailureShape } from "../domain.ts";
import { record } from "../data.ts";
import type { Schema } from "../codec/json.ts";
import { createJsonActionFetch, type JsonFetchSite } from "../platform/action-json.ts";

// T24 same-origin credentials. Protected actions authenticate through
// the session cookie, so the browser fetch consumer must send
// same-origin credentials on both GET loads and POST saves.
const hash = (key: unknown) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify(key))
    .digest("hex");
const shape = (
  kind: string,
  declaration: string,
  fields: { name: string; type: string }[] = [],
): FailureShape => ({
  identity: hash([kind, declaration]),
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
const header = shape("record", "can.std.http@1::header", [
  { name: "name", type: str.identity },
  { name: "value", type: str.identity },
]);
const headers: FailureShape = {
  ...shape("array", ""),
  identity: hash(["array", "", header.identity]),
  element: header.identity,
};
const errorNames = [
  "http::transport_failed",
  "http::invalid_request",
  "http::body_limit",
  "http::status_error",
  "codec::invalid_data",
];
const declarations = catalogue.errors.filter((e) => errorNames.includes(e.name));
const fieldKinds = new Map(
  catalogue.errors.flatMap((e) => e.fields.map((f) => [e.name + ":" + f.name, f.type])),
);
const errors = declarations.map((d) =>
  shape(
    "error",
    d.identity,
    d.fields.map((f) => ({
      name: f.name,
      type:
        fieldKinds.get(d.name + ":" + f.name) === "int"
          ? int.identity
          : fieldKinds.get(d.name + ":" + f.name) === "http::header[]"
            ? headers.identity
            : str.identity,
    })),
  ),
);
const domain = createDomainRuntime({
  declarations: declarations.map((d) => ({ ...d, parameters: 0 })),
  shapes: [str, int, header, headers, ...errors],
});
const id = (declaration: string) => hash(["error", declaration]);
const api = createJsonActionFetch(domain, {
  transport: id("can.std.http@1::transport_failed"),
  invalidRequest: id("can.std.http@1::invalid_request"),
  bodyLimit: id("can.std.http@1::body_limit"),
  statusError: id("can.std.http@1::status_error"),
  invalidData: id("can.std.codec@1::invalid_data"),
  header: header.identity,
});
const strNode = { identity: "t24::str", kind: "primitive", name: "str" };
const response: Schema = {
  root: "t24::outcome",
  nodes: [
    {
      identity: "t24::outcome",
      kind: "variant",
      name: "outcome",
      leaves: ["t24::done"],
    },
    {
      identity: "t24::done",
      kind: "record",
      name: "done",
      fields: [{ name: "label", type: "t24::str" }],
    },
    strNode,
  ],
};
const request: Schema = {
  root: "t24::wire",
  nodes: [
    {
      identity: "t24::wire",
      kind: "record",
      name: "wire",
      fields: [{ name: "label", type: "t24::str" }],
    },
    strNode,
  ],
};
const cases = [{ leaf: "t24::done", status: 200 }];
const getSite: JsonFetchSite = {
  action: "t24::load",
  method: "GET",
  path: "/invoices/{invoice_id}",
  captures: [{ name: "invoice_id", type: "str" }],
  response,
  cases,
};
const postSite: JsonFetchSite = {
  action: "t24::save",
  method: "POST",
  path: "/invoices/save",
  captures: [],
  request,
  response,
  cases,
};

test("action fetch sends same-origin credentials on GET and POST", async () => {
  const seen: unknown[] = [];
  const realFetch = globalThis.fetch;
  globalThis.fetch = (async (url: unknown, init?: unknown) => {
    seen.push(init);
    return new Response(JSON.stringify({ case: "done", value: { label: "ok" } }), {
      status: 200,
      headers: { "content-type": "application/json" },
    });
  }) as unknown as typeof fetch;
  try {
    const loaded = await api.get("load", "inv-1", getSite, undefined);
    expect(loaded.kind).toBe("ok");
    const saved = await api.post(
      "save",
      record("t24::wire", [["label", "ok"]]),
      postSite,
      undefined,
    );
    expect(saved.kind).toBe("ok");
  } finally {
    globalThis.fetch = realFetch;
  }
  expect(seen.length).toBe(2);
  for (const init of seen) {
    expect((init as { credentials?: unknown }).credentials).toBe("same-origin");
  }
});
