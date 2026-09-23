# Closed distribution catalogue

Generated from compiler/internal/catalogue/catalogue.json; do not edit this mirror.
Revision: **1**. Target: bun-1.4.2-darwin-arm64-v1. Source SHA-256: e4b64d70c19a181494f2d60456bc70d026a903e0593337caccd7b85045e127a3.

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
- checks → can.std.checks@1
- cli → can.std.cli@1
- clock → can.std.clock@1
- codec → can.std.codec@1
- collections → can.std.collections@1
- cookie → can.std.cookie@1
- crypto → can.std.crypto@1
- csrf → can.std.csrf@1
- env → can.std.env@1
- files → can.std.files@1
- html → can.std.html@1
- htmx → can.std.htmx@1
- http → can.std.http@1
- io → can.std.io@1
- json → can.std.json@1
- llm → can.std.llm@1
- log → can.std.log@1
- number → can.std.number@1
- option → can.std.option@1
- password → can.std.password@1
- path → can.std.path@1
- process → can.std.process@1
- random → can.std.random@1
- sql → can.std.sql@1
- stream → can.std.stream@1
- text → can.std.text@1
- time → can.std.time@1
- url → can.std.url@1
- ws → can.std.ws@1

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
| http::sse_event | record |  | str data, str event, str id, str retry | true |
| http::multipart_form | record |  | http::multipart_field[] fields, http::multipart_file[] files | true |
| http::multipart_field | record |  | str name, str value | true |
| http::multipart_file | record |  | str name, str filename, str content_type, bytes::buffer content | true |
| http::response | record | T:data | int status, http::header[] headers, T body | true |
| http::failure_detail | variant |  | http::invalid_request, http::credentials_missing, http::transport_failed, http::timeout, http::body_limit, http::status_error, codec::invalid_data | false |
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
| http::tls_config | opaque |  |  | false |
| sql::pool | opaque |  |  | false |
| sql::transaction | opaque |  |  | false |
| sql::sqlite_file_options | record |  | str mode, int busy_timeout_ms | true |
| sql::commit | record | T:data | T value | true |
| sql::rollback | record | T:data | T value | true |
| sql::decision | variant | T:data | sql::commit&lt;T&gt;, sql::rollback&lt;T&gt; | false |
| files::file_info | record |  | str kind, int size | true |
| files::entry | record |  | str path, str kind | true |
| process::options | record |  | str cwd, bool inherit_env, str[] env, bytes::buffer stdin, int stdout_limit, int stderr_limit, int deadline_ms, int grace_ms | true |
| process::result | record |  | bytes::buffer stdout, bytes::buffer stderr, int code, str signal | true |
| stream::reader | opaque | T:data |  | false |
| stream::writer | opaque |  |  | false |
| crypto::key | opaque |  |  | false |
| crypto::keypair | record |  | crypto::key private_key, crypto::key public_key | true |
| crypto::sealed | record |  | bytes::buffer nonce, bytes::buffer ciphertext | true |
| url::parts | record |  | str scheme, str host, int port, str path, str query, str fragment | true |
| url::query_pair | record |  | str name, str value | true |
| text::regex | opaque |  |  | false |
| text::regex_match | record |  | str text, int start, int end, str[] groups | true |
| time::instant | opaque |  |  | false |
| time::civil | record |  | int year, int month, int day, int hour, int minute, int second, int millisecond | true |
| ws::session | opaque |  |  | false |
| ws::connection | record |  | ws::session session, stream::reader&lt;ws::event&gt; events, str protocol | true |
| ws::event | variant |  | ws::text, ws::binary, ws::drain, ws::closed | false |
| ws::text | record |  | str text | true |
| ws::binary | record |  | bytes::buffer data | true |
| ws::drain | record |  |  | true |
| ws::closed | record |  | int code, str reason | true |
| cookie::attributes | record |  | str path, option::value&lt;str&gt; domain, bool secure, bool http_only, cookie::same_site same_site, option::value&lt;int&gt; max_age, option::value&lt;int&gt; expires_ms | true |
| cookie::same_site | variant |  | cookie::strict, cookie::lax, cookie::none | false |
| cookie::strict | record |  |  | true |
| cookie::lax | record |  |  | true |
| cookie::none | record |  |  | true |
| cookie::cookie | opaque |  |  | false |
| cookie::collection | record |  | cookie::pair[] pairs | true |
| cookie::pair | record |  | str name, str value | true |

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
| 1010 | checks::failed |  | str reason |
| 1100 | http::invalid_request |  | str reason |
| 1101 | http::credentials_missing |  | str variable |
| 1102 | http::transport_failed |  | str phase |
| 1103 | http::timeout |  | int timeout_ms |
| 1104 | http::body_limit |  | int limit |
| 1105 | http::status_error |  | int status, http::header[] headers |
| 1106 | http::request_failed |  | http::failure_detail detail |
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
| 1300 | files::not_found |  | str path |
| 1301 | files::denied |  | str path, str operation |
| 1302 | files::already_exists |  | str path |
| 1303 | files::invalid_path |  | str path, str reason |
| 1304 | files::io_error |  | str path, str operation |
| 1305 | files::limit_exceeded |  | int limit |
| 1306 | files::not_empty |  | str path |
| 1307 | files::cross_device |  | str source, str destination |
| 1308 | files::unexpected_kind |  | str path, str operation |
| 1310 | process::spawn_failed |  | str executable |
| 1311 | process::timeout |  | int deadline_ms |
| 1312 | process::output_limit |  | str stream, int limit |
| 1313 | process::nonzero |  | int code, str signal |
| 1314 | process::invalid_config |  | str field, str reason |
| 1315 | process::io_error |  | str operation |
| 1316 | stream::read_failed |  | str reason |
| 1317 | stream::write_failed |  | str reason |
| 1318 | stream::cancelled |  | str reason |
| 1319 | stream::close_failed |  | str reason |
| 1320 | password::cost_rejected |  | int profile |
| 1321 | password::invalid_hash |  | str reason |
| 1322 | crypto::invalid_key |  | str reason |
| 1323 | crypto::invalid_nonce |  | int length |
| 1324 | crypto::key_misuse |  | str operation, str algorithm |
| 1325 | crypto::decrypt_failed |  |  |
| 1326 | url::invalid_url |  | str reason |
| 1327 | text::invalid_regex |  | str reason |
| 1328 | text::invalid_limit |  | int limit |
| 1329 | time::out_of_range |  | int millis |
| 1330 | time::invalid_zone |  | str zone |
| 1331 | time::nonexistent_time |  |  |
| 1332 | time::invalid_option |  | str reason |
| 1333 | ws::connect_failed |  | str reason |
| 1334 | ws::upgrade_failed |  | str reason |
| 1335 | ws::unsupported_protocol |  | str protocol |
| 1336 | ws::send_failed |  | str reason |
| 1337 | ws::invalid_close |  | str reason |
| 1338 | ws::limit_exceeded |  | int limit |
| 1339 | ws::invalid_url |  | str reason |
| 1340 | ws::invalid_protocol |  | str protocol |
| 1341 | cookie::invalid_cookie |  | str reason |
| 1342 | csrf::invalid_config |  | str reason |

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
| text::compile_regex | str pattern, str flags → text::regex | [text::invalid_regex] |  | RegExp | Compile validated patterns with i/m/s/u/v flags into opaque handles; other flags and bad patterns reject; execute in ordinary assertions. | real | B1-13 / B1-13 |
| text::matches | text::regex regex, str text, int limit → text::regex_match[] | [text::invalid_limit] |  | RegExp | Scan with a fresh global pass per call (no shared lastIndex), UTF-16 offsets, empty-match advancement, absent captures as empty; execute in ordinary assertions. | real | B1-13 / B1-13 |
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
| bytes::encode_base64 | bytes::buffer buffer → str | [] |  | Buffer | Encode standard base64 with padding; execute in ordinary assertions. | real | B1-13 / B1-13 |
| bytes::decode_base64 | str text → bytes::buffer | [codec::invalid_data] |  | Buffer | Decode standard base64 only after strict alphabet/padding gates; malformed text rejects, never truncates; execute in ordinary assertions. | real | B1-13 / B1-13 |
| bytes::encode_hex | bytes::buffer buffer → str | [] |  | Buffer | Encode lowercase hex; execute in ordinary assertions. | real | B1-13 / B1-13 |
| bytes::decode_hex | str text → bytes::buffer | [codec::invalid_data] |  | Buffer | Decode hex only after strict even-length alphabet gates; malformed text rejects, never truncates; execute in ordinary assertions. | real | B1-13 / B1-13 |
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
| password::hash | str password, int profile → str | [password::cost_rejected] |  | Bun.password | Hash with async Bun.password (argon2id) under fixed preset 0..2 (fast 8MiB/1 pass, balanced 64MiB/2 passes, secure 256MiB/3 passes); other profiles reject; supplied assertion boundary. | supplied | B1-08 / B1-08 |
| password::verify | str password, str encoded → bool | [password::invalid_hash] |  | Bun.password | Gate the qualified argon2id envelope (version 19, memory 8 KiB..1 GiB, time 1..32, parallelism 1..4, 32-byte salt and hash) before native verify; malformed hashes reject while wrong passwords read false; execute in ordinary assertions. | real | B1-08 / B1-08 |
| crypto::sha256 | bytes::buffer buffer → bytes::buffer | [] |  | Bun.CryptoHasher | Hash copied immutable bytes with a fresh Bun.CryptoHasher and return owned digest bytes; execute in ordinary assertions. | real | I30 / P8 |
| crypto::hmac_sha256 | bytes::buffer key, bytes::buffer message → bytes::buffer | [crypto::invalid_key] |  | crypto.subtle | Import the raw HMAC/SHA-256 key per call and sign; empty keys reject; execute in ordinary assertions. | real | B1-08 / B1-08 |
| crypto::generate_aes_key |  → crypto::key | [] |  | crypto.subtle | Generate a native-nonextractable AES-256-GCM encrypt/decrypt handle; supplied assertion boundary. | supplied | B1-08 / B1-08 |
| crypto::generate_ed25519_keypair |  → crypto::keypair | [] |  | crypto.subtle | Generate sign-only private plus verify-only public handles; supplied assertion boundary. | supplied | B1-08 / B1-08 |
| crypto::import_ed25519_public | bytes::buffer public → crypto::key | [crypto::invalid_key] |  | crypto.subtle | Admit 32-byte verify-only Ed25519 public keys; other lengths reject; execute in ordinary assertions. | real | B1-08 / B1-08 |
| crypto::export_ed25519_public | crypto::key key → bytes::buffer | [crypto::key_misuse] |  | crypto.subtle | Export raw bytes only from verify-only Ed25519 handles; sign-capable and AES handles misuse; execute in ordinary assertions. | real | B1-08 / B1-08 |
| crypto::encrypt_aes_gcm | crypto::key key, bytes::buffer nonce, bytes::buffer plaintext, bytes::buffer associated_data → bytes::buffer | [crypto::invalid_nonce, crypto::key_misuse] |  | crypto.subtle | Encrypt with a fixed 12-byte nonce and 128-bit tag through usage-checked handles; execute in ordinary assertions. | real | B1-08 / B1-08 |
| crypto::encrypt_aes_gcm_sealed | crypto::key key, bytes::buffer plaintext, bytes::buffer associated_data → crypto::sealed | [crypto::key_misuse] |  | crypto.subtle | Mint a native-random 12-byte nonce and return it with the ciphertext; generation is randomness, not a guarantee of caller nonce discipline; supplied assertion boundary. | supplied | B1-08 / B1-08 |
| crypto::decrypt_aes_gcm | crypto::key key, bytes::buffer nonce, bytes::buffer ciphertext, bytes::buffer associated_data → bytes::buffer | [crypto::invalid_nonce, crypto::key_misuse, crypto::decrypt_failed] |  | crypto.subtle | Decrypt with a fixed 12-byte nonce and 128-bit tag; tampering, wrong keys, and associated-data mismatch collapse to decrypt_failed; execute in ordinary assertions. | real | B1-08 / B1-08 |
| crypto::sign_ed25519 | crypto::key key, bytes::buffer message → bytes::buffer | [crypto::key_misuse] |  | crypto.subtle | Sign through sign-capable handles; deterministic 64-byte signatures; execute in ordinary assertions. | real | B1-08 / B1-08 |
| crypto::verify_ed25519 | crypto::key key, bytes::buffer message, bytes::buffer signature → bool | [crypto::key_misuse] |  | crypto.subtle | Verify through verify-capable handles; mismatch reads false; execute in ordinary assertions. | real | B1-08 / B1-08 |
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
| http::request_multipart | http::request request, int max_bytes, int max_file_bytes → http::multipart_form | [http::body_limit, http::invalid_request, codec::invalid_data] |  | TextDecoder | Parse bounded flat multipart into generic field/file records. | scoped | B1-06 / P10 |
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
| http::response_stream | http::body_status status, http::server_headers headers → http::server_response | [http::invalid_request] |  | ReadableStream | Build a pending response whose bounded queue the vended writer fills. | real | B1-06 / P10 |
| http::response_writer | http::server_response response → stream::writer | [http::invalid_request] |  | ReadableStream | Vend the pending response writer exactly once for short-write production. | supplied | B1-06 / P10 |
| http::response_sse | http::body_status status, http::server_headers headers → http::server_response | [http::invalid_request] |  | ReadableStream | Build a pending event-stream response for validated SSE production. | real | B1-06 / P10 |
| http::sse_send | stream::writer writer, http::sse_event event → int | [http::invalid_request, http::body_limit, stream::write_failed] |  | ReadableStream, TextEncoder | Validate one event frame and append it atomically to the queue. | supplied | B1-06 / P10 |
| http::sse_comment | stream::writer writer, str text → int | [http::invalid_request, http::body_limit, stream::write_failed] |  | ReadableStream, TextEncoder | Validate one comment line and append it atomically to the queue. | supplied | B1-06 / P10 |
| http::route_get | str path, $callback callback → http::route; static path | [http::invalid_route] | callback(http::request) → http::server_response emits [] | URL | Validate exact normalized path and mount a named boxed Can callback. | real | I32 / P10 |
| http::route_post | str path, $callback callback → http::route; static path | [http::invalid_route] | callback(http::request) → http::server_response emits [] | URL | Validate exact normalized path and mount a named boxed Can callback. | real | I32 / P10 |
| http::route_put | str path, $callback callback → http::route; static path | [http::invalid_route] | callback(http::request) → http::server_response emits [] | URL | Validate exact normalized path and mount a named boxed Can callback. | real | B1-06 / P10 |
| http::route_patch | str path, $callback callback → http::route; static path | [http::invalid_route] | callback(http::request) → http::server_response emits [] | URL | Validate exact normalized path and mount a named boxed Can callback. | real | B1-06 / P10 |
| http::route_delete | str path, $callback callback → http::route; static path | [http::invalid_route] | callback(http::request) → http::server_response emits [] | URL | Validate exact normalized path and mount a named boxed Can callback. | real | B1-06 / P10 |
| http::route_options | str path, $callback callback → http::route; static path | [http::invalid_route] | callback(http::request) → http::server_response emits [] | URL | Validate exact normalized path and mount a named boxed Can callback. | real | B1-06 / P10 |
| http::route_head | str path, $callback callback → http::route; static path | [http::invalid_route] | callback(http::request) → http::server_response emits [] | URL | Validate exact normalized path and mount a named boxed Can callback. | real | B1-06 / P10 |
| http::make_router | http::route[] routes → http::router | [http::duplicate_route, http::ambiguous_route] |  | Map | Closed exact dispatch with 404/405/Allow, no implicit HEAD. | real | I32 / P10 |
| http::route_stream | http::route route → http::route | [] |  | Map | Mark a route for lazy bodies consumed once through a stream reader. | real | B1-06 / P10 |
| http::request_body_stream | http::request request, int max_chunk → stream::reader&lt;bytes::buffer&gt; | [http::body_limit, http::invalid_request] |  | ReadableStream | Open the one-shot body reader: live wire bytes or replayed buffered bytes. | supplied | B1-06 / P10 |
| http::make_server_config | str host, int port, int body_limit, int shutdown_ms → http::server_config | [http::invalid_server_config] |  | Number | Validate bounded config before server start. | real | I33 / P10 |
| http::server_start | http::server_config config, http::router router → http::server | [http::bind_failed] |  | Bun.serve | Register ownership; await each Can callback and sanitize standard failures. | supplied | I33 / P6,P10 |
| http::make_tls_config | bytes::buffer cert, bytes::buffer key → http::tls_config | [http::invalid_server_config] |  | TextDecoder | Validate PEM certificate chain and private key structure before server start. | real | B1-06 / P10 |
| http::server_start_tls | http::server_config config, http::router router, http::tls_config tls → http::server | [http::bind_failed] |  | Bun.serve | Register ownership; serve TLS with the validated material and sanitize standard failures. | supplied | B1-06 / P6,P10 |
| http::server_wait | http::server server → void | [http::shutdown_failed] |  | Bun.Server.stop, process.on | Wait for graceful stop(false), native stop and leases; timeout does not revoke ownership. | supplied | I33 / P6,P10 |
| http::server_stop | http::server server → void | [http::shutdown_failed] |  | Bun.Server.stop, process.on | Wait for graceful stop(false), native stop and leases; timeout does not revoke ownership. | supplied | I33 / P6,P10 |
| sql::pool_open | str connection_variable, int max_connections → sql::pool | [http::credentials_missing, sql::connection_failed] |  | Bun.SQL | Read selected credential; open PostgreSQL with bigint:true and validated max. | supplied | I35 / P12 |
| sql::pool_close | sql::pool pool, int timeout_ms → void | [sql::close_failed] |  | Bun.SQL.close | Drain leases then close with remaining deadline; leave timed-out close owned. | supplied | I35 / P6,P12 |
| sql::sqlite_open_memory |  → sql::pool | [sql::connection_failed] |  | Bun.SQL | Open in-memory SQLite with safeIntegers:true; the database lives while the pool is open. | supplied | B1-02 / B1-02 |
| sql::sqlite_open_file | str path, sql::sqlite_file_options options → sql::pool | [sql::connection_failed] |  | Bun.SQL | Open file SQLite with safeIntegers:true; mode is ro, rw or rwc and busy_timeout_ms bounds lock waits. | supplied | B1-02 / B1-02 |
| sql::mysql_open | str connection_variable, int max_connections → sql::pool | [http::credentials_missing, sql::connection_failed] |  | Bun.SQL | Read selected credential; open MySQL with bigint:true, forced TLS, validated max, and a pinned UTC session. | supplied | B1-03 / B1-03 |
| sql::query_one | P:sql_parameters, R:sql_row; sql::pool handle, str descriptor, P parameters → R; static descriptor | [sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::constraint_failed, sql::row_missing, sql::row_count, sql::schema_mismatch] |  | Bun.SQL tagged template | Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1. | supplied | I35 / P12 |
| sql::query_optional | P:sql_parameters, R:sql_row; sql::pool handle, str descriptor, P parameters → option::value&lt;R&gt;; static descriptor | [sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::constraint_failed, sql::row_count, sql::schema_mismatch] |  | Bun.SQL tagged template | Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1. | supplied | I35 / P12 |
| sql::query_rows | P:sql_parameters, R:sql_row; sql::pool handle, str descriptor, P parameters, int max_rows → R[]; static descriptor | [sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::constraint_failed, sql::row_limit, sql::schema_mismatch] |  | Bun.SQL tagged template | Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1. | supplied | I35 / P12 |
| sql::execute | P:sql_parameters; sql::pool handle, str descriptor, P parameters → int; static descriptor | [sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::constraint_failed] |  | Bun.SQL tagged template | Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1. | supplied | I35 / P12 |
| sql::transaction_query_one | P:sql_parameters, R:sql_row; sql::transaction handle, str descriptor, P parameters → R; static descriptor | [sql::unsupported_value, sql::query_failed, sql::constraint_failed, sql::row_missing, sql::row_count, sql::schema_mismatch] |  | Bun.SQL tagged template | Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1. | supplied | I35 / P12 |
| sql::transaction_query_optional | P:sql_parameters, R:sql_row; sql::transaction handle, str descriptor, P parameters → option::value&lt;R&gt;; static descriptor | [sql::unsupported_value, sql::query_failed, sql::constraint_failed, sql::row_count, sql::schema_mismatch] |  | Bun.SQL tagged template | Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1. | supplied | I35 / P12 |
| sql::transaction_query_rows | P:sql_parameters, R:sql_row; sql::transaction handle, str descriptor, P parameters, int max_rows → R[]; static descriptor | [sql::unsupported_value, sql::query_failed, sql::constraint_failed, sql::row_limit, sql::schema_mismatch] |  | Bun.SQL tagged template | Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1. | supplied | I35 / P12 |
| sql::transaction_execute | P:sql_parameters; sql::transaction handle, str descriptor, P parameters → int; static descriptor | [sql::unsupported_value, sql::query_failed, sql::constraint_failed] |  | Bun.SQL tagged template | Use parser-derived static template segments; validate typed rows and bind server-side LIMIT2/max+1. | supplied | I35 / P12 |
| sql::with_transaction | T:data; sql::pool pool, $callback callback → T | [sql::connection_failed, sql::transaction_failed, sql::commit_unknown] | callback(sql::transaction) → sql::decision&lt;T&gt; emits [] | Bun.SQL.begin | Drain scoped leases; private rollback sentinel; retain commit uncertainty and original standard failures. | scoped | I38 / P6,P12 |
| checks::require | bool condition, str reason → void | [checks::failed] |  | Boolean branch, domain.create | Evaluate condition then reason once each; false produces checks::failed with the exact authored reason. Record the call-site span and invocation path in private occurrence metadata. | real | LF08 / C9.2 |
| files::read_bytes | str path, int max_bytes → bytes::buffer | [files::not_found, files::denied, files::invalid_path, files::unexpected_kind, files::limit_exceeded, files::io_error] |  | Bun.file | Reject empty/NUL paths and negative limits before input; stream Bun.file chunks counting bigint bytes before retaining, copy each native view, cancel and release on overflow; map EISDIR to unexpected_kind; supplied assertion boundary. | supplied | B1-01 / B1-01 |
| files::read_text | str path, int max_bytes → str | [files::not_found, files::denied, files::invalid_path, files::unexpected_kind, files::limit_exceeded, codec::invalid_data, files::io_error] |  | Bun.file, TextDecoder | Bounded read_bytes then fatal UTF-8 decode; undecodable input is codec::invalid_data with the read path; supplied assertion boundary. | supplied | B1-01 / B1-01 |
| files::write_bytes | str path, bytes::buffer value, bool overwrite → void | [files::not_found, files::already_exists, files::denied, files::invalid_path, files::io_error] |  | node:fs/promises.writeFile | Copy Can bytes out; write with flag wx when overwrite is false so exclusive creation is atomic; never auto-create missing parents; supplied assertion boundary. | supplied | B1-01 / B1-01 |
| files::write_text | str path, str value, bool overwrite → void | [files::not_found, files::already_exists, files::denied, files::invalid_path, files::io_error] |  | TextEncoder, node:fs/promises.writeFile | Encode UTF-8 then the write_bytes contract; supplied assertion boundary. | supplied | B1-01 / B1-01 |
| files::stat | str path, bool follow_symlinks → files::file_info | [files::not_found, files::denied, files::invalid_path, files::io_error] |  | node:fs/promises.stat, node:fs/promises.lstat | Use stat when following and lstat otherwise; project kind file/directory/symlink/other with exact bigint size; supplied assertion boundary. | supplied | B1-01 / B1-01 |
| files::exists | str path → bool | [files::denied, files::invalid_path, files::io_error] |  | node:fs/promises.lstat | Report true for any entry kind including dangling symlinks; only missing paths report false, never permission failures; supplied assertion boundary. | supplied | B1-01 / B1-01 |
| files::list | str path, int max_entries → files::entry[] | [files::not_found, files::denied, files::invalid_path, files::unexpected_kind, files::limit_exceeded, files::io_error] |  | node:fs/promises.readdir, node:path.resolve, node:path.join | Read typed entries once, resolve each child to an absolute path, sort lexically, and reject over-limit directories instead of truncating; supplied assertion boundary. | supplied | B1-01 / B1-01 |
| files::mkdir | str path, bool recursive → void | [files::not_found, files::already_exists, files::denied, files::invalid_path, files::io_error] |  | node:fs/promises.mkdir | Create one directory or a recursive chain; an existing path fails only when recursive is false; supplied assertion boundary. | supplied | B1-01 / B1-01 |
| files::copy | str source, str destination, bool overwrite → void | [files::not_found, files::already_exists, files::denied, files::invalid_path, files::unexpected_kind, files::io_error] |  | node:fs/promises.copyFile | Copy bytes with COPYFILE_EXCL unless overwrite; attribute missing-path failures to the absent side best-effort; supplied assertion boundary. | supplied | B1-01 / B1-01 |
| files::move | str source, str destination, bool overwrite → void | [files::not_found, files::already_exists, files::denied, files::invalid_path, files::unexpected_kind, files::not_empty, files::cross_device, files::io_error] |  | node:fs/promises.rename | Rename without copy fallback; cross-device moves fail explicitly and never silently lose atomicity; supplied assertion boundary. | supplied | B1-01 / B1-01 |
| files::remove | str path, bool recursive → void | [files::not_found, files::denied, files::invalid_path, files::not_empty, files::io_error] |  | node:fs/promises.rm, node:fs/promises.rmdir, node:fs/promises.unlink | Remove one file/symlink/empty directory, or a recursive tree only when requested; a non-empty directory without recursion is not_empty; supplied assertion boundary. | supplied | B1-01 / B1-01 |
| files::glob | str base, str pattern, bool follow_symlinks, int max_entries → str[] | [files::not_found, files::denied, files::invalid_path, files::limit_exceeded, files::io_error] |  | Bun.Glob | Enumerate natively with the entry cap enforced during iteration, return absolute sorted paths including directories, and never silently truncate; supplied assertion boundary. | supplied | B1-01 / B1-01 |
| path::resolve | str base, str[] parts → str | [] |  | node:path.resolve | Resolve parts against the base with native normalization; pure computation. | real | B1-01 / B1-01 |
| path::join | str[] parts → str | [] |  | node:path.join | Join segments with native normalization; pure computation. | real | B1-01 / B1-01 |
| path::basename | str path → str | [] |  | node:path.basename | Return the final segment natively; pure computation. | real | B1-01 / B1-01 |
| path::extension | str path → str | [] |  | node:path.extname | Return the native extension including the leading dot, or empty; pure computation. | real | B1-01 / B1-01 |
| process::run | str executable, str[] args, process::options options → process::result | [files::not_found, files::denied, process::spawn_failed, process::timeout, process::output_limit, process::invalid_config, process::io_error] |  | Bun.spawn | Spawn detached in its own process group with piped stdio, no shell; drain both streams concurrently under caps, enforce the deadline, and terminate the group SIGTERM-then-SIGKILL with a grace before escalation; reap every child and register the run as an owned resource so scope drain kills survivors; supplied assertion boundary. | supplied | B1-04 / B1-04 |
| process::require_success | process::result value → process::result | [process::nonzero] |  | domain.create | Return the result unchanged when it exited zero, else nonzero with the observed code and signal; pure computation. | real | B1-04 / B1-04 |
| process::which | str name → str | [files::not_found, process::invalid_config] |  | Bun.which | Resolve the executable natively; an unresolvable name is files::not_found and an empty name is invalid_config; supplied assertion boundary. | supplied | B1-04 / B1-04 |
| stream::read_many | T:data; stream::reader&lt;T&gt; reader, int max_items → T[] | [stream::read_failed, stream::cancelled, files::limit_exceeded] |  | ReadableStreamDefaultReader.read | One native pull per batch step with no prefetch queue; empty batch is the normal end; bytes items split at max_chunk with copied views, text items \n-framed with fatal UTF-8; interrupted reads report cancelled and deliver nothing; failures are terminal; use-after-close/foreign-owner throw standard resource-state. | supplied | B1-05 / B1-05 |
| stream::write_some | stream::writer writer, bytes::buffer chunk → int | [stream::write_failed] |  | FileSink.write | Copy payload bytes out of immutable values, report accepted count, leave short-write retries to the caller; no coalescing queue. | supplied | B1-05 / B1-05 |
| stream::close_reader | T:data; stream::reader&lt;T&gt; reader → void | [stream::close_failed] |  | ReadableStreamDefaultReader.cancel | Terminal owner close with bounded shutdown; cancels the native reader and releases the lock exactly once; close twice throws standard resource-state. | supplied | B1-05 / B1-05 |
| stream::close_writer | stream::writer writer → void | [stream::close_failed] |  | FileSink.end | Terminal owner close with bounded shutdown; flushes and ends the sink exactly once; close twice throws standard resource-state. | supplied | B1-05 / B1-05 |
| stream::cancel_reader | T:data; stream::reader&lt;T&gt; reader, str reason → void | [stream::close_failed] |  | ReadableStreamDefaultReader.cancel | Record the reason, then terminal owner close; an in-flight read reports cancelled instead of partial items. | supplied | B1-05 / B1-05 |
| stream::cancel_writer | stream::writer writer, str reason → void | [stream::close_failed] |  | FileSink.end | Record the reason, then terminal owner close; racing writes complete under their lease. | supplied | B1-05 / B1-05 |
| files::read_stream | str path, int max_chunk → stream::reader&lt;bytes::buffer&gt; | [files::not_found, files::denied, files::invalid_path, files::unexpected_kind, files::limit_exceeded, files::io_error] |  | Bun.file | Stat at open for acquisition errors, then lazy streaming pulls; later filesystem changes fail reads, not the open. | supplied | B1-05 / B1-05 |
| files::read_lines_stream | str path, int max_line → stream::reader&lt;str&gt; | [files::not_found, files::denied, files::invalid_path, files::unexpected_kind, files::limit_exceeded, files::io_error] |  | Bun.file, TextDecoder | Stat at open for acquisition errors, then fatal streaming UTF-8 decode with \n framing, CR tolerance, trailing segment delivery and line caps. | supplied | B1-05 / B1-05 |
| files::write_stream | str path → stream::writer | [files::not_found, files::denied, files::invalid_path, files::unexpected_kind, files::io_error] |  | FileSink | Create or truncate at open with acquisition errors, then lazy accepted-count writes; write-after-end is unreachable through owner close. | supplied | B1-05 / B1-05 |
| url::parse | str text → url::parts | [url::invalid_url] |  | URL | Parse absolute http/https URLs into immutable part records; other schemes and malformed text reject, userinfo never projects; execute in ordinary assertions. | real | B1-13 / B1-13 |
| url::resolve | str base, str input → url::parts | [url::invalid_url] |  | URL | Resolve relative references against absolute http/https bases per WHATWG URL; execute in ordinary assertions. | real | B1-13 / B1-13 |
| url::to_string | url::parts url → str | [url::invalid_url] |  | URL | Serialize part records back to href form; execute in ordinary assertions. | real | B1-13 / B1-13 |
| url::query_all | url::parts url, str name → str[] | [] |  | URLSearchParams | Read every form-decoded value for one query key in document order; execute in ordinary assertions. | real | B1-13 / B1-13 |
| url::query_pairs | url::parts url → url::query_pair[] | [] |  | URLSearchParams | Project every form-decoded query pair in document order, duplicates kept; execute in ordinary assertions. | real | B1-13 / B1-13 |
| url::with_query | url::parts url, url::query_pair[] pairs → url::parts | [url::invalid_url] |  | URL, URLSearchParams | Rebuild the query string from ordered pairs with form encoding, keeping fragment and parts; execute in ordinary assertions. | real | B1-13 / B1-13 |
| time::instant_from_epoch_millis | int millis → time::instant | [time::out_of_range] |  | Date | Admit epoch milliseconds inside the native Date span as opaque instants; execute in ordinary assertions. | real | B1-13 / B1-13 |
| time::instant_epoch_millis | time::instant instant → int | [] |  | BigInt | Project the exact epoch milliseconds from an instant; execute in ordinary assertions. | real | B1-13 / B1-13 |
| time::format_in_zone | time::instant instant, str locale, str zone, str date_style, str time_style → str | [time::invalid_zone, time::invalid_option] |  | Intl.DateTimeFormat | Format with explicit locale, IANA zone, and full/long/medium/short/none styles; execute in ordinary assertions. | real | B1-13 / B1-13 |
| time::resolve_zoned_time | time::civil civil, str zone, int policy → time::instant | [time::invalid_zone, time::nonexistent_time, time::invalid_option] |  | Date, Intl.DateTimeFormat | Resolve civil time in a zone with explicit earlier(0)/later(1) DST policy; gaps reject after round-trip verification; execute in ordinary assertions. | real | B1-13 / B1-13 |
| ws::connect | str url, str[] protocols, int max_message_bytes, int max_queued_events, int max_send_bytes, int deadline_ms, bool insecure_tls → ws::connection | [ws::connect_failed, ws::invalid_url, ws::invalid_protocol, ws::limit_exceeded] |  | WebSocket | Open a client session, wait for the handshake under the caller deadline, and vend the session with its event reader. | supplied | B1-07 / B1-07 |
| ws::accept | http::request request, str protocol, int max_message_bytes, int max_queued_events, int max_send_bytes → ws::connection | [ws::upgrade_failed, ws::unsupported_protocol, ws::invalid_protocol, ws::limit_exceeded] |  | Bun.serve, Request | Upgrade a routed request after handler authentication and vend the session with its event reader. | supplied | B1-07 / B1-07 |
| ws::send_text | ws::session session, str text → int | [ws::send_failed] |  | WebSocket, Bun.serve | Queue one text message; server saturation fails so the caller retries after drain. | supplied | B1-07 / B1-07 |
| ws::send_bytes | ws::session session, bytes::buffer data → int | [ws::send_failed] |  | WebSocket, Bun.serve | Queue one binary message; server saturation fails so the caller retries after drain. | supplied | B1-07 / B1-07 |
| ws::close | ws::session session, int code, str reason → void | [ws::invalid_close] |  | WebSocket, Bun.serve | Validate the close code and reason on both sides, send the frame, and terminally close the session. | supplied | B1-07 / B1-07 |
| cookie::parse | str header → cookie::collection | [] |  | CookieMap | Parse a Cookie header into first-wins lookup over the retained ordered pair list. | real | B1-09 / B1-09 |
| cookie::get | cookie::collection collection, str name → option::value&lt;str&gt; | [] |  | CookieMap | Return the first pair value for the name, or none when absent. | real | B1-09 / B1-09 |
| cookie::make | str name, str value, cookie::attributes attributes → cookie::cookie | [cookie::invalid_cookie] |  | Cookie | Validate the name and expiry, then wrap a native cookie. | real | B1-09 / B1-09 |
| cookie::serialize | cookie::cookie cookie → str | [] |  | Cookie | Render one Set-Cookie field value from a validated cookie. | real | B1-09 / B1-09 |
| cookie::expire | str name, str path, option::value&lt;str&gt; domain → cookie::cookie | [cookie::invalid_cookie] |  | CookieMap | Build an epoch-expiry tombstone scoped to the matching path and domain. | real | B1-09 / B1-09 |
| csrf::generate | str secret, str session_id, int expires_in_ms → str | [csrf::invalid_config] |  | CSRF | Mint a session-bound token with explicit secret and fixed base64url/sha256. | supplied | B1-09 / B1-09 |
| csrf::verify | str secret, str session_id, str token, int max_age_ms → bool | [csrf::invalid_config] |  | CSRF | Verify a token against explicit secret, session and age; token faults answer false. | real | B1-09 / B1-09 |

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
| judge | http::request_failed, ai::invalid_question, ai::invalid_answer |  | questions, handlers, continuation | LF10 |
| fetch_body | http::request_failed |  |  | LF10 |
| fetch_envelope | http::request_failed |  |  | LF10 |
| llm | http::invalid_request, http::transport_failed, http::timeout, http::body_limit, http::status_error, codec::invalid_data, llm::refused, llm::truncated, llm::invalid_response | authenticated: http::credentials_missing |  | I28 |

Standard failure categories: arithmetic, bounds, resource_state, assertion, native_exception, cleanup.

Property descriptors (array.length, str.length, bytes.length) require no
call marker. Method descriptor names such as array.map are internal lookup
keys, not new reserved source packages. Package names cli and json remain
reserved even though this slice declares no callable members in them.
