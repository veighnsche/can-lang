// K16 bounded self-check: owned remote contexts, a preserved server,
// confirmed closes after worker death.
//
// Run with: bun tools/runtime/test-services/browser-driver/remote-check.ts
// Local controls only. No browsers, processes, flags, environment,
// sampling, networks, files, timers, live hosts, or live runtimes. Every
// loop below is bounded by a small constant. Live host-dependent controls
// wait for the corresponding qualified profile and Q task.

import { strict as assert } from "node:assert";
import {
  assertRemoteCleanupWitnessed,
  BROWSER_REMOTE_CODES,
  BROWSER_REMOTE_INDEPENDENCE_SCOPE,
  BROWSER_REMOTE_SCHEMA_VERSION,
  BrowserRemoteError,
  BrowserRemoteService,
  checkRemoteLimits,
  FORBIDDEN_CLOSE_KINDS,
  layerOfRemoteCode,
  type BrowserRemoteCode,
  type BrowserRemoteLayer,
  type BrowserRemoteLimits,
  type CloseFacts,
  type ExternalCloseAuthority,
} from "./remote.ts";

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectRemoteError(
  body: () => unknown,
  code: BrowserRemoteCode,
  layer: BrowserRemoteLayer,
): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof BrowserRemoteError)) {
      assert.fail(`expected a BrowserRemoteError, got ${String(error)}`);
    }
    assert.equal(error.code, code);
    assert.equal(error.layer, layer);
    assert.ok(error.message.includes(`[${layer}:${code}]`));
    return;
  }
  assert.fail(`expected failure ${layer}:${code}`);
}

// Seeded independent double: proves close confirmation from its own
// seeded table alone. Close tokens are verified by lookup; invented
// tokens and cross-context tokens never verify.
class FakeCloseAuthority implements ExternalCloseAuthority {
  readonly authority: string =
    "external: seeded remote-context close double; proves confirmation " +
    "from its own seeded table alone";
  private readonly table = new Map<string, { token: string; closed: boolean }>();
  private seq = 0;

  seedClose(contextId: string, closed = true): string {
    if (this.table.size >= 8) {
      throw new Error("fake close table full");
    }
    this.seq += 1;
    const token = `cls-seed-${this.seq}-${contextId}`;
    this.table.set(contextId, { token, closed });
    return token;
  }

  verifyClose(contextId: string, token: string): CloseFacts {
    for (const [id, entry] of this.table) {
      if (entry.token === token) {
        if (id !== contextId) {
          throw new BrowserRemoteError("forged-close", "close is for another context");
        }
        return Object.freeze({
          contextId: id,
          closed: entry.closed,
          digest: `fake:${id}:${entry.closed ? "closed" : "open"}`,
        });
      }
    }
    throw new BrowserRemoteError("forged-close", "close token is unknown");
  }
}

function roomyLimits(): BrowserRemoteLimits {
  return checkRemoteLimits({ maxServers: 4, maxConnections: 8, maxContextsPerConnection: 8 });
}

function openService(): { service: BrowserRemoteService; authority: FakeCloseAuthority } {
  const authority = new FakeCloseAuthority();
  const service = new BrowserRemoteService(
    [
      { owner: "owner-n", server: "firefox-server" },
      { owner: "owner-n", server: "chromium-server" },
    ],
    roomyLimits(),
    authority,
  );
  return { service, authority };
}

function openConnection(): {
  service: BrowserRemoteService;
  authority: FakeCloseAuthority;
  connToken: string;
} {
  const { service, authority } = openService();
  service.connect("owner-n", "firefox-server", "conn-1");
  const connToken = service.connectionTokenForTest("owner-n", "conn-1");
  return { service, authority, connToken };
}

