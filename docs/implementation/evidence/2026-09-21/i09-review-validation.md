# I09 independent review corrections

The monitoring review identified three reproducible implementation gaps in
`fd3daee`. This correction addresses all three before beginning I10.

- Recheck the project/source/dependency/asset snapshot immediately before current
  publication, after staging or reuse, and recheck the owned output layout.
  Regression hooks mutate source after staging, after generation rename and just
  before publication; the previous current pointer stays byte-identical. The last
  check covers reused generations too. This does not promise atomic filesystem
  snapshots against arbitrary edits after the final check.
- Reserve the root metadata file `manifest.json`, case aliases and every path
  below it before output construction. Both preparation and decoded-manifest
  validation use the shared path rule, so these collisions cannot enter staging.
- Parse Bun-transformed JavaScript with vendored upstream Acorn 8.18.0 and reject
  dynamic `ImportExpression` nodes, including computed/template arguments and
  nested dormant functions. Check static import/re-export sources against the
  inventory as well. Bun retains syntax and non-ESM scan checks. This repairs an
  internal validation guarantee; no authored-Can escape or arbitrary-JS sandbox
  claim is made.

`i09-review-before.txt` records the initial filesystem regression failures.
The `before-current` hook was introduced with the fix; the earlier `staged` and
`generation` hooks independently reproduce the preexisting stale publication.
The monitor's packaged computed-import reproduction is corroborated by the
new packaged negative cases, which must now fail validation and publication.

Validation passed:

```sh
GOCACHE=/tmp/can-i02-go-cache go test -count=1 ./compiler/internal/driver
/Users/vince/.bun/bin/bun test tools/runtime/output-graph.test.ts
CAN_BUN=/Users/vince/.bun/bin/bun GOCACHE=/tmp/can-i02-go-cache go test -count=1 ./...
CAN_BUN_ARCHIVE=/tmp/can-i01-bun.zip GOCACHE=/tmp/can-i02-go-cache go test -count=1 -v ./compiler/internal/driver ./tests/integration -run '^(TestPackagedOutputValidationAndExecution|TestDevelopmentSidecar)$'
git diff --check
```

Corresponding raw outputs are `i09-review-driver-tests.txt`,
`i09-review-graph-tests.txt` (2 tests, 14 expectations),
`i09-review-go-tests.txt` and `i09-review-offline-tests.txt`. The packaged test
runs under macOS network denial and exercises successful generated execution,
private runtime sharing and child-held generation leases after parser validation.

The parser archive's npm SHA-512 integrity was verified before extraction.
`tools/runtime/vendor/acorn.lock.json` records upstream revision and archive,
module and license hashes. Distribution tests check the vendored files against
that lock; the sidecar's full manifest pins them for offline invocation. The MIT
license is retained under distribution notices. The parser implementation is
unmodified upstream code. Three fresh design consultations, wording checks and
results are retained in `i09-import-jev/`; agreement was treated as advice.
