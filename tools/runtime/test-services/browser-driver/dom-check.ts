// K13 bounded self-check: immediate DOM and event provenance.
//
// Run with: bun tools/runtime/test-services/browser-driver/dom-check.ts
// Local controls only. No DOM, browsers, processes, flags, environment,
// sampling, networks, files, timers, wall clocks, live hosts, or live
// runtimes. Task and microtask order below is explicitly scripted, and
// every loop is bounded by a small constant. Live host-dependent controls
// wait for the qualified profile and Q task.

import { strict as assert } from "node:assert";
import {
  assertBatchImmediate,
  assertCloneFresh,
  assertOccurrenceObserved,
  BROWSER_DOM_CODES,
  BROWSER_DOM_INDEPENDENCE_SCOPE,
  BROWSER_DOM_SCHEMA_VERSION,
  BrowserDomError,
  BrowserDomService,
  CANCEL_MODES,
  checkDomLimits,
  digestClone,
  DOM_EVENTS,
  DOM_OPS,
  FORBIDDEN_OCCURRENCE_KINDS,
  layerOfDomCode,
  NODE_KINDS,
  type BrowserDomCode,
  type BrowserDomLayer,
  type BrowserDomLimits,
} from "./dom.ts";

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectDomError(body: () => unknown, code: BrowserDomCode, layer: BrowserDomLayer): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof BrowserDomError)) {
      assert.fail(`expected a BrowserDomError, got ${String(error)}`);
    }
    assert.equal(error.code, code);
    assert.equal(error.layer, layer);
    assert.ok(error.message.includes(`[${layer}:${code}]`));
    return;
  }
  assert.fail(`expected failure ${layer}:${code}`);
}

const TEXT_DIGEST = `sha256:${"ab".repeat(32)}`;

function roomyLimits(): BrowserDomLimits {
  return checkDomLimits({
    maxContexts: 4,
    maxNodes: 32,
    maxBatches: 16,
    maxOpsPerBatch: 8,
    maxListeners: 16,
    maxOccurrences: 16,
    maxTranscripts: 16,
  });
}

function openService(): BrowserDomService {
  const service = new BrowserDomService(
    [
      { owner: "owner-n", context: "ctx-a" },
      { owner: "owner-n", context: "ctx-b" },
    ],
    roomyLimits(),
  );
  service.bindContext("owner-n", "ctx-a");
  return service;
}

function seedPair(service: BrowserDomService): void {
  service.seedNode("owner-n", "ctx-a", "root", { kind: "element", textDigest: TEXT_DIGEST });
  service.seedNode("owner-n", "ctx-a", "leaf", { kind: "text", textDigest: TEXT_DIGEST });
}

function beginBatch(service: BrowserDomService, batchId = "batch-1"): string {
  service.beginBatch("owner-n", "ctx-a", batchId);
  return service.batchTokenForTest("owner-n", "ctx-a", batchId);
}

function emitOccurrence(
  service: BrowserDomService,
  token: string,
  batchId: string,
  occurrenceId: string,
  nodeId = "leaf",
  event = "click",
): void {
  service.emitInBatch("owner-n", "ctx-a", batchId, token, {
    nodeId,
    occurrenceId,
    event,
  });
}

// --- Acceptance: finite same-task batch ------------------------------------

