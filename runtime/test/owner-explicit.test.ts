import { test, expect } from "bun:test";
import {
  runOwnedRoot,
  registerResource,
  closeResource,
  runExplicitRoot,
  launchOwnedWithContext,
  launchNativeWithContext,
  registerResourceWithContext,
  useResourceWithContext,
  closeResourceWithContext,
  resourceStatus,
  withScopeWithContext,
  guardCallbackWithContext,
  registerCallableCaptures,
  type OwnerContext,
  type Resource,
  type OwnerDiagnostic,
} from "../owner.ts";
import { settleWithContext } from "../coordination.ts";
import { success, failure, invoke, value, type Completion } from "../completion.ts";
import { resourceStateFailure, standardFailureKind } from "../failure.ts";
const origin = { source: "test:owner-explicit", start: 0, end: 0, invocation: [] };
function deferred<T>() {
  let resolve!: (value: T) => void, reject!: (cause: unknown) => void;
  const promise = new Promise<T>((a, b) => {
    resolve = a;
    reject = b;
  });
  return { promise, resolve, reject };
}
const empty = () => success(undefined);

test("explicit interleaved roots isolate context and reject cross-root handles", async () => {
  const ready = deferred<Resource>(),
    release = deferred<void>();
  let a!: Resource;
  const first = runExplicitRoot(async (ctx) => {
    a = registerResourceWithContext(ctx, "pool", { root: "a" }, empty);
    ready.resolve(a);
    await release.promise;
    const read = await useResourceWithContext(ctx, a, "pool", (operation, native) => {
      expect(operation).toBe(ctx);
      return success((native as { root: string }).root);
    });
    expect(value(read)).toBe("a");
    await closeResourceWithContext(ctx, a, "pool");
    return success("first");
  });
  const second = runExplicitRoot(async (ctx) => {
    const foreign = await ready.promise;
    const refused = await invoke(
      () => useResourceWithContext(ctx, foreign, "pool", () => success(undefined)),
      origin,
    );
    expect(refused.kind).toBe("standard");
    if (refused.kind === "standard")
      expect(standardFailureKind(refused.value)).toBe("resource_state");
    const b = registerResourceWithContext(ctx, "pool", { root: "b" }, empty);
    await Promise.resolve();
    await Promise.resolve();
    expect(
      value(
        await useResourceWithContext(ctx, b, "pool", (operation, native) => {
          expect(operation).toBe(ctx);
          return success((native as { root: string }).root);
        }),
      ),
    ).toBe("b");
    await closeResourceWithContext(ctx, b, "pool");
    release.resolve();
    return success("second");
  });
  const results = await Promise.all([first, second]);
  expect(results.map((result) => value(result.completion))).toEqual(["first", "second"]);
  expect(results.every((result) => !result.cleanupFailed)).toBe(true);
});

test("explicit tokens ignore the ambient async store", async () => {
  const result = await runOwnedRoot(async () => {
    const ambientToken = registerResource("pool", { side: "ambient" }, empty);
    const explicit = await runExplicitRoot(async (ctx) => {
      const token = registerResourceWithContext(ctx, "pool", { side: "explicit" }, empty);
      await Promise.resolve();
      const own = await useResourceWithContext(ctx, token, "pool", (operation, native) => {
        expect(operation).toBe(ctx);
        return success((native as { side: string }).side);
      });
      expect(value(own)).toBe("explicit");
      const refused = await invoke(
        () => useResourceWithContext(ctx, ambientToken, "pool", () => success(undefined)),
        origin,
      );
      expect(refused.kind).toBe("standard");
      if (refused.kind === "standard")
        expect(standardFailureKind(refused.value)).toBe("resource_state");
      await closeResourceWithContext(ctx, token, "pool");
      return success("explicit");
    });
    expect(value(explicit.completion)).toBe("explicit");
    expect(explicit.cleanupFailed).toBe(false);
    await closeResource(ambientToken, "pool");
    return empty();
  });
  expect(result.cleanupFailed).toBe(false);
});

