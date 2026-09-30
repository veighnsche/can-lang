# HTML authoring — additional evidence, 2026-09-30

Status: bounded investigation complete. No production language, runtime or Manolea source change.

This supplements the preserved [companion investigation](html-authoring-investigation-2026-09-30.md). Its separate packet records 131 passing attached assertion roots and 15 rendered observations, an independent HTML-parser check, and a same-type wrong-field mutation that those success-only assertions missed. This packet adds node-array splices, a second hostile dataset, matched component structural defects, precise structured diagnostic remapping and ownership/source-stability controls. Counts and qualification claims remain specific to each packet.

Reusable ordinary Can components are viable today. A checked tree gives literal page ancestry an explicit representation and can reject some structural mistakes before execution. This comparison supports a further hybrid feasibility experiment; it does **not** establish that tree notation is generally more reliable for AI agents or justify adopting grammar now.

The next experiment should make reusable node/fragment trees and contextual diagnostics work together with ordinary Can helpers. Keep the architecture choice open until those contracts and a broader, matched agent trial have evidence.

## One real page, shared contracts

The subject is Manolea's service draft editor in [services.can](/Users/vince/Projects/manolea-2/src/services/services.can:249), its [publication control](/Users/vince/Projects/manolea-2/src/services/services.can:168), and the shared [form controls](/Users/vince/Projects/manolea-2/src/jobs/jobs.can:128). Source and snippet hashes are in [source-manifest.json](preparation/html-authoring-2026-09-30/source-manifest.json).

The page has a heading, escaped notice, POST edit form, and separate publication control. The controls preserve id, CSRF, slug/title/deliverable/description fields, label associations, routes and length limits. An unsaved draft produces the original empty paragraph for publication; a saved draft produces a separate POST form.

Three fixtures express that page:

- [baseline.can](preparation/html-authoring-2026-09-30/baseline.can) copies the original helpers and layout, removes local package qualifiers and adds a saved case.
- [components.can](preparation/html-authoring-2026-09-30/components.can) adds immutable input-slot records, named slot factories and fixed heading/paragraph helpers. Ordinary Can assembles the form and document.
- [editor.tree](preparation/html-authoring-2026-09-30/editor.tree) expresses literal ancestry and explicit text/node/node-array splices. The [research lowerer](preparation/html-authoring-2026-09-30/lower_tree.py) translates it into existing Can constructors.

Both alternatives use the same ordinary Can preparation calls, local `prepared_editor parts` record and actual method/action attribute tokens. Dynamic behavior remains ordinary Can. No raw markup path or implicit text-to-node conversion is introduced.

The component layout is ordinary Can:

```can
match chain
    call heading("Service draft") as html::node heading
    call paragraph(notice) as html::node notice_node
    call html::make_tag("form") as html::tag form_tag
    call html::element(form_tag, [parts.method_attr, parts.action_attr], parts.controls) as html::node form
    call html::document("Service draft", [parts.viewport], [heading, notice_node, form, parts.publication]) as html::safe document
    html::invalid_structure
    ok => ok document
```

The experimental tree layout consumes the same prepared values:

```text
tree document "Service draft"
    head
        node parts.viewport
    body
        element h1 []
            text "Service draft"
        element p []
            text notice
        element form [parts.method_attr, parts.action_attr]
            nodes parts.controls
        node parts.publication
```

This notation is a research subset, not a proposed final spelling. It accepts one document root, four-space ancestry, pure names/fields and a shared string-literal subset. It does not yet represent reusable node/fragment roots, loops, calls or arbitrary expressions within a tree.

## Comparison for AI coding agents

| Criterion | Reusable typed components | Checked tree composition |
| --- | --- | --- |
| Reliable generation | Existing function/effect rules; named slot factories centralize wiring and fixed tags. Slot label/name/id members remain interchangeable strings, so semantic transpositions can still typecheck. | Explicit roles distinguish escaped text, one opaque node and a node-array splice. Literal ancestry is visible. The prototype restricts expressions to prepared values; general generation reliability remains unmeasured. |
| Hierarchy and local reasoning | Helper names communicate purpose; reusable internals live in normal functions. Document arrays and separately built values determine ancestry, and opaque return types hide descendant structure. | Parent/child and sibling structure are directly represented. Literal nested forms can be rejected with both parent and child locations. Opaque splices still hide their contents. |
| Diagnostics | Can already reports typed expression errors. Current HTML structure errors expose a short reason and unlocated `can:html` origin. Mandatory assertion fixtures for opaque values need careful design. | Prototype syntax checks name an authored row and sometimes a related parent. Two ASCII typed-splice errors map accurately through the existing structured checker. Dynamic structure errors retain the same coarse runtime origin. |
| Safe edits | Reuse/extraction requires a function declaration, exports where appropriate, assertions and exact effect forwarding. The demonstrated intro extraction and sibling reorder preserve expected output after shared fixture corrections. | Grouping and moving literal subtrees preserves visible ancestry when indentation is correct. Both demonstrated edits preserve expected output. Reusable tree extraction and malformed-edit recovery remain gates. |
| Validation boundary | Opaque constructors, escaping, URL policies and partial runtime content checks remain authoritative. | The same constructors and runtime contracts remain authoritative. Literal static checks supplement them; they cannot certify arbitrary splice contents. |

