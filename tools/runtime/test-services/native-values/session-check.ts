// K02 bounded self-check: inert same-realm session mechanics only.
//
// Run with: bun tools/runtime/test-services/native-values/session-check.ts
// Local controls only. No services, transports, timers, files, or live
// runtimes. Every loop below is bounded by a small constant.

import { strict as assert } from "node:assert";
import {
  checkFactsInert,
  checkLimits,
  decodeHandleWire,
  encodeHandleWire,
  mintHandle,
  NativeSchemaError,
  type NativeHandle,
  type NativeValueLimits,
} from "./schema.ts";
import {
  createHostileValue,
  decodeEnvelope,
  encodeEnvelope,
  NATIVE_HOSTILE_DESCRIPTORS,
  NativeSessionService,
  zeroHostileCounters,
  type MakeFacts,
} from "./session.ts";

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectSchemaError(body: () => unknown, kind: string): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof NativeSchemaError)) {
      assert.fail("expected a NativeSchemaError");
    }
    assert.equal(error.kind, kind);
    return;
  }
  assert.fail(`expected rejection with kind ${kind}`);
}

function serviceLimits(): NativeValueLimits {
  return checkLimits({
    max_sessions: 4,
    max_handles_per_session: 32,
    maxPendingActions: 2,
    maxObserveEntries: 8,
    maxObserveBytes: 64,
    maxGatesPerSession: 2,
    maxFaultsPerSession: 2,
  });
}

function tinyLimits(): NativeValueLimits {
  return checkLimits({
    max_sessions: 1,
    max_handles_per_session: 2,
    maxPendingActions: 1,
    maxObserveEntries: 2,
    maxObserveBytes: 8,
    maxGatesPerSession: 1,
    maxFaultsPerSession: 1,
  });
}

// --- Construction ------------------------------------------------------------

check("make-every-kind", () => {
  const service = new NativeSessionService(serviceLimits());
  const session = service.open("owner-a", "s1");
  const prim = service.make(session, "owner-a", "primitive_bits", {
    tag: "f64_bits",
    hi: 0x80000000,
    lo: 0,
  });
  assert.equal(prim.facts.kind, "primitive_bits");
  assert.equal(prim.facts.bytes, 8);
  assert.equal(prim.handle.kind, "value");
  assert.equal(prim.handle.owner, "owner-a");
  assert.equal(prim.handle.sessionId, "s1");
  assert(Object.isFrozen(prim.handle));
  assert(Object.isFrozen(prim.facts));
  const text = service.make(session, "owner-a", "text", { tag: "text", text: "hi" });
  assert.equal(text.facts.bytes, 2);
  const entries = service.make(session, "owner-a", "ordered_entries", {
    tag: "entries",
    entries: [
      ["a", 1],
      ["b", "two"],
    ],
  });
  assert.equal(entries.facts.kind, "ordered_entries");
  for (const descriptor of NATIVE_HOSTILE_DESCRIPTORS) {
    const made = service.make(session, "owner-a", "hostile_descriptor", {
      tag: "hostile",
      descriptor,
    });
    assert.equal(made.facts.descriptor, descriptor);
    checkFactsInert(made.facts, "hostile make facts");
  }
  assert.equal(NATIVE_HOSTILE_DESCRIPTORS.length, 3);
  assert.equal(service.liveCellCount, 6);
});

check("aliases-share-cell", () => {
  const service = new NativeSessionService(serviceLimits());
  const session = service.open("owner-a", "s1");
  const text = service.make(session, "owner-a", "text", { tag: "text", text: "aliased" });
  const first = service.alias(session, "owner-a", text.handle);
  const second = service.alias(session, "owner-a", text.handle);
  assert.notEqual(first.handle.id, text.handle.id);
  assert.notEqual(second.handle.id, first.handle.id);
  assert.equal(first.facts.cell, text.facts.cell);
  assert.equal(second.facts.cell, text.facts.cell);
  assert.equal(first.facts.alias_of, text.handle.id);
  const hostile = service.make(session, "owner-a", "hostile_descriptor", {
    tag: "hostile",
    descriptor: "getter_then",
  });
  const twin = service.alias(session, "owner-a", hostile.handle);
  assert.equal(twin.facts.cell, hostile.facts.cell);
  assert.equal(twin.facts.descriptor, "getter_then");
  assert.equal(service.counters(session, "owner-a").aliases, 3);
});

// --- Hostile fixtures are live, and the service never touches them ------------

check("fixtures-are-hostile", () => {
  const counters = zeroHostileCounters();
  const getter = createHostileValue("getter_then", counters) as Record<string, unknown>;
  assert.equal(counters.getterReads, 0);
  assert.equal(getter["then"], undefined);
  assert.equal(counters.getterReads, 1);
  const throwing = createHostileValue("throwing_thenable", counters) as {
    then: () => unknown;
  };
  assert.throws(() => throwing.then(), /hostile then called/);
  assert.equal(counters.thenCalls, 1);
  const revoked = createHostileValue("revoked_proxy", counters) as Record<string, unknown>;
  assert.throws(() => revoked["anything"], TypeError);
});

