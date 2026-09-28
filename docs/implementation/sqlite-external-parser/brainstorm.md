**Can SQL architecture brainstorm — research, not an implementation decision**

The shortlist should change. The strongest candidates are an existing pure-Go SQLite parser and the existing upstream Tree-sitter Go packages. Creating another repository just to hold our copied C should be a fallback.

**A correction to the original rationale**

Can's provenance says no Go module ships the selected SQLite grammar. The exact revision Can already pinned contains both a [Go module](https://github.com/defin/tree-sitter-sqlite3/blob/7f69bb66845beaac48d467f4f7d107ea2002865e/go.mod) and a [Go binding](https://github.com/defin/tree-sitter-sqlite3/blob/7f69bb66845beaac48d467f4f7d107ea2002865e/bindings/go/binding.go). That statement is incorrect. I repeated it before verifying upstream; the earlier recommendation for a new owned module was premature.

There is a real complication: the grammar uses Tree-sitter ABI 15, while its declared Go runtime dependency supports only ABI 14. A compatible official runtime revision exists. The upstream binding test only constructs a language object, so it does not establish successful parsing. This requires a correct dependency pin and qualification; it does not establish a need to vendor the source in Can.

The official runtime's ABI-compatible source is identifiable through the [Go proxy's version record](https://proxy.golang.org/github.com/tree-sitter/go-tree-sitter/@v/v0.25.0.info), although the corresponding live GitHub tag is absent. A reproducible implementation must resolve that provenance detail rather than copy a version string blindly.

**What we are trying to improve**

Several different goals were bundled together in the discussion:

| Goal | What would actually address it |
|---|---|
| Make language statistics reflect authored code | Accurate vendored/generated classification |
| Remove copied upstream code from Can's tracked tree | External module or another source-distribution boundary |
| Remove SQLite's C build and custom bridge | A suitable native Go parser, or a different SQL contract |
| Reduce total disk, build or executable cost | Evidence about the complete dependency graph; moving files alone is insufficient |
| Reduce maintenance and inconsistency | Reuse established packages behind one Can-owned analysis contract |
| Simplify the language/compiler | Reconsider which SQL properties Can promises to prove |

The current tracked SQLite directory contains 88 files and 6,729,649 bytes. The generated parser is 5,809,039 bytes, or 86.32% of that directory. The two bridge files total 277 physical lines and 7,985 bytes; their tests and Can's higher-level SQL analysis are additional authored code. These counts do not reproduce GitHub's percentage or establish that half of all project work is SQLite.

**What the parsers currently buy us**

All three dialect backends already feed one [compiler-owned analysis interface](/Users/vince/Projects/can-lang/compiler/internal/sql/parser.go). They establish offline syntax acceptance, statement count and kind, parameter occurrence locations, parameter coverage, top-level LIMIT structure and admitted RETURNING forms. Exact source positions let Can construct native bound-value templates without mistaking a question mark inside a string or comment for a parameter.

They do not establish that query columns match the real database schema. The [compiler checks declared record types](/Users/vince/Projects/can-lang/compiler/internal/check/sql_descriptors.go); [runtime decoding checks actual rows](/Users/vince/Projects/can-lang/runtime/platform/sql/values.ts). A SELECT LIMIT also does not bound all database work or prove the absence of effects inside functions or nested SQL. Those are important limits when judging the value of this machinery.

Execution already uses [Bun's native SQL interface](https://bun.sh/docs/runtime/sql). Replacing the compiler parser does not require replacing the database driver.

**Go alternatives: a parser is different from an engine**

| Candidate | What it is | Fit for Can |
|---|---|---|
| [sqlc-dev/meyer](https://github.com/sqlc-dev/meyer/tree/v0.1.2) | An independently authored, dependency-free Go SQLite parser | Strong candidate: typed AST, byte ranges, parameter nodes, LIMIT/RETURNING structure and positioned errors |
| [rqlite/sql](https://github.com/rqlite/sql) | Another Go SQLite parser | Real candidate, but documented syntax gaps and an offset adaptation requirement |
| [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) | SQLite's C engine translated into Go | Useful cgo-free engine; the compiler's required public AST/span interface is not established |
| [zombiezen/go-sqlite](https://github.com/zombiezen/go-sqlite), [glebarez/go-sqlite](https://github.com/glebarez/go-sqlite) | APIs/drivers using the modernc engine | Different interfaces to the translated engine, not separate parser implementations |
| [ncruces/go-sqlite3](https://github.com/ncruces/go-sqlite3) | Currently SQLite Wasm translated into Go using wasm2go | Another cgo-free engine path; still not a demonstrated replacement for Can's syntax-tree analysis |

Meyer v0.1.2 predates Can's SQLite decision, requires Go 1.24, and has no module dependencies. Its upstream reports 21,326 corpus cases plus differential, roundtrip and fuzz testing against SQLite 3.53.4. Current [sqlc source imports it](https://github.com/sqlc-dev/sqlc/blob/main/internal/engine/sqlite/parse.go). These are promising adoption/API facts and upstream test claims, not results we reproduced. It is a young dependency and needs Can-specific qualification.

Rqlite's [scanner increments positions per rune](https://github.com/rqlite/sql/blob/master/scanner.go), so those offsets cannot be used directly to slice UTF-8 source bytes. A mapping could repair that boundary; complete syntax compatibility would remain a separate question.

Full engines can prepare SQL and expose binding metadata, but SQLite's [prepare API](https://www.sqlite.org/c3ref/prepare.html) returns executable statements and an uncompiled tail. Its [parameter-count API](https://www.sqlite.org/c3ref/bind_parameter_count.html) returns the largest binding index, not every occurrence's source span or its role in a LIMIT clause. Adding a complete second engine therefore needs more justification than the label “pure Go.” This review did not establish a mature independent Go reimplementation of the entire SQLite engine; the engine candidates above largely preserve SQLite by translation.

**The architectural options**

| Option | Benefit | Cost, uncertainty or changed promise |
|---|---|---|
| **1. Replace the SQLite backend with Meyer** | Removes SQLite's generated C, Tree-sitter runtime and custom C bridge; Can keeps its analysis contract | Different parser, version and options require correctness qualification; performance is unmeasured |
| **2. Import the existing grammar and official runtime Go packages** | Removes vendored sources and may remove the bespoke serialization bridge while keeping the current grammar | Native C still compiles; requires compatible ABI pin, object-lifetime handling and equivalent error/span behavior |
| **3. Use another Go parser such as rqlite/sql** | Removes SQLite C without inventing a parser | More known adaptation/coverage work than Meyer based on current evidence |
| **4. Maintain a small external binding module or fork** | Controlled patches, versions and source ownership | Another repository and release process; justified only by a concrete upstream gap |
| **5. Use a submodule or pinned source archive plus patches** | Explicit upstream origin; fewer directly tracked upstream files | Checkout/fetch/extraction steps, offline provisioning and cleanup; source and compilation still exist |
| **6. Store compact grammar and generate the C** | Tracks the authored grammar instead of large generated tables | Adds generator/version/toolchain work; generating only for dependency releases moves that work away from consumers |
| **7. Distribute a prebuilt native or Wasm parser** | Can reduce consumer C-build requirements | Native platform matrix or Wasm execution/translation layer, artifact integrity and portability work; no performance advantage established here |
| **8. Use one multi-dialect parser family** | Potentially fewer integration APIs | Shared AST does not prove accurate SQLite, PostgreSQL and MySQL behavior; would require a genuinely stronger candidate |
| **9. Move SQL checking into a separate tool** | Core compiler can avoid heavy parser dependencies; checking can remain available during development | Another pipeline/tool boundary; mandatory checked artifacts need reliable regeneration and provenance |
| **10. Use explicit bound-value slots and native runtime preparation** | SQL values flow directly to Bun templates; potentially removes full parsers from the compiler | Offline SQL syntax, statement-kind and LIMIT-shape guarantees must be removed or supplied elsewhere |
| **11. Build queries from a small typed query representation** | Statement shape, value slots and output-limit placement can be correct by construction | Creates a query API and dialect renderers; unrestricted SQL requires a clearly defined escape path |
| **12. Add schema-backed development checking** | Can verify tables, columns and some type correspondence beyond today's checks | Requires migrated schemas/engines, versioned evidence and a plan for runtime schema drift |

ZIP storage and language-statistics overrides are possible housekeeping measures. They do not eliminate the dependency or establish a better compiler architecture. In particular, a ZIP must be expanded before ordinary cgo compilation.

**Three coherent end states**

1. **Keep checked raw SQL, simplify its dependencies.** Preserve the current useful analysis contract and use maintained external parser modules. Meyer is the stronger structural simplification to qualify; official Tree-sitter bindings are the path with fewer expected grammar changes.
2. **Keep native SQL, make compile-time checking optional or separate.** Introduce explicit value slots that lower directly to Bun templates, retain runtime row decoding/cardinality/resource ownership, and offer an engine/schema-aware development checker. This is a meaningful reduction in compiler responsibility, with an explicit change to its promises.
3. **Offer a small constructed-query API plus native SQL escape.** Common queries get structure and bound values by construction; advanced dialect SQL remains possible under clearly stated checks. This is worth exploring if query construction improves authoring, not simply to make a language pie chart smaller.

Moving validation to artifacts deserves care: exact SQL bytes, descriptor settings, dialect, parser/checker versions and any relevant schema snapshot must identify what was checked. A hash catches accidental staleness; it does not prove a fabricated artifact is valid. Complexity is relocated, not magically removed.

Keeping raw SQL with old placeholder spellings while deleting the parser still requires dialect-aware lexing. Replacing it with a few regular expressions would silently recreate a less reliable parser. Explicit parameter slots are the design change that removes that particular need.

**My recommendation**

Do not create another owned parser repository yet. Keep two finalists:

- **Meyer**, for a meaningful simplification of SQLite's implementation dependency.
- **The existing grammar with compatible official Go bindings**, as a baseline with less expected grammar change and a credible fallback.

A bounded qualification should run both through the same intended SQL contract: exact UTF-8 spans, repeated and numbered parameters, quoted/commented lookalikes, multiple statements, CTEs, LIMIT forms, malformed syntax and RETURNING. It should verify deterministic cleanup and resource handling, then check distribution/dependency provisioning. Differences should be judged against SQLite and the intended Can contract; zero external users means preserving an old error just to keep a golden file is not a requirement.

The first probe can establish the upstream-binding baseline, followed by Meyer against the same cases. If Meyer satisfies the contract with acceptable resource behavior and maintenance confidence, it is my preferred direction for the SQLite backend. Neither option removes PostgreSQL's cgo requirement from the whole compiler. No speed, memory or executable-size win should be claimed before measurement, and no broad measurements are needed for this brainstorming stage.

Separately, the most worthwhile product-level brainstorm is explicit SQL value slots plus optional/schema-aware checking. That deserves an authoring comparison before committing to another query language or a mandatory artifact pipeline. The owner has not yet selected weaker compile-time guarantees.

**Evidence and limits**

Three independent research lanes reviewed upstream dependencies, Go engines/parsers, and Can's contracts/footprint. Three freshly reworded Jev consultations preferred probing the official bindings first, with selection probabilities of 0.59, 0.77 and 0.52; Meyer remained the only substantial competing option. That advice concerns probe order under the current contract, not the eventual winning architecture. There was no selected-option disagreement, but the probability variation discourages treating unanimity as certainty.

[Requests, responses, wording audit and assessment](/private/tmp/can-sql-full-brainstorm-evidence-2026-09-28.json) are retained as compact design evidence. The earlier extraction proposal is annotated with the correcting discoveries. This work used source inspection and tracked-file counts only: no builds, tests, dependency installs, clones, benchmarks, implementation or publishing. No temporary build workspaces or private caches were created.
