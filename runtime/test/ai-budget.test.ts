// R14 native token-budget guard contract tests (H07): profile
// qualification, ai_budget::within scope, guarded dispatch, usage
// settlement, and redacted accounting over the F01 durable ledger.
// Adapter integration over HTTP lives in ai-budget-adapters.test.ts.
import { test, expect } from "bun:test";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { createHash } from "node:crypto";
import {
  createBudgetGuard,
  createProfileRegistry,
  decodeProviderUsage,
  guardConnection,
  type AccountingReport,
  type BudgetTypes,
} from "../ai/budget.ts";
import { ownBytes } from "../bytes.ts";
import { failure, success, type Completion } from "../completion.ts";
import { dataProperty, record } from "../data.ts";
import {
  concreteTypeDigestInput,
  createDomainRuntime,
  domainFailureDiagnostics,
  type FailureShape,
} from "../domain.ts";
import { epochSchedule } from "../outbound/epoch.ts";
import { openFileLedgerStore } from "../outbound/file-ledger.ts";
import {
  configurePool,
  createMemoryLedgerStore,
  epochStatus,
  fence,
  ledgerHealth,
  reconcile,
  reserve,
  type LedgerStore,
} from "../outbound/ledger.ts";
import { invocationContext } from "../outbound/identity.ts";

const HOUR = 3_600_000;
const PROFILE = { provider: "typesafe", model: "jev", version: "1.13.0" };
const origin = { source: "test:ai-budget", start: 0, end: 0, invocation: [] };
const OPERATION = "test:ai/budgeted";

const hash = (key: unknown): string =>
  createHash("sha256").update(concreteTypeDigestInput(key)).digest("hex");
const shape = (
  kind: string,
  declaration: string,
  fields: { name: string; type: string }[] = [],
): FailureShape => ({
  identity: hash([kind, declaration]),
  kind,
  declaration,
  fields,
  arguments: [],
  leaves: [],
  inputs: [],
  errors: [],
});
// Project-spelled declaration identities: the published
// can.std.ai_budget@1 catalogue entries arrive through E; the guard
// only needs the type identities injected here.
const str = shape("primitive", "str");
const int = shape("primitive", "int");
const exceededShape = shape("error", "can.project.ai_budget.exceeded", [
  { name: "pool", type: str.identity },
  { name: "required", type: int.identity },
  { name: "remaining", type: int.identity },
  { name: "epoch_reset_ms", type: int.identity },
]);
const unavailableShape = shape("error", "can.project.ai_budget.unavailable", [
  { name: "reason", type: str.identity },
  { name: "correlation", type: str.identity },
  { name: "invocation", type: str.identity },
]);
const domain = createDomainRuntime({
  declarations: [
    {
      identity: "can.project.ai_budget.exceeded",
      name: "can.std.ai_budget@1::exceeded",
      parameters: 0,
    },
    {
      identity: "can.project.ai_budget.unavailable",
      name: "can.std.ai_budget@1::unavailable",
      parameters: 0,
    },
  ],
  shapes: [str, int, exceededShape, unavailableShape],
});
const budgetTypes: BudgetTypes = {
  exceeded: exceededShape.identity,
  unavailable: unavailableShape.identity,
};

function setup(limit = 1000, upperBound = 100) {
  const store = createMemoryLedgerStore("ai-budget");
  const registry = createProfileRegistry();
  registry.qualify({ ...PROFILE, upperBound });
  const reports: AccountingReport[] = [];
  let now = 5;
  const guard = createBudgetGuard(domain, budgetTypes, {
    store,
    registry,
    nowMs: () => now,
    report: (entry) => reports.push(entry),
  });
  const connection = guardConnection(guard, { permittedPool: "default", profile: PROFILE });
  return {
    store,
    registry,
    reports,
    guard,
    connection,
    setNow: (value: number) => {
      now = value;
    },
  };
}

async function configured(store: LedgerStore, limit = 1000): Promise<void> {
  const done = await configurePool(
    store,
    epochSchedule({ pool: "default", limit, periodMs: HOUR, anchorMs: 0, version: "1" }),
  );
  expect(done).toEqual({ configured: true });
}

const context = (tenant = "acme", pool = "default", correlation = "req-1") =>
  invocationContext(tenant, pool, correlation);
const body = (value: unknown): Uint8Array =>
  new TextEncoder().encode(typeof value === "string" ? value : JSON.stringify(value));
const usageBody = (input: unknown, output: unknown, extra: object = {}): Uint8Array =>
  body({ usage: { input_tokens: input, output_tokens: output, ...extra } });

function checkExceeded(
  result: Completion<unknown>,
  expected: { pool: string; required: bigint; remaining: bigint; epoch_reset_ms: bigint },
): void {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain failure");
  const details = domainFailureDiagnostics(result.value);
  expect(details.declaration.name).toBe("can.std.ai_budget@1::exceeded");
  expect(details.provenance).toMatchObject({ boundary: "native", operation: OPERATION });
  expect(details.payload).toMatchObject(expected);
}

