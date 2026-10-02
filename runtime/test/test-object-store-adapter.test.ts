// NT-I17 object-store adapter contract: the 15 operations
// drive one adapter-held K27 service with owner admission
// first; facts map to nominal records with snake_case fields
// (ints as bigint, bytes as bytes::buffer copies,
// continuations as option::value<str>); revoked owners fail
// test::stale_handle without touching the service; and every
// service rejection maps VERBATIM to
// store::store_fault{layer, code} so K27 verdicts (ungranted
// prefixes, forged tokens/continuations, escaping keys,
// unsettled reads, incomplete scans) stay visible to Can rows
// instead of collapsing into stale_handle.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import type { Completion } from "../completion.ts";
import { dataArray, dataProperty, record, recordIdentity } from "../data.ts";
import { ownBytes } from "../bytes.ts";
import { createTestOwner } from "../test-support/owner.ts";
import { createStore } from "../test-support/slices/i17/store.ts";

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
  (error) => error.name === "test::stale_handle" || error.name === "store::store_fault",
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
const SOME = typeId("option::some");
const NONE = typeId("option::none");

function rig() {
  const owner = createTestOwner(domain, { invalidGrant: err("test::stale_handle") });
  const store = createStore(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      storeFault: err("store::store_fault"),
      optionSome: SOME,
      optionNone: NONE,
      prefixReceipt: typeId("store::prefix_receipt"),
      sessionFacts: typeId("store::session_facts"),
      objectFacts: typeId("store::object_facts"),
      writeFacts: typeId("store::write_facts"),
      pageFacts: typeId("store::page_facts"),
      pendingFacts: typeId("store::pending_facts"),
      storedObject: typeId("store::stored_object"),
      cleanupReceipt: typeId("store::cleanup_receipt"),
    },
    owner,
  );
  return { owner, store };
}

const GRANT = "ng1-0123456789abcdef0123456789abcdef";
const OWNER = "owner-a";
const PREFIX = "tenant-a/owned";

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
const buf = (bytes: readonly number[]): unknown => ownBytes(new Uint8Array(bytes));
const none = (): unknown => record(NONE, []);
const some = (value: string): unknown => record(SOME, [["value", value]]);

async function bootstrap() {
  const { owner, store } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  await ok(store.openPrefix(handle, OWNER, PREFIX));
  await ok(store.openSession(handle, OWNER, PREFIX, "s0"));
  const token = (await ok(store.sessionTokenForTest(handle, OWNER, PREFIX, "s0"))) as string;
  return { owner, store, handle, token };
}

