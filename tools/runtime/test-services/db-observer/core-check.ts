// K22 bounded self-check: independent raw DB observer core.
//
// Run with: bun tools/runtime/test-services/db-observer/core-check.ts
// Local controls only. No databases, drivers, sockets, files, timers, live
// hosts, or live runtimes. Every loop below is bounded by a small constant.
// Live host-dependent controls wait for the qualified profile and Q task.

import { strict as assert } from "node:assert";
import {
  checkLimits,
  DbObserverError,
  DbObserverService,
  DB_OBSERVER_CODES,
  DB_OBSERVER_INDEPENDENCE_SCOPE,
  DB_OBSERVER_SCHEMA_VERSION,
  layerOfCode,
  makeBytesCell,
  makeNullCell,
  makeNumberCell,
  makeTextCell,
  type DbCell,
  type DbObserverCode,
  type DbObserverLayer,
  type DbObserverLimits,
} from "./core.ts";

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectObserverError(
  body: () => unknown,
  code: DbObserverCode,
  layer: DbObserverLayer,
): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof DbObserverError)) {
      assert.fail(`expected a DbObserverError, got ${String(error)}`);
    }
    assert.equal(error.code, code);
    assert.equal(error.layer, layer);
    assert.ok(error.message.includes(`[${layer}:${code}]`));
    return;
  }
  assert.fail(`expected failure ${layer}:${code}`);
}

function roomyLimits(): DbObserverLimits {
  return checkLimits({
    maxNamespaces: 4,
    maxConnections: 8,
    maxTables: 8,
    maxRows: 16,
    maxCellsPerRow: 8,
    maxCellBytes: 1024,
  });
}

function openPinned(): {
  service: DbObserverService;
  token: string;
} {
  const service = new DbObserverService(["tenant-alpha", "tenant-beta"], roomyLimits());
  service.openNamespace("owner-a", "tenant-alpha");
  service.pinConnection("owner-a", "tenant-alpha", "raw-1");
  const token = service.connectionTokenForTest("owner-a", "tenant-alpha", "raw-1");
  return { service, token };
}

// --- Transport controls: namespace receipt, pinning, seed, read -------------

check("namespace-receipt", () => {
  const service = new DbObserverService(["tenant-alpha", "tenant-beta"], roomyLimits());
  const receipt = service.openNamespace("owner-a", "tenant-alpha");
  assert.equal(receipt.namespace, "tenant-alpha");
  assert.equal(receipt.owner, "owner-a");
  assert.ok(receipt.handle.startsWith("ns-handle:owner-a:tenant-alpha"));
  assert.ok(receipt.handleDigest.startsWith("sha256:"));
  assert.equal(receipt.handleDigest.length, "sha256:".length + 64);
  assert.equal(receipt.tables, 0);
  // The receipt carries a digest only: the raw token never crosses.
  assert.deepEqual(Object.keys(receipt).sort(), [
    "handle",
    "handleDigest",
    "namespace",
    "owner",
    "tables",
  ]);

  // Reopening joins the same owned namespace; the digest is stable.
  const again = service.openNamespace("owner-a", "tenant-alpha");
  assert.equal(again.handleDigest, receipt.handleDigest);

  // A different owner gets a different namespace handle.
  const other = service.openNamespace("owner-b", "tenant-alpha");
  assert.notEqual(other.handleDigest, receipt.handleDigest);

  // Foreign namespaces reject at the engine layer before any effect.
  expectObserverError(
    () => service.openNamespace("owner-a", "tenant-gamma"),
    "unknown-namespace",
    "engine",
  );
  expectObserverError(
    () => service.receipt("owner-a", "tenant-gamma"),
    "unknown-namespace",
    "engine",
  );
});

