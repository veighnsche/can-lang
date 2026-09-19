#!/usr/bin/env python3
"""Prepare self-contained JEV packets; --submit sends only explicitly selected IDs.

Run from any directory. Requests/responses are audit artifacts, never language
approval. This script reads TYPESAFE_API_KEY only for the fixed HTTPS endpoint;
credentials and response headers are never persisted. No automatic retry,
truncation, redirect, or overwrite of an existing response is permitted.
"""

import argparse
import datetime
import hashlib
import json
import math
import os
from pathlib import Path
import re
import urllib.error
import urllib.request

BASE = Path(__file__).resolve().parent
MODEL = "jev-1.13.0"
ENDPOINT = "https://api.typesafe.ai/v1/systemone"
REQUEST_BYTE_BUDGET = 30000

# Whole sections, selected deliberately for each decision. Never slice text by
# length. Every packet also receives all of shared-model.md and its alternatives.
SECTIONS = {
    "sequencing": {
        "authority-evidence.md": ["Current module, ownership, and revision model", "Contracts, proof, and acceptance"],
        "host-async.md": ["Current confirmed findings F03-F05", "Four distinct boundaries"],
        "language-core.md": ["Implemented contextual typing boundary"]},
    "successes": {
        "language-core.md": ["Implemented success and error outcomes", "Implemented contextual typing boundary"],
        "host-async.md": ["Current emitted layouts and exact numerics", "Four distinct boundaries"]},
    "calls": {
        "language-core.md": ["Implemented calls, sequencing, and patterns", "Implemented generics, function values, and recursion"],
        "authority-evidence.md": ["Effects and capabilities"]},
    "modules": {
        "authority-evidence.md": ["Current module, ownership, and revision model", "Effects and capabilities", "Documentation contradictions to resolve before review"],
        "host-async.md": ["Current confirmed findings F03-F05"]},
    "pure_source_tests": {
        "authority-evidence.md": ["Tests, scripts, coverage, and location", "Contracts, proof, and acceptance", "Effects and capabilities", "Termination and resource bounds"]},
    "host_boundary": {
        "host-async.md": ["Shared language model", "Current emitted layouts and exact numerics", "Four distinct boundaries", "Trust, authority, and faults", "Current confirmed findings F03-F05", "Actual stdlib customers and demand"],
        "authority-evidence.md": ["Effects and capabilities"]},
    "error_abstraction": {
        "language-core.md": ["Implemented success and error outcomes", "Implemented calls, sequencing, and patterns", "Implemented generics, function values, and recursion"],
        "authority-evidence.md": ["Contracts, proof, and acceptance"]},
    "b05_process": {
        "host-async.md": ["B05 proposal A: uniform completion-capable callable ABI", "B05 proposal B: uniform private control-frame machine", "Proposed settlement contract and unresolved differences", "Balanced decision and process options"]},
    "contextual_typing": {
        "language-core.md": ["Implemented success and error outcomes", "Implemented contextual typing boundary", "Implemented generics, function values, and recursion"]},
    "collection_spelling": {
        "language-core.md": ["Implemented value and numeric model", "Implemented contextual typing boundary", "Implemented generics, function values, and recursion", "Documentation contradictions and live customers"]},
    "literals": {
        "language-core.md": ["Implemented value and numeric model", "Implemented contextual typing boundary"],
        "host-async.md": ["Current emitted layouts and exact numerics", "Trust, authority, and faults"]},
    "boolean_evaluation": {
        "language-core.md": ["Implemented value and numeric model", "Implemented calls, sequencing, and patterns"],
        "authority-evidence.md": ["Contracts, proof, and acceptance"]},
    "argument_labels": {
        "language-core.md": ["Implemented calls, sequencing, and patterns", "Implemented generics, function values, and recursion"],
        "authority-evidence.md": ["Tests, scripts, coverage, and location"]},
}


def encoded(value):
    return json.dumps(value, ensure_ascii=False, indent=2).encode("utf-8") + b"\n"


def digest(data):
    return hashlib.sha256(data).hexdigest()


def sections(path):
    text = path.read_text()
    parts = re.split(r"(?m)^## (.+)\n", text)
    return {parts[i]: "## " + parts[i] + "\n" + parts[i + 1]
            for i in range(1, len(parts), 2)}


