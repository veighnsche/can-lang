// Cancel-absent operation boundary over the E01 budget/vocabulary (E04).
//
// X-R04-1 is NEGATIVE on every SQL dialect: Query.cancel() flips a client
// flag only while backends run to completion, and cancelling before
// execution never settles. SQL therefore takes the cancel-absent branch:
// the visible boundary returns at the effective bound while the native
// operation stays owned until settlement, reporting the honest
// unknown-write outcome with supervisor escalation. This module is that
// boundary. It is pure TypeScript with no node imports, so browser
// action clients share the validation and race with server adapters;
// the server-only ambient scope holder lives in request-scope.ts.
//
// Timers and signals here hold no lease and revoke none: expiry changes
// nothing about the running operation except what the visible layer
// reports. Escalation is automatic, never caller-invoked: every
// unknown-write return files an escalation record through the sink, and
// the late settlement is observed separately through the same sink. Sinks
// must not throw; the dispatch sink and the module ring never do.
import {
  unknownWrite,
  type RequestBudget,
  type UnknownWrite,
  type UnknownWriteSource,
} from "./request-budget.ts";

export type { UnknownWrite, UnknownWriteSource };

// EscalationCause names what expired the visible boundary. "budget" is
// timer expiry or an already exhausted budget; "disconnect" and
// "shutdown" arrive as the request-scope signal reason; "cancel" is the
// caller's own signal on paths whose native cancel qualified (fetch and
// browser actions — never SQL).
export type EscalationCause = "budget" | "disconnect" | "shutdown" | "cancel";

// EscalationRecord is filed once per unknown-write return: the honest
// outcome, what expired the boundary, and the bounds that raced. seq is
// a module-monotonic link shared with the late-settlement record.
export type EscalationRecord = Readonly<{
  seq: number;
  marker: UnknownWrite;
  cause: EscalationCause;
  boundMs: number | undefined;
  effectiveMs: number;
  atMs: number;
}>;

// LateRecord observes the owned operation settling after the boundary
// already returned. It never rewrites the returned marker. settled
// reports promise settlement only; adapters needing domain detail
// observe the operation themselves. Native causes never enter records.
export type LateRecord = Readonly<{
  seq: number;
  settled: "resolved" | "rejected";
  atMs: number;
}>;

export type EscalationSink = Readonly<{
  escalate(record: EscalationRecord): void;
  late(record: LateRecord): void;
}>;

const MAX_MILLISECONDS = 2147483647;

export function checkOperationBound(boundMs: number): void {
  if (!Number.isSafeInteger(boundMs) || boundMs < 1 || boundMs > MAX_MILLISECONDS)
    throw new TypeError("invalid operation bound");
}

// effectiveBoundMs reports min(bound, remaining budget) floored at zero,
// like the request budget. An absent bound means ambient-only: the
// operation is bounded by the remaining budget alone. Absent bound and
// absent budget is unbounded (Infinity), the legacy path. Zero means
// "return unknown-write without starting", never a zero-length wait.
export function effectiveBoundMs(boundMs?: number, budget?: RequestBudget): number {
  if (boundMs !== undefined) checkOperationBound(boundMs);
  const remaining = budget?.remainingMilliseconds() ?? Number.POSITIVE_INFINITY;
  const bound = boundMs ?? Number.POSITIVE_INFINITY;
  return Math.max(0, Math.min(bound, remaining));
}

export type BoundaryRace = Readonly<{
  source: UnknownWriteSource;
  boundMs?: number;
  budget?: RequestBudget;
  // Request-scope expiry (disconnect/shutdown propagation). The signal
  // reason selects the escalation cause; an unrecognized reason still
  // expires the boundary as a budget outcome.
  scopeSignal?: AbortSignal;
  // Caller cancel. Only paths with a qualified native cancel (fetch and
  // browser actions) may pass one: SQL rejects it structurally, because
  // X-R04-1 admits no cancel operand on any SQL path.
  signal?: AbortSignal;
  sink?: EscalationSink;
}>;

export type BoundaryOutcome<T> =
  | Readonly<{ kind: "settled"; value: T }>
  | Readonly<{ kind: "unknown"; marker: UnknownWrite; escalation: EscalationRecord }>;

let nextSequence = 0;
function sequence(): number {
  nextSequence += 1;
  return nextSequence;
}

// The module ring is the fallback sink when no request scope supplies
// one (scope-less budgeted calls in tests and non-request contexts).
// Bounded: the oldest records drop first and every drop is counted, so
// E06 draining observes loss instead of silently missing it.
const RING_CAP = 256;
const ringEscalations: EscalationRecord[] = [];
const ringLates: LateRecord[] = [];
let ringDropped = 0;
function ringPush<T>(ring: T[], record: T): void {
  ring.push(record);
  if (ring.length > RING_CAP) {
    ring.shift();
    ringDropped += 1;
  }
}
const ringSink: EscalationSink = Object.freeze({
  escalate(record: EscalationRecord): void {
    ringPush(ringEscalations, record);
  },
  late(record: LateRecord): void {
    ringPush(ringLates, record);
  },
});

