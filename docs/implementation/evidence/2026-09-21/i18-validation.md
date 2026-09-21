# I18 validation report v1

I18 completes deterministic authored fixture allocation and boundary-specific evidence reporting. Compiler checks, generated Can execution, runtime conformance and negative cases below passed with qualified Bun 1.4.2, revision `744846f844374847c902b5e7fd59b4342a51ef99`.

## Implementation

Checked call, method, callable-reference and coordination sites use resolved declaration identity plus syntax preorder ordinal. Source spans serve only as checker lookup keys. Fixture table identity derives from its checked call site. Formatting changes and unrelated checker-local allocation do not alter sites.

Generated calls carry immutable context views. Each view owns an invocation identity and an explicit execution frame; shared root state owns lexical FIFO queues, violations and evidence. Calls suspend their parent while running a child and resume it atomically when the child finishes. Coordination reserves every written/spread position before launch. A participant frame remains active until the native selection adapter observes completion. When that completion can settle the aggregate, an active selection continuation covers the native Promise reaction gap until the caller resumes. Native all/allSettled/any/race still select the outcome.

The barrier releases only the least pending full invocation path when all active frames are waiting or finished. It repeats this check after each release; it does not infer quiescence from a tick count. Allocation consumes exactly the next selected lexical row, then validates arguments without searching. The root execution frame finishes before owned loser drain; leftover checks happen after drain. Raw provider fixture registration follows shared root ownership across child context views.

Native closures receive creation receipts, including private frozen receiver/near capture aliases. Dynamic receipts retain root and parent creation path and reject cross-root migration. Pure initialization receipts have a separate immutable root-neutral identity and can be reused in isolated assertion paths without sharing state. Diagnostic fingerprints hash creation coordinates instead of exposing captured values.

A deterministic native slice carries the same checked argument/result contract and fixture boundary as an ordinary call. Preparation occurs once and the real native operation runs only when no selected row replaces it. Chain fixtures remain inside named wrappers, consistent with the existing single-invocation fixture grammar.

## Requirement evidence

| Requirement | Authoritative coverage |
| --- | --- |
| Stable lexical identity, parent occurrences and no offset-based identity | `TestLexicalSitesUseSyntaxPreorder`, `TestFixtureSiteIgnoresCheckerLocalAllocation`, `assert-identity.test.ts` |
| Exact native single-call admission, including `"abc".slice(1, 3)` | `TestNativeSliceFixturesUseCheckedInvocationContract`, staged `native-slice.can`; selected fake result, unselected real `"bc"`, wrong argument and static type/arity/result rejection |
| Repeated identical and unequal arguments use ordered rows | Staged `queues.can` roots `identical_arguments` and `repeated`; runtime repeated same-site occurrence tests |
| Recursive, mutually recursive and transitive shared helper queues | Staged `recursion_root`, `mutual_root`, `repeated`; runtime queue/execution tests |
| Same creation site, different callable values and frozen near/receiver captures | `callable.test.ts`, staged `captured_pair`, `receivers` and `direct_receivers`; negative captured-call test requires creation site/instance in the emitted full path |
| Direct/spread reservation, nested groups and concurrent shared helpers | Identity/barrier/execution tests; staged `spread`, `concurrent`, `settled`, `any`, `race` |
| Reversed host arrival preserves canonical assignment | Controlled real Promise gates delay the first participant past the second request in all four modes; `assert-execution.test.ts` requires canonical row order and early native selection before late fixture delivery |
| Arbitrarily delayed pure descendants and native reaction gaps | Barrier tests keep descendants or a selection continuation active across multiple microtasks; integrated mode tests verify no early fixture release |
| First-row mismatch never searches later rows | Runtime execution test plus staged argument mutation; reports contain expected table/row/path and actual full path |
| Exhaustion and post-drain leftovers | Staged removal of the second row fails at row 1; adding a third row to an early race fails with unused row 2, proving its loser consumed row 1 before closure |
| Ambiguous claims, malformed/foreign tokens and invalid transitions | Queue and barrier tests reject duplicate claims, forged identities/frames/queues, cross-root tokens, malformed completions and invalid participant ordering |
| Alternate schedules remain conformance-only | Barrier reverse-comparator test; malformed comparator rejects all requests without stranded frames; generated calls expose no scheduling syntax |
| External access fails closed with sticky violations | Existing staged output/native-boundary tests and raw-provider tests; every harness violation now records its actual invocation path |
| Opaque and callable identity cannot be replaced by structural equality | Existing assertion equality tests preserve compiler-owned identity and reject distinct opaque handles; statically malformed fixture results fail checker admission |
| Five evidence categories remain separate | `report.ts` and its tests distinguish real-can, supplied-completion, raw-provider-fixture, bun-conformance and live-quality; ordinary assertion jobs reject the latter two labels |

## Validation

- `go test ./compiler/... ./tests/integration -count=1` passes with `CAN_BUN`, `CAN_BUN_ARCHIVE`, `CAN_TSC` and the isolated Go cache configured. This includes development-sidecar integrity, current compiler behavior, strict generated TypeScript and offline staged execution.
- `bun test runtime/test`: **136 tests, 961 expectations, zero failures** across 27 files.
- Strict TypeScript checking passes for all new assertion modules, changed context/fixture/provider/runner/callable/coordination modules and their new tests.
- `TestCurrentBundledAssertions` runs the new queue fixture's **22 assertion roots**, native slice cases and negative mutations. The release is staged, Bun is invoked through the absolute qualified layout, network access is denied, `PATH=/nonexistent`, and the working directory is unrelated to the source/release.
- Existing staged callable and coordination suites also pass after context and receipt integration.
- `git diff --check` passes.

The ordinary Can rows establish `real-can` and `supplied-completion` evidence. Raw provider tests separately exercise request encoding/response parsing. Controlled pinned-Bun Promise and staged-release tests establish conformance evidence. No live external-quality result is claimed.

Design consultations: [explicit frames](i18-jev/README.md) and [callable creation](i18-callable-jev/README.md). All six requests/responses and both wording audits are retained. Agreement was checked against implementation and tests rather than treated as proof.
