// NT-I15 poison-observer adapter contract: the 17 operations
// drive one adapter-held K25 service with owner admission first;
// facts map to nominal records with snake_case fields (ints as
// bigint, cells as db::cell records, poison_digest as
// option::value<str>); revoked owners fail test::stale_handle
// without touching the service; and every service rejection maps
// VERBATIM to db::db_fault{layer, code} so K25 verdicts (bad
// SQLSTATE, unsettled reads, missing evidence, cross-attempt
// probes) stay visible to Can rows instead of collapsing into
// stale_handle.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import type { Completion } from "../completion.ts";
import { array, dataArray, dataProperty, record, recordIdentity } from "../data.ts";
import { createTestOwner } from "../test-support/owner.ts";
import { createDbPoison } from "../test-support/slices/i15/db_poison.ts";

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
const SOME = typeId("option::some");
const NONE = typeId("option::none");
const SEEDROW = typeId("db::seed_row");

function rig() {
  const owner = createTestOwner(domain, { invalidGrant: err("test::stale_handle") });
  const db = createDbPoison(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      dbFault: err("db::db_fault"),
      numberCell: NUM,
      textCell: TXT,
      bytesCell: BYT,
      nullCell: NUL,
      optionSome: SOME,
      optionNone: NONE,
      poisonAttemptRecord: typeId("db::poison_attempt_record"),
      poisonErrorFacts: typeId("db::poison_error_facts"),
      poisonCallbackFacts: typeId("db::poison_callback_facts"),
      poisonSettlementRecord: typeId("db::poison_settlement_record"),
      sentinelRowFacts: typeId("db::sentinel_row_facts"),
      freshSentinelFacts: typeId("db::fresh_sentinel_facts"),
      replaySentinelFacts: typeId("db::replay_sentinel_facts"),
      sentinelComparison: typeId("db::sentinel_comparison"),
      settlementAgreement: typeId("db::settlement_agreement"),
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
const seedRow = (cells: readonly unknown[]): unknown =>
  record(SEEDROW, [["cells", array([...cells])]]);
const SENTINEL = () => seedRow([num("7"), txt("seven")]);

async function bootstrap() {
  const { owner, db } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const begun = await ok(
    db.beginPoisonAttempt(handle, ["number", "text"], SENTINEL(), "DELETE FROM items"),
  );
  return { owner, db, handle, attempt: field(begun, "attempt") as string };
}

test("db poison/control attempts pin kind and optional digest", async () => {
  const { db, handle, attempt } = await bootstrap();
  expect(attempt).toBe("attempt-0");
  const reread = await ok(db.attemptRecord(handle, "attempt-0"));
  expect(field(reread, "kind")).toBe("poison");
  expect([...dataArray(field(reread, "schema"))]).toEqual(["number", "text"]);
  expect(String(field(reread, "sentinel_digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  const digest = field(reread, "poison_digest");
  expect(recordIdentity(digest)).toBe(SOME);
  expect(String(field(digest, "value"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  const control = await ok(db.beginControlAttempt(handle, ["number"], seedRow([num("1")])));
  expect(field(control, "attempt")).toBe("attempt-1");
  expect(field(control, "kind")).toBe("control");
  expect(recordIdentity(field(control, "poison_digest"))).toBe(NONE);
  const badTag = await failed(
    db.beginPoisonAttempt(handle, ["bogus"], SENTINEL(), "DELETE FROM items") as Promise<
      Completion<never>
    >,
  );
  expect(badTag.declaration.name).toBe("db::db_fault");
  expect(field(badTag.payload, "code")).toBe("malformed-statement");
  const unknown = await failed(db.attemptRecord(handle, "attempt-9") as Promise<Completion<never>>);
  expect(field(unknown.payload, "code")).toBe("unknown-connection");
});

test("db poison error pins SQLSTATE code, class, and bounded detail", async () => {
  const { db, handle, attempt } = await bootstrap();
  const missing = await failed(db.errorFacts(handle, attempt) as Promise<Completion<never>>);
  expect(field(missing.payload, "code")).toBe("no-conversation");
  const badCodeFirst = await failed(
    db.recordPoisonError(handle, attempt, "nope", "x") as Promise<Completion<never>>,
  );
  expect(badCodeFirst.declaration.name).toBe("db::db_fault");
  expect(field(badCodeFirst.payload, "code")).toBe("malformed-statement");
  const successCode = await failed(
    db.recordPoisonError(handle, attempt, "00000", "x") as Promise<Completion<never>>,
  );
  expect(field(successCode.payload, "code")).toBe("malformed-statement");
  const recorded = await ok(db.recordPoisonError(handle, attempt, "23505", "duplicate key"));
  expect(field(recorded, "code")).toBe("23505");
  expect(field(recorded, "class")).toBe("23");
  expect(field(recorded, "detail")).toBe("duplicate key");
  expect(String(field(recorded, "detail_digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  const reread = await ok(db.errorFacts(handle, attempt));
  expect(field(reread, "detail_digest")).toBe(field(recorded, "detail_digest"));
  const twice = await failed(
    db.recordPoisonError(handle, attempt, "23505", "again") as Promise<Completion<never>>,
  );
  expect(field(twice.payload, "code")).toBe("connection-busy");
});

test("db callback reports are inert labels proving nothing", async () => {
  const { db, handle, attempt } = await bootstrap();
  const missing = await failed(db.callbackFacts(handle, attempt) as Promise<Completion<never>>);
  expect(field(missing.payload, "code")).toBe("no-conversation");
  const badWord = await failed(
    db.recordCallbackReport(handle, attempt, "ok") as Promise<Completion<never>>,
  );
  expect(badWord.declaration.name).toBe("db::db_fault");
  expect(field(badWord.payload, "code")).toBe("malformed-statement");
  const success = await ok(db.recordCallbackReport(handle, attempt, "success"));
  expect(field(success, "reported")).toBe("success");
  expect(field(success, "proves")).toBe("nothing");
  const control = await ok(db.beginControlAttempt(handle, ["number"], seedRow([num("1")])));
  const threw = await ok(
    db.recordCallbackReport(handle, field(control, "attempt") as string, "threw"),
  );
  expect(field(threw, "reported")).toBe("threw");
  expect(field(threw, "proves")).toBe("nothing");
});

test("db poison settlement is terminal facts only", async () => {
  const { db, handle, attempt } = await bootstrap();
  const unsettled = await failed(
    db.poisonSettlementFacts(handle, attempt) as Promise<Completion<never>>,
  );
  expect(field(unsettled.payload, "code")).toBe("no-conversation");
  const badWord = await failed(
    db.recordPoisonSettlement(handle, attempt, "unknown") as Promise<Completion<never>>,
  );
  expect(badWord.declaration.name).toBe("db::db_fault");
  expect(field(badWord.payload, "code")).toBe("malformed-statement");
  const settled = await ok(db.recordPoisonSettlement(handle, attempt, "rolled-back"));
  expect(field(settled, "outcome")).toBe("rolled-back");
  expect(field(settled, "terminal")).toBe(true);
  const reread = await ok(db.poisonSettlementFacts(handle, attempt));
  expect(field(reread, "outcome")).toBe("rolled-back");
  const twice = await failed(
    db.recordPoisonSettlement(handle, attempt, "committed") as Promise<Completion<never>>,
  );
  expect(field(twice.payload, "code")).toBe("connection-busy");
});

test("db fresh/replay sentinel reads compare within one attempt", async () => {
  const { db, handle, attempt } = await bootstrap();
  const fresh = await ok(db.recordFreshRead(handle, attempt, [seedRow([num("7"), txt("seven")])]));
  expect(field(fresh, "read")).toBe("fresh");
  expect(field(fresh, "row_count")).toBe(1n);
  const rows = dataArray(field(fresh, "rows"));
  expect(field(rows[0], "seq")).toBe(0n);
  const replay = await ok(
    db.recordReplayRead(handle, attempt, [seedRow([num("7"), txt("seven")])]),
  );
  expect(field(replay, "read")).toBe("replay");
  const match = await ok(db.compareFreshToReplay(handle, fresh, replay));
  expect(field(match, "match")).toBe(true);
  const other = await ok(db.beginControlAttempt(handle, ["number", "text"], SENTINEL()));
  const otherAttempt = field(other, "attempt") as string;
  const otherFresh = await ok(
    db.recordFreshRead(handle, otherAttempt, [seedRow([num("7"), txt("seven")])]),
  );
  const otherReplay = await ok(
    db.recordReplayRead(handle, otherAttempt, [seedRow([num("8"), txt("seven")])]),
  );
  const mismatch = await ok(db.compareFreshToReplay(handle, otherFresh, otherReplay));
  expect(field(mismatch, "match")).toBe(false);
  expect([...dataArray(field(mismatch, "mismatches"))]).toEqual([0n]);
  const crossed = await failed(
    db.compareFreshToReplay(handle, fresh, otherFresh) as Promise<Completion<never>>,
  );
  expect(crossed.declaration.name).toBe("db::db_fault");
  expect(field(crossed.payload, "code")).toBe("malformed-statement");
});

test("db sentinel probes find exact rows per read", async () => {
  const { db, handle, attempt } = await bootstrap();
  const fresh = await ok(db.recordFreshRead(handle, attempt, [seedRow([num("7"), txt("seven")])]));
  expect(await ok(db.sentinelPresentInFresh(handle, attempt, fresh))).toBe(true);
  const replay = await ok(db.recordReplayRead(handle, attempt, [seedRow([num("9"), txt("nine")])]));
  expect(await ok(db.sentinelPresentInReplay(handle, attempt, replay))).toBe(false);
  const other = await ok(db.beginControlAttempt(handle, ["number", "text"], SENTINEL()));
  const otherAttempt = field(other, "attempt") as string;
  const present = await ok(
    db.recordReplayRead(handle, otherAttempt, [seedRow([num("7"), txt("seven")])]),
  );
  expect(await ok(db.sentinelPresentInReplay(handle, otherAttempt, present))).toBe(true);
  const crossed = await failed(
    db.sentinelPresentInFresh(handle, otherAttempt, fresh) as Promise<Completion<never>>,
  );
  expect(crossed.declaration.name).toBe("db::db_fault");
  expect(field(crossed.payload, "code")).toBe("malformed-statement");
});

test("db settlement agreement needs terminal facts plus both reads", async () => {
  const { db, handle, attempt } = await bootstrap();
  const noSettlement = await failed(
    db.settlementAgreesWithReads(handle, attempt) as Promise<Completion<never>>,
  );
  expect(noSettlement.declaration.name).toBe("db::db_fault");
  expect(field(noSettlement.payload, "code")).toBe("no-conversation");
  await ok(db.recordPoisonSettlement(handle, attempt, "rolled-back"));
  const noReads = await failed(
    db.settlementAgreesWithReads(handle, attempt) as Promise<Completion<never>>,
  );
  expect(field(noReads.payload, "code")).toBe("no-conversation");
  await ok(db.recordFreshRead(handle, attempt, [seedRow([num("1"), txt("other")])]));
  await ok(db.recordReplayRead(handle, attempt, [seedRow([num("1"), txt("other")])]));
  const agreed = await ok(db.settlementAgreesWithReads(handle, attempt));
  expect(field(agreed, "agree")).toBe(true);
  expect(field(agreed, "reason")).toBe("sentinel-absent-after-rollback");
  const control = await ok(db.beginControlAttempt(handle, ["number", "text"], SENTINEL()));
  const cid = field(control, "attempt") as string;
  await ok(db.recordPoisonSettlement(handle, cid, "committed"));
  await ok(db.recordFreshRead(handle, cid, [seedRow([num("7"), txt("seven")])]));
  await ok(db.recordReplayRead(handle, cid, [seedRow([num("7"), txt("seven")])]));
  const committed = await ok(db.settlementAgreesWithReads(handle, cid));
  expect(field(committed, "agree")).toBe(true);
  expect(field(committed, "reason")).toBe("sentinel-present-after-commit");
  const survived = await ok(db.beginControlAttempt(handle, ["number", "text"], SENTINEL()));
  const sid = field(survived, "attempt") as string;
  await ok(db.recordPoisonSettlement(handle, sid, "rolled-back"));
  await ok(db.recordFreshRead(handle, sid, [seedRow([num("7"), txt("seven")])]));
  await ok(db.recordReplayRead(handle, sid, [seedRow([num("7"), txt("seven")])]));
  const disagreed = await ok(db.settlementAgreesWithReads(handle, sid));
  expect(field(disagreed, "agree")).toBe(false);
  expect(field(disagreed, "reason")).toBe("sentinel-survived-rollback");
});

test("db poison revoked owners fail stale without touching the service", async () => {
  const { owner, db } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  await ok(db.beginPoisonAttempt(handle, ["number"], seedRow([num("1")]), "DELETE FROM items"));
  await ok(owner.releaseGrant(handle));
  const stale = await failed(db.attemptRecord(handle, "attempt-0") as Promise<Completion<never>>);
  expect(stale.declaration.name).toBe("test::stale_handle");
  const staleBool = await failed(
    db.sentinelPresentInFresh(handle, "attempt-0", {}) as Promise<Completion<never>>,
  );
  expect(staleBool.declaration.name).toBe("test::stale_handle");
});

test("db poison malformed inputs throw before the service runs", async () => {
  const { db, handle, attempt } = await bootstrap();
  await expect(db.attemptRecord(handle, 7)).rejects.toThrow("db::attempt_record attempt");
  await expect(db.recordFreshRead(handle, attempt, 7)).rejects.toThrow(
    "db::record_fresh_read rows",
  );
  await expect(
    db.beginPoisonAttempt(
      handle,
      ["number"],
      seedRow([record("test:bogus", [])]),
      "DELETE FROM items",
    ),
  ).rejects.toThrow("db::cell");
});
