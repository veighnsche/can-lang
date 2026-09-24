# Stage 3 — Implementation reconciliation

**Historical implementation round:** this record describes I01–I50. Later LF01–LF21 and T01–T27 added or revised capabilities, including browser Can. Read the [post-upgrade reconciliation](../syntax-taste/post-upgrade-reconciliation-2026-09-24.md) for current status; earlier scope exclusions below are historical.

Completed after [research](research.md), [brainstorm](brainstorm.md) and the fresh [Jev experiment](evidence/2026-09-21/jev/). This stage chooses engineering direction. It does not amend C/Q/A/P or claim their implementation.

## Jev results and disagreement investigation

Three calls to `jev-latest` returned `jev-1.13.0`. Each contained the same three decisions and alternatives with every context, instruction and option description rewritten. Exact requests, responses, hashes, wording comparisons, manual semantic review and validation are preserved. [Preflight](evidence/2026-09-21/jev/preflight.md), [results](evidence/2026-09-21/jev/results.json). All returned keys, ranges, sums and selected maxima passed validation.

| Decision | Round1 (confidence) | Round2 | Round3 | Engineering disposition |
|---|---|---|---|---|
| Compiler structure | fresh .48 | insufficient .37 | insufficient .39 | New typed pipeline, with an early executable checkpoint; no claim of measured delivery-time superiority |
| Bun packaging | sidecar .94 | sidecar .77 | sidecar .96 | Versioned sidecar distribution selected for planning; signature/Gatekeeper release gate remains mandatory |
| SQL binding path | insufficient .17 | insufficient .33 | CGo-first .84 | Neutral comparative spike; exact binding adoption deferred to its measured exit gate |

The compiler answers alternate between fresh and insufficient; neither competing implementation strategy attracted substantial probability. Re-inspection of `compiler/main.go:450–535` shows generic expansion, proof certification, recursion refusal, static checking and execution in one pipeline. `compiler/linked.go:23–63` rejects capabilities that the new design deliberately supports. These are concrete coupling costs, not timing measurements. A new typed pipeline therefore remains the engineering choice, bounded by M1: if the CLI/assertion spine cannot be isolated without broad duplication, revise the internal module boundaries before expanding features. Do not reintroduce the old IR as a compatibility contract.

SQL disagreement reflects a real evidence limit: both bindings reuse the right parser, but no local performance/size/scanner parity comparison exists. The third wording changes the classification substantially even though the manually checked facts/options remain equivalent. We cannot infer Jev's reasoning from probabilities. We therefore drop the prior CGo-first preference as an adoption bias and schedule the same acceptance corpus against both. Choose the smallest maintainable passing binding from measured results; correctness and scanner fidelity precede startup or size. This bounded engineering choice does not require a syntax decision or block unrelated work.

Packaging agreement is supporting advice only. The decisive engineering fact is that no one-file constraint offsets extraction complexity, while sidecar packaging satisfies no separate Bun installation. No repeated calls were used to obtain agreement, and no confidence value authorizes implementation or release.

## Selected architecture

Keep the repository and Go compiler entry, replace the language pipeline. Proposed new directories are `compiler/internal/{source,syntax,project,resolve,types,check,ir,emit,driver}`, `runtime/{completion,owner,codec,transport,ai,assert,platform}`, and `compiler/testdata/current`. These are planned paths, not existing implementations. `std` becomes current catalogue declarations, ordinary types and domain examples. Keep shared runtime state in private modules and simple operations inline. Use structured typed identities and source spans throughout; no new stringly typed mega-IR.

Use one emitted execution semantics for normal programs and source assertions. Go never reimplements JS strings, floats, JSON, collection algorithms or the runtime completion machine to “verify” programs. Static graph/type/admission checks stay in Go. Compiler-owned assertion instrumentation switches external boundaries to fixtures; it does not change pure Can behavior or become a general-purpose production scheduler.

Intrinsic signatures, error IDs, opaque types, codecs and native mappings come from one closed distribution catalogue. It generates/checks the matching Go descriptors and private TS declarations. There is no project plugin mechanism, foreign backend import, user-defined protocol or hidden escape hatch.

## Contract decisions carried through to work

| Cross-cutting contract | Implementation consequence | Admission evidence |
|---|---|---|
| C3/C4 package/type identity | Typed symbol IDs; qualified nominal tags; exports separated by eligible kind | Same-name cross-package negatives and generated-field collision cases |
| C5/Q1/Q8 region ownership | Explicit region IDs in IR; handler completion never redispatched as its participant | Nested terminal/value match and handler-failure traces |
| C6/Q3 thenable data | Private null-prototype boxes at every Promise boundary, including callback wrappers and root | Data with callable `then` survives all combinators and containers |
| Q2/Q7/Q9 settlement | Prepare all before any launch; native Promise combinators; settlement versus handler order separate | Controlled target promises, empty cases including pending race |
| Q6 aggregate typing | Original failures input-ordered; infer concrete all_failed only when observed/forwarded; finite contextual type evidence | Generic/heterogeneous/nested/empty cases and ambiguity negatives |
| Q10/P6 ownership | Captured handles carry leases; observe losers; drain resource owner before actual native close/commit | Early failure, late rejection, escaped transaction and hung-owner tests |
| A6 boundary codec | Native parse/stringify plus bounded exact numeric and duplicate-key adaptations | Duplicate escaped keys, huge exponents, depth/node/byte boundaries, negative zero |
| A7–A9 native judgment | First-class declarations, prepare/register/batch/validate/handle/continue phases; answer bindings only in continuation | Whole-batch invalid answer causes zero handlers; source-order handler stop |
| A10 generation | Text/closed typed record output; no tools/streaming; exact schema and completion validation | Refusal/truncation/unknown field/type/limit cases |
| P3–P5 assertion semantics | Root/call-site/occurrence/participant/callable identity; shared FIFO barriers; five evidence levels | Repeated/transitive/parallel fixtures, mismatches and unused rows |
| P9–P12 trust/platform | Opaque construction, exact routes, static assets/SQL, native execution | Forgery, path/query/HTML injection, cardinality and transaction tests |
| C8/P2 initialization | Inert graph only; manifest descriptors; no load-time application effects | Cycles/forbidden calls fail before runtime |

## Open engineering gates and exclusions

No unresolved author-visible syntax is introduced here. I36 must select and pin the SQL binding/grammar after comparison. I02/I39 must prove the sidecar package on a clean macOS machine with trust policy enabled; signing credentials are needed at release time, not for this preparation. I35 must verify pool-open connection establishment, repeated placeholders and driver error metadata against a disposable PostgreSQL instance. I40 must prove Can→TS→stack mapping, including non-ASCII columns and generated frames. Failure of an upstream conformance gate is a blocker for that capability; it is not permission for an unreviewed substitute algorithm.

P14 exclusions stay explicit: decimal primitive, advanced numeric algorithms, full casefold, untyped JSON, user effects/proofs/ownership syntax, externs/backend imports/subprocesses, runtime project filesystem, non-PostgreSQL dialects, migrations/streaming, browser Can/client components/authored JS/TS, LLM tools, arbitrary route captures and streaming responses. Linux is future qualification; Windows is excluded from this initial plan. macOS x64 upstream availability is not Can support.

The [coverage ledger](coverage.md) connects every C/Q/A/P section, F01–F15 and the initial capability inventory to tasks. Legacy tests and goldens are candidates for replacement, never acceptance criteria by themselves.
