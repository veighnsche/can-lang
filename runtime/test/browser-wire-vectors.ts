// Shared Bun/browser wire-codec vectors for the browser target. This module
// has no test-runner dependency so the same vectors execute under `bun test`
// and inside named browsers through a `bun build --target browser` bundle.
// Every result is JSON-serializable for cross-engine comparison.
import { encodeJSON, decodeJSON, type Schema, type SchemaNode } from "../codec/json.ts";
import { CodecIssue } from "../codec/budget.ts";
import { ownBytes, copyBytes } from "../bytes.ts";
import { record, recordIdentity } from "../data.ts";

export type WireVectorResult = { name: string; pass: boolean; detail: string };

const primitives: SchemaNode[] = ["int", "str", "bool", "float"].map((name) => ({
  identity: name,
  kind: "primitive",
  name,
}));
const schema = (root: string, nodes: SchemaNode[] = []): Schema => ({
  root,
  nodes: [...primitives, ...nodes],
});
const origin = { source: "test:browser-wire", start: 0, end: 0, invocation: [] };
const from = (text: string) => ownBytes(new TextEncoder().encode(text));
const text = (bytes: unknown) => new TextDecoder().decode(copyBytes(bytes, origin));

function issue(run: () => unknown): { reason: string; path: string } {
  try {
    run();
  } catch (e) {
    if (e instanceof CodecIssue) return { reason: e.reason, path: e.path };
    throw e;
  }
  throw new Error("accepted invalid JSON");
}

export function runBrowserWireVectors(): WireVectorResult[] {
  const results: WireVectorResult[] = [];
  const check = (name: string, run: () => string) => {
    try {
      results.push({ name, pass: true, detail: run() });
    } catch (e) {
      results.push({ name, pass: false, detail: String((e as Error)?.message ?? e) });
    }
  };

  check("int64-beyond-safe", () => {
    const decoded = decodeJSON(schema("int"), from("9007199254740993.0")) as bigint;
    if (typeof decoded !== "bigint" || String(decoded) !== "9007199254740993")
      throw new Error(`decoded ${String(decoded)}`);
    const encoded = text(encodeJSON(schema("int"), decoded));
    if (encoded !== "9007199254740993") throw new Error(`encoded ${encoded}`);
    return encoded;
  });

  check("int64-huge-exponent", () => {
    const decoded = decodeJSON(schema("int"), from("1e309")) as bigint;
    const rendered = String(decoded);
    if (typeof decoded !== "bigint" || rendered !== `${10n ** 309n}`)
      throw new Error(`decoded ${rendered.slice(0, 16)}...`);
    return `length=${rendered.length}`;
  });

  check("float-finite", () => {
    const decoded = decodeJSON(schema("float"), from("1.25")) as number;
    if (decoded !== 1.25) throw new Error(`decoded ${String(decoded)}`);
    const negativeZero = decodeJSON(schema("float"), from("-0")) as number;
    if (!Object.is(negativeZero, -0)) throw new Error("lost negative zero");
    const encoded = text(encodeJSON(schema("float"), -0));
    if (encoded !== "-0") throw new Error(`encoded ${encoded}`);
    return encoded;
  });

  check("float-nonfinite-rejected", () => {
    const got = issue(() => decodeJSON(schema("float"), from("1e309")));
    if (got.reason !== "nonfinite") throw new Error(`reason ${got.reason}`);
    return `${got.reason}@${got.path}`;
  });

  const variant = schema("v", [
    { identity: "v", kind: "variant", name: "p::v", leaves: ["none", "some"] },
    { identity: "none", kind: "record", name: "p::none" },
    {
      identity: "some",
      kind: "record",
      name: "p::some<int>",
      fields: [{ name: "value", type: "int" }],
    },
  ]);
  check("variant-canonical-tag", () => {
    const data = record("some", [["value", 9007199254740993n]]);
    const encoded = text(encodeJSON(variant, data));
    if (encoded !== '{"case":"p::some<int>","value":{"value":9007199254740993}}')
      throw new Error(`encoded ${encoded}`);
    const identity = recordIdentity(decodeJSON(variant, from(encoded)));
    if (identity !== "some") throw new Error(`identity ${String(identity)}`);
    return encoded;
  });

  check("variant-wrong-tag", () => {
    const got = issue(() =>
      decodeJSON(variant, from('{"case":"p::some<str>","value":{"value":1}}')),
    );
    if (got.reason !== "variant_tag" || got.path !== "/case")
      throw new Error(`${got.reason}@${got.path}`);
    return `${got.reason}@${got.path}`;
  });

  const option = schema("opt", [
    { identity: "opt", kind: "variant", name: "p::opt", leaves: ["absent", "present"] },
    { identity: "absent", kind: "record", name: "p::none" },
    {
      identity: "present",
      kind: "record",
      name: "p::some<int>",
      fields: [{ name: "value", type: "int" }],
    },
  ]);
  check("option-some-none", () => {
    const some = text(encodeJSON(option, record("present", [["value", 7n]])));
    if (some !== '{"case":"p::some<int>","value":{"value":7}}') throw new Error(`some ${some}`);
    const none = text(encodeJSON(option, record("absent", [])));
    if (none !== '{"case":"p::none","value":{}}') throw new Error(`none ${none}`);
    if (recordIdentity(decodeJSON(option, from(some))) !== "present") throw new Error("some lost");
    if (recordIdentity(decodeJSON(option, from(none))) !== "absent") throw new Error("none lost");
    return `${some} ${none}`;
  });

  const single = schema("r", [
    { identity: "r", kind: "record", name: "p::r", fields: [{ name: "a", type: "int" }] },
  ]);
  check("duplicate-member", () => {
    const got = issue(() => decodeJSON(single, from('{"a":1,"\\u0061":2}')));
    if (got.reason !== "duplicate_member" || got.path !== "/a")
      throw new Error(`${got.reason}@${got.path}`);
    return `${got.reason}@${got.path}`;
  });

  check("duplicate-native-syntax", () => {
    const got = issue(() => decodeJSON(single, from('{"a":1,"a":2,}')));
    if (got.reason !== "invalid_json") throw new Error(`reason ${got.reason}`);
    return got.reason;
  });

  check("extra-member", () => {
    const got = issue(() => decodeJSON(single, from('{"a":1,"z":0}')));
    if (got.reason !== "extra_member" || got.path !== "/z")
      throw new Error(`${got.reason}@${got.path}`);
    return `${got.reason}@${got.path}`;
  });

  check("missing-member", () => {
    const got = issue(() => decodeJSON(single, from('{"z":0}')));
    if (got.reason !== "missing_member" || got.path !== "/a")
      throw new Error(`${got.reason}@${got.path}`);
    return `${got.reason}@${got.path}`;
  });

  check("nested-roundtrip-stable", () => {
    const nested = schema("root", [
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
    const wire = '{"child":{"count":-0},"count":9007199254740993.0}';
    const once = text(encodeJSON(nested, decodeJSON(nested, from(wire))));
    if (once !== '{"count":9007199254740993,"child":{"count":-0}}')
      throw new Error(`encoded ${once}`);
    const twice = text(encodeJSON(nested, decodeJSON(nested, from(once))));
    if (twice !== once) throw new Error(`unstable ${twice}`);
    return once;
  });

  return results;
}
