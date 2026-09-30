// K15: load Can browser artifacts and export facts.
//
// A lease service over in-memory doubles only: no browser, no process,
// no flags, no environment, no sampling, no I/O. The service leases
// declared C artifacts to their owners with opaque minted tokens, loads
// declared modules with digest-pinned descriptors, marks DOM-ready
// through a finite readiness vocabulary, and exports bounded typed data
// in an exact declared representation. Missing readiness and missing
// exports refuse: an export without ready, a facts read without an
// export, and an unsupported exact representation never yield empty
// facts that look like success.
//
// Independence scope from the launch/driver service under test (normative):
//   SHARES: nothing executable. The service never imports, calls, or reads
//   driver code, driver-decoded values, receipts, flags, environment
//   variables, or process state. The only strings that may coincide are
//   the artifact/module names under test (supplied by the test, not by
//   the driver) and the vocabulary this file uses for facts. Facts carry
//   digests only: raw exported values never cross.
//   PROVES: artifact-lease admission, digest-pinned module load, DOM-ready
//   marking, bounded typed export with explicit exact representation,
//   missing ready/export refusal, unsupported-representation refusal with
//   a named reason, and error provenance (every failure names its layer).
//   Driver claims are compared AGAINST these facts; the service never
//   derives a fact FROM a driver claim.
//   Consequently a missing readiness, a missing export, or an unsupported
//   exact representation can never become an empty pass: only a ready
//   export in a declared representation proves export facts.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, browsers, processes, or live runtimes. Local
// controls only. Live host-dependent controls wait for the corresponding
// qualified profile and Q task.

import { createHash, randomBytes } from "node:crypto";

export const BROWSER_ARTIFACT_SCHEMA_VERSION = "1" as const;

// ---------------------------------------------------------------------------
// Independence scope (normative)
// ---------------------------------------------------------------------------

export const BROWSER_ARTIFACT_INDEPENDENCE_SCOPE: string =
  "independent: shares no driver code, decoded values, receipts, flags, " +
  "environment, or process state; proves artifact-lease admission, " +
  "digest-pinned module load, DOM-ready marking, bounded typed export with " +
  "explicit exact representation, missing ready/export refusal, " +
  "unsupported-representation refusal, and layered error provenance from " +
  "its own seeded doubles alone";

// ---------------------------------------------------------------------------
// Layers, codes, and the layered error
// ---------------------------------------------------------------------------

export const BROWSER_ARTIFACT_LAYERS = ["artifact", "load", "export"] as const;
export type BrowserArtifactLayer = (typeof BROWSER_ARTIFACT_LAYERS)[number];

// Closed failure-code vocabulary. Every code maps to exactly one layer:
//   artifact: lease admission, ownership, capacity, lifecycle.
//   load: module binding, lease tokens, digests, DOM-ready marking.
//   export: export tokens, exact representations, typed cells, missing
//     exports, and export-proof admission.
export const BROWSER_ARTIFACT_CODES = [
  "unknown-artifact",
  "wrong-owner",
  "capacity-exhausted",
  "artifact-closed",
  "forged-token",
  "unknown-module",
  "digest-unknown",
  "ready-open",
  "unknown-readiness",
  "unknown-representation",
  "unknown-cell",
  "export-open",
  "forbidden-proof",
] as const;
export type BrowserArtifactCode = (typeof BROWSER_ARTIFACT_CODES)[number];

const CODE_LAYER: Readonly<Record<BrowserArtifactCode, BrowserArtifactLayer>> = {
  "unknown-artifact": "artifact",
  "wrong-owner": "artifact",
  "capacity-exhausted": "artifact",
  "artifact-closed": "artifact",
  "forged-token": "load",
  "unknown-module": "load",
  "digest-unknown": "load",
  "ready-open": "load",
  "unknown-readiness": "load",
  "unknown-representation": "export",
  "unknown-cell": "export",
  "export-open": "export",
  "forbidden-proof": "export",
};

export function layerOfArtifactCode(code: BrowserArtifactCode): BrowserArtifactLayer {
  return CODE_LAYER[code];
}

