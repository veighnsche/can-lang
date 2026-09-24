"""Prepare three independent reworded Jev judgments for generic-helper trial."""

from pathlib import Path
import datetime
import json
import os
import sys
import urllib.request

OUT = Path(__file__).resolve().parent / "jev-generic"
OUT.mkdir(exist_ok=True)

facts = {
    "purpose": [
        "Can is for AI coding agents, with correctness and reliable edits first and total tokens per successful task second. This question selects a bounded technical experiment, not syntax or production semantics. There are no external users requiring compatibility. Generated TypeScript should use native JS/Bun operations with only contract adapters.",
        "The language serves agent-authored programs rather than human taste. Prioritize sound behavior and edits; measure whole successful-task tokens as a secondary objective. We seek the next controlled prototype, not a final grammar choice. Old spellings need no migration support, and lowering should delegate equivalent operations to JS/Bun.",
        "Assess a technical trial for an AI-agent language. A successful program and safe repair outrank source brevity; total task tokens are a secondary measured cost. The result cannot enact a syntax decision. With zero external adopters, compatibility is irrelevant; generated TS should rely on native JS/Bun behavior where equivalent.",
    ],
    "observations": [
        "At current HEAD, 28/28 assertions pass in an isolated helper project with two callers. Caller A emits only domain::a_failure and caller B only domain::b_failure. Direct match chain keeps each exact bound but repeats work. A shared fixed-bound audit_fixed declares both errors, so each caller handles an impossible other arm. A generic outcome<int,failure> data variant preserves each payload type but requires conversions into and out of completion. Explicit transform and inspect callable parameters expose the operations required by the public helper. Concrete specialization checks operations inside a generic body but its signature alone does not reveal those requirements.",
        "A reproducible Can probe passed 28 checks across two disjoint singleton failure domains. Repeated direct chains retain precise outward errors. One named fixed-bound helper accepts either callback yet advertises their union, forcing an unused error branch per caller. Parameterized result-as-data transports different failure records but adds ingress/egress adapters. The helper can state its transformation and predicate as ordinary callable inputs; without those inputs, generic body requirements appear only when a concrete instance is checked.",
        "The tested baseline has a separate library and callers whose only failures are domain::a_failure and domain::b_failure respectively; all 28 assertions passed. Direct sequencing duplicates the shared operation, fixed-bound reuse widens caller handling to both kinds, and outcome<int,failure> reuse needs explicit completion/data conversions. Passing transform and inspect functions makes required operations visible in the public type. Authored generic bodies are validated after substitution, leaving implicit body-operation needs absent from the declaration header.",
    ],
    "limits": [
        "The probe used singleton error sets and measured source lines, not agent success or total tokens. An arbitrary finite caller error set cannot currently be a source error type parameter; compiler-owned collection combinators specialize callback errors. A trial must preserve named finite outward errors and fail when an operation contract is unmet. Avoid adding many intrinsics without evidence.",
        "Only one error per caller and one integer helper were exercised. Source length is not an agent benchmark. Source functions cannot quantify over an arbitrary callback error set, though catalogue operations can compute such sets. Keep public failures finite and inspectable and reject missing callable requirements; require comparative benefit before expanding core grammar.",
        "Neither multi-error growth nor model edit/repair cost has been tested. Current error-use grammar requires named error declarations, while selected built-ins derive callback bounds. Any proposal must state finite outward possibilities and visible operation requirements. A one-off reduction in lines cannot justify a new language mechanism.",
    ],
}

