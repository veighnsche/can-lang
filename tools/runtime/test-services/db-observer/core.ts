// K22: independent raw DB observer core.
//
// An INDEPENDENT raw database observer over in-memory doubles only: no real
// database, no driver, no sockets, no I/O. The observer owns its namespace
// handle, pins one identity-bound raw connection per conversation, stores
// typed exact cells (number/text/bytes/null), and names the failing layer
// (engine/driver/sql) on every error.
//
// Independence scope from the Can adapter under test (normative):
//   SHARES: nothing executable. The observer never imports, calls, or reads
//   adapter code, adapter-decoded values, or adapter connection state. The
//   only strings that may coincide are the namespace/table names under test
//   (supplied by the test, not by the adapter) and the P29 grant vocabulary
//   the Go mirror uses for receipts.
//   PROVES INDEPENDENTLY: exact stored cells (number lexeme, text code units,
//   bytes, null-vs-empty), row order, row count, namespace receipt identity,
//   connection pinning (one identity, one conversation), and error provenance
//   (every failure names its layer). Adapter claims are compared AGAINST
//   these facts; the observer never derives a fact FROM an adapter claim.
//   Consequently a coincidentally correct adapter value still fails when the
//   observer's exact cell disagrees (narrowed bigint, re-encoded text,
//   dropped byte, reordered row).
//
// Split with the Go mirror (tools/native-test-owner/external/db_observer.go):
//   TS core owns typed exact cells, row order, read-out facts, and the
//   bounded self-check. The Go mirror owns P29-journaled namespace/connection
//   discipline (grants, receipts, cleanup order) using the registry's own
//   vocabulary. Both enforce the same contract: owned namespace admission,
//   pinned identity-bound connection, single conversation per connection,
//   and layered errors. Neither reads the Can adapter.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, or live runtimes. Local controls only.

import { createHash, randomBytes } from "node:crypto";

export const DB_OBSERVER_SCHEMA_VERSION = "1" as const;

// ---------------------------------------------------------------------------
// Independence scope (normative, mirrored in db_observer.go)
// ---------------------------------------------------------------------------

export const DB_OBSERVER_INDEPENDENCE_SCOPE: string =
  "independent: shares no adapter code, decoded values, or connection state; " +
  "proves namespace receipt, pinned connection, exact cells, row order/count, " +
  "and layered error provenance from its own seeded doubles alone";

// ---------------------------------------------------------------------------
// Layers, codes, and the layered error
// ---------------------------------------------------------------------------

export const DB_OBSERVER_LAYERS = ["engine", "driver", "sql"] as const;
export type DbObserverLayer = (typeof DB_OBSERVER_LAYERS)[number];

// Closed failure-code vocabulary. Every code maps to exactly one layer:
//   engine: namespace admission, capacity, ownership, lifecycle.
//   driver: connection pinning, tokens, conversation discipline.
//   sql:    statements, cell lexemes, types, row shape.
export const DB_OBSERVER_CODES = [
  "unknown-namespace",
  "namespace-closed",
  "capacity-exhausted",
  "wrong-owner",
  "closed-handle",
  "unknown-connection",
  "forged-token",
  "connection-busy",
  "no-conversation",
  "unknown-table",
  "malformed-lexeme",
  "wrong-cell-type",
  "row-arity",
  "malformed-statement",
] as const;
export type DbObserverCode = (typeof DB_OBSERVER_CODES)[number];

const CODE_LAYER: Readonly<Record<DbObserverCode, DbObserverLayer>> = {
  "unknown-namespace": "engine",
  "namespace-closed": "engine",
  "capacity-exhausted": "engine",
  "wrong-owner": "engine",
  "closed-handle": "engine",
  "unknown-connection": "driver",
  "forged-token": "driver",
  "connection-busy": "driver",
  "no-conversation": "driver",
  "unknown-table": "sql",
  "malformed-lexeme": "sql",
  "wrong-cell-type": "sql",
  "row-arity": "sql",
  "malformed-statement": "sql",
};