check("finite-same-task-batch", () => {
  const service = openService();
  seedPair(service);
  // The declared vocabularies are finite and exact.
  assert.deepEqual([...DOM_OPS], ["clone", "append", "emit"]);
  assert.deepEqual([...NODE_KINDS], ["element", "text"]);
  assert.deepEqual([...DOM_EVENTS], ["click", "input", "mutation"]);
  const token = beginBatch(service);
  const attestation = service.attestation("owner-n", "ctx-a", "batch-1");
  assert.equal(attestation.open, true);
  assert.equal(attestation.taskSeq, 0);
  // One batch runs clone, append, and emit in recorded order.
  const clone = service.cloneInBatch("owner-n", "ctx-a", "batch-1", token, { nodeId: "leaf" });
  assert.equal(clone.opSeq, 0);
  const append = service.appendInBatch("owner-n", "ctx-a", "batch-1", token, {
    parentId: "root",
    childId: clone.cloneId,
  });
  assert.equal(append.opSeq, 1);
  emitOccurrence(service, token, "batch-1", "occ-1");
  // Open-batch facts stay readable while the batch runs.
  const open = service.batchFacts("owner-n", "ctx-a", "batch-1");
  assert.equal(open.sealed, false);
  assert.equal(open.yielded, false);
  assert.equal(open.yieldSeq, null);
  assert.deepEqual(
    open.ops.map((entry) => entry.op),
    ["clone", "append", "emit"],
  );
  assert.deepEqual(
    open.ops.map((entry) => entry.seq),
    [0, 1, 2],
  );
  // Re-beginning an open id rejoins with the same attested token.
  service.beginBatch("owner-n", "ctx-a", "batch-1");
  assert.equal(service.batchTokenForTest("owner-n", "ctx-a", "batch-1"), token);
  const sealed = service.sealBatch("owner-n", "ctx-a", "batch-1", token);
  assert.equal(sealed.sealed, true);
  assert.equal(sealed.yielded, false);
  assert.equal(sealed.ops.length, 3);
  const verdict = assertBatchImmediate(sealed);
  assert.equal(verdict.immediate, true);
  assert.equal(verdict.ops, 3);
  // Sealed ids never re-open; sealed batches refuse new operations.
  expectDomError(() => service.beginBatch("owner-n", "ctx-a", "batch-1"), "batch-closed", "batch");
  expectDomError(() => service.attestation("owner-n", "ctx-a", "batch-1"), "batch-closed", "batch");
  expectDomError(
    () => service.cloneInBatch("owner-n", "ctx-a", "batch-1", token, { nodeId: "leaf" }),
    "batch-closed",
    "batch",
  );
  expectDomError(
    () => service.sealBatch("owner-n", "ctx-a", "batch-1", token),
    "batch-closed",
    "batch",
  );
  // An empty batch seals too: zero ops is a fact, not a gap.
  const emptyToken = beginBatch(service, "batch-empty");
  const emptySealed = service.sealBatch("owner-n", "ctx-a", "batch-empty", emptyToken);
  assert.equal(emptySealed.ops.length, 0);
  assert.equal(assertBatchImmediate(emptySealed).ops, 0);
  // Invented tokens and unknown batches are never authority.
  const otherToken = beginBatch(service, "batch-2");
  expectDomError(
    () => service.cloneInBatch("owner-n", "ctx-a", "batch-2", "batch-invented", { nodeId: "leaf" }),
    "forged-token",
    "batch",
  );
  expectDomError(
    () => service.sealBatch("owner-n", "ctx-a", "batch-2", "batch-invented"),
    "forged-token",
    "batch",
  );
  expectDomError(
    () => service.batchFacts("owner-n", "ctx-a", "batch-never"),
    "unknown-batch",
    "batch",
  );
  expectDomError(
    () => service.batchTokenForTest("owner-n", "ctx-a", "batch-never"),
    "unknown-batch",
    "batch",
  );
  service.sealBatch("owner-n", "ctx-a", "batch-2", otherToken);
  // A second emit of the same occurrence id refuses: ids are single-use.
  const replayToken = beginBatch(service, "batch-3");
  expectDomError(
    () =>
      service.emitInBatch("owner-n", "ctx-a", "batch-3", replayToken, {
        nodeId: "leaf",
        occurrenceId: "occ-1",
        event: "click",
      }),
    "occurrence-closed",
    "event",
  );
  // A re-seed refuses: node doubles are never swapped.
  expectDomError(
    () => service.seedNode("owner-n", "ctx-a", "leaf", { kind: "text", textDigest: TEXT_DIGEST }),
    "node-closed",
    "dom",
  );
  service.sealBatch("owner-n", "ctx-a", "batch-3", replayToken);
});

check("batch-op-capacity", () => {
  const service = new BrowserDomService(
    [{ owner: "o", context: "c" }],
    checkDomLimits({
      maxContexts: 1,
      maxNodes: 8,
      maxBatches: 2,
      maxOpsPerBatch: 1,
      maxListeners: 2,
      maxOccurrences: 4,
      maxTranscripts: 2,
    }),
  );
  service.bindContext("o", "c");
  service.seedNode("o", "c", "n", { kind: "element", textDigest: TEXT_DIGEST });
  service.beginBatch("o", "c", "b");
  const token = service.batchTokenForTest("o", "c", "b");
  service.cloneInBatch("o", "c", "b", token, { nodeId: "n" });
  expectDomError(
    () =>
      service.emitInBatch("o", "c", "b", token, { nodeId: "n", occurrenceId: "e", event: "click" }),
    "capacity-exhausted",
    "dom",
  );
  service.sealBatch("o", "c", "b", token);
});

// --- Acceptance: yield detectable ------------------------------------------

