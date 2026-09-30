"""Produce and submit three fully rewritten advisory architecture consultations."""
import json, os, sys, urllib.request
from pathlib import Path
OUT=Path(__file__).resolve().parent
STATE={
'goal':[
'Can needs the entire authorized editor repair: every independently diagnosable static error and warning with an accurate source token or expression range, current grammar, useful semantic features during typing, and a verified current installation in Cursor. No external users require compatibility with former APIs, spellings or goldens. Runtime execution, assertion execution and network operations must remain outside keystroke analysis.',
'The user authorized completing Can editor support and installing the reviewed result in Cursor. Coverage must include all independent static findings at their true source ranges, up-to-date syntax coloring, and semantic assistance for unfinished buffers. There are zero external consumers to protect through old interfaces or expected outputs. Editing must not run Can programs, execute assertions or perform network effects.',
'The requested endpoint is a repaired and verified Cursor extension: precise underlines for every statically ascertainable independent error/warning, grammar matching current Can, and working semantic editor operations while code is incomplete. Historical syntax, APIs and golden results impose no compatibility duty because Can has no external users. Keep automatic analysis inert, excluding application, test-case and network execution.'
],
'architecture':[
'At 9c283544, syntax.Lex stops at its first diagnostic and syntax.Parse catches one panic, discarding the partial File. project.LoadWithOverlay returns the first bad source. resolve.Build defines names then exports/imports/signatures in shared mutable state. types.Builder retains a sticky failure. checkProgramForTarget shares bindings, specializations, registry, warnings and SQL sites across passes. A bare append-and-continue can consume corrupt or absent state. CheckSnapshot currently resolves twice and publishes only one error, or warnings after success.',
'The inspected revision 9c283544 has first-failure lexer, parser and project loader behavior; the parser recovery loses its partially built File. Resolution mutates a common world through name/export/import/signature phases, and a failed types.Builder remains failed. Program checking reuses mutable bindings, specialization data, registry, warnings and SQL records. Continuing blindly after an error is unsafe. The current snapshot duplicates resolution and delivers either its first error or the successful check warnings.',
'Evidence from 9c283544: lexing exits after a diagnostic, parsing recovers a single panic without keeping the partial File, and overlay loading aborts on the first invalid source. Resolver passes share mutable symbols; type construction has a persistent failure field; checker passes share bindings, registry, specializations, warning collection and SQL-site data. These dependencies prevent safe naive continuation. Snapshot diagnosis performs resolution twice and exposes one failure unless checking completes and supplies warnings.'
],
'positions':[
'Original-byte to UTF-16 conversion already passes BOM/CRLF/emoji/combining-character tests. Qualified names and call callees retain spans; operator AST fields only retain text. Some semantic failures use the entire call/declaration, others are unlocated and land on line one, and multiline syntax spans are widened to a line. Duplicate assertion names, type/native/SQL contract failures and project configuration attribution are concrete gaps. Notes become warnings.',
'Tested coordinate conversion handles original bytes, UTF-16, BOM, CRLF, astral characters and combining marks. The AST already locates qualified names and callees but drops operator token ranges. Diagnostic producers sometimes choose full expressions/declarations or omit location entirely; syntax conversion broadens cross-line spans. Verified affected cases include repeated assertion labels, type/native/SQL contracts, configuration errors and notes wrongly classified as warnings.',
'The byte/UTF-16 boundary has passing coverage for BOM and CRLF plus emoji and combining code points. Callee and qualified-name locations exist, whereas operators survive only as text in the AST. Remaining failures include overbroad expression/declaration underlines, spanless messages at the first line, cross-line syntax ranges replaced by whole-line ranges, misplaced duplicate assertion/type/native/SQL/configuration errors, and informational notes rendered as warnings.'
],
'editor':[
'The source has definition, hover, references, completion, rename and comment-preserving formatting. Independent refWalker and compWalker code already knows many scopes. A sibling syntax error can remove the Graph/World needed by features. Completion currently declines malformed buffers; hover/definition omit locals. New signature help, symbols for documents and workspaces, semantic tokens, folding, inlays, code actions and safe range/on-type formatting should reuse compiler facts. No guessed types/imports or synthetic repairs may enter published diagnostics or executable output.',
'Existing editor code implements navigation, hover, references, completion, safe rename and trivia-preserving formatting; refWalker and compWalker each encode scope knowledge. Graph/World loss after a malformed sibling disables assistance, with incomplete-buffer completion and local hover/definition already missing. The authorized additions are signatures, both document and workspace symbol views, semantic coloring, folds, hints, validated actions, and safe formatting of selected ranges or triggered during typing. Derive those from compiler evidence; speculative types/imports and temporary source repairs cannot become diagnostic or emitted-program facts.',
'Current facilities include formatting that retains comments alongside completion, references, rename, hover and definitions, using separate reference and completion walkers for scope information. Invalid sibling syntax can eliminate the shared graph/world. Local navigation/hover and completion on broken text are known gaps. Complete signature help, document/workspace symbols, semantic tokens, folds, inlays, compiler-validated fixes, selection-range formatting and on-type formatting using the same semantic facts. Never promote invented types, imports or repair text into errors or generated execution.'
],
'resources':[
'This is a 256GB MacBook with about 9.6GiB free. Broad builds and benchmarks remain deferred. Use one shared checkout per owner, bounded correctness tests, ordinary shared dependency caches and automatic temporary cleanup. A managed worktree isolates this repair from another Muse session. There must be no repeated full-project compiler run per discovered error or per editor feature. Analysis should be reusable per source/configuration version and invalidate on all dependency changes.',
'The host has a 256GB disk and approximately 9.6GiB available; do not run deferred performance measurements or broad build matrices. Keep checks bounded, use shared standard caches, reclaim owned temporaries and isolate this assignment in its managed worktree because another Muse session exists. Rechecking the entire project for each error or query feature is unacceptable. Reuse version-bound analysis and invalidate it whenever source or configuration dependencies change.',
'Laptop limits are material: 256GB capacity, around 9.6GiB remaining, no authorized benchmarks or large build sweeps. Run bounded verification with shared caches and owned-artifact cleanup; this task has a managed worktree separate from a pre-existing Muse session. Avoid whole-project repeats driven by error count or feature requests. Cache only appropriately versioned analysis and refresh it for every relevant source/configuration dependency edit.'
],
'correctness':[
'All independent units should keep their own valid errors/warnings when another unit fails; dependent units with unavailable prerequisite facts must be marked blocked rather than emitting invented unknown-name cascades. Malformed project headers and unterminated lexical constructs may legitimately prevent dependent semantic checks. Strict compilation must reject any errors and never emit partial invalid programs. Error recovery must use the canonical Can grammar/validators. No global phase shortcut may permanently omit healthy unrelated functions.',
'A failure in one unit must not erase trustworthy diagnostics in independent units. Missing prerequisites should block dependent evaluation instead of generating spurious unknown symbols. Broken headers or unclosed lexical structure can prevent dependent semantic results. Compilation remains unsuccessful whenever errors exist, with no partial invalid emission. Recovery must share Can parsing and validation logic, and healthy unrelated functions must not be skipped merely because an earlier global phase failed.',
'Preserve independent findings despite errors elsewhere while recording blocked dependencies instead of manufacturing follow-on name errors. Semantic work can be unavailable where malformed headers or unterminated syntax remove essential facts. An error forbids successful compilation and partial invalid output. Canonical parser/checker logic remains authoritative; global early failure must not silently hide independent healthy-function diagnostics.'
]
}
QUESTIONS={
'recovery':{
'instructions':[
'Which architecture best satisfies the complete multi-diagnostic repair given the mutable compiler prerequisites and laptop limits? Weigh implementation risk, sound independence and long-term shared semantics. This classification advises a reviewed design; it is not correctness proof.',
'Select the strongest recovery implementation for all independent static diagnostics under the described compiler-state hazards and host constraints. Compare sound coverage, engineering risk and semantic reuse; the result is advice that still requires code review and tests.',
'Choose the recovery strategy most suitable for delivering the entire diagnostic contract with these dependency and resource facts. Balance reliable independence against change risk and maintenance; your selection cannot establish correctness by itself.'
],
'criteria':{
'canonical_recovery':[
'Refactor canonical compiler stages to accumulate positioned diagnostics and retain partial valid state. Track declaration/dependency validity, isolate failed mutable work before committing, and continue independent units. CLI and editor share validators; CLI never emits when errors remain. Higher refactor cost buys systematic coverage without repeated full builds.',
'Add sound collection within the compiler itself: preserve validated partial results, mark invalid or blocked units, and commit declaration/checker state only after successful validation. Both editor and CLI use those routines, with compilation gated on zero errors. This is a larger change but supports comprehensive recovery without project replay.',
'Use a unified compiler analysis that records exact issues, commits valid unit state transactionally and propagates explicit invalid/blocked dependencies while unrelated checks proceed. Reuse it for editor and strict compile, refusing emission after any error. Accept the substantial refactor to avoid per-error whole-project execution.'
],
'closure_isolation':[
'Partition the real dependency graph into independent validation closures and invoke the existing strict validators on isolated in-memory compiler states, caching shared prerequisites and never rewriting source. Merge located findings and block tainted dependents. This may reduce shared-state surgery, but existing global validators still require new per-closure entry points and state partitioning, exact dependency closure, duplicated compiler state and careful cross-closure global checks.',
'Discover independent source dependency closures and check each using isolated state with the current validators; retain common prerequisite caches and preserve all source text. Combine positioned outcomes, blocking contaminated dependents. Potentially less invasive changes still require per-closure validation entry points and partitioning of the currently global state, alongside dependency-graph complexity, repeated state and global consistency joins.',
'Keep validation routines mostly strict and execute them in separate in-memory contexts for genuine independent dependency closures, sharing cached prerequisites without masking or editing source. Aggregate accurate diagnostics and block dependent failures. This could limit refactoring, but adapting the global validators requires new closure-scoped entry points and state partitioning as well as closure discovery, state duplication and reconciliation of project-wide invariants.'
],
'focused_prototype_first':[
'The supplied evidence is not enough to select safely; perform a bounded recovery prototype focused on shared type-builder failure and dependency commits before committing to either architecture. Keep the entire requested repair scope and avoid benchmarks.',
'Withhold architectural commitment until a small correctness prototype tests isolation of failed type construction and valid dependency commits. Do not reduce the authorized feature/diagnostic scope or launch performance work.',
'First resolve the uncertain failure-state boundary with a tightly scoped type-builder/dependency-commit experiment, then choose the design. The full repair remains required and measurement runs remain excluded.'
]}
},
'scheduling':{
'instructions':[
'Which request scheduling design should accompany the repair, assuming the chosen sound compiler recovery and required version correctness? Compare responsiveness and race complexity on one laptop without assuming unmeasured speedups.',
'Choose the editor request execution model to pair with correct recovered analysis. Consider timely cancellation, version coherence and implementation complexity on this constrained host; no benchmark benefit has been established.',
'Select how the server should schedule its repaired analysis and queries, given the need for fresh versions, bounded laptop work and no measured performance claims. Judge responsiveness against concurrency failure risk.'
],
'criteria':{
'single_worker':[
'Keep protocol input responsive while one analysis worker processes immutable versioned snapshots. Coalesce superseded edits, cancel/check staleness at compiler stage boundaries, reuse completed snapshot/index data for queries, and publish only current per-root versions. Serialize mutable compiler work; add deterministic race/lifecycle tests.',
'Decouple message intake from a single compiler worker using immutable version snapshots and one current job per root. Replace obsolete queued edits, observe cancellation between stages, serve queries from reusable completed analysis and discard stale publications. Do not run parallel mutable checks; verify ordering with deterministic tests.',
'Use responsive protocol reading plus one serialized analysis executor. Freeze input versions, merge newer edits into pending work, honor cancellation at safe stage points and reject out-of-date results. Reuse snapshot/index facts for features and test scheduling/lifecycle deterministically instead of parallelizing compiler mutation.'
],
'synchronous_cache':[
'Retain the synchronous protocol loop with one cached immutable analysis per root/version. All diagnostics/features reuse it; edits invalidate it and cancellation takes effect only between complete requests. This avoids worker races but input cannot be consumed while a long analysis runs.',
'Continue reading and executing requests serially, caching results by complete root/version identity for every feature. Invalidate on edits and observe cancellation at request boundaries. It has simpler state ownership but cannot react to new input during a lengthy check.',
'Leave protocol dispatch synchronous and share a version-keyed analysis cache across queries. Refresh after relevant changes and handle cancellation between requests only. This reduces concurrency risk at the cost of blocking message intake for the duration of compilation analysis.'
],
'focused_evidence_first':[
'Neither scheduling choice is justified yet; first exercise bounded cancellation/edit ordering correctness scenarios and inspect existing stage boundaries, without performance measurements or expanding concurrency.',
'Postpone the scheduler choice until small correctness probes establish stage cancellation and queued-edit behavior; keep benchmarks and extra parallelism out of the probe.',
'Obtain targeted correctness evidence for interruptible stages and edit ordering before selecting either request model, without timing studies or additional concurrency.'
]}
}}
for key,variants in STATE.items():assert len(variants)==3 and len(set(variants))==3,key
for name,q in QUESTIONS.items():
 assert len(set(q['instructions']))==3,name
 for key,variants in q['criteria'].items():assert len(variants)==3 and len(set(variants))==3,(name,key)
