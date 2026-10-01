// NT-I16 behavioral port of tools/runtime/test-services/db-observer/deadline.ts
// (K26, accepted). The runtime ship boundary forbids runtime/ from importing
// tools/, so the I16 slice carries its own copy of the deadline observer.
// Observer logic below is unchanged from K26; only this header and the core
// import (../i13/db-core-service.ts, the NT-I13 port of the K22 core) differ.

// K26: separate SQL deadline from server settlement.
//
// An INDEPENDENT raw observer over in-memory doubles only: no real database,
// no driver, no sockets, no I/O. Over one retained namespace, this module
// records dispatch (at most one per work label), the visible deadline, the
// driver's local settlement, the server acknowledgment, quiescence/fence
// facts, retained lease facts, and engine-scoped cancellation. Every failure
// reuses the K22 core's layered DbObserverError (engine/driver/sql).
//
// Driver/server boundary (normative):
//   Driver settlement is the driver's LOCAL view only: it records what the
//   driver saw when the visible deadline passed (completed or timed-out) and
//   carries an explicit serverEffect:"unresolved" tag. Server effect
//   (applied, absent, or unknown) is proven by server acknowledgment alone.
//   See DEADLINE_DRIVER_SERVER_BOUNDARY.
//
// Cancellation boundary (normative):
//   A cancel request never claims server cancellation unless an engine-
//   specific capable grant backs it. The adapter implements a server-cancel
//   channel for postgres only; sqlite and mysql cannot back a cancellation
//   claim, so acquiring a cancel grant for them refuses, and any cancel
//   request without a capable grant (unknown, invented, or wrong-engine)
//   refuses with forged-token. A recorded cancel proves nothing about the
//   server and never unblocks cleanup. See DEADLINE_CANCEL_BOUNDARY.
//
// Cleanup rule (normative):
//   Release requires driver settlement, a KNOWN server effect (applied or
//   absent), and a fence. An unknown server effect blocks cleanup, as does
//   a missing settlement, acknowledgment, or fence. QD4 owns the D4 verdict
//   and reads these facts; see DEADLINE_QD4_REQUIREMENTS.
//
// Independence scope from the Can adapter under test (normative):
//   SHARES: nothing executable. The observer never imports, calls, or reads
//   adapter code, adapter-decoded values, or adapter connection state. The
//   only strings that may coincide are the work/namespace labels under test,
//   supplied by the test, never by the adapter.
//   PROVES INDEPENDENTLY: single dispatch, visible deadline, driver-local
//   settlement, server effect, quiescence, fence, retained lease, and
//   capable-grant cancellation from its own doubles alone.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, or live runtimes. Local controls only.

import { createHash, randomBytes } from "node:crypto";
import { DbObserverError, type DbObserverLimits } from "../i13/db-core-service.ts";

export const DEADLINE_OBSERVER_SCHEMA_VERSION = "1" as const;

// Recorded driver/server boundary: driver settlement never implies server
// settlement; server effect is proven by server acknowledgment alone.
export const DEADLINE_DRIVER_SERVER_BOUNDARY: string =
  "driver-local: driver settlement records what the driver saw when the visible " +
  "deadline passed and never implies server settlement — server effect is proven " +
  "by server acknowledgment alone, and the D4 verdict belongs to QD4, which reads " +
  "these facts, and nothing here claims execution credit";

// Recorded cancellation boundary: no capable engine grant, no cancel claim.
export const DEADLINE_CANCEL_BOUNDARY: string =
  "capable-grant-only: a cancel request never claims server cancellation unless " +
  "an engine-specific capable grant backs it — cancellation without a capable " +
  "grant refuses, a recorded cancel proves nothing about the server, and no " +
  "generic grant is ever treated as server-cancellation authority";

// What QD4 still needs beyond this module's raw facts.
export const DEADLINE_QD4_REQUIREMENTS: readonly string[] = Object.freeze([
  "visible-deadline",
  "driver-settlement",
  "server-ack",
  "fence",
  "qd4-verdict",
]);

