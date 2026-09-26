# Evidence packet R05–R08 — Can at `2cb1bc3` (2026-09-26)

Baseline: repo at `2cb1bc35435ab5075d51f9f945adfb0e1f9dfe64`, working tree clean except this round's untracked docs. Baseline equals reviewed revision (zero code drift), so all retained review evidence stays valid. Toolchain confirmed present: Bun 1.4.2, Go 1.27.1 darwin/arm64. Original probe scratch tree still present under `/tmp/can-review-core-20260926`. No repo files written; no probes run (justification per topic below).

---

## R05 — Iteration and collections (F-R05-01..05)

### (1) Best supported idiom + actual limitation

- **Existing collections:** use native array combinators — `map`, `filter`, `fold`, `find`, `some`, `every`, `for_each`, `sort_by` ([array.go](/Users/vince/Projects/can-lang/compiler/internal/check/array.go:112)). Checker: `arrayStep` ([array.go](/Users/vince/Projects/can-lang/compiler/internal/check/array.go:119)); runtime lowers to native promise-chained reduction, e.g. `fold` via `source.reduce(...)` over `Promise.resolve` ([array.ts](/Users/vince/Projects/can-lang/runtime/collections/array.ts:134)). Sequential, callback failure aborts with no partial result (tech-spec C-callback trace, `technical-spec.md:403`).
- **Dynamically continuing work** (state machines, pagination, polling): no supported constant-stack form. Only idiom is `relay call` self-recursion, which is **completion forwarding, not tail-call elimination** ("`relay` forwards completion; its name does not establish tail-call elimination", post-upgrade-language-review-961f921, ¶100; relay emits `return <call> as $canCompletion<...>` with no loop conversion — [regions.go](/Users/vince/Projects/can-lang/compiler/internal/emit/regions.go:584)).
- **Limitation (measured):** checked sync tail-recursive countdown `count(remaining-1, total+1)` passes at 100 steps, fails at 20,000 with `standard native_exception`, `RangeError: Maximum call stack size exceeded` on Bun 1.4.2. Retained: source `core-probes/recurse-large/src/main.can`, emitted line `$canRegion1 = await $canInvoke(() => ... $canFunction0 ...)` (generated `.ts:436`), logs `recurse-small-runtime-v3.log` (`ok`) vs `recurse-large-runtime-v3.log` (stack trace alternating `$canFunction0` ↔ `invoke` at `runtime/completion.ts:95`, where `call()` runs synchronously before any await — [completion.ts](/Users/vince/Projects/can-lang/runtime/completion.ts:90)).
- **Collection copy costs (source-confirmed, unmeasured):** every map `insert`/`replace`/`remove` does `new Map(source)` + mutation ([map.ts](/Users/vince/Projects/can-lang/runtime/collections/map.ts:56)); set `insert` does `new Set(source).add` ([set.ts](/Users/vince/Projects/can-lang/runtime/collections/set.ts:43)). Repeated insertion over n unique keys is O(n²) copy work. No bulk-build API (P18 deferred, retain copy-on-point-update).

### (2) Platform / primary-source facts

- Bun 1.4.2 (pinned, darwin-arm64) and V8/JSC-class engines: no guaranteed proper tail calls for this `async`/`await` thunk shape; the observed overflow is an emitted-program observation, not a language-spec threshold. Prior Sept-24 probe overflowed at 10,000 on the same class of machine — threshold is environment-dependent, and U06/P05 explicitly state **there is no portable numeric stack threshold**.
- `fold`'s promise-chain shape avoids native-stack growth (each step is a microtask continuation) but allocates O(n) promise nodes; memory behavior at 100k unmeasured.

### (3) Probes run

None. The 100-vs-20k claim is undisputed, fully retained (source + generated code + logs + harness `core-probes/probe_test.go` + `overlay.json`), revision-identical, and toolchain-identical. A rerun would only re-confirm an environment-specific overflow point the dispositions already decline to generalize.

### (4) Facts vs uncertainties vs counterexamples

