# a75 — Variant elimination

Status: shipped. Foundation: [a72 design](a72-variant-design.md)
(decisions 1–8), [a73 registry](a73-variant-registry.md),
[a74 values](a74-variant-values.md). Matching is the last
closed-world proof before the a76 pilot.

## Rule

A value match whose single scrutinee has variant type eliminates
by case: exactly one `on Case binder` arm per case of the current
declaration. The binder names the whole case value; payload
projects through field access. `_` discards a payload but never
stands in for a case.

```can
match state
  on Login__Anonymous _ => Ok(message = "Sign in")
  on Login__Authenticated a => Ok(message = a.session.user_id)
```

## Checking

- Scrutinee typing first: the match takes the case path only
  when exactly one scrutinee resolves to a variant parent
  (param, field path, binder, or constructed value). Anything
  else keeps today's bool/str/wild rules untouched.
- Case arms only: bool, str, and `_` arms in a variant match
  are rejected (CAN4105). A second scrutinee alongside a
  variant is rejected even for `_` (CAN4105).
- Membership: every arm names a case of the scrutinee's union.
  Unknown cases, `Ok`, dotted errors, and cases of another
  union are stale arms (CAN4102), mirroring call matches.
- Duplicates rejected (CAN4105). Missing cases rejected one
  diagnostic per case (CAN4101), mirroring the call-match
  missing-outcome shape.
- Case-shaped patterns against a non-variant scrutinee are
  rejected (CAN4106), mirroring the proof's non-call-match rule.
- Binder env: the binder is typed as the qualified case, so
  projection resolves through the case's own fields and a
  same-named field of a sibling case does not leak across
  arms. Whole-union smuggling stays out: the binder's static
  type is the case, never the parent — returning the union
  means returning the scrutinee or reconstructing the case.
  Binder shadowing follows the existing open question (left
  undiagnosed, like call binders).
- Arm bodies check against the threaded want, like `Ok` arms.

## Proof, coverage, catalogs

- The value-table proof tolerates case-shaped patterns and
  proves nothing about them: the checker above owns every
  variant-match diagnostic, so the proof never contradicts it
  with a legacy bool/str verdict.
- CAN4107 walks every match node generically: each case arm
  must execute across the decision table. Data-case arms never
  qualify for the error-relay exception (MatchCall-only by
  construction; relay also requires a dotted error identity).
- Catalogs see nothing: eachRaise and the handling-site scan
  are dotted-error-only by construction.

## Runtime and emit

- Eval dispatches on the carrier tag and binds the case value;
  a miss is unreachable past the checker and fails closed.
- TS lowers to a switch on a generated temporary's
  `$can_kind` (evaluate once), `const binder = tmp` per
  payload arm, narrowing the same temporary for fields, and a
  defensive default that throws — the call-match shape,
  which strict tsc already narrows through.
- `==` over unions stays out (a74): matching is the
  comparison story.

## Out of scope

Same-revision drift (§2.3 identity slice), the a76 pilot,
verifier work.