check("yield-detectable", () => {
  const service = openService();
  seedPair(service);
  const token = beginBatch(service);
  service.cloneInBatch("owner-n", "ctx-a", "batch-1", token, { nodeId: "leaf" });
  // The scripted yield names every open batch it interrupted.
  const task = service.yieldTask("owner-n", "ctx-a");
  assert.equal(task.taskSeq, 1);
  assert.deepEqual([...task.yieldedBatches], ["batch-1"]);
  // The yield is visible on the open batch before the seal.
  const open = service.batchFacts("owner-n", "ctx-a", "batch-1");
  assert.equal(open.yielded, true);
  assert.equal(open.yieldSeq, 1);
  // The next operation refuses: the yield is detectable at the next op.
  expectDomError(
    () => service.cloneInBatch("owner-n", "ctx-a", "batch-1", token, { nodeId: "leaf" }),
    "batch-yielded",
    "batch",
  );
  expectDomError(
    () =>
      service.emitInBatch("owner-n", "ctx-a", "batch-1", token, {
        nodeId: "leaf",
        occurrenceId: "occ-late",
        event: "click",
      }),
    "batch-yielded",
    "batch",
  );
  // Sealing stays available so the yield provenance is always readable.
  const sealed = service.sealBatch("owner-n", "ctx-a", "batch-1", token);
  assert.equal(sealed.sealed, true);
  assert.equal(sealed.yielded, true);
  assert.equal(sealed.yieldSeq, 1);
  assert.equal(sealed.ops.length, 1);
  expectDomError(() => assertBatchImmediate(sealed), "batch-yielded", "batch");
  // A batch begun after the yield runs in the new task and stays immediate.
  const freshToken = beginBatch(service, "batch-2");
  emitOccurrence(service, freshToken, "batch-2", "occ-2");
  const freshSealed = service.sealBatch("owner-n", "ctx-a", "batch-2", freshToken);
  assert.equal(freshSealed.taskSeq, 1);
  assert.equal(freshSealed.yielded, false);
  assert.equal(assertBatchImmediate(freshSealed).immediate, true);
  // A yield with no open batch interrupts nothing but still advances time.
  const quiet = service.yieldTask("owner-n", "ctx-a");
  assert.equal(quiet.taskSeq, 2);
  assert.deepEqual([...quiet.yieldedBatches], []);
  // An unsealed batch presented as proof refuses before any verdict.
  const pendingToken = beginBatch(service, "batch-3");
  emitOccurrence(service, pendingToken, "batch-3", "occ-3");
  expectDomError(
    () => assertBatchImmediate(service.batchFacts("owner-n", "ctx-a", "batch-3")),
    "batch-pending",
    "batch",
  );
  // Malformed and tampered batch facts refuse before any verdict.
  expectDomError(() => assertBatchImmediate(null), "unknown-batch", "batch");
  expectDomError(
    () => assertBatchImmediate({ ...freshSealed, digest: `sha256:${"00".repeat(32)}` }),
    "unknown-batch",
    "batch",
  );
  expectDomError(
    () => assertBatchImmediate({ ...freshSealed, ops: [{ seq: 0, op: "paint", ref: "x" }] }),
    "unknown-batch",
    "batch",
  );
  service.sealBatch("owner-n", "ctx-a", "batch-3", pendingToken);
});

// --- Acceptance: returned original handle detectable -----------------------

check("clone-freshness", () => {
  const service = openService();
  seedPair(service);
  const token = beginBatch(service);
  const clone = service.cloneInBatch("owner-n", "ctx-a", "batch-1", token, { nodeId: "leaf" });
  // The minted clone id can never equal the original id.
  assert.notEqual(clone.cloneId, clone.originalId);
  assert.equal(clone.originalId, "leaf");
  assert.equal(
    clone.digest,
    digestClone(clone.originalId, clone.cloneId, "owner-n", "ctx-a", "batch-1", clone.opSeq),
  );
  const verdict = assertCloneFresh(clone);
  assert.equal(verdict.fresh, true);
  assert.equal(verdict.cloneId, clone.cloneId);
  // The clone double records its original; the original records none.
  assert.equal(service.nodeFacts("owner-n", "ctx-a", clone.cloneId).cloneOf, "leaf");
  assert.equal(service.nodeFacts("owner-n", "ctx-a", "leaf").cloneOf, null);
  // A second clone of the same original mints a distinct id.
  const again = service.cloneInBatch("owner-n", "ctx-a", "batch-1", token, { nodeId: "leaf" });
  assert.notEqual(again.cloneId, clone.cloneId);
  assert.equal(assertCloneFresh(again).fresh, true);
  service.sealBatch("owner-n", "ctx-a", "batch-1", token);
  // Seeded negative: a forged clone that returns the original handle.
  const forged = {
    originalId: "leaf",
    cloneId: "leaf",
    owner: "owner-n",
    context: "ctx-a",
    batchId: "batch-1",
    opSeq: 0,
    digest: digestClone("leaf", "leaf", "owner-n", "ctx-a", "batch-1", 0),
  };
  expectDomError(() => assertCloneFresh(forged), "original-returned", "dom");
  // Missing-evidence controls: malformed and tampered clone facts.
  expectDomError(() => assertCloneFresh(null), "unknown-node", "dom");
  expectDomError(() => assertCloneFresh({ ...clone, opSeq: "0" }), "unknown-node", "dom");
  expectDomError(
    () => assertCloneFresh({ ...clone, digest: `sha256:${"00".repeat(32)}` }),
    "unknown-node",
    "dom",
  );
});

