import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, type FailureShape } from "../domain.ts";
import { success, value, errorType, errorPayload, type Completion } from "../completion.ts";
import { record, dataProperty } from "../data.ts";
import { runOwnedRoot, resourceStatus } from "../owner.ts";
import { createServer } from "../platform/server.ts";
import { createRouter } from "../platform/router.ts";
import { browserPolicy } from "../platform/assets.ts";
import type { Schema } from "../codec/json.ts";
import {
  ACTION_JSON_BODY_LIMIT,
  createJsonActions,
  createJsonActionFetch,
  type ActionJsonHandler,
  type EmittedActionEntry,
  type JsonFetchSite,
} from "../platform/action-json.ts";

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
  "http::invalid_route",
  "http::duplicate_route",
  "http::ambiguous_route",
  "http::invalid_server_config",
  "http::bind_failed",
  "http::shutdown_failed",
  "stream::read_failed",
  "stream::write_failed",
  "stream::close_failed",
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
const jsonActions = createJsonActions(domain, {
  invalid: id("can.std.http@1::invalid_request"),
  invalidData: id("can.std.codec@1::invalid_data"),
  close: id("can.std.stream@1::close_failed"),
  writeFailed: id("can.std.stream@1::write_failed"),
  limit: id("can.std.http@1::body_limit"),
});
const api = createJsonActionFetch(domain, {
  transport: id("can.std.http@1::transport_failed"),
  invalidRequest: id("can.std.http@1::invalid_request"),
  bodyLimit: id("can.std.http@1::body_limit"),
  statusError: id("can.std.http@1::status_error"),
  invalidData: id("can.std.codec@1::invalid_data"),
  header: header.identity,
});

const strNode = { identity: "t23::str", kind: "primitive", name: "str" };
const intNode = { identity: "t23::int", kind: "primitive", name: "int" };
const saveRequest: Schema = {
  root: "t23::seal_wire",
  nodes: [
    {
      identity: "t23::seal_wire",
      kind: "record",
      name: "seal_wire",
      fields: [
        { name: "label", type: "t23::str" },
        { name: "seats", type: "t23::int" },
      ],
    },
    strNode,
    intNode,
  ],
};
const sealResponse: Schema = {
  root: "t23::seal_outcome",
  nodes: [
    {
      identity: "t23::seal_outcome",
      kind: "variant",
      name: "seal_outcome",
      leaves: ["t23::sealed", "t23::seal_failed"],
    },
    {
      identity: "t23::sealed",
      kind: "record",
      name: "sealed",
      fields: [{ name: "label", type: "t23::str" }],
    },
    {
      identity: "t23::seal_failed",
      kind: "record",
      name: "seal_failed",
      fields: [{ name: "reason", type: "t23::str" }],
    },
    strNode,
  ],
};
const loadResponse: Schema = {
  root: "t23::load_outcome",
  nodes: [
    {
      identity: "t23::load_outcome",
      kind: "variant",
      name: "load_outcome",
      leaves: ["t23::found", "t23::missing"],
    },
    {
      identity: "t23::found",
      kind: "record",
      name: "found",
      fields: [{ name: "label", type: "t23::str" }],
    },
    {
      identity: "t23::missing",
      kind: "record",
      name: "missing",
      fields: [{ name: "reason", type: "t23::str" }],
    },
    strNode,
  ],
};
const sealCases = [
  { leaf: "t23::sealed", status: 200 },
  { leaf: "t23::seal_failed", status: 422 },
];
const loadCases = [
  { leaf: "t23::found", status: 200 },
  { leaf: "t23::missing", status: 403 },
];
const saveSite: JsonFetchSite = {
  action: "t23::seal_invoice",
  method: "POST",
  path: "/invoices/sealed",
  captures: [],
  request: saveRequest,
  response: sealResponse,
  cases: sealCases,
};
const loadSite: JsonFetchSite = {
  action: "t23::load_line",
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
  action: "t23::load_seal",
  method: "GET",
  path: "/invoices/sealed",
  captures: [],
  response: sealResponse,
  cases: sealCases,
};
const saveEntry: EmittedActionEntry = {
  identity: "t23::seal_invoice",
  method: "POST",
  path: "/invoices/sealed",
  captures: [],
  body: { mode: "json", type: "t23::seal_wire", schema: saveRequest },
  handler: "t23::seal_validated",
  result: "t23::seal_outcome",
  cases: sealCases,
  responseSchema: sealResponse,
};
const loadEntry: EmittedActionEntry = {
  identity: "t23::load_seal",
  method: "GET",
  path: "/invoices/sealed",
  captures: [],
  handler: "t23::load_sealed",
  result: "t23::seal_outcome",
  cases: sealCases,
  responseSchema: sealResponse,
};
const saveHandler: ActionJsonHandler = async (inputs) => {
  const seats = dataProperty(inputs[0], "seats") as bigint;
  const label = dataProperty(inputs[0], "label") as string;
  return seats > 0n
    ? success(record("t23::sealed", [["label", label]]))
    : success(record("t23::seal_failed", [["reason", "empty"]]));
};
const loadHandler: ActionJsonHandler = async (inputs) => {
  expect(inputs.length).toBe(0);
  return success(record("t23::sealed", [["label", "listed"]]));
};