export class BrowserArtifactError extends Error {
  readonly layer: BrowserArtifactLayer;
  readonly code: BrowserArtifactCode;
  constructor(code: BrowserArtifactCode, message: string) {
    super(`[${CODE_LAYER[code]}:${code}] ${message}`);
    this.name = "BrowserArtifactError";
    this.layer = CODE_LAYER[code];
    this.code = code;
  }
}

function fail(code: BrowserArtifactCode, message: string): never {
  throw new BrowserArtifactError(code, message);
}

// ---------------------------------------------------------------------------
// Bounds and names
// ---------------------------------------------------------------------------

export type BrowserArtifactLimits = Readonly<{
  maxArtifacts: number;
  maxModulesPerArtifact: number;
  maxRowsPerExport: number;
  maxCellsPerRow: number;
  maxExportsPerModule: number;
}>;

const LIMIT_KEYS = [
  "maxArtifacts",
  "maxModulesPerArtifact",
  "maxRowsPerExport",
  "maxCellsPerRow",
  "maxExportsPerModule",
] as const;

export function checkArtifactLimits(value: unknown): BrowserArtifactLimits {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new BrowserArtifactError("unknown-artifact", "limits must be an object");
  }
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    if (!(LIMIT_KEYS as readonly string[]).includes(key)) {
      throw new BrowserArtifactError("unknown-artifact", `unknown limit field: ${key}`);
    }
  }
  const out: Record<string, number> = {};
  for (const key of LIMIT_KEYS) {
    const entry = record[key];
    if (!Number.isSafeInteger(entry) || (entry as number) <= 0) {
      throw new BrowserArtifactError("unknown-artifact", `${key} must be a positive integer`);
    }
    out[key] = entry as number;
  }
  return Object.freeze(out) as BrowserArtifactLimits;
}

// Finite exact-representation vocabulary. Every export states one; there
// is no default, and anything outside this vocabulary is not an exact
// representation at all.
export const REPRESENTATIONS = [
  "integer-lexeme",
  "ieee-bits",
  "utf8-text",
  "bytes-digest",
] as const;
export type Representation = (typeof REPRESENTATIONS)[number];

// Finite DOM-readiness vocabulary. Readiness arises only from markReady,
// never from load alone.
export const READY_STATES = ["interactive", "complete"] as const;
export type ReadyState = (typeof READY_STATES)[number];

const MAX_NAME_LEN = 128;
const MAX_VALUE_LEN = 512;
const MODULE_DIGEST_RE = /^sha256:[0-9a-f]{64}$/;
const INTEGER_LEXEME_RE = /^-?(0|[1-9][0-9]*)$/;
const IEEE_BITS_RE = /^0x[0-9a-fA-F]{16}$/;

function checkName(value: unknown, what: string, code: BrowserArtifactCode): string {
  if (typeof value !== "string" || value === "" || value.length > MAX_NAME_LEN) {
    fail(code, `${what} must be a non-empty name of at most ${MAX_NAME_LEN} chars`);
  }
  if (!/^[A-Za-z0-9][A-Za-z0-9_.:-]*$/.test(value)) {
    fail(code, `${what} carries illegal characters: ${value}`);
  }
  return value;
}

function checkOwner(value: unknown): string {
  return checkName(value, "owner", "wrong-owner");
}

function checkArtifactName(value: unknown): string {
  return checkName(value, "artifact", "unknown-artifact");
}

function checkModuleName(value: unknown): string {
  return checkName(value, "module", "unknown-module");
}

function checkColumnName(value: unknown): string {
  return checkName(value, "column", "unknown-cell");
}

function checkRepresentation(value: unknown): Representation {
  if (typeof value !== "string" || !(REPRESENTATIONS as readonly string[]).includes(value)) {
    fail(
      "unknown-representation",
      `unsupported exact representation: ${String(value)} (supported: ${REPRESENTATIONS.join(", ")})`,
    );
  }
  return value as Representation;
}

function checkReadyState(value: unknown): ReadyState {
  if (typeof value !== "string" || !(READY_STATES as readonly string[]).includes(value)) {
    fail("unknown-readiness", `not a declared DOM-ready state: ${String(value)}`);
  }
  return value as ReadyState;
}

