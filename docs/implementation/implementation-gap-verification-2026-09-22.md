# Targeted implementation-gap verification — 2026-09-22

Status: executable verification of the current implementation, not a language redesign or a compiler fix. A fresh distribution was built from the working checkout at commit `760405f537bb8a2fb6d7f277ba27888a7e3f5dc2`, using the locally available pinned Bun 1.4.2 archive. Compiler/runtime/distribution sources had no working-tree changes. All project output was confined to temporary project copies. [Inputs, hashes, scripts and raw results](../syntax-taste/evidence/2026-09-22/implementation-gaps/README.md) are saved.

## Findings

| ID | Observation | Classification | Consequence for scope |
| --- | --- | --- | --- |
| G1 | `build` publishes despite an assertion whose expected value is wrong; `assert` correctly fails it | **Confirmed missing build guarantee and inaccurate README claim**, not broken assertion comparison | LD22 remains implement; the B5 verified-build contract is not current behavior |
| G2 | `assert` switches `dist/current.json` to assertion output before an assertion fails or hangs | **Confirmed publication coupling** in the existing driver, not evidence of corrupt/partial generations | B5's separation of test staging from production publication has an executable reproducer |
| G3 | An empty-race assertion remains pending until the external test limit kills it | **Confirmed assertion-runner liveness gap**; pending production empty-race behavior is intentional | LD23 remains implement; LD24 retains production behavior |
| G4 | Semantic/name diagnostics at source line 8 highlight line 1 in LSP; lexical errors highlight line 8 correctly | **Confirmed semantic diagnostic quality limitation**, not a universal source-map failure | LD44 should carry semantic spans through the bridge; keep working lexical/runtime locations |
| G5 | Resource escape shapes compile; runtime rejects use outside the scope before native work | **Intentional runtime-enforced limitation plus absent optional early diagnostics**; no safety bypass demonstrated | LD30 retains runtime ownership; LD31 remains deferred pending a justified, bounded static-analysis design |

These categories distinguish the current implementation from the newly selected [behavior contracts](language-behavior-contracts-2026-09-22.md). A missing behavior promised by the new design is not automatically evidence that an old runtime contract was violated.

## G1–G2. Build/assertion behavior and publication

The [failing-assertion project](../syntax-taste/evidence/2026-09-22/implementation-gaps/projects/failing-assertion/src/main.can) contains a valid `main` and this deliberately wrong but type-correct assertion:

```can
fn int answer
    emits []
    asserts
        wrong: => ok 2
    ok 1
```

Observed with the fresh bundled launcher:

| Command | Exit | Result |
| --- | --- | --- |
| `build` on passing control | 0 | Production generation published |
| `assert` on passing control | 0 | All assertions pass |
| `build` on wrong assertion | 0 | Production generation published; no assertion failure reported |
| `assert` on wrong assertion | 1 | `can.assertion-report`, `passed: false`, `reason: "outcome mismatch"` for `answer/wrong`; `main/empty` passes |

Thus the assertion engine detects the mismatch when invoked. The missing step is on the build path. The README's statement that emitted assertions run as part of every build was false for this checkout; it has been corrected to describe current behavior and link the planned contract.

The same raw result records save `dist/current.json` immediately before and after assertion execution. It changes for passing, failing and externally stopped assertion runs. Source inspection confirms why: [build](../../compiler/internal/driver/commands.go) checks and calls `publishProgram(..., false)`; [assert](../../compiler/internal/driver/assert.go) calls `publishProgram(..., true)` before executing the selected output. The code uses the same current-selection store for both.

This does **not** show partial output or loss of old immutable generations. It shows that test output becomes current before test success. B5's “failed verification leaves production current unchanged” needs a publication-policy change, not a claim that existing atomic rename/lease handling is broken. The existing CLI/assertion guides already describe separate execution more accurately than the old README sentence.

## G3. Hanging assertions

The [pending project](../syntax-taste/evidence/2026-09-22/implementation-gaps/projects/pending-assertion/src/main.can) uses current syntax:

```can
fn int pending
    emits []
    asserts
        never: => ok 0
    callable int () emits [][] actions = []
    int result = match call race with error
        ...actions
        ok int value => ok value
    ok result
```

Its `build` exits 0 in about 0.65 seconds. Its `assert` produces no stdout/stderr report and remains alive until the **external Python supervisor** kills its process group after 4 seconds (observed 4.01 seconds). Exit `-9` in the saved result is the probe's termination, not a compiler timeout or successful assertion handling.

A finite observation does not prove a process will never exit. Here it combines with the selected pending semantics of empty `race with error`, and inspection of the [assert runner](../../runtime/assert/runner.ts) and driver execution path showing no assertion deadline. This establishes a reproducer for the liveness gap without waiting indefinitely. The original wording “one empty race hangs the build” was inaccurate: today's build does not execute it; assertion execution hangs. Once build is assertion-gated, the deadline must accompany that guarantee.

A separate [recursive stress probe](../syntax-taste/evidence/2026-09-22/implementation-gaps/projects/cpu-assertion/src/main.can) is a useful negative result: it exited 1 on its own after about 2.21 seconds and reported an outcome mismatch with call frames at line 10. It **does not demonstrate a non-yielding CPU-loop hang**. The safe report does not expose the underlying cause, so this verification does not attribute that failure to a specific engine limit. No extra resource-intensive stress run was needed to establish the empty-race gap.

Retain production Promise/race semantics. Bound assertion execution in the supervisor; do not reinterpret a pending operation as a successful fallback.

## G4. Diagnostic locations

Each project was tested both through `build` and a real stdio LSP `initialize`/`textDocument/didOpen` session. Saved results include the raw messages and source lines.

| Probe | Actual offending source | LSP range (zero-based) | Result |
| --- | --- | --- | --- |
| [Return type mismatch](../syntax-taste/evidence/2026-09-22/implementation-gaps/projects/diagnostic-semantic/src/main.can) | Line 8, `ok "wrong"` in an `int` function | Line 0, columns 0–11 | Highlights `package app`, not the expression |
| [Missing body name](../syntax-taste/evidence/2026-09-22/implementation-gaps/projects/diagnostic-resolve/src/main.can) | Line 8, `ok absent` | Line 0, columns 0–11 | Same incorrect location; this is name lookup during body checking |
| [Invalid lexical token](../syntax-taste/evidence/2026-09-22/implementation-gaps/projects/diagnostic-parse/src/main.can) | Line 8, `@` | Line 7, columns 7–8 | Correct token location; code `CAN-LEX-PUNCTUATION` |
| Passing control | No invalid source | Empty diagnostics | No false positive |

CLI semantic messages retain byte offsets (114 and 116), but the LSP adapter publishes a first-line range and no diagnostic code for these failures. The [bridge](../../compiler/internal/driver/diagnostics.go) explicitly documents and implements this fallback when structured spans are absent. Therefore this is a known representation/quality limitation, not evidence that the parser discarded every location or that the editor invented the wrong range.

The recursive assertion's emitted runtime report has source frames at line 10. That further limits the finding: **semantic editor spans** need repair; runtime and lexical locations must not be broadly described as broken. These probes do not establish location quality for every missing-arm, error-bound or fixture mismatch diagnostic; those remain targeted acceptance cases for LD44.

## G5. Resource escapes: admission versus usable lifetime

### Current-source admission

Six minimal source projects build successfully and their attached assertions pass:

| Shape | Reproducer | What the check establishes |
| --- | --- | --- |
| Direct return | [resource-direct](../syntax-taste/evidence/2026-09-22/implementation-gaps/projects/resource-direct/src/main.can) | A function can return its `sql::transaction` input |
| Record containing handles | [resource-record](../syntax-taste/evidence/2026-09-22/implementation-gaps/projects/resource-record/src/main.can) | A concrete callback body can place the handle in `held.handles` and return `sql::commit<held>` |
| Array of handles | [resource-array](../syntax-taste/evidence/2026-09-22/implementation-gaps/projects/resource-array/src/main.can) | The analogous `sql::commit<sql::transaction[]>` body is accepted |
| Callable capture | [resource-closure](../syntax-taste/evidence/2026-09-22/implementation-gaps/projects/resource-closure/src/main.can) | A callable returned from a function can retain `near sql::transaction tx` |
| Record through transaction signature | [resource-transaction-record](../syntax-taste/evidence/2026-09-22/implementation-gaps/projects/resource-transaction-record/src/main.can) | `sql::with_transaction<held>(pool, callable capture)` accepts that callback/result contract |
| Array through transaction signature | [resource-transaction-array](../syntax-taste/evidence/2026-09-22/implementation-gaps/projects/resource-transaction-array/src/main.can) | `sql::with_transaction<sql::transaction[]>` likewise accepts the concrete contract |

The record/array assertions deliberately exercise the harmless empty branch; the unsafe-looking branch is still in the concrete checked body. Calls to `with_transaction` in those assertion roots use ordinary supplied-completion fixtures, so their passing tests **do not prove runtime escape safety**. They establish full-check/emission acceptance without a live database. The runtime checks below separately exercise the actual scope and transaction adapters.

### Runtime enforcement

Two narrow offline runs total **10 passing tests, 0 failures**:

1. Four new owner-runtime tests return a scoped resource directly, in a record, in an array or through a capture-registered callable. In-scope use succeeds; post-scope use yields `resource_state`; the native-operation counter stays at one. The existing SQL test “the handle dies with its scope” also passes.
2. Four new tests use the **real `createSQLTransactions` adapter with a fake `Bun.SQL` driver**, returning each of those shapes as the committed value. The transaction completes successfully, but every subsequent query through the escaped handle fails with `resource_state`. Each begins exactly once; each fake transaction records **zero native query calls** after escape. The existing successful in-scope SQL query/commit test passes as a control.

Sources and logs: [owner-lifetime tests](../syntax-taste/evidence/2026-09-22/implementation-gaps/resource-lifetime.test.ts), [SQL-return tests](../syntax-taste/evidence/2026-09-22/implementation-gaps/sql-escape.test.ts), [owner log](../syntax-taste/evidence/2026-09-22/implementation-gaps/resource-runtime.txt), [SQL log](../syntax-taste/evidence/2026-09-22/implementation-gaps/sql-runtime.txt).

These are runtime-adapter tests, not a live PostgreSQL test or a single end-to-end emitted Can transaction run. The fake driver makes it possible to prove rejection occurs before native query work. They do not prove all possible alias, asynchronous interleaving, lease or native-driver behaviors.

### Classification and design consequence

[P6](../syntax-taste/platform-testing-spec.md#p6-opaque-values-callbacks-and-resource-enforcement) explicitly assigns no static ownership meaning to returning/recording/capturing handles in this slice and requires rejection on out-of-scope use. The measured behavior agrees. [Runtime scope checking](../../runtime/owner.ts) and [transaction handling](../../runtime/platform/transaction.ts) enforce the use boundary.

The compiler comment that “scope machinery rejects every escape” should not be read as a claim that every return/capture is rejected statically or at return time. The demonstrated guarantee is that the escaped token cannot be used outside its permitted lifetime. That wording is broader than the observed enforcement point.

Keep LD31 **defer with a reason**. Its prerequisite reproducer now exists, but these observations do not justify a general ownership type system or a simple syntactic return ban. Before selecting extra static analysis, demonstrate a useful authoring case, specify which safe in-scope uses must remain admitted, and test a bounded analysis across direct/contained/captured values. The existing runtime safety boundary must remain regardless.

## Verification limits

No compiler/runtime fix, live service call, registry migration, implementation task-list change or new design consultation was performed. The task was empirical verification; Jev preference cannot establish whether a reproducer passes. No full suite was run. Targeted CLI/LSP results, ten runtime tests and code-path inspection establish the findings above, with explicit distinctions between source admission, assertion substitution and native-adapter execution.
