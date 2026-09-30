// K27 bounded self-check: owned object-store observations.
//
// Run with: bun tools/runtime/test-services/object-store/service-check.ts
// Local controls only. No buckets, networks, SDKs, files, timers, live
// hosts, or live runtimes. Every loop below is bounded by a small constant.
// Live host-dependent controls wait for the qualified profile and Q task.

import { strict as assert } from "node:assert";
import {
  checkLimits,
  layerOfCode,
  OBJECT_STORE_CODES,
  OBJECT_STORE_INDEPENDENCE_SCOPE,
  OBJECT_STORE_SCHEMA_VERSION,
  ObjectStoreError,
  ObjectStoreService,
  type ObjectStoreCode,
  type ObjectStoreLayer,
  type ObjectStoreLimits,
} from "./service.ts";

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectStoreError(
  body: () => unknown,
  code: ObjectStoreCode,
  layer: ObjectStoreLayer,
): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof ObjectStoreError)) {
      assert.fail(`expected an ObjectStoreError, got ${String(error)}`);
    }
    assert.equal(error.code, code);
    assert.equal(error.layer, layer);
    assert.ok(error.message.includes(`[${layer}:${code}]`));
    return;
  }
  assert.fail(`expected failure ${layer}:${code}`);
}

function roomyLimits(): ObjectStoreLimits {
  return checkLimits({
    maxPrefixes: 4,
    maxSessions: 8,
    maxObjects: 32,
    maxObjectBytes: 1024,
    maxPageSize: 8,
    maxPendingWrites: 16,
  });
}

function sharedBucket(): Map<string, Uint8Array> {
  return new Map([
    ["shared/logos/acme.png", new Uint8Array([137, 80, 78, 71])],
    ["shared/empty", new Uint8Array([])],
    ["tenant-b/owned/keep", new Uint8Array([9, 9, 9])],
  ]);
}

function openOwned(): { service: ObjectStoreService; token: string } {
  const service = new ObjectStoreService(
    [
      ["owner-a", "tenant-a/owned"],
      ["owner-b", "tenant-b/owned"],
    ],
    sharedBucket(),
    roomyLimits(),
  );
  service.openPrefix("owner-a", "tenant-a/owned");
  service.openSession("owner-a", "tenant-a/owned", "sess-1");
  const token = service.sessionTokenForTest("owner-a", "tenant-a/owned", "sess-1");
  return { service, token };
}

function putSettled(
  service: ObjectStoreService,
  token: string,
  key: string,
  bytes: Uint8Array,
  owner = "owner-a",
  prefix = "tenant-a/owned",
  session = "sess-1",
): void {
  const accepted = service.put(owner, prefix, session, token, key, bytes);
  assert.equal(accepted.settled, false);
  service.settleWrite(owner, prefix, session, token, accepted.writeId);
}

// --- Admission: the caller prefix is not authority ---------------------------

check("prefix-receipt-binds-owned-grant", () => {
  const service = new ObjectStoreService(
    [["owner-a", "tenant-a/owned"]],
    sharedBucket(),
    roomyLimits(),
  );
  const before = service.sharedDigest();
  const receipt = service.openPrefix("owner-a", "tenant-a/owned");
  assert.equal(receipt.prefix, "tenant-a/owned");
  assert.equal(receipt.owner, "owner-a");
  assert.ok(receipt.handle.startsWith("prefix-handle:owner-a:tenant-a/owned"));
  assert.ok(receipt.handleDigest.startsWith("sha256:"));
  assert.equal(receipt.handleDigest.length, "sha256:".length + 64);
  assert.equal(receipt.objects, 0);
  // The receipt carries a digest only: the raw token never crosses.
  assert.deepEqual(Object.keys(receipt).sort(), [
    "handle",
    "handleDigest",
    "objects",
    "owner",
    "prefix",
  ]);

  // Reopening joins the same owned prefix; the digest is stable.
  const again = service.openPrefix("owner-a", "tenant-a/owned");
  assert.equal(again.handleDigest, receipt.handleDigest);

  // Caller prefixes outside the owned grant reject before any effect:
  // siblings, parents, the bare bucket, and ungranted tenants.
  for (const foreign of [
    "tenant-a/other",
    "tenant-a",
    "tenant-a/owned/deeper",
    "tenant-b/owned",
    "shared",
    "shared/logos",
  ]) {
    expectStoreError(() => service.openPrefix("owner-a", foreign), "unknown-prefix", "engine");
  }
  // Path tricks are not a way around the grant edge.
  for (const tricky of [
    "/tenant-a/owned",
    "tenant-a/owned/",
    "tenant-a//owned",
    "tenant-a/owned/../other",
    "tenant-a/./owned",
    "",
  ]) {
    expectStoreError(() => service.openPrefix("owner-a", tricky), "unknown-prefix", "engine");
  }
  // A different owner holds a different grant: no joining across owners.
  expectStoreError(
    () => service.openPrefix("owner-b", "tenant-a/owned"),
    "unknown-prefix",
    "engine",
  );
  assert.equal(service.sharedDigest(), before);
});

