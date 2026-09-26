// Cross-process ledger race worker for runtime/test/outbound-ledger-sql.test.ts.
// One argv-driven reservation burst against a shared ledger; the parent
// asserts exact cross-process totals. Not a test itself.
//
// Usage: ledger-race-worker.ts <kind> <target> <tenant> <pool> <prefix> <count> <bound> <nowMs>
// kind: file | sqlite | postgres | mysql
// target: filesystem path for file/sqlite, or the NAME of an env var
// holding the database URL for postgres/mysql (the URL itself never
// travels on argv, keeping credentials out of process listings).
import { openFileLedgerStore } from "../outbound/file-ledger.ts";
import { reserve, type LedgerStore } from "../outbound/ledger.ts";
import { openSqlLedgerStore, type SqlLedgerStore } from "../outbound/sql-ledger.ts";

const [kind, target, tenant, pool, prefix, countText, boundText, nowText] = Bun.argv.slice(2);
const count = Number(countText);
const upperBound = Number(boundText);
const nowMs = Number(nowText);
if (
  (kind !== "file" && kind !== "sqlite" && kind !== "postgres" && kind !== "mysql") ||
  target === undefined ||
  tenant === undefined ||
  pool === undefined ||
  prefix === undefined ||
  !Number.isSafeInteger(count) ||
  count < 1 ||
  !Number.isSafeInteger(upperBound) ||
  upperBound < 1 ||
  !Number.isSafeInteger(nowMs) ||
  nowMs < 0
) {
  console.error(
    "usage: ledger-race-worker.ts <file|sqlite|postgres|mysql> <target> <tenant> <pool> <prefix> <count> <bound> <nowMs>",
  );
  process.exit(2);
}

let store: LedgerStore;
let close: (() => Promise<void>) | undefined;
if (kind === "file") {
  store = openFileLedgerStore(target);
} else if (kind === "sqlite") {
  const opened: SqlLedgerStore = await openSqlLedgerStore({ dialect: "sqlite", filename: target });
  store = opened;
  close = () => opened.close();
} else {
  const url = process.env[target];
  if (url === undefined || url === "") {
    console.error(`missing database URL env var: ${target}`);
    process.exit(2);
  }
  const opened: SqlLedgerStore = await openSqlLedgerStore(
    kind === "postgres" ? { dialect: "postgres", url } : { dialect: "mysql", url },
  );
  store = opened;
  close = () => opened.close();
}

let admitted = 0;
let exceeded = 0;
try {
  for (let index = 0; index < count; index += 1) {
    const outcome = await reserve(store, {
      tenant,
      pool,
      correlation: `${prefix}-req-${index}`,
      profile: { provider: "typesafe", model: "jev", version: "1.13.0" },
      qualified: true,
      upperBound,
      invocationId: `${prefix}-inv-${index}`,
      nowMs,
    });
    if (outcome.outcome === "admitted") admitted += 1;
    else if (outcome.outcome === "exceeded") exceeded += 1;
    else {
      console.error(`unexpected outcome: ${outcome.outcome}`);
      process.exit(1);
    }
  }
} finally {
  if (close !== undefined) await close();
}
console.log(JSON.stringify({ admitted, exceeded }));