function openContext(): {
  service: BrowserRemoteService;
  authority: FakeCloseAuthority;
  connToken: string;
  ctxToken: string;
} {
  const { service, authority, connToken } = openConnection();
  service.openContext("owner-n", "conn-1", connToken, "ctx-1");
  const ctxToken = service.contextTokenForTest("owner-n", "conn-1", "ctx-1");
  return { service, authority, connToken, ctxToken };
}

// --- Shared-server admission ------------------------------------------------

check("server-admitted-and-live", () => {
  const { service } = openService();
  const facts = service.serverFacts("firefox-server");
  assert.equal(facts.server, "firefox-server");
  assert.equal(facts.owner, "owner-n");
  assert.equal(facts.live, true);
  assert.deepEqual([...facts.connections], []);
  assert.ok(facts.digest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(facts).sort(), ["connections", "digest", "live", "owner", "server"]);
  // Foreign and malformed servers refuse before any effect.
  for (const server of ["safari-server", "", "firefox-server ", "FIREFOX-SERVER"]) {
    expectRemoteError(() => service.serverFacts(server), "unknown-server", "service");
    expectRemoteError(
      () => service.connect("owner-n", "firefox-server-x", "conn-9"),
      "unknown-server",
      "service",
    );
  }
  assert.equal(service.connectionCount, 0);
  // Empty and duplicate declarations never construct.
  const authority = new FakeCloseAuthority();
  expectRemoteError(
    () => new BrowserRemoteService([], roomyLimits(), authority),
    "unknown-server",
    "service",
  );
  expectRemoteError(
    () =>
      new BrowserRemoteService(
        [
          { owner: "o", server: "s" },
          { owner: "o", server: "s" },
        ],
        roomyLimits(),
        authority,
      ),
    "unknown-server",
    "service",
  );
});

// --- Acceptance: a case close-server attempt rejects before effect -----------

check("close-server-rejects-before-effect", () => {
  const { service, connToken, ctxToken } = openContext();
  service.openContext("owner-n", "conn-1", connToken, "ctx-2");
  const before = service.serverFacts("firefox-server");
  const connBefore = service.connectionFacts("owner-n", "conn-1");
  const ctxBefore = service.contextFacts("owner-n", "conn-1", "ctx-1");
  // The attempt rejects at the remote layer.
  expectRemoteError(
    () => service.closeServer("owner-n", "firefox-server"),
    "forbidden-server-close",
    "remote",
  );
  // The refusal precedes every mutation: the server is provably live and
  // untouched, down to the digest.
  const after = service.serverFacts("firefox-server");
  assert.deepEqual(after, before);
  assert.equal(after.live, true);
  assert.deepEqual([...after.connections], ["conn-1"]);
  assert.deepEqual(service.connectionFacts("owner-n", "conn-1"), connBefore);
  assert.deepEqual(service.contextFacts("owner-n", "conn-1", "ctx-1"), ctxBefore);
  assert.equal(service.connectionCount, 1);
  // Server ids are not contexts: cleanup paths aimed at a server name
  // refuse without touching the server.
  expectRemoteError(
    () => service.requestContextClose("owner-n", "conn-1", "firefox-server", ctxToken),
    "unknown-context",
    "remote",
  );
  expectRemoteError(
    () => service.sealContextClose("owner-n", "conn-1", "firefox-server", ctxToken, "cls-seed-1-x"),
    "unknown-context",
    "remote",
  );
  assert.deepEqual(service.serverFacts("firefox-server"), before);
  // A foreign owner and an undeclared server refuse too, before effect.
  expectRemoteError(
    () => service.closeServer("owner-x", "firefox-server"),
    "forbidden-server-close",
    "remote",
  );
  expectRemoteError(() => service.closeServer("owner-n", "elsewhere"), "unknown-server", "service");
  assert.deepEqual(service.serverFacts("firefox-server"), before);
});

// --- Owned connections and contexts, separate from the server ----------------

