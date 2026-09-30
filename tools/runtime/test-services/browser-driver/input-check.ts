// K10 bounded self-check: pending input separated from settlement.
//
// Run with: bun tools/runtime/test-services/browser-driver/input-check.ts
// Local controls only. No browsers, processes, flags, environment, sampling,
// networks, files, timers, live hosts, or live runtimes. Every loop below
// is bounded by a small constant. Live host-dependent controls wait for the
// qualified profile and Q task.

import { strict as assert } from "node:assert";
import {
  assertInputSettled,
  BROWSER_INPUT_CODES,
  BROWSER_INPUT_INDEPENDENCE_SCOPE,
  BROWSER_INPUT_SCHEMA_VERSION,
  BrowserInputError,
  BrowserInputService,
  checkInputLimits,
  digestSettlement,
  FORBIDDEN_SETTLEMENT_KINDS,
  INPUT_ACTIONS,
  layerOfInputCode,
  ROUTE_ABORT_REASONS,
  SETTLEMENT_OUTCOMES,
  WITNESS_OUTCOMES,
  type BrowserInputCode,
  type BrowserInputLayer,
  type BrowserInputLimits,
} from "./input.ts";

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectInputError(
  body: () => unknown,
  code: BrowserInputCode,
  layer: BrowserInputLayer,
): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof BrowserInputError)) {
      assert.fail(`expected a BrowserInputError, got ${String(error)}`);
    }
    assert.equal(error.code, code);
    assert.equal(error.layer, layer);
    assert.ok(error.message.includes(`[${layer}:${code}]`));
    return;
  }
  assert.fail(`expected failure ${layer}:${code}`);
}

const NODE_DIGEST = `sha256:${"cd".repeat(32)}`;

function roomyLimits(): BrowserInputLimits {
  return checkInputLimits({
    maxContexts: 4,
    maxInputs: 16,
    maxRoutes: 16,
    maxCapturesPerInput: 4,
    maxNodesPerCapture: 8,
    maxWitnesses: 16,
  });
}

function openService(): BrowserInputService {
  const service = new BrowserInputService(
    [
      { owner: "owner-n", context: "ctx-a" },
      { owner: "owner-n", context: "ctx-b" },
    ],
    roomyLimits(),
  );
  service.bindContext("owner-n", "ctx-a");
  return service;
}

function beginClick(
  service: BrowserInputService,
  actionId = "click-1",
): { attestation: unknown; token: string } {
  const attestation = service.beginInput("owner-n", "ctx-a", actionId, {
    action: "click",
    target: "submit",
  });
  const token = service.inputTokenForTest("owner-n", "ctx-a", actionId);
  return { attestation, token };
}

function domDescriptor(
  nodeCount = 2,
  truncated = false,
): {
  nodes: { name: string; nodeDigest: string }[];
  truncated: boolean;
} {
  const nodes = [];
  for (let index = 0; index < nodeCount; index += 1) {
    nodes.push({ name: `node-${index}`, nodeDigest: NODE_DIGEST });
  }
  return { nodes, truncated };
}

// --- Acceptance: input_begin/input_settle with pending/settled separation ---

