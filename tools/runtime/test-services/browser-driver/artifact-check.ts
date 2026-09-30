// K15 bounded self-check: Can browser artifacts and export facts.
//
// Run with: bun tools/runtime/test-services/browser-driver/artifact-check.ts
// Local controls only. No browsers, processes, flags, environment, sampling,
// networks, files, timers, live hosts, or live runtimes. Every loop below
// is bounded by a small constant. Live host-dependent controls wait for the
// qualified profile and Q task.

import { strict as assert } from "node:assert";
import {
  assertExportProved,
  BROWSER_ARTIFACT_CODES,
  BROWSER_ARTIFACT_INDEPENDENCE_SCOPE,
  BROWSER_ARTIFACT_SCHEMA_VERSION,
  BrowserArtifactError,
  BrowserArtifactService,
  checkArtifactLimits,
  digestExport,
  FORBIDDEN_EXPORT_KINDS,
  layerOfArtifactCode,
  READY_STATES,
  REPRESENTATIONS,
  type BrowserArtifactCode,
  type BrowserArtifactLayer,
  type BrowserArtifactLimits,
} from "./artifact.ts";

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectArtifactError(
  body: () => unknown,
  code: BrowserArtifactCode,
  layer: BrowserArtifactLayer,
): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof BrowserArtifactError)) {
      assert.fail(`expected a BrowserArtifactError, got ${String(error)}`);
    }
    assert.equal(error.code, code);
    assert.equal(error.layer, layer);
    assert.ok(error.message.includes(`[${layer}:${code}]`));
    return;
  }
  assert.fail(`expected failure ${layer}:${code}`);
}

const MODULE_DIGEST = `sha256:${"cd".repeat(32)}`;
const BYTES_DIGEST = `sha256:${"ef".repeat(32)}`;

function roomyLimits(): BrowserArtifactLimits {
  return checkArtifactLimits({
    maxArtifacts: 4,
    maxModulesPerArtifact: 4,
    maxRowsPerExport: 8,
    maxCellsPerRow: 4,
    maxExportsPerModule: 8,
  });
}

function openService(): BrowserArtifactService {
  const service = new BrowserArtifactService(
    [
      { owner: "owner-n", artifact: "can-app", modules: ["main", "worker"] },
      { owner: "owner-n", artifact: "can-shell", modules: ["shell"] },
    ],
    roomyLimits(),
  );
  service.leaseArtifact("owner-n", "can-app");
  return service;
}

function loadDescriptor(module = "main"): {
  module: string;
  moduleDigest: string;
  byteLength: number;
} {
  return { module, moduleDigest: MODULE_DIGEST, byteLength: 4096 };
}

function openModule(
  module = "main",
  artifact = "can-app",
): { service: BrowserArtifactService; lease: string; token: string } {
  const service = openService();
  const lease = service.leaseTokenForTest("owner-n", artifact);
  service.loadModule("owner-n", artifact, lease, loadDescriptor(module));
  const token = service.moduleTokenForTest("owner-n", artifact, module);
  return { service, lease, token };
}

function openReady(
  module = "main",
  artifact = "can-app",
): { service: BrowserArtifactService; lease: string; token: string } {
  const opened = openModule(module, artifact);
  opened.service.markReady("owner-n", artifact, module, opened.token, "complete");
  return opened;
}

function exportDescriptor(representation = "integer-lexeme"): {
  representation: string;
  rows: { cells: { column: string; value: string }[] }[];
} {
  const value =
    representation === "ieee-bits"
      ? "0x3ff8000000000000"
      : representation === "utf8-text"
        ? "hello-can"
        : representation === "bytes-digest"
          ? BYTES_DIGEST
          : "42";
  return { representation, rows: [{ cells: [{ column: "v", value }] }] };
}

// --- Artifact leases and admission-first ------------------------------------

