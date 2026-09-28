"""Bounded Jev transport/audit tests; all remote calls are mocked."""

import io
import json
import os
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch
import urllib.error
import zipfile

import jev_service


def evidence():
    return {"kind": "can.performance-report", "schema_version": 1,
            "manifest": {"status": "complete", "quality": "measurement", "source": {"revision": "test"},
                         "requested_suites": ["runtime"], "completed_suites": ["runtime"]},
            "rankings": {}, "cases": {"runtime/tiny": {
                "unit": "ns/op", "parameters": {}, "timing_scope": "bounded synthetic operation",
                "iterations_per_sample": 1,
                "distribution": {"median": 2, "min": 1, "max": 3, "median_absolute_deviation": 1, "n": 3}}}}


def requests():
    return [{"model": "jev-latest", "state": {"explanation": f"variant {index}"}, "questions": {
        "runtime": {"type": "score", "instructions": f"Judge supplied evidence using phrasing {index}",
                    "criteria": [f"variant {index} level {level}" for level in range(6)]}}} for index in range(3)]


def response(request):
    return {"model": "jev-test", "answers": {"runtime": {
        "type": "score", "score": 4, "confidence": 1,
        "probabilities": {str(index): int(index == 4) for index in range(6)},
        "legend": {str(index): text for index, text in enumerate(request["questions"]["runtime"]["criteria"])}}},
        "usage": {"input_tokens": 10, "output_tokens": 10}}


class JevServiceTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix="can-jev-test-")
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.output = self.root / "test.grading.zip"
        self.data = evidence()
        self.requests = requests()
        self.enriched = dict(self.data, assessment={"synthetic": True})
        self.grading = SimpleNamespace(build_requests=lambda data: self.requests,
                                       make_payload=lambda data, requests, responses: {"synthetic": True})
        for mutation in (patch.dict(os.environ, {"TYPESAFE_API_KEY": "secret-test-key"}),
                         patch.dict("sys.modules", {"jev_grading": self.grading}),
                         patch.object(jev_service, "attach_assessment", return_value=self.enriched)):
            mutation.start()
            self.addCleanup(mutation.stop)
        self.calls = []

    def remote(self, request, **kwargs):
        self.assertEqual(request.full_url, jev_service.ENDPOINT)
        self.assertEqual(request.method, "POST")
        self.assertEqual(request.get_header("Authorization"), "Bearer secret-test-key")
        self.assertEqual(kwargs, {"timeout": 55})
        decoded = json.loads(request.data)
        self.calls.append(decoded)
        return io.BytesIO(json.dumps(response(decoded)).encode())

    def archive(self):
        with zipfile.ZipFile(self.output) as archive:
            entries = {name: archive.read(name) for name in archive.namelist()}
        self.assertLess(self.output.stat().st_size, 5 * 1024 * 1024)
        self.assertNotIn(b"secret-test-key", b"".join(entries.values()))
        self.assertFalse(list(self.root.glob(".can-jev-*")))
        return entries

    def test_three_fresh_calls_publish_complete_audit(self):
        with patch.object(jev_service.urllib.request, "urlopen", side_effect=self.remote):
            self.assertEqual(jev_service.run_grading(self.data, self.output), self.enriched)
        self.assertEqual(self.calls, self.requests)
        entries = self.archive()
        self.assertEqual(set(entries), {"assessment.json", "report.json", *(f"requests/{i}.json" for i in range(1, 4)),
                                       *(f"responses/{i}.json" for i in range(1, 4))})
        self.assertEqual(json.loads(entries["report.json"]), self.enriched)


    def test_no_eligible_questions_requires_no_key_and_makes_no_paid_calls(self):
        self.requests = [dict(request, questions={}) for request in self.requests]
        with patch.dict(os.environ, {"TYPESAFE_API_KEY": ""}):
            with patch.object(jev_service.urllib.request, "urlopen") as remote:
                self.assertEqual(jev_service.run_grading(self.data, self.output), self.enriched)
        remote.assert_not_called()
        entries = self.archive()
        for index in range(1, 4):
            self.assertEqual(json.loads(entries[f"responses/{index}.json"])["model"], "not-invoked")

    def test_missing_key_fails_without_network_and_keeps_requests(self):
        with patch.dict(os.environ, {"TYPESAFE_API_KEY": ""}):
            with patch.object(jev_service.urllib.request, "urlopen") as remote:
                with self.assertRaisesRegex(RuntimeError, "TYPESAFE_API_KEY"):
                    jev_service.run_grading(self.data, self.output)
        remote.assert_not_called()
        entries = self.archive()
        self.assertIn("failure.json", entries)
        self.assertNotIn("assessment.json", entries)

    def test_http_failure_is_redacted_preserved_and_never_retried(self):
        failure = urllib.error.HTTPError(jev_service.ENDPOINT, 429, "rate limit", {}, io.BytesIO(b"secret-test-key echoed"))
        with patch.object(jev_service.urllib.request, "urlopen", side_effect=failure) as remote:
            with self.assertRaisesRegex(RuntimeError, "HTTP 429"):
                jev_service.run_grading(self.data, self.output)
        self.assertEqual(remote.call_count, 1)
        self.assertEqual(self.archive()["responses/1.txt"], b"[REDACTED] echoed")

    def test_network_failure_and_malformed_response_stop_before_more_charges(self):
        invalid = response(self.requests[0])
        invalid["answers"]["runtime"]["score"] = 0
        for raw in (b"invalid JSON secret-test-key", json.dumps(invalid).encode(), b"x" * (jev_service.RESPONSE_LIMIT + 1)):
            with self.subTest(raw=raw[:50]):
                with patch.object(jev_service.urllib.request, "urlopen", return_value=io.BytesIO(raw)) as remote:
                    with self.assertRaises(RuntimeError):
                        jev_service.run_grading(self.data, self.output)
                self.assertEqual(remote.call_count, 1)
                self.archive()
                self.output.unlink()
        with patch.object(jev_service.urllib.request, "urlopen", side_effect=TimeoutError("secret-test-key timeout")) as remote:
            with self.assertRaisesRegex(RuntimeError, "timeout") as caught:
                jev_service.run_grading(self.data, self.output)
        self.assertNotIn("secret-test-key", str(caught.exception))
        self.assertEqual(remote.call_count, 1)
        self.archive()

    def test_ineligible_or_existing_archive_prevents_network(self):
        with patch.object(jev_service.urllib.request, "urlopen") as remote:
            for quality in ("smoke", "measurement"):
                data = evidence()
                data["manifest"].update(quality=quality, status="failed")
                with self.assertRaises(ValueError):
                    jev_service.run_grading(data, self.output)
            self.output.write_bytes(b"original")
            with self.assertRaises(FileExistsError):
                jev_service.run_grading(self.data, self.output)
        remote.assert_not_called()
        self.assertEqual(self.output.read_bytes(), b"original")

    def test_interruption_keeps_audit_and_reclaims_staging(self):
        with patch.object(jev_service.urllib.request, "urlopen", side_effect=KeyboardInterrupt):
            with self.assertRaises(KeyboardInterrupt):
                jev_service.run_grading(self.data, self.output)
        self.assertIn("failure.json", self.archive())

    def test_race_preserves_foreign_archive_and_removes_staging(self):
        original_link = os.link
        def race(source, destination):
            Path(destination).write_bytes(b"concurrent archive")
            original_link(source, destination)
        with patch.object(jev_service.urllib.request, "urlopen", side_effect=self.remote):
            with patch.object(jev_service.os, "link", side_effect=race):
                with self.assertRaises(FileExistsError):
                    jev_service.run_grading(self.data, self.output)
        self.assertEqual(self.output.read_bytes(), b"concurrent archive")
        self.assertFalse(list(self.root.glob(".can-jev-*")))


if __name__ == "__main__":
    unittest.main()
