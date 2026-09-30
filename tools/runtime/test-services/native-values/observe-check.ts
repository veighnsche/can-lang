// K03 bounded self-check: exact observations and seals.
//
// Run with: bun tools/runtime/test-services/native-values/observe-check.ts
// Local controls only. No services, transports, timers, files, live hosts,
// or live runtimes. Every loop below is bounded by a small constant.

import { strict as assert } from "node:assert";
import {
  checkFactsInert,
  checkLimits,
  type NativeHandle,
  type NativeValueLimits,
  NativeSchemaError,
} from "./schema.ts";
import {
  createHostileValue,
  decodeEnvelope,
  encodeEnvelope,
  NATIVE_HOSTILE_DESCRIPTORS,
  zeroHostileCounters,
} from "./session.ts";
import {
  checkObserveBounds,
  classifyF64,
  f64BitsOf,
  f64FromBits,
  NativeObserveService,
  tagNumber,
  type ObserveBounds,
} from "./observe.ts";

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

function roomyLimits(): NativeValueLimits {
  return checkLimits({
    max_sessions: 4,
    max_handles_per_session: 64,
    maxPendingActions: 2,
    maxObserveEntries: 16,
    maxObserveBytes: 4096,
    maxGatesPerSession: 2,
    maxFaultsPerSession: 2,
  });
}

function tinyLimits(): NativeValueLimits {
  return checkLimits({
    max_sessions: 1,
    max_handles_per_session: 8,
    maxPendingActions: 1,
    maxObserveEntries: 2,
    maxObserveBytes: 64,
    maxGatesPerSession: 1,
    maxFaultsPerSession: 1,
  });
}

function fullBounds(limits: NativeValueLimits): ObserveBounds {
  return checkObserveBounds(
    { maxEntries: limits.maxObserveEntries, maxBytes: limits.maxObserveBytes },
    limits,
  );
}

function f64Payload(hi: number, lo: number): Record<string, unknown> {
  return { tag: "f64_bits", hi, lo };
}

// --- Normalization: -0 vs 0, NaN payloads, subnormals stay distinct ---------

check("bits-are-exact", () => {
  const limits = roomyLimits();
  const bounds = fullBounds(limits);
  const service = new NativeObserveService(limits);
  const session = service.open("owner-a", "s1");
  const posZero = service.make(session, "owner-a", "primitive_bits", f64Payload(0, 0));
  const negZero = service.make(session, "owner-a", "primitive_bits", f64Payload(0x80000000, 0));
  const facts = service.observe(
    session,
    "owner-a",
    [posZero.handle, negZero.handle],
    "ieee_bits",
    bounds,
  );
  const pos = facts.observations[0]?.result as Record<string, unknown>;
  const neg = facts.observations[1]?.result as Record<string, unknown>;
  assert.equal(pos["hi"], 0);
  assert.equal(neg["hi"], 0x80000000);
  assert.equal(pos["class"], "zero");
  assert.equal(neg["class"], "zero");
  assert.equal(pos["sign"], 0);
  assert.equal(neg["sign"], 1);
  assert.notDeepEqual({ hi: pos["hi"], lo: pos["lo"] }, { hi: neg["hi"], lo: neg["lo"] });

  const subnormal = service.make(session, "owner-a", "primitive_bits", f64Payload(0, 1));
  const sub = service.observe(session, "owner-a", [subnormal.handle], "ieee_bits", bounds)
    .observations[0]?.result as Record<string, unknown>;
  assert.equal(sub["class"], "subnormal");
  assert.equal(sub["exponent"], 0);
  assert.equal(sub["mantissa_lo"], 1);

  const inf = service.observe(
    session,
    "owner-a",
    [service.make(session, "owner-a", "primitive_bits", f64Payload(0x7ff00000, 0)).handle],
    "ieee_bits",
    bounds,
  ).observations[0]?.result as Record<string, unknown>;
  assert.equal(inf["class"], "inf");

  // NaN payloads are preserved bit-for-bit: no canonicalization on the way in.
  const canonical = service.make(session, "owner-a", "primitive_bits", f64Payload(0x7ff80000, 0));
  const payload = service.make(
    session,
    "owner-a",
    "primitive_bits",
    f64Payload(0x7ff80000, 0x12345678),
  );
  const nans = service.observe(
    session,
    "owner-a",
    [canonical.handle, payload.handle],
    "ieee_bits",
    bounds,
  );
  const nanA = nans.observations[0]?.result as Record<string, unknown>;
  const nanB = nans.observations[1]?.result as Record<string, unknown>;
  assert.equal(nanA["class"], "nan");
  assert.equal(nanB["class"], "nan");
  assert.equal(nanA["lo"], 0);
  assert.equal(nanB["lo"], 0x12345678);
  assert.equal(nanB["mantissa_lo"], 0x12345678);

  // The explicit bit read re-derives words from the raw value and agrees with
  // the construction literals for exactly representable values.
  const reread = service.effectfulRead(session, "owner-a", negZero.handle, "bits");
  assert.equal(reread["outcome"], "returned");
  assert.equal(reread["hi"], 0x80000000);
  assert.equal(reread["matches_construction"], true);

  // Signaling NaNs report the construction literals exactly on literal
  // paths: engine quieting through a JS number never moves observed bits.
  const snan = service.make(session, "owner-a", "primitive_bits", f64Payload(0x7ff00001, 0));
  const snanBits = service.observe(session, "owner-a", [snan.handle], "ieee_bits", bounds)
    .observations[0]?.result as Record<string, unknown>;
  assert.equal(snanBits["hi"], 0x7ff00001);
  assert.equal(snanBits["class"], "nan");
  const snanLexeme = service.observe(session, "owner-a", [snan.handle], "lexeme", bounds)
    .observations[0]?.result as Record<string, unknown>;
  assert.equal(snanLexeme["tag"], "f64_bits");
  assert.equal(snanLexeme["hi"], 0x7ff00001);

  // Non-floats report explicit gaps, never invented bits.
  const text = service.make(session, "owner-a", "text", { tag: "text", text: "not a float" });
  const gap = service.observe(session, "owner-a", [text.handle], "ieee_bits", bounds)
    .observations[0]?.result as Record<string, unknown>;
  assert.equal(gap["gap"], "not-a-float");
  assert.equal(gap["scalar"], "text");
});

