// K03: exact native observations and seals.
//
// Reads inert facts back out from behind opaque value handles: IEEE-754 bit
// facts (exact hi/lo words, class, sign, exponent, mantissa), integer lexemes,
// alias-identity facts, and explicitly counted effectful reads. Counter
// intervals slice the explicit-read log, and a final session seal binds every
// construction and observation fact into one digest; sealed sessions reject
// further reads and makes.
//
// The hostile rule: any getter/then/proxy touch is an explicit counted read
// op, never implicit. All non-read paths (make, alias, observe, counters,
// interval, seal) derive from stored inert construction literals and the
// handle table only; they never dereference a raw hostile value, so the K02
// zero-read guarantee holds for every non-read path. Only effectfulRead
// touches raw hostile values, inside one counted op with an inert outcome.
//
// Construction mirrors NativeSessionService exactly (same closed make/hostile
// vocabularies, same limits, same fixtures and counters) because K02 cells are
// private by design; this service additionally stores the inert construction
// literals so read-out is exact without touching raw values.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, or live runtimes. Local controls only.

import { createHash } from "node:crypto";
import {
  checkFactsInert,
  NativeSchemaError,
  type NativeHandle,
  type NativeValueLimits,
} from "./schema.ts";
import { NativeHandleRegistry, type CloseReceipt } from "./schema-handles.ts";
import {
  createHostileValue,
  NATIVE_HOSTILE_DESCRIPTORS,
  NATIVE_MAKE_KINDS,
  zeroHostileCounters,
  type HostileCounters,
  type NativeHostileDescriptor,
  type NativeMakeKind,
} from "./session.ts";

export const NATIVE_OBSERVE_SCHEMA_VERSION = "1" as const;

// ---------------------------------------------------------------------------
// Closed observe-kind and read-op vocabularies
// ---------------------------------------------------------------------------

export const NATIVE_OBSERVE_KINDS = [
  "scalar_tag",
  "ieee_bits",
  "lexeme",
  "descriptor",
  "entries",
  "identity",
  "counters",
] as const;
export type NativeObserveKind = (typeof NATIVE_OBSERVE_KINDS)[number];

export const NATIVE_READ_OPS = ["bits", "text", "entries", "property_then", "call_then"] as const;
export type NativeReadOp = (typeof NATIVE_READ_OPS)[number];

function isObserveKind(value: unknown): value is NativeObserveKind {
  return typeof value === "string" && (NATIVE_OBSERVE_KINDS as readonly string[]).includes(value);
}

function isReadOp(value: unknown): value is NativeReadOp {
  return typeof value === "string" && (NATIVE_READ_OPS as readonly string[]).includes(value);
}

function isMakeKind(value: unknown): value is NativeMakeKind {
  return typeof value === "string" && (NATIVE_MAKE_KINDS as readonly string[]).includes(value);
}

function isHostileDescriptor(value: unknown): value is NativeHostileDescriptor {
  return (
    typeof value === "string" && (NATIVE_HOSTILE_DESCRIPTORS as readonly string[]).includes(value)
  );
}

// ---------------------------------------------------------------------------
// Exact f64 bit facts (pure: DataView round-trips, never canonicalizes)
// ---------------------------------------------------------------------------

export type F64Bits = Readonly<{ hi: number; lo: number }>;

export function f64BitsOf(value: number): F64Bits {
  const view = new DataView(new ArrayBuffer(8));
  view.setFloat64(0, value);
  return { hi: view.getUint32(0), lo: view.getUint32(4) };
}

export function f64FromBits(hi: number, lo: number): number {
  const view = new DataView(new ArrayBuffer(8));
  view.setUint32(0, hi);
  view.setUint32(4, lo);
  return view.getFloat64(0);
}

export type F64Class = "zero" | "subnormal" | "normal" | "inf" | "nan";

export type F64Classification = Readonly<{
  class: F64Class;
  sign: number;
  exponent: number;
  mantissa_hi: number;
  mantissa_lo: number;
}>;

