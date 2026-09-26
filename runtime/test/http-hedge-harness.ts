// X-R04-2 hedge measurement harness (E05, test-only).
//
// Boundary-race hedge over the E04 cancel-absent vocabulary: each replica
// races its own raceBoundary against the shared request scope (budget,
// disconnect/shutdown signal, escalation sink). The primary starts at once;
// the secondary starts after hedgeDelayMs unless the primary already won.
// The first settled replica wins and the handler responds with its value;
// the loser is never waited on. Its boundary outcome (settled late,
// expired, or rejected) and any late settlement flow to the shared sink
// through the existing escalation/late records — no new vocabulary, no
// supervisor move, no catalogue change.
//
// Terminal precedence when nothing settles is expired over rejected: an
// expired replica's commit is honestly unknown, and a known failure
// elsewhere never resolves that uncertainty (C-C honesty rules).
//
// Replica shape contract (measured, not enforced): replicas must be
// side-effect-free or idempotent reads. The harness starts both replicas
// whenever the primary does not win before the delay, so hedged writes
// would double-apply. Abort policy: when abortLosers is set, the hedge
// aborts each still-pending cancelable loser's controller at the win.
// Cancel-present replicas (fetch) wire that signal into their native
// abort; cancel-absent replicas (SQL) set cancelable false and run to
// settlement owned. A signal is never passed to a SQL boundary:
// raceBoundary rejects one structurally.
import {
  checkOperationBound,
  raceBoundary,
  type EscalationRecord,
  type EscalationSink,
} from "../transport/operation-budget.ts";
import type {
  RequestBudget,
  UnknownWrite,
  UnknownWriteSource,
} from "../transport/request-budget.ts";

export type HedgeReplica<T> = Readonly<{
  source: UnknownWriteSource;
  cancelable: boolean;
  start: (input: Readonly<{ signal: AbortSignal | undefined }>) => Promise<T>;
}>;

export type HedgeOptions = Readonly<{
  boundMs: number;
  budget?: RequestBudget;
  scopeSignal?: AbortSignal;
  sink?: EscalationSink;
  hedgeDelayMs: number;
  abortLosers: boolean;
}>;

export type ReplicaFate<T> =
  | Readonly<{ replica: number; fate: "settled"; value: T }>
  | Readonly<{ replica: number; fate: "expired"; escalation: EscalationRecord }>
  | Readonly<{ replica: number; fate: "rejected"; error: unknown }>;

type HedgeCommon<T> = Readonly<{
  // Live snapshot of replica fates so far, in terminal order.
  fates: () => readonly ReplicaFate<T>[];
  // Resolves when every launched replica is terminal. This is the
  // boundary horizon, not native settlement: an expired replica's
  // operation may still run owned until its late record lands.
  settledAll: Promise<readonly ReplicaFate<T>[]>;
}>;

export type HedgeOutcome<T> = HedgeCommon<T> &
  Readonly<
    | { kind: "settled"; value: T; winner: number }
    | { kind: "unknown"; marker: UnknownWrite; escalation: EscalationRecord }
    | { kind: "failed"; errors: readonly unknown[] }
  >;

