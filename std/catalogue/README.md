# Closed distribution catalogue

Generated from compiler/internal/catalogue/catalogue.json; do not edit this mirror.
Revision: **1**. Target: bun-1.4.2-darwin-arm64-v1. Source SHA-256: c1517500a51df8d56f3147c7cc7bf29e31bd75902ea6b0fb6b87a124b12ed1f8.

This is the complete approved descriptor inventory, not a claim that every
runtime adapter is implemented. Each native recipe names its implementation
task. No user host protocol, kernel, opaque representation or catalogue package
registration is available. Standard failures stay outside domain emits.

Regenerate with go run ./compiler/internal/catalogue/cmd/cataloguegen; verify
with the same command plus --check. Go tests also reject stale mirrors.

## Packages

- ai → can.std.ai@1
- asset → can.std.asset@1
- bytes → can.std.bytes@1
- cli → can.std.cli@1
- clock → can.std.clock@1
- codec → can.std.codec@1
- collections → can.std.collections@1
- crypto → can.std.crypto@1
- env → can.std.env@1
- html → can.std.html@1
- htmx → can.std.htmx@1
- http → can.std.http@1
- io → can.std.io@1
- json → can.std.json@1
- llm → can.std.llm@1
- log → can.std.log@1
- number → can.std.number@1
- option → can.std.option@1
- random → can.std.random@1
- sql → can.std.sql@1
- text → can.std.text@1

## Types

| Name | Kind | Parameters | Fields or leaves | Constructor |
| --- | --- | --- | --- | --- |
| choice_option | record |  | str key, str description | true |
| standard_failure | opaque |  | read-only: int occurrence_id, str kind, str message | false |
| option::none | record |  |  | true |
| option::some | record | T:data | T value | true |
| option::value | variant | T:data | option::none, option::some&lt;T&gt; | false |
| number::division | record |  | int quotient, int remainder | true |
| number::rounded | record |  | int value, int remainder_numerator, int denominator | true |
| collections::entry | record | K:map_key, V:data | K key, V value | true |
| collections::map | opaque | K:map_key, V:data |  | false |
| collections::set | opaque | K:map_key |  | false |
| http::header | record |  | str name, str value | true |
| http::response | record | T:data | int status, http::header[] headers, T body | true |
| bytes::buffer | opaque |  |  | false |
| html::node | opaque |  |  | false |
| html::safe | opaque |  |  | false |
| html::url | opaque |  |  | false |
| html::attribute | opaque |  |  | false |
| html::tag | opaque |  |  | false |
| htmx::target | opaque |  |  | false |
| htmx::swap | opaque |  |  | false |
| http::request | opaque |  |  | false |
| http::server_response | opaque |  |  | false |
| http::status | opaque |  |  | false |
| http::body_status | opaque |  |  | false |
| http::server_headers | opaque |  |  | false |
| http::route | opaque |  |  | false |
| http::router | opaque |  |  | false |
| http::server | opaque |  |  | false |
| http::server_config | opaque |  |  | false |
| sql::pool | opaque |  |  | false |
| sql::transaction | opaque |  |  | false |
| sql::commit | record | T:data | T value | true |
| sql::rollback | record | T:data | T value | true |
| sql::decision | variant | T:data | sql::commit&lt;T&gt;, sql::rollback&lt;T&gt; | false |

## Domain errors

| ID | Kind | Parameters | Ordered payload |
| --- | --- | --- | --- |
| 100 | all_failed | F:failure_variant | F[] failures |
| 1000 | number::inexact |  | str reason |
| 1001 | text::invalid_number |  | str input |
| 1002 | text::invalid_bool |  | str input |
| 1003 | number::invalid_bool |  | int value |
| 1004 | text::empty_separator |  |  |
| 1005 | text::empty_pattern |  |  |
| 1006 | text::invalid_unicode |  | str reason |
| 1007 | collections::key_absent |  |  |
| 1008 | collections::key_exists |  |  |
| 1009 | number::zero_divisor |  |  |
| 1100 | http::invalid_request |  | str reason |
| 1101 | http::credentials_missing |  | str variable |
| 1102 | http::transport_failed |  | str phase |
| 1103 | http::timeout |  | int timeout_ms |
| 1104 | http::body_limit |  | int limit |
| 1105 | http::status_error |  | int status, http::header[] headers |
| 1110 | codec::invalid_data |  | str path, str reason |
| 1120 | ai::invalid_question |  | str reason |
| 1121 | ai::invalid_answer |  | str question, str reason |
| 1130 | llm::refused |  | str reason |
| 1131 | llm::truncated |  |  |
| 1132 | llm::invalid_response |  | str reason |
| 1210 | io::read_failed |  | str operation |
| 1211 | io::write_failed |  | str operation |
| 1212 | io::limit_exceeded |  | int limit |
| 1220 | html::invalid_structure |  | str reason |
| 1221 | html::invalid_url |  | str reason |
| 1222 | htmx::invalid_target |  | str reason |
| 1223 | htmx::invalid_interval |  | int milliseconds |
| 1230 | http::invalid_route |  | str reason |
| 1231 | http::duplicate_route |  | str method, str path |
| 1232 | http::ambiguous_route |  | str first, str second |
| 1233 | http::invalid_server_config |  | str reason |
| 1234 | http::bind_failed |  | str address |
| 1235 | http::shutdown_failed |  | str phase |
| 1240 | sql::connection_failed |  | str phase |
| 1241 | sql::query_failed |  | str operation, str code |
| 1242 | sql::row_missing |  | str query |
| 1243 | sql::row_count |  | str query, int actual |
| 1244 | sql::schema_mismatch |  | str path, str reason |
| 1245 | sql::constraint_failed |  | str constraint |
| 1246 | sql::transaction_failed |  | str phase |
| 1247 | sql::commit_unknown |  | str transaction_id |
| 1248 | sql::close_failed |  | str reason |
| 1249 | sql::row_limit |  | int limit |
| 1250 | sql::unsupported_value |  | str path, str reason |
| 1260 | clock::invalid_duration |  | int milliseconds |
| 1261 | random::invalid_length |  | int length |
| 1262 | env::invalid_name |  | str name |
| 1263 | log::write_failed |  | str level |