export function classifyF64(hi: number, lo: number): F64Classification {
  const sign = hi >>> 31;
  const exponent = (hi >>> 20) & 0x7ff;
  const mantissa_hi = hi & 0xfffff;
  const mantissa_lo = lo >>> 0;
  const mantissaZero = mantissa_hi === 0 && mantissa_lo === 0;
  let cls: F64Class = "normal";
  if (exponent === 0x7ff) {
    cls = mantissaZero ? "inf" : "nan";
  } else if (exponent === 0) {
    cls = mantissaZero ? "zero" : "subnormal";
  }
  return { class: cls, sign, exponent, mantissa_hi, mantissa_lo };
}

// ---------------------------------------------------------------------------
// Canonical JSON and the seal digest (local, deterministic)
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
  throw new NativeSchemaError("failed", "native-io", "seal input carries a non-inert value");
}

export function digestFacts(value: unknown): string {
  // No key-count scan here: the seal input is service-built from validated
  // inert literals, and no input value can escape through a one-way digest.
  // canonicalize still rejects non-inert leaves structurally.
  return `sha256:${createHash("sha256").update(canonicalize(value), "utf8").digest("hex")}`;
}

// ---------------------------------------------------------------------------
// Observe bounds and inert value tagging
// ---------------------------------------------------------------------------

export type ObserveBounds = Readonly<{ maxEntries: number; maxBytes: number }>;

const BOUND_KEYS = ["maxEntries", "maxBytes"] as const;

export function checkObserveBounds(value: unknown, limits: NativeValueLimits): ObserveBounds {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new NativeSchemaError("rejected", "invalid-request", "observe bounds must be an object");
  }
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    if (!(BOUND_KEYS as readonly string[]).includes(key)) {
      throw new NativeSchemaError("rejected", "invalid-request", `unknown bound field: ${key}`);
    }
  }
  const maxEntries = record["maxEntries"];
  const maxBytes = record["maxBytes"];
  if (!Number.isSafeInteger(maxEntries) || (maxEntries as number) <= 0) {
    throw new NativeSchemaError("rejected", "invalid-request", "maxEntries must be positive");
  }
  if (!Number.isSafeInteger(maxBytes) || (maxBytes as number) <= 0) {
    throw new NativeSchemaError("rejected", "invalid-request", "maxBytes must be positive");
  }
  if ((maxEntries as number) > limits.maxObserveEntries) {
    throw new NativeSchemaError("rejected", "resource-limit", "maxEntries exceeds the entry cap");
  }
  if ((maxBytes as number) > limits.maxObserveBytes) {
    throw new NativeSchemaError("rejected", "resource-limit", "maxBytes exceeds the byte cap");
  }
  return Object.freeze({ maxEntries, maxBytes }) as ObserveBounds;
}

// Exact tagging for one inert JSON-ish value. Safe integers (never -0) cross
// as lexemes; every other number crosses as exact f64 bits, so -0 vs 0 and
// rounding outcomes stay distinguishable. Nested values stay explicit gaps:
// the observer never deep-reads past one level.
export type TaggedValue = Readonly<Record<string, unknown>>;

export function tagNumber(value: number): TaggedValue {
  if (Number.isSafeInteger(value) && !Object.is(value, -0)) {
    return Object.freeze({ tag: "integer", lexeme: String(value) });
  }
  const { hi, lo } = f64BitsOf(value);
  return Object.freeze({ tag: "f64_bits", hi, lo, ...classifyF64(hi, lo) });
}

export function tagInertValue(value: unknown): TaggedValue {
  if (value === null) return Object.freeze({ tag: "null" });
  if (typeof value === "boolean") return Object.freeze({ tag: "boolean", value });
  if (typeof value === "string") return Object.freeze({ tag: "text", text: value });
  if (typeof value === "number") return tagNumber(value);
  if (typeof value === "undefined") return Object.freeze({ tag: "undefined" });
  return Object.freeze({ tag: "json", gap: "nested-opaque" });
}

// ---------------------------------------------------------------------------
// Fact shapes (all inert data)
// ---------------------------------------------------------------------------

