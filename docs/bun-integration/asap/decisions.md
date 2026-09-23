# Decisions and bounded gates

These are recommendations and implementation instructions for the separate Bun milestone, not retroactive edits to the ongoing language-fix contract.

| ID | Disposition | Evidence and next action |
|---|---|---|
| D01 | Use native library operations for ordinary transformations/I/O. | Add catalogue/runtime binding with existing Can contracts. No custom filesystem, HTTP/SQL driver, parser, cipher or calendar engine. |
| D02 | Prefer Bun.SQL SQLite with `safeIntegers:true`; qualify before finalizing. | Probe: `bigint:true` alone rounded 9007199254740993n; adding `safeIntegers:true` preserved it. Direct `bun:sqlite` was exact too. Test async transaction, rollback, busy, persistence and cancellation. |
| D03 | Gate SQL grammar admission; keep both SQLite and MySQL ASAP. | Current libpg_query is PostgreSQL-specific. Implement dialect propagation now, but do not pretend it parses the other grammars. See G-SQL below. |
| D04 | Start event surface comparison immediately; select syntax from it. | Streams, process output and WebSocket need repeated dispatch and ownership. Preserve grouped state and attached assertions. No universal primitive or functions-only veto is preselected. |
| D05 | Document parser-specific numeric/duplicate semantics. | TOML rejected the large integer; YAML/JSON5/JSONL rounded it. Native parsed-value projection is not lossless token preservation. Existing exact JSON behavior stays intact. |
| D06 | Strict JSONL must verify the whole input. | Native `parse` accepted a valid prefix before invalid input. Use checked parseChunk or bounded line framing with Can's existing native-backed exact JSON path. |
| D07 | Native Markdown HTML stays ordinary text. | Default output retained script and javascript: URL. Safe mode requires the existing safe HTML rules and a verified callback path, not a cast. |
| D08 | Require explicit CSRF secret/session identity. | Native A-session token failed B-session verification in probe. Shared secret alone is not session binding. |
| D09 | Application filesystem roots describe resolution, not a sandbox. | Native symlink and race behavior require explicit semantics. Compiler project confinement is a separate concern. |
| D10 | Bound process output and own termination. | Probe verified argument arrays and nonzero exit. Pipes must drain concurrently; process-tree guarantees require targeted OS evidence. |
| D11 | Keep local assertion ownership. | Reusable fixtures supply input, not global exemption. Extend the completed LF provider rather than starting a competing test subsystem. |
| D12 | Preserve all thirteen ASAP tasks. | SQL/event/HTML gates block dependent subparts only. Continue independent work, record blockers, never quietly move a family to later. |

## G-SQL: decide actual grammar validation before claiming multi-dialect admission

Owner: B1-02 frontend slice; B1-03 consumes its result. Output: a decision record, executable corpus, negative source spans and a qualified dialect backend interface.

1. Extract the current descriptor guarantees: single statement, approved statement kind, placeholder spans, top-level limit/cardinality and source diagnostics. Keep them as acceptance rows.
2. Build a compact corpus containing quoted semicolons, comments, dollar-like text, MySQL backticks and escapes/SQL modes, SQLite parameter spellings, CTEs, RETURNING, multiple statements and LIMIT variants. Include valid and invalid statements in every supported dialect.
3. Compare a maintained pinned dialect grammar dependency against native preparation. Evaluate offline availability, grammar coverage, accurate spans, license/build cost and native-lowering policy. A host compiler parser dependency is distinct from an npm runtime driver, but it still requires justification and qualification.
4. Native prepare can depend on schema or a running server; runtime syntax rejection is not an offline compiler proof. Do not run arbitrary SQL against a user's database during build. If preserving all current guarantees needs a maintained parser, make that explicit and qualify it. If guarantees change, reconcile the authoritative contract with concrete before/after evidence before implementation proceeds.
5. Keep static query fragments and value parameters separate. A generic string template alone does not establish statement kind or bounded result semantics. No ad hoc regex lexer, manual placeholder rewriting or semicolon split.
6. Consult Jev in three freshly rewritten packets for the newly concrete backend alternatives once this corpus supplies relevant evidence. Save the requests/responses and investigate divergent advice. This preparation does not choose an unresearched parser package.

Exit: a genuine dialect-aware validation implementation is selected and its corpus passes. Until then SQLite native-adapter prototypes can proceed, but full B1-02/B1-03 cannot be marked complete.

## G-EVENT: choose source form without turning preparation into an indefinite redesign

Owner: B1-05; first bounded spike before wiring long-lived APIs. Output: nine complete comparative programs (three workflows × three surfaces), a behavior table and one selected grammar/API record.

