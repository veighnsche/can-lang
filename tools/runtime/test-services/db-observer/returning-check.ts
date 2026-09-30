// K23 bounded self-check: raw RETURNING and final-row observations.
//
// Run with: bun tools/runtime/test-services/db-observer/returning-check.ts
// Local controls only. No databases, drivers, sockets, files, timers, live
// hosts, or live runtimes. Every loop below is bounded by a small constant.
// Live host-dependent controls wait for the qualified profile and Q task.
// QD1 holds D1 credit; nothing here claims execution credit.

import { strict as assert } from "node:assert";
import {
  checkLimits,
  DbObserverError,
  type DbCell,
  type DbObserverCode,
  type DbObserverLayer,
  type DbObserverLimits,
} from "./core.ts";
import {
  RETURNING_CREDIT_REQUIREMENTS,
  RETURNING_MAX_COMPILES,
  RETURNING_NON_CREDIT_BOUNDARY,
  RETURNING_OBSERVER_SCHEMA_VERSION,
  ReturningObserver,
} from "./returning.ts";

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

function openObserver(): ReturningObserver {
  return new ReturningObserver(roomyLimits());
}

// --- Compile records beside an ordinary C fixture ---------------------------

check("compile-record-pins-schema", () => {
  const observer = openObserver();
  const record = observer.recordCompile(
    "c-fixture-d1",
    "INSERT INTO t (id, name) VALUES (1, 'a')",
    ["number", "text"],
  );
  assert.equal(record.compile, "compile-0");
  assert.equal(record.fixture, "c-fixture-d1");
  assert.ok(record.statementDigest.startsWith("sha256:"));
  assert.equal(record.statementDigest.length, "sha256:".length + 64);
  assert.deepEqual(Array.from(record.schema), ["number", "text"]);
  // Facts carry a digest only for the statement; the pin is exact tags.
  assert.deepEqual(Object.keys(record).sort(), ["compile", "fixture", "schema", "statementDigest"]);

  // The same statement under a second compile pins independently.
  const again = observer.recordCompile("c-fixture-d1", "INSERT INTO t (id, name) VALUES (1, 'a')", [
    "number",
    "text",
  ]);
  assert.equal(again.compile, "compile-1");
  assert.equal(again.statementDigest, record.statementDigest);
  assert.equal(observer.compileCount, 2);
  assert.equal(observer.compileRecord("compile-0").fixture, "c-fixture-d1");

  // Malformed compile records reject at the sql layer with nothing stored.
  expectObserverError(
    () => observer.recordCompile("", "SELECT 1", ["number"]),
    "malformed-statement",
    "sql",
  );
  expectObserverError(
    () => observer.recordCompile("bad fixture!", "SELECT 1", ["number"]),
    "malformed-statement",
    "sql",
  );
  expectObserverError(
    () => observer.recordCompile("c-ok", "", ["number"]),
    "malformed-statement",
    "sql",
  );
  expectObserverError(
    () => observer.recordCompile("c-ok", "SELECT 1", []),
    "malformed-statement",
    "sql",
  );
  expectObserverError(
    () => observer.recordCompile("c-ok", "SELECT 1", ["json"]),
    "malformed-statement",
    "sql",
  );
  expectObserverError(() => observer.compileRecord("not-a-compile"), "malformed-statement", "sql");
  expectObserverError(() => observer.compileRecord("compile-99"), "unknown-connection", "driver");
  assert.equal(observer.compileCount, 2);

  // The compile table is bounded.
  const tight = new ReturningObserver(roomyLimits());
  for (let index = 0; index < RETURNING_MAX_COMPILES; index++) {
    tight.recordCompile(`c-${index}`, "SELECT 1", ["null"]);
  }
  expectObserverError(
    () => tight.recordCompile("c-overflow", "SELECT 1", ["null"]),
    "capacity-exhausted",
    "engine",
  );
});

