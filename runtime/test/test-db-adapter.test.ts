// NT-I13 database-observer adapter contract: the 20 operations drive
// one adapter-held K22 core plus one K23 RETURNING observer with
// owner admission first; the grant doubles as the service-level
// scenario owner; facts map to nominal records with snake_case
// fields (ints as bigint, bytes cells as bytes::buffer copies);
// revoked owners fail test::stale_handle without touching either
// service; and every service rejection maps VERBATIM to
// db::db_fault{layer, code} so K22/K23 verdicts (swap, forgery,
// arity, narrowing, caps) stay visible to Can rows instead of
// collapsing into stale_handle.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import type { Completion } from "../completion.ts";
import { array, dataArray, dataProperty, record, recordIdentity } from "../data.ts";
import { copyBytes, ownBytes } from "../bytes.ts";
import { createTestOwner } from "../test-support/owner.ts";
import { createDbObserver } from "../test-support/slices/i13/db.ts";

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
const SEEDROW = typeId("db::seed_row");

function rig() {
  const owner = createTestOwner(domain, { invalidGrant: err("test::stale_handle") });
  const db = createDbObserver(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      dbFault: err("db::db_fault"),
      numberCell: NUM,
      textCell: TXT,
      bytesCell: BYT,
      nullCell: NUL,
      namespaceReceipt: typeId("db::namespace_receipt"),
      connectionFacts: typeId("db::connection_facts"),
      rowFacts: typeId("db::row_facts"),
      readFacts: typeId("db::read_facts"),
      seedFacts: typeId("db::seed_facts"),
      conversationFacts: typeId("db::conversation_facts"),
      rowComparison: typeId("db::row_comparison"),
      compileFacts: typeId("db::compile_facts"),
      returningRowFacts: typeId("db::returning_row_facts"),
      returningPayloadFacts: typeId("db::returning_payload_facts"),
      finalRowsFacts: typeId("db::final_rows_facts"),
      returningComparison: typeId("db::returning_comparison"),
      returningCreditVerdict: typeId("db::returning_credit_verdict"),
    },
    owner,
  );
  return { owner, db };
}

const GRANT = "ng1-0123456789abcdef0123456789abcdef";
const NS = "tenant-alpha";

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
const byt = (bytes: Uint8Array): unknown => record(BYT, [["data", ownBytes(bytes)]]);
const nul = (): unknown => record(NUL, []);
const seedRow = (cells: readonly unknown[]): unknown =>
  record(SEEDROW, [["cells", array([...cells])]]);

async function bootstrap() {
  const { owner, db } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  await ok(db.openNamespace(handle, NS));
  await ok(db.pinConnection(handle, NS, "raw-1"));
  const token = (await ok(db.connectionTokenForTest(handle, NS, "raw-1"))) as string;
  return { owner, db, handle, token };
}

