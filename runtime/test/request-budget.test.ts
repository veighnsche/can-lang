import { test, expect } from "bun:test";
import { success, value } from "../completion.ts";
import {
  closeResource,
  launchOwned,
  registerResource,
  resourceStatus,
  runOwnedRoot,
  useResource,
} from "../owner.ts";
import {
  createRequestBudget,
  isUnknownWrite,
  unknownWrite,
  type UnknownWriteSource,
} from "../transport/request-budget.ts";

function fakeClock(start = 1000): { now: () => number; advance: (ms: number) => void } {
  let at = start;
  return {
    now: () => at,
    advance: (ms: number) => {
      at += ms;
    },
  };
}

test("each operation's effective deadline is min(its bound, remaining budget)", () => {
  const clock = fakeClock();
  const budget = createRequestBudget(100, clock.now);
  expect(budget.totalMilliseconds).toBe(100);
  expect(budget.remainingMilliseconds()).toBe(100);
  expect(budget.expired()).toBe(false);
  // A bound below the remaining budget passes through untouched.
  expect(budget.effectiveMilliseconds(60)).toBe(60);
  clock.advance(60);
  expect(budget.remainingMilliseconds()).toBe(40);
  // Serial per-operation budgets are rejected: the second 60ms bound sees
  // only the shared 40ms that remain, not a fresh 60ms.
  expect(budget.effectiveMilliseconds(60)).toBe(40);
  expect(budget.effectiveMilliseconds(25)).toBe(25);
  clock.advance(40);
  expect(budget.expired()).toBe(true);
  expect(budget.remainingMilliseconds()).toBe(0);
  // Exhaustion reports zero — "return unknown-write without starting" —
  // and never reports negative time however far past expiry the clock runs.
  expect(budget.effectiveMilliseconds(60)).toBe(0);
  clock.advance(1000);
  expect(budget.remainingMilliseconds()).toBe(0);
  expect(budget.expired()).toBe(true);
});

test("budgets and operation bounds reject non-positive and unrepresentable input", () => {
  for (const bad of [0, -1, 1.5, Number.NaN, Number.POSITIVE_INFINITY, 2147483648]) {
    expect(() => createRequestBudget(bad, fakeClock().now)).toThrow(TypeError);
  }
  const budget = createRequestBudget(100, fakeClock().now);
  for (const bad of [0, -1, 2.5, Number.NaN, Number.NEGATIVE_INFINITY, 2147483648]) {
    expect(() => budget.effectiveMilliseconds(bad)).toThrow(TypeError);
  }
  expect(createRequestBudget(1, fakeClock().now).remainingMilliseconds()).toBe(1);
  expect(createRequestBudget(2147483647, fakeClock().now).remainingMilliseconds()).toBe(2147483647);
});

test("the default budget runs on the monotonic clock", async () => {
  const budget = createRequestBudget(50);
  expect(budget.remainingMilliseconds()).toBeGreaterThan(0);
  expect(budget.expired()).toBe(false);
  await Bun.sleep(60);
  expect(budget.expired()).toBe(true);
  expect(budget.effectiveMilliseconds(50)).toBe(0);
});

test("unknown-write is one frozen vocabulary across SQL, S3, actions, workers and fetch", () => {
  const seen: UnknownWriteSource[] = ["sql", "s3", "action", "worker", "fetch"];
  for (const source of seen) {
    const outcome = unknownWrite(source);
    expect(Object.isFrozen(outcome)).toBe(true);
    expect({ ...outcome }).toEqual({
      kind: "unknown-write",
      source,
      commit: "unknown",
      owned: true,
      escalation: "supervisor",
    });
    expect(isUnknownWrite(outcome)).toBe(true);
  }
  expect(() => unknownWrite("pg" as UnknownWriteSource)).toThrow(TypeError);
  expect(() => unknownWrite("" as UnknownWriteSource)).toThrow(TypeError);
  // Lookalikes that resolve the commit, drop ownership, or skip escalation
  // are not the vocabulary: uncertainty is preserved structurally.
  for (const fake of [
    null,
    undefined,
    42,
    "unknown-write",
    { kind: "unknown-write", source: "sql", commit: "unknown", owned: true },
    {
      kind: "unknown-write",
      source: "sql",
      commit: "committed",
      owned: true,
      escalation: "supervisor",
    },
    {
      kind: "unknown-write",
      source: "sql",
      commit: "aborted",
      owned: true,
      escalation: "supervisor",
    },
    {
      kind: "unknown-write",
      source: "sql",
      commit: "unknown",
      owned: false,
      escalation: "supervisor",
    },
    { kind: "unknown-write", source: "sql", commit: "unknown", owned: true, escalation: "none" },
    {
      kind: "unknown-write",
      source: "pg",
      commit: "unknown",
      owned: true,
      escalation: "supervisor",
    },
  ]) {
    expect(isUnknownWrite(fake)).toBe(false);
  }
});