check("connection-context-owned-separately", () => {
  const { service } = openService();
  const attestation = service.connect("owner-n", "firefox-server", "conn-1");
  assert.equal(attestation.connectionId, "conn-1");
  assert.equal(attestation.owner, "owner-n");
  assert.equal(attestation.server, "firefox-server");
  assert.ok(attestation.handleDigest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(attestation).sort(), [
    "connectionId",
    "handleDigest",
    "owner",
    "server",
  ]);
  // Reconnecting re-attests; the digest is stable.
  const joined = service.connect("owner-n", "firefox-server", "conn-1");
  assert.equal(joined.handleDigest, attestation.handleDigest);
  // The id never transfers between owners or servers.
  expectRemoteError(
    () => service.connect("owner-x", "firefox-server", "conn-1"),
    "wrong-owner",
    "service",
  );
  expectRemoteError(
    () => service.connect("owner-n", "chromium-server", "conn-1"),
    "unknown-connection",
    "remote",
  );
  const connToken = service.connectionTokenForTest("owner-n", "conn-1");
  const ctxAttestation = service.openContext("owner-n", "conn-1", connToken, "ctx-1");
  assert.equal(ctxAttestation.contextId, "ctx-1");
  assert.equal(ctxAttestation.connectionId, "conn-1");
  assert.equal(ctxAttestation.server, "firefox-server");
  assert.ok(ctxAttestation.handleDigest.startsWith("sha256:"));
  assert.equal(ctxAttestation.connectionDigest, attestation.handleDigest);
  assert.deepEqual(Object.keys(ctxAttestation).sort(), [
    "connectionDigest",
    "connectionId",
    "contextId",
    "handleDigest",
    "owner",
    "server",
  ]);
  // The context token is opaque, distinct, and digest-only in facts.
  const ctxToken = service.contextTokenForTest("owner-n", "conn-1", "ctx-1");
  assert.notEqual(ctxToken, connToken);
  assert.ok(!JSON.stringify(ctxAttestation).includes(ctxToken));
  assert.ok(!JSON.stringify(attestation).includes(connToken));
  // Reopening joins the same attestation.
  const again = service.openContext("owner-n", "conn-1", connToken, "ctx-1");
  assert.equal(again.handleDigest, ctxAttestation.handleDigest);
  // The server lists the connection but owns neither token.
  const server = service.serverFacts("firefox-server");
  assert.deepEqual([...server.connections], ["conn-1"]);
  assert.ok(!JSON.stringify(server).includes(connToken));
  const facts = service.connectionFacts("owner-n", "conn-1");
  assert.deepEqual([...facts.contexts], ["ctx-1"]);
  assert.equal(facts.workerDeathNoted, false);
});

