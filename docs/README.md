# docs — can-lang design records (reviewer index)

Start with the current decisions and technical specification. Older implementation
records below describe the previous design and are not current language policy.

## Current design

- [Approved syntax decisions](syntax-taste/decisions.md)
- [Technical specification](syntax-taste/technical-spec.md)
- [AI and external data](syntax-taste/ai-io-spec.md)
- [Concurrency and coordination](syntax-taste/coordination-spec.md)
- [Platform and testing](syntax-taste/platform-testing-spec.md)
- [Latest design review](syntax-taste/deep-design-review-2026-09-20.md) — historical findings reconciled by the current specification.
- [Jev consultations and evidence](syntax-taste/jev-design-consultations-2026-09-20.md)

## Historical records

- [Zero-compatibility syntax and ABI audit](can-language-audit.md) — older,
  unapproved redesign recommendations. Its supporting probe archive was removed.
- [Stdlib implementation record](stdlib-remaining.md) — previous implementation
  history, not the implementation plan for the current design.

The status map and reviewer guidance below are historical and incomplete;
they do not override the current decisions or specification. Older `aNN`
documents generally live under `a/`.

## Status map

| Doc | Status | One line |
|---|---|---|
| `REQUIREMENTS.md` (repo root) | Living: v0.1 freeze + ratified amendments tagged a04–a12 | The rules; see freeze note below |
| `a02-machine-artifacts.md` | Shipped | Codes, `--format=json`, `normalize`, catalog |
| `a03-branch-coverage.md` | Shipped | Test-per-arm law over green tables |
| `a04-type-discipline.md` | Shipped | Brands, `seal`, exact `dec`, no floats |
| `a05-expressiveness.md` | Landed (plan) | The gap + build order; items 1–6, 8 shipped, 7 declined |
| `a06-arithmetic.md` | Shipped | `+`, `-`, `*` exact-or-loud; division deferred with reason |
| `a07-helpers.md` | Shipped | Same-file calls, no pin, no call-site `given` |
| `a08-termination.md` | Shipped | Proven self-recursion via `decreases`; cycles refused |
| `a09-effects.md` | Shipped | Private cells, declared capabilities, per-test stores |
| `a10-numerics.md` | Shipped | One numeric semantics: unbounded ints, exact decs, exact TS emit |
| `a11-recursion.md` | Shipped | Program-wide recursion ban, guarded unit steps, returned-outcome theorem |
| `a12-contracts.md` | Shipped | Producer-owned emits, complete error expectations, exchange script rows |
| `a13-stdlib.md` | Landed (part) | Stdlib rows 0–2: linkage decision, quota-counter validation, scalar catalog |
| `a14-tsc.md` | Landed (decision) | R11 tsc clause suspended until a real gate ships; canlc is the sole verifier |
| `a15-brands.md` | Shipped | Bodies seal only their own module's brands (CAN6004); tests/scripts name any brand |
| `a16-text.md` | Landed (part) | `+` concatenates strings; Unicode-scalar indexing decided; scalar-access surface open |
| `a17-division.md` | Landed (part) | Exact Euclidean `/` and `%` on ints; dec refused (CAN6005); gcd family waits on fuel |
| `a35-typed-fragments.md` | Sketch (pre-decision) | HTMX-shaped fragment endpoints after unions, HTTP, components; revisit trigger, no rule yet |
| `a36-seq-typed-construction.md` | Shipped (S1) | Typed `Seq<T>` literals, element checking, CAN6007; length/access/append are later slices |
| `a37-seq-length.md` | Shipped (S2) | `#` counts sequence elements; `std__seq__length` realized by `#`, no second spelling |
| `a38-seq-access.md` | Shipped (S3) | `xs[i]` over `Seq<T>` yields `T`; bounded traversal under existing `decreases`; `std/seq` deferred |
| `a39-seq-append.md` | Shipped (S4) | `Seq<T> + T` copy-on-append; concat refused; compiler complete, customers next |
| `a40-str-join.md` | Shipped (customer 1) | `std__str__join` preserves order; positional first-element test; split next |
| `a41-str-split.md` | Shipped (customer 2) | `std__str__split` retains empties, leftmost policy; wrapper return; mints `text.empty_separator` |
| `a42-fragment-join.md` | Shipped (customer 3) | `html__fragment__join` over explicit children; same-brand `+`; attributes fork posed |
| `a43-named-attributes.md` | Shipped (fork verdict B) | Name-carrying wrappers; pairs minted, never parsed; `make` next |
| `a45-bytes-values.md` | Shipped (Bytes B1) | `Bytes` value admission; `Bytes(Seq<int>[...])` literals `0..255`; `Uint8Array` emit; export/codecs next |
| `a46-bytes-export.md` | Shipped (Bytes B2) | Owner-local `exports_utf8` grants; `bytes__utf8__export` kernel; whole-program certification barrier; generic encoder next |
| `a47-bytes-encode.md` | Shipped (Bytes B3) | Public `bytes__utf8__encode` kernel; kernel descriptor table; strict `str` admission; Render consumer next |
| `can-idioms.md` | Living | `.can` style guide from blessed code: truth tables, bool-field match, guard shapes, fuel workers, arm coverage |

## Historical reading order for a reviewer

1. `REQUIREMENTS.md` Goal + R1–R9 (the thesis and the shape).
2. `a05-expressiveness.md` (what was missing and in what order).
3. `a06` → `a09` in order (each spec pairs a power with its proof).
4. `sketches/` live shape: `auth-login/`, `retry-loop/`, `counter/`;
   `std/` blessed library: `quota/`, `scalars/`.
5. `can-idioms.md` before writing or refactoring any `.can` file.
5. `CLEAN_ROOM_REVIEW.md` (design input; historical record, see note).

## Historical reading and editing rules

- The v0.1 freeze means: no silent drift. Amendments land tagged with
  their version (`(a07)`), never by rewriting a ratified rule.
- Every expressive power names its proof cost; a feature whose proof
  is "future work" is a bug with a roadmap.
- One rule, one `CANnnnn` code (`compiler/code.go` is the registry).
- Verify claims mechanically: `go test ./...`,
  `go run ./tools/modcheck`, `go run ./tools/gramcheck`.
  Goldens live beside their sketches; `broken-login/` titles are
  enforced by the suite, not by inspection.
