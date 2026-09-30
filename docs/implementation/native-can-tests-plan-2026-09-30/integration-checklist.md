# Capability promotion, retirement and review

Status: **planned, not started**. [Main plan](../native-can-tests-plan-2026-09-30.md) defines execution, commits, resource bounds and shared retirement gates.

Each checkbox is a task acceptance, not merely source completion. Draft after `start_after`; complete only after both dependency lists and the task checks pass. Record evidence using the [progress template](progress-template.json). Proposed paths do not imply existing implementation. Cards follow dependency order; independent lanes may run concurrently within the main plan’s limits.

I tasks require two small commits: checked bindings/helpers first, accepted R/N/S/A delta evidence afterward. The single integration writer can batch compatible ready slices into one bounded build; each scope keeps its own receipts. Z retirement tasks record per-file/row partial progress so ready deletions do not wait for the whole suite.

## I01 — Integrate and accept native value transport capability slice

- [ ] **I01 accepted** — owner lane: `integration-reference`.

**Start after:** K01, K02, K03, P28. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i01/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I01.json`, `tests/native-can/src/capabilities/i01/`, `tests/native-can/examples/capabilities/i01/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I01.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate native value transport`.

## I02 — Integrate and accept generated C ingress capability slice

- [ ] **I02 accepted** — owner lane: `integration-reference`.

**Start after:** I01, K04, K05, P28. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i02/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I02.json`, `tests/native-can/src/capabilities/i02/`, `tests/native-can/examples/capabilities/i02/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I02.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate generated C ingress`.

## I03 — Integrate and accept late native events capability slice

- [ ] **I03 accepted** — owner lane: `integration-reference`.

**Start after:** I02, K06, P28. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i03/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I03.json`, `tests/native-can/src/capabilities/i03/`, `tests/native-can/examples/capabilities/i03/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I03.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate late native events`.

## I04 — Integrate and accept HTTP and controlled peers capability slice

- [ ] **I04 accepted** — owner lane: `integration-reference`.

**Start after:** K20, P28, P29. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i04/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I04.json`, `tests/native-can/src/capabilities/i04/`, `tests/native-can/examples/capabilities/i04/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I04.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate HTTP and controlled peers`.

## I06 — Integrate and accept browser core capability slice

- [ ] **I06 accepted** — owner lane: `integration-reference`.

**Start after:** I04, K07, K08, K09, K10, K11, P28, P29. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i06/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I06.json`, `tests/native-can/src/capabilities/i06/`, `tests/native-can/examples/capabilities/i06/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I06.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate browser core`.

## I13 — Integrate and accept database row observation capability slice

- [ ] **I13 accepted** — owner lane: `integration-reference`.

**Start after:** K22, K23, P28, P29. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i13/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I13.json`, `tests/native-can/src/capabilities/i13/`, `tests/native-can/examples/capabilities/i13/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I13.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate database row observation`.

## I07 — Integrate and accept browser DOM events capability slice

- [ ] **I07 accepted** — owner lane: `integration-reference`.

**Start after:** I06, K13, P28, P29. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i07/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I07.json`, `tests/native-can/src/capabilities/i07/`, `tests/native-can/examples/capabilities/i07/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I07.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate browser DOM events`.

## I08 — Integrate and accept page-origin Fetch capability slice

- [ ] **I08 accepted** — owner lane: `integration-reference`.

**Start after:** I06, K14, P28, P29. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i08/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I08.json`, `tests/native-can/src/capabilities/i08/`, `tests/native-can/examples/capabilities/i08/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I08.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate page-origin Fetch`.

## I09 — Integrate and accept browser Can artifacts capability slice

- [ ] **I09 accepted** — owner lane: `integration-reference`.

**Start after:** I06, I01, K15, P28, P29. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i09/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I09.json`, `tests/native-can/src/capabilities/i09/`, `tests/native-can/examples/capabilities/i09/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I09.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate browser Can artifacts`.

## I10 — Integrate and accept remote browser contexts capability slice

- [ ] **I10 accepted** — owner lane: `integration-reference`.

**Start after:** I06, K16, P28, P29. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i10/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I10.json`, `tests/native-can/src/capabilities/i10/`, `tests/native-can/examples/capabilities/i10/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I10.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate remote browser contexts`.

## I11 — Integrate and accept descriptor delivery capability slice

- [ ] **I11 accepted** — owner lane: `integration-reference`.

**Start after:** K17, K18, P28. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i11/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I11.json`, `tests/native-can/src/capabilities/i11/`, `tests/native-can/examples/capabilities/i11/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I11.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate descriptor delivery`.

## I12 — Integrate and accept inherited generation lease capability slice

- [ ] **I12 accepted** — owner lane: `integration-reference`.

**Start after:** I11, K19, P28. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i12/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I12.json`, `tests/native-can/src/capabilities/i12/`, `tests/native-can/examples/capabilities/i12/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I12.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate inherited generation lease`.

## I05 — Integrate and accept WebSocket observation capability slice

- [ ] **I05 accepted** — owner lane: `integration-reference`.