check("no-implicit-reads", () => {
  const service = new NativeSessionService(serviceLimits());
  const session = service.open("owner-a", "s1");
  const handles: Array<{ handle: NativeHandle<"value">; facts: MakeFacts }> = [];
  for (const descriptor of NATIVE_HOSTILE_DESCRIPTORS) {
    const made = service.make(session, "owner-a", "hostile_descriptor", {
      tag: "hostile",
      descriptor,
    });
    handles.push(made);
    handles.push(service.alias(session, "owner-a", made.handle));
  }
  for (const { handle, facts } of handles) {
    const json = encodeEnvelope(handle, facts);
    const decoded = decodeEnvelope(json);
    assert.deepEqual(encodeHandleWire(decoded.handle), encodeHandleWire(handle));
    assert.deepEqual(decoded.facts, { ...facts });
  }
  const counters = service.counters(session, "owner-a");
  assert.equal(counters.makes, 3);
  assert.equal(counters.aliases, 3);
  assert.equal(counters.getterReads, 0);
  assert.equal(counters.thenCalls, 0);
  assert.equal(counters.proxyTraps, 0);
});

// --- Inert transport ------------------------------------------------------------

check("envelope-round-trip", () => {
  const service = new NativeSessionService(serviceLimits());
  const session = service.open("owner-a", "s1");
  const made = service.make(session, "owner-a", "text", { tag: "text", text: "wire" });
  const json = encodeEnvelope(made.handle, made.facts);
  assert.equal(typeof json, "string");
  const decoded = decodeEnvelope(json);
  assert.equal(decoded.handle.kind, "value");
  assert.equal(decoded.handle.id, made.handle.id);
  assert.equal(decoded.handle.sessionId, "s1");
  assert.deepEqual(decoded.facts, { ...made.facts });
});

check("envelope-rejects-hostile", () => {
  const service = new NativeSessionService(serviceLimits());
  const session = service.open("owner-a", "s1");
  const made = service.make(session, "owner-a", "text", { tag: "text", text: "wire" });
  // oxlint-disable-next-line no-thenable -- Intentional thenable-shaped facts: negative control proving the envelope stays inert.
  expectSchemaError(() => encodeEnvelope(made.handle, { value: { then: () => 1 } }), "native-io");
  expectSchemaError(() => encodeEnvelope(made.handle, { run: () => 1 }), "native-io");
  expectSchemaError(() => decodeEnvelope(42), "invalid-request");
  expectSchemaError(() => decodeEnvelope("not json"), "invalid-request");
  expectSchemaError(
    () => decodeEnvelope(JSON.stringify({ handle: {}, facts: {} })),
    "invalid-request",
  );
  const good = JSON.parse(encodeEnvelope(made.handle, made.facts)) as Record<string, unknown>;
  expectSchemaError(
    () => decodeEnvelope(JSON.stringify({ ...good, raw: { tag: "smuggled" } })),
    "invalid-request",
  );
  expectSchemaError(
    () =>
      decodeEnvelope(
        JSON.stringify({
          ...good,
          handle: { ...(good["handle"] as Record<string, unknown>), raw: {} },
        }),
      ),
    "invalid-request",
  );
  expectSchemaError(
    // oxlint-disable-next-line no-thenable -- Intentional thenable-shaped wire facts: negative control for envelope decoding.
    () => decodeEnvelope(JSON.stringify({ ...good, facts: { then: 1 } })),
    "native-io",
  );
});

// --- Ownership ------------------------------------------------------------------

check("wrong-owner-rejects", () => {
  const service = new NativeSessionService(serviceLimits());
  const a = service.open("owner-a", "s1");
  service.open("owner-b", "s2");
  const value = service.make(a, "owner-a", "text", { tag: "text", text: "owned" });
  expectSchemaError(
    () => service.make(a, "owner-b", "text", { tag: "text", text: "x" }),
    "wrong-owner",
  );
  expectSchemaError(() => service.alias(a, "owner-b", value.handle), "wrong-owner");
  expectSchemaError(() => service.counters(a, "owner-b"), "wrong-owner");
  expectSchemaError(() => service.close(a, "owner-b"), "wrong-owner");
  // Cross-session use rejects even for one owner holding both sessions.
  const b = service.open("owner-a", "s3");
  expectSchemaError(() => service.alias(b, "owner-a", value.handle), "wrong-owner");
  // The cells are untouched by the rejected attempts.
  assert.equal(service.liveCellCount, 1);
});