export const DEADLINE_MAX_WORKS = 8;
export const DEADLINE_MAX_CANCEL_GRANTS = 8;

export const DEADLINE_ENGINES = ["postgres", "sqlite", "mysql"] as const;
export type DeadlineEngine = (typeof DEADLINE_ENGINES)[number];

export const DEADLINE_DRIVER_OUTCOMES = ["completed", "timed-out"] as const;
export type DeadlineDriverOutcome = (typeof DEADLINE_DRIVER_OUTCOMES)[number];

export const DEADLINE_SERVER_EFFECTS = ["applied", "absent", "unknown"] as const;
export type DeadlineServerEffect = (typeof DEADLINE_SERVER_EFFECTS)[number];

const MAX_LABEL_LEN = 128;
const MAX_STATEMENT_LEN = 4096;

// Engine-specific cancel capability: the adapter implements a server-cancel
// channel for postgres only. sqlite is in-process (there is no server to
// cancel) and this adapter implements no mysql cancel channel, so neither
// can back a cancellation claim. Capability is a property of the engine
// adapter, never of a generic grant.
const CANCEL_CAPABLE: Readonly<Record<DeadlineEngine, boolean>> = Object.freeze({
  postgres: true,
  sqlite: false,
  mysql: false,
});

export function isCancelCapable(engine: DeadlineEngine): boolean {
  return CANCEL_CAPABLE[engine];
}

function checkLabel(value: unknown, what: string): string {
  if (typeof value !== "string" || value === "" || value.length > MAX_LABEL_LEN) {
    throw new DbObserverError(
      "malformed-statement",
      `${what} must be a non-empty label of at most ${MAX_LABEL_LEN} chars`,
    );
  }
  if (!/^[A-Za-z0-9][A-Za-z0-9_.:-]*$/.test(value)) {
    throw new DbObserverError(
      "malformed-statement",
      `${what} carries illegal characters: ${value}`,
    );
  }
  return value;
}

function checkStatement(value: unknown): string {
  if (typeof value !== "string" || value === "") {
    throw new DbObserverError("malformed-statement", "statement must be a non-empty string");
  }
  if (value.length > MAX_STATEMENT_LEN) {
    throw new DbObserverError("capacity-exhausted", "statement exceeds the length cap");
  }
  return value;
}

function checkEngine(value: unknown): DeadlineEngine {
  if (!(DEADLINE_ENGINES as readonly unknown[]).includes(value)) {
    throw new DbObserverError("malformed-statement", `unknown engine: ${String(value)}`);
  }
  return value as DeadlineEngine;
}

function checkDriverOutcome(value: unknown): DeadlineDriverOutcome {
  if (!(DEADLINE_DRIVER_OUTCOMES as readonly unknown[]).includes(value)) {
    throw new DbObserverError("malformed-statement", `unknown driver outcome: ${String(value)}`);
  }
  return value as DeadlineDriverOutcome;
}

function checkServerEffect(value: unknown): DeadlineServerEffect {
  if (!(DEADLINE_SERVER_EFFECTS as readonly unknown[]).includes(value)) {
    throw new DbObserverError("malformed-statement", `unknown server effect: ${String(value)}`);
  }
  return value as DeadlineServerEffect;
}

function checkDeadlineMs(value: unknown): number {
  if (!Number.isSafeInteger(value) || (value as number) < 0) {
    throw new DbObserverError(
      "malformed-statement",
      "deadline must be a non-negative integer tick",
    );
  }
  return value as number;
}

function digestOf(text: string): string {
  return `sha256:${createHash("sha256").update(text, "utf8").digest("hex")}`;
}

// ---------------------------------------------------------------------------
// Facts
// ---------------------------------------------------------------------------

