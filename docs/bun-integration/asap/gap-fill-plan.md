## Goal

Close the five gaps recorded in the ASAP B1 completion report, either by
shipping the missing behavior with tests and docs or by promoting each
gap to a confirmed wontfix with recorded rationale. No gap stays
implicitly open.

## Success Criteria

- Each gap below is either implemented (catalogue, runtime, tests,
  example/docs updated) or closed as wontfix with a decisions.md entry
  explaining what was tried and why it stops.
- `bun test runtime/test/`, `go test ./...` (archive + live services),
  `make catalogue-check`, `modcheck`, and the fresh-emit strict `tsc`
  gate are all green at the end.
- No new external dependencies; all native behavior pinned to Bun 1.4.2
  with probe evidence.

## Context And Current Facts

The [completion report](completion-report.md) records five gaps:

1. **B1-12 `render_with` deferred.** Native Markdown callbacks are
   synchronous string transforms; Can handlers are async
   Completion-returning functions and cannot be passed in directly
   (see `b1-12-markdown.md` "Callback execution constraint").
   `runtime/platform/markdown.ts` keeps native callbacks private and
   synchronous behind a two-phase bridge.
2. **B1-07 vacuous tasks.** (a) B1-07.07 event-primitive syntax:
   G-EVENT selected the shared-event pull surface
   (`stream::reader<ws::event>`, no new grammar) after nine
   comparison programs (`evidence/b1-05-comparison/`, decisions.md
   G-EVENT section). The queue task only fires "if the primitive
   wins" — it did not. The G-EVENT section requires fresh Jev
   consultation for any eventual concrete syntax decision.
   (b) B1-07.08 ws pub/sub: no probe coverage exists (zero
   `publish`/`subscribe`/`topic` hits in
   `evidence/native-probe-results.json`). Bun documents a native
   topic API on `ServerWebSocket` (`.subscribe`, `.unsubscribe`,
   `.publish` excluding self) plus `Server.publish` (all
   subscribers); see Sources.
3. **B1-11 JSONL byte offsets.** `runtime/codec/jsonl.ts` frames over
   bytes, so offsets are computable, but `codec::invalid_data` carries
   only `{path, reason}` where path is a value pointer
   (`/<index>/<member>`); a byte offset is not a location in the
   projected value.
4. **B1-11 YAML/JSON5 duplicate rejection.** Pinned probes show
   last-wins with no native duplicate metadata
   (`formats-native-results.json`); `runtime/codec/formats.ts`
   projects parsed values only.
5. **B1-06 incremental multipart.** `runtime/platform/multipart.ts`
   hand-parses framing over the already-buffered body; the inspected
   Bun server docs page shows no streaming multipart surface (weak
   negative — the pinned-binary probe is the real evidence).

## Constraints And Non-goals

- Follow AGENTS.md: native-first (compile to Bun operations, do not
  reimplement runtimes), zero-users/no-compat-burden (catalogue
  shapes may change), three-round Jev consultation with rewrite
  discipline for difficult design decisions.
- Binding units stay per-feature (`runtime_bindings.go`,
  `program_state.go`, `runtime_markdown.go` pattern); size guard and
  protected hubs per `execution-queue.md`.
- Non-goals: Bun version upgrades, revisiting settled G-EVENT/G-HTML
  outcomes beyond unit C1's explicit gate, unrelated refactors.

## Key Decisions

- **Spike-first with wontfix as an acceptable outcome.** Gaps 2a, 4,
  and 5 may be unfillable without violating native-first or settled
  architecture; each gets a bounded spike ending in a ship/wontfix
  decision recorded in decisions.md. A confirmed wontfix counts as
  closed.
- **Order by certainty, not queue number: D, B, A, F, E, C.**
  Front-load shippable units (offsets, pub/sub, render_with) and push
  the highest-risk/largest-scope units (dup rejection, streaming
  multipart, new syntax) last so early exits still bank value.
- **Catalogue edits serialize.** Units touching `catalogue.json` run
  sequentially (or batch regens) to avoid mirror conflicts; each
  catalogue change ships with `make catalogue` mirrors in the same
  commit.
- **No external parser dependencies.** Gap 4 must not be solved by
  adopting a non-native YAML/JSON5 parser; the spike may only consider
  native-compatible detection or confirm wontfix.
- **Gap 2a re-opens G-EVENT explicitly or not at all.** Unit C1 either
  produces the Jev-required syntax decision with comparison evidence
  or records the library surface as final; there is no silent third
  path.

## Recommended Approach

Treat each gap as an independent work unit with its own probe/spike
gate, following established B1 conventions (probe → catalogue →
runtime → fixture/integration/example → docs). Serialize only the
catalogue regens. Run units in D, B, A, F, E, C order.

## Work Plan

