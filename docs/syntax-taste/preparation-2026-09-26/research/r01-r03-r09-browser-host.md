# Evidence packet: R01–R03, R09 — Can at `2cb1bc3` (2026-09-26)

Revision: `2cb1bc35435ab5075d51f9f945adfb0e1f9dfe64`. All file:line anchors verified against this tree. No repo writes made; no probes rerun (justified per topic below).

---

## R01 — Host integration (F-R01-01..05)

### (1) Current idiom + limitation

**Closed catalogue (F-R01-01).** The compiler owns the native surface; ordinary Can packages cannot add JS/npm bindings:

- `compiler/internal/browser/browser.go:22-31` — only two build profiles exist (`bun`, `browser`); "no worker profile and no arbitrary authored-JS profile."
- `compiler/internal/project/manifest.go:97` — manifest admits exactly `source_root`, `error_registry`, `project`, `dependencies`, `assets`, `sql`.
- `compiler/internal/project/assets.go:29-48` — admitted media are css/png/jpg/gif/webp/avif/ico/woff/woff2/txt/json/csv; `.js/.mjs/.cjs/.ts/.html/.svg/.wasm/.map` etc. are explicitly rejected (`rejectedAssetExt`). Bytes are signature-checked, never trusted by caller type.
- `compiler/internal/browser/browser.go:57-99` — transitive capability closure denies SQL/process/files/env/crypto/password/s3/io/stream/ws prefixes plus exact server-lifecycle ops; `CheckProgram` (`:130`) walks reachable calls, callable refs, specializations, initializers, and unreachable emitted functions. Assertion roots are excluded (run under Bun, never ship).
- `compiler/internal/resolve/symbols.go:392-403` — unresolved external package names fail; dependencies resolve only to Can packages.
- `compiler/internal/driver/browser.go:51-120` — verified pipeline: capability gate → supervised assertion staging under Bun → audited browser generation → content-addressed asset selection. `browser_bundle.go` + `audit.go`/`script.go` (B01 fix) structurally audit the emitted graph; quoted `"Bun."` compiles, real host imports reject.

**What already works without compiler changes:**

- Server-side bounded processes: `can.std.process@1::run` (`catalogue.json:10496`) — `Bun.spawn`, detached process group, no shell, piped stdio under caps, deadline enforcement, SIGTERM→SIGKILL escalation, scope-drain kill. Example at `examples/process/src/main.can:21` (`:30` call site).
- Literal-endpoint native connections: `compiler/internal/check/connections.go:32-83` — `endpoint` must be an http/https literal (no userinfo/query/fragment), `timeout_ms` 1–2³¹−1, `max_body_bytes`, static headers; emitted to runtime AI transport (`compiler/internal/emit/runtime_ai.go:22`). This is the fixed-origin point (F-R01-05): literal origins + confined paths stay unless R11 decides otherwise.
- Ordinary HTTP routes + shared JSON/form actions interoperate with independently implemented services at a data boundary (`examples/webhook/src/web/web.can:8`).
- Companion frontend calling a Can backend is architecturally possible today, but there is **no supported recipe** for contract parity, serving, auth, or release coordination; the reviewed browser build accepts no arbitrary JS/npm imports.

**Exact gap for in-process browser capabilities and third-party widgets (F-R01-02/03):** a rich editor, charting widget, identity/payment SDK, or missing browser API has no supported path — it requires compiler/runtime catalogue changes. The platform reconciliation (`platform-reconciliation.md:37-58`) narrows the earlier "every product feature a compiler project" claim: it applies to *missing in-process browser capabilities*, not all integrations. Three Jev consultations chose three different extension approaches (reviewed adapters / catalogue growth / companion frontend) — disagreement retained as a decision boundary, not a vote. No second-vendor-without-compiler-patch demonstration exists.

**Reopening bar (F-R01-04, P11 deferred):** compare **two concrete missing native operations** through both designs (reviewed adapter vs catalogue addition), covering errors, ownership, assertions, transitive browser admission. Not yet done for any pair.

### (2) Platform/primary-source facts

