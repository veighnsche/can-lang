// K07 bounded self-check: independent browser host-effect observation.
//
// Run with: bun tools/runtime/test-services/browser-driver/admission-check.ts
// Local controls only. No browsers, processes, flags, environment, sampling,
// networks, files, timers, live hosts, or live runtimes. Every loop below
// is bounded by a small constant. Live host-dependent controls wait for the
// qualified profile and Q task.

import { strict as assert } from "node:assert";
import {
  admitLaunch,
  assertNoUiProof,
  BROWSER_ADMISSION_SCHEMA_VERSION,
  BROWSER_OBSERVER_CODES,
  BROWSER_OBSERVER_INDEPENDENCE_SCOPE,
  BrowserObserverError,
  BrowserObserverService,
  checkLimits,
  EFFECT_CHANNELS,
  FORBIDDEN_PROOF_KINDS,
  layerOfCode,
  type BrowserObserverCode,
  type BrowserObserverLayer,
  type BrowserObserverLimits,
} from "./admission.ts";

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectObserverError(
  body: () => unknown,
  code: BrowserObserverCode,
  layer: BrowserObserverLayer,
): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof BrowserObserverError)) {
      assert.fail(`expected a BrowserObserverError, got ${String(error)}`);
    }
    assert.equal(error.code, code);
    assert.equal(error.layer, layer);
    assert.ok(error.message.includes(`[${layer}:${code}]`));
    return;
  }
  assert.fail(`expected failure ${layer}:${code}`);
}

function roomyLimits(): BrowserObserverLimits {
  return checkLimits({
    maxScopes: 4,
    maxLaunches: 8,
    maxTicksPerLaunch: 8,
    maxCorrectionsPerLaunch: 16,
  });
}

function openService(): BrowserObserverService {
  const service = new BrowserObserverService(
    [
      { owner: "owner-n", scope: "host-ui", channels: ["window", "icon"] },
      { owner: "owner-n", scope: "host-ui-full", channels: [...EFFECT_CHANNELS] },
    ],
    roomyLimits(),
  );
  service.openScope("owner-n", "host-ui");
  return service;
}

function openLaunch(): { service: BrowserObserverService; token: string } {
  const service = openService();
  admitLaunch(service, "owner-n", "host-ui", "launch-1");
  const token = service.launchTokenForTest("owner-n", "host-ui", "launch-1");
  return { service, token };
}

function observeFull(
  service: BrowserObserverService,
  token: string,
  launch = "launch-1",
  state = "absent",
): void {
  for (const channel of ["window", "icon"]) {
    service.observe("owner-n", "host-ui", launch, token, channel, state);
  }
}

// --- Admission: the scope grant is the only authority -----------------------

check("scope-receipt-binds-owned-grant", () => {
  const service = new BrowserObserverService(
    [{ owner: "owner-n", scope: "host-ui", channels: ["window", "icon"] }],
    roomyLimits(),
  );
  const receipt = service.openScope("owner-n", "host-ui");
  assert.equal(receipt.scope, "host-ui");
  assert.equal(receipt.owner, "owner-n");
  assert.deepEqual([...receipt.channels], ["window", "icon"]);
  assert.ok(receipt.handleDigest.startsWith("sha256:"));
  assert.equal(receipt.handleDigest.length, "sha256:".length + 64);
  assert.equal(receipt.launches, 0);
  // The receipt carries a digest only: the raw token never crosses.
  assert.deepEqual(Object.keys(receipt).sort(), [
    "channels",
    "handleDigest",
    "launches",
    "owner",
    "scope",
  ]);

  // Reopening joins the same owned scope; the digest is stable.
  const again = service.openScope("owner-n", "host-ui");
  assert.equal(again.handleDigest, receipt.handleDigest);

  // Scopes outside the owned grant reject before any effect.
  for (const foreign of ["host-ui-2", "host", "", "host-ui/full"]) {
    expectObserverError(() => service.openScope("owner-n", foreign), "unknown-scope", "observer");
  }
  // A different owner holds a different grant: no joining across owners.
  expectObserverError(() => service.openScope("owner-x", "host-ui"), "unknown-scope", "observer");
  // Empty and over-full declarations never construct.
  expectObserverError(
    () => new BrowserObserverService([{ owner: "o", scope: "s", channels: [] }], roomyLimits()),
    "unknown-scope",
    "observer",
  );
  expectObserverError(
    () =>
      new BrowserObserverService(
        [{ owner: "o", scope: "s", channels: ["window", "window"] }],
        roomyLimits(),
      ),
    "unknown-scope",
    "observer",
  );
});

