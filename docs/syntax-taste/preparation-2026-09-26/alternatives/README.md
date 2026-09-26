# P04 interaction map + experiment register

## Shared contracts (resolve before lanes diverge — feeds P07.3/P08)

| Contract | Topics | What must agree |
|---|---|---|
| C-A iteration/lowering | R05 | Lowering proof or primitive semantics; evaluation order, failure identity, fixture paths, diagnostics |
| C-B failure representation | R06, R07, R12 | `emits` bounds, result-data conventions, occurrence/provenance, redaction rules |
| C-C ownership/lifetime | R04, R02, R11, R15 | Drain vs supervise, lease preservation, disposal, shutdown escalation, cancel/unknown-write honesty |
| C-D browser boundary | R01, R02, R09 | Snapshot/property additions, append/ownership, capability admission, LSP query shapes |
| C-E action/route wire | R03, R02, R10 | Capture grammar, HTML document vs fragment policy, status/leaf agreement |
| C-F SQL boundary | R10, R11, R04 | Descriptor admission (RETURNING? locking reads?), conflict recipe, deadline operands, migration provenance |
| C-G outbound policy | R11, R14, R01 | Destination rules, credential binding, budget/correlation identity |
| C-H deployment pairing | R13, R02, R03 | Paired build identity, retention, old-browser behavior, rollout/rollback |
| C-I authoring policy | R08, R09 | Boolean order, local elision, near binding vs rename/format/hover/reference designs |

## Dependency edges (research-established)

- R02 → R01: generic DOM additions + two-instance library proof precede the
  adapter-vs-companion discrimination.
- R09 → R08: safe rename waits on the near-binding decision; format/hover/
  references do not.
- R03 independent of R02 native work and R10/P10 router questions.
- R04 ↔ R15: deadline/cancel contract shape shared (operation budgets,
  unknown-write honesty).
- R04 ↔ R11: worker budgets/concurrency ride on request/operation contracts.
- R10 ↔ R11: claim mechanism depends on the locking-read probe outcome.
- R06 ↔ R07: failure ergonomics + assertion setup share the emits/evidence story.
- R12 ↔ R04: shutdown/failure reporting shares occurrence/correlation identity.
- R13 ↔ all: qualification environments gate acceptance, not design.
- R14 ↔ R11: AI budgets share outbound-policy identity choices.

## Experiment register (P04.4)

Preparation ran no new live experiments: all review claims reuse retained
evidence at the identical revision, and the two remedy-discriminating probes
need provisioned live services with credentials (PG creds absent here; S3
endpoint unprovisioned). Each entry states what it can and cannot establish.

| ID | Question | Needs | Can establish | Cannot establish |
|---|---|---|---|---|
| X-R04-1 | Does `Bun.SQL Query.cancel()` abort server-side per dialect? | Live PG (+MySQL) with creds | Whether O1 (operation deadlines) has a SQL foothold | Request-budget composition policy |
| X-R04-2 | Real HTTP hedge: O1 vs O2 | Staging server + stall injection | Response/latency, lease, diagnostic, shutdown comparison | Universal latency SLO |
| X-R04-3 | Bun serve-stack disconnect signal? | Pinned Bun + probe | Whether ingress-lifetime projection is possible | Policy choice itself |
| X-R15-1 | Does `sink.end(Error)` release without publish? | Disposable bucket + creds | Whether O1 (true cancel) is implementable | Concurrent-observer behavior (needs X-R15-2) |
| X-R15-2 | Cancel-while-replacing + observer visibility | Disposable bucket + creds | Preservation + non-observation proof | Service-failure matrix (needs X-R15-3) |
| X-R15-3 | Never-settling awaits under deadline | Hung reader/writer/end/stat harness | Bounded-return proof for the deadline design | — |
| X-R06-1 | Result-data retry helper, two error sets | Ordinary Can authoring | Concision + provenance cost; attempt/sequence correctness via oracle | Whether O3 syntax is warranted (decision follows) |
| X-R07-1 | Factory-pattern extraction example | Ordinary Can authoring | Measured setup cost on a real extraction | Whether setup syntax is warranted (decision follows) |
| X-R08-1/2/3 | Authoring-policy repair trials | Registered held-out tasks | Measured repair cost per policy (Boolean/locals/near) | The user's taste decision (informs Q1–Q3) |
| X-R02-1 | Library extraction + second app | R02 native additions | Whether O1 library-first composes; minimal native set | Component-syntax need (only if O1 fails) |
| X-R01-1 | Third-party widget: adapter vs companion | X-R02-1 done | Discrimination data for the host decision | — |
| X-R10-1 | Locking-read descriptor probe | Live PG | Whether claim-via-locking-read is expressible | Claim policy itself |
| X-R11-1 | Two-worker claim + crash recovery | Live PG + 2 workers | SUPERSEDED under companion scope (P09-m3): reframed as companion claim/lease + crash-redelivery qualification per the B3 contract | Carrier choice (user scope S1, decided: companion) |

No option becomes accepted merely because a prototype exists. X-items that
need only Can authoring (X-R06-1, X-R07-1, X-R08-*) can run during
implementation planning support; live-service items belong to implementation
lanes with provisioned environments.