export function layerOfCode(code: DbObserverCode): DbObserverLayer {
  return CODE_LAYER[code];
}

export class DbObserverError extends Error {
  readonly layer: DbObserverLayer;
  readonly code: DbObserverCode;
  constructor(code: DbObserverCode, message: string) {
    super(`[${CODE_LAYER[code]}:${code}] ${message}`);
    this.name = "DbObserverError";
    this.layer = CODE_LAYER[code];
    this.code = code;
  }
}

function fail(code: DbObserverCode, message: string): never {
  throw new DbObserverError(code, message);
}

// ---------------------------------------------------------------------------
// Bounds and names
// ---------------------------------------------------------------------------

export type DbObserverLimits = Readonly<{
  maxNamespaces: number;
  maxConnections: number;
  maxTables: number;
  maxRows: number;
  maxCellsPerRow: number;
  maxCellBytes: number;
}>;

const LIMIT_KEYS = [
  "maxNamespaces",
  "maxConnections",
  "maxTables",
  "maxRows",
  "maxCellsPerRow",
  "maxCellBytes",
] as const;

export function checkLimits(value: unknown): DbObserverLimits {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new DbObserverError("malformed-statement", "limits must be an object");
  }
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    if (!(LIMIT_KEYS as readonly string[]).includes(key)) {
      throw new DbObserverError("malformed-statement", `unknown limit field: ${key}`);
    }
  }
  const out: Record<string, number> = {};
  for (const key of LIMIT_KEYS) {
    const entry = record[key];
    if (!Number.isSafeInteger(entry) || (entry as number) <= 0) {
      throw new DbObserverError("malformed-statement", `${key} must be a positive integer`);
    }
    out[key] = entry as number;
  }
  return Object.freeze(out) as DbObserverLimits;
}

const MAX_NAME_LEN = 128;

function checkName(value: unknown, what: string): string {
  if (typeof value !== "string" || value === "" || value.length > MAX_NAME_LEN) {
    fail(
      "malformed-statement",
      `${what} must be a non-empty name of at most ${MAX_NAME_LEN} chars`,
    );
  }
  if (!/^[A-Za-z0-9][A-Za-z0-9_.:-]*$/.test(value)) {
    fail("malformed-statement", `${what} carries illegal characters: ${value}`);
  }
  return value;
}

// ---------------------------------------------------------------------------
// Typed exact cells
// ---------------------------------------------------------------------------

export const DB_CELL_TAGS = ["number", "text", "bytes", "null"] as const;
export type DbCellTag = (typeof DB_CELL_TAGS)[number];

// Exactness contract (mirrors K03 read-out discipline: bits/lexemes/gaps):
//   number: the source lexeme is stored verbatim and never coerced through a
//     float. "1.10" stays "1.10", "9007199254740993" stays exact, "-0" stays
//     distinct from "0". Only canonical numeric lexemes are admitted.
//   text: stored code-unit-exact; never re-encoded, normalized, or trimmed.
//   bytes: stored byte-exact; compared byte-for-byte.
//   null: distinct from empty text, empty bytes, and zero. There is exactly
//     one null.
export type DbCell = Readonly<
  | { tag: "number"; lexeme: string }
  | { tag: "text"; text: string }
  | { tag: "bytes"; bytes: Uint8Array }
  | { tag: "null" }
>;

const NUMBER_LEXEME = /^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$/;

export function makeNumberCell(lexeme: unknown): DbCell {
  if (typeof lexeme !== "string" || !NUMBER_LEXEME.test(lexeme)) {
    fail("malformed-lexeme", `not a canonical numeric lexeme: ${String(lexeme)}`);
  }
  return Object.freeze({ tag: "number", lexeme }) as DbCell;
}

export function makeTextCell(text: unknown): DbCell {
  if (typeof text !== "string") {
    fail("wrong-cell-type", "text cell needs a string");
  }
  return Object.freeze({ tag: "text", text }) as DbCell;
}

