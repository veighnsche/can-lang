# Lane B failure conventions (interface C-B)

Owner: failure conventions (Lane B). Consumers: all Can-authoring lanes,
E06 (R12 redacted hook), B03 (Q5 gate), B04 (Q4 gate), B05/H11 (W3 verdict).

This directory holds the executable conventions plus their evidence.
Nothing here changes the compiler or runtime: conventions only. `emits`
stays finite and explicit everywhere; occurrence/provenance rules are
unchanged; redaction policy stays owned by `entry.ts`.

## Map

| Path | Contents |
|---|---|
| `retry/` | B01 result-data project: shared helper, two domains, traces (35 roots) |
| `retry-fixed/` | B01 fixed-bound comparison project (15 roots) |
| `retry-negative/` | B01 frozen no-retry mutant (11 oracle roots fail) |
| `owner-setup/` | B02 extraction end-state: factory + probes (14 roots) |
| `owner-negative/` | B02 tainted-row negative (fails closed, located frames) |
| `owner-forge/` | B02 forged-owner negative (check-time rejection) |
| `harness_test.go`, `retry_test.go`, `owner_test.go` | Black-box `canlc assert` driver + 11 W3 legs |
| `x-r06-1.md` | X-R06-1 experiment record, measurements, Q5 assessment |
| `x-r07-1.md` | X-R07-1 experiment record, measurements, Q4 assessment |

Run the legs with a prebuilt bundle (`CONV_BUNDLE`) or let the harness
build through the shared cache (`CAN_BUN_ARCHIVE`, same root/key scheme
as `tests/integration`):

```sh
CONV_BUNDLE=/path/to/bundle go test ./tests/failure-conventions/ -v
```

## Result-data convention (R06)

One generic helper serves unrelated failure sets; boundary adapters
convert declared domain errors to data and back. Reference
implementation: `retry/src/retry/retry.can`.

- **R1 shapes.** Use the nominal `completed<item>` / `rejected<failure>`
  records under `variant outcome<item, failure>`. No ad-hoc tuples.
- **R2 helper purity.** The helper takes
  `callable outcome<item, failure> (arg) emits []` plus its input and a
  remaining-attempts bound, and returns the first completion or the last
  rejection. It never catches standard failures: they propagate with
  occurrence intact (`blast_kind` pins this).
- **R3 faithful sets.** Each fallible boundary declares its failure set
  as a variant whose members are the errors themselves
  (`variant load_failure / unavailable / forbidden`). Authored and
  catalogue errors are both admitted members; both round-trip exactly
  (`read_quote` proves the catalogue case). Lossy `failure = str`
  projections are a known weaker form: allowed only where the boundary
  is terminal and documents the loss.
- **R4 capture arms.** One arm per error, rebuilding the bound error
  value into the set. A member value never infers its variant through a
  generic constructor, so each arm uses the annotated-local form:
  `as failure => do / <set> leaf = <error>(...) / ok rejected(leaf)`.
  Three lines per error; no per-leaf helper pays off below four arms
  sharing one set (measured in `x-r06-1.md`).
- **R5 re-raise.** Match the outcome, bind the rejected leaf into a
  local (field paths do not narrow), match the leaf, and reconstruct
  each error constructor. One line per error per layer. Bare error
  values are not valid data-match bodies.
- **R6 trace shells.** Tag rejections with `traced<failure>` records
  (`str operation`, `failure cause`) via `tag`; completed values pass
  through with a plain `ok` of the narrowed leaf. Layers nest
  (`traced<traced<load_failure>>`); recover the original value through
  `.cause` chains and re-raise it unchanged (`read_traced` proves the
  round-trip). One inline layer costs five lines; per-layer helpers do
  not pay off (measured).
- **R7 oracle rows.** Attempt counts and failure-then-success order are
  enforced by FIFO `when` rows at the adapter's call site, selected by
  root name. Rules: rows live in the root's package (cross-package rows
  never activate); every root sharing one when-table needs a distinct
  name (rows are selected by name, queues are per root); size each
  queue to the exact attempt count so under-retry fails
  unused-fixture and over-retry fails missing-fixture. The
  no-retry mutant and the over-attempt legs pin both directions.
- **R8 nested literals.** Deeply nested generic expectations need full
  explicit type arguments on every level
  (`rejected<traced<traced<load_failure>>>(traced<...>(...))`); partial
  annotation still conflicts. Verbose but exact.
- **R9 redaction preservation (handoff to E06).** Adapters keep full
  error values for server-side re-raise and diagnostics. Any failure
  data crossing into client-visible responses must first project
  through the `entry.ts` policy: domain identity, category plus
  occurrence, no native messages, paths, or secrets. Standard-failure
  observations (`kind`, `message`, `occurrence_id`) never cross that
  boundary unprojected. The R12 hook owns the projection point; these
  adapters own keeping convertible values convertible (no stringifying
  at capture).

## Factory convention (R07)

One private factory per fallible-constructor input shape feeds owner
values to assertion rows. Reference: `owner-setup/`.

- **F1 shape.** `fn <owner> fixture_<name>(<raw inputs>) emits []`:
  match the constructor call, forward the `ok` value, trap unexpected
  rejection. Never in `provides` (foreign calls reject at check).
- **F2 trap idiom.** `ids::invalid => relay call fixture_id(1 / 0)`.
  The relay argument faults before any call happens, so the run fails
  with a located `arithmetic` standard fault: fixed message
  `arithmetic: integer division by zero`, frames at the trap span plus
  the tainted input site. One line per constructor error arm.
- **F3 specification probes.** Each factory pins its fault once with
  standard-catch probes asserting `kind == "arithmetic"` and the exact
  message. These are negative-behavior specifications, not a handling
  pattern: production code never catches fixture traps (the tainted-row
  leg proves fail-closed instead).
- **F4 evidence chain.** The factory's own row self-executes
  (`unit: 1 => ok call fixture_id(1)`); correctness evidence comes
  from the triple: owner-package literal roots, factory self-row, and
  consumer projection roots through the helper. Never forge owner
  values (`ids::user_id(1)` in a foreign row rejects at check) and
  never weaken a row to dodge the trap.
- **F5 repair.** Constructor evolution is compiler-guided: a new error
  fails the check with `missing completion arm for <error>` until the
  factory arm lands. The extracted owner-accepting helper itself is
  untouched by constructor change (pinned byte-identical by the repair
  leg).

## Gate inputs

- Q5 (finite error-set syntax, gated B03): `x-r06-1.md` recommends
  **inactive** with measurements and trip conditions.
- Q4 (checked setup syntax, gated B04): `x-r07-1.md` recommends
  **inactive** with measurements and trip conditions.
