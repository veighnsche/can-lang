// K08 bounded self-check: N-owned pinned-driver launches, external reclaim.
//
// Run with: bun tools/runtime/test-services/browser-driver/launch-check.ts
// Local controls only. No browsers, processes, flags, environment, sampling,
// networks, files, timers, live hosts, or live runtimes. Every loop below
// is bounded by a small constant. Actual launch, host effects, and
// service-death controls are QB0.

import { strict as assert } from "node:assert";
import {
  assertCleanupWitnessed,
  BROWSER_LAUNCH_CODES,
  BROWSER_LAUNCH_INDEPENDENCE_SCOPE,
  BROWSER_LAUNCH_SCHEMA_VERSION,
  BrowserLaunchError,
  BrowserLaunchService,
  checkLaunchLimits,
  FORBIDDEN_WITNESS_KINDS,
  layerOfLaunchCode,
  type BrowserLaunchCode,
  type BrowserLaunchLayer,
  type BrowserLaunchLimits,
  type ExternalWitness,
  type WitnessFacts,
} from "./launch.ts";

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectLaunchError(
  body: () => unknown,
  code: BrowserLaunchCode,
  layer: BrowserLaunchLayer,
): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof BrowserLaunchError)) {
      assert.fail(`expected a BrowserLaunchError, got ${String(error)}`);
    }
    assert.equal(error.code, code);
    assert.equal(error.layer, layer);
    assert.ok(error.message.includes(`[${layer}:${code}]`));
    return;
  }
  assert.fail(`expected failure ${layer}:${code}`);
}

// Seeded outside-service double: proves discovery, containment, and
// reclaim from its own seeded table alone. Witness tokens are verified
// by lookup; invented tokens and cross-launch tokens never verify.
class FakeExternalWitness implements ExternalWitness {
  readonly authority: string =
    "external: seeded child-discovery/containment/reclaim double; proves " +
    "discovery, containment, and reclaim from its own seeded table alone";
  private readonly table = new Map<
    string,
    { token: string; children: string[]; contained: boolean }
  >();
  private seq = 0;

  seedReclaim(launchId: string, children: readonly string[], contained = true): string {
    if (this.table.size >= 8) {
      throw new Error("fake witness table full");
    }
    this.seq += 1;
    const token = `wit-seed-${this.seq}-${launchId}`;
    this.table.set(launchId, { token, children: [...children].sort(), contained });
    return token;
  }

  verifyWitness(launchId: string, token: string): WitnessFacts {
    for (const [id, entry] of this.table) {
      if (entry.token === token) {
        if (id !== launchId) {
          throw new BrowserLaunchError("forged-witness", "witness is for another launch");
        }
        return Object.freeze({
          launchId: id,
          children: Object.freeze([...entry.children]),
          contained: entry.contained,
          digest: `fake:${id}:${entry.children.join(",")}`,
        });
      }
    }
    throw new BrowserLaunchError("forged-witness", "witness token is unknown");
  }
}

function roomyLimits(): BrowserLaunchLimits {
  return checkLaunchLimits({
    maxDrivers: 4,
    maxObservers: 4,
    maxLaunches: 8,
    maxChildrenPerLaunch: 8,
  });
}

function openService(): { service: BrowserLaunchService; witness: FakeExternalWitness } {
  const witness = new FakeExternalWitness();
  const service = new BrowserLaunchService(
    [
      { owner: "owner-n", driver: "firefox", launcher: "firefox-launcher" },
      { owner: "owner-n", driver: "chromium", launcher: "chromium-launcher" },
    ],
    [{ owner: "owner-n", scope: "host-ui" }],
    roomyLimits(),
    witness,
  );
  service.bindObserver("owner-n", "host-ui");
  return { service, witness };
}

function openLaunch(): {
  service: BrowserLaunchService;
  witness: FakeExternalWitness;
  token: string;
} {
  const { service, witness } = openService();
  service.launch("owner-n", "host-ui", "firefox", "firefox-launcher", "launch-1", "profile-a");
  const token = service.launchTokenForTest("owner-n", "firefox", "launch-1");
  return { service, witness, token };
}

