// E04: cancel-absent boundary and request-scope unit legs.
import { describe, expect, test } from "bun:test";
import {
  createCollectorSink,
  drainBudgetEscalations,
  effectiveBoundMs,
  raceBoundary,
} from "../transport/operation-budget.ts";
import { createRequestBudget, isUnknownWrite } from "../transport/request-budget.ts";
import {
  createRequestScope,
  currentRequestScope,
  runWithRequestScope,
} from "../transport/request-scope.ts";

describe("effectiveBoundMs", () => {
  test("min(own bound, remaining budget), floored at zero", () => {
    let at = 1000;
    const budget = createRequestBudget(100, () => at);
    expect(effectiveBoundMs(60, budget)).toBe(60);
    at += 60;
    expect(effectiveBoundMs(60, budget)).toBe(40);
    at += 40;
    expect(effectiveBoundMs(60, budget)).toBe(0);
  });

  test("absent bound is ambient-only; absent both is unbounded", () => {
    let at = 1000;
    const budget = createRequestBudget(100, () => at);
    expect(effectiveBoundMs(undefined, budget)).toBe(100);
    at += 30;
    expect(effectiveBoundMs(undefined, budget)).toBe(70);
    expect(effectiveBoundMs(undefined, undefined)).toBe(Number.POSITIVE_INFINITY);
    expect(effectiveBoundMs(25, undefined)).toBe(25);
  });

  test("bad bounds throw before any budget is consulted", () => {
    for (const bad of [0, -1, 1.5, Number.NaN, 2147483648]) {
      expect(() => effectiveBoundMs(bad, undefined)).toThrow(TypeError);
    }
  });
});

