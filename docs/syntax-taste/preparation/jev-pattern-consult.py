"""Prepare and send three fresh pattern-intent Jev consultations."""

from pathlib import Path
import datetime
import json
import os
import sys
import urllib.request

OUT = Path(__file__).resolve().parent / "jev-pattern"
OUT.mkdir(exist_ok=True)

facts = {
    "audience": [
        "Can is a language for AI coding agents. Human familiarity and comfort are not design goals. Correctness and reliable editing dominate; total tokens per successful agent task are a secondary measured outcome.",
        "The intended author is an AI coding agent, even where the resulting syntax is hard for a person. Prefer correct generation and safe refactoring; count whole successful-task tokens only as a secondary metric.",
        "Can serves machine coding workflows. Human-friendly spelling has no independent value. The primary tests are semantic success and edit reliability, with prompts, code, diagnostics and retries measured secondarily as total task tokens.",
    ],
    "observed_rule": [
        "In a match over paid | pending | declined, the compiler resolves known bare leaf names as tests, but an unresolved unqualified bare name becomes a whole-value binder with exhaustive coverage. A final-arm typo decliend compiles and still hides a later refunded leaf; declined instead becomes non-exhaustive after refunded is added. decliend() already fails. These outcomes were executed at an earlier source-equivalent compiler revision and inspected in the current checker.",
        "Current ordinary matching gives paid/pending/declined bare identifiers nominal-case meaning when resolvable. Unknown plain decliend binds the entire scrutinee, so a misspelled last arm remains accepted when refunded extends the variant. A correctly spelled three-arm match reports missing refunded; a nonexistent decliend() constructor is rejected. The probe ran on the reviewed revision and current checker source is unchanged.",
        "The checked nominal variant has leaves paid, pending and declined. Name-pattern resolution treats a recognized leaf as a case and any unknown simple identifier as an all-covering capture. The tested typo decliend at the end therefore masks both the intended declined check and a subsequently added refunded case, while declined exposes that addition and decliend() diagnoses the typo. Current source retains the same rule as the executed probe.",
    ],
    "constraints": [
        "Any replacement must make decliend a compile error in that position, preserve an explicit catch-all and a way to capture the whole value, work in nested/array patterns, keep exhaustiveness meaningful under a new leaf, and lower to native JavaScript/Bun with only Can-contract adapters. No old syntax compatibility is required. These are technical prototype choices; the user chooses syntax later.",
        "The candidate rule has to reject the misspelled final case, admit deliberate wildcard/whole-value binding, and compose through nested and array patterns without swallowing newly added leaves. Generated TypeScript should use equivalent JS/Bun operations plus contract glue. With no external adopters, migration is not a constraint. A later user discussion decides concrete spelling.",
        "A viable semantic experiment must distinguish intended nominal tests from captures at the typo site, provide intentional fallback and binding, handle embedded patterns, and retain future-case detection. Native JS/Bun lowering is required, with adapters only for Can semantics; backward compatibility is unnecessary. Treat proposed keywords as illustrative pending user syntax choice.",
    ],
}

question = {
    "instructions": [
        "Which coherent pattern-intent rule should be prototyped first for safe agent edits, given that all options reject the observed typo? Answer from the supplied rules and constraints, not an imagined benchmark.",
        "Select the strongest first technical trial among these complete matching policies. All diagnose decliend; weigh compositional checks and likely repair clarity while acknowledging missing agent measurements.",
        "For a bounded compiler experiment, choose the rule most likely to preserve case-exhaustiveness under edits without surprising nested behavior. Each alternative satisfies the typo acceptance gate; task-token evidence is still absent.",
    ],
    "criteria": {
        "explicit_binding_everywhere": [
            "Bare identifiers are always resolved nominal tests or errors at every pattern depth. A distinct bind x form captures a value; _ ignores it. Apply the same intent rule to variant, record and array subpatterns.",
            "Across all matching contexts, a plain name must name a nominal leaf; an unknown one is invalid. Use bind x for captures and _ for discard, including array elements and nested destructuring.",
            "Make unadorned names checked case references universally, with bind x and _ as explicit capture/discard forms in both top-level and embedded patterns.",
        ],
        "explicit_both_intents": [
            "Nominal tests must spell leaf(); captures must spell bind x; _ is fallback. Unadorned names have no pattern meaning in nominal-variant positions, including nested ones.",
            "Reserve constructor-shaped leaf() for tests and bind x for value capture within every variant-valued pattern location; keep _ as ignore. A lone name there is diagnosed.",
            "In variant contexts, require both alternatives to show intent: leaf() for nominal checking and bind x for capture, with _ for catch-all; a bare identifier is invalid.",
        ],
        "contextual_variant_rule": [
            "Where the expected type is a nominal variant or leaf, bare known leaf names test, unknown bare names error, bind x captures and _ ignores. Retain existing bare binders in other pattern contexts, including nonvariant array elements.",
            "Use type-directed checking only for variant-valued locations: recognized simple leaf names are tests and unresolved names fail; bind x and _ are deliberate fallback. Outside that type context, plain binder behavior remains.",
            "At positions statically expected to hold a nominal variant, plain names must resolve as admitted leaves; captures use bind x and discards _. Other expected types still interpret bare names as captures, even inside arrays.",
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
        "questions": {
            "pattern_intent_trial": {
                "type": "choice",
                "instructions": question["instructions"][i],
                "criteria": {k: v[i] for k, v in question["criteria"].items()},
            }
        },
    }
    (OUT / f"request-{i + 1}.json").write_text(json.dumps(payload, indent=2) + "\n")

(OUT / "wording-audit.json").write_text(
    json.dumps({
        "semantic_check": "All three requests retain the same observed typo behavior, agent-first objective, native-lowering and no-compatibility constraints, and three complete alternative rules. Every explanatory state field, question and option description is freshly worded. Illustrative bind x/leaf() identifiers are stable technical examples. No prior Jev answer is included.",
        "limits": "This mechanical uniqueness check and human semantic audit cannot prove absence of wording bias; disagreements require investigation.",
    }, indent=2) + "\n"
)

if "--send" in sys.argv:
    key = os.environ["TYPESAFE_API_KEY"]
    for i in range(1, 4):
        req = urllib.request.Request(
            "https://api.typesafe.ai/v1/systemone",
            data=(OUT / f"request-{i}.json").read_bytes(),
            headers={"Content-Type": "application/json", "Authorization": "Bearer " + key},
        )
        started = datetime.datetime.now(datetime.timezone.utc).isoformat()
        with urllib.request.urlopen(req, timeout=60) as response:
            body, status = response.read(), response.status
        (OUT / f"response-{i}.json").write_bytes(body + b"\n")
        (OUT / f"response-{i}.metadata.json").write_text(
            json.dumps({"startedAt": started, "status": status, "endpoint": "v1/systemone"}, indent=2) + "\n"
        )
        data = json.loads(body)
        print(json.dumps({"request": i, "answers": data.get("answers"), "usage": data.get("usage")}))
