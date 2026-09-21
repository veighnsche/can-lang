# I26 validation — named fetch modes and immutable envelopes

Named fetch declarations now retain checked IR for ordinary inputs, method/path,
ordered query/header entries, optional explicit request encoding and concrete
response mode/schema/envelope identity. Emission follows the existing native
phase-plan machinery: evaluate each descriptor once in written order, preserve
ordinary callable/fixture handling, then invoke the maintained transport adapter.
No authored backend hooks or fallback transport are introduced.

The finite GET/HEAD/POST/PUT/PATCH/DELETE/OPTIONS inventory is supported. Existing
syntax rejects GET/HEAD bodies; checked signatures reject unsupported response
roots and JSON-record HEAD. JSON bodies use the shared exact typed codec, text
uses UTF-8 without a BOM, and bytes use a defensive private-buffer copy. Effective
Content-Type defaults and request-over-default replacement are applied before
authentication. Statically known JSON conflicts reject; dynamic conflicts produce
invalid_request(content_type). Outgoing size failures map to body_limit.

Results decode as typed ordinary JSON records, strict UTF-8 text or opaque bytes.
JSON requires application/json or a +json subtype; text accepts absent media type;
both reject non-UTF-8 charset and malformed/ambiguous media metadata. Text preserves
an initial BOM, while the shared JSON codec rejects it. Byte mode bypasses media
checks. Decoder errors retain codec::invalid_data path/reason. Body-only non-2xx
status fails before decoding; envelopes retain final 200–599 statuses and contain
frozen nominal status/headers/body data. Header snapshots preserve separate
Set-Cookie entries and native normalized combination for other repeated names.

The existing deadline, bounded consumption, no-redirect, one-attempt and ownership
transport machinery remains shared. Credential lookup occurs once per launched
request. Assertion execution uses raw HTTP fixtures or rejects a live boundary
before authentication; bodyless raw fixtures now compare an absent body with the
expected zero-byte wire body. Ordinary supplied completions remain consumer-only
evidence.

Validation evidence:

- Six named-fetch runtime tests, 92 expectations: loopback exact integers beyond
  binary64, text BOM, byte envelopes, repeated headers, explicit encodings,
  credential lookup count, query encoding, strict media/charset, malformed UTF-8,
  empty/duplicate JSON, request limits and dynamic conflicts. Raw fixture status
  probes cover 200, 299, 300, 418 and 599; missing fixtures reject before auth.
- Runtime adapter/test strict TypeScript checking passed.
- Compiler fixture retains all eight fetch plans, all seven methods and callable
  references. Nine negative mutations reject invalid method/body/root/header
  modes, numeric query values, missing bounds and static Content-Type conflicts.
- Staged CLI loopback execution makes eight requests across all seven methods and
  three body encodings. Server checks exact JSON digits, bytes, UTF-8, headers,
  repeated query values and synthetic authentication. Bad status/media/charset/
  record payloads expose the exact allocated error IDs (diagnostic payloads stay
  redacted). Assertions are network-denied and generated TypeScript is strict.
- Full runtime suite passed: 181 tests, 19,375 expectations, zero failures.

Source fixture: `compiler/testdata/current/fetch/main.can`. Native wire behavior
is tested separately from supplied-completion consumer assertions. Staged
transport qualification now includes the new named-fetch runtime cases.

The full compiler and integration gate passed with pinned archive and CAN_TSC:
`go test ./compiler/... ./tests/integration -count=1`. The final staged fetch
project passed all three offline assertion roots (including a callable reference),
main execution, negative live-response probes and strict generated TypeScript.
The staged runtime transport suite also passed in the loopback-only sandbox.
