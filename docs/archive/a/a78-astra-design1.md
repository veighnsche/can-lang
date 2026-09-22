**The design is six separate builds, ordered: constants → forward arms → range arms → or-patterns → unary minus → boolean operators.** Constant bounds ship with the first range implementation; ASCII predicate functions follow boolean operators as a separate stdlib change.

Review-harness artifacts, not in repo: `sandbox:/mnt/data/a78-design-package.zip`, `sandbox:/mnt/data/a78-design-records-and-plan.md`.

The package contains six decision records, grammar and proof deltas, proposed diagnostic goldens, migration bodies, per-item phased plans, and rerunnable probes. It is pinned to **`7c33f7cd980c667aa39bbfa4848ccff6759370f6`**. No repository files, compiler code, branches, commits, or accepted baselines were changed.

## Corrections that affect the plan

Several commissioning assumptions need explicit amendments rather than implementation against stale evidence.

**`case` is not dead grammar.** The inspected parser accepts variant declarations and case rows. The cleanup preserves `case`, removes the genuinely obsolete vocabulary, and reclaims `|` and `and` only when their implementations land. See `compiler/parse.go:1322–1365`.

**The requested relay counts do not match the inspected source.** `scheme_token` contains eight self-call/Ok relays, not seven. The four named, measurable candidates are authority **12**, attribute-value **10**, text-escape worker **8**, and scheme worker **8**: **38 identity reconstructions**, not 35. These are source shapes, not claims that all 38 qualify for structural coverage certificates.

**Native JavaScript negation is incorrect for `dec`.** Decimals emit as canonical strings; unary minus must use exact decimal lowering rather than coerce them to JS numbers. See `compiler/emit.go:11–15,500–510`.

The package names one unresolved **evidence** item: **U1**, the missing round-2 inventory mapping `12 + 10 + 6 + 7` to four qualified worker names and a commit. No syntax or semantic decision is left waiting on it.

## The six decisions

| Item                  | Chosen design                                                                                                                                                                       | Deliberately rejected                                                                                                |
| --------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| **Named constants**   | Top-level `const Name: Type rev N = literal`; `int`, `str`, `dec`, `bool`; providable and explicitly pinned; literal substitution everywhere required, including tests and `given`. | Computed initializers, aliases as initializers, aggregates, brands, runtime initialization, zero-argument functions. |
| **Forward arms**      | `on K e => forward e`, including `Ok`; exact reconstruction from the current arm’s named payload binder.                                                                            | `_`, another binder, projections, mapping, retagging, dropping fields, returning the incoming union directly.        |
| **Range arms**        | Integer singleton or inclusive `low..high`; signed literals and integer constants in v1; exact arbitrary-precision domain analysis.                                                 | Open or half-open ranges, decimal ranges, computed bounds, implicit guards.                                          |
| **Or-patterns**       | Per-slot `p1 \| p2`; binder-free scalar patterns, including ranges and constants; one source arm and one RHS.                                                                       | Outcome/variant alternatives, binder unification, wildcard alternatives, alternative tuple syntax.                   |
| **Unary minus**       | Prefix `-` on `int` and `dec`; existing signed literals unchanged.                                                                                                                  | Unary `+`, numeric coercion, alternate termination-step spellings.                                                   |
| **Boolean operators** | Keyword `and`, `or`, `not`; strict left-to-right evaluation; eager helper lowering.                                                                                                 | Symbolic aliases, short-circuiting, operand calls, truthiness, fault-suppressing optimizations.                      |

### Constants: test scope is part of the feature

The canonical form is:

```can
const Html__AsciiColon: int rev 1 = 58
```

A foreign consumer imports it through `uses [Html__AsciiColon@1]`. References work in bodies, test inputs, expected outcomes, `given` arguments and outcomes, supported patterns, and contract expressions. A constant used only in evidence still makes its import used.

The compiler substitutes checked literal values; it emits **no runtime constant or import dependency**. This adopts the repository’s inline-emission precedent, without pretending that precedent already settled the constants design.

Two details prevent shortcuts:

* Evidence resolves constants in its **owning module**, not whichever function environment happens to be active.
* The ranking checker retains original syntax/provenance. Substitution must not make `n - One` another spelling of the certified `n - 1`.