- **Fact:** sync `relay` self-recursion grows the native stack; 20k sync steps overflow on Bun 1.4.2; `emits []` promises nothing about stack capacity (C9: stack exhaustion "not promised recoverable", `technical-spec.md` C9).
- **Fact:** no tail-loop conversion exists in `compiler/internal/emit` (audit grep, completion-audit ¶143).
- **Uncertainty:** exact overflow threshold on any other engine/version (known to vary: 10k vs 20k across runs); memory profile of 100k-step fold; behavior of recursion after genuine suspension (different scheduling; unprobed).
- **Counterexample / limit of claim:** probe covers only the synchronous tail-relay shape. Async recursion after a real suspension point and array-fold traversal are explicitly *not* claimed to fail; folds over existing arrays are the supported stack-safe path today.

### (5) Guarantees to preserve

Left-to-right awaited evaluation, exactly-once argument evaluation, callback error identity (a failing callback aborts the combinator; caller handles the declared set), fixture/diagnostic paths through iteration, immutability of user values.

### (6) Open questions

- **Technical decision:** self-tail lowering vs a small immutable-state iteration primitive vs documented scope limit (P05 reopening: compare on pagination/state-machine work incl. failures, fixtures, diagnostics; 100k-step acceptance with measured stack/memory). Bulk map/set construction contract (P18: collisions, order, callback errors, ownership) is a separate technical + library decision.
- **No user syntax choice** unless the iteration primitive introduces new surface syntax; relay's forwarding meaning must not be silently redefined.

---

## R06 — Generic failure composition (F-R06-01..05)

### (1) Best supported idiom + actual limitation

- **Idiom today:** (a) fixed finite `emits` bounds per helper (couples helper to consumer domains, may overstate per-caller errors); or (b) **result-as-data**: nominal `completed<item>` / `rejected<failure>` records + `outcome<item,failure>` variant, callable returning the variant with `emits []`, `match` + `relay call operation()` on rejection. Retained working probe: `core-probes/result-data-generic/src/main.can:11-21`, five assertion roots pass (`result-data-generic-assert-v2.log`: retry pass/fail, succeed/fail units, main).
- **Limitation:** authored `emits` cannot quantify over an error set. `fn item retry<item, failure>` with `emits [failure]` rejects `no eligible declaration for "failure"` (`error-generic-compile.log`; probe source `core-probes/error-generic/src/main.can:4-7`). Intentional: "There is no implicit trait search and no error-set type parameter; a bare variable in `emits` is not a nominal error" ([exported_generics.go](/Users/vince/Projects/can-lang/compiler/internal/check/exported_generics.go:33)). Meanwhile native collection helpers preserve callback-specific bounds (rejection tests [array_test.go](/Users/vince/Projects/can-lang/compiler/internal/check/array_test.go:48)) — an expressiveness gap between built-ins and authored libraries.
- **Result-data limits (reconciliation-narrowed):** retry expectations are `ok call succeed()` / `ok call fail()` (value factories, forced by a separate generic-inference limitation: concrete leaf expectations `ok completed(3)` reject with `generic assertion ... inference shape mismatch` — see `main.can.initial:16-17`). A no-retry implementation (delete second call, return first outcome) would pass the same assertions. **Retry count, failure-then-success sequencing, distinct payloads, and bounded attempts are unproven.** Next acceptance needs distinguishable outcome sequences + independent oracle.
- **Related:** authored error completions already work (`completions.go:349-361` checks `FailureBody` against declared bound); the invoice `env::required("")` startup-failure idiom is an obsolete-workaround docs defect (R16), not a language gap.

### (2) Platform / primary-source facts

- C4 (decisions.md SURFACE-077–078 + `technical-spec.md` C4): callable types spell exact error bounds; error-subset compatibility; finite catalogue callback specialization. `emits` is an authored upper bound on domain errors; standard failures always outside it (decisions.md `emits [...]` section; C9).
- No effect system, no error-set polymorphism in any shipped spec; DI-03/LD14 deferred, P04 deferred with reopening condition (two-domain + wrapper-around-wrapper comparison).

