// Shared immutable SQL value projection. The compiler binds every query
// to shared parameter/row scalar schemas; this codec validates outbound
// parameters before launch and decodes result rows into immutable records
// after. Values only ever travel as bound template parameters; encoding
// rejects out-of-cover values and decoding fails closed on any shape it
// does not understand, never silently casting. The dialect profile only
// selects representation variants the compiler schema already admits.
import type { Completion } from "../../completion.ts";
import { record, dataProperty, recordIdentity } from "../../data.ts";
import { ownBytes, copyBytes, isBytes } from "../../bytes.ts";
import type { FailureOrigin } from "../../failure.ts";
import type { SQLFailures } from "./errors.ts";

export interface SQLPlanField {
  readonly name: string;
  readonly kind: "bool" | "int" | "float" | "str" | "bytes" | "option";
  readonly inner?: "bool" | "int" | "float" | "str" | "bytes";
  readonly some?: string;
  readonly none?: string;
}
export interface SQLPlanSchema {
  readonly root: string;
  readonly fields: readonly SQLPlanField[];
}
export interface SQLPlan {
  readonly params: SQLPlanSchema;
  readonly rows?: SQLPlanSchema;
  readonly some?: string;
  readonly none?: string;
}
export interface SQLValueProfile {
  // "native" decodes booleans from native booleans; "int01" from the
  // exact integers 0 and 1, arriving as bigints under safeIntegers.
  // Encoding accepts booleans under both profiles.
  readonly booleans: "native" | "int01";
  // "naive_utc_string" decodes driver Date objects (DATETIME and
  // TIMESTAMP under the driver's pinned UTC session) as naive
  // "YYYY-MM-DD HH:MM:SS[.mmm]" wall-clock text; "reject" fails them.
  readonly datetimes: "reject" | "naive_utc_string";
  // "canonical" additionally decodes canonical digit strings as exact
  // integers with int64 range enforcement; "reject" fails them.
  readonly integerStrings: "reject" | "canonical";
}
export const postgresValueProfile: SQLValueProfile = {
  booleans: "native",
  datetimes: "reject",
  integerStrings: "reject",
};
export const sqliteValueProfile: SQLValueProfile = {
  booleans: "int01",
  datetimes: "reject",
  integerStrings: "reject",
};
export const mysqlValueProfile: SQLValueProfile = {
  booleans: "int01",
  datetimes: "naive_utc_string",
  integerStrings: "canonical",
};

export const MIN_INT64 = -(1n << 63n);
export const MAX_INT64 = (1n << 63n) - 1n;
const object = (value: unknown): value is object =>
  value !== null && (typeof value === "object" || typeof value === "function");

// Canonical naive rendering of a driver Date: UTC fields, fractional
// seconds only when nonzero. The driver pin makes UTC the wall clock.
function naiveUTCString(value: Date): string {
  const pad = (n: number, width: number) => String(n).padStart(width, "0");
  const base =
    `${pad(value.getUTCFullYear(), 4)}-${pad(value.getUTCMonth() + 1, 2)}-${pad(value.getUTCDate(), 2)} ` +
    `${pad(value.getUTCHours(), 2)}:${pad(value.getUTCMinutes(), 2)}:${pad(value.getUTCSeconds(), 2)}`;
  const ms = value.getUTCMilliseconds();
  return ms === 0 ? base : `${base}.${pad(ms, 3)}`;
}

export type SQLValueCodec = {
  readonly encodeParams: (
    plan: SQLPlan,
    params: unknown,
  ) => { ok: true; values: unknown[] } | { ok: false; failure: Completion<never> };
  readonly decodeRow: (
    schema: SQLPlanSchema,
    row: unknown,
  ) => { ok: true; value: unknown } | { ok: false; failure: Completion<never> };
};