export type DispatchFacts = Readonly<{
  work: string;
  engine: DeadlineEngine;
  namespace: string;
  statementDigest: string;
  handleDigest: string;
}>;

export type DeadlineFacts = Readonly<{
  work: string;
  engine: DeadlineEngine;
  deadlineMs: number;
  digest: string;
}>;

export type DriverSettlementFacts = Readonly<{
  work: string;
  engine: DeadlineEngine;
  outcome: DeadlineDriverOutcome;
  serverEffect: "unresolved";
  digest: string;
}>;

export type ServerAckFacts = Readonly<{
  work: string;
  engine: DeadlineEngine;
  effect: DeadlineServerEffect;
  digest: string;
}>;

export type CancelGrantFacts = Readonly<{
  engine: DeadlineEngine;
  grantDigest: string;
}>;

// A recorded cancel is inert toward the server: it names the work and the
// capable engine grant behind it and carries a fixed proves tag so no
// reader can mistake it for server-cancellation evidence.
export type CancelFacts = Readonly<{
  work: string;
  engine: DeadlineEngine;
  proves: "nothing-about-server";
  digest: string;
}>;

// A fence surveys; it settles nothing and resolves no release. Cleanup
// still requires driver settlement plus a known server effect.
export type FenceFacts = Readonly<{
  work: string;
  engine: DeadlineEngine;
  settles: "nothing";
  ackDigest: string;
}>;

export type LeaseFacts = Readonly<{
  work: string;
  engine: DeadlineEngine;
  namespace: string;
  handleDigest: string;
  retained: boolean;
}>;

export type DeadlineReleaseAck = Readonly<{
  work: string;
  engine: DeadlineEngine;
  released: true;
  ackDigest: string;
}>;

type WorkEntry = {
  work: string;
  engine: DeadlineEngine;
  token: string;
  statementDigest: string;
  deadlineMs: number | undefined;
  driverOutcome: DeadlineDriverOutcome | undefined;
  serverEffect: DeadlineServerEffect | undefined;
  cancel: CancelFacts | undefined;
  fenced: boolean;
  released: boolean;
};

type CancelGrantEntry = {
  engine: DeadlineEngine;
  grant: string;
};

// ---------------------------------------------------------------------------
// Observer: dispatch, deadline, driver settlement, server ack, fence, lease
// ---------------------------------------------------------------------------

export class DeadlineObserver {
  private readonly owner: string;
  private readonly namespace: string;
  private readonly works = new Map<string, WorkEntry>();
  private readonly cancelGrants = new Map<string, CancelGrantEntry>();
  private readonly quiesced = new Set<DeadlineEngine>();

  constructor(owner: unknown, namespace: unknown, _limits: DbObserverLimits) {
    void _limits;
    this.owner = checkLabel(owner, "owner");
    this.namespace = checkLabel(namespace, "namespace");
  }

  get workCount(): number {
    return this.works.size;
  }

  get liveWorkCount(): number {
    let count = 0;
    for (const entry of this.works.values()) {
      if (!entry.released) count++;
    }
    return count;
  }

  // Dispatch one SQL work item. At most one dispatch per work label: a
  // second dispatch refuses, and dispatch to a quiesced engine refuses.
  // The statement is named by digest only, never executed. The token is
  // minted here and returned alongside the facts exactly once; every later
  // call must present it.
  dispatch(
    work: unknown,
    statement: unknown,
    engine: unknown,
  ): Readonly<{ facts: DispatchFacts; token: string }> {
    const label = checkLabel(work, "work");
    const text = checkStatement(statement);
    const pinned = checkEngine(engine);
    if (this.quiesced.has(pinned)) {
      throw new DbObserverError("namespace-closed", `engine quiesced: ${pinned}`);
    }
    if (this.works.has(label)) {
      throw new DbObserverError("connection-busy", `work already dispatched: ${label}`);
    }
    if (this.works.size >= DEADLINE_MAX_WORKS) {
      throw new DbObserverError("capacity-exhausted", "work table full");
    }
    const token = `deadline-${randomBytes(16).toString("hex")}`;
    const entry: WorkEntry = {
      work: label,
      engine: pinned,
      token,
      statementDigest: digestOf(`stmt|${text}`),
      deadlineMs: undefined,
      driverOutcome: undefined,
      serverEffect: undefined,
      cancel: undefined,
      fenced: false,
      released: false,
    };
    this.works.set(label, entry);
    return Object.freeze({
      facts: this.dispatchFactsOf(entry),
      token,
    });
  }

