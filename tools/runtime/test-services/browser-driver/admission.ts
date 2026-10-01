// K07: independent browser host-effect observation and launch admission.
//
// An INDEPENDENT host-UI observer over in-memory doubles only: no browser,
// no process, no flags, no environment, no sampling, no I/O. The observer
// binds to a declared host-UI effect scope at open, attests launches with
// opaque minted tokens, records per-tick UI-effect observations through
// confirmed disposal, marks interruptions and gaps explicitly, and publishes
// durable corrections. Observation survives driver death: a dead driver is
// a noted fact, never a cleanup witness and never the end of the interval.
// Observer loss makes the interval unknown, and a missing observer blocks
// launch: admission refuses without a live observer binding.
//
// Independence scope from the launch/driver service under test (normative):
//   SHARES: nothing executable. The observer never imports, calls, or reads
//   driver code, driver-decoded values, driver receipts, launch flags,
//   environment variables, or process state. The only strings that may
//   coincide are the scope/launch names under test (supplied by the test,
//   not by the driver) and the vocabulary this file uses for facts.
//   PROVES INDEPENDENTLY: scope admission, launch attestation, per-tick
//   UI-effect observations from its own seeded doubles alone, gap/unknown
//   marking, durable corrections, disposal receipts, and error provenance
//   (every failure names its layer). Driver claims are compared AGAINST
//   these facts; the observer never derives a fact FROM a driver claim.
//   Consequently no headless flag, HOME value, launch flag set, or process
//   sample can ever prove no-UI: only an observer attestation over a fully
//   observed interval with zero seen effects proves no-UI.
//
// Split with the Go mirror
// (tools/native-test-owner/host/browser_observer/observer.go):
//   TS owns typed interval mechanics, the admission adapter, and the
//   bounded self-check. The Go mirror owns the same contract outside the
//   launch/driver service under independent N/T ownership: scope
//   discipline, minted launch tokens, the interval through disposal, and
//   the unknown-seal. Both enforce the same contract: owned-scope
//   admission, opaque tokens verified by lookup, digest-only frozen facts,
//   and layered errors. Neither reads the driver.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, browsers, processes, or live runtimes. Local
// controls only.

import { createHash, randomBytes } from "node:crypto";

export const BROWSER_ADMISSION_SCHEMA_VERSION = "1" as const;

// ---------------------------------------------------------------------------
// Independence scope (normative, mirrored in observer.go)
// ---------------------------------------------------------------------------

export const BROWSER_OBSERVER_INDEPENDENCE_SCOPE: string =
  "independent: shares no driver code, decoded values, receipts, flags, " +
  "environment, or process state; proves scope receipt, launch attestation, " +
  "per-tick UI effects, gaps, corrections, disposal, and layered error " +
  "provenance from its own seeded doubles alone";

// ---------------------------------------------------------------------------
// Layers, codes, and the layered error
// ---------------------------------------------------------------------------

export const BROWSER_OBSERVER_LAYERS = ["observer", "admission", "interval"] as const;
export type BrowserObserverLayer = (typeof BROWSER_OBSERVER_LAYERS)[number];

// Closed failure-code vocabulary. Every code maps to exactly one layer:
//   observer: scope admission, capacity, ownership, lifecycle, loss.
//   admission: launch binding, tokens, forbidden no-UI proofs.
//   interval: effect keys, ticks, gaps, corrections, disposal seal.
export const BROWSER_OBSERVER_CODES = [
  "unknown-scope",
  "scope-closed",
  "capacity-exhausted",
  "wrong-owner",
  "observer-lost",
  "observer-interrupted",
  "no-observer",
  "forged-token",
  "unknown-launch",
  "launch-closed",
  "forbidden-proof",
  "unknown-effect",
  "out-of-scope",
  "gap-open",
  "correction-unknown",
  "correction-stale",
  "disposal-open",
  "already-disposed",
] as const;
export type BrowserObserverCode = (typeof BROWSER_OBSERVER_CODES)[number];

const CODE_LAYER: Readonly<Record<BrowserObserverCode, BrowserObserverLayer>> = {
  "unknown-scope": "observer",
  "scope-closed": "observer",
  "capacity-exhausted": "observer",
  "wrong-owner": "observer",
  "observer-lost": "observer",
  "observer-interrupted": "observer",
  "no-observer": "admission",
  "forged-token": "admission",
  "unknown-launch": "admission",
  "launch-closed": "admission",
  "forbidden-proof": "admission",
  "unknown-effect": "interval",
  "out-of-scope": "interval",
  "gap-open": "interval",
  "correction-unknown": "interval",
  "correction-stale": "interval",
  "disposal-open": "interval",
  "already-disposed": "interval",
};

export function layerOfCode(code: BrowserObserverCode): BrowserObserverLayer {
  return CODE_LAYER[code];
}

export class BrowserObserverError extends Error {
  readonly layer: BrowserObserverLayer;
  readonly code: BrowserObserverCode;
  constructor(code: BrowserObserverCode, message: string) {
    super(`[${CODE_LAYER[code]}:${code}] ${message}`);
    this.name = "BrowserObserverError";
    this.layer = CODE_LAYER[code];
    this.code = code;
  }
}

function fail(code: BrowserObserverCode, message: string): never {
  throw new BrowserObserverError(code, message);
}

