import { test, expect } from "bun:test";
import { mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { epochSchedule } from "../outbound/epoch.ts";
import {
  configurePool,
  createMemoryLedgerStore,
  epochStatus,
  fence,
  ledgerHealth,
  meteringProfile,
  meteringUsage,
  profileId,
  reconcile,
  releaseUndispatched,
  reserve,
  settle,
  transitionPool,
  usageTotal,
  type LedgerState,
  type LedgerStore,
  type ReserveInput,
} from "../outbound/ledger.ts";
import { openFileLedgerStore } from "../outbound/file-ledger.ts";
import { LEDGER_STATEMENTS, ledgerDDL } from "../outbound/ledger-schema.ts";

const HOUR = 3_600_000;
const PROFILE = { provider: "typesafe", model: "jev", version: "1.13.0" };

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

test("pool configuration pins the first schedule and rejects re-pinning", async () => {
  const store = createMemoryLedgerStore();
  await configured(store);
  expect(await configurePool(store, schedule(500))).toEqual({
    configured: false,
    reason: "invalid-policy",
  });
  expect(await transitionPool(store, "missing", { version: "2" }, 0)).toEqual({
    transitioned: false,
    reason: "invalid-policy",
  });
  const moved = await transitionPool(store, "default", { version: "2" }, 10);
  expect(moved.transitioned).toBe(true);
  // A version-preserving transition is a caller bug: it rejects rather
  // than writing a no-op schedule row.
  await expect(transitionPool(store, "default", { version: "2" }, 10)).rejects.toThrow(TypeError);
});

test("atomic exact-fit admission and immediate typed rejection of excess", async () => {
  const store = createMemoryLedgerStore();
  await configured(store, 250);
  const first = await reserve(store, input({ upperBound: 100, nowMs: 7 }));
  expect(first.outcome).toBe("admitted");
  if (first.outcome !== "admitted") throw new Error("unreachable");
  expect(first.duplicate).toBe(false);
  expect(first.remaining).toBe(150);
  expect(first.epochResetMs).toBe(HOUR);
  expect(first.record.state).toBe("held");
  // Exact fit consumes the last headroom atomically.
  const exact = await reserve(
    store,
    input({ upperBound: 150, correlation: "req-2", invocationId: "inv-exact", nowMs: 8 }),
  );
  expect(exact.outcome).toBe("admitted");
  if (exact.outcome !== "admitted") throw new Error("unreachable");
  expect(exact.remaining).toBe(0);
  // One token more rejects immediately with the exceeded detail H maps
  // to ai_budget::exceeded: pool, required U, remaining, reset time.
  const over = await reserve(
    store,
    input({ upperBound: 1, correlation: "req-3", invocationId: "inv-over", nowMs: 9 }),
  );
  if (over.outcome !== "exceeded") throw new Error("unreachable");
  expect({ ...over, pool: over.pool as string }).toEqual({
    outcome: "exceeded",
    pool: "default",
    required: 1,
    remaining: 0,
    epochResetMs: HOUR,
  });
  const status = await epochStatus(store, "acme", "default", 0);
  expect(status).toEqual({
    status: "found",
    row: expect.objectContaining({ committed: 0, held: 250, limit: 250 }),
    unresolved: 2,
  });
});

test("reserve rejects unqualified, unconfigured, and pre-anchor admission", async () => {
  const store = createMemoryLedgerStore();
  await configured(store);
  const unqualified = await reserve(store, input({ qualified: false }));
  expect(unqualified.outcome).toBe("unavailable");
  if (unqualified.outcome !== "unavailable") throw new Error("unreachable");
  expect(unqualified.reason).toBe("missing-qualification");
  expect(unqualified.correlation as string).toBe("req-1");
  expect(await reserve(store, input({ pool: "no-such-pool", nowMs: 5 }))).toEqual(
    expect.objectContaining({ outcome: "unavailable", reason: "invalid-policy" }),
  );
  const future = createMemoryLedgerStore();
  await configurePool(future, schedule(100, HOUR, 10 * HOUR, "1"));
  expect(await reserve(future, input({ nowMs: 5 }))).toEqual(
    expect.objectContaining({ outcome: "unavailable", reason: "invalid-policy" }),
  );
  for (const bad of [
    input({ tenant: "" }),
    input({ pool: "" }),
    input({ correlation: "" }),
    input({ upperBound: 0 }),
    input({ upperBound: -3 }),
    input({ upperBound: 1.5 }),
    input({ profile: { provider: "", model: "m", version: "v" } }),
    input({ nowMs: -1 }),
  ]) {
    await expect(reserve(store, bad)).rejects.toThrow(TypeError);
  }
});

test("reserve retries by invocation ID instead of double-charging", async () => {
  const store = createMemoryLedgerStore();
  await configured(store, 150);
  const first = await reserve(store, input({ invocationId: "inv-retry", nowMs: 5 }));
  expect(first.outcome).toBe("admitted");
  // Same ID with identical parameters returns the admission again.
  const retry = await reserve(store, input({ invocationId: "inv-retry", nowMs: 6 }));
  expect(retry).toEqual(expect.objectContaining({ outcome: "admitted", duplicate: true }));
  const status = await epochStatus(store, "acme", "default", 0);
  expect(status).toEqual(expect.objectContaining({ row: expect.objectContaining({ held: 100 }) }));
  // Same ID with different parameters is a metering conflict, not a
  // second reservation.
  expect(
    await reserve(store, input({ invocationId: "inv-retry", upperBound: 50, nowMs: 6 })),
  ).toEqual(expect.objectContaining({ outcome: "unavailable", reason: "conflicting-settlement" }));
});

test("fencing is idempotent and never follows settlement", async () => {
  const store = createMemoryLedgerStore();
  await configured(store);
  const admitted = await reserve(store, input({ invocationId: "inv-fence", nowMs: 5 }));
  if (admitted.outcome !== "admitted") throw new Error("unreachable");
  const fenced = await fence(store, "inv-fence");
  expect(fenced).toEqual(
    expect.objectContaining({
      outcome: "fenced",
      duplicate: false,
      record: expect.objectContaining({ state: "fenced", fenced: true }),
    }),
  );
  expect(await fence(store, "inv-fence")).toEqual(
    expect.objectContaining({ outcome: "fenced", duplicate: true }),
  );
  expect(await fence(store, "inv-missing")).toEqual({
    outcome: "unavailable",
    reason: "unknown-invocation",
  });
  const settled = await settle(store, {
    invocationId: "inv-fence",
    inputTokens: 30,
    outputTokens: 20,
  });
  expect(settled.outcome).toBe("settled");
  expect(await fence(store, "inv-fence")).toEqual({
    outcome: "unavailable",
    reason: "conflicting-settlement",
  });
  await expect(fence(store, "")).rejects.toThrow(TypeError);
});

test("settlement debits actuals, releases headroom, and stays idempotent", async () => {
  const store = createMemoryLedgerStore();
  await configured(store, 200);
  await reserve(store, input({ invocationId: "inv-settle", nowMs: 5 }));
  const done = await settle(store, {
    invocationId: "inv-settle",
    inputTokens: 30,
    outputTokens: 20,
  });
  expect(done).toEqual(
    expect.objectContaining({
      outcome: "settled",
      actual: 50,
      released: 50,
      duplicate: false,
      record: expect.objectContaining({
        state: "settled",
        usage: { inputTokens: 30, outputTokens: 20 },
        breach: false,
      }),
    }),
  );
  const status = await epochStatus(store, "acme", "default", 0);
  expect(status).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 50, held: 0 }) }),
  );
  // Combined input+output accounting: a second settlement with the same
  // counters is a no-op, different counters conflict.
  expect(
    await settle(store, { invocationId: "inv-settle", inputTokens: 30, outputTokens: 20 }),
  ).toEqual(expect.objectContaining({ outcome: "settled", duplicate: true, released: 0 }));
  expect(
    await settle(store, { invocationId: "inv-settle", inputTokens: 31, outputTokens: 20 }),
  ).toEqual({ outcome: "unavailable", reason: "conflicting-settlement" });
  expect(
    await settle(store, { invocationId: "inv-missing", inputTokens: 1, outputTokens: 1 }),
  ).toEqual({ outcome: "unavailable", reason: "unknown-invocation" });
});

