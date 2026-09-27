import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, type FailureShape } from "../domain.ts";
import { errorType, errorPayload, type Completion } from "../completion.ts";
import type { Schema } from "../codec/json.ts";
import {
  fetchJsonAction,
  createJsonActionFetch,
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

const PIN = "f".repeat(64);
const OTHER = "0".repeat(64);
const MISMATCH = `{"schemaVersion":1,"kind":"can.generation-mismatch","serverGeneration":"${OTHER}"}`;

const strNode = { identity: "c06::str", kind: "primitive", name: "str" };
const outcomeSchema: Schema = {
  root: "c06::seal_outcome",
  nodes: [
    {
      identity: "c06::seal_outcome",
      kind: "variant",
      name: "seal_outcome",
      leaves: ["c06::sealed", "c06::contested"],
    },
    {
      identity: "c06::sealed",
      kind: "record",
      name: "sealed",
      fields: [{ name: "label", type: "c06::str" }],
    },
    {
      identity: "c06::contested",
      kind: "record",
      name: "contested",
      fields: [{ name: "reason", type: "c06::str" }],
    },
    strNode,
  ],
};
// Wire spelling mirrors the adapter codec: {"case": leaf name, "value": fields}.
const contestedBody = JSON.stringify({
  case: "contested",
  value: { reason: "stale" },
});

function checkFailure(result: Completion<unknown>, declaration: string, payload: object) {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw new Error("expected domain failure");
  expect(errorType(result)).toBe(id(declaration));
  expect(errorPayload(result)).toMatchObject(payload);
}

function withDocument<T>(slot: string | null | undefined, run: () => Promise<T>): Promise<T> {
  const holder = globalThis as Record<string, unknown>;
  const had = Object.hasOwn(holder, "document");
  const prior = holder["document"];
  if (slot === undefined) {
    delete holder["document"];
  } else {
    holder["document"] = {
      querySelector: (selectors: string) => {
        expect(selectors).toBe("script[data-can-generation]");
        return slot === null
          ? null
          : { getAttribute: (name: string) => (name === "data-can-generation" ? slot : null) };
      },
    };
  }
  try {
    return run();
  } finally {
    if (had) holder["document"] = prior;
    else delete holder["document"];
  }
}

async function withForwardingFetch<T>(
  port: number,
  seen: { url: string; generation: string | null }[],
  run: () => Promise<T>,
): Promise<T> {
  const realFetch = globalThis.fetch;
  globalThis.fetch = (async (url: unknown, init?: unknown) => {
    expect(typeof url).toBe("string");
    const target = url as string;
    expect(target.startsWith("/") && !target.startsWith("//")).toBe(true);
    const headers = new Headers((init as { headers?: HeadersInit }).headers);
    seen.push({ url: target, generation: headers.get("can-generation") });
    return realFetch(`http://127.0.0.1:${port}${target}`, init as never);
  }) as unknown as typeof fetch;
  try {
    return await run();
  } finally {
    globalThis.fetch = realFetch;
  }
}

test("fetch sends the page slot and only the page slot", async () => {
  const seen: (string | null)[] = [];
  const stub = Bun.serve({
    hostname: "127.0.0.1",
    port: 18771,
    fetch: (native) => {
      seen.push(native.headers.get("can-generation"));
      return new Response("plain", { status: 200 });
    },
  });
  try {
    const call = () =>
      fetchJsonAction({
        url: "http://127.0.0.1:18771/echo",
        method: "GET",
        response: outcomeSchema,
        cases: [{ leaf: "c06::sealed", status: 200 }],
      });
    // The echo body is not JSON; every call classifies codec. Only the
    // observed header matters here.
    await withDocument(PIN, async () => expect((await call()).kind).toBe("codec"));
    await withDocument(null, async () => expect((await call()).kind).toBe("codec"));
    await withDocument("nope", async () => expect((await call()).kind).toBe("codec"));
    await withDocument(undefined, async () => expect((await call()).kind).toBe("codec"));
    expect(seen).toEqual([PIN, null, null, null]);
  } finally {
    await stub.stop(true);
  }
});

test("exact 409 refusals map to generation_mismatch ahead of domain cases", async () => {
  const stub = Bun.serve({
    hostname: "127.0.0.1",
    port: 18772,
    fetch: (native) => {
      const path = new URL(native.url).pathname;
      if (path === "/mismatch")
        return new Response(MISMATCH, {
          status: 409,
          headers: { "content-type": "application/json" },
        });
      if (path === "/conflict")
        return new Response(contestedBody, {
          status: 409,
          headers: { "content-type": "application/json" },
        });
      return new Response("Conflict", { status: 409 });
    },
  });
  try {
    const declared = [{ leaf: "c06::sealed", status: 200 }];
    const withConflict = [...declared, { leaf: "c06::contested", status: 409 }];
    await withDocument(undefined, async () => {
      // Exact shape maps even where 409 is undeclared ...
      const refused = await fetchJsonAction({
        url: "http://127.0.0.1:18772/mismatch",
        method: "GET",
        response: outcomeSchema,
        cases: declared,
      });
      expect(refused).toEqual({ kind: "generation_mismatch", serverGeneration: OTHER });
      // ... and ahead of a declared 409 domain case.
      const ahead = await fetchJsonAction({
        url: "http://127.0.0.1:18772/mismatch",
        method: "GET",
        response: outcomeSchema,
        cases: withConflict,
      });
      expect(ahead).toEqual({ kind: "generation_mismatch", serverGeneration: OTHER });
      // A domain 409 keeps its domain path.
      const domain = await fetchJsonAction({
        url: "http://127.0.0.1:18772/conflict",
        method: "GET",
        response: outcomeSchema,
        cases: withConflict,
      });
      expect(domain.kind).toBe("ok");
      if (domain.kind !== "ok") throw new Error("domain 409 misclassified");
      expect(domain.leaf).toBe("c06::contested");
      // A non-shape 409 without a declared case stays unexpected.
      const unknown = await fetchJsonAction({
        url: "http://127.0.0.1:18772/foreign",
        method: "GET",
        response: outcomeSchema,
        cases: declared,
      });
      expect(unknown).toEqual({ kind: "unexpected_status", status: 409 });
      // A non-shape 409 against a declared case fails its contract.
      const broken = await fetchJsonAction({
        url: "http://127.0.0.1:18772/foreign",
        method: "GET",
        response: outcomeSchema,
        cases: withConflict,
      });
      expect(broken.kind).toBe("codec");
    });
  } finally {
    await stub.stop(true);
  }
});

test("over-limit 409 bodies keep ordinary classification", async () => {
  const stub = Bun.serve({
    hostname: "127.0.0.1",
    port: 18773,
    fetch: () =>
      new Response("x".repeat(300), {
        status: 409,
        headers: { "content-type": "application/json" },
      }),
  });
  try {
    await withDocument(undefined, async () => {
      const declared = await fetchJsonAction({
        url: "http://127.0.0.1:18773/big",
        method: "GET",
        response: outcomeSchema,
        cases: [
          { leaf: "c06::sealed", status: 200 },
          { leaf: "c06::contested", status: 409 },
        ],
        responseLimit: 64,
      });
      expect(declared).toEqual({ kind: "codec", path: "", reason: "byte_limit" });
      const undeclared = await fetchJsonAction({
        url: "http://127.0.0.1:18773/big",
        method: "GET",
        response: outcomeSchema,
        cases: [{ leaf: "c06::sealed", status: 200 }],
        responseLimit: 64,
      });
      expect(undeclared).toEqual({ kind: "unexpected_status", status: 409 });
    });
  } finally {
    await stub.stop(true);
  }
});

test("the adapter lowers refusals to the generation transport phase", async () => {
  const seen: { url: string; generation: string | null }[] = [];
  const site: JsonFetchSite = {
    action: "c06::load_seal",
    method: "GET",
    path: "/sealed",
    captures: [],
    response: outcomeSchema,
    cases: [{ leaf: "c06::sealed", status: 200 }],
  };
  const stub = Bun.serve({
    hostname: "127.0.0.1",
    port: 18774,
    fetch: () =>
      new Response(MISMATCH, { status: 409, headers: { "content-type": "application/json" } }),
  });
  try {
    await withForwardingFetch(18774, seen, async () => {
      await withDocument(PIN, async () => {
        const refused = await api.get("c06::load_seal", site, undefined);
        checkFailure(refused, "can.std.http@1::transport_failed", { phase: "generation" });
      });
    });
    expect(seen).toEqual([{ url: "/sealed", generation: PIN }]);
  } finally {
    await stub.stop(true);
  }
});
