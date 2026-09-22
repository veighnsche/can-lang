# a90 — Arm-level `forward call` (proposal)

Status: proposal (pre-decision). No rule, no code.

Parent: the relay-toll discussion behind
`html__text__escape_from` (`std/html/html.can`): all-forward
`match call` matches cost 3 lines per site and cannot be
abstracted today.

## Problem

Every call dispatches `Ok` plus every emitted kind as a
`match` scrutinee (CAN3003/CAN4109), so a relay — "return
what `f` returns" — costs a full match even when every arm
forwards:

```can
"&" => match call html__text__escape_from(orig, s[1:#s], acc + "&amp;", n - 1)
  on Ok r => forward r
  on html.nul_byte e => forward e
```

The toll is per syntactic call site: helper-abstraction
re-pays it per use site (the helper call needs its own
dispatch), and the four escape calls must stay four
(distinct argument tuples). 97 `=> forward` arms exist
across `std/` + sketches; the escape workers are the
motivator, not the only consumer.

## Semantics (recommended)

`<pat> => forward call f(args)` as a whole arm RHS, in
value and call arms alike:

```can
false => match s[0:1]
  "&" => forward call html__text__escape_from(orig, s[1:#s], acc + "&amp;", n - 1)
  "<" => forward call html__text__escape_from(orig, s[1:#s], acc + "&lt;", n - 1)
  ">" => forward call html__text__escape_from(orig, s[1:#s], acc + "&gt;", n - 1)
  _ => forward call html__text__escape_from(orig, s[1:#s], acc + s[0:1], n - 1)
```

At check time each site elaborates into exactly today's
handwritten shape — before proofs, runs, and emit see the
function (the `forward`/`chain` precedent: elaboration, not
proof). The escape body drops from 19 lines to 9 with
identical rows, coverage, and emit.

v1 callees are same-file locals only (self + siblings);
foreign, uses-pinned, and extern callees are refused (they
need a `given` design that does not exist). Chain tail and
`else` keep refusing (existing parse errors cover).

## Checker obligations

1. **Ordering.** A new pass before `elaborateForwards`
   (order const→forwardcall→forwards→chains), exactly once:
   its output contains `forward r/e` arms the existing pass
   must then expand. Same doubling rationale as the three
   existing passes.
2. **Emits inclusion free via ordering.** Elaborated
   error arms face CAN4001 exactly like handwritten relays
   (`TestForwardSameEmitsCheck` precedent): a forwarded kind
   outside the caller's emits reports there, with the same
   message. No subset check of its own.
3. **Locality.** The callee resolves to a same-file `FnDecl`
   (`prog.Fns` + `FnFile`), else the new code. No `given`
   story is needed: locals execute.
4. **Whole RHS.** Only the complete arm RHS elaborates
   (CAN3011-style); `forward call` inside larger expressions
   is refused. The shape already parses through the arm-RHS
   path, so match arms need no `parse.go` change.
5. **`Ok` shape, relays, coverage free via ordering.**
   Elaborated arms hit the existing pass: the `forwardOk`
   shape check, CAN4108 certificates, CAN4107 normal arms.
   Existing rows satisfy migrated sites with zero churn.
6. **Termination free via elaboration.** Elaborated calls
   sit where the site sat, under the same guards, so the
   decreases and cycle walks see ordinary call scrutinees
   (chain obligation 7 precedent).
7. **Spans.** Elaborated arms share the site line; arm
   identity is `(node, index)` (`markTaken`), so obligations
   stay distinct, and LSP anchors to the site line.
8. **Catalog.** `buildCatalog` walks post-elaboration AST
   (chain precedent): elaborated arms appear as handling
   sites with site-line rows, and row-text dedup applies
   unchanged — distinct sites list distinctly.
9. **One code.** CAN3013 `CodeBadForwardCall` for all static
   refusals — CAN3011-family precedent over strict
   one-code-per-refusal.

## Rejected alternatives

- **Fused single site** (`{patterns} forward call
  f(...{pieces})`): patterns bind no scrutinee; the
  positional pattern↔piece zip fails silently (all pieces
  `str`); template substitution is macro machinery; and it
  inverts the deliberate or-pattern shared-outcome rule
  (C7/CAN4111). Saves ~2 lines over the recommendation.
- **`let`-binding** (bind the piece, call once): names the
  callee once but yields 11–13 lines vs 9 — multiplicity is
  conserved — with bigger machinery (env threading,
  shadowing rules). Demand-driven follow-up; `a28` holds
  its slot.
- **User-defined metaprogramming**: collides with CAN4107
  coverage, golden diagnostics and attribution, syntactic
  termination (CAN3005–3009), `given` site identity, `a77`
  revision fingerprints, effects transitivity, and build
  determinism. Compiler-owned elaboration (this note) is
  the blessed species.
- **Multi-call product**: deferred with no consumer, and
  eager evaluation non-terminates on the escape shape
  (call-arm proposal §2).
- **Helper abstraction**: re-pays the toll per use site, as
  shown above.

## Non-goals

Foreign/`given` support; chain-tail support; `let`;
fusion. Each reopens on its own consumer and note, never as
slice growth. (The lint rule suggesting the shape shipped as
C16/CAN3419 alongside the feature.)

## Open questions

1. Locals-only v1? Recommendation: yes.
2. Call-match arms included as well as value arms?
   Recommendation: yes — the mechanism is
   position-independent, and payload-dependent relays
   benefit. (The asymmetry with plain `forward`, refused
   in value arms by CAN3011, is principled: `forward`
   relays an outcome, `forward call` creates the dispatch.)
3. Migration breadth: the two motivating workers only
   (recommendation; `a86` precedent) vs every
   mechanically-convertible site?

## Toward approval

Accepting this note means: implement as one slice
(CAN3013 registry + explain, `compiler/forwardcall.go` +
`buildWorld` wire-up, `forwardcall_test.go` probes
mirroring `forward_test.go`, escape-worker migration with
regenerated goldens, `REQUIREMENTS.md` + can-idioms Tier 1
(guide since removed) amendments), with emit parity — elaborated vs handwritten
byte-identical TS — as the strong check. Migration edits
`std/html/html.can` byte-exactly with the NUL count
asserted before and after. Only then consider follow-ups.