The three workflows are: (1) read a large file, decode JSONL and stop after a typed record predicate; (2) stream child stdout/stderr, update grouped progress state, cancel on a deadline and reap; (3) accept a socket, retain typed session state, handle text/binary/drain/close and stop on handler failure.

Each surface must show acquisition, repeated dispatch, grouped state, explicit handler error sets, normal end, failure, backpressure, cancellation, cleanup and locally attached fixture tests. Compare ordinary operations with named handlers; domain-specific native forms; and a shared event form. Count hidden scheduling behavior, manual boilerplate, diagnostic precision and compiler/runtime machinery. Do not decide on character count alone.

Use the existing stream/process/socket native probe as a starting harness. A match-shaped native declaration may win; ordinary value-match semantics must remain one-shot. Pick a grammar only after these examples, then implement format/check/IR/emission/assertions together. Continue SQL and pure library capabilities while this is being resolved. Fresh Jev consultation is required for the eventual concrete syntax decision, because the preparation consultation only judged sequencing.

Exit recorded 2026-09-23: nine programs, behavior table and sketches in [b1-05-comparison](evidence/b1-05-comparison/README.md); probe extensions `stream_pull`, `stream_filesink`, `stream_alias`, `stream_abort`, `stream_process_reader` in [probe results](evidence/native-probe-results.json); three fresh Jev rounds with rewrite discipline and disagreement audit in [consultations-b1-05](evidence/consultations-b1-05/decision-audit.md). Selected: shared-event pull (opaque `stream::reader<T>`/`stream::writer` with `read_many`/`write_some`/`close`/`cancel`), no new grammar, drain-by-demand, repeated-selector FIFO transcripts, library consume deferred. Jev preferred library 3/3; overridden on B1-06.02 incremental reuse, B1-05.07 same-path fixtures, caller-visible acceptance rows, and the internal incoherence of library+fifo. No parser/checker/IR files are introduced: pull needs catalogue entries only.

## G-HTML: verify that native callback output can satisfy Can safe HTML

Owner: B1-12 safe-render slice. Output: callback trace for plain text, entities, nested emphasis/code, raw HTML and hostile links; a safe construction prototype; browser-level hostile corpus.

Bun's documented callbacks receive accumulated child strings rather than a promised typed AST. They are synchronous; async Can handlers cannot be passed directly to the native renderer. The initial safe adapter must keep native callbacks private and synchronous, and public callback/structured APIs need a separately qualified staging contract. Verify whether native callback boundaries permit reuse of the Can builder without double escaping or trusting unvalidated child strings. Raw-HTML suppression and URL filtering are necessary but not a complete proof. If a private validated fragment bridge is needed, specify its invariant and test every way source text reaches it. Do not make this bridge publicly constructible.

String rendering can be completed independently. Safe mode and any claimed structured AST remain incomplete until qualified. Do not introduce an external sanitizer or a handwritten Markdown parser to bypass the gate.

## Three fresh Jev consultations completed for this preparation

The [audit](evidence/consultations/pre-dispatch-audit.md), all three requests, responses and SHA metadata are saved in [consultation evidence](evidence/consultations/). The TypeSafe skill and live API/Choice documentation were used. Every explanatory context, instruction and option description was rewritten, with facts and alternatives preserved.

| Question | Selected in all three | Selected-option probabilities |
|---|---|---|
| SQLite backend preference | Qualify unified Bun.SQL with safeIntegers, fallback only on demonstrated gate failure | 0.99 / 0.82 / 0.95 |
| Event syntax sequencing | Immediate complete-program comparison alongside independent APIs | 1.00 / 0.70 / 0.99 |
| Typed format guarantee | Bounded native semantics with explicit precision limitations | 1.00 / 1.00 / 1.00 |
| Markdown trust | Ordinary string plus separately qualified safe rendering | 0.98 / 1.00 / 0.99 |

Model: `jev-1.13.0`. There was no selected-option disagreement. Packet 2 assigned 0.29 to immediately committing a universal event syntax and 0.18 to direct SQLite; this wording sensitivity means unanimity is not proof. The native safeIntegers result supports qualifying the common SQL path, but unresolved async transaction behavior remains a real fallback criterion. The existing lack of nine complete comparison programs prevents the event advice from authorizing a syntax choice. Parser/HTML recommendations also follow directly from observed data loss and unsafe output; classifier agreement adds no correctness guarantee.

These consultations did not evaluate a concrete SQL parser dependency, approve final grammar, establish browser safety or prove all thirteen implementations. Those conclusions need the gates and acceptance evidence above.
