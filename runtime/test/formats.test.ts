import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createCodec, type Schema, type SchemaNode } from "../codec/json.ts";
import { CodecIssue, type Reason } from "../codec/budget.ts";
import { createJsonlFramer } from "../codec/jsonl.ts";
import { openByteCell } from "../transport/stream/readable.ts";
import { registerReader } from "../transport/stream/lifecycle.ts";
import { createStreamReads } from "../transport/stream/readable.ts";
import { value, success, failure } from "../completion.ts";
import type { Completion } from "../completion.ts";
import { runOwnedRoot } from "../owner.ts";
import { dataArray, dataProperty, record } from "../data.ts";
import { ownBytes } from "../bytes.ts";
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
const decls = catalogue.errors.filter((e) => [1110, 1316, 1318, 1319].includes(e.id));
const declarations = decls.map((e) => ({ ...e, parameters: 0 }));
const errors = decls.map((e) =>
  shape(
    "error",
    e.identity,
    e.fields.map((f) => ({ name: f.name, type: f.type === "int" ? int.identity : str.identity })),
  ),
);
const domain = createDomainRuntime({ declarations, shapes: [str, int, ...errors] });
const identity = (name: string) => errors.find((e) => e.declaration === name)!.identity;
const failOf = (result: Completion) => {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain");
  return domainFailureDiagnostics(result.value);
};
function check(result: Completion, id: number, payload: object) {
  const d = failOf(result);
  expect(d.declaration.id).toBe(id);
  expect(d.payload).toMatchObject(payload);
}
const origin = { source: "test:formats", start: 0, end: 0, invocation: [] };
const fail = (id: string, fields: readonly (readonly [string, unknown])[], cause?: unknown) =>
  failure(domain.create(id, record(id, fields), origin, cause));
const primitives: SchemaNode[] = ["int", "str", "bool", "float"].map((name) => ({
  identity: name,
  kind: "primitive",
  name,
}));
const schema = (root: string, nodes: SchemaNode[] = []): Schema => ({
  root,
  nodes: [...primitives, ...nodes],
});
const row: SchemaNode = {
  identity: "row",
  kind: "record",
  name: "t::row",
  fields: [
    { name: "name", type: "str" },
    { name: "n", type: "int" },
  ],
};
const rowSchema = schema("row", [row]);
const codec = createCodec(rowSchema, domain, {
  invalidData: identity("can.std.codec@1::invalid_data"),
  readFailed: identity("can.std.stream@1::read_failed"),
  cancelled: identity("can.std.stream@1::cancelled"),
});
const reads = createStreamReads(domain, {
  readFailed: identity("can.std.stream@1::read_failed"),
  cancelled: identity("can.std.stream@1::cancelled"),
  closeFailed: identity("can.std.stream@1::close_failed"),
  limitExceeded: identity("can.std.codec@1::invalid_data"),
});
const from = (text: string) => ownBytes(new TextEncoder().encode(text));
const byteReader = (chunks: Uint8Array[]): object => {
  let index = 0;
  const stream = new ReadableStream<Uint8Array>({
    pull(controller) {
      if (index < chunks.length) controller.enqueue(chunks[index++]);
      else controller.close();
    },
  });
  return registerReader(
    openByteCell(stream, 65536n),
    fail,
    identity("can.std.stream@1::close_failed"),
    { scopeManaged: true },
  );
};
function fails(run: () => unknown, reason: Reason, path?: string) {
  try {
    run();
    throw Error("accepted invalid document");
  } catch (e) {
    expect(e).toBeInstanceOf(CodecIssue);
    expect((e as CodecIssue).reason).toBe(reason);
    if (path !== undefined) expect((e as CodecIssue).path).toBe(path);
  }
}
async function owned(body: () => Promise<void>): Promise<void> {
  const out = await runOwnedRoot(async () => {
    await body();
    return success(undefined);
  });
  expect(out.completion.kind).toBe("ok");
  expect(out.cleanupFailed).toBe(false);
}
const handler = (seen: unknown[]) => async (v: unknown) => {
  seen.push(v);
  return success(undefined);
};

