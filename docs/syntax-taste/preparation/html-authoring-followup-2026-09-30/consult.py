"""Three fresh, fully reworded advisory classifications; no prior answers."""
import datetime
import json
import os
from pathlib import Path
import sys
import urllib.request

ROOT = Path(__file__).resolve().parent
OUT = ROOT/'jev'
OUT.mkdir(exist_ok=True)
contexts = [
    "Can targets AI authors; prioritize reliable generation, explicit contracts, hierarchy, precise diagnostics and safe edits over brevity. Compare one real Manolea service-draft editor: named text controls with limits, CSRF, save POST and a separate publication form present only for saved records (blank records have an empty paragraph). The existing page already reuses control helpers. Candidate C adds typed field-slot records and fixed heading/paragraph helpers. Candidate T reuses the same typed preparation and places HTML layout in an indentation tree lowered to current safe constructors. This is an investigation, not authorization to ship a language change. Two initial shared prototype helpers did not build: bare ok cannot assert a record containing opaque values; an assertion argument cannot directly call domain-fallible attribute constructors. Removing those helper layers and inlining ordinary typed preparation allowed 131 assertions to pass. Both corrected candidates render exactly the baseline for blank, saved and hostile data. One independent same-model low-effort agent per notation grouped heading/notice in a section and moved publication before the edit form; both edits preserve exact expected output in all three states after the shared fixture repairs. This n=1 trial per notation is not a reliability estimate. Typed slots still contain strings and do not prevent a same-type wrong field binding. The tree detects literal nested forms, void children, unknown tags and repeated identical attribute expressions early. Opaque publication moved inside the save form passes static checking but fails at runtime for saved records; blank passes. Both share opaque HTML tokens, escaping and runtime validation. The tree is a small external prototype with incomplete checking and no integrated editor or compiler-to-tree diagnostics; its source map is partial. It cannot statically inspect an opaque node's subtree. There is no evidence on broad generation success rates, repair cost or multi-page reuse, and no production source changed.",
    "Evaluate HTML writing for an AI-oriented Can language using semantic accuracy, clearly stated contracts, visible nesting, actionable errors and safe transformations, not line savings. The chosen application screen is Manolea's service editor, whose bounded text fields, hidden CSRF, draft POST and conditional publication POST must survive unchanged; without an existing ID, the publication position contains an empty p. Reusable control functions exist in the original implementation. In alternative C, field metadata is collected into typed slots and h1/p construction becomes fixed helpers. Alternative T consumes identical typed preparation through an indented layout notation that emits the established trusted HTML calls. We are investigating and have not approved a production syntax addition. The initial common scaffolding had two invalid assertion patterns: a record wrapping opaque values was expected with bare ok, and fallible attribute creation appeared directly in assertion inputs. Replacing that scaffolding with ordinary local bindings made all 131 assertion roots pass. Three rendered states (empty record, persisted record and hostile text) are byte-equal to the original for C and T. Separate agents with the same model and low reasoning each attempted the same section-wrapping/publication-reordering edit. After the common scaffolding corrections, each first edit matches its expected output across the three states; one sample each cannot establish comparative AI reliability. Ordinary strings remain inside the slot record, so a type-correct value from the wrong field is still possible. Prototype T rejects known literal bad tags, nested forms, void children and an attribute expression repeated verbatim. It accepts a publication node spliced into the draft form; the runtime refuses that nesting only when the publication node actually contains a form, while the empty case succeeds. Trusted tokens, text encoding and runtime structure checks are common to both alternatives. T is an external partial checker/lowerer rather than implemented Can syntax; editor integration and complete source-location diagnostics are missing, and opaque child trees cannot be inspected statically. Broader AI completion rates, repair expenditure and reuse across pages have not been measured. Application and compiler production files remain unchanged.",
    "The decision concerns an investigation of one Manolea draft-service page in Can, which is intended for AI coding. Judge generation correctness, explicit typing, parent-child reasoning, error localization and edit preservation ahead of compactness. This screen has limited text inputs, a CSRF hidden input, a save POST, and a sibling publish POST for an existing ID; the no-ID branch substitutes an empty paragraph. Baseline code already calls reusable control builders. C introduces typed slot data for input metadata plus dedicated heading and paragraph functions. T keeps that same typed preparation but spells the layout as an indented tree compiled into the existing safe constructor API. No production language adoption is authorized by this comparison. During execution, the common prototype failed because bare ok is not a valid expectation for an opaque-containing record and because domain-fallible attribute calls cannot be used directly as assertion arguments. Inline preparation removed both extra helper layers, after which 131 roots passed. C and T each produce byte-identical baseline HTML for blank, saved and hostile samples. A matched pair of independent agents (identical model, low effort) performed one combined edit apiece: wrap title and notice in section, then put publication immediately ahead of the edit form. Following the shared fixture fixes, both original edit attempts satisfy the exact three-state output oracle. That tiny sample supplies no general success-rate claim. Slot members remain string-typed; choosing another string field still typechecks. T's early rules cover literal nested forms, unknown tags, children of void elements and reuse of an identical attribute expression. A misplaced opaque publication splice is accepted statically; runtime validation catches it for saved records but allows the blank paragraph case. Escaping, opaque trusted values and runtime checking are unchanged across approaches. The external tree experiment only partly validates structure, lacks native editor support and full mapped diagnostics, and cannot see inside opaque splices. Neither multi-page reuse nor broad AI generation and repair costs were tested. No shipping application or compiler implementation was edited."
]
questions = [
    "Given only this evidence, what next step best serves dependable AI HTML authoring without assuming an unmeasured advantage?",
    "Select the justified follow-up for reliable AI page construction, accounting for the demonstrated limits and untested benefits.",
    "Which development direction is warranted by these observations while avoiding unsupported claims of generation superiority?"
]
options = {
    'components_only': [
        "Use ordinary typed components as the lasting authoring model and close the tree experiment; accept hierarchy expressed through named nodes and child arrays.",
        "Commit to function-and-record composition alone, discontinue tree exploration, and keep structural relationships in bindings and arrays.",
        "Retain components as the final approach with no further tree work, expressing nesting using the existing Can value composition API."
    ],
    'adopt_tree_now': [
        "Adopt tree syntax alongside typed components now, implementing its compiler and editor integration plus mapped diagnostics as production work.",
        "Choose the hybrid for immediate production development: build native parsing, checking, editor support and source errors for tree layouts while reusing typed helpers.",
        "Proceed directly to a shipping tree-and-components model, adding integrated language tooling and diagnostic mapping."
    ],
    'components_now_tree_candidate': [
        "Use the buildable typed components now and retain the tree as a candidate layout layer. Require original-source diagnostics and further matched semantic edit/repair evidence before deciding on production syntax.",
        "Keep current delivery on valid typed helpers, but leave the declarative layout option open. First demonstrate source-local errors and additional equivalent generation/edit/repair tasks, then decide adoption.",
        "Favor usable components for present work while continuing to regard trees as an unadopted composition proposal; native-location diagnostic proof and more controlled semantic authoring trials should precede a shipping decision."
    ]
}
assert len(set(contexts)) == len(set(questions)) == 3
assert all(len(set(v)) == 3 for v in options.values())
for i in range(3):
    payload = {'model':'jev-latest','state':{'evidence':contexts[i]},'questions':{'html_authoring_direction':{'type':'choice','instructions':questions[i],'criteria':{k:v[i] for k,v in options.items()}}}}
    (OUT/f'request-{i+1}.json').write_text(json.dumps(payload,indent=2)+'\n')

if '--send' in sys.argv:
    r=json.loads((ROOT/'probe-results.json').read_text())
    assert r.get('all_comparison_checks_passed')
    assert all(r['equivalence'].values()) and all(r['edit_equivalence'].values())
    assert r['build']['assertions']['passed']==131
    assert (OUT/'wording-audit.json').is_file()
    for i in range(1,4):
        request=urllib.request.Request('https://api.typesafe.ai/v1/systemone',data=(OUT/f'request-{i}.json').read_bytes(),headers={'Content-Type':'application/json','Authorization':'Bearer '+os.environ['TYPESAFE_API_KEY']})
        start=datetime.datetime.now(datetime.timezone.utc).isoformat()
        with urllib.request.urlopen(request,timeout=45) as response:
            data,status=response.read(),response.status
        (OUT/f'response-{i}.json').write_bytes(data+b'\n')
        (OUT/f'response-{i}.metadata.json').write_text(json.dumps({'started_utc':start,'http_status':status,'endpoint':'https://api.typesafe.ai/v1/systemone'},indent=2)+'\n')
        parsed=json.loads(data)
        print(json.dumps({'request':i,'answers':parsed.get('answers'),'usage':parsed.get('usage')}),flush=True)