export function makeBytesCell(bytes: unknown): DbCell {
  if (!(bytes instanceof Uint8Array)) {
    fail("wrong-cell-type", "bytes cell needs a Uint8Array");
  }
  return Object.freeze({ tag: "bytes", bytes: Uint8Array.from(bytes) }) as DbCell;
}

export function makeNullCell(): DbCell {
  return Object.freeze({ tag: "null" }) as DbCell;
}

function checkCell(value: unknown, maxCellBytes: number): DbCell {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail("wrong-cell-type", "cell must be a tagged object");
  }
  const record = value as Record<string, unknown>;
  switch (record["tag"]) {
    case "number":
      return makeNumberCell(record["lexeme"]);
    case "text": {
      const cell = makeTextCell(record["text"]);
      if ((cell as { text: string }).text.length > maxCellBytes) {
        fail("capacity-exhausted", "text cell exceeds the byte cap");
      }
      return cell;
    }
    case "bytes": {
      const raw = record["bytes"];
      const bytes =
        raw instanceof Uint8Array
          ? raw
          : Array.isArray(raw) && raw.every((b) => Number.isInteger(b) && b >= 0 && b <= 255)
            ? Uint8Array.from(raw as number[])
            : fail("wrong-cell-type", "bytes cell needs a Uint8Array or byte array");
      if (bytes.length > maxCellBytes) {
        fail("capacity-exhausted", "bytes cell exceeds the byte cap");
      }
      return makeBytesCell(bytes);
    }
    case "null":
      return makeNullCell();
    default:
      return fail("wrong-cell-type", `unknown cell tag: ${String(record["tag"])}`);
  }
}

function cellsEqual(a: DbCell, b: DbCell): boolean {
  if (a.tag !== b.tag) return false;
  switch (a.tag) {
    case "number":
      return a.lexeme === (b as { lexeme: string }).lexeme;
    case "text":
      return a.text === (b as { text: string }).text;
    case "bytes": {
      const x = a.bytes;
      const y = (b as { bytes: Uint8Array }).bytes;
      if (x.length !== y.length) return false;
      for (let i = 0; i < x.length; i++) {
        if (x[i] !== y[i]) return false;
      }
      return true;
    }
    case "null":
      return true;
  }
}

function cellDigest(cell: DbCell): string {
  const hash = createHash("sha256");
  hash.update(cell.tag, "utf8");
  hash.update("|", "utf8");
  switch (cell.tag) {
    case "number":
      hash.update(cell.lexeme, "utf8");
      break;
    case "text":
      hash.update(cell.text, "utf8");
      break;
    case "bytes":
      hash.update(cell.bytes);
      break;
    case "null":
      hash.update("null", "utf8");
      break;
  }
  return `sha256:${hash.digest("hex")}`;
}

// ---------------------------------------------------------------------------
// Facts
// ---------------------------------------------------------------------------

export type NamespaceReceipt = Readonly<{
  namespace: string;
  owner: string;
  handle: string;
  handleDigest: string;
  tables: number;
}>;

export type ConnectionFacts = Readonly<{
  connection: string;
  namespace: string;
  owner: string;
  handleDigest: string;
  pinned: boolean;
  conversationOpen: boolean;
}>;

export type RowFacts = Readonly<{
  seq: number;
  cells: readonly DbCell[];
  digest: string;
}>;

export type ReadFacts = Readonly<{
  connection: string;
  namespace: string;
  table: string;
  rows: readonly RowFacts[];
  rowCount: number;
  digest: string;
}>;

type NamespaceRecord = {
  namespace: string;
  owner: string;
  handle: string;
  token: string;
  closed: boolean;
  tables: Map<string, DbCell[][]>;
};

type ConnectionRecord = {
  connection: string;
  namespace: string;
  owner: string;
  token: string;
  pinned: boolean;
  conversationOpen: boolean;
  closed: boolean;
};

function mintToken(prefix: string): string {
  return `${prefix}-${randomBytes(16).toString("hex")}`;
}

