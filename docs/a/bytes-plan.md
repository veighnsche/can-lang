# Bytes workstream plan (issue #42) — v3, resolution of open items

Central decision: the generic UTF-8 encoder stays strictly `str → Bytes`.
A separately authorized, owner-local bytes-export boundary serves Render.
Generic UTF-8 and Render both preserve NUL; each decoder's grammar is explicit.

History: v1 (workflow synthesis) → verdict (request changes; slices
rebracketed, NUL withdrawn, Gap2/Gap9 closed as stale) → v2 → this
resolution → B2 pre-implementation review folded (certificate lifecycle
barrier, registration/identity/shape corrections, order-independence
fixtures) → B4 pre-implementation review folded (v3 example rows joined —
split outcomes are `CAN1000`; promotion-chain audit; NUL vector; escaping
regressions; artifact expectations; downstream unresolved). Reviewer caveat: it saw v1/v2 + prompt only (v3 and the B1
doc failed to retrieve on its side; both exist locally) — its cited
anchors were re-verified here against `de82756` before folding. Prior docs: [bytes-workstream.md](bytes-workstream.md),
`bytes-plan-review-prompt.md` (referenced prompt bundle; file not in repo),
[bytes-open-items-prompt.md](bytes-open-items-prompt.md).

**No compiler gates have been run.** Below are implementation decisions and
required acceptance evidence — not claims of compiling/passing code.
Baseline commit `de82756`. Anchor sweep (post-resolution, all green):
`CAN6008–CAN6012` free (`compiler/code.go:23–84`, type family ends
`CAN6007`); `isScalar` maps every `tsBase` entry to identity comparison
(`compiler/emit.go:247–252`) with `emitEquality` scalar-first dispatch
(`260–292`); `seals_from` same-module authority + `CAN6003` promotion
rejection (`compiler/types.go:412–419,1166–1186`); exact
`got != want → CAN6003` rule (`compiler/types.go:464`); `Module.ID`
canonical identity vs display `File` (`compiler/parse.go:195–205,1281–1282`);
positional-construction rejection the Bytes branch must precede
(`compiler/eval.go:508–511`); `vEq` + `verifyExhaustiveAll` + `EmitsOf`
(`compiler/eval.go:286,1292–1301`); `contradictScriptOk` trust list
(`compiler/check.go:763–810`); `checkCoverage`/`relayStatus`
(`compiler/lsp.go:357–405`); UTF-8 source gate
(`compiler/parse.go:1042–1050`); fault triptych
(`docs/fault-contracts.md:9–14`); `text.empty_separator` declared+raised
but header-omitted (`std/text/text.can:21,27,390–405`); brand base
(`std/html/html.can:8–12,28–30`); Seq atomic-boundary + `vEq`-as-support
(`docs/a36-seq-typed-construction.md:106–118`); one-operation + kernel
plumbing warning (`docs/a37-seq-length.md:39–55`); `checkCalls`/`checkGiven`/
`evCallMatch`/`emitValue`/`emitModule`/`ctor` parse sites exist. Not
line-verified: emit/eval/check interior sub-spans (function-level
touchpoints confirmed by name; drift risk cosmetic only). `checkStateDecl` admits four scalars with `CAN6002` rejection
(`compiler/check.go:1607–1638`); deterministic-call `given` rejection exists
(`compiler/check.go:647–671`, `CodeGivenOnLocal`); positional-construction
rejection via `isKwargList` (`compiler/eval.go:509`, `compiler/parse.go:949`).

## 1. Brand-export authority: explicit owner-local grant, one function shape

New declaration form:

```can
exports_utf8 Html__Safe via html__render__utf8@1
```

Authorizes one function revision to disclose that brand's representation as
UTF-8 Bytes. Changes nothing about the brand declaration, `seals_from`,
argument compatibility, or admissible `Html__Safe` contents. Implement
`Utf8ExportDecl{Brand, Function, Revision, Line}` plus a dedicated validation
pass after declaration resolution. It defines no value/function: not in
`provides`. Grant valid only when:

| Requirement | Exact rule                                                              |
| ----------- | ----------------------------------------------------------------------- |
| Authority   | Grant, brand declaration, and exporter share one canonical `Module.ID` (basename, label, prefix, or underlying `str` insufficient). |
| Identity    | Brand and function resolve unambiguously; grant revision equals function revision. Ambiguity invalidates, no first-wins inheritance. |
| Parameters  | Exactly one parameter of exactly the granted brand.                     |
| Result      | Exactly `Bytes__Value` (declared record, single field `value: Bytes`).  |
| Contract    | `emits []`, no effects, no decreases clause.                            |
| Body        | Exactly one call match over `bytes__utf8__export(parameter)`; one `Ok` arm returning `r.value` unchanged. No `given`, helpers, extra branches, conversion, or string intermediates. |
| Scope       | Attaches to that function and call site — not the file, the brand, or a caller's `uses`. |

Follows the owner-local authority model (sealing checks use declaring-file
identity + exact nominal types; `seals_from` admits explicit same-module
promotions; canonical module identity ≠ display filename).
Anchors: `compiler/types.go:350–630`; `compiler/parse.go:154–205`. Store a
checked authorization certificate (brand, function, revision, parameter,
export-call node); evaluator and emitter reject restricted export calls
without it. Never implement as `underlyingType == "str"` or
`callerFile == brandFile` alone.

Certificate lifecycle barrier (B2 review correction — verified hole):
`checkProgram` loops modules through `checkSem`, which runs
`checkScriptConsistency` before `checkTypes`, and `contradictScriptOk`
trusts on any provider evaluation error (`compiler/main.go:317–348`;
`compiler/lsp.go:270,272`; `compiler/check.go:763–810`). Issuing
certificates during per-function type checking would let a consumer
checked before its exporter pass on trusted script evidence, with the
exporter certified only later. Required order: resolve declarations
and identities → reject ambiguous grants/targets → check export
shapes and relevant types for all modules → issue certificates for
this program → run linkage evaluation and decision tables → emit with
those same certificates. Certificates bind program + declaration
identities + revision + parameter + call site; grant removal,
signature, or body changes invalidate even when `fn@rev` is unchanged.
Invalid/missing export authority must never become trusted evidence
through the linkage fallback — distinguish it from ordinary sandbox
modeling failure. Exhaustiveness, coverage, and test-data seals
cannot manufacture a certificate.

Grant registration: separate pass over `Module.Decls` retaining the
owning module — never inside the `buildWorld` switch, which skips
empty-name declarations (`compiler/check.go:113–150`). Grants are not
provided symbols (no `provides` obligation, no double-definition).
Identity invariant: same validated module identity in this program,
not three matching strings; reject empty/duplicate identities and
ambiguous targets before certification (no first-wins for brands).
Loader note: normal CLI `parsePaths` preserves full paths and rejects
duplicate identities (`compiler/main.go:233–258`); legacy `parseModule`
strips to basename (`compiler/parse.go:1022–1040`) but is currently
uncalled — the invariant covers it if that changes.

Legal Render example (additions to the HTML module; add function to
`provides`; `Bytes__Value` per item 9; `Html__Safe` stays an
ordinary-child-fragment brand — no script/style/attribute/URL authority;
anchor `std/html/html.can:1–12,28–42`):

```can
exports_utf8 Html__Safe via html__render__utf8@1

fn html__render__utf8(document: Html__Safe) -> Bytes__Value rev 1
  emits []
  tests
    render_empty(document = seal Html__Safe("")) => Ok(value = Bytes(Seq<int>[]))
    render_entity(document = seal Html__Safe("&amp;")) => Ok(value = Bytes(Seq<int>[38, 97, 109, 112, 59]))
=
  match call bytes__utf8__export(document)
    on Ok r => Ok(value = r.value)
```

Body shape is an exact AST predicate, not "find an export call": `IsMatch`,
`Kind == MatchCall`, exactly one scrutinee naming `bytes__utf8__export` with
exactly one positional argument that is the bare parameter reference
(`isBareRef`, not a field or expression); `Given == nil` (reject even an
empty table); exactly one arm, a bound `Ok` variant whose non-match RHS is
`Ok` with exactly one named field `value` holding `Ref[ok-binder, "value"]`.
Count arms, not outcome kinds — exhaustiveness maps and coverage are not
authorization. Ordinary callers of the public exporter stay legal. Caveat:
the parser overwrites repeated `emits`/`effects` metadata
(`compiler/parse.go:1190–1260`), so the rule governs effective AST metadata,
not source-line uniqueness.

