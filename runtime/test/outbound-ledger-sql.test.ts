// F04 ledger backend qualification: the F01 budget-ledger service
// contract executed against every durable SQL backend (SQLite file,
// PostgreSQL, MySQL) plus cross-process races over file and SQL
// backends. Live legs need CAN_TEST_POSTGRES_URL / CAN_TEST_MYSQL_URL
// (concrete per-run URLs, or the provision template with a <db>
// placeholder selecting can_f04); otherwise they skip. URLs and
// passwords are never logged. SQLite legs always run.
import { expect, test } from "bun:test";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { epochSchedule } from "../outbound/epoch.ts";
import { openFileLedgerStore } from "../outbound/file-ledger.ts";
import {
  configurePool,
  createMemoryLedgerStore,
  epochStatus,
  fence,
  ledgerHealth,
  reconcile,
  releaseUndispatched,
  reserve,
  settle,
  transitionPool,
  type LedgerStore,
  type ReserveInput,
} from "../outbound/ledger.ts";
import {
  applyLedgerDDL,
  openSqlLedgerStore,
  type SqlLedgerStore,
  type SqlLedgerTarget,
} from "../outbound/sql-ledger.ts";

const HOUR = 3_600_000;
const PROFILE = { provider: "typesafe", model: "jev", version: "1.13.0" };
const RACE_URL_ENV = "F04_RACE_DATABASE_URL";

function dbUrl(env: string, scheme: string, fallbackDb: string): string | undefined {
  const template = process.env[env];
  if (template === undefined || template === "") return undefined;
  if (!template.startsWith(scheme)) throw new Error(`refusing non-${scheme} test URL`);
  return template.includes("<db>") ? template.replace("<db>", fallbackDb) : template;
}

const PG_URL = dbUrl("CAN_TEST_POSTGRES_URL", "postgres", "can_f04");
const MYSQL_URL = dbUrl("CAN_TEST_MYSQL_URL", "mysql", "can_f04");

function schedule(limit = 1000, periodMs = HOUR, anchorMs = 0, version = "1") {
  return epochSchedule({ pool: "default", limit, periodMs, anchorMs, version });
}

function input(overrides: Partial<ReserveInput> = {}): ReserveInput {
  return {
    tenant: "acme",
    pool: "default",
    correlation: "req-1",
    profile: PROFILE,
    qualified: true,
    upperBound: 100,
    ...overrides,
  };
}

async function configured(store: LedgerStore, limit = 1000) {
  const done = await configurePool(store, schedule(limit));
  expect(done).toEqual({ configured: true });
}

type Raw = InstanceType<typeof Bun.SQL>;

function staticT(text: string): TemplateStringsArray {
  return Object.assign([text], { raw: [text] }) as unknown as TemplateStringsArray;
}

async function openRaw(target: SqlLedgerTarget): Promise<Raw> {
  if (target.dialect === "sqlite") {
    const client = new Bun.SQL({
      adapter: "sqlite",
      filename: target.filename,
      safeIntegers: true,
    });
    await client.connect();
    return client;
  }
  const client = new Bun.SQL(
    target.url,
    target.dialect === "postgres"
      ? { adapter: "postgres", bigint: true, max: 2 }
      : { adapter: "mysql", bigint: true, max: 2, tls: true },
  );
  await client.connect();
  return client;
}

const LEDGER_TABLES = [
  "ai_budget_invocations",
  "ai_budget_epochs",
  "ai_budget_schedules",
  "ai_budget_quarantine",
  "ai_budget_state",
];

async function dropLedgerTables(target: SqlLedgerTarget): Promise<void> {
  const raw = await openRaw(target);
  try {
    for (const table of LEDGER_TABLES) {
      await raw(staticT(`DROP TABLE IF EXISTS ${table}`));
    }
  } finally {
    await raw.close();
  }
}

type Opened = {
  store: SqlLedgerStore;
  target: SqlLedgerTarget;
  reopen: () => Promise<SqlLedgerStore>;
  cleanup: () => Promise<void>;
};

async function setupSqliteFile(): Promise<Opened> {
  const directory = await mkdtemp(join(tmpdir(), "f04-sqlite-"));
  const target: SqlLedgerTarget = { dialect: "sqlite", filename: join(directory, "ledger.db") };
  await applyLedgerDDL(target);
  const store = await openSqlLedgerStore(target);
  return {
    store,
    target,
    reopen: () => openSqlLedgerStore(target),
    cleanup: async () => {
      // Idempotent: restart legs close the first handle early.
      await store.close().catch(() => undefined);
      await rm(directory, { recursive: true, force: true });
    },
  };
}

