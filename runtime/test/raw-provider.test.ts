import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import {
  createDomainRuntime,
  domainFailureDiagnostics,
  isDomainFailure,
  type FailureShape,
} from "../domain.ts";
import { type HTTPTypes } from "../transport/http.ts";
import { createNamedFetch } from "../transport/named.ts";
import { runOwnedRoot } from "../owner.ts";
import { success, value, type Completion } from "../completion.ts";
import { record, recordIdentity, dataProperty } from "../data.ts";
import { assertionContext, contextReport, closeContext } from "../assert/context.ts";
import { provideRawHTTP, providerHTTP, rawEnvironment } from "../assert/provider.ts";
import { runAssertion } from "../assert/runner.ts";
import { assertionEqual, completionValueEqual } from "../assert/runner.ts";
import { ownBytes } from "../bytes.ts";
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
const declarations = catalogue.errors.filter(
  (e) => (e.id >= 1100 && e.id <= 1106) || e.id === 1110,
);
const detailIdentity = hash(["variant", "can.std.http@1::failure_detail"]);
const errors = declarations.map((d) =>
  shape(
    "error",
    d.identity,
    d.fields.map((f) => ({
      name: f.name,
      type:
        f.type === "str"
          ? str.identity
          : f.type === "int"
            ? int.identity
            : f.type === "http::failure_detail"
              ? detailIdentity
              : headers.identity,
    })),
  ),
);
const detail: FailureShape = {
  identity: detailIdentity,
  kind: "variant",
  declaration: "can.std.http@1::failure_detail",
  fields: [],
  arguments: [],
  leaves: errors.filter((_, i) => declarations[i].id !== 1106).map((e) => e.identity),
  inputs: [],
  errors: [],
};
const domain = createDomainRuntime({
  declarations: declarations.map((d) => ({ ...d, parameters: 0 })),
  shapes: [str, int, header, headers, ...errors, detail],
});
const types = Object.fromEntries(
  ["invalid", "credential", "transport", "timeout", "limit", "status"].map((name, i) => [
    name,
    errors[i].identity,
  ]),
) as unknown as HTTPTypes;
const ids = {
  ...types,
  header: header.identity,
  invalidData: errors[7].identity,
  failed: errors[6].identity,
};
const origin = { source: "test:raw-provider", start: 0, end: 0, invocation: [] };
const operation = "test:raw/fetch";
const responseSchema = {
  root: "sample",
  nodes: [
    {
      identity: "sample",
      kind: "record",
      name: "sample",
      fields: [{ name: "count", type: "int" }],
    },
    { identity: "int", kind: "primitive", name: "int" },
  ],
};
const base64 = (text: string) => Buffer.from(text, "utf-8").toString("base64");
function checkDetail(result: Completion, leaf: string, payload: object) {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw new Error("expected domain");
  const d = domainFailureDiagnostics(result.value);
  expect(d.declaration.id).toBe(1106);
  const leafValue = dataProperty(d.payload, "detail");
  expect(recordIdentity(leafValue)).toBe(leaf);
  expect(leafValue).toMatchObject(payload);
  if (!isDomainFailure(d.cause)) throw new Error("expected private original cause");
}

