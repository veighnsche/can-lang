// NT-I03 behavioral port of tools/runtime/test-services/native-values/late.ts
// (K06, accepted). The runtime ship boundary forbids runtime/ from importing
// tools/, so the I03 slice carries its own copy of the late-occurrence
// service. Service logic below is unchanged from K06; only this header and
// the support import (./late-support.ts, itself a port of the K01
// checkFactsInert closure) differ.

// K06: late native occurrence and lease facts.
//
// Two gated participants (worker + observer) behind one selected identity.
// Events admitted after a participant's gate are late; every emission binds
// the selected identity plus the emitting participant, so an occurrence swap
// (one participant claiming another's event) is detectable via the identity
// binding. Sequence/count reconciliation exposes dropped late events, lease
// facts observed before the witnessed terminal event reject as unbound
// (mirroring the K05 eager-read rule), and worker death yields a terminal
// incomplete outcome: never a pass, never silent.
//
// Vocabulary mirrors the P09 workspace lease owner (holder-bound leases,
// release facts) and the P11 recovery owner (gated admission, terminal drain
// facts, incomplete-never-silent outcomes), re-expressed here as inert
// local mechanics.
//
// Returned facts are inert counter/identity/sequence data only and pass
// checkFactsInert. Pure in-memory mechanics for the bounded self-check: no
// I/O, timers, transports, services, or live runtimes. Local controls only.

import { createHash } from "node:crypto";
import { checkFactsInert, NativeSchemaError } from "./late-support.ts";

export const NATIVE_LATE_SCHEMA_VERSION = "1" as const;

// ---------------------------------------------------------------------------
// Closed vocabularies
// ---------------------------------------------------------------------------

// The two gated participants behind the selected identity.
export const LATE_PARTICIPANTS = ["worker", "observer"] as const;
export type LateParticipant = (typeof LATE_PARTICIPANTS)[number];

// Reviewed identities; exactly one is selected per service.
export const LATE_IDENTITIES = ["late.alpha", "late.beta"] as const;
export type LateIdentity = (typeof LATE_IDENTITIES)[number];

// Late occurrence kinds: a fault installed late, a late handle use, or the
// witnessed terminal event that binds lease facts.
export const LATE_EVENT_KINDS = ["fault", "use", "terminal"] as const;
export type LateEventKind = (typeof LATE_EVENT_KINDS)[number];

// Terminal completion tags for one participant (mirrors the K05 witness).
export const LATE_TERMINALS = ["completed", "rejected", "failed"] as const;
export type LateTerminal = (typeof LATE_TERMINALS)[number];

// Terminal run outcomes. "complete" needs a witnessed terminal event from
// both participants; anything else, including worker death, is "incomplete"
// with a named reason, never silent.
export const LATE_OUTCOMES = ["complete", "incomplete"] as const;
export type LateOutcome = (typeof LATE_OUTCOMES)[number];

function isParticipant(value: unknown): value is LateParticipant {
  return typeof value === "string" && (LATE_PARTICIPANTS as readonly string[]).includes(value);
}

function isIdentity(value: unknown): value is LateIdentity {
  return typeof value === "string" && (LATE_IDENTITIES as readonly string[]).includes(value);
}

function isEventKind(value: unknown): value is LateEventKind {
  return typeof value === "string" && (LATE_EVENT_KINDS as readonly string[]).includes(value);
}

function isTerminal(value: unknown): value is LateTerminal {
  return typeof value === "string" && (LATE_TERMINALS as readonly string[]).includes(value);
}

// ---------------------------------------------------------------------------
// Bounded caps and id shapes
// ---------------------------------------------------------------------------

export const MAX_LATE_EVENTS = 16;
export const MAX_LATE_LEASES = 8;

const LEASE_PATTERN = /^llease[1-9][0-9]*$/;

// ---------------------------------------------------------------------------
// Binding digest (local, deterministic; independent of the K03 seal)
// ---------------------------------------------------------------------------

function canonicalize(value: unknown): string {
  if (value === null) return "null";
  if (typeof value === "number" || typeof value === "string" || typeof value === "boolean") {
    return JSON.stringify(value) as string;
  }
  if (Array.isArray(value)) {
    return `[${value.map((item) => canonicalize(item)).join(",")}]`;
  }
  if (typeof value === "object") {
    const record = value as Record<string, unknown>;
    const keys = Object.keys(record).sort();
    const body = keys.map((key) => `${JSON.stringify(key)}:${canonicalize(record[key])}`).join(",");
    return `{${body}}`;
  }
  throw new NativeSchemaError("failed", "native-io", "late input carries a non-inert value");
}