function checkModuleDigest(value: unknown): string {
  if (typeof value !== "string" || !MODULE_DIGEST_RE.test(value)) {
    fail("digest-unknown", "module digest must be a sha256 digest of pinned bytes");
  }
  return value;
}

function checkDescriptor(value: unknown, what: string): Record<string, unknown> {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail("unknown-cell", `${what} must be an object`);
  }
  return value as Record<string, unknown>;
}

function checkNoExtraKeys(
  record: Record<string, unknown>,
  keys: readonly string[],
  what: string,
): void {
  for (const key of Object.keys(record)) {
    if (!keys.includes(key)) {
      fail("unknown-cell", `${what} carries an unknown field: ${key}`);
    }
  }
}

// Validate one exported cell value against its exact representation. The
// representation decides the shape; unshaped values refuse without effect.
function checkCellValue(representation: Representation, value: unknown): string {
  if (typeof value !== "string" || value === "" || value.length > MAX_VALUE_LEN) {
    fail(
      "unknown-cell",
      `exported value must be a non-empty string of at most ${MAX_VALUE_LEN} chars`,
    );
  }
  switch (representation) {
    case "integer-lexeme":
      if (!INTEGER_LEXEME_RE.test(value)) {
        fail("unknown-cell", `not an exact integer lexeme: ${value}`);
      }
      return value;
    case "ieee-bits":
      if (!IEEE_BITS_RE.test(value)) {
        fail("unknown-cell", `not exact f64 bits: ${value}`);
      }
      return value;
    case "utf8-text":
      if (value.includes("\0")) {
        fail("unknown-cell", "exported text carries a NUL byte");
      }
      return value;
    case "bytes-digest":
      if (!MODULE_DIGEST_RE.test(value)) {
        fail("unknown-cell", "exported bytes digest is not a sha256 digest");
      }
      return value;
  }
}

// ---------------------------------------------------------------------------
// Facts (frozen, digest-only)
// ---------------------------------------------------------------------------

export type ArtifactLeaseFacts = Readonly<{
  owner: string;
  artifact: string;
  modules: readonly string[];
  handleDigest: string;
}>;

export type ModuleLoadFacts = Readonly<{
  artifact: string;
  module: string;
  owner: string;
  byteLength: number;
  moduleDigest: string;
  leaseDigest: string;
  handleDigest: string;
}>;

export type ReadyFacts = Readonly<{
  artifact: string;
  module: string;
  owner: string;
  domState: ReadyState;
  ready: boolean;
  digest: string;
}>;

export type ExportCellFacts = Readonly<{
  column: string;
  valueDigest: string;
}>;

export type ExportRowFacts = Readonly<{
  cells: readonly ExportCellFacts[];
}>;

export type ExportFacts = Readonly<{
  artifact: string;
  module: string;
  owner: string;
  representation: Representation;
  domState: ReadyState;
  ready: boolean;
  rows: readonly ExportRowFacts[];
  digest: string;
}>;

export type ModuleFacts = Readonly<{
  module: string;
  loaded: boolean;
  ready: boolean;
  domState: ReadyState | null;
  exports: number;
  digest: string;
}>;

export type ArtifactFacts = Readonly<{
  artifact: string;
  owner: string;
  modules: readonly ModuleFacts[];
  released: boolean;
  digest: string;
}>;

export type ReleaseReceipt = Readonly<{
  artifact: string;
  owner: string;
  modules: number;
  exports: number;
  digest: string;
}>;

// An export proof claim. Only ready-export is admissible; every other
// kind rejects with forbidden-proof before any verdict is read, so a
// missing readiness, a missing export, or raw bytes can never pass.
export const FORBIDDEN_EXPORT_KINDS = ["unready-export", "empty-facts", "raw-bytes"] as const;
export type ForbiddenExportKind = (typeof FORBIDDEN_EXPORT_KINDS)[number];

export type ExportProofClaim =
  | Readonly<{ kind: "ready-export"; facts: ExportFacts }>
  | Readonly<{ kind: ForbiddenExportKind; detail: string }>;

export type ExportVerdict = Readonly<{
  artifact: string;
  module: string;
  representation: Representation;
  rows: number;
  cells: number;
  digest: string;
}>;

