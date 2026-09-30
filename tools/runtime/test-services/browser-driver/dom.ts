// K13: immediate DOM and event provenance.
//
// A provenance service over in-memory doubles only: no DOM, no browser, no
// process, no flags, no environment, no sampling, no I/O, no timers. The
// service binds owned contexts, seeds inert nodes as digest-only doubles,
// runs finite same-task batches of clone/append/emit operations, attaches
// same-node listeners that acknowledge recorded occurrences, cancels
// occurrences only through an exact (occurrence, node, listener) match, and
// serves a microtask-order observer whose transcript order is explicitly
// scripted, never wall-clock. Task and microtask order is scripted by the
// test: yieldTask advances the task clock, and observeMicrotasks assigns
// microtask sequence numbers in the exact order the test supplies.
//
// Independence scope from the launch/driver service under test (normative):
//   SHARES: nothing executable. The service never imports, calls, or reads
//   driver code, driver-decoded values, receipts, flags, environment
//   variables, or process state. The only strings that may coincide are
//   the context/node/batch/listener/occurrence names under test (supplied
//   by the test, not by the driver) and the vocabulary this file uses for
//   facts. Facts carry digests only: raw DOM bytes never cross.
//   PROVES: context admission, node identity, clone freshness, finite
//   same-task batches, task yields, same-node listener acknowledgment,
//   exact cancellation, scripted microtask order, occurrence verdicts that
//   reject unobserved claims, and error provenance (every failure names its
//   layer). Driver claims are compared AGAINST these facts; the service
//   never derives a fact FROM a driver claim.
//   Consequently a yielded batch, a cancelled-by-mismatch claim, a
//   returned original handle, or a missing occurrence can never pass
//   silently: each carries a named fact or a closed-vocabulary code.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, browsers, processes, or live runtimes. Local
// controls only. Live host-dependent controls wait for the corresponding
// qualified profile and Q task.

import { createHash, randomBytes } from "node:crypto";

export const BROWSER_DOM_SCHEMA_VERSION = "1" as const;

// ---------------------------------------------------------------------------
// Independence scope (normative)
// ---------------------------------------------------------------------------

export const BROWSER_DOM_INDEPENDENCE_SCOPE: string =
  "independent: shares no driver code, decoded values, receipts, flags, " +
  "environment, or process state; proves context admission, node identity, " +
  "clone freshness, finite same-task batches, task yields, same-node " +
  "listener acknowledgment, exact cancellation, scripted microtask order, " +
  "occurrence verdicts, and layered error provenance from its own seeded " +
  "doubles alone";

// ---------------------------------------------------------------------------
// Layers, codes, and the layered error
// ---------------------------------------------------------------------------

export const BROWSER_DOM_LAYERS = ["dom", "batch", "event"] as const;
export type BrowserDomLayer = (typeof BROWSER_DOM_LAYERS)[number];

// Closed failure-code vocabulary. Every code maps to exactly one layer:
//   dom: context admission, ownership, capacity, lifecycle, node identity,
//     clone freshness, and append provenance.
//   batch: batch binding, tokens, lifecycle, finite operations, the
//     scripted task clock, yields, and the immediate seal.
//   event: listener attachment, event vocabulary, occurrence identity,
//     same-node acknowledgment, exact cancellation, the scripted microtask
//     observer, and occurrence verdicts.
export const BROWSER_DOM_CODES = [
  "unknown-context",
  "no-context",
  "wrong-owner",
  "capacity-exhausted",
  "context-closed",
  "unknown-node",
  "node-closed",
  "original-returned",
  "forged-token",
  "unknown-batch",
  "batch-closed",
  "batch-pending",
  "batch-yielded",
  "unknown-listener",
  "listener-mismatch",
  "unknown-event",
  "unknown-occurrence",
  "occurrence-closed",
  "occurrence-missing",
  "unknown-transcript",
  "wrong-cancellation",
  "already-cancelled",
  "forbidden-proof",
] as const;
export type BrowserDomCode = (typeof BROWSER_DOM_CODES)[number];

const CODE_LAYER: Readonly<Record<BrowserDomCode, BrowserDomLayer>> = {
  "unknown-context": "dom",
  "no-context": "dom",
  "wrong-owner": "dom",
  "capacity-exhausted": "dom",
  "context-closed": "dom",
  "unknown-node": "dom",
  "node-closed": "dom",
  "original-returned": "dom",
  "forged-token": "batch",
  "unknown-batch": "batch",
  "batch-closed": "batch",
  "batch-pending": "batch",
  "batch-yielded": "batch",
  "unknown-listener": "event",
  "listener-mismatch": "event",
  "unknown-event": "event",
  "unknown-occurrence": "event",
  "occurrence-closed": "event",
  "occurrence-missing": "event",
  "unknown-transcript": "event",
  "wrong-cancellation": "event",
  "already-cancelled": "event",
  "forbidden-proof": "event",
};

export function layerOfDomCode(code: BrowserDomCode): BrowserDomLayer {
  return CODE_LAYER[code];
}

export class BrowserDomError extends Error {
  readonly layer: BrowserDomLayer;
  readonly code: BrowserDomCode;
  constructor(code: BrowserDomCode, message: string) {
    super(`[${CODE_LAYER[code]}:${code}] ${message}`);
    this.name = "BrowserDomError";
    this.layer = CODE_LAYER[code];
    this.code = code;
  }
}

function fail(code: BrowserDomCode, message: string): never {
  throw new BrowserDomError(code, message);
}

// ---------------------------------------------------------------------------
// Bounds and names
// ---------------------------------------------------------------------------

export type BrowserDomLimits = Readonly<{
  maxContexts: number;
  maxNodes: number;
  maxBatches: number;
  maxOpsPerBatch: number;
  maxListeners: number;
  maxOccurrences: number;
  maxTranscripts: number;
}>;

const LIMIT_KEYS = [
  "maxContexts",
  "maxNodes",
  "maxBatches",
  "maxOpsPerBatch",
  "maxListeners",
  "maxOccurrences",
  "maxTranscripts",
] as const;

export function checkDomLimits(value: unknown): BrowserDomLimits {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new BrowserDomError("unknown-context", "limits must be an object");
  }
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    if (!(LIMIT_KEYS as readonly string[]).includes(key)) {
      throw new BrowserDomError("unknown-context", `unknown limit field: ${key}`);
    }
  }
  const out: Record<string, number> = {};
  for (const key of LIMIT_KEYS) {
    const entry = record[key];
    if (!Number.isSafeInteger(entry) || (entry as number) <= 0) {
      throw new BrowserDomError("unknown-context", `${key} must be a positive integer`);
    }
    out[key] = entry as number;
  }
  return Object.freeze(out) as BrowserDomLimits;
}

// Finite node-kind vocabulary: every seeded node declares exactly one.
export const NODE_KINDS = ["element", "text"] as const;
export type NodeKind = (typeof NODE_KINDS)[number];

// Finite batch-operation vocabulary: every batch op is exactly one.
export const DOM_OPS = ["clone", "append", "emit"] as const;
export type DomOp = (typeof DOM_OPS)[number];