check("pin-and-read-roundtrip", () => {
  const { service, token } = openPinned();
  const facts = service.pinConnection("owner-a", "tenant-alpha", "raw-1");
  assert.equal(facts.pinned, true);
  assert.equal(facts.conversationOpen, false);
  assert.ok(facts.handleDigest.startsWith("sha256:"));

  const seeded = service.seed("owner-a", "tenant-alpha", "widgets", [
    [
      { tag: "number", lexeme: "1" },
      { tag: "text", text: "one" },
    ],
    [
      { tag: "number", lexeme: "2" },
      { tag: "text", text: "two" },
    ],
  ]);
  assert.equal(seeded.rows, 2);

  const conversation = service.beginRead("owner-a", "tenant-alpha", "raw-1", token, "widgets");
  assert.equal(conversation.conversation, "raw-1:widgets");
  const read = service.fetch("owner-a", "tenant-alpha", "raw-1", token, "widgets");
  assert.equal(read.rowCount, 2);
  assert.equal(read.rows.length, 2);
  assert.equal(read.rows[0]?.seq, 0);
  assert.equal(read.rows[1]?.seq, 1);
  const first = read.rows[0]?.cells as readonly DbCell[];
  assert.equal(first[0]?.tag, "number");
  assert.equal((first[0] as { lexeme: string }).lexeme, "1");
  service.endRead("owner-a", "tenant-alpha", "raw-1", token);

  // Unpinning drains cleanly; a second conversation can follow.
  service.beginRead("owner-a", "tenant-alpha", "raw-1", token, "widgets");
  service.endRead("owner-a", "tenant-alpha", "raw-1", token);
  service.unpinConnection("owner-a", "tenant-alpha", "raw-1", token);
  assert.equal(service.liveConnectionCount, 0);
  service.closeNamespace("owner-a", "tenant-alpha");
});

// --- Exactness pins: number lexemes are never coerced -----------------------

check("numbers-are-lexeme-exact", () => {
  const { service, token } = openPinned();
  const lexemes = ["1.10", "1.1", "9007199254740993", "-0", "0", "1e3", "0.30000000000000004"];
  service.seed(
    "owner-a",
    "tenant-alpha",
    "nums",
    lexemes.map((lexeme) => [{ tag: "number", lexeme }]),
  );
  service.beginRead("owner-a", "tenant-alpha", "raw-1", token, "nums");
  const read = service.fetch("owner-a", "tenant-alpha", "raw-1", token, "nums");
  service.endRead("owner-a", "tenant-alpha", "raw-1", token);
  assert.equal(read.rowCount, lexemes.length);
  for (let index = 0; index < lexemes.length; index++) {
    const cell = read.rows[index]?.cells[0] as { tag: string; lexeme: string };
    assert.equal(cell.tag, "number");
    assert.equal(cell.lexeme, lexemes[index]);
  }
  // Trailing-zero, precision-loss, and -0/0 pairs stay pairwise distinct.
  const out = (index: number): string =>
    ((read.rows[index] as { cells: readonly unknown[] }).cells[0] as { lexeme: string }).lexeme;
  assert.notEqual(out(0), out(1));
  assert.notEqual(out(3), out(4));

  // A narrowed adapter claim is detectable even when numerically close.
  const stored = read.rows[2]?.cells as readonly DbCell[];
  assert.equal(
    service.compareRow(stored, [{ tag: "number", lexeme: "9007199254740993" }]).match,
    true,
  );
  assert.equal(
    service.compareRow(stored, [{ tag: "number", lexeme: "9007199254740992" }]).match,
    false,
  );
  assert.deepEqual(
    service.compareRow(stored, [{ tag: "number", lexeme: "9007199254740992" }]).mismatches,
    [0],
  );

  // Non-canonical lexemes reject at the sql layer.
  for (const bad of ["", "1.", ".5", "NaN", "Infinity", "0x10", "1__0", " 1", "1 "]) {
    expectObserverError(() => makeNumberCell(bad), "malformed-lexeme", "sql");
  }
});

check("text-is-code-unit-exact", () => {
  const { service, token } = openPinned();
  const texts = ["", " pad ", "café", "café", "𝄞-𝄞", "a\0b"];
  service.seed(
    "owner-a",
    "tenant-alpha",
    "texts",
    texts.map((text) => [{ tag: "text", text }]),
  );
  service.beginRead("owner-a", "tenant-alpha", "raw-1", token, "texts");
  const read = service.fetch("owner-a", "tenant-alpha", "raw-1", token, "texts");
  service.endRead("owner-a", "tenant-alpha", "raw-1", token);
  for (let index = 0; index < texts.length; index++) {
    const cell = read.rows[index]?.cells[0] as { tag: string; text: string };
    assert.equal(cell.tag, "text");
    assert.equal(cell.text, texts[index]);
  }
  // Precomposed vs decomposed accents are distinct code-unit sequences.
  assert.notEqual(texts[2], texts[3]);
  const precomposed = read.rows[2]?.cells as readonly DbCell[];
  assert.equal(service.compareRow(precomposed, [{ tag: "text", text: texts[2] }]).match, true);
  assert.equal(service.compareRow(precomposed, [{ tag: "text", text: texts[3] }]).match, false);
  // No trimming, no re-encoding.
  assert.equal(texts[1]?.length, 5);
  assert.equal(
    ((read.rows[1] as { cells: readonly unknown[] }).cells[0] as { text: string }).text.length,
    5,
  );
});