function digestToken(token: string): string {
  return `sha256:${createHash("sha256").update(token, "utf8").digest("hex")}`;
}

// ---------------------------------------------------------------------------
// Service: owned namespace, pinned raw connection, exact reads
// ---------------------------------------------------------------------------

export class DbObserverService {
  private readonly limits: DbObserverLimits;
  private readonly declared: ReadonlySet<string>;
  private readonly namespaces = new Map<string, NamespaceRecord>();
  private readonly connections = new Map<string, ConnectionRecord>();

  constructor(namespaces: readonly string[], limits: DbObserverLimits) {
    if (namespaces.length === 0) {
      throw new DbObserverError("malformed-statement", "declare at least one namespace");
    }
    if (namespaces.length > limits.maxNamespaces) {
      throw new DbObserverError("capacity-exhausted", "declared namespaces exceed the cap");
    }
    const seen = new Set<string>();
    for (const name of namespaces) {
      checkName(name, "namespace");
      if (seen.has(name)) {
        throw new DbObserverError("malformed-statement", `duplicate namespace: ${name}`);
      }
      seen.add(name);
    }
    this.declared = seen;
    this.limits = limits;
  }

  get namespaceCount(): number {
    return this.namespaces.size;
  }

  get liveConnectionCount(): number {
    let count = 0;
    for (const conn of this.connections.values()) {
      if (!conn.closed) count++;
    }
    return count;
  }

  // Own one declared namespace. The handle token is minted here and never
  // accepted from the caller: invented handles are not authority.
  openNamespace(owner: string, namespace: string): NamespaceReceipt {
    checkName(owner, "owner");
    checkName(namespace, "namespace");
    if (!this.declared.has(namespace)) {
      fail("unknown-namespace", `namespace is not declared: ${namespace}`);
    }
    const key = `${owner}\0${namespace}`;
    const prior = this.namespaces.get(key);
    if (prior !== undefined && !prior.closed) {
      return this.receiptOf(prior);
    }
    if (this.namespaces.size >= this.limits.maxNamespaces) {
      fail("capacity-exhausted", "namespace table full");
    }
    const token = mintToken("ns");
    const record: NamespaceRecord = {
      namespace,
      owner,
      handle: `ns-handle:${owner}:${namespace}`,
      token,
      closed: false,
      tables: new Map(),
    };
    this.namespaces.set(key, record);
    return this.receiptOf(record);
  }

  private receiptOf(record: NamespaceRecord): NamespaceReceipt {
    return Object.freeze({
      namespace: record.namespace,
      owner: record.owner,
      handle: record.handle,
      handleDigest: digestToken(record.token),
      tables: record.tables.size,
    });
  }

  receipt(owner: string, namespace: string): NamespaceReceipt {
    const record = this.requireNamespace(owner, namespace);
    return this.receiptOf(record);
  }

  private requireNamespace(owner: string, namespace: string): NamespaceRecord {
    const record = this.namespaces.get(`${owner}\0${namespace}`);
    if (record === undefined || record.closed) {
      fail("unknown-namespace", `no open namespace for owner: ${owner}/${namespace}`);
    }
    return record as NamespaceRecord;
  }

  closeNamespace(owner: string, namespace: string): void {
    const record = this.requireNamespace(owner, namespace);
    for (const conn of this.connections.values()) {
      if (!conn.closed && conn.owner === owner && conn.namespace === namespace) {
        fail("namespace-closed", `connection still pinned: ${conn.connection}`);
      }
    }
    record.closed = true;
  }

