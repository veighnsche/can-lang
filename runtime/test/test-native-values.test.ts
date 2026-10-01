// NT-I01 native-values contract: the runtime catalogue mirror carries the
// native package with 11 operations over 5 opaque handles, constructible
// records using TS wire keys verbatim, and merge-defined contracts for the
// service methods that do not exist yet. Markers native:: and
// can.std.native@1 keep the NT-I01 catalogue package referenced from
// runtime test evidence. No $canNative behavior exists yet; this file pins
// the mirror shape only.
import { test, expect } from "bun:test";
import { catalogue } from "../catalogue.ts";

const NATIVE_OPS = [
  "native::open",
  "native::describe",
  "native::make",
  "native::invoke",
  "native::settle",
  "native::observe",
  "native::allocate_gate",
  "native::release",
  "native::install_fault",
  "native::restore",
  "native::close",
] as const;

test("native package mirror carries the NT-I01 operation surface", () => {
  const pkg = catalogue.packages.find((entry) => entry.name === "native");
  expect(pkg?.identity).toBe("can.std.native@1");
  const ops = catalogue.operations.filter((op) => op.name.startsWith("native::"));
  expect(ops.map((op) => op.name).sort()).toEqual([...NATIVE_OPS].sort());
  for (const op of ops) {
    expect(op.assertion).toBe("supplied");
    expect(op.lowering.task).toBe("NT-I01");
    expect(op.lowering.native.length).toBeGreaterThan(0);
    expect(op.emits).toEqual([]);
  }
});

test("native limits mirror uses TS wire keys verbatim", () => {
  const limits = catalogue.types.find((entry) => entry.name === "native::limits");
  const fields = Object.fromEntries(
    (limits?.fields ?? []).map((field) => [field.name, field.type]),
  );
  expect(fields).toEqual({
    max_sessions: "int",
    max_handles_per_session: "int",
    maxPendingActions: "int",
    maxObserveEntries: "int",
    maxObserveBytes: "int",
    maxGatesPerSession: "int",
    maxFaultsPerSession: "int",
  });
});

test("native handles are opaque and the literal value is a variant", () => {
  for (const name of [
    "native::session",
    "native::value_handle",
    "native::pending_action",
    "native::gate",
    "native::fault",
  ]) {
    const handle = catalogue.types.find((entry) => entry.name === name);
    expect(handle?.kind).toBe("opaque");
    expect(handle?.constructible).toBe(false);
  }
  const literal = catalogue.types.find((entry) => entry.name === "native::literal_value");
  expect(literal?.kind).toBe("variant");
  expect(literal?.leaves).toEqual([
    "native::literal_int",
    "native::literal_text",
    "native::literal_bool",
  ]);
});
