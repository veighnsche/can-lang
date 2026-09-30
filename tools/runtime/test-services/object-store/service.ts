// K27: owned object-store observations.
//
// An INDEPENDENT observer over a scoped object prefix, over in-memory
// doubles only: no bucket, no network, no SDK, no I/O. The observer binds
// to an owned prefix grant at open, pins identity-bound sessions, stores
// byte-exact objects, lists through explicit opaque continuations, and
// settles accepted writes explicitly. Every failure names its layer
// (engine/driver/store).
//
// Independence scope from the Can adapter under test (normative):
//   SHARES: nothing executable. The observer never imports, calls, or reads
//   adapter code, adapter-decoded values, or adapter client state. The only
//   strings that may coincide are the bucket/prefix names under test
//   (supplied by the test, not by the adapter) and the P29 grant vocabulary
//   the Go mirror uses for receipts.
//   PROVES INDEPENDENTLY: prefix admission (caller prefix is not authority),
//   session pinning, byte-exact objects, paginated listing shape with
//   explicit completeness, accepted-vs-settled write settlement, cleanup
//   receipts, and error provenance (every failure names its layer). Adapter
//   claims are compared AGAINST these facts; the observer never derives a
//   fact FROM an adapter claim.
//   Consequently an incomplete page or an eventually empty list can never
//   prove cleanup: only a terminal listing at the current generation, with
//   zero live objects, zero pending writes, and zero live sessions, seals.
//
// Split with the Go mirror (tools/native-test-owner/external/object_store.go):
//   TS service owns typed exact bytes, pagination mechanics, write
//   settlement, and the bounded self-check. The Go mirror owns P29-journaled
//   prefix discipline (grants, receipts, cleanup order) using the registry's
//   own vocabulary. Both enforce the same contract: owned-prefix admission,
//   pinned identity-bound sessions, opaque tokens and continuations,
//   digest-only facts, and layered errors. Neither reads the Can adapter.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, or live runtimes. Local controls only.

import { createHash, randomBytes } from "node:crypto";

export const OBJECT_STORE_SCHEMA_VERSION = "1" as const;

// ---------------------------------------------------------------------------
// Independence scope (normative, mirrored in object_store.go)
// ---------------------------------------------------------------------------

export const OBJECT_STORE_INDEPENDENCE_SCOPE: string =
  "independent: shares no adapter code, decoded values, or client state; " +
  "proves prefix receipt, pinned sessions, exact bytes, paginated listing " +
  "completeness, write settlement, and layered error provenance from its " +
  "own seeded doubles alone";

// ---------------------------------------------------------------------------
// Layers, codes, and the layered error
// ---------------------------------------------------------------------------

export const OBJECT_STORE_LAYERS = ["engine", "driver", "store"] as const;
export type ObjectStoreLayer = (typeof OBJECT_STORE_LAYERS)[number];

// Closed failure-code vocabulary. Every code maps to exactly one layer:
//   engine: prefix admission, capacity, ownership, lifecycle, cleanup seal.
//   driver: session pinning, tokens, continuation discipline.
//   store:  keys, bytes, page shape, write settlement.
export const OBJECT_STORE_CODES = [
  "unknown-prefix",
  "prefix-closed",
  "capacity-exhausted",
  "wrong-owner",
  "closed-handle",
  "keys-remain",
  "pending-writes",
  "scan-incomplete",
  "session-live",
  "unknown-session",
  "forged-token",
  "forged-continuation",
  "stale-continuation",
  "unknown-key",
  "key-escapes-prefix",
  "malformed-key",
  "write-unknown",
  "write-busy",
  "write-settled",
  "bad-page",
] as const;
export type ObjectStoreCode = (typeof OBJECT_STORE_CODES)[number];

const CODE_LAYER: Readonly<Record<ObjectStoreCode, ObjectStoreLayer>> = {
  "unknown-prefix": "engine",
  "prefix-closed": "engine",
  "capacity-exhausted": "engine",
  "wrong-owner": "engine",
  "closed-handle": "engine",
  "keys-remain": "engine",
  "pending-writes": "engine",
  "scan-incomplete": "engine",
  "session-live": "engine",
  "unknown-session": "driver",
  "forged-token": "driver",
  "forged-continuation": "driver",
  "stale-continuation": "driver",
  "unknown-key": "store",
  "key-escapes-prefix": "store",
  "malformed-key": "store",
  "write-unknown": "store",
  "write-busy": "store",
  "write-settled": "store",
  "bad-page": "store",
};

