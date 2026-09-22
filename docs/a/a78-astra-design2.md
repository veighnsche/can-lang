**Recommended build order: constants → forward arms → ranges → or-patterns → boolean operators → unary minus.** These are six separately reviewable builds; or-patterns are a distinct reopening of the existing deferral.

The designs are pinned to commit `b5525eac54e563fdabdb41a1bc98bb66b10f35e0`.

Review-harness artifacts, not in repo: `sandbox:/mnt/data/a78-design-and-implementation-plan.md`, `sandbox:/mnt/data/a78-design-bundle.zip`.

The bundle includes six decision records, per-item phased plans, complete migration sketches, nine proposed diagnostic fixtures, boundary-test rows, and rerunnable design probes. **No compiler changes were made, and no repository compiler gates were run.**

## Decisions

| Item              | Chosen design                                                                                                                                                                    | Explicitly rejected                                                                                                             |                                                                                                                  |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| Named constants   | Typed, revisioned, providible scalar-literal declarations; available in bodies, tests, expected values, `given`, and permitted patterns. Inline their resolved literals into TS. | Functions disguised as constants; state cells; evaluable initializers; record, sequence, or brand constants in v1.              |                                                                                                                  |
| Forward arms      | `on Kind e => forward e`, as the entire immediate call-arm RHS. Includes compatible `Ok` payloads.                                                                               | `_`, outer binders, fields, constructors, inferred propagation, or a new coverage exemption.                                    |                                                                                                                  |
| Range arms        | Inclusive integer `lower..upper`; singleton integers; integer constant bounds in the first release. Symbolic interval/product proof.                                             | Computed bounds, open-ended range syntax, machine-width integers, and using ranges as new termination guards.                   |                                                                                                                  |
| Or-patterns       | `                                                                                                                                                                                | ` combines scalar alternatives within one slot; one source arm and one RHS. Static usefulness checks for arms and alternatives. | Payload-binding alternatives, variants/errors/`Ok`, wildcard alternatives, and mixed scalar types within a slot. |
| Boolean operators | Strict, once-only, left-to-right `and`, `or`, `not`. Binary operations emit through strict helper calls.                                                                         | Symbol aliases, short-circuiting, implicit truthiness, and calls inside operands.                                               |                                                                                                                  |
| Unary minus       | Prefix `-` for exact `int` and `dec`; preserve existing negative literal behavior.                                                                                               | Unary `+`, coercions, native numeric negation of decimal strings, and relaxed `decreases` recognition.                          |                                                                                                                  |

## Important corrections and constraints

### 1. The relay migration is 36 replacements, not 35

At the pinned source, `html__url__scheme_token` contains **eight** identity success relays: two in the first-character path and six in the continuation path. The requested seven is an inaccurate count. The six-relay worker is `html__attributes__make_from`. The migration receipt is therefore:

**12 authority + 10 value_from + 6 attributes__make_from + 8 scheme_token = 36.**

Sources: `std/html/html.can:175–214,484–543,551–636,923–950`.

Forwarding alone removes **no arms, calls, or test rows**. Its benefit is eliminating field-by-field retranscription.

### 2. Constants need an inventory rule, not merely name-insensitive hashing

The chosen form is:

```can
const html__ASCII_COLON: int rev 1 = 58
```

The constant’s primitive type and decoded literal value are semantic content. Its spelling is not. References serialize as resolved typed literals, so renaming a constant does not change the consuming expression’s semantic content.

However, the existing revision inventory includes names in its keys. Literal expansion alone would therefore leave a rename-sensitive inventory comparison. The record specifies a **constant-only, trusted inventory relocation**, preserving owner, revision, type, and value. It requires no revision bump and does not guess relationships between equal-valued constants. Ordinary nominal declaration identities are unchanged. This is an explicit refinement of a77, not a silent reinterpretation of R4. Source: `compiler/revision.go:86–96`; the attachment expressly requires constant-name changes to be content-inert.  

Test scope is included in the first constant release. Repeated scalar values can be named inside existing nested constructors and `seal` expressions, but those constructors and seals remain in their original test context. **The design does not turn test-authorized brand construction into exportable production data.**

