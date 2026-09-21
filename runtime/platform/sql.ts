// Typed Bun.SQL pools and rows. The compiler binds every query to a
// checked I37 descriptor value plus shared scalar schemas; this module
// owns connection establishment, pre-launch parameter validation, native
// failure classification, and immutable row decoding. Values only ever
// travel as bound template parameters through Bun.SQL; the string-call
// form, unsafe helpers, and second lexers are never used.
import { success, failure, type Completion, type AssertionContext } from "../completion.ts";
import { denyLiveBoundary } from "../assert/context.ts";
import { record, array, dataProperty, recordIdentity } from "../data.ts";
import { createDomainRuntime } from "../domain.ts";
import { resourceStateFailure } from "../failure.ts";
import { ownBytes, copyBytes, isBytes } from "../bytes.ts";
import { registerResource, useResource, closeResource } from "../owner.ts";
import { createSQLDescriptors, type SQLDescriptor } from "./sql-descriptor.ts";

const origin = Object.freeze({ source: "can:sql", start: 0, end: 0, invocation: Object.freeze([]) });
type Descriptors = ReturnType<typeof createSQLDescriptors>;
type Native = InstanceType<typeof Bun.SQL>;
type Contracts = Readonly<{
  credentialsMissing: string; connectionFailed: string; queryFailed: string;
  rowMissing: string; rowCount: string; schemaMismatch: string; constraintFailed: string;
  closeFailed: string; rowLimit: string; unsupportedValue: string;
}>;
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
type PoolState = {
  client: Native;
  closeDeadlineMs?: number;
  closeStartedAt?: number;
};
const pools = new WeakMap<object, PoolState>();
const object = (value: unknown): value is object => value !== null && (typeof value === "object" || typeof value === "function");
export function isSQLPoolValue(kind: string | undefined, value: unknown): boolean {
  return kind === "pool" && object(value) && pools.has(value);
}
const MIN_INT64 = -(1n << 63n);
const MAX_INT64 = (1n << 63n) - 1n;
const variableName = /^[A-Z_][A-Z0-9_]*$/;

function isPostgresFailure(value: unknown): value is Error & { code: string; errno?: unknown; constraint?: unknown } {
  return value instanceof Error && value.name === "PostgresError" && typeof (value as { code?: unknown }).code === "string";
}

export function createSQLPools(
  domain: ReturnType<typeof createDomainRuntime>,
  contracts: Contracts,
  lookup: (name: string) => string | undefined,
  descriptors: Descriptors,
) {
  const fail = (identity: string, fields: readonly (readonly [string, unknown])[]): Completion<never> =>
    failure(domain.create(identity, record(identity, fields), origin));
  const missingCredential = (variable: string) => fail(contracts.credentialsMissing, [["variable", variable]]);
  const connectionFailed = (phase: string) => fail(contracts.connectionFailed, [["phase", phase]]);
  const queryFailed = (operation: string, code: string) => fail(contracts.queryFailed, [["operation", operation], ["code", code]]);
  const mismatch = (path: string, reason: string) => fail(contracts.schemaMismatch, [["path", path], ["reason", reason]]);
  const badValue = (path: string, reason: string) => fail(contracts.unsupportedValue, [["path", path], ["reason", reason]]);
  function readPool(value: unknown): PoolState {
    if (!object(value) || !pools.has(value)) throw resourceStateFailure(undefined, origin);
    return pools.get(value)!;
  }
  function classify(operation: string, cause: unknown): Completion<never> {
    // Only PostgresError classifies; anything else is a driver defect and
    // propagates to the generic fault path. Sanitized fields only: never
    // message, detail, options, URL, or credentials.
    if (!isPostgresFailure(cause)) throw cause;
    if (cause.code === "ERR_POSTGRES_CONNECTION_REFUSED" || cause.code === "ERR_POSTGRES_CONNECTION_CLOSED") {
      return connectionFailed("query");
    }
    const errno = typeof cause.errno === "string" ? cause.errno : "";
    const constraint = typeof cause.constraint === "string" ? cause.constraint : "";
    if (constraint !== "" || (errno.length === 5 && errno.startsWith("23"))) {
      return fail(contracts.constraintFailed, [["constraint", constraint !== "" ? constraint : errno]]);
    }
    return queryFailed(operation, errno !== "" ? errno : cause.code !== "" ? cause.code : "unknown");
  }
  function encodeScalar(kind: string, value: unknown, path: string): { ok: true; value: unknown } | { ok: false; failure: Completion<never> } {
    switch (kind) {
      case "bool":
        return typeof value === "boolean" ? { ok: true, value } : { ok: false, failure: badValue(path, "type") };
      case "int":
        if (typeof value !== "bigint") return { ok: false, failure: badValue(path, "type") };
        return value < MIN_INT64 || value > MAX_INT64
          ? { ok: false, failure: badValue(path, "int_range") }
          : { ok: true, value };
      case "float":
        if (typeof value !== "number") return { ok: false, failure: badValue(path, "type") };
        return Number.isFinite(value) ? { ok: true, value } : { ok: false, failure: badValue(path, "nonfinite_float") };
      case "str":
        if (typeof value !== "string") return { ok: false, failure: badValue(path, "type") };
        return value.isWellFormed() ? { ok: true, value } : { ok: false, failure: badValue(path, "unicode_scalar") };
      case "bytes":
        return isBytes(value) ? { ok: true, value: copyBytes(value, origin) } : { ok: false, failure: badValue(path, "type") };
      default:
        throw new TypeError("invalid compiler sql schema");
    }
  }
  function encodeField(field: SQLPlanField, value: unknown): { ok: true; value: unknown } | { ok: false; failure: Completion<never> } {
    const path = "/" + field.name;
    if (field.kind !== "option") return encodeScalar(field.kind, value, path);
    if (field.inner === undefined || field.some === undefined || field.none === undefined) throw new TypeError("invalid compiler sql schema");
    const identity = recordIdentity(value);
    if (identity === field.none) return { ok: true, value: null };
    if (identity !== field.some) return { ok: false, failure: badValue(path, "option_shape") };
    const inner = dataProperty(value, "value");
    const encoded = encodeScalar(field.inner, inner, path + "/value");
    if (!encoded.ok) return encoded;
    return { ok: true, value: encoded.value };
  }
  function encodeParams(plan: SQLPlan, params: unknown): { ok: true; values: unknown[] } | { ok: false; failure: Completion<never> } {
    if (recordIdentity(params) !== plan.params.root) throw new TypeError("invalid compiler sql parameters");
    const values: unknown[] = [];
    for (const field of plan.params.fields) {
      const encoded = encodeField(field, dataProperty(params, field.name));
      if (!encoded.ok) return encoded;
      values.push(encoded.value);
    }
    return { ok: true, values };
  }
  function decodeScalar(kind: string, value: unknown, path: string): { ok: true; value: unknown } | { ok: false; failure: Completion<never> } {
    switch (kind) {
      case "bool":
        return typeof value === "boolean" ? { ok: true, value } : { ok: false, failure: mismatch(path, "type") };
      case "int":
        if (typeof value === "bigint") {
          return value < MIN_INT64 || value > MAX_INT64
            ? { ok: false, failure: mismatch(path, "int_range") }
            : { ok: true, value };
        }
        if (typeof value === "number" && Number.isSafeInteger(value)) return { ok: true, value: BigInt(value) };
        return { ok: false, failure: mismatch(path, typeof value === "number" ? "unsafe_integer" : "type") };
      case "float":
        if (typeof value !== "number") return { ok: false, failure: mismatch(path, "type") };
        return Number.isFinite(value) ? { ok: true, value } : { ok: false, failure: mismatch(path, "nonfinite_float") };
      case "str":
        if (typeof value !== "string") return { ok: false, failure: mismatch(path, "type") };
        return value.isWellFormed() ? { ok: true, value } : { ok: false, failure: mismatch(path, "unicode_scalar") };
      case "bytes":
        return value instanceof Uint8Array ? { ok: true, value: ownBytes(value) } : { ok: false, failure: mismatch(path, "type") };
      default:
        throw new TypeError("invalid compiler sql schema");
    }
  }
  function decodeField(field: SQLPlanField, value: unknown): { ok: true; value: unknown } | { ok: false; failure: Completion<never> } {
    const path = "/" + field.name;
    if (value === null || value === undefined) {
      if (field.kind === "option" && field.none !== undefined) return { ok: true, value: record(field.none, []) };
      return { ok: false, failure: mismatch(path, "null") };
    }
    if (field.kind !== "option") return decodeScalar(field.kind, value, path);
    if (field.inner === undefined || field.some === undefined || field.none === undefined) throw new TypeError("invalid compiler sql schema");
    const decoded = decodeScalar(field.inner, value, path + "/value");
    if (!decoded.ok) return decoded;
    return { ok: true, value: record(field.some, [["value", decoded.value]]) };
  }
  function decodeRow(schema: SQLPlanSchema, row: unknown): { ok: true; value: unknown } | { ok: false; failure: Completion<never> } {
    if (!object(row) || Array.isArray(row)) return { ok: false, failure: mismatch("", "type") };
    const columns = Object.keys(row);
    if (columns.length !== schema.fields.length) {
      const names = new Set(schema.fields.map(field => field.name));
      for (const column of columns) {
        if (!names.has(column)) return { ok: false, failure: mismatch("/" + column, "extra_column") };
      }
      for (const field of schema.fields) {
        if (!Object.hasOwn(row, field.name)) return { ok: false, failure: mismatch("/" + field.name, "missing_column") };
      }
      return { ok: false, failure: mismatch("", "type") };
    }
    const fields: (readonly [string, unknown])[] = [];
    for (const field of schema.fields) {
      if (!Object.hasOwn(row, field.name)) return { ok: false, failure: mismatch("/" + field.name, "missing_column") };
      const decoded = decodeField(field, (row as Record<string, unknown>)[field.name]);
      if (!decoded.ok) return decoded;
      fields.push([field.name, decoded.value]);
    }
    return { ok: true, value: record(schema.root, fields) };
  }
  type Launched =
    | { readonly kind: "failed"; readonly completion: Completion<never> }
    | { readonly kind: "rows"; readonly rows: readonly unknown[] };
  async function launch(
    operation: string, descriptor: SQLDescriptor, plan: SQLPlan, pool: unknown, params: unknown, limit: unknown, context?: AssertionContext,
  ): Promise<Launched> {
    denyLiveBoundary(context, origin);
    const encoded = encodeParams(plan, params);
    if (!encoded.ok) return { kind: "failed", completion: encoded.failure };
    const template = descriptors.template(descriptor, [...encoded.values, limit]);
    const outcome = await useResource(pool, "sql-pool", async (native: unknown): Promise<Completion<readonly unknown[]>> => {
      const client = native as Native;
      let result: unknown;
      try {
        result = await client(template.strings, ...template.values);
      } catch (cause) {
        return classify(operation, cause);
      }
      if (!Array.isArray(result)) return queryFailed(operation, "bad_result");
      return success(result);
    });
    if (outcome.kind !== "ok") return { kind: "failed", completion: outcome };
    return { kind: "rows", rows: outcome.value };
  }
  return Object.freeze({
    async open(variable: unknown, maxConnections: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      denyLiveBoundary(context, origin);
      if (typeof variable !== "string") throw new TypeError("invalid compiler sql variable");
      if (!variableName.test(variable)) return missingCredential(variable);
      // The credential is read exactly once, by exact name: no enumeration,
      // no logging, and no reread for later operations on this pool.
      const url = lookup(variable);
      if (url === undefined) return missingCredential(variable);
      if (typeof maxConnections !== "bigint") throw new TypeError("invalid compiler sql config");
      if (maxConnections < 1n || maxConnections > 2147483647n) return connectionFailed("config");
      const client = new Bun.SQL(url, { adapter: "postgres", bigint: true, max: Number(maxConnections) });
      try {
        // Construction is lazy; awaiting connect proves establishment rather
        // than merely building a client. Any refusal surfaces here.
        await client.connect();
      } catch {
        try { await client.close(); } catch { /* already failed; report the connection */ }
        return connectionFailed("connect");
      }
      const state: PoolState = { client };
      const token = registerResource("sql-pool", client, async (): Promise<Completion<void>> => {
        // Leases are drained before this runs; bound the native close by the
        // remaining pool_close deadline, or wait unbounded on scope drain.
        try {
          if (state.closeDeadlineMs !== undefined && state.closeStartedAt !== undefined) {
            const remaining = Math.max(0, state.closeDeadlineMs - (Date.now() - state.closeStartedAt));
            await client.close({ timeout: remaining / 1000 });
          } else {
            await client.close();
          }
        } catch {
          return fail(contracts.closeFailed, [["reason", "close"]]);
        }
        return success(undefined);
      });
      pools.set(token, state);
      return success(token);
    },
    async close(pool: unknown, timeoutMs: unknown, context?: AssertionContext): Promise<Completion<undefined>> {
      denyLiveBoundary(context, origin);
      const state = readPool(pool);
      if (typeof timeoutMs !== "bigint") throw new TypeError("invalid compiler sql timeout");
      if (timeoutMs < 0n || timeoutMs > 2147483647n) return fail(contracts.closeFailed, [["reason", "invalid_timeout"]]);
      state.closeDeadlineMs = Number(timeoutMs);
      state.closeStartedAt = Date.now();
      // Deny new owners, drain leases, then close natively: a timeout leaves
      // the resource closing and observed while the close continues.
      const completion = await closeResource(pool, "sql-pool", {
        milliseconds: Number(timeoutMs),
        failure: () => fail(contracts.closeFailed, [["reason", "timeout"]]),
      });
      return completion.kind === "ok" ? success(undefined) : completion;
    },
    async queryOne(descriptor: SQLDescriptor, plan: SQLPlan, pool: unknown, params: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      if (plan.rows === undefined) throw new TypeError("invalid compiler sql plan");
      const outcome = await launch("query_one", descriptor, plan, pool, params, 2, context);
      if (outcome.kind === "failed") return outcome.completion;
      const rows = outcome.rows;
      if (rows.length === 0) return fail(contracts.rowMissing, [["query", descriptor.name]]);
      if (rows.length > 1) return fail(contracts.rowCount, [["query", descriptor.name], ["actual", 2n]]);
      const decoded = decodeRow(plan.rows, rows[0]);
      return decoded.ok ? success(decoded.value) : decoded.failure;
    },
    async queryOptional(descriptor: SQLDescriptor, plan: SQLPlan, pool: unknown, params: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      if (plan.rows === undefined || plan.some === undefined || plan.none === undefined) throw new TypeError("invalid compiler sql plan");
      const outcome = await launch("query_optional", descriptor, plan, pool, params, 2, context);
      if (outcome.kind === "failed") return outcome.completion;
      const rows = outcome.rows;
      if (rows.length === 0) return success(record(plan.none, []));
      if (rows.length > 1) return fail(contracts.rowCount, [["query", descriptor.name], ["actual", 2n]]);
      const decoded = decodeRow(plan.rows, rows[0]);
      if (!decoded.ok) return decoded.failure;
      return success(record(plan.some, [["value", decoded.value]]));
    },
    async queryRows(descriptor: SQLDescriptor, plan: SQLPlan, pool: unknown, params: unknown, maxRows: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      if (plan.rows === undefined) throw new TypeError("invalid compiler sql plan");
      if (typeof maxRows !== "bigint") throw new TypeError("invalid compiler sql bound");
      if (maxRows < 0n) return badValue("max_rows", "negative");
      if (maxRows >= MAX_INT64) return badValue("max_rows", "overflow");
      // The bound is validated before conversion; the fetch asks for one
      // past the bound so observed overflow is exact, never estimated.
      const outcome = await launch("query_rows", descriptor, plan, pool, params, maxRows + 1n, context);
      if (outcome.kind === "failed") return outcome.completion;
      const rows = outcome.rows;
      if (BigInt(rows.length) > maxRows) return fail(contracts.rowLimit, [["limit", maxRows]]);
      const decoded: unknown[] = [];
      for (const row of rows) {
        const one = decodeRow(plan.rows, row);
        if (!one.ok) return one.failure;
        decoded.push(one.value);
      }
      return success(array(decoded));
    },
    async execute(descriptor: SQLDescriptor, plan: SQLPlan, pool: unknown, params: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      denyLiveBoundary(context, origin);
      const encoded = encodeParams(plan, params);
      if (!encoded.ok) return encoded.failure;
      const template = descriptors.template(descriptor, encoded.values);
      const outcome = await useResource(pool, "sql-pool", async (native: unknown): Promise<Completion<unknown>> => {
        const client = native as Native;
        let result: unknown;
        try {
          result = await client(template.strings, ...template.values);
        } catch (cause) {
          return classify("execute", cause);
        }
        const count = (result as { count?: unknown } | null)?.count;
        if (typeof count !== "number" || !Number.isSafeInteger(count) || count < 0) {
          return queryFailed("execute", "bad_count");
        }
        return success(BigInt(count));
      });
      return outcome;
    },
  });
}
