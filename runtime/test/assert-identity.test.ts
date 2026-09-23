import { test, expect } from "bun:test";
import {
  rootIdentity,
  invocationIdentity,
  participantIdentities,
  callableIdentity,
  callableCaptures,
  invocationPath,
  compareInvocations,
  type InvocationIdentity,
  type CallableIdentity,
} from "../assert/identity.ts";
const root = { package: "p", declaration: "p::main", name: "sample" };

test("invocation occurrences belong to each parent and compare numeric preorder ordinals", () => {
  const identity = rootIdentity(root),
    first = invocationIdentity(identity, "p::main#2"),
    second = invocationIdentity(identity, "p::main#2"),
    laterSite = invocationIdentity(identity, "p::main#10");
  expect(invocationPath(first).segments[0].occurrence).toBe(0);
  expect(invocationPath(second).segments[0].occurrence).toBe(1);
  expect(compareInvocations(first, second)).toBeLessThan(0);
  expect(compareInvocations(second, laterSite)).toBeLessThan(0);
  const a = invocationIdentity(first, "p::helper#0"),
    b = invocationIdentity(second, "p::helper#0");
  expect(invocationPath(a).segments.at(-1)?.occurrence).toBe(0);
  expect(invocationPath(b).segments.at(-1)?.occurrence).toBe(0);
  expect(compareInvocations(a, b)).toBeLessThan(0);
  expect(Object.isFrozen(invocationPath(a).segments)).toBe(true);
});

test("direct, spread and nested positions are reserved independently of arrival order", () => {
  const identity = rootIdentity(root),
    positions = [[0], [1, 0], [1, 1], [2]];
  const participants = participantIdentities(identity, "p::main#3", positions);
  const visits = [3, 1, 2, 0].map((index) =>
    invocationIdentity(participants[index], "p::helper#1"),
  );
  visits.sort(compareInvocations);
  expect(visits.map((value) => invocationPath(value).segments.at(-1)?.participant)).toEqual(
    positions,
  );
  const nested = participantIdentities(participants[2], "p::helper#4", [[0], [1, 0]]);
  expect(invocationPath(nested[1]).segments.at(-1)?.participant).toEqual([1, 1, 1, 0]);
  positions[0][0] = 99;
  expect(invocationPath(participants[0]).segments[0].participant).toEqual([0]);
  expect(() => participantIdentities(identity, "p::main#3", [[0], [0]])).toThrow("duplicate");
});

test("callable receipts freeze captures and distinguish repeated creation sites without exposing values", () => {
  const identity = rootIdentity(root),
    receiver = Object.freeze({ private: "receiver" }),
    captures: unknown[] = [receiver, 1n];
  const a = callableIdentity(identity, "p::make#1", captures),
    b = callableIdentity(identity, "p::make#1", [receiver, 2n]);
  captures[1] = 99n;
  expect(callableCaptures(a)).toEqual([receiver, 1n]);
  expect(Object.isFrozen(callableCaptures(a))).toBe(true);
  const first = invocationIdentity(identity, "p::main#5", a),
    second = invocationIdentity(identity, "p::main#5", b);
  const firstReceipt = invocationPath(first).segments[0].callable!,
    secondReceipt = invocationPath(second).segments[0].callable!;
  expect(firstReceipt.occurrence).toBe(0);
  expect(secondReceipt.occurrence).toBe(1);
  expect(firstReceipt.instance).not.toBe(secondReceipt.instance);
  expect(JSON.stringify(invocationPath(first))).not.toContain("receiver");
  const repeated = rootIdentity(root),
    same = callableIdentity(repeated, "p::make#1", [receiver, 1n]);
  expect(
    invocationPath(invocationIdentity(repeated, "p::main#5", same)).segments[0].callable?.instance,
  ).toBe(firstReceipt.instance);
});

test("identity tokens fail closed across forged values and isolated roots", () => {
  const a = rootIdentity(root),
    b = rootIdentity(root);
  expect(() => invocationIdentity({} as InvocationIdentity, "p::main#0")).toThrow(
    "invalid invocation",
  );
  expect(() => invocationIdentity(a, "p::main#0", {} as CallableIdentity)).toThrow(
    "invalid callable",
  );
  expect(() => compareInvocations(a, b)).toThrow("different assertion roots");
  expect(() => invocationIdentity(b, "p::main#0", callableIdentity(a, "p::make#0", []))).toThrow(
    "another assertion root",
  );
  for (const site of ["", "p::main", "p::main#-1", "p::main#01", "p::main#9007199254740992"]) {
    expect(() => invocationIdentity(a, site)).toThrow("invalid lexical site");
  }
  for (const position of [[], [-1], [1.5], [0, 1, 2]])
    expect(() => participantIdentities(a, "p::main#0", [position])).toThrow("invalid participant");
});