test("explicit nested scopes and guarded callbacks survive suspension", async () => {
  const gate = deferred<void>(),
    invoked = deferred<void>();
  let token!: Resource,
    guarded!: () => Promise<Completion<unknown>>,
    uses = 0,
    innerClosed = false;
  const result = await runExplicitRoot(async (ctx) => {
    const outer = await withScopeWithContext(ctx, async (scopeCtx, scope) => {
      expect(scopeCtx).not.toBe(ctx);
      token = registerResourceWithContext(scopeCtx, "transaction", {}, empty, {
        scopeManaged: true,
      });
      const inner = await withScopeWithContext(scopeCtx, async (innerCtx, innerScope) => {
        expect(innerCtx).not.toBe(scopeCtx);
        expect(innerScope).not.toBe(scope);
        const child = registerResourceWithContext(innerCtx, "transaction", {}, empty, {
          scopeManaged: true,
        });
        expect(resourceStatus(child).state).toBe("open");
        return success(child);
      });
      innerClosed = resourceStatus(value(inner)).state === "closed";
      guarded = guardCallbackWithContext(scope, async (callbackCtx, ...args) => {
        expect(args).toEqual([]);
        invoked.resolve();
        await gate.promise;
        return useResourceWithContext(callbackCtx, token, "transaction", () => {
          uses++;
          return success(undefined);
        });
      });
      const pending = guarded();
      await invoked.promise;
      gate.resolve();
      const outcome = await pending;
      expect(outcome.kind).toBe("ok");
      return empty();
    });
    return outer;
  });
  expect(result.completion.kind).toBe("ok");
  expect(innerClosed).toBe(true);
  expect(uses).toBe(1);
  expect(resourceStatus(token).state).toBe("closed");
  expect(result.cleanupFailed).toBe(false);
  const refused = await invoke(() => guarded(), origin);
  expect(refused.kind).toBe("standard");
  if (refused.kind === "standard")
    expect(standardFailureKind(refused.value)).toBe("resource_state");
});

test("explicit leases acquire after suspension and closing preserves subleases", async () => {
  const proceed = deferred<void>(),
    finished = deferred<void>();
  const events: string[] = [];
  let token!: Resource;
  const result = await runExplicitRoot(async (ctx) => {
    token = registerResourceWithContext(ctx, "test.pool", { value: 9 }, () => {
      events.push("native-close");
      return empty();
    });
    const group = launchOwnedWithContext(ctx, [
      {
        captures: [token],
        run: async (taskCtx) => {
          await proceed.promise;
          const used = await useResourceWithContext(
            taskCtx,
            token,
            "test.pool",
            (operation, native) => {
              expect(operation).toBe(taskCtx);
              events.push("sublease");
              return success((native as { value: number }).value);
            },
          );
          finished.resolve();
          return used;
        },
      },
    ]);
    group.publish([]);
    const closing = closeResourceWithContext(ctx, token, "test.pool", {
      milliseconds: 0,
      failure: () => failure(resourceStateFailure(undefined, origin)),
    });
    const timed = await closing;
    expect(timed.kind).toBe("standard");
    expect(resourceStatus(token)).toMatchObject({ state: "closing", leases: 1 });
    expect(events).toEqual([]);
    const refused = await invoke(() => {
      launchOwnedWithContext(ctx, [{ captures: [token], run: empty }]);
      return empty();
    }, origin);
    expect(refused.kind).toBe("standard");
    expect(resourceStatus(token).leases).toBe(1);
    proceed.resolve();
    await finished.promise;
    return empty();
  });
  expect(events).toEqual(["sublease", "native-close"]);
  expect(resourceStatus(token).state).toBe("closed");
  expect(result.cleanupFailed).toBe(true);
});