check("artifact-lease-admission", () => {
  const service = new BrowserArtifactService(
    [{ owner: "owner-n", artifact: "can-app", modules: ["main"] }],
    roomyLimits(),
  );
  const lease = service.leaseArtifact("owner-n", "can-app");
  assert.equal(lease.owner, "owner-n");
  assert.equal(lease.artifact, "can-app");
  assert.deepEqual([...lease.modules], ["main"]);
  assert.ok(lease.handleDigest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(lease).sort(), ["artifact", "handleDigest", "modules", "owner"]);

  // Re-leasing joins the same owned lease; the digest is stable.
  const again = service.leaseArtifact("owner-n", "can-app");
  assert.equal(again.handleDigest, lease.handleDigest);

  // Artifacts outside the owned grant reject before any effect.
  for (const foreign of ["can-zzz", "", "can-app/x"]) {
    expectArtifactError(
      () => service.leaseArtifact("owner-n", foreign),
      "unknown-artifact",
      "artifact",
    );
  }
  expectArtifactError(
    () => service.leaseArtifact("owner-x", "can-app"),
    "unknown-artifact",
    "artifact",
  );

  // A second identity granted the same artifact name cannot join it:
  // leases bind to exactly one owner.
  const shared = new BrowserArtifactService(
    [
      { owner: "owner-n", artifact: "can-app", modules: ["main"] },
      { owner: "owner-x", artifact: "can-app", modules: ["main"] },
    ],
    roomyLimits(),
  );
  shared.leaseArtifact("owner-n", "can-app");
  expectArtifactError(() => shared.leaseArtifact("owner-x", "can-app"), "wrong-owner", "artifact");
  assert.equal(service.leaseCount, 1);
});

check("missing-lease-blocks-load", () => {
  const service = new BrowserArtifactService(
    [{ owner: "owner-n", artifact: "can-app", modules: ["main"] }],
    roomyLimits(),
  );
  // Never leased: load refuses without a live lease binding.
  expectArtifactError(
    () => service.loadModule("owner-n", "can-app", "lease-seed", loadDescriptor()),
    "unknown-artifact",
    "artifact",
  );
  // Ungranted artifact: load refuses the same way.
  expectArtifactError(
    () => service.loadModule("owner-n", "elsewhere", "lease-seed", loadDescriptor()),
    "unknown-artifact",
    "artifact",
  );
  // Live lease: the same load admits and pins.
  service.leaseArtifact("owner-n", "can-app");
  const lease = service.leaseTokenForTest("owner-n", "can-app");
  const loaded = service.loadModule("owner-n", "can-app", lease, loadDescriptor());
  assert.equal(loaded.module, "main");
  assert.equal(loaded.moduleDigest, MODULE_DIGEST);
  assert.deepEqual(Object.keys(loaded).sort(), [
    "artifact",
    "byteLength",
    "handleDigest",
    "leaseDigest",
    "module",
    "moduleDigest",
    "owner",
  ]);
});

// --- Digest-pinned module load -------------------------------------------------

check("module-load-pins-bytes", () => {
  const { service, lease } = openModule();
  // Re-loading joins the identical pinned load only.
  const again = service.loadModule("owner-n", "can-app", lease, loadDescriptor());
  assert.equal(again.moduleDigest, MODULE_DIGEST);
  // A differing descriptor refuses rather than swapping pinned bytes.
  const other = `sha256:${"99".repeat(32)}`;
  expectArtifactError(
    () =>
      service.loadModule("owner-n", "can-app", lease, {
        module: "main",
        moduleDigest: other,
        byteLength: 4096,
      }),
    "digest-unknown",
    "load",
  );
  expectArtifactError(
    () => service.loadModule("owner-n", "can-app", lease, { ...loadDescriptor(), byteLength: 1 }),
    "digest-unknown",
    "load",
  );
  // Modules outside the declared grant reject before any effect.
  for (const bad of ["helper", "", "main/x"]) {
    expectArtifactError(
      () => service.loadModule("owner-n", "can-app", lease, loadDescriptor(bad)),
      "unknown-module",
      "load",
    );
  }
  // Digests must pin sha256 bytes; lengths must be positive integers.
  for (const bad of ["", "sha256:zz", "md5:abc", MODULE_DIGEST.slice(0, -1)]) {
    expectArtifactError(
      () =>
        service.loadModule("owner-n", "can-app", lease, {
          ...loadDescriptor("worker"),
          moduleDigest: bad,
        }),
      "digest-unknown",
      "load",
    );
  }
  for (const bad of [0, -1, 1.5, Number.NaN]) {
    expectArtifactError(
      () =>
        service.loadModule("owner-n", "can-app", lease, {
          ...loadDescriptor("worker"),
          byteLength: bad,
        }),
      "digest-unknown",
      "load",
    );
  }
  // The rejected loads left no trace: worker is still unloaded.
  expectArtifactError(
    () => service.moduleTokenForTest("owner-n", "can-app", "worker"),
    "unknown-module",
    "load",
  );
});

