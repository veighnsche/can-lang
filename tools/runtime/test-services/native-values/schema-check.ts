// K01 bounded self-check: frozen schema mechanics only.
//
// Run with: bun tools/runtime/test-services/native-values/schema-check.ts
// Local controls only. No services, transports, timers, files, or live
// runtimes. Every loop below is bounded by a small constant.

import { strict as assert } from "node:assert";
import {
  checkFactsInert,
  checkLimits,
  checkRequest,
  checkResult,
  decodeHandleWire,
  describeOperation,
  encodeHandleWire,
  FORBIDDEN_ARG_KEYS,
  isNativeValueOperation,
  MECHANICAL_KINDS,
  mintHandle,
  NATIVE_OUTCOMES,
  NATIVE_VALUE_CATALOGUE,
  NATIVE_VALUE_LIMIT_KEYS,
  NATIVE_VALUE_OPERATIONS,
  NativeSchemaError,
  type NativeValueLimits,
} from "./schema.ts";
import { NativeHandleRegistry } from "./schema-handles.ts";

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

const DIGEST = `sha256:${"ab".repeat(32)}`;

function baseRequest(operation: string, args: Record<string, unknown>): Record<string, unknown> {
  return {
    schemaVersion: "1",
    runId: "run1",
    operationId: "op1",
    ownerGrant: "owner-a",
    operation,
    argumentsDigest: DIGEST,
    deadlineMs: { clock: "n-monotonic", ms: 1000 },
    args,
  };
}

function tinyLimits(): NativeValueLimits {
  return checkLimits({
    maxSessions: 2,
    maxHandlesPerSession: 4,
    maxPendingActions: 1,
    maxObserveEntries: 8,
    maxObserveBytes: 64,
    maxGatesPerSession: 1,
    maxFaultsPerSession: 1,
  });
}

// --- Catalogue finiteness ---------------------------------------------------

check("catalogue is finite, frozen, and unique", () => {
  assert.equal(NATIVE_VALUE_CATALOGUE.length, 11);
  assert.equal(NATIVE_VALUE_OPERATIONS.length, 11);
  assert(Object.isFrozen(NATIVE_VALUE_CATALOGUE));
  assert(Object.isFrozen(NATIVE_VALUE_CATALOGUE[0].args));
  const names = NATIVE_VALUE_CATALOGUE.map((entry) => entry.name);
  const identities = NATIVE_VALUE_CATALOGUE.map((entry) => entry.identity);
  assert.equal(new Set(names).size, names.length);
  assert.equal(new Set(identities).size, identities.length);
  for (const entry of NATIVE_VALUE_CATALOGUE) {
    assert(isNativeValueOperation(entry.name));
    assert.equal(describeOperation(entry.name), entry);
  }
});

check("every entry declares assimilation and ownership contracts", () => {
  for (const entry of NATIVE_VALUE_CATALOGUE) {
    assert(["sync", "dispatch", "poll"].includes(entry.timing));
    assert(["none", "native-only"].includes(entry.assimilation));
    assert(entry.reply.length > 0);
    assert(entry.facts.length > 0);
    for (const arg of entry.args) {
      assert(arg.name.length > 0 && arg.type.length > 0 && arg.note.length > 0);
    }
    // Only native.invoke may dispatch async work; only native.settle polls
    // it; nothing else touches assimilation.
    if (entry.name === "native.invoke") {
      assert.equal(entry.timing, "dispatch");
      assert.equal(entry.assimilation, "native-only");
    } else if (entry.name === "native.settle") {
      assert.equal(entry.timing, "poll");
      assert.equal(entry.assimilation, "none");
    } else {
      assert.equal(entry.timing, "sync");
      assert.equal(entry.assimilation, "none");
    }
  }
  const invoke = describeOperation("native.invoke");
  assert(invoke.handlesOut.includes("action"));
  const settle = describeOperation("native.settle");
  assert.deepEqual([...settle.handlesIn], ["action"]);
});

check("catalogue declares no module/export/eval or oracle argument", () => {
  const forbidden = new Set<string>(FORBIDDEN_ARG_KEYS as readonly string[]);
  for (const entry of NATIVE_VALUE_CATALOGUE) {
    for (const arg of entry.args) {
      assert(!forbidden.has(arg.name), `forbidden arg in ${entry.name}: ${arg.name}`);
    }
    for (const fact of entry.facts) {
      assert(!forbidden.has(fact), `forbidden fact in ${entry.name}: ${fact}`);
    }
  }
});

// --- Unknown operations -----------------------------------------------------

check("unknown operations are rejected", () => {
  for (const name of [
    "native.qualify_json",
    "native.test_array",
    "native.eval",
    "native",
    "",
    "NATIVE.OPEN",
    "native.open ",
    "browser.open",
  ]) {
    assert(!isNativeValueOperation(name));
    expectSchemaError(() => describeOperation(name), "unsupported-capability");
    expectSchemaError(() => checkRequest(baseRequest(name, {})), "unsupported-capability");
  }
});

// --- Request envelope --------------------------------------------------------

