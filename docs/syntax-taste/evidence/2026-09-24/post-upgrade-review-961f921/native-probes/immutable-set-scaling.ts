import { createSet } from "/Users/vince/Projects/can-lang/runtime/collections/set.ts";

const ops = createSet("probe", "int");
for (const n of [10_000, 20_000, 40_000]) {
  let current = (await ops.empty()).value;
  const start = performance.now();
  for (let i = 0; i < n; i++) current = (await ops.add(current, BigInt(i))).value;
  console.log("immutable_set", n, Math.round(performance.now() - start));
}
