# I25 validation — immutable native maps and sets

The closed twelve-operation collections catalogue now lowers to native-backed
immutable values. Map and Set storage remains private in module WeakMaps behind
frozen opaque tokens. Each entry records its sealed concrete collection identity;
matching factories share provenance while forged, copied, proxied and differently
typed tokens reject. Map values retain their original aliases, including a
then-bearing object that must not be assimilated by Promise resolution.

Map operations use native copies, has/get/set/delete and ordered entry iteration.
Inserted duplicates emit key_exists (1008); absent get/replace/remove emit
key_absent (1007). Entry arrays and nominal entry records are frozen, and replace
retains key position. Keys are restricted to int, bool and str in both checked
contracts and runtime admission. Native Set union and difference retain the
required order. Intersection uses Array.from(left).filter(key => right.has(key))
and a fresh Set, avoiding the smaller-set native intersection order change.
Both size branches produce [2,1] for left [3,2,1] and right containing [1,2].

Compiler admission parses the maintained generic catalogue into existing finite
inference patterns. Concrete specialization records retain exact contracts and
collection/entry identities. Emission binds native factories to those identities
through the existing completion, callable, argument-preparation and fixture ABI.
There are no invented source bodies, loose inference placeholders or new IR
execution mechanism. Explicit and inferred applications, contextual empty
constructors, callable references, array callbacks and fixed literal spreads are
covered. Runtime domain admission recognizes exact collection provenance.

Validation:

- Runtime collections: 4 tests, 61 expectations; order, aliases, immutable copies,
  allocated errors, all three key families, invalid keys, fake/copy/proxy tokens,
  wrong concrete identities, matching factories and both intersection branches.
- Runtime adapter/test strict TypeScript checking passed.
- Compiler positive examples and seven negative contract mutations passed.
- Staged release: 19 Can assertion roots, executable main and generated strict
  TypeScript passed offline through the absolute bundled runtime. Main checks
  replacement preserves its original alias and folds an array into a native set.
- Full compiler and offline integration gate passed with the pinned Bun archive
  and CAN_TSC configured: `go test ./compiler/... ./tests/integration -count=1`.
- Full runtime suite: 171 tests, 19,254 expectations, zero failures.

The shared current map/set source project is `std/map/current`. Both historical
package READMEs identify it as the maintained replacement; legacy source/generated
files remain migration inputs for I43/I44 and are not fallback implementations.

Three fresh Jev requests and responses are retained in [i25-jev](i25-jev/README.md).
Their unanimous preference informed the implementation; the behavior checks above
provide the validation evidence.
