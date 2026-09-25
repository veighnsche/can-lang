import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { errorType, errorPayload, type Completion } from "../completion.ts";
import { record, dataProperty } from "../data.ts";
import { browserPolicy } from "../platform/assets.ts";
import type { Schema } from "../codec/json.ts";
import { ACTION_JSON_BODY_LIMIT } from "../platform/action-json.ts";
import { createActionClient } from "../platform/action-client.ts";
import type { JsonFetchSite } from "../platform/action-json.ts";

// UP13 symbol-based action client: action::request/post lower the
// authored captures record plus the spliced client site to canonical
// same-origin URLs and native fetch. Relative URLs forward to a
// loopback stub through the same-origin path a browser takes; every
// observed URL must satisfy the served content-security-policy.
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
const api = createActionClient(domain, {
  transport: id("can.std.http@1::transport_failed"),
  invalidRequest: id("can.std.http@1::invalid_request"),
  bodyLimit: id("can.std.http@1::body_limit"),
  statusError: id("can.std.http@1::status_error"),
  invalidData: id("can.std.codec@1::invalid_data"),
  header: header.identity,
});

const strNode = { identity: "u13::str", kind: "primitive", name: "str" };
const intNode = { identity: "u13::int", kind: "primitive", name: "int" };
const saveRequest: Schema = {
  root: "u13::seal_wire",
  nodes: [
    {
      identity: "u13::seal_wire",
      kind: "record",
      name: "seal_wire",
      fields: [
        { name: "label", type: "u13::str" },
        { name: "seats", type: "u13::int" },
      ],
    },
    strNode,
    intNode,
  ],
};
const sealResponse: Schema = {
  root: "u13::seal_outcome",
  nodes: [
    {
      identity: "u13::seal_outcome",
      kind: "variant",
      name: "seal_outcome",
      leaves: ["u13::sealed", "u13::seal_failed"],
    },
    {
      identity: "u13::sealed",
      kind: "record",
      name: "sealed",
      fields: [{ name: "label", type: "u13::str" }],
    },
    {
      identity: "u13::seal_failed",
      kind: "record",
      name: "seal_failed",
      fields: [{ name: "reason", type: "u13::str" }],
    },
    strNode,
  ],
};
const loadResponse: Schema = {
  root: "u13::load_outcome",
  nodes: [
    {
      identity: "u13::load_outcome",
      kind: "variant",
      name: "load_outcome",
      leaves: ["u13::found", "u13::missing"],
    },
    {
      identity: "u13::found",
      kind: "record",
      name: "found",
      fields: [{ name: "label", type: "u13::str" }],
    },
    {
      identity: "u13::missing",
      kind: "record",
      name: "missing",
      fields: [{ name: "reason", type: "u13::str" }],
    },
    strNode,
  ],
};
const sealCases = [
  { leaf: "u13::sealed", status: 200 },
  { leaf: "u13::seal_failed", status: 422 },
];
const loadCases = [
  { leaf: "u13::found", status: 200 },
  { leaf: "u13::missing", status: 403 },
];
const loadSite: JsonFetchSite = {
  action: "u13::load_line",
  method: "GET",
  path: "/invoices/{invoice_id}/lines/{line}",
  captures: [
    { name: "invoice_id", type: "str" },
    { name: "line", type: "int" },
  ],
  response: loadResponse,
  cases: loadCases,
};
const exactLoadSite: JsonFetchSite = {
  action: "u13::load_seal",
  method: "GET",
  path: "/invoices/sealed",
  captures: [],
  response: sealResponse,
  cases: sealCases,
};
const saveSite: JsonFetchSite = {
  action: "u13::seal_invoice",
  method: "POST",
  path: "/invoices/sealed",
  captures: [],
  request: saveRequest,
  response: sealResponse,
  cases: sealCases,
};
const keyedSaveSite: JsonFetchSite = {
  action: "u13::seal_keyed",
  method: "POST",
  path: "/invoices/{invoice_id}/sealed",
  captures: [{ name: "invoice_id", type: "str" }],
  request: saveRequest,
  response: sealResponse,
  cases: sealCases,
};
const key = (invoice_id: string, line: bigint) =>
  record("u13::invoice_key", [
    ["invoice_id", invoice_id],
    ["line", line],
  ]);
