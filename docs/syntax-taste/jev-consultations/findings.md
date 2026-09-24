# Jev consultations on the implementation task list

24 September 2026. Advisory evidence for the [implementation task list](../can-implementation-task-list-2026-09-24.md)
(T01–T27), which is a plan, not a claim that any production change has shipped. Three fresh live requests to
`jev-latest` (all answered by `jev-1.13.0`) cover the three hardest cross-cutting implementation decisions below.
Every explanatory state field, instruction and option description was rewritten in full across the three rounds
(48 fields; 16 per request); task IDs, gate names, `Bun 1.4.2`, `Debian 13 amd64/glibc`,
`canlc build --target browser`, `operation_id`, status codes, byte/row caps, DI identifiers, probe counts and code
spellings stayed exact. No request contains any prior Jev answer. The [TypeSafe skill](../../../.agents/skills/typesafe-ai/SKILL.md)
was followed, including a [fresh read of the live index, API, Choice, state and confidence documentation](source-refresh.json).
Jev performed no research and returned typed choices, distributions and confidence values, not prose reasons.

## Why these three decisions

T03/T05 owner, package and error semantics already carry dedicated triple consultations in `preparation/`. The
remaining hardest judgments in the task graph are the three Astra-class integration points whose failure modes span
lanes and gates: (1) **T09 core integration** — how T02–T08 converge without blind merges or lost cross-feature
semantics while invoice/browser lanes keep moving; (2) **T14 replay/uncertainty** — the atomicity, disclosure and
retry contract that decides whether a lost acknowledgement can duplicate an invoice effect or leak tenant data;
(3) **T21/T22 browser closure** — the compile-time boundary that decides whether server secrets and capabilities can
reach the browser bundle. Each has at least two superficially cheaper alternatives (deferred merging, split
transactions, runtime guards), which is what makes the choice worth an outside judgment.

## Exact evidence

| Round | Submitted request | Saved response |
| --- | --- | --- |
| 1 | [request 1](request-1.json) | [response 1](response-1.json) |
| 2 | [request 2](request-2.json) | [response 2](response-2.json) |
| 3 | [request 3](request-3.json) | [response 3](response-3.json) |

[Preflight](preflight.json) records the approved request hashes (verified unchanged at send time), mechanical
uniqueness results (no identical corresponding field, no shared 8+-word sentence, pairwise 3-gram similarity
0.016–0.025; the only shared 8-grams are the exact `bun run` command sequences required verbatim from AGENTS.md),
and manual equivalence review. The [wording audit](wording-audit.json) lists the preserved fact sets, and
[consult.py](consult.py) reproduces the requests byte-for-byte and sent exactly three API calls with no retry.
Reported usage totals **7,602 input and 492 output tokens**. Credentials were supplied through the authorization
header and are absent from saved evidence.

## Results

Selected probability is the model's probability for that candidate, not the probability that the implementation
plan is correct. Confidence is a separate provider value. Order is round 1 / 2 / 3.

| Decision | Selected in every round | Probability | Confidence |
| --- | --- | --- | --- |
| Core integration (T09) | `single_locked_integration` | 1.00 / 1.00 / 1.00 | 1.00 / 1.00 / 1.00 |
| Replay/uncertainty (T14) | `same_transaction_ledger` | 1.00 / 1.00 / 1.00 | 1.00 / 1.00 / 1.00 |
| Browser closure (T21/T22) | `strict_transitive_closure` | 1.00 / 1.00 / 1.00 | 1.00 / 1.00 / 1.00 |

## Disagreement investigation

