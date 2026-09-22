# a89: std__validate__schema — design (decided, S0 closed)

Status: decided 2026-09-18. Rulings recorded below; no code.
S1 slices (P1, S1a, S1b) are authorized; S2–S4 are gated
milestones, not approved slices.
Parent: `docs/ASTRA_STDLIB.md` §1.5 (`validate__schema`),
§6 order 9 (schemas and canonical codecs).

## Provenance

Design-agent verdict returned against pin `67b27c78da49`.
At that pin `docs/a/a89-schema-validation-design.md` was an
untracked local file, so the verdict adjudicates the
proposal as quoted in the task, not the full a89 text.
This update reconciles the gate by recording the rulings
here: where this doc and the quoted proposal differ, the
rulings below win. No code changes or compiler gates were
run for the verdict.

## Verdict summary

**Approve a narrowly scoped constraint-validation pilot.
Do not count it as delivery of the generic
`std__validate__schema` catalogue entry.**

Correction to the proposal's blocker framing: a codec is
not a prerequisite for useful constraint validation. A
correctly typed record can still violate a range, length,
or membership policy — the existing quota validators
already demonstrate that distinction. What remains blocked
is the reusable, shape-polymorphic `Schema<T>` facility,
not validation of one concrete record.

## Why the catalogue signature cannot land as written

`(value: T, schema: Schema<T>) → T !
[validation.schema_violation]` still needs three things
that do not exist:

1. **Generics (order 8).** `Schema<T>` cannot be named and
   no `fn` abstracts over `T`. `knownType`
   (`compiler/types.go`) admits base types, records,
   brands, `Bytes`, and `Seq` over plain elements only.
2. **Descriptors and traversal (order 9).** No schema
   descriptor values and no way to walk a record
   generically: field access is static (`r.field`). The
   existing `std/schema` is asset approval only — "the
   asset minimum of Schema, not the general layer"
   (`docs/schema-design-prompt.md`).
3. **A reuse mechanism for arbitrary `T`.** Even with
   generics, nameable `Schema<T>` does not create record
   traversal; checked field binding and interpretation
   are a separate admission (see S3 gate).

## Fixed pilot

A **quota-request admission program**, extending the
existing quota example. No parsing, authentication, or
policy registry.

| Record               | Fields                                                                                                          |
| -------------------- | --------------------------------------------------------------------------------------------------------------- |
| `Quota__Request`       | `label: str`, `amount: int`, `mode: str`                                                                        |
| `Quota__RequestSchema` | `label_minimum: int`, `label_maximum: int`, `amount_lower: int`, `amount_upper: int`, `allowed_modes: Seq<str>` |
| `Quota__RequestValue`  | `value: Quota__Request`                                                                                         |

All are ordinary revisioned records. The concrete operation
is **`quota__request__validate`** — not a generic-looking
`std__validate__schema`, and no generic alias. The
motivating program chooses its schema and validates a
request before calling the existing `quota__consume`.
Baseline policy (program-owned, not stdlib doctrine):
label length **1–12 Unicode scalars**, amount **1–10
inclusive**, modes **`"interactive"` or `"batch"`**.

Success returns the original request inside the named
success record: no normalization, no protected
"validated" capability, no promise downstream callers
cannot bypass validation.

Below, **`V(path, rule, value)`** abbreviates
`validation.schema_violation(path: str, rule: str,
value: str)`.

## Seven rulings

### 1. Schema means a reusable constraint bundle — CONFIRM

The schema supplies additional predicates over an
already well-typed value; fixed field accesses bind each
constraint to its field. Useful without reflection or
decoding — but one concrete record does not deliver the
generic catalogue contract. Proof cost: one explicit
field-to-validator binding per field, with evidence for
the right field, bounds, and offending value.

| Input                                                  | Rejection                                  | Error                                   |
| ------------------------------------------------------ | ------------------------------------------ | --------------------------------------- |
| `amount = 11`, upper bound `10`                        | Violation despite correct shape            | `V("amount", "int.closed_range", "11")` |
| `mode = "admin"`                                       | Not in the supplied membership policy      | `V("mode", "str.one_of", "admin")`      |
| `amount` built from `str` instead of `int`             | Source type mismatch, before validation    | Existing `CAN6003`                      |

### 2. Validation and decoding have separate ownership — CONFIRM

The validator accepts an existing `Quota__Request`; it
neither parses wire data nor repairs representations.
Future decoders own syntax, presence, construction, and
wire-number interpretation, and must map failures into
their own declared contracts rather than leak validator
errors.

