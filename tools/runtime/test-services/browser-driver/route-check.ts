// K11 bounded self-check: one-contact route tokens.
//
// Run with: bun tools/runtime/test-services/browser-driver/route-check.ts
// Local controls only. No browsers, processes, flags, environment, sampling,
// networks, files, timers, live hosts, or live runtimes. Every loop below
// is bounded by a small constant. Live host-dependent controls wait for the
// qualified profile and Q task.

import { strict as assert } from "node:assert";
import {
  BROWSER_ROUTE_CODES,
  BROWSER_ROUTE_INDEPENDENCE_SCOPE,
  BROWSER_ROUTE_SCHEMA_VERSION,
  BrowserRouteError,
  BrowserRouteService,
  checkRouteLimits,
  digestContact,
  digestDelivery,
  layerOfRouteCode,
  PARTIAL_BODY_REASONS,
  RESOLVE_MODES,
  ROUTE_JOURNAL_OPS,
  ROUTE_METHODS,
  type BrowserRouteCode,
  type BrowserRouteLayer,
  type BrowserRouteLimits,
} from "./route.ts";

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectRouteError(
  body: () => unknown,
  code: BrowserRouteCode,
  layer: BrowserRouteLayer,
): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof BrowserRouteError)) {
      assert.fail(`expected a BrowserRouteError, got ${String(error)}`);
    }
    assert.equal(error.code, code);
    assert.equal(error.layer, layer);
    assert.ok(error.message.includes(`[${layer}:${code}]`));
    return;
  }
  assert.fail(`expected failure ${layer}:${code}`);
}

const BODY_A = `sha256:${"aa".repeat(32)}`;
const BODY_B = `sha256:${"bb".repeat(32)}`;

function roomyLimits(): BrowserRouteLimits {
  return checkRouteLimits({ maxContexts: 4, maxRoutes: 16, maxBodyBytes: 4096 });
}

function openService(): BrowserRouteService {
  const service = new BrowserRouteService(
    [
      { owner: "owner-n", context: "ctx-a" },
      { owner: "owner-n", context: "ctx-b" },
    ],
    roomyLimits(),
  );
  service.bindContext("owner-n", "ctx-a");
  return service;
}

function holdRoute(service: BrowserRouteService, routeId = "route-1"): { token: string } {
  service.holdRoute("owner-n", "ctx-a", routeId, { method: "GET", url: "/items" });
  return { token: service.routeTokenForTest("owner-n", "ctx-a", routeId) };
}

function completeBody(
  digest: string = BODY_A,
  byteLength = 12,
): {
  bodyDigest: string;
  byteLength: number;
  complete: boolean;
} {
  return { bodyDigest: digest, byteLength, complete: true };
}

// --- Acceptance: hold/contact/resolve with immutable captured body ---------

