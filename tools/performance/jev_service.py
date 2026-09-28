"""Three fresh Jev consultations with compact, credential-free audit evidence."""

import json
import os
import sys
from pathlib import Path
import tempfile
import urllib.error
import urllib.request
import zipfile

from assessment import measured_slices, attach_assessment, evidence_fingerprint, validate_score_answer

ENDPOINT = "https://api.typesafe.ai/v1/systemone"
TIMEOUT_SECONDS = 55
REQUEST_LIMIT = 384 * 1024
RESPONSE_LIMIT = 256 * 1024
ARCHIVE_LIMIT = 5 * 1024 * 1024


def preflight_archive(path):
    path = Path(path).expanduser()
    if os.path.lexists(path):
        raise FileExistsError(f"Grading archive already exists: {path}")
    if not path.parent.is_dir():
        raise ValueError(f"Grading output parent must be an existing directory: {path.parent}")
    if not os.access(path.parent, os.W_OK):
        raise PermissionError(f"Grading output parent is not writable: {path.parent}")
    return path


def _encode(value):
    return (json.dumps(value, ensure_ascii=False, indent=2, allow_nan=False) + "\n").encode()


def _redact(raw, key):
    # Only payloads are retained: HTTP headers and request objects never enter evidence.
    return raw.replace(key.encode(), b"[REDACTED]") if key else raw


def _publish(entries, path):
    if sum(len(raw) for raw in entries.values()) >= ARCHIVE_LIMIT:
        raise RuntimeError("Grading evidence exceeds the 5 MiB retention bound")
    with tempfile.NamedTemporaryFile(prefix=".can-jev-", suffix=".tmp", dir=path.parent) as staging:
        with zipfile.ZipFile(staging, "w", compression=zipfile.ZIP_DEFLATED) as archive:
            for name, raw in entries.items():
                archive.writestr(name, raw)
        staging.flush()
        if os.fstat(staging.fileno()).st_size >= ARCHIVE_LIMIT:
            raise RuntimeError("Grading archive exceeds the 5 MiB retention bound")
        os.link(staging.name, path)


def _validate_response(request, response):
    if not isinstance(response, dict) or not isinstance(response.get("model"), str) or not response["model"]:
        raise ValueError("Jev response must identify its model")
    answers = response.get("answers")
    if not isinstance(answers, dict) or set(answers) != set(request["questions"]):
        raise ValueError("Jev response must answer exactly the requested suites")
    for suite, question in request["questions"].items():
        answer = answers[suite]
        validate_score_answer(answer)
        expected_legend = {str(index): criterion for index, criterion in enumerate(question["criteria"])}
        if answer.get("legend") != expected_legend:
            raise ValueError(f"Jev answer for {suite} does not match the requested rubric")


def run_grading(data, archive_path):
    """Grade verified measured data; publish requests, replies and enriched report.

    The CLI verifies raw measurement records before calling this function. Each
    request runs once. No HTTP failure is automatically retried or substituted.
    """
    path = preflight_archive(archive_path)
    measured_slices(data)
    evidence_fingerprint(data)
    import jev_grading
    requests = jev_grading.build_requests(data)
    if not isinstance(requests, list) or len(requests) != 3:
        raise ValueError("Grading requires three fresh Jev request variants")
    entries = {}
    for index, request in enumerate(requests, 1):
        if request.get("model") != "jev-latest" or not isinstance(request.get("questions"), dict):
            raise ValueError("Grading requests require jev-latest and suite questions")
        if any(question.get("type") != "score" or len(question.get("criteria", [])) != 6
               for question in request["questions"].values()):
            raise ValueError("Grading requires six-level Score questions")
        encoded = _encode(request)
        if len(encoded) > REQUEST_LIMIT:
            raise ValueError("Jev grading request exceeds the bounded input limit")
        entries[f"requests/{index}.json"] = encoded
    key = os.environ.get("TYPESAFE_API_KEY", "").strip()
    responses = []
    try:
        has_questions = any(request["questions"] for request in requests)
        if not has_questions:
            print("Jev grading omitted: no eligible suite questions; all slices remain ungraded.", file=sys.stderr, flush=True)
            responses = [{"model": "not-invoked", "answers": {}, "not_invoked_reason": "No eligible suite questions"}
                         for _ in requests]
            for index, response in enumerate(responses, 1):
                entries[f"responses/{index}.json"] = _encode(response)
        if not key and has_questions:
            raise RuntimeError("Automatic PDF grading requires TYPESAFE_API_KEY; use --no-grades for offline export")
        for index, request in enumerate(requests if has_questions else [], 1):
            print(f"Jev grading consultation {index}/3 (no automatic retry)", file=sys.stderr, flush=True)
            outbound = urllib.request.Request(ENDPOINT, data=entries[f"requests/{index}.json"], method="POST",
                                             headers={"Authorization": f"Bearer {key}", "Content-Type": "application/json"})
            try:
                with urllib.request.urlopen(outbound, timeout=TIMEOUT_SECONDS) as reply:
                    raw = reply.read(RESPONSE_LIMIT + 1)
            except urllib.error.HTTPError as error:
                with error:
                    raw = error.read(RESPONSE_LIMIT + 1)
                entries[f"responses/{index}.txt"] = _redact(raw[:RESPONSE_LIMIT], key)
                raise RuntimeError(f"Jev consultation {index} failed with HTTP {error.code}; no retry was made") from None
            except (OSError, urllib.error.URLError) as error:
                detail = _redact(str(error).encode(), key).decode(errors="replace")[:1000]
                raise RuntimeError(f"Jev consultation {index} failed: {detail}; no retry was made") from None
            entries[f"responses/{index}.json"] = _redact(raw[:RESPONSE_LIMIT], key)
            if len(raw) > RESPONSE_LIMIT:
                raise RuntimeError(f"Jev consultation {index} exceeded the bounded response limit")
            try:
                response = json.loads(_redact(raw, key))
                _validate_response(request, response)
            except (ValueError, UnicodeError) as error:
                detail = _redact(str(error).encode(), key).decode(errors="replace")[:1000]
                raise RuntimeError(f"Jev consultation {index} returned an invalid response: {detail}") from None
            responses.append(response)
        payload = jev_grading.make_payload(data, requests, responses)
        enriched = attach_assessment(data, payload)
        entries["assessment.json"] = _redact(_encode(payload), key)
        entries["report.json"] = _redact(_encode(enriched), key)
        # Check retention before success; failures can still keep the remote evidence.
        if sum(len(raw) for raw in entries.values()) >= ARCHIVE_LIMIT:
            del entries["report.json"]
            raise RuntimeError("Enriched grading report exceeds the 5 MiB retention bound")
    except (Exception, KeyboardInterrupt) as error:
        detail = _redact(str(error).encode(), key).decode(errors="replace")[:2000]
        entries["failure.json"] = _encode({"status": "failed", "error": detail,
                                          "completed_consultations": len(responses)})
        _publish({name: _redact(raw, key) for name, raw in entries.items()}, path)
        if isinstance(error, KeyboardInterrupt):
            raise
        raise RuntimeError(f"{detail}. Grading evidence saved: {path}") from None
    _publish({name: _redact(raw, key) for name, raw in entries.items()}, path)
    return enriched
