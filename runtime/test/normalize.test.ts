import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import {
  createDomainRuntime,
  domainFailureDiagnostics,
  isDomainFailure,
  type FailureShape,
} from "../domain.ts";
import { createNormalizer } from "../transport/normalize.ts";
import { success, failure, caught } from "../completion.ts";
import { record, recordIdentity, dataProperty, array } from "../data.ts";
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
    "http::request_failed",
    "codec::invalid_data",
  ].includes(e.name),
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
    .filter((_, i) => declarations[i].name !== "http::request_failed")
    .map((e) => e.identity),
  inputs: [],
  errors: [],
};
const domain = createDomainRuntime({
  declarations: declarations.map((d) => ({ ...d, parameters: 0 })),
  shapes: [str, int, header, headers, ...errors, detail],
});
const byName = (name: string) => errors[declarations.findIndex((d) => d.name === name)].identity;
const origin = { source: "test:normalize", start: 0, end: 0, invocation: [] };
const operation = "test:normalize/fetch";
const normalize = createNormalizer(domain, {
  failed: byName("http::request_failed"),
  leaves: [
    "http::invalid_request",
    "http::credentials_missing",
    "http::transport_failed",
    "http::timeout",
    "http::body_limit",
    "http::status_error",
    "codec::invalid_data",
  ].map(byName),
});
const leafPayloads: ReadonlyArray<readonly [string, ReadonlyArray<readonly [string, unknown]>]> = [
  ["http::invalid_request", [["reason", "url"]]],
  ["http::credentials_missing", [["variable", "TOKEN"]]],
  ["http::transport_failed", [["phase", "connect"]]],
  ["http::timeout", [["timeout_ms", 5n]]],
  ["http::body_limit", [["limit", 3n]]],
  [
    "http::status_error",
    [
      ["status", 418n],
      [
        "headers",
        array([
          record(header.identity, [
            ["name", "content-type"],
            ["value", "text/plain"],
          ]),
        ]),
      ],
    ],
  ],
  [
    "codec::invalid_data",
    [
      ["path", "/0"],
      ["reason", "invalid_json"],
    ],
  ],
];

test("all seven native leaves map to request_failed with exact Detail and private cause", () => {
  for (const [name, fields] of leafPayloads) {
    const token = domain.create(byName(name), record(byName(name), fields), origin, undefined, {
      boundary: "native",
      operation,
    });
    const mapped = normalize.map(failure(token), operation);
    expect(mapped.kind).toBe("domain");
    if (mapped.kind !== "domain") throw Error("expected domain failure");
    const details = domainFailureDiagnostics(mapped.value);
    expect(details.declaration.name).toBe("http::request_failed");
    expect(details.provenance).toEqual({ boundary: "native", operation });
    expect(details.occurrenceID).not.toBe(domainFailureDiagnostics(token).occurrenceID);
    const leaf = dataProperty(details.payload, "detail");
    expect(recordIdentity(leaf)).toBe(byName(name));
    const stable = (v: unknown) =>
      JSON.stringify(v, (_, x) => (typeof x === "bigint" ? `#${x}` : x));
    expect(stable(leaf)).toBe(stable(domainFailureDiagnostics(token).payload));
    expect(leaf).toBe(domainFailureDiagnostics(token).payload);
    if (!isDomainFailure(details.cause)) throw Error("expected private original cause");
    expect(details.cause).toBe(token);
  }
});

test("emitted, foreign-operation, non-leaf, standard and success outcomes pass through", () => {
  const emitted = domain.create(
    byName("codec::invalid_data"),
    record(byName("codec::invalid_data"), [
      ["path", ""],
      ["reason", "authored"],
    ]),
    origin,
  );
  expect(normalize.map(failure(emitted), operation)).toMatchObject({ kind: "domain" });
  const foreign = domain.create(
    byName("http::transport_failed"),
    record(byName("http::transport_failed"), [["phase", "connect"]]),
    origin,
    undefined,
    { boundary: "native", operation: "test:normalize/other" },
  );
  const foreignMapped = normalize.map(failure(foreign), operation);
  expect(foreignMapped.kind).toBe("domain");
  if (foreignMapped.kind !== "domain") throw Error("expected domain");
  expect(domainFailureDiagnostics(foreignMapped.value).declaration.name).toBe(
    "http::transport_failed",
  );
  const narrow = createNormalizer(domain, {
    failed: byName("http::request_failed"),
    leaves: [byName("http::invalid_request")],
  });
  const nonLeaf = domain.create(
    byName("http::transport_failed"),
    record(byName("http::transport_failed"), [["phase", "connect"]]),
    origin,
    undefined,
    { boundary: "native", operation },
  );
  const nonLeafMapped = narrow.map(failure(nonLeaf), operation);
  expect(nonLeafMapped.kind).toBe("domain");
  if (nonLeafMapped.kind !== "domain") throw Error("expected domain");
  expect(domainFailureDiagnostics(nonLeafMapped.value).declaration.name).toBe(
    "http::transport_failed",
  );
  const ok = success(1n);
  expect(normalize.map(ok, operation)).toBe(ok);
  const standard = caught(Error("boom"), origin);
  expect(normalize.map(standard, operation)).toBe(standard);
});
