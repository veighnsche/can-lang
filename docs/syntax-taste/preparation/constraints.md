# Current constraints for Can design preparation

24 September 2026 · preparation evidence, not a revised language decision

This is the P2.2 constraint register for the [preparation checklist](../can-design-preparation-2026-09-24.md).
It records present authority and possible conflicts for later design work.
The authoritative record is [decisions.md](../decisions.md) with its incorporated
specifications and [September 22 dispositions](../../implementation/language-design-dispositions-2026-09-22.md).
The [recommendation program](../can-recommendation-program-2026-09-24.md)
proposes new acceptance gates; it does not itself amend those decisions.

## Confirmed objective and repository instructions

| Constraint | Status and consequence | Source |
| --- | --- | --- |
| Can is made for AI coding agents; human readability, familiarity and comfort are not design goals | User confirmed. Evaluate syntax and tooling by benefit to agents. Human difficulty is acceptable when it serves agents. | [Design direction](../decisions.md#design-direction) |
| Token efficiency is a secondary measured objective | The user selected total tokens per successful agent task, including context, code, diagnostics and retries. Specify a comparable protocol before using the metric to decide trade-offs. | [Design direction](../decisions.md#design-direction) |
| Do not ask the user further design or syntax questions | The user explicitly directed the preparation team to stop asking and consult Jev instead. Preserve the two syntax answers already given; resolve remaining choices by evidence, three fresh Jev consultations for difficult decisions, and recorded engineering judgment. | User instruction, 24 September 2026; [confirmed answers](confirmed-syntax-choices.md) |
| There are no external users or compatibility obligation | Changed syntax, ABI, generated layouts or goldens need not preserve old behavior for compatibility. Accepted semantics and deliberate tests still matter. | User's `AGENTS.md` instructions supplied for this repository; [recommendation scope](../can-recommendation-program-2026-09-24.md) |
| Equivalent native JavaScript/Bun operations are preferred | Generated TypeScript should call native operations, adding only adapters needed for Can's contracts and immutability. | User's `AGENTS.md` instructions; [Can-to-Bun boundary](../decisions.md#can-to-bun-boundary) |
| Difficult technical design choices consult Jev three fresh times | Supply the evidence; rewrite all explanatory prose in each request, check equivalence/variation, save pairs and investigate disagreement. Jev is advisory and does not research. | User's `AGENTS.md` instructions; [decision maintenance](../decisions.md#maintaining-this-reference) |
| Authored runtime TypeScript has required lint/format/check/test workflow | Applies to future implementation tasks under `runtime/` or `tools/runtime/`; it does not require running runtime checks for this document-only preparation. | User's `AGENTS.md` instructions |

## Currently selected design and platform limits

These are constraints on describing **current Can**, not immutable vetoes on a
properly recorded future design change.

| Area | Current rule or qualified target | Source |
| --- | --- | --- |
| Language core | Primarily functional direction; top-level named functions; nominal immutable values, closed variants and explicit public error bounds. Effects are deferred. | [Design direction](../decisions.md#design-direction), [current decisions](../decisions.md) |
| Native AI | Dedicated Noul, Choice, Score, judge, fetch and LLM forms; grouped state and explicit connection identity are preserved. | [Design direction](../decisions.md#design-direction), [AI dispositions](../../implementation/language-design-dispositions-2026-09-22.md#native-ai-state-and-connections) |
| Platform admission | Can source cannot embed JavaScript/TypeScript/Bun code or import arbitrary backend APIs. Approved operations are distribution owned and lower through the compiler to native operations. | [Can-to-Bun boundary](../decisions.md#can-to-bun-boundary) |
| Initial web frontend | Bun executes Can and emits HTML/HTMX. The current decision excludes a Can browser target from the initial scope. | [Initial web frontend](../decisions.md#initial-web-frontend-server-rendered-html-with-htmx) |
| Resource and race semantics | Runtime ownership/leases are retained as the safety boundary; native race winner behavior and empty-race pending behavior are retained. | [Resource/race dispositions](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts) |
| Qualified runtime | The pinned distribution is `bun-1.4.2-darwin-arm64-v1`; a Linux SaaS claim needs its own target and evidence. | [Distribution target](../../../distribution/target.json), [Gate 4](../can-recommendation-program-2026-09-24.md#gate-4--qualify-the-service-that-runs-the-flows) |
| Design authority | Existing choices remain authoritative until explicitly revised; prototype success is evidence, not a silent specification change. | [Authority and maintenance](../decisions.md), [recommendation status](../can-recommendation-program-2026-09-24.md) |

## Proposed changes that cross current dispositions

| Proposed outcome or open choice | Current disposition it may revise | Required preparation before promotion |
| --- | --- | --- |
| Unknown intended pattern leaf must diagnose instead of becoming a binding | LD37 retains broad bare-name behavior | Counterexamples, alternatives and Jev review are complete; the user selected `bind name` before directing no further questions. Integrate its full rule without reopening the answer. |
| Explicit cross-package scenario ownership | LD28 defers root-owned symbolic seams | Establish the refactoring failure and compare lexical templates/whole-helper stubs without losing checked arguments or queue isolation. |
| Independent package names and error identities | LD36 defers error-ID redesign; package-name collision is separate | Audit canonical identity and diagnostics; exercise unrelated dependencies without editing them. |
| Authored finite error-set parameters | LD14 defers them | Show concrete current-idiom failure or material cost before trial; no inferred public errors or general effect system follows automatically. |
| Explicit capture binding | LD39 defers it | Compare explicit context records and concrete rename/shadow case before any syntax promotion. |
| Rich Can-authored frontend | U7 selected server-rendered HTML/HTMX for initial scope | Establish scope and browser-safe capability/wire/lifecycle contracts; separately qualify a browser target before broad frontend recommendation. |
| Linux-hosted SaaS | Current distribution is Darwin ARM64 | Name, package and qualify a Linux target, including relevant native API behavior and service flows. |
| Package-controlled validated immutable values | Present authored records are transparent to importers | Specify construction, equality, matching, copy-update, decoding, schema and fixtures together; keep authorization operation bound. |
| Route/form/UI and end-to-end behavior guarantees | Current safe URL, forms, HTTP and HTML do not connect all identities or prove visible HTMX behavior | Prove a realistic tenant invoice flow and signed webhook under actual database/protocol conditions. |

The [full inventory](decision-inventory.md) will own complete finding coverage and
additional earlier dispositions. This register supplies constraints to evidence
workers without deciding which proposals should be accepted.
