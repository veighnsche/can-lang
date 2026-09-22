# a72: closed tagged unions (borrow #3) — design

Status: decided; slices shipped (a73 registry,
a74 values, a75 elimination, a76 pilot). Variant
sequences in field position remain deferred
(`checkDeclFields` Decision 6).
Parent: `docs/ASTRA_FSHARP_BORROW.md` §3 (finite,
monomorphic, closed unions). This doc adapts that
sketch to CAN specifics. Verdicts: keep closed
unions; semantic boundaries and slice order revised
as below. Slices a73–a76 have since shipped.

## Goal

Make invalid state combinations unrepresentable for
the stdlib rows waiting on this (HTTP methods, form
states, validation results), reusing the existing
outcome machinery (ctors, variant patterns,
`$can_kind` discrimination) instead of inventing a
parallel case system.

## Syntax (proposed, not parsed)

Declaration mirrors `type`, with `case` rows:

```can
variant Login__State rev 1 (
  case Anonymous()
  case Authenticated(session: Auth__Session)
  case Locked(user_id: str, remaining_seconds: int)
)
```

Case names are qualified (`Login__Anonymous`,
`Login__Authenticated`, `Login__Locked`): the
declaration binds the short row spelling to the
qualified name, and every other position spells the
qualified name. R3 naming already admits the shape.

Construction reuses kwargs ctors:

```can
Login__Authenticated(session = s)
Login__Anonymous()
```

Patterns reuse variant arms on value matches:

```can
match state
  on Login__Anonymous _ => Ok(message = "Sign in")
  on Login__Authenticated a => Ok(message = a.session.user_id)
  on Login__Locked l => Ok(message = l.user_id)
```

## Key decisions

1. **Closed, checked at match time (CONFIRMED).**
   A value match whose scrutinee has variant type
   `V` must contain exactly one arm for every case
   of `V`. An unqualified catch-all `_` and
   residual/rest syntax are rejected. `on V__Case _`
   remains legal: it selects one named case while
   discarding that case's payload (the existing
   `variantWild` is a named-case discard, not a
   catch-all). Case-independent operations may pass
   a `V` value without matching it — the closure
   claim is "added cases break incomplete
   eliminators", not "added cases break every
   consumer". Matches over non-variant scrutinees
   keep today's rules untouched, including `_`.
   Negative controls: missing case, duplicate case,
   a case from another union, `_` before or after
   named arms, `on Case _` missing one sibling.
2. **One tagged representation, two semantic
   categories (REVISED).** A common tagged-payload
   carrier and shared structural algorithms are
   reused; error semantics are not. Variant cases
   get their own membership registry — never
   inserted into `Program.Errors` or `EmitsOf`.
   Consequences, each pinned by slice probes:
   whole-union `vEq` includes resolved case
   identity (`Choice__First() ≠ Choice__Second()`
   despite equal empty payloads, in direct
   expectations, nested records, and linkage
   comparisons); normalization/`describe` keeps a
   canonical variant-constructor format distinct
   from the `err(...)` envelope; `Ok(state =
   Login__Locked(...))` scripts as success while a
   bare `Login__Locked(...)` is rejected as
   neither `Ok` nor a declared error, and
   constructing/projecting a data case creates no
   `errors.json` entry; data-case arms need
   execution evidence and never qualify for the
   error-relay exception; a case belongs to
   exactly one nominal union (tag/field shape
   alone never admits it elsewhere). Case binders
   are case-specific payload views (`a.session`
   fails in the `Locked` arm); returning the whole
   union means returning the scrutinee or
   reconstructing its case. Structural evaluator
   equality is mandatory; no new `==`/`!=` over
   unions rides along. Constructor checking
   validates exact named fields, duplicates,
   omissions, unknown fields, and field types.
3. **TS mapping reuses `$can_kind` (CONFIRMED,
   with obligations).** Each case emits as a union
   member tagged with its qualified name; nullary
   cases emit as tag-only members. Three
   obligations: the scrutinee evaluates once into
   a generated temporary, the `switch` reads that
   temporary's tag, and payload binding narrows
   the same temporary (never re-evaluate for
   fields). Functions returning unions keep the
   outer outcome envelope (`{ $can_kind: "ok";
   state: Login__State }` — bare variant returns
   would need their own decision). A defensive
   emitted default follows existing target policy
   but never excuses an incomplete source match.
   "No new runtime helpers" covers construction
   and dispatch; union type definitions, imports,
   constructor tags, temporary typing, and
   case-specific binding still need emitter
   plumbing.
4. **Exhaustiveness now, identity separately
   (REVISED).** Exhaustiveness checks the current
   union declaration and rejects every currently
   incomplete match — but that does not close
   same-revision drift: a passthrough consumer
   with no eliminator (`Ok(state = state)`) passes
   unchanged while its pinned `rev 1` silently
   gains a case. Revision integrity is therefore a
   separate obligation: case add/remove and
   payload-shape changes require a revision change
   checked against a previously accepted identity
   baseline, and until that mechanism exists the
   compiler claims no same-revision drift
   detection. Recommendation adopted: the §2.3
   identity slice is a prerequisite to public
   variant release. The variant fingerprint covers
   qualified identity, resolved case names, named
   payload fields/types, and depended-upon shapes;
   cases inherit the parent revision (no floating
   case revisions); drift diagnostics point at the
   changed case/field (or the declaration on
   removal). Two probe families stay separate:
   added-case-with-incomplete-match (proof) vs
   added-case-at-same-rev-with-complete-matches
   (identity — a version-bump test is not a
   substitute).