check("append-rejects-original", () => {
  const service = openService();
  seedPair(service);
  const token = beginBatch(service);
  const clone = service.cloneInBatch("owner-n", "ctx-a", "batch-1", token, { nodeId: "leaf" });
  // Presenting a seeded original as the appended child refuses: a
  // returned original handle never appends silently.
  expectDomError(
    () =>
      service.appendInBatch("owner-n", "ctx-a", "batch-1", token, {
        parentId: "root",
        childId: "leaf",
      }),
    "original-returned",
    "dom",
  );
  // Unknown parent and child names refuse before any provenance is read.
  expectDomError(
    () =>
      service.appendInBatch("owner-n", "ctx-a", "batch-1", token, {
        parentId: "nowhere",
        childId: clone.cloneId,
      }),
    "unknown-node",
    "dom",
  );
  expectDomError(
    () =>
      service.appendInBatch("owner-n", "ctx-a", "batch-1", token, {
        parentId: "root",
        childId: "nowhere",
      }),
    "unknown-node",
    "dom",
  );
  // The minted clone appends, and the double records its parent.
  const append = service.appendInBatch("owner-n", "ctx-a", "batch-1", token, {
    parentId: "root",
    childId: clone.cloneId,
  });
  assert.equal(append.parentId, "root");
  assert.equal(append.childId, clone.cloneId);
  service.sealBatch("owner-n", "ctx-a", "batch-1", token);
  // Cloning an unknown node refuses at the dom layer.
  const otherToken = beginBatch(service, "batch-2");
  expectDomError(
    () => service.cloneInBatch("owner-n", "ctx-a", "batch-2", otherToken, { nodeId: "nowhere" }),
    "unknown-node",
    "dom",
  );
  expectDomError(() => service.nodeFacts("owner-n", "ctx-a", "nowhere"), "unknown-node", "dom");
  service.sealBatch("owner-n", "ctx-a", "batch-2", otherToken);
});

// --- Acceptance: same-node listener acknowledgment -------------------------

check("same-node-acknowledgment", () => {
  const service = openService();
  seedPair(service);
  service.addListener("owner-n", "ctx-a", "leaf", "listen-1", { event: "click" });
  service.addListener("owner-n", "ctx-a", "root", "listen-root", { event: "click" });
  // Re-attaching the identical pair rejoins; a retarget refuses.
  service.addListener("owner-n", "ctx-a", "leaf", "listen-1", { event: "click" });
  expectDomError(
    () => service.addListener("owner-n", "ctx-a", "root", "listen-1", { event: "click" }),
    "listener-mismatch",
    "event",
  );
  expectDomError(
    () => service.addListener("owner-n", "ctx-a", "nowhere", "listen-x", { event: "click" }),
    "unknown-node",
    "dom",
  );
  expectDomError(
    () => service.listenerFacts("owner-n", "ctx-a", "listen-never"),
    "unknown-listener",
    "event",
  );
  const token = beginBatch(service);
  emitOccurrence(service, token, "batch-1", "occ-1");
  service.sealBatch("owner-n", "ctx-a", "batch-1", token);
  // The same-node listener acknowledges; the count carries on the facts.
  const first = service.acknowledge("owner-n", "ctx-a", {
    occurrenceId: "occ-1",
    nodeId: "leaf",
    listenerId: "listen-1",
  });
  assert.equal(first.ackSeq, 0);
  const second = service.acknowledge("owner-n", "ctx-a", {
    occurrenceId: "occ-1",
    nodeId: "leaf",
    listenerId: "listen-1",
  });
  assert.equal(second.ackSeq, 1);
  assert.equal(service.occurrenceFacts("owner-n", "ctx-a", "occ-1").acks, 2);
  // An acknowledgment naming another node refuses: same-node only.
  expectDomError(
    () =>
      service.acknowledge("owner-n", "ctx-a", {
        occurrenceId: "occ-1",
        nodeId: "root",
        listenerId: "listen-root",
      }),
    "listener-mismatch",
    "event",
  );
  // A listener attached to another node cannot acknowledge this occurrence.
  expectDomError(
    () =>
      service.acknowledge("owner-n", "ctx-a", {
        occurrenceId: "occ-1",
        nodeId: "leaf",
        listenerId: "listen-root",
      }),
    "listener-mismatch",
    "event",
  );
  // Unknown listeners and occurrences refuse before any ack is recorded.
  expectDomError(
    () =>
      service.acknowledge("owner-n", "ctx-a", {
        occurrenceId: "occ-1",
        nodeId: "leaf",
        listenerId: "listen-never",
      }),
    "unknown-listener",
    "event",
  );
  expectDomError(
    () =>
      service.acknowledge("owner-n", "ctx-a", {
        occurrenceId: "occ-never",
        nodeId: "leaf",
        listenerId: "listen-1",
      }),
    "unknown-occurrence",
    "event",
  );
  expectDomError(
    () => service.occurrenceFacts("owner-n", "ctx-a", "occ-never"),
    "unknown-occurrence",
    "event",
  );
});