check("hold-contact-resolve-separated", () => {
  const service = openService();
  // Every declared method holds with a stable rejoin attestation.
  assert.deepEqual([...ROUTE_METHODS], ["GET", "POST", "PUT", "DELETE", "HEAD"]);
  for (const method of ROUTE_METHODS) {
    const attestation = service.holdRoute("owner-n", "ctx-a", `rte-${method}`, {
      method,
      url: "/local/path",
    });
    assert.equal(attestation.method, method);
    assert.equal(attestation.held, true);
    assert.ok(attestation.urlDigest.startsWith("sha256:"));
    assert.ok(attestation.handleDigest.startsWith("sha256:"));
    assert.deepEqual(Object.keys(attestation).sort(), [
      "context",
      "contextDigest",
      "handleDigest",
      "held",
      "method",
      "owner",
      "routeId",
      "urlDigest",
    ]);
    const joined = service.holdRoute("owner-n", "ctx-a", `rte-${method}`, {
      method,
      url: "/local/path",
    });
    assert.equal(joined.handleDigest, attestation.handleDigest);
  }
  assert.equal(service.routeCount("owner-n", "ctx-a"), 5);

  // The contact records exactly once with the immutable captured body.
  const { token } = holdRoute(service);
  const status = service.routeFacts("owner-n", "ctx-a", "route-1");
  assert.equal(status.held, true);
  assert.equal(status.contacted, false);
  assert.equal(status.contactId, null);
  // Missing contact refuses instead of reading as an empty body.
  expectRouteError(
    () => service.contactFacts("owner-n", "ctx-a", "route-1"),
    "no-contact",
    "contact",
  );
  const contact = service.contactUpstream("owner-n", "ctx-a", "route-1", token, completeBody());
  assert.equal(contact.complete, true);
  assert.equal(contact.bodyDigest, BODY_A);
  assert.equal(contact.byteLength, 12);
  assert.ok(contact.contactId.startsWith("ctc-"));
  assert.ok(contact.digest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(contact).sort(), [
    "bodyDigest",
    "byteLength",
    "complete",
    "contactId",
    "context",
    "digest",
    "owner",
    "routeId",
  ]);
  assert.ok(Object.isFrozen(contact));
  assert.deepEqual(service.contactFacts("owner-n", "ctx-a", "route-1"), contact);

  // Every declared resolve mode resolves with the same captured body.
  assert.deepEqual([...RESOLVE_MODES], ["fulfill", "abort", "continue"]);
  for (const mode of RESOLVE_MODES) {
    const routeId = `route-${mode}`;
    service.holdRoute("owner-n", "ctx-a", routeId, { method: "GET", url: "/items" });
    const modeToken = service.routeTokenForTest("owner-n", "ctx-a", routeId);
    service.contactUpstream("owner-n", "ctx-a", routeId, modeToken, completeBody());
    const delivery = service.resolveRoute("owner-n", "ctx-a", routeId, modeToken, { mode });
    assert.equal(delivery.mode, mode);
    assert.equal(delivery.bodyDigest, BODY_A);
    assert.equal(delivery.byteLength, 12);
    assert.ok(delivery.deliveryId.startsWith("dlv-"));
    assert.ok(delivery.contactId.startsWith("ctc-"));
    assert.notEqual(delivery.deliveryId, delivery.contactId);
    assert.deepEqual(Object.keys(delivery).sort(), [
      "bodyDigest",
      "byteLength",
      "contactId",
      "context",
      "deliveryId",
      "digest",
      "mode",
      "owner",
      "routeId",
    ]);
    assert.ok(Object.isFrozen(delivery));
    assert.deepEqual(service.deliveryFacts("owner-n", "ctx-a", delivery.deliveryId), delivery);
    const settled = service.routeFacts("owner-n", "ctx-a", routeId);
    assert.equal(settled.held, false);
    assert.equal(settled.resolved, true);
    assert.equal(settled.deliveryId, delivery.deliveryId);
    assert.equal(settled.mode, mode);
  }
  assert.equal(service.deliveryCount("owner-n", "ctx-a"), 3);

  // Resolution is terminal: no second resolve, no re-hold, no attestation.
  const delivery = service.resolveRoute("owner-n", "ctx-a", "route-1", token, {
    mode: "fulfill",
  });
  assert.equal(delivery.mode, "fulfill");
  expectRouteError(
    () => service.resolveRoute("owner-n", "ctx-a", "route-1", token, { mode: "abort" }),
    "route-closed",
    "route",
  );
  expectRouteError(
    () => service.holdRoute("owner-n", "ctx-a", "route-1", { method: "GET", url: "/items" }),
    "route-closed",
    "route",
  );
  expectRouteError(
    () => service.attestation("owner-n", "ctx-a", "route-1"),
    "route-closed",
    "route",
  );
  // Contact and delivery facts stay readable after resolution.
  assert.deepEqual(service.contactFacts("owner-n", "ctx-a", "route-1"), contact);
  assert.deepEqual(service.deliveryFacts("owner-n", "ctx-a", delivery.deliveryId), delivery);
});

// --- Acceptance: duplicate upstream contact cannot pass ---------------------

