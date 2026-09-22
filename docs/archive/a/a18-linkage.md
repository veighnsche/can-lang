# a18 — script/outcome consistency (issue #1)

Scripts are claims about providers. Until now nothing checked the
claim: a `given` row could script any outcome the emits allowed, and a
wrong Ok value passed the build. a18 proves the claim when it can be
proven: for a foreign can call, every scripted **Ok** outcome is
evaluated against the provider body on the script's own exchange args,
and a mismatch fails the build with **CAN3110**:

```
script lie contradicts lib__double:
  provider computes Ok(value = 8), script says Ok(value = 999)
```

## The rule

For each foreign-can call site, for each exchange row with an Ok
outcome: bind the row's args to the callee params, evaluate the callee
body in a sandbox (fresh store, no scripts, depth bound), and compare
with `vEq` — the same equality the test runner uses, so the check and
reality cannot disagree.

## What stays trusted

- **Scripted errors.** `emits` is an upper bound that admits
  unrealized entries (a12), and failure injection is what scripts are
  for: `db.down()` for `u_01` is the flagship's flaky test, not a lie.
- **Externs.** No body, nothing to evaluate.
- **Anything the sandbox cannot model.** A provider needing its own
  scripts, unreachable state, depth exhaustion, or values `vEq`
  cannot compare is trusted, never failed. A proof attempt that
  cannot run proves nothing; panics inside the evaluator recover to
  trust. A check must never fail a build it cannot model.

CAN3110 joins the prove-first gate: a contradicted script blocks test
execution like any other unproven claim.

## The flagship gets honest

The check caught four fictional rows in `auth.can`: `failed_attempts
= 5` and `pw_hash = "wrong"` for `u_01`, which `db__get_user` never
computes. The fix stages failure against real provider state instead:

- `db.can` gains a second user, `u_02`, whose row really carries five
  failed attempts. The lockout tests now log in as `u_02`.
- The wrong-password tests now supply a genuinely wrong password
  (`seal Auth__Password("wrong")`) against the real `u_01` row; the
  mismatch moves to where it belongs, the `check_pw` script.
- `auth.ts` and `errors.json` are byte-identical: no bodies, no
  errors, and no test names changed. Only `db.ts` gains the new arm.

Revs are untouched: the interface (params, return type, emits) is
unchanged. Whether decision-table growth alone should bump a rev is
open.

## Open

- Error *payloads* are unverified: `user_not_found(id = "WRONG")`
  passes. Only the kind is checked (existing StubNotInEmit rule).
  Verifying payloads of provider-yielded error kinds is possible
  future work; unyielded kinds must stay trusted or failure injection
  dies.
- Exchange args must be ground for the check to run; a row that
  references caller locals is trusted (and still enforced at runtime
  by the arg-equality proof).
