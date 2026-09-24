# Implementation-plan review record

24–25 September 2026. This is an audit of planning documents against current
source and selected contracts. It is not qualification of the proposed code.

The initial source audits identified the shared checker/emitter files, current
action `handles` coupling, request snapshot lifetime, browser Node dependencies,
unpruned emission and asset-manifest gaps. The task list puts common compiler
edits in one lane, sequences server lifetime/dispatch changes, reserves generated
outputs for integration and adds complete module/installed-build proof.

Independent dependency review identified two handoff corrections:

- UP18 needs UP16's shared package snapshot to qualify the actual pairing; UP16
  is now an explicit dependency, even though the conservative wave table
  already placed it earlier.
- UP19 and UP20 can migrate server and grid concurrently, but neither alone
  can claim the combined application has booted. Their completion rules now
  distinguish component checks from the required combined UP21/23 observations.

- Asset tests also require an explicit owner: UP18 now owns
  `tests/integration/assets_test.go` migration and UP23 owns its browser script.
  Source inspection showed those handwritten launchers boot the **server** for
  lower-level asset tests, rather than injecting Can browser semantics. Useful
  tests remain with updated ABI/HTML policy; they are not counted as supported
  browser startup evidence. Required Chromium/WebKit qualification uses UP23/25.

The independent coverage review found all 11 Fix outcomes and the complete
invoice mutation/installed acceptance matrix represented, including exact
Origin, replay expiry/auth, request revocation, full runtime closure, atomic
generic proof, attached focus and the exclusion of test-provided semantics.
It identified no unintended compatibility promise or deferred feature added
to scope. The dependency review found the stated graph and waves acyclic and
ordered, subject to the corrected handoffs above. The machine validator checks
the final tables separately and records their content hashes.

No new public-language decision was needed to order these tasks. Production
proof remains entirely in the listed tasks; planning review and Jev advice
cannot complete them.