// --- RETURNING payload rows: row count plus typed exact rows -----------------

check("returning-payload-exact-rows", () => {
  const observer = openObserver();
  const record = observer.recordCompile("c-fixture-d1", "INSERT INTO t (id, name) RETURNING id", [
    "number",
    "text",
    "bytes",
    "null",
  ]);
  const payload = observer.recordReturning(record.compile, [
    [
      { tag: "number", lexeme: "9007199254740993" },
      { tag: "text", text: "café" },
      { tag: "bytes", bytes: [0, 255, 1] },
      { tag: "null" },
    ],
    [
      { tag: "number", lexeme: "-0" },
      { tag: "text", text: "" },
      { tag: "bytes", bytes: [] },
      { tag: "null" },
    ],
    [
      { tag: "number", lexeme: "1.10" },
      { tag: "text", text: " pad " },
      { tag: "bytes", bytes: new Uint8Array([222, 173]) },
      { tag: "null" },
    ],
  ]);
  assert.equal(payload.compile, record.compile);
  assert.equal(payload.rowCount, 3);
  assert.equal(payload.rows.length, 3);
  assert.deepEqual(
    payload.rows.map((row) => row.seq),
    [0, 1, 2],
  );
  // Exact cells cross verbatim: lexemes, code units, bytes, null.
  const first = payload.rows[0]?.cells as readonly DbCell[];
  assert.equal((first[0] as { lexeme: string }).lexeme, "9007199254740993");
  assert.equal((first[1] as { text: string }).text, "café");
  assert.deepEqual(Array.from((first[2] as { bytes: Uint8Array }).bytes), [0, 255, 1]);
  assert.equal(first[3]?.tag, "null");
  // Trailing zeros and -0 survive: no float coercion anywhere.
  const third = payload.rows[2]?.cells[0] as { lexeme: string };
  assert.equal(third.lexeme, "1.10");

  // Re-reading returns the identical digest: payload facts are stable.
  const reread = observer.payloadFacts(record.compile);
  assert.equal(reread.digest, payload.digest);
  assert.equal(reread.rowCount, 3);
  assert.deepEqual(
    reread.rows.map((row) => row.digest),
    payload.rows.map((row) => row.digest),
  );

  // Malformed payload rows reject with the payload left unrecorded.
  const bare = openObserver();
  const pin = bare.recordCompile("c-fixture-d1", "INSERT INTO t (a) RETURNING a", ["number"]);
  expectObserverError(
    () => bare.recordReturning(pin.compile, [[{ tag: "number", lexeme: "12a" }]]),
    "malformed-lexeme",
    "sql",
  );
  expectObserverError(
    () => bare.recordReturning(pin.compile, [["not-a-cell"]]),
    "wrong-cell-type",
    "sql",
  );
  // Missing evidence stays missing: nothing was stored by the failures.
  expectObserverError(() => bare.payloadFacts(pin.compile), "no-conversation", "driver");
});

// --- Final-row read-out: the observer's own independent re-read --------------

check("final-row-readout", () => {
  const observer = openObserver();
  const record = observer.recordCompile("c-fixture-d1", "INSERT INTO t (id) RETURNING id", [
    "number",
  ]);
  // No final rows yet: the read-out is explicitly missing, never empty-pass.
  expectObserverError(() => observer.finalFacts(record.compile), "no-conversation", "driver");

  const payload = observer.recordReturning(record.compile, [[{ tag: "number", lexeme: "7" }]]);
  const final = observer.recordFinalRows(record.compile, [[{ tag: "number", lexeme: "7" }]]);
  assert.equal(final.compile, record.compile);
  assert.equal(final.rowCount, 1);
  assert.equal(final.rows[0]?.seq, 0);
  const finalCell = final.rows[0]?.cells[0] as { lexeme: string };
  assert.equal(finalCell.lexeme, "7");

  // Payload and final read-out agree exactly: same digest, match verdict.
  assert.equal(final.digest, payload.digest);
  const comparison = observer.comparePayloadToFinal(payload, final);
  assert.equal(comparison.match, true);
  assert.deepEqual(Array.from(comparison.mismatches), []);
});

