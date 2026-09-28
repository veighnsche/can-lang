import { test, expect } from "bun:test";
import { CodecIssue, type Reason } from "./budget.ts";
import { parseJSONText } from "./document.ts";
import { decodeJSON, type Schema, type SchemaNode } from "./json.ts";
import { ownBytes } from "../bytes.ts";

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

test("numeric roots keep verbatim source evidence", () => {
  for (const [text, parsed] of [
    ["9007199254740993", 9007199254740992],
    ["1e309", Infinity],
    ["-0", -0],
    ["1e2", 100],
  ] as const) {
    const doc = parseJSONText(text);
    expect(Object.is(doc.parsed, parsed)).toBe(true);
    expect(doc.tokens.get(doc.rootHolder)?.get("")).toBe(text);
  }
});

test("nonnumeric roots parse with values only", () => {
  for (const [text, parsed] of [
    ['"s"', "s"],
    ["true", true],
    ["null", null],
  ] as const) {
    const doc = parseJSONText(text);
    expect(doc.parsed).toBe(parsed);
    expect(doc.rootHolder).toBeDefined();
    expect(doc.tokens.get(doc.rootHolder)?.get("")).toBeUndefined();
  }
});

test("nested holders keep numeric tokens under empty and repeated names", () => {
  const doc = parseJSONText('{"":9007199254740993,"a":{"x":1},"b":{"x":2}}');
  const root = doc.parsed as Record<string, unknown>;
  expect(doc.tokens.get(root)?.get("")).toBe("9007199254740993");
  expect(doc.tokens.get(root.a as object)?.get("x")).toBe("1");
  expect(doc.tokens.get(root.b as object)?.get("x")).toBe("2");
  expect(doc.tokens.get(root)?.get("a")).toBeUndefined();
  expect(root[""]).toBe(9007199254740992);
});

test("arrays keep numeric tokens by index", () => {
  const doc = parseJSONText('[9007199254740993,"s",true,null,-0]');
  const items = doc.parsed as unknown[];
  expect(items[0]).toBe(9007199254740992);
  expect(Object.is(items[4], -0)).toBe(true);
  expect(doc.tokens.get(items as unknown as object)?.get("0")).toBe("9007199254740993");
  expect(doc.tokens.get(items as unknown as object)?.get("4")).toBe("-0");
  expect(doc.tokens.get(items as unknown as object)?.get("1")).toBeUndefined();
  expect(doc.tokens.get(items as unknown as object)?.get("2")).toBeUndefined();
  expect(doc.tokens.get(items as unknown as object)?.get("3")).toBeUndefined();
});

test("nonnumeric members decode to parsed values", () => {
  const s = schema("r", [
    {
      identity: "r",
      kind: "record",
      name: "p::r",
      fields: [
        { name: "label", type: "str" },
        { name: "flag", type: "bool" },
        { name: "ratio", type: "float" },
      ],
    },
  ]);
  const result = decodeJSON(s, from('{"label":"s","flag":true,"ratio":1e2}')) as any;
  expect(result.label).toBe("s");
  expect(result.flag).toBe(true);
  expect(result.ratio).toBe(100);
});

test("syntax refusal precedes duplicate detection", () => {
  fails(() => parseJSONText('{"a":1,"a":}'), "invalid_json");
  fails(() => parseJSONText('{"a":1,"a":2}'), "duplicate_member", "/a");
  fails(() => parseJSONText('{"a":1,"a":2,}'), "invalid_json");
});
