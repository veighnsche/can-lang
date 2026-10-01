// NT-P28 S1c test evidence adapter. Sealed evidence bundles: open mints
// an opaque bundle handle; append records named byte blobs; seal
// freezes the bundle and returns a digest plus a closed-vocabulary
// receipt kind (qualified-report, n-receipt, cleanup-receipt). The seal
// digest is sha256 over length-framed name/bytes pairs in insertion
// order, so the attested bytes are exactly the appended bytes. Names
// are 1..128 chars without NUL or separators and unique per bundle;
// the seal consumes the bundle (append/seal after seal is closed).
import { createHash } from "node:crypto";
import { failure, success, type AssertionContext, type Completion } from "../completion.ts";
import { copyBytes, isBytes, type Bytes } from "../bytes.ts";
import { record } from "../data.ts";
import { createDomainRuntime } from "../domain.ts";
import type { FailureOrigin } from "../failure.ts";
import { ownerTag, type TestOwner } from "./owner.ts";

const evidenceBrand = Symbol("can.test.evidence");
export type EvidenceHandle = Readonly<{ readonly [evidenceBrand]: number }>;

export const RECEIPT_KINDS = ["qualified-report", "n-receipt", "cleanup-receipt"] as const;
export type ReceiptKind = (typeof RECEIPT_KINDS)[number];

const MAX_BUNDLES = 256;
const MAX_NAME_CHARS = 128;
const MAX_ENTRIES = 1024;

const origin: FailureOrigin = Object.freeze({
  source: "can:test-evidence",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export type TestEvidenceErrors = Readonly<{
  staleHandle: string;
  closedHandle: string;
  invalidName: string;
  invalidKind: string;
  receipt: string;
}>;

type EvidenceEntry = Readonly<{ readonly name: string; readonly data: Bytes }>;

type EvidenceCell = {
  readonly owner: unknown;
  readonly entries: EvidenceEntry[];
  sealed: boolean;
};

function isEvidenceHandle(value: unknown): value is EvidenceHandle {
  return (
    typeof value === "object" &&
    value !== null &&
    typeof (value as Record<symbol, unknown>)[evidenceBrand] === "number"
  );
}

function evidenceName(id: number): string {
  return `evidence#${id}`;
}

export function createTestEvidence(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: TestEvidenceErrors,
  owner: TestOwner,
) {
  const bundles = new Map<number, EvidenceCell>();
  let nextId = 0;

  function fail(
    identity: string,
    fields: readonly (readonly [string, unknown])[],
  ): Completion<never> {
    return failure(domain.create(identity, record(identity, fields), origin));
  }

  function use(id: number): EvidenceCell | Completion<never> {
    const cell = bundles.get(id);
    if (cell === undefined || !owner.ownerLive(cell.owner))
      return fail(errors.staleHandle, [["handle", evidenceName(id)]]);
    return cell;
  }

  return Object.freeze({
    async openEvidence(
      ownerHandle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<EvidenceHandle>> {
      const tag = ownerTag(ownerHandle);
      if (tag === null) throw new TypeError("test::evidence_open needs an owner handle");
      if (!owner.ownerLive(ownerHandle)) return fail(errors.staleHandle, [["handle", tag]]);
      if (bundles.size >= MAX_BUNDLES) throw new TypeError("test evidence table cap reached");
      nextId += 1;
      bundles.set(nextId, { owner: ownerHandle, entries: [], sealed: false });
      const handle: EvidenceHandle = Object.freeze({ [evidenceBrand]: nextId });
      return success(handle);
    },

    async appendEvidence(
      handle: unknown,
      name: unknown,
      data: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<undefined>> {
      if (!isEvidenceHandle(handle))
        throw new TypeError("test::evidence_append needs an evidence handle");
      const id = handle[evidenceBrand];
      const cell = use(id);
      if (!("entries" in cell)) return cell;
      if (cell.sealed) return fail(errors.closedHandle, [["handle", evidenceName(id)]]);
      if (typeof name !== "string" || name.length === 0)
        return fail(errors.invalidName, [
          ["name", typeof name === "string" ? name : "?"],
          ["reason", "empty"],
        ]);
      if (name.includes("\0"))
        return fail(errors.invalidName, [
          ["name", name],
          ["reason", "nul_byte"],
        ]);
      if (name.includes("/"))
        return fail(errors.invalidName, [
          ["name", name],
          ["reason", "separator"],
        ]);
      if (name.length > MAX_NAME_CHARS)
        return fail(errors.invalidName, [
          ["name", name],
          ["reason", "too_long"],
        ]);
      if (cell.entries.some((entry) => entry.name === name))
        return fail(errors.invalidName, [
          ["name", name],
          ["reason", "duplicate"],
        ]);
      if (cell.entries.length >= MAX_ENTRIES)
        return fail(errors.invalidName, [
          ["name", name],
          ["reason", "bundle_full"],
        ]);
      if (!isBytes(data)) throw new TypeError("test::evidence_append needs bytes data");
      cell.entries.push({ name, data });
      return success(undefined);
    },

    async sealEvidence(
      handle: unknown,
      kind: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      if (!isEvidenceHandle(handle))
        throw new TypeError("test::evidence_seal needs an evidence handle");
      const id = handle[evidenceBrand];
      const cell = use(id);
      if (!("entries" in cell)) return cell;
      if (cell.sealed) return fail(errors.closedHandle, [["handle", evidenceName(id)]]);
      if (typeof kind !== "string" || !(RECEIPT_KINDS as readonly string[]).includes(kind))
        return fail(errors.invalidKind, [["kind", typeof kind === "string" ? kind : "?"]]);
      cell.sealed = true;
      const hash = createHash("sha256");
      let bytes = 0n;
      for (const entry of cell.entries) {
        const nameBytes = new TextEncoder().encode(entry.name);
        const data = copyBytes(entry.data, origin);
        const frame = new Uint8Array(8 + nameBytes.byteLength + 8 + data.byteLength);
        const view = new DataView(frame.buffer);
        view.setBigUint64(0, BigInt(nameBytes.byteLength));
        frame.set(nameBytes, 8);
        view.setBigUint64(8 + nameBytes.byteLength, BigInt(data.byteLength));
        frame.set(data, 8 + nameBytes.byteLength + 8);
        hash.update(frame);
        bytes += BigInt(data.byteLength);
      }
      return success(
        record(errors.receipt, [
          ["digest", `sha256:${hash.digest("hex")}`],
          ["kind", kind],
          ["bytes", bytes],
          ["entries", BigInt(cell.entries.length)],
        ]),
      );
    },
  });
}

export type TestEvidence = ReturnType<typeof createTestEvidence>;