check("begin-pending-settle-separated", () => {
  const service = openService();
  // Every declared action begins pending with a stable rejoin attestation.
  assert.deepEqual([...INPUT_ACTIONS], ["click", "type", "press"]);
  for (const action of INPUT_ACTIONS) {
    const attestation = service.beginInput("owner-n", "ctx-a", `act-${action}`, {
      action,
      target: "target-1",
    });
    assert.equal(attestation.action, action);
    assert.equal(attestation.pending, true);
    assert.equal(attestation.domCaptures, 0);
    assert.ok(attestation.handleDigest.startsWith("sha256:"));
    assert.deepEqual(Object.keys(attestation).sort(), [
      "action",
      "actionId",
      "context",
      "contextDigest",
      "domCaptures",
      "handleDigest",
      "owner",
      "pending",
      "target",
    ]);
    const joined = service.beginInput("owner-n", "ctx-a", `act-${action}`, {
      action,
      target: "target-1",
    });
    assert.equal(joined.handleDigest, attestation.handleDigest);
  }
  assert.equal(service.inputCount("owner-n", "ctx-a"), 3);

  // Pending status is evidence: pending with no outcome and no settlement.
  const { token } = beginClick(service);
  const pending = service.inputFacts("owner-n", "ctx-a", "click-1");
  assert.equal(pending.pending, true);
  assert.equal(pending.outcome, null);
  assert.equal(pending.fulfilledByRoute, null);
  // Settlement facts do not exist while pending: the missing evidence
  // refuses instead of reading as unsettled-zero.
  expectInputError(
    () => service.settlementFacts("owner-n", "ctx-a", "click-1"),
    "input-pending",
    "input",
  );
  // Direct settle actuates and freezes the settlement facts.
  const settlement = service.settleInput("owner-n", "ctx-a", "click-1", token, "actuated");
  assert.equal(settlement.outcome, "actuated");
  assert.equal(settlement.fulfilledByRoute, null);
  assert.ok(settlement.digest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(settlement).sort(), [
    "action",
    "actionId",
    "context",
    "digest",
    "domCaptures",
    "fulfilledByRoute",
    "outcome",
    "owner",
    "target",
  ]);
  assert.ok(Object.isFrozen(settlement));
  // The settlement reads back identically; status flips to settled.
  assert.deepEqual(service.settlementFacts("owner-n", "ctx-a", "click-1"), settlement);
  const settled = service.inputFacts("owner-n", "ctx-a", "click-1");
  assert.equal(settled.pending, false);
  assert.equal(settled.outcome, "actuated");
  // Settlement is terminal: no second settle, no re-begin, no attestation.
  expectInputError(
    () => service.settleInput("owner-n", "ctx-a", "click-1", token, "rejected"),
    "action-closed",
    "input",
  );
  expectInputError(
    () => service.beginInput("owner-n", "ctx-a", "click-1", { action: "click", target: "submit" }),
    "action-closed",
    "input",
  );
  expectInputError(
    () => service.attestation("owner-n", "ctx-a", "click-1"),
    "action-closed",
    "input",
  );
  // The rejected outcome settles directly too.
  service.beginInput("owner-n", "ctx-a", "click-2", { action: "click", target: "cancel" });
  const token2 = service.inputTokenForTest("owner-n", "ctx-a", "click-2");
  const rejected = service.settleInput("owner-n", "ctx-a", "click-2", token2, "rejected");
  assert.equal(rejected.outcome, "rejected");
  assert.equal(rejected.fulfilledByRoute, null);
});

// --- Acceptance: action identity --------------------------------------------

check("action-identity-tokens-and-vocabulary", () => {
  const service = openService();
  const { token } = beginClick(service);
  // Actions outside the finite vocabulary reject before any effect.
  for (const bad of ["drag", "", "CLICK", "click "]) {
    expectInputError(
      () => service.beginInput("owner-n", "ctx-a", "act-bad", { action: bad, target: "t" }),
      "unknown-action",
      "input",
    );
  }
  // Malformed descriptors reject without effect.
  for (const bad of [
    null,
    "click",
    { action: "click" },
    { action: "click", target: "t", extra: 1 },
    { action: "click", target: "" },
  ]) {
    expectInputError(
      () => service.beginInput("owner-n", "ctx-a", "act-bad", bad),
      "unknown-action",
      "input",
    );
  }
  assert.equal(service.inputCount("owner-n", "ctx-a"), 1);
  // Re-beginning with a different descriptor refuses rather than
  // retargeting the pending input.
  expectInputError(
    () => service.beginInput("owner-n", "ctx-a", "click-1", { action: "type", target: "submit" }),
    "action-mismatch",
    "input",
  );
  expectInputError(
    () => service.beginInput("owner-n", "ctx-a", "click-1", { action: "click", target: "other" }),
    "action-mismatch",
    "input",
  );
  // Invented tokens are never authority, on capture or settle.
  for (const forged of ["", "act-deadbeef", `${token.slice(0, -1)}x`]) {
    expectInputError(
      () => service.captureDom("owner-n", "ctx-a", "click-1", forged, domDescriptor()),
      "forged-token",
      "input",
    );
    expectInputError(
      () => service.settleInput("owner-n", "ctx-a", "click-1", forged, "actuated"),
      "forged-token",
      "input",
    );
  }
  // Unknown actions reject before the token is even read.
  expectInputError(
    () => service.settleInput("owner-n", "ctx-a", "click-9", token, "actuated"),
    "unknown-action",
    "input",
  );
  // Tokens never transfer between actions.
  service.beginInput("owner-n", "ctx-a", "click-2", { action: "click", target: "t" });
  const token2 = service.inputTokenForTest("owner-n", "ctx-a", "click-2");
  assert.notEqual(token2, token);
  expectInputError(
    () => service.settleInput("owner-n", "ctx-a", "click-2", token, "actuated"),
    "forged-token",
    "input",
  );
  expectInputError(
    () => service.settleInput("owner-n", "ctx-a", "click-1", token2, "actuated"),
    "forged-token",
    "input",
  );
  // The live tokens settle under their own bindings.
  service.settleInput("owner-n", "ctx-a", "click-1", token, "actuated");
  service.settleInput("owner-n", "ctx-a", "click-2", token2, "rejected");
});

// --- Acceptance: bounded DOM capture -----------------------------------------

