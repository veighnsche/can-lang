// K01: frozen native-value typed operation schema.
//
// Finite operation catalogue, inert handle types, and the sync/async
// assimilation and ownership contracts from the shared capability design
// (native.open/describe/make/invoke/settle/observe/gate/release/fault/
// restore/close). Local schema validation only: no transports, services,
// timers, or live runtimes.
//
// Hard exclusions, enforced by the validators below and the bounded
// self-check in schema-check.ts:
//   - no arbitrary module path / export name / eval source argument;
//   - no raw hostile value (function, thenable, Promise) in any reply;
//   - no expected outcome / oracle field in any request or reply;
//   - no unlimited or missing handle/pending/byte bound.

export const NATIVE_VALUE_SCHEMA_VERSION = "1" as const;

export const NATIVE_VALUE_PACKAGE = "native" as const;
export const NATIVE_VALUE_IDENTITY = "can.test.native@1" as const;

// ---------------------------------------------------------------------------
// Finite operation catalogue
// ---------------------------------------------------------------------------

export const NATIVE_VALUE_OPERATIONS = [
  "native.open",
  "native.describe",
  "native.make",
  "native.invoke",
  "native.settle",
  "native.observe",
  "native.gate",
  "native.release",
  "native.fault",
  "native.restore",
  "native.close",
] as const;

export type NativeValueOperationName = (typeof NATIVE_VALUE_OPERATIONS)[number];

// timing declares which side owns asynchrony. "sync" operations complete
// inside the call with an inert reply. "dispatch" operations complete inside
// the call with either an inert value handle or a pending-action handle; the
// pending action itself settles only through native.settle. "poll" observes
// a pending action without adding assimilation.
export type OperationTiming = "sync" | "dispatch" | "poll";

// assimilation declares exactly which thenable/await behavior belongs to the
// native API itself. "none" means the RPC wrapper must not read `then`, call
// anything, or await; the reply envelope is inert data. "native-only" means
// only the await that is part of the documented native API semantics may
// run, inside the service realm, and its outcome still crosses the boundary
// as an inert handle/completion envelope, never as a raw value.
export type AssimilationContract = "none" | "native-only";

export type OperationReply =
  | "session-handle"
  | "descriptor-or-unknown"
  | "value-handle"
  | "handle-or-pending-action"
  | "settlement-or-pending"
  | "inert-facts"
  | "gate-handle"
  | "release-facts"
  | "fault-handle"
  | "restore-outcome"
  | "close-receipt";

export type CatalogueArg = Readonly<{
  name: string;
  type: string;
  required: boolean;
  note: string;
}>;

export type CatalogueEntry = Readonly<{
  name: NativeValueOperationName;
  identity: string;
  summary: string;
  timing: OperationTiming;
  assimilation: AssimilationContract;
  reply: OperationReply;
  args: readonly CatalogueArg[];
  facts: readonly string[];
  // Handle kinds consumed (must be presented) and produced (minted) by the
  // operation. Sessions never cross: every handle binds its session id.
  handlesIn: readonly HandleKind[];
  handlesOut: readonly HandleKind[];
}>;

function freeze<T>(value: T): Readonly<T> {
  if (value !== null && typeof value === "object") {
    for (const child of Object.values(value)) freeze(child);
    Object.freeze(value);
  }
  return value;
}

const SESSION_ARG: CatalogueArg = {
  name: "session",
  type: "native::session",
  required: true,
  note: "Owning session handle; cross-session use rejects.",
};

export const NATIVE_VALUE_CATALOGUE: readonly CatalogueEntry[] = freeze([
  {
    name: "native.open",
    identity: `${NATIVE_VALUE_IDENTITY}::open`,
    summary: "Open an isolated subject/observation session for one registered runtime.",
    timing: "sync",
    assimilation: "none",
    reply: "session-handle",
    args: [
      {
        name: "runtime",
        type: "native::registered_runtime",
        required: true,
        note: "Closed enum of reviewed runtimes; never a module path.",
      },
      {
        name: "observer",
        type: "native::observer_id",
        required: true,
        note: "Reviewed observer identity.",
      },
      {
        name: "scope",
        type: "native::scope_ref",
        required: true,
        note: "Run/case/attempt scope reference.",
      },
      {
        name: "limits",
        type: "native::limits",
        required: true,
        note: "Finite session limits; see NativeValueLimits.",
      },
    ],
    facts: ["session", "supported_catalogue"],
    handlesIn: [],
    handlesOut: ["session"],
  },
  {
    name: "native.describe",
    identity: `${NATIVE_VALUE_IDENTITY}::describe`,
    summary: "Effect-free presence/descriptor metadata for one registered API.",
    timing: "sync",
    assimilation: "none",
    reply: "descriptor-or-unknown",
    args: [
      SESSION_ARG,
      {
        name: "api",
        type: "native::registered_api",
        required: true,
        note: "Closed enum of registered APIs; never module/export/eval text.",
      },
    ],
    facts: ["presence", "descriptor"],
    handlesIn: ["session"],
    handlesOut: [],
  },
  {
    name: "native.make",
    identity: `${NATIVE_VALUE_IDENTITY}::make`,
    summary: "Construct a raw value behind a handle; reports construction facts only.",
    timing: "sync",
    assimilation: "none",
    reply: "value-handle",
    args: [
      SESSION_ARG,
      {
        name: "kind",
        type: "native::make_kind",
        required: true,
        note: "Closed enum: primitive_bits, text, ordered_entries, hostile_descriptor.",
      },
      {
        name: "payload",
        type: "native::inert_literal",
        required: true,
        note: "Tagged inert literal (IEEE bits, exact text, ordered entries, reviewed hostile descriptor).",
      },
    ],
    facts: ["value", "construction"],
    handlesIn: ["session"],
    handlesOut: ["value"],
  },
  {
    name: "native.invoke",
    identity: `${NATIVE_VALUE_IDENTITY}::invoke`,
    summary: "Run exactly one registered typed operation against receiver/argument handles.",
    timing: "dispatch",
    assimilation: "native-only",
    reply: "handle-or-pending-action",
    args: [
      SESSION_ARG,
      {
        name: "operation",
        type: "native::registered_operation",
        required: true,
        note: "Closed enum of registered typed operations; never module/export/eval text.",
      },
      {
        name: "receiver",
        type: "native::value_handle",
        required: false,
        note: "Receiver handle, when the operation takes one.",
      },
      {
        name: "arguments",
        type: "native::value_handle[]",
        required: true,
        note: "Bounded argument-handle list.",
      },
    ],
    facts: ["result_or_action"],
    handlesIn: ["session", "value"],
    handlesOut: ["value", "action"],
  },
  {
    name: "native.settle",
    identity: `${NATIVE_VALUE_IDENTITY}::settle`,
    summary: "Poll one pending action; deadline expiry leaves it pending without assimilation.",
    timing: "poll",
    assimilation: "none",
    reply: "settlement-or-pending",
    args: [
      {
        name: "action",
        type: "native::pending_action",
        required: true,
        note: "Pending-action handle from native.invoke.",
      },
      {
        name: "deadline",
        type: "native::deadline",
        required: true,
        note: "Finite poll deadline; expiry keeps ownership pending.",
      },
    ],
    facts: ["settlement", "pending"],
    handlesIn: ["action"],
    handlesOut: ["value"],
  },
  {
    name: "native.observe",
    identity: `${NATIVE_VALUE_IDENTITY}::observe`,
    summary: "Read inert facts (bits, lexemes, identity, counters) from handles.",
    timing: "sync",
    assimilation: "none",
    reply: "inert-facts",
    args: [
      SESSION_ARG,
      {
        name: "handles",
        type: "native::value_handle[]",
        required: true,
        note: "Bounded handle list; identity is realm-local.",
      },
      {
        name: "kind",
        type: "native::observe_kind",
        required: true,
        note: "Closed enum: scalar_tag, ieee_bits, lexeme, descriptor, entries, identity, bytes, counters, events.",
      },
      {
        name: "bounds",
        type: "native::observe_bounds",
        required: true,
        note: "Finite entry/byte caps; gaps stay explicit.",
      },
    ],
    facts: ["observations", "gaps", "effectful_reads"],
    handlesIn: ["session", "value"],
    handlesOut: [],
  },
  {
    name: "native.gate",
    identity: `${NATIVE_VALUE_IDENTITY}::gate`,
    summary: "Allocate a gate once; only async continuations may suspend on it.",
    timing: "sync",
    assimilation: "none",
    reply: "gate-handle",
    args: [
      SESSION_ARG,
      {
        name: "gate",
        type: "native::gate_spec",
        required: true,
        note: "Named gate spec with finite waiter bound.",
      },
      {
        name: "action",
        type: "native::bounded_action_ref",
        required: true,
        note: "Bounded action the gate guards.",
      },
    ],
    facts: ["gate", "arrival"],
    handlesIn: ["session"],
    handlesOut: ["gate"],
  },
  {
    name: "native.release",
    identity: `${NATIVE_VALUE_IDENTITY}::release`,
    summary: "Release a gate exactly once; repeats join the first release.",
    timing: "sync",
    assimilation: "none",
    reply: "release-facts",
    args: [
      { name: "gate", type: "native::gate", required: true, note: "Gate handle from native.gate." },
    ],
    facts: ["release"],
    handlesIn: ["gate"],
    handlesOut: [],
  },
  {
    name: "native.fault",
    identity: `${NATIVE_VALUE_IDENTITY}::fault`,
    summary: "Install a reviewed fault on a fresh session only.",
    timing: "sync",
    assimilation: "none",
    reply: "fault-handle",
    args: [
      SESSION_ARG,
      {
        name: "target",
        type: "native::reviewed_fault_target",
        required: true,
        note: "Closed enum of reviewed fault targets; never patched onto R or a foreign session.",
      },
      {
        name: "mode",
        type: "native::fault_mode",
        required: true,
        note: "Closed enum: absent, throw, revoked, counted_delegate.",
      },
    ],
    facts: ["fault", "install"],
    handlesIn: ["session"],
    handlesOut: ["fault"],
  },
  {
    name: "native.restore",
    identity: `${NATIVE_VALUE_IDENTITY}::restore`,
    summary: "Remove a fault; reports timing/outcome without replaying effects.",
    timing: "sync",
    assimilation: "none",
    reply: "restore-outcome",
    args: [
      {
        name: "fault",
        type: "native::fault",
        required: true,
        note: "Fault handle from native.fault.",
      },
    ],
    facts: ["restore"],
    handlesIn: ["fault"],
    handlesOut: [],
  },
  {
    name: "native.close",
    identity: `${NATIVE_VALUE_IDENTITY}::close`,
    summary: "Stop admission, settle/cancel work, destroy handles, return a receipt.",
    timing: "sync",
    assimilation: "none",
    reply: "close-receipt",
    args: [
      SESSION_ARG,
      {
        name: "deadline",
        type: "native::deadline",
        required: true,
        note: "Finite close deadline.",
      },
    ],
    facts: ["released", "remaining", "forced"],
    handlesIn: ["session"],
    handlesOut: [],
  },
]);

export function isNativeValueOperation(name: unknown): name is NativeValueOperationName {
  return typeof name === "string" && (NATIVE_VALUE_OPERATIONS as readonly string[]).includes(name);
}

export function describeOperation(name: string): CatalogueEntry {
  const entry = NATIVE_VALUE_CATALOGUE.find((item) => item.name === name);
  if (!entry) {
    throw new NativeSchemaError(
      "rejected",
      "unsupported-capability",
      `unknown native-value operation: ${name}`,
    );
  }
  return entry;
}

// ---------------------------------------------------------------------------
// Outcomes and mechanical kinds (mirrors schemas/native-test/operation)
// ---------------------------------------------------------------------------

export const NATIVE_OUTCOMES = [
  "completed",
  "rejected",
  "failed",
  "deadline",
  "indeterminate",
] as const;
export type NativeOutcome = (typeof NATIVE_OUTCOMES)[number];

export const MECHANICAL_KINDS = [
  "ok",
  "invalid-request",
  "unsupported-capability",
  "wrong-owner",
  "stale-handle",
  "closed-handle",
  "not-found",
  "permission",
  "changed-input",
  "kind-collision",
  "resource-limit",
  "native-io",
  "transport-failure",
  "unresolved-cleanup",
] as const;
export type MechanicalKind = (typeof MECHANICAL_KINDS)[number];

export class NativeSchemaError extends Error {
  readonly outcome: NativeOutcome;
  readonly kind: MechanicalKind;
  constructor(outcome: NativeOutcome, kind: MechanicalKind, message: string) {
    super(message);
    this.name = "NativeSchemaError";
    this.outcome = outcome;
    this.kind = kind;
  }
}

// ---------------------------------------------------------------------------
// Inert handle types
// ---------------------------------------------------------------------------

export const HANDLE_KINDS = ["session", "value", "action", "gate", "fault"] as const;
export type HandleKind = (typeof HANDLE_KINDS)[number];

const handleBrand: unique symbol = Symbol("native-value-handle");

// An in-memory handle. Opaque to callers: it carries owner/kind/session/
// generation authority and never the raw native value. Handles are minted by
// the service only; there is no public constructor from raw values.
export type NativeHandle<K extends HandleKind = HandleKind> = Readonly<{
  readonly [handleBrand]: K;
  kind: K;
  id: string;
  sessionId: string;
  owner: string;
  generation: number;
}>;

// The wire spelling of a handle: plain data with the same authority fields.
// Decoding validates every field; unknown or missing fields reject.
export type NativeHandleWire = Readonly<{
  kind: HandleKind;
  id: string;
  sessionId: string;
  owner: string;
  generation: number;
}>;

const ID_PATTERN = /^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$/;

function check_id(field: string, value: unknown): string {
  if (typeof value !== "string" || !ID_PATTERN.test(value)) {
    throw new NativeSchemaError("rejected", "invalid-request", `invalid ${field}`);
  }
  return value;
}

export function mintHandle<K extends HandleKind>(
  kind: K,
  id: string,
  sessionId: string,
  owner: string,
  generation: number,
): NativeHandle<K> {
  check_id("handle id", id);
  check_id("session id", sessionId);
  if (typeof owner !== "string" || owner.length === 0 || owner.length > 256) {
    throw new NativeSchemaError("rejected", "invalid-request", "invalid handle owner");
  }
  if (!Number.isSafeInteger(generation) || generation < 0) {
    throw new NativeSchemaError("rejected", "invalid-request", "invalid handle generation");
  }
  return Object.freeze({
    [handleBrand]: kind,
    kind,
    id,
    sessionId,
    owner,
    generation,
  }) as NativeHandle<K>;
}

const HANDLE_WIRE_KEYS = ["kind", "id", "sessionId", "owner", "generation"] as const;

export function decodeHandleWire(value: unknown): NativeHandle {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new NativeSchemaError("rejected", "invalid-request", "handle must be an object");
  }
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    if (!(HANDLE_WIRE_KEYS as readonly string[]).includes(key)) {
      throw new NativeSchemaError("rejected", "invalid-request", `unknown handle field: ${key}`);
    }
  }
  const kind = record["kind"];
  if (typeof kind !== "string" || !(HANDLE_KINDS as readonly string[]).includes(kind)) {
    throw new NativeSchemaError("rejected", "invalid-request", "invalid handle kind");
  }
  const generation = record["generation"];
  if (!Number.isSafeInteger(generation) || (generation as number) < 0) {
    throw new NativeSchemaError("rejected", "invalid-request", "invalid handle generation");
  }
  return mintHandle(
    kind as HandleKind,
    check_id("handle id", record["id"]),
    check_id("session id", record["sessionId"]),
    checkOwner(record["owner"]),
    generation as number,
  );
}

export function encodeHandleWire(handle: NativeHandle): NativeHandleWire {
  return Object.freeze({
    kind: handle.kind,
    id: handle.id,
    sessionId: handle.sessionId,
    owner: handle.owner,
    generation: handle.generation,
  });
}

function checkOwner(value: unknown): string {
  if (typeof value !== "string" || value.length === 0 || value.length > 256) {
    throw new NativeSchemaError("rejected", "invalid-request", "invalid owner grant");
  }
  return value;
}

// Ownership check for handle use. Wrong owner, stale generation, and
// closed-session use are distinct explicit errors; sessions never share.
export function checkHandleUse(
  handle: NativeHandle,
  opts: Readonly<{ owner: string; sessionLive: boolean; liveGeneration: number }>,
): void {
  if (handle.owner !== opts.owner) {
    throw new NativeSchemaError("rejected", "wrong-owner", "handle owner mismatch");
  }
  if (!opts.sessionLive) {
    throw new NativeSchemaError("rejected", "closed-handle", "session is closed");
  }
  if (handle.generation !== opts.liveGeneration) {
    throw new NativeSchemaError("rejected", "stale-handle", "stale handle generation");
  }
}

// ---------------------------------------------------------------------------
// Finite limits: every bound required, positive, and finite
// ---------------------------------------------------------------------------

export type NativeValueLimits = Readonly<{
  max_sessions: number;
  max_handles_per_session: number;
  maxPendingActions: number;
  maxObserveEntries: number;
  maxObserveBytes: number;
  maxGatesPerSession: number;
  maxFaultsPerSession: number;
}>;

export const NATIVE_VALUE_LIMIT_KEYS = [
  "max_sessions",
  "max_handles_per_session",
  "maxPendingActions",
  "maxObserveEntries",
  "maxObserveBytes",
  "maxGatesPerSession",
  "maxFaultsPerSession",
] as const;

export function checkLimits(value: unknown): NativeValueLimits {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new NativeSchemaError("rejected", "invalid-request", "limits must be an object");
  }
  const record = value as Record<string, unknown>;
  const out: Record<string, number> = {};
  for (const key of NATIVE_VALUE_LIMIT_KEYS) {
    const field = record[key];
    if (!Number.isSafeInteger(field) || (field as number) <= 0) {
      throw new NativeSchemaError(
        "rejected",
        "invalid-request",
        `limit ${key} must be a positive finite integer`,
      );
    }
    out[key] = field as number;
  }
  for (const key of Object.keys(record)) {
    if (!(NATIVE_VALUE_LIMIT_KEYS as readonly string[]).includes(key)) {
      throw new NativeSchemaError("rejected", "invalid-request", `unknown limit field: ${key}`);
    }
  }
  return Object.freeze(out) as NativeValueLimits;
}

// ---------------------------------------------------------------------------
// Request envelope with forbidden-surface rejection
// ---------------------------------------------------------------------------

// Argument keys that must never appear: arbitrary module/export/eval surface
// and embedded expected outcomes / oracles. Requests carrying any of these
// reject before any effect.
export const FORBIDDEN_ARG_KEYS = Object.freeze([
  "module",
  "modulePath",
  "export",
  "exportName",
  "source",
  "code",
  "script",
  "eval",
  "evaluate",
  "expected",
  "expect",
  "expectedValue",
  "oracle",
  "answer",
] as const);

const FORBIDDEN_SET: ReadonlySet<string> = new Set(FORBIDDEN_ARG_KEYS as readonly string[]);

const MAX_SCAN_DEPTH = 8;
const MAX_SCAN_KEYS = 256;

export function rejectForbiddenArgs(args: unknown): void {
  let seen = 0;
  const visit = (node: unknown, depth: number): void => {
    if (node === null || typeof node !== "object") return;
    if (depth > MAX_SCAN_DEPTH) {
      throw new NativeSchemaError("rejected", "invalid-request", "arguments nest too deeply");
    }
    if (Array.isArray(node)) {
      for (const item of node) visit(item, depth + 1);
      return;
    }
    for (const key of Object.keys(node)) {
      seen += 1;
      if (seen > MAX_SCAN_KEYS) {
        throw new NativeSchemaError("rejected", "invalid-request", "too many argument keys");
      }
      if (FORBIDDEN_SET.has(key)) {
        throw new NativeSchemaError(
          "rejected",
          "invalid-request",
          `forbidden argument key: ${key}`,
        );
      }
      visit((node as Record<string, unknown>)[key], depth + 1);
    }
  };
  visit(args, 0);
}

export type NativeValueClock = Readonly<{
  clock: "n-monotonic" | "wall-utc";
  ms: number;
}>;

export type NativeValueRequest = Readonly<{
  schema_version: typeof NATIVE_VALUE_SCHEMA_VERSION;
  run_id: string;
  operation_id: string;
  owner_grant: string;
  operation: NativeValueOperationName;
  arguments_digest: string;
  deadline_ms: NativeValueClock;
  args: Readonly<Record<string, unknown>>;
}>;

const REQUEST_KEYS = [
  "schema_version",
  "run_id",
  "operation_id",
  "owner_grant",
  "operation",
  "arguments_digest",
  "deadline_ms",
  "args",
] as const;

const DIGEST_PATTERN = /^sha256:[0-9a-f]{64}$/;

export function checkRequest(value: unknown): NativeValueRequest {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new NativeSchemaError("rejected", "invalid-request", "request must be an object");
  }
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    if (!(REQUEST_KEYS as readonly string[]).includes(key)) {
      throw new NativeSchemaError("rejected", "invalid-request", `unknown request field: ${key}`);
    }
  }
  if (record["schema_version"] !== NATIVE_VALUE_SCHEMA_VERSION) {
    throw new NativeSchemaError("rejected", "invalid-request", "unsupported schema_version");
  }
  const run_id = check_id("run id", record["run_id"]);
  const operation_id = check_id("operation id", record["operation_id"]);
  const owner_grant = checkOwner(record["owner_grant"]);
  const operationRaw = record["operation"];
  if (!isNativeValueOperation(operationRaw)) {
    throw new NativeSchemaError(
      "rejected",
      "unsupported-capability",
      `unknown native-value operation: ${String(operationRaw)}`,
    );
  }
  const digest = record["arguments_digest"];
  if (typeof digest !== "string" || !DIGEST_PATTERN.test(digest)) {
    throw new NativeSchemaError("rejected", "invalid-request", "invalid arguments digest");
  }
  const deadlineRaw = record["deadline_ms"];
  if (deadlineRaw === null || typeof deadlineRaw !== "object" || Array.isArray(deadlineRaw)) {
    throw new NativeSchemaError("rejected", "invalid-request", "invalid deadline");
  }
  const deadline = deadlineRaw as Record<string, unknown>;
  const clock = deadline["clock"];
  const ms = deadline["ms"];
  if (
    (clock !== "n-monotonic" && clock !== "wall-utc") ||
    !Number.isSafeInteger(ms) ||
    (ms as number) < 0
  ) {
    throw new NativeSchemaError("rejected", "invalid-request", "invalid deadline");
  }
  for (const key of Object.keys(deadline)) {
    if (key !== "clock" && key !== "ms") {
      throw new NativeSchemaError("rejected", "invalid-request", `unknown deadline field: ${key}`);
    }
  }
  const args = record["args"];
  if (args === null || typeof args !== "object" || Array.isArray(args)) {
    throw new NativeSchemaError("rejected", "invalid-request", "args must be an object");
  }
  rejectForbiddenArgs(args);
  // Declared args must match the catalogue entry; undeclared keys reject so
  // no unreviewed surface can smuggle module/export/eval-shaped parameters.
  const entry = describeOperation(operationRaw);
  const declared = new Set(entry.args.map((arg) => arg.name));
  for (const key of Object.keys(args)) {
    if (!declared.has(key)) {
      throw new NativeSchemaError(
        "rejected",
        "invalid-request",
        `undeclared argument for ${entry.name}: ${key}`,
      );
    }
  }
  for (const arg of entry.args) {
    if (arg.required && (args as Record<string, unknown>)[arg.name] === undefined) {
      throw new NativeSchemaError(
        "rejected",
        "invalid-request",
        `missing required argument for ${entry.name}: ${arg.name}`,
      );
    }
  }
  return Object.freeze({
    schema_version: NATIVE_VALUE_SCHEMA_VERSION,
    run_id,
    operation_id,
    owner_grant,
    operation: operationRaw,
    arguments_digest: digest,
    deadline_ms: Object.freeze({ clock, ms }),
    args: Object.freeze({ ...(args as Record<string, unknown>) }),
  }) as NativeValueRequest;
}

// ---------------------------------------------------------------------------
// Result envelope: inert facts only
// ---------------------------------------------------------------------------

export type NativeValueResult = Readonly<{
  schema_version: typeof NATIVE_VALUE_SCHEMA_VERSION;
  run_id: string;
  operation_id: string;
  outcome: NativeOutcome;
  kind: MechanicalKind;
  facts?: Readonly<Record<string, unknown>>;
  partial?: Readonly<Record<string, unknown>>;
}>;

// Reject any reply payload that could carry a live hostile value: functions,
// symbols, bigints-as-values, Promises, and any object with a `then`
// property (which await would assimilate). Exact integers cross as lexeme
// strings; floats cross as tagged IEEE-754 bits; values cross as handles.
export function checkFactsInert(facts: unknown, what: string): void {
  let seen = 0;
  const visit = (node: unknown, depth: number): void => {
    if (node === null) return;
    const kind = typeof node;
    if (kind === "function" || kind === "symbol" || kind === "bigint") {
      throw new NativeSchemaError("failed", "native-io", `${what} carries a non-inert ${kind}`);
    }
    if (typeof node !== "object") return;
    if (depth > MAX_SCAN_DEPTH) {
      throw new NativeSchemaError("failed", "native-io", `${what} nests too deeply`);
    }
    if (node instanceof Promise) {
      throw new NativeSchemaError("failed", "native-io", `${what} carries a raw Promise`);
    }
    if ("then" in (node as Record<string, unknown>)) {
      throw new NativeSchemaError("failed", "native-io", `${what} carries a thenable-shaped value`);
    }
    if (Array.isArray(node)) {
      for (const item of node) visit(item, depth + 1);
      return;
    }
    const keys = Object.keys(node);
    if (keys.length > 32) {
      throw new NativeSchemaError("failed", "native-io", `${what} exceeds the fact-map bound`);
    }
    for (const key of keys) {
      seen += 1;
      if (seen > MAX_SCAN_KEYS) {
        throw new NativeSchemaError("failed", "native-io", `${what} carries too many keys`);
      }
      if (FORBIDDEN_SET.has(key)) {
        throw new NativeSchemaError("failed", "native-io", `${what} carries forbidden key: ${key}`);
      }
      visit((node as Record<string, unknown>)[key], depth + 1);
    }
  };
  visit(facts, 0);
}

export function checkResult(value: unknown): NativeValueResult {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new NativeSchemaError("failed", "transport-failure", "result must be an object");
  }
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    if (
      key !== "schema_version" &&
      key !== "run_id" &&
      key !== "operation_id" &&
      key !== "outcome" &&
      key !== "kind" &&
      key !== "facts" &&
      key !== "partial"
    ) {
      throw new NativeSchemaError("failed", "transport-failure", `unknown result field: ${key}`);
    }
  }
  if (record["schema_version"] !== NATIVE_VALUE_SCHEMA_VERSION) {
    throw new NativeSchemaError("failed", "transport-failure", "unsupported result schema_version");
  }
  const outcome = record["outcome"];
  if (typeof outcome !== "string" || !(NATIVE_OUTCOMES as readonly string[]).includes(outcome)) {
    throw new NativeSchemaError("failed", "transport-failure", "invalid result outcome");
  }
  const kind = record["kind"];
  if (typeof kind !== "string" || !(MECHANICAL_KINDS as readonly string[]).includes(kind)) {
    throw new NativeSchemaError("failed", "transport-failure", "invalid result kind");
  }
  if (outcome === "completed" && kind !== "ok") {
    throw new NativeSchemaError("failed", "transport-failure", "completed result requires kind ok");
  }
  if (outcome !== "completed" && kind === "ok") {
    throw new NativeSchemaError(
      "failed",
      "transport-failure",
      "non-completed result requires a non-ok kind",
    );
  }
  if (record["facts"] !== undefined) checkFactsInert(record["facts"], "result facts");
  if (record["partial"] !== undefined) checkFactsInert(record["partial"], "result partial");
  return value as NativeValueResult;
}
