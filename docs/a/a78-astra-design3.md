## Decision

**Build six separate closures in this order: constants → forward arms → ranges → or-patterns → boolean operators → unary minus.** Ranges depend on constants; or-patterns depend on ranges. Forwarding retains a61’s design rather than introducing a different error mechanism.

Review-harness artifacts, not in repo: `sandbox:/mnt/data/a78-gap-closures-design.md`, `sandbox:/mnt/data/a78-gap-closures-design.zip` (migration bodies, diagnostic specifications, rerunnable probes).

The package targets commit `e87e5cb997aa12a7c9d54453270b00fcc6205e96`. It contains six individual records, per-phase gates and rollback instructions, thirteen proposed diagnostic allocations, complete replacement-body sketches, and an evidence map.

**Status: design-only. No compiler changes were made. Repository compiler, modcheck, gramcheck, and TypeScript gates were not run.** The independently executed design checks are reported below.

## 1. Named constants: closed data declarations, available across files

Choose a real, typed, revisioned declaration:

```can
const ascii__COLON: int rev 1 = 58
const ascii__UPPER_FIRST: int rev 1 = 65
const ascii__UPPER_LAST: int rev 1 = 90
```

The ASCII declarations live in a shared `std/ascii/ascii.can` provider. HTML imports them through ordinary pinned `uses` entries. This makes the actual magic-bound migration a cross-file acceptance proof, rather than demonstrating providibility only in a synthetic fixture.

**Constants are closed data, not compile-time functions.** V1 admits `int`, `str`, `dec`, `bool`, existing brands, finite acyclic records, and sequences of those types. Initializers may contain literals, constant references, record/sequence construction, and existing authorized seals. Calls, arithmetic, matches, state, indexing, and slicing are excluded. Bytes and variant-valued constants are explicitly outside this first slice.

References work in bodies, tests, expected payloads, `given` exchange arguments and outcomes, and contracts. Bool/string constants work as patterns immediately; integer patterns and constant range bounds arrive together in the range slice. This addresses a78’s finding that nameable values have their largest payoff in tests, not merely in implementation bodies.

Forward references are resolved through a checked dependency graph. Cycles, duplicate declarations, ambiguous providers, and shadowing are rejected. Brand initialization is checked under the **provider’s authority**; importing a constant neither grants sealing authority nor erases its nominal type.

**Emit inline literal/constructor data, not exported TS constants.** Source dependencies and revision pins still exist, but there is no runtime constant import, initialization order, or shared mutable JavaScript object. The relevant standing precedent is R11’s inline, used-only numeric support rather than a runtime dependency—not an imaginary existing decision about constants.

### Constant identity

The design uses a77’s separation of interface identity, proof freshness, and test evidence. That requires a **tagged R4 clarification**, because the original R4 wording says any code change needs a revision, whereas shipped a77 explicitly excludes bodies and tests from interface identity.

A constant’s semantic content is its expanded **typed value and nominal dependencies**, not its spelling. Fresh-name renames pair by identical owner, revision, and typed fingerprint. Ambiguous name permutations require an explicit rename relation in the trusted baseline; they are not inferred from a suspicious value swap.

Consequently, a coordinated rename is revision-inert, but a changed value cannot disappear with the name. Source lookup and pin checking remain explicit: this does not promise that an obsolete identifier will continue to resolve.

## 2. Forward arms: exactly a manual payload reconstruction

Keep a61’s proposed spelling:

```can
on html.invalid_url e => forward e
on Ok r => forward r
```

`forward` is allowed only as the direct RHS of its own call-outcome arm. Its operand must be that arm’s resolved, named payload binder. `_`, an enclosing binder, a field projection, or a reconstructed expression cannot be forwarded.

The checker elaborates it into the ordinary constructor:

```can
on html.invalid_url e => html.invalid_url(value = e.value)
```

For `Ok`, it reconstructs the complete success payload and runs the same return-type checks as the handwritten construction. For an error, the kind must remain identical and belong to the caller’s `emits`. This preserves a61’s “same checks, same union construction” requirement.

**A4107 is unchanged.** In particular, forwarding does not automatically certify an arm. The existing certificate applies only to an unshadowed, bound error arm of a local call that reconstructs its complete unchanged payload. Untaken `Ok` arms and imported-call error arms still require execution evidence.

The emitted shape remains the existing flat tagged union:

```ts
return { $can_kind: "html.invalid_url", value: e.value };
```

It is not replaced by an unchecked `return e`. The current generated code uses `$can_kind` and flat payload fields.

### Two important migration details

The pinned source has **36 identity reconstructions**, not 35:

| Worker                        | Reconstructions |
| ----------------------------- | --------------: |
| `html__url__authority`        |              12 |
| `html__attribute__value_from` |              10 |
| `html__attributes__make_from` |               6 |
| `html__url__scheme_token`     |           **8** |

Scheme-token has two first-character recursive sites and six continuation sites; the seven-site receipt omits one. The package includes a checkout inventory script to rerun this count.

Conversely, this arm in `html__attribute__id` **must remain manual**:

```can
on html.nul_byte e => html.nul_byte(value = value)
```

It selects the enclosing input `value`, not `e.value`. Matching test values do not establish structural identity; the negative probe uses different strings to expose the difference.

## 3. Range arms: exact integer partitions, constant bounds in v1

Choose integer singletons and closed ranges:

```can
ascii__COLON => ...
ascii__UPPER_FIRST..ascii__UPPER_LAST => ...
```

Bounds are inclusive. A range requires `lower < upper`; equal endpoints use the singleton spelling. Bounds must be integer literals or visible integer constants—never variables, calls, or expressions.

This deliberately extends a28’s bool/string/wildcard slot fragment. It does not quietly reinterpret its deferral: a28 explicitly deferred OR-patterns, guards, and multi-call; ranges extend the admitted value domain, while the separate OR record reopens that explicit deferral.

### Proof rule

For every integer slot, collect exact arbitrary-precision cut points:

```text
lower
upper + 1
```

These partition the whole integer domain into finite intervals and two unbounded tails. Pattern membership is constant within each resulting atom. Feed those atoms into the existing symbolic product-coverage machinery; do not enumerate integers, narrow to machine integers, or infer that an `int` scrutinee is restricted to ASCII.

For arm \(i\):

```text
useful(i) = space(i) minus all earlier arm spaces
uncovered = total product minus all arm spaces
```

Compilation requires `uncovered` to be empty. A `_` in one product row does not establish coverage elsewhere:

```can
0..9, true => ...
_, false => ...
```

still misses `(-1, true)`.

Partial overlaps are legal under first-match semantics. After `1..10`, an arm `5..15` is useful over `11..15`; a completely shadowed arm is rejected by the new unreachable-arm diagnostic.

**One range arm remains one A4107 obligation.** Boundary testing is not silently promoted into a new universal language coverage law. Nevertheless, boundary and neighbor rows are mandatory acceptance fixtures for these two stdlib migrations, with inputs and expected answers written independently of the range constants.

The canonical `n <= 0` guard and `n - 1` recursive step remain unchanged. A range-based substitute does not acquire a new termination certificate.

## 4. Or-patterns: one RHS, but every written alternative needs evidence

Choose `|` inside a single pattern slot:

```can
ascii__SLASH | ascii__QUERY | ascii__HASH => ...
```

Ranges bind tighter than `|`; `|` binds tighter than the comma between slots. V1 combines nonbinding scalar patterns: bools, strings, integer singletons/ranges, and suitable constants. No wildcard inside an alternative list, variant binders, or call-outcome alternatives.

**Coverage has two layers:**

* A4107 remains one obligation for the source arm.
* Every explicitly written alternative in every slot requires a passing test that selects it in the winning arm.

This is not a Cartesian-product coverage requirement. But `1..10 | 5..15` requires a witness in `11..15` for the second alternative: input `7` cannot credit both. An alternative whose effective selection region is empty is rejected statically.

That preserves the purpose of a78’s complaint: reducing handwriting must not make a mistyped delimiter invisible to the test law.

### Authority’s four terminator checks cannot be flattened naively

This rewrite is rejected:

```can
match n <= 0, s[0]
```

It indexes even when the base branch should return. Nor is `n <= 0` equivalent to an empty string for direct worker inputs. The source also distinguishes end-of-input from delimiter stops: an empty `prev` rejects at end but is not rejected by those delimiter arms.

The chosen migration uses one ordinary, explicitly tested helper:

```can
match at_end, prev
  true, "" => html.invalid_url(value = orig)
  _, "-" => html.invalid_url(value = orig)
  _, _ => Ok(value = orig, tail = s, n = n)
```

The base branch calls it with `at_end = true`; the grouped slash/query/hash arm calls it with `false`. Thus the four hyphen-terminator decisions become **one helper arm**, without moving indexing across the base guard.

The helper’s three arms and six own test rows are included in the migration cost. They are not hidden outside the receipt.

## 5. Boolean operators: strict keywords with eager target lowering

Choose:

```can
and
or
not
```

Reject `&&`, `||`, and `!` as alternative source spellings. Reclaim the grammar’s orphaned `and`; retain `!=` as inequality.

Precedence, low to high:

| Level | Operators                                    |
| ----- | -------------------------------------------- |
| 1     | `or`                                         |
| 2     | `and`                                        |
| 3     | Prefix `not`                                 |
| 4     | Existing comparisons                         |
| 5     | Binary `+ -`                                 |
| 6     | `* / %`                                      |
| 7     | Prefix numeric `-`                           |
| 8     | Existing tighter length/postfix/atomic forms |

Thus `not a == b and c` means `(not (a == b)) and c`. Existing comparison associativity is preserved rather than changed incidentally.

Both operands must be bool. Evaluation is left-to-right, and the right operand evaluates whenever the left returns normally, regardless of its value.

The well-typed fault counterexample is:

```can
false and ((1 / 0) == 0)
```