describe("raceBoundary", () => {
  test("settling work returns its value with no escalation", async () => {
    const collected = createCollectorSink();
    let started = false;
    const outcome = await raceBoundary(
      { source: "sql", boundMs: 5000, sink: collected.sink },
      async () => {
        started = true;
        await Bun.sleep(10);
        return "rows";
      },
    );
    expect(started).toBe(true);
    expect(outcome).toEqual({ kind: "settled", value: "rows" });
    expect(collected.escalations).toEqual([]);
    expect(collected.lates).toEqual([]);
  });

  test("timer expiry returns unknown-write while work stays owned until late settlement", async () => {
    const collected = createCollectorSink();
    let op!: Promise<string>;
    const outcome = await raceBoundary(
      { source: "sql", boundMs: 15, sink: collected.sink },
      () => (op = Bun.sleep(60).then(() => "late-rows")),
    );
    expect(outcome.kind).toBe("unknown");
    if (outcome.kind !== "unknown") throw new Error("unreachable");
    expect(isUnknownWrite(outcome.marker)).toBe(true);
    expect(outcome.marker).toMatchObject({ source: "sql", commit: "unknown" });
    expect(outcome.escalation).toMatchObject({
      cause: "budget",
      boundMs: 15,
      effectiveMs: 15,
    });
    expect(collected.escalations).toHaveLength(1);
    expect(collected.lates).toEqual([]);
    // The operation was not touched: it still settles, and the late
    // settlement is observed separately without rewriting the marker.
    expect(await op).toBe("late-rows");
    await Bun.sleep(5);
    expect(collected.lates).toHaveLength(1);
    expect(collected.lates[0]).toMatchObject({
      seq: outcome.escalation.seq,
      settled: "resolved",
    });
    expect(isUnknownWrite(outcome.marker)).toBe(true);
    expect(outcome.marker).toMatchObject({ commit: "unknown", owned: true });
  });

  test("exhausted budget returns without starting", async () => {
    const collected = createCollectorSink();
    let at = 1000;
    const budget = createRequestBudget(10, () => at);
    at += 50;
    let started = false;
    const outcome = await raceBoundary(
      { source: "fetch", boundMs: 5000, budget, sink: collected.sink },
      async () => {
        started = true;
        return "never";
      },
    );
    expect(started).toBe(false);
    expect(outcome.kind).toBe("unknown");
    if (outcome.kind !== "unknown") throw new Error("unreachable");
    expect(outcome.escalation).toMatchObject({ cause: "budget", effectiveMs: 0 });
    expect(collected.escalations).toHaveLength(1);
    // Nothing started, so no late settlement can arrive.
    await Bun.sleep(10);
    expect(collected.lates).toEqual([]);
  });

  test("caller cancel aborts the boundary, never the work", async () => {
    const collected = createCollectorSink();
    const stop = new AbortController();
    let op!: Promise<string>;
    const raced = raceBoundary(
      { source: "fetch", boundMs: 5000, signal: stop.signal, sink: collected.sink },
      () => (op = Bun.sleep(60).then(() => "late-body")),
    );
    await Bun.sleep(10);
    stop.abort(new Error("caller went away"));
    const outcome = await raced;
    expect(outcome.kind).toBe("unknown");
    if (outcome.kind !== "unknown") throw new Error("unreachable");
    expect(outcome.escalation.cause).toBe("cancel");
    expect(await op).toBe("late-body");
    await Bun.sleep(5);
    expect(collected.lates[0]).toMatchObject({ settled: "resolved" });
  });

  test("pre-aborted caller signal returns without starting", async () => {
    const collected = createCollectorSink();
    const stop = new AbortController();
    stop.abort();
    let started = false;
    const outcome = await raceBoundary(
      { source: "action", boundMs: 5000, signal: stop.signal, sink: collected.sink },
      async () => {
        started = true;
        return "never";
      },
    );
    expect(started).toBe(false);
    expect(outcome.kind).toBe("unknown");
    if (outcome.kind !== "unknown") throw new Error("unreachable");
    expect(outcome.escalation).toMatchObject({ cause: "cancel", effectiveMs: 0 });
  });

  test("scope expiry propagates its reason as the cause", async () => {
    for (const reason of ["disconnect", "shutdown"] as const) {
      const collected = createCollectorSink();
      const scope = createRequestScope({ sink: collected.sink });
      let op!: Promise<string>;
      const raced = raceBoundary(
        { source: "sql", boundMs: 5000, scopeSignal: scope.signal, sink: collected.sink },
        () => (op = Bun.sleep(60).then(() => "late-rows")),
      );
      await Bun.sleep(10);
      scope.expire(reason);
      const outcome = await raced;
      expect(outcome.kind).toBe("unknown");
      if (outcome.kind !== "unknown") throw new Error("unreachable");
      expect(outcome.escalation.cause).toBe(reason);
      expect(await op).toBe("late-rows");
    }
  });

  test("pre-expired scope returns without starting", async () => {
    const collected = createCollectorSink();
    const scope = createRequestScope({ sink: collected.sink });
    scope.expire("disconnect");
    let started = false;
    const outcome = await raceBoundary(
      { source: "sql", scopeSignal: scope.signal, sink: collected.sink },
      async () => {
        started = true;
        return "never";
      },
    );
    expect(started).toBe(false);
    expect(outcome.kind).toBe("unknown");
    if (outcome.kind !== "unknown") throw new Error("unreachable");
    expect(outcome.escalation.cause).toBe("disconnect");
  });

  test("an unrecognized scope reason still expires as a budget outcome", async () => {
    const collected = createCollectorSink();
    const foreign = new AbortController();
    let started = false;
    const raced = raceBoundary(
      { source: "sql", boundMs: 5000, scopeSignal: foreign.signal, sink: collected.sink },
      async () => {
        started = true;
        await Bun.sleep(60);
        return "late";
      },
    );
    await Bun.sleep(10);
    foreign.abort({ unexpected: true });
    const outcome = await raced;
    expect(started).toBe(true);
    expect(outcome.kind).toBe("unknown");
    if (outcome.kind !== "unknown") throw new Error("unreachable");
    expect(outcome.escalation.cause).toBe("budget");
  });

  test("sql structurally rejects caller cancel while accepting scope expiry", async () => {
    const collected = createCollectorSink();
    const stop = new AbortController();
    await expect(
      raceBoundary(
        { source: "sql", boundMs: 100, signal: stop.signal, sink: collected.sink },
        async () => "never",
      ),
    ).rejects.toThrow(TypeError);
    expect(collected.escalations).toEqual([]);
  });

  test("operation rejection propagates instead of becoming unknown-write", async () => {
    const collected = createCollectorSink();
    const failure = new Error("native blew up");
    await expect(
      raceBoundary({ source: "fetch", boundMs: 5000, sink: collected.sink }, async () => {
        await Bun.sleep(5);
        throw failure;
      }),
    ).rejects.toBe(failure);
    expect(collected.escalations).toEqual([]);
    expect(collected.lates).toEqual([]);
  });

  test("late rejection after boundary return is observed, not unhandled", async () => {
    const collected = createCollectorSink();
    let op!: Promise<string>;
    const outcome = await raceBoundary(
      { source: "fetch", boundMs: 15, sink: collected.sink },
      () =>
        (op = Bun.sleep(40).then(() => {
          throw new Error("late native fault");
        })),
    );
    expect(outcome.kind).toBe("unknown");
    if (outcome.kind !== "unknown") throw new Error("unreachable");
    await expect(op).rejects.toThrow("late native fault");
    await Bun.sleep(5);
    expect(collected.lates).toHaveLength(1);
    expect(collected.lates[0]).toMatchObject({
      seq: outcome.escalation.seq,
      settled: "rejected",
    });
  });

  test("unbounded races settle without timers", async () => {
    const collected = createCollectorSink();
    const outcome = await raceBoundary({ source: "sql", sink: collected.sink }, async () => {
      await Bun.sleep(10);
      return "rows";
    });
    expect(outcome).toEqual({ kind: "settled", value: "rows" });
    expect(collected.escalations).toEqual([]);
  });

  test("the module ring is the fallback sink and counts drops", async () => {
    drainBudgetEscalations();
    try {
      const outcome = await raceBoundary({ source: "worker", boundMs: 10 }, () =>
        Bun.sleep(50).then(() => "late"),
      );
      expect(outcome.kind).toBe("unknown");
      const drained = drainBudgetEscalations();
      expect(drained.escalations).toHaveLength(1);
      expect(drained.dropped).toBe(0);
      expect(drained.escalations[0]?.marker).toMatchObject({ source: "worker" });
    } finally {
      drainBudgetEscalations();
    }
  });

  test("the ring drops oldest first and reports every drop", async () => {
    drainBudgetEscalations();
    try {
      // File through the race without starting: exhausted budgets keep
      // this fast with no timers.
      let at = 1000;
      const budget = createRequestBudget(10, () => at);
      at += 50;
      const filings: Promise<unknown>[] = [];
      for (let i = 0; i < 300; i++) {
        filings.push(
          raceBoundary({ source: "worker", boundMs: 5000, budget }, async () => "never"),
        );
      }
      await Promise.all(filings);
      const drained = drainBudgetEscalations();
      expect(drained.escalations).toHaveLength(256);
      expect(drained.dropped).toBe(44);
      expect(drained.lates).toEqual([]);
    } finally {
      drainBudgetEscalations();
    }
  });
});