// --- Acceptance: a missing observer blocks launch ----------------------------

check("missing-observer-blocks-launch", () => {
  const service = new BrowserObserverService(
    [{ owner: "owner-n", scope: "host-ui", channels: ["window"] }],
    roomyLimits(),
  );
  // Never opened: admission refuses without a live observer binding.
  expectObserverError(
    () => admitLaunch(service, "owner-n", "host-ui", "launch-1"),
    "no-observer",
    "admission",
  );
  // Ungranted scope: admission refuses the same way.
  expectObserverError(
    () => admitLaunch(service, "owner-n", "elsewhere", "launch-1"),
    "no-observer",
    "admission",
  );
  // Foreign owner: admission refuses the same way.
  service.openScope("owner-n", "host-ui");
  expectObserverError(
    () => admitLaunch(service, "owner-x", "host-ui", "launch-1"),
    "no-observer",
    "admission",
  );
  // Live binding: the same launch admits and attests.
  const attestation = admitLaunch(service, "owner-n", "host-ui", "launch-1");
  assert.equal(attestation.launchId, "launch-1");
  assert.equal(attestation.scope, "host-ui");
  assert.equal(attestation.owner, "owner-n");
  assert.ok(attestation.handleDigest.startsWith("sha256:"));
  assert.ok(attestation.observerDigest.startsWith("sha256:"));
  assert.ok(attestation.scopeDigest.startsWith("sha256:"));
  assert.equal(attestation.tick, 0);
  assert.deepEqual(Object.keys(attestation).sort(), [
    "handleDigest",
    "launchId",
    "observerDigest",
    "owner",
    "scope",
    "scopeDigest",
    "tick",
  ]);
  // Joining a live launch re-attests; the digest is stable.
  const joined = admitLaunch(service, "owner-n", "host-ui", "launch-1");
  assert.equal(joined.handleDigest, attestation.handleDigest);
});

// --- Acceptance: only the observer's own observations prove no-UI -----------

check("forbidden-proofs-rejected", () => {
  // Every forbidden kind rejects before any verdict is read.
  assert.deepEqual(
    [...FORBIDDEN_PROOF_KINDS],
    ["headless-flag", "home-env", "launch-flags", "process-sample"],
  );
  for (const kind of FORBIDDEN_PROOF_KINDS) {
    expectObserverError(
      () => assertNoUiProof({ kind, detail: `--headless at seed` }),
      "forbidden-proof",
      "admission",
    );
  }
  // Unknown kinds and malformed claims reject the same way.
  for (const claim of [
    { kind: "driver-receipt", detail: "driver says clean" },
    { kind: "screenshot-hash", detail: "abc" },
    { kind: 7, detail: "x" },
    { detail: "no kind" },
    null,
    "observer-attestation",
  ]) {
    expectObserverError(() => assertNoUiProof(claim), "forbidden-proof", "admission");
  }
  // A forged attestation digest never verifies.
  const { service, token } = openLaunch();
  observeFull(service, token);
  service.endTick("owner-n", "host-ui", "launch-1", token);
  observeFull(service, token);
  const interval = service.intervalFacts("owner-n", "host-ui", "launch-1");
  const forged = { ...interval, digest: `sha256:${"0".repeat(64)}` };
  expectObserverError(
    () => assertNoUiProof({ kind: "observer-attestation", interval: forged }),
    "forbidden-proof",
    "admission",
  );
  // An interval with an open tick never proves: only sealed ticks judge.
  expectObserverError(
    () => assertNoUiProof({ kind: "observer-attestation", interval }),
    "forbidden-proof",
    "admission",
  );
});