test("a visible timeout leaves the operation owned and the lease unrevoked", async () => {
  // Observations stay in `seen`: invoke() would swallow an assertion throw
  // inside the root body and surface it only as a failed completion.
  const seen: Record<string, unknown> = {};
  const owned = await runOwnedRoot(async () => {
    const budget = createRequestBudget(20);
    let closed = false;
    const token = registerResource("probe", Object.freeze({ tag: "probe" }), async () => {
      closed = true;
      return success(undefined);
    });
    // The child holds its lease past budget expiry: the visible boundary
    // returns while the native work is still running.
    const group = launchOwned([
      {
        captures: [],
        run: async () => {
          const held = await useResource(token, "probe", async (native) => {
            await Bun.sleep(80);
            return success((native as { tag: string }).tag);
          });
          return success(value(held));
        },
      },
    ]);
    group.publish([0]);
    await Bun.sleep(40);
    seen.expired = budget.expired();
    seen.atExpiry = { ...resourceStatus(token), id: 0 };
    const again = value(await useResource(token, "probe", (native) => success(native)));
    seen.reused = (again as { tag: string }).tag;
    // Drain-by-default: closing waits for the still-owned work instead of
    // revoking it, even though the budget already expired.
    let settled = false;
    const closing = closeResource(token, "probe").then((completion) => {
      settled = true;
      return completion;
    });
    await Bun.sleep(10);
    seen.settledEarly = settled;
    seen.closedEarly = closed;
    const [child] = await Promise.all([group.promises[0], closing]);
    seen.child = value(child!);
    seen.settled = settled;
    seen.closed = closed;
    seen.final = { ...resourceStatus(token), id: 0 };
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
  expect(seen.expired).toBe(true);
  // No lease was revoked by the timer: the child's lease is still held and
  // a fresh use of the same resource still works after expiry.
  expect(seen.atExpiry).toMatchObject({ state: "open", leases: 1 });
  expect(seen.reused).toBe("probe");
  expect(seen.settledEarly).toBe(false);
  expect(seen.closedEarly).toBe(false);
  expect(seen.child).toBe("probe");
  expect(seen.settled).toBe(true);
  expect(seen.closed).toBe(true);
  expect(seen.final).toMatchObject({ state: "closed", leases: 0 });
});

test("the boundary reports unknown-write while the late settlement stays observable", async () => {
  const budget = createRequestBudget(15);
  let late: string | undefined;
  const operation = Bun.sleep(50).then(() => {
    late = "late-value";
    return late;
  });
  // The cancel-absent boundary: at budget expiry the visible layer
  // returns the honest unknown outcome without touching the operation.
  const boundary = await Promise.race([
    operation.then((settled) => ({ kind: "settled" as const, settled })),
    Bun.sleep(budget.effectiveMilliseconds(5000)).then(() => unknownWrite("sql")),
  ]);
  expect(isUnknownWrite(boundary)).toBe(true);
  expect(boundary).toMatchObject({ kind: "unknown-write", source: "sql", commit: "unknown" });
  expect(late).toBe(undefined);
  expect(await operation).toBe("late-value");
  // The late settlement never rewrites the returned marker: the boundary
  // outcome still reports commit unknown after the value lands.
  expect(isUnknownWrite(boundary)).toBe(true);
  expect(boundary).toMatchObject({ commit: "unknown", owned: true, escalation: "supervisor" });
});
