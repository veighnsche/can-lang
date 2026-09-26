// Two-process admission worker for runtime/test/ai-budget.test.ts.
// One argv-driven reservation burst against a shared file ledger; the
// parent asserts exact cross-process totals. Not a test itself.
import { openFileLedgerStore } from "../outbound/file-ledger.ts";
import { reserve } from "../outbound/ledger.ts";

const [path, tenant, pool, prefix, countText, boundText, nowText] = Bun.argv.slice(2);
const count = Number(countText);
const upperBound = Number(boundText);
const nowMs = Number(nowText);
if (
  path === undefined ||
  tenant === undefined ||
  pool === undefined ||
  prefix === undefined ||
  !Number.isSafeInteger(count) ||
  !Number.isSafeInteger(upperBound) ||
  !Number.isSafeInteger(nowMs)
) {
  console.error(
    "usage: ai-budget-admission-worker.ts <path> <tenant> <pool> <prefix> <count> <bound> <nowMs>",
  );
  process.exit(2);
}
const store = openFileLedgerStore(path);
let admitted = 0;
let exceeded = 0;
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
console.log(JSON.stringify({ admitted, exceeded }));
