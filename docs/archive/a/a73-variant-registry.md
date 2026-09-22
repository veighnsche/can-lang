# a73: variant registry foundation (unions 1/4)

Declaration parsing, AST, qualified membership,
ownership/collisions, field types, combined cycle
check, parent-revision pins. Explicitly frontend
foundation: no constructor evaluation, no pattern
proof, no emit. Construction attempts fail on
unknown ctors naturally; probes assert the specific
registry rule in every negative, paired with a
valid-declaration positive.

1. Parse: `variant Name rev N (` + `case
   Short(...)` rows + `)`, mirroring `type`.
   `VariantDecl{Name, Rev, Cases, Line}`,
   `VariantCase{Short, Fields, Line}`.
   Elaboration (documented, single rule):
   qualified = domain + `__` + short, where
   domain is the variant name before its first
   `__`. Short rows only in declarations
   (qualified rows rejected); qualified names
   everywhere else.
2. World: `provides` registration, `declaredRev`,
   provides-miss — all mirroring `TypeDecl`.
   `prog.Variants` (name → decl) and `prog.Cases`
   (qualified case → parent). Hard errors, never
   first-wins: duplicate variant names (same or
   sibling module), duplicate qualified cases
   across variants, collisions with record type
   names and compiler-owned records, `Bytes`
   shadow. (`ErrorDecl` dotted names cannot
   collide by shape.)
3. Fields: each case's fields validated like
   record fields (known types via `knownType`,
   which additionally accepts variant names so
   payloads may name variants). `Seq<V>` with a
   variant element rejected in case fields and
   record fields alike (deferred surface; no
   existing source can contain it).
4. Cycles: combined record/variant dependency
   graph at declaration admission (existing
   `checkRecordCycles` extended, not duplicated).
5. Pins: consumer `uses [V@N]` resolves through
   the parent (no per-case revisions); wrong rev
   fails `CodeUsesRev`; uppercase pins skip the
   unused warning as today.

Out of scope: construction, patterns, proof,
runtime, emit (a74–a75); pilots (a76).

## Rollback

`git checkout -- compiler/parse.go compiler/check.go
compiler/types.go compiler/eval.go` plus delete
`compiler/variant_reg_test.go`.

## Test plan

- Probe first (red): `compiler/variant_reg_test.go`.
  Positive: three-case declaration (nullary +
  payloads incl. variant-typed payload) diagnoses
  clean; consumer pins correct rev clean.
  Negatives, each asserting its rule: empty
  variant, duplicate case, cross-variant case
  collision, record-name collision, unknown
  payload type, `Seq<Variant>` field, direct
  self-cycle, record-mediated cycle, duplicate
  variant name, wrong-rev pin, qualified row in
  declaration.
- `go test -count=1 ./...`, `go run ./tools/modcheck`,
  `go run ./tools/gramcheck`.