Export input type is call-site-specific: the descriptor holds result shape,
explicit empty emits, and dispatch policy, but the permitted nominal input
comes from the certificate for the current owner and call node — never a
global signature, `underlying == str`, or last-registered exporter. Positive
test: two independently granted brands in one program, both module orders,
plus a third ungranted brand still rejected.

Secret path stays closed — generic kernel parameter is exactly `str`, so
`Vault__Secret → bytes__utf8__encode` fails `CAN6003` at the argument
(anchors `compiler/types.go:350–630,630–650`); no decoder repairs the missing
edge. Only the first forbidden edge must fail — downstream correctly-typed
edges need no invented diagnostic. Required rejection fixtures: generic encode of `Vault__Secret` then
decode (`CAN6003` at encoder); ungranted-function export then decode
(authority diagnostic); `Vault__Secret` to public Render (`CAN6003`);
cross-module grant (authority diagnostic); granted function returning `str` /
extra fields / calling helpers (shape diagnostic); same-file private brand to
granted public-brand exporter (nominal mismatch); private value hidden in a
record field or `Seq<str>` (exact-type rejection); seals in test/given data
(create no authority). Before the real decoder lands, use a test-only
declared Bytes-to-text sink; demand the authority/type diagnostic, not
unknown-callee. Repeat all with the real decoder in its slice. Limits: an
owner granting export to a private brand changes its disclosure contract;
permitted promotions into an exported brand reach the exporter — inspect
incoming `seals_from` paths for the HTML grant. Guarantee: no new
representation-recovery path for an unrelated unexported brand in checked
CAN — not an information-flow theorem, not protection against hostile
TypeScript bypassing erased brands.

## 2. NUL triptych

| Boundary           | U+0000 behavior                                | Declared outcome                                     |
| ------------------ | ---------------------------------------------- | ---------------------------------------------------- |
| Generic UTF-8 encode | Accept; encode byte `00`                     | `emits []`                                           |
| Generic UTF-8 decode | Accept valid byte `00` → U+0000 in `str`     | No NUL error; malformed → `encoding.invalid_utf8`    |
| HTML escaping      | Reject, unchanged                              | `html.nul_byte(value: str)`, complete original input |
| Render of `Html__Safe` | Accept and preserve; no filter/reject/escape | `emits []`                                         |

Policy: "the encoder rejects NUL," not "every `Html__Text` is NUL-free";
promotion preserves NUL; `html__text__node` seals with `emits []` without the
escaping encoder (anchors `encoder-nul-policy.md:9–24,38–43`;
`std/html/html.can:97–107`). Required Render rows (`N` = U+0000 notation):
`"" → []`; `N → [0]`; `Na → [0,97]`; `aNb → [97,0,98]`; `abN → [97,98,0]`;
`"&amp;" → [38,97,109,112,59]` (no second escaping). Mirror NUL-position rows
in generic encode/decode; keep HTML-escape rejection rows. Fixtures use the
source's existing raw-NUL convention — no invented `"\0"`/`"\u0000"`.
Render promises exact serialized bytes, not preservation of parsed HTML text.

## 3. Literal admission: B1 is a statically validated literal facility

Accept only `Bytes(Seq<int>[0, 127, 128, 255])`-shaped literals:
exactly one positional argument, an explicit `Seq<int>` literal, every member
an integer-literal AST node with mathematical value in `0..255`. No
constant-folding of arbitrary expressions. `big.Int` comparison for
arbitrarily large values (no narrowing first). General `Seq<int> → Bytes`
conversion is deferred — no runtime path in this workstream.

| Case                                  | Decision           | Failure contract (compile diagnostics, no `emits`/payload) |
| ------------------------------------- | ------------------ | ---------------------------------------------------------- |
| `Bytes(Seq<int>[-1])` / `[256]`       | Reject statically  | Byte-element-range diagnostic; index + exact value          |
| Arbitrarily large integer             | Reject if outside  | Same diagnostic, complete decimal value                     |
| `Bytes(Seq<int>[i])`, runtime `i`     | Reject even if a test value is in range | Bytes-literal-shape diagnostic               |
| `Bytes(xs)`, runtime `xs: Seq<int>`   | Reject             | Bytes-literal-shape diagnostic                              |