function checkFailure(result: Completion<unknown>, identity: string, payload: object) {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw new Error("expected domain failure");
  expect(errorType(result)).toBe(identity);
  expect(errorPayload(result)).toMatchObject(payload);
}

async function serve(
  port: number,
  mounts: readonly (readonly [EmittedActionEntry, ActionJsonHandler])[],
): Promise<unknown> {
  const config = value(await server.makeConfig("127.0.0.1", BigInt(port), 65536n, 5000n));
  const routes: unknown[] = [];
  for (const [entry, handler] of mounts)
    routes.push(value(await jsonActions.mount(router, entry, handler)));
  const table = value(await router.make(routes));
  return value(await server.start(config, table));
}

async function stop(token: unknown): Promise<void> {
  expect(resourceStatus(token)).toMatchObject({ kind: "server", state: "open", leases: 0 });
  expect((await server.stop(token)).kind).toBe("ok");
  expect(resourceStatus(token)).toMatchObject({ kind: "server", state: "closed", leases: 0 });
}

// withForwardingFetch runs the adapter against a live server through the
// same-origin path a browser takes: the adapter only ever builds relative
// URLs, and the stub asserts that relativity before forwarding to the
// loopback server. Every observed URL must satisfy the served
// content-security-policy, which pins connect-src 'self'.
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