async function setupServer(
  target: SqlLedgerTarget,
): Promise<Omit<Opened, "target"> & { target: SqlLedgerTarget }> {
  await dropLedgerTables(target);
  await applyLedgerDDL(target);
  const store = await openSqlLedgerStore(target);
  return {
    store,
    target,
    reopen: () => openSqlLedgerStore(target),
    cleanup: async () => {
      // Idempotent: restart legs close the first handle early.
      await store.close().catch(() => undefined);
    },
  };
}

type Backend = {
  name: string;
  run: typeof test;
  setup: () => Promise<Opened>;
};

const backends: Backend[] = [
  { name: "sqlite-file", run: test, setup: setupSqliteFile },
  {
    name: "postgres",
    run: test.skipIf(PG_URL === undefined),
    setup: () => {
      if (PG_URL === undefined) throw new Error("unreachable: postgres leg skipped");
      return setupServer({ dialect: "postgres", url: PG_URL });
    },
  },
  {
    name: "mysql",
    run: test.skipIf(MYSQL_URL === undefined),
    setup: () => {
      if (MYSQL_URL === undefined) throw new Error("unreachable: mysql leg skipped");
      return setupServer({ dialect: "mysql", url: MYSQL_URL });
    },
  },
];

for (const backend of backends) {
  backend.run(`${backend.name}: exact-fit admission and immediate typed rejection`, async () => {
    const opened = await backend.setup();
    try {
      await configured(opened.store, 250);
      const first = await reserve(opened.store, input({ upperBound: 100, nowMs: 7 }));
      expect(first.outcome).toBe("admitted");
      if (first.outcome !== "admitted") throw new Error("unreachable");
      expect(first.remaining).toBe(150);
      expect(first.epochResetMs).toBe(HOUR);
      const exact = await reserve(
        opened.store,
        input({ invocationId: "inv-exact", correlation: "req-2", upperBound: 150, nowMs: 8 }),
      );
      expect(exact).toEqual(expect.objectContaining({ outcome: "admitted", remaining: 0 }));
      const rejected = await reserve(
        opened.store,
        input({ invocationId: "inv-over", correlation: "req-3", upperBound: 1, nowMs: 9 }),
      );
      expect(rejected).toEqual(
        expect.objectContaining({
          outcome: "exceeded",
          required: 1,
          remaining: 0,
          epochResetMs: HOUR,
        }),
      );
      // The rejection reserves nothing: no third invocation exists.
      expect(await epochStatus(opened.store, "acme", "default", 0)).toEqual(
        expect.objectContaining({
          row: expect.objectContaining({ committed: 0, held: 250 }),
          unresolved: 2,
        }),
      );
      expect(await reconcile(opened.store, "inv-over")).toEqual({ status: "absent" });
    } finally {
      await opened.cleanup();
    }
  });

  backend.run(`${backend.name}: reserve, fence, settle, release lifecycle`, async () => {
    const opened = await backend.setup();
    try {
      await configured(opened.store, 1000);
      const admitted = await reserve(opened.store, input({ invocationId: "inv-a", nowMs: 5 }));
      expect(admitted.outcome).toBe("admitted");
      // Duplicate reserve with identical inputs returns the admission.
      expect(await reserve(opened.store, input({ invocationId: "inv-a", nowMs: 6 }))).toEqual(
        expect.objectContaining({ outcome: "admitted", duplicate: true }),
      );
      // Same invocation ID with different terms conflicts.
      expect(
        await reserve(opened.store, input({ invocationId: "inv-a", upperBound: 50, nowMs: 6 })),
      ).toEqual(
        expect.objectContaining({ outcome: "unavailable", reason: "conflicting-settlement" }),
      );
      expect(await fence(opened.store, "inv-a")).toEqual(
        expect.objectContaining({ outcome: "fenced", duplicate: false }),
      );
      expect(await fence(opened.store, "inv-a")).toEqual(
        expect.objectContaining({ outcome: "fenced", duplicate: true }),
      );
      const settled = await settle(opened.store, {
        invocationId: "inv-a",
        inputTokens: 30,
        outputTokens: 20,
      });
      expect(settled).toEqual(
        expect.objectContaining({ outcome: "settled", actual: 50, released: 50, duplicate: false }),
      );
      expect(
        await settle(opened.store, { invocationId: "inv-a", inputTokens: 30, outputTokens: 20 }),
      ).toEqual(expect.objectContaining({ outcome: "settled", released: 0, duplicate: true }));
      expect(
        await settle(opened.store, { invocationId: "inv-a", inputTokens: 30, outputTokens: 21 }),
      ).toEqual(
        expect.objectContaining({ outcome: "unavailable", reason: "conflicting-settlement" }),
      );
      expect(await fence(opened.store, "inv-a")).toEqual(
        expect.objectContaining({ outcome: "unavailable", reason: "conflicting-settlement" }),
      );
      // An unfenced attempt releases; a fenced one keeps its hold.
      await reserve(
        opened.store,
        input({ invocationId: "inv-b", correlation: "req-b", upperBound: 200, nowMs: 7 }),
      );
      expect(await releaseUndispatched(opened.store, "inv-b")).toEqual(
        expect.objectContaining({ outcome: "released", duplicate: false }),
      );
      expect(await releaseUndispatched(opened.store, "inv-b")).toEqual(
        expect.objectContaining({ outcome: "released", duplicate: true }),
      );
      await reserve(
        opened.store,
        input({ invocationId: "inv-c", correlation: "req-c", upperBound: 300, nowMs: 8 }),
      );
      await fence(opened.store, "inv-c");
      expect(await releaseUndispatched(opened.store, "inv-c")).toEqual(
        expect.objectContaining({ outcome: "held" }),
      );
      expect(await fence(opened.store, "inv-missing")).toEqual(
        expect.objectContaining({ outcome: "unavailable", reason: "unknown-invocation" }),
      );
      expect(await epochStatus(opened.store, "acme", "default", 0)).toEqual(
        expect.objectContaining({
          row: expect.objectContaining({ committed: 50, held: 300 }),
          unresolved: 1,
        }),
      );
    } finally {
      await opened.cleanup();
    }
  });

  backend.run(
    `${backend.name}: unknown holds survive restart until authoritative settlement`,
    async () => {
      const opened = await backend.setup();
      try {
        await configured(opened.store, 500);
        await reserve(opened.store, input({ invocationId: "inv-hold", nowMs: 5 }));
        await fence(opened.store, "inv-hold");
        expect(
          await settle(opened.store, {
            invocationId: "inv-hold",
            inputTokens: -1,
            outputTokens: 0,
          }),
        ).toEqual(expect.objectContaining({ outcome: "unavailable", reason: "invalid-usage" }));
        expect(
          await settle(opened.store, {
            invocationId: "inv-hold",
            inputTokens: 50,
            outputTokens: null,
          }),
        ).toEqual(expect.objectContaining({ outcome: "unavailable", reason: "invalid-usage" }));
        // Restart: close and reopen against the same durable database.
        await opened.store.close();
        const restarted = await opened.reopen();
        try {
          expect(await reconcile(restarted, "inv-hold")).toEqual(
            expect.objectContaining({
              status: "found",
              record: expect.objectContaining({ state: "fenced", upperBound: 100 }),
            }),
          );
          expect(await epochStatus(restarted, "acme", "default", 0)).toEqual(
            expect.objectContaining({
              row: expect.objectContaining({ committed: 0, held: 100 }),
              unresolved: 1,
            }),
          );
          // Only authoritative usage reconciles the hold.
          expect(
            await settle(restarted, {
              invocationId: "inv-hold",
              inputTokens: 60,
              outputTokens: 10,
            }),
          ).toEqual(expect.objectContaining({ outcome: "settled", actual: 70, released: 30 }));
          expect(await epochStatus(restarted, "acme", "default", 0)).toEqual(
            expect.objectContaining({ row: expect.objectContaining({ committed: 70, held: 0 }) }),
          );
        } finally {
          await restarted.close();
        }
      } finally {
        await opened.cleanup();
      }
    },
  );

  backend.run(
    `${backend.name}: epoch transitions pin live rows; late usage settles originally`,
    async () => {
      const opened = await backend.setup();
      try {
        await configured(opened.store, 200);
        await reserve(opened.store, input({ invocationId: "inv-old", nowMs: 5 }));
        const moved = await transitionPool(
          opened.store,
          "default",
          { limit: 50, version: "2" },
          10,
        );
        expect(moved).toEqual(
          expect.objectContaining({
            transitioned: true,
            schedule: expect.objectContaining({ limit: 50, anchorMs: HOUR, version: "2" }),
          }),
        );
        // The live epoch keeps its pinned limit under the old version.
        await reserve(
          opened.store,
          input({ invocationId: "inv-old2", correlation: "req-2", nowMs: 11 }),
        );
        expect(await epochStatus(opened.store, "acme", "default", 0)).toEqual(
          expect.objectContaining({
            row: expect.objectContaining({ limit: 200, scheduleVersion: "1", held: 200 }),
          }),
        );
        // The expired epoch's holds never spend the new epoch's allowance.
        expect(
          await reserve(
            opened.store,
            input({
              invocationId: "inv-new",
              correlation: "req-3",
              upperBound: 50,
              nowMs: HOUR + 1,
            }),
          ),
        ).toEqual(expect.objectContaining({ outcome: "admitted" }));
        expect(await epochStatus(opened.store, "acme", "default", HOUR)).toEqual(
          expect.objectContaining({
            row: expect.objectContaining({ limit: 50, scheduleVersion: "2", held: 50 }),
          }),
        );
        // Late actuals debit the original admission epoch.
        expect(
          await settle(opened.store, {
            invocationId: "inv-old",
            inputTokens: 60,
            outputTokens: 40,
          }),
        ).toEqual(expect.objectContaining({ outcome: "settled", actual: 100 }));
        expect(await epochStatus(opened.store, "acme", "default", 0)).toEqual(
          expect.objectContaining({ row: expect.objectContaining({ committed: 100, held: 100 }) }),
        );
        expect(await epochStatus(opened.store, "acme", "default", HOUR)).toEqual(
          expect.objectContaining({ row: expect.objectContaining({ committed: 0, held: 50 }) }),
        );
      } finally {
        await opened.cleanup();
      }
    },
  );

  backend.run(`${backend.name}: profile breach quarantines durably across restart`, async () => {
    const opened = await backend.setup();
    try {
      await configured(opened.store, 1000);
      await reserve(opened.store, input({ invocationId: "inv-breach", nowMs: 5 }));
      const settled = await settle(opened.store, {
        invocationId: "inv-breach",
        inputTokens: 150,
        outputTokens: 0,
      });
      expect(settled).toEqual(
        expect.objectContaining({
          outcome: "settled",
          actual: 150,
          released: 0,
          breach: expect.objectContaining({
            profile: "typesafe/jev/1.13.0",
            upperBound: 100,
            actual: 150,
          }),
        }),
      );
      expect(
        await reserve(
          opened.store,
          input({ invocationId: "inv-after", correlation: "req-2", nowMs: 6 }),
        ),
      ).toEqual(expect.objectContaining({ outcome: "unavailable", reason: "breached-profile" }));
      await opened.store.close();
      const restarted = await opened.reopen();
      try {
        expect(
          await reserve(
            restarted,
            input({ invocationId: "inv-after2", correlation: "req-3", nowMs: 7 }),
          ),
        ).toEqual(expect.objectContaining({ outcome: "unavailable", reason: "breached-profile" }));
        expect(await ledgerHealth(restarted)).toEqual(
          expect.objectContaining({
            status: "ok",
            committed: 150,
            held: 0,
            unresolved: 0,
            quarantined: ["typesafe/jev/1.13.0"],
          }),
        );
      } finally {
        await restarted.close();
      }
    } finally {
      await opened.cleanup();
    }
  });

  backend.run(
    `${backend.name}: concurrent reserves and settlements stay exact`,
    async () => {
      const opened = await backend.setup();
      try {
        await configured(opened.store, 1000);
        const first = await Promise.all(
          Array.from({ length: 8 }, (_, index) =>
            reserve(
              opened.store,
              input({
                invocationId: `inv-p-${index}`,
                correlation: `req-p-${index}`,
                upperBound: 100,
                nowMs: 5,
              }),
            ),
          ),
        );
        expect(first.map((outcome) => outcome.outcome)).toEqual(Array(8).fill("admitted"));
        const topUp = await Promise.all(
          Array.from({ length: 4 }, (_, index) =>
            reserve(
              opened.store,
              input({
                invocationId: `inv-q-${index}`,
                correlation: `req-q-${index}`,
                upperBound: 100,
                nowMs: 6,
              }),
            ),
          ),
        );
        const admitted = topUp.filter((outcome) => outcome.outcome === "admitted").length;
        const exceeded = topUp.filter((outcome) => outcome.outcome === "exceeded").length;
        expect([admitted, exceeded]).toEqual([2, 2]);
        expect(await epochStatus(opened.store, "acme", "default", 0)).toEqual(
          expect.objectContaining({
            row: expect.objectContaining({ committed: 0, held: 1000 }),
            unresolved: 10,
          }),
        );
        const ids = [
          ...Array.from({ length: 8 }, (_, index) => `inv-p-${index}`),
          ...topUp.flatMap((outcome, index) =>
            outcome.outcome === "admitted" ? [`inv-q-${index}`] : [],
          ),
        ];
        expect(ids).toHaveLength(10);
        const settled = await Promise.all(
          ids.map((invocationId) =>
            settle(opened.store, { invocationId, inputTokens: 60, outputTokens: 10 }),
          ),
        );
        expect(new Set(settled.map((outcome) => outcome.outcome))).toEqual(new Set(["settled"]));
        expect(await epochStatus(opened.store, "acme", "default", 0)).toEqual(
          expect.objectContaining({ row: expect.objectContaining({ committed: 700, held: 0 }) }),
        );
      } finally {
        await opened.cleanup();
      }
    },
    { timeout: 60_000 },
  );

  backend.run(`${backend.name}: a stale commit loses the race without corruption`, async () => {
    const opened = await backend.setup();
    let second: SqlLedgerStore | undefined;
    try {
      await configured(opened.store, 1000);
      const staleView = await opened.store.load();
      second = await opened.reopen();
      const winner = await reserve(second, input({ invocationId: "inv-winner", nowMs: 5 }));
      expect(winner.outcome).toBe("admitted");
      const staleNext = { ...staleView, version: staleView.version + 1 };
      expect(await opened.store.commit(staleView.version, staleNext)).toEqual({ committed: false });
      expect(await reconcile(opened.store, "inv-winner")).toEqual(
        expect.objectContaining({ status: "found" }),
      );
    } finally {
      if (second !== undefined) await second.close().catch(() => undefined);
      await opened.cleanup();
    }
  });

  backend.run(`${backend.name}: the CHECK floor rejects invalid rows natively`, async () => {
    const opened = await backend.setup();
    const raw = await openRaw(opened.target);
    try {
      await configured(opened.store, 200);
      // Bun.SQL query promises settle only under direct await, so the
      // assertions below use try/catch instead of .rejects matchers.
      let negative = false;
      try {
        await raw(
          staticT(
            `INSERT INTO ai_budget_epochs (tenant, pool, epoch_start, epoch_end, token_limit, schedule_version, committed, held)
           VALUES ('acme', 'default', 0, 3600000, 200, '1', -1, 0)`,
          ),
        );
      } catch {
        negative = true;
      }
      expect(negative).toBe(true);
      let state = false;
      try {
        await raw(staticT(`UPDATE ai_budget_state SET version = -1 WHERE id = 1`));
      } catch {
        state = true;
      }
      expect(state).toBe(true);
      await reserve(opened.store, input({ invocationId: "inv-chk", nowMs: 5 }));
      let openState = false;
      try {
        await raw(
          staticT(
            `UPDATE ai_budget_invocations SET state = 'maybe' WHERE invocation_id = 'inv-chk'`,
          ),
        );
      } catch {
        openState = true;
      }
      expect(openState).toBe(true);
      // The rejected writes changed nothing.
      expect(await epochStatus(opened.store, "acme", "default", 0)).toEqual(
        expect.objectContaining({ row: expect.objectContaining({ committed: 0, held: 100 }) }),
      );
    } finally {
      await raw.close();
      await opened.cleanup();
    }
  });

  backend.run(`${backend.name}: missing tables fail closed, never auto-create`, async () => {
    const opened = await backend.setup();
    try {
      await dropLedgerTables(opened.target);
      expect(await reserve(opened.store, input({ nowMs: 5 }))).toEqual(
        expect.objectContaining({ outcome: "unavailable", reason: "storage-unavailable" }),
      );
      expect(await fence(opened.store, "inv-x")).toEqual(
        expect.objectContaining({ outcome: "unavailable", reason: "storage-unavailable" }),
      );
      expect(
        await settle(opened.store, { invocationId: "inv-x", inputTokens: 1, outputTokens: 1 }),
      ).toEqual(expect.objectContaining({ outcome: "unavailable", reason: "storage-unavailable" }));
      expect(await reconcile(opened.store, "inv-x")).toEqual(
        expect.objectContaining({ status: "unavailable" }),
      );
      expect(await epochStatus(opened.store, "acme", "default", 0)).toEqual(
        expect.objectContaining({ status: "unavailable" }),
      );
      expect(await ledgerHealth(opened.store)).toEqual(
        expect.objectContaining({ status: "unavailable" }),
      );
    } finally {
      await opened.cleanup();
    }
  });

  backend.run(`${backend.name}: maximum-length identities round-trip and quarantine`, async () => {
    const opened = await backend.setup();
    try {
      await configured(opened.store, 1000);
      const long128 = `t${"e".repeat(127)}`;
      const long256 = `m${"o".repeat(255)}`;
      const profile = { provider: long256, model: long256, version: long256 };
      const admitted = await reserve(
        opened.store,
        input({
          tenant: long128,
          pool: "default",
          correlation: long128,
          invocationId: long128,
          profile,
          upperBound: 100,
          nowMs: 5,
        }),
      );
      expect(admitted.outcome).toBe("admitted");
      // A breaching settlement exercises the digest quarantine key plus
      // the widened display columns on every dialect.
      expect(
        await settle(opened.store, { invocationId: long128, inputTokens: 150, outputTokens: 0 }),
      ).toEqual(expect.objectContaining({ outcome: "settled", actual: 150 }));
      expect(
        await reserve(
          opened.store,
          input({ invocationId: "inv-after", correlation: "req-2", profile, nowMs: 6 }),
        ),
      ).toEqual(expect.objectContaining({ outcome: "unavailable", reason: "breached-profile" }));
      await opened.store.close();
      const restarted = await opened.reopen();
      try {
        expect(await reconcile(restarted, long128)).toEqual(
          expect.objectContaining({
            status: "found",
            record: expect.objectContaining({ breach: true }),
          }),
        );
        expect(await ledgerHealth(restarted)).toEqual(
          expect.objectContaining({ quarantined: [`${long256}/${long256}/${long256}`] }),
        );
      } finally {
        await restarted.close();
      }
    } finally {
      await opened.cleanup();
    }
  });

  backend.run(`${backend.name}: huge limits keep exact accounting through BIGINT`, async () => {
    const opened = await backend.setup();
    try {
      await configured(opened.store, Number.MAX_SAFE_INTEGER);
      const admitted = await reserve(
        opened.store,
        input({ invocationId: "inv-huge", upperBound: 100, nowMs: 5 }),
      );
      expect(admitted).toEqual(
        expect.objectContaining({ outcome: "admitted", remaining: Number.MAX_SAFE_INTEGER - 100 }),
      );
      expect(
        await settle(opened.store, { invocationId: "inv-huge", inputTokens: 30, outputTokens: 20 }),
      ).toEqual(expect.objectContaining({ outcome: "settled", actual: 50, released: 50 }));
      expect(await epochStatus(opened.store, "acme", "default", 0)).toEqual(
        expect.objectContaining({ row: expect.objectContaining({ committed: 50, held: 0 }) }),
      );
    } finally {
      await opened.cleanup();
    }
  });

  backend.run(`${backend.name}: health distinguishes committed, held, and unresolved`, async () => {
    const opened = await backend.setup();
    try {
      await configured(opened.store, 1000);
      await reserve(opened.store, input({ invocationId: "inv-held", upperBound: 100, nowMs: 5 }));
      await reserve(
        opened.store,
        input({ invocationId: "inv-spent", correlation: "req-2", upperBound: 200, nowMs: 6 }),
      );
      await fence(opened.store, "inv-spent");
      await settle(opened.store, { invocationId: "inv-spent", inputTokens: 150, outputTokens: 10 });
      await reserve(
        opened.store,
        input({ invocationId: "inv-freed", correlation: "req-3", upperBound: 50, nowMs: 7 }),
      );
      await releaseUndispatched(opened.store, "inv-freed");
      expect(await ledgerHealth(opened.store)).toEqual(
        expect.objectContaining({
          status: "ok",
          epochs: 1,
          committed: 160,
          held: 100,
          unresolved: 1,
          quarantined: [],
        }),
      );
    } finally {
      await opened.cleanup();
    }
  });

  backend.run(
    `${backend.name}: crash recovery reconciles by invocation ID`,
    async () => {
      const opened = await backend.setup();
      const raw = await openRaw(opened.target);
      try {
        await configured(opened.store, 1000);
        await reserve(opened.store, input({ invocationId: "inv-base", upperBound: 100, nowMs: 5 }));
        const ghost = {
          invocationId: "inv-ghost",
          tenant: "acme",
          pool: "default",
          correlation: "req-ghost",
          upperBound: 100,
        };
        const stageWrite = async (tx: Raw) => {
          await tx`INSERT INTO ai_budget_invocations
            (invocation_id, tenant, pool, correlation, epoch_start, epoch_end, token_limit,
             schedule_version, provider, model, profile_version, upper_bound, state, fenced, breach)
            VALUES (${ghost.invocationId}, ${ghost.tenant}, ${ghost.pool}, ${ghost.correlation},
             0, 3600000, 1000, '1', 'typesafe', 'jev', '1.13.0', ${ghost.upperBound}, 'held', 0, 0)`;
          await tx`UPDATE ai_budget_epochs SET held = held + ${ghost.upperBound}
            WHERE tenant = ${ghost.tenant} AND pool = ${ghost.pool} AND epoch_start = 0`;
          await tx`UPDATE ai_budget_state SET version = version + 1 WHERE id = 1`;
        };
        // Crash before commit: the rolled-back write is invisible and
        // reconciles absent — never dispatch on it.
        await raw
          .begin(async (native: unknown) => {
            await stageWrite(native as Raw);
            throw new Error("crash-before-commit");
          })
          .catch((cause: unknown) => {
            if ((cause as Error).message !== "crash-before-commit") throw cause;
          });
        expect(await reconcile(opened.store, ghost.invocationId)).toEqual({ status: "absent" });
        expect(await epochStatus(opened.store, "acme", "default", 0)).toEqual(
          expect.objectContaining({ row: expect.objectContaining({ held: 100 }) }),
        );
        // Crash after commit: the durable reservation reconciles found
        // and settles normally; the service continues from the bumped
        // version without losing it.
        await raw.begin(async (native: unknown) => {
          await stageWrite(native as Raw);
        });
        expect(await reconcile(opened.store, ghost.invocationId)).toEqual(
          expect.objectContaining({
            status: "found",
            record: expect.objectContaining({ state: "held", upperBound: 100 }),
          }),
        );
        const next = await reserve(
          opened.store,
          input({ invocationId: "inv-next", correlation: "req-next", upperBound: 50, nowMs: 6 }),
        );
        expect(next).toEqual(expect.objectContaining({ outcome: "admitted", remaining: 750 }));
        expect(
          await settle(opened.store, {
            invocationId: ghost.invocationId,
            inputTokens: 60,
            outputTokens: 10,
          }),
        ).toEqual(expect.objectContaining({ outcome: "settled", actual: 70, released: 30 }));
        expect(await epochStatus(opened.store, "acme", "default", 0)).toEqual(
          expect.objectContaining({ row: expect.objectContaining({ committed: 70, held: 150 }) }),
        );
      } finally {
        await raw.close();
        await opened.cleanup();
      }
    },
    { timeout: 60_000 },
  );
}