// --- Pinned-driver admission ------------------------------------------------

check("pinned-launcher-admits-launch", () => {
  const { service } = openService();
  const binding = service.bindObserver("owner-n", "host-ui");
  assert.equal(binding.owner, "owner-n");
  assert.equal(binding.scope, "host-ui");
  assert.ok(binding.handleDigest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(binding).sort(), ["handleDigest", "owner", "scope"]);

  const attestation = service.launch(
    "owner-n",
    "host-ui",
    "firefox",
    "firefox-launcher",
    "launch-1",
    "profile-a",
  );
  assert.equal(attestation.launchId, "launch-1");
  assert.equal(attestation.driver, "firefox");
  assert.equal(attestation.launcher, "firefox-launcher");
  assert.equal(attestation.requestedProfile, "profile-a");
  assert.ok(attestation.handleDigest.startsWith("sha256:"));
  assert.ok(attestation.observerDigest.startsWith("sha256:"));
  assert.equal(attestation.observerDigest, binding.handleDigest);
  assert.deepEqual(Object.keys(attestation).sort(), [
    "driver",
    "handleDigest",
    "launchId",
    "launcher",
    "observerDigest",
    "owner",
    "requestedProfile",
    "scope",
  ]);

  // Joining a live launch re-attests; the digest is stable.
  const joined = service.launch(
    "owner-n",
    "host-ui",
    "firefox",
    "firefox-launcher",
    "launch-1",
    "profile-a",
  );
  assert.equal(joined.handleDigest, attestation.handleDigest);

  // Joining with differing arguments refuses rather than misattributing.
  expectLaunchError(
    () =>
      service.launch("owner-n", "host-ui", "firefox", "firefox-launcher", "launch-1", "profile-b"),
    "unknown-launch",
    "launch",
  );
  expectLaunchError(
    () =>
      service.launch(
        "owner-n",
        "host-ui",
        "chromium",
        "chromium-launcher",
        "launch-1",
        "profile-a",
      ),
    "unknown-launch",
    "launch",
  );

  // A foreign bound owner cannot join another identity's launch id.
  const twoOwner = new BrowserLaunchService(
    [
      { owner: "owner-n", driver: "firefox", launcher: "firefox-launcher" },
      { owner: "owner-x", driver: "firefox", launcher: "firefox-launcher" },
    ],
    [
      { owner: "owner-n", scope: "host-ui" },
      { owner: "owner-x", scope: "host-ui-x" },
    ],
    roomyLimits(),
    new FakeExternalWitness(),
  );
  twoOwner.bindObserver("owner-n", "host-ui");
  twoOwner.bindObserver("owner-x", "host-ui-x");
  twoOwner.launch("owner-n", "host-ui", "firefox", "firefox-launcher", "launch-1", "profile-a");
  expectLaunchError(
    () =>
      twoOwner.launch(
        "owner-x",
        "host-ui-x",
        "firefox",
        "firefox-launcher",
        "launch-1",
        "profile-a",
      ),
    "wrong-owner",
    "service",
  );

  // A second pinned driver launches under its own launcher.
  service.launch("owner-n", "host-ui", "chromium", "chromium-launcher", "launch-2", "profile-b");
  assert.equal(service.launchCount, 2);
});