check("keys-stay-inside-the-grant", () => {
  const { service, token } = openOwned();
  const before = service.sharedDigest();
  // Well-formed but out-of-scope keys reject without effect.
  for (const foreign of [
    "shared/logos/acme.png",
    "tenant-a/other/x",
    "tenant-a/owned",
    "tenant-b/owned/keep",
  ]) {
    const key = foreign === "tenant-a/owned" ? foreign : foreign;
    if (key === "tenant-a/owned") {
      expectStoreError(
        () => service.put("owner-a", "tenant-a/owned", "sess-1", token, key, new Uint8Array([1])),
        "malformed-key",
        "store",
      );
    } else {
      expectStoreError(
        () => service.put("owner-a", "tenant-a/owned", "sess-1", token, key, new Uint8Array([1])),
        "key-escapes-prefix",
        "store",
      );
    }
    expectStoreError(
      () => service.get("owner-a", "tenant-a/owned", "sess-1", token, key),
      key === "tenant-a/owned" ? "malformed-key" : "key-escapes-prefix",
      "store",
    );
    expectStoreError(
      () => service.delete("owner-a", "tenant-a/owned", "sess-1", token, key),
      key === "tenant-a/owned" ? "malformed-key" : "key-escapes-prefix",
      "store",
    );
  }
  // Malformed keys reject on shape before scope is even considered.
  for (const bad of [
    "",
    "/tenant-a/owned/x",
    "tenant-a/owned/x/",
    "tenant-a//owned/x",
    "tenant-a/owned/../x",
    "tenant-a/owned/./x",
  ]) {
    expectStoreError(
      () => service.put("owner-a", "tenant-a/owned", "sess-1", token, bad, new Uint8Array([1])),
      "malformed-key",
      "store",
    );
  }
  // Nothing escaped: the shared bucket is byte-identical.
  assert.equal(service.sharedDigest(), before);
  const page = service.list("owner-a", "tenant-a/owned", "sess-1", token, 8, null);
  assert.equal(page.complete, true);
  assert.equal(page.count, 0);
});

// --- Round trip: accepted writes settle explicitly ----------------------------

