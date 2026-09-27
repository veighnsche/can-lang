# G05 evidence — validated rename and editor workflow (2026-09-26)

Outcome: **done**. `textDocument/rename` served over the G03 reference
index with `--write`-discipline overlay validation; AU-LSP-rename and
AU-Q3-rename legs pass; the ordered G01→G05 workflow is demonstrated
end to end; G01–G04 suites stay green.

## Contract and gates observed

- Task-list G05 (R09/R08; Q8/Q3; C-I/C-D): rename uses the reference
  index and `--write`-style overlay validation; includes
  function-local bindings/captures without capturing or renaming
  other-scope names; updates `with` sites; respects fallback
  name-coupling.
- Prerequisites: G04 done (completion service over the G03 index),
  A02 done (explicit near bindings with `with` provenance).
- BLK-02 resolution: renames cover body-local and captured
  references by binding identity; shadowed/same-spelled other-scope
  names stay untouched.
- C-I reconciliation: "`with` bindings reference callee parameters"
  — a near-parameter rename carries its explicit pins (one identity
  in the index); implicit fallback captures keep their own
  caller-scope identity.
- P09 scenario walk: a near-parameter rename that would break
  fallback lookups fails loudly at check time — the rename declines
  rather than applying a breaking edit, and the author repairs
  callers explicitly.
- No runtime TypeScript touched: RH scripts not applicable (Go-only
  lane; `gofmt`/`go vet` gates applied instead).
- Task-list checkboxes untouched per lane instructions.

## Implementation

`compiler/lsp.go` (G-owned rename section, following the G03/G04
precedent of extending `lsp.go` — the I44 retirement gate pins the
launcher to six production files):

- Wire: `textDocument/rename` request handling with `-32602` on
  invalid params, `renameProvider` capability, null decline wherever
  the name, token, or validated candidate cannot support the rename.
- `renamePlan`: resolves the cursor token through the G03
  `buildReferenceIndex`/`occurrenceAt` identity model and lays out
  the replacement of every occurrence sharing the identity,
  declaration site included. Qualified occurrences span the whole
  `pkg::name` form, so only the trailing segment is replaced.
- Name validation: the lexer identifier spelling minus hard
  keywords. Contextual words stay admissible — the candidate
  re-check owns positional breakage.
- Validation (two independent gates, both must pass):
  - No-new-error overlay check: the renamed text checks through a
    copied overlay carrying every other open buffer, and every
    candidate error must have a live counterpart (severity, code,
    file, line span, digit-blanked message — checker messages
    embed byte offsets that shift under any length-changing
    rename). Pre-existing diagnostics ride along; added ones veto.
    The shared G-lane fixtures themselves carry one inert-
    initialization error, so a must-be-clean gate would veto every
    rename; the subset rule adapts the `--write` discipline to
    editor reality.
  - Identity stability: every renamed token must re-resolve in the
    candidate index, all to one shared identity, whose group must
    equal exactly the renamed set — no use lost to another
    binding, none gained from one. This catches capture the
    re-check alone would miss (a use silently resolving to a
    different binding still checks).
- Same-spelling rename yields an empty `changes` edit (no-op
  success); everything else failing yields null (atomic: the editor
  applies nothing).

## AU legs (19 tests, all pass)