check("bounded-dom-capture", () => {
  const service = openService();
  const { token } = beginClick(service);
  // Captures carry node digests only, count nodes, and state truncation.
  const first = service.captureDom("owner-n", "ctx-a", "click-1", token, domDescriptor(2, false));
  assert.equal(first.captureSeq, 0);
  assert.equal(first.nodeCount, 2);
  assert.equal(first.truncated, false);
  assert.ok(first.digest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(first).sort(), [
    "actionId",
    "captureSeq",
    "digest",
    "nodeCount",
    "truncated",
  ]);
  const second = service.captureDom("owner-n", "ctx-a", "click-1", token, domDescriptor(3, true));
  assert.equal(second.captureSeq, 1);
  assert.equal(second.truncated, true);
  assert.deepEqual(service.domCaptures("owner-n", "ctx-a", "click-1"), [first, second]);
  assert.equal(service.inputFacts("owner-n", "ctx-a", "click-1").domCaptures, 2);
  // Raw bytes never cross: only digest-shaped node digests capture.
  for (const bad of [
    { nodes: [], truncated: false },
    { nodes: [{ name: "n", nodeDigest: "not-a-digest" }], truncated: false },
    { nodes: [{ name: "n", nodeDigest: "" }], truncated: false },
    { nodes: [{ name: "n", html: "<div/>" }], truncated: false },
    { nodes: [{ name: "", nodeDigest: NODE_DIGEST }], truncated: false },
    { nodes: [{ nodeDigest: NODE_DIGEST }], truncated: false },
    { nodes: "digest-string", truncated: false },
    null,
  ]) {
    expectInputError(
      () => service.captureDom("owner-n", "ctx-a", "click-1", token, bad),
      "unknown-dom",
      "input",
    );
  }
  // Truncation has no default: it states explicitly.
  expectInputError(
    () =>
      service.captureDom("owner-n", "ctx-a", "click-1", token, {
        nodes: [{ name: "n", nodeDigest: NODE_DIGEST }],
      }),
    "unknown-dom",
    "input",
  );
  // Captures land only while pending: settled actions refuse new captures.
  service.settleInput("owner-n", "ctx-a", "click-1", token, "actuated");
  expectInputError(
    () => service.captureDom("owner-n", "ctx-a", "click-1", token, domDescriptor()),
    "action-closed",
    "input",
  );
  // Earlier captures stay readable after settlement.
  assert.equal(service.domCaptures("owner-n", "ctx-a", "click-1").length, 2);
  // Per-input and per-capture bounds refuse without effect.
  const tight = new BrowserInputService(
    [{ owner: "o", context: "c" }],
    checkInputLimits({
      maxContexts: 1,
      maxInputs: 2,
      maxRoutes: 2,
      maxCapturesPerInput: 1,
      maxNodesPerCapture: 2,
      maxWitnesses: 2,
    }),
  );
  tight.bindContext("o", "c");
  tight.beginInput("o", "c", "a1", { action: "click", target: "t" });
  const tok = tight.inputTokenForTest("o", "c", "a1");
  tight.captureDom("o", "c", "a1", tok, domDescriptor(2, false));
  expectInputError(
    () => tight.captureDom("o", "c", "a1", tok, domDescriptor(1, false)),
    "capacity-exhausted",
    "input",
  );
  expectInputError(
    () => tight.captureDom("o", "c", "a1", tok, domDescriptor(3, true)),
    "capacity-exhausted",
    "input",
  );
  assert.equal(tight.domCaptures("o", "c", "a1").length, 1);
});

// --- Acceptance: lock-free route service while a click is pending ------------