// --- Tokens are opaque and verified by lookup ----------------------------------

check("tokens-opaque-and-verified", () => {
  const { service, lease, token } = openModule();
  const descriptor = loadDescriptor("worker");
  // Invented lease tokens are never authority for load.
  for (const forged of ["", "lease-deadbeef", `${lease.slice(0, -1)}x`]) {
    expectArtifactError(
      () => service.loadModule("owner-n", "can-app", forged, descriptor),
      "forged-token",
      "load",
    );
  }
  // Invented module tokens are never authority for ready or export.
  for (const forged of ["", "mod-deadbeef", `${token.slice(0, -1)}x`]) {
    expectArtifactError(
      () => service.markReady("owner-n", "can-app", "main", forged, "complete"),
      "forged-token",
      "load",
    );
    expectArtifactError(
      () => service.exportData("owner-n", "can-app", "main", forged, exportDescriptor()),
      "forged-token",
      "load",
    );
  }
  // Unknown modules reject before the token is even read.
  expectArtifactError(
    () => service.markReady("owner-n", "can-app", "helper", token, "complete"),
    "unknown-module",
    "load",
  );
  // Tokens never transfer between modules: the lease token is not a
  // module token, and one module token is not another's.
  service.loadModule("owner-n", "can-app", lease, descriptor);
  const workerToken = service.moduleTokenForTest("owner-n", "can-app", "worker");
  assert.notEqual(workerToken, token);
  expectArtifactError(
    () => service.markReady("owner-n", "can-app", "worker", token, "complete"),
    "forged-token",
    "load",
  );
  expectArtifactError(
    () => service.markReady("owner-n", "can-app", "main", workerToken, "complete"),
    "forged-token",
    "load",
  );
  expectArtifactError(
    () => service.markReady("owner-n", "can-app", "main", lease, "complete"),
    "forged-token",
    "load",
  );
  // The live tokens work under their own bindings.
  service.markReady("owner-n", "can-app", "main", token, "complete");
  service.markReady("owner-n", "can-app", "worker", workerToken, "interactive");
});

// --- DOM-ready -----------------------------------------------------------------

check("dom-ready-marks-module", () => {
  const { service, token } = openModule();
  assert.deepEqual([...READY_STATES], ["interactive", "complete"]);
  // Readiness states outside the finite vocabulary reject without effect.
  for (const bad of ["loading", "", "COMPLETE", "complete "]) {
    expectArtifactError(
      () => service.markReady("owner-n", "can-app", "main", token, bad),
      "unknown-readiness",
      "load",
    );
  }
  // interactive then complete advances; the receipt names the state.
  const first = service.markReady("owner-n", "can-app", "main", token, "interactive");
  assert.equal(first.domState, "interactive");
  assert.equal(first.ready, true);
  assert.ok(first.digest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(first).sort(), [
    "artifact",
    "digest",
    "domState",
    "module",
    "owner",
    "ready",
  ]);
  const second = service.markReady("owner-n", "can-app", "main", token, "complete");
  assert.equal(second.domState, "complete");
  // Regressing from complete refuses rather than unseeing readiness.
  expectArtifactError(
    () => service.markReady("owner-n", "can-app", "main", token, "interactive"),
    "unknown-readiness",
    "load",
  );
  const facts = service.artifactFacts("owner-n", "can-app");
  assert.equal(facts.modules[0]?.domState, "complete");
  assert.equal(facts.modules[0]?.ready, true);
});

// --- Acceptance: missing ready/export never becomes an empty pass ---------------

