// NT-I14 transaction-observer adapter. Owner-bound operations over
// one adapter-held TransactionObserver
// (db-transaction-service.ts, a verbatim K24 port): deterministic
// local mechanics, no transport, following the I03/I04 doubles
// precedent per the I13 Jev decision (evidence/I13-jev), which
// settled the observer-family merge shape. A revoked or foreign
// owner fails test::stale_handle without touching the service.
// Unlike the K22 core, the K24 observer takes no scenario owner;
// admission is adapter-side only. Every service rejection maps
// VERBATIM to db::db_fault{layer, code}: K24 rejection codes are
// verdicts the capability exists to detect (busy actor, forged
// token, unsettled release, wrong-handle ids), not handle
// staleness. Service messages are dropped at the boundary:
// layer+code is the stable asserted contract; messages embed
// dynamic names. Facts map to nominal records with snake_case
// fields; ints cross as bigint; id cells cross as copies.
import { failure, success, type AssertionContext, type Completion } from "../../../completion.ts";
import { array, dataProperty, record, recordIdentity } from "../../../data.ts";
import { createDomainRuntime } from "../../../domain.ts";
import { copyBytes, isBytes, ownBytes } from "../../../bytes.ts";
import { ownerTag, type TestOwner } from "../../owner.ts";
import {
  checkLimits,
  DbObserverError,
  type DbCell,
  type DbObserverLimits,
} from "../i13/db-core-service.ts";
import {
  TransactionObserver,
  type ActorIdentityFacts,
  type CallbackEntryFacts,
  type IdentityComparison,
  type LastInsertIdFacts,
  type ReleaseAck,
  type SettlementFacts,
} from "./db-transaction-service.ts";