check("routes-stay-lock-free-while-click-pending", () => {
  const service = openService();
  // Begin a click and leave it pending for the whole check.
  const { token } = beginClick(service);
  // Route calls take no input token and consult no input state: hold,
  // deliver, and abort all succeed with the click still pending.
  const held = service.holdRoute("owner-n", "ctx-a", "route-1");
  assert.equal(held.held, true);
  assert.ok(held.handleDigest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(held).sort(), [
    "context",
    "contextDigest",
    "handleDigest",
    "held",
    "owner",
    "routeId",
  ]);
  const rejoined = service.holdRoute("owner-n", "ctx-a", "route-1");
  assert.equal(rejoined.handleDigest, held.handleDigest);
  const routeToken = service.routeTokenForTest("owner-n", "ctx-a", "route-1");
  assert.notEqual(routeToken, token);
  assert.equal(service.inputFacts("owner-n", "ctx-a", "click-1").pending, true);
  const delivered = service.deliverRoute("owner-n", "ctx-a", "route-1", routeToken);
  assert.equal(delivered.verdict, "delivered");
  assert.equal(delivered.abortReason, null);
  assert.equal(delivered.fulfillsActionId, null);
  assert.deepEqual(Object.keys(delivered).sort(), [
    "abortReason",
    "context",
    "digest",
    "fulfillsActionId",
    "owner",
    "routeId",
    "verdict",
  ]);
  // Delivery leaves the click pending: routes never settle inputs.
  assert.equal(service.inputFacts("owner-n", "ctx-a", "click-1").pending, true);
  expectInputError(
    () => service.settlementFacts("owner-n", "ctx-a", "click-1"),
    "input-pending",
    "input",
  );
  // A second route aborts (without fulfills) while the click stays pending.
  service.holdRoute("owner-n", "ctx-a", "route-2");
  const routeToken2 = service.routeTokenForTest("owner-n", "ctx-a", "route-2");
  const aborted = service.abortRoute("owner-n", "ctx-a", "route-2", routeToken2, {
    reason: "timeout",
  });
  assert.equal(aborted.verdict, "aborted");
  assert.equal(aborted.abortReason, "timeout");
  assert.equal(service.inputFacts("owner-n", "ctx-a", "click-1").pending, true);
  // Route tokens are opaque and verified by lookup in their own namespace.
  const forgedTokens = ["", "rte-deadbeef", `${routeToken2.slice(0, -1)}x`];
  for (let index = 0; index < forgedTokens.length; index += 1) {
    const forged = forgedTokens[index] as string;
    const probeId = `route-probe-${index}`;
    service.holdRoute("owner-n", "ctx-a", probeId);
    expectInputError(
      () => service.deliverRoute("owner-n", "ctx-a", probeId, forged),
      "forged-route-token",
      "route",
    );
    expectInputError(
      () => service.abortRoute("owner-n", "ctx-a", probeId, forged, { reason: "aborted" }),
      "forged-route-token",
      "route",
    );
    // An input token is never a route token, even a live one.
    expectInputError(
      () => service.deliverRoute("owner-n", "ctx-a", probeId, token),
      "forged-route-token",
      "route",
    );
    // Unknown routes reject before the token is even read.
    expectInputError(
      () => service.deliverRoute("owner-n", "ctx-a", "route-ghost", forged),
      "unknown-route",
      "route",
    );
    const probeToken = service.routeTokenForTest("owner-n", "ctx-a", probeId);
    service.deliverRoute("owner-n", "ctx-a", probeId, probeToken);
    // Handled ids never re-open.
    expectInputError(() => service.holdRoute("owner-n", "ctx-a", probeId), "route-closed", "route");
    expectInputError(
      () => service.deliverRoute("owner-n", "ctx-a", probeId, probeToken),
      "route-closed",
      "route",
    );
  }
  // Held routes refuse delivery facts with route-pending, never a zero read.
  service.holdRoute("owner-n", "ctx-a", "route-held");
  expectInputError(
    () => service.deliveryFacts("owner-n", "ctx-a", "route-held"),
    "route-pending",
    "route",
  );
  const heldStatus = service.routeFacts("owner-n", "ctx-a", "route-held");
  assert.equal(heldStatus.held, true);
  assert.equal(heldStatus.verdict, null);
  // The click settles on its own terms at the end.
  service.settleInput("owner-n", "ctx-a", "click-1", token, "actuated");
  assert.equal(service.settlementFacts("owner-n", "ctx-a", "click-1").outcome, "actuated");
});

// --- Acceptance: route abort may fulfill input, explicitly or not at all ----

