# Native Can testing: design reconciliation and review

Status: **design revised, independently reviewed and consulted with Jev; ready
for dependency-ordered implementation planning. No implementation or integrated
qualification is accepted.**

The existing full Can → TypeScript/Bun executor, reference-built Can judge and
external Go resource owner remain the architecture. The preparatory findings
support continuing with it; they do not establish a reason to build an interpreter.
The changes below close specification gaps without crediting migration coverage.
This document records the current review; earlier source inventories and raw probe
records retain their original observation scopes.

## Findings incorporated

The [canonical execution summary](preparation/native-can-test-preparatory-checks-2026-09-30/execution-summary.json)
and [results](native-can-test-preparatory-results-2026-09-30.md) distinguish three
supported mechanisms from the browser isolation failure.

| Finding | Revised contract | What remains unproved |
| --- | --- | --- |
| PM-N1 preserved one hostile getter/alias through inert handles; deliberate assimilation was detected | Raw values stay in the subject realm; async transport carries inert typed envelopes; effectful reads are named actions | Exact-number/Proxy/thenable matrix, full Can transport and final zero-access intervals |
| PM-N2 reached actual `encodeJSON` directly and detected an eager-read shim | Generated C ingress requires a version-bound independent invocation witness, local completion authentication and R-owned expectations; candidate self-report is insufficient | Ordinary C-compiled call path, emitted `createCodec` wiring and the witness's own bypass controls |
| PM-F1 demonstrated separate binary stdin and fd 3 snapshot, withheld EOF, malformed data and non-reader cleanup | Separate offered/accepted bytes, EOF, writer exit and child result; startup does not wait for readiness before closing fd 3; deadline failure remains sticky | Go `ExtraFiles`, candidate CLI sanitization, repeated launch state and fd 4 inherited lease |
| PM-B1 route handling remained live during a real pending click; serialized service deadlocked | `input_begin/settle`, route contact/delivery, application witness and terminal seals are distinct typed observations; Can owns the choreography | Full Can routing, captured-body corruption, durable replay, all required engines and driver/worker-death controls |
| Aborting a route during close could fulfill the actual click | Preserve actual settlement; do not infer delivery or issue a success-only DOM read after closing; Can checks the declared abort/settlement order | Binding-level positive and negative controls with complete error/callback observation |
| A Keychain prompt invalidated otherwise successful browser protocol/process receipts | Browser credential/host-UI isolation becomes admission and retirement evidence; known late effects append an invalidation rather than rewriting history | Qualified host observer and noninteractive launch profile; missing flags are a plausible cause, not proof |
| Resource counts were sampled and a fixed owned graph was reclaimed | Samples do not qualify strict limits, detached descendants or host-effect isolation; keep serial admission and the existing finite envelopes | Enforcing host mechanisms, independent external owner, abnormal-death recovery and receipt qualification |

The user already canceled the dialog. No further UI investigation or browser run
belongs to this review. All earlier owned execution roots/processes were removed;
the review allocates none.

## Coverage preservation

The [292-row migration ledger](native-can-tests-migration-ledger-2026-09-30.md)
remains authoritative for retained facets, dynamic variants, named fixtures,
environments, delegated oracles and external consumers. The current tracked
`tests/` inventory still has the same 224 files and hashes as its captured
snapshot. This is a drift check, not evidence that the old suite passes.

The ledger now maps each PM result to its relevant consumers and excluded
claims. No row is migrated or deletable. Browser rows additionally require
qualified native host-effect isolation; a passing route exchange cannot replace
that gate. Required unavailable engines/services remain blocked, and historical
experiments/static data retain their explicit classifications. Deferred resource
measurements remain obligations rather than disappearing into a quick profile.

Before deletion, every retained row/facet/variant/environment needs ordinary Can
case/check IDs, valid execution evidence, relevant seeded defects and missing-
evidence controls, matching cleanup receipts, and resolved current callers.
Counts alone cannot prove parity. New DB sentinels and stronger cleanup controls
remain explicitly stronger qualification conditions, not retroactive claims
about what the old host harness observed.

## Prevent a hidden host harness

The archived preparatory Python/TypeScript sources contain fixed expectations
because they are temporary research instruments. They cannot become the final
runner, a generic `run_probe` endpoint, an active CI check or a source string
embedded in Can. Preserve their compact evidence; port their test policy to Can.

