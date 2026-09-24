#!/usr/bin/env python3
"""Illustrative concrete type-state reachability for three written call edges.

This model is intentionally not a compiler or soundness proof. It only applies
the type-argument transformations in the neighboring Can fixtures.
"""

from __future__ import annotations

import json
from pathlib import Path


def successor(case: str, state: tuple[str, str]) -> tuple[str, str] | None:
    name, item = state
    if case == "stationary-mutual":
        return ("second" if name == "first" else "first", item)
    if case == "expanding-mutual":
        return ("b", f"{item}[]") if name == "a" else ("a", item)
    if case == "acyclic-nested-public":
        return ("identity", f"box<{item}>") if name == "nested" else None
    raise ValueError(case)


def trace(case: str, start: tuple[str, str], maximum: int = 12) -> dict[str, object]:
    seen: dict[tuple[str, str], int] = {}
    states: list[str] = []
    current: tuple[str, str] | None = start
    while current is not None and len(states) < maximum:
        if current in seen:
            return {"states": states, "outcome": "repeated state", "repeat_at": seen[current]}
        seen[current] = len(states)
        states.append(f"{current[0]}<{current[1]}>")
        current = successor(case, current)
    return {"states": states, "outcome": "stopped" if current is None else "step cap reached"}


if __name__ == "__main__":
    cases = {
        "stationary-mutual": ("first", "int"),
        "expanding-mutual": ("a", "int"),
        "acyclic-nested-public": ("nested", "int"),
    }
    output = {name: trace(name, initial) for name, initial in cases.items()}
    (Path(__file__).parent / "flow-model-results.json").write_text(json.dumps(output, indent=2, sort_keys=True) + "\n")
