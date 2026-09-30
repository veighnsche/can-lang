# HTML authoring on a real Manolea page

30 September 2026. Investigation and disposable development evidence; no production syntax or application changes.

**Use reusable typed components for current work. Keep a checked tree as a candidate composition layer, not an adopted replacement.** The tree makes parent/child relationships more explicit and can reject some literal mistakes earlier. This experiment does not establish that it generates more reliably or makes all edits safer. Components and trees solve different parts of the authoring problem and can share the same trusted HTML backend.

## Page and comparison

The specimen is `manolea-2/src/services/services.can::editor` (source lines 249–282), with its actual `publish_control`/`state_form` and control builders from `src/jobs/jobs.can`. It has hidden ID/CSRF inputs, slug/title/deliverable/description fields, labels and length limits, a save POST to `/service/edit`, and a separate publish POST to `/service/state`. Blank drafts render an empty paragraph in the publication position; saved drafts render the second form. This state-dependent subtree is a useful real edit hazard.

The source hashes still matched the earlier captured originals. The original page already reuses control functions: this is **better component contracts versus explicit layout composition**, not components versus entirely raw HTML. Both corrected candidates prepare the same seven control nodes, publication, viewport and typed attributes. The tree exposes the seven controls individually rather than hiding the whole form behind one splice.

The component candidate adds `input_slot`, named slot constructors and fixed heading/paragraph helpers. Its layout ends with ordinary Can composition:

```can
call heading("Service draft") as html::node heading
call paragraph(notice) as html::node notice_node
call html::make_tag("form") as html::tag form_tag
call html::element(form_tag, [method_attr, action_attr], [id_field, csrf_field, slug_field, title_field, deliverable_field, description_field, submit]) as html::node form
call html::document("Service draft", [viewport], [heading, notice_node, form, publication]) as html::safe document
```

The competing layout is **experimental notation, not supported Can source**:

```text
tree document "Service draft"
    head
        node viewport
    body
        element h1 []
            text "Service draft"
        element p []
            text notice
        element form [method_attr, action_attr]
            node id_field
            node csrf_field
            node slug_field
            node title_field
            node deliverable_field
            node description_field
            node submit
        node publication
```

The bounded Python lowerer admits literal tags, pure names/field references and a restricted string-literal subset. It emits current Can constructors; it is neither a new HTML renderer nor a general Can expression parser. Calls, branching and preparation remain ordinary Can. Conditional publication is exercised; arbitrary loops/dynamic-list edits are not.

## What the evidence says

| Criterion | Reusable typed components | Checked tree |
| --- | --- | --- |
| Reliable AI generation | Existing Can syntax and function contracts; named slots centralize label/name/id/limit metadata. Slots still contain strings, so wrong same-type bindings remain possible. | Restricted vocabulary makes layout generation explicit, but adds grammar and tooling the agent must learn. One matched edit succeeded; no general reliability advantage established. |
| Hierarchy | Parentage is expressed in child arrays and named intermediate nodes; review may require following several bindings. | Parentage and sibling order are visible in one block. This is the clearest demonstrated advantage. Opaque splices still hide internal subtrees. |
| Diagnostics | Compiler rejects wrong argument types. Observed CLI errors name the file and `editor` region, without the exact expression line. Runtime assertion failures reported `outcome mismatch`. | Literal syntax/structure errors have tree line numbers and sometimes a related parent line. Splice/type errors currently point into generated Can, also at function scope. The saved source map is partial and not integrated into diagnostics. |
| Safe edits | Reuse protects fixed metadata from repetition. Reordering a child array was straightforward. Same-type value substitutions can silently pass assertions. | Moving an indented subtree makes the intended structural edit easy to inspect. A valid indentation change can still place an opaque form inside another form; static tree checking does not see through that splice. |
| Trust and cost | Uses current opaque HTML types, URL/attribute admission and escaping. Helper assertions must themselves be valid Can. | Uses exactly those same trust boundaries. Production adoption would also need parser/checker, formatting/editor support and original-source diagnostic mapping; it provides no extra escaping guarantee. |

One independent agent per notation, using the same model and reasoning level, received the same semantic edit: group heading and notice in a section, then place publication immediately before the edit form as its sibling. Both first edit outputs passed the exact expected rendered-output checks after common fixture corrections. This is **one matched trial per approach**, not a statistical generation benchmark, token comparison, repair benchmark or creation-from-scratch test. Agents had different necessary surface references; there was no crossover or repeated sampling. No preference is inferred from line count.

## Executed checks and counterexamples

A development sidecar used the existing pinned Bun 1.4.2 archive and Can source commit `6d5435baf453bec4f5c85b89cbdff1ebaaa5e45c`. The final combined build passed **131/131 assertion roots**. Baseline, components, lowered tree and the two edited candidates produced **15 rendered cases**: blank, saved and hostile input for each. Original candidates were byte-identical to baseline; edited candidates matched each other and the requested wrapping/reordering oracle.

