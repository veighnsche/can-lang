# a77 — Revision identity enforcement

Status: shipped. Foundation: [a67 §2.3](a67-testing-contracts-design.md),
[a72 decision 4](a72-variant-design.md), the §2.3 review verdict
(CONFIRM policy/anchoring/strictness/pins/probes; REVISE
fingerprint, canonicalization, baseline, removal, diagnostic,
sequencing). First mechanism: comparison with an enforcement
boundary, not compatibility inference.

## Rule

One revision owns one reviewed interface; callers stay explicitly
pinned; changed implementations require fresh proof. Three
identities — interface drift, proof freshness, test evidence —
are maintained by one mechanism and never conflated: excluding a
field from interface identity does not reuse evidence that
depended on it.

## Fingerprint model

Computed from resolved, non-erased interface declarations plus
their complete identity-bearing shape dependencies:

- Declaration kind, qualified name, canonical owner.
- Function params with names, order, and resolved types;
  return type and declared errors with nominal identities.
- Existing `effects` capabilities (authority, not
  postconditions).
- `requires` block presence, outcome-indexed `ensures` with
  outcome identities, predicates, and resolved bindings —
  including Boolean match blocks in full.
- Referenced records, errors, variants, brands as a
  deterministic transitive shape closure, not just names.
- Brand authority: underlying type, owner, promotion sources,
  export grants — or an explicit disclaimer that this
  mechanism does not protect that authority.
- Bodies and termination annotations are proof-freshness
  inputs, not interface contents. Tests and `given` are
  excluded from interface identity; changing them requires
  fresh test/linkage evidence, never a new API revision.

## Canonicalization (versioned)

Normalization runs on the resolved declaration model and its
format version is recorded; incompatible baseline formats are
refused, never silently rehashed:

- Strip comments and source locations (kept separately for
  diagnostic anchoring); never strip characters inside
  literals — compare decoded literal values.
- Keep categories, nominal identities, ownership, parameter
  order, field names, binding relationships, and absent-vs-
  present contract blocks (removal must not normalize into a
  default).
- Serialize integers exactly; reject duplicate or unresolved
  declarations before set treatment.
- Sort only explicitly unordered collections (variant case
  sets); never sort expression operands or predicate rows.
- Case entries retain parent identity: a moved case keeps its
  tag but changes membership.
- No equivalence proving, no invalid-declaration repair, no
  erasure of nominal ownership or unsupported subexpressions.

## Baseline workflow

The acceptance workflow — never candidate source — selects the
trusted baseline and protected inventory, and reports both:

- Candidate manifests are proposals only. Generation is not
  acceptance; normal checking is read-only over accepted
  history.
- New declarations record explicit `NEW`; new projects
  initialize through an explicit trust-root over an empty
  inventory, never confused with ordinary drift checking.
- Missing or incompatible baseline authority fails (or reports
  explicitly unverified); never substitutes emptiness or the
  candidate tree. Candidate-supplied baselines are rejected as
  authority substitution.
- Trust limit, stated: an actor authorized to replace both
  policy and baseline can bless drift intentionally. The
  mechanism guarantees drift is noticed, not that approval is
  honest.

## Enforcement

- Same revision + changed fingerprint: reject, even when
  apparently compatible. Identity, not compatibility.
- Revision is not hashed into the content fingerprint: rev 2
  with identical content stays a meaningful, silent case.
- Changed revision: callers stay pinned, no auto-retargeting;
  `CodeUsesRev` at the consumer pin, identity diagnostics at
  the provider. A correctly bumped provider never earns a
  drift diagnostic.
- Inventories compare both ways: baseline-only is REMOVED,
  candidate-only is ADDED, shared identities compare
  fingerprints. Renames are removal plus addition without an
  explicit migration. Tombstones catch resurrection. Removals
  anchor at surviving inventory with the old location carried
  in the explanation.
- Freshness: body edits reprove the function and invalidate
  proofs that incorporated it; callee-contract edits invalidate
  dependent proof assumptions transitively; test/`given` edits
  rerun evidence without touching interface identity. Reuse of
  a derivation and validity of the whole chain stay distinct.

## Diagnostic: CodeRevisionIdentity

Numeric allocation at implementation. Message names the
declaration, revision, and changed case/field; `Expected`
carries baseline identity plus the prior structural fragment
and format version; `Found` carries the fragment and change
kind (added, removed, changed, rebound, ownership); `Hint`
restores the accepted interface or publishes/repins — never
"regenerate the baseline". Transitive changes include the
dependency path. Drift, missing-baseline, and stale-pin are
three diagnostics with three repairs. An `canlc explain` entry
follows the a71 convention.

## Probes (verdict §9)

Primary: pass-through consumer unchanged while `Model__Expired`
joins `Model__State rev 1` — current checks pass, linked vector
passes, and only the revision-aware harness may accept: it must
demand `CodeRevisionIdentity` naming the added case before any
linked root executes. Controls on the same corpus: comments and
additive rows stay silent; added case with complete eliminators
is identity rejection without missing-case failure; incomplete
eliminator stays the a75 missing-case rejection; rev bump with
stale pin is `CodeUsesRev`, never a substitute; proper bump and
repin records acceptance; candidate-supplied baselines are
rejected. Assert the specific diagnostic and identity, in
module-order permutations — never just a nonzero result.

## Out of scope

Verifier core (§2.2) starts after this slice specifies the
mechanism; enforcement here is active before any public
contract-bearing stdlib interface is accepted or the variant
release prerequisite is claimed. Compatibility inference,
multi-version loading, and proof caching are explicitly not
this slice.