function checkUnavailable(
  result: Completion<unknown>,
  reason: string,
): { correlation: string; invocation: string } {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain failure");
  const details = domainFailureDiagnostics(result.value);
  expect(details.declaration.name).toBe("can.std.ai_budget@1::unavailable");
  expect(details.provenance).toMatchObject({ boundary: "native", operation: OPERATION });
  expect(dataProperty(details.payload, "reason")).toBe(reason);
  return {
    correlation: dataProperty(details.payload, "correlation") as string,
    invocation: dataProperty(details.payload, "invocation") as string,
  };
}

function countedSend(
  completion: Completion<unknown>,
  responseBody: Uint8Array | undefined,
  delayMs = 0,
): {
  send: () => Promise<{ completion: Completion<unknown>; body: Uint8Array | undefined }>;
  calls: () => number;
} {
  let calls = 0;
  return {
    send: async () => {
      calls += 1;
      if (delayMs > 0) await Bun.sleep(delayMs);
      return { completion, body: responseBody };
    },
    calls: () => calls,
  };
}

test("profile registry pins qualified bounds and rejects conflicts", () => {
  const registry = createProfileRegistry();
  expect(registry.boundFor("typesafe", "jev", "1.13.0")).toBeUndefined();
  expect(registry.boundFor("", "", "")).toBeUndefined();
  const qualified = registry.qualify({ ...PROFILE, upperBound: 100 });
  expect(qualified).toEqual({ ...PROFILE, upperBound: 100 });
  expect(Object.isFrozen(qualified)).toBe(true);
  expect(registry.boundFor("typesafe", "jev", "1.13.0")).toBe(100);
  // Same proof re-qualifies idempotently; a changed bound conflicts.
  expect(registry.qualify({ ...PROFILE, upperBound: 100 })).toBe(qualified);
  expect(() => registry.qualify({ ...PROFILE, upperBound: 101 })).toThrow(TypeError);
  // A changed version is a different, unqualified profile.
  expect(registry.boundFor("typesafe", "jev", "1.14.0")).toBeUndefined();
  for (const upperBound of [0, -1, 1.5, Number.MAX_SAFE_INTEGER + 1, "100", undefined])
    expect(() => registry.qualify({ ...PROFILE, upperBound })).toThrow(TypeError);
  expect(() => registry.qualify({ ...PROFILE, version: "", upperBound: 1 })).toThrow(TypeError);
  expect(registry.profiles()).toEqual([{ ...PROFILE, upperBound: 100 }]);
});

test("usage decoder sums top-level counters and ignores breakdowns", () => {
  const decode = (value: unknown): unknown =>
    decodeProviderUsage(
      typeof value === "string" ? ownBytes(new TextEncoder().encode(value)) : value,
      65536,
    );
  expect(decode(ownBytes(body({ usage: { input_tokens: 10, output_tokens: 5 } })))).toEqual({
    inputTokens: 10,
    outputTokens: 5,
  });
  // Cached/reasoning breakdowns never count a second time.
  expect(
    decode(
      ownBytes(
        body({
          model: "jev",
          usage: {
            input_tokens: 10,
            output_tokens: 5,
            input_tokens_details: { cached_tokens: 8 },
            output_tokens_details: { reasoning_tokens: 4 },
          },
          answers: {},
        }),
      ),
    ),
  ).toEqual({ inputTokens: 10, outputTokens: 5 });
  for (const value of [
    undefined,
    "{}",
    "[]",
    "null",
    "{not json",
    JSON.stringify({ usage: null }),
    JSON.stringify({ usage: [] }),
    JSON.stringify({ usage: {} }),
    JSON.stringify({ usage: { input_tokens: -1, output_tokens: 0 } }),
    JSON.stringify({ usage: { input_tokens: 1.5, output_tokens: 0 } }),
    JSON.stringify({ usage: { input_tokens: "10", output_tokens: 5 } }),
    JSON.stringify({ usage: { input_tokens: 10 } }),
    JSON.stringify({ usage: { input_tokens: Number.MAX_SAFE_INTEGER, output_tokens: 1 } }),
  ])
    expect(decode(value === undefined ? undefined : ownBytes(body(value)))).toBeUndefined();
  // Bodies past the connection cap are untrustworthy, never decoded.
  expect(decode(ownBytes(body({ usage: { input_tokens: 1, output_tokens: 1 } })))).toBeDefined();
  expect(
    decodeProviderUsage(ownBytes(body({ usage: { input_tokens: 1, output_tokens: 1 } })), 4),
  ).toBeUndefined();
});

test("within runs the operation with a fresh scope context", async () => {
  const { guard, store } = setup();
  const seen: unknown[] = [];
  const result = await guard.within(
    "acme",
    "default",
    async (scope) => {
      seen.push(scope);
      expect(Object.isFrozen(scope)).toBe(true);
      return success("scoped-value");
    },
    origin,
    OPERATION,
  );
  expect(result).toEqual(success("scoped-value"));
  expect(seen).toHaveLength(1);
  expect(seen[0]).toMatchObject({ tenant: "acme", pool: "default" });
  // The scope adds context, not tokens: no ledger row exists.
  expect(await epochStatus(store, "acme", "default", 0)).toEqual({ status: "absent" });
});