### 3. Forwarding preserves the existing certificate boundary

The current exception to execution evidence is narrower than “identity forwarding”: it applies to an unshadowed, bound **error arm of a local call** with a complete, unchanged reconstruction. It does not certify `Ok` arms or foreign-error arms. The design retains that exact boundary. Source: `compiler/lsp.go:405–513`.

Thus:

```can
on Ok r => forward r
```

still requires an actual passing test to take the arm.

The mapping in `html__attribute__id` stays manual:

```can
on html.nul_byte e => html.nul_byte(value = value)
```

It selects the outer parameter `value`, not `e.value`. The forwarding rule deliberately does not use relational reasoning or coincidentally equal test values to classify that as identity. Source: `std/html/html.can:425–443`.

### 4. Range and alternative proofs remain structural

For integer ranges, the proposed checker partitions the integer domain at each `lower` and `upper + 1`, using arbitrary-precision arithmetic and explicit unbounded tails. It then extends the existing symbolic product proof rather than enumerating integers.

For each arm:

```text
useful domain = arm domain − earlier arms
```

For the entire match:

```text
uncovered domain = full product − union of all arms
```

An empty useful domain is a static error. A nonempty uncovered domain is an exhaustiveness error, with concrete witnesses. The three-witness limit limits reporting, not proof. This extends the current symbolic product machinery in `compiler/eval.go:1900–2040,2056–2210`.

**A4107 remains one obligation per useful source arm**, not per range endpoint or alternative. Boundary rows are mandatory for these migrations, but are not introduced as a universal new language law.

The value-pattern `_` covers a proved scalar remainder. It never becomes a catch-all for missing call outcomes or nominal variant cases.

### 5. Four authority terminator checks cannot collapse through `|` alone

EOF is behind the fuel guard; `/`, `?`, and `#` inspect a character. Combining them this way would evaluate `s[0]` on empty input:

```can
match n <= 0, s[0]
```

The source keeps those computations on different control-flow paths. Source: `std/html/html.can:582–599`.

The proposed migration retains `match n <= 0`, groups the three character delimiters, and uses one ordinary local finishing helper:

```can
html__ASCII_SLASH | html__ASCII_QUESTION | html__ASCII_HASH => match call html__url__authority_finish(orig, s, n, prev)
  on Ok r => forward r
  on html.invalid_url e => forward e
```

Its body contains the single shared predecessor check:

```can
match prev == "-"
  true => html.invalid_url(value = orig)
  false => Ok(value = orig, tail = s, n = n)
```

The EOF path calls it **after retaining the existing empty-predecessor rejection**. That distinction matters: the existing delimiter paths do not perform that rejection.

The cost is explicit: two helper call sites, four outcome arms, and three helper test rows. This is **or-pattern grouping plus an ordinary helper extraction**, not a claim that alternatives alone can combine guarded EOF handling.

### 6. Strict booleans require more than parenthesized `&&`

Under the proposed operand types:

```can
false and (1 / 0)
```

is a type error—the right operand is an integer. The well-typed fault-contract counterexample is:

```can
false and ((1 / 0) == 0)
```

That must fault loudly, as must the corresponding `true or` expression. It must not become a typed returned error or a successful boolean result. This follows the repository’s distinction between typed outcomes and primitive domain faults. Source: `docs/fault-contracts.md:7–15`.

The proposed target shape is:

```ts
function $canBoolAnd(left: boolean, right: boolean): boolean {
  return left && right;
}

const result = $canBoolAnd(leftExpression, rightExpression);
```

Both arguments evaluate before the helper combines their values. The interpreter likewise evaluates and stores both child results before applying the truth table. A left fault still stops execution before the right operand; strictness is not execution after failure.

The ASCII predicate additions follow as **three separately gated stdlib operations**: digit, alpha, then alnum. Alnum composes explicit call results:

```can
match call std__char__is_alpha(code)
  on Ok a => match call std__char__is_digit(code)
    on Ok d => Ok(value = a.value or d.value)
```

No operand-call exception or multi-call match is introduced.

### 7. Decimal unary minus cannot emit native `-x`

The emitter represents `dec` as canonical digit strings and routes arithmetic through exact helpers. Therefore integer negation can emit `(-x)`, but decimal negation must reuse exact subtraction from decimal zero. Source: `compiler/emit.go:430–568`.

The unary record is deliberately small. Its migration changes only the `0 - scale` argument in `std__dec__divide_round_half_even_result`; it does not touch the shared-tail/`let` problem in `round_half_even`. Source: `std/scalars/scalars.can:1088–1150`.

## Migration receipt

These counts include **all nested source arms**, including call-outcome arms—not just classifier leaves. Proposed counts were checked against the supplied replacement bodies.

| Exhibit                 | Reviewed source arms |  After ranges / flatness |               After alternatives | Test rows                                                      |
| ----------------------- | -------------------: | -----------------------: | -------------------------------: | -------------------------------------------------------------- |
| `scheme_token`          |                   40 |                       23 |                               11 | Keep 14; add 20 boundary rows → **34**                         |
| `authority`             |                   54 |                       40 | 28, plus 2 in `authority_finish` | Keep 26; add 12 boundary rows → **38**, plus **3** helper rows |
| `value_from`            |                   22 |                       19 |                        Unchanged | Keep all **12**                                                |
| `attributes__make_from` |                   18 | Forward RHS changes only |                        Unchanged | Keep its **3** own rows and existing transitive evidence       |

The baseline bodies and rows are in `std/html/html.can:175–214,484–543,551–636,923–950`.

The boundary rows preserve consumer-specific behavior. For example, `/` terminates authority with an unchanged tail; `:` is accepted by scheme continuation but rejected by authority. They are not generated by assuming that every value outside an alphanumeric range should return false.

## Diagnostics and implementation gates

Nine new codes are proposed against the reviewed registry:

| Code      | Rule                                   |
| --------- | -------------------------------------- |
| `CAN2003` | Constant naming                        |
| `CAN2104` | Unknown constant                       |
| `CAN2105` | External constant missing from `uses`  |
| `CAN2205` | Duplicate same-module constant         |
| `CAN6014` | Nonliteral constant initializer        |
| `CAN3011` | Invalid forward placement or binder    |
| `CAN4110` | Fully unreachable new-form pattern arm |
| `CAN4111` | Redundant alternative                  |
| `CAN4112` | Reversed integer range                 |

These allocations were checked against `compiler/code.go:23–102`. The bundle supplies a complete proposed violating source world and a code/message/payload/span contract for each; they are **draft golden inputs, not captured compiler output**. Existing type, pin, coverage, termination, and fault codes are reused rather than duplicated.

Each implementation plan specifies the requested phases, exact gates, and rollback unit. Shared gates include:

```sh
go test -count=1 ./...
go run ./tools/modcheck
go run ./tools/gramcheck

(cd tscheck && npm ci --no-audit --no-fund)
(cd tscheck && ./node_modules/.bin/tsc -p tsconfig.json)
```

The plan also adds a required generated-TS runtime parity gate. Type checking alone cannot detect accidental short-circuit behavior.

The editor cleanup removes the documented dead vocabulary, then reintroduces `|` and `and` only with their implemented features. It does not treat the current regex-based gramcheck as a full TextMate precedence test; that limitation is explicit in its source. Sources: `editors/vscode/syntaxes/can.tmGrammar.json:1`, `tools/gramcheck/main.go:1–5`.

## Validation status

The independent design model passed **64,512** old/new worker-state comparisons, **1,000** interval-product comparisons, and checks of the proposed body counts. A handwritten JavaScript helper probe passed **13 assertions** on local Node **22.16.0**. Those checks do not establish compiler acceptance, A4107 correctness, termination certification, or generated-code parity.

**No `go test`, modcheck, gramcheck, repository tsc, or compiler-generated runtime gate was run.** The local clone failed because the runtime could not resolve GitHub; repository inspection used the connected GitHub tool.

One source limitation remains named in the plan: I located the `ai-lock.json` requirements and sketch references, but not an executable archival writer/historical-provider loader. The implementation plan therefore targets the shipped a77 revision-baseline mechanism and specifies the canonicalization contract without claiming historical revision coexistence.