test("invalid usage keeps the full hold unresolved", async () => {
  const store = createMemoryLedgerStore();
  await configured(store, 200);
  await reserve(store, input({ invocationId: "inv-unknown", nowMs: 5 }));
  for (const usage of [
    { inputTokens: -1, outputTokens: 0 },
    { inputTokens: 0, outputTokens: -5 },
    { inputTokens: 1.5, outputTokens: 0 },
    { inputTokens: Number.NaN, outputTokens: 0 },
    { inputTokens: Number.MAX_SAFE_INTEGER, outputTokens: 1 },
    { inputTokens: "30", outputTokens: 0 },
    { inputTokens: undefined, outputTokens: 0 },
  ]) {
    expect(await settle(store, { invocationId: "inv-unknown", ...usage })).toEqual({
      outcome: "unavailable",
      reason: "invalid-usage",
    });
  }
  // The hold is untouched and the attempt still reconciles as held.
  const status = await epochStatus(store, "acme", "default", 0);
  expect(status).toEqual(
    expect.objectContaining({
      row: expect.objectContaining({ committed: 0, held: 100 }),
      unresolved: 1,
    }),
  );
  expect(await reconcile(store, "inv-unknown")).toEqual(
    expect.objectContaining({
      status: "found",
      record: expect.objectContaining({ state: "held" }),
    }),
  );
  // Authoritative usage still settles the same attempt later.
  expect(
    await settle(store, { invocationId: "inv-unknown", inputTokens: 40, outputTokens: 10 }),
  ).toEqual(expect.objectContaining({ outcome: "settled", actual: 50, released: 50 }));
});

