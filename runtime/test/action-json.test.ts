import { test, expect } from "bun:test";
import { connect } from "node:net";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, type FailureShape } from "../domain.ts";
import { success, failure, value, errorType } from "../completion.ts";
import { record, dataProperty } from "../data.ts";
import { runOwnedRoot, resourceStatus } from "../owner.ts";
import { createServer } from "../platform/server.ts";
import { createRouter } from "../platform/router.ts";
import { nativeResponse, snapshotRequest } from "../platform/http.ts";
import type { Schema } from "../codec/json.ts";
import {
  ACTION_JSON_BODY_LIMIT,
  createJsonActions,
  fetchJsonAction,
  type ActionJsonHandler,
  type EmittedActionEntry,
} from "../platform/action-json.ts";

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
const jsonActions = createJsonActions(domain, {
  invalid: id("can.std.http@1::invalid_request"),
  invalidData: id("can.std.codec@1::invalid_data"),
  close: id("can.std.stream@1::close_failed"),
  writeFailed: id("can.std.stream@1::write_failed"),
  limit: id("can.std.http@1::body_limit"),
});

const strNode = { identity: "t13::str", kind: "primitive", name: "str" };
const intNode = { identity: "t13::int", kind: "primitive", name: "int" };
const floatNode = { identity: "t13::float", kind: "primitive", name: "float" };
const saveRequest: Schema = {
  root: "t13::seal_wire",
  nodes: [
    {
      identity: "t13::seal_wire",
      kind: "record",
      name: "seal_wire",
      fields: [
        { name: "label", type: "t13::str" },
        { name: "seats", type: "t13::int" },
      ],
    },
    strNode,
    intNode,
  ],
};
const sealResponse: Schema = {
  root: "t13::seal_outcome",
  nodes: [
    {
      identity: "t13::seal_outcome",
      kind: "variant",
      name: "seal_outcome",
      leaves: ["t13::sealed", "t13::seal_failed"],
    },
    {
      identity: "t13::sealed",
      kind: "record",
      name: "sealed",
      fields: [{ name: "label", type: "t13::str" }],
    },
    {
      identity: "t13::seal_failed",
      kind: "record",
      name: "seal_failed",
      fields: [{ name: "reason", type: "t13::str" }],
    },
    strNode,
  ],
};
const measureRequest: Schema = {
  root: "t13::measure_wire",
  nodes: [
    {
      identity: "t13::measure_wire",
      kind: "record",
      name: "measure_wire",
      fields: [
        { name: "seats", type: "t13::int" },
        { name: "ratio", type: "t13::float" },
      ],
    },
    intNode,
    floatNode,
  ],
};
const measureResponse: Schema = {
  root: "t13::measure_outcome",
  nodes: [
    {
      identity: "t13::measure_outcome",
      kind: "variant",
      name: "measure_outcome",
      leaves: ["t13::measure_ok", "t13::measure_failed"],
    },
    {
      identity: "t13::measure_ok",
      kind: "record",
      name: "measure_ok",
      fields: [
        { name: "seats", type: "t13::int" },
        { name: "ratio", type: "t13::float" },
      ],
    },
    {
      identity: "t13::measure_failed",
      kind: "record",
      name: "measure_failed",
      fields: [{ name: "reason", type: "t13::str" }],
    },
    intNode,
    floatNode,
    strNode,
  ],
};
const sealCases = [
  { leaf: "t13::sealed", status: 200 },
  { leaf: "t13::seal_failed", status: 422 },
];
const saveEntry: EmittedActionEntry = {
  identity: "t13::seal_invoice",
  method: "POST",
  path: "/invoices/sealed",
  captures: [],
  body: { mode: "json", type: "t13::seal_wire", schema: saveRequest },
  handler: "t13::seal_validated",
  result: "t13::seal_outcome",
  cases: sealCases,
  responseSchema: sealResponse,
};
const loadEntry: EmittedActionEntry = {
  identity: "t13::load_seal",
  method: "GET",
  path: "/invoices/sealed",
  captures: [],
  handler: "t13::load_sealed",
  result: "t13::seal_outcome",
  cases: sealCases,
  responseSchema: sealResponse,
};
const measureEntry: EmittedActionEntry = {
  identity: "t13::measure_load",
  method: "POST",
  path: "/measure",
  captures: [],
  body: { mode: "json", type: "t13::measure_wire", schema: measureRequest },
  handler: "t13::measure_validated",
  result: "t13::measure_outcome",
  cases: [
    { leaf: "t13::measure_ok", status: 200 },
    { leaf: "t13::measure_failed", status: 422 },
  ],
  responseSchema: measureResponse,
};

