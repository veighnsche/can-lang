# Native coordination

The four source forms lower to `Promise.all`, `Promise.allSettled`, `Promise.any` and `Promise.race`, respectively. Participant adapters retain protected completion values and use private rejection carriers. Native Promise operations select the result; Can handlers run afterward in their specified region and order.

All written callees, arguments, nested calls and callable spreads finish preparation before any participant launches. A method chain remains one participant: only receiver bindings requiring earlier chain results remain inside it. Preparation failure escapes before launch and cannot enter a participant arm. Spreads flatten in written order and retain their homogeneous nullary callable contract.

Every participant is registered with the runtime owner, including prepared captures, before launch. Early completion does not cancel losers or release their leases. Late standard failures produce one sanitized diagnostic; completed callers do not change. First-success races consume failures preceding the winning success while preserving diagnostics for later standard failures. Root shutdown drains outstanding owners.

Concurrent handlers map elements in input order after native selection; shared rejection handlers return the whole fallback array. A handler failure exits directly, stops later handlers and never reenters participant arms. Void forms produce no observable result array. Empty concurrent expansions succeed with empty arrays, first-success race selects its empty aggregate handler, and first-completion race remains pending without an implicit timeout.

An ignored `all_failed` payload needs no named aggregate type. Observing `all_failed.failures` requires one expected named failure variant; forwarding obtains the exact specialization from the enclosing error bound. The variant must cover all declared participant domain failures plus `standard_failure`. Discovery retains only authored expected types, then the entire handler is checked again with sealed evidence. Conflicting or missing variants fail; no variant search, inferred error set or array covariance is introduced. Nested handlers have distinct implicit aggregate aliases.

Aggregate arrays preserve input order, duplicate occurrences, original nominal domain payloads and opaque native failure snapshots. Distinct specializations of one bare error kind require explicit normalization before sharing a bare arm. The source composition fixture performs an exhaustive elementwise transformation into a common named variant.

The [I19 validation report](evidence/2026-09-21/i19-validation.md) maps the nine required traces to executable evidence. Deterministic source fixture queue allocation remains the separate I18 task; coordination does not accept a `when` table beneath its participant list.
