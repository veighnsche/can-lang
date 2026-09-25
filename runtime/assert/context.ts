// Compiler-private assertion state travels explicitly through generated calls.
// No process-global active root can leak across independent asynchronous runs.
import {
  assertionFailure,
  standardFailureDiagnostics,
  type FailureOrigin,
  type StandardFailure,
} from "../failure.ts";

import { diagnosticFrames } from "../diagnostics.ts";
import {
  rootIdentity,
  invocationIdentity,
  participantIdentities,
  invocationPath,
  type InvocationIdentity,
  type InvocationPath,
  type CallableIdentity,
} from "./lineage.ts";
import {
  createBarrier,
  reserveFrame,
  startFrame,
  suspendFrame,
  resumeFrame,
  finishFrame,
  abandonFrame,
  fixtureEvent,
  barrierState,
  type Barrier,
  type Frame,
} from "./barrier.ts";
import {
  fixtureQueues,
  allocateFixture,
  closeQueues,
  FixtureQueueError,
  type FixtureQueues,
  type Allocation,
} from "./queue.ts";
import type { Completion } from "../completion.ts";
import { createEvidence, recordEvidence, evidenceReport, type Evidence } from "./report.ts";

export type AssertionRoot = Readonly<{
  package: string;
  declaration: string;
  name: string;
  links?: readonly string[];
}>;
declare const contextBrand: unique symbol;
export type AssertionContext = Readonly<{ readonly [contextBrand]: true }>;
type Violation =
  | "missing fixture"
  | "argument mismatch"
  | "ambiguous fixture"
  | "malformed fixture"
  | "unexpected live boundary"
  | "unused fixture"
  | "outcome mismatch";