test("usage above the bound records actuals and quarantines the profile", async () => {
  const store = createMemoryLedgerStore();
  await configured(store, 1000);
  await reserve(store, input({ invocationId: "inv-breach", upperBound: 100, nowMs: 5 }));
  const breached = await settle(store, {
    invocationId: "inv-breach",
    inputTokens: 400,
    outputTokens: 300,
  });
  expect(breached).toEqual(
    expect.objectContaining({
      outcome: "settled",
      actual: 700,
      released: 0,
      duplicate: false,
      breach: { profile: "typesafe/jev/1.13.0", upperBound: 100, actual: 700 },
      record: expect.objectContaining({ breach: true }),
    }),
  );
  // Actuals record without clamping: the row shows 700 committed.
  const status = await epochStatus(store, "acme", "default", 0);
  expect(status).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 700, held: 0 }) }),
  );
  // The breached profile cannot admit new sends; others still can.
  expect(await reserve(store, input({ correlation: "req-2", nowMs: 6 }))).toEqual(
    expect.objectContaining({ outcome: "unavailable", reason: "breached-profile" }),
  );
  const other = await reserve(
    store,
    input({
      correlation: "req-3",
      profile: { provider: "typesafe", model: "jev", version: "1.14.0" },
      upperBound: 100,
      nowMs: 6,
    }),
  );
  expect(other.outcome).toBe("admitted");
  const health = await ledgerHealth(store);
  expect(health).toEqual(
    expect.objectContaining({
      status: "ok",
      committed: 700,
      held: 100,
      unresolved: 1,
      quarantined: ["typesafe/jev/1.13.0"],
    }),
  );
});

