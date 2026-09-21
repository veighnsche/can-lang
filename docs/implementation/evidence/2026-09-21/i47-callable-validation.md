# I47 correction — structural callable and arm data

Report version: 1. Date: 2026-09-21.

The independent monitoring review identified a concrete public API failure:
`Resolve("append", GeneratedTargetID, GeneratedRevision,
map[string]string{"T":"callable int () emits []"}, nil, admission)` failed
before type admission. The original catalogue descriptor parser represented only
names, generics and arrays. Seven regression cases reproduced that failure for
callables, choice arms, void-result contracts, and nested generic functional data.

Concrete types now use I04's authoritative `syntax.ParseType`, with an explicit
bridge into structural catalogue nodes. Results, inputs and emitted error types
remain structured through generic substitution. The inventory's uppercase template
notation retains its separate small parser and is never a fallback for malformed
concrete source types. No second callable source grammar was introduced.

Data admission validates functional inputs/results and error kinds. A void result
is permitted; void inputs and void array elements are not. Project nominal data
requires checked `data` evidence and project error bounds require checked `error`
evidence from TypeAdmission. The hook cannot invent a type or error in a reserved
catalogue package. Known wire/key/form/SQL exclusions remain enforced, including
inside catalogue generic containers. This correction changes representation and
specialization, not callable invocation rules or executable-arm spread syntax.

Concrete argument spelling is canonicalized before specialization. Callback
contracts are compared after the same source-type conversion, preserving nested
bounds while accepting insignificant spacing differences. Generic error identity
arguments use the source bridge as well. ErrorIdentity continues to describe an
already checked specialization; it is not a replacement for I49's registry and
error-kind checking.

Verification:

- [Catalogue tests](i47-callable-tests.txt): all seven initial positive regressions;
  structural conversion and nested substitution; callback-bound preservation;
  malformed/void/unknown-reserved error contracts; project evidence routing; wire,
  map-key, sort-key, form and SQL boundary refusal; existing catalogue/drift tests.
- [Full Go suite](i47-callable-go-suite.txt): all packages pass.
- [Bun catalogue mirror](i47-callable-bun-tests.txt): one suite passes on pinned
  Bun1.4.2. The authored inventory and generated mirrors did not change.
- [Packaged boundary](i47-callable-packaged.txt): all nine runtime mirror checks and
  the Go/TypeScript error occurrence round-trip pass through the staged absolute
  runtime, with networking denied and no Bun on PATH.
- [Three rewritten Jev consultations](i47-callable-jev/README.md) selected the
  structural bridge. These were design advice, not the acceptance oracle.

I05 was inspected but remains unchecked and unimplemented. This correction is
committed separately before resuming manifest/package resolution.