## Operations

Callbacks have exact ordered input/result types. Derived means the bound is
the actual callback's finite domain error set; it is not authored effect syntax.
Real assertions run computation; supplied requires a boundary fixture;
scoped combines real adapter computation with fixture-owned opaque handles or
callbacks. Later assertion work must enforce those rules before side effects.

| Operation | Receiver; inputs → result | Domain bound | Callback contracts | Native mapping | Adapter contract | Assertion | Task / references |
| --- | --- | --- | --- | --- | --- | --- | --- |
| text::from_int | int value → str | [] |  | String | Native formatting without coercion at source boundaries. | real | I22 / C6 |
| text::from_float | float value → str | [] |  | String | Native formatting without coercion at source boundaries. | real | I22 / C6 |
| text::from_bool | bool value → str | [] |  | String | Native formatting without coercion at source boundaries. | real | I22 / C6 |
| number::int_to_float | int value → float | [number::inexact] |  | Number, BigInt | Validate exactness, finiteness or the 0/1 boolean range before conversion. | real | I22 / C6 |
| number::float_to_int | float value → int | [number::inexact] |  | Number, BigInt | Validate exactness, finiteness or the 0/1 boolean range before conversion. | real | I22 / C6 |
| number::bool_to_int | bool value → int | [] |  | Number, BigInt | Validate exactness, finiteness or the 0/1 boolean range before conversion. | real | I22 / C6 |
| number::int_to_bool | int value → bool | [number::invalid_bool] |  | Number, BigInt | Validate exactness, finiteness or the 0/1 boolean range before conversion. | real | I22 / C6 |
| number::floor | float value → float | [] |  | Math.floor | Preserve native IEEE outcomes and signed zero. | real | I22 / C6 |
| number::ceil | float value → float | [] |  | Math.ceil | Preserve native IEEE outcomes and signed zero. | real | I22 / C6 |
| number::trunc | float value → float | [] |  | Math.trunc | Preserve native IEEE outcomes and signed zero. | real | I22 / C6 |
| number::round | float value → float | [] |  | Math.round | Preserve native IEEE outcomes and signed zero. | real | I22 / C6 |
| number::is_finite | float value → bool | [] |  | Number.isFinite | Preserve native IEEE outcomes and signed zero. | real | I22 / C6 |
| number::is_nan | float value → bool | [] |  | Number.isNaN | Preserve native IEEE outcomes and signed zero. | real | I22 / C6 |
| text::to_int | str value → int | [text::invalid_number] |  | BigInt | Full-match the specified grammar; reject nonfinite parsed floats. | real | I22 / C6 |
| text::to_float | str value → float | [text::invalid_number] |  | Number | Full-match the specified grammar; reject nonfinite parsed floats. | real | I22 / C6 |
| text::to_bool | str value → bool | [text::invalid_bool] |  | === | Full-match the specified grammar; reject nonfinite parsed floats. | real | I22 / C6 |
| array.map | T:data, U:data; receiver T[]; $callback callback → U[] | [] + callback errors | callback(T) → U emits derived | Array.fromAsync, Array.prototype.keys, Array.prototype.map | Iterate indices sequentially, box callback output, then unwrap synchronously. | real | I21 / C7 |
| array.filter | T:data; receiver T[]; $callback callback → T[] | [] + callback errors | callback(T) → bool emits derived | Array.fromAsync, Array.prototype.filter | Collect ordered boolean decisions before native filtering. | real | I21 / C7 |
| array.for_each | T:data; receiver T[]; $callback callback → void | [] + callback errors | callback(T) → void emits derived | Array.prototype.reduce | Await each callback through a native reduce chain, stopping on failure. | real | I21 / C7 |
| array.fold | T:data, U:data; receiver T[]; U initial, $callback callback → U | [] + callback errors | callback(U,T) → U emits derived | Array.prototype.reduce | Box and await each accumulator with the explicit initial value. | real | I21 / C7 |
| array.find | T:data; receiver T[]; $callback callback → option::value&lt;T&gt; | [] + callback errors | callback(T) → bool emits derived | Array.prototype.values | Bounded ascending await adapter stops on the first true predicate. | real | I21 / C7 |
| array.some | T:data; receiver T[]; $callback callback → bool | [] + callback errors | callback(T) → bool emits derived | Array.prototype.values | Bounded ascending await adapter stops on true; empty returns false. | real | I21 / C7 |
| array.every | T:data; receiver T[]; $callback callback → bool | [] + callback errors | callback(T) → bool emits derived | Array.prototype.values | Bounded ascending await adapter stops on false; empty returns true. | real | I21 / C7 |
| array.sort_by | T:data, K:sort_key; receiver T[]; $callback callback → T[] | [] + callback errors | callback(T) → K emits derived | Array.fromAsync, Array.prototype.toSorted | Evaluate each key once, reject nonfinite keys, compare native keys with index tie-break. | real | I21 / C7 |
| array.slice | T:data; receiver T[]; int start, int end → T[] | [] |  | Array.prototype.slice | Normalize slice bounds in bigint; return immutable native copies. | real | I21 / C6,C7 |
| array.concat | T:data; receiver T[]; T[] other → T[] | [] |  | Array.prototype.concat | Normalize slice bounds in bigint; return immutable native copies. | real | I21 / C6,C7 |
| array.to_reversed | T:data; receiver T[];  → T[] | [] |  | Array.prototype.toReversed | Normalize slice bounds in bigint; return immutable native copies. | real | I21 / C6,C7 |
| append | T:data; T[] items, T value → T[] | [] |  | array spread | Copy the source and append one value without mutation. | real | I21 / C7 |
| str.includes | receiver str; str value → bool | [] |  | String.prototype.includes | Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars. | real | I24 / C6,C7 |
| str.starts_with | receiver str; str value → bool | [] |  | String.prototype.startsWith | Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars. | real | I24 / C6,C7 |
| str.ends_with | receiver str; str value → bool | [] |  | String.prototype.endsWith | Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars. | real | I24 / C6,C7 |
| str.to_lower_case | receiver str;  → str | [] |  | String.prototype.toLowerCase | Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars. | real | I24 / C6,C7 |
| str.to_upper_case | receiver str;  → str | [] |  | String.prototype.toUpperCase | Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars. | real | I24 / C6,C7 |
| str.trim | receiver str;  → str | [] |  | String.prototype.trim | Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars. | real | I24 / C6,C7 |
| str.slice | receiver str; int start, int end → str | [] |  | String.prototype.slice | Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars. | real | I24 / C6,C7 |
| str.split | receiver str; str separator → str[] | [text::empty_separator] |  | String.prototype.split | Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars. | real | I24 / C6,C7 |
| str.replace_all | receiver str; str search, str replacement → str | [text::empty_pattern] |  | String.prototype.replaceAll | Use C6 bigint slice normalization; reject empty split/search; replace via a function for literal dollars. | real | I24 / C6,C7 |
| text::join | str[] items, str separator → str | [] |  | Array.prototype.join | Native join with explicit separator. | real | I24 / C7 |
| text::scalars | str value → int[] | [text::invalid_unicode] |  | String.prototype.Symbol.iterator, String.prototype.codePointAt | Validate scalar values; chunk bulk code points; use locale und for graphemes and NFC for normalization. | real | I24 / C7 |
| text::from_scalars | int[] value → str | [text::invalid_unicode] |  | String.fromCodePoint | Validate scalar values; chunk bulk code points; use locale und for graphemes and NFC for normalization. | real | I24 / C7 |
| text::graphemes | str value → str[] | [text::invalid_unicode] |  | Intl.Segmenter | Validate scalar values; chunk bulk code points; use locale und for graphemes and NFC for normalization. | real | I24 / C7 |
| text::normalize_nfc | str value → str | [text::invalid_unicode] |  | String.prototype.normalize | Validate scalar values; chunk bulk code points; use locale und for graphemes and NFC for normalization. | real | I24 / C7 |
| collections::empty_map | K:map_key, V:data;  → collections::map&lt;K,V&gt; | [] |  | Map | Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order. | real | I25 / C7 |
| collections::empty_set | K:map_key;  → collections::set&lt;K&gt; | [] |  | Set | Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order. | real | I25 / C7 |
| collections::get | K:map_key, V:data; collections::map&lt;K,V&gt; map, K key → V | [collections::key_absent] |  | Map.prototype.has, Map.prototype.get | Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order. | real | I25 / C7 |
| collections::insert | K:map_key, V:data; collections::map&lt;K,V&gt; map, K key, V value → collections::map&lt;K,V&gt; | [collections::key_exists] |  | Map, Map.prototype.set | Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order. | real | I25 / C7 |
| collections::replace | K:map_key, V:data; collections::map&lt;K,V&gt; map, K key, V value → collections::map&lt;K,V&gt; | [collections::key_absent] |  | Map, Map.prototype.set | Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order. | real | I25 / C7 |
| collections::remove | K:map_key, V:data; collections::map&lt;K,V&gt; map, K key → collections::map&lt;K,V&gt; | [collections::key_absent] |  | Map, Map.prototype.delete | Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order. | real | I25 / C7 |
| collections::entries | K:map_key, V:data; collections::map&lt;K,V&gt; map → collections::entry&lt;K,V&gt;[] | [] |  | Map.prototype.entries, Array.from | Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order. | real | I25 / C7 |
| collections::contains | K:map_key; collections::set&lt;K&gt; set, K key → bool | [] |  | Set.prototype.has | Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order. | real | I25 / C7 |
| collections::add | K:map_key; collections::set&lt;K&gt; set, K key → collections::set&lt;K&gt; | [] |  | Set, Set.prototype.add | Validate opaque provenance and key kind; return owned immutable copies and preserve insertion order. | real | I25 / C7 |
| collections::union | K:map_key; collections::set&lt;K&gt; left, collections::set&lt;K&gt; right → collections::set&lt;K&gt; | [] |  | Set.prototype.union | Intersection/difference filter the left iteration order; union retains left then unseen right keys. | real | I25 / C7 |
| collections::intersection | K:map_key; collections::set&lt;K&gt; left, collections::set&lt;K&gt; right → collections::set&lt;K&gt; | [] |  | Set, Set.prototype.has, Array.prototype.filter | Intersection/difference filter the left iteration order; union retains left then unseen right keys. | real | I25 / C7 |
| collections::difference | K:map_key; collections::set&lt;K&gt; left, collections::set&lt;K&gt; right → collections::set&lt;K&gt; | [] |  | Set, Set.prototype.has, Array.prototype.filter | Intersection/difference filter the left iteration order; union retains left then unseen right keys. | real | I25 / C7 |
| number::divmod | int numerator, int denominator → number::division | [number::zero_divisor] |  | bigint /, bigint %, bigint comparisons | Apply only C8 sign normalization and finite half-even remainder adjustment. | real | I23 / C8 |
| number::euclidean_divmod | int numerator, int denominator → number::division | [number::zero_divisor] |  | bigint /, bigint %, bigint comparisons | Apply only C8 sign normalization and finite half-even remainder adjustment. | real | I23 / C8 |
| number::round_ratio_half_even | int numerator, int denominator → number::rounded | [number::zero_divisor] |  | bigint /, bigint %, bigint comparisons | Apply only C8 sign normalization and finite half-even remainder adjustment. | real | I23 / C8 |
| bytes::empty |  → bytes::buffer | [] |  | Uint8Array | Validate provenance, byte range and scalar text; copy buffers; decode UTF-8 fatally. | real | I13 / A2 |
| bytes::from_ints | int[] values → bytes::buffer | [codec::invalid_data] |  | Uint8Array.from | Validate provenance, byte range and scalar text; copy buffers; decode UTF-8 fatally. | real | I13 / A2 |
| bytes::to_ints | bytes::buffer buffer → int[] | [] |  | Array.from | Validate provenance, byte range and scalar text; copy buffers; decode UTF-8 fatally. | real | I13 / A2 |
| bytes::from_utf8 | str value → bytes::buffer | [codec::invalid_data] |  | TextEncoder | Validate provenance, byte range and scalar text; copy buffers; decode UTF-8 fatally. | real | I13 / A2 |
| bytes::to_utf8 | bytes::buffer buffer → str | [codec::invalid_data] |  | TextDecoder | Validate provenance, byte range and scalar text; copy buffers; decode UTF-8 fatally. | real | I13 / A2 |
| codec::encode_json | T:wire; T value → bytes::buffer | [codec::invalid_data] |  | JSON.parse, JSON.rawJSON, JSON.stringify, TextEncoder, TextDecoder | Derive nominal schema; preserve numeric tokens; guard duplicates, scalar text, cycles and A6 budgets. | real | I14 / A2,A6 |
| codec::decode_json | T:wire; bytes::buffer buffer → T | [codec::invalid_data] |  | JSON.parse, JSON.rawJSON, JSON.stringify, TextEncoder, TextDecoder | Derive nominal schema; preserve numeric tokens; guard duplicates, scalar text, cycles and A6 budgets. | real | I14 / A2,A6 |
| array.length | T:data; receiver T[];  → int | [] |  | Array.prototype.length, BigInt | Exact native length widened to bigint; validate opaque bytes before access. | real | I07 / C6,A2 |
| str.length | receiver str;  → int | [] |  | String.prototype.length, BigInt | Exact native length widened to bigint; validate opaque bytes before access. | real | I07 / C6,A2 |
| bytes.length | receiver bytes::buffer;  → int | [] |  | Uint8Array.prototype.byteLength, BigInt | Exact native length widened to bigint; validate opaque bytes before access. | real | I13 / C6,A2 |
| io::stdin_bytes | int max_bytes → bytes::buffer | [io::limit_exceeded, io::read_failed] |  | Bun.stdin | Reject negative limits before input; count incrementally as bigint, cancel on overflow, preserve immutable bytes and fatal BOM-preserving UTF-8. Map only expected native I/O errors. | supplied | I29 / P8 |
| io::stdin_text | int max_bytes → str | [io::limit_exceeded, io::read_failed, codec::invalid_data] |  | Bun.stdin, TextDecoder | Reject negative limits before input; count incrementally as bigint, cancel on overflow, preserve immutable bytes and fatal BOM-preserving UTF-8. Map only expected native I/O errors. | supplied | I29 / P8 |
| io::stdout_write | bytes::buffer buffer → int | [io::write_failed] |  | Bun.write, Bun.stdout | Await write and return exact byte count. | supplied | I29 / P8 |
| io::stderr_write | bytes::buffer buffer → int | [io::write_failed] |  | Bun.write, Bun.stderr | Await write and return exact byte count. | supplied | I29 / P8 |
| clock::wall_millis |  → int | [] |  | Date.now, BigInt | Return native epoch milliseconds exactly as bigint; supplied assertion boundary. | supplied | I30 / P8 |
| clock::monotonic_millis |  → float | [] |  | performance.now | Return native monotonic milliseconds as float; supplied assertion boundary. | supplied | I30 / P8 |
| clock::sleep_millis | int milliseconds → void | [clock::invalid_duration] |  | Bun.sleep | Validate bigint milliseconds in 0--2147483647 before Number conversion and await Bun.sleep; supplied assertion boundary. | supplied | I30 / P8 |
| random::secure_bytes | int length → bytes::buffer | [random::invalid_length] |  | crypto.getRandomValues, Uint8Array | Validate bigint length in 0--65536 before allocation; fill a fresh native array and preserve immutable byte ownership; supplied assertion boundary. | supplied | I30 / P8 |
| random::uuid_v4 |  → str | [] |  | crypto.randomUUID | Return native crypto.randomUUID; supplied assertion boundary. | supplied | I30 / P8 |
| crypto::sha256 | bytes::buffer buffer → bytes::buffer | [] |  | Bun.CryptoHasher | Hash copied immutable bytes with a fresh Bun.CryptoHasher and return owned digest bytes; execute in ordinary assertions. | real | I30 / P8 |
| env::required | str name → str | [env::invalid_name, http::credentials_missing] |  | Bun.env | Validate uppercase environment names before exact caller-snapshot lookup. Absence is distinct from a present empty string; assertion execution requires supplied completions. | supplied | I29 / P8 |
| env::optional | str name → option::value&lt;str&gt; | [env::invalid_name] |  | Bun.env | Validate uppercase environment names before exact caller-snapshot lookup. Absence is distinct from a present empty string; assertion execution requires supplied completions. | supplied | I29 / P8 |
| log::write_info | str message → void | [log::write_failed] |  | JSON.stringify, Bun.write | Serialize exactly level/info and message string fields with native JSON.stringify, append newline, await stderr; map serialization and expected I/O failures to a level-only payload; supplied assertion boundary. | supplied | I30 / P8 |
| log::write_error | str message → void | [log::write_failed] |  | JSON.stringify, Bun.write | Serialize exactly level/error and message string fields with native JSON.stringify, append newline, await stderr; map serialization and expected I/O failures to a level-only payload; supplied assertion boundary. | supplied | I30 / P8 |
| html::make_tag | str name → html::tag | [html::invalid_structure] |  | Set.prototype.has | Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization. | real | I31 / P6,P9 |
| html::text | str value → html::node | [] |  | Bun.escapeHTML | Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization. | real | I31 / P6,P9 |
| html::parse_url | str value → html::url | [html::invalid_url] |  | URL | Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization. | real | I31 / P6,P9 |
| html::text_attribute | str name, str value → html::attribute | [html::invalid_structure] |  | Bun.escapeHTML | Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization. | real | I31 / P6,P9 |
| html::url_attribute | str name, html::url url → html::attribute | [html::invalid_structure] |  | Bun.escapeHTML | Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization. | real | I31 / P6,P9 |
| html::element | html::tag tag, html::attribute[] attributes, html::node[] children → html::node | [html::invalid_structure] |  | Array.prototype.join, Bun.escapeHTML | Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization. | real | I31 / P6,P9 |
| html::fragment | html::node[] nodes → html::safe | [html::invalid_structure] |  | Array.prototype.join | Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization. | real | I31 / P6,P9 |
| html::text_fragment | str value → html::safe | [] |  | Bun.escapeHTML | Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization. | real | I31 / P6,P9 |
| html::stylesheet | html::url url → html::node | [html::invalid_url] |  | Bun.escapeHTML | Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization. | real | I31 / P6,P9 |
| html::meta_viewport |  → html::node | [] |  | string literal | Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization. | real | I31 / P6,P9 |
| html::document | str title, html::node[] head, html::node[] body → html::safe | [html::invalid_structure] |  | Bun.escapeHTML, Array.prototype.join | Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization. | real | I31 / P6,P9 |
| htmx::get | html::url url → html::attribute | [html::invalid_url] |  | Bun.escapeHTML | Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization. | real | I31 / P6,P9 |
| htmx::post | html::url url → html::attribute | [html::invalid_url] |  | Bun.escapeHTML | Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization. | real | I31 / P6,P9 |
| htmx::target_id | str id → htmx::target | [htmx::invalid_target] |  | String.prototype.match | Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization. | real | I31 / P6,P9 |
| htmx::target_attribute | htmx::target target → html::attribute | [] |  | Bun.escapeHTML | Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization. | real | I31 / P6,P9 |
| htmx::indicator_id | str id → html::attribute | [htmx::invalid_target] |  | Bun.escapeHTML | Enforce P9 closed tags/attributes, context, URL policy and opaque provenance before native serialization. | real | I31 / P6,P9 |
| htmx::swap_inner |  → html::attribute | [] |  | string literal | Construct only the fixed P9 HTMX attribute. | real | I31 / P9 |
| htmx::swap_outer |  → html::attribute | [] |  | string literal | Construct only the fixed P9 HTMX attribute. | real | I31 / P9 |
| htmx::trigger_change |  → html::attribute | [] |  | string literal | Construct only the fixed P9 HTMX attribute. | real | I31 / P9 |
| htmx::disable_this |  → html::attribute | [] |  | string literal | Construct only the fixed P9 HTMX attribute. | real | I31 / P9 |
| htmx::trigger_input_changed | int delay_ms → html::attribute | [htmx::invalid_interval] |  | String | Validate P9 interval bounds and construct the fixed trigger. | real | I31 / P9 |
| htmx::trigger_every | int interval_ms → html::attribute | [htmx::invalid_interval] |  | String | Validate P9 interval bounds and construct the fixed trigger. | real | I31 / P9 |
| htmx::runtime_head |  → html::node | [] |  | string literal | Emit only the pinned P11 local HTMX script and fixed response/config policy. | real | I34 / P9,P11 |
| asset::url | str name → html::url; static name | [html::invalid_url] |  | URL | Resolve a static declared asset to its validated build-manifest URL. | real | I34 / P11 |
| http::request_method | http::request request → str | [] |  | Request, URL, Headers | Read the immutable request snapshot and expose copied Can data. | scoped | I32 / P10 |
| http::request_path | http::request request → str | [] |  | Request, URL, Headers | Read the immutable request snapshot and expose copied Can data. | scoped | I32 / P10 |
| http::request_headers | http::request request → http::header[] | [] |  | Request, URL, Headers | Read the immutable request snapshot and expose copied Can data. | scoped | I32 / P10 |
| http::query_one | http::request request, str name → str | [http::invalid_request] |  | URLSearchParams | Check missing/repeated single query values; preserve repeated order. | scoped | I32 / P10 |
| http::query_all | http::request request, str name → str[] | [http::invalid_request] |  | URLSearchParams | Check missing/repeated single query values; preserve repeated order. | scoped | I32 / P10 |
| http::request_body | http::request request, int max_bytes → bytes::buffer | [http::body_limit] |  | Request, URLSearchParams, JSON.parse, TextDecoder | Read cached bounded bytes; apply the declared media/percent/UTF-8/typed codec policy. | scoped | I32 / P10 |
| http::request_json | T:wire; http::request request, int max_bytes → T | [http::body_limit, http::invalid_request, codec::invalid_data] |  | Request, URLSearchParams, JSON.parse, TextDecoder | Read cached bounded bytes; apply the declared media/percent/UTF-8/typed codec policy. | scoped | I32 / P10 |
| http::request_form | T:form; http::request request, int max_bytes → T | [http::body_limit, http::invalid_request, codec::invalid_data] |  | Request, URLSearchParams, JSON.parse, TextDecoder | Read cached bounded bytes; apply the declared media/percent/UTF-8/typed codec policy. | scoped | I32 / P10 |
| http::make_status | int status → http::status | [http::invalid_request] |  | Number | Admit 200–599; body status additionally excludes 204, 205 and 304. | real | I32 / P10 |
| http::make_body_status | int status → http::body_status | [http::invalid_request] |  | Number | Admit 200–599; body status additionally excludes 204, 205 and 304. | real | I32 / P10 |
| http::status_ok |  → http::body_status | [] |  | Number | Construct the corresponding fixed validated status. | real | I32 / P10 |
| http::status_unprocessable |  → http::body_status | [] |  | Number | Construct the corresponding fixed validated status. | real | I32 / P10 |
| http::status_internal |  → http::body_status | [] |  | Number | Construct the corresponding fixed validated status. | real | I32 / P10 |
| http::status_unavailable |  → http::body_status | [] |  | Number | Construct the corresponding fixed validated status. | real | I32 / P10 |
| http::make_server_headers | http::header[] headers → http::server_headers | [http::invalid_request] |  | Headers | Validate names/values and reject forbidden content/hop-by-hop fields. | real | I32 / P10 |
| http::empty_server_headers |  → http::server_headers | [] |  | Headers | Create the empty immutable header set. | real | I32 / P10 |
| http::response_empty | http::status status, http::server_headers headers → http::server_response | [] |  | Response, Headers, TextEncoder | Fixed media type and nosniff; no body for empty; JSON uses the shared exact codec. | real | I32 / P10 |
| http::response_bytes | http::body_status status, http::server_headers headers, bytes::buffer body → http::server_response | [] |  | Response, Headers, TextEncoder | Fixed media type and nosniff; no body for empty; JSON uses the shared exact codec. | real | I32 / P10 |
| http::response_text | http::body_status status, http::server_headers headers, str body → http::server_response | [] |  | Response, Headers, TextEncoder | Fixed media type and nosniff; no body for empty; JSON uses the shared exact codec. | real | I32 / P10 |
| http::response_html | http::body_status status, http::server_headers headers, html::safe body → http::server_response | [] |  | Response, Headers, TextEncoder | Fixed media type and nosniff; no body for empty; JSON uses the shared exact codec. | real | I32 / P10 |
| http::response_json | T:wire; http::body_status status, http::server_headers headers, T body → http::server_response | [codec::invalid_data] |  | Response, Headers, JSON.stringify | Fixed media type and nosniff; no body for empty; JSON uses the shared exact codec. | real | I32 / P10 |
| http::route_get | str path, $callback callback → http::route; static path | [http::invalid_route] | callback(http::request) → http::server_response emits [] | URL | Validate exact normalized path and mount a named boxed Can callback. | real | I32 / P10 |
| http::route_post | str path, $callback callback → http::route; static path | [http::invalid_route] | callback(http::request) → http::server_response emits [] | URL | Validate exact normalized path and mount a named boxed Can callback. | real | I32 / P10 |
| http::make_router | http::route[] routes → http::router | [http::duplicate_route, http::ambiguous_route] |  | Map | Closed exact dispatch with 404/405/Allow, no implicit HEAD. | real | I32 / P10 |
| http::make_server_config | str host, int port, int body_limit, int shutdown_ms → http::server_config | [http::invalid_server_config] |  | Number | Validate bounded config before server start. | real | I33 / P10 |
| http::server_start | http::server_config config, http::router router → http::server | [http::bind_failed] |  | Bun.serve | Register ownership; await each Can callback and sanitize standard failures. | supplied | I33 / P6,P10 |
| http::server_wait | http::server server → void | [http::shutdown_failed] |  | Bun.Server.stop, process.on | Wait for graceful stop(false), native stop and leases; timeout does not revoke ownership. | supplied | I33 / P6,P10 |
| http::server_stop | http::server server → void | [http::shutdown_failed] |  | Bun.Server.stop, process.on | Wait for graceful stop(false), native stop and leases; timeout does not revoke ownership. | supplied | I33 / P6,P10 |
| sql::pool_open | str connection_variable, int max_connections → sql::pool | [http::credentials_missing, sql::connection_failed] |  | Bun.SQL | Read selected credential; open PostgreSQL with bigint:true and validated max. | supplied | I35 / P12 |
| sql::pool_close | sql::pool pool, int timeout_ms → void | [sql::close_failed] |  | Bun.SQL.close | Drain leases then close with remaining deadline; leave timed-out close owned. | supplied | I35 / P6,P12 |
| sql::query_one | P:sql_parameters, R:sql_row; sql::pool handle, str descriptor, P parameters → R; static descriptor | [sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::constraint_failed, sql::row_missing, sql::row_count, sql::schema_mismatch] |  | Bun.SQL tagged template | Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1. | supplied | I35 / P12 |
| sql::query_optional | P:sql_parameters, R:sql_row; sql::pool handle, str descriptor, P parameters → option::value&lt;R&gt;; static descriptor | [sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::constraint_failed, sql::row_count, sql::schema_mismatch] |  | Bun.SQL tagged template | Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1. | supplied | I35 / P12 |
| sql::query_rows | P:sql_parameters, R:sql_row; sql::pool handle, str descriptor, P parameters, int max_rows → R[]; static descriptor | [sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::constraint_failed, sql::row_limit, sql::schema_mismatch] |  | Bun.SQL tagged template | Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1. | supplied | I35 / P12 |
| sql::execute | P:sql_parameters; sql::pool handle, str descriptor, P parameters → int; static descriptor | [sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::constraint_failed] |  | Bun.SQL tagged template | Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1. | supplied | I35 / P12 |
| sql::transaction_query_one | P:sql_parameters, R:sql_row; sql::transaction handle, str descriptor, P parameters → R; static descriptor | [sql::unsupported_value, sql::query_failed, sql::constraint_failed, sql::row_missing, sql::row_count, sql::schema_mismatch] |  | Bun.SQL tagged template | Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1. | supplied | I35 / P12 |
| sql::transaction_query_optional | P:sql_parameters, R:sql_row; sql::transaction handle, str descriptor, P parameters → option::value&lt;R&gt;; static descriptor | [sql::unsupported_value, sql::query_failed, sql::constraint_failed, sql::row_count, sql::schema_mismatch] |  | Bun.SQL tagged template | Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1. | supplied | I35 / P12 |
| sql::transaction_query_rows | P:sql_parameters, R:sql_row; sql::transaction handle, str descriptor, P parameters, int max_rows → R[]; static descriptor | [sql::unsupported_value, sql::query_failed, sql::constraint_failed, sql::row_limit, sql::schema_mismatch] |  | Bun.SQL tagged template | Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1. | supplied | I35 / P12 |
| sql::transaction_execute | P:sql_parameters; sql::transaction handle, str descriptor, P parameters → int; static descriptor | [sql::unsupported_value, sql::query_failed, sql::constraint_failed] |  | Bun.SQL tagged template | Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1. | supplied | I35 / P12 |
| sql::with_transaction | T:data; sql::pool pool, $callback callback → T | [sql::connection_failed, sql::transaction_failed, sql::commit_unknown] | callback(sql::transaction) → sql::decision&lt;T&gt; emits [] | Bun.SQL.begin | Drain scoped leases; private rollback sentinel; retain commit uncertainty and original standard failures. | scoped | I38 / P6,P12 |