check("remote-tokens-opaque-and-verified", () => {
  const { service, connToken, ctxToken } = openContext();
  // Invented tokens are never authority, on any token-bearing call.
  for (const forged of ["", "conn-deadbeef", `${connToken.slice(0, -1)}x`]) {
    expectRemoteError(
      () => service.openContext("owner-n", "conn-1", forged, "ctx-2"),
      "forged-token",
      "remote",
    );
    expectRemoteError(
      () => service.noteWorkerDeath("owner-n", "conn-1", forged),
      "forged-token",
      "remote",
    );
  }
  for (const forged of ["", "ctx-deadbeef", `${ctxToken.slice(0, -1)}x`]) {
    expectRemoteError(
      () => service.requestContextClose("owner-n", "conn-1", "ctx-1", forged),
      "forged-token",
      "remote",
    );
    expectRemoteError(
      () => service.loseCloseConfirmation("owner-n", "conn-1", "ctx-1", forged),
      "forged-token",
      "remote",
    );
    expectRemoteError(
      () => service.sealContextClose("owner-n", "conn-1", "ctx-1", forged, "cls-seed-1-ctx-1"),
      "forged-token",
      "remote",
    );
  }
  // Connection and context tokens never substitute for each other.
  expectRemoteError(
    () => service.openContext("owner-n", "conn-1", ctxToken, "ctx-2"),
    "forged-token",
    "remote",
  );
  expectRemoteError(
    () => service.requestContextClose("owner-n", "conn-1", "ctx-1", connToken),
    "forged-token",
    "remote",
  );
  // Unknown connections and contexts reject before the token is read.
  expectRemoteError(
    () => service.openContext("owner-n", "conn-9", connToken, "ctx-2"),
    "unknown-connection",
    "remote",
  );
  expectRemoteError(
    () => service.requestContextClose("owner-n", "conn-1", "ctx-9", ctxToken),
    "unknown-context",
    "remote",
  );
  // Foreign owners fail every check.
  expectRemoteError(
    () => service.openContext("owner-x", "conn-1", connToken, "ctx-2"),
    "wrong-owner",
    "service",
  );
  expectRemoteError(
    () => service.contextFacts("owner-x", "conn-1", "ctx-1"),
    "wrong-owner",
    "service",
  );
  // Tokens never transfer between connections or contexts.
  service.connect("owner-n", "firefox-server", "conn-2");
  const connToken2 = service.connectionTokenForTest("owner-n", "conn-2");
  assert.notEqual(connToken2, connToken);
  expectRemoteError(
    () => service.openContext("owner-n", "conn-2", connToken, "ctx-9"),
    "forged-token",
    "remote",
  );
  service.openContext("owner-n", "conn-1", connToken, "ctx-2");
  const ctxToken2 = service.contextTokenForTest("owner-n", "conn-1", "ctx-2");
  expectRemoteError(
    () => service.requestContextClose("owner-n", "conn-1", "ctx-1", ctxToken2),
    "forged-token",
    "remote",
  );
});

// --- Worker death: opens stop, confirmed cleanup proceeds -------------------

check("worker-death-noted-and-gated", () => {
  const { service, authority, connToken, ctxToken } = openContext();
  // Noting death is idempotent: the fact sticks.
  service.noteWorkerDeath("owner-n", "conn-1", connToken);
  service.noteWorkerDeath("owner-n", "conn-1", connToken);
  const facts = service.connectionFacts("owner-n", "conn-1");
  assert.equal(facts.workerDeathNoted, true);
  // A dead worker opens nothing further.
  expectRemoteError(
    () => service.openContext("owner-n", "conn-1", connToken, "ctx-2"),
    "worker-dead",
    "remote",
  );
  // But the owned context cleans through the independent authority:
  // request, then the confirmed seal.
  const request = service.requestContextClose("owner-n", "conn-1", "ctx-1", ctxToken);
  assert.equal(request.closeState, "close-pending");
  assert.equal(request.closed, false);
  assert.deepEqual(Object.keys(request).sort(), [
    "closeState",
    "closed",
    "connectionId",
    "contextId",
    "digest",
    "handleDigest",
    "owner",
  ]);
  const closeToken = authority.seedClose("ctx-1");
  const receipt = service.sealContextClose("owner-n", "conn-1", "ctx-1", ctxToken, closeToken);
  assert.equal(receipt.contextId, "ctx-1");
  assert.equal(receipt.connectionId, "conn-1");
  assert.equal(receipt.server, "firefox-server");
  assert.equal(receipt.workerDeathNoted, true);
  assert.ok(receipt.digest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(receipt).sort(), [
    "connectionId",
    "contextId",
    "digest",
    "owner",
    "server",
    "workerDeathNoted",
  ]);
  // The receipt is durable and the server still stands.
  assert.deepEqual(service.closeReceipt("owner-n", "conn-1", "ctx-1"), receipt);
  assert.equal(service.serverFacts("firefox-server").live, true);
  const ctxFacts = service.contextFacts("owner-n", "conn-1", "ctx-1");
  assert.equal(ctxFacts.closeState, "closed");
  assert.equal(ctxFacts.closed, true);
  // After the seal every further close mutation refuses.
  expectRemoteError(
    () => service.sealContextClose("owner-n", "conn-1", "ctx-1", ctxToken, closeToken),
    "already-closed",
    "cleanup",
  );
  expectRemoteError(
    () => service.requestContextClose("owner-n", "conn-1", "ctx-1", ctxToken),
    "already-closed",
    "cleanup",
  );
  expectRemoteError(
    () => service.loseCloseConfirmation("owner-n", "conn-1", "ctx-1", ctxToken),
    "already-closed",
    "cleanup",
  );
});