const body = (label: string, seats: bigint) =>
  record("u13::seal_wire", [
    ["label", label],
    ["seats", seats],
  ]);

function checkFailure(result: Completion<unknown>, identity: string, payload: object) {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw new Error("expected domain failure");
  expect(errorType(result)).toBe(identity);
  expect(errorPayload(result)).toMatchObject(payload);
}
function checkOperation(result: Completion<unknown>, action: string) {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw new Error("expected domain failure");
  expect(domainFailureDiagnostics(result.value).provenance).toEqual({
    boundary: "native",
    operation: action,
  });
}

async function withForwardingFetch<T>(
  port: number,
  seen: string[],
  run: () => Promise<T>,
): Promise<T> {
  expect(browserPolicy).toContain("connect-src 'self'");
  const realFetch = globalThis.fetch;
  globalThis.fetch = (async (url: unknown, init?: unknown) => {
    expect(typeof url).toBe("string");
    const target = url as string;
    expect(target.startsWith("/") && !target.startsWith("//")).toBe(true);
    seen.push(target);
    return realFetch(`http://127.0.0.1:${port}${target}`, init as never);
  }) as unknown as typeof fetch;
  try {
    return await run();
  } finally {
    globalThis.fetch = realFetch;
  }
}

const seenRequests: { method: string; path: string; contentType: string | null; body: string }[] =
  [];
function stubServer(port: number): { stop: () => void } {
  const server = Bun.serve({
    port,
    async fetch(request) {
      const url = new URL(request.url);
      const text = await request.text();
      seenRequests.push({
        method: request.method,
        path: url.pathname,
        contentType: request.headers.get("content-type"),
        body: text,
      });
      const json = (text: string, status: number) =>
        new Response(text, { status, headers: { "content-type": "application/json" } });
      if (request.method === "POST" && url.pathname === "/invoices/sealed") {
        const wire = JSON.parse(text) as { label?: unknown; seats?: unknown };
        return typeof wire.seats === "number" && wire.seats > 0
          ? json(`{"case":"sealed","value":{"label":${JSON.stringify(wire.label)}}}`, 200)
          : json('{"case":"seal_failed","value":{"reason":"empty"}}', 422);
      }
      switch (url.pathname) {
        case "/invoices/sealed":
          return json('{"case":"sealed","value":{"label":"listed"}}', 200);
        case "/invoices/a%20b/lines/3":
          return json('{"case":"found","value":{"label":"a b"}}', 200);
        case "/invoices/gone/lines/1":
          return json('{"case":"missing","value":{"reason":"gone"}}', 403);
        case "/invoices/bad/lines/1":
          return json("not json", 200);
        case "/invoices/html/lines/1":
          return new Response("<p>no</p>", {
            status: 200,
            headers: { "content-type": "text/html" },
          });
        case "/invoices/mismatch/lines/1":
          return json('{"case":"missing","value":{"reason":"x"}}', 200);
        case "/invoices/big/lines/1":
          return json(`{"case":"found","value":{"label":"${"y".repeat(9000)}"}}`, 200);
        case "/invoices/boom/lines/1":
          return new Response("boom", { status: 500 });
        case "/invoices/k7/sealed":
          return json('{"case":"sealed","value":{"label":"keyed"}}', 200);
        default:
          return new Response("Not Found", { status: 404 });
      }
    },
  });
  return {
    stop: () => {
      void server.stop(true);
    },
  };
}

test("request builds canonical capture URLs and GET carries no body", async () => {
  const stub = stubServer(18517);
  try {
    seenRequests.length = 0;
    const seen: string[] = [];
    await withForwardingFetch(18517, seen, async () => {
      const found = await api.request(key("a b", 3n), loadSite, undefined);
      expect(found.kind).toBe("ok");
      if (found.kind !== "ok") throw new Error("wrong outcome");
      expect(dataProperty(found.value, "label")).toBe("a b");
      const missing = await api.request(key("gone", 1n), loadSite, undefined);
      expect(missing.kind).toBe("ok");
      if (missing.kind !== "ok") throw new Error("wrong outcome");
      expect(dataProperty(missing.value, "reason")).toBe("gone");
    });
    expect(seen).toEqual(["/invoices/a%20b/lines/3", "/invoices/gone/lines/1"]);
    expect(seenRequests.length).toBe(2);
    for (const observed of seenRequests) {
      expect(observed.method).toBe("GET");
      expect(observed.body).toBe("");
      expect(observed.contentType).toBeNull();
    }
  } finally {
    stub.stop();
  }
});