| Responsibility | Required owner |
| --- | --- |
| Cases, fixtures, expected bytes/counters, timeout meaning, retries, selection and result reduction | Ordinary Can functions compiled by R |
| Raw value construction, same-realm identity/counters, native invocation and browser input/route mechanics | Reviewed typed bindings; no expected answer or whole-test operation |
| Deadlines, byte framing, reservations, process/descriptor/resource authority and release facts | N and its owned services, outside killable case workers |
| Candidate generated execution and production adapter behavior | Isolated C subjects; their output is observation, not authority |

Every new binding must document its native operation, typed arguments, observable
facts, synchronous/async behavior, effects and ownership. Reject fixture branches,
native scenario scheduling/retries, executable strings and case-specific expected
outcomes. Finite immediate DOM batches may contain actions, not control flow.
This is a review gate on implementation; no catalogue is implemented by this text.

AI agents remain the intended authors. Ordinary helpers are unrestricted within
normal Can target/effect rules. Explicit action/check identities, failure kinds,
fixture consumption and ownership make generation and repair predictable.
The [authoring audit](preparation/native-can-test-authoring-2026-09-30/agent-authorship/audit.md)
still gates concrete packages/examples; current exploratory snippets are not a
claim that the final API has already been optimized or compiled.

## Trust and lifecycle boundaries

R, N, suite source/artifact and C have separate identities and acceptance records.
There is still no accepted R bundle or qualified N executable in the inventoried
inputs. The candidate never compiles its authoritative judge, enters its realm
or promotes itself. Transitional host bootstrap witnesses may qualify the first
seed/owner only through separately reviewed bounded controls.

N2 now requires an independently reviewed witness at the adapter boundary reached
by the actual generated subject, not a candidate `invoked` flag or a matching
error shape. Its artifact/export/signature and hook provenance are fixed before
execution. A bypass returning plausible output must fail; wrong exports, eager
reads and forged completions have separate controls. The exact hook is early
implementation/qualification work; if it cannot meet the contract, reconsider
that bridge before dependent native migration. A direct source call does not
close this question.

Browser admission binds the full launch configuration and test-only credential
policy to an independent host-effect observation scope through disposal.
Headless mode, driver flags and temporary HOME are not proof. The preferred
implementation candidate uses the pinned driver's launcher inside an N-owned
service; externally registered ownership must survive launch/service failure.
Direct spawning requires an equally qualified, versioned complete launch recipe.
Three Jev consultations supported the driver-owned launch candidate and placing
compiled-Can acceptance early in the implementation plan. The alternatives and
probability spread are examined in the consultation record. No new browser
execution is admitted by either option. If the host cannot establish
the required boundary, it remains unavailable or a qualified isolated host must
be selected explicitly.

Receipts distinguish input settlement, route contact/delivery, application facts,
sealed event intervals, host effects and resource release. A later credible
isolation failure appends a correction keyed to the original run/report/receipt;
qualification consumers resolve the current chain. A fresh successful run does
not erase the failed one. Missing observations, malformed reports, unknown
effects and incomplete cleanup remain non-success.

The existing failure policy remains: stop admission on shared trust/ownership
loss or unresolved effects, retain partial evidence, drain/terminate within the
outer budget and reclaim only proven owned resources. A PID/name/path sighting
is not signal authority. Foreign processes may prevent safe deletion without
becoming kill targets. Expected subordinate failure needs an intact outer judge
and owner; killing the real judge cannot pass its own test.

## Resource limits and remaining gates