// --- Acceptance: a lost close confirmation remains unresolved ----------------

check("lost-confirmation-remains-unresolved", () => {
  const { service, authority, ctxToken } = openContext();
  service.requestContextClose("owner-n", "conn-1", "ctx-1", ctxToken);
  const closeToken = authority.seedClose("ctx-1");
  // The confirmation is lost: the close stays explicitly unresolved.
  service.loseCloseConfirmation("owner-n", "conn-1", "ctx-1", ctxToken);
  const facts = service.contextFacts("owner-n", "conn-1", "ctx-1");
  assert.equal(facts.closeState, "unknown");
  assert.equal(facts.closed, false);
  // A lost confirmation is never treated as closed: no later seal reads
  // it as closed, even with the authority's own token.
  expectRemoteError(
    () => service.sealContextClose("owner-n", "conn-1", "ctx-1", ctxToken, closeToken),
    "close-unconfirmed",
    "cleanup",
  );
  // No receipt exists for an unresolved close.
  expectRemoteError(
    () => service.closeReceipt("owner-n", "conn-1", "ctx-1"),
    "close-unconfirmed",
    "cleanup",
  );
  // A second request cannot re-pending an unresolved close.
  expectRemoteError(
    () => service.requestContextClose("owner-n", "conn-1", "ctx-1", ctxToken),
    "close-unconfirmed",
    "cleanup",
  );
  // The context is still explicitly pending/unknown afterwards.
  const later = service.contextFacts("owner-n", "conn-1", "ctx-1");
  assert.equal(later.closeState, "unknown");
  assert.equal(later.closed, false);
  // Losing without a pending request refuses: nothing was awaited.
  service.openContext(
    "owner-n",
    "conn-1",
    service.connectionTokenForTest("owner-n", "conn-1"),
    "ctx-2",
  );
  const ctxToken2 = service.contextTokenForTest("owner-n", "conn-1", "ctx-2");
  expectRemoteError(
    () => service.loseCloseConfirmation("owner-n", "conn-1", "ctx-2", ctxToken2),
    "close-unconfirmed",
    "cleanup",
  );
  // A seal without a request refuses the same way: a request is never
  // itself a close, and a close is never assumed.
  const unrequested = authority.seedClose("ctx-2");
  expectRemoteError(
    () => service.sealContextClose("owner-n", "conn-1", "ctx-2", ctxToken2, unrequested),
    "close-unconfirmed",
    "cleanup",
  );
  expectRemoteError(
    () => service.closeReceipt("owner-n", "conn-1", "ctx-2"),
    "orphan-open",
    "cleanup",
  );
});

// --- The independent seal: lookup, pending, and verdict ----------------------

