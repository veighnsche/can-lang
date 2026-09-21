# I30 finite native host utilities

Clock operations use Date.now, performance.now and awaited Bun.sleep. Sleep
validates bigint duration before conversion (0 through 2147483647). Secure random
bytes validate 0 through 65536 before allocating a fresh array for native
crypto.getRandomValues; UUID uses crypto.randomUUID. SHA-256 uses a fresh
Bun.CryptoHasher over a copy of the immutable input and returns owned bytes.

Logging serializes only the two string fields `level` and `message`, appends one
newline and awaits Bun.write to stderr. Native JSON escaping preserves text,
including newlines and lone UTF-16 surrogates. Serialization TypeError/RangeError
and recognized native I/O errors produce log::write_failed with only a level
payload. Unrelated defects retain the standard channel. No value inspection or
application toJSON hook enters serialization.

The compiler admits the finite I30 catalogue and emits these maintained adapters.
Clock, sleep, random, UUID and log calls require supplied assertion completions;
SHA-256 is ordinary deterministic computation. No fallback reads or writes occur
when a host-observation fixture is absent.

## Evidence

- Runtime tests check millisecond units, monotonic observations, awaited sleep,
  lower/upper invalid duration and length values (including huge bigint), random
  sizes 0/1/65536, byte immutability, UUID version/variant syntax, SHA-256 empty
  and abc vectors and a binary oracle from node:crypto.
- Logging tests check exact JSON lines and awaited writes, serialization and
  EPIPE failures, level-only error payloads, standard defects, and proxy inputs
  without traps. Actual entry diagnostics retain error 1263 but omit private
  messages and native causes.
- Missing host fixtures fail without ambient operations. A normal assertion
  actually hashes input and reports `real-can` evidence.
- `TestCurrentBundledUtilities` runs the absolute packaged CLI offline with an
  empty PATH. The same source passes supplied boundary assertions and real
  native execution; SHA-256 stays real in both. Its stderr is exactly the two
  authored JSON log lines. Invalid duration/length values return 1260/1261
  without later logging or secret disclosure. Generated TypeScript is strict.
- A separate assertion expected expression invokes native SHA-256 and converts
  its result to integer bytes, covering the generated assertion-module import
  path in addition to ordinary function emission.

Final gate: `bun test runtime` passed 226 tests and 21005 expectations. Strict
TypeScript passed for utility adapters/tests and the staged generated program.
After adding the assertion-expression import regression, the final full
`go test ./compiler/... ./tests/integration -count=1` passed with the pinned Bun
archive and generated TypeScript checking (integration 110.352s).