export type ObserveCounters = Readonly<{
  makes: number;
  aliases: number;
  observations: number;
  explicitReads: number;
  sealed: boolean;
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

export type PerHandleObservation = Readonly<{
  handle: string;
  cell: string;
  result: Readonly<Record<string, unknown>>;
}>;

export type ObserveFacts = Readonly<{
  kind: NativeObserveKind;
  observations: readonly PerHandleObservation[];
  alias_groups?: readonly Readonly<{ cell: string; handles: readonly string[] }>[];
  counters?: ObserveCounters;
}>;

export type ReadFacts = Readonly<Record<string, unknown>>;

export type IntervalFacts = Readonly<{
  from: number;
  to: number;
  reads: readonly Readonly<{ seq: number; op: NativeReadOp; cell: string; handle: string }>[];
  hostileDelta: Readonly<{ getterReads: number; thenCalls: number; proxyTraps: number }>;
}>;

export type SealFacts = Readonly<{
  digest: string;
  session: string;
  cells: number;
  handles: number;
  makes: number;
  aliases: number;
  observations: number;
  explicitReads: number;
  joined: boolean;
}>;

export type ObserveCloseReceipt = CloseReceipt &
  Readonly<{
    cellsReleased: number;
  }>;

// ---------------------------------------------------------------------------
// Cells and sessions
// ---------------------------------------------------------------------------

type ConstructionRecord =
  | Readonly<{ tag: "f64_bits"; hi: number; lo: number }>
  | Readonly<{ tag: "text"; text: string }>
  | Readonly<{ tag: "entries"; entries: ReadonlyArray<readonly [string, unknown]> }>
  | Readonly<{ tag: "hostile"; descriptor: NativeHostileDescriptor }>;

type ObserveCell = {
  cellId: string;
  sessionId: string;
  kind: NativeMakeKind;
  descriptor: NativeHostileDescriptor | null;
  bytes: number;
  raw: unknown;
  record: ConstructionRecord;
};

type ReadLogEntry = {
  seq: number;
  op: NativeReadOp;
  cell: string;
  handle: string;
  hostile: HostileCounters;
};

type SessionObs = {
  makes: number;
  aliases: number;
  observations: number;
  explicitReads: number;
  hostile: HostileCounters;
  log: ReadLogEntry[];
  obsLog: Array<{ seq: number; kind: NativeObserveKind; handles: string[] }>;
  seal: SealFacts | null;
};

function scalarOf(cell: ObserveCell): string {
  if (cell.kind === "primitive_bits") return "f64";
  if (cell.kind === "text") return "text";
  if (cell.kind === "ordered_entries") return "entries";
  return "hostile";
}

function errorName(error: unknown): string {
  return error instanceof Error ? error.name : "unknown";
}

// ---------------------------------------------------------------------------
// Inert payload construction (mirrors K02; stores literals for exact read-out)
// ---------------------------------------------------------------------------

const MAX_COPY_DEPTH = 8;

function copyInert(value: unknown, depth: number): unknown {
  if (value === null) return null;
  const kind = typeof value;
  if (kind === "string" || kind === "number" || kind === "boolean" || kind === "undefined") {
    return value;
  }
  if (depth > MAX_COPY_DEPTH) {
    throw new NativeSchemaError("rejected", "invalid-request", "make payload nests too deeply");
  }
  if (Array.isArray(value)) {
    return Object.freeze(value.map((item) => copyInert(item, depth + 1)));
  }
  if (kind === "object") {
    const out: Record<string, unknown> = {};
    const record = value as Record<string, unknown>;
    for (const key of Object.keys(record)) {
      out[key] = copyInert(record[key], depth + 1);
    }
    return Object.freeze(out);
  }
  throw new NativeSchemaError("failed", "native-io", "make payload carries a non-inert value");
}

type BuiltCell = {
  raw: unknown;
  record: ConstructionRecord;
  descriptor: NativeHostileDescriptor | null;
  bytes: number;
};

function requirePayloadObject(payload: unknown): Record<string, unknown> {
  if (payload === null || typeof payload !== "object" || Array.isArray(payload)) {
    throw new NativeSchemaError("rejected", "invalid-request", "make payload must be an object");
  }
  return payload as Record<string, unknown>;
}

function buildCell(
  kind: NativeMakeKind,
  payload: unknown,
  hostile: HostileCounters,
  limits: NativeValueLimits,
): BuiltCell {
  const record = requirePayloadObject(payload);
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
    return {
      raw: f64FromBits(hi as number, lo as number),
      record: Object.freeze({ tag: "f64_bits", hi: hi as number, lo: lo as number }),
      descriptor: null,
      bytes: 8,
    };
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
    return {
      raw: text,
      record: Object.freeze({ tag: "text", text }),
      descriptor: null,
      bytes: text.length,
    };
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
      return Object.freeze([entry[0], copyInert(entry[1], 0)] as const);
    });
    const bytes = JSON.stringify(pairs).length;
    if (bytes > limits.maxObserveBytes) {
      throw new NativeSchemaError("rejected", "resource-limit", "entries exceed the byte cap");
    }
    const frozen = Object.freeze([...pairs]);
    return {
      raw: frozen,
      record: Object.freeze({ tag: "entries", entries: frozen }),
      descriptor: null,
      bytes,
    };
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
  return {
    raw: createHostileValue(descriptor, hostile),
    record: Object.freeze({ tag: "hostile", descriptor }),
    descriptor,
    bytes: 0,
  };
}

// ---------------------------------------------------------------------------
// Service: construction plus exact non-effectful observation
// ---------------------------------------------------------------------------

export class NativeObserveService {
  private readonly registry: NativeHandleRegistry;
  private readonly limits: NativeValueLimits;
  private readonly cells = new Map<string, ObserveCell>();
  private readonly sessions = new Map<string, SessionObs>();
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
    this.sessions.set(sessionId, {
      makes: 0,
      aliases: 0,
      observations: 0,
      explicitReads: 0,
      hostile: zeroHostileCounters(),
      log: [],
      obsLog: [],
      seal: null,
    });
    return session;
  }

  make(
    session: NativeHandle,
    owner: string,
    kind: unknown,
    payload: unknown,
  ): { handle: NativeHandle<"value">; facts: MakeFacts } {
    const live = this.requireWritableSession(session, owner);
    if (!isMakeKind(kind)) {
      throw new NativeSchemaError(
        "rejected",
        "unsupported-capability",
        `unknown make kind: ${String(kind)}`,
      );
    }
    const state = this.requireState(live.sessionId);
    const built = buildCell(kind, payload, state.hostile, this.limits);
    const handle = this.registry.mint(live, owner, "value");
    this.cellCounter += 1;
    const cell: ObserveCell = {
      cellId: `cell${this.cellCounter}`,
      sessionId: live.sessionId,
      kind,
      descriptor: built.descriptor,
      bytes: built.bytes,
      raw: built.raw,
      record: built.record,
    };
    this.cells.set(handle.id, cell);
    state.makes += 1;
    return { handle, facts: this.cellFacts(cell, null) };
  }

  // Mint a second handle to the same cell. The two handles share one cell
  // id; no raw value is read or copied.
  alias(
    session: NativeHandle,
    owner: string,
    value: NativeHandle,
  ): { handle: NativeHandle<"value">; facts: MakeFacts } {
    const live = this.requireWritableSession(session, owner);
    const cell = this.requireValueCell(value, owner, live.sessionId);
    const handle = this.registry.mint(live, owner, "value");
    this.cells.set(handle.id, cell);
    this.requireState(live.sessionId).aliases += 1;
    return { handle, facts: this.cellFacts(cell, value.id) };
  }

  counters(session: NativeHandle, owner: string): ObserveCounters {
    const live = this.requireReadableSession(session, owner);
    return this.snapshot(live.sessionId);
  }

  private requireReadableSession(session: NativeHandle, owner: string): NativeHandle<"session"> {
    if (session === null || typeof session !== "object" || session.kind !== "session") {
      throw new NativeSchemaError("rejected", "invalid-request", "needs a session handle");
    }
    this.registry.use(session, owner);
    return session as NativeHandle<"session">;
  }

  // make, alias, observe, and effectfulRead all reject once sealed.
  private requireWritableSession(session: NativeHandle, owner: string): NativeHandle<"session"> {
    const live = this.requireReadableSession(session, owner);
    if (this.requireState(live.sessionId).seal !== null) {
      throw new NativeSchemaError("rejected", "closed-handle", "session is sealed");
    }
    return live;
  }

  private requireState(sessionId: string): SessionObs {
    const state = this.sessions.get(sessionId);
    if (!state) {
      throw new NativeSchemaError("rejected", "stale-handle", "unknown session counters");
    }
    return state;
  }

  private requireValueCell(value: NativeHandle, owner: string, sessionId: string): ObserveCell {
    if (value === null || typeof value !== "object" || value.kind !== "value") {
      throw new NativeSchemaError("rejected", "invalid-request", "needs a value handle");
    }
    this.registry.use(value, owner);
    if (value.sessionId !== sessionId) {
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
    return cell;
  }

  private snapshot(sessionId: string): ObserveCounters {
    const state = this.requireState(sessionId);
    return Object.freeze({
      makes: state.makes,
      aliases: state.aliases,
      observations: state.observations,
      explicitReads: state.explicitReads,
      sealed: state.seal !== null,
      getterReads: state.hostile.getterReads,
      thenCalls: state.hostile.thenCalls,
      proxyTraps: state.hostile.proxyTraps,
    });
  }

  private cellFacts(cell: ObserveCell, aliasOf: string | null): MakeFacts {
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

  // Non-effectful observation. Derives every fact from stored inert
  // construction literals and the handle table; never dereferences a raw
  // value, so hostile getters, `then`, and Proxy traps cannot fire here.
  observe(
    session: NativeHandle,
    owner: string,
    handles: unknown,
    kind: unknown,
    bounds: unknown,
  ): ObserveFacts {
    const live = this.requireWritableSession(session, owner);
    if (!isObserveKind(kind)) {
      throw new NativeSchemaError(
        "rejected",
        "unsupported-capability",
        `unknown observe kind: ${String(kind)}`,
      );
    }
    const checked = checkObserveBounds(bounds, this.limits);
    const cells = this.requireObservationCells(handles, owner, live.sessionId);
    const state = this.requireState(live.sessionId);
    const snapshot = this.snapshot(live.sessionId);
    const observations = cells.map(({ handleId, cell }) =>
      Object.freeze({
        handle: handleId,
        cell: cell.cellId,
        result: this.observeOne(kind, cell, checked),
      }),
    );
    const facts: Record<string, unknown> = {
      kind,
      observations: Object.freeze(observations),
    };
    if (kind === "identity") {
      facts["alias_groups"] = this.aliasGroups(live.sessionId);
    }
    if (kind === "counters") {
      facts["counters"] = snapshot;
    }
    state.observations += 1;
    state.obsLog.push({
      seq: state.observations,
      kind,
      handles: observations.map((entry) => entry.handle),
    });
    const frozen = Object.freeze(facts) as ObserveFacts;
    checkFactsInert(frozen, "observe facts");
    return frozen;
  }

  private requireObservationCells(
    handles: unknown,
    owner: string,
    sessionId: string,
  ): Array<{ handleId: string; cell: ObserveCell }> {
    if (!Array.isArray(handles) || handles.length === 0) {
      throw new NativeSchemaError("rejected", "invalid-request", "observe needs a handle list");
    }
    if (handles.length > this.limits.maxObserveEntries) {
      throw new NativeSchemaError("rejected", "resource-limit", "observe handle list too long");
    }
    return handles.map((handle) => {
      const cell = this.requireValueCell(handle as NativeHandle, owner, sessionId);
      return { handleId: (handle as NativeHandle).id, cell };
    });
  }

  private observeOne(
    kind: NativeObserveKind,
    cell: ObserveCell,
    bounds: ObserveBounds,
  ): Readonly<Record<string, unknown>> {
    const scalar = scalarOf(cell);
    if (kind === "scalar_tag") {
      const result: Record<string, unknown> = { scalar };
      if (cell.descriptor !== null) result["descriptor"] = cell.descriptor;
      return Object.freeze(result);
    }
    if (kind === "descriptor") {
      const result: Record<string, unknown> = { scalar, bytes: cell.bytes };
      if (cell.descriptor !== null) result["descriptor"] = cell.descriptor;
      return Object.freeze(result);
    }
    if (kind === "identity") {
      return Object.freeze({ cell: cell.cellId });
    }
    if (kind === "counters") {
      return Object.freeze({ cell: cell.cellId });
    }
    if (kind === "ieee_bits") {
      if (cell.record.tag !== "f64_bits") {
        return Object.freeze({ gap: "not-a-float", scalar });
      }
      return Object.freeze({
        tag: "f64_bits",
        hi: cell.record.hi,
        lo: cell.record.lo,
        ...classifyF64(cell.record.hi, cell.record.lo),
      });
    }
    if (kind === "lexeme") {
      return this.lexemeOf(cell, bounds);
    }
    return this.entriesOf(cell, bounds);
  }

  private lexemeOf(cell: ObserveCell, bounds: ObserveBounds): Readonly<Record<string, unknown>> {
    const record = cell.record;
    if (record.tag === "f64_bits") {
      // The reconstructed value decides integer-vs-bits only; the reported
      // words are always the construction literals, so engine NaN
      // canonicalization (e.g. signaling bit quieting through a JS number)
      // can never move the observed bits.
      const value = f64FromBits(record.hi, record.lo);
      if (Number.isSafeInteger(value) && !Object.is(value, -0)) {
        return Object.freeze({ tag: "integer", lexeme: String(value) });
      }
      return Object.freeze({
        tag: "f64_bits",
        hi: record.hi,
        lo: record.lo,
        ...classifyF64(record.hi, record.lo),
      });
    }
    if (record.tag === "text") {
      if (record.text.length > bounds.maxBytes) {
        return Object.freeze({
          tag: "text",
          text: record.text.slice(0, bounds.maxBytes),
          gap: "byte-cap",
          omitted: record.text.length - bounds.maxBytes,
        });
      }
      return Object.freeze({ tag: "text", text: record.text });
    }
    if (record.tag === "entries") {
      return this.taggedPairs(record.entries, bounds);
    }
    return Object.freeze({ gap: "effectful", scalar: "hostile", descriptor: cell.descriptor });
  }

  private entriesOf(cell: ObserveCell, bounds: ObserveBounds): Readonly<Record<string, unknown>> {
    if (cell.record.tag !== "entries") {
      if (cell.record.tag === "hostile") {
        return Object.freeze({ gap: "effectful", scalar: "hostile", descriptor: cell.descriptor });
      }
      return Object.freeze({ gap: "not-entries", scalar: scalarOf(cell) });
    }
    return this.taggedPairs(cell.record.entries, bounds);
  }

  private taggedPairs(
    pairs: ReadonlyArray<readonly [string, unknown]>,
    bounds: ObserveBounds,
  ): Readonly<Record<string, unknown>> {
    const tagged: Array<Readonly<Record<string, unknown>>> = [];
    let bytes = 2;
    let omitted = 0;
    let gap: string | null = null;
    for (const [key, value] of pairs) {
      if (tagged.length >= bounds.maxEntries) {
        gap = "entry-cap";
        omitted = pairs.length - tagged.length;
        break;
      }
      const entry = Object.freeze({ key, value: tagInertValue(value) });
      const size = JSON.stringify(entry).length + 1;
      if (bytes + size > bounds.maxBytes) {
        gap = "byte-cap";
        omitted = pairs.length - tagged.length;
        break;
      }
      bytes += size;
      tagged.push(entry);
    }
    const result: Record<string, unknown> = { entries: Object.freeze(tagged) };
    if (gap !== null) {
      result["gap"] = gap;
      result["omitted"] = omitted;
    }
    return Object.freeze(result);
  }

  private aliasGroups(
    sessionId: string,
  ): readonly Readonly<{ cell: string; handles: readonly string[] }>[] {
    const byCell = new Map<string, string[]>();
    for (const [handleId, cell] of this.cells) {
      if (cell.sessionId !== sessionId) continue;
      const group = byCell.get(cell.cellId);
      if (group) {
        group.push(handleId);
      } else {
        byCell.set(cell.cellId, [handleId]);
      }
    }
    const groups: Array<Readonly<{ cell: string; handles: readonly string[] }>> = [];
    for (const [cellId, handleIds] of byCell) {
      if (handleIds.length < 2) continue;
      groups.push(Object.freeze({ cell: cellId, handles: Object.freeze([...handleIds]) }));
    }
    groups.sort((a, b) => (a.cell < b.cell ? -1 : 1));
    return Object.freeze(groups);
  }

  // The ONLY path that touches raw values. Each call performs exactly one
  // counted read op against one cell and appends one log entry carrying the
  // hostile counters after the touch, so intervals can report exact deltas.
  effectfulRead(session: NativeHandle, owner: string, handle: unknown, op: unknown): ReadFacts {
    const live = this.requireWritableSession(session, owner);
    if (!isReadOp(op)) {
      throw new NativeSchemaError(
        "rejected",
        "unsupported-capability",
        `unknown read op: ${String(op)}`,
      );
    }
    const cell = this.requireValueCell(handle as NativeHandle, owner, live.sessionId);
    const state = this.requireState(live.sessionId);
    const outcome = this.performRead(op, cell);
    state.explicitReads += 1;
    const seq = state.explicitReads;
    state.log.push({
      seq,
      op,
      cell: cell.cellId,
      handle: (handle as NativeHandle).id,
      hostile: { ...state.hostile },
    });
    const facts = Object.freeze({
      op,
      handle: (handle as NativeHandle).id,
      cell: cell.cellId,
      seq,
      ...outcome,
    });
    checkFactsInert(facts, "read facts");
    return facts;
  }

  private performRead(op: NativeReadOp, cell: ObserveCell): Record<string, unknown> {
    if (op === "bits" || op === "text" || op === "entries") {
      return this.performScalarRead(op, cell);
    }
    if (cell.descriptor === null) {
      throw new NativeSchemaError(
        "rejected",
        "invalid-request",
        `read op ${op} needs a hostile cell`,
      );
    }
    if (op === "property_then") {
      return this.readThenProperty(cell);
    }
    return this.callThen(cell);
  }

  // Scalar reads re-derive facts from the raw value behind the handle and
  // cross-check f64 words against the construction literals. Raws here are
  // primitives or frozen inert pairs, so no user code can run.
  private performScalarRead(op: NativeReadOp, cell: ObserveCell): Record<string, unknown> {
    if (op === "bits") {
      if (cell.record.tag !== "f64_bits" || typeof cell.raw !== "number") {
        throw new NativeSchemaError(
          "rejected",
          "invalid-request",
          "read op bits needs an f64 cell",
        );
      }
      const { hi, lo } = f64BitsOf(cell.raw);
      return {
        outcome: "returned",
        tag: "f64_bits",
        hi,
        lo,
        ...classifyF64(hi, lo),
        matches_construction: hi === cell.record.hi && lo === cell.record.lo,
      };
    }
    if (op === "text") {
      if (cell.record.tag !== "text" || typeof cell.raw !== "string") {
        throw new NativeSchemaError(
          "rejected",
          "invalid-request",
          "read op text needs a text cell",
        );
      }
      return { outcome: "returned", tag: "text", text: cell.raw };
    }
    if (cell.record.tag !== "entries" || !Array.isArray(cell.raw)) {
      throw new NativeSchemaError(
        "rejected",
        "invalid-request",
        "read op entries needs an entries cell",
      );
    }
    const tagged = (cell.raw as ReadonlyArray<readonly [string, unknown]>).map(([key, value]) =>
      Object.freeze({ key, value: tagInertValue(value) }),
    );
    return { outcome: "returned", tag: "entries", entries: Object.freeze(tagged) };
  }

  // One explicit property touch. The getter-backed fixture fires here; the
  // throwing thenable yields its (uncalled) function, reported opaque; the
  // revoked proxy throws before any trap runs.
  private readThenProperty(cell: ObserveCell): Record<string, unknown> {
    try {
      const then = (cell.raw as Record<string, unknown>)["then"];
      if (typeof then === "function") {
        return { outcome: "returned", typeof: "function", gap: "callable-opaque" };
      }
      return { outcome: "returned", typeof: typeof then };
    } catch (error) {
      return { outcome: "threw", name: errorName(error) };
    }
  }

  // One explicit assimilation attempt. Only the throwing thenable actually
  // calls: the getter fixture has nothing callable, and the revoked proxy
  // throws on touch.
  private callThen(cell: ObserveCell): Record<string, unknown> {
    let then: unknown;
    try {
      then = (cell.raw as Record<string, unknown>)["then"];
    } catch (error) {
      return { outcome: "threw", name: errorName(error) };
    }
    if (typeof then !== "function") {
      return { outcome: "not-callable", typeof: typeof then };
    }
    try {
      (cell.raw as { then: () => unknown }).then();
      return { outcome: "returned", typeof: "undefined" };
    } catch (error) {
      return { outcome: "threw", name: errorName(error) };
    }
  }

  // Slice the explicit-read log over (from, to] with the hostile-counter
  // delta across exactly that range. Read-only: works after the seal.
  interval(session: NativeHandle, owner: string, from: unknown, to?: unknown): IntervalFacts {
    const live = this.requireReadableSession(session, owner);
    const state = this.requireState(live.sessionId);
    const end = to === undefined ? state.explicitReads : to;
    if (!Number.isSafeInteger(from) || (from as number) < 0) {
      throw new NativeSchemaError("rejected", "invalid-request", "interval from must be >= 0");
    }
    if (!Number.isSafeInteger(end) || (end as number) < (from as number)) {
      throw new NativeSchemaError("rejected", "invalid-request", "interval to must cover from");
    }
    if ((end as number) > state.explicitReads) {
      throw new NativeSchemaError("rejected", "invalid-request", "interval to is in the future");
    }
    if ((end as number) - (from as number) > this.limits.maxObserveEntries) {
      throw new NativeSchemaError("rejected", "resource-limit", "interval range too wide");
    }
    const reads = state.log
      .filter((entry) => entry.seq > (from as number) && entry.seq <= (end as number))
      .map((entry) =>
        Object.freeze({ seq: entry.seq, op: entry.op, cell: entry.cell, handle: entry.handle }),
      );
    const before =
      (from as number) === 0
        ? zeroHostileCounters()
        : (state.log[(from as number) - 1]?.hostile ?? zeroHostileCounters());
    const after =
      (end as number) === 0
        ? zeroHostileCounters()
        : (state.log[(end as number) - 1]?.hostile ?? zeroHostileCounters());
    const facts = Object.freeze({
      from: from as number,
      to: end as number,
      reads: Object.freeze(reads),
      hostileDelta: Object.freeze({
        getterReads: after.getterReads - before.getterReads,
        thenCalls: after.thenCalls - before.thenCalls,
        proxyTraps: after.proxyTraps - before.proxyTraps,
      }),
    });
    checkFactsInert(facts, "interval facts");
    return facts;
  }

  // Bind every construction and observation fact into one digest and freeze
  // the session: afterwards make, alias, observe, and effectfulRead reject.
  // counters and interval stay readable; reseal joins with the same digest.
  seal(session: NativeHandle, owner: string): SealFacts {
    const live = this.requireReadableSession(session, owner);
    const state = this.requireState(live.sessionId);
    if (state.seal !== null) {
      return Object.freeze({ ...state.seal, joined: true });
    }
    const cells: Array<Record<string, unknown>> = [];
    let handles = 0;
    for (const [handleId, cell] of this.cells) {
      if (cell.sessionId !== live.sessionId) continue;
      handles += 1;
      cells.push({
        handle: handleId,
        cell: cell.cellId,
        kind: cell.kind,
        bytes: cell.bytes,
        record: cell.record,
      });
    }
    cells.sort((a, b) => (String(a["handle"]) < String(b["handle"]) ? -1 : 1));
    const digest = digestFacts({
      schema_version: NATIVE_OBSERVE_SCHEMA_VERSION,
      session: live.sessionId,
      cells,
      reads: state.log,
      observations: state.obsLog,
      counters: {
        makes: state.makes,
        aliases: state.aliases,
        observations: state.observations,
        explicitReads: state.explicitReads,
      },
      hostile: { ...state.hostile },
    });
    const seal: SealFacts = Object.freeze({
      digest,
      session: live.sessionId,
      cells: new Set(cells.map((cell) => cell["cell"])).size,
      handles,
      makes: state.makes,
      aliases: state.aliases,
      observations: state.observations,
      explicitReads: state.explicitReads,
      joined: false,
    });
    checkFactsInert(seal, "seal facts");
    state.seal = seal;
    return seal;
  }

  close(session: NativeHandle, owner: string): ObserveCloseReceipt {
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
}
