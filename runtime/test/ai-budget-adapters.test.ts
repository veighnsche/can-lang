// Guarded native-AI adapter tests (H07): the Responses and SystemOne
// adapters route every bound call through the R14 guard — reserve
// before send, settle authoritative usage after — against a local
// stub. No live provider calls. Guard internals live in
// ai-budget.test.ts.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import {
  createBudgetGuard,
  createProfileRegistry,
  guardConnection,
  type AccountingReport,
  type BudgetTypes,
} from "../ai/budget.ts";
import { createResponses, type ResponseTypes } from "../ai/responses.ts";
import { createTypeSafe, type AITypes, type NoulDescriptor } from "../ai/typesafe.ts";
import {
  concreteTypeDigestInput,
  createDomainRuntime,
  domainFailureDiagnostics,
  type FailureShape,
} from "../domain.ts";
import { dataProperty, record, recordIdentity } from "../data.ts";
import { success, type Completion } from "../completion.ts";
import { runOwnedRoot } from "../owner.ts";
import { epochSchedule } from "../outbound/epoch.ts";
import {
  configurePool,
  createMemoryLedgerStore,
  epochStatus,
  type LedgerStore,
} from "../outbound/ledger.ts";
import { invocationContext } from "../outbound/identity.ts";

const HOUR = 3_600_000;
const PROFILE = { provider: "typesafe", model: "jev", version: "1.13.0" };
const LLM_PROFILE = { provider: "openai", model: "responses", version: "v1" };
const origin = { source: "test:ai-budget-adapters", start: 0, end: 0, invocation: [] };
const OPERATION = "test:ai/guarded";

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
const str = shape("primitive", "str");
const int = shape("primitive", "int");
const header = shape("record", "can.std.http@1::header", [
  { name: "name", type: str.identity },
  { name: "value", type: str.identity },
]);
const headers: FailureShape = {
  ...shape("array", ""),
  identity: hash(["array", "", header.identity]),
  element: header.identity,
};
const wanted = [
  "http::invalid_request",
  "http::credentials_missing",
  "http::transport_failed",
  "http::timeout",
  "http::body_limit",
  "http::status_error",
  "http::request_failed",
  "codec::invalid_data",
  "ai::invalid_question",
  "ai::invalid_answer",
  "llm::refused",
  "llm::truncated",
  "llm::invalid_response",
];
const declarations = catalogue.errors.filter((entry) => wanted.includes(entry.name));
const detailIdentity = hash(["variant", "can.std.http@1::failure_detail"]);
const errors = declarations.map((entry) =>
  shape(
    "error",
    entry.identity,
    entry.fields.map((field) => ({
      name: field.name,
      type:
        field.type === "str"
          ? str.identity
          : field.type === "int"
            ? int.identity
            : field.type === "http::failure_detail"
              ? detailIdentity
              : headers.identity,
    })),
  ),
);
const detail: FailureShape = {
  identity: detailIdentity,
  kind: "variant",
  declaration: "can.std.http@1::failure_detail",
  fields: [],
  arguments: [],
  leaves: errors
    .filter(
      (_, index) =>
        ![
          "http::request_failed",
          "ai::invalid_question",
          "ai::invalid_answer",
          "llm::refused",
          "llm::truncated",
          "llm::invalid_response",
        ].includes(declarations[index].name),
    )
    .map((entry) => entry.identity),
  inputs: [],
  errors: [],
};
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
    ...declarations.map((entry) => ({ ...entry, parameters: 0 })),
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
  shapes: [str, int, header, headers, ...errors, detail, exceededShape, unavailableShape],
});
const byName = new Map(declarations.map((entry, index) => [entry.name, errors[index].identity]));
const http = {
  invalid: byName.get("http::invalid_request")!,
  credential: byName.get("http::credentials_missing")!,
  transport: byName.get("http::transport_failed")!,
  timeout: byName.get("http::timeout")!,
  limit: byName.get("http::body_limit")!,
  status: byName.get("http::status_error")!,
  header: header.identity,
};
const aiTypes: AITypes = {
  ...http,
  invalidData: byName.get("codec::invalid_data")!,
  invalidQuestion: byName.get("ai::invalid_question")!,
  invalidAnswer: byName.get("ai::invalid_answer")!,
  failed: byName.get("http::request_failed")!,
};
const responseTypes: ResponseTypes = {
  ...http,
  invalidData: byName.get("codec::invalid_data")!,
  refused: byName.get("llm::refused")!,
  truncated: byName.get("llm::truncated")!,
  invalidResponse: byName.get("llm::invalid_response")!,
};
const budgetTypes: BudgetTypes = {
  exceeded: exceededShape.identity,
  unavailable: unavailableShape.identity,
};

const schema = {
  root: "state",
  nodes: [
    { identity: "state", kind: "record", name: "state", fields: [{ name: "amount", type: "int" }] },
    { identity: "int", kind: "primitive", name: "int" },
  ],
};
const state = record("state", [["amount", 9007199254740993n]]);
const question: NoulDescriptor = {
  kind: "noul",
  instructions: "Check amount",
  trueDescription: "Yes",
  falseDescription: "No",
  minimum: 0.5,
};

function harness(upperBound = 100, llmBound = 200) {
  const store: LedgerStore = createMemoryLedgerStore("ai-budget-adapters");
  const registry = createProfileRegistry();
  registry.qualify({ ...PROFILE, upperBound });
  registry.qualify({ ...LLM_PROFILE, upperBound: llmBound });
  const reports: AccountingReport[] = [];
  const guard = createBudgetGuard(domain, budgetTypes, {
    store,
    registry,
    nowMs: () => 5,
    report: (entry) => reports.push(entry),
  });
  const judge = guardConnection(guard, { permittedPool: "default", profile: PROFILE });
  const llm = guardConnection(guard, { permittedPool: "default", profile: LLM_PROFILE });
  return { store, registry, reports, guard, judge, llm };
}

async function configured(store: LedgerStore, limit = 1000): Promise<void> {
  const done = await configurePool(
    store,
    epochSchedule({ pool: "default", limit, periodMs: HOUR, anchorMs: 0, version: "1" }),
  );
  expect(done).toEqual({ configured: true });
}

const systemOneBody = (extra: object = {}) => ({
  model: "resolved",
  usage: { input_tokens: 40, output_tokens: 10 },
  answers: { q0: { type: "noul", noul: 0.5 } },
  ...extra,
});
const responsesBody = (extra: object = {}) => ({
  id: "response-1",
  object: "response",
  status: "completed",
  output: [
    {
      type: "message",
      role: "assistant",
      status: "completed",
      content: [{ type: "output_text", text: "hello" }],
    },
  ],
  usage: { input_tokens: 120, output_tokens: 30 },
  ...extra,
});

function stubServer(reply: () => { status: number; json: unknown }) {
  const seen: { method: string; path: string; body: string }[] = [];
  const server = Bun.serve({
    hostname: "127.0.0.1",
    port: 0,
    async fetch(request) {
      const { status, json } = reply();
      seen.push({
        method: request.method,
        path: new URL(request.url).pathname,
        body: await request.text(),
      });
      return Response.json(json, { status });
    },
  });
  return {
    server,
    seen,
    url: (path: string): string => new URL(path, server.url).href,
    [Symbol.asyncDispose]: async (): Promise<void> => {
      await server.stop(true);
    },
  };
}

function checkName(result: Completion<unknown>, name: string): Record<string, unknown> {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain failure");
  const details = domainFailureDiagnostics(result.value);
  expect(details.declaration.name).toBe(name);
  return details.payload as Record<string, unknown>;
}

test("guarded SystemOne settles authoritative usage and reports", async () => {
  const { store, reports, judge } = harness();
  await configured(store);
  await using api = stubServer(() => ({ status: 200, json: systemOneBody() }));
  const guarded = createTypeSafe(domain, aiTypes, () => "fixture-secret", judge);
  const root = await runOwnedRoot(async () => {
    const result = await guarded.ask(
      {
        endpoint: api.url("/systemone"),
        timeoutMilliseconds: 1000,
        maxBodyBytes: 8192,
        headers: [],
      },
      "jev-latest",
      schema,
      state,
      [question],
      origin,
      OPERATION,
      undefined,
      invocationContext("acme", "default", "req-judge"),
    );
    expect(result.kind).toBe("ok");
    if (result.kind !== "ok") throw Error("expected answers");
    expect(result.value).toEqual([{ kind: "noul", probability: 0.5 }]);
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
  expect(api.seen).toHaveLength(1);
  expect(reports).toEqual([
    expect.objectContaining({
      tenant: "acme",
      pool: "default",
      correlation: "req-judge",
      profile: "typesafe/jev/1.13.0",
      upperBound: 100,
      outcome: "settled",
      actual: 50,
      released: 50,
    }),
  ]);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 50, held: 0 }) }),
  );
});

test("guarded Responses settles usage from the envelope", async () => {
  const { store, reports, llm } = harness();
  await configured(store);
  await using api = stubServer(() => ({ status: 200, json: responsesBody() }));
  const guarded = createResponses(domain, responseTypes, () => "fixture-secret", llm);
  const root = await runOwnedRoot(async () => {
    const result = await guarded.generate<string>(
      {
        endpoint: api.url("/responses"),
        timeoutMilliseconds: 1000,
        maxBodyBytes: 65536,
        headers: [],
      },
      "explicit-model",
      2048,
      "triage instructions",
      schema,
      record("state", [["amount", 1n]]),
      undefined,
      origin,
      OPERATION,
      undefined,
      invocationContext("acme", "default", "req-llm"),
    );
    expect(result).toEqual(success("hello"));
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
  expect(api.seen).toHaveLength(1);
  expect(reports).toEqual([
    expect.objectContaining({
      correlation: "req-llm",
      outcome: "settled",
      actual: 150,
      released: 50,
    }),
  ]);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 150, held: 0 }) }),
  );
});

test("guarded request bytes match unguarded bytes exactly", async () => {
  const { store, judge, llm } = harness();
  await configured(store);
  await using api = stubServer(() => ({ status: 200, json: systemOneBody() }));
  const guarded = createTypeSafe(domain, aiTypes, () => "fixture-secret", judge);
  const plain = createTypeSafe(domain, aiTypes, () => "fixture-secret");
  const connection = {
    endpoint: api.url("/systemone"),
    timeoutMilliseconds: 1000,
    maxBodyBytes: 8192,
    headers: [],
  };
  const root = await runOwnedRoot(async () => {
    const scope = invocationContext("acme", "default", "req-bytes");
    const first = await guarded.ask(
      connection,
      "jev-latest",
      schema,
      state,
      [question],
      origin,
      OPERATION,
      undefined,
      scope,
    );
    const second = await plain.ask(
      connection,
      "jev-latest",
      schema,
      state,
      [question],
      origin,
      OPERATION,
    );
    expect(first.kind).toBe("ok");
    expect(second.kind).toBe("ok");
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
  expect(api.seen).toHaveLength(2);
  expect(api.seen[0].body).toBe(api.seen[1].body);
  expect(api.seen[0].method).toBe("POST");
  await using llmApi = stubServer(() => ({ status: 200, json: responsesBody() }));
  const guardedLlm = createResponses(domain, responseTypes, () => "fixture-secret", llm);
  const plainLlm = createResponses(domain, responseTypes, () => "fixture-secret");
  const llmConnection = {
    endpoint: llmApi.url("/responses"),
    timeoutMilliseconds: 1000,
    maxBodyBytes: 65536,
    headers: [],
  };
  const llmRoot = await runOwnedRoot(async () => {
    const args = [
      "explicit-model",
      2048,
      "triage instructions",
      schema,
      record("state", [["amount", 1n]]),
      undefined,
      origin,
      OPERATION,
    ] as const;
    const first = await guardedLlm.generate<string>(
      llmConnection,
      ...args,
      undefined,
      invocationContext("acme", "default", "req-llm-bytes"),
    );
    const second = await plainLlm.generate<string>(llmConnection, ...args);
    expect(first.kind).toBe("ok");
    expect(second.kind).toBe("ok");
    return success(undefined);
  });
  expect(llmRoot.completion.kind).toBe("ok");
  expect(llmApi.seen).toHaveLength(2);
  expect(llmApi.seen[0].body).toBe(llmApi.seen[1].body);
  // Nonstreaming scope is preserved on the guarded path.
  expect(JSON.parse(llmApi.seen[0].body)).toMatchObject({
    stream: false,
    store: false,
    background: false,
  });
});

test("rejected guarded calls never dispatch and keep typed failures", async () => {
  const { store, judge } = harness(100);
  await configured(store, 100);
  await using api = stubServer(() => ({ status: 200, json: systemOneBody() }));
  const guarded = createTypeSafe(domain, aiTypes, () => "fixture-secret", judge);
  const connection = {
    endpoint: api.url("/systemone"),
    timeoutMilliseconds: 1000,
    maxBodyBytes: 8192,
    headers: [],
  };
  const root = await runOwnedRoot(async () => {
    // Missing scope context: the bound adapter has no bypass.
    const bypass = await guarded.ask(
      connection,
      "jev-latest",
      schema,
      state,
      [question],
      origin,
      OPERATION,
    );
    const missing = checkName(bypass, "can.std.ai_budget@1::unavailable");
    expect(missing).toMatchObject({ reason: "missing-context" });
    // Pool mismatch cannot create fresh allowance.
    const pool = await guarded.ask(
      connection,
      "jev-latest",
      schema,
      state,
      [question],
      origin,
      OPERATION,
      undefined,
      invocationContext("acme", "other", "req-pool"),
    );
    expect(checkName(pool, "can.std.ai_budget@1::unavailable")).toMatchObject({
      reason: "invalid-policy",
    });
    // One exact-fit admission consumes the epoch.
    const admitted = await guarded.ask(
      connection,
      "jev-latest",
      schema,
      state,
      [question],
      origin,
      OPERATION,
      undefined,
      invocationContext("acme", "default", "req-fit"),
    );
    expect(admitted.kind).toBe("ok");
    // Excess rejects immediately with detail, unmapped by the normalizer.
    const excess = await guarded.ask(
      connection,
      "jev-latest",
      schema,
      state,
      [question],
      origin,
      OPERATION,
      undefined,
      invocationContext("acme", "default", "req-excess"),
    );
    expect(checkName(excess, "can.std.ai_budget@1::exceeded")).toMatchObject({
      pool: "default",
      required: 100n,
      remaining: 50n,
      epoch_reset_ms: BigInt(HOUR),
    });
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
  expect(api.seen).toHaveLength(1);
  expect(api.seen[0].path).toBe("/systemone");
});

test("unqualified guarded connections reject before send", async () => {
  const store = createMemoryLedgerStore("ai-budget-adapters-unqualified");
  await configured(store);
  const guard = createBudgetGuard(domain, budgetTypes, {
    store,
    registry: createProfileRegistry(),
    nowMs: () => 5,
  });
  const binding = guardConnection(guard, { permittedPool: "default", profile: PROFILE });
  await using api = stubServer(() => ({ status: 200, json: systemOneBody() }));
  const guarded = createTypeSafe(domain, aiTypes, () => "fixture-secret", binding);
  const root = await runOwnedRoot(async () => {
    const result = await guarded.ask(
      {
        endpoint: api.url("/systemone"),
        timeoutMilliseconds: 1000,
        maxBodyBytes: 8192,
        headers: [],
      },
      "jev-latest",
      schema,
      state,
      [question],
      origin,
      OPERATION,
      undefined,
      invocationContext("acme", "default", "req-unqualified"),
    );
    expect(checkName(result, "can.std.ai_budget@1::unavailable")).toMatchObject({
      reason: "missing-qualification",
      correlation: "req-unqualified",
    });
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
  expect(api.seen).toHaveLength(0);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual({ status: "absent" });
});

test("status failures keep the provider completion and the full hold", async () => {
  const { store, reports, judge } = harness();
  await configured(store);
  await using api = stubServer(() => ({ status: 500, json: { error: "provider down" } }));
  const guarded = createTypeSafe(domain, aiTypes, () => "fixture-secret", judge);
  const root = await runOwnedRoot(async () => {
    const result = await guarded.ask(
      {
        endpoint: api.url("/systemone"),
        timeoutMilliseconds: 1000,
        maxBodyBytes: 8192,
        headers: [],
      },
      "jev-latest",
      schema,
      state,
      [question],
      origin,
      OPERATION,
      undefined,
      invocationContext("acme", "default", "req-status"),
    );
    // The existing normalized provider failure stays intact.
    expect(result.kind).toBe("domain");
    if (result.kind !== "domain") throw Error("expected domain failure");
    const details = domainFailureDiagnostics(result.value);
    expect(details.declaration.name).toBe("http::request_failed");
    const leaf = dataProperty(details.payload, "detail");
    expect(recordIdentity(leaf)).toBe(byName.get("http::status_error"));
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
  expect(api.seen).toHaveLength(1);
  expect(reports).toEqual([
    expect.objectContaining({ correlation: "req-status", outcome: "unresolved" }),
  ]);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({
      row: expect.objectContaining({ committed: 0, held: 100 }),
      unresolved: 1,
    }),
  );
});

test("invalid answers with valid usage settle while failing", async () => {
  const { store, reports, judge } = harness();
  await configured(store);
  await using api = stubServer(() => ({
    status: 200,
    json: systemOneBody({ answers: {} }),
  }));
  const guarded = createTypeSafe(domain, aiTypes, () => "fixture-secret", judge);
  const root = await runOwnedRoot(async () => {
    const result = await guarded.ask(
      {
        endpoint: api.url("/systemone"),
        timeoutMilliseconds: 1000,
        maxBodyBytes: 8192,
        headers: [],
      },
      "jev-latest",
      schema,
      state,
      [question],
      origin,
      OPERATION,
      undefined,
      invocationContext("acme", "default", "req-invalid"),
    );
    expect(checkName(result, "ai::invalid_answer")).toMatchObject({
      question: "",
      reason: "question_ids",
    });
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
  expect(reports).toEqual([
    expect.objectContaining({
      correlation: "req-invalid",
      outcome: "settled",
      actual: 50,
      released: 50,
    }),
  ]);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 50, held: 0 }) }),
  );
});

test("refusal and truncation with usage settle while failing", async () => {
  const { store, reports, llm } = harness();
  await configured(store);
  const refusal = {
    id: "response-2",
    object: "response",
    status: "completed",
    output: [
      {
        type: "message",
        role: "assistant",
        status: "completed",
        content: [{ type: "refusal", refusal: "no" }],
      },
    ],
    usage: { input_tokens: 100, output_tokens: 5 },
  };
  await using refused = stubServer(() => ({ status: 200, json: refusal }));
  const guarded = createResponses(domain, responseTypes, () => "fixture-secret", llm);
  const connection = {
    endpoint: refused.url("/responses"),
    timeoutMilliseconds: 1000,
    maxBodyBytes: 65536,
    headers: [],
  };
  const args = [
    "explicit-model",
    2048,
    "triage instructions",
    schema,
    record("state", [["amount", 1n]]),
    undefined,
    origin,
    OPERATION,
  ] as const;
  const first = await runOwnedRoot(async () => {
    const result = await guarded.generate<string>(
      connection,
      ...args,
      undefined,
      invocationContext("acme", "default", "req-refused"),
    );
    expect(checkName(result, "llm::refused")).toMatchObject({ reason: "provider_refusal" });
    return success(undefined);
  });
  expect(first.completion.kind).toBe("ok");
  const truncated = {
    id: "response-3",
    object: "response",
    status: "incomplete",
    incomplete_details: { reason: "max_output_tokens" },
    output: [],
    usage: { input_tokens: 150, output_tokens: 40 },
  };
  await using cut = stubServer(() => ({ status: 200, json: truncated }));
  const second = await runOwnedRoot(async () => {
    const result = await guarded.generate<string>(
      { ...connection, endpoint: cut.url("/responses") },
      ...args,
      undefined,
      invocationContext("acme", "default", "req-truncated"),
    );
    expect(checkName(result, "llm::truncated")).toMatchObject({});
    return success(undefined);
  });
  expect(second.completion.kind).toBe("ok");
  expect(reports).toEqual([
    expect.objectContaining({
      correlation: "req-refused",
      outcome: "settled",
      actual: 105,
      released: 95,
    }),
    expect.objectContaining({
      correlation: "req-truncated",
      outcome: "settled",
      actual: 190,
      released: 10,
    }),
  ]);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 295, held: 0 }) }),
  );
});

test("breach over the stub invalidates a successful guarded call", async () => {
  const { store, reports, judge } = harness(100);
  await configured(store);
  await using api = stubServer(() => ({
    status: 200,
    json: systemOneBody({ usage: { input_tokens: 90, output_tokens: 20 } }),
  }));
  const guarded = createTypeSafe(domain, aiTypes, () => "fixture-secret", judge);
  const root = await runOwnedRoot(async () => {
    const result = await guarded.ask(
      {
        endpoint: api.url("/systemone"),
        timeoutMilliseconds: 1000,
        maxBodyBytes: 8192,
        headers: [],
      },
      "jev-latest",
      schema,
      state,
      [question],
      origin,
      OPERATION,
      undefined,
      invocationContext("acme", "default", "req-breach"),
    );
    expect(checkName(result, "can.std.ai_budget@1::unavailable")).toMatchObject({
      reason: "breached-profile",
      correlation: "req-breach",
    });
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
  expect(api.seen).toHaveLength(1);
  expect(reports).toEqual([expect.objectContaining({ outcome: "breach", actual: 110 })]);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({ row: expect.objectContaining({ committed: 110, held: 0 }) }),
  );
});

test("missing usage succeeds with an unresolved hold", async () => {
  const { store, reports, judge } = harness();
  await configured(store);
  const { usage: _dropped, ...noUsage } = systemOneBody();
  void _dropped;
  await using api = stubServer(() => ({ status: 200, json: noUsage }));
  const guarded = createTypeSafe(domain, aiTypes, () => "fixture-secret", judge);
  const root = await runOwnedRoot(async () => {
    const result = await guarded.ask(
      {
        endpoint: api.url("/systemone"),
        timeoutMilliseconds: 1000,
        maxBodyBytes: 8192,
        headers: [],
      },
      "jev-latest",
      schema,
      state,
      [question],
      origin,
      OPERATION,
      undefined,
      invocationContext("acme", "default", "req-nometering"),
    );
    expect(result.kind).toBe("ok");
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
  expect(reports).toEqual([
    expect.objectContaining({ correlation: "req-nometering", outcome: "unresolved" }),
  ]);
  expect(await epochStatus(store, "acme", "default", 0)).toEqual(
    expect.objectContaining({
      row: expect.objectContaining({ committed: 0, held: 100 }),
      unresolved: 1,
    }),
  );
});

test("guarded calls isolate tenants and encode failures never reserve", async () => {
  const { store, judge } = harness(100);
  await configured(store, 100);
  await using api = stubServer(() => ({ status: 200, json: systemOneBody() }));
  const guarded = createTypeSafe(domain, aiTypes, () => "fixture-secret", judge);
  const connection = {
    endpoint: api.url("/systemone"),
    timeoutMilliseconds: 1000,
    maxBodyBytes: 8192,
    headers: [],
  };
  const root = await runOwnedRoot(async () => {
    // Invalid questions fail before admission: no reservation exists.
    const invalid = await guarded.ask(
      connection,
      "jev-latest",
      schema,
      state,
      [{ ...question, instructions: "" }],
      origin,
      OPERATION,
      undefined,
      invocationContext("nobody", "default", "req-invalid"),
    );
    expect(checkName(invalid, "ai::invalid_question")).toMatchObject({ reason: "instructions" });
    for (const tenant of ["a", "b"]) {
      const result = await guarded.ask(
        connection,
        "jev-latest",
        schema,
        state,
        [question],
        origin,
        OPERATION,
        undefined,
        invocationContext(tenant, "default", `req-${tenant}`),
      );
      expect(result.kind).toBe("ok");
    }
    return success(undefined);
  });
  expect(root.completion.kind).toBe("ok");
  expect(api.seen).toHaveLength(2);
  expect(await epochStatus(store, "nobody", "default", 0)).toEqual({ status: "absent" });
  for (const tenant of ["a", "b"])
    expect(await epochStatus(store, tenant, "default", 0)).toEqual(
      expect.objectContaining({ row: expect.objectContaining({ committed: 50, held: 0 }) }),
    );
});

test("within scopes a guarded call end to end", async () => {
  const { store, reports, guard, judge } = harness();
  await configured(store);
  await using api = stubServer(() => ({ status: 200, json: systemOneBody() }));
  const guarded = createTypeSafe(domain, aiTypes, () => "fixture-secret", judge);
  const connection = {
    endpoint: api.url("/systemone"),
    timeoutMilliseconds: 1000,
    maxBodyBytes: 8192,
    headers: [],
  };
  const root = await runOwnedRoot(() =>
    guard.within(
      "acme",
      "default",
      async (scope) => {
        const result = await guarded.ask(
          connection,
          "jev-latest",
          schema,
          state,
          [question],
          origin,
          OPERATION,
          undefined,
          scope,
        );
        if (result.kind !== "ok") return result;
        return success(
          result.value.map((answer) => (answer.kind === "noul" ? answer.probability : -1)),
        );
      },
      origin,
      OPERATION,
    ),
  );
  expect(root.completion).toEqual(success([0.5]));
  expect(api.seen).toHaveLength(1);
  expect(reports).toEqual([
    expect.objectContaining({ tenant: "acme", outcome: "settled", actual: 50 }),
  ]);
});
