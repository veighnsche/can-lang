# Readiness audit: known blockers and unresolved scope

Audited against the committed handoff on 2026-09-22. **This is a prepared implementation programme, not thirteen fully settled designs with guaranteed native feasibility.** The earlier preparation verified samples and supplied implementation instructions; it did not eliminate every blocker. This audit collects the known issues that were dispersed across capability plans.

There is no honest guarantee against undiscovered blockers. The useful guarantee is procedural: known blockers are named, each blocks only the relevant deliverable, and an unresolved requirement cannot disappear behind a completed parent checkbox.

## Capability-by-capability readiness

“Can start” means meaningful implementation can proceed. “Cannot close yet” names design or feasibility evidence required beyond writing the ordinary adapter/tests. Every capability also needs its normal acceptance suite.

| Capability | Can start | Cannot close yet / unresolved contract | Resolution owner and proof |
|---|---|---|---|
| B1-01 Files/path/glob | Yes, ordinary bounded APIs. | Overwrite/move atomicity, symlink following and resolution semantics need explicit contracts. A filesystem sandbox is not established or promised. No known missing native API blocks the basic capability. | B1-01: native exclusive-create, symlink and cross-device cases; publish guarantees before claiming atomic/constrained operations. |
| B1-02 SQLite/shared SQL | Yes, dialect plumbing and SQLite native adapter. | **SQL grammar backend is undecided.** The current PostgreSQL parser does not qualify SQLite/MySQL syntax. Exact mappings, locking and query interruption also need proof. Native awaited transactions/rollback/reopen have passed a sample, not Can integration. | B1-02 / G-SQL: select actual parser/admission design, corpus and spans; qualify native connection/lifetime contract. |
| B1-03 MySQL | Yes, after the shared descriptor interface is usable. | Inherits the SQL grammar decision. **No real MySQL qualification has run.** BIGINT/DECIMAL, datetime, errors and transactions need service evidence. | B1-03: controlled service and the three-engine corpus. Service unavailable means this acceptance remains blocked, not the entire queue. |
| B1-04 Processes | Yes, bounded direct-child execution. | **Descendant cleanup guarantee is unresolved.** Killing one PID is not process-tree cleanup; pipe/deadline/reaping races need evidence. | B1-04: owned child/grandchild reproducer and native platform strategy, or an explicit reconciled direct-child limitation. No silent guarantee reduction. |
| B1-05 Streams/lifecycles | Yes, native ownership prototype and comparison cases. | **Public surface/primitive grammar is not selected.** Repeated dispatch, calculated handler errors and resource admission must be proved through checker/IR/runtime, not just a JavaScript stream sample. | B1-05 / G-EVENT: three-workflow comparison, selected source contract, owner/backpressure/fixture evidence. |
| B1-06 HTTP | Yes, bounded methods/headers/routing extensions. | Depends on streams for incremental bodies. **Incremental multipart parsing has not been shown feasible using only the allowed native APIs.** TLS and post-publication failure handling remain unqualified. | B1-06: actual pinned API proof for multipart, local TLS and disconnect/publication tests. Buffered FormData does not complete a promised streaming multipart subtask. |
| B1-07 WebSocket | Yes, native client/server adapters and ownership prototypes. | Inherits the event surface decision. **Client/server backpressure contracts differ**; no proven common drain behavior. Handler serialization/queue overflow/close semantics still need proof. | B1-07 + B1-05: pin each side's contract and run slow-consumer/close-race tests. No invented client drain event. |
| B1-08 Crypto | Yes, password/HMAC/AES-GCM adapters. | **The full algorithm/import/export subset is not frozen.** Ed25519 is a qualification candidate; only password and AES-GCM samples ran. | B1-08: enumerate admitted algorithms/key formats/usages, native vectors and negative imports. Record excluded candidates explicitly. |
| B1-09 Cookies/CSRF | Yes, typed transformations and explicit session-bound CSRF. | Cookie duplicate/prefix/SameSite/expiry policies need final contract and tests; HTTP header integration depends on B1-06. No known missing native entry point blocks the basic capability. | B1-09: policy table, real native verification and repeated Set-Cookie acceptance. |
| B1-10 S3 | Yes, configuration/mapping/fixture work and API qualification. | **No real S3-compatible service qualification has run.** Required signing, listing, multipart and cancellation details have not been proved on the pinned target. Constructor presence is insufficient. | B1-10: exact native API checks plus isolated service tests for every claimed operation, including abort cleanup. |
| B1-11 Formats | Yes, explicitly bounded native-semantics decoding and JSONL framing prototypes. | **Exactness and duplicate rules are not uniform or fully frozen.** Already-parsed numbers cannot recover lost tokens; native alias/date behavior needs admission rules. Streaming completion depends on B1-05. | B1-11: per-format semantics matrix and probes. Preserve existing exact JSON; label any weaker format contract rather than claiming universal losslessness. |
| B1-12 Markdown | Yes, ordinary-string rendering. | **Safe rendering/structured callback design is unresolved.** Native callbacks consume/return strings synchronously; async Can handlers cannot be directly substituted. Existing safe HTML builders do not automatically solve this bridge. | B1-12 / G-HTML: safe fragment invariant, callback trace/prototype and hostile browser corpus; public callback staging must preserve Can errors/ownership. |
| B1-13 Utilities | Yes, URL/query/encoding and explicit-instant utilities. | Regex offset/flag behavior and strict encoding policy need contracts. **Hard regex timeouts are not established.** Local civil-time conversion through DST is not settled if included. | B1-13: explicit unit/format table; qualify any hard deadline or local-time conversion before admitting that extra guarantee. |

