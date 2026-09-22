# v1.2 — Producer-owned contracts + complete observations (spec)

Status: shipped. Amendment to R5 (errors), R8 (stubs), and R10
(tooling): `emits` becomes a producer-owned upper bound, error
expectations become complete values, and script rows become
request/response exchanges. No new semantics for bodies, arms,
coverage, or emit — this change is entirely about what the tables
prove. Shipped sketches migrate mechanically; production emit is
byte-identical.

## The hole (why this exists)

Three related defects, verified against the compiler during this
amendment:

1. Consumer stubs manufactured provider honesty. `db__get_user`
   declares `db.down` and never raises it, yet compiled clean —
   because callers stubbed `db.down`. The identical provider alone
   failed the stale-emits rule; with a stubbing consumer it passed.
   The verdict on a producer's declaration flipped on
   consumer-authored text.
2. Scripts proved outcomes, not requests. Changing the login lookup
   from `db__get_user(id)` to `db__get_user("the-wrong-user")`
   changed nothing: all 14 tests still passed, correctly typed.
3. Error expectations proved kinds, not payloads. Returning
   `auth.login_failed(user_id = "somebody-else")` for user `id`
   preserved every error expectation.

## Rule

- `emits` is an explicit conservative upper bound. Unrealized
  entries are allowed (`db.down` needs no stub, no raising body);
  every entry must name a declared error. The stale-emits refusal
  (`CAN4003`) is deleted with the circularity it enforced.
- Error expectations are complete constructions:
  `=> auth.login_failed(user_id = "u_99")`, compared kind and
  payload. Bare kinds (`=> auth.login_failed`) prove nothing about
  the payload and are refused (`CAN3204`).
- Every script row is an exchange:
  `key => [exchange args (id = "u_99") outcome db.user_not_found(id = "u_99")]`.
  At each hit the actual call args resolve through the callee
  signature (positional included) and must equal the expected args
  under the same names, with nothing unexpected; then the outcome
  must be `Ok` or a callee-declared emit, as before. Outcome-only
  rows are refused (`CAN3109`). Zero-arg calls write
  `exchange args () outcome ...` — one spelling, no exceptions.
- `exchange`, `args`, `outcome` join the control keywords (grammar
  + gramcheck). Rows stay single-line: the shape fits the existing
  one-expression-per-row grammar (this deviates from the audit's
  multi-line sketch, deliberately — same proof, no new block
  structure).

## Theorem (written down, not waved at)

A script proves "this request received this permitted response":
same names in, equal values in, declared outcome out, consumed
exactly once in order. Totality, consumption, leftover, and `-`
rules are unchanged.

## Implementation

- `check.go`: the stale-emits loop, `stubbedKinds`, and `CAN4003`
  are deleted; `checkEmits` keeps unknown-kind and foreign-raise
  and gains the emits-entry existence check. `checkStub` requires
  the exchange shape (`CAN3109`) and validates outcomes as before.
- `types.go`: expectations accept dotted error constructors
  (existence + field checks); bare kinds are `CAN3204`. Stub
  checking descends into exchange outcomes and types expected args
  against callee params. Extern manifests gain the same
  emits-entry existence check.
- `parse.go`: `exchange args (...) outcome ...` parses depth-aware
  (balanced parens, full expressions for arg values); expected args
  must be named. `Small` gains `Outcome`; walkers cover it;
  `exchange` outside a script row fails loud in eval.
- `eval.go`: complete error expectations compare kind + `vEq`
  payloads; script consumption checks exchange args (names and
  values, both directions) before producing the outcome.
- `catalog.go`: kinds flow from complete expectations and exchange
  outcomes; `errors.json` content is unchanged (same kinds).
- `code.go`: `CAN3109` (row shape), `CAN3204` (bare kind);
  `CAN4003` deleted with its rule. Both row-shape codes join the
  prove-first gate: malformed evidence blocks execution, so each
  mistake yields exactly one diagnostic.
- Tests: payload-mismatch and bare-kind tests; arg-mismatch,
  arg-name, and row-shape tests; the audit's wrong-user and
  wrong-payload mutations now fail with named diffs. Gallery gains
  `incomplete-expectation.can` and `missing-exchange.can` (one
  fault each); `stale-emits.can` becomes `unknown-emits.can`.
- Grammar: `exchange`, `args`, `outcome` are control keywords with
  gramcheck samples.

## Consequences

- Breaks (stated plainly): bare error expectations, outcome-only
  script rows, and unrealized-emits reliance all fail now. Every
  shipped test row was rewritten with identical outcomes; `auth.ts`,
  `retry.ts`, `errors.json`, and `normalize` output are
  byte-identical — evidence changed, programs did not.
- During migration the new checks caught two real pre-existing
  facts about the sketches: `auth__check_pw` names its second
  param `hash` (callers pass `pw_hash` positionally — now pinned
  per test), and unreachable-test rows must still be well-formed
  exchanges or explicit `-`.
- Open question 4 (multi-call scripts) is answered for the
  single-site case: ordering comes from per-site lists, identity
  from test keys, arguments from exchanges. Cross-site causal
  traces stay future work.

## Open decisions (do not block)

- Module-level `emits` stays as-is (question 7 untouched).
- Whether `Ok` expectations should require field-complete records
  already holds by construction (records are total) — no change.
- Silent-`Ok` and bare-`Ok()` forms are unchanged: `Ok` was never
  kind-only.
