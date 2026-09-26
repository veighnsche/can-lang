# P07.3 interaction reconciliation — shared interfaces first

Contracts C-A..C-I from the interaction map, reconciled to single owners.
Dependent lanes start against these interfaces; no lane redefines them.

## C-A iteration/lowering (R05)

Owner: core lowering. Interface: the tail-proof rule (what counts as
lowerable) + the shared native-loop lowering + step-indexed diagnostics.
Consumers: W4 workloads, worker batch (R11 companion batch shape mirrors the
same step/failure reporting), R09 completion (no new keywords unless Q6-O3
triggers). Decision: lowering ships before any primitive surface is even
proposed.

## C-B failure representation (R06, R07, R12)

Owner: failure conventions. Interface: `emits` stays finite/explicit
everywhere; result-data outcome record/variant shapes fixed by X-R06-1;
occurrence/provenance rules unchanged; redaction policy owned by `entry.ts`
and reused by the R12 hook. Consumers: retry/trace helpers, factory
fixtures, boundary reporters. No error-set syntax without the Q5 gate.

## C-C ownership/lifetime (R04, R02, R11, R15)

Owner: lifetime policy. Interface: the written request-policy spec (R04):
drain-by-default, leases never timer-revoked, supervised-loser rules only
if X-R04-2 requires them, shutdown escalation, honest unknown-write
outcomes. Consumers: browser view disposal (unchanged semantics, relied
upon), companion protocol (cancel/unknown outcomes mirror Can-side
honesty), S3 cancel/deadline contracts. One unknown-write vocabulary across
SQL/S3/actions/workers.

## C-D browser boundary (R01, R02, R09)

Owner: browser platform. Interface: snapshot record shape (additive fields
only), live-property setter signatures, append/ownership rules, capability
admission list, LSP query shapes over `World`. Consumers: widget libraries,
second app, adapter tier (if any browser capability ships as an adapter),
editor features. Additive-only: existing snapshots keep working.

## C-E action/route wire (R03, R02, R10)

Owner: action contracts. Interface: capture grammar + canonical URL building
+ ambiguity rejection + status/leaf agreement + `document` vs `swap inner`
modes + HTML guard policy. Consumers: captured reads, grid/form actions,
companion protocol endpoints (same status honesty). One URL grammar.

## C-F SQL boundary (R10, R11, R04)

Owner: SQL contracts. Interface: descriptor admission (SELECT breadth as
today; RETURNING/locking clauses only per probe/demo outcomes), row
validation profiles per dialect, conflict recipe (lookup-first), deadline
operands (only post-qualification), operator-owned DDL. Consumers: PG app,
MySQL parity legs, companion ledger/outbox, request budgets.

## C-G outbound policy (R11, R14, R01)

Owner: Lane F (data owner) — destination rule format, credential binding
(env-name-only, never logged), redirect policy, budget/correlation identity
vocabulary, assertion/fixture story for tenant-supplied URLs. Consumers:
companion delivery (F), AI budget enforcement (H consumes the identity via
an explicit F→H handoff), any adapter making network calls (D consumes).
Single identity vocabulary for tenant/correlation across HTTP and AI.
(P09-B2: single owner resolves the F/H planning cycle; M7: identity-
affecting budget parts are decided with F first.)

## C-H deployment pairing (R13, R02, R03)

Owner: release engineering. Interface: paired-build hash identity +
shared-lock binding, 7-day retention, network-denied smoke, old-browser
acceptance definition, rollout/rollback + credential provisioning recipe.
Consumers: server/browser builds, companion releases (paired or versioned
with the protocol), UP25 runs.
(P09-B4 interop policy, decided here — Lane H qualifies against it, it does
not define it:) no compatibility obligation → fail-closed. Every paired
build carries a server generation; the browser bundle sends its generation
on action transport, and on mismatch the server answers a typed
generation-mismatch error (never silent wrong behavior); the app presents a
blocking refresh prompt. Old digest URLs stay servable per retention, but
old app logic against a new server is a defined error, not interop.

## C-I authoring policy (R08, R09)

Owner: surface policy. Interface: Q1 canonical form (false-first) +
formatter rule; Q2 lint shape + keep-source formatting; Q3 `with` grammar +
checking + rename references; hover/completion/rename query semantics.
Consumers: checker, formatter, LSP, all example sources (migrated by
the formatter — no compatibility obligation).
(P09-B1: advisory vehicle = warning severity in the check pipeline,
surfaced via CLI diagnostics + LSP `publishDiagnostics`; exit code stays 0
for warnings-only and builds are unaffected. `canlc lint` stays retired;
no new command.)

## Reconciliation notes

- C-C × C-G: budget/correlation identity (C-G) plugs into the request policy
  (C-C); neither owner invents the other's vocabulary.
- C-D × C-I: `with` bindings reference callee parameters — the browser
  libraries' `near` uses become the first rename tests.
- C-E × C-H: captured reads ship through the same paired-build identity as
  JSON/form actions.
- C-F × C-C: SQL deadline operands (C-F) implement the operation-contract
  half of the request policy (C-C).