export function digestLateBinding(value: unknown): string {
  return `sha256:${createHash("sha256").update(canonicalize(value), "utf8").digest("hex")}`;
}

// ---------------------------------------------------------------------------
// Fact shapes (all inert data)
// ---------------------------------------------------------------------------

export type LateSelectFacts = Readonly<{
  identity: LateIdentity;
  joined: boolean;
}>;

export type LateEnrollFacts = Readonly<{
  participant: LateParticipant;
  identity: LateIdentity;
}>;

export type LateGateFacts = Readonly<{
  participant: LateParticipant;
  gate_seq: number;
  joined: boolean;
}>;

export type LateEventFacts = Readonly<{
  participant: LateParticipant;
  identity: LateIdentity;
  kind: LateEventKind;
  seq: number;
  late: boolean;
  dropped: readonly number[];
}>;

export type LateTerminalFacts = Readonly<{
  participant: LateParticipant;
  identity: LateIdentity;
  terminal: LateTerminal;
  binding: string;
}>;

export type LateObservationFacts = Readonly<{
  participant: LateParticipant;
  identity: LateIdentity;
  kind: LateEventKind;
  seq: number;
  late: boolean;
  terminal: LateTerminal | null;
  binding: string | null;
}>;

export type LateLeaseFacts = Readonly<{
  lease: string;
  holder: LateParticipant;
  identity: LateIdentity;
  seq: number;
  released: boolean;
}>;

export type LateReleaseFacts = Readonly<{
  lease: string;
  holder: LateParticipant;
  identity: LateIdentity;
  terminal: LateTerminal;
  binding: string;
}>;

export type LateReconcileFacts = Readonly<{
  participant: LateParticipant;
  admitted: number;
  late: number;
  dropped: readonly number[];
  next_seq: number;
}>;

export type LateOutcomeFacts = Readonly<{
  outcome: LateOutcome;
  reason: string;
  worker_terminal: LateTerminal | null;
  observer_terminal: LateTerminal | null;
  worker_dead: boolean;
}>;

export type LateCounters = Readonly<{
  participants: number;
  admitted: number;
  late: number;
  dropped: number;
  terminals: number;
  leases: number;
  releases: number;
  observations: number;
  rejected: number;
}>;

// ---------------------------------------------------------------------------
// Service: gated late occurrences and terminal-bound leases
// ---------------------------------------------------------------------------

type EventRecord = {
  seq: number;
  kind: LateEventKind;
  late: boolean;
};

type ParticipantState = {
  enrolled: boolean;
  gateSeq: number | null;
  nextSeq: number;
  events: EventRecord[];
  dropped: number[];
  terminal: LateTerminal | null;
  binding: string | null;
};

type LeaseRecord = {
  leaseId: string;
  holder: LateParticipant;
  seq: number;
  released: boolean;
};

export class LateOccurrenceService {
  private identity: LateIdentity | null = null;
  private workerDead = false;
  private readonly states: Record<LateParticipant, ParticipantState> = {
    worker: LateOccurrenceService.freshState(),
    observer: LateOccurrenceService.freshState(),
  };
  private readonly leases = new Map<string, LeaseRecord>();
  private leaseCounter = 0;
  private admittedCount = 0;
  private lateCount = 0;
  private observationCount = 0;
  private releaseCount = 0;
  private rejectedCount = 0;

  private static freshState(): ParticipantState {
    return {
      enrolled: false,
      gateSeq: null,
      nextSeq: 1,
      events: [],
      dropped: [],
      terminal: null,
      binding: null,
    };
  }

  counters(): LateCounters {
    let dropped = 0;
    let terminals = 0;
    for (const participant of LATE_PARTICIPANTS) {
      dropped += this.states[participant].dropped.length;
      if (this.states[participant].terminal !== null) terminals += 1;
    }
    return Object.freeze({
      participants: (this.states.worker.enrolled ? 1 : 0) + (this.states.observer.enrolled ? 1 : 0),
      admitted: this.admittedCount,
      late: this.lateCount,
      dropped,
      terminals,
      leases: this.leases.size,
      releases: this.releaseCount,
      observations: this.observationCount,
      rejected: this.rejectedCount,
    });
  }

