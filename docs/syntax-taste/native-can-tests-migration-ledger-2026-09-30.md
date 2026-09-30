# Native Can tests: coverage migration ledger

Status: **coverage mapped; every replacement and retirement still pending**.
The [post-probe design review](native-can-test-design-review-2026-09-30.md)
preserves this ledger and adds the evidence-scope mapping below. No preparatory
observation advances a row's replacement or deletion status.
This is step 2 after the [completion contract](native-can-tests-completion-contract-2026-09-30.md).
It is a source-backed migration ledger, not an implementation plan, a passing
test report or a choice of execution backend. No tests, builds, browser sessions,
database scenarios or performance measurements were run for this ledger.

Each entry records what the existing test protects, its current observations,
variants and environment gates, required capabilities, a proposed ordinary Can
replacement, and the evidence needed before removing its old harness. The
replacement must cover **all** listed facets; a large entry may become several
Can cases. `CAN-<row ID>` identifies that replacement obligation without fixing
a file layout, API or one-to-one test structure.

## Coverage and navigation

The ledger contains **292 entries**, covering all **146 Go test functions**, all
**58 textual Bun test declarations**, both `TestMain` functions, authored
scenario/support scripts, historical evidence and external callers. These are
source declaration counts, not executed case counts: loops, subtests and browser
profiles expand them. Several rows describe different layers of one obligation;
their counts must not be added to claim extra coverage.

| Ledger part | Entries | What it maps |
| --- | ---: | --- |
| [Compiler, language, CLI, distribution and runtime integration](preparation/native-can-tests-migration-ledger-2026-09-30/core.md) | 51 | 48 Go tests and raw-fixture support: diagnostics, assertions, production builds, standard library, native operations, packaging and delegated runtime semantics |
| [Applications and browsers](preparation/native-can-tests-migration-ledger-2026-09-30/browser.md) | 56 | 34 Go tests plus 22 browser/setup scripts: actual compiled clients, HTTP/UI behavior, independent DOM facts, multi-engine observations and fault injection |
| [SQL, conventions, historical checks and shared support](preparation/native-can-tests-migration-ledger-2026-09-30/sql-history.md) | 143 | 41 Go tests, all 58 Bun declarations, SQL drivers, native qualification, cache safety, prototypes and historical evidence |
| [Service lifecycle, replay, crashes, storage and resources](preparation/native-can-tests-migration-ledger-2026-09-30/lifecycle.md) | 26 | 23 Go tests plus shared harness/driver entries: concurrency, cancellation, replay, shutdown, companion workers, S3 and bounded resource use |
| [External callers and release/CI connections](preparation/native-can-tests-migration-ledger-2026-09-30/external-callers.md) | 16 | Qualification, Linux scripts, CI, host conformance consumers and operator instructions |

Supporting maps:

- [Every tracked `/tests` file](preparation/native-can-tests-migration-ledger-2026-09-30/file-map.md): all **224 files** have a role and a route to the ledger. The [JSON](preparation/native-can-tests-migration-ledger-2026-09-30/file-map.json) distinguishes direct rows, cited dependencies and family context; family context does not claim a direct call.
- [Delegated runtime oracle checklist](preparation/native-can-tests-migration-ledger-2026-09-30/delegated-oracles.md): **30 explicitly invoked runtime Bun suites, with 243 textual test declarations**, linked to their parent rows. This is a subordinate parity checklist, not 243 completed semantic migrations. Browser wire-vector logic is additionally tracked by BROWSER-039/040.
- [Source inventory](preparation/native-can-tests-migration-ledger-2026-09-30/source-index.json): paths, hashes, declaration anchors, 40 textual Go subtest sites and lexical browser check/call sites. Browser call indexing includes helper/DOM calls and is not a count of distinct assertions.
- [Validation evidence](preparation/native-can-tests-migration-ledger-2026-09-30/validation.json): source and ledger consistency checks. Each detail page links its authored JSON rows; the [schema](preparation/native-can-tests-migration-ledger-2026-09-30/schema.json) defines fields and capability identifiers.

The `/tests` snapshot was captured at HEAD
`8f6e884f3d57f88b2959a7e52d9d86c6e6175afd`; all 224 captured file hashes still
matched at validation. External runtime suites have their own captured hashes.
Reconcile changed sources before implementation and again before deletion.
This source audit makes no claim that the current suite passes.

## Shared requirements exposed by the mapping

These are capability families, **not a proposal for one new language primitive
per kind of test**. Ordinary Can helper functions should compose them. The
existing capability investigation describes which operations already exist;
this ledger records what a complete migration must be able to express.