| Input                                        | Rejection                              | Error                                              |
| -------------------------------------------- | -------------------------------------- | -------------------------------------------------- |
| `Bytes`/`str` where `Quota__Request` required | Wrong argument type; no implicit decode | Existing `CAN6003`                                |
| `mode = "Interactive"`                       | Exact membership; no case folding      | `V("mode", "str.one_of", "Interactive")`           |
| Thirteen-scalar label                        | Length failure; no truncation          | `V("label", "str.length_scalars", original_label)` |

An accepted `" build "` stays unchanged. Validation is
not normalization.

### 3. Pilot-first, ordinary records, static calls — CONFIRM

Expressible with existing records, scalar validators,
canonical integer formatting, and a separately admitted
string-membership operation. No schema declaration,
runtime field enumeration, function values, or
compiler-recognized descriptor. **Malformed schema
records are explicit outcomes**, not assumed away: the
pilot exercises and translates the producers'
invalid-bounds outcomes rather than prevalidating into
unwitnessable arms.

| Input                 | Rejection                     | Error                                                 |
| --------------------- | ----------------------------- | ----------------------------------------------------- |
| `label_minimum = -1`  | Invalid length-policy minimum | `V("", "schema.label_minimum", "-1")`                 |
| Label bounds `5, 1`   | Reversed policy bounds        | `V("", "schema.label_bounds", "minimum=5;maximum=1")` |
| Amount bounds `10, 0` | Reversed policy bounds        | `V("", "schema.amount_bounds", "lower=10;upper=0")`   |

### 4. Three-string payload — REJECT AS STATED; retain layout with bounded semantics

Retain exactly **`validation.schema_violation(path: str,
rule: str, value: str)`** as a closed diagnostic contract
for this pilot — not a universal value serialization.
String offenders pass through unchanged; integer offenders
use the canonical public converter (not its fuel-taking
worker); the two bounds cases use the exact formats
below. Output-context escaping belongs to the eventual
sink. No brand-to-base disclosure (`CAN6003` enforced).

**Frozen mapping:**

| Producer outcome         | `path`     | `rule`                   | `value`                   |
| ------------------------ | ---------- | ------------------------ | ------------------------- |
| Negative `label_minimum` | `""`       | `"schema.label_minimum"` | Canonical offending int   |
| Invalid label bounds     | `""`       | `"schema.label_bounds"`  | `minimum=<int>;maximum=<int>` |
| Invalid label length     | `"label"`  | `"str.length_scalars"`   | Original offending string |
| Invalid amount bounds    | `""`       | `"schema.amount_bounds"` | `lower=<int>;upper=<int>` |
| Amount outside range     | `"amount"` | `"int.closed_range"`     | Canonical offending int   |
| Mode not allowed         | `"mode"`   | `"str.one_of"`           | Original offending string |

Empty path on malformed-policy errors means **the
constraint bundle for this value is invalid**; the fixed
`schema.*` rule token distinguishes that from a
data-field violation. Dec/bool formatting may later reuse
their canonical converters, but the str/int pilot is not
evidence those paths were exercised.

**Fail-fast order (fixed):** (1) `label_minimum`
nonnegative; (2) label-length validation incl.
invalid-bounds; (3) amount-range validation incl.
invalid-bounds; (4) mode membership. A malformed *later*
constraint never preempts an earlier data violation
(e.g. empty label + reversed amount bounds reports the
label failure).

| Input / defect                                   | Rejection                                | Error                                                  |
| ------------------------------------------------ | ---------------------------------------- | ------------------------------------------------------ |
| `amount = -9007199254740993`, baseline bounds    | Exact rejection, no host-number rounding | `V("amount", "int.closed_range", "-9007199254740993")` |
| Branded value to an ordinary scalar converter    | No implicit disclosure                   | Existing `CAN6003`                                     |
| Renderer truncates/rounds an offender            | Payload mismatch in conformance fixture  | Existing `CAN4200`                                     |

### 5. Dot-joined string paths, empty string for root — CONFIRM for the pilot

Paths identify positions in the validated record. Fixed
segments only: `label`, `amount`, `mode`, plus `request`
for the nesting witness. No escaping grammar, index
convention, or public path-builder API. Note the
correction: `Seq<str>` in an error field is **not
prohibited by the checker** (`checkDeclFields` uses
ordinary known-type checking; only direct variant
sequences are refused). Strings reduce the exercised
surface; they do not establish a language limitation.

| Input / defect                          | Rejection                                        | Error                                               |
| --------------------------------------- | ------------------------------------------------ | --------------------------------------------------- |
| `"a.b"` over max length `1`             | Dot stays value data, not a separator            | `V("label", "str.length_scalars", "a.b")`           |
| Malformed amount policy at top level    | Root constraint-bundle failure                   | `V("", "schema.amount_bounds", "lower=10;upper=0")` |
| Path derived from offending text        | Wrong report, caught by frozen expectation       | Existing `CAN4200`                                  |

