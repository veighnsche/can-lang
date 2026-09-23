// Scoped SQL transactions over one native pool.begin call. The compiler
// binds with_transaction to a concrete result type plus commit/rollback
// leaf identities, and binds in-transaction queries to the same checked
// descriptors and shared plans as pool queries. This module owns the
// transaction scope: exactly one live attempt per dynamic extent, a
// private owner-scoped handle, drain-before-commit, rollback through a
// private identity-checked sentinel, and commit-unknown classification
// when the native layer rejects after a commit decision. Query validation,
// decoding, and native classification are the shared pool core, never a
// parallel transaction subsystem.
import { AsyncLocalStorage } from "node:async_hooks";
import { success, invoke, type Completion, type AssertionContext } from "../../completion.ts";
import { denyLiveBoundary } from "../../assert/context.ts";
import { record, dataProperty, recordIdentity } from "../../data.ts";
import { createDomainRuntime } from "../../domain.ts";
import { resourceStateFailure } from "../../failure.ts";
import { registerResource, useResource, withScope, guardCallback, resourceStatus } from "../../owner.ts";
import { createSQLOperations, poolDialect, type SQLNative, type SQLDialect } from "./pool.ts";
import { createSQLDescriptors, type SQLDescriptor } from "./descriptor.ts";
import { createSQLFailures, type SQLTxContracts } from "./errors.ts";
import { isPostgresFailure } from "./postgres.ts";
import { isSQLiteFailure } from "./sqlite.ts";
import { isMySQLFailure } from "./mysql.ts";
import type { SQLPlan } from "./values.ts";

const origin = Object.freeze({ source: "can:sql-transaction", start: 0, end: 0, invocation: Object.freeze([]) });
type Descriptors = ReturnType<typeof createSQLDescriptors>;
type Contracts = SQLTxContracts;
type Attempt = { readonly id: string };
const nesting = new AsyncLocalStorage<Attempt>();
type TxState = { readonly pool: unknown; readonly id: string };
const handles = new WeakMap<object, TxState>();
const attempts = new WeakMap<object, number>();
const object = (value: unknown): value is object => value !== null && (typeof value === "object" || typeof value === "function");
export function isSQLTransactionValue(kind: string | undefined, value: unknown): boolean {
  return kind === "transaction" && object(value) && handles.has(value);
}

// The rollback sentinel is private to this module and matched by identity
// outside begin. A standard failure is never mistaken for it: only these
// two classes divert the begin outcome, and the primary completion a
// TxPrimary carries is returned verbatim.
class TxRollback {
  readonly value: unknown;
  constructor(value: unknown) { this.value = value; }
}
class TxPrimary {
  readonly completion: Completion<never>;
  constructor(completion: Completion<never>) { this.completion = completion; }
}
type TxCallable = (handle: unknown, context?: AssertionContext) => Promise<Completion<unknown>>;

// Static cleanup template: one frozen literal, never interpolated.
const rollbackText = "ROLLBACK";
const rollbackStrings = Object.freeze(Object.assign([rollbackText], { raw: Object.freeze([rollbackText]) })) as unknown as TemplateStringsArray;