test("raw response compares the exact request and runs the real decoder", async () => {
  const context = assertionContext({ package: "app", declaration: "app::load", name: "decoded" });
  provideRawHTTP(context, operation, {
    environment: {},
    exchange: {
      request: {
        method: "GET",
        url: "http://127.0.0.1:1/receipt?labels=a+b&labels=%2B",
        headers: [["x-probe", "yes"]],
        body: { bytes_base64: "" },
      },
      outcome: {
        response: {
          status: 200,
          headers: [["content-type", "application/json"]],
          body_base64: base64('{"count":9007199254740993}'),
        },
      },
    },
  });
  const root = await runOwnedRoot(async () => {
    const api = createNamedFetch(domain, ids, () => {
      throw new Error("ambient credential read");
    });
    const c = {
      endpoint: "http://127.0.0.1:1/",
      timeoutMilliseconds: 1000,
      maxBodyBytes: 1024,
      headers: [],
    };
    const result = await api.request<any>(
      c,
      {
        path: "/receipt",
        method: "GET",
        query: [{ name: "labels", value: ["a b", "+"] }],
        headers: [{ name: "x_probe", value: "yes" }],
      },
      undefined,
      { mode: "json", schema: responseSchema },
      origin,
      operation,
      context,
    );
    expect(value(result).count).toBe(9007199254740993n);
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
  closeContext(context);
  const report = contextReport(context);
  expect(report.violations).toEqual([]);
  expect(report.evidence).toContain("raw-provider-fixture");
});

test("raw request mismatches fail before any outcome is released", async () => {
  const mutations: Array<
    (request: {
      method: string;
      url: string;
      headers: [string, string][];
      body: { bytes_base64: string };
    }) => void
  > = [
    (request) => {
      request.method = "POST";
    },
    (request) => {
      request.url = "http://127.0.0.1:1/other";
    },
    (request) => {
      request.headers = [["x-probe", "no"]];
    },
    (request) => {
      request.body = { bytes_base64: base64("x") };
    },
  ];
  for (const mutate of mutations) {
    const context = assertionContext({
      package: "app",
      declaration: "app::load",
      name: "mismatch",
    });
    const request = {
      method: "GET",
      url: "http://127.0.0.1:1/receipt",
      headers: [["x-probe", "yes"]] as [string, string][],
      body: { bytes_base64: "" },
    };
    mutate(request);
    provideRawHTTP(context, operation, {
      environment: {},
      exchange: {
        request,
        outcome: {
          response: {
            status: 200,
            headers: [["content-type", "text/plain"]],
            body_base64: base64("ok"),
          },
        },
      },
    });
    const root = await runOwnedRoot(async () => {
      const api = createNamedFetch(domain, ids, () => undefined);
      const c = {
        endpoint: "http://127.0.0.1:1/",
        timeoutMilliseconds: 1000,
        maxBodyBytes: 1024,
        headers: [],
      };
      const result = await api.request<string>(
        c,
        {
          path: "/receipt",
          method: "GET",
          query: [],
          headers: [{ name: "x_probe", value: "yes" }],
        },
        undefined,
        { mode: "text" },
        origin,
        operation,
        context,
      );
      expect(result.kind).toBe("standard");
      return success(undefined);
    });
    expect(root.completion.kind).toBe("ok");
    closeContext(context);
    expect(contextReport(context).violations).toContain("argument mismatch");
  }
});

test("raw json bodies compare exact number tokens ignoring member order", async () => {
  const run = async (expected: string, actual: unknown, schema: any, ok: boolean) => {
    const context = assertionContext({ package: "app", declaration: "app::send", name: "json" });
    provideRawHTTP(context, operation, {
      environment: {},
      exchange: {
        request: {
          method: "POST",
          url: "http://127.0.0.1:1/json",
          headers: [["content-type", "application/json"]],
          body: { json_utf8: expected },
        },
        outcome: {
          response: {
            status: 200,
            headers: [["content-type", "text/plain"]],
            body_base64: base64("ok"),
          },
        },
      },
    });
    const root = await runOwnedRoot(async () => {
      const api = createNamedFetch(domain, ids, () => undefined);
      const c = {
        endpoint: "http://127.0.0.1:1/",
        timeoutMilliseconds: 1000,
        maxBodyBytes: 1024,
        headers: [],
      };
      const result = await api.request<string>(
        c,
        { path: "/json", method: "POST", query: [], headers: [] },
        { mode: "json", value: actual, schema },
        { mode: "text" },
        origin,
        operation,
        context,
      );
      expect(result.kind).toBe(ok ? "ok" : "standard");
      return success(undefined);
    });
    expect(root.completion.kind).toBe("ok");
    closeContext(context);
    return contextReport(context).violations;
  };
  expect(
    await run(
      '{"count":9007199254740993}',
      record("sample", [["count", 9007199254740993n]]),
      responseSchema,
      true,
    ),
  ).toEqual([]);
  expect(
    await run('{"count":1.0}', record("sample", [["count", 1n]]), responseSchema, false),
  ).toContain("argument mismatch");
  expect(
    await run('{"count":1,"extra":true}', record("sample", [["count", 1n]]), responseSchema, false),
  ).toContain("argument mismatch");
  const context = assertionContext({ package: "app", declaration: "app::send", name: "duplicate" });
  provideRawHTTP(context, operation, {
    environment: {},
    exchange: {
      request: {
        method: "POST",
        url: "http://127.0.0.1:1/json",
        headers: [["content-type", "application/json"]],
        body: { json_utf8: '{"a":1,"a":2}' },
      },
      outcome: { response: { status: 200, headers: [], body_base64: base64("ok") } },
    },
  });
  const root = await runOwnedRoot(async () => {
    const api = createNamedFetch(domain, ids, () => undefined);
    const c = {
      endpoint: "http://127.0.0.1:1/",
      timeoutMilliseconds: 1000,
      maxBodyBytes: 1024,
      headers: [],
    };
    const result = await api.request<string>(
      c,
      { path: "/json", method: "POST", query: [], headers: [] },
      { mode: "json", value: record("sample", [["count", 1n]]), schema: responseSchema },
      { mode: "text" },
      origin,
      operation,
      context,
    );
    expect(result.kind).toBe("standard");
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
  closeContext(context);
  expect(contextReport(context).violations).toContain("malformed fixture");
});

test("raw failure outcomes execute the real normalization path", async () => {
  const run = async (outcome: { failure: any }, leaf: string, payload: object) => {
    const context = assertionContext({ package: "app", declaration: "app::load", name: "failure" });
    provideRawHTTP(context, operation, {
      environment: {},
      exchange: {
        request: {
          method: "GET",
          url: "http://127.0.0.1:1/down",
          headers: [],
          body: { bytes_base64: "" },
        },
        outcome,
      },
    });
    const root = await runOwnedRoot(async () => {
      const api = createNamedFetch(domain, ids, () => undefined);
      const c = {
        endpoint: "http://127.0.0.1:1/",
        timeoutMilliseconds: 1000,
        maxBodyBytes: 1024,
        headers: [],
      };
      checkDetail(
        await api.request<string>(
          c,
          { path: "/down", method: "GET", query: [], headers: [] },
          undefined,
          { mode: "text" },
          origin,
          operation,
          context,
        ),
        leaf,
        payload,
      );
      return success(undefined);
    });
    expect(root.completion.kind).toBe("ok");
    closeContext(context);
    const report = contextReport(context);
    expect(report.violations).toEqual([]);
    expect(report.evidence).toContain("raw-provider-fixture");
  };
  await run({ failure: { kind: "transport", phase: "connect" } }, errors[2].identity, {
    phase: "connect",
  });
  await run({ failure: { kind: "timeout" } }, errors[3].identity, { timeout_ms: 1000n });
  await run({ failure: { kind: "body_limit" } }, errors[4].identity, { limit: 1024n });
});

test("raw environment supplies fake credentials and absence never leaks ambient", async () => {
  const context = assertionContext({
    package: "app",
    declaration: "app::load",
    name: "credential",
  });
  provideRawHTTP(context, operation, {
    environment: { TOKEN: "fake-secret" },
    exchange: {
      request: {
        method: "GET",
        url: "http://127.0.0.1:1/private",
        headers: [["authorization", "Bearer fake-secret"]],
        body: { bytes_base64: "" },
      },
      outcome: {
        response: {
          status: 200,
          headers: [["content-type", "text/plain"]],
          body_base64: base64("ok"),
        },
      },
    },
  });
  const root = await runOwnedRoot(async () => {
    const api = createNamedFetch(domain, ids, () => {
      throw new Error("ambient credential read");
    });
    const c = {
      endpoint: "http://127.0.0.1:1/",
      timeoutMilliseconds: 1000,
      maxBodyBytes: 1024,
      headers: [],
      bearerEnvironment: "TOKEN",
    };
    expect(rawEnvironment(context, operation)!("TOKEN")).toBe("fake-secret");
    expect(rawEnvironment(context, operation)!("OTHER")).toBe(undefined);
    expect(
      value(
        await api.request<string>(
          c,
          { path: "/private", method: "GET", query: [], headers: [] },
          undefined,
          { mode: "text" },
          origin,
          operation,
          context,
        ),
      ),
    ).toBe("ok");
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
  closeContext(context);
  expect(contextReport(context).violations).toEqual([]);
  const missing = assertionContext({ package: "app", declaration: "app::load", name: "absent" });
  provideRawHTTP(missing, operation, {
    environment: {},
    exchange: {
      request: {
        method: "GET",
        url: "http://127.0.0.1:1/private",
        headers: [],
        body: { bytes_base64: "" },
      },
      outcome: { response: { status: 200, headers: [], body_base64: base64("ok") } },
    },
  });
  const absent = await runOwnedRoot(async () => {
    const api = createNamedFetch(domain, ids, () => "live-secret");
    const c = {
      endpoint: "http://127.0.0.1:1/",
      timeoutMilliseconds: 1000,
      maxBodyBytes: 1024,
      headers: [],
      bearerEnvironment: "TOKEN",
    };
    checkDetail(
      await api.request<string>(
        c,
        { path: "/private", method: "GET", query: [], headers: [] },
        undefined,
        { mode: "text" },
        origin,
        operation,
        missing,
      ),
      errors[1].identity,
      { variable: "TOKEN" },
    );
    return success(undefined);
  });
  expect(absent.completion.kind).toBe("ok");
  closeContext(missing);
  expect(contextReport(missing).violations).toContain("unused fixture");
});

test("null exchange forbids transport but admits pretransport failures", async () => {
  const context = assertionContext({ package: "app", declaration: "app::load", name: "null" });
  provideRawHTTP(context, operation, { environment: {}, exchange: null });
  const root = await runOwnedRoot(async () => {
    const api = createNamedFetch(domain, ids, () => undefined);
    const c = {
      endpoint: "http://127.0.0.1:1/",
      timeoutMilliseconds: 1000,
      maxBodyBytes: 1024,
      headers: [],
      bearerEnvironment: "TOKEN",
    };
    checkDetail(
      await api.request<string>(
        c,
        { path: "/x", method: "GET", query: [], headers: [] },
        undefined,
        { mode: "text" },
        origin,
        operation,
        context,
      ),
      errors[1].identity,
      { variable: "TOKEN" },
    );
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
  closeContext(context);
  expect(contextReport(context).violations).toEqual([]);
  const attempt = assertionContext({ package: "app", declaration: "app::load", name: "attempt" });
  provideRawHTTP(attempt, operation, { environment: { TOKEN: "fake" }, exchange: null });
  const blocked = await runOwnedRoot(async () => {
    const api = createNamedFetch(domain, ids, () => undefined);
    const c = {
      endpoint: "http://127.0.0.1:1/",
      timeoutMilliseconds: 1000,
      maxBodyBytes: 1024,
      headers: [],
      bearerEnvironment: "TOKEN",
    };
    const result = await api.request<string>(
      c,
      { path: "/x", method: "GET", query: [], headers: [] },
      undefined,
      { mode: "text" },
      origin,
      operation,
      attempt,
    );
    expect(result.kind).toBe("standard");
    return success(undefined);
  });
  expect(blocked.completion.kind).toBe("ok");
  closeContext(attempt);
  expect(contextReport(attempt).violations).toContain("unexpected live boundary");
});

test("raw fixtures are owned per operation and consumed exactly once", async () => {
  const context = assertionContext({ package: "app", declaration: "app::load", name: "owned" });
  provideRawHTTP(context, operation, {
    environment: {},
    exchange: {
      request: {
        method: "GET",
        url: "http://127.0.0.1:1/one",
        headers: [],
        body: { bytes_base64: "" },
      },
      outcome: {
        response: {
          status: 200,
          headers: [["content-type", "text/plain"]],
          body_base64: base64("ok"),
        },
      },
    },
  });
  expect(providerHTTP(context, origin, "test:raw/other", 1024)).toBe(undefined);
  expect(rawEnvironment(context, "test:raw/other")).toBe(undefined);
  expect(() => provideRawHTTP(context, operation, { environment: {}, exchange: null })).toThrow();
  const root = await runOwnedRoot(async () => {
    const api = createNamedFetch(domain, ids, () => undefined);
    const c = {
      endpoint: "http://127.0.0.1:1/",
      timeoutMilliseconds: 1000,
      maxBodyBytes: 1024,
      headers: [],
    };
    expect(
      value(
        await api.request<string>(
          c,
          { path: "/one", method: "GET", query: [], headers: [] },
          undefined,
          { mode: "text" },
          origin,
          operation,
          context,
        ),
      ),
    ).toBe("ok");
    const again = await api.request<string>(
      c,
      { path: "/one", method: "GET", query: [], headers: [] },
      undefined,
      { mode: "text" },
      origin,
      operation,
      context,
    );
    expect(again.kind).toBe("standard");
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
  closeContext(context);
  expect(contextReport(context).violations).toContain("missing fixture");
});

test("completion comparison matches genuine bytes by content without forging", async () => {
  const left = ownBytes(new Uint8Array([0, 255, 1])),
    right = ownBytes(new Uint8Array([0, 255, 1])),
    other = ownBytes(new Uint8Array([0, 255, 2]));
  expect(assertionEqual(left, right)).toBe(false);
  expect(completionValueEqual(left, right)).toBe(true);
  expect(completionValueEqual(left, other)).toBe(false);
  expect(completionValueEqual(left, [0, 255, 1])).toBe(false);
  expect(
    completionValueEqual(record("sample", [["count", 1n]]), record("sample", [["count", 1n]])),
  ).toBe(true);
  const report = await runAssertion({
    root: { package: "app", declaration: "app::load", name: "bytes" },
    actual: async () => success(left),
    expected: async () => success(right),
  });
  expect(report.passed).toBe(true);
});

test("malformed raw registrations fail fast without consuming the boundary", async () => {
  const context = assertionContext({ package: "app", declaration: "app::load", name: "malformed" });
  const bad = [
    {
      environment: {},
      exchange: {
        request: {
          method: "GET",
          url: "http://127.0.0.1:1/",
          headers: [],
          body: { bytes_base64: "!!!" },
        },
        outcome: { response: { status: 200, headers: [], body_base64: "" } },
      },
    },
    {
      environment: {},
      exchange: {
        request: {
          method: "GET",
          url: "http://127.0.0.1:1/",
          headers: [],
          body: { bytes_base64: "", json_utf8: "{}" },
        },
        outcome: { response: { status: 200, headers: [], body_base64: "" } },
      },
    },
    {
      environment: {},
      exchange: {
        request: {
          method: "GET",
          url: "http://127.0.0.1:1/",
          headers: [],
          body: { bytes_base64: "" },
        },
        outcome: { response: { status: 99, headers: [], body_base64: "" } },
      },
    },
    {
      environment: {},
      exchange: {
        request: {
          method: "GET",
          url: "http://127.0.0.1:1/",
          headers: [],
          body: { bytes_base64: "" },
        },
        outcome: { failure: { kind: "transport", phase: "warp" } },
      },
    },
  ] as any[];
  for (const spec of bad)
    expect(() => provideRawHTTP(context, operation + bad.indexOf(spec), spec)).toThrow();
  closeContext(context);
  expect(contextReport(context).violations).toContain("malformed fixture");
});
