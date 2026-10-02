// NT-I17 object-store adapter. Owner-bound operations over one
// adapter-held ObjectStoreService (object-store-service.ts, a
// verbatim K27 port): deterministic local mechanics, no
// transport, following the I03/I04 doubles precedent per the
// I13 Jev decision (evidence/I13-jev), which settled the
// observer-family merge shape. A revoked or foreign owner fails
// test::stale_handle without touching the service. The K27
// service takes no scenario owner; admission is adapter-side
// only. The held service runs under the K27 accepted control
// vocabulary (grants owner-a/tenant-a/owned +
// owner-b/tenant-b/owned, the 3-key sharedBucket seed,
// roomyLimits); those fixtures are the observer's own, never
// adapter client state. The surface shape follows
// evidence/I17-jev (new store:: package; verbatim
// session_token_for_test like db::connection_token_for_test).
// Every service rejection maps VERBATIM to
// store::store_fault{layer, code}: K27 rejection codes are
// verdicts the capability exists to detect (ungranted
// prefixes, forged tokens/continuations, escaping keys,
// unsettled reads, incomplete scans), not handle staleness.
// Service messages are dropped at the boundary: layer+code is
// the stable asserted contract; messages embed dynamic names.
// Facts map to nominal records with snake_case fields; ints
// cross as bigint; bytes cross as bytes::buffer copies;
// continuations cross as option::value<str>, none for the
// first page.
import { failure, success, type AssertionContext, type Completion } from "../../../completion.ts";
import { array, dataProperty, record, recordIdentity } from "../../../data.ts";
import { createDomainRuntime } from "../../../domain.ts";
import { copyBytes, isBytes, ownBytes } from "../../../bytes.ts";
import { ownerTag, type TestOwner } from "../../owner.ts";
import {
  checkLimits,
  ObjectStoreError,
  ObjectStoreService,
  type CleanupReceipt,
  type ObjectFacts,
  type ObjectStoreLimits,
  type PageFacts,
  type PendingFacts,
  type PrefixReceipt,
  type SessionFacts,
  type WriteFacts,
} from "./object-store-service.ts";