// --- Rounding: nearest-even f64 outcomes are exact bit facts ----------------

check("rounding-is-exact", () => {
  assert.deepEqual(f64BitsOf(0.1), { hi: 0x3fb99999, lo: 0x9999999a });
  assert.deepEqual(f64BitsOf(1 / 3), { hi: 0x3fd55555, lo: 0x55555555 });
  assert.notDeepEqual(f64BitsOf(0.1 + 0.2), f64BitsOf(0.3));
  assert.equal(f64FromBits(0x80000000, 0), -0);
  assert.equal(Object.is(f64FromBits(0x80000000, 0), -0), true);
  assert.deepEqual(classifyF64(0x7ff00000, 0), {
    class: "inf",
    sign: 0,
    exponent: 0x7ff,
    mantissa_hi: 0,
    mantissa_lo: 0,
  });

  const limits = roomyLimits();
  const bounds = fullBounds(limits);
  const service = new NativeObserveService(limits);
  const session = service.open("owner-a", "s1");
  for (const value of [0.1, 0.1 + 0.2, 0.3, 1 / 3, -0]) {
    const { hi, lo } = f64BitsOf(value);
    const made = service.make(session, "owner-a", "primitive_bits", f64Payload(hi, lo));
    const back = service.observe(session, "owner-a", [made.handle], "ieee_bits", bounds)
      .observations[0]?.result as Record<string, unknown>;
    assert.equal(back["hi"], hi);
    assert.equal(back["lo"], lo);
  }
  const sums = service.observe(
    session,
    "owner-a",
    [
      service.make(session, "owner-a", "primitive_bits", {
        tag: "f64_bits",
        ...f64BitsOf(0.1 + 0.2),
      }).handle,
      service.make(session, "owner-a", "primitive_bits", {
        tag: "f64_bits",
        ...f64BitsOf(0.3),
      }).handle,
    ],
    "ieee_bits",
    bounds,
  );
  const a = sums.observations[0]?.result as Record<string, unknown>;
  const b = sums.observations[1]?.result as Record<string, unknown>;
  assert.notEqual(a["lo"], b["lo"]);
});

// --- Lexemes: integers as text, everything else as bits ---------------------