| # | Leg | Test | Result |
| --- | --- | --- | --- |
| 1 | Capability advertised, G01–G04 caps intact | InitializeAdvertisesRename | pass |
| 2 | Body-local: declaration, with-value capture, call use | RenameLocalVariable | pass |
| 3 | Same-spelled inputs rename only their own function | RenameSameSpelledLocalsStayDistinct | pass |
| 4 | Match rebind vs outer binding stay distinct | RenameShadowedBinding | pass |
| 5 | Input renamed at declaration, body, nested arm | RenameNestedCapture | pass |
| 6 | Near parameter carries declaration, body use, with pin; applied edit adds no diagnosis (AU-Q3-rename) | RenameWithBindingToCalleeParameter | pass |
| 7 | Callable callee renames with its declaration | RenameCallbackCallee | pass |
| 8 | Shared-record field renames with its declaration | RenameSharedRecordField | pass |
| 9 | Second-file use joins the rename | RenameCrossFile | pass |
| 10 | Capture declines in both directions + anti-steal | RenameCaptureDeclines | pass |
| 11 | Shadowing-correct, contextual, member spellings succeed | RenameAdmissibleSpellings | pass |
| 12 | Provided name declines, header never left stale | RenameProvidedNameDeclines | pass |
| 13 | Full ordered workflow with applies and repair proof (AU-LSP-rename) | EditorWorkflow | pass |
| 14 | Near-param and fallback-side renames decline (AU-Q3-rename) | RenameFallbackCouplingDeclines | pass |
| 15 | Keyword/non-identifier/wildcard names decline | RenameInvalidNameDeclines | pass |
| 16 | Unresolved callee and whitespace decline | RenameUnresolvedDeclines | pass |
| 17 | Same spelling yields an empty edit | RenameSameNameNoOp | pass |
| 18 | Warned file renames + publishes exactly its warning | RenameWarnedFile | pass |
| 19 | Malformed params answer -32602 | RenameInvalidParams | pass |

## Rename and workflow evidence, incl. negatives

- Shadowed negatives: the match-bound `total` renames only its
  pattern + arm use while the outer `total` keeps only its own
  (leg 4); each `value` input renames only its own function
  (leg 3); renaming the outer `total` to `seed` declines because
  the renamed binding would steal the arm's later `seed` uses
  (leg 10).
- Capture negatives: inner-to-outer (`total`→`seed`) and
  outer-to-inner (`seed`→`total`) renames decline (leg 10) — the
  identity-stability gate, since the re-check alone would still
  pass on the silently re-resolved uses.
- Fallback coupling (AU-Q3-rename): with a caller relying on
  fallback lookup, renaming the callee's near parameter declines
  (the candidate check raises CAN-CHECK-CAPTURE), and renaming the
  caller's fallback binding declines symmetrically (leg 14). The
  explicit-`with` rename carries its pin and the applied edit
  checks with nothing added (leg 6).
- No-op/decline behavior: unresolvable tokens, whitespace,
  invalid spellings, and provided names (whose `provides` entries
  are not index occurrences) decline to null (legs 12, 15, 16);
  same-spelling rename yields an empty edit (leg 17).
- Conflict behavior: any rename whose candidate introduces a new
  check error, or whose renamed tokens do not re-resolve to
  exactly one shared identity, declines atomically — the editor
  applies nothing.
- Workflow (AU-LSP-rename, leg 13): format a true-first buffer to
  canonical false-first (G01), hover the shared-record field
  contract (G02), find the callback's references (G03), complete
  the caller locals (G04), rename the callback (G05), apply,
  change the shared-record field (G05), apply, then prove repair:
  the renamed callback references resolve under the new spelling
  and the project publishes exactly its one pre-existing
  diagnosis.

## Verification

- `go test ./compiler/ -run 'TestG05' -count=1`: 19/19 pass
  (155s).
- `go test ./compiler/ -count=1`: full suite green incl.
  G01/G02/G03/G04 and the `TestNoPredecessorPaths` retirement gate
  (392s).
- `gofmt -l compiler/`: clean. `go vet ./compiler/...`: clean.
- Env: macOS darwin/arm64, go1.27.1.
- Commits (worker): `fc85d837` (engine), `ace91b09` (tests).

## Handoff to B/C/H authoring consumers

- Service entry: `rename` on `lspServer` plus the
  `textDocument/rename` wire in `compiler/lsp.go`; result shape
  is `{"changes": {uri: [{range, newText}]}}`, null on decline,
  empty `changes` on same-spelling no-op.
- Known boundary (unchanged from G02/G03/G04): header `provides`
  entries are not reference-index occurrences, so renaming a
  provided declaration declines rather than leaving the header
  stale (leg 12); member calls on checked result types and
  function-local receivers stay outside the known-receiver set,
  so affected member renames decline the same safe way.
