// H08 eval pipeline: one guarded SystemOne connection over a durable
// metered ledger. Every case dispatch reserves through the H07 budget
// guard before sending and settles authoritative usage after; there
// is no unguarded call path. Memory ledgers are refused for metered
// runs (F04 disqualified process-local backends); PG is the default,
// file/SQLite-file the documented alternates.
import { createHash } from "node:crypto";
import {
  createBudgetGuard,
  createProfileRegistry,
  guardConnection,
  type AccountingReport,
  type BudgetGuard,
  type BudgetTypes,
  type ProfileRegistry,
} from "../../../runtime/ai/budget.ts";
import { createTypeSafe, type AITypes, type Answer } from "../../../runtime/ai/typesafe.ts";
import { catalogue } from "../../../runtime/catalogue.ts";
import type { Completion } from "../../../runtime/completion.ts";
import {
  concreteTypeDigestInput,
  createDomainRuntime,
  type FailureShape,
} from "../../../runtime/domain.ts";
import type { FailureOrigin } from "../../../runtime/failure.ts";
import { runOwnedRoot } from "../../../runtime/owner.ts";
import { openFileLedgerStore } from "../../../runtime/outbound/file-ledger.ts";
import { invocationContext } from "../../../runtime/outbound/identity.ts";
import { ledgerHealth, type LedgerStore } from "../../../runtime/outbound/ledger.ts";
import {
  applyLedgerDDL,
  openSqlLedgerStore,
  type SqlLedgerStore,
} from "../../../runtime/outbound/sql-ledger.ts";
import type { Schema } from "../../../runtime/codec/json.ts";
import type { QuestionDescriptor } from "../../../runtime/ai/questions.ts";
import type { Connection } from "../../../runtime/transport/request.ts";

