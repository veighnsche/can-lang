# Language design fixes — implementation plan

Status: ready for implementation planning handoff on 2026-09-22. The accompanying [ordered task list](language-fixes-tasks-2026-09-22.md) contains **21 pending tasks**. This plan covers the 16 accepted September 22 language fixes; it does not restart the completed I01–I50 compiler replacement or claim that the new behavior is implemented.

## Contract and scope

[Selected decisions](../syntax-taste/decisions.md) and the incorporated [core](../syntax-taste/technical-spec.md), [AI/I/O](../syntax-taste/ai-io-spec.md), [coordination](../syntax-taste/coordination-spec.md) and [platform/testing](../syntax-taste/platform-testing-spec.md) specifications define behavior. The [49-finding ledger](language-design-dispositions-2026-09-22.md) fixes scope, the [16-change acceptance specification](language-change-acceptance-2026-09-22.md) fixes completion evidence, and the [verification report](implementation-gap-verification-2026-09-22.md) distinguishes actual defects from intentional limitations. The behavior document is a navigation index, not an overriding alternative specification.

LD29 is now closed at the design level by C9.2: `checks::require` emits the ordinary `checks::failed` domain error and uses existing assertion expectations. No accepted language-design gate remains before planning these tasks. A plan is not implementation evidence: all AE records remain evidence-required.

Preserve native AI forms, grouped state, explicit contracts, attached assertions, exact-name captures, nominal immutable data and the four coordination meanings. Error-set parameters, changed captures, state-callable redesign, static resource escape analysis, general inheritance, LLM/SQL normalization and root-owned fixture overrides stay outside this work. The 18 retain and 15 defer dispositions remain visible in the ledger; nothing is silently removed or queued for implementation.

No backwards-compatible syntax, ABI, generated layouts or old goldens are required. Change current source fixtures alongside the implementation that changes their contract. Reuse native JavaScript/Bun operations and existing runtime adapters; do not build a second compiler, scheduler, collection library, assertion engine or publication store.

## Implementation sequence and milestone gates

Run the task list in its written order. The explicit dependency graph is acyclic and each dependency precedes its consumer. The order provides one safe serial execution path; it is not a claim that every adjacent task has a technical dependency.

| Milestone | Tasks | Observable exit |
| --- | --- | --- |
| 1. Evidence, locations and bounded private tests | LF01–LF04 | Revision inputs are pinned; semantic locations work; assertion execution never publishes; pending and CPU-blocked workers produce bounded reports |
| 2. Completion contracts | LF05–LF08 | Exact generic heads, final-success order, standard snapshots and named domain checks execute with correct positive/negative cases |
| 3. Operation normalization and owned tests | LF09–LF13 | Origin-aware fetch/judge normalization, attached raw native tests, inherited wrapper policies and lexical fixture templates work together |
| 4. Verified publication | LF14–LF15 | Captured source/fixture/asset identities and dependency locks cover the whole graph; build publishes only after every required root passes |
| 5. Composition, tooling and explanation | LF16–LF19 | Native map-based aggregate conversion, validated diagnostic fixes, safe comment-preserving formatting and accurate guides have evidence |
| 6. Measured and integrated acceptance | LF20–LF21 | Equivalent-program/workflow measurements and local bundle qualification support every accepted change; all AE/BC evidence is audited |

The critical chains are:

- Private assertion staging → external supervision → complete input capture → all-root verification → atomic production publication.
- Exact error heads and origin metadata → public normalization → raw native roots → derived wrappers → exact-target templates.
- Semantic spans → new-form provenance → checker-validated fixes → safe formatting → trustworthy edit-cost measurements.

Publication is wired last among the test-system changes because it must cover the final native/wrapper/template root set and captured fixture bytes. Until LF15 passes, a green build still cannot be presented as the revised all-root guarantee. The hanging-assertion repair does not change production empty-race behavior.

## Current implementation and code ownership

The starting verification baseline is commit `760405f537bb8a2fb6d7f277ba27888a7e3f5dc2`; LF01 records the actual implementation checkout and detects drift. The saved evidence already confirms build/assert publication coupling, a pending assertion with no final report, and lost semantic LSP locations. Resource handles escaping source scope are admitted but later use is blocked by runtime ownership before native work; preserve that boundary, do not prescribe a new escape analyzer.

Use the current repository owners:

| Area | Existing owners and intended work |
| --- | --- |
| Source/grammar | `compiler/internal/source`, `compiler/internal/syntax`: spans, exact heads, native/wrapper assertions, fixture declarations and preserved trivia |
| Resolution and checking | `compiler/internal/resolve`, `compiler/internal/types`, `compiler/internal/check`: nominal specialization, origin obligations, calculated bounds, local queues and complete diagnostics |
| Typed lowering | `compiler/internal/ir`, `compiler/internal/emit`: carry checked distinctions and source provenance; never rediscover origin or ownership from generated strings |
| Catalogue | `compiler/internal/catalogue/catalogue.json` and its generator: checks ID 1010, request_failed ID 1106, typed detail variant and generated mirrors |
| Runtime | `runtime/domain.ts`, `failure.ts`, `transport`, `ai`, `assertions`, `collections/array.ts`: preserve native behavior and add only contract adapters |
| Driver/project | `compiler/internal/driver`, `compiler/internal/project`: private test staging, isolated root processes, immutable graph capture, fixture locks and atomic publication |
| Tooling/evidence | Compiler CLI/editor bridge, `tools`, `compiler/testdata/current`, `tests/integration`, `tests/conformance`, `docs`: validated fixes, formatter, actual programs and local qualification |

