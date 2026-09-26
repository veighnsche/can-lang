// E02 (X-R04-1): native SQL cancellation qualification probes, pinned Bun 1.4.2.
//
// Each leg retains the native `Query` handle and observes what `cancel()`
// actually does per dialect, with a second connection confirming the
// server-side state. These legs pin the QUALIFIED behavior: if a future
// Bun changes native cancel, the legs fail loudly and X-R04-1 must be
// re-qualified before any caller operand maps to it.
//
// Live legs need provisioned URLs plus one isolated database minted via
// `distribution/provision-local.sh db mkdb <name>`; without the URLs they
// skip visibly instead of passing vacuously. The URL and database name
// never appear in assertions or messages.
import { describe, expect, test } from "bun:test";

const PG_URL = process.env["CAN_TEST_POSTGRES_URL"];
const MYSQL_URL = process.env["CAN_TEST_MYSQL_URL"];
const RUN_DB = process.env["CAN_E02_DB"] ?? "can_e02_run1";
const pgLive = test.skipIf(PG_URL === undefined);
const myLive = test.skipIf(MYSQL_URL === undefined);

type Settled =
  | { readonly ok: true }
  | { readonly ok: false; readonly name: string; readonly code: string };

function describeCause(cause: unknown): { name: string; code: string } {
  if (cause instanceof Error) {
    const code = (cause as { code?: unknown }).code;
    return { name: cause.name, code: typeof code === "string" ? code : "" };
  }
  return { name: typeof cause, code: "" };
}

function watch<T>(query: Promise<T>): Promise<Settled> {
  return query.then(
    () => ({ ok: true as const }),
    (cause: unknown) => ({ ok: false as const, ...describeCause(cause) }),
  );
}

async function freshPG(url: string, max: number): Promise<InstanceType<typeof Bun.SQL>> {
  const sql = new Bun.SQL(url, { adapter: "postgres", bigint: true, max });
  await sql.connect();
  return sql;
}

async function freshMySQL(url: string, max: number): Promise<InstanceType<typeof Bun.SQL>> {
  const sql = new Bun.SQL(url, { adapter: "mysql", bigint: true, max, tls: true });
  await sql.connect();
  return sql;
}

async function pgSleepers(probe: InstanceType<typeof Bun.SQL>, db: string): Promise<number> {
  const rows =
    (await probe`SELECT count(*)::int AS n FROM pg_stat_activity WHERE datname = ${db} AND query LIKE '%pg_sleep%' AND pid <> pg_backend_pid()`) as Array<{
      n: number;
    }>;
  return rows[0]?.n ?? -1;
}

async function mysqlSleepers(probe: InstanceType<typeof Bun.SQL>): Promise<number> {
  const rows = (await probe`SHOW PROCESSLIST`) as Array<{ Info?: unknown }>;
  return rows.filter((row) => typeof row.Info === "string" && row.Info.includes("SLEEP")).length;
}

describe("X-R04-1 postgresql: cancel has no server-side effect", () => {
  pgLive(
    "mid-flight cancel leaves the backend running to completion",
    async () => {
      const url = PG_URL!.replace("/<db>", `/${RUN_DB}`);
      const sql = await freshPG(url, 2);
      try {
        const probe = await freshPG(url, 1);
        try {
          const query = sql`SELECT pg_sleep(6)`;
          const started = Date.now();
          const settled = watch(query);
          await Bun.sleep(600);
          // The backend is really executing before cancel is attempted.
          expect(await pgSleepers(probe, RUN_DB)).toBe(1);
          void query.cancel();
          expect(query.cancelled).toBe(true);
          const early = await Promise.race([settled, Bun.sleep(2500).then(() => "still-running")]);
          if (early !== "still-running") {
            throw new Error(
              "native pg cancel behavior changed; re-qualify X-R04-1 before mapping any operand",
            );
          }
          // Still executing server-side well after the cancel call.
          expect(await pgSleepers(probe, RUN_DB)).toBe(1);
          const final = await settled;
          expect(final).toEqual({ ok: true });
          expect(Date.now() - started).toBeGreaterThanOrEqual(6000);
        } finally {
          await probe.close();
        }
      } finally {
        await sql.close();
      }
    },
    15000,
  );

  pgLive("cancel before execution leaves the await unsettled", async () => {
    const url = PG_URL!.replace("/<db>", `/${RUN_DB}`);
    const sql = await freshPG(url, 1);
    try {
      const query = sql`SELECT 1 AS one`;
      void query.cancel();
      // A cancelled lazy query never settles: E04 must never cancel a
      // query that has not started executing.
      const race = await Promise.race([
        watch(query).then(() => "settled"),
        Bun.sleep(2000).then(() => "unsettled"),
      ]);
      expect(race).toBe("unsettled");
    } finally {
      await sql.close();
    }
  });

  pgLive("aborting a queued reservation is not query cancellation", async () => {
    const url = PG_URL!.replace("/<db>", `/${RUN_DB}`);
    const sql = await freshPG(url, 1);
    try {
      const holder = await sql.reserve();
      const held = watch(holder`SELECT pg_sleep(3)`);
      await Bun.sleep(300);
      const stop = new AbortController();
      const queued = sql.reserve({ signal: stop.signal }).then(
        () => "reserved",
        (cause: unknown) => `rejected:${cause instanceof Error ? cause.message : typeof cause}`,
      );
      await Bun.sleep(300);
      stop.abort(new Error("stop-waiting"));
      expect(await queued).toBe("rejected:stop-waiting");
      // The running query is unaffected by the reservation abort.
      expect(await held).toEqual({ ok: true });
      holder.release();
    } finally {
      await sql.close();
    }
  });

  pgLive("aborting a reserve signal does not touch its running query", async () => {
    const url = PG_URL!.replace("/<db>", `/${RUN_DB}`);
    const sql = await freshPG(url, 1);
    try {
      const stop = new AbortController();
      const reserved = await sql.reserve({ signal: stop.signal });
      const settled = watch(reserved`SELECT pg_sleep(2)`);
      await Bun.sleep(300);
      stop.abort(new Error("too-late"));
      expect(await settled).toEqual({ ok: true });
      reserved.release();
    } finally {
      await sql.close();
    }
  });
});

describe("X-R04-1 mysql: cancel has no server-side effect", () => {
  myLive(
    "mid-flight cancel leaves the server thread running to completion",
    async () => {
      const url = MYSQL_URL!.replace("/<db>", `/${RUN_DB}`);
      const sql = await freshMySQL(url, 2);
      try {
        const probe = await freshMySQL(url, 1);
        try {
          const query = sql`SELECT SLEEP(6)`;
          const started = Date.now();
          const settled = watch(query);
          await Bun.sleep(800);
          expect(await mysqlSleepers(probe)).toBe(1);
          void query.cancel();
          expect(query.cancelled).toBe(true);
          const early = await Promise.race([settled, Bun.sleep(2500).then(() => "still-running")]);
          if (early !== "still-running") {
            throw new Error(
              "native mysql cancel behavior changed; re-qualify X-R04-1 before mapping any operand",
            );
          }
          expect(await mysqlSleepers(probe)).toBe(1);
          const final = await settled;
          expect(final).toEqual({ ok: true });
          expect(Date.now() - started).toBeGreaterThanOrEqual(6000);
        } finally {
          await probe.close();
        }
      } finally {
        await sql.close();
      }
    },
    15000,
  );

  myLive("cancel before execution leaves the await unsettled", async () => {
    const url = MYSQL_URL!.replace("/<db>", `/${RUN_DB}`);
    const sql = await freshMySQL(url, 1);
    try {
      const query = sql`SELECT 1 AS one`;
      void query.cancel();
      const race = await Promise.race([
        watch(query).then(() => "settled"),
        Bun.sleep(2000).then(() => "unsettled"),
      ]);
      expect(race).toBe("unsettled");
    } finally {
      await sql.close();
    }
  });
});

describe("X-R04-1 sqlite: synchronous execution admits no interleaved cancel", () => {
  test("a timer due mid-query fires only after the query settles", async () => {
    const sql = new Bun.SQL(":memory:", { adapter: "sqlite" });
    try {
      const query = sql`WITH RECURSIVE c(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM c WHERE x < 8000) SELECT count(*) AS n FROM c a, c b`;
      const started = Date.now();
      let timerFiredAt = -1;
      const timer = setTimeout(() => {
        timerFiredAt = Date.now() - started;
        void query.cancel();
      }, 50);
      try {
        const rows = (await query) as Array<{ n: number }>;
        const elapsed = Date.now() - started;
        const cancelledAtSettle = query.cancelled;
        expect(rows).toEqual([{ n: 64000000 }]);
        // The 50ms timer was due long before the synchronous scan
        // finished, yet cancel never interleaved: the flag flips only
        // after settlement, when the event loop runs again.
        expect(elapsed).toBeGreaterThan(50);
        expect(cancelledAtSettle).toBe(false);
        expect(timerFiredAt).toBe(-1);
      } finally {
        clearTimeout(timer);
      }
    } finally {
      await sql.close();
    }
  });

  test("cancel before execution leaves the await unsettled", async () => {
    const sql = new Bun.SQL(":memory:", { adapter: "sqlite" });
    try {
      const query = sql`SELECT 1 AS one`;
      void query.cancel();
      const race = await Promise.race([
        watch(query).then(() => "settled"),
        Bun.sleep(1000).then(() => "unsettled"),
      ]);
      expect(race).toBe("unsettled");
    } finally {
      await sql.close();
    }
  });
});
