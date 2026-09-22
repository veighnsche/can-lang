# Workstream prompt: Bytes primitive type (issue #42)

## Goal

Add a `Bytes` primitive type to the language, closing issue #42 and
unblocking two waiting consumers: `html__render__utf8` (Render bundle)
and the text conversions in `std/text/text.can` (hex/base64/utf8).

## Why a workstream, not a slice

A new primitive touches the full compiler pipeline — literal syntax,
type rules, evaluation, TypeScript emit — at the same scale as `Seq`,
which took multiple version slices (a36 literals, a37 length/indexing,
…). Same files the type-system thread is cutting in now; coordinate,
don't collide.

## Required surface (from the consumers)

- `utf8` encode: `str → Bytes` (this alone unblocks Render).
- `utf8` decode: `Bytes → str` with a failure for invalid sequences
  (new error kind; errors carry the offending value, per convention).
- `hex` encode/decode, `base64` encode/decode (unblocks text.can).
- Empty bytes value; equality on bytes (needed for tests).
- Literal syntax: OPEN QUESTION, yours to decide. Candidates: hex
  literals, constructor from `Seq<int>`, or both. `Seq` literal
  syntax (`Seq<T>[…]`, a36) is the template to mirror or reuse.
- TypeScript emit representation: OPEN QUESTION. `Uint8Array` is the
  obvious candidate; pin it explicitly.

## Constraints from the existing codebase

- `Seq` slices are the process template: one capability per version,
  each with direct-row tests before the next lands.
- Fault contracts hold for the new type: every failure is a declared,
  value-carrying error kind; NUL policy (`docs/encoder-nul-policy.md`)
  applies to the utf8 boundary in both directions.
- If you write std-level wrappers with tests, the checker enforces
  three rules we learned the hard way: a caller declares every kind
  its callees can raise (even caught ones); matches are exhaustive
  over callee emits; recursive calls must re-raise errors unchanged
  (no kind conversion on self-calls).
- Non-goals: do not widen `Html__Safe`; do not touch the script/style
  exclusion; no stringly byte-passing (a `str` carrying bytes is the
  fake completion this workstream exists to prevent).

## Acceptance

- `(document: Html__Safe) → Bytes` is implementable as
  `html__render__utf8` with no new wall (no brand smuggling, no
  unseal).
- The `text.can` conversions land on the new type.
- Full compiler suite green; no existing `.can` module changes
  required (consumers adapt to Bytes, not vice versa).
