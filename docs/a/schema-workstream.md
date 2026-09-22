# Workstream prompt: Schema / asset approval (unblocks HTML Assets)

## Goal

Define the Schema layer's first deliverable: the asset-approval
machinery that unblocks the HTML Assets bundle (`html__asset__stylesheet`,
`html__asset__script`, registry). No workstream owns this today — the
catalogue demands it, no plan schedules it. This prompt scopes the
minimum that makes Assets buildable, nothing more.

## Why this exists

A stylesheet or script tag points at the outside world. The catalogue
contract is "approved external assets, not raw CSS or script strings" —
the approval *is* the feature. Without it, `script` would bless
arbitrary script sources (stored XSS by construction), and
`stylesheet` would be a bare `link` wrapper adding nothing. The HTML
library cannot invent this model from inside; it must arrive as a
language layer, like Functions, Async, and Resources (see
`docs/a/a13-stdlib.md:118`, where Schema is named with those layers).

## Input: settled trust model (gating)

This workstream starts from the rulings produced under
`docs/schema-design-prompt.md`, after adversarial security review —
not alongside them, not before them. The twelve attack-surface items
arrive as settled rulings with rationale and failure rows; the open
questions above arrive answered. If any ruling is missing or marked
`unresolved`, the workstream stops at the boundary and says so rather
than assuming a trust decision. No slice may bake in an unmade ruling.

## Required surface (from the consumer)

- `ApprovedAsset`: a value proving an external asset passed approval.
  What it carries is OPEN (URL + integrity hash is the obvious shape;
  host allowlist is weaker — decide explicitly).
- `AssetPolicy`: the policy an approval is checked against. Who sets
  it, where it lives, and who can change it are OPEN (per-program
  declaration is the obvious shape; per-module is weaker).
- Registry: where approved assets are recorded and looked up. Format,
  location, and who may add entries are OPEN. Minimum: the HTML
  builders can ask "is this asset approved under this policy" and get
  a checkable yes/no, with rejection as a declared, value-carrying
  error kind (never a silent empty).
- `stylesheet` and `script` must be implementable as
  `(asset: ApprovedAsset, policy: AssetPolicy) → Html__Safe` with no
  wall-smuggling (no raw-URL parameter, no unseal, no stringly approval).

## Constraints from the existing codebase

- Approval is a construction-time check with committed positive and
  negative rows, like every gate in `std/html` (named gates + `make`'s
  duplicate detection are the template: validate at the boundary,
  carry the proof as a type).
- Fault contracts hold: every rejection is a declared error carrying
  the offending value; new kinds follow the promised operation.
- The checker rules apply to any std-level wrappers (declared emits
  cover callee kinds; exhaustive arms; identity relay on self-calls).
- Non-goals: inline script/style stay excluded (not approved — absent);
  the registry approves, it does not fetch; no fetching, no execution,
  no HTTP layer in this workstream; the HTML library consumes Schema,
  it does not define it.

## Attack surface (design against all of these)

Approval is a security boundary, not metadata. Each item below needs
an explicit ruling in the design, not a silent default.

- **Forged approval.** A raw URL, a boolean flag, or a stringly token
  where `ApprovedAsset` is expected. The proof must be a type the
  caller cannot construct — same rule as every brand in `std/html`.
- **Confused deputy.** One module's approval spent in another's
  context. Approval must bind the asset, the role (stylesheet vs
  script), and the consuming site — the Bytes B2 owner-local authority
  model is the template (grant, brand, and function share identity).
- **Registry poisoning.** If any program or module can add its own
  entries, approval is theater. Write access to the registry is the
  whole game: name exactly who approves, and what stops self-approval.
- **Host without integrity.** An approved host serving a changed file
  bypasses host-only approval silently. Decide: content hash required,
  host allowlist required, or both — each with its failure row.
- **Time-of-check vs time-of-use.** Approved now, changed before fetch.
  Approval at construction and verification at retrieval are two
  different moments; name which one this workstream owns and which one
  it explicitly leaves to the future fetching layer.
- **Plain transport.** An approved `http` asset is MITM-rewritable.
  Approval should pin secure transport or justify the exception.
- **Stale approval.** A compromised asset stays blessed forever without
  removal or expiry. Registry needs deletion/rotation, not just adds.
- **Transitive inclusion.** An approved script that loads further
  scripts, an approved stylesheet with `@import` chains — approving
  the entry does not approve the tree. Rule the transitive case or
  forbid the loading primitives; do not leave it silent.
- **Role mismatch.** CSS approved, spent as script (MIME confusion).
  Approval binds the asset to its role; cross-role use fails closed.
- **Approval is not sandboxing.** An approved script still runs with
  full page privileges; an approved stylesheet can still exfiltrate
  via CSS callbacks. Approval answers "is this the file we meant" —
  never "is this file harmless." Version pinning (exact, not ranges)
  belongs in the decision.
- **Registry recon.** Rejection errors carry the offending asset (fault
  contracts), never registry contents — errors must not enumerate what
  *is* approved.
- **Fixture authority.** Seals in test/given data create no approval,
  same as export certificates: test fixtures prove the check runs,
  never that the asset is blessed.

## Acceptance

- Both asset builders implementable with the approved-only contract;
  an unapproved asset fails loudly at construction.
- A hostile-asset fixture set: unapproved URL, wrong hash, policy
  mismatch, registry removal — each rejected, each with its row.
- Full suite green (`go test ./...`, modcheck, gramcheck); no changes
  required to existing modules (Assets adapts to Schema, not vice versa).