check("foreign-driver-and-launcher-reject", () => {
  const { service } = openService();
  // Unknown drivers and malformed names reject before any effect.
  for (const driver of ["safari", "", "firefox ", "FIREFOX"]) {
    expectLaunchError(
      () =>
        service.launch("owner-n", "host-ui", driver, "firefox-launcher", "launch-9", "profile-a"),
      "unknown-driver",
      "service",
    );
  }
  // The pinned launcher is exact: any other launcher for the same
  // driver rejects, even a launcher pinned to another driver.
  for (const launcher of ["firefox-launcher-2", "chromium-launcher"]) {
    expectLaunchError(
      () => service.launch("owner-n", "host-ui", "firefox", launcher, "launch-9", "profile-a"),
      "launcher-unpinned",
      "service",
    );
  }
  // A malformed launcher name fails validation before the pin comparison.
  expectLaunchError(
    () => service.launch("owner-n", "host-ui", "firefox", "", "launch-9", "profile-a"),
    "unknown-driver",
    "service",
  );
  assert.equal(service.launchCount, 0);
  // Malformed profiles reject at the service layer too.
  expectLaunchError(
    () => service.launch("owner-n", "host-ui", "firefox", "firefox-launcher", "launch-9", ""),
    "unknown-driver",
    "service",
  );
  // Empty and duplicate declarations never construct.
  const witness = new FakeExternalWitness();
  expectLaunchError(
    () => new BrowserLaunchService([], [{ owner: "o", scope: "s" }], roomyLimits(), witness),
    "unknown-driver",
    "service",
  );
  expectLaunchError(
    () =>
      new BrowserLaunchService(
        [
          { owner: "o", driver: "d", launcher: "l" },
          { owner: "o", driver: "d", launcher: "l" },
        ],
        [{ owner: "o", scope: "s" }],
        roomyLimits(),
        witness,
      ),
    "unknown-driver",
    "service",
  );
  expectLaunchError(
    () =>
      new BrowserLaunchService(
        [{ owner: "o", driver: "d", launcher: "l" }],
        [],
        roomyLimits(),
        witness,
      ),
    "no-observer",
    "service",
  );
});

// --- Acceptance: a missing qualified observer blocks launch ------------------

check("missing-observer-blocks-launch", () => {
  const witness = new FakeExternalWitness();
  const service = new BrowserLaunchService(
    [{ owner: "owner-n", driver: "firefox", launcher: "firefox-launcher" }],
    [{ owner: "owner-n", scope: "host-ui" }],
    roomyLimits(),
    witness,
  );
  // Never bound: admission refuses without a live qualified binding.
  expectLaunchError(
    () => service.launch("owner-n", "host-ui", "firefox", "firefox-launcher", "launch-1", "p"),
    "no-observer",
    "service",
  );
  // Ungranted scope: admission refuses the same way.
  expectLaunchError(
    () => service.launch("owner-n", "elsewhere", "firefox", "firefox-launcher", "launch-1", "p"),
    "no-observer",
    "service",
  );
  // Foreign owner: admission refuses the same way.
  service.bindObserver("owner-n", "host-ui");
  expectLaunchError(
    () => service.launch("owner-x", "host-ui", "firefox", "firefox-launcher", "launch-1", "p"),
    "no-observer",
    "service",
  );
  // A second identity granted the same scope name cannot join it:
  // bindings hold exactly one owner.
  const shared = new BrowserLaunchService(
    [{ owner: "owner-n", driver: "firefox", launcher: "firefox-launcher" }],
    [
      { owner: "owner-n", scope: "host-ui" },
      { owner: "owner-x", scope: "host-ui" },
    ],
    roomyLimits(),
    witness,
  );
  shared.bindObserver("owner-n", "host-ui");
  expectLaunchError(() => shared.bindObserver("owner-x", "host-ui"), "wrong-owner", "service");
  // Live binding: the same launch admits.
  service.launch("owner-n", "host-ui", "firefox", "firefox-launcher", "launch-1", "p");
  const token = service.launchTokenForTest("owner-n", "firefox", "launch-1");
  // Loss: later launches refuse, while in-flight facts stay readable.
  service.loseObserver("owner-n", "host-ui");
  expectLaunchError(
    () => service.launch("owner-n", "host-ui", "firefox", "firefox-launcher", "launch-2", "p"),
    "no-observer",
    "service",
  );
  const facts = service.launchFacts("owner-n", "firefox", "launch-1");
  assert.equal(facts.launchId, "launch-1");
  assert.equal(token.length > 0, true);
});

// --- Identity capture before context exposure --------------------------------