check("observer-attestation-judges-ui-effects", () => {
  // Positive: a fully observed interval with zero seen effects proves no-UI.
  const clean = openLaunch();
  observeFull(clean.service, clean.token);
  clean.service.endTick("owner-n", "host-ui", "launch-1", clean.token);
  observeFull(clean.service, clean.token);
  clean.service.sealDisposal("owner-n", "host-ui", "launch-1", clean.token);
  const cleanInterval = clean.service.intervalFacts("owner-n", "host-ui", "launch-1");
  assert.equal(cleanInterval.disposed, true);
  assert.equal(cleanInterval.unknown, false);
  const cleanVerdict = assertNoUiProof({ kind: "observer-attestation", interval: cleanInterval });
  assert.equal(cleanVerdict.launchId, "launch-1");
  assert.equal(cleanVerdict.uiSeen, false);
  assert.equal(cleanVerdict.unknown, false);

  // Positive: the observer attests UI effects from its own observations.
  const dirty = openLaunch();
  dirty.service.observe("owner-n", "host-ui", "launch-1", dirty.token, "window", "seen");
  dirty.service.observe("owner-n", "host-ui", "launch-1", dirty.token, "icon", "absent");
  dirty.service.sealDisposal("owner-n", "host-ui", "launch-1", dirty.token);
  const dirtyInterval = dirty.service.intervalFacts("owner-n", "host-ui", "launch-1");
  const dirtyVerdict = assertNoUiProof({ kind: "observer-attestation", interval: dirtyInterval });
  assert.equal(dirtyVerdict.uiSeen, true);
  assert.equal(dirtyVerdict.unknown, false);

  // Missing evidence: an unknown interval judges unknown, never no-UI.
  const gappy = openLaunch();
  gappy.service.observe("owner-n", "host-ui", "launch-1", gappy.token, "window", "absent");
  gappy.service.endTick("owner-n", "host-ui", "launch-1", gappy.token);
  gappy.service.markUnknown("owner-n", "host-ui", "launch-1", gappy.token, 0, "icon");
  observeFull(gappy.service, gappy.token);
  gappy.service.sealDisposal("owner-n", "host-ui", "launch-1", gappy.token);
  const gappyInterval = gappy.service.intervalFacts("owner-n", "host-ui", "launch-1");
  assert.equal(gappyInterval.unknown, true);
  const gappyVerdict = assertNoUiProof({ kind: "observer-attestation", interval: gappyInterval });
  assert.equal(gappyVerdict.uiSeen, false);
  assert.equal(gappyVerdict.unknown, true);
});

// --- Launch tokens are opaque and verified by lookup -------------------------

check("launch-tokens-opaque-and-verified", () => {
  const { service, token } = openLaunch();
  // Invented tokens are never authority, on any token-bearing call.
  for (const forged of ["", "lnch-deadbeef", `${token.slice(0, -1)}x`]) {
    expectObserverError(
      () => service.observe("owner-n", "host-ui", "launch-1", forged, "window", "absent"),
      "forged-token",
      "admission",
    );
    expectObserverError(
      () => service.endTick("owner-n", "host-ui", "launch-1", forged),
      "forged-token",
      "admission",
    );
    expectObserverError(
      () => service.noteDriverDeath("owner-n", "host-ui", "launch-1", forged),
      "forged-token",
      "admission",
    );
    expectObserverError(
      () => service.sealDisposal("owner-n", "host-ui", "launch-1", forged),
      "forged-token",
      "admission",
    );
  }
  // Unknown launches reject before the token is even read.
  expectObserverError(
    () => service.observe("owner-n", "host-ui", "launch-9", token, "window", "absent"),
    "unknown-launch",
    "admission",
  );
  // Tokens never transfer between launches.
  admitLaunch(service, "owner-n", "host-ui", "launch-2");
  const token2 = service.launchTokenForTest("owner-n", "host-ui", "launch-2");
  assert.notEqual(token2, token);
  expectObserverError(
    () => service.observe("owner-n", "host-ui", "launch-2", token, "window", "absent"),
    "forged-token",
    "admission",
  );
  expectObserverError(
    () => service.observe("owner-n", "host-ui", "launch-1", token2, "window", "absent"),
    "forged-token",
    "admission",
  );
  // The live token works on both launches under its own binding.
  service.observe("owner-n", "host-ui", "launch-1", token, "window", "absent");
  service.observe("owner-n", "host-ui", "launch-2", token2, "window", "absent");
});

