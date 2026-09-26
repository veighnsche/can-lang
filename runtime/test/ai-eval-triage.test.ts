// H08 triage eval tests: the frozen protocol plus the serial guarded
// runner against a scripted loopback stub. No live provider calls.
// Stubbed verdicts exercise pipeline plumbing only; quality counts
// as evidence solely for live-provenanced runs (see report.ts).
import { expect, test } from "bun:test";
import { copyFile, mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  authorizeLive,
  CREDENTIAL_ENV,
  createSpendTracker,
  SPEND_CAP_ENV,
} from "../../tools/runtime/ai-eval/live-gate.ts";
import {
  caseState,
  loadProtocol,
  pricePerToken,
  qualifiedBound,
  stateSchema,
  type TriageCase,
  type TriageProtocol,
  type TriageRegistration,
} from "../../tools/runtime/ai-eval/protocol.ts";
import { assembleReport } from "../../tools/runtime/ai-eval/report.ts";
import { runSerial } from "../../tools/runtime/ai-eval/runner.ts";
import {
  createEvalDomain,
  createPipeline,
  expandDbUrl,
  openMeteredLedger,
  type EvalDomain,
} from "../../tools/runtime/ai-eval/setup.ts";
import { choiceBody, scriptedBodies, serveScripted } from "../../tools/runtime/ai-eval/stub.ts";
import { ledgerHealth } from "../outbound/ledger.ts";
import type { Connection } from "../transport/request.ts";

const HOUR = 3_600_000;
const PROTOCOL_DIR = join(import.meta.dir, "../../tools/runtime/ai-eval/protocol");
const PROTOCOL: TriageProtocol = await loadProtocol(PROTOCOL_DIR);
const KEYS = PROTOCOL.registration.categories.map((category) => category.key);
const PINNED = {
  provider: PROTOCOL.registration.pinned.provider,
  model: PROTOCOL.registration.pinned.model,
  version: PROTOCOL.registration.pinned.version,
};
const WIRE_MODEL = `${PINNED.model}-${PINNED.version}`;

const PG_URL = expandDbUrl(process.env.CAN_TEST_POSTGRES_URL, "postgres", "can_h08_run1");

async function fileLedger() {
  const dir = await mkdtemp(join(tmpdir(), "h08-eval-"));
  const ledger = await openMeteredLedger({ kind: "file", path: join(dir, "ledger.json") });
  return {
    ledger,
    cleanup: async (): Promise<void> => {
      await ledger.close();
      await rm(dir, { recursive: true, force: true });
    },
  };
}

function connection(endpoint: string): Connection {
  return {
    endpoint,
    timeoutMilliseconds: PROTOCOL.registration.pinned.timeoutMs,
    maxBodyBytes: PROTOCOL.registration.pinned.maxBodyBytes,
    headers: [],
  };
}

function pipelineFor(
  evalDomain: EvalDomain,
  store: Parameters<typeof createPipeline>[0]["store"],
  endpoint: string,
  upperBound?: number,
  pool = "default",
) {
  return createPipeline({
    eval: evalDomain,
    store,
    connection: connection(endpoint),
    stateSchema: stateSchema(),
    model: WIRE_MODEL,
    pinned: PINNED,
    pool,
    readEnvironment: () => undefined,
    ...(upperBound === undefined ? {} : { upperBound: { value: upperBound, testOnly: true } }),
  });
}

async function runCases(
  cases: readonly TriageCase[],
  bodies: readonly unknown[],
  options: { upperBound?: number; limit?: number; runId: string; pool?: string } & (
    | { backend: "file" }
    | { backend: "pg"; url: string }
  ),
) {
  const evalDomain = createEvalDomain();
  const stub = serveScripted(bodies);
  const held =
    options.backend === "pg"
      ? await openMeteredLedger({ kind: "pg", url: options.url })
      : await fileLedger().then(async (file) => ({ ...file.ledger, cleanup: file.cleanup }));
  try {
    const pool = options.pool ?? "default";
    const pipeline = pipelineFor(
      evalDomain,
      held.store,
      stub.url("/systemone"),
      options.upperBound,
      pool,
    );
    const { records, summary } = await runSerial({
      eval: evalDomain,
      protocol: PROTOCOL,
      setName: "representative",
      cases,
      pipeline,
      store: held.store,
      tenant: "h08-eval",
      pool,
      runId: options.runId,
      epoch: { limit: options.limit ?? 1_000_000, periodMs: HOUR, anchorMs: 0, version: "1" },
    });
    const health = await ledgerHealth(held.store);
    return {
      records,
      summary,
      health,
      requests: stub.requests(),
      maxConcurrency: stub.maxConcurrency(),
    };
  } finally {
    await held.close();
    if ("cleanup" in held && typeof held.cleanup === "function") await held.cleanup();
    await stub.close();
  }
}