The current `publishProgram` path performs output validation and selects current for both production and assertion modules. Split its responsibilities without weakening existing ownership, confinement, exclusive writer or reader-lease handling. Reuse that staging/selection split for both assert and build; do not implement two assertion supervisors.

For the formatter, LF18 defines the planned single-file interface `canlc format <file>` (stdout) and `--write` (validated atomic replacement). It resolves the containing project and validates the proposed overlay before writing. This is a concrete tooling implementation choice, not a new Can grammar proposal. Invalid or stale source is left untouched; formatting does not repair semantic errors.

## Acceptance coverage and closure owners

Several tasks deliberately contribute to one acceptance record. “Primary closure” identifies where the feature's complete evidence is assembled; LF21 additionally checks all records together. Completing prerequisite infrastructure alone does not mark an AE record passed.

| Acceptance | Change | Contributing tasks | Primary closure |
| --- | --- | --- | --- |
| AE07 | Normalized fetch/judge failures | LF09, LF10, LF11, LF12, LF20 | LF12 for behavior; LF20 for comparison |
| AE09 | Wrapper inheritance/calculated errors | LF05, LF09, LF11, LF12 | LF12 |
| AE11 | Failure-first completion arms | LF05, LF06 | LF06 |
| AE15 | Standard snapshots | LF05, LF07 | LF07 |
| AE17 | Exact generic-error patterns | LF05, LF06 | LF05 |
| AE18 | Explicit native map conversion | LF05, LF07, LF16 | LF16 |
| AE22 | Assertion-verified publication | LF03, LF04, LF11–LF15 | LF15 |
| AE23 | Supervised deadlines | LF02–LF04, LF15 | LF04; build integration LF15 |
| AE25 | Native/handler testing | LF11, LF12 | LF12 |
| AE27 | Local fixture reuse | LF13, LF14 | LF14 |
| AE29 | Named runtime checks | LF08, LF09, LF15 | LF08; provenance/publication regression later |
| AE33 | Capability admission guide | LF19 | LF19 |
| AE44 | Locations, explanations, safe fixes | LF02, LF17 | LF17 |
| AE45 | Safe source formatting | LF18 | LF18 |
| AE47 | Accurate finite-rule documentation | LF19 | LF19 |
| AE49 | Equivalent source/workflow measurements | LF01, LF20 | LF20 |

Every task's positive, negative and integration evidence supplements the full AE requirements rather than replacing them. Preserve all 24 BC cases, including eight-call normalization, inherited overrides, selective recovery, original provenance, generic matching across modes, sticky harness behavior, exact request fixtures and publication failure paths.

## Evidence and validation workflow

Before changing behavior, LF01 captures the original complete sources, expected outcomes and pinned tool identities. Reuse existing saved observations; rerun only what is necessary to establish the actual starting checkout. Do not overwrite a failing baseline with a passing output or call a proposed program admitted before it runs.

Each implementation task must save the changed complete source/fixture set, exact command, compiler/catalogue/Bun identities, expected/actual exit and diagnostics, relevant emitted TypeScript and runtime/fixture evidence. Assert success includes comparison and full harness/drain checks. Missing, skipped, timed-out or mocked-at-the-wrong-boundary cases are not passes. All task checkboxes start unchecked and gain evidence links only after their exit criteria pass.

Use focused Go tests beside changed compiler packages, runtime tests against the pinned bundled Bun, and complete source fixtures under `compiler/testdata/current` with integration tests in `tests/integration`. Catalogue changes use `make catalogue` and `make catalogue-check`; the existing broad Go check is `go test ./...`. Consult the existing bundle instructions for exact local runtime paths rather than assuming an ambient Bun version. Broaden to the complete relevant suites at LF21; do not rerun every unrelated integration test after each small edit. Required platform services must be reported honestly when unavailable, never silently marked green.

Keep evidence labels distinct: `real-can`, `supplied-completion`, `raw-provider-fixture`, `policy-fixture`, `bun-conformance`, `live-quality`. Mandatory native assertions use fake credentials and deterministic raw exchanges; live provider quality is not a build gate. The final local bundle check does not authorize publication, signing credentials or paid live calls.

For AE49, first prove equivalent outputs, errors, operation counts, ordering, local test ownership and coverage. Then report production source separately from total tests/templates/raw data. Measure 63→9 infrastructure bound entries and 56→8 forwarding arms only within the fixed eight-call region; also report wrapper/native-test costs and actual edit/workflow results. Never treat characters as tokens or a smaller file as correctness evidence.

## Change control and completion

This plan introduces no genuinely new difficult language decision. Reuse the saved normalization/wrapper/testing consultations and the separate LD29 consultations. If implementation produces a concrete conflict, retain its failing program, identify the affected contract and resolve that issue before dependent work. A genuinely new difficult design follows the repository's three fresh Jev consultation rule; a routine implementation choice or task ordering does not need another classification vote.

Preserve completed history in [plan.md](plan.md), [tasks.md](tasks.md) and [remaining-tasks.md](remaining-tasks.md). Their checked boxes do not count toward this revision. The implementation index points at this plan/list as the current work queue.

Completion requires all 21 task exits, every dimension of all 16 AE records, all applicable BC cases and the integrated retained-design regressions. LF21 records local qualification and updates implementation-status documentation. No release upload is part of this plan.

The [machine-readable task graph](evidence/2026-09-22/language-fixes-plan/tasks.json) and [validation report](evidence/2026-09-22/language-fixes-plan/validation.json) make the order and coverage checkable. Planning validation checks links, prerequisites, scope coverage and pending status; it does not certify compiler implementation.