// --- The interval runs through confirmed disposal ----------------------------

check("interval-observations-through-disposal", () => {
  const { service, token } = openLaunch();
  // Tick 0: full observation, then close.
  const first = service.observe("owner-n", "host-ui", "launch-1", token, "window", "absent");
  assert.equal(first.closed, false);
  observeFull(service, token);
  const sealed0 = service.endTick("owner-n", "host-ui", "launch-1", token);
  assert.equal(sealed0.closed, true);
  assert.equal(sealed0.tick, 0);
  // The driver dies mid-interval: the observer notes it and keeps going.
  service.noteDriverDeath("owner-n", "host-ui", "launch-1", token);
  // Late host effects after driver death are still observed.
  service.observe("owner-n", "host-ui", "launch-1", token, "window", "seen");
  service.observe("owner-n", "host-ui", "launch-1", token, "icon", "absent");
  const facts = service.intervalFacts("owner-n", "host-ui", "launch-1");
  assert.equal(facts.driverDeathNoted, true);
  assert.equal(facts.disposed, false);
  assert.equal(facts.ticks.length, 2);
  // A dead driver never seals: disposal still requires the observer seal.
  expectObserverError(
    () => service.disposalReceipt("owner-n", "host-ui", "launch-1"),
    "disposal-open",
    "interval",
  );
  const receipt = service.sealDisposal("owner-n", "host-ui", "launch-1", token);
  assert.equal(receipt.launchId, "launch-1");
  assert.equal(receipt.ticks, 2);
  assert.equal(receipt.driverDeathNoted, true);
  assert.equal(receipt.unknown, false);
  assert.ok(receipt.digest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(receipt).sort(), [
    "corrections",
    "digest",
    "driverDeathNoted",
    "launchId",
    "owner",
    "scope",
    "ticks",
    "unknown",
  ]);
  // The receipt is durable: it reads back identically after disposal.
  assert.deepEqual(service.disposalReceipt("owner-n", "host-ui", "launch-1"), receipt);
  // After disposal every mutation rejects and the id never re-attests.
  expectObserverError(
    () => service.observe("owner-n", "host-ui", "launch-1", token, "window", "absent"),
    "launch-closed",
    "admission",
  );
  expectObserverError(
    () => service.endTick("owner-n", "host-ui", "launch-1", token),
    "launch-closed",
    "admission",
  );
  expectObserverError(
    () => service.sealDisposal("owner-n", "host-ui", "launch-1", token),
    "already-disposed",
    "interval",
  );
  expectObserverError(
    () => admitLaunch(service, "owner-n", "host-ui", "launch-1"),
    "launch-closed",
    "admission",
  );
  // Interval facts stay readable after disposal.
  const after = service.intervalFacts("owner-n", "host-ui", "launch-1");
  assert.equal(after.disposed, true);
  assert.equal(after.ticks.length, 2);
});

// --- Unknown and gap keys ----------------------------------------------------

check("unknown-gap-keys", () => {
  const { service, token } = openLaunch();
  // Silence is a gap, never proof of absence.
  service.observe("owner-n", "host-ui", "launch-1", token, "window", "absent");
  const tick0 = service.endTick("owner-n", "host-ui", "launch-1", token);
  const icon = tick0.channels.find((entry) => entry.channel === "icon");
  assert.equal(icon?.state, "gap");
  assert.equal(icon?.origin, "gap");
  assert.equal(icon?.corrected, false);
  // An unresolved gap blocks the seal.
  observeFull(service, token);
  expectObserverError(
    () => service.sealDisposal("owner-n", "host-ui", "launch-1", token),
    "gap-open",
    "interval",
  );
  // A gap in the open tick blocks the seal too: nothing unobserved seals.
  const service2 = openService();
  admitLaunch(service2, "owner-n", "host-ui", "launch-1");
  const token2 = service2.launchTokenForTest("owner-n", "host-ui", "launch-1");
  service2.observe("owner-n", "host-ui", "launch-1", token2, "window", "absent");
  expectObserverError(
    () => service2.sealDisposal("owner-n", "host-ui", "launch-1", token2),
    "gap-open",
    "interval",
  );
  // Publishing unknown resolves the gap honestly: the seal carries unknown.
  service.markUnknown("owner-n", "host-ui", "launch-1", token, 0, "icon");
  const receipt = service.sealDisposal("owner-n", "host-ui", "launch-1", token);
  assert.equal(receipt.unknown, true);
  const facts = service.intervalFacts("owner-n", "host-ui", "launch-1");
  assert.equal(facts.unknown, true);
});