// Finite event vocabulary: every listener and occurrence names exactly one.
export const DOM_EVENTS = ["click", "input", "mutation"] as const;
export type DomEvent = (typeof DOM_EVENTS)[number];

// Finite cancellation modes: every cancellation states its mode explicitly.
export const CANCEL_MODES = ["prevent-default", "stop-propagation"] as const;
export type CancelMode = (typeof CANCEL_MODES)[number];

const MAX_NAME_LEN = 128;
const TEXT_DIGEST_RE = /^sha256:[0-9a-f]{64}$/;

function checkName(value: unknown, what: string, code: BrowserDomCode): string {
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

function checkNodeId(value: unknown): string {
  return checkName(value, "node", "unknown-node");
}

function checkBatchId(value: unknown): string {
  return checkName(value, "batch", "unknown-batch");
}

function checkListenerId(value: unknown): string {
  return checkName(value, "listener", "unknown-listener");
}

function checkOccurrenceId(value: unknown): string {
  return checkName(value, "occurrence", "unknown-occurrence");
}

function checkKind(value: unknown): NodeKind {
  if (typeof value !== "string" || !(NODE_KINDS as readonly string[]).includes(value)) {
    fail("unknown-node", `not a declared node kind: ${String(value)}`);
  }
  return value as NodeKind;
}

function checkEvent(value: unknown): DomEvent {
  if (typeof value !== "string" || !(DOM_EVENTS as readonly string[]).includes(value)) {
    fail("unknown-event", `not a declared DOM event: ${String(value)}`);
  }
  return value as DomEvent;
}

function checkCancelMode(value: unknown): CancelMode {
  if (typeof value !== "string" || !(CANCEL_MODES as readonly string[]).includes(value)) {
    fail("wrong-cancellation", `not a declared cancellation mode: ${String(value)}`);
  }
  return value as CancelMode;
}

function checkTextDigest(value: unknown): string {
  if (typeof value !== "string" || !TEXT_DIGEST_RE.test(value)) {
    fail("unknown-node", "nodes carry text digests, never raw bytes");
  }
  return value;
}

function checkDescriptor(
  value: unknown,
  what: string,
  code: BrowserDomCode,
): Record<string, unknown> {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail(code, `${what} must be an object`);
  }
  return value as Record<string, unknown>;
}

