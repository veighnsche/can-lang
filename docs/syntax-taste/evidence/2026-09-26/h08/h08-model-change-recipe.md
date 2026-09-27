# H08 model-change recipe (for IC2/H14 and future operators)

How to evaluate replacing the pinned triage model with a candidate.
The recipe is registered now and executes once the H08 live gate
opens (spend approval, price table, qualified bound — see
`h08-report.md`). Harness entry: `bun tools/runtime/ai-eval/main.ts`.

## Rules

1. A model change is a new profile. Never reuse the baseline U: pin
   the candidate provider/model/version and re-qualify its own bound
   under X-R14-1 before any budgeted send. Until then the harness
   rejects every candidate call with
   `ai_budget::unavailable/missing-qualification` — that rejection
   is the guard working, not a result.
2. Same frozen cases, same protocol version. Run the registered
   representative (12) plus held-out (6) sets serially, with the
   same request bytes except the model field. If the cases must
   change, re-register first (new hashes); comparisons across
   registrations are invalid.
3. One variable at a time. Candidate-vs-baseline runs share tenant,
   pool, epoch schedule, timeout, and ledger backend.

## Steps

1. Register the candidate: provider/model/version, endpoint,
   re-qualified U with its proof reference, effective date.
2. Baseline run (live): representative set, PG ledger, fresh run DB
   (`provision-local.sh db mkdb can_h08_run<N> pg`), record report.
3. Candidate run (live): identical invocation except the pinned
   model; the runner enforces serial execution and the spend cap.
4. Compare: per-case flip table (baseline verdict → candidate
   verdict for all 18 cases), accuracy delta on the representative
   set, held-out accuracy reported separately, token/cost deltas,
   latency-max delta (reported, no SLO), budget/correlation checks
   (ledger match, case-correlation propagation, zero unexpected
   holds).
5. Promote only if the candidate passes the registered threshold
   (≥10/12 correct, ≤1 abstain, 0 invalid-output, 0 rejected, ledger
   match, correlation ok) with versions and limitations recorded.
   Neutral or adverse results are retained equally; they never
   reverse the selection by intuition.
6. Rollback is re-pinning the baseline identity: its U stays valid
   because the pinned identity never changed.

## Artifacts per comparison

- Both run reports (live provenance, `evidence: true`).
- Flip table + delta summary with model versions pinned.
- Ledger-backend note (PG default; alternates only per F04).
- Limitations: single serial run each; host wall-clock latency, not
  a provider SLO; USD only via the pinned price table.

## Follow-up notes (2026-09-27; binding on future qualification runs)

- **Pinned tariff.** The v3 price table carries the published
  input tariff (4.2e-8 USD/token) as the single gate price with
  source URL and retrieval date. Settled input-plus-output tokens
  billed at the input rate overstate true spend (output is free)
  in the safe direction: spend tracking reaches the cap early.
  Handoff (outside the follow-up slice): exact split billing
  needs a `tools/runtime/ai-eval/` change billing each side at
  its own rate — `quoteSplitCost` in `runtime/ai/bound.ts`
  already implements the arithmetic with tests.
- **Bound-qualification protocol (boundary-plus-sweep).** Before
  any candidate U is adopted for a pinned identity: (1) an
  over-budget probe establishing reject-without-consumption (typed
  error, zero usage); (2) a near-budget accept bracketing the
  enforced cap from below with exact usage; (3) the full 18-case
  sweep with ledger-reconciled usage; (4) adversarial max-length
  repeats plus byte-identical determinism repeats; (5) every
  figure verified through `verifyCandidate` in
  `runtime/ai/bound.ts` and filed as a redacted artifact (hashes,
  lengths, usage — no prompt text, bodies, or credentials).
  Stopping rule: rejects carry no usage, so the cap bracket's
  upper edge is rate-derived — dense bracketing shrinks ε, but
  only provider publication of the exact enforced figure closes
  it. Record ε explicitly; never present a bracket edge as the
  cap value.
- **Re-verification on provider change.** The pinned tariff and
  any adopted U rest on provider publication plus the verification
  above. If the Models page changes price/context figures, or a
  new model version ships, re-run this protocol for the new
  identity before any budgeted send — never carry U or the tariff
  across an unpublished change.
- **Candidate (unadopted).** U=33280 (32768 input + 512 output)
  for `typesafe/jev/1.13.0` is recorded in
  `h08-followup/bound-u-investigation.md` as a measurement-backed
  candidate only. Adopting it requires the exact enforced input
  figure plus output-side grounding (provider text or a recorded
  program decision) and then the protocol above through the live
  gate.
