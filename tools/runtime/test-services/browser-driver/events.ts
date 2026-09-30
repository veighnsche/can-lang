// K09: browser event capture and terminal seal.
//
// A capture service over in-memory doubles only: no browser, no process,
// no flags, no environment, no sampling, no I/O. The service binds owned
// contexts, opens context/page/navigation watches with opaque minted
// tokens, captures request/response/body-complete/navigation/capture-error
// events through complete body/error callbacks, records explicit gaps,
// serves cursor reads and durable checkpoints, and seals each watch with
// a terminal receipt. Only a sealed gapless interval with zero observed
// events proves absence: a pending capture, a gap, or a quiet (open or
// late-attached) prefix never proves absence.
//
// Independence scope from the launch/driver service under test (normative):
//   SHARES: nothing executable. The service never imports, calls, or reads
//   driver code, driver-decoded values, receipts, flags, environment
//   variables, or process state. The only strings that may coincide are
//   the context/watch/capture names under test (supplied by the test, not
//   by the driver) and the vocabulary this file uses for facts. Facts
//   carry digests only: raw body bytes never cross.
//   PROVES: context admission, watch attestation, ordered event capture
//   with explicit repeated-header and decoding scope, pending-capture
//   accounting, gap marking, cursor/checkpoint discipline, the terminal
//   seal, and error provenance (every failure names its layer). Driver
//   claims are compared AGAINST these facts; the service never derives a
//   fact FROM a driver claim.
//   Consequently a pending capture, a gap, or a quiet prefix can never
//   prove absence: only a sealed gapless interval with zero observed
//   events proves it.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, browsers, processes, or live runtimes. Local
// controls only. Live host-dependent controls wait for the corresponding
// qualified profile and Q task.

import { createHash, randomBytes } from "node:crypto";

export const BROWSER_EVENT_SCHEMA_VERSION = "1" as const;

// ---------------------------------------------------------------------------
// Independence scope (normative)
// ---------------------------------------------------------------------------

export const BROWSER_EVENT_INDEPENDENCE_SCOPE: string =
  "independent: shares no driver code, decoded values, receipts, flags, " +
  "environment, or process state; proves context admission, watch " +
  "attestation, ordered capture with explicit repeated-header and decoding " +
  "scope, pending accounting, gaps, checkpoints, the terminal seal, and " +
  "layered error provenance from its own seeded doubles alone";

// ---------------------------------------------------------------------------
// Layers, codes, and the layered error
// ---------------------------------------------------------------------------

export const BROWSER_EVENT_LAYERS = ["capture", "watch", "seal"] as const;
export type BrowserEventLayer = (typeof BROWSER_EVENT_LAYERS)[number];

// Closed failure-code vocabulary. Every code maps to exactly one layer:
//   capture: context admission, ownership, capacity, lifecycle.
//   watch: watch binding, tokens, targets, events, captures, header and
//     decoding scope, cursors.
//   seal: checkpoints, the terminal seal, and absence-proof admission.
export const BROWSER_EVENT_CODES = [
  "unknown-context",
  "no-context",
  "wrong-owner",
  "capacity-exhausted",
  "context-closed",
  "forged-token",
  "unknown-watch",
  "watch-closed",
  "unknown-target",
  "unknown-event",
  "capture-pending",
  "capture-unknown",
  "unknown-header-scope",
  "unknown-decoding",
  "cursor-unknown",
  "checkpoint-stale",
  "already-sealed",
  "seal-open",
  "forbidden-proof",
] as const;
export type BrowserEventCode = (typeof BROWSER_EVENT_CODES)[number];

const CODE_LAYER: Readonly<Record<BrowserEventCode, BrowserEventLayer>> = {
  "unknown-context": "capture",
  "no-context": "capture",
  "wrong-owner": "capture",
  "capacity-exhausted": "capture",
  "context-closed": "capture",
  "forged-token": "watch",
  "unknown-watch": "watch",
  "watch-closed": "watch",
  "unknown-target": "watch",
  "unknown-event": "watch",
  "capture-pending": "watch",
  "capture-unknown": "watch",
  "unknown-header-scope": "watch",
  "unknown-decoding": "watch",
  "cursor-unknown": "watch",
  "checkpoint-stale": "seal",
  "already-sealed": "seal",
  "seal-open": "seal",
  "forbidden-proof": "seal",
};

export function layerOfEventCode(code: BrowserEventCode): BrowserEventLayer {
  return CODE_LAYER[code];
}

export class BrowserEventError extends Error {
  readonly layer: BrowserEventLayer;
  readonly code: BrowserEventCode;
  constructor(code: BrowserEventCode, message: string) {
    super(`[${CODE_LAYER[code]}:${code}] ${message}`);
    this.name = "BrowserEventError";
    this.layer = CODE_LAYER[code];
    this.code = code;
  }
}

function fail(code: BrowserEventCode, message: string): never {
  throw new BrowserEventError(code, message);
}

// ---------------------------------------------------------------------------
// Bounds and names
// ---------------------------------------------------------------------------

export type BrowserEventLimits = Readonly<{
  maxContexts: number;
  maxWatches: number;
  maxEventsPerWatch: number;
  maxPendingPerWatch: number;
}>;

const LIMIT_KEYS = [
  "maxContexts",
  "maxWatches",
  "maxEventsPerWatch",
  "maxPendingPerWatch",
] as const;

export function checkEventLimits(value: unknown): BrowserEventLimits {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new BrowserEventError("unknown-context", "limits must be an object");
  }
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    if (!(LIMIT_KEYS as readonly string[]).includes(key)) {
      throw new BrowserEventError("unknown-context", `unknown limit field: ${key}`);
    }
  }
  const out: Record<string, number> = {};
  for (const key of LIMIT_KEYS) {
    const entry = record[key];
    if (!Number.isSafeInteger(entry) || (entry as number) <= 0) {
      throw new BrowserEventError("unknown-context", `${key} must be a positive integer`);
    }
    out[key] = entry as number;
  }
  return Object.freeze(out) as BrowserEventLimits;
}

// Finite watch targets: every watch observes exactly one.
export const WATCH_TARGETS = ["context", "page", "navigation"] as const;
export type WatchTarget = (typeof WATCH_TARGETS)[number];

// Finite captured-event vocabulary. Gap entries are derived: they arise
// only from noteGap, never from direct emission.
export const EVENT_KINDS = [
  "request",
  "response",
  "body-complete",
  "navigation",
  "capture-error",
  "gap",
] as const;
export type EventKind = (typeof EVENT_KINDS)[number];

// Directly emitted kinds: gap is derived and never emitted.
export const EMITTED_KINDS = [
  "request",
  "response",
  "body-complete",
  "navigation",
  "capture-error",
] as const;

// Finite request-method vocabulary.
export const REQUEST_METHODS = [
  "GET",
  "POST",
  "PUT",
  "DELETE",
  "HEAD",
  "OPTIONS",
  "PATCH",
] as const;
export type RequestMethod = (typeof REQUEST_METHODS)[number];

// Explicit repeated-header scope: how repeated response headers folded.
// There is no default; every response states its scope.
export const HEADER_SCOPES = ["single", "repeated-joined", "repeated-list"] as const;
export type HeaderScope = (typeof HEADER_SCOPES)[number];

// Explicit decoding scope: how captured body bytes decode. There is no
// default; every response and body completion states its scope.
export const DECODING_SCOPES = ["raw-bytes", "utf8-strict", "utf8-replace"] as const;
export type DecodingScope = (typeof DECODING_SCOPES)[number];

// Finite gap reasons. late-attach marks a quiet prefix the watch never
// observed: the capture started before the watch attached.
export const GAP_REASONS = ["dropped-events", "late-attach", "observer-overrun"] as const;
export type GapReason = (typeof GAP_REASONS)[number];

// Finite capture-error reasons for the error callback.
export const CAPTURE_ERRORS = ["aborted", "timeout", "reset", "refused"] as const;
export type CaptureErrorReason = (typeof CAPTURE_ERRORS)[number];

// Finite absence subjects: what a sealed interval may claim absent.
export const ABSENCE_SUBJECTS = [
  "request",
  "response",
  "navigation",
  "capture-error",
  "any",
] as const;
export type AbsenceSubject = (typeof ABSENCE_SUBJECTS)[number];

const MAX_NAME_LEN = 128;
const BODY_DIGEST_RE = /^sha256:[0-9a-f]{64}$/;

function checkName(value: unknown, what: string, code: BrowserEventCode): string {
  if (typeof value !== "string" || value === "" || value.length > MAX_NAME_LEN) {
    fail(code, `${what} must be a non-empty name of at most ${MAX_NAME_LEN} chars`);
  }
  if (!/^[A-Za-z0-9][A-Za-z0-9_.:-]*$/.test(value)) {
    fail(code, `${what} carries illegal characters: ${value}`);
  }
  return value;
}

function checkOwner(value: unknown): string {
  return checkName(value, "owner", "wrong-owner");
}

function checkContextName(value: unknown): string {
  return checkName(value, "context", "unknown-context");
}

function checkWatchName(value: unknown): string {
  return checkName(value, "watch", "unknown-watch");
}

function checkCaptureName(value: unknown): string {
  return checkName(value, "capture", "capture-unknown");
}

function checkTarget(value: unknown): WatchTarget {
  if (typeof value !== "string" || !(WATCH_TARGETS as readonly string[]).includes(value)) {
    fail("unknown-target", `not a declared watch target: ${String(value)}`);
  }
  return value as WatchTarget;
}

function checkMethod(value: unknown): RequestMethod {
  if (typeof value !== "string" || !(REQUEST_METHODS as readonly string[]).includes(value)) {
    fail("unknown-event", `not a declared request method: ${String(value)}`);
  }
  return value as RequestMethod;
}

function checkStatus(value: unknown): number {
  if (!Number.isSafeInteger(value) || (value as number) < 100 || (value as number) > 599) {
    fail("unknown-event", `status must be an integer 100-599, got: ${String(value)}`);
  }
  return value as number;
}

function checkHeaderScope(value: unknown): HeaderScope {
  if (typeof value !== "string" || !(HEADER_SCOPES as readonly string[]).includes(value)) {
    fail("unknown-header-scope", `repeated-header scope must be explicit, got: ${String(value)}`);
  }
  return value as HeaderScope;
}

function checkDecodingScope(value: unknown): DecodingScope {
  if (typeof value !== "string" || !(DECODING_SCOPES as readonly string[]).includes(value)) {
    fail("unknown-decoding", `decoding scope must be explicit, got: ${String(value)}`);
  }
  return value as DecodingScope;
}

function checkGapReason(value: unknown): GapReason {
  if (typeof value !== "string" || !(GAP_REASONS as readonly string[]).includes(value)) {
    fail("unknown-event", `not a declared gap reason: ${String(value)}`);
  }
  return value as GapReason;
}

function checkCaptureError(value: unknown): CaptureErrorReason {
  if (typeof value !== "string" || !(CAPTURE_ERRORS as readonly string[]).includes(value)) {
    fail("unknown-event", `not a declared capture error: ${String(value)}`);
  }
  return value as CaptureErrorReason;
}

function checkAbsenceSubject(value: unknown): AbsenceSubject {
  if (typeof value !== "string" || !(ABSENCE_SUBJECTS as readonly string[]).includes(value)) {
    fail("forbidden-proof", `not a declared absence subject: ${String(value)}`);
  }
  return value as AbsenceSubject;
}

function checkDescriptor(value: unknown, what: string): Record<string, unknown> {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail("unknown-event", `${what} must be an object`);
  }
  return value as Record<string, unknown>;
}

function checkNoExtraKeys(
  record: Record<string, unknown>,
  keys: readonly string[],
  what: string,
): void {
  for (const key of Object.keys(record)) {
    if (!keys.includes(key)) {
      fail("unknown-event", `${what} carries an unknown field: ${key}`);
    }
  }
}

// ---------------------------------------------------------------------------
// Facts (frozen, digest-only)
// ---------------------------------------------------------------------------

export type ContextBindingFacts = Readonly<{
  owner: string;
  context: string;
  handleDigest: string;
}>;

export type WatchAttestation = Readonly<{
  watchId: string;
  owner: string;
  context: string;
  target: WatchTarget;
  contextDigest: string;
  handleDigest: string;
  cursor: number;
}>;

export type RequestEventFacts = Readonly<{
  seq: number;
  kind: "request";
  captureId: string;
  method: RequestMethod;
  target: string;
  digest: string;
}>;

export type ResponseEventFacts = Readonly<{
  seq: number;
  kind: "response";
  captureId: string;
  status: number;
  headerScope: HeaderScope;
  decodingScope: DecodingScope;
  digest: string;
}>;

export type BodyEventFacts = Readonly<{
  seq: number;
  kind: "body-complete";
  captureId: string;
  byteLength: number;
  bodyDigest: string;
  decodingScope: DecodingScope;
  digest: string;
}>;

export type NavigationEventFacts = Readonly<{
  seq: number;
  kind: "navigation";
  navId: string;
  from: string;
  to: string;
  digest: string;
}>;

export type CaptureErrorEventFacts = Readonly<{
  seq: number;
  kind: "capture-error";
  captureId: string;
  reason: CaptureErrorReason;
  digest: string;
}>;

export type GapEventFacts = Readonly<{
  seq: number;
  kind: "gap";
  dropped: number;
  reason: GapReason;
  digest: string;
}>;

export type EventFacts =
  | RequestEventFacts
  | ResponseEventFacts
  | BodyEventFacts
  | NavigationEventFacts
  | CaptureErrorEventFacts
  | GapEventFacts;

export type CursorReadFacts = Readonly<{
  watchId: string;
  cursor: number;
  nextCursor: number;
  events: readonly EventFacts[];
  digest: string;
}>;

export type CheckpointFacts = Readonly<{
  watchId: string;
  cursor: number;
  events: number;
  digest: string;
}>;

export type IntervalFacts = Readonly<{
  watchId: string;
  owner: string;
  context: string;
  target: WatchTarget;
  events: readonly EventFacts[];
  pending: readonly string[];
  gapped: boolean;
  sealed: boolean;
  checkpoint: number;
  digest: string;
}>;

export type SealReceipt = Readonly<{
  watchId: string;
  owner: string;
  context: string;
  target: WatchTarget;
  events: number;
  gapped: boolean;
  checkpoint: number;
  digest: string;
}>;

// An absence proof claim. Only sealed-interval is admissible; every other
// kind rejects with forbidden-proof before any verdict is read.
export const FORBIDDEN_ABSENCE_KINDS = [
  "pending-capture",
  "open-watch",
  "quiet-prefix",
  "event-log",
] as const;
export type ForbiddenAbsenceKind = (typeof FORBIDDEN_ABSENCE_KINDS)[number];

export type AbsenceProofClaim =
  | Readonly<{ kind: "sealed-interval"; interval: IntervalFacts; absenceOf: AbsenceSubject }>
  | Readonly<{ kind: ForbiddenAbsenceKind; detail: string }>;

export type AbsenceVerdict = Readonly<{
  watchId: string;
  absenceOf: AbsenceSubject;
  absent: boolean;
  gapped: boolean;
  digest: string;
}>;

export type RequestDescriptor = Readonly<{
  captureId: string;
  method: RequestMethod;
  target: string;
}>;

export type ResponseDescriptor = Readonly<{
  captureId: string;
  status: number;
  headerScope: HeaderScope;
  decodingScope: DecodingScope;
}>;

export type NavigationDescriptor = Readonly<{
  navId: string;
  from: string;
  to: string;
}>;

export type BodyDescriptor = Readonly<{
  byteLength: number;
  bodyDigest: string;
  decodingScope: DecodingScope;
}>;

export type GapDescriptor = Readonly<{
  dropped: number;
  reason: GapReason;
}>;

type PendingRecord = {
  captureId: string;
  requestSeq: number;
  responseSeq: number | null;
};

type WatchRecord = {
  watchId: string;
  owner: string;
  context: string;
  target: WatchTarget;
  token: string;
  events: EventFacts[];
  pending: Map<string, PendingRecord>;
  checkpoint: number;
  sealed: boolean;
  receipt: SealReceipt | null;
  // Well-formed occurrences refused for capacity reasons. Any count above
  // zero gaps the interval: the seal must never claim a complete log it
  // could not record.
  unrecordedDropped: number;
};

type ContextState = "open" | "closed";

type ContextRecord = {
  context: string;
  owner: string;
  token: string;
  state: ContextState;
  watches: Map<string, WatchRecord>;
};

function mintToken(prefix: string): string {
  return `${prefix}-${randomBytes(16).toString("hex")}`;
}

function digestText(text: string): string {
  return `sha256:${createHash("sha256").update(text, "utf8").digest("hex")}`;
}

function canonicalEvent(event: EventFacts): string {
  switch (event.kind) {
    case "request":
      return `request:${event.captureId}:${event.method}:${event.target}`;
    case "response":
      return `response:${event.captureId}:${event.status}:${event.headerScope}:${event.decodingScope}`;
    case "body-complete":
      return `body-complete:${event.captureId}:${event.byteLength}:${event.bodyDigest}:${event.decodingScope}`;
    case "navigation":
      return `navigation:${event.navId}:${event.from}:${event.to}`;
    case "capture-error":
      return `capture-error:${event.captureId}:${event.reason}`;
    case "gap":
      return `gap:${event.dropped}:${event.reason}`;
  }
}

// The interval digest is recomputed from carried events, pending captures,
// and flags, so assertAbsenceProved re-verifies a presented interval
// instead of trusting its digest string.
export function digestEventInterval(
  watchId: string,
  events: readonly EventFacts[],
  pending: readonly string[],
  flags: string,
): string {
  const hash = createHash("sha256");
  hash.update(watchId, "utf8");
  hash.update("\0", "utf8");
  for (const event of events) {
    hash.update(String(event.seq), "utf8");
    hash.update(":", "utf8");
    hash.update(canonicalEvent(event), "utf8");
    hash.update("\n", "utf8");
  }
  hash.update("\0", "utf8");
  for (const captureId of [...pending].sort()) {
    hash.update(captureId, "utf8");
    hash.update(";", "utf8");
  }
  hash.update("\0", "utf8");
  hash.update(flags, "utf8");
  return `sha256:${hash.digest("hex")}`;
}

// ---------------------------------------------------------------------------
// Service: owned contexts, watches, capture, checkpoint, terminal seal
// ---------------------------------------------------------------------------

export type ContextGrant = Readonly<{
  owner: string;
  context: string;
}>;

export class BrowserEventService {
  private readonly limits: BrowserEventLimits;
  private readonly grants = new Map<string, ContextGrant>();
  private readonly contexts = new Map<string, ContextRecord>();

  constructor(grants: readonly ContextGrant[], limits: BrowserEventLimits) {
    if (grants.length === 0) {
      throw new BrowserEventError("unknown-context", "declare at least one owned context");
    }
    if (grants.length > limits.maxContexts) {
      throw new BrowserEventError("capacity-exhausted", "declared contexts exceed the cap");
    }
    for (const grant of grants) {
      checkOwner(grant.owner);
      checkContextName(grant.context);
      const key = `${grant.owner}\0${grant.context}`;
      if (this.grants.has(key)) {
        fail("unknown-context", `duplicate owned context: ${grant.owner}/${grant.context}`);
      }
      this.grants.set(key, Object.freeze({ owner: grant.owner, context: grant.context }));
    }
    this.limits = limits;
  }

  get contextCount(): number {
    return this.contexts.size;
  }

  watchCount(owner: string, context: string): number {
    return this.requireContext(owner, context).watches.size;
  }

  // Bind one owned context. Only an exact declared (owner, context) grant
  // admits; a context the test never granted rejects before any effect.
  // Contexts bind to one owner: a second identity joining the same
  // context name fails the owner check.
  bindContext(owner: string, context: string): ContextBindingFacts {
    checkOwner(owner);
    checkContextName(context);
    const grant = this.grants.get(`${owner}\0${context}`);
    if (grant === undefined) {
      fail("unknown-context", `no owned grant for context: ${owner}/${context}`);
    }
    const prior = this.contexts.get(context);
    if (prior !== undefined && prior.state !== "closed") {
      if (prior.owner !== owner) {
        fail("wrong-owner", `context is owned by another identity: ${context}`);
      }
      return this.bindingFactsOf(prior);
    }
    if (this.contexts.size >= this.limits.maxContexts) {
      fail("capacity-exhausted", "context table full");
    }
    const record: ContextRecord = {
      context,
      owner,
      token: mintToken("ctx"),
      state: "open",
      watches: new Map(),
    };
    this.contexts.set(context, record);
    return this.bindingFactsOf(record);
  }

  private bindingFactsOf(record: ContextRecord): ContextBindingFacts {
    return Object.freeze({
      owner: record.owner,
      context: record.context,
      handleDigest: digestText(record.token),
    });
  }

  binding(owner: string, context: string): ContextBindingFacts {
    return this.bindingFactsOf(this.requireContext(owner, context));
  }

  private requireContext(owner: string, context: string): ContextRecord {
    const record = this.contexts.get(context);
    if (record === undefined) {
      fail("unknown-context", `no bound context for owner: ${owner}/${context}`);
    }
    const contextRecord = record as ContextRecord;
    if (contextRecord.owner !== owner) {
      fail("wrong-owner", "context is owned by another identity");
    }
    if (contextRecord.state === "closed") {
      fail("context-closed", `context is closed: ${owner}/${context}`);
    }
    return contextRecord;
  }

  // Admission-first: the watch gate reports only whether a LIVE binding
  // exists. Never bound, closed, or foreign all refuse with no-context:
  // a missing context blocks capture, whatever the reason.
  private requireLiveBinding(owner: string, context: string): ContextRecord {
    checkOwner(owner);
    checkContextName(context);
    const record = this.contexts.get(context);
    if (record === undefined || record.owner !== owner || record.state !== "open") {
      fail("no-context", `no live context binding for watch: ${owner}/${context}`);
    }
    return record as ContextRecord;
  }

  closeContext(owner: string, context: string): void {
    const record = this.requireContext(owner, context);
    for (const watch of record.watches.values()) {
      if (!watch.sealed) {
        fail("seal-open", `watch interval still open: ${watch.watchId}`);
      }
    }
    record.state = "closed";
  }

  // Open one watch under a live context binding. The watch token is minted
  // here and verified by table lookup on every later mutation, so an
  // invented token is never authority. Watch ids are single-use per
  // context: a sealed id never re-opens.
  watch(owner: string, context: string, watchId: string, target: string): WatchAttestation {
    const record = this.requireLiveBinding(owner, context);
    checkWatchName(watchId);
    const endpoint = checkTarget(target);
    let total = 0;
    for (const contextRecord of this.contexts.values()) {
      total += contextRecord.watches.size;
    }
    if (total >= this.limits.maxWatches) {
      fail("capacity-exhausted", "watch table full");
    }
    const prior = record.watches.get(watchId);
    if (prior !== undefined) {
      if (!prior.sealed) {
        if (prior.target !== endpoint) {
          fail("unknown-target", `watch id is already open on another target: ${watchId}`);
        }
        return this.attestationOf(record, prior);
      }
      fail("watch-closed", `watch id is single-use and already sealed: ${watchId}`);
    }
    const watch: WatchRecord = {
      watchId,
      owner,
      context,
      target: endpoint,
      token: mintToken("wtch"),
      events: [],
      pending: new Map(),
      checkpoint: 0,
      sealed: false,
      receipt: null,
      unrecordedDropped: 0,
    };
    record.watches.set(watchId, watch);
    return this.attestationOf(record, watch);
  }

  private attestationOf(record: ContextRecord, watch: WatchRecord): WatchAttestation {
    return Object.freeze({
      watchId: watch.watchId,
      owner: watch.owner,
      context: record.context,
      target: watch.target,
      contextDigest: digestText(record.token),
      handleDigest: digestText(watch.token),
      cursor: watch.events.length,
    });
  }

  attestation(owner: string, context: string, watchId: string): WatchAttestation {
    const record = this.requireContext(owner, context);
    return this.attestationOf(record, this.requireWatch(record, watchId));
  }

  private requireWatch(record: ContextRecord, watchId: string): WatchRecord {
    const watch = record.watches.get(watchId);
    if (watch === undefined) {
      fail("unknown-watch", `no open watch: ${watchId}`);
    }
    return watch as WatchRecord;
  }

  // Verify the watch token by table lookup. Unknown watches and invented
  // tokens are never authority; sealed watches report watch-closed.
  private requireLiveWatch(
    owner: string,
    context: string,
    watchId: string,
    token: string,
  ): { record: ContextRecord; watch: WatchRecord } {
    const record = this.requireContext(owner, context);
    const watch = this.requireWatch(record, watchId);
    if (watch.sealed) {
      fail("watch-closed", `watch is sealed: ${watchId}`);
    }
    if (token === "" || token !== watch.token) {
      fail("forged-token", "watch token is not the attested token");
    }
    return { record, watch };
  }

  // Test-only accessor: the raw token crosses exactly here so bounded
  // controls can present it on later calls. Facts carry digests only.
  watchTokenForTest(owner: string, context: string, watchId: string): string {
    const record = this.requireContext(owner, context);
    return this.requireWatch(record, watchId).token;
  }

  private appendEvent(watch: WatchRecord, event: EventFacts): void {
    if (watch.events.length >= this.limits.maxEventsPerWatch) {
      // The occurrence is well-formed but unrecordable: poison the
      // interval so the seal can never claim it gapless.
      watch.unrecordedDropped += 1;
      fail("capacity-exhausted", "event log full: occurrence unrecorded, the interval is gapped");
    }
    watch.events.push(event);
  }

  private checkRequestDescriptor(value: unknown): RequestDescriptor {
    const record = checkDescriptor(value, "request");
    checkNoExtraKeys(record, ["captureId", "method", "target"], "request");
    const captureId = checkCaptureName(record["captureId"]);
    const method = checkMethod(record["method"]);
    const target = checkName(record["target"], "request target", "unknown-event");
    return { captureId, method, target };
  }

  private checkResponseDescriptor(value: unknown): ResponseDescriptor {
    const record = checkDescriptor(value, "response");
    checkNoExtraKeys(record, ["captureId", "status", "headerScope", "decodingScope"], "response");
    const captureId = checkCaptureName(record["captureId"]);
    const status = checkStatus(record["status"]);
    const headerScope = checkHeaderScope(record["headerScope"]);
    const decodingScope = checkDecodingScope(record["decodingScope"]);
    return { captureId, status, headerScope, decodingScope };
  }

  private checkNavigationDescriptor(value: unknown): NavigationDescriptor {
    const record = checkDescriptor(value, "navigation");
    checkNoExtraKeys(record, ["navId", "from", "to"], "navigation");
    const navId = checkName(record["navId"], "navigation", "unknown-event");
    const from = checkName(record["from"], "navigation source", "unknown-event");
    const to = checkName(record["to"], "navigation destination", "unknown-event");
    return { navId, from, to };
  }

  private checkBodyDescriptor(value: unknown): BodyDescriptor {
    const record = checkDescriptor(value, "body");
    checkNoExtraKeys(record, ["byteLength", "bodyDigest", "decodingScope"], "body");
    const byteLength = record["byteLength"];
    if (!Number.isSafeInteger(byteLength) || (byteLength as number) < 0) {
      fail(
        "unknown-event",
        `body byteLength must be a non-negative integer, got: ${String(byteLength)}`,
      );
    }
    const bodyDigest = record["bodyDigest"];
    if (typeof bodyDigest !== "string" || !BODY_DIGEST_RE.test(bodyDigest)) {
      fail("unknown-event", "body completion carries a body digest, never raw bytes");
    }
    const decodingScope = checkDecodingScope(record["decodingScope"]);
    return { byteLength: byteLength as number, bodyDigest: bodyDigest as string, decodingScope };
  }

  private checkGapDescriptor(value: unknown): GapDescriptor {
    const record = checkDescriptor(value, "gap");
    checkNoExtraKeys(record, ["dropped", "reason"], "gap");
    const dropped = record["dropped"];
    if (!Number.isSafeInteger(dropped) || (dropped as number) <= 0) {
      fail("unknown-event", `gap dropped must be a positive integer, got: ${String(dropped)}`);
    }
    const reason = checkGapReason(record["reason"]);
    return { dropped: dropped as number, reason };
  }

  // Capture one request: opens a pending capture the body/error callbacks
  // must settle. A capture id is single-flight per watch.
  emitRequest(
    owner: string,
    context: string,
    watchId: string,
    token: string,
    descriptor: unknown,
  ): RequestEventFacts {
    const request = this.checkRequestDescriptor(descriptor);
    const { watch } = this.requireLiveWatch(owner, context, watchId, token);
    if (watch.pending.has(request.captureId)) {
      fail("capture-pending", `capture is already pending: ${request.captureId}`);
    }
    if (watch.pending.size >= this.limits.maxPendingPerWatch) {
      watch.unrecordedDropped += 1;
      fail("capacity-exhausted", "pending table full: request unrecorded, the interval is gapped");
    }
    const seq = watch.events.length;
    const facts: RequestEventFacts = Object.freeze({
      seq,
      kind: "request",
      captureId: request.captureId,
      method: request.method,
      target: request.target,
      digest: digestText(`event:${watchId}:${seq}:request:${request.captureId}`),
    });
    this.appendEvent(watch, facts);
    watch.pending.set(request.captureId, {
      captureId: request.captureId,
      requestSeq: seq,
      responseSeq: null,
    });
    return facts;
  }

  // Capture one response: joins the request's pending capture and states
  // the repeated-header and decoding scope explicitly. The capture stays
  // pending until the body/error callbacks settle it.
  emitResponse(
    owner: string,
    context: string,
    watchId: string,
    token: string,
    descriptor: unknown,
  ): ResponseEventFacts {
    const response = this.checkResponseDescriptor(descriptor);
    const { watch } = this.requireLiveWatch(owner, context, watchId, token);
    const pending = watch.pending.get(response.captureId);
    if (pending === undefined) {
      fail("capture-unknown", `no pending capture for response: ${response.captureId}`);
    }
    if ((pending as PendingRecord).responseSeq !== null) {
      fail("capture-pending", `capture already holds a response: ${response.captureId}`);
    }
    const seq = watch.events.length;
    const facts: ResponseEventFacts = Object.freeze({
      seq,
      kind: "response",
      captureId: response.captureId,
      status: response.status,
      headerScope: response.headerScope,
      decodingScope: response.decodingScope,
      digest: digestText(`event:${watchId}:${seq}:response:${response.captureId}`),
    });
    this.appendEvent(watch, facts);
    (pending as PendingRecord).responseSeq = seq;
    return facts;
  }

  // Capture one navigation: atomic, never pending.
  emitNavigation(
    owner: string,
    context: string,
    watchId: string,
    token: string,
    descriptor: unknown,
  ): NavigationEventFacts {
    const navigation = this.checkNavigationDescriptor(descriptor);
    const { watch } = this.requireLiveWatch(owner, context, watchId, token);
    const seq = watch.events.length;
    const facts: NavigationEventFacts = Object.freeze({
      seq,
      kind: "navigation",
      navId: navigation.navId,
      from: navigation.from,
      to: navigation.to,
      digest: digestText(`event:${watchId}:${seq}:navigation:${navigation.navId}`),
    });
    this.appendEvent(watch, facts);
    return facts;
  }

  // Complete one pending capture with its full body facts: byte length, a
  // body digest (raw bytes never cross), and the explicit decoding scope.
  completeBody(
    owner: string,
    context: string,
    watchId: string,
    token: string,
    captureId: string,
    descriptor: unknown,
  ): BodyEventFacts {
    const body = this.checkBodyDescriptor(descriptor);
    const id = checkCaptureName(captureId);
    const { watch } = this.requireLiveWatch(owner, context, watchId, token);
    const pending = watch.pending.get(id);
    if (pending === undefined) {
      fail("capture-unknown", `no pending capture to complete: ${id}`);
    }
    const seq = watch.events.length;
    const facts: BodyEventFacts = Object.freeze({
      seq,
      kind: "body-complete",
      captureId: id,
      byteLength: body.byteLength,
      bodyDigest: body.bodyDigest,
      decodingScope: body.decodingScope,
      digest: digestText(`event:${watchId}:${seq}:body-complete:${id}`),
    });
    this.appendEvent(watch, facts);
    watch.pending.delete(id);
    return facts;
  }

  // Fail one pending capture with a declared error reason. The error
  // callback settles the capture exactly like the body callback.
  failCapture(
    owner: string,
    context: string,
    watchId: string,
    token: string,
    captureId: string,
    reason: unknown,
  ): CaptureErrorEventFacts {
    const error = checkCaptureError(reason);
    const id = checkCaptureName(captureId);
    const { watch } = this.requireLiveWatch(owner, context, watchId, token);
    const pending = watch.pending.get(id);
    if (pending === undefined) {
      fail("capture-unknown", `no pending capture to fail: ${id}`);
    }
    const seq = watch.events.length;
    const facts: CaptureErrorEventFacts = Object.freeze({
      seq,
      kind: "capture-error",
      captureId: id,
      reason: error,
      digest: digestText(`event:${watchId}:${seq}:capture-error:${id}`),
    });
    this.appendEvent(watch, facts);
    watch.pending.delete(id);
    return facts;
  }

  // Mark an explicit gap: dropped events the watch never observed. Gaps
  // seal honestly: the receipt carries gapped, and a gapped interval can
  // never prove absence.
  noteGap(
    owner: string,
    context: string,
    watchId: string,
    token: string,
    descriptor: unknown,
  ): GapEventFacts {
    const gap = this.checkGapDescriptor(descriptor);
    const { watch } = this.requireLiveWatch(owner, context, watchId, token);
    const seq = watch.events.length;
    const facts: GapEventFacts = Object.freeze({
      seq,
      kind: "gap",
      dropped: gap.dropped,
      reason: gap.reason,
      digest: digestText(`event:${watchId}:${seq}:gap:${gap.reason}`),
    });
    this.appendEvent(watch, facts);
    return facts;
  }

  // Read the log from a cursor. Reads are facts: no token is needed, and
  // reads stay available after the seal. A cursor past the tip or off the
  // integer line is unknown.
  readFrom(owner: string, context: string, watchId: string, cursor: unknown): CursorReadFacts {
    if (!Number.isSafeInteger(cursor) || (cursor as number) < 0) {
      fail("cursor-unknown", `cursor is not a non-negative integer: ${String(cursor)}`);
    }
    const record = this.requireContext(owner, context);
    const watch = this.requireWatch(record, watchId);
    const from = cursor as number;
    if (from > watch.events.length) {
      fail("cursor-unknown", `cursor is past the event tip: ${String(cursor)}`);
    }
    const events = Object.freeze(watch.events.slice(from));
    return Object.freeze({
      watchId: watch.watchId,
      cursor: from,
      nextCursor: watch.events.length,
      events,
      digest: digestText(`read:${watchId}:${from}:${watch.events.length}`),
    });
  }

  // Advance the durable checkpoint. Checkpoints move forward only and
  // never past the tip: a backward checkpoint is stale, a tip-past
  // checkpoint is unknown. The seal requires the checkpoint at the tip.
  checkpoint(
    owner: string,
    context: string,
    watchId: string,
    token: string,
    cursor: unknown,
  ): CheckpointFacts {
    if (!Number.isSafeInteger(cursor) || (cursor as number) < 0) {
      fail("cursor-unknown", `cursor is not a non-negative integer: ${String(cursor)}`);
    }
    const { watch } = this.requireLiveWatch(owner, context, watchId, token);
    const at = cursor as number;
    if (at > watch.events.length) {
      fail("cursor-unknown", `cursor is past the event tip: ${String(cursor)}`);
    }
    if (at < watch.checkpoint) {
      fail("checkpoint-stale", `checkpoint moves forward only: ${at} < ${watch.checkpoint}`);
    }
    watch.checkpoint = at;
    return Object.freeze({
      watchId: watch.watchId,
      cursor: at,
      events: watch.events.length,
      digest: digestText(`checkpoint:${watchId}:${at}:${watch.events.length}`),
    });
  }

  private watchGapped(watch: WatchRecord): boolean {
    return watch.unrecordedDropped > 0 || watch.events.some((event) => event.kind === "gap");
  }

  private intervalFlags(watch: WatchRecord): string {
    return [
      `gapped:${this.watchGapped(watch) ? 1 : 0}`,
      `sealed:${watch.sealed ? 1 : 0}`,
      `checkpoint:${watch.checkpoint}`,
    ].join("|");
  }

  // Interval facts stay readable while open and after the seal: the
  // interval is evidence, and neither pending captures nor gaps hide it.
  intervalFacts(owner: string, context: string, watchId: string): IntervalFacts {
    const record = this.requireContext(owner, context);
    const watch = this.requireWatch(record, watchId);
    const events = Object.freeze([...watch.events]);
    const pending = Object.freeze([...watch.pending.keys()].sort());
    const flags = this.intervalFlags(watch);
    return Object.freeze({
      watchId: watch.watchId,
      owner: watch.owner,
      context: record.context,
      target: watch.target,
      events,
      pending,
      gapped: this.watchGapped(watch),
      sealed: watch.sealed,
      checkpoint: watch.checkpoint,
      digest: digestEventInterval(watch.watchId, events, pending, flags),
    });
  }

  // Seal the watch with its terminal receipt. Pending captures block the
  // seal: nothing unsettled seals. A checkpoint behind the tip blocks the
  // seal too: checkpoint the full log first. Gaps seal honestly: the
  // receipt carries gapped, and a gapped interval never proves absence.
  sealWatch(owner: string, context: string, watchId: string, token: string): SealReceipt {
    const record = this.requireContext(owner, context);
    const watch = this.requireWatch(record, watchId);
    if (watch.sealed) {
      fail("already-sealed", `watch is already sealed: ${watchId}`);
    }
    if (token === "" || token !== watch.token) {
      fail("forged-token", "watch token is not the attested token");
    }
    if (watch.pending.size > 0) {
      const ids = [...watch.pending.keys()].sort().join(",");
      fail("capture-pending", `captures still pending: ${ids}`);
    }
    if (watch.checkpoint !== watch.events.length) {
      fail(
        "checkpoint-stale",
        `checkpoint ${watch.checkpoint} is behind the event tip ${watch.events.length}`,
      );
    }
    watch.sealed = true;
    const receipt: SealReceipt = Object.freeze({
      watchId: watch.watchId,
      owner: watch.owner,
      context: record.context,
      target: watch.target,
      events: watch.events.length,
      gapped: this.watchGapped(watch),
      checkpoint: watch.checkpoint,
      digest: digestText(`seal:${watchId}:${watch.events.length}:${this.intervalFlags(watch)}`),
    });
    watch.receipt = receipt;
    return receipt;
  }

  sealReceipt(owner: string, context: string, watchId: string): SealReceipt {
    const record = this.requireContext(owner, context);
    const watch = this.requireWatch(record, watchId);
    if (!watch.sealed || watch.receipt === null) {
      fail("seal-open", `watch interval is still open: ${watchId}`);
    }
    return watch.receipt as SealReceipt;
  }
}

// ---------------------------------------------------------------------------
// Absence verdict: only a sealed gapless interval judges
// ---------------------------------------------------------------------------

function checkAbsenceProofShape(value: unknown): AbsenceProofClaim {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail("forbidden-proof", "absence proof must be a sealed-interval object");
  }
  const record = value as Record<string, unknown>;
  if (record["kind"] === "sealed-interval") {
    const interval = record["interval"] as IntervalFacts;
    if (interval === null || typeof interval !== "object" || Array.isArray(interval)) {
      fail("forbidden-proof", "sealed interval carries no interval facts");
    }
    const absenceOf = checkAbsenceSubject(record["absenceOf"]);
    return { kind: "sealed-interval", interval, absenceOf };
  }
  if (typeof record["kind"] === "string") {
    fail("forbidden-proof", `absence proof kind is inadmissible: ${String(record["kind"])}`);
  }
  fail("forbidden-proof", "absence proof carries no kind");
}

// Judge one absence proof claim. Only a verified sealed gapless interval
// with zero observed events of the claimed subject proves absence:
// pending-capture, open-watch, quiet-prefix, event-log, and any other
// non-interval claim reject with forbidden-proof before any verdict is
// read; an open or gapped interval never proves absence either.
export function assertAbsenceProved(claim: unknown): AbsenceVerdict {
  const proof = checkAbsenceProofShape(claim);
  if (proof.kind !== "sealed-interval") {
    fail("forbidden-proof", `absence proof kind is inadmissible: ${proof.kind}`);
  }
  const sealed = proof as {
    kind: "sealed-interval";
    interval: IntervalFacts;
    absenceOf: AbsenceSubject;
  };
  const interval = sealed.interval;
  if (
    typeof interval.watchId !== "string" ||
    !Array.isArray(interval.events) ||
    !Array.isArray(interval.pending) ||
    !interval.pending.every((entry) => typeof entry === "string") ||
    typeof interval.gapped !== "boolean" ||
    typeof interval.sealed !== "boolean" ||
    !Number.isSafeInteger(interval.checkpoint) ||
    typeof interval.digest !== "string"
  ) {
    fail("forbidden-proof", "sealed interval carries malformed facts");
  }
  for (const event of interval.events) {
    if (event === null || typeof event !== "object" || typeof event.kind !== "string") {
      fail("forbidden-proof", "sealed interval carries a malformed event");
    }
    // Undeclared kinds reject here so the digest hash below never meets
    // a kind outside the closed vocabulary (which would crash it with a
    // raw TypeError instead of a layered failure).
    if (!(EVENT_KINDS as readonly string[]).includes(event.kind)) {
      fail("forbidden-proof", `interval carries an undeclared event kind: ${event.kind}`);
    }
  }
  const flags = [
    `gapped:${interval.gapped ? 1 : 0}`,
    `sealed:${interval.sealed ? 1 : 0}`,
    `checkpoint:${interval.checkpoint}`,
  ].join("|");
  const recomputed = digestEventInterval(
    interval.watchId,
    interval.events,
    interval.pending,
    flags,
  );
  if (recomputed !== interval.digest) {
    fail("forbidden-proof", "sealed interval digest does not verify");
  }
  if (!interval.sealed) {
    fail("forbidden-proof", "open intervals never prove absence: seal the watch first");
  }
  if (interval.pending.length > 0) {
    fail("forbidden-proof", "intervals with pending captures never prove absence");
  }
  if (interval.gapped || interval.events.some((event) => event.kind === "gap")) {
    return Object.freeze({
      watchId: interval.watchId,
      absenceOf: sealed.absenceOf,
      absent: false,
      gapped: true,
      digest: digestText(`verdict:${interval.watchId}:${sealed.absenceOf}:gapped`),
    });
  }
  const observed =
    sealed.absenceOf === "any"
      ? interval.events.length
      : interval.events.filter((event) => event.kind === sealed.absenceOf).length;
  return Object.freeze({
    watchId: interval.watchId,
    absenceOf: sealed.absenceOf,
    absent: observed === 0,
    gapped: false,
    digest: digestText(`verdict:${interval.watchId}:${sealed.absenceOf}:${observed === 0 ? 1 : 0}`),
  });
}