  // Seed one table with exact rows. Seeding is the observer's own fixture
  // path: rows come from test literals, never from the adapter. Row order
  // is insertion order and is preserved verbatim on every read.
  seed(
    owner: string,
    namespace: string,
    table: string,
    rows: readonly (readonly unknown[])[],
  ): Readonly<{ table: string; rows: number }> {
    const record = this.requireNamespace(owner, namespace);
    checkName(table, "table");
    if (!record.tables.has(table) && record.tables.size >= this.limits.maxTables) {
      fail("capacity-exhausted", "table cap reached");
    }
    if (rows.length > this.limits.maxRows) {
      fail("capacity-exhausted", "seed rows exceed the row cap");
    }
    const exact: DbCell[][] = [];
    let arity = -1;
    for (const row of rows) {
      if (!Array.isArray(row)) {
        fail("row-arity", "each seed row must be an array of cells");
      }
      if (row.length > this.limits.maxCellsPerRow) {
        fail("capacity-exhausted", "seed row exceeds the cell cap");
      }
      if (arity < 0) {
        arity = row.length;
      } else if (row.length !== arity) {
        fail("row-arity", `ragged seed row: want ${arity} cells, got ${row.length}`);
      }
      exact.push(row.map((cell) => checkCell(cell, this.limits.maxCellBytes)));
    }
    record.tables.set(table, exact);
    return Object.freeze({ table, rows: exact.length });
  }

  // Pin one raw connection to an owned namespace. The connection binds to
  // the owner's identity at pin time; the token is minted here and verified
  // by table lookup on every use, so invented tokens are never authority.
  pinConnection(owner: string, namespace: string, connection: string): ConnectionFacts {
    const record = this.requireNamespace(owner, namespace);
    checkName(connection, "connection");
    if (record.owner !== owner) {
      fail("wrong-owner", "namespace is owned by another identity");
    }
    const key = `${owner}\0${namespace}\0${connection}`;
    const prior = this.connections.get(key);
    if (prior !== undefined && !prior.closed) {
      return this.connectionFactsOf(prior);
    }
    if (this.liveConnectionCount >= this.limits.maxConnections) {
      fail("capacity-exhausted", "connection table full");
    }
    const entry: ConnectionRecord = {
      connection,
      namespace,
      owner,
      token: mintToken("conn"),
      pinned: true,
      conversationOpen: false,
      closed: false,
    };
    this.connections.set(key, entry);
    return this.connectionFactsOf(entry);
  }

  private connectionFactsOf(conn: ConnectionRecord): ConnectionFacts {
    return Object.freeze({
      connection: conn.connection,
      namespace: conn.namespace,
      owner: conn.owner,
      handleDigest: digestToken(conn.token),
      pinned: conn.pinned,
      conversationOpen: conn.conversationOpen,
    });
  }

  connectionTokenForTest(owner: string, namespace: string, connection: string): string {
    const conn = this.requireConnection(owner, namespace, connection);
    return conn.token;
  }

  private requireConnection(
    owner: string,
    namespace: string,
    connection: string,
  ): ConnectionRecord {
    const conn = this.connections.get(`${owner}\0${namespace}\0${connection}`);
    if (conn === undefined || conn.closed) {
      fail("unknown-connection", `no pinned connection: ${connection}`);
    }
    return conn as ConnectionRecord;
  }

  private requireToken(conn: ConnectionRecord, token: string): void {
    if (token === "" || token !== conn.token) {
      fail("forged-token", "connection token is not the pinned token");
    }
  }

  unpinConnection(owner: string, namespace: string, connection: string, token: string): void {
    const conn = this.requireConnection(owner, namespace, connection);
    if (conn.owner !== owner || conn.namespace !== namespace) {
      fail("wrong-owner", "connection is pinned to another identity");
    }
    this.requireToken(conn, token);
    if (conn.conversationOpen) {
      fail("connection-busy", "a conversation is still open");
    }
    conn.closed = true;
  }

  // Begin one read conversation. Single-conversation discipline: a pinned
  // connection carries at most one open conversation, so interleaved reads
  // can never silently share or reorder state.
  beginRead(
    owner: string,
    namespace: string,
    connection: string,
    token: string,
    table: string,
  ): Readonly<{ conversation: string }> {
    const conn = this.requireConnection(owner, namespace, connection);
    if (conn.owner !== owner || conn.namespace !== namespace) {
      fail("wrong-owner", "connection is pinned to another identity");
    }
    this.requireToken(conn, token);
    if (conn.conversationOpen) {
      fail("connection-busy", "one conversation is already open");
    }
    checkName(table, "table");
    const record = this.requireNamespace(owner, namespace);
    if (!record.tables.has(table)) {
      fail("unknown-table", `no such table in namespace: ${table}`);
    }
    conn.conversationOpen = true;
    return Object.freeze({ conversation: `${conn.connection}:${table}` });
  }