New codes (verified free against `compiler/code.go:3–20,75–105`; rebase must
re-check collisions): `CAN6008 CodeBytesLiteral`, `CAN6009
CodeBytesElementRange`, `CAN6010 CodeBytesExportAuthority`, `CAN6011
CodeBytesExportShape`, `CAN6012 CodePrimitiveShadow`. Append to `allCodes`
and JSON diagnostic goldens. Parser/AST: reuse `Small{Kind: "ctor", Ctor:
"Bytes"}` (anchors `compiler/parse.go:650–820`; `compiler/eval.go:265–585`) —
no new `Small` kind or lexer rule; existing `walkSmallTrees` traversal
applies, with nested-constructor traversal rows. Evaluator Bytes branch must
precede the `isKwargList` positional-construction rejection (addition at
record lookup is too late). Export grant adds a declaration node, not a
value-expression node.

## 4. Emit: `Uint8Array` with explicit typed dispatch

(Anchors `compiler/emit.go:13–60,209–292,523–582,870–884,937–1005,1408–1474`.)

| Site | Implementation |
| ---- | -------------- |
| `tsBase`, `tsTypeB` | `Bytes → Uint8Array` (`Seq<Bytes>` then yields `Uint8Array[]` via the uniform Seq mapping). |
| `isScalar` | Exclude Bytes from native identity comparison; `tsBase` membership must not imply identity-comparability. |
| `emitEquality` | Direct Bytes operand handled before the scalar branch; during this workstream reject defensively (direct operators deferred) — never emit `===` for Bytes. |
| `emitValue` constructor | Recognize Bytes before ordinary named fields; revalidate shape/range for direct emitter callers; emit numeric literals (`Uint8Array.from([0, 255])`). |
| Dynamic construction | No lowering ships; a later conversion must range-check each bigint before `Number(...)` (unchecked coercion is not an implementation — `Uint8Array.from([0n,255n])` throws, `Uint8Array.from([-1,256])` yields `[255,0]`). |
| `leafType`/annotations | Recognize validated Bytes constructors in `leafType`; annotate checked constructors/references as `Bytes`. `childType` stays annotation-first, then `leafType` (no per-kind arm). |
| Codec calls | Resolve parameter metadata, result unions, helper selection from the kernel descriptor; ordinary result-switch lowering, no Ok-only path. |
| Helper tracking | Track codec + comparison dependencies; init in `emitModule`; insert once, fixed order, after body emission discovers usage. |

Nested equality: B1 preserves existing structural `==` for admitted records
and error payloads containing Bytes (compositional admission, not
unobservability). Add a typed-array branch before `$canEqVal`'s generic
object traversal (anchor `compiler/emit.go:773–811`): length + ordered
contents; typed array never equals an ordinary numeric array via enumerable
keys. Generate branch + helper only when the compared declared shape can
contain Bytes (records, sequences). Rows: record `value: Bytes` (equal
separately-allocated `[0,255]`; unequal: byte, length, order); nested record;
same-kind error payloads (equal complete input vs changed/truncated); record
with `Seq<Bytes>` (ordered members vs reordered/changed). Pin helper absence
when unneeded, single insertion when shared. Aggregate `!=` is not repaired
here (evaluator routes `==` via `vEq` but restricts other comparisons to
scalar runtime kinds — separate repair).

## 5. Codec decision tables (chosen contracts, not host-library inference)

Byte columns use hex octet notation for readability — not hex literal syntax.

