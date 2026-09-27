// Copied into the prepared family work directory; imports share emitted runtime
// instances with the Can workload, preserving private brands and identities.
import assert from "node:assert/strict";
import {array, record, update, recordIdentity, registerOpaqueContents} from "./generated/runtime/data.ts";
import {success, value, invoke} from "./generated/runtime/completion.ts";
import * as arrays from "./generated/runtime/collections/array.ts";
import {createMap} from "./generated/runtime/collections/map.ts";
import {createDomainRuntime} from "./generated/runtime/domain.ts";
import {ownBytes, copyBytes, createBytes} from "./generated/runtime/bytes.ts";
import {decodeJSON, encodeJSON} from "./generated/runtime/codec/json.ts";
import {CodecIssue} from "./generated/runtime/codec/budget.ts";
import {initialize, doubled, frequency} from "./generated/generated-adapter.ts";
import * as emitted from "./generated/generated-adapter.ts";
import {ownCallable} from "./generated/runtime/callable.ts";
import {runOwnedRoot, registerResource, useResource, closeResource} from "./generated/runtime/owner.ts";

const [suite, profile, iterationText, warmupText, sizeText, mode] = process.argv.slice(2);
const iterations = Number(iterationText), warmups = Number(warmupText), size = Number(sizeText);
assert(Number.isSafeInteger(size) && size > 0);
assert(Number.isSafeInteger(iterations) && iterations > 0);
const origin = {source: "performance", start: 0, end: 0, invocation: []};
const trace = {site: "performance-map", origin};
const input = array(Array.from({length: size}, (_, i) => BigInt(i)));
const inputSnapshot = [...input];
const expected = input.map(x => x * 2n);
const maps = createMap<string, bigint>(createDomainRuntime({declarations: [], shapes: []}),
  {map: "perf::map", entry: "perf::entry", absent: "perf::absent", exists: "perf::exists"}, "str");
const cases: {name: string; run: () => unknown; check: (result: any) => unknown;
  checks: string[]; contract: string; bytes?: number}[] = [];
const add = (name: string, run: () => unknown, check: (result: any) => unknown,
  contract: string, checks: string[], bytes?: number) => cases.push({name, run, check, contract, checks, bytes});
const arrayCheck = (result: any) => {
  const output = value(result);
  assert.deepEqual(output, expected); assert(Object.isFrozen(output));
  assert.deepEqual(input, inputSnapshot); assert(Object.isFrozen(input));
};

