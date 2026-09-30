// K09 bounded self-check: browser event capture and terminal seal.
//
// Run with: bun tools/runtime/test-services/browser-driver/events-check.ts
// Local controls only. No browsers, processes, flags, environment, sampling,
// networks, files, timers, live hosts, or live runtimes. Every loop below
// is bounded by a small constant. Live host-dependent controls wait for the
// qualified profile and Q task.

import { strict as assert } from "node:assert";
import {
  ABSENCE_SUBJECTS,
  assertAbsenceProved,
  BROWSER_EVENT_CODES,
  BROWSER_EVENT_INDEPENDENCE_SCOPE,
  BROWSER_EVENT_SCHEMA_VERSION,
  BrowserEventError,
  BrowserEventService,
  checkEventLimits,
  DECODING_SCOPES,
  FORBIDDEN_ABSENCE_KINDS,
  HEADER_SCOPES,
  layerOfEventCode,
  WATCH_TARGETS,
  type BrowserEventCode,
  type BrowserEventLayer,
  type BrowserEventLimits,
} from "./events.ts";

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectEventError(
  body: () => unknown,
  code: BrowserEventCode,
  layer: BrowserEventLayer,
): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof BrowserEventError)) {
      assert.fail(`expected a BrowserEventError, got ${String(error)}`);
    }
    assert.equal(error.code, code);
    assert.equal(error.layer, layer);
    assert.ok(error.message.includes(`[${layer}:${code}]`));
    return;
  }
  assert.fail(`expected failure ${layer}:${code}`);
}

const BODY_DIGEST = `sha256:${"ab".repeat(32)}`;

function roomyLimits(): BrowserEventLimits {
  return checkEventLimits({
    maxContexts: 4,
    maxWatches: 8,
    maxEventsPerWatch: 48,
    maxPendingPerWatch: 8,
  });
}

function openService(): BrowserEventService {
  const service = new BrowserEventService(
    [
      { owner: "owner-n", context: "ctx-a" },
      { owner: "owner-n", context: "ctx-b" },
    ],
    roomyLimits(),
  );
  service.bindContext("owner-n", "ctx-a");
  return service;
}

function openWatch(
  target = "page",
  watchId = "watch-1",
): { service: BrowserEventService; token: string } {
  const service = openService();
  service.watch("owner-n", "ctx-a", watchId, target);
  const token = service.watchTokenForTest("owner-n", "ctx-a", watchId);
  return { service, token };
}

function bodyDescriptor(byteLength = 12): {
  byteLength: number;
  bodyDigest: string;
  decodingScope: string;
} {
  return { byteLength, bodyDigest: BODY_DIGEST, decodingScope: "utf8-strict" };
}

// Seal the standard way: checkpoint the tip, then seal.
function sealTip(service: BrowserEventService, token: string, watchId = "watch-1"): void {
  const tip = service.readFrom("owner-n", "ctx-a", watchId, 0).nextCursor;
  service.checkpoint("owner-n", "ctx-a", watchId, token, tip);
  service.sealWatch("owner-n", "ctx-a", watchId, token);
}

// --- Context/page/navigation watches and admission-first --------------------

check("context-watch-admission", () => {
  const service = new BrowserEventService([{ owner: "owner-n", context: "ctx-a" }], roomyLimits());
  const binding = service.bindContext("owner-n", "ctx-a");
  assert.equal(binding.owner, "owner-n");
  assert.equal(binding.context, "ctx-a");
  assert.ok(binding.handleDigest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(binding).sort(), ["context", "handleDigest", "owner"]);

  // Rebinding joins the same owned context; the digest is stable.
  const again = service.bindContext("owner-n", "ctx-a");
  assert.equal(again.handleDigest, binding.handleDigest);

  // Contexts outside the owned grant reject before any effect.
  for (const foreign of ["ctx-z", "", "ctx-a/x"]) {
    expectEventError(() => service.bindContext("owner-n", foreign), "unknown-context", "capture");
  }
  expectEventError(() => service.bindContext("owner-x", "ctx-a"), "unknown-context", "capture");

  // Every declared target opens a watch with a stable rejoin attestation.
  assert.deepEqual([...WATCH_TARGETS], ["context", "page", "navigation"]);
  for (const target of WATCH_TARGETS) {
    const attestation = service.watch("owner-n", "ctx-a", `watch-${target}`, target);
    assert.equal(attestation.target, target);
    assert.equal(attestation.cursor, 0);
    assert.equal(attestation.contextDigest, binding.handleDigest);
    assert.ok(attestation.handleDigest.startsWith("sha256:"));
    assert.deepEqual(Object.keys(attestation).sort(), [
      "context",
      "contextDigest",
      "cursor",
      "handleDigest",
      "owner",
      "target",
      "watchId",
    ]);
    const joined = service.watch("owner-n", "ctx-a", `watch-${target}`, target);
    assert.equal(joined.handleDigest, attestation.handleDigest);
  }
  // Joining with another target refuses rather than retargeting the watch.
  expectEventError(
    () => service.watch("owner-n", "ctx-a", "watch-page", "context"),
    "unknown-target",
    "watch",
  );
  // Targets outside the finite vocabulary reject before any effect.
  for (const bad of ["frame", "", "PAGE", "page/1"]) {
    expectEventError(
      () => service.watch("owner-n", "ctx-a", "watch-bad", bad),
      "unknown-target",
      "watch",
    );
  }
  assert.equal(service.watchCount("owner-n", "ctx-a"), 3);
});

