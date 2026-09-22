# Language design finding dispositions — 2026-09-22

Status: scope disposition requested by the user. This ledger supersedes recommendation status and suggested sequencing in the September 22 reviews. It does not change admitted syntax, authorize compiler changes, or create a new implementation task list. The existing completed implementation plan remains historical evidence.

**Can's baseline is preserved: native AI forms, grouped state, explicit contracts and source-attached assertions.** Also retain nominal immutable data, named functions, exhaustive matching, explicit connection selection and equivalent native JavaScript/Bun operations. There are no external users and no compatibility requirement; retaining a design below means it has value, not that migration is expensive.

Every atomic finding has exactly one disposition:

- **implement** — include the stated outcome in the next design/implementation scope. Exact contracts and acceptance cases must be settled before coding; proposed spellings are not adopted by this label.
- **retain current design** — keep the named behavior and decline the proposed replacement. This is a decision, not an untracked backlog item.
- **defer with a reason** — exclude from the next implementation scope. Each row names both the reason and the evidence that would reopen it.

**Coverage: 49 dispositions — 16 implement, 18 retain current design, 15 defer with a reason.** The source mapping below covers R1–R11, F1–F9, all confirmed revisions and additional candidates, all 14 full-review sections, and all 15 before/after sections.

Compound findings are split so that accepting a repair does not accidentally accept its suggested redesign. Repeated findings share a disposition ID. IDs below identify decisions, not implementation tasks.

## Demonstrated-need gate for larger abstractions

User-confirmed scope constraint: **error-set parameters (LD14), changed capture syntax (LD39), and state-callable redesign (LD06) remain deferred until a concrete demonstrated need justifies implementation scope.** Their appearance in an earlier review or before/after sketch is not authorization to implement them or a prerequisite for the accepted fixes.

Reopening requires a reproducible real program or agent-authoring task, the best solution using current Can, and evidence of the remaining limitation or material cost. Compare the proposed change against that baseline, including contracts, diagnostics, tests and edit propagation. A shorter spelling, a deliberately complex regression fixture, or classifier preference alone does not demonstrate need. A prototype may help evaluate an evidenced problem; its existence does not automatically promote the abstraction to implementation scope. Record any later disposition change and its supporting evidence explicitly.

Keep concrete error bounds, exact-name `near` captures, grouped state and the current named callable adapters meanwhile. Wrapper-only `emits calculated`, exact generic-error discrimination and lexical fixture templates do not authorize broader effect inference, capture changes or state flattening.

## Native AI, state and connections

| ID | Finding / proposed change | Disposition | Reason and scope boundary |
| --- | --- | --- | --- |
| LD01 | Replace native kinds with one `question` declaration or ordinary library functions | **retain current design** | Keep `noul`, `choice`, `score`, `judge`, `llm` and `fetch`, including generated-record capabilities. Their checked behavior and AI purpose justify distinct forms; counting keywords does not establish redundancy. |
| LD02 | Common question section order, one Choice production, and new `shape` headers | **defer with a reason** | These are grammar cleanups without demonstrated agent-workflow benefit. Reopen with equivalent native AI examples showing fewer authoring errors after normalization and tooling improvements; assess each cleanup separately. |
| LD03 | Replace `choice_arm` with ordinary callables, or merely share its stored type spelling | **retain current design** | Static descriptions, registration identity and capture restrictions are meaningful. Neither proposed unification establishes an equivalent contract. Keep the dedicated arm kind and type. |
| LD04 | Make connections ordinary records, batch by value equality, or infer the nearest connection and omit `from` | **retain current design** | Explicit nominal connection selection makes transport and batch identity inspectable. Equal configuration values are not a sufficient substitute for that identity. |
| LD05 | Flatten `given`/`state`, use per-parameter state markers, or serialize the last record argument implicitly | **retain current design** | Keep the grouped state boundary visible. Eliminating an adapter does not justify erasing a selected AI input distinction. This also rejects the flat alternative in the before/after document. |
| LD06 | First-class native callable contracts that preserve grouped state | **defer with a reason** | The adapter cost is real, but a sound representation and an ordinary AI composition use case are still missing. Reopen with a grouped-state-preserving prototype covering references, captures, fixtures and error bounds, measured against the current adapter. |

## Failure contracts and coordination