  private dispatchFactsOf(entry: WorkEntry): DispatchFacts {
    return Object.freeze({
      work: entry.work,
      engine: entry.engine,
      namespace: this.namespace,
      statementDigest: entry.statementDigest,
      handleDigest: digestOf(entry.token),
    });
  }

  dispatchFacts(work: unknown): DispatchFacts {
    return this.dispatchFactsOf(this.requireEntry(checkLabel(work, "work")));
  }

  private requireEntry(work: string): WorkEntry {
    const entry = this.works.get(work);
    if (entry === undefined) {
      throw new DbObserverError("unknown-connection", `work never dispatched: ${work}`);
    }
    return entry;
  }

  private requireToken(entry: WorkEntry, token: string): void {
    if (token === "" || token !== entry.token) {
      throw new DbObserverError("forged-token", "work token is not the pinned token");
    }
  }

  private requireLive(entry: WorkEntry): void {
    if (entry.released) {
      throw new DbObserverError("closed-handle", "handle already released");
    }
  }

  // Observe the visible deadline for one dispatched work: the driver-
  // visible tick, supplied by the test (no timers run here). Single
  // record: observing twice refuses. Driver settlement requires the
  // deadline first, so the deadline stays its own fact.
  observeDeadline(work: unknown, token: string, deadlineMs: unknown): DeadlineFacts {
    const entry = this.requireEntry(checkLabel(work, "work"));
    this.requireToken(entry, token);
    this.requireLive(entry);
    if (entry.deadlineMs !== undefined) {
      throw new DbObserverError("connection-busy", "deadline already observed");
    }
    entry.deadlineMs = checkDeadlineMs(deadlineMs);
    return this.deadlineFacts(entry.work);
  }

  deadlineFacts(work: unknown): DeadlineFacts {
    const entry = this.requireEntry(checkLabel(work, "work"));
    if (entry.deadlineMs === undefined) {
      throw new DbObserverError("no-conversation", `no deadline observed for ${entry.work}`);
    }
    return Object.freeze({
      work: entry.work,
      engine: entry.engine,
      deadlineMs: entry.deadlineMs,
      digest: digestOf(`deadline|${entry.work}|${entry.engine}|${entry.deadlineMs}`),
    });
  }

  // Record the driver's local settlement: what the driver saw when the
  // visible deadline passed. Facts only, driver-local: the fact carries
  // serverEffect:"unresolved", so it can never be read as server
  // settlement. Single record.
  recordDriverSettlement(work: unknown, token: string, outcome: unknown): DriverSettlementFacts {
    const entry = this.requireEntry(checkLabel(work, "work"));
    this.requireToken(entry, token);
    this.requireLive(entry);
    if (entry.deadlineMs === undefined) {
      throw new DbObserverError("no-conversation", "observe the visible deadline first");
    }
    if (entry.driverOutcome !== undefined) {
      throw new DbObserverError("connection-busy", "driver already settled");
    }
    entry.driverOutcome = checkDriverOutcome(outcome);
    return this.driverFacts(entry.work);
  }

  driverFacts(work: unknown): DriverSettlementFacts {
    const entry = this.requireEntry(checkLabel(work, "work"));
    if (entry.driverOutcome === undefined) {
      throw new DbObserverError("no-conversation", `no driver settlement for ${entry.work}`);
    }
    return Object.freeze({
      work: entry.work,
      engine: entry.engine,
      outcome: entry.driverOutcome,
      serverEffect: "unresolved" as const,
      digest: digestOf(`driver|${entry.work}|${entry.engine}|${entry.driverOutcome}`),
    });
  }

  // Record the server acknowledgment: the reconciled server effect. The
  // acknowledgment requires driver settlement first (reconciliation runs
  // after the driver view settles) and is single-record. "unknown" is an
  // explicit recorded effect, not a gap — and it blocks cleanup.
  recordServerAck(work: unknown, token: string, effect: unknown): ServerAckFacts {
    const entry = this.requireEntry(checkLabel(work, "work"));
    this.requireToken(entry, token);
    this.requireLive(entry);
    if (entry.driverOutcome === undefined) {
      throw new DbObserverError("no-conversation", "record driver settlement first");
    }
    if (entry.serverEffect !== undefined) {
      throw new DbObserverError("connection-busy", "server already acknowledged");
    }
    entry.serverEffect = checkServerEffect(effect);
    return this.serverFacts(entry.work);
  }

  serverFacts(work: unknown): ServerAckFacts {
    const entry = this.requireEntry(checkLabel(work, "work"));
    if (entry.serverEffect === undefined) {
      throw new DbObserverError("no-conversation", `no server acknowledgment for ${entry.work}`);
    }
    return Object.freeze({
      work: entry.work,
      engine: entry.engine,
      effect: entry.serverEffect,
      digest: digestOf(`server|${entry.work}|${entry.engine}|${entry.serverEffect}`),
    });
  }

  // Mint one engine-scoped cancel grant. Only cancel-capable engines can
  // back a cancellation claim: acquiring a grant for any other engine
  // refuses rather than emitting an unsupported claim. The grant is opaque;
  // facts carry its digest only.
  acquireCancelGrant(engine: unknown): Readonly<{ facts: CancelGrantFacts; grant: string }> {
    const pinned = checkEngine(engine);
    if (!CANCEL_CAPABLE[pinned]) {
      throw new DbObserverError(
        "malformed-statement",
        `engine has no server-cancel channel: ${pinned}`,
      );
    }
    if (this.cancelGrants.size >= DEADLINE_MAX_CANCEL_GRANTS) {
      throw new DbObserverError("capacity-exhausted", "cancel-grant table full");
    }
    const grant = `cancel-${randomBytes(16).toString("hex")}`;
    this.cancelGrants.set(grant, { engine: pinned, grant });
    return Object.freeze({
      facts: Object.freeze({ engine: pinned, grantDigest: digestOf(grant) }),
      grant,
    });
  }

  // Request cancellation of one dispatched work under a capable grant. The
  // grant must be this observer's own live grant for the work's engine:
  // unknown, invented, or wrong-engine grants refuse with forged-token, and
  // a generic handle is never cancellation authority. A recorded cancel
  // proves nothing about the server, never changes the server effect, and
  // never unblocks cleanup; once the server has acknowledged, a cancel is
  // stale and refuses.
  requestCancel(work: unknown, token: string, grant: string): CancelFacts {
    const entry = this.requireEntry(checkLabel(work, "work"));
    this.requireToken(entry, token);
    this.requireLive(entry);
    const held = this.cancelGrants.get(grant);
    if (held === undefined) {
      throw new DbObserverError("forged-token", "cancel grant is unknown");
    }
    if (held.engine !== entry.engine) {
      throw new DbObserverError("forged-token", "cancel grant is for another engine");
    }
    if (entry.serverEffect !== undefined) {
      throw new DbObserverError("connection-busy", "server already acknowledged; cancel is stale");
    }
    if (entry.cancel !== undefined) {
      throw new DbObserverError("connection-busy", "cancel already requested");
    }
    const facts: CancelFacts = Object.freeze({
      work: entry.work,
      engine: entry.engine,
      proves: "nothing-about-server" as const,
      digest: digestOf(`cancel|${entry.work}|${entry.engine}`),
    });
    entry.cancel = facts;
    return facts;
  }