// --- Non-credit boundary: raw RETURNING alone must never credit C ------------

check("raw-returning-alone-never-credits-c", () => {
  const observer = openObserver();
  const record = observer.recordCompile("c-fixture-d1", "INSERT INTO t (id) RETURNING id", [
    "number",
  ]);
  const payload = observer.recordReturning(record.compile, [[{ tag: "number", lexeme: "1" }]]);
  const final = observer.recordFinalRows(record.compile, [[{ tag: "number", lexeme: "1" }]]);
  // Even a fully agreeing payload/final pair is raw evidence only.
  assert.equal(observer.comparePayloadToFinal(payload, final).match, true);

  const verdict = observer.creditVerdict();
  assert.equal(verdict.creditC, false);
  assert.equal(verdict.reason, RETURNING_NON_CREDIT_BOUNDARY);
  assert.deepEqual(Object.keys(verdict).sort(), ["creditC", "reason"]);

  // The boundary is a fixed export naming QD1 as the credit holder.
  assert.ok(RETURNING_NON_CREDIT_BOUNDARY.startsWith("non-credit:"));
  assert.ok(RETURNING_NON_CREDIT_BOUNDARY.includes("alone cannot credit"));
  assert.ok(RETURNING_NON_CREDIT_BOUNDARY.includes("QD1"));
  assert.ok(RETURNING_NON_CREDIT_BOUNDARY.includes("never claims execution credit"));
  assert.deepEqual(Array.from(RETURNING_CREDIT_REQUIREMENTS), [
    "c-side-witness",
    "final-row-agreement",
    "qd1-verdict",
  ]);
  assert.equal(RETURNING_OBSERVER_SCHEMA_VERSION, "1");
});

// --- Bigint narrowing is detectable ------------------------------------------

check("bigint-narrowing-detectable", () => {
  const observer = openObserver();
  const record = observer.recordCompile("c-fixture-d1", "INSERT INTO t (id) RETURNING id", [
    "number",
  ]);
  const payload = observer.recordReturning(record.compile, [
    [{ tag: "number", lexeme: "9007199254740993" }],
    [{ tag: "number", lexeme: "1.10" }],
    [{ tag: "number", lexeme: "-0" }],
  ]);
  const bigint = payload.rows[0]?.cells as readonly DbCell[];
  // The exact lexeme matches; every narrowed or coerced claim mismatches.
  assert.equal(
    observer.compareClaimedRow(bigint, [{ tag: "number", lexeme: "9007199254740993" }]).match,
    true,
  );
  for (const narrowed of ["9007199254740992", "9007199254740994", "9.007199254740992e15"]) {
    const result = observer.compareClaimedRow(bigint, [{ tag: "number", lexeme: narrowed }]);
    assert.equal(result.match, false);
    assert.deepEqual(Array.from(result.mismatches), [0]);
  }
  // Decimal coercion and -0/0 confusion are detectable too.
  const decimal = payload.rows[1]?.cells as readonly DbCell[];
  assert.equal(
    observer.compareClaimedRow(decimal, [{ tag: "number", lexeme: "1.1" }]).match,
    false,
  );
  const minusZero = payload.rows[2]?.cells as readonly DbCell[];
  assert.equal(
    observer.compareClaimedRow(minusZero, [{ tag: "number", lexeme: "0" }]).match,
    false,
  );
  assert.equal(
    observer.compareClaimedRow(minusZero, [{ tag: "number", lexeme: "-0" }]).match,
    true,
  );
  // A mistyped claim cell is a mismatch, never a throw.
  assert.equal(observer.compareClaimedRow(bigint, [{ tag: "text", text: "x" }]).match, false);
  assert.equal(observer.compareClaimedRow(bigint, []).match, false);
  assert.deepEqual(Array.from(observer.compareClaimedRow(bigint, []).mismatches), [-1]);
});

