// K10: separate pending browser input from settlement.
//
// An input service over in-memory doubles only: no browser, no process,
// no flags, no environment, no sampling, no I/O. The service binds owned
// contexts, begins pending inputs with opaque minted tokens, captures
// bounded DOM snapshots while an input is pending, settles each input
// exactly once with a declared outcome, serves a lock-free route board
// that stays usable while any click is pending, and records independent
// application witnesses. Click settlement, route delivery, and the
// application witness are three separate facts with disjoint shapes: a
// route delivery never settles an input, a witness never settles an
// input, and only an explicit abort-fulfills link lets a route abort
// fulfill a pending input.
//
// Independence scope from the launch/driver service under test (normative):
//   SHARES: nothing executable. The service never imports, calls, or reads
//   driver code, driver-decoded values, receipts, flags, environment
//   variables, or process state. The only strings that may coincide are
//   the context/action/route/witness names under test (supplied by the
//   test, not by the driver) and the vocabulary this file uses for facts.
//   Facts carry digests only: raw DOM bytes never cross.
//   PROVES: context admission, action identity, pending/settled separation,
//   bounded DOM capture, lock-free route delivery and abort, the explicit
//   abort-fulfills path, independent application witness, settlement
//   verdicts that reject conflated claims, and error provenance (every
//   failure names its layer). Driver claims are compared AGAINST these
//   facts; the service never derives a fact FROM a driver claim.
//   Consequently a route delivery, an application witness, a DOM capture,
//   or a pending attestation can never prove settlement: only a verified
//   settlement fact proves it.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, browsers, processes, or live runtimes. Local
// controls only. Live host-dependent controls wait for the corresponding
// qualified profile and Q task.

import { createHash, randomBytes } from "node:crypto";

export const BROWSER_INPUT_SCHEMA_VERSION = "1" as const;

// ---------------------------------------------------------------------------
// Independence scope (normative)
// ---------------------------------------------------------------------------

export const BROWSER_INPUT_INDEPENDENCE_SCOPE: string =
  "independent: shares no driver code, decoded values, receipts, flags, " +
  "environment, or process state; proves context admission, action identity, " +
  "pending/settled separation, bounded DOM capture, lock-free route delivery " +
  "and abort, the explicit abort-fulfills path, independent application " +
  "witness, settlement verdicts, and layered error provenance from its own " +
  "seeded doubles alone";

// ---------------------------------------------------------------------------
// Layers, codes, and the layered error
// ---------------------------------------------------------------------------

export const BROWSER_INPUT_LAYERS = ["input", "route", "witness"] as const;
export type BrowserInputLayer = (typeof BROWSER_INPUT_LAYERS)[number];

// Closed failure-code vocabulary. Every code maps to exactly one layer:
//   input: context admission, ownership, capacity, lifecycle, action
//     identity, tokens, pending/settled separation, DOM capture, and
//     settlement outcomes.
//   route: the lock-free route board: hold, tokens, delivery, abort, and
//     the explicit abort-fulfills link.
//   witness: application-witness admission and outcomes, and the settlement
//     verdict that rejects conflated claims.
export const BROWSER_INPUT_CODES = [
  "unknown-context",
  "no-context",
  "wrong-owner",
  "capacity-exhausted",
  "context-closed",
  "forged-token",
  "unknown-action",
  "action-closed",
  "action-mismatch",
  "input-pending",
  "unknown-dom",
  "unknown-outcome",
  "forged-route-token",
  "unknown-route",
  "route-closed",
  "route-pending",
  "unknown-route-outcome",
  "unknown-witness",
  "unknown-witness-outcome",
  "forbidden-proof",
] as const;
export type BrowserInputCode = (typeof BROWSER_INPUT_CODES)[number];

const CODE_LAYER: Readonly<Record<BrowserInputCode, BrowserInputLayer>> = {
  "unknown-context": "input",
  "no-context": "input",
  "wrong-owner": "input",
  "capacity-exhausted": "input",
  "context-closed": "input",
  "forged-token": "input",
  "unknown-action": "input",
  "action-closed": "input",
  "action-mismatch": "input",
  "input-pending": "input",
  "unknown-dom": "input",
  "unknown-outcome": "input",
  "forged-route-token": "route",
  "unknown-route": "route",
  "route-closed": "route",
  "route-pending": "route",
  "unknown-route-outcome": "route",
  "unknown-witness": "witness",
  "unknown-witness-outcome": "witness",
  "forbidden-proof": "witness",
};

export function layerOfInputCode(code: BrowserInputCode): BrowserInputLayer {
  return CODE_LAYER[code];
}

export class BrowserInputError extends Error {
  readonly layer: BrowserInputLayer;
  readonly code: BrowserInputCode;
  constructor(code: BrowserInputCode, message: string) {
    super(`[${CODE_LAYER[code]}:${code}] ${message}`);
    this.name = "BrowserInputError";
    this.layer = CODE_LAYER[code];
    this.code = code;
  }
}

function fail(code: BrowserInputCode, message: string): never {
  throw new BrowserInputError(code, message);
}

// ---------------------------------------------------------------------------
// Bounds and names
// ---------------------------------------------------------------------------

export type BrowserInputLimits = Readonly<{
  maxContexts: number;
  maxInputs: number;
  maxRoutes: number;
  maxCapturesPerInput: number;
  maxNodesPerCapture: number;
  maxWitnesses: number;
}>;

const LIMIT_KEYS = [
  "maxContexts",
  "maxInputs",
  "maxRoutes",
  "maxCapturesPerInput",
  "maxNodesPerCapture",
  "maxWitnesses",
] as const;

export function checkInputLimits(value: unknown): BrowserInputLimits {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new BrowserInputError("unknown-context", "limits must be an object");
  }
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    if (!(LIMIT_KEYS as readonly string[]).includes(key)) {
      throw new BrowserInputError("unknown-context", `unknown limit field: ${key}`);
    }
  }
  const out: Record<string, number> = {};
  for (const key of LIMIT_KEYS) {
    const entry = record[key];
    if (!Number.isSafeInteger(entry) || (entry as number) <= 0) {
      throw new BrowserInputError("unknown-context", `${key} must be a positive integer`);
    }
    out[key] = entry as number;
  }
  return Object.freeze(out) as BrowserInputLimits;
}

// Finite input-action vocabulary: every input performs exactly one.
export const INPUT_ACTIONS = ["click", "type", "press"] as const;
export type InputAction = (typeof INPUT_ACTIONS)[number];

// Finite settlement outcomes. actuated and rejected settle directly through
// input_settle; aborted-by-route is derived and arises only through the
// explicit abort-fulfills path on route abort, never through direct settle.
export const SETTLEMENT_OUTCOMES = ["actuated", "rejected", "aborted-by-route"] as const;
export type SettlementOutcome = (typeof SETTLEMENT_OUTCOMES)[number];

export const DIRECT_OUTCOMES = ["actuated", "rejected"] as const;

// Finite route-abort reasons. Every abort states its reason explicitly.
export const ROUTE_ABORT_REASONS = ["aborted", "timeout", "closed"] as const;
export type RouteAbortReason = (typeof ROUTE_ABORT_REASONS)[number];

// Finite route verdicts: a held route delivers or aborts, exactly once.
export const ROUTE_VERDICTS = ["delivered", "aborted"] as const;
export type RouteVerdict = (typeof ROUTE_VERDICTS)[number];

// Finite application-witness outcomes. The witness records what the
// application showed; it never settles an input.
export const WITNESS_OUTCOMES = ["effect-seen", "effect-absent"] as const;
export type WitnessOutcome = (typeof WITNESS_OUTCOMES)[number];

const MAX_NAME_LEN = 128;
const NODE_DIGEST_RE = /^sha256:[0-9a-f]{64}$/;

function checkName(value: unknown, what: string, code: BrowserInputCode): string {
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

function checkActionId(value: unknown): string {
  return checkName(value, "action", "unknown-action");
}

function checkRouteId(value: unknown): string {
  return checkName(value, "route", "unknown-route");
}

function checkWitnessId(value: unknown): string {
  return checkName(value, "witness", "unknown-witness");
}

function checkAction(value: unknown): InputAction {
  if (typeof value !== "string" || !(INPUT_ACTIONS as readonly string[]).includes(value)) {
    fail("unknown-action", `not a declared input action: ${String(value)}`);
  }
  return value as InputAction;
}

function checkDirectOutcome(value: unknown): SettlementOutcome {
  if (typeof value === "string" && (DIRECT_OUTCOMES as readonly string[]).includes(value)) {
    return value as SettlementOutcome;
  }
  if (value === "aborted-by-route") {
    fail("unknown-outcome", "aborted-by-route is derived only through an explicit route abort");
  }
  fail("unknown-outcome", `not a declared direct settlement outcome: ${String(value)}`);
}

function checkAbortReason(value: unknown): RouteAbortReason {
  if (typeof value !== "string" || !(ROUTE_ABORT_REASONS as readonly string[]).includes(value)) {
    fail("unknown-route-outcome", `not a declared route abort reason: ${String(value)}`);
  }
  return value as RouteAbortReason;
}

function checkWitnessOutcome(value: unknown): WitnessOutcome {
  if (typeof value !== "string" || !(WITNESS_OUTCOMES as readonly string[]).includes(value)) {
    fail("unknown-witness-outcome", `not a declared witness outcome: ${String(value)}`);
  }
  return value as WitnessOutcome;
}

function checkDescriptor(
  value: unknown,
  what: string,
  code: BrowserInputCode,
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
  code: BrowserInputCode,
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

export type InputAttestation = Readonly<{
  actionId: string;
  owner: string;
  context: string;
  action: InputAction;
  target: string;
  contextDigest: string;
  handleDigest: string;
  domCaptures: number;
  pending: true;
}>;

export type DomCaptureFacts = Readonly<{
  actionId: string;
  captureSeq: number;
  nodeCount: number;
  truncated: boolean;
  digest: string;
}>;

// Fact one: the input settlement. Only this shape proves settlement. The
// fulfilledByRoute link is present exactly when the abort-fulfills path
// settled the input, and absent otherwise.
export type InputSettlementFacts = Readonly<{
  actionId: string;
  owner: string;
  context: string;
  action: InputAction;
  target: string;
  outcome: SettlementOutcome;
  fulfilledByRoute: string | null;
  domCaptures: number;
  digest: string;
}>;

export type InputStatusFacts = Readonly<{
  actionId: string;
  owner: string;
  context: string;
  action: InputAction;
  target: string;
  pending: boolean;
  outcome: SettlementOutcome | null;
  fulfilledByRoute: string | null;
  domCaptures: number;
  digest: string;
}>;

export type RouteAttestation = Readonly<{
  routeId: string;
  owner: string;
  context: string;
  contextDigest: string;
  handleDigest: string;
  held: true;
}>;

// Fact two: the route delivery. A delivered or aborted route proves route
// handling only; it never proves input settlement, even when the route
// carried an abort-fulfills link (the link settles the input separately).
export type RouteDeliveryFacts = Readonly<{
  routeId: string;
  owner: string;
  context: string;
  verdict: RouteVerdict;
  abortReason: RouteAbortReason | null;
  fulfillsActionId: string | null;
  digest: string;
}>;

export type RouteStatusFacts = Readonly<{
  routeId: string;
  owner: string;
  context: string;
  held: boolean;
  verdict: RouteVerdict | null;
  abortReason: RouteAbortReason | null;
  fulfillsActionId: string | null;
  digest: string;
}>;

// Fact three: the application witness. The witness records what the
// application showed for a named action; it never settles an input and
// never reads the input table.
export type ApplicationWitnessFacts = Readonly<{
  witnessId: string;
  owner: string;
  context: string;
  actionId: string | null;
  outcome: WitnessOutcome;
  digest: string;
}>;

export type InputDescriptor = Readonly<{
  action: InputAction;
  target: string;
}>;

export type DomNodeDescriptor = Readonly<{
  name: string;
  nodeDigest: string;
}>;

export type DomCaptureDescriptor = Readonly<{
  nodes: readonly DomNodeDescriptor[];
  truncated: boolean;
}>;

export type RouteAbortDescriptor = Readonly<{
  reason: RouteAbortReason;
  fulfillsActionId: string | null;
}>;

export type WitnessDescriptor = Readonly<{
  actionId: string | null;
  outcome: WitnessOutcome;
}>;

// A settlement proof claim. Only input-settlement is admissible; every
// other kind rejects with forbidden-proof before any verdict is read.
export const FORBIDDEN_SETTLEMENT_KINDS = [
  "route-delivery",
  "app-witness",
  "dom-capture",
  "input-attestation",
] as const;
export type ForbiddenSettlementKind = (typeof FORBIDDEN_SETTLEMENT_KINDS)[number];

export type SettlementProofClaim =
  | Readonly<{ kind: "input-settlement"; settlement: InputSettlementFacts }>
  | Readonly<{ kind: ForbiddenSettlementKind; detail: string }>;

export type SettlementVerdict = Readonly<{
  actionId: string;
  outcome: SettlementOutcome;
  fulfilledByRoute: string | null;
  settled: boolean;
  digest: string;
}>;

type InputRecord = {
  actionId: string;
  owner: string;
  context: string;
  action: InputAction;
  target: string;
  token: string;
  pending: boolean;
  outcome: SettlementOutcome | null;
  fulfilledByRoute: string | null;
  captures: DomCaptureFacts[];
};

type RouteRecord = {
  routeId: string;
  owner: string;
  context: string;
  token: string;
  held: boolean;
  verdict: RouteVerdict | null;
  abortReason: RouteAbortReason | null;
  fulfillsActionId: string | null;
};

type WitnessRecord = {
  witnessId: string;
  owner: string;
  context: string;
  actionId: string | null;
  outcome: WitnessOutcome;
};

type ContextState = "open" | "closed";

type ContextRecord = {
  context: string;
  owner: string;
  token: string;
  state: ContextState;
  inputs: Map<string, InputRecord>;
  routes: Map<string, RouteRecord>;
  witnesses: Map<string, WitnessRecord>;
};

function mintToken(prefix: string): string {
  return `${prefix}-${randomBytes(16).toString("hex")}`;
}

function digestText(text: string): string {
  return `sha256:${createHash("sha256").update(text, "utf8").digest("hex")}`;
}

// The settlement digest is recomputed from carried fields, so
// assertInputSettled re-verifies a presented settlement instead of trusting
// its digest string.
export function digestSettlement(
  actionId: string,
  owner: string,
  context: string,
  action: string,
  target: string,
  outcome: string,
  fulfilledByRoute: string | null,
  domCaptures: number,
): string {
  const hash = createHash("sha256");
  hash.update(actionId, "utf8");
  hash.update("\0", "utf8");
  hash.update(owner, "utf8");
  hash.update("\0", "utf8");
  hash.update(context, "utf8");
  hash.update("\0", "utf8");
  hash.update(action, "utf8");
  hash.update("\0", "utf8");
  hash.update(target, "utf8");
  hash.update("\0", "utf8");
  hash.update(outcome, "utf8");
  hash.update("\0", "utf8");
  hash.update(fulfilledByRoute ?? "", "utf8");
  hash.update("\0", "utf8");
  hash.update(String(domCaptures), "utf8");
  return `sha256:${hash.digest("hex")}`;
}

export function digestRouteDelivery(
  routeId: string,
  owner: string,
  context: string,
  verdict: string,
  abortReason: string | null,
  fulfillsActionId: string | null,
): string {
  return digestText(
    `route:${routeId}:${owner}:${context}:${verdict}:${abortReason ?? ""}:${fulfillsActionId ?? ""}`,
  );
}

export function digestWitness(
  witnessId: string,
  owner: string,
  context: string,
  actionId: string | null,
  outcome: string,
): string {
  return digestText(`witness:${witnessId}:${owner}:${context}:${actionId ?? ""}:${outcome}`);
}

// ---------------------------------------------------------------------------
// Service: owned contexts, pending input, lock-free routes, witness
// ---------------------------------------------------------------------------

export type ContextGrant = Readonly<{
  owner: string;
  context: string;
}>;

export class BrowserInputService {
  private readonly limits: BrowserInputLimits;
  private readonly grants = new Map<string, ContextGrant>();
  private readonly contexts = new Map<string, ContextRecord>();

  constructor(grants: readonly ContextGrant[], limits: BrowserInputLimits) {
    if (grants.length === 0) {
      throw new BrowserInputError("unknown-context", "declare at least one owned context");
    }
    if (grants.length > limits.maxContexts) {
      throw new BrowserInputError("capacity-exhausted", "declared contexts exceed the cap");
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

  inputCount(owner: string, context: string): number {
    return this.requireContext(owner, context).inputs.size;
  }

  routeCount(owner: string, context: string): number {
    return this.requireContext(owner, context).routes.size;
  }

  witnessCount(owner: string, context: string): number {
    return this.requireContext(owner, context).witnesses.size;
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
      inputs: new Map(),
      routes: new Map(),
      witnesses: new Map(),
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

  // Admission-first: the begin/hold/witness gate reports only whether a LIVE
  // binding exists. Never bound, closed, or foreign all refuse with
  // no-context: a missing context blocks input, route, and witness work,
  // whatever the reason.
  private requireLiveBinding(owner: string, context: string): ContextRecord {
    checkOwner(owner);
    checkContextName(context);
    const record = this.contexts.get(context);
    if (record === undefined || record.owner !== owner || record.state !== "open") {
      fail("no-context", `no live context binding: ${owner}/${context}`);
    }
    return record as ContextRecord;
  }

  // Close one context. Pending inputs block the close, and so do held
  // routes: nothing unsettled closes. Settled inputs, delivered or aborted
  // routes, and witnesses stay out of the way: witnesses never block.
  closeContext(owner: string, context: string): void {
    const record = this.requireContext(owner, context);
    for (const input of record.inputs.values()) {
      if (input.pending) {
        fail("input-pending", `input still pending: ${input.actionId}`);
      }
    }
    for (const route of record.routes.values()) {
      if (route.held) {
        fail("route-pending", `route still held: ${route.routeId}`);
      }
    }
    record.state = "closed";
  }

  private checkInputDescriptor(value: unknown): InputDescriptor {
    const record = checkDescriptor(value, "input", "unknown-action");
    checkNoExtraKeys(record, ["action", "target"], "input", "unknown-action");
    const action = checkAction(record["action"]);
    const target = checkName(record["target"], "input target", "unknown-action");
    return { action, target };
  }

  private checkDomDescriptor(value: unknown): DomCaptureDescriptor {
    const record = checkDescriptor(value, "dom capture", "unknown-dom");
    checkNoExtraKeys(record, ["nodes", "truncated"], "dom capture", "unknown-dom");
    const nodes = record["nodes"];
    if (!Array.isArray(nodes) || nodes.length === 0) {
      fail("unknown-dom", "dom capture carries a non-empty node list, never raw bytes");
    }
    if (nodes.length > this.limits.maxNodesPerCapture) {
      fail("capacity-exhausted", `dom capture exceeds ${this.limits.maxNodesPerCapture} nodes`);
    }
    const checked: DomNodeDescriptor[] = [];
    for (const entry of nodes) {
      if (entry === null || typeof entry !== "object" || Array.isArray(entry)) {
        fail("unknown-dom", "dom capture nodes carry name and node digest only");
      }
      const node = entry as Record<string, unknown>;
      checkNoExtraKeys(node, ["name", "nodeDigest"], "dom node", "unknown-dom");
      const name = checkName(node["name"], "dom node", "unknown-dom");
      const nodeDigest = node["nodeDigest"];
      if (typeof nodeDigest !== "string" || !NODE_DIGEST_RE.test(nodeDigest)) {
        fail("unknown-dom", "dom capture carries node digests, never raw bytes");
      }
      checked.push({ name, nodeDigest });
    }
    const truncated = record["truncated"];
    if (typeof truncated !== "boolean") {
      fail("unknown-dom", "dom capture states truncated explicitly");
    }
    return { nodes: checked, truncated };
  }

  private checkAbortDescriptor(value: unknown): RouteAbortDescriptor {
    const record = checkDescriptor(value, "route abort", "unknown-route-outcome");
    checkNoExtraKeys(
      record,
      ["reason", "fulfillsActionId"],
      "route abort",
      "unknown-route-outcome",
    );
    const reason = checkAbortReason(record["reason"]);
    const fulfills = record["fulfillsActionId"];
    if (fulfills === undefined || fulfills === null) {
      return { reason, fulfillsActionId: null };
    }
    return { reason, fulfillsActionId: checkName(fulfills, "fulfilled action", "unknown-action") };
  }

  private checkWitnessDescriptor(value: unknown): WitnessDescriptor {
    const record = checkDescriptor(value, "witness", "unknown-witness-outcome");
    checkNoExtraKeys(record, ["actionId", "outcome"], "witness", "unknown-witness-outcome");
    const actionId = record["actionId"];
    const outcome = checkWitnessOutcome(record["outcome"]);
    if (actionId === undefined || actionId === null) {
      return { actionId: null, outcome };
    }
    // The witness names an action but never reads the input table: any
    // well-formed name records, and recording never settles.
    return { actionId: checkName(actionId, "witnessed action", "unknown-witness"), outcome };
  }

  // Begin one pending input (input_begin). The action token is minted here
  // and verified by table lookup on every later mutation, so an invented
  // token is never authority. Action ids are single-use per context: a
  // settled id never re-opens. Re-beginning a pending id with the identical
  // descriptor rejoins; a different descriptor refuses with
  // action-mismatch rather than retargeting the pending input.
  beginInput(
    owner: string,
    context: string,
    actionId: string,
    descriptor: unknown,
  ): InputAttestation {
    const input = this.checkInputDescriptor(descriptor);
    const record = this.requireLiveBinding(owner, context);
    const id = checkActionId(actionId);
    let total = 0;
    for (const contextRecord of this.contexts.values()) {
      total += contextRecord.inputs.size;
    }
    if (total >= this.limits.maxInputs) {
      fail("capacity-exhausted", "input table full");
    }
    const prior = record.inputs.get(id);
    if (prior !== undefined) {
      if (!prior.pending) {
        fail("action-closed", `action id is single-use and already settled: ${id}`);
      }
      if (prior.action !== input.action || prior.target !== input.target) {
        fail("action-mismatch", `action id is already pending with another descriptor: ${id}`);
      }
      return this.attestationOf(record, prior);
    }
    const entry: InputRecord = {
      actionId: id,
      owner,
      context,
      action: input.action,
      target: input.target,
      token: mintToken("act"),
      pending: true,
      outcome: null,
      fulfilledByRoute: null,
      captures: [],
    };
    record.inputs.set(id, entry);
    return this.attestationOf(record, entry);
  }

  private attestationOf(record: ContextRecord, input: InputRecord): InputAttestation {
    return Object.freeze({
      actionId: input.actionId,
      owner: input.owner,
      context: record.context,
      action: input.action,
      target: input.target,
      contextDigest: digestText(record.token),
      handleDigest: digestText(input.token),
      domCaptures: input.captures.length,
      pending: true as const,
    });
  }

  attestation(owner: string, context: string, actionId: string): InputAttestation {
    const record = this.requireContext(owner, context);
    const input = this.requireInput(record, actionId);
    if (!input.pending) {
      fail("action-closed", `action is settled, not pending: ${actionId}`);
    }
    return this.attestationOf(record, input);
  }

  private requireInput(record: ContextRecord, actionId: string): InputRecord {
    const input = record.inputs.get(actionId);
    if (input === undefined) {
      fail("unknown-action", `no begun input: ${actionId}`);
    }
    return input as InputRecord;
  }

  // Verify the action token by table lookup. Unknown actions and invented
  // tokens are never authority; settled actions report action-closed.
  private requireLiveInput(
    owner: string,
    context: string,
    actionId: string,
    token: string,
  ): { record: ContextRecord; input: InputRecord } {
    const record = this.requireContext(owner, context);
    const input = this.requireInput(record, actionId);
    if (!input.pending) {
      fail("action-closed", `action is settled: ${actionId}`);
    }
    if (token === "" || token !== input.token) {
      fail("forged-token", "action token is not the attested token");
    }
    return { record, input };
  }

  // Test-only accessor: the raw token crosses exactly here so bounded
  // controls can present it on later calls. Facts carry digests only.
  inputTokenForTest(owner: string, context: string, actionId: string): string {
    const record = this.requireContext(owner, context);
    return this.requireInput(record, actionId).token;
  }

  // Capture one bounded DOM snapshot for a pending input. Captures carry
  // node digests only, state truncation explicitly, and land only while the
  // input is pending: settled actions refuse new captures.
  captureDom(
    owner: string,
    context: string,
    actionId: string,
    token: string,
    descriptor: unknown,
  ): DomCaptureFacts {
    const capture = this.checkDomDescriptor(descriptor);
    const { input } = this.requireLiveInput(owner, context, actionId, token);
    if (input.captures.length >= this.limits.maxCapturesPerInput) {
      fail("capacity-exhausted", "dom capture table full for this input");
    }
    const seq = input.captures.length;
    const hash = createHash("sha256");
    hash.update(`${actionId}:${seq}:${capture.truncated ? 1 : 0}`, "utf8");
    for (const node of capture.nodes) {
      hash.update("\0", "utf8");
      hash.update(node.name, "utf8");
      hash.update(node.nodeDigest, "utf8");
    }
    const facts: DomCaptureFacts = Object.freeze({
      actionId: input.actionId,
      captureSeq: seq,
      nodeCount: capture.nodes.length,
      truncated: capture.truncated,
      digest: `sha256:${hash.digest("hex")}`,
    });
    input.captures.push(facts);
    return facts;
  }

  domCaptures(owner: string, context: string, actionId: string): readonly DomCaptureFacts[] {
    const record = this.requireContext(owner, context);
    return Object.freeze([...this.requireInput(record, actionId).captures]);
  }

  private statusDigestOf(input: InputRecord): string {
    return digestText(
      `status:${input.actionId}:${input.pending ? "pending" : "settled"}:${input.outcome ?? ""}:${input.fulfilledByRoute ?? ""}:${input.captures.length}`,
    );
  }

  // Input status facts stay readable while pending and after settlement:
  // the status is evidence, and pending inputs never hide it. Reads take
  // no token.
  inputFacts(owner: string, context: string, actionId: string): InputStatusFacts {
    const record = this.requireContext(owner, context);
    const input = this.requireInput(record, actionId);
    return Object.freeze({
      actionId: input.actionId,
      owner: input.owner,
      context: record.context,
      action: input.action,
      target: input.target,
      pending: input.pending,
      outcome: input.outcome,
      fulfilledByRoute: input.fulfilledByRoute,
      domCaptures: input.captures.length,
      digest: this.statusDigestOf(input),
    });
  }

  // Settle one pending input (input_settle) with a direct outcome. Only
  // actuated and rejected settle directly: aborted-by-route is derived and
  // arises only through the explicit abort-fulfills path. Settlement is
  // terminal: the id never re-opens and never settles twice.
  settleInput(
    owner: string,
    context: string,
    actionId: string,
    token: string,
    outcome: unknown,
  ): InputSettlementFacts {
    const direct = checkDirectOutcome(outcome);
    const { record, input } = this.requireLiveInput(owner, context, actionId, token);
    input.pending = false;
    input.outcome = direct;
    input.fulfilledByRoute = null;
    return this.settlementOf(record, input);
  }

  private settlementOf(record: ContextRecord, input: InputRecord): InputSettlementFacts {
    return Object.freeze({
      actionId: input.actionId,
      owner: input.owner,
      context: record.context,
      action: input.action,
      target: input.target,
      outcome: input.outcome as SettlementOutcome,
      fulfilledByRoute: input.fulfilledByRoute,
      domCaptures: input.captures.length,
      digest: digestSettlement(
        input.actionId,
        input.owner,
        record.context,
        input.action,
        input.target,
        input.outcome as string,
        input.fulfilledByRoute,
        input.captures.length,
      ),
    });
  }

  // Settlement facts exist only for settled inputs. Pending inputs refuse
  // with input-pending: pending and settled are never conflated.
  settlementFacts(owner: string, context: string, actionId: string): InputSettlementFacts {
    const record = this.requireContext(owner, context);
    const input = this.requireInput(record, actionId);
    if (input.pending) {
      fail("input-pending", `input is still pending: ${actionId}`);
    }
    return this.settlementOf(record, input);
  }

  // -------------------------------------------------------------------------
  // Lock-free route board: hold, deliver, abort while inputs stay pending.
  //
  // Route calls take no input token and consult no input state except
  // through the explicit fulfillsActionId link on abort. A pending click
  // never blocks a route, and a route never settles an input silently.
  // -------------------------------------------------------------------------

  // Hold one route. Route tokens live in their own namespace: minted here,
  // verified by lookup on deliver and abort. Route ids are single-use per
  // context: a delivered or aborted id never re-opens, while re-holding a
  // held id rejoins.
  holdRoute(owner: string, context: string, routeId: string): RouteAttestation {
    const record = this.requireLiveBinding(owner, context);
    const id = checkRouteId(routeId);
    let total = 0;
    for (const contextRecord of this.contexts.values()) {
      total += contextRecord.routes.size;
    }
    if (total >= this.limits.maxRoutes) {
      fail("capacity-exhausted", "route table full");
    }
    const prior = record.routes.get(id);
    if (prior !== undefined) {
      if (!prior.held) {
        fail("route-closed", `route id is single-use and already handled: ${id}`);
      }
      return this.routeAttestationOf(record, prior);
    }
    const route: RouteRecord = {
      routeId: id,
      owner,
      context,
      token: mintToken("rte"),
      held: true,
      verdict: null,
      abortReason: null,
      fulfillsActionId: null,
    };
    record.routes.set(id, route);
    return this.routeAttestationOf(record, route);
  }

  private routeAttestationOf(record: ContextRecord, route: RouteRecord): RouteAttestation {
    return Object.freeze({
      routeId: route.routeId,
      owner: route.owner,
      context: record.context,
      contextDigest: digestText(record.token),
      handleDigest: digestText(route.token),
      held: true as const,
    });
  }

  private requireRoute(record: ContextRecord, routeId: string): RouteRecord {
    const route = record.routes.get(routeId);
    if (route === undefined) {
      fail("unknown-route", `no held route: ${routeId}`);
    }
    return route as RouteRecord;
  }

  private requireHeldRoute(
    owner: string,
    context: string,
    routeId: string,
    token: string,
  ): { record: ContextRecord; route: RouteRecord } {
    const record = this.requireContext(owner, context);
    const route = this.requireRoute(record, routeId);
    if (!route.held) {
      fail("route-closed", `route is already handled: ${routeId}`);
    }
    if (token === "" || token !== route.token) {
      fail("forged-route-token", "route token is not the attested token");
    }
    return { record, route };
  }

  // Test-only accessor: the raw route token crosses exactly here so bounded
  // controls can present it on later calls. Facts carry digests only.
  routeTokenForTest(owner: string, context: string, routeId: string): string {
    const record = this.requireContext(owner, context);
    return this.requireRoute(record, routeId).token;
  }

  // Deliver one held route. Delivery proves route handling only: it never
  // touches any input, pending or settled.
  deliverRoute(owner: string, context: string, routeId: string, token: string): RouteDeliveryFacts {
    const { record, route } = this.requireHeldRoute(owner, context, routeId, token);
    route.held = false;
    route.verdict = "delivered";
    route.abortReason = null;
    route.fulfillsActionId = null;
    return this.deliveryOf(record, route);
  }

  // Abort one held route, optionally fulfilling one named pending input
  // through the explicit fulfillsActionId link. The link is explicit or it
  // does not exist: an abort without fulfillsActionId leaves every input
  // exactly as pending as before, and the fulfilled input settles with
  // outcome aborted-by-route carrying the fulfilling route id.
  abortRoute(
    owner: string,
    context: string,
    routeId: string,
    token: string,
    descriptor: unknown,
  ): RouteDeliveryFacts {
    const abort = this.checkAbortDescriptor(descriptor);
    const { record, route } = this.requireHeldRoute(owner, context, routeId, token);
    if (abort.fulfillsActionId !== null) {
      const input = this.requireInput(record, abort.fulfillsActionId);
      if (!input.pending) {
        fail("action-closed", `fulfilled action is already settled: ${abort.fulfillsActionId}`);
      }
      input.pending = false;
      input.outcome = "aborted-by-route";
      input.fulfilledByRoute = route.routeId;
    }
    route.held = false;
    route.verdict = "aborted";
    route.abortReason = abort.reason;
    route.fulfillsActionId = abort.fulfillsActionId;
    return this.deliveryOf(record, route);
  }

  private deliveryOf(record: ContextRecord, route: RouteRecord): RouteDeliveryFacts {
    return Object.freeze({
      routeId: route.routeId,
      owner: route.owner,
      context: record.context,
      verdict: route.verdict as RouteVerdict,
      abortReason: route.abortReason,
      fulfillsActionId: route.fulfillsActionId,
      digest: digestRouteDelivery(
        route.routeId,
        route.owner,
        record.context,
        route.verdict as string,
        route.abortReason,
        route.fulfillsActionId,
      ),
    });
  }

  // Route delivery facts exist only for handled routes. Held routes refuse
  // with route-pending: held and handled are never conflated.
  deliveryFacts(owner: string, context: string, routeId: string): RouteDeliveryFacts {
    const record = this.requireContext(owner, context);
    const route = this.requireRoute(record, routeId);
    if (route.held) {
      fail("route-pending", `route is still held: ${routeId}`);
    }
    return this.deliveryOf(record, route);
  }

  // Route status facts stay readable while held and after handling. Reads
  // take no token.
  routeFacts(owner: string, context: string, routeId: string): RouteStatusFacts {
    const record = this.requireContext(owner, context);
    const route = this.requireRoute(record, routeId);
    return Object.freeze({
      routeId: route.routeId,
      owner: route.owner,
      context: record.context,
      held: route.held,
      verdict: route.verdict,
      abortReason: route.abortReason,
      fulfillsActionId: route.fulfillsActionId,
      digest: digestText(`route-status:${route.routeId}:${route.held ? "held" : "handled"}`),
    });
  }

  // -------------------------------------------------------------------------
  // Application witness: independent, never settling.
  // -------------------------------------------------------------------------

  // Record one application witness. Witness ids are single-use per context.
  // Recording never reads the input table and never settles: a witness for
  // a pending action leaves it pending, and a witness for an unknown name
  // still records.
  witnessApplication(
    owner: string,
    context: string,
    witnessId: string,
    descriptor: unknown,
  ): ApplicationWitnessFacts {
    const witness = this.checkWitnessDescriptor(descriptor);
    const record = this.requireLiveBinding(owner, context);
    const id = checkWitnessId(witnessId);
    let total = 0;
    for (const contextRecord of this.contexts.values()) {
      total += contextRecord.witnesses.size;
    }
    if (total >= this.limits.maxWitnesses) {
      fail("capacity-exhausted", "witness table full");
    }
    if (record.witnesses.has(id)) {
      fail("unknown-witness", `witness id is single-use and already recorded: ${id}`);
    }
    const entry: WitnessRecord = {
      witnessId: id,
      owner,
      context,
      actionId: witness.actionId,
      outcome: witness.outcome,
    };
    record.witnesses.set(id, entry);
    return this.witnessFactsOf(record, entry);
  }

  private witnessFactsOf(record: ContextRecord, witness: WitnessRecord): ApplicationWitnessFacts {
    return Object.freeze({
      witnessId: witness.witnessId,
      owner: witness.owner,
      context: record.context,
      actionId: witness.actionId,
      outcome: witness.outcome,
      digest: digestWitness(
        witness.witnessId,
        witness.owner,
        record.context,
        witness.actionId,
        witness.outcome,
      ),
    });
  }

  witnessFacts(owner: string, context: string, witnessId: string): ApplicationWitnessFacts {
    const record = this.requireContext(owner, context);
    const witness = record.witnesses.get(witnessId);
    if (witness === undefined) {
      fail("unknown-witness", `no recorded witness: ${witnessId}`);
    }
    return this.witnessFactsOf(record, witness as WitnessRecord);
  }
}

// ---------------------------------------------------------------------------
// Settlement verdict: only a verified settlement judges
// ---------------------------------------------------------------------------

function checkSettlementProofShape(value: unknown): SettlementProofClaim {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail("forbidden-proof", "settlement proof must be an input-settlement object");
  }
  const record = value as Record<string, unknown>;
  if (record["kind"] === "input-settlement") {
    const settlement = record["settlement"] as InputSettlementFacts;
    if (settlement === null || typeof settlement !== "object" || Array.isArray(settlement)) {
      fail("forbidden-proof", "settlement proof carries no settlement facts");
    }
    return { kind: "input-settlement", settlement };
  }
  if (typeof record["kind"] === "string") {
    fail("forbidden-proof", `settlement proof kind is inadmissible: ${String(record["kind"])}`);
  }
  fail("forbidden-proof", "settlement proof carries no kind");
}

// Judge one settlement proof claim. Only a verified input settlement
// proves settlement: route-delivery, app-witness, dom-capture,
// input-attestation, and any other non-settlement claim reject with
// forbidden-proof before any verdict is read.
export function assertInputSettled(claim: unknown): SettlementVerdict {
  const proof = checkSettlementProofShape(claim);
  if (proof.kind !== "input-settlement") {
    fail("forbidden-proof", `settlement proof kind is inadmissible: ${proof.kind}`);
  }
  const settled = proof as { kind: "input-settlement"; settlement: InputSettlementFacts };
  const facts = settled.settlement;
  if (
    typeof facts.actionId !== "string" ||
    typeof facts.owner !== "string" ||
    typeof facts.context !== "string" ||
    typeof facts.action !== "string" ||
    typeof facts.target !== "string" ||
    typeof facts.outcome !== "string" ||
    (facts.fulfilledByRoute !== null && typeof facts.fulfilledByRoute !== "string") ||
    !Number.isSafeInteger(facts.domCaptures) ||
    typeof facts.digest !== "string"
  ) {
    fail("forbidden-proof", "settlement proof carries malformed facts");
  }
  if (!(SETTLEMENT_OUTCOMES as readonly string[]).includes(facts.outcome)) {
    fail("forbidden-proof", `settlement carries an undeclared outcome: ${facts.outcome}`);
  }
  if (!(INPUT_ACTIONS as readonly string[]).includes(facts.action)) {
    fail("forbidden-proof", `settlement carries an undeclared action: ${facts.action}`);
  }
  const recomputed = digestSettlement(
    facts.actionId,
    facts.owner,
    facts.context,
    facts.action,
    facts.target,
    facts.outcome,
    facts.fulfilledByRoute,
    facts.domCaptures,
  );
  if (recomputed !== facts.digest) {
    fail("forbidden-proof", "settlement digest does not verify");
  }
  // The abort-fulfills link is exact: aborted-by-route always names its
  // route, and direct outcomes never do.
  if (facts.outcome === "aborted-by-route" && facts.fulfilledByRoute === null) {
    fail("forbidden-proof", "aborted-by-route settlement names no fulfilling route");
  }
  if (facts.outcome !== "aborted-by-route" && facts.fulfilledByRoute !== null) {
    fail("forbidden-proof", "direct settlement carries a fulfilling route");
  }
  return Object.freeze({
    actionId: facts.actionId,
    outcome: facts.outcome,
    fulfilledByRoute: facts.fulfilledByRoute,
    settled: true,
    digest: digestText(`verdict:${facts.actionId}:${facts.outcome}`),
  });
}