check("valid requests for every operation pass", () => {
  const argsByOperation: Record<string, Record<string, unknown>> = {
    "native.open": { runtime: "bun", observer: "obs1", scope: "scope1", limits: tinyLimits() },
    "native.describe": { session: { kind: "session" }, api: "json" },
    "native.make": {
      session: { kind: "session" },
      kind: "text",
      payload: { tag: "text", text: "hi" },
    },
    "native.invoke": { session: { kind: "session" }, operation: "encodeJSON", arguments: [] },
    "native.settle": { action: { kind: "action" }, deadline: { ms: 10 } },
    "native.observe": {
      session: { kind: "session" },
      handles: [],
      kind: "ieee_bits",
      bounds: { entries: 1, bytes: 8 },
    },
    "native.gate": { session: { kind: "session" }, gate: { name: "g" }, action: { ref: "a" } },
    "native.release": { gate: { kind: "gate" } },
    "native.fault": { session: { kind: "session" }, target: "json", mode: "throw" },
    "native.restore": { fault: { kind: "fault" } },
    "native.close": { session: { kind: "session" }, deadline: { ms: 10 } },
  };
  for (const name of NATIVE_VALUE_OPERATIONS) {
    const request = checkRequest(baseRequest(name, argsByOperation[name] ?? {}));
    assert.equal(request.operation, name);
    assert(Object.isFrozen(request));
  }
});

check("malformed requests reject before any effect", () => {
  const good = () => baseRequest("native.make", { session: {}, kind: "text", payload: {} });
  expectSchemaError(() => checkRequest({ ...good(), schemaVersion: "2" }), "invalid-request");
  expectSchemaError(
    () => checkRequest({ ...good(), argumentsDigest: "md5:zzz" }),
    "invalid-request",
  );
  expectSchemaError(() => checkRequest({ ...good(), operationId: "" }), "invalid-request");
  expectSchemaError(() => checkRequest({ ...good(), extra: 1 }), "invalid-request");
  expectSchemaError(() => checkRequest({ ...good(), args: "text" }), "invalid-request");
  expectSchemaError(
    () => checkRequest(baseRequest("native.make", { session: {} })),
    "invalid-request",
  );
  expectSchemaError(
    () =>
      checkRequest(
        baseRequest("native.make", { session: {}, kind: "text", payload: {}, module: "fs" }),
      ),
    "invalid-request",
  );
  // Forbidden keys reject even when nested inside a declared argument.
  expectSchemaError(
    () =>
      checkRequest(
        baseRequest("native.make", {
          session: {},
          kind: "text",
          payload: { nested: { expected: 1 } },
        }),
      ),
    "invalid-request",
  );
  for (const key of FORBIDDEN_ARG_KEYS) {
    expectSchemaError(
      () =>
        checkRequest(baseRequest("native.describe", { session: {}, api: "x", [key]: "smuggled" })),
      "invalid-request",
    );
  }
});

// --- Inert replies ------------------------------------------------------------

check("hostile reply payloads reject", () => {
  checkFactsInert({ tag: "f64_bits", hi: 0, lo: 0 }, "facts");
  checkFactsInert({ lexeme: "9007199254740993", path: "$[0]" }, "facts");
  expectSchemaError(() => checkFactsInert({ run: () => 1 }, "facts"), "native-io");
  expectSchemaError(() => checkFactsInert({ value: Promise.resolve(1) }, "facts"), "native-io");
  // oxlint-disable-next-line no-thenable -- Intentional thenable: negative control proving hostile replies reject.
  expectSchemaError(() => checkFactsInert({ value: { then: () => 1 } }, "facts"), "native-io");
  expectSchemaError(() => checkFactsInert({ expected: 1 }, "facts"), "native-io");
  expectSchemaError(() => checkFactsInert({ module: "fs" }, "facts"), "native-io");
  expectSchemaError(() => checkFactsInert(Symbol("x"), "facts"), "native-io");
});

check("result outcome/kind agreement holds", () => {
  assert.equal(NATIVE_OUTCOMES.length, 5);
  assert.equal(MECHANICAL_KINDS.length, 14);
  const good = {
    schemaVersion: "1",
    runId: "run1",
    operationId: "op1",
    outcome: "completed",
    kind: "ok",
    facts: { tag: "bool", value: true },
  };
  assert.equal(checkResult(good).outcome, "completed");
  expectSchemaError(() => checkResult({ ...good, kind: "native-io" }), "transport-failure");
  expectSchemaError(
    () => checkResult({ ...good, outcome: "failed", kind: "ok" }),
    "transport-failure",
  );
  expectSchemaError(() => checkResult({ ...good, outcome: "nope" }), "transport-failure");
  // oxlint-disable-next-line no-thenable -- Intentional thenable-shaped fact: negative control for result validation.
  expectSchemaError(() => checkResult({ ...good, facts: { then: 1 } }), "native-io");
});

// --- Handles ------------------------------------------------------------------

