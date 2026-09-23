// B1-11 native format qualification on the pinned Bun 1.4.2 binary.
// Run: bun docs/bun-integration/asap/evidence/formats-native-probe.ts
// Rows pin parser behavior the adapters rely on: unsafe-integer handling,
// duplicate keys, nonfinite scalars, TOML Temporal dates, YAML 1.2 core
// typing, alias sharing, cyclic anchors, multi-document arrays, JSON5
// extensions, and JSONL framing tolerance plus prefix-success.
const T = (Bun as any).TOML, Y = (Bun as any).YAML, J5 = (Bun as any).JSON5, JL = (Bun as any).JSONL;
if (Bun.version !== "1.4.2") throw new Error(`pinned Bun 1.4.2 required, saw ${Bun.version}`);
if (typeof T?.parse !== "function" || typeof Y?.parse !== "function" || typeof J5?.parse !== "function") {
  throw new Error("missing native format parser");
}
if (typeof JL?.parse !== "function" || typeof JL?.parseChunk !== "function") {
  throw new Error("missing native JSONL parser");
}
type Row = { status: "ok"; value: unknown } | { status: "error"; name: string; message: string };
const rows: Record<string, Row> = {};
const attempt = (k: string, f: () => unknown) => {
  try { rows[k] = { status: "ok", value: f() }; }
  catch (e: any) { rows[k] = { status: "error", name: e?.name ?? "Error", message: String(e?.message ?? e).slice(0, 200) }; }
};
const show = (v: any): unknown => {
  if (typeof v === "number") return Number.isNaN(v) ? "NaN" : !Number.isFinite(v) ? String(v) : v;
  if (v && typeof v === "object") return { ctor: v.constructor?.name, str: String(v).slice(0, 60) };
  return v;
};
attempt("surface", () => ({
  toml: Object.getOwnPropertyNames(T).sort(), yaml: Object.getOwnPropertyNames(Y).sort(),
  json5: Object.getOwnPropertyNames(J5).sort(), jsonl: Object.getOwnPropertyNames(JL).sort(),
}));
attempt("toml-big", () => T.parse("n = 9007199254740993"));
attempt("yaml-big", () => Y.parse("n: 9007199254740993"));
attempt("json5-big", () => J5.parse("{n: 9007199254740993}"));
attempt("toml-dup", () => T.parse("a = 1\na = 2"));
attempt("yaml-dup", () => Y.parse("a: 1\na: 2"));
attempt("json5-dup", () => J5.parse("{a: 1, a: 2}"));
attempt("nonfinite", () => ({
  toml: [show(T.parse("a = nan").a), show(T.parse("a = inf").a), show(T.parse("a = -inf").a)],
  yaml: [show(Y.parse("a: .nan").a), show(Y.parse("a: .inf").a)],
  json5: [show(J5.parse("{a: NaN}").a), show(J5.parse("{a: Infinity}").a), show(J5.parse("{a: -Infinity}").a)],
}));
attempt("toml-dates", () => ({
  date: show(T.parse("d = 2024-01-02").d), datetime: show(T.parse("d = 2024-01-02T03:04:05Z").d),
  local: show(T.parse("d = 2024-01-02T03:04:05").d), time: show(T.parse("d = 03:04:05").d),
}));
attempt("yaml-scalars", () => ({
  iso: show(Y.parse("d: 2024-01-02T03:04:05Z").d),
  bools: [show(Y.parse("a: true").a), show(Y.parse("a: yes").a), show(Y.parse("a: on").a), show(Y.parse("a: True").a)],
  nulls: [show(Y.parse("a: null").a), show(Y.parse("a: ~").a), show(Y.parse("a: Null").a), show(Y.parse("a:").a)],
  ints: [show(Y.parse("a: 0x10").a), show(Y.parse("a: 0o17").a), show(Y.parse("a: 1_000").a)],
}));
attempt("yaml-structure", () => {
  const aliased: any = Y.parse("a: &x [1]\nb: *x");
  return {
    multidoc: Y.parse("a: 1\n---\na: 2"),
    merge: Y.parse("a: &x {k: 1}\nb: {<<: *x, j: 2}"),
    aliasShared: aliased.a === aliased.b,
  };
});
attempt("yaml-selfref", () => {
  const v: any = Y.parse("a: &x {b: *x}");
  return v?.a?.b === v?.a ? "cyclic" : "acyclic";
});
attempt("roots", () => ({
  yamlScalar: Y.parse("just a string"), json5Array: J5.parse("[1,]"),
  json5Comment: J5.parse("{/* c */a: 1,}"), json5Single: J5.parse("{a: 'x'}"),
  tomlInts: [show(T.parse("a = 0x10").a), show(T.parse("a = 0o17").a), show(T.parse("a = 1_000").a)],
  tomlMixed: T.parse("a = [1, 'x']"),
}));
attempt("toml-rootarr", () => T.parse("[1,2]"));
attempt("jsonl-framing", () => ({
  crlf: JL.parse('{"a":1}\r\n{"b":2}\r\n'), blank: JL.parse('{"a":1}\n\n{"b":2}\n'),
  nonl: JL.parse('{"a":1}'), multiline: JL.parse('{\n"a": 1\n}\n{"b": 2}\n'),
  bignum: JL.parse('{"n": 9007199254740993}\n'),
}));
attempt("jsonl-trailing-garbage", () => JL.parse('{"ok":1}\nng\n'));
attempt("chunk", () => {
  const r: any = JL.parseChunk('{"a":1}\n{"b":2', undefined);
  return {
    keys: Object.keys(r).sort(), values: r.values, read: r.read, done: r.done,
    error: r.error ? { name: r.error.name, message: String(r.error.message).slice(0, 80) } : null,
    empty: JL.parseChunk('', undefined), trailing: JL.parseChunk('{"a":1}\n', undefined),
  };
});
const seen = new Set();
console.log(JSON.stringify({ bun: Bun.version, rows }, (k, v) => {
  if (v && typeof v === "object") { if (seen.has(v)) return "<cycle>"; seen.add(v); }
  return v;
}, 1));