// ---------------------------------------------------------------------------
// Bounds and names
// ---------------------------------------------------------------------------

export type BrowserObserverLimits = Readonly<{
  maxScopes: number;
  maxLaunches: number;
  maxTicksPerLaunch: number;
  maxCorrectionsPerLaunch: number;
}>;

const LIMIT_KEYS = [
  "maxScopes",
  "maxLaunches",
  "maxTicksPerLaunch",
  "maxCorrectionsPerLaunch",
] as const;

export function checkLimits(value: unknown): BrowserObserverLimits {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new BrowserObserverError("unknown-effect", "limits must be an object");
  }
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    if (!(LIMIT_KEYS as readonly string[]).includes(key)) {
      throw new BrowserObserverError("unknown-effect", `unknown limit field: ${key}`);
    }
  }
  const out: Record<string, number> = {};
  for (const key of LIMIT_KEYS) {
    const entry = record[key];
    if (!Number.isSafeInteger(entry) || (entry as number) <= 0) {
      throw new BrowserObserverError("unknown-effect", `${key} must be a positive integer`);
    }
    out[key] = entry as number;
  }
  return Object.freeze(out) as BrowserObserverLimits;
}

// Finite host-UI effect vocabulary. A scope observes a non-empty subset;
// anything outside this vocabulary is not an effect key at all.
export const EFFECT_CHANNELS = ["window", "icon", "notification", "focus"] as const;
export type EffectChannel = (typeof EFFECT_CHANNELS)[number];

// Directly observed states: the observer either saw a host-UI effect on
// the channel during the tick or it saw none. Derived states (gap,
// unknown) are never accepted from observe(): they arise only from
// endTick, interruption, loss, or durable publication.
export const OBSERVED_STATES = ["seen", "absent"] as const;
export type ObservedState = (typeof OBSERVED_STATES)[number];

export const EFFECT_STATES = ["seen", "absent", "gap", "unknown"] as const;
export type EffectState = (typeof EFFECT_STATES)[number];

const MAX_NAME_LEN = 128;

