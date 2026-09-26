# P06 syntax and authoring-policy questions (FINAL PACKET)

Status: evidence-complete (P03 research + P05 Jev done; anchors verified).
Sent to the user 2026-09-26; answers recorded in `p06-answers.md`.
Each question carries complete examples, consequences, and scope. No Jev
consultation decides these; Jev packets cover only technical aspects.

## Direction questions (affect scope; ask early)

### S1 Target application classes

Which of these are in scope for the next implementation round?

- (a) Controlled Bun backend / server-rendered tenant CRUD (review: credible pilot now)
- (b) Can-authored browser CRUD like invoice-grid (review: credible evaluation; validate controls/lifetime/reuse first)
- (c) Rich client with specialist widgets, browser SDKs, offline behavior (review: establish host/UI path first)
- (d) Integration-heavy service with tenant webhooks + durable workers (review: prove policy/worker/deadlines/observation or assign to companions)
- (e) Production object-upload workflow (review: resolve S3 contract first)

Consequence: (c)–(e) pull in R01/R11/R15 design work; (a)–(b) keep the round to
R02/R03/R06/R07/R08/library scope plus defect repairs.

### S2 Supported platforms

Confirm: browsers (Chromium + WebKit pinned?), databases (SQLite + PostgreSQL;
MySQL?), object storage (which S3-compatible service for qualification?), Linux
packaging (Debian 13+ amd64 on the x86 machine). What explicitly stays out?

### S3 Final ambition vs milestones

Is the target "normal SaaS choice" in one round, or a bounded pilot + named
follow-ups? Proposed milestone order from the review: (1) S3 contracts +
request budgets + failure reporting; (2) second app with shared libraries +
native control semantics + one external integration; (3) library-authoring
exercise (owner setup, result-data composition, iteration); (4) operate pair +
worker on Linux with faults + rolling deployment. Accept / reorder / cut?

### S4 Roadmap standing

The review's SaaS recommendation is one input, not the whole roadmap. Do
AI-native judgment features, editor tooling, and Linux operations keep their
standing, or does SaaS breadth reprioritize them?

## Q1 Boolean match arm order (F-R08-01; conflicts with user decision 2026-09-23)

