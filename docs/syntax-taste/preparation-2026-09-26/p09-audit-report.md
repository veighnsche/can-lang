# P09 independent audit report (saved verbatim from the audit subagent)

Verdict at delivery: NOT READY — 9 blockers, all small recorded decisions.
Resolution: all B1–B9, M1–M9, m1–m6 resolved below in p09-readiness.md; recheck recorded there.

---

# P09 Readiness Audit — Can implementation preparation (2026-09-26 round)

**Overall verdict: NOT READY to write the implementation plan.** Coverage and evidence quality are strong and all source anchors I checked hold, but 9 blockers (all small, all preparation-scope) must be resolved first. None requires new research — each needs an explicit decision recorded in the named artifact.

## P09.1 — Coverage, choices, evidence, contracts, boundaries

| Item | Verdict |
|---|---|
| Coverage of R01–R16 | **PASS.** Every topic has ledger entries, a research packet, alternatives, a disposition, a contract (or explicit gate), and acceptance. All DEFER/REJECT rows (P19, P10, P07-race, Can-worker, etc.) carry reopening conditions. |
| User choices honored in contracts | **PASS with 1 blocker (B1).** Q1, Q3–Q8 and S-answers trace exactly into contracts. Q2 ("demote to advisory lint") presumes lint machinery that does not exist (`canlc lint` was retired; no advisory-diagnostic vehicle exists) — the delivery mechanism is unspecified. |
| Evidence quality | **PASS.** Facts vs uncertainties are labeled in all four research packets; no measured advantage is claimed from intuition (P02.2 thresholds are comparison rules, not SLOs); Jev agreement is treated as advice with disagreements investigated (verified against raw responses). Conditioned branches are explicit for R15; partial for R04/R01 (see B7, B8, M1). |
| Cross-topic contract consistency (C-A…C-I) | **PASS with gaps.** Content is used identically across `alternatives/README.md`, `p07-reconciliation.md`, `p07-contracts.md` — except: C-G has no lane owner (B2); bulk construction (R05 O-B2) has no contract home or lane (B5); R16-03 bullet contradicts the R11 auth contract (B9); lane F omits C-A/C-E inputs it is named a consumer of (M2). |
| Lane/ownership boundaries | **FAIL (blockers).** Generated files are protected and catalogue merge has one owner, but several multi-lane files are unowned with no handoffs (B6), and lane H's R16 outputs touch C/F-owned files (B9). |

## P09.2 — Scenario walks

| Scenario | Result |
|---|---|
| 1. Second-app edit propagation | **PASSES.** Lane C owns libs + both apps; breaking-edit and capture/wire legs extend U03; A→C formatter handoff exists. Minor: re-migration duty if C authors before A's formatter lands is unstated (m6). |
| 2. Cancel-while-replacing S3 key | **BREAKS under O2.** Under O1 (`end(Error)` verifies) the walk passes. Under O2 (honest-destructive) the W5 leg "preserves original bytes+etag" fails by construction, and the caller-acknowledgment source shape is unspecified (B8). |
| 3. Stalled-SQL request + SIGTERM | **BREAKS.** Budget composition (serial vs shared) is deferred to lane E's implementation-time spec — a material public-contract choice. W5 SQL/SIGTERM/disconnect legs are unconditioned on X-R04-1/X-R04-3; the cancel-absent branch has no bounded-behavior story (B7). |
| 4. Old-browser vs new-server rollout | **BREAKS.** Expected interop behavior (wire-compat vs fail-closed vs forced refresh) and its mechanism are unspecified, and the "old-browser acceptance definition" is itself a lane-H implementation deliverable — acceptance deferred to implementation, violating P07.4 (B4). |
| 5. Two-worker crash recovery via companion | **BREAKS.** `pending`/`ack` has no claim/lease: two workers can fetch the same rows and double-deliver (idempotent ack does not prevent duplicate downstream delivery). Delivery guarantee, poison/dead-letter rule, and companion restart responsibility are all unspecified (B3). |
| 6. 100k-step fault at step 60k | **PASSES with minor gaps.** Lowering + preserved failure identity walk cleanly. Step-index reporting shape is unspecified (M8); the proof predicate "no pending frame-owned work" needs an exact definition + negative-fixture list in the lane-A task (M9). |

## P09.3 — Findings

### Blockers (resolve before planning; each is a recorded decision, not new research)

