# a76 — Pilot consumer

Status: shipped. Foundation: [a72 design](a72-variant-design.md)
(slice 4), [a73 registry](a73-variant-registry.md),
[a74 values](a74-variant-values.md),
[a75 elimination](a75-variant-elimination.md). First real
module using variants end to end; acceptance for the epic.

## Rule

One new module, one operation, no new language surface. If the
pilot needs a language change, that is a bug in a73–a75, not
scope: stop and report instead of inventing surface.

## The pilot: form-state-to-message

Provider `form`: a `Form__State` union (nullary `Empty`,
`Editing` with a record draft, `Submitted` with a scalar)
eliminated to a message string. Every arm witnessed by the
decision table; nullary and payload cases distinguished in
expectations.

Consumer `shell`: pins the parent, calls `form__message`
through `uses`, scripts variant-carrying exchange args in
`given`, and passes the union through unexamined (`echo`
field) beside the matched result — the decision-4
passthrough shape living next to a real elimination.

## Proof obligations

- Diagnose clean on both modules (inline probes, then the
  committed sketches).
- Linked execution through the settled runner: `shell__greet`
  over `Empty` and `Submitted` runs the real provider, no
  scripts. Contradiction control: a wrong expectation fails
  with a payload mismatch (the CAN3110 shape); a wrong-tag
  echo fails on tag identity in linkage.
- Catalog absence: `errors.json` carries no `Form` entries.
- Generated artifacts: committed `form.ts`/`shell.ts`
  goldens, byte-compared by the suite, strict-tsc green via
  the existing sketches gate. No grammar change, so no
  editor/canlc refresh beyond the normal build.

## Out of scope

Stdlib work, a second consumer, §2.3 identity (drift stays
open and claimed-open), verifier work, new surface.