  // Select the one identity both participants run behind. Selecting the
  // same identity joins; selecting a different one rejects, so a run can
  // never straddle two identities.
  select(identity: unknown): LateSelectFacts {
    if (!isIdentity(identity)) {
      throw this.reject("unsupported-capability", `unknown late identity: ${String(identity)}`);
    }
    if (this.identity !== null && this.identity !== identity) {
      throw this.reject("changed-input", "late identity is already selected");
    }
    const joined = this.identity !== null;
    this.identity = identity;
    const facts = Object.freeze({ identity, joined });
    checkFactsInert(facts, "late select facts");
    return facts;
  }

  // Enroll one gated participant behind the selected identity.
  enroll(participant: unknown): LateEnrollFacts {
    const selected = this.requireIdentity();
    if (!isParticipant(participant)) {
      throw this.reject(
        "unsupported-capability",
        `unknown late participant: ${String(participant)}`,
      );
    }
    const state = this.states[participant];
    if (participant === "worker" && this.workerDead) {
      throw this.reject("permission", "worker is dead; enrollment is closed");
    }
    state.enrolled = true;
    const facts = Object.freeze({ participant, identity: selected });
    checkFactsInert(facts, "late enroll facts");
    return facts;
  }

  // Arm one participant's gate at its next sequence number. Admissions at
  // or after the gate sequence are late. Repeats join the first gate.
  gate(participant: unknown): LateGateFacts {
    if (!isParticipant(participant)) {
      throw this.reject(
        "unsupported-capability",
        `unknown late participant: ${String(participant)}`,
      );
    }
    const state = this.requireEnrolled(participant);
    if (state.gateSeq !== null) {
      const joined = Object.freeze({
        participant,
        gate_seq: state.gateSeq,
        joined: true,
      });
      checkFactsInert(joined, "late gate facts");
      return joined;
    }
    state.gateSeq = state.nextSeq;
    const facts = Object.freeze({ participant, gate_seq: state.gateSeq, joined: false });
    checkFactsInert(facts, "late gate facts");
    return facts;
  }

  // Admit one occurrence. The sequence must be exactly next, or ahead with
  // the skipped numbers recorded as dropped; a replayed sequence rejects.
  // Admission closes once the participant's terminal event is witnessed, so
  // the terminal binding always covers every admitted occurrence. Every
  // admission binds the selected identity plus the emitting participant.
  emit(participant: unknown, kind: unknown, seq: unknown): LateEventFacts {
    const selected = this.requireIdentity();
    if (!isParticipant(participant)) {
      throw this.reject(
        "unsupported-capability",
        `unknown late participant: ${String(participant)}`,
      );
    }
    if (!isEventKind(kind)) {
      throw this.reject("unsupported-capability", `unknown late event kind: ${String(kind)}`);
    }
    if (!Number.isSafeInteger(seq) || (seq as number) <= 0) {
      throw this.reject("invalid-request", "late event sequence must be a positive integer");
    }
    const wanted = seq as number;
    const state = this.requireEnrolled(participant);
    this.requireLiveWorker(participant);
    if (state.terminal !== null) {
      throw this.reject("permission", "participant terminated; admission is closed");
    }
    if (wanted < state.nextSeq) {
      throw this.reject("stale-handle", "late event sequence was already admitted");
    }
    if (this.admittedCount >= MAX_LATE_EVENTS) {
      throw this.reject("resource-limit", "late event cap reached");
    }
    // A gap wider than the service can ever admit is meaningless and would
    // let one call allocate an unbounded dropped list.
    if (wanted - state.nextSeq > MAX_LATE_EVENTS) {
      throw this.reject("resource-limit", "late event gap exceeds the admission bound");
    }
    const dropped: number[] = [];
    for (let missing = state.nextSeq; missing < wanted; missing += 1) {
      dropped.push(missing);
      state.dropped.push(missing);
    }
    const late = state.gateSeq !== null && wanted >= state.gateSeq;
    state.events.push({ seq: wanted, kind, late });
    state.nextSeq = wanted + 1;
    this.admittedCount += 1;
    if (late) this.lateCount += 1;
    const facts = Object.freeze({
      participant,
      identity: selected,
      kind,
      seq: wanted,
      late,
      dropped: Object.freeze([...dropped]),
    });
    checkFactsInert(facts, "late event facts");
    return facts;
  }