check("stale-handle-rejects", () => {
  const service = new NativeSessionService(serviceLimits());
  const session = service.open("owner-a", "s1");
  const value = service.make(session, "owner-a", "text", { tag: "text", text: "stale" });
  // A forged generation no longer matches the live session generation.
  const wire = encodeHandleWire(value.handle);
  const forged = decodeHandleWire({ ...wire, generation: wire.generation + 1 });
  expectSchemaError(() => service.alias(session, "owner-a", forged), "stale-handle");
  // A handle naming an unknown session is stale, not merely foreign.
  const orphan = mintHandle("value", "ghost:h1", "ghost", "owner-a", 0);
  expectSchemaError(() => service.alias(session, "owner-a", orphan), "stale-handle");
  expectSchemaError(
    () => service.counters(mintHandle("session", "ghost:root", "ghost", "owner-a", 0), "owner-a"),
    "stale-handle",
  );
});

check("kind-mismatch-rejects", () => {
  const service = new NativeSessionService(serviceLimits());
  const session = service.open("owner-a", "s1");
  const value = service.make(session, "owner-a", "text", { tag: "text", text: "kind" });
  expectSchemaError(
    () => service.make(value.handle, "owner-a", "text", { tag: "text", text: "x" }),
    "invalid-request",
  );
  expectSchemaError(() => service.alias(session, "owner-a", session), "invalid-request");
  expectSchemaError(() => service.close(value.handle, "owner-a"), "invalid-request");
});

// --- Close reclaims children ------------------------------------------------------

check("close-reclaims", () => {
  const service = new NativeSessionService(serviceLimits());
  const a = service.open("owner-a", "s1");
  const b = service.open("owner-a", "s2");
  const value = service.make(a, "owner-a", "text", { tag: "text", text: "doomed" });
  service.alias(a, "owner-a", value.handle);
  service.make(a, "owner-a", "hostile_descriptor", {
    tag: "hostile",
    descriptor: "revoked_proxy",
  });
  service.make(b, "owner-a", "text", { tag: "text", text: "spared" });
  assert.equal(service.liveCellCount, 4);
  const receipt = service.close(a, "owner-a");
  assert.equal(receipt.sessionId, "s1");
  assert.equal(receipt.released, 3);
  assert.equal(receipt.remaining, 0);
  assert.equal(receipt.cellsReleased, 3);
  assert.equal(service.liveCellCount, 1);
  // Repeated close joins the first disposal; nothing is double-freed.
  const joined = service.close(a, "owner-a");
  assert.equal(joined.released, 3);
  assert.equal(joined.cellsReleased, 0);
  assert.equal(service.liveCellCount, 1);
  // Closed handles reject; the surviving session is unaffected.
  expectSchemaError(
    () => service.make(a, "owner-a", "text", { tag: "text", text: "x" }),
    "closed-handle",
  );
  expectSchemaError(() => service.alias(a, "owner-a", value.handle), "closed-handle");
  expectSchemaError(() => service.counters(a, "owner-a"), "closed-handle");
  service.make(b, "owner-a", "text", { tag: "text", text: "still live" });
  assert.equal(service.liveCellCount, 2);
});

// --- Caps and closed vocabularies --------------------------------------------------

check("caps-and-vocabulary", () => {
  const service = new NativeSessionService(tinyLimits());
  const session = service.open("owner-a", "s1");
  expectSchemaError(() => service.open("owner-a", "s2"), "resource-limit");
  // Vocabulary and payload bounds reject before any mint, so they run first.
  expectSchemaError(
    () => service.make(session, "owner-a", "native::eval", { tag: "text", text: "x" }),
    "unsupported-capability",
  );
  expectSchemaError(
    () =>
      service.make(session, "owner-a", "hostile_descriptor", {
        tag: "hostile",
        descriptor: "raw_function",
      }),
    "unsupported-capability",
  );
  expectSchemaError(
    () => service.make(session, "owner-a", "text", { tag: "text", text: "0123456789abcdef" }),
    "resource-limit",
  );
  expectSchemaError(
    () =>
      service.make(session, "owner-a", "ordered_entries", {
        tag: "entries",
        entries: [
          ["a", 1],
          ["b", 2],
          ["c", 3],
        ],
      }),
    "resource-limit",
  );
  expectSchemaError(
    () => service.make(session, "owner-a", "text", { tag: "entries", entries: [] }),
    "invalid-request",
  );
  expectSchemaError(() => service.make(session, "owner-a", "text", ["text"]), "invalid-request");
  service.make(session, "owner-a", "text", { tag: "text", text: "one" });
  service.make(session, "owner-a", "text", { tag: "text", text: "two" });
  expectSchemaError(
    () => service.make(session, "owner-a", "text", { tag: "text", text: "three" }),
    "resource-limit",
  );
});

console.log(
  JSON.stringify({
    kind: "can.native-values-session-check",
    schema_version: "1",
    checks: passed,
    count: passed.length,
  }),
);
