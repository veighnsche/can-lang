import { test, expect } from "bun:test";
import {
  decodeForm,
  decodeActionForm,
  FormIssue,
  type FormSchema,
  type ActionFormResult,
} from "../platform/form.ts";
import { CodecIssue } from "../codec/budget.ts";
import { ownBytes } from "../bytes.ts";
import { record, array, dataProperty, dataArray } from "../data.ts";
const schema: FormSchema = {
  root: "wire",
  fields: [
    { name: "customer", kind: "str" },
    {
      name: "lines",
      kind: "rows",
      rows: {
        row: "row",
        order: "lines_order",
        collection: "rows",
        item: "item",
        fields: [
          { name: "sku", kind: "str" },
          { name: "tags", kind: "array" },
          { name: "note", kind: "optional", some: "some", none: "none" },
        ],
      },
    },
  ],
};
const ids = { rejected: "rejected", rawEntry: "raw", issue: "issue" };
const bytes = (text: string) => ownBytes(new TextEncoder().encode(text));
const limit = (text: string) => new TextEncoder().encode(text).length;
function action(text: string): ActionFormResult {
  return decodeActionForm(schema, bytes(text), limit(text), ids);
}
function wire(text: string): unknown {
  const result = action(text);
  expect(result.kind).toBe("wire");
  return result.kind === "wire" ? result.value : null;
}
function rejected(text: string): { raw: [string, string][]; issues: [string, string][] } {
  const result = action(text);
  expect(result.kind).toBe("rejected");
  if (result.kind !== "rejected") throw Error("accepted");
  const raw = dataArray(dataProperty(result.value, "raw")).map(
    (entry) => [dataProperty(entry, "name"), dataProperty(entry, "value")] as [string, string],
  );
  const issues = dataArray(dataProperty(result.value, "issues")).map(
    (entry) => [dataProperty(entry, "name"), dataProperty(entry, "reason")] as [string, string],
  );
  return { raw, issues };
}
function row(key: string, sku: string, tags: string[], note: unknown) {
  return record("item", [
    ["key", key],
    [
      "value",
      record("row", [
        ["sku", sku],
        ["tags", array(tags)],
        ["note", note],
      ]),
    ],
  ]);
}
test("keyed rows decode in exact order with scalar, repeated and optional values", () => {
  expect(
    wire(
      "customer=ann&lines_order=b&lines_order=a" +
        "&lines%5Bb%5D%5Bsku%5D=x&lines%5Bb%5D%5Btags%5D=t1&lines%5Bb%5D%5Btags%5D=t2" +
        "&lines%5Ba%5D%5Bsku%5D=y&lines%5Ba%5D%5Bnote%5D=n",
    ),
  ).toEqual(
    record("wire", [
      ["customer", "ann"],
      [
        "lines",
        record("rows", [
          ["order", array(["b", "a"])],
          [
            "items",
            array([
              row("b", "x", ["t1", "t2"], record("none", [])),
              row("a", "y", [], record("some", [["value", "n"]])),
            ]),
          ],
        ]),
      ],
    ]),
  );
  // An empty grid submits no order or entries and stays valid.
  expect(wire("customer=ann")).toEqual(
    record("wire", [
      ["customer", "ann"],
      [
        "lines",
        record("rows", [
          ["order", array([])],
          ["items", array([])],
        ]),
      ],
    ]),
  );
});
test("duplicate order keys reject but retain raw", () => {
  const bad = rejected("customer=ann&lines_order=a&lines_order=a&lines%5Ba%5D%5Bsku%5D=x");
  expect(bad.issues).toEqual([["lines_order", "form_repeated"]]);
  expect(bad.raw).toEqual([
    ["customer", "ann"],
    ["lines_order", "a"],
    ["lines_order", "a"],
    ["lines[a][sku]", "x"],
  ]);
});
test("entries without order keys reject as partial", () => {
  const bad = rejected("customer=ann&lines_order=a&lines%5Bz%5D%5Bsku%5D=x");
  expect(bad.issues).toEqual([
    ["lines[a]", "form_missing"],
    ["lines[z]", "form_missing"],
  ]);
  expect(bad.raw).toEqual([
    ["customer", "ann"],
    ["lines_order", "a"],
    ["lines[z][sku]", "x"],
  ]);
});
test("order keys without entries reject as partial", () => {
  const bad = rejected("customer=ann&lines_order=a&lines_order=b&lines%5Ba%5D%5Bsku%5D=x");
  expect(bad.issues).toEqual([["lines[b]", "form_missing"]]);
  expect(bad.raw).toEqual([
    ["customer", "ann"],
    ["lines_order", "a"],
    ["lines_order", "b"],
    ["lines[a][sku]", "x"],
  ]);
});
test("unknown row fields reject and never enter raw", () => {
  const bad = rejected(
    "customer=ann&lines_order=a&lines%5Ba%5D%5Bsku%5D=x&lines%5Ba%5D%5Bnope%5D=y",
  );
  expect(bad.issues).toEqual([["lines[a][nope]", "type"]]);
  expect(bad.raw).toEqual([
    ["customer", "ann"],
    ["lines_order", "a"],
    ["lines[a][sku]", "x"],
  ]);
});
test("duplicate scalar entries reject", () => {
  const bad = rejected(
    "customer=ann&lines_order=a&lines%5Ba%5D%5Bsku%5D=x&lines%5Ba%5D%5Bsku%5D=y",
  );
  expect(bad.issues).toEqual([["lines[a][sku]", "form_repeated"]]);
  expect(bad.raw).toEqual([
    ["customer", "ann"],
    ["lines_order", "a"],
    ["lines[a][sku]", "x"],
    ["lines[a][sku]", "y"],
  ]);
});
test("missing row scalars reject", () => {
  const bad = rejected("customer=ann&lines_order=a&lines%5Ba%5D%5Btags%5D=t");
  expect(bad.issues).toEqual([["lines[a][sku]", "form_missing"]]);
});
test("row count beyond 64 rejects with row_limit", () => {
  const keys = Array.from({ length: 65 }, (_, i) => "k" + String(i).padStart(2, "0"));
  const text =
    "customer=ann&" +
    keys.map((k) => "lines_order=" + k).join("&") +
    "&" +
    keys.map((k) => `lines%5B${k}%5D%5Bsku%5D=x`).join("&");
  const bad = rejected(text);
  expect(bad.issues).toEqual([["lines[k64]", "row_limit"]]);
  expect(bad.raw.length).toBe(1 + 65 + 65);
  expect(bad.raw[0]).toEqual(["customer", "ann"]);
});
test("row values beyond 2048 bytes reject", () => {
  const big = "x".repeat(2049);
  const bad = rejected(`customer=ann&lines_order=a&lines%5Ba%5D%5Bsku%5D=${big}`);
  expect(bad.issues).toEqual([["lines[a][sku]", "row_value_limit"]]);
  const edge = "y".repeat(2048);
  expect(() => wire(`customer=ann&lines_order=a&lines%5Ba%5D%5Bsku%5D=${edge}`)).not.toThrow();
});
test("malformed order values and keys reject", () => {
  const bad = rejected("customer=ann&lines_order=a+b&lines_order=ok&lines%5Bok%5D%5Bsku%5D=x");
  expect(bad.issues).toEqual([["lines_order", "type"]]);
  expect(bad.raw).toEqual([
    ["customer", "ann"],
    ["lines_order", "a b"],
    ["lines_order", "ok"],
    ["lines[ok][sku]", "x"],
  ]);
  const keyed = rejected("customer=ann&lines%5Bbad+key%5D%5Bsku%5D=x");
  expect(keyed.issues).toEqual([["lines[bad key][sku]", "type"]]);
  expect(keyed.raw).toEqual([["customer", "ann"]]);
});
test("throwing decode maps first issue to declared failures", () => {
  const throwing = (text: string) => decodeForm(schema, bytes(text), limit(text));
  try {
    throwing("customer=ann&lines_order=a&lines_order=a&lines%5Ba%5D%5Bsku%5D=x");
    throw Error("accepted");
  } catch (e) {
    expect(e).toBeInstanceOf(FormIssue);
    expect((e as FormIssue).reason).toBe("form_repeated");
  }
  try {
    throwing("customer=ann&lines_order=a&lines%5Ba%5D%5Bnope%5D=x");
    throw Error("accepted");
  } catch (e) {
    expect(e).toBeInstanceOf(CodecIssue);
    expect((e as CodecIssue).reason).toBe("type");
    expect((e as CodecIssue).path).toBe("/lines[a][nope]");
  }
  try {
    throwing("customer=ann&lines_order=a&lines%5Ba%5D%5Btags%5D=t");
    throw Error("accepted");
  } catch (e) {
    expect(e).toBeInstanceOf(FormIssue);
    expect((e as FormIssue).reason).toBe("form_missing");
  }
  const keys = Array.from({ length: 65 }, (_, i) => "k" + i);
  try {
    throwing(
      "customer=ann&" +
        keys.map((k) => "lines_order=" + k).join("&") +
        "&" +
        keys.map((k) => `lines%5B${k}%5D%5Bsku%5D=x`).join("&"),
    );
    throw Error("accepted");
  } catch (e) {
    expect(e).toBeInstanceOf(CodecIssue);
    expect((e as CodecIssue).reason).toBe("row_limit");
    expect((e as CodecIssue).path).toBe("/lines[k64]");
  }
  try {
    throwing(`customer=ann&lines_order=a&lines%5Ba%5D%5Bsku%5D=${"x".repeat(2049)}`);
    throw Error("accepted");
  } catch (e) {
    expect(e).toBeInstanceOf(CodecIssue);
    expect((e as CodecIssue).reason).toBe("row_value_limit");
    expect((e as CodecIssue).path).toBe("/lines[a][sku]");
  }
});
test("malformed entry shapes reject as unknown", () => {
  for (const name of [
    "lines%5Ba%5D",
    "lines%5Ba%5D%5Bsku%5D%5Bx%5D",
    "lines",
    "lines%5B%5D",
    "lines%5B%5D%5Bsku%5D",
  ]) {
    const bad = rejected(`customer=ann&${name}=x`);
    expect(bad.issues.length).toBe(1);
    expect(bad.issues[0]![1]).toBe("type");
    expect(bad.raw).toEqual([["customer", "ann"]]);
  }
});