check("lexemes-are-exact", () => {
  assert.deepEqual(tagNumber(42), { tag: "integer", lexeme: "42" });
  assert.deepEqual(tagNumber(-7), { tag: "integer", lexeme: "-7" });
  assert.deepEqual(tagNumber(Number.MAX_SAFE_INTEGER), {
    tag: "integer",
    lexeme: "9007199254740991",
  });
  // 2^53 is not a safe integer: bits, not a lossy lexeme.
  assert.equal(tagNumber(2 ** 53)["tag"], "f64_bits");
  // -0 never collapses onto the "0" lexeme.
  const negZeroTag = tagNumber(-0);
  assert.equal(negZeroTag["tag"], "f64_bits");
  assert.equal(negZeroTag["hi"], 0x80000000);
  assert.deepEqual(tagNumber(0), { tag: "integer", lexeme: "0" });

  const limits = roomyLimits();
  const bounds = fullBounds(limits);
  const service = new NativeObserveService(limits);
  const session = service.open("owner-a", "s1");
  const entries = service.make(session, "owner-a", "ordered_entries", {
    tag: "entries",
    entries: [
      ["i", 42],
      ["frac", 0.1],
      ["negzero", -0],
      ["zero", 0],
      ["s", "hi"],
      ["b", true],
      ["n", null],
      ["nest", [1, 2]],
    ],
  });
  const facts = service.observe(session, "owner-a", [entries.handle], "lexeme", bounds)
    .observations[0]?.result as Record<string, unknown>;
  const tagged = facts["entries"] as Array<Record<string, unknown>>;
  assert.equal(facts["gap"], undefined);
  assert.equal(tagged.length, 8);
  assert.deepEqual(tagged[0], { key: "i", value: { tag: "integer", lexeme: "42" } });
  const frac = tagged[1]?.["value"] as Record<string, unknown>;
  assert.deepEqual({ hi: frac["hi"], lo: frac["lo"] }, f64BitsOf(0.1));
  const negzero = tagged[2]?.["value"] as Record<string, unknown>;
  assert.equal(negzero["tag"], "f64_bits");
  assert.equal(negzero["hi"], 0x80000000);
  assert.deepEqual(tagged[3], { key: "zero", value: { tag: "integer", lexeme: "0" } });
  assert.deepEqual(tagged[4], { key: "s", value: { tag: "text", text: "hi" } });
  assert.deepEqual(tagged[5], { key: "b", value: { tag: "boolean", value: true } });
  assert.deepEqual(tagged[6], { key: "n", value: { tag: "null" } });
  assert.deepEqual(tagged[7], { key: "nest", value: { tag: "json", gap: "nested-opaque" } });

  // The entries kind reports the same ordered pairs; other scalars gap.
  const viaEntries = service.observe(session, "owner-a", [entries.handle], "entries", bounds)
    .observations[0]?.result as Record<string, unknown>;
  assert.deepEqual(viaEntries["entries"], facts["entries"]);
  const text = service.make(session, "owner-a", "text", { tag: "text", text: "exact" });
  const textLexeme = service.observe(session, "owner-a", [text.handle], "lexeme", bounds)
    .observations[0]?.result as Record<string, unknown>;
  assert.deepEqual(textLexeme, { tag: "text", text: "exact" });
  const textEntries = service.observe(session, "owner-a", [text.handle], "entries", bounds)
    .observations[0]?.result as Record<string, unknown>;
  assert.equal(textEntries["gap"], "not-entries");

  // Hostile cells gap on value kinds without any touch.
  const hostile = service.make(session, "owner-a", "hostile_descriptor", {
    tag: "hostile",
    descriptor: "getter_then",
  });
  const hostileLexeme = service.observe(session, "owner-a", [hostile.handle], "lexeme", bounds)
    .observations[0]?.result as Record<string, unknown>;
  assert.equal(hostileLexeme["gap"], "effectful");
  assert.equal(hostileLexeme["descriptor"], "getter_then");
  assert.equal(service.counters(session, "owner-a").getterReads, 0);
});

// --- Alias identity: two handles one cell vs two cells ----------------------