// --- Acceptance: wrong cancellation detectable -----------------------------

check("wrong-cancellation", () => {
  const service = openService();
  seedPair(service);
  assert.deepEqual([...CANCEL_MODES], ["prevent-default", "stop-propagation"]);
  service.addListener("owner-n", "ctx-a", "leaf", "listen-1", { event: "click" });
  service.addListener("owner-n", "ctx-a", "root", "listen-root", { event: "click" });
  service.addListener("owner-n", "ctx-a", "leaf", "listen-input", { event: "input" });
  const token = beginBatch(service);
  emitOccurrence(service, token, "batch-1", "occ-1");
  emitOccurrence(service, token, "batch-1", "occ-2");
  emitOccurrence(service, token, "batch-1", "occ-3");
  service.sealBatch("owner-n", "ctx-a", "batch-1", token);
  // Every declared mode cancels through the same-node listener.
  const first = service.cancelOccurrence("owner-n", "ctx-a", {
    occurrenceId: "occ-1",
    nodeId: "leaf",
    listenerId: "listen-1",
    mode: "prevent-default",
  });
  assert.equal(first.mode, "prevent-default");
  const second = service.cancelOccurrence("owner-n", "ctx-a", {
    occurrenceId: "occ-2",
    nodeId: "leaf",
    listenerId: "listen-1",
    mode: "stop-propagation",
  });
  assert.equal(second.mode, "stop-propagation");
  const facts = service.occurrenceFacts("owner-n", "ctx-a", "occ-1");
  assert.equal(facts.cancelled, true);
  assert.equal(facts.cancelMode, "prevent-default");
  // A cancellation naming another node is a wrong cancellation.
  expectDomError(
    () =>
      service.cancelOccurrence("owner-n", "ctx-a", {
        occurrenceId: "occ-3",
        nodeId: "root",
        listenerId: "listen-root",
        mode: "prevent-default",
      }),
    "wrong-cancellation",
    "event",
  );
  // A listener from another node cannot cancel this occurrence.
  expectDomError(
    () =>
      service.cancelOccurrence("owner-n", "ctx-a", {
        occurrenceId: "occ-3",
        nodeId: "leaf",
        listenerId: "listen-root",
        mode: "prevent-default",
      }),
    "wrong-cancellation",
    "event",
  );
  // A listener for another event cannot cancel this occurrence either.
  expectDomError(
    () =>
      service.cancelOccurrence("owner-n", "ctx-a", {
        occurrenceId: "occ-3",
        nodeId: "leaf",
        listenerId: "listen-input",
        mode: "prevent-default",
      }),
    "wrong-cancellation",
    "event",
  );
  // An undeclared mode is a wrong cancellation before any lookup.
  expectDomError(
    () =>
      service.cancelOccurrence("owner-n", "ctx-a", {
        occurrenceId: "occ-3",
        nodeId: "leaf",
        listenerId: "listen-1",
        mode: "cancel-hard",
      }),
    "wrong-cancellation",
    "event",
  );
  // Unknown listeners and occurrences refuse; a second cancel refuses.
  expectDomError(
    () =>
      service.cancelOccurrence("owner-n", "ctx-a", {
        occurrenceId: "occ-3",
        nodeId: "leaf",
        listenerId: "listen-never",
        mode: "prevent-default",
      }),
    "unknown-listener",
    "event",
  );
  expectDomError(
    () =>
      service.cancelOccurrence("owner-n", "ctx-a", {
        occurrenceId: "occ-never",
        nodeId: "leaf",
        listenerId: "listen-1",
        mode: "prevent-default",
      }),
    "unknown-occurrence",
    "event",
  );
  expectDomError(
    () =>
      service.cancelOccurrence("owner-n", "ctx-a", {
        occurrenceId: "occ-1",
        nodeId: "leaf",
        listenerId: "listen-1",
        mode: "prevent-default",
      }),
    "already-cancelled",
    "event",
  );
  // The refused cancellations left occ-3 live.
  const live = service.occurrenceFacts("owner-n", "ctx-a", "occ-3");
  assert.equal(live.cancelled, false);
  assert.equal(live.cancelMode, null);
});

