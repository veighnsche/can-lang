# Native Can migration checklist

Status: **audited 2026-10-01; see current task states below**. The [lane plan](lane-plan.md) defines dispatch and path reservations; the [master ledger](tasks.json) counts each of 143 task IDs exactly once. Historical source work exists; this planning audit runs no implementation or qualification.

Each row may be mapped, authored and reviewed after its own start gates. Listed gates are minimum prerequisites; facet mapping adds any further gate required by the actual mechanism, variant or environment. Its own accept gates, rather than the group union, control its completion. Historical/static relevance decisions can finish before live runner qualification; a retained live behavior must then satisfy its conditional gates. Full-suite qualification is a later campaign/release gate. No deferred performance work is authorized.

For every live row: retain all protected facets, dynamic variants, named fixtures, target/environment gates and delegated declarations; put actions and expected values in ordinary Can; exercise positive, seeded defect and missing-evidence controls; save exact R/N/S/A/C identities, Can report, N cleanup receipt and current correction snapshot.

Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

M43–M45 complete with qualified Can policy and reviewed staged caller patches. Integrator Z03 owns actual activation; integrator Z02 owns executable harness retirement.

## M01 — Assertion and check evidence

- [ ] **M01: planned** — owner lane: `migration-core`.

Proposed owner: `tests/native-can/migration/m01-assertion-and-check-evidence/**`. Group start after: P15, P20, P27, P28, K01, K02, K03. Group accept after: P23, QN1. Logical gates: SUITE, DIAG, FAULT.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Port structured assertion selection, source spans, sticky violations, true/false reasons and malformed/missing evidence controls into Can-owned cases.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Port structured assertion selection, source spans, sticky violations, true/false reasons and malformed/missing evidence controls into Can-owned cases. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: DELEGATE-007; shared delegate references: none.

