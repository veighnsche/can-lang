# Native Can testing: reference, candidate and cleanup prerequisites

Status: **available inputs identified; prerequisites for integrated Can-owned qualification are not yet ready**.

**Sequence correction:** the initial handoff below overreached by making implemented, qualified R/N universal prerequisites for feasibility research. They remain requirements for integrated Can-owned qualification. A separately scoped mechanics experiment may use existing identified tools, independent fixed/raw observations and a bounded external cleanup parent to investigate a specific boundary. Its result cannot qualify R/N or replace a Can test. The overall sequence remains source/design → minimal experiments where needed → design reconciliation → ordered implementation plan. No experiment has been run or authorized by this correction.

The source/evidence inventory identifies the current candidate and verifies existing Bun inputs. It does **not** find an accepted reference compiler distribution or an implemented, qualified generic external cleanup owner. Those roles remain unassigned at the executable level. A mechanics probe must report that trust limit; it cannot infer accepted R/N roles from the candidate checkout or a successful candidate report.

The [machine-readable prerequisite record](preparation/native-can-test-prerequisites-2026-09-30/prerequisites.json) records `is_acceptance_receipt=false`, `execution_admission.ready=false`, empty admitted experiments/capabilities, and null reference/owner executables. This admission field concerns integrated Can-owned execution. Mechanics-probe admission is recorded separately. The four [scoped](native-can-test-preparatory-checks-2026-09-30.md) jobs have now run: PM-N1/N2/F1 supported narrow mechanics, and PM-B1 failed browser isolation despite route protocol support. Their [results](native-can-test-preparatory-results-2026-09-30.md) and [design reconciliation](native-can-test-design-review-2026-09-30.md) do not change integrated admission. Further use of the failed browser launch recipe is blocked. This is an audit artifact, not implemented launcher enforcement.

The inventory below is the earlier observation at `288a38200cd779ff027377cf34f528a782740d54`. Later preparatory scoping observed HEAD `4cf8f91d97f45cf937c3464de54bc6275e70d49f` and saved [separate input identities](preparation/native-can-test-preparatory-checks-2026-09-30/inputs.json). Preserve both scopes; do not reuse the earlier source identity as a current executable or lease.

## 1. Concrete inventory

| Role/input | Exact selection or observation | State |
| --- | --- | --- |
| **R — qualified reference** | No matching accepted distribution inventoried; executable and acceptance receipt remain null | **Missing.** Historical reports and byte hashes do not establish this role |
| **C — candidate source** | `/Users/vince/Projects/can-lang`, commit `288a38200cd779ff027377cf34f528a782740d54`, tree `a48981794ca55b326f1c1e49dbfdd4c10e15dcfe` | Identified. Product-source paths are clean; existing design documents and `AGENTS.md` changes are outside this source selection |
| **C — Bun subject runtime** | `/Users/vince/Projects/can-lang/.local-deps/pinned/bun-darwin-aarch64/bun` | Existing bytes match pinned Bun 1.4.2 darwin-arm64; not executed in this audit |
| **Bun build input** | `/Users/vince/Projects/can-lang/.local-deps/bun-darwin-aarch64.zip` | Archive and streamed executable member match both pins; no extraction/download needed |
| **C — compiler distribution/generated subjects** | No executable compiler bundle in inspected locations; no selected generated subjects | Not materialized. Source commit is not an executable/output digest |
| **N — external cleanup owner** | Previously selected architecture: separately identified trusted Go process outside Can workers and C subjects | **Executable missing.** Existing mechanisms do not provide the full ownership/receipt contract |
| Bootstrap Go input | `/opt/homebrew/bin/go` resolves to `/opt/homebrew/Cellar/go/1.27.1/libexec/bin/go` | Bytes inventoried; version inferred from installed path, not executed or behavior-qualified. CGo compiler/SDK and module availability remain unqualified |
| Host | Darwin kernel `27.0.0`, `arm64` | Platform observed; containment, quotas and live-service readiness not qualified |

Verified Bun identities:

- Runtime SHA-256: `35d20dd0263e5c950194434b925454fdfa9ba6e4467da960410fa05b08a7a5b5`.
- Archive SHA-256: `90987a3a16d7db556d886ac3d551e7b6d3edf0a1cf43acaed622e8676be1d12f`.
- Target: `bun-1.4.2-darwin-arm64-v1`; revision declared by the matching [target manifest](../../distribution/target.json): `744846f844374847c902b5e7fd59b4342a51ef99`.

The ambient `/Users/vince/.bun/bin/bun` also matches, but future execution should select the explicit recorded artifact, revalidate it and hold the required lease/immutability guarantee. `CAN_BUN_ARCHIVE` is unset; the archive is available by explicit path. No `canlc` is on PATH. `dist/development` contains no bundle; inspected standard user installation roots and repository `bin/build/out` contain no compiler. This was a bounded local inventory, not a claim that no accepted distribution exists on any disk or remote host. Other tasks' worktrees were not inspected or repurposed.

## 2. Why the existing evidence cannot supply R yet

Five saved native-capability reports identify the same Bun executable hash and report passing historical checks. None matches **both** the current target manifest and native probe source hashes. Their subject is the native runtime; they do not qualify a Can compiler or the future runner. The [reference audit](preparation/native-can-test-prerequisites-2026-09-30/reference-audit.md) distinguishes historical I45 release/package evidence and later source reviews from an exact R acceptance record.

| Current input | SHA-256 |
| --- | --- |
| `distribution/target.json` | `79a61045c07d345254767554ce0a5c1f31c48129b94a425183379a3e2e68f6df` |
| `tests/conformance/native.ts` | `e1af4ae127651b10d08b4be6cf18d67b3bdda5403c8a51972053be1c0d380f35` |

The [grouped-errors completion record](grouped-errors-and-labels-completion-record-2026-09-30.md) supplies useful recent compiler evidence, including bounded package/execution checks and independent review. It does not identify an installed compiler distribution, qualify the new live-suite path or prove N's cleanup. Preserve that scope.

An accepted R record needs:

1. Exact compiler distribution/executable, catalogue, private runtime, Bun and trusted binding identities, source/build provenance and host scope.
2. Reviewed independent fixed/native observations and seeded failure controls for the compiler/runtime capabilities relied on by the judge. Historical evidence contributes only where source/artifact and obligation correspondence is demonstrated.
3. A separately reviewed suite/expected-vector snapshot and, before accepting the first full runner, wrong-observation, early-`ok`, missing-check, malformed-report, worker/controller-death, timeout and cleanup-failure controls.
4. Explicit limits and recorded promotion. Hash-valid packaging and candidate self-tests cannot silently perform it.

R and C may later happen to share bytes at a baseline. Their roles, acceptance records and execution authority remain separate. New candidate builds cannot replace R during a run. Ordinary Can helpers remain unrestricted under normal effects/targets; this introduces neither a testing DSL nor a second interpreter.

## 3. Candidate identity includes actual materialized inputs

The recorded candidate commit is not an immutable source lease or an executable identity. Revalidate selected inputs before future materialization, then bind compiler bundles and generated subjects by their own manifests/digests. Record compiler, executor and observer separately.

[`distribution.Build`](../../distribution/build.go) walks every regular file in `runtime`, `tools/runtime`, `distribution/assets` and `distribution/notices`. Those directories currently contain three ignored `.DS_Store` files. A HEAD-only identity omits bytes that this builder copies.

The [asset inventory](preparation/native-can-test-prerequisites-2026-09-30/candidate-assets.json) records all **336 observed files, 4,092,369 bytes**, including those ignored files, without copying or changing them. A later build must fix its selected input set, record the resulting manifest and invalidate on drift. Removing OS metadata is a change to build inputs; do not silently label the new bundle identical to this inventory. No metadata or shared cache was deleted here.

N1 needs the recorded raw Bun input; N2 also needs a materialized C compiler/subject and reviewed same-realm ingress. Neither obtains a trusted judge through candidate `canlc run`, which builds using its own compiler. The ordinary-Can nonpublishing live-probe path remains implementation work.

## 4. External owner selection and limits

The **selected owner architecture is N: a trusted generic Go process external to all Can workers and candidate subjects**. It needs its own executable digest, source/build provenance, capability record and qualification receipt. No complete executable was found in the audited source/tool inventory.

The [cleanup audit](preparation/native-can-test-prerequisites-2026-09-30/cleanup-audit.md) identifies useful pieces:

| Existing mechanism | Useful behavior | Missing for N |
| --- | --- | --- |
| Go assertion supervisor | Fresh workers, deadlines, direct-worker kill/reap, outcome classification | General live-case/resource protocol, descendant ownership and matching cleanup receipt |
| Go entry launcher | Sanitized Bun startup, fd 3 pipe, generation lease, non-reader writer handling | Durable pre-effect ownership, cleanup failure reporting and separate CLI/direct-entry policies |
| `tempcache` | Marked/locked directories, identity checks, conservative recovery and deletion errors | Process/descriptor/browser/DB ownership outside workers and reservation before allocation |
| Performance Python owner | Process-session/group and scratch accounting | Generic Can capability API and qualified descendant containment; this is measurement-specific tooling |

N owns registration before effects, process/descriptor lifetime, bounded transport, scratch and later browser/DB resource services. Can decides scenario order, readiness, expectations and verdicts. N's receipt lists released resources, forced actions, unresolved effects and cleanup failures; an early Can `ok` cannot override it.

Before a worker-death probe, demonstrate its required host guarantee. Killing a group does not prove detached descendants cannot survive; disk polling is not a strict quota; a path/digest is not filesystem confinement. Missing required guarantees block that variant and cannot be relabeled a narrower successful run.

Reuse the Go mechanisms as inputs to the previously chosen owner design. `RunSupervised` cannot simply be renamed N, and the performance harness cannot become the permanent native test runner. No owner code was implemented here.

## 5. Implementation acceptance dependencies, not a replacement research sequence

The following P0–P5 list decomposes the finished system's bootstrap/acceptance work. It does not mean all these components must be implemented before investigating native mechanics or writing the implementation plan. Full Can integration checks carry the relevant dependencies into their implementation lanes.

| ID | Necessary result | Current state / acceptance evidence |
| --- | --- | --- |
| P0 — inventory | Separate R/C/N roles and exact existing input identities, absent artifacts explicit | **Complete as an audit.** JSON, asset list and source reviews saved; admission remains false |
| P1 — bootstrap owner | Separately built Go N for the first required process/scratch/descriptor guarantees | Missing implementation/receipt. Qualify native mechanics with bounded independent bootstrap controls and pre-registered external cleanup; existing host checks may contribute transitional evidence |
| P2 — reference seed | Accepted existing bundle, or separately built/reviewed seed with explicit source/bootstrap inputs and promotion | No accepted artifact found. Verified archive/Bun are reusable inputs; source HEAD and historical test totals are not acceptance |
| P3 — probe entry/suite trust | R checks ordinary Can probe/helper source and required offline roots; bounded nonpublishing entry, typed transport and reviewed expected vectors | Missing live-entry/bindings. Implement only what the selected experiment needs; unrelated browser/DB surfaces need not come first |
| P4 — candidate materialization | C bundle/output identities, copied input manifest and per-probe observer/runtime roles | Source/native inputs known; compiler/subject manifests absent. N1 targets raw Bun; N2 and CLI F1 require relevant C artifacts |
| P5 — admission record | Matching R/N receipts, selected C/S identities, demonstrated host guarantees, finite budgets and cleanup plan | No integrated Can qualification until its prerequisites are present. Separately scoped mechanics evidence cannot change this authority |

This avoids a bootstrap loop: qualify the first owner/seed through an explicit reviewed host bootstrap boundary, then use accepted R/N for ordinary Can probes. Transitional host checks supply evidence for that boundary; they do not become a hidden permanent replacement suite. The unimplemented full Can runner cannot be the sole oracle proving its own first owner/reference. The first complete runner still needs separate negative controls before promotion.

For P1, name a separate **host bootstrap parent/witness T** in the bounded job definition before launching anything. T is outside the tested owner's/worker's killable subtree; its executable and any reviewed bootstrap script have their own identities. T records the owned process start identities, descriptors and scratch authority outside the child before effects, retains finite stop/reap and ownership-safe recovery authority, and independently checks N's receipt against actual release facts. The initial fixtures use an explicit bounded process/resource graph; they cannot claim containment of arbitrary detached descendants without a demonstrated enforcing mechanism. Keep T alive in owner/worker-death controls. If T itself disappears, the job has no successful terminal receipt: retain its compact ownership record and require recovery/inactivity proof before further admission. Do not intentionally kill T until a separately scoped recovery test provides another intact observer. T is transitional bootstrap machinery, not an already available N or permanent foreign suite; its exact implementation/identity remains part of P1's unfulfilled job definition.

P1/P2 are implementation/qualification work to include in the ordered implementation plan. The scoped mechanics questions have been executed and reconciled. The next implementation plan must keep their remaining Can integration qualification as explicit dependencies. Before executing either, define exact tools/inputs, independent observations, deadline including cleanup, process/memory/scratch ceilings, ownership/recovery and controls. A mechanics spike must not require construction of the complete architecture it is supposed to assess. Broad builds, live-service starts and measurements remain deferred; this document runs none.

Use the [feasibility register](preparation/native-can-test-hard-cases-2026-09-30/experiments.md) for the remaining integrated F1/N1/N2/B1 acceptance and specialized binding gates. Preserve each evidence scope when producing implementation lanes containing R/N bootstrap, integration acceptance and migration dependencies. Planning does not require accepting those unimplemented components in advance.

## 6. Evidence and verification scope

This round read source/reports, inspected bounded known locations, hashed files and streamed the pinned archive member through a hash without extraction. It started no candidate executable, compiler build, test, browser, database, measurement or remote service. No temporary execution directory, checkout, bundle copy or private build cache was created. Existing tool inputs and unrelated worktrees/output remain in place.

This applies the settled R/C/N trust design; it makes no new backend or trust-promotion choice requiring another Jev consultation. Earlier consultations remain advice supporting that design, never acceptance evidence for missing artifacts.

Evidence: [reference audit](preparation/native-can-test-prerequisites-2026-09-30/reference-audit.md), [cleanup audit](preparation/native-can-test-prerequisites-2026-09-30/cleanup-audit.md), [prerequisite record](preparation/native-can-test-prerequisites-2026-09-30/prerequisites.json), [review record](preparation/native-can-test-prerequisites-2026-09-30/review.md), [validation](preparation/native-can-test-prerequisites-2026-09-30/validation.json). These establish the inventory and remaining gates, not executable readiness.
