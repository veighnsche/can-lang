# docs — can-lang design records (reviewer index)

For current implementation status, start with the
[post-upgrade reconciliation](syntax-taste/post-upgrade-reconciliation-2026-09-24.md).
The original [I01–I50 ledger](implementation/tasks.md), September 22
[LF01–LF21 ledger](implementation/language-fixes-tasks-2026-09-22.md), and
September 24 [T01–T27 ledger](syntax-taste/can-implementation-task-list-2026-09-24.md)
record completed execution rounds. Their checked tasks do not establish that
every accepted contract is fulfilled; the reconciliation identifies remaining
gaps against current source. [Decisions](syntax-taste/decisions.md) and selected
design packets describe requirements, while dated evidence records describe
the particular checks that ran.

## Current recommendation program (2026-09-24)

- [Post-upgrade language review](syntax-taste/post-upgrade-language-review-961f921-2026-09-24.md) — source-based assessment after the upgrade.
- [Finding-by-finding reconciliation](syntax-taste/post-upgrade-reconciliation-2026-09-24.md) — bugs, unfinished accepted requirements and new proposals, with current evidence and earlier decisions.
- [Next-upgrade dispositions](syntax-taste/post-upgrade-dispositions-2026-09-24.md) — selected fix, retain or defer scope for every reconciled finding.
- [Selected next-upgrade behavior](syntax-taste/post-upgrade-selected-behavior-2026-09-24.md) — proposed Can source, compile/failure matrix, browser build and native lowering contract for the fixes.
- [Invoice composition acceptance](syntax-taste/post-upgrade-invoice-acceptance-2026-09-24.md) — live server/browser/database gate, supported delivery path and cross-target contract-edit checks for the existing invoice examples.
- [Post-upgrade implementation plan](syntax-taste/post-upgrade-implementation-plan-2026-09-24.md) — architecture, migration sequence and verification boundaries for all 11 fixes.
- [Ordered implementation tasks and lanes](syntax-taste/post-upgrade-implementation-tasks-2026-09-24.md) — 27 tasks with prerequisites, source ownership, concurrent worker waves and completion checks.
- [Consolidated Can recommendation program](syntax-taste/can-recommendation-program-2026-09-24.md) — non-normative roadmap and acceptance gates for server-driven and Can-authored browser SaaS; reconciles the two September 24 reviews.

## Design records (2026-09-20)
- [Approved syntax decisions](syntax-taste/decisions.md)
- [Technical specification](syntax-taste/technical-spec.md)
- [AI and external data](syntax-taste/ai-io-spec.md)
- [Concurrency and coordination](syntax-taste/coordination-spec.md)
- [Platform and testing](syntax-taste/platform-testing-spec.md)
- [Latest design review](syntax-taste/deep-design-review-2026-09-20.md) — historical findings reconciled by the current specification.
- [Jev consultations and evidence](syntax-taste/jev-design-consultations-2026-09-20.md)

## Implementation preparation

- [Five-stage implementation packet](implementation/README.md) — research, alternatives, reconciliation, plan, and task ledger for the current design.

## Historical records

- [Zero-compatibility syntax and ABI audit](archive/can-language-audit.md) — older,
  unapproved redesign recommendations. Its supporting probe archive was removed.
- [Stdlib implementation record](archive/stdlib-remaining.md) — previous implementation
  history, not the implementation plan for the current design.

The status map and reviewer guidance below are historical and incomplete;
they do not override the current decisions or specification. Older `aNN`
documents generally live under `archive/a/`.

## Status map

| Doc | Status | One line |
|---|---|---|
| `REQUIREMENTS.md` (repo root) | Historical: v0.1 freeze superseded, bannered, body untouched | Predecessor rules; see banner |
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
| `can-idioms.md` | Removed | Predecessor style guide; documented retired `lint`/`explain`/CAN codes as living guidance. Deleted with the archived design web. |

## Historical reading order for a reviewer

1. `REQUIREMENTS.md` Goal + R1–R9 (the predecessor thesis and shape;
   superseded — see its banner).
2. `a05-expressiveness.md` (what was missing and in what order).
3. `a06` → `a09` in order (each spec pairs a power with its proof).
4. The retired gallery as recorded in `docs/archive/sketches/README.md` and the
   `std/` package histories (sources deleted in I44).
5. ~~`can-idioms.md`~~ (removed; predecessor style guide that misclaimed living status).
6. `docs/archive/sketches/CLEAN_ROOM_REVIEW.md` (design input; historical record, see note).

## Historical reading and editing rules

- The v0.1 freeze meant: no silent drift. Amendments landed tagged
  with their version (`(a07)`), never by rewriting a ratified rule.
- Every expressive power named its proof cost; a feature whose proof
  was "future work" was a bug with a roadmap.
- One rule, one `CANnnnn` code (the registry was deleted with the
  predecessor toolchain in I44).
- Verify current claims mechanically: `go test ./...`,
  `bun test runtime/test/`, `go run ./tools/modcheck`,
  `go run ./tools/gramcheck`, plus the `tscheck/` fresh-emit gate.
  Committed goldens are gone; maintained examples assert and build
  fresh from staged layouts.
