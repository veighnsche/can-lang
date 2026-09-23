// Typed SQL pools and shared query orchestration. The compiler binds every
// query to a checked descriptor value plus shared scalar schemas; this
// module owns connection establishment, pool lifetimes, pre-launch
// parameter validation, and immutable row decoding dispatch. Values only
// ever travel as bound template parameters through native clients; the
// string-call form, unsafe helpers, and second lexers are never used.
// Dialect specifics (client options, failure shapes, metadata) live in
// the per-dialect modules; this file only orchestrates them.
import { success, type Completion, type AssertionContext } from "../../completion.ts";
import { denyLiveBoundary } from "../../assert/context.ts";
import { record, array } from "../../data.ts";
import { createDomainRuntime } from "../../domain.ts";
import { resourceStateFailure } from "../../failure.ts";
import { registerResource, useResource, closeResource } from "../../owner.ts";
import { createSQLDescriptors, type SQLDescriptor } from "./descriptor.ts";
import { createSQLFailures, type SQLCoreContracts, type SQLPoolContracts } from "./errors.ts";
import { createValueCodec, postgresValueProfile, MAX_INT64, type SQLPlan } from "./values.ts";
import { classifyPostgres, postgresAffectedRows, openPostgresClient } from "./postgres.ts";
import { postgresMaxConnections } from "./config.ts";

const origin = Object.freeze({ source: "can:sql", start: 0, end: 0, invocation: Object.freeze([]) });
type Descriptors = ReturnType<typeof createSQLDescriptors>;
export type SQLNative = InstanceType<typeof Bun.SQL>;
type Native = SQLNative;
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
const variableName = /^[A-Z_][A-Z0-9_]*$/;

// createSQLOperations owns the query behavior shared by pools and scoped
// transactions: pre-launch validation, native classification, and immutable
// decoding. The token kind selects which registered resource holds the
// native client lease; pool-only open/close/credential logic stays out.
export function createSQLOperations(
  domain: ReturnType<typeof createDomainRuntime>,
  contracts: SQLCoreContracts,
  descriptors: Descriptors,
) {
  const failures = createSQLFailures(domain, contracts, origin);
  const codec = createValueCodec(origin, failures, postgresValueProfile);
  const fail = failures.fail;
  type Launched =
    | { readonly kind: "failed"; readonly completion: Completion<never> }
    | { readonly kind: "rows"; readonly rows: readonly unknown[] };
  async function launch(
    operation: string, descriptor: SQLDescriptor, plan: SQLPlan, token: unknown, kind: "sql-pool" | "sql-tx", params: unknown, limit: unknown, context?: AssertionContext,
  ): Promise<Launched> {
    denyLiveBoundary(context, origin);
    const encoded = codec.encodeParams(plan, params);
    if (!encoded.ok) return { kind: "failed", completion: encoded.failure };
    const template = descriptors.template(descriptor, [...encoded.values, limit]);
    const outcome = await useResource(token, kind, async (native: unknown): Promise<Completion<readonly unknown[]>> => {
      const client = native as Native;
      let result: unknown;
      try {
        result = await client(template.strings, ...template.values);
      } catch (cause) {
        return classifyPostgres(operation, cause, failures, contracts);
      }
      if (!Array.isArray(result)) return failures.queryFailed(operation, "bad_result");
      return success(result);
    });
    if (outcome.kind !== "ok") return { kind: "failed", completion: outcome };
    return { kind: "rows", rows: outcome.value };
  }
  return Object.freeze({
    async queryOne(descriptor: SQLDescriptor, plan: SQLPlan, token: unknown, kind: "sql-pool" | "sql-tx", params: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      if (plan.rows === undefined) throw new TypeError("invalid compiler sql plan");
      const outcome = await launch("query_one", descriptor, plan, token, kind, params, 2, context);
      if (outcome.kind === "failed") return outcome.completion;
      const rows = outcome.rows;
      if (rows.length === 0) return fail(contracts.rowMissing, [["query", descriptor.name]]);
      if (rows.length > 1) return fail(contracts.rowCount, [["query", descriptor.name], ["actual", 2n]]);
      const decoded = codec.decodeRow(plan.rows, rows[0]);
      return decoded.ok ? success(decoded.value) : decoded.failure;
    },
    async queryOptional(descriptor: SQLDescriptor, plan: SQLPlan, token: unknown, kind: "sql-pool" | "sql-tx", params: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      if (plan.rows === undefined || plan.some === undefined || plan.none === undefined) throw new TypeError("invalid compiler sql plan");
      const outcome = await launch("query_optional", descriptor, plan, token, kind, params, 2, context);
      if (outcome.kind === "failed") return outcome.completion;
      const rows = outcome.rows;
      if (rows.length === 0) return success(record(plan.none, []));
      if (rows.length > 1) return fail(contracts.rowCount, [["query", descriptor.name], ["actual", 2n]]);
      const decoded = codec.decodeRow(plan.rows, rows[0]);
      if (!decoded.ok) return decoded.failure;
      return success(record(plan.some, [["value", decoded.value]]));
    },
    async queryRows(descriptor: SQLDescriptor, plan: SQLPlan, token: unknown, kind: "sql-pool" | "sql-tx", params: unknown, maxRows: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      if (plan.rows === undefined) throw new TypeError("invalid compiler sql plan");
      if (typeof maxRows !== "bigint") throw new TypeError("invalid compiler sql bound");
      if (maxRows < 0n) return failures.badValue("max_rows", "negative");
      if (maxRows >= MAX_INT64) return failures.badValue("max_rows", "overflow");
      // The bound is validated before conversion; the fetch asks for one
      // past the bound so observed overflow is exact, never estimated.
      const outcome = await launch("query_rows", descriptor, plan, token, kind, params, maxRows + 1n, context);
      if (outcome.kind === "failed") return outcome.completion;
      const rows = outcome.rows;
      if (BigInt(rows.length) > maxRows) return fail(contracts.rowLimit, [["limit", maxRows]]);
      const decoded: unknown[] = [];
      for (const row of rows) {
        const one = codec.decodeRow(plan.rows, row);
        if (!one.ok) return one.failure;
        decoded.push(one.value);
      }
      return success(array(decoded));
    },
    async execute(descriptor: SQLDescriptor, plan: SQLPlan, token: unknown, kind: "sql-pool" | "sql-tx", params: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      denyLiveBoundary(context, origin);
      const encoded = codec.encodeParams(plan, params);
      if (!encoded.ok) return encoded.failure;
      const template = descriptors.template(descriptor, encoded.values);
      const outcome = await useResource(token, kind, async (native: unknown): Promise<Completion<unknown>> => {
        const client = native as Native;
        let result: unknown;
        try {
          result = await client(template.strings, ...template.values);
        } catch (cause) {
          return classifyPostgres("execute", cause, failures, contracts);
        }
        return postgresAffectedRows("execute", result, failures);
      });
      return outcome;
    },
  });
}

export function createSQLPools(
  domain: ReturnType<typeof createDomainRuntime>,
  contracts: SQLPoolContracts,
  lookup: (name: string) => string | undefined,
  descriptors: Descriptors,
) {
  const failures = createSQLFailures(domain, contracts, origin);
  const fail = failures.fail;
  const missingCredential = (variable: string) => fail(contracts.credentialsMissing, [["variable", variable]]);
  const connectionFailed = failures.connectionFailed;
  function readPool(value: unknown): PoolState {
    if (!object(value) || !pools.has(value)) throw resourceStateFailure(undefined, origin);
    return pools.get(value)!;
  }
  const core = createSQLOperations(domain, contracts, descriptors);
  return Object.freeze({
    async open(variable: unknown, maxConnections: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      denyLiveBoundary(context, origin);
      if (typeof variable !== "string") throw new TypeError("invalid compiler sql variable");
      if (!variableName.test(variable)) return missingCredential(variable);
      // The credential is read exactly once, by exact name: no enumeration,
      // no logging, and no reread for later operations on this pool.
      const url = lookup(variable);
      if (url === undefined) return missingCredential(variable);
      const checked = postgresMaxConnections(maxConnections, failures);
      if (!checked.ok) return checked.failure;
      const opened = await openPostgresClient(url, checked.max, failures);
      if (!opened.ok) return opened.failure;
      const client = opened.client;
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
      return core.queryOne(descriptor, plan, pool, "sql-pool", params, context);
    },
    async queryOptional(descriptor: SQLDescriptor, plan: SQLPlan, pool: unknown, params: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      return core.queryOptional(descriptor, plan, pool, "sql-pool", params, context);
    },
    async queryRows(descriptor: SQLDescriptor, plan: SQLPlan, pool: unknown, params: unknown, maxRows: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      return core.queryRows(descriptor, plan, pool, "sql-pool", params, maxRows, context);
    },
    async execute(descriptor: SQLDescriptor, plan: SQLPlan, pool: unknown, params: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      return core.execute(descriptor, plan, pool, "sql-pool", params, context);
    },
  });
}