test("within propagates callback failures and contains callback throws", async () => {
  const { guard } = setup();
  const callbackFailure = failure(
    domain.create(
      budgetTypes.unavailable,
      record(budgetTypes.unavailable, [
        ["reason", "missing-qualification"],
        ["correlation", "req-9"],
        ["invocation", "inv-9"],
      ]),
      origin,
    ),
  );
  const propagated = await guard.within(
    "acme",
    "default",
    async () => callbackFailure,
    origin,
    OPERATION,
  );
  expect(propagated).toBe(callbackFailure);
  const thrown = await guard.within(
    "acme",
    "default",
    async () => {
      throw Error("callback bug");
    },
    origin,
    OPERATION,
  );
  expect(thrown.kind).toBe("standard");
});

test("within rejects invalid scope values and nested replacement", async () => {
  const { guard } = setup();
  for (const [tenant, pool] of [
    ["", "default"],
    ["acme", ""],
    ["has space", "default"],
    ["acme", "default!"],
  ])
    checkUnavailable(
      await guard.within(tenant, pool, async () => success(0), origin, OPERATION),
      "invalid-policy",
    );
  // Same-context nesting returns the active context unchanged.
  const active = invocationContext("acme", "default", "req-active");
  let nested: unknown;
  const same = await guard.within(
    "acme",
    "default",
    async (scope) => {
      nested = scope;
      return success(0);
    },
    origin,
    OPERATION,
    active,
  );
  expect(same.kind).toBe("ok");
  expect(nested).toBe(active);
  // Replacing the tenant or pool fails closed.
  checkUnavailable(
    await guard.within("other", "default", async () => success(0), origin, OPERATION, active),
    "invalid-policy",
  );
  checkUnavailable(
    await guard.within("acme", "other", async () => success(0), origin, OPERATION, active),
    "invalid-policy",
  );
  checkUnavailable(
    await guard.within("acme", "default", async () => success(0), origin, OPERATION, {
      tenant: "acme",
    }),
    "invalid-policy",
  );
});

test("dispatch rejects missing or malformed context before touching the ledger", async () => {
  const { connection, store, reports } = setup();
  await configured(store);
  const { send, calls } = countedSend(success("value"), usageBody(10, 5));
  const missing = checkUnavailable(
    await connection.dispatch({
      context: undefined,
      where: origin,
      operation: OPERATION,
      maxBodyBytes: 8192,
      send,
    }),
    "missing-context",
  );
  expect(missing.correlation.length).toBeGreaterThan(0);
  expect(missing.invocation.length).toBeGreaterThan(0);
  for (const bad of [
    null,
    "req-1",
    { tenant: "acme" },
    { tenant: "", pool: "default", correlation: "req-1" },
  ])
    checkUnavailable(
      await connection.dispatch({
        context: bad,
        where: origin,
        operation: OPERATION,
        maxBodyBytes: 8192,
        send,
      }),
      "invalid-policy",
    );
  checkUnavailable(
    await connection.dispatch({
      context: context(),
      where: origin,
      operation: OPERATION,
      maxBodyBytes: 0,
      send,
    }),
    "invalid-policy",
  );
  expect(calls()).toBe(0);
  expect(reports).toEqual([]);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual({ status: "absent" });
});

test("dispatch enforces the connection permitted pool", async () => {
  const { guard, store, reports } = setup();
  await configured(store);
  const { send, calls } = countedSend(success("value"), usageBody(10, 5));
  const attempt = { where: origin, operation: OPERATION, maxBodyBytes: 8192, send };
  // An arbitrary pool name never creates fresh allowance.
  checkUnavailable(
    await guard.dispatch(
      { permittedPool: "default", profile: PROFILE },
      { ...attempt, context: context("acme", "other", "req-1") },
    ),
    "invalid-policy",
  );
  const mismatch = guardConnection(guard, { permittedPool: "other", profile: PROFILE });
  checkUnavailable(await mismatch.dispatch({ ...attempt, context: context() }), "invalid-policy");
  const malformed = guardConnection(guard, { permittedPool: "", profile: PROFILE });
  checkUnavailable(await malformed.dispatch({ ...attempt, context: context() }), "invalid-policy");
  expect(calls()).toBe(0);
  expect(reports).toEqual([]);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual({ status: "absent" });
  expect(await epochStatus(store, "acme", "other", 0)).toEqual({ status: "absent" });
});

test("unqualified profiles reject before send with no ledger row", async () => {
  const store = createMemoryLedgerStore("ai-budget-unqualified");
  await configured(store);
  const guard = createBudgetGuard(domain, budgetTypes, {
    store,
    registry: createProfileRegistry(),
    nowMs: () => 5,
  });
  const connection = guardConnection(guard, { permittedPool: "default", profile: PROFILE });
  const { send, calls } = countedSend(success("value"), usageBody(10, 5));
  const failed = checkUnavailable(
    await connection.dispatch({
      context: context(),
      where: origin,
      operation: OPERATION,
      maxBodyBytes: 8192,
      send,
    }),
    "missing-qualification",
  );
  expect(failed.correlation).toBe("req-1");
  expect(calls()).toBe(0);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual({ status: "absent" });
});