test("fetch POST and exact GET round-trip through the live JSON adapter", async () => {
  const owned = await runOwnedRoot(async () => {
    const token = await serve(18515, [
      [saveEntry, saveHandler],
      [loadEntry, loadHandler],
    ]);
    const seen: string[] = [];
    await withForwardingFetch(18515, seen, async () => {
      const saved = await api.post(
        "seal_invoice",
        record("t23::seal_wire", [
          ["label", "inv-3"],
          ["seats", 4n],
        ]),
        saveSite,
        undefined,
      );
      expect(saved.kind).toBe("ok");
      if (saved.kind !== "ok") throw new Error("wrong outcome");
      expect(dataProperty(saved.value, "label")).toBe("inv-3");
      const failed = await api.post(
        "seal_invoice",
        record("t23::seal_wire", [
          ["label", "inv-3"],
          ["seats", 0n],
        ]),
        saveSite,
        undefined,
      );
      expect(failed.kind).toBe("ok");
      if (failed.kind !== "ok") throw new Error("wrong outcome");
      expect(dataProperty(failed.value, "reason")).toBe("empty");
      const loaded = await api.get("load_seal", exactLoadSite, undefined);
      expect(loaded.kind).toBe("ok");
      if (loaded.kind !== "ok") throw new Error("wrong outcome");
      expect(dataProperty(loaded.value, "label")).toBe("listed");
    });
    expect(seen).toEqual(["/invoices/sealed", "/invoices/sealed", "/invoices/sealed"]);
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

function stubServer(port: number): { stop: () => void } {
  const server = Bun.serve({
    port,
    fetch(request) {
      const url = new URL(request.url);
      switch (url.pathname) {
        case "/invoices/a%20b/lines/3":
          return new Response('{"case":"found","value":{"label":"a b"}}', {
            status: 200,
            headers: { "content-type": "application/json" },
          });
        case "/invoices/gone/lines/1":
          return new Response('{"case":"missing","value":{"reason":"gone"}}', {
            status: 403,
            headers: { "content-type": "application/json" },
          });
        case "/invoices/bad/lines/1":
          return new Response("not json", {
            status: 200,
            headers: { "content-type": "application/json" },
          });
        case "/invoices/html/lines/1":
          return new Response("<p>no</p>", {
            status: 200,
            headers: { "content-type": "text/html" },
          });
        case "/invoices/mismatch/lines/1":
          return new Response('{"case":"missing","value":{"reason":"x"}}', {
            status: 200,
            headers: { "content-type": "application/json" },
          });
        case "/invoices/boom/lines/1":
          return new Response("boom", { status: 500 });
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

test("fetch GET builds canonical capture URLs over live HTTP", async () => {
  const stub = stubServer(18516);
  try {
    const seen: string[] = [];
    await withForwardingFetch(18516, seen, async () => {
      const found = await api.get("load_line", "a b", 3n, loadSite, undefined);
      expect(found.kind).toBe("ok");
      if (found.kind !== "ok") throw new Error("wrong outcome");
      expect(dataProperty(found.value, "label")).toBe("a b");
      const missing = await api.get("load_line", "gone", 1n, loadSite, undefined);
      expect(missing.kind).toBe("ok");
      if (missing.kind !== "ok") throw new Error("wrong outcome");
      expect(dataProperty(missing.value, "reason")).toBe("gone");
    });
    expect(seen).toEqual(["/invoices/a%20b/lines/3", "/invoices/gone/lines/1"]);
  } finally {
    stub.stop();
  }
});

test("fetch maps bad representations to codec and foreign statuses to status_error", async () => {
  const stub = stubServer(18516);
  try {
    const seen: string[] = [];
    await withForwardingFetch(18516, seen, async () => {
      const bad = await api.get("load_line", "bad", 1n, loadSite, undefined);
      checkFailure(bad, id("can.std.codec@1::invalid_data"), {});
      const html = await api.get("load_line", "html", 1n, loadSite, undefined);
      checkFailure(html, id("can.std.codec@1::invalid_data"), { reason: "media_type" });
      const mismatch = await api.get("load_line", "mismatch", 1n, loadSite, undefined);
      checkFailure(mismatch, id("can.std.codec@1::invalid_data"), { reason: "variant_tag" });
      const boom = await api.get("load_line", "boom", 1n, loadSite, undefined);
      checkFailure(boom, id("can.std.http@1::status_error"), { status: 500n, headers: [] });
      const gone = await api.get("load_line", "gone-x", 404n, loadSite, undefined);
      checkFailure(gone, id("can.std.http@1::status_error"), { status: 404n });
    });
    expect(seen.length).toBe(5);
  } finally {
    stub.stop();
  }
});

test("fetch rejects unbuildable captures and oversize or inadmissible bodies", async () => {
  let calls = 0;
  const realFetch = globalThis.fetch;
  globalThis.fetch = (async () => {
    calls++;
    return new Response("{}", { status: 200 });
  }) as unknown as typeof fetch;
  try {
    const slash = await api.get("load_line", "a/b", 1n, loadSite, undefined);
    checkFailure(slash, id("can.std.http@1::invalid_request"), { reason: "capture-value" });
    const huge = await api.get("load_line", "ok", 2n ** 70n, loadSite, undefined);
    checkFailure(huge, id("can.std.http@1::invalid_request"), { reason: "capture-value" });
    const mistyped = await api.get("load_line", 7n, 1n, loadSite, undefined);
    checkFailure(mistyped, id("can.std.http@1::invalid_request"), { reason: "capture-type" });
    const big = await api.post(
      "seal_invoice",
      record("t23::seal_wire", [
        ["label", "x".repeat(ACTION_JSON_BODY_LIMIT + 1)],
        ["seats", 1n],
      ]),
      saveSite,
      undefined,
    );
    checkFailure(big, id("can.std.http@1::body_limit"), { limit: BigInt(ACTION_JSON_BODY_LIMIT) });
    const inadmissible = await api.post("seal_invoice", {}, saveSite, undefined);
    checkFailure(inadmissible, id("can.std.http@1::invalid_request"), { reason: "request" });
    expect(calls).toBe(0);
  } finally {
    globalThis.fetch = realFetch;
  }
});

test("fetch maps transport faults and aborts to transport failures, never domain cases", async () => {
  const realFetch = globalThis.fetch;
  try {
    globalThis.fetch = (async () => {
      throw new TypeError("connection refused");
    }) as unknown as typeof fetch;
    const refused = await api.get("load_line", "a", 1n, loadSite, undefined);
    checkFailure(refused, id("can.std.http@1::transport_failed"), { phase: "connect" });
    globalThis.fetch = (async () => {
      throw new DOMException("settled after teardown", "AbortError");
    }) as unknown as typeof fetch;
    // Browser cancellation (navigation, disposal, stack abort) rejects the
    // fetch; the adapter reports a cancelled transport failure. It is a
    // failure, not a domain case, and carries no rollback claim: the
    // server may or may not have committed, and only a reread can tell.
    const aborted = await api.post(
      "seal_invoice",
      record("t23::seal_wire", [
        ["label", "inv-9"],
        ["seats", 1n],
      ]),
      saveSite,
      undefined,
    );
    expect(aborted.kind).toBe("domain");
    checkFailure(aborted, id("can.std.http@1::transport_failed"), { phase: "cancelled" });
  } finally {
    globalThis.fetch = realFetch;
  }
});

test("fetch validates its site, verb, arity and name", async () => {
  const body = record("t23::seal_wire", [
    ["label", "inv-1"],
    ["seats", 1n],
  ]);
  await expect(api.get(42, "a", 1n, loadSite, undefined)).rejects.toThrow(
    "fetch action name is not a string",
  );
  await expect(api.get("load_line", "a", 1n, undefined, undefined)).rejects.toThrow(
    "fetch site carries no action identity",
  );
  await expect(
    api.get("load_line", "a", 1n, { ...loadSite, method: "POST", request: saveRequest }, undefined),
  ).rejects.toThrow("disagrees with its verb");
  await expect(
    api.post("seal_invoice", body, { ...saveSite, method: "GET", request: undefined }, undefined),
  ).rejects.toThrow("disagrees with its verb");
  await expect(api.get("load_line", "a", loadSite, undefined)).rejects.toThrow("takes 2 inputs");
  await expect(api.get("load_line", "a", 1n, 2n, loadSite, undefined)).rejects.toThrow(
    "takes 2 inputs",
  );
  await expect(api.post("seal_invoice", saveSite, undefined)).rejects.toThrow("takes 1 inputs");
  await expect(
    api.get("load_seal", { ...exactLoadSite, request: saveRequest }, undefined),
  ).rejects.toThrow("bodyless GET site with a request contract");
  const { request: _dropped, ...bare } = saveSite;
  await expect(api.post("seal_invoice", body, bare, undefined)).rejects.toThrow(
    "POST site with no request contract",
  );
  await expect(
    api.get("load_line", "a", 1n, { ...loadSite, path: "https://evil.test/x" }, undefined),
  ).rejects.toThrow("action route");
});
