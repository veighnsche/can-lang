// NT-P28 base test owner adapter. Admits N-issued grant shapes into
// worker-local opaque owner handles; issuance itself is verified by N at
// dispatch, never here. Handles are unforgeable branded objects bound to
// one owner table: cross-table handles fail closed as caller bugs, and
// released ids are never reused so stale stays stale. Release cascades:
// channels opened under a released owner go stale.
import { failure, success, type AssertionContext, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { createDomainRuntime } from "../domain.ts";

const ownerBrand = Symbol("can.test.owner");
export type OwnerHandle = Readonly<{ readonly [ownerBrand]: number }>;

// N mints ng1- + 32 lowercase hex (protocol/dispatch/grant.go); the adapter
// admits the shape only.
const GRANT_PREFIX = "ng1-";
const GRANT_HEX_LEN = 32;
const GRANT_LEN = GRANT_PREFIX.length + GRANT_HEX_LEN;
const HEX = /^[0-9a-f]+$/;
const MAX_OWNERS = 256;

const origin = Object.freeze({
  source: "can:test-owner",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export type TestOwnerErrors = Readonly<{
  invalidGrant: string;
}>;

type OwnerCell = { readonly grant: string; live: boolean };

function handleName(id: number): string {
  return `owner#${id}`;
}

export function isOwnerHandle(value: unknown): value is OwnerHandle {
  return (
    typeof value === "object" &&
    value !== null &&
    typeof (value as Record<symbol, unknown>)[ownerBrand] === "number"
  );
}

// Diagnostic tag for a presented owner handle, or null when the value is
// not an owner handle at all.
export function ownerTag(value: unknown): string | null {
  if (!isOwnerHandle(value)) return null;
  return handleName(value[ownerBrand]);
}

export function createTestOwner(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: TestOwnerErrors,
) {
  const owners = new Map<number, OwnerCell>();
  let nextId = 0;

  function invalidGrant(reason: string): Completion<never> {
    return failure(
      domain.create(errors.invalidGrant, record(errors.invalidGrant, [["reason", reason]]), origin),
    );
  }

  return Object.freeze({
    async admitGrant(
      grant: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<OwnerHandle>> {
      if (typeof grant !== "string" || grant.length === 0) return invalidGrant("empty");
      if (grant.length !== GRANT_LEN) return invalidGrant("bad-length");
      if (!grant.startsWith(GRANT_PREFIX)) return invalidGrant("bad-prefix");
      if (!HEX.test(grant.slice(GRANT_PREFIX.length))) return invalidGrant("bad-charset");
      if (owners.size >= MAX_OWNERS) throw new TypeError("test owner table cap reached");
      nextId += 1;
      owners.set(nextId, { grant, live: true });
      const handle: OwnerHandle = Object.freeze({ [ownerBrand]: nextId });
      return success(handle);
    },
    async releaseGrant(handle: unknown, _context?: AssertionContext): Promise<Completion<void>> {
      if (!isOwnerHandle(handle)) throw new TypeError("test::grant_release needs an owner handle");
      const id = handle[ownerBrand];
      const cell = owners.get(id);
      if (cell === undefined) throw new TypeError(`foreign owner handle ${handleName(id)}`);
      cell.live = false;
      return success(undefined);
    },
    // Transport cascade input: only live owners admit channel operations.
    // Unknown ids are dead, never an error here; the channel side names
    // the presented channel handle in stale_handle.
    ownerLive(handle: unknown): boolean {
      if (!isOwnerHandle(handle)) return false;
      return owners.get(handle[ownerBrand])?.live === true;
    },
    // Grant string for N-owner request envelopes (NT-I01): the live grant
    // bound to this handle, or null for foreign/dead handles. The caller
    // enforces admission-first; this accessor only reads.
    grantOf(handle: unknown): string | null {
      if (!isOwnerHandle(handle)) return null;
      const cell = owners.get(handle[ownerBrand]);
      if (cell === undefined || !cell.live) return null;
      return cell.grant;
    },
  });
}

export type TestOwner = ReturnType<typeof createTestOwner>;