check("identity-captured-before-exposure", () => {
  const { service, token } = openLaunch();
  const identity = service.captureIdentity(
    "owner-n",
    "firefox",
    "launch-1",
    token,
    "profile-effective",
    ["child-b", "child-a"],
  );
  assert.equal(identity.effectiveProfile, "profile-effective");
  assert.equal(identity.requestedProfile, "profile-a");
  assert.deepEqual([...identity.children], ["child-a", "child-b"]);
  assert.ok(identity.handleDigest.startsWith("sha256:"));
  assert.ok(identity.digest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(identity).sort(), [
    "children",
    "digest",
    "driver",
    "effectiveProfile",
    "handleDigest",
    "launchId",
    "launcher",
    "owner",
    "requestedProfile",
  ]);

  const grant = service.exposeContext("owner-n", "firefox", "launch-1", token);
  assert.equal(grant.profile, "profile-effective");
  assert.deepEqual([...grant.children], ["child-a", "child-b"]);
  assert.ok(grant.handleDigest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(grant).sort(), [
    "children",
    "digest",
    "driver",
    "handleDigest",
    "launchId",
    "launcher",
    "owner",
    "profile",
  ]);
  // The context token is opaque, distinct, and digest-only in facts.
  const contextToken = service.contextTokenForTest("owner-n", "firefox", "launch-1");
  assert.notEqual(contextToken, token);
  assert.ok(!JSON.stringify(grant).includes(contextToken));
  // Exposing twice joins the same grant.
  const again = service.exposeContext("owner-n", "firefox", "launch-1", token);
  assert.equal(again.handleDigest, grant.handleDigest);
});

check("exposure-without-identity-rejects", () => {
  const { service, token } = openLaunch();
  // Exposure before capture refuses: identity first.
  expectLaunchError(
    () => service.exposeContext("owner-n", "firefox", "launch-1", token),
    "identity-open",
    "launch",
  );
  expectLaunchError(
    () => service.contextTokenForTest("owner-n", "firefox", "launch-1"),
    "identity-open",
    "launch",
  );
  // The capture window is pre-exposure: a later capture refines.
  service.captureIdentity("owner-n", "firefox", "launch-1", token, "profile-old", ["child-a"]);
  const refined = service.captureIdentity("owner-n", "firefox", "launch-1", token, "profile-new", [
    "child-b",
  ]);
  assert.equal(refined.effectiveProfile, "profile-new");
  assert.deepEqual([...refined.children], ["child-b"]);
  // Exposure freezes the identity: capture after exposure refuses.
  service.exposeContext("owner-n", "firefox", "launch-1", token);
  expectLaunchError(
    () => service.captureIdentity("owner-n", "firefox", "launch-1", token, "profile-x", []),
    "identity-open",
    "launch",
  );
  // Malformed, repeated, and over-cap child identities never capture.
  const { service: fresh, token: freshToken } = openLaunch();
  for (const bad of ["", "child/x", "CHILD ", "child..x..y..z..!"]) {
    expectLaunchError(
      () => fresh.captureIdentity("owner-n", "firefox", "launch-1", freshToken, "p", [bad]),
      "unknown-child",
      "reclaim",
    );
  }
  expectLaunchError(
    () => fresh.captureIdentity("owner-n", "firefox", "launch-1", freshToken, "p", ["c", "c"]),
    "unknown-child",
    "reclaim",
  );
  expectLaunchError(
    () =>
      fresh.captureIdentity(
        "owner-n",
        "firefox",
        "launch-1",
        freshToken,
        "p",
        "c" as unknown as readonly unknown[],
      ),
    "unknown-child",
    "reclaim",
  );
  expectLaunchError(
    () => fresh.captureIdentity("owner-n", "firefox", "launch-1", freshToken, "", ["c"]),
    "unknown-driver",
    "service",
  );
});

// --- Driver death: noting, gating, and external reclaim ----------------------

