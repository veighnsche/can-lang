// H08 eval CLI: `bun tools/runtime/ai-eval/main.ts --mode=offline ...`
// runs the frozen triage protocol against a scripted stub (plumbing,
// non-evidence). `--mode=live` authorizes against the live gate and
// refuses with the exact missing inputs unless credentials, spend
// cap, price table, and a qualified bound are all present; when they
// are, it runs the same frozen cases serially against the pinned
// provider with per-run spend enforcement.
import { tmpdir } from "node:os";
import { join } from "node:path";
import { CREDENTIAL_ENV, authorizeLive, createSpendTracker } from "./live-gate.ts";
import {
  loadProtocol,
  pricePerToken,
  qualifiedBound,
  stateSchema,
  type TriageCase,
  type TriageProtocol,
} from "./protocol.ts";
import { assembleReport, writeReport, type RunProvenance } from "./report.ts";
import { runSerial } from "./runner.ts";
import {
  createEvalDomain,
  createPipeline,
  expandDbUrl,
  openMeteredLedger,
  type EvalDomain,
  type MeteredLedger,
} from "./setup.ts";
import { scriptedBodies, serveScripted } from "./stub.ts";
import type { Connection } from "../../../runtime/transport/request.ts";

type Args = {
  mode: string;
  set: string;
  backend: string;
  db: string;
  out: string;
  tenant: string;
  pool: string;
  limit: number;
  testUpperBound: number;
};

function parseArgs(argv: readonly string[]): Args {
  const get = (name: string, fallback: string): string => {
    const hit = argv.find((arg) => arg.startsWith(`--${name}=`));
    return hit === undefined ? fallback : hit.slice(name.length + 3);
  };
  const limit = Number(get("limit", "1000000"));
  const testUpperBound = Number(get("test-upper-bound", "5000"));
  if (!Number.isSafeInteger(limit) || limit < 1) throw new Error("invalid --limit");
  if (!Number.isSafeInteger(testUpperBound) || testUpperBound < 1)
    throw new Error("invalid --test-upper-bound");
  return {
    mode: get("mode", "offline"),
    set: get("set", "representative"),
    backend: get("backend", "pg"),
    db: get("db", "can_h08_run1"),
    out: get("out", ""),
    tenant: get("tenant", "h08-eval"),
    pool: get("pool", "default"),
    limit,
    testUpperBound,
  };
}

async function openLedger(args: Args): Promise<MeteredLedger> {
  if (args.backend === "pg") {
    const url = expandDbUrl(process.env.CAN_TEST_POSTGRES_URL, "postgres", args.db);
    if (url === undefined) throw new Error("pg backend needs CAN_TEST_POSTGRES_URL");
    return openMeteredLedger({ kind: "pg", url });
  }
  if (args.backend === "file")
    return openMeteredLedger({ kind: "file", path: join(tmpdir(), `h08-ledger-${args.db}.json`) });
  if (args.backend === "sqlite")
    return openMeteredLedger({
      kind: "sqlite",
      filename: join(tmpdir(), `h08-ledger-${args.db}.sqlite`),
    });
  throw new Error(`unknown --backend=${args.backend}`);
}

function selectCases(
  args: Args,
  protocol: TriageProtocol,
): { cases: readonly TriageCase[]; sha: string } {
  if (args.set === "representative")
    return {
      cases: protocol.representative,
      sha: protocol.registration.caseSets.representative.sha256,
    };
  if (args.set === "heldout")
    return { cases: protocol.heldout, sha: protocol.registration.caseSets.heldout.sha256 };
  if (args.set === "both")
    return {
      cases: [...protocol.representative, ...protocol.heldout],
      sha: protocol.registration.caseSets.representative.sha256,
    };
  throw new Error(`unknown --set=${args.set}`);
}