| ID | Finding / proposed change | Disposition | Reason and scope boundary |
| --- | --- | --- | --- |
| LD07 | Repeated seven-error fetch/judge infrastructure contracts | **implement** | Language-owned normalization must give ordinary callers one typed public failure, without author-written seven-error wrappers. Cover request preparation, transport and decoding at their actual origin; retain selective recovery details and private diagnostic cause. Preserve application/AI errors and standard failures in their own channels. A2.4 now fixes `http::request_failed`, typed detail alternatives and the origin boundary. |
| LD08 | Wrap individual questions or change judge batching while normalizing | **retain current design** | Normalize around the judge, never separately around `noul`, `choice` or `score`. Preserve one request, whole-batch validation and source-ordered handlers. Handler-origin errors must not be mistaken for native errors with the same nominal type. |
| LD09 | Derive fetch/judge wrappers using additions and overrides only | **implement** | Inherit omitted handling; the last override wins. Resolve handling without reissuing the operation or redispatching the batch. Selective overrides need a defined way to delegate to inherited handling. Keep the effective public contract explicit and inspectable; derive/check its obligations rather than copying boilerplate. A3.2 now selects `wrap`, origin-specific handling, terminal `inherit` and explicit `emits calculated`. |
| LD10 | Multiple wrapper bases, general inheritance, changed wrapper signatures, automatic retries, or extending normalization automatically to LLM/SQL | **defer with a reason** | These exceed the confirmed fetch/judge boundary and introduce independent precedence and failure contracts. Reopen only for a concrete operation needing them, with evidence that ordinary composition or a single derivation chain is insufficient. |
| LD11 | Errors-first, final-`ok` success/error matches | **implement** | Apply the confirmed order consistently to the grammar/checker, diagnostics, examples and fixtures. Make enforcement part of the proposed contract; do not impose an otherwise unspecified relative order among error arms or change recovery semantics. |
| LD12 | Replace explicit public bounds with `emits [=callee]` | **retain current design** | A callee implementation change must not silently define a caller's API. Keep explicit public `emits`; language-owned normalization reduces the obligations that need naming. Wrapper inheritance must still expose a checkable public contract. |
| LD13 | Named error-set aliases | **defer with a reason** | Aliases abbreviate declarations but do not normalize outcomes. Reopen if realistic post-normalization programs still repeat substantial identical application-error sets, with a concrete public expansion/checking rule. |
| LD14 | Finite error-set parameters for authored higher-order functions | **defer with a reason** | This is a substantial generic-contract extension. Reopen with an AI composition example that fixed bounds cannot express adequately and a prototype proving finite substitution, diagnostics and native lowering. Keep concrete explicit bounds meanwhile. |
| LD15 | String-only ordinary standard-failure catches | **implement** | Expose the existing structured snapshot concept in ordinary catches: kind, occurrence identity and message, without message parsing. Keep standard failures outside domain `emits`; do not expose arbitrary native exceptions. C9.1 now fixes `[_] as standard_failure f` and the immutable snapshot contract. |
| LD16 | Closed global failure union, non-generic `all_failed`, or replacing race failure with an ordinary result variant | **retain current design** | Application errors are open to application authors; a distribution-wide closed union cannot represent them. Preserve typed `all_failed<E>`, nominal specializations and the existing success/domain/standard channels. The proposed ordinary variant also cannot be assumed valid under current variant-leaf rules. |
| LD17 | Bare error names cannot distinguish exact generic specializations | **implement** | Support exact specialization discrimination and corresponding coverage/error-bound checking, including applicable coordination contexts. Prove distinct `all_failed<A>` and `all_failed<B>` remain distinct. Do not solve this through implicit generic-container covariance. |
| LD18 | Recursive aggregate-conversion scaffolding | **implement** | Demonstrate and validate the shorter explicit map-based conversion where a common aggregate is actually required; retain explicit nominal reconstruction. This is initially a fixture/documentation repair, not a new implicit conversion rule. A failure of that idiom would require a separately evidenced design decision. |
| LD19 | Unify completion headers/arm layouts or remove `match chain` | **retain current design** | Keep the four coordination meanings and chain sequencing. Per-participant recovery and whole-result recovery have different owners and outcomes; sharing an arm layout can conceal that difference. Existing chain use contradicts the claim that it is unused. |
| LD20 | New sequencing syntax or reducing fetch nesting | **defer with a reason** | Nesting is explicitly outside the confirmed normalization scope. Reopen with a real program that current chain/shared recovery cannot express clearly, after measuring normalized code. |
| LD21 | Early-stopping accumulation such as `fold_until` | **defer with a reason** | Recursion already exists, so this is an ergonomic/performance candidate rather than an expressiveness blocker. Reopen with a real early-stopping accumulator and measured cost; prefer a small native-backed catalogue operation over general loop redesign. |

