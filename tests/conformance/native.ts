import { types } from "node:util";
import { readFileSync, realpathSync } from "node:fs";
import { createHash } from "node:crypto";
import { resolve } from "node:path";
import { strict as assert } from "node:assert";

export const sha256 = (data: string | Uint8Array) => createHash("sha256").update(data).digest("hex");
export function identityFailures(target: any, actual: any): string[] {
  return ["name", "version", "revision", "platform", "architecture", "sha256"]
    .filter(key => target[key] !== actual[key]).map(key => `runtime.${key}: expected ${target[key]}, got ${actual[key]}`);
}
export function apiAvailable(name: string): boolean {
  if (name === "node:util.types.isProxy") return typeof types.isProxy === "function";
  return typeof name.split(".").reduce((value, key) => value?.[key], globalThis as any) === "function";
}
const probes: Record<string, () => unknown> = {
  "json-source-and-raw-integers": () => {
    const tokens: string[] = [];
    const value = JSON.parse('{"large":9007199254740993,"zero":-0}', (key, value, context) => {
      if (key) tokens.push(context.source);
      return value;
    });
    assert.deepEqual(tokens, ["9007199254740993", "-0"]);
    assert(Object.is(value.zero, -0));
    assert.equal(JSON.stringify({ large: JSON.rawJSON(tokens[0]) }), '{"large":9007199254740993}');
  },
  "ordered-fromAsync": async () => {
    const events: string[] = [];
    const result = await Array.fromAsync([0, 1, 2], async i => {
      events.push(`start${i}`); await Promise.resolve(); events.push(`end${i}`); return i;
    });
    assert.deepEqual(result, [0, 1, 2]);
    assert.deepEqual(events, ["start0", "end0", "start1", "end1", "start2", "end2"]);
  },
  "nonthenable-box": async () => {
    let calls = 0;
    const value = { then() { calls++; } };
    const box = Object.assign(Object.create(null), { value });
    assert.equal((await Promise.resolve(box)).value, value);
    assert.equal((await Array.fromAsync([0], async () => box))[0].value, value);
    assert.equal(calls, 0);
  },
  "strict-equality": () => {
    assert(Bun.deepEquals({ x: NaN }, { x: NaN }, true));
    assert(!Bun.deepEquals({ x: 0 }, { x: -0 }, true));
  },
  "immutable-arrays": () => {
    const input = [3, 1, 2];
    assert.deepEqual(input.toSorted(), [1, 2, 3]);
    assert.deepEqual(input.toReversed(), [2, 1, 3]);
    assert.deepEqual(input, [3, 1, 2]);
  },
  "unicode-and-literal-replacement": () => {
    assert.equal("😀".length, 2); assert.equal([..."😀"].length, 1);
    assert.equal("e\u0301".normalize("NFC"), "é");
    assert.equal([...new Intl.Segmenter("en", { granularity: "grapheme" }).segment("e\u0301")].length, 1);
    assert.equal("aa".replaceAll("a", () => "$&"), "$&$&");
  },
  "fatal-utf8": () => assert.throws(() => new TextDecoder("utf-8", { fatal: true }).decode(new Uint8Array([255]))),
  "native-intersection-order": () => assert.deepEqual([...new Set([3, 2, 1]).intersection(new Set([1, 2]))], [1, 2]),
  "bun-io-handles": () => {
    assert(Array.isArray(Bun.argv)); assert.equal(typeof Bun.env, "object");
    for (const handle of [Bun.stdin, Bun.stdout, Bun.stderr]) assert.equal(typeof handle.stream, "function");
  },
};

export async function qualify(manifest: any, actual: any, available = apiAvailable) {
  const failures = identityFailures(manifest.runtime, actual);
  const capabilities: Record<string, boolean> = {};
  for (const name of manifest.requiredNativeAPIs) {
    capabilities[name] = available(name);
    if (!capabilities[name]) failures.push(`missing native API: ${name}`);
  }
  const behaviors: Record<string, boolean> = {};
  for (const name of manifest.requiredBehaviorProbes) {
    try {
      assert.equal(typeof probes[name], "function", `unknown probe: ${name}`);
      await probes[name](); behaviors[name] = true;
    } catch (error) { behaviors[name] = false; failures.push(`${name}: ${String(error)}`); }
  }
  return { schemaVersion: 1, kind: "can.native-capability-report", targetId: manifest.targetId,
    actual, capabilities, behaviors, failures, passed: failures.length === 0 };
}

if (import.meta.main) {
  const [root, osVersion] = process.argv.slice(2);
  const manifestBytes = readFileSync(resolve(root, "distribution/target.json"));
  const manifest = JSON.parse(manifestBytes.toString());
  const actual = { name: "bun", version: Bun.version, revision: Bun.revision,
    platform: process.platform, architecture: process.arch, osVersion,
    executable: realpathSync(process.execPath), sha256: sha256(readFileSync(process.execPath)),
    versions: process.versions };
  const report = await qualify(manifest, actual);
  if (actual.executable !== realpathSync(resolve(root, manifest.runtime.executable))) report.failures.push("runtime executable is outside the staged layout");
  const parts = (s: string) => s.split(".").map(Number);
  const [major, minor = 0] = parts(osVersion ?? "0");
  const [minMajor, minMinor = 0] = parts(manifest.runtime.minimumOSVersion);
  if (major < minMajor || (major === minMajor && minor < minMinor)) report.failures.push("unsupported OS version");
  report.passed = report.failures.length === 0;
  console.log(JSON.stringify({ ...report, manifestSHA256: sha256(manifestBytes),
    suiteSHA256: sha256(readFileSync(import.meta.path)), generatedAt: new Date().toISOString() }, null, 2));
  process.exitCode = report.passed ? 0 : 1;
}