check("missing-readiness-refuses-export", () => {
  const { service, token } = openModule();
  // Loaded but never marked ready: every representation refuses with
  // ready-open, and no facts are yielded.
  for (const representation of REPRESENTATIONS) {
    expectArtifactError(
      () =>
        service.exportData("owner-n", "can-app", "main", token, exportDescriptor(representation)),
      "ready-open",
      "load",
    );
  }
  // The refused exports left no recorded export behind.
  expectArtifactError(
    () => service.exportFacts("owner-n", "can-app", "main", 0),
    "export-open",
    "export",
  );
  const facts = service.artifactFacts("owner-n", "can-app");
  assert.equal(facts.modules[0]?.exports, 0);
  // Marking ready unblocks the same export.
  service.markReady("owner-n", "can-app", "main", token, "complete");
  const exported = service.exportData(
    "owner-n",
    "can-app",
    "main",
    token,
    exportDescriptor("integer-lexeme"),
  );
  assert.equal(exported.ready, true);
  assert.equal(exported.rows.length, 1);
});

check("missing-export-refuses-facts", () => {
  const { service, token } = openReady();
  // Ready but never exported: facts reads refuse with export-open.
  for (const index of [0, 1, -1]) {
    expectArtifactError(
      () => service.exportFacts("owner-n", "can-app", "main", index),
      "export-open",
      "export",
    );
  }
  // Empty rows and empty cells refuse as missing exports, never as
  // empty facts that look like success.
  expectArtifactError(
    () =>
      service.exportData("owner-n", "can-app", "main", token, {
        representation: "integer-lexeme",
        rows: [],
      }),
    "export-open",
    "export",
  );
  expectArtifactError(
    () =>
      service.exportData("owner-n", "can-app", "main", token, {
        representation: "integer-lexeme",
        rows: [{ cells: [] }],
      }),
    "export-open",
    "export",
  );
  // Still no recorded export after the refused attempts.
  expectArtifactError(
    () => service.exportFacts("owner-n", "can-app", "main", 0),
    "export-open",
    "export",
  );
  // A real export records exactly one facts entry.
  service.exportData("owner-n", "can-app", "main", token, exportDescriptor());
  const facts = service.exportFacts("owner-n", "can-app", "main", 0);
  assert.equal(facts.rows.length, 1);
  expectArtifactError(
    () => service.exportFacts("owner-n", "can-app", "main", 1),
    "export-open",
    "export",
  );
});

check("unsupported-representation-refuses-with-reason", () => {
  const { service, token } = openReady();
  assert.deepEqual(
    [...REPRESENTATIONS],
    ["integer-lexeme", "ieee-bits", "utf8-text", "bytes-digest"],
  );
  // Representations outside the finite vocabulary refuse with a named
  // reason, and yield no facts that could pass empty.
  for (const bad of ["float-text", "", "INTEGER-LEXEME", "integer-lexeme "]) {
    try {
      service.exportData("owner-n", "can-app", "main", token, {
        representation: bad,
        rows: [{ cells: [{ column: "v", value: "42" }] }],
      });
      assert.fail(`expected failure export:unknown-representation for ${bad}`);
    } catch (error) {
      assert.ok(error instanceof BrowserArtifactError);
      assert.equal(error.code, "unknown-representation");
      assert.equal(error.layer, "export");
      assert.ok(error.message.includes("[export:unknown-representation]"));
      assert.ok(error.message.includes(String(bad)));
      assert.ok(error.message.includes("supported: integer-lexeme, ieee-bits"));
    }
  }
  // The refused representations recorded nothing.
  expectArtifactError(
    () => service.exportFacts("owner-n", "can-app", "main", 0),
    "export-open",
    "export",
  );
  // Every declared representation exports once ready.
  for (const representation of REPRESENTATIONS) {
    const exported = service.exportData(
      "owner-n",
      "can-app",
      "main",
      token,
      exportDescriptor(representation),
    );
    assert.equal(exported.representation, representation);
  }
  assert.equal(service.exportFacts("owner-n", "can-app", "main", 3).representation, "bytes-digest");
});

// --- Bounded typed data export ---------------------------------------------------

