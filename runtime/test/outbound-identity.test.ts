import { test, expect } from "bun:test";
import {
  correlationId,
  contextReport,
  invocationContext,
  invocationId,
  isInvocationContext,
  nestContext,
  newCorrelationId,
  newInvocationId,
  poolId,
  tenantId,
} from "../outbound/identity.ts";

test("tenant/pool/correlation/invocation share one validated vocabulary", () => {
  expect(tenantId("acme") as string).toBe("acme");
  expect(poolId("support-triage.v1:default") as string).toBe("support-triage.v1:default");
  expect(correlationId("req-2026-09-26_001") as string).toBe("req-2026-09-26_001");
  expect(invocationId("0193a2f0-bucket-7") as string).toBe("0193a2f0-bucket-7");
  // Max length fits; one char more does not.
  expect(tenantId("a".repeat(128)).length).toBe(128);
  for (const bad of [
    "",
    "a".repeat(129),
    "-lead",
    ".lead",
    "has space",
    "slash/x",
    "at@x",
    "hash#x",
    "ünicode",
    7,
    undefined,
    null,
    {},
  ]) {
    expect(() => tenantId(bad)).toThrow(TypeError);
    expect(() => poolId(bad)).toThrow(TypeError);
    expect(() => correlationId(bad)).toThrow(TypeError);
    expect(() => invocationId(bad)).toThrow(TypeError);
  }
});

test("minted correlation and invocation IDs are unique valid identities", () => {
  const first = newCorrelationId();
  const second = newCorrelationId();
  expect(first).not.toBe(second);
  expect(correlationId(first)).toBe(first);
  const attempt = newInvocationId();
  expect(invocationId(attempt)).toBe(attempt);
  expect(invocationId(newInvocationId())).not.toBe(attempt);
});

test("invocation contexts freeze trusted tenant/pool/correlation", () => {
  const context = invocationContext("acme", "default", "req-1");
  expect(context.tenant as string).toBe("acme");
  expect(context.pool as string).toBe("default");
  expect(context.correlation as string).toBe("req-1");
  expect(Object.isFrozen(context)).toBe(true);
  expect(isInvocationContext(context)).toBe(true);
  // Omitted correlation is minted, never empty.
  const minted = invocationContext("acme", "default");
  expect(correlationId(minted.correlation)).toBe(minted.correlation);
  // Browser-shaped claims with no trusted tenant never build a context.
  expect(() => invocationContext("", "default", "req-1")).toThrow(TypeError);
  expect(() => invocationContext("acme", "", "req-1")).toThrow(TypeError);
  expect(isInvocationContext(null)).toBe(false);
  expect(isInvocationContext({ tenant: "acme", pool: "default" })).toBe(false);
  expect(isInvocationContext({ tenant: "", pool: "default", correlation: "r" })).toBe(false);
});

test("nested scopes keep the active tenant/pool and reject replacement", () => {
  const active = invocationContext("acme", "default", "req-1");
  const nested = invocationContext("acme", "default", "req-2");
  // Same-context nesting returns the active record unchanged: the outer
  // correlation stays authoritative and nothing leaks across scopes.
  expect(nestContext(active, nested)).toBe(active);
  expect(() => nestContext(active, invocationContext("other", "default", "req-1"))).toThrow(
    TypeError,
  );
  expect(() => nestContext(active, invocationContext("acme", "other", "req-1"))).toThrow(TypeError);
});

test("context reports carry identity only, never secrets", () => {
  const context = invocationContext("acme", "default", "req-1");
  const report = contextReport(context);
  expect(report).toEqual({ tenant: "acme", pool: "default", correlation: "req-1" });
  expect(Object.isFrozen(report)).toBe(true);
  expect(Object.keys(report).sort()).toEqual(["correlation", "pool", "tenant"]);
});