test("exact-fit admission settles and excess rejects with detail before send", async () => {
  const { connection, store, reports } = setup(100, 100);
  await configured(store, 100);
  const first = countedSend(success("first"), usageBody(50, 10));
  const result = await connection.dispatch({
    context: context(),
    where: origin,
    operation: OPERATION,
    maxBodyBytes: 8192,
    send: first.send,
  });
  expect(result.kind).toBe("ok");
  expect(first.calls()).toBe(1);
  expect(reports).toEqual([
    expect.objectContaining({
      tenant: "acme",
      pool: "default",
      correlation: "req-1",
      profile: "typesafe/jev/1.13.0",
      upperBound: 100,
      epochStart: 0,
      epochEnd: HOUR,
      outcome: "settled",
      actual: 60,
      released: 40,
    }),
  ]);
  expect(reports[0].invocation.length).toBeGreaterThan(0);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 60, held: 0 }) }),
  );
  // committed(60) + U(100) no longer fits: immediate typed rejection,
  // zero provider I/O, no report for a reservation that never existed.
  const second = countedSend(success("second"), usageBody(1, 1));
  checkExceeded(
    await connection.dispatch({
      context: context("acme", "default", "req-2"),
      where: origin,
      operation: OPERATION,
      maxBodyBytes: 8192,
      send: second.send,
    }),
    { pool: "default", required: 100n, remaining: 40n, epoch_reset_ms: BigInt(HOUR) },
  );
  expect(second.calls()).toBe(0);
  expect(reports).toHaveLength(1);
});

test("metering is independent of whether the result validates", async () => {
  const { connection, store, reports } = setup();
  await configured(store);
  const providerFailure = failure(
    domain.create(
      budgetTypes.unavailable,
      record(budgetTypes.unavailable, [
        ["reason", "invalid-usage"],
        ["correlation", "req-provider"],
        ["invocation", "inv-provider"],
      ]),
      origin,
    ),
  );
  const { send, calls } = countedSend(providerFailure, usageBody(70, 10));
  const result = await connection.dispatch({
    context: context(),
    where: origin,
    operation: OPERATION,
    maxBodyBytes: 8192,
    send,
  });
  expect(result).toBe(providerFailure);
  expect(calls()).toBe(1);
  expect(reports).toEqual([
    expect.objectContaining({ outcome: "settled", actual: 80, released: 20 }),
  ]);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 80, held: 0 }) }),
  );
});

test("breach invalidates the result and quarantines the profile", async () => {
  const { connection, guard, registry, store, reports } = setup(1000, 100);
  await configured(store);
  const { send } = countedSend(success("value"), usageBody(90, 20));
  const failed = checkUnavailable(
    await connection.dispatch({
      context: context(),
      where: origin,
      operation: OPERATION,
      maxBodyBytes: 8192,
      send,
    }),
    "breached-profile",
  );
  expect(failed.correlation).toBe("req-1");
  expect(reports).toEqual([
    expect.objectContaining({ outcome: "breach", actual: 110, upperBound: 100 }),
  ]);
  // Actuals record without clamping: committed holds the full 110.
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 110, held: 0 }) }),
  );
  // The breached profile stays quarantined for new budgeted sends.
  const retry = countedSend(success("value"), usageBody(1, 1));
  checkUnavailable(
    await connection.dispatch({
      context: context("acme", "default", "req-2"),
      where: origin,
      operation: OPERATION,
      maxBodyBytes: 8192,
      send: retry.send,
    }),
    "breached-profile",
  );
  expect(retry.calls()).toBe(0);
  // An independent profile is unaffected.
  registry.qualify({ provider: "other", model: "m", version: "9", upperBound: 50 });
  const other = guardConnection(guard, {
    permittedPool: "default",
    profile: { provider: "other", model: "m", version: "9" },
  });
  const fine = countedSend(success("fine"), usageBody(10, 5));
  expect(
    (
      await other.dispatch({
        context: context("acme", "default", "req-3"),
        where: origin,
        operation: OPERATION,
        maxBodyBytes: 8192,
        send: fine.send,
      })
    ).kind,
  ).toBe("ok");
  expect(await ledgerHealth(store)).toEqual(
    expect.objectContaining({
      status: "ok",
      committed: 125,
      held: 0,
      unresolved: 0,
      quarantined: ["typesafe/jev/1.13.0"],
    }),
  );
});