check("abort-fulfills-explicit-path", () => {
  assert.deepEqual([...SETTLEMENT_OUTCOMES], ["actuated", "rejected", "aborted-by-route"]);
  assert.deepEqual([...ROUTE_ABORT_REASONS], ["aborted", "timeout", "closed"]);
  const service = openService();
  const { token } = beginClick(service);
  // Every declared abort reason delivers the explicit fulfills path.
  for (const reason of ROUTE_ABORT_REASONS) {
    const actionId = `click-${reason}`;
    service.beginInput("owner-n", "ctx-a", actionId, { action: "click", target: "t" });
    const routeId = `route-${reason}`;
    service.holdRoute("owner-n", "ctx-a", routeId);
    const routeToken = service.routeTokenForTest("owner-n", "ctx-a", routeId);
    const delivery = service.abortRoute("owner-n", "ctx-a", routeId, routeToken, {
      reason,
      fulfillsActionId: actionId,
    });
    assert.equal(delivery.verdict, "aborted");
    assert.equal(delivery.abortReason, reason);
    assert.equal(delivery.fulfillsActionId, actionId);
    const settlement = service.settlementFacts("owner-n", "ctx-a", actionId);
    assert.equal(settlement.outcome, "aborted-by-route");
    assert.equal(settlement.fulfilledByRoute, routeId);
    const verdict = assertInputSettled({ kind: "input-settlement", settlement });
    assert.equal(verdict.settled, true);
    assert.equal(verdict.fulfilledByRoute, routeId);
  }
  // An abort without fulfillsActionId leaves every input pending: the
  // fulfills link is explicit or it does not exist, never silent.
  service.holdRoute("owner-n", "ctx-a", "route-quiet");
  const quietToken = service.routeTokenForTest("owner-n", "ctx-a", "route-quiet");
  const quiet = service.abortRoute("owner-n", "ctx-a", "route-quiet", quietToken, {
    reason: "aborted",
  });
  assert.equal(quiet.fulfillsActionId, null);
  assert.equal(service.inputFacts("owner-n", "ctx-a", "click-1").pending, true);
  // Aborted-by-route never settles directly: it is derived only through
  // the abort path.
  expectInputError(
    () => service.settleInput("owner-n", "ctx-a", "click-1", token, "aborted-by-route"),
    "unknown-outcome",
    "input",
  );
  for (const bad of ["fulfilled", "", "ACTUATED", "aborted"]) {
    expectInputError(
      () => service.settleInput("owner-n", "ctx-a", "click-1", token, bad),
      "unknown-outcome",
      "input",
    );
  }
  assert.equal(service.inputFacts("owner-n", "ctx-a", "click-1").pending, true);
  // Fulfills faults refuse without settling anything: unknown action,
  // settled action, undeclared reason, malformed descriptor.
  service.holdRoute("owner-n", "ctx-a", "route-fault");
  const faultToken = service.routeTokenForTest("owner-n", "ctx-a", "route-fault");
  expectInputError(
    () =>
      service.abortRoute("owner-n", "ctx-a", "route-fault", faultToken, {
        reason: "aborted",
        fulfillsActionId: "click-ghost",
      }),
    "unknown-action",
    "input",
  );
  service.settleInput("owner-n", "ctx-a", "click-1", token, "actuated");
  expectInputError(
    () =>
      service.abortRoute("owner-n", "ctx-a", "route-fault", faultToken, {
        reason: "aborted",
        fulfillsActionId: "click-1",
      }),
    "action-closed",
    "input",
  );
  for (const bad of ["crash", "", "ABORTED", null]) {
    expectInputError(
      () => service.abortRoute("owner-n", "ctx-a", "route-fault", faultToken, { reason: bad }),
      "unknown-route-outcome",
      "route",
    );
  }
  expectInputError(
    () => service.abortRoute("owner-n", "ctx-a", "route-fault", faultToken, null),
    "unknown-route-outcome",
    "route",
  );
  // The faulted route is still held after every rejected abort.
  assert.equal(service.routeFacts("owner-n", "ctx-a", "route-fault").held, true);
});

// --- Acceptance: settlement, delivery, and witness are three separate facts -

