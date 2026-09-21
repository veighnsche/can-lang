# I16 validation

The current frontend parses, resolves and checks native declarations without
executing providers or authored declaration bodies.

- [Full Go suite](i16-go-tests.txt) passed with the qualified Bun available.
- [Detailed frontend suites](i16-frontend-tests.txt) passed for syntax, resolution,
  type graphs and checking. The optional Bun-backed failure-plan test is skipped
  in this detailed invocation; it runs in the full suite above.
- [Full staged integration, driver and emitter suites](i16-offline-tests.txt)
  passed. TestCurrentBundledNativeDeclarations builds a fresh release and invokes
  its absolute compiler outside the project with nonexistent PATH, network denied
  and the configured authentication variable absent. The fixture builds without
  credential lookup or requests. Cross-registration answer use, unknown metadata
  and wrong handler result types fail through that same frontend.
- Native syntax tests cover connections, fetch, LLM, judge, Noul, ordinary/record
  Choice and Score, dynamic description spreads and named arms. Parse/format/parse
  is stable. Missing emits, wrong section order, duplicate/missing Noul handlers,
  illegal record minimum, invalid shared handlers and misplaced probability fail.
- Resolver tests establish distinct declaration-kind eligibility, generated record
  identity and field order, independent export requirements, ordinary name
  collisions, wrong connection kind, and duplicate inputs/state/metadata.
- Program tests cover zero-, one- and multi-field state, variadic given followed
  by state, required grouping, full intrinsic error sets and profile identity.
  Judge registration arguments cannot access earlier answers; nonvoid binding
  annotations are exact and the continuation receives the checked answer locals.
- Handler tests cover Noul probability, dynamic Choice key scope and array type,
  Score metadata, generated record handlers, and metadata unavailable during
  descriptor preparation. Named arms reject free captures and invalid descriptions
  and supply evidence for inert named-arm initialization.
- Generated spread tests preserve inline/spread/metadata order and reject duplicate
  expanded fields, wrong arm result types and non-arm fields. Shape discovery is
  followed by full sealed expression checking; it is not source evaluation.
- The parser regression distinguishing a named arm declaration from a stored
  `choice_arm<T> emits [...]` binding passes the existing emitter suite.

[Shape design consultations](i16-jev/README.md) retain final requests/responses,
a failed HTTP 529 attempt and a superseded wording-audit round. Advice does not
replace the tests.

Runtime provider lowering is intentionally tracked by I17 and I26–I28. This
validation proves the frontend and declaration contracts, not provider execution,
provider answer quality or completion of the remaining implementation ledger.
