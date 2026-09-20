# Postflight review: capability decisions

## Method and evidence boundary

This review covers `integer_codec`, `generation_shapes`, `fixture_queue`, and
`generic_checking`. It compares the three raw Jev responses with the current
contracts in [A](../../ai-io-spec.md), [P](../../platform-testing-spec.md), and
[C](../../technical-spec.md). No API call was made.

The repeated categorical selections are corroborating advice, not a ballot or
proof. The decision for each topic below rests on whether its current contract
closes the required behavior with bounded implementation work, native reuse,
and explicit evidence gates. Confidence is reported only to show the shape of
the raw evidence; it does not replace that engineering assessment.

| Topic | Round 1 | Round 2 | Round 3 |
| --- | --- | --- | --- |
| exact integer codec | `exact_integral_tokens` 0.99; confidence 0.98 | 0.99; confidence 0.99 | 0.98; confidence 0.98 |
| generated shapes | `bounded_initial_subset` 0.98; confidence 0.97 | 1.00; confidence 1.00 | 0.96; confidence 0.94 |
| fixture allocation | `canonical_shared_fifo` 0.96; confidence 0.95 | 0.94; confidence 0.93 | 0.99; confidence 0.97 |
| generic checking | `concrete_specialization` 0.99; confidence 0.97 | 0.95; confidence 0.93 | 0.95; confidence 0.93 |

The numeric values in the table are the selected option probabilities followed
by the separately returned Choice confidence.

## Exact integral JSON tokens

**Assessment: retain `exact_integral_tokens`. No changed conclusion and no
design blocker.**

The engineering case is stronger than categorical agreement. A6 uses one typed
codec for standalone JSON, fetch, and generated records, while A10 emits JSON
Schema `integer` for generated `int` fields. Restricting input to digit-only
integer lexemes would reject `1.0` and `1e0` even though their exact values are
integral and the provider schema admits them. That would turn valid provider or
HTTP representations into Can domain failures for spelling alone.

A6.2 closes the two risks that originally made broader acceptance questionable:

- exactness comes from the native `JSON.parse` source token rather than its
  rounded `Number` value;
- integrality is decided from coefficient, fraction length, exponent, and
  trailing zeros before any expansion;
- signed exponent comparisons stay bounded and do not construct an unbounded
  native number or bigint;
- output length is checked against the remaining shared byte budget before
  zero padding, decimal allocation, or `BigInt` conversion; and
- a cumulative budget prevents many small exponent tokens from each expanding
  to the full limit.

This is a narrow adapter around native parsing and native `BigInt`, rather than
a second general JSON parser or a decimal arithmetic facility. It also gives
stable failures: nonintegral exact values produce `integer_token`, excessive
canonical output produces `byte_limit`, quoted numbers remain `type`, and no
rounded `Number` can be repaired.

The retained cost is meaningful. The compiler/runtime must keep per-holder
source tokens until typed traversal completes, implement bounded signed-exponent
comparison correctly, share the integer expansion budget across the document,
and preserve the specified error precedence. The digit-only alternative would
be smaller and easier to audit. Its simplicity does not outweigh the schema
interoperability defect now that A6 supplies a bounded algorithm.

The remaining gate is target evidence: pinned Bun must provide the source-token
context used by A6, and conformance must cover negative zero, underflow and
overflow spellings, huge positive and negative exponents, cumulative budgets,
and values beyond binary64 precision. Failure of that target capability would
block this lowering on that target; it is not an unresolved language-design
choice and must not be papered over with `Number` recovery.

## Generated output shapes

**Assessment: retain `bounded_initial_subset`. No changed conclusion and no
initial-capability blocker.**

The initial production trace needs a closed ordinary record containing nested
records, arrays, and scalar fields. A10.1 directly covers that trace after
concrete generic resolution, emits closed required-field object schemas, and
then validates the received output independently through A6. The fixed depth
8, property count 1024, and schema byte count 65536 bounds make schema
construction and diagnostics finite. Unsupported types fail at compile time
with a field path.

The broader shared codec does not by itself justify a broader provider schema.
Supporting named variants and recursive records would require a second body of
work: graph identity, reference emission, discriminated alternatives, recursive
budget accounting, provider-specific schema restrictions, and raw conformance
for each reference and alternative shape. Provider documentation showing some
`anyOf` and recursive support establishes possibility, not portability or need.
The current trace gains no capability from taking on that coupling.

The tradeoff is deliberate duplication of shape boundaries. A6 can encode
tagged variants and finite recursive values that A10 generation rejects. Users
cannot ask generation to return those otherwise valid Can wire types, and
repeated nonrecursive structures are expanded rather than referenced. If a
future closed program actually requires generated variants or recursive data,
`reuse_broader_type_graph` remains the right direction to reconsider; it should
arrive with a provider schema graph contract and conformance suite, not through
an implicit widening of A10.

There is no present blocker. The compiler must still prove the schema budgets,
reject every unsupported reachable field before launch, and test that provider
schema adherence never bypasses A6 validation. Those are implementation and
conformance obligations already stated by A, not reasons to expand the initial
surface.

## Canonical fixture allocation

**Assessment: retain `canonical_shared_fifo`. No changed conclusion and no
design blocker, with an important evidence limitation.**

P3 gives FIFO assignment an identity independent of native completion timing.
The root includes package, declaration, and assertion. Each invocation path
adds its lexical call site, occurrence, coordination participant path, and
callable instance; receiver and `near` captures contribute a diagnostic
fingerprint. Coordination reserves participant identities in written and spread
order. At a shared lexical table, the canonical scheduler assigns the next row
to the next reserved invocation identity, validates the expected arguments, and
never searches ahead.

That contract preserves the useful meaning of repeated rows: they form one
ordered script across recursive or concurrent visits to the same transitive
helper. `participant_local_fifo` would avoid shared allocation but replay the
same lexical script independently for each participant; authors would need new
wrappers or call sites to script cross-participant outcomes. `argument_matching`
would make fixture selection depend on values and still could not distinguish
two repeated calls with equal arguments. It would also hide a mistaken call by
searching later rows rather than reporting the first assigned-row mismatch.

The retained cost is a test-only scheduler with path-order barriers and precise
dynamic identity bookkeeping. Its ordering is intentionally artificial. A
canonical FIFO assertion proves consumer behavior for one reproducible schedule
and detects missing, malformed, mismatched, ambiguous, or unused fixtures; it
does not prove the timing behavior of production promises.

P3 now states the necessary separate evidence: Bun-native conformance tests use
controlled promises to force both orders for two participants and
representative larger and nested interleavings. This separation is material to
the conclusion. Without it, canonical FIFO would risk being mistaken for
production concurrency evidence. With it, the fixture design stays deterministic
without adding participant-selector syntax or live calls.

No language blocker remains. Implementation must demonstrate stable lexical
ordinals, occurrence scoping, callable-instance identity, prepare-before-launch
behavior, barrier progress, and complete drainage of started owners. A failure
to implement those identities would invalidate this candidate rather than
justify falling back to timing-based queue consumption.

## Concrete generic specialization

**Assessment: retain `concrete_specialization`. No changed conclusion and no
design blocker.**

C4 permits useful generic functions without adding a typeclass, trait, bound,
overload, or effect-parameter language. The declaration is checked immediately
for grammar, names, and errors independent of its type parameters. Each
reachable concrete specialization then checks the entire body, including
branches not executed by assertions, and operations on a parameter are accepted
only when that concrete type supplies them. Diagnostics identify both the
generic source and the requesting application.

This policy matches Can's closed-build setting. Dependencies provide source and
concrete applications are known during compilation; there is no external-user
ABI or old generated layout to preserve. The compiler can reject expanding
polymorphic recursion, permit same-specialization recursion, keep nominal
generics invariant, and derive finite callback error bounds for compiler-known
catalogue operations. Assertions remain evidence for only their concrete
instances rather than a false proof of a universal template.

`parametric_only` would make more errors appear at the generic declaration and
reduce specialization work, which is a real diagnostic and compiler-complexity
advantage. Its cost is a severe expressiveness restriction in the absence of
constraint syntax: generic arithmetic, member access, and other algorithms
would require concrete named wrappers even when every reachable use is valid.
That cost is not warranted by the current product boundary.

The retained liabilities are delayed use-site errors for some public templates,
potential repeated checking/code generation, and the need for deterministic
specialization memoization and cycle detection. C4 already fixes the semantic
limits: explicit concrete type arguments after specialization, no inferred
unions or error sets, full-body checking, source-plus-use diagnostics, and
rejection of infinitely expanding specialization. These are engineering tasks,
not missing source semantics.

There is no current blocker. If future distribution boundaries require shipping
generic declarations without their bodies or guaranteeing that every possible
consumer type checks before use, this choice would need reopening or actual
constraint syntax. Neither condition belongs to the present closed compiler
contract.

## Overall disposition

All four current conclusions remain justified after the fresh consultations.
The result is not “three votes win.” Each selected contract now has a concrete
reason the alternatives do not serve the initial slice as well:

- exact integral tokens align shared codec semantics with JSON Schema while
  bounding expansion before allocation;
- the generated-shape subset closes the demonstrated workflow without taking
  on provider-specific recursive schema machinery;
- canonical FIFO gives repeated fixtures one deterministic meaning while native
  controlled-interleaving tests carry production timing evidence; and
- concrete specialization preserves useful generic algorithms within a closed,
  finite compilation graph and no new constraint syntax.

No changed conclusion or specification blocker was found. The remaining risks
are target conformance and implementation obligations already exposed by A, P,
and C. None warrants a new source feature or a silent weakening of the chosen
contracts.