UTF-8: encode all CAN scalar strings, preserving every scalar incl. NUL and
U+FEFF; no BOM insert/remove; no normalization. Decode strictly: malformed →
`encoding.invalid_utf8(value = original_bytes)`; never a decoded prefix or
U+FFFD substitution. Grammar excludes overlongs, surrogates, > U+10FFFF
([RFC 3629](https://www.rfc-editor.org/info/rfc3629/)).

| Input bytes   | Decode result                           |
| ------------- | --------------------------------------- |
| Empty         | `""`                                    |
| `00`          | U+0000                                  |
| `41`          | `"A"`                                   |
| `C3 A9`       | `"é"`                                   |
| `F0 9F 98 80` | `"😀"`                                  |
| `EF BF BD`    | Actual U+FFFD (valid data, not repair)  |
| `EF BB BF 41` | U+FEFF + `"A"` (BOM preserved)          |
| `80` / `C2` / `E2 28 A1` / `C0 80` / `ED A0 80` / `F4 90 80 80` / `FF` | Error: isolated continuation / truncation / bad continuation / overlong NUL / surrogate / above range / bad leading byte |

Plus scalar-boundary vectors, malformed-after-valid-prefix, repeated BOMs,
and inverse encoding rows for valid cases. TypeScript: strict state machine,
or validate-then-`TextDecoder("utf-8", {fatal: true, ignoreBOM: true})`
(preserves BOM). Never blanket-catch host/resource exceptions as invalid
UTF-8 ([Encoding Standard](https://encoding.spec.whatwg.org/)). Source parser
already rejects malformed UTF-8 (`compiler/parse.go:1042–1050`); the encoder
is total over CAN scalars — a host lone surrogate is a loud host-contract
fault, not `html.nul_byte` (anchor `fault-contracts.md:9–14`).

Hex: lowercase ASCII encode, two chars/byte. Decode: either letter case,
even length, entire input; no prefix/separators/whitespace/lookalikes.

| Decode input | Result |
| ------------ | ------ |
| `""` → empty Bytes; `"00"` → `00`; `"00fF"` → `00 FF` | success (`"ff" → FF` does not UTF-8-validate output) |
| `"0"`, `"0g"`, `"0x00"`, any whitespace, embedded NUL/non-ASCII | `encoding.invalid_hex(original)` |

Base64: standard `A–Z a–z 0–9 + /` only; canonical padded encode, no
wrapping. Decode rejects whitespace, URL substitutions, misplaced/excess
padding, missing required padding, nonzero unused pad bits (base64url is a
different alphabet per RFC 4648 — deferred;
[RFC 4648](https://www.rfc-editor.org/info/rfc4648/)).

| Decode input | Result |
| ------------ | ------ |
| `""` → empty; `"Zg=="` → `66`; `"Zm8="` → `66 6F`; `"Zm9v"` → `66 6F 6F`; `"AA=="` → `00`; `"/w=="` → `FF` (no UTF-8 validation) | success |
| `"Zg"`, `"Zg="`, `"Zg==="` (padding) / `"Zh=="`, `"Zm9="` (nonzero pad bits) / embedded `=`, whitespace, NUL, `-`, `_`, non-ASCII | `encoding.invalid_base64(original)` |

Validate complete consumption (no permissive decoder / regex anchor missing
trailing newline). Conformance harness checks round-trip laws with domains:
UTF-8 identity (scalar strings, valid bytes); byte identity (hex/base64);
hex lowercase canonicalization; exact text identity (canonical base64).
Cross-operation laws run in the Go/target harness — no manufactured
unwitnessable CAN decoder-error arm for a round-trip function.

## 6. Codec protocol: deterministic kernels + stdlib wrappers

One kernel descriptor table authorizes signatures, success records, emitted
kinds, evaluator dispatch, target helper selection:

| Kernel | Input | Success record | Explicit `EmitsOf` |
| ------ | ----- | -------------- | ------------------ |
| `bytes__utf8__export` | exact brand per export certificate | `Bytes__Value` | `[]` |
| `bytes__utf8__encode` | `str` | `Bytes__Value` | `[]` |
| `bytes__utf8__decode` | `Bytes` | `Encoding__Text` | `[encoding.invalid_utf8]` |
| `bytes__hex__encode` | `Bytes` | `Encoding__Text` | `[]` |
| `bytes__hex__decode` | `str` | `Bytes__Value` | `[encoding.invalid_hex]` |
| `bytes__base64__encode` | `Bytes` | `Encoding__Text` | `[]` |
| `bytes__base64__decode` | `str` | `Bytes__Value` | `[encoding.invalid_base64]` |

Export input is an authorization specialization, not a generic/any-brand
parameter. Registration paths (anchors `compiler/check.go:430–498,635–679`;
`compiler/eval.go:836–860,1300–1328`; `compiler/emit.go:937–1005`): call
checking recognizes kernels in `checkCalls`, keeping the no-outside-scrutinee
/ no-nested-calls prohibitions; parameters/results resolve via
`callee`/shared contract lookup with declared result-record `Ok` bindings;
existing binding machinery for ordinary codecs (exporter's exact body fixes
its operand); kernels deterministic — reject `given` with `CodeGivenOnLocal`,
no failure injection; evaluation dispatches before local/foreign handling,
rejection as language error value (not Go error); exhaustiveness requires the
kernel entry to exist, then exactly `ok ∪ EmitsOf[kernel]` (missing entry =
compiler failure); emission populates callee/result unions from the
descriptor, normal discriminated-result switch; ordinary wrappers are normal
functions (normal `uses`/foreign-call rules for callers). Invariant: a kernel
is callable only with parameter contract + result declaration + explicit
emits entry + evaluator + emitter all installed. Every decoder gets
missing-arm and stale-arm rejections — none inherits `dec__parts`' absent
entry (its comment depends on totality). The exhaustiveness loop treats
absent and empty `EmitsOf` alike, so B2 must verify the export kernel's
contract exists independently of its empty error set.

## 7. Type-checking registration: ordinary composition, no implicit conversions

(Anchors `compiler/types.go:630–990,1040 onward`.)

| Touchpoint | Change |
| ---------- | ------ |
| `knownType` | Admit `Bytes`; keep exact nominal comparison. |
| `typeOf` | Primitive Bytes constructor separate from record constructors. |
| `value` / `checkCtor` | Literal shape, element types, literalness, range in every value position; preserve child diagnostics + `exec`. |
| Reference annotations | `T = "Bytes"` for Bytes values incl. record/codec-result fields. |
| Comparison admission | Reject direct `==`, `!=`, ordering; keep supported structural record `==`. |
| `callee` / args / match bindings | Complete kernel signatures + success-record shapes; no unchecked kernel argument. |
| `checkTypes` / returns | Reject bare `-> Bytes` pre-evaluation; require wrapper record. |
| `checkExternSig` | Bytes params / Bytes-containing records permitted; record return still required. |
| `checkDeclFields` | Bytes fields in records + error declarations. |
| `checkBrandDecl` | Brands stay string-backed; no `brand X is Bytes`. |
| Export checking | Validate grant + exact body, then authorize only the checked call site. |

Support `Seq<Bytes>` in B1 (uniform known-element-type rule, not a
Bytes-specific exception); nested sequences stay rejected. Rows:
empty/nonempty `Seq<Bytes>`, append + checked access with existing Seq ops,
wrong-member rejection, `Seq<int>`/`Seq<Bytes>` mismatch, nested-Seq
rejection — repeated across body values, call args, record fields, test args,
expected results, `given` args/results (Seq coverage matrix). Length,
indexing, slicing, concatenation, seq conversions: unsupported.

## 8. Evaluation: dedicated runtime kind + linkage-fallback evidence

Owned `[]byte` runtime kind — never `Value.S` or untagged int sequences.
Constructor validates before creating; kernels emit fresh buffers, preserve
complete original input on rejection; admitted values immutable. Extend
`evSmall` (primitive constructor before positional-record rejection),
`evCallMatch` (deterministic codec dispatch), `vEq` (length + ordered
contents; `false, nil` for unequal valid Bytes), `normalizeValue` (canonical
`Bytes(Seq<int>[0, 255])`), diagnostic/result rendering (Bytes distinct from
text/Seq). Anchors `compiler/eval.go:15–33,287–340,343–585,999–1055`.

Linkage fixture (required `CAN3110`, not acceptance): provider computes
`Ok(value = Bytes(Seq<int>[0, 255]))`, call-site script claims
`Ok(value = Bytes(Seq<int>[0]))`. Repeat: reordered/changed bytes, Bytes in
record/sequence, provider computing Bytes via the real kernel once encoding
exists; plus an intentionally wrong direct expected result (`CAN4200`).
Obligation: no new unevaluable/incomparable path — not "no Bytes program ever
enters the fallback" (unavailable foreign deps, resource/depth failures, eval
errors, panics still trust; anchor `compiler/check.go:763–810`). Test
`evSmall`/`vEq` directly so a malformed fixture's rejection cannot stand in
for the intended evidence.

## 9. Errors, declarations, catalogs: compiler-owned contracts

Registry owns:

```can
type Bytes__Value rev 1 (value: Bytes)
type Encoding__Text rev 1 (value: str)
error encoding.invalid_utf8(value: Bytes)
error encoding.invalid_hex(value: str)
error encoding.invalid_base64(value: str)
```

Each error lands with its decoder — no unimplemented future surface; typed
declarations, not synthetic untyped shapes like `parts`. One shared lookup
over compiler-owned + source declarations; update world construction,
`newTycker`, `recordDecl`, `recordShapes`, return-union construction, extern
unions, catalog generation (anchors `compiler/check.go:99–176`;
`compiler/emit.go:81–91,140–206` — `Program.Errors` holds field names while
checking/emission need typed `ErrorDecl` fields). Emit builtin record
definitions into referencing generated modules. Reject source shadowing of
primitive/kernel/compiler-owned names. Keep `errors.json` schema (name-only
`fields`); typed declarations authoritative. Kernel attribution as
`kernel.bytes__utf8__decode` (no invented `.can` line); wrappers via
`eachRaise`. Regenerate `errors.json`, TypeScript, normalization and
diagnostic goldens. Wrapper arms reconstruct same kind + unchanged original
input; `emits` lists re-raised kinds; no callee-emits-superset rule;
`eachRaise`'s conservative bound-reference behavior unchanged.

## 10. State cells: defer

No `checkStateDecl` change. `state Store__payload: Bytes =
Bytes(Seq<int>[])` stays rejected (`CAN6002`) — add that regression beside
the scalar-state cases. Primitive ≠ state-admissible.

## 11. Scope rulings

| Surface | Ruling |
| ------- | ------ |
| First slice | Values, literal validation, typed-container composition, evaluator equality, normalization, necessary emission. No codecs/consumers. |
| Export authority | Own slice; permission checking + restricted export land atomically. |
| Render | Own consumer slice. |
| Runtime `Seq<int> → Bytes` | Written deferral. |
| Hex literal sugar | Written deferral; never bundled with a codec. |
| Base64url | Written deferral of both ops; standard base64 rejects URL-only chars. |
| Direct `==`/`!=` | Written deferral of both; test equality is not this surface. |
| Bytes state / byte collection operators | Written deferral. |
| JSON/random/hash | Outside this workstream. |

Preserves Seq's value-admission vs operations/customers distinction
(anchors `a36-seq-typed-construction.md:13–17,106–118`;
`a37-seq-length.md:39–55`).

## 12. Slice gates + return wrappers

Every slice: versioned scope/rollback doc, committed direct rows,
README/index updates, regenerated outputs where applicable. Every new CAN
match arm executed by a committed row (identity-relay exception is not
substitute evidence; anchors `a36-seq-typed-construction.md:134–155`;
`compiler/lsp.go:357–405`). After each slice, forcibly:

```sh
go test -count=1 ./...
go run ./tools/modcheck
go run ./tools/gramcheck
```

Emission-changing slices also run committed target-runtime vectors against Go
outcomes; missing/failed runner = missing evidence, never skipped-green. No
claim of restoring the separate repo-wide TypeScript gate. Return
conventions: bytes → `Bytes__Value`, text → `Encoding__Text`; bare
`→ Bytes`/`→ str` is shorthand (`declaredOkShape` derives output fields from
declared records, not example rows).

## 13. Remnants

Module-header inventory: add `text.empty_separator` to
`std/text/text.can`'s header (declared + raised at `text.can:390–405` with two
rejection rows; catalog `std/text/errors.json:14–23`). Header = inventory of
the module's declared function outcomes, not a union of callee outcomes — no
new global checker law. Gap8 wording + constructor regressions:

| Position | Diagnostic |
| -------- | ---------- |
| `Bytes([0, 1])` | `CAN6007` (bare list) |
| Bare `Seq` annotation incl. Bytes-constructing bodies | `CAN6002` |
| Malformed `Seq<...` inside Bytes | `CAN1000` / `CodeParse` |
| `Bytes(Seq)`, unbound value name | `CAN6003` at the reference |
| Valid `Seq<int>` syntax, runtime member | `CAN6008` |
| Integer literal outside `0..255` | `CAN6009` |

Check children first; suppress derivative Bytes-shape diagnostics when the
child's own rule explains the failure; never overwrite the original code for
Bytes-nesting. Old Gap9 stays closed — no Bytes migration of the
branded-append fixture.

---

# Ordered execution: B1–B15 (deliberate renumber; new B2 = export authority, old B2 encoder = B3)

Kernel fixtures are compiler acceptance fixtures, not shipped public
consumers. Each slice closes item-12 gates before the next lands.

| Slice | Single capability | Slice-specific acceptance |
| ----- | ----------------- | ------------------------- |
| B1 | Value admission + literal construction | Five-case matrix; all value positions; empty/nonempty/order/repeats; `Seq<Bytes>` composition; nested structural `==`; normalization; wrong-result `CAN3110`/`CAN4200`; direct-operator + state rejection; emitter numeric-literal pins. |
| B2 | Owner-authorized typed UTF-8 export | Grant declaration (separate registration pass) + exact AST exporter shape; lifecycle barrier (certificates before any linkage evaluation); branded export rows incl. own byte-correctness (empty/ASCII/non-ASCII/supplementary/NUL/BOM); unrelated-brand denials; same-basename/different-owner rejection via both loader routes; no string-returning or helper-forwarding exporter; explicit `EmitsOf` entry with independent existence check; two-grant both-orders test; sink controls (allowed route / denied route / real-decoder repeat); order-independence: `CAN3110` for wrong scripted export bytes under both module orders; grant-removal invalidation control; no real HTML consumer yet. |
| B3 | Generic UTF-8 encode kernel | Strict `str` admission; NUL/BOM/Unicode vectors; no brand acceptance; empty `EmitsOf`; deterministic no-`given` rule. |
| B4 | `html__render__utf8` consumer | HTML-owned grant + function (single-line rows — split outcomes are `CAN1000`); empty/entity/markup/Unicode/NUL-position rows; exact serialization with no second escaping (distinguish decode `[38]` from double-escape); 8-byte NUL vector `[65, 0, 38, 97, 109, 112, 59, 66]`; disclosure audit closes on Text/Attributes/Attribute chain + opaque composition; unchanged `Html__Safe`, promotion, escaping; `html.ts` regen (new fn + union member), `errors.json` regen expecting byte-identical; README remnants (`seals_from` description, errors-empty claim, `utf8 waits` comment at `html.can:1580`); composition probes in temp-copy harness, never stdlib rows; downstream consumer contract stays unresolved (claim is exact serialization, not end-to-end delivery). |
| B5 | `std__utf8__encode` wrapper | Declared Bytes wrapper result; computed encoding rows; inventories + artifacts updated. |
| B6 | UTF-8 decode kernel | Strict grammar + complete-input errors; BOM/NUL preservation; fallible contract; missing/stale-arm rejections; full encode→decode brand-rejection fixtures; harness round-trip laws. |
| B7 | `std__utf8__decode` wrapper | Valid/invalid rows; exhaustive match; unchanged reconstruction; full Bytes payload verified. |
| B8 | Hex encode kernel | Lowercase; empty, zero, `FF`, ordered multi-byte; no text interpretation. |
| B9 | `std__hex__encode` wrapper | Output rows; inventory/artifact updates. |
| B10 | Hex decode kernel | Entire-input even-length ASCII grammar; mixed-case success; prefix/whitespace/NUL/non-ASCII failures; original payload; arbitrary output bytes. |
| B11 | `std__hex__decode` wrapper | Computed success/failure rows; unchanged propagation. |
| B12 | Standard base64 encode kernel | Empty + all length remainders; standard alphabet; canonical padding; no wrapping. |
| B13 | `std__base64__encode` wrapper | Output rows; inventory/artifact updates. |
| B14 | Standard base64 decode kernel | Alphabet, complete consumption, padding placement, zero unused bits, whitespace/URL rejection; original payload; arbitrary output bytes. |
| B15 | `std__base64__decode` wrapper | Computed outcomes; exhaustive unchanged propagation; final consumer acceptance + regeneration. |

Pre-encoder security condition: export authority explicitly granted,
owner-local, shape-checked, and incapable of authorizing the generic
encoder. The later decoder supplies the final executable composition
regression without revealing an ungranted permission.