test("release needs no-dispatch proof: fenced attempts keep their hold", async () => {
  const store = createMemoryLedgerStore();
  await configured(store, 300);
  await reserve(store, input({ invocationId: "inv-early", nowMs: 5 }));
  await reserve(store, input({ invocationId: "inv-sent", correlation: "req-2", nowMs: 5 }));
  await fence(store, "inv-sent");
  // Never fenced: the fence proves no send, U releases.
  expect(await releaseUndispatched(store, "inv-early")).toEqual(
    expect.objectContaining({
      outcome: "released",
      duplicate: false,
      record: expect.objectContaining({ state: "released" }),
    }),
  );
  expect(await releaseUndispatched(store, "inv-early")).toEqual(
    expect.objectContaining({ outcome: "released", duplicate: true }),
  );
  // Fenced: no proof of no-dispatch, the full hold stays unresolved.
  expect(await releaseUndispatched(store, "inv-sent")).toEqual(
    expect.objectContaining({
      outcome: "held",
      record: expect.objectContaining({ state: "fenced" }),
    }),
  );
  const status = await epochStatus(store, "acme", "default", 0);
  expect(status).toEqual(
    expect.objectContaining({
      row: expect.objectContaining({ committed: 0, held: 100 }),
      unresolved: 1,
    }),
  );
  expect(await releaseUndispatched(store, "inv-missing")).toEqual({
    outcome: "unavailable",
    reason: "unknown-invocation",
  });
  // Usage arriving after a proven no-dispatch release is conflicting
  // evidence, not a settlement.
  expect(
    await settle(store, { invocationId: "inv-early", inputTokens: 1, outputTokens: 1 }),
  ).toEqual({ outcome: "unavailable", reason: "conflicting-settlement" });
});

test("late usage settles to the original admission epoch", async () => {
  const store = createMemoryLedgerStore();
  await configured(store, 200);
  await reserve(store, input({ invocationId: "inv-late", nowMs: HOUR - 10 }));
  // The next epoch opens with fresh headroom for new admissions.
  const next = await reserve(
    store,
    input({ invocationId: "inv-next", correlation: "req-2", upperBound: 200, nowMs: HOUR + 1 }),
  );
  expect(next.outcome).toBe("admitted");
  // Late actuals still debit the original epoch, not the current one.
  expect(
    await settle(store, { invocationId: "inv-late", inputTokens: 60, outputTokens: 40 }),
  ).toEqual(expect.objectContaining({ outcome: "settled", actual: 100 }));
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 100, held: 0 }) }),
  );
  expect(await epochStatus(store, "acme", "default", HOUR)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 0, held: 200 }) }),
  );
});

test("epoch transitions keep live rows pinned and govern future epochs", async () => {
  const store = createMemoryLedgerStore();
  await configured(store, 200);
  await reserve(store, input({ invocationId: "inv-old", nowMs: 5 }));
  const moved = await transitionPool(store, "default", { limit: 50, version: "2" }, 10);
  expect(moved).toEqual(
    expect.objectContaining({
      transitioned: true,
      schedule: expect.objectContaining({ limit: 50, anchorMs: HOUR, version: "2" }),
    }),
  );
  // The live epoch keeps its pinned limit: 100 more still fits the 200.
  expect(
    await reserve(store, input({ invocationId: "inv-old2", correlation: "req-2", nowMs: 11 })),
  ).toEqual(expect.objectContaining({ outcome: "admitted" }));
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({
      row: expect.objectContaining({ limit: 200, scheduleVersion: "1", held: 200 }),
    }),
  );
  // Future epochs admit under the new schedule version and limit.
  expect(
    await reserve(
      store,
      input({ invocationId: "inv-new", correlation: "req-3", upperBound: 50, nowMs: HOUR + 1 }),
    ),
  ).toEqual(expect.objectContaining({ outcome: "admitted" }));
  expect(await epochStatus(store, "acme", "default", HOUR)).toEqual(
    expect.objectContaining({
      row: expect.objectContaining({ limit: 50, scheduleVersion: "2", held: 50 }),
    }),
  );
  expect(
    await reserve(
      store,
      input({ invocationId: "inv-new2", correlation: "req-4", upperBound: 1, nowMs: HOUR + 2 }),
    ),
  ).toEqual(expect.objectContaining({ outcome: "exceeded", remaining: 0 }));
});

