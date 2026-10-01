// NT-I14 transaction-observer adapter contract: the 9 operations
// drive one adapter-held K24 service with owner admission first;
// facts map to nominal records with snake_case fields (ints as
// bigint, id cells as db::cell records); revoked owners fail
// test::stale_handle without touching the service; and every
// service rejection maps VERBATIM to db::db_fault{layer, code} so
// K24 verdicts (busy actor, forged token, unsettled release,
// wrong-handle ids) stay visible to Can rows instead of
// collapsing into stale_handle.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import type { Completion } from "../completion.ts";
import { dataArray, dataProperty, record, recordIdentity } from "../data.ts";
import { createTestOwner } from "../test-support/owner.ts";
import { createDbTransaction } from "../test-support/slices/i14/db_transaction.ts";

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
const NUM = typeId("db::number_cell");
const TXT = typeId("db::text_cell");
const BYT = typeId("db::bytes_cell");
const NUL = typeId("db::null_cell");

function rig() {
  const owner = createTestOwner(domain, { invalidGrant: err("test::stale_handle") });
  const db = createDbTransaction(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      dbFault: err("db::db_fault"),
      numberCell: NUM,
      textCell: TXT,
      bytesCell: BYT,
      nullCell: NUL,
      callbackEntryFacts: typeId("db::callback_entry_facts"),
      callbackEntry: typeId("db::callback_entry"),
      actorIdentityFacts: typeId("db::actor_identity_facts"),
      lastInsertIdRecord: typeId("db::last_insert_id_record"),
      identityComparison: typeId("db::identity_comparison"),
      settlementRecord: typeId("db::settlement_record"),
      releaseRecord: typeId("db::release_record"),
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
const num = (lexeme: string): unknown => record(NUM, [["lexeme", lexeme]]);
const txt = (value: string): unknown => record(TXT, [["text", value]]);
const claim = (actor: string, connection: string, id: unknown): unknown =>
  record(typeId("db::last_insert_id_claim"), [
    ["actor", actor],
    ["connection", connection],
    ["id", id],
  ]);

async function bootstrap() {
  const { owner, db } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const entered = await ok(db.enterCallback(handle, "a1"));
  const token = field(entered, "token") as string;
  return { owner, db, handle, token };
}

test("db enter/actorFacts lifecycle pins identity and rejects re-entry", async () => {
  const { db, handle } = await bootstrap();
  const facts = await ok(db.actorFacts(handle, "a1"));
  expect(field(facts, "actor")).toBe("a1");
  expect(field(facts, "connection")).toBe("conn:a1");
  expect(field(facts, "entry_seq")).toBe(0n);
  expect(String(field(facts, "handle_digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  expect(field(facts, "settled")).toBe(false);
  expect(field(facts, "released")).toBe(false);
  const reenter = await failed(db.enterCallback(handle, "a1") as Promise<Completion<never>>);
  expect(reenter.declaration.name).toBe("db::db_fault");
  expect(field(reenter.payload, "layer")).toBe("driver");
  expect(field(reenter.payload, "code")).toBe("connection-busy");
  const unknown = await failed(db.actorFacts(handle, "ghost") as Promise<Completion<never>>);
  expect(field(unknown.payload, "code")).toBe("unknown-connection");
});

test("db LAST_INSERT_ID binds the id to the handle", async () => {
  const { db, handle, token } = await bootstrap();
  const missing = await failed(db.lastInsertIdFacts(handle, "a1") as Promise<Completion<never>>);
  expect(field(missing.payload, "code")).toBe("no-conversation");
  const recorded = await ok(db.recordLastInsertId(handle, "a1", token, num("42")));
  expect(field(recorded, "actor")).toBe("a1");
  expect(field(recorded, "connection")).toBe("conn:a1");
  expect(recordIdentity(field(recorded, "id"))).toBe(NUM);
  expect(field(field(recorded, "id"), "lexeme")).toBe("42");
  expect(String(field(recorded, "digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  const reread = await ok(db.lastInsertIdFacts(handle, "a1"));
  expect(field(reread, "digest")).toBe(field(recorded, "digest"));
  const forged = await failed(
    db.recordLastInsertId(handle, "a1", "bogus", num("1")) as Promise<Completion<never>>,
  );
  expect(forged.declaration.name).toBe("db::db_fault");
  expect(field(forged.payload, "code")).toBe("forged-token");
});

test("db compare_last_insert_id checks identity before value and never throws", async () => {
  const { db, handle, token } = await bootstrap();
  const stored = await ok(db.recordLastInsertId(handle, "a1", token, num("42")));
  const exact = await ok(db.compareLastInsertId(handle, stored, claim("a1", "conn:a1", num("42"))));
  expect(field(exact, "match")).toBe(true);
  const swapped = await ok(
    db.compareLastInsertId(handle, stored, claim("a2", "conn:a1", num("42"))),
  );
  expect(field(swapped, "match")).toBe(false);
  expect([...dataArray(field(swapped, "mismatches"))]).toEqual([-1n]);
  const drifted = await ok(
    db.compareLastInsertId(handle, stored, claim("a1", "conn:a1", num("43"))),
  );
  expect(field(drifted, "match")).toBe(false);
  expect([...dataArray(field(drifted, "mismatches"))]).toEqual([0n]);
  const both = await ok(db.compareLastInsertId(handle, stored, claim("a2", "conn:a9", txt("x"))));
  expect(field(both, "match")).toBe(false);
  expect([...dataArray(field(both, "mismatches"))]).toEqual([-1n, 0n]);
});

test("db settlement records facts only; words and order enforced", async () => {
  const { db, handle, token } = await bootstrap();
  const unsettled = await failed(db.settlementFacts(handle, "a1") as Promise<Completion<never>>);
  expect(field(unsettled.payload, "code")).toBe("no-conversation");
  const badOutcome = await failed(
    db.recordSettlement(handle, "a1", token, "commited", "postgres") as Promise<Completion<never>>,
  );
  expect(badOutcome.declaration.name).toBe("db::db_fault");
  expect(field(badOutcome.payload, "layer")).toBe("sql");
  expect(field(badOutcome.payload, "code")).toBe("malformed-statement");
  const badEngine = await failed(
    db.recordSettlement(handle, "a1", token, "committed", "oracle") as Promise<Completion<never>>,
  );
  expect(field(badEngine.payload, "code")).toBe("malformed-statement");
  const settled = await ok(db.recordSettlement(handle, "a1", token, "committed", "postgres"));
  expect(field(settled, "outcome")).toBe("committed");
  expect(field(settled, "engine")).toBe("postgres");
  expect(String(field(settled, "digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  const reread = await ok(db.settlementFacts(handle, "a1"));
  expect(field(reread, "digest")).toBe(field(settled, "digest"));
  const twice = await failed(
    db.recordSettlement(handle, "a1", token, "committed", "postgres") as Promise<Completion<never>>,
  );
  expect(field(twice.payload, "code")).toBe("connection-busy");
});

test("db release requires settlement and acks exactly once", async () => {
  const { db, handle, token } = await bootstrap();
  const early = await failed(db.release(handle, "a1", token) as Promise<Completion<never>>);
  expect(early.declaration.name).toBe("db::db_fault");
  expect(field(early.payload, "code")).toBe("connection-busy");
  const noAck = await failed(db.releaseAck(handle, "a1") as Promise<Completion<never>>);
  expect(field(noAck.payload, "code")).toBe("no-conversation");
  await ok(db.recordSettlement(handle, "a1", token, "rolled-back", "sqlite"));
  const ack = await ok(db.release(handle, "a1", token));
  expect(field(ack, "actor")).toBe("a1");
  expect(field(ack, "released")).toBe(true);
  expect(String(field(ack, "ack_digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  const reread = await ok(db.releaseAck(handle, "a1"));
  expect(field(reread, "ack_digest")).toBe(field(ack, "ack_digest"));
  const twice = await failed(db.release(handle, "a1", token) as Promise<Completion<never>>);
  expect(field(twice.payload, "code")).toBe("closed-handle");
  const afterRelease = await failed(
    db.recordLastInsertId(handle, "a1", token, num("1")) as Promise<Completion<never>>,
  );
  expect(field(afterRelease.payload, "code")).toBe("closed-handle");
});

test("db transaction revoked owners fail stale without touching the service", async () => {
  const { owner, db } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  await ok(db.enterCallback(handle, "a1"));
  await ok(owner.releaseGrant(handle));
  const stale = await failed(db.actorFacts(handle, "a1") as Promise<Completion<never>>);
  expect(stale.declaration.name).toBe("test::stale_handle");
  const stalePure = await failed(
    db.compareLastInsertId(handle, {}, {}) as Promise<Completion<never>>,
  );
  expect(stalePure.declaration.name).toBe("test::stale_handle");
});

test("db transaction malformed inputs throw before the service runs", async () => {
  const { db, handle, token } = await bootstrap();
  await expect(db.enterCallback(handle, 7)).rejects.toThrow("db::enter_callback actor");
  await expect(db.recordLastInsertId(handle, "a1", token, 7)).rejects.toThrow("db::cell");
  await expect(db.recordSettlement(handle, "a1", token, "committed", 7)).rejects.toThrow(
    "db::record_settlement engine",
  );
});