test("toml decodes typed configuration and fails its native violations", async () => {
  const s = schema("config", [
    {
      identity: "config",
      kind: "record",
      name: "t::config",
      fields: [
        { name: "title", type: "str" },
        { name: "server", type: "server" },
      ],
    },
    {
      identity: "server",
      kind: "record",
      name: "t::server",
      fields: [
        { name: "host", type: "str" },
        { name: "ports", type: "ports" },
      ],
    },
    { identity: "ports", kind: "array", name: "int[]", element: "int" },
  ]);
  const c = createCodec(s, domain, {
    invalidData: identity("can.std.codec@1::invalid_data"),
    readFailed: identity("can.std.stream@1::read_failed"),
    cancelled: identity("can.std.stream@1::cancelled"),
  });
  const out = value(
    await c.decodeToml(from('title = "t"\n[server]\nhost = "h"\nports = [80, 443]\n')),
  ) as any;
  expect(out.title).toBe("t");
  expect(out.server.host).toBe("h");
  expect(dataArray(out.server.ports)).toEqual([80n, 443n]);
  check(
    await c.decodeToml(from('title = "t"\n[server]\nhost = "h"\nports = [80, 9007199254740993]\n')),
    1110,
    { path: "", reason: "invalid_toml" },
  );
  check(
    await c.decodeToml(from('title = "a"\ntitle = "b"\n[server]\nhost = "h"\nports = []\n')),
    1110,
    { path: "", reason: "invalid_toml" },
  );
  check(await c.decodeToml(from("[1,2]")), 1110, { path: "", reason: "invalid_toml" });
  const f = createCodec(schema("float"), domain, {
    invalidData: identity("can.std.codec@1::invalid_data"),
    readFailed: identity("can.std.stream@1::read_failed"),
    cancelled: identity("can.std.stream@1::cancelled"),
  });
  check(await f.decodeToml(from("a = nan")), 1110, { path: "", reason: "type" });
  const d = createCodec(
    schema("d", [
      { identity: "d", kind: "record", name: "t::d", fields: [{ name: "d", type: "str" }] },
    ]),
    domain,
    {
      invalidData: identity("can.std.codec@1::invalid_data"),
      readFailed: identity("can.std.stream@1::read_failed"),
      cancelled: identity("can.std.stream@1::cancelled"),
    },
  );
  check(await d.decodeToml(from("d = 2024-01-02")), 1110, { path: "/d", reason: "type" });
});

test("yaml projects core scalars and pins rounding, sharing and cycles", async () => {
  const out = value(await codec.decodeYaml(from('name: "r"\nn: 7\n'))) as any;
  expect(out.name).toBe("r");
  expect(out.n).toBe(7n);
  // The native parser rounds 9007199254740993 to 9007199254740992, which
  // is itself not a safe integer, so projection fails closed. No rounded
  // value can land back inside the safe range (no representable double
  // sits between MAX_SAFE_INTEGER and 2^53), so integers stay exact.
  check(await codec.decodeYaml(from('name: "r"\nn: 9007199254740993\n')), 1110, {
    path: "/n",
    reason: "integer_token",
  });
  const edge = value(await codec.decodeYaml(from('name: "r"\nn: 9007199254740991\n'))) as any;
  expect(edge.n).toBe(9007199254740991n);
  // Duplicate keys resolve last-wins with no native metadata to reject them.
  const dup = value(await codec.decodeYaml(from('name: "a"\nname: "b"\nn: 1\n'))) as any;
  expect(dup.name).toBe("b");
  const strs = createCodec(
    schema("w", [
      {
        identity: "w",
        kind: "record",
        name: "t::w",
        fields: [
          { name: "a", type: "str" },
          { name: "b", type: "str" },
          { name: "c", type: "str" },
        ],
      },
    ]),
    domain,
    {
      invalidData: identity("can.std.codec@1::invalid_data"),
      readFailed: identity("can.std.stream@1::read_failed"),
      cancelled: identity("can.std.stream@1::cancelled"),
    },
  );
  const w = value(await strs.decodeYaml(from("a: 2024-01-02T03:04:05Z\nb: yes\nc: on\n"))) as any;
  expect(w.a).toBe("2024-01-02T03:04:05Z");
  expect(w.b).toBe("yes");
  expect(w.c).toBe("on");
  // Shared aliases project per occurrence.
  const pair = createCodec(
    schema("p", [
      {
        identity: "p",
        kind: "record",
        name: "t::p",
        fields: [
          { name: "a", type: "int" },
          { name: "b", type: "int" },
        ],
      },
    ]),
    domain,
    {
      invalidData: identity("can.std.codec@1::invalid_data"),
      readFailed: identity("can.std.stream@1::read_failed"),
      cancelled: identity("can.std.stream@1::cancelled"),
    },
  );
  const shared = value(await pair.decodeYaml(from("a: &x 4\nb: *x\n"))) as any;
  expect(shared.a).toBe(4n);
  expect(shared.b).toBe(4n);
  // Genuine cycles fail instead of recursing forever.
  const loop = createCodec(
    schema("a", [
      { identity: "a", kind: "record", name: "t::a", fields: [{ name: "b", type: "a" }] },
    ]),
    domain,
    {
      invalidData: identity("can.std.codec@1::invalid_data"),
      readFailed: identity("can.std.stream@1::read_failed"),
      cancelled: identity("can.std.stream@1::cancelled"),
    },
  );
  check(await loop.decodeYaml(from("b: &x\n  b: *x\n")), 1110, { path: "/b/b", reason: "cycle" });
  // Multi-document streams project against array schemas.
  const rows = createCodec(
    schema("rows", [{ identity: "rows", kind: "array", name: "t::row[]", element: "row" }, row]),
    domain,
    {
      invalidData: identity("can.std.codec@1::invalid_data"),
      readFailed: identity("can.std.stream@1::read_failed"),
      cancelled: identity("can.std.stream@1::cancelled"),
    },
  );
  const docs = value(await rows.decodeYaml(from("name: a\nn: 1\n---\nname: b\nn: 2\n"))) as unknown;
  expect(dataArray(docs).map((r) => dataProperty(r, "name"))).toEqual(["a", "b"]);
});

test("json5 accepts extensions and pins rounding and duplicates", async () => {
  const out = value(await codec.decodeJson5(from('{/* c */name: "r", // d\nn: 7,}'))) as any;
  expect(out.name).toBe("r");
  expect(out.n).toBe(7n);
  const single = value(await codec.decodeJson5(from("{name: 'q', n: 1}"))) as any;
  expect(single.name).toBe("q");
  check(await codec.decodeJson5(from('{name: "r", n: 9007199254740993}')), 1110, {
    path: "/n",
    reason: "integer_token",
  });
  const edge = value(await codec.decodeJson5(from('{name: "r", n: 9007199254740991}'))) as any;
  expect(edge.n).toBe(9007199254740991n);
  const dup = value(await codec.decodeJson5(from('{name: "a", name: "b", n: 1}'))) as any;
  expect(dup.name).toBe("b");
  check(await codec.decodeJson5(from('{name: "a", n:}')), 1110, {
    path: "",
    reason: "invalid_json5",
  });
});

test("jsonl bounded decode is exact and strict", async () => {
  const rows = value(
    await codec.decodeJsonl(from('{"name":"a","n":9007199254740993}\n{"name":"b","n":-0}\n')),
  ) as unknown;
  const items = dataArray(rows);
  expect(items.length).toBe(2);
  expect(dataProperty(items[0], "n")).toBe(9007199254740993n);
  expect(dataProperty(items[1], "n")).toBe(0n);
  // Blank lines skipped, CRLF and unterminated final line accepted.
  const framed = value(
    await codec.decodeJsonl(from('\n{"name":"a","n":1}\r\n\r\n{"name":"b","n":2}')),
  ) as unknown;
  expect(dataArray(framed).length).toBe(2);
  expect(dataArray(value(await codec.decodeJsonl(from(""))) as unknown).length).toBe(0);
  // Multi-line records, trailing garbage and truncation fail loudly.
  check(await codec.decodeJsonl(from('{\n"name": "a", "n": 1\n}\n')), 1110, {
    path: "/0",
    reason: "invalid_json",
  });
  check(await codec.decodeJsonl(from('{"name":"a","n":1}\nng\n')), 1110, {
    path: "/1",
    reason: "invalid_json",
  });
  check(await codec.decodeJsonl(from('{"name":"a","n":1}\n{"name":"b","n":')), 1110, {
    path: "/1",
    reason: "invalid_json",
  });
  // Record faults carry the record index.
  check(await codec.decodeJsonl(from('{"name":"a","n":1}\n{"name":"b","n":1,"n":2}\n')), 1110, {
    path: "/1/n",
    reason: "duplicate_member",
  });
  check(await codec.decodeJsonl(from('{"name":"a","n":1}\n{"name":"b","n":"x"}\n')), 1110, {
    path: "/1/n",
    reason: "type",
  });
});