test("pre-anchor transitions replace a schedule before it governs", async () => {
  const store = createMemoryLedgerStore();
  await configurePool(store, schedule(200, HOUR, 10 * HOUR, "1"));
  // Replacing the future schedule keeps one anchor: the newest version
  // with a reached anchor governs, so the stillborn v1 never admits.
  const moved = await transitionPool(store, "default", { limit: 80, version: "2" }, 5);
  expect(moved).toEqual(
    expect.objectContaining({
      transitioned: true,
      schedule: expect.objectContaining({ limit: 80, anchorMs: 10 * HOUR, version: "2" }),
    }),
  );
  expect(await reserve(store, input({ nowMs: 6 }))).toEqual(
    expect.objectContaining({ outcome: "unavailable", reason: "invalid-policy" }),
  );
  expect(
    await reserve(store, input({ invocationId: "inv-v2", upperBound: 80, nowMs: 10 * HOUR })),
  ).toEqual(expect.objectContaining({ outcome: "admitted" }));
  expect(await epochStatus(store, "acme", "default", 10 * HOUR)).toEqual(
    expect.objectContaining({
      row: expect.objectContaining({ limit: 80, scheduleVersion: "2", held: 80 }),
    }),
  );
});

test("concurrent reserves stay exact under contention", async () => {
  const store = createMemoryLedgerStore();
  await configured(store, 1000);
  const outcomes = await Promise.all(
    Array.from({ length: 50 }, (_, index) =>
      reserve(
        store,
        input({
          invocationId: `inv-race-${index}`,
          correlation: `req-${index}`,
          upperBound: 30,
          nowMs: 5,
        }),
      ),
    ),
  );
  const admitted = outcomes.filter((outcome) => outcome.outcome === "admitted");
  const exceeded = outcomes.filter((outcome) => outcome.outcome === "exceeded");
  // floor(1000 / 30) = 33 admissions; the rest reject, none duplicate.
  expect(admitted.length).toBe(33);
  expect(exceeded.length).toBe(17);
  const status = await epochStatus(store, "acme", "default", 0);
  expect(status).toEqual(
    expect.objectContaining({
      row: expect.objectContaining({ committed: 0, held: 990 }),
      unresolved: 33,
    }),
  );
});

test("commit-unknown recovery reconciles by invocation ID", async () => {
  const backing = createMemoryLedgerStore();
  await configured(backing);
  let commits = 0;
  let persistThenThrow = true;
  const flaky: LedgerStore = {
    name: "flaky",
    load: () => backing.load(),
    // First commit persists, then reports failure: the classic
    // ambiguous outcome the caller must reconcile, never assume.
    commit: async (expectedVersion: number, next: LedgerState) => {
      commits += 1;
      const result = await backing.commit(expectedVersion, next);
      if (persistThenThrow) {
        persistThenThrow = false;
        throw new Error("connection reset during commit");
      }
      return result;
    },
  };
  const ambiguous = await reserve(
    flaky,
    input({ invocationId: "inv-ambiguous", correlation: "req-amb", nowMs: 5 }),
  );
  expect(ambiguous).toEqual(
    expect.objectContaining({
      outcome: "unavailable",
      reason: "ambiguous-commit",
      correlation: "req-amb",
      invocationId: "inv-ambiguous",
    }),
  );
  expect(commits).toBe(1);
  // Reconciliation finds the durable record the commit hid.
  expect(await reconcile(flaky, "inv-ambiguous")).toEqual(
    expect.objectContaining({
      status: "found",
      record: expect.objectContaining({ state: "held", upperBound: 100 }),
    }),
  );
  // Retrying the same reservation returns it without double-charging.
  const retry = await reserve(
    flaky,
    input({ invocationId: "inv-ambiguous", correlation: "req-amb", nowMs: 6 }),
  );
  expect(retry).toEqual(expect.objectContaining({ outcome: "admitted", duplicate: true }));
  const status = await epochStatus(flaky, "acme", "default", 0);
  expect(status).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ held: 100 }), unresolved: 1 }),
  );
  // Absent IDs stay absent: never dispatch on an unknown reservation.
  expect(await reconcile(flaky, "inv-never-reserved")).toEqual({ status: "absent" });
});