test("sqlite memory opens and fails closed without schema", async () => {
  const store = await openSqlLedgerStore({ dialect: "sqlite", filename: ":memory:" });
  try {
    expect(await reserve(store, input({ nowMs: 5 }))).toEqual(
      expect.objectContaining({ outcome: "unavailable", reason: "storage-unavailable" }),
    );
  } finally {
    await store.close();
  }
});

test("memory backend is process-local, not durable storage", async () => {
  const first = createMemoryLedgerStore();
  await configured(first, 100);
  await reserve(first, input({ nowMs: 5 }));
  // A second handle starts empty: nothing is shared or persisted.
  const second = createMemoryLedgerStore();
  expect(await epochStatus(second, "acme", "default", 0)).toEqual({ status: "absent" });
});

test("invalid ledger targets reject with TypeError", async () => {
  await expect(openSqlLedgerStore({ dialect: "sqlite", filename: "" })).rejects.toThrow(TypeError);
  await expect(openSqlLedgerStore({ dialect: "postgres", url: "" })).rejects.toThrow(TypeError);
  await expect(openSqlLedgerStore({ dialect: "mysql", url: "" })).rejects.toThrow(TypeError);
  await expect(openSqlLedgerStore({ dialect: "duckdb" } as never)).rejects.toThrow(TypeError);
});

