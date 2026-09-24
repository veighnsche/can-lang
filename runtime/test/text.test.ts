import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createText } from "../text.ts";
import { dataProperty } from "../data.ts";
import { value, type Completion } from "../completion.ts";
const shape = (kind: string, declaration: string): FailureShape => ({
  identity: createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration]))
    .digest("hex"),
  kind,
  declaration,
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
});
const text = shape("primitive", "str"),
  integers = shape("primitive", "int"),
  declarations = catalogue.errors.filter((e) =>
    [
      "text::empty_separator",
      "text::empty_pattern",
      "text::invalid_unicode",
      "text::invalid_regex",
      "text::invalid_limit",
    ].includes(e.name),
  );
const errors = declarations.map((e) => ({
  ...shape("error", e.identity),
  fields: e.fields.map((f) => ({
    name: f.name,
    type: f.type === "int" ? integers.identity : text.identity,
  })),
}));
const domain = createDomainRuntime({
  declarations: declarations.map((e) => ({
    identity: e.identity,
    name: e.name,
    parameters: 0,
  })),
  shapes: [text, integers, ...errors],
});
const api = createText(domain, {
  emptySeparator: errors[0].identity,
  emptyPattern: errors[1].identity,
  invalidUnicode: errors[2].identity,
  invalidRegex: errors[3].identity,
  invalidLimit: errors[4].identity,
  match: "text::regex_match",
});
function invalid(result: Completion, name: string) {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain failure");
  expect(domainFailureDiagnostics(result.value).declaration.name).toBe(name);
}

test("search, case and trim delegate native code-unit and Unicode behavior", async () => {
  for (const input of ["", "A😀B", "İßΣς", "\u00a0\ufeff hello \u2029", "\ud800"]) {
    for (const needle of ["", "A", "😀", "\ud83d", "z"]) {
      expect(value(await api.includes(input, needle))).toBe(input.includes(needle));
      expect(value(await api.startsWith(input, needle))).toBe(input.startsWith(needle));
      expect(value(await api.endsWith(input, needle))).toBe(input.endsWith(needle));
    }
    expect(value(await api.toLowerCase(input))).toBe(input.toLowerCase());
    expect(value(await api.toUpperCase(input))).toBe(input.toUpperCase());
    expect(value(await api.trim(input))).toBe(input.trim());
  }
  expect(value(await api.slice("A😀B", 1n, 2n))).toBe("\ud83d");
  expect(value(await api.slice("A😀B", -(10n ** 50n), 10n ** 50n))).toBe("A😀B");
});

test("literal replacement preserves dollar sequences and split retains every empty piece", async () => {
  for (const replacement of ["$&", "$`", "$'", "$$", "$1", "😀"]) {
    expect(value(await api.replaceAll("aba", "a", replacement))).toBe(
      replacement + "b" + replacement,
    );
  }
  expect(value(await api.replaceAll("a.a", ".", "$&"))).toBe("a$&a");
  const pieces = value(await api.split(",a,,", ","));
  expect(pieces).toEqual(["", "a", "", ""]);
  expect(Object.isFrozen(pieces)).toBe(true);
  expect(value(await api.split("", ","))).toEqual([""]);
  expect(value(await api.join(pieces, ","))).toBe(",a,,");
  expect(value(await api.join([], ","))).toBe("");
  invalid(await api.split("text", ""), "text::empty_separator");
  invalid(await api.replaceAll("text", "", "x"), "text::empty_pattern");
});

test("scalar and grapheme APIs deliberately differ from UTF-16 indexing", async () => {
  const input = "A😀e\u0301👩‍👩‍👧‍👦";
  const scalars = value(await api.scalars(input));
  expect(scalars).toEqual(Array.from(input, (c) => BigInt(c.codePointAt(0)!)));
  expect(scalars.length).toBeLessThan(input.length);
  expect(Object.isFrozen(scalars)).toBe(true);
  expect(value(await api.fromScalars(scalars))).toBe(input);
  const segments = value(await api.graphemes(input));
  expect(segments).toEqual(["A", "😀", "e\u0301", "👩‍👩‍👧‍👦"]);
  expect(Object.isFrozen(segments)).toBe(true);
  expect(value(await api.normalizeNFC("e\u0301"))).toBe("é");
  expect(value(await api.scalars(""))).toEqual([]);
  expect(value(await api.graphemes(""))).toEqual([]);
  expect(value(await api.fromScalars([]))).toBe("");
});