async function main(): Promise<number> {
  const args = parseArgs(process.argv.slice(2));
  if (args.mode !== "offline" && args.mode !== "live")
    throw new Error(`unknown --mode=${args.mode}`);
  const protocol = await loadProtocol(join(import.meta.dir, "protocol"));
  const { cases, sha } = selectCases(args, protocol);
  const readEnvironment = (name: string): string | undefined => process.env[name];
  const evalDomain: EvalDomain = createEvalDomain();
  const runId = `h08-${args.set}-${Date.now()}`;
  const keys = protocol.registration.categories.map((category) => category.key);
  const wireModel = `${protocol.registration.pinned.model}-${protocol.registration.pinned.version}`;

  let provenance: RunProvenance = "stub";
  let connection: Connection;
  let upperBound: { value: number; testOnly: boolean };
  let testUpperBound: number | null = args.testUpperBound;
  let spend: ReturnType<typeof createSpendTracker> | undefined;
  let stub: ReturnType<typeof serveScripted> | undefined;

  if (args.mode === "live") {
    const decision = authorizeLive(process.env, protocol.registration);
    if (!decision.authorized) {
      for (const missing of decision.missing) console.error(`live blocked: ${missing}`);
      return 2;
    }
    // The gate passed, so both validated reads below are defined.
    upperBound = { value: qualifiedBound(protocol.registration)!, testOnly: false };
    spend = createSpendTracker(decision.spendCapUsd, pricePerToken(protocol.registration)!);
    testUpperBound = null;
    provenance = "live";
    connection = {
      endpoint: protocol.registration.pinned.endpoint,
      timeoutMilliseconds: protocol.registration.pinned.timeoutMs,
      maxBodyBytes: protocol.registration.pinned.maxBodyBytes,
      bearerEnvironment: CREDENTIAL_ENV,
      headers: [],
    };
  } else {
    stub = serveScripted(scriptedBodies(keys, cases, 100));
    connection = {
      endpoint: stub.url("/systemone"),
      timeoutMilliseconds: protocol.registration.pinned.timeoutMs,
      maxBodyBytes: protocol.registration.pinned.maxBodyBytes,
      headers: [],
    };
    upperBound = { value: args.testUpperBound, testOnly: true };
  }

  try {
    const ledger = await openLedger(args);
    try {
      const pipeline = createPipeline({
        eval: evalDomain,
        store: ledger.store,
        connection,
        stateSchema: stateSchema(),
        model: wireModel,
        pinned: {
          provider: protocol.registration.pinned.provider,
          model: protocol.registration.pinned.model,
          version: protocol.registration.pinned.version,
        },
        pool: args.pool,
        readEnvironment,
        upperBound,
      });
      const { records, summary } = await runSerial({
        eval: evalDomain,
        protocol,
        setName: args.set,
        cases,
        pipeline,
        store: ledger.store,
        tenant: args.tenant,
        pool: args.pool,
        runId,
        epoch: { limit: args.limit, periodMs: 3_600_000, anchorMs: 0, version: "1" },
        ...(spend === undefined ? {} : { spend }),
      });
      const report = assembleReport({
        registration: protocol.registration,
        runId,
        setName: args.set,
        caseSetSha256: sha,
        provenance,
        backend: ledger.backend,
        tenant: args.tenant,
        pool: args.pool,
        testUpperBound,
        records,
        summary,
        spentUsd: spend?.spentUsd() ?? null,
      });
      const out =
        args.out === ""
          ? join(tmpdir(), `h08-${args.set}-${args.backend}-${provenance}.json`)
          : args.out;
      await writeReport(out, report);
      console.log(
        `${args.set}/${provenance}: ${summary.correct} correct ${summary.incorrect} incorrect ` +
          `${summary.abstain} abstain ${summary.invalidOutput} invalid ${summary.rejected} rejected ` +
          `tokens=${summary.tokensIn}+${summary.tokensOut} maxMs=${summary.latencyMaxMs} ` +
          `ledgerMatch=${summary.ledger.match} correlationOk=${summary.correlationOk} -> ${out}`,
      );
      return 0;
    } finally {
      await ledger.close();
    }
  } finally {
    await stub?.close();
  }
}

const code = await main();
process.exit(code);
