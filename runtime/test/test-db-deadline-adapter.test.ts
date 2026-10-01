// NT-I16 deadline-observer adapter contract: the 17 operations
// drive one adapter-held K26 service with owner admission first;
// facts map to nominal records with snake_case fields
// (deadline_ms as bigint, tokens and grants as opaque strings);
// revoked owners fail test::stale_handle without touching the
// service; and every service rejection maps VERBATIM to
// db::db_fault{layer, code} so K26 verdicts (double dispatch,
// missing deadline, forged grants, unknown-effect cleanup
// blocks) stay visible to Can rows instead of collapsing into
// stale_handle.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import type { Completion } from "../completion.ts";
import { dataArray, dataProperty } from "../data.ts";
import { createTestOwner } from "../test-support/owner.ts";
import { createDbDeadline } from "../test-support/slices/i16/db_deadline.ts";

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
const declarations = catalogue.errors.filter(
  (error) => error.name === "test::stale_handle" || error.name === "db::db_fault",
);
const shapeOf = (id: string, fields: readonly { name: string; type: string }[]): FailureShape => ({
  identity: identity("error", id),
  kind: "error",
  declaration: id,
  arguments: [],
  fields: fields.map((field) => ({ name: field.name, type: text.identity })),
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
  shapes: [text, ...errorShapes.values()],
});
const err = (name: string): string => {
  const shape = errorShapes.get(name);
  if (shape === undefined) throw Error(`no shape for ${name}`);
  return shape.identity;
};

const typeId = (name: string): string => `test:type:${name}`;

function rig() {
  const owner = createTestOwner(domain, { invalidGrant: err("test::stale_handle") });
  const db = createDbDeadline(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      dbFault: err("db::db_fault"),
      dispatchedWork: typeId("db::dispatched_work"),
      dispatchRecord: typeId("db::dispatch_record"),
      deadlineRecord: typeId("db::deadline_record"),
      driverSettlementFacts: typeId("db::driver_settlement_facts"),
      serverAckFacts: typeId("db::server_ack_facts"),
      cancelGrant: typeId("db::cancel_grant"),
      cancelGrantFacts: typeId("db::cancel_grant_facts"),
      cancelRecord: typeId("db::cancel_record"),
      fenceRecord: typeId("db::fence_record"),
      leaseRecord: typeId("db::lease_record"),
      deadlineReleaseRecord: typeId("db::deadline_release_record"),
    },
    owner,
  );
  return { owner, db };
}

const GRANT = "ng1-0123456789abcdef0123456789abcdef";

async function ok<T>(completion: Promise<Completion<T>>): Promise<T> {
  const result = await completion;
  expect(result.kind).toBe("ok");
  if (result.kind !== "ok") throw Error("expected ok");
  return result.value;
}

async function failed(completion: Promise<Completion<never>>) {
  const result = await completion;
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain failure");
  return domainFailureDiagnostics(result.value);
}

const field = (value: unknown, name: string): unknown => dataProperty(value, name);

async function bootstrap() {
  const { owner, db } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const sent = await ok(db.dispatch(handle, "work-0", "SELECT 1", "postgres"));
  const facts = field(sent, "facts");
  return { owner, db, handle, token: field(sent, "token") as string, facts };
}