test("same-ID redispatch never resends a settled reservation", async () => {
  const { connection, store, reports } = setup();
  await configured(store);
  const attempt = { context: context(), where: origin, operation: OPERATION, maxBodyBytes: 8192 };
  const first = countedSend(success("one"), usageBody(50, 0));
  expect(
    await connection.dispatch({ ...attempt, invocationId: "inv-redeliver", send: first.send }),
  ).toEqual(success("one"));
  // A settled reservation is never resent: fencing a settled attempt
  // conflicts instead of delivering duplicate provider work.
  const redelivery = countedSend(success("one-again"), usageBody(50, 0));
  checkUnavailable(
    await connection.dispatch({ ...attempt, invocationId: "inv-redeliver", send: redelivery.send }),
    "conflicting-settlement",
  );
  expect(redelivery.calls()).toBe(0);
  expect(reports.map((entry) => entry.outcome)).toEqual(["settled"]);
  // Conflicting usage on the same record conflicts the same way.
  const conflict = countedSend(success("other"), usageBody(60, 0));
  checkUnavailable(
    await connection.dispatch({ ...attempt, invocationId: "inv-redeliver", send: conflict.send }),
    "conflicting-settlement",
  );
  expect(conflict.calls()).toBe(0);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 50, held: 0 }) }),
  );
  // Same ID with changed reservation parameters conflicts pre-send.
  const changed = countedSend(success("changed"), usageBody(1, 1));
  checkUnavailable(
    await connection.dispatch({
      context: context("acme", "default", "req-other"),
      where: origin,
      operation: OPERATION,
      maxBodyBytes: 8192,
      invocationId: "inv-redeliver",
      send: changed.send,
    }),
    "conflicting-settlement",
  );
  expect(changed.calls()).toBe(0);
});

test("provider failures without usage keep the original completion and the full hold", async () => {
  const { connection, guard, store, reports } = setup();
  await configured(store);
  const providerFailure = failure(
    domain.create(
      budgetTypes.unavailable,
      record(budgetTypes.unavailable, [
        ["reason", "timeout"],
        ["correlation", "req-provider"],
        ["invocation", "inv-provider"],
      ]),
      origin,
    ),
  );
  const { send, calls } = countedSend(providerFailure, undefined);
  const result = await connection.dispatch({
    context: context(),
    where: origin,
    operation: OPERATION,
    maxBodyBytes: 8192,
    send,
  });
  expect(result).toBe(providerFailure);
  expect(calls()).toBe(1);
  expect(reports).toEqual([expect.objectContaining({ outcome: "unresolved" })]);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({
      row: expect.objectContaining({ committed: 0, held: 100 }),
      unresolved: 1,
    }),
  );
  // Late authoritative usage reconciles the same record by invocation ID.
  const late = await guard.settleUsage({
    invocationId: reports[0].invocation,
    inputTokens: 30,
    outputTokens: 5,
  });
  expect(late).toEqual({ outcome: "settled", actual: 35, released: 65, duplicate: false });
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({
      row: expect.objectContaining({ committed: 35, held: 0 }),
      unresolved: 0,
    }),
  );
});

test("send crashes retain the hold and surface the cause", async () => {
  const { connection, store, reports } = setup();
  await configured(store);
  let calls = 0;
  const result = await connection.dispatch({
    context: context(),
    where: origin,
    operation: OPERATION,
    maxBodyBytes: 8192,
    send: async () => {
      calls += 1;
      throw Error("provider boom");
    },
  });
  expect(result.kind).toBe("standard");
  expect(calls).toBe(1);
  expect(reports).toEqual([expect.objectContaining({ outcome: "unresolved" })]);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({
      row: expect.objectContaining({ committed: 0, held: 100 }),
      unresolved: 1,
    }),
  );
});

test("invalid usage keeps the hold and preserves the result", async () => {
  const { connection, store, reports } = setup();
  await configured(store);
  for (const responseBody of [
    usageBody(-1, 5),
    usageBody(10.5, 5),
    body({ usage: { input_tokens: 1 } }),
    body({}),
    body("{not json"),
    new Uint8Array([0xff, 0xfe]),
  ]) {
    const { send } = countedSend(success("kept"), responseBody);
    expect(
      await connection.dispatch({
        context: context("acme", "default", `req-${reports.length}`),
        where: origin,
        operation: OPERATION,
        maxBodyBytes: 8192,
        send,
      }),
    ).toEqual(success("kept"));
  }
  expect(reports.map((entry) => entry.outcome)).toEqual(Array(6).fill("unresolved"));
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({
      row: expect.objectContaining({ committed: 0, held: 600 }),
      unresolved: 6,
    }),
  );
});

test("tenant rows isolate allowances under one shared pool schedule", async () => {
  const { connection, store } = setup(100, 100);
  await configured(store, 100);
  const attempt = { where: origin, operation: OPERATION, maxBodyBytes: 8192 };
  const spent = countedSend(success("a"), usageBody(100, 0));
  expect(
    await connection.dispatch({
      ...attempt,
      context: context("a", "default", "req-a"),
      send: spent.send,
    }),
  ).toEqual(success("a"));
  // Tenant a exhausted its own row; tenant b keeps a full allowance.
  const denied = countedSend(success("a2"), usageBody(1, 0));
  checkExceeded(
    await connection.dispatch({
      ...attempt,
      context: context("a", "default", "req-a2"),
      send: denied.send,
    }),
    { pool: "default", required: 100n, remaining: 0n, epoch_reset_ms: BigInt(HOUR) },
  );
  const allowed = countedSend(success("b"), usageBody(100, 0));
  expect(
    await connection.dispatch({
      ...attempt,
      context: context("b", "default", "req-b"),
      send: allowed.send,
    }),
  ).toEqual(success("b"));
  expect(await epochStatus(store, "a", "default", 0)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 100, held: 0 }) }),
  );
  expect(await epochStatus(store, "b", "default", 0)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 100, held: 0 }) }),
  );
});