type FixturePathDiagnostic = Readonly<{
  reason: Violation;
  expected: Allocation | null;
  actual: InvocationPath;
  origin: FailureOrigin | null;
}>;
type State = {
  owner: object;
  barrier: Barrier;
  queues: FixtureQueues;
  paths: FixturePathDiagnostic[];
  origins: Map<string, FailureOrigin>;
  root: AssertionRoot;
  violations: Violation[];
  failures: StandardFailure[];
  evidence: Evidence;
  closed: boolean;
  tables: Map<string, { used: number; total: number; origin: FailureOrigin }>;
  links: Map<string, boolean>;
  scope: unknown;
};
const linkOrigin: FailureOrigin = Object.freeze({
  source: "can:assertion",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
type View = { shared: State; identity: InvocationIdentity; frame: Frame };
const contexts = new WeakMap<object, View>();
function view(context: AssertionContext): View {
  const found = context !== null && typeof context === "object" ? contexts.get(context) : undefined;
  if (!found) throw new TypeError("invalid assertion context");
  return found;
}
function makeView(shared: State, identity: InvocationIdentity): AssertionContext {
  const value = Object.freeze(Object.create(null));
  contexts.set(value, { shared, identity, frame: reserveFrame(shared.barrier, identity) });
  return value;
}
export function assertionContext(root: AssertionRoot): AssertionContext {
  if (!root.package || !root.declaration || !root.name)
    throw new TypeError("incomplete assertion root");
  const links = root.links ?? [];
  if (
    !Array.isArray(links) ||
    links.some((link) => typeof link !== "string" || !link) ||
    new Set(links).size !== links.length
  )
    throw new TypeError("invalid scenario links");
  const identity = rootIdentity(root),
    barrier = createBarrier(identity);
  const evidence = createEvidence("assertion");
  recordEvidence(evidence, "real-can");
  const context = makeView(
    {
      owner: Object.freeze({}),
      barrier,
      queues: fixtureQueues(identity),
      paths: [],
      origins: new Map(),
      root: Object.freeze(
        root.links === undefined ? { ...root } : { ...root, links: Object.freeze([...links]) },
      ),
      violations: [],
      failures: [],
      evidence,
      closed: false,
      tables: new Map(),
      links: new Map(links.map((link) => [link, false])),
      scope: undefined,
    },
    identity,
  );
  startFrame(view(context).frame);
  return context;
}
// useScenarioLink records that a linked scenario supplied at least one
// row at an entered table. Links that never supply a row fail the root
// as unused fixtures when the context closes.
export function useScenarioLink(context: AssertionContext, scenario: string): void {
  const current = state(context);
  if (current.links.has(scenario)) current.links.set(scenario, true);
}
function state(context: AssertionContext): State {
  return view(context).shared;
}
export function violation(
  context: AssertionContext,
  reason: Violation,
  origin: FailureOrigin,
  expected: Allocation | null = null,
  actual: InvocationPath = invocationPath(view(context).identity),
): StandardFailure {
  const current = state(context),
    failure = assertionFailure(reason, origin);
  // Placeholder platform origins carry no reserving invocation and address
  // no lexical site; reports omit them instead of leaking synthetic markers.
  current.paths.push(
    Object.freeze({ reason, expected, actual, origin: origin.invocation.length ? origin : null }),
  );
  current.violations.push(reason);
  current.failures.push(failure);
  return failure;
}
export function denyLiveBoundary(
  context: AssertionContext | undefined,
  origin: FailureOrigin,
): void {
  if (context === undefined) return;
  const current = state(context);
  const reason = current.closed ? "unexpected live boundary" : "missing fixture";
  throw violation(context, reason, origin);
}
export function suppliedEvidence(context: AssertionContext): void {
  recordEvidence(state(context).evidence, "supplied-completion");
}
// One inert harness token per assertion root stands in for every elided
// ingress scope argument. Readers deny live execution under assertion
// context, so the token only ever meets when-row identity comparison.
// The token carries a private brand so browser operations can fail it
// closed as disposed instead of throwing a resource-state fault: Can
// code cannot forge the brand or name a handle-typed value, so only
// elided harness arguments ever match.
const scopeBrand = Symbol("can.assert.scope");
export function isAssertScope(value: unknown): boolean {
  return (
    value !== null &&
    (typeof value === "object" || typeof value === "function") &&
    (value as Record<symbol, unknown>)[scopeBrand] === true
  );
}
export function scopeRequest(context: AssertionContext | undefined): unknown {
  if (context === undefined) throw new TypeError("harness scope requires an assertion context");
  const current = state(context);
  if (current.scope === undefined) {
    const token: Record<symbol, unknown> = Object.create(null);
    token[scopeBrand] = true;
    current.scope = Object.freeze(token);
  }
  return current.scope;
}
export function rawProviderEvidence(context: AssertionContext): void {
  recordEvidence(state(context).evidence, "raw-provider-fixture");
}
export function policyFixtureEvidence(context: AssertionContext): void {
  recordEvidence(state(context).evidence, "policy-fixture");
}
export function contextReport(context: AssertionContext) {
  const current = state(context);
  const frames = Object.freeze(
    current.failures.flatMap((failure) => {
      const details = standardFailureDiagnostics(failure);
      return diagnosticFrames(details.cause, details.boundaryOrigin ?? details.origin);
    }),
  );
  return Object.freeze({
    root: current.root,
    frames,
    violations: Object.freeze([...current.violations]),
    evidence: evidenceReport(current.evidence),
    fixturePaths: Object.freeze([...current.paths]),
  });
}
export function fixtureIndex(
  context: AssertionContext,
  table: string,
  total: number,
  origin: FailureOrigin,
): number {
  registerFixtureTable(context, table, total, origin);
  const current = state(context),
    queue = current.tables.get(table)!;
  if (queue.used >= queue.total) throw violation(context, "missing fixture", origin);
  return queue.used++;
}
export function registerFixtureTable(
  context: AssertionContext,
  table: string,
  total: number,
  origin: FailureOrigin,
): void {
  const current = state(context);
  if (current.closed) throw violation(context, "unexpected live boundary", origin);
  let queue = current.tables.get(table);
  if (!queue) {
    queue = { used: 0, total, origin };
    current.tables.set(table, queue);
  }
  if (queue.total !== total) throw violation(context, "ambiguous fixture", origin);
}
export function closeContext(context: AssertionContext): void {
  const current = state(context);
  for (const queue of current.tables.values())
    if (queue.used !== queue.total) violation(context, "unused fixture", queue.origin);
  for (const problem of closeQueues(current.queues)) {
    violation(
      context,
      problem.reason,
      current.origins.get(problem.expected.table)!,
      problem.expected,
      problem.actual,
    );
  }
  for (const [link, used] of current.links) {
    if (used) continue;
    const path = invocationPath(view(context).identity);
    violation(
      context,
      "unused fixture",
      linkOrigin,
      Object.freeze({ table: link, row: 0, path }),
      path,
    );
  }
  current.closed = true;
}

// The root frame ends before ownership drain, allowing surviving participants to
// reach and consume their remaining fixtures. Queue closure follows that drain.
export function finishAssertionExecution(context: AssertionContext): void {
  finishFrame(view(context).frame);
}
export function contextIdentity(context: AssertionContext): InvocationIdentity {
  return view(context).identity;
}
export async function callContext<T>(
  context: AssertionContext | undefined,
  site: string,
  run: (child: AssertionContext | undefined) => Promise<T> | T,
  callable?: CallableIdentity,
): Promise<T> {
  if (context === undefined) return run(undefined);
  const parent = view(context),
    child = makeView(parent.shared, invocationIdentity(parent.identity, site, callable));
  suspendFrame(parent.frame);
  startFrame(view(child).frame);
  try {
    return await run(child);
  } finally {
    finishFrame(view(child).frame);
    resumeFrame(parent.frame);
  }
}
export function scheduledFixture(
  context: AssertionContext,
  table: string,
  total: number,
  origin: FailureOrigin,
  run: (allocation: Allocation) => Promise<Completion>,
): Promise<Completion> {
  const current = view(context);
  current.shared.origins.set(table, origin);
  return fixtureEvent(current.frame, async () => {
    try {
      return await run(allocateFixture(current.shared.queues, table, total, current.identity));
    } catch (cause) {
      if (cause instanceof FixtureQueueError) {
        throw violation(context, cause.reason, origin, cause.expected, cause.actual);
      }
      throw cause;
    }
  });
}
export function fixtureMismatch(
  context: AssertionContext,
  allocation: Allocation,
  reason: "argument mismatch" | "malformed fixture" | "outcome mismatch",
  origin: FailureOrigin,
): StandardFailure {
  return violation(context, reason, origin, allocation);
}
export type CoordinationContexts = Readonly<{
  contexts: readonly AssertionContext[];
  start: (index: number) => void;
  observed: (index: number, completion: Completion) => void;
  selected: () => void;
  abort: () => void;
}>;
export function coordinationContexts(
  context: AssertionContext,
  site: string,
  positions: readonly (readonly number[])[],
  mode: "all" | "settled" | "any" | "race",
): CoordinationContexts {
  const parent = view(context),
    children = participantIdentities(parent.identity, site, positions).map((identity) =>
      makeView(parent.shared, identity),
    );
  const started = new Set<number>(),
    observed = new Set<number>();
  let gate: Frame | undefined,
    returned = false;
  // Empty settling aggregates already have a pending native selection reaction.
  // Keep that continuation active before suspending the parent, just as observed
  // does for a nonempty aggregate whose result can now be selected.
  if (children.length === 0 && mode !== "race") {
    gate = reserveFrame(
      parent.shared.barrier,
      invocationIdentity(parent.identity, "can:assertion-selection#0"),
    );
    startFrame(gate);
  }
  suspendFrame(parent.frame);
  function child(index: number): View {
    if (!Number.isSafeInteger(index) || index < 0 || index >= children.length)
      throw new TypeError("invalid participant index");
    return view(children[index]);
  }
  return Object.freeze({
    contexts: Object.freeze(children),
    start(index: number) {
      const value = child(index);
      if (started.has(index)) throw new TypeError("participant already started");
      startFrame(value.frame);
      started.add(index);
    },
    observed(index: number, completion: Completion) {
      const value = child(index);
      if (!started.has(index) || observed.has(index))
        throw new TypeError("invalid participant observation");
      observed.add(index);
      const ready =
        mode === "race" ||
        observed.size === children.length ||
        (mode === "all" && completion.kind !== "ok") ||
        (mode === "any" && completion.kind === "ok");
      if (ready && !returned && gate === undefined) {
        gate = reserveFrame(
          parent.shared.barrier,
          invocationIdentity(parent.identity, "can:assertion-selection#0"),
        );
        startFrame(gate);
      }
      finishFrame(value.frame);
    },
    selected() {
      if (returned) throw new TypeError("coordination already resumed");
      returned = true;
      resumeFrame(parent.frame);
      if (gate) finishFrame(gate);
    },
    abort() {
      if (returned || started.size) throw new TypeError("cannot abandon started coordination");
      for (const value of children) abandonFrame(view(value).frame);
      returned = true;
      resumeFrame(parent.frame);
      if (gate) finishFrame(gate);
    },
  });
}

export function contextOwner(context: AssertionContext): object {
  return state(context).owner;
}
// contextProgress exposes the barrier's structural pending count and frame
// paths/phases for supervisor progress. Paths carry invocation structure
// only, never argument values, captures, or secrets.
export function contextProgress(context: AssertionContext) {
  return barrierState(state(context).barrier);
}
