# a84 — Asset provenance and acceptance (S4 paper design)

Status: design (paper first, per the slices plan). The in-language half
(`schema__asset__recheck` + binding rows) ships in S4; the acceptance
workflow and any compiler diagnostics for it are future work, explicitly
not claimed here. Until they land, builders stay test-gated (see
`std/schema/README.md`).

## Problem

S1–S3 authorize construction from explicit values: snapshot, policy
handle, site, head sequence, time. Every one of those is caller-
supplied data. The language cannot tell a production snapshot from a
fixture snapshot, the governing policy from a convenient one, or now
from then. That distinction is provenance, and it lives outside the
language — this paper puts the boundary in one place so no slice
has to re-argue it.

Rulings consumed (all settled in `docs/archive/a/a83-astra-schema.md`): external
approver with pinned key and no self-approval (§2); one policy per
program deployment (§3); full binding incl. snapshot, site, and
validity interval (§4); construction-time authorization only (§5);
finite expiry, monotonic sequence, retained revocations, rotation by
repinning (§7); separate fixture trust root (§12). Deferred items
below are quoted from the a83 deferred table.

## What stays outside the language

- Approver enrollment, key pinning and rotation, and the signing/
  acceptance configuration: owned by deployment operation.
- The trusted head sequence and the trusted clock: supplied by the
  acceptance workflow at construction/acceptance time, never read
  from program state, never defaulted.
- Secure document delivery, redirect/MIME enforcement, withdrawal of
  deployed artifacts: retrieval and operations layers, not Schema.
- The fixture trust root and fixture registry: owned by the test
  harness, never the production keys (S5 separates them).

## Acceptance record (per program deployment)

One immutable record per deployment, written by the acceptance
workflow before construction is admitted:

- program id, policy id (the single governing policy, §3),
- registry snapshot id + sequence (the accepted head, §7),
- trust-root pin (approver key id + rotation counter, §2),
- proposer/approver identities (separation record, §2),
- freshness timestamp (rechecked before deployment acceptance, §7).

Construction is admitted only against the accepted record; the
record is immutable during construction. Policy or snapshot changes
mid-construction invalidate the session (no hot swap, §3).

## In-language half (shipped in S4)

- `schema__asset__recheck` re-runs authorization over supplied
  values and requires witness equality with a freshly minted
  witness. It binds request, site, snapshot content, policy handle
  agreement, head currency, and validity interval — everything the
  witness can carry plus everything the caller supplies.
- What recheck does NOT do (structural, not a gap): the witness
  carries no snapshot sequence (integers have no string form), so
  currency is enforced on the supplied snapshot, not historically
  on the witness. A replay under identical bindings and a current
  snapshot re-verifies; rotation (new revision/digest), revocation,
  expiry, or context change fails closed. Rows:
  `recheck_ok`, `recheck_tampered`, `recheck_mixed_policy`,
  `recheck_wrong_site`, `recheck_stale`.
- Conflicting entries for one revision reject the snapshot
  in-language (`approve_conflict`, same indistinguishable shape —
  the recon rule forbids a distinct kind); snapshot ADMISSION
  rejects them with `SchemaAuthorityInvalid` once the workflow
  exists (below).

## Compiler hooks (future diagnostics, not implemented)

`SchemaAuthorityInvalid` stays a compiler/acceptance diagnostic,
never an `emits` outcome. Reserved emission points:

- conflicting-entries snapshot presented for admission;
- snapshot sequence below the accepted head (replay);
- fixture-signed material presented under production keys;
- caller-supplied site names where a certified resolved call node
  is required (call-node certification itself is future work:
  the compiler, not the caller, will vouch for program+module+
  function+revision+position);
- missing trust root, illegal authority construction, invalid sink
  certification, malformed snapshot (already reserved in S2 for the
  bridge path).

No hook fires until the acceptance workflow supplies genuine
provenance inputs; adding a hook without its input is theater and
is out of scope for every slice.

## Freshness protocol (before deployment acceptance)

1. Acceptance workflow selects head snapshot + policy for the
   program and freezes the record.
2. Construction runs against exactly that record (recheck at the
   boundary where builders consume witnesses).
3. Before deployment acceptance, re-verify currency (head still
   head, interval still valid); revocation or rotation since step
   2 fails acceptance, never silently passes.

## Non-goals

Fetching, TLS/DNS enforcement, response handling, execution,
additional roles or digest profiles, runtime request mediation,
page/DOM fragments, sandbox or harmlessness claims. The registry
approves; it does not fetch, now or later.