def prepare():
    questions = json.loads((BASE / "questions.json").read_text())
    assert set(questions) == set(SECTIONS) and len(questions) == 13
    packets = BASE / "requests"
    packets.mkdir(exist_ok=True)
    manifest = {}
    for key, decision in questions.items():
        evidence = {}
        sources = {"shared-model.md": digest((BASE / "shared-model.md").read_bytes()),
                   "questions.json": digest((BASE / "questions.json").read_bytes())}
        for filename, selected in SECTIONS[key].items():
            path = BASE / filename
            available = sections(path)
            evidence[filename] = "\n".join(available[heading] for heading in selected)
            sources[filename] = digest(path.read_bytes())
        criteria = dict(decision["options"])
        criteria["defer"] = {
            "design": "Leave this choice unresolved pending missing evidence or a better alternative.",
            "benefit": "Avoids mistaking a limited preference judgment for an adequate design decision.",
            "cost": "Delays a decision and the work that depends on it.",
            "obligation": "Use when no offered direction is supported by the supplied facts; do not invent repository facts or assume the audit author's preference is correct.",
            "after": "Record the unresolved semantic, authority, cost or usability issue before implementation."}
        request = {
            "model": MODEL,
            "state": {
                "shared_can_model": (BASE / "shared-model.md").read_text(),
                "decision_current_example": decision["before"],
                "topic_evidence": evidence,
                "scope": "Evidence excerpts are whole named sections. All alternatives' after examples are proposed/schematic, not current Can grammar. No repository access, links, other requests, historical JEV scores or unseen facts may be assumed. Choose a direction, not implementation approval."
            },
            "questions": {key: {
                "type": "choice",
                "instructions": decision["question"] + " Compare each option's benefit, cost and obligations using the supplied current evidence. An option's label or order conveys no preference. Select defer if the supplied evidence does not justify any offered direction.",
                "criteria": criteria}}
        }
        body = encoded(request)
        # Byte proxy is deliberately pessimistic, not an exact provider tokenizer.
        # Our envelope is below 30k UTF-8 bytes, leaving margin against the live
        # documented 32k state+question and 64k aggregate token limits. Record the
        # provider's actual billed input count after submission; do not truncate.
        if len(body) > REQUEST_BYTE_BUDGET:
            raise ValueError(f"{key}: {len(body)} bytes exceeds budget; deliberately narrow selected sections")
        target = packets / (key + ".json")
        receipt = BASE / "responses" / (key + ".json")
        if receipt.exists():
            old = json.loads(receipt.read_text())
            if old["request_sha256"] != digest(body):
                raise ValueError(f"{key}: request changed after recorded response; create a new review version")
        target.write_bytes(body)
        manifest[key] = {"request": "requests/" + key + ".json", "request_sha256": digest(body),
                         "request_utf8_bytes": len(body), "source_sha256": sources,
                         "included_sections": SECTIONS[key], "question_count": 1}
    (BASE / "packet-manifest.json").write_bytes(encoded({
        "model": MODEL, "endpoint": ENDPOINT,
        "documented_limits": {"state_plus_longest_question_tokens": 32000, "whole_request_tokens": 64000},
        "preflight": "Reject over 30000 serialized UTF-8 bytes, including JSON overhead. This is a conservative engineering proxy, not an exact provider token count. No text truncation or API truncation option. Receipts record actual provider input usage.",
        "packets": manifest}))
    return manifest


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise RuntimeError("Refusing API redirect")


def validate_response(response, request):
    assert response["model"] == MODEL
    assert set(response["answers"]) == set(request["questions"])
    for key, answer in response["answers"].items():
        assert answer["type"] == "choice"
        probs = answer["probabilities"]
        assert set(probs) == set(request["questions"][key]["criteria"])
        assert all(isinstance(p, (int, float)) and not isinstance(p, bool)
                   and math.isfinite(p) and 0 <= p <= 1 for p in probs.values())
        assert abs(sum(probs.values()) - 1) < 0.000001
        assert answer["choice"] in probs
        assert probs[answer["choice"]] == max(probs.values())
        confidence = answer["confidence"]
        assert isinstance(confidence, (int, float)) and not isinstance(confidence, bool)
        assert math.isfinite(confidence) and 0 <= confidence <= 1
    assert isinstance(response["usage"]["input_tokens"], int)
    assert 0 < response["usage"]["input_tokens"] < 32000


def submit(key, entry):
    receipts = BASE / "responses"
    receipts.mkdir(exist_ok=True)
    output = receipts / (key + ".json")
    raw_output = receipts / (key + ".body.json")
    if output.exists() or raw_output.exists():
        raise ValueError(f"Refusing duplicate submission for {key}")
    body = (BASE / entry["request"]).read_bytes()
    assert digest(body) == entry["request_sha256"]
    credential = os.environ.get("TYPESAFE_API_KEY")
    if not credential:
        raise RuntimeError("TYPESAFE_API_KEY is not configured")
    req = urllib.request.Request(ENDPOINT, data=body, method="POST",
                                 headers={"Authorization": "Bearer " + credential,
                                          "Content-Type": "application/json"})
    started = datetime.datetime.now(datetime.timezone.utc).isoformat()
    try:
        with urllib.request.build_opener(NoRedirect()).open(req, timeout=55) as result:
            status, raw = result.status, result.read()
    except urllib.error.HTTPError as error:
        status, raw = error.code, error.read()
    # Save original bytes even if JSON/schema validation fails. Only this fixed
    # API's response body is saved; headers and the request credential are absent.
    with raw_output.open("xb") as saved:
        saved.write(raw)
    receipt = {"started_at": started, "completed_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
               "endpoint": ENDPOINT, "request_sha256": digest(body), "http_status": status,
               "response_body_sha256": digest(raw), "response_body": key + ".body.json"}
    output.write_bytes(encoded(receipt))
    if status != 200:
        raise RuntimeError(f"{key}: HTTP {status}; response saved, no retry")
    response = json.loads(raw)
    validate_response(response, json.loads(body))
    receipt.update({"response": response, "validated": True})
    output.write_bytes(encoded(receipt))
    answer = response["answers"][key]
    print(json.dumps({"question": key, "choice": answer["choice"],
                      "confidence": answer["confidence"], "probabilities": answer["probabilities"],
                      "usage": response["usage"]}), flush=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--submit", nargs="+", choices=list(SECTIONS))
    args = parser.parse_args()
    manifest = prepare()
    if args.submit:
        for key in args.submit:
            submit(key, manifest[key])
    else:
        print(json.dumps({key: entry["request_utf8_bytes"] for key, entry in manifest.items()}, indent=2))