function checkNoExtraKeys(
  record: Record<string, unknown>,
  keys: readonly string[],
  what: string,
  code: BrowserDomCode,
): void {
  for (const key of Object.keys(record)) {
    if (!keys.includes(key)) {
      fail(code, `${what} carries an unknown field: ${key}`);
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

export type NodeFacts = Readonly<{
  nodeId: string;
  owner: string;
  context: string;
  kind: string;
  textDigest: string;
  cloneOf: string | null;
  digest: string;
}>;

export type CloneFacts = Readonly<{
  originalId: string;
  cloneId: string;
  owner: string;
  context: string;
  batchId: string;
  opSeq: number;
  digest: string;
}>;

export type AppendFacts = Readonly<{
  parentId: string;
  childId: string;
  owner: string;
  context: string;
  batchId: string;
  opSeq: number;
  digest: string;
}>;

export type OpFacts = Readonly<{
  seq: number;
  op: DomOp;
  ref: string;
}>;

export type BatchAttestation = Readonly<{
  batchId: string;
  owner: string;
  context: string;
  taskSeq: number;
  contextDigest: string;
  handleDigest: string;
  open: true;
}>;

// Fact one: the sealed batch. The yielded flag names the task yield that
// interrupted the batch, or reports false with a null yieldSeq when the
// batch ran within one task. Only a sealed unyielded batch proves
// immediacy; see assertBatchImmediate.
export type BatchFacts = Readonly<{
  batchId: string;
  owner: string;
  context: string;
  taskSeq: number;
  sealed: boolean;
  ops: readonly OpFacts[];
  yielded: boolean;
  yieldSeq: number | null;
  digest: string;
}>;

export type TaskFacts = Readonly<{
  owner: string;
  context: string;
  taskSeq: number;
  yieldedBatches: readonly string[];
  digest: string;
}>;

export type ListenerFacts = Readonly<{
  listenerId: string;
  owner: string;
  context: string;
  nodeId: string;
  event: DomEvent;
  digest: string;
}>;

// Fact two: the recorded occurrence. microtaskSeq stays null until the
// scripted microtask observer assigns it: a null slot is a missing
// occurrence, detectable through assertOccurrenceObserved, never a quiet
// pass. Cancel state and the acknowledgment count are carried so the
// digest re-verifies the whole record.
export type OccurrenceFacts = Readonly<{
  occurrenceId: string;
  owner: string;
  context: string;
  event: DomEvent;
  nodeId: string;
  batchId: string;
  taskSeq: number;
  microtaskSeq: number | null;
  cancelled: boolean;
  cancelMode: CancelMode | null;
  acks: number;
  digest: string;
}>;

export type AckFacts = Readonly<{
  occurrenceId: string;
  nodeId: string;
  listenerId: string;
  ackSeq: number;
  digest: string;
}>;

export type CancelFacts = Readonly<{
  occurrenceId: string;
  nodeId: string;
  listenerId: string;
  mode: CancelMode;
  digest: string;
}>;

export type TranscriptFacts = Readonly<{
  transcriptSeq: number;
  owner: string;
  context: string;
  order: readonly string[];
  taskSeq: number;
  digest: string;
}>;

export type NodeDescriptor = Readonly<{
  kind: NodeKind;
  textDigest: string;
}>;

export type ListenerDescriptor = Readonly<{
  event: DomEvent;
}>;

export type CloneDescriptor = Readonly<{
  nodeId: string;
}>;

export type AppendDescriptor = Readonly<{
  parentId: string;
  childId: string;
}>;

export type EmitDescriptor = Readonly<{
  nodeId: string;
  occurrenceId: string;
  event: DomEvent;
}>;

export type AckDescriptor = Readonly<{
  occurrenceId: string;
  nodeId: string;
  listenerId: string;
}>;

export type CancelDescriptor = Readonly<{
  occurrenceId: string;
  nodeId: string;
  listenerId: string;
  mode: CancelMode;
}>;

export type ObserveDescriptor = Readonly<{
  order: readonly string[];
}>;

// An occurrence proof claim. Only observed-occurrence is admissible; every
// other kind rejects with forbidden-proof before any verdict is read.
export const FORBIDDEN_OCCURRENCE_KINDS = [
  "unobserved-occurrence",
  "batch-op",
  "listener-ack",
] as const;
export type ForbiddenOccurrenceKind = (typeof FORBIDDEN_OCCURRENCE_KINDS)[number];

export type OccurrenceProofClaim =
  | Readonly<{ kind: "observed-occurrence"; occurrence: OccurrenceFacts }>
  | Readonly<{ kind: ForbiddenOccurrenceKind; detail: string }>;

export type OccurrenceVerdict = Readonly<{
  occurrenceId: string;
  observed: true;
  microtaskSeq: number;
  digest: string;
}>;

export type CloneVerdict = Readonly<{
  originalId: string;
  cloneId: string;
  fresh: true;
  digest: string;
}>;

export type BatchVerdict = Readonly<{
  batchId: string;
  immediate: true;
  ops: number;
  digest: string;
}>;

type NodeRecord = {
  nodeId: string;
  owner: string;
  context: string;
  kind: NodeKind;
  textDigest: string;
  cloneOf: string | null;
  parentId: string | null;
};

type BatchRecord = {
  batchId: string;
  owner: string;
  context: string;
  token: string;
  taskSeq: number;
  sealed: boolean;
  ops: { seq: number; op: DomOp; ref: string }[];
  yielded: boolean;
  yieldSeq: number | null;
};

type ListenerRecord = {
  listenerId: string;
  owner: string;
  context: string;
  nodeId: string;
  event: DomEvent;
};

type OccurrenceRecord = {
  occurrenceId: string;
  owner: string;
  context: string;
  event: DomEvent;
  nodeId: string;
  batchId: string;
  taskSeq: number;
  microtaskSeq: number | null;
  cancelled: boolean;
  cancelMode: CancelMode | null;
  acks: number;
};

type TranscriptRecord = {
  transcriptSeq: number;
  owner: string;
  context: string;
  order: string[];
  taskSeq: number;
};

type ContextState = "open" | "closed";

type ContextRecord = {
  context: string;
  owner: string;
  token: string;
  state: ContextState;
  taskSeq: number;
  cloneSeq: number;
  nodes: Map<string, NodeRecord>;
  batches: Map<string, BatchRecord>;
  listeners: Map<string, ListenerRecord>;
  occurrences: Map<string, OccurrenceRecord>;
  transcripts: TranscriptRecord[];
};

function mintToken(prefix: string): string {
  return `${prefix}-${randomBytes(16).toString("hex")}`;
}

function digestText(text: string): string {
  return `sha256:${createHash("sha256").update(text, "utf8").digest("hex")}`;
}

// Clone, batch, and occurrence digests are recomputed from carried fields,
// so the verdict functions re-verify presented facts instead of trusting
// their digest strings.
export function digestClone(
  originalId: string,
  cloneId: string,
  owner: string,
  context: string,
  batchId: string,
  opSeq: number,
): string {
  return digestText(
    `clone:${originalId}:${cloneId}:${owner}:${context}:${batchId}:${String(opSeq)}`,
  );
}

export function digestAppend(
  parentId: string,
  childId: string,
  owner: string,
  context: string,
  batchId: string,
  opSeq: number,
): string {
  return digestText(
    `append:${parentId}:${childId}:${owner}:${context}:${batchId}:${String(opSeq)}`,
  );
}

function encodeOps(ops: readonly { seq: number; op: string; ref: string }[]): string {
  return ops.map((entry) => `${String(entry.seq)}:${entry.op}:${entry.ref}`).join(",");
}

export function digestBatch(
  batchId: string,
  owner: string,
  context: string,
  taskSeq: number,
  sealed: boolean,
  ops: readonly { seq: number; op: string; ref: string }[],
  yielded: boolean,
  yieldSeq: number | null,
): string {
  const hash = createHash("sha256");
  hash.update(batchId, "utf8");
  hash.update("\0", "utf8");
  hash.update(owner, "utf8");
  hash.update("\0", "utf8");
  hash.update(context, "utf8");
  hash.update("\0", "utf8");
  hash.update(String(taskSeq), "utf8");
  hash.update("\0", "utf8");
  hash.update(sealed ? "sealed" : "open", "utf8");
  hash.update("\0", "utf8");
  hash.update(encodeOps(ops), "utf8");
  hash.update("\0", "utf8");
  hash.update(yielded ? "yielded" : "immediate", "utf8");
  hash.update("\0", "utf8");
  hash.update(yieldSeq === null ? "" : String(yieldSeq), "utf8");
  return `sha256:${hash.digest("hex")}`;
}

export function digestOccurrence(
  occurrenceId: string,
  owner: string,
  context: string,
  event: string,
  nodeId: string,
  batchId: string,
  taskSeq: number,
  microtaskSeq: number | null,
  cancelled: boolean,
  cancelMode: string | null,
  acks: number,
): string {
  const hash = createHash("sha256");
  hash.update(occurrenceId, "utf8");
  hash.update("\0", "utf8");
  hash.update(owner, "utf8");
  hash.update("\0", "utf8");
  hash.update(context, "utf8");
  hash.update("\0", "utf8");
  hash.update(event, "utf8");
  hash.update("\0", "utf8");
  hash.update(nodeId, "utf8");
  hash.update("\0", "utf8");
  hash.update(batchId, "utf8");
  hash.update("\0", "utf8");
  hash.update(String(taskSeq), "utf8");
  hash.update("\0", "utf8");
  hash.update(microtaskSeq === null ? "" : String(microtaskSeq), "utf8");
  hash.update("\0", "utf8");
  hash.update(cancelled ? "cancelled" : "live", "utf8");
  hash.update("\0", "utf8");
  hash.update(cancelMode ?? "", "utf8");
  hash.update("\0", "utf8");
  hash.update(String(acks), "utf8");
  return `sha256:${hash.digest("hex")}`;
}

// ---------------------------------------------------------------------------
// Service: owned contexts, nodes, same-task batches, listeners, occurrences
// ---------------------------------------------------------------------------

export type ContextGrant = Readonly<{
  owner: string;
  context: string;
}>;

export class BrowserDomService {
  private readonly limits: BrowserDomLimits;
  private readonly grants = new Map<string, ContextGrant>();
  private readonly contexts = new Map<string, ContextRecord>();

  constructor(grants: readonly ContextGrant[], limits: BrowserDomLimits) {
    if (grants.length === 0) {
      throw new BrowserDomError("unknown-context", "declare at least one owned context");
    }
    if (grants.length > limits.maxContexts) {
      throw new BrowserDomError("capacity-exhausted", "declared contexts exceed the cap");
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

  nodeCount(owner: string, context: string): number {
    return this.requireContext(owner, context).nodes.size;
  }

  batchCount(owner: string, context: string): number {
    return this.requireContext(owner, context).batches.size;
  }

  listenerCount(owner: string, context: string): number {
    return this.requireContext(owner, context).listeners.size;
  }

  occurrenceCount(owner: string, context: string): number {
    return this.requireContext(owner, context).occurrences.size;
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
      taskSeq: 0,
      cloneSeq: 0,
      nodes: new Map(),
      batches: new Map(),
      listeners: new Map(),
      occurrences: new Map(),
      transcripts: [],
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

  // Admission-first: the seed/observe/begin gate reports only whether a LIVE
  // binding exists. Never bound, closed, or foreign all refuse with
  // no-context: a missing context blocks DOM work, whatever the reason.
  private requireLiveBinding(owner: string, context: string): ContextRecord {
    checkOwner(owner);
    checkContextName(context);
    const record = this.contexts.get(context);
    if (record === undefined || record.owner !== owner || record.state !== "open") {
      fail("no-context", `no live context binding: ${owner}/${context}`);
    }
    return record as ContextRecord;
  }

  // Close one context. Open batches block the close: nothing unsealed
  // closes. Sealed batches, nodes, listeners, occurrences, and transcripts
  // stay out of the way.
  closeContext(owner: string, context: string): void {
    const record = this.requireContext(owner, context);
    for (const batch of record.batches.values()) {
      if (!batch.sealed) {
        fail("batch-pending", `batch still open: ${batch.batchId}`);
      }
    }
    record.state = "closed";
  }

  // Advance the scripted task clock. There is no wall clock: the test
  // scripts every task boundary explicitly. Each open batch is marked
  // yielded at the new task sequence, and the returned facts name every
  // batch the yield interrupted.
  yieldTask(owner: string, context: string): TaskFacts {
    const record = this.requireLiveBinding(owner, context);
    record.taskSeq += 1;
    const yielded: string[] = [];
    for (const batch of record.batches.values()) {
      if (!batch.sealed && !batch.yielded) {
        batch.yielded = true;
        batch.yieldSeq = record.taskSeq;
        yielded.push(batch.batchId);
      }
    }
    return Object.freeze({
      owner: record.owner,
      context: record.context,
      taskSeq: record.taskSeq,
      yieldedBatches: Object.freeze([...yielded]),
      digest: digestText(`task:${record.owner}:${record.context}:${String(record.taskSeq)}`),
    });
  }

  // -------------------------------------------------------------------------
  // Nodes and listeners: inert doubles, same-node attachment.
  // -------------------------------------------------------------------------

  private checkNodeDescriptor(value: unknown): NodeDescriptor {
    const record = checkDescriptor(value, "node", "unknown-node");
    checkNoExtraKeys(record, ["kind", "textDigest"], "node", "unknown-node");
    return { kind: checkKind(record["kind"]), textDigest: checkTextDigest(record["textDigest"]) };
  }

  private checkListenerDescriptor(value: unknown): ListenerDescriptor {
    const record = checkDescriptor(value, "listener", "unknown-listener");
    checkNoExtraKeys(record, ["event"], "listener", "unknown-listener");
    return { event: checkEvent(record["event"]) };
  }

  // Seed one inert node double. Node ids are single-use per context: a
  // re-seed refuses with node-closed rather than swapping the double.
  seedNode(owner: string, context: string, nodeId: string, descriptor: unknown): NodeFacts {
    const node = this.checkNodeDescriptor(descriptor);
    const record = this.requireLiveBinding(owner, context);
    const id = checkNodeId(nodeId);
    let total = 0;
    for (const contextRecord of this.contexts.values()) {
      total += contextRecord.nodes.size;
    }
    if (total >= this.limits.maxNodes) {
      fail("capacity-exhausted", "node table full");
    }
    if (record.nodes.has(id)) {
      fail("node-closed", `node id is single-use and already seeded: ${id}`);
    }
    const entry: NodeRecord = {
      nodeId: id,
      owner,
      context,
      kind: node.kind,
      textDigest: node.textDigest,
      cloneOf: null,
      parentId: null,
    };
    record.nodes.set(id, entry);
    return this.nodeFactsOf(record, entry);
  }

  private nodeFactsOf(record: ContextRecord, node: NodeRecord): NodeFacts {
    return Object.freeze({
      nodeId: node.nodeId,
      owner: node.owner,
      context: record.context,
      kind: node.kind,
      textDigest: node.textDigest,
      cloneOf: node.cloneOf,
      digest: digestText(
        `node:${node.nodeId}:${node.kind}:${node.textDigest}:${node.cloneOf ?? ""}`,
      ),
    });
  }

  nodeFacts(owner: string, context: string, nodeId: string): NodeFacts {
    const record = this.requireContext(owner, context);
    const node = record.nodes.get(nodeId);
    if (node === undefined) {
      fail("unknown-node", `no seeded node: ${nodeId}`);
    }
    return this.nodeFactsOf(record, node as NodeRecord);
  }

  // Attach one listener to one node for one event. Re-attaching the same
  // id with the identical (node, event) pair rejoins; a different pair
  // refuses with listener-mismatch rather than retargeting the listener.
  addListener(
    owner: string,
    context: string,
    nodeId: string,
    listenerId: string,
    descriptor: unknown,
  ): ListenerFacts {
    const listener = this.checkListenerDescriptor(descriptor);
    const record = this.requireLiveBinding(owner, context);
    const node = checkNodeId(nodeId);
    const id = checkListenerId(listenerId);
    let total = 0;
    for (const contextRecord of this.contexts.values()) {
      total += contextRecord.listeners.size;
    }
    if (total >= this.limits.maxListeners) {
      fail("capacity-exhausted", "listener table full");
    }
    const nodeRecord = record.nodes.get(node);
    if (nodeRecord === undefined) {
      fail("unknown-node", `listener attaches to no seeded node: ${node}`);
    }
    const prior = record.listeners.get(id);
    if (prior !== undefined) {
      if (prior.nodeId !== node || prior.event !== listener.event) {
        fail("listener-mismatch", `listener id is already attached elsewhere: ${id}`);
      }
      return this.listenerFactsOf(record, prior);
    }
    const entry: ListenerRecord = {
      listenerId: id,
      owner,
      context,
      nodeId: node,
      event: listener.event,
    };
    record.listeners.set(id, entry);
    return this.listenerFactsOf(record, entry);
  }

  private listenerFactsOf(record: ContextRecord, listener: ListenerRecord): ListenerFacts {
    return Object.freeze({
      listenerId: listener.listenerId,
      owner: listener.owner,
      context: record.context,
      nodeId: listener.nodeId,
      event: listener.event,
      digest: digestText(`listener:${listener.listenerId}:${listener.nodeId}:${listener.event}`),
    });
  }

  listenerFacts(owner: string, context: string, listenerId: string): ListenerFacts {
    const record = this.requireContext(owner, context);
    const listener = record.listeners.get(listenerId);
    if (listener === undefined) {
      fail("unknown-listener", `no attached listener: ${listenerId}`);
    }
    return this.listenerFactsOf(record, listener as ListenerRecord);
  }

  // -------------------------------------------------------------------------
  // Same-task batches: finite clone/append/emit runs.
  // -------------------------------------------------------------------------

  private checkCloneDescriptor(value: unknown): CloneDescriptor {
    const record = checkDescriptor(value, "clone", "unknown-node");
    checkNoExtraKeys(record, ["nodeId"], "clone", "unknown-node");
    return { nodeId: checkNodeId(record["nodeId"]) };
  }

  private checkAppendDescriptor(value: unknown): AppendDescriptor {
    const record = checkDescriptor(value, "append", "unknown-node");
    checkNoExtraKeys(record, ["parentId", "childId"], "append", "unknown-node");
    return {
      parentId: checkNodeId(record["parentId"]),
      childId: checkNodeId(record["childId"]),
    };
  }

  private checkEmitDescriptor(value: unknown): EmitDescriptor {
    const record = checkDescriptor(value, "emit", "unknown-occurrence");
    checkNoExtraKeys(record, ["nodeId", "occurrenceId", "event"], "emit", "unknown-occurrence");
    return {
      nodeId: checkNodeId(record["nodeId"]),
      occurrenceId: checkOccurrenceId(record["occurrenceId"]),
      event: checkEvent(record["event"]),
    };
  }

  // Begin one same-task batch. The batch token is minted here and verified
  // by table lookup on every later mutation, so an invented token is
  // never authority. Batch ids are single-use per context once sealed: a
  // sealed id never re-opens. Re-beginning an open id rejoins.
  beginBatch(owner: string, context: string, batchId: string): BatchAttestation {
    const record = this.requireLiveBinding(owner, context);
    const id = checkBatchId(batchId);
    let total = 0;
    for (const contextRecord of this.contexts.values()) {
      total += contextRecord.batches.size;
    }
    if (total >= this.limits.maxBatches) {
      fail("capacity-exhausted", "batch table full");
    }
    const prior = record.batches.get(id);
    if (prior !== undefined) {
      if (prior.sealed) {
        fail("batch-closed", `batch id is single-use and already sealed: ${id}`);
      }
      return this.attestationOf(record, prior);
    }
    const entry: BatchRecord = {
      batchId: id,
      owner,
      context,
      token: mintToken("batch"),
      taskSeq: record.taskSeq,
      sealed: false,
      ops: [],
      yielded: false,
      yieldSeq: null,
    };
    record.batches.set(id, entry);
    return this.attestationOf(record, entry);
  }

  private attestationOf(record: ContextRecord, batch: BatchRecord): BatchAttestation {
    return Object.freeze({
      batchId: batch.batchId,
      owner: batch.owner,
      context: record.context,
      taskSeq: batch.taskSeq,
      contextDigest: digestText(record.token),
      handleDigest: digestText(batch.token),
      open: true as const,
    });
  }

  attestation(owner: string, context: string, batchId: string): BatchAttestation {
    const record = this.requireContext(owner, context);
    const batch = this.requireBatch(record, batchId);
    if (batch.sealed) {
      fail("batch-closed", `batch is sealed, not open: ${batchId}`);
    }
    return this.attestationOf(record, batch);
  }

  private requireBatch(record: ContextRecord, batchId: string): BatchRecord {
    const batch = record.batches.get(batchId);
    if (batch === undefined) {
      fail("unknown-batch", `no begun batch: ${batchId}`);
    }
    return batch as BatchRecord;
  }

  // Verify the batch token by table lookup. Unknown batches and invented
  // tokens are never authority; sealed batches report batch-closed.
  private requireLiveBatch(
    owner: string,
    context: string,
    batchId: string,
    token: string,
  ): { record: ContextRecord; batch: BatchRecord } {
    const record = this.requireContext(owner, context);
    const batch = this.requireBatch(record, batchId);
    if (batch.sealed) {
      fail("batch-closed", `batch is sealed: ${batchId}`);
    }
    if (token === "" || token !== batch.token) {
      fail("forged-token", "batch token is not the attested token");
    }
    return { record, batch };
  }

  // A yielded batch refuses further operations: the yield is detectable at
  // the next op, and the sealed facts carry the yield provenance.
  private requireImmediate(batch: BatchRecord): void {
    if (batch.yielded) {
      fail("batch-yielded", `batch was interrupted by a task yield: ${batch.batchId}`);
    }
  }

  private requireOpCapacity(batch: BatchRecord): void {
    if (batch.ops.length >= this.limits.maxOpsPerBatch) {
      fail("capacity-exhausted", "batch op table full for this batch");
    }
  }

  // Test-only accessor: the raw token crosses exactly here so bounded
  // controls can present it on later calls. Facts carry digests only.
  batchTokenForTest(owner: string, context: string, batchId: string): string {
    const record = this.requireContext(owner, context);
    return this.requireBatch(record, batchId).token;
  }

  // Clone one node inside an open batch. The clone id is minted fresh and
  // can never equal the original id, so a returned original handle is
  // detectable: see original-returned and assertCloneFresh.
  cloneInBatch(
    owner: string,
    context: string,
    batchId: string,
    token: string,
    descriptor: unknown,
  ): CloneFacts {
    const clone = this.checkCloneDescriptor(descriptor);
    const { record, batch } = this.requireLiveBatch(owner, context, batchId, token);
    this.requireImmediate(batch);
    const original = record.nodes.get(clone.nodeId);
    if (original === undefined) {
      fail("unknown-node", `clone names no seeded node: ${clone.nodeId}`);
    }
    let total = 0;
    for (const contextRecord of this.contexts.values()) {
      total += contextRecord.nodes.size;
    }
    if (total >= this.limits.maxNodes) {
      fail("capacity-exhausted", "node table full");
    }
    this.requireOpCapacity(batch);
    let cloneId = "";
    for (let attempt = 0; attempt <= this.limits.maxNodes; attempt += 1) {
      const candidate = `${clone.nodeId}-clone-${String(record.cloneSeq)}`;
      record.cloneSeq += 1;
      if (!record.nodes.has(candidate)) {
        cloneId = candidate;
        break;
      }
    }
    if (cloneId === "") {
      fail("capacity-exhausted", "no fresh clone id available");
    }
    const source = original as NodeRecord;
    record.nodes.set(cloneId, {
      nodeId: cloneId,
      owner,
      context,
      kind: source.kind,
      textDigest: source.textDigest,
      cloneOf: source.nodeId,
      parentId: null,
    });
    const seq = batch.ops.length;
    batch.ops.push({ seq, op: "clone", ref: cloneId });
    return Object.freeze({
      originalId: source.nodeId,
      cloneId,
      owner,
      context,
      batchId: batch.batchId,
      opSeq: seq,
      digest: digestClone(source.nodeId, cloneId, owner, context, batch.batchId, seq),
    });
  }

  // Append one minted clone under one parent inside an open batch. The
  // child must be a minted clone of this context: presenting a seeded
  // original (or any non-clone) as the child refuses with
  // original-returned, so a returned original handle never appends
  // silently.
  appendInBatch(
    owner: string,
    context: string,
    batchId: string,
    token: string,
    descriptor: unknown,
  ): AppendFacts {
    const append = this.checkAppendDescriptor(descriptor);
    const { record, batch } = this.requireLiveBatch(owner, context, batchId, token);
    this.requireImmediate(batch);
    const parent = record.nodes.get(append.parentId);
    if (parent === undefined) {
      fail("unknown-node", `append names no seeded parent: ${append.parentId}`);
    }
    const child = record.nodes.get(append.childId);
    if (child === undefined) {
      fail("unknown-node", `append names no seeded child: ${append.childId}`);
    }
    const childRecord = child as NodeRecord;
    if (childRecord.cloneOf === null) {
      fail(
        "original-returned",
        `append child is an original handle, not a minted clone: ${append.childId}`,
      );
    }
    this.requireOpCapacity(batch);
    childRecord.parentId = (parent as NodeRecord).nodeId;
    const seq = batch.ops.length;
    batch.ops.push({ seq, op: "append", ref: `${append.parentId}:${append.childId}` });
    return Object.freeze({
      parentId: append.parentId,
      childId: append.childId,
      owner,
      context,
      batchId: batch.batchId,
      opSeq: seq,
      digest: digestAppend(append.parentId, append.childId, owner, context, batch.batchId, seq),
    });
  }

  // Emit one occurrence on one node inside an open batch. Occurrence ids
  // are single-use per context: a re-emit refuses with occurrence-closed
  // rather than swapping the record.
  emitInBatch(
    owner: string,
    context: string,
    batchId: string,
    token: string,
    descriptor: unknown,
  ): OccurrenceFacts {
    const emit = this.checkEmitDescriptor(descriptor);
    const { record, batch } = this.requireLiveBatch(owner, context, batchId, token);
    this.requireImmediate(batch);
    const node = record.nodes.get(emit.nodeId);
    if (node === undefined) {
      fail("unknown-node", `emit names no seeded node: ${emit.nodeId}`);
    }
    let total = 0;
    for (const contextRecord of this.contexts.values()) {
      total += contextRecord.occurrences.size;
    }
    if (total >= this.limits.maxOccurrences) {
      fail("capacity-exhausted", "occurrence table full");
    }
    if (record.occurrences.has(emit.occurrenceId)) {
      fail(
        "occurrence-closed",
        `occurrence id is single-use and already recorded: ${emit.occurrenceId}`,
      );
    }
    this.requireOpCapacity(batch);
    const entry: OccurrenceRecord = {
      occurrenceId: emit.occurrenceId,
      owner,
      context,
      event: emit.event,
      nodeId: emit.nodeId,
      batchId: batch.batchId,
      taskSeq: record.taskSeq,
      microtaskSeq: null,
      cancelled: false,
      cancelMode: null,
      acks: 0,
    };
    record.occurrences.set(emit.occurrenceId, entry);
    const seq = batch.ops.length;
    batch.ops.push({ seq, op: "emit", ref: emit.occurrenceId });
    return this.occurrenceFactsOf(record, entry);
  }

  // Seal one open batch. Sealing stays available after a yield so the
  // yield provenance is always readable: the sealed facts carry yielded
  // and the interrupting yieldSeq. Sealed ids never re-open.
  sealBatch(owner: string, context: string, batchId: string, token: string): BatchFacts {
    const { record, batch } = this.requireLiveBatch(owner, context, batchId, token);
    batch.sealed = true;
    return this.batchFactsOf(record, batch);
  }

  private batchFactsOf(record: ContextRecord, batch: BatchRecord): BatchFacts {
    const ops: OpFacts[] = batch.ops.map((entry) =>
      Object.freeze({ seq: entry.seq, op: entry.op, ref: entry.ref }),
    );
    return Object.freeze({
      batchId: batch.batchId,
      owner: batch.owner,
      context: record.context,
      taskSeq: batch.taskSeq,
      sealed: batch.sealed,
      ops: Object.freeze(ops),
      yielded: batch.yielded,
      yieldSeq: batch.yieldSeq,
      digest: digestBatch(
        batch.batchId,
        batch.owner,
        record.context,
        batch.taskSeq,
        batch.sealed,
        batch.ops,
        batch.yielded,
        batch.yieldSeq,
      ),
    });
  }

  // Batch facts stay readable while open and after sealing. Reads take no
  // token.
  batchFacts(owner: string, context: string, batchId: string): BatchFacts {
    const record = this.requireContext(owner, context);
    return this.batchFactsOf(record, this.requireBatch(record, batchId));
  }

  // -------------------------------------------------------------------------
  // Acknowledgment, cancellation, and the microtask-order observer.
  // -------------------------------------------------------------------------

  private checkAckDescriptor(value: unknown): AckDescriptor {
    const record = checkDescriptor(value, "acknowledgment", "unknown-occurrence");
    checkNoExtraKeys(
      record,
      ["occurrenceId", "nodeId", "listenerId"],
      "acknowledgment",
      "unknown-occurrence",
    );
    return {
      occurrenceId: checkOccurrenceId(record["occurrenceId"]),
      nodeId: checkNodeId(record["nodeId"]),
      listenerId: checkListenerId(record["listenerId"]),
    };
  }

  private checkCancelDescriptor(value: unknown): CancelDescriptor {
    const record = checkDescriptor(value, "cancellation", "wrong-cancellation");
    checkNoExtraKeys(
      record,
      ["occurrenceId", "nodeId", "listenerId", "mode"],
      "cancellation",
      "wrong-cancellation",
    );
    return {
      occurrenceId: checkOccurrenceId(record["occurrenceId"]),
      nodeId: checkNodeId(record["nodeId"]),
      listenerId: checkListenerId(record["listenerId"]),
      mode: checkCancelMode(record["mode"]),
    };
  }

  private checkObserveDescriptor(value: unknown): ObserveDescriptor {
    const record = checkDescriptor(value, "observer script", "unknown-occurrence");
    checkNoExtraKeys(record, ["order"], "observer script", "unknown-occurrence");
    const order = record["order"];
    if (!Array.isArray(order) || order.length === 0) {
      fail("unknown-occurrence", "observer script carries a non-empty occurrence order");
    }
    const checked: string[] = [];
    for (const entry of order) {
      checked.push(checkOccurrenceId(entry));
    }
    return { order: checked };
  }

  private requireOccurrence(record: ContextRecord, occurrenceId: string): OccurrenceRecord {
    const occurrence = record.occurrences.get(occurrenceId);
    if (occurrence === undefined) {
      fail("unknown-occurrence", `no recorded occurrence: ${occurrenceId}`);
    }
    return occurrence as OccurrenceRecord;
  }

  private occurrenceFactsOf(record: ContextRecord, occurrence: OccurrenceRecord): OccurrenceFacts {
    return Object.freeze({
      occurrenceId: occurrence.occurrenceId,
      owner: occurrence.owner,
      context: record.context,
      event: occurrence.event,
      nodeId: occurrence.nodeId,
      batchId: occurrence.batchId,
      taskSeq: occurrence.taskSeq,
      microtaskSeq: occurrence.microtaskSeq,
      cancelled: occurrence.cancelled,
      cancelMode: occurrence.cancelMode,
      acks: occurrence.acks,
      digest: digestOccurrence(
        occurrence.occurrenceId,
        occurrence.owner,
        record.context,
        occurrence.event,
        occurrence.nodeId,
        occurrence.batchId,
        occurrence.taskSeq,
        occurrence.microtaskSeq,
        occurrence.cancelled,
        occurrence.cancelMode,
        occurrence.acks,
      ),
    });
  }

  occurrenceFacts(owner: string, context: string, occurrenceId: string): OccurrenceFacts {
    const record = this.requireContext(owner, context);
    return this.occurrenceFactsOf(record, this.requireOccurrence(record, occurrenceId));
  }

  // Acknowledge one occurrence through one same-node listener. The
  // descriptor node must be the occurrence's recorded node, and the
  // listener must be attached to that same node for the occurrence's
  // event: anything else is a same-node mismatch, never a quiet ack.
  acknowledge(owner: string, context: string, descriptor: unknown): AckFacts {
    const ack = this.checkAckDescriptor(descriptor);
    const record = this.requireLiveBinding(owner, context);
    const occurrence = this.requireOccurrence(record, ack.occurrenceId);
    if (ack.nodeId !== occurrence.nodeId) {
      fail("listener-mismatch", `acknowledgment names another node: ${ack.nodeId}`);
    }
    const listener = record.listeners.get(ack.listenerId);
    if (listener === undefined) {
      fail("unknown-listener", `acknowledgment names no attached listener: ${ack.listenerId}`);
    }
    const listenerRecord = listener as ListenerRecord;
    if (listenerRecord.nodeId !== occurrence.nodeId || listenerRecord.event !== occurrence.event) {
      fail(
        "listener-mismatch",
        `listener is not a same-node listener of this occurrence: ${ack.listenerId}`,
      );
    }
    const ackSeq = occurrence.acks;
    occurrence.acks += 1;
    return Object.freeze({
      occurrenceId: occurrence.occurrenceId,
      nodeId: occurrence.nodeId,
      listenerId: listenerRecord.listenerId,
      ackSeq,
      digest: digestText(
        `ack:${occurrence.occurrenceId}:${occurrence.nodeId}:${listenerRecord.listenerId}:${String(ackSeq)}`,
      ),
    });
  }

  // Cancel one occurrence through one same-node listener with an explicit
  // mode. The descriptor must name the exact recorded (occurrence, node)
  // pair and a listener attached to that same node for the occurrence's
  // event: a wrong node, a foreign listener, or an undeclared mode is a
  // wrong cancellation, never a quiet cancel. Each occurrence cancels at
  // most once.
  cancelOccurrence(owner: string, context: string, descriptor: unknown): CancelFacts {
    const cancel = this.checkCancelDescriptor(descriptor);
    const record = this.requireLiveBinding(owner, context);
    const occurrence = this.requireOccurrence(record, cancel.occurrenceId);
    if (cancel.nodeId !== occurrence.nodeId) {
      fail("wrong-cancellation", `cancellation names another node: ${cancel.nodeId}`);
    }
    const listener = record.listeners.get(cancel.listenerId);
    if (listener === undefined) {
      fail("unknown-listener", `cancellation names no attached listener: ${cancel.listenerId}`);
    }
    const listenerRecord = listener as ListenerRecord;
    if (listenerRecord.nodeId !== occurrence.nodeId || listenerRecord.event !== occurrence.event) {
      fail(
        "wrong-cancellation",
        `listener is not a same-node listener of this occurrence: ${cancel.listenerId}`,
      );
    }
    if (occurrence.cancelled) {
      fail("already-cancelled", `occurrence is already cancelled: ${cancel.occurrenceId}`);
    }
    occurrence.cancelled = true;
    occurrence.cancelMode = cancel.mode;
    return Object.freeze({
      occurrenceId: occurrence.occurrenceId,
      nodeId: occurrence.nodeId,
      listenerId: listenerRecord.listenerId,
      mode: cancel.mode,
      digest: digestText(
        `cancel:${occurrence.occurrenceId}:${occurrence.nodeId}:${listenerRecord.listenerId}:${cancel.mode}`,
      ),
    });
  }

  // Observe recorded occurrences in one explicitly scripted microtask
  // order. There is no wall clock: the order array assigns microtask
  // sequence numbers slot by slot, and the transcript preserves it. Every
  // listed occurrence must be recorded and unobserved; a missing
  // occurrence refuses with unknown-occurrence, and a repeated or
  // already observed one refuses with occurrence-closed.
  observeMicrotasks(owner: string, context: string, descriptor: unknown): TranscriptFacts {
    const script = this.checkObserveDescriptor(descriptor);
    const record = this.requireLiveBinding(owner, context);
    let total = 0;
    for (const contextRecord of this.contexts.values()) {
      total += contextRecord.transcripts.length;
    }
    if (total >= this.limits.maxTranscripts) {
      fail("capacity-exhausted", "transcript table full");
    }
    const seen = new Set<string>();
    for (const occurrenceId of script.order) {
      if (seen.has(occurrenceId)) {
        fail("occurrence-closed", `observer script repeats an occurrence: ${occurrenceId}`);
      }
      seen.add(occurrenceId);
      const occurrence = this.requireOccurrence(record, occurrenceId);
      if (occurrence.microtaskSeq !== null) {
        fail("occurrence-closed", `occurrence is already observed: ${occurrenceId}`);
      }
    }
    let slot = 0;
    for (const contextRecord of this.contexts.values()) {
      for (const occurrence of contextRecord.occurrences.values()) {
        if (occurrence.microtaskSeq !== null && occurrence.microtaskSeq >= slot) {
          slot = occurrence.microtaskSeq + 1;
        }
      }
    }
    const order: string[] = [];
    for (const occurrenceId of script.order) {
      const occurrence = record.occurrences.get(occurrenceId) as OccurrenceRecord;
      occurrence.microtaskSeq = slot;
      slot += 1;
      order.push(occurrenceId);
    }
    const entry: TranscriptRecord = {
      transcriptSeq: record.transcripts.length,
      owner,
      context,
      order,
      taskSeq: record.taskSeq,
    };
    record.transcripts.push(entry);
    return this.transcriptFactsOf(record, entry);
  }

  private transcriptFactsOf(record: ContextRecord, transcript: TranscriptRecord): TranscriptFacts {
    return Object.freeze({
      transcriptSeq: transcript.transcriptSeq,
      owner: transcript.owner,
      context: record.context,
      order: Object.freeze([...transcript.order]),
      taskSeq: transcript.taskSeq,
      digest: digestText(
        `transcript:${String(transcript.transcriptSeq)}:${transcript.order.join(",")}`,
      ),
    });
  }

  transcriptFacts(owner: string, context: string, transcriptSeq: number): TranscriptFacts {
    const record = this.requireContext(owner, context);
    if (!Number.isSafeInteger(transcriptSeq) || transcriptSeq < 0) {
      fail("unknown-transcript", `no recorded transcript: ${String(transcriptSeq)}`);
    }
    const transcript = record.transcripts[transcriptSeq];
    if (transcript === undefined) {
      fail("unknown-transcript", `no recorded transcript: ${String(transcriptSeq)}`);
    }
    return this.transcriptFactsOf(record, transcript as TranscriptRecord);
  }
}

// ---------------------------------------------------------------------------
// Verdicts: only verified clone, batch, and occurrence facts judge
// ---------------------------------------------------------------------------

function checkOccurrenceProofShape(value: unknown): OccurrenceProofClaim {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail("forbidden-proof", "occurrence proof must be an observed-occurrence object");
  }
  const record = value as Record<string, unknown>;
  if (record["kind"] === "observed-occurrence") {
    const occurrence = record["occurrence"] as OccurrenceFacts;
    if (occurrence === null || typeof occurrence !== "object" || Array.isArray(occurrence)) {
      fail("forbidden-proof", "occurrence proof carries no occurrence facts");
    }
    return { kind: "observed-occurrence", occurrence };
  }
  if (typeof record["kind"] === "string") {
    fail("forbidden-proof", `occurrence proof kind is inadmissible: ${String(record["kind"])}`);
  }
  fail("forbidden-proof", "occurrence proof carries no kind");
}

// Judge one clone-freshness claim. Only a verified clone whose minted id
// differs from the original proves freshness: a returned original handle
// rejects with original-returned, and malformed or unverifiable facts
// reject with unknown-node before any verdict is read.
export function assertCloneFresh(facts: unknown): CloneVerdict {
  if (facts === null || typeof facts !== "object" || Array.isArray(facts)) {
    fail("unknown-node", "clone proof must be a clone facts object");
  }
  const record = facts as Record<string, unknown>;
  const originalId = record["originalId"];
  const cloneId = record["cloneId"];
  const owner = record["owner"];
  const context = record["context"];
  const batchId = record["batchId"];
  const opSeq = record["opSeq"];
  const digest = record["digest"];
  if (
    typeof originalId !== "string" ||
    typeof cloneId !== "string" ||
    typeof owner !== "string" ||
    typeof context !== "string" ||
    typeof batchId !== "string" ||
    !Number.isSafeInteger(opSeq) ||
    typeof digest !== "string"
  ) {
    fail("unknown-node", "clone proof carries malformed facts");
  }
  const recomputed = digestClone(
    originalId as string,
    cloneId as string,
    owner as string,
    context as string,
    batchId as string,
    opSeq as number,
  );
  if (recomputed !== digest) {
    fail("unknown-node", "clone digest does not verify");
  }
  if (cloneId === originalId) {
    fail("original-returned", "clone id is the returned original handle");
  }
  return Object.freeze({
    originalId: originalId as string,
    cloneId: cloneId as string,
    fresh: true as const,
    digest: digestText(`clone-verdict:${originalId as string}:${cloneId as string}`),
  });
}

function checkBatchFactsShape(value: unknown): {
  batchId: string;
  owner: string;
  context: string;
  taskSeq: number;
  sealed: boolean;
  ops: { seq: number; op: string; ref: string }[];
  yielded: boolean;
  yieldSeq: number | null;
  digest: string;
} {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail("unknown-batch", "batch proof must be a batch facts object");
  }
  const record = value as Record<string, unknown>;
  const batchId = record["batchId"];
  const owner = record["owner"];
  const context = record["context"];
  const taskSeq = record["taskSeq"];
  const sealed = record["sealed"];
  const ops = record["ops"];
  const yielded = record["yielded"];
  const yieldSeq = record["yieldSeq"];
  const digest = record["digest"];
  if (
    typeof batchId !== "string" ||
    typeof owner !== "string" ||
    typeof context !== "string" ||
    !Number.isSafeInteger(taskSeq) ||
    typeof sealed !== "boolean" ||
    !Array.isArray(ops) ||
    typeof yielded !== "boolean" ||
    (yieldSeq !== null && !Number.isSafeInteger(yieldSeq)) ||
    typeof digest !== "string"
  ) {
    fail("unknown-batch", "batch proof carries malformed facts");
  }
  const checked: { seq: number; op: string; ref: string }[] = [];
  for (const entry of ops as unknown[]) {
    if (entry === null || typeof entry !== "object" || Array.isArray(entry)) {
      fail("unknown-batch", "batch proof carries a malformed op");
    }
    const op = entry as Record<string, unknown>;
    if (
      !Number.isSafeInteger(op["seq"]) ||
      typeof op["op"] !== "string" ||
      !(DOM_OPS as readonly string[]).includes(op["op"] as string) ||
      typeof op["ref"] !== "string"
    ) {
      fail("unknown-batch", "batch proof carries an undeclared op");
    }
    checked.push({ seq: op["seq"] as number, op: op["op"] as string, ref: op["ref"] as string });
  }
  return {
    batchId: batchId as string,
    owner: owner as string,
    context: context as string,
    taskSeq: taskSeq as number,
    sealed: sealed as boolean,
    ops: checked,
    yielded: yielded as boolean,
    yieldSeq: yieldSeq as number | null,
    digest: digest as string,
  };
}

// Judge one batch-immediacy claim. Only a verified sealed batch that ran
// within one task proves immediacy: an unsealed batch refuses with
// batch-pending, a yielded batch refuses with batch-yielded, and
// malformed or unverifiable facts refuse with unknown-batch.
export function assertBatchImmediate(facts: unknown): BatchVerdict {
  const batch = checkBatchFactsShape(facts);
  const recomputed = digestBatch(
    batch.batchId,
    batch.owner,
    batch.context,
    batch.taskSeq,
    batch.sealed,
    batch.ops,
    batch.yielded,
    batch.yieldSeq,
  );
  if (recomputed !== batch.digest) {
    fail("unknown-batch", "batch digest does not verify");
  }
  if (!batch.sealed) {
    fail("batch-pending", "batch is still open, not sealed");
  }
  if (batch.yielded) {
    fail("batch-yielded", "batch was interrupted by a task yield");
  }
  return Object.freeze({
    batchId: batch.batchId,
    immediate: true as const,
    ops: batch.ops.length,
    digest: digestText(`batch-verdict:${batch.batchId}:${String(batch.ops.length)}`),
  });
}

// Judge one occurrence proof claim. Only a verified observed occurrence
// proves observation: unobserved-occurrence, batch-op, listener-ack, and
// any other non-observation claim reject with forbidden-proof before any
// verdict is read, and a recorded but unobserved occurrence rejects with
// occurrence-missing.
export function assertOccurrenceObserved(claim: unknown): OccurrenceVerdict {
  const proof = checkOccurrenceProofShape(claim);
  if (proof.kind !== "observed-occurrence") {
    fail("forbidden-proof", `occurrence proof kind is inadmissible: ${proof.kind}`);
  }
  const admitted = proof as { kind: "observed-occurrence"; occurrence: OccurrenceFacts };
  const facts = admitted.occurrence;
  if (
    typeof facts.occurrenceId !== "string" ||
    typeof facts.owner !== "string" ||
    typeof facts.context !== "string" ||
    typeof facts.event !== "string" ||
    typeof facts.nodeId !== "string" ||
    typeof facts.batchId !== "string" ||
    !Number.isSafeInteger(facts.taskSeq) ||
    (facts.microtaskSeq !== null && !Number.isSafeInteger(facts.microtaskSeq)) ||
    typeof facts.cancelled !== "boolean" ||
    (facts.cancelMode !== null && typeof facts.cancelMode !== "string") ||
    !Number.isSafeInteger(facts.acks) ||
    typeof facts.digest !== "string"
  ) {
    fail("forbidden-proof", "occurrence proof carries malformed facts");
  }
  if (!(DOM_EVENTS as readonly string[]).includes(facts.event)) {
    fail("forbidden-proof", `occurrence carries an undeclared event: ${facts.event}`);
  }
  if (
    facts.cancelMode !== null &&
    !(CANCEL_MODES as readonly string[]).includes(facts.cancelMode)
  ) {
    fail("forbidden-proof", `occurrence carries an undeclared cancel mode: ${facts.cancelMode}`);
  }
  const recomputed = digestOccurrence(
    facts.occurrenceId,
    facts.owner,
    facts.context,
    facts.event,
    facts.nodeId,
    facts.batchId,
    facts.taskSeq,
    facts.microtaskSeq,
    facts.cancelled,
    facts.cancelMode,
    facts.acks,
  );
  if (recomputed !== facts.digest) {
    fail("forbidden-proof", "occurrence digest does not verify");
  }
  if (facts.microtaskSeq === null) {
    fail("occurrence-missing", "occurrence was recorded but never observed");
  }
  return Object.freeze({
    occurrenceId: facts.occurrenceId,
    observed: true as const,
    microtaskSeq: facts.microtaskSeq,
    digest: digestText(`occurrence-verdict:${facts.occurrenceId}:${String(facts.microtaskSeq)}`),
  });
}
