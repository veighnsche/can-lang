# P06 answers — user decisions 2026-09-26

Exact scope recorded per answer before any dependent use. Previously confirmed
choices not reopened here need no repeat confirmation.

## Scope (S1–S4)

- **S-ambition → Full breadth now.** Everything is in this round: rich-client
  widgets, full worker operations (under the companion-service home below),
  object-upload contracts, and AI quality/cost qualification. The review's
  4-step sequence stands as milestone order: (1) S3/budgets/reporting,
  (2) second app + libraries + one integration, (3) library-authoring
  exercise, (4) Linux operation with faults.
- **S-platforms → Widest matrix.** Full MySQL live parity plus more browsers.
  Concrete reading for P07: Chromium 140 + WebKit 26 (existing pins) + Firefox
  via pinned Playwright 1.55.1; SQLite + PostgreSQL + MySQL live legs;
  disposable S3-compatible bucket for storage qualification; Linux
  qualification on the x86 machine (UP25). Nothing in the current matrix is
  dropped.
- **S-worker → Companion service.** Carrier/scheduling live outside Can with
  a documented protocol + auth envelope (webhook-sample architecture,
  hardened). Combined with Full breadth: the round qualifies the Can +
  companion pair at full ops scope — it does not build a Can worker.
- **S-roadmap → Keep independent standing.** SaaS breadth is one input; AI
  judgment features, editor tooling, and Linux operations keep their own
  deliverables (R09 full sequence, R13 lifecycle + UP25, R14 budgets + eval).

## Syntax and authoring policy (Q1–Q8)

- **Q1 Boolean order → Allow both, format-canonical.** Scope: ordinary
  single-scrutinee Boolean data matches only. `true`-first programs check;
  the formatter rewrites to `false`-first (fixpoint-stable, validated like
  `--write`). Branch selection unchanged; completion/coordination/
  multi-scrutinee/AI-criteria modes untouched. This explicitly reverses the
  2026-09-23 user decision and P15 Retain.
- **Q2 final locals → Demote to advisory lint.** Scope: the C8 four-clause
  shape only. The checker stops rejecting; `subtotal`-style domain locals
  are valid. Coherent reading recorded: the formatter keeps source as
  written (no forced inline — forced rewriting would defeat the decision);
  lint flags the accidental-alias shape. Type/error behavior preserved (the
  check diagnoses only; evaluation never depended on it).
- **Q3 near binding → Explicit bindings.** Direction approved; exact grammar
  is specified in P07 (below) and stays within this approved surface —
  no unseen companion syntax is implied. Unlisted near-inputs keep current
  name lookup (partial-with-fallback); editor safe-rename follows.
- **Q4 setup region → Yes, if factories fail.** Conditional only. Factory
  pattern runs first (X-R07-1); a checked setup region may be proposed only
  on demonstrated excess cost, must never forge owner values or skip
  verification, and then explicitly reopens LD29's closed gate.
- **Q5 error-set parameter → Result-data first, param if needed.**
  Conditional only. X-R06-1 first; finite explicit error-set syntax only if
  adapters stay extensive. No broad effect system under any outcome.
- **Q6 iteration → Lowering first, syntax if needed.** Self-tail lowering
  with identical observable behavior first; explicit iteration syntax only if
  the proof boundary excludes needed shapes. Evaluation order, failure
  identity, fixture paths, diagnostics preserved under all outcomes.
- **Q7 captured routes → Yes, captured reads.** Reuse capture/URL machinery
  for server-rendered reads. Exact grammar from the P07 technical packet;
  no SPA router implied.
- **Q8 editor → Full sequence in order.** Format → hover → references →
  completion → rename. Rename follows the Q3 near-binding packet.

## P06.4 contract check (2026-09-26)

- Q1: no semantic or feasibility change — checker deletion + formatter rule
  + fixtures. No research/Jev revisit (taste decision, evidence-complete).
- Q2: no semantic change — the C8 check never altered evaluation; removal
  cannot change runtime behavior. Lint/format additions are additive. No
  revisit.
- Q3: changes the callable surface — a P07 technical packet is required
  (written below as part of consolidation, not deferred to implementation):
  `callable <name> with <param> = <expr>, ...` binds listed near-inputs
  explicitly by callee parameter name; each listed name must be a declared
  near-input; each expression checks against the declared input type in the
  creation scope; unlisted near-inputs use existing name lookup; unknown
  names, duplicates, and type mismatches are checker errors stamped
  CAN-CHECK-CAPTURE; emission captures explicit expressions instead of
  looked-up names; formatter preserves binding order; LSP rename treats
  `with` bindings as references to the callee parameter. No surface beyond
  this returns to the user.
- Q4–Q6 conditional: no grammar selected; conditions + gates recorded above.
- Q7: bounded grammar/checker delta specified in P07; stays within the
  approved direction.

## Task-writer blocker answers (1A / 2A / 3A)

The user answered the three blocker questions with `1A`, `2A`, `3A`.

- **1A — Include locals.** Find References and safe rename cover variables
  defined inside functions as well as project declarations and the selected
  `with` parameter references. Bindings are distinguished by their actual
  scope, including identical names in different scopes. This closes BLK-02;
  it does not change the separately selected definition-lookup behavior.
- **2A — Tenant token allowance.** Each tenant has a configurable input/output
  token allowance. Money-based or combined money/token product budgets were
  not selected. Live evaluation spend caps remain a separate operational rule.
- **3A — Immediate rejection.** A new AI call that would exceed its allowance
  is rejected immediately with a typed budget-exceeded failure for the
  application to handle. No allowance-waiting queue or selectable queue mode
  is selected.

The user choices do not themselves choose the accounting/reset window,
concurrent-call reservations, settlement/unknown-usage policy, enforcement
point or exact failure contract. Those technical details are now selected in
the [R14 follow-up contract](blocker-resolution/README.md), after three fresh
Jev consultations and disagreement investigation. BLK-01 is closed for design;
native profile qualification and implementation remain required. These user
answers select no new grammar.
