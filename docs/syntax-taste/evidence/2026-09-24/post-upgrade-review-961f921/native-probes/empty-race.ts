import { settle } from "/Users/vince/Projects/can-lang/runtime/coordination.ts";
import { runOwnedRoot } from "/Users/vince/Projects/can-lang/runtime/owner.ts";

const observed = await Promise.race([
  runOwnedRoot(async () => {
    await settle("race", []);
    return { kind: "ok" as const, value: 0 };
  }),
  new Promise<string>((resolve) => setTimeout(() => resolve("timed out"), 50)),
]);
console.log(observed);