test("explicit root settles late losers and observes late failures", async () => {
  const winner = deferred<Completion>(),
    loser = deferred<Completion>(),
    selected = deferred<void>();
  const diagnostics: OwnerDiagnostic[] = [];
  let done = false;
  const root = runExplicitRoot(
    async (ctx) => {
      const group = launchOwnedWithContext(ctx, [
        { run: () => winner.promise, captures: [] },
        { run: () => loser.promise, captures: [] },
      ]);
      const result = await group.promises[0];
      group.publish([0]);
      selected.resolve();
      return result;
    },
    (d) => {
      diagnostics.push(d);
    },
  ).then((result) => {
    done = true;
    return result;
  });
  winner.resolve(success(7n));
  await selected.promise;
  await Promise.resolve();
  expect(done).toBe(false);
  const cause = Object.assign(new Error("late boom"), { privatePayload: "private secret" });
  loser.reject(cause);
  const result = await root;
  expect(value(result.completion)).toBe(7n);
  expect(result.cleanupFailed).toBe(false);
  expect(diagnostics.length).toBe(1);
  expect(diagnostics[0].message).toBe("Error: late boom");
  expect(diagnostics[0].phase).toBe("late");
  expect(diagnostics[0].category).toBe("native_exception");
  expect(JSON.stringify(diagnostics)).not.toContain("private secret");
});

test("explicit guarded callback faults settle without replacing the root", async () => {
  const registered = deferred<void>(),
    bodyExit = deferred<void>(),
    ready = deferred<void>(),
    finish = deferred<void>();
  let callback!: () => Promise<Completion<unknown>>,
    token!: Resource,
    done = false;
  const diagnostics: OwnerDiagnostic[] = [];
  const root = runExplicitRoot(
    (ctx) =>
      withScopeWithContext(ctx, async (scopeCtx, scope) => {
        callback = guardCallbackWithContext(scope, async (callbackCtx) => {
          token = registerResourceWithContext(callbackCtx, "callback", {}, empty, {
            scopeManaged: true,
          });
          ready.resolve();
          await finish.promise;
          throw new Error("callback boom");
        });
        registered.resolve();
        await bodyExit.promise;
        return empty();
      }),
    (d) => {
      diagnostics.push(d);
    },
  ).then((result) => {
    done = true;
    return result;
  });
  await registered.promise;
  const pending = callback();
  bodyExit.resolve();
  await ready.promise;
  await Promise.resolve();
  await Promise.resolve();
  expect(done).toBe(false);
  expect(resourceStatus(token).leases).toBe(1);
  finish.resolve();
  const callbackResult = await pending;
  expect(callbackResult.kind).toBe("standard");
  if (callbackResult.kind === "standard")
    expect(standardFailureKind(callbackResult.value)).toBe("native_exception");
  const result = await root;
  expect(result.completion.kind).toBe("ok");
  expect(result.cleanupFailed).toBe(false);
  expect(resourceStatus(token).state).toBe("closed");
  expect(diagnostics).toEqual([]);
});

test("explicit disposal closes once and refuses post-close token use", async () => {
  let closes = 0,
    ctx!: OwnerContext,
    token!: Resource;
  const result = await runExplicitRoot(async (rootCtx) => {
    ctx = rootCtx;
    token = registerResourceWithContext(ctx, "pool", {}, () => {
      closes++;
      return empty();
    });
    expect((await closeResourceWithContext(ctx, token, "pool")).kind).toBe("ok");
    const again = await invoke(() => closeResourceWithContext(ctx, token, "pool"), origin);
    expect(again.kind).toBe("standard");
    return empty();
  });
  expect(closes).toBe(1);
  expect(result.cleanupFailed).toBe(false);
  const late = await invoke(
    () => useResourceWithContext(ctx, token, "pool", () => success(undefined)),
    origin,
  );
  expect(late.kind).toBe("standard");
  if (late.kind === "standard") expect(standardFailureKind(late.value)).toBe("resource_state");
});