export function layerOfCode(code: ObjectStoreCode): ObjectStoreLayer {
  return CODE_LAYER[code];
}

export class ObjectStoreError extends Error {
  readonly layer: ObjectStoreLayer;
  readonly code: ObjectStoreCode;
  constructor(code: ObjectStoreCode, message: string) {
    super(`[${CODE_LAYER[code]}:${code}] ${message}`);
    this.name = "ObjectStoreError";
    this.layer = CODE_LAYER[code];
    this.code = code;
  }
}

function fail(code: ObjectStoreCode, message: string): never {
  throw new ObjectStoreError(code, message);
}

// ---------------------------------------------------------------------------
// Bounds and names
// ---------------------------------------------------------------------------

export type ObjectStoreLimits = Readonly<{
  maxPrefixes: number;
  maxSessions: number;
  maxObjects: number;
  maxObjectBytes: number;
  maxPageSize: number;
  maxPendingWrites: number;
}>;

const LIMIT_KEYS = [
  "maxPrefixes",
  "maxSessions",
  "maxObjects",
  "maxObjectBytes",
  "maxPageSize",
  "maxPendingWrites",
] as const;

export function checkLimits(value: unknown): ObjectStoreLimits {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new ObjectStoreError("bad-page", "limits must be an object");
  }
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    if (!(LIMIT_KEYS as readonly string[]).includes(key)) {
      throw new ObjectStoreError("bad-page", `unknown limit field: ${key}`);
    }
  }
  const out: Record<string, number> = {};
  for (const key of LIMIT_KEYS) {
    const entry = record[key];
    if (!Number.isSafeInteger(entry) || (entry as number) <= 0) {
      throw new ObjectStoreError("bad-page", `${key} must be a positive integer`);
    }
    out[key] = entry as number;
  }
  return Object.freeze(out) as ObjectStoreLimits;
}

const MAX_NAME_LEN = 128;
const MAX_KEY_LEN = 512;

function checkOwner(value: unknown): string {
  if (typeof value !== "string" || value === "" || value.length > MAX_NAME_LEN) {
    fail("wrong-owner", `owner must be a non-empty name of at most ${MAX_NAME_LEN} chars`);
  }
  if (!/^[A-Za-z0-9][A-Za-z0-9_.:-]*$/.test(value)) {
    fail("wrong-owner", `owner carries illegal characters: ${value}`);
  }
  return value;
}

function checkDeclaredPrefix(value: unknown): string {
  if (typeof value !== "string" || value === "" || value.length > MAX_KEY_LEN) {
    fail("unknown-prefix", "prefix must be a non-empty path");
  }
  if (value.startsWith("/") || value.endsWith("/") || value.includes("//")) {
    fail("unknown-prefix", `prefix is not a clean relative path: ${value}`);
  }
  for (const segment of value.split("/")) {
    if (segment === "" || segment === "." || segment === "..") {
      fail("unknown-prefix", `prefix carries an empty or dot segment: ${value}`);
    }
    if (!/^[A-Za-z0-9][A-Za-z0-9_.:-]*$/.test(segment)) {
      fail("unknown-prefix", `prefix carries illegal characters: ${value}`);
    }
  }
  return value;
}

function checkSessionName(value: unknown): string {
  if (typeof value !== "string" || value === "" || value.length > MAX_NAME_LEN) {
    fail("bad-page", `session must be a non-empty name of at most ${MAX_NAME_LEN} chars`);
  }
  if (!/^[A-Za-z0-9][A-Za-z0-9_.:-]*$/.test(value)) {
    fail("bad-page", `session carries illegal characters: ${value}`);
  }
  return value;
}

