import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createTransport, type HTTPTypes } from "../transport/http.ts";
import { runOwnedRoot } from "../owner.ts";
import { success, type Completion } from "../completion.ts";
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
const declarations = catalogue.errors.filter((e) =>
  [
    "http::invalid_request",
    "http::credentials_missing",
    "http::transport_failed",
    "http::timeout",
    "http::body_limit",
    "http::status_error",
  ].includes(e.name),
);
const errors = declarations.map((d) =>
  shape(
    "error",
    d.identity,
    d.fields.map((f) => ({
      name: f.name,
      type: f.type === "str" ? str.identity : f.type === "int" ? int.identity : headers.identity,
    })),
  ),
);
const domain = createDomainRuntime({
  declarations: declarations.map((d) => ({ ...d, parameters: 0 })),
  shapes: [str, int, header, headers, ...errors],
});
const types = Object.fromEntries(
  ["invalid", "credential", "transport", "timeout", "limit", "status"].map((name, i) => [
    name,
    errors[i].identity,
  ]),
) as unknown as HTTPTypes;
const api = createTransport(domain, { ...types, header: header.identity }, () => undefined);
const origin = { source: "test:http", start: 0, end: 0, invocation: [] };
const connection = {
  endpoint: "http://127.0.0.1:1/",
  timeoutMilliseconds: 100,
  maxBodyBytes: 3,
  headers: [],
};
const request = { path: "/", method: "GET" as const, query: [], headers: [] };
function check(result: Completion, name: string, payload: object) {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw new Error("expected domain");
  const d = domainFailureDiagnostics(result.value);
  expect(d.declaration.name).toBe(name);
  expect(d.payload).toMatchObject(payload);
}
test("transport failures construct exact nominal catalogue occurrences", async () => {
  const root = await runOwnedRoot(async () => {
    check(
      await api.request(
        connection,
        { ...request, path: "https://other.test" },
        () => success(0n),
        origin,
        "test:http/transport",
      ),
      "http::invalid_request",
      { reason: "origin" },
    );
    check(
      await api.request(
        { ...connection, bearerEnvironment: "TOKEN" },
        request,
        () => success(0n),
        origin,
        "test:http/transport",
      ),
      "http::credentials_missing",
      { variable: "TOKEN" },
    );
    check(
      await api.request(
        connection,
        { ...request, method: "POST", body: new Uint8Array(4) },
        () => success(0n),
        origin,
        "test:http/transport",
      ),
      "http::body_limit",
      { limit: 3n },
    );
    check(
      await api.request(connection, request, () => success(0n), origin, "test:http/transport"),
      "http::transport_failed",
      { phase: "connect" },
    );
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
});
test("status and timeout use typed payloads while unexpected decoder faults stay standard", async () => {
  const server = Bun.serve({
    hostname: "127.0.0.1",
    port: 0,
    fetch(req) {
      return new Response("ok", {
        status: new URL(req.url).pathname === "/bad" ? 401 : 200,
        headers: { "x-test": "yes" },
      });
    },
  });
  try {
    const root = await runOwnedRoot(async () => {
      const c = { ...connection, endpoint: server.url.href };
      check(
        await api.request(
          c,
          { ...request, path: "/bad" },
          () => success(0n),
          origin,
          "test:http/transport",
        ),
        "http::status_error",
        { status: 401n },
      );
      check(
        await api.request(
          { ...c, timeoutMilliseconds: 5 },
          request,
          () => {
            const start = performance.now();
            while (performance.now() - start < 10) {}
            return success(0n);
          },
          origin,
          "test:http/transport",
        ),
        "http::timeout",
        { timeout_ms: 5n },
      );
      const unexpected = await api.request(
        c,
        request,
        () => {
          throw new TypeError("decoder defect");
        },
        origin,
        "test:http/transport",
      );
      expect(unexpected.kind).toBe("standard");
      return success(undefined);
    });
    expect(root.completion.kind).toBe("ok");
  } finally {
    await server.stop(true);
  }
});