- **D1 — JSONL offset error-shape design (B1-11).** Decide how a byte
  offset is reported: extend `invalid_data` (new field breaks the
  closed shape — allowed, zero users), add a sibling error, or encode
  in `reason`. Three-round Jev consultation per AGENTS.md.
  Surfaces: catalogue errors, `runtime/codec/*` callers of
  `CodecIssue`, `budget.ts` Reason.
- **D2 — Implement offsets.** Track cumulative byte offsets in
  `createJsonlFramer`, thread through `projectJsonlRecord`, pin with
  runtime tests (whole-payload + streaming consume paths).
- **B1 — Probe Bun topic API (B1-07.08).** Pinned-binary probe of
  `subscribe`/`unsubscribe`/`publish`/`Server.publish`: return
  counts, self-exclusion, close/unsubscribe semantics, backpressure,
  binary payloads, invalid topics. Save probe + results JSON.
- **B2 — Ship ws pub/sub.** Catalogue ops, `runtime/platform/websocket.ts`
  adapter over existing session scope/ownership, loopback tests,
  fixture asserts, example extension, contract rows.
- **A1 — `render_with` staged-bridge design (B1-12).** Design the
  private sync-collect → async-replay bridge: event capture inside
  native callbacks, replay through checked Can handlers with error
  bound and ownership integration, no Promise stringification.
  Prototype in scratch; three-round Jev consultation on the design.
- **A2 — Ship `render_with`.** Catalogue ops, checker/emitter,
  runtime, hostile-corpus tests, maintained example, plan-doc update.
- **F1 — Streaming multipart spike (B1-06).** Probe pinned Bun for a
  streaming multipart surface. If none, prototype an incremental
  framer over `request.body` reusing `multipart.ts` reason taxonomy
  and budgets; measure peak memory vs whole-body path. Decision:
  ship incremental ops or confirm wontfix.
- **F2 (conditional) — Ship incremental multipart.** Catalogue ops,
  adapter, chunked/disconnect fixtures, contract rows.
- **E1 — Duplicate-detection spike (B1-11).** Determine whether
  YAML/JSON5 duplicate keys are detectable without abandoning native
  parsers (bounded approaches only, e.g. structural pre-scan with
  documented limits). Decision: strict mode or confirm wontfix.
- **E2 (conditional) — Ship strict duplicate mode.** Adapter,
  projection policy, tests, contract rows.
- **C1 — Event-syntax decision spike (B1-07.07).** Demonstrate the
  concrete pain of the library surface on the w3-socket workflow;
  run the G-EVENT-required fresh Jev syntax consultation with
  comparison evidence. Decision: new syntax (→ C2) or library final.
- **C2 (conditional) — Ship event syntax.** Parser/format/check/IR/
  emitter support together with attached event assertions, per the
  queue task; migrate ws docs.
- **Z — Final gate.** Full suites + tsc gate, decisions.md entries
  for every unit, queue/plan-doc updates, completion-report
  addendum.

## Validation Plan

- D2: `bun test runtime/test/formats.test.ts`-adjacent new asserts;
  `go test ./tests/integration -run 'Formats|JSONL'`.
- B1: probe script exit 0 with committed results JSON; B2: `bun test
  runtime/test/websocket.test.ts`, `go test ./tests/integration -run
  'WebSocket|websocket'`, example assert via `TestStdlibMaintained`.
- A2: new `runtime/test/markdown-handlers.test.ts` incl. handler-error
  and double-escape regressions; `go test ./tests/integration -run
  Markdown`; hostile corpus pinned.
- F1/E1/C1: decision record in decisions.md with evidence links is
  the validation artifact; F2/E2/C2 reuse their capability suites.
- Z: `bun test runtime/test/`, `go test ./...` with
  `CAN_BUN_ARCHIVE` + live MinIO/MySQL, `make catalogue-check`,
  `go run ./tools/modcheck`, fresh-emit `tsc -p tsconfig.json`.
- Highest-risk validation: C1 — a new-syntax decision would be the
  largest scope change; its Jev + comparison evidence must be
  airtight before C2 starts.

## Risks / Rollback

- C2 scope explosion (new grammar ripples through LSP/formatter).
  Mitigation: C1 gate defaults to library-final unless evidence is
  compelling; C2 is separately approved.
- Catalogue churn across parallel units. Mitigation: serialized
  regens (Key Decisions).
- Hand-written incremental parsing (F2) drifting from native
  semantics. Mitigation: differential tests against the whole-body
  path; same reason taxonomy.
- Rollback: each unit is independently revertible (feature files +
  catalogue entries); wontfix outcomes touch docs only.

## Open Questions

None. Priority order is part of this plan's approval; technical
unknowns are owned by the spike units above.

## Sources

- https://bun.sh/docs/runtime/http/websockets
- https://bun.sh/docs/runtime/http/server
