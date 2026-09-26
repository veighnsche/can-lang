"""Wording audit for the 2026-09-26 preparation Jev round.

Checks, before any request is sent:
  1. All three requests parse and share model, state keys, question keys,
     option keys, and question types (structural equivalence).
  2. Every corresponding explanatory field (state values, instructions,
     criteria descriptions) differs pairwise across the three requests
     (full-request wording variation).
Writes jev-wording-audit.json with per-field pairwise comparisons.
Semantic equivalence was reviewed manually while drafting (see README).
"""
import json
from pathlib import Path
from difflib import SequenceMatcher

here = Path(__file__).resolve().parent
reqs = [json.loads((here / f"jev-request-{n}.json").read_text()) for n in (1, 2, 3)]

report = {"structural": {}, "fields": {}, "ok": True}


def fail(msg):
    report["ok"] = False
    report.setdefault("failures", []).append(msg)


# 1. Structural equivalence
models = [r.get("model") for r in reqs]
report["structural"]["model"] = models
if len(set(models)) != 1:
    fail(f"model differs: {models}")
state_keys = [sorted(r.get("state", {})) for r in reqs]
report["structural"]["state_keys"] = state_keys[0]
if not (state_keys[0] == state_keys[1] == state_keys[2]):
    fail(f"state keys differ: {state_keys}")
q_keys = [sorted(r.get("questions", {})) for r in reqs]
report["structural"]["question_keys"] = q_keys[0]
if not (q_keys[0] == q_keys[1] == q_keys[2]):
    fail(f"question keys differ: {q_keys}")
for q in q_keys[0]:
    types = [reqs[i]["questions"][q].get("type") for i in range(3)]
    opts = [sorted(reqs[i]["questions"][q].get("criteria", {})) for i in range(3)]
    if len(set(types)) != 1:
        fail(f"{q}: type differs: {types}")
    if not (opts[0] == opts[1] == opts[2]):
        fail(f"{q}: option keys differ: {opts}")
    report["structural"][q] = {"type": types[0], "options": opts[0]}

# 2. Full wording variation over every explanatory field
fields = []
for sk in state_keys[0]:
    fields.append((f"state.{sk}", [reqs[i]["state"][sk] for i in range(3)]))
for q in q_keys[0]:
    fields.append((
        f"questions.{q}.instructions",
        [reqs[i]["questions"][q]["instructions"] for i in range(3)],
    ))
    for opt in sorted(reqs[0]["questions"][q]["criteria"]):
        fields.append((
            f"questions.{q}.criteria.{opt}",
            [reqs[i]["questions"][q]["criteria"][opt] for i in range(3)],
        ))

report["field_count"] = len(fields)
for name, texts in fields:
    pairs = {}
    for a, b in ((0, 1), (0, 2), (1, 2)):
        ratio = round(SequenceMatcher(None, texts[a], texts[b]).ratio(), 3)
        pairs[f"{a+1}v{b+1}"] = {
            "identical": texts[a] == texts[b],
            "similarity": ratio,
            "lengths": [len(texts[a]), len(texts[b])],
        }
        if texts[a] == texts[b]:
            fail(f"{name}: requests {a+1} and {b+1} identical")
    report["fields"][name] = pairs

report["semantic_equivalence_note"] = (
    "Manual review at draft time: each triple preserves the same facts, "
    "constraints, questions, and alternatives; only explanatory phrasing "
    "varies. Exact identifiers (2cb1bc3, Bun 1.4.2, checks::require, LD29, "
    "counts 100/20000, field counts) and all option keys are stable."
)
(here / "jev-wording-audit.json").write_text(json.dumps(report, indent=2) + "\n")
print(json.dumps({"ok": report["ok"], "fields": len(fields),
                  "failures": report.get("failures", [])}))
