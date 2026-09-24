# DI-06 private-effect fixture comparison

24 September 2026. This is a hand-authored current-Can mechanism probe at revision `02a549d28fd5fc5c3996160e65a97de332390d30`. The [case registration](probes/fixture-hard-case/case-registration.md) was written before execution. The [runner](probes/fixture-hard-case/run.py), [main reports](probes/fixture-hard-case/results.json) and [mutation controls](probes/fixture-hard-case/controls.json) preserve the exact programs and outputs. The runner uses a Go overlay of the existing compiler test entry in `/private/tmp`; no production source, compiler, runtime or canonical specification was changed.

The helper's public operation is `stamp(int)`. Its private `sample_time()` calls `clock::wall_millis()`; `stamp` turns the time into text and appends `"!"`. Two ordinary app functions, `preview` and `receipt`, call `stamp`, and a third app function carries the assertion expecting `"1000!"`. This makes the time dependency an internal library detail in the starting API. An assertion-only callable argument would therefore either alter that functional signature or require another exported entry.

## Executed results

All eight main projects checked and emitted. They ran 58 attached roots: one expected caller failure in the renamed ambient case and 57 passes. The six preregistered mutation projects also checked and emitted. The decisive caller-root results were:

| Current-Can form | `customer` caller | Caller-only rename to `renamed` | Mutation control at `renamed` | Caller-path evidence |
| --- | --- | --- | --- | --- |
| Helper lexical `customer` clock row | Pass | **Fail**: actual clock time, not 1000; no checker diagnostic | Changing the helper selector to `renamed` repairs the caller, adding a helper-file edit | Pass: `real-can`, `supplied-completion`; failure: `real-can` |
| Caller `when` row on complete `helper::stamp(7)` | Pass | Pass, after renaming the caller-local selector | Changing the helper suffix from `!` to `?` leaves the caller passing | `real-can`, `supplied-completion` |
| Public `stamp(int, callable now)` | Pass | Pass | Changing only fake time from 1000 to 2000 fails; changing only helper suffix from `!` to `?` also fails | `real-can` in all three, with outcome mismatch for each mutation |
| Public `stamp(int)` plus exported `stamp_with_clock(int, callable now)` | Pass | Pass | The same separate fake-time and suffix changes each fail | `real-can` in all three, with outcome mismatch for each mutation |

The ordinary callable is authored Can code, not a fixture row. Both injected forms really invoke the helper's conversion and suffix; the two independent mutations establish that the caller path depends on the controlled time and subsequent helper logic. The stub deliberately supplies the entire result, so its green result after the suffix mutation cannot count as helper-body coverage. The ambient row still demonstrates the original ownership defect: an unrelated caller root name activates a helper-owned fixture without an explicit link.

## API and authored-edit cost

| Form | Helper / app source lines in this fixed pilot | Helper exports | App calls of helper operation | Contract/edit impact relative to starting form |
| --- | ---: | --- | ---: | --- |
| Ambient baseline | 22 / 29 | `stamp` | 3 | `stamp(int)` keeps the clock private; changing the caller assertion label alone breaks its implicit fixture link. Repair needs a second edit in the helper. |
| Whole-helper stub | 21 / 32 | `stamp` | 3 | The assertion gains a local `when` row and its selector moves with the renamed root; normal callers and the helper API stay put. The helper body is absent from this caller path. |
| Public parameter | 18 / 43 | `stamp` | 3 | `stamp` gains `callable int () emits [] now`; all **three** app call expressions change. App imports `clock`, defines a real-clock adapter for the two ordinary calls, and defines a fake for the assertion. The helper no longer owns its time read. |
| Separate test entry | 34 / 34 | `stamp`, `stamp_with_clock` | 3 | The two ordinary `stamp(int)` calls stay unchanged; the assertion changes **one** call to `stamp_with_clock(7, callable fake_now)`. The helper adds one export and a shared callable-taking implementation, publishing the otherwise private clock dependency as test-facing API. |

The line counts and call-site counts describe these authored samples, not task-wide agent cost. The separate entry is a serious current-language alternative: it keeps production calls stable and gives honest integration coverage. It still makes a private implementation dependency part of the package's exported contract and gives agents two ways to enter the same behavior. Nothing observed proves that cost is worse than a new scenario language feature.

## Smallest proposed helper-owned scenario