// --- Wrong row schema is detectable ------------------------------------------

check("wrong-row-schema-detectable", () => {
  const observer = openObserver();
  const record = observer.recordCompile("c-fixture-d1", "INSERT INTO t (id, name) RETURNING id", [
    "number",
    "text",
  ]);
  // Ragged rows reject against the pin at record time.
  expectObserverError(
    () => observer.recordReturning(record.compile, [[{ tag: "number", lexeme: "1" }]]),
    "row-arity",
    "sql",
  );
  expectObserverError(
    () =>
      observer.recordReturning(record.compile, [
        [{ tag: "number", lexeme: "1" }, { tag: "text", text: "a" }, { tag: "null" }],
      ]),
    "row-arity",
    "sql",
  );
  // Swapped column types reject: the pin is tag-exact per column.
  expectObserverError(
    () =>
      observer.recordReturning(record.compile, [
        [
          { tag: "text", text: "1" },
          { tag: "text", text: "a" },
        ],
      ]),
    "wrong-cell-type",
    "sql",
  );
  expectObserverError(
    () =>
      observer.recordFinalRows(record.compile, [[{ tag: "number", lexeme: "1" }, { tag: "null" }]]),
    "wrong-cell-type",
    "sql",
  );
  // Null is distinct from empty text even in claimed-row comparison.
  const payload = observer.recordReturning(record.compile, [
    [
      { tag: "number", lexeme: "1" },
      { tag: "text", text: "" },
    ],
  ]);
  const stored = payload.rows[0]?.cells as readonly DbCell[];
  assert.equal(
    observer.compareClaimedRow(stored, [
      { tag: "number", lexeme: "1" },
      { tag: "text", text: "" },
    ]).match,
    true,
  );
  assert.equal(
    observer.compareClaimedRow(stored, [{ tag: "number", lexeme: "1" }, { tag: "null" }]).match,
    false,
  );
});

// --- Payload/final divergence is detectable ----------------------------------

check("payload-final-divergence-detectable", () => {
  const observer = openObserver();
  const record = observer.recordCompile("c-fixture-d1", "INSERT INTO t (id, b) RETURNING id", [
    "number",
    "bytes",
  ]);
  const payload = observer.recordReturning(record.compile, [
    [
      { tag: "number", lexeme: "1" },
      { tag: "bytes", bytes: [0, 255, 1, 254] },
    ],
    [
      { tag: "number", lexeme: "2" },
      { tag: "bytes", bytes: [9] },
    ],
  ]);
  // One flipped byte in row 0 diverges; the mismatch names the row.
  const final = observer.recordFinalRows(record.compile, [
    [
      { tag: "number", lexeme: "1" },
      { tag: "bytes", bytes: [0, 255, 1, 253] },
    ],
    [
      { tag: "number", lexeme: "2" },
      { tag: "bytes", bytes: [9] },
    ],
  ]);
  assert.notEqual(final.digest, payload.digest);
  const comparison = observer.comparePayloadToFinal(payload, final);
  assert.equal(comparison.match, false);
  assert.deepEqual(Array.from(comparison.mismatches), [0]);

  // A row-count gap between payload and final read-out mismatches too.
  const short = observer.recordFinalRows(record.compile, [
    [
      { tag: "number", lexeme: "1" },
      { tag: "bytes", bytes: [0, 255, 1, 254] },
    ],
  ]);
  assert.deepEqual(Array.from(observer.comparePayloadToFinal(payload, short).mismatches), [-1]);

  // Facts from different compiles never compare: cross-compile verdicts throw.
  const other = observer.recordCompile("c-fixture-d1", "INSERT INTO t (id, b) RETURNING id", [
    "number",
    "bytes",
  ]);
  const otherFinal = observer.recordFinalRows(other.compile, [
    [
      { tag: "number", lexeme: "1" },
      { tag: "bytes", bytes: [0, 255, 1, 254] },
    ],
    [
      { tag: "number", lexeme: "2" },
      { tag: "bytes", bytes: [9] },
    ],
  ]);
  expectObserverError(
    () => observer.comparePayloadToFinal(payload, otherFinal),
    "malformed-statement",
    "sql",
  );
});