export async function hedge<T>(
  options: HedgeOptions,
  replicas: readonly HedgeReplica<T>[],
): Promise<HedgeOutcome<T>> {
  if (replicas.length !== 2) throw new TypeError("hedge runs exactly two replicas");
  const first = replicas[0];
  const second = replicas[1];
  if (first === undefined || second === undefined)
    throw new TypeError("hedge runs exactly two replicas");
  checkOperationBound(options.boundMs);
  if (
    !Number.isSafeInteger(options.hedgeDelayMs) ||
    options.hedgeDelayMs < 0 ||
    options.hedgeDelayMs > 2147483647
  )
    throw new TypeError("invalid hedge delay");
  if (typeof options.abortLosers !== "boolean") throw new TypeError("invalid hedge abort policy");

  const ordered = [first, second] as const;
  const controllers = [new AbortController(), new AbortController()] as const;
  const launched = new Set<number>();
  const pending = new Set<number>();
  const fates: ReplicaFate<T>[] = [];
  let done = false;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let resolveOutcome!: (outcome: HedgeOutcome<T>) => void;
  let resolveAll!: (fates: readonly ReplicaFate<T>[]) => void;
  const decided = new Promise<HedgeOutcome<T>>((resolve) => {
    resolveOutcome = resolve;
  });
  const settledAll = new Promise<readonly ReplicaFate<T>[]>((resolve) => {
    resolveAll = resolve;
  });
  const snapshot = (): readonly ReplicaFate<T>[] => Object.freeze([...fates]);
  const common = (): HedgeCommon<T> => ({ fates: snapshot, settledAll });

  function abortPending(winner: number): void {
    if (!options.abortLosers) return;
    for (const index of pending) {
      if (index === winner || !ordered[index]!.cancelable) continue;
      controllers[index]!.abort("hedged");
    }
  }

  function note(index: number, fate: ReplicaFate<T>): void {
    pending.delete(index);
    fates.push(fate);
    if (!done) {
      if (fate.fate === "settled") {
        done = true;
        if (timer !== undefined) clearTimeout(timer);
        abortPending(index);
        resolveOutcome({ ...common(), kind: "settled", value: fate.value, winner: index });
      } else if (pending.size === 0 && launched.size === ordered.length) {
        done = true;
        if (timer !== undefined) clearTimeout(timer);
        const expired = fates.find((entry) => entry.fate === "expired");
        if (expired !== undefined && expired.fate === "expired") {
          resolveOutcome({
            ...common(),
            kind: "unknown",
            marker: expired.escalation.marker,
            escalation: expired.escalation,
          });
        } else {
          resolveOutcome({
            ...common(),
            kind: "failed",
            errors: Object.freeze(
              fates.map((entry) => (entry.fate === "rejected" ? entry.error : undefined)),
            ),
          });
        }
      }
    }
    if (done && pending.size === 0) resolveAll(snapshot());
  }

  function launch(index: number): void {
    if (done || launched.has(index)) return;
    const replica = ordered[index]!;
    launched.add(index);
    pending.add(index);
    const signal =
      options.abortLosers && replica.cancelable ? controllers[index]!.signal : undefined;
    raceBoundary(
      {
        source: replica.source,
        boundMs: options.boundMs,
        budget: options.budget,
        scopeSignal: options.scopeSignal,
        signal,
        sink: options.sink,
      },
      () => replica.start({ signal }),
    ).then(
      (outcome) =>
        note(
          index,
          outcome.kind === "settled"
            ? { replica: index, fate: "settled", value: outcome.value }
            : { replica: index, fate: "expired", escalation: outcome.escalation },
        ),
      (error: unknown) => note(index, { replica: index, fate: "rejected", error }),
    );
  }

  launch(0);
  if (options.hedgeDelayMs === 0) launch(1);
  else timer = setTimeout(() => launch(1), options.hedgeDelayMs);
  return decided;
}

// Stall-injection upstream for hedge legs: real HTTP over an ephemeral
// loopback port. Every hit is recorded with its replica tag and whether
// the serve-side request signal later aborted (client-side abort
// observation, E04 stallServer precedent). Query params clamp into
// range; unknown paths are 404.
export type UpstreamHit = Readonly<{ path: string; replica: string | null; aborted: boolean }>;

export type StallUpstream = Readonly<{
  url: string;
  hits: () => readonly UpstreamHit[];
  stop: () => void;
}>;

function clampedInt(raw: string | null, fallback: number, max: number): number {
  const parsed = raw === null ? Number.NaN : Number(raw);
  if (!Number.isSafeInteger(parsed) || parsed < 0) return fallback;
  return Math.min(parsed, max);
}

export function startStallUpstream(): StallUpstream {
  const hits: { path: string; replica: string | null; aborted: boolean }[] = [];
  const server = Bun.serve({
    hostname: "127.0.0.1",
    port: 0,
    async fetch(request) {
      const url = new URL(request.url);
      const hit = {
        path: url.pathname,
        replica: request.headers.get("x-replica"),
        aborted: false,
      };
      hits.push(hit);
      const signal = (request as unknown as { signal?: AbortSignal }).signal;
      signal?.addEventListener(
        "abort",
        () => {
          hit.aborted = true;
        },
        { once: true },
      );
      if (url.pathname === "/fast") return new Response("fast");
      if (url.pathname === "/slow") {
        await Bun.sleep(clampedInt(url.searchParams.get("ms"), 2000, 10000));
        return new Response("slow");
      }
      if (url.pathname === "/fail") {
        await Bun.sleep(clampedInt(url.searchParams.get("ms"), 0, 10000));
        const code = clampedInt(url.searchParams.get("code"), 500, 599);
        return new Response("fail", { status: code < 400 ? 500 : code });
      }
      return new Response("nope", { status: 404 });
    },
  });
  return {
    url: server.url.href,
    hits: () => Object.freeze([...hits]),
    stop: () => {
      void server.stop(true);
    },
  };
}