export type CellDescriptor = Readonly<{
  column: string;
  value: string;
}>;

export type RowDescriptor = Readonly<{
  cells: readonly CellDescriptor[];
}>;

export type ExportDescriptor = Readonly<{
  representation: Representation;
  rows: readonly RowDescriptor[];
}>;

export type ModuleDescriptor = Readonly<{
  module: string;
  moduleDigest: string;
  byteLength: number;
}>;

type ModuleRecord = {
  module: string;
  token: string;
  byteLength: number;
  moduleDigest: string;
  ready: boolean;
  domState: ReadyState | null;
  exports: ExportFacts[];
};

type LeaseRecord = {
  artifact: string;
  owner: string;
  token: string;
  declaredModules: readonly string[];
  modules: Map<string, ModuleRecord>;
  released: boolean;
};

function mintToken(prefix: string): string {
  return `${prefix}-${randomBytes(16).toString("hex")}`;
}

function digestText(text: string): string {
  return `sha256:${createHash("sha256").update(text, "utf8").digest("hex")}`;
}

// The export digest is recomputed from carried fields, so
// assertExportProved re-verifies a presented attestation instead of
// trusting its digest string.
export function digestExport(
  artifact: string,
  module: string,
  representation: string,
  domState: string,
  rows: readonly ExportRowFacts[],
): string {
  const hash = createHash("sha256");
  hash.update(artifact, "utf8");
  hash.update("\0", "utf8");
  hash.update(module, "utf8");
  hash.update("\0", "utf8");
  hash.update(representation, "utf8");
  hash.update("\0", "utf8");
  hash.update(domState, "utf8");
  for (const row of rows) {
    hash.update("\n", "utf8");
    for (const cell of row.cells) {
      hash.update(`${cell.column}=${cell.valueDigest}`, "utf8");
      hash.update(";", "utf8");
    }
  }
  return `sha256:${hash.digest("hex")}`;
}

// ---------------------------------------------------------------------------
// Service: leases, module load, DOM-ready, bounded typed export
// ---------------------------------------------------------------------------

export type ArtifactGrant = Readonly<{
  owner: string;
  artifact: string;
  modules: readonly string[];
}>;

export class BrowserArtifactService {
  private readonly limits: BrowserArtifactLimits;
  private readonly grants = new Map<string, ArtifactGrant>();
  private readonly leases = new Map<string, LeaseRecord>();

  constructor(grants: readonly ArtifactGrant[], limits: BrowserArtifactLimits) {
    if (grants.length === 0) {
      throw new BrowserArtifactError("unknown-artifact", "declare at least one owned artifact");
    }
    if (grants.length > limits.maxArtifacts) {
      throw new BrowserArtifactError("capacity-exhausted", "declared artifacts exceed the cap");
    }
    for (const grant of grants) {
      checkOwner(grant.owner);
      checkArtifactName(grant.artifact);
      if (grant.modules.length === 0) {
        fail("unknown-artifact", `artifact ${grant.artifact} declares no modules`);
      }
      if (grant.modules.length > limits.maxModulesPerArtifact) {
        fail("capacity-exhausted", `artifact ${grant.artifact} declares too many modules`);
      }
      const seen = new Set<string>();
      for (const module of grant.modules) {
        checkModuleName(module);
        if (seen.has(module)) {
          fail("unknown-artifact", `artifact ${grant.artifact} repeats module: ${module}`);
        }
        seen.add(module);
      }
      const key = `${grant.owner}\0${grant.artifact}`;
      if (this.grants.has(key)) {
        fail("unknown-artifact", `duplicate owned artifact: ${grant.owner}/${grant.artifact}`);
      }
      this.grants.set(
        key,
        Object.freeze({
          owner: grant.owner,
          artifact: grant.artifact,
          modules: [...grant.modules],
        }),
      );
    }
    this.limits = limits;
  }

  get leaseCount(): number {
    return this.leases.size;
  }

