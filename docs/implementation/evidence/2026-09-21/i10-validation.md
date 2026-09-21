# I10 validation

Completed 2026-09-21. [Completion protocol](../../completions.md) documents the
checker, region IR, native emission, protected carriers and integration interfaces.

| Acceptance requirement | Implemented evidence |
| --- | --- |
| Generated async functions return private nonthenable carriers | `RegionEmitter.Function` returns `Completion<T>` for success, constructed/forwarded domain failure and standard failure; `runtime/completion.test.ts` checks private brands, fixed data properties, freezing, null prototypes and absence of `then` |
| Distinct value and terminal regions | `CheckRegion` tags terminals with the function/handler ID; value matches have ordinary typed results, and do/call/chain/relay retain their owner; `TestRegionTypesAndOwnership` rejects a foreign region return |
| Nested regions and void completions compose | The contract suite checks the same positive bodies as function and handler regions; generated execution covers nested ordinary/call matches, do, relay and bare void success; runtime dispatch tests nest a local handler then continue the outer handler |
| Missing/extraneous/duplicate arms fail | Exact success/domain/optional-standard coverage, incompatible bindings and ambiguous generic bare errors reject in `TestCompletionRegionContracts` |
| Nonterminal misuse fails | Value-match completion arms, fallible ordinary operands, missing/nonvoid `ok`, nonvoid call steps, missing terminals and inaccessible chain-success bindings reject |
| Handler failure never redispatches | Generated domain and standard handler-failure traces escape the selected handler; one-shot runtime dispatch preserves the fresh occurrence; actual handler escapes exclude handled invocation errors |
| Callable `then` data survives | Native ordinary calls, callbacks, element mapping and all four Promise bridges preserve the payload without invoking its field; generated relay returns the same receipt object |
| Evaluation and pattern contracts remain native | Generated tests cover argument faults before launch, once-only literal spreads, trailing variadic packing, short-circuit async operands, method-chain union/stop behavior, constructor products, ranges, alternatives, arrays/rest and recursive data |
| Packaged integration | Absolute staged Bun executes generated regions and carrier tests with network denied; existing output publication/shared-runtime/lease and sidecar integrity checks also pass |

Passed commands and raw output:

```sh
CAN_BUN=/Users/vince/.bun/bin/bun GOCACHE=/tmp/can-i02-go-cache go test -count=1 ./...
# i10-go-tests.txt
CAN_REGION_TEST_OUTPUT=/tmp/can-i10-regions.ts CAN_BUN=/Users/vince/.bun/bin/bun GOCACHE=/tmp/can-i02-go-cache go test -count=1 -v ./compiler/internal/emit -run 'CompletionRegion|RegionTypes'
# i10-region-tests.txt
/Users/vince/.bun/bin/bun test runtime tests/conformance tools/runtime/output-graph.test.ts
# i10-runtime-tests.txt: 35 pass, 0 fail, 1,133 expectations
CAN_BUN=/Users/vince/.bun/bin/bun CAN_BUN_ARCHIVE=/tmp/can-i01-bun.zip GOCACHE=/tmp/can-i02-go-cache go test -count=1 -v ./compiler/internal/emit ./compiler/internal/driver ./tests/integration -run '^(TestEmittedCompletionRegions|TestPackagedOutputValidationAndExecution|TestDevelopmentSidecar)$'
# i10-offline-integration.txt
```

`i10-typescript-tests.txt` records the successful strict TypeScript 7.0.2 check of
the generated integration program and imported completion runtime, exact installed
host-definition versions and flags. Dependencies were installed in a temporary
validation directory with scripts disabled. Dependency declaration checking was
skipped; this is not a claim that the complete I45 release gate is implemented.
The generated file can be reproduced with the `CAN_REGION_TEST_OUTPUT` command.

Three fresh carrier-design consultations and raw results are in `i10-jev/`.
Their agreement was treated as advice; the executable tests establish behavior.
Native target capability evidence remains separate in the I01/I09 reports.

Current CLI routing is I11. Captured callable instances, native state groups,
assertion fixtures, concrete generic body specialization and the full coordination
scheduler retain their respective later tasks. Their owning passes supply sealed
contracts to the region interfaces; unsupported constructs fail explicitly and
never use the predecessor interpreter/emitter.
