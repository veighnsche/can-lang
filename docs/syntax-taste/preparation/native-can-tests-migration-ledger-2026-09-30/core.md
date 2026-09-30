# Native Can migration ledger: core

Status: **proposed replacements; implementation and parity evidence pending**.

[Overview](../../native-can-tests-migration-ledger-2026-09-30.md) · [Machine-readable rows](core.json)

Every deletion condition is conjunctive with the overview's common gate. It covers **all** protected facets, variants and delegated oracles, even where a row's short replacement sentence mentions only its first facet. Row IDs name coverage obligations, not one-to-one implementation files.

## CORE-001

**TestCurrentBundledArrays** — [tests/integration/arrays_test.go](/Users/vince/Projects/can-lang/tests/integration/arrays_test.go:16)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-001`. Evidence: pending.

**Protects**

- Array map/filter/search/reduce/sort/copy preserve order, aliases, immutable sources, callback scheduling and failure stop
- main and contracts fixtures yield 46 and 28 passing Can assertion roots
- production run is silent and strict emitted TypeScript optionally typechecks

**Current observations:** Staged runtime/array.test.ts plus Can assert JSON and run exit/stdout/stderr optional strict tsc

**Variants:** main 46 roots contracts 28 roots callback failure, owner context, thenable/getter hostility and native delegation

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional with CAN_BUN

**Capabilities:** CAN PROC FILES WORK NATIVE BUILD SUITE

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Array map/filter/search/reduce/sort/copy preserve order, aliases, immutable sources, callback scheduling and failure stop. Native infrastructure may supply raw process, file, or host observations only. Every related runtime TypeScript test scenario and pass/fail oracle must be reauthored in Can; for host-only malformed values, a generic native probe may construct Can-selected inputs and return raw observations.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [runtime/test/array.test.ts](/Users/vince/Projects/can-lang/runtime/test/array.test.ts); [compiler/testdata/current/arrays/main.can](/Users/vince/Projects/can-lang/compiler/testdata/current/arrays/main.can); [compiler/testdata/current/arrays/contracts.can](/Users/vince/Projects/can-lang/compiler/testdata/current/arrays/contracts.can)

**Notes:** The Go test matches a pinned human-readable Bun pass-count substring; source currently contains more textual Bun test declarations (and array has dynamic callback variants). Reconcile per-declaration obligations and actual selection before migration; do not inherit the old count as a compatibility target. No suite run or failure claimed.

## CORE-002

**TestCurrentBundledAssertions** — [tests/integration/assertions_test.go](/Users/vince/Projects/can-lang/tests/integration/assertions_test.go:16)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-002`. Evidence: pending.

**Protects**

- Structured assertion report, full and selected scopes, supplied versus real completion
- mismatch and malformed/static fixtures fail and do not publish
- duplicate/ambiguous selectors and exact full selector
- native fixture argument validation, shared queue exhaustion/leftovers and callable creation path
- caught external I/O cannot make missing fixture pass, supplied boundary can, domain completion and initialization failure

**Current observations:** canlc assert JSON kind/schema/roots/evidence/status with separate stderr dist/current.json before/after optional strict tsc

**Variants:** basic, native-slice, queues fixtures full, short and qualified selectors bad expected value, bad fixture type, duplicate root, bad arguments, missing/unused rows, caught I/O, domain and initializer

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional with CAN_BUN

