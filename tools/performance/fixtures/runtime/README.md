# Runtime fixtures and measured boundaries

The expanded driver covers four slices with distinct contracts:

| Slice | Initial cases | Expanded cases | Added boundaries |
|---|---:|---:|---|
| generated | 4 | 24 | Arithmetic/branching, reduction, nominal records, tagged variants, named failure recovery, generic/captured callables, Unicode text, UTF-8/base64 |
| runtime | 5 | 12 | Nominal record updates, owned capture invocation, completion success/error controls, resource lifetime, private bytes and boundary copies |
| codecs | 3 | 10 | Nested nominal Unicode records, nested immutability, exact encode/decode, measured type/member/duplicate/UTF-8 rejection paths |
| startup | 3 | 5 | A multi-operation emitted application as a precompiled bundle and as an independently pretranspiled JavaScript module graph |

## Generated execution

All Can fixtures are checked and emitted during preparation. Binding discovery
uses emitted origin identities, never numeric function names. `bindings.json`
records each entrypoint's module path, export, source bytes and SHA-256.
Initialization, record/variant fixture construction and correctness checks are
outside the generated execution timers.

`doubled.can` maps bigint values into an immutable array. `frequency.can` folds
up to ten distinct words through immutable map updates. `fold.can` reduces
bigints with a zero seed. `recovery.can` handles both `checks::failed` and success
by returning booleans. `paths.can` exercises bigint expressions and branches,
nominal record update/projection, record variants, a concrete int instantiation
of generic identity, a closure with a captured scalar offset, NFC normalization,
and UTF-8/base64 operations. Empty folds, both branch/variant/recovery paths,
negative captures and rejected Unicode surrogates receive untimed checks.

Handwritten native references preserve each named tested endpoint contract.
Scalar-entrypoint scenarios use the same host batch loop/await on both sides;
`ns/op` means one complete size-element workload, including that loop. Native
record updates explicitly use the runtime nominal adapter, because identity and
private metadata are part of the tested output. Generic/captured map references
preserve scalar values, frozen arrays and source snapshots; arbitrary callback
failures and resource captures are outside their comparison. The recovered
failure reference returns the same final boolean without allocating an internal
domain failure. Unicode/bytes references cover valid scalar strings, use native
`String.isWellFormed`, and do not claim identical failure payloads for malformed
inputs. These contracts permit native optimization of internal operations.

## Runtime controls

Bulk map construction and point insertion are separate workloads. The native
point-update reference uses private storage, frozen handles and production
ownership metadata registration. No mutable Map escapes. Successful map values
and old snapshots are checked; map failure registries are outside this scenario.

Additional cases exercise repeated nominal record updates, construction and
invocation of a scalar-capturing owned callable, completion invocation chains,
conversion of thrown TypeErrors to standard completions, and an owned root that
registers, uses and closes every resource exactly once. Private UTF-8 bytes are
round-tripped through production controls. A native boundary-copy case mutates
the returned copy during untimed checking and verifies the private/source bytes
are unchanged.

## Codecs

Production JSON schema projection and encoding cover bigint arrays and arrays
of nominal records containing Unicode/escaped text, bools, nested records and
string arrays. Checks include exact integers above 2^53, all nested fields,
nominal identities, frozen nested values, exact JSON bytes and input preservation.

Measured invalid nested payloads include a wrong type and missing member in the
last row, duplicate object members, and invalid UTF-8. Their exact CodecIssue
reasons and available paths are checked after timing. Rejection position matters;
these cases are not interchangeable with successful decoding. Input byte rates
use the entire supplied payload size and do not imply every rejected byte was
visited by schema projection. Native JSON.parse remains explicitly a parser
component, not a codec-contract replacement.

## Startup and profiles

Fresh Bun processes launch prepared JavaScript. The first three controls load a
minimal program, production completion controls, and an emitted array function.
The application controls run fold, captured mapping, immutable record update,
Unicode normalization, byte encoding and named error recovery. Both application
configurations have the same correctness assertions; one loads a prepared ESM
bundle, and the other loads the emitted module layout transpiled to `.js` once
with Bun.Transpiler during preparation. Relative static/literal dynamic import
specifiers are adjusted to `.js`; package and bun:/node: imports are unchanged.
The prepared module count is a footprint diagnostic, not a count of modules
actually loaded by Bun.

Parent startup wall time includes empty private environment snapshot setup,
process creation, JS parsing/module loading/evaluation, correctness assertions,
output capture and child exit. Generated-only execution timers exclude checks;
startup intentionally measures the complete ready-and-exit child. OS/Bun caches
are uncontrolled. Raw `launch_ms` retains individual launch durations and batch
means are `ms/launch`. Compilation is never inside measured launches.

Quick has three measured batches and standard has seven. `--iterations` gives
workload calls per batch, `--warmups` discarded warmup batches, and `--size`
array/map element counts or repeated text blocks. Startup uses fixed fixtures and
each iteration is a fresh process. The coordinator controls independent trials.
No request tails, allocation counts or peak-memory values are inferred from batch
averages or heap deltas; unsupported memory metrics remain unavailable.

## Independent emitter

Other families can compile the emitter from the repository root:

```sh
go build -o /absolute/work/perfemit ./compiler/perfemit
/absolute/work/perfemit /absolute/project /absolute/generated /absolute/repo/runtime
```

`drivers/runtime.py:adapter` discovers entrypoint exports by origin identity.
Import `array` and `value` from the same generated runtime tree as the function to
retain private identities. Full host-runtime imports require an empty JSON
launcher snapshot on fd 3. Existing dependencies are linked; none are installed.