function checkName(value: unknown, what: string, code: BrowserObserverCode): string {
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

function checkScopeName(value: unknown): string {
  return checkName(value, "scope", "unknown-scope");
}

function checkLaunchName(value: unknown): string {
  return checkName(value, "launch", "unknown-launch");
}

function checkChannel(value: unknown): EffectChannel {
  if (typeof value !== "string" || !(EFFECT_CHANNELS as readonly string[]).includes(value)) {
    fail("unknown-effect", `not a declared host-UI effect key: ${String(value)}`);
  }
  return value as EffectChannel;
}

function checkObservedState(value: unknown): ObservedState {
  if (typeof value !== "string" || !(OBSERVED_STATES as readonly string[]).includes(value)) {
    fail("unknown-effect", `observation must be seen or absent, got: ${String(value)}`);
  }
  return value as ObservedState;
}

// ---------------------------------------------------------------------------
// Facts (frozen, digest-only)
// ---------------------------------------------------------------------------

export type ScopeReceipt = Readonly<{
  scope: string;
  owner: string;
  channels: readonly EffectChannel[];
  handleDigest: string;
  launches: number;
}>;

export type LaunchAttestation = Readonly<{
  launchId: string;
  scope: string;
  owner: string;
  scopeDigest: string;
  observerDigest: string;
  handleDigest: string;
  tick: number;
}>;

export type ChannelFacts = Readonly<{
  channel: EffectChannel;
  state: EffectState;
  // Origin of the state: observed (direct observe), gap (endTick default),
  // interruption, loss, correction, or published-unknown.
  origin: string;
  corrected: boolean;
}>;

export type TickFacts = Readonly<{
  launchId: string;
  tick: number;
  channels: readonly ChannelFacts[];
  closed: boolean;
  digest: string;
}>;

export type CorrectionFacts = Readonly<{
  correctionId: string;
  launchId: string;
  tick: number;
  channel: EffectChannel;
  before: EffectState;
  after: EffectState;
  digest: string;
}>;

export type IntervalFacts = Readonly<{
  launchId: string;
  scope: string;
  owner: string;
  ticks: readonly TickFacts[];
  corrections: readonly CorrectionFacts[];
  driverDeathNoted: boolean;
  disposed: boolean;
  // True when any tick carries unknown (interruption, loss, or published
  // unknown) or the observer was lost after full observation. An unknown
  // interval can never prove no-UI.
  unknown: boolean;
  digest: string;
}>;

export type DisposalReceipt = Readonly<{
  launchId: string;
  scope: string;
  owner: string;
  ticks: number;
  corrections: number;
  driverDeathNoted: boolean;
  unknown: boolean;
  digest: string;
}>;

// A no-UI proof claim. Only observer-attestation is admissible; every
// other kind rejects with forbidden-proof before any verdict is read.
export const FORBIDDEN_PROOF_KINDS = [
  "headless-flag",
  "home-env",
  "launch-flags",
  "process-sample",
] as const;
export type ForbiddenProofKind = (typeof FORBIDDEN_PROOF_KINDS)[number];

export type NoUiProofClaim =
  | Readonly<{ kind: "observer-attestation"; interval: IntervalFacts }>
  | Readonly<{ kind: ForbiddenProofKind; detail: string }>;

export type NoUiVerdict = Readonly<{
  launchId: string;
  uiSeen: boolean;
  unknown: boolean;
  digest: string;
}>;

type TickEntry = {
  state: EffectState;
  origin: string;
  corrected: boolean;
};

type TickRecord = {
  tick: number;
  entries: Map<EffectChannel, TickEntry>;
  closed: boolean;
};

type LaunchRecord = {
  launchId: string;
  scope: string;
  owner: string;
  token: string;
  ticks: TickRecord[];
  corrections: CorrectionFacts[];
  correctionSeq: number;
  driverDeathNoted: boolean;
  disposed: boolean;
  disposal: DisposalReceipt | null;
  // Lost records observer loss for a live interval. Once set, the
  // interval is unknown even when every tick was fully observed: the
  // observer died before the seal, so late host effects between the last
  // observation and disposal are unobserved. Mirrors Go launchRecord.lost.
  lost: boolean;
};

type ScopeState = "open" | "interrupted" | "lost" | "closed";

type ScopeRecord = {
  scope: string;
  owner: string;
  channels: readonly EffectChannel[];
  token: string;
  state: ScopeState;
  launches: Map<string, LaunchRecord>;
};

function mintToken(prefix: string): string {
  return `${prefix}-${randomBytes(16).toString("hex")}`;
}

function digestText(text: string): string {
  return `sha256:${createHash("sha256").update(text, "utf8").digest("hex")}`;
}

function canonicalTick(tick: TickRecord): string {
  const parts: string[] = [`tick:${tick.tick}:closed:${tick.closed ? 1 : 0}`];
  const channels = [...tick.entries.keys()].sort();
  for (const channel of channels) {
    const entry = tick.entries.get(channel) as TickEntry;
    parts.push(`${channel}=${entry.state}/${entry.origin}/${entry.corrected ? 1 : 0}`);
  }
  return parts.join("|");
}

// The interval digest is recomputed from carried ticks and corrections, so
// assertNoUiProof can re-verify a presented attestation instead of trusting
// its digest string.
export function digestInterval(
  launchId: string,
  ticks: readonly TickFacts[],
  corrections: readonly CorrectionFacts[],
  flags: string,
): string {
  const hash = createHash("sha256");
  hash.update(launchId, "utf8");
  hash.update("\0", "utf8");
  for (const tick of ticks) {
    hash.update(String(tick.tick), "utf8");
    hash.update(tick.closed ? "C" : "O", "utf8");
    for (const entry of tick.channels) {
      hash.update(
        `${entry.channel}=${entry.state}/${entry.origin}/${entry.corrected ? 1 : 0}`,
        "utf8",
      );
      hash.update(";", "utf8");
    }
    hash.update("\n", "utf8");
  }
  hash.update("\0", "utf8");
  for (const correction of corrections) {
    hash.update(
      `${correction.correctionId}:${correction.tick}:${correction.channel}:${correction.before}>${correction.after}`,
      "utf8",
    );
    hash.update(";", "utf8");
  }
  hash.update("\0", "utf8");
  hash.update(flags, "utf8");
  return `sha256:${hash.digest("hex")}`;
}

// ---------------------------------------------------------------------------
// Service: owned scope, launch attestation, interval through disposal
// ---------------------------------------------------------------------------

export type ScopeGrant = Readonly<{
  owner: string;
  scope: string;
  channels: readonly EffectChannel[];
}>;

export class BrowserObserverService {
  private readonly limits: BrowserObserverLimits;
  private readonly grants = new Map<string, ScopeGrant>();
  private readonly scopes = new Map<string, ScopeRecord>();

  constructor(grants: readonly ScopeGrant[], limits: BrowserObserverLimits) {
    if (grants.length === 0) {
      throw new BrowserObserverError("unknown-scope", "declare at least one owned scope");
    }
    if (grants.length > limits.maxScopes) {
      throw new BrowserObserverError("capacity-exhausted", "declared scopes exceed the cap");
    }
    for (const grant of grants) {
      checkOwner(grant.owner);
      checkScopeName(grant.scope);
      if (grant.channels.length === 0) {
        fail("unknown-scope", `scope ${grant.scope} declares no effect channels`);
      }
      if (grant.channels.length > EFFECT_CHANNELS.length) {
        fail("capacity-exhausted", `scope ${grant.scope} declares too many channels`);
      }
      const seen = new Set<string>();
      for (const channel of grant.channels) {
        checkChannel(channel);
        if (seen.has(channel)) {
          fail("unknown-scope", `scope ${grant.scope} repeats channel: ${channel}`);
        }
        seen.add(channel);
      }
      const key = `${grant.owner}\0${grant.scope}`;
      if (this.grants.has(key)) {
        fail("unknown-scope", `duplicate owned scope: ${grant.owner}/${grant.scope}`);
      }
      this.grants.set(
        key,
        Object.freeze({ owner: grant.owner, scope: grant.scope, channels: [...grant.channels] }),
      );
    }
    this.limits = limits;
  }

  get scopeCount(): number {
    return this.scopes.size;
  }

  launchCount(owner: string, scope: string): number {
    return this.requireScope(owner, scope).launches.size;
  }

  // Open one owned scope. Only an exact declared (owner, scope) grant
  // admits; a scope the test never granted rejects before any effect.
  // Scopes bind to one owner: a second identity joining the same scope
  // name fails the owner check.
  openScope(owner: string, scope: string): ScopeReceipt {
    checkOwner(owner);
    checkScopeName(scope);
    const grant = this.grants.get(`${owner}\0${scope}`);
    if (grant === undefined) {
      fail("unknown-scope", `no owned grant for scope: ${owner}/${scope}`);
    }
    const prior = this.scopes.get(scope);
    if (prior !== undefined && prior.state !== "closed") {
      if (prior.owner !== owner) {
        fail("wrong-owner", `scope is owned by another identity: ${scope}`);
      }
      return this.receiptOf(prior);
    }
    if (this.scopes.size >= this.limits.maxScopes) {
      fail("capacity-exhausted", "scope table full");
    }
    const record: ScopeRecord = {
      scope,
      owner,
      channels: (grant as ScopeGrant).channels,
      token: mintToken("obs"),
      state: "open",
      launches: new Map(),
    };
    this.scopes.set(scope, record);
    return this.receiptOf(record);
  }

  private receiptOf(record: ScopeRecord): ScopeReceipt {
    return Object.freeze({
      scope: record.scope,
      owner: record.owner,
      channels: Object.freeze([...record.channels]),
      handleDigest: digestText(record.token),
      launches: record.launches.size,
    });
  }

  receipt(owner: string, scope: string): ScopeReceipt {
    return this.receiptOf(this.requireScope(owner, scope));
  }

  private requireScope(owner: string, scope: string): ScopeRecord {
    const record = this.scopes.get(scope);
    if (record === undefined) {
      fail("unknown-scope", `no open scope for owner: ${owner}/${scope}`);
    }
    const scopeRecord = record as ScopeRecord;
    if (scopeRecord.owner !== owner) {
      fail("wrong-owner", "scope is owned by another identity");
    }
    if (scopeRecord.state === "closed") {
      fail("scope-closed", `scope is closed: ${owner}/${scope}`);
    }
    if (scopeRecord.state === "lost") {
      fail("observer-lost", `observer is lost for scope: ${owner}/${scope}`);
    }
    return scopeRecord;
  }

  // Admission-first: the launch gate reports only whether a LIVE binding
  // exists. Never opened, closed, lost, or interrupted all refuse with
  // no-observer: a missing observer blocks launch, whatever the reason.
  private requireLiveBinding(owner: string, scope: string): ScopeRecord {
    checkOwner(owner);
    checkScopeName(scope);
    const record = this.scopes.get(scope);
    if (record === undefined || record.owner !== owner || record.state !== "open") {
      fail("no-observer", `no live observer binding for launch: ${owner}/${scope}`);
    }
    return record as ScopeRecord;
  }

  closeScope(owner: string, scope: string): void {
    const record = this.requireScope(owner, scope);
    for (const launch of record.launches.values()) {
      if (!launch.disposed) {
        fail("disposal-open", `launch interval still open: ${launch.launchId}`);
      }
    }
    record.state = "closed";
  }

  // Interrupt the observer: the open tick's unobserved channels become
  // unknown (interruption erases the observed/gap distinction for the open
  // tick), observation suspends until resume, and the interrupted ticks
  // stay unknown forever. Already-observed entries are untouched.
  interruptScope(owner: string, scope: string): void {
    const record = this.requireScope(owner, scope);
    if (record.state === "interrupted") {
      fail("observer-interrupted", `scope already interrupted: ${owner}/${scope}`);
    }
    // Refusal precedes effect: every live launch needs room for its
    // post-resume tick before anything is marked (Go parity).
    for (const launch of record.launches.values()) {
      if (launch.disposed) continue;
      const open = launch.ticks[launch.ticks.length - 1];
      if (open === undefined || open.closed) continue;
      if (launch.ticks.length >= this.limits.maxTicksPerLaunch) {
        fail("capacity-exhausted", "tick table full: dispose the launch");
      }
    }
    record.state = "interrupted";
    for (const launch of record.launches.values()) {
      if (launch.disposed) continue;
      const open = launch.ticks[launch.ticks.length - 1];
      if (open === undefined || open.closed) continue;
      for (const channel of record.channels) {
        if (!open.entries.has(channel)) {
          open.entries.set(channel, { state: "unknown", origin: "interruption", corrected: false });
        }
      }
      open.closed = true;
      // Post-resume observation needs an open tick; the interrupted tick
      // stays closed and unknown forever.
      launch.ticks.push({ tick: open.tick + 1, entries: new Map(), closed: false });
    }
  }

  resumeScope(owner: string, scope: string): void {
    const record = this.scopes.get(scope);
    if (record === undefined) {
      fail("unknown-scope", `no open scope for owner: ${owner}/${scope}`);
    }
    const scopeRecord = record as ScopeRecord;
    if (scopeRecord.owner !== owner) {
      fail("wrong-owner", "scope is owned by another identity");
    }
    if (scopeRecord.state === "closed") {
      fail("scope-closed", `scope is closed: ${owner}/${scope}`);
    }
    if (scopeRecord.state === "lost") {
      fail("observer-lost", `lost observer cannot resume: ${owner}/${scope}`);
    }
    if (scopeRecord.state !== "interrupted") {
      fail("observer-interrupted", `scope is not interrupted: ${owner}/${scope}`);
    }
    scopeRecord.state = "open";
  }

  // Lose the observer permanently: every live interval becomes unknown and
  // stays unknown; only an honest unknown-seal remains. Lost observers
  // never resume and never admit new launches. The lost flag seals unknown
  // even for fully observed ticks (Go parity).
  loseObserver(owner: string, scope: string): void {
    const record = this.requireScope(owner, scope);
    record.state = "lost";
    for (const launch of record.launches.values()) {
      if (launch.disposed) continue;
      launch.lost = true;
      for (const tick of launch.ticks) {
        for (const channel of record.channels) {
          const entry = tick.entries.get(channel);
          if (entry === undefined) {
            tick.entries.set(channel, { state: "unknown", origin: "loss", corrected: false });
          } else if (entry.state === "gap") {
            entry.state = "unknown";
            entry.origin = "loss";
          }
        }
        tick.closed = true;
      }
    }
  }

  // Attest one launch under a live observer binding. The launch token is
  // minted here and verified by table lookup on every later call, so an
  // invented token is never authority.
  attestLaunch(owner: string, scope: string, launchId: string): LaunchAttestation {
    // Name validation precedes the binding check (Go AttestLaunch order):
    // a malformed launch name rejects as unknown-launch even with no live
    // binding. requireLiveBinding re-validates owner and scope idempotently.
    checkOwner(owner);
    checkScopeName(scope);
    checkLaunchName(launchId);
    const record = this.requireLiveBinding(owner, scope);
    let total = 0;
    for (const scopeRecord of this.scopes.values()) {
      total += scopeRecord.launches.size;
    }
    if (total >= this.limits.maxLaunches) {
      fail("capacity-exhausted", "launch table full");
    }
    const prior = record.launches.get(launchId);
    if (prior !== undefined) {
      if (!prior.disposed) {
        return this.attestationOf(record, prior);
      }
      fail("launch-closed", `launch id is single-use and already disposed: ${launchId}`);
    }
    const launch: LaunchRecord = {
      launchId,
      scope,
      owner,
      token: mintToken("lnch"),
      ticks: [{ tick: 0, entries: new Map(), closed: false }],
      corrections: [],
      correctionSeq: 0,
      driverDeathNoted: false,
      disposed: false,
      disposal: null,
      lost: false,
    };
    record.launches.set(launchId, launch);
    return this.attestationOf(record, launch);
  }

  private attestationOf(record: ScopeRecord, launch: LaunchRecord): LaunchAttestation {
    const open = launch.ticks[launch.ticks.length - 1] as TickRecord;
    return Object.freeze({
      launchId: launch.launchId,
      scope: record.scope,
      owner: record.owner,
      scopeDigest: digestText(
        `scope:${record.owner}:${record.scope}:${[...record.channels].sort().join(",")}`,
      ),
      observerDigest: digestText(record.token),
      handleDigest: digestText(launch.token),
      tick: open.tick,
    });
  }

  attestation(owner: string, scope: string, launchId: string): LaunchAttestation {
    const record = this.requireScope(owner, scope);
    const launch = this.requireLaunch(record, launchId);
    return this.attestationOf(record, launch);
  }

  private requireLaunch(record: ScopeRecord, launchId: string): LaunchRecord {
    const launch = record.launches.get(launchId);
    if (launch === undefined) {
      fail("unknown-launch", `no attested launch: ${launchId}`);
    }
    return launch as LaunchRecord;
  }

  // Verify the launch token by table lookup. Unknown launches and invented
  // tokens are never authority; disposed launches report launch-closed.
  private requireLiveLaunch(
    owner: string,
    scope: string,
    launchId: string,
    token: string,
  ): { record: ScopeRecord; launch: LaunchRecord } {
    const record = this.requireScope(owner, scope);
    if (record.state === "interrupted") {
      fail("observer-interrupted", `observer is interrupted for scope: ${owner}/${scope}`);
    }
    const launch = this.requireLaunch(record, launchId);
    if (launch.disposed) {
      fail("launch-closed", `launch is disposed: ${launchId}`);
    }
    if (token === "" || token !== launch.token) {
      fail("forged-token", "launch token is not the attested token");
    }
    return { record, launch };
  }

  // Test-only accessor: the raw token crosses exactly here so bounded
  // controls can present it on later calls. Facts carry digests only.
  launchTokenForTest(owner: string, scope: string, launchId: string): string {
    const record = this.requireScope(owner, scope);
    return this.requireLaunch(record, launchId).token;
  }

  // Record one UI-effect observation at the launch's open tick. Only the
  // launch's declared scope channels admit; a channel outside the finite
  // vocabulary is an unknown effect key, and a vocabulary channel outside
  // the declared scope is out of scope. Both reject without effect.
  observe(
    owner: string,
    scope: string,
    launchId: string,
    token: string,
    channel: string,
    state: string,
  ): TickFacts {
    const key = checkChannel(channel);
    const observed = checkObservedState(state);
    const { record, launch } = this.requireLiveLaunch(owner, scope, launchId, token);
    if (!record.channels.includes(key)) {
      fail("out-of-scope", `channel is outside the declared scope: ${key}`);
    }
    const open = launch.ticks[launch.ticks.length - 1] as TickRecord;
    if (open.closed) {
      fail("gap-open", `tick ${open.tick} is closed; end the tick before observing`);
    }
    open.entries.set(key, { state: observed, origin: "observed", corrected: false });
    return this.tickFactsOf(launch, open);
  }

  // Close the open tick and open the next one. Channels with no
  // observation become explicit gap entries: silence is a gap, never
  // proof of absence.
  endTick(owner: string, scope: string, launchId: string, token: string): TickFacts {
    const { record, launch } = this.requireLiveLaunch(owner, scope, launchId, token);
    const open = launch.ticks[launch.ticks.length - 1] as TickRecord;
    if (open.closed) {
      fail("gap-open", `tick ${open.tick} is already closed`);
    }
    for (const channel of record.channels) {
      if (!open.entries.has(channel)) {
        open.entries.set(channel, { state: "gap", origin: "gap", corrected: false });
      }
    }
    open.closed = true;
    const facts = this.tickFactsOf(launch, open);
    if (launch.ticks.length >= this.limits.maxTicksPerLaunch) {
      fail("capacity-exhausted", "tick table full: dispose the launch");
    }
    launch.ticks.push({ tick: open.tick + 1, entries: new Map(), closed: false });
    return facts;
  }

  // Note the driver's death. The interval continues: late host effects
  // after driver death are observed through confirmed disposal, and a
  // dead driver never seals anything on its own.
  noteDriverDeath(owner: string, scope: string, launchId: string, token: string): void {
    const { launch } = this.requireLiveLaunch(owner, scope, launchId, token);
    launch.driverDeathNoted = true;
  }

  // Publish one durable correction. Only gap and unknown entries admit
  // correction, exactly once each: a directly observed seen/absent entry
  // is never rewritten, and a corrected entry is never corrected again.
  // Corrections are journaled in order and survive disposal.
  publishCorrection(
    owner: string,
    scope: string,
    launchId: string,
    token: string,
    tick: number,
    channel: string,
    after: string,
  ): CorrectionFacts {
    const key = checkChannel(channel);
    const corrected = checkObservedState(after);
    const { record, launch } = this.requireLiveLaunch(owner, scope, launchId, token);
    if (!record.channels.includes(key)) {
      fail("out-of-scope", `channel is outside the declared scope: ${key}`);
    }
    if (!Number.isSafeInteger(tick) || tick < 0) {
      fail("correction-unknown", `no such tick: ${String(tick)}`);
    }
    const tickRecord = launch.ticks[tick];
    if (tickRecord === undefined || !tickRecord.closed) {
      fail("correction-unknown", `no closed tick entry for tick ${tick}`);
    }
    const entry = (tickRecord as TickRecord).entries.get(key);
    if (entry === undefined) {
      fail("correction-unknown", `no entry for tick ${tick} channel ${key}`);
    }
    const target = entry as TickEntry;
    if (target.state !== "gap" && target.state !== "unknown") {
      fail("correction-stale", `tick ${tick} channel ${key} holds a direct observation`);
    }
    if (target.corrected) {
      fail("correction-stale", `tick ${tick} channel ${key} is already corrected`);
    }
    if (launch.corrections.length >= this.limits.maxCorrectionsPerLaunch) {
      fail("capacity-exhausted", "correction journal full");
    }
    launch.correctionSeq += 1;
    const facts: CorrectionFacts = Object.freeze({
      correctionId: `c${launch.correctionSeq}`,
      launchId,
      tick,
      channel: key,
      before: target.state,
      after: corrected,
      digest: digestText(`correction:${launchId}:${tick}:${key}:${target.state}>${corrected}`),
    });
    target.state = corrected;
    target.origin = "correction";
    target.corrected = true;
    launch.corrections.push(facts);
    return facts;
  }

  // Publish a durable unknown for one gap entry: the gap is explicitly
  // unresolved, and the interval stays honestly unknown. Unknown entries
  // can never prove no-UI.
  markUnknown(
    owner: string,
    scope: string,
    launchId: string,
    token: string,
    tick: number,
    channel: string,
  ): CorrectionFacts {
    const key = checkChannel(channel);
    const { record, launch } = this.requireLiveLaunch(owner, scope, launchId, token);
    if (!record.channels.includes(key)) {
      fail("out-of-scope", `channel is outside the declared scope: ${key}`);
    }
    if (!Number.isSafeInteger(tick) || tick < 0) {
      fail("correction-unknown", `no such tick: ${String(tick)}`);
    }
    const tickRecord = launch.ticks[tick];
    if (tickRecord === undefined || !tickRecord.closed) {
      fail("correction-unknown", `no closed tick entry for tick ${tick}`);
    }
    const entry = (tickRecord as TickRecord).entries.get(key);
    if (entry === undefined) {
      fail("correction-unknown", `no entry for tick ${tick} channel ${key}`);
    }
    const target = entry as TickEntry;
    if (target.state !== "gap") {
      fail("correction-stale", `tick ${tick} channel ${key} is not an open gap`);
    }
    if (launch.corrections.length >= this.limits.maxCorrectionsPerLaunch) {
      fail("capacity-exhausted", "correction journal full");
    }
    launch.correctionSeq += 1;
    const facts: CorrectionFacts = Object.freeze({
      correctionId: `c${launch.correctionSeq}`,
      launchId,
      tick,
      channel: key,
      before: "gap",
      after: "unknown",
      digest: digestText(`correction:${launchId}:${tick}:${key}:gap>unknown`),
    });
    target.state = "unknown";
    target.origin = "published-unknown";
    target.corrected = true;
    launch.corrections.push(facts);
    return facts;
  }

  private tickFactsOf(launch: LaunchRecord, tick: TickRecord): TickFacts {
    const channels: ChannelFacts[] = [];
    for (const [channel, entry] of [...tick.entries.entries()].sort(([a], [b]) =>
      a < b ? -1 : 1,
    )) {
      channels.push(
        Object.freeze({
          channel,
          state: entry.state,
          origin: entry.origin,
          corrected: entry.corrected,
        }),
      );
    }
    return Object.freeze({
      launchId: launch.launchId,
      tick: tick.tick,
      channels: Object.freeze(channels),
      closed: tick.closed,
      digest: digestText(`tick:${launch.launchId}:${canonicalTick(tick)}`),
    });
  }

  tickFacts(owner: string, scope: string, launchId: string, tick: number): TickFacts {
    const record = this.requireScope(owner, scope);
    const launch = this.requireLaunch(record, launchId);
    const found = launch.ticks[tick];
    if (found === undefined) {
      fail("unknown-effect", `no such tick: ${String(tick)}`);
    }
    return this.tickFactsOf(launch, found as TickRecord);
  }

  private intervalUnknown(launch: LaunchRecord): boolean {
    if (launch.lost) {
      return true;
    }
    for (const tick of launch.ticks) {
      for (const entry of tick.entries.values()) {
        if (entry.state === "unknown") return true;
      }
    }
    return false;
  }

  private intervalFlags(launch: LaunchRecord): string {
    return [
      `driver-death:${launch.driverDeathNoted ? 1 : 0}`,
      `disposed:${launch.disposed ? 1 : 0}`,
      `unknown:${this.intervalUnknown(launch) ? 1 : 0}`,
    ].join("|");
  }

  // Interval facts stay readable after disposal and after observer loss:
  // the interval is evidence, and loss makes it unknown rather than
  // unreadable.
  intervalFacts(owner: string, scope: string, launchId: string): IntervalFacts {
    const record = this.scopes.get(scope);
    if (record === undefined) {
      fail("unknown-scope", `no open scope for owner: ${owner}/${scope}`);
    }
    const scopeRecord = record as ScopeRecord;
    if (scopeRecord.owner !== owner) {
      fail("wrong-owner", "scope is owned by another identity");
    }
    if (scopeRecord.state === "closed") {
      fail("scope-closed", `scope is closed: ${owner}/${scope}`);
    }
    const launch = this.requireLaunch(scopeRecord, launchId);
    const ticks = launch.ticks.map((tick) => this.tickFactsOf(launch, tick));
    const corrections = Object.freeze([...launch.corrections]);
    const flags = this.intervalFlags(launch);
    return Object.freeze({
      launchId: launch.launchId,
      scope: scopeRecord.scope,
      owner: scopeRecord.owner,
      ticks: Object.freeze(ticks),
      corrections,
      driverDeathNoted: launch.driverDeathNoted,
      disposed: launch.disposed,
      unknown: this.intervalUnknown(launch),
      digest: digestInterval(launch.launchId, ticks, corrections, flags),
    });
  }

  // The durable correction journal: ordered, frozen, and readable after
  // disposal. Publication order is the journal order.
  correctionLog(owner: string, scope: string, launchId: string): readonly CorrectionFacts[] {
    const facts = this.intervalFacts(owner, scope, launchId);
    return facts.corrections;
  }

  // Seal the interval through confirmed disposal. Every tick must be
  // closed with no open gap: fully observed open ticks close implicitly,
  // but any gap entry — open or in an unclosed tick — refuses with
  // gap-open until corrected or published unknown. Unknown entries seal
  // honestly: the receipt carries unknown, and unknown never proves no-UI.
  sealDisposal(owner: string, scope: string, launchId: string, token: string): DisposalReceipt {
    const record = this.scopes.get(scope);
    if (record === undefined) {
      fail("unknown-scope", `no open scope for owner: ${owner}/${scope}`);
    }
    const scopeRecord = record as ScopeRecord;
    if (scopeRecord.owner !== owner) {
      fail("wrong-owner", "scope is owned by another identity");
    }
    if (scopeRecord.state === "closed") {
      fail("scope-closed", `scope is closed: ${owner}/${scope}`);
    }
    if (scopeRecord.state === "interrupted") {
      fail("observer-interrupted", `observer is interrupted for scope: ${owner}/${scope}`);
    }
    const launch = this.requireLaunch(scopeRecord, launchId);
    if (launch.disposed) {
      fail("already-disposed", `launch is already disposed: ${launchId}`);
    }
    if (token === "" || token !== launch.token) {
      fail("forged-token", "launch token is not the attested token");
    }
    // A lost observer seals unknown: every remaining gap becomes unknown
    // and the receipt says so.
    if (scopeRecord.state === "lost") {
      for (const tick of launch.ticks) {
        for (const channel of scopeRecord.channels) {
          const entry = tick.entries.get(channel);
          if (entry === undefined || entry.state === "gap") {
            tick.entries.set(channel, {
              state: "unknown",
              origin: "loss",
              corrected: entry?.corrected ?? false,
            });
          }
        }
        tick.closed = true;
      }
    } else {
      // Refusal precedes effect: every gap is checked before any tick
      // closes, so a refused seal mutates nothing.
      const open = launch.ticks[launch.ticks.length - 1] as TickRecord;
      if (!open.closed) {
        for (const channel of scopeRecord.channels) {
          if (!open.entries.has(channel)) {
            fail("gap-open", `tick ${open.tick} is open with unobserved channels`);
          }
        }
      }
      for (const tick of launch.ticks) {
        if (tick === open && !open.closed) continue;
        for (const entry of tick.entries.values()) {
          if (entry.state === "gap") {
            fail("gap-open", `tick ${tick.tick} holds an unresolved gap`);
          }
        }
      }
      open.closed = true;
    }
    launch.disposed = true;
    const receipt: DisposalReceipt = Object.freeze({
      launchId: launch.launchId,
      scope: scopeRecord.scope,
      owner: scopeRecord.owner,
      ticks: launch.ticks.length,
      corrections: launch.corrections.length,
      driverDeathNoted: launch.driverDeathNoted,
      unknown: this.intervalUnknown(launch),
      digest: digestText(
        `disposal:${launch.launchId}:${launch.ticks.length}:${launch.corrections.length}:${this.intervalFlags(launch)}`,
      ),
    });
    launch.disposal = receipt;
    return receipt;
  }

  disposalReceipt(owner: string, scope: string, launchId: string): DisposalReceipt {
    const record = this.scopes.get(scope);
    if (record === undefined) {
      fail("unknown-scope", `no open scope for owner: ${owner}/${scope}`);
    }
    const scopeRecord = record as ScopeRecord;
    if (scopeRecord.owner !== owner) {
      fail("wrong-owner", "scope is owned by another identity");
    }
    if (scopeRecord.state === "closed") {
      fail("scope-closed", `scope is closed: ${owner}/${scope}`);
    }
    const launch = this.requireLaunch(scopeRecord, launchId);
    if (!launch.disposed || launch.disposal === null) {
      fail("disposal-open", `launch interval is still open: ${launchId}`);
    }
    return launch.disposal as DisposalReceipt;
  }
}

// ---------------------------------------------------------------------------
// Admission adapter: the typed launch gate
// ---------------------------------------------------------------------------

// Admit one browser launch. Admission-first: the observer binding is
// checked before anything else, and a missing observer blocks launch with
// no-observer. Only a live binding attests.
export function admitLaunch(
  service: BrowserObserverService,
  owner: string,
  scope: string,
  launchId: string,
): LaunchAttestation {
  return service.attestLaunch(owner, scope, launchId);
}

function checkProofShape(value: unknown): NoUiProofClaim {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail("forbidden-proof", "no-UI proof must be an observer attestation object");
  }
  const record = value as Record<string, unknown>;
  if (record["kind"] === "observer-attestation") {
    const interval = record["interval"] as IntervalFacts;
    if (interval === null || typeof interval !== "object" || Array.isArray(interval)) {
      fail("forbidden-proof", "observer attestation carries no interval facts");
    }
    return { kind: "observer-attestation", interval };
  }
  if (typeof record["kind"] === "string") {
    fail("forbidden-proof", `no-UI proof kind is inadmissible: ${String(record["kind"])}`);
  }
  fail("forbidden-proof", "no-UI proof carries no kind");
}

