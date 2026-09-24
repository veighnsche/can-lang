# Independent core-language review — Can 961f921a6a8cf40be54735683caf29613c19cbd8

Reviewed 2026-09-24 from source, current tests, all current gallery examples, and all maintained std examples. No previous reviews, verdicts, design-preparation, consultation, reconciliation, disposition, or task acceptance records were read. No Jev consultation performed: coordinator owns that step. No production source edits. Temporary compiler test helpers were removed; standalone probe source/output is preserved in `/tmp/can-upgrade-core-961f921-evidence/`.

## Recommendation within this domain

This is a substantial, working core with coherent immutable data, nominal records, representation confinement, pattern coverage, exact integers, checked domain errors, and protected native async lowering. It is credible for bounded application work. I would not yet recommend the core broadly as a general SaaS implementation language: exported generic helpers cannot compose even with a generic identity function; ordinary tail recursion exhausts the native stack at modest depth; and compiler-enforced presentation rules impede routine refactors. Those concerns are independent of platform/library breadth. A confirmed Unicode regex correctness defect should be fixed, but does not itself imply a foundational language redesign.

The highest-value next work is composition/refactor stability, not adding many isolated constructs. No backwards compatibility is needed. Every proposed implementation experiment below can preserve native JavaScript/Bun lowering.

## Verified strengths

* **Immutability is implemented, not merely a type annotation.** `runtime/data.ts:21–55` constructs fresh frozen null-prototype nominal records, preserves aliases on updates, and freezes fresh arrays. Map/set backing stores are inaccessible through frozen tokens and mutations copy native stores (`runtime/collections/map.ts:25–33,47–79`; `set.ts`). Current runtime tests check forged handles, nominal identities, aliases, ordering, and copy updates. This meaningfully reduces accidental shared-state bugs.
* **Owner-controlled values create a useful abstraction boundary.** `owner record` supports public names with package-confined construction, field projection, updates, and destructuring. Tests in `compiler/internal/check/owner_record_test.go:99–177` reject those foreign operations and reject codec bypasses, including nested generic composition in `core_integration_test.go`. Public factory/accessor APIs work; generic transport of a value works. This is enough to express validated domain values without callers recreating representation invariants. Equality is still representation-derived; see limitation below.
* **Patterns distinguish binding from nominal cases.** Explicit `bind` avoids a misspelled variant leaf silently becoming a catch-all. `pattern_bind_test.go:43–110` exercises nested product patterns, remainder bindings, and compatible alternative binders; missing and overlapping coverage is checked by the pattern matrix (`pattern_coverage.go:12–29`). Finite work bounds are a reasonable compiler resource policy, not an application type-safety hole.
* **Contracts accurately separate domain failures from standard failures.** Concrete domain effects are checked and completion arms cover the exact invoked bound. `completion_matches.go:192–200` rejects duplicate arms and enforces success-last. Carrier implementation in `runtime/completion.ts:15–73` uses private WeakSet-branded frozen boxes, preventing application `then` fields from becoming accidental promises. The core runtime tests verify that data survives async boundaries and handlers do not redispatch their own failures.
* **Integer and monetary semantics are unusually solid.** BigInt integers avoid the JavaScript safe-integer trap; conversion to float checks exact representability (`runtime/number.ts`). Integer divide/remainder report primitive standard faults (`runtime/primitive.ts:16–26`). Maintained `std/ratio/current/src/main.can` covers signed truncating/Euclidean division, half-even rounding, conservation of remainder, and values beyond binary64. Explicit exact-amount operations return a named zero-divisor error, letting expected invalid input be handled without treating it as a programming fault.
* **Text units are explicit in the available operations.** Basic strings follow native UTF-16 indexing/slicing; scalar and grapheme APIs supply the other common views and reject malformed surrogates (`runtime/text.ts:84–119`). Maintained examples demonstrate the distinction rather than pretending code units are user-perceived characters. Literal replace-all avoids JavaScript replacement-token surprises.
* **Collections preserve callback ordering and failure behavior.** Array map/filter/fold/search/sort carry callback error sets and stop after failures. Runtime callbacks do not accidentally assimilate returned application data. Sorting computes keys once, then uses native stable sorting. Named callables work in records, arrays, and arguments with immutable captured values.

## Findings and experiments

### C1 — Broad-library blocker: exporting a generic helper destroys safe composition

Observed with `CheckProgram` against the current source. A private `wrapper<item>` returning `call identity(value)` passes. Adding both functions to `provides` makes that same source fail:

```
generic function can.project.root/app::identity cannot be instantiated with opaque type parameter wrapper<item> from exported generic declaration can.project.root/app::wrapper: pass the value through a non-generic contract or an explicit callable input
```

Minimal body (add the standard `main` below):

```can
package app
    provides [identity, wrapper]
    uses []
fn item identity<item>
    emits []
    given
        item value
    asserts
        sample: 1 => ok 1
    ok value
fn item wrapper<item>
    emits []
    given
        item value
    asserts
        sample: 1 => ok 1
    ok call identity(value)
```

Changing only `provides [identity, wrapper]` to `provides []` passes. This is not an assertion-result failure or hypothetical lack of traits. The declaration checker explicitly forbids cross-generic instantiation of opaque parameters while allowing same-declaration recursion (`check/exported_generics.go:13–27,35–39`; `types/specialize.go:141–154`). Existing `TestExportedGenericRejectsRepresentationOperations` pins a similar rejection. Extracting a helper from an exported generic can therefore break a formerly valid implementation; exporting a tested private library helper changes its language rules.

Counterargument: opaque checking prevents a dependency implementation edit from silently narrowing a published generic's admitted types, and requiring explicit operator dictionaries is sensible. That does not justify rejecting composition of two separately checked universally quantified contracts. The identity example requires no representation operation and no trait search.

Smallest useful experiment: add symbolic application of another declaration-checked generic, substituting symbolic types into its checked signature without entering the concrete emission specialization cache. Require identity-wrapper and two-level map-wrapper examples to pass; still reject numeric `+` on an unconstrained type variable; verify a callee body edit adding unsupported operations fails at that callee. A later choice is to unify private/public generic semantics, or give template semantics an explicit declaration form. Visibility should not silently select two type systems.

### C2 — General iteration limitation: `relay` tail recursion is not stack-safe

The actual compiler emitted this source through `CheckProgram` and `RegionEmitter.Function`:

```can
fn int countdown
    emits []
    given
        int remaining
    asserts
        sample: 2 => ok 0
    match remaining is 0
        false => relay call countdown(remaining - 1)
        true => ok remaining
```

Pinned Bun 1.4.2 on this machine:

```
100 ok
1000 ok
10000 standard
native_exception RangeError: Maximum call stack size exceeded.
```

The emitted TypeScript and exact Can program are preserved with output. The probe invokes generated functions directly, outside the CLI supervisor, but uses the real completion runtime. `emit/regions.go:388–390,481–486` emits nested native calls inside await/catch, with no relay trampoline. It is therefore incorrect to infer tail-call safety from the word `relay` or the async ABI.

This matters for pagination, repeated polling/state transitions, interpreters, graph traversal, and other loops not reducible to iterating an already-built array. The gallery promotes recursion (`07`, `08`, `09`, `10`, `11`, `27`), though several comments appropriately constrain examples to small inputs. The stack threshold is environment-dependent; 10,000 is an observed failure, not a guaranteed limit.

Counterargument: array folds are an existing stack-safe traversal alternative; `runtime/collections/array.ts:86–95` chains native promises. Much SaaS business logic does not require deep recursion. This makes the issue a general-purpose limitation rather than a claim every application crashes.

Smallest useful experiment: lower same-function terminal relay to a native `while` loop with fresh prepared argument temporaries and preserved completion/error/trace semantics. Test 100,000 iterations, argument evaluation exactly once, a domain error midway, and an assertion fixture on the recursive edge. Alternatively offer one native-lowered iterative state primitive. Avoid rebuilding arbitrary recursion as an interpreter.

### C3 — Correctness defect: Unicode empty regex matches repeat at one position

Direct current runtime probe, using successful paths only:

```ts
const text = createText({} as any, {match: 'match'} as any);
const compiled = await text.compileRegex('(?:)', 'u');
const hits = await text.findMatches(compiled.value, '😀x', 5n);
```

Observed match starts: `[0,0,0,0,0]`. Native `'😀x'.matchAll(/(?:)/gu)` yields `[0,2,3]`. Same defect with `v`; no-Unicode mode correctly yields `[0,1,2,3]`. Exact probe and output preserved.

`runtime/text.ts:172–184` increments `lastIndex` by one after an empty match. Under Unicode matching this puts the next index inside a surrogate pair and Bun's engine returns the same prior position; the result cap terminates the scan, but produces repeated incorrect matches and misses later positions. Existing empty-hit tests pass because they do not cover this interaction.

Smallest fix experiment: iterate native `matchAll` with an early result cap, keeping the same immutable records, group representation and error contracts. Test empty expressions and lookarounds over astral text with both `u` and `v`; compare offsets to native iteration. This directly follows the native-lowering requirement. A narrow correct AdvanceStringIndex adapter is another option.

### C4 — Refactor/ergonomics issue: compile errors enforce presentation choices

Observed independently:

```can
    ok amount * 2
```

passes, while

```can
    int invoice_total = amount * 2
    ok invoice_total
```

fails: `unnecessary local invoice_total ...; replace with ok amount * 2`.

The implementation is careful to preserve typing and avoid changing faulting expressions (`check/locals.go:84–111`); this is not an unsafe optimizer. The design problem is treating a meaningful name as an illegal program. The syntactic whitelist means replacing multiplication with a call or division can make the name legal (`locals.go:113–138`), making extraction and arithmetic refactors affect admissibility for reasons unrelated to correctness.

Likewise a Boolean match with `true` then `false` fails with `ordinary Boolean match requires false before true` (`completion_matches.go:441–450`). Ordered matching otherwise permits source order and already checks usefulness. The rule applies only to a single Boolean column and a literal pattern, so it does not express a universal ordering principle across equivalent match shapes.

Counterargument: canonical presentation can reduce stylistic variability and make machine-generated code uniform. A formatter or opt-in lint can enforce these tastes without rejecting ordinary explanatory names and useful branch emphasis.

Smallest experiment: downgrade the unnecessary-local and Boolean-order diagnostics to lint/formatter advice, leaving type, exhaustiveness and unreachable-arm errors unchanged. Re-run the core suite after updating tests that specifically require these stylistic rejections. Evaluate readability on gallery and invoice code, rather than treating current tests as proof that this taste is desirable.

Related limitation: multiline parenthesized/bracketed expressions are lexically forbidden (`syntax/lexer.go:90–98`). Function assertions are mandatory nonempty syntax (`declarations.go:259–266`). These are explicit policies, not missing implementation. Together with explicit types and long generic contracts, single-line calls/assertions grow difficult to maintain. A small multiline-delimiter parser/formatter experiment would address real layout pressure without adding new semantics. Mandatory assertions bring useful executable documentation, but one example per function is not evidence of universal correctness.

### C5 — Abstraction limitation: builtin callback effects cannot be expressed generically by authors

Array operations derive their complete domain error set from the actual callback (`check/array.go:276–311`). Authored callable types and function `emits` contain concrete finite nominal bounds; there is deliberately no error-set parameter (`exported_generics.go:26–27`). Thus users can consume effect-polymorphic builtin map/fold but cannot write a reusable wrapper that transparently preserves an arbitrary callback's effect set. Private generic templates do not automatically parameterize the error bound either.

Counterargument: explicit finite errors are easy to inspect and stable. Application-specific wrappers can choose a concrete union and adapt lower-level errors. Those are good defaults, but do not replace generic retry/traversal/instrumentation library contracts.

Smallest experiment: one explicit error-set parameter restricted initially to callable bounds and the enclosing `emits`; demonstrate an authored `apply` and retry wrapper that preserve two unrelated error sets. Check specialization identity and exported symbolic validation rather than inferring hidden broad effects. This is a design recommendation supported by implementation constraints, not a newly observed type-safety defect.

### C6 — Capture refactor hazard: parameter spelling is behavioral API

`near` captures by looking up the declaration's parameter name in the caller's lexical scope (`check/callables.go:124–150`). Named receiver/near capture is immutable and happens when the callable value is formed. Existing tests verify correct capture timing and missing/wrong-type errors.

However renaming a callee's `near int prefix` to `near int suffix`, with the body correspondingly renamed, changes which same-typed caller local is captured, even though ordinary direct calls retain positional behavior and the reduced callable type is unchanged. If the second name exists, there need not be a compile error. This is an implementation-backed semantic consequence; I did not run a fresh renaming execution probe. Tests in `callables_test.go:43–51` cover missing captures, but that counterexample has two valid names.

Counterargument: explicit `near` intentionally advertises name-based environmental capture and avoids inline closure syntax. Within a disciplined small package it is succinct. Across general application/library boundaries, the caller's capture mapping should be reviewable where the closure is made.

Smallest experiment: add explicit named capture bindings at a callable reference, or a capture record construction, preserving native closures and immutable captures. Compare a filter capturing a tenant ID and a later parameter rename. No need to add unrestricted anonymous functions before testing this narrower improvement.

### C7 — Common collection workload limitation: immutable map construction has no bulk path

Verified current authoritative catalogue operations: `empty_map`, `empty_set`, `get`, `insert`, `replace`, `remove`, `entries`, `contains`, `add`, `union`, `intersection`, `difference` (12 entries). Searched current runtime/check/emit/catalogue/std/example source for `from_entries`, `group_by`, `combine`, `bulk`; no map bulk builder exists at this HEAD. This is not inferred from a stale README or only from the gallery.

Each successful insert/replace copies the full native Map (`runtime/collections/map.ts:56,67`); each created token additionally records a copy of all values for resource containment (`map.ts:25–28`). The recommended gallery word-frequency fold (`examples/gallery/src/24-map-word-frequencies.can:13–27`) therefore copies growing state for each item, requiring quadratic work for a sequence of distinct words. It also repeats get/insert/replace error plumbing, including impossible branches recovered as unchanged data.

This is an algorithmic source finding, not a measured general SaaS performance claim. Small maps and explicit immutable updates are fine, and retaining native Map is a sound constraint. Array operations cover many bounded transforms.

Smallest experiment: a bulk `from_entries` constructor with an explicit duplicate policy and/or a key/count/group aggregate implemented with one private native Map then one immutable published handle. Test duplicate policy, insertion order, callback errors, aliases, resource containment, and scaling at 1k/10k/100k distinct entries. This keeps immutability and native lowering and avoids inventing a persistent-map implementation.

## Additional semantics to retain or document

* Variants are extensional named leaf sets, while records are nominal and generic records/arrays invariant (`types/compatibility.go:9–39`). Different variant names with the same leaves are assignable. This is coherent union semantics, but a variant name is not a nominal brand; use owner records for distinct validated concepts. No evidence here warrants making all variants nominal.
* Callable argument/result types are exact; only finite error bounds widen (`compatibility.go:41–60`). This is conservative and can require adapters around subtype-like variant relationships, but prevents implicit callable coercion surprises.
* Owner-record automatic equality still observes hidden representation (`types/inhabitation.go:118–159`; `owner_record_test.go:82–91`). Adding a hidden field can change consumer equality; adding an opaque/callable field can remove equality eligibility entirely. The owner boundary prevents representation access/forgery, but is not complete representation independence. Consider owner-selected equality if broad abstraction stability becomes a goal. This is a design consequence, not an observed breach.
* Float equality intentionally uses Object.is-like identity: NaN equals itself and +0 differs from -0 (`check/expressions.go:231–239`, `emit/expressions.go:182`, `runtime/data.test.ts`). Ordering uses native numeric comparisons. Document this clearly; no unsoundness was observed.
* `emits []` does not mean total, non-faulting or pure: indexing, integer division, resource/native failure, and recursion exhaustion can produce standard failures. This is implemented consistently. Expected input rejection should use named validation/division operations; do not equate an empty domain bound with mathematical totality.
* Native regex cap limits result count, not CPU time (`runtime/text.ts` comments near MAX_MATCHES). Arbitrary user patterns still need application-level policy; no regex sandbox or deadline was claimed by this review.

## Exact inspected example inventory

Read all 29 gallery source files: `01-hello-world`, `02-cli-argument`, `03-arithmetic`, `04-temperature`, `05-leap-year`, `06-fizzbuzz`, `07-fibonacci`, `08-factorial`, `09-gcd`, `10-prime-check`, `11-palindrome`, `12-word-character-count`, `13-boolean-match`, `14-record-variant-match`, `15-exhaustive-match`, `16-optional-values`, `17-named-error-recovery`, `18-relay`, `19-immutable-record-update`, `20-array-map`, `21-array-filter`, `22-array-fold`, `23-array-sort`, `24-map-word-frequencies`, `25-generic-function`, `26-callable-argument`, `27-recursion-vs-collection`, `28-attached-assertions`, and `main`, all under `examples/gallery/src/*.can`.

Read all maintained std `.can` files found under std: `std/map/current/src/main.can`, `std/ratio/current/src/main.can`, `std/scalars/current/src/main.can`, `std/text/current/src/main.can`. Read callable capture fixture `compiler/testdata/current/callables/captures.can`. Gallery arithmetic/recursion examples are pedagogical and mostly small-input examples, not scale demonstrations. Its generic identity and callable examples do not exercise exported helper composition, so cannot rebut C1.

Primary implementation inspected (whole files or relevant sections, not a claim of line-by-line full repository audit):

- Syntax: `declarations.go`, `lexer.go`, `matches.go`, syntax README.
- Check: `callables.go`, `exported_generics.go`, `locals.go`, `completions.go`, `completion_matches.go`, `pattern_coverage.go`, `patterns.go`, `expressions.go`, `recursion.go`, `array.go`, `collections.go`, checker README.
- Types: `compatibility.go`, `inhabitation.go`, `specialize.go`.
- Emit: `regions.go`, `expressions.go`, `program.go`, `modules.go`, `runtime_core.go`, `program_entry.go` sections.
- Runtime: `data.ts`, `completion.ts`, `primitive.ts`, `number.ts`, `text.ts`, `collections/array.ts`, `collections/map.ts`, `collections/set.ts`, `failure.ts` sections.
- Tests read in depth: check `exported_generics_test.go`, `owner_record_test.go`, `core_integration_test.go`, `pattern_bind_test.go`, `callables_test.go`; emit `callables_test.go`, `modules_test.go`; runtime `data.test.ts`, and selected text/number/collection test inventory. Test names/results from the executed suites below were also inspected.
- Inventory/operational references: `AGENTS.md`, `package.json`, compiler README, `docs/implementation/cli.md`, `docs/implementation/completions.md`, `std/map/README.md`, current `compiler/internal/catalogue/catalogue.json`, `compiler/current_types.go`. Historical links in those documents were not followed. Some operational guides contain stale task-stage prose; conclusions were verified against implementation rather than trusting it.

## Commands, outputs, limitations

1. `git rev-parse HEAD` => `961f921a6a8cf40be54735683caf29613c19cbd8`.
2. `go build -o /tmp/canlc-core-961f921 ./compiler && go test ./compiler/internal/check ./compiler/internal/types ./compiler/internal/syntax ./compiler/internal/emit` => success. Go printed a nonfatal module-stat-cache permission warning during build. First three test packages were cached; emit ran. Native emitter tests that require CAN_BUN may skip in this first command.
3. `CAN_BUN=$(command -v bun) go test ./compiler/internal/emit -count=1` => success, 13.720s, enabling native emitter execution tests.
4. `bun test ./runtime/primitive.test.ts ./runtime/data.test.ts ./runtime/completion.test.ts ./runtime/test/number.test.ts ./runtime/test/text.test.ts ./runtime/test/array.test.ts ./runtime/test/collections.test.ts ./runtime/test/exact-amount.test.ts` => 43 pass, 0 fail, 19,271 expectations across 8 files. An earlier version without `./` also selected mirrored distribution test files (172 pass); repeated with exact local paths so the stated focused result is current-source only.
5. `bun run check:runtime` => lint, formatting check, TypeScript check success. No runtime edits, so no fix/format mutation needed.
6. Temporary `TestCoreBlindProbe`, calling existing `programFixture` with complete independent source strings: private generic composition pass; exported generic composition rejection shown above; inline arithmetic pass; named final local rejection shown above; true-first Boolean branch rejection shown above. Helper removed after test. These were direct full program checks, unlike inspect-types.
7. Temporary `TestCoreBlindExamples`: full `project.Load` + `CheckProgram` succeeded for gallery (40 functions, 74 assertions), map (12/19), ratio (7/23), scalars (19/40), text (28/45). Helper removed. This validates current example checking, not execution of every attached assertion in the CLI supervisor.
8. Temporary `TestCoreBlindRecursionProbe`: parsed/checked complete `/tmp/can-core-countdown.can`, emitted countdown with real RegionEmitter and CompletionImports, then ran emitted source with Bun 1.4.2. Exact output above. A first attempt to print failure details read `result.failure`, an incorrect harness field; corrected to actual `result.value`, yielding the recorded native exception. No production code changed.
9. `bun /tmp/can-core-regex-probe.ts` => output above; successful runtime path uses dummy error-domain object because no error branch is reached. This isolates text scanning behavior, not catalogue wiring. Existing compiler/std tests establish public `text::compile_regex`/`text::matches` binding.
10. `inspect-types examples/gallery` succeeded but is explicitly only declaration inspection (`compiler/current_types.go`); not counted as body validation. An initial `inspect-types --help` yielded a path error because this subcommand has no such flag.

No complete application load test, browser UX test, all-project test run, or performance benchmark was performed. No claim of exhaustive soundness proof. Near-rename, hidden owner equality, generic effects and collection asymptotics are source-supported design consequences; generic export, local style, Boolean order, recursion depth and regex behavior were directly probed. The final worktree showed no changed tracked production files; another agent's untracked review-evidence directory was left untouched.

Common main used in generic/style program probes:

```can
fn void main
    emits []
    given
        str[] args
    asserts
        sample: [] => ok
    ok
```