The existing termination rule is deliberately syntactic, including the canonical guard and unit step. See `compiler/check.go:1320–1464`.

### Forward: identical construction, identical evidence rules

```diff
- on html.invalid_url e => html.invalid_url(value = e.value)
+ on html.invalid_url e => forward e
```

Forward elaborates to the same complete constructor and passes ordinary payload, type, authority, and `emits` checks.

For `Ok`, the source payload must have exactly the destination success field set and resolved field types. Different wrapper-record names are not automatically disqualifying: the operation remains explicit field reconstruction, not a nominal record cast.

This mapping remains manual:

```can
on html.nul_byte e => html.nul_byte(value = value)
```

Here, `value` is the outer input—not `e.value`. Replacing it requires a dataflow equality argument that this feature does not introduce. The distinction is visible in `html__attribute__id`.

**Coverage does not become weaker.** Only the existing local-error identity certificate remains available. Untaken `Ok` and foreign-call forward arms do not acquire certificates; certified arms are never reported as executed. This preserves a61’s explicit-arm design and the current `relayStatus` boundary.

### Ranges and or-patterns: prove pattern spaces, preserve source arms

Ranges use exact integer partitions, including unbounded tails. For a product arm, its effective space is its pattern space minus all earlier arms.

The selected rules are:

* **First match wins.** Partial overlaps are legal.
* A completely covered later arm is rejected.
* Every or-alternative must contribute some space after earlier arms and earlier alternatives in its slot.
* Exhaustiveness requires the entire typed product to be covered—not merely the presence of an underscore somewhere.
* The existing typed value wildcard remains available; no error or variant catch-all is introduced.

**CAN4107 stays one obligation per source arm.** A range endpoint is not an arm, and an alternative sharing the same RHS is not a second arm. Boundary and per-alternative examples are mandatory for the named migrations, but are not silently promoted into a new language-wide coverage law. The current checker already distinguishes execution evidence from authorized structural certificates.

The proof implementation must not confuse a witness-reporting or optimization budget with successful exhaustiveness. The current final-`else` optimization already consumes a proof result rather than inventing one in emit; the new domains retain that separation.

### Boolean operators: strict even when the result is already known

The correctly typed fault probes are:

```can
false and ((1 / 0) == 0)
true or ((1 / 0) == 0)
```

Both fault loudly. The prompt’s `false and (1 / 0)` instead fails typing because its right operand is an integer.

This follows the repository’s distinction between returned typed errors and loud primitive-domain faults. Boolean syntax must neither suppress the fault nor convert it into an `emits` outcome.

The chosen lowering is:

```ts
function $canBoolAnd(left: boolean, right: boolean): boolean {
  return left && right;
}

$canBoolAnd(lhs, rhs);
```

Both argument expressions evaluate before the helper combines their values. ECMAScript evaluates preceding arguments before subsequent argument expressions. Direct `(lhs && rhs)` is rejected because parentheses do not make it strict. ([TC39][1])

Consequently, `(n > 0) and (s[0] == 65)` is **not an indexing guard**. Domain-dependent conditional evaluation still needs an explicit exhaustive match.

### Unary minus: small surface, exact lowering

```diff
- Ok(value = 0 - value)
+ Ok(value = -value)
```

For integers, emit `(-value)`. For decimals, emit `$canDecSub("0.0", value)`. Preserve `-3` and `d"-0.5"` literal behavior, reject unary plus, and keep `n + -1` outside the canonical recursion-step rule. The named migration is `std__int__negate`, retaining its three existing rows.

## Migration results and the authority trap

The package includes replacement bodies, not just migration promises.

| Exhibit                       | Original source arms | After ranges |        After or-patterns |                       Planned committed rows |
| ----------------------------- | -------------------: | -----------: | -----------------------: | -------------------------------------------: |
| `html__url__scheme_token`     |                   40 |           23 |                   **11** |            **62**: 14 retained + 48 boundary |
| `html__url__authority`        |                   54 |           40 | **27**, including helper | **56**: 26 retained + 24 boundary + 6 helper |
| `html__attribute__value_from` |                   22 |       **18** |                       18 |         **19**: 12 retained + 7 direct cases |

These are source-arm derivations from the inspected and proposed bodies, counting nested outcome arms but excluding test rows. Actual AST/catalog confirmation remains an implementation gate. The original bodies and rows are in the pinned HTML source.