test("unavailable storage fails closed across every operation", async () => {
  const down: LedgerStore = {
    name: "down",
    load: async () => {
      throw new Error("storage unreachable");
    },
    commit: async () => {
      throw new Error("storage unreachable");
    },
  };
  expect(await reserve(down, input({ nowMs: 5 }))).toEqual(
    expect.objectContaining({ outcome: "unavailable", reason: "storage-unavailable" }),
  );
  expect(await fence(down, "inv-x")).toEqual({
    outcome: "unavailable",
    reason: "storage-unavailable",
  });
  expect(await settle(down, { invocationId: "inv-x", inputTokens: 1, outputTokens: 1 })).toEqual({
    outcome: "unavailable",
    reason: "storage-unavailable",
  });
  expect(await releaseUndispatched(down, "inv-x")).toEqual({
    outcome: "unavailable",
    reason: "storage-unavailable",
  });
  expect(await reconcile(down, "inv-x")).toEqual({
    status: "unavailable",
    reason: "storage-unavailable",
  });
  expect(await epochStatus(down, "acme", "default", 0)).toEqual({
    status: "unavailable",
    reason: "storage-unavailable",
  });
  expect(await ledgerHealth(down)).toEqual({
    status: "unavailable",
    reason: "storage-unavailable",
  });
});

