# Current validation pattern

Validation is ordinary Can, not a validator catalogue: named domain
functions over records, errors, match, and arrays. The maintained
demonstration is the admitted form-validation application, which
validates repeated/optional form values and renders 422 feedback
through safe server-side constructors. Monomorphic
`std__validate__*` helpers, constraint descriptors, envelope
witnesses, and pinned converter versions are excluded; each domain
owns its checks and messages.

The adjacent legacy source and generated files below are historical
migration inputs scheduled for retirement by I43/I44, not the current
implementation.

## Historical implementation

# quota-counter — validation plus a bounded counter

- `quota.can` — `mod quota`: monomorphic scalar validators
  (`std__validate__require`, `std__validate__int_range`,
  `std__validate__int_nonnegative`, `std__validate__str_nonempty`,
  `std__validate__exclusive_pair`, `std__validate__str_one_of`
  plus its `_from` worker) plus a quota counter
  (`quota__consume`, `quota__usage`) and the S1a request pilot
  (`quota__request__validate`, `quota__request__admit` over
  `Quota__Request` / `Quota__RequestSchema`, plus the S1b
  envelope witness `quota__envelope__validate`): the module pins
  `std__convert__int_to_str@1` for canonical integer rendering
  in violation payloads, with per-site `given` scripts (a18
  verifies the scripted renders against the real converter).
  S2a adds the closed check layer (`std__validate__length_check`,
  `std__validate__range_check`, `std__validate__membership_check`
  over `Validate__Length` / `Validate__Range` /
  `Validate__Membership`): one constraint in, frozen
  `validation.schema_violation` triple out, so consumers bind
  checks to fields without rewriting reconstruction arms
  (see `docs/a91-schema-s2-design.md`).
  (`quota__consume`, `quota__usage`) that reuses them through
  same-file local calls, so every call executes its body.
  Validators return the accepted value or a producer-owned typed
  error; bounds are inclusive and reversed bounds fail instead of
  being silently swapped. Every test starts from init.
- `quota.ts` + `errors.json` — committed golden TS prod emit
  (tests/given stripped; the cell is a module-scope `let`).
  Regenerate: `go run ./compiler --out std/quota
  std/quota/quota.can std/scalars/scalars.can`, then delete the
  co-emitted `std/quota/scalars.ts` (quota keeps only its own
  goldens); verify: `go test ./...`.
- Known packaging gap (S1a): `quota.ts` imports `./scalars`,
  which resolves in whole-program compiles but dangles beside
  the committed goldens — and the specifier is extensionless,
  so node ESM needs a harness rewrite to `./scalars.ts` (see
  `rewriteSpecifier` in the parity tests). Cross-dir TS imports
  need their own emit slice; see the a89 implementation
  amendment.

Rules: `/REQUIREMENTS.md`. Program: `docs/a13-stdlib.md` (row 1).