Suggested first small commit: `test(native-can): add assertion-and-check-evidence cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### CORE-002 — TestCurrentBundledAssertions

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 5 protected facets, 3 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-003 — TestAssertionFailureLocations

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 2 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-006 — TestCurrentBundledChecks

Disposition: migrate. Start after: P15, P20, P27, P28, K01, K02, K03. Accept after: P23, QN1.

- [ ] Account for 2 protected facets, 2 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Map delegated oracle evidence: DELEGATE-007.

### CORE-007 — TestChecksMismatchSpan

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M02 — Callables, generics and template checking

- [ ] **M02: planned** — owner lane: `migration-core`.

Proposed owner: `tests/native-can/migration/m02-callables-generics-and-templat/**`. Group start after: P15, P20, P27, P28. Group accept after: P23. Logical gates: SUITE, DIAG, BUILD.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Exercise captures, generic dependency chains, located invalid edits and template fixtures through candidate builds and Can comparisons.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Exercise captures, generic dependency chains, located invalid edits and template fixtures through candidate builds and Can comparisons. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add callables-generics-and-templat cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### CORE-005 — TestCurrentBundledCallables

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 2 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-023 — TestCurrentBundledGenerics

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 3 protected facets, 5 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-024 — TestCurrentBundledGenericChain

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 3 protected facets, 4 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-041 — TestCurrentBundledTemplates

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 4 protected facets, 2 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M03 — Arrays, bytes, codecs and collections

- [ ] **M03: planned** — owner lane: `migration-core`.

Proposed owner: `tests/native-can/migration/m03-arrays-bytes-codecs-and-collec/**`. Group start after: P15, P27, P28, K01, K02, K03, K04, K05. Group accept after: P23, QN1, QN2. Logical gates: SUITE, BUILD, N1, N2.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Map each delegated array/collection declaration and dynamic variant; test immutable copies, order and codec bytes with C adapter ingress where required.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Map each delegated array/collection declaration and dynamic variant; test immutable copies, order and codec bytes with C adapter ingress where required. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: DELEGATE-006, DELEGATE-008; shared delegate references: none.

Suggested first small commit: `test(native-can): add arrays-bytes-codecs-and-collec cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### CORE-001 — TestCurrentBundledArrays

Disposition: migrate. Start after: P15, P27, P28, K01, K02, K03, K04, K05. Accept after: P23, QN1, QN2.

- [ ] Account for 3 protected facets, 3 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Map delegated oracle evidence: DELEGATE-006.

### CORE-004 — TestCurrentBundledBytes

Disposition: migrate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-009 — TestCurrentBundledCodec

Disposition: migrate. Start after: P15, P27, P28, K04, K05. Accept after: P23, QN2.

- [ ] Account for 1 protected facets, 1 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-010 — TestCurrentBundledCollections

Disposition: migrate. Start after: P15, P27, P28, K04, K05. Accept after: P23, QN2.

- [ ] Account for 4 protected facets, 2 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Map delegated oracle evidence: DELEGATE-008.

## M04 — Coordination, occurrence identity and ownership

- [ ] **M04: planned** — owner lane: `migration-core`.

Proposed owner: `tests/native-can/migration/m04-coordination-occurrence-identi/**`. Group start after: P15, P27, P28, K01, K02, K03, K04, K05, K06. Group accept after: P23, QN1, QN2, QN3. Logical gates: SUITE, N1, N2, N3, CHILD.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Preserve all/allSettled/any/race selection, losing diagnostics, same occurrence identity and lease lifetime with terminal event seals.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Preserve all/allSettled/any/race selection, losing diagnostics, same occurrence identity and lease lifetime with terminal event seals. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: DELEGATE-009, DELEGATE-014, DELEGATE-015, DELEGATE-016; shared delegate references: none.

Suggested first small commit: `test(native-can): add coordination-occurrence-identi cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### CORE-012 — TestCurrentBundledCoordination

Disposition: migrate. Start after: P15, P27, P28, K01, K02, K03, K04, K05, K06. Accept after: P23, QN1, QN2, QN3.

- [ ] Account for 2 protected facets, 2 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Map delegated oracle evidence: DELEGATE-009.

### CORE-013 — TestStandardSnapshotIdentity

Disposition: migrate. Start after: P15, P27, P28, K01, K02, K03, K06. Accept after: P23, QN1, QN3.

- [ ] Account for 1 protected facets, 1 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-033 — TestBundledOwnershipOffline

Disposition: migrate. Start after: P15, P28, K04, K05, K06. Accept after: P23, QN2, QN3.

- [ ] Account for 3 protected facets, 1 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Map delegated oracle evidence: DELEGATE-014, DELEGATE-015, DELEGATE-016.

## M05 — Exact amounts, numbers, text and URL time

- [ ] **M05: planned** — owner lane: `migration-core`.

Proposed owner: `tests/native-can/migration/m05-exact-amounts-numbers-text-and/**`. Group start after: P15, P27, P28, P20. Group accept after: P23. Logical gates: SUITE, N1, N2.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Compare exact numeric bits/lexemes, rounding, Unicode/regex and URL/date cases; map delegated declarations instead of old pass counts.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Compare exact numeric bits/lexemes, rounding, Unicode/regex and URL/date cases; map delegated declarations instead of old pass counts. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: DELEGATE-010, DELEGATE-011, DELEGATE-013, DELEGATE-022, DELEGATE-030; shared delegate references: none.

Suggested first small commit: `test(native-can): add exact-amounts-numbers-text-and cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### CORE-017 — TestCurrentBundledExactAmounts

Disposition: migrate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 2 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Map delegated oracle evidence: DELEGATE-011.

### CORE-032 — TestCurrentBundledNumbers

Disposition: migrate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 2 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Map delegated oracle evidence: DELEGATE-013.

### CORE-042 — TestCurrentBundledText

Disposition: migrate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 2 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Map delegated oracle evidence: DELEGATE-022.

### CORE-045 — TestCurrentBundledUtilities

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 3 protected facets, 3 listed variants and 4 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Map delegated oracle evidence: DELEGATE-010, DELEGATE-030.

## M06 — Crypto, files and streams

- [ ] **M06: planned** — owner lane: `migration-core`.

Proposed owner: `tests/native-can/migration/m06-crypto-files-and-streams/**`. Group start after: P15, P20, P27, P28, K01, K02, K03. Group accept after: P23, QN1. Logical gates: SUITE, WORK, PROC, N1.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Run native commands and hostile file/stream cases with independent bytes, EOF/error, symlink and bounded cleanup facts.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Run native commands and hostile file/stream cases with independent bytes, EOF/error, symlink and bounded cleanup facts. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: DELEGATE-012, DELEGATE-021; shared delegate references: none.

Suggested first small commit: `test(native-can): add crypto-files-and-streams cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### CORE-014 — TestCurrentCryptoCommands

Disposition: migrate. Start after: P15, P20, P27, P28, K01, K02, K03. Accept after: P23, QN1.

- [ ] Account for 3 protected facets, 2 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-019 — TestCurrentBundledFiles

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 4 protected facets, 3 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Map delegated oracle evidence: DELEGATE-012.

### CORE-040 — TestCurrentBundledStreams

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 3 protected facets, 4 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Map delegated oracle evidence: DELEGATE-021.

## M07 — Fetch, formats, generation and consumer wrappers

- [ ] **M07: planned** — owner lane: `migration-core`.

Proposed owner: `tests/native-can/migration/m07-fetch-formats-generation-and-c/**`. Group start after: P15, P20, P27, P28, K20, K01, K02, K03. Group accept after: P23, QHTTP, QN1. Logical gates: SUITE, HTTP, PEER, WIRE, BUILD.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Use owned peers/provider fixtures to test raw requests, document formats, HTML, generation and layered recovery with Can-selected expected bytes.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Use owned peers/provider fixtures to test raw requests, document formats, HTML, generation and layered recovery with Can-selected expected bytes. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add fetch-formats-generation-and-c cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### CORE-018 — TestCurrentBundledFetch

Disposition: migrate. Start after: P15, P20, P27, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 4 protected facets, 3 listed variants and 4 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-021 — TestCurrentFormatsDocuments

Disposition: migrate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 3 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-022 — TestCurrentBundledGeneration

Disposition: migrate. Start after: P15, P20, P27, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 4 protected facets, 2 listed variants and 4 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-025 — TestCurrentBundledHTML

Disposition: migrate. Start after: P15, P27, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 2 protected facets, 1 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-029 — TestCurrentBundledNoulJudge

Disposition: migrate. Start after: P15, P20, P27, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 3 protected facets, 1 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-035 — TestCurrentBundledMixedQuestions

Disposition: migrate. Start after: P15, P20, P27, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 4 protected facets, 1 listed variants and 4 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-048 — TestCurrentBundledWrappers

Disposition: migrate. Start after: P15, P20, P27, P28, K20, K01, K02, K03. Accept after: P23, QHTTP, QN1.

- [ ] Account for 3 protected facets, 1 listed variants and 4 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M08 — HTTP servers, TLS, WebSocket and transport

- [ ] **M08: planned** — owner lane: `migration-core`.

Proposed owner: `tests/native-can/migration/m08-http-servers-tls-websocket-and/**`. Group start after: P15, P27, P28, K20, K21. Group accept after: P23, QHTTP, QWS. Logical gates: SUITE, CHILD, HTTP, PEER, WIRE, CLOCK.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Preserve server/cookie/CSRF/TLS/WS effects, malformed transport, readiness and graceful versus forced shutdown with complete native receipts.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Preserve server/cookie/CSRF/TLS/WS effects, malformed transport, readiness and graceful versus forced shutdown with complete native receipts. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: DELEGATE-023, DELEGATE-024, DELEGATE-025, DELEGATE-026, DELEGATE-027, DELEGATE-028, DELEGATE-029; shared delegate references: none.

Suggested first small commit: `test(native-can): add http-servers-tls-websocket-and cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### CORE-011 — TestCurrentBundledCookiesCSRF

Disposition: migrate. Start after: P15, P27, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 2 protected facets, 1 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-026 — TestCurrentBundledHTTP

Disposition: migrate. Start after: P15, P27, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 3 protected facets, 4 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-038 — TestCurrentBundledServer

Disposition: migrate. Start after: P15, P27, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 2 protected facets, 1 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-043 — TestCurrentBundledTLS

Disposition: migrate. Start after: P15, P27, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 2 protected facets, 3 listed variants and 4 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-044 — TestBundledTransportLoopback

Disposition: migrate. Start after: P15, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 3 protected facets, 1 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Map delegated oracle evidence: DELEGATE-023, DELEGATE-024, DELEGATE-025, DELEGATE-026, DELEGATE-027, DELEGATE-028, DELEGATE-029.

### CORE-047 — TestCurrentBundledWebSocket

Disposition: migrate. Start after: P15, P27, P28, K20, K21. Accept after: P23, QHTTP, QWS.

- [ ] Account for 2 protected facets, 1 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M09 — CLI, formatter, input and process fixtures

- [ ] **M09: planned** — owner lane: `migration-core`.

Proposed owner: `tests/native-can/migration/m09-cli-formatter-input-and-proces/**`. Group start after: P15, P20, P27, P28, K17, K18, K01, K02, K03, K07, K08, K09. Group accept after: P23, QF1, QN1, QB0. Logical gates: SUITE, PROC, ENV, F1, F2, DIAG.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Exercise argv/env/fd maps, formatter identity, input capture, native declarations, process descendants and maintained project discovery.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Exercise argv/env/fd maps, formatter identity, input capture, native declarations, process descendants and maintained project discovery. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: DELEGATE-017; shared delegate references: none.

Suggested first small commit: `test(native-can): add cli-formatter-input-and-proces cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### CORE-008 — TestCurrentBundledCLI

Disposition: migrate. Start after: P15, P20, P27, P28, K17, K18. Accept after: P23, QF1.

- [ ] Account for 3 protected facets, 4 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-020 — TestFormatPreservesExecution

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 3 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-027 — TestCurrentBundledInputCapture

Disposition: migrate. Start after: P15, P20, P27, P28, K01, K02, K03. Accept after: P23, QN1.

- [ ] Account for 3 protected facets, 2 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-028 — TestCurrentBundledInputEnvironment

Disposition: migrate. Start after: P15, P20, P27, P28, K17, K18. Accept after: P23, QF1.

- [ ] Account for 4 protected facets, 2 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-030 — TestCurrentMarkdownRender

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 3 protected facets, 1 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-031 — TestCurrentBundledNativeDeclarations

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 1 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-034 — TestCurrentBundledProcess

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 4 protected facets, 3 listed variants and 4 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Map delegated oracle evidence: DELEGATE-017.

### CORE-039 — TestStdlibMaintained

Disposition: migrate. Start after: P15, P27, P28, K07, K08, K09. Accept after: P23, QB0.

- [ ] Account for 4 protected facets, 3 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M10 — Distribution, verified builds and release

- [ ] **M10: planned** — owner lane: `migration-core`.

Proposed owner: `tests/native-can/migration/m10-distribution-verified-builds-a/**`. Group start after: P15, P20, P27, P28, K17, K19, K20. Group accept after: P23, QF2, QHTTP. Logical gates: SUITE, BUILD, ARCHIVE, ENV, DIAG.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Check packaged Bun/runtime identity, deterministic verified build, install/update refusal, artifact manifests and strict emitted TypeScript where required.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Check packaged Bun/runtime identity, deterministic verified build, install/update refusal, artifact manifests and strict emitted TypeScript where required. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: DELEGATE-001, DELEGATE-002, DELEGATE-003, DELEGATE-004, DELEGATE-005; shared delegate references: none.

Suggested first small commit: `test(native-can): add distribution-verified-builds-a cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### CORE-015 — TestDevelopmentSidecar

Disposition: migrate. Start after: P15, P20, P27, P28, K17, K19. Accept after: P23, QF2.

- [ ] Account for 3 protected facets, 3 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Map delegated oracle evidence: DELEGATE-001, DELEGATE-002, DELEGATE-003, DELEGATE-004, DELEGATE-005.

### CORE-016 — TestDistributionShipsBrowserBundleTool

Disposition: migrate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-036 — TestReleaseInstallUpdate

Disposition: migrate. Start after: P15, P27, P28, K17. Accept after: P23.

- [ ] Account for 3 protected facets, 2 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-037 — TestInstallRootRefusals

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-046 — TestCurrentBundledVerifiedBuild

Disposition: migrate. Start after: P15, P20, P27, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 6 protected facets, 1 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M11 — Raw fixture staging mechanics

- [ ] **M11: planned** — owner lane: `migration-core`.

Proposed owner: `tests/native-can/migration/m11-raw-fixture-staging-mechanics/**`. Group start after: P15, P28. Group accept after: P23. Logical gates: FILES, WORK, N1.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Replace authorization/environment fixture transformations with typed Can-selected byte operations; preserve exact fixture bytes and malformed-fixture refusal.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Replace authorization/environment fixture transformations with typed Can-selected byte operations; preserve exact fixture bytes and malformed-fixture refusal. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add raw-fixture-staging-mechanics cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### CORE-049 — stageRawFixtures

Disposition: replace_support. Start after: P15, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-050 — stripStagedAuthorization

Disposition: replace_support. Start after: P15, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### CORE-051 — clearStagedFixtureEnvironments

Disposition: replace_support. Start after: P15, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M12 — Application server and browser cases

- [ ] **M12: planned** — owner lane: `migration-browser`.

Proposed owner: `tests/native-can/migration/m12-application-server-and-browser/**`. Group start after: P15, P27, P28, K20, K22, K07, K08, K09, K10, K11, K17, K16. Group accept after: P23, QHTTP, QD1, QB0, QB1base, QB5. Logical gates: B1, B5, DB, HTTP, CHILD.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Stage four apps, run required service and browser variants, and qualify local/remote Firefox context ownership without treating absent services as passes.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Stage four apps, run required service and browser variants, and qualify local/remote Firefox context ownership without treating absent services as passes. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add application-server-and-browser cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### BROWSER-001 — TestApplicationsStaged

Disposition: migrate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 4 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-002 — TestApplicationsStagedForms

Disposition: migrate. Start after: P15, P27, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 2 protected facets, 4 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-003 — TestApplicationsNativeAIStubbed

Disposition: migrate. Start after: P15, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 2 protected facets, 3 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-004 — TestApplicationsLive

Disposition: migrate. Start after: P15, P28, K20, K22. Accept after: P23, QHTTP, QD1.

- [ ] Account for 2 protected facets, 4 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-005 — TestApplicationsBrowser

Disposition: migrate. Start after: P15, P28, K20, K07, K08, K09, K22, K10, K11. Accept after: P23, QHTTP, QB0, QD1, QB1base.

- [ ] Account for 2 protected facets, 2 listed variants and 4 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-048 — firefox-remote.mjs

Disposition: replace_support. Start after: P15, P28, K17, K07, K08, K09, K16. Accept after: P23, QB0, QB5.

- [ ] Account for 1 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M13 — Assets and browser target builds

- [ ] **M13: planned** — owner lane: `migration-browser`.

Proposed owner: `tests/native-can/migration/m13-assets-and-browser-target-buil/**`. Group start after: P15, P20, P27, P28, K20, K07, K08, K09. Group accept after: P23, QHTTP, QB0. Logical gates: BUILD, BROWSER, FILES.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Check asset pairing, CSP/local bytes and target isolation; review old bundler and SHA shim relevance before any conditional retirement.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Check asset pairing, CSP/local bytes and target isolation; review old bundler and SHA shim relevance before any conditional retirement. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add assets-and-browser-target-buil cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: Conditional rows require per-row current-contract relevance and reviewer disposition; preserve historical records/static fixtures where still useful. The integrator alone may retire shared old files after all linked rows and callers are resolved.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### BROWSER-006 — TestCurrentBundledAssets

Disposition: migrate. Start after: P15, P20, P27, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 2 protected facets, 2 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-007 — TestCurrentBrowserAssets

Disposition: migrate. Start after: P15, P27, P28, K20, K07, K08, K09. Accept after: P23, QHTTP, QB0.

- [ ] Account for 1 protected facets, 1 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-008 — TestCurrentPairedAssets

Disposition: migrate. Start after: P15, P27, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 2 protected facets, 5 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-009 — TestBrowserBuildTarget

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 3 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-037 — build-grid.mjs

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### BROWSER-053 — sha256-shim.mjs

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 2 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

## M14 — Browser wire codec parity

- [ ] **M14: planned** — owner lane: `migration-browser`.

Proposed owner: `tests/native-can/migration/m14-browser-wire-codec-parity/**`. Group start after: P15, P27, P28, K07, K08, K09, K15. Group accept after: P23, QB0, QB4. Logical gates: B4, N1, BUILD.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Replace bundled TypeScript vector oracle with Can-owned vectors; compare Bun/browser facts and reject unexpected node builtins.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Replace bundled TypeScript vector oracle with Can-owned vectors; compare Bun/browser facts and reject unexpected node builtins. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add browser-wire-codec-parity cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### BROWSER-010 — TestBrowserWireCodecParity

Disposition: migrate. Start after: P15, P27, P28, K07, K08, K09, K15. Accept after: P23, QB0, QB4.

- [ ] Account for 2 protected facets, 3 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-038 — build-vectors.mjs

Disposition: replace_support. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-039 — codec-parity.mjs

Disposition: migrate. Start after: P15, P28, K07, K08, K09, K15. Accept after: P23, QB0, QB4.

- [ ] Account for 2 protected facets, 3 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-040 — codec-vectors-entry.ts

Disposition: replace_support. Start after: P15, P28, K07, K08, K09, K15. Accept after: P23, QB0, QB4.

- [ ] Account for 2 protected facets, 1 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M15 — Native DOM and emitted controls

- [ ] **M15: planned** — owner lane: `migration-browser`.

Proposed owner: `tests/native-can/migration/m15-native-dom-and-emitted-control/**`. Group start after: P15, P28, K20, K07, K08, K09, K13, P27, K22, K01, K02, K03. Group accept after: P23, QHTTP, QB0, QB2, QD1, QN1. Logical gates: B2, B3, BROWSER, NATIVE.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Compare independent IDL/event/caret/file facts to emitted Can echoes, preserving physical versus synthetic input and same-task ordering controls.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Compare independent IDL/event/caret/file facts to emitted Can echoes, preserving physical versus synthetic input and same-task ordering controls. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add native-dom-and-emitted-control cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### BROWSER-011 — TestC02NativeMatrix

Disposition: migrate. Start after: P15, P28, K20, K07, K08, K09, K13. Accept after: P23, QHTTP, QB0, QB2.

- [ ] Account for 2 protected facets, 3 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-012 — TestC02ControlsMatrix

Disposition: migrate. Start after: P15, P27, P28, K07, K08, K09, K22, K01, K02, K03, K13. Accept after: P23, QB0, QD1, QN1, QB2.

- [ ] Account for 2 protected facets, 4 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-043 — controls-native.mjs

Disposition: migrate. Start after: P15, P28, K07, K08, K09, K13. Accept after: P23, QB0, QB2.

- [ ] Account for 3 protected facets, 3 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-044 — controls.mjs

Disposition: migrate. Start after: P15, P28, K20, K07, K08, K09, K01, K02, K03, K13. Accept after: P23, QHTTP, QB0, QN1, QB2.

- [ ] Account for 3 protected facets, 3 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M16 — Grid and compare contract build matrix

- [ ] **M16: planned** — owner lane: `migration-browser`.

Proposed owner: `tests/native-can/migration/m16-grid-and-compare-contract-buil/**`. Group start after: P15, P27, P28, P20, K07, K08, K09, K22, K10, K11, K20. Group accept after: P23, QB0, QD1, QB1base, QHTTP. Logical gates: BUILD, DIAG, B1, DB.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Keep real Can roots, contract edit diagnostics, paired identities and required-engine preflight distinct; reject stale imports and wrong target entry.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Keep real Can roots, contract edit diagnostics, paired identities and required-engine preflight distinct; reject stale imports and wrong target entry. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add grid-and-compare-contract-buil cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### BROWSER-013 — TestC06Positive

Disposition: migrate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-014 — TestC06NegativeAPIBreak

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 3 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-015 — TestC06CaptureEdits

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 6 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-016 — TestC06ServedMatrix

Disposition: migrate. Start after: P15, P27, P28, K07, K08, K09, K22, K10, K11. Accept after: P23, QB0, QD1, QB1base.

- [ ] Account for 2 protected facets, 7 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-017 — TestGate5ServedMatrix

Disposition: migrate. Start after: P15, P27, P28, K20, K07, K08, K09, K22, K10, K11. Accept after: P23, QHTTP, QB0, QD1, QB1base.

- [ ] Account for 3 protected facets, 6 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-018 — TestGate5GridStatic

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M17 — Invoice contract mutation matrix

- [ ] **M17: planned** — owner lane: `migration-browser`.

Proposed owner: `tests/native-can/migration/m17-invoice-contract-mutation-matr/**`. Group start after: P15, P27, P28, K20, K07, K08, K09, K22, P20. Group accept after: P23, QHTTP, QB0, QD1. Logical gates: BUILD, DIAG, HTTP, DB, B1.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Exercise route/capture/field/leaf/status/body-mode edits and stale manifest pairings with exact HTTP/DB outcomes and noncommit controls.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Exercise route/capture/field/leaf/status/body-mode edits and stale manifest pairings with exact HTTP/DB outcomes and noncommit controls. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add invoice-contract-mutation-matr cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### BROWSER-019 — TestInvoiceContractRoute

Disposition: migrate. Start after: P15, P27, P28, K20, K07, K08, K09, K22. Accept after: P23, QHTTP, QB0, QD1.

- [ ] Account for 2 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-020 — TestInvoiceContractCapture

Disposition: migrate. Start after: P15, P27, P28, K20, K07, K08, K09, K22. Accept after: P23, QHTTP, QB0, QD1.

- [ ] Account for 2 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-021 — TestInvoiceContractField

Disposition: migrate. Start after: P15, P27, P28, K20, K07, K08, K09, K22. Accept after: P23, QHTTP, QB0, QD1.

- [ ] Account for 2 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-022 — TestInvoiceContractLeaf

Disposition: migrate. Start after: P15, P27, P28, K20, K07, K08, K09, K22. Accept after: P23, QHTTP, QB0, QD1.

- [ ] Account for 2 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-023 — TestInvoiceContractStatus

Disposition: migrate. Start after: P15, P27, P28, K20, K07, K08, K09, K22. Accept after: P23, QHTTP, QB0, QD1.

- [ ] Account for 2 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-024 — TestInvoiceContractLimit

Disposition: migrate. Start after: P15, P27, P28, K20, K07, K08, K09, K22. Accept after: P23, QHTTP, QB0, QD1.

- [ ] Account for 2 protected facets, 3 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-025 — TestInvoiceContractBodyMode

Disposition: migrate. Start after: P15, P20, P27, P28, K20, K22. Accept after: P23, QHTTP, QD1.

- [ ] Account for 2 protected facets, 4 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-026 — TestInvoiceContractManifestMismatch

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-027 — TestInvoiceContractNoRegistration

Disposition: migrate. Start after: P15, P27, P28, K20, K22. Accept after: P23, QHTTP, QD1.

- [ ] Account for 2 protected facets, 4 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M18 — Invoice live server, guard and DB fixtures

- [ ] **M18: planned** — owner lane: `migration-browser`.

Proposed owner: `tests/native-can/migration/m18-invoice-live-server-guard-and-/**`. Group start after: P15, P27, P28, K20, K22, K17, K18, K07, K08, K09, K10, K11, K26, K13. Group accept after: P23, QHTTP, QD1, QF1, QB0, QB1base, QB1, QD4, QB2. Logical gates: BUILD, B1, B2, DB, F1.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Use owned SQLite/PG seeds and direct reads; verify startup refusal, escaped fragments, guard occurrence matrix and full browser reports.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Use owned SQLite/PG seeds and direct reads; verify startup refusal, escaped fragments, guard occurrence matrix and full browser reports. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add invoice-live-server-guard-and- cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### BROWSER-028 — TestInvoiceGridStagedBrowserBuild

Disposition: migrate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-029 — TestInvoiceFormLive

Disposition: migrate. Start after: P15, P27, P28, K20, K22. Accept after: P23, QHTTP, QD1.

- [ ] Account for 3 protected facets, 4 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-030 — TestInvoiceStartupRefusal

Disposition: migrate. Start after: P15, P28, K17, K18. Accept after: P23, QF1.

- [ ] Account for 1 protected facets, 3 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-031 — TestInvoiceGridPagePaired

Disposition: migrate. Start after: P15, P27, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 2 protected facets, 3 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-032 — TestInvoiceBrowser

Disposition: migrate. Start after: P15, P28, K07, K08, K09, K22, K10, K11, K26. Accept after: P23, QB0, QD1, QB1base, QB1, QD4.

- [ ] Account for 2 protected facets, 2 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-033 — TestInvoiceHTMLFragments

Disposition: migrate. Start after: P15, P28, K20. Accept after: P23, QHTTP.

- [ ] Account for 2 protected facets, 6 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-034 — TestInvoiceBrowserGuardDOM

Disposition: migrate. Start after: P15, P28, K07, K08, K09, K13. Accept after: P23, QB0, QB2.

- [ ] Account for 2 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-055 — invoice/driver.ts

Disposition: replace_support. Start after: P15, P28, K22. Accept after: P23, QD1.

- [ ] Account for 2 protected facets, 9 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-056 — applications/seed-driver.ts

Disposition: replace_support. Start after: P15, P28, K17, K22. Accept after: P23, QD1.

- [ ] Account for 1 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M19 — Accounts, assets, dashboard and forms UI

- [ ] **M19: planned** — owner lane: `migration-browser`.

Proposed owner: `tests/native-can/migration/m19-accounts-assets-dashboard-and-/**`. Group start after: P15, P28, K20, K07, K08, K09, K22, K10, K11. Group accept after: P23, QHTTP, QB0, QD1, QB1base. Logical gates: B1, B2, BROWSER, HTTP.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Reauthor HTMX, CSP, quiet/swap, escaping and polling sequences in Can with independent DOM/network facts and terminal seals.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Reauthor HTMX, CSP, quiet/swap, escaping and polling sequences in Can with independent DOM/network facts and terminal seals. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add accounts-assets-dashboard-and- cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### BROWSER-035 — accounts.mjs

Disposition: migrate. Start after: P15, P28, K20, K07, K08, K09, K22, K10, K11. Accept after: P23, QHTTP, QB0, QD1, QB1base.

- [ ] Account for 3 protected facets, 3 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-036 — assets.mjs

Disposition: migrate. Start after: P15, P28, K20, K07, K08, K09, K10, K11. Accept after: P23, QHTTP, QB0, QB1base.

- [ ] Account for 3 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-045 — dashboard.mjs

Disposition: migrate. Start after: P15, P28, K20, K07, K08, K09, K22, K10, K11. Accept after: P23, QHTTP, QB0, QD1, QB1base.

- [ ] Account for 2 protected facets, 2 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-049 — forms.mjs

Disposition: migrate. Start after: P15, P28, K20, K07, K08, K09, K10, K11. Accept after: P23, QHTTP, QB0, QB1base.

- [ ] Account for 3 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M20 — Grid, compare, drift and conformance UI

- [ ] **M20: planned** — owner lane: `migration-browser`.

Proposed owner: `tests/native-can/migration/m20-grid-compare-drift-and-conform/**`. Group start after: P15, P28, K20, K07, K08, K09, K10, K11, P27, K22, K13, K14. Group accept after: P23, QHTTP, QB0, QB1base, QD1, QB2, QB3, QB1. Logical gates: B1, B2, B3, B4, DB.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Reauthor actions for grid/compare/drift/conformance/empty clients; check generated identity, replay/conflict, real keys and full engine variants.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Reauthor actions for grid/compare/drift/conformance/empty clients; check generated identity, replay/conflict, real keys and full engine variants. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add grid-compare-drift-and-conform cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### BROWSER-041 — compare.mjs

Disposition: migrate. Start after: P15, P28, K20, K07, K08, K09, K10, K11. Accept after: P23, QHTTP, QB0, QB1base.

- [ ] Account for 3 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-042 — conformance.mjs

Disposition: migrate. Start after: P15, P28, K20, K07, K08, K09, K10, K11. Accept after: P23, QHTTP, QB0, QB1base.

- [ ] Account for 3 protected facets, 3 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-046 — drift.mjs

Disposition: migrate. Start after: P15, P28, K20, K07, K08, K09, K10, K11. Accept after: P23, QHTTP, QB0, QB1base.

- [ ] Account for 3 protected facets, 3 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-047 — empty.mjs

Disposition: migrate. Start after: P15, P27, P28, K20, K07, K08, K09. Accept after: P23, QHTTP, QB0.

- [ ] Account for 3 protected facets, 1 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-050 — grid.mjs

Disposition: migrate. Start after: P15, P28, K20, K07, K08, K09, K22, K13, K14, K10, K11. Accept after: P23, QHTTP, QB0, QD1, QB2, QB3, QB1base, QB1.

- [ ] Account for 4 protected facets, 3 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-054 — w1-grid.mjs

Disposition: migrate. Start after: P15, P28, K20, K07, K08, K09, K10, K11. Accept after: P23, QHTTP, QB0, QB1base.

- [ ] Account for 3 protected facets, 3 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M21 — Invoice contract and form UI

- [ ] **M21: planned** — owner lane: `migration-browser`.

Proposed owner: `tests/native-can/migration/m21-invoice-contract-and-form-ui/**`. Group start after: P15, P28, K20, K07, K08, K09, K22, K13, K14, K10, K11, K26. Group accept after: P23, QHTTP, QB0, QD1, QB2, QB3, QB1base, QB1, QD4. Logical gates: B1, B2, B3, DB.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Reauthor invoice browser choreography, delayed/corrupt delivery, swaps, draft/focus retention and independent settled DB effects.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Reauthor invoice browser choreography, delayed/corrupt delivery, swaps, draft/focus retention and independent settled DB effects. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add invoice-contract-and-form-ui cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### BROWSER-051 — invoice-contract.mjs

Disposition: migrate. Start after: P15, P28, K20, K07, K08, K09, K22. Accept after: P23, QHTTP, QB0, QD1.

- [ ] Account for 3 protected facets, 4 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### BROWSER-052 — invoice.mjs

Disposition: migrate. Start after: P15, P28, K20, K07, K08, K09, K22, K13, K14, K10, K11, K26. Accept after: P23, QHTTP, QB0, QD1, QB2, QB3, QB1base, QB1, QD4.

- [ ] Account for 3 protected facets, 4 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M22 — PostgreSQL query semantics

- [ ] **M22: planned** — owner lane: `migration-history`.

Proposed owner: `tests/native-can/migration/m22-postgresql-query-semantics/**`. Group start after: P15, P20, P27, P28, K17, K22, K18. Group accept after: P23, QD1, QF1. Logical gates: D1, DB, SUITE.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Move SQL setup and query answers into Can; verify optional/one/many/execute, null/bytes/limits, hostile binds and native rows.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Move SQL setup and query answers into Can; verify optional/one/many/execute, null/bytes/limits, hostile binds and native rows. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add postgresql-query-semantics cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: Conditional rows require per-row current-contract relevance and reviewer disposition; preserve historical records/static fixtures where still useful. The integrator alone may retire shared old files after all linked rows and callers are resolved.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### HISTORY-001 — TestCurrentSQLQueries

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 3 protected facets, 4 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-002 — TestCurrentSQLQueriesLive

Disposition: migrate. Start after: P15, P28, K17, K22, K18. Accept after: P23, QD1, QF1.

- [ ] Account for 2 protected facets, 6 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-015 — queries-driver.ts

Disposition: migrate. Start after: P15, P28, K22. Accept after: P23, QD1.

- [ ] Account for 2 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-020 — queries.sql

Disposition: retain_fixture. Start after: P15. Accept after: reviewed disposition. If retained live: QD1.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Confirm exact static fixture bytes, named consumer and starting-state assumptions.
- [ ] Keep or move the fixture as data; do not count it as a Can-authored scenario or delete it with a driver.

## M23 — Transactions and persistence

- [ ] **M23: planned** — owner lane: `migration-history`.

Proposed owner: `tests/native-can/migration/m23-transactions-and-persistence/**`. Group start after: P15, P20, P27, P28, K22, K17, K18. Group accept after: P23, QD1, QF1. Logical gates: D1, D2, DB, SUITE.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Prove commit/rollback across compiled calls, pinned transaction identity and independent final rows using owned PostgreSQL namespace.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Prove commit/rollback across compiled calls, pinned transaction identity and independent final rows using owned PostgreSQL namespace. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add transactions-and-persistence cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: Conditional rows require per-row current-contract relevance and reviewer disposition; preserve historical records/static fixtures where still useful. The integrator alone may retire shared old files after all linked rows and callers are resolved.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### HISTORY-003 — TestCurrentSQLTransactions

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 1 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-004 — TestCurrentSQLTransactionsLive

Disposition: migrate. Start after: P15, P28, K22, K17, K18. Accept after: P23, QD1, QF1.

- [ ] Account for 2 protected facets, 6 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-017 — transactions-driver.ts

Disposition: migrate. Start after: P15, P28, K22. Accept after: P23, QD1.

- [ ] Account for 2 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-022 — transactions.sql

Disposition: retain_fixture. Start after: P15. Accept after: reviewed disposition. If retained live: QD1.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Confirm exact static fixture bytes, named consumer and starting-state assumptions.
- [ ] Keep or move the fixture as data; do not count it as a Can-authored scenario or delete it with a driver.

## M24 — Descriptor SQL wiring

- [ ] **M24: planned** — owner lane: `migration-history`.

Proposed owner: `tests/native-can/migration/m24-descriptor-sql-wiring/**`. Group start after: P15, P20, P27, P28, K22, K17, K18. Group accept after: P23, QD1, QF1. Logical gates: D1, DB, BUILD.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Invoke emitted descriptors with repeat placeholders and hostile text; compare exact operations to direct-driver facts, preserving static SQL fixture content.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Invoke emitted descriptors with repeat placeholders and hostile text; compare exact operations to direct-driver facts, preserving static SQL fixture content. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add descriptor-sql-wiring cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: Conditional rows require per-row current-contract relevance and reviewer disposition; preserve historical records/static fixtures where still useful. The integrator alone may retire shared old files after all linked rows and callers are resolved.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### HISTORY-005 — TestCurrentSQLDescriptors

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 1 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-006 — TestCurrentSQLDescriptorsLive

Disposition: migrate. Start after: P15, P28, K22, K17, K18. Accept after: P23, QD1, QF1.

- [ ] Account for 3 protected facets, 7 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-011 — descriptors-driver.ts

Disposition: migrate. Start after: P15, P28, K22. Accept after: P23, QD1.

- [ ] Account for 2 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-018 — descriptors.sql

Disposition: retain_fixture. Start after: P15. Accept after: reviewed disposition. If retained live: QD1.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Confirm exact static fixture bytes, named consumer and starting-state assumptions.
- [ ] Keep or move the fixture as data; do not count it as a Can-authored scenario or delete it with a driver.

## M25 — F02 locking and F03 relational slice

- [ ] **M25: planned** — owner lane: `migration-history`.

Proposed owner: `tests/native-can/migration/m25-f02-locking-and-f03-relational/**`. Group start after: P15, P28, K22, K24, K01, K02, K03, K25, K26. Group accept after: P23, QD1, QD2, QN1, QD3, QD4. Logical gates: D1, D2, D3, D4, DB.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Retain lock/RETURNING/error facets; add sentinel rollback and confirmed settlement controls while labeling them stronger than old evidence.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Retain lock/RETURNING/error facets; add sentinel rollback and confirmed settlement controls while labeling them stronger than old evidence. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add f02-locking-and-f03-relational cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### HISTORY-007 — TestF02LiveLockingAndReturning

Disposition: migrate. Start after: P15, P28, K22, K24. Accept after: P23, QD1, QD2.

- [ ] Account for 3 protected facets, 6 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Keep measurement facets deferred until explicitly authorized; do not invent or substitute a threshold.

### HISTORY-008 — TestF03LiveRelationalSlice

Disposition: migrate. Start after: P15, P28, K22, K01, K02, K03, K24, K25, K26. Accept after: P23, QD1, QN1, QD2, QD3, QD4.

- [ ] Account for 3 protected facets, 7 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-012 — f02-live-driver.ts

Disposition: migrate. Start after: P15, P28, K22, K24. Accept after: P23, QD1, QD2.

- [ ] Account for 2 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Keep measurement facets deferred until explicitly authorized; do not invent or substitute a threshold.

### HISTORY-013 — f03-live-driver.ts

Disposition: migrate. Start after: P15, P28, K22, K24, K25, K26. Accept after: P23, QD1, QD2, QD3, QD4.

- [ ] Account for 2 protected facets, 7 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M26 — MySQL and SQLite persistence

- [ ] **M26: planned** — owner lane: `migration-history`.

Proposed owner: `tests/native-can/migration/m26-mysql-and-sqlite-persistence/**`. Group start after: P15, P28, K22, K17, K18. Group accept after: P23, QD1, QF1. Logical gates: D1, D2, DB, F1.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Run dialect-specific Can assertions and owned seed/cleanup; keep same-connection MySQL identity and exact SQLite rows explicit.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Run dialect-specific Can assertions and owned seed/cleanup; keep same-connection MySQL identity and exact SQLite rows explicit. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add mysql-and-sqlite-persistence cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: Conditional rows require per-row current-contract relevance and reviewer disposition; preserve historical records/static fixtures where still useful. The integrator alone may retire shared old files after all linked rows and callers are resolved.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### HISTORY-009 — TestCurrentMySQLPersistence

Disposition: migrate. Start after: P15, P28, K22, K17, K18. Accept after: P23, QD1, QF1.

- [ ] Account for 2 protected facets, 5 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-010 — TestCurrentSQLitePersistence

Disposition: migrate. Start after: P15, P28, K22, K17, K18. Accept after: P23, QD1, QF1.

- [ ] Account for 2 protected facets, 5 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-014 — mysql-driver.ts

Disposition: migrate. Start after: P15, P28, K22. Accept after: P23, QD1.

- [ ] Account for 2 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-016 — sqlite-driver.ts

Disposition: migrate. Start after: P15, P28, K22. Accept after: P23, QD1.

- [ ] Account for 2 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-019 — mysql-seed.sql

Disposition: retain_fixture. Start after: P15. Accept after: reviewed disposition. If retained live: QD1.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Confirm exact static fixture bytes, named consumer and starting-state assumptions.
- [ ] Keep or move the fixture as data; do not count it as a Can-authored scenario or delete it with a driver.

### HISTORY-021 — sqlite-seed.sql

Disposition: retain_fixture. Start after: P15. Accept after: reviewed disposition. If retained live: QD1.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Confirm exact static fixture bytes, named consumer and starting-state assumptions.
- [ ] Keep or move the fixture as data; do not count it as a Can-authored scenario or delete it with a driver.

## M27 — Owner factory and isolation conventions

- [ ] **M27: planned** — owner lane: `migration-history`.

Proposed owner: `tests/native-can/migration/m27-owner-factory-and-isolation-co/**`. Group start after: P15, P20, P28. Group accept after: P23. Logical gates: SUITE, DIAG, FAULT.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Recreate green/tainted roots, private factory boundary, forged-record rejection and guided repair diagnostics as Can cases.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Recreate green/tainted roots, private factory boundary, forged-record rejection and guided repair diagnostics as Can cases. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add owner-factory-and-isolation-co cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### HISTORY-023 — TestOwnerSetupGreen

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-024 — TestOwnerNegativeFailsClosed

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-025 — TestOwnerForgeRejected

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-026 — TestFactoryPrivateToPackage

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-027 — TestFactoryRepairGuided

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M28 — Retry convention and mutation controls

- [ ] **M28: planned** — owner lane: `migration-history`.

Proposed owner: `tests/native-can/migration/m28-retry-convention-and-mutation-/**`. Group start after: P15, P20, P28. Group accept after: P23. Logical gates: SUITE, FAULT.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Preserve FIFO fixture consumption, fixed/result-data alternatives, no-retry and over-attempt mutants, and unrelated-file unchanged evidence.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Preserve FIFO fixture consumption, fixed/result-data alternatives, no-retry and over-attempt mutants, and unrelated-file unchanged evidence. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add retry-convention-and-mutation- cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### HISTORY-028 — TestRetryConventionGreen

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-029 — TestRetryFixedGreen

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-030 — TestRetryNoRetryMutantFails

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-031 — TestRetryOverAttemptFails

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-032 — TestAddErrorIsolationResultData

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-033 — TestAddErrorIsolationFixed

Disposition: migrate. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M29 — Cache ownership, reuse and recovery

- [ ] **M29: planned** — owner lane: `migration-history`.

Proposed owner: `tests/native-can/migration/m29-cache-ownership-reuse-and-reco/**`. Group start after: P15, P28, P27, K17, P20. Group accept after: P23. Logical gates: WORK, BUILD, CHILD, CLOCK.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Combine related cache mechanics only after mapping each row: key sensitivity, corruption refusal, one producer, bounded slots and conservative foreign/active recovery.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Combine related cache mechanics only after mapping each row: key sensitivity, corruption refusal, one producer, bounded slots and conservative foreign/active recovery. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add cache-ownership-reuse-and-reco cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### HISTORY-034 — TestClosePreservesParentAndConcurrentCache

Disposition: migrate. Start after: P15, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-035 — TestRecoveryRequiresInactivityAndPreservesForeignPaths

Disposition: migrate. Start after: P15, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-036 — TestUnavailableLivenessRetainsOrphan

Disposition: migrate. Start after: P15, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-037 — TestRecoveryRejectsSymlinkMarker

Disposition: migrate. Start after: P15, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-038 — TestRecoveryWorkIsBounded

Disposition: migrate. Start after: P15, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-039 — TestMain

Disposition: replace_support. Start after: P15, P27, P28, K17. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-040 — staging/assert-report helpers

Disposition: replace_support. Start after: P15, P20, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-041 — tempcache implementation

Disposition: replace_support. Start after: P15, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-016 — TestHarnessKeyStable

Disposition: consolidate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-017 — TestHarnessKeySensitive

Disposition: consolidate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-018 — TestHarnessVerifyClean

Disposition: consolidate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-019 — TestHarnessVerifyContaminated

Disposition: consolidate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-020 — TestHarnessEntryComplete

Disposition: consolidate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-021 — TestHeavySlotsBounded

Disposition: consolidate. Start after: P15, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-022 — TestHeavySlotCap

Disposition: consolidate. Start after: P15, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-023 — TestHarnessSharedEndToEnd

Disposition: consolidate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-024 — TestMain

Disposition: replace_support. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 3 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M30 — Baseline, T26 and T27 historical relevance

- [ ] **M30: planned** — owner lane: `migration-history`.

Proposed owner: `tests/native-can/migration/m30-baseline-t26-and-t27-historica/**`. Group start after: P15, P27, P28, K17. Group accept after: P23. Logical gates: SUITE, BUILD, DOC, FMT.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Review each dated/frozen assertion for current-contract relevance; retain historical records, migrate current catalogue/smoke checks, and keep broad meta-runner out of Can-owned credit.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Review each dated/frozen assertion for current-contract relevance; retain historical records, migrate current catalogue/smoke checks, and keep broad meta-runner out of Can-owned credit. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add baseline-t26-and-t27-historica cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: Conditional rows require per-row current-contract relevance and reviewer disposition; preserve historical records/static fixtures where still useful. The integrator alone may retire shared old files after all linked rows and callers are resolved.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### HISTORY-042 — TestBaselineToolchainMatchesFreeze

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-043 — TestRegistryComplete

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-044 — TestHarnessSmokeRepeatable

Disposition: migrate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-045 — TestT26ProductGuideCoversContracts

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-046 — TestT26AgentComparisonReportConsistent

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-047 — TestT26GeneratedArtifactsFresh

Disposition: migrate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-048 — TestT27DIInventoryComplete

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-049 — TestT27AcceptedItemsMapToTestsAndDocs

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-050 — TestT27NoDeferredMechanismPresent

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-051 — TestT27GateEvidenceLinked

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-052 — TestT27GeneratedArtifactsFresh

Disposition: consolidate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-053 — TestT27RecommendationScoped

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-054 — baseline smoke/full shell runner

Disposition: migrate. Start after: P15, P27, P28, K17. Accept after: P23.

- [ ] Account for 1 protected facets, 3 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-055 — baseline.json

Disposition: preserve_record. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.

### HISTORY-056 — registry.json

Disposition: preserve_record. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.

### HISTORY-057 — t26-agent-comparison-2026-09-24.json

Disposition: preserve_record. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.

### HISTORY-058 — t26-agent-comparison-2026-09-24.md

Disposition: preserve_record. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.

### HISTORY-059 — README.md

Disposition: preserve_record. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.

### HISTORY-060 — candidate.json

Disposition: preserve_record. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.

## M31 — Prototype admission guards

- [ ] **M31: planned** — owner lane: `migration-history`.

Proposed owner: `tests/native-can/migration/m31-prototype-admission-guards/**`. Group start after: P15, P27, P28. Group accept after: P23. Logical gates: SUITE, BUILD, DOC.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Keep current non-admission census and closed review manifest checks; conditional prototype behavior is not automatically production coverage.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Keep current non-admission census and closed review manifest checks; conditional prototype behavior is not automatically production coverage. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add prototype-admission-guards cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### HISTORY-061 — TestPrototypeCapabilitiesUnadmitted

Disposition: migrate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-062 — TestPrototypesUnreferenced

Disposition: migrate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-063 — TestReviewManifestCoversPrototypes

Disposition: migrate. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M32 — Prototype local adapters

- [ ] **M32: active** — owner lane: `migration-history`. Evidence: [evidence/M32.json](evidence/M32.json).

**Audit correction:** Fifteen retained ports have source/dispositions but the evidence defers execution, identities and qualification until P23. Aggregate completion requires those conditional row gates.

Proposed owner: `tests/native-can/migration/m32-prototype-local-adapters/**`. Group start after: P15. Group accept after: row decisions only. Logical gates: N1, N2, BROWSER.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. For chart, clipboard and storage adapter declarations, decide retain/migrate/obsolete per current contract; preserve no-host-contact, mapping and opacity facts if retained.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. For chart, clipboard and storage adapter declarations, decide retain/migrate/obsolete per current contract; preserve no-host-contact, mapping and opacity facts if retained. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add prototype-local-adapters cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: Conditional rows require per-row current-contract relevance and reviewer disposition; preserve historical records/static fixtures where still useful. The integrator alone may retire shared old files after all linked rows and callers are resolved.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### HISTORY-064 — "create, select, update, dispose roundtrip"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-065 — "invalid specs reject before host contact"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-066 — "unavailable SDK fails closed without host contact"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-067 — "native SDK failure maps to unavailable with no native text"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-068 — "deep copies: caller mutation cannot reach the widget or callbacks"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-069 — "callback throw is sealed: one diagnostic, SDK never sees it"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-070 — "nested events serialize in dispatch order"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-071 — "reentrant update applies immediately during dispatch"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-072 — "dispose poisons: late events dropped, ops fail, double dispose ok"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-073 — "dispose during callback completes the callback then drops the rest"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-074 — "tokens are opaque: SDK handles never leak to callers"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-075 — "surface census: three ops, three failure leaves, one callback"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-076 — "roundtrip: write then read returns the text"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-077 — "empty clipboard reads the empty leaf"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-078 — "invalid writes reject before host contact"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-079 — "denied permission fails closed before clipboard contact"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-080 — "unavailable target fails closed without host contact"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-081 — "native denial and native failure map with no native text"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-082 — "reentrancy: concurrent writes settle independently in host order"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-083 — "surface census: two ops, four failure leaves, zero callbacks"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-084 — "roundtrip: set, get, remove, missing reads null"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-085 — "invalid keys and values reject before host contact"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-086 — "quota breach maps to quota_exceeded with no native text"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-087 — "unavailable target fails closed without host contact"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-088 — "native security failure maps to unavailable with no native text"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-089 — "immutable copies: results are frozen primitives with no aliasing"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-090 — "surface census: three ops, three failure leaves, zero callbacks"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

## M33 — Prototype companion surfaces and packaging

- [ ] **M33: active** — owner lane: `migration-history`. Evidence: [evidence/M33.json](evidence/M33.json).

**Audit correction:** Eight retained ports remain provisional; recorded parse/relevance checks do not establish required controls, R/N/S/A/C identities or environment receipts.

Proposed owner: `tests/native-can/migration/m33-prototype-companion-surfaces-a/**`. Group start after: P15. Group accept after: row decisions only. Logical gates: PEER, WIRE, N1.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Review companion chart/clipboard/storage protocol and bundle-size assertions individually; migrate retained semantics without freezing old sizes or prototype ABI.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Review companion chart/clipboard/storage protocol and bundle-size assertions individually; migrate retained semantics without freezing old sizes or prototype ABI. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add prototype-companion-surfaces-a cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: Conditional rows require per-row current-contract relevance and reviewer disposition; preserve historical records/static fixtures where still useful. The integrator alone may retire shared old files after all linked rows and callers are resolved.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### HISTORY-091 — "render, select, release roundtrip"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-092 — "invalid specs reject client-side without sending"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-093 — "select-after-release expires; double release is idempotent"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-094 — "out-of-range index rejects against a live render"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-095 — "denied destination, bad credential, and offline fail closed"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-096 — "version mismatch fails closed both directions"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-097 — "auth parity: same policy both sides; secret never logged"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-098 — "envelope bytes are deterministic and measured"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-099 — "pastebin roundtrip: write then read returns the text"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-100 — "empty companion cell reads the empty leaf"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-101 — "denied destination and bad credential fail closed"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-102 — "version mismatch and transport failure map honestly"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-103 — "auth parity: same policy both sides; secret never logged"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-104 — "semantic gap: companion cell and device cell are disjoint"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-105 — "envelope bytes are deterministic and measured"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-106 — "roundtrip: set, get, remove, missing reads null"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-107 — "invalid keys reject client-side without sending"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-108 — "denied destination fails closed without sending"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-109 — "bad credential maps to unauthorized on both sides"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-110 — "version mismatch fails closed both directions"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-111 — "transport failure maps to transport_failed"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-112 — "malformed response maps to protocol_error"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-113 — "server quota passes through as quota_exceeded"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-114 — "auth parity: same policy admits on both sides; secret never logged"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-115 — "envelope bytes are deterministic and measured"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-116 — "prototype bundles are byte-identical across builds"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-117 — "bundle sizes are pinned"

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

## M34 — Prototype source and review records

- [x] **M34: complete** — owner lane: `migration-history`. Evidence: [evidence/M34.json](evidence/M34.json).

Proposed owner: `tests/native-can/migration/m34-prototype-source-and-review-re/**`. Group start after: P15. Group accept after: row decisions only. Logical gates: DOC, BUILD.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Preserve review artifacts and decide conditional retirement of six unadmitted prototype sources; no test credit from their historical presence.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Preserve review artifacts and decide conditional retirement of six unadmitted prototype sources; no test credit from their historical presence. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add prototype-source-and-review-re cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: Conditional rows require per-row current-contract relevance and reviewer disposition; preserve historical records/static fixtures where still useful. The integrator alone may retire shared old files after all linked rows and callers are resolved.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### HISTORY-118 — REVIEW-MANIFEST.json

Disposition: preserve_record. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.

### HISTORY-119 — x-r01-1.md

Disposition: preserve_record. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.

### HISTORY-120 — t1-sketches.md

Disposition: preserve_record. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.

### HISTORY-121 — chart-widget.ts

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-122 — clipboard.ts

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-123 — storage.ts

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-124 — chart-widget.ts

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-125 — clipboard.ts

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

### HISTORY-126 — storage.ts

Disposition: historical_candidate. Start after: P15. Accept after: reviewed disposition. If retained live: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.
- [ ] For any retained behavior, author a separate Can case and require its qualified gates before retirement.
- [ ] If retained as live behavior, use if_retained_accept_after for the new Can port; the relevance decision itself needs no live qualification.

## M35 — Native runtime qualification

- [ ] **M35: planned** — owner lane: `migration-history`.

Proposed owner: `tests/native-can/migration/m35-native-runtime-qualification/**`. Group start after: P15, P27, P28, K17, K01, K02, K03, K04, K05. Group accept after: P23, QN1, QN2. Logical gates: N1, N2, N3, ENV, ARCHIVE.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Reauthor every native behavior/identity probe in Can with independent raw facts, absent-API and changed-identity controls; coordinate linked EDGE-001 caller.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Reauthor every native behavior/identity probe in Can with independent raw facts, absent-API and changed-identity controls; coordinate linked EDGE-001 caller. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add native-runtime-qualification cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### HISTORY-127 — "qualified native APIs and behaviors"

Disposition: migrate. Start after: P15, P27, P28, K17, K01, K02, K03, K04, K05. Accept after: P23, QN1, QN2.

- [ ] Account for 1 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-128 — `refuse missing ${name} without fallback`

Disposition: migrate. Start after: P15, P27, P28, K17, K01, K02, K03, K04, K05. Accept after: P23, QN1, QN2.

- [ ] Account for 1 protected facets, 6 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-129 — `refuse different ${key}`

Disposition: migrate. Start after: P15, P27, P28, K17, K01, K02, K03, K04, K05. Accept after: P23, QN1, QN2.

- [ ] Account for 1 protected facets, 6 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-130 — "refuse missing AsyncLocalStorage without fallback"

Disposition: migrate. Start after: P15, P27, P28, K17, K04, K05. Accept after: P23, QN2.

- [ ] Account for 1 protected facets, 0 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### HISTORY-131 — native qualification implementation

Disposition: migrate. Start after: P15, P28, K17, K01, K02, K03, K04, K05. Accept after: P23, QN1, QN2.

- [ ] Account for 2 protected facets, 3 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M36 — Authoring policy comparison records

- [x] **M36: complete** — owner lane: `migration-history`. Evidence: [evidence/M36.json](evidence/M36.json).

Proposed owner: `tests/native-can/migration/m36-authoring-policy-comparison-re/**`. Group start after: P15. Group accept after: row decisions only. Logical gates: DOC, SUITE.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Preserve dated unrun agent-trial records and six Can comparison fixtures; require a current relevance decision before active checks are retired.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Preserve dated unrun agent-trial records and six Can comparison fixtures; require a current relevance decision before active checks are retired. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add authoring-policy-comparison-re cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: Conditional rows require per-row current-contract relevance and reviewer disposition; preserve historical records/static fixtures where still useful. The integrator alone may retire shared old files after all linked rows and callers are resolved.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### HISTORY-132 — README.md

Disposition: preserve_record. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.

### HISTORY-133 — registry.json

Disposition: preserve_record. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.

### HISTORY-134 — x-r08-comparison-2026-09-26.md

Disposition: preserve_record. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.

### HISTORY-135 — README.md

Disposition: preserve_record. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.

### HISTORY-136 — main.can

Disposition: retain_fixture. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 2 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Confirm exact static fixture bytes, named consumer and starting-state assumptions.
- [ ] Keep or move the fixture as data; do not count it as a Can-authored scenario or delete it with a driver.

### HISTORY-137 — main.can

Disposition: retain_fixture. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 2 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Confirm exact static fixture bytes, named consumer and starting-state assumptions.
- [ ] Keep or move the fixture as data; do not count it as a Can-authored scenario or delete it with a driver.

### HISTORY-138 — main.can

Disposition: retain_fixture. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 2 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Confirm exact static fixture bytes, named consumer and starting-state assumptions.
- [ ] Keep or move the fixture as data; do not count it as a Can-authored scenario or delete it with a driver.

### HISTORY-139 — main.can

Disposition: retain_fixture. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 2 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Confirm exact static fixture bytes, named consumer and starting-state assumptions.
- [ ] Keep or move the fixture as data; do not count it as a Can-authored scenario or delete it with a driver.

### HISTORY-140 — main.can

Disposition: retain_fixture. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 2 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Confirm exact static fixture bytes, named consumer and starting-state assumptions.
- [ ] Keep or move the fixture as data; do not count it as a Can-authored scenario or delete it with a driver.

### HISTORY-141 — main.can

Disposition: retain_fixture. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 2 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Confirm exact static fixture bytes, named consumer and starting-state assumptions.
- [ ] Keep or move the fixture as data; do not count it as a Can-authored scenario or delete it with a driver.

### HISTORY-142 — x-r02-1.md

Disposition: preserve_record. Start after: P15. Accept after: reviewed disposition.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Review each assertion or record against the current contract and active callers.
- [ ] Record retain/migrate/obsolete decision with rationale; preserved history is not execution evidence.

## M37 — Native qualification caller bridge

- [ ] **M37: planned** — owner lane: `migration-history`.

Proposed owner: `tests/native-can/migration/m37-native-qualification-caller-br/**`. Group start after: P15, P28, K17, K01, K02, K03. Group accept after: P23, QN1. Logical gates: SUITE, N1, ENV, ARCHIVE.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Replace native.ts policy with Can-owned cases and typed facts; coordinate EDGE-001 Python isolation/provenance/aggregation without duplicate credit.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Replace native.ts policy with Can-owned cases and typed facts; coordinate EDGE-001 Python isolation/provenance/aggregation without duplicate credit. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add native-qualification-caller-br cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### HISTORY-143 — distribution native qualification caller

Disposition: rewire. Start after: P15, P28, K17, K01, K02, K03. Accept after: P23, QN1.

- [ ] Account for 1 protected facets, 2 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Rewire caller without dropping provisioning or unrelated independent gates.
- [ ] Check the new report/receipt/correction contract before retiring old invocation.

## M38 — Gate3 route and server matrix

- [ ] **M38: planned** — owner lane: `migration-life`.

Proposed owner: `tests/native-can/migration/m38-gate3-route-and-server-matrix/**`. Group start after: P15, P20, P27, P28, K20, K22. Group accept after: P23, QHTTP, QD1. Logical gates: BUILD, DIAG, DB, HTTP.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Rebuild changed contracts and exercise exact load/save, adapter routing, malformed captures and unchanged handler-entry evidence.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Rebuild changed contracts and exercise exact load/save, adapter routing, malformed captures and unchanged handler-entry evidence. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add gate3-route-and-server-matrix cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### LIFE-001 — TestGate3ContractEdits

Disposition: migrate. Start after: P15, P20, P27, P28. Accept after: P23.

- [ ] Account for 2 protected facets, 1 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-002 — TestGate3RouteRebuildLive

Disposition: migrate. Start after: P15, P27, P28, K20, K22. Accept after: P23, QHTTP, QD1.

- [ ] Account for 2 protected facets, 1 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-003 — TestGate3ServerMatrix

Disposition: migrate. Start after: P15, P27, P28, K20, K22. Accept after: P23, QHTTP, QD1.

- [ ] Account for 7 protected facets, 3 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-004 — TestGate3AdapterMatrix

Disposition: migrate. Start after: P15, P27, P28, K20, K22. Accept after: P23, QHTTP, QD1.

- [ ] Account for 4 protected facets, 5 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M39 — Gate3 concurrency, replay and fault lifecycle

- [ ] **M39: planned** — owner lane: `migration-life`.

Proposed owner: `tests/native-can/migration/m39-gate3-concurrency-replay-and-f/**`. Group start after: P15, P27, P28, K20, K22, K24, K26. Group accept after: P23, QHTTP, QD1, QD2, QD4. Logical gates: DB, D1, D4, CHILD, CLOCK.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Test one-winner/seven-conflict, replay expiry/revocation, postcommit renderer fault and startup/shutdown using settled independent effects.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Test one-winner/seven-conflict, replay expiry/revocation, postcommit renderer fault and startup/shutdown using settled independent effects. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: DELEGATE-018; shared delegate references: none.

Suggested first small commit: `test(native-can): add gate3-concurrency-replay-and-f cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### LIFE-005 — TestGate3Concurrency

Disposition: migrate. Start after: P15, P27, P28, K20, K22, K24. Accept after: P23, QHTTP, QD1, QD2.

- [ ] Account for 4 protected facets, 3 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-006 — TestGate3ReplayExpiry

Disposition: migrate. Start after: P15, P27, P28, K20, K22, K24. Accept after: P23, QHTTP, QD1, QD2.

- [ ] Account for 4 protected facets, 3 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-007 — TestGate3RendererFault

Disposition: migrate. Start after: P15, P27, P28, K20, K22. Accept after: P23, QHTTP, QD1.

- [ ] Account for 3 protected facets, 2 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-008 — TestGate3Lifecycle

Disposition: migrate. Start after: P15, P27, P28, K20, K22, K26. Accept after: P23, QHTTP, QD1, QD4.

- [ ] Account for 4 protected facets, 3 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Map delegated oracle evidence: DELEGATE-018.

## M40 — Gate4 installed artifact, webhook and companion

- [ ] **M40: planned** — owner lane: `migration-life`.

Proposed owner: `tests/native-can/migration/m40-gate4-installed-artifact-webho/**`. Group start after: P15, P27, P28, K17, K20, K22, K01, K02, K03, K06, K19, K24, K26. Group accept after: P23, QHTTP, QD1, QN1, QN3, QF2, QD2, QD4. Logical gates: ARCHIVE, PEER, DB, CHILD, METRIC.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Preserve installed/development identity, exact HMAC webhook bytes and two-process companion report/receipt semantics; defer authorized measurement legs.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Preserve installed/development identity, exact HMAC webhook bytes and two-process companion report/receipt semantics; defer authorized measurement legs. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: DELEGATE-019, DELEGATE-020; shared delegate references: DELEGATE-009, DELEGATE-016, DELEGATE-026, DELEGATE-028.

Suggested first small commit: `test(native-can): add gate4-installed-artifact-webho cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### LIFE-009 — TestGate4FaultMatrix

Disposition: migrate. Start after: P15, P27, P28, K17, K20, K22, K01, K02, K03, K06, K19. Accept after: P23, QHTTP, QD1, QN1, QN3, QF2.

- [ ] Account for 5 protected facets, 3 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Map delegated oracle evidence: DELEGATE-009, DELEGATE-016, DELEGATE-019, DELEGATE-020, DELEGATE-026, DELEGATE-028.

### LIFE-010 — TestWebhookSliceLive

Disposition: migrate. Start after: P15, P27, P28, K20, K22. Accept after: P23, QHTTP, QD1.

- [ ] Account for 6 protected facets, 4 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-011 — TestCompanionPairLiveF06

Disposition: migrate. Start after: P15, P27, P28, K20, K22, K24, K26. Accept after: P23, QHTTP, QD1, QD2, QD4.

- [ ] Account for 8 protected facets, 10 listed variants and 4 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.
- [ ] Keep measurement facets deferred until explicitly authorized; do not invent or substitute a threshold.

## M41 — Linux, S3 and UP23 verdict

- [ ] **M41: planned** — owner lane: `migration-life`.

Proposed owner: `tests/native-can/migration/m41-linux-s3-and-up23-verdict/**`. Group start after: P15, P27, P28, K17, K22, K01, K02, K03, K20, K27, K18, K07, K08, K09, K10, K11. Group accept after: P23, QD1, QN1, QHTTP, QStore, QF1, QB0, QB1base, QB1. Logical gates: ARCHIVE, DB, STORE, B1.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Use installed Linux artifacts, independent PG/SQLite facts, preowned S3 prefix and full two-engine UP23 guard evidence with cleanup.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Use installed Linux artifacts, independent PG/SQLite facts, preowned S3 prefix and full two-engine UP23 guard evidence with cleanup. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add linux-s3-and-up23-verdict cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### LIFE-012 — TestLinuxInstalledArtifactSmoke

Disposition: migrate. Start after: P15, P27, P28, K17, K22, K01, K02, K03. Accept after: P23, QD1, QN1.

- [ ] Account for 3 protected facets, 2 listed variants and 2 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-013 — TestLinuxPostgresRoundtrip

Disposition: migrate. Start after: P15, P28, K17, K22, K01, K02, K03. Accept after: P23, QD1, QN1.

- [ ] Account for 2 protected facets, 1 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-014 — TestCurrentS3Objects

Disposition: migrate. Start after: P15, P27, P28, K20, K27, K17, K18. Accept after: P23, QHTTP, QStore, QF1.

- [ ] Account for 5 protected facets, 1 listed variants and 3 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-015 — TestUP23WriteVerdict

Disposition: migrate. Start after: P15, P27, P28, K20, K07, K08, K09, K22, K10, K11. Accept after: P23, QHTTP, QB0, QD1, QB1base, QB1.

- [ ] Account for 3 protected facets, 2 listed variants and 1 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M42 — Webhook and S3 independent setup

- [ ] **M42: planned** — owner lane: `migration-life`.

Proposed owner: `tests/native-can/migration/m42-webhook-and-s3-independent-set/**`. Group start after: P15, P28, K22, K27. Group accept after: P23, QD1, QStore. Logical gates: DB, STORE, WORK.

Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Replace driver policy with Can-selected seeds and raw readback; register namespaces/prefixes before effects and report cleanup failure.

Acceptance: Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Replace driver policy with Can-selected seeds and raw readback; register namespaces/prefixes before effects and report cleanup failure. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests.

Evidence: Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add webhook-and-s3-independent-set cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### LIFE-025 — setup / inspect

Disposition: replace_support. Start after: P15, P28, K22. Accept after: P23, QD1.

- [ ] Account for 2 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

### LIFE-026 — setup

Disposition: replace_support. Start after: P15, P28, K27. Accept after: P23, QStore.

- [ ] Account for 2 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Map every protected facet, named fixture, dynamic variant and required environment to Can case/check IDs.
- [ ] Implement one row-sized Can action/expectation slice under the owned prefix; commit early.
- [ ] Exercise positive, seeded defect and missing-evidence controls, then save report and cleanup receipt.

## M43 — Native and Linux external consumers

- [ ] **M43: planned** — owner lane: `migration-edge`.

Proposed owner: `tests/native-can/migration/m43-native-and-linux-external-cons/**`. Group start after: P15, P28, K17, K01, K02, K03, K22. Group accept after: P23, QN1, QD1. Logical gates: SUITE, ARCHIVE, DB, N1.

Prepare the Can-side policy, row mapping and a reviewed patch for the existing external callers under integrator ownership. Exercise the new report/receipt contract in a staged or read-only check where feasible; Z03 applies and activates the shared caller patch after dependencies pass. Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Rewire Python/Linux qualification and PG/SQLite drivers after Can-owned native and installed-artifact evidence; preserve provisioning mechanics.

Acceptance: This task can finish with a reviewed, exact caller patch and evidence contract before Z03 activates it; do not require Z03 completion or edit shared caller files in this slice. Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Rewire Python/Linux qualification and PG/SQLite drivers after Can-owned native and installed-artifact evidence; preserve provisioning mechanics. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests. Task completion accepts the qualified Can policy and reviewed staged caller patch. Actual shared-caller activation belongs to Z03; no circular prerequisite requires it already applied here.

Evidence: Proposed patch, caller inventory and staged contract evidence; integrator activation and old invocation retirement are Z03 work. Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add native-and-linux-external-cons cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### EDGE-001 — main / native qualification

Disposition: rewire. Start after: P15, P28, K17, K01, K02, K03. Accept after: P23, QN1.

- [ ] Account for 3 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Prepare a reviewed shared-caller patch; preserve provisioning and unrelated independent gates.
- [ ] Validate report/receipt/correction handling in a staged check, then hand activation to integrator Z03.

### EDGE-002 — installed artifact smoke

Disposition: rewire. Start after: P15, P28, K17, K22. Accept after: P23, QD1.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Prepare a reviewed shared-caller patch; preserve provisioning and unrelated independent gates.
- [ ] Validate report/receipt/correction handling in a staged check, then hand activation to integrator Z03.

### EDGE-003 — PostgreSQL smoke

Disposition: rewire. Start after: P15, P28, K17, K22. Accept after: P23, QD1.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Prepare a reviewed shared-caller patch; preserve provisioning and unrelated independent gates.
- [ ] Validate report/receipt/correction handling in a staged check, then hand activation to integrator Z03.

### EDGE-004 — native PostgreSQL roundtrip

Disposition: migrate. Start after: P15, P28, K22. Accept after: P23, QD1.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Prepare a reviewed shared-caller patch; preserve provisioning and unrelated independent gates.
- [ ] Validate report/receipt/correction handling in a staged check, then hand activation to integrator Z03.

### EDGE-005 — native SQLite seed

Disposition: migrate. Start after: P15, P28, K22. Accept after: P23, QD1.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Prepare a reviewed shared-caller patch; preserve provisioning and unrelated independent gates.
- [ ] Validate report/receipt/correction handling in a staged check, then hand activation to integrator Z03.

## M44 — CI and host conformance consumers

- [ ] **M44: planned** — owner lane: `migration-edge`.

Proposed owner: `tests/native-can/migration/m44-ci-and-host-conformance-consum/**`. Group start after: P15, P27, P28, K17, K07, K08, K09. Group accept after: P23, QB0. Logical gates: SUITE, BUILD, B1, ENV, WORK.

Prepare the Can-side policy, row mapping and a reviewed patch for the existing external callers under integrator ownership. Exercise the new report/receipt contract in a staged or read-only check where feasible; Z03 applies and activates the shared caller patch after dependencies pass. Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Replace in-scope CI invocations and host harness dependencies with qualified Can report plus N receipt; keep independent whole-repository gates separate.

Acceptance: This task can finish with a reviewed, exact caller patch and evidence contract before Z03 activates it; do not require Z03 completion or edit shared caller files in this slice. Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Replace in-scope CI invocations and host harness dependencies with qualified Can report plus N receipt; keep independent whole-repository gates separate. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests. Task completion accepts the qualified Can policy and reviewed staged caller patch. Actual shared-caller activation belongs to Z03; no circular prerequisite requires it already applied here.

Evidence: Proposed patch, caller inventory and staged contract evidence; integrator activation and old invocation retirement are Z03 work. Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add ci-and-host-conformance-consum cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### EDGE-006 — native qualification / browser provision / UP23 / whole-repository gate

Disposition: rewire. Start after: P15, P27, P28, K17, K07, K08, K09. Accept after: P23, QB0.

- [ ] Account for 2 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Prepare a reviewed shared-caller patch; preserve provisioning and unrelated independent gates.
- [ ] Validate report/receipt/correction handling in a staged check, then hand activation to integrator Z03.

### EDGE-007 — release qualification

Disposition: rewire. Start after: P15, P27, P28, K17. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Prepare a reviewed shared-caller patch; preserve provisioning and unrelated independent gates.
- [ ] Validate report/receipt/correction handling in a staged check, then hand activation to integrator Z03.

### EDGE-008 — fresh emit and strict TypeScript gate

Disposition: rewire. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Prepare a reviewed shared-caller patch; preserve provisioning and unrelated independent gates.
- [ ] Validate report/receipt/correction handling in a staged check, then hand activation to integrator Z03.

### EDGE-009 — canlcCache / TestMain / canlcBinary

Disposition: rewire. Start after: P15, P27, P28. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Prepare a reviewed shared-caller patch; preserve provisioning and unrelated independent gates.
- [ ] Validate report/receipt/correction handling in a staged check, then hand activation to integrator Z03.

### EDGE-010 — browser dependency consumer

Disposition: rewire. Start after: P15, P28, K17, K07, K08, K09. Accept after: P23, QB0.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Prepare a reviewed shared-caller patch; preserve provisioning and unrelated independent gates.
- [ ] Validate report/receipt/correction handling in a staged check, then hand activation to integrator Z03.

### EDGE-011 — vendor B browser dependency consumer

Disposition: rewire. Start after: P15, P28, K17, K07, K08, K09. Accept after: P23, QB0.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Prepare a reviewed shared-caller patch; preserve provisioning and unrelated independent gates.
- [ ] Validate report/receipt/correction handling in a staged check, then hand activation to integrator Z03.

### EDGE-012 — Linux provisioning/launcher

Disposition: rewire. Start after: P15, P28, K17. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Prepare a reviewed shared-caller patch; preserve provisioning and unrelated independent gates.
- [ ] Validate report/receipt/correction handling in a staged check, then hand activation to integrator Z03.

### EDGE-013 — qualification prerequisites and browser-path preflight

Disposition: rewire. Start after: P15, P28, K17. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Prepare a reviewed shared-caller patch; preserve provisioning and unrelated independent gates.
- [ ] Validate report/receipt/correction handling in a staged check, then hand activation to integrator Z03.

## M45 — Operator documentation callers

- [ ] **M45: planned** — owner lane: `migration-edge`.

Proposed owner: `tests/native-can/migration/m45-operator-documentation-callers/**`. Group start after: P15, P28, K17, K07, K08, K09. Group accept after: P23, QB0. Logical gates: DOC, SUITE.

Prepare the Can-side policy, row mapping and a reviewed patch for the existing external callers under integrator ownership. Exercise the new report/receipt contract in a staged or read-only check where feasible; Z03 applies and activates the shared caller patch after dependencies pass. Complete rows one at a time under this owned prefix. Per-row start/accept gates and substeps in row_gates govern progress; the group-level accept_after is only the union for finishing this whole group. For each row, map protected facets, dynamic variants, required environments, related fixtures and delegated oracle declarations; add ordinary Can actions and expectations in small commits. Update native/Linux/browser operator commands only after the new qualified entrypoint and dependency provisioning are real.

Acceptance: This task can finish with a reviewed, exact caller patch and evidence contract before Z03 activates it; do not require Z03 completion or edit shared caller files in this slice. Each retained row has an explicit case/check mapping, positive and seeded negative/missing-evidence controls, identified C/R/N/S/A inputs, required environment runs and matching cleanup receipts. Conditional historical rows require reviewed current-contract dispositions. Update native/Linux/browser operator commands only after the new qualified entrypoint and dependency provisioning are real. Independently review each new helper/case/expected-vector source closure, complete mandatory offline checks with R, and bind the new suite/artifact S/A identities before execution credit; prior P23 or I acceptance does not cover newly added tests. Task completion accepts the qualified Can policy and reviewed staged caller patch. Actual shared-caller activation belongs to Z03; no circular prerequisite requires it already applied here.

Evidence: Proposed patch, caller inventory and staged contract evidence; integrator activation and old invocation retirement are Z03 work. Per-row source/fixture map, Can check IDs, qualified report and N receipt or recorded blocked/conditional decision. Delegate ownership: none; shared delegate references: none.

Suggested first small commit: `test(native-can): add operator-documentation-callers cases` (start with one row or fixture; keep later rows in separate commits).

Retirement: No shared old file is deleted by this slice. The integrator may retire one only after all linked rows, delegated oracles, required variants/environments, controls, cleanup receipts and callers have complete accepted evidence.

- [ ] Map the first ready row facets, variants, environment and active callers.
- [ ] Commit one ordinary Can case or reviewed historical/fixture decision.
- [ ] Repeat row by row, preserving blocked status and per-row gate evidence.
- [ ] Hand shared old-file retirement to integrator Z02 only after linked rows/callers resolve.

### EDGE-014 — operator commands

Disposition: rewire. Start after: P15. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Prepare a reviewed shared-caller patch; preserve provisioning and unrelated independent gates.
- [ ] Validate report/receipt/correction handling in a staged check, then hand activation to integrator Z03.

### EDGE-015 — Linux operator commands

Disposition: rewire. Start after: P15. Accept after: P23.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Prepare a reviewed shared-caller patch; preserve provisioning and unrelated independent gates.
- [ ] Validate report/receipt/correction handling in a staged check, then hand activation to integrator Z03.

### EDGE-016 — shared browser installation instructions

Disposition: rewire. Start after: P15, P28, K17, K07, K08, K09. Accept after: P23, QB0.

- [ ] Account for 1 protected facets, 0 listed variants and 0 environment gates; use the [coverage map](coverage-map.json) for exact text.
- [ ] Identify the already qualified Can row/report consumed here.
- [ ] Prepare a reviewed shared-caller patch; preserve provisioning and unrelated independent gates.
- [ ] Validate report/receipt/correction handling in a staged check, then hand activation to integrator Z03.

## Shared retirement handoff

- [ ] Integrator rechecks source hashes and all current direct/indirect callers before an old-file change.
- [ ] Resolve all 30 delegated runtime suites. Each has one owner group and any secondary parent cites that body; invoking Bun tests from Can earns no migrated oracle credit.
- [ ] Reconcile HISTORY-143 with EDGE-001 as policy and caller views of one native-qualification dependency.
- [ ] Preserve useful static fixtures and dated records; decide every historical candidate against the current contract.
- [ ] Prepare reviewed CI, Linux, host conformance and operator patches; integrator Z03 activates them after relevant Can evidence and cleanup receipts are accepted, preserving unrelated whole-repository gates.
- [ ] Retire only integrator-owned old files whose full row/caller set passes the retirement map gate.