**Capabilities:** CAN PROC FILES WORK DIAG BUILD SUITE FAULT

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Structured assertion report, full and selected scopes, supplied versus real completion. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/assertions/basic.can](/Users/vince/Projects/can-lang/compiler/testdata/current/assertions/basic.can); [compiler/testdata/current/assertions/native-slice.can](/Users/vince/Projects/can-lang/compiler/testdata/current/assertions/native-slice.can); [compiler/testdata/current/assertions/queues.can](/Users/vince/Projects/can-lang/compiler/testdata/current/assertions/queues.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-003

**TestAssertionFailureLocations** — [tests/integration/assertions_test.go](/Users/vince/Projects/can-lang/tests/integration/assertions_test.go:262)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-003`. Evidence: pending.

**Protects**

- Initialization failure across two roots and caught sticky missing-fixture violation retain exact Can source byte span, line and column
- failure reports omit private paths, secret, stack and application payload

**Current observations:** Assertion JSON entries/reason/frames and rejected status with empty stderr scan output for forbidden host details

**Variants:** initialization failure caught io::stdout_write fixture violation

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec

**Capabilities:** CAN PROC FILES WORK DIAG SUITE FAULT

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Initialization failure across two roots and caught sticky missing-fixture violation retain exact Can source byte span, line and column. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-004

**TestCurrentBundledBytes** — [tests/integration/bytes_test.go](/Users/vince/Projects/can-lang/tests/integration/bytes_test.go:15)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-004`. Evidence: pending.

**Protects**

- Byte roundtrip Can assertions and production execution preserve byte data and offline behavior

**Current observations:** canlc assert JSON roots and run output/status

**Variants:** roundtrip.can

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec

**Capabilities:** CAN PROC FILES WORK BUILD SUITE

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Byte roundtrip Can assertions and production execution preserve byte data and offline behavior. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/bytes/roundtrip.can](/Users/vince/Projects/can-lang/compiler/testdata/current/bytes/roundtrip.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-005

**TestCurrentBundledCallables** — [tests/integration/callables_test.go](/Users/vince/Projects/can-lang/tests/integration/callables_test.go:16)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-005`. Evidence: pending.

**Protects**

- captures.can has 11 passing Can assertion roots and a silent production run
- Captured callable invocation retains selected assertion root context: unsupplied I/O boundary is a missing fixture without leaking payload; merely constructing callable reference does not execute its target

**Current observations:** canlc assert JSON roots and run output/status

**Variants:** captures fixture selected guarded callable with and without relay

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec

**Capabilities:** CAN PROC FILES WORK DIAG BUILD SUITE FAULT

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Callable capture fixture assertions and production execution preserve closure/capture behavior. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/callables/captures.can](/Users/vince/Projects/can-lang/compiler/testdata/current/callables/captures.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-006

**TestCurrentBundledChecks** — [tests/integration/checks_test.go](/Users/vince/Projects/can-lang/tests/integration/checks_test.go:16)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-006`. Evidence: pending.

**Protects**

- checks require true/false reason, source occurrence, hostile reason validation, JSON preservation and sticky harness violation
- main fixture assertions and silent production run

**Current observations:** Staged runtime/checks.test.ts Can assert JSON/report root count and run exit/output optional strict tsc

**Variants:** main fixture hostile reasons and caught failures

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional with CAN_BUN

**Capabilities:** CAN PROC FILES WORK NATIVE DIAG BUILD SUITE

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: checks require true/false reason, source occurrence, hostile reason validation, JSON preservation and sticky harness violation. Native infrastructure may supply raw process, file, or host observations only. Every related runtime TypeScript test scenario and pass/fail oracle must be reauthored in Can; for host-only malformed values, a generic native probe may construct Can-selected inputs and return raw observations.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [runtime/test/checks.test.ts](/Users/vince/Projects/can-lang/runtime/test/checks.test.ts); [compiler/testdata/current/checks/main.can](/Users/vince/Projects/can-lang/compiler/testdata/current/checks/main.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-007

**TestChecksMismatchSpan** — [tests/integration/checks_test.go](/Users/vince/Projects/can-lang/tests/integration/checks_test.go:119)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-007`. Evidence: pending.

**Protects**

- Failed require over decoded value records explicit reason call site, frames, evidence and exact public source span while suppressing private root/bundle paths

**Current observations:** Failed assert report JSON, status 1, empty stderr, inspected frames/evidence and redaction

**Variants:** one failing decoded-value require

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec

**Capabilities:** CAN PROC FILES WORK DIAG SUITE FAULT

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Failed require over decoded value records explicit reason call site, frames, evidence and exact public source span while suppressing private root/bundle paths. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-008

**TestCurrentBundledCLI** — [tests/integration/cli_test.go](/Users/vince/Projects/can-lang/tests/integration/cli_test.go:16)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-008`. Evidence: pending.

**Protects**

- Offline staged CLI via symlink ignores hostile BUN_OPTIONS, forwards Unicode, flag-like and empty argv literally, and emits exact stdout
- Closed stdout pipe maps to io::write_failed; cross-file call and forward initializer work
- Build JSON identity, static entry/type failures, runtime bounds, initializer failure, domain redaction and malformed CLI usage preserve correct status and current selection

**Current observations:** canlc build/run/assert status and separated streams pipe failure current.json identity diagnostic JSON

**Variants:** argv: Unicode/newline, --inspect, --, -e, preload-like, empty, BOM and spaces closed pipe cross-module forward initializer bad entry, static mismatch, bounds, initializer, domain failure, malformed invocations

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec

**Capabilities:** CAN PROC FILES WORK DIAG BUILD SUITE FAULT

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Offline staged CLI forwards literal argv and emits expected stdout. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/cli/echo.can](/Users/vince/Projects/can-lang/compiler/testdata/current/cli/echo.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-009

**TestCurrentBundledCodec** — [tests/integration/codec_test.go](/Users/vince/Projects/can-lang/tests/integration/codec_test.go:15)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-009`. Evidence: pending.

**Protects**

- Codec roundtrip Can assertions and production execution preserve encoding/decoding behavior

**Current observations:** canlc assert JSON roots and run output/status

**Variants:** roundtrip.can

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec

**Capabilities:** CAN PROC FILES WORK BUILD SUITE

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Codec roundtrip Can assertions and production execution preserve encoding/decoding behavior. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/codec/roundtrip.can](/Users/vince/Projects/can-lang/compiler/testdata/current/codec/roundtrip.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-010

**TestCurrentBundledCollections** — [tests/integration/collections_test.go](/Users/vince/Projects/can-lang/tests/integration/collections_test.go:16)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-010`. Evidence: pending.

**Protects**

- Map/set immutable copy and nominal key behavior
- forged and revoked values reject safely
- native set order/equality and atomic bulk construction
- Can fixture assertions and silent production run

**Current observations:** Staged runtime/collections.test.ts Can assert JSON/report count and run output optional strict tsc

**Variants:** main 19 roots duplicate bulk key, forged token, revoked backing, intersection branches

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional with CAN_BUN

**Capabilities:** CAN PROC FILES WORK NATIVE BUILD SUITE

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Map/set immutable copy and nominal key behavior. Native infrastructure may supply raw process, file, or host observations only. Every related runtime TypeScript test scenario and pass/fail oracle must be reauthored in Can; for host-only malformed values, a generic native probe may construct Can-selected inputs and return raw observations.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [runtime/test/collections.test.ts](/Users/vince/Projects/can-lang/runtime/test/collections.test.ts); [std/map/current/src/main.can](/Users/vince/Projects/can-lang/std/map/current/src/main.can)

**Notes:** The Go test matches a pinned human-readable Bun pass-count substring; source currently contains more textual Bun test declarations (and array has dynamic callback variants). Reconcile per-declaration obligations and actual selection before migration; do not inherit the old count as a compatibility target. No suite run or failure claimed.

## CORE-011

**TestCurrentBundledCookiesCSRF** — [tests/integration/cookies_csrf_test.go](/Users/vince/Projects/can-lang/tests/integration/cookies_csrf_test.go:17)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-011`. Evidence: pending.

**Protects**

- 18 Can assertion roots include supplied-completion and real-can evidence
- Compiled boot/mint plus owned server: establish returns session cookie; token without cookie, wrong session and bogus token reject; correct session succeeds; stop and wait cleanly

**Current observations:** Can assert JSON and production build/run; generated-function presence and optional strict tsc Current embedded TypeScript harness supplies HTTP scenario and oracle for status/body/cookies plus owner cleanup; this must move to Can

**Variants:** establish; bare token; matching theme/session cookie; wrong session; bogus token; stop/wait

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional

**Capabilities:** CAN PROC FILES WORK HTTP CHILD BUILD SUITE

**Proposed Can replacement:** Write ordinary Can scenario/helpers that boot server, mint token, issue each request, compare status/body/Set-Cookie and assert owned stop/wait completion; generic native HTTP/process mechanics may expose observations. Keep staged Can assertion roots as separate coverage; retire the embedded TypeScript scenario only after parity.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/cookies/session.can](/Users/vince/Projects/can-lang/compiler/testdata/current/cookies/session.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-012

**TestCurrentBundledCoordination** — [tests/integration/coordination_test.go](/Users/vince/Projects/can-lang/tests/integration/coordination_test.go:16)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-012`. Evidence: pending.

**Protects**

- Promise all/allSettled/any/race selection, ordered outcomes, empty modes, domain/standard failures, owner retention and native delegation
- Can fixture assertions and silent production run

**Current observations:** Staged runtime/coordination.test.ts assert JSON and run status/output optional strict tsc

**Variants:** main fixture late/losing failures, handler faults, empty race and snapshot identity

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional with CAN_BUN

**Capabilities:** CAN PROC FILES WORK NATIVE CHILD BUILD SUITE

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Promise all/allSettled/any/race selection, ordered outcomes, empty modes, domain/standard failures, owner retention and native delegation. Native infrastructure may supply raw process, file, or host observations only. Every related runtime TypeScript test scenario and pass/fail oracle must be reauthored in Can; for host-only malformed values, a generic native probe may construct Can-selected inputs and return raw observations.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [runtime/test/coordination.test.ts](/Users/vince/Projects/can-lang/runtime/test/coordination.test.ts); [compiler/testdata/current/coordination/main.can](/Users/vince/Projects/can-lang/compiler/testdata/current/coordination/main.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-013

**TestStandardSnapshotIdentity** — [tests/integration/coordination_test.go](/Users/vince/Projects/can-lang/tests/integration/coordination_test.go:118)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-013`. Evidence: pending.

**Protects**

- One standard occurrence seen through ordinary catch and coordination aggregate retains identity rather than minting a new error

**Current observations:** Can assertion report and optional generated TypeScript typecheck

**Variants:** catch then aggregate of same standard occurrence

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional with CAN_BUN

**Capabilities:** CAN PROC FILES WORK NATIVE BUILD SUITE

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: One standard occurrence seen through ordinary catch and coordination aggregate retains identity rather than minting a new error. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-014

**TestCurrentCryptoCommands** — [tests/integration/crypto_test.go](/Users/vince/Projects/can-lang/tests/integration/crypto_test.go:49)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-014`. Evidence: pending.

**Protects**

- Crypto assertions, deterministic build identity across rebuild and relocation, production crypto wiring, strict emission and live native commands
- wrong argument type and missing emits reject
- roundtrip/HMAC/password cases pass, key misuse and tampering return domain errors

**Current observations:** Can assert JSON, build IDs/artifact inspection, optional tsc fd-3 env snapshot production process output/status

**Variants:** roundtrip, hmac, ask correct/wrong, misuse, tamper two negative source mutations

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional

**Capabilities:** CAN PROC FILES WORK NATIVE DIAG BUILD SUITE FAULT

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Crypto assertions, deterministic build identity across rebuild and relocation, production crypto wiring, strict emission and live native commands. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/crypto/commands.can](/Users/vince/Projects/can-lang/compiler/testdata/current/crypto/commands.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-015

**TestDevelopmentSidecar** — [tests/integration/distribution_test.go](/Users/vince/Projects/can-lang/tests/integration/distribution_test.go:19)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-015`. Evidence: pending.

**Protects**

- Pinned macOS arm64 distribution build is offline, immutable, non-overwriting and symlink launcher uses packaged Bun while ignoring hostile ambient configs and keeping env/argv
- Parser rendering is inert and idempotent; obsolete grammar rejects; project identities remain relocation-stable and read-only, stale lock rejects; type inspection rejects uninhabited record
- Native data, primitive, failure/domain and completion contracts from delegated runtime suites; missing, non-executable, tampered, wrong-arch and symlinked sidecar plus tampered manifest/tool reject

**Current observations:** Independent source/bundle tree hashes packaged runtime-check JSON and parse/inspect diagnostics staged native suite reports mutation exit codes

**Variants:** parser, project, declaration, native data/primitives/failures/completions sidecar missing/permission/integrity/arch/path manifest and tool corruption

**Environment/selection gates:** CAN_BUN_ARCHIVE pinned archive macOS arm64 sandbox-exec

**Capabilities:** CAN PROC FILES WORK ARCHIVE NATIVE DIAG BUILD ENV SUITE FAULT

**Proposed Can replacement:** Can-owned cases select all distribution, parser, identity, negative mutation and native contract scenarios and compare independent process/file/host observations. Delegated runtime/data, primitive, failure, domain and completion test declarations require Can oracles, not a Can call to bun test; archive/signature mechanics may remain native.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/parser/offline.can](/Users/vince/Projects/can-lang/compiler/testdata/current/parser/offline.can); [compiler/testdata/current/project](/Users/vince/Projects/can-lang/compiler/testdata/current/project); [runtime/data.test.ts](/Users/vince/Projects/can-lang/runtime/data.test.ts); [runtime/primitive.test.ts](/Users/vince/Projects/can-lang/runtime/primitive.test.ts); [runtime/failure.test.ts](/Users/vince/Projects/can-lang/runtime/failure.test.ts); [runtime/domain.test.ts](/Users/vince/Projects/can-lang/runtime/domain.test.ts); [runtime/completion.test.ts](/Users/vince/Projects/can-lang/runtime/completion.test.ts); [distribution](/Users/vince/Projects/can-lang/distribution)

**Notes:** Current runnable obligation, with packaged native suite pass counts as smoke evidence only. No exact old generated layout is a compatibility target.

## CORE-016

**TestDistributionShipsBrowserBundleTool** — [tests/integration/distribution_test.go](/Users/vince/Projects/can-lang/tests/integration/distribution_test.go:359)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-016`. Evidence: pending.

**Protects**

- Distribution manifest binds browser bundle tool and browser/runtime dependencies
- tool has native browser bundling protocol and excludes plugin/alias/external injection machinery

**Current observations:** Read shipped bytes and verify digest against distribution manifest inspect required and forbidden tool content

**Variants:** browser bundle tool and named browser runtime assets

**Environment/selection gates:** CAN_BUN_ARCHIVE pinned archive

**Capabilities:** CAN FILES ARCHIVE BUILD SUITE

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Distribution manifest binds browser bundle tool and browser/runtime dependencies. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [tools/runtime/browser-bundle.ts](/Users/vince/Projects/can-lang/tools/runtime/browser-bundle.ts); [runtime/browser/formats.ts](/Users/vince/Projects/can-lang/runtime/browser/formats.ts); [runtime/browser/cookies.ts](/Users/vince/Projects/can-lang/runtime/browser/cookies.ts); [runtime/browser/csrf.ts](/Users/vince/Projects/can-lang/runtime/browser/csrf.ts); [runtime/browser/clock.ts](/Users/vince/Projects/can-lang/runtime/browser/clock.ts); [runtime/browser/log.ts](/Users/vince/Projects/can-lang/runtime/browser/log.ts); [runtime/browser/markdown.ts](/Users/vince/Projects/can-lang/runtime/browser/markdown.ts); [runtime/browser/assets.ts](/Users/vince/Projects/can-lang/runtime/browser/assets.ts); [runtime/assert/lineage.ts](/Users/vince/Projects/can-lang/runtime/assert/lineage.ts)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-017

**TestCurrentBundledExactAmounts** — [tests/integration/exact_amount_test.go](/Users/vince/Projects/can-lang/tests/integration/exact_amount_test.go:16)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-017`. Evidence: pending.

**Protects**

- Signed exact divmod, half-even rounding, large rational/minor units and zero-denominator domain errors
- main Can assertions and silent production run

**Current observations:** Staged runtime/exact-amount.test.ts Can assert JSON and run output/status optional strict tsc

**Variants:** main 23 roots both signs, parity ties, huge integers, zero denominators

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional with CAN_BUN

**Capabilities:** CAN PROC FILES WORK NATIVE BUILD SUITE

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Signed exact divmod, half-even rounding, large rational/minor units and zero-denominator domain errors. Native infrastructure may supply raw process, file, or host observations only. Every related runtime TypeScript test scenario and pass/fail oracle must be reauthored in Can; for host-only malformed values, a generic native probe may construct Can-selected inputs and return raw observations.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [runtime/test/exact-amount.test.ts](/Users/vince/Projects/can-lang/runtime/test/exact-amount.test.ts); [std/ratio/current/src/main.can](/Users/vince/Projects/can-lang/std/ratio/current/src/main.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-018

**TestCurrentBundledFetch** — [tests/integration/fetch_test.go](/Users/vince/Projects/can-lang/tests/integration/fetch_test.go:83)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-018`. Evidence: pending.

**Protects**

- Raw offline fetch fixtures and selective recovery cover eight operations plus null exchange
- live server validates method, query duplicate/order/encoding, headers and auth, body media and exact big integer
- response status/media/charset/malformed data fail correctly
- invalid computed header blocks before credentials or requests

**Current observations:** Can assert JSON 20 roots independent httptest request count/headers/body and process status/diagnostics optional strict tsc

**Variants:** GET/PUT/POST/PATCH/HEAD/DELETE/OPTIONS and envelope status/media/charset/malformed invalid header and missing credential

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_I26_TOKEN supplied CAN_TSC optional with CAN_BUN

**Capabilities:** CAN PROC FILES WORK HTTP NATIVE DIAG BUILD SUITE FAULT PEER

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Raw offline fetch fixtures and selective recovery cover eight operations plus null exchange. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/fetch/main.can](/Users/vince/Projects/can-lang/compiler/testdata/current/fetch/main.can); [compiler/testdata/current/fetch/fixtures](/Users/vince/Projects/can-lang/compiler/testdata/current/fetch/fixtures)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-019

**TestCurrentBundledFiles** — [tests/integration/files_test.go](/Users/vince/Projects/can-lang/tests/integration/files_test.go:64)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-019`. Evidence: pending.

**Protects**

- Native file bytes/text, errors, limits, exclusive create, symlinks, list/copy/move/remove/glob/path and unsupplied-assertion isolation
- Can live file tree writes and reads
- wrong type, unhandled error and unknown operation reject with source diagnostics
- generated code uses native file operations

**Current observations:** Staged runtime/test/files.test.ts Can assert JSON and run output plus independent disk bytes compiler rejection diagnostics and production artifact inspection

**Variants:** 15 runtime cases live file tree three rejection programs

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec

**Capabilities:** CAN PROC FILES WORK NATIVE DIAG BUILD SUITE FAULT

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Native file bytes/text, errors, limits, exclusive create, symlinks, list/copy/move/remove/glob/path and unsupplied-assertion isolation. Native infrastructure may supply raw process, file, or host observations only. Every related runtime TypeScript test scenario and pass/fail oracle must be reauthored in Can; for host-only malformed values, a generic native probe may construct Can-selected inputs and return raw observations.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [runtime/test/files.test.ts](/Users/vince/Projects/can-lang/runtime/test/files.test.ts)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-020

**TestFormatPreservesExecution** — [tests/integration/format_test.go](/Users/vince/Projects/can-lang/tests/integration/format_test.go:35)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-020`. Evidence: pending.

**Protects**

- Formatter changes messy source while retaining comments and assertion root identities
- second format is byte-idempotent
- invalid source exits 1 without touching file

**Current observations:** Before/after Can assert JSON roots file byte comparison format command status

**Variants:** messy valid source, second pass, syntactically invalid source

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN PROC FILES WORK FMT DIAG SUITE FAULT

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Formatter changes messy source while retaining comments and assertion root identities. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-021

**TestCurrentFormatsDocuments** — [tests/integration/formats_test.go](/Users/vince/Projects/can-lang/tests/integration/formats_test.go:49)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-021`. Evidence: pending.

**Protects**

- Can assertions plus compiled CLI parse TOML, YAML, JSON5 and JSONL documents with exact output, including 64-bit JSONL numbers and blank lines
- Usage paths for missing/unknown commands; absent file and malformed TOML/JSONL yield exact domain errors

**Current observations:** Can assertion JSON, production build artifact and fd-3 environment snapshot, output/status

**Variants:** usage nil/bogus/toml valid svc.toml, point.yaml, point.json5, points.jsonl missing TOML, duplicate-key TOML, truncated JSONL

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec

**Capabilities:** CAN PROC FILES WORK NATIVE BUILD SUITE

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Document format rendering/decoding and compiled CLI commands preserve bytes, output and domain failures. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [tests/integration/testdata/formats/main.can](/Users/vince/Projects/can-lang/tests/integration/testdata/formats/main.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-022

**TestCurrentBundledGeneration** — [tests/integration/generation_test.go](/Users/vince/Projects/can-lang/tests/integration/generation_test.go:21)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-022`. Evidence: pending.

**Protects**

- LLM generation fixture assertions and live request/response path
- refusal, truncation, invalid response and bad record decode map to distinct errors
- blank ask and missing auth reject before launch
- strict emitted TypeScript

**Current observations:** Can assert JSON, independent local HTTP request observations, run status/diagnostics and optional tsc

**Variants:** refusal, truncated, invalid, record blank ask, auth omission

**Environment/selection gates:** CAN_BUN_ARCHIVE CAN_I28_TOKEN supplied macOS sandbox-exec CAN_TSC optional

**Capabilities:** CAN PROC FILES WORK HTTP NATIVE DIAG BUILD SUITE FAULT PEER

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: LLM generation fixture assertions and live request/response path. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/native/generation.can](/Users/vince/Projects/can-lang/compiler/testdata/current/native/generation.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-023

**TestCurrentBundledGenerics** — [tests/integration/generics_test.go](/Users/vince/Projects/can-lang/tests/integration/generics_test.go:20)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-023`. Evidence: pending.

**Protects**

- definitions/helper/main have six passing Can roots and silent production run
- A reachable concrete generic instance rejects a wrong-type branch even when that branch is never taken at runtime
- finite-transition, finite-field-chain and spread-inferred source variants each assert and run successfully, including added array/terminal/inferred forms

**Current observations:** Can assert JSON six roots, silent run, build rejection and diagnostics

**Variants:** definitions/helper/main wrong-type unreachable runtime branch finite-transition base, extra array, inferred call finite-field-chain base, terminal before/after spread-inferred

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec

**Capabilities:** CAN PROC FILES WORK DIAG BUILD SUITE FAULT

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Generic definitions/helper/main assert and run. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/generics/definitions.can](/Users/vince/Projects/can-lang/compiler/testdata/current/generics/definitions.can); [compiler/testdata/current/generics/helper.can](/Users/vince/Projects/can-lang/compiler/testdata/current/generics/helper.can); [compiler/testdata/current/generics/main.can](/Users/vince/Projects/can-lang/compiler/testdata/current/generics/main.can); [compiler/testdata/current/generics/finite-transition.can](/Users/vince/Projects/can-lang/compiler/testdata/current/generics/finite-transition.can); [compiler/testdata/current/generics/finite-field-chain.can](/Users/vince/Projects/can-lang/compiler/testdata/current/generics/finite-field-chain.can); [compiler/testdata/current/generics/spread-inferred.can](/Users/vince/Projects/can-lang/compiler/testdata/current/generics/spread-inferred.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-024

**TestCurrentBundledGenericChain** — [tests/integration/generics_test.go](/Users/vince/Projects/can-lang/tests/integration/generics_test.go:142)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-024`. Evidence: pending.

**Protects**

- Public generic dependency chain verifies locked source/fixture identity, nested concrete instance behavior, canonical instance emission and source-map links
- strict TS and nonvacuous expected-value corruption
- stale lock, failed component, owner negatives reject without false publication

**Current observations:** Can assertions/build/run output, independent lock digest, production artifact and map inspection, optional tsc source mutations

**Variants:** root and local dependency nested<int>, identity<box<int>> 14 instances assertion corruption, stale lock, symbolic component and owner negatives

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional

**Capabilities:** CAN PROC FILES WORK DIAG BUILD SUITE FAULT

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Public generic dependency chain verifies locked source/fixture identity, nested concrete instance behavior, canonical instance emission and source-map links. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/generics](/Users/vince/Projects/can-lang/compiler/testdata/current/generics)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-025

**TestCurrentBundledHTML** — [tests/integration/html_test.go](/Users/vince/Projects/can-lang/tests/integration/html_test.go:18)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-025`. Evidence: pending.

**Protects**

- Can HTML assertions carry real-can evidence
- Request-derived hostile text is escaped in title and fragment; output retains expected htmx attributes/assets, two safe scripts and no injected script/image; fragment render equals native escape result

**Current observations:** Can assert and build plus optional strict tsc Current embedded TypeScript harness fetches hostile request text, renders compiled Can functions, compares HTML bytes and injection absence

**Variants:** hostile input with script/image/ampersand/quotes; page and fragment

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional

**Capabilities:** CAN PROC FILES WORK HTTP NATIVE BUILD SUITE

**Proposed Can replacement:** Can scenario chooses hostile request input, calls render/fragment and compares complete relevant HTML observations including escaping and forbidden injected tags; native HTTP and HTML serialization may provide raw bytes only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/html/main.can](/Users/vince/Projects/can-lang/compiler/testdata/current/html/main.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-026

**TestCurrentBundledHTTP** — [tests/integration/http_test.go](/Users/vince/Projects/can-lang/tests/integration/http_test.go:17)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-026`. Evidence: pending.

**Protects**

- 18 Can assertion roots include supplied-completion and real-can evidence
- Compiled mounted router handles GET query cardinality/encoding, POST JSON/media/malformed and form duplicate/empty/invalid percent cases
- Generic/wrapped routes, 404/405 with Allow, PUT/PATCH/DELETE/OPTIONS/HEAD behavior, bounded upload, stream echo, event stream and multipart attachment preserve exact status/header/body

**Current observations:** Can assert JSON and build/run; optional strict tsc Current embedded TypeScript harness drives private ingress and checks replies; its scenario/oracle must move to Can

**Variants:** GET /a valid, absent, repeated, plus and percent-encoded query POST /b JSON, media and malformed; /s form duplicate/optional/empty/invalid /g and /w; missing/method mismatches and Allow PUT/PATCH/DELETE/OPTIONS/HEAD; upload/echo/events/multipart

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional

**Capabilities:** CAN PROC FILES WORK HTTP CHILD BUILD SUITE

**Proposed Can replacement:** Can-owned normal scenario builds router, selects every listed request, dispatches over generic loopback ingress and compares status, headers, body, Allow and cleanup. Native ingress and HTTP client may return raw observations, never case-specific verdicts.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/http/main.can](/Users/vince/Projects/can-lang/compiler/testdata/current/http/main.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-027

**TestCurrentBundledInputCapture** — [tests/integration/input_capture_test.go](/Users/vince/Projects/can-lang/tests/integration/input_capture_test.go:46)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-027`. Evidence: pending.

**Protects**

- Four assertion roots include raw-provider-fixture evidence and timeout policy; independently length-framed lock digest binds vendor source and raw fixture trees
- Changing only timeout from 5000 to 6000 yields new options identity with unchanged source/dependency identity and second staged generation
- Staged modules carry captured raw response bytes; fixture-only dependency change rejects as stale digest without rewriting lock

**Current observations:** Can assert JSON/timeout/evidence; independently computed source and fixture lock digests Staged manifest input identities and emitted fixture byte marker; rejection diagnostic and unchanged lock

**Variants:** assert timeout 5000 and 6000 fixture bytes before and after mutation

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec

**Capabilities:** CAN PROC FILES WORK BUILD NATIVE DIAG SUITE FAULT

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Captured dependency source and raw fixture bytes affect build identity. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-028

**TestCurrentBundledInputEnvironment** — [tests/integration/io_env_test.go](/Users/vince/Projects/can-lang/tests/integration/io_env_test.go:15)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-028`. Evidence: pending.

**Protects**

- Original environment lookup preserves Unicode, empty value and hostile BUN_OPTIONS as data
- missing/invalid names fail with allocated errors
- bounds/fixture mutation remain consistent
- assertions, live run and strict emission

**Current observations:** Can assert report, process stdout/stderr/status under hostile environment and optional tsc

**Variants:** VALUE, EMPTY, BUN_OPTIONS, MISSING, lower, A=B and empty name per-fixture subtests

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional

**Capabilities:** CAN PROC FILES WORK ENV NATIVE DIAG BUILD SUITE

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Original environment lookup preserves Unicode, empty value and hostile BUN_OPTIONS as data. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/cli](/Users/vince/Projects/can-lang/compiler/testdata/current/cli)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-029