check("missing-context-blocks-watch", () => {
  const service = new BrowserEventService([{ owner: "owner-n", context: "ctx-a" }], roomyLimits());
  // Never bound: admission refuses without a live context binding.
  expectEventError(
    () => service.watch("owner-n", "ctx-a", "watch-1", "page"),
    "no-context",
    "capture",
  );
  // Ungranted context: admission refuses the same way.
  expectEventError(
    () => service.watch("owner-n", "elsewhere", "watch-1", "page"),
    "no-context",
    "capture",
  );
  // Foreign owner: admission refuses the same way.
  service.bindContext("owner-n", "ctx-a");
  expectEventError(
    () => service.watch("owner-x", "ctx-a", "watch-1", "page"),
    "no-context",
    "capture",
  );
  // A second identity granted the same context name cannot join it:
  // contexts bind to exactly one owner.
  const shared = new BrowserEventService(
    [
      { owner: "owner-n", context: "ctx-a" },
      { owner: "owner-x", context: "ctx-a" },
    ],
    roomyLimits(),
  );
  shared.bindContext("owner-n", "ctx-a");
  expectEventError(() => shared.bindContext("owner-x", "ctx-a"), "wrong-owner", "capture");
  // Live binding: the same watch admits and attests.
  service.watch("owner-n", "ctx-a", "watch-1", "page");
  const token = service.watchTokenForTest("owner-n", "ctx-a", "watch-1");
  assert.ok(token.length > 0);
  // Foreign owners fail capture, reads, and facts.
  expectEventError(
    () =>
      service.emitRequest("owner-x", "ctx-a", "watch-1", token, {
        captureId: "c1",
        method: "GET",
        target: "root",
      }),
    "wrong-owner",
    "capture",
  );
  expectEventError(
    () => service.readFrom("owner-x", "ctx-a", "watch-1", 0),
    "wrong-owner",
    "capture",
  );
  expectEventError(
    () => service.intervalFacts("owner-x", "ctx-a", "watch-1"),
    "wrong-owner",
    "capture",
  );
});

// --- Tokens are opaque and verified by lookup --------------------------------

check("watch-tokens-opaque-and-verified", () => {
  const { service, token } = openWatch();
  const request = { captureId: "c1", method: "GET", target: "root" };
  // Invented tokens are never authority, on any token-bearing call.
  for (const forged of ["", "wtch-deadbeef", `${token.slice(0, -1)}x`]) {
    expectEventError(
      () => service.emitRequest("owner-n", "ctx-a", "watch-1", forged, request),
      "forged-token",
      "watch",
    );
    expectEventError(
      () =>
        service.emitNavigation("owner-n", "ctx-a", "watch-1", forged, {
          navId: "n1",
          from: "a",
          to: "b",
        }),
      "forged-token",
      "watch",
    );
    expectEventError(
      () => service.completeBody("owner-n", "ctx-a", "watch-1", forged, "c1", bodyDescriptor()),
      "forged-token",
      "watch",
    );
    expectEventError(
      () => service.failCapture("owner-n", "ctx-a", "watch-1", forged, "c1", "aborted"),
      "forged-token",
      "watch",
    );
    expectEventError(
      () =>
        service.noteGap("owner-n", "ctx-a", "watch-1", forged, {
          dropped: 1,
          reason: "dropped-events",
        }),
      "forged-token",
      "watch",
    );
    expectEventError(
      () => service.checkpoint("owner-n", "ctx-a", "watch-1", forged, 0),
      "forged-token",
      "watch",
    );
    expectEventError(
      () => service.sealWatch("owner-n", "ctx-a", "watch-1", forged),
      "forged-token",
      "watch",
    );
  }
  // Unknown watches reject before the token is even read.
  expectEventError(
    () => service.emitRequest("owner-n", "ctx-a", "watch-9", token, request),
    "unknown-watch",
    "watch",
  );
  // Tokens never transfer between watches.
  service.watch("owner-n", "ctx-a", "watch-2", "page");
  const token2 = service.watchTokenForTest("owner-n", "ctx-a", "watch-2");
  assert.notEqual(token2, token);
  expectEventError(
    () => service.emitRequest("owner-n", "ctx-a", "watch-2", token, request),
    "forged-token",
    "watch",
  );
  expectEventError(
    () => service.emitRequest("owner-n", "ctx-a", "watch-1", token2, request),
    "forged-token",
    "watch",
  );
  // The live token works on both watches under its own binding.
  service.emitRequest("owner-n", "ctx-a", "watch-1", token, request);
  service.emitRequest("owner-n", "ctx-a", "watch-2", token2, request);
});

// --- Complete body/error callbacks -------------------------------------------

