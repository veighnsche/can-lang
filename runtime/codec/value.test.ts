import { test, expect } from "bun:test";
import { array, record, recordIdentity, dataProperty } from "../data.ts";
import { ownBytes, copyBytes } from "../bytes.ts";
import { CodecIssue, maxNodes, type Reason } from "./budget.ts";
import {
  decodeJSONValue as decode,
  encodeJSONValue as encode,
  type JSONValueIds,
} from "./value.ts";
const prefix = "test:json_";
const ids = Object.fromEntries(
  ["null", "bool", "int", "float", "string", "array", "object", "member"].map((kind) => [
    kind,
    prefix + kind,
  ]),
) as JSONValueIds;
const decodeJSONValue = (value: unknown, bytes?: number) => decode(value, ids, bytes);
const encodeJSONValue = (value: unknown, bytes?: number) => encode(value, ids, bytes);
const from = (text: string) => ownBytes(new TextEncoder().encode(text));
const origin = { source: "test", start: 0, end: 0, invocation: [] };
const output = (value: unknown) =>
  new TextDecoder().decode(copyBytes(encodeJSONValue(value), origin));
const leaf = (kind: string, value: unknown) => record(prefix + kind, [["value", value]]);
function fails(run: () => unknown, reason: Reason, path?: string) {
  try {
    run();
    throw Error("accepted invalid JSON");
  } catch (cause) {
    expect(cause).toBeInstanceOf(CodecIssue);
    expect((cause as CodecIssue).reason).toBe(reason);
    if (path !== undefined) expect((cause as CodecIssue).path).toBe(path);
  }
}
test("open envelope, nested members, every scalar and exact integers round trip", () => {
  const source =
    '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"x","extra":[null,true,1.25]},"id":"opaque","unknown":{"id":9007199254740993},"__proto__":null}';
  const decoded = decodeJSONValue(from(source));
  expect(recordIdentity(decoded)).toBe(prefix + "object");
  expect(output(decoded)).toBe(source);
  expect(output(decodeJSONValue(from('{"method":"notify"}')))).toBe('{"method":"notify"}');
  expect(dataProperty(decodeJSONValue(from("9007199254740993")), "value")).toBe(9007199254740993n);
  expect(output(decodeJSONValue(from("1" + "0".repeat(310))))).toBe("1" + "0".repeat(310));
  const checkFrozen = (value: unknown): void => {
    if (value === null || typeof value !== "object") return;
    expect(Object.isFrozen(value)).toBe(true);
    for (const child of Object.values(value)) checkFrozen(child);
  };
  checkFrozen(decoded);
  expect(decodeJSONValue(encodeJSONValue(decoded))).toEqual(decoded);
});
test("float discriminants and negative zero survive encoding", () => {
  for (const value of [1, 0, -0, 1.25, 1e21, 1e-9]) {
    const decoded = decodeJSONValue(encodeJSONValue(leaf("float", value)));
    expect(recordIdentity(decoded)).toBe(prefix + "float");
    expect(Object.is(dataProperty(decoded, "value"), value)).toBe(true);
  }
  expect(output(leaf("float", 1))).toBe("1.0");
  expect(output(leaf("float", -0))).toBe("-0");
});
test("native parsing keeps strict duplicate, malformed, scalar and UTF-8 checks", () => {
  fails(() => decodeJSONValue(from('{"a/b":{"~x":0,"~x":1}}')), "duplicate_member", "/a~1b/~0x");
  fails(() => decodeJSONValue(from('{"x":}')), "invalid_json");
  fails(() => decodeJSONValue(from('"\\ud800"')), "unicode_scalar");
  fails(() => decodeJSONValue(from('{"\\ud800":0}')), "unicode_scalar");
  fails(() => decodeJSONValue(ownBytes(new Uint8Array([255]))), "utf8");
  fails(() => decodeJSONValue(from("\ufeffnull")), "invalid_json");
  fails(() => decodeJSONValue(from("1e309")), "nonfinite");
});
test("wire byte, depth and node budgets apply in both directions", () => {
  fails(() => decodeJSONValue(from("null"), 3), "byte_limit");
  fails(() => encodeJSONValue(record(prefix + "null", []), 3), "byte_limit");
  fails(() => decodeJSONValue(from("[".repeat(65) + "null" + "]".repeat(65))), "depth_limit");
  let value = record(prefix + "null", []);
  for (let i = 0; i < 65; i++) value = record(prefix + "array", [["values", array([value])]]);
  fails(() => encodeJSONValue(value), "depth_limit");
  fails(() => decodeJSONValue(from("[" + "null,".repeat(maxNodes - 1) + "null]")), "node_limit");
  const repeated = array(Array(maxNodes).fill(record(prefix + "null", [])));
  fails(() => encodeJSONValue(record(prefix + "array", [["values", repeated]])), "node_limit");
}, 30000);
test("encoding rejects duplicate members, invalid data, cycles and nonfinite values", () => {
  const member = record(prefix + "member", [
    ["name", "x"],
    ["value", leaf("int", 1n)],
  ]);
  fails(
    () => encodeJSONValue(record(prefix + "object", [["members", array([member, member])]])),
    "duplicate_member",
    "/x",
  );
  fails(() => encodeJSONValue({ value: 1n }), "type");
  fails(() => encodeJSONValue(leaf("int", 1)), "type");
  fails(() => encodeJSONValue(leaf("float", Infinity)), "nonfinite");
  fails(() => encodeJSONValue(leaf("string", "\ud800")), "unicode_scalar");
  const members: unknown[] = [];
  const cyclic = record(prefix + "array", [["values", members]]);
  members.push(cyclic);
  fails(() => encodeJSONValue(cyclic), "cycle", "/0");
  let invoked = false;
  const accessor: unknown[] = [];
  Object.defineProperty(accessor, 0, {
    get() {
      invoked = true;
      return leaf("int", 1n);
    },
  });
  fails(() => encodeJSONValue(record(prefix + "array", [["values", accessor]])), "type");
  expect(invoked).toBe(false);
});