test("unreachable servers fail with static text, never the URL", async () => {
  await expect(
    openSqlLedgerStore({ dialect: "postgres", url: "postgres://127.0.0.1:1/nope" }),
  ).rejects.toThrow("ledger postgres unreachable");
  await expect(
    openSqlLedgerStore({ dialect: "mysql", url: "mysql://127.0.0.1:1/nope" }),
  ).rejects.toThrow("ledger mysql unreachable");
});

type RaceSetup = { target: string; raceEnv: Record<string, string>; cleanup: () => Promise<void> };

async function setupFileRace(): Promise<RaceSetup> {
  const directory = await mkdtemp(join(tmpdir(), "f04-race-file-"));
  const target = join(directory, "ledger.json");
  await configured(openFileLedgerStore(target), 1000);
  return {
    target,
    raceEnv: {},
    cleanup: async () => {
      await rm(directory, { recursive: true, force: true });
    },
  };
}

async function setupSqlRace(target: SqlLedgerTarget): Promise<RaceSetup> {
  await dropLedgerTables(target);
  await applyLedgerDDL(target);
  const store = await openSqlLedgerStore(target);
  await configured(store, 1000);
  await store.close();
  // Server workers receive the database URL through their own child
  // environment only — never on argv, never in the parent's env.
  return {
    target: target.dialect === "sqlite" ? target.filename : RACE_URL_ENV,
    raceEnv: target.dialect === "sqlite" ? {} : { [RACE_URL_ENV]: target.url },
    cleanup: async () => undefined,
  };
}