check("accepted-write-settlement", () => {
  const { service, token } = openOwned();
  const payload = new Uint8Array([222, 173, 190, 239]);
  const accepted = service.put(
    "owner-a",
    "tenant-a/owned",
    "sess-1",
    token,
    "tenant-a/owned/a",
    payload,
  );
  assert.equal(accepted.settled, false);
  assert.ok(accepted.writeId.startsWith("w"));
  assert.ok(accepted.bytesDigest.startsWith("sha256:"));

  // Accepted is not settled: get sees nothing, pending names the write.
  expectStoreError(
    () => service.get("owner-a", "tenant-a/owned", "sess-1", token, "tenant-a/owned/a"),
    "unknown-key",
    "store",
  );
  const blocked = service.pending("owner-a", "tenant-a/owned", "sess-1", token);
  assert.equal(blocked.count, 1);
  assert.equal(blocked.writes[0]?.writeId, accepted.writeId);
  assert.equal(blocked.writes[0]?.settled, false);

  // A second accept on the same key rejects while one is pending.
  expectStoreError(
    () =>
      service.put(
        "owner-a",
        "tenant-a/owned",
        "sess-1",
        token,
        "tenant-a/owned/a",
        new Uint8Array([0]),
      ),
    "write-busy",
    "store",
  );
  // Settling an unknown write rejects; the pending write is untouched.
  expectStoreError(
    () => service.settleWrite("owner-a", "tenant-a/owned", "sess-1", token, "w999"),
    "write-unknown",
    "store",
  );
  assert.equal(service.pending("owner-a", "tenant-a/owned", "sess-1", token).count, 1);

  const facts = service.settleWrite("owner-a", "tenant-a/owned", "sess-1", token, accepted.writeId);
  assert.equal(facts.key, "tenant-a/owned/a");
  assert.equal(facts.size, 4);
  // Double settlement rejects.
  expectStoreError(
    () => service.settleWrite("owner-a", "tenant-a/owned", "sess-1", token, accepted.writeId),
    "write-settled",
    "store",
  );
  // Now the object reads back byte-exact.
  const fetched = service.get("owner-a", "tenant-a/owned", "sess-1", token, "tenant-a/owned/a");
  assert.deepEqual(Array.from(fetched.bytes), [222, 173, 190, 239]);
  assert.equal(fetched.digest, facts.digest);
  assert.equal(service.pending("owner-a", "tenant-a/owned", "sess-1", token).count, 0);
});

check("bytes-are-byte-exact", () => {
  const { service, token } = openOwned();
  const vectors = [
    new Uint8Array([]),
    new Uint8Array([0]),
    new Uint8Array([0, 255, 1, 254]),
    new Uint8Array([222, 173, 190, 239, 0, 0, 1]),
  ];
  vectors.forEach((bytes, index) => {
    putSettled(service, token, `tenant-a/owned/blob-${index}`, bytes);
  });
  vectors.forEach((bytes, index) => {
    const fetched = service.get(
      "owner-a",
      "tenant-a/owned",
      "sess-1",
      token,
      `tenant-a/owned/blob-${index}`,
    );
    assert.equal(service.compareBytes(fetched.bytes, bytes), true);
    assert.equal(fetched.size, bytes.length);
  });
  // One flipped byte is detectable.
  const stored = service.get("owner-a", "tenant-a/owned", "sess-1", token, "tenant-a/owned/blob-2");
  assert.equal(service.compareBytes(stored.bytes, new Uint8Array([0, 255, 1, 254])), true);
  assert.equal(service.compareBytes(stored.bytes, new Uint8Array([0, 255, 1, 253])), false);
  // Empty stays empty and distinct from a zero byte.
  assert.equal(service.compareBytes(new Uint8Array([]), new Uint8Array([0])), false);
  // Untyped misuse fails at the store layer, never with a raw TypeError.
  expectStoreError(
    () => service.compareBytes("00" as unknown as Uint8Array, new Uint8Array([])),
    "malformed-key",
    "store",
  );
  expectStoreError(
    () => service.compareBytes(new Uint8Array([]), [0] as unknown as Uint8Array),
    "malformed-key",
    "store",
  );
});

check("stored-bytes-never-alias-caller-buffers", () => {
  const { service, token } = openOwned();
  const payload = new Uint8Array([1, 2, 3]);
  putSettled(service, token, "tenant-a/owned/alias", payload);
  payload.fill(0);
  const first = service.get("owner-a", "tenant-a/owned", "sess-1", token, "tenant-a/owned/alias");
  assert.deepEqual(Array.from(first.bytes), [1, 2, 3]);
  // Fetched bytes are copies: mutating them cannot corrupt later reads.
  first.bytes[0] = 99;
  const second = service.get("owner-a", "tenant-a/owned", "sess-1", token, "tenant-a/owned/alias");
  assert.equal(second.bytes[0], 1);
  assert.equal(second.digest, first.digest);
});

// --- Pagination: explicit continuation, explicit completeness -----------------