// --- Payload bytes are copies; every failure names its layer -----------------

check("payload-bytes-are-copies", () => {
  const observer = openObserver();
  const record = observer.recordCompile("c-fixture-d1", "INSERT INTO t (b) RETURNING b", ["bytes"]);
  const buffer = new Uint8Array([1, 2, 3]);
  const payload = observer.recordReturning(record.compile, [[{ tag: "bytes", bytes: buffer }]]);
  // Mutating the caller's buffer cannot corrupt the recorded payload.
  buffer.fill(0);
  const reread = observer.payloadFacts(record.compile);
  const rereadBytes = reread.rows[0]?.cells[0] as { bytes: Uint8Array };
  assert.deepEqual(Array.from(rereadBytes.bytes), [1, 2, 3]);
  assert.equal(reread.digest, payload.digest);

  // Mutating a returned facts buffer cannot corrupt later reads either.
  const cell = payload.rows[0]?.cells[0];
  assert.ok(cell !== undefined && cell.tag === "bytes");
  if (cell.tag === "bytes") {
    cell.bytes[0] = 99;
  }
  const again = observer.payloadFacts(record.compile);
  const againBytes = again.rows[0]?.cells[0] as { bytes: Uint8Array };
  assert.equal(againBytes.bytes[0], 1);
  assert.equal(again.digest, payload.digest);
});

check("every-failure-names-its-layer", () => {
  // Row and statement caps name the engine layer.
  const observer = openObserver();
  expectObserverError(
    () =>
      observer.recordCompile("c-ok", "SELECT 1", [
        "null",
        "null",
        "null",
        "null",
        "null",
        "null",
        "null",
        "null",
        "null",
      ]),
    "capacity-exhausted",
    "engine",
  );
  const record = observer.recordCompile("c-ok", "SELECT 1", ["null"]);
  const tooMany: readonly (readonly unknown[])[] = Array.from({ length: 17 }, () => [
    { tag: "null" },
  ]);
  expectObserverError(
    () => observer.recordReturning(record.compile, tooMany),
    "capacity-exhausted",
    "engine",
  );
  expectObserverError(
    () => observer.recordFinalRows(record.compile, tooMany),
    "capacity-exhausted",
    "engine",
  );
  expectObserverError(
    () => observer.recordCompile("c-ok", `SELECT '${"x".repeat(5000)}'`, ["null"]),
    "capacity-exhausted",
    "engine",
  );
  // Oversized text cells name the engine layer; mistyped cells name sql.
  expectObserverError(
    () => observer.recordReturning("compile-99", [[{ tag: "text", text: "x".repeat(2000) }]]),
    "unknown-connection",
    "driver",
  );
  const textPin = observer.recordCompile("c-text", "SELECT 1", ["text"]);
  expectObserverError(
    () => observer.recordReturning(textPin.compile, [[{ tag: "text", text: "x".repeat(2000) }]]),
    "capacity-exhausted",
    "engine",
  );
  // Malformed row containers reject without recording anything.
  expectObserverError(
    () =>
      observer.recordReturning(textPin.compile, [
        "not-a-row",
      ] as unknown as readonly (readonly unknown[])[]),
    "row-arity",
    "sql",
  );
  expectObserverError(() => observer.payloadFacts(textPin.compile), "no-conversation", "driver");
});

console.log(
  JSON.stringify({
    kind: "can.db-observer-returning-check",
    schema_version: RETURNING_OBSERVER_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
