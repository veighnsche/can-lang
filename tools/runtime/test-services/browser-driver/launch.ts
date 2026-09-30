// K08: N-owned pinned-driver browser launches with external reclaim witness.
//
// An N-owned launch service over in-memory doubles only: no browser, no
// process, no flags, no environment, no sampling, no I/O. The service
// launches through a pinned driver's launcher, captures the effective
// profile and child identities before any context is exposed, notes driver
// death as a fact, and seals reclaim only against an external witness: a
// receipt from the dead driver is never its own cleanup witness, and driver
// death mid-launch cannot orphan children. A missing qualified observer
// blocks launch: admission refuses without a live qualified binding.
//
// Independence scope from the launch/driver service under test (normative):
//   SHARES: nothing executable. The service never imports, calls, or reads
//   driver code, driver-decoded values, launch flags, environment
//   variables, or process state. The only strings that may coincide are
//   the driver/launch/child names under test (supplied by the test, not by
//   the driver) and the vocabulary this file uses for facts. Witness
//   tokens are verified by lookup against the injected external authority
//   and never parsed.
//   PROVES: pinned-launcher admission, launch identity capture before
//   context exposure, driver-death noting, orphan accounting, externally
//   witnessed reclaim, and error provenance (every failure names its
//   layer). Driver claims are compared AGAINST these facts; the service
//   never derives a cleanup fact FROM a driver claim.
//   Consequently a driver receipt can never prove cleanup: only an
//   external witness over every captured child proves it.
//
// Split with the Go mirror
// (tools/native-test-owner/external/browser_local.go):
//   TS owns the typed service mechanics, the witness-verifier seam, and
//   the bounded self-check. The Go mirror owns the same contract outside
//   the service under independent N ownership: pinned-driver admission,
//   child discovery/containment/reclaim, and witness minting. Both enforce
//   the same contract: pinned launchers, opaque tokens verified by lookup,
//   digest-only frozen facts, and layered errors. Neither launches a
//   browser.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, browsers, processes, or live runtimes. Local
// controls only. Actual launch, host effects, and service-death controls
// are QB0.

import { createHash, randomBytes } from "node:crypto";

export const BROWSER_LAUNCH_SCHEMA_VERSION = "1" as const;

// ---------------------------------------------------------------------------
// Independence scope (normative, mirrored in browser_local.go)
// ---------------------------------------------------------------------------

export const BROWSER_LAUNCH_INDEPENDENCE_SCOPE: string =
  "independent: shares no driver code, decoded values, receipts, flags, " +
  "environment, or process state; proves pinned-launcher admission, launch " +
  "identity capture, orphan accounting, externally witnessed reclaim, and " +
  "layered error provenance from its own seeded doubles alone";

// ---------------------------------------------------------------------------
// Layers, codes, and the layered error
// ---------------------------------------------------------------------------

export const BROWSER_LAUNCH_LAYERS = ["service", "launch", "reclaim"] as const;
export type BrowserLaunchLayer = (typeof BROWSER_LAUNCH_LAYERS)[number];

// Closed failure-code vocabulary. Every code maps to exactly one layer:
//   service: pinned drivers, observer gate, ownership, capacity.
//   launch: launch binding, tokens, death, identity/exposure ordering.
//   reclaim: witness admission, children, orphans, and the reclaim seal.
export const BROWSER_LAUNCH_CODES = [
  "unknown-driver",
  "launcher-unpinned",
  "no-observer",
  "wrong-owner",
  "capacity-exhausted",
  "forged-token",
  "unknown-launch",
  "launch-closed",
  "driver-dead",
  "identity-open",
  "forbidden-witness",
  "forged-witness",
  "unknown-child",
  "orphan-open",
  "already-reclaimed",
] as const;
export type BrowserLaunchCode = (typeof BROWSER_LAUNCH_CODES)[number];

const CODE_LAYER: Readonly<Record<BrowserLaunchCode, BrowserLaunchLayer>> = {
  "unknown-driver": "service",
  "launcher-unpinned": "service",
  "no-observer": "service",
  "wrong-owner": "service",
  "capacity-exhausted": "service",
  "forged-token": "launch",
  "unknown-launch": "launch",
  "launch-closed": "launch",
  "driver-dead": "launch",
  "identity-open": "launch",
  "forbidden-witness": "reclaim",
  "forged-witness": "reclaim",
  "unknown-child": "reclaim",
  "orphan-open": "reclaim",
  "already-reclaimed": "reclaim",
};

export function layerOfLaunchCode(code: BrowserLaunchCode): BrowserLaunchLayer {
  return CODE_LAYER[code];
}

export class BrowserLaunchError extends Error {
  readonly layer: BrowserLaunchLayer;
  readonly code: BrowserLaunchCode;
  constructor(code: BrowserLaunchCode, message: string) {
    super(`[${CODE_LAYER[code]}:${code}] ${message}`);
    this.name = "BrowserLaunchError";
    this.layer = CODE_LAYER[code];
    this.code = code;
  }
}

function fail(code: BrowserLaunchCode, message: string): never {
  throw new BrowserLaunchError(code, message);
}

// ---------------------------------------------------------------------------
// Bounds and names
// ---------------------------------------------------------------------------

export type BrowserLaunchLimits = Readonly<{
  maxDrivers: number;
  maxObservers: number;
  maxLaunches: number;
  maxChildrenPerLaunch: number;
}>;

const LIMIT_KEYS = ["maxDrivers", "maxObservers", "maxLaunches", "maxChildrenPerLaunch"] as const;

export function checkLaunchLimits(value: unknown): BrowserLaunchLimits {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new BrowserLaunchError("unknown-driver", "limits must be an object");
  }
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    if (!(LIMIT_KEYS as readonly string[]).includes(key)) {
      throw new BrowserLaunchError("unknown-driver", `unknown limit field: ${key}`);
    }
  }
  const out: Record<string, number> = {};
  for (const key of LIMIT_KEYS) {
    const entry = record[key];
    if (!Number.isSafeInteger(entry) || (entry as number) <= 0) {
      throw new BrowserLaunchError("unknown-driver", `${key} must be a positive integer`);
    }
    out[key] = entry as number;
  }
  return Object.freeze(out) as BrowserLaunchLimits;
}

const MAX_NAME_LEN = 128;

function checkName(value: unknown, what: string, code: BrowserLaunchCode): string {
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

function checkDriverName(value: unknown): string {
  return checkName(value, "driver", "unknown-driver");
}

function checkLauncherName(value: unknown): string {
  return checkName(value, "launcher", "unknown-driver");
}

function checkProfileName(value: unknown): string {
  return checkName(value, "profile", "unknown-driver");
}

function checkScopeName(value: unknown): string {
  return checkName(value, "scope", "no-observer");
}

function checkLaunchName(value: unknown): string {
  return checkName(value, "launch", "unknown-launch");
}

function checkChildName(value: unknown): string {
  return checkName(value, "child", "unknown-child");
}

// ---------------------------------------------------------------------------
// Facts (frozen, digest-only) and the external witness seam
// ---------------------------------------------------------------------------

export type ObserverBindingFacts = Readonly<{
  owner: string;
  scope: string;
  handleDigest: string;
}>;

export type LaunchAttestation = Readonly<{
  launchId: string;
  scope: string;
  owner: string;
  driver: string;
  launcher: string;
  requestedProfile: string;
  observerDigest: string;
  handleDigest: string;
}>;

export type LaunchIdentityFacts = Readonly<{
  launchId: string;
  owner: string;
  driver: string;
  launcher: string;
  requestedProfile: string;
  effectiveProfile: string;
  children: readonly string[];
  handleDigest: string;
  digest: string;
}>;

export type ContextGrantFacts = Readonly<{
  launchId: string;
  owner: string;
  driver: string;
  launcher: string;
  profile: string;
  children: readonly string[];
  handleDigest: string;
  digest: string;
}>;

export type LaunchFacts = Readonly<{
  launchId: string;
  owner: string;
  driver: string;
  launcher: string;
  requestedProfile: string;
  effectiveProfile: string | null;
  exposed: boolean;
  driverDeathNoted: boolean;
  reclaimed: boolean;
  capturedChildren: readonly string[];
  reclaimedChildren: readonly string[];
  digest: string;
}>;

// Externally witnessed reclaim facts, returned by the injected authority.
// The service shape-checks every field: a lying verifier fails lookup.
export type WitnessFacts = Readonly<{
  launchId: string;
  children: readonly string[];
  contained: boolean;
  digest: string;
}>;

// The outside-service N child-discovery/containment/reclaim authority. The
// service holds it as an opaque verifier: witness tokens are verified by
// lookup through this seam and never parsed. Local controls inject a
// seeded double; QB0 wires the qualified authority.
export type ExternalWitness = {
  readonly authority: string;
  verifyWitness(launchId: string, token: string): WitnessFacts;
};

export type ReclaimReceipt = Readonly<{
  launchId: string;
  owner: string;
  driver: string;
  launcher: string;
  profile: string | null;
  children: readonly string[];
  driverDeathNoted: boolean;
  digest: string;
}>;

// A cleanup proof claim. Only external-witness is admissible; a driver
// receipt and every other kind reject with forbidden-witness before any
// verdict is read.
export const FORBIDDEN_WITNESS_KINDS = ["driver-receipt", "launcher-log", "exit-code"] as const;
export type ForbiddenWitnessKind = (typeof FORBIDDEN_WITNESS_KINDS)[number];

export type CleanupProofClaim =
  | Readonly<{ kind: "external-witness"; receipt: ReclaimReceipt }>
  | Readonly<{ kind: ForbiddenWitnessKind; detail: string }>;

export type CleanupVerdict = Readonly<{
  launchId: string;
  witnessed: boolean;
  children: readonly string[];
  digest: string;
}>;

type LaunchRecord = {
  launchId: string;
  scope: string;
  owner: string;
  driver: string;
  launcher: string;
  requestedProfile: string;
  effectiveProfile: string | null;
  children: string[];
  token: string;
  exposed: boolean;
  contextToken: string | null;
  driverDeathNoted: boolean;
  reclaimed: boolean;
  reclaimedChildren: string[];
  receipt: ReclaimReceipt | null;
};

type BindingState = "open" | "lost";

type BindingRecord = {
  scope: string;
  owner: string;
  token: string;
  state: BindingState;
};

function mintToken(prefix: string): string {
  return `${prefix}-${randomBytes(16).toString("hex")}`;
}

function digestText(text: string): string {
  return `sha256:${createHash("sha256").update(text, "utf8").digest("hex")}`;
}

function sortedUnique(names: readonly string[]): string[] {
  return [...new Set(names)].sort();
}

// The reclaim digest is recomputed from carried fields, so
// assertCleanupWitnessed re-verifies a presented receipt instead of
// trusting its digest string.
export function digestReclaim(
  launchId: string,
  owner: string,
  driver: string,
  launcher: string,
  profile: string | null,
  children: readonly string[],
  driverDeathNoted: boolean,
): string {
  const hash = createHash("sha256");
  hash.update(launchId, "utf8");
  hash.update("\0", "utf8");
  hash.update(owner, "utf8");
  hash.update("\0", "utf8");
  hash.update(driver, "utf8");
  hash.update("\0", "utf8");
  hash.update(launcher, "utf8");
  hash.update("\0", "utf8");
  hash.update(profile ?? "", "utf8");
  hash.update("\0", "utf8");
  for (const child of sortedUnique(children)) {
    hash.update(child, "utf8");
    hash.update(";", "utf8");
  }
  hash.update("\0", "utf8");
  hash.update(driverDeathNoted ? "dead" : "live", "utf8");
  return `sha256:${hash.digest("hex")}`;
}

// ---------------------------------------------------------------------------
// Service: pinned launches, identity capture, externally witnessed reclaim
// ---------------------------------------------------------------------------

export type DriverGrant = Readonly<{
  owner: string;
  driver: string;
  launcher: string;
}>;

export type ObserverGrant = Readonly<{
  owner: string;
  scope: string;
}>;

export class BrowserLaunchService {
  private readonly limits: BrowserLaunchLimits;
  private readonly drivers = new Map<string, DriverGrant>();
  private readonly observers = new Map<string, ObserverGrant>();
  private readonly bindings = new Map<string, BindingRecord>();
  private readonly launches = new Map<string, LaunchRecord>();
  private readonly witness: ExternalWitness;

  constructor(
    drivers: readonly DriverGrant[],
    observers: readonly ObserverGrant[],
    limits: BrowserLaunchLimits,
    witness: ExternalWitness,
  ) {
    if (
      witness === null ||
      typeof witness !== "object" ||
      typeof witness.verifyWitness !== "function"
    ) {
      throw new BrowserLaunchError(
        "forbidden-witness",
        "an external witness authority is required",
      );
    }
    if (drivers.length === 0) {
      throw new BrowserLaunchError("unknown-driver", "declare at least one pinned driver");
    }
    if (drivers.length > limits.maxDrivers) {
      throw new BrowserLaunchError("capacity-exhausted", "declared drivers exceed the cap");
    }
    for (const grant of drivers) {
      checkOwner(grant.owner);
      checkDriverName(grant.driver);
      checkLauncherName(grant.launcher);
      const key = `${grant.owner}\0${grant.driver}`;
      if (this.drivers.has(key)) {
        fail("unknown-driver", `duplicate pinned driver: ${grant.owner}/${grant.driver}`);
      }
      this.drivers.set(
        key,
        Object.freeze({ owner: grant.owner, driver: grant.driver, launcher: grant.launcher }),
      );
    }
    if (observers.length === 0) {
      throw new BrowserLaunchError("no-observer", "declare at least one qualified observer scope");
    }
    if (observers.length > limits.maxObservers) {
      throw new BrowserLaunchError("capacity-exhausted", "declared observers exceed the cap");
    }
    for (const grant of observers) {
      checkOwner(grant.owner);
      checkScopeName(grant.scope);
      const key = `${grant.owner}\0${grant.scope}`;
      if (this.observers.has(key)) {
        fail("no-observer", `duplicate qualified observer scope: ${grant.owner}/${grant.scope}`);
      }
      this.observers.set(key, Object.freeze({ owner: grant.owner, scope: grant.scope }));
    }
    this.limits = limits;
    this.witness = witness;
  }

  get launchCount(): number {
    return this.launches.size;
  }

  // Bind one qualified observer scope. Only an exact declared (owner,
  // scope) grant admits; anything else refuses before any effect.
  // Bindings hold one owner: a second identity joining the same scope
  // name fails the owner check.
  bindObserver(owner: string, scope: string): ObserverBindingFacts {
    checkOwner(owner);
    checkScopeName(scope);
    const grant = this.observers.get(`${owner}\0${scope}`);
    if (grant === undefined) {
      fail("no-observer", `no qualified grant for observer scope: ${owner}/${scope}`);
    }
    const prior = this.bindings.get(scope);
    if (prior !== undefined) {
      if (prior.owner !== owner) {
        fail("wrong-owner", `observer scope is owned by another identity: ${scope}`);
      }
      return this.bindingFactsOf(prior);
    }
    if (this.bindings.size >= this.limits.maxObservers) {
      fail("capacity-exhausted", "observer binding table full");
    }
    const record: BindingRecord = { scope, owner, token: mintToken("obs"), state: "open" };
    this.bindings.set(scope, record);
    return this.bindingFactsOf(record);
  }

  private bindingFactsOf(record: BindingRecord): ObserverBindingFacts {
    return Object.freeze({
      owner: record.owner,
      scope: record.scope,
      handleDigest: digestText(record.token),
    });
  }

  // Lose the observer permanently: later launches refuse with
  // no-observer, while in-flight launch facts stay readable.
  loseObserver(owner: string, scope: string): void {
    checkOwner(owner);
    checkScopeName(scope);
    const record = this.bindings.get(scope);
    if (record === undefined || record.owner !== owner) {
      fail("no-observer", `no live observer binding to lose: ${owner}/${scope}`);
    }
    (record as BindingRecord).state = "lost";
  }

  // Admission-first: the launch gate reports only whether a LIVE
  // qualified binding exists. Never bound, lost, or foreign all refuse
  // with no-observer: a missing qualified observer blocks launch,
  // whatever the reason.
  private requireLiveBinding(owner: string, scope: string): BindingRecord {
    checkOwner(owner);
    checkScopeName(scope);
    const record = this.bindings.get(scope);
    if (record === undefined || record.owner !== owner || record.state !== "open") {
      fail("no-observer", `no live qualified observer binding for launch: ${owner}/${scope}`);
    }
    return record as BindingRecord;
  }

  // Launch through the pinned driver's launcher. The observer binding is
  // checked before anything else; the presented launcher must equal the
  // pinned launcher exactly.
  launch(
    owner: string,
    scope: string,
    driver: string,
    launcher: string,
    launchId: string,
    requestedProfile: string,
  ): LaunchAttestation {
    const binding = this.requireLiveBinding(owner, scope);
    checkLaunchName(launchId);
    checkDriverName(driver);
    checkLauncherName(launcher);
    checkProfileName(requestedProfile);
    const grant = this.drivers.get(`${owner}\0${driver}`);
    if (grant === undefined) {
      fail("unknown-driver", `no pinned launcher grant for driver: ${owner}/${driver}`);
    }
    if ((grant as DriverGrant).launcher !== launcher) {
      fail("launcher-unpinned", `launcher is not the pinned launcher for driver: ${driver}`);
    }
    if (this.launches.size >= this.limits.maxLaunches) {
      fail("capacity-exhausted", "launch table full");
    }
    const prior = this.launches.get(launchId);
    if (prior !== undefined) {
      if (prior.reclaimed) {
        fail("launch-closed", `launch id is single-use and already reclaimed: ${launchId}`);
      }
      // Joining re-attests the identical launch only: a foreign owner or
      // differing arguments refuse rather than misattributing the launch.
      if (prior.owner !== owner) {
        fail("wrong-owner", `launch id is owned by another identity: ${launchId}`);
      }
      if (
        prior.scope !== scope ||
        prior.driver !== driver ||
        prior.launcher !== launcher ||
        prior.requestedProfile !== requestedProfile
      ) {
        fail("unknown-launch", `launch id is already launched: ${launchId}`);
      }
      return this.attestationOf(binding, prior);
    }
    const record: LaunchRecord = {
      launchId,
      scope,
      owner,
      driver,
      launcher,
      requestedProfile,
      effectiveProfile: null,
      children: [],
      token: mintToken("lnch"),
      exposed: false,
      contextToken: null,
      driverDeathNoted: false,
      reclaimed: false,
      reclaimedChildren: [],
      receipt: null,
    };
    this.launches.set(launchId, record);
    return this.attestationOf(binding, record);
  }

  private attestationOf(binding: BindingRecord, launch: LaunchRecord): LaunchAttestation {
    return Object.freeze({
      launchId: launch.launchId,
      scope: binding.scope,
      owner: launch.owner,
      driver: launch.driver,
      launcher: launch.launcher,
      requestedProfile: launch.requestedProfile,
      observerDigest: digestText(binding.token),
      handleDigest: digestText(launch.token),
    });
  }

  attestation(owner: string, driver: string, launchId: string): LaunchAttestation {
    const launch = this.requireLaunch(owner, driver, launchId);
    const binding = this.bindings.get(launch.scope) as BindingRecord;
    return this.attestationOf(binding, launch);
  }

  private requireLaunch(owner: string, driver: string, launchId: string): LaunchRecord {
    const launch = this.launches.get(launchId);
    if (launch === undefined) {
      fail("unknown-launch", `no pinned launch: ${launchId}`);
    }
    const record = launch as LaunchRecord;
    if (record.owner !== owner) {
      fail("wrong-owner", "launch is owned by another identity");
    }
    if (record.driver !== driver) {
      fail("unknown-driver", "launch belongs to another pinned driver");
    }
    return record;
  }

  // Verify the launch token by table lookup. Unknown launches and
  // invented tokens are never authority; reclaimed launches report
  // launch-closed.
  private requireLiveLaunch(
    owner: string,
    driver: string,
    launchId: string,
    token: string,
  ): LaunchRecord {
    const launch = this.requireLaunch(owner, driver, launchId);
    if (launch.reclaimed) {
      fail("launch-closed", `launch is reclaimed: ${launchId}`);
    }
    if (token === "" || token !== launch.token) {
      fail("forged-token", "launch token is not the attested token");
    }
    return launch;
  }

  // Test-only accessors: raw tokens cross exactly here so bounded
  // controls can present them on later calls. Facts carry digests only.
  launchTokenForTest(owner: string, driver: string, launchId: string): string {
    return this.requireLaunch(owner, driver, launchId).token;
  }

  contextTokenForTest(owner: string, driver: string, launchId: string): string {
    const launch = this.requireLaunch(owner, driver, launchId);
    if (!launch.exposed || launch.contextToken === null) {
      fail("identity-open", "context is not exposed: capture identity and expose first");
    }
    return launch.contextToken as string;
  }

  // Capture the effective profile and child identities the pinned
  // launcher reports. The capture window is pre-exposure: later captures
  // refine the record, and exposure freezes it. A dead launcher reports
  // nothing further.
  captureIdentity(
    owner: string,
    driver: string,
    launchId: string,
    token: string,
    effectiveProfile: string,
    childIds: readonly unknown[],
  ): LaunchIdentityFacts {
    checkProfileName(effectiveProfile);
    if (!Array.isArray(childIds)) {
      fail("unknown-child", "child identities must be an array");
    }
    const children: string[] = [];
    for (const child of childIds) {
      children.push(checkChildName(child));
    }
    if (new Set(children).size !== children.length) {
      fail("unknown-child", "child identities repeat a child");
    }
    if (children.length > this.limits.maxChildrenPerLaunch) {
      fail("capacity-exhausted", "child table full for launch");
    }
    const launch = this.requireLiveLaunch(owner, driver, launchId, token);
    if (launch.driverDeathNoted) {
      fail("driver-dead", `launcher is dead for launch: ${launchId}`);
    }
    if (launch.exposed) {
      fail("identity-open", "launch identity froze at context exposure");
    }
    launch.effectiveProfile = effectiveProfile;
    launch.children = sortedUnique(children);
    return Object.freeze({
      launchId: launch.launchId,
      owner: launch.owner,
      driver: launch.driver,
      launcher: launch.launcher,
      requestedProfile: launch.requestedProfile,
      effectiveProfile,
      children: Object.freeze([...launch.children]),
      handleDigest: digestText(launch.token),
      digest: digestText(
        `identity:${launch.launchId}:${launch.driver}:${effectiveProfile}:${launch.children.join(",")}`,
      ),
    });
  }

  // Expose the launch context. Identity first: exposure without a
  // captured identity refuses, and a dead launcher exposes nothing.
  // Exposing twice joins the same grant.
  exposeContext(owner: string, driver: string, launchId: string, token: string): ContextGrantFacts {
    const launch = this.requireLiveLaunch(owner, driver, launchId, token);
    if (launch.driverDeathNoted) {
      fail("driver-dead", `launcher is dead for launch: ${launchId}`);
    }
    if (launch.exposed && launch.contextToken !== null) {
      return this.contextGrantOf(launch);
    }
    if (launch.effectiveProfile === null) {
      fail("identity-open", "capture the launch identity before context exposure");
    }
    launch.contextToken = mintToken("ctx");
    launch.exposed = true;
    return this.contextGrantOf(launch);
  }

  private contextGrantOf(launch: LaunchRecord): ContextGrantFacts {
    return Object.freeze({
      launchId: launch.launchId,
      owner: launch.owner,
      driver: launch.driver,
      launcher: launch.launcher,
      profile: launch.effectiveProfile as string,
      children: Object.freeze([...launch.children]),
      handleDigest: digestText(launch.contextToken as string),
      digest: digestText(
        `context:${launch.launchId}:${launch.effectiveProfile}:${launch.children.join(",")}`,
      ),
    });
  }

  // Note the driver's death. The launch continues under external
  // ownership: capture and exposure stop, while discovery, containment,
  // and reclaim proceed outside the dead driver.
  noteDriverDeath(owner: string, driver: string, launchId: string, token: string): void {
    const launch = this.requireLiveLaunch(owner, driver, launchId, token);
    launch.driverDeathNoted = true;
  }

  // Launch facts stay readable after death and after reclaim: the launch
  // is evidence, and neither death nor reclaim hides it.
  launchFacts(owner: string, driver: string, launchId: string): LaunchFacts {
    const launch = this.requireLaunch(owner, driver, launchId);
    return Object.freeze({
      launchId: launch.launchId,
      owner: launch.owner,
      driver: launch.driver,
      launcher: launch.launcher,
      requestedProfile: launch.requestedProfile,
      effectiveProfile: launch.effectiveProfile,
      exposed: launch.exposed,
      driverDeathNoted: launch.driverDeathNoted,
      reclaimed: launch.reclaimed,
      capturedChildren: Object.freeze([...launch.children]),
      reclaimedChildren: Object.freeze([...launch.reclaimedChildren]),
      digest: digestText(
        `launch:${launch.launchId}:${launch.effectiveProfile ?? ""}:${launch.children.join(",")}:` +
          `${launch.reclaimedChildren.join(",")}:${launch.driverDeathNoted ? 1 : 0}:` +
          `${launch.reclaimed ? 1 : 0}`,
      ),
    });
  }

  private checkWitnessShape(value: unknown): WitnessFacts {
    if (value === null || typeof value !== "object" || Array.isArray(value)) {
      fail("forged-witness", "witness facts are malformed");
    }
    const record = value as Record<string, unknown>;
    if (typeof record["launchId"] !== "string" || record["launchId"] === "") {
      fail("forged-witness", "witness names no launch");
    }
    if (!Array.isArray(record["children"])) {
      fail("forged-witness", "witness names no child set");
    }
    for (const child of record["children"] as unknown[]) {
      if (typeof child !== "string" || child === "") {
        fail("forged-witness", "witness names a malformed child");
      }
    }
    if (typeof record["contained"] !== "boolean") {
      fail("forged-witness", "witness states no containment");
    }
    if (typeof record["digest"] !== "string" || record["digest"] === "") {
      fail("forged-witness", "witness carries no digest");
    }
    return {
      launchId: record["launchId"] as string,
      children: [...(record["children"] as string[])],
      contained: record["contained"] as boolean,
      digest: record["digest"] as string,
    };
  }

  // Seal reclaim against the external witness. The witness token is
  // verified by lookup through the injected authority; every captured
  // child must appear in the witnessed set or the seal names the
  // orphans. A driver receipt is never verification: only the external
  // authority's lookup witnesses cleanup.
  sealReclaim(
    owner: string,
    driver: string,
    launchId: string,
    token: string,
    witnessToken: string,
  ): ReclaimReceipt {
    const launch = this.requireLaunch(owner, driver, launchId);
    if (launch.reclaimed) {
      fail("already-reclaimed", `launch is already reclaimed: ${launchId}`);
    }
    if (token === "" || token !== launch.token) {
      fail("forged-token", "launch token is not the attested token");
    }
    if (typeof witnessToken !== "string" || witnessToken === "") {
      fail("forged-witness", "witness token is not an attested token");
    }
    let facts: WitnessFacts;
    try {
      facts = this.checkWitnessShape(this.witness.verifyWitness(launchId, witnessToken));
    } catch (error) {
      if (error instanceof BrowserLaunchError) throw error;
      fail("forged-witness", "external witness refused the token");
    }
    const witnessed = facts as WitnessFacts;
    if (witnessed.launchId !== launchId) {
      fail("forged-witness", "witness is for another launch");
    }
    if (!witnessed.contained) {
      fail("orphan-open", "witness reports uncontained children");
    }
    const covered = new Set(witnessed.children);
    const missing = launch.children.filter((child) => !covered.has(child));
    if (missing.length > 0) {
      fail("orphan-open", `children lack external reclaim: ${missing.join(",")}`);
    }
    launch.reclaimedChildren = sortedUnique(witnessed.children);
    launch.reclaimed = true;
    const receipt: ReclaimReceipt = Object.freeze({
      launchId: launch.launchId,
      owner: launch.owner,
      driver: launch.driver,
      launcher: launch.launcher,
      profile: launch.effectiveProfile,
      children: Object.freeze([...launch.reclaimedChildren]),
      driverDeathNoted: launch.driverDeathNoted,
      digest: digestReclaim(
        launch.launchId,
        launch.owner,
        launch.driver,
        launch.launcher,
        launch.effectiveProfile,
        launch.reclaimedChildren,
        launch.driverDeathNoted,
      ),
    });
    launch.receipt = receipt;
    return receipt;
  }

  reclaimReceipt(owner: string, driver: string, launchId: string): ReclaimReceipt {
    const launch = this.requireLaunch(owner, driver, launchId);
    if (!launch.reclaimed || launch.receipt === null) {
      fail("orphan-open", `launch reclaim is still open: ${launchId}`);
    }
    return launch.receipt as ReclaimReceipt;
  }
}

// ---------------------------------------------------------------------------
// Cleanup verdict: only the external witness judges
// ---------------------------------------------------------------------------

function checkCleanupProofShape(value: unknown): CleanupProofClaim {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail("forbidden-witness", "cleanup proof must be an external witness object");
  }
  const record = value as Record<string, unknown>;
  if (record["kind"] === "external-witness") {
    const receipt = record["receipt"] as ReclaimReceipt;
    if (receipt === null || typeof receipt !== "object" || Array.isArray(receipt)) {
      fail("forbidden-witness", "external witness carries no reclaim receipt");
    }
    return { kind: "external-witness", receipt };
  }
  if (typeof record["kind"] === "string") {
    fail("forbidden-witness", `cleanup proof kind is inadmissible: ${String(record["kind"])}`);
  }
  fail("forbidden-witness", "cleanup proof carries no kind");
}

