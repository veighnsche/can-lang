# Current exact JSON codec

JSON is the exact typed codec, not a value-AST library:
`codec::encode_json<T>` and `codec::decode_json<T>` over compiler
schema descriptors, with `bytes::from_utf8` and `bytes::to_utf8` at
the text boundary. Exact numbers, duplicate rejection, and shared
finite budgets are enforced natively. Fuel parsers, `Json__Value`
trees, monomorphic schema families, and standalone hex/base64
codecs are excluded. Current demonstrations live in the `codec`
fixtures and the admitted applications (native-ai encodes its typed
report; account-search decodes forms at the server boundary).

The adjacent legacy source and generated files below are historical
migration inputs scheduled for retirement by I43/I44, not the current
implementation.

## Historical implementation

# json — JSON value AST, render, scalar codecs, schemas

- `json.can` — `mod json`: `Json__Value` is the parsed tree
  (`Null`/`Bool`/`Num`/`Str`/`Arr`/`Obj`; `Num` carries the wire
  text verbatim, never a lossy float). `std__json__render_value`
  prints any tree through a single self-recursive frame machine:
  only direct self-recursion is admitted, so the natural
  value/fields/array mutual recursion is inexpressible and an
  explicit `Seq<Json__Frame>` stack carries the pending work
  (tag-dispatched records — variant sequences are not admitted;
  stack top is the back since sequence concatenation is not in
  v1). Each step burns one fuel from `std__json__RENDER_FUEL`
  (1M); exhaustion raises payloadless
  `json.render_budget_exhausted`. `std__json__escape` handles
  `"`/`\`/TAB/LF/CR plus `\u00XX` for other C0 controls via an
  armless hex table. Typed codecs bridge scalars to `Json__Doc`
  (`int`/`str`/`bool`/`dec` × encode/decode); decodes reject
  mistyped trees with `json.schema_mismatch` and bad numeric
  text with `json.numeric_out_of_range` (translated from
  `convert.*`). Schemas are a monomorphic family (JEV 0.74 over
  generic-uniform): `Json__IntSchema` & co. carry exact
  encode/decode callbacks invoked through bare-name apply
  wrappers (`invoke` heads must be bare names).
- `json.ts` + `errors.json` — committed golden TS prod emit
  (tests stripped). Regenerate: `go run ./compiler --out
  /tmp/jg std/json/json.can std/scalars/scalars.can`, copy
  `json.ts` + `errors.json`; verify: `go test ./...`.

Rules: `/REQUIREMENTS.md`. Program: `docs/ASTRA_STDLIB.md` §1.8.
Byte-level parse landed in S11b: `std__json__parse_value`
guards empty input, then `std__json__parse_step` runs a
single self-recursive 12-state machine over `Seq<Json__PFrame>`
(empty dispatch + `ArrFirst`/`ArrVal`/`ArrNext` + `ObjFirst`/
`ObjKey`/`ObjColon`/`ObjKeyVal`/`ObjNext` + `StrKey`/`StrVal`/
`NumAcc` + `Tail`). Value states push nested frames; key and
punctuation continuations replace the top frame (a lingering
`ObjFirst` under `StrKey` breaks `parse_attach`, found
in-slice). `StrKey` inherits the parent's fields/keys for
duplicate detection; numbers accumulate raw and validate
through the 9-state numcheck; `Tail` rejects trailing values.
Text drivers landed in S12: per-scalar `encode_text`
(value → schema Doc → render) and `decode_text`
(parse → schema Doc decode) for `int`/`str`/`bool`/`dec`,
reusing the S9 `given` exchanges by test name (`frac`/`exp`
rows added for the int/dec out-of-range paths).