  cancelFacts(work: unknown): CancelFacts {
    const entry = this.requireEntry(checkLabel(work, "work"));
    if (entry.cancel === undefined) {
      throw new DbObserverError("no-conversation", `no cancel requested for ${entry.work}`);
    }
    return entry.cancel;
  }

  // Quiesce one engine: stop new dispatches and report the live work set.
  // Idempotent: repeats return the same live set without further effect.
  quiesceEngine(engine: unknown): readonly string[] {
    const pinned = checkEngine(engine);
    this.quiesced.add(pinned);
    const live: string[] = [];
    for (const entry of this.works.values()) {
      if (!entry.released && entry.engine === pinned) {
        live.push(entry.work);
      }
    }
    live.sort();
    return Object.freeze(live);
  }

  // Fence one work under its quiesced engine. Fencing requires quiescence
  // first; repeats join the retained acknowledgment. The fence surveys —
  // it is recorded even with an unknown server effect — and settles
  // nothing: cleanup still requires a known server effect plus release.
  fence(work: unknown, token: string): FenceFacts {
    const entry = this.requireEntry(checkLabel(work, "work"));
    this.requireToken(entry, token);
    this.requireLive(entry);
    if (!this.quiesced.has(entry.engine)) {
      throw new DbObserverError("no-conversation", "quiesce the engine before fencing");
    }
    entry.fenced = true;
    return this.fenceFacts(entry.work);
  }

  fenceFacts(work: unknown): FenceFacts {
    const entry = this.requireEntry(checkLabel(work, "work"));
    if (!entry.fenced) {
      throw new DbObserverError("no-conversation", `no fence recorded for ${entry.work}`);
    }
    return Object.freeze({
      work: entry.work,
      engine: entry.engine,
      settles: "nothing" as const,
      ackDigest: digestOf(`fence|${entry.work}|${entry.engine}`),
    });
  }

  // The retained namespace lease: the namespace stays retained for the
  // work until explicit release. Readable at any time; after release the
  // lease shows released.
  leaseFacts(work: unknown): LeaseFacts {
    const entry = this.requireEntry(checkLabel(work, "work"));
    return Object.freeze({
      work: entry.work,
      engine: entry.engine,
      namespace: this.namespace,
      handleDigest: digestOf(entry.token),
      retained: !entry.released,
    });
  }

  // Release one work after full reconciliation. Release requires driver
  // settlement, a KNOWN server effect, and a fence: an unknown server
  // effect blocks cleanup, as does anything missing on the path.
  release(work: unknown, token: string): DeadlineReleaseAck {
    const entry = this.requireEntry(checkLabel(work, "work"));
    this.requireToken(entry, token);
    if (entry.released) {
      throw new DbObserverError("closed-handle", "handle already released");
    }
    if (entry.driverOutcome === undefined) {
      throw new DbObserverError("connection-busy", "record driver settlement before release");
    }
    if (entry.serverEffect === undefined) {
      throw new DbObserverError("no-conversation", "no server acknowledgment recorded");
    }
    if (entry.serverEffect === "unknown") {
      throw new DbObserverError("connection-busy", "server effect unknown blocks cleanup");
    }
    if (!entry.fenced) {
      throw new DbObserverError("no-conversation", "fence before release");
    }
    entry.released = true;
    return this.releaseAck(entry.work);
  }

  releaseAck(work: unknown): DeadlineReleaseAck {
    const entry = this.requireEntry(checkLabel(work, "work"));
    if (!entry.released) {
      throw new DbObserverError("no-conversation", `no release recorded for ${entry.work}`);
    }
    return Object.freeze({
      work: entry.work,
      engine: entry.engine,
      released: true as const,
      ackDigest: digestOf(`release|${entry.work}|${entry.engine}`),
    });
  }
}