## Cross-cutting blockers and scope boundaries

- **Active LF implementation:** integration must use the finished relevant error/assertion/ownership contracts or isolate independent work. Preparation did not verify the final LF result. Do not edit around half-completed shared interfaces or interrupt that implementer.
- **Synchronous native execution:** JavaScript timer races cannot preempt synchronous regex, parsing, Markdown or SQLite work. Input/result caps are not hard CPU deadlines. Where a hard deadline is required, native interruptibility or an allowed isolation design is a feasibility gate. Do not silently pull the later worker redesign into ASAP.
- **SQL parser dependency:** the plan names alternatives, not a selected maintained parser. “Use a parser” is still unfinished design. A compiler dependency must be justified; a native database runtime does not supply an offline multi-dialect compiler proof automatically.
- **Public API closure:** many signatures are interface notation. Routine naming and error tables can be settled while implementing; changing Can guarantees, introducing new syntax or weakening exactness requires explicit contract reconciliation and the repository's difficult-design consultation process.
- **Service provisioning:** MySQL and S3 are acceptance prerequisites for their own branches. No production credentials, paid infrastructure or service installation is assumed by this handoff.
- **No automatic fallback completion:** “if native support is missing, keep the subtask visible” means the capability remains incomplete. It does not authorize an implementer to call the whole milestone done after dropping multipart, safe Markdown, exactness or process cleanup promises.

## Scheduling corrections

`ASAP-01` must record service readiness and capability-specific blockers, not require every external service to exist before any implementation can begin. Its wording has been corrected accordingly.

The machine queue's cross-capability edges are conservative full-capability dependencies. They are valid for completion, but some partial work can start earlier: bounded format parsing before streams, bounded HTTP before streaming, and crypto/utility work before service qualification. Track partial progress without checking off the parent. If splitting the queue further, give each slice explicit prerequisites and keep every original acceptance obligation mapped to a completion step.

G-EVENT uses comparative proposed Can programs and native harnesses. It must not require the fully implemented downstream HTTP/WebSocket/JSONL capability before choosing the upstream stream surface; that would introduce a dependency cycle. The comparison chooses the design, and subsequent integration tests validate its implementation.

## What this audit did and did not do

This audit reviewed all thirteen plans, their stored step/risk inventory, the decision ledger and execution dependencies. It promoted existing concerns into one visible register and corrected a preflight scheduling ambiguity. It did not execute new native feasibility probes or resolve the SQL/event/HTML designs. Those remain outstanding work, not hidden completed preparation.

## File-layout prerequisite

The [required filetree](filetree.md) adds ASAP-ARCH before feature additions. Existing large entry points must delegate to cohesive feature modules. This is implementation organization with a size guard and responsibility review, not a claim that the outstanding native/design gates are resolved.