| Shared requirement | Capability IDs | Examples and design consequence |
| --- | --- | --- |
| Can-owned registration, selection, sequencing, comparisons and reporting | `SUITE`, `CAN` | Compile/assert/build checks, live scenarios and browser cases need one consistent policy layer. Existing attached assertions remain useful for deterministic logic but cannot supply live effects. |
| Processes, structured observations and owned workspaces | `PROC`, `FILES`, `WORK`, `BUILD`, `DIAG` | Most families create inputs, invoke a production candidate, inspect outputs and clean up. Reuse a build within a run and preserve candidate provenance. Separate diagnostic bytes, exit status and report scope. |
| Stateful process and resource lifetime | `CHILD`, `CLOCK`, `ENV`, `FAULT` | Descriptor 3 credentials, readiness, signals, crash/restart, hung workers, descendants and resource cleanup recur across service tests. Generic supervision must survive failure of the Can controller. |
| Independent native observations and controlled hostile inputs | `NATIVE`, `FAULT` | Forged resources, getters/proxies, late failures, exact occurrence identity, raw numeric values and native delegation cannot all be checked by round-tripping the same Can adapter. The safe observation/fault boundary needs design. |
| Normal HTTP plus deliberately broken transport and controlled peers | `HTTP`, `WIRE`, `PEER` | Status/body/header checks share helpers. Trickle input, disconnects, truncated replies, provider stubs and gated sends need lower-level mechanics and Can-authored schedules. |
| Browser actions and independent DOM/event/network observations | `BROWSER`, `HTTP`, `NATIVE` | Multiple engines, remote Firefox contexts, interception, composition/caret events, delayed replies and page/server rendezvous recur. Moving test-specific `page.evaluate` callbacks into an adapter would preserve a host harness. |
| Independent database state and object storage | `DB`, `STORE` | PostgreSQL, SQLite and MySQL setup/inspection must expose raw facts so Can owns transactions, fault sequences and expectations. S3 needs prefix ownership and failure-visible cleanup before any upload. |
| Packaging, qualification and maintained evidence | `ARCHIVE`, `ENV`, `DOC`, `FMT` | Installation, integrity, isolation, provenance, generated-artifact freshness, formatting and current documentation need explicit obligations. Old paths, output layouts and dated wording are not compatibility requirements. |
| Authorized resource qualification | `METRIC`, `CLOCK`, `WORK` | The companion's wall/RSS guards are real existing checks; throughput is logged, not gated. Preserve or explicitly re-scope the obligation. This ledger does not authorize measurements now. |

The subsequent [shared capability contracts](native-can-test-capabilities-2026-09-30.md)
define browser actions/interception, independent SQL/native observations,
hostile-value construction and externally owned lifetimes. The ledger identifies
their consumers; concrete bindings, host enforcement and coverage proof remain
pending. Specifying a capability does not mean Can already exposes it or that
any row is ready for retirement.

The [lifecycle contract](native-can-test-lifecycle-2026-09-30.md) now fixes failure
accounting, missing-environment behavior, complete versus partial qualification,
cancellation/recovery, concurrency and same-run build reuse for every replacement.
Its policy ceilings and future negative controls are not execution evidence.

## Consolidation and obsolescence candidates

No check is declared obsolete solely because its directory or file is old.
These are evidence-backed review candidates, with conditional dispositions in
their individual rows.

| Candidate | Ledger anchors | Proposed treatment |
| --- | --- | --- |
| Repeated build/staging and resource scaffolding | LIFE-016–024; HISTORY-034–041; CORE-049–051 | Consolidate reusable mechanics and Can policy. Preserve key sensitivity, contamination refusal, partial-fill recovery, concurrency bounds and foreign/active-path protection. Update host conformance consumers before deleting shared support. |
| Repeated browser/service observations in verdict gates | BROWSER rows; LIFE-015 | Reuse observations only when candidate, engine/profile and coverage provenance match. A shorter report must still require every named leg and failed/missing evidence must fail closed. |
| Generated-artifact freshness checked twice | HISTORY-047 and HISTORY-052 | One current Can check may cover both obligations, retaining meaningful catalogue/generated-output validation without old marker or file-layout compatibility. |
| Linux qualification repeated through shell and Go | LIFE-012/013; EDGE-002–005 | One qualified installed-artifact profile may replace duplicated scenario policy. Keep independent DB observations, offline/native qualification and target provenance. |
| Frozen baseline and dated release recommendations | HISTORY-042/043/045/046/048–053 | Separate useful current behavioral/doc checks from old freeze, exact wording, historical IDs and exclusions. Preserve records; require a current-contract relevance decision for each retired assertion. |
| Prototype adapter/companion experiments and bundle sizes | HISTORY-064–126 | Preserve comparison evidence or migrate retained behavior as explicitly scoped prototype tests. Current production non-admission and review-manifest checks (HISTORY-061–063) remain separate obligations. |
| Old grid bundler and SHA-256 shim | BROWSER-037/053 | No active tracked integration Go caller was found; the T27 source-presence check and historical evidence still reference them. Resolve those dependencies before retirement. |
| Human-readable pass counts and emitted-layout pins | Core delegated-suite rows; source checklist | Review as harness drift, not desired semantics. For example, the collections wrapper expects `4 pass` while its current runtime source has 10 test declarations; arrays expects `10 pass` while its source has 15 textual sites plus dynamic variants. No execution failure is claimed here. |

The failure-convention experiments contain current negative/positive compiler
and runtime checks; their origin as experiments is not a deletion reason.
Static Can, JSON, SQL, HTML and binary inputs may remain as fixtures. Authored
drivers that sequence operations or decide outcomes must be replaced, even if
they currently live under `testdata`.

Two scope edges require care:

- HISTORY-143 and EDGE-001 are linked views of the same qualification dependency. `native.ts` owns test policy; the Python caller adds isolation, provenance and aggregation policy. Both must be accounted for, without counting them as separate coverage wins.
- The baseline shell's full mode launches broad repository Go and Bun gates. Independent compiler/runtime unit suites are outside automatic migration scope and may remain independent CI gates. A Can entrypoint invoking those old suites does not establish Can-owned coverage. Explicitly delegated in-scope oracle suites are tracked above; resolve the old meta-runner's role rather than silently expanding scope to the whole repository or crediting delegation as migration.

## Gaps the replacement must address

The S3 test attempts deferred deletion, but registers cleanup after seeding and
suppresses cleanup failures. Its final empty listing protects only the happy
path. LIFE-014 records the distinction: immediate prefix ownership, recovery
after failure/interruption, and reporting unresolved cleanup are stronger
acceptance conditions from the completion contract, not claims about today's
harness.

The companion's 200-row leg requires both workers to receive first-round work,
finishes within at most ten rounds, and checks each ID once in report steps,
downstream receipts, ledger effects and delivered attempts. Its crash leg has
different, intentional duplicate-delivery expectations. Those distinctions stay
separate in LIFE-011.

Optional services and engines currently cause skips or partial runs in several
families. A replacement must preserve the qualified scope explicitly. An absent
database, browser, OpenSSL or native qualification leg cannot be converted into
full success. Existing report substring/count checks should become Can-owned
structured evidence checks with missing/failed-evidence controls.

## Common gate before deleting any old harness

### Preparatory evidence is not replacement coverage

| Evidence | Relevant consumers/facets | Retained acceptance still required |
| --- | --- | --- |
| PM-N1: one inert getter/alias path and contamination control | NATIVE/FAULT, including HISTORY-143 / EDGE-001 and delegated runtime oracles | Full N1 value/control matrix, ordinary Can comparisons, identity in the owning realm and complete observation intervals |
| PM-N2: direct source `encodeJSON` call | Native codec/collection adapter facets in core/delegated rows | C-compiled ordinary Can ingress and generated wiring, exact adapter invocation, local completion authentication, R oracle and seeded C defect; direct source access substitutes for none of these |
| PM-F1: Python/POSIX → Bun descriptor behavior | ENV/CHILD launch, credentials and generation-lease facets | Go `ExtraFiles`, candidate product CLI hostile-input policy, repeated fresh launches, F1 Can assertions and F2 inherited lease lifetime |
| PM-B1: route rendezvous and deadlock observations, overall isolation failure | Browser held-save/replay rows and browser callers | Can-directed routing, independent input/contact/delivery/effect observations, complete body/event seals, required engines, service-death cleanup and admitted credential/host-UI isolation; durable replay additionally needs independent settled DB evidence |

Every replacement evidence entry must name its ledger row, facet, variant and
environment, and distinguish preparatory support from integrated qualification.
Source instruments stored with PM records may remain historical evidence but
must have no active CI/release/test invocation. Removing a host file requires
checking every current caller, including indirect/delegated and newly added ones.
Browser retirement requires the qualified host-effect interval and its controls;
PM-B1's repaired protocol receipt cannot override its later isolation failure.

### Shared deletion requirements

A row's specific `delete_when` condition is additional to all of these:

1. Every retained protected facet, dynamic case, named fixture, engine/service
   profile and delegated oracle has a concrete Can case/helper mapping. Any
   retirement has a recorded current-contract rationale and reviewer decision.
2. Ordinary Can owns scenario actions, expectations, comparisons, selection,
   retries, scheduling and reporting. No foreign test script, embedded host
   callback or old suite supplies the migrated verdict under a new wrapper.
3. Evidence identifies the candidate, input/configuration, target and qualified
   scope; positive, negative and missing-evidence controls prove the oracle can
   detect the failures it claims to protect. Independent observations remain
   independent where round-trip testing could hide a shared defect.
4. Required environments have actually run. Skipped or unavailable legs are
   recorded as incomplete, never inferred from another engine or profile.
5. Time, output, concurrency and storage are bounded; owned resources are
   registered before use and cleaned on success, failure and interruption.
   Recovery preserves active/foreign work. Retain compact evidence by default;
   heavy retention needs an explicit expiry and reclamation.
6. CI, release, developer commands and shared consumers use the replacement.
   The final host dependency is removed or reduced to generic mechanics. Retire
   a shared file only after **every** remaining consumer is resolved.
7. Re-scan the current tree and reconcile additions/changes against this ledger.
   Declaration counts are a bookkeeping check, not proof of behavioral parity.

The [test authoring design](native-can-test-authoring-2026-09-30.md) now works
through five representative cases on paper, including mandatory offline roots,
live execution and reporting. The remaining design work must resolve and check
the shared capability boundaries against this ledger's hardest cases, then
produce ordered implementation lanes under the chosen
[execution architecture](native-can-test-architecture-2026-09-30.md). These unexecuted
examples do not establish replacement coverage or authorize harness deletion.