check("request-response-body-flow", () => {
  const { service, token } = openWatch();
  const request = service.emitRequest("owner-n", "ctx-a", "watch-1", token, {
    captureId: "c1",
    method: "GET",
    target: "root",
  });
  assert.equal(request.seq, 0);
  assert.equal(request.kind, "request");
  assert.equal(request.captureId, "c1");
  assert.ok(request.digest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(request).sort(), [
    "captureId",
    "digest",
    "kind",
    "method",
    "seq",
    "target",
  ]);
  // The request opens a pending capture: a replay refuses.
  expectEventError(
    () =>
      service.emitRequest("owner-n", "ctx-a", "watch-1", token, {
        captureId: "c1",
        method: "GET",
        target: "root",
      }),
    "capture-pending",
    "watch",
  );
  const response = service.emitResponse("owner-n", "ctx-a", "watch-1", token, {
    captureId: "c1",
    status: 200,
    headerScope: "repeated-joined",
    decodingScope: "utf8-strict",
  });
  assert.equal(response.seq, 1);
  assert.equal(response.headerScope, "repeated-joined");
  assert.equal(response.decodingScope, "utf8-strict");
  // A second response for the same capture refuses: one response joins.
  expectEventError(
    () =>
      service.emitResponse("owner-n", "ctx-a", "watch-1", token, {
        captureId: "c1",
        status: 200,
        headerScope: "single",
        decodingScope: "utf8-strict",
      }),
    "capture-pending",
    "watch",
  );
  // A response without its request refuses: responses join captures.
  expectEventError(
    () =>
      service.emitResponse("owner-n", "ctx-a", "watch-1", token, {
        captureId: "c-orphan",
        status: 200,
        headerScope: "single",
        decodingScope: "utf8-strict",
      }),
    "capture-unknown",
    "watch",
  );
  const body = service.completeBody("owner-n", "ctx-a", "watch-1", token, "c1", bodyDescriptor(12));
  assert.equal(body.seq, 2);
  assert.equal(body.kind, "body-complete");
  assert.equal(body.byteLength, 12);
  assert.equal(body.bodyDigest, BODY_DIGEST);
  assert.equal(body.decodingScope, "utf8-strict");
  // Facts carry digests only: the raw token and raw bytes never cross.
  const facts = service.intervalFacts("owner-n", "ctx-a", "watch-1");
  assert.deepEqual([...facts.pending], []);
  assert.ok(!JSON.stringify(facts).includes(token));
  // Settling twice refuses: the capture is no longer pending.
  expectEventError(
    () => service.completeBody("owner-n", "ctx-a", "watch-1", token, "c1", bodyDescriptor()),
    "capture-unknown",
    "watch",
  );
  expectEventError(
    () => service.failCapture("owner-n", "ctx-a", "watch-1", token, "c1", "aborted"),
    "capture-unknown",
    "watch",
  );
});

check("error-callback-settles-capture", () => {
  const { service, token } = openWatch();
  service.emitRequest("owner-n", "ctx-a", "watch-1", token, {
    captureId: "c1",
    method: "POST",
    target: "submit",
  });
  const failure = service.failCapture("owner-n", "ctx-a", "watch-1", token, "c1", "timeout");
  assert.equal(failure.seq, 1);
  assert.equal(failure.kind, "capture-error");
  assert.equal(failure.reason, "timeout");
  assert.deepEqual(Object.keys(failure).sort(), ["captureId", "digest", "kind", "reason", "seq"]);
  assert.deepEqual([...service.intervalFacts("owner-n", "ctx-a", "watch-1").pending], []);
  // Errors outside the finite vocabulary reject without effect.
  service.emitRequest("owner-n", "ctx-a", "watch-1", token, {
    captureId: "c2",
    method: "GET",
    target: "root",
  });
  for (const bad of ["crash", "", "TIMEOUT", "aborted "]) {
    expectEventError(
      () => service.failCapture("owner-n", "ctx-a", "watch-1", token, "c2", bad),
      "unknown-event",
      "watch",
    );
  }
  // The unsettled capture is still pending after the rejected errors.
  assert.deepEqual([...service.intervalFacts("owner-n", "ctx-a", "watch-1").pending], ["c2"]);
  service.failCapture("owner-n", "ctx-a", "watch-1", token, "c2", "reset");
  assert.deepEqual([...service.intervalFacts("owner-n", "ctx-a", "watch-1").pending], []);
});

// --- Acceptance: pending capture, gap, or quiet prefix never proves absence --

check("pending-capture-blocks-seal-and-proof", () => {
  const { service, token } = openWatch();
  service.emitRequest("owner-n", "ctx-a", "watch-1", token, {
    captureId: "c1",
    method: "GET",
    target: "root",
  });
  service.emitResponse("owner-n", "ctx-a", "watch-1", token, {
    captureId: "c1",
    status: 200,
    headerScope: "single",
    decodingScope: "raw-bytes",
  });
  // The seal refuses while the capture is pending, and names the capture.
  const tip = service.readFrom("owner-n", "ctx-a", "watch-1", 0).nextCursor;
  service.checkpoint("owner-n", "ctx-a", "watch-1", token, tip);
  try {
    service.sealWatch("owner-n", "ctx-a", "watch-1", token);
    assert.fail("expected capture-pending");
  } catch (error) {
    assert.ok(error instanceof BrowserEventError);
    assert.equal(error.code, "capture-pending");
    assert.ok(String(error).includes("c1"));
  }
  expectEventError(() => service.sealReceipt("owner-n", "ctx-a", "watch-1"), "seal-open", "seal");
  // A pending interval never proves absence, whatever the subject.
  const interval = service.intervalFacts("owner-n", "ctx-a", "watch-1");
  assert.deepEqual([...interval.pending], ["c1"]);
  for (const subject of ["navigation", "any"] as const) {
    expectEventError(
      () => assertAbsenceProved({ kind: "sealed-interval", interval, absenceOf: subject }),
      "forbidden-proof",
      "seal",
    );
  }
  // Settling the capture unblocks the seal.
  service.completeBody("owner-n", "ctx-a", "watch-1", token, "c1", bodyDescriptor());
  sealTip(service, token);
  const receipt = service.sealReceipt("owner-n", "ctx-a", "watch-1");
  assert.equal(receipt.gapped, false);
});

