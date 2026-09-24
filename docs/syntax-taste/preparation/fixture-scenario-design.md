# DI-06: bounded exported helper scenarios

24 September 2026. Planned source and runtime contract, not current Can
behavior. The [hard private-effect comparison](fixture-hard-case.md) ran both
ordinary injection idioms and a disposable exported-scenario spike. Three
fresh [post-spike Jev consultations](jev-fixture-post-spike/findings.md)
favored the scenario while retaining substantial probability on a separate
test entry in one wording. Engineering selects the scenario for the planned
design because the caller can deliberately exercise the real `stamp(int)` body
without publishing its private clock dependency. Agent-task cost remains to
be measured against the executed `stamp_with_clock` baseline.

## Source surface and static linkage

The helper package exports a scenario symbol beside its ordinary function:

```can
package helper
    provides [stamp, fixed_time]
    uses [clock, text]

scenario fixed_time for stamp
    path [sample_time, clock::wall_millis]
    cases
        => ok 1000
```

This is **new proposed syntax**. `scenario` is a top-level contextual
declaration. `for stamp` resolves one exact executable function in the
declaring package, including concrete type arguments when the callee is
generic. `path` lists an exact static call chain from that function to the
supplied target. Each step must resolve to exactly one lexical invocation
within the preceding function; zero or multiple matching sites diagnose at
the scenario declaration with the candidate source spans. A recursive cycle
in this declared path is rejected in this bounded version. The target's
`cases` reuse the existing inert fixture row input/result grammar, type and
opaque-value restrictions. Rows for repeated dynamic occurrences are written
in FIFO order; an empty-argument target uses the existing empty case input
form as above. The target may be a supported catalogue/native operation or
ordinary function subject to the same exact completion and fixture-mode
checks as current lexical rows. The scenario is erased from production output.

Only the helper owner may name its private `sample_time` path in the scenario.
An importer sees the exported scenario symbol and its `for stamp` target, not
the private path or row internals. `provides` controls scenario visibility;
re-export does not let a foreign package author new private-path rows. The
scenario and every referenced source/raw fixture byte enter the dependency
lock. A renamed/missing target, changed argument/result/error contract,
ambiguous static step or inaccessible scenario is a build error. The compiler
does not silently select real execution when a requested link breaks.

A caller attaches that exported plan at one lexical helper invocation inside
its existing `when` table:

```can
match call helper::stamp(7)
    when
        sample: 7 using scenario helper::fixed_time
    ok str value => ok value
```

`using scenario` is a new contextual row form, distinct from an ordinary
`sample: 7 => ok ...` supplied completion and from `sample: use template(...)`
which expands a fixture for the call itself. The row's `7` follows the
callee's exact argument grammar and is checked against that invocation's
actual values. It names one root selector and one lexical call site. The
scenario's declared `for` callee must be the call's resolved declaration and
concrete specialization; matching signatures alone do not suffice. At one
site/selector, a full-callee supplied completion and a scenario link cannot
both govern the same invocation. Each expected dynamic occurrence needs its
own link row; missing, duplicate, ambiguous and unused links are failures.
Display-label equality in a foreign package never activates helper rows.

Existing lexical `when` rows and `fixture` templates remain same-owner:
only roots declared in the same canonical package instance may select them.
The helper can still test its own private clock without exporting a scenario.
The new exported symbol is the only mechanism for cross-package deliberate
activation of helper-owned internal behavior. A whole-helper stub remains
valid caller-unit evidence but cannot claim real helper-body coverage.

## Invocation and evidence semantics

At the selected caller invocation, carry an immutable scenario plan through
the declared static call chain. Only that dynamic invocation and its
descendants may reserve the scenario's rows. The existing root/table/invocation
FIFO key remains; each parallel participant and recursive helper invocation
receives its own dynamic ancestry, so equal display labels and equal arguments
in other roots cannot consume the selected rows. A scenario path may not be
recursive in this first version, but a selected helper may invoke unrelated
recursive work; its ownership and queue keys remain isolated. A later
recursive-path feature requires a separate explicit depth/occurrence contract.
Unexpected extra target calls, missing target calls and unused rows fail the
selected assertion. No search-ahead matches later expected arguments. The
scenario plan is not process-global state and never survives the root run.

The supplied internal target completion is labeled `supplied-completion`;
ordinary helper statements before and after it are labeled `real-can`.
`raw-provider-fixture` is used only if a raw provider adapter actually runs.
No result here proves live clock quality, SQL/HTTP behavior or browser output.
An assertion with a selected scenario cannot be counted as real execution of
the supplied target. A valid link to a scenario whose cases are exhausted is
an assertion failure, not a fall-through to native I/O.

Generated TypeScript uses normal native calls for the helper body and a small
assertion-only context adapter to carry the selected plan and reserve typed
rows. No custom interpreter or production runtime override registry is
introduced. Production output excludes the scenario and the assertion adapter
where assertions are not being executed. Lock and report identities use the
canonical package instance from [DI-05](package-error-contract.md), not a
short package name or caller display label.

## Production acceptance

1. Build the private-clock `stamp(int)` case unchanged for normal callers.
   Link `fixed_time` at the caller's exact `stamp(7)` site; rename only the
   caller root from `customer` to `renamed` and retain the supplied clock plus
   real suffix. Mutating clock result or helper suffix must fail the caller.
2. Reject a deleted, private, wrong-callee, wrong-site, wrong-argument,
   wrong-completion, stale-target or ambiguous path link before execution.
   Reject duplicate and unused required links. A foreign coincidentally named
   root cannot activate a helper-local lexical row.
3. Exercise two sequential and two concurrent links with equal arguments,
   nested helper calls and independent roots with duplicate display labels.
   Check exact FIFO reservation per root/table/invocation/participant. A
   recursive path declaration is rejected with a specific diagnostic; unrelated
   recursion within the helper does not leak selection.
4. Confirm supplied/real/raw evidence labels and no native clock call at the
   selected target. A whole-helper stub's green result after suffix mutation
   remains labeled as lacking helper-body coverage.
5. Compare creation, caller-label rename, helper-target rename and diagnostic
   repair against `stamp_with_clock` with registered held-out variants and the
   same tools/models. Semantic correctness leads; count total tokens per
   successful task and wall time second. The new mechanism is not credited
   with measured agent benefit from the disposable spike.

The [disposable prototype](fixture-hard-case.md#smallest-proposed-helper-owned-scenario)
passed one and two sequential links with a hard-coded sidecar validator and
temporary generated-output patch. It did **not** implement or verify the Can
source forms above, general static checks, concurrent isolation or recursive
behavior. Those are implementation acceptance obligations, not established
facts.