  // Record the witnessed terminal event for one participant. The binding
  // digest commits identity, participant, terminal tag, and admitted
  // counts; a second terminal for the same participant rejects as a replay.
  terminal(participant: unknown, tag: unknown): LateTerminalFacts {
    const selected = this.requireIdentity();
    if (!isParticipant(participant)) {
      throw this.reject(
        "unsupported-capability",
        `unknown late participant: ${String(participant)}`,
      );
    }
    if (!isTerminal(tag)) {
      throw this.reject("unsupported-capability", `unknown late terminal tag: ${String(tag)}`);
    }
    const state = this.requireEnrolled(participant);
    this.requireLiveWorker(participant);
    if (state.terminal !== null) {
      throw this.reject("stale-handle", "participant already has a witnessed terminal event");
    }
    state.terminal = tag;
    const binding = digestLateBinding({
      schema_version: NATIVE_LATE_SCHEMA_VERSION,
      identity: selected,
      participant,
      terminal: tag,
      admitted: state.events.length,
      dropped: state.dropped,
    });
    state.binding = binding;
    const facts = Object.freeze({ participant, identity: selected, terminal: tag, binding });
    checkFactsInert(facts, "late terminal facts");
    return facts;
  }

  // Observe one admitted occurrence. The claimant must be the participant
  // the event was admitted under: a swapped claim (observer naming a
  // worker event, or vice versa) rejects on the identity binding.
  observe(claimant: unknown, participant: unknown, seq: unknown): LateObservationFacts {
    const selected = this.requireIdentity();
    if (!isParticipant(claimant)) {
      throw this.reject("unsupported-capability", `unknown late claimant: ${String(claimant)}`);
    }
    if (!isParticipant(participant)) {
      throw this.reject(
        "unsupported-capability",
        `unknown late participant: ${String(participant)}`,
      );
    }
    if (claimant !== participant) {
      throw this.reject("wrong-owner", "occurrence is bound to another participant; swap detected");
    }
    const state = this.requireEnrolled(participant);
    const event = state.events.find((item) => item.seq === seq);
    if (event === undefined) {
      throw this.reject("stale-handle", "no admitted occurrence at this sequence");
    }
    this.observationCount += 1;
    const facts = Object.freeze({
      participant,
      identity: selected,
      kind: event.kind,
      seq: event.seq,
      late: event.late,
      terminal: state.terminal,
      binding: state.binding,
    });
    checkFactsInert(facts, "late observation facts");
    return facts;
  }

  // Grant a holder-bound lease to one participant (P09 vocabulary: the
  // lease names its holder and the selected identity).
  grantLease(participant: unknown): LateLeaseFacts {
    const selected = this.requireIdentity();
    if (!isParticipant(participant)) {
      throw this.reject(
        "unsupported-capability",
        `unknown late participant: ${String(participant)}`,
      );
    }
    this.requireEnrolled(participant);
    this.requireLiveWorker(participant);
    if (this.leases.size >= MAX_LATE_LEASES) {
      throw this.reject("resource-limit", "late lease cap reached");
    }
    this.leaseCounter += 1;
    const leaseId = `llease${this.leaseCounter}`;
    this.leases.set(leaseId, {
      leaseId,
      holder: participant,
      seq: this.leaseCounter,
      released: false,
    });
    const facts = Object.freeze({
      lease: leaseId,
      holder: participant,
      identity: selected,
      seq: this.leaseCounter,
      released: false,
    });
    checkFactsInert(facts, "late lease facts");
    return facts;
  }

  // Read the facts bound to one lease. The holder's terminal event must be
  // witnessed first: pre-terminal reads reject as unbound, mirroring the
  // K05 eager-read rule. A claimant other than the holder rejects on the
  // identity binding.
  observeLease(claimant: unknown, leaseId: unknown): LateLeaseFacts {
    const selected = this.requireIdentity();
    if (!isParticipant(claimant)) {
      throw this.reject("unsupported-capability", `unknown late claimant: ${String(claimant)}`);
    }
    const lease = this.requireLease(leaseId);
    if (claimant !== lease.holder) {
      throw this.reject("wrong-owner", "lease is bound to another participant; swap detected");
    }
    const state = this.requireEnrolled(lease.holder);
    if (state.terminal === null) {
      throw this.reject("permission", "no witnessed terminal event; lease facts are unbound");
    }
    this.observationCount += 1;
    const facts = Object.freeze({
      lease: lease.leaseId,
      holder: lease.holder,
      identity: selected,
      seq: lease.seq,
      released: lease.released,
    });
    checkFactsInert(facts, "late lease facts");
    return facts;
  }

