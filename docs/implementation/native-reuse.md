# Native reuse and custom-component budget

Stage 1 inventory, carried into stages 3–5. “New” means new Can glue, never permission to recreate the upstream facility. Every runtime helper needs a row here (or a reviewed addition naming a precise contract mismatch). References C/Q/A/P refer to the current companion specifications linked in [research](research.md).

| Capability | Upstream operation/library | Exact Can adapter and reason | Existing disposition / task |
|---|---|---|---|
| Parse Can | Go text/Unicode utilities; Can-specific lexer/parser | C2 indentation, contextual keywords, raw/triple strings, nested comments and sections have no compatible upstream JS grammar | Replace parse/scan assumptions; I03–I04 |
| Bind/type/lower Can | Go maps/graphs; compiler-owned typed IR | C3–C5 nominal identities, region ownership, expected types, generics and finite errors are language-specific | Adapt algorithms only; I05–I10 |
| Integer/float/bool | JS bigint/Number/Math/operators | Checked bigint indices, zero/fault classification, no mixed coercion; native signed zero and NaN contracts | Delete dec arithmetic/prover kernels; I07,I22 |
| Equality/data | Object.is; Bun.deepEquals(strict); native object/array copies | Canonical nominal tags; callable/opaque exclusion; immutable ownership | Replace old Value equality; I06,I07 |
| Exact quantities | Native bigint `/`, `%`, comparisons | C8 Euclidean sign adjustment and finite half-even rounding; no new arithmetic engine | Replace ratio/division fuel code; I23 |
| Text | Native strings, String, replaceAll, join | Bigint-to-index checks, literal replacement function, empty separator/pattern errors | Replace text loops and formatting kernels; I22,I24 |
| Unicode | String iterator/codePointAt/fromCodePoint, Intl.Segmenter, normalize | Validate scalar strings; chunk bulk arguments; pin ICU behavior | No Unicode tables/segmentation implementation; I24 |
| Arrays | slice/concat/spread/toReversed/fromAsync/reduce/filter/toSorted | Indices avoid thenable input assimilation; callback boxes, ordered decisions/keys, short-circuit await bridge | Replace seq recursion; I21 |
| Maps/sets | Native Map/Set copies, has/get/set/delete, array filter | Opaque identity, scalar keys, absent/duplicate errors; intersection left-order repair | Replace map/set algorithms; I25 |
| Optional/outcome data | Ordinary tagged Can records/variants | Catalogue identity for none/some; domain outcomes materialized explicitly | Replace wrappers; no universal authored completion API; I06 |
| Completion/coordination | Promise.all/allSettled/any/race | Nonthenable boxes, domain/standard rejection tags, preparation, input-order handlers, all_failed typing | New language lowering, not custom scheduler; I10,I19 |
| Resource ownership | Native promises, Bun server/SQL lifecycle | Registry, captured-handle leases, observer retention, owner drain and timeout boundaries | New necessary provenance bookkeeping; I20,I33,I35 |
| Assertions | Same emitted Bun program plus controlled native promises | P3 invocation identity, FIFO barriers, opaque fixture tokens and evidence labels | Delete Go evaluator/proof admission; I12,I18 |
| Bytes | Uint8Array, TextEncoder, fatal TextDecoder | Copy/own buffers, scalar admission, reject invalid decode; finite boundary mappings | Replace old bytes grants and codec loops; I13 |
| Typed JSON | JSON.parse with source context; JSON.rawJSON; JSON.stringify | Duplicate-key scanner only, exact integer-token normalization/budgets, nominal schema validation; native owns grammar/string escaping | Replace custom JSON parser/schema library; I14 |
| Connection/fetch | URL, URLSearchParams, Headers, fetch, AbortController, streams | Same origin, one attempt/no redirect, secret capture, body budgets, media/UTF-8/status policy and total deadline | New transport policy over native I/O; I15,I26 |
| Noul/Choice/Score | TypeSafe v1 System One | Native declaration semantics, descriptors, one batch, validate whole answer map, thresholds/generated fields/handlers | New compiler/provider adapter, not SDK-only feature or model recreation; I16,I17,I27 |
| LLM text/records | OpenAI Responses API | Closed nonstream schema subset, exact codec, refusal/truncation/invalid distinctions; no tools | New maintained protocol profile; I28 |
| CLI/I/O | Bun.argv/stdin/stdout/stderr/write | Strip runtime args, bound input, await output, map errors, drain root before exit | Replace host externs; I11,I29 |
| Clock/random/crypto/env/log | Date.now/performance.now/Bun.sleep; crypto.getRandomValues/randomUUID; CryptoHasher; Bun.env; JSON.stringify | Native range conversion, finite error/data contracts, supplied test boundaries | No PRNG/hash/log serializer rewrite; I29,I30 |
| Safe HTML | Bun.escapeHTML, native URL | Closed context/tag/attribute constructors and opaque trusted nodes; native escape alone is not trust provenance | Replace proof brands; no handwritten entity encoder; I31 |
| HTTP request/response/routes | Request, Response, Headers, URL, URLSearchParams, Bun.serve | Cached bounded body, strict malformed-percent guard, exact GET/POST 404/405/Allow, await handler, sanitized500 | Native transport; finite dispatch table only; I32,I33 |
| HTMX/static assets | Exact upstream HTMX4.0.0; Bun.file; SHA hashing | P11 static policy, digest checks, closed asset types, reserved route, no eval, same-origin, noSwap policy, 422swap | No client Can runtime or new browser framework; I34 |
| PostgreSQL syntax | libpg_query actual parser/scanner through existing binding | Manifest schema/statement-shape/parameter-span/LIMIT checks | No SQL lexer/parser; I36,I37 |
| PostgreSQL execution | Bun.SQL templates/begin/close | Template spans bind repeated parameter values; range/null/row decoding, row cap, commit/rollback/unknown-commit and leases | No transport/pool/transaction protocol; I35,I38 |
| Project/lock manifests | Go encoding/json, filepath and SHA-256 libraries | Strict known fields, duplicate detection through decoder tokens, canonical contained paths and Can identity policy | No custom JSON grammar or hashing; I05,I47 |
| Distribution/build | Go os/exec, archive/hash tools; upstream Bun executable | Exact runtime lookup/pin, verified bundle/update and dist ownership | No fork/linkable Bun; I01,I02,I39 |
| Source maps | gen-mapping and upstream trace-mapping if needed | Compiler records source spans/synthetic frames; helper encodes maps | No custom VLQ; I09,I40 |
| Editor/diagnostics | Existing LSP transport; vscode-languageclient | Adapt diagnostics and eligible-name information to new front-end without executing assertions | Reuse protocol scaffolding, replace old semantic hooks; I41 |

The `stdlib` surface is a closed compiler catalogue plus ordinary Can data declarations and domain examples. It is not a second implementation of native methods in Can. Tests may call native APIs directly as reference oracles, but production algorithms must not be duplicated in Go for assertion execution.
