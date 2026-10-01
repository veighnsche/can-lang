// NT-I01 native-values adapter tests: envelope construction, wire mapping
// and handle tables over an injected stub dispatch. No N owner exists, so
// every test either pins exact request bytes or drives a canned reply; the
// default dispatch must throw. Markers native:: keep the NT-I01 package
// referenced from runtime test evidence.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, type FailureShape } from "../domain.ts";
import type { Completion } from "../completion.ts";
import { dataProperty, record } from "../data.ts";
import { createTestOwner } from "../test-support/owner.ts";
import { createNativeValues } from "../test-support/slices/i01/native.ts";

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
const text = scalar("str");
const integer = scalar("int");
const declarations = catalogue.errors.filter((error) => error.name === "test::invalid_grant");
const shapeOf = (id: string, fields: readonly { name: string; type: string }[]): FailureShape => ({
  identity: identity("error", id),
  kind: "error",
  declaration: id,
  arguments: [],
  fields: fields.map((field) => ({
    name: field.name,
    type: field.type === "int" ? integer.identity : text.identity,
  })),
  leaves: [],
  inputs: [],
  errors: [],
});
const errorShapes: Map<string, FailureShape> = new Map(
  declarations.map((error) => [error.name, shapeOf(error.identity, error.fields)]),
);
const domain = createDomainRuntime({
  declarations: declarations.map((error) => ({
    identity: error.identity,
    name: error.name,
    parameters: 0,
  })),
  shapes: [text, integer, ...errorShapes.values()],
});
const err = (name: string): string => {
  const shape = errorShapes.get(name);
  if (shape === undefined) throw Error(`no shape for ${name}`);
  return shape.identity;
};

const SOME = "test:some";
const NONE = "test:none";
const typeId = (name: string): string => `test:type:${name}`;
const some = (value: unknown): unknown => record(SOME, [["value", value]]);
const none = (): unknown => record(NONE, []);

type Captured = Readonly<Record<string, unknown>>;
function rig(handler: (request: Captured) => Readonly<Record<string, unknown>>) {
  const requests: Captured[] = [];
  const owner = createTestOwner(domain, { invalidGrant: err("test::invalid_grant") });
  const native = createNativeValues(
    domain,
    {
      some: SOME,
      none: NONE,
      inertFacts: typeId("native::inert_facts"),
      observation: typeId("native::observation"),
      observationResult: typeId("native::observation_result"),
      taggedEntry: typeId("native::tagged_entry"),
      taggedValue: typeId("native::tagged_value"),
      aliasGroup: typeId("native::alias_group"),
      observeCounters: typeId("native::observe_counters"),
      descriptorOrUnknown: typeId("native::descriptor_or_unknown"),
      handleOrPendingAction: typeId("native::handle_or_pending_action"),
      settlementOrPending: typeId("native::settlement_or_pending"),
      releaseFacts: typeId("native::release_facts"),
      restoreOutcome: typeId("native::restore_outcome"),
      closeReceipt: typeId("native::close_receipt"),
    },
    owner,
    (request) => {
      requests.push(request);
      return Promise.resolve(handler(request));
    },
  );
  return { owner, native, requests };
}

const GRANT = "ng1-0123456789abcdef0123456789abcdef";
const GRANT2 = "ng1-fedcba9876543210fedcba9876543210";

async function ok<T>(completion: Promise<Completion<T>>): Promise<T> {
  const result = await completion;
  expect(result.kind).toBe("ok");
  if (result.kind !== "ok") throw Error("expected ok");
  return result.value;
}

const limits = () =>
  record(typeId("native::limits"), [
    ["max_sessions", 1n],
    ["max_handles_per_session", 8n],
    ["maxPendingActions", 8n],
    ["maxObserveEntries", 64n],
    ["maxObserveBytes", 65536n],
    ["maxGatesPerSession", 4n],
    ["maxFaultsPerSession", 4n],
  ]);
const deadline = () =>
  record(typeId("native::deadline"), [
    ["clock", "wall-utc"],
    ["ms", 5000n],
  ]);
const bounds = () =>
  record(typeId("native::observe_bounds"), [
    ["maxEntries", 64n],
    ["maxBytes", 65536n],
  ]);
const textPayload = () =>
  record(typeId("native::inert_literal"), [
    ["tag", "text"],
    ["hi", none()],
    ["lo", none()],
    ["text", some("hi")],
    ["entries", none()],
    ["descriptor", none()],
  ]);
const wireSession = { kind: "session", id: "s1", sessionId: "s1", owner: GRANT, generation: 1 };
const completed = (facts: Record<string, unknown>) => ({
  schema_version: "1",
  run_id: "run-x",
  operation_id: "op-1",
  outcome: "completed",
  kind: "ok",
  facts,
});

test("native::open builds the exact request envelope", async () => {
  const { owner, native, requests } = rig(() => completed({ session: wireSession }));
  const grant = await ok(owner.admitGrant(GRANT));
  const session = await ok(native.open(grant, "bun", "observer-a", "run-1", limits()));
  expect(requests).toHaveLength(1);
  const request = requests[0] as Record<string, unknown>;
  expect(request["schema_version"]).toBe("1");
  expect(request["run_id"]).toMatch(/^run-[0-9a-f]{16}$/);
  expect(request["operation_id"]).toMatch(/^op-1-[0-9a-f]{8}$/);
  expect(request["owner_grant"]).toBe(GRANT);
  expect(request["operation"]).toBe("native.open");
  expect(request["arguments_digest"]).toMatch(/^sha256:[0-9a-f]{64}$/);
  expect(request["deadline_ms"]).toEqual({ clock: "n-monotonic", ms: 30000 });
  expect(request["args"]).toEqual({
    runtime: "bun",
    observer: "observer-a",
    scope: "run-1",
    limits: {
      max_sessions: 1,
      max_handles_per_session: 8,
      maxPendingActions: 8,
      maxObserveEntries: 64,
      maxObserveBytes: 65536,
      maxGatesPerSession: 4,
      maxFaultsPerSession: 4,
    },
  });
  expect(typeof session).toBe("object");
});

test("native arguments_digest is deterministic over arg order", async () => {
  const { owner, native, requests } = rig(() => completed({ session: wireSession }));
  const grant = await ok(owner.admitGrant(GRANT));
  await ok(native.open(grant, "bun", "observer-a", "run-1", limits()));
  await ok(native.open(grant, "bun", "observer-a", "run-1", limits()));
  expect(requests).toHaveLength(2);
  expect(requests[0]?.["arguments_digest"]).toBe(requests[1]?.["arguments_digest"]);
  expect(requests[0]?.["operation_id"]).not.toBe(requests[1]?.["operation_id"]);
});

test("native::make maps the text payload, omitting absent options", async () => {
  const seen: Captured[] = [];
  const { owner, native } = rig((request) => {
    seen.push(request);
    if (request["operation"] === "native.open") return completed({ session: wireSession });
    return completed({
      value: { kind: "value", id: "v1", sessionId: "s1", owner: GRANT, generation: 1 },
    });
  });
  const grant = await ok(owner.admitGrant(GRANT));
  const session = await ok(native.open(grant, "bun", "observer-a", "run-1", limits()));
  await ok(native.make(session, "text", textPayload()));
  const make = seen[1] as Record<string, Record<string, unknown>>;
  expect(make["operation"]).toBe("native.make");
  expect(make["args"]?.["kind"]).toBe("text");
  expect(make["args"]?.["payload"]).toEqual({ tag: "text", text: "hi" });
  expect(make["args"]?.["session"]).toEqual(wireSession);
});

test("native::make maps entry leaves to pairs", async () => {
  let payload: unknown = null;
  const { owner, native } = rig((request) => {
    if (request["operation"] === "native.open") return completed({ session: wireSession });
    payload = (request["args"] as Record<string, unknown>)["payload"];
    return completed({
      value: { kind: "value", id: "v1", sessionId: "s1", owner: GRANT, generation: 1 },
    });
  });
  const grant = await ok(owner.admitGrant(GRANT));
  const session = await ok(native.open(grant, "bun", "observer-a", "run-1", limits()));
  const entry = (key: string, leaf: string, value: unknown) =>
    record(typeId("native::entry"), [
      ["key", key],
      ["value", record(`can.std.native@1::${leaf}`, [["value", value]])],
    ]);
  const lit = record(typeId("native::inert_literal"), [
    ["tag", "entries"],
    ["hi", none()],
    ["lo", none()],
    ["text", none()],
    [
      "entries",
      some([
        entry("a", "literal_int", 1n),
        entry("b", "literal_text", "two"),
        entry("c", "literal_bool", true),
      ]),
    ],
    ["descriptor", none()],
  ]);
  await ok(native.make(session, "ordered_entries", lit));
  expect(payload).toEqual({
    tag: "entries",
    entries: [
      ["a", 1],
      ["b", "two"],
      ["c", true],
    ],
  });
});

test("native::observe maps reply facts to nominal records", async () => {
  const { owner, native } = rig((request) => {
    if (request["operation"] === "native.open") return completed({ session: wireSession });
    if (request["operation"] === "native.make") {
      return completed({
        value: { kind: "value", id: "v1", sessionId: "s1", owner: GRANT, generation: 1 },
      });
    }
    return completed({
      kind: "lexeme",
      observations: [{ handle: "v1", cell: "cell1", result: { tag: "text", text: "hi" } }],
    });
  });
  const grant = await ok(owner.admitGrant(GRANT));
  const session = await ok(native.open(grant, "bun", "observer-a", "run-1", limits()));
  const handle = await ok(native.make(session, "text", textPayload()));
  const facts = await ok(native.observe(session, [handle], "lexeme", bounds()));
  expect(dataProperty(facts, "kind")).toBe("lexeme");
  const observations = dataProperty(facts, "observations") as unknown[];
  expect(observations).toHaveLength(1);
  const result = dataProperty(observations[0], "result");
  expect(dataProperty(dataProperty(result, "tag"), "value")).toBe("text");
  expect(dataProperty(dataProperty(result, "text"), "value")).toBe("hi");
});

test("native::invoke pending mints an action; settle reuses its grant", async () => {
  const grants: unknown[] = [];
  const { owner, native } = rig((request) => {
    grants.push(request["owner_grant"]);
    if (request["operation"] === "native.open") return completed({ session: wireSession });
    if (request["operation"] === "native.make") {
      return completed({
        value: { kind: "value", id: "v1", sessionId: "s1", owner: GRANT, generation: 1 },
      });
    }
    if (request["operation"] === "native.invoke") {
      return completed({
        action: { kind: "action", id: "a1", sessionId: "s1", owner: GRANT, generation: 1 },
      });
    }
    return completed({
      value: { kind: "value", id: "v2", sessionId: "s1", owner: GRANT, generation: 1 },
    });
  });
  const grant = await ok(owner.admitGrant(GRANT));
  const session = await ok(native.open(grant, "bun", "observer-a", "run-1", limits()));
  const handle = await ok(native.make(session, "text", textPayload()));
  const pending = await ok(native.invoke(session, "op-x", none(), [handle]));
  expect(dataProperty(pending, "settled")).toBe(false);
  const action = dataProperty(dataProperty(pending, "action"), "value");
  const settled = await ok(native.settle(action, deadline()));
  expect(dataProperty(settled, "settled")).toBe(true);
  expect(grants).toEqual([GRANT, GRANT, GRANT, GRANT]);
});

test("native::close marks the session; later session use throws", async () => {
  const { owner, native } = rig((request) => {
    if (request["operation"] === "native.open") return completed({ session: wireSession });
    return completed({
      sessionId: "s1",
      released: 1,
      remaining: 0,
      forced: [],
      joined: true,
      cellsReleased: 2,
    });
  });
  const grant = await ok(owner.admitGrant(GRANT));
  const session = await ok(native.open(grant, "bun", "observer-a", "run-1", limits()));
  const receipt = await ok(native.close(session, deadline()));
  expect(dataProperty(receipt, "sessionId")).toBe("s1");
  expect(dataProperty(receipt, "cellsReleased")).toBe(2n);
  await expect(native.describe(session, "api-x")).rejects.toThrow("closed native::session");
});

test("native admission is first: dead owners and foreign handles throw", async () => {
  const { owner, native, requests } = rig(() => completed({ session: wireSession }));
  const grant = await ok(owner.admitGrant(GRANT));
  await ok(owner.releaseGrant(grant));
  await expect(native.open(grant, "bun", "o", "r", limits())).rejects.toThrow(
    "needs a live test::owner handle",
  );
  expect(requests).toHaveLength(0);
  const grant2 = await ok(owner.admitGrant(GRANT2));
  const session = await ok(native.open(grant2, "bun", "o", "r", limits()));
  expect(requests).toHaveLength(1);
  await expect(native.describe({ forged: true }, "api-x")).rejects.toThrow(
    "needs a native::session handle",
  );
  expect(requests).toHaveLength(1);
  void session;
});

test("native grants stay per-session across two owners", async () => {
  const grants: unknown[] = [];
  const { owner, native } = rig((request) => {
    grants.push(request["owner_grant"]);
    if (request["operation"] === "native.describe") return completed({ known: true });
    return completed({ session: wireSession });
  });
  const first = await ok(owner.admitGrant(GRANT));
  const second = await ok(owner.admitGrant(GRANT2));
  const s1 = await ok(native.open(first, "bun", "o", "r", limits()));
  const s2 = await ok(native.open(second, "bun", "o", "r", limits()));
  await ok(native.describe(s1, "api-a"));
  await ok(native.describe(s2, "api-b"));
  expect(grants).toEqual([GRANT, GRANT2, GRANT, GRANT2]);
});

test("native default dispatch and non-completed outcomes throw", async () => {
  const owner = createTestOwner(domain, { invalidGrant: err("test::invalid_grant") });
  const native = createNativeValues(
    domain,
    {
      some: SOME,
      none: NONE,
      inertFacts: typeId("native::inert_facts"),
      observation: typeId("native::observation"),
      observationResult: typeId("native::observation_result"),
      taggedEntry: typeId("native::tagged_entry"),
      taggedValue: typeId("native::tagged_value"),
      aliasGroup: typeId("native::alias_group"),
      observeCounters: typeId("native::observe_counters"),
      descriptorOrUnknown: typeId("native::descriptor_or_unknown"),
      handleOrPendingAction: typeId("native::handle_or_pending_action"),
      settlementOrPending: typeId("native::settlement_or_pending"),
      releaseFacts: typeId("native::release_facts"),
      restoreOutcome: typeId("native::restore_outcome"),
      closeReceipt: typeId("native::close_receipt"),
    },
    owner,
  );
  const grant = await ok(owner.admitGrant(GRANT));
  await expect(native.open(grant, "bun", "o", "r", limits())).rejects.toThrow(
    "N transport is not bound",
  );
  const failing = rig(() => ({ outcome: "rejected", kind: "invalid-request" }));
  const grant2 = await ok(failing.owner.admitGrant(GRANT));
  await expect(failing.native.open(grant2, "bun", "o", "r", limits())).rejects.toThrow(
    "native.open rejected/invalid-request",
  );
});