export function createValueCodec(
  origin: FailureOrigin,
  failures: SQLFailures,
  profile: SQLValueProfile,
): SQLValueCodec {
  const mismatch = failures.mismatch;
  const badValue = failures.badValue;
  function encodeScalar(
    kind: string,
    value: unknown,
    path: string,
  ): { ok: true; value: unknown } | { ok: false; failure: Completion<never> } {
    switch (kind) {
      case "bool":
        return typeof value === "boolean"
          ? { ok: true, value }
          : { ok: false, failure: badValue(path, "type") };
      case "int":
        if (typeof value !== "bigint") return { ok: false, failure: badValue(path, "type") };
        return value < MIN_INT64 || value > MAX_INT64
          ? { ok: false, failure: badValue(path, "int_range") }
          : { ok: true, value };
      case "float":
        if (typeof value !== "number") return { ok: false, failure: badValue(path, "type") };
        return Number.isFinite(value)
          ? { ok: true, value }
          : { ok: false, failure: badValue(path, "nonfinite_float") };
      case "str":
        if (typeof value !== "string") return { ok: false, failure: badValue(path, "type") };
        return value.isWellFormed()
          ? { ok: true, value }
          : { ok: false, failure: badValue(path, "unicode_scalar") };
      case "bytes":
        return isBytes(value)
          ? { ok: true, value: copyBytes(value, origin) }
          : { ok: false, failure: badValue(path, "type") };
      default:
        throw new TypeError("invalid compiler sql schema");
    }
  }
  function encodeField(
    field: SQLPlanField,
    value: unknown,
  ): { ok: true; value: unknown } | { ok: false; failure: Completion<never> } {
    const path = "/" + field.name;
    if (field.kind !== "option") return encodeScalar(field.kind, value, path);
    if (field.inner === undefined || field.some === undefined || field.none === undefined)
      throw new TypeError("invalid compiler sql schema");
    const identity = recordIdentity(value);
    if (identity === field.none) return { ok: true, value: null };
    if (identity !== field.some) return { ok: false, failure: badValue(path, "option_shape") };
    const inner = dataProperty(value, "value");
    const encoded = encodeScalar(field.inner, inner, path + "/value");
    if (!encoded.ok) return encoded;
    return { ok: true, value: encoded.value };
  }
  function encodeParams(
    plan: SQLPlan,
    params: unknown,
  ): { ok: true; values: unknown[] } | { ok: false; failure: Completion<never> } {
    if (recordIdentity(params) !== plan.params.root)
      throw new TypeError("invalid compiler sql parameters");
    const values: unknown[] = [];
    for (const field of plan.params.fields) {
      const encoded = encodeField(field, dataProperty(params, field.name));
      if (!encoded.ok) return encoded;
      values.push(encoded.value);
    }
    return { ok: true, values };
  }
  function decodeScalar(
    kind: string,
    value: unknown,
    path: string,
  ): { ok: true; value: unknown } | { ok: false; failure: Completion<never> } {
    switch (kind) {
      case "bool":
        if (typeof value === "boolean") return { ok: true, value };
        // SQLite stores booleans as INTEGER 0/1, which arrives as bigint
        // under safeIntegers; a REAL cell holding exactly 0 or 1 decodes
        // identically. Anything else is a mismatch, never a coercion.
        if (profile.booleans === "int01" && (value === 0n || value === 0))
          return { ok: true, value: false };
        if (profile.booleans === "int01" && (value === 1n || value === 1))
          return { ok: true, value: true };
        return { ok: false, failure: mismatch(path, "type") };
      case "int":
        if (typeof value === "bigint") {
          return value < MIN_INT64 || value > MAX_INT64
            ? { ok: false, failure: mismatch(path, "int_range") }
            : { ok: true, value };
        }
        if (typeof value === "number" && Number.isSafeInteger(value))
          return { ok: true, value: BigInt(value) };
        // MySQL may render integers as canonical digit strings; they
        // decode exactly with the same int64 enforcement, so unsigned
        // values outside the Can range reject as int_range.
        if (
          profile.integerStrings === "canonical" &&
          typeof value === "string" &&
          /^-?\d+$/.test(value)
        ) {
          const parsed = BigInt(value);
          return parsed < MIN_INT64 || parsed > MAX_INT64
            ? { ok: false, failure: mismatch(path, "int_range") }
            : { ok: true, value: parsed };
        }
        return {
          ok: false,
          failure: mismatch(path, typeof value === "number" ? "unsafe_integer" : "type"),
        };
      case "float":
        if (typeof value !== "number") return { ok: false, failure: mismatch(path, "type") };
        return Number.isFinite(value)
          ? { ok: true, value }
          : { ok: false, failure: mismatch(path, "nonfinite_float") };
      case "str":
        if (typeof value === "string") {
          return value.isWellFormed()
            ? { ok: true, value }
            : { ok: false, failure: mismatch(path, "unicode_scalar") };
        }
        // DATETIME and TIMESTAMP arrive as Date objects; under the
        // driver's pinned UTC session their UTC fields are the naive
        // wall clock, rendered canonically. An Invalid Date mismatches.
        if (profile.datetimes === "naive_utc_string" && value instanceof Date) {
          if (!Number.isFinite(value.getTime()))
            return { ok: false, failure: mismatch(path, "type") };
          return { ok: true, value: naiveUTCString(value) };
        }
        return { ok: false, failure: mismatch(path, "type") };
      case "bytes":
        return value instanceof Uint8Array
          ? { ok: true, value: ownBytes(value) }
          : { ok: false, failure: mismatch(path, "type") };
      default:
        throw new TypeError("invalid compiler sql schema");
    }
  }
  function decodeField(
    field: SQLPlanField,
    value: unknown,
  ): { ok: true; value: unknown } | { ok: false; failure: Completion<never> } {
    const path = "/" + field.name;
    if (value === null || value === undefined) {
      if (field.kind === "option" && field.none !== undefined)
        return { ok: true, value: record(field.none, []) };
      return { ok: false, failure: mismatch(path, "null") };
    }
    if (field.kind !== "option") return decodeScalar(field.kind, value, path);
    if (field.inner === undefined || field.some === undefined || field.none === undefined)
      throw new TypeError("invalid compiler sql schema");
    const decoded = decodeScalar(field.inner, value, path + "/value");
    if (!decoded.ok) return decoded;
    return { ok: true, value: record(field.some, [["value", decoded.value]]) };
  }
  function decodeRow(
    schema: SQLPlanSchema,
    row: unknown,
  ): { ok: true; value: unknown } | { ok: false; failure: Completion<never> } {
    if (!object(row) || Array.isArray(row)) return { ok: false, failure: mismatch("", "type") };
    const columns = Object.keys(row);
    if (columns.length !== schema.fields.length) {
      const names = new Set(schema.fields.map((field) => field.name));
      for (const column of columns) {
        if (!names.has(column))
          return { ok: false, failure: mismatch("/" + column, "extra_column") };
      }
      for (const field of schema.fields) {
        if (!Object.hasOwn(row, field.name))
          return { ok: false, failure: mismatch("/" + field.name, "missing_column") };
      }
      return { ok: false, failure: mismatch("", "type") };
    }
    const fields: (readonly [string, unknown])[] = [];
    for (const field of schema.fields) {
      if (!Object.hasOwn(row, field.name))
        return { ok: false, failure: mismatch("/" + field.name, "missing_column") };
      const decoded = decodeField(field, (row as Record<string, unknown>)[field.name]);
      if (!decoded.ok) return decoded;
      fields.push([field.name, decoded.value]);
    }
    return { ok: true, value: record(schema.root, fields) };
  }
  return Object.freeze({ encodeParams, decodeRow });
}
