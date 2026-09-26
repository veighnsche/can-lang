import { test, expect } from "bun:test";
import {
  credentialValue,
  destinationPolicy,
  envName,
  evaluateDestination,
  evaluateRedirect,
  isPrivateAddressLiteral,
  policyReport,
  redactUrl,
} from "../outbound/destination-policy.ts";
import fixtures from "../outbound/fixtures/destination-fixtures.json";
import companionPolicyFile from "../outbound/fixtures/companion-policy.json";

type FixturePolicy = ReturnType<typeof destinationPolicy>;
const policies = Object.fromEntries(
  Object.entries(fixtures.policies).map(([name, raw]) => [name, destinationPolicy(raw)]),
) as Record<string, FixturePolicy>;

test("policies validate strictly: versions, rules, hops, and scopes", () => {
  const base = fixtures.policies.strict;
  expect(policies.strict.version).toBe("2026-09-26.1");
  expect(policies.strict.rules.length).toBe(3);
  expect(Object.isFrozen(policies.strict)).toBe(true);
  expect(Object.isFrozen(policies.strict.rules)).toBe(true);
  for (const bad of [
    null,
    [],
    { ...base, version: "" },
    { ...base, version: "has space" },
    { ...base, rules: "hooks.example.com" },
    { ...base, rules: [{ scheme: "https" }] },
    { ...base, rules: [{ scheme: "gopher", host: "x.example" }] },
    { ...base, rules: [{ scheme: "https", host: "" }] },
    { ...base, rules: [{ scheme: "https", host: "*.example.com", port: 0 }] },
    { ...base, rules: [{ scheme: "https", host: "*.example.com", port: 65536 }] },
    { ...base, rules: [{ scheme: "https", host: "*.example.com", port: 443.5 }] },
    { ...base, rules: [{ scheme: "https", host: "*.example.com", pathPrefix: "no-slash" }] },
    { ...base, rules: [{ scheme: "https", host: "*.example.com", credential: "lower" }] },
    { ...base, rules: [{ scheme: "https", host: "has space.com" }] },
    { ...base, rules: [{ scheme: "https", host: "*.999.999.999.999" }] },
    { ...base, redirect: "sometimes" },
    { ...base, maxRedirectHops: -1 },
    { ...base, maxRedirectHops: 9 },
    { ...base, loopback: "sometimes" },
    { ...base, privateNetworks: "sometimes" },
  ]) {
    expect(() => destinationPolicy(bad)).toThrow(TypeError);
  }
  // Scopes default to deny when the file omits them.
  expect(policies.strict.loopback).toBe("deny");
  expect(policies.strict.privateNetworks).toBe("deny");
  expect(() => envName("lower")).toThrow(TypeError);
  expect(envName("TENANT_HOOKS_TOKEN") as string).toBe("TENANT_HOOKS_TOKEN");
});

test("URL fixtures: allowed, denied, and the exact reason", () => {
  for (const kase of fixtures.destinations) {
    const policy = policies[kase.policy];
    const got = evaluateDestination(policy, kase.url);
    expect(`${kase.url} -> ${got.decision}`).toBe(`${kase.url} -> ${kase.decision}`);
    if (kase.decision === "allowed") {
      expect(got.reason).toBe("allowed");
      expect(got.ruleIndex).toBe((kase as { rule?: number }).rule);
      expect((got.credential as string | undefined) ?? null).toBe(
        (kase as { credential?: string }).credential ?? null,
      );
    } else {
      expect(got.reason).toBe((kase as { reason: string }).reason);
      expect(got.ruleIndex).toBeUndefined();
    }
  }
});

test("redirect fixtures: deny, same-origin, and re-admission", () => {
  for (const kase of fixtures.redirects) {
    const policy = policies[kase.policy];
    const got = evaluateRedirect(policy, kase.request, kase.location, kase.hops);
    expect(`${kase.location} -> ${got.decision}`).toBe(`${kase.location} -> ${kase.decision}`);
    if (kase.decision === "allowed") {
      expect(got.reason).toBe("allowed");
      expect(got.target).toBe((kase as { target: string }).target);
    } else {
      expect(got.reason).toBe((kase as { reason: string }).reason);
      expect(got.target).toBeUndefined();
    }
  }
  expect(() => evaluateRedirect(policies.strict, "https://a.example/", "/", -1)).toThrow(TypeError);
});

