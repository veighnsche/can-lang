// R14 native tenant token-budget guard: ai_budget::within scope, guarded
// dispatch, usage settlement, and profile qualification (H07).
//
// F01 supplies the authenticated tenant/pool vocabulary
// (runtime/outbound/identity.ts) and the durable reserve/fence/settle
// ledger (runtime/outbound/ledger.ts); this module consumes both as-is
// and enforces the published token-budget contract at dispatch:
// qualify a complete-call upper bound U, atomically reserve U before
// sending, reject immediately with typed failures, decode authoritative
// usage, and retain unknown holds under fixed-epoch rules.
//
// Server startup injects the ledger store, the qualified-profile
// registry, and the per-connection binding (permitted pool + pinned
// profile). Guarded adapters reject calls without scope context and
// offer no unguarded path: unmetered applications stay explicitly
// separate. Metered profiles stay disabled until their
// complete-request bound qualifies here; unqualified profiles reject
// before send. No estimated cap, post-debit, or queue substitutes.
//
// Failures, reports, and ledger records never carry credentials,
// endpoints, prompt text, or raw provider bodies.
import { record } from "../data.ts";
import { ownBytes } from "../bytes.ts";
import { parseDocument } from "../codec/document.ts";
import { caught, checkedCompletion, failure, invoke, type Completion } from "../completion.ts";
import { createDomainRuntime } from "../domain.ts";
import type { FailureOrigin } from "../failure.ts";
import {
  invocationContext,
  invocationId,
  isInvocationContext,
  nestContext,
  newCorrelationId,
  newInvocationId,
  poolId,
  type CorrelationId,
  type InvocationContext,
  type InvocationId,
  type PoolId,
} from "../outbound/identity.ts";
import {
  fence,
  meteringProfile,
  meteringUsage,
  profileId,
  reconcile,
  releaseUndispatched,
  reserve,
  settle,
  usageTotal,
  type InvocationRecord,
  type LedgerStore,
  type ReleaseOutcome,
  type UnavailableReason,
  type Usage,
} from "../outbound/ledger.ts";

export type BudgetTypes = Readonly<{ exceeded: string; unavailable: string }>;

// QualifiedProfile pins one metering proof: provider/model/version
// identity plus the conservative upper bound U for the complete
// encoded call's provider input plus output consumption. Bytes or an
// estimated tokenizer count never qualify; changing provider, model,
// or version invalidates qualification unless the existing proof
// explicitly covers that change.
export type QualifiedProfile = Readonly<{
  provider: string;
  model: string;
  version: string;
  upperBound: number;
}>;

export type ProfileRegistry = Readonly<{
  qualify: (profile: {
    provider: unknown;
    model: unknown;
    version: unknown;
    upperBound: unknown;
  }) => QualifiedProfile;
  boundFor: (provider: unknown, model: unknown, version: unknown) => number | undefined;
  profiles: () => readonly QualifiedProfile[];
}>;

export function createProfileRegistry(): ProfileRegistry {
  const bounds = new Map<string, QualifiedProfile>();
  const keyOf = (provider: string, model: string, version: string): string =>
    JSON.stringify([provider, model, version]);
  return Object.freeze({
    qualify(profile: {
      provider: unknown;
      model: unknown;
      version: unknown;
      upperBound: unknown;
    }): QualifiedProfile {
      const pinned = meteringProfile(profile);
      if (!Number.isSafeInteger(profile.upperBound) || (profile.upperBound as number) < 1)
        throw new TypeError("invalid profile upper bound");
      const upperBound = profile.upperBound as number;
      const key = keyOf(pinned.provider, pinned.model, pinned.version);
      const existing = bounds.get(key);
      if (existing !== undefined) {
        if (existing.upperBound !== upperBound)
          throw new TypeError("conflicting profile qualification");
        return existing;
      }
      const qualified: QualifiedProfile = Object.freeze({ ...pinned, upperBound });
      bounds.set(key, qualified);
      return qualified;
    },
    boundFor(provider: unknown, model: unknown, version: unknown): number | undefined {
      let pinned;
      try {
        pinned = meteringProfile({ provider, model, version });
      } catch {
        return undefined;
      }
      return bounds.get(keyOf(pinned.provider, pinned.model, pinned.version))?.upperBound;
    },
    profiles(): readonly QualifiedProfile[] {
      return Object.freeze([...bounds.values()]);
    },
  });
}