### (3) Probes run

None. Both the rejection (`error-generic-compile.log`) and the 5-root pass (`result-data-generic-assert-v2.log`) are retained, revision-identical, and undisputed. The open item (retry count/sequence) needs a *new* acceptance exercise at implementation time, not a rerun.

### (4) Facts vs uncertainties vs counterexamples

- **Fact:** public generics have real declaration-level symbolic checking (opaque type variables, explicit callable/dictionary inputs, acyclic nested instantiation) — not duck templates ([exported_generics.go](/Users/vince/Projects/can-lang/compiler/internal/check/exported_generics.go:15)).
- **Fact:** result-data generic checks, emits, executes; representation + value forwarding proven for deterministic callbacks.
- **Uncertainty:** conversion/provenance cost of wrapping native/SQL/HTTP emitted errors into data and back at scale; agent repair cost of fixed-bound vs result-data styles (no held-out trials run).
- **Counterexample to "impossible":** none claimed — fixed bounds and result-data both express retry; the gap is generality/concision, and built-ins already have the precise-bounds behavior users can't author.

### (5) Guarantees to preserve

Explicit finite `emits` on every declaration; checked propagation with no implicit domain-error flow; stable qualified error identity (C9); public-signature opacity rules (no leaked private types).

### (6) Open questions

- **Technical decision (R06):** make result-data concise first (conversion/provenance story + real reusable retry/composition adapter validated); evaluate finite error-set parameter syntax only if adapters remain extensive (review §5 ¶2; P04 reopening comparison). No broad effect system established as necessary.
- **User syntax choice** only if the finite-error-set parameter is adopted.

---

## R07 — Owner values and assertion setup (F-R07-01..04)

### (1) Best supported idiom + actual limitation