test("store prefix admission binds exact grants only", async () => {
  const { store, handle } = await bootstrap();
  const receipt = await ok(store.receipt(handle, OWNER, PREFIX));
  expect(field(receipt, "prefix")).toBe(PREFIX);
  expect(field(receipt, "owner")).toBe(OWNER);
  expect(field(receipt, "handle")).toBe(`prefix-handle:${OWNER}:${PREFIX}`);
  expect(String(field(receipt, "handle_digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  expect(field(receipt, "objects")).toBe(0n);
  for (const prefix of ["tenant-a", "tenant-a/owned/deeper", "tenant-a/other", "tenant-a/../x"]) {
    const refused = await failed(
      store.openPrefix(handle, OWNER, prefix) as Promise<Completion<never>>,
    );
    expect(refused.declaration.name).toBe("store::store_fault");
    expect(field(refused.payload, "layer")).toBe("engine");
    expect(field(refused.payload, "code")).toBe("unknown-prefix");
  }
  const wrongOwner = await failed(
    store.openPrefix(handle, "owner-z", PREFIX) as Promise<Completion<never>>,
  );
  expect(field(wrongOwner.payload, "code")).toBe("unknown-prefix");
  const pinned = await failed(
    store.closePrefix(handle, OWNER, PREFIX) as Promise<Completion<never>>,
  );
  expect(field(pinned.payload, "code")).toBe("prefix-closed");
});

test("store sessions pin and tokens verify by lookup", async () => {
  const { store, handle, token } = await bootstrap();
  const facts = await ok(store.openSession(handle, OWNER, PREFIX, "s0"));
  expect(field(facts, "session")).toBe("s0");
  expect(field(facts, "pinned")).toBe(true);
  expect(String(field(facts, "handle_digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  expect(token).toMatch(/^sess-[0-9a-f]{32}$/);
  const unknown = await failed(
    store.sessionTokenForTest(handle, OWNER, PREFIX, "s9") as Promise<Completion<never>>,
  );
  expect(unknown.declaration.name).toBe("store::store_fault");
  expect(field(unknown.payload, "layer")).toBe("driver");
  expect(field(unknown.payload, "code")).toBe("unknown-session");
  const forged = await failed(
    store.closeSession(handle, OWNER, PREFIX, "s0", "sess-forged") as Promise<Completion<never>>,
  );
  expect(field(forged.payload, "code")).toBe("forged-token");
  await ok(store.openSession(handle, OWNER, PREFIX, "s1"));
  const foreign = await failed(
    store.closeSession(handle, "owner-b", "tenant-b/owned", "s0", token) as Promise<
      Completion<never>
    >,
  );
  expect(field(foreign.payload, "code")).toBe("unknown-session");
  await ok(store.closeSession(handle, OWNER, PREFIX, "s0", token));
  const closed = await failed(
    store.sessionTokenForTest(handle, OWNER, PREFIX, "s0") as Promise<Completion<never>>,
  );
  expect(field(closed.payload, "code")).toBe("unknown-session");
});

test("store writes settle explicitly and read byte-exact", async () => {
  const { store, handle, token } = await bootstrap();
  const payload = buf([1, 2, 3, 4]);
  const staged = await ok(store.put(handle, OWNER, PREFIX, "s0", token, `${PREFIX}/a`, payload));
  expect(field(staged, "write_id")).toBe("w1");
  expect(field(staged, "settled")).toBe(false);
  expect(String(field(staged, "bytes_digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
  const invisible = await failed(
    store.get(handle, OWNER, PREFIX, "s0", token, `${PREFIX}/a`) as Promise<Completion<never>>,
  );
  expect(invisible.declaration.name).toBe("store::store_fault");
  expect(field(invisible.payload, "code")).toBe("unknown-key");
  const queued = await ok(store.pending(handle, OWNER, PREFIX, "s0", token));
  expect(field(queued, "count")).toBe(1n);
  const busy = await failed(
    store.put(handle, OWNER, PREFIX, "s0", token, `${PREFIX}/a`, buf([9])) as Promise<
      Completion<never>
    >,
  );
  expect(field(busy.payload, "code")).toBe("write-busy");
  const settled = await ok(store.settleWrite(handle, OWNER, PREFIX, "s0", token, "w1"));
  expect(field(settled, "size")).toBe(4n);
  const fetched = await ok(store.get(handle, OWNER, PREFIX, "s0", token, `${PREFIX}/a`));
  expect(field(fetched, "size")).toBe(4n);
  expect(await ok(store.compareBytes(handle, field(fetched, "bytes"), payload))).toBe(true);
  const resettle = await failed(
    store.settleWrite(handle, OWNER, PREFIX, "s0", token, "w1") as Promise<Completion<never>>,
  );
  expect(field(resettle.payload, "code")).toBe("write-settled");
  const missing = await failed(
    store.settleWrite(handle, OWNER, PREFIX, "s0", token, "w9") as Promise<Completion<never>>,
  );
  expect(field(missing.payload, "code")).toBe("write-unknown");
  await ok(store.delete(handle, OWNER, PREFIX, "s0", token, `${PREFIX}/a`));
  const gone = await failed(
    store.get(handle, OWNER, PREFIX, "s0", token, `${PREFIX}/a`) as Promise<Completion<never>>,
  );
  expect(field(gone.payload, "code")).toBe("unknown-key");
});

test("store keys stay inside the owned grant", async () => {
  const { store, handle, token } = await bootstrap();
  const escapes = await failed(
    store.put(handle, OWNER, PREFIX, "s0", token, "shared/logos/acme.png", buf([1])) as Promise<
      Completion<never>
    >,
  );
  expect(escapes.declaration.name).toBe("store::store_fault");
  expect(field(escapes.payload, "layer")).toBe("store");
  expect(field(escapes.payload, "code")).toBe("key-escapes-prefix");
  const malformed = await failed(
    store.put(handle, OWNER, PREFIX, "s0", token, `${PREFIX}//a`, buf([1])) as Promise<
      Completion<never>
    >,
  );
  expect(field(malformed.payload, "code")).toBe("malformed-key");
  const self = await failed(
    store.put(handle, OWNER, PREFIX, "s0", token, PREFIX, buf([1])) as Promise<Completion<never>>,
  );
  expect(field(self.payload, "code")).toBe("malformed-key");
});

test("store compare_bytes predicates exact equality", async () => {
  const { store, handle } = await bootstrap();
  expect(await ok(store.compareBytes(handle, buf([1, 2]), buf([1, 2])))).toBe(true);
  expect(await ok(store.compareBytes(handle, buf([1, 2]), buf([1, 3])))).toBe(false);
  expect(await ok(store.compareBytes(handle, buf([1]), buf([1, 2])))).toBe(false);
  expect(await ok(store.compareBytes(handle, buf([]), buf([])))).toBe(true);
});

test("store listings page with opaque single-use continuations", async () => {
  const { store, handle, token } = await bootstrap();
  for (const key of ["a", "b", "c"]) {
    const staged = await ok(
      store.put(handle, OWNER, PREFIX, "s0", token, `${PREFIX}/${key}`, buf([7])),
    );
    await ok(
      store.settleWrite(handle, OWNER, PREFIX, "s0", token, field(staged, "write_id") as string),
    );
  }
  const first = await ok(store.list(handle, OWNER, PREFIX, "s0", token, 2n, none()));
  expect(field(first, "count")).toBe(2n);
  expect(field(first, "complete")).toBe(false);
  expect([...dataArray(field(first, "keys"))].map((entry) => field(entry, "key"))).toEqual([
    `${PREFIX}/a`,
    `${PREFIX}/b`,
  ]);
  const edge = field(first, "next_continuation");
  expect(recordIdentity(edge)).toBe(SOME);
  const second = await ok(
    store.list(handle, OWNER, PREFIX, "s0", token, 2n, some(field(edge, "value") as string)),
  );
  expect(field(second, "count")).toBe(1n);
  expect(field(second, "complete")).toBe(true);
  expect(recordIdentity(field(second, "next_continuation"))).toBe(NONE);
  const consumed = await failed(
    store.list(
      handle,
      OWNER,
      PREFIX,
      "s0",
      token,
      2n,
      some(field(edge, "value") as string),
    ) as Promise<Completion<never>>,
  );
  expect(consumed.declaration.name).toBe("store::store_fault");
  expect(field(consumed.payload, "code")).toBe("forged-continuation");
  const invented = await failed(
    store.list(handle, OWNER, PREFIX, "s0", token, 2n, some("cont-invented")) as Promise<
      Completion<never>
    >,
  );
  expect(field(invented.payload, "code")).toBe("forged-continuation");
  const badLimit = await failed(
    store.list(handle, OWNER, PREFIX, "s0", token, 0n, none()) as Promise<Completion<never>>,
  );
  expect(field(badLimit.payload, "code")).toBe("bad-page");
  const overLimit = await failed(
    store.list(handle, OWNER, PREFIX, "s0", token, 9n, none()) as Promise<Completion<never>>,
  );
  expect(field(overLimit.payload, "code")).toBe("bad-page");
});

test("store continuations go stale on mutation", async () => {
  const { store, handle, token } = await bootstrap();
  for (const key of ["a", "b"]) {
    const staged = await ok(
      store.put(handle, OWNER, PREFIX, "s0", token, `${PREFIX}/${key}`, buf([7])),
    );
    await ok(
      store.settleWrite(handle, OWNER, PREFIX, "s0", token, field(staged, "write_id") as string),
    );
  }
  const first = await ok(store.list(handle, OWNER, PREFIX, "s0", token, 1n, none()));
  const edge = field(first, "next_continuation") as unknown;
  const staged = await ok(store.put(handle, OWNER, PREFIX, "s0", token, `${PREFIX}/c`, buf([8])));
  await ok(
    store.settleWrite(handle, OWNER, PREFIX, "s0", token, field(staged, "write_id") as string),
  );
  const stale = await failed(
    store.list(
      handle,
      OWNER,
      PREFIX,
      "s0",
      token,
      1n,
      some(field(edge, "value") as string),
    ) as Promise<Completion<never>>,
  );
  expect(stale.declaration.name).toBe("store::store_fault");
  expect(field(stale.payload, "code")).toBe("stale-continuation");
});

test("store seal needs quiesced sessions, writes, keys, and a terminal scan", async () => {
  const { store, handle, token } = await bootstrap();
  const live = await failed(store.sealCleanup(handle, OWNER, PREFIX) as Promise<Completion<never>>);
  expect(live.declaration.name).toBe("store::store_fault");
  expect(field(live.payload, "code")).toBe("session-live");
  const staged = await ok(store.put(handle, OWNER, PREFIX, "s0", token, `${PREFIX}/a`, buf([1])));
  await ok(store.closeSession(handle, OWNER, PREFIX, "s0", token));
  const writes = await failed(
    store.sealCleanup(handle, OWNER, PREFIX) as Promise<Completion<never>>,
  );
  expect(field(writes.payload, "code")).toBe("pending-writes");
  await ok(store.openSession(handle, OWNER, PREFIX, "s1"));
  const token1 = (await ok(store.sessionTokenForTest(handle, OWNER, PREFIX, "s1"))) as string;
  await ok(
    store.settleWrite(handle, OWNER, PREFIX, "s1", token1, field(staged, "write_id") as string),
  );
  await ok(store.closeSession(handle, OWNER, PREFIX, "s1", token1));
  const keys = await failed(store.sealCleanup(handle, OWNER, PREFIX) as Promise<Completion<never>>);
  expect(field(keys.payload, "code")).toBe("keys-remain");
  await ok(store.openSession(handle, OWNER, PREFIX, "s2"));
  const token2 = (await ok(store.sessionTokenForTest(handle, OWNER, PREFIX, "s2"))) as string;
  await ok(store.delete(handle, OWNER, PREFIX, "s2", token2, `${PREFIX}/a`));
  await ok(store.closeSession(handle, OWNER, PREFIX, "s2", token2));
  const scan = await failed(store.sealCleanup(handle, OWNER, PREFIX) as Promise<Completion<never>>);
  expect(field(scan.payload, "code")).toBe("scan-incomplete");
  await ok(store.openSession(handle, OWNER, PREFIX, "s3"));
  const token3 = (await ok(store.sessionTokenForTest(handle, OWNER, PREFIX, "s3"))) as string;
  const terminal = await ok(store.list(handle, OWNER, PREFIX, "s3", token3, 8n, none()));
  expect(field(terminal, "complete")).toBe(true);
  await ok(store.closeSession(handle, OWNER, PREFIX, "s3", token3));
  const sealed = await ok(store.sealCleanup(handle, OWNER, PREFIX));
  expect(field(sealed, "objects")).toBe(0n);
  expect(field(sealed, "pending")).toBe(0n);
  expect(String(field(sealed, "digest"))).toMatch(/^sha256:[0-9a-f]{64}$/);
});

test("store shared digest watches the foreign bucket", async () => {
  const { store, handle, token } = await bootstrap();
  const before = (await ok(store.sharedDigest(handle))) as string;
  expect(before).toMatch(/^sha256:[0-9a-f]{64}$/);
  const staged = await ok(store.put(handle, OWNER, PREFIX, "s0", token, `${PREFIX}/a`, buf([1])));
  await ok(
    store.settleWrite(handle, OWNER, PREFIX, "s0", token, field(staged, "write_id") as string),
  );
  await ok(store.delete(handle, OWNER, PREFIX, "s0", token, `${PREFIX}/a`));
  expect(await ok(store.sharedDigest(handle))).toBe(before);
});

test("store revoked owners fail stale without touching the service", async () => {
  const { owner, store } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  await ok(store.openPrefix(handle, OWNER, PREFIX));
  await ok(store.openSession(handle, OWNER, PREFIX, "s0"));
  await ok(owner.releaseGrant(handle));
  const stale = await failed(store.receipt(handle, OWNER, PREFIX) as Promise<Completion<never>>);
  expect(stale.declaration.name).toBe("test::stale_handle");
  const stalePure = await failed(store.sharedDigest(handle) as Promise<Completion<never>>);
  expect(stalePure.declaration.name).toBe("test::stale_handle");
});

test("store malformed inputs throw before the service runs", async () => {
  const { store, handle, token } = await bootstrap();
  await expect(store.receipt(handle, 7, PREFIX)).rejects.toThrow("store::receipt grant_owner");
  await expect(
    store.put(handle, OWNER, PREFIX, "s0", token, `${PREFIX}/a`, "nope"),
  ).rejects.toThrow("store::put payload");
  await expect(store.compareBytes(handle, buf([1]), "nope")).rejects.toThrow(
    "store::compare_bytes claimed",
  );
  await expect(store.list(handle, OWNER, PREFIX, "s0", token, 2n, "nope")).rejects.toThrow(
    "store::list continuation",
  );
  await expect(store.list(handle, OWNER, PREFIX, "s0", token, "2", none())).rejects.toThrow(
    "store::list limit",
  );
});