// decodeProviderUsage extracts trustworthy top-level input/output
// counters from one raw provider response body. Both native adapters
// report top-level usage.input_tokens/usage.output_tokens; only those
// two counters sum into the tenant allowance — cached, reasoning, or
// other breakdown fields are never added a second time. Anything
// untrustworthy (missing/invalid usage, unparsable body, byte cap)
// yields undefined so the guard retains the full hold; it never
// throws for provider data.
export function decodeProviderUsage(input: unknown, bytes: number): Usage | undefined {
  if (input === undefined) return undefined;
  let parsed: unknown;
  try {
    parsed = parseDocument(input, bytes).parsed;
  } catch {
    return undefined;
  }
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) return undefined;
  const usage = (parsed as Record<string, unknown>).usage;
  if (usage === null || typeof usage !== "object" || Array.isArray(usage)) return undefined;
  const shaped = usage as Record<string, unknown>;
  return meteringUsage(shaped.input_tokens, shaped.output_tokens);
}

export type AccountingOutcome = "settled" | "unresolved" | "breach" | "released";

// AccountingReport is the sanitized correlated report emitted for
// every admitted attempt: safe identities and token counters only.
// No credentials, endpoints, prompt text, or raw provider bodies.
export type AccountingReport = Readonly<{
  tenant: string;
  pool: string;
  correlation: string;
  invocation: string;
  profile: string;
  upperBound: number;
  epochStart: number;
  epochEnd: number;
  outcome: AccountingOutcome;
  actual?: number;
  released?: number;
  duplicate?: boolean;
}>;

export type BudgetOperation<T> = (
  context: InvocationContext,
) => Completion<T> | Promise<Completion<T>>;

export type GuardedSend<T> = Readonly<{
  completion: Completion<T>;
  body: Uint8Array | undefined;
}>;

export type GuardedAttempt<T> = Readonly<{
  context: unknown;
  where: FailureOrigin;
  operation: string;
  maxBodyBytes: number;
  invocationId?: unknown;
  send: () => Promise<GuardedSend<T>>;
}>;

export type GuardedConnectionConfig = Readonly<{
  permittedPool: unknown;
  profile: Readonly<{ provider: unknown; model: unknown; version: unknown }>;
}>;

export type GuardedConnection = Readonly<{
  connection: GuardedConnectionConfig;
  dispatch: <T>(attempt: GuardedAttempt<T>) => Promise<Completion<T>>;
}>;

export type SettleUsageInput = Readonly<{
  invocationId: unknown;
  inputTokens: unknown;
  outputTokens: unknown;
}>;

export type SettleUsageOutcome =
  | Readonly<{
      outcome: "settled";
      actual: number;
      released: number;
      duplicate: boolean;
      breach?: Readonly<{ profile: string; upperBound: number; actual: number }>;
    }>
  | Readonly<{ outcome: "unavailable"; reason: UnavailableReason }>;

export type BudgetDeps = Readonly<{
  store: LedgerStore;
  registry: ProfileRegistry;
  nowMs?: () => number;
  report?: (report: AccountingReport) => void;
}>;

export type BudgetGuard = Readonly<{
  within: <T>(
    tenant: unknown,
    pool: unknown,
    operation: BudgetOperation<T>,
    where: FailureOrigin,
    operationName: string,
    active?: unknown,
  ) => Promise<Completion<T>>;
  dispatch: <T>(
    connection: GuardedConnectionConfig,
    attempt: GuardedAttempt<T>,
  ) => Promise<Completion<T>>;
  settleUsage: (input: SettleUsageInput) => Promise<SettleUsageOutcome>;
  releaseReservation: (invocationId: unknown) => Promise<ReleaseOutcome>;
}>;