// --- Durable correction publication ------------------------------------------

check("durable-correction-publication", () => {
  const { service, token } = openLaunch();
  service.observe("owner-n", "host-ui", "launch-1", token, "window", "absent");
  service.endTick("owner-n", "host-ui", "launch-1", token);
  observeFull(service, token);
  // A gap corrects exactly once, journaled in order.
  const first = service.publishCorrection(
    "owner-n",
    "host-ui",
    "launch-1",
    token,
    0,
    "icon",
    "absent",
  );
  assert.equal(first.correctionId, "c1");
  assert.equal(first.before, "gap");
  assert.equal(first.after, "absent");
  assert.ok(first.digest.startsWith("sha256:"));
  const entry = service
    .tickFacts("owner-n", "host-ui", "launch-1", 0)
    .channels.find((e) => e.channel === "icon");
  assert.equal(entry?.state, "absent");
  assert.equal(entry?.origin, "correction");
  assert.equal(entry?.corrected, true);
  // Replay and rewrite both reject: corrections are single-shot and never
  // touch a direct observation.
  expectObserverError(
    () => service.publishCorrection("owner-n", "host-ui", "launch-1", token, 0, "icon", "seen"),
    "correction-stale",
    "interval",
  );
  expectObserverError(
    () => service.publishCorrection("owner-n", "host-ui", "launch-1", token, 0, "window", "seen"),
    "correction-stale",
    "interval",
  );
  expectObserverError(
    () => service.markUnknown("owner-n", "host-ui", "launch-1", token, 0, "window"),
    "correction-stale",
    "interval",
  );
  // Unknown ticks, open ticks, and missing entries reject without effect.
  expectObserverError(
    () => service.publishCorrection("owner-n", "host-ui", "launch-1", token, 9, "icon", "absent"),
    "correction-unknown",
    "interval",
  );
  expectObserverError(
    () => service.publishCorrection("owner-n", "host-ui", "launch-1", token, 1, "icon", "absent"),
    "correction-unknown",
    "interval",
  );
  // The corrected gap seals clean: the interval is fully resolved.
  const receipt = service.sealDisposal("owner-n", "host-ui", "launch-1", token);
  assert.equal(receipt.unknown, false);
  assert.equal(receipt.corrections, 1);
  // The journal is durable: ordered, frozen, readable after disposal.
  const log = service.correctionLog("owner-n", "host-ui", "launch-1");
  assert.equal(log.length, 1);
  assert.equal(log[0]?.correctionId, "c1");
  assert.ok(Object.isFrozen(log));
  const verdict = assertNoUiProof({
    kind: "observer-attestation",
    interval: service.intervalFacts("owner-n", "host-ui", "launch-1"),
  });
  assert.equal(verdict.uiSeen, false);
  assert.equal(verdict.unknown, false);
});

// --- Interruption and loss make the interval unknown -------------------------

