import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import {
  createDomainRuntime,
  domainFailureDiagnostics,
  isDomainFailure,
  type FailureShape,
} from "../domain.ts";
import { createTransport, type HTTPTypes } from "../transport/http.ts";
import { createNamedFetch } from "../transport/named.ts";
import { createTypeSafe, type AITypes, type NoulDescriptor } from "../ai/typesafe.ts";
import { runOwnedRoot } from "../owner.ts";
import { success, failure, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { standardFailureKind, standardFailureMessage } from "../failure.ts";
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
  (e) => (e.id >= 1100 && e.id <= 1106) || e.id === 1110 || e.id === 1120 || e.id === 1121,
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
  leaves: errors
    .filter((_, i) => declarations[i].id !== 1106 && declarations[i].id < 1120)
    .map((e) => e.identity),
  inputs: [],
  errors: [],
};
const domain = createDomainRuntime({
  declarations: declarations.map((d) => ({ ...d, parameters: 0 })),
  shapes: [str, int, header, headers, ...errors, detail],
});
const httpTypes = Object.fromEntries(
  ["invalid", "credential", "transport", "timeout", "limit", "status"].map((name, i) => [
    name,
    errors[i].identity,
  ]),
) as unknown as HTTPTypes;
const ids = {
  ...httpTypes,
  header: header.identity,
  invalidData: errors[7].identity,
  invalidQuestion: errors[8].identity,
  invalidAnswer: errors[9].identity,
  failed: errors[6].identity,
};
const origin = { source: "test:provenance", start: 0, end: 0, invocation: [] };
const connection = {
  endpoint: "http://127.0.0.1:1/",
  timeoutMilliseconds: 100,
  maxBodyBytes: 1024,
  headers: [],
};
const request = { path: "/", method: "GET" as const, query: [], headers: [] };
const question: NoulDescriptor = {
  kind: "noul",
  instructions: "Check amount",
  trueDescription: "Yes",
  falseDescription: "No",
  minimum: 0.5,
};
const schema = {
  root: "state",
  nodes: [
    { identity: "state", kind: "record", name: "state", fields: [{ name: "amount", type: "int" }] },
    { identity: "int", kind: "primitive", name: "int" },
  ],
};
const domainDetails = (result: Completion) => {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain failure");
  return domainFailureDiagnostics(result.value);
};