- Bun **1.4.2** pinned everywhere: `distribution/target-linux-amd64.json:3` (`bun-1.4.2-linux-amd64-v1`), fix-evidence qualification ran darwin/arm64 Bun 1.4.2, Chromium **140.0.7339.186**, WebKit **26.0**, PostgreSQL **17.11** (`post-upgrade-fix-evidence-2026-09-26.md:17-19`).
- Browser tests pin Playwright **1.55.1** (`tests/integration/browser/bun.lock:8`); gate5 matrix requires Chromium + WebKit (`tests/integration/gate5_frontend_test.go:7-8`).
- Browser fetch is same-origin only: `runtime/platform/action-json.ts:560-563` fails closed unless URL starts with `/` (not `//`), matching served CSP `connect-src 'self'`.
- Linux target exists and is narrow: Debian 13+ amd64/glibc, x86-64 ELF sidecar verify (`distribution/manifest.go:196-213`, `distribution/os_linux.go:25`, `distribution/verify.go:112`); stale "no Linux" README text is a docs defect, not a platform absence. Not rerun in review; installed-target runs deferred to x86 (UP25).

### (3) Probes

None run. R01's open questions are design/recipe gaps (adapter contract shape, companion recipe), not disputed runtime behavior. Existing evidence (audit tests `TestBrowserBuildTarget`, gate5 closure legs, `browser-owner-rerun.log` 2/0 outside sandbox) already establishes the closure mechanism.

### (4) Facts vs uncertainties vs counterexamples

- **Fact:** closed catalogue + transitive closure + structural audit are implemented and tested.
- **Fact:** no supported authored-JS/npm boundary exists in either profile.
- **Fact:** server escape hatches (bounded processes, literal-endpoint connections, plain HTTP) exist and are the correct non-exaggeration of the limitation.
- **Uncertainty:** which extension strategy (reviewed adapter vs companion vs catalogue-only) best serves the actual next browser requirement — deliberately unresolved; needs the discriminating experiment (one real third-party widget built both ways in isolated prototypes).
- **Counterexample to "can't integrate with anything":** `process::run` + HTTP boundaries + companion frontend already cover CLI/SDK-behind-service and separate-frontend shapes.

### (5) Guarantees to preserve

Controlled distribution catalogue + capability gate; audited content-addressed same-origin browser asset under CSP; no secret bytes in browser closure (gate5 scans served bytes); transitive denial including generic specializations; assertion-root exclusion from shipped asset.

### (6) Technical decision vs user syntax choice

- **Technical decision:** adapter-vs-catalogue-vs-companion boundary for in-process browser capabilities; what a "reviewed adapter" audit contract contains (target availability, I/O types + immutability conversion, failure mapping, callback/reentrancy, ownership/disposal, reproducible packaging). Needs the two-concrete-ops comparison (P11 bar) — all technical.
- **No user syntax choice** in R01 unless an adapter declaration syntax is proposed; that follows the technical decision.

---

## R02 — Browser controls / reusable UI (F-R02-01..07)

### (1) Current idiom + limitation

**Event snapshot (F-R02-01).** `runtime/platform/browser.ts:320-326` snapshots exactly four fields:

```ts
record(contracts.event, [["kind", event.type],
  ["target", <target.id>], ["value", <target.value>], ["key", <event.key>]])
```

Catalogue record `can.std.browser@1::event` has exactly `kind/target/value/key` (`catalogue.json:1824`). Missing: checkbox `checked`, multiselect values, file handles, modifier keys, composition/IME state, selection/caret. Nine admitted events: click, dblclick, input, change, keydown, keyup, focus, blur, submit (`browser.ts:136-146`, `compiler/internal/check/browser.go:85-88`). No pointer/touch/drag/composition, History, storage, layout/observer, canvas/SVG, or browser WebSocket (`browser.go:71` ws is server-only). ~22 browser ops in catalogue (`check/browser.go:21-42` lists 22 identities; `catalogue.json` has 33 browser-identity entries including types/errors).

**set_attribute vs live property (F-R02-02).** `browser.ts:424-435` delegates to native `setAttribute` after name/URL admission. There is **no** live `value`/`checked` property setter and no selection/caret accessor. Retained Chromium probe (`platform-probes/dom-semantics.mjs` + `.out.json`, Playwright Chromium per repo pin) establishes:

- After editing input to `user edit`, `setAttribute('value','normalized')` → attribute `normalized`, live `input.value` stays `user edit`.
- Checked and unchecked checkbox `change` events produce the identical Can snapshot `{kind:'change',target:'check',value:'on',key:''}`.

This is a native-semantic probe matching the adapter's exact calls — not end-to-end emitted Can, and not an attribute-setter bug (`set_attribute` keeps its promised attribute semantics). Workarounds exist but are narrow: app-state toggling approximates a controlled checkbox; dirty-input rebuild loses focus/selection.

**Current Can idiom that works:** controlled inputs keyed by id (`"qty:"+row`), per-field `input` listeners folding `e.value` into immutable draft state via versioned CAS cells (`read_state`/`replace_state`, `browser.ts:637-678`), in-place `set_text` status/totals refresh so caret never jumps (`web.can:779-828`, `on_id_input`/`on_qty_input`/`on_price_input`). Checkbox/reset/autofill/multiselect/upload/IME beyond this need native additions first — a library cannot derive what the boundary never exposes.

**View append/ownership (F-R02-04).** `browser.ts:446-458`: cross-view append is denied (`browser::rejected("scope")`) **except** into the app-root anchor, which belongs to a dedicated inert scope and outlives all views (`:334-353`). View disposal aborts listeners (AbortController), clears timers, removes non-root nodes, disposes cells (`:286-300`). Listener/timer callbacks must be named Can references; failures report once through the sealed reporter and end that dispatch only (S02, `browser.ts:307-319`, `runtime/browser/diagnostics.ts`). Cancel-policy listeners (`on_cancel_key` for keydown/keyup + exact key, `on_cancel_event` for submit; U05) run sync `preventDefault` inside native dispatch before the single immutable snapshot (`browser.ts:497-558`); disposed views can neither cancel nor dispatch.

**Grid manual-callback/focus-repair state (F-R02-03/06/07).** `examples/invoice-grid/src/web/web.can` (~1186 lines) + records model (~1525 lines):

- `build_rows` (`web.can:510-...`) manually constructs cells/attributes/listeners per row; row key carried in control ids + `near` captures.
- Render is dispose-then-rebuild via view swap (`render_replace`, `web.can:739-760`): dispose previous render view first (no duplicate ids coexist, dead listeners abort), state cell lives in a long-lived view so attempt-owned pending saves survive render disposal.
- Attach-then-focus order (S03): `build_view` appends the complete tree, then focuses row intent or chrome control (`web.can:719-736`); `focus_chrome` (`:484`) targets attached controls only. Blocked-save notice survives rerender + is announced; pending save visible until settlement; late replies/disposal can't clobber newer edits (fix evidence S03, grid-matrix legs in `tests/integration/browser/grid.mjs`).
- Known UX limitation (not a defect): row amount cells stay stale while aggregate totals update; test waits for rerender (`grid.mjs:234`).
- Preserved strengths (F-R02-07): immutable drafts, versioned state, stale-result/uncertainty/denial/conflict modeling (`records.can:25`, `:64`), disposal safety.

Un-extracted: keyed editable table + field/error/status component as ordinary shared libraries consumable by two apps (F-R02-05 acceptance). P08 (richer fields) and P12 (reuse/components) deferred: reopening needs an all-Can workflow + bounded immutable projection + per-field browser tests, and a typed field/keyed-editor prototype measured on one-field edits, focus, reorder, late replies, disposal. P10 (history/WebSocket) deferred with separate workflow conditions.

### (2) Platform/primary-source facts

- DOM semantics (living standard, as probed): `setAttribute('value',…)` sets the *default/content* attribute; the live `IDL` `input.value` reflects user edits independently once dirty. `HTMLInputElement.value` for checkbox defaults to `"on"` regardless of `checked` — hence identical snapshots. Any fix must read `checked`/live-value IDL properties and, for normalization-with-caret, set live properties (and eventually `setRangeText`/selection APIs), not attributes.
- Pinned qualification browsers: Chromium 140.0.7339.186, WebKit 26.0 via Playwright 1.55.1; gate5 conformance suite (`tests/integration/browser/conformance.mjs`, incl. U05 cancel + S02 report + disposal legs) runs in both named browsers.
- `setTimeout` range enforced both sides: 0–2147483647 ms (`check/browser.go:47`, `browser.ts:35`, delay denial instead of native overflow clamp).
- Query bootstrap budgets: whole `location.search` ≤ 8192 UTF-8 bytes, each value ≤ 256, keys `[a-z][a-z0-9_]*` ≤ 64 (`browser.ts:36-40`, `check/browser.go:172-191`); duplicates/malformed → `browser::invalid_query`.

### (3) Probes

None rerun. The retained `dom-semantics` probe outputs are undisputed, its source pins the exact native calls to `browser.ts:320-326,433`, and rerunning under the same pinned Playwright/Chromium would only re-confirm identical DOM semantics. No claim in F-R02-01/02 is disputed, so per the one-probe-only-if-disputed rule, no rerun was needed.

### (4) Facts vs uncertainties vs counterexamples

- **Fact:** 4-field snapshot; attribute-only writes; 9 events; root-only cross-view append; dispose-then-rebuild grid render.
- **Fact:** retained probe proves checkbox-state blindness and dirty-input attribute/live divergence.
- **Uncertainty:** the minimal sufficient native addition set (live value/checked setter + checked/multiselect/file/modifier/composition snapshot fields + selection/caret ops?) — each needs the P08 all-Can-workflow + bounded-projection + browser-test treatment.
- **Uncertainty:** whether nested-widget ownership needs a primitive change or composes from the existing lifetime model — needs the two-app extraction experiment.
- **Counterexample to "needs React/vDOM":** runtime uses native EventTarget/AbortController + versioned CAS cells; reuse via Can functions/records/callables is unproven but not ruled out — prove library composition before adding syntax.

### (5) Guarantees to preserve

Immutable event snapshots (sync capture, single dispatch); view-scoped listener/timer lifetimes; named-callback-only listeners; disposal semantics (abort/clear/detach, no post-disposal mutation); one sanitized located diagnostic per failed handler; sync registration-time cancellation policy; grid strengths (drafts, versions, conflicts, uncertainty honesty, disposal safety).

### (6) Technical decision vs user syntax choice

- **Technical:** native snapshot/property addition set + bounded immutable projections; nested-ownership primitive vs library pattern; extraction design for shared table/field components.
- **Library work:** the actual field/table extraction + second-app consumption (F-R02-05).
- **No user syntax choice** — declarative component syntax explicitly deferred until demonstrated library limitations (P12); do not assume it.

---

## R03 — Captured HTML routes (F-R03-01..02)

### (1) Current idiom + limitation

**Two-sided gap (F-R03-01):**

- Plain routes reject captures: `compiler/internal/check/http.go:369-373` — any segment starting with `:` fails `checkRoutePath` ("invalid static route path"). Path must also be a static string literal + named callback (`:333-353`).
- GET actions require JSON: `compiler/internal/check/actions.go:318-324` — mode `none` (bodyless GET) mandates `body json`. JSON POST → JSON; form POST → HTML (`:325-333`). HTML cases additionally require `swap inner` per case (`:367-375`), and swap is rejected for non-HTML actions — the HTML response machinery is currently HTMX-fragment-shaped (S01: exact `hx-status` exceptions, `action::missing_target`/`action::protocol` guard policy).

**Current workaround:** captured HTML reads use query URLs — e.g. `/invoices/form?tenant_id=…&invoice_id=…` (`examples/invoice/src/web/web.can:407-441`: `http::query_one(req,"tenant_id")`, `text::to_int` parsing, 400s on invalid). Works, typed at the application level, not an authorization defect — but a surprising hole: the typed capture/URL-building/ambiguity machinery exists for JSON actions and can't serve server-rendered reads.

**Exact gap (F-R03-02):** no checked route form combines `:capture` segments + typed capture extraction + HTML document response + truthful denied/not-found/error statuses. Reuse target: the shared action capture/route machinery (`runtime/platform/action-routes.ts`, `action_bindings.go` client/server sites) rather than a second URL grammar; independent of any SPA router (P10 deferred separately).

