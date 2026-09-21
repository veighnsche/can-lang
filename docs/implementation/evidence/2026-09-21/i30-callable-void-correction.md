# I30 void utility callable correction

Independent review found that first-class references to clock::sleep_millis,
log::write_info and log::write_error generated TypeScript TS2322 failures.
The Can void representation is `undefined`, while those native adapters declared
`Completion<void>`. Direct calls passed because their generated completion path
did not expose the incompatible callable return type.

The three adapters now declare `Completion<undefined>`, matching the value they
already return. The maintained staged utility fixture invokes all three through
first-class callable references, including supplied assertion outcomes and
negative duration cases. Strict generated TypeScript and staged execution pass
(6.366s). The runtime utility suite passes six tests and 85 expectations, and
strict TypeScript passes for its adapters/tests.

The final full compiler/integration gate passed with the pinned Bun archive and
strict generated TypeScript: `go test ./compiler/... ./tests/integration
-count=1` (integration 113.037s).