test("epoch transitions bound late settlement to the original epoch", async () => {
  const { connection, guard, store, reports, setNow } = setup(100, 100);
  await configured(store, 100);
  const attempt = { where: origin, operation: OPERATION, maxBodyBytes: 8192 };
  const pending = countedSend(success("pending"), undefined);
  expect(await connection.dispatch({ ...attempt, context: context(), send: pending.send })).toEqual(
    success("pending"),
  );
  // A new epoch carries its own allowance: two allowances can fall on
  // either side of a boundary. This is an allowance per configured
  // period, not a rolling rate limit.
  setNow(HOUR + 5);
  const next = countedSend(success("next"), usageBody(100, 0));
  expect(
    await connection.dispatch({
      ...attempt,
      context: context("acme", "default", "req-2"),
      send: next.send,
    }),
  ).toEqual(success("next"));
  // Late usage still settles to the original admission epoch; the new
  // epoch neither moves nor forgives old usage.
  const late = await guard.settleUsage({
    invocationId: reports[0].invocation,
    inputTokens: 60,
    outputTokens: 0,
  });
  expect(late).toEqual({ outcome: "settled", actual: 60, released: 40, duplicate: false });
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 60, held: 0 }) }),
  );
  expect(await epochStatus(store, "acme", "default", HOUR)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 100, held: 0 }) }),
  );
});

test("parallel admission stays exact under contention", async () => {
  const { connection, store, reports } = setup(500, 40);
  await configured(store, 500);
  const outcomes = await Promise.all(
    Array.from({ length: 20 }, (_, index) => {
      const { send } = countedSend(success(index), undefined);
      return connection.dispatch({
        context: context("acme", "default", `req-${index}`),
        where: origin,
        operation: OPERATION,
        maxBodyBytes: 8192,
        invocationId: `inv-parallel-${index}`,
        send,
      });
    }),
  );
  expect(outcomes.filter((outcome) => outcome.kind === "ok")).toHaveLength(12);
  expect(outcomes.filter((outcome) => outcome.kind === "domain")).toHaveLength(8);
  for (const outcome of outcomes) {
    if (outcome.kind !== "domain") continue;
    expect(domainFailureDiagnostics(outcome.value).declaration.name).toBe(
      "can.std.ai_budget@1::exceeded",
    );
  }
  expect(reports).toHaveLength(12);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({
      row: expect.objectContaining({ committed: 0, held: 480 }),
      unresolved: 12,
    }),
  );
});

test("concurrent same-ID dispatches never double-send", async () => {
  const { connection, store, reports } = setup();
  await configured(store);
  const { send, calls } = countedSend(success("sent"), undefined, 20);
  const attempt = {
    context: context(),
    where: origin,
    operation: OPERATION,
    maxBodyBytes: 8192,
    invocationId: "inv-race",
    send,
  };
  const [first, second] = await Promise.all([
    connection.dispatch(attempt),
    connection.dispatch(attempt),
  ]);
  expect(calls()).toBe(1);
  expect([first.kind, second.kind].sort()).toEqual(["domain", "ok"]);
  checkUnavailable(first.kind === "domain" ? first : second, "conflicting-settlement");
  expect(reports.map((entry) => entry.outcome)).toEqual(["unresolved"]);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({
      row: expect.objectContaining({ committed: 0, held: 100 }),
      unresolved: 1,
    }),
  );
});