**The four authority terminator checks cannot simply become one eager EOF/character match.** Reading `s[0]` at EOF would introduce a fault. There is also a behavioral distinction worth preserving: empty `prev` is rejected at EOF, but accepted at a live delimiter.

The selected migration adds one local, tested finish helper:

```can
match n <= 0, prev
  true, "" => html.invalid_url(value = orig)
  _, "-" => html.invalid_url(value = orig)
  _, _ => Ok(value = orig, tail = s, n = n)
```

It is called from the existing EOF branch and the merged live-delimiter branch. Thus the four hyphen decisions become one, while the recursive worker retains its original single-scrutinee count guard. The helper adds explicit calls and six tests; the plan does not describe that extraction as zero-cost.

## Diagnostic and revision contracts

Six new diagnostic allocations are specified against the pinned registry. Each has a complete proposed source fixture and JSON golden, including message, span, and `Expected`/`Found`/`Hint`.

| Code        | Rule                                                     |
| ----------- | -------------------------------------------------------- |
| **CAN2205** | Constant declaration collision                           |
| **CAN6014** | Constant initializer is not a scalar literal             |
| **CAN3011** | Forward operand is not the current arm’s payload binder  |
| **CAN4110** | Invalid integer range bounds                             |
| **CAN4111** | Earlier arms completely cover this arm                   |
| **CAN4112** | An or-alternative contributes no remaining pattern space |

Existing type, call-position, pin, termination, coverage, and identity rules keep their existing codes. The new codes are **specified in the package, not registered in the repository**.

Revision handling follows a77’s separation of interface identity, proof freshness, and test evidence—not a blanket API bump for every body edit.

The package makes the difficult cases explicit:

**Constant renaming is revision-inert, not value-inert.** Inventory comparison first matches unchanged names, then pairs remaining equal owner/revision/type/value entries as renames. Swapping values under existing names cannot masquerade as a rename. References and import spellings must still be updated coherently.

**Formatting is inert; structure is not.** Forward normalizes to its manual reconstruction. Ranges and alternatives retain ordered semantic structure. Boolean operands are never sorted or folded in a way that erases faulting evaluation.

Canonical formats advance through **2–6**, with forward sharing format 2. Every transition requires a reviewed baseline bridge; ordinary checking refuses incompatible formats rather than silently rehashing or accepting the candidate as its own authority.

## Validation and build gates

The supplied standalone models were executed successfully:

| Executed check                                     |                    Result |
| -------------------------------------------------- | ------------------------: |
| Two ASCII classifier models                        | **1,112,067 inputs each** |
| Old/new worker-model comparisons, including faults |               **170,381** |
| Deterministic two-slot interval/or oracle cases    |                 **1,000** |
| Strict boolean helper assertions                   |                    **20** |
| Shared range-endpoint ±1 mutants detected          |              **12 of 12** |

One useful counterexample emerged: extending digit-last from 57 to 58 is masked in the scheme worker by its earlier colon arm. The combined authority corpus detects it through colon rejection. The mutation gate therefore runs both consumers rather than demanding that every mutation alter the scheme result.

These are **transcribed models, not compiled CAN equivalence proofs**. Their receipts and limitations are included in the package.

Each build has its own phase-by-phase work, gates, and rollback unit. Required implementation gates include:

```sh
go test -count=1 ./...
go run ./tools/modcheck
go run ./tools/gramcheck

(cd tscheck &&
  npm ci --no-audit --no-fund &&
  ./node_modules/.bin/tsc -p tsconfig.json)

# New suites required by this plan:
go test -count=1 ./compiler -run '^TestA78.*Runtime$'
```

The runtime suites must actually contain fixtures and fail—not skip—when required tooling is missing. The plan also retains explicit negative probes for ranking shortcuts, hidden calls, unsupported canonical nodes, false coverage certificates, and fault-order changes.

**No compiler gates were run:** no Go build/test, CAN compilation, modcheck, gramcheck, tsc, emitted-CAN runtime suite, or accepted-baseline comparison. No green compiler build is claimed.

[1]: https://tc39.es/ecma262/multipage/ecmascript-language-expressions.html "https://tc39.es/ecma262/multipage/ecmascript-language-expressions.html"