// Judge one cleanup proof claim. Only a verified external-witness receipt
// proves cleanup: driver receipts, launcher logs, exit codes, and any
// other non-witness claim reject with forbidden-witness before any verdict
// is read, and a forged receipt never verifies.
export function assertCleanupWitnessed(claim: unknown): CleanupVerdict {
  const proof = checkCleanupProofShape(claim);
  if (proof.kind !== "external-witness") {
    fail("forbidden-witness", `cleanup proof kind is inadmissible: ${proof.kind}`);
  }
  const receipt = (proof as { kind: "external-witness"; receipt: ReclaimReceipt }).receipt;
  if (
    typeof receipt.launchId !== "string" ||
    typeof receipt.owner !== "string" ||
    typeof receipt.driver !== "string" ||
    typeof receipt.launcher !== "string" ||
    (typeof receipt.profile !== "string" && receipt.profile !== null) ||
    !Array.isArray(receipt.children) ||
    typeof receipt.driverDeathNoted !== "boolean" ||
    typeof receipt.digest !== "string"
  ) {
    fail("forbidden-witness", "external witness carries a malformed receipt");
  }
  for (const child of receipt.children) {
    if (typeof child !== "string") {
      fail("forbidden-witness", "external witness carries a malformed receipt");
    }
  }
  const recomputed = digestReclaim(
    receipt.launchId,
    receipt.owner,
    receipt.driver,
    receipt.launcher,
    receipt.profile,
    receipt.children,
    receipt.driverDeathNoted,
  );
  if (recomputed !== receipt.digest) {
    fail("forbidden-witness", "witness receipt digest does not verify");
  }
  return Object.freeze({
    launchId: receipt.launchId,
    witnessed: true,
    children: Object.freeze([...receipt.children]),
    digest: digestText(`verdict:${receipt.launchId}:${receipt.children.join(",")}`),
  });
}