**Start after:** I04, K21, P28, P29. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i05/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I05.json`, `tests/native-can/src/capabilities/i05/`, `tests/native-can/examples/capabilities/i05/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I05.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate WebSocket observation`.

## I14 — Integrate and accept database transaction identity capability slice

- [ ] **I14 accepted** — owner lane: `integration-reference`.

**Start after:** I13, K24, P28, P29. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i14/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I14.json`, `tests/native-can/src/capabilities/i14/`, `tests/native-can/examples/capabilities/i14/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I14.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate database transaction identity`.

## I15 — Integrate and accept poisoned transaction observation capability slice

- [ ] **I15 accepted** — owner lane: `integration-reference`.

**Start after:** I13, K25, P28, P29. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i15/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I15.json`, `tests/native-can/src/capabilities/i15/`, `tests/native-can/examples/capabilities/i15/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I15.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate poisoned transaction observation`.

## I16 — Integrate and accept database deadline settlement capability slice

- [ ] **I16 accepted** — owner lane: `integration-reference`.

**Start after:** I14, K26, P28, P29. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i16/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I16.json`, `tests/native-can/src/capabilities/i16/`, `tests/native-can/examples/capabilities/i16/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I16.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate database deadline settlement`.

## I17 — Integrate and accept object-store ownership capability slice

- [ ] **I17 accepted** — owner lane: `integration-reference`.