There is no categorical disagreement: all nine judgments are unanimous at 1.00. That unanimity is investigated
rather than celebrated. The supplied constraints overdetermine the outcome: the task list's own acceptance rules
(single locked revision, no second business effect, no server capability in the browser bundle) are stated in the
shared state, and each rejected alternative explicitly concedes the corresponding property (opportunistic merging,
a ledger-absent window, runtime-only guards). The consultation therefore functions mainly as a coherence check —
the task list's selections are the only options fully consistent with its stated contracts — not as independent
evidence that those contracts are right. Full-contract options are also necessarily longer than their alternatives
because they carry byte caps, status codes and seam lists; one instruction explicitly directs judgment from
contract reliability rather than prose length, but length/detail asymmetry cannot be ruled out as an influence.
Fixed option order, correlated questions, one shared model and one evidence base make these nine correlated
advisory votes. No calibrated acceptance threshold is claimed, and a 1.00 reported probability proves nothing
about implementation correctness.

## Engineering judgments (not Jev output)

The retained positions below follow from the failure-mode arguments, not from the vote count. Each matches the
corresponding task-list selection; the consultation changed none of them.

1. **Core integration: keep T09 single-locked integration with lanes on stated prerequisites.** Blind-merging
   compiler/catalogue edits from parallel workers risks silently dropping checker, package-identity or diagnostic
   semantics that per-lane suites cannot see; the four named cross-feature negatives (owner record across a
   generic/package seam, variant leaf across imports, fixture link after alias rename, error report after lineage
   change) exist precisely because those seams are invisible to single-feature tests. One regeneration from the
   integrated source keeps goldens attributable to one revision. Blocking all lanes on T09 would idle T18/T19
   shutdown and platform work whose prerequisites the graph already satisfies, so lanes continue on their named
   dependencies while T16 waits for the integrated core.

2. **Replay/uncertainty: keep the same-transaction ledger with identical-ID replay and shared 403 denial.** A
   split ledger leaves a crash window where the invoice row changed but no replay key exists, converting a
   recoverable ambiguity into an undetectable duplicate on retry. Minting a fresh `operation_id` per retry makes
   duplication certain rather than possible. Same-transaction atomicity plus same-digest replay / different-digest
   conflict is the minimal contract that lets the grid distinguish "my write committed" from "another writer
   moved the revision", which a snapshot reread alone cannot settle. Collapsing foreign and nonexistent targets
   into one post-authentication 403 removes the existence oracle without complicating the adapter case tables
   (five HTML, five JSON save cases, three GET load cases, 400 before protected entry).

3. **Browser closure: keep the distinct browser root/profile with transitive compile-time capability closure.**
   Runtime guards ship server-capable code into the client bundle and convert a build-time rejection into a
   hope that every call site is guarded, including paths reached through generic specialization and callable
   references that are easy to miss in review. A shared profile also invites accidental inclusion of SQL handles,
   process access or secret bytes in emitted output. The transitive closure plus emitted-graph/sourcemap audit
   makes forbidden reachability a compiler diagnostic instead of a runtime incident, and the single
   compiler-produced content-addressed same-origin asset keeps the deployed script class reviewable. Authoring
   convenience and worker execution are explicitly out of scope for the first grid.

## Sources, limits, remaining work

Task and contract facts come from the [task list](../can-implementation-task-list-2026-09-24.md), the
[integrated action contract](../preparation/integrated-action-contract.md), the
[browser target design](../preparation/browser-target-design.md), the
[product acceptance matrix](../preparation/product-acceptance-matrix.md) and the
[evaluation protocol](../preparation/evaluation-protocol.md). Probe counts (11 owner roots, 13 nested-pattern
cases, 16-case variant matrix, 28/28 helper assertions, 12 FormData checks, 82 route assertions, HTMX 503
observation, 18-assertion invoice sketch) are bounded disposable spikes, restated from those packets and not
rerun here.

These calls ran no Can code, exercised no live database, HTTP exchange, browser grid or Linux artifact, started
no held-out agent comparison, measured no token effect, and established no calibration. Parser/checker, lowering,
ledger, dispatch, browser-target and shutdown verification remain T09–T27 implementation obligations. The limits
stand even with unanimous 1.00 advice.