## Native declaration profiles

These are the required intrinsic bounds, before checking the complete authored
emits declaration. Conditional modes are never narrowed by constant-success
speculation. Bound unions use the declared question/handler/body contracts.

| Mode | Fixed errors | Conditions | Additional bound sources | Task |
| --- | --- | --- | --- | --- |
| noul | ai::invalid_question, ai::invalid_answer |  | handlers | I27 |
| choice | ai::invalid_question, ai::invalid_answer |  | handlers | I27 |
| record_choice | ai::invalid_question, ai::invalid_answer |  | handlers | I27 |
| score | ai::invalid_question, ai::invalid_answer |  | handlers | I27 |
| record_score | ai::invalid_question, ai::invalid_answer |  | handlers | I27 |
| choice_arm |  |  | body | I27 |
| judge | http::invalid_request, http::transport_failed, http::timeout, http::body_limit, http::status_error, codec::invalid_data, ai::invalid_question, ai::invalid_answer | authenticated: http::credentials_missing | questions, handlers, continuation | I17 |
| fetch_body | http::invalid_request, http::transport_failed, http::timeout, http::body_limit, http::status_error | authenticated: http::credentials_missing; uses_codec: codec::invalid_data |  | I26 |
| fetch_envelope | http::invalid_request, http::transport_failed, http::timeout, http::body_limit | authenticated: http::credentials_missing; uses_codec: codec::invalid_data |  | I26 |
| llm | http::invalid_request, http::transport_failed, http::timeout, http::body_limit, http::status_error, codec::invalid_data, llm::refused, llm::truncated, llm::invalid_response | authenticated: http::credentials_missing |  | I28 |

Standard failure categories: arithmetic, bounds, resource_state, assertion, native_exception, cleanup.

Property descriptors (array.length, str.length, bytes.length) require no
call marker. Method descriptor names such as array.map are internal lookup
keys, not new reserved source packages. Package names cli and json remain
reserved even though this slice declares no callable members in them.
