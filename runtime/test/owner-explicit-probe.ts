// Chromium-bundled explicit owner probe. Uses only the explicit token API:
// bundling stubs node:async_hooks with throwing methods, so any ambient
// context access fails the probe loudly instead of silently passing.
import {
  runExplicitRoot,
  launchOwnedWithContext,
  registerResourceWithContext,
  useResourceWithContext,
  closeResourceWithContext,
  resourceStatus,
  withScopeWithContext,
  guardCallbackWithContext,
  type OwnerContext,
  type OwnerDiagnostic,
  type Resource,
} from "../owner.ts";
import { settleWithContext } from "../coordination.ts";
import { success, invoke, type Completion } from "../completion.ts";
import { standardFailureKind } from "../failure.ts";

const origin = { source: "probe:owner-explicit", start: 0, end: 0, invocation: [] };
function deferred<T>() {
  let resolve!: (value: T) => void, reject!: (cause: unknown) => void;
  const promise = new Promise<T>((a, b) => {
    resolve = a;
    reject = b;
  });
  return { promise, resolve, reject };
}
const empty = () => success(undefined);
const describe = (completion: Completion<unknown>): string =>
  completion.kind === "standard" ? standardFailureKind(completion.value) : completion.kind;

export type ExplicitProbeResult = Readonly<{
  interleaved: Readonly<{
    outcomes: readonly string[];
    crossRefused: boolean;
    resources: readonly string[];
    clean: boolean;
  }>;
  guarded: Readonly<{
    completion: string;
    rootCompletion: string;
    resource: string;
    afterClose: string;
    clean: boolean;
  }>;
  leaseAfterSuspension: Readonly<{ completion: string; subleased: boolean; clean: boolean }>;
  lateLoser: Readonly<{
    completion: string;
    diagnostics: number;
    phase: string | null;
    clean: boolean;
  }>;
  disposal: Readonly<{
    closes: number;
    doubleRefused: boolean;
    postCloseRefused: boolean;
    clean: boolean;
  }>;
  nestedScopes: Readonly<{
    innerWasOpen: boolean;
    innerClosedWhileOuterOpen: boolean;
    outerClosed: boolean;
    clean: boolean;
  }>;
  coordination: Readonly<{ selection: string; values: readonly unknown[]; clean: boolean }>;
}>;

async function probeInterleaved(): Promise<ExplicitProbeResult["interleaved"]> {
  const gates = [deferred<void>(), deferred<void>()],
    used = [deferred<void>(), deferred<void>()];
  const tokens: Resource[] = [];
  let crossRefused = false;
  const roots = ["A", "B"].map((name, i) =>
    runExplicitRoot(async (ctx) => {
      const token = registerResourceWithContext(ctx, "probe", name, () => success(undefined), {
        scopeManaged: true,
      });
      tokens.push(token);
      if (i === 1) {
        const refused = await invoke(
          () => useResourceWithContext(ctx, tokens[0], "probe", () => success(undefined)),
          origin,
        );
        crossRefused = refused.kind === "standard" && describe(refused) === "resource_state";
      }
      await gates[i].promise;
      const result = await invoke(
        () =>
          useResourceWithContext(ctx, token, "probe", (operation, native) =>
            success(`${name}:${String(operation === ctx)}:${String(native)}`),
          ),
        origin,
      );
      used[i].resolve();
      await Promise.resolve();
      return result;
    }),
  );
  gates[1].resolve();
  await used[1].promise;
  gates[0].resolve();
  const results = await Promise.all(roots);
  return {
    outcomes: results.map((r) => describe(r.completion)),
    crossRefused,
    resources: tokens.map((token) => resourceStatus(token).state),
    clean: results.every((r) => !r.cleanupFailed),
  };
}

async function probeGuarded(): Promise<ExplicitProbeResult["guarded"]> {
  const registered = deferred<void>(),
    bodyExit = deferred<void>(),
    ready = deferred<void>(),
    finish = deferred<void>();
  let callback!: () => Promise<Completion<unknown>>, token!: Resource;
  const root = runExplicitRoot((ctx) =>
    withScopeWithContext(ctx, async (scopeCtx, scope) => {
      callback = guardCallbackWithContext(scope, async (callbackCtx) => {
        token = registerResourceWithContext(callbackCtx, "callback", {}, empty, {
          scopeManaged: true,
        });
        ready.resolve();
        await finish.promise;
        return useResourceWithContext(callbackCtx, token, "callback", () => success(undefined));
      });
      registered.resolve();
      await bodyExit.promise;
      return empty();
    }),
  );
  await registered.promise;
  const pending = callback();
  bodyExit.resolve();
  await ready.promise;
  finish.resolve();
  const callbackResult = await pending;
  const rootResult = await root;
  const afterClose = await invoke(() => callback(), origin);
  return {
    completion: describe(callbackResult),
    rootCompletion: describe(rootResult.completion),
    resource: resourceStatus(token).state,
    afterClose: describe(afterClose),
    clean: !rootResult.cleanupFailed,
  };
}