check("interruption-makes-interval-unknown", () => {
  const { service, token } = openLaunch();
  service.observe("owner-n", "host-ui", "launch-1", token, "window", "absent");
  // Interruption: the open tick's unobserved channel becomes unknown and
  // observation suspends; the observed entry is untouched.
  service.interruptScope("owner-n", "host-ui");
  expectObserverError(
    () => admitLaunch(service, "owner-n", "host-ui", "launch-2"),
    "no-observer",
    "admission",
  );
  expectObserverError(
    () => service.observe("owner-n", "host-ui", "launch-1", token, "icon", "absent"),
    "observer-interrupted",
    "observer",
  );
  expectObserverError(
    () => service.sealDisposal("owner-n", "host-ui", "launch-1", token),
    "observer-interrupted",
    "observer",
  );
  const during = service.intervalFacts("owner-n", "host-ui", "launch-1");
  assert.equal(during.unknown, true);
  // Resume: observation continues, but the interrupted tick stays unknown.
  service.resumeScope("owner-n", "host-ui");
  expectObserverError(
    () => service.resumeScope("owner-n", "host-ui"),
    "observer-interrupted",
    "observer",
  );
  admitLaunch(service, "owner-n", "host-ui", "launch-2");
  const token2 = service.launchTokenForTest("owner-n", "host-ui", "launch-2");
  observeFull(service, token2, "launch-2");
  service.sealDisposal("owner-n", "host-ui", "launch-2", token2);
  // The interrupted entry corrects durably: unknown resolves to observed.
  const fix = service.publishCorrection(
    "owner-n",
    "host-ui",
    "launch-1",
    token,
    0,
    "icon",
    "absent",
  );
  assert.equal(fix.before, "unknown");
  observeFull(service, token);
  const receipt = service.sealDisposal("owner-n", "host-ui", "launch-1", token);
  assert.equal(receipt.unknown, false);
});

check("observer-loss-seals-unknown-and-blocks-launch", () => {
  const { service, token } = openLaunch();
  observeFull(service, token);
  service.endTick("owner-n", "host-ui", "launch-1", token);
  // Loss: the interval becomes unknown, stays readable, and seals unknown.
  service.loseObserver("owner-n", "host-ui");
  expectObserverError(
    () => admitLaunch(service, "owner-n", "host-ui", "launch-2"),
    "no-observer",
    "admission",
  );
  expectObserverError(
    () => service.observe("owner-n", "host-ui", "launch-1", token, "window", "absent"),
    "observer-lost",
    "observer",
  );
  expectObserverError(() => service.resumeScope("owner-n", "host-ui"), "observer-lost", "observer");
  const facts = service.intervalFacts("owner-n", "host-ui", "launch-1");
  assert.equal(facts.unknown, true);
  const receipt = service.sealDisposal("owner-n", "host-ui", "launch-1", token);
  assert.equal(receipt.unknown, true);
  const verdict = assertNoUiProof({
    kind: "observer-attestation",
    interval: service.intervalFacts("owner-n", "host-ui", "launch-1"),
  });
  assert.equal(verdict.unknown, true);
});

// --- Unknown scopes, foreign channels, foreign owners ------------------------

check("unknown-scope-handling", () => {
  const { service, token } = openLaunch();
  // Channels outside the finite vocabulary are unknown effect keys.
  for (const bad of ["", "cursor", "window ", "WINDOW", "gpu-usage"]) {
    expectObserverError(
      () => service.observe("owner-n", "host-ui", "launch-1", token, bad, "absent"),
      "unknown-effect",
      "interval",
    );
  }
  // Vocabulary channels outside the declared scope are out of scope.
  for (const foreign of ["notification", "focus"]) {
    expectObserverError(
      () => service.observe("owner-n", "host-ui", "launch-1", token, foreign, "absent"),
      "out-of-scope",
      "interval",
    );
  }
  // The full scope observes every vocabulary channel.
  service.openScope("owner-n", "host-ui-full");
  admitLaunch(service, "owner-n", "host-ui-full", "launch-f");
  const tokenF = service.launchTokenForTest("owner-n", "host-ui-full", "launch-f");
  for (const channel of EFFECT_CHANNELS) {
    service.observe("owner-n", "host-ui-full", "launch-f", tokenF, channel, "absent");
  }
  // Non seen/absent states never observe: gap and unknown are derived.
  for (const bad of ["gap", "unknown", "", "maybe"]) {
    expectObserverError(
      () => service.observe("owner-n", "host-ui", "launch-1", token, "window", bad),
      "unknown-effect",
      "interval",
    );
  }
  // Foreign owners fail every check: scope, launch, token, and facts.
  expectObserverError(
    () => service.observe("owner-x", "host-ui", "launch-1", token, "window", "absent"),
    "wrong-owner",
    "observer",
  );
  expectObserverError(
    () => service.intervalFacts("owner-x", "host-ui", "launch-1"),
    "wrong-owner",
    "observer",
  );
  // A second identity granted the same scope name cannot join it: scopes
  // bind to exactly one owner.
  const shared = new BrowserObserverService(
    [
      { owner: "owner-n", scope: "host-ui", channels: ["window"] },
      { owner: "owner-x", scope: "host-ui", channels: ["window"] },
    ],
    roomyLimits(),
  );
  shared.openScope("owner-n", "host-ui");
  expectObserverError(() => shared.openScope("owner-x", "host-ui"), "wrong-owner", "observer");
  // Closing with a live launch refuses; disposal then close succeeds.
  expectObserverError(() => service.closeScope("owner-n", "host-ui"), "disposal-open", "interval");
  observeFull(service, token);
  service.sealDisposal("owner-n", "host-ui", "launch-1", token);
  service.closeScope("owner-n", "host-ui");
  expectObserverError(
    () => admitLaunch(service, "owner-n", "host-ui", "launch-9"),
    "no-observer",
    "admission",
  );
  expectObserverError(
    () => service.intervalFacts("owner-n", "host-ui", "launch-1"),
    "scope-closed",
    "observer",
  );
});