check("settlement-delivery-witness-never-conflated", () => {
  assert.deepEqual([...WITNESS_OUTCOMES], ["effect-seen", "effect-absent"]);
  const service = openService();
  const { token } = beginClick(service);
  service.captureDom("owner-n", "ctx-a", "click-1", token, domDescriptor(1, false));
  service.settleInput("owner-n", "ctx-a", "click-1", token, "actuated");
  const settlement = service.settlementFacts("owner-n", "ctx-a", "click-1");
  service.holdRoute("owner-n", "ctx-a", "route-1");
  const routeToken = service.routeTokenForTest("owner-n", "ctx-a", "route-1");
  const delivery = service.deliverRoute("owner-n", "ctx-a", "route-1", routeToken);
  const witness = service.witnessApplication("owner-n", "ctx-a", "wit-1", {
    actionId: "click-1",
    outcome: "effect-seen",
  });
  // The three facts carry disjoint identifying keys: no shape reads as
  // another fact.
  assert.ok(!("verdict" in settlement) && !("witnessId" in settlement));
  assert.ok(!("outcome" in delivery) && !("witnessId" in delivery));
  assert.ok(!("outcome" in delivery) && "verdict" in delivery);
  assert.ok(!("verdict" in witness) && !("fulfilledByRoute" in witness));
  assert.deepEqual(Object.keys(witness).sort(), [
    "actionId",
    "context",
    "digest",
    "outcome",
    "owner",
    "witnessId",
  ]);
  assert.ok(Object.isFrozen(witness));
  // Positive: the verified settlement judges settled.
  const verdict = assertInputSettled({ kind: "input-settlement", settlement });
  assert.equal(verdict.actionId, "click-1");
  assert.equal(verdict.outcome, "actuated");
  assert.equal(verdict.settled, true);
  assert.deepEqual(Object.keys(verdict).sort(), [
    "actionId",
    "digest",
    "fulfilledByRoute",
    "outcome",
    "settled",
  ]);
  // Negative: route delivery and application witness never prove
  // settlement, whatever they carry.
  assert.deepEqual(
    [...FORBIDDEN_SETTLEMENT_KINDS],
    ["route-delivery", "app-witness", "dom-capture", "input-attestation"],
  );
  for (const kind of FORBIDDEN_SETTLEMENT_KINDS) {
    expectInputError(
      () => assertInputSettled({ kind, detail: "handler ran" }),
      "forbidden-proof",
      "witness",
    );
  }
  for (const claim of [
    { kind: "route-delivery", detail: JSON.stringify(delivery) },
    { kind: "app-witness", detail: JSON.stringify(witness) },
    { kind: "driver-receipt", detail: "driver says clicked" },
    { kind: 7, detail: "x" },
    { detail: "no kind" },
    null,
    "input-settlement",
  ]) {
    expectInputError(() => assertInputSettled(claim), "forbidden-proof", "witness");
  }
  // Settlement claims without facts reject the same way.
  for (const claim of [
    { kind: "input-settlement", detail: "no settlement" },
    { kind: "input-settlement", settlement: null },
    { kind: "input-settlement", settlement: { ...settlement, outcome: "fulfilled" } },
    { kind: "input-settlement", settlement: { ...settlement, digest: `sha256:${"0".repeat(64)}` } },
  ]) {
    expectInputError(() => assertInputSettled(claim), "forbidden-proof", "witness");
  }
  // The abort-fulfills link is exact in both directions: even with a
  // valid digest, a direct outcome carrying a route and an abort outcome
  // missing its route both refuse.
  const forgedLink = {
    ...settlement,
    fulfilledByRoute: "route-1",
    digest: digestSettlement(
      settlement.actionId,
      settlement.owner,
      settlement.context,
      settlement.action,
      settlement.target,
      settlement.outcome,
      "route-1",
      settlement.domCaptures,
    ),
  };
  expectInputError(
    () => assertInputSettled({ kind: "input-settlement", settlement: forgedLink }),
    "forbidden-proof",
    "witness",
  );
  const missingLink = {
    ...settlement,
    outcome: "aborted-by-route" as const,
    fulfilledByRoute: null,
    digest: digestSettlement(
      settlement.actionId,
      settlement.owner,
      settlement.context,
      settlement.action,
      settlement.target,
      "aborted-by-route",
      null,
      settlement.domCaptures,
    ),
  };
  expectInputError(
    () => assertInputSettled({ kind: "input-settlement", settlement: missingLink }),
    "forbidden-proof",
    "witness",
  );
  // An undeclared action refuses even with a recomputed digest: closed
  // vocabularies hold at the verdict boundary too.
  const forgedAction = {
    ...settlement,
    action: "explode",
    digest: digestSettlement(
      settlement.actionId,
      settlement.owner,
      settlement.context,
      "explode",
      settlement.target,
      settlement.outcome,
      settlement.fulfilledByRoute,
      settlement.domCaptures,
    ),
  };
  expectInputError(
    () => assertInputSettled({ kind: "input-settlement", settlement: forgedAction }),
    "forbidden-proof",
    "witness",
  );
});

// --- Witness independence: records without reading or settling -------------

check("witness-independent-never-settles", () => {
  const service = openService();
  beginClick(service);
  // A witness for a pending action leaves it pending.
  const seen = service.witnessApplication("owner-n", "ctx-a", "wit-1", {
    actionId: "click-1",
    outcome: "effect-seen",
  });
  assert.equal(seen.actionId, "click-1");
  assert.equal(seen.outcome, "effect-seen");
  assert.equal(service.inputFacts("owner-n", "ctx-a", "click-1").pending, true);
  expectInputError(
    () => service.settlementFacts("owner-n", "ctx-a", "click-1"),
    "input-pending",
    "input",
  );
  // A witness for an unknown name still records: the witness never reads
  // the input table.
  const ambient = service.witnessApplication("owner-n", "ctx-a", "wit-2", {
    actionId: "click-ghost",
    outcome: "effect-absent",
  });
  assert.equal(ambient.actionId, "click-ghost");
  const unnamed = service.witnessApplication("owner-n", "ctx-a", "wit-3", {
    outcome: "effect-absent",
  });
  assert.equal(unnamed.actionId, null);
  assert.deepEqual(service.witnessFacts("owner-n", "ctx-a", "wit-1"), seen);
  // Witness ids are single-use; outcomes outside the vocabulary refuse.
  expectInputError(
    () =>
      service.witnessApplication("owner-n", "ctx-a", "wit-1", {
        actionId: "click-1",
        outcome: "effect-seen",
      }),
    "unknown-witness",
    "witness",
  );
  expectInputError(
    () => service.witnessFacts("owner-n", "ctx-a", "wit-ghost"),
    "unknown-witness",
    "witness",
  );
  for (const bad of ["seen", "", "EFFECT-SEEN", null]) {
    expectInputError(
      () =>
        service.witnessApplication("owner-n", "ctx-a", "wit-bad", {
          actionId: "click-1",
          outcome: bad,
        }),
      "unknown-witness-outcome",
      "witness",
    );
  }
  assert.equal(service.witnessCount("owner-n", "ctx-a"), 3);
});