test("explicit wrong-kind, forged and forged-context handles reject", async () => {
  let touched = 0;
  const result = await runExplicitRoot(async (ctx) => {
    const token = registerResourceWithContext(ctx, "pool", {}, empty);
    for (const [handle, kind] of [
      [token, "transaction"],
      [{}, "pool"],
    ] as const) {
      const refused = await invoke(
        () =>
          useResourceWithContext(ctx, handle, kind, () => {
            touched++;
            return empty();
          }),
        origin,
      );
      expect(refused.kind).toBe("standard");
      if (refused.kind === "standard")
        expect(standardFailureKind(refused.value)).toBe("resource_state");
    }
    const forgedContext = await invoke(
      () =>
        useResourceWithContext("not-a-context" as unknown as OwnerContext, token, "pool", () =>
          success(undefined),
        ),
      origin,
    );
    expect(forgedContext.kind).toBe("standard");
    if (forgedContext.kind === "standard")
      expect(standardFailureKind(forgedContext.value)).toBe("native_exception");
    await closeResourceWithContext(ctx, token, "pool");
    return empty();
  });
  expect(result.cleanupFailed).toBe(false);
  expect(touched).toBe(0);
});

test("explicit participants retain callable capture evidence across suspension", async () => {
  const gate = deferred<void>();
  let token!: Resource,
    closed = false,
    uses = 0;
  const result = await runExplicitRoot(async (ctx) => {
    token = registerResourceWithContext(
      ctx,
      "pool",
      {},
      () => {
        closed = true;
        return empty();
      },
      { scopeManaged: true },
    );
    const callback = async (taskCtx: OwnerContext): Promise<Completion<unknown>> => {
      await gate.promise;
      uses++;
      return useResourceWithContext(taskCtx, token, "pool", () => success(undefined));
    };
    registerCallableCaptures(callback, [token]);
    const group = launchOwnedWithContext(ctx, [
      { captures: [callback], run: (taskCtx) => callback(taskCtx) },
    ]);
    group.publish([]);
    expect(resourceStatus(token).leases).toBe(1);
    gate.resolve();
    const outcome = await group.promises[0];
    expect(outcome.kind).toBe("ok");
    await closeResourceWithContext(ctx, token, "pool");
    return empty();
  });
  expect(uses).toBe(1);
  expect(closed).toBe(true);
  expect(result.cleanupFailed).toBe(false);
});

test("explicit native subwork joins its owner's leases", async () => {
  const result = await runExplicitRoot(async (ctx) => {
    const token = registerResourceWithContext(ctx, "pool", {}, empty);
    const group = launchOwnedWithContext(ctx, [
      {
        captures: [token],
        run: async (taskCtx) => {
          const nested = launchNativeWithContext(taskCtx, {
            captures: [token],
            run: (nativeCtx) =>
              useResourceWithContext(nativeCtx, token, "pool", () => success("nested")),
          });
          const [outcome] = await Promise.all(nested.promises);
          nested.publish([0]);
          return outcome;
        },
      },
    ]);
    const [outcome] = await Promise.all(group.promises);
    group.publish([0]);
    await closeResourceWithContext(ctx, token, "pool");
    return outcome;
  });
  expect(value(result.completion)).toBe("nested");
  expect(result.cleanupFailed).toBe(false);
});

test("explicit coordination settles participants with threaded context", async () => {
  const gate = deferred<void>();
  const result = await runExplicitRoot(async (ctx) => {
    const token = registerResourceWithContext(ctx, "pool", { n: 3 }, empty);
    const selection = await settleWithContext(ctx, "all", [
      {
        captures: [token],
        run: async (taskCtx) => {
          await gate.promise;
          return useResourceWithContext(taskCtx, token, "pool", (operation, native) => {
            expect(operation).toBe(taskCtx);
            return success((native as { n: number }).n);
          });
        },
      },
      {
        captures: [],
        run: async () => {
          gate.resolve();
          return success(1n);
        },
      },
    ]);
    expect(selection.kind).toBe("all");
    if (selection.kind !== "all") throw new Error("expected all");
    expect(selection.outcomes.map((outcome) => value(outcome))).toEqual([3, 1n]);
    await closeResourceWithContext(ctx, token, "pool");
    return empty();
  });
  expect(result.cleanupFailed).toBe(false);
});