// --- Vocabulary, scope statement, and capacity --------------------------------

check("every-failure-names-its-layer", () => {
  // The code vocabulary is closed and every code maps to exactly one layer.
  assert.equal(BROWSER_OBSERVER_CODES.length, 18);
  assert.equal(BROWSER_ADMISSION_SCHEMA_VERSION, "1");
  for (const code of BROWSER_OBSERVER_CODES) {
    const layer = layerOfCode(code);
    assert.ok(["observer", "admission", "interval"].includes(layer));
    const error = new BrowserObserverError(code, "probe");
    assert.equal(error.layer, layer);
    assert.equal(error.code, code);
    assert.ok(error.message.startsWith(`[${layer}:${code}]`));
  }
  // Limit shapes reject at the interval layer with nothing constructed.
  expectObserverError(() => checkLimits(null), "unknown-effect", "interval");
  expectObserverError(() => checkLimits({ maxScopes: 1 }), "unknown-effect", "interval");
  expectObserverError(
    () =>
      checkLimits({
        maxScopes: 1,
        maxLaunches: 1,
        maxTicksPerLaunch: 1,
        maxCorrectionsPerLaunch: 1,
        maxBrowsers: 1,
      }),
    "unknown-effect",
    "interval",
  );
  // Capacity faults name the observer layer.
  const tiny = new BrowserObserverService(
    [{ owner: "o", scope: "s", channels: ["window"] }],
    checkLimits({ maxScopes: 1, maxLaunches: 1, maxTicksPerLaunch: 1, maxCorrectionsPerLaunch: 1 }),
  );
  tiny.openScope("o", "s");
  tiny.attestLaunch("o", "s", "l");
  const tok = tiny.launchTokenForTest("o", "s", "l");
  expectObserverError(() => tiny.attestLaunch("o", "s", "l2"), "capacity-exhausted", "observer");
  tiny.observe("o", "s", "l", tok, "window", "absent");
  expectObserverError(() => tiny.endTick("o", "s", "l", tok), "capacity-exhausted", "observer");
});

check("independence-scope-stated", () => {
  // The scope statement is a fixed export: the observer shares no driver
  // code, decoded values, receipts, flags, environment, or process state,
  // and proves its facts from its own seeded doubles alone.
  assert.ok(BROWSER_OBSERVER_INDEPENDENCE_SCOPE.startsWith("independent:"));
  assert.ok(BROWSER_OBSERVER_INDEPENDENCE_SCOPE.includes("no driver code"));
  assert.ok(BROWSER_OBSERVER_INDEPENDENCE_SCOPE.includes("seeded doubles"));
});

console.log(
  JSON.stringify({
    kind: "can.browser-driver-admission-check",
    schema_version: BROWSER_ADMISSION_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