check("handle wire round-trips and rejects unknown fields", () => {
  const handle = mintHandle("value", "s1:h1", "s1", "owner-a", 0);
  assert(Object.isFrozen(handle));
  const wire = encodeHandleWire(handle);
  const decoded = decodeHandleWire(JSON.parse(JSON.stringify(wire)));
  assert.equal(decoded.kind, "value");
  assert.equal(decoded.id, "s1:h1");
  expectSchemaError(() => decodeHandleWire({ ...wire, raw: {} }), "invalid-request");
  expectSchemaError(() => decodeHandleWire({ ...wire, kind: "process" }), "invalid-request");
  expectSchemaError(() => decodeHandleWire({ ...wire, id: "" }), "invalid-request");
  expectSchemaError(() => decodeHandleWire({ ...wire, generation: -1 }), "invalid-request");
  expectSchemaError(() => decodeHandleWire([]), "invalid-request");
});

// --- Registry: caps and move rules ----------------------------------------------

check("handle caps reject instead of growing", () => {
  const registry = new NativeHandleRegistry(tinyLimits());
  const session = registry.openSession("owner-a", "s1");
  for (let i = 0; i < 4; i += 1) {
    registry.mint(session, "owner-a", "value");
  }
  expectSchemaError(() => registry.mint(session, "owner-a", "value"), "resource-limit");
  expectSchemaError(() => registry.openSession("owner-a", "s1"), "kind-collision");
  registry.openSession("owner-a", "s2");
  expectSchemaError(() => registry.openSession("owner-a", "s3"), "resource-limit");
});

check("pending-action, gate, and fault sub-caps hold", () => {
  const registry = new NativeHandleRegistry(tinyLimits());
  const session = registry.openSession("owner-a", "s1");
  registry.mint(session, "owner-a", "action");
  expectSchemaError(() => registry.mint(session, "owner-a", "action"), "resource-limit");
  registry.mint(session, "owner-a", "gate");
  expectSchemaError(() => registry.mint(session, "owner-a", "gate"), "resource-limit");
  registry.mint(session, "owner-a", "fault");
  expectSchemaError(() => registry.mint(session, "owner-a", "fault"), "resource-limit");
});

check("wrong-owner and cross-session use reject without moving", () => {
  const registry = new NativeHandleRegistry(tinyLimits());
  const a = registry.openSession("owner-a", "s1");
  const b = registry.openSession("owner-b", "s2");
  const value = registry.mint(a, "owner-a", "value");
  registry.use(value, "owner-a");
  expectSchemaError(() => registry.use(value, "owner-b"), "wrong-owner");
  const foreign = registry.mint(b, "owner-b", "value");
  expectSchemaError(() => registry.use(foreign, "owner-a"), "wrong-owner");
  // A handle is bound to its own session id; presenting it under another
  // session's authority never transfers it.
  assert.notEqual(value.sessionId, b.sessionId);
  expectSchemaError(() => registry.closeSession(a, "owner-b"), "wrong-owner");
});

check("close reclaims children and stale handles reject", () => {
  const registry = new NativeHandleRegistry(tinyLimits());
  const session = registry.openSession("owner-a", "s1");
  const value = registry.mint(session, "owner-a", "value");
  const action = registry.mint(session, "owner-a", "action");
  registry.settleAction(action, "owner-a");
  const receipt = registry.closeSession(session, "owner-a");
  assert.equal(receipt.released, 2);
  assert.equal(receipt.remaining, 0);
  // Repeated close joins the same disposal.
  assert.equal(registry.closeSession(session, "owner-a"), receipt);
  expectSchemaError(() => registry.use(value, "owner-a"), "closed-handle");
  expectSchemaError(() => registry.mint(session, "owner-a", "value"), "closed-handle");
});

check("release-once joins repeats", () => {
  const registry = new NativeHandleRegistry(tinyLimits());
  const session = registry.openSession("owner-a", "s1");
  const gate = registry.mint(session, "owner-a", "gate");
  assert.equal(registry.releaseOnce(gate, "owner-a"), false);
  assert.equal(registry.releaseOnce(gate, "owner-a"), true);
  expectSchemaError(() => registry.releaseOnce(gate, "owner-b"), "wrong-owner");
});

// --- Limits ---------------------------------------------------------------------

check("limits must be finite and complete", () => {
  assert.equal(NATIVE_VALUE_LIMIT_KEYS.length, 7);
  for (const key of NATIVE_VALUE_LIMIT_KEYS) {
    const base = tinyLimits() as unknown as Record<string, unknown>;
    for (const bad of [0, -1, 1.5, Number.NaN, "unlimited", undefined]) {
      expectSchemaError(() => checkLimits({ ...base, [key]: bad }), "invalid-request");
    }
    const missing = { ...base };
    delete missing[key];
    expectSchemaError(() => checkLimits(missing), "invalid-request");
  }
  expectSchemaError(() => checkLimits({ ...tinyLimits(), maxAnything: 1 }), "invalid-request");
  expectSchemaError(() => checkLimits([]), "invalid-request");
});

console.log(
  JSON.stringify({
    kind: "can.native-values-schema-check",
    schemaVersion: "1",
    checks: passed,
    count: passed.length,
  }),
);