An independent HTML parser check also verified actual field values, label relationships, limits, CSRF, actions, methods, separate forms, edited hierarchy and hostile values remaining data. Its synthetic wrong-title output control was rejected. These are serialized-output checks, not browser parsing, visual rendering, submission or server authorization tests.

| Deliberate change | Observed result |
| --- | --- |
| Pass a node/array where text is required | Both candidate styles rejected during Can checking; function-level diagnostic. |
| Spread a single node with tree `nodes` | Can checking rejected it with `array spread requires array`; no integrated tree location. |
| Literal form inside form | Tree rejected at line 10 and identified parent line 9. |
| Child under a void element, repeated identical attribute expression, unknown tag | Tree rejected at the authored row. |
| Move publication inside the save form | Both candidates reached runtime assertion execution: blank passed, saved/hostile failed. A direct compiled-publication/runtime check confirmed `html::invalid_structure` for the saved case. |
| Feed `fields.deliverable` into the title component | All attached assertions passed. Nominal slot typing and success-only HTML assertions do not prove the intended value binding. |

The lowerer does not statically reject every structure that the runtime rejects. Independent parse probes accepted literal nested anchors, `p` under `ul`, and different symbols that might resolve to duplicate attribute names. It must not be advertised as a complete HTML content-model checker. HTML itself excludes descendant forms from a form and gives `p` a phrasing-content model; valid indentation alone is insufficient. See the [HTML form definition](https://html.spec.whatwg.org/multipage/forms.html#the-form-element) and [paragraph definition](https://html.spec.whatwg.org/multipage/grouping-content.html#the-p-element).

## A consequential prototype correction

The inherited static receipt did not establish build readiness. Full execution rejected two shared prototype patterns:

1. `prepare` returned a record containing opaque HTML values but used bare `=> ok` expectations. Bare success-only expectations are accepted for direct opaque/callable results, not this record result. This does not prove records containing opaque values are unusable; it proves these assertions were invalid.
2. `post_form` directly called domain-fallible attribute constructors inside its assertion arguments, which required explicit completion handling.

Both extra helper layers were removed from the disposable candidates and their operations placed into normal typed bindings/form construction. The leaf components remained reusable. The original untracked experiments were preserved. This favors less scaffolding for this page and shows why declaration inspection, formatting and syntax sketches must not be treated as executable evidence.

Preparation order matches between the two corrected alternatives, but differs from the original page: publication is prepared before outer form assembly. The three successful rendered states are equivalent; global failure timing for arbitrary future constructor changes is not established.

Six unsuccessful development attempts are retained separately: two exposed the inherited assertion defects; four required probe corrections (package layout, binding rewrite, generated-module lookup and the runtime's FD 3 environment input). Three page builds passed during observer development, including the final run. Failed attempts are not qualifying successes. All temporary sidecars/projects were reclaimed; only compact source and result evidence remains.

## Recommendation and advisory review

Use small typed components to own semantic metadata and fixed structure. Do not infer stronger guarantees from a record whose members remain ordinary strings, or from a helper name such as `post_form` when its signature accepts arbitrary attributes. Keep page layout explicit and test rendered field bindings and state-dependent branches.

Retain the tree as a **potential layout layer over the same components**. Before adopting it, the useful next evidence is mapped diagnostics at the original bad splice/attribute and additional matched generation/edit/repair tasks that check semantics. That is an evidence gap, not a requirement to build a general UI framework or a large benchmark now. Keep runtime checks for opaque/dynamic content and preserve explicit fallible preparation; do not add raw HTML or claim full static HTML validity.

Three fresh Jev SystemOne consultations selected `components_now_tree_candidate`. Each received the same observations and alternatives with fully rewritten context, question and option prose; an independent wording/equivalence review removed uneven caveats before sending. All three returned probability 1.0 for that option, with confidence 0.99/1.0/1.0. There was no answer disagreement to investigate. Agreement remains advice, not proof of correctness or removal of framing bias. Total reported provider usage was 2,498 input and 159 output tokens. The recommendation rests on the executable evidence and tooling gaps above.

Evidence and replay sources: [comparison packet](preparation/html-authoring-followup-2026-09-30/), [execution results](preparation/html-authoring-followup-2026-09-30/probe-results.json), [output checks](preparation/html-authoring-followup-2026-09-30/output-checks.json), [Jev requests/responses and wording audit](preparation/html-authoring-followup-2026-09-30/jev/). No production compiler/runtime/application files, dependencies, services or deployments changed. Multi-page reuse, full HTML conformance, browser behavior and general AI generation/repair reliability remain unmeasured.
