// NT-I03 late-occurrence adapter. Owner-bound operations over one
// adapter-held LateOccurrenceService (late-service.ts, a verbatim K06
// port): deterministic local mechanics, no transport, following the I04
// doubles precedent per the I03 Jev decision (evidence/I03-jev,
// unanimous adapter_held_service). A revoked or foreign owner fails
// test::stale_handle without touching the service. Every service
// rejection maps VERBATIM to late::late_fault{kind, reason}: K06
// rejection kinds are verdicts the capability exists to detect (swap,
// replay, unbound lease, caps), not handle staleness, so collapsing
// any of them into stale_handle would hide the finding from Can rows.
// (The consult briefs sketched an i11-style stale mapping; the K06
// check names prove these kinds are verdicts, so verbatim mapping
// supersedes that gloss. The adapter-held shape is unaffected.)
import { failure, success, type AssertionContext, type Completion } from "../../../completion.ts";
import { array, record } from "../../../data.ts";
import { createDomainRuntime } from "../../../domain.ts";
import { ownerTag, type TestOwner } from "../../owner.ts";
import { LateOccurrenceService } from "./late-service.ts";
import { NativeSchemaError } from "./late-support.ts";

const origin = Object.freeze({
  source: "can:late",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export type LateOccurrenceErrors = Readonly<{
  staleHandle: string;
  lateFault: string;
  some: string;
  none: string;
  selectFacts: string;
  enrollFacts: string;
  gateFacts: string;
  eventFacts: string;
  terminalFacts: string;
  observationFacts: string;
  leaseFacts: string;
  releaseFacts: string;
  reconcileFacts: string;
  outcomeFacts: string;
  counters: string;
}>;

function readStr(value: unknown, what: string): string {
  if (typeof value !== "string") throw new TypeError(`${what} needs a str`);
  return value;
}

function readInt(value: unknown, what: string): number {
  if (typeof value !== "bigint") throw new TypeError(`${what} needs an int`);
  return Number(value);
}

function intArray(values: readonly number[]): unknown {
  return array(values.map((n) => BigInt(n)));
}

function optionalOf(someIdentity: string, noneIdentity: string, value: unknown): unknown {
  if (value === null || value === undefined) return record(noneIdentity, []);
  return record(someIdentity, [["value", value]]);
}

export function createLateOccurrence(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: LateOccurrenceErrors,
  owner: TestOwner,
) {
  const service = new LateOccurrenceService();

  function failStale(handleTag: string): Completion<never> {
    return failure(
      domain.create(
        errors.staleHandle,
        record(errors.staleHandle, [["handle", handleTag]]),
        origin,
      ),
    );
  }

  function failFault(kind: string, reason: string): Completion<never> {
    return failure(
      domain.create(
        errors.lateFault,
        record(errors.lateFault, [
          ["kind", kind],
          ["reason", reason],
        ]),
        origin,
      ),
    );
  }

  function admitOwner(ownerHandle: unknown, what: string): string | Completion<never> {
    const tag = ownerTag(ownerHandle);
    if (tag === null) throw new TypeError(`${what} needs a test::owner handle`);
    const grant = owner.grantOf(ownerHandle);
    if (grant === null) return failStale(tag);
    return grant;
  }

  function attempt<T>(run: () => T): Completion<T> {
    try {
      return success(run());
    } catch (error) {
      if (error instanceof NativeSchemaError) return failFault(error.kind, error.message);
      throw error;
    }
  }

  function selectRecord(facts: { identity: string; joined: boolean }): unknown {
    return record(errors.selectFacts, [
      ["identity", facts.identity],
      ["joined", facts.joined],
    ]);
  }

  function enrollRecord(facts: { participant: string; identity: string }): unknown {
    return record(errors.enrollFacts, [
      ["participant", facts.participant],
      ["identity", facts.identity],
    ]);
  }

  function gateRecord(facts: { participant: string; gate_seq: number; joined: boolean }): unknown {
    return record(errors.gateFacts, [
      ["participant", facts.participant],
      ["gate_seq", BigInt(facts.gate_seq)],
      ["joined", facts.joined],
    ]);
  }

  function eventRecord(facts: {
    participant: string;
    identity: string;
    kind: string;
    seq: number;
    late: boolean;
    dropped: readonly number[];
  }): unknown {
    return record(errors.eventFacts, [
      ["participant", facts.participant],
      ["identity", facts.identity],
      ["kind", facts.kind],
      ["seq", BigInt(facts.seq)],
      ["late", facts.late],
      ["dropped", intArray(facts.dropped)],
    ]);
  }

  function terminalRecord(facts: {
    participant: string;
    identity: string;
    terminal: string;
    binding: string;
  }): unknown {
    return record(errors.terminalFacts, [
      ["participant", facts.participant],
      ["identity", facts.identity],
      ["terminal", facts.terminal],
      ["binding", facts.binding],
    ]);
  }

  function observationRecord(facts: {
    participant: string;
    identity: string;
    kind: string;
    seq: number;
    late: boolean;
    terminal: string | null;
    binding: string | null;
  }): unknown {
    return record(errors.observationFacts, [
      ["participant", facts.participant],
      ["identity", facts.identity],
      ["kind", facts.kind],
      ["seq", BigInt(facts.seq)],
      ["late", facts.late],
      ["terminal", optionalOf(errors.some, errors.none, facts.terminal)],
      ["binding", optionalOf(errors.some, errors.none, facts.binding)],
    ]);
  }

  function leaseRecord(facts: {
    lease: string;
    holder: string;
    identity: string;
    seq: number;
    released: boolean;
  }): unknown {
    return record(errors.leaseFacts, [
      ["lease", facts.lease],
      ["holder", facts.holder],
      ["identity", facts.identity],
      ["seq", BigInt(facts.seq)],
      ["released", facts.released],
    ]);
  }

  function releaseRecord(facts: {
    lease: string;
    holder: string;
    identity: string;
    terminal: string;
    binding: string;
  }): unknown {
    return record(errors.releaseFacts, [
      ["lease", facts.lease],
      ["holder", facts.holder],
      ["identity", facts.identity],
      ["terminal", facts.terminal],
      ["binding", facts.binding],
    ]);
  }

  function reconcileRecord(facts: {
    participant: string;
    admitted: number;
    late: number;
    dropped: readonly number[];
    next_seq: number;
  }): unknown {
    return record(errors.reconcileFacts, [
      ["participant", facts.participant],
      ["admitted", BigInt(facts.admitted)],
      ["late", BigInt(facts.late)],
      ["dropped", intArray(facts.dropped)],
      ["next_seq", BigInt(facts.next_seq)],
    ]);
  }

  function outcomeRecord(facts: {
    outcome: string;
    reason: string;
    worker_terminal: string | null;
    observer_terminal: string | null;
    worker_dead: boolean;
  }): unknown {
    return record(errors.outcomeFacts, [
      ["outcome", facts.outcome],
      ["reason", facts.reason],
      ["worker_terminal", optionalOf(errors.some, errors.none, facts.worker_terminal)],
      ["observer_terminal", optionalOf(errors.some, errors.none, facts.observer_terminal)],
      ["worker_dead", facts.worker_dead],
    ]);
  }

  function countersRecord(facts: {
    participants: number;
    admitted: number;
    late: number;
    dropped: number;
    terminals: number;
    leases: number;
    releases: number;
    observations: number;
    rejected: number;
  }): unknown {
    return record(errors.counters, [
      ["participants", BigInt(facts.participants)],
      ["admitted", BigInt(facts.admitted)],
      ["late", BigInt(facts.late)],
      ["dropped", BigInt(facts.dropped)],
      ["terminals", BigInt(facts.terminals)],
      ["leases", BigInt(facts.leases)],
      ["releases", BigInt(facts.releases)],
      ["observations", BigInt(facts.observations)],
      ["rejected", BigInt(facts.rejected)],
    ]);
  }

  return Object.freeze({
    async select(
      ownerHandle: unknown,
      identity: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "late::select");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(identity, "late::select identity");
      return attempt(() => selectRecord(service.select(id)));
    },

    async enroll(
      ownerHandle: unknown,
      participant: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "late::enroll");
      if (typeof admitted !== "string") return admitted;
      const who = readStr(participant, "late::enroll participant");
      return attempt(() => enrollRecord(service.enroll(who)));
    },

    async armGate(
      ownerHandle: unknown,
      participant: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "late::arm_gate");
      if (typeof admitted !== "string") return admitted;
      const who = readStr(participant, "late::arm_gate participant");
      return attempt(() => gateRecord(service.gate(who)));
    },

    async emit(
      ownerHandle: unknown,
      participant: unknown,
      kind: unknown,
      seq: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "late::emit");
      if (typeof admitted !== "string") return admitted;
      const who = readStr(participant, "late::emit participant");
      const what = readStr(kind, "late::emit kind");
      const at = readInt(seq, "late::emit seq");
      return attempt(() => eventRecord(service.emit(who, what, at)));
    },

    async witnessTerminal(
      ownerHandle: unknown,
      participant: unknown,
      terminal: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "late::witness_terminal");
      if (typeof admitted !== "string") return admitted;
      const who = readStr(participant, "late::witness_terminal participant");
      const tag = readStr(terminal, "late::witness_terminal terminal");
      return attempt(() => terminalRecord(service.terminal(who, tag)));
    },

    async observe(
      ownerHandle: unknown,
      claimant: unknown,
      participant: unknown,
      seq: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "late::observe");
      if (typeof admitted !== "string") return admitted;
      const who = readStr(claimant, "late::observe claimant");
      const whose = readStr(participant, "late::observe participant");
      const at = readInt(seq, "late::observe seq");
      return attempt(() => observationRecord(service.observe(who, whose, at)));
    },

    async grantLease(
      ownerHandle: unknown,
      participant: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "late::grant_lease");
      if (typeof admitted !== "string") return admitted;
      const who = readStr(participant, "late::grant_lease participant");
      return attempt(() => leaseRecord(service.grantLease(who)));
    },

    async observeLease(
      ownerHandle: unknown,
      claimant: unknown,
      lease: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "late::observe_lease");
      if (typeof admitted !== "string") return admitted;
      const who = readStr(claimant, "late::observe_lease claimant");
      const id = readStr(lease, "late::observe_lease lease");
      return attempt(() => leaseRecord(service.observeLease(who, id)));
    },

    async releaseLease(
      ownerHandle: unknown,
      claimant: unknown,
      lease: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "late::release_lease");
      if (typeof admitted !== "string") return admitted;
      const who = readStr(claimant, "late::release_lease claimant");
      const id = readStr(lease, "late::release_lease lease");
      return attempt(() => releaseRecord(service.releaseLease(who, id)));
    },

    async reconcile(
      ownerHandle: unknown,
      participant: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "late::reconcile");
      if (typeof admitted !== "string") return admitted;
      const who = readStr(participant, "late::reconcile participant");
      return attempt(() => reconcileRecord(service.reconcile(who)));
    },

    async killWorker(
      ownerHandle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "late::kill_worker");
      if (typeof admitted !== "string") return admitted;
      return attempt(() => outcomeRecord(service.killWorker()));
    },

    async readOutcome(
      ownerHandle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "late::read_outcome");
      if (typeof admitted !== "string") return admitted;
      return attempt(() => outcomeRecord(service.outcome()));
    },

    async readCounters(
      ownerHandle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "late::read_counters");
      if (typeof admitted !== "string") return admitted;
      return attempt(() => countersRecord(service.counters()));
    },
  });
}