test("request normalizes checked colon templates like mount sites", async () => {
  const stub = stubServer(18517);
  try {
    seenRequests.length = 0;
    const seen: string[] = [];
    const colonSite: JsonFetchSite = { ...loadSite, path: "/invoices/:invoice_id/lines/:line" };
    await withForwardingFetch(18517, seen, async () => {
      const found = await api.request(key("a b", 3n), colonSite, undefined);
      expect(found.kind).toBe("ok");
      if (found.kind !== "ok") throw new Error("wrong outcome");
      expect(dataProperty(found.value, "label")).toBe("a b");
    });
    expect(seen).toEqual(["/invoices/a%20b/lines/3"]);
    expect(seenRequests.length).toBe(1);
  } finally {
    stub.stop();
  }
});

test("request and post without captures take the short arity", async () => {
  const stub = stubServer(18517);
  try {
    const seen: string[] = [];
    await withForwardingFetch(18517, seen, async () => {
      const loaded = await api.request(exactLoadSite, undefined);
      expect(loaded.kind).toBe("ok");
      if (loaded.kind !== "ok") throw new Error("wrong outcome");
      expect(dataProperty(loaded.value, "label")).toBe("listed");
      const saved = await api.post(body("inv-3", 4n), saveSite, undefined);
      expect(saved.kind).toBe("ok");
      if (saved.kind !== "ok") throw new Error("wrong outcome");
      expect(dataProperty(saved.value, "label")).toBe("inv-3");
      const failed = await api.post(body("inv-3", 0n), saveSite, undefined);
      expect(failed.kind).toBe("ok");
      if (failed.kind !== "ok") throw new Error("wrong outcome");
      expect(dataProperty(failed.value, "reason")).toBe("empty");
      const keyed = await api.post(
        record("u13::key", [["invoice_id", "k7"]]),
        body("k7", 2n),
        keyedSaveSite,
        undefined,
      );
      expect(keyed.kind).toBe("ok");
      if (keyed.kind !== "ok") throw new Error("wrong outcome");
      expect(dataProperty(keyed.value, "label")).toBe("keyed");
    });
    expect(seen).toEqual([
      "/invoices/sealed",
      "/invoices/sealed",
      "/invoices/sealed",
      "/invoices/k7/sealed",
    ]);
  } finally {
    stub.stop();
  }
});

test("client rejects arity and captures-record disagreement before fetching", async () => {
  let calls = 0;
  const realFetch = globalThis.fetch;
  globalThis.fetch = (async () => {
    calls++;
    return new Response("{}", { status: 200 });
  }) as unknown as typeof fetch;
  try {
    await expect(api.request(loadSite)).rejects.toThrow(
      "action request takes its captures record and site",
    );
    await expect(api.request(key("a", 1n), loadSite, undefined, undefined)).rejects.toThrow(
      "action request takes its captures record and site",
    );
    await expect(api.post(body("b", 1n), saveSite)).rejects.toThrow(
      "action post takes its captures record, wire body and site",
    );
    await expect(
      api.post(key("a", 1n), body("b", 1n), saveSite, undefined, undefined),
    ).rejects.toThrow("action post takes its captures record, wire body and site");
    await expect(api.request(key("a", 1n), exactLoadSite, undefined)).rejects.toThrow(
      "disagrees on its captures record",
    );
    await expect(api.request(loadSite, undefined)).rejects.toThrow(
      "disagrees on its captures record",
    );
    await expect(api.post(saveSite, undefined)).rejects.toThrow(
      "action post takes its captures record, wire body and site",
    );
    expect(calls).toBe(0);
  } finally {
    globalThis.fetch = realFetch;
  }
});

