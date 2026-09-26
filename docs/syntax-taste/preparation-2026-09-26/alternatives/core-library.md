# P04 alternatives — core/library authoring (R05–R08)

Comparison dimensions for every option: semantic guarantees, composition,
agent authoring/refactoring, diagnostics, native lowering, lifecycle behavior,
maintenance, implementation cost. "Retain" is always an option. Candidate
syntax is marked ILLUSTRATIVE and selects nothing.

## R05 iteration and collections

### Dynamic continuation (F-R05-01/02/04/05)

- **O1 retain (bounded policy).** Keep relay forwarding + array folds;
  document that dynamically continuing work beyond the native stack is out of
  scope. Cost: docs only. Fails W4 (100k-step state machine) by construction;
  acceptable only if the user scopes W4 out (S1/S3).
- **O2 self-tail lowering.** Compiler proves `relay call` self-calls in tail
  position and lowers them to native loops, preserving observable call/test
  behavior. Guarantees: evaluation order, exactly-once arguments, failure
  identity, fixture invocation identity preserved by construction (same
  completion protocol, different schedule). Diagnostics: non-tail or
  non-self recursion keeps current behavior with a precise "not lowered"
  note. Lowering: native `while` loop — fully in the AGENTS.md direction.
  Lifecycle: no new ownership shape. Cost: checker proof + emitter loop +
  negative fixtures (mutual recursion, pending leases in frame — must not
  lower when a frame holds unowned-completing work). Risk: the proof boundary
  must be exact; an over-broad proof changes semantics silently.
- **O3 iteration primitive (ILLUSTRATIVE surface).** An explicit
  immutable-state iteration form lowering to a native loop. Guarantees: same
  as O2 plus a visible contract (state threading explicit in source).
  Authoring: new spelling for agents to learn; refactoring from relay style
  needs a rule. Diagnostics: step-indexed failures possible. Cost: grammar +
  checker + emitter + fixtures + formatter. Needs a user surface decision
  (Q6) if selected. Risk: inventing semantics O2 gets for free.

Representative program (works under O2/O3, overflows today):

```text
fn int count
    emits []
    given
        int remaining
        int total
    asserts
        done: 0, 5 => ok 5
    match remaining is 0
        false => relay call count(remaining - 1, total + 1)
        true => ok total
```

Failure case: injected fault at step 60,000 must report the step and the
  declared failure, not a native stack overflow.

Distinguishing experiment (implementation time): 100k-step state machine +
  growing aggregation + bounded worker batch under O2 lowering, measuring
  stack/memory and fault localization; only if O2's proof boundary excludes
  needed shapes does O3 earn its syntax.

### Bulk collection construction (F-R05-03)

- **O-B1 retain.** Copy-on-point-update; repeated distinct-key insertion stays
  quadratic. Cost zero; fails the growing-aggregation half of W4 at scale.
- **O-B2 native build + immutable publication.** One native `Map`/`Set`
  construction inside the adapter, published once as an immutable Can value.
  Needs defined: duplicate/collision policy, iteration order, callback-error
  behavior (for from-callback builders), ownership of the builder. Lowering:
  native construction — AGENTS.md-aligned. Cost: catalogue contract +
  adapter + fixtures. No syntax needed (ordinary calls).

## R06 generic failure composition

- **O1 retain fixed bounds.** Each helper enumerates its concrete `emits`.
  Guarantees: today's explicit finite bounds, zero new machinery. Cost:
  separate helpers per error family; adding an error to one caller can force
  unrelated edits. Fails W3 generality.
- **O2 result-data first (favored experiment).** Nominal
  `completed<item>` / `rejected<failure>` + `outcome<item,failure>` variant,
  callable returning the variant with `emits []`, adaptation at completion
  boundaries. Proven to represent and forward today (5 roots pass); unproven:
  retry count/sequence, conversion/provenance cost at scale. Composition:
  ordinary generics, no new kind. Authoring: adapter noise at boundaries;
  needs a concise convention + worked retry/trace helper. Diagnostics:
  failures become data — match arms stay explicit. Cost: library + guidance,
  no compiler change. Experiment: reusable helper over two unrelated callback
  error sets with distinguishable outcome sequences + independent oracle
  verifying attempt counts and failure-then-success ordering.
- **O3 finite error-set parameter (ILLUSTRATIVE surface).** A small explicit
  error-set parameter in callable/function contracts. Guarantees: preserves
  explicit finite bounds while abstracting over them. Composition: matches
  what native collection helpers already do. Authoring: new generic-kind
  concept for agents. Cost: checker/emitter/fixtures + C4/C5/C9 contract
  updates. Needs a user surface decision (Q5). No broad effect system is
  established as necessary — O3 must stay finite and explicit.

Decision rule: run the O2 experiment; adopt O3 only if adapters remain
extensive after a genuine concision effort.

## R07 owner values and assertion setup

- **O1 factory pattern (favored first).** Private no-domain-error fixture
  helpers that call fallible constructors and fail closed (standard fault)
  on unexpected rejection. No grammar change; owner confinement intact.
  Cost: an extra function + assertion obligation per fixture; needs a
  documented convention + worked extraction example (handler→helper with an
  owner input) measuring authoring/repair cost.
- **O2 setup region (ILLUSTRATIVE surface).** A checked assertion setup
  region running ordinary construction + completion handling. Authoring:
  removes fixture boilerplate. Risk: must never permit forged owner values
  or skipped verification. Needs a user decision (Q4); LD29's closed gate
  ("no new assertion grammar") means O2 reopens a settled choice and needs
  explicit justification.
- **O3 scenario coverage.** Selected private helpers satisfy testing
  obligations through explicit linked consumer scenarios instead of local
  rows. Reduces per-helper rows but weakens locality of evidence; needs a
  linking/verification rule.

Decision rule: work the O1 extraction example first; compare measured cost
against O2/O3 before proposing grammar.

## R08 authoring policies and captures

All three reversals are user choices (Q1–Q3); the technical contribution is
comparison-trial design so the user decides with measured rather than
intuited costs:

- **Boolean order:** trial = agent repair tasks on `true`-first rejections
  vs formatter-canonicalized `false`-first sources; measure attempts + tokens
  to green. Retain keeps the enforced rule; relax moves canonicalization to
  format/lint with identical branch selection either way.
- **Final locals:** trial = meaningful-domain-local edits vs accidental-alias
  edits under (a) current error, (b) advisory lint; measure repair cost and
  false-positive rate. The C8 predicate's four clauses bound the trial.
- **Near binding:** trial = registered same-type rename/shadow repair tasks
  under (a) current name-based capture + alias locals/context records, (b)
  illustrative explicit site bindings; measure breakage + repair cost. Safe
  editor rename (R09) is the companion investment either way.

No option becomes accepted merely because a trial exists; trials inform Q1–Q3.
