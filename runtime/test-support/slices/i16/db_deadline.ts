// NT-I16 deadline-observer adapter. Owner-bound operations over one
// adapter-held DeadlineObserver (db-deadline-service.ts, a verbatim
// K26 port): deterministic local mechanics, no transport,
// following the I03/I04 doubles precedent per the I13 Jev
// decision (evidence/I13-jev), which settled the observer-family
// merge shape. A revoked or foreign owner fails
// test::stale_handle without touching the service. The K26
// observer takes no scenario owner; admission is adapter-side
// only. The held observer runs under the K26 accepted control
// vocabulary (owner-n / ns-alpha, deadline-check.ts); those
// labels are the observer's own, never adapter connection state.
// Every service rejection maps VERBATIM to
// db::db_fault{layer, code}: K26 rejection codes are verdicts
// the capability exists to detect (double dispatch, missing
// deadline, unsettled reads, forged grants, unknown-effect
// cleanup blocks), not handle staleness. Service messages are
// dropped at the boundary: layer+code is the stable asserted
// contract; messages embed dynamic names. Facts map to nominal
// records with snake_case fields; deadline_ms crosses as bigint;
// tokens and grants cross as opaque strings exactly once.
import { failure, success, type AssertionContext, type Completion } from "../../../completion.ts";
import { array, record } from "../../../data.ts";
import { createDomainRuntime } from "../../../domain.ts";
import { ownerTag, type TestOwner } from "../../owner.ts";
import { checkLimits, DbObserverError, type DbObserverLimits } from "../i13/db-core-service.ts";
import {
  DeadlineObserver,
  type CancelFacts,
  type CancelGrantFacts,
  type DeadlineFacts,
  type DeadlineReleaseAck,
  type DispatchFacts,
  type DriverSettlementFacts,
  type FenceFacts,
  type LeaseFacts,
  type ServerAckFacts,
} from "./db-deadline-service.ts";