**TestCurrentBundledNoulJudge** — [tests/integration/judge_test.go](/Users/vince/Projects/can-lang/tests/integration/judge_test.go:21)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-029`. Evidence: pending.

**Protects**

- Noul judge ordinary consumer uses supplied completion and live provider
- invalid answer, blank descriptor and faulty handler are rejected at correct stage
- raw provider fixture drives compiled Can without host verdict

**Current observations:** Can assert report, independent HTTP provider request observations, build/run status and raw response data

**Variants:** normal, invalid answer, blank descriptor, faulty handler, raw provider

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_I17_TOKEN supplied

**Capabilities:** CAN PROC FILES WORK HTTP NATIVE DIAG BUILD SUITE FAULT PEER

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Noul judge ordinary consumer uses supplied completion and live provider. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/native/noul.can](/Users/vince/Projects/can-lang/compiler/testdata/current/native/noul.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-030

**TestCurrentMarkdownRender** — [tests/integration/markdown_test.go](/Users/vince/Projects/can-lang/tests/integration/markdown_test.go:34)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-030`. Evidence: pending.

**Protects**

- Can markdown assertion fixture passes
- Compiled CLI renders heading, preserves raw HTML per current markdown contract and gives exact usage text for missing/unknown arguments
- Replacing render_safe with text HTML cannot bind html::safe; omitted html::invalid_url emits rejects at build

