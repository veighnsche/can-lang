# a15 — Constructor-controlled brands (seal scope)

Status: landed. The brand forgery hole is closed: executable code
mints only its own module's brands.

## Rule

`seal` in a function body must name a brand declared in the same
file. `seal` in a test expectation, test argument, or `given`
script row may name any declared brand.

Bodies are code: they run. Tests and scripts are checked data:
expected values and exchange rows are compared kind and payload,
never executed as minting. The declaring file owns every
executable seal site, so `grep seal` stays the whole audit — now
with a mechanical guarantee behind it instead of a convention.

## Why this shape

The alternative — seals only their own brand everywhere, tests
included — breaks the flagship: `auth.can` tests and scripts
`Db__Hash` values declared in `db.can`, and cross-module test
data is legitimate. The rule trusts what is compared and
restricts what runs. A consumer that wants a foreign brand in a
body must import it through a declared `extern` declassifier,
unchanged from a04.

## Proof cost

One check in `checkTypes` (new code CAN6004): the tycker tracks
executable versus data positions and the brand-to-file map, and
refuses a body seal whose brand lives in another file. No
evaluation or emit change: seals still erase to strings, and the
evaluator never mints.

## Consequences (verified, not assumed)

- The flagship compiles untouched: `db.can` seals its own brand
  in its body, and every `auth.can` seal sits in a test or script
  row. All gallery goldens are byte-identical.
- Negative examples ship with the change: `TestSealForeignBrandInBody`
  fails the build on a forged body seal while
  `TestSealForeignBrandInData` proves the same seal clean in rows,
  and `docs/archive/sketches/broken-login/foreign-seal.can` carries the single
  CAN6004 squiggle (its provider gained the shared `Db__Hash`).
- This is the precondition the HTML layer needs: when `Html__Safe`
  arrives, no consumer body will be able to forge it. Nominal
  distinction plus this rule is the safety argument; nominal
  distinction alone never was.

## Open decisions (do not block)

- Whether declassifier `extern`s need a declared-brand manifest
  beyond today's module-local rule.
- Whether seal sites want a per-brand allowlist once brands
  carry capabilities, or file ownership stays sufficient.
