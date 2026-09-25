"""Verify the three UP15 Jev requests share no prose while preserving facts."""
import json
from pathlib import Path
HERE = Path(__file__).resolve().parent
reqs = [json.load(open(HERE / f"request-{i}.json")) for i in (1, 2, 3)]

def prose(req):
    out = set()
    for q in req["questions"].values():
        out.add(("instructions", q["instructions"]))
        for k, v in q["criteria"].items():
            out.add(("criteria", v))
    for k, v in req["state"].items():
        out.add(("state", v))
    return out

p = [prose(r) for r in reqs]
for i in range(3):
    for j in range(i + 1, 3):
        shared = p[i] & p[j]
        print(f"request-{i+1} vs request-{j+1}: {len(shared)} shared prose strings")
        for kind, text in sorted(shared):
            print(f"  SHARED [{kind}]: {text[:120]}")

# Semantic-equivalence checklist: required facts/constraints/alternatives present in each.
checks = {
    "59 reachable modules": ["59"],
    "module inventory of blockers": ["TOML", "Cookie", "CSRF", "sleep", "stderr", "escapeHTML", "Markdown", "asset", "node:crypto"],
    "audit fail-closed rule": ["never trust", "forbidden", "fails"],
    "frozen check/emit": ["concurrent", "frozen", "locked", "cannot change"],
    "zero users/no compat": ["zero", "no compat"],
    "overlay + gate extensible": ["overlay", "substitut", "gate", "denylist", "capability"],
    "server tests green": ["server", "green", "passing"],
    "canonical assets owned elsewhere": ["asset", "lane"],
}
for i, r in enumerate(reqs):
    blob = json.dumps(r)
    missing = [name for name, alts in checks.items() if not any(a in blob for a in alts)]
    print(f"request-{i+1} missing facts: {missing or 'none'}")
# Question coverage: each request must ask all six topics.
for i, r in enumerate(reqs):
    print(f"request-{i+1} questions: {len(r['questions'])}")