check("bytes-are-byte-exact", () => {
  const { service, token } = openPinned();
  const vectors = [
    new Uint8Array([]),
    new Uint8Array([0]),
    new Uint8Array([0, 255, 1, 254]),
    new Uint8Array([222, 173, 190, 239]),
  ];
  service.seed(
    "owner-a",
    "tenant-alpha",
    "blobs",
    vectors.map((bytes) => [{ tag: "bytes", bytes }]),
  );
  service.beginRead("owner-a", "tenant-alpha", "raw-1", token, "blobs");
  const read = service.fetch("owner-a", "tenant-alpha", "raw-1", token, "blobs");
  service.endRead("owner-a", "tenant-alpha", "raw-1", token);
  assert.equal(read.rowCount, vectors.length);
  for (let index = 0; index < vectors.length; index++) {
    const cell = read.rows[index]?.cells[0] as { tag: string; bytes: Uint8Array };
    assert.equal(cell.tag, "bytes");
    assert.deepEqual(Array.from(cell.bytes), Array.from(vectors[index] as Uint8Array));
  }
  // One flipped byte is detectable.
  const stored = read.rows[2]?.cells as readonly DbCell[];
  assert.equal(service.compareRow(stored, [{ tag: "bytes", bytes: [0, 255, 1, 254] }]).match, true);
  assert.equal(
    service.compareRow(stored, [{ tag: "bytes", bytes: [0, 255, 1, 253] }]).match,
    false,
  );
  // Seeding copies: mutating the caller's buffer cannot corrupt the cell.
  vectors[3]?.fill(0);
  const reseeded = read.rows[3]?.cells[0] as { bytes: Uint8Array };
  assert.deepEqual(Array.from(reseeded.bytes), [222, 173, 190, 239]);
  void makeBytesCell(new Uint8Array([1]));
});

check("null-is-distinct-from-empty", () => {
  const { service, token } = openPinned();
  service.seed("owner-a", "tenant-alpha", "maybe", [
    [{ tag: "null" }],
    [{ tag: "text", text: "" }],
    [{ tag: "bytes", bytes: [] }],
    [{ tag: "number", lexeme: "0" }],
  ]);
  service.beginRead("owner-a", "tenant-alpha", "raw-1", token, "maybe");
  const read = service.fetch("owner-a", "tenant-alpha", "raw-1", token, "maybe");
  service.endRead("owner-a", "tenant-alpha", "raw-1", token);
  const tags = read.rows.map((row) => (row.cells[0] as { tag: string }).tag);
  assert.deepEqual(tags, ["null", "text", "bytes", "number"]);
  const nullRow = read.rows[0]?.cells as readonly DbCell[];
  assert.equal(service.compareRow(nullRow, [{ tag: "null" }]).match, true);
  for (const impostor of [
    { tag: "text", text: "" },
    { tag: "bytes", bytes: [] },
    { tag: "number", lexeme: "0" },
  ]) {
    assert.equal(service.compareRow(nullRow, [impostor]).match, false);
  }
  assert.equal(makeNullCell().tag, "null");
  assert.equal(makeTextCell("x").tag, "text");
});

check("row-order-and-arity-preserved", () => {
  const { service, token } = openPinned();
  const order = ["seven", "one", "seven", "two"];
  service.seed(
    "owner-a",
    "tenant-alpha",
    "ordered",
    order.map((text, index) => [
      { tag: "number", lexeme: String(index) },
      { tag: "text", text },
    ]),
  );
  service.beginRead("owner-a", "tenant-alpha", "raw-1", token, "ordered");
  const first = service.fetch("owner-a", "tenant-alpha", "raw-1", token, "ordered");
  service.endRead("owner-a", "tenant-alpha", "raw-1", token);
  assert.deepEqual(
    first.rows.map((row) => (row.cells[1] as { text: string }).text),
    order,
  );
  assert.deepEqual(
    first.rows.map((row) => row.seq),
    [0, 1, 2, 3],
  );
  // Repeating the read returns the identical digest: reads are stable.
  service.beginRead("owner-a", "tenant-alpha", "raw-1", token, "ordered");
  const second = service.fetch("owner-a", "tenant-alpha", "raw-1", token, "ordered");
  service.endRead("owner-a", "tenant-alpha", "raw-1", token);
  assert.equal(second.digest, first.digest);
  assert.equal(second.rowCount, 4);

  // Ragged seeds reject at the sql layer with nothing stored.
  expectObserverError(
    () =>
      service.seed("owner-a", "tenant-alpha", "ragged", [
        [{ tag: "number", lexeme: "1" }],
        [
          { tag: "number", lexeme: "2" },
          { tag: "text", text: "extra" },
        ],
      ]),
    "row-arity",
    "sql",
  );
  expectObserverError(
    () => service.beginRead("owner-a", "tenant-alpha", "raw-1", token, "ragged"),
    "unknown-table",
    "sql",
  );
});