Repetition or fewer lines does not decide this comparison. The useful question is whether each representation makes generation obligations explicit and lets an agent identify and repair a mistaken edit without reconstructing hidden dependencies.

The directions are complementary: ordinary functions provide reuse and dynamic contracts; trees can describe literal shape. Admitting known literal tags and checking literal constructor arguments could also improve ordinary Can without tree grammar. Those diagnostic benefits are not exclusive to notation.

## Executed observations

[probe-results.json](preparation/html-authoring-2026-09-30/probe-results.json) records stable compiler/runtime source fingerprints, tool and dependency-manifest hashes, candidate hashes, diagnostics and rendered-output hashes.

One small Go emission instrument was built and reused serially. All 13 executable candidates passed full assertion-program checking and TypeScript emission. Direct execution of their editor functions produced 52 observations across blank, saved and two hostile-input datasets:

| Observation | Count |
| --- | ---: |
| Exact match to independent explicit markup | 28 |
| Expected HTML domain failure | 22 |
| Successful blank-publication mutation because it contains no form | 2 |

The 28 exact matches comprise the original page and both alternatives on four datasets, plus grouping/reorder variants for each alternative on those datasets. The Python oracle explicitly specifies markup, escaping, field attributes, publication branches and sibling order; it does not call the runtime serializer to construct expectations. It proves the exercised rendered output, not browser interaction or universal HTML validity.

The 22 failures include matched component tag/void-child/duplicate-attribute controls (12), nested publication forms in both surfaces on saved-id datasets (6), and a head-only viewport spliced into the body (4).

| Seeded defect | Observed rejection |
| --- | --- |
| Literal form tag typo | Tree check: `HTML-TREE-TAG`, row 9. Matched component: full checking succeeds, runtime reason `tag`. |
| Literal void element with child | Tree check: row 8, related parent row 7. Matched component: runtime `void_children`. |
| Same immutable attribute repeated | Tree check: row 9. Matched component: runtime `duplicate_attribute`. Distinct tokens with the same attribute name are not statically resolved by this prototype. |
| Literal form inside a form | Tree check: child row 10 and parent row 9. |
| Empty attribute slot / effectful splice | Tree syntax rejects the malformed list / requests preparation in ordinary Can. |
| String used as node / node-array splice | Full Can checking rejects both. The structured driver's zero-based UTF-16 spans map to tree rows 11/10, columns 14/19 for these ASCII expressions. |
| Opaque publication node moved inside edit form | Both layouts fail at runtime for saved ids with `nested_element`; blank ids succeed because publication is an empty paragraph. |
| Opaque viewport moved into body | Runtime `document_context`. |

Every observed runtime structure failure has `source: "can:html", start: 0, end: 0, invocation: []`. A source map alone cannot improve that origin. Contextual validation must connect the failing parent operation and implicated child/attribute to authored locations while preserving original error occurrence identity.

## Two assertion findings

The first preparation helper returned a record containing opaque HTML tokens and used bare `=> ok` assertions. Full checking rejects this: bare success is permitted for a direct opaque/callable result, not that record. The retained [record-helper snapshot](preparation/html-authoring-2026-09-30/record-helper-blocked.can) is a negative control.

The second form helper accepted prepared opaque attributes. Its assertion tried to construct those arguments through domain-fallible calls; full checking rejects those expressions without explicit completion handling. The [attribute-helper snapshot](preparation/html-authoring-2026-09-30/attribute-helper-blocked.can) records that fixture problem.

These findings do not make the runtime APIs inherently invalid. They show that a proposed reusable component API needs executable mandatory-fixture recipes and independent observations of opaque results. For this comparison, preparation became an outer chain with a local record in an `ok => do` block, and form assembly remained inline.

`inspect-types` establishes declaration/type-model facts, not full executable body/assertion validity. The final evidence uses full checking and emission. Attached assertions were checked/emitted but were **not executed**; output observations come from direct calls to emitted editors.