  // Lease one owned artifact. Only an exact declared (owner, artifact)
  // grant admits; an artifact the test never granted rejects before any
  // effect. Leases bind to one owner: a second identity joining the same
  // artifact name fails the owner check.
  leaseArtifact(owner: string, artifact: string): ArtifactLeaseFacts {
    checkOwner(owner);
    checkArtifactName(artifact);
    const grant = this.grants.get(`${owner}\0${artifact}`);
    if (grant === undefined) {
      fail("unknown-artifact", `no owned grant for artifact: ${owner}/${artifact}`);
    }
    const prior = this.leases.get(artifact);
    if (prior !== undefined && !prior.released) {
      if (prior.owner !== owner) {
        fail("wrong-owner", `artifact is leased by another identity: ${artifact}`);
      }
      return this.leaseFactsOf(prior);
    }
    if (this.leases.size >= this.limits.maxArtifacts) {
      fail("capacity-exhausted", "artifact lease table full");
    }
    const record: LeaseRecord = {
      artifact,
      owner,
      token: mintToken("lease"),
      declaredModules: (grant as ArtifactGrant).modules,
      modules: new Map(),
      released: false,
    };
    this.leases.set(artifact, record);
    return this.leaseFactsOf(record);
  }

  private leaseFactsOf(record: LeaseRecord): ArtifactLeaseFacts {
    return Object.freeze({
      owner: record.owner,
      artifact: record.artifact,
      modules: Object.freeze([...record.declaredModules]),
      handleDigest: digestText(record.token),
    });
  }

  private requireLease(owner: string, artifact: string): LeaseRecord {
    checkOwner(owner);
    checkArtifactName(artifact);
    const lease = this.leases.get(artifact);
    if (lease === undefined || lease.released) {
      fail("unknown-artifact", `no live lease for artifact: ${owner}/${artifact}`);
    }
    const record = lease as LeaseRecord;
    if (record.owner !== owner) {
      fail("wrong-owner", "artifact is leased by another identity");
    }
    return record;
  }

  // Verify the lease token by table lookup. Unknown leases and invented
  // tokens are never authority; released leases report artifact-closed.
  private requireLiveLease(owner: string, artifact: string, token: string): LeaseRecord {
    const lease = this.leases.get(artifact);
    if (lease === undefined) {
      fail("unknown-artifact", `no live lease for artifact: ${owner}/${artifact}`);
    }
    const record = lease as LeaseRecord;
    if (record.owner !== owner) {
      fail("wrong-owner", "artifact is leased by another identity");
    }
    if (record.released) {
      fail("artifact-closed", `artifact lease is released: ${artifact}`);
    }
    if (token === "" || token !== record.token) {
      fail("forged-token", "lease token is not the attested token");
    }
    return record;
  }

  private requireModule(owner: string, artifact: string, module: string): ModuleRecord {
    const lease = this.requireLease(owner, artifact);
    checkModuleName(module);
    const record = lease.modules.get(module);
    if (record === undefined) {
      fail("unknown-module", `module is not loaded: ${artifact}/${module}`);
    }
    return record as ModuleRecord;
  }

  // Verify the module token by table lookup. Unknown modules and
  // invented tokens are never authority.
  private requireLiveModule(
    owner: string,
    artifact: string,
    module: string,
    token: string,
  ): { lease: LeaseRecord; module: ModuleRecord } {
    const lease = this.requireLease(owner, artifact);
    checkModuleName(module);
    const record = lease.modules.get(module);
    if (record === undefined) {
      fail("unknown-module", `module is not loaded: ${artifact}/${module}`);
    }
    const live = record as ModuleRecord;
    if (token === "" || token !== live.token) {
      fail("forged-token", "module token is not the attested token");
    }
    return { lease, module: live };
  }

  // Test-only accessors: raw tokens cross exactly here so bounded
  // controls can present them on later calls. Facts carry digests only.
  leaseTokenForTest(owner: string, artifact: string): string {
    return this.requireLease(owner, artifact).token;
  }

  moduleTokenForTest(owner: string, artifact: string, module: string): string {
    return this.requireModule(owner, artifact, module).token;
  }