### 6. Nest through ordinary calls and explicit prefixing — CONFIRM

One envelope operation calls the flat validator once. On
failure it preserves `rule` and `value` and rewrites only
`path`: empty child path → `"request"`; nonempty →
`"request." + child.path`. This is an explicit error
transformation, **not an identity relay or unchanged
`forward`** — both branches need real witnesses. Flat
pilot and envelope stay separately shippable; the
envelope is not the independent second consumer for S2.

| Child failure                                       | Required parent failure                                    | Forbidden                        |
| --------------------------------------------------- | ---------------------------------------------------------- | -------------------------------- |
| `V("", "schema.amount_bounds", "lower=10;upper=0")` | `V("request", "schema.amount_bounds", "lower=10;upper=0")` | `"request."`                     |
| `V("amount", "int.closed_range", "11")`             | `V("request.amount", "int.closed_range", "11")`            | `"amount"`, `"request.request.amount"` |
| Child rejects a request                             | Parent rejects before quota consumption                    | Success or quota mutation        |

### 7. Record revision as revision identity — REJECT; retain ordinary revisions, no `SchemaRevision`

The record `rev` identifies its declaration, not every
runtime policy value or the validator's interpretation.
Three existing distinctions: schema-record revision owns
shape; the validator's contract and function revision own
interpretation, ordering, rule tokens, and rendering; the
schema argument supplies policy values. Per a77, bodies
are proof-freshness inputs, not interface identity — so
reordering checks does **not** automatically trigger
interface-drift detection.

| Change / input                                              | Rejection                       | Error                              |
| ----------------------------------------------------------- | ------------------------------- | ---------------------------------- |
| Shape change without required rev update vs accepted baseline | Interface-identity drift      | Existing `CAN6013`                 |
| Provider rev bump with stale consumer pin                   | Stale revision pin              | Existing `CAN2103`                 |
| Reordered checks change first failure on multi-failure input | Frozen behavioral expectation fails | Existing `CAN4200`, not `CAN6013` |

Bounds `10` vs `20` may share a record revision and
disagree about `amount = 15`. **Do not cache validation
by type name and revision alone.**

## Open questions — closed

1. Pilot record: quota-request admission (fixed above).
2. Payload: `(path, rule, value: str)` single kind, frozen
   mapping, empty-path-for-root convention.
3. Ordering: membership first — P1 `str_one_of`, then
   S1a, then S1b. No silent membership duplication.

## Non-goals (all slices here)

Business policies in generic validation; generic
`Schema<T>`; migration; JSON/form/row codecs;
`Seq<Variant>` outcomes; any compiler change in
P1/S1a/S1b; any claim the catalogue row is delivered.

## Finalized near-term plan

### Common acceptance gate (every slice)

Operation, motivating use, complete outcome expectations,
negative examples, termination/domain arguments,
documentation, and generated artifacts — the §6 admission
rule, not merely "the suite passed." Run separately per
slice:

```text
go test -count=1 ./...
go run ./tools/modcheck
go run ./tools/gramcheck
```

Cross-file pure composition requires **decision-table
coverage plus real linked execution** (`runLinkedPure`,
a68 — no script fallback, no arm-coverage credit, and it
refuses effectful graphs, so it never evidences the
stateful quota-consumption path). Representative
generated TypeScript must also be executed: a
byte-identical golden freezes output but does not
demonstrate target behavior.

### P1 — String membership, before S1

One operation: monomorphic `std__validate__str_one_of`
(`str`, `Seq<str>`) → original string via a named success
record, or `validation.not_allowed(value: str)`. No
comparator callback, no bundling into the schema slice.

Fixed semantics: exact existing string equality; no
trimming, folding, or normalization. Empty allowed is a
valid deny-all policy. Duplicates do not change
membership. Rows for empty/singleton sequences,
first/middle/last matches, absence, duplicates, and
case/Unicode distinctions. Every inspected index in
bounds; checked decreasing argument over the finite
sequence — no fuel exhaustion masquerading as "not
allowed." Complete payloads, generated-runtime agreement.

### S1a — Flat quota-request validator

One operation: `quota__request__validate` per the fixed
pilot above. Policy values stay in the motivating
program. No asset-module changes, no generic alias.

| Obligation                     | Required evidence                                                                                 |
| ------------------------------ | ------------------------------------------------------------------------------------------------- |
| Every constraint               | Independent negative rows: short/long label, below/above amount, membership rejection             |
| Every malformed-policy outcome | Negative minimum + both reversed-bound cases, exact root payloads                                 |
| Boundaries                     | Inclusive label/amount endpoints; Unicode-scalar length boundaries                                |
| Deterministic priority         | Multi-failure rows, incl. earlier data failure vs later malformed policy                          |
| Preservation                   | Success returns every input field unchanged; no reconstructed substitutes                         |
| Exact reporting                | Empty strings, Unicode/control strings where admitted, integers beyond host safe-integer range    |
| Real composition               | Linked-pure vectors through the scalar validators, membership impl, and public int converter      |
| Target behavior                | Generated TS agrees on success, kind, all payload fields, and failure ordering                    |
| Motivating program             | Rejected admission never reaches quota consumption                                                |

Translate using the producer's actual error payload —
never a convenient constant or a different request field.
Keep normal `given` evidence where required, but do not
mistake it for execution of the imported implementation.
No new compiler diagnostics: schema failures are declared
runtime outcomes.

### S1b — One envelope composition witness

One operation: envelope validator with one
`request: Quota__Request` child, one call to S1a. Direct
and linked rows cover child success, a root policy
failure, and a non-root field failure. Prefixes exactly
once; preserves `rule`/`value`; success preserves the
whole envelope. No general path utility, recursive
descriptors, or second blessed abstraction.

## Gated milestones (replace S2–S4; not implementation-ready)

| Milestone                  | Admission gate                                                                                                             | Required proof before implementation                                                                                                               |
| -------------------------- | -------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| **S2: closed descriptors** | A second independent concrete consumer shows repeated binding/interpretation work that shared scalar calls do not remove    | Closed field bindings, correct field types, complete rule dispatch, deterministic ordering, termination/domain arguments, schema/record-mismatch rejection |
| **S3: generic `Schema<T>`** | Explicit generics **and** a checked mechanism binding descriptors to `T`                                                   | Wrong-type schema applications reject; every admitted descriptor has a sound interpretation; pilot success/error/ordering behavior preserved          |
| **S4: codec integration**  | A reviewed order-9 wire contract and descriptor-to-record construction boundary exist                                      | Separate decode/encode operations, complete decoder-owned errors, exact numerics, target conformance, the real round-trip program                   |

S2: a second caller of the same pilot is insufficient —
the missing source is a second independent motivating
`.can` program plus concrete duplication exhibits. Until
then, do not admit closed-descriptor machinery; even
then, specialized calls may stay cheaper.

S3: generics are necessary, not sufficient — nameable
`Schema<T>` does not create record traversal. No
automatic pilot rename, compatibility alias, or §1.5
completion when order 8 lands; the roadmap separates
order 8 (generics/functions) from order 9
(schemas/codecs).

S4: descriptor sharing must not collapse validation and
decoding into one operation. The order-9 gate stays the
catalogue round trip (large integer, exact decimal,
optional field, collection) with malformed-input and
target-conformance evidence.

**Sequence: P1 string membership → S1a flat validator →
S1b nesting witness.**

## Implementation amendments (S1a)

1. **Cross-file converter, dangling committed import.**
   S1a pins `std__convert__int_to_str@1` from scalars.can
   (user ruling over same-file vendoring). Emit hardcodes
   same-dir, extensionless imports (`"./scalars"`), so the
   committed `std/quota/quota.ts` import dangles beside the
   goldens; it resolves in whole-program compiles (golden
   test, linked vectors), and node parity rewrites the
   specifier to `./scalars.ts` in the temp copy only
   (`rewriteSpecifier`; goldens pin emitted bytes verbatim).
   Cross-dir TS import paths and specifier style need their
   own emit slice — not smuggled into S1a. Regen deletes
   the co-emitted `std/quota/scalars.ts`.
2. **Payload reconstruction instead of payload reads.**
   `CodeForeignRaise` (`eachRaise`, `compiler/check.go`)
   treats reading a matched error binder's fields as
   raising that kind, so the verdict's "use the
   producer's actual error payload" instruction cannot
   hold under the frozen single-kind emits. S1a binds
   `_` and reconstructs from its own inputs
   (`request.label`, schema bounds, `request.amount`,
   `request.mode`). The values are identical to the
   producer payloads by construction — same-file
   executed calls whose bodies echo their inputs — so
   the verdict's intent (no fabricated payloads) holds;
   only the mechanism differs. Decision tables, linked
   vectors, and node parity pin the exact payloads.
3. **modcheck given-scope fix (tool, in-slice).** modcheck
   demanded every given block key every test in the FILE
   (a rule never exercised on a multi-concern file with
   foreign calls); quota.can would need ~450 mostly-`-`
   entries. Fixed to reaching scope — own fn rows plus
   same-file transitive callers, mirroring check.go
   `reachingTests` — with unknown keys still rejected.
   Single-fn files behave byte-identically (both demo
   fixtures and all suite messages unchanged); three
   regression tests pin the new scope. Precedent: the a13
   in-slice emitter fix. No language change.