## Agent edit demonstration and limits

Two fresh Sol medium agents independently edited one surface each. Each grouped the introduction and moved publication before the edit form. Component grouping extracted a helper; tree grouping inserted a literal section. These have the same markup objective but are not identical extraction tasks.

Root corrected inherited component scaffolding by removing the invalid form-helper fixture and excess inner error arm. The requested grouping/helper and sibling-order changes were preserved. Initial blocked-template attempts are retained separately. [pilot-protocol.json](preparation/html-authoring-2026-09-30/pilot-protocol.json) records assignments, support corrections and provenance limits. Corrected pre-repair raw files were not independently captured; this is an edit demonstration, not proof of unassisted first-pass success.

One page, two agents and two edits do not establish general generation reliability, cost or wall-time superiority. There was no line-count, token or performance benchmark.

Preparation differs from the original call order: publication is built before layout assembly. The successful render cases preserve output; general first-failure ordering and occurrence-identity equivalence to the original are unproven. The existing runtime checks selected content rules, not a complete HTML model, label/ID agreement, CSRF correctness or accessibility.

## Jev consultation and review

Three fresh requests rewrote all explanatory context, questions and option descriptions while retaining the same facts, constraints and alternatives. Root and an independent agent audited the final requests before submission. Requests, raw responses and the [wording audit](preparation/html-authoring-2026-09-30/jev/wording-audit.json) are retained.

The classification API and question schema were checked against the live [TypeSafe API](https://docs.typesafe.ai/api) and [Choice guidance](https://docs.typesafe.ai/primitives/choice). Jev received the evidence; it did not research the repository.

Consultation findings and disagreement analysis are in [jev/findings.md](preparation/html-authoring-2026-09-30/jev/findings.md). Its advice informs follow-up scope; it does not qualify either architecture. These requests evaluate this additional packet and broader next-direction alternatives. The companion report’s consultations asked a different selection question; their distributions must not be pooled or treated as an architectural vote.

Independent source/evidence review prompted matched component controls, corrected provenance, an explicit pilot limitation, stable source/instrument fingerprints and cleanup recovery checks. Exact structured type-error mapping was rechecked after correcting the research instrument's zero-/one-based coordinate conversion.

## Remaining gates

1. **Reusable node/fragment trees:** put an extracted subtree in an ordinary typed Can helper, exercise required assertions, consume its node in another tree, and preserve dynamic data and effects. The current document-only instrument cannot answer this.
2. **Shared diagnostics and opaque fixtures:** specify valid component fixtures and independent rendered observations; add parent/child/attribute runtime context and precise compiler/LSP mapping, including Unicode and opaque splices.
3. **Mechanical edit behavior:** challenge malformed indentation, grouping/extraction, conditional and array splices, and repairs. Preserve AST meaning, output and relevant failure/occurrence contracts.
4. **Matched agent trial:** use corrected shared contracts and equivalent tasks; measure semantic correctness, unwanted changes, diagnostic localization and repair success. Separate extraction from literal grouping.

The first two gates should stabilize the comparison's mechanisms before broad authoring evaluation. This is a recommendation for a bounded follow-up, not a production implementation plan.

## Resource ownership and reproduction

The [probe script](preparation/html-authoring-2026-09-30/run_probe.py) builds one small instrument with the shared Go cache, generates fixture output only, and symlinks to existing runtime sources. Each command has a deadline; exceptions and handled termination kill its process group. Temporary-directory cleanup is registered immediately. Ownership markers protect live/foreign work and recover abandoned marked directories when no live owner/group remains.

[cleanup-controls.json](preparation/html-authoring-2026-09-30/cleanup-controls.json) records eight bounded controls: dead owned recovery, live owner, foreign owner, malformed groups, boolean PID, mismatched path, non-object marker and live child-group refusal. All control directories/processes were reclaimed. A hard interruption before a marker is written can leave an unmarked directory, which recovery deliberately does not delete as foreign/unknown work. Cleanup failures are surfaced.

From this repository, rerun:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 docs/syntax-taste/preparation/html-authoring-2026-09-30/run_probe.py
```

The signature-inspection binary path is recorded in the script and hashed in results; emission uses current repository compiler source. Bun receives an empty descriptor-3 environment snapshot. No browser, service, database, performance measurement, application/distribution build, dependency copy or persistent private build cache is required. Compiler/runtime/tool files must stay stable during the run or the evidence is rejected. All emitted workspaces were reclaimed; only compact fixtures, instruments and evidence remain.
