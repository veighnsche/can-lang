// Browser-profile staged driver. This module runs inside a staged copy of
// runtime/ where the seven host-coupled modules are overlaid with their
// browser alternates, so every import below resolves to profile code. It
// must import no Node builtin and no assertion module. The outer test
// supplies JSON fixtures (error plan, diagnostic table) and reports failures.
import { array, dataArray, dataKeys, dataProperty, record, recordIdentity } from "../data.ts";
import {
  createDomainRuntime,
  domainFailureDiagnostics,
  isDomainFailure,
  verifyErrorPlan,
  type ErrorPlan,
} from "../browser/domain.ts";
import {
  configureBrowserDiagnostics,
  diagnosticFrames,
  reportBrowserDiagnostic,
} from "../browser/diagnostics.ts";
import {
  closeResourceWithContext,
  guardCallbackWithContext,
  launchOwned,
  launchOwnedWithContext,
  registerResourceWithContext,
  runExplicitRoot,
  useResourceWithContext,
  withScopeWithContext,
  type OwnerContext,
} from "../browser/owner.ts";
import {
  callableEqual,
  callableInstance,
  callableReceipt,
  ownCallable,
} from "../browser/callable.ts";
import { aggregate, handle, settleWithContext } from "../browser/coordination.ts";
import { runBrowserEntry } from "../browser/entry.ts";
import { isHostNativeError, isHostProxy } from "../browser/reflect.ts";
import {
  captureStandard,
  describeNativeFailure,
  isStandardFailure,
  standardFailureKind,
  standardFailureMessage,
} from "../failure.ts";
import {
  caught,
  checkedCompletion,
  failure,
  isCompletion,
  success,
  value,
  type Completion,
} from "../completion.ts";
import { intDivide } from "../primitive.ts";
import { createNumbers } from "../number.ts";
import { decodeJSON, encodeJSON } from "../codec/json.ts";
import { copyBytes, isBytes, ownBytes } from "../bytes.ts";
import { createText } from "../text.ts";

export type StagedFixtures = {
  plan: ErrorPlan;
  table: { index: unknown; maps: Record<string, unknown> };
};