check("aliases-share-identity", () => {
  const limits = roomyLimits();
  const bounds = fullBounds(limits);
  const service = new NativeObserveService(limits);
  const session = service.open("owner-a", "s1");
  const payload = { tag: "text", text: "same bytes" };
  const first = service.make(session, "owner-a", "text", payload);
  const second = service.make(session, "owner-a", "text", payload);
  assert.notEqual(first.facts.cell, second.facts.cell);

  // No aliases yet: identity reports cells, no groups.
  const before = service.observe(
    session,
    "owner-a",
    [first.handle, second.handle],
    "identity",
    bounds,
  );
  assert.equal(before.observations[0]?.cell, first.facts.cell);
  assert.equal(before.observations[1]?.cell, second.facts.cell);
  assert.deepEqual(before.alias_groups, []);

  // Re-aliasing changes the identity facts: one group appears.
  const twin = service.alias(session, "owner-a", first.handle);
  assert.equal(twin.facts.cell, first.facts.cell);
  assert.equal(twin.facts.alias_of, first.handle.id);
  const retwin = service.alias(session, "owner-a", twin.handle);
  assert.equal(retwin.facts.cell, first.facts.cell);
  assert.equal(retwin.facts.alias_of, twin.handle.id);
  const after = service.observe(
    session,
    "owner-a",
    [first.handle, twin.handle, retwin.handle, second.handle],
    "identity",
    bounds,
  );
  assert.equal(after.alias_groups?.length, 1);
  const group = after.alias_groups?.[0];
  assert.equal(group?.cell, first.facts.cell);
  assert.deepEqual(
    [...(group?.handles ?? [])].sort(),
    [first, twin, retwin].map((h) => h.handle.id).sort(),
  );

  // Aliasing the other handle swaps in a second group; groups stay distinct.
  const other = service.alias(session, "owner-a", second.handle);
  const swapped = service.observe(
    session,
    "owner-a",
    [first.handle, twin.handle, second.handle, other.handle],
    "identity",
    bounds,
  );
  assert.equal(swapped.alias_groups?.length, 2);
  const cells = (swapped.alias_groups ?? []).map((entry) => entry.cell).sort();
  assert.deepEqual(cells, [first.facts.cell, second.facts.cell].sort());

  // Hostile aliases share the cell without touching the fixture.
  const hostile = service.make(session, "owner-a", "hostile_descriptor", {
    tag: "hostile",
    descriptor: "throwing_thenable",
  });
  const hostileTwin = service.alias(session, "owner-a", hostile.handle);
  assert.equal(hostileTwin.facts.cell, hostile.facts.cell);
  assert.equal(service.counters(session, "owner-a").thenCalls, 0);
});

// --- Zero implicit reads: every non-read path leaves fixtures untouched -----

check("observe-is-non-effectful", () => {
  const limits = roomyLimits();
  const bounds = fullBounds(limits);
  const service = new NativeObserveService(limits);
  const session = service.open("owner-a", "s1");
  const handles: NativeHandle<"value">[] = [];
  for (const descriptor of NATIVE_HOSTILE_DESCRIPTORS) {
    const made = service.make(session, "owner-a", "hostile_descriptor", {
      tag: "hostile",
      descriptor,
    });
    handles.push(made.handle);
    handles.push(service.alias(session, "owner-a", made.handle).handle);
  }
  const kinds = [
    "scalar_tag",
    "descriptor",
    "ieee_bits",
    "lexeme",
    "entries",
    "identity",
    "counters",
  ] as const;
  for (const kind of kinds) {
    const facts = service.observe(session, "owner-a", handles, kind, bounds);
    assert.equal(facts.observations.length, handles.length);
    checkFactsInert(facts, `${kind} facts`);
    // Observed facts cross the K02 envelope untouched.
    const json = encodeEnvelope(handles[0] as NativeHandle<"value">, {
      kind,
      cells: facts.observations.length,
    });
    assert.deepEqual(decodeEnvelope(json).facts, { kind, cells: facts.observations.length });
  }
  const tags = service.observe(
    session,
    "owner-a",
    [handles[0] as NativeHandle<"value">],
    "scalar_tag",
    bounds,
  ).observations[0]?.result as Record<string, unknown>;
  assert.equal(tags["scalar"], "hostile");
  assert.equal(tags["descriptor"], "getter_then");
  const counters = service.counters(session, "owner-a");
  assert.equal(counters.observations, kinds.length + 1);
  assert.equal(counters.explicitReads, 0);
  assert.equal(counters.getterReads, 0);
  assert.equal(counters.thenCalls, 0);
  assert.equal(counters.proxyTraps, 0);
  const empty = service.interval(session, "owner-a", 0);
  assert.deepEqual(empty.reads, []);
  assert.deepEqual(empty.hostileDelta, { getterReads: 0, thenCalls: 0, proxyTraps: 0 });
});

// --- Assimilation: thenables are touched only by explicit counted reads -----