The proposed source form is a **contract sketch**, not executable Can syntax. The helper would export a named `fixed_time` scenario for `stamp`; only the helper can declare a row for its private `sample_time` → `clock::wall_millis()` call, returning `ok 1000` for `()`. The app assertion would attach `helper::fixed_time` to its particular `helper::stamp(7)` invocation. The link is between the qualified callee declaration, the helper-owned scenario, and the caller's exact call site; the caller's `customer` or `renamed` display label has no selection role. The invoked `stamp` body still converts the supplied time and appends `"!"`.

The minimum static checks are: exported scenario visibility; exact owner and callee identity; unique referenced call site; internal target and empty argument tuple; `int` success completion and declared error bounds; and rejection of missing, inaccessible, ambiguous, duplicate, unused or stale selected links. A selected link must never silently fall through to a real clock read. At runtime, a scenario plan would travel with that invocation and reserve its internal row in the existing root/table/invocation FIFO context. Reports would mark the internal clock completion `supplied-completion` and the helper-body operations `real-can`.

The [disposable prototype](probes/fixture-hard-case/scenario-prototype.py) exercises the central runtime promise over actual generated Can code without changing repository compiler/runtime sources. A sidecar `scenario-link.json` names the qualified owner, callee, private clock target, row and exact caller call site. The prototype validator checks those fields against the prepared source and generated call sites; a postprocessor patches only the generated app call and a copied `/private/tmp` runtime. The copied context passes a selected scenario through nested helper calls. Its fixture selector first restricts ordinary lexical rows to the root's package, then permits the helper-owned `fixed_time` row at the exact linked target. A selected invocation consumes the current FIFO table and records supplied-completion evidence.

| Disposable F2 check | Observation |
| --- | --- |
| One linked `stamp(7)` under caller root `renamed` | All seven roots pass; the caller reports `real-can` and `supplied-completion`. The two ordinary `stamp(int)` app call sites and helper's functional signature are unchanged. |
| Helper suffix changed to `?` | The linked caller fails, as does the helper's local unit assertion; the caller still reports the supplied internal clock completion. |
| Two linked `stamp(7)` calls in one caller root | All seven roots pass when the scenario plan declares two occurrences. The existing FIFO allocates each internal clock row in order along separate invocation paths. |
| Six malformed sidecar links | Changed scenario name, target, callee, caller site, success type or argument tuple each raise a prototype validation error before execution. |

The prototype uses an external manifest and generated-output rewriting rather than a Can declaration and real compiler checker. Its six negative validations compare against hard-coded known fields; they do not prove general type/visibility diagnostics. For two occurrences, the spike duplicates the one prepared row in the selected runtime plan, which illustrates the existing queue behavior but leaves the final scenario occurrence syntax and checked count rule undecided. Concurrent or recursive invocation, nested exported scenarios, changed owner signatures, unused links, private metadata leakage and whole-task agent cost remain untested. The patched context marker scopes one selected scenario at a time and is not a concurrency design.

## Decision and limits

The evidence warrants **owner-scoped lexical fixture selection** and accurate stub provenance as definite implementation tasks. Foreign assertion display names should no longer activate helper rows. The disposable F2 spike makes the helper-owned scenario route technically plausible for this private-clock case: it preserves `stamp(int)` and normal call sites, survives the caller rename, reaches the real helper body, and uses the existing FIFO for two sequential links. It does not establish that adding scenario grammar, checker, emitter and runtime contracts is preferable to the fully executed `stamp_with_clock` idiom. A final design choice should weigh the private test API against that new language surface, then require nested/recursive/concurrent and link-drift conformance in implementation. F3 seams are only needed if a scenario cannot express a concrete target. Creation, held-out refactor and repair trials must measure complete successful-agent-task tokens before claiming efficiency.

Three fresh [Jev consultations](probes/fixture-hard-case/jev/findings.md) on the current-Can hard case preferred a bounded scenario prototype in two requests and the exported test entry in one. They were sent **before** the disposable F2 spike, so their distributions do not incorporate its result. The disagreement reflects a real tradeoff between a demonstrated current-language path and a new language feature. Jev is advice, not evidence that either mechanism wins.

The current-Can pilot does not run live clock conformance or measure agent-task tokens. The generated-output prototype tests one and two sequential linked calls, but is not a compiler implementation of F1 or F2 and cannot qualify general source diagnostics, recursive/concurrent isolation, or native clock behavior. The assertion reports and sidecar checks support only the measured boundaries above.