requests=[]
for i in range(3):
 request={'model':'jev-latest','state':{k:v[i] for k,v in STATE.items()},'questions':{k:{'type':'choice','instructions':v['instructions'][i],'criteria':{c:d[i] for c,d in v['criteria'].items()}} for k,v in QUESTIONS.items()}}
 requests.append(request)
 (OUT/f'request-{i+1}.json').write_text(json.dumps(request,indent=2)+'\n')
(OUT/'wording-audit.json').write_text(json.dumps({'distinct_explanatory_fields':True,'manual_semantic_review':'All three packets preserve the same revision, architecture, observed failures, feature scope, resource facts, correctness requirements and alternatives. Objective, context, evidence, constraints, instructions, questions and every option were rewritten; stable schema keys and technical identifiers remain. No candidate or constraint was omitted in a rewrite. Agreement is advisory, not proof or guaranteed bias removal.','external_docs':['https://docs.typesafe.ai/api.md','https://docs.typesafe.ai/primitives/choice.md']},indent=2)+'\n')
if '--send' in sys.argv:
 for i in range(1,4):
  req=urllib.request.Request('https://api.typesafe.ai/v1/systemone',data=(OUT/f'request-{i}.json').read_bytes(),headers={'Authorization':'Bearer '+os.environ['TYPESAFE_API_KEY'],'Content-Type':'application/json'})
  with urllib.request.urlopen(req,timeout=60) as r: answer=json.load(r)
  (OUT/f'response-{i}.json').write_text(json.dumps(answer,indent=2)+'\n')
  print(json.dumps({'round':i,'model':answer.get('model'),'answers':answer.get('answers'),'usage':answer.get('usage')}),flush=True)