test("db namespace lifecycle joins, receipts, and closes", async () => {
  const { db, handle, token } = await bootstrap();
  const first = await ok(db.openNamespace(handle, NS));
  expect(field(first, "namespace")).toBe(NS);
  expect(field(first, "tables")).toBe(0n);
  expect(String(field(first, "handle"))).toMatch(/^ns-handle:/);
  const joined = await ok(db.openNamespace(handle, NS));
  expect(field(joined, "handle")).toBe(field(first, "handle"));
  const receipt = await ok(db.receipt(handle, NS));
  expect(field(receipt, "handle")).toBe(field(first, "handle"));
  expect(String(field(receipt, "handle_digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  const undeclared = await failed(
    db.openNamespace(handle, "tenant-gamma") as Promise<Completion<never>>,
  );
  expect(undeclared.declaration.name).toBe("db::db_fault");
  expect(field(undeclared.payload, "layer")).toBe("engine");
  expect(field(undeclared.payload, "code")).toBe("unknown-namespace");
  const pinnedClose = await failed(db.closeNamespace(handle, NS) as Promise<Completion<never>>);
  expect(field(pinnedClose.payload, "code")).toBe("namespace-closed");
  await ok(db.unpinConnection(handle, NS, "raw-1", token));
  await ok(db.closeNamespace(handle, NS));
  const closed = await failed(db.receipt(handle, NS) as Promise<Completion<never>>);
  expect(field(closed.payload, "code")).toBe("unknown-namespace");
});

test("db pin/seed/begin/fetch/end/unpin roundtrip keeps cells exact", async () => {
  const { db, handle, token } = await bootstrap();
  const pinned = await ok(db.pinConnection(handle, NS, "raw-2"));
  expect(field(pinned, "connection")).toBe("raw-2");
  expect(field(pinned, "pinned")).toBe(true);
  expect(field(pinned, "conversation_open")).toBe(false);
  const seeded = await ok(
    db.seed(handle, NS, "items", [
      seedRow([num("1.10"), txt("héllo"), byt(new Uint8Array([1, 2, 3])), nul()]),
      seedRow([num("9007199254740993"), txt(""), byt(new Uint8Array([])), nul()]),
    ]),
  );
  expect(field(seeded, "table")).toBe("items");
  expect(field(seeded, "rows")).toBe(2n);
  const conversation = await ok(db.beginRead(handle, NS, "raw-1", token, "items"));
  expect(field(conversation, "conversation")).toBe("raw-1:items");
  const read = await ok(db.fetch(handle, NS, "raw-1", token, "items"));
  expect(field(read, "table")).toBe("items");
  expect(field(read, "row_count")).toBe(2n);
  expect(String(field(read, "digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  const rows = dataArray(field(read, "rows"));
  expect(field(rows[0], "seq")).toBe(0n);
  const cells = dataArray(field(rows[0], "cells"));
  expect(recordIdentity(cells[0])).toBe(NUM);
  expect(field(cells[0], "lexeme")).toBe("1.10");
  expect(field(cells[1], "text")).toBe("héllo");
  expect(recordIdentity(cells[2])).toBe(BYT);
  expect([
    ...copyBytes(field(cells[2], "data"), { source: "test", start: 0, end: 0, invocation: [] }),
  ]).toEqual([1, 2, 3]);
  expect(recordIdentity(cells[3])).toBe(NUL);
  const second = dataArray(field(rows[1], "cells"));
  expect(field(second[0], "lexeme")).toBe("9007199254740993");
  await ok(db.endRead(handle, NS, "raw-1", token));
  await ok(db.unpinConnection(handle, NS, "raw-1", token));
  const ragged = await failed(
    db.seed(handle, NS, "items", [seedRow([num("1")]), seedRow([num("1"), num("2")])]) as Promise<
      Completion<never>
    >,
  );
  expect(ragged.declaration.name).toBe("db::db_fault");
  expect(field(ragged.payload, "layer")).toBe("sql");
  expect(field(ragged.payload, "code")).toBe("row-arity");
});

test("db bytes cells cross as copies in both directions", async () => {
  const { db, handle, token } = await bootstrap();
  const held = new Uint8Array([9, 8, 7]);
  await ok(db.seed(handle, NS, "blobs", [seedRow([byt(held)])]));
  held[0] = 0;
  await ok(db.beginRead(handle, NS, "raw-1", token, "blobs"));
  const first = await ok(db.fetch(handle, NS, "raw-1", token, "blobs"));
  const cell = dataArray(field(dataArray(field(first, "rows"))[0], "cells"))[0];
  const seen = copyBytes(field(cell, "data"), { source: "test", start: 0, end: 0, invocation: [] });
  expect([...seen]).toEqual([9, 8, 7]);
  seen[0] = 0;
  const again = await ok(db.fetch(handle, NS, "raw-1", token, "blobs"));
  const cellAgain = dataArray(field(dataArray(field(again, "rows"))[0], "cells"))[0];
  expect([
    ...copyBytes(field(cellAgain, "data"), { source: "test", start: 0, end: 0, invocation: [] }),
  ]).toEqual([9, 8, 7]);
  await ok(db.endRead(handle, NS, "raw-1", token));
});

test("db compare_row detects narrowing, impostors, and arity", async () => {
  const { db, handle } = await bootstrap();
  const exact = await ok(
    db.compareRow(handle, [num("9007199254740993"), nul()], [num("9007199254740993"), nul()]),
  );
  expect(field(exact, "match")).toBe(true);
  const narrowed = await ok(
    db.compareRow(handle, [num("9007199254740993")], [num("9007199254740992")]),
  );
  expect(field(narrowed, "match")).toBe(false);
  expect([...(field(narrowed, "mismatches") as readonly unknown[])]).toEqual([0n]);
  const impostor = await ok(db.compareRow(handle, [nul()], [txt("")]));
  expect(field(impostor, "match")).toBe(false);
  const arity = await ok(db.compareRow(handle, [num("1"), num("2")], [num("1")]));
  expect(field(arity, "match")).toBe(false);
  expect([...(field(arity, "mismatches") as readonly unknown[])]).toEqual([-1n]);
});

test("db record_compile pins schema; bad tags fault verbatim", async () => {
  const { db, handle } = await bootstrap();
  const compiled = await ok(
    db.recordCompile(handle, "seed-fixture", "INSERT INTO items VALUES (1)", ["number", "text"]),
  );
  expect(field(compiled, "compile")).toBe("compile-0");
  expect(field(compiled, "fixture")).toBe("seed-fixture");
  expect(String(field(compiled, "statement_digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  expect([...(field(compiled, "schema") as readonly unknown[])]).toEqual(["number", "text"]);
  const reread = await ok(db.compileRecord(handle, "compile-0"));
  expect(field(reread, "statement_digest")).toBe(field(compiled, "statement_digest"));
  const badTag = await failed(
    db.recordCompile(handle, "seed-fixture", "SELECT 1", ["bogus"]) as Promise<Completion<never>>,
  );
  expect(badTag.declaration.name).toBe("db::db_fault");
  expect(field(badTag.payload, "layer")).toBe("sql");
  expect(field(badTag.payload, "code")).toBe("malformed-statement");
  const unknown = await failed(db.compileRecord(handle, "compile-9") as Promise<Completion<never>>);
  expect(field(unknown.payload, "code")).toBe("unknown-connection");
});

test("db RETURNING payload vs final rows compare; cross-compile faults", async () => {
  const { db, handle } = await bootstrap();
  const compiled = await ok(db.recordCompile(handle, "d1", "INSERT ... RETURNING id", ["number"]));
  const id = field(compiled, "compile") as string;
  const missing = await failed(db.payloadFacts(handle, id) as Promise<Completion<never>>);
  expect(field(missing.payload, "code")).toBe("no-conversation");
  const payload = await ok(
    db.recordReturning(handle, id, [seedRow([num("7")]), seedRow([num("8")])]),
  );
  expect(field(payload, "compile")).toBe(id);
  expect(field(payload, "row_count")).toBe(2n);
  const reread = await ok(db.payloadFacts(handle, id));
  expect(field(reread, "digest")).toBe(field(payload, "digest"));
  const final = await ok(
    db.recordFinalRows(handle, id, [seedRow([num("7")]), seedRow([num("8")])]),
  );
  expect(field(final, "row_count")).toBe(2n);
  const match = await ok(db.comparePayloadToFinal(handle, payload, final));
  expect(field(match, "match")).toBe(true);
  const drifted = await ok(
    db.recordFinalRows(handle, id, [seedRow([num("7")]), seedRow([num("9")])]),
  );
  const mismatch = await ok(db.comparePayloadToFinal(handle, payload, drifted));
  expect(field(mismatch, "match")).toBe(false);
  const other = await ok(db.recordCompile(handle, "d1b", "INSERT ... RETURNING id", ["number"]));
  const otherPayload = await ok(
    db.recordReturning(handle, field(other, "compile") as string, [seedRow([num("7")])]),
  );
  const crossed = await failed(
    db.comparePayloadToFinal(handle, payload, otherPayload) as Promise<Completion<never>>,
  );
  expect(crossed.declaration.name).toBe("db::db_fault");
  expect(field(crossed.payload, "code")).toBe("malformed-statement");
});

test("db compare_claimed_row never throws; credit_verdict never credits", async () => {
  const { db, handle } = await bootstrap();
  const match = await ok(db.compareClaimedRow(handle, [txt("a")], [txt("a")]));
  expect(field(match, "match")).toBe(true);
  const mismatch = await ok(db.compareClaimedRow(handle, [txt("a")], [txt("b")]));
  expect(field(mismatch, "match")).toBe(false);
  expect([...(field(mismatch, "mismatches") as readonly unknown[])]).toEqual([0n]);
  const verdict = await ok(db.creditVerdict(handle));
  expect(field(verdict, "credit_c")).toBe(false);
  expect(String(field(verdict, "reason")).length).toBeGreaterThan(0);
});

test("db revoked owners fail stale without touching the service", async () => {
  const { owner, db } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  await ok(db.openNamespace(handle, NS));
  await ok(owner.releaseGrant(handle));
  const stale = await failed(db.receipt(handle, NS) as Promise<Completion<never>>);
  expect(stale.declaration.name).toBe("test::stale_handle");
  const stalePure = await failed(db.compareRow(handle, [], []) as Promise<Completion<never>>);
  expect(stalePure.declaration.name).toBe("test::stale_handle");
});

test("db malformed inputs throw before the service runs", async () => {
  const { db, handle } = await bootstrap();
  await expect(db.seed(handle, NS, 7, [])).rejects.toThrow("db::seed table");
  await expect(db.fetch(handle, NS, "raw-1", 7, "items")).rejects.toThrow("db::fetch token");
  await expect(db.compareRow(handle, [record("test:bogus", [])], [])).rejects.toThrow(
    "db::compare_row stored",
  );
  await expect(db.compareRow(handle, [record(BYT, [["data", "nope"]])], [])).rejects.toThrow(
    "bytes::buffer",
  );
});
