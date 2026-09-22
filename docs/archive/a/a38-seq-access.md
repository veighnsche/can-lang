# a38 — Seq checked access and traversal, S3

Status: shipped. P0 approved under standing approval; P1–P4
implemented and green: `[]` extended to `Seq<T>` bases
(element-typed `typeOf`, member fetch, generic `$canSeqAt`),
checked-get and traversal workers with decision tables in
`compiler/seq_s3_test.go`. One correction from review: the
early-stop `short` row takes `B` too, so the CAN4107 probe drops
zero/negative-budget AND short rows — full traversals still end
in `E`, never `B`.

## Goal

Read one element by index with static element-type preservation,
demonstrated by a terminating full traversal. This is the last
traversal proof the customers need (`str__join` walks forward;
`split`/`fragment`/`attributes` walk and build).

## Success Criteria

- `xs[i]` over `xs: Seq<T>` evaluates to the member with static
  type `T` (brands preserved: `Seq<M__B>[i]` is `M__B`, not `str`).
- Out-of-range raw access never clamps or wraps: it fails loud
  (`seq index out of range`), reachable only by unguarded use.
- A bounded worker visits every element exactly once with `fuel =
  #xs + 1`, terminating under the existing `decreases` rule — no
  new termination theorem.
- `go test -count=1 ./...`, `modcheck`, `gramcheck` green.

## Context And Current Facts

- `s[i]` over `str` is raw guarded-at-use-site access: the
  compiler checks shapes, `.can` guards ensure bounds, and
  `std__str__scalar_at` (`std/text/text.can:89-104`) maps the
  guard failures to `text.index_out_of_range`. There is no
  catch mechanism for index failures, in `.can` or the compiler:
  an unguarded out-of-range index is a loud evaluation error,
  never a value. S3 follows this shape exactly.
- S1 gives `Seq<T>` types and values; S2 gives `#` bounds. Raw
  `[]` over Seq is therefore a three-site extension (checker
  operand rule + element-typed `typeOf`, evaluator member fetch,
  emitter lowering), same size as S2.
- The existing termination rule (`docs/archive/a/a08-termination.md`,
  `checkDecreases`): `decreases fuel` decl, canonical `fuel <= 0`
  guard, self-call under the false arm with exactly `fuel - 1`.
  `std__str__find_from` is the house worker shape (guard match,
  work match, relaying `match call` recursion).

## Constraints And Non-goals

- One operation: indexed read. No append/concat/slice, no
  sequence equality, no customers.
- No new catch/option surface: the checked wrapper is guards +
  an error constructor, exactly like `scalar_at`.
- No `std/seq` module yet: the checked-get wrapper and traversal
  worker live in the test module as proof. The canonical
  `sequence.*` error namespace is minted with `std/seq` at the
  first customer slice, which also decides what else the module
  carries. The error SCHEMA (`index`, `length` fields) is
  ratified here; only the namespace moves.
- No `REQUIREMENTS.md` edit: proposal only.

## Key Decisions

1. Extend `[]`, not a `seq__get` kernel call. Same argument as
   S2 (`#`): a kernel needs call-resolution plumbing for
   identical semantics, and `[]` is already the index spelling.
   `std__seq__get` is realized by the guard idiom, not a second
   spelling.
2. Static type of `xs[i]` is the element type `T`. A brand
   member stays branded at check time (it erases to a string
   only at runtime, a10/a15 as usual).
3. Traversal worker shape (parameters `xs, position, fuel,
   count, trace`):

   ```text
   B: fuel <= 0 → return count and trace
   P: otherwise, match position < #xs
     E: false → return count and trace (ordinary exhaustion)
     I: true → recurse position+1, fuel-1, count+1,
        trace+"("+xs[position]+")", relaying Ok
   ```

   Entry: `position=0, fuel=#xs+1, count=0, trace=""`.
   Invariant `position + fuel = L + 1`: `L` steps take `I`,
   then `E` fires with `fuel=1` still left — the terminal
   out-of-range position is ordinary exhaustion, and `B` is
   genuinely untaken by full traversals (witnessed instead by
   direct zero/negative-budget rows). No new termination
   theorem: the proof is the existing guard plus `fuel - 1`.