5. **Inhabited variants only (CONFIRMED,
   rationale revised).** At least one case is
   required; all admitted payload dependencies
   satisfy the nonrecursive admission rules. Empty
   types and elimination from impossible values
   are out of scope (an empty type can express
   impossibility — just not in this epic). A
   nullary case has no payload fields and emits
   only its discriminator. Tag identity
   distinguishes nullary cases everywhere
   (`Choice__First() ≠ Choice__Second()` in
   direct expectations, nested records, and
   linkage).
6. **Named payloads, combined cycle check, errors
   separate (REVISED).** Payload fields use
   existing named, supported types; no anonymous
   payload types, no new generics. Cycle
   validation traverses the combined
   record/variant dependency graph at declaration
   admission (a variant→record→variant loop must
   fail even though the variant never names
   itself), before value admission. `Seq<Variant>`
   is deferred (existing records with preexisting
   Seq fields keep working; no new
   variant-sequence surface slips through a
   generic known-type check). Errors retain their
   outer-outcome role, `emits` contracts, and
   failure-injection semantics; error values never
   become cases implicitly. Explicit error-to-data
   adapters (one adapter per boundary, payloads
   preserved as specified, no second validator
   generated from cases) are the accepted cost of
   the split — e.g. mapping `validation.*` errors
   into `Form__Validation` data cases.
7. **Single scrutinee, type-enforced (CONFIRMED,
   strengthened).** A match containing a
   variant-typed scrutinee has exactly one
   scrutinee, enforced from checked scrutinee
   types before pattern proof — even when the
   pattern is `_` (`match state, enabled / _, true
   => ...` is rejected). Nested single-scrutinee
   matches stay available with their own
   exhaustiveness and coverage obligations.
   Non-variant multi-slot rules are unchanged.
8. **Case ownership is explicit (new).** A case has
   exactly one parent variant and one globally
   unambiguous qualified constructor name.
   Duplicate case identities — including
   collisions with record constructors — are hard
   errors, never first-wins. The parent variant is
   the provided/revision-pinned item; importing it
   makes its declared public cases available,
   without independent case revisions. Short
   declaration-row spellings elaborate by one
   documented rule and receive exactly the same
   collision checks.

## Rejected alternatives

- **Open variants / catch-all `_`:** defeats the
  add-a-case invalidation, which is the feature.
- **Unqualified case names with import resolution:**
  a second name-resolution mechanism for zero
  gain; qualification is already the R3 shape.
- **Separate tag field (not `$can_kind`):** forks
  emit, match lowering, and the tsc gate's
  discriminated-union reading for no semantic gain.
- **Merging errors into variants (UPHELD for
  this epic):** preserve the outer-outcome
  protocol with explicit error-to-data adapters.
  Correction adopted: shared tagged storage does
  not inherently destroy failure injection — the
  argument is the semantic/compatibility
  boundary, not a theorem.

## Slices (redrawn per verdict, each probe-first + gates)

1. a73 — declaration/registry foundation: parsing,
   AST, qualified case membership, ownership +
   collision rules (decision 8), duplicates, empty,
   bad payload types, combined cycle checking,
   parent-revision resolution. Explicitly frontend
   foundation: no executable variant values yet.
2. a74 — complete value admission: construction,
   exact fields, nominal typing, all admitted
   value positions, evaluator representation,
   `vEq`, normalization, TS type/constructor
   emission, cross-module type references,
   wrapper-return discipline. Strict-tsc checking
   starts here. Variant matches explicitly
   unavailable until a75. Public admission waits
   for the §2.3 identity check while decision 4's
   enforced integrity stands.
3. a75 — complete elimination: case patterns,
   case-specific binders, exhaustiveness,
   duplicate/wrong-case rejection, `_` and
   multi-slot refusal, runtime dispatch,
   evaluate-once TS switch, CAN4107 behavior
   (every accepted arm executes; error-relay
   semantics unchanged).
4. a76 — one pilot consumer: one operation in a
   new module (form-state-to-message suffices —
   no HTTP subsystem import), with real nullary
   and payload cases, witnessed handlers,
   foreign scripted callability, linked execution
   through the settled runner, generated
   artifacts.

## Probes (revised)

- `Login__Expired` added, one match left
  incomplete: proof rejects the missing case;
  complete consumers pass unchanged (positive
  control exercises the same pipeline).
- `Locked` arm projecting `session`: rejected
  (case-specific binder).
- Same-revision case addition with all matches
  complete (or no matches at all): identity
  check rejects drift — a version-bump test is
  not a substitute.
- Nullary identity: `Choice__First()` vs
  `Choice__Second()` distinguished in
  expectations, nested records, linkage; plus
  the CAN3110 fixture (provider `Ok(choice =
  Choice__First())` vs scripted `Ok(choice =
  Choice__Second())` must contradict; bare
  `Choice__Second()` rejected as an outcome;
  error-injection control and `errors.json`
  absence asserted).
- Emit: tag-only nullary member; evaluate-once
  `switch`; strict-tsc green from a74.

## Open questions for the chatbot

None structural — positions above are all taken.
The verdict should CONFIRM/REVISE each numbered
decision, with counterexamples where REVISE.