**It must fault loudly.** The simpler `false and (1 / 0)` is ill-typed, not a short-circuit test. Primitive faults do not become typed `emits` outcomes, and a boolean conjunction is not a domain guard for indexing or division.

Bare target lowering to `emit(left) && emit(right)` would violate that decision. Use eager helper arguments:

```ts
function $canAnd(left: boolean, right: boolean): boolean {
  return left && right;
}

$canAnd(leftExpression, rightExpression)
```

Both source computations occur before the helper body. Emit these helpers only when used. No source call operands, thunks, closures, or early-return syntax are introduced.

The ASCII predicate row follows as three separate stdlib slices: alpha, digit, then alnum. They use the shared constants and the existing `Bool__Value` wrapper. Alnum composes explicitly bound call results:

```can
match call std__ascii__is_alpha(code)
  on Ok a => match call std__ascii__is_digit(code)
    on Ok d => Ok(value = a.value or d.value)
```

Calls have not become legal operands.

## 6. Unary minus: prefix `-`, no unary plus

Add prefix minus for `int` and `dec`, binding tighter than multiplication and binary subtraction. The parser must handle both `a * -b` and `a - -b`; adding only a leading-minus branch is insufficient.

Existing `-3` and `d"-0.5"` retain their literal behavior and emitted bytes. Negated literal spellings normalize back to those existing literal forms.

For integers, emit native bigint negation. **For decimals, do not emit JavaScript `-x`: decimals are canonical strings.** Use an exact sign-toggle helper that preserves canonical zero and never converts through `Number`.

The migration changes the scale argument in `std__dec__divide_round_half_even_result` from `0 - scale` to `-scale`. It does not refactor the separate round-half-even `let` exhibit.

## Migration receipt

These counts include every source match arm, including call-outcome arms. The proposed body counts are checked by the included scripts; they are not claims that a new compiler accepted those bodies.

| Exhibit                | Pinned arms / own rows | After ranges and forwarding |    After OR |
| ---------------------- | ---------------------: | --------------------------: | ----------: |
| Scheme-token           |                40 / 14 |                     23 / 38 | **11 / 38** |
| Authority              |                54 / 26 |                     40 / 38 |     24 / 38 |
| New authority helper   |                      — |                           — |       3 / 6 |
| **Authority combined** |                54 / 26 |                     40 / 38 | **27 / 44** |
| Value-from             |                22 / 12 |                 **19 / 12** |     19 / 12 |

Value-from’s five-arm string match is explicitly a migration acceptance task using already available syntax—not falsely credited as a new range feature. Its outer fuel and NUL guards remain intact.

## Diagnostics, tooling, and execution gates

The package fixes thirteen proposed codes, with message templates, payload strings, source anchors, minimal violations, and golden paths:

| Item                            | New codes                                                      |
| ------------------------------- | -------------------------------------------------------------- |
| Constants                       | `CAN2003`, `CAN2205`, `CAN2401`–`CAN2405`                      |
| Forward                         | `CAN3011`                                                      |
| Ranges                          | `CAN4110`–`CAN4112`                                            |
| OR                              | `CAN4113`, `CAN4114`                                           |
| Boolean operators / unary minus | Reuse existing parse, type, call, fault, and termination codes |

One additional prerequisite emerged from the actual source: **LSP publication currently omits codes and payload fields**, despite carrying diagnostics internally. The plan therefore includes a small transport fix and actual wire-level goldens. Comparing internal `Diag` arrays alone would not satisfy CLI/LSP parity.

Every feature has its own design → grammar → parse → proof/check → eval → emit → diagnostics → editor grammar → goldens → migration → tagged-docs phases, with rollback instructions. The final gates include:

```sh
go test -count=1 ./...
go run ./tools/modcheck
go run ./tools/gramcheck

(cd tscheck && npm ci --no-audit --no-fund &&
  ./node_modules/.bin/tsc -p tsconfig.json)

go test -count=1 ./compiler -run '^TestA78[CFROBN]'
go test -count=1 ./compiler -run '^TestA78.*Runtime'
```

The new runtime tests must execute **actual generated output**, not merely type-check it. TextMate cleanup deliberately removes the fictional vocabulary, then reclaims only the tokens that ship.

### Checks actually executed

| Independent design check                    |        Result |
| ------------------------------------------- | ------------: |
| Scheme old/new model vectors                | 99,090 passed |
| Authority old/new model vectors             | 71,442 passed |
| Integer/bool product tables                 |  2,000 passed |
| Huge-integer translations                   |  4,000 passed |
| Negative controls                           |      6 passed |
| JS eager-lowering and exact-negation checks |     20 passed |

These checks include inconsistent worker inputs and loud-fault behavior. They **do not establish compiler correctness**.

The one named source-integration limitation is the future `ai-lock.json` reader/writer and accepted schema: that implementation was not established by the retrieved sources. The current accepted-baseline design is fully specified, and the future lock writer is required to use the same resolved canonicalizer—not a second raw-source hash.

Guards beyond ranges, multi-call, higher-order operations, record update, table lookup, and `let` remain outside this package.