  // Fetch every row of the open conversation in stored order. Cells cross
  // exactly: number lexemes verbatim, text code-unit-exact, bytes
  // byte-exact, null distinct from empty. The conversation stays open until
  // endRead so the caller cannot interleave a second conversation.
  fetch(
    owner: string,
    namespace: string,
    connection: string,
    token: string,
    table: string,
  ): ReadFacts {
    const conn = this.requireConnection(owner, namespace, connection);
    if (conn.owner !== owner || conn.namespace !== namespace) {
      fail("wrong-owner", "connection is pinned to another identity");
    }
    this.requireToken(conn, token);
    if (!conn.conversationOpen) {
      fail("no-conversation", "begin a read conversation first");
    }
    const record = this.requireNamespace(owner, namespace);
    const stored = record.tables.get(table);
    if (stored === undefined) {
      fail("unknown-table", `no such table in namespace: ${table}`);
    }
    const rows: RowFacts[] = (stored as DbCell[][]).map((cells, index) => {
      // Bytes cells cross as copies: the stored Uint8Array must never alias
      // a caller-held buffer, or caller mutation would corrupt observer
      // state and later digests. Strings are immutable; only bytes need it.
      const out = cells.map((cell) =>
        cell.tag === "bytes"
          ? (Object.freeze({
              tag: "bytes",
              bytes: Uint8Array.from(cell.bytes),
            }) as DbCell)
          : cell,
      );
      const digest = `sha256:${createHash("sha256")
        .update(out.map((cell) => cellDigest(cell)).join(","), "utf8")
        .digest("hex")}`;
      return Object.freeze({ seq: index, cells: Object.freeze(out), digest });
    });
    const digest = `sha256:${createHash("sha256")
      .update(rows.map((row) => row.digest).join(","), "utf8")
      .digest("hex")}`;
    return Object.freeze({
      connection: conn.connection,
      namespace,
      table,
      rows: Object.freeze(rows),
      rowCount: rows.length,
      digest,
    });
  }

  endRead(owner: string, namespace: string, connection: string, token: string): void {
    const conn = this.requireConnection(owner, namespace, connection);
    if (conn.owner !== owner || conn.namespace !== namespace) {
      fail("wrong-owner", "connection is pinned to another identity");
    }
    this.requireToken(conn, token);
    if (!conn.conversationOpen) {
      fail("no-conversation", "no conversation is open");
    }
    conn.conversationOpen = false;
  }

  // Compare one adapter-claimed row against the observer's exact row: every
  // cell must match tag-for-tag and bit-for-bit, so narrowing, re-encoding,
  // dropped bytes, and null/empty confusion are all detectable. The
  // comparison reads observer facts only; it never trusts the claim.
  compareRow(
    stored: readonly DbCell[],
    claimed: readonly unknown[],
  ): Readonly<{
    match: boolean;
    mismatches: readonly number[];
  }> {
    const mismatches: number[] = [];
    if (claimed.length !== stored.length) {
      return Object.freeze({ match: false, mismatches: Object.freeze([-1]) });
    }
    for (let index = 0; index < stored.length; index++) {
      let exact: DbCell;
      try {
        exact = checkCell(claimed[index], this.limits.maxCellBytes);
      } catch {
        mismatches.push(index);
        continue;
      }
      if (!cellsEqual(stored[index] as DbCell, exact)) {
        mismatches.push(index);
      }
    }
    return Object.freeze({ match: mismatches.length === 0, mismatches: Object.freeze(mismatches) });
  }
}