test("db dispatch pins facts and a single-use token", async () => {
  const { db, handle, token, facts } = await bootstrap();
  expect(field(facts, "work")).toBe("work-0");
  expect(field(facts, "engine")).toBe("postgres");
  expect(field(facts, "namespace")).toBe("ns-alpha");
  expect(String(field(facts, "statement_digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  expect(String(field(facts, "handle_digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  expect(typeof token).toBe("string");
  const reread = await ok(db.dispatchFacts(handle, "work-0"));
  expect(field(reread, "handle_digest")).toBe(field(facts, "handle_digest"));
  const twice = await failed(
    db.dispatch(handle, "work-0", "SELECT 1", "postgres") as Promise<Completion<never>>,
  );
  expect(twice.declaration.name).toBe("db::db_fault");
  expect(field(twice.payload, "code")).toBe("connection-busy");
  const badEngine = await failed(
    db.dispatch(handle, "work-1", "SELECT 1", "oracle") as Promise<Completion<never>>,
  );
  expect(field(badEngine.payload, "code")).toBe("malformed-statement");
  const unknown = await failed(db.dispatchFacts(handle, "work-9") as Promise<Completion<never>>);
  expect(field(unknown.payload, "code")).toBe("unknown-connection");
  const forged = await failed(
    db.observeDeadline(handle, "work-0", "tok-forged", 50n) as Promise<Completion<never>>,
  );
  expect(field(forged.payload, "code")).toBe("forged-token");
});

test("db deadline is a single visible tick", async () => {
  const { db, handle, token } = await bootstrap();
  const missing = await failed(db.deadlineFacts(handle, "work-0") as Promise<Completion<never>>);
  expect(missing.declaration.name).toBe("db::db_fault");
  expect(field(missing.payload, "code")).toBe("no-conversation");
  const negative = await failed(
    db.observeDeadline(handle, "work-0", token, -1n) as Promise<Completion<never>>,
  );
  expect(field(negative.payload, "code")).toBe("malformed-statement");
  const observed = await ok(db.observeDeadline(handle, "work-0", token, 50n));
  expect(field(observed, "deadline_ms")).toBe(50n);
  expect(field(observed, "engine")).toBe("postgres");
  expect(String(field(observed, "digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  const reread = await ok(db.deadlineFacts(handle, "work-0"));
  expect(field(reread, "digest")).toBe(field(observed, "digest"));
  const twice = await failed(
    db.observeDeadline(handle, "work-0", token, 60n) as Promise<Completion<never>>,
  );
  expect(field(twice.payload, "code")).toBe("connection-busy");
});

test("db driver settlement is local and needs the deadline first", async () => {
  const { db, handle, token } = await bootstrap();
  const noDeadline = await failed(
    db.recordDriverSettlement(handle, "work-0", token, "completed") as Promise<Completion<never>>,
  );
  expect(noDeadline.declaration.name).toBe("db::db_fault");
  expect(field(noDeadline.payload, "code")).toBe("no-conversation");
  await ok(db.observeDeadline(handle, "work-0", token, 50n));
  const badWord = await failed(
    db.recordDriverSettlement(handle, "work-0", token, "unknown") as Promise<Completion<never>>,
  );
  expect(field(badWord.payload, "code")).toBe("malformed-statement");
  const settled = await ok(db.recordDriverSettlement(handle, "work-0", token, "timed-out"));
  expect(field(settled, "outcome")).toBe("timed-out");
  expect(field(settled, "server_effect")).toBe("unresolved");
  const reread = await ok(db.driverFacts(handle, "work-0"));
  expect(field(reread, "digest")).toBe(field(settled, "digest"));
  const twice = await failed(
    db.recordDriverSettlement(handle, "work-0", token, "completed") as Promise<Completion<never>>,
  );
  expect(field(twice.payload, "code")).toBe("connection-busy");
});

test("db server ack reconciles after the driver and unknown is explicit", async () => {
  const { db, handle, token } = await bootstrap();
  const noDriver = await failed(
    db.recordServerAck(handle, "work-0", token, "applied") as Promise<Completion<never>>,
  );
  expect(noDriver.declaration.name).toBe("db::db_fault");
  expect(field(noDriver.payload, "code")).toBe("no-conversation");
  await ok(db.observeDeadline(handle, "work-0", token, 50n));
  await ok(db.recordDriverSettlement(handle, "work-0", token, "completed"));
  const badWord = await failed(
    db.recordServerAck(handle, "work-0", token, "denied") as Promise<Completion<never>>,
  );
  expect(field(badWord.payload, "code")).toBe("malformed-statement");
  const acked = await ok(db.recordServerAck(handle, "work-0", token, "unknown"));
  expect(field(acked, "effect")).toBe("unknown");
  const reread = await ok(db.serverFacts(handle, "work-0"));
  expect(field(reread, "digest")).toBe(field(acked, "digest"));
  const twice = await failed(
    db.recordServerAck(handle, "work-0", token, "applied") as Promise<Completion<never>>,
  );
  expect(field(twice.payload, "code")).toBe("connection-busy");
});

test("db cancel needs a capable engine grant and proves nothing", async () => {
  const { db, handle, token } = await bootstrap();
  const uncapable = await failed(
    db.acquireCancelGrant(handle, "sqlite") as Promise<Completion<never>>,
  );
  expect(uncapable.declaration.name).toBe("db::db_fault");
  expect(field(uncapable.payload, "code")).toBe("malformed-statement");
  const missing = await failed(db.cancelFacts(handle, "work-0") as Promise<Completion<never>>);
  expect(field(missing.payload, "code")).toBe("no-conversation");
  const granted = await ok(db.acquireCancelGrant(handle, "postgres"));
  const grantFacts = field(granted, "facts");
  expect(field(grantFacts, "engine")).toBe("postgres");
  expect(String(field(grantFacts, "grant_digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  const grant = field(granted, "grant") as string;
  const invented = await failed(
    db.requestCancel(handle, "work-0", token, "cancel-invented") as Promise<Completion<never>>,
  );
  expect(field(invented.payload, "code")).toBe("forged-token");
  const cancelled = await ok(db.requestCancel(handle, "work-0", token, grant));
  expect(field(cancelled, "proves")).toBe("nothing-about-server");
  const reread = await ok(db.cancelFacts(handle, "work-0"));
  expect(field(reread, "digest")).toBe(field(cancelled, "digest"));
  const twice = await failed(
    db.requestCancel(handle, "work-0", token, grant) as Promise<Completion<never>>,
  );
  expect(field(twice.payload, "code")).toBe("connection-busy");
  const mysqlWork = await ok(db.dispatch(handle, "work-1", "SELECT 1", "mysql"));
  const stale = await failed(
    db.requestCancel(handle, "work-1", field(mysqlWork, "token") as string, grant) as Promise<
      Completion<never>
    >,
  );
  expect(field(stale.payload, "code")).toBe("forged-token");
});

test("db quiesce reports live work and fences settle nothing", async () => {
  const { db, handle, token } = await bootstrap();
  await ok(db.dispatch(handle, "work-1", "SELECT 2", "postgres"));
  const earlyFence = await failed(db.fence(handle, "work-0", token) as Promise<Completion<never>>);
  expect(earlyFence.declaration.name).toBe("db::db_fault");
  expect(field(earlyFence.payload, "code")).toBe("no-conversation");
  const live = await ok(db.quiesceEngine(handle, "postgres"));
  expect([...dataArray(live)]).toEqual(["work-0", "work-1"]);
  const refused = await failed(
    db.dispatch(handle, "work-2", "SELECT 3", "postgres") as Promise<Completion<never>>,
  );
  expect(field(refused.payload, "code")).toBe("namespace-closed");
  const fenced = await ok(db.fence(handle, "work-0", token));
  expect(field(fenced, "settles")).toBe("nothing");
  const reread = await ok(db.fenceFacts(handle, "work-0"));
  expect(field(reread, "ack_digest")).toBe(field(fenced, "ack_digest"));
  const unfenced = await failed(db.fenceFacts(handle, "work-1") as Promise<Completion<never>>);
  expect(field(unfenced.payload, "code")).toBe("no-conversation");
});

test("db lease stays retained until explicit release", async () => {
  const { db, handle, token } = await bootstrap();
  const held = await ok(db.leaseFacts(handle, "work-0"));
  expect(field(held, "namespace")).toBe("ns-alpha");
  expect(field(held, "retained")).toBe(true);
  await ok(db.observeDeadline(handle, "work-0", token, 50n));
  await ok(db.recordDriverSettlement(handle, "work-0", token, "completed"));
  await ok(db.recordServerAck(handle, "work-0", token, "applied"));
  await ok(db.quiesceEngine(handle, "postgres"));
  await ok(db.fence(handle, "work-0", token));
  const still = await ok(db.leaseFacts(handle, "work-0"));
  expect(field(still, "retained")).toBe(true);
  const released = await ok(db.deadlineRelease(handle, "work-0", token));
  expect(field(released, "released")).toBe(true);
  const freed = await ok(db.leaseFacts(handle, "work-0"));
  expect(field(freed, "retained")).toBe(false);
  const ack = await ok(db.deadlineReleaseAck(handle, "work-0"));
  expect(field(ack, "ack_digest")).toBe(field(released, "ack_digest"));
});

test("db release needs settlement, known effect, and a fence", async () => {
  const { db, handle, token } = await bootstrap();
  const noAck = await failed(db.deadlineReleaseAck(handle, "work-0") as Promise<Completion<never>>);
  expect(noAck.declaration.name).toBe("db::db_fault");
  expect(field(noAck.payload, "code")).toBe("no-conversation");
  const noDriver = await failed(
    db.deadlineRelease(handle, "work-0", token) as Promise<Completion<never>>,
  );
  expect(field(noDriver.payload, "code")).toBe("connection-busy");
  await ok(db.observeDeadline(handle, "work-0", token, 50n));
  await ok(db.recordDriverSettlement(handle, "work-0", token, "timed-out"));
  await ok(db.recordServerAck(handle, "work-0", token, "unknown"));
  await ok(db.quiesceEngine(handle, "postgres"));
  await ok(db.fence(handle, "work-0", token));
  const unknownBlocks = await failed(
    db.deadlineRelease(handle, "work-0", token) as Promise<Completion<never>>,
  );
  expect(field(unknownBlocks.payload, "code")).toBe("connection-busy");
  const sent = await ok(db.dispatch(handle, "work-1", "SELECT 2", "mysql"));
  const other = field(sent, "token") as string;
  await ok(db.observeDeadline(handle, "work-1", other, 10n));
  await ok(db.recordDriverSettlement(handle, "work-1", other, "completed"));
  await ok(db.recordServerAck(handle, "work-1", other, "absent"));
  await ok(db.quiesceEngine(handle, "mysql"));
  const noFence = await failed(
    db.deadlineRelease(handle, "work-1", other) as Promise<Completion<never>>,
  );
  expect(field(noFence.payload, "code")).toBe("no-conversation");
  await ok(db.fence(handle, "work-1", other));
  await ok(db.deadlineRelease(handle, "work-1", other));
  const twice = await failed(
    db.deadlineRelease(handle, "work-1", other) as Promise<Completion<never>>,
  );
  expect(field(twice.payload, "code")).toBe("closed-handle");
});

test("db deadline revoked owners fail stale without touching the service", async () => {
  const { owner, db } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const sent = await ok(db.dispatch(handle, "work-0", "SELECT 1", "postgres"));
  await ok(owner.releaseGrant(handle));
  const stale = await failed(db.dispatchFacts(handle, "work-0") as Promise<Completion<never>>);
  expect(stale.declaration.name).toBe("test::stale_handle");
  const staleList = await failed(
    db.quiesceEngine(handle, "postgres") as Promise<Completion<never>>,
  );
  expect(staleList.declaration.name).toBe("test::stale_handle");
  expect(typeof field(sent, "token")).toBe("string");
});

test("db deadline malformed inputs throw before the service runs", async () => {
  const { db, handle, token } = await bootstrap();
  await expect(db.dispatchFacts(handle, 7)).rejects.toThrow("db::dispatch_facts work");
  await expect(db.observeDeadline(handle, "work-0", token, "50")).rejects.toThrow(
    "db::observe_deadline deadline_ms",
  );
  await expect(db.requestCancel(handle, "work-0", token, 7)).rejects.toThrow(
    "db::request_cancel grant",
  );
});
