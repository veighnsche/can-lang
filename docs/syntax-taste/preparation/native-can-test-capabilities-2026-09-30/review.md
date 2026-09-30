# Final shared-capability review

Status: design review and document validation only. No Can/compiler/runtime implementation, tests, builds, live services or performance runs.

The [main contract](../../native-can-test-capabilities-2026-09-30.md) was independently reviewed for [resource ownership](review-resources.md), [browser behavior](review-browser.md) and [observation/diagnostic semantics](review-observations.md). The reviewers' findings refer to the draft they saw; the following dispositions record the final revisions.

| Finding | Resolution in the main contract |
| --- | --- |
| Crash after dispatch but before reply | Durable pre-dispatch identity; after restart an unresolved dispatch returns indeterminate and cannot be automatically re-dispatched |
| Partial cleanup incorrectly frees capacity | Retain ownership, leases and quota charges for unresolved processes, descriptors, paths and remote resources; release only confirmed freed resources |
| Revision/read/pagination overpromises external-write detection | Explicit scoped-revision, qualified-snapshot, detected-change and unknown stability; no strong snapshot claim from an open handle, timestamps or before/after hashes |
| Sealed named files omit additions/removals | Full-tree versus partial scope; include directory/link/namespace identity and meaningful absent lookups for compiler inputs |
| Invalid control has no post-seal transition | New reservation/materialization from a read lease, explicit mutation, then seal; sealed valid inputs are never reopened implicitly |
| Subject-realm Can callback might become the judge | R builds observer callback artifacts; their execution results remain subject facts judged in the reference worker; no private reporting authority |
| Passive API discovery invokes hostile getters/traps | Describe only safe known metadata; unknown is explicit and effectful discovery requires an observed action |
| Async transport accidentally assimilates raw thenables | Catalogue declares synchronous/raw versus asynchronous semantics; inert handle/completion envelopes add no implicit thenable access |
| Independent native DOM event facts missing | Generic target/type/phase subscription records actual delivered event fields, including trusted/composition/modifier facts |
| Action acknowledgment mistaken for event completion | Confirm listener boundary before action; identify action/document/navigation and checkpoint watermark, with no final-event or cross-process-causality claim |
| Route fetch/delivery completion and retry ambiguous | Full bounded body/EOF required for completed fetch; token state machine, permanent contact consumption, explicit fulfill/abort choices and separate upstream/delivery identities |
| Route pre-contact wording overstates network isolation | Guarantee scoped to that driver's routed fetch; full egress needs separately qualified host/remote enforcement |
| Quiet checkpoint confused with final seal | Separate `browser.checkpoint` and `browser.seal`; final close drains tracked body/error callbacks and reports pending/gaps |
| Raw response representation ambiguous | State capture layer, header normalization and transfer/content decoding; fulfill representation must be compatible; malformed-wire cases use controlled peers |
| Early rejection claims whole-project analysis | Separate declared input closure from actual reads, with unvisited units and per-stage completion/stop states; source rejection remains distinct from infrastructure failure |

The draft source notes also mentioned suspending getters and accepting quiet intervals in a seal. The main contract explicitly restricts synchronous callbacks to local immediate execution and quiet intervals to prefix evidence. Notes remain exploratory source context, not competing API specifications.

Remaining implementation gates are intentionally open: durable launch/recovery mechanisms; enforced descendant/filesystem/network/disk limits on supported hosts; finite resource-profile values; concrete Can declarations/native catalogue and fixtures; exact versioned transport schema; and binding-by-binding ledger coverage. None was silently marked implemented or measured. The selected execution architecture, full Can authorship and no-host-harness endpoint remain unchanged.

The architecture, authoring design and migration ledger now link this contract and distinguish specified semantics from pending implementation proof. No ledger obligation is marked migrated or eligible for deletion by this review.

See [validation.json](validation.json) for local link/consultation checks and source fingerprints. No temporary execution allocations were made; only compact repository documentation and consultation evidence are retained.
