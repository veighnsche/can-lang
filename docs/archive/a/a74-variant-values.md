# a74 — Variant value admission

Status: shipped. Foundation: [a72 design](a72-variant-design.md)
(amended: constructors are case-qualified, matching binds payloads),
[a73 registry](a73-variant-registry.md). Matching itself is a75.

## Rule

A declared case constructs in any value position by naming the
case-qualified constructor (`Parent__Case`) with exact named fields.
The constructor carries the **parent** type outward (nominal typing):
a `Login__Authenticated(...)` is a `Login__State`, never its payload
shape. The wrapper discipline holds: bare variants are not function
bodies, returns, or test expectations — functions return records,
tests expect `Ok(...)` or errors.

## Admission surface

- Construction: `Parent__Case(field = v, ...)` with exactly the
  declared case fields; unknown, missing (non-eval paths mirror
  records), repeated fields, and field type mismatches rejected at
  check time. Positional construction stays outside the subset.
- Nominal typing: constructor's static type is the parent variant;
  field, param, arg, and Ok-payload positions compare through it.
  Payload-shape subtyping is not admitted: a record with the same
  fields is not a case.
- Evaluator carrier: `Kind "variant"`, `Tag` = qualified case name,
  `Dict` = payload. Never `Kind "err"`: variants are data and must
  not trip error paths (`on e.kind` arms, emits accounting, catalogs).
- Equality: vEq compares tag identity first, then payload. A case
  never equals a record, an error, or a different case — even with
  an identical payload.
- Description: canonical form names the qualified case; distinct
  from `err(...)` and `Ok(...)` renderings.
- Stub/outcome positions: `given` tables and bare test expectations
  still accept only `Ok(...)` and errors — a bare case there is
  rejected. Variants ride *inside* `Ok(...)` payloads.
- Catalogs: constructing a case creates no `errors.json` entry and
  feeds no emits accounting. `eachRaise`-style scans see only
  dotted errors.
- Returns: `-> Login__State` is rejected like bare-brand/Seq/Bytes
  returns: entries return wrapper records.
- Equality boundary: `==`/`!=` over variants is refused at check
  time (like Bytes/Seq): cases compare by matching in a75, never
  by `==`. Structural comparison lives in the test evaluator
  (vEq) only, so expectations over variant payloads verify, and
  `==` over wrapper records containing variants stays correct in
  emit (the tag key compares structurally).
- Identity prerequisite: a variant parent colliding with a record
  type name is rejected (variant identity collision), so emit
  never maps one TS name to two shapes. Case/record collisions
  were already rejected in a73.

## TypeScript emission

- One exported union type per variant, one member per case, each
  member carrying the qualified tag as a discriminant.
- Constructor functions (or literals) producing the tagged shape.
- Cross-module type references import from the provider stem:
  variant parents join the stem table like records.
- Strict `tsc` (not lenient) gates emitted unions from this version
  on, per the a72 verdict.

## Out of scope

Matching/binding (a75), pilot consumers (a76), §2.3 identity beyond
the existing global case registry, verifier work.