test("file backend survives restart with holds intact", async () => {
  const directory = await mkdtemp(join(tmpdir(), "ledger-restart-"));
  try {
    const path = join(directory, "ledger.json");
    const first = openFileLedgerStore(path);
    await configured(first, 200);
    await reserve(first, input({ invocationId: "inv-restart", nowMs: 5 }));
    await fence(first, "inv-restart");
    // Reopening the path is the whole recovery procedure.
    const second = openFileLedgerStore(path);
    expect(await reconcile(second, "inv-restart")).toEqual(
      expect.objectContaining({
        status: "found",
        record: expect.objectContaining({ state: "fenced", fenced: true, upperBound: 100 }),
      }),
    );
    expect(await epochStatus(second, "acme", "default", 0)).toEqual(
      expect.objectContaining({
        row: expect.objectContaining({ committed: 0, held: 100, scheduleVersion: "1" }),
        unresolved: 1,
      }),
    );
    // No timer, disposal, or restart refunds the hold: settlement still
    // needs authoritative usage.
    expect(
      await settle(second, { invocationId: "inv-restart", inputTokens: 70, outputTokens: 10 }),
    ).toEqual(expect.objectContaining({ outcome: "settled", actual: 80, released: 20 }));
    const health = await ledgerHealth(second);
    expect(health).toEqual(
      expect.objectContaining({ status: "ok", committed: 80, held: 0, unresolved: 0 }),
    );
    // Lockfiles never leak out of completed operations.
    const { readdir } = await import("node:fs/promises");
    expect((await readdir(directory)).sort()).toEqual(["ledger.json"]);
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
});

test("file backend serializes concurrent writers exactly", async () => {
  const directory = await mkdtemp(join(tmpdir(), "ledger-file-race-"));
  try {
    const path = join(directory, "ledger.json");
    const store = openFileLedgerStore(path);
    await configured(store, 500);
    const outcomes = await Promise.all(
      Array.from({ length: 20 }, (_, index) =>
        reserve(
          store,
          input({
            invocationId: `inv-file-${index}`,
            correlation: `req-${index}`,
            upperBound: 40,
            nowMs: 5,
          }),
        ),
      ),
    );
    // floor(500 / 40) = 12 admissions at 480 held; zero lost updates.
    expect(outcomes.filter((outcome) => outcome.outcome === "admitted").length).toBe(12);
    expect(outcomes.filter((outcome) => outcome.outcome === "exceeded").length).toBe(8);
    const reopened = openFileLedgerStore(path);
    expect(await epochStatus(reopened, "acme", "default", 0)).toEqual(
      expect.objectContaining({
        row: expect.objectContaining({ held: 480 }),
        unresolved: 12,
      }),
    );
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
});

test("file backend fails closed on corruption and bad paths", async () => {
  const directory = await mkdtemp(join(tmpdir(), "ledger-corrupt-"));
  try {
    const path = join(directory, "ledger.json");
    await writeFile(path, "{ not valid json");
    const corrupt = openFileLedgerStore(path);
    expect(await reserve(corrupt, input({ nowMs: 5 }))).toEqual(
      expect.objectContaining({ outcome: "unavailable", reason: "storage-unavailable" }),
    );
    expect(await reconcile(corrupt, "inv-x")).toEqual({
      status: "unavailable",
      reason: "storage-unavailable",
    });
    // Corruption never auto-repairs into an empty ledger.
    const { readFile } = await import("node:fs/promises");
    expect(await readFile(path, "utf8")).toBe("{ not valid json");
    expect(() => openFileLedgerStore("")).toThrow(TypeError);
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
});

test("metering helpers pin profile identity and validate provider counters", () => {
  const profile = meteringProfile(PROFILE);
  expect(profileId(profile)).toBe("typesafe/jev/1.13.0");
  expect(Object.isFrozen(profile)).toBe(true);
  expect(() => meteringProfile({ provider: "", model: "m", version: "v" })).toThrow(TypeError);
  expect(() => meteringProfile({ provider: "p", model: "m", version: "x".repeat(257) })).toThrow(
    TypeError,
  );
  const usage = meteringUsage(400, 300);
  expect(usage).toEqual({ inputTokens: 400, outputTokens: 300 });
  expect(usageTotal(usage!)).toBe(700);
  expect(meteringUsage(-1, 0)).toBeUndefined();
  expect(meteringUsage(0, 1.5)).toBeUndefined();
  expect(meteringUsage(Number.MAX_SAFE_INTEGER, 1)).toBeUndefined();
});

test("health reports distinguish committed, held, and unresolved without secrets", async () => {
  const store = createMemoryLedgerStore();
  await configured(store, 1000);
  await reserve(store, input({ invocationId: "inv-h1", nowMs: 5 }));
  await reserve(store, input({ invocationId: "inv-h2", correlation: "req-2", nowMs: 5 }));
  await fence(store, "inv-h1");
  await settle(store, { invocationId: "inv-h2", inputTokens: 25, outputTokens: 25 });
  const health = await ledgerHealth(store);
  expect(health).toEqual({
    status: "ok",
    epochs: 1,
    committed: 50,
    held: 100,
    unresolved: 1,
    quarantined: [],
  });
  const exposed = JSON.stringify({
    health,
    status: await epochStatus(store, "acme", "default", 0),
  });
  expect(exposed.includes("credential")).toBe(false);
  expect(exposed.includes("endpoint")).toBe(false);
  expect(await epochStatus(store, "acme", "default", HOUR)).toEqual({ status: "absent" });
});

function staticTemplate(statement: string): TemplateStringsArray {
  const strings = Object.freeze(
    Object.assign([statement], { raw: Object.freeze([statement]) }),
  ) as unknown as TemplateStringsArray;
  return strings;
}

async function expectSqlThrow(operation: Promise<unknown>): Promise<void> {
  try {
    await operation;
  } catch (cause) {
    expect(String((cause as { code?: unknown }).code ?? cause)).toContain("CONSTRAINT");
    return;
  }
  throw new Error("expected SQL to violate its CHECK floor");
}

test("the SQLite ledger schema executes with its CHECK floor", async () => {
  const sql = new Bun.SQL({ adapter: "sqlite", filename: ":memory:" });
  try {
    for (const statement of LEDGER_STATEMENTS.sqlite) {
      await sql(staticTemplate(statement));
    }
    // Version row seeds at zero; the CAS update is the commit gate.
    const seeded = (await sql(
      staticTemplate(`SELECT version FROM ai_budget_state WHERE id = 1`),
    )) as unknown as {
      version: number;
    }[];
    expect(seeded.map((row) => row.version)).toEqual([0]);
    // Schedule versions round-trip keyed by pool and anchor.
    await sql(
      staticTemplate(
        `INSERT INTO ai_budget_schedules (pool, token_limit, period_ms, anchor_ms, version)
          VALUES ('default', 200, 3600000, 0, '1'),
                 ('default', 50, 3600000, 3600000, '2')`,
      ),
    );
    const schedules = (await sql(
      staticTemplate(
        `SELECT version FROM ai_budget_schedules
          WHERE pool = 'default' AND anchor_ms <= 4000000 ORDER BY anchor_ms DESC LIMIT 1`,
      ),
    )) as unknown as { version: string }[];
    expect(schedules).toEqual([{ version: "2" }]);
    // Epoch and invocation rows round-trip through the published shape.
    await sql(
      staticTemplate(
        `INSERT INTO ai_budget_epochs
          (tenant, pool, epoch_start, epoch_end, token_limit, schedule_version, committed, held)
          VALUES ('acme', 'default', 0, 3600000, 200, '1', 50, 100)`,
      ),
    );
    await sql(
      staticTemplate(
        `INSERT INTO ai_budget_invocations
          (invocation_id, tenant, pool, correlation, epoch_start, epoch_end, token_limit,
           schedule_version, provider, model, profile_version, upper_bound, state, fenced, breach)
          VALUES ('inv-1', 'acme', 'default', 'req-1', 0, 3600000, 200,
           '1', 'typesafe', 'jev', '1.13.0', 100, 'held', 0, 0)`,
      ),
    );
    const rows = (await sql(
      staticTemplate(`SELECT committed, held FROM ai_budget_epochs WHERE tenant = 'acme'`),
    )) as unknown as { committed: number; held: number }[];
    expect(rows).toEqual([{ committed: 50, held: 100 }]);
    // The CHECK floor rejects negative counters and open-ended states.
    // Bun.SQL query promises settle only under direct await, so the
    // assertions below use try/catch instead of .rejects matchers.
    await expectSqlThrow(
      sql(
        staticTemplate(
          `INSERT INTO ai_budget_epochs
            (tenant, pool, epoch_start, epoch_end, token_limit, schedule_version, committed, held)
            VALUES ('acme', 'default', 3600000, 7200000, 200, '1', -1, 0)`,
        ),
      ),
    );
    await expectSqlThrow(
      sql(
        staticTemplate(
          `UPDATE ai_budget_invocations SET state = 'maybe' WHERE invocation_id = 'inv-1'`,
        ),
      ),
    );
  } finally {
    await sql.close();
  }
});

test("every dialect publishes the five ledger tables plus the seed", () => {
  for (const dialect of ["sqlite", "postgres", "mysql"] as const) {
    const ddl = ledgerDDL(dialect);
    for (const table of [
      "ai_budget_state",
      "ai_budget_schedules",
      "ai_budget_epochs",
      "ai_budget_invocations",
      "ai_budget_quarantine",
    ]) {
      expect(ddl.includes(`CREATE TABLE ${table}`)).toBe(true);
    }
    expect(ddl.includes("INSERT INTO ai_budget_state (id, version) VALUES (1, 0);")).toBe(true);
  }
  // MySQL keys fit the utf8mb4 prefix limit; Postgres counts in BIGINT.
  expect(ledgerDDL("mysql").includes("VARCHAR(191)")).toBe(true);
  expect(ledgerDDL("mysql").includes("TEXT PRIMARY KEY")).toBe(false);
  expect(ledgerDDL("postgres").includes("BIGINT")).toBe(true);
  expect(() => ledgerDDL("duckdb" as never)).toThrow(TypeError);
});