check("typed-cells-validate-per-representation", () => {
  const { service, token } = openReady();
  // Each representation accepts its own exact shape.
  const shapes: [string, string][] = [
    ["integer-lexeme", "-17"],
    ["ieee-bits", "0xBFF0000000000000"],
    ["utf8-text", "can-do"],
    ["bytes-digest", BYTES_DIGEST],
  ];
  for (const [representation, value] of shapes) {
    const exported = service.exportData("owner-n", "can-app", "main", token, {
      representation,
      rows: [{ cells: [{ column: "v", value }] }],
    });
    assert.equal(exported.representation, representation);
    assert.ok(exported.digest.startsWith("sha256:"));
  }
  // Each representation rejects the others' shapes without effect.
  const rejects: [string, string][] = [
    ["integer-lexeme", "0x3ff8000000000000"],
    ["integer-lexeme", "4.5"],
    ["integer-lexeme", "007"],
    ["ieee-bits", "42"],
    ["ieee-bits", "0x123"],
    ["utf8-text", "bad\0byte"],
    ["bytes-digest", "not-a-digest"],
    ["bytes-digest", "42"],
  ];
  for (const [representation, value] of rejects) {
    expectArtifactError(
      () =>
        service.exportData("owner-n", "can-app", "main", token, {
          representation,
          rows: [{ cells: [{ column: "v", value }] }],
        }),
      "unknown-cell",
      "export",
    );
  }
  // Repeated columns, unknown fields, and malformed rows refuse too.
  expectArtifactError(
    () =>
      service.exportData("owner-n", "can-app", "main", token, {
        representation: "integer-lexeme",
        rows: [
          {
            cells: [
              { column: "v", value: "1" },
              { column: "v", value: "2" },
            ],
          },
        ],
      }),
    "unknown-cell",
    "export",
  );
  expectArtifactError(
    () =>
      service.exportData("owner-n", "can-app", "main", token, {
        representation: "integer-lexeme",
        rows: [{ cells: [{ column: "v", value: "1", unit: "px" }] }],
      }),
    "unknown-cell",
    "export",
  );
  // Facts carry digests only: raw tokens and raw values never cross.
  const facts = service.exportFacts("owner-n", "can-app", "main", 0);
  const rendered = JSON.stringify(facts);
  assert.ok(!rendered.includes(token));
  assert.ok(!rendered.includes("-17"));
  assert.ok(facts.rows[0]?.cells[0]?.valueDigest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(facts).sort(), [
    "artifact",
    "digest",
    "domState",
    "module",
    "owner",
    "ready",
    "representation",
    "rows",
  ]);
});

// --- Export verdict ---------------------------------------------------------------

check("ready-export-verdict-judges-facts", () => {
  const { service, token } = openReady();
  const facts = service.exportData(
    "owner-n",
    "can-app",
    "main",
    token,
    exportDescriptor("utf8-text"),
  );
  const verdict = assertExportProved({ kind: "ready-export", facts });
  assert.equal(verdict.artifact, "can-app");
  assert.equal(verdict.module, "main");
  assert.equal(verdict.representation, "utf8-text");
  assert.equal(verdict.rows, 1);
  assert.equal(verdict.cells, 1);
  assert.ok(verdict.digest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(verdict).sort(), [
    "artifact",
    "cells",
    "digest",
    "module",
    "representation",
    "rows",
  ]);
  // A forged digest never verifies.
  expectArtifactError(
    () => assertExportProved({ kind: "ready-export", facts: { ...facts, digest: "sha256:dead" } }),
    "forbidden-proof",
    "export",
  );
  // Facts without readiness refuse: unready facts prove nothing.
  expectArtifactError(
    () => assertExportProved({ kind: "ready-export", facts: { ...facts, ready: false } }),
    "ready-open",
    "load",
  );
  // Facts in an unsupported representation refuse with a named reason.
  try {
    assertExportProved({
      kind: "ready-export",
      facts: { ...facts, representation: "float-text" },
    });
    assert.fail("expected failure export:unknown-representation");
  } catch (error) {
    assert.ok(error instanceof BrowserArtifactError);
    assert.equal(error.code, "unknown-representation");
    assert.ok(error.message.includes("float-text"));
    assert.ok(error.message.includes("supported:"));
  }
  // Presented empty rows never verify, even with a recomputed digest: an
  // empty pass is inadmissible, not merely undelivered.
  const emptyRows = {
    ...facts,
    rows: [],
    digest: digestExport("can-app", "main", "utf8-text", "complete", []),
  };
  expectArtifactError(
    () => assertExportProved({ kind: "ready-export", facts: emptyRows }),
    "forbidden-proof",
    "export",
  );
  // Presented rows with no cells refuse the same way.
  const emptyCells = {
    ...facts,
    rows: [{ cells: [] }],
    digest: digestExport("can-app", "main", "utf8-text", "complete", [{ cells: [] }]),
  };
  expectArtifactError(
    () => assertExportProved({ kind: "ready-export", facts: emptyCells }),
    "forbidden-proof",
    "export",
  );
  // A readiness state outside the declared vocabulary refuses by name.
  expectArtifactError(
    () => assertExportProved({ kind: "ready-export", facts: { ...facts, domState: "loading" } }),
    "unknown-readiness",
    "load",
  );
});

check("forbidden-export-kinds-rejected", () => {
  const { service, token } = openReady();
  const facts = service.exportData("owner-n", "can-app", "main", token, exportDescriptor());
  assert.deepEqual([...FORBIDDEN_EXPORT_KINDS], ["unready-export", "empty-facts", "raw-bytes"]);
  // Every forbidden kind rejects before any verdict is read, even when
  // it smuggles genuine facts or an empty row set.
  for (const kind of [...FORBIDDEN_EXPORT_KINDS, "sealed-interval", ""]) {
    expectArtifactError(
      () => assertExportProved({ kind, detail: "driver claims export", facts }),
      "forbidden-proof",
      "export",
    );
  }
  expectArtifactError(() => assertExportProved(null), "forbidden-proof", "export");
  expectArtifactError(
    () => assertExportProved({ kind: "ready-export" }),
    "forbidden-proof",
    "export",
  );
});

// --- Owners, lifecycle, bounds, and provenance ---------------------------------------

check("foreign-owners-fail-every-call", () => {
  const { service, lease, token } = openModule();
  expectArtifactError(
    () => service.loadModule("owner-x", "can-app", lease, loadDescriptor("worker")),
    "wrong-owner",
    "artifact",
  );
  expectArtifactError(
    () => service.markReady("owner-x", "can-app", "main", token, "complete"),
    "wrong-owner",
    "artifact",
  );
  expectArtifactError(
    () => service.exportData("owner-x", "can-app", "main", token, exportDescriptor()),
    "wrong-owner",
    "artifact",
  );
  expectArtifactError(
    () => service.exportFacts("owner-x", "can-app", "main", 0),
    "wrong-owner",
    "artifact",
  );
  expectArtifactError(() => service.artifactFacts("owner-x", "can-app"), "wrong-owner", "artifact");
  expectArtifactError(
    () => service.releaseArtifact("owner-x", "can-app", lease),
    "wrong-owner",
    "artifact",
  );
});

check("release-closes-lease", () => {
  const { service, lease, token } = openReady();
  service.exportData("owner-n", "can-app", "main", token, exportDescriptor());
  const receipt = service.releaseArtifact("owner-n", "can-app", lease);
  assert.equal(receipt.artifact, "can-app");
  assert.equal(receipt.modules, 1);
  assert.equal(receipt.exports, 1);
  assert.ok(receipt.digest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(receipt).sort(), [
    "artifact",
    "digest",
    "exports",
    "modules",
    "owner",
  ]);
  // Released leases report closed on token-bearing calls and unknown on
  // lookups: the lease id is single-use.
  expectArtifactError(
    () => service.loadModule("owner-n", "can-app", lease, loadDescriptor("worker")),
    "artifact-closed",
    "artifact",
  );
  expectArtifactError(
    () => service.markReady("owner-n", "can-app", "main", token, "complete"),
    "unknown-artifact",
    "artifact",
  );
  expectArtifactError(
    () => service.artifactFacts("owner-n", "can-app"),
    "unknown-artifact",
    "artifact",
  );
});