check("effectful-reads-are-explicit-and-counted", () => {
  const limits = roomyLimits();
  const service = new NativeObserveService(limits);
  const session = service.open("owner-a", "s1");
  const getter = service.make(session, "owner-a", "hostile_descriptor", {
    tag: "hostile",
    descriptor: "getter_then",
  });
  const throwing = service.make(session, "owner-a", "hostile_descriptor", {
    tag: "hostile",
    descriptor: "throwing_thenable",
  });
  const revoked = service.make(session, "owner-a", "hostile_descriptor", {
    tag: "hostile",
    descriptor: "revoked_proxy",
  });

  // A property touch fires the getter and reports the inert outcome.
  const touched = service.effectfulRead(session, "owner-a", getter.handle, "property_then");
  assert.equal(touched["outcome"], "returned");
  assert.equal(touched["typeof"], "undefined");
  assert.equal(touched["seq"], 1);
  assert.equal(service.counters(session, "owner-a").getterReads, 1);

  // Reading `then` off the throwing thenable does NOT call it: the function
  // comes back opaque and thenCalls stays zero. Read is not assimilation.
  const peeked = service.effectfulRead(session, "owner-a", throwing.handle, "property_then");
  assert.equal(peeked["outcome"], "returned");
  assert.equal(peeked["typeof"], "function");
  assert.equal(peeked["gap"], "callable-opaque");
  assert.equal(service.counters(session, "owner-a").thenCalls, 0);

  // Only the explicit call op assimilates, and the throw stays inert.
  const called = service.effectfulRead(session, "owner-a", throwing.handle, "call_then");
  assert.equal(called["outcome"], "threw");
  assert.equal(called["name"], "Error");
  assert.equal(service.counters(session, "owner-a").thenCalls, 1);

  // The getter fixture has nothing callable: one more getter fire, no call.
  const uncallable = service.effectfulRead(session, "owner-a", getter.handle, "call_then");
  assert.equal(uncallable["outcome"], "not-callable");
  assert.equal(uncallable["typeof"], "undefined");
  assert.equal(service.counters(session, "owner-a").getterReads, 2);

  // The revoked proxy throws before any trap runs, on either op.
  for (const op of ["property_then", "call_then"] as const) {
    const broken = service.effectfulRead(session, "owner-a", revoked.handle, op);
    assert.equal(broken["outcome"], "threw");
    assert.equal(broken["name"], "TypeError");
  }
  assert.equal(service.counters(session, "owner-a").proxyTraps, 0);

  // Scalar reads are counted too, with zero hostile delta.
  const f64 = service.make(session, "owner-a", "primitive_bits", f64Payload(0x3ff00000, 0));
  const text = service.make(session, "owner-a", "text", { tag: "text", text: "read me" });
  const entries = service.make(session, "owner-a", "ordered_entries", {
    tag: "entries",
    entries: [["k", 1]],
  });
  const bits = service.effectfulRead(session, "owner-a", f64.handle, "bits");
  assert.equal(bits["matches_construction"], true);
  const back = service.effectfulRead(session, "owner-a", text.handle, "text");
  assert.equal(back["text"], "read me");
  const pairs = service.effectfulRead(session, "owner-a", entries.handle, "entries");
  assert.deepEqual(pairs["entries"], [{ key: "k", value: { tag: "integer", lexeme: "1" } }]);

  // Wrong-op attempts reject without counting or touching.
  expectSchemaError(
    () => service.effectfulRead(session, "owner-a", text.handle, "bits"),
    "invalid-request",
  );
  expectSchemaError(
    () => service.effectfulRead(session, "owner-a", text.handle, "property_then"),
    "invalid-request",
  );
  expectSchemaError(
    () => service.effectfulRead(session, "owner-a", f64.handle, "call_then"),
    "invalid-request",
  );
  expectSchemaError(
    () => service.effectfulRead(session, "owner-a", getter.handle, "await"),
    "unsupported-capability",
  );
  const counters = service.counters(session, "owner-a");
  assert.equal(counters.explicitReads, 9);
  assert.equal(counters.observations, 0);
  assert.deepEqual(
    {
      getterReads: counters.getterReads,
      thenCalls: counters.thenCalls,
      proxyTraps: counters.proxyTraps,
    },
    { getterReads: 2, thenCalls: 1, proxyTraps: 0 },
  );
});

// --- Counter intervals slice the explicit-read log exactly ------------------

