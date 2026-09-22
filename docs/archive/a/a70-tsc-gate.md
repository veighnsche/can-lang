# a70: tsc gate reinstatement (a14 criteria)

Ships all three a14 reinstatement criteria. No `.can`
changes; emit changes only if strict finds real
defects (narrowing or per-function result types per
the priced consequence — never suppressions).

1. CI job (`.github/workflows/tsc.yml`): pinned
   TypeScript, `tsc --strict` over every committed
   golden `.ts` plus extern-stub fixtures. Toolchain
   lives in `tscheck/` (own `package.json` pin +
   `tsconfig.json`, lockfile committed).
2. Green run on current goldens, incl. multi-shape
   ok unions. Real strict failures get emit fixes.
3. Ownership policy (`docs/a70-tsc-ownership.md`, never written, or
   a14 amendment): `tsc` owns type + contract shape;
   `canlc` exclusively owns exhaustiveness,
   termination, decision tables. REQUIREMENTS R11
   tagged amendment reinstating the clause.
4. Extern stubs: `docs/archive/sketches/auth-login/auth.externs.ts`
   (`auth__check_pw`), `docs/archive/sketches/retry-loop/retry.externs.ts`
   (`net__fetch`), signatures read off the `.can`
   extern decls + call sites. (Drift-checking stubs
   against decls is a named follow-up, not this
   slice.)

Out of scope: contracts verifier, unions, any `.can`
semantics; `tsc` never checks coverage.

## Rollback

Delete `.github/workflows/tsc.yml`, `tscheck/`,
stubs, policy doc; `git checkout -- REQUIREMENTS.md`
plus any emit files touched.

## Test plan

- `tsc --strict` green locally via the same command
  CI runs; `go test -count=1 ./...` still green
  after any emit fix (goldens regen + Go-side
  assertions updated as needed).
- `go run ./tools/modcheck`, `go run ./tools/gramcheck`.
- Workflow YAML validated (actionlint if present,
  else `python -c yaml.safe_load` + schema eyeball).