| ID | Severity | Artifact + section | Fix |
|---|---|---|---|
| B1 | Blocker | `p07-contracts.md` R08/Q2; `p07-reconciliation.md` C-I | Specify the advisory-lint vehicle (check-pipeline severity + LSP surfacing + exit-code rule, and/or a new `canlc lint`), or return Q2 to the user with vehicle options. `compiler/main.go:216` confirms `canlc lint` is retired — no vehicle exists today. |
| B2 | Blocker | `p07-reconciliation.md` C-G; `p08-lanes.md` P08.1 rows F/H | Assign C-G to one lane: recommend **F** (destination rules, credential binding, redirects, identity vocabulary, assertion story); H consumes the identity for AI budgets; add explicit F→H handoff. Current split implies a planning cycle (F consumes what H decides; H follows F). |
| B3 | Blocker | `p07-contracts.md` R11; `p07-reconciliation.md` C-G/C-F | Specify the companion claim/lease mechanism (endpoint or table rule + worker identity + expiry), the delivery guarantee (at-least-once + unknown-write vocabulary), the poison/dead-letter rule and its owner (Can `decide_ack` vs companion), and companion restart responsibility. |
| B4 | Blocker | `p07-contracts.md` R13; `p07-reconciliation.md` C-H; `p07-acceptance.md` W6 | Specify the old-browser/new-server interop policy + detection mechanism in C-H (given no compat obligation, presumably fail-closed/refresh — state it and how). Lane H qualifies against it; it must not define it. |
| B5 | Blocker | `p08-lanes.md` P08.1/P08.3 | Give bulk construction (R05 O-B2) a lane: recommend **A** (precedent: `check/array.go` → A). Add `runtime/collections/*` → A; catalogue map/set section appended via the E-owned merge. |
| B6 | Blocker | `p08-lanes.md` P08.3 | Assign one owner + explicit handoffs for: `runtime/platform/{action-json,action-routes,html,htmx-guard}.ts` (recommend E, server boundary; C hands R03 patch to E), `action-client.ts` (recommend E for the R04 deadline; or C with E handoff — pick one), `check/{actions,action_bindings,browser}.go` (recommend C; E hands R04-operand patch to C), `emit/action*.go` (recommend A; C hands R03 emission patch to A), `syntax/*` R03 grammar slice (recommend A; C hands patch to A). |
| B7 | Blocker | `p07-contracts.md` R04; `p07-acceptance.md` W5 | Settle budget composition (serial vs shared) or record explicit branches; condition W5 SQL/SIGTERM/disconnect legs on X-R04-1/X-R04-3 with a specified cancel-absent acceptance story (supervisor-bound, honest unknown outcomes). |
| B8 | Blocker | `p07-contracts.md` R15; `p07-acceptance.md` W5 | Condition W5 S3 legs per branch (O1 preservation proof vs O2 destructive-with-acknowledgment); specify the O2 acknowledgment source shape (flag arg, separate op, or suppression — or constrain O2 out). Keep the O2-deadline user-scoping note. |
| B9 | Blocker | `p07-dispositions.md` R16; `p07-contracts.md` R11; `p08-lanes.md` P08.1 row H | Resolve contradiction: R16-03's "endpoints unauthenticated" bullet vs R11's "authentication added". Move R16-01 → lane C and R16-03 → lane F (conditioned on the auth design, bullet rewritten to describe the envelope); H keeps R16-02/R16-04. |

### Majors (fix in preparation or as first plan tasks)