export const EVAL_OPERATION = "h08:triage/evaluate";
export const EVAL_ORIGIN: FailureOrigin = Object.freeze({
  source: "h08:triage-eval",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

const hash = (key: unknown): string =>
  createHash("sha256").update(concreteTypeDigestInput(key)).digest("hex");

const shape = (
  kind: string,
  declaration: string,
  fields: { name: string; type: string }[] = [],
): FailureShape => ({
  identity: hash([kind, declaration]),
  kind,
  declaration,
  fields,
  arguments: [],
  leaves: [],
  inputs: [],
  errors: [],
});

export type EvalDomain = Readonly<{
  domain: ReturnType<typeof createDomainRuntime>;
  aiTypes: AITypes;
  budgetTypes: BudgetTypes;
}>;

// createEvalDomain builds the failure-type universe the guarded
// adapter and budget guard share: transport failures, AI question/
// answer failures, and the ai_budget exceeded/unavailable heads.
export function createEvalDomain(): EvalDomain {
  const str = shape("primitive", "str");
  const int = shape("primitive", "int");
  const header = shape("record", "can.std.http@1::header", [
    { name: "name", type: str.identity },
    { name: "value", type: str.identity },
  ]);
  const headers: FailureShape = {
    ...shape("array", ""),
    identity: hash(["array", "", header.identity]),
    element: header.identity,
  };
  const wanted = [
    "http::invalid_request",
    "http::credentials_missing",
    "http::transport_failed",
    "http::timeout",
    "http::body_limit",
    "http::status_error",
    "http::request_failed",
    "codec::invalid_data",
    "ai::invalid_question",
    "ai::invalid_answer",
  ];
  const declarations = catalogue.errors.filter((entry) => wanted.includes(entry.name));
  const detailIdentity = hash(["variant", "can.std.http@1::failure_detail"]);
  const errors = declarations.map((entry) =>
    shape(
      "error",
      entry.identity,
      entry.fields.map((field) => ({
        name: field.name,
        type:
          field.type === "str"
            ? str.identity
            : field.type === "int"
              ? int.identity
              : field.type === "http::failure_detail"
                ? detailIdentity
                : headers.identity,
      })),
    ),
  );
  const detail: FailureShape = {
    identity: detailIdentity,
    kind: "variant",
    declaration: "can.std.http@1::failure_detail",
    fields: [],
    arguments: [],
    leaves: errors
      .filter(
        (_, index) =>
          !["http::request_failed", "ai::invalid_question", "ai::invalid_answer"].includes(
            declarations[index].name,
          ),
      )
      .map((entry) => entry.identity),
    inputs: [],
    errors: [],
  };
  const exceededShape = shape("error", "can.project.ai_budget.exceeded", [
    { name: "pool", type: str.identity },
    { name: "required", type: int.identity },
    { name: "remaining", type: int.identity },
    { name: "epoch_reset_ms", type: int.identity },
  ]);
  const unavailableShape = shape("error", "can.project.ai_budget.unavailable", [
    { name: "reason", type: str.identity },
    { name: "correlation", type: str.identity },
    { name: "invocation", type: str.identity },
  ]);
  const domain = createDomainRuntime({
    declarations: [
      ...declarations.map((entry) => ({ ...entry, parameters: 0 })),
      {
        identity: "can.project.ai_budget.exceeded",
        name: "can.std.ai_budget@1::exceeded",
        parameters: 0,
      },
      {
        identity: "can.project.ai_budget.unavailable",
        name: "can.std.ai_budget@1::unavailable",
        parameters: 0,
      },
    ],
    shapes: [str, int, header, headers, ...errors, detail, exceededShape, unavailableShape],
  });
  const byName = new Map(declarations.map((entry, index) => [entry.name, errors[index].identity]));
  const http = {
    invalid: byName.get("http::invalid_request")!,
    credential: byName.get("http::credentials_missing")!,
    transport: byName.get("http::transport_failed")!,
    timeout: byName.get("http::timeout")!,
    limit: byName.get("http::body_limit")!,
    status: byName.get("http::status_error")!,
    header: header.identity,
  };
  return {
    domain,
    aiTypes: {
      ...http,
      invalidData: byName.get("codec::invalid_data")!,
      invalidQuestion: byName.get("ai::invalid_question")!,
      invalidAnswer: byName.get("ai::invalid_answer")!,
      failed: byName.get("http::request_failed")!,
    },
    budgetTypes: { exceeded: exceededShape.identity, unavailable: unavailableShape.identity },
  };
}

export type MeteredLedgerSpec =
  | Readonly<{ kind: "pg"; url: string }>
  | Readonly<{ kind: "file"; path: string }>
  | Readonly<{ kind: "sqlite"; filename: string }>;

export type MeteredLedger = Readonly<{
  store: LedgerStore;
  close: () => Promise<void>;
  backend: string;
}>;

// expandDbUrl resolves a <db>-template provision URL (F04 convention)
// to one concrete per-run database URL.
export function expandDbUrl(
  template: string | undefined,
  scheme: string,
  db: string,
): string | undefined {
  if (template === undefined || template === "") return undefined;
  if (!template.startsWith(scheme)) throw new Error(`refusing non-${scheme} ledger URL`);
  return template.includes("<db>") ? template.replace("<db>", db) : template;
}

async function openSqlWithDdl(
  target: { dialect: "sqlite"; filename: string } | { dialect: "postgres"; url: string },
): Promise<SqlLedgerStore> {
  // Disposable run databases only: apply the operator DDL when the
  // schema is absent, reuse it when a previous run created it. The
  // probe distinguishes missing schema (storage-unavailable) from a
  // live store; anything else fails closed in open.
  const probe = await openSqlLedgerStore(target);
  const health = await ledgerHealth(probe);
  if (health.status === "ok") return probe;
  await probe.close();
  await applyLedgerDDL(target);
  return openSqlLedgerStore(target);
}

export async function openMeteredLedger(spec: MeteredLedgerSpec): Promise<MeteredLedger> {
  if (spec.kind === "file")
    return { store: openFileLedgerStore(spec.path), close: async () => undefined, backend: "file" };
  if (spec.kind === "sqlite") {
    const store = await openSqlWithDdl({ dialect: "sqlite", filename: spec.filename });
    return { store, close: () => store.close(), backend: "sqlite-file" };
  }
  const store = await openSqlWithDdl({ dialect: "postgres", url: spec.url });
  return { store, close: () => store.close(), backend: "postgres" };
}

export type EvalPipeline = Readonly<{
  guard: BudgetGuard;
  registry: ProfileRegistry;
  reports: AccountingReport[];
  ask: (
    state: unknown,
    questions: readonly QuestionDescriptor[],
    budgetContext: { tenant: string; pool: string; correlation: string },
  ) => Promise<Completion<readonly Answer[]>>;
}>;

export type PipelineInput = Readonly<{
  eval: EvalDomain;
  store: LedgerStore;
  connection: Connection;
  stateSchema: Schema;
  model: string;
  pinned: Readonly<{ provider: string; model: string; version: string }>;
  pool: string;
  readEnvironment: (name: string) => string | undefined;
  nowMs?: () => number;
  // testUpperBound qualifies the pinned profile inside this pipeline
  // ONLY so offline plumbing can run end to end. It is never a bound
  // qualification: X-R14-1 needs a real complete-call proof, and every
  // run carrying a test bound is labeled non-evidence.
  // upperBound qualifies the pinned profile inside this pipeline.
  // testOnly bounds exist so offline plumbing can run end to end and
  // are NEVER a qualification: every run carrying one is labeled
  // non-evidence. A live bound arrives only from a re-registration
  // whose X-R14-1 proof qualified it.
  upperBound?: Readonly<{ value: number; testOnly: boolean }>;
}>;

export function createPipeline(input: PipelineInput): EvalPipeline {
  const registry = createProfileRegistry();
  if (input.upperBound !== undefined)
    registry.qualify({ ...input.pinned, upperBound: input.upperBound.value });
  const reports: AccountingReport[] = [];
  const guard = createBudgetGuard(input.eval.domain, input.eval.budgetTypes, {
    store: input.store,
    registry,
    nowMs: input.nowMs,
    report: (entry) => reports.push(entry),
  });
  const binding = guardConnection(guard, { permittedPool: input.pool, profile: input.pinned });
  const judge = createTypeSafe(
    input.eval.domain,
    input.eval.aiTypes,
    input.readEnvironment,
    binding,
  );
  return {
    guard,
    registry,
    reports,
    ask: async (state, questions, budgetContext) => {
      const { completion } = await runOwnedRoot(() =>
        judge.ask(
          input.connection,
          input.model,
          input.stateSchema,
          state,
          questions,
          EVAL_ORIGIN,
          EVAL_OPERATION,
          undefined,
          invocationContext(budgetContext.tenant, budgetContext.pool, budgetContext.correlation),
        ),
      );
      return completion;
    },
  };
}