test("ambiguous and unavailable storage fail closed without dispatch", async () => {
  const backing = createMemoryLedgerStore("ai-budget-flaky");
  let commits = 0;
  let failLoad = false;
  const failCommits = new Set<number>();
  let contendFrom = 0;
  let contendUntil = 0;
  const store: LedgerStore = {
    name: "flaky",
    load: async () => {
      if (failLoad) throw Error("disk gone");
      return backing.load();
    },
    commit: async (expected, next) => {
      commits += 1;
      if (failCommits.has(commits)) throw Error("commit unknown");
      if (commits >= contendFrom && commits < contendUntil) return { committed: false };
      return backing.commit(expected, next);
    },
  };
  const registry = createProfileRegistry();
  registry.qualify({ ...PROFILE, upperBound: 100 });
  const reports: AccountingReport[] = [];
  const guard = createBudgetGuard(domain, budgetTypes, {
    store,
    registry,
    nowMs: () => 5,
    report: (entry) => reports.push(entry),
  });
  const connection = guardConnection(guard, { permittedPool: "default", profile: PROFILE });
  const attempt = { context: context(), where: origin, operation: OPERATION, maxBodyBytes: 8192 };
  // Pool configuration burns commit 1 through the counting wrapper so
  // each dispatch below burns known ordinals.
  await configured(store);

  // Ambiguous reserve (commit 2): never dispatch; nothing reserved yet.
  failCommits.add(2);
  const ambiguous = countedSend(success("x"), usageBody(1, 1));
  checkUnavailable(
    await connection.dispatch({ ...attempt, send: ambiguous.send }),
    "ambiguous-commit",
  );
  expect(ambiguous.calls()).toBe(0);
  expect(reports).toEqual([]);
  // Unavailable load: fail closed the same way.
  failLoad = true;
  const dark = countedSend(success("x"), usageBody(1, 1));
  checkUnavailable(
    await connection.dispatch({ ...attempt, send: dark.send }),
    "storage-unavailable",
  );
  expect(dark.calls()).toBe(0);
  failLoad = false;

  // Ambiguous fence (reserve 3 ok, fence 4 unknown): never dispatch;
  // the admitted hold stays durable and reported.
  failCommits.add(4);
  const unfenced = countedSend(success("x"), undefined);
  checkUnavailable(
    await connection.dispatch({
      ...attempt,
      context: context("acme", "default", "req-fence"),
      invocationId: "inv-fence",
      send: unfenced.send,
    }),
    "ambiguous-commit",
  );
  expect(unfenced.calls()).toBe(0);
  expect(reports).toEqual([
    expect.objectContaining({ outcome: "unresolved", invocation: "inv-fence" }),
  ]);
  expect(await reconcile(store, "inv-fence")).toMatchObject({
    status: "found",
    record: expect.objectContaining({ state: "held", fenced: false, upperBound: 100 }),
  });

  // Definitively failed fence (reserve 5 ok, fence 6..69 contend):
  // the proven never-fenced reservation releases.
  contendFrom = 6;
  contendUntil = 70;
  const released = countedSend(success("x"), undefined);
  checkUnavailable(
    await connection.dispatch({
      ...attempt,
      context: context("acme", "default", "req-release"),
      invocationId: "inv-release-fence",
      send: released.send,
    }),
    "storage-unavailable",
  );
  expect(released.calls()).toBe(0);
  contendFrom = 0;
  contendUntil = 0;
  expect(reports.map((entry) => entry.outcome)).toEqual(["unresolved", "released"]);
  expect(await reconcile(store, "inv-release-fence")).toMatchObject({
    status: "found",
    record: expect.objectContaining({ state: "released" }),
  });

  // Ambiguous settlement (reserve/fence ok, settle unknown): the
  // provider completion stays intact while the hold stays unresolved;
  // a later retry reconciles by invocation ID.
  const unsettled = countedSend(success("kept"), usageBody(40, 5));
  const pending = connection.dispatch({
    ...attempt,
    context: context("acme", "default", "req-settle"),
    invocationId: "inv-settle",
    send: unsettled.send,
  });
  failCommits.add(commits + 3);
  expect(await pending).toEqual(success("kept"));
  expect(reports.map((entry) => entry.outcome)).toEqual(["unresolved", "released", "unresolved"]);
  expect(
    await guard.settleUsage({ invocationId: "inv-settle", inputTokens: 40, outputTokens: 5 }),
  ).toEqual({
    outcome: "settled",
    actual: 45,
    released: 55,
    duplicate: false,
  });
});

test("release needs no-dispatch proof at guard level", async () => {
  const { guard, store, reports } = setup();
  await configured(store);
  const admitted = await reserve(store, {
    tenant: "acme",
    pool: "default",
    correlation: "req-release",
    profile: PROFILE,
    qualified: true,
    upperBound: 100,
    invocationId: "inv-release",
    nowMs: 5,
  });
  expect(admitted.outcome).toBe("admitted");
  const released = await guard.releaseReservation("inv-release");
  expect(released.outcome).toBe("released");
  expect(reports).toEqual([
    expect.objectContaining({ outcome: "released", invocation: "inv-release" }),
  ]);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ held: 0 }) }),
  );
  // Fenced attempts keep their hold.
  await reserve(store, {
    tenant: "acme",
    pool: "default",
    correlation: "req-fenced",
    profile: PROFILE,
    qualified: true,
    upperBound: 100,
    invocationId: "inv-fenced",
    nowMs: 5,
  });
  expect(await fence(store, "inv-fenced")).toMatchObject({ outcome: "fenced" });
  expect(await guard.releaseReservation("inv-fenced")).toMatchObject({ outcome: "held" });
  expect(reports.map((entry) => entry.outcome)).toEqual(["released", "unresolved"]);
  expect(await guard.releaseReservation("inv-missing")).toMatchObject({
    outcome: "unavailable",
    reason: "unknown-invocation",
  });
});