const origin = Object.freeze({
  source: "can:db",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export type DbTransactionErrors = Readonly<{
  staleHandle: string;
  dbFault: string;
  numberCell: string;
  textCell: string;
  bytesCell: string;
  nullCell: string;
  callbackEntryFacts: string;
  callbackEntry: string;
  actorIdentityFacts: string;
  lastInsertIdRecord: string;
  identityComparison: string;
  settlementRecord: string;
  releaseRecord: string;
}>;

function readStr(value: unknown, what: string): string {
  if (typeof value !== "string") throw new TypeError(`${what} needs a str`);
  return value;
}

function readCell(value: unknown, errors: DbTransactionErrors, what: string): DbCell {
  const identity = recordIdentity(value);
  if (identity === errors.numberCell) {
    return { tag: "number", lexeme: readStr(dataProperty(value, "lexeme"), `${what} lexeme`) };
  }
  if (identity === errors.textCell) {
    return { tag: "text", text: readStr(dataProperty(value, "text"), `${what} text`) };
  }
  if (identity === errors.bytesCell) {
    const data = dataProperty(value, "data");
    if (!isBytes(data)) throw new TypeError(`${what} data needs a bytes::buffer`);
    return { tag: "bytes", bytes: copyBytes(data, origin) };
  }
  if (identity === errors.nullCell) return { tag: "null" };
  throw new TypeError(`${what} needs a db::cell`);
}

const ADAPTER_LIMITS: DbObserverLimits = checkLimits({
  maxNamespaces: 4,
  maxConnections: 8,
  maxTables: 8,
  maxRows: 16,
  maxCellsPerRow: 8,
  maxCellBytes: 1024,
});

export function createDbTransaction(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: DbTransactionErrors,
  owner: TestOwner,
) {
  const service = new TransactionObserver(ADAPTER_LIMITS);

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

  function cellRecord(cell: DbCell): unknown {
    switch (cell.tag) {
      case "number":
        return record(errors.numberCell, [["lexeme", cell.lexeme]]);
      case "text":
        return record(errors.textCell, [["text", cell.text]]);
      case "bytes":
        return record(errors.bytesCell, [["data", ownBytes(cell.bytes)]]);
      case "null":
        return record(errors.nullCell, []);
    }
  }

  function entryFactsRecord(facts: CallbackEntryFacts): unknown {
    return record(errors.callbackEntryFacts, [
      ["actor", facts.actor],
      ["connection", facts.connection],
      ["entry_seq", BigInt(facts.entrySeq)],
      ["handle_digest", facts.handleDigest],
    ]);
  }

  function actorRecord(facts: ActorIdentityFacts): unknown {
    return record(errors.actorIdentityFacts, [
      ["actor", facts.actor],
      ["connection", facts.connection],
      ["handle_digest", facts.handleDigest],
      ["entry_seq", BigInt(facts.entrySeq)],
      ["settled", facts.settled],
      ["released", facts.released],
    ]);
  }

  function lastInsertIdRecord(facts: LastInsertIdFacts): unknown {
    return record(errors.lastInsertIdRecord, [
      ["actor", facts.actor],
      ["connection", facts.connection],
      ["handle_digest", facts.handleDigest],
      ["id", cellRecord(facts.id)],
      ["digest", facts.digest],
    ]);
  }

  function comparisonRecord(facts: IdentityComparison): unknown {
    return record(errors.identityComparison, [
      ["match", facts.match],
      ["mismatches", array(facts.mismatches.map((n) => BigInt(n)))],
    ]);
  }

  function settlementRecord(facts: SettlementFacts): unknown {
    return record(errors.settlementRecord, [
      ["actor", facts.actor],
      ["connection", facts.connection],
      ["outcome", facts.outcome],
      ["engine", facts.engine],
      ["digest", facts.digest],
    ]);
  }

  function releaseRecord(facts: ReleaseAck): unknown {
    return record(errors.releaseRecord, [
      ["actor", facts.actor],
      ["connection", facts.connection],
      ["released", facts.released],
      ["ack_digest", facts.ackDigest],
    ]);
  }

  function readClaim(
    value: unknown,
    what: string,
  ): { actor: string; connection: string; id: DbCell } {
    return {
      actor: readStr(dataProperty(value, "actor"), `${what} actor`),
      connection: readStr(dataProperty(value, "connection"), `${what} connection`),
      id: readCell(dataProperty(value, "id"), errors, `${what} id`),
    };
  }

  function readStoredId(value: unknown, what: string): LastInsertIdFacts {
    return {
      actor: readStr(dataProperty(value, "actor"), `${what} actor`),
      connection: readStr(dataProperty(value, "connection"), `${what} connection`),
      handleDigest: readStr(dataProperty(value, "handle_digest"), `${what} handle_digest`),
      id: readCell(dataProperty(value, "id"), errors, `${what} id`),
      digest: readStr(dataProperty(value, "digest"), `${what} digest`),
    };
  }

  return Object.freeze({
    async enterCallback(
      ownerHandle: unknown,
      actor: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::enter_callback");
      if (typeof admitted !== "string") return admitted;
      const name = readStr(actor, "db::enter_callback actor");
      return attempt(() => {
        const entered = service.enterCallback(name);
        return record(errors.callbackEntry, [
          ["facts", entryFactsRecord(entered.facts)],
          ["token", entered.token],
        ]);
      });
    },

    async actorFacts(
      ownerHandle: unknown,
      actor: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::actor_facts");
      if (typeof admitted !== "string") return admitted;
      const name = readStr(actor, "db::actor_facts actor");
      return attempt(() => actorRecord(service.actorFacts(name)));
    },

    async recordLastInsertId(
      ownerHandle: unknown,
      actor: unknown,
      token: unknown,
      id: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::record_last_insert_id");
      if (typeof admitted !== "string") return admitted;
      const name = readStr(actor, "db::record_last_insert_id actor");
      const tok = readStr(token, "db::record_last_insert_id token");
      const cell = readCell(id, errors, "db::record_last_insert_id id");
      return attempt(() => lastInsertIdRecord(service.recordLastInsertId(name, tok, cell)));
    },

    async lastInsertIdFacts(
      ownerHandle: unknown,
      actor: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::last_insert_id_facts");
      if (typeof admitted !== "string") return admitted;
      const name = readStr(actor, "db::last_insert_id_facts actor");
      return attempt(() => lastInsertIdRecord(service.lastInsertIdFacts(name)));
    },

    async compareLastInsertId(
      ownerHandle: unknown,
      stored: unknown,
      claimed: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::compare_last_insert_id");
      if (typeof admitted !== "string") return admitted;
      const have = readStoredId(stored, "db::compare_last_insert_id stored");
      const want = readClaim(claimed, "db::compare_last_insert_id claimed");
      return attempt(() => comparisonRecord(service.compareLastInsertId(have, want)));
    },

    async recordSettlement(
      ownerHandle: unknown,
      actor: unknown,
      token: unknown,
      outcome: unknown,
      engine: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::record_settlement");
      if (typeof admitted !== "string") return admitted;
      const name = readStr(actor, "db::record_settlement actor");
      const tok = readStr(token, "db::record_settlement token");
      const out = readStr(outcome, "db::record_settlement outcome");
      const eng = readStr(engine, "db::record_settlement engine");
      return attempt(() => settlementRecord(service.recordSettlement(name, tok, out, eng)));
    },

    async settlementFacts(
      ownerHandle: unknown,
      actor: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::settlement_facts");
      if (typeof admitted !== "string") return admitted;
      const name = readStr(actor, "db::settlement_facts actor");
      return attempt(() => settlementRecord(service.settlementFacts(name)));
    },

    async release(
      ownerHandle: unknown,
      actor: unknown,
      token: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::release");
      if (typeof admitted !== "string") return admitted;
      const name = readStr(actor, "db::release actor");
      const tok = readStr(token, "db::release token");
      return attempt(() => releaseRecord(service.release(name, tok)));
    },

    async releaseAck(
      ownerHandle: unknown,
      actor: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::release_ack");
      if (typeof admitted !== "string") return admitted;
      const name = readStr(actor, "db::release_ack actor");
      return attempt(() => releaseRecord(service.releaseAck(name)));
    },
  });
}