**Current observations:** Can assert report, build and fd-3 CLI output/status Compiler diagnostics for two source mutations

**Variants:** heading; raw script input; nil/bogus/incomplete usage; two compiler negatives

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec

**Capabilities:** CAN PROC FILES WORK DIAG BUILD SUITE FAULT

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Markdown rendering Can fixture assertions and compiled output preserve supported formatting and escaping. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/markdown/main.can](/Users/vince/Projects/can-lang/compiler/testdata/current/markdown/main.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-031

**TestCurrentBundledNativeDeclarations** — [tests/integration/native_test.go](/Users/vince/Projects/can-lang/tests/integration/native_test.go:15)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-031`. Evidence: pending.

**Protects**

- Native declaration fixture and committed raw fixtures build successfully
- Six changed declarations reject: dynamic question descriptor, unknown model field, wrong result type, missing exported member, dynamic ask and bytes where string is required

**Current observations:** canlc build status and diagnostics for baseline and six mutated Can sources

**Variants:** valid declarations plus six compiler-rejection mutations

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec

**Capabilities:** CAN PROC FILES WORK DIAG BUILD SUITE FAULT

**Proposed Can replacement:** Can-owned cases select valid and six invalid source variants, invoke production compiler and compare rejection phase/diagnostic and build result; static raw fixtures remain inputs.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/native/declarations.can](/Users/vince/Projects/can-lang/compiler/testdata/current/native/declarations.can); [compiler/testdata/current/native/fixtures](/Users/vince/Projects/can-lang/compiler/testdata/current/native/fixtures)

**Notes:** This Go function checks build acceptance/rejection only; it does not run Can assertions or production entry.

## CORE-032

**TestCurrentBundledNumbers** — [tests/integration/numbers_test.go](/Users/vince/Projects/can-lang/tests/integration/numbers_test.go:16)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-032`. Evidence: pending.

**Protects**

- Native number formatting/conversion/parsing/predicates preserve bigint, IEEE and grammar boundaries
- main Can assertions and silent production run

**Current observations:** Staged runtime/number.test.ts Can assert JSON and run output optional strict tsc

**Variants:** main 40 roots bigint, NaN/infinity, exact represented values and invalid parses

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional with CAN_BUN

**Capabilities:** CAN PROC FILES WORK NATIVE BUILD SUITE

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Native number formatting/conversion/parsing/predicates preserve bigint, IEEE and grammar boundaries. Native infrastructure may supply raw process, file, or host observations only. Every related runtime TypeScript test scenario and pass/fail oracle must be reauthored in Can; for host-only malformed values, a generic native probe may construct Can-selected inputs and return raw observations.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [runtime/test/number.test.ts](/Users/vince/Projects/can-lang/runtime/test/number.test.ts); [std/scalars/current/src/main.can](/Users/vince/Projects/can-lang/std/scalars/current/src/main.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-033

**TestBundledOwnershipOffline** — [tests/integration/owners_test.go](/Users/vince/Projects/can-lang/tests/integration/owners_test.go:13)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-033`. Evidence: pending.

**Protects**

- Owner scopes preserve losing outcomes once, deduplicated late diagnostics, leases, native async context and cross-root rejection
- Automatic reverse cleanup reports omitted/failed closes; reentrant close is idempotent, wrong-kind/forged handles fail without touching native state
- CLI/assertion wait for late work; external supervisor kills nonsettling leased and unleased owners

**Current observations:** Staged runtime owner/owner-roots/owner-hung Bun suites currently contain scenarios and pass/fail oracles; Go checks zero fail and named evidence strings

**Variants:** leased=false and true hung owner; omitted/automatic close; late loser; forged/cross-root/escaped handle

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec

**Capabilities:** CAN PROC FILES WORK CHILD NATIVE FAULT SUITE

**Proposed Can replacement:** Can cases and normal helper functions construct owner/lease scenarios, compare order, errors and cleanup; generic native scope/supervisor mechanics expose the independent trace, including forced kill. Migrate each delegated runtime test oracle, not just the Bun suite exit code.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [runtime/test/owner.test.ts](/Users/vince/Projects/can-lang/runtime/test/owner.test.ts); [runtime/test/owner-roots.test.ts](/Users/vince/Projects/can-lang/runtime/test/owner-roots.test.ts); [runtime/test/owner-hung.test.ts](/Users/vince/Projects/can-lang/runtime/test/owner-hung.test.ts)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-034

**TestCurrentBundledProcess** — [tests/integration/process_test.go](/Users/vince/Projects/can-lang/tests/integration/process_test.go:48)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-034`. Evidence: pending.

**Protects**

- Native process stdout/stderr/stdin, exit/signal, literal argv, cwd/env, output/deadline limits, descendant kill and abandoned-run cleanup
- live Can /bin/echo
- wrong type, missing arm, unknown operation reject
- generated code delegates to Bun.spawn/which

**Current observations:** Staged runtime/test/process.test.ts Can assert JSON, live process status/output, compiler diagnostics and artifact inspection

**Variants:** 13 runtime cases live echo three rejection programs

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec /bin/echo system PATH

**Capabilities:** CAN PROC FILES WORK CHILD NATIVE DIAG BUILD SUITE FAULT

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Native process stdout/stderr/stdin, exit/signal, literal argv, cwd/env, output/deadline limits, descendant kill and abandoned-run cleanup. Native infrastructure may supply raw process, file, or host observations only. Every related runtime TypeScript test scenario and pass/fail oracle must be reauthored in Can; for host-only malformed values, a generic native probe may construct Can-selected inputs and return raw observations.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [runtime/test/process.test.ts](/Users/vince/Projects/can-lang/runtime/test/process.test.ts)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-035

**TestCurrentBundledMixedQuestions** — [tests/integration/questions_test.go](/Users/vince/Projects/can-lang/tests/integration/questions_test.go:56)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-035`. Evidence: pending.

**Protects**

- Mixed question consumer handles boolean/string/choice/number/confidence and nested followup
- invalid answer, duplicate criterion, faulty handler/record reject before launch
- lower confidence selects fallback
- raw provider fixture request comparison and strict emission

**Current observations:** Can assert report independent provider requests/response traces, run diagnostics and optional tsc

**Variants:** normal, invalid, duplicate, handler/record faults, confidence fallback, two-stage followup, raw provider

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_I27_TOKEN supplied CAN_TSC optional

**Capabilities:** CAN PROC FILES WORK HTTP NATIVE DIAG BUILD SUITE FAULT PEER

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Mixed question consumer handles boolean/string/choice/number/confidence and nested followup. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/native/questions.can](/Users/vince/Projects/can-lang/compiler/testdata/current/native/questions.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-036

**TestReleaseInstallUpdate** — [tests/integration/release_test.go](/Users/vince/Projects/can-lang/tests/integration/release_test.go:28)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-036`. Evidence: pending.

**Protects**

- Offline signed release install/update and old-version continuity
- detached record, payload, manifest and archive tampering reject
- failed/racing updates preserve selection and reject conflicts

**Current observations:** Independent archive/signature/hash and installed tree checks selected symlink and executable output/status

**Variants:** valid install/update flipped record byte, edited zip/manifest, failed update, racing installs

**Environment/selection gates:** CAN_BUN_ARCHIVE pinned archive macOS arm64

**Capabilities:** CAN PROC FILES WORK ARCHIVE BUILD ENV SUITE FAULT

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Offline signed release install/update and old-version continuity. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [distribution](/Users/vince/Projects/can-lang/distribution)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-037

**TestInstallRootRefusals** — [tests/integration/release_test.go](/Users/vince/Projects/can-lang/tests/integration/release_test.go:360)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-037`. Evidence: pending.

**Protects**

- Update refuses a symlink install root before reading deliberately nonexistent archive/record paths
- Selection on an absent empty root returns no selected version without error

**Current observations:** distribution.Update error text with nonexistent inputs; distribution.Selection empty result

**Variants:** symlink root; absent root

**Environment/selection gates:** CAN_BUN_ARCHIVE presence only

**Capabilities:** CAN PROC FILES WORK ARCHIVE DIAG SUITE FAULT

**Proposed Can replacement:** Can-owned cases choose symlink and absent-root inputs and compare independent installer/selection observations; archive operation mechanics may remain native.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [distribution](/Users/vince/Projects/can-lang/distribution)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-038

**TestCurrentBundledServer** — [tests/integration/server_test.go](/Users/vince/Projects/can-lang/tests/integration/server_test.go:17)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-038`. Evidence: pending.

**Protects**

- Seven Can assertion roots include supplied-completion and real-can evidence
- Compiled Can boot serves /health as 200 healthy; owned stop/wait succeeds, then serve(0) completes with clean owner cleanup

**Current observations:** Can assert JSON and build/run; optional strict tsc Embedded TypeScript harness currently owns health request, status/body expectation and owner cleanup verdict

**Variants:** boot fixed loopback port; GET /health; stop/wait; serve(0)

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional

**Capabilities:** CAN PROC FILES WORK HTTP CHILD BUILD SUITE

**Proposed Can replacement:** Can-owned ordinary scenario calls boot and serve, issues health request, compares status/body and checks stop/wait and cleanup observations; generic native server/client mechanics may expose values.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/http/server.can](/Users/vince/Projects/can-lang/compiler/testdata/current/http/server.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-039

**TestStdlibMaintained** — [tests/integration/stdlib_test.go](/Users/vince/Projects/can-lang/tests/integration/stdlib_test.go:199)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-039`. Evidence: pending.

**Protects**

- Every discovered maintained std/example project has nonempty real-Can assertions
- two builds share identity and no stray emit
- standard projects run clean
- browser-shaped invoice projects use browser target and are covered live elsewhere

**Current observations:** Can assert JSON/evidence independent build IDs and emit filesystem scan run status for std projects

**Variants:** dynamic discoverMaintained std/ and examples/ projects browser target invoice-grid/invoice-compare optional fresh emit copies

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_FRESH_EMIT_DIR optional

**Capabilities:** CAN PROC FILES WORK BUILD BROWSER SUITE

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Every discovered maintained std/example project has nonempty real-Can assertions. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [std](/Users/vince/Projects/can-lang/std); [examples](/Users/vince/Projects/can-lang/examples)

**Notes:** Discovery set must be enumerated in final file map; browser runtime claims remain tied to applications/gate5 cases.

## CORE-040

**TestCurrentBundledStreams** — [tests/integration/streams_test.go](/Users/vince/Projects/can-lang/tests/integration/streams_test.go:196)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-040`. Evidence: pending.

**Protects**

- Native byte/line readers and writers preserve order, UTF-8 framing, caps, cancellation, close/failure semantics and copies
- Can live stream tree, overrun/underrun transcript violations
- wrong type, missing arm, wrong result binding and unknown operation reject

**Current observations:** Staged runtime/test/streams.test.ts Can assert JSON, independent disk bytes and live output, compiler diagnostics

**Variants:** 11 runtime cases live tree transcript over/underrun four rejection programs

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec

**Capabilities:** CAN PROC FILES WORK NATIVE DIAG BUILD SUITE FAULT

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Native byte/line readers and writers preserve order, UTF-8 framing, caps, cancellation, close/failure semantics and copies. Native infrastructure may supply raw process, file, or host observations only. Every related runtime TypeScript test scenario and pass/fail oracle must be reauthored in Can; for host-only malformed values, a generic native probe may construct Can-selected inputs and return raw observations.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [runtime/test/streams.test.ts](/Users/vince/Projects/can-lang/runtime/test/streams.test.ts)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-041

**TestCurrentBundledTemplates** — [tests/integration/templates_test.go](/Users/vince/Projects/can-lang/tests/integration/templates_test.go:20)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-041`. Evidence: pending.

**Protects**

- Target/helper/consumer fixture rows verify across two Can files
- production erases fixture declarations and live run uses real function
- wrong exact target fails staged build
- strict emission

**Current observations:** Can assert JSON 13 roots, production artifact and run output, build diagnostic and optional tsc

**Variants:** main/helpers wrong target mutation

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional with CAN_BUN

**Capabilities:** CAN PROC FILES WORK DIAG BUILD SUITE FAULT

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Target/helper/consumer fixture rows verify across two Can files. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/templates/main.can](/Users/vince/Projects/can-lang/compiler/testdata/current/templates/main.can); [compiler/testdata/current/templates/helpers.can](/Users/vince/Projects/can-lang/compiler/testdata/current/templates/helpers.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-042

**TestCurrentBundledText** — [tests/integration/text_test.go](/Users/vince/Projects/can-lang/tests/integration/text_test.go:16)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-042`. Evidence: pending.

**Protects**

- Native Unicode/code-unit/grapheme, replacement/split, scalar validity and regex matching caps/offsets
- Can fixture assertions and silent production run

**Current observations:** Staged runtime/text.test.ts Can assert JSON and run output optional strict tsc

**Variants:** main 45 roots malformed surrogate/scalar, bad regex, zero-width matches

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional with CAN_BUN

**Capabilities:** CAN PROC FILES WORK NATIVE BUILD SUITE

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Native Unicode/code-unit/grapheme, replacement/split, scalar validity and regex matching caps/offsets. Native infrastructure may supply raw process, file, or host observations only. Every related runtime TypeScript test scenario and pass/fail oracle must be reauthored in Can; for host-only malformed values, a generic native probe may construct Can-selected inputs and return raw observations.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [runtime/test/text.test.ts](/Users/vince/Projects/can-lang/runtime/test/text.test.ts); [std/text/current/src/main.can](/Users/vince/Projects/can-lang/std/text/current/src/main.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-043

**TestCurrentBundledTLS** — [tests/integration/tls_test.go](/Users/vince/Projects/can-lang/tests/integration/tls_test.go:17)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-043`. Evidence: pending.

**Protects**

- Compiled Can TLS server with fresh local self-signed chain serves expected health response and rejects untrusted fetch
- production source and strict emission maintain TLS lifecycle

**Current observations:** Can assert JSON, openssl certificate generation, independent HTTPS client and failure observation, artifact inspection/optional tsc

**Variants:** healthy trusted request untrusted request cert/chain lifecycle

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec openssl CAN_TSC optional

**Capabilities:** CAN PROC FILES WORK HTTP CHILD NATIVE BUILD SUITE

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Compiled Can TLS server with fresh local self-signed chain serves expected health response and rejects untrusted fetch. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/http/tls.can](/Users/vince/Projects/can-lang/compiler/testdata/current/http/tls.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-044

**TestBundledTransportLoopback** — [tests/integration/transport_test.go](/Users/vince/Projects/can-lang/tests/integration/transport_test.go:13)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-044`. Evidence: pending.

**Protects**

- Request preparation validates URLs, auth and headers before launch while preserving duplicate cookies and normalizing fields
- Raw and named fetch observe exact bodies, media/charset, redirects, timeout/cancellation phases, byte caps and nominal transport errors
- Owned continuations drain and retain resources; late decoder/transport faults preserve chosen timeout and report diagnostics correctly

**Current observations:** Seven staged transport runtime suites currently contain case actions and assertions; Go checks zero fail and named evidence strings

**Variants:** body/request/owned/fetch/http/named/late suites; one-attempt, stalled body, redirect, malformed reply, late fault

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec localhost bind/outbound

**Capabilities:** CAN PROC FILES WORK WIRE HTTP CHILD NATIVE FAULT CLOCK SUITE

**Proposed Can replacement:** Can-authored ordinary cases configure request/peer faults, issue operations, and compare raw transport observations, timing phase and owner trace; generic native network/clock/ownership mechanics may supply events. Migrate delegated runtime oracles individually.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [runtime/test/transport-body.test.ts](/Users/vince/Projects/can-lang/runtime/test/transport-body.test.ts); [runtime/test/transport-request.test.ts](/Users/vince/Projects/can-lang/runtime/test/transport-request.test.ts); [runtime/test/transport-owned.test.ts](/Users/vince/Projects/can-lang/runtime/test/transport-owned.test.ts); [runtime/test/transport-fetch.test.ts](/Users/vince/Projects/can-lang/runtime/test/transport-fetch.test.ts); [runtime/test/transport-http.test.ts](/Users/vince/Projects/can-lang/runtime/test/transport-http.test.ts); [runtime/test/transport-named.test.ts](/Users/vince/Projects/can-lang/runtime/test/transport-named.test.ts); [runtime/test/transport-late.test.ts](/Users/vince/Projects/can-lang/runtime/test/transport-late.test.ts)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-045

**TestCurrentBundledUtilities** — [tests/integration/utilities_test.go](/Users/vince/Projects/can-lang/tests/integration/utilities_test.go:15)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-045`. Evidence: pending.

**Protects**

- Native URL normalization/query and datetime zone/locale/civil edge cases
- compiled Can utilities conversion/bounds and silent production run
- strict emission

**Current observations:** Staged runtime/test/url.test.ts and datetime.test.ts Can assert JSON/run output and optional tsc

**Variants:** URL 5 cases datetime 7 cases Can utility from/to conversion table

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec pinned ICU CAN_TSC optional

**Capabilities:** CAN PROC FILES WORK NATIVE DIAG BUILD SUITE

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Native URL normalization/query and datetime zone/locale/civil edge cases. Native infrastructure may supply raw process, file, or host observations only. Every related runtime TypeScript test scenario and pass/fail oracle must be reauthored in Can; for host-only malformed values, a generic native probe may construct Can-selected inputs and return raw observations.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [runtime/test/url.test.ts](/Users/vince/Projects/can-lang/runtime/test/url.test.ts); [runtime/test/datetime.test.ts](/Users/vince/Projects/can-lang/runtime/test/datetime.test.ts); [compiler/testdata/current/cli/utilities.can](/Users/vince/Projects/can-lang/compiler/testdata/current/cli/utilities.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-046

**TestCurrentBundledVerifiedBuild** — [tests/integration/verified_build_test.go](/Users/vince/Projects/can-lang/tests/integration/verified_build_test.go:22)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-046`. Evidence: pending.

**Protects**

- Passing complete build binds source/dependency/catalogue/compiler/runtime/options identities, verifies all roots and publishes production without assertion runner
- failed, timed-out, uncovered native and sticky fixture builds never publish
- selected assert is partial and never publishes
- prior selected generation survives failed rebuild
- locked dependency roots gate publication
- offline build makes zero live HTTP requests and published run makes one

**Current observations:** Build/assert JSON, current/manifest hash, production artifact inspection, independent HTTP count, run status and negative diagnostics

**Variants:** passing, wrong assertion, failed rebuild, selected assert, timeout, uncovered native, unused fixture, locked vendor mutation, live fetch

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec

**Capabilities:** CAN PROC FILES WORK BUILD HTTP NATIVE DIAG SUITE FAULT PEER

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Passing complete build binds source/dependency/catalogue/compiler/runtime/options identities, verifies all roots and publishes production without assertion runner. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects](/Users/vince/Projects/can-lang/docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects)

**Notes:** The frozen projects under docs/syntax-taste/evidence are current test inputs here, not proof that this live build/negative qualification has already passed. Proposed migration only.

## CORE-047

**TestCurrentBundledWebSocket** — [tests/integration/websocket_test.go](/Users/vince/Projects/can-lang/tests/integration/websocket_test.go:17)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-047`. Evidence: pending.

**Protects**

- 13 Can assertion roots include supplied-completion and real-can evidence
- Compiled boot/link/push/push_raw/drain/bye/release sequence preserves text and binary frames, close code/reason, final empty drain, stop/wait and clean owner result

**Current observations:** Can assert JSON and build/run plus optional strict tsc Current embedded TypeScript harness owns websocket sequence and expected values; this oracle must move to Can

**Variants:** text hi, bytes [1,2,3], close 1000 done, empty drain

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_TSC optional

**Capabilities:** CAN PROC FILES WORK WIRE CHILD BUILD SUITE

**Proposed Can replacement:** Can-owned ordinary scenario drives socket lifecycle and compares protocol, send counts, text/binary frame contents, close event and owner cleanup. Native websocket mechanics expose events and bytes without fixed case assertions.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/ws/socket.can](/Users/vince/Projects/can-lang/compiler/testdata/current/ws/socket.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-048

**TestCurrentBundledWrappers** — [tests/integration/wrap_test.go](/Users/vince/Projects/can-lang/tests/integration/wrap_test.go:20)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-CORE-048`. Evidence: pending.

**Protects**

- Nested wrapper/recovery policies preserve selected fixture rows and error identity
- 404 and 429/503 recovery paths, successful load bypass, and wrong emitted local key fail verification
- strict emission

**Current observations:** Can assert JSON 13 roots, independent HTTP server modes/counts, run status and optional tsc

**Variants:** 404, 429 busy, 503 broken, success, missing local key

**Environment/selection gates:** CAN_BUN_ARCHIVE macOS sandbox-exec CAN_I17_TOKEN supplied CAN_TSC optional

**Capabilities:** CAN PROC FILES WORK HTTP NATIVE DIAG BUILD SUITE FAULT PEER

**Proposed Can replacement:** Can-authored normal functions select these scenarios, perform the actions, and compare returned observations against Can expectations: Nested wrapper/recovery policies preserve selected fixture rows and error identity. Native infrastructure may supply raw process, file, or host observations only.

**Old harness may be deleted when:** Delete this Go check only after identified Can cases and helpers reproduce every protected behavior, observation, variant, negative control and required environment in this row on the production candidate; record independent qualification evidence and rewire active callers first.

**Related source:** [compiler/testdata/current/wrap/main.can](/Users/vince/Projects/can-lang/compiler/testdata/current/wrap/main.can)

**Notes:** Proposed migration only; no backend selected or qualification run.

## CORE-049

**stageRawFixtures** — [tests/integration/raw_fixtures_test.go](/Users/vince/Projects/can-lang/tests/integration/raw_fixtures_test.go:16)

Kind: `support`. Proposed disposition: `replace_support`. Replacement obligation: `CAN-CORE-049`. Evidence: pending.

**Protects**

- Raw fixture staging rewrites only selected origin/endpoint references while preserving exact request/response bytes and required fixture set; static fixture files remain inputs rather than scenarios

**Current observations:** Current Go helper copies and rewrites compiler/testdata/current/<area>/fixtures before Can assert/build

**Variants:** fetch and other current raw-fixture areas; URL relocation

**Environment/selection gates:** None recorded.

**Capabilities:** FILES NATIVE SUITE

**Proposed Can replacement:** Can scenario chooses fixture inputs and expected raw exchange; generic native file copy/read may stage bytes and report observations without choosing cases or judging matches.

**Old harness may be deleted when:** Remove this helper after all calling Go scenarios are migrated and Can-owned cases demonstrate exact raw-exchange comparisons, fixture provenance and endpoint relocation without host-authored case policy.

**Related source:** [compiler/testdata/current/fetch/fixtures](/Users/vince/Projects/can-lang/compiler/testdata/current/fetch/fixtures); [compiler/testdata/current/native/fixtures](/Users/vince/Projects/can-lang/compiler/testdata/current/native/fixtures); [compiler/testdata/current/templates/fixtures](/Users/vince/Projects/can-lang/compiler/testdata/current/templates/fixtures); [compiler/testdata/current/wrap/fixtures](/Users/vince/Projects/can-lang/compiler/testdata/current/wrap/fixtures); [tests/integration/fetch_test.go](/Users/vince/Projects/can-lang/tests/integration/fetch_test.go); [tests/integration/templates_test.go](/Users/vince/Projects/can-lang/tests/integration/templates_test.go); [tests/integration/judge_test.go](/Users/vince/Projects/can-lang/tests/integration/judge_test.go); [tests/integration/native_test.go](/Users/vince/Projects/can-lang/tests/integration/native_test.go); [tests/integration/questions_test.go](/Users/vince/Projects/can-lang/tests/integration/questions_test.go); [tests/integration/generation_test.go](/Users/vince/Projects/can-lang/tests/integration/generation_test.go); [tests/integration/wrap_test.go](/Users/vince/Projects/can-lang/tests/integration/wrap_test.go)

**Notes:** Proposed support migration; static fixture files can remain when used as inputs.

## CORE-050

**stripStagedAuthorization** — [tests/integration/raw_fixtures_test.go](/Users/vince/Projects/can-lang/tests/integration/raw_fixtures_test.go:44)

Kind: `support`. Proposed disposition: `replace_support`. Replacement obligation: `CAN-CORE-050`. Evidence: pending.

**Protects**

- Removes staged authorization request-header pairs from raw fixtures for direct conformance probes; preserves other headers and refuses malformed fixture structure

**Current observations:** Current helper rewrites staged can.native-fixture.v1 JSON inputs for direct native conformance probes

**Variants:** judge, questions and generation direct raw-provider probes

**Environment/selection gates:** None recorded.

**Capabilities:** FILES NATIVE SUITE

**Proposed Can replacement:** Can scenario selects fixture variant and its expected raw exchange; native fixture copier may apply Can-selected environment/header transformation without owning scenario or verdict.

**Old harness may be deleted when:** Remove after all listed callers are migrated and Can cases demonstrate raw-provider behavior with transformed fixtures and no hidden host policy.

**Related source:** [tests/integration/judge_test.go](/Users/vince/Projects/can-lang/tests/integration/judge_test.go); [tests/integration/questions_test.go](/Users/vince/Projects/can-lang/tests/integration/questions_test.go); [tests/integration/generation_test.go](/Users/vince/Projects/can-lang/tests/integration/generation_test.go)

**Notes:** Proposed support replacement; static raw fixtures may remain as input.

## CORE-051

**clearStagedFixtureEnvironments** — [tests/integration/raw_fixtures_test.go](/Users/vince/Projects/can-lang/tests/integration/raw_fixtures_test.go:102)

Kind: `support`. Proposed disposition: `replace_support`. Replacement obligation: `CAN-CORE-051`. Evidence: pending.

**Protects**

- Empties per-fixture environment snapshots for direct conformance probes while retaining exchange body and request comparisons

**Current observations:** Current helper rewrites staged can.native-fixture.v1 JSON inputs for direct native conformance probes

**Variants:** judge, questions and generation direct raw-provider probes

**Environment/selection gates:** None recorded.

**Capabilities:** FILES NATIVE SUITE

**Proposed Can replacement:** Can scenario selects fixture variant and its expected raw exchange; native fixture copier may apply Can-selected environment/header transformation without owning scenario or verdict.

**Old harness may be deleted when:** Remove after all listed callers are migrated and Can cases demonstrate raw-provider behavior with transformed fixtures and no hidden host policy.

**Related source:** [tests/integration/judge_test.go](/Users/vince/Projects/can-lang/tests/integration/judge_test.go); [tests/integration/questions_test.go](/Users/vince/Projects/can-lang/tests/integration/questions_test.go); [tests/integration/generation_test.go](/Users/vince/Projects/can-lang/tests/integration/generation_test.go)

**Notes:** Proposed support replacement; static raw fixtures may remain as input.