check("driver-death-noted-and-gated", () => {
  const { service, token } = openLaunch();
  service.captureIdentity("owner-n", "firefox", "launch-1", token, "profile-a", ["child-a"]);
  // Noting death is idempotent: the fact sticks.
  service.noteDriverDeath("owner-n", "firefox", "launch-1", token);
  service.noteDriverDeath("owner-n", "firefox", "launch-1", token);
  const facts = service.launchFacts("owner-n", "firefox", "launch-1");
  assert.equal(facts.driverDeathNoted, true);
  assert.equal(facts.exposed, false);
  assert.equal(facts.reclaimed, false);
  // A dead launcher reports and exposes nothing further.
  expectLaunchError(
    () => service.captureIdentity("owner-n", "firefox", "launch-1", token, "p", ["c"]),
    "driver-dead",
    "launch",
  );
  expectLaunchError(
    () => service.exposeContext("owner-n", "firefox", "launch-1", token),
    "driver-dead",
    "launch",
  );
});

check("launch-tokens-opaque-and-verified", () => {
  const { service, witness, token } = openLaunch();
  const witnessToken = witness.seedReclaim("launch-1", []);
  // Invented tokens are never authority, on any token-bearing call.
  for (const forged of ["", "lnch-deadbeef", `${token.slice(0, -1)}x`]) {
    expectLaunchError(
      () => service.captureIdentity("owner-n", "firefox", "launch-1", forged, "p", []),
      "forged-token",
      "launch",
    );
    expectLaunchError(
      () => service.exposeContext("owner-n", "firefox", "launch-1", forged),
      "forged-token",
      "launch",
    );
    expectLaunchError(
      () => service.noteDriverDeath("owner-n", "firefox", "launch-1", forged),
      "forged-token",
      "launch",
    );
    expectLaunchError(
      () => service.sealReclaim("owner-n", "firefox", "launch-1", forged, witnessToken),
      "forged-token",
      "launch",
    );
  }
  // Unknown launches reject before the token is even read.
  expectLaunchError(
    () => service.exposeContext("owner-n", "firefox", "launch-9", token),
    "unknown-launch",
    "launch",
  );
  // Foreign owners and drivers fail every check.
  expectLaunchError(
    () => service.exposeContext("owner-x", "firefox", "launch-1", token),
    "wrong-owner",
    "service",
  );
  expectLaunchError(
    () => service.exposeContext("owner-n", "chromium", "launch-1", token),
    "unknown-driver",
    "service",
  );
  expectLaunchError(
    () => service.launchFacts("owner-x", "firefox", "launch-1"),
    "wrong-owner",
    "service",
  );
  // Tokens never transfer between launches.
  service.launch("owner-n", "host-ui", "firefox", "firefox-launcher", "launch-2", "p");
  const token2 = service.launchTokenForTest("owner-n", "firefox", "launch-2");
  assert.notEqual(token2, token);
  expectLaunchError(
    () => service.exposeContext("owner-n", "firefox", "launch-2", token),
    "forged-token",
    "launch",
  );
  expectLaunchError(
    () => service.exposeContext("owner-n", "firefox", "launch-1", token2),
    "forged-token",
    "launch",
  );
  // The live token works on both launches under its own binding.
  service.captureIdentity("owner-n", "firefox", "launch-1", token, "p", []);
  service.captureIdentity("owner-n", "firefox", "launch-2", token2, "p", []);
});