// --- Admission-first and owner checks ----------------------------------------

check("missing-context-blocks-begin-hold-witness", () => {
  const service = new BrowserInputService([{ owner: "owner-n", context: "ctx-a" }], roomyLimits());
  const descriptor = { action: "click", target: "t" };
  // Never bound: begin, hold, and witness all refuse without a live binding.
  expectInputError(
    () => service.beginInput("owner-n", "ctx-a", "click-1", descriptor),
    "no-context",
    "input",
  );
  expectInputError(() => service.holdRoute("owner-n", "ctx-a", "route-1"), "no-context", "input");
  expectInputError(
    () =>
      service.witnessApplication("owner-n", "ctx-a", "wit-1", {
        actionId: null,
        outcome: "effect-seen",
      }),
    "no-context",
    "input",
  );
  // Ungranted context and foreign owner refuse the same way.
  expectInputError(
    () => service.beginInput("owner-n", "elsewhere", "click-1", descriptor),
    "no-context",
    "input",
  );
  service.bindContext("owner-n", "ctx-a");
  expectInputError(
    () => service.beginInput("owner-x", "ctx-a", "click-1", descriptor),
    "no-context",
    "input",
  );
  expectInputError(() => service.holdRoute("owner-x", "ctx-a", "route-1"), "no-context", "input");
  // A second identity granted the same context name cannot join it:
  // contexts bind to exactly one owner.
  const shared = new BrowserInputService(
    [
      { owner: "owner-n", context: "ctx-a" },
      { owner: "owner-x", context: "ctx-a" },
    ],
    roomyLimits(),
  );
  shared.bindContext("owner-n", "ctx-a");
  expectInputError(() => shared.bindContext("owner-x", "ctx-a"), "wrong-owner", "input");
  // Live binding: begin, hold, and witness admit together.
  service.beginInput("owner-n", "ctx-a", "click-1", descriptor);
  service.holdRoute("owner-n", "ctx-a", "route-1");
  service.witnessApplication("owner-n", "ctx-a", "wit-1", {
    actionId: "click-1",
    outcome: "effect-seen",
  });
  const token = service.inputTokenForTest("owner-n", "ctx-a", "click-1");
  // Foreign owners fail capture, settle, route handling, and reads.
  expectInputError(
    () => service.captureDom("owner-x", "ctx-a", "click-1", token, domDescriptor()),
    "wrong-owner",
    "input",
  );
  expectInputError(
    () => service.settleInput("owner-x", "ctx-a", "click-1", token, "actuated"),
    "wrong-owner",
    "input",
  );
  const routeToken = service.routeTokenForTest("owner-n", "ctx-a", "route-1");
  expectInputError(
    () => service.deliverRoute("owner-x", "ctx-a", "route-1", routeToken),
    "wrong-owner",
    "input",
  );
  expectInputError(() => service.inputFacts("owner-x", "ctx-a", "click-1"), "wrong-owner", "input");
  expectInputError(() => service.routeFacts("owner-x", "ctx-a", "route-1"), "wrong-owner", "input");
  expectInputError(() => service.witnessFacts("owner-x", "ctx-a", "wit-1"), "wrong-owner", "input");
});

// --- Close gates: pending inputs and held routes block; witness never does ---

