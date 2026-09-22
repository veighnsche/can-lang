# Prompt: resolve the Bytes plan open items (issue #42)

Paste this into a fresh agent/reviewer with [docs/bytes-plan.md](bytes-plan.md)
(v2) attached plus the context files listed at the bottom.

---

`docs/bytes-plan.md` v2 records the verdict on the original plan. Everything
below is still open. Resolve each item with a concrete, implementable decision
backed by file:line evidence from the attached context. Do not bundle
independently shippable surfaces into one slice; do not strengthen the brand
by assertion; do not copy `dec__parts`' total-kernel shortcut for fallible
decoders. No compiler gates have been run yet — state that, and do not claim
green.

1. Brand-export authority (pre-B2 gate, highest risk). Specify the narrowly
   authorized typed bytes-export boundary that lets `html__render__utf8`
   accept `Html__Safe` without a general unseal. Deliver: the exact rule
   (which function shapes may cross, on what authority), one legal Render
   example, and rejection evidence for an unrelated brand across the full
   encode → decode path (prove no `Secret`-like brand leaks to `str`).
2. NUL triptych. Pin three separate behaviors: (a) generic UTF-8 codec —
   does the `str → Bytes` encoder reject U+0000, and if so under which exact
   error kind + payload signature; (b) HTML escaping — unchanged, quote the
   boundary; (c) Render on a NUL-bearing `Html__Safe` — accept, reject, or
   escape, with rows. `Html__Safe` does not mean NUL-free
   (`std/html/html.can:97–107`, `encoder-nul-policy.md:38–43`).
3. Literal admission contract. Decide all five cases and their failure
   contracts: `Bytes(Seq<int>[-1])`, `[256]`, arbitrarily large integers,
   runtime `i`, runtime `xs: Seq<int>`. Separate the statically validated
   literal facility from the general conversion. Add the explicit
   primitive-constructor case and the parser/AST call: reuse `ctor` or mint
   a new node (with `walkSmallTrees` audit if new).
4. Emit implementation. Specify each site: `isScalar` exclusion/routing,
   `emitEquality` dispatch order, `emitValue` validated lowering (validate
   first + numeric literals for literals; bigint range-check before
   conversion for dynamic), helper usage-tracking/insertion, codec call
   lowering, checker-annotation/`leafType` handling (no imaginary
   `childType` arm), and the nested-equality contract with Bytes-in-record
   and Bytes-in-error-payload rows.
5. Codec decision tables. Malformed UTF-8 (fatal vs replacement), NUL, BOM
   preservation, hex grammar, base64 padding/alphabet/whitespace. Choosing a
   host decoder settles none of these — decide each explicitly.
6. Codec protocol. Kernels or ordinary declared functions? Register call
   checking, signature/result typing, given-table policy, exhaustiveness,
   emission, and kernel `EmitsOf` entries. Fallible decoders must not reuse
   the absent-`EmitsOf` total-kernel shortcut.
7. Type-checking registration. Name every touchpoint: `knownType`,
   `typeOf`, value/constructor validation, comparison admission,
   declaration-field checks, the authorized encode-input rule, and the
   `Seq<Bytes>` supported/refused decision with tests.
8. Evaluation. Runtime value representation, constructor/codec evaluation,
   structural `vEq`, diagnostic/value normalization. Prove Bytes never
   enters the `contradictScriptOk` unevaluable fallback, and add an
   intentionally incorrect byte-result linkage row that must fail.
9. Errors and catalogs. Ownership of the new `ErrorDecl`s, typed fields,
   function contracts, kernel `EmitsOf`; regenerate registries and emitted
   artifacts.
10. State cells. `checkStateDecl` enumerates four scalar types — explicitly
    defer Bytes-as-state or extend it with literal shapes + tests. Do not
    promise it via the word "primitive".
11. Scope rulings. First-slice boundary, hex-literal sugar slice placement,
    base64url inclusion, public `==`/`!=` on Bytes. Each gets its own slice
    or a written deferral — none ride along silently.
12. Slice gates. Per slice: committed direct rows, README/slice doc,
    generated outputs where applicable, three forced gates. Plus: name the
    return wrapper record for codec/Render signatures (bare `→ Bytes` is
    shorthand; `declaredOkShape` requires declared records).
13. Remnants. Module-header emits-list inventory (the narrow remainder of
    old Gap2) and Gap8's syntax-position wording + rejection rows through
    the new constructor.

Return: per-item decision + evidence (or `unresolved:` with the exact
missing source), then the ordered slice list with gate criteria, ready to
execute as B1…Bn.

---

## Context files

- `docs/bytes-plan.md` (the v2 plan under resolution)
- `docs/bytes-workstream.md` (requirements, acceptance)
- `docs/a/a36-seq-typed-construction.md`, `docs/a/a37-seq-length.md` (template)
- `docs/encoder-nul-policy.md`, `docs/fault-contracts.md` (boundaries)
- `std/text/text.can`, `std/html/html.can` (consumers)
- `compiler/emit.go`, `compiler/check.go`, `compiler/eval.go`,
  `compiler/parse.go`, `compiler/types.go`, `compiler/lsp.go` (follow the
  plan's line references from these files outward)