4. Test-local error kind `t.out_of_range(index: int, length:
   int)` for the wrapper rows. Schema matches the future
   canonical `sequence.index_out_of_range`; namespace only.
5. TS lowering for `a[i]` is a generic `$canSeqAt<T>(a: T[], i:
   bigint): T` helper that throws `seq index out of range`
   outside bounds — symmetric with `$canStrAt`, types precise,
   no `any`. Agreement with the evaluator holds on valid
   (guarded) programs; invalid programs already fail the build
   in Go before TS matters.

## Work Plan

1. P0 — Proposal (this file). No code.
2. P1 — Checker: `stridx` accepts `Seq<T>` bases (index still
   `int`); `typeOf` yields the element type; non-Seq non-str
   bases keep the pinned diagnostic verbatim.
3. P2 — Eval: `seq` base fetches `Arr[i]`; out-of-range (incl.
   non-int64) is `seq index out of range`, never a clamp.
4. P3 — Emit: `Seq<T>` base lowers to `$canSeqAt`; helper
   emitted only when used.
5. P4 — Rows below in `compiler/seq_s3_test.go`; README index
   row; three gates; commit.

## Validation Plan

Checked-get wrapper `t__get(xs, index)` (guards à la
`scalar_at`):

| Sequence, index | Expected | Arm |
| --- | --- | --- |
| `S["b", "a"], 0` | `Ok("b")` | ok/ok |
| `S["b", "a"], 1` | `Ok("a")` | ok/ok |
| `S[""], 0` | `Ok("")` | ok/ok |
| `S["a"], -1` | `t.out_of_range(index=-1, length=1)` | outer-false |
| `S["a"], 1` | `t.out_of_range(index=1, length=1)` | inner-false |
| `S[], 0` | `t.out_of_range(index=0, length=0)` | inner-false |
| `Seq<M__B>[seal("A")], 0` | `Ok(seal("A"))` as `M__B` | ok/ok, brand preserved |

Traversal `t__walk_from` (all rows `count=0, trace=""` entry):

| `xs`, position, fuel | Expected | Arms |
| --- | --- | --- |
| `S["x"], 0, 0` | `Ok(0, "")` | `B` |
| `S["x"], 0, -1` | `Ok(0, "")` | `B` |
| `S[], 0, 1` | `Ok(0, "")` | `P, E` |
| `S["x"], 0, 2` | `Ok(1, "(x)")` | `P, I, R, E` |
| `S["b", "", "a"], 0, 4` | `Ok(3, "(b)()(a)")` | `P, I, R, E` |
| `S["a", "b"], 0, 1` | `Ok(1, "(a)")` | `P, I, R, B` |

Entry wrapper `t__walk` (`fuel=#xs+1`): `[]`, `[""]`,
`["A", "", "&B"]`-shaped rows proving end-to-end order.

Negative rows:

| Mutation / input | Expected |
| --- | --- |
| Drop `decreases fuel` | `CAN3005` (proof travels with the worker) |
| Worker without zero/negative-budget/short rows | `CAN4107` (`B` really untaken by full traversals, which end in `E`) |
| Unguarded `xs[5]` on length 2 | test fails `seq index out of range` (no clamp) |
| `xs["a"]` (non-int index over Seq) | existing `cannot index with` CAN6003 |
| `5[0]`-family | existing base rule, byte-identical |

Remaining `decreases` mutations (`fuel-2`, recursion outside
the guard, Seq-typed measure) are owned by the existing
termination suites; S3 cites them, not duplicates them.

## Risks / Rollback

- Risk: `[]` overloading confuses scalar vs element reads.
  Mitigation: operand type decides, same as `#`; diagnostics
  name `str` only for non-Seqs.
- Risk: TS/Go divergence on unguarded access (throw vs
  undefined). Accepted: invalid programs fail the Go build
  first; agreement is claimed on valid programs only.
- Rollback: revert the three `stridx` arms plus helper and
  the test file; string indexing byte-identical before/after.

## Open Questions

- None for S3. S4 owns whether append is a kernel or second
  `[]`-family spelling (slice assignment is NOT proposed).
