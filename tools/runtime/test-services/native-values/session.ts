// K02: inert same-realm native sessions.
//
// Raw hostile values (getter-backed `then`, throwing thenables, revoked
// proxies) live only behind opaque value handles inside the service realm.
// Callers move handles plus inert facts; the transport path encodes handles
// and facts without ever dereferencing a handle to its raw value, so it can
// never trigger a getter, assimilate `then`, or touch a revoked Proxy.
// Per-session counters prove the hostile surface stayed untouched.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, or live runtimes. Reading raw values back out
// (bits, lexemes, identity, explicit effectful reads) belongs to K03; this
// module reports construction facts only.

import {
  checkFactsInert,
  decodeHandleWire,
  encodeHandleWire,
  NativeSchemaError,
  type NativeHandle,
  type NativeHandleWire,
  type NativeValueLimits,
} from "./schema.ts";
import { NativeHandleRegistry, type CloseReceipt } from "./schema-handles.ts";

export const NATIVE_SESSION_SCHEMA_VERSION = "1" as const;

// ---------------------------------------------------------------------------
// Closed make-kind and hostile-descriptor vocabularies
// ---------------------------------------------------------------------------

export const NATIVE_MAKE_KINDS = [
  "primitive_bits",
  "text",
  "ordered_entries",
  "hostile_descriptor",
] as const;
export type NativeMakeKind = (typeof NATIVE_MAKE_KINDS)[number];

export const NATIVE_HOSTILE_DESCRIPTORS = [
  "getter_then",
  "throwing_thenable",
  "revoked_proxy",
] as const;
export type NativeHostileDescriptor = (typeof NATIVE_HOSTILE_DESCRIPTORS)[number];

function isMakeKind(value: unknown): value is NativeMakeKind {
  return typeof value === "string" && (NATIVE_MAKE_KINDS as readonly string[]).includes(value);
}

function isHostileDescriptor(value: unknown): value is NativeHostileDescriptor {
  return (
    typeof value === "string" && (NATIVE_HOSTILE_DESCRIPTORS as readonly string[]).includes(value)
  );
}

// ---------------------------------------------------------------------------
// Hostile fixtures and their counters
// ---------------------------------------------------------------------------

// Counts touches of the hostile surface. Fixtures close over the owning
// session's counters; every service operation must leave these at zero.
export type HostileCounters = {
  getterReads: number;
  thenCalls: number;
  proxyTraps: number;
};

export function zeroHostileCounters(): HostileCounters {
  return { getterReads: 0, thenCalls: 0, proxyTraps: 0 };
}

// Build one hostile value. Construction itself never triggers the hostile
// surface: the getter is only defined, the throwing `then` is only stored,
// and the Proxy is revoked before it is returned.
export function createHostileValue(
  descriptor: NativeHostileDescriptor,
  counters: HostileCounters,
): unknown {
  if (descriptor === "getter_then") {
    const target: Record<string, unknown> = {};
    // oxlint-disable-next-line no-thenable -- Intentional getter-backed `then`: K02 fixture proving the service never reads it.
    Object.defineProperty(target, "then", {
      enumerable: true,
      configurable: false,
      get() {
        counters.getterReads += 1;
        return undefined;
      },
    });
    return target;
  }
  if (descriptor === "throwing_thenable") {
    return {
      // oxlint-disable-next-line no-thenable -- Intentional hostile thenable: K02 fixture proving the service never assimilates `then`.
      then() {
        counters.thenCalls += 1;
        throw new Error("hostile then called");
      },
    };
  }
  const { proxy, revoke } = Proxy.revocable(
    {},
    {
      get() {
        counters.proxyTraps += 1;
        return undefined;
      },
    },
  );
  revoke();
  return proxy;
}

// ---------------------------------------------------------------------------
// Session counters and facts (all inert data)
// ---------------------------------------------------------------------------

export type SessionCounters = Readonly<{
  makes: number;
  aliases: number;
  getterReads: number;
  thenCalls: number;
  proxyTraps: number;
}>;

export type MakeFacts = Readonly<{
  cell: string;
  kind: NativeMakeKind;
  descriptor?: string;
  bytes: number;
  alias_of?: string;
}>;

