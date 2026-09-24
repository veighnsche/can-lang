import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createChecks } from "../checks.ts";
import { success, value } from "../completion.ts";
import { runAssertion } from "../assert/runner.ts";
import { withFixture } from "../assert/fixtures.ts";
const identity = (kind: string, declaration: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration]))
    .digest("hex");
const scalar = (name: string): FailureShape => ({
  identity: identity("primitive", name),
  kind: "primitive",
  declaration: name,
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
});
const text = scalar("str"),
  boolean = scalar("bool");
const declarations = catalogue.errors.filter((error) => error.name === "checks::failed");
const errors = declarations.map((error) => ({
  identity: identity("error", error.identity),
  kind: "error",
  declaration: error.identity,
  arguments: [],
  fields: error.fields.map((field) => ({ name: field.name, type: text.identity })),
  leaves: [],
  inputs: [],
  errors: [],
}));
const domain = createDomainRuntime({
  declarations: declarations.map((error) => ({
    identity: error.identity,
    name: error.name,
    parameters: 0,
  })),
  shapes: [text, boolean, ...errors],
});
const api = createChecks(domain, { failed: errors[0].identity });
const origin = Object.freeze({
  source: "checks.test.ts",
  start: 11,
  end: 42,
  invocation: Object.freeze(["can.project.root/checks::guard#0"]),
});

test("true completes void while false produces checks::failed with the exact authored reason", async () => {
  expect(value(await api.require(true, "unused", origin))).toBe(undefined);
  const failed = await api.require(false, "reason text", origin);
  expect(failed.kind).toBe("domain");
  if (failed.kind !== "domain") throw Error("expected domain failure");
  const details = domainFailureDiagnostics(failed.value);
  expect(details.declaration.name).toBe("checks::failed");
  expect(details.declaration.name).toBe("checks::failed");
  expect(details.payload).toMatchObject({ reason: "reason text" });
  expect(details.provenance).toEqual({ boundary: "emitted", operation: "" });
});

test("call-site span and invocation path stay in private occurrence metadata", async () => {
  const failed = await api.require(false, "traced", origin);
  if (failed.kind !== "domain") throw Error("expected domain failure");
  const details = domainFailureDiagnostics(failed.value);
  expect(details.origin).toMatchObject({ source: "checks.test.ts", start: 11, end: 42 });
  expect([...details.origin.invocation]).toEqual(["can.project.root/checks::guard#0"]);
  expect(JSON.stringify(details.payload)).toBe(JSON.stringify({ reason: "traced" }));
});

test("invalid hostile reasons throw during validation without invoking user code", async () => {
  let invoked = 0;
  const trap = () => {
    invoked++;
    return "pwned";
  };
  const hostile = {
    toString: trap,
    valueOf: trap,
    toJSON: trap,
    [Symbol.toPrimitive]: trap,
  } as unknown as string;
  await expect(api.require(false, hostile, origin)).rejects.toThrow(TypeError);
  expect(invoked).toBe(0);
  const frozen = Object.freeze("exact") as string;
  const failed = await api.require(false, frozen, origin);
  if (failed.kind !== "domain") throw Error("expected domain failure");
  expect(domainFailureDiagnostics(failed.value).payload).toMatchObject({ reason: "exact" });
});

test("hostile reason text round-trips exactly through JSON without invoking user code", async () => {
  const reason = '<script>alert(1)</script>&"\n\t\\😀';
  const failed = await api.require(false, reason, origin);
  if (failed.kind !== "domain") throw Error("expected domain failure");
  const payload = domainFailureDiagnostics(failed.value).payload as { reason: string };
  expect(payload.reason).toBe(reason);
  expect(JSON.parse(JSON.stringify(payload))).toEqual({ reason });
});

test("a recovered application check cannot clear a separate harness violation", async () => {
  const root = Object.freeze({
    package: "can.project.root/positive_checks",
    declaration: "can.project.root/positive_checks::require_positive",
    name: "zero",
  });
  const report = await runAssertion({
    root,
    expected: async () => success(undefined),
    actual: async (context) => {
      const checked = await api.require(false, "recovered", origin);
      if (checked.kind !== "domain") return checked;
      try {
        await withFixture(
          context,
          "table",
          [
            {
              selector: "zero",
              owner: "can.project.root/positive_checks",
              arguments: async () => success([1n]),
              expected: async () => success(1n),
            },
          ],
          [2n],
          async () => success(0n),
          origin,
        );
      } catch {}
      return success(undefined);
    },
  });
  expect(report.passed).toBe(false);
  expect(report).toMatchObject({ reason: "harness violation" });
  expect([...report.violations]).toContain("argument mismatch");
});