async function probeLeaseAfterSuspension(): Promise<ExplicitProbeResult["leaseAfterSuspension"]> {
  const proceed = deferred<void>();
  let subleased = false;
  const result = await runExplicitRoot(async (ctx) => {
    const token = registerResourceWithContext(ctx, "pool", { n: 7 }, empty);
    const group = launchOwnedWithContext(ctx, [
      {
        captures: [token],
        run: async (taskCtx: OwnerContext) => {
          await proceed.promise;
          return useResourceWithContext(taskCtx, token, "pool", (operation, native) => {
            subleased = operation === taskCtx && (native as { n: number }).n === 7;
            return success(undefined);
          });
        },
      },
    ]);
    group.publish([]);
    proceed.resolve();
    const outcome = await group.promises[0];
    await closeResourceWithContext(ctx, token, "pool");
    return outcome;
  });
  return {
    completion: describe(result.completion),
    subleased,
    clean: !result.cleanupFailed,
  };
}

async function probeLateLoser(): Promise<ExplicitProbeResult["lateLoser"]> {
  const winner = deferred<Completion<unknown>>(),
    loser = deferred<Completion<unknown>>(),
    selected = deferred<void>();
  const diagnostics: OwnerDiagnostic[] = [];
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
    (diagnostic) => {
      diagnostics.push(diagnostic);
    },
  );
  winner.resolve(success("winner"));
  await selected.promise;
  loser.reject(new Error("late boom"));
  const result = await root;
  return {
    completion: describe(result.completion),
    diagnostics: diagnostics.length,
    phase: diagnostics.length ? diagnostics[0].phase : null,
    clean: !result.cleanupFailed,
  };
}

async function probeDisposal(): Promise<ExplicitProbeResult["disposal"]> {
  let closes = 0,
    ctx!: OwnerContext,
    token!: Resource;
  const result = await runExplicitRoot(async (rootCtx) => {
    ctx = rootCtx;
    token = registerResourceWithContext(ctx, "pool", {}, () => {
      closes++;
      return empty();
    });
    await closeResourceWithContext(ctx, token, "pool");
    return empty();
  });
  const double = await invoke(() => closeResourceWithContext(ctx, token, "pool"), origin);
  const postClose = await invoke(
    () => useResourceWithContext(ctx, token, "pool", () => success(undefined)),
    origin,
  );
  return {
    closes,
    doubleRefused: double.kind === "standard",
    postCloseRefused: postClose.kind === "standard" && describe(postClose) === "resource_state",
    clean: !result.cleanupFailed,
  };
}

async function probeNestedScopes(): Promise<ExplicitProbeResult["nestedScopes"]> {
  let innerWasOpen = false,
    innerClosedWhileOuterOpen = false,
    outerToken!: Resource;
  const result = await runExplicitRoot(async (ctx) => {
    return withScopeWithContext(ctx, async (scopeCtx) => {
      outerToken = registerResourceWithContext(scopeCtx, "outer", {}, empty, {
        scopeManaged: true,
      });
      let innerToken!: Resource;
      await withScopeWithContext(scopeCtx, async (innerCtx) => {
        innerToken = registerResourceWithContext(innerCtx, "inner", {}, empty, {
          scopeManaged: true,
        });
        innerWasOpen = resourceStatus(innerToken).state === "open";
        return empty();
      });
      innerClosedWhileOuterOpen =
        resourceStatus(innerToken).state === "closed" &&
        resourceStatus(outerToken).state === "open";
      return empty();
    });
  });
  return {
    innerWasOpen,
    innerClosedWhileOuterOpen,
    outerClosed: resourceStatus(outerToken).state === "closed",
    clean: !result.cleanupFailed,
  };
}

async function probeCoordination(): Promise<ExplicitProbeResult["coordination"]> {
  const gate = deferred<void>();
  const result = await runExplicitRoot(async (ctx) => {
    const token = registerResourceWithContext(ctx, "pool", { n: 3 }, empty);
    const selection = await settleWithContext(ctx, "all", [
      {
        captures: [token],
        run: async (taskCtx: OwnerContext) => {
          await gate.promise;
          return useResourceWithContext(taskCtx, token, "pool", (operation, native) =>
            success(`${String(operation === taskCtx)}:${String((native as { n: number }).n)}`),
          );
        },
      },
      {
        captures: [],
        run: async () => {
          gate.resolve();
          return success("second");
        },
      },
    ]);
    const values =
      selection.kind === "all"
        ? selection.outcomes.map((outcome) =>
            outcome.kind === "ok" ? outcome.value : outcome.kind,
          )
        : [selection.kind];
    await closeResourceWithContext(ctx, token, "pool");
    return success(values);
  });
  return {
    selection: "all",
    values: result.completion.kind === "ok" ? (result.completion.value as readonly unknown[]) : [],
    clean: !result.cleanupFailed,
  };
}

export async function probeExplicitOwner(): Promise<ExplicitProbeResult> {
  return {
    interleaved: await probeInterleaved(),
    guarded: await probeGuarded(),
    leaseAfterSuspension: await probeLeaseAfterSuspension(),
    lateLoser: await probeLateLoser(),
    disposal: await probeDisposal(),
    nestedScopes: await probeNestedScopes(),
    coordination: await probeCoordination(),
  };
}