check("duplicate-contact-refuses", () => {
  const service = openService();
  const { token } = holdRoute(service);
  const first = service.contactUpstream(
    "owner-n",
    "ctx-a",
    "route-1",
    token,
    completeBody(BODY_A, 12),
  );
  // A second contact with different bytes refuses at the contact layer.
  expectRouteError(
    () => service.contactUpstream("owner-n", "ctx-a", "route-1", token, completeBody(BODY_B, 12)),
    "duplicate-contact",
    "contact",
  );
  // A second contact with the same digest but a different length refuses
  // too: either field differing is a second contact, not a rejoin.
  expectRouteError(
    () => service.contactUpstream("owner-n", "ctx-a", "route-1", token, completeBody(BODY_A, 13)),
    "duplicate-contact",
    "contact",
  );
  // The recorded contact is unchanged and the journal proves one contact.
  assert.deepEqual(service.contactFacts("owner-n", "ctx-a", "route-1"), first);
  const journal = service.operationJournal("owner-n", "ctx-a", "route-1");
  assert.deepEqual(
    journal.map((entry) => entry.op),
    ["hold", "contact"],
  );
  assert.deepEqual(
    journal.map((entry) => entry.seq),
    [0, 1],
  );
  // Seeded control: a fresh route still contacts exactly once, so the
  // refusal above is per-token state, not a global block.
  service.holdRoute("owner-n", "ctx-a", "route-2", { method: "POST", url: "/submit" });
  const token2 = service.routeTokenForTest("owner-n", "ctx-a", "route-2");
  const other = service.contactUpstream(
    "owner-n",
    "ctx-a",
    "route-2",
    token2,
    completeBody(BODY_B, 7),
  );
  assert.notEqual(other.contactId, first.contactId);
  assert.equal(other.bodyDigest, BODY_B);
});

// --- Acceptance: partial body cannot pass -----------------------------------

check("partial-body-refuses-with-named-reason", () => {
  assert.deepEqual([...PARTIAL_BODY_REASONS], ["truncated", "withheld", "aborted"]);
  const service = openService();
  const { token } = holdRoute(service);
  // Every declared reason refuses at the contact layer and names itself.
  for (const reason of PARTIAL_BODY_REASONS) {
    try {
      service.contactUpstream("owner-n", "ctx-a", "route-1", token, {
        bodyDigest: BODY_A,
        byteLength: 12,
        complete: false,
        reason,
      });
      assert.fail(`expected partial-body for ${reason}`);
    } catch (error) {
      assert.ok(error instanceof BrowserRouteError);
      assert.equal(error.code, "partial-body");
      assert.equal(error.layer, "contact");
      assert.ok(error.message.includes(reason));
    }
  }
  // No contact recorded after any partial attempt: the missing evidence
  // refuses rather than reading as an empty body.
  expectRouteError(
    () => service.contactFacts("owner-n", "ctx-a", "route-1"),
    "no-contact",
    "contact",
  );
  assert.equal(service.routeFacts("owner-n", "ctx-a", "route-1").contacted, false);
  assert.deepEqual(
    service.operationJournal("owner-n", "ctx-a", "route-1").map((entry) => entry.op),
    ["hold"],
  );
  // Malformed bodies refuse without effect, before any contact records.
  for (const bad of [
    null,
    "digest-string",
    { bodyDigest: BODY_A, byteLength: 12 },
    { bodyDigest: BODY_A, byteLength: 12, complete: true, reason: "truncated" },
    { bodyDigest: BODY_A, byteLength: 12, complete: false },
    { bodyDigest: BODY_A, byteLength: 12, complete: false, reason: "dropped" },
    { bodyDigest: BODY_A, byteLength: 12, complete: false, reason: "" },
    { bodyDigest: "not-a-digest", byteLength: 12, complete: true },
    { bodyDigest: BODY_A, byteLength: -1, complete: true },
    { bodyDigest: BODY_A, byteLength: 1.5, complete: true },
    { bodyDigest: BODY_A, byteLength: 12, complete: "yes" },
    { bodyDigest: BODY_A, byteLength: 12, complete: true, bytes: "raw" },
  ]) {
    expectRouteError(
      () => service.contactUpstream("owner-n", "ctx-a", "route-1", token, bad),
      "unknown-body",
      "contact",
    );
  }
  // A partial attempt never poisons the token: the complete body still
  // records exactly once afterwards.
  const contact = service.contactUpstream(
    "owner-n",
    "ctx-a",
    "route-1",
    token,
    completeBody(BODY_A, 12),
  );
  assert.equal(contact.complete, true);
  assert.equal(service.routeFacts("owner-n", "ctx-a", "route-1").contacted, true);
});

// --- Acceptance: unknown delivery cannot pass -------------------------------

