# B1-13 common-utilities contract confirmation

Driver: Bun 1.4.2 (`URL`, `URLSearchParams`, `RegExp`, `Buffer`,
`Date`, `Intl`). Every row below is an executed observation from the
B1-13 probes or the unit suites unless marked as policy.

## URL

`url::parse` admits absolute http/https only; other schemes read
`invalid_url` with reason `scheme`, malformed text with `syntax`.
Userinfo never projects: `https://user:pass@h/p` parses with host `h`
and serializes back without credentials. Parts are immutable records
(scheme, host, port, path, query, fragment) with port 0 when absent,
including default ports. `url::resolve` follows WHATWG relative
resolution against absolute bases.

Query work uses form decoding (`application/x-www-form-urlencoded`):
`+` reads as a space, which differs deliberately from path
percent-decoding. Repeated keys keep document order, `?a` and `?a=`
both read as `("a", "")`, and `%E2%82%AC` reads as `€`.
`url::with_query` rebuilds from ordered pairs with form encoding
(space writes as `+`), keeping the fragment. `new URL("http:foo")`
parses (as `http://foo/`), so the syntax tests use verified-bad
inputs such as `http://[::1` and `http://`.

## Regex

`text::compile_regex` admits flags `i m s u v`; `g`, `y`, `d`, and
unknown flags reject with reason `flags`, while bad patterns and bad
combinations (such as `u` with `v`) reject with `syntax`.
`text::matches` runs a fresh global pass per call, so no shared
`lastIndex` can leak between scans. Start/end are UTF-16 code units,
consistent with `str.slice`/`str.length` (an astral `.` match spans
0..2). Empty matches advance one unit per hit and terminate; absent
captures read as `""`. The caller limit admits 0..10000 and caps only
the result count: catastrophic patterns still burn CPU per match, and
regex deadlines are not promised.

## Byte encodings

Base64 is the standard alphabet with required padding (`length % 4 ==
0`, `=` only at the end, no base64url); hex requires even length over
`0-9a-fA-F` (uppercase decodes, encoding emits lowercase). The gates
exist because `Buffer` is lenient: it skips whitespace, decodes
unpadded `QQ`, and truncates hex at the first invalid pair. Malformed
input reads `codec::invalid_data` with reason `base64`/`hex`, never a
truncation. Empty text decodes to empty bytes on both paths.

## Time

An instant is an opaque handle over exact epoch milliseconds,
range-checked to ±8.64e15 (the native `Date` span; one past either end
reads `Invalid Date` natively) before any `Number` conversion.
`time::format_in_zone` takes explicit locale, IANA zone, and
`full/long/medium/short/none` styles with the locale-default hour
cycle on the pinned ICU: `en-US`/`Asia/Tokyo` medium/short over
2026-09-23T12:00Z reads `Sep 23, 2026 at 9:00 PM`. Unknown zones read
`time::invalid_zone`; bad styles, structurally-invalid locales, bad
policies, and out-of-domain civil parts read `time::invalid_option`.
There is no machine-local zone anywhere: every zone is explicit.

Civil resolution takes an explicit earlier(0)/later(1) DST policy.
The adapter collects the distinct offsets in a ±24h window (any
transition that could affect the civil time falls inside), builds one
candidate per offset, and verifies each by formatting back: a
candidate that does not round-trip is a gap, never silently kept.
Verified on America/New_York 2026: `2026-03-08 02:30` reads
`time::nonexistent_time`; `2026-11-01 01:30` resolves to 05:30Z under
policy 0 and 06:30Z under policy 1. Civil years are literal
(`Date.UTC` maps 0..99 to 19xx, so the adapter corrects), and
milliseconds travel outside the second-precision offset arithmetic
(the truncated formatter reading would otherwise skew sub-second
inputs by their own millisecond field). Durations stay plain int
millis: no dedicated type or arithmetic is in scope.

## Fixtures

All sixteen operations are real: every assertion executes natively
with fixed vectors, and no B1-13 operation is a supplied boundary.