- **Limitation:** assertion rows are single-line `name: args => expected` with expression-only arguments ([declarations.go](/Users/vince/Projects/can-lang/compiler/internal/syntax/declarations.go:285)) and single-line expected completions; the checker builds a synthetic `relay`-root invocation ([assertions.go](/Users/vince/Projects/can-lang/compiler/internal/check/assertions.go:87)) that requires explicit completion handling. So `sample: call ids::parse(1) => ok 1` (fallible smart constructor for `owner record user_id`) rejects: `domain-fallible call requires explicit completion handling` (`owner-test-input-compile-v2.log`; sources `owner-test-input/src/app/main.can:9`, `src/ids/ids.can:1-16`). First assertion row mandatory ([declarations.go](/Users/vince/Projects/can-lang/compiler/internal/syntax/declarations.go:259)).
- **Valid workaround (source-described, not newly compiled):** private named fixture helper in the consumer package (`provides []` unchanged) that calls `ids::parse(1)`, handles `invalid`, returns the owner value; assertions use `call fixture_id()`. Owner boundary intact (no construction/decoding in consumer). Unexpected rejection aborts via a standard-fault path (today: deliberate arithmetic/indexing fault — works, poor readability; LD29's `checks::require(bool, str) -> void emits [checks::failed]` is the explicit-check contract but is a runtime check, not setup grammar).
- **Workaround costs:** extra function + its own mandatory assertion row; expected owner value often self-compares via the same producer (independent evidence must come from the owner package's literal tests + consumer projection tests); no test-only declaration facility established; fixture-failure expression is unreadable.

### (2) Primary-source facts

- LD29 design gate is **closed**: `checks::require` selected after three consultations; "No new standard category or assertion grammar is introduced" (decisions.md:46). A setup-region syntax "does not inherit from LD29" and needs an explicit user decision (F-R07-04).
- Fixture machinery that exists: typed fixture templates with local ownership (P3.1), attached native/wrapper assertions (P4.1), scenario links + fixture queues (`assertions.go` fixture table requires checked lexical call site, `assertions.go:117`); harness-supplied scope elision covers genuine opaque ingress/browser handles (`browser_elision_test.go`), not domain owners.

### (3) Probes run

None. The rejection probe is retained and undisputed; the reconciliation explicitly frames the remainder as authoring cost, and prescribes documenting/exercising the fixture-factory pattern before any syntax is considered. Nothing to rerun; the next step is a worked extraction example, which is new authoring work, not verification.

### (4) Facts vs uncertainties vs counterexamples

- **Fact:** inline fallible-constructor assertion input is rejected; private fixture helper is expressible within current grammar.
- **Fact:** owner confinement (no foreign construction/decoding) is correctly what blocks the naive row — the rejection defends a real boundary (`owner_record_test.go:111-149`).
- **Uncertainty:** measured authoring/repair cost of the factory pattern on a real handler→helper extraction (no such worked example exists yet); whether a clearer test-setup failure primitive suffices without grammar change.
- **Counterexample to "untestable":** none needed — impossibility was never claimed; the issue is cost/clarity.

### (5) Guarantees to preserve

Owner construction/representation confinement; mandatory attached assertions with executed roots; no forged owner values; `emits []` meaning (no escaping domain errors, not totality/purity).

### (6) Open questions

- **Technical decision:** document + exercise the private fixture-factory pattern on an extraction/refactoring example first; compare factories vs checked test-local setup; possibly add a small explicit test-setup failure mechanism. Setup-region *syntax* is a possible-syntax technical decision requiring explicit user approval (LD29 gate closed).
- **User syntax choice** only if setup-region grammar is proposed.

---

## R08 — Authoring policies and captures (F-R08-01..04)

Three retained rules, each with exact checker location and minimal examples. All confirmed present at this revision.

### Rule 1 — Boolean arm order (explicit user decision 2026-09-23; P15 Retain)

- **Rule:** in an ordinary single-scrutinee Boolean data match, `false` arm before `true` arm. Does not change branch selection; does not extend to completion/coordination matches, multi-scrutinee ordering, or native AI criteria (decisions.md:1649-1652).
- **Checker:** [completion_matches.go](/Users/vince/Projects/can-lang/compiler/internal/check/completion_matches.go:444) — tracks `seenTrue`, rejects `false` after `true` with `ordinary Boolean match requires false before true`.
- **Rejected** (retained probe `recurse-small/src/main.can.initial` + `recurse-small-compile.log`):
  ```can
  match remaining is 0
      true => ok total
      false => relay call count(remaining - 1, total + 1)
  ```
- **Accepted** (`recurse-small/src/main.can:12-14`): same arms with `false` first.
- Review proposal (C1): allow both orders, canonicalize via format/lint. **Requires user decision to reverse P15**; must not silently alter other match modes.

### Rule 2 — Local-elision (unnecessary-local) predicate (P14: retain narrow rule, defer demotion)

- **Rule:** decidable 4-clause predicate, "and nothing broader" (`technical-spec.md` C8, ¶"The unnecessary-local error"): (1) typed local immediately followed by terminal `ok name` (or as final value of a value-producing match arm); (2) exactly one direct use, never needed by implicit `near` capture; (3) initializer AST restricted to literals/names/parens/non-faulting field reads/primitive ops — excludes calls, constructors, array literals, match, indexing, division/remainder/power/shift, resources, `%`; (4) substituted check in terminal's expected type yields identical concrete type + variant conversion.
- **Checker:** [locals.go](/Users/vince/Projects/can-lang/compiler/internal/check/locals.go:84) (`countLocalUses` → single-use/no-capture test, `simpleLocalSyntax`, `nonFaultingLocalIR`, substitution re-check), diagnostic `unnecessary local <name> ...; replace with <expr>`.
- **Rejected** (retained `named-final-local/src/main.can:11-12` + log):
  ```can
  int subtotal = price * quantity
  ok subtotal
  ```
  Diagnostic: `unnecessary local subtotal at byte 162; replace with ok price * quantity`.
- **Accepted (rule leaves room):** bindings with two uses, `near`-captured bindings, initializers with calls/effects/indexing/division, bindings whose annotation pins inference (e.g. empty-array element type), match-arm values that aren't the narrow terminal shape.
- Review proposal (C2): allow meaningful final locals. **User authoring-policy choice** (P14 reopening: compare agent edits on meaningful-domain-local vs accidental-alias cases).

### Rule 3 — `near` name-based capture (LD38/LD39 + DI-08 retained; P13 defer syntax)

- **Rule:** `near <type> <name>` in `given` declares an input captured from the `callable <name>` creation scope; the surrounding immutable binding must have **exactly the declared name** and matching type; differently named bindings do not satisfy it; direct calls supply `near` inputs positionally like all others (decisions.md SURFACE-082, "Near inputs").
- **Checker:** [callables.go](/Users/vince/Projects/can-lang/compiler/internal/check/callables.go:127) — for each `Near[i]` input, checks `NameExpr{declaration.Names[i]}` (the **callee's** parameter spelling) in the caller scope; type must match exactly (`callables.go:140-141`); failures stamped `CAN-CHECK-CAPTURE`.
- **Accepted** (`compiler/testdata/current/callables/captures.can:15-46`): `combine` declares `near int prefix`, `near int suffix`; `compute` has immutable `prefix`/`suffix` in scope, so `callable combine` captures them; assertions supply near + ordinary positionally (`sample: 3, 4, 5 => ok 12`).
- **Rejected shape:** creating `callable combine` where the scope binds the value under a different name (e.g. `int base` instead of `prefix`) fails capture lookup even though types match; renaming the library's `near` parameter renames every capturing call site's obligation. Caller-side alias locals are the current workaround (expressiveness intact, rename cost real).
- Review proposal (C3): explicit site bindings + safe rename. **User syntax choice + technical packet** (P13 reopening: registered same-type rename/shadow repair comparison; show context records/diagnostics insufficient before grammar change). Interacts with R09 editor rename tooling.

### (2) Platform facts — R08

None platform-dependent; all three are compile-time checker policies. LSP advertises diagnostics + definition only ([lsp.go](/Users/vince/Projects/can-lang/compiler/lsp.go:156)); no rename/format support exists, which sharpens the `near`-rename cost but is an R09 tooling gap, not part of these rules.

### (3) Probes run — R08

None. All three rejections are retained as compiled negative probes with exact diagnostics, and the rules are source-verified at the identical revision. No behavior is disputed.

### (4) Facts vs uncertainties — R08

- **Fact:** all three rejections are intentional policies with recorded decisions, not compiler bugs; valid programs exist in each case (reordered arms, inlined expression, alias locals).
- **Uncertainty:** actual agent authoring/repair cost of each policy (zero held-out trials to date; token-efficiency unmeasured); whether formatter/lint canonicalization would preserve the policies' intent.

### (5) Guarantees to preserve

Exhaustive-match checking, `near` capture type-exactness, evaluation/typing preservation in any elision change, scoped-ness of the Boolean rule (must not leak into completion/coordination/multi-scrutinee modes).

### (6) Open questions — R08

All three reversals are **user syntax/authoring-policy choices** (C1–C3), not technical necessities. The technical contribution preparation owes is the P13/P14 reopening comparison evidence (rename/shadow repair trials; meaningful-local vs accidental-alias edit trials) so the user decides with costs measured rather than intuited.

---

## Cross-topic notes

- **Existing guarantees spanning R05–R08:** immutability, explicit finite `emits`, owner confinement, left-to-right awaited evaluation with exactly-once arguments, checked generics with symbolic public proofs, mandatory executed assertions, native lowering with contract-preserving adapters only. No proposal above weakens these without a re-decision.
- **No-compat standing rule** (AGENTS.md): old spellings/diagnostics need not be preserved if the user revises a policy.
- **Why no reruns:** every claim reused here is either (a) a retained executed probe at the identical revision + toolchain, undisputed and with stated limits, or (b) a source-verified checker/runtime fact re-read at HEAD. Per the reuse rule (P02.3), rerun only on changed code, missing coverage, or uncertainty — none applies.