import { record, array } from "../data.ts";
import { copyBytes, byteLength } from "../bytes.ts";
import { success, failure, type Completion, type AssertionContext } from "../completion.ts";
import { createDomainRuntime } from "../domain.ts";
import { resourceStateFailure } from "../failure.ts";
import { reject, childPath, Budget } from "../codec/budget.ts";
// Keyed-row bounds mirror the compiler constants: row keys stay short and
// bracket-free so wire names parse unambiguously, rows stay few, and row
// values stay small enough to redisplay verbatim in a 422 fragment.
export const maxFormRows = 64;
export const maxFormRowBytes = 2048;
const rowKeyPattern = /^[A-Za-z0-9_-]{1,64}$/;
const fieldNamePattern = /^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$/;
export type FormRowsSchema = Readonly<{
  row: string;
  order: string;
  collection: string;
  item: string;
  fields: readonly FormField[];
}>;
export type FormField = Readonly<{
  name: string;
  kind: string;
  some?: string;
  none?: string;
  rows?: FormRowsSchema;
}>;
export type FormSchema = Readonly<{
  root: string;
  fields: readonly FormField[];
}>;
export type FormIssueReason =
  | "form_missing"
  | "form_repeated"
  | "type"
  | "row_limit"
  | "row_value_limit";
export class FormIssue extends Error {
  constructor(readonly reason: "invalid_form_encoding" | "form_missing" | "form_repeated") {
    super("invalid form");
  }
}
const origin = Object.freeze({
  source: "can:form",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
// Validation delegates UTF-8 percent decoding to the native operation before
// URLSearchParams applies its otherwise replacement-tolerant decoding policy.
export function strictParameters(
  text: string,
  encodingReason: "invalid_form_encoding" = "invalid_form_encoding",
): URLSearchParams {
  if (/%(?![0-9a-fA-F]{2})/.test(text)) throw new FormIssue(encodingReason);
  for (const pair of text.split("&")) {
    if (pair === "") continue;
    const separator = pair.indexOf("="),
      key = separator < 0 ? pair : pair.slice(0, separator),
      value = separator < 0 ? "" : pair.slice(separator + 1);
    let name: string;
    try {
      name = decodeURIComponent(key.replaceAll("+", " "));
    } catch (cause) {
      if (!(cause instanceof URIError)) throw cause;
      reject("", "utf8");
    }
    try {
      decodeURIComponent(value.replaceAll("+", " "));
    } catch (cause) {
      if (!(cause instanceof URIError)) throw cause;
      reject(childPath("", name!), "utf8");
    }
  }
  return new URLSearchParams(text);
}
function formParams(input: unknown, limit: number): { params: URLSearchParams; budget: Budget } {
  if (byteLength(input) > BigInt(limit)) reject("", "byte_limit");
  const budget = new Budget(Math.max(1, limit));
  budget.visit(1, "");
  let text: string;
  try {
    text = new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(
      copyBytes(input, origin),
    );
  } catch (cause) {
    if (!(cause instanceof TypeError)) throw cause;
    reject("", "utf8");
  }
  return { params: strictParameters(text), budget };
}
type RowEntry = Readonly<{ coll: string; key: string; field: string; value: string }>;
type Classified = {
  top: Map<string, string[]>;
  orders: Map<string, string[]>;
  entries: RowEntry[];
  keysInOrder: Map<string, string[]>;
  known: [string, string][];
  unknown: string[];
};
// One classification pass over document-ordered pairs. Unknown names are
// reported once each, in first-seen order; known pairs are retained
// verbatim for rejected-value redisplay.
function classifyForm(pairs: Iterable<readonly [string, unknown]>, schema: FormSchema): Classified {
  const top = new Map<string, string[]>(),
    orders = new Map<string, string[]>(),
    entries: RowEntry[] = [],
    keysInOrder = new Map<string, string[]>(),
    known: [string, string][] = [],
    unknown: string[] = [],
    seenUnknown = new Set<string>();
  const scalars = new Set(schema.fields.filter((f) => f.kind !== "rows").map((f) => f.name)),
    colls = schema.fields.filter((f) => f.kind === "rows");
  for (const [name, raw] of pairs) {
    const value = typeof raw === "string" ? raw : undefined;
    if (value === undefined || scalars.has(name)) {
      if (value !== undefined && scalars.has(name)) {
        known.push([name, value]);
        const list = top.get(name) ?? [];
        list.push(value);
        top.set(name, list);
        continue;
      }
      if (!seenUnknown.has(name)) {
        seenUnknown.add(name);
        unknown.push(name);
      }
      continue;
    }
    let matched = false;
    for (const coll of colls) {
      const rows = coll.rows;
      if (rows === undefined) throw new TypeError("invalid compiler form schema");
      if (name === rows.order) {
        known.push([name, value]);
        const list = orders.get(coll.name) ?? [];
        list.push(value);
        orders.set(coll.name, list);
        if (rowKeyPattern.test(value)) trackKey(keysInOrder, coll.name, value);
        matched = true;
        break;
      }
      const parsed = parseRowEntry(coll.name, name);
      if (parsed !== undefined) {
        const [key, field] = parsed;
        if (rowKeyPattern.test(key) && rows.fields.some((f) => f.name === field)) {
          known.push([name, value]);
          entries.push({ coll: coll.name, key, field, value });
          trackKey(keysInOrder, coll.name, key);
          matched = true;
          break;
        }
      }
    }
    if (!matched && !seenUnknown.has(name)) {
      seenUnknown.add(name);
      unknown.push(name);
    }
  }
  return { top, orders, entries, keysInOrder, known, unknown };
}
function trackKey(keysInOrder: Map<string, string[]>, coll: string, key: string): void {
  const seen = keysInOrder.get(coll) ?? [];
  if (!seen.includes(key)) seen.push(key);
  keysInOrder.set(coll, seen);
}
// Row entries spell coll[key][field] exactly: one key segment, one field
// segment, no nesting. Keys exclude brackets by grammar, so the first "]["
// split is unambiguous; anything else stays unknown.
function parseRowEntry(coll: string, name: string): readonly [string, string] | undefined {
  const head = coll + "[";
  if (!name.startsWith(head) || !name.endsWith("]")) return undefined;
  const inner = name.slice(head.length, -1),
    separator = inner.indexOf("][");
  if (separator < 0) return undefined;
  const key = inner.slice(0, separator),
    field = inner.slice(separator + 2);
  if (key === "" || field === "" || field.includes("[") || field.includes("]")) return undefined;
  return [key, field] as const;
}
const rowAddress = (coll: string, key: string) => `${coll}[${key}]`;
const entryName = (coll: string, key: string, field: string) => `${coll}[${key}][${field}]`;
const utf8Length = (value: string) => new TextEncoder().encode(value).byteLength;
type Analysis = Readonly<{
  value: unknown;
  raw: readonly (readonly [string, string])[];
  issues: readonly (readonly [string, FormIssueReason])[];
}>;
// Single-pass structural analysis shared by the throwing decoder and the
// action adapter. A null value means issues is non-empty.
function analyzeForm(schema: FormSchema, classified: Classified, budget: Budget): Analysis {
  const issues: [string, FormIssueReason][] = [];
  for (const name of classified.unknown) issues.push([name, "type"]);
  const fields: [string, unknown][] = [];
  for (const field of schema.fields) {
    const path = childPath("", field.name);
    if (field.kind === "rows") {
      const rows = field.rows;
      if (
        rows === undefined ||
        !rows.row ||
        !rows.order ||
        !rows.collection ||
        !rows.item ||
        !Array.isArray(rows.fields)
      )
        throw new TypeError("invalid compiler form schema");
      fields.push([field.name, analyzeRows(field.name, rows, classified, budget, issues)]);
      continue;
    }
    budget.visit(field.kind === "str" ? 1 : 2, path);
    const values = classified.top.get(field.name) ?? [];
    if (field.kind === "array") {
      for (let i = 0; i < values.length; i++) budget.visit(2, childPath(path, i));
      fields.push([field.name, array(values)]);
      continue;
    }
    if (values.length > 1) {
      issues.push([field.name, "form_repeated"]);
      continue;
    }
    if (field.kind === "str") {
      if (values.length === 0) {
        issues.push([field.name, "form_missing"]);
        continue;
      }
      fields.push([field.name, values[0]]);
    } else if (field.kind === "optional") {
      if (!field.some || !field.none) throw new TypeError("invalid compiler form schema");
      budget.visit(3, path);
      if (values.length !== 0) budget.visit(3, childPath(path, "value"));
      fields.push([
        field.name,
        values.length === 0 ? record(field.none, []) : record(field.some, [["value", values[0]]]),
      ]);
    } else throw new TypeError("unknown compiler form field");
  }
  if (issues.length !== 0) return { value: null, raw: classified.known, issues };
  // Typed fields were derived by the compiler; shared codec error paths and
  // node/depth budgets apply without serializing a second JSON representation.
  return { value: record(schema.root, fields), raw: classified.known, issues };
}
function analyzeRows(
  coll: string,
  rows: FormRowsSchema,
  classified: Classified,
  budget: Budget,
  issues: [string, FormIssueReason][],
): unknown {
  const path = childPath("", coll);
  budget.visit(2, path);
  const seen = new Set<string>(),
    ordered: string[] = [];
  for (const key of classified.orders.get(coll) ?? []) {
    budget.visit(2, childPath(path, key));
    if (!rowKeyPattern.test(key)) {
      issues.push([rows.order, "type"]);
      continue;
    }
    if (seen.has(key)) {
      issues.push([rows.order, "form_repeated"]);
      continue;
    }
    seen.add(key);
    ordered.push(key);
  }
  const byKey = new Map<string, Map<string, string[]>>();
  for (const entry of classified.entries) {
    if (entry.coll !== coll) continue;
    let fields = byKey.get(entry.key);
    if (fields === undefined) {
      fields = new Map();
      byKey.set(entry.key, fields);
    }
    const list = fields.get(entry.field) ?? [];
    list.push(entry.value);
    fields.set(entry.field, list);
  }
  // The bound counts distinct keys in merged document order and admits the
  // first 64 to exactness and field checks; excess rows keep only the
  // limit issue plus their retained raw pairs.
  const distinct = classified.keysInOrder.get(coll) ?? [];
  let limited = false;
  if (distinct.length > maxFormRows) {
    issues.push([rowAddress(coll, distinct[maxFormRows]!), "row_limit"]);
    limited = true;
  }
  const admitted = new Set(distinct.slice(0, maxFormRows));
  for (const key of ordered) {
    if (!byKey.has(key)) issues.push([rowAddress(coll, key), "form_missing"]);
  }
  for (const key of byKey.keys()) {
    if (!seen.has(key) && (!limited || admitted.has(key)))
      issues.push([rowAddress(coll, key), "form_missing"]);
  }
  const items: unknown[] = [];
  for (const key of ordered) {
    const row = byKey.get(key);
    if (row === undefined) continue;
    budget.visit(3, childPath(path, key));
    const rowFields: [string, unknown][] = [];
    for (const def of rows.fields) {
      const values = row.get(def.name) ?? [],
        name = entryName(coll, key, def.name),
        valuePath = childPath(childPath(path, key), def.name);
      budget.visit(2, valuePath);
      for (const text of values) {
        budget.visit(2, valuePath);
        if (utf8Length(text) > maxFormRowBytes) issues.push([name, "row_value_limit"]);
      }
      if (def.kind === "array") {
        rowFields.push([def.name, array(values)]);
        continue;
      }
      if (values.length > 1) {
        issues.push([name, "form_repeated"]);
        continue;
      }
      if (def.kind === "str") {
        if (values.length === 0) {
          issues.push([name, "form_missing"]);
          continue;
        }
        rowFields.push([def.name, values[0]]);
      } else if (def.kind === "optional") {
        if (!def.some || !def.none) throw new TypeError("invalid compiler form schema");
        budget.visit(3, valuePath);
        rowFields.push([
          def.name,
          values.length === 0 ? record(def.none, []) : record(def.some, [["value", values[0]]]),
        ]);
      } else throw new TypeError("unknown compiler form field");
    }
    items.push(
      record(rows.item, [
        ["key", key],
        ["value", record(rows.row, rowFields)],
      ]),
    );
  }
  return record(rows.collection, [
    ["order", array(ordered)],
    ["items", array(items)],
  ]);
}
export function decodeForm(schema: FormSchema, input: unknown, limit: number): unknown {
  const { params, budget } = formParams(input, limit);
  const analysis = analyzeForm(schema, classifyForm(params, schema), budget);
  if (analysis.value === null) {
    const [name, reason] = analysis.issues[0]!;
    if (reason === "form_missing" || reason === "form_repeated") throw new FormIssue(reason);
    reject(childPath("", name), reason);
  }
  return analysis.value;
}
export type ActionFormIdentities = Readonly<{ rejected: string; rawEntry: string; issue: string }>;
export type ActionFormResult = Readonly<
  { kind: "wire"; value: unknown } | { kind: "rejected"; value: unknown }
>;
// Adapter decode: one native FormData parse over the strictly gated pairs,
// then structural analysis. Success yields the wire record; any duplicate,
// unknown, partial, order or limit violation yields a rejected value that
// retains every known raw pair in document order.
export function decodeActionForm(
  schema: FormSchema,
  input: unknown,
  limit: number,
  ids: ActionFormIdentities,
): ActionFormResult {
  if (!ids.rejected || !ids.rawEntry || !ids.issue)
    throw new TypeError("invalid compiler form action");
  const { params, budget } = formParams(input, limit);
  const data = new FormData();
  for (const [name, value] of params) data.append(name, value);
  const analysis = analyzeForm(schema, classifyForm(data.entries(), schema), budget);
  if (analysis.value === null)
    return {
      kind: "rejected",
      value: record(ids.rejected, [
        [
          "raw",
          array(
            analysis.raw.map(([name, value]) =>
              record(ids.rawEntry, [
                ["name", name],
                ["value", value],
              ]),
            ),
          ),
        ],
        [
          "issues",
          array(
            analysis.issues.map(([name, reason]) =>
              record(ids.issue, [
                ["name", name],
                ["reason", reason],
              ]),
            ),
          ),
        ],
      ]),
    };
  return { kind: "wire", value: analysis.value };
}
const collections = new WeakMap<object, string>(),
  rows = new WeakMap<object, string>(),
  fields = new WeakMap<object, string>();
function token(map: WeakMap<object, string>, name: string): unknown {
  const value = Object.freeze(Object.create(null));
  map.set(value, name);
  return value;
}
function readToken(map: WeakMap<object, string>, value: unknown): string {
  if (
    value === null ||
    (typeof value !== "object" && typeof value !== "function") ||
    !map.has(value)
  )
    throw resourceStateFailure(undefined, origin);
  return map.get(value)!;
}
export function createForm(
  domain: ReturnType<typeof createDomainRuntime>,
  types: Readonly<{ unknownField: string; invalidName: string }>,
) {
  const unknown = (name: string) =>
    failure(
      domain.create(types.unknownField, record(types.unknownField, [["name", name]]), origin),
    );
  const invalid = (name: string) =>
    failure(domain.create(types.invalidName, record(types.invalidName, [["name", name]]), origin));
  const checked = (name: unknown): string => {
    if (typeof name !== "string") throw new TypeError("invalid compiler form name");
    return name;
  };
  return Object.freeze({
    async namedCollection(
      name: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      // Membership was checked against the wire record at compile time; the
      // runtime re-validates the identifier spelling only.
      const text = checked(name);
      if (!fieldNamePattern.test(text)) return unknown(text);
      return success(token(collections, text));
    },
    async namedField(name: unknown, _context?: AssertionContext): Promise<Completion<unknown>> {
      const text = checked(name);
      if (!fieldNamePattern.test(text)) return unknown(text);
      return success(token(fields, text));
    },
    async rowKey(name: unknown, _context?: AssertionContext): Promise<Completion<unknown>> {
      const text = checked(name);
      if (!rowKeyPattern.test(text)) return invalid(text);
      return success(token(rows, text));
    },
    async orderName(collection: unknown, _context?: AssertionContext): Promise<Completion<string>> {
      return success(readToken(collections, collection) + "_order");
    },
    async inputName(
      collection: unknown,
      row: unknown,
      field: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<string>> {
      return success(
        `${readToken(collections, collection)}[${readToken(rows, row)}][${readToken(fields, field)}]`,
      );
    },
    async fieldName(field: unknown, _context?: AssertionContext): Promise<Completion<string>> {
      return success(readToken(fields, field));
    },
  });
}
