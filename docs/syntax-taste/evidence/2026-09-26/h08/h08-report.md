# H08 report — qualify AI budgets and model-change performance (W6-AI)

**Verdict: W6-AI BLOCKED (honest, per the definition of done).**
The full eval harness, frozen protocol, and rejection-only fixtures
are built and green; live quality/cost/latency measurement and bound
qualification cannot run yet. Correct rejection-only fixtures are not
claimed as a live pass anywhere in this record.

## Exact missing inputs (all three required to unblock live)

1. **Explicit spend-cap approval** (H04 ask item 2): no max-USD cap
   has been recorded. The harness enforces this mechanically —
   `--mode=live` refuses without `CAN_EVAL_SPEND_CAP_USD`.
2. **Pinned per-token USD price table**: TypeSafe publishes no
   usable price in-repo and no operator price sheet was supplied,
   so token counts are reported and USD cost is unknown. A USD cap
   cannot be enforced against unknown USD cost; live stays refused
   until prices pin.
3. **Qualified complete-call bound U** for one pinned
   provider/model/version: no derivation qualifies (analysis in
   `h08-x-r14-1.md`). Live budgeted dispatch is mechanically
   impossible until one does — the H07 guard rejects unqualified
   profiles before send.

## Credential posture (observed, nothing printed or stored)

- `TYPESAFE_API_KEY` is present in the operator shell (presence
  only; the value never appears in any artifact, log, or report —
  the harness reads it solely as a Bearer env name at dispatch).
  Presence satisfies H04 ask item 1 for the SystemOne leg only;
  items 2–3 above still block.
- No OpenAI credential env name is adopted and no access exists;
  the Responses leg is blocked on access in addition to items 2–3.
- **Zero live provider calls were made by H08.** Every run below
  uses the scripted loopback stub or skips live legs.

## What ran (all green)

- Frozen protocol v2 registered before any live result:
  12 representative + 6 held-out labeled tickets, pinned
  `typesafe/jev/1.13.0`, measures, pass threshold, serial-execution
  and ledger rules (`h08-registration.md`; hashes inside).
- Harness (`tools/runtime/ai-eval/`): protocol loader with hash
  enforcement, live gate, serial guarded runner over the real H07
  guard + SystemOne adapter, PG/file/SQLite metered ledgers
  (memory refused), spend tracker, redacted reports with
  stub-vs-live evidence labeling, CLI with `--mode=offline|live`.
- `runtime/test/ai-eval-triage.test.ts`: **12/12 pass**
  (177 expects), including unqualified-profile rejection with zero
  dispatch, immediate `exceeded` with zero dispatch, over-cap input
  refusal, the full verdict mix with exact token sums (1156+276,
  no double-count of breakdown fields), unknown-hold retention
  with intact completion, breach quarantine, live-gate unit legs,
  report non-evidence labeling + case-text redaction, serial proof
  (max concurrency 1), and a 3-case metered run against live PG
  (`can_h08_run1`).
- Raw fixture regression: `ai-budget`, `ai-budget-adapters`,
  `outbound-ledger`, `outbound-ledger-sql` suites — **106/106
  pass** (835 expects, live PG+MySQL legs included).
- Offline end-to-end on PG (`can_h08_run3`, filed as
  `h08-offline-plumbing-run.json`): ledger committed 1432 ==
  expected 1432, correlation ok, one conservative unresolved hold
  of U=5000 for the usage-less reply. Face-labeled
  `provenance: stub, evidence: false`.
- `bun run check:runtime` green; `go test
  ./compiler/internal/browser/ -run TestRuntimeInventoryMatchesBodies`
  green (no runtime modules added).
- Unrunnable here (recorded, not passed): native-ai `canlc assert`
  and staged integration legs need `CAN_BUN_ARCHIVE`, which is
  unset in this worktree — they skip, and no claim is made for
  them.

## Handoff to IC2/H14

- To unblock: (1) user records the USD spend cap, (2) operator pins
  the TypeSafe price table in the registration, (3) a bound U
  qualifies per `h08-x-r14-1.md` (needs provider-published
  enforced caps + live verification). Then `bun
  tools/runtime/ai-eval/main.ts --mode=live --set=representative
  --backend=pg --db=can_h08_run<N>` executes W6-AI; the model-change
  recipe (`h08-model-change-recipe.md`) governs candidate
  comparisons.
- F04 backend policy honored throughout: PG default for metered
  legs (`can_h08_run1` test leg, `can_h08_run3` CLI run;
  `can_h08_run2` superseded by the final-harness rerun),
  file/SQLite alternates only for plumbing, memory refused by
  construction. The PG test leg isolates reruns by pool (schedules
  configure once per ledger) and the ledger check reads the run's
  own epoch row, so shared ledgers cannot pollute verdicts.
- No Jev consultation was run: H04 blocks all paid provider calls
  without spend approval (classifier consultations included), and
  the blocked verdict is contract-determined — unqualified profile
  plus missing live inputs — not a design judgment call.
- Files: `tools/runtime/ai-eval/` (6 TS modules + frozen
  `protocol/`), `runtime/test/ai-eval-triage.test.ts`,
  `docs/syntax-taste/evidence/2026-09-26/h08/` (this report,
  registration, X-R14-1 analysis, model-change recipe, plumbing
  run JSON).