test("protocol loads frozen with verified hashes and valid labels", () => {
  expect(PROTOCOL.registration.protocolVersion).toBe("2");
  expect(PROTOCOL.representative).toHaveLength(12);
  expect(PROTOCOL.heldout).toHaveLength(6);
  const keys = new Set(KEYS);
  for (const triage of [...PROTOCOL.representative, ...PROTOCOL.heldout]) {
    expect(keys.has(triage.label)).toBe(true);
    expect(() => caseState(triage, PROTOCOL.registration.stateCaps)).not.toThrow();
  }
  expect(PROTOCOL.question).toEqual({
    kind: "choice",
    instructions: PROTOCOL.registration.instructions,
    options: PROTOCOL.registration.categories.map((category) => ({
      key: category.key,
      description: category.description,
    })),
  });
});

test("tampered case files refuse to load without re-registration", async () => {
  const dir = await mkdtemp(join(tmpdir(), "h08-tamper-"));
  try {
    await copyFile(join(PROTOCOL_DIR, "registration.json"), join(dir, "registration.json"));
    await copyFile(join(PROTOCOL_DIR, "heldout-cases.json"), join(dir, "heldout-cases.json"));
    const tampered = JSON.parse(
      await Bun.file(join(PROTOCOL_DIR, "representative-cases.json")).text(),
    ) as { cases: { label: string }[] };
    tampered.cases[0].label = "general";
    await writeFile(join(dir, "representative-cases.json"), JSON.stringify(tampered));
    await expect(loadProtocol(dir)).rejects.toThrow("without re-registration");
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
});

test("unqualified profile rejects every case before dispatch", async () => {
  const bodies = scriptedBodies(KEYS, PROTOCOL.representative, 100);
  const { records, summary, health, requests } = await runCases(PROTOCOL.representative, bodies, {
    backend: "file",
    runId: "test-unqualified",
  });
  expect(requests).toBe(0);
  expect(summary.rejected).toBe(12);
  expect(summary.correct).toBe(0);
  for (const record of records) expect(record.reason).toBe("unavailable-missing-qualification");
  expect(health.status).toBe("ok");
  if (health.status === "ok") {
    expect(health.committed).toBe(0);
    expect(health.held).toBe(0);
  }
  expect(summary.ledger.match).toBe(true);
  expect(summary.correlationOk).toBe(true);
});

test("exceeded budget rejects immediately with zero dispatch", async () => {
  const bodies = scriptedBodies(KEYS, PROTOCOL.representative, 100);
  const { records, summary, requests } = await runCases(PROTOCOL.representative, bodies, {
    backend: "file",
    upperBound: 5000,
    limit: 4999,
    runId: "test-exceeded",
  });
  expect(requests).toBe(0);
  expect(summary.rejected).toBe(12);
  for (const record of records) expect(record.reason).toBe("exceeded");
});

test("over-cap input refuses without dispatch", async () => {
  const over: TriageCase = {
    id: "rep-over",
    subject: "x",
    body: `y${"y".repeat(2000)}`,
    label: "general",
  };
  expect(() => caseState(over, PROTOCOL.registration.stateCaps)).toThrow(
    "over registered state cap",
  );
  const bodies = scriptedBodies(KEYS, [over], 100);
  const { records, requests } = await runCases([over], bodies, {
    backend: "file",
    upperBound: 5000,
    runId: "test-over-cap",
  });
  expect(requests).toBe(0);
  expect(records).toHaveLength(1);
  expect(records[0].verdict).toBe("rejected");
  expect(records[0].reason).toBe("input-over-cap");
});

test("scripted run exercises every verdict path serially", async () => {
  const bodies = scriptedBodies(KEYS, PROTOCOL.representative, 100);
  const { records, summary, requests, maxConcurrency } = await runCases(
    PROTOCOL.representative,
    bodies,
    { backend: "file", upperBound: 5000, runId: "test-verdicts" },
  );
  expect(requests).toBe(12);
  expect(maxConcurrency).toBe(1);
  expect({
    correct: summary.correct,
    incorrect: summary.incorrect,
    abstain: summary.abstain,
    invalidOutput: summary.invalidOutput,
    rejected: summary.rejected,
  }).toEqual({ correct: 9, incorrect: 1, abstain: 1, invalidOutput: 1, rejected: 0 });
  const byId = new Map(records.map((record) => [record.caseId, record]));
  expect(byId.get("rep-07")?.verdict).toBe("incorrect");
  expect(byId.get("rep-09")?.verdict).toBe("abstain");
  expect(byId.get("rep-10")?.verdict).toBe("invalid-output");
  // Usage-less reply: provider completion stays intact (judged
  // correct) while the full hold stays unresolved and durable.
  const usageLess = byId.get("rep-11");
  expect(usageLess?.verdict).toBe("correct");
  expect(usageLess?.inputTokens).toBeUndefined();
  expect(usageLess?.reports).toHaveLength(1);
  expect(usageLess?.reports[0].outcome).toBe("unresolved");
  // Extra breakdown fields never double-count: rep-12 settles
  // input+output only despite 9999 cached/reasoning tokens.
  const breakdown = byId.get("rep-12");
  expect(breakdown?.inputTokens).toBe(111);
  expect(breakdown?.outputTokens).toBe(31);
  expect(breakdown?.reports[0].outcome).toBe("settled");
  expect(breakdown?.reports[0].actual).toBe(142);
  expect(summary.tokensIn).toBe(1156);
  expect(summary.tokensOut).toBe(276);
  expect(summary.ledger.match).toBe(true);
  expect(summary.ledger.unresolved).toBe(1);
  expect(summary.correlationOk).toBe(true);
  for (const record of records) expect(record.correlationOk).toBe(true);
});

test("usage over the bound breaches and quarantines the profile", async () => {
  const cases = PROTOCOL.representative.slice(0, 2);
  const bodies = scriptedBodies(KEYS, cases, 10_000);
  const { records, summary } = await runCases(cases, bodies, {
    backend: "file",
    upperBound: 5000,
    runId: "test-breach",
  });
  expect(summary.rejected).toBe(2);
  expect(records[0].reason).toBe("unavailable-breached-profile");
  expect(records[0].reports[0].outcome).toBe("breach");
  // The quarantined profile stays blocked for the next budgeted send.
  expect(records[1].reason).toBe("unavailable-breached-profile");
});

test("live gate names every missing input and validates the cap", () => {
  const priced = (overrides: Partial<TriageRegistration>): TriageRegistration =>
    ({
      ...PROTOCOL.registration,
      ...overrides,
    }) as TriageRegistration;
  const none = authorizeLive({}, PROTOCOL.registration);
  expect(none.authorized).toBe(false);
  if (!none.authorized) expect(none.missing).toHaveLength(4);
  const keyOnly = authorizeLive({ [CREDENTIAL_ENV]: "x" }, PROTOCOL.registration);
  expect(keyOnly.authorized).toBe(false);
  if (!keyOnly.authorized) expect(keyOnly.missing.join(" ")).not.toContain(CREDENTIAL_ENV);
  for (const bad of ["abc", "-5", "0"]) {
    const decided = authorizeLive(
      { [CREDENTIAL_ENV]: "x", [SPEND_CAP_ENV]: bad },
      PROTOCOL.registration,
    );
    expect(decided.authorized).toBe(false);
    if (!decided.authorized) expect(decided.missing.join(" ")).toContain("must parse as USD > 0");
  }
  const capped = authorizeLive(
    { [CREDENTIAL_ENV]: "x", [SPEND_CAP_ENV]: "5" },
    priced({ priceTable: { usdPerToken: 0.000_001 } }),
  );
  expect(capped.authorized).toBe(false);
  if (!capped.authorized)
    expect(capped.missing).toEqual(["qualified complete-call bound U (profile unqualified)"]);
  expect(qualifiedBound(PROTOCOL.registration)).toBeUndefined();
  expect(pricePerToken(PROTOCOL.registration)).toBeUndefined();
  const qualified = priced({
    priceTable: { usdPerToken: 0.000_001 },
    boundStatus: { status: "qualified", detail: "fixture", upperBound: 5000 },
  });
  expect(qualifiedBound(qualified)).toBe(5000);
  expect(pricePerToken(qualified)).toBe(0.000_001);
  const open = authorizeLive({ [CREDENTIAL_ENV]: "x", [SPEND_CAP_ENV]: "5" }, qualified);
  expect(open).toEqual({ authorized: true, spendCapUsd: 5 });
});

test("spend tracker stops a run at the cap", () => {
  const tracker = createSpendTracker(1, 0.01);
  expect(tracker.exhausted()).toBe(false);
  tracker.charge(50);
  expect(tracker.spentUsd()).toBeCloseTo(0.5, 10);
  expect(tracker.exhausted()).toBe(false);
  tracker.charge(50);
  expect(tracker.exhausted()).toBe(true);
  expect(() => createSpendTracker(0, 0.01)).toThrow("invalid spend cap");
  expect(() => createSpendTracker(1, 0)).toThrow("invalid token price");
});

test("stub reports are labeled non-evidence and redact case text", async () => {
  const bodies = scriptedBodies(KEYS, PROTOCOL.representative, 100);
  const { records, summary } = await runCases(PROTOCOL.representative, bodies, {
    backend: "file",
    upperBound: 5000,
    runId: "test-report",
  });
  const report = assembleReport({
    registration: PROTOCOL.registration,
    runId: "test-report",
    setName: "representative",
    caseSetSha256: PROTOCOL.registration.caseSets.representative.sha256,
    provenance: "stub",
    backend: "file",
    tenant: "h08-eval",
    pool: "default",
    testUpperBound: 5000,
    records,
    summary,
    spentUsd: null,
  });
  expect(report.evidence).toBe(false);
  expect(report.quality.evidence).toBe(false);
  expect(report.quality.accuracy).toBeNull();
  expect(report.quality.pass).toBe(false);
  expect(report.quality.note).toContain("plumbing only");
  expect(report.cost.usd).toBeNull();
  const text = JSON.stringify(report);
  for (const triage of PROTOCOL.representative) {
    expect(text).not.toContain(triage.subject);
    expect(text).not.toContain(triage.body.slice(0, 32));
  }
  expect(text).not.toContain("TYPESAFE_API_KEY=x");
});

test("choice stub helper builds decodable SystemOne answers", () => {
  const body = choiceBody(KEYS, "billing", 0.92, { input_tokens: 10, output_tokens: 5 });
  expect(JSON.stringify(body)).toContain("billing");
});

(PG_URL === undefined ? test.skip : test)(
  "metered run settles against the PG ledger (F04 default)",
  async () => {
    const cases = PROTOCOL.representative.slice(0, 3);
    const bodies = scriptedBodies(KEYS, cases, 100);
    const { records, summary, requests, maxConcurrency } = await runCases(cases, bodies, {
      backend: "pg",
      url: PG_URL!,
      upperBound: 5000,
      runId: `test-pg-${Date.now()}`,
      // Pool schedules configure once per ledger: isolate reruns by pool.
      pool: `h08pg${Date.now()}`,
    });
    expect(requests).toBe(3);
    expect(maxConcurrency).toBe(1);
    expect(records.map((record) => record.verdict)).toEqual(["correct", "correct", "correct"]);
    expect(summary.ledger.match).toBe(true);
    expect(summary.correlationOk).toBe(true);
    expect(summary.tokensIn).toBe(100 + 101 + 102);
    expect(summary.tokensOut).toBe(20 + 21 + 22);
  },
);