test("credential values resolve at send time and never reach logs", () => {
  const canary = "live-credential-value-9f31-must-never-log";
  const readEnvironment = (key: string) => (key === "TENANT_HOOKS_TOKEN" ? canary : undefined);
  const policy = policies.strict;
  const allowed = evaluateDestination(policy, "https://hooks.example.com/tenant/acme");
  expect(allowed.decision).toBe("allowed");
  expect(allowed.credential as string).toBe("TENANT_HOOKS_TOKEN");
  // Resolution works for the send path and reports missing/empty safely.
  expect(credentialValue(allowed.credential!, readEnvironment)).toBe(canary);
  expect(credentialValue(envName("MISSING_VAR"), readEnvironment)).toBeUndefined();
  expect(credentialValue(envName("TENANT_HOOKS_TOKEN"), () => "")).toBeUndefined();
  // No decision, redirect, redaction, or report string carries the value.
  const exposed = JSON.stringify({
    allowed,
    denied: evaluateDestination(policy, "https://evil.com/"),
    redirected: evaluateRedirect(policy, "https://hooks.example.com/tenant/a", "/tenant/b", 0),
    redacted: redactUrl("https://user:pass@hooks.example.com/tenant/a#x"),
    report: policyReport(policy),
    policy,
  });
  expect(exposed.includes(canary)).toBe(false);
  expect(exposed.includes("pass@")).toBe(false);
  expect(redactUrl("https://user:pass@hooks.example.com/tenant/a#x")).toBe(
    "https://hooks.example.com/tenant/a",
  );
  expect(redactUrl(":::not a url")).toBe("<invalid-url>");
});

test("private-literal scopes cover v4, v6, loopback, and link-local", () => {
  for (const literal of [
    "10.1.2.3",
    "172.16.0.1",
    "172.31.255.255",
    "192.168.0.1",
    "169.254.8.9",
  ]) {
    expect(isPrivateAddressLiteral(literal)).toBe(true);
    expect(evaluateDestination(policies.strict, `http://${literal}/`).reason).toBe(
      "private-network-denied",
    );
  }
  for (const literal of ["::1", "fc00::1", "fd12::9", "fe80::1"]) {
    expect(isPrivateAddressLiteral(literal)).toBe(true);
  }
  for (const literal of ["8.8.8.8", "1.1.1.1", "172.15.0.1", "172.32.0.1", "2001:db8::1"]) {
    expect(isPrivateAddressLiteral(literal)).toBe(false);
  }
  // Public literals still need a rule; the scope is not an admission.
  expect(evaluateDestination(policies.strict, "http://8.8.8.8/").reason).toBe("no-matching-rule");
});

test("the shipped companion policy parses and enforces its bindings", () => {
  const policy = destinationPolicy(companionPolicyFile);
  const tenant = evaluateDestination(policy, "https://hooks.example.com/tenant/acme/hook");
  expect(tenant.decision).toBe("allowed");
  expect(tenant.credential as string).toBe("TENANT_HOOKS_TOKEN");
  const partner = evaluateDestination(policy, "https://api.partner.example:8443/hook");
  expect(partner.decision).toBe("allowed");
  expect(partner.credential as string).toBe("PARTNER_HOOKS_TOKEN");
  expect(evaluateDestination(policy, "https://hooks.example.com/unscoped").decision).toBe("denied");
  expect(
    evaluateRedirect(policy, "https://hooks.example.com/tenant/a", "https://evil.com/", 0).reason,
  ).toBe("target-no-matching-rule");
  const report = policyReport(policy);
  expect(report).toEqual({
    version: "2026-09-26.1",
    redirect: "allowlisted",
    maxRedirectHops: 2,
    rules: ["hooks.example.com", "*.partner.example"],
  });
});