const raceBackends: {
  kind: "file" | "sqlite" | "postgres" | "mysql";
  run: typeof test;
  setup: () => Promise<RaceSetup>;
}[] = [
  { kind: "file", run: test, setup: setupFileRace },
  {
    kind: "sqlite",
    run: test,
    setup: async () => {
      const directory = await mkdtemp(join(tmpdir(), "f04-race-sqlite-"));
      const target: SqlLedgerTarget = { dialect: "sqlite", filename: join(directory, "ledger.db") };
      const opened = await setupSqlRace(target);
      const cleanup = opened.cleanup;
      return {
        target: opened.target,
        raceEnv: opened.raceEnv,
        cleanup: async () => {
          await cleanup();
          await rm(directory, { recursive: true, force: true });
        },
      };
    },
  },
  {
    kind: "postgres",
    run: test.skipIf(PG_URL === undefined),
    setup: () => {
      if (PG_URL === undefined) throw new Error("unreachable: postgres race skipped");
      return setupSqlRace({ dialect: "postgres", url: PG_URL });
    },
  },
  {
    kind: "mysql",
    run: test.skipIf(MYSQL_URL === undefined),
    setup: () => {
      if (MYSQL_URL === undefined) throw new Error("unreachable: mysql race skipped");
      return setupSqlRace({ dialect: "mysql", url: MYSQL_URL });
    },
  },
];