test("named Unicode operations reject unpaired surrogates and invalid scalar values", async () => {
  for (const input of ["\ud800", "\udfff", "a\ud800b", "\udc00\ud800"]) {
    invalid(await api.scalars(input), "text::invalid_unicode");
    invalid(await api.graphemes(input), "text::invalid_unicode");
    invalid(await api.normalizeNFC(input), "text::invalid_unicode");
  }
  for (const scalar of [-1n, 0xd800n, 0xdfffn, 0x110000n, 10n ** 100n])
    invalid(await api.fromScalars([65n, scalar, 66n]), "text::invalid_unicode");
  for (const scalar of [0n, 0xd7ffn, 0xe000n, 0x10ffffn])
    expect(value(await api.fromScalars([scalar]))).toBe(String.fromCodePoint(Number(scalar)));
});

test("bulk fromCodePoint uses bounded native argument chunks", async () => {
  const input = Object.freeze(
    Array.from({ length: 150000 }, (_, i) => (i % 2 === 0 ? 65n : 0x1f600n)),
  );
  const result = value(await api.fromScalars(input));
  expect(result).toBe("A😀".repeat(75000));
  expect(input[1]).toBe(0x1f600n);
});

test("regex compiles validated flags and rejects bad patterns", async () => {
  expect(typeof value(await api.compileRegex("(a+)", "im"))).toBe("object");
  for (const flags of ["g", "y", "d", "z", "ii "])
    invalid(await api.compileRegex("a", flags), "text::invalid_regex");
  invalid(await api.compileRegex("(a", ""), "text::invalid_regex");
  invalid(await api.compileRegex("a", "uv"), "text::invalid_regex");
});

test("matches report UTF-16 offsets, groups, and absent captures", async () => {
  const handle = value(await api.compileRegex("(\\w+)@(\\w+)", ""));
  const hits = value(await api.findMatches(handle, "a@b c@d", 10n)) as unknown[];
  expect(hits.length).toBe(2);
  expect([
    dataProperty(hits[0], "text"),
    dataProperty(hits[0], "start"),
    dataProperty(hits[0], "end"),
    dataProperty(hits[0], "groups"),
  ]).toEqual(["a@b", 0n, 3n, ["a", "b"]]);
  const either = value(await api.compileRegex("(a)|(b)", ""));
  const lone = value(await api.findMatches(either, "b", 10n)) as unknown[];
  expect(dataProperty(lone[0], "groups")).toEqual(["", "b"]);
  const astral = value(await api.compileRegex(".", "u"));
  const first = value(await api.findMatches(astral, "\ud83d\ude00x", 10n)) as unknown[];
  expect([dataProperty(first[0], "start"), dataProperty(first[0], "end")]).toEqual([0n, 2n]);
});
test("empty matches advance like native matchAll and honor the maximum cap", async () => {
  for (const flags of ["u", "v"]) {
    const handle = value(await api.compileRegex("(?:)", flags));
    const hits = value(await api.findMatches(handle, "😀x", 10n)) as unknown[];
    const starts = hits.map((hit) => dataProperty(hit, "start"));
    const native = [..."😀x".matchAll(new RegExp("(?:)", `${flags}g`))].map((m) =>
      BigInt(m.index!),
    );
    expect(starts).toEqual([0n, 2n, 3n]);
    expect(starts).toEqual(native);
    expect(hits.map((hit) => [dataProperty(hit, "text"), dataProperty(hit, "end")])).toEqual([
      ["", 0n],
      ["", 2n],
      ["", 3n],
    ]);
  }
  const plain = value(await api.compileRegex("(?:)", ""));
  const ascii = value(await api.findMatches(plain, "😀x", 10n)) as unknown[];
  expect(ascii.map((hit) => dataProperty(hit, "start"))).toEqual([0n, 1n, 2n, 3n]);
  const capped = value(await api.findMatches(plain, "a".repeat(9999), 10000n)) as unknown[];
  expect(capped.length).toBe(10000);
  expect(dataProperty(capped[9999], "start")).toBe(9999n);
});
test("matches advance past empty hits without shared lastIndex and honor limits", async () => {
  const handle = value(await api.compileRegex("x*", ""));
  const first = value(await api.findMatches(handle, "ab", 10n)) as unknown[];
  expect(first.map((hit) => [dataProperty(hit, "text"), dataProperty(hit, "start")])).toEqual([
    ["", 0n],
    ["", 1n],
    ["", 2n],
  ]);
  const second = value(await api.findMatches(handle, "ab", 10n)) as unknown[];
  expect(second.length).toBe(3);
  expect((value(await api.findMatches(handle, "ab", 1n)) as unknown[]).length).toBe(1);
  expect((value(await api.findMatches(handle, "ab", 0n)) as unknown[]).length).toBe(0);
  invalid(await api.findMatches(handle, "ab", -1n), "text::invalid_limit");
  invalid(await api.findMatches(handle, "ab", 10001n), "text::invalid_limit");
});