check("unknown-delivery-refuses", () => {
  const service = openService();
  const { token } = holdRoute(service);
  const contact = service.contactUpstream("owner-n", "ctx-a", "route-1", token, completeBody());
  const delivery = service.resolveRoute("owner-n", "ctx-a", "route-1", token, {
    mode: "fulfill",
  });
  // Positive control: the minted delivery reads back identically.
  assert.deepEqual(service.deliveryFacts("owner-n", "ctx-a", delivery.deliveryId), delivery);
  // Unknown delivery ids refuse at the delivery layer.
  for (const ghost of ["dlv-ghost", "delivery-9", delivery.deliveryId.slice(0, -1).concat("x")]) {
    expectRouteError(
      () => service.deliveryFacts("owner-n", "ctx-a", ghost),
      "unknown-delivery",
      "delivery",
    );
  }
  // A contact id is never a delivery id: the namespaces are disjoint.
  expectRouteError(
    () => service.deliveryFacts("owner-n", "ctx-a", contact.contactId),
    "unknown-delivery",
    "delivery",
  );
  // Malformed delivery ids refuse the same way.
  for (const bad of ["", "has space", "dlv-*"]) {
    expectRouteError(
      () => service.deliveryFacts("owner-n", "ctx-a", bad),
      "unknown-delivery",
      "delivery",
    );
  }
  // Seeded missing-evidence control: a held route has no delivery to
  // read, and resolving out of order refuses before any delivery mints.
  holdRoute(service, "route-held");
  expectRouteError(
    () => service.deliveryFacts("owner-n", "ctx-a", "dlv-route-held"),
    "unknown-delivery",
    "delivery",
  );
  const heldToken = service.routeTokenForTest("owner-n", "ctx-a", "route-held");
  expectRouteError(
    () => service.resolveRoute("owner-n", "ctx-a", "route-held", heldToken, { mode: "fulfill" }),
    "no-contact",
    "contact",
  );
  // Resolve modes outside the vocabulary refuse before any delivery mints.
  for (const bad of ["retry", "", "FULFILL", null]) {
    expectRouteError(
      () => service.resolveRoute("owner-n", "ctx-a", "route-held", heldToken, { mode: bad }),
      "unknown-resolve-mode",
      "delivery",
    );
  }
  // A resolve descriptor carrying a body refuses: deliveries reuse the
  // immutable captured body and never accept new bytes.
  for (const bad of [
    null,
    { mode: "fulfill", bodyDigest: BODY_B },
    { mode: "fulfill", byteLength: 3 },
    { mode: "fulfill", contactId: contact.contactId },
  ]) {
    expectRouteError(
      () => service.resolveRoute("owner-n", "ctx-a", "route-held", heldToken, bad),
      "unknown-resolve-mode",
      "delivery",
    );
  }
  assert.equal(service.routeFacts("owner-n", "ctx-a", "route-held").resolved, false);
  assert.equal(service.deliveryCount("owner-n", "ctx-a"), 1);
});

// --- Acceptance: repeated native operation joins ----------------------------

check("repeated-operation-joins-without-recontact", () => {
  assert.deepEqual([...ROUTE_JOURNAL_OPS], ["hold", "contact", "resolve"]);
  const service = openService();
  const { token } = holdRoute(service);
  // Re-holding the identical descriptor rejoins with a stable digest.
  const first = service.attestation("owner-n", "ctx-a", "route-1");
  const rejoined = service.holdRoute("owner-n", "ctx-a", "route-1", {
    method: "GET",
    url: "/items",
  });
  assert.equal(rejoined.handleDigest, first.handleDigest);
  // Re-holding with a different descriptor refuses rather than
  // retargeting the held route.
  expectRouteError(
    () => service.holdRoute("owner-n", "ctx-a", "route-1", { method: "POST", url: "/items" }),
    "route-mismatch",
    "route",
  );
  expectRouteError(
    () => service.holdRoute("owner-n", "ctx-a", "route-1", { method: "GET", url: "/other" }),
    "route-mismatch",
    "route",
  );
  assert.deepEqual(
    service.operationJournal("owner-n", "ctx-a", "route-1").map((entry) => entry.op),
    ["hold"],
  );
  // The identical native contact re-issue returns the same recorded
  // result: same contact id, same digest, no journal growth.
  const contact = service.contactUpstream(
    "owner-n",
    "ctx-a",
    "route-1",
    token,
    completeBody(BODY_A, 12),
  );
  const expectedDigest = digestContact(
    "route-1",
    contact.contactId,
    "owner-n",
    "ctx-a",
    BODY_A,
    12,
  );
  assert.equal(contact.digest, expectedDigest);
  const joined = service.contactUpstream(
    "owner-n",
    "ctx-a",
    "route-1",
    token,
    completeBody(BODY_A, 12),
  );
  assert.deepEqual(joined, contact);
  assert.equal(joined.contactId, contact.contactId);
  assert.deepEqual(
    service.operationJournal("owner-n", "ctx-a", "route-1").map((entry) => entry.op),
    ["hold", "contact"],
  );
  // The delivery digest recomputes from carried fields too.
  const delivery = service.resolveRoute("owner-n", "ctx-a", "route-1", token, {
    mode: "fulfill",
  });
  assert.equal(
    delivery.digest,
    digestDelivery(
      "route-1",
      delivery.deliveryId,
      contact.contactId,
      "owner-n",
      "ctx-a",
      "fulfill",
      BODY_A,
      12,
    ),
  );
  assert.deepEqual(
    service.operationJournal("owner-n", "ctx-a", "route-1").map((entry) => entry.op),
    ["hold", "contact", "resolve"],
  );
});

