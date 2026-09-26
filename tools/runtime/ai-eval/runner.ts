// H08 serial triage runner: one guarded SystemOne call per frozen
// case, strictly one at a time, inside ai_budget::within scope. Each
// case yields a verdict (correct/incorrect/abstain/invalid-output/
// rejected) plus latency, token usage reconciled from the durable
// ledger, and the accounting reports it produced. The runner never
// infers quality from response shape: verdicts compare the decoded
// choice against the frozen label, and quality counts as evidence
// only when report.ts marks the run live-provenanced.
import { dataProperty } from "../../../runtime/data.ts";
import { domainFailureDiagnostics } from "../../../runtime/domain.ts";
import type { Completion } from "../../../runtime/completion.ts";
import type { Answer } from "../../../runtime/ai/typesafe.ts";
import { epochSchedule } from "../../../runtime/outbound/epoch.ts";
import {
  configurePool,
  epochStatus,
  reconcile,
  type LedgerStore,
} from "../../../runtime/outbound/ledger.ts";
import type { SpendTracker } from "./live-gate.ts";
import { caseState, type TriageCase, type TriageProtocol } from "./protocol.ts";
import { EVAL_OPERATION, EVAL_ORIGIN, type EvalDomain, type EvalPipeline } from "./setup.ts";

export type CaseVerdict = "correct" | "incorrect" | "abstain" | "invalid-output" | "rejected";

export type CaseReport = Readonly<{
  invocation: string;
  outcome: string;
  actual?: number;
  released?: number;
  duplicate?: boolean;
}>;

export type CaseRecord = Readonly<{
  caseId: string;
  label: string;
  verdict: CaseVerdict;
  reason?: string;
  choice?: string;
  confidence?: number;
  latencyMs: number;
  scopeCorrelation?: string;
  correlationOk: boolean;
  inputTokens?: number;
  outputTokens?: number;
  reports: readonly CaseReport[];
}>;

export type LedgerCheck = Readonly<{
  committed: number;
  expectedSettledSum: number;
  held: number;
  unresolved: number;
  match: boolean;
}>;

export type RunSummary = Readonly<{
  runId: string;
  setName: string;
  cases: number;
  correct: number;
  incorrect: number;
  abstain: number;
  invalidOutput: number;
  rejected: number;
  accuracy: number | null;
  tokensIn: number;
  tokensOut: number;
  latencyMaxMs: number;
  ledger: LedgerCheck;
  correlationOk: boolean;
  stoppedAtCap: boolean;
  pass: boolean;
}>;

export type RunInput = Readonly<{
  eval: EvalDomain;
  protocol: TriageProtocol;
  setName: string;
  cases: readonly TriageCase[];
  pipeline: EvalPipeline;
  store: LedgerStore;
  tenant: string;
  pool: string;
  runId: string;
  epoch: Readonly<{ limit: number; periodMs: number; anchorMs: number; version: string }>;
  nowMs?: () => number;
  spend?: SpendTracker;
}>;

function failureName(completion: Completion<unknown>, fallback: string): string {
  if (completion.kind !== "domain") return fallback;
  try {
    return domainFailureDiagnostics(completion.value).declaration.name;
  } catch {
    return fallback;
  }
}

function failureField(completion: Completion<unknown>, field: string): string | undefined {
  if (completion.kind !== "domain") return undefined;
  try {
    const payload = domainFailureDiagnostics(completion.value).payload;
    const value = dataProperty(payload, field);
    return typeof value === "string" ? value : undefined;
  } catch {
    return undefined;
  }
}

function diagnose(
  completion: Completion<readonly Answer[]>,
  label: string,
  floor: number,
): { verdict: CaseVerdict; reason?: string; choice?: string; confidence?: number } {
  if (completion.kind === "ok") {
    const answers = completion.value;
    if (!Array.isArray(answers) || answers.length !== 1 || answers[0]?.kind !== "choice")
      return { verdict: "invalid-output", reason: "unexpected-answer-shape" };
    const answer = answers[0];
    if (answer.confidence < floor)
      return { verdict: "abstain", choice: answer.choice, confidence: answer.confidence };
    if (answer.choice === label)
      return { verdict: "correct", choice: answer.choice, confidence: answer.confidence };
    return { verdict: "incorrect", choice: answer.choice, confidence: answer.confidence };
  }
  const name = failureName(completion, "unknown-failure");
  if (name === "can.std.ai_budget@1::exceeded") return { verdict: "rejected", reason: "exceeded" };
  if (name === "can.std.ai_budget@1::unavailable")
    return {
      verdict: "rejected",
      reason: `unavailable-${failureField(completion, "reason") ?? "unknown"}`,
    };
  if (name.endsWith("ai::invalid_answer"))
    return {
      verdict: "invalid-output",
      reason: failureField(completion, "reason") ?? "invalid-answer",
    };
  return { verdict: "rejected", reason: name };
}

export async function runSerial(
  input: RunInput,
): Promise<{ records: CaseRecord[]; summary: RunSummary }> {
  const now = input.nowMs ?? (() => Date.now());
  const runStarted = now();
  const configured = await configurePool(
    input.store,
    epochSchedule({
      pool: input.pool,
      limit: input.epoch.limit,
      periodMs: input.epoch.periodMs,
      anchorMs: input.epoch.anchorMs,
      version: input.epoch.version,
    }),
  );
  if (!("configured" in configured) || configured.configured !== true)
    throw new TypeError("eval ledger pool already configured: use a fresh run database or pool");
  const records: CaseRecord[] = [];
  let stoppedAtCap = false;
  // Strictly serial: one provider call at a time, awaited before the
  // next case starts. No batching, no Promise.all over dispatches.
  for (const triage of input.cases) {
    if (input.spend?.exhausted()) {
      stoppedAtCap = true;
      break;
    }
    let state: unknown;
    try {
      state = caseState(triage, input.protocol.registration.stateCaps);
    } catch {
      records.push({
        caseId: triage.id,
        label: triage.label,
        verdict: "rejected",
        reason: "input-over-cap",
        latencyMs: 0,
        correlationOk: true,
        reports: [],
      });
      continue;
    }
    const before = input.pipeline.reports.length;
    const started = now();
    let completion: Completion<readonly Answer[]>;
    let scopeCorrelation: string | undefined;
    try {
      completion = await input.pipeline.guard.within(
        input.tenant,
        input.pool,
        (context) => {
          scopeCorrelation = context.correlation;
          return input.pipeline.ask(state, [input.protocol.question], context);
        },
        EVAL_ORIGIN,
        EVAL_OPERATION,
      );
    } catch {
      records.push({
        caseId: triage.id,
        label: triage.label,
        verdict: "rejected",
        reason: "harness-throw",
        latencyMs: now() - started,
        correlationOk: true,
        reports: [],
      });
      continue;
    }
    const latencyMs = now() - started;
    const freshEntries = input.pipeline.reports.slice(before);
    // The scope mints one correlation per case; every accounting
    // report the case produced must carry it (pre-admission
    // rejections produce no reports and pass vacuously).
    const caseCorrelationOk =
      scopeCorrelation !== undefined &&
      freshEntries.every((entry) => entry.correlation === scopeCorrelation);
    const fresh = freshEntries.map((entry): CaseReport => ({
      invocation: entry.invocation,
      outcome: entry.outcome,
      ...(entry.actual === undefined ? {} : { actual: entry.actual }),
      ...(entry.released === undefined ? {} : { released: entry.released }),
      ...(entry.duplicate === undefined ? {} : { duplicate: entry.duplicate }),
    }));
    // Token splits reconcile from the durable ledger by invocation
    // id — never from the report alone or the provider body.
    let inputTokens: number | undefined;
    let outputTokens: number | undefined;
    for (const entry of fresh) {
      const found = await reconcile(input.store, entry.invocation);
      if (found.status === "found" && found.record.usage !== undefined) {
        inputTokens = (inputTokens ?? 0) + found.record.usage.inputTokens;
        outputTokens = (outputTokens ?? 0) + found.record.usage.outputTokens;
      }
      if (entry.outcome === "settled" && entry.actual !== undefined && entry.duplicate !== true)
        input.spend?.charge(entry.actual);
    }
    const judged = diagnose(
      completion,
      triage.label,
      input.protocol.registration.abstainConfidenceFloor,
    );
    records.push({
      caseId: triage.id,
      label: triage.label,
      verdict: judged.verdict,
      ...(judged.reason === undefined ? {} : { reason: judged.reason }),
      ...(judged.choice === undefined ? {} : { choice: judged.choice }),
      ...(judged.confidence === undefined ? {} : { confidence: judged.confidence }),
      latencyMs,
      ...(scopeCorrelation === undefined ? {} : { scopeCorrelation }),
      correlationOk: caseCorrelationOk,
      ...(inputTokens === undefined ? {} : { inputTokens }),
      ...(outputTokens === undefined ? {} : { outputTokens }),
      reports: fresh,
    });
  }
  const count = (verdict: CaseVerdict): number =>
    records.filter((r) => r.verdict === verdict).length;
  const correct = count("correct");
  const incorrect = count("incorrect");
  const abstain = count("abstain");
  const invalidOutput = count("invalid-output");
  const rejected = count("rejected");
  const decided = correct + incorrect;
  const tokensIn = records.reduce((sum, r) => sum + (r.inputTokens ?? 0), 0);
  const tokensOut = records.reduce((sum, r) => sum + (r.outputTokens ?? 0), 0);
  // The check reads this run's own epoch row — never a store-wide
  // aggregate — so shared ledgers cannot pollute the verdict.
  const runEpochStart =
    input.epoch.anchorMs +
    Math.floor((runStarted - input.epoch.anchorMs) / input.epoch.periodMs) * input.epoch.periodMs;
  const status = await epochStatus(input.store, input.tenant, input.pool, runEpochStart);
  const expectedSettledSum = records.reduce(
    (sum, r) =>
      sum +
      r.reports.reduce(
        (inner, e) =>
          inner + (e.outcome === "settled" && e.duplicate !== true ? (e.actual ?? 0) : 0),
        0,
      ),
    0,
  );
  // No admission means no epoch row exists; with nothing settled
  // that absence matches vacuously, otherwise it fails.
  const ledger: LedgerCheck =
    status.status === "found"
      ? {
          committed: status.row.committed,
          expectedSettledSum,
          held: status.row.held,
          unresolved: status.unresolved,
          match: status.row.committed === expectedSettledSum,
        }
      : status.status === "absent" && expectedSettledSum === 0
        ? { committed: 0, expectedSettledSum, held: 0, unresolved: 0, match: true }
        : { committed: -1, expectedSettledSum, held: -1, unresolved: -1, match: false };
  const correlationOk = records.every((record) => record.correlationOk);
  const pass = input.protocol.registration.thresholds.pass;
  const summary: RunSummary = {
    runId: input.runId,
    setName: input.setName,
    cases: records.length,
    correct,
    incorrect,
    abstain,
    invalidOutput,
    rejected,
    accuracy: decided === 0 ? null : correct / decided,
    tokensIn,
    tokensOut,
    latencyMaxMs: records.reduce((max, r) => Math.max(max, r.latencyMs), 0),
    ledger,
    correlationOk,
    stoppedAtCap,
    pass:
      records.length === pass.of &&
      correct >= pass.correctMin &&
      abstain <= pass.abstainMax &&
      invalidOutput <= pass.invalidOutputMax &&
      rejected <= pass.rejectedMax &&
      ledger.match &&
      correlationOk &&
      !stoppedAtCap,
  };
  return { records, summary };
}
