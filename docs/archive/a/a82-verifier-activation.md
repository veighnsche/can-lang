# a82: verifier activation + evidence reporting

Status: shipped. Contracted functions must verify at compile
time; unsupported, unproven, or inconclusive contracts block
acceptance. Pure addition: `compileAll` and `diagnose` call the
unchanged `VerifyContracts`.

## Wiring

- `compileAll` (compiler/main.go): after the ordinary gate and
  the baseline block, `VerifyContracts` findings fail the build
  through the existing `failDiags` path (prose and `--format
  json` alike). Identity keeps its precedence: a drifted
  interface reports before solver time is spent.
- `diagnose` (compiler/lsp.go): proof findings append only
  when the world otherwise checks clean, mirroring the
  identity hook. Broken programs gain no proof noise; the
  editor and the CLI cannot diverge on acceptance.
- Test-only program builders (`revisionProg`, `admitProg`)
  call `checkProgram` directly and are unaffected: mutant and
  negative fixtures still construct.
- `canlc baseline` generation is unchanged: generation is not
  acceptance and never proves.

## Report

Passing prose compilations print the verdict position 7
record after the test lines:

```text
contract verification: succeeded
verified declarations: [m__max@1]
uncontracted declarations: not universally verified: [m__plain@1]
scope: supported source semantics with verified callees
```

`contractIdentities` (compiler/verify_prove.go) splits the
program in order; JSON mode stays silent on success per its
existing contract.

## Fallout

None. Inventory before wiring: no shipped `.can` carries
`requires`/`ensures` (html.can's match is the word
"requires" in a comment), every diagnose fixture expecting
clean verifies, and every error-expecting fixture skips
proof through the error gate. The full suite passes
unchanged apart from the five new probes.

## Operational note

z3 is now a required dependency for contract-bearing
programs: without a solver those programs report CAN4305
and do not compile. Uncontracted programs never invoke the
solver and are unaffected. CI installs z3 (verifier.yml).

## Verification

- 5 durable probes in compiler/contract_activate_test.go:
  CLI blocks unproven builds, accepts verified ones, editor
  squiggles CAN4304, broken programs gain no CAN43xx noise,
  identities split contracted/uncontracted.
- Live CLI demo: max compiles with report and emits; max+1
  (green rows) fails with `unproven` and emits nothing.
- Gates: suite ok, modcheck 25 OK, gramcheck OK, tsc clean.