const origin = Object.freeze({
  source: "can:store",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export type ObjectStoreErrors = Readonly<{
  staleHandle: string;
  storeFault: string;
  optionSome: string;
  optionNone: string;
  prefixReceipt: string;
  sessionFacts: string;
  objectFacts: string;
  writeFacts: string;
  pageFacts: string;
  pendingFacts: string;
  storedObject: string;
  cleanupReceipt: string;
}>;

function readStr(value: unknown, what: string): string {
  if (typeof value !== "string") throw new TypeError(`${what} needs a str`);
  return value;
}

function readInt(value: unknown, what: string): number {
  if (typeof value !== "bigint") throw new TypeError(`${what} needs an int`);
  return Number(value);
}

function readBytes(value: unknown, what: string): Uint8Array {
  if (!isBytes(value)) throw new TypeError(`${what} needs a bytes::buffer`);
  return copyBytes(value, origin);
}

const ADAPTER_LIMITS: ObjectStoreLimits = checkLimits({
  maxPrefixes: 4,
  maxSessions: 8,
  maxObjects: 32,
  maxObjectBytes: 1024,
  maxPageSize: 8,
  maxPendingWrites: 16,
});

const SHARED_SEED: ReadonlyMap<string, Uint8Array> = new Map([
  ["shared/logos/acme.png", new Uint8Array([137, 80, 78, 71])],
  ["shared/empty", new Uint8Array([])],
  ["tenant-b/owned/keep", new Uint8Array([9, 9, 9])],
]);

export function createStore(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: ObjectStoreErrors,
  owner: TestOwner,
) {
  const service = new ObjectStoreService(
    [
      ["owner-a", "tenant-a/owned"],
      ["owner-b", "tenant-b/owned"],
    ],
    SHARED_SEED,
    ADAPTER_LIMITS,
  );

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
        errors.storeFault,
        record(errors.storeFault, [
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
      if (error instanceof ObjectStoreError) return failFault(error.layer, error.code);
      throw error;
    }
  }

  function readContinuation(value: unknown, what: string): string | null {
    const identity = recordIdentity(value);
    if (identity === errors.optionNone) return null;
    if (identity === errors.optionSome) {
      return readStr(dataProperty(value, "value"), `${what} value`);
    }
    throw new TypeError(`${what} needs an option::value<str>`);
  }

  function writeContinuation(token: string | null): unknown {
    if (token === null) return record(errors.optionNone, []);
    return record(errors.optionSome, [["value", token]]);
  }

  function receiptRecord(facts: PrefixReceipt): unknown {
    return record(errors.prefixReceipt, [
      ["prefix", facts.prefix],
      ["owner", facts.owner],
      ["handle", facts.handle],
      ["handle_digest", facts.handleDigest],
      ["objects", BigInt(facts.objects)],
    ]);
  }

  function sessionRecord(facts: SessionFacts): unknown {
    return record(errors.sessionFacts, [
      ["session", facts.session],
      ["prefix", facts.prefix],
      ["owner", facts.owner],
      ["handle_digest", facts.handleDigest],
      ["pinned", facts.pinned],
    ]);
  }

  function objectRecord(facts: ObjectFacts): unknown {
    return record(errors.objectFacts, [
      ["key", facts.key],
      ["size", BigInt(facts.size)],
      ["digest", facts.digest],
    ]);
  }

  function writeRecord(facts: WriteFacts): unknown {
    return record(errors.writeFacts, [
      ["write_id", facts.writeId],
      ["key", facts.key],
      ["bytes_digest", facts.bytesDigest],
      ["settled", facts.settled],
    ]);
  }

  function pageRecord(facts: PageFacts): unknown {
    return record(errors.pageFacts, [
      ["prefix", facts.prefix],
      ["keys", array(facts.keys.map(objectRecord))],
      ["count", BigInt(facts.count)],
      ["complete", facts.complete],
      ["next_continuation", writeContinuation(facts.nextContinuation)],
      ["generation", BigInt(facts.generation)],
      ["digest", facts.digest],
    ]);
  }

  function pendingRecord(facts: PendingFacts): unknown {
    return record(errors.pendingFacts, [
      ["prefix", facts.prefix],
      ["writes", array(facts.writes.map(writeRecord))],
      ["count", BigInt(facts.count)],
    ]);
  }

  function cleanupRecord(facts: CleanupReceipt): unknown {
    return record(errors.cleanupReceipt, [
      ["prefix", facts.prefix],
      ["owner", facts.owner],
      ["handle_digest", facts.handleDigest],
      ["objects", BigInt(facts.objects)],
      ["pending", BigInt(facts.pending)],
      ["generation", BigInt(facts.generation)],
      ["digest", facts.digest],
    ]);
  }

  return Object.freeze({
    async openPrefix(
      ownerHandle: unknown,
      grantOwner: unknown,
      prefix: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "store::open_prefix");
      if (typeof admitted !== "string") return admitted;
      const identity = readStr(grantOwner, "store::open_prefix grant_owner");
      const scope = readStr(prefix, "store::open_prefix prefix");
      return attempt(() => receiptRecord(service.openPrefix(identity, scope)));
    },

    async receipt(
      ownerHandle: unknown,
      grantOwner: unknown,
      prefix: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "store::receipt");
      if (typeof admitted !== "string") return admitted;
      const identity = readStr(grantOwner, "store::receipt grant_owner");
      const scope = readStr(prefix, "store::receipt prefix");
      return attempt(() => receiptRecord(service.receipt(identity, scope)));
    },

    async closePrefix(
      ownerHandle: unknown,
      grantOwner: unknown,
      prefix: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<void>> {
      const admitted = admitOwner(ownerHandle, "store::close_prefix");
      if (typeof admitted !== "string") return admitted;
      const identity = readStr(grantOwner, "store::close_prefix grant_owner");
      const scope = readStr(prefix, "store::close_prefix prefix");
      return attempt(() => {
        service.closePrefix(identity, scope);
      });
    },

    async openSession(
      ownerHandle: unknown,
      grantOwner: unknown,
      prefix: unknown,
      session: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "store::open_session");
      if (typeof admitted !== "string") return admitted;
      const identity = readStr(grantOwner, "store::open_session grant_owner");
      const scope = readStr(prefix, "store::open_session prefix");
      const name = readStr(session, "store::open_session session");
      return attempt(() => sessionRecord(service.openSession(identity, scope, name)));
    },

    async sessionTokenForTest(
      ownerHandle: unknown,
      grantOwner: unknown,
      prefix: unknown,
      session: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "store::session_token_for_test");
      if (typeof admitted !== "string") return admitted;
      const identity = readStr(grantOwner, "store::session_token_for_test grant_owner");
      const scope = readStr(prefix, "store::session_token_for_test prefix");
      const name = readStr(session, "store::session_token_for_test session");
      return attempt(() => service.sessionTokenForTest(identity, scope, name));
    },

    async closeSession(
      ownerHandle: unknown,
      grantOwner: unknown,
      prefix: unknown,
      session: unknown,
      token: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<void>> {
      const admitted = admitOwner(ownerHandle, "store::close_session");
      if (typeof admitted !== "string") return admitted;
      const identity = readStr(grantOwner, "store::close_session grant_owner");
      const scope = readStr(prefix, "store::close_session prefix");
      const name = readStr(session, "store::close_session session");
      const pin = readStr(token, "store::close_session token");
      return attempt(() => {
        service.closeSession(identity, scope, name, pin);
      });
    },

    async put(
      ownerHandle: unknown,
      grantOwner: unknown,
      prefix: unknown,
      session: unknown,
      token: unknown,
      key: unknown,
      payload: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "store::put");
      if (typeof admitted !== "string") return admitted;
      const identity = readStr(grantOwner, "store::put grant_owner");
      const scope = readStr(prefix, "store::put prefix");
      const name = readStr(session, "store::put session");
      const pin = readStr(token, "store::put token");
      const object = readStr(key, "store::put key");
      const bytes = readBytes(payload, "store::put payload");
      return attempt(() => writeRecord(service.put(identity, scope, name, pin, object, bytes)));
    },

    async settleWrite(
      ownerHandle: unknown,
      grantOwner: unknown,
      prefix: unknown,
      session: unknown,
      token: unknown,
      writeId: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "store::settle_write");
      if (typeof admitted !== "string") return admitted;
      const identity = readStr(grantOwner, "store::settle_write grant_owner");
      const scope = readStr(prefix, "store::settle_write prefix");
      const name = readStr(session, "store::settle_write session");
      const pin = readStr(token, "store::settle_write token");
      const id = readStr(writeId, "store::settle_write write_id");
      return attempt(() => objectRecord(service.settleWrite(identity, scope, name, pin, id)));
    },

    async pending(
      ownerHandle: unknown,
      grantOwner: unknown,
      prefix: unknown,
      session: unknown,
      token: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "store::pending");
      if (typeof admitted !== "string") return admitted;
      const identity = readStr(grantOwner, "store::pending grant_owner");
      const scope = readStr(prefix, "store::pending prefix");
      const name = readStr(session, "store::pending session");
      const pin = readStr(token, "store::pending token");
      return attempt(() => pendingRecord(service.pending(identity, scope, name, pin)));
    },

    async get(
      ownerHandle: unknown,
      grantOwner: unknown,
      prefix: unknown,
      session: unknown,
      token: unknown,
      key: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "store::get");
      if (typeof admitted !== "string") return admitted;
      const identity = readStr(grantOwner, "store::get grant_owner");
      const scope = readStr(prefix, "store::get prefix");
      const name = readStr(session, "store::get session");
      const pin = readStr(token, "store::get token");
      const object = readStr(key, "store::get key");
      return attempt(() => {
        const found = service.get(identity, scope, name, pin, object);
        return record(errors.storedObject, [
          ["key", found.key],
          ["bytes", ownBytes(found.bytes)],
          ["size", BigInt(found.size)],
          ["digest", found.digest],
        ]);
      });
    },

    async delete(
      ownerHandle: unknown,
      grantOwner: unknown,
      prefix: unknown,
      session: unknown,
      token: unknown,
      key: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<void>> {
      const admitted = admitOwner(ownerHandle, "store::delete");
      if (typeof admitted !== "string") return admitted;
      const identity = readStr(grantOwner, "store::delete grant_owner");
      const scope = readStr(prefix, "store::delete prefix");
      const name = readStr(session, "store::delete session");
      const pin = readStr(token, "store::delete token");
      const object = readStr(key, "store::delete key");
      return attempt(() => {
        service.delete(identity, scope, name, pin, object);
      });
    },

    async compareBytes(
      ownerHandle: unknown,
      stored: unknown,
      claimed: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "store::compare_bytes");
      if (typeof admitted !== "string") return admitted;
      const have = readBytes(stored, "store::compare_bytes stored");
      const want = readBytes(claimed, "store::compare_bytes claimed");
      return attempt(() => service.compareBytes(have, want));
    },

    async list(
      ownerHandle: unknown,
      grantOwner: unknown,
      prefix: unknown,
      session: unknown,
      token: unknown,
      limit: unknown,
      continuation: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "store::list");
      if (typeof admitted !== "string") return admitted;
      const identity = readStr(grantOwner, "store::list grant_owner");
      const scope = readStr(prefix, "store::list prefix");
      const name = readStr(session, "store::list session");
      const pin = readStr(token, "store::list token");
      const size = readInt(limit, "store::list limit");
      const edge = readContinuation(continuation, "store::list continuation");
      return attempt(() => pageRecord(service.list(identity, scope, name, pin, size, edge)));
    },

    async sealCleanup(
      ownerHandle: unknown,
      grantOwner: unknown,
      prefix: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "store::seal_cleanup");
      if (typeof admitted !== "string") return admitted;
      const identity = readStr(grantOwner, "store::seal_cleanup grant_owner");
      const scope = readStr(prefix, "store::seal_cleanup prefix");
      return attempt(() => cleanupRecord(service.sealCleanup(identity, scope)));
    },

    async sharedDigest(
      ownerHandle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "store::shared_digest");
      if (typeof admitted !== "string") return admitted;
      return attempt(() => service.sharedDigest());
    },
  });
}