  // Load one declared module through the live lease. The lease token is
  // verified by lookup before the descriptor is read; the module must be
  // declared in the grant, the digest must pin sha256 bytes, and the
  // byte length must be a positive bounded integer. Re-loading joins the
  // identical load only: a differing descriptor refuses rather than
  // swapping the pinned bytes.
  loadModule(owner: string, artifact: string, token: string, descriptor: unknown): ModuleLoadFacts {
    const lease = this.requireLiveLease(owner, artifact, token);
    const record = checkDescriptor(descriptor, "module load");
    checkNoExtraKeys(record, ["module", "moduleDigest", "byteLength"], "module load");
    const module = checkModuleName(record["module"]);
    const moduleDigest = checkModuleDigest(record["moduleDigest"]);
    const byteLengthRaw = record["byteLength"];
    if (!Number.isSafeInteger(byteLengthRaw) || (byteLengthRaw as number) <= 0) {
      fail("digest-unknown", "module byte length must be a positive integer");
    }
    const byteLength = byteLengthRaw as number;
    if (!lease.declaredModules.includes(module)) {
      fail("unknown-module", `module is not declared for artifact: ${artifact}/${module}`);
    }
    const prior = lease.modules.get(module);
    if (prior !== undefined) {
      if (prior.moduleDigest !== moduleDigest || prior.byteLength !== byteLength) {
        fail("digest-unknown", `module load pins different bytes: ${artifact}/${module}`);
      }
      return this.loadFactsOf(lease, prior);
    }
    if (lease.modules.size >= this.limits.maxModulesPerArtifact) {
      fail("capacity-exhausted", "module table full for artifact");
    }
    const loaded: ModuleRecord = {
      module,
      token: mintToken("mod"),
      byteLength,
      moduleDigest,
      ready: false,
      domState: null,
      exports: [],
    };
    lease.modules.set(module, loaded);
    return this.loadFactsOf(lease, loaded);
  }

  private loadFactsOf(lease: LeaseRecord, module: ModuleRecord): ModuleLoadFacts {
    return Object.freeze({
      artifact: lease.artifact,
      module: module.module,
      owner: lease.owner,
      byteLength: module.byteLength,
      moduleDigest: module.moduleDigest,
      leaseDigest: digestText(lease.token),
      handleDigest: digestText(module.token),
    });
  }

  // Mark one loaded module DOM-ready. The module token is verified by
  // lookup; the readiness state must be declared. Re-marking with the
  // same state joins; advancing from interactive to complete re-marks,
  // and regressing from complete refuses rather than unseeing readiness.
  markReady(
    owner: string,
    artifact: string,
    module: string,
    token: string,
    domState: unknown,
  ): ReadyFacts {
    const state = checkReadyState(domState);
    const { module: record } = this.requireLiveModule(owner, artifact, module, token);
    if (record.ready && record.domState === "complete" && state === "interactive") {
      fail("unknown-readiness", "readiness cannot regress from complete to interactive");
    }
    record.ready = true;
    record.domState = state;
    return Object.freeze({
      artifact,
      module: record.module,
      owner,
      domState: record.domState,
      ready: true,
      digest: digestText(`ready:${artifact}:${record.module}:${record.domState}`),
    });
  }

  private checkExportDescriptor(descriptor: unknown): {
    representation: Representation;
    rows: { column: string; value: string }[][];
  } {
    const record = checkDescriptor(descriptor, "export");
    checkNoExtraKeys(record, ["representation", "rows"], "export");
    const representation = checkRepresentation(record["representation"]);
    if (!Array.isArray(record["rows"])) {
      fail("unknown-cell", "export rows must be an array");
    }
    const rows = record["rows"] as unknown[];
    if (rows.length === 0) {
      fail("export-open", "export carries no rows: a missing export is not empty facts");
    }
    if (rows.length > this.limits.maxRowsPerExport) {
      fail("capacity-exhausted", "export carries too many rows");
    }
    const checked: { column: string; value: string }[][] = [];
    for (const row of rows) {
      const rowRecord = checkDescriptor(row, "export row");
      checkNoExtraKeys(rowRecord, ["cells"], "export row");
      if (!Array.isArray(rowRecord["cells"])) {
        fail("unknown-cell", "export row cells must be an array");
      }
      const cells = rowRecord["cells"] as unknown[];
      if (cells.length === 0) {
        fail("export-open", "export row carries no cells: a missing export is not empty facts");
      }
      if (cells.length > this.limits.maxCellsPerRow) {
        fail("capacity-exhausted", "export row carries too many cells");
      }
      const checkedCells: { column: string; value: string }[] = [];
      const seen = new Set<string>();
      for (const cell of cells) {
        const cellRecord = checkDescriptor(cell, "export cell");
        checkNoExtraKeys(cellRecord, ["column", "value"], "export cell");
        const column = checkColumnName(cellRecord["column"]);
        if (seen.has(column)) {
          fail("unknown-cell", `export row repeats column: ${column}`);
        }
        seen.add(column);
        checkedCells.push({ column, value: checkCellValue(representation, cellRecord["value"]) });
      }
      checked.push(checkedCells);
    }
    return { representation, rows: checked };
  }