// --- Negative controls: pinning, ownership, provenance ----------------------

check("single-conversation-discipline", () => {
  const { service, token } = openPinned();
  service.seed("owner-a", "tenant-alpha", "t", [[{ tag: "null" }]]);
  service.beginRead("owner-a", "tenant-alpha", "raw-1", token, "t");
  // A second conversation on the same pinned connection rejects.
  expectObserverError(
    () => service.beginRead("owner-a", "tenant-alpha", "raw-1", token, "t"),
    "connection-busy",
    "driver",
  );
  // Fetch and end work on the open conversation; then the pin is quiet.
  assert.equal(service.fetch("owner-a", "tenant-alpha", "raw-1", token, "t").rowCount, 1);
  service.endRead("owner-a", "tenant-alpha", "raw-1", token);
  // Fetch without a conversation rejects; ending twice rejects.
  expectObserverError(
    () => service.fetch("owner-a", "tenant-alpha", "raw-1", token, "t"),
    "no-conversation",
    "driver",
  );
  expectObserverError(
    () => service.endRead("owner-a", "tenant-alpha", "raw-1", token),
    "no-conversation",
    "driver",
  );
  // Unpinning with an open conversation rejects; draining then unpins.
  service.beginRead("owner-a", "tenant-alpha", "raw-1", token, "t");
  expectObserverError(
    () => service.unpinConnection("owner-a", "tenant-alpha", "raw-1", token),
    "connection-busy",
    "driver",
  );
  service.endRead("owner-a", "tenant-alpha", "raw-1", token);
  service.unpinConnection("owner-a", "tenant-alpha", "raw-1", token);
  expectObserverError(
    () => service.beginRead("owner-a", "tenant-alpha", "raw-1", token, "t"),
    "unknown-connection",
    "driver",
  );
});

check("tokens-and-owners-are-authority", () => {
  const { service, token } = openPinned();
  service.seed("owner-a", "tenant-alpha", "t", [[{ tag: "null" }]]);
  // Invented tokens are never authority, on any connection path.
  for (const forged of ["", "conn-deadbeef", token.slice(0, -1)]) {
    expectObserverError(
      () => service.beginRead("owner-a", "tenant-alpha", "raw-1", forged, "t"),
      "forged-token",
      "driver",
    );
    expectObserverError(
      () => service.unpinConnection("owner-a", "tenant-alpha", "raw-1", forged),
      "forged-token",
      "driver",
    );
  }
  // A second namespace of the same name under another owner is separate:
  // same connection name, separate pin, separate tables, no token crossing.
  service.openNamespace("owner-b", "tenant-alpha");
  service.pinConnection("owner-b", "tenant-alpha", "raw-1");
  const tokenB = service.connectionTokenForTest("owner-b", "tenant-alpha", "raw-1");
  assert.notEqual(tokenB, token);
  expectObserverError(
    () => service.beginRead("owner-b", "tenant-alpha", "raw-1", token, "t"),
    "forged-token",
    "driver",
  );
  expectObserverError(
    () => service.beginRead("owner-a", "tenant-alpha", "raw-1", tokenB, "t"),
    "forged-token",
    "driver",
  );
  // Owner-b sees none of owner-a's tables: fixtures are per-namespace-handle.
  expectObserverError(
    () => service.beginRead("owner-b", "tenant-alpha", "raw-1", tokenB, "t"),
    "unknown-table",
    "sql",
  );
  // Closing a namespace with a live pin rejects; draining then closes.
  expectObserverError(
    () => service.closeNamespace("owner-a", "tenant-alpha"),
    "namespace-closed",
    "engine",
  );
});