No ceiling or concurrency increase follows from the probes. Keep one live case,
one verification worker and one build producer with the defined phase ordering.
The quick envelope remains 4 GiB, 64 processes and 512 MiB temporary data, with
finite stream/event/operation limits and cleanup reserved inside the total
deadline. The larger profile and disk floor remain as specified in the
[lifecycle contract](native-can-test-lifecycle-2026-09-30.md#5-concurrency-and-resource-admission).

Browser samples of 1.72 GiB and 15 processes do not prove future sufficiency or
strict enforcement. Each host profile needs identified enforcement, charged scope
and a demonstrable finite overshoot bound; poll-and-kill has no such proof here.
Unavailable required enforcement blocks integrated execution. No broad build,
performance run, service start, download or persistent cache was created here.

These are dependencies now assigned in the [implementation plan](../implementation/native-can-tests-plan-2026-09-30.md), not tasks executed in this design round:

| Acceptance gate | Must precede |
| --- | --- |
| Independent N ownership, enforcement, framing, recovery and receipt controls; separately accepted R and suite snapshot | Authoritative integrated Can results |
| Nonpublishing ordinary Can entry, required offline verification and canonical typed package examples | Can-owned live case acceptance |
| N1 full representation/identity controls and N2 compiled ingress with independent invocation witness | Candidate-native adapter migration; N3 then covers late identity/lifetime |
| Noninteractive browser launch/host observer plus B1 route/settlement/service-death controls | Browser migration; B2–B5 retain their task semantics, page Fetch, codec and remote cleanup gates |
| Go/Can F1 delivery/startup controls, then F2 inherited lease | Launcher/environment and lease coverage retirement |
| D1–D4 independent DB observation, fresh-pool, rollback and settlement controls | Affected SQL facets and browser durable replay |
| Full row/facet/environment evidence, controls, callers and correction-aware receipts | Deletion of each old harness |

All 14 integrated entries in the [feasibility register](preparation/native-can-test-hard-cases-2026-09-30/experiments.md)
remain unrun. Unimplemented acceptance does not require constructing the whole
system before writing its plan. Failure at a gate blocks dependent migrations
and reopens the affected contract; it does not silently trigger a host fallback.

## Independent review and disposition

Three separate reviewers examined the existing design and retained raw evidence:
[coverage](preparation/native-can-test-design-review-2026-09-30/coverage-review.md),
[trust](preparation/native-can-test-design-review-2026-09-30/trust-review.md), and
[lifecycle/resources](preparation/native-can-test-design-review-2026-09-30/lifecycle-review.md).
Their initial line references describe the pre-revision snapshot; the canonical
contracts linked here contain the resolutions.

| Review issue | Resolution |
| --- | --- |
| Missing host-UI admission/deletion scope | Added to completion, capabilities, lifecycle, ledger and B1 acceptance |
| Earlier clean receipt contradicted by late host effect | Defined append-only correction chain and conservative consumer behavior |
| Click fulfillment conflated with delivered result | Split actual input, route, application and close/seal evidence |
| Candidate can self-report adapter invocation | Defined independent version-bound N2 witness and bypass control |
| Preparatory support could be misread as row qualification | Added per-mechanism coverage exclusions and corrected stale status text |
| Application replay key confused with native at-most-once operation ID | Same application `operation_id`, fresh native request/action/route IDs; no replay of uncertain native operations |
| Flexible native catalogue could absorb test policy | Explicit binding review gate; Can retains scenarios and expectations |
| Resource samples could be treated as quotas | Preserved strict admission as an unqualified host gate and kept serial ceilings |

Three semantically matched, fully reworded Jev requests, their pre-send audit and
all responses are saved under [consultation evidence](preparation/native-can-test-design-review-2026-09-30/consultation.md).
All three selected the pinned driver launch inside N-owned supervision and an
early compiled-Can qualification lane before dependent migrations. The probability
spread leaves meaningful alternatives; the record investigates those tradeoffs.
These are advisory classifications, not proof of feasibility or bias removal.

The initial transmission rejection was resolved after the user clarified that
the project is open source and its technical context is not proprietary. That
standing authorization is now recorded in [AGENTS.md](../../AGENTS.md). The exact
saved summaries were then sent successfully; historical failed attempts remain
recorded. No consultation or design-review action is still blocked.

The [validation record](preparation/native-can-test-design-review-2026-09-30/validation.json)
covers source-ledger drift, document/evidence consistency and retained gate status.
It is static verification, not runtime qualification. No product source changed,
no host harness was deleted and no replacement coverage was credited.

## Implementation handoff

The [implementation plan](../implementation/native-can-tests-plan-2026-09-30.md) now assigns reference acceptance, the external owner, capability slices, integrated qualification, per-row migration and harness retirement to dependency-ordered lanes. Its [planning review](../implementation/native-can-tests-plan-2026-09-30/review.md) records sequencing and ownership corrections. All implementation tasks remain planned; this handoff adds no runtime or coverage credit.