export async function runStagedProfile(fixtures: StagedFixtures): Promise<string[]> {
  const passed: string[] = [];
  const check = async (name: string, fn: () => void | Promise<void>): Promise<void> => {
    await fn();
    passed.push(name);
  };
  const assert = (cond: unknown, name: string): void => {
    if (!cond) throw new Error(`staged assertion failed: ${name}`);
  };
  const origin = {
    source: "can.project.root/app/main.can",
    start: 10,
    end: 20,
    invocation: [] as string[],
  };
  const shapeBy = (kind: string, declaration: string): string => {
    const found = fixtures.plan.shapes.find(
      (s) => s.kind === kind && s.declaration === declaration,
    );
    if (!found) throw new Error(`staged fixture lacks ${kind} ${declaration}`);
    return found.identity;
  };

  // Reflect predicates: genuine values agree, hostile proxies are contained.
  await check("reflect", () => {
    assert(isHostProxy({}) === false, "plain object is not a proxy");
    assert(isHostProxy([]) === false, "array is not a proxy");
    assert(isHostProxy(null) === false, "null is not a proxy");
    assert(isHostProxy(1n) === false, "bigint is not a proxy");
    let traps = 0;
    const trap = (): never => {
      traps++;
      throw new Error("trap");
    };
    const throwing = new Proxy({}, { getPrototypeOf: trap });
    assert(isHostProxy(throwing) === true, "throwing proxy detected");
    assert(traps === 1, "proxy check fires one bounded trap");
    const revoked = Proxy.revocable({}, {});
    revoked.revoke();
    assert(isHostProxy(revoked.proxy) === true, "revoked proxy detected");
    assert(isHostNativeError(new TypeError("x")) === true, "native error detected");
    class Custom extends RangeError {}
    assert(isHostNativeError(new Custom("x")) === true, "subclass detected");
    assert(isHostNativeError({}) === false, "plain object is not an error");
    assert(isHostNativeError(throwing) === false, "throwing proxy is not an error");
    // A cyclic prototype proxy terminates instead of hanging instanceof.
    const cyclic: unknown = new Proxy({}, { getPrototypeOf: () => cyclic as object });
    assert(isHostNativeError(cyclic) === false, "cyclic proxy terminates");
  });

  // Data guards fail closed with the canonical identities.
  await check("data", () => {
    const rec = record("p::r", [
      ["a", 1n],
      ["b", "x"],
    ]);
    assert(recordIdentity(rec) === "p::r", "record identity");
    assert(recordIdentity({}) === undefined, "foreign object unbranded");
    assert(recordIdentity(new Proxy(rec, {})) === undefined, "proxy unbranded");
    assert(dataProperty(rec, "a") === 1n, "property read");
    assert(Object.isFrozen(rec) && Object.isFrozen(array([rec])), "values frozen");
    let traps = 0;
    const trap = (): never => {
      traps++;
      throw new Error("trap");
    };
    const throwing = new Proxy({}, { ownKeys: trap, getOwnPropertyDescriptor: trap });
    for (const fn of [
      () => dataKeys(throwing),
      () => dataProperty(throwing, "a"),
      () => dataArray(throwing),
    ]) {
      try {
        fn();
        throw new Error("proxy admitted");
      } catch (cause) {
        assert(
          cause instanceof TypeError && cause.message === "expected non-proxy data object",
          "proxy fails closed with canonical identity",
        );
      }
    }
    assert(traps <= 3, "proxy guards fire bounded traps");
    const revoked = Proxy.revocable([], {});
    revoked.revoke();
    try {
      dataArray(revoked.proxy);
      throw new Error("revoked admitted");
    } catch (cause) {
      assert(cause instanceof TypeError, "revoked fails closed");
    }
    try {
      // oxlint-disable-next-line no-sparse-arrays -- the sparse hole is the rejection vector under test.
      dataArray([1, , 3] as unknown[]);
      throw new Error("sparse admitted");
    } catch (cause) {
      assert(cause instanceof TypeError, "sparse array rejected");
    }
  });

  // Failure sanitization never reads getters or leaks causes.
  await check("failure", () => {
    let invoked = 0;
    const trap = (): never => {
      invoked++;
      throw new Error("secret");
    };
    const hostile = {
      get message(): string {
        return trap();
      },
      get name(): string {
        return trap();
      },
    };
    assert(describeNativeFailure(hostile) === "native object", "hostile getters unread");
    assert(describeNativeFailure(new RangeError("range")) === "RangeError: range", "native error");
    class Custom extends TypeError {}
    assert(describeNativeFailure(new Custom("custom")) === "TypeError: custom", "subclass name");
    const throwing = new Proxy(
      {},
      { getPrototypeOf: trap, ownKeys: trap, getOwnPropertyDescriptor: trap },
    );
    assert(describeNativeFailure(throwing) === "native proxy", "throwing proxy contained");
    const revoked = Proxy.revocable(new Error("secret"), {});
    revoked.revoke();
    assert(describeNativeFailure(revoked.proxy) === "native proxy", "revoked contained");
    assert(
      !JSON.stringify(captureStandard({ credential: "s3cret" }, origin)).includes("s3cret"),
      "cause not serialized",
    );
    assert(invoked <= 2, "sanitizer fires bounded traps");
    const occurrence = captureStandard(new Error("safe"), origin);
    assert(isStandardFailure(occurrence), "branded occurrence");
    assert(!isStandardFailure(new Proxy(occurrence, {})), "proxy not branded");
    assert(standardFailureKind(occurrence) === "native_exception", "kind preserved");
    assert(Reflect.ownKeys(occurrence).length === 0, "occurrence carries no fields");
  });

  // Completions stay immutable and boxed.
  await check("completion", () => {
    const ok = success(1n);
    assert(isCompletion(ok) && ok.kind === "ok", "success boxed");
    assert(Object.isFrozen(ok), "completion frozen");
    assert(value(ok) === 1n, "value extracts");
    const failed = caught(new Error("x"), origin);
    assert(failed.kind === "standard", "caught boxes standard");
    try {
      value(failed as Completion<bigint>);
      throw new Error("failure value extracted");
    } catch (cause) {
      assert(isStandardFailure(cause), "thrown occurrence stays branded");
    }
    for (const forged of [{ kind: "ok", value: 1 }, new Proxy(success(1), {})]) {
      assert(!isCompletion(forged), "forged completion rejected");
      try {
        checkedCompletion(forged as Completion);
        throw new Error("forged accepted");
      } catch {
        /* expected */
      }
    }
  });

  // Sealed domain: verified plans behave, forged plans never construct.
  await check("domain", async () => {
    const runtime = createDomainRuntime(await verifyErrorPlan(fixtures.plan));
    const intId = shapeBy("primitive", "int");
    const errId = shapeBy("error", "can.project.root/app::fault");
    const payload = record(errId, [["value", 7n]]);
    assert(runtime.accepts(errId, payload) === true, "valid payload accepted");
    assert(runtime.accepts(errId, record(errId, [["value", "7"]])) === false, "mistyped rejected");
    assert(runtime.accepts(errId, { value: 7n }) === false, "unbranded rejected");
    assert(runtime.accepts(errId, new Proxy(payload, {})) === false, "proxy rejected");
    const first = runtime.create(errId, payload, origin);
    assert(isDomainFailure(first), "domain branded");
    assert(domainFailureDiagnostics(first).payload === payload, "payload exact");
    assert(runtime.checkBound(first, [errId]) === first, "bound checked");
    try {
      runtime.checkBound(first, [intId]);
      throw new Error("escaping bound accepted");
    } catch (cause) {
      assert(cause instanceof TypeError, "escaping bound rejected");
    }
    const tampered = JSON.parse(JSON.stringify(fixtures.plan)) as ErrorPlan;
    (tampered.shapes[0] as { identity: string }).identity = "0".repeat(64);
    try {
      await verifyErrorPlan(tampered);
      throw new Error("forged plan verified");
    } catch (cause) {
      assert(cause instanceof TypeError, "forged plan rejected");
    }
    try {
      createDomainRuntime(JSON.parse(JSON.stringify(fixtures.plan)) as ErrorPlan);
      throw new Error("unverified plan constructed");
    } catch (cause) {
      assert(
        cause instanceof TypeError && (cause as TypeError).message === "unverified error plan",
        "unverified construction rejected",
      );
    }
  });

  // Sealed diagnostics: table locations resolve, faults report once.
  await check("diagnostics", () => {
    configureBrowserDiagnostics(fixtures.table);
    const frames = diagnosticFrames("private primitive cause", origin);
    assert(frames.length === 1, "checked origin frame present");
    assert(
      frames[0].file === "app/main.can" && frames[0].line === 2 && frames[0].column === 5,
      "sealed source location exact",
    );
    assert(frames[0].synthetic === true, "origin frame marked synthetic");
    const errors: unknown[][] = [];
    const nativeError = console.error;
    console.error = (...args: unknown[]) => {
      errors.push(args);
    };
    try {
      const fault = caught(new Error("bearer-secret"), origin);
      const first = reportBrowserDiagnostic(fault, "startup");
      assert(first !== undefined, "fault reports");
      assert(first!.kind === "can.runtime-diagnostic", "record kind");
      assert(first!.phase === "startup", "record phase");
      assert(first!.category === "native_exception", "record category");
      assert(
        first!.file === "app/main.can" && first!.line === 2 && first!.column === 5,
        "record located",
      );
      assert(Object.isFrozen(first), "record frozen");
      assert(!JSON.stringify(first).includes("bearer-secret"), "record sanitized");
      assert(reportBrowserDiagnostic(fault, "startup") === undefined, "occurrence deduped");
      assert(
        reportBrowserDiagnostic(success(undefined), "startup") === undefined,
        "success silent",
      );
      assert(errors.length === 1 && errors[0][0] === first, "default sink fires once");
    } finally {
      console.error = nativeError;
    }
    const unlocated = diagnosticFrames("x", { source: "nope", start: 0, end: 0, invocation: [] });
    assert(unlocated.length === 0, "unknown source yields no frames");
    const hinted = new Error("hint");
    Object.defineProperty(hinted, "stack", {
      value: "Error: hint\n    at work (app.js:1:5)",
      configurable: true,
    });
    const hintFrames = diagnosticFrames(hinted, {
      source: "nope",
      start: 0,
      end: 0,
      invocation: [],
    });
    assert(hintFrames.length === 1, "sealed map hint resolves");
    assert(
      hintFrames[0].file === "app/main.can" &&
        hintFrames[0].line === 2 &&
        hintFrames[0].column === 5 &&
        hintFrames[0].synthetic === false,
      "hint frame exact and non-synthetic",
    );
  });

  // Explicit ownership: leases isolate across interleaved awaits.
  await check("owner", async () => {
    try {
      launchOwned([]);
      throw new Error("ambient launch admitted");
    } catch (cause) {
      assert(
        cause instanceof TypeError &&
          cause.message === "ambient ownership is unavailable in the browser profile",
        "ambient launch fails closed",
      );
    }
    const events: string[] = [];
    const work = async (ctx: OwnerContext, name: string) => {
      const token = registerResourceWithContext(ctx, "slot", {}, () => success(undefined));
      await useResourceWithContext(ctx, token, "slot", async () => {
        events.push(`${name}:acquired`);
        await new Promise((resolve) => setTimeout(resolve, 5));
        events.push(`${name}:held`);
        return success(undefined);
      });
      const closed = await closeResourceWithContext(ctx, token, "slot");
      assert(closed.kind === "ok", "work closes its resource");
    };
    const first = runExplicitRoot(async (ctx) => {
      await work(ctx, "a");
      return success(undefined);
    });
    const second = runExplicitRoot(async (ctx) => {
      await work(ctx, "b");
      return success(undefined);
    });
    const [a, b] = await Promise.all([first, second]);
    assert(a.completion.kind === "ok" && b.completion.kind === "ok", "roots settle");
    assert(events.filter((e) => e.startsWith("a:")).length === 2, "leases isolated");
    // A throwing proxy in captures cannot acquire or hang the launch.
    await runExplicitRoot(async (ctx) => {
      const proxy = new Proxy(
        {},
        {
          ownKeys: () => {
            throw new Error("trap");
          },
        },
      );
      const group = launchOwnedWithContext(ctx, [
        { captures: [proxy], run: () => success(undefined) },
      ]);
      group.publish([0]);
      const [settled] = await Promise.all(group.promises);
      assert(settled.kind === "ok", "proxy capture skipped safely");
      return success(undefined);
    });
    // Late callbacks fault instead of joining a closed scope.
    await runExplicitRoot(async (ctx) => {
      let guarded!: (value: unknown) => Promise<Completion<unknown>>;
      const scoped = await withScopeWithContext(ctx, (_child, scope) => {
        guarded = guardCallbackWithContext(scope, async () => success(undefined));
        return success(undefined);
      });
      assert(scoped.kind === "ok", "scope settles");
      try {
        await guarded(undefined);
        throw new Error("late callback admitted");
      } catch (cause) {
        assert(isStandardFailure(cause), "late callback faults");
      }
      return success(undefined);
    });
  });

  // Assert-free callables keep target/capture equality.
  await check("callable", () => {
    const target = (...args: unknown[]) => success(args.length);
    const one = ownCallable("site#0", "target", [1n, "x"], target);
    const two = ownCallable("site#1", "target", [1n, "x"], (...args: unknown[]) =>
      success(args.length),
    );
    const other = ownCallable("site#2", "other", [1n, "x"], (...args: unknown[]) =>
      success(args.length),
    );
    assert(callableInstance(one) !== undefined, "receipt issued");
    assert(callableInstance(() => 0) === undefined, "foreign function unbranded");
    assert(callableReceipt(one)?.target === "target", "receipt target");
    assert(callableEqual(one, two, Object.is) === true, "same target and captures equal");
    assert(callableEqual(one, other, Object.is) === false, "different target unequal");
    assert(callableEqual(one, () => 0, Object.is) === false, "foreign unequal");
    try {
      ownCallable("", "target", [], () => 0);
      throw new Error("empty site admitted");
    } catch (cause) {
      assert(cause instanceof TypeError, "empty site rejected");
    }
  });

  // Coordination selection and dispatch without assertion context.
  await check("coordination", async () => {
    await runExplicitRoot(async (ctx) => {
      const run = (v: unknown) => ({ captures: [], run: () => success(v) });
      const all = await settleWithContext(ctx, "all", [run(1n), run(2n)]);
      assert(all.kind === "all" && all.outcomes.length === 2, "all settles");
      const collected = await handle(
        all,
        {
          each: (_i, c) => c,
          shared: (_i, c) => c,
          allFailed: (outcomes) => success(outcomes),
        },
        true,
        origin,
      );
      assert(collected.kind === "ok", "all dispatches");
      const race = await settleWithContext(ctx, "race", [run("slow"), run("fast")]);
      assert(race.kind === "one", "race selects one");
      const runtime = createDomainRuntime(await verifyErrorPlan(fixtures.plan));
      const faultId = shapeBy("error", "can.project.root/app::fault");
      const offlineId = shapeBy("error", "can.project.root/app::offline");
      const combinedId = shapeBy("error", "can.prelude@1::all_failed");
      const snapshot = captureStandard({ private: "native cause" }, origin);
      const outcomes = [
        failure(snapshot),
        failure(runtime.create(faultId, record(faultId, [["value", 9007199254740993n]]), origin)),
        failure(runtime.create(offlineId, record(offlineId, [["message", "down"]]), origin)),
      ];
      const agg = aggregate(outcomes, runtime, combinedId, origin);
      assert(agg.kind === "domain", "aggregate builds");
      if (agg.kind === "domain") {
        const values = (domainFailureDiagnostics(agg.value).payload as { failures: unknown[] })
          .failures;
        assert(values.length === 3, "aggregate keeps members");
        assert(values[0] === snapshot, "standard member keeps identity");
      }
      return success(undefined);
    });
  });

  // Entry runs main once after readiness and reports startup faults.
  await check("entry", async () => {
    let observed = 0;
    const listeners = new Map<string, () => void>();
    const host = {
      readyState: "loading",
      addEventListener: (type: string, listener: () => void) => {
        listeners.set(type, listener);
      },
    };
    const errors: unknown[][] = [];
    const nativeError = console.error;
    console.error = (...args: unknown[]) => {
      errors.push(args);
    };
    try {
      const pending = runBrowserEntry(
        {
          table: fixtures.table,
          main: async (ctx) => {
            observed++;
            const token = registerResourceWithContext(ctx, "boot", {}, () => success(undefined));
            return closeResourceWithContext(ctx, token, "boot");
          },
        },
        host,
      );
      await new Promise((resolve) => setTimeout(resolve, 5));
      assert(observed === 0, "main waits for readiness");
      listeners.get("DOMContentLoaded")!();
      await pending;
      assert(observed === 1, "main runs exactly once");
      const reports: unknown[] = [];
      const outcome = await runBrowserEntry(
        {
          table: fixtures.table,
          main: () => {
            throw new Error("boot-secret");
          },
          report: (record) => {
            reports.push(record);
          },
        },
        { readyState: "complete", addEventListener: () => {} },
      ).then(
        () => ({ rejected: false, record: undefined as unknown }),
        (record: unknown) => ({ rejected: true, record }),
      );
      assert(outcome.rejected, "startup fault rejects");
      const rec = outcome.record as { phase: string; category: string };
      assert(
        rec.phase === "startup" && rec.category === "native_exception",
        "startup record sealed",
      );
      assert(!JSON.stringify(rec).includes("boot-secret"), "startup record sanitized");
      assert(reports.length === 1 && reports[0] === rec, "injected reporter fires once");
    } finally {
      console.error = nativeError;
    }
  });

  // Preserved semantics: int64, bytes, codecs and text through the profile.
  await check("semantics", async () => {
    try {
      intDivide(1n, 0n);
      throw new Error("division by zero admitted");
    } catch (cause) {
      const boxed = caught(cause, origin);
      assert(boxed.kind === "standard", "int fault standard");
      if (boxed.kind === "standard")
        assert(
          standardFailureMessage(boxed.value) === "arithmetic: integer division by zero",
          "int fault message exact",
        );
    }
    const bytes = ownBytes(new Uint8Array([104, 105]));
    assert(isBytes(bytes), "bytes branded");
    assert(copyBytes(bytes, origin).join(",") === "104,105", "bytes exact");
    const runtime = createDomainRuntime(await verifyErrorPlan(fixtures.plan));
    const faultId = shapeBy("error", "can.project.root/app::fault");
    const numbers = createNumbers(runtime, {
      inexact: faultId,
      invalidNumber: faultId,
      invalidTextBool: faultId,
      invalidIntBool: faultId,
    });
    const converted = await numbers.fromInt(9007199254740993n);
    assert(converted.kind === "ok" && value(converted) === "9007199254740993", "int64 exact");
    const text = createText(runtime, {
      emptySeparator: faultId,
      emptyPattern: faultId,
      invalidUnicode: faultId,
      invalidRegex: faultId,
      invalidLimit: faultId,
      match: faultId,
    });
    const compiled = await text.compileRegex("(?:)", "u");
    assert(compiled.kind === "ok", "regex compiles");
    if (compiled.kind === "ok") {
      const found = await text.findMatches(value(compiled), "😀x", 5n);
      assert(found.kind === "ok", "matches run");
      if (found.kind === "ok") {
        const starts = (value(found) as { start: bigint }[]).map((m) => m.start);
        assert(starts.join(",") === "0,2,3", "empty unicode advance exact");
      }
    }
    const schema = {
      root: "r",
      nodes: [
        { identity: "int", kind: "primitive", name: "int" },
        {
          identity: "r",
          kind: "record",
          name: "p::r",
          fields: [{ name: "v", type: "int" }],
        },
      ],
    } as never;
    const encoded = encodeJSON(schema, record("r", [["v", 42n]]));
    assert(isBytes(encoded), "codec encodes");
    const decoded = decodeJSON(schema, encoded) as { v: bigint };
    assert(decoded.v === 42n, "codec round-trips int64");
  });

  return passed;
}