// A caller key is well-formed only as a clean relative path. Well-formedness
// is checked before scope: a malformed key rejects at the store layer even
// when it would also escape.
function checkKeyShape(key: unknown): string {
  if (typeof key !== "string" || key === "" || key.length > MAX_KEY_LEN) {
    fail("malformed-key", "key must be a non-empty path");
  }
  if (key.startsWith("/") || key.endsWith("/") || key.includes("//")) {
    fail("malformed-key", `key is not a clean relative path: ${key}`);
  }
  for (const segment of key.split("/")) {
    if (segment === "" || segment === "." || segment === "..") {
      fail("malformed-key", `key carries an empty or dot segment: ${key}`);
    }
  }
  return key;
}

// ---------------------------------------------------------------------------
// Facts
// ---------------------------------------------------------------------------

export type PrefixReceipt = Readonly<{
  prefix: string;
  owner: string;
  handle: string;
  handleDigest: string;
  objects: number;
}>;

export type SessionFacts = Readonly<{
  session: string;
  prefix: string;
  owner: string;
  handleDigest: string;
  pinned: boolean;
}>;

export type ObjectFacts = Readonly<{
  key: string;
  size: number;
  digest: string;
}>;

export type WriteFacts = Readonly<{
  writeId: string;
  key: string;
  bytesDigest: string;
  settled: boolean;
}>;

export type PageFacts = Readonly<{
  prefix: string;
  keys: readonly ObjectFacts[];
  count: number;
  // Explicit completeness: only a page with complete === true observed the
  // terminal position of the listing. A page with complete === false proves
  // nothing about keys beyond its edge, and no page alone proves cleanup.
  complete: boolean;
  nextContinuation: string | null;
  generation: number;
  digest: string;
}>;

export type PendingFacts = Readonly<{
  prefix: string;
  writes: readonly WriteFacts[];
  count: number;
}>;

export type CleanupReceipt = Readonly<{
  prefix: string;
  owner: string;
  handleDigest: string;
  objects: number;
  pending: number;
  generation: number;
  digest: string;
}>;

type PendingWrite = {
  writeId: string;
  key: string;
  bytes: Uint8Array;
  settled: boolean;
};

type ContinuationRecord = {
  token: string;
  // Lexicographic edge: the next listing resumes strictly after this key.
  after: string;
  generation: number;
  consumed: boolean;
};

type PrefixRecord = {
  prefix: string;
  owner: string;
  handle: string;
  token: string;
  closed: boolean;
  objects: Map<string, Uint8Array>;
  pending: Map<string, PendingWrite>;
  continuations: Map<string, ContinuationRecord>;
  writeSeq: number;
  // Bumped on every accepted write, settlement, and delete. A terminal scan
  // records the generation it observed; the seal requires a terminal scan
  // at the current generation so no mutation can hide behind an old page.
  generation: number;
  sealedGeneration: number;
  sealedEmpty: boolean;
  handleSeq: number;
};

type SessionRecord = {
  session: string;
  prefix: string;
  owner: string;
  token: string;
  pinned: boolean;
  closed: boolean;
};

function mintToken(prefix: string): string {
  return `${prefix}-${randomBytes(16).toString("hex")}`;
}

function digestToken(token: string): string {
  return `sha256:${createHash("sha256").update(token, "utf8").digest("hex")}`;
}

function digestBytes(bytes: Uint8Array): string {
  return `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
}

// ---------------------------------------------------------------------------
// Service: owned prefix, pinned sessions, exact bytes, paginated cleanup
// ---------------------------------------------------------------------------

export class ObjectStoreService {
  private readonly limits: ObjectStoreLimits;
  // Owned grants: exact (owner, prefix) pairs. The caller-supplied prefix is
  // never authority on its own; openPrefix joins only an exact declared
  // grant.
  private readonly grants: ReadonlySet<string>;
  private readonly prefixes = new Map<string, PrefixRecord>();
  private readonly sessions = new Map<string, SessionRecord>();
  // The shared bucket outside every owned prefix. Seeded once as foreign
  // fixtures; no observer operation reads or mutates it. The self-check
  // pins its digest before and after every control.
  private readonly shared: ReadonlyMap<string, Uint8Array>;

  constructor(
    grants: readonly (readonly [owner: string, prefix: string])[],
    sharedKeys: ReadonlyMap<string, Uint8Array>,
    limits: ObjectStoreLimits,
  ) {
    if (grants.length === 0) {
      throw new ObjectStoreError("unknown-prefix", "declare at least one owned grant");
    }
    if (grants.length > limits.maxPrefixes) {
      throw new ObjectStoreError("capacity-exhausted", "declared grants exceed the cap");
    }
    const seen = new Set<string>();
    for (const [owner, prefix] of grants) {
      checkOwner(owner);
      checkDeclaredPrefix(prefix);
      const key = `${owner}\0${prefix}`;
      if (seen.has(key)) {
        throw new ObjectStoreError("unknown-prefix", `duplicate owned grant: ${owner}/${prefix}`);
      }
      seen.add(key);
    }
    this.grants = seen;
    this.limits = limits;
    const foreign = new Map<string, Uint8Array>();
    for (const [key, bytes] of sharedKeys) {
      checkKeyShape(key);
      foreign.set(key, Uint8Array.from(bytes));
    }
    this.shared = foreign;
  }

  get prefixCount(): number {
    return this.prefixes.size;
  }

  get liveSessionCount(): number {
    let count = 0;
    for (const session of this.sessions.values()) {
      if (!session.closed) count++;
    }
    return count;
  }

  sharedDigest(): string {
    const hash = createHash("sha256");
    for (const key of [...this.shared.keys()].sort()) {
      hash.update(key, "utf8");
      hash.update("\0", "utf8");
      hash.update(this.shared.get(key) as Uint8Array);
      hash.update("\x01", "utf8");
    }
    return `sha256:${hash.digest("hex")}`;
  }

  // Open one owned prefix. The caller prefix binds to the owned grant at
  // open: only an exact (owner, prefix) grant admits. A prefix that merely
  // shares the bucket, reaches above the grant, or was never granted
  // rejects before any effect.
  openPrefix(owner: string, prefix: string): PrefixReceipt {
    checkOwner(owner);
    checkDeclaredPrefix(prefix);
    if (!this.grants.has(`${owner}\0${prefix}`)) {
      fail("unknown-prefix", `no owned grant for caller prefix: ${owner}/${prefix}`);
    }
    const key = `${owner}\0${prefix}`;
    const prior = this.prefixes.get(key);
    if (prior !== undefined && !prior.closed) {
      return this.receiptOf(prior);
    }
    if (this.prefixes.size >= this.limits.maxPrefixes) {
      fail("capacity-exhausted", "prefix table full");
    }
    const record: PrefixRecord = {
      prefix,
      owner,
      handle: `prefix-handle:${owner}:${prefix}`,
      token: mintToken("pfx"),
      closed: false,
      objects: new Map(),
      pending: new Map(),
      continuations: new Map(),
      writeSeq: 0,
      generation: 0,
      sealedGeneration: -1,
      sealedEmpty: false,
      handleSeq: 0,
    };
    this.prefixes.set(key, record);
    return this.receiptOf(record);
  }

  private receiptOf(record: PrefixRecord): PrefixReceipt {
    return Object.freeze({
      prefix: record.prefix,
      owner: record.owner,
      handle: record.handle,
      handleDigest: digestToken(record.token),
      objects: record.objects.size,
    });
  }

  receipt(owner: string, prefix: string): PrefixReceipt {
    return this.receiptOf(this.requirePrefix(owner, prefix));
  }

  private requirePrefix(owner: string, prefix: string): PrefixRecord {
    const record = this.prefixes.get(`${owner}\0${prefix}`);
    if (record === undefined || record.closed) {
      fail("unknown-prefix", `no open prefix for owner: ${owner}/${prefix}`);
    }
    return record as PrefixRecord;
  }

  closePrefix(owner: string, prefix: string): void {
    const record = this.requirePrefix(owner, prefix);
    for (const session of this.sessions.values()) {
      if (!session.closed && session.owner === owner && session.prefix === prefix) {
        fail("prefix-closed", `session still pinned: ${session.session}`);
      }
    }
    record.closed = true;
  }

  // Pin one session to an owned prefix. The token is minted here and
  // verified by table lookup on every use, so invented tokens are never
  // authority.
  openSession(owner: string, prefix: string, session: string): SessionFacts {
    const record = this.requirePrefix(owner, prefix);
    checkSessionName(session);
    if (record.owner !== owner) {
      fail("wrong-owner", "prefix is owned by another identity");
    }
    const key = `${owner}\0${prefix}\0${session}`;
    const prior = this.sessions.get(key);
    if (prior !== undefined && !prior.closed) {
      return this.sessionFactsOf(prior);
    }
    if (this.liveSessionCount >= this.limits.maxSessions) {
      fail("capacity-exhausted", "session table full");
    }
    const entry: SessionRecord = {
      session,
      prefix,
      owner,
      token: mintToken("sess"),
      pinned: true,
      closed: false,
    };
    this.sessions.set(key, entry);
    return this.sessionFactsOf(entry);
  }

  private sessionFactsOf(session: SessionRecord): SessionFacts {
    return Object.freeze({
      session: session.session,
      prefix: session.prefix,
      owner: session.owner,
      handleDigest: digestToken(session.token),
      pinned: session.pinned,
    });
  }

  sessionTokenForTest(owner: string, prefix: string, session: string): string {
    return this.requireSession(owner, prefix, session).token;
  }

  private requireSession(owner: string, prefix: string, session: string): SessionRecord {
    const entry = this.sessions.get(`${owner}\0${prefix}\0${session}`);
    if (entry === undefined || entry.closed) {
      fail("unknown-session", `no pinned session: ${session}`);
    }
    return entry as SessionRecord;
  }

  private requireToken(session: SessionRecord, token: string): void {
    if (token === "" || token !== session.token) {
      fail("forged-token", "session token is not the pinned token");
    }
  }

  // All effects stay inside the opened prefix. A key at or above the grant
  // edge, or under any other path, rejects without effect: the shared
  // bucket is untouched.
  private requireScopedKey(record: PrefixRecord, key: string): string {
    checkKeyShape(key);
    if (key !== record.prefix && !key.startsWith(`${record.prefix}/`)) {
      fail("key-escapes-prefix", `key is outside the owned prefix: ${key}`);
    }
    if (key === record.prefix) {
      fail("malformed-key", "key must name an object under the prefix, not the prefix itself");
    }
    return key;
  }

  closeSession(owner: string, prefix: string, session: string, token: string): void {
    const entry = this.requireSession(owner, prefix, session);
    if (entry.owner !== owner || entry.prefix !== prefix) {
      fail("wrong-owner", "session is pinned to another identity");
    }
    this.requireToken(entry, token);
    entry.closed = true;
  }

  // Accept one write. Accepted is not settled: the bytes are staged under a
  // write id and become visible only through settleWrite. Staging copies the
  // bytes, so later caller mutation cannot corrupt the staged object.
  put(
    owner: string,
    prefix: string,
    session: string,
    token: string,
    key: string,
    bytes: Uint8Array,
  ): WriteFacts {
    const record = this.requirePrefix(owner, prefix);
    const entry = this.requireSession(owner, prefix, session);
    if (entry.owner !== owner || entry.prefix !== prefix) {
      fail("wrong-owner", "session is pinned to another identity");
    }
    this.requireToken(entry, token);
    this.requireScopedKey(record, key);
    if (!(bytes instanceof Uint8Array)) {
      fail("malformed-key", "put needs a Uint8Array payload");
    }
    if (bytes.length > this.limits.maxObjectBytes) {
      fail("capacity-exhausted", "object exceeds the byte cap");
    }
    if (record.pending.size >= this.limits.maxPendingWrites) {
      fail("capacity-exhausted", "pending-write table full");
    }
    for (const staged of record.pending.values()) {
      if (!staged.settled && staged.key === key) {
        fail("write-busy", `a write is already accepted for key: ${key}`);
      }
    }
    if (!record.objects.has(key) && record.objects.size >= this.limits.maxObjects) {
      fail("capacity-exhausted", "object table full");
    }
    record.writeSeq += 1;
    const writeId = `w${record.writeSeq}`;
    record.pending.set(writeId, {
      writeId,
      key,
      bytes: Uint8Array.from(bytes),
      settled: false,
    });
    record.generation += 1;
    return Object.freeze({
      writeId,
      key,
      bytesDigest: digestBytes(bytes),
      settled: false,
    });
  }

  settleWrite(
    owner: string,
    prefix: string,
    session: string,
    token: string,
    writeId: string,
  ): ObjectFacts {
    const record = this.requirePrefix(owner, prefix);
    const entry = this.requireSession(owner, prefix, session);
    if (entry.owner !== owner || entry.prefix !== prefix) {
      fail("wrong-owner", "session is pinned to another identity");
    }
    this.requireToken(entry, token);
    const staged = record.pending.get(writeId);
    if (staged === undefined) {
      fail("write-unknown", `no accepted write: ${writeId}`);
    }
    const write = staged as PendingWrite;
    if (write.settled) {
      fail("write-settled", `write already settled: ${writeId}`);
    }
    write.settled = true;
    record.objects.set(write.key, Uint8Array.from(write.bytes));
    record.generation += 1;
    return Object.freeze({
      key: write.key,
      size: write.bytes.length,
      digest: digestBytes(write.bytes),
    });
  }

  pending(owner: string, prefix: string, session: string, token: string): PendingFacts {
    const record = this.requirePrefix(owner, prefix);
    const entry = this.requireSession(owner, prefix, session);
    if (entry.owner !== owner || entry.prefix !== prefix) {
      fail("wrong-owner", "session is pinned to another identity");
    }
    this.requireToken(entry, token);
    const writes = [...record.pending.values()]
      .filter((write) => !write.settled)
      .sort((a, b) => (a.writeId < b.writeId ? -1 : 1))
      .map((write) =>
        Object.freeze({
          writeId: write.writeId,
          key: write.key,
          bytesDigest: digestBytes(write.bytes),
          settled: false,
        }),
      );
    return Object.freeze({
      prefix: record.prefix,
      writes: Object.freeze(writes),
      count: writes.length,
    });
  }

  // Fetch one settled object, byte-exact. Accepted-but-unsettled writes are
  // not visible here; they are observable only through pending(). Bytes
  // cross as copies: the stored buffer must never alias a caller buffer.
  get(
    owner: string,
    prefix: string,
    session: string,
    token: string,
    key: string,
  ): Readonly<{ key: string; bytes: Uint8Array; size: number; digest: string }> {
    const record = this.requirePrefix(owner, prefix);
    const entry = this.requireSession(owner, prefix, session);
    if (entry.owner !== owner || entry.prefix !== prefix) {
      fail("wrong-owner", "session is pinned to another identity");
    }
    this.requireToken(entry, token);
    this.requireScopedKey(record, key);
    const stored = record.objects.get(key);
    if (stored === undefined) {
      fail("unknown-key", `no settled object: ${key}`);
    }
    const bytes = stored as Uint8Array;
    return Object.freeze({
      key,
      bytes: Uint8Array.from(bytes),
      size: bytes.length,
      digest: digestBytes(bytes),
    });
  }

  delete(owner: string, prefix: string, session: string, token: string, key: string): void {
    const record = this.requirePrefix(owner, prefix);
    const entry = this.requireSession(owner, prefix, session);
    if (entry.owner !== owner || entry.prefix !== prefix) {
      fail("wrong-owner", "session is pinned to another identity");
    }
    this.requireToken(entry, token);
    this.requireScopedKey(record, key);
    if (!record.objects.has(key)) {
      fail("unknown-key", `no settled object: ${key}`);
    }
    record.objects.delete(key);
    record.generation += 1;
  }

  compareBytes(stored: Uint8Array, claimed: Uint8Array): boolean {
    // Untyped callers must still fail at the store layer, never with a raw
    // TypeError: every failure names its layer.
    if (!(stored instanceof Uint8Array) || !(claimed instanceof Uint8Array)) {
      fail("malformed-key", "compareBytes needs Uint8Array operands");
    }
    if (stored.length !== claimed.length) return false;
    for (let index = 0; index < stored.length; index++) {
      if (stored[index] !== claimed[index]) return false;
    }
    return true;
  }

  // List one page of settled keys in lexicographic order. The continuation
  // is opaque and minted here: the first page takes null, later pages take
  // the exact token of the previous page. A continuation records the
  // generation it was minted at; any mutation since rejects as stale rather
  // than silently skipping or repeating keys. A page that reaches the end
  // is terminal (complete === true) and records the scan; every other page
  // is explicitly incomplete.
  list(
    owner: string,
    prefix: string,
    session: string,
    token: string,
    limit: number,
    continuation: string | null,
  ): PageFacts {
    const record = this.requirePrefix(owner, prefix);
    const entry = this.requireSession(owner, prefix, session);
    if (entry.owner !== owner || entry.prefix !== prefix) {
      fail("wrong-owner", "session is pinned to another identity");
    }
    this.requireToken(entry, token);
    if (!Number.isSafeInteger(limit) || limit <= 0 || limit > this.limits.maxPageSize) {
      fail("bad-page", `limit must be within 1..${this.limits.maxPageSize}`);
    }
    let after = "";
    if (continuation !== null) {
      const seen = record.continuations.get(continuation);
      if (seen === undefined || seen.consumed) {
        fail("forged-continuation", "continuation is not a live listing edge");
      }
      const edge = seen as ContinuationRecord;
      if (edge.generation !== record.generation) {
        fail("stale-continuation", "the listing moved since this continuation was minted");
      }
      edge.consumed = true;
      after = edge.after;
    }
    const names = [...record.objects.keys()].sort().filter((name) => name > after);
    const page = names.slice(0, limit);
    const terminal = page.length === names.length;
    let next: string | null = null;
    if (!terminal) {
      record.handleSeq += 1;
      next = mintToken("cont");
      record.continuations.set(next, {
        token: next,
        after: page[page.length - 1] as string,
        generation: record.generation,
        consumed: false,
      });
    } else {
      record.sealedGeneration = record.generation;
      record.sealedEmpty = names.length === 0;
    }
    const keys = page.map((name) => {
      const bytes = record.objects.get(name) as Uint8Array;
      return Object.freeze({ key: name, size: bytes.length, digest: digestBytes(bytes) });
    });
    const digest = `sha256:${createHash("sha256")
      .update(keys.map((entry) => `${entry.key}=${entry.digest}`).join(","), "utf8")
      .digest("hex")}`;
    return Object.freeze({
      prefix: record.prefix,
      keys: Object.freeze(keys),
      count: keys.length,
      complete: terminal,
      nextContinuation: next,
      generation: record.generation,
      digest,
    });
  }

  // Seal the prefix as cleaned up. The seal requires all four, jointly:
  // zero live sessions, zero unsettled writes, zero live objects, and a
  // terminal listing observed at the current generation. An incomplete
  // page, a stale terminal scan, or an eventually empty list observed while
  // a write was still pending each reject with the reason named.
  sealCleanup(owner: string, prefix: string): CleanupReceipt {
    const record = this.requirePrefix(owner, prefix);
    if (record.owner !== owner) {
      fail("wrong-owner", "prefix is owned by another identity");
    }
    for (const session of this.sessions.values()) {
      if (!session.closed && session.owner === owner && session.prefix === prefix) {
        fail("session-live", `session still pinned: ${session.session}`);
      }
    }
    for (const staged of record.pending.values()) {
      if (!staged.settled) {
        fail("pending-writes", `write accepted but never settled: ${staged.writeId}`);
      }
    }
    if (record.objects.size > 0) {
      fail("keys-remain", `${record.objects.size} object(s) still live`);
    }
    if (record.sealedGeneration !== record.generation || !record.sealedEmpty) {
      fail("scan-incomplete", "no terminal empty listing at the current generation");
    }
    const digest = `sha256:${createHash("sha256")
      .update(`${record.prefix}|${record.owner}|${record.generation}`, "utf8")
      .digest("hex")}`;
    return Object.freeze({
      prefix: record.prefix,
      owner: record.owner,
      handleDigest: digestToken(record.token),
      objects: 0,
      pending: 0,
      generation: record.generation,
      digest,
    });
  }
}