export function drainBudgetEscalations(): Readonly<{
  escalations: readonly EscalationRecord[];
  lates: readonly LateRecord[];
  dropped: number;
}> {
  const drained = Object.freeze({
    escalations: Object.freeze([...ringEscalations]),
    lates: Object.freeze([...ringLates]),
    dropped: ringDropped,
  });
  ringEscalations.length = 0;
  ringLates.length = 0;
  ringDropped = 0;
  return drained;
}

// createCollectorSink is the request-scope and test sink: live arrays
// the owner drains (dispatch hands them to the E06 reporter).
export function createCollectorSink(): Readonly<{
  sink: EscalationSink;
  escalations: readonly EscalationRecord[];
  lates: readonly LateRecord[];
}> {
  const escalations: EscalationRecord[] = [];
  const lates: LateRecord[] = [];
  return Object.freeze({
    sink: Object.freeze({
      escalate(record: EscalationRecord): void {
        escalations.push(record);
      },
      late(record: LateRecord): void {
        lates.push(record);
      },
    }),
    escalations,
    lates,
  });
}

function scopeCause(signal: AbortSignal): EscalationCause {
  const reason = signal.reason;
  if (reason === "disconnect" || reason === "shutdown") return reason;
  return "budget";
}

function fileUnknown(
  race: BoundaryRace,
  cause: EscalationCause,
  effective: number,
): Extract<BoundaryOutcome<never>, { kind: "unknown" }> {
  const sink = race.sink ?? ringSink;
  const marker = unknownWrite(race.source);
  const escalation = Object.freeze({
    seq: sequence(),
    marker,
    cause,
    boundMs: race.boundMs,
    effectiveMs: effective,
    atMs: performance.now(),
  });
  sink.escalate(escalation);
  return Object.freeze({ kind: "unknown", marker, escalation });
}

// raceBoundary runs start() under the effective bound and returns either
// its value or the honest unknown-write outcome. start is invoked at
// most once, and never when the boundary is already expired: an
// aborted signal or an exhausted budget returns unknown-write without
// starting any native work. On expiry the operation stays owned — the
// race never cancels, settles, or otherwise touches it — while the
// escalation record and the later settlement observation flow to the
// sink automatically. A rejected operation propagates its rejection;
// adapters convert expected native causes before the race sees them.
export async function raceBoundary<T>(
  race: BoundaryRace,
  start: () => Promise<T>,
): Promise<BoundaryOutcome<T>> {
  if (race.source === "sql" && race.signal !== undefined)
    throw new TypeError("sql operations take no cancel signal");
  if (race.signal?.aborted) return fileUnknown(race, "cancel", 0);
  if (race.scopeSignal?.aborted) return fileUnknown(race, scopeCause(race.scopeSignal), 0);
  const effective = effectiveBoundMs(race.boundMs, race.budget);
  if (effective <= 0) return fileUnknown(race, "budget", effective);
  return new Promise<BoundaryOutcome<T>>((resolve, reject) => {
    const sink = race.sink ?? ringSink;
    let done = false;
    const operation = start();
    const finish = (settle: () => void): void => {
      if (done) return;
      done = true;
      if (timer !== undefined) clearTimeout(timer);
      race.scopeSignal?.removeEventListener("abort", onScope);
      race.signal?.removeEventListener("abort", onCancel);
      settle();
    };
    const expire = (cause: EscalationCause): void => {
      if (done) return;
      const marker = unknownWrite(race.source);
      const escalation = Object.freeze({
        seq: sequence(),
        marker,
        cause,
        boundMs: race.boundMs,
        effectiveMs: effective,
        atMs: performance.now(),
      });
      // The operation stays owned: this observer only records its late
      // settlement and never rewrites the returned marker. Both branches
      // are handled, so the still-running work cannot reject unobserved.
      operation.then(
        () => {
          sink.late(
            Object.freeze({ seq: escalation.seq, settled: "resolved", atMs: performance.now() }),
          );
        },
        () => {
          sink.late(
            Object.freeze({ seq: escalation.seq, settled: "rejected", atMs: performance.now() }),
          );
        },
      );
      sink.escalate(escalation);
      finish(() => {
        resolve(Object.freeze({ kind: "unknown", marker, escalation }));
      });
    };
    const onScope = (): void => {
      expire(race.scopeSignal === undefined ? "budget" : scopeCause(race.scopeSignal));
    };
    const onCancel = (): void => {
      expire("cancel");
    };
    const timer =
      effective === Number.POSITIVE_INFINITY
        ? undefined
        : setTimeout(() => {
            expire("budget");
          }, effective);
    race.scopeSignal?.addEventListener("abort", onScope, { once: true });
    race.signal?.addEventListener("abort", onCancel, { once: true });
    operation.then(
      (value) => {
        finish(() => {
          resolve(Object.freeze({ kind: "settled", value }));
        });
      },
      (cause: unknown) => {
        finish(() => {
          reject(cause);
        });
      },
    );
  });
}