### (2) Platform/primary-source facts

- Runtime route mount re-validates; URL dot-segment normalization is deliberately runtime-owned (`http.go:355-358` comment).
- `GET` construction takes no body by grammar (`actions.go:317`); action statuses must be 200–599 excluding 204/205/304 since every case renders a body (`actions.go:361-366`).
- Served HTML fragments policy is compiler-owned (`runtime/platform/html.ts`, `htmx-guard.ts`; S01 fix evidence) — captured reads would extend, not bypass, this.

### (3) Probes

None needed: both restrictions are checker source facts with exact lines, and the query-URL workaround is visible in the invoice example. No runtime behavior is in dispute.

### (4) Facts vs uncertainties vs counterexamples

- **Fact:** `:capture` rejected in plain routes; `body json` forced on GET actions; query-URL workaround in use.
- **Uncertainty (bounded):** the precise checked shape — captured GET + `body html`? capture-aware plain routes? — and how `swap inner` / HTMX-fragment policy relates to full-document reads (document rendering vs fragment swap needs a design answer).
- **Counterexample to "routing is untyped":** JSON actions already have typed captures, canonical URL building, ambiguity rejection (`runtime/test/action-mount.test.ts:389,609,770`) — the machinery to reuse exists.

### (5) Guarantees to preserve

Static path literals; typed capture/URL-building/ambiguity checks; exact status/leaf agreement; body-limit budgets; truthful denied/not-found/error statuses; HTML guard policy (missing-target/OOB/control-header rejection).

### (6) Technical decision vs user syntax choice

- **Technical decision (bounded):** which checked form carries captured HTML reads and how document rendering composes with the fragment-swap policy. Likely small grammar/checker delta — syntax follows the technical shape, not a taste vote.

---

## R09 — Editor workflows (F-R09-01..03)

### (1) Current idiom + limitation

**LSP today (F-R09-01):** `compiler/lsp.go:156` advertises exactly `textDocumentSync: 1, definitionProvider: true`. Handled: `didOpen/didChange/didClose` (full-text sync), `publishDiagnostics`, `definition`, `shutdown/exit`. Everything else → `-32601 unknown method` (`lsp.go:214-218`). **No** completion, hover, references, rename, or formatting.

- Diagnostics: every keystroke runs `driver.CheckSnapshot` over the in-memory overlay through parse/resolve/check (`lsp.go:261-304`); per-file versions; related locations; severity fixed 1; `Fixes` candidates exist on the `Diagnostic` struct (`diagnostics.go:34-36`) but are not surfaced over LSP.
- Definition: `driver.Definition` (`diagnostics.go:326+`) resolves through file/package/prelude/import scopes — nominal types, calls, constructors, module values, generated declarations, record fields behind annotation-known receivers. **Declines** body-local names ("rather than guess") and catalogue/prelude symbols without source files ("unjumpable"). UTF-16 positions throughout.
- Snapshot is inert by contract: never builds/runs/asserts/dials/reads env/emits/mutates (`diagnostics.go:57-66`); carries `Graph`, `World` (when resolution succeeded), `Diagnostics`, `Versions`.

**Formatter machinery exists (delivery, not invention):**

- `syntax.FormatTrivia` (`compiler/internal/syntax/format_trivia.go:16`) — canonical render with trivia preserved.
- `canlc format [--write] FILE.can` (`compiler/current_format.go:21-63`) — stdout mode is pure syntax; `--write` additionally requires the original to check clean, the formatted output to check clean via overlay, file-unchanged-since-read, fixpoint stability, and atomic replace. "Anything invalid, stale or inequivalent leaves the file untouched."
- Physical-line grammar retained; P16 (multiline delimiters) deferred.

**What each missing capability needs:**