const saveHandler: ActionJsonHandler = async (inputs) => {
  const seats = dataProperty(inputs[0], "seats") as bigint;
  const label = dataProperty(inputs[0], "label") as string;
  return seats > 0n
    ? success(record("t13::sealed", [["label", label]]))
    : success(record("t13::seal_failed", [["reason", "empty"]]));
};
const loadHandler: ActionJsonHandler = async (inputs) => {
  expect(inputs.length).toBe(0);
  return success(record("t13::sealed", [["label", "listed"]]));
};
const measureHandler: ActionJsonHandler = async (inputs) => {
  const seats = dataProperty(inputs[0], "seats") as bigint;
  const ratio = dataProperty(inputs[0], "ratio") as number;
  return success(
    record("t13::measure_ok", [
      ["seats", seats],
      ["ratio", ratio],
    ]),
  );
};

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

function rawRequest(port: number, head: string, body: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const socket = connect(port, "127.0.0.1", () => {
      socket.write(head + "\r\n\r\n" + body);
    });
    let raw = "";
    socket.on("data", (chunk) => {
      raw += chunk.toString();
    });
    socket.on("close", () => resolve(raw));
    socket.on("error", reject);
  });
}

test("POST save round-trips JSON with finite case statuses", async () => {
  const owned = await runOwnedRoot(async () => {
    const token = await serve(18461, [[saveEntry, saveHandler]]);
    const saved = await fetch("http://127.0.0.1:18461/invoices/sealed", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ label: "inv-1", seats: 2 }),
    });
    expect(saved.status).toBe(200);
    expect(saved.headers.get("content-type")).toBe("application/json; charset=utf-8");
    expect(saved.headers.get("x-content-type-options")).toBe("nosniff");
    expect(await saved.text()).toBe('{"case":"sealed","value":{"label":"inv-1"}}');
    const failed = await fetch("http://127.0.0.1:18461/invoices/sealed", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ label: "inv-1", seats: 0 }),
    });
    expect(failed.status).toBe(422);
    expect(failed.headers.get("content-type")).toBe("application/json; charset=utf-8");
    expect(await failed.text()).toBe('{"case":"seal_failed","value":{"reason":"empty"}}');
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("POST rejects malformed and duplicate JSON with a fixed 400", async () => {
  const owned = await runOwnedRoot(async () => {
    const token = await serve(18462, [[saveEntry, saveHandler]]);
    const bodies = [
      '{"label":}',
      "not json",
      '{"label":"a","label":"b","seats":2}',
      '{"label":1,"seats":2}',
      '{"label":"a"}',
      '{"label":"a","seats":2,"extra":true}',
      "",
    ];
    for (const body of bodies) {
      const response = await fetch("http://127.0.0.1:18462/invoices/sealed", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body,
      });
      expect(response.status).toBe(400);
      expect(response.headers.get("content-type")).toBe("text/plain; charset=utf-8");
      expect(response.headers.get("x-content-type-options")).toBe("nosniff");
      expect(await response.text()).toBe("Bad Request");
    }
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("POST enforces the JSON media type and the 8192-byte limit", async () => {
  expect(ACTION_JSON_BODY_LIMIT).toBe(8192);
  const owned = await runOwnedRoot(async () => {
    const token = await serve(18463, [[saveEntry, saveHandler]]);
    const headerSets: Record<string, string>[] = [
      {},
      { "content-type": "text/plain" },
      { "content-type": "application/json; charset=latin-1" },
      { "content-type": "application/json; charset=utf-8; version=2" },
      { "content-type": "not-a-media-type" },
    ];
    for (const headers of headerSets) {
      const response = await fetch("http://127.0.0.1:18463/invoices/sealed", {
        method: "POST",
        headers,
        body: JSON.stringify({ label: "inv-1", seats: 2 }),
      });
      expect(response.status).toBe(415);
      expect(await response.text()).toBe("Unsupported Media Type");
    }
    const utf8 = await fetch("http://127.0.0.1:18463/invoices/sealed", {
      method: "POST",
      headers: { "content-type": "application/json; charset=utf-8" },
      body: JSON.stringify({ label: "inv-1", seats: 2 }),
    });
    expect(utf8.status).toBe(200);
    const prefix = '{"label":"';
    const suffix = '","seats":2}';
    const exact = await fetch("http://127.0.0.1:18463/invoices/sealed", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: prefix + "p".repeat(8192 - prefix.length - suffix.length) + suffix,
    });
    expect(exact.status).toBe(200);
    const over = await fetch("http://127.0.0.1:18463/invoices/sealed", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: prefix + "p".repeat(8193 - prefix.length - suffix.length) + suffix,
    });
    expect(over.status).toBe(413);
    expect(over.headers.get("content-type")).toBe("text/plain; charset=utf-8");
    expect(await over.text()).toBe("Payload Too Large");
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("GET load answers JSON and rejects any body bytes", async () => {
  const owned = await runOwnedRoot(async () => {
    const token = await serve(18464, [[loadEntry, loadHandler]]);
    const loaded = await fetch("http://127.0.0.1:18464/invoices/sealed");
    expect(loaded.status).toBe(200);
    expect(loaded.headers.get("content-type")).toBe("application/json; charset=utf-8");
    expect(await loaded.text()).toBe('{"case":"sealed","value":{"label":"listed"}}');
    const emptyLength = await rawRequest(
      18464,
      "GET /invoices/sealed HTTP/1.1\r\nHost: 127.0.0.1:18464\r\nConnection: close\r\nContent-Length: 0",
      "",
    );
    expect(emptyLength.split("\r\n")[0]).toBe("HTTP/1.1 200 OK");
    const withBody = await rawRequest(
      18464,
      "GET /invoices/sealed HTTP/1.1\r\nHost: 127.0.0.1:18464\r\nConnection: close\r\nContent-Length: 1",
      "x",
    );
    // The pinned Bun drops GET bodies before ingress, so this arrives empty;
    // either delivery outcome stays inside the contract: a delivered body is
    // a fixed 400, a dropped one loads normally, and neither serves HTML.
    const [statusLine, ...rest] = withBody.split("\r\n");
    expect(["HTTP/1.1 200 OK", "HTTP/1.1 400 Bad Request"]).toContain(statusLine);
    expect(rest.join("\r\n").toLowerCase()).not.toContain("text/html");
    if (statusLine === "HTTP/1.1 400 Bad Request")
      expect(rest.join("\r\n")).toContain("Bad Request");
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("GET callback rejects delivered body bytes with a fixed 400", async () => {
  const run = jsonActions.callback(loadEntry, loadHandler);
  const bare = new Request("http://127.0.0.1:18464/invoices/sealed");
  const clean = await snapshotRequest(bare, 65536);
  if (clean.kind !== "request") throw new Error("snapshot rejected");
  const loaded = nativeResponse(value(await run(clean.value)));
  expect(loaded.status).toBe(200);
  expect(await loaded.text()).toBe('{"case":"sealed","value":{"label":"listed"}}');
  // Native Request construction forbids GET bodies, so the delivered-bytes
  // case shadows the method after building a body-carrying request.
  const smuggled = new Request("http://127.0.0.1:18464/invoices/sealed", {
    method: "POST",
    body: "x",
  });
  Object.defineProperty(smuggled, "method", { value: "GET", configurable: true });
  const dirty = await snapshotRequest(smuggled, 65536);
  if (dirty.kind !== "request") throw new Error("snapshot rejected");
  const rejected = nativeResponse(value(await run(dirty.value)));
  expect(rejected.status).toBe(400);
  expect(rejected.headers.get("content-type")).toBe("text/plain; charset=utf-8");
  expect(await rejected.text()).toBe("Bad Request");
});

test("handler and encoding failures become a fixed 500, never HTML", async () => {
  const forged: ActionJsonHandler = async () => success(record("t13::archived", [["label", "x"]]));
  const broken: ActionJsonHandler = async () =>
    failure(
      domain.create(
        id("can.std.codec@1::invalid_data"),
        record(id("can.std.codec@1::invalid_data"), [
          ["path", ""],
          ["reason", "type"],
        ]),
        origin,
      ),
    );
  const owned = await runOwnedRoot(async () => {
    const config = value(await server.makeConfig("127.0.0.1", 18465n, 65536n, 5000n));
    const forgedRoute = value(
      await jsonActions.mount(
        router,
        { ...saveEntry, identity: "t13::forged", path: "/forged" },
        forged,
      ),
    );
    const brokenRoute = value(
      await jsonActions.mount(
        router,
        { ...saveEntry, identity: "t13::broken", path: "/broken" },
        broken,
      ),
    );
    const table = value(await router.make([forgedRoute, brokenRoute]));
    const token = value(await server.start(config, table));
    for (const path of ["/forged", "/broken"]) {
      const response = await fetch(`http://127.0.0.1:18465${path}`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ label: "inv-1", seats: 2 }),
      });
      expect(response.status).toBe(500);
      expect(response.headers.get("content-type")).toBe("text/plain; charset=utf-8");
      expect(await response.text()).toBe("Internal Server Error");
    }
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("mount rejects captures, form bodies and malformed metadata", async () => {
  const formEntry: EmittedActionEntry = {
    ...saveEntry,
    identity: "t13::form_save",
    body: { mode: "form", type: "t13::line_wire", form: { root: "t13::line_wire", fields: [] } },
  };
  for (const [name, entry] of [
    ["captures", { ...saveEntry, captures: [{ name: "invoice_id", type: "str" }] }],
    ["form body", formEntry],
    ["get with body", { ...loadEntry, body: saveEntry.body }],
    ["post without body", { ...saveEntry, body: undefined }],
    ["unknown method", { ...saveEntry, method: "PUT" }],
    ["empty cases", { ...saveEntry, cases: [] }],
    ["bodiless status", { ...saveEntry, cases: [{ leaf: "t13::sealed", status: 204 }] }],
    [
      "duplicate leaf",
      {
        ...saveEntry,
        cases: [
          { leaf: "t13::sealed", status: 200 },
          { leaf: "t13::sealed", status: 201 },
        ],
      },
    ],
    ["missing response schema", { ...saveEntry, responseSchema: undefined }],
  ] as const) {
    let message = "";
    try {
      await jsonActions.mount(router, entry, saveHandler);
    } catch (cause) {
      message = (cause as Error).message;
    }
    expect(message).not.toBe("");
    if (name === "captures") expect(message).toContain("exact JSON paths only");
    if (name === "form body") expect(message).toContain("JSON bodies only");
  }
  let handlerMessage = "";
  try {
    await jsonActions.mount(router, saveEntry, "sealed" as unknown as ActionJsonHandler);
  } catch (cause) {
    handlerMessage = (cause as Error).message;
  }
  expect(handlerMessage).toContain("no callable handler");
  const badPath = await jsonActions.mount(
    router,
    { ...saveEntry, path: "invoices/sealed" },
    saveHandler,
  );
  expect(badPath.kind).toBe("domain");
  expect(errorType(badPath)).toBe(id("can.std.http@1::invalid_route"));
});

test("int64 and finite floats survive the JSON action round trip", async () => {
  const owned = await runOwnedRoot(async () => {
    const token = await serve(18466, [[measureEntry, measureHandler]]);
    const response = await fetch("http://127.0.0.1:18466/measure", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: '{"seats":9007199254740993,"ratio":0.1}',
    });
    expect(response.status).toBe(200);
    // The int64 keeps exact digits through decode and encode; it never passes
    // through a lossy native number on the adapter path.
    expect(await response.text()).toBe(
      '{"case":"measure_ok","value":{"seats":9007199254740993,"ratio":0.1}}',
    );
    const nonfinite = await fetch("http://127.0.0.1:18466/measure", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: '{"seats":1,"ratio":1e999}',
    });
    expect(nonfinite.status).toBe(400);
    expect(await nonfinite.text()).toBe("Bad Request");
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("nonfinite handler values become a fixed 500", async () => {
  const wild: ActionJsonHandler = async () =>
    success(
      record("t13::measure_ok", [
        ["seats", 1n],
        ["ratio", Number.POSITIVE_INFINITY],
      ]),
    );
  const owned = await runOwnedRoot(async () => {
    const token = await serve(18467, [[{ ...measureEntry, path: "/wild" }, wild]]);
    const response = await fetch("http://127.0.0.1:18467/wild", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: '{"seats":1,"ratio":0.5}',
    });
    expect(response.status).toBe(500);
    expect(response.headers.get("content-type")).toBe("text/plain; charset=utf-8");
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

function stubServer(port: number): { stop: () => void } {
  const server = Bun.serve({
    port,
    hostname: "127.0.0.1",
    fetch(request) {
      const path = new URL(request.url).pathname;
      switch (path) {
        case "/ok":
          return Response.json({ case: "sealed", value: { label: "inv-9" } });
        case "/rejected":
          return Response.json(
            { case: "seal_failed", value: { reason: "stale" } },
            { status: 422 },
          );
        case "/teapot":
          return new Response("short and stout", { status: 418 });
        case "/plain":
          return new Response("sealed", {
            status: 200,
            headers: { "content-type": "text/plain; charset=utf-8" },
          });
        case "/html":
          return new Response("<b>sealed</b>", {
            status: 200,
            headers: { "content-type": "text/html; charset=utf-8" },
          });
        case "/badtag":
          return new Response('{"case":"archived","value":{"label":"x"}}', {
            status: 200,
            headers: { "content-type": "application/json" },
          });
        case "/duplicate":
          return new Response('{"case":"sealed","value":{"label":"a","label":"b"}}', {
            status: 200,
            headers: { "content-type": "application/json" },
          });
        case "/mismatch":
          return new Response('{"case":"seal_failed","value":{"reason":"x"}}', {
            status: 200,
            headers: { "content-type": "application/json" },
          });
        case "/big":
          return new Response('{"case":"sealed","value":{"label":"' + "q".repeat(64) + '"}}', {
            status: 200,
            headers: { "content-type": "application/json" },
          });
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

test("fetch maps finite cases to ok and other statuses to unexpected_status", async () => {
  const stub = stubServer(18468);
  try {
    const base = {
      method: "GET" as const,
      response: sealResponse,
      cases: sealCases,
    };
    const ok = await fetchJsonAction({ ...base, url: "http://127.0.0.1:18468/ok" });
    expect(ok.kind).toBe("ok");
    if (ok.kind !== "ok") throw new Error("wrong outcome");
    expect(ok.status).toBe(200);
    expect(ok.leaf).toBe("t13::sealed");
    expect(dataProperty(ok.value, "label")).toBe("inv-9");
    const rejected = await fetchJsonAction({ ...base, url: "http://127.0.0.1:18468/rejected" });
    expect(rejected.kind).toBe("ok");
    if (rejected.kind !== "ok") throw new Error("wrong outcome");
    expect(rejected.status).toBe(422);
    expect(rejected.leaf).toBe("t13::seal_failed");
    for (const path of ["/teapot", "/missing"]) {
      const outcome = await fetchJsonAction({ ...base, url: `http://127.0.0.1:18468${path}` });
      expect(outcome.kind).toBe("unexpected_status");
      if (outcome.kind !== "unexpected_status") throw new Error("wrong outcome");
      expect(outcome.status).toBe(path === "/teapot" ? 418 : 404);
    }
  } finally {
    stub.stop();
  }
});

test("fetch reports codec when a finite status carries a bad representation", async () => {
  const stub = stubServer(18469);
  try {
    const base = {
      method: "GET" as const,
      response: sealResponse,
      cases: sealCases,
    };
    for (const [path, reason] of [
      ["/plain", "media_type"],
      ["/html", "media_type"],
      ["/badtag", "variant_tag"],
      ["/duplicate", "duplicate_member"],
      ["/mismatch", "variant_tag"],
    ] as const) {
      const outcome = await fetchJsonAction({ ...base, url: `http://127.0.0.1:18469${path}` });
      expect(outcome.kind).toBe("codec");
      if (outcome.kind !== "codec") throw new Error("wrong outcome");
      expect(outcome.reason).toBe(reason);
    }
    const bounded = await fetchJsonAction({
      ...base,
      url: "http://127.0.0.1:18469/big",
      responseLimit: 16,
    });
    expect(bounded).toMatchObject({ kind: "codec", reason: "byte_limit" });
  } finally {
    stub.stop();
  }
});

test("fetch reports transport and abort distinctly", async () => {
  const refused = await fetchJsonAction({
    method: "GET",
    url: "http://127.0.0.1:1/unroutable",
    response: sealResponse,
    cases: sealCases,
  });
  expect(refused).toEqual({ kind: "transport", phase: "connect" });
  const controller = new AbortController();
  controller.abort();
  const aborted = await fetchJsonAction({
    method: "GET",
    url: "http://127.0.0.1:18468/ok",
    response: sealResponse,
    cases: sealCases,
    signal: controller.signal,
  });
  expect(aborted).toEqual({ kind: "aborted" });
});

test("fetch enforces POST body and GET bodiless preconditions", async () => {
  const body = record("t13::seal_wire", [
    ["label", "inv-1"],
    ["seats", 2n],
  ]);
  await expect(
    fetchJsonAction({
      method: "POST",
      url: "http://127.0.0.1:18470/save",
      request: saveRequest,
      response: sealResponse,
      cases: sealCases,
    }),
  ).rejects.toThrow("POST needs a body value");
  await expect(
    fetchJsonAction({
      method: "GET",
      url: "http://127.0.0.1:18470/load",
      body,
      response: sealResponse,
      cases: sealCases,
    }),
  ).rejects.toThrow("GET carries no body");
  await expect(
    fetchJsonAction({
      method: "GET",
      url: "http://127.0.0.1:18470/load",
      request: saveRequest,
      response: sealResponse,
      cases: sealCases,
    }),
  ).rejects.toThrow("GET carries no request schema");
  const oversize = record("t13::seal_wire", [
    ["label", "p".repeat(8192)],
    ["seats", 2n],
  ]);
  await expect(
    fetchJsonAction({
      method: "POST",
      url: "http://127.0.0.1:18470/save",
      request: saveRequest,
      body: oversize,
      response: sealResponse,
      cases: sealCases,
    }),
  ).rejects.toThrow("exceeds the wire limit");
  const mistyped = record("t13::seal_wire", [
    ["label", 7n],
    ["seats", 2n],
  ]);
  await expect(
    fetchJsonAction({
      method: "POST",
      url: "http://127.0.0.1:18470/save",
      request: saveRequest,
      body: mistyped,
      response: sealResponse,
      cases: sealCases,
    }),
  ).rejects.toThrow("not request-admissible");
  await expect(
    fetchJsonAction({
      method: "GET",
      url: "http://127.0.0.1:18470/load",
      response: sealResponse,
      cases: sealCases,
      responseLimit: 0,
    }),
  ).rejects.toThrow("positive response limit");
});

test("fetch POST consumes a mounted JSON action end to end", async () => {
  const owned = await runOwnedRoot(async () => {
    const token = await serve(18470, [
      [saveEntry, saveHandler],
      [loadEntry, loadHandler],
    ]);
    const saved = await fetchJsonAction({
      method: "POST",
      url: "http://127.0.0.1:18470/invoices/sealed",
      request: saveRequest,
      body: record("t13::seal_wire", [
        ["label", "inv-3"],
        ["seats", 4n],
      ]),
      response: sealResponse,
      cases: sealCases,
    });
    expect(saved.kind).toBe("ok");
    if (saved.kind !== "ok") throw new Error("wrong outcome");
    expect(saved.status).toBe(200);
    expect(saved.leaf).toBe("t13::sealed");
    expect(dataProperty(saved.value, "label")).toBe("inv-3");
    const loaded = await fetchJsonAction({
      method: "GET",
      url: "http://127.0.0.1:18470/invoices/sealed",
      response: sealResponse,
      cases: sealCases,
    });
    expect(loaded.kind).toBe("ok");
    if (loaded.kind !== "ok") throw new Error("wrong outcome");
    expect(dataProperty(loaded.value, "label")).toBe("listed");
    await stop(token);
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});