question = {
    "instructions": [
        "Which next bounded trial most directly tests whether authored reusable helpers need an error-set abstraction beyond the strongest current idioms? Do not infer final language adoption.",
        "Choose the most informative experiment after the fixed-bound and data-result probe, while protecting explicit finite failures and operation contracts.",
        "What should be prototyped next to resolve the library-composition gap on evidence rather than source-line preference? Treat this as experiment selection only.",
    ],
    "criteria": {
        "explicit_error_set_trial": [
            "Prototype an explicitly named finite error-set parameter for authored helpers; compare against direct chains and result-as-data on two larger disjoint error sets, with explicit callable inputs for required operations.",
            "Test a written failure-row variable in a source helper, specialized to each caller, against the existing direct and data-adapter baselines; make transform/inspect callable requirements public and exercise multi-error domains.",
            "Build a narrow authored generic finite-bound prototype and measure larger A/B caller sets plus refactor/diagnostic repair, retaining visible operation parameters and current idioms as controls.",
        ],
        "data_result_only": [
            "Treat generic outcome data plus adapters as permanently sufficient; spend the next trial measuring that design under multi-error callers without a compiler prototype.",
            "Extend the present result-as-data example to larger failure sets and use explicit completion conversions as the lasting general abstraction, with no trial of error-set parameters.",
            "Keep the generic data variant and boundary adapters as the sole reusable mechanism; benchmark its scale and agent repair before contemplating any new bound syntax.",
        ],
        "fixed_bound_only": [
            "Retain one written union bound in each reusable helper and require callers to handle or map unreachable members; measure only this current approach on larger cases.",
            "Scale the fixed error-union baseline, accepting impossible branches in callers and avoiding a new generic mechanism even if the disjoint domains remain separate.",
            "Use current fixed outward error lists as the permanent source pattern, adapting every caller's unused cases; test whether agent tools handle that burden as error sets grow.",
        ],
        "catalogue_intrinsics": [
            "Add compiler-owned specializations for each useful retry/audit helper, as collection combinators do, instead of testing a general authored failure-set parameter.",
            "Move the reusable workflow into maintained catalogue operations with computed callback errors, leaving ordinary source helpers fixed-bound.",
            "Build separate distribution intrinsics for fallible helper patterns and measure their use, rather than making finite error sets an authored generic dimension.",
        ],
    },
}

assert all(len(v) == 3 and len(set(v)) == 3 for v in facts.values())
assert len(set(question["instructions"])) == 3
assert all(len(v) == 3 and len(set(v)) == 3 for v in question["criteria"].values())
for i in range(3):
    payload = {
        "model": "jev-latest",
        "state": {k: v[i] for k, v in facts.items()},
        "questions": {"next_generic_trial": {
            "type": "choice",
            "instructions": question["instructions"][i],
            "criteria": {k: v[i] for k, v in question["criteria"].items()},
        }},
    }
    (OUT / f"request-{i + 1}.json").write_text(json.dumps(payload, indent=2) + "\n")
(OUT / "wording-audit.json").write_text(json.dumps({
    "semantic_check": "All three requests preserve the same 28/28 probe result, baseline alternatives, singleton-limit, finite-error and visible-operation constraints. Every explanatory state field, instruction and option description has a distinct full phrasing; technical identifiers remain exact. Prior Jev answers are excluded.",
    "limit": "Manual equivalence review plus text uniqueness cannot prove absence of framing effects; investigate disagreement.",
}, indent=2) + "\n")

if "--send" in sys.argv:
    key = os.environ["TYPESAFE_API_KEY"]
    for i in range(1, 4):
        req = urllib.request.Request("https://api.typesafe.ai/v1/systemone",
            data=(OUT / f"request-{i}.json").read_bytes(),
            headers={"Content-Type": "application/json", "Authorization": "Bearer " + key})
        started = datetime.datetime.now(datetime.timezone.utc).isoformat()
        with urllib.request.urlopen(req, timeout=60) as response:
            body, status = response.read(), response.status
        (OUT / f"response-{i}.json").write_bytes(body + b"\n")
        (OUT / f"response-{i}.metadata.json").write_text(json.dumps(
            {"startedAt": started, "status": status, "endpoint": "v1/systemone"}, indent=2) + "\n")
        data = json.loads(body)
        print(json.dumps({"request": i, "answers": data.get("answers"), "usage": data.get("usage")}))