| ID | Artifact + section | Fix |
|---|---|---|
| M1 | `p07-acceptance.md` W2; `p07-dispositions.md` R01 | W2 ("no compiler patch") prejudges X-R01-1: a catalogue-only outcome fails it. Condition W2 on the discrimination outcome. |
| M2 | `p08-lanes.md` P08.1 row F; `p07-reconciliation.md` C-A/C-E | Row F omits C-A (companion batch mirror) and C-E (endpoint status honesty) inputs though named a consumer. Add inputs or drop the consumer claim. |
| M3 | `p08-lanes.md` P08.5; `p07-acceptance.md` W1 | P08.5 presumes Firefox runners exist; gate5 runs Chromium+WebKit only (Firefox accepted by `invoice-contract.mjs` but not in the matrix). Add an explicit lane-C provisioning task. |
| M4 | `p08-lanes.md` P08.1/P08.5 (P09.4) | Provisioning (S3 bucket/creds, PG/MySQL instances, AI creds + spend approval, x86 scheduling) is implicit. Add explicit provisioning tasks + owners gating W5/W6 legs. |
| M5 | `p07-dispositions.md` R02 | Full-breadth scope vs P10 history/WebSocket deferral needs an explicit rich-client boundary in the R16-04 story requirements. |
| M6 | `p07-acceptance.md` W6 | Qualify "migration recipe" as the operator-owned procedure (per P19 deferral), not versioned migrations. |
| M7 | `p07-dispositions.md` R14 | "Token budget" scoping is deferred to lane H but touches shared C-G identity. Decide identity-affecting parts with the C-G owner (B2) first. |
| M8 | `p07-contracts.md` R05/C-A; `p07-acceptance.md` W4 | Specify the step-index reporting shape (failure field, log linkage, or occurrence linkage). |
| M9 | `p07-contracts.md` R05 | Define the tail-proof predicate exactly (leases? owned values? timers?) + negative-fixture list (mutual recursion, frame-held leases) in the lane-A task. |

### Minors

m1: `p07-dispositions.md` R13 — split F-R13-01's dual SATISFIED/CHANGE into sub-rows for mapping clarity. m2: list single-lane owners explicitly (`runtime/ai/*`, `examples/native-ai/**`, top-level `README.md` → H; `check/browser.go` → C; `tests/` subdirs per lane). m3: mark X-R11-1 superseded under companion scope. m4: S3 orphaned-part accounting under the O2 branch is unspecified. m5: W4 "bounded worker batch" reinterpretation under companion scope (C-A mirror) should be restated in the lane-F task. m6: state the formatter re-migration duty if lane C authors before lane A's formatter lands.

## Anchor spot-checks (12 file:line anchors + Jev records — all hold)

Confirmed at the recorded revision: catalogue cancel contract (`catalogue.json:12399` "never deletes the key") vs `s3.ts:199-225` `scrub()` end+delete with the self-acknowledging comment; Boolean-order rejection (`completion_matches.go:444-451`); C8 local check (`locals.go:84-95`); near-capture callee-name lookup + `CAN-CHECK-CAPTURE` (`callables.go:127-141`); LSP advertisement (`lsp.go:156`, sync + definition only); static-route capture rejection (`http.go:369-373`); GET-requires-JSON (`actions.go:318-324`); silent-500 conversions (`server.ts:300-313`); 4-field event snapshot (`browser.ts:320-326`); RETURNING rejection (`cardinality.go:54`); invoice `env::required("")` + false comment (`web.can:502-510`); stale Linux sentence (`README.md:99`); unauthenticated `outbox_pending` (`webhook/web.can:184`, no secret). `action-json.ts:253` signal field exists and the projection at `:576-584` never supplies it — R04 gap confirmed. `server.ts:488-511` confirms the shutdown nuance (no signal-path wait bound). Jev distributions in `findings.md` match the raw `jev-response-{1,2,3}.json` exactly, including both flips (C2 error_rows 0.50, C3 setup_region 0.53) and the host weakening (0.71→0.60→0.48).

## P09.4 — Acceptance owner + dependency-map destinations

Mapped: W1→C, W2→D, W3→B, W4→A (lowering; batch mirror→F per C-A), W5→E, W6→F (PG/MySQL/companion) + H (UP25/AI/story), Q1–Q3→A, LSP→G, R16→H (see B9). **Missing destinations:** O-B2 lane (B5); Q2 lint vehicle (B1); C-G owner (B2); R03/R04 shared files (B6); Firefox provisioning (M3); environment/credential provisioning tasks (M4). **Qualification blockers (how the plan must remove each):** disposable S3 bucket + creds (provisioning task, gates E's X-R15-* and W5 S3 legs); live PG/MySQL instances + creds (gates F's X-R10-1, W6 data legs; E's X-R04-1); x86 machine access + H-scheduled exclusive UP25 windows (never emulate); AI provider creds + spend caps + H-serialized live-model runs (gates R14 eval); Firefox browser provisioning (gates W1/W6 matrix legs). Unavailable live evidence is not a pass — each needs an explicit provisioning prerequisite, not an implicit leg dependency.

**Remaining blockers to close preparation:** B1–B9 above. Recommended order: B2 (ownership) → B6/B5/B9 (lanes/files) → B3/B4/B7/B8 (contract branches + acceptance conditioning) → M1–M9.