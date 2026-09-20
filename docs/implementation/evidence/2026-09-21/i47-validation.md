# I47 validation

Authoritative inventory: compiler/internal/catalogue/catalogue.json, schema 1,
revision 1, target bun-1.4.2-darwin-arm64-v1. Inventory coverage was reconciled
against C6–C9, A2, P2 and P6–P12: 21 reserved packages, 34 catalogue data/opaque
types, 51 allocated domain errors, 144 operations/properties, and 10 native
execution modes. The prelude contains append, choice_option, all_failed and
standard_failure. Primitive operators remain C6 compiler expressions, not an
extra source package or extensible host-operation registry.

Passed in this checkout:

- `go test -count=1 ./...` with GOCACHE=/tmp/can-i02-go-cache: all packages passed;
  archive-dependent integration tests skip in this ordinary invocation.
- `go run ./compiler/internal/catalogue/cmd/cataloguegen --check`: all generated
  Go, TS, registry and documentation mirrors match the checked inventory.
- `bun tscheck/node_modules/typescript/bin/tsc --strict --noEmit --target ESNext --module ESNext runtime/catalogue.ts`:
  strict checking of the generated mirror passed using the existing pinned tsc.
- `bun test runtime/test/catalogue.test.ts`: passed all nine runtime catalogue
  invariants through the shared check function on installed Bun 1.4.2.
- Explicit archive-backed `go test -count=1 -v ./compiler/internal/catalogue`:
  positive, negative, generic/conditional-bound, mirror-drift and packaged
  boundary tests passed. The exact output is [i47-catalogue-tests.txt](i47-catalogue-tests.txt).
  The integration uses the hash-verified staged Bun, network-denying macOS
  sandbox, unrelated cwd and PATH without Bun. Error 1110's identity, payload
  and occurrence are identical on the Go and generated TypeScript sides.

The domain occurrence is a supplied integration fixture. This proves catalogue
agreement, not production failure handling (I49), new-language compilation, or
implementation of every native adapter described by the catalogue. Recipes
retain their owning implementation-task IDs for those later gates.
