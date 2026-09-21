# Owned native bytes

The existing catalogue signatures now compile to the shared bytes runtime:
`empty`, `from_ints`, `to_ints`, `from_utf8`, `to_utf8`, and the non-call `length`
projection. Ordinary callables can reference the functions and capture buffers.
Buffers have no authored constructor, fields, indexing, or mutation operation.

A frozen opaque token owns storage in a private WeakMap. Admission copies into a
plain Uint8Array, including native Buffer inputs (whose slice would alias).
Native consumers receive copies; no maintained function exports the backing
array. Integer conversion validates bigint octets in 0–255 before narrowing via
native Uint8Array.from. Array.from returns bigint values in a fresh frozen array.
Length widens the exact native length to bigint without copying the contents.

UTF-8 encoding uses TextEncoder and a BOM-preserving fatal TextDecoder round-trip
to reject lone surrogates. Decoding uses fatal TextDecoder with ignoreBOM enabled,
so an initial U+FEFF remains ordinary text. Expected failures use error 1110 and
the specified reasons: byte_range at the offending array index, unicode_scalar
for invalid text, utf8 for malformed encoding, and type for wrong scalar input.
Forged opaque buffers fail as resource_state before native access. Arbitrary
native faults retain the standard-failure path.

The CLI now consumes this shared representation and no longer owns UTF-8
conversion. These ordinary computations run during assertions without fixtures.
The integration fixture exercises native Request/Response bodies without network
traffic; it does not claim the later named-fetch transport policy is implemented.

[Validation evidence](evidence/2026-09-21/i13-validation.md).