check("gap-never-proves-absence", () => {
  const { service, token } = openWatch();
  service.emitNavigation("owner-n", "ctx-a", "watch-1", token, {
    navId: "n1",
    from: "home",
    to: "away",
  });
  // The gap entry is explicit: dropped count and declared reason.
  const gap = service.noteGap("owner-n", "ctx-a", "watch-1", token, {
    dropped: 3,
    reason: "dropped-events",
  });
  assert.equal(gap.kind, "gap");
  assert.equal(gap.dropped, 3);
  assert.equal(gap.reason, "dropped-events");
  assert.deepEqual(Object.keys(gap).sort(), ["digest", "dropped", "kind", "reason", "seq"]);
  // Gaps seal honestly: the terminal receipt carries gapped.
  sealTip(service, token);
  const receipt = service.sealReceipt("owner-n", "ctx-a", "watch-1");
  assert.equal(receipt.gapped, true);
  const interval = service.intervalFacts("owner-n", "ctx-a", "watch-1");
  assert.equal(interval.gapped, true);
  assert.equal(interval.sealed, true);
  // A gapped interval never proves absence, even with zero matching events.
  for (const subject of ["request", "capture-error", "any"] as const) {
    const verdict = assertAbsenceProved({ kind: "sealed-interval", interval, absenceOf: subject });
    assert.equal(verdict.absent, false);
    assert.equal(verdict.gapped, true);
  }
  // A late-attached leading gap is a quiet prefix made explicit: sealed,
  // gapped, and still never absence.
  const late = openService();
  late.watch("owner-n", "ctx-a", "watch-late", "context");
  const lateToken = late.watchTokenForTest("owner-n", "ctx-a", "watch-late");
  late.noteGap("owner-n", "ctx-a", "watch-late", lateToken, { dropped: 9, reason: "late-attach" });
  const lateTip = late.readFrom("owner-n", "ctx-a", "watch-late", 0).nextCursor;
  late.checkpoint("owner-n", "ctx-a", "watch-late", lateToken, lateTip);
  late.sealWatch("owner-n", "ctx-a", "watch-late", lateToken);
  const lateInterval = late.intervalFacts("owner-n", "ctx-a", "watch-late");
  const lateVerdict = assertAbsenceProved({
    kind: "sealed-interval",
    interval: lateInterval,
    absenceOf: "any",
  });
  assert.equal(lateVerdict.absent, false);
  assert.equal(lateVerdict.gapped, true);
});

check("unrecorded-loss-gaps-the-interval", () => {
  // A well-formed occurrence refused for capacity reasons is unrecorded
  // loss: the seal must carry gapped, never a gapless completeness claim.
  const tight = new BrowserEventService(
    [{ owner: "owner-n", context: "ctx-a" }],
    checkEventLimits({
      maxContexts: 1,
      maxWatches: 1,
      maxEventsPerWatch: 2,
      maxPendingPerWatch: 4,
    }),
  );
  tight.bindContext("owner-n", "ctx-a");
  tight.watch("owner-n", "ctx-a", "watch-1", "page");
  const token = tight.watchTokenForTest("owner-n", "ctx-a", "watch-1");
  tight.emitNavigation("owner-n", "ctx-a", "watch-1", token, { navId: "n1", from: "a", to: "b" });
  tight.emitNavigation("owner-n", "ctx-a", "watch-1", token, { navId: "n2", from: "b", to: "c" });
  expectEventError(
    () =>
      tight.emitNavigation("owner-n", "ctx-a", "watch-1", token, {
        navId: "n3",
        from: "c",
        to: "d",
      }),
    "capacity-exhausted",
    "capture",
  );
  tight.checkpoint("owner-n", "ctx-a", "watch-1", token, 2);
  const receipt = tight.sealWatch("owner-n", "ctx-a", "watch-1", token);
  assert.equal(receipt.gapped, true);
  const verdict = assertAbsenceProved({
    kind: "sealed-interval",
    interval: tight.intervalFacts("owner-n", "ctx-a", "watch-1"),
    absenceOf: "any",
  });
  assert.equal(verdict.absent, false);
  assert.equal(verdict.gapped, true);
  // A pending-table refusal is the same class of loss.
  const pendingTight = new BrowserEventService(
    [{ owner: "owner-n", context: "ctx-a" }],
    checkEventLimits({
      maxContexts: 1,
      maxWatches: 1,
      maxEventsPerWatch: 8,
      maxPendingPerWatch: 1,
    }),
  );
  pendingTight.bindContext("owner-n", "ctx-a");
  pendingTight.watch("owner-n", "ctx-a", "watch-1", "page");
  const pendingToken = pendingTight.watchTokenForTest("owner-n", "ctx-a", "watch-1");
  pendingTight.emitRequest("owner-n", "ctx-a", "watch-1", pendingToken, {
    captureId: "c1",
    method: "GET",
    target: "a",
  });
  expectEventError(
    () =>
      pendingTight.emitRequest("owner-n", "ctx-a", "watch-1", pendingToken, {
        captureId: "c2",
        method: "GET",
        target: "b",
      }),
    "capacity-exhausted",
    "capture",
  );
  pendingTight.failCapture("owner-n", "ctx-a", "watch-1", pendingToken, "c1", "aborted");
  const pendingTip = pendingTight.readFrom("owner-n", "ctx-a", "watch-1", 0).nextCursor;
  pendingTight.checkpoint("owner-n", "ctx-a", "watch-1", pendingToken, pendingTip);
  const pendingReceipt = pendingTight.sealWatch("owner-n", "ctx-a", "watch-1", pendingToken);
  assert.equal(pendingReceipt.gapped, true);
});