test("late settlement reconciles unknown, duplicate, and conflicting usage", async () => {
  const { connection, guard, store, reports } = setup();
  await configured(store);
  const { send } = countedSend(success("late"), undefined);
  await connection.dispatch({
    context: context(),
    where: origin,
    operation: OPERATION,
    maxBodyBytes: 8192,
    send,
  });
  const id = reports[0].invocation;
  expect(await guard.settleUsage({ invocationId: id, inputTokens: 40, outputTokens: 5 })).toEqual({
    outcome: "settled",
    actual: 45,
    released: 55,
    duplicate: false,
  });
  expect(await guard.settleUsage({ invocationId: id, inputTokens: 40, outputTokens: 5 })).toEqual({
    outcome: "settled",
    actual: 45,
    released: 0,
    duplicate: true,
  });
  expect(await guard.settleUsage({ invocationId: id, inputTokens: 1, outputTokens: 1 })).toEqual({
    outcome: "unavailable",
    reason: "conflicting-settlement",
  });
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 45, held: 0 }) }),
  );
  // Unknown invocations and invalid usage fail closed.
  expect(
    await guard.settleUsage({ invocationId: "inv-absent", inputTokens: 1, outputTokens: 1 }),
  ).toEqual({
    outcome: "unavailable",
    reason: "unknown-invocation",
  });
  const { send: pendingSend } = countedSend(success("pending"), undefined);
  await connection.dispatch({
    context: context("acme", "default", "req-pending"),
    where: origin,
    operation: OPERATION,
    maxBodyBytes: 8192,
    send: pendingSend,
  });
  const pending = reports.find((entry) => entry.correlation === "req-pending")!;
  expect(
    await guard.settleUsage({ invocationId: pending.invocation, inputTokens: -1, outputTokens: 0 }),
  ).toEqual({
    outcome: "unavailable",
    reason: "invalid-usage",
  });
  expect(await reconcile(store, pending.invocation)).toMatchObject({
    status: "found",
    record: expect.objectContaining({ state: "fenced" }),
  });
});

test("two processes share one file ledger with exact admission", async () => {
  const directory = await mkdtemp(join(tmpdir(), "ai-budget-process-"));
  try {
    const path = join(directory, "ledger.json");
    const store = openFileLedgerStore(path);
    await configured(store, 500);
    const worker = join(dirname(import.meta.path), "ai-budget-admission-worker.ts");
    const run = (prefix: string) =>
      Bun.spawn(["bun", worker, path, "acme", "default", prefix, "10", "40", "5"], {
        stdout: "pipe",
        stderr: "pipe",
      });
    const first = run("worker-a");
    const second = run("worker-b");
    const [codeA, codeB] = await Promise.all([first.exited, second.exited]);
    const outA = await new Response(first.stdout).text();
    const outB = await new Response(second.stdout).text();
    if (codeA !== 0 || codeB !== 0) {
      const errA = await new Response(first.stderr).text();
      const errB = await new Response(second.stderr).text();
      throw Error(`worker failed: ${codeA}/${codeB}\n${errA}\n${errB}`);
    }
    const total =
      (JSON.parse(outA) as { admitted: number; exceeded: number }).admitted +
      (JSON.parse(outB) as { admitted: number; exceeded: number }).admitted;
    // floor(500 / 40) = 12 admissions at 480 held; zero lost updates.
    expect(total).toBe(12);
    const reopened = openFileLedgerStore(path);
    expect(await epochStatus(reopened, "acme", "default", 0)).toEqual(
      expect.objectContaining({
        row: expect.objectContaining({ committed: 0, held: 480 }),
        unresolved: 12,
      }),
    );
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
});

test("failures and reports carry identities and counters, never secrets", async () => {
  const { connection, store, reports } = setup(100, 100);
  await configured(store, 100);
  const secretBody = body({
    prompt: "sk-live-topsecret-prompt",
    endpoint: "https://provider.internal/v1",
    usage: { input_tokens: 60, output_tokens: 10 },
  });
  const { send } = countedSend(success("done"), secretBody);
  expect(
    await connection.dispatch({
      context: context("acme", "default", "req-redacted"),
      where: origin,
      operation: OPERATION,
      maxBodyBytes: 8192,
      send,
    }),
  ).toEqual(success("done"));
  const denied = countedSend(success("x"), undefined);
  const failed = await connection.dispatch({
    context: context("acme", "default", "req-denied"),
    where: origin,
    operation: OPERATION,
    maxBodyBytes: 8192,
    send: denied.send,
  });
  expect(failed.kind).toBe("domain");
  const serialized = JSON.stringify(
    {
      reports,
      exceeded: failed.kind === "domain" ? domainFailureDiagnostics(failed.value) : null,
    },
    (_key, value) => (typeof value === "bigint" ? `bigint:${value.toString()}` : value),
  );
  for (const secret of ["sk-live-topsecret-prompt", "provider.internal"])
    expect(serialized).not.toContain(secret);
  // Redacted correlation for H08: every report joins tenant, pool,
  // correlation, invocation, profile, and epoch to its outcome.
  expect(Object.keys(reports[0]).sort()).toEqual([
    "actual",
    "correlation",
    "epochEnd",
    "epochStart",
    "invocation",
    "outcome",
    "pool",
    "profile",
    "released",
    "tenant",
    "upperBound",
  ]);
  expect(reports[0]).toMatchObject({
    tenant: "acme",
    pool: "default",
    correlation: "req-redacted",
    profile: "typesafe/jev/1.13.0",
    upperBound: 100,
    epochStart: 0,
    epochEnd: HOUR,
    outcome: "settled",
    actual: 70,
    released: 30,
  });
});