check("close-gates-on-inputs-and-routes", () => {
  const service = openService();
  const { token } = beginClick(service);
  service.holdRoute("owner-n", "ctx-a", "route-1");
  service.witnessApplication("owner-n", "ctx-a", "wit-1", {
    actionId: "click-1",
    outcome: "effect-absent",
  });
  // A pending input blocks the close and names the action.
  try {
    service.closeContext("owner-n", "ctx-a");
    assert.fail("expected input-pending");
  } catch (error) {
    assert.ok(error instanceof BrowserInputError);
    assert.equal(error.code, "input-pending");
    assert.ok(String(error).includes("click-1"));
  }
  service.settleInput("owner-n", "ctx-a", "click-1", token, "actuated");
  // A held route blocks the close and names the route.
  try {
    service.closeContext("owner-n", "ctx-a");
    assert.fail("expected route-pending");
  } catch (error) {
    assert.ok(error instanceof BrowserInputError);
    assert.equal(error.code, "route-pending");
    assert.ok(String(error).includes("route-1"));
  }
  // Handling the route unblocks: witnesses never blocked.
  const routeToken = service.routeTokenForTest("owner-n", "ctx-a", "route-1");
  service.deliverRoute("owner-n", "ctx-a", "route-1", routeToken);
  service.closeContext("owner-n", "ctx-a");
  // After the close, admission refuses and facts report closed.
  expectInputError(
    () => service.beginInput("owner-n", "ctx-a", "click-2", { action: "click", target: "t" }),
    "no-context",
    "input",
  );
  expectInputError(() => service.holdRoute("owner-n", "ctx-a", "route-2"), "no-context", "input");
  expectInputError(
    () => service.inputFacts("owner-n", "ctx-a", "click-1"),
    "context-closed",
    "input",
  );
});

// --- Vocabulary, scope statement, and capacity --------------------------------

check("every-failure-names-its-layer", () => {
  // The code vocabulary is closed and every code maps to exactly one layer.
  assert.equal(BROWSER_INPUT_CODES.length, 20);
  assert.equal(BROWSER_INPUT_SCHEMA_VERSION, "1");
  for (const code of BROWSER_INPUT_CODES) {
    const layer = layerOfInputCode(code);
    assert.ok(["input", "route", "witness"].includes(layer));
    const error = new BrowserInputError(code, "probe");
    assert.equal(error.layer, layer);
    assert.equal(error.code, code);
    assert.ok(error.message.startsWith(`[${layer}:${code}]`));
  }
  // Limit shapes reject at the input layer with nothing constructed.
  expectInputError(() => checkInputLimits(null), "unknown-context", "input");
  expectInputError(() => checkInputLimits({ maxContexts: 1 }), "unknown-context", "input");
  expectInputError(
    () =>
      checkInputLimits({
        maxContexts: 1,
        maxInputs: 1,
        maxRoutes: 1,
        maxCapturesPerInput: 1,
        maxNodesPerCapture: 1,
        maxWitnesses: 1,
        maxBrowsers: 1,
      }),
    "unknown-context",
    "input",
  );
  // Empty and duplicate declarations never construct.
  expectInputError(() => new BrowserInputService([], roomyLimits()), "unknown-context", "input");
  expectInputError(
    () =>
      new BrowserInputService(
        [
          { owner: "o", context: "c" },
          { owner: "o", context: "c" },
        ],
        roomyLimits(),
      ),
    "unknown-context",
    "input",
  );
  // Capacity faults name the input layer.
  const tiny = new BrowserInputService(
    [{ owner: "o", context: "c" }],
    checkInputLimits({
      maxContexts: 1,
      maxInputs: 1,
      maxRoutes: 1,
      maxCapturesPerInput: 1,
      maxNodesPerCapture: 1,
      maxWitnesses: 1,
    }),
  );
  tiny.bindContext("o", "c");
  tiny.beginInput("o", "c", "a1", { action: "click", target: "t" });
  expectInputError(
    () => tiny.beginInput("o", "c", "a2", { action: "click", target: "t" }),
    "capacity-exhausted",
    "input",
  );
  tiny.holdRoute("o", "c", "r1");
  expectInputError(() => tiny.holdRoute("o", "c", "r2"), "capacity-exhausted", "input");
  tiny.witnessApplication("o", "c", "w1", { actionId: null, outcome: "effect-seen" });
  expectInputError(
    () => tiny.witnessApplication("o", "c", "w2", { actionId: null, outcome: "effect-seen" }),
    "capacity-exhausted",
    "input",
  );
  // Facts carry digests only: raw tokens never cross into facts.
  const service = openService();
  const { token } = beginClick(service);
  const facts = service.inputFacts("owner-n", "ctx-a", "click-1");
  assert.ok(!JSON.stringify(facts).includes(token));
});

check("independence-scope-stated", () => {
  // The scope statement is a fixed export: the service shares no driver
  // code, decoded values, receipts, flags, environment, or process state,
  // and proves its facts from its own seeded doubles alone.
  assert.ok(BROWSER_INPUT_INDEPENDENCE_SCOPE.startsWith("independent:"));
  assert.ok(BROWSER_INPUT_INDEPENDENCE_SCOPE.includes("no driver code"));
  assert.ok(BROWSER_INPUT_INDEPENDENCE_SCOPE.includes("seeded doubles"));
  assert.ok(BROWSER_INPUT_INDEPENDENCE_SCOPE.includes("abort-fulfills"));
});

console.log(
  JSON.stringify({
    kind: "can.browser-driver-input-check",
    schema_version: BROWSER_INPUT_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