// Judge one no-UI proof claim. Only a verified observer attestation over
// a fully observed interval with zero seen effects proves no-UI:
// headless flags, HOME values, launch flag sets, process samples, and any
// other non-observer claim reject with forbidden-proof before any verdict
// is read, and a forged or unknown interval never verifies.
export function assertNoUiProof(claim: unknown): NoUiVerdict {
  const proof = checkProofShape(claim);
  if (proof.kind !== "observer-attestation") {
    fail("forbidden-proof", `no-UI proof kind is inadmissible: ${proof.kind}`);
  }
  const interval = (proof as { kind: "observer-attestation"; interval: IntervalFacts }).interval;
  if (!Array.isArray(interval.ticks) || !Array.isArray(interval.corrections)) {
    fail("forbidden-proof", "observer attestation carries a malformed interval");
  }
  const flags = [
    `driver-death:${interval.driverDeathNoted ? 1 : 0}`,
    `disposed:${interval.disposed ? 1 : 0}`,
    `unknown:${interval.unknown ? 1 : 0}`,
  ].join("|");
  const recomputed = digestInterval(interval.launchId, interval.ticks, interval.corrections, flags);
  if (recomputed !== interval.digest) {
    fail("forbidden-proof", "observer attestation digest does not verify");
  }
  let uiSeen = false;
  for (const tick of interval.ticks) {
    if (!tick.closed) {
      fail("forbidden-proof", `attested interval holds an open tick: ${tick.tick}`);
    }
    for (const entry of tick.channels) {
      if (entry.state === "seen") uiSeen = true;
      if (entry.state === "gap") {
        fail("forbidden-proof", `attested interval holds an open gap: tick ${tick.tick}`);
      }
    }
  }
  return Object.freeze({
    launchId: interval.launchId,
    uiSeen,
    unknown: interval.unknown,
    digest: digestText(
      `verdict:${interval.launchId}:${uiSeen ? 1 : 0}:${interval.unknown ? 1 : 0}`,
    ),
  });
}
