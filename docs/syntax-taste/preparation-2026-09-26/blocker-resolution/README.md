# Task-writer blocker resolution — 26 September 2026

**Design status: BLK-01 and BLK-02 closed. Implementation remains pending.**
User authority is [1A / 2A / 3A](../p06-answers.md#task-writer-blocker-answers-1a--2a--3a).
This supplement completes R09/R14 and is incorporated by the selected contracts,
acceptance matrix and implementation task list. No production code or qualification
tests were run. Jev consultations below are advice, not qualification evidence.

## BLK-02: local references and rename

Find References and safe rename include variables defined inside functions,
project declarations and the previously selected `with` parameter references.
Resolve references by binding identity, including nested captures; spelling alone
never joins shadowed variables or variables in other scopes. Rename retains
overlay validation and cannot capture another binding. Definition lookup's
separate conservative behavior is unchanged. G03/G05 implement and qualify this
choice; their preparation block is removed.

## BLK-01: selected tenant token accounting contract

### Unit and reset policy

- One tenant allowance counts **input_tokens + output_tokens**, unweighted,
  across the tenant's budgeted provider/model connections. Do not add cached or
  reasoning-token breakdowns a second time. No money quota, separate mandatory
  input/output quotas, or allowance-waiting queue is selected.
- Trusted server configuration supplies a budget-pool identity, nonnegative
  integer token limit, positive fixed period length in milliseconds, and UTC
  epoch anchor. No arbitrary daily/monthly default is built into Can. All
  counters and arithmetic must fit checked nonnegative Can integers.
- The ledger's authoritative clock determines the half-open epoch containing
  admission. Each row is keyed by authenticated tenant, budget pool and epoch
  start. All processes share that row. Pin the applicable policy/version in the
  epoch record; policy edits apply to future epochs and never reset a live row.
  A period/anchor change becomes effective only at a boundary of the old
  schedule; that boundary is the new schedule's anchor. Reject overlapping or
  retroactive transitions and pin the schedule version in every epoch. An
  existing epoch always retains its original start, end and limit.
- Late usage settles to the **original admission epoch**. Starting another epoch
  does not move or forgive old usage. Fixed epochs can permit two allowances on
  either side of a boundary; this is an allowance per configured period, not a
  rolling rate limit. That distinction is intentional and must be documented.

### Admission and enforcement

- H supplies a server-native guard immediately before provider dispatch and at
  usage settlement. F supplies the durable ledger backend and C-G tenant/pool/
  correlation identities; E integrates the shared invocation/transport context.
  Browser-supplied tenant claims alone never authorize an accounting identity.
- Use the existing call/callable syntax for a server-only catalogue scope
  operation `ai_budget::within(tenant, pool, operation)`. Trusted application
  authorization supplies tenant/pool, and `operation` is a checked callable.
  The scope adds context, not tokens, and carries callback-specific failures
  plus the two budget failure heads below, using the existing native callable
  requirement machinery. Nested scopes cannot replace an active tenant/pool;
  same-context nesting is allowed. Scope completion/disposal cannot leak a
  tenant context into another invocation. This is a native library operation,
  not a new grammar form or an authored generic error-set feature.
- Server startup configuration marks the provider connections requiring this
  guard and injects their policy/ledger services. Their native adapters reject
  missing context and cannot bypass the guard via a direct call. Explicitly
  validate that the context pool is the connection's configured permitted pool;
  an arbitrary pool name cannot create fresh allowance. Unmetered applications
  remain explicitly separate; W6's tenant feature uses only guarded
  connections. Native AI grammar and its result-value types remain intact.
- A **qualified metering profile** must supply a conservative upper bound U for
  the complete encoded call's provider input plus output consumption. Include
  instructions, schemas/questions, provider overhead and applicable output
  limits. Bytes or an estimated tokenizer count are not proof of such a bound.
  Pin provider/model/profile identity and version; changing them invalidates
  qualification unless the existing proof explicitly covers that change.
- Before sending, atomically reserve U only if `committed + held + U <= limit`.
  Otherwise return `ai_budget::exceeded` immediately, without provider I/O and
  without waiting for refill. “Immediately” permits the bounded atomic ledger
  operation; it forbids a quota-wait queue, not all storage latency.
- A profile without the required bound returns `ai_budget::unavailable` before
  dispatch. Neither existing adapter currently has a qualified whole-call bound.
  The implementation must establish one before claiming successful budgeted
  operation; it cannot silently switch to estimated admission or post-debit.
- Ledger reserve/dispatch-marker/settle steps use short native SQL transactions
  or equivalent atomic operations with the same durable contract. Never keep a
  transaction or connection lease open while awaiting the provider. SQLite,
  PostgreSQL and MySQL backend qualification remains required.

### Invocation identity, settlement and uncertainty

- Each actual provider attempt has a durable unique invocation ID and pinned
  tenant/pool/epoch/profile/reservation U. Logical request correlation may group
  attempts, but retry sends need different accounting records/reservations.
- Reserve commits must be known before dispatch. Reconcile commit uncertainty
  by invocation ID; do not send based on an ambiguous reservation. Persist a
  once-only dispatch fence before provider I/O. A crashed or uncertain fenced
  attempt is not automatically resent using the same reservation.
- Validate trustworthy top-level input/output counters as nonnegative integers
  even on refusal/truncation/other provider outcomes that report usage. Metering
  is independent of whether the authored result subsequently validates. Keep
  provider/model provenance internally; do not expose credentials or prompt text.
- With authoritative actual usage A <= U, idempotently remove the hold, debit A
  and release U-A in the original epoch. Duplicate settlement has no effect;
  conflicting settlement is a metering failure. Known pre-dispatch rejection can
  release U only when the dispatch fence proves no send could have occurred.
- When trustworthy usage or definitive no-dispatch proof is unavailable,
  missing/invalid usage, timeout, cancellation, non-2xx, process failure or
  uncertain dispatch leaves the **full hold unresolved and durable**. An outcome
  label never overrides authoritative settlement evidence. No timer,
  request disposal, restart or empty response automatically refunds it. Only
  authoritative usage or definitive no-dispatch evidence can later reconcile it.
  Old-epoch unresolved holds remain an audit obligation, not current-epoch credit.
- If trustworthy usage exceeds U, record the actual amount without clamping,
  quarantine that metering profile from new budgeted sends, and report a profile
  breach. A qualification assumption failed; do not claim the strict cap held or
  conceal the excess as a normal successful accounting outcome.
- The existing provider/domain completion remains intact when settlement is
  uncertain; retain the hold and emit a sanitized correlated accounting report.
  A trustworthy detected profile breach or conflicting usage invalidates the
  budgeted result with `ai_budget::unavailable`. Reporting and retries must not
  duplicate provider work or erase the original occurrence/provenance record.

### Typed failures and lifecycle

- `ai_budget::exceeded` means the known admission reservation does not fit.
  Its inert detail contains the opaque budget-pool ID, required U, remaining
  tokens and epoch reset time. The application can handle it as an ordinary
  finite domain failure. It never means a money-price or provider-rate failure.
- `ai_budget::unavailable` covers missing/invalid context or policy, missing
  qualification, unavailable/ambiguous admission ledger, invalid metering,
  conflicting settlement or a breached profile. Use a finite reason variant
  plus safe correlation/occurrence identity; never attach native messages,
  endpoints, credential values or raw provider bodies.
- Add these explicit native failure bounds to guarded invocations and the scope
  operation; preserve existing provider failures and result types. There is no
  queue option, new language keyword or broad effect system. Later API examples
  use the existing catalogue call/type/error syntax.
- Cancellation ends user-visible work under C-C, never erases usage or revokes
  a live lease. Durable accounting outlives an individual request and process.
  Missing admission storage fails closed under request deadlines. Health and
  operator reports distinguish committed tokens, held tokens and unresolved
  attempts without logging secrets.

### Qualification and implementation handoff

- **F01:** C-G identities and durable native ledger service contract; supply
  atomic reserve/fence/settle/reconcile operations and backend fixtures. It
  publishes before H07; it does not wait for AI enforcement. F04 supplies the
  live dialect evidence. E handles shared runtime integration via file handoffs.
- **H07:** `ai_budget::within`, guarded dispatch, usage extraction/validation,
  profile qualification, strict admission and typed failures. Catalogue/checker
  integration goes through existing A/E ownership; no generated hand edits.
- **H08 / X-R14-1:** qualify at least one actual budgeted feature/profile and
  execute W6-AI. Primary worked feature: server-side support-ticket triage into
  existing finite structured categories. Register representative/held-out cases,
  provider/model versions, measures and application-quality threshold before
  looking at evaluation results; report correctness versus labels, abstentions/
  invalid output, cost and latency. This feature fixes an evaluation destination,
  not a claim of model quality. Keep raw provider fixtures distinct from live
  metering/quality evidence. Live evaluation spend approval still applies.
- Positive evidence: exact-fit/rejected admission; parallel requests across two
  processes; no bypass through direct guarded calls; immutable tenant identity;
  summed counts without double-counted breakdowns; idempotent settlement;
  epoch-boundary/late results; restart and commit-unknown recovery; callback
  failures, refusal/truncation, malformed/missing usage, cancellation and profile
  breach; no secrets in reporting. Rejection must prove zero provider dispatch.
- Negative qualification is specified: unsupported bound/profile or unavailable
  ledger rejects with typed failure, and unknown usage preserves holds. **If no
  real profile can be qualified, W6-AI is blocked**, despite correct rejection
  tests. Return the affected contract/scope to preparation; never silently make
  the cap soft or substitute mocked usage for successful live qualification.

## Evidence and Jev use

Source inspection found: [Responses](/Users/vince/Projects/can-lang/runtime/ai/responses.ts:78)
sends max_output_tokens but discards usage; [SystemOne](/Users/vince/Projects/can-lang/runtime/ai/questions.ts:185)
has model/state/questions only and drops usage/model from its returned answer
projection; [connection metadata](/Users/vince/Projects/can-lang/compiler/internal/check/connections.go:119)
has no token ledger; [non-2xx bodies](/Users/vince/Projects/can-lang/runtime/transport/fetch.ts:90)
never reach AI decoding. Existing fixtures omit usage.
No whole-call bound, atomic reservation or durable invocation ID exists there.
These are implementation gaps, not proof that the selected semantics already run.

Live references read: [TypeSafe documentation index](https://docs.typesafe.ai/llms.txt),
[HTTP API](https://docs.typesafe.ai/api) and [Choice](https://docs.typesafe.ai/primitives/choice).
The API documents completed input/output usage and no request output-cap field.
Markdown URLs failed in the browser tool; the normal pages succeeded. No other
provider's token-bound guarantee is inferred from this documentation.

Three fresh requests were prepared before the first response. All 25 explanatory
fields differ pairwise; facts, constraints and alternatives were manually checked
and independently reviewed. The review tightened “retry charges” to “retry
accounting records” before sending. [Wording audit](jev-wording-audit.json),
`jev-request-1.json` through `jev-request-3.json`, exact response files and
metadata preserve the consultation. `consult.py` reads credentials only from
the environment. No agreement is treated as proof or guaranteed bias removal.

| Question | Request 1 | Request 2 | Request 3 |
| --- | --- | --- | --- |
| Admission | qualified_bound 0.99 | qualified_bound 0.82 | qualified_bound 1.00 |
| Enforcement | durable_native 0.72 | durable_native 1.00 | durable_native 0.93 |
| Window | configured_fixed 0.95 | configured_fixed 0.81 | rolling 0.58 (fixed 0.42) |
| Unknown usage | retain 0.98 | retain 0.99 | retain 0.99 |
| Counting | sum 0.94 | sum 0.80 | sum 1.00 |

Resolved model: `jev-1.13.0`. Total usage: **4,361 input + 593 output tokens**.

**Disagreement investigation:** rolling windows prevent boundary bursts that
fixed epochs permit, but require expiring individual consumption/reservations
and more complex late-settlement reconstruction. Fixed epochs keep one durable
accounting identity per attempt and allow the application to configure its
period without a universal calendar policy. The user selected a token allowance,
not a rolling rate guarantee. Engineering therefore selects configured fixed
epochs, explicitly documenting the boundary behavior. The third response is a
valid counter-case, not a vote discarded to manufacture unanimity.

**Other counterevidence:** authored wrappers are simpler to add (request 1 gives
them 0.28) but leave native calls bypassable unless every author routes correctly;
the selected native guard enforces coverage at the actual dispatch boundary.
Estimated admission may enable more providers sooner but cannot satisfy a strict
upper bound; fail-closed qualification can make initial provider availability
limited. Retaining unknown holds is conservative and can underutilize an epoch,
but refunding possible consumption breaks the allowance. These costs remain
visible in qualification and operational documentation.
