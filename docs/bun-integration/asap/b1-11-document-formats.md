# B1-11 — Typed TOML, YAML, JSON5 and JSONL codecs

Status: prepared for ASAP execution; implementation and acceptance still pending. Dependencies: B1-05. Surface: Library.

Expose typed TOML, YAML, JSON5 and JSONL using Bun parsers, preserving honest numeric/duplicate behavior. Bounded parse APIs can be built before B1-05; incremental JSONL completes after streams. Native YAML/JSON5/JSONL rounded an unsafe integer in the probe, TOML rejected it, and JSONL.parse silently returned a valid prefix. Never advertise ordinary parse-then-project as lossless.

## Where to implement

Paths are repository-relative; **new** means a proposed file/directory, not existing code. The shared wiring below is part of this task, not optional cleanup.

| Path | State | Responsibility |
|---|---|---|
| `runtime/codec/formats.ts` | new | Native format dispatch and typed immutable projection. |
| `runtime/codec/jsonl.ts` | new | Strict full-input JSONL and incremental framing. |
| `runtime/codec/json.ts` | existing | Reuse schema semantics without weakening exact JSON token handling. |
| `runtime/codec/budget.ts` | existing | Shared byte/node/depth/record caps. |
| `compiler/internal/check/codec.go` | existing | Admitted schema types and format-specific restrictions. |
| `runtime/test/formats.test.ts` | new | Precision, duplicate and malformed documents. |
| `tests/integration/formats_test.go` | new | Typed parse programs and native lowering. |
| `compiler/internal/catalogue/catalogue.json` | existing | Canonical public types, signatures, exact error identities, native lowering and assertion policy. |
| `compiler/internal/emit/program.go` | existing | Operation identity -> runtime target binding, imports and adapter construction. |
| `runtime/modules.json` | existing | Register each runtime module and exact import edges. |
| `distribution/target.json` | existing | Add actually required qualified native APIs; do not change the pinned runtime casually. |

Generated mirrors are produced with `make catalogue`: `compiler/internal/catalogue/generated.go`, `runtime/catalogue.ts`, `std/catalogue/README.md`. Never hand-edit them. See [shared integration contract](integration-contract.md) for specs, module boundaries, fixture ownership and completion rules.

## Proposed API contract

This is interface notation, **not accepted Can syntax or a claim that these operations compile today**. Resolve final names against the current catalogue and express them in existing Can declaration syntax; only primitive gates may add grammar.

```text
toml::decode<T>(text, limits) -> T
 yaml::decode<T>(text, limits) -> T
json5::decode<T>(text, limits) -> T
jsonl::decode<T>(bytes, limits) -> [T]
jsonl::consume<T>(byte_stream, handler, limits) -> terminal_result
format-specific native semantics are explicit; exact JSON keeps its existing contract
```

## Implementation sequence

1. Define a format matrix: accepted scalar kinds, null/optional mapping, dates/times, duplicate keys, aliases/merges, multiple documents, nonfinite floats, integers and root shapes. Use native options only when verified; default behavior is not uniform across parsers.
2. Create typed projection sharing Can schema rules and immutable data constructors, with per-format scalar conversion policy. Native numbers used as ints must be finite safe integers; document that this is parsed-value semantics, not proof of every original fractional token.
3. Keep the existing JSON exact-integer decoder intact. For strict JSONL, prefer bounded line framing plus the existing native-backed exact JSON decode path per record when that satisfies the selected JSONL contract. Framing lines is not implementing JSON grammar.
4. If using Bun.JSONL.parseChunk, inspect error, read and done and reject malformed/truncated final input. Never use Bun.JSONL.parse prefix success as complete validation. Specify blank lines, CRLF, final line without newline and multiline JSON acceptance explicitly.
5. Incrementally decode UTF-8 across chunk boundaries; cap bytes per record, record count and total retained output. Await each typed handler. Retain only the incomplete suffix, not every prior chunk.
6. For YAML/TOML native special objects (dates, aliases, maps), project only admitted forms and reject cycles/unsupported values. Node/depth checks after parsing do not protect native parse CPU/memory; a preparse byte cap is still necessary.
7. Test duplicate behavior empirically. If exact duplicate rejection is promised but native metadata cannot support it, keep that mode explicitly blocked rather than writing an ad hoc YAML/JSON5 grammar.
8. Add declared syntax/type/limit errors with safe paths and native source locations when trustworthy. Preserve source offsets through framing.

## Acceptance evidence

- Success: typed configuration from each format, nested records/arrays, strict JSONL records split across arbitrary UTF-8 chunks.
- Rejected: malformed suffix after valid prefix, unsafe int, nonfinite float, wrong schema field, unsupported native date/cycle, excessive depth/record bytes.
- Runtime: 9007199254740993 regression for every format; exact JSONL path preserves it if selected; duplicates documented; truncated last record fails; stream cancellation stops reads.

## Fixtures and local assertions

Parsing is real computation. Fixtures supply input byte streams only, and handlers remain locally tested. Compare bounded and incremental decoding over every split point of a short Unicode document. Store native parser failures as observations, not successful Can acceptance.

## Native lowering sketch

This sketch describes the native operation, not a complete Can adapter. Real adapters must return the existing Completion representation, retain source origin and apply declared error boundaries.

```ts
const parsed = Bun.YAML.parse(text); // or TOML / JSON5
return projectNativeDocument(schema, parsed, limits, formatPolicy);
// JSONL: native-backed exact JSON decoder per bounded record, or checked parseChunk.
// Never stringify a rounded parsed number and claim its original spelling was recovered.
```

## Gates and limitations

Numeric exactness is a contract choice, not a postprocessing trick. A target parser losing source information may require a narrower accepted type/profile. Make limitations visible in API docs and tests; do not reduce the entire format capability to a silent TODO.

## Source and proof

- [Pinned native observations](evidence/native-probe-results.json) and [reproducer](evidence/native-probe.ts). Only listed probe behavior was executed.
- [Official Bun documentation](https://bun.sh/docs/runtime/toml)
- [Official Bun documentation](https://bun.sh/docs/runtime/yaml)
- [Official Bun documentation](https://bun.sh/docs/runtime/json5)
- [Official Bun documentation](https://bun.sh/docs/runtime/jsonl)

Do not check off this capability until its acceptance cases, local fixture behavior, generated TypeScript evidence and all required real-native runs are recorded. Use [execution queue](execution-queue.md) to continue immediately with the next unblocked capability.