check("death-mid-launch-reclaims-without-orphans", () => {
  const { service, witness, token } = openLaunch();
  service.captureIdentity("owner-n", "firefox", "launch-1", token, "profile-a", [
    "child-a",
    "child-b",
  ]);
  service.exposeContext("owner-n", "firefox", "launch-1", token);
  // The driver dies mid-launch: the outside authority discovers both
  // children and the seal carries them all. Nothing orphans.
  service.noteDriverDeath("owner-n", "firefox", "launch-1", token);
  const witnessToken = witness.seedReclaim("launch-1", ["child-a", "child-b"]);
  const receipt = service.sealReclaim("owner-n", "firefox", "launch-1", token, witnessToken);
  assert.equal(receipt.launchId, "launch-1");
  assert.equal(receipt.driver, "firefox");
  assert.equal(receipt.launcher, "firefox-launcher");
  assert.equal(receipt.profile, "profile-a");
  assert.deepEqual([...receipt.children], ["child-a", "child-b"]);
  assert.equal(receipt.driverDeathNoted, true);
  assert.ok(receipt.digest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(receipt).sort(), [
    "children",
    "digest",
    "driver",
    "driverDeathNoted",
    "launchId",
    "launcher",
    "owner",
    "profile",
  ]);
  // The receipt is durable: it reads back identically after reclaim.
  assert.deepEqual(service.reclaimReceipt("owner-n", "firefox", "launch-1"), receipt);
  const facts = service.launchFacts("owner-n", "firefox", "launch-1");
  assert.equal(facts.reclaimed, true);
  assert.deepEqual([...facts.capturedChildren], ["child-a", "child-b"]);
  assert.deepEqual([...facts.reclaimedChildren], ["child-a", "child-b"]);
  // After reclaim every mutation rejects and the id never relaunches.
  expectLaunchError(
    () => service.sealReclaim("owner-n", "firefox", "launch-1", token, witnessToken),
    "already-reclaimed",
    "reclaim",
  );
  expectLaunchError(
    () => service.captureIdentity("owner-n", "firefox", "launch-1", token, "p", []),
    "launch-closed",
    "launch",
  );
  expectLaunchError(
    () => service.exposeContext("owner-n", "firefox", "launch-1", token),
    "launch-closed",
    "launch",
  );
  expectLaunchError(
    () => service.noteDriverDeath("owner-n", "firefox", "launch-1", token),
    "launch-closed",
    "launch",
  );
  expectLaunchError(
    () => service.launch("owner-n", "host-ui", "firefox", "firefox-launcher", "launch-1", "p"),
    "launch-closed",
    "launch",
  );
});

check("partial-witness-leaves-orphan-open", () => {
  const { service, witness, token } = openLaunch();
  service.captureIdentity("owner-n", "firefox", "launch-1", token, "p", ["child-a", "child-b"]);
  // A witness missing a captured child names the orphan and seals nothing.
  const partial = witness.seedReclaim("launch-1", ["child-a"]);
  expectLaunchError(
    () => service.sealReclaim("owner-n", "firefox", "launch-1", token, partial),
    "orphan-open",
    "reclaim",
  );
  try {
    service.sealReclaim("owner-n", "firefox", "launch-1", token, partial);
    assert.fail("expected orphan-open");
  } catch (error) {
    assert.ok(String(error).includes("child-b"));
  }
  // An uncontained witness seals nothing either.
  const loose = witness.seedReclaim("launch-1", ["child-a", "child-b"], false);
  expectLaunchError(
    () => service.sealReclaim("owner-n", "firefox", "launch-1", token, loose),
    "orphan-open",
    "reclaim",
  );
  // Reading the receipt before the seal refuses: reclaim is still open.
  expectLaunchError(
    () => service.reclaimReceipt("owner-n", "firefox", "launch-1"),
    "orphan-open",
    "reclaim",
  );
  // Invented witness tokens are never authority.
  for (const forged of ["", "wit-seed-99-launch-1", `${partial.slice(0, -1)}x`]) {
    expectLaunchError(
      () => service.sealReclaim("owner-n", "firefox", "launch-1", token, forged),
      "forged-witness",
      "reclaim",
    );
  }
  // Witness tokens never transfer between launches.
  service.launch("owner-n", "host-ui", "firefox", "firefox-launcher", "launch-2", "p");
  const token2 = service.launchTokenForTest("owner-n", "firefox", "launch-2");
  service.captureIdentity("owner-n", "firefox", "launch-2", token2, "p", []);
  const other = witness.seedReclaim("launch-2", []);
  expectLaunchError(
    () => service.sealReclaim("owner-n", "firefox", "launch-1", token, other),
    "forged-witness",
    "reclaim",
  );
  // Extra witnessed children beyond the captured set are recorded, not
  // refused: external ownership covers more than the launcher reported.
  service.captureIdentity("owner-n", "firefox", "launch-2", token2, "p", ["child-a"]);
  const wide = witness.seedReclaim("launch-2", ["child-a", "child-extra"]);
  const receipt = service.sealReclaim("owner-n", "firefox", "launch-2", token2, wide);
  assert.deepEqual([...receipt.children], ["child-a", "child-extra"]);
});

check("death-before-capture-seals-externally", () => {
  // Death before any capture: the launcher reported nothing, so the
  // external authority's discovery is the whole reclaimed set.
  const { service, witness, token } = openLaunch();
  service.noteDriverDeath("owner-n", "firefox", "launch-1", token);
  const witnessToken = witness.seedReclaim("launch-1", ["child-found"]);
  const receipt = service.sealReclaim("owner-n", "firefox", "launch-1", token, witnessToken);
  assert.equal(receipt.profile, null);
  assert.deepEqual([...receipt.children], ["child-found"]);
  assert.equal(receipt.driverDeathNoted, true);
  const facts = service.launchFacts("owner-n", "firefox", "launch-1");
  assert.deepEqual([...facts.capturedChildren], []);
  assert.deepEqual([...facts.reclaimedChildren], ["child-found"]);
  const verdict = assertCleanupWitnessed({ kind: "external-witness", receipt });
  assert.equal(verdict.witnessed, true);
  assert.deepEqual([...verdict.children], ["child-found"]);
});

// --- A receipt from the dead driver is never its own cleanup witness --------

check("driver-receipt-never-witnesses-cleanup", () => {
  assert.deepEqual([...FORBIDDEN_WITNESS_KINDS], ["driver-receipt", "launcher-log", "exit-code"]);
  // Every forbidden kind rejects before any verdict is read.
  for (const kind of FORBIDDEN_WITNESS_KINDS) {
    expectLaunchError(
      () => assertCleanupWitnessed({ kind, detail: "driver says clean" }),
      "forbidden-witness",
      "reclaim",
    );
  }
  // Unknown kinds and malformed claims reject the same way.
  for (const claim of [
    { kind: "obituary", detail: "driver died quietly" },
    { kind: "process-sample", detail: "no pid found" },
    { kind: 7, detail: "x" },
    { detail: "no kind" },
    null,
    "external-witness",
  ]) {
    expectLaunchError(() => assertCleanupWitnessed(claim), "forbidden-witness", "reclaim");
  }
  // A forged receipt digest never verifies.
  const { service, witness, token } = openLaunch();
  service.captureIdentity("owner-n", "firefox", "launch-1", token, "p", ["child-a"]);
  service.noteDriverDeath("owner-n", "firefox", "launch-1", token);
  const witnessToken = witness.seedReclaim("launch-1", ["child-a"]);
  const receipt = service.sealReclaim("owner-n", "firefox", "launch-1", token, witnessToken);
  const forged = { ...receipt, digest: `sha256:${"0".repeat(64)}` };
  expectLaunchError(
    () => assertCleanupWitnessed({ kind: "external-witness", receipt: forged }),
    "forbidden-witness",
    "reclaim",
  );
  // A receipt with edited children never verifies either.
  const edited = { ...receipt, children: [] };
  expectLaunchError(
    () => assertCleanupWitnessed({ kind: "external-witness", receipt: edited }),
    "forbidden-witness",
    "reclaim",
  );
});

check("external-witness-verdict-judges-cleanup", () => {
  // Positive: a sealed receipt verifies and judges witnessed cleanup.
  const { service, witness, token } = openLaunch();
  service.captureIdentity("owner-n", "firefox", "launch-1", token, "p", ["child-a"]);
  const witnessToken = witness.seedReclaim("launch-1", ["child-a"]);
  const receipt = service.sealReclaim("owner-n", "firefox", "launch-1", token, witnessToken);
  const verdict = assertCleanupWitnessed({ kind: "external-witness", receipt });
  assert.equal(verdict.launchId, "launch-1");
  assert.equal(verdict.witnessed, true);
  assert.deepEqual([...verdict.children], ["child-a"]);
  assert.ok(verdict.digest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(verdict).sort(), ["children", "digest", "launchId", "witnessed"]);
  // Missing evidence: an unsealed launch has no receipt to judge.
  const { service: bare } = openService();
  bare.launch("owner-n", "host-ui", "firefox", "firefox-launcher", "launch-1", "p");
  expectLaunchError(
    () => bare.reclaimReceipt("owner-n", "firefox", "launch-1"),
    "orphan-open",
    "reclaim",
  );
});

// --- Vocabulary, scope statement, and capacity --------------------------------

check("every-failure-names-its-layer", () => {
  // The code vocabulary is closed and every code maps to exactly one layer.
  assert.equal(BROWSER_LAUNCH_CODES.length, 15);
  assert.equal(BROWSER_LAUNCH_SCHEMA_VERSION, "1");
  for (const code of BROWSER_LAUNCH_CODES) {
    const layer = layerOfLaunchCode(code);
    assert.ok(["service", "launch", "reclaim"].includes(layer));
    const error = new BrowserLaunchError(code, "probe");
    assert.equal(error.layer, layer);
    assert.equal(error.code, code);
    assert.ok(error.message.startsWith(`[${layer}:${code}]`));
  }
  // Limit shapes reject at the service layer with nothing constructed.
  expectLaunchError(() => checkLaunchLimits(null), "unknown-driver", "service");
  expectLaunchError(() => checkLaunchLimits({ maxDrivers: 1 }), "unknown-driver", "service");
  expectLaunchError(
    () =>
      checkLaunchLimits({
        maxDrivers: 1,
        maxObservers: 1,
        maxLaunches: 1,
        maxChildrenPerLaunch: 1,
        maxBrowsers: 1,
      }),
    "unknown-driver",
    "service",
  );
  // A missing witness authority never constructs: cleanup without an
  // external owner is unwitnessable.
  expectLaunchError(
    () =>
      new BrowserLaunchService(
        [{ owner: "o", driver: "d", launcher: "l" }],
        [{ owner: "o", scope: "s" }],
        roomyLimits(),
        null as unknown as ExternalWitness,
      ),
    "forbidden-witness",
    "reclaim",
  );
  // Capacity faults name the service layer.
  const witness = new FakeExternalWitness();
  const tiny = new BrowserLaunchService(
    [{ owner: "o", driver: "d", launcher: "l" }],
    [{ owner: "o", scope: "s" }],
    checkLaunchLimits({ maxDrivers: 1, maxObservers: 1, maxLaunches: 1, maxChildrenPerLaunch: 1 }),
    witness,
  );
  tiny.bindObserver("o", "s");
  tiny.launch("o", "s", "d", "l", "l1", "p");
  const tok = tiny.launchTokenForTest("o", "d", "l1");
  expectLaunchError(
    () => tiny.launch("o", "s", "d", "l", "l2", "p"),
    "capacity-exhausted",
    "service",
  );
  expectLaunchError(
    () => tiny.captureIdentity("o", "d", "l1", tok, "p", ["a", "b"]),
    "capacity-exhausted",
    "service",
  );
});

check("independence-scope-stated", () => {
  // The scope statement is a fixed export: the service shares no driver
  // code, decoded values, receipts, flags, environment, or process state,
  // and proves its facts from its own seeded doubles alone.
  assert.ok(BROWSER_LAUNCH_INDEPENDENCE_SCOPE.startsWith("independent:"));
  assert.ok(BROWSER_LAUNCH_INDEPENDENCE_SCOPE.includes("no driver code"));
  assert.ok(BROWSER_LAUNCH_INDEPENDENCE_SCOPE.includes("seeded doubles"));
  // The witness seam is external by construction: the double states its
  // outside-service authority.
  const witness = new FakeExternalWitness();
  assert.ok(witness.authority.startsWith("external:"));
});

console.log(
  JSON.stringify({
    kind: "can.browser-driver-launch-check",
    schema_version: BROWSER_LAUNCH_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
