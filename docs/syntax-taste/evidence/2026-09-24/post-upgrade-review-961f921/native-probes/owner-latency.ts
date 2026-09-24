import { settle } from "/Users/vince/Projects/can-lang/runtime/coordination.ts";
import { runOwnedRoot } from "/Users/vince/Projects/can-lang/runtime/owner.ts";
import { success } from "/Users/vince/Projects/can-lang/runtime/completion.ts";

const start = performance.now();
const result = await runOwnedRoot(async () => {
  const selected = await settle("any", [
    { captures: [], run: () => success(1) },
    { captures: [], run: async () => { await Bun.sleep(80); return success(2); } },
  ]);
  console.log("selected_ms", Math.round(performance.now() - start));
  return selected.kind === "one" ? selected.completion : success(0);
});
console.log("root_ms", Math.round(performance.now() - start), "kind", result.completion.kind);
