import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createIO } from "../platform/io.ts";
import { createEnvironment } from "../platform/env.ts";
import { copyBytes } from "../bytes.ts";
import { record, recordIdentity } from "../data.ts";
import { success, invoke, type Completion } from "../completion.ts";
import { runAssertion } from "../assert/runner.ts";
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
    "http::credentials_missing",
    "codec::invalid_data",
    "io::read_failed",
    "io::limit_exceeded",
    "env::invalid_name",
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
const identity = (name: string) => errors.find((e) => e.declaration === name)!.identity;
const types = {
  readFailed: identity("can.std.io@1::read_failed"),
  limit: identity("can.std.io@1::limit_exceeded"),
  invalidData: identity("can.std.codec@1::invalid_data"),
};
const origin = { source: "test", start: 0, end: 0, invocation: [] };
function check(result: Completion, name: string, payload: object) {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain");
  const d = domainFailureDiagnostics(result.value);
  expect(d.declaration.name).toBe(name);
  expect(d.payload).toMatchObject(payload);
}
function input(chunks: Uint8Array[]) {
  let i = 0;
  return () =>
    new ReadableStream<Uint8Array>({
      pull(c) {
        if (i < chunks.length) c.enqueue(chunks[i++]!);
        else c.close();
      },
    });
}
test("bounded stdin keeps exact bytes, huge limits and split UTF-8", async () => {
  const api = createIO(domain, types, input([new Uint8Array([0, 255]), new Uint8Array([1])]));
  const result = await api.stdinBytes(3n);
  expect(result.kind).toBe("ok");
  if (result.kind !== "ok") throw Error();
  expect(Array.from(copyBytes(result.value, origin))).toEqual([0, 255, 1]);
  const data = new TextEncoder().encode("\ufeffhé😀");
  const text = await createIO(
    domain,
    types,
    input(Array.from(data, (v) => new Uint8Array([v]))),
  ).stdinText(10n ** 100n);
  expect(text).toEqual(success("\ufeffhé😀"));
  expect(await createIO(domain, types, input([])).stdinText(0n)).toEqual(success(""));
  check(
    await createIO(domain, types, input([new Uint8Array([255])])).stdinText(1n),
    "codec::invalid_data",
    {
      path: "",
      reason: "utf8",
    },
  );
});
test("negative limits never open input; overflow cancels without partial success", async () => {
  let opens = 0,
    cancels = 0;
  const api = createIO(domain, types, () => {
    opens++;
    return new ReadableStream({
      start(c) {
        c.enqueue(new Uint8Array([1, 2]));
      },
      cancel() {
        cancels++;
        return Promise.reject(new Error("cleanup"));
      },
    });
  });
  check(await api.stdinBytes(-1n), "io::limit_exceeded", { limit: -1n });
  expect(opens).toBe(0);
  check(await api.stdinBytes(1n), "io::limit_exceeded", { limit: 1n });
  expect(opens).toBe(1);
  expect(cancels).toBe(1);
  check(
    await createIO(domain, types, input([new Uint8Array([0])])).stdinText(0n),
    "io::limit_exceeded",
    {
      limit: 0n,
    },
  );
});
test("expected native read failures map, defects remain standard", async () => {
  for (const method of ["stdinBytes", "stdinText"] as const) {
    const api = createIO(
      domain,
      types,
      () =>
        new ReadableStream({
          start(c) {
            c.error(Object.assign(new Error("private"), { code: "EIO" }));
          },
        }),
    );
    check(await api[method](10n), "io::read_failed", {
      operation: method === "stdinBytes" ? "stdin_bytes" : "stdin_text",
    });
  }
  const api = createIO(domain, types, () => {
    throw new TypeError("defect");
  });
  expect((await invoke(() => api.stdinBytes(1n), origin)).kind).toBe("standard");
});
test("environment validates before lookup and preserves presence and exact value", async () => {
  let calls = 0;
  const env = createEnvironment(
    domain,
    {
      invalidName: identity("can.std.env@1::invalid_name"),
      missing: identity("can.std.http@1::credentials_missing"),
      some: "some",
      none: "none",
    },
    (name) => {
      calls++;
      return name === "EMPTY" ? "" : name === "VALUE" ? "hé\n" : undefined;
    },
  );
  for (const bad of ["", "lower", "A-B", "1A", "A=B", "A\0", "À", "A\n"]) {
    check(await env.required(bad), "env::invalid_name", { name: bad });
    check(await env.optional(bad), "env::invalid_name", { name: bad });
  }
  expect(calls).toBe(0);
  expect(await env.required("EMPTY")).toEqual(success(""));
  expect(await env.required("VALUE")).toEqual(success("hé\n"));
  check(await env.required("MISSING"), "http::credentials_missing", { variable: "MISSING" });
  for (const [name, id, value] of [
    ["EMPTY", "some", ""],
    ["VALUE", "some", "hé\n"],
    ["MISSING", "none", undefined],
  ]) {
    const result = await env.optional(name!);
    expect(result.kind).toBe("ok");
    if (result.kind !== "ok") throw Error();
    expect(recordIdentity(result.value)).toBe(id);
    expect(result.value).toEqual(record(id!, value === undefined ? [] : [["value", value]]));
  }
});
test("unsupplied assertions cannot read stdin or environment", async () => {
  let reads = 0;
  const io = createIO(domain, types, () => {
    reads++;
    throw Error("ambient");
  });
  const env = createEnvironment(
    domain,
    { invalidName: "", missing: "", some: "", none: "" },
    () => {
      reads++;
      throw Error("ambient");
    },
  );
  for (const actual of [
    (context) => io.stdinBytes(1n, context),
    (context) => env.required("VALUE", context),
    (context) => env.optional("VALUE", context),
  ] as ((
    context: Parameters<typeof runAssertion>[0]["actual"] extends (context: infer C) => unknown
      ? C
      : never,
  ) => Promise<Completion>)[]) {
    const result = await runAssertion({
      root: { package: "test", declaration: "boundary", name: "missing" },
      actual,
      expected: async () => success(undefined),
    });
    expect(result.passed).toBe(false);
  }
  expect(reads).toBe(0);
});
