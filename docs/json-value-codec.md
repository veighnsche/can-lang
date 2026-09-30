# Immutable JSON values (P09)

`codec::decode_json_value(bytes::buffer)` returns `codec::json_value`.
`codec::encode_json_value(codec::json_value)` returns `bytes::buffer`.
Both emit `codec::invalid_data(path, reason)`.

The ordinary nominal variant has these constructible leaves:

| Constructor | Payload |
| --- | --- |
| `codec::json_null()` | none |
| `codec::json_bool(value)` | `bool` |
| `codec::json_int(value)` | `int` |
| `codec::json_float(value)` | `float` |
| `codec::json_string(value)` | `str` |
| `codec::json_array(values)` | `codec::json_value[]` |
| `codec::json_object(members)` | `codec::json_member[]` |

`codec::json_member(name, value)` holds `str name` and `codec::json_value value`.
Authored construction and pattern matching use normal Can record/variant rules.
The compiler passes concrete nominal identities to the maintained runtime.

Integer-spelled number tokens decode exactly to bigint-backed `json_int`, including
values beyond binary64 range. Tokens with a fraction or exponent, and `-0`, decode
to finite `json_float`. Encoding integral floats retains a decimal marker, and
negative zero retains its sign, so the cases survive a round trip. Fraction/exponent
spellings follow native binary64 semantics. Object enumeration follows native
JavaScript key ordering; JSON whitespace and original numeric spelling are not retained.

The implementation projects the existing bounded `parseDocument` result into
frozen records and arrays. Encoding validates nominal data and budgets before
native `JSON.stringify`, using `JSON.rawJSON` for exact numeric output. It rejects
duplicate object names, nonfinite floats, invalid Unicode scalars, cycles and
hostile data access. Existing UTF-8, 8 MiB byte, 64-container depth and one-million
wire-node limits apply. Typed `decode_json<T>` remains closed and unchanged.

## Completion evidence

- [x] P09.1: catalogue variant, records and operations; generated mirrors refreshed.
- [x] P09.2: runtime adapter and inventory, strict validation and immutable output.
- [x] P09.3: checker and Bun/browser emission bindings with concrete identities.
- [x] P09.4: runtime tests plus real Can fixture (10 assertions: envelope, notification,
  malformed/duplicate input, exact integer, nested scalar round trip, construction,
  main, and variant matching).
- [x] `bun run lint:fix:runtime`, `bun run format:runtime`, `bun run check:runtime`.
- [x] Focused Bun tests: 25 passed, 1,466 expectations across document/value,
  existing typed JSON/integer codec, and runtime module inventory tests.
- [x] Full catalogue package tests and generator `--check`; focused checker codec,
  emitter fixture, binding merge, core integration and browser binding tests.

Jev/System One was unavailable in the coordinating session; no consultation is
claimed. Source evidence and scope came from the P09 assignment in Manolea's
parallel implementation checklist and the existing codec/data implementation.

Temporary fixture directories use Go `t.TempDir` cleanup. The dependency symlink
was removed by an exit trap; no bundle, copied runtime, cache or prompt is retained.
An initial integration harness mistakenly wrote empty dependency artifacts through
its runtime symlink. The zero-byte tracked files were restored from clean base
`841fe28067ce906202f2a1610081b35123de8790`; the corrected harness skips runtime
placeholders. Final audit confirmed every other tracked runtime file matches that
base, with no zero-byte TypeScript files. The primary checkout was untouched.
