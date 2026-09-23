import { test, expect } from "bun:test";
import { encodeJSON, decodeJSON, type Schema, type SchemaNode } from "../codec/json.ts";
import { CodecIssue, type Reason } from "../codec/budget.ts";
import { ownBytes, copyBytes } from "../bytes.ts";
import { record, recordIdentity, array } from "../data.ts";
const origin = { source: "test:codec", start: 0, end: 0, invocation: [] };
const primitives: SchemaNode[] = ["int", "str", "bool", "float"].map((name) => ({
  identity: name,
  kind: "primitive",
  name,
}));
const schema = (root: string, nodes: SchemaNode[] = []): Schema => ({
  root,
  nodes: [...primitives, ...nodes],
});
const from = (text: string) => ownBytes(new TextEncoder().encode(text));
const text = (bytes: unknown) => new TextDecoder().decode(copyBytes(bytes, origin));
function fails(run: () => unknown, reason: Reason, path?: string) {
  try {
    run();
    throw Error("accepted invalid JSON");
  } catch (e) {
    expect(e).toBeInstanceOf(CodecIssue);
    expect((e as CodecIssue).reason).toBe(reason);
    if (path !== undefined) expect((e as CodecIssue).path).toBe(path);
  }
}

test("native holder tokens follow schema types rather than repeated field spellings", () => {
  const s = schema("root", [
    {
      identity: "root",
      kind: "record",
      name: "p::root",
      fields: [
        { name: "count", type: "int" },
        { name: "child", type: "child" },
      ],
    },
    {
      identity: "child",
      kind: "record",
      name: "p::child",
      fields: [{ name: "count", type: "float" }],
    },
  ]);
  const result = decodeJSON(s, from('{"child":{"count":-0},"count":9007199254740993.0}')) as any;
  expect(result.count).toBe(9007199254740993n);
  expect(Object.is(result.child.count, -0)).toBe(true);
  expect(recordIdentity(result)).toBe("root");
  expect(text(encodeJSON(s, result))).toBe('{"count":9007199254740993,"child":{"count":-0}}');
  expect(decodeJSON(schema("int"), from("1e309"))).toBe(10n ** 309n);
  fails(() => decodeJSON(schema("float"), from("1e309")), "nonfinite");
});
test("duplicates decode escaped keys and native syntax takes precedence", () => {
  const s = schema("r", [
    { identity: "r", kind: "record", name: "p::r", fields: [{ name: "a", type: "int" }] },
  ]);
  fails(() => decodeJSON(s, from('{"a":1,"\\u0061":2}')), "duplicate_member", "/a");
  fails(() => decodeJSON(s, from('{"a":1,"a":2,}')), "invalid_json");
  fails(() => decodeJSON(s, from('{"a":1,"a":}')), "invalid_json");
  fails(() => decodeJSON(s, from('\ufeff{"a":1}')), "invalid_json");
  fails(() => decodeJSON(s, from('{"a":1,"z":0,"b":0}')), "extra_member", "/b");
  fails(() => decodeJSON(s, from('{"z":0}')), "missing_member", "/a");
  fails(() => decodeJSON(s, from('{"a":"1"}')), "type", "/a");
});
test("variants retain nominal leaf identity and use canonical tags", () => {
  const s = schema("v", [
    { identity: "v", kind: "variant", name: "p::v", leaves: ["none", "some"] },
    { identity: "none", kind: "record", name: "p::none" },
    {
      identity: "some",
      kind: "record",
      name: "p::some<int>",
      fields: [{ name: "value", type: "int" }],
    },
  ]);
  const data = record("some", [["value", 9007199254740993n]]);
  const encoded = text(encodeJSON(s, data));
  expect(encoded).toBe('{"case":"p::some<int>","value":{"value":9007199254740993}}');
  expect(recordIdentity(decodeJSON(s, from(encoded)))).toBe("some");
  fails(
    () => decodeJSON(s, from('{"case":"p::some<str>","value":{"value":1}}')),
    "variant_tag",
    "/case",
  );
});
test("encoding counts delimiters, escaping, signs, and cumulative integer expansion", () => {
  expect(text(encodeJSON(schema("str"), "\n", 4))).toBe('"\\n"');
  fails(() => encodeJSON(schema("str"), "\n", 3), "byte_limit");
  expect(text(encodeJSON(schema("float"), -0, 2))).toBe("-0");
  fails(() => encodeJSON(schema("float"), -0, 1), "byte_limit");
  const s = schema("a", [{ identity: "a", kind: "array", name: "int[]", element: "int" }]);
  expect(text(encodeJSON(s, array([1n, 2n]), 5))).toBe("[1,2]");
  fails(() => encodeJSON(s, array([1n, 2n]), 4), "byte_limit");
  fails(() => decodeJSON(s, from("[1e9,1e9]"), 15), "byte_limit", "/1");
  fails(() => encodeJSON(schema("str"), "\ud800"), "unicode_scalar");
  fails(() => decodeJSON(schema("str"), from('"\\ud800"')), "unicode_scalar");
  fails(() => decodeJSON(schema("str"), ownBytes(new Uint8Array([0xff]))), "utf8");
});
test("recursive data admits shared subtrees and rejects cycles and depth overflow", () => {
  const s = schema("node", [
    {
      identity: "node",
      kind: "record",
      name: "p::node",
      fields: [{ name: "children", type: "children" }],
    },
    { identity: "children", kind: "array", name: "p::node[]", element: "node" },
  ]);
  const bottom = record("node", [["children", array([])]]);
  const shared = record("node", [["children", array([bottom, bottom])]]);
  expect(text(encodeJSON(s, shared))).toBe('{"children":[{"children":[]},{"children":[]}]}');
  const items: unknown[] = [];
  const cyclic = record("node", [["children", items]]);
  items.push(cyclic);
  fails(() => encodeJSON(s, cyclic), "cycle");
  let deep: any = bottom;
  for (let i = 0; i < 32; i++) deep = record("node", [["children", array([deep])]]);
  fails(() => encodeJSON(s, deep), "depth_limit");
  fails(() => decodeJSON(s, from("[".repeat(65) + "0" + "]".repeat(65))), "depth_limit");
});

test("exact public byte/depth/node bounds reject limit plus one", () => {
  const limit = 8_388_608;
  const scalar = "x".repeat(limit - 2);
  const encoded = encodeJSON(schema("str"), scalar);
  expect(copyBytes(encoded, origin).length).toBe(limit);
  expect((decodeJSON(schema("str"), encoded) as string).length).toBe(limit - 2);
  fails(() => encodeJSON(schema("str"), scalar + "x"), "byte_limit");
  fails(() => decodeJSON(schema("str"), from('"' + scalar + 'x"')), "byte_limit");
  const recursive = schema("a", [
    { identity: "a", kind: "array", name: "recursive[]", element: "a" },
  ]);
  let nested: unknown[] = [];
  for (let i = 1; i < 64; i++) nested = [nested];
  expect(text(encodeJSON(recursive, nested))).toBe("[".repeat(64) + "]".repeat(64));
  expect(decodeJSON(recursive, from("[".repeat(64) + "]".repeat(64)))).toEqual(nested);
  fails(() => encodeJSON(recursive, [nested]), "depth_limit");
  const bools = schema("a", [{ identity: "a", kind: "array", name: "bool[]", element: "bool" }]);
  // The root array consumes one node, so 999,999 primitive elements fit.
  const many = "[" + "true,".repeat(999_998) + "true]";
  expect((decodeJSON(bools, from(many)) as unknown[]).length).toBe(999_999);
  expect(text(encodeJSON(bools, Array(999_999).fill(true)))).toBe(many);
  fails(() => decodeJSON(bools, from("[" + "true,".repeat(999_999) + "true]")), "node_limit");
  fails(() => encodeJSON(bools, Array(1_000_000).fill(true)), "node_limit");
}, 30_000);

test("hostile properties never execute during defensive schema validation", () => {
  const s = schema("r", [
    { identity: "r", kind: "record", name: "p::r", fields: [{ name: "value", type: "int" }] },
  ]);
  let traps = 0;
  const forged = new Proxy(
    {},
    {
      get() {
        traps++;
        throw Error("secret");
      },
      ownKeys() {
        traps++;
        throw Error("secret");
      },
    },
  );
  fails(() => encodeJSON(s, forged), "type");
  expect(traps).toBe(0);
  const invalid = record("r", [["value", null]]);
  fails(() => encodeJSON(s, invalid), "type", "/value");
  const arraySchema = schema("a", [
    { identity: "a", kind: "array", name: "int[]", element: "int" },
  ]);
  const getter: any[] = [];
  Object.defineProperty(getter, 0, {
    get() {
      traps++;
      throw Error("secret");
    },
  });
  fails(() => encodeJSON(arraySchema, getter), "type");
  expect(traps).toBe(0);
});

test("nested duplicate diagnostics retain RFC6901 paths before syntax precedence", () => {
  const s = schema("int"); // Duplicate detection precedes typed root validation.
  fails(() => decodeJSON(s, from('{"outer":{"a":1,"\\u0061":2}}')), "duplicate_member", "/outer/a");
  fails(
    () => decodeJSON(s, from('{"a/b":[{"~":1,"\\u007e":2}]}')),
    "duplicate_member",
    "/a~1b/0/~0",
  );
  fails(() => decodeJSON(s, from('{"outer":{"a":1,"a":2},}')), "invalid_json", "");
});
test("revoked array proxies become codec type failures", () => {
  const { proxy, revoke } = Proxy.revocable([], {});
  revoke();
  const s = schema("a", [{ identity: "a", kind: "array", name: "int[]", element: "int" }]);
  fails(() => encodeJSON(s, proxy), "type", "");
});

test("one schema and codec preserve precision across raw fetch, AI state and LLM text boundaries", async () => {
  const shared = schema("receipt", [
    {
      identity: "receipt",
      kind: "record",
      name: "app::receipt",
      fields: [{ name: "amount", type: "int" }],
    },
  ]);
  const state = record("receipt", [["amount", 9007199254740993n]]);
  const outbound = encodeJSON(shared, state, 1024);
  const request = new Request("https://fixture.invalid/receipts", {
    method: "POST",
    body: new Uint8Array(copyBytes(outbound, origin)),
  });
  expect(await request.text()).toBe('{"amount":9007199254740993}');
  const response = new Response('{"amount":9007199254740993.0}', { status: 200 });
  const fetchPayload = decodeJSON(
    shared,
    ownBytes(new Uint8Array(await response.arrayBuffer())),
    1024,
  );
  // This is the generated text boundary after provider envelope extraction.
  const llmPayload = decodeJSON(shared, from('{"amount":90071992547409930e-1}'), 1024);
  expect(fetchPayload).toEqual(state);
  expect(llmPayload).toEqual(state);
  expect(text(encodeJSON(shared, fetchPayload, 1024))).toBe(text(outbound));
});