// --- Acceptance: microtask-order observer and missing occurrence -----------

check("microtask-order-observer", () => {
  const service = openService();
  seedPair(service);
  const token = beginBatch(service);
  emitOccurrence(service, token, "batch-1", "occ-a");
  emitOccurrence(service, token, "batch-1", "occ-b");
  emitOccurrence(service, token, "batch-1", "occ-c");
  service.sealBatch("owner-n", "ctx-a", "batch-1", token);
  // The scripted order assigns microtask slots slot by slot: no wall clock.
  const transcript = service.observeMicrotasks("owner-n", "ctx-a", {
    order: ["occ-b", "occ-a"],
  });
  assert.equal(transcript.transcriptSeq, 0);
  assert.deepEqual([...transcript.order], ["occ-b", "occ-a"]);
  assert.equal(service.occurrenceFacts("owner-n", "ctx-a", "occ-b").microtaskSeq, 0);
  assert.equal(service.occurrenceFacts("owner-n", "ctx-a", "occ-a").microtaskSeq, 1);
  assert.equal(service.occurrenceFacts("owner-n", "ctx-a", "occ-c").microtaskSeq, null);
  // A later transcript continues the slot numbering, and stays re-readable.
  const later = service.observeMicrotasks("owner-n", "ctx-a", { order: ["occ-c"] });
  assert.equal(later.transcriptSeq, 1);
  assert.equal(service.occurrenceFacts("owner-n", "ctx-a", "occ-c").microtaskSeq, 2);
  assert.deepEqual(service.transcriptFacts("owner-n", "ctx-a", 0).order, ["occ-b", "occ-a"]);
  expectDomError(
    () => service.transcriptFacts("owner-n", "ctx-a", 7),
    "unknown-transcript",
    "event",
  );
  expectDomError(
    () => service.transcriptFacts("owner-n", "ctx-a", -1),
    "unknown-transcript",
    "event",
  );
  // A missing occurrence in the script refuses: never silently skipped.
  expectDomError(
    () => service.observeMicrotasks("owner-n", "ctx-a", { order: ["occ-never"] }),
    "unknown-occurrence",
    "event",
  );
  expectDomError(
    () => service.observeMicrotasks("owner-n", "ctx-a", { order: [] }),
    "unknown-occurrence",
    "event",
  );
  // Repeated and already observed occurrences refuse as single-use slots.
  expectDomError(
    () => service.observeMicrotasks("owner-n", "ctx-a", { order: ["occ-a"] }),
    "occurrence-closed",
    "event",
  );
  const fresh = openService();
  fresh.bindContext("owner-n", "ctx-b");
  fresh.seedNode("owner-n", "ctx-b", "n", { kind: "element", textDigest: TEXT_DIGEST });
  fresh.beginBatch("owner-n", "ctx-b", "b");
  const freshToken = fresh.batchTokenForTest("owner-n", "ctx-b", "b");
  fresh.emitInBatch("owner-n", "ctx-b", "b", freshToken, {
    nodeId: "n",
    occurrenceId: "e",
    event: "click",
  });
  fresh.sealBatch("owner-n", "ctx-b", "b", freshToken);
  expectDomError(
    () => fresh.observeMicrotasks("owner-n", "ctx-b", { order: ["e", "e"] }),
    "occurrence-closed",
    "event",
  );
});

