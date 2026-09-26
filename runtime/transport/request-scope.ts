// Ambient per-request operation scope (E04, server-only).
//
// Dispatch installs one scope per handler run; budgeted SQL and fetch
// adapters resolve it implicitly, so disconnect and shutdown expire
// each in-flight operation with its own unknown-write outcome. This
// module imports node:async_hooks and stays server-side: browser action
// clients take explicit bounds only and never import it.
import { AsyncLocalStorage } from "node:async_hooks";
import { createCollectorSink, type EscalationSink } from "./operation-budget.ts";
import { createRequestBudget, type RequestBudget } from "./request-budget.ts";

// ScopeAbortReason is the complete vocabulary for request-scope expiry:
// peer disconnect observed on the qualified serve-side Request.signal,
// or server shutdown propagating into live requests.
export type ScopeAbortReason = "disconnect" | "shutdown";

export type RequestScope = Readonly<{
  // Shared time budget, or undefined when the request carries no time
  // bound yet (a later server-config slice supplies the total; until
  // then expiry arrives only through expire()). Never invent a default
  // total here: unbounded-until-expired preserves legacy behavior.
  budget: RequestBudget | undefined;
  // Aborts exactly once, on the first expire() call, carrying the first
  // reason. Operations race this signal alongside their timers.
  signal: AbortSignal;
  // Idempotent: the first reason wins and later calls are silent no-ops.
  expire(reason: ScopeAbortReason): void;
  expired(): boolean;
  // Collector the owner drains; dispatch hands it to the E06 reporter.
  sink: EscalationSink;
}>;

const scopes = new AsyncLocalStorage<RequestScope>();

export function createRequestScope(options?: {
  totalMs?: number;
  sink?: EscalationSink;
}): RequestScope {
  const controller = new AbortController();
  let expired = false;
  return Object.freeze({
    budget: options?.totalMs === undefined ? undefined : createRequestBudget(options.totalMs),
    signal: controller.signal,
    expire(reason: ScopeAbortReason): void {
      if (expired) return;
      if (reason !== "disconnect" && reason !== "shutdown")
        throw new TypeError("invalid scope abort reason");
      expired = true;
      controller.abort(reason);
    },
    expired(): boolean {
      return expired;
    },
    sink: options?.sink ?? createCollectorSink().sink,
  });
}

export function runWithRequestScope<T>(scope: RequestScope, body: () => T): T {
  return scopes.run(scope, body);
}

export function currentRequestScope(): RequestScope | undefined {
  return scopes.getStore();
}

// scopeRaceInput resolves the budget half of a boundary race: an
// explicit budget wins over the ambient scope budget for time
// accounting, while the ambient scope signal (disconnect/shutdown
// liveness) and sink always apply when a scope is active.
export function scopeRaceInput(explicit?: RequestBudget): Readonly<{
  budget: RequestBudget | undefined;
  scopeSignal: AbortSignal | undefined;
  sink: EscalationSink | undefined;
}> {
  const scope = currentRequestScope();
  return {
    budget: explicit ?? scope?.budget,
    scopeSignal: scope?.signal,
    sink: scope?.sink,
  };
}