check("capacity-bounds-refuse", () => {
  const tiny = new BrowserArtifactService(
    [{ owner: "o", artifact: "a", modules: ["m"] }],
    checkArtifactLimits({
      maxArtifacts: 1,
      maxModulesPerArtifact: 1,
      maxRowsPerExport: 1,
      maxCellsPerRow: 1,
      maxExportsPerModule: 1,
    }),
  );
  tiny.leaseArtifact("o", "a");
  const lease = tiny.leaseTokenForTest("o", "a");
  tiny.loadModule("o", "a", lease, { module: "m", moduleDigest: MODULE_DIGEST, byteLength: 8 });
  const token = tiny.moduleTokenForTest("o", "a", "m");
  tiny.markReady("o", "a", "m", token, "complete");
  expectArtifactError(
    () =>
      tiny.exportData("o", "a", "m", token, {
        representation: "integer-lexeme",
        rows: [{ cells: [{ column: "a", value: "1" }] }, { cells: [{ column: "a", value: "2" }] }],
      }),
    "capacity-exhausted",
    "artifact",
  );
  expectArtifactError(
    () =>
      tiny.exportData("o", "a", "m", token, {
        representation: "integer-lexeme",
        rows: [
          {
            cells: [
              { column: "a", value: "1" },
              { column: "b", value: "2" },
            ],
          },
        ],
      }),
    "capacity-exhausted",
    "artifact",
  );
  tiny.exportData("o", "a", "m", token, exportDescriptor());
  expectArtifactError(
    () => tiny.exportData("o", "a", "m", token, exportDescriptor()),
    "capacity-exhausted",
    "artifact",
  );
  // Declared grants past the caps never construct.
  expectArtifactError(
    () =>
      new BrowserArtifactService(
        [
          { owner: "o", artifact: "a", modules: ["m"] },
          { owner: "o", artifact: "b", modules: ["m"] },
        ],
        checkArtifactLimits({
          maxArtifacts: 1,
          maxModulesPerArtifact: 1,
          maxRowsPerExport: 1,
          maxCellsPerRow: 1,
          maxExportsPerModule: 1,
        }),
      ),
    "capacity-exhausted",
    "artifact",
  );
});

check("every-failure-names-its-layer", () => {
  assert.equal(BROWSER_ARTIFACT_CODES.length, 13);
  const seen = new Map<string, string>();
  for (const code of BROWSER_ARTIFACT_CODES) {
    const layer = layerOfArtifactCode(code);
    assert.ok(["artifact", "load", "export"].includes(layer));
    const error = new BrowserArtifactError(code, "probe");
    assert.equal(error.layer, layer);
    assert.equal(error.code, code);
    assert.ok(error.message.startsWith(`[${layer}:${code}]`));
    seen.set(code, layer);
  }
  // Spot-check the layer map: admission and bounds are artifact faults,
  // tokens and readiness are load faults, representations and proofs are
  // export faults.
  assert.equal(seen.get("unknown-artifact"), "artifact");
  assert.equal(seen.get("capacity-exhausted"), "artifact");
  assert.equal(seen.get("forged-token"), "load");
  assert.equal(seen.get("ready-open"), "load");
  assert.equal(seen.get("unknown-representation"), "export");
  assert.equal(seen.get("export-open"), "export");
  assert.equal(seen.get("forbidden-proof"), "export");
});

check("independence-scope-stated", () => {
  // The scope statement is a fixed export: the service shares no driver
  // code, decoded values, receipts, flags, environment, or process state,
  // and proves its facts from its own seeded doubles alone.
  assert.ok(BROWSER_ARTIFACT_INDEPENDENCE_SCOPE.startsWith("independent:"));
  assert.ok(BROWSER_ARTIFACT_INDEPENDENCE_SCOPE.includes("no driver code"));
  assert.ok(BROWSER_ARTIFACT_INDEPENDENCE_SCOPE.includes("seeded doubles"));
  assert.ok(BROWSER_ARTIFACT_INDEPENDENCE_SCOPE.includes("representation"));
});

console.log(
  JSON.stringify({
    kind: "can.browser-driver-artifact-check",
    schema_version: BROWSER_ARTIFACT_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
