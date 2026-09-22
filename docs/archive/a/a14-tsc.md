# a14 — tsc verification story (decision)

Status: landed (decision). No code change; no golden change.

## Finding

R11 carries one claim with zero mechanism: "`tsc` re-checks types and
contracts, not coverage." Nothing runs `tsc` anywhere. The repo has
no CI directory, no TypeScript toolchain, and no gate that executes
it. The claim has been aspirational since it was written.

## Decision

The clause is suspended until a gate ships. `canlc` is the sole
verifier: its check phases plus the hermetic decision-table run
are the complete proof, and emitted TypeScript is unverified
output until the reinstatement criteria below are met.

## Reinstatement criteria

A future PR reinstates the clause by shipping all three:

1. A CI job running `tsc --strict` over every committed golden
   `.ts` file plus extern-stub fixtures for each `.externs`
   import the goldens reference.
2. A green run on the current goldens, including the multi-shape
   ok unions from the a13 emitter change.
3. A documented policy for what `tsc` owns (type and contract
   shape) versus what stays exclusively in `canlc` (exhaustiveness,
   termination, decision tables).

## Priced consequence, recorded now

Current emit may not pass `--strict`: field access lands on
union-typed bindings, and ok members share one `kind`
discriminant across shapes. If the gate arrives and fails on
real output, the fix is emit narrowing or per-function result
types — not suppressions. That cost is accepted in advance so
the future gate cannot be negotiated down to a green-looking
check that proves nothing.

## Consequences (accepted by writing this down)

- REQUIREMENTS.md is untouched: the freeze rule means amendments
  land tagged, never by silent rewrite. This doc is the tag.
- No test or tooling change ships here; inventing an unrunnable
  CI job to look resolved would be exactly the aspirational
  pattern this decision retires.

## Reinstated (a70)

The clause is live again: R11's "`tsc` re-checks types and
contracts, not coverage" is now a running gate, not an
aspiration.

- Gate: `.github/workflows/tsc.yml` runs pinned
  TypeScript 5.9.2 (`tscheck/`, lockfile committed)
  `tsc --strict --noEmit` over every committed golden
  `.ts` (9 files: 5 std + 4 sketches) plus 2
  hand-written extern-stub fixtures
  (`docs/archive/sketches/auth-login/auth.externs.ts`,
  `docs/archive/sketches/retry-loop/retry.externs.ts`). Green on
  arrival: the priced emit-narrowing risk did not
  materialize, so no emit change shipped.
- Ownership: `tsc` owns type + contract shape of
  emitted output. Everything else — exhaustiveness,
  termination, decision tables, coverage — stays
  exclusively in `canlc` (`go test ./...`). A green
  `tsc` run never substitutes for the canlc proof, and
  an canlc-green program never skips the `tsc` shape
  check on changed emit.
- Follow-up (not this slice): drift-checking stubs
  against their `.can` extern decls (a signature
  change today rots its stub silently until a human
  notices the mismatch).

## Checker upgrade (2026-09-18 note; history above untouched)

- Pin moved `5.9.2` -> `7.0.2` (native `tsgo`; bin still
  `tsc`, workflow command unchanged). Upgrade run as a
  rev-pin bump, VS Code playbook: 5.9 gate green at
  baseline, 7.0 gate green after, zero diagnostic diff
  on the goldens, negative control confirmed (broken
  copy fails `TS2322`, intact copy passes).
- One config change, checker-mandated: TS 7 removed
  `moduleResolution: node` (`TS5108`), so `tsconfig`
  now reads `bundler` — the mode that keeps resolving
  the emit's extensionless relative imports (`./db`).
  `node16`/`nodenext` would demand `.js` extensions
  the settled emit shape does not produce. No `.ts`
  file changed; gate semantics unchanged.
- Same pass: workflow actions `checkout`/`setup-node`
  `v4` -> `v6` (Node 20 runner runtime removed
  2026-09-16; v4 ran on it), extension client
  `vscode-languageclient` `9.0.1` -> `10.1.1`
  (API-compatible for the 37-line stdio client;
  `engines.vscode` floor `1.85` -> `1.91` per the
  dep's requirement), both VS Code deps exact-pinned
  to match `tscheck/`.