export type SessionCloseReceipt = CloseReceipt &
  Readonly<{
    cellsReleased: number;
  }>;

type SessionCounts = {
  makes: number;
  aliases: number;
  hostile: HostileCounters;
};

type ValueCell = {
  cellId: string;
  sessionId: string;
  kind: NativeMakeKind;
  descriptor: NativeHostileDescriptor | null;
  bytes: number;
  raw: unknown;
};

// ---------------------------------------------------------------------------
// Inert payload construction
// ---------------------------------------------------------------------------

type BuiltValue = {
  raw: unknown;
  descriptor: NativeHostileDescriptor | null;
  bytes: number;
};

function requirePayloadObject(payload: unknown): Record<string, unknown> {
  if (payload === null || typeof payload !== "object" || Array.isArray(payload)) {
    throw new NativeSchemaError("rejected", "invalid-request", "make payload must be an object");
  }
  return payload as Record<string, unknown>;
}

function buildRawValue(
  kind: NativeMakeKind,
  payload: unknown,
  hostile: HostileCounters,
  limits: NativeValueLimits,
): BuiltValue {
  const record = requirePayloadObject(payload);
  // Hostile content (functions, thenables, Promises) rejects before any
  // shape-dependent read beyond the plain tag field lookup above.
  checkFactsInert(record, "make payload");
  const tag = record["tag"];
  if (kind === "primitive_bits") {
    if (tag !== "f64_bits") {
      throw new NativeSchemaError(
        "rejected",
        "invalid-request",
        "primitive_bits payload needs tag f64_bits",
      );
    }
    const hi = record["hi"];
    const lo = record["lo"];
    if (!Number.isSafeInteger(hi) || (hi as number) < 0 || (hi as number) > 0xffffffff) {
      throw new NativeSchemaError("rejected", "invalid-request", "f64_bits hi must be a uint32");
    }
    if (!Number.isSafeInteger(lo) || (lo as number) < 0 || (lo as number) > 0xffffffff) {
      throw new NativeSchemaError("rejected", "invalid-request", "f64_bits lo must be a uint32");
    }
    const view = new DataView(new ArrayBuffer(8));
    view.setUint32(0, hi as number);
    view.setUint32(4, lo as number);
    return { raw: view.getFloat64(0), descriptor: null, bytes: 8 };
  }
  if (kind === "text") {
    if (tag !== "text" || typeof record["text"] !== "string") {
      throw new NativeSchemaError(
        "rejected",
        "invalid-request",
        "text payload needs a string text field",
      );
    }
    const text = record["text"] as string;
    if (text.length > limits.maxObserveBytes) {
      throw new NativeSchemaError("rejected", "resource-limit", "text exceeds the byte cap");
    }
    return { raw: text, descriptor: null, bytes: text.length };
  }
  if (kind === "ordered_entries") {
    if (tag !== "entries" || !Array.isArray(record["entries"])) {
      throw new NativeSchemaError(
        "rejected",
        "invalid-request",
        "ordered_entries payload needs an entries array",
      );
    }
    const entries = record["entries"] as unknown[];
    if (entries.length > limits.maxObserveEntries) {
      throw new NativeSchemaError("rejected", "resource-limit", "entries exceed the entry cap");
    }
    const pairs: ReadonlyArray<readonly [string, unknown]> = entries.map((entry) => {
      if (!Array.isArray(entry) || entry.length !== 2 || typeof entry[0] !== "string") {
        throw new NativeSchemaError(
          "rejected",
          "invalid-request",
          "each entry must be a [key, value] pair",
        );
      }
      return Object.freeze([entry[0], entry[1]] as const);
    });
    const bytes = JSON.stringify(pairs).length;
    if (bytes > limits.maxObserveBytes) {
      throw new NativeSchemaError("rejected", "resource-limit", "entries exceed the byte cap");
    }
    return { raw: Object.freeze([...pairs]), descriptor: null, bytes };
  }
  if (tag !== "hostile") {
    throw new NativeSchemaError(
      "rejected",
      "invalid-request",
      "hostile_descriptor payload needs tag hostile",
    );
  }
  const descriptor = record["descriptor"];
  if (!isHostileDescriptor(descriptor)) {
    throw new NativeSchemaError(
      "rejected",
      "unsupported-capability",
      `unknown hostile descriptor: ${String(descriptor)}`,
    );
  }
  return { raw: createHostileValue(descriptor, hostile), descriptor, bytes: 0 };
}