check("missing-occurrence-verdict", () => {
  const service = openService();
  seedPair(service);
  const token = beginBatch(service);
  emitOccurrence(service, token, "batch-1", "occ-seen");
  emitOccurrence(service, token, "batch-1", "occ-missing");
  service.sealBatch("owner-n", "ctx-a", "batch-1", token);
  service.observeMicrotasks("owner-n", "ctx-a", { order: ["occ-seen"] });
  // Only a verified observed occurrence proves observation.
  const seen = service.occurrenceFacts("owner-n", "ctx-a", "occ-seen");
  const verdict = assertOccurrenceObserved({ kind: "observed-occurrence", occurrence: seen });
  assert.equal(verdict.observed, true);
  assert.equal(verdict.microtaskSeq, 0);
  // A recorded but unobserved occurrence is missing, never a quiet pass.
  const missing = service.occurrenceFacts("owner-n", "ctx-a", "occ-missing");
  assert.equal(missing.microtaskSeq, null);
  expectDomError(
    () => assertOccurrenceObserved({ kind: "observed-occurrence", occurrence: missing }),
    "occurrence-missing",
    "event",
  );
  // Every forbidden proof kind rejects before any verdict is read.
  assert.deepEqual(
    [...FORBIDDEN_OCCURRENCE_KINDS],
    ["unobserved-occurrence", "batch-op", "listener-ack"],
  );
  for (const kind of FORBIDDEN_OCCURRENCE_KINDS) {
    expectDomError(
      () => assertOccurrenceObserved({ kind, detail: "seeded" }),
      "forbidden-proof",
      "event",
    );
  }
  expectDomError(
    () => assertOccurrenceObserved({ kind: "driver-log", detail: "seeded" }),
    "forbidden-proof",
    "event",
  );
  expectDomError(() => assertOccurrenceObserved(null), "forbidden-proof", "event");
  expectDomError(
    () => assertOccurrenceObserved({ kind: "observed-occurrence", occurrence: null }),
    "forbidden-proof",
    "event",
  );
  // Missing-evidence controls: malformed, undeclared, and tampered facts.
  expectDomError(
    () =>
      assertOccurrenceObserved({
        kind: "observed-occurrence",
        occurrence: { ...seen, acks: "0" },
      }),
    "forbidden-proof",
    "event",
  );
  expectDomError(
    () =>
      assertOccurrenceObserved({
        kind: "observed-occurrence",
        occurrence: { ...seen, event: "paint" },
      }),
    "forbidden-proof",
    "event",
  );
  expectDomError(
    () =>
      assertOccurrenceObserved({
        kind: "observed-occurrence",
        occurrence: { ...seen, digest: `sha256:${"00".repeat(32)}` },
      }),
    "forbidden-proof",
    "event",
  );
});

// --- Admission, ownership, and lifecycle ------------------------------------

check("admission-owner-lifecycle", () => {
  const service = openService();
  seedPair(service);
  // A missing context blocks DOM work, whatever the reason.
  expectDomError(() => service.beginBatch("owner-n", "ctx-never", "b"), "no-context", "dom");
  expectDomError(
    () =>
      service.seedNode("owner-n", "ctx-never", "n", { kind: "element", textDigest: TEXT_DIGEST }),
    "no-context",
    "dom",
  );
  expectDomError(
    () => service.observeMicrotasks("owner-n", "ctx-never", { order: ["e"] }),
    "no-context",
    "dom",
  );
  expectDomError(() => service.yieldTask("owner-n", "ctx-never"), "no-context", "dom");
  expectDomError(() => service.bindContext("owner-n", "ctx-never"), "unknown-context", "dom");
  // A foreign owner fails the owner check.
  expectDomError(() => service.binding("owner-x", "ctx-a"), "wrong-owner", "dom");
  expectDomError(() => service.batchFacts("owner-x", "ctx-a", "batch-1"), "wrong-owner", "dom");
  // An open batch blocks the close; sealing unblocks it.
  beginBatch(service);
  expectDomError(() => service.closeContext("owner-n", "ctx-a"), "batch-pending", "batch");
  const token = service.batchTokenForTest("owner-n", "ctx-a", "batch-1");
  service.sealBatch("owner-n", "ctx-a", "batch-1", token);
  service.closeContext("owner-n", "ctx-a");
  // After the close, admission refuses and facts report closed.
  expectDomError(() => service.beginBatch("owner-n", "ctx-a", "b2"), "no-context", "dom");
  expectDomError(
    () => service.occurrenceFacts("owner-n", "ctx-a", "occ-1"),
    "context-closed",
    "dom",
  );
  expectDomError(() => service.batchFacts("owner-n", "ctx-a", "batch-1"), "context-closed", "dom");
});

// --- Vocabulary, scope statement, and capacity --------------------------------