  // Export bounded typed data in one exact representation. Readiness
  // first: an export without ready refuses with ready-open, and an
  // unsupported representation refuses with unknown-representation and a
  // named reason before any cell is read. Facts carry digests only: raw
  // exported values never cross.
  exportData(
    owner: string,
    artifact: string,
    module: string,
    token: string,
    descriptor: unknown,
  ): ExportFacts {
    const { module: record } = this.requireLiveModule(owner, artifact, module, token);
    if (!record.ready || record.domState === null) {
      fail("ready-open", `module is not DOM-ready: ${artifact}/${module}`);
    }
    const checked = this.checkExportDescriptor(descriptor);
    if (record.exports.length >= this.limits.maxExportsPerModule) {
      fail("capacity-exhausted", "export table full for module");
    }
    const rows: ExportRowFacts[] = checked.rows.map((row) =>
      Object.freeze({
        cells: Object.freeze(
          row.map((cell) =>
            Object.freeze({ column: cell.column, valueDigest: digestText(cell.value) }),
          ),
        ),
      }),
    );
    const facts: ExportFacts = Object.freeze({
      artifact,
      module: record.module,
      owner,
      representation: checked.representation,
      domState: record.domState,
      ready: true,
      rows: Object.freeze(rows),
      digest: digestExport(artifact, record.module, checked.representation, record.domState, rows),
    });
    record.exports.push(facts);
    return facts;
  }

  // Read one recorded export. A module with no export refuses with
  // export-open: a missing export is never empty facts.
  exportFacts(owner: string, artifact: string, module: string, index: number): ExportFacts {
    const record = this.requireModule(owner, artifact, module);
    if (!Number.isSafeInteger(index) || index < 0 || index >= record.exports.length) {
      fail("export-open", `no recorded export at index: ${artifact}/${module}`);
    }
    return record.exports[index] as ExportFacts;
  }

  // Artifact facts stay readable while leased: the lease is evidence.
  // Released leases report artifact-closed.
  artifactFacts(owner: string, artifact: string): ArtifactFacts {
    const lease = this.requireLease(owner, artifact);
    const modules: ModuleFacts[] = [...lease.modules.values()].map((record) =>
      Object.freeze({
        module: record.module,
        loaded: true,
        ready: record.ready,
        domState: record.domState,
        exports: record.exports.length,
        digest: digestText(
          `module:${record.module}:${record.ready ? record.domState : "unready"}:${record.exports.length}`,
        ),
      }),
    );
    return Object.freeze({
      artifact: lease.artifact,
      owner: lease.owner,
      modules: Object.freeze(modules),
      released: false,
      digest: digestText(
        `artifact:${lease.artifact}:${modules.map((entry) => `${entry.module}=${entry.exports}`).join(",")}`,
      ),
    });
  }

  // Release the lease. Loaded modules, readiness, and recorded exports
  // are tallied on the receipt; the lease id is single-use, so a later
  // lease of the same artifact name starts a fresh record.
  releaseArtifact(owner: string, artifact: string, token: string): ReleaseReceipt {
    const lease = this.requireLiveLease(owner, artifact, token);
    const exports = [...lease.modules.values()].reduce(
      (total, record) => total + record.exports.length,
      0,
    );
    const receipt: ReleaseReceipt = Object.freeze({
      artifact: lease.artifact,
      owner: lease.owner,
      modules: lease.modules.size,
      exports,
      digest: digestText(`release:${lease.artifact}:${lease.modules.size}:${exports}`),
    });
    lease.released = true;
    return receipt;
  }
}