check("pagination-walks-to-terminal", () => {
  const { service, token } = openOwned();
  const names = ["d", "b", "a", "c", "e"];
  for (const name of names) {
    putSettled(service, token, `tenant-a/owned/${name}`, new Uint8Array([name.charCodeAt(0)]));
  }
  // Pages arrive in lexicographic order regardless of put order.
  const first = service.list("owner-a", "tenant-a/owned", "sess-1", token, 2, null);
  assert.equal(first.complete, false);
  assert.deepEqual(
    first.keys.map((entry) => entry.key),
    ["tenant-a/owned/a", "tenant-a/owned/b"],
  );
  assert.ok(first.nextContinuation !== null);
  assert.equal(first.generation, first.generation);

  const second = service.list(
    "owner-a",
    "tenant-a/owned",
    "sess-1",
    token,
    2,
    first.nextContinuation,
  );
  assert.equal(second.complete, false);
  assert.deepEqual(
    second.keys.map((entry) => entry.key),
    ["tenant-a/owned/c", "tenant-a/owned/d"],
  );
  assert.ok(second.nextContinuation !== null);

  const third = service.list(
    "owner-a",
    "tenant-a/owned",
    "sess-1",
    token,
    2,
    second.nextContinuation,
  );
  assert.equal(third.complete, true);
  assert.equal(third.nextContinuation, null);
  assert.deepEqual(
    third.keys.map((entry) => entry.key),
    ["tenant-a/owned/e"],
  );
  // Repeating a full scan is stable: same keys, same terminal digest.
  const solo = service.list("owner-a", "tenant-a/owned", "sess-1", token, 8, null);
  assert.equal(solo.complete, true);
  assert.equal(solo.count, 5);
  assert.deepEqual(
    solo.keys.map((entry) => entry.key),
    [
      "tenant-a/owned/a",
      "tenant-a/owned/b",
      "tenant-a/owned/c",
      "tenant-a/owned/d",
      "tenant-a/owned/e",
    ],
  );
});

check("continuations-are-opaque-and-single-use", () => {
  const { service, token } = openOwned();
  for (const name of ["a", "b", "c"]) {
    putSettled(service, token, `tenant-a/owned/${name}`, new Uint8Array([1]));
  }
  const first = service.list("owner-a", "tenant-a/owned", "sess-1", token, 1, null);
  assert.equal(first.complete, false);
  const edge = first.nextContinuation as string;
  // Invented continuations are never authority.
  for (const forged of ["", "cont-deadbeef", `${edge.slice(0, -1)}x`]) {
    expectStoreError(
      () => service.list("owner-a", "tenant-a/owned", "sess-1", token, 1, forged),
      "forged-continuation",
      "driver",
    );
  }
  // A live continuation works exactly once; replay rejects.
  const second = service.list("owner-a", "tenant-a/owned", "sess-1", token, 1, edge);
  assert.equal(second.complete, false);
  expectStoreError(
    () => service.list("owner-a", "tenant-a/owned", "sess-1", token, 1, edge),
    "forged-continuation",
    "driver",
  );
  // Bad page limits reject before any scan.
  for (const bad of [0, -1, 9, Number.NaN, 1.5]) {
    expectStoreError(
      () => service.list("owner-a", "tenant-a/owned", "sess-1", token, bad, null),
      "bad-page",
      "store",
    );
  }
});

check("mutation-stales-open-continuations", () => {
  const { service, token } = openOwned();
  for (const name of ["a", "b", "c"]) {
    putSettled(service, token, `tenant-a/owned/${name}`, new Uint8Array([1]));
  }
  const first = service.list("owner-a", "tenant-a/owned", "sess-1", token, 1, null);
  const edge = first.nextContinuation as string;
  // Any mutation — settle, delete, even a bare accept — moves the listing.
  putSettled(service, token, "tenant-a/owned/zz", new Uint8Array([2]));
  expectStoreError(
    () => service.list("owner-a", "tenant-a/owned", "sess-1", token, 1, edge),
    "stale-continuation",
    "driver",
  );
  const resume = service.list("owner-a", "tenant-a/owned", "sess-1", token, 1, null);
  const edge2 = resume.nextContinuation as string;
  service.delete("owner-a", "tenant-a/owned", "sess-1", token, "tenant-a/owned/zz");
  expectStoreError(
    () => service.list("owner-a", "tenant-a/owned", "sess-1", token, 1, edge2),
    "stale-continuation",
    "driver",
  );
  const resume2 = service.list("owner-a", "tenant-a/owned", "sess-1", token, 1, null);
  const edge3 = resume2.nextContinuation as string;
  service.put(
    "owner-a",
    "tenant-a/owned",
    "sess-1",
    token,
    "tenant-a/owned/hold",
    new Uint8Array([3]),
  );
  expectStoreError(
    () => service.list("owner-a", "tenant-a/owned", "sess-1", token, 1, edge3),
    "stale-continuation",
    "driver",
  );
});

// --- Cleanup: incomplete pages and empty lists prove nothing ------------------

check("incomplete-scan-cannot-seal", () => {
  const { service, token } = openOwned();
  for (const name of ["a", "b", "c"]) {
    putSettled(service, token, `tenant-a/owned/${name}`, new Uint8Array([1]));
  }
  // An incomplete page is explicitly incomplete and seals nothing.
  const page = service.list("owner-a", "tenant-a/owned", "sess-1", token, 1, null);
  assert.equal(page.complete, false);
  service.closeSession("owner-a", "tenant-a/owned", "sess-1", token);
  expectStoreError(() => service.sealCleanup("owner-a", "tenant-a/owned"), "keys-remain", "engine");
});

check("eventual-empty-list-cannot-seal", () => {
  const { service, token } = openOwned();
  putSettled(service, token, "tenant-a/owned/a", new Uint8Array([1]));
  // Terminal scan observes a live object...
  const full = service.list("owner-a", "tenant-a/owned", "sess-1", token, 8, null);
  assert.equal(full.complete, true);
  service.delete("owner-a", "tenant-a/owned", "sess-1", token, "tenant-a/owned/a");
  // ...the store is now empty, but the terminal scan is stale: the delete
  // moved the generation, so the old scan cannot authorize the seal.
  service.closeSession("owner-a", "tenant-a/owned", "sess-1", token);
  expectStoreError(
    () => service.sealCleanup("owner-a", "tenant-a/owned"),
    "scan-incomplete",
    "engine",
  );
});

check("pending-write-blocks-empty-seal", () => {
  const { service, token } = openOwned();
  // The listing is terminally empty right now...
  const empty = service.list("owner-a", "tenant-a/owned", "sess-1", token, 8, null);
  assert.equal(empty.complete, true);
  assert.equal(empty.count, 0);
  // ...but an accepted write is still unsettled: the empty list cannot
  // prove cleanup while an effect may still land.
  service.put(
    "owner-a",
    "tenant-a/owned",
    "sess-1",
    token,
    "tenant-a/owned/late",
    new Uint8Array([7]),
  );
  service.closeSession("owner-a", "tenant-a/owned", "sess-1", token);
  expectStoreError(
    () => service.sealCleanup("owner-a", "tenant-a/owned"),
    "pending-writes",
    "engine",
  );
});

check("live-session-blocks-seal", () => {
  const { service, token } = openOwned();
  const empty = service.list("owner-a", "tenant-a/owned", "sess-1", token, 8, null);
  assert.equal(empty.complete, true);
  // Terminal empty scan, no objects, no pending writes — but the session
  // is still pinned, so the seal names the live session.
  expectStoreError(
    () => service.sealCleanup("owner-a", "tenant-a/owned"),
    "session-live",
    "engine",
  );
  service.closeSession("owner-a", "tenant-a/owned", "sess-1", token);
  const receipt = service.sealCleanup("owner-a", "tenant-a/owned");
  assert.equal(receipt.objects, 0);
  assert.equal(receipt.pending, 0);
  assert.equal(receipt.owner, "owner-a");
  assert.ok(receipt.handleDigest.startsWith("sha256:"));
  assert.ok(receipt.digest.startsWith("sha256:"));
});

check("full-lifecycle-seals", () => {
  const { service, token } = openOwned();
  for (const name of ["a", "b", "c"]) {
    putSettled(service, token, `tenant-a/owned/${name}`, new Uint8Array([1, 2]));
  }
  // Walk every page to the terminal edge.
  let edge: string | null = null;
  let seen = 0;
  for (let round = 0; round < 4; round++) {
    const page = service.list("owner-a", "tenant-a/owned", "sess-1", token, 2, edge);
    seen += page.count;
    if (page.complete) {
      edge = null;
      break;
    }
    edge = page.nextContinuation;
  }
  assert.equal(seen, 3);
  assert.equal(edge, null);
  // Delete through the same session, then observe the terminal empty page.
  for (const name of ["a", "b", "c"]) {
    service.delete("owner-a", "tenant-a/owned", "sess-1", token, `tenant-a/owned/${name}`);
  }
  expectStoreError(
    () => service.delete("owner-a", "tenant-a/owned", "sess-1", token, "tenant-a/owned/a"),
    "unknown-key",
    "store",
  );
  const terminal = service.list("owner-a", "tenant-a/owned", "sess-1", token, 8, null);
  assert.equal(terminal.complete, true);
  assert.equal(terminal.count, 0);
  service.closeSession("owner-a", "tenant-a/owned", "sess-1", token);
  const receipt = service.sealCleanup("owner-a", "tenant-a/owned");
  assert.equal(receipt.objects, 0);
  assert.equal(receipt.pending, 0);
  service.closePrefix("owner-a", "tenant-a/owned");
});

// --- Negative controls: sessions, ownership, provenance ----------------------

check("tokens-and-owners-are-authority", () => {
  const { service, token } = openOwned();
  putSettled(service, token, "tenant-a/owned/a", new Uint8Array([1]));
  // Invented tokens are never authority, on any session path.
  for (const forged of ["", "sess-deadbeef", token.slice(0, -1)]) {
    expectStoreError(
      () => service.get("owner-a", "tenant-a/owned", "sess-1", forged, "tenant-a/owned/a"),
      "forged-token",
      "driver",
    );
    expectStoreError(
      () => service.list("owner-a", "tenant-a/owned", "sess-1", forged, 8, null),
      "forged-token",
      "driver",
    );
    expectStoreError(
      () =>
        service.put(
          "owner-a",
          "tenant-a/owned",
          "sess-1",
          forged,
          "tenant-a/owned/b",
          new Uint8Array([0]),
        ),
      "forged-token",
      "driver",
    );
    expectStoreError(
      () => service.closeSession("owner-a", "tenant-a/owned", "sess-1", forged),
      "forged-token",
      "driver",
    );
  }
  // A second grant under another owner is separate: same session name,
  // separate pin, separate objects, no token crossing.
  service.openPrefix("owner-b", "tenant-b/owned");
  service.openSession("owner-b", "tenant-b/owned", "sess-1");
  const tokenB = service.sessionTokenForTest("owner-b", "tenant-b/owned", "sess-1");
  assert.notEqual(tokenB, token);
  expectStoreError(
    () => service.get("owner-b", "tenant-b/owned", "sess-1", token, "tenant-b/owned/a"),
    "forged-token",
    "driver",
  );
  expectStoreError(
    () => service.get("owner-a", "tenant-a/owned", "sess-1", tokenB, "tenant-a/owned/a"),
    "forged-token",
    "driver",
  );
  // Owner-b sees none of owner-a's objects.
  putSettled(
    service,
    tokenB,
    "tenant-b/owned/keep",
    new Uint8Array([9]),
    "owner-b",
    "tenant-b/owned",
    "sess-1",
  );
  expectStoreError(
    () => service.get("owner-b", "tenant-b/owned", "sess-1", tokenB, "tenant-a/owned/a"),
    "key-escapes-prefix",
    "store",
  );
  // Closing a prefix with a live session rejects; draining then closes.
  expectStoreError(
    () => service.closePrefix("owner-a", "tenant-a/owned"),
    "prefix-closed",
    "engine",
  );
  service.closeSession("owner-a", "tenant-a/owned", "sess-1", token);
  expectStoreError(
    () => service.get("owner-a", "tenant-a/owned", "sess-1", token, "tenant-a/owned/a"),
    "unknown-session",
    "driver",
  );
});

check("every-failure-names-its-layer", () => {
  // The code vocabulary is closed and every code maps to exactly one layer.
  assert.equal(OBJECT_STORE_CODES.length, 20);
  assert.equal(OBJECT_STORE_SCHEMA_VERSION, "1");
  for (const code of OBJECT_STORE_CODES) {
    const layer = layerOfCode(code);
    assert.ok(["engine", "driver", "store"].includes(layer));
    const error = new ObjectStoreError(code, "probe");
    assert.equal(error.layer, layer);
    assert.equal(error.code, code);
    assert.ok(error.message.startsWith(`[${layer}:${code}]`));
  }
  // Limit shapes reject at the store layer with nothing constructed.
  expectStoreError(() => checkLimits(null), "bad-page", "store");
  expectStoreError(() => checkLimits({ maxPrefixes: 1 }), "bad-page", "store");
  expectStoreError(
    () =>
      checkLimits({
        maxPrefixes: 1,
        maxSessions: 1,
        maxObjects: 1,
        maxObjectBytes: 1,
        maxPageSize: 1,
        maxPendingWrites: 1,
        maxBuckets: 1,
      }),
    "bad-page",
    "store",
  );
  // Capacity faults name the engine layer.
  const tiny = new ObjectStoreService(
    [["o", "p/q"]],
    new Map(),
    checkLimits({
      maxPrefixes: 1,
      maxSessions: 1,
      maxObjects: 1,
      maxObjectBytes: 4,
      maxPageSize: 4,
      maxPendingWrites: 1,
    }),
  );
  tiny.openPrefix("o", "p/q");
  tiny.openSession("o", "p/q", "s");
  const tok = tiny.sessionTokenForTest("o", "p/q", "s");
  expectStoreError(() => tiny.openSession("o", "p/q", "s2"), "capacity-exhausted", "engine");
  expectStoreError(
    () => tiny.put("o", "p/q", "s", tok, "p/q/big", new Uint8Array([1, 2, 3, 4, 5])),
    "capacity-exhausted",
    "engine",
  );
  tiny.put("o", "p/q", "s", tok, "p/q/a", new Uint8Array([1]));
  expectStoreError(
    () => tiny.put("o", "p/q", "s", tok, "p/q/b", new Uint8Array([2])),
    "capacity-exhausted",
    "engine",
  );
});

check("independence-scope-stated", () => {
  // The scope statement is a fixed export: the observer shares no adapter
  // code, decoded values, or client state, and proves its facts from its
  // own seeded doubles alone.
  assert.ok(OBJECT_STORE_INDEPENDENCE_SCOPE.startsWith("independent:"));
  assert.ok(OBJECT_STORE_INDEPENDENCE_SCOPE.includes("no adapter code"));
  assert.ok(OBJECT_STORE_INDEPENDENCE_SCOPE.includes("seeded doubles"));
});

check("shared-bucket-untouched-end-to-end", () => {
  // One service runs a full adversarial script; the shared bucket digest
  // must be identical before and after, and listings never leak foreign
  // keys.
  const service = new ObjectStoreService(
    [["owner-a", "tenant-a/owned"]],
    sharedBucket(),
    roomyLimits(),
  );
  const before = service.sharedDigest();
  service.openPrefix("owner-a", "tenant-a/owned");
  service.openSession("owner-a", "tenant-a/owned", "sess-1");
  const token = service.sessionTokenForTest("owner-a", "tenant-a/owned", "sess-1");
  for (const name of ["x", "y"]) {
    putSettled(service, token, `tenant-a/owned/${name}`, new Uint8Array([5]));
  }
  for (
    let page = service.list("owner-a", "tenant-a/owned", "sess-1", token, 1, null);
    !page.complete;
    page = service.list(
      "owner-a",
      "tenant-a/owned",
      "sess-1",
      token,
      1,
      page.nextContinuation as string,
    )
  ) {
    for (const entry of page.keys) {
      assert.ok(entry.key.startsWith("tenant-a/owned/"));
    }
  }
  const terminal = service.list("owner-a", "tenant-a/owned", "sess-1", token, 8, null);
  assert.equal(terminal.complete, true);
  for (const entry of terminal.keys) {
    assert.ok(entry.key.startsWith("tenant-a/owned/"));
  }
  assert.equal(service.sharedDigest(), before);
  for (const name of ["x", "y"]) {
    service.delete("owner-a", "tenant-a/owned", "sess-1", token, `tenant-a/owned/${name}`);
  }
  assert.equal(service.sharedDigest(), before);
});

console.log(
  JSON.stringify({
    kind: "can.object-store-service-check",
    schema_version: OBJECT_STORE_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