test("native transport failures carry their producing operation", async () => {
  const api = createTransport(domain, { ...httpTypes, header: header.identity }, () => undefined);
  const root = await runOwnedRoot(async () => {
    const failed = await api.request(
      connection,
      request,
      () => success(0n),
      origin,
      "test:provenance/fetch-a",
    );
    const details = domainDetails(failed);
    expect(details.declaration.id).toBe(1102);
    expect(details.provenance).toEqual({
      boundary: "native",
      operation: "test:provenance/fetch-a",
    });
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
});

test("native and authored codec failures with identical payloads stay distinguishable", async () => {
  const api = createNamedFetch(domain, ids, () => undefined);
  const root = await runOwnedRoot(async () => {
    const mapped = domainDetails(
      await api.request(
        connection,
        { ...request, method: "POST" },
        { mode: "text", value: "\ud800" },
        { mode: "text" },
        origin,
        "test:provenance/fetch-b",
      ),
    );
    expect(mapped.declaration.id).toBe(1106);
    expect(mapped.provenance).toEqual({ boundary: "native", operation: "test:provenance/fetch-b" });
    if (!isDomainFailure(mapped.cause)) throw Error("expected private original cause");
    const native = domainFailureDiagnostics(mapped.cause);
    expect(native.declaration.id).toBe(1110);
    expect(native.provenance).toEqual({ boundary: "native", operation: "test:provenance/fetch-b" });
    const authored = domain.create(
      ids.invalidData,
      record(ids.invalidData, [
        ["path", ""],
        ["reason", "unicode_scalar"],
      ]),
      origin,
    );
    const authoredDetails = domainFailureDiagnostics(authored);
    expect(authoredDetails.declaration.id).toBe(1110);
    expect(JSON.stringify(authoredDetails.payload)).toBe(JSON.stringify(native.payload));
    expect(authoredDetails.provenance).toEqual({ boundary: "emitted", operation: "" });
    const key = (boundary: string, name: string) => `${boundary}:${name}`;
    expect(key(native.provenance.boundary, native.declaration.name)).not.toBe(
      key(authoredDetails.provenance.boundary, authoredDetails.declaration.name),
    );
    expect(native.declaration.name).toBe(authoredDetails.declaration.name);
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
});

test("judge validation keeps emitted provenance while judge decoding stays native", async () => {
  const api = createTypeSafe(domain, ids as unknown as AITypes, () => undefined);
  const state = record("state", [["amount", 9007199254740993n]]);
  const invalid = domainDetails(
    await api.ask(
      connection,
      "jev-latest",
      schema,
      state,
      [question, { ...question, instructions: "" }],
      origin,
      "test:provenance/judge",
    ),
  );
  expect(invalid.declaration.id).toBe(1120);
  expect(invalid.provenance).toEqual({ boundary: "emitted", operation: "test:provenance/judge" });
  const mapped = domainDetails(
    await api.ask(
      connection,
      "jev-latest",
      schema,
      record("state", [["amount", 1]]),
      [question],
      origin,
      "test:provenance/judge",
    ),
  );
  expect(mapped.declaration.id).toBe(1106);
  expect(mapped.provenance).toEqual({ boundary: "native", operation: "test:provenance/judge" });
  if (!isDomainFailure(mapped.cause)) throw Error("expected private original cause");
  const undecodable = domainFailureDiagnostics(mapped.cause);
  expect(undecodable.declaration.id).toBe(1110);
  expect(undecodable.provenance).toEqual({
    boundary: "native",
    operation: "test:provenance/judge",
  });
});

test("unknown native exceptions keep the standard channel and are never relabelled", async () => {
  const api = createTransport(domain, { ...httpTypes, header: header.identity }, () => undefined);
  const root = await runOwnedRoot(async () => {
    const defect = await api.request(
      connection,
      {
        ...request,
        exchange: async () => {
          throw Error("adapter defect");
        },
      },
      () => success(0n),
      origin,
      "test:provenance/fetch-c",
    );
    expect(defect.kind).toBe("standard");
    if (defect.kind !== "standard") throw Error("expected standard failure");
    expect(standardFailureKind(defect.value)).toBe("native_exception");
    expect(standardFailureMessage(defect.value)).toContain("adapter defect");
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
});

test("cause and occurrence identity survive forwarding while equal reconstructions differ", () => {
  const cause = domain.create(
    ids.transport,
    record(ids.transport, [["phase", "connect"]]),
    origin,
    undefined,
    { boundary: "native", operation: "test:provenance/fetch-d" },
  );
  const first = domainFailureDiagnostics(cause);
  const forwarded = failure(cause);
  expect(forwarded.kind).toBe("domain");
  if (forwarded.kind !== "domain") throw Error("expected domain failure");
  expect(domainFailureDiagnostics(forwarded.value).occurrenceID).toBe(first.occurrenceID);
  const same = domain.create(
    ids.transport,
    record(ids.transport, [["phase", "connect"]]),
    origin,
    undefined,
    { boundary: "native", operation: "test:provenance/fetch-d" },
  );
  expect(domainFailureDiagnostics(same).occurrenceID).not.toBe(first.occurrenceID);
  expect(same).not.toBe(cause);
  const wrapped = domain.create(
    ids.invalid,
    record(ids.invalid, [["reason", "url"]]),
    origin,
    cause,
    { boundary: "native", operation: "test:provenance/fetch-d" },
  );
  expect(domainFailureDiagnostics(wrapped).cause).toBe(cause);
});

test("provenance validation rejects unattributed native and malformed tags", () => {
  const payload = record(ids.transport, [["phase", "connect"]]);
  expect(
    domainFailureDiagnostics(domain.create(ids.transport, payload, origin)).provenance,
  ).toEqual({ boundary: "emitted", operation: "" });
  for (const provenance of [
    { boundary: "native", operation: "" },
    { boundary: "bogus", operation: "x" },
    { boundary: "native", operation: 7 },
    "native",
    null,
    42,
  ]) {
    expect(() => domain.create(ids.transport, payload, origin, undefined, provenance)).toThrow(
      TypeError,
    );
  }
});