// --- Route identity: tokens, methods, and local-only urls -------------------

check("route-identity-tokens-and-vocabulary", () => {
  const service = openService();
  const { token } = holdRoute(service);
  // Methods outside the finite vocabulary reject before any effect.
  for (const bad of ["FETCH", "", "get", "GET "]) {
    expectRouteError(
      () => service.holdRoute("owner-n", "ctx-a", "route-bad", { method: bad, url: "/x" }),
      "unknown-route",
      "route",
    );
  }
  // Live hosts never route: only bounded local paths hold.
  for (const bad of [
    "https://example.com/items",
    "http://localhost:9/items",
    "//example.com/items",
    "items",
    "",
  ]) {
    expectRouteError(
      () => service.holdRoute("owner-n", "ctx-a", "route-bad", { method: "GET", url: bad }),
      "unknown-route",
      "route",
    );
  }
  // Malformed descriptors reject without effect.
  for (const bad of [
    null,
    "GET /items",
    { method: "GET" },
    { method: "GET", url: "/x", extra: 1 },
    { method: "GET", url: "/bad url" },
  ]) {
    expectRouteError(
      () => service.holdRoute("owner-n", "ctx-a", "route-bad", bad),
      "unknown-route",
      "route",
    );
  }
  assert.equal(service.routeCount("owner-n", "ctx-a"), 1);
  // Invented tokens are never authority, on contact or resolve.
  for (const forged of ["", "rte-deadbeef", `${token.slice(0, -1)}x`]) {
    expectRouteError(
      () => service.contactUpstream("owner-n", "ctx-a", "route-1", forged, completeBody()),
      "forged-token",
      "route",
    );
    expectRouteError(
      () => service.resolveRoute("owner-n", "ctx-a", "route-1", forged, { mode: "fulfill" }),
      "forged-token",
      "route",
    );
  }
  // Unknown routes reject before the token is even read.
  expectRouteError(
    () => service.contactUpstream("owner-n", "ctx-a", "route-9", token, completeBody()),
    "unknown-route",
    "route",
  );
  expectRouteError(
    () => service.resolveRoute("owner-n", "ctx-a", "route-9", token, { mode: "fulfill" }),
    "unknown-route",
    "route",
  );
  // Tokens never transfer between routes.
  service.holdRoute("owner-n", "ctx-a", "route-2", { method: "GET", url: "/other" });
  const token2 = service.routeTokenForTest("owner-n", "ctx-a", "route-2");
  assert.notEqual(token2, token);
  expectRouteError(
    () => service.contactUpstream("owner-n", "ctx-a", "route-2", token, completeBody()),
    "forged-token",
    "route",
  );
  expectRouteError(
    () => service.contactUpstream("owner-n", "ctx-a", "route-1", token2, completeBody()),
    "forged-token",
    "route",
  );
  // The live tokens contact and resolve under their own bindings.
  service.contactUpstream("owner-n", "ctx-a", "route-1", token, completeBody());
  service.contactUpstream("owner-n", "ctx-a", "route-2", token2, completeBody());
  service.resolveRoute("owner-n", "ctx-a", "route-1", token, { mode: "fulfill" });
  service.resolveRoute("owner-n", "ctx-a", "route-2", token2, { mode: "abort" });
});

