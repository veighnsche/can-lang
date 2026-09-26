# P04 alternatives — browser/host integration (R01–R03, R09)

## R01 host integration

- **O1 catalogue-only (retain + grow).** Demonstrated missing operations are
  added to the central catalogue one by one; compiler-only admission stays.
  Guarantees: capability gate, transitive closure, audited bundle unchanged.
  Composition: new ops compose like existing ones. Maintenance: every new
  capability is a compiler/runtime change — the cost the review criticizes.
  Suited to a deliberately bounded platform (then publish the scope + the
  companion architecture clearly).
- **O2 reviewed adapters.** An explicitly trusted adapter-package boundary
  with declared types, failure mappings, target capabilities, ownership/
  disposal rules, and conformance checks. Guarantees: admission stays
  reviewed; third parties can still not self-admit. Composition: adapters
  compose as ordinary dependencies once admitted. Maintenance: the review
  checklist + conformance suite become permanent obligations. Needs the P11
  two-concrete-ops comparison before selection.
- **O3 companion implementation.** Keep the language surface fixed; document
  a companion JavaScript/service boundary (contract parity, serving, auth,
  release coordination) for unsupported features. Guarantees: Can side
  unchanged; companion side explicitly outside Can's verification.
  Maintenance: a second implementation + its drift risk. Best when the
  missing capability is large, fast-moving, or genuinely foreign (rich
  editor, SDK).

Discriminating experiment (isolated prototypes, no production code): add the
universal DOM primitives R02 demands → extract one ordinary Can widget →
integrate one genuinely required third-party widget twice (O2 adapter vs O3
companion) and compare: target availability handling, I/O copying, failure
mapping, callback/reentrancy, ownership/disposal, reproducible packaging.
W2 acceptance (second vendor without a compiler patch) decides.

## R02 browser controls and reusable UI

### Native addition set (needed before any library can finish)

Candidate additions, each needing the P08 treatment (all-Can workflow +
bounded immutable projection + per-field browser tests in Chromium 140 and
WebKit 26): live value/checked properties (read + setter), checkbox state in
snapshots, multiselect values, file selection handles, modifier keys,
composition/IME state, selection/caret accessors. Each addition is a small
catalogue + adapter + checker slice; no syntax needed. The minimal sufficient
set is decided by the extraction experiment below, not upfront.

### Composition (F-R02-03/04/05)

- **O1 library-first (favored).** Reusable Can field, keyed-row, and
  lifecycle libraries over named functions/records/callables; nested widgets
  compose within the existing lifetime model (root-only cross-view append).
  Experiment: extract the grid's field handling + keyed table, consume from
  a second app, and cover normalization, reset/autofill, multiselect, upload
  selection, IME, caret/focus, reorder, independent disposal, late replies.
- **O2 ownership-primitive change.** If O1 proves that nested cross-view
  composition cannot be expressed safely, design a primitive change to the
  append/ownership contract. Needs a demonstrated case of the precise
  inexpressible shape — not assumed.
- **O3 declarative component syntax (deferred).** Only after demonstrated
  library limitations (P12). Not designed in this round.

Failure cases for the experiment: reply arriving after unmount must not
clobber newer edits; disposal during a pending save must not leak listeners
or mutate detached nodes; IME composition + normalization must preserve
caret.

## R03 captured HTML routes

Bounded technical choice; reuse existing capture/URL machinery, no SPA router.

- **O1 captured GET + HTML body.** Extend the action checker so a bodyless
  GET may declare `body html` with `:capture` segments, typed extraction,
  and document rendering. Keeps one URL grammar; must reconcile the
  `swap inner` HTMX-fragment policy with full-document reads (fragment swap
  vs document render needs a design answer).
- **O2 capture-aware plain routes.** Allow `:capture` segments in plain
  routes with typed extraction + HTML document responses. Smaller action
  surface change; duplicates some capture machinery unless carefully shared.

Either way, preserve: static path literals, typed capture/URL/ambiguity
checks, exact status/leaf agreement, body-limit budgets, HTML guard policy.
Likely a small grammar/checker delta; syntax follows the technical shape (Q7
asks only the in-principle question).

## R09 editor workflows

Tooling scope/order decision (no syntax; rename couples to R08):

1. **Format (cheapest):** wire `textDocument/formatting` to the existing
   `formatSource` + overlay validation. Open: whole-document only vs range
   (renderer supports whole-document).
2. **Hover:** new type-at-offset query over `World` + checked types; reuse
   definition's scope walk as the front half.
3. **References:** whole-project reference index (invert `findReference`);
   decide body-local inclusion (definition currently declines).
4. **Completion:** scope-aware candidates respecting keywords, `near`
   obligations, arities — needs a precision/recall design to avoid noise.
5. **Rename (waits on R08):** references + safe-edit validation reusing
   `--write` discipline; must handle `near` name-coupling or follow the Q3
   outcome.

Acceptance stays the review's: extract/rename a callback, change a shared
record, inspect its contract, repair callers with normal editor tools. The
user prioritizes (Q8); preparation recommends format → hover → references →
completion → rename order on cost/dependency grounds.