  // Release one lease exactly once. Releasing before the holder's terminal
  // event is witnessed is an early release and rejects as unbound; repeats
  // join the first release with the same binding and no additional count.
  releaseLease(claimant: unknown, leaseId: unknown): LateReleaseFacts {
    const selected = this.requireIdentity();
    if (!isParticipant(claimant)) {
      throw this.reject("unsupported-capability", `unknown late claimant: ${String(claimant)}`);
    }
    const lease = this.requireLease(leaseId);
    if (claimant !== lease.holder) {
      throw this.reject("wrong-owner", "lease is bound to another participant; swap detected");
    }
    const state = this.requireEnrolled(lease.holder);
    if (state.terminal === null || state.binding === null) {
      throw this.reject("permission", "early release; no witnessed terminal event binds it");
    }
    if (!lease.released) {
      lease.released = true;
      this.releaseCount += 1;
    }
    const facts = Object.freeze({
      lease: lease.leaseId,
      holder: lease.holder,
      identity: selected,
      terminal: state.terminal,
      binding: state.binding,
    });
    checkFactsInert(facts, "late release facts");
    return facts;
  }

  // Reconcile admitted vs expected sequences for one participant. A
  // non-empty dropped list proves a dropped late event; it stays reported
  // on every later reconciliation.
  reconcile(participant: unknown): LateReconcileFacts {
    if (!isParticipant(participant)) {
      throw this.reject(
        "unsupported-capability",
        `unknown late participant: ${String(participant)}`,
      );
    }
    const state = this.requireEnrolled(participant);
    const facts = Object.freeze({
      participant,
      admitted: state.events.length,
      late: state.events.filter((item) => item.late).length,
      dropped: Object.freeze([...state.dropped]),
      next_seq: state.nextSeq,
    });
    checkFactsInert(facts, "late reconcile facts");
    return facts;
  }

  // Mark the worker dead. Admissions, terminals, and leases for the worker
  // close from here on; the run outcome below stays incomplete with the
  // worker-dead reason.
  killWorker(): LateOutcomeFacts {
    this.workerDead = true;
    return this.outcome();
  }

  // Terminal run outcome (P11 vocabulary: complete only on a full terminal
  // proof, otherwise incomplete with a named reason, never silent and
  // never a pass on partial facts).
  outcome(): LateOutcomeFacts {
    const workerTerminal = this.states.worker.terminal;
    const observerTerminal = this.states.observer.terminal;
    let outcome: LateOutcome = "complete";
    let reason = "both terminals witnessed";
    if (this.workerDead) {
      outcome = "incomplete";
      reason = "worker-dead";
    } else if (workerTerminal === null || observerTerminal === null) {
      outcome = "incomplete";
      reason = "terminal proof incomplete";
    }
    const facts = Object.freeze({
      outcome,
      reason,
      worker_terminal: workerTerminal,
      observer_terminal: observerTerminal,
      worker_dead: this.workerDead,
    });
    checkFactsInert(facts, "late outcome facts");
    return facts;
  }

  private requireIdentity(): LateIdentity {
    if (this.identity === null) {
      throw this.reject("invalid-request", "select a late identity first");
    }
    return this.identity;
  }

  private requireEnrolled(participant: LateParticipant): ParticipantState {
    const state = this.states[participant];
    if (!state.enrolled) {
      throw this.reject("invalid-request", `${participant} is not enrolled`);
    }
    return state;
  }

  private requireLiveWorker(participant: LateParticipant): void {
    if (participant === "worker" && this.workerDead) {
      throw this.reject("permission", "worker is dead; admission is closed");
    }
  }

  private requireLease(leaseId: unknown): LeaseRecord {
    if (typeof leaseId !== "string" || !LEASE_PATTERN.test(leaseId)) {
      throw this.reject("not-found", "no such late lease");
    }
    const lease = this.leases.get(leaseId);
    if (lease === undefined) {
      throw this.reject("not-found", "no such late lease");
    }
    return lease;
  }

  private reject(
    kind:
      | "invalid-request"
      | "unsupported-capability"
      | "changed-input"
      | "wrong-owner"
      | "not-found"
      | "permission"
      | "stale-handle"
      | "resource-limit",
    message: string,
  ): NativeSchemaError {
    this.rejectedCount += 1;
    return new NativeSchemaError("rejected", kind, message);
  }
}