- **Format (LSP):** thin — wire `textDocument/formatting` to `formatSource` + overlay validation already in `current_format.go`. Main questions: whole-document only (matches trivia renderer) vs range formatting (unsupported by renderer); async validation latency per keystroke-save.
- **Hover:** needs a type-at-offset query over `World` + checked types (the checker has declared/inferred contracts; no LSP-shaped accessor exists). Definition's `findReference` + scope walk is the reusable front half.
- **Completion:** needs scope-aware candidate enumeration (file/package/prelude/import + locals + members behind known receivers) — largest new surface; must respect hard keywords, `near` obligations, and checked arities to be useful rather than noisy.
- **References:** needs whole-project reference index (resolve all files, invert `findReference`); builds on the same scope machinery as definition but must also decide whether body-locals (currently declined by definition) are included.
- **Rename:** needs references + safe-edit validation (overlay re-check per edited file, reusing `--write`'s validate-before-replace discipline). **Interacts with `near` (F-R09-03 / F-R08-03):** `near` captures bind by callee parameter name in caller scope (`compiler/internal/check/callables.go:127-135` — captures resolve `declaration.Names[i]` in the *caller's* lexical scope); renaming a library `near` parameter changes caller obligations without changing the callable type. Safe rename must either understand this coupling or wait for the R08 explicit-binding decision. P13 (explicit `near` binding) is deferred pending a rename/shadow repair comparison.
- **Acceptance (F-R09-02):** extract/rename a callback, change a shared record, inspect its inferred/declared contract, repair callers — all with normal editor tools while keeping semantic guarantees.

### (2) Platform/primary-source facts

- LSP transport: stdio with `Content-Length` framing (`lsp.go:60-68`); `--stdio` flag accepted and ignored; `--baseline` retired with an explicit error (`lsp.go:96-112`).
- Coordinate system: zero-based lines, UTF-16 columns (`lineWidth` counts astral as 2, `diagnostics.go:303-316`).
- `discoverRoot` walks up to `can.project.json` (`lsp.go:331-351`); files outside a project diagnose "missing project" rather than silently.
- Overlay: `project.Overlay` versioned per-path (`lsp.go:222-246`); `CheckSnapshot` runs without requiring an entry point so library files diagnose like programs (`diagnostics.go:75-77`).

### (3) Probes

None needed: capability advertisement is a one-line source fact, and each helper's existence/scope is established by reading `lsp.go`, `diagnostics.go`, `current_format.go`, and `callables.go`. No behavior disputed.

### (4) Facts vs uncertainties vs counterexamples

- **Fact:** diagnostics + definition only; formatter machinery complete behind CLI; `near` name-coupling confirmed at `callables.go:127-135`.
- **Uncertainty:** hover/completion/reference query design (new compiler queries vs LSP-side assembly from `World`); range-format feasibility; rename's dependence on the R08 `near` outcome.
- **Counterexample to "no editor story":** per-keystroke overlay diagnosis + cross-package definition + validated atomic CLI formatting is a real, if minimal, editor workflow — the gap is breadth, not absence.

### (5) Guarantees to preserve

Inert snapshots (no build/run/network/env/emission); diagnostics identical to CLI pipeline; definition declines rather than guesses (extend, don't weaken); formatting never writes invalid/stale/inequivalent output; `near` capture semantics unchanged unless R08 decides.

### (6) Technical decision vs user syntax choice

- **Technical/tooling decision:** LSP feature scope and order (format-first is cheapest; completion needs a precision/recall design); reference-index architecture; rename validation discipline.
- **User syntax choice (via R08):** only if safe rename forces explicit `near` bindings (P13/C3) — that grammar change needs an explicit user decision; the LSP work itself does not.

---

## Cross-topic notes

- **R02 → R01 dependency:** the smallest discriminating experiment (platform-reconciliation §"Host-extension recommendation") orders the work: add only the generic DOM contracts the checkbox/dirty-input evidence demands → extract a two-instance form widget as an ordinary Can library → then pick one real third-party widget and prototype reviewed-adapter vs companion. R01's decision quality depends on R02's library proof.
- **R09 → R08 dependency:** safe rename waits on the `near`-binding decision; formatting/hover/references do not.
- **R03 is independent** of the R02 native-control work and the R10/P10 router questions — bounded checker/emitter slice.
- **No fanless-machine concerns triggered:** zero probe executions; all findings from retained evidence + source inspection at the reviewed revision.