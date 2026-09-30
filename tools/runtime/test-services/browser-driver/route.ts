// K11: one-contact browser route tokens.
//
// A route service over in-memory doubles only: no browser, no process,
// no flags, no environment, no sampling, no I/O, no live hosts. The
// service binds owned contexts, holds routes with opaque minted tokens,
// contacts the (seeded) upstream exactly once per token, captures one
// immutable complete body per route, resolves each route exactly once
// with a declared resolve mode, and records deliveries under delivery
// ids that live in a namespace disjoint from contact ids. A second
// upstream contact for one token refuses, a partial body refuses with a
// named reason, an unknown delivery refuses, and a repeated native
// operation rejoins: an idempotent re-issue returns the same recorded
// result and never re-contacts.
//
// Independence scope from the launch/driver service under test (normative):
//   SHARES: nothing executable. The service never imports, calls, or reads
//   driver code, driver-decoded values, receipts, flags, environment
//   variables, or process state. The only strings that may coincide are
//   the context/route names under test (supplied by the test, not by the
//   driver) and the vocabulary this file uses for facts.
//   Facts carry digests only: raw bodies and raw URLs never cross.
//   PROVES: context admission, route identity, one-contact enforcement,
//   immutable captured bodies, resolve modes, disjoint contact/delivery
//   identity, idempotent rejoin, and error provenance (every failure
//   names its layer). Driver claims are compared AGAINST these facts;
//   the service never derives a fact FROM a driver claim.
//   Consequently a second contact, a partial body, or an unknown
//   delivery can never prove route handling: only a verified delivery
//   fact proves it.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, browsers, processes, or live runtimes. Local
// controls only. Live host-dependent controls wait for the corresponding
// qualified profile and Q task.

import { createHash, randomBytes } from "node:crypto";

export const BROWSER_ROUTE_SCHEMA_VERSION = "1" as const;

// ---------------------------------------------------------------------------
// Independence scope (normative)
// ---------------------------------------------------------------------------

export const BROWSER_ROUTE_INDEPENDENCE_SCOPE: string =
  "independent: shares no driver code, decoded values, receipts, flags, " +
  "environment, or process state; proves context admission, route identity, " +
  "one-contact enforcement, immutable captured bodies, resolve modes, " +
  "disjoint contact/delivery identity, idempotent rejoin, and layered " +
  "error provenance from its own seeded doubles alone";

// ---------------------------------------------------------------------------
// Layers, codes, and the layered error
// ---------------------------------------------------------------------------

export const BROWSER_ROUTE_LAYERS = ["route", "contact", "delivery"] as const;
export type BrowserRouteLayer = (typeof BROWSER_ROUTE_LAYERS)[number];

// Closed failure-code vocabulary. Every code maps to exactly one layer:
//   route: context admission, ownership, capacity, lifecycle, route
//     identity, tokens, held/resolved separation, and the operation
//     journal.
//   contact: the single upstream contact: fetch-once enforcement,
//     complete-body capture, and the named partial-body refusal.
//   delivery: resolve modes, delivery minting, and delivery reads.
export const BROWSER_ROUTE_CODES = [
  "unknown-context",
  "no-context",
  "wrong-owner",
  "capacity-exhausted",
  "context-closed",
  "forged-token",
  "unknown-route",
  "route-closed",
  "route-mismatch",
  "route-pending",
  "duplicate-contact",
  "partial-body",
  "unknown-body",
  "no-contact",
  "unknown-delivery",
  "unknown-resolve-mode",
] as const;
export type BrowserRouteCode = (typeof BROWSER_ROUTE_CODES)[number];

const CODE_LAYER: Readonly<Record<BrowserRouteCode, BrowserRouteLayer>> = {
  "unknown-context": "route",
  "no-context": "route",
  "wrong-owner": "route",
  "capacity-exhausted": "route",
  "context-closed": "route",
  "forged-token": "route",
  "unknown-route": "route",
  "route-closed": "route",
  "route-mismatch": "route",
  "route-pending": "route",
  "duplicate-contact": "contact",
  "partial-body": "contact",
  "unknown-body": "contact",
  "no-contact": "contact",
  "unknown-delivery": "delivery",
  "unknown-resolve-mode": "delivery",
};

export function layerOfRouteCode(code: BrowserRouteCode): BrowserRouteLayer {
  return CODE_LAYER[code];
}

export class BrowserRouteError extends Error {
  readonly layer: BrowserRouteLayer;
  readonly code: BrowserRouteCode;
  constructor(code: BrowserRouteCode, message: string) {
    super(`[${CODE_LAYER[code]}:${code}] ${message}`);
    this.name = "BrowserRouteError";
    this.layer = CODE_LAYER[code];
    this.code = code;
  }
}

function fail(code: BrowserRouteCode, message: string): never {
  throw new BrowserRouteError(code, message);
}

// ---------------------------------------------------------------------------
// Bounds and names
// ---------------------------------------------------------------------------

export type BrowserRouteLimits = Readonly<{
  maxContexts: number;
  maxRoutes: number;
  maxBodyBytes: number;
}>;

const LIMIT_KEYS = ["maxContexts", "maxRoutes", "maxBodyBytes"] as const;

export function checkRouteLimits(value: unknown): BrowserRouteLimits {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new BrowserRouteError("unknown-context", "limits must be an object");
  }
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    if (!(LIMIT_KEYS as readonly string[]).includes(key)) {
      throw new BrowserRouteError("unknown-context", `unknown limit field: ${key}`);
    }
  }
  const out: Record<string, number> = {};
  for (const key of LIMIT_KEYS) {
    const entry = record[key];
    if (!Number.isSafeInteger(entry) || (entry as number) <= 0) {
      throw new BrowserRouteError("unknown-context", `${key} must be a positive integer`);
    }
    out[key] = entry as number;
  }
  return Object.freeze(out) as BrowserRouteLimits;
}

// Finite HTTP method vocabulary: every route intercepts exactly one.
export const ROUTE_METHODS = ["GET", "POST", "PUT", "DELETE", "HEAD"] as const;
export type RouteMethod = (typeof ROUTE_METHODS)[number];

// Finite resolve modes: a contacted route resolves exactly once, in one
// of these modes. fulfill delivers the immutable captured body, abort
// refuses the route while still citing the captured body, and continue
// passes the route through with the captured body on record.
export const RESOLVE_MODES = ["fulfill", "abort", "continue"] as const;
export type ResolveMode = (typeof RESOLVE_MODES)[number];

// Finite partial-body reasons: an incomplete body refuses with exactly
// one of these named reasons.
export const PARTIAL_BODY_REASONS = ["truncated", "withheld", "aborted"] as const;
export type PartialBodyReason = (typeof PARTIAL_BODY_REASONS)[number];

// Finite journal operations: hold, the single contact, and the single
// resolve. Rejoins append nothing.
export const ROUTE_JOURNAL_OPS = ["hold", "contact", "resolve"] as const;
export type RouteJournalOp = (typeof ROUTE_JOURNAL_OPS)[number];

const MAX_NAME_LEN = 128;
const MAX_URL_LEN = 256;
const BODY_DIGEST_RE = /^sha256:[0-9a-f]{64}$/;
const URL_RE = /^[A-Za-z0-9/_.:?=&%+~-]+$/;

function checkName(value: unknown, what: string, code: BrowserRouteCode): string {
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

function checkRouteId(value: unknown): string {
  return checkName(value, "route", "unknown-route");
}

function checkDeliveryId(value: unknown): string {
  return checkName(value, "delivery", "unknown-delivery");
}

function checkMethod(value: unknown): RouteMethod {
  if (typeof value !== "string" || !(ROUTE_METHODS as readonly string[]).includes(value)) {
    fail("unknown-route", `not a declared route method: ${String(value)}`);
  }
  return value as RouteMethod;
}

// Local paths only: absolute URLs name live hosts, and live hosts wait
// for the qualified profile. Only bounded local paths route here.
function checkRouteUrl(value: unknown): string {
  if (typeof value !== "string" || value === "" || value.length > MAX_URL_LEN) {
    fail(
      "unknown-route",
      `route url must be a non-empty local path of at most ${MAX_URL_LEN} chars`,
    );
  }
  if (!value.startsWith("/") || value.startsWith("//") || !URL_RE.test(value)) {
    fail("unknown-route", `route url is not a bounded local path: ${value}`);
  }
  return value;
}

function checkResolveMode(value: unknown): ResolveMode {
  if (typeof value !== "string" || !(RESOLVE_MODES as readonly string[]).includes(value)) {
    fail("unknown-resolve-mode", `not a declared resolve mode: ${String(value)}`);
  }
  return value as ResolveMode;
}

function checkPartialReason(value: unknown): PartialBodyReason {
  if (typeof value !== "string" || !(PARTIAL_BODY_REASONS as readonly string[]).includes(value)) {
    fail("unknown-body", `not a declared partial-body reason: ${String(value)}`);
  }
  return value as PartialBodyReason;
}

function checkDescriptor(
  value: unknown,
  what: string,
  code: BrowserRouteCode,
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
  code: BrowserRouteCode,
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

export type RouteContextBindingFacts = Readonly<{
  owner: string;
  context: string;
  handleDigest: string;
}>;

export type RouteAttestation = Readonly<{
  routeId: string;
  owner: string;
  context: string;
  method: RouteMethod;
  urlDigest: string;
  contextDigest: string;
  handleDigest: string;
  held: true;
}>;

// Fact one: the single upstream contact. Only a complete body records;
// complete is therefore always true and the captured digest and length
// are immutable from here on.
export type ContactFacts = Readonly<{
  routeId: string;
  contactId: string;
  owner: string;
  context: string;
  bodyDigest: string;
  byteLength: number;
  complete: true;
  digest: string;
}>;

// Fact two: the route delivery. The delivery reuses the immutable
// captured body: resolve descriptors carry a mode only, never a body,
// so a delivery can never smuggle in a second contact's bytes.
export type DeliveryFacts = Readonly<{
  routeId: string;
  deliveryId: string;
  contactId: string;
  owner: string;
  context: string;
  mode: ResolveMode;
  bodyDigest: string;
  byteLength: number;
  digest: string;
}>;

export type RouteStatusFacts = Readonly<{
  routeId: string;
  owner: string;
  context: string;
  method: RouteMethod;
  urlDigest: string;
  held: boolean;
  contacted: boolean;
  contactId: string | null;
  resolved: boolean;
  deliveryId: string | null;
  mode: ResolveMode | null;
  digest: string;
}>;

export type RouteJournalEntry = Readonly<{
  seq: number;
  op: RouteJournalOp;
  digest: string;
}>;

export type RouteDescriptor = Readonly<{
  method: RouteMethod;
  url: string;
}>;

export type ContactBodyDescriptor = Readonly<{
  bodyDigest: string;
  byteLength: number;
}>;

export type ResolveDescriptor = Readonly<{
  mode: ResolveMode;
}>;

type RouteRecord = {
  routeId: string;
  owner: string;
  context: string;
  method: RouteMethod;
  url: string;
  token: string;
  held: boolean;
  contactId: string | null;
  bodyDigest: string | null;
  byteLength: number | null;
  deliveryId: string | null;
  mode: ResolveMode | null;
  journal: RouteJournalEntry[];
};

type DeliveryRecord = {
  deliveryId: string;
  routeId: string;
  contactId: string;
  mode: ResolveMode;
};

type ContextState = "open" | "closed";

type ContextRecord = {
  context: string;
  owner: string;
  token: string;
  state: ContextState;
  routes: Map<string, RouteRecord>;
  deliveries: Map<string, DeliveryRecord>;
};

function mintToken(prefix: string): string {
  return `${prefix}-${randomBytes(16).toString("hex")}`;
}

function digestText(text: string): string {
  return `sha256:${createHash("sha256").update(text, "utf8").digest("hex")}`;
}

// The contact digest is recomputed from carried fields, so contactFacts
// re-verifies a presented contact instead of trusting its digest string.
export function digestContact(
  routeId: string,
  contactId: string,
  owner: string,
  context: string,
  bodyDigest: string,
  byteLength: number,
): string {
  return digestText(
    `contact:${routeId}:${contactId}:${owner}:${context}:${bodyDigest}:${byteLength}`,
  );
}

export function digestDelivery(
  routeId: string,
  deliveryId: string,
  contactId: string,
  owner: string,
  context: string,
  mode: string,
  bodyDigest: string,
  byteLength: number,
): string {
  return digestText(
    `delivery:${routeId}:${deliveryId}:${contactId}:${owner}:${context}:${mode}:${bodyDigest}:${byteLength}`,
  );
}

// ---------------------------------------------------------------------------
// Service: owned contexts, one-contact routes, disjoint deliveries
// ---------------------------------------------------------------------------

export type RouteContextGrant = Readonly<{
  owner: string;
  context: string;
}>;

export class BrowserRouteService {
  private readonly limits: BrowserRouteLimits;
  private readonly grants = new Map<string, RouteContextGrant>();
  private readonly contexts = new Map<string, ContextRecord>();

  constructor(grants: readonly RouteContextGrant[], limits: BrowserRouteLimits) {
    if (grants.length === 0) {
      throw new BrowserRouteError("unknown-context", "declare at least one owned context");
    }
    if (grants.length > limits.maxContexts) {
      throw new BrowserRouteError("capacity-exhausted", "declared contexts exceed the cap");
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

  routeCount(owner: string, context: string): number {
    return this.requireContext(owner, context).routes.size;
  }

  deliveryCount(owner: string, context: string): number {
    return this.requireContext(owner, context).deliveries.size;
  }

  // Bind one owned context. Only an exact declared (owner, context) grant
  // admits; a context the test never granted rejects before any effect.
  // Contexts bind to one owner: a second identity joining the same
  // context name fails the owner check.
  bindContext(owner: string, context: string): RouteContextBindingFacts {
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
      routes: new Map(),
      deliveries: new Map(),
    };
    this.contexts.set(context, record);
    return this.bindingFactsOf(record);
  }

  private bindingFactsOf(record: ContextRecord): RouteContextBindingFacts {
    return Object.freeze({
      owner: record.owner,
      context: record.context,
      handleDigest: digestText(record.token),
    });
  }

  binding(owner: string, context: string): RouteContextBindingFacts {
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

  // Admission-first: the hold/contact/resolve gate reports only whether a
  // LIVE binding exists. Never bound, closed, or foreign all refuse with
  // no-context: a missing context blocks route work, whatever the reason.
  private requireLiveBinding(owner: string, context: string): ContextRecord {
    checkOwner(owner);
    checkContextName(context);
    const record = this.contexts.get(context);
    if (record === undefined || record.owner !== owner || record.state !== "open") {
      fail("no-context", `no live context binding: ${owner}/${context}`);
    }
    return record as ContextRecord;
  }

  // Close one context. Unresolved routes block the close: nothing
  // uncontacted or contacted-but-unresolved closes. Contacted and
  // resolved routes stay out of the way only once delivered.
  closeContext(owner: string, context: string): void {
    const record = this.requireContext(owner, context);
    for (const route of record.routes.values()) {
      if (route.held) {
        fail("route-pending", `route still held: ${route.routeId}`);
      }
    }
    record.state = "closed";
  }

  private checkRouteDescriptor(value: unknown): RouteDescriptor {
    const record = checkDescriptor(value, "route", "unknown-route");
    checkNoExtraKeys(record, ["method", "url"], "route", "unknown-route");
    const method = checkMethod(record["method"]);
    const url = checkRouteUrl(record["url"]);
    return { method, url };
  }

  // The body descriptor is validated before any token is read, so a
  // partial body refuses with its named reason even before admission
  // proves the token. Complete bodies carry no reason; incomplete
  // bodies name exactly one reason and always refuse.
  private checkContactBody(value: unknown): ContactBodyDescriptor {
    const record = checkDescriptor(value, "contact body", "unknown-body");
    checkNoExtraKeys(
      record,
      ["bodyDigest", "byteLength", "complete", "reason"],
      "contact body",
      "unknown-body",
    );
    const bodyDigest = record["bodyDigest"];
    if (typeof bodyDigest !== "string" || !BODY_DIGEST_RE.test(bodyDigest)) {
      fail("unknown-body", "contact body carries a digest, never raw bytes");
    }
    const byteLength = record["byteLength"];
    if (!Number.isSafeInteger(byteLength) || (byteLength as number) < 0) {
      fail("unknown-body", "contact body byte length must be a non-negative integer");
    }
    if ((byteLength as number) > this.limits.maxBodyBytes) {
      fail("capacity-exhausted", `contact body exceeds ${this.limits.maxBodyBytes} bytes`);
    }
    const complete = record["complete"];
    if (typeof complete !== "boolean") {
      fail("unknown-body", "contact body states complete explicitly");
    }
    const reason = record["reason"];
    if (complete) {
      if (reason !== undefined) {
        fail("unknown-body", "a complete body carries no partial-body reason");
      }
      return { bodyDigest: bodyDigest as string, byteLength: byteLength as number };
    }
    // Incomplete bodies refuse with the named reason: partial bodies
    // never record, whatever the token carries.
    fail("partial-body", `contact body incomplete: ${checkPartialReason(reason)}`);
  }

  private checkResolveDescriptor(value: unknown): ResolveDescriptor {
    const record = checkDescriptor(value, "resolve", "unknown-resolve-mode");
    checkNoExtraKeys(record, ["mode"], "resolve", "unknown-resolve-mode");
    return { mode: checkResolveMode(record["mode"]) };
  }

  // Hold one route. The route token is minted here and verified by table
  // lookup on every later mutation, so an invented token is never
  // authority. Route ids are single-use per context: a resolved id never
  // re-opens. Re-holding a held id with the identical descriptor
  // rejoins; a different descriptor refuses with route-mismatch rather
  // than retargeting the held route.
  holdRoute(
    owner: string,
    context: string,
    routeId: string,
    descriptor: unknown,
  ): RouteAttestation {
    const route = this.checkRouteDescriptor(descriptor);
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
        fail("route-closed", `route id is single-use and already resolved: ${id}`);
      }
      if (prior.method !== route.method || prior.url !== route.url) {
        fail("route-mismatch", `route id is already held with another descriptor: ${id}`);
      }
      return this.attestationOf(record, prior);
    }
    const entry: RouteRecord = {
      routeId: id,
      owner,
      context,
      method: route.method,
      url: route.url,
      token: mintToken("rte"),
      held: true,
      contactId: null,
      bodyDigest: null,
      byteLength: null,
      deliveryId: null,
      mode: null,
      journal: [],
    };
    record.routes.set(id, entry);
    this.appendJournal(entry, "hold");
    return this.attestationOf(record, entry);
  }

  private attestationOf(record: ContextRecord, route: RouteRecord): RouteAttestation {
    return Object.freeze({
      routeId: route.routeId,
      owner: route.owner,
      context: record.context,
      method: route.method,
      urlDigest: digestText(route.url),
      contextDigest: digestText(record.token),
      handleDigest: digestText(route.token),
      held: true as const,
    });
  }

  attestation(owner: string, context: string, routeId: string): RouteAttestation {
    const record = this.requireContext(owner, context);
    const route = this.requireRoute(record, routeId);
    if (!route.held) {
      fail("route-closed", `route is resolved, not held: ${routeId}`);
    }
    return this.attestationOf(record, route);
  }

  private requireRoute(record: ContextRecord, routeId: string): RouteRecord {
    const route = record.routes.get(routeId);
    if (route === undefined) {
      fail("unknown-route", `no held route: ${routeId}`);
    }
    return route as RouteRecord;
  }

  // Verify the route token by table lookup. Unknown routes and invented
  // tokens are never authority; resolved routes report route-closed.
  private requireLiveRoute(
    owner: string,
    context: string,
    routeId: string,
    token: string,
  ): { record: ContextRecord; route: RouteRecord } {
    const record = this.requireContext(owner, context);
    const route = this.requireRoute(record, routeId);
    if (!route.held) {
      fail("route-closed", `route is resolved: ${routeId}`);
    }
    if (token === "" || token !== route.token) {
      fail("forged-token", "route token is not the attested token");
    }
    return { record, route };
  }

  // Test-only accessor: the raw token crosses exactly here so bounded
  // controls can present it on later calls. Facts carry digests only.
  routeTokenForTest(owner: string, context: string, routeId: string): string {
    const record = this.requireContext(owner, context);
    return this.requireRoute(record, routeId).token;
  }

  private appendJournal(route: RouteRecord, op: RouteJournalOp): void {
    const seq = route.journal.length;
    route.journal.push(
      Object.freeze({
        seq,
        op,
        digest: digestText(`journal:${route.routeId}:${op}:${seq}`),
      }),
    );
  }

  // Contact the upstream once (fetch once). The first complete body
  // records under a minted contact id and freezes: the captured digest
  // and length are immutable from here on. A re-issue of the identical
  // native operation rejoins and returns the same recorded contact
  // without appending to the journal; any differing second contact
  // refuses with duplicate-contact.
  contactUpstream(
    owner: string,
    context: string,
    routeId: string,
    token: string,
    body: unknown,
  ): ContactFacts {
    const contact = this.checkContactBody(body);
    const { record, route } = this.requireLiveRoute(owner, context, routeId, token);
    if (route.contactId !== null) {
      if (route.bodyDigest === contact.bodyDigest && route.byteLength === contact.byteLength) {
        return this.contactOf(record, route);
      }
      fail("duplicate-contact", `route already contacted upstream: ${routeId}`);
    }
    route.contactId = mintToken("ctc");
    route.bodyDigest = contact.bodyDigest;
    route.byteLength = contact.byteLength;
    this.appendJournal(route, "contact");
    return this.contactOf(record, route);
  }

  private contactOf(record: ContextRecord, route: RouteRecord): ContactFacts {
    return Object.freeze({
      routeId: route.routeId,
      contactId: route.contactId as string,
      owner: route.owner,
      context: record.context,
      bodyDigest: route.bodyDigest as string,
      byteLength: route.byteLength as number,
      complete: true as const,
      digest: digestContact(
        route.routeId,
        route.contactId as string,
        route.owner,
        record.context,
        route.bodyDigest as string,
        route.byteLength as number,
      ),
    });
  }

  // Contact facts exist only for contacted routes. Uncontacted routes
  // refuse with no-contact: contacted and uncontacted are never
  // conflated, and a missing contact never reads as an empty body.
  contactFacts(owner: string, context: string, routeId: string): ContactFacts {
    const record = this.requireContext(owner, context);
    const route = this.requireRoute(record, routeId);
    if (route.contactId === null) {
      fail("no-contact", `route has no recorded upstream contact: ${routeId}`);
    }
    return this.contactOf(record, route);
  }

  // Resolve one contacted route in exactly one mode. Resolve descriptors
  // carry a mode only, never a body: the delivery reuses the immutable
  // captured body. Delivery ids mint into their own namespace, disjoint
  // from contact ids. Resolution is terminal: the id never re-opens and
  // never resolves twice.
  resolveRoute(
    owner: string,
    context: string,
    routeId: string,
    token: string,
    descriptor: unknown,
  ): DeliveryFacts {
    const resolve = this.checkResolveDescriptor(descriptor);
    const { record, route } = this.requireLiveRoute(owner, context, routeId, token);
    if (route.contactId === null) {
      fail("no-contact", `route has no recorded upstream contact: ${routeId}`);
    }
    route.deliveryId = mintToken("dlv");
    route.mode = resolve.mode;
    route.held = false;
    record.deliveries.set(route.deliveryId, {
      deliveryId: route.deliveryId,
      routeId: route.routeId,
      contactId: route.contactId,
      mode: resolve.mode,
    });
    this.appendJournal(route, "resolve");
    return this.deliveryOf(record, route);
  }

  private deliveryOf(record: ContextRecord, route: RouteRecord): DeliveryFacts {
    return Object.freeze({
      routeId: route.routeId,
      deliveryId: route.deliveryId as string,
      contactId: route.contactId as string,
      owner: route.owner,
      context: record.context,
      mode: route.mode as ResolveMode,
      bodyDigest: route.bodyDigest as string,
      byteLength: route.byteLength as number,
      digest: digestDelivery(
        route.routeId,
        route.deliveryId as string,
        route.contactId as string,
        route.owner,
        record.context,
        route.mode as string,
        route.bodyDigest as string,
        route.byteLength as number,
      ),
    });
  }

  // Delivery facts read by delivery id only. Unknown ids refuse with
  // unknown-delivery, and contact ids never resolve here: the contact
  // and delivery namespaces are disjoint.
  deliveryFacts(owner: string, context: string, deliveryId: string): DeliveryFacts {
    const record = this.requireContext(owner, context);
    const id = checkDeliveryId(deliveryId);
    const delivery = record.deliveries.get(id);
    if (delivery === undefined) {
      fail("unknown-delivery", `no recorded delivery: ${deliveryId}`);
    }
    return this.deliveryOf(record, this.requireRoute(record, (delivery as DeliveryRecord).routeId));
  }

  private statusDigestOf(route: RouteRecord): string {
    return digestText(
      `status:${route.routeId}:${route.held ? "held" : "resolved"}:${route.contactId ?? ""}:${route.deliveryId ?? ""}:${route.mode ?? ""}`,
    );
  }

  // Route status facts stay readable while held and after resolution:
  // the status is evidence, and held routes never hide it. Reads take
  // no token.
  routeFacts(owner: string, context: string, routeId: string): RouteStatusFacts {
    const record = this.requireContext(owner, context);
    const route = this.requireRoute(record, routeId);
    return Object.freeze({
      routeId: route.routeId,
      owner: route.owner,
      context: record.context,
      method: route.method,
      urlDigest: digestText(route.url),
      held: route.held,
      contacted: route.contactId !== null,
      contactId: route.contactId,
      resolved: !route.held,
      deliveryId: route.deliveryId,
      mode: route.mode,
      digest: this.statusDigestOf(route),
    });
  }

  // The operation journal records hold, the single contact, and the
  // single resolve. Rejoins append nothing, so the journal length
  // proves the upstream was contacted at most once.
  operationJournal(owner: string, context: string, routeId: string): readonly RouteJournalEntry[] {
    const record = this.requireContext(owner, context);
    return Object.freeze([...this.requireRoute(record, routeId).journal]);
  }
}