test("client maps unbuildable captures and inadmissible bodies without fetching", async () => {
  let calls = 0;
  const realFetch = globalThis.fetch;
  globalThis.fetch = (async () => {
    calls++;
    return new Response("{}", { status: 200 });
  }) as unknown as typeof fetch;
  try {
    const slash = await api.request(key("a/b", 1n), loadSite, undefined);
    checkFailure(slash, id("can.std.http@1::invalid_request"), { reason: "capture-value" });
    checkOperation(slash, "u13::load_line");
    const mistyped = await api.request(
      record("u13::invoice_key", [
        ["invoice_id", 7n],
        ["line", 1n],
      ]),
      loadSite,
      undefined,
    );
    checkFailure(mistyped, id("can.std.http@1::invalid_request"), { reason: "capture-type" });
    await expect(
      api.request(record("u13::invoice_key", [["invoice_id", "a"]]), loadSite, undefined),
    ).rejects.toThrow("expected own data property");
    const big = await api.post(
      body("x".repeat(ACTION_JSON_BODY_LIMIT + 1), 1n),
      saveSite,
      undefined,
    );
    checkFailure(big, id("can.std.http@1::body_limit"), {
      limit: BigInt(ACTION_JSON_BODY_LIMIT),
    });
    checkOperation(big, "u13::seal_invoice");
    const inadmissible = await api.post({}, saveSite, undefined);
    checkFailure(inadmissible, id("can.std.http@1::invalid_request"), { reason: "request" });
    expect(calls).toBe(0);
  } finally {
    globalThis.fetch = realFetch;
  }
});

test("client maps codec violations and foreign statuses distinctly", async () => {
  const stub = stubServer(18517);
  try {
    const seen: string[] = [];
    await withForwardingFetch(18517, seen, async () => {
      const bad = await api.request(key("bad", 1n), loadSite, undefined);
      checkFailure(bad, id("can.std.codec@1::invalid_data"), {});
      checkOperation(bad, "u13::load_line");
      const html = await api.request(key("html", 1n), loadSite, undefined);
      checkFailure(html, id("can.std.codec@1::invalid_data"), { reason: "media_type" });
      const mismatch = await api.request(key("mismatch", 1n), loadSite, undefined);
      checkFailure(mismatch, id("can.std.codec@1::invalid_data"), { reason: "variant_tag" });
      const big = await api.request(key("big", 1n), loadSite, undefined);
      checkFailure(big, id("can.std.codec@1::invalid_data"), { reason: "byte_limit" });
      const boom = await api.request(key("boom", 1n), loadSite, undefined);
      checkFailure(boom, id("can.std.http@1::status_error"), { status: 500n, headers: [] });
      checkOperation(boom, "u13::load_line");
      const gone = await api.request(key("gone-x", 404n), loadSite, undefined);
      checkFailure(gone, id("can.std.http@1::status_error"), { status: 404n });
    });
    expect(seen.length).toBe(6);
  } finally {
    stub.stop();
  }
});

test("client maps transport faults and aborts to transport failures", async () => {
  const realFetch = globalThis.fetch;
  try {
    globalThis.fetch = (async () => {
      throw new TypeError("connection refused");
    }) as unknown as typeof fetch;
    const refused = await api.request(key("a", 1n), loadSite, undefined);
    checkFailure(refused, id("can.std.http@1::transport_failed"), { phase: "connect" });
    checkOperation(refused, "u13::load_line");
    globalThis.fetch = (async () => {
      throw new DOMException("aborted", "AbortError");
    }) as unknown as typeof fetch;
    const aborted = await api.post(body("b", 1n), saveSite, undefined);
    checkFailure(aborted, id("can.std.http@1::transport_failed"), { phase: "cancelled" });
  } finally {
    globalThis.fetch = realFetch;
  }
});

test("client enforces verb and contract agreement on the spliced site", async () => {
  await expect(api.request(saveSite, undefined)).rejects.toThrow("disagrees with its verb");
  await expect(api.post(body("b", 1n), exactLoadSite, undefined)).rejects.toThrow(
    "disagrees with its verb",
  );
  await expect(
    api.request(key("a", 1n), { ...loadSite, request: saveRequest }, undefined),
  ).rejects.toThrow("bodyless GET site with a request contract");
  await expect(
    api.post(body("b", 1n), { ...saveSite, request: undefined }, undefined),
  ).rejects.toThrow("POST site with no request contract");
  await expect(api.request(key("a", 1n), { ...loadSite, action: "" }, undefined)).rejects.toThrow(
    "carries no action identity",
  );
});
