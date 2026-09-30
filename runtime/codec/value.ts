import {
  array,
  dataArray,
  dataKeys,
  dataProperty,
  record,
  recordIdentity,
  type RecordValue,
} from "../data.ts";
import { ownBytes, type Bytes } from "../bytes.ts";
import { success, failure, type Completion, type AssertionContext } from "../completion.ts";
import { createDomainRuntime } from "../domain.ts";
import { Budget, CodecIssue, reject, childPath, standaloneBytes } from "./budget.ts";
import { decodeInteger, encodeInteger } from "./numbers.ts";
import { parseDocument } from "./document.ts";

export type JSONValueIds = Readonly<
  Record<"null" | "bool" | "int" | "float" | "string" | "array" | "object" | "member", string>
>;
const encoder = new TextEncoder();
const nativeJSON = JSON as JSON & { rawJSON(text: string): unknown };
const origin = Object.freeze({
  source: "can:codec",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
function scalar(text: string, path: string): void {
  if (
    new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(encoder.encode(text)) !== text
  )
    reject(path, "unicode_scalar");
}
function checked<T>(read: () => T, path: string): T {
  try {
    return read();
  } catch (cause) {
    if (cause instanceof TypeError) reject(path, "type");
    throw cause;
  }
}
// Only wire containers consume depth. Integer spellings stay bigint-exact;
// decimal/exponent spellings and -0 are floats.
export function decodeJSONValue(
  input: unknown,
  ids: JSONValueIds,
  bytes = standaloneBytes,
): RecordValue {
  const budget = new Budget(bytes);
  const { parsed, rootHolder, tokens } = parseDocument(input, bytes);
  function visit(
    value: unknown,
    holder: object,
    key: string,
    path: string,
    depth: number,
  ): RecordValue {
    budget.visit(depth + (value !== null && typeof value === "object" ? 1 : 0), path);
    if (value === null) return record(ids.null, []);
    if (typeof value === "boolean") return record(ids.bool, [["value", value]]);
    if (typeof value === "string") {
      scalar(value, path);
      return record(ids.string, [["value", value]]);
    }
    if (typeof value === "number") {
      const token = tokens.get(holder)?.get(key);
      if (token === undefined) throw new TypeError("missing native numeric token evidence");
      if (/^-?\d+$/.test(token) && token !== "-0")
        return record(ids.int, [["value", decodeInteger(token, budget, path)]]);
      if (!Number.isFinite(value)) reject(path, "nonfinite");
      return record(ids.float, [["value", value]]);
    }
    if (Array.isArray(value))
      return record(ids.array, [
        [
          "values",
          array(
            value.map((item, i) => visit(item, value, String(i), childPath(path, i), depth + 1)),
          ),
        ],
      ]);
    if (value === undefined || typeof value !== "object") reject(path, "type");
    return record(ids.object, [
      [
        "members",
        array(
          Object.keys(value).map((name) => {
            const next = childPath(path, name);
            scalar(name, next);
            return record(ids.member, [
              ["name", name],
              [
                "value",
                visit((value as Record<string, unknown>)[name], value, name, next, depth + 1),
              ],
            ]);
          }),
        ),
      ],
    ]);
  }
  return visit(parsed, rootHolder, "", "", 0);
}
export function encodeJSONValue(input: unknown, ids: JSONValueIds, bytes = standaloneBytes): Bytes {
  const budget = new Budget(bytes),
    active = new Set<object>();
  const kinds = new Map(Object.entries(ids).map(([kind, id]) => [id, kind as keyof JSONValueIds]));
  function text(value: unknown, path: string): string {
    if (typeof value !== "string") reject(path, "type");
    if (value.length > budget.remaining) reject(path, "byte_limit");
    scalar(value, path);
    budget.charge(encoder.encode(JSON.stringify(value)).length, path);
    return value;
  }
  function fields(
    value: unknown,
    identity: keyof JSONValueIds,
    names: string[],
    path: string,
  ): unknown[] {
    if (recordIdentity(value) !== ids[identity]) reject(path, "type");
    const keys = checked(() => dataKeys(value), path).filter((key) => typeof key === "string");
    const extra = keys.filter((key) => !names.includes(key));
    if (extra.length) reject(childPath(path, extra[0]), "extra_member");
    return names.map((name) => checked(() => dataProperty(value, name), path));
  }
  function visit(value: unknown, path: string, depth: number): unknown {
    const kind = kinds.get(recordIdentity(value) ?? "");
    if (kind === undefined) reject(path, "type");
    const container = kind === "array" || kind === "object";
    budget.visit(depth + (container ? 1 : 0), path);
    if (active.has(value as object)) reject(path, "cycle");
    active.add(value as object);
    try {
      if (kind === "null") {
        fields(value, kind, [], path);
        budget.charge(4, path);
        return null;
      }
      if (!container) {
        const [item] = fields(value, kind, ["value"], path);
        switch (kind) {
          case "bool":
            if (typeof item !== "boolean") reject(path, "type");
            budget.charge(item ? 4 : 5, path);
            return item;
          case "string":
            return text(item, path);
          case "int":
            if (typeof item !== "bigint") reject(path, "type");
            return nativeJSON.rawJSON(encodeInteger(item, budget, path));
          case "float": {
            if (typeof item !== "number") reject(path, "type");
            if (!Number.isFinite(item)) reject(path, "nonfinite");
            let token = Object.is(item, -0) ? "-0" : JSON.stringify(item);
            // Retain the float case through a decode/encode round trip.
            if (/^-?\d+$/.test(token) && token !== "-0") token += ".0";
            budget.charge(token.length, path);
            return nativeJSON.rawJSON(token);
          }
          default:
            reject(path, "variant_tag");
        }
      }
      const [items] = fields(value, kind, [kind === "array" ? "values" : "members"], path);
      const length = checked(() => dataProperty(items, "length"), path);
      if (typeof length !== "number" || !Number.isSafeInteger(length) || length < 0)
        reject(path, "type");
      budget.charge(2 + Math.max(0, length - 1) + (kind === "object" ? length : 0), path);
      const entries = checked(() => dataArray(items), path);
      if (kind === "array")
        return entries.map((item, i) => visit(item, childPath(path, i), depth + 1));
      const result = Object.create(null) as Record<string, unknown>;
      for (const entry of entries) {
        const [name, child] = fields(entry, "member", ["name", "value"], path);
        if (typeof name !== "string") reject(path, "type");
        const next = childPath(path, name);
        if (Object.hasOwn(result, name)) reject(next, "duplicate_member");
        text(name, next);
        result[name] = visit(child, next, depth + 1);
      }
      return result;
    } finally {
      active.delete(value as object);
    }
  }
  // Projection adapts nominal records and bigint; native stringify owns JSON syntax.
  const encoded = encoder.encode(JSON.stringify(visit(input, "", 0)));
  if (encoded.length !== bytes - budget.remaining)
    throw new TypeError("codec preflight/formatter length mismatch");
  return ownBytes(encoded);
}
export function createJSONValueCodec<T>(
  domain: ReturnType<typeof createDomainRuntime>,
  invalidData: string,
  ids: JSONValueIds,
) {
  function run<T>(operation: () => T): Completion<T> {
    try {
      return success(operation());
    } catch (cause) {
      if (!(cause instanceof CodecIssue)) throw cause;
      return failure(
        domain.create(
          invalidData,
          record(invalidData, [
            ["path", cause.path],
            ["reason", cause.reason],
          ]),
          origin,
        ),
      );
    }
  }
  return Object.freeze({
    async decode(value: unknown, _context?: AssertionContext): Promise<Completion<T>> {
      return run(() => decodeJSONValue(value, ids) as T);
    },
    async encode(value: unknown, _context?: AssertionContext): Promise<Completion<Bytes>> {
      return run(() => encodeJSONValue(value, ids));
    },
  });
}