**Start after:** K27, P28, P29. **Additional acceptance prerequisites:** P06, P26, P14.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/slices/i17/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/I17.json`, `tests/native-can/src/capabilities/i17/`, `tests/native-can/examples/capabilities/i17/`.

**Concrete change:** Integration writer merges this slice’s finite descriptors, native adapters and required checker/emitter hooks, regenerates catalogue mirrors, then separately builds and accepts the changed R/N capability scope with predecessor checks and independent bootstrap controls. Commit bindings first and acceptance evidence separately. Source changes stay provisional until acceptance; unrelated queued slices do not delay this one. Draft bindings once start dependencies are accepted; reference/owner promotion waits for accept_after. Batch compatible ready slices into one bounded build when possible, but record independent per-slice controls, receipts and acceptance; do not make unrelated slices prerequisites. Own a complete ordinary Can helper package and tiny typed example for this slice, including signatures, effects, opaque handles, errors and mandatory assertions. Do not defer the API surface to the later Q case.

**Acceptance checks:** Catalogue generation/check and relevant bounded local controls pass. Independent review verifies ordinary Can types/effects, exact input/runtime hashes, wrong-target/missing-binding/forged-owner controls and cleanup. Admit new features using reviewed fixed/native witnesses where predecessor R cannot express them; do not let the new Q suite be its only acceptance oracle. Retain predecessor authority if acceptance is inconclusive. Live-only host behavior is explicitly excluded until its Q gate; R/N binding acceptance cannot assert browser isolation or DB settlement without evidence. Compile/check the complete Can example provisionally and qualify its fixed contract controls independently before promotion. Independently review and accept the changed Can helper/expected-vector delta as S/A, bound to suite source closure and R-built artifact hashes. Prior P23 suite acceptance does not cover these new helpers.

**Completion evidence:** evidence/I17.json: predecessor/new R/N identities, delta scope, independent control outcomes, cleanup receipt and review; source and acceptance commit SHAs Include predecessor/new S/A identities, helper closure hashes, expected-vector review and independent assertion controls.

**Early commit:** `feat(testing): integrate object-store ownership`.

## Z01 — Reconcile every retained facet with concrete Can evidence

- [ ] **Z01 accepted** — owner lane: `integration-retirement`.

**Start after:** P16. **Additional acceptance prerequisites:** P23.

**Files:** `tests/native-can/src/coverage/migration/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/coverage/`.

**Concrete change:** Extend the Can registry/reducer with per-row facet, variant, environment and delegated-declaration mappings. Include source hashes, dispositions, expected case/check IDs, required receipts and correction chains. Validate each ready row incrementally; a group label or pass count is not coverage.

**Acceptance checks:** Dropped facet, missing engine, undeclared duplicate credit, unaccepted reference and later invalidation each prevent complete coverage. Validate the 292-row ledger and 30 delegate inventory against current sources.

**Completion evidence:** evidence/Z01.json plus per-row/file control and commit links.

**Early commit:** `feat(testing): reconcile migration coverage evidence`.

## Z02 — Retire qualified harness symbols and files incrementally

- [ ] **Z02 accepted** — owner lane: `integration-retirement`.

**Start after:** Z01. **Additional acceptance prerequisites:** P23.

**Files:** `old executable harnesses and shared support listed in retirement-map.json`.

**Concrete change:** Integration owner applies the shared deletion gate separately to each file/symbol once its actual rows/callers are qualified. Record per-file pending/eligible/retired state and deletion commit; do not wait for unrelated files. Static fixtures and reviewed historical records keep declared roles. Re-scan direct and reverse references before each deletion.

**Acceptance checks:** No file-wide retirement while any direct/related obligation or caller remains unresolved. Per-row retained facets, qualified replacement/control evidence and current cleanup/correction receipts are mandatory. Run affected replacement/caller checks before accepting each deletion; record partial progress without completing the task until all eligible executable host policy is resolved.

**Completion evidence:** evidence/Z02.json plus per-row/file control and commit links.

**Early commit:** `refactor(tests): retire qualified harness slices`.

## Z03 — Activate qualified caller replacements and operator entrypoints

- [ ] **Z03 accepted** — owner lane: `integration-retirement`.

**Start after:** P21, Z01. **Additional acceptance prerequisites:** M43, M44, M45.

**Files:** `distribution/qualify.py`, `distribution/`, `host/conformance/`, `.github/workflows/`, `operator documentation paths in retirement-map.json`.

**Concrete change:** Apply submitted M43–M45 edits through one integration writer. Activate each caller only after its own referenced replacement scope is qualified; keep independent out-of-scope compiler/runtime gates separate. Move selection/scenario/oracle/aggregation policy into Can, leaving only generic invocation and provisioning.

**Acceptance checks:** Partial/unrun/malformed or invalidated report plus receipt is rejected; required service legs are not hidden skips. No active caller invokes retired host policy or counts host unit invocations as migrated Can coverage.

**Completion evidence:** evidence/Z03.json plus per-row/file control and commit links.

**Early commit:** `refactor(testing): activate qualified Can callers`.

## Z04 — Retire transitional bootstrap scenario harnesses

- [ ] **Z04 accepted** — owner lane: `integration-retirement`.

**Start after:** Z01, P23. **Additional acceptance prerequisites:** I01, I02, I03, I04, I05, I06, I07, I08, I09, I10, I11, I12, I13, I14, I15, I16, I17.

**Files:** `tools/native-test-bootstrap/`, `tests/native-can/bootstrap/`, `docs/implementation/native-can-tests-plan-2026-09-30/evidence/bootstrap-retirement/`.

**Concrete change:** Use the accepted outer Can judge to own ongoing runner/owner qualification. Remove transitional host scenario/verdict policy from active suite/CI/release paths after equivalent independent Can controls land. Retain only generic finite native witnessing/ownership mechanisms and compact dated bootstrap evidence.

**Acceptance checks:** Kill subordinate controller/owner while independent outer witness survives; loss of the outer judge stays incomplete. Reverse-call scan confirms no permanent alternate host test suite. Do not delete bootstrap evidence or pretend candidate self-comparison replaces independent authority.

**Completion evidence:** evidence/Z04.json plus per-row/file control and commit links.

**Early commit:** `refactor(testing): retire transitional host test policy`.

## Z05 — Audit full completion and run the authorized qualification matrix

- [ ] **Z05 accepted** — owner lane: `integration-retirement`.

**Start after:** Z01, Z02, Z03, Z04, P24, M01, M02, M03, M04, M05, M06, M07, M08, M09, M10, M11, M12, M13, M14, M15, M16, M17, M18, M19, M20, M21, M22, M23, M24, M25, M26, M27, M28, M29, M30, M31, M32, M33, M34, M35, M36, M37, M38, M39, M40, M41, M42, M43, M44, M45, QN1, QN2, QN3, QB1, QB2, QB3, QB4, QB5, QF1, QF2, QWS, QD1, QD2, QD3, QD4, QStore, QB0, QB1base, QHTTP. **Additional acceptance prerequisites:** none.

**Files:** `docs/implementation/native-can-tests-plan-2026-09-30/evidence/completion/`.

**Concrete change:** Reconcile fresh inventory and all retained environments, dynamic variants, 30 delegated suites, 14 hard gates and caller receipts. Run the selected complete matrix serially under accepted profiles; reuse builds within the run. Reconcile deferred metrics explicitly: keep completion blocked until authorized retained measurement legs run, or a reviewed current-contract retirement decision removes an obsolete obligation.

**Acceptance checks:** All retained coverage has Can-owned scenarios/oracles, independent facts and valid cleanup/correction-aware receipts. Historical/static dispositions are reviewed. No retained obligation disappears because of missing service, Linux/engine profile or deferred load. Broad performance work remains subject to existing user deferral.

**Completion evidence:** evidence/Z05.json plus per-row/file control and commit links.

**Early commit:** `test(testing): record complete migration qualification`.

## Z06 — Request independent final review and resolve findings

- [ ] **Z06 accepted** — owner lane: `integration-retirement`.

**Start after:** Z05. **Additional acceptance prerequisites:** none.

**Files:** `docs/implementation/native-can-tests-plan-2026-09-30/evidence/final-review/`.

**Concrete change:** Send the main plan’s final review prompt with exact commit range and evidence to an independent reviewer. Fix actionable findings in small commits; rerun affected checks and any invalidated acceptance. Reconcile current source hashes and record open limitations without declaring completion.

**Acceptance checks:** Reviewer checks coverage, hidden host policy, R/N/C authority, native/browser/descriptor/DB controls, lifecycle/resource scope and deletion order. Finish only with no unresolved completion blocker, clean owned scratch and current accepted evidence; never infer permission to publish.

**Completion evidence:** evidence/Z06.json plus per-row/file control and commit links.

**Early commit:** `docs(testing): record independent migration review`.