export function createSQLTransactions(
  domain: ReturnType<typeof createDomainRuntime>,
  contracts: Contracts,
  descriptors: Descriptors,
) {
  const core = createSQLOperations(domain, contracts, descriptors, (token, kind) => {
    if (kind !== "sql-tx" || !object(token)) return poolDialect(token);
    const state = handles.get(token);
    return state === undefined ? undefined : poolDialect(state.pool);
  });
  const failures = createSQLFailures(domain, contracts, origin);
  const fail = failures.fail;
  const transactionFailed = (phase: string) => fail(contracts.transactionFailed, [["phase", phase]]);
  // A natively refused begin on an open pool reports the connection error,
  // exactly like the same native event on the query path; the begin phase
  // names the site. Any other native begin failure is transactional.
  function classifyBegin(phase: "begin" | "callback", cause: unknown, dialect: SQLDialect): Completion<never> {
    if (dialect === "sqlite") {
      if (!isSQLiteFailure(cause)) throw cause;
      if (cause.code === "ERR_SQLITE_CONNECTION_CLOSED") {
        return failures.connectionFailed(phase);
      }
      return transactionFailed(phase);
    }
    if (dialect === "mysql") {
      if (!isMySQLFailure(cause)) throw cause;
      if (cause.code === "ERR_MYSQL_CONNECTION_REFUSED" || cause.code === "ERR_MYSQL_CONNECTION_CLOSED") {
        return failures.connectionFailed(phase);
      }
      return transactionFailed(phase);
    }
    if (!isPostgresFailure(cause)) throw cause;
    if (cause.code === "ERR_POSTGRES_CONNECTION_REFUSED" || cause.code === "ERR_POSTGRES_CONNECTION_CLOSED") {
      return failures.connectionFailed(phase);
    }
    return transactionFailed(phase);
  }
  return Object.freeze({
    async queryOne(descriptor: SQLDescriptor, plan: SQLPlan, handle: unknown, params: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      return core.queryOne(descriptor, plan, handle, "sql-tx", params, context);
    },
    async queryOptional(descriptor: SQLDescriptor, plan: SQLPlan, handle: unknown, params: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      return core.queryOptional(descriptor, plan, handle, "sql-tx", params, context);
    },
    async queryRows(descriptor: SQLDescriptor, plan: SQLPlan, handle: unknown, params: unknown, maxRows: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      return core.queryRows(descriptor, plan, handle, "sql-tx", params, maxRows, context);
    },
    async execute(descriptor: SQLDescriptor, plan: SQLPlan, handle: unknown, params: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      return core.execute(descriptor, plan, handle, "sql-tx", params, context);
    },
    async withTransaction(pool: unknown, callback: unknown, leaves: unknown, context?: AssertionContext): Promise<Completion<unknown>> {
      denyLiveBoundary(context, origin);
      if (typeof callback !== "function") throw new TypeError("invalid compiler sql callback");
      const commit = (leaves as { commit?: unknown } | null)?.commit;
      const rollback = (leaves as { rollback?: unknown } | null)?.rollback;
      if (typeof commit !== "string" || commit === "" || typeof rollback !== "string" || rollback === "" || commit === rollback) {
        throw new TypeError("invalid compiler sql decision leaves");
      }
      // Nested entry is rejected through the scope contract: one live
      // transaction per dynamic extent, whatever pool it names.
      if (nesting.getStore() !== undefined) throw resourceStateFailure(undefined, origin);
      return useResource(pool, "sql-pool", async (native: unknown): Promise<Completion<unknown>> => {
        const client = native as SQLNative;
        const attempt = (object(pool) ? attempts.get(pool) ?? 0 : 0) + 1;
        if (object(pool)) attempts.set(pool, attempt);
        // Structured and safe: the pool resource id plus a per-pool attempt
        // counter. No driver payload, URL, or credential ever enters the ID.
        const id = `${resourceStatus(pool).id}-${attempt}`;
        return nesting.run({ id }, async (): Promise<Completion<unknown>> => {
          let began = false;
          let enteredCommit = false;
          try {
            // The native callback stays pending through scope drain: commit
            // returns normally only after registered owners settle.
            const value = await client.begin(async (tx: unknown) => {
              began = true;
              const outcome = await withScope(async (scope): Promise<Completion<unknown>> => {
                // The handle is scope-managed: drain closes it without a cleanup
                // mark, and any later use fails the scope check.
                const handle = registerResource("sql-tx", tx, async () => success(undefined), { scopeManaged: true });
                handles.set(handle, { pool, id });
                const guarded = guardCallback(scope, callback as TxCallable);
                const completed = await invoke(() => guarded(handle, undefined), origin);
                if (completed.kind !== "ok") return completed;
                const identity = recordIdentity(completed.value);
                if (identity === commit) return success({ commit: true as const, value: dataProperty(completed.value, "value") });
                if (identity === rollback) return success({ commit: false as const, value: dataProperty(completed.value, "value") });
                throw new TypeError("invalid compiler sql decision");
              });
              if (outcome.kind !== "ok") throw new TxPrimary(outcome as Completion<never>);
              const boxed = outcome.value as { commit: boolean; value: unknown };
              if (!boxed.commit) throw new TxRollback(boxed.value);
              enteredCommit = true;
              return boxed.value;
            });
            return success(value);
          } catch (cause) {
            if (cause instanceof TxRollback) return success(cause.value);
            if (cause instanceof TxPrimary) return cause.completion;
            // A native rejection after a commit decision is genuinely
            // unknown: the commit may have landed. Report it once, with the
            // safe attempt ID, and never retry.
            if (enteredCommit) {
              // SQLite leaves a failed COMMIT's transaction open: the
              // violating row stays readable until ROLLBACK, which would
              // poison the pool for every later operation. PostgreSQL
              // and MySQL abort a failed COMMIT on their own, so only
              // SQLite pays for this cleanup.
              // It runs under the held pool lease; a dead connection
              // rejects here too, and the outcome stays commit-unknown.
              if (isSQLiteFailure(cause)) {
                try { await client(rollbackStrings); } catch { /* already reported below */ }
              }
              return fail(contracts.commitUnknown, [["transaction_id", id]]);
            }
            return classifyBegin(began ? "callback" : "begin", cause, poolDialect(pool) ?? "postgresql");
          }
        });
      });
    },
  });
}