for (const race of raceBackends) {
  race.run(
    `two processes share one ${race.kind} ledger with exact admission`,
    async () => {
      const opened = await race.setup();
      try {
        const worker = join(dirname(import.meta.path), "ledger-race-worker.ts");
        const started = Date.now();
        const run = (prefix: string) =>
          Bun.spawn(
            ["bun", worker, race.kind, opened.target, "acme", "default", prefix, "25", "40", "5"],
            { stdout: "pipe", stderr: "pipe", env: { ...process.env, ...opened.raceEnv } },
          );
        const first = run("worker-a");
        const second = run("worker-b");
        const [codeA, codeB] = await Promise.all([first.exited, second.exited]);
        const outA = await new Response(first.stdout).text();
        const outB = await new Response(second.stdout).text();
        if (codeA !== 0 || codeB !== 0) {
          const errA = await new Response(first.stderr).text();
          const errB = await new Response(second.stderr).text();
          throw new Error(`worker failed: ${codeA}/${codeB}\n${errA}\n${errB}`);
        }
        const parsedA = JSON.parse(outA) as { admitted: number; exceeded: number };
        const parsedB = JSON.parse(outB) as { admitted: number; exceeded: number };
        const admitted = parsedA.admitted + parsedB.admitted;
        const exceeded = parsedA.exceeded + parsedB.exceeded;
        // floor(1000 / 40) = 25 admissions at 1000 held; zero lost updates.
        expect(admitted).toBe(25);
        expect(exceeded).toBe(25);
        let verify: LedgerStore;
        if (race.kind === "file") {
          verify = openFileLedgerStore(opened.target);
        } else if (race.kind === "sqlite") {
          verify = await openSqlLedgerStore({ dialect: "sqlite", filename: opened.target });
        } else {
          const url = opened.raceEnv[RACE_URL_ENV];
          if (url === undefined) throw new Error("unreachable: race URL missing");
          verify =
            race.kind === "postgres"
              ? await openSqlLedgerStore({ dialect: "postgres", url })
              : await openSqlLedgerStore({ dialect: "mysql", url });
        }
        try {
          expect(await epochStatus(verify, "acme", "default", 0)).toEqual(
            expect.objectContaining({
              row: expect.objectContaining({ committed: 0, held: 1000 }),
              unresolved: 25,
            }),
          );
        } finally {
          if (race.kind !== "file") await (verify as SqlLedgerStore).close();
        }
        console.log(
          `F04SUMMARY ${JSON.stringify({ backend: race.kind, admitted, exceeded, wallMs: Date.now() - started })}`,
        );
      } finally {
        await opened.cleanup();
      }
    },
    { timeout: 120_000 },
  );
}