// ---------------------------------------------------------------------------
// Inert envelope transport (pure: handles and facts only)
// ---------------------------------------------------------------------------

export const MAX_ENVELOPE_BYTES = 65536;

const ENVELOPE_KEYS = ["schema_version", "handle", "facts"] as const;

export type SessionEnvelopeFacts = Readonly<Record<string, unknown>>;

// Encode one handle plus its inert facts as a JSON string. The handle is
// encoded from its authority fields only; the raw value behind it is never
// dereferenced, so hostile getters, `then`, and Proxy traps cannot fire.
export function encodeEnvelope(handle: NativeHandle, facts: SessionEnvelopeFacts): string {
  if (facts === null || typeof facts !== "object" || Array.isArray(facts)) {
    throw new NativeSchemaError("rejected", "invalid-request", "envelope facts must be an object");
  }
  checkFactsInert(facts, "envelope facts");
  const wire: NativeHandleWire = encodeHandleWire(handle);
  checkFactsInert(wire, "envelope handle");
  const json = JSON.stringify({
    schema_version: NATIVE_SESSION_SCHEMA_VERSION,
    handle: wire,
    facts,
  });
  if (json.length > MAX_ENVELOPE_BYTES) {
    throw new NativeSchemaError("rejected", "resource-limit", "envelope exceeds the byte cap");
  }
  return json;
}

export function decodeEnvelope(json: unknown): {
  handle: NativeHandle;
  facts: SessionEnvelopeFacts;
} {
  if (typeof json !== "string") {
    throw new NativeSchemaError("rejected", "invalid-request", "envelope must be a string");
  }
  if (json.length > MAX_ENVELOPE_BYTES) {
    throw new NativeSchemaError("rejected", "resource-limit", "envelope exceeds the byte cap");
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(json);
  } catch {
    throw new NativeSchemaError("rejected", "invalid-request", "envelope is not JSON");
  }
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new NativeSchemaError("rejected", "invalid-request", "envelope must be an object");
  }
  const record = parsed as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    if (!(ENVELOPE_KEYS as readonly string[]).includes(key)) {
      throw new NativeSchemaError("rejected", "invalid-request", `unknown envelope field: ${key}`);
    }
  }
  if (record["schema_version"] !== NATIVE_SESSION_SCHEMA_VERSION) {
    throw new NativeSchemaError("rejected", "invalid-request", "unsupported envelope version");
  }
  const handle = decodeHandleWire(record["handle"]);
  const facts = record["facts"];
  if (facts === null || typeof facts !== "object" || Array.isArray(facts)) {
    throw new NativeSchemaError("rejected", "invalid-request", "envelope facts must be an object");
  }
  checkFactsInert(facts, "envelope facts");
  return { handle, facts: facts as SessionEnvelopeFacts };
}

// ---------------------------------------------------------------------------
// Service: sessions own cells; handles never carry raw values
// ---------------------------------------------------------------------------

export class NativeSessionService {
  private readonly registry: NativeHandleRegistry;
  private readonly limits: NativeValueLimits;
  private readonly cells = new Map<string, ValueCell>();
  private readonly sessions = new Map<string, SessionCounts>();
  private readonly closedSessions = new Set<string>();
  private cellCounter = 0;

  constructor(limits: NativeValueLimits) {
    this.limits = limits;
    this.registry = new NativeHandleRegistry(limits);
  }

  get sessionCount(): number {
    return this.registry.sessionCount;
  }

  get liveCellCount(): number {
    return this.cells.size;
  }

  open(owner: string, sessionId: string): NativeHandle<"session"> {
    const session = this.registry.openSession(owner, sessionId);
    this.sessions.set(sessionId, { makes: 0, aliases: 0, hostile: zeroHostileCounters() });
    return session;
  }