check("quiet-prefix-never-proves-absence", () => {
  // An open watch with zero events is a quiet prefix, never absence proof.
  const { service, token } = openWatch();
  const quiet = service.intervalFacts("owner-n", "ctx-a", "watch-1");
  assert.equal(quiet.events.length, 0);
  assert.equal(quiet.sealed, false);
  for (const subject of ["navigation", "any"] as const) {
    expectEventError(
      () => assertAbsenceProved({ kind: "sealed-interval", interval: quiet, absenceOf: subject }),
      "forbidden-proof",
      "seal",
    );
  }
  // Still quiet after capture starts: open intervals never prove, however
  // long the silence before the first event.
  service.emitNavigation("owner-n", "ctx-a", "watch-1", token, {
    navId: "n1",
    from: "home",
    to: "away",
  });
  const open = service.intervalFacts("owner-n", "ctx-a", "watch-1");
  expectEventError(
    () => assertAbsenceProved({ kind: "sealed-interval", interval: open, absenceOf: "request" }),
    "forbidden-proof",
    "seal",
  );
  // Every forbidden absence kind rejects before any verdict is read.
  assert.deepEqual(
    [...FORBIDDEN_ABSENCE_KINDS],
    ["pending-capture", "open-watch", "quiet-prefix", "event-log"],
  );
  for (const kind of FORBIDDEN_ABSENCE_KINDS) {
    expectEventError(
      () => assertAbsenceProved({ kind, detail: "nothing seen yet" }),
      "forbidden-proof",
      "seal",
    );
  }
  // Unknown kinds and malformed claims reject the same way.
  for (const claim of [
    { kind: "driver-receipt", detail: "driver says quiet" },
    { kind: "screenshot-hash", detail: "abc" },
    { kind: 7, detail: "x" },
    { detail: "no kind" },
    null,
    "sealed-interval",
  ]) {
    expectEventError(() => assertAbsenceProved(claim), "forbidden-proof", "seal");
  }
  // Sealed-interval claims without facts or subject reject the same way.
  sealTip(service, token);
  const sealed = service.intervalFacts("owner-n", "ctx-a", "watch-1");
  for (const claim of [
    { kind: "sealed-interval", detail: "no interval" },
    { kind: "sealed-interval", interval: sealed, detail: "no subject" },
    { kind: "sealed-interval", interval: sealed, absenceOf: "body-complete" },
    { kind: "sealed-interval", interval: sealed, absenceOf: "everything" },
  ]) {
    expectEventError(() => assertAbsenceProved(claim), "forbidden-proof", "seal");
  }
  // A forged interval digest never verifies.
  const forged = { ...sealed, digest: `sha256:${"0".repeat(64)}` };
  expectEventError(
    () => assertAbsenceProved({ kind: "sealed-interval", interval: forged, absenceOf: "any" }),
    "forbidden-proof",
    "seal",
  );
  // Forged event kinds and non-string pending entries reject at the seal
  // layer instead of crashing the digest hash with a raw TypeError.
  const bogusKind = JSON.parse(JSON.stringify(sealed)) as typeof sealed;
  (bogusKind.events as unknown[]).push({ seq: 99, kind: "bogus-kind" });
  expectEventError(
    () => assertAbsenceProved({ kind: "sealed-interval", interval: bogusKind, absenceOf: "any" }),
    "forbidden-proof",
    "seal",
  );
  const bogusPending = JSON.parse(JSON.stringify(sealed)) as typeof sealed;
  (bogusPending.pending as unknown[]).push(7);
  expectEventError(
    () =>
      assertAbsenceProved({ kind: "sealed-interval", interval: bogusPending, absenceOf: "any" }),
    "forbidden-proof",
    "seal",
  );
});

// --- Acceptance: only a sealed gapless interval with zero events proves ------

check("sealed-gapless-zero-proves-absence", () => {
  assert.deepEqual(
    [...ABSENCE_SUBJECTS],
    ["request", "response", "navigation", "capture-error", "any"],
  );
  // Positive: a sealed gapless interval with zero navigations proves the
  // navigation absence it claims.
  const { service, token } = openWatch();
  service.emitRequest("owner-n", "ctx-a", "watch-1", token, {
    captureId: "c1",
    method: "GET",
    target: "root",
  });
  service.completeBody("owner-n", "ctx-a", "watch-1", token, "c1", bodyDescriptor());
  sealTip(service, token);
  const interval = service.intervalFacts("owner-n", "ctx-a", "watch-1");
  assert.equal(interval.sealed, true);
  assert.equal(interval.gapped, false);
  const clean = assertAbsenceProved({
    kind: "sealed-interval",
    interval,
    absenceOf: "navigation",
  });
  assert.equal(clean.absenceOf, "navigation");
  assert.equal(clean.absent, true);
  assert.equal(clean.gapped, false);
  assert.ok(clean.digest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(clean).sort(), [
    "absenceOf",
    "absent",
    "digest",
    "gapped",
    "watchId",
  ]);
  // The same interval does not prove request absence: one request observed.
  const dirty = assertAbsenceProved({
    kind: "sealed-interval",
    interval,
    absenceOf: "request",
  });
  assert.equal(dirty.absent, false);
  assert.equal(dirty.gapped, false);
  // Positive: a sealed gapless interval with zero events at all proves the
  // empty absence.
  const empty = openService();
  empty.watch("owner-n", "ctx-a", "watch-empty", "navigation");
  const emptyToken = empty.watchTokenForTest("owner-n", "ctx-a", "watch-empty");
  empty.checkpoint("owner-n", "ctx-a", "watch-empty", emptyToken, 0);
  empty.sealWatch("owner-n", "ctx-a", "watch-empty", emptyToken);
  const emptyInterval = empty.intervalFacts("owner-n", "ctx-a", "watch-empty");
  const emptyVerdict = assertAbsenceProved({
    kind: "sealed-interval",
    interval: emptyInterval,
    absenceOf: "any",
  });
  assert.equal(emptyVerdict.absent, true);
});

// --- Acceptance: repeated-header/decoding scope explicit in facts -------------

check("repeated-header-decoding-scope-explicit", () => {
  assert.deepEqual([...HEADER_SCOPES], ["single", "repeated-joined", "repeated-list"]);
  assert.deepEqual([...DECODING_SCOPES], ["raw-bytes", "utf8-strict", "utf8-replace"]);
  // Every declared scope pair captures and freezes into the facts.
  const { service, token } = openWatch();
  let seq = 0;
  for (const headerScope of HEADER_SCOPES) {
    for (const decodingScope of DECODING_SCOPES) {
      const captureId = `c-${headerScope}-${decodingScope}`;
      service.emitRequest("owner-n", "ctx-a", "watch-1", token, {
        captureId,
        method: "GET",
        target: "root",
      });
      seq += 1;
      const response = service.emitResponse("owner-n", "ctx-a", "watch-1", token, {
        captureId,
        status: 200,
        headerScope,
        decodingScope,
      });
      assert.equal(response.headerScope, headerScope);
      assert.equal(response.decodingScope, decodingScope);
      assert.ok(Object.isFrozen(response));
      seq += 1;
      const body = service.completeBody("owner-n", "ctx-a", "watch-1", token, captureId, {
        byteLength: 4,
        bodyDigest: BODY_DIGEST,
        decodingScope,
      });
      assert.equal(body.decodingScope, decodingScope);
      seq += 1;
    }
  }
  assert.equal(seq, HEADER_SCOPES.length * DECODING_SCOPES.length * 3);
  // Scopes outside the finite vocabulary reject without effect.
  service.emitRequest("owner-n", "ctx-a", "watch-1", token, {
    captureId: "c-scope",
    method: "GET",
    target: "root",
  });
  for (const bad of ["joined", "", "SINGLE", "repeated"]) {
    expectEventError(
      () =>
        service.emitResponse("owner-n", "ctx-a", "watch-1", token, {
          captureId: "c-scope",
          status: 200,
          headerScope: bad,
          decodingScope: "utf8-strict",
        }),
      "unknown-header-scope",
      "watch",
    );
  }
  for (const bad of ["utf8", "", "UTF8-STRICT", "latin1"]) {
    expectEventError(
      () =>
        service.emitResponse("owner-n", "ctx-a", "watch-1", token, {
          captureId: "c-scope",
          status: 200,
          headerScope: "single",
          decodingScope: bad,
        }),
      "unknown-decoding",
      "watch",
    );
    expectEventError(
      () =>
        service.completeBody("owner-n", "ctx-a", "watch-1", token, "c-scope", {
          byteLength: 1,
          bodyDigest: BODY_DIGEST,
          decodingScope: bad,
        }),
      "unknown-decoding",
      "watch",
    );
  }
  // Missing scopes reject the same way: there is no default scope.
  expectEventError(
    () =>
      service.emitResponse("owner-n", "ctx-a", "watch-1", token, {
        captureId: "c-scope",
        status: 200,
        decodingScope: "utf8-strict",
      }),
    "unknown-header-scope",
    "watch",
  );
  expectEventError(
    () =>
      service.emitResponse("owner-n", "ctx-a", "watch-1", token, {
        captureId: "c-scope",
        status: 200,
        headerScope: "single",
      }),
    "unknown-decoding",
    "watch",
  );
  // Raw bodies never cross: only a digest-shaped bodyDigest completes.
  for (const bad of [{ body: "raw" }, { bodyDigest: "not-a-digest" }, { bodyDigest: "" }]) {
    expectEventError(
      () =>
        service.completeBody("owner-n", "ctx-a", "watch-1", token, "c-scope", {
          byteLength: 3,
          decodingScope: "raw-bytes",
          ...bad,
        }),
      "unknown-event",
      "watch",
    );
  }
  // The capture is still pending after every rejected scope and body.
  assert.deepEqual([...service.intervalFacts("owner-n", "ctx-a", "watch-1").pending], ["c-scope"]);
});

// --- Cursors, checkpoint, and close/seal --------------------------------------

check("cursors-checkpoint-seal", () => {
  const { service, token } = openWatch();
  service.emitNavigation("owner-n", "ctx-a", "watch-1", token, {
    navId: "n1",
    from: "home",
    to: "away",
  });
  service.emitNavigation("owner-n", "ctx-a", "watch-1", token, {
    navId: "n2",
    from: "away",
    to: "home",
  });
  // Cursor reads page the log: full read, partial read, tip read.
  const full = service.readFrom("owner-n", "ctx-a", "watch-1", 0);
  assert.equal(full.cursor, 0);
  assert.equal(full.nextCursor, 2);
  assert.equal(full.events.length, 2);
  assert.equal(full.events[0]?.kind, "navigation");
  const page = service.readFrom("owner-n", "ctx-a", "watch-1", 1);
  assert.equal(page.cursor, 1);
  assert.equal(page.nextCursor, 2);
  assert.equal(page.events.length, 1);
  const tip = service.readFrom("owner-n", "ctx-a", "watch-1", 2);
  assert.equal(tip.events.length, 0);
  assert.equal(tip.nextCursor, 2);
  // Cursors past the tip or off the integer line are unknown.
  for (const bad of [3, 99, -1, 1.5, "0", null]) {
    expectEventError(
      () => service.readFrom("owner-n", "ctx-a", "watch-1", bad),
      "cursor-unknown",
      "watch",
    );
    expectEventError(
      () => service.checkpoint("owner-n", "ctx-a", "watch-1", token, bad),
      "cursor-unknown",
      "watch",
    );
  }
  // Checkpoints move forward only.
  const first = service.checkpoint("owner-n", "ctx-a", "watch-1", token, 1);
  assert.equal(first.cursor, 1);
  assert.equal(first.events, 2);
  assert.deepEqual(Object.keys(first).sort(), ["cursor", "digest", "events", "watchId"]);
  service.checkpoint("owner-n", "ctx-a", "watch-1", token, 1);
  expectEventError(
    () => service.checkpoint("owner-n", "ctx-a", "watch-1", token, 0),
    "checkpoint-stale",
    "seal",
  );
  // The seal requires the checkpoint at the tip.
  expectEventError(
    () => service.sealWatch("owner-n", "ctx-a", "watch-1", token),
    "checkpoint-stale",
    "seal",
  );
  service.checkpoint("owner-n", "ctx-a", "watch-1", token, 2);
  const receipt = service.sealWatch("owner-n", "ctx-a", "watch-1", token);
  assert.equal(receipt.watchId, "watch-1");
  assert.equal(receipt.events, 2);
  assert.equal(receipt.checkpoint, 2);
  assert.equal(receipt.gapped, false);
  assert.ok(receipt.digest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(receipt).sort(), [
    "checkpoint",
    "context",
    "digest",
    "events",
    "gapped",
    "owner",
    "target",
    "watchId",
  ]);
  // The receipt is durable: it reads back identically after the seal, and
  // reads stay available after the seal.
  assert.deepEqual(service.sealReceipt("owner-n", "ctx-a", "watch-1"), receipt);
  assert.equal(service.readFrom("owner-n", "ctx-a", "watch-1", 0).events.length, 2);
  // After the seal every mutation rejects and the id never re-opens.
  expectEventError(
    () =>
      service.emitNavigation("owner-n", "ctx-a", "watch-1", token, {
        navId: "n3",
        from: "a",
        to: "b",
      }),
    "watch-closed",
    "watch",
  );
  expectEventError(
    () => service.checkpoint("owner-n", "ctx-a", "watch-1", token, 2),
    "watch-closed",
    "watch",
  );
  expectEventError(
    () =>
      service.noteGap("owner-n", "ctx-a", "watch-1", token, { dropped: 1, reason: "late-attach" }),
    "watch-closed",
    "watch",
  );
  expectEventError(
    () => service.sealWatch("owner-n", "ctx-a", "watch-1", token),
    "already-sealed",
    "seal",
  );
  expectEventError(
    () => service.watch("owner-n", "ctx-a", "watch-1", "page"),
    "watch-closed",
    "watch",
  );
  // Closing with a live watch refuses; sealing then close succeeds.
  service.watch("owner-n", "ctx-a", "watch-2", "page");
  expectEventError(() => service.closeContext("owner-n", "ctx-a"), "seal-open", "seal");
  const token2 = service.watchTokenForTest("owner-n", "ctx-a", "watch-2");
  service.checkpoint("owner-n", "ctx-a", "watch-2", token2, 0);
  service.sealWatch("owner-n", "ctx-a", "watch-2", token2);
  service.closeContext("owner-n", "ctx-a");
  expectEventError(
    () => service.watch("owner-n", "ctx-a", "watch-3", "page"),
    "no-context",
    "capture",
  );
  expectEventError(
    () => service.intervalFacts("owner-n", "ctx-a", "watch-1"),
    "context-closed",
    "capture",
  );
});

// --- Navigation watches stay atomic and isolated per target -------------------

check("navigation-watches-atomic-and-isolated", () => {
  const service = openService();
  service.watch("owner-n", "ctx-a", "watch-ctx", "context");
  service.watch("owner-n", "ctx-a", "watch-page", "page");
  service.watch("owner-n", "ctx-a", "watch-nav", "navigation");
  const ctxToken = service.watchTokenForTest("owner-n", "ctx-a", "watch-ctx");
  const pageToken = service.watchTokenForTest("owner-n", "ctx-a", "watch-page");
  const navToken = service.watchTokenForTest("owner-n", "ctx-a", "watch-nav");
  // Navigation captures atomically: no pending capture opens.
  const nav = service.emitNavigation("owner-n", "ctx-a", "watch-nav", navToken, {
    navId: "n1",
    from: "home",
    to: "away",
  });
  assert.equal(nav.kind, "navigation");
  assert.equal(nav.navId, "n1");
  assert.deepEqual(Object.keys(nav).sort(), ["digest", "from", "kind", "navId", "seq", "to"]);
  assert.deepEqual([...service.intervalFacts("owner-n", "ctx-a", "watch-nav").pending], []);
  // Events land only in their own watch: sibling watches stay empty.
  assert.equal(service.intervalFacts("owner-n", "ctx-a", "watch-ctx").events.length, 0);
  assert.equal(service.intervalFacts("owner-n", "ctx-a", "watch-page").events.length, 0);
  service.emitRequest("owner-n", "ctx-a", "watch-page", pageToken, {
    captureId: "c1",
    method: "GET",
    target: "root",
  });
  assert.equal(service.intervalFacts("owner-n", "ctx-a", "watch-nav").events.length, 1);
  service.completeBody("owner-n", "ctx-a", "watch-page", pageToken, "c1", bodyDescriptor());
  // Malformed descriptors reject without effect on any layer of the log.
  for (const bad of [
    null,
    "n2",
    { navId: "n2", from: "a" },
    { navId: "n2", from: "a", to: "b", extra: 1 },
    { navId: "", from: "a", to: "b" },
  ]) {
    expectEventError(
      () => service.emitNavigation("owner-n", "ctx-a", "watch-nav", navToken, bad),
      "unknown-event",
      "watch",
    );
  }
  for (const bad of [
    { captureId: "c9", method: "FETCH", target: "root" },
    { captureId: "c9", method: "GET", target: "" },
    { captureId: "c9", method: "GET" },
  ]) {
    expectEventError(
      () => service.emitRequest("owner-n", "ctx-a", "watch-ctx", ctxToken, bad),
      "unknown-event",
      "watch",
    );
  }
  for (const bad of [99, 600, 20.5, "200"]) {
    expectEventError(
      () =>
        service.emitResponse("owner-n", "ctx-a", "watch-page", pageToken, {
          captureId: "c1",
          status: bad,
          headerScope: "single",
          decodingScope: "utf8-strict",
        }),
      "unknown-event",
      "watch",
    );
  }
  for (const bad of [{ dropped: 0 }, { dropped: -1 }, { dropped: 1, reason: "quiet" }, null]) {
    expectEventError(
      () => service.noteGap("owner-n", "ctx-a", "watch-nav", navToken, bad),
      "unknown-event",
      "watch",
    );
  }
  assert.equal(service.intervalFacts("owner-n", "ctx-a", "watch-nav").events.length, 1);
});

// --- Vocabulary, scope statement, and capacity --------------------------------

check("every-failure-names-its-layer", () => {
  // The code vocabulary is closed and every code maps to exactly one layer.
  assert.equal(BROWSER_EVENT_CODES.length, 19);
  assert.equal(BROWSER_EVENT_SCHEMA_VERSION, "1");
  for (const code of BROWSER_EVENT_CODES) {
    const layer = layerOfEventCode(code);
    assert.ok(["capture", "watch", "seal"].includes(layer));
    const error = new BrowserEventError(code, "probe");
    assert.equal(error.layer, layer);
    assert.equal(error.code, code);
    assert.ok(error.message.startsWith(`[${layer}:${code}]`));
  }
  // Limit shapes reject at the capture layer with nothing constructed.
  expectEventError(() => checkEventLimits(null), "unknown-context", "capture");
  expectEventError(() => checkEventLimits({ maxContexts: 1 }), "unknown-context", "capture");
  expectEventError(
    () =>
      checkEventLimits({
        maxContexts: 1,
        maxWatches: 1,
        maxEventsPerWatch: 1,
        maxPendingPerWatch: 1,
        maxBrowsers: 1,
      }),
    "unknown-context",
    "capture",
  );
  // Empty and duplicate declarations never construct.
  expectEventError(() => new BrowserEventService([], roomyLimits()), "unknown-context", "capture");
  expectEventError(
    () =>
      new BrowserEventService(
        [
          { owner: "o", context: "c" },
          { owner: "o", context: "c" },
        ],
        roomyLimits(),
      ),
    "unknown-context",
    "capture",
  );
  // Capacity faults name the capture layer.
  const tiny = new BrowserEventService(
    [{ owner: "o", context: "c" }],
    checkEventLimits({
      maxContexts: 1,
      maxWatches: 1,
      maxEventsPerWatch: 1,
      maxPendingPerWatch: 1,
    }),
  );
  tiny.bindContext("o", "c");
  tiny.watch("o", "c", "w", "page");
  const tok = tiny.watchTokenForTest("o", "c", "w");
  expectEventError(() => tiny.watch("o", "c", "w2", "page"), "capacity-exhausted", "capture");
  tiny.emitNavigation("o", "c", "w", tok, { navId: "n1", from: "a", to: "b" });
  expectEventError(
    () => tiny.emitNavigation("o", "c", "w", tok, { navId: "n2", from: "a", to: "b" }),
    "capacity-exhausted",
    "capture",
  );
  const pendingTiny = new BrowserEventService(
    [{ owner: "o", context: "c" }],
    checkEventLimits({
      maxContexts: 1,
      maxWatches: 1,
      maxEventsPerWatch: 8,
      maxPendingPerWatch: 1,
    }),
  );
  pendingTiny.bindContext("o", "c");
  pendingTiny.watch("o", "c", "w", "page");
  const ptok = pendingTiny.watchTokenForTest("o", "c", "w");
  pendingTiny.emitRequest("o", "c", "w", ptok, { captureId: "c1", method: "GET", target: "root" });
  expectEventError(
    () =>
      pendingTiny.emitRequest("o", "c", "w", ptok, {
        captureId: "c2",
        method: "GET",
        target: "root",
      }),
    "capacity-exhausted",
    "capture",
  );
});

check("independence-scope-stated", () => {
  // The scope statement is a fixed export: the service shares no driver
  // code, decoded values, receipts, flags, environment, or process state,
  // and proves its facts from its own seeded doubles alone.
  assert.ok(BROWSER_EVENT_INDEPENDENCE_SCOPE.startsWith("independent:"));
  assert.ok(BROWSER_EVENT_INDEPENDENCE_SCOPE.includes("no driver code"));
  assert.ok(BROWSER_EVENT_INDEPENDENCE_SCOPE.includes("seeded doubles"));
  assert.ok(BROWSER_EVENT_INDEPENDENCE_SCOPE.includes("decoding"));
});

console.log(
  JSON.stringify({
    kind: "can.browser-driver-events-check",
    schema_version: BROWSER_EVENT_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