check("seal-needs-independent-confirmation", () => {
  const { service, authority, ctxToken } = openContext();
  service.requestContextClose("owner-n", "conn-1", "ctx-1", ctxToken);
  // An authority reporting the context still open seals nothing.
  const open = authority.seedClose("ctx-1", false);
  expectRemoteError(
    () => service.sealContextClose("owner-n", "conn-1", "ctx-1", ctxToken, open),
    "orphan-open",
    "cleanup",
  );
  // Invented close tokens are never authority.
  for (const forged of ["", "cls-seed-99-ctx-1", `${open.slice(0, -1)}x`]) {
    expectRemoteError(
      () => service.sealContextClose("owner-n", "conn-1", "ctx-1", ctxToken, forged),
      "forged-close",
      "cleanup",
    );
  }
  // Close tokens never transfer between contexts.
  const { service: second, authority: secondAuthority } = openService();
  second.connect("owner-n", "firefox-server", "conn-1");
  const secondConn = second.connectionTokenForTest("owner-n", "conn-1");
  second.openContext("owner-n", "conn-1", secondConn, "ctx-2");
  const secondCtx = second.contextTokenForTest("owner-n", "conn-1", "ctx-2");
  second.requestContextClose("owner-n", "conn-1", "ctx-2", secondCtx);
  const other = secondAuthority.seedClose("ctx-2");
  expectRemoteError(
    () => service.sealContextClose("owner-n", "conn-1", "ctx-1", ctxToken, other),
    "forged-close",
    "cleanup",
  );
  // A lying verifier fails the shape check.
  const liar = new BrowserRemoteService([{ owner: "o", server: "s" }], roomyLimits(), {
    authority: "external: liar",
    verifyClose: () => ({ contextId: "", closed: "yes", digest: "" }) as unknown as CloseFacts,
  });
  liar.connect("o", "s", "c");
  const liarConn = liar.connectionTokenForTest("o", "c");
  liar.openContext("o", "c", liarConn, "x");
  const liarCtx = liar.contextTokenForTest("o", "c", "x");
  liar.requestContextClose("o", "c", "x", liarCtx);
  expectRemoteError(
    () => liar.sealContextClose("o", "c", "x", liarCtx, "cls-anything"),
    "forged-close",
    "cleanup",
  );
});

check("worker-receipt-never-confirms-cleanup", () => {
  assert.deepEqual([...FORBIDDEN_CLOSE_KINDS], ["worker-receipt", "server-log", "exit-code"]);
  // Every forbidden kind rejects before any verdict is read.
  for (const kind of FORBIDDEN_CLOSE_KINDS) {
    expectRemoteError(
      () => assertRemoteCleanupWitnessed({ kind, detail: "worker says clean" }),
      "forbidden-close",
      "cleanup",
    );
  }
  // Unknown kinds and malformed claims reject the same way.
  for (const claim of [
    { kind: "obituary", detail: "worker died quietly" },
    { kind: "process-sample", detail: "no pid found" },
    { kind: 7, detail: "x" },
    { detail: "no kind" },
    null,
    "external-close",
  ]) {
    expectRemoteError(() => assertRemoteCleanupWitnessed(claim), "forbidden-close", "cleanup");
  }
  // A forged receipt digest never verifies.
  const { service, authority, connToken, ctxToken } = openContext();
  service.noteWorkerDeath("owner-n", "conn-1", connToken);
  service.requestContextClose("owner-n", "conn-1", "ctx-1", ctxToken);
  const closeToken = authority.seedClose("ctx-1");
  const receipt = service.sealContextClose("owner-n", "conn-1", "ctx-1", ctxToken, closeToken);
  const forged = { ...receipt, digest: `sha256:${"0".repeat(64)}` };
  expectRemoteError(
    () => assertRemoteCleanupWitnessed({ kind: "external-close", receipt: forged }),
    "forbidden-close",
    "cleanup",
  );
  // A receipt with an edited worker-death bit never verifies either.
  const edited = { ...receipt, workerDeathNoted: false };
  expectRemoteError(
    () => assertRemoteCleanupWitnessed({ kind: "external-close", receipt: edited }),
    "forbidden-close",
    "cleanup",
  );
});