// --- Admission-first and owner checks ----------------------------------------

check("missing-context-blocks-hold-contact-resolve", () => {
  const service = new BrowserRouteService([{ owner: "owner-n", context: "ctx-a" }], roomyLimits());
  const descriptor = { method: "GET", url: "/x" };
  // Never bound: hold refuses without a live binding, while contact
  // and resolve refuse through the context read path.
  expectRouteError(
    () => service.holdRoute("owner-n", "ctx-a", "route-1", descriptor),
    "no-context",
    "route",
  );
  expectRouteError(
    () => service.contactUpstream("owner-n", "ctx-a", "route-1", "rte-x", completeBody()),
    "unknown-context",
    "route",
  );
  expectRouteError(
    () => service.resolveRoute("owner-n", "ctx-a", "route-1", "rte-x", { mode: "fulfill" }),
    "unknown-context",
    "route",
  );
  // Ungranted contexts refuse the same way.
  expectRouteError(
    () => service.holdRoute("owner-n", "elsewhere", "route-1", descriptor),
    "no-context",
    "route",
  );
  expectRouteError(
    () => service.contactUpstream("owner-n", "elsewhere", "route-1", "rte-x", completeBody()),
    "unknown-context",
    "route",
  );
  service.bindContext("owner-n", "ctx-a");
  expectRouteError(
    () => service.holdRoute("owner-x", "ctx-a", "route-1", descriptor),
    "no-context",
    "route",
  );
  expectRouteError(
    () => service.contactUpstream("owner-x", "ctx-a", "route-1", "rte-x", completeBody()),
    "wrong-owner",
    "route",
  );
  // A second identity granted the same context name cannot join it:
  // contexts bind to exactly one owner.
  const shared = new BrowserRouteService(
    [
      { owner: "owner-n", context: "ctx-a" },
      { owner: "owner-x", context: "ctx-a" },
    ],
    roomyLimits(),
  );
  shared.bindContext("owner-n", "ctx-a");
  expectRouteError(() => shared.bindContext("owner-x", "ctx-a"), "wrong-owner", "route");
  // Live binding: hold, contact, and resolve admit together.
  service.holdRoute("owner-n", "ctx-a", "route-1", descriptor);
  const token = service.routeTokenForTest("owner-n", "ctx-a", "route-1");
  const contact = service.contactUpstream("owner-n", "ctx-a", "route-1", token, completeBody());
  const delivery = service.resolveRoute("owner-n", "ctx-a", "route-1", token, {
    mode: "continue",
  });
  // Foreign owners fail contact, resolve, and reads.
  expectRouteError(
    () => service.contactUpstream("owner-x", "ctx-a", "route-1", token, completeBody()),
    "wrong-owner",
    "route",
  );
  expectRouteError(
    () => service.resolveRoute("owner-x", "ctx-a", "route-1", token, { mode: "fulfill" }),
    "wrong-owner",
    "route",
  );
  expectRouteError(() => service.routeFacts("owner-x", "ctx-a", "route-1"), "wrong-owner", "route");
  expectRouteError(
    () => service.contactFacts("owner-x", "ctx-a", "route-1"),
    "wrong-owner",
    "route",
  );
  expectRouteError(
    () => service.deliveryFacts("owner-x", "ctx-a", delivery.deliveryId),
    "wrong-owner",
    "route",
  );
  assert.ok(contact.contactId.startsWith("ctc-"));
});

// --- Close gates: held routes block ------------------------------------------