export function createBudgetGuard(
  domain: ReturnType<typeof createDomainRuntime>,
  types: BudgetTypes,
  deps: BudgetDeps,
): BudgetGuard {
  const clock = deps.nowMs ?? (() => Date.now());
  const emit = deps.report;
  // In-flight invocation IDs for this guard instance: a second
  // concurrent dispatch reusing an ID never sends on the same
  // reservation. Sequential reuse (retry after an ambiguous commit)
  // still reconciles by ID once the first dispatch returns.
  const inflight = new Set<string>();

  function exceeded(
    pool: PoolId,
    required: number,
    remaining: number,
    epochResetMs: number,
    where: FailureOrigin,
    operation: string,
  ): Completion<never> {
    return failure(
      domain.create(
        types.exceeded,
        record(types.exceeded, [
          ["pool", pool as string],
          ["required", BigInt(required)],
          ["remaining", BigInt(remaining)],
          ["epoch_reset_ms", BigInt(epochResetMs)],
        ]),
        where,
        undefined,
        { boundary: "native", operation },
      ),
    );
  }

  function unavailable(
    reason: UnavailableReason,
    correlation: CorrelationId,
    id: InvocationId,
    where: FailureOrigin,
    operation: string,
  ): Completion<never> {
    return failure(
      domain.create(
        types.unavailable,
        record(types.unavailable, [
          ["reason", reason],
          ["correlation", correlation as string],
          ["invocation", id as string],
        ]),
        where,
        undefined,
        { boundary: "native", operation },
      ),
    );
  }

  function report(
    record_: InvocationRecord,
    outcome: AccountingOutcome,
    detail: Readonly<{ actual?: number; released?: number; duplicate?: boolean }> = {},
  ): void {
    emit?.(
      Object.freeze({
        tenant: record_.tenant as string,
        pool: record_.pool as string,
        correlation: record_.correlation as string,
        invocation: record_.invocationId as string,
        profile: record_.profileIdentity,
        upperBound: record_.upperBound,
        epochStart: record_.epochStart,
        epochEnd: record_.epochEnd,
        outcome,
        ...detail,
      }),
    );
  }

  async function dispatch<T>(
    connection: GuardedConnectionConfig,
    attempt: GuardedAttempt<T>,
  ): Promise<Completion<T>> {
    const { where, operation } = attempt;
    // Missing or malformed scope context rejects before any ledger
    // I/O; the failure still carries safe correlation identity.
    if (attempt.context === undefined)
      return unavailable(
        "missing-context",
        newCorrelationId(),
        newInvocationId(),
        where,
        operation,
      );
    if (!isInvocationContext(attempt.context))
      return unavailable("invalid-policy", newCorrelationId(), newInvocationId(), where, operation);
    const context = attempt.context;
    if (!Number.isSafeInteger(attempt.maxBodyBytes) || attempt.maxBodyBytes < 1)
      return unavailable(
        "invalid-policy",
        context.correlation,
        newInvocationId(),
        where,
        operation,
      );
    // An arbitrary pool name never creates fresh allowance: the scope
    // pool must be the connection's configured permitted pool.
    let permitted: PoolId;
    try {
      permitted = poolId(connection.permittedPool);
    } catch {
      return unavailable(
        "invalid-policy",
        context.correlation,
        newInvocationId(),
        where,
        operation,
      );
    }
    if (context.pool !== permitted)
      return unavailable(
        "invalid-policy",
        context.correlation,
        newInvocationId(),
        where,
        operation,
      );
    let profile;
    try {
      profile = meteringProfile(connection.profile);
    } catch {
      return unavailable(
        "invalid-policy",
        context.correlation,
        newInvocationId(),
        where,
        operation,
      );
    }
    const upperBound = deps.registry.boundFor(profile.provider, profile.model, profile.version);
    if (upperBound === undefined)
      return unavailable(
        "missing-qualification",
        context.correlation,
        newInvocationId(),
        where,
        operation,
      );
    let id: InvocationId;
    try {
      id =
        attempt.invocationId === undefined ? newInvocationId() : invocationId(attempt.invocationId);
    } catch {
      return unavailable(
        "invalid-policy",
        context.correlation,
        newInvocationId(),
        where,
        operation,
      );
    }
    if (inflight.has(id as string))
      return unavailable("conflicting-settlement", context.correlation, id, where, operation);
    // Registered synchronously with the check so concurrent same-ID
    // dispatches serialize here instead of racing through reserve and
    // fence toward a double send on one reservation.
    inflight.add(id as string);
    try {
      const reserved = await reserve(deps.store, {
        tenant: context.tenant,
        pool: context.pool,
        correlation: context.correlation,
        profile: { provider: profile.provider, model: profile.model, version: profile.version },
        qualified: true,
        upperBound,
        invocationId: id,
        nowMs: clock(),
      });
      if (reserved.outcome === "exceeded")
        return exceeded(
          reserved.pool,
          reserved.required,
          reserved.remaining,
          reserved.epochResetMs,
          where,
          operation,
        );
      if (reserved.outcome === "unavailable")
        return unavailable(
          reserved.reason,
          reserved.correlation,
          reserved.invocationId,
          where,
          operation,
        );
      const admitted = reserved.record;
      const fenced = await fence(deps.store, id);
      if (fenced.outcome === "unavailable") {
        if (fenced.reason === "ambiguous-commit") {
          // The fence may or may not have persisted: never dispatch on
          // the ambiguous reservation; the hold stays durable.
          report(admitted, "unresolved");
          return unavailable("ambiguous-commit", admitted.correlation, id, where, operation);
        }
        if (fenced.reason === "conflicting-settlement")
          return unavailable(fenced.reason, admitted.correlation, id, where, operation);
        // A definitively unfenced attempt proves no send could have
        // occurred, so the reservation releases; anything else keeps
        // its hold.
        const released = await releaseUndispatched(deps.store, id);
        if (released.outcome === "released") report(released.record, "released");
        else report(admitted, "unresolved");
        return unavailable(fenced.reason, admitted.correlation, id, where, operation);
      }
      let sent: GuardedSend<T>;
      try {
        sent = await attempt.send();
        checkedCompletion(sent.completion);
      } catch (cause) {
        // Crash or uncertain provider attempt: the fenced hold stays
        // durable and the original cause stays intact.
        report(admitted, "unresolved");
        return caught(cause, where);
      }
      const usage =
        sent.body === undefined
          ? undefined
          : decodeProviderUsage(ownBytes(sent.body), attempt.maxBodyBytes);
      if (usage === undefined) {
        // Missing/invalid usage, timeout, cancellation, non-2xx, or
        // any failure the transport reports without a decoded body:
        // the provider completion stays intact and the full hold
        // stays unresolved and durable.
        report(admitted, "unresolved");
        return sent.completion;
      }
      const settled = await settle(deps.store, {
        invocationId: id,
        inputTokens: usage.inputTokens,
        outputTokens: usage.outputTokens,
      });
      if (settled.outcome === "settled") {
        if (settled.breach !== undefined) {
          // A trustworthy detected profile breach invalidates the
          // budgeted result even when the provider call succeeded.
          report(admitted, "breach", { actual: settled.actual });
          return unavailable("breached-profile", admitted.correlation, id, where, operation);
        }
        report(
          admitted,
          "settled",
          settled.duplicate
            ? { actual: settled.actual, released: 0, duplicate: true }
            : { actual: settled.actual, released: settled.released },
        );
        return sent.completion;
      }
      if (settled.reason === "conflicting-settlement")
        return unavailable(settled.reason, admitted.correlation, id, where, operation);
      // Uncertain settlement keeps the provider completion intact;
      // only authoritative usage or definitive no-dispatch evidence
      // can later reconcile the hold.
      report(admitted, "unresolved");
      return sent.completion;
    } finally {
      inflight.delete(id as string);
    }
  }

  async function settleUsage(input: SettleUsageInput): Promise<SettleUsageOutcome> {
    const settled = await settle(deps.store, input);
    if (settled.outcome === "settled") {
      if (settled.breach !== undefined) {
        report(settled.record, "breach", { actual: settled.actual });
        return Object.freeze({
          outcome: "settled",
          actual: settled.actual,
          released: settled.released,
          duplicate: settled.duplicate,
          breach: settled.breach,
        });
      }
      report(
        settled.record,
        "settled",
        settled.duplicate
          ? { actual: settled.actual, released: 0, duplicate: true }
          : { actual: settled.actual, released: settled.released },
      );
      return Object.freeze({
        outcome: "settled",
        actual: settled.actual,
        released: settled.released,
        duplicate: settled.duplicate,
      });
    }
    if (
      settled.reason === "invalid-usage" ||
      settled.reason === "storage-unavailable" ||
      settled.reason === "ambiguous-commit"
    ) {
      // Late or retried usage that cannot reconcile yet: report the
      // hold it leaves behind when the record is known.
      const found = await reconcile(deps.store, input.invocationId);
      if (found.status === "found") {
        if (found.record.state === "settled" && found.record.usage !== undefined)
          report(found.record, "settled", {
            actual: usageTotal(found.record.usage),
            released: 0,
            duplicate: true,
          });
        else report(found.record, "unresolved");
      }
    }
    return Object.freeze({ outcome: "unavailable", reason: settled.reason });
  }

  async function releaseReservation(id: unknown): Promise<ReleaseOutcome> {
    const released = await releaseUndispatched(deps.store, id);
    if (released.outcome === "released") report(released.record, "released");
    else if (released.outcome === "held") report(released.record, "unresolved");
    return released;
  }

  async function within<T>(
    tenant: unknown,
    pool: unknown,
    operation: BudgetOperation<T>,
    where: FailureOrigin,
    operationName: string,
    active?: unknown,
  ): Promise<Completion<T>> {
    let next: InvocationContext;
    try {
      next = invocationContext(tenant, pool);
    } catch {
      return unavailable(
        "invalid-policy",
        newCorrelationId(),
        newInvocationId(),
        where,
        operationName,
      );
    }
    let context = next;
    if (active !== undefined) {
      if (!isInvocationContext(active))
        return unavailable(
          "invalid-policy",
          next.correlation,
          newInvocationId(),
          where,
          operationName,
        );
      try {
        context = nestContext(active, next);
      } catch {
        // Nested scopes cannot replace the active tenant/pool.
        return unavailable(
          "invalid-policy",
          active.correlation,
          newInvocationId(),
          where,
          operationName,
        );
      }
    }
    // The scope adds context, not tokens: callback failures return
    // unchanged and completion leaks no tenant into another
    // invocation.
    return invoke(() => operation(context), where);
  }

  return Object.freeze({ within, dispatch, settleUsage, releaseReservation });
}

// guardConnection binds one guarded provider connection — its
// configured permitted pool and pinned metering profile — to a
// dispatch closure. Guarded adapters accept this binding and route
// every call through it; there is no unguarded path on a bound
// adapter. The returned shape is structural on purpose: shipped
// adapters mirror it without importing this server-native module.
export function guardConnection(
  guard: BudgetGuard,
  config: GuardedConnectionConfig,
): GuardedConnection {
  const frozen: GuardedConnectionConfig = Object.freeze({
    permittedPool: config.permittedPool,
    profile: Object.freeze({ ...config.profile }),
  });
  return Object.freeze({
    connection: frozen,
    dispatch: <T>(attempt: GuardedAttempt<T>): Promise<Completion<T>> =>
      guard.dispatch(frozen, attempt),
  });
}

export function budgetProfileLabel(profile: {
  provider: string;
  model: string;
  version: string;
}): string {
  return profileId(meteringProfile(profile));
}
