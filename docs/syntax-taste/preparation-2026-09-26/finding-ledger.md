# Finding ledger — 2026-09-26 round (P01.4)

Stable IDs `F-Rxx-nn`. Each row: source → kind → destination topic.
Kinds: DEF defect, MIS missing capability, DES design choice, LIB library,
TOOL tooling, OPS operations/qualification, DOC documentation.
Disposition is decided in P07; this ledger only guarantees coverage.

## R01 Host integration (review §1)

| ID | Source | Kind | Concern / acceptance |
|---|---|---|---|
| F-R01-01 | Review §1 ¶1; manifest/assets/capability closure | DES | Closed catalogue: ordinary packages cannot add JS/npm bindings; browser build admits no arbitrary scripts. |
| F-R01-02 | Review §1 ¶3; Jev host split | DES | Adapter vs catalogue-addition vs companion for a genuinely required third-party widget. |
| F-R01-03 | Review §1 acceptance | MIS | Second vendor integration without vendor-specific compiler patch or hand-edited generated code; capability/lifecycle tests stay meaningful. |
| F-R01-04 | Prior P11 (deferred), DI-19 retained | DES | Reopening needs two concrete missing ops compared through both designs. |
| F-R01-05 | Prior O04 (fixed-origin retained) | DES | Literal origins + confined paths stay unless R11 decides otherwise. |

## R02 Browser controls and reusable UI (review §2 ¶1–2)

| ID | Source | Kind | Concern / acceptance |
|---|---|---|---|
| F-R02-01 | Review §2 ¶1; browser.ts:320; dom-semantics probe | MIS | Event snapshot lacks checked state, files, multiselect, modifiers, composition/IME. |
| F-R02-02 | Review §2 ¶1; setAttribute probe | MIS | No live value/checked property setter; attribute write after edit leaves live value unchanged. |
| F-R02-03 | Review §2 ¶2; browser.ts:451; grid rendering | LIB | Extract manual field callbacks/rebuild/focus repair into ordinary libraries. |
| F-R02-04 | Review §2 ¶2; append contract | DES | Nested widget ownership: cross-view children restricted except at app root. |
| F-R02-05 | Review §2 acceptance | MIS | Two apps share keyed editable table + field/error controls; normalization, reset/autofill, multiselect, upload, IME, caret/focus, reorder, disposal, late replies. |
| F-R02-06 | Prior P08/P10/P12 deferred; S03 fixed | DES/LIB | Richer fields need an all-Can workflow + bounded immutable projection + browser tests per field. |
| F-R02-07 | Prior O-observations on grid | — | Grid strengths preserved as constraints (drafts, versions, conflicts, disposal). |

## R03 Captured HTML routes (review §2 ¶3)

| ID | Source | Kind | Concern / acceptance |
|---|---|---|---|
| F-R03-01 | Review §2 ¶3; http.go:369; actions.go:315 | MIS | Plain routes reject captures; GET actions require JSON; captured HTML reads use query URLs. |
| F-R03-02 | Review §2 ¶3 | DES | Reuse capture/URL machinery for server-rendered reads; independent of SPA router. |

## R04 Budgets, cancellation, ownership (review §3)

| ID | Source | Kind | Concern / acceptance |
|---|---|---|---|
| F-R04-01 | Review §3 ¶1; race-drain probe | DES | Race selects winner; owning root stays pending until loser completes; request drains before response. |
| F-R04-02 | Review §3 ¶1; pool.ts:88; server.ts:289 | MIS | SQL/transaction awaits lack caller deadlines; server HTTP clients already have them. |
| F-R04-03 | Review §3 ¶2; action_bindings.go:374; action-json.ts:576 | MIS | Browser `action::request/post` expose no transport deadline/cancellation; projection supplies no signal. |
| F-R04-04 | Review §3 ¶3; server.test.ts:664 | DES | Enforceable native budgets + explicit request ownership; cooperative cancellation/supervised work evaluated; leases preserved; external-supervisor shutdown boundary documented. |
| F-R04-05 | Review §3 acceptance | OPS | Stalled SQL/headers/bodies; disconnect; overlap; SIGTERM; bounded behavior; escalation; no post-disposal use; honest unknown-write outcomes. |
| F-R04-06 | Prior P06/P07 deferred; DI-15/LD30 retained | DES | Changed ownership needs live hedge measurement + comparison evidence. |

## R05 Iteration and collections (review §4)

| ID | Source | Kind | Concern / acceptance |
|---|---|---|---|
| F-R05-01 | Review §4 ¶1; recurse-large-runtime-v3.log | MIS | Sync tail recursion overflows at 20k on Bun 1.4.2 (100 OK); no promised tail-call guarantee. |
| F-R05-02 | Review §4 ¶2 | DES | Iteration primitive vs proven self-tail lowering; preserve order, failure identity, fixtures, diagnostics. |
| F-R05-03 | Review §4 ¶2; runtime/collections | LIB/MIS | Repeated immutable map/set insertion is quadratic; native bulk construction/aggregation contracts needed. |
| F-R05-04 | Review §4 acceptance | OPS | 100k-step state machine, growing aggregation, bounded worker batch; measured stack/memory + diagnostics. |
| F-R05-05 | Prior P05/P18 deferred; LD21 deferred | DES | Reopening comparisons specified in dispositions. |

## R06 Generic failure composition (review §5 ¶1–2)

| ID | Source | Kind | Concern / acceptance |
|---|---|---|---|
| F-R06-01 | Review §5 ¶1; independent-core.md | MIS | Authored `emits` cannot quantify over error sets; native helpers preserve callback bounds. |
| F-R06-02 | Review §5 ¶1; result-data probe | DES | Result-data helper checked/emitted/executed 5 roots; proves representation, not retry count/sequence. |
| F-R06-03 | Review §5 ¶2 | DES | Make result-data concise first; evaluate finite error-set parameter only if adapters remain extensive. No broad effect system established. |
| F-R06-04 | Review §5 acceptance (a) | LIB | One helper reused with two unrelated callback error sets; add error to one caller without editing others; verify retry count/sequence. |
| F-R06-05 | Prior P04 deferred; DI-03/LD14 deferred | DES | Reopening: two-domain + wrapper-around-wrapper comparison. |

## R07 Owner values and assertion setup (review §5 ¶3)

| ID | Source | Kind | Concern / acceptance |
|---|---|---|---|
| F-R07-01 | Review §5 ¶3; declarations.go:259; assertions.go:87 | MIS | Legitimate fallible smart constructor cannot be called in assertion argument; row has no setup region. |
| F-R07-02 | Review §5 ¶3 | DES | Private no-domain-error fixture helper is valid; adds function + assertion obligation. No public sample API required. |
| F-R07-03 | Review §5 acceptance (b) | LIB/DES | Extract owner-accepting handler helper with useful tests + minimal plumbing; compare factories vs checked test-local setup. |
| F-R07-04 | Prior LD29 gate closed (no new assertion grammar) | DES | Setup-region syntax needs explicit user decision; does not inherit from LD29. |

## R08 Authoring policies and captures (review §6 ¶1)

| ID | Source | Kind | Concern / acceptance |
|---|---|---|---|
| F-R08-01 | Review §6 ¶1; completion_matches.go:444 | DES | Exhaustive Boolean `true`-before-`false` rejected (intentional policy). Review: allow both, format/lint canonical. |
| F-R08-02 | Review §6 ¶1; locals.go:84 | DES | Meaningful final local (`subtotal = price*quantity; ok subtotal`) rejected (intentional policy). Review: allow. |
| F-R08-03 | Review §6 ¶1; callables.go:127 | DES | `near` binds by callee parameter name in caller scope; library rename changes caller obligations. Review: explicit binding + safe rename. |
| F-R08-04 | Prior P13/P14/P15; user Boolean decision 2026-09-23 | DES | All three touch retained rules; P15 is an explicit user Retain. User choices required. |

## R09 Editor workflows (review §6 ¶2–3)

| ID | Source | Kind | Concern / acceptance |
|---|---|---|---|
| F-R09-01 | Review §6 ¶2; lsp.go:156 | TOOL | LSP advertises diagnostics + definition only; no completion/hover/references/rename/format. Formatter machinery exists. |
| F-R09-02 | Review §6 acceptance | TOOL | Extract/rename callback, change shared record, inspect contract, repair callers with normal editor tools. |
| F-R09-03 | F-R08-03 dependency | TOOL | Safe rename support interacts with `near` binding design. |

## R10 Relational breadth (review production-breadth ¶1)

| ID | Source | Kind | Concern / acceptance |
|---|---|---|---|
| F-R10-01 | Review breadth ¶1; sql_schema.go:32; cardinality.go:30 | MIS | Narrow supported SQL: scalar fields, bounded SELECT, mutations without RETURNING; operator-owned schema. |
| F-R10-02 | Review breadth ¶1 | MIS | JSON/timestamp/decimal need chosen encodings or codecs; minor-unit money already OK. |
| F-R10-03 | Review breadth ¶1 | OPS | Prove real PostgreSQL app: generated identities, precise values, nullable audit times, JSON, concurrent replay, migration recipe. |
| F-R10-04 | Review breadth ¶1 | DES | SQLite reread-after-conflict does not transfer to aborted PostgreSQL transactions. |
| F-R10-05 | Prior P19/P20 deferred; DI-14/LD35 retained | DES | RETURNING/schema-tooling reopening conditions recorded. |

## R11 Durable workers and outbound policy (review breadth ¶2)

| ID | Source | Kind | Concern / acceptance |
|---|---|---|---|
| F-R11-01 | Review breadth ¶2; webhook README:79 | MIS | Carrier + retry scheduling live outside Can; webhook shows storage only. |
| F-R11-02 | Review breadth ¶2 | MIS | Complete Can worker: bounded concurrency, claims across two workers, retries/backoff, poison jobs, crash recovery. |
| F-R11-03 | Review breadth ¶2 | DES | Fixed-origin HTTP needs explicit destination policy for tenant webhooks; dynamic WebSocket URLs already exist — do not overgeneralize. |
| F-R11-04 | Prior O04/O05 retained | DES | Fixed-origin + trusted-carrier envelope stay unless this round decides otherwise. |

## R12 Failure observation (review breadth ¶3)

| ID | Source | Kind | Concern / acceptance |
|---|---|---|---|
| F-R12-01 | Review breadth ¶3; server.ts:300; log.ts:25 | MIS | Unexpected handler completions → fixed 500 without default boundary reporting. |
| F-R12-02 | Review breadth ¶3 | DES | Redacted request-failure hook with correlation/source identity; reuse main/late-owner/browser reporting machinery. |
| F-R12-03 | Review breadth ¶3 | LIB | Authors can catch + log today; relying on every app adding wrappers is fragile. |

## R13 Deployment and support (review breadth ¶4)

| ID | Source | Kind | Concern / acceptance |
|---|---|---|---|
| F-R13-01 | Review breadth ¶4; manifest.go:196; linux/README; pairing.go:19 | OPS | Linux target exists (Debian 13+ amd64/glibc, pinned runtime, packaging, Docker smoke); review did not rerun it. |
| F-R13-02 | Review breadth ¶4 | OPS | Qualify paired server/browser deployment: credentials, migrations, rollout/rollback, retained assets. |
| F-R13-03 | Review breadth ¶4 | OPS | Already-open browser vs newly deployed server acceptance scenario. |
| F-R13-04 | Machine constraint | OPS | UP25 Linux-amd64 qualification deferred to the user's x86 machine; never x86-emulate or long full-tilt run on this MacBook Air. |

## R14 AI product qualification (review breadth ¶5)

| ID | Source | Kind | Concern / acceptance |
|---|---|---|---|
| F-R14-01 | Review breadth ¶5; responses.ts:81 | OPS | Shape guarantees implemented; prediction quality/cost/latency not established for SaaS features. |
| F-R14-02 | Review breadth ¶5 | OPS | Demonstrate tenant budgets/correlation + model-change evaluation. |
| F-R14-03 | Review breadth ¶5 | DES | Responses adapter narrow non-streaming by choice; expand only for an accepted requirement. |

## R15 S3 cancellation and deadlines (review concrete problems 1–2)

| ID | Source | Kind | Concern / acceptance |
|---|---|---|---|
| F-R15-01 | Review problem 1; catalogue.json:12399; s3.ts:204 | DEF | **Contract discrepancy (data-loss implications):** catalogue says cancel releases sink without completing and never deletes key; cleanup calls `sink.end()` + `file(key).delete()`, removing pre-existing objects. Source-confirmed; no live deletion experiment performed. |
| F-R15-02 | Review problem 1 | OPS | Test cancel-while-replacing existing key, observer visibility, service failures. Absence of a new test key proves nothing about preservation. |
| F-R15-03 | Review problem 2; s3.ts:413 | DEF | Stream deadline checks only between awaits; cannot bound never-settling reader/writer/flush/completion/metadata ops. |
| F-R15-04 | Review problem 2 | OPS | Qualification needs stalled-await test, not only slow finite producer. |

## R16 Examples and documentation (review concrete problems 3–4)

| ID | Source | Kind | Concern / acceptance |
|---|---|---|---|
| F-R16-01 | Review problem 3; web.can:502; regions.go:575 | DEF | Invoice startup calls `env::required("")` to synthesize failure; claims only catalogue calls originate errors. Authored completions already work. |
| F-R16-02 | Review problem 4; webhook README:92 | DOC | Top-level Linux/browser claims contradict implementation (partially corrected post-reconciliation; re-verify). |
| F-R16-03 | Review problem 4 | DOC | Webhook omits constant-time comparison + carrier authorization: scoped demo, not internet-facing template. Bound clearly. |
| F-R16-04 | Review "what to build next" | DOC | One consistent supported story tied to selected contracts. |

## Cross-cutting preservation criteria (all rows)

Immutable domain data, owner construction, explicit failures, shared
client/server contracts, checked generics, audited browser capabilities and
ownership, transaction commit uncertainty, assertion/fixture evidence, native
operations. Any alternative violating these without an explicit user-level
re-decision is out of scope.
