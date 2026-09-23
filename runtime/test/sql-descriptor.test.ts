import { describe, expect, test } from "bun:test";
import { createSQLDescriptors, type SQLDescriptorEntry } from "../platform/sql/descriptor.ts";

const search: SQLDescriptorEntry = {
  dialect: "postgresql",
  cardinality: "many",
  kind: "SelectStmt",
  segments: [
    { text: "SELECT id FROM accounts WHERE x = " },
    { param: 1 },
    { text: " LIMIT " },
    { param: 2 },
  ],
  params: ["term"],
  paramType: "p",
  rowType: "r",
  limit: 2,
  total: 2,
  version: 170007,
};

describe("sql descriptors", () => {
  test("declare and template a checked descriptor", () => {
    const sql = createSQLDescriptors({ "": { search_accounts: search } });
    const descriptor = sql.declareDescriptor("", "search_accounts");
    expect(descriptor.numbers).toEqual([1, 2]);
    expect([...descriptor.strings]).toEqual(["SELECT id FROM accounts WHERE x = ", " LIMIT ", ""]);
    expect([...(descriptor.strings.raw ?? [])]).toEqual([
      "SELECT id FROM accounts WHERE x = ",
      " LIMIT ",
      "",
    ]);
    const built = sql.template(descriptor, ["%x%", 25]);
    expect([...built.strings]).toEqual([...descriptor.strings]);
    expect(built.values).toEqual(["%x%", 25]);
    // Rebuilt text tiles the original statement bytes exactly.
    let text = built.strings[0] ?? "";
    for (let i = 0; i < built.values.length; i++)
      text += `$${descriptor.numbers[i]}${built.strings[i + 1]}`;
    expect(text).toBe("SELECT id FROM accounts WHERE x = $1 LIMIT $2");
  });
  test("repeated parameters share one prepared value", () => {
    const sql = createSQLDescriptors({
      "": {
        r: {
          ...search,
          segments: [
            { text: "SELECT " },
            { param: 1 },
            { text: "," },
            { param: 1 },
            { text: " LIMIT " },
            { param: 2 },
          ],
          dialect: "postgresql",
          cardinality: "one",
        },
      },
    });
    const built = sql.template(sql.declareDescriptor("", "r"), ["a", 2]);
    expect(built.values).toEqual(["a", "a", 2]);
    expect([...built.strings]).toEqual(["SELECT ", ",", " LIMIT ", ""]);
  });
  test("zero-parameter descriptors template to bare strings", () => {
    const sql = createSQLDescriptors({
      "": {
        wipe: {
          dialect: "postgresql",
          cardinality: "execute",
          kind: "DeleteStmt",
          segments: [{ text: "DELETE FROM t" }],
          params: [],
          paramType: "p",
          rowType: "r",
          limit: 0,
          total: 0,
          version: 170007,
        },
      },
    });
    const built = sql.template(sql.declareDescriptor("", "wipe"), []);
    expect([...built.strings]).toEqual(["DELETE FROM t"]);
    expect(built.values).toEqual([]);
  });
  test("unknown names, forged values, and arity breaks throw", () => {
    const sql = createSQLDescriptors({ "": { search_accounts: search } });
    expect(() => sql.declareDescriptor("", "elsewhere")).toThrow("undeclared sql descriptor");
    const both = createSQLDescriptors({
      "": { q: search },
      vendor: {
        q: {
          dialect: "postgresql",
          cardinality: "execute",
          kind: "DeleteStmt",
          segments: [{ text: "DELETE FROM t WHERE id = " }, { param: 1 }],
          params: ["term"],
          paramType: "p",
          rowType: "r",
          limit: 0,
          total: 1,
          version: 170007,
        },
      },
    });
    expect(both.declareDescriptor("", "q").kind).toBe("SelectStmt");
    expect(both.declareDescriptor("vendor", "q").kind).toBe("DeleteStmt");
    expect(() => both.declareDescriptor("other", "q")).toThrow("undeclared sql descriptor");
    const descriptor = sql.declareDescriptor("", "search_accounts");
    expect(() => sql.template({ ...descriptor }, ["%x%", 25])).toThrow("forged sql descriptor");
    expect(() => sql.template(descriptor, ["%x%"])).toThrow("arity");
    expect(() => sql.template(descriptor, ["%x%", 25, 1])).toThrow("arity");
    expect(() => sql.template(descriptor, "nope" as unknown as unknown[])).toThrow(
      "invalid sql values",
    );
  });
  test("corrupt tables refuse at construction", () => {
    const bad: [string, Partial<SQLDescriptorEntry>][] = [
      [
        "cardinality",
        { dialect: "postgresql", cardinality: "bogus" as SQLDescriptorEntry["cardinality"] },
      ],
      ["kind", { kind: "" }],
      ["segments", { segments: [] }],
      ["segment", { segments: [{ text: "x", param: 1 } as unknown as { text: string }] }],
      ["range", { segments: [{ param: 9 }] }],
      ["params", { params: [""] }],
      ["identities", { paramType: "" }],
      ["limit", { limit: 0 }],
      ["names", { params: [] }],
      ["version", { version: 160001 }],
      ["dialect", { dialect: "mysql" as SQLDescriptorEntry["dialect"] }],
      ["sqlite-version", { dialect: "sqlite", version: 170007 }],
    ];
    for (const [label, patch] of bad) {
      expect(
        () => createSQLDescriptors({ "": { search_accounts: { ...search, ...patch } } }),
        label,
      ).toThrow(TypeError);
    }
  });
});