describe("request-scope", () => {
  test("the ambient scope resolves implicitly and nests", () => {
    expect(currentRequestScope()).toBe(undefined);
    const outer = createRequestScope();
    const inner = createRequestScope();
    runWithRequestScope(outer, () => {
      expect(currentRequestScope()).toBe(outer);
      runWithRequestScope(inner, () => {
        expect(currentRequestScope()).toBe(inner);
      });
      expect(currentRequestScope()).toBe(outer);
    });
    expect(currentRequestScope()).toBe(undefined);
  });

  test("the ambient scope survives suspension", async () => {
    const scope = createRequestScope();
    await runWithRequestScope(scope, async () => {
      await Bun.sleep(10);
      expect(currentRequestScope()).toBe(scope);
    });
  });

  test("expire is idempotent and the first reason wins", () => {
    const scope = createRequestScope();
    expect(scope.expired()).toBe(false);
    expect(scope.signal.aborted).toBe(false);
    scope.expire("disconnect");
    expect(scope.expired()).toBe(true);
    expect(scope.signal.aborted).toBe(true);
    expect(scope.signal.reason).toBe("disconnect");
    scope.expire("shutdown");
    expect(scope.signal.reason).toBe("disconnect");
  });

  test("unknown abort reasons are rejected", () => {
    const scope = createRequestScope();
    expect(() => scope.expire("timeout" as "disconnect")).toThrow(TypeError);
    expect(scope.expired()).toBe(false);
  });

  test("totalMs installs a shared budget; absent total is unbounded-until-expired", () => {
    expect(createRequestScope().budget).toBe(undefined);
    const scope = createRequestScope({ totalMs: 100 });
    expect(scope.budget?.totalMilliseconds).toBe(100);
    expect(() => createRequestScope({ totalMs: 0 })).toThrow(TypeError);
  });

  test("an explicit collector sink observes scope races", async () => {
    const collected = createCollectorSink();
    const scope = createRequestScope({ sink: collected.sink });
    const outcome = await runWithRequestScope(scope, () =>
      raceBoundary(
        {
          source: "sql",
          boundMs: 10,
          budget: currentRequestScope()?.budget,
          scopeSignal: currentRequestScope()?.signal,
          sink: currentRequestScope()?.sink,
        },
        () => Bun.sleep(50).then(() => "late"),
      ),
    );
    expect(outcome.kind).toBe("unknown");
    expect(collected.escalations).toHaveLength(1);
  });
});
