# B1-13 — Common URL, text, byte and time utilities

Status: prepared for ASAP execution; implementation and acceptance still pending. Dependencies: none. Surface: Library.

Fill actual gaps in URL/query, text/regex, byte encodings and time. Existing Can already has substantial text, bytes, clocks and number APIs: extend those instead of creating duplicate spellings. Keep exact integers, Unicode behavior and instant-versus-calendar distinctions explicit.

## Where to implement

Paths are repository-relative; **new** means a proposed file/directory, not existing code. The shared wiring below is part of this task, not optional cleanup.

| Path | State | Responsibility |
|---|---|---|
| `runtime/platform/url.ts` | new | URL and ordered duplicate-preserving query operations. |
| `runtime/text.ts` | existing | Only missing regex/text operations. |
| `runtime/bytes.ts` | existing | Immutable encoding/decoding bridges where appropriate. |
| `runtime/platform/datetime.ts` | new | Native Date/Intl projection and timezone operations. |
| `runtime/platform/clock.ts` | existing | Reuse existing wall/monotonic/sleep boundaries. |
| `tests/integration/utilities_test.go` | existing | Missing capability acceptance. |
| `tests/integration/text_test.go` | existing | Unicode/regex offsets. |
| `tests/integration/bytes_test.go` | existing | Encoding error and alias cases. |
| `compiler/internal/catalogue/catalogue.json` | existing | Canonical public types, signatures, exact error identities, native lowering and assertion policy. |
| `compiler/internal/emit/program.go` | existing | Operation identity -> runtime target binding, imports and adapter construction. |
| `runtime/modules.json` | existing | Register each runtime module and exact import edges. |
| `distribution/target.json` | existing | Add actually required qualified native APIs; do not change the pinned runtime casually. |

Generated mirrors are produced with `make catalogue`: `compiler/internal/catalogue/generated.go`, `runtime/catalogue.ts`, `std/catalogue/README.md`. Never hand-edit them. See [shared integration contract](integration-contract.md) for specs, module boundaries, fixture ownership and completion rules.

## Proposed API contract

This is interface notation, **not accepted Can syntax or a claim that these operations compile today**. Resolve final names against the current catalogue and express them in existing Can declaration syntax; only primitive gates may add grammar.

```text
url::parse(text, base?) -> url; url::query_all(url, name) -> [str]
url::with_query(url, ordered_pairs) -> url
text::regex(pattern, explicit_flags) -> opaque_regex
text::matches(regex, text, limit) -> [match]
bytes::decode_utf8(bytes, strict); encode_utf8(text)
bytes::encode/decode_base64; encode/decode_hex
 time::instant_from_epoch_millis(int); format_instant(instant)
 time::format_in_zone(instant, zone, locale, options)
```

## Implementation sequence

1. Diff desired operations against the current catalogue and list omissions before adding names. Reuse existing normalize/scalars/graphemes/join and clock APIs.
2. Use URL/URLSearchParams with typed immutable projections. Preserve repeated query keys and distinguish URL decoding from form decoding. Define relative URL resolution and admitted schemes.
3. Use native RegExp; reset or hide lastIndex and mutable state. Specify flags, capture absence, Unicode index units, zero-length match advancement and max result count.
4. Use native TextEncoder/TextDecoder and qualified Buffer/Web byte encoders. Reject malformed hex/base64 under the documented strict policy instead of silently truncating. Copy mutable views at both edges.
5. Represent an instant separately from local date/time and duration. Range-check exact Can int before Number conversion for Date. Use Intl for explicit locale/zone formatting; deterministic tests specify both.
6. Define DST ambiguity/nonexistence if local-time-to-instant conversion is exposed. Do not quietly pick native normalization as the Can contract. Restrict initial conversion to well-defined instant/offset inputs until this is specified.
7. Add known vectors and negative compiler examples. Keep wall time/randomness in their existing assertion boundary; pure parsing/formatting remains real.

## Acceptance evidence

- Success: duplicate query values, relative URL resolution, escaped Unicode, strict UTF-8, encoding vectors, epoch and timezone formatting.
- Rejected: malformed URL/regex/encoding, unsafe date range, unsupported flags, invalid timezone and ambiguous local conversion without policy.
- Runtime: regex zero-length progression, native UTF-16 offsets converted or documented consistently, deterministic locale formatting, byte alias mutation prevented.
- Catalogue inventory proves existing functions were reused rather than renamed with compatibility aliases.

## Fixtures and local assertions

Pure operations execute natively inside local tests. Supply explicit timezone/locale and fixed instants. Test DST transitions on the pinned ICU/runtime and record target sensitivity. Never snapshot the machine-local current date.

## Native lowering sketch

This sketch describes the native operation, not a complete Can adapter. Real adapters must return the existing Completion representation, retain source origin and apply declared error boundaries.

```ts
const url = new URL(text, base);
const values = url.searchParams.getAll(name);
const regex = new RegExp(pattern, flags);
const date = new Date(checkedEpochMilliseconds);
const formatted = new Intl.DateTimeFormat(locale, options).format(date);
```

## Gates and limitations

Native RegExp can consume substantial CPU; a result-count limit does not bound matching time. Admit that limitation or use an enforceable isolation strategy before claiming regex deadlines. Do not implement a new regex or calendar engine.

## Source and proof

- [Pinned native observations](evidence/native-probe-results.json) and [reproducer](evidence/native-probe.ts). Only listed probe behavior was executed.
- [Official Bun documentation](https://bun.sh/docs/runtime/web-apis)
- [Official Bun documentation](https://bun.sh/docs/runtime/binary-data)
- [Official Bun documentation](https://bun.sh/docs/runtime/utils)

Do not check off this capability until its acceptance cases, local fixture behavior, generated TypeScript evidence and all required real-native runs are recorded. Use [execution queue](execution-queue.md) to continue immediately with the next unblocked capability.
