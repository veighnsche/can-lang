# I14 design consultations

The three saved requests preserve these facts: native JSON owns syntax and
formatting; ints use source tokens and rawJSON; floats retain negative zero;
nominal recursive types require finite schemas; byte/depth/node budgets include
cumulative integer expansion and pre-formatting bounds. A separate JSON engine
or eagerly mirrored serialization tree is prohibited. Sealed types, completion
carriers and owned bytes already exist. I14 needs explicit source codec calls
before I46 implements general function specialization/inference.

Each request rewrites its entire context, both questions and all alternative
explanations. Pairwise wording checks and a semantic review were performed before
sending. Alternatives remain native adapters with lazy access views, an eager
wire tree, or a complete custom parser/formatter; admission alternatives remain
codec-only explicit specialization, runtime-only deferral, or general generics.

All three selected the native adapters (confidence 1.0). All selected explicit
codec specialization (confidence 1.0, 0.98, 1.0). There was no choice disagreement.
This is advice, not proof: tests independently exercise exact numeric conversion,
resource limits, native syntax precedence, nominal reconstruction and authored
source calls. General generic inference remains I46 work.

The live TypeSafe index, HTTP API and Choice documentation were read through
HTTP after the browser fetch failed. Requests used the documented SystemOne
endpoint and jev-latest; actual responses identify jev-1.13.0. No credentials are
included in saved evidence.