const origin = Object.freeze({
  source: "can:db",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export type DbDeadlineErrors = Readonly<{
  staleHandle: string;
  dbFault: string;
  dispatchedWork: string;
  dispatchRecord: string;
  deadlineRecord: string;
  driverSettlementFacts: string;
  serverAckFacts: string;
  cancelGrant: string;
  cancelGrantFacts: string;
  cancelRecord: string;
  fenceRecord: string;
  leaseRecord: string;
  deadlineReleaseRecord: string;
}>;

function readStr(value: unknown, what: string): string {
  if (typeof value !== "string") throw new TypeError(`${what} needs a str`);
  return value;
}

function readInt(value: unknown, what: string): number {
  if (typeof value !== "bigint") throw new TypeError(`${what} needs an int`);
  return Number(value);
}

const ADAPTER_LIMITS: DbObserverLimits = checkLimits({
  maxNamespaces: 4,
  maxConnections: 8,
  maxTables: 8,
  maxRows: 16,
  maxCellsPerRow: 8,
  maxCellBytes: 1024,
});

export function createDbDeadline(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: DbDeadlineErrors,
  owner: TestOwner,
) {
  const service = new DeadlineObserver("owner-n", "ns-alpha", ADAPTER_LIMITS);

  function failStale(handleTag: string): Completion<never> {
    return failure(
      domain.create(
        errors.staleHandle,
        record(errors.staleHandle, [["handle", handleTag]]),
        origin,
      ),
    );
  }

  function failFault(layer: string, code: string): Completion<never> {
    return failure(
      domain.create(
        errors.dbFault,
        record(errors.dbFault, [
          ["layer", layer],
          ["code", code],
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
      if (error instanceof DbObserverError) return failFault(error.layer, error.code);
      throw error;
    }
  }

  function dispatchRecord(facts: DispatchFacts): unknown {
    return record(errors.dispatchRecord, [
      ["work", facts.work],
      ["engine", facts.engine],
      ["namespace", facts.namespace],
      ["statement_digest", facts.statementDigest],
      ["handle_digest", facts.handleDigest],
    ]);
  }

  function deadlineRecord(facts: DeadlineFacts): unknown {
    return record(errors.deadlineRecord, [
      ["work", facts.work],
      ["engine", facts.engine],
      ["deadline_ms", BigInt(facts.deadlineMs)],
      ["digest", facts.digest],
    ]);
  }

  function driverRecord(facts: DriverSettlementFacts): unknown {
    return record(errors.driverSettlementFacts, [
      ["work", facts.work],
      ["engine", facts.engine],
      ["outcome", facts.outcome],
      ["server_effect", facts.serverEffect],
      ["digest", facts.digest],
    ]);
  }

  function serverRecord(facts: ServerAckFacts): unknown {
    return record(errors.serverAckFacts, [
      ["work", facts.work],
      ["engine", facts.engine],
      ["effect", facts.effect],
      ["digest", facts.digest],
    ]);
  }

  function grantFactsRecord(facts: CancelGrantFacts): unknown {
    return record(errors.cancelGrantFacts, [
      ["engine", facts.engine],
      ["grant_digest", facts.grantDigest],
    ]);
  }

  function cancelRecord(facts: CancelFacts): unknown {
    return record(errors.cancelRecord, [
      ["work", facts.work],
      ["engine", facts.engine],
      ["proves", facts.proves],
      ["digest", facts.digest],
    ]);
  }

  function fenceRecord(facts: FenceFacts): unknown {
    return record(errors.fenceRecord, [
      ["work", facts.work],
      ["engine", facts.engine],
      ["settles", facts.settles],
      ["ack_digest", facts.ackDigest],
    ]);
  }

  function leaseRecord(facts: LeaseFacts): unknown {
    return record(errors.leaseRecord, [
      ["work", facts.work],
      ["engine", facts.engine],
      ["namespace", facts.namespace],
      ["handle_digest", facts.handleDigest],
      ["retained", facts.retained],
    ]);
  }

  function releaseRecord(facts: DeadlineReleaseAck): unknown {
    return record(errors.deadlineReleaseRecord, [
      ["work", facts.work],
      ["engine", facts.engine],
      ["released", facts.released],
      ["ack_digest", facts.ackDigest],
    ]);
  }

  return Object.freeze({
    async dispatch(
      ownerHandle: unknown,
      work: unknown,
      statement: unknown,
      engine: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::dispatch");
      if (typeof admitted !== "string") return admitted;
      const label = readStr(work, "db::dispatch work");
      const text = readStr(statement, "db::dispatch statement");
      const pinned = readStr(engine, "db::dispatch engine");
      return attempt(() => {
        const out = service.dispatch(label, text, pinned);
        return record(errors.dispatchedWork, [
          ["facts", dispatchRecord(out.facts)],
          ["token", out.token],
        ]);
      });
    },

    async dispatchFacts(
      ownerHandle: unknown,
      work: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::dispatch_facts");
      if (typeof admitted !== "string") return admitted;
      const label = readStr(work, "db::dispatch_facts work");
      return attempt(() => dispatchRecord(service.dispatchFacts(label)));
    },

    async observeDeadline(
      ownerHandle: unknown,
      work: unknown,
      token: unknown,
      deadlineMs: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::observe_deadline");
      if (typeof admitted !== "string") return admitted;
      const label = readStr(work, "db::observe_deadline work");
      const pin = readStr(token, "db::observe_deadline token");
      const tick = readInt(deadlineMs, "db::observe_deadline deadline_ms");
      return attempt(() => deadlineRecord(service.observeDeadline(label, pin, tick)));
    },

    async deadlineFacts(
      ownerHandle: unknown,
      work: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::deadline_facts");
      if (typeof admitted !== "string") return admitted;
      const label = readStr(work, "db::deadline_facts work");
      return attempt(() => deadlineRecord(service.deadlineFacts(label)));
    },

    async recordDriverSettlement(
      ownerHandle: unknown,
      work: unknown,
      token: unknown,
      outcome: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::record_driver_settlement");
      if (typeof admitted !== "string") return admitted;
      const label = readStr(work, "db::record_driver_settlement work");
      const pin = readStr(token, "db::record_driver_settlement token");
      const seen = readStr(outcome, "db::record_driver_settlement outcome");
      return attempt(() => driverRecord(service.recordDriverSettlement(label, pin, seen)));
    },

    async driverFacts(
      ownerHandle: unknown,
      work: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::driver_facts");
      if (typeof admitted !== "string") return admitted;
      const label = readStr(work, "db::driver_facts work");
      return attempt(() => driverRecord(service.driverFacts(label)));
    },

    async recordServerAck(
      ownerHandle: unknown,
      work: unknown,
      token: unknown,
      effect: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::record_server_ack");
      if (typeof admitted !== "string") return admitted;
      const label = readStr(work, "db::record_server_ack work");
      const pin = readStr(token, "db::record_server_ack token");
      const proven = readStr(effect, "db::record_server_ack effect");
      return attempt(() => serverRecord(service.recordServerAck(label, pin, proven)));
    },

    async serverFacts(
      ownerHandle: unknown,
      work: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::server_facts");
      if (typeof admitted !== "string") return admitted;
      const label = readStr(work, "db::server_facts work");
      return attempt(() => serverRecord(service.serverFacts(label)));
    },

    async acquireCancelGrant(
      ownerHandle: unknown,
      engine: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::acquire_cancel_grant");
      if (typeof admitted !== "string") return admitted;
      const pinned = readStr(engine, "db::acquire_cancel_grant engine");
      return attempt(() => {
        const out = service.acquireCancelGrant(pinned);
        return record(errors.cancelGrant, [
          ["facts", grantFactsRecord(out.facts)],
          ["grant", out.grant],
        ]);
      });
    },

    async requestCancel(
      ownerHandle: unknown,
      work: unknown,
      token: unknown,
      grant: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::request_cancel");
      if (typeof admitted !== "string") return admitted;
      const label = readStr(work, "db::request_cancel work");
      const pin = readStr(token, "db::request_cancel token");
      const capable = readStr(grant, "db::request_cancel grant");
      return attempt(() => cancelRecord(service.requestCancel(label, pin, capable)));
    },

    async cancelFacts(
      ownerHandle: unknown,
      work: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::cancel_facts");
      if (typeof admitted !== "string") return admitted;
      const label = readStr(work, "db::cancel_facts work");
      return attempt(() => cancelRecord(service.cancelFacts(label)));
    },

    async quiesceEngine(
      ownerHandle: unknown,
      engine: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::quiesce_engine");
      if (typeof admitted !== "string") return admitted;
      const pinned = readStr(engine, "db::quiesce_engine engine");
      return attempt(() => array([...service.quiesceEngine(pinned)]));
    },

    async fence(
      ownerHandle: unknown,
      work: unknown,
      token: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::fence");
      if (typeof admitted !== "string") return admitted;
      const label = readStr(work, "db::fence work");
      const pin = readStr(token, "db::fence token");
      return attempt(() => fenceRecord(service.fence(label, pin)));
    },

    async fenceFacts(
      ownerHandle: unknown,
      work: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::fence_facts");
      if (typeof admitted !== "string") return admitted;
      const label = readStr(work, "db::fence_facts work");
      return attempt(() => fenceRecord(service.fenceFacts(label)));
    },

    async leaseFacts(
      ownerHandle: unknown,
      work: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::lease_facts");
      if (typeof admitted !== "string") return admitted;
      const label = readStr(work, "db::lease_facts work");
      return attempt(() => leaseRecord(service.leaseFacts(label)));
    },

    async deadlineRelease(
      ownerHandle: unknown,
      work: unknown,
      token: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::deadline_release");
      if (typeof admitted !== "string") return admitted;
      const label = readStr(work, "db::deadline_release work");
      const pin = readStr(token, "db::deadline_release token");
      return attempt(() => releaseRecord(service.release(label, pin)));
    },

    async deadlineReleaseAck(
      ownerHandle: unknown,
      work: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::deadline_release_ack");
      if (typeof admitted !== "string") return admitted;
      const label = readStr(work, "db::deadline_release_ack work");
      return attempt(() => releaseRecord(service.releaseAck(label)));
    },
  });
}
