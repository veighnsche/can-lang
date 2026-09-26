# P07.2 selected-change contracts

Each accepted change: observable behavior, grammar/checking rules,
errors/provenance, ownership, native mappings, package/wire boundaries, and
the prior rules it supersedes. Conditional syntax (Q4/Q5/Q6-O3) is specified
only as gates here — its grammar is written if its experiment triggers it.

## R08 authoring changes (user-selected) — interface C-I

**Q1 Boolean order.** Checker deletes the `false`-after-`true` error in
`completion_matches.go` for ordinary single-scrutinee Boolean data matches.
Formatter rewrites `true`-first arms to `false`-first (fixpoint-stable,
overlay-validated like `--write`). Diagnostics for exhaustiveness/coverage
unchanged. No runtime/ownership/wire effect. Supersedes: decisions.md
"Ordinary Boolean match arm order" enforcement + P15 Retain (reversed by the
same user authority; the canonical spelling remains false-first).

**Q2 final locals.** Checker deletes the C8 `unnecessary local` error
(`locals.go`); evaluation never depended on it. The same four-clause shape
is reported as a check-pipeline *warning* (`accidental alias — consider
inlining`), surfaced via CLI diagnostics and LSP `publishDiagnostics`;
warnings never fail a build and exit code stays 0 for warnings-only
(P09-B1: `canlc lint` stays retired, no new command).
Formatter preserves the binding as written. No type/error/runtime change.
Supersedes: C8 validity rule + DI-23 + P14 (demoted, explicitly reopened).

**Q3 near bindings.** New grammar: `callable <name> with <param> = <expr>,
...` binds listed near-inputs explicitly by callee parameter name. Checking:
each listed name must be a declared near-input of the target (unknown names,
duplicates, and non-near names are `CAN-CHECK-CAPTURE` errors); each
expression checks against the declared input type in the creation scope;
unlisted near-inputs use existing name lookup; direct-call argument supply
unchanged (positional). Emission captures explicit expressions instead of
looked-up names. Provenance: `with` bindings are rename-references to the
callee parameter. Formatter preserves binding order. Supersedes: pure
name-based capture obligation (LD38/LD39 + DI-08 + P13) — fallback retained,
explicit pinning added.

## R03 captured reads (Q7) — interface C-E

Captured GET actions may declare `body html` with response mode `document`
(full-page render) in addition to `swap inner` (fragment). Checking reuses
capture/URL/ambiguity machinery; statuses stay 200–599 excl. 204/205/304;
every case renders a body; HTML guard policy (missing-target/OOB/
control-header rejection) applies to documents and fragments alike.
Ownership: request-scoped like existing actions. Wire: same capture URL
building + canonical URLs. No plain-route grammar change. (O1 selected over
O2 to avoid a second URL grammar.)

## R05 iteration + bulk construction — interface C-A

**Self-tail lowering.** The checker proves `relay call` self-calls in tail
position with no pending frame-owned work; the emitter lowers proven shapes
to native `while` loops. Observable call/test/fixture behavior identical;
non-proven recursion keeps current behavior with a "not lowered" diagnostic
note. Ownership/lifecycle unchanged. Native mapping: completion protocol
over a loop instead of nested thunks. If the proof boundary excludes needed
W4 shapes, an explicit primitive is proposed (Q6 gate) — same lowering, new
surface. Preserves: order, exactly-once args, failure identity, fixtures,
diagnostics; no portable numeric threshold is claimed.
(P09-M9 proof predicate, exact:) lowerable iff the tail call targets the
enclosing function itself AND the frame holds no live lease, no owned value
requiring drain, no pending timer, and no deferred completion. Negative
fixtures (lane-A task): mutual recursion, frame-held lease, pending timer,
non-self tail call, value-producing post-processing after the call.
(P09-M8:) lowered-loop failures carry the step index in the failure detail
plus the occurrence ID linking to logs.

**Bulk construction.** Catalogue additions (e.g. map/set from-arrays) with:
one native construction + single immutable publication; defined duplicate
policy, iteration order, per-element failure behavior, and ownership (builder
never escapes). No new syntax. Supersedes nothing (additive); P18 reopened
and resolved by this contract.

## R06/R07 library conventions (experiment-gated) — interface C-B

**Result-data convention.** Nominal outcome records + boundary adapters as
the first-class reusable-helper pattern; X-R06-1 fixes the concise form,
conversion/provenance rules, and attempt/sequence oracles. Finite error-set
syntax only under the Q5 gate (proposed only with full C4/C5/C9 contract).

**Factory convention.** Private fixture helpers with explicit
unexpected-rejection policy as the first-class owner-setup pattern; X-R07-1
fixes the worked form + measured cost. Setup-region syntax only under the Q4
gate (then explicitly reopens LD29).

## R01 host boundary (experiment-gated) — interfaces C-D, C-G

X-R01-1 fixes, per widget class, the catalogue-vs-adapter-vs-companion
assignment. Adapter tier (if selected for a class): declared I/O types with
immutable copying, failure mapping, target availability, callback/reentrancy
rules, ownership/disposal, reproducible packaging, conformance suite.
Companion tier: contract parity, serving, auth, release coordination recipe.
Neither tier permits self-admission: distribution review stays. Package
boundary: adapters ship as reviewed packages; companions as separate
runtimes/services with a typed data boundary.

## R02 native additions + libraries — interface C-D

Each addition is a catalogue + checker + adapter slice with a bounded
immutable projection (snapshot fields are immutable records; live setters
take explicit values; selection/caret ops are explicit calls). Libraries are
ordinary Can (functions/records/callables) over the existing lifetime model.
Ownership: view-scoped listeners/timers, disposal aborts, no post-disposal
mutation — unchanged. Second-app consumption proves reuse (W1).

## R04 request policy + operation contracts — interface C-C

A written request-policy spec owns: per-adapter interruption contracts
(SQL cancel semantics post-X-R04-1, S3/stream behavior, fetch abort),
budget composition, disconnect/SIGTERM propagation, shutdown escalation
(waiting deadlines leave work owned; supervisor SIGKILL bounds nonsettling
work), and honest unknown-write outcomes everywhere. Caller deadline/cancel
operands on SQL and browser `action::request/post` map 1:1 to qualified
native contracts — no operand without a native meaning.
Ownership: leases never revoked by timers. Supervised-loser (O2) design only
if the hedge measurement (X-R04-2) shows O1 insufficient.
(P09-B7, decided here:) budgets compose as one SHARED request budget —
each operation's effective deadline is min(its own bound, remaining budget).
Serial per-operation budgets (5s+5s+5s = 15s user-visible) are rejected as
the default; per-operation bounds remain mandatory inputs. Cancel-absent
branch (X-R04-1/X-R04-3 negative): the request boundary still returns at
budget expiry while the native op stays owned until settlement, reporting
an honest unknown-write outcome with supervisor escalation — bounded and
honest at the user-visible layer without pretending to abort the unabortable.

## R12 redacted hook — interfaces C-B (redaction), C-C (identity)

Automatic per-request reporter at `serveOuter` (+ dispatch/adapter layers if
the packet places them there): one sanitized diagnostic per unexpected
failure with correlation/source identity; redaction reuses `entry.ts` policy
(domain identity, category + occurrence, no native messages/paths/secrets);
client responses stay fixed 500s. Exactly-once-per-occurrence discipline
shared with owner/browser reporters. Manual catch+log remains the policy for
expected failures.

## R15 S3 (conditioned contracts) — interface C-C

Cancel: true "never materializes; never deletes" if `end(Error)` verifies
(X-R15-1); else (P09-B8) `cancel_upload` is REMOVED from the catalogue and
replaced by explicitly-named `discard_upload` carrying the destructive
contract + unknown-outcome reporting — the hazard is removed structurally,
not by acknowledgment flags. No silent delete under any outcome. Deadline:
effective bounded-abort design if one verifies (X-R15-3); else the honestly
documented between-awaits bound (which then fails W5 and needs explicit
user scoping). Terminal-state guards and unknown-outcome honesty hold under
all outcomes. Supersedes: the contradicted catalogue line 12399 and any
reading of the current deadline as an abort guarantee.

## R10 relational — interface C-F

RETURNING: admitted only post-demo with per-dialect row/cardinality/error
rules. Encodings: blessed guide (minor-unit money, ms-epoch ints, opaque
TEXT + codecs); new column kinds only on demonstrated insufficiency.
Conflict recipe: lookup-first for PG (example port + docs); savepoints only
on demonstrated need. Worker-claim reads: locking-clause probe (X-R10-1)
decides expressibility; no syntax either way. Migration: operator-owned DDL
stays the stated boundary (P19 deferral retained).

## R11 companion pair (S-worker) — interfaces C-F, C-G, C-C

Can side: ledger/outbox/attempts, idempotent decide/ack transactions,
carrier protocol endpoints **with authentication added** (closes the R16-03
gap), commit-unknown reconciliation. Companion side: delivery, retry
scheduling, bounded concurrency, poison handling, crash recovery, tenant
destination policy enforcement (allowlist/pattern, credential binding,
redirects). Shared: versioned protocol + conformance suite + auth envelope.
Fixed-origin Can HTTP and trusted-carrier scope retained on the Can side.
(P09-B3 claim/lease + guarantees, all decided here:)
claim = lease-table rows claimed by a `claim_outbox(worker_id, lease_until)`
transaction (lookup-first, portable — no locking-read dependency);
worker identity + lease expiry + heartbeat in the protocol version;
delivery guarantee = at-least-once with idempotent ack + the C-C
unknown-write vocabulary (duplicates absorbed by idempotency, never by
assuming single delivery); poison = after N protocol-versioned attempts the
companion marks the row via a `dead_letter` endpoint and Can owns the
dead-letter table + `decide_ack` dead-letter rules; companion restart =
supervisor restarts the companion, unacked rows redeliver, no Can-side
recovery beyond idempotency.

## R09 LSP sequence (Q8) — interfaces C-I, C-D

Format (wire `textDocument/formatting` to `formatSource` + overlay
validation; whole-document), hover (type-at-offset over `World`), references
(project index including body-local variables, selected by the user's 1A
blocker answer), completion
(scope-aware with keyword/near/arity precision rules), rename (references +
`--write`-discipline validation; `with` bindings as near-parameter
references). Snapshots stay inert; diagnostics identical to CLI; definition
keeps declining rather than guessing. Find References and safe rename resolve
local bindings by scope identity: shadowed or identically named bindings in
other scopes are not references to the selected variable. Existing `with`
callee-parameter references remain included. This closes BLK-02 without
changing definition lookup or introducing syntax.

## R13 deployment + R14 AI + R16 docs — interfaces C-H, C-G

R13: UP25 qualification on x86 (packaging/install/smoke + PG roundtrip),
lifecycle recipe (service unit, health, rollout/rollback, credential
provisioning), old-browser acceptance (fail-closed generation handshake per
C-H — H qualifies against the policy, it does not define it). R14:
the [blocker-resolution contract](blocker-resolution/README.md) now specifies
tenant input-plus-output token allowances, immediate typed rejection, configured
fixed epochs, durable native admission reservations, authoritative settlement
and conservative unknown-use holds. F owns C-G identity and ledger service;
H owns native AI enforcement/usage/profile qualification; E owns shared context
and transport integration. The ordinary catalogue scope operation uses existing
call syntax, preserving native AI grammar. A complete-request bound must be
qualified before budgeted dispatch; unqualified profiles fail closed and cannot
count as successful W6-AI qualification. Model-change evaluation uses the
registered support-ticket-triage feature; streaming stays out. R16: the three
stated corrections (R16-01 owned by C, R16-03 by F
rewritten to describe the auth envelope, R16-02/R16-04 by H) + supported-
story doc after boundaries land; the story states the rich-client boundary
explicitly (which browser subsystems are supported; history/WebSocket/
specialist widgets out per P10 deferral). All keep `canlc assert` green
(U06 gate).
