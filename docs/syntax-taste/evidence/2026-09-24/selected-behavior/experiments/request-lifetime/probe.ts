import {
  abandonRequest,
  isRequest,
  requestSnapshot,
  snapshotRequest,
  snapshotRequestLazy,
} from "../../../../../../../runtime/platform/http.ts";
import { strict as assert } from "node:assert";

async function probe(kind: "buffered" | "lazy") {
  const native = new Request("https://example.test/invoices/7", {
    method: "POST",
    headers: { "x-probe": kind },
    body: "draft",
  });
  const captured =
    kind === "buffered"
      ? await snapshotRequest(native, 4096)
      : await snapshotRequestLazy(native, 4096);
  if (captured.kind !== "request") throw new Error(`unexpected ${kind} rejection`);
  const token = captured.value;
  const before = {
    recognized: isRequest(token),
    method: requestSnapshot(token).method,
  };
  await abandonRequest(token);
  let afterSnapshot: string;
  try {
    afterSnapshot = requestSnapshot(token).method;
  } catch (cause) {
    afterSnapshot = cause instanceof Error ? cause.name : String(cause);
  }
  return {
    kind,
    before,
    after: { recognized: isRequest(token), snapshot: afterSnapshot },
  };
}

const cases = [await probe("buffered"), await probe("lazy")];
for (const result of cases) {
  assert.deepEqual(result.before, { recognized: true, method: "POST" });
  assert.deepEqual(result.after, { recognized: true, snapshot: "POST" });
}
console.log(JSON.stringify({ bun: Bun.version, cases }, null, 2));