test("consume matches bounded decode at every split point", async () => {
  await owned(async () => {
    const text =
      '{"name":"héllo ❄","n":1}\n\n{"name":"b","n":9007199254740993}\r\n{"name":"c","n":-12}';
    const raw = new TextEncoder().encode(text);
    const want = dataArray(value(await codec.decodeJsonl(ownBytes(raw))) as unknown).map((r) => [
      dataProperty(r, "name"),
      dataProperty(r, "n"),
    ]);
    expect(want.length).toBe(3);
    for (let split = 1; split < raw.length; split++) {
      const seen: unknown[] = [];
      const count = value(
        await codec.consume(byteReader([raw.slice(0, split), raw.slice(split)]), handler(seen)),
      ) as bigint;
      expect(count).toBe(3n);
      expect(seen.map((r) => [dataProperty(r, "name"), dataProperty(r, "n")])).toEqual(want);
    }
    // Whole input as one cell plus empty-cell edges.
    const seen: unknown[] = [];
    expect(value(await codec.consume(byteReader([raw]), handler(seen))) as bigint).toBe(3n);
    expect(seen.length).toBe(3);
    const empty: unknown[] = [];
    expect(value(await codec.consume(byteReader([]), handler(empty))) as bigint).toBe(0n);
    expect(empty.length).toBe(0);
  });
});

test("consume stops on cancellation, read failure and record faults", async () => {
  await owned(async () => {
    const enc = new TextEncoder();
    let release!: () => void;
    let started = false;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    const gated = new ReadableStream<Uint8Array>({
      async pull(c) {
        started = true;
        await gate;
        c.enqueue(enc.encode('{"name":"a","n":1}\n'));
        c.close();
      },
    });
    const cancellable = registerReader(
      openByteCell(gated, 65536n),
      fail,
      identity("can.std.stream@1::close_failed"),
      { scopeManaged: true },
    );
    const seen: unknown[] = [];
    const pending = codec.consume(cancellable, handler(seen));
    while (!started) await Promise.resolve();
    value(await reads.cancelReader(cancellable, "test-done"));
    release();
    check(await pending, 1318, { reason: "test-done" });
    expect(seen.length).toBe(0);
    const boom = new ReadableStream<Uint8Array>({
      pull() {
        throw Object.assign(new Error("reader boom"), { code: "reader_boom" });
      },
    });
    const failing = registerReader(
      openByteCell(boom, 65536n),
      fail,
      identity("can.std.stream@1::close_failed"),
      { scopeManaged: true },
    );
    check(await codec.consume(failing, handler([])), 1316, { reason: "reader_boom" });
    const partial: unknown[] = [];
    check(
      await codec.consume(
        byteReader([new TextEncoder().encode('{"name":"a","n":1}\n{"name":"b","n":"x"}\n')]),
        handler(partial),
      ),
      1110,
      { path: "/1/n", reason: "type" },
    );
    expect(partial.length).toBe(1);
  });
});

test("oversized records fail the framing byte cap", () => {
  const framer = createJsonlFramer(16);
  expect(framer.push(new TextEncoder().encode('{"a":1}\n'))).toHaveLength(1);
  fails(
    () => {
      framer.push(new TextEncoder().encode('{"0123456789abcdef"'));
    },
    "byte_limit",
    "/1",
  );
});