check("counter-intervals", () => {
  const limits = roomyLimits();
  const service = new NativeObserveService(limits);
  const session = service.open("owner-a", "s1");
  const getter = service.make(session, "owner-a", "hostile_descriptor", {
    tag: "hostile",
    descriptor: "getter_then",
  });
  const throwing = service.make(session, "owner-a", "hostile_descriptor", {
    tag: "hostile",
    descriptor: "throwing_thenable",
  });
  const f64 = service.make(session, "owner-a", "primitive_bits", f64Payload(0, 0));
  service.effectfulRead(session, "owner-a", getter.handle, "property_then");
  service.effectfulRead(session, "owner-a", throwing.handle, "call_then");
  service.effectfulRead(session, "owner-a", f64.handle, "bits");

  const all = service.interval(session, "owner-a", 0);
  assert.equal(all.from, 0);
  assert.equal(all.to, 3);
  assert.deepEqual(
    all.reads.map((entry) => entry.op),
    ["property_then", "call_then", "bits"],
  );
  assert.deepEqual(all.hostileDelta, { getterReads: 1, thenCalls: 1, proxyTraps: 0 });

  const middle = service.interval(session, "owner-a", 1, 2);
  assert.equal(middle.reads.length, 1);
  assert.equal(middle.reads[0]?.seq, 2);
  assert.deepEqual(middle.hostileDelta, { getterReads: 0, thenCalls: 1, proxyTraps: 0 });

  const tail = service.interval(session, "owner-a", 2, 3);
  assert.deepEqual(tail.hostileDelta, { getterReads: 0, thenCalls: 0, proxyTraps: 0 });

  const point = service.interval(session, "owner-a", 3, 3);
  assert.deepEqual(point.reads, []);

  expectSchemaError(() => service.interval(session, "owner-a", -1), "invalid-request");
  expectSchemaError(() => service.interval(session, "owner-a", 2, 1), "invalid-request");
  expectSchemaError(() => service.interval(session, "owner-a", 0, 4), "invalid-request");

  // Wide ranges reject under tight caps.
  const tight = new NativeObserveService(tinyLimits());
  const ts = tight.open("owner-a", "s1");
  const a = tight.make(ts, "owner-a", "text", { tag: "text", text: "a" });
  tight.effectfulRead(ts, "owner-a", a.handle, "text");
  tight.effectfulRead(ts, "owner-a", a.handle, "text");
  tight.effectfulRead(ts, "owner-a", a.handle, "text");
  expectSchemaError(() => tight.interval(ts, "owner-a", 0, 3), "resource-limit");
  assert.equal(tight.interval(ts, "owner-a", 1, 3).reads.length, 2);
});

// --- The final seal binds every fact and freezes the session ----------------

function buildSealedScenario(extraObserve: boolean): { digest: string; reads: number } {
  const limits = roomyLimits();
  const bounds = fullBounds(limits);
  const service = new NativeObserveService(limits);
  const session = service.open("owner-a", "s1");
  const f64 = service.make(session, "owner-a", "primitive_bits", f64Payload(0x80000000, 0));
  const text = service.make(session, "owner-a", "text", { tag: "text", text: "sealed" });
  service.alias(session, "owner-a", f64.handle);
  service.observe(session, "owner-a", [f64.handle, text.handle], "ieee_bits", bounds);
  if (extraObserve) {
    service.observe(session, "owner-a", [text.handle], "lexeme", bounds);
  }
  service.effectfulRead(session, "owner-a", text.handle, "text");
  const seal = service.seal(session, "owner-a");
  return { digest: seal.digest, reads: seal.explicitReads };
}

check("seal-binds-and-freezes", () => {
  const limits = roomyLimits();
  const bounds = fullBounds(limits);
  const service = new NativeObserveService(limits);
  const session = service.open("owner-a", "s1");
  const f64 = service.make(session, "owner-a", "primitive_bits", f64Payload(0x80000000, 0));
  const text = service.make(session, "owner-a", "text", { tag: "text", text: "sealed" });
  service.alias(session, "owner-a", f64.handle);
  service.observe(session, "owner-a", [f64.handle, text.handle], "ieee_bits", bounds);
  service.effectfulRead(session, "owner-a", text.handle, "text");
  const seal = service.seal(session, "owner-a");
  assert.match(seal.digest, /^sha256:[0-9a-f]{64}$/);
  assert.equal(seal.session, "s1");
  assert.equal(seal.cells, 2);
  assert.equal(seal.handles, 3);
  assert.equal(seal.makes, 2);
  assert.equal(seal.aliases, 1);
  assert.equal(seal.observations, 1);
  assert.equal(seal.explicitReads, 1);
  assert.equal(seal.joined, false);
  assert.equal(Object.isFrozen(seal), true);

  // Deterministic: the same scenario in a fresh service seals identically,
  // and any observation difference changes the digest.
  assert.equal(buildSealedScenario(false).digest, seal.digest);
  assert.notEqual(buildSealedScenario(true).digest, seal.digest);

  // Observation history (not just counts) is bound: the same construction
  // with one same-count different-kind observation seals differently.
  function sealWithKind(kind: "ieee_bits" | "lexeme"): string {
    const svc = new NativeObserveService(roomyLimits());
    const sess = svc.open("owner-a", "s1");
    const made = svc.make(sess, "owner-a", "primitive_bits", f64Payload(0x3ff00000, 0));
    svc.observe(sess, "owner-a", [made.handle], kind, fullBounds(roomyLimits()));
    return svc.seal(sess, "owner-a").digest;
  }
  assert.notEqual(sealWithKind("ieee_bits"), sealWithKind("lexeme"));

  // Sealed sessions reject further reads and makes, but stay inspectable.
  expectSchemaError(
    () => service.make(session, "owner-a", "text", { tag: "text", text: "late" }),
    "closed-handle",
  );
  expectSchemaError(() => service.alias(session, "owner-a", text.handle), "closed-handle");
  expectSchemaError(
    () => service.observe(session, "owner-a", [text.handle], "lexeme", bounds),
    "closed-handle",
  );
  expectSchemaError(
    () => service.effectfulRead(session, "owner-a", text.handle, "text"),
    "closed-handle",
  );
  assert.equal(service.counters(session, "owner-a").sealed, true);
  assert.equal(service.interval(session, "owner-a", 0).reads.length, 1);
  const joined = service.seal(session, "owner-a");
  assert.equal(joined.joined, true);
  assert.equal(joined.digest, seal.digest);

  // Close still reclaims a sealed session.
  const receipt = service.close(session, "owner-a");
  assert.equal(receipt.cellsReleased, 3);
  expectSchemaError(() => service.counters(session, "owner-a"), "closed-handle");
});

// --- Fixtures stay armed: they bite when touched, never otherwise -----------

check("fixtures-still-bite", () => {
  // Standalone control: each fixture bites on a direct touch.
  const counters = zeroHostileCounters();
  const getter = createHostileValue("getter_then", counters) as Record<string, unknown>;
  assert.equal(getter["then"], undefined);
  assert.equal(counters.getterReads, 1);
  const throwing = createHostileValue("throwing_thenable", counters) as { then: () => unknown };
  assert.throws(() => throwing.then(), /hostile then called/);
  assert.equal(counters.thenCalls, 1);
  const revoked = createHostileValue("revoked_proxy", counters) as Record<string, unknown>;
  assert.throws(() => revoked["anything"], TypeError);

  // Service control: the same fixtures behind handles stay at zero until the
  // explicit op touches them, and the bite is then counted.
  const limits = roomyLimits();
  const bounds = fullBounds(limits);
  const service = new NativeObserveService(limits);
  const session = service.open("owner-a", "s1");
  const made = service.make(session, "owner-a", "hostile_descriptor", {
    tag: "hostile",
    descriptor: "getter_then",
  });
  service.alias(session, "owner-a", made.handle);
  service.observe(session, "owner-a", [made.handle], "descriptor", bounds);
  service.counters(session, "owner-a");
  service.interval(session, "owner-a", 0);
  const quiet = service.counters(session, "owner-a");
  assert.equal(quiet.getterReads, 0);
  assert.equal(quiet.explicitReads, 0);
  service.effectfulRead(session, "owner-a", made.handle, "property_then");
  const bitten = service.counters(session, "owner-a");
  assert.equal(bitten.getterReads, 1);
  assert.equal(bitten.explicitReads, 1);
});

// --- Bounds, ownership, and closed vocabularies ------------------------------

check("bounds-and-ownership", () => {
  const limits = roomyLimits();
  const bounds = fullBounds(limits);
  const service = new NativeObserveService(limits);
  const session = service.open("owner-a", "s1");
  const text = service.make(session, "owner-a", "text", { tag: "text", text: "hello world" });

  expectSchemaError(
    () => service.observe(session, "owner-a", [text.handle], "raw_value", bounds),
    "unsupported-capability",
  );
  expectSchemaError(
    () => service.observe(session, "owner-a", [], "lexeme", bounds),
    "invalid-request",
  );
  expectSchemaError(
    () => service.observe(session, "owner-a", [text.handle], "lexeme", null),
    "invalid-request",
  );
  expectSchemaError(
    () =>
      service.observe(session, "owner-a", [text.handle], "lexeme", { maxEntries: 0, maxBytes: 8 }),
    "invalid-request",
  );
  expectSchemaError(
    () =>
      service.observe(session, "owner-a", [text.handle], "lexeme", {
        maxEntries: limits.maxObserveEntries + 1,
        maxBytes: limits.maxObserveBytes,
      }),
    "resource-limit",
  );
  expectSchemaError(
    () =>
      service.observe(session, "owner-a", [text.handle], "lexeme", {
        maxEntries: 1,
        maxBytes: 8,
        raw: true,
      }),
    "invalid-request",
  );
  const tooMany = Array.from({ length: limits.maxObserveEntries + 1 }, () => text.handle);
  expectSchemaError(
    () => service.observe(session, "owner-a", tooMany, "lexeme", bounds),
    "resource-limit",
  );
  expectSchemaError(
    () => service.observe(session, "owner-a", [null], "lexeme", bounds),
    "invalid-request",
  );
  expectSchemaError(
    () => service.observe(session, "owner-a", [session], "lexeme", bounds),
    "invalid-request",
  );

  // Tight bounds truncate with explicit gaps instead of silent loss.
  const entries = service.make(session, "owner-a", "ordered_entries", {
    tag: "entries",
    entries: [
      ["a", 1],
      ["b", 2],
      ["c", 3],
      ["d", 4],
    ],
  });
  const capped = service.observe(
    session,
    "owner-a",
    [entries.handle],
    "entries",
    checkObserveBounds({ maxEntries: 2, maxBytes: limits.maxObserveBytes }, limits),
  ).observations[0]?.result as Record<string, unknown>;
  assert.equal((capped["entries"] as unknown[]).length, 2);
  assert.equal(capped["gap"], "entry-cap");
  assert.equal(capped["omitted"], 2);
  const clipped = service.observe(
    session,
    "owner-a",
    [text.handle],
    "lexeme",
    checkObserveBounds({ maxEntries: 8, maxBytes: 5 }, limits),
  ).observations[0]?.result as Record<string, unknown>;
  assert.deepEqual(
    { text: clipped["text"], gap: clipped["gap"], omitted: clipped["omitted"] },
    { text: "hello", gap: "byte-cap", omitted: 6 },
  );

  // Ownership: wrong owners and cross-session handles reject everywhere.
  service.open("owner-b", "s2");
  const foreign = service.open("owner-a", "s3");
  const foreignValue = service.make(foreign, "owner-a", "text", { tag: "text", text: "foreign" });
  expectSchemaError(
    () => service.observe(session, "owner-b", [text.handle], "lexeme", bounds),
    "wrong-owner",
  );
  expectSchemaError(
    () => service.effectfulRead(session, "owner-b", text.handle, "text"),
    "wrong-owner",
  );
  expectSchemaError(() => service.interval(session, "owner-b", 0), "wrong-owner");
  expectSchemaError(() => service.seal(session, "owner-b"), "wrong-owner");
  expectSchemaError(
    () => service.observe(session, "owner-a", [foreignValue.handle], "lexeme", bounds),
    "wrong-owner",
  );
  expectSchemaError(
    () => service.effectfulRead(session, "owner-a", foreignValue.handle, "text"),
    "wrong-owner",
  );

  // Closed vocabularies reject before any effect.
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

  // Closed sessions reject reads and makes.
  service.close(session, "owner-a");
  expectSchemaError(
    () => service.observe(session, "owner-a", [text.handle], "lexeme", bounds),
    "closed-handle",
  );
  expectSchemaError(
    () => service.effectfulRead(session, "owner-a", text.handle, "text"),
    "closed-handle",
  );
  expectSchemaError(() => service.seal(session, "owner-a"), "closed-handle");
});

console.log(
  JSON.stringify({
    kind: "can.native-values-observe-check",
    schema_version: "1",
    checks: passed,
    count: passed.length,
  }),
);
