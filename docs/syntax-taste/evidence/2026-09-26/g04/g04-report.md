# G04 evidence — scope-aware completion (2026-09-26)

Outcome: **done**. `textDocument/completion` served over the checked
overlay snapshot with scope/near/arity precision; AU-LSP-completion
legs pass; G01–G03 suites stay green.

## Contract and gates observed

- Task-list G04 (R09; Q8; C-I/C-D/C-A): respect keyword/near/arity
  obligations; consume the settled browser query and the conditional
  iteration surface only if active.
- Prerequisites: G03 done (reference index with explicit approved
  local-name scope — body-local and captured references by binding
  identity, shadowed/same-spelled other-scope names untouched),
  A06 INACTIVE (Q6 gate — no iteration keywords, no with-parameter
  expansion beyond the G03 `withName` support), C02 done (C-D LSP
  query shapes over `World`; catalogue controls consumable via the
  normal qualified-symbol path).
- C-A reconciliation: "R09 completion (no new keywords unless Q6-O3
  triggers)" — Q6 did not trigger, so the keyword sets below are the
  complete selected surface.
- No runtime TypeScript touched: RH scripts not applicable (Go-only
  lane; `gofmt`/`go vet` gates applied instead).
- Task-list checkboxes untouched per lane instructions.

## Implementation

`compiler/lsp.go` (G-owned completion/index section, following the G03
precedent of extending `lsp.go` — the I44 retirement gate pins the
launcher to six production files):

- Wire: `textDocument/completion` request handling with `-32602` on
  invalid params, `completionProvider` capability with `.`/`:` trigger
  characters, null decline where the snapshot cannot support the query.
- `compWalker`: one-file walk mirroring the G03 `refWalker` traversal
  order, retaining scope spans plus cursor-context markers (qualified
  names with usage, types, emits bounds, with-pins, member/method
  names, callees, constructor names, callable gaps, declaration
  leads/gaps, declaration sites, literals).
- Shared with G03: `calleeFunction` (checked-function callee
  resolution) and `knownReceiverRecord` (annotation-known receiver
  set) extracted as free functions; G03 delegates unchanged (its 13
  tests still pass).
- Candidates per context (each item: label, LSP kind, checked
  detail, provenance; sorted deterministically; freshly built per
  request, no snapshot state escapes):
  - general body: visible locals (order- and shadow-correct,
    captures included) + World symbols minus shadowed names +
    12 statement/expression keywords;
  - call callee: call-eligible symbols + callable-annotated locals
    with exact `(params) -> result` arity; callable callee:
    reference-eligible symbols only, no locals;
  - with pin: the resolved callee's near parameters only;
  - member: record fields / methods behind known receivers only;
  - qualified `pkg::`: imported-package members under the
    position's usage (C-D catalogue path, e.g. `browser::mount`),
    aliases on the package part;
  - type / emits-bound / constructor / header / signature /
    top-level / bare-callable-gap: the exact sets in the table
    below; declaration sites, literals, comments, and uncovered
    native declarations yield an empty list.
- Keyword sets (closed; unselected grammars contribute nothing):
  general 12, type 5 (`bool float int str void`), top 9
  (`+ error fn record variant`), header 4, signature 4, with-gap 1.
  Strict name positions offer no keywords.

## AU-LSP-completion legs (24 tests, all pass)

| # | Leg | Test | Result |
| --- | --- | --- | --- |
| 1 | Capability advertised, G01–G03 caps intact | InitializeAdvertisesCompletion | pass |
| 2 | Body-local + input scope, other-scope names absent | CompletionLocalScope | pass |
| 3 | Initializer order (own name invisible) | CompletionOrderSensitive | pass |
| 4 | Captures in, shadowed/later names out, outer returns | CompletionCapturedAndShadowed | pass |
| 5 | Local suppresses shadowed World symbol | CompletionShadowedSymbolSuppressed | pass |
| 6 | Unparseable buffer and unopened file decline null | CompletionUnparseableDeclinesNull | pass |
| 7 | Caller repair: callee set + exact arities, no keywords | CompletionCalleeArity | pass |
| 8 | Callback extraction: reference-eligible only | CompletionCallbackExtraction | pass |
| 9 | With pins: near params only; unknown callee empty | CompletionWithPins | pass |
| 10 | Shared-record fields; unknown/local receiver empty | CompletionMemberFields | pass |
| 11 | Chained method with exact arity, not fields | CompletionMemberMethod | pass |
| 12 | Constructor: constructible only | CompletionConstructor | pass |
| 13 | Declaration sites empty (rename's territory) | CompletionDeclSiteEmpty | pass |
| 14 | Type position: type-eligible + type keywords | CompletionTypeContext | pass |
| 15 | Emits bound/gap: errors only, no keywords | CompletionErrorBound | pass |
| 16 | Top level: starters + type names, no locals/functions | CompletionTopLevel | pass |
| 17 | Header: header keywords + file decls + aliases | CompletionHeader | pass |
| 18 | Signature gaps: section keywords only | CompletionSignature | pass |
| 19 | Bare callable gap: `with` only | CompletionCallableWith | pass |
| 20 | `browser::` members (catalogue, no arity guess), aliases, unimported empty | CompletionQualifiedBrowser | pass |
| 21 | String and comment positions empty | CompletionStringAndCommentEmpty | pass |
| 22 | Warned file completes + publishes exactly its warning | CompletionWarnedFile | pass |
| 23 | Same request twice byte-identical (stateless) | CompletionDeterministic | pass |
| 24 | No unselected keyword in any context; general set exact | CompletionNoUnselectedKeywords | pass |

## Scope negatives (all pinned by tests)

- Shadowed outer binding never surfaces beside the inner one, and the
  arm binding never leaks past the match (leg 4; each name offered
  exactly once).
- Same-spelled inputs of other functions (`prefix`, `value`) absent
  from unrelated bodies (leg 2).
- Later bindings invisible inside their own initializers and matches
  (legs 3–4).
- A local shadowing a package function suppresses the unreachable
  symbol (leg 5).
- Locals declined as callable callees and with-pin callees, matching
  G03 `withName` (legs 8–9); unknown callees/receivers yield empty
  lists for the checker to diagnose (legs 9–10).
- Unimported packages yield nothing (leg 20); unparseable buffers
  decline null (leg 6).

## Verification

- `go test ./compiler/ -run 'TestG04' -count=1`: 24/24 pass (197s).
- `go test ./compiler/ -count=1`: full suite green incl. G01/G02/G03
  and the `TestNoPredecessorPaths` retirement gate (139s).
- `gofmt -l compiler/`: clean. `go vet ./compiler/...`: clean.
- Env: macOS darwin/arm64, go1.27.1.
- Commits (worker): `29715ce2` (wire + scope engine),
  `cb43cff6` (strict contexts), `5f3d8870` (lsp.go merge +
  qualified/parity tests).

## Handoff to G05 and authoring consumers

- Service entry: `completion(snapshot, file, line, character)` plus
  the `textDocument/completion` wire in `compiler/lsp.go`; item
  shape `{label, kind, detail, documentation}` with the kind/detail
  conventions above.
- Rename (G05) reuses the same scope/identity model: binding
  visibility (`from` offsets), `calleeFunction`, and
  `knownReceiverRecord` are shared code with references/completion.
- Known boundary (unchanged from G02/G03): legit method calls on
  checked result types and function-local receivers decline member
  candidates — their nominal types need checking, and the service
  guesses nothing. The method leg therefore chains onto a module
  value (resolve-clean, check-diagnosed) to pin the method path.
