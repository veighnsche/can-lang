# v0.7 — Helper extraction (spec)

Status: shipped. Expressiveness item 3+4: intra-module
calls that visibly shrink the flagship's duplicated retry arms.

## Rule

A `match call f(...)` scrutinee may name a function defined in the
same module (same file). Call sites need no new syntax: the callee
kind is resolved, not spelled.

Two kinds of callee, distinguished by locality:

- Foreign: a `uses`-pinned can function from another file, or a
  module-local `extern`. Scripted at the call site through `given`,
  exactly as today. Never executed; the stub is the outcome.
- Local: a function defined in the same file. Executed, never
  stubbed. The call site carries no `given` table — a local call is
  deterministic for its test, so there is nothing to script there.

"Deterministic" means fixed by the test: argument values plus the
`given` rows inside the helper, keyed by the caller's test name
flowing through. A helper that itself calls foreign functions keeps
exactly one `given` table per inner call, written once in the helper,
covering every test that can reach it: the helper's own tests plus
every transitive caller's tests. Tests that cannot reach the inner
call are written `-`, as today. The retry pair's two `check_pw`
tables (18 rows doing one job) become one table (12 rows).

## Same-module pin rule: none

Local callees take no `@rev` pin and no `uses` entry. Pins are for
cross-file edges; locality is visible at the call site (the
definition is in the same file). `modcheck` already rejects a
same-file `uses` entry as resolving nowhere, so the pin rule needs
no new enforcement — attempting a pin fails loudly.

Totality of argument passing is arity plus types, already checked
through the callee contract (`callee()` resolves local functions
today): wrong count or wrong type is `CAN6003` at the call site.

## Termination: no recursion

Helpers must not call in a cycle — not themselves, not each other.
Compile-time tests execute helper bodies inline, so a cycle is a
hang before any proof runs. The local call graph (same-file edges
only) must be acyclic; a cycle is a static error at the call site.

> Amendment (a11): the ban is program-wide — see
> `a11-recursion.md`. Same-file cycles stay with this local check;
> cycles touching two or more files are refused by the global check
> (`CAN3005`, reported at the closing call site). Only proven
> direct self-recursion is admitted anywhere.
A runtime depth cap backs the static rule (unreachable past the
gate, as with arithmetic overflow). Termination in general still
belongs to the loops proposal; this ban keeps helpers from
smuggling it in early.

> Amendment (a10): the arithmetic-overflow analogy is retired —
> ints are unbounded and have no overflow mode (`a10-numerics.md`).
> The depth cap itself is unchanged.

## Exhaustiveness (unchanged proof)

A match on a local call covers exactly the helper's `emits` plus
`Ok` — the same proof, resolving through the same table. Missing
arms and stale arms are compile errors as before. The helper's own
`emits` must be honest: every kind it can produce is declared, every
declared kind is raised or stubbed.

## Coverage (shared, not per-function)

Branch coverage is assessed over the module, not the function: a
helper arm counts as taken when the helper's own tests take it or
when any caller's test flows through it. Helpers still ship their
own decision tables (R7 is universal — a function without tests is
still a warning), but the arms they share with callers need no
duplicate tests. Coverage stays the test-per-arm law; only the
counting scope widens to the file, matching `modcheck`'s file-wide
totality.

Consequences for the static checks:

- `given` on a local call site is an error (nothing to script).
- A missing local `given` is not an error (nothing required).
- An inner `given` missing a reaching test (own or transitive
  caller) is a dangling-test error; a key naming no test in the
  file is a dead-script warning.

## TS emit (no new boundary)

A local call emits as a plain same-file function call; its match
emits over the helper's Result union, already in the union table.
The module-wide Ok-shape agreement is unchanged: a helper returning
the module's session shape keeps it, so the flagship's union does
not move. Helpers take only emittable parameter types (base types
and brands — records cross as fields, not as values).

## Consequences (accepted before building)

- The flagship extracts `auth__verify`: the lockout check plus the
  `check_pw` match, written once with one `given` table. Both
  `auth__login` arms become three-arm matches over the helper with
  no `given`. `auth__login` rev 4 → 5 (code changed); helper ships
  at rev 1; `provides` gains the helper; `uses`, module `emits`,
  and `db` are untouched.
- Goldens: `auth.ts` gains the helper function and slimmer login
  arms, `errors.json` is regenerated (same kinds, new handling
  sites), normalize gains the helper's decision-table rows.
- `broken-login/` keeps exactly its titled squiggles (verified by
  the existing suite, not assumed).
- Grammar: untouched (no new syntax). `modcheck`: untouched
  (file-wide totality already describes the new tables).

## Open decisions (do not block)

- Whether helpers should ever be cross-file without pins (leans:
  no; pins are the cross-file story, locality is the feature).
- Whether helper-local value bindings deserve their own coverage
  nuance (leans: no; arms are the law, bindings ride along).
- Larger integer model, division, termination mechanism: with
  their own proposals, not smuggled inside this one.