check("close-gates-on-held-routes", () => {
  const service = openService();
  const { token } = holdRoute(service);
  // A held route blocks the close and names the route, even uncontacted.
  try {
    service.closeContext("owner-n", "ctx-a");
    assert.fail("expected route-pending");
  } catch (error) {
    assert.ok(error instanceof BrowserRouteError);
    assert.equal(error.code, "route-pending");
    assert.ok(String(error).includes("route-1"));
  }
  // Contact alone does not unblock: only resolution does.
  service.contactUpstream("owner-n", "ctx-a", "route-1", token, completeBody());
  expectRouteError(() => service.closeContext("owner-n", "ctx-a"), "route-pending", "route");
  service.resolveRoute("owner-n", "ctx-a", "route-1", token, { mode: "fulfill" });
  service.closeContext("owner-n", "ctx-a");
  // After the close, admission refuses and facts report closed.
  expectRouteError(
    () => service.holdRoute("owner-n", "ctx-a", "route-2", { method: "GET", url: "/x" }),
    "no-context",
    "route",
  );
  expectRouteError(
    () => service.routeFacts("owner-n", "ctx-a", "route-1"),
    "context-closed",
    "route",
  );
});

// --- Vocabulary, scope statement, and capacity --------------------------------

check("every-failure-names-its-layer", () => {
  // The code vocabulary is closed and every code maps to exactly one layer.
  assert.equal(BROWSER_ROUTE_CODES.length, 16);
  assert.equal(BROWSER_ROUTE_SCHEMA_VERSION, "1");
  for (const code of BROWSER_ROUTE_CODES) {
    const layer = layerOfRouteCode(code);
    assert.ok(["route", "contact", "delivery"].includes(layer));
    const error = new BrowserRouteError(code, "probe");
    assert.equal(error.layer, layer);
    assert.equal(error.code, code);
    assert.ok(error.message.startsWith(`[${layer}:${code}]`));
  }
  // Limit shapes reject at the route layer with nothing constructed.
  expectRouteError(() => checkRouteLimits(null), "unknown-context", "route");
  expectRouteError(() => checkRouteLimits({ maxContexts: 1 }), "unknown-context", "route");
  expectRouteError(
    () => checkRouteLimits({ maxContexts: 1, maxRoutes: 1, maxBodyBytes: 1, maxBrowsers: 1 }),
    "unknown-context",
    "route",
  );
  // Empty and duplicate declarations never construct.
  expectRouteError(() => new BrowserRouteService([], roomyLimits()), "unknown-context", "route");
  expectRouteError(
    () =>
      new BrowserRouteService(
        [
          { owner: "o", context: "c" },
          { owner: "o", context: "c" },
        ],
        roomyLimits(),
      ),
    "unknown-context",
    "route",
  );
  // Capacity faults name the route layer.
  const tiny = new BrowserRouteService(
    [{ owner: "o", context: "c" }],
    checkRouteLimits({ maxContexts: 1, maxRoutes: 1, maxBodyBytes: 8 }),
  );
  tiny.bindContext("o", "c");
  tiny.holdRoute("o", "c", "r1", { method: "GET", url: "/x" });
  expectRouteError(
    () => tiny.holdRoute("o", "c", "r2", { method: "GET", url: "/y" }),
    "capacity-exhausted",
    "route",
  );
  const tinyToken = tiny.routeTokenForTest("o", "c", "r1");
  expectRouteError(
    () => tiny.contactUpstream("o", "c", "r1", tinyToken, completeBody(BODY_A, 9)),
    "capacity-exhausted",
    "route",
  );
  assert.equal(tiny.routeFacts("o", "c", "r1").contacted, false);
  // Facts carry digests only: raw tokens, urls, and reasons never cross.
  const service = openService();
  const { token } = holdRoute(service);
  const facts = service.routeFacts("owner-n", "ctx-a", "route-1");
  assert.ok(!JSON.stringify(facts).includes(token));
  assert.ok(!JSON.stringify(facts).includes("/items"));
});

check("independence-scope-stated", () => {
  // The scope statement is a fixed export: the service shares no driver
  // code, decoded values, receipts, flags, environment, or process state,
  // and proves its facts from its own seeded doubles alone.
  assert.ok(BROWSER_ROUTE_INDEPENDENCE_SCOPE.startsWith("independent:"));
  assert.ok(BROWSER_ROUTE_INDEPENDENCE_SCOPE.includes("no driver code"));
  assert.ok(BROWSER_ROUTE_INDEPENDENCE_SCOPE.includes("seeded doubles"));
  assert.ok(BROWSER_ROUTE_INDEPENDENCE_SCOPE.includes("one-contact"));
});

console.log(
  JSON.stringify({
    kind: "can.browser-driver-route-check",
    schema_version: BROWSER_ROUTE_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