  make(
    session: NativeHandle,
    owner: string,
    kind: unknown,
    payload: unknown,
  ): { handle: NativeHandle<"value">; facts: MakeFacts } {
    const live = this.requireSessionHandle(session, owner);
    if (!isMakeKind(kind)) {
      throw new NativeSchemaError(
        "rejected",
        "unsupported-capability",
        `unknown make kind: ${String(kind)}`,
      );
    }
    const counts = this.requireCounts(live.sessionId);
    const built = buildRawValue(kind, payload, counts.hostile, this.limits);
    const handle = this.registry.mint(live, owner, "value");
    this.cellCounter += 1;
    const cell: ValueCell = {
      cellId: `cell${this.cellCounter}`,
      sessionId: live.sessionId,
      kind,
      descriptor: built.descriptor,
      bytes: built.bytes,
      raw: built.raw,
    };
    this.cells.set(handle.id, cell);
    counts.makes += 1;
    return { handle, facts: this.cellFacts(cell, null) };
  }

  // Mint a second handle to the same cell. The two handles share one cell
  // id; no raw value is read or copied.
  alias(
    session: NativeHandle,
    owner: string,
    value: NativeHandle,
  ): { handle: NativeHandle<"value">; facts: MakeFacts } {
    const live = this.requireSessionHandle(session, owner);
    if (value.kind !== "value") {
      throw new NativeSchemaError("rejected", "invalid-request", "alias needs a value handle");
    }
    this.registry.use(value, owner);
    if (value.sessionId !== live.sessionId) {
      throw new NativeSchemaError(
        "rejected",
        "wrong-owner",
        "value handle is bound to a different session",
      );
    }
    const cell = this.cells.get(value.id);
    if (!cell) {
      throw new NativeSchemaError("rejected", "stale-handle", "unknown value cell");
    }
    const handle = this.registry.mint(live, owner, "value");
    this.cells.set(handle.id, cell);
    this.requireCounts(live.sessionId).aliases += 1;
    return { handle, facts: this.cellFacts(cell, value.id) };
  }

  counters(session: NativeHandle, owner: string): SessionCounters {
    const live = this.requireSessionHandle(session, owner);
    const counts = this.requireCounts(live.sessionId);
    return Object.freeze({
      makes: counts.makes,
      aliases: counts.aliases,
      getterReads: counts.hostile.getterReads,
      thenCalls: counts.hostile.thenCalls,
      proxyTraps: counts.hostile.proxyTraps,
    });
  }

  close(session: NativeHandle, owner: string): SessionCloseReceipt {
    if (session.kind !== "session") {
      throw new NativeSchemaError("rejected", "invalid-request", "close needs a session handle");
    }
    const receipt = this.registry.closeSession(session as NativeHandle<"session">, owner);
    let cellsReleased = 0;
    for (const [id, cell] of this.cells) {
      if (cell.sessionId === session.sessionId) {
        this.cells.delete(id);
        cellsReleased += 1;
      }
    }
    if (!this.closedSessions.has(session.sessionId)) {
      this.closedSessions.add(session.sessionId);
      this.sessions.delete(session.sessionId);
    }
    return Object.freeze({ ...receipt, cellsReleased });
  }

  private requireSessionHandle(session: NativeHandle, owner: string): NativeHandle<"session"> {
    if (session.kind !== "session") {
      throw new NativeSchemaError("rejected", "invalid-request", "needs a session handle");
    }
    this.registry.use(session, owner);
    return session as NativeHandle<"session">;
  }

  private requireCounts(sessionId: string): SessionCounts {
    const counts = this.sessions.get(sessionId);
    if (!counts) {
      throw new NativeSchemaError("rejected", "stale-handle", "unknown session counters");
    }
    return counts;
  }

  private cellFacts(cell: ValueCell, aliasOf: string | null): MakeFacts {
    const facts: Record<string, unknown> = {
      cell: cell.cellId,
      kind: cell.kind,
      bytes: cell.bytes,
    };
    if (cell.descriptor !== null) {
      facts["descriptor"] = cell.descriptor;
    }
    if (aliasOf !== null) {
      facts["alias_of"] = aliasOf;
    }
    const frozen = Object.freeze(facts) as MakeFacts;
    checkFactsInert(frozen, "make facts");
    return frozen;
  }
}
