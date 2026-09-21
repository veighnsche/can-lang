# I12 — basic mandatory assertions execute emitted code

Mandatory concrete assertion rows now produce checked actual-invocation and
expected-completion regions. Inputs, receivers, variadics, success values and
domain bounds use the current checker. The driver emits private root modules,
publishes through the existing validation boundary, and invokes the absolute
bundled Bun. Library assertion projects do not require `main`.

- **P+:** the staged arithmetic fixture executes real native bigint multiplication,
  including a value beyond 2^53. Runtime tests hold an actual body behind a promise
  and count precisely one invocation. A deliberately wrong numeric expectation
  fails rather than returning a scripted success. Root identity contains package,
  declaration and assertion name; short selectors require uniqueness in their
  package and full selectors disambiguate collisions.
- **N−:** missing/malformed rows, duplicate assertion names, wrong arguments,
  wrong expected success types and malformed supplied completions fail statically.
  Checking covers all concrete roots before selection/publication. The staged
  malformed-fixture test leaves the previous current generation unchanged.
  A live stdout call without a fixture is refused before writing; even a Can
  standard catch cannot erase its sticky harness violation. Expected evaluation
  failure prevents subject execution. Initialization failure fails the suite.
  A closed report pipe returns nonzero without a native stack.
- **INT:** a new offline release-layout test invokes `assert` from outside the
  project with `PATH=/nonexistent` and networking denied. Pure arithmetic remains
  real; a basic `when` replaces one invocation while its continuation runs.
  Supplied I/O prevents the application text from reaching stdout. Both expected
  and supplied nominal domain failures execute correctly. Reports distinguish
  `real-can` and `supplied-completion`, without provider/runtime-quality claims.

The private context is an explicit hidden function argument, also preserved in
callable ABI types and maintained adapters. Native strict deep equality owns data
comparison, preceded by a memoized opaque-identity guard so empty opaque tokens
cannot compare equal accidentally. Separate roots remain isolated across awaits;
reports serialize no application values, native messages or stacks.

Evidence:

1. `CAN_BUN=/Users/vince/.bun/bin/bun GOCACHE=/tmp/can-i02-go-cache go test ./...`
   — [full Go suite](i12-go-tests.txt).
2. `/Users/vince/.bun/bin/bun test runtime` — 36 tests and 1,179 expectations;
   [native runtime results](i12-runtime-tests.txt).
3. TypeScript 7.0.2 strict check of newly emitted assertion/fixture modules and
   maintained runtime/tests — [strict results](i12-typescript-tests.txt). Options:
   `--noEmit --strict --skipLibCheck --target esnext --module esnext
   --moduleResolution bundler --allowImportingTsExtensions --types bun,node`.
4. `CAN_BUN_ARCHIVE=/private/tmp/can-i01-bun.zip
   CAN_BUN=/Users/vince/.bun/bin/bun GOCACHE=/tmp/can-i02-go-cache go test
   ./tests/integration ./compiler/internal/driver ./compiler/internal/emit
   -count=1` — [staged offline suites](i12-offline-tests.txt), including production
   CLI regression coverage. The final report-pipe correction was then checked by
   the [focused offline assertion suite](i12-assertion-tests.txt).
5. [Actual emitted assertion report](i12-assertion-report.json), generated from
   `compiler/testdata/current/assertions/basic.can` in the staged bundle.
6. [Three fresh Jev consultations and decision rationale](i12-jev/README.md).

[Usage and current boundaries](../../assertions.md) describe the single-invocation
fixture slice. I18 remains unchecked: typed-AST invocation paths, captured
callable identities, coordination barriers and full fixture-token admission are
not claimed here. I20 still owns late-work/resource draining; I46 owns concrete
generic assertion instances. No legacy evaluator or live-provider fallback is
used by this command.
