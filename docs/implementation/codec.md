# Typed native JSON boundary

The compiler admits `codec::encode_json<T>` and `codec::decode_json<T>` with one
explicit checked concrete type argument. They can also be named callable values.
General source generic functions and inference remain I46. The sealed codec
schema is a finite graph: recursive fields refer to identities, and variants
list concrete nominal leaves with source-qualified canonical tags. Private
package/declaration identities remain separate, and ambiguous wire tags reject. Unsupported types fail even
when reached through a field or array. Bytes and other opaque values, callables,
choice arms and void are not JSON data.

Decode first bounds input bytes, preserves BOM during fatal UTF-8 conversion,
and scans only keys, nesting and value counts. The scanner constructs no values
and never admits syntax. Duplicate diagnosis is deferred until native JSON.parse
accepts the complete bounded document. Pending duplicates retain the escaped
RFC 6901 path of the offending key, including enclosing array indexes. Its source-context reviver associates
primitive tokens with their holder and property, including the root. Typed
traversal uses original numeric tokens for integers, native Number values for
floats, and rebuilds frozen arrays and nominal records after exact shape checks.
Missing fields follow declaration order; extra fields sort by decoded key.

Integer normalization compares the signed exponent text against bounded native
counts before any exponent expansion or bigint construction. Zero handles any
exponent without expansion. Nonintegrality precedes byte-limit diagnosis. Every
integer's canonical signed decimal length charges the same operation budget.
Encoding proves bigint length before String and bounds strings before native
escaping. Preflight counts every delimiter, key, colon, comma, wrapper and
primitive, alongside depth, node and cycle checks. Shared subtrees are permitted.

JSON.stringify performs final formatting through lazy container access views in
schema order. These views expose one container at a time over the original data;
there is no independently assembled wire-value tree. Integers use native rawJSON
and floats use rawJSON for negative zero. Actual UTF-8 output length must agree
with preflight. Native failures outside expected syntax/encoding cases remain
standard failures.

The standalone byte budget is 8,388,608; depth is 64 and value nodes 1,000,000.
Private transport consumers may replace only the byte limit (up to 67,108,864).
`encodeJSON`/`decodeJSON` provide the shared schema-directed implementation for
later fetch, AI state and LLM output adapters. `createCodec` supplies the protected
completion ABI and exact `codec::invalid_data` error for authored calls. Later provider tasks
must reuse this implementation and apply their specified transport classifications;
this task does not claim those not-yet-implemented providers execute.

[Validation](evidence/2026-09-21/i14-validation.md) includes staged source execution,
native limit tests and the three saved design consultations.