Current rule (enforced, `completion_matches.go`: "ordinary Boolean match
requires false before true"):

```text
match member
    false => 100
    true => 80
```

Rejected today:

```text
match member
    true => 80
    false => 100
```

Review proposal: allow both orders; leave canonical spelling to
formatting/lint. Scope if accepted: ordinary single-scrutinee Boolean data
matches only — completion/coordination matches, multi-scrutinee ordering, and
native AI true/false criteria unchanged. Applies to all existing sources
(no compatibility obligation; formatter would normalize).

Options: (a) retain enforced `false`-first; (b) allow both, formatter-canonicalize;
(c) allow both with no canonicalization.

## Q2 Meaningful final locals (F-R08-02; touches C8/DI-23 validity rule)

Current rule (enforced, narrow C8 four-predicate check in `locals.go`):
a single-use nonfaulting local immediately returned unchanged is an error:

```text
subtotal = price * quantity
ok subtotal
```

→ `unnecessary local subtotal ...; replace with ok price * quantity`

Review proposal: allow meaningful domain locals like `subtotal`; leave
canonical spelling to formatting/lint. Scope: the C8 predicate (last binding
+ terminal forwards it + single use + simple syntax). Arbitrary multi-use or
faulting locals are already unaffected.

Options: (a) retain the validity error; (b) demote to advisory lint
(canonicalize or not?); (c) remove the check entirely.

Consequence of (b)/(c): mandatory-assertion repair cost may drop for
explanatory locals; formatter/lint must define the canonical style.

## Q3 `near` capture binding (F-R08-03; touches LD38/LD39 + DI-08 retained rule)

Current rule: `near` inputs bind by the callee's parameter name, looked up in
the caller's scope at `callable` creation (`callables.go`):

```text
fn int multiply
    given
        near int factor
        int number
    ...
f = callable multiply   // captures caller's `factor` binding
```

Renaming the library parameter `factor` → `rate` changes what the caller's
`callable multiply` captures (or breaks it), even though the callable type is
unchanged. Same-type shadowing can silently change the captured value.

Review proposal: explicit binding at the capture site + safe rename support.
Illustrative direction only (exact syntax needs a technical packet first):

```text
f = callable multiply with factor = my_factor
```

Questions: (a) is explicit binding wanted at all, or are explicit context
records (current recommended workaround) sufficient? (b) if wanted, which
surface: site bindings, renamed-capture diagnostics, editor rename support,
or a combination? Scope: captures only; direct-call argument supply is
positional and unaffected.

## Q4 Assertion setup for owner values (F-R07-01; touches LD29 closed gate)

Current rule: assertion rows are one-line input ⇒ expectation; inputs cannot
handle domain failure, so a legitimate fallible smart constructor cannot be
invoked in an assertion argument. Valid today: a private no-domain-error
fixture helper that calls the constructor and fails via standard fault on
unexpected rejection — at the cost of an extra function + assertion obligation.

```text
// wanted but rejected: fallible construction inside the row input
// available today: private fixture helper + standard-fault-on-reject
```

Review proposal: compare reusable test factories with checked test-local
setup before weakening mandatory examples or permitting forged values.
Technical comparison runs first (P04/P05); the user decides only if a syntax
change (e.g. a setup region) is proposed, since LD29 explicitly closed the
gate with "no new assertion grammar."

Question (asked after the technical packet): if factories prove too costly,
is a checked setup region acceptable in principle, and what must it never do
(forge owner values, skip verification)?

## Q5 Finite error-set parameters (F-R06-01; touches DI-03/LD14 deferral)

Current rule: authored `emits` enumerates concrete error types; only native
collection helpers preserve callback-specific failure bounds. A reusable
retry/trace helper over two unrelated callback error sets must enumerate a
fixed family or translate failures into result data.

Review proposal: make result-data concise first; evaluate an explicit finite
error-set parameter only if adapters remain extensive. No broad effect system
is established as necessary.

Technical comparison runs first (P04/P05). The user decides only the surface
question if a parameter form is proposed: is a new generic-kind/error-parameter
syntax acceptable in principle for the AI-agent audience, or must reusable
failure composition stay within result data + fixed bounds?

## Q6 Iteration surface (F-R05-02; touches LD21 deferral, relay contract)

Current rule: no tail-call promise; `relay` forwards completion without stack
guarantees; native folds cover existing arrays; sync tail recursion overflows
(20k steps on Bun 1.4.2).

Review proposal: compare a small immutable-state iteration primitive vs proven
self-tail lowering to native loops; a new keyword alone is insufficient evidence.

Technical comparison runs first (P04/P05). The user decides only if new syntax
is proposed: is a new iteration form acceptable in principle, and must it
preserve evaluation order, failure identity, fixture paths, and diagnostic cost
(all currently required)?

## Q7 Captured HTML routes (F-R03-01)

Current rule: plain routes reject captures (`:segment` fails static validation);
GET actions require JSON; captured HTML read pages use query URLs.

Review proposal: reuse existing capture/URL machinery for server-rendered reads.
This is a bounded extension, independent of a SPA router.

Question: accept captured server-rendered read routes in principle? Exact
grammar/checking rules come from the technical packet; no SPA router is implied.

## Q8 Editor workflow priority (F-R09-01)

Current: LSP advertises sync + definition only; formatter machinery exists.
Review acceptance: extract/rename callback, change shared record, inspect
contract, repair callers with normal editor tools.

Question: which editor capabilities are required in the next round
(completion / hover / references / rename / formatting), in what priority
order? Rename interacts with Q3.