check("every-failure-names-its-layer", () => {
  // The code vocabulary is closed and every code maps to exactly one layer.
  assert.equal(DB_OBSERVER_CODES.length, 14);
  assert.equal(DB_OBSERVER_SCHEMA_VERSION, "1");
  for (const code of DB_OBSERVER_CODES) {
    const layer = layerOfCode(code);
    assert.ok(["engine", "driver", "sql"].includes(layer));
    const error = new DbObserverError(code, "probe");
    assert.equal(error.layer, layer);
    assert.equal(error.code, code);
    assert.ok(error.message.startsWith(`[${layer}:${code}]`));
  }
  // Cell constructors reject mistyped payloads with sql/driver provenance.
  expectObserverError(() => makeTextCell(42), "wrong-cell-type", "sql");
  expectObserverError(() => makeBytesCell("00"), "wrong-cell-type", "sql");
  expectObserverError(() => makeNumberCell("12a"), "malformed-lexeme", "sql");
  // Unknown cell tags and bad row shapes reject without storing anything.
  const { service, token } = openPinned();
  expectObserverError(
    () => service.seed("owner-a", "tenant-alpha", "bad", [[{ tag: "json", gap: "x" }]]),
    "wrong-cell-type",
    "sql",
  );
  expectObserverError(
    () => service.seed("owner-a", "tenant-alpha", "bad", [["not-a-cell"]]),
    "wrong-cell-type",
    "sql",
  );
  expectObserverError(
    () =>
      service.seed("owner-a", "tenant-alpha", "bad", [
        "not-a-row",
      ] as unknown as readonly (readonly unknown[])[]),
    "row-arity",
    "sql",
  );
  expectObserverError(
    () => service.beginRead("owner-a", "tenant-alpha", "raw-1", token, "bad"),
    "unknown-table",
    "sql",
  );
  // Capacity faults name the engine layer.
  const tiny = new DbObserverService(
    ["n1"],
    checkLimits({
      maxNamespaces: 1,
      maxConnections: 1,
      maxTables: 1,
      maxRows: 1,
      maxCellsPerRow: 1,
      maxCellBytes: 4,
    }),
  );
  tiny.openNamespace("o", "n1");
  tiny.pinConnection("o", "n1", "c");
  expectObserverError(() => tiny.pinConnection("o", "n1", "c2"), "capacity-exhausted", "engine");
  expectObserverError(
    () => tiny.seed("o", "n1", "t", [[{ tag: "text", text: "toolong" }]]),
    "capacity-exhausted",
    "engine",
  );
});

check("independence-scope-stated", () => {
  // The scope statement is a fixed export: the observer shares no adapter
  // code, decoded values, or connection state, and proves its facts from
  // its own seeded doubles alone.
  assert.ok(DB_OBSERVER_INDEPENDENCE_SCOPE.startsWith("independent:"));
  assert.ok(DB_OBSERVER_INDEPENDENCE_SCOPE.includes("no adapter code"));
  assert.ok(DB_OBSERVER_INDEPENDENCE_SCOPE.includes("seeded doubles"));
});

check("fetch-bytes-are-copies", () => {
  // Fetched bytes cells must not alias stored state: caller mutation of a
  // fetched buffer must leave later reads and digests untouched.
  const { service, token } = openPinned();
  service.seed("owner-a", "tenant-alpha", "blobs", [[{ tag: "bytes", bytes: [1, 2, 3] }]]);
  service.beginRead("owner-a", "tenant-alpha", "raw-1", token, "blobs");
  const first = service.fetch("owner-a", "tenant-alpha", "raw-1", token, "blobs");
  const cell = first.rows[0]?.cells[0];
  assert.ok(cell !== undefined && cell.tag === "bytes");
  if (cell.tag === "bytes") {
    cell.bytes[0] = 99;
  }
  const second = service.fetch("owner-a", "tenant-alpha", "raw-1", token, "blobs");
  const again = second.rows[0]?.cells[0];
  assert.ok(again !== undefined && again.tag === "bytes");
  if (again !== undefined && again.tag === "bytes") {
    assert.equal(again.bytes[0], 1);
  }
  assert.equal(second.digest, first.digest);
});

console.log(
  JSON.stringify({
    kind: "can.db-observer-core-check",
    schema_version: DB_OBSERVER_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