## Assertions, resources and platform contracts

| ID | Finding / proposed change | Disposition | Reason and scope boundary |
| --- | --- | --- | --- |
| LD22 | Production publication does not establish the claimed assertion-verified build guarantee | **implement** | Define and enforce required assertion success before publishing the production artifact. Assertion-only execution modules are distinct from production publication. Specify failure/timeout behavior and verify the actual command path; the original statement that a hanging assertion currently hangs every build was too broad. |
| LD23 | A pending assertion can prevent a final report | **implement** | Add an externally supervised deadline, failure status and useful pending-invocation diagnostics. The supervisor must also handle a worker that cannot service an in-process timer. P15.1 fixes the root budget, out-of-process termination and publication gate. |
| LD24 | Change production empty-race semantics to solve test hangs | **retain current design** | Preserve the native pending behavior. The harness owns bounded test execution; production semantics need not change to make tests terminate. |
| LD25 | Authored native request and handler behavior has inadequate test coverage | **implement** | Extend deterministic coverage to prepared requests, state serialization, raw response validation and actual handlers/fallbacks. Preserve attached assertions and lexical fixture checks. Existing adapter conformance is useful but is not complete authored-behavior coverage. Keep live model-quality evaluation separate from mandatory deterministic builds. P4.1 fixes attached fetch/judge/LLM/wrapper roots; independent questions keep their judge-owned testing boundary. |
| LD26 | Hoist fixtures to function-name/ordinal roots, weaken expected-argument checking, or detach assertions from declarations | **retain current design** | Keep checked arguments, lexical invocation identity and source-attached assertions. Root ordinals are not a stable replacement for participant paths/callable identities; less repetition is not sufficient reason to weaken the contract. |
| LD27 | Named checked fixture templates reused at lexical sites | **implement** | Reopened by the user's subsequent request to finish fixture reuse. Adopt only the inert exact-target templates and local expansion contract in B6 of the [behavior designs](language-behavior-contracts-2026-09-22.md). Preserve root/queue ownership, checked arguments and captures; verify their diagnostic and edit behavior during implementation. Root-owned overrides remain deferred under LD28. |
| LD28 | Stable symbolic seams for root-owned test overrides | **defer with a reason** | This is a different fixture identity model, not just content reuse. Reopen only when a concrete integration test cannot be served by lexical fixtures/templates, with collision, refactoring and concurrency semantics specified. |
| LD29 | Intentional division by zero used to report failed runtime checks | **implement** | Provide a direct, named failed-check mechanism with a useful reason and diagnostic; migrate those fixture idioms. Preserve attached input/output assertions. C9.2 now selects checks::require with checks::failed domain completions and existing assertion expectations after three fresh consultations. Keep harness violations sticky: catching one must not make a test pass. Do not implicitly invent a second assertion DSL. |
| LD30 | Replace runtime resource ownership/leases with simple syntactic escape rules | **retain current design** | Keep runtime ownership, leases and scope enforcement as the correctness boundary. Aliases, returned aggregates and captures make direct syntax-only checks incomplete. |
| LD31 | Additional transitive static resource-escape diagnostics | **defer with a reason** | [Targeted verification G5](implementation-gap-verification-2026-09-22.md#g5-resource-escapes-admission-versus-usable-lifetime) now reproduces accepted direct/contained/captured escapes, while runtime use is rejected before native work. This is the selected runtime-enforced limitation, not a demonstrated safety bypass. A useful authoring case and bounded sound diagnostic analysis remain unproven; require those and accepted in-scope counterexamples before selecting a solution. Retain runtime enforcement. |
| LD32 | Distribution-only platform catalogue | **retain current design** | Keep the controlled native boundary. A checked foreign signature alone does not supply codecs, ownership, effects, immutable results and deterministic fixtures. |
| LD33 | Unclear capability admission/evolution process | **implement** | Document how capabilities are proposed, admitted, tested and retired, with their complete native contract. No compatibility promise or arbitrary foreign code is introduced. Extensibility does not gate the normalization and harness repairs. |
| LD34 | Third-party declarative native binding manifests | **defer with a reason** | No blocked program demonstrates the need for a new trust/contract boundary. Reopen with such a program and a complete adapter contract, showing why a distribution addition is insufficient. |
| LD35 | Move static JSON manifests, registries and SQL descriptors into Can declarations | **retain current design** | Keep compiler-validated build-time configuration. Its existence does not make contracts untested or require source-language syntax. A SQL-specific ergonomic case can be proposed separately with evidence. |
| LD36 | Manual error IDs and mirrored registry identity | **defer with a reason** | Redundant bookkeeping is suspected, but consumers and identity/reporting requirements have not been fully audited. Reopen after that audit demonstrates removable duplication and defines canonical identity plus generated reporting codes. No need to preserve old IDs for compatibility. |

## Syntax, documentation and authoring feedback

| ID | Finding / proposed change | Disposition | Reason and scope boundary |
| --- | --- | --- | --- |
| LD37 | Rewrite `ok`/`as`/`...`, delete relay/spread forms, or globally change bare-name pattern binding | **retain current design** | Keep the current construction, narrowing, binding and forwarding distinctions. A broad vocabulary rewrite has no demonstrated benefit. Exact generic patterns are handled by LD17; confusing context should receive precise diagnostics under LD44. |
| LD38 | Unify receivers with `near`, delete `near`, or replace methods with capture records | **retain current design** | Keep receivers, named `near` captures and ordinary arguments as distinct roles. Their ownership/capture semantics are not interchangeable. |
| LD39 | Explicit capture bindings at `callable` creation | **defer with a reason** | Preserve current exact-name capture behavior for now. Reopen with a concrete dependency-edit failure and an explicit-binding prototype that improves on dependency visibility through tooling without weakening ownership checks. |
| LD40 | Parenthesized callable-array types | **defer with a reason** | `emits [][]` is visually awkward but has defined precedence. Reopen with measured parse/authoring errors and a complete grammar comparison, including arrays of callables versus callable array results. |
| LD41 | Unify error/record field layouts and collection API homes | **defer with a reason** | These are two independent consistency candidates without demonstrated behavioral payoff. Reopen each with representative authoring evidence during catalogue/formatting review; native operations remain the implementation basis. |
| LD42 | Allow fully effectful top-level initialization | **retain current design** | Keep inert initialization and explicit startup. Source order alone does not define cross-file effects, cycles, fixtures, failure handling or resource ownership. |
| LD43 | Raw-string escaped-quote exception or removal of single-line raw strings | **retain current design** | Keep the four forms and literal raw semantics. Existing triple-raw strings address embedded quotes without introducing an exceptional escape rule. |
| LD44 | Imprecise semantic spans, unclear obligations and unsafe/missing fix suggestions | **implement** | Carry precise spans and obligation provenance for missing arms, outward errors, fixtures and inference. Explain coordination recovery ownership and captured dependencies at relevant diagnostics/tooling surfaces. Suggested edits must be compiler-validated; terminology changes alone do not solve this. |
| LD45 | No safe canonical source formatter | **implement** | Extend the existing renderer into comment-preserving source formatting with an appropriate command. Preserve comments/trivia and check semantic stability/idempotence. Correct the claim that no renderer exists; `parse --render` already uses one. |
| LD46 | Multiline calls, bounds, arrays and assertions | **defer with a reason** | First measure remaining line pressure after normalization and safe formatting. Reopen with representative long forms and a consistent delimiter/layout proposal, without detaching assertions. |
| LD47 | Inaccurate inference, numeric, configuration and implementation-status descriptions | **implement** | Document finite generic equality inference, expected literal typing, leaf/narrower-variant inclusion, callable error subsets, invariant containers and explicit numeric conversion. Include surprising native numeric/string cases. Correct “JSON Schema validated” to the actual compiler validation mechanism, and distinguish build, assertion execution and renderer behavior. Do not repeat the false replacement claim that existing variant values never enter wider variants. |
| LD48 | Redesign primitive semantics or weaken core nominal/explicit contracts | **retain current design** | Preserve exact integers, explicit floats/conversions, current native-backed equality/string/collection behavior, immutability, nominal types, exhaustive matches, mandatory public contracts and named functions. Explain surprising cases instead of introducing gratuitous host divergence. |
| LD49 | Unmeasured verbosity and speculative ergonomics claims | **implement** | Evaluate equivalent behavior and test coverage, including AI authoring errors and edit propagation. Use the unchanged eight-fetch structure to count declarations/arms and prove selective recovery. The review's 63→9 entries and 56→8 arms are sketch measurements, not executable acceptance evidence or model-token counts. No arbitrary Can/TypeScript ratio is an acceptance threshold. |

## Coverage of every source finding

Sources: [R: redundancy review](../syntax-taste/redundancy-review-2026-09-22.md), [F: design review](../syntax-taste/design-review-2026-09-22.md), [DR: confirmed requirements and additional candidates](design-revisions.md), [Full: full language review](../syntax-taste/full-language-review-2026-09-22.md), [BA: before/after](../syntax-taste/language-design-before-after-2026-09-22.md). This mapping includes rejected alternatives, not only favored recommendations.

| Source finding | Disposition IDs covering all parts |
| --- | --- |
| R1 A; B section order, Choice layout and shape header | LD01, LD02 |
| R2 A and B | LD03 |
| R3 A and B | LD04 |
| R4 A and B | LD19, LD44 |
| R5 A and B | LD37, LD11, LD44 |
| R6 A and B; adapter cost | LD05, LD06 |
| R7 A and B; standard channel and aggregate specialization | LD16, LD15, LD17, LD18 |
| R8 A and B; repeated public obligations | LD13, LD12, LD07 |
| R9 A and B, including weaker argument checks | LD26, LD27, LD28 |
| R10 A and B; hidden dependencies | LD38, LD39, LD44 |
| R11 field layout | LD41 |
| R11 bare-name patterns | LD37, LD17 |
| R11 collection API homes | LD41 |
| R11 top-level initialization | LD42 |
| R11 string forms | LD43 |
| R “What is not redundant”; F strengths | LD01, LD04, LD08, LD26, LD48; baseline above |
| F1 closed boundary, process, declarative extensions, proposed priority | LD32, LD33, LD34 |
| F2 authored native assertions and standard catches | LD25, LD15 |
| F3 chain/nesting and halting traversal | LD19, LD20, LD21 |
| F4 runtime ownership and static escapes | LD30, LD31 |
| F5 hanging assertions and empty races | LD22, LD23, LD24 |
| F6 verbosity, aliases, fixture reuse, long lines | LD49, LD13, LD27, LD46 |
| F7 inference/coercion claims | LD47, LD48 |
| F8 JSON validation, source descriptors, registries | LD35, LD36, LD47 |
| F9 formatter, layout and diagnostics | LD45, LD46, LD44 |
| DR seven errors, detail/recovery, propagation and fixtures | LD07, LD12, LD25, LD49 |
| DR judge-only boundary, derivation/overrides, open multiple bases | LD08, LD09, LD10 |
| DR errors-first order and compiler enforcement | LD11 |
| DR eight-call benchmark and nesting exclusion | LD49, LD20 |
| DR additional: error enumeration | LD07, LD12, LD13 |
| DR additional: aggregate conversions | LD17, LD18, LD16 |
| DR additional: callable arrays | LD40 |
| DR additional: captures | LD38, LD39, LD44 |
| DR additional: division-by-zero checks | LD29 |
| DR interpretation limits: regression complexity, recursion, existing chain | LD06, LD14, LD19, LD20, LD21, LD49 |
| Full §1 normalization and payload alternatives | LD07, LD08, LD10, LD12 |
| Full §2 derivation, delegation, signature/bounds and bases | LD09, LD10, LD12 |
| Full §3 fetch measurements and match order | LD49, LD11 |
| Full §4 build and assertion liveness | LD22, LD23, LD24 |
| Full §5 snapshots | LD15, LD16 |
| Full §6 generic discrimination and aggregate mapping | LD17, LD18, LD16 |
| Full §7 coordination | LD19, LD44 |
| Full §8 native callables, grouped and flat alternatives | LD05, LD06 |
| Full §9 authored higher-order error parameters | LD14 |
| Full §10 captures, receivers and callable arrays | LD38, LD39, LD40, LD44 |
| Full §11 native testing, fixtures and runtime checks | LD25, LD26, LD27, LD28, LD29 |
| Full §12 scoped resources | LD30, LD31 |
| Full §13 platform boundary, IDs and JSON | LD32, LD33, LD34, LD35, LD36, LD47 |
| Full §14 primitives, initialization, traversal, tooling and small cleanup | LD01, LD02, LD21, LD41, LD42, LD43, LD44, LD45, LD46, LD47, LD48 |
| BA §1–4 normalization, details, wrappers and judge | LD07, LD08, LD09, LD10, LD12 |
| BA §5 match order | LD11 |
| BA §6 standard failures | LD15 |
| BA §7 generic failures | LD17, LD18 |
| BA §8 native callable alternatives | LD05, LD06 |
| BA §9 captures and callable arrays | LD38, LD39, LD40 |
| BA §10 error parameters | LD14 |
| BA §11 fixtures | LD26, LD27, LD28 |
| BA §12 native tests | LD25 |
| BA §13 build/assertions | LD22, LD23, LD24 |
| BA §14 runtime checks | LD29 |
| BA §15 resources, IDs and tooling | LD30, LD31, LD36, LD44, LD45, LD46 |
| BA retained designs and status; identity reassessment | LD01, LD03, LD04, LD05, LD08, LD14, LD19, LD26, LD27, LD38, LD42, LD43, LD48 |

The earlier suggested sequences are superseded by these dispositions. In particular, native-form unification, flattened state, fixture hoisting and platform extension are not prerequisites for the accepted repairs. Deferred entries stay visible here; they must not silently become implementation tasks or disappear from future scope reviews.

## Evidence and design follow-up

[Acceptance evidence for all accepted changes](language-change-acceptance-2026-09-22.md) defines before/after, valid/rejected programs, runtime, fixture and native-lowering requirements for every `implement` row. Its coverage validation is not a claim that the changes pass; LD29 now has a selected C9.2 contract but still needs implementation evidence.

[Targeted implementation verification](implementation-gap-verification-2026-09-22.md) now supplies executable evidence for LD22/LD23, semantic locations under LD44, and resource admission/enforcement under LD30/LD31. It distinguishes missing build guarantees and runner feedback from intentional production and resource semantics; no deferred design is promoted by these checks alone.

This is a consolidation of reviewed alternatives under the user's explicit identity baseline, not a new syntax selection. The [full-review consultation record](../syntax-taste/evidence/2026-09-22/full-language-review/README.md) contains three freshly reworded Jev packets for the difficult alternatives and reconciles disagreements. The [identity reassessment](../syntax-taste/evidence/2026-09-22/identity-review/README.md) contains three further saved consultations supporting focused repairs while retaining grouped state and `near`. Their agreement is advice, not proof. No new consultations were made solely to classify duplicate findings in this ledger. Any newly selected difficult contract must still follow the repository's three-consultation rule.

The user subsequently requested completion of the main behavior designs. Those selected contracts are now owned by C/A/Q/P and located through the [behavior-contract index](language-behavior-contracts-2026-09-22.md), which closes the gates it explicitly lists and reopens LD27 only. The original gate inventory was: normalization payload/origin and fixture rules; wrapper dispatch, parent delegation and visible bounds; match enforcement; snapshot and generic-pattern semantics; assertion publication/deadline/native-coverage rules; and failed-check behavior. For each, attach a valid before/after example, rejected-program cases and native lowering/verification evidence. Reproduce suspected defects against the current code before calling them confirmed. The [reconciliation record](../syntax-taste/evidence/2026-09-22/document-reconciliation/README.md) now maps these contracts into the authoritative specifications. Derive dependency-ordered tasks from those specifications and the acceptance records once remaining design gates are closed. The new contracts do not reopen retained designs or automatically commission other deferred prototypes. LD29 was subsequently closed by [three fresh consultations](../syntax-taste/evidence/2026-09-22/ld29-checks/README.md) and the C9.2 contract. All accepted repairs still require their implementation evidence; no deferred abstraction was promoted.
