// H08 run report: one JSON artifact per eval run. Quality counts as
// evidence ONLY for live-provenanced runs; stubbed runs are plumbing
// checks and say so on the face of the report (no shape-to-quality
// inference, no silent promotion of fixture verdicts). Reports carry
// case ids and labels, never ticket text, credentials, or endpoints.
import { mkdir, writeFile } from "node:fs/promises";
import { dirname } from "node:path";
import type { TriageRegistration } from "./protocol.ts";
import type { CaseRecord, RunSummary } from "./runner.ts";

export type RunProvenance = "stub" | "live";

export type EvalReport = Readonly<{
  task: string;
  experiment: string;
  workload: string;
  protocolVersion: string;
  provenance: RunProvenance;
  evidence: boolean;
  generatedAt: string;
  runId: string;
  setName: string;
  caseSetSha256: string;
  model: Readonly<{ provider: string; model: string; version: string }>;
  ledgerBackend: string;
  tenant: string;
  pool: string;
  testUpperBound: number | null;
  quality: Readonly<{
    evidence: boolean;
    note: string;
    correct: number;
    incorrect: number;
    abstain: number;
    invalidOutput: number;
    rejected: number;
    accuracy: number | null;
    pass: boolean;
  }>;
  cost: Readonly<{ tokensIn: number; tokensOut: number; usd: number | null; usdNote: string }>;
  latency: Readonly<{ maxMs: number; note: string }>;
  budget: Readonly<{
    ledgerCommitted: number;
    ledgerExpected: number;
    ledgerMatch: boolean;
    held: number;
    unresolved: number;
    correlationOk: boolean;
  }>;
  records: readonly CaseRecord[];
  limitations: readonly string[];
}>;

export function assembleReport(
  input: Readonly<{
    registration: TriageRegistration;
    runId: string;
    setName: string;
    caseSetSha256: string;
    provenance: RunProvenance;
    backend: string;
    tenant: string;
    pool: string;
    testUpperBound: number | null;
    records: readonly CaseRecord[];
    summary: RunSummary;
    spentUsd: number | null;
    generatedAt?: string;
  }>,
): EvalReport {
  const live = input.provenance === "live";
  return {
    task: input.registration.task,
    experiment: input.registration.experiment,
    workload: input.registration.workload,
    protocolVersion: input.registration.protocolVersion,
    provenance: input.provenance,
    evidence: live,
    generatedAt: input.generatedAt ?? new Date().toISOString(),
    runId: input.runId,
    setName: input.setName,
    caseSetSha256: input.caseSetSha256,
    model: {
      provider: input.registration.pinned.provider,
      model: input.registration.pinned.model,
      version: input.registration.pinned.version,
    },
    ledgerBackend: input.backend,
    tenant: input.tenant,
    pool: input.pool,
    testUpperBound: input.testUpperBound,
    quality: {
      evidence: live,
      note: live
        ? "Live provider answers judged against frozen labels."
        : "Stub-scripted answers: pipeline plumbing only, not a quality claim.",
      correct: input.summary.correct,
      incorrect: input.summary.incorrect,
      abstain: input.summary.abstain,
      invalidOutput: input.summary.invalidOutput,
      rejected: input.summary.rejected,
      accuracy: live ? input.summary.accuracy : null,
      pass: live && input.summary.pass,
    },
    cost: {
      tokensIn: input.summary.tokensIn,
      tokensOut: input.summary.tokensOut,
      usd: input.spentUsd,
      usdNote:
        input.spentUsd === null
          ? "No price table pinned: USD cost unknown."
          : "USD via the pinned price table.",
    },
    latency: {
      maxMs: input.summary.latencyMaxMs,
      note: "Wall clock around guarded dispatch per case; reported only, no SLO.",
    },
    budget: {
      ledgerCommitted: input.summary.ledger.committed,
      ledgerExpected: input.summary.ledger.expectedSettledSum,
      ledgerMatch: input.summary.ledger.match,
      held: input.summary.ledger.held,
      unresolved: input.summary.ledger.unresolved,
      correlationOk: input.summary.correlationOk,
    },
    records: input.records,
    limitations: live
      ? [
          "Single serial run; no sustained-load or concurrency claim.",
          "Latency is wall clock on the eval host, not a provider SLO.",
        ]
      : [
          "Stubbed provider: quality/cost/latency figures are plumbing checks, not evidence.",
          "Test upper bound is not a qualified metering bound.",
        ],
  };
}

export async function writeReport(path: string, report: EvalReport): Promise<void> {
  await mkdir(dirname(path), { recursive: true });
  await writeFile(path, `${JSON.stringify(report, null, 2)}\n`, "utf8");
}