if (suite === "generated") {
  await initialize();
  add("generated.doubled.can", () => doubled(input), arrayCheck,
    "bigint-array-map-frozen-success", ["values", "frozen-result", "input-unchanged"]);
  add("generated.doubled.native", async () => success(Object.freeze(input.map(x => x * 2n))), arrayCheck,
    "bigint-array-map-frozen-success", ["values", "frozen-result", "input-unchanged"]);
  const unique = Math.min(size, 10);
  const words = array(Array.from({length: size}, (_, i) => "w" + (i % unique)));
  const saved = [...words];
  const check = (result: any) => {assert.equal(value(result), BigInt(Math.ceil(size / unique))); assert.deepEqual(words, saved);};
  add("generated.frequency.can", () => frequency(words, "w0"), check,
    "scalar-string-frequency-immutable-map-fold-success", ["frequency", "input-unchanged"]);
  add("generated.frequency.native", async () => {
    let counts = new Map<string, bigint>();
    for (const word of words) counts = new Map(counts).set(word, (counts.get(word) ?? 0n) + 1n);
    return success(counts.get("w0"));
  }, check, "scalar-string-frequency-immutable-map-fold-success", ["frequency", "input-unchanged"]);
  // Scalar-entrypoint scenarios use an identical host batch on both sides.
  // Each operation below is one size-element workload, not one element.
  const mixed = array(Array.from({length: size}, (_, i) => i % 3 === 0 ? -BigInt(i + 1) : BigInt(i)));
  const batch = async (items: readonly any[], operation: (item: any) => any) => {
    const outputs = [];
    for (const item of items) outputs.push(value(await operation(item)));
    return success(array(outputs));
  };
  const expectedArray = (expected: readonly unknown[], sources: readonly unknown[] = input) => (result: any) => {
    assert.deepEqual(value(result), expected); assert(Object.isFrozen(value(result)));
    assert(Object.isFrozen(sources)); assert.deepEqual(input, inputSnapshot);
  };
  const arithmetic = (number: bigint) => number < 0n ? number * number + 3n : (number + 3n) * number + 1n;
  for (const number of [-4n, 0n, 4n]) assert.equal(value(await emitted.branch_arithmetic(number)), arithmetic(number));
  assert.equal(value(await emitted.sum(array([]))), 0n);
  for (const [name, operation] of [["can", emitted.branch_arithmetic], ["native", async (number: bigint) => success(arithmetic(number))]] as const)
    add("generated.branch-arithmetic." + name, () => batch(mixed, operation), expectedArray(mixed.map(arithmetic), mixed),
      "bigint-mixed-sign-branches-frozen-result-batch", ["all-values", "both-branches", "frozen-result", "input-unchanged"]);
  const total = input.reduce((acc, number) => acc + number, 0n);
  for (const [name, operation] of [["can", emitted.sum], ["native", async (values: readonly bigint[]) => success(values.reduce((acc, number) => acc + number, 0n))]] as const)
    add("generated.fold-sum." + name, () => operation(input), result => {assert.equal(value(result), total); assert.deepEqual(input, inputSnapshot);},
      "bigint-array-reduction-zero-seed", ["sum", "input-unchanged"]);
  const counters = [];
  for (const number of input) counters.push(value(await emitted.make_counter(number)));
  const counterInput = array(counters), counterSnapshot = counters.map(item => [item.value, item.label]);
  const checkCounter = (result: any) => {
    const outputs = value(result);
    assert.deepEqual(outputs.map((item: any) => [item.value, item.label]), input.map(number => [number + 7n, "stable"]));
    assert(Object.isFrozen(outputs));
    for (let i = 0; i < size; i++) {assert(Object.isFrozen(outputs[i])); assert.equal(recordIdentity(outputs[i]), recordIdentity(counterInput[i])); assert.notEqual(outputs[i], counterInput[i]);}
    assert.deepEqual(counterInput.map(item => [item.value, item.label]), counterSnapshot);
  };
  add("generated.record-update.can", () => batch(counterInput, item => emitted.update_counter(item, 7n)), checkCounter,
    "nominal-record-field-update-preserve-label-and-prior-snapshot", ["all-fields", "nominal-identity", "new-frozen-record", "old-version-unchanged"]);
  add("generated.record-update.native-adapter", () => batch(counterInput, async item => success(update(item, [["value", item.value + 7n]]))), checkCounter,
    "nominal-record-field-update-preserve-label-and-prior-snapshot", ["all-fields", "nominal-identity", "new-frozen-record", "old-version-unchanged"]);
  for (const [name, operation] of [["can", emitted.project_counter], ["native", async (item: any) => success(item.value * 2n + BigInt(item.label.length))]] as const)
    add("generated.record-projection." + name, () => batch(counterInput, operation), expectedArray(input.map(number => number * 2n + 6n), counterInput),
      "record-bigint-and-string-field-projection-batch", ["projected-values", "frozen-result", "input-unchanged"]);
  const shapes = [];
  for (const number of input) shapes.push(value(await emitted.make_shape(number)));
  const shapeInput = array(shapes), circleIdentity = recordIdentity(value(await emitted.make_shape(2n)));
  assert.equal(value(await emitted.shape_area(value(await emitted.make_shape(2n)))), 4n);
  assert.equal(value(await emitted.shape_area(value(await emitted.make_shape(3n)))), 12n);
  assert.equal(value(await emitted.is_positive(0n)), false);
  assert.equal(value(await emitted.is_positive(3n)), true);
  const shapeExpected = input.map(number => number % 2n === 0n ? number * number : number * (number + 1n));
  for (const [name, operation] of [["can", emitted.shape_area], ["native", async (item: any) => success(recordIdentity(item) === circleIdentity ? item.radius * item.radius : item.width * item.height)]] as const)
    add("generated.tagged-variant." + name, () => batch(shapeInput, operation), expectedArray(shapeExpected, shapeInput),
      "nominal-circle-or-rectangle-variant-dispatch", ["both-variants", "all-values", "frozen-result", "input-unchanged"]);
  for (const [name, operation] of [["can", emitted.is_positive], ["native", async (number: bigint) => success(number > 0n)]] as const)
    add("generated.failure-recovery." + name, () => batch(mixed, operation), expectedArray(mixed.map(number => number > 0n), mixed),
      "named-require-failure-recovered-to-boolean-scalar-inputs", ["recovered-failure", "successful-require", "all-values", "frozen-result"]);
  for (const [name, operation] of [["can", emitted.generic_pass], ["native", async (values: readonly bigint[]) => success(Object.freeze(values.map(number => number)))]] as const)
    add("generated.generic-map." + name, () => operation(input), expectedArray(input),
      "int-instantiation-generic-identity-callback-array-map", ["values", "frozen-result", "input-unchanged"]);
  for (const [name, operation] of [["can", emitted.captured_map], ["native", async (values: readonly bigint[], offset: bigint) => success(Object.freeze(values.map(number => number + offset)))]] as const)
    add("generated.captured-map." + name, () => operation(input, 17n), expectedArray(input.map(number => number + 17n)),
      "scalar-offset-capture-created-per-map-call", ["captured-offset", "all-values", "frozen-result", "input-unchanged"]);
  assert.deepEqual(value(await emitted.captured_map(array([1n, 2n]), -3n)), [-2n, -1n]);
  const unicode = "  E\u0301 😀 HéLLo  ".repeat(size);
  const normalized = unicode.trim().toLowerCase().normalize("NFC");
  const scalarValid = (text: string) => {
    if (!text.isWellFormed()) throw Error("invalid Unicode scalar");
    return new TextEncoder().encode(text);
  };
  for (const [name, operation] of [["can", emitted.normalize_text], ["native", async (text: string) => {const lowered = text.trim().toLowerCase(); if (!lowered.isWellFormed()) throw Error("invalid Unicode scalar"); return success(lowered.normalize("NFC"));}]] as const)
    add("generated.unicode-normalize." + name, () => operation(unicode), result => assert.equal(value(result), normalized),
      "valid-unicode-trim-lowercase-nfc", ["unicode-values", "decomposed-accent", "non-bmp-text"]);
  assert.throws(() => scalarValid("\ud800"), /invalid Unicode scalar/);
  const encoded = Buffer.from(unicode, "utf8").toString("base64");
  for (const [name, operation] of [["can", emitted.base64_text], ["native", async (text: string) => success(Buffer.from(scalarValid(text)).toString("base64"))]] as const)
    add("generated.utf8-base64." + name, () => operation(unicode), result => assert.equal(value(result), encoded),
      "valid-unicode-to-utf8-standard-base64-string", ["exact-base64", "non-ascii-text"]);
  assert.equal((await emitted.normalize_text("\ud800")).kind, "domain");
  assert.equal((await emitted.base64_text("\ud800")).kind, "domain");
} else if (suite === "runtime") {
  add("runtime.array.map", () => arrays.map(input, x => success(x * 2n), trace), arrayCheck,
    "runtime-completion-callback-array-map", ["values", "frozen-result", "input-unchanged"]);
  add("runtime.array.map.native", async () => success(Object.freeze(input.map(x => x * 2n))), arrayCheck,
    "scalar-array-result-contract-native", ["values", "frozen-result", "input-unchanged"]);
  const entries = array(Array.from({length: size}, (_, i) => Object.freeze({key: "k" + i, value: BigInt(i)})));
  const checkMap = async (result: any) => {
    const map = value(result);
    assert(Object.isFrozen(map));
    const contents = value(await maps.entries(map));
    assert.equal(contents.length, size); assert(Object.isFrozen(contents));
    assert.deepEqual(contents.map(entry => [entry.key, entry.value]), entries.map(entry => [entry.key, entry.value]));
    assert.equal(value(await maps.get(map, "k" + (size - 1))), BigInt(size - 1));
    assert.equal(entries.length, size); assert.equal(entries[0].value, 0n);
  };
  add("runtime.map.bulk-build", () => maps.build_map(entries), checkMap,
    "scalar-immutable-map-bulk-build", ["entries", "get", "frozen-handle", "input-unchanged"]);
  // Point updates are a separate workload: they preserve every prior snapshot.
  const empty = value(await maps.empty());
  add("runtime.map.insert-chain", async () => {
    let current = empty;
    for (const entry of entries) current = value(await maps.insert(current, entry.key, entry.value));
    return success(current);
  }, async result => {await checkMap(result); assert.equal(value(await maps.entries(empty)).length, 0);},
  "scalar-immutable-map-point-updates", ["entries", "get", "old-version-unchanged"]);
  const backing = new WeakMap<object, Map<string, bigint>>();
  const publish = (contents: Map<string, bigint>) => {
    const token = Object.freeze(Object.create(null));
    backing.set(token, contents); registerOpaqueContents(token, Array.from(contents.values())); return token;
  };
  add("runtime.map.insert-chain.native", async () => {
    const first = publish(new Map()); let current = first;
    for (const entry of entries) {
      const old = backing.get(current)!;
      if (old.has(entry.key)) throw Error("duplicate key");
      current = publish(new Map(old).set(entry.key, entry.value));
    }
    return success({first, current});
  }, result => {
    const {first, current} = value(result);
    assert.equal(backing.get(first)!.size, 0); assert.equal(backing.get(current)!.size, size);
    assert.deepEqual([...backing.get(current)!], entries.map(entry => [entry.key, entry.value]));
    assert.equal(backing.get(current)!.get("k" + (size - 1)), BigInt(size - 1));
    assert(Object.isFrozen(current));
  }, "scalar-immutable-private-map-point-updates-with-ownership-metadata", ["entries", "get", "old-version-unchanged", "frozen-handle"]);
  const first = value(await maps.insert(empty, "snapshot", 1n));
  const second = value(await maps.replace(first, "snapshot", 2n));
  assert.equal(value(await maps.get(first, "snapshot")), 1n);
  assert.equal(value(await maps.get(second, "snapshot")), 2n);
  const recordSource = record("perf::counter", [["value", 0n], ["label", "stable"]]);
  add("runtime.record.update-chain", () => {
    let current = recordSource;
    for (let i = 0; i < size; i++) current = update(current, [["value", current.value as bigint + 1n]]);
    return current;
  }, current => {assert.equal(current.value, BigInt(size)); assert.equal(current.label, "stable"); assert.equal(recordIdentity(current), "perf::counter"); assert(Object.isFrozen(current)); assert.equal(recordSource.value, 0n);},
    "nominal-frozen-record-point-updates", ["value", "preserved-field", "identity", "old-version-unchanged"]);
  add("runtime.callable.capture-and-map", () => {
    const offset = 17n;
    const action = ownCallable("performance::captured#0", "performance::add", [offset], (number: bigint) => success(number + offset), []);
    return arrays.map(input, action, trace);
  }, result => {assert.deepEqual(value(result), input.map(number => number + 17n)); assert(Object.isFrozen(value(result))); assert.deepEqual(input, inputSnapshot);},
    "owned-scalar-capture-created-and-invoked-by-runtime-map", ["capture-value", "all-values", "frozen-result", "input-unchanged"]);
  add("runtime.completion.invoke-chain", async () => {
    let completion = success(0n);
    for (const number of input) completion = await invoke(() => success(value(completion) + number), origin);
    return completion;
  }, result => {assert.equal(value(result), input.reduce((sum, number) => sum + number, 0n)); assert(Object.isFrozen(result));},
    "completion-success-invocation-chain", ["sum", "branded-success", "frozen-completion"]);
  add("runtime.completion.capture-thrown-error", async () => {
    const results = [];
    for (let i = 0; i < size; i++) results.push(await invoke(() => {throw new TypeError("performance fixture");}, origin));
    return results;
  }, results => {assert.equal(results.length, size); for (const result of results) {assert.equal(result.kind, "standard"); assert(Object.isFrozen(result));}},
    "thrown-typeerror-becomes-standard-completion", ["all-errors-captured", "standard-kind", "frozen-completions"]);
  add("runtime.owner.resource-lifecycle", async () => {
    let closes = 0, sum = 0n;
    const result = await runOwnedRoot(async () => {
      for (const number of input) {
        const handle = registerResource("performance", number, () => {closes++; return success(undefined);});
        sum += value(await useResource(handle, "performance", item => success(item)));
        await closeResource(handle, "performance");
      }
      return success({sum, closes});
    });
    return result.completion;
  }, result => {assert.deepEqual(value(result), {sum: input.reduce((sum, number) => sum + number, 0n), closes: size});},
    "owned-root-register-use-explicit-close", ["all-values-used", "exactly-one-close-per-resource"]);
  const utf8Text = "héllo 😀\n".repeat(size), utf8Bytes = new TextEncoder().encode(utf8Text), byteInput = ownBytes(utf8Bytes);
  const byteControls = createBytes(createDomainRuntime({declarations: [], shapes: []}), "perf::invalid-data");
  add("runtime.bytes.utf8-roundtrip", async () => byteControls.toUTF8(value(await byteControls.fromUTF8(utf8Text))),
    result => assert.equal(value(result), utf8Text), "valid-unicode-private-bytes-roundtrip", ["exact-text", "multibyte-utf8"], utf8Bytes.length);
  add("runtime.bytes.boundary-copy", () => copyBytes(byteInput, origin), result => {
    assert.deepEqual(result, utf8Bytes); result[0] = 0;
    assert.deepEqual(copyBytes(byteInput, origin), utf8Bytes); assert.equal(utf8Bytes[0], 104);
  }, "opaque-bytes-expose-independent-native-copy", ["all-bytes", "returned-copy-mutation-isolated", "source-unchanged"], utf8Bytes.length);
} else if (suite === "codecs") {
  const schema = {root: "items", nodes: [
    {identity: "int", kind: "primitive", name: "int"},
    {identity: "items", kind: "array", name: "int[]", element: "int"}
  ]} as const;
  const text = JSON.stringify(input.map(Number));
  const payload = ownBytes(new TextEncoder().encode(text));
  const rejectText = (text: string, reason: string) => assert.throws(
    () => decodeJSON(schema, ownBytes(new TextEncoder().encode(text))),
    error => error instanceof CodecIssue && error.reason === reason);
  rejectText('["wrong"]', "type");
  rejectText('[1,]', "invalid_json");
  rejectText('{"x":1,"x":2}', "duplicate_member");
  assert.throws(() => decodeJSON(schema, ownBytes(new Uint8Array([0xff]))),
    error => error instanceof CodecIssue && error.reason === "utf8");
  // Production JSON projects exact integer tokens, including values above 2^53.
  assert.deepEqual(decodeJSON(schema, ownBytes(new TextEncoder().encode('[9007199254740993]'))), [9007199254740993n]);
  assert.equal(new TextDecoder().decode(copyBytes(encodeJSON(schema, array([9007199254740993n])), origin)), "[9007199254740993]");
  const checks = ["values", "frozen-array", "input-unchanged", "reject-type", "reject-malformed-json", "reject-duplicates", "reject-utf8", "exact-large-integer"];
  add("codecs.json.decode-contract", () => decodeJSON(schema, payload), result => {
    assert.deepEqual(result, input); assert(Object.isFrozen(result)); assert.deepEqual(input, inputSnapshot);
    assert.equal(new TextDecoder().decode(copyBytes(payload, origin)), text);
  }, "production-json-bigint-array-schema", checks, new TextEncoder().encode(text).length);
  add("codecs.json.encode-contract", () => encodeJSON(schema, input), result => {
    assert.equal(new TextDecoder().decode(copyBytes(result, origin)), text); assert.deepEqual(input, inputSnapshot);
  }, "production-json-bigint-array-schema", ["exact-json-output", "input-unchanged", "exact-large-integer"], new TextEncoder().encode(text).length);
  add("codecs.json.native-parse-component", () => JSON.parse(text), result => assert.deepEqual(result, input.map(Number)),
    "unchecked-native-parser-component-not-contract-equivalent", ["component-values"], new TextEncoder().encode(text).length);
  const nestedSchema = {root: "rows", nodes: [
    {identity: "int", kind: "primitive", name: "int"},
    {identity: "str", kind: "primitive", name: "str"},
    {identity: "bool", kind: "primitive", name: "bool"},
    {identity: "tags", kind: "array", name: "str[]", element: "str"},
    {identity: "perf::meta", kind: "record", name: "meta", fields: [{name: "active", type: "bool"}, {name: "tags", type: "tags"}]},
    {identity: "perf::row", kind: "record", name: "row", fields: [{name: "id", type: "int"}, {name: "title", type: "str"}, {name: "meta", type: "perf::meta"}]},
    {identity: "rows", kind: "array", name: "row[]", element: "perf::row"}
  ]} as const;
  const nativeRows = Array.from({length: size}, (_, i) => ({id: i, title: `Row ${i}: é😀 \"quoted\"\n`, meta: {active: i % 2 === 0, tags: ["alpha", "E\u0301", ""]}}));
  const nestedInput = array(nativeRows.map(row => record("perf::row", [["id", BigInt(row.id)], ["title", row.title],
    ["meta", record("perf::meta", [["active", row.meta.active], ["tags", array([...row.meta.tags])]])]])));
  const nestedText = JSON.stringify(nativeRows), nestedBytes = new TextEncoder().encode(nestedText), nestedPayload = ownBytes(nestedBytes);
  add("codecs.json.nested-records.decode-contract", () => decodeJSON(nestedSchema, nestedPayload), result => {
    assert.deepEqual(result, nestedInput); assert(Object.isFrozen(result));
    for (const row of result) {assert(Object.isFrozen(row)); assert(Object.isFrozen(row.meta)); assert(Object.isFrozen(row.meta.tags)); assert.equal(recordIdentity(row), "perf::row");}
    assert.deepEqual(copyBytes(nestedPayload, origin), nestedBytes);
  }, "nested-nominal-record-array-bigint-unicode-bool-tags", ["all-fields", "nominal-identities", "nested-frozen-values", "utf8-payload-unchanged"], nestedBytes.length);
  add("codecs.json.nested-records.encode-contract", () => encodeJSON(nestedSchema, nestedInput), result => {
    assert.equal(new TextDecoder().decode(copyBytes(result, origin)), nestedText);
    assert.deepEqual(decodeJSON(nestedSchema, result), nestedInput);
  }, "nested-nominal-record-array-bigint-unicode-bool-tags", ["exact-json", "roundtrip-record-identities", "escaped-unicode-text"], nestedBytes.length);
  add("codecs.json.nested-records.native-parse-component", () => JSON.parse(nestedText), result => assert.deepEqual(result, nativeRows),
    "unchecked-native-parser-component-not-contract-equivalent", ["all-component-fields"], nestedBytes.length);
  const rejection = (name: string, payload: Uint8Array, reason: string, path?: string) => {
    const owned = ownBytes(payload), snapshot = payload.slice();
    add("codecs.json.reject." + name, () => {
      try {decodeJSON(nestedSchema, owned);} catch (error) {return error;}
      throw Error("invalid fixture was accepted");
    }, error => {assert(error instanceof CodecIssue); assert.equal(error.reason, reason); if (path !== undefined) assert.equal(error.path, path); assert.deepEqual(copyBytes(owned, origin), snapshot);},
      "production-nested-schema-rejection-" + reason, ["rejected", "exact-reason", ...(path !== undefined ? ["exact-path"] : []), "payload-unchanged"], payload.length);
  };
  const wrongType = nativeRows.map((row, i) => i === size - 1 ? {...row, id: "wrong"} : row);
  rejection("type-at-last-row", new TextEncoder().encode(JSON.stringify(wrongType)), "type", `/${size - 1}/id`);
  const missing = nativeRows.map((row, i) => i === size - 1 ? {id: row.id, title: row.title} : row);
  rejection("missing-member-at-last-row", new TextEncoder().encode(JSON.stringify(missing)), "missing_member", `/${size - 1}/meta`);
  rejection("duplicate-member", new TextEncoder().encode(nestedText.replace('"id":0', '"id":0,"id":1')), "duplicate_member");
  const invalidUtf8 = nestedBytes.slice(); invalidUtf8[invalidUtf8.length - 1] = 0xff;
  rejection("invalid-utf8", invalidUtf8, "utf8");
} else {
  throw Error("unknown suite " + suite);
}
assert(cases.length > 0);
const results = [];
for (const test of cases) {
  await test.check(await test.run());
  const samples: number[] = [], warmupSamples: number[] = [];
  let sink: unknown;
  if (mode !== "validate") {
    for (let batch = 0; batch < warmups + (profile === "quick" ? 3 : 7); batch++) {
      const start = performance.now();
      for (let i = 0; i < iterations; i++) sink = await test.run();
      const elapsed = (performance.now() - start) * 1e6 / iterations;
      (batch < warmups ? warmupSamples : samples).push(elapsed);
      await test.check(sink);
    }
  }
  results.push({name: test.name, unit: "ns/op", samples, warmup_samples: warmupSamples,
    iterations_per_sample: iterations, timing_scope: "Wall time of repeated workload calls including await/completion and result publication; fixtures, initialization and correctness checks excluded",
    parameters: {size, contract: test.contract, ...(test.name.includes("frequency") ? {unique_words: Math.min(size, 10)} : {})}, correctness: {passed: true, checks: test.checks},
    metrics: {allocation_bytes: "unavailable", peak_rss: "unavailable",
      ...(test.bytes ? {payload_bytes: test.bytes, throughput_basis: "supplied input payload bytes; rejection does not imply full schema traversal", bytes_per_second: samples.map(ns => test.bytes! * 1e9 / ns)} : {})}});
}
console.log(JSON.stringify({schema_version: 1, suite, status: "complete", cases: results,
  notes: ["Representative language and runtime workload coverage; no allocation or peak-memory estimate is inferred from heap deltas.",
    "Native references preserve each named tested result contract. Generic/captured callbacks use scalar values; resource capture and arbitrary callback failures are outside these comparisons.",
    "Recovered-failure native references test the final boolean endpoint and deliberately avoid creating an internal domain failure; record-update references explicitly include the nominal runtime adapter.",
    "Generated Unicode/bytes native references cover valid scalar strings; emitted surrogate rejection is separately checked outside timing. Scalar-entrypoint batches include the same host loop/await on both sides, and ns/op means one size-element workload.",
    "Native JSON.parse is a parser component only and does not implement Can codec contracts."], artifacts: []}));