// ---------------------------------------------------------------------------
// Export verdict: only a ready export in a declared representation judges
// ---------------------------------------------------------------------------

function checkExportProofShape(value: unknown): ExportProofClaim {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail("forbidden-proof", "export proof must be a ready-export object");
  }
  const record = value as Record<string, unknown>;
  if (record["kind"] === "ready-export") {
    const facts = record["facts"] as ExportFacts;
    if (facts === null || typeof facts !== "object" || Array.isArray(facts)) {
      fail("forbidden-proof", "ready export carries no export facts");
    }
    return { kind: "ready-export", facts };
  }
  if (typeof record["kind"] === "string") {
    fail("forbidden-proof", `export proof kind is inadmissible: ${String(record["kind"])}`);
  }
  fail("forbidden-proof", "export proof carries no kind");
}

// Judge one export proof claim. Only a ready export in a declared exact
// representation proves export facts: unready exports, empty facts, raw
// bytes, and any other non-ready claim reject with forbidden-proof before
// any verdict is read; facts without readiness refuse with ready-open;
// facts in an unsupported representation refuse with
// unknown-representation and a named reason; and a forged digest never
// verifies. None of these yield a verdict that looks like success.
export function assertExportProved(claim: unknown): ExportVerdict {
  const proof = checkExportProofShape(claim);
  if (proof.kind !== "ready-export") {
    fail("forbidden-proof", `export proof kind is inadmissible: ${proof.kind}`);
  }
  const facts = (proof as { kind: "ready-export"; facts: ExportFacts }).facts;
  if (
    typeof facts.artifact !== "string" ||
    typeof facts.module !== "string" ||
    typeof facts.owner !== "string" ||
    typeof facts.representation !== "string" ||
    typeof facts.domState !== "string" ||
    typeof facts.ready !== "boolean" ||
    !Array.isArray(facts.rows) ||
    typeof facts.digest !== "string"
  ) {
    fail("forbidden-proof", "ready export carries malformed facts");
  }
  if (!(REPRESENTATIONS as readonly string[]).includes(facts.representation)) {
    fail(
      "unknown-representation",
      `unsupported exact representation: ${String(facts.representation)} ` +
        `(supported: ${REPRESENTATIONS.join(", ")})`,
    );
  }
  if (!facts.ready) {
    fail("ready-open", "export facts without DOM-ready prove nothing");
  }
  if (!(READY_STATES as readonly string[]).includes(facts.domState)) {
    fail("unknown-readiness", `not a declared DOM-ready state: ${String(facts.domState)}`);
  }
  // The service never issues empty rows or cells; the judge re-checks so
  // presented empty facts can never verify as an empty pass.
  if (facts.rows.length === 0) {
    fail("forbidden-proof", "ready export carries no rows: a missing export is not empty facts");
  }
  let cells = 0;
  for (const row of facts.rows) {
    if (row === null || typeof row !== "object" || !Array.isArray(row.cells)) {
      fail("forbidden-proof", "ready export carries a malformed row");
    }
    if (row.cells.length === 0) {
      fail(
        "forbidden-proof",
        "ready export row carries no cells: a missing export is not empty facts",
      );
    }
    for (const cell of row.cells) {
      if (typeof cell.column !== "string" || typeof cell.valueDigest !== "string") {
        fail("forbidden-proof", "ready export carries a malformed cell");
      }
      cells += 1;
    }
  }
  const recomputed = digestExport(
    facts.artifact,
    facts.module,
    facts.representation,
    facts.domState,
    facts.rows,
  );
  if (recomputed !== facts.digest) {
    fail("forbidden-proof", "export facts digest does not verify");
  }
  return Object.freeze({
    artifact: facts.artifact,
    module: facts.module,
    representation: facts.representation as Representation,
    rows: facts.rows.length,
    cells,
    digest: digestText(`verdict:${facts.artifact}:${facts.module}:${facts.rows.length}:${cells}`),
  });
}