check("every-failure-names-its-layer", () => {
  // The code vocabulary is closed and every code maps to exactly one layer.
  assert.equal(BROWSER_DOM_CODES.length, 23);
  assert.equal(BROWSER_DOM_SCHEMA_VERSION, "1");
  for (const code of BROWSER_DOM_CODES) {
    const layer = layerOfDomCode(code);
    assert.ok(["dom", "batch", "event"].includes(layer));
    const error = new BrowserDomError(code, "probe");
    assert.equal(error.layer, layer);
    assert.equal(error.code, code);
    assert.ok(error.message.startsWith(`[${layer}:${code}]`));
  }
  // Limit shapes reject at the dom layer with nothing constructed.
  expectDomError(() => checkDomLimits(null), "unknown-context", "dom");
  expectDomError(() => checkDomLimits({ maxContexts: 1 }), "unknown-context", "dom");
  expectDomError(
    () =>
      checkDomLimits({
        maxContexts: 1,
        maxNodes: 1,
        maxBatches: 1,
        maxOpsPerBatch: 1,
        maxListeners: 1,
        maxOccurrences: 1,
        maxTranscripts: 1,
        maxBrowsers: 1,
      }),
    "unknown-context",
    "dom",
  );
  // Empty and duplicate declarations never construct.
  expectDomError(() => new BrowserDomService([], roomyLimits()), "unknown-context", "dom");
  expectDomError(
    () =>
      new BrowserDomService(
        [
          { owner: "o", context: "c" },
          { owner: "o", context: "c" },
        ],
        roomyLimits(),
      ),
    "unknown-context",
    "dom",
  );
  // Capacity faults name the dom layer.
  const tiny = new BrowserDomService(
    [{ owner: "o", context: "c" }],
    checkDomLimits({
      maxContexts: 1,
      maxNodes: 1,
      maxBatches: 1,
      maxOpsPerBatch: 4,
      maxListeners: 1,
      maxOccurrences: 1,
      maxTranscripts: 1,
    }),
  );
  tiny.bindContext("o", "c");
  tiny.seedNode("o", "c", "n", { kind: "element", textDigest: TEXT_DIGEST });
  expectDomError(
    () => tiny.seedNode("o", "c", "m", { kind: "element", textDigest: TEXT_DIGEST }),
    "capacity-exhausted",
    "dom",
  );
  tiny.beginBatch("o", "c", "b");
  expectDomError(() => tiny.beginBatch("o", "c", "b2"), "capacity-exhausted", "dom");
  tiny.addListener("o", "c", "n", "l", { event: "click" });
  expectDomError(
    () => tiny.addListener("o", "c", "n", "l2", { event: "click" }),
    "capacity-exhausted",
    "dom",
  );
  const tinyToken = tiny.batchTokenForTest("o", "c", "b");
  tiny.emitInBatch("o", "c", "b", tinyToken, { nodeId: "n", occurrenceId: "e", event: "click" });
  expectDomError(
    () =>
      tiny.emitInBatch("o", "c", "b", tinyToken, {
        nodeId: "n",
        occurrenceId: "e2",
        event: "click",
      }),
    "capacity-exhausted",
    "dom",
  );
  tiny.observeMicrotasks("o", "c", { order: ["e"] });
  tiny.sealBatch("o", "c", "b", tinyToken);
  // Facts carry digests only: raw tokens never cross into facts.
  const service = openService();
  seedPair(service);
  const token = beginBatch(service);
  emitOccurrence(service, token, "batch-1", "occ-1");
  const facts = service.batchFacts("owner-n", "ctx-a", "batch-1");
  assert.ok(!JSON.stringify(facts).includes(token));
  const occurrence = service.occurrenceFacts("owner-n", "ctx-a", "occ-1");
  assert.ok(!JSON.stringify(occurrence).includes(token));
});

check("independence-scope-stated", () => {
  // The scope statement is a fixed export: the service shares no driver
  // code, decoded values, receipts, flags, environment, or process state,
  // and proves its facts from its own seeded doubles alone.
  assert.ok(BROWSER_DOM_INDEPENDENCE_SCOPE.startsWith("independent:"));
  assert.ok(BROWSER_DOM_INDEPENDENCE_SCOPE.includes("no driver code"));
  assert.ok(BROWSER_DOM_INDEPENDENCE_SCOPE.includes("seeded doubles"));
  assert.ok(BROWSER_DOM_INDEPENDENCE_SCOPE.includes("microtask"));
});

console.log(
  JSON.stringify({
    kind: "can.browser-driver-dom-check",
    schema_version: BROWSER_DOM_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