check("independent-close-verdict-judges-cleanup", () => {
  // Positive: a sealed receipt verifies and judges witnessed cleanup.
  const { service, authority, ctxToken } = openContext();
  service.requestContextClose("owner-n", "conn-1", "ctx-1", ctxToken);
  const closeToken = authority.seedClose("ctx-1");
  const receipt = service.sealContextClose("owner-n", "conn-1", "ctx-1", ctxToken, closeToken);
  const verdict = assertRemoteCleanupWitnessed({ kind: "external-close", receipt });
  assert.equal(verdict.contextId, "ctx-1");
  assert.equal(verdict.connectionId, "conn-1");
  assert.equal(verdict.witnessed, true);
  assert.ok(verdict.digest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(verdict).sort(), [
    "connectionId",
    "contextId",
    "digest",
    "witnessed",
  ]);
  // A closed id never reopens: context ids are single-use.
  const connToken = service.connectionTokenForTest("owner-n", "conn-1");
  expectRemoteError(
    () => service.openContext("owner-n", "conn-1", connToken, "ctx-1"),
    "context-closed",
    "remote",
  );
});

// --- Vocabulary, scope statement, and capacity --------------------------------

check("every-failure-names-its-layer", () => {
  // The code vocabulary is closed and every code maps to exactly one layer.
  assert.equal(BROWSER_REMOTE_CODES.length, 14);
  assert.equal(BROWSER_REMOTE_SCHEMA_VERSION, "1");
  for (const code of BROWSER_REMOTE_CODES) {
    const layer = layerOfRemoteCode(code);
    assert.ok(["service", "remote", "cleanup"].includes(layer));
    const error = new BrowserRemoteError(code, "probe");
    assert.equal(error.layer, layer);
    assert.equal(error.code, code);
    assert.ok(error.message.startsWith(`[${layer}:${code}]`));
  }
  // Limit shapes reject at the service layer with nothing constructed.
  expectRemoteError(() => checkRemoteLimits(null), "unknown-server", "service");
  expectRemoteError(() => checkRemoteLimits({ maxServers: 1 }), "unknown-server", "service");
  expectRemoteError(
    () =>
      checkRemoteLimits({ maxServers: 1, maxConnections: 1, maxContextsPerConnection: 1, maxX: 1 }),
    "unknown-server",
    "service",
  );
  // A missing close authority never constructs: cleanup without an
  // independent owner is unconfirmable.
  expectRemoteError(
    () =>
      new BrowserRemoteService(
        [{ owner: "o", server: "s" }],
        roomyLimits(),
        null as unknown as ExternalCloseAuthority,
      ),
    "forbidden-close",
    "cleanup",
  );
  // Capacity faults name the service layer.
  const tiny = new BrowserRemoteService(
    [{ owner: "o", server: "s" }],
    checkRemoteLimits({ maxServers: 1, maxConnections: 1, maxContextsPerConnection: 1 }),
    new FakeCloseAuthority(),
  );
  tiny.connect("o", "s", "c1");
  const tok = tiny.connectionTokenForTest("o", "c1");
  expectRemoteError(() => tiny.connect("o", "s", "c2"), "capacity-exhausted", "service");
  tiny.openContext("o", "c1", tok, "x1");
  expectRemoteError(() => tiny.openContext("o", "c1", tok, "x2"), "capacity-exhausted", "service");
});

check("independence-scope-stated", () => {
  // The scope statement is a fixed export: the service shares no driver
  // code, decoded values, receipts, flags, environment, network state, or
  // process state, and proves its facts from its own seeded doubles alone.
  assert.ok(BROWSER_REMOTE_INDEPENDENCE_SCOPE.startsWith("independent:"));
  assert.ok(BROWSER_REMOTE_INDEPENDENCE_SCOPE.includes("no driver code"));
  assert.ok(BROWSER_REMOTE_INDEPENDENCE_SCOPE.includes("seeded doubles"));
  // The close seam is independent by construction: the double states its
  // outside-service authority.
  const authority = new FakeCloseAuthority();
  assert.ok(authority.authority.startsWith("external:"));
});

console.log(
  JSON.stringify({
    kind: "can.browser-driver-remote-check",
    schema_version: BROWSER_REMOTE_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
