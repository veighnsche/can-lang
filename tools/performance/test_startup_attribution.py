"""Startup attribution packet tests (G27).

Covers the frozen G25 contract in docs/performance/generated-startup-tasks.md
(section "Frozen G25 contract") for tools/performance/startup-attribution.py
(driver) and tools/performance/startup-attribution.ts (AST probe, run under
node). Style mirrors tools/performance/drivers/test_runtime_driver.py:
stdlib unittest, importlib spec loading for hyphenated modules, tempfile,
unittest.mock. Fast bounded checks only; no network, no bun/go.
"""
import contextlib
import errno
import hashlib
import importlib.util
import io
import json
import os
import re
import shutil
import signal
import subprocess
import sys
import tempfile
import threading
import time
import unittest
from pathlib import Path

HERE = Path(__file__).resolve()
REPO = HERE.parents[2]
spec = importlib.util.spec_from_file_location(
    "startup_attribution", HERE.with_name("startup-attribution.py"))
startup = importlib.util.module_from_spec(spec)
spec.loader.exec_module(startup)

PROBE_TS = HERE.with_name("startup-attribution.ts")
NODE = shutil.which("node")
needs_node = unittest.skipUnless(NODE, "node is required for the AST probe")

FULL_STATE = """let count = 0;
export function $canInitialize() {
  const a = makeA(1,
    2);
  const b = combine(a, makeB(a));
  const c = new Foo(a);
  const d = base.method(b);
  const e = 1, f = 2;
  count = 5;
  b.value = 3;
  sideEffect(a, b);
}
"""

IMPORT_LINE = ('import {__canStartupMark as __canStartupMark}'
               ' from "./startup-probe.ts";\n')


def run_probe(*args, timeout=120):
    return subprocess.run(
        [NODE, "--no-warnings", str(PROBE_TS), *args],
        capture_output=True, text=True, timeout=timeout, cwd=str(REPO))


def write_state(directory, name, text):
    path = Path(directory) / name
    path.write_text(text)
    return path


class ProbeTransformTests(unittest.TestCase):
    @needs_node
    def test_inventory_and_instrument_roundtrip(self):
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(directory, "state.ts", FULL_STATE)
            manifest_path = Path(directory) / "manifest.json"
            result = run_probe("inventory", "--state", str(state),
                               "--manifest", str(manifest_path))
            self.assertEqual(result.returncode, 0, msg=result.stderr)
            self.assertEqual(result.stderr, "")
            manifest = json.loads(manifest_path.read_text())
            self.assertEqual(manifest["schema"],
                             "can.startup-probe-manifest/1")
            self.assertEqual(Path(manifest["state_path"]).resolve(),
                             state.resolve())
            self.assertEqual(manifest["state_sha256"],
                             hashlib.sha256(FULL_STATE.encode()).hexdigest())
            self.assertEqual(manifest["state_bytes"],
                             len(FULL_STATE.encode()))
            initializer = manifest["initializer"]
            self.assertEqual(initializer["name"], "$canInitialize")
            self.assertEqual(initializer["statement_count"], 8)
            self.assertIsNone(manifest["probe_specifier"])
            self.assertIsNone(manifest["instrumented_sha256"])
            statements = manifest["statements"]
            self.assertEqual(len(statements), 8)
            self.assertEqual(initializer["start"], statements[0]["start"])
            self.assertEqual(initializer["end"], statements[-1]["end"])
            expected = [
                ("VariableStatement", "a", ["makeA"]),
                ("VariableStatement", "b", ["combine", "makeB"]),
                ("VariableStatement", "c", ["new:Foo"]),
                ("VariableStatement", "d", ["base.method"]),
                ("VariableStatement", "e,f", []),
                ("ExpressionStatement", "count", []),
                ("ExpressionStatement", "complex:", []),
                ("ExpressionStatement", "call:sideEffect", ["sideEffect"]),
            ]
            previous_start, previous_end = -1, -1
            for position, item in enumerate(statements):
                kind, binding, factories = expected[position]
                self.assertEqual(item["index"], position)
                start, end = item["start"], item["end"]
                self.assertLess(start, end)
                self.assertGreater(start, previous_start)
                self.assertGreater(end, previous_end)
                self.assertGreaterEqual(start, previous_end)
                previous_start, previous_end = start, end
                span = FULL_STATE[start:end]
                self.assertTrue(span.endswith(";"))
                self.assertEqual(item["bytes"], len(span.encode()))
                self.assertEqual(item["sha256"],
                                 hashlib.sha256(span.encode()).hexdigest())
                self.assertEqual(item["kind"], kind)
                if binding == "complex:":
                    self.assertTrue(item["binding"].startswith("complex:"),
                                    msg=item["binding"])
                else:
                    self.assertEqual(item["binding"], binding)
                self.assertEqual(item["factories"], factories)
            multiline = FULL_STATE[statements[0]["start"]:statements[0]["end"]]
            self.assertIn("\n", multiline)
            later = "".join(FULL_STATE[item["start"]:item["end"]]
                            for item in statements[1:4])
            self.assertIn("a", later)
            self.assertTrue(startup.check_manifest_coverage(manifest, FULL_STATE))
            out_path = Path(directory) / "state.out.ts"
            manifest2_path = Path(directory) / "manifest2.json"
            result = run_probe(
                "instrument", "--state", str(state), "--out", str(out_path),
                "--manifest", str(manifest2_path),
                "--probe-specifier", "./startup-probe.ts")
            self.assertEqual(result.returncode, 0, msg=result.stderr)
            self.assertEqual(result.stderr, "")
            instrumented = out_path.read_text()
            manifest2 = json.loads(manifest2_path.read_text())
            self.assertTrue(instrumented.startswith(IMPORT_LINE))
            self.assertEqual(manifest2["probe_specifier"],
                             "./startup-probe.ts")
            self.assertEqual(
                manifest2["instrumented_sha256"],
                hashlib.sha256(instrumented.encode()).hexdigest())
            self.assertEqual(manifest2["statements"],
                             manifest["statements"])
            expected_out = IMPORT_LINE + FULL_STATE
            for index in range(len(statements) - 1, -1, -1):
                item = statements[index]
                start, end = item["start"], item["end"]
                offset = len(IMPORT_LINE)
                expected_out = (
                    expected_out[:offset + start]
                    + f"__canStartupMark({index},0);\n"
                    + expected_out[offset + start:offset + end]
                    + f"\n__canStartupMark({index},1);"
                    + expected_out[offset + end:])
            self.assertEqual(instrumented, expected_out)
            for index, item in enumerate(statements):
                span = FULL_STATE[item["start"]:item["end"]]
                self.assertEqual(
                    instrumented.count(f"__canStartupMark({index},0);"), 1)
                self.assertEqual(
                    instrumented.count(f"__canStartupMark({index},1);"), 1)
                self.assertIn(span, instrumented)
            self.assertEqual(instrumented.count("__canStartupMark("), 16)
            self.assertEqual(instrumented.count("{"),
                             FULL_STATE.count("{") + 1)
            self.assertEqual(instrumented.count("}"),
                             FULL_STATE.count("}") + 1)
            self.assertTrue(startup.check_manifest_coverage(
                manifest2, FULL_STATE))

    @needs_node
    def test_single_statement_minimal(self):
        with tempfile.TemporaryDirectory() as directory:
            text = ("export function $canInitialize() {\n"
                    "  const only = 1;\n}\n")
            state = write_state(directory, "state.ts", text)
            manifest_path = Path(directory) / "manifest.json"
            out_path = Path(directory) / "out.ts"
            result = run_probe(
                "instrument", "--state", str(state), "--out", str(out_path),
                "--manifest", str(manifest_path),
                "--probe-specifier", "./startup-probe.ts")
            self.assertEqual(result.returncode, 0, msg=result.stderr)
            manifest = json.loads(manifest_path.read_text())
            self.assertEqual(manifest["initializer"]["statement_count"], 1)
            item = manifest["statements"][0]
            self.assertEqual(
                (item["kind"], item["binding"], item["factories"]),
                ("VariableStatement", "only", []))
            out = out_path.read_text()
            self.assertIn("__canStartupMark(0,0);\n", out)
            self.assertIn("\n__canStartupMark(0,1);", out)

    @needs_node
    def test_adjacent_statements_succeed(self):
        with tempfile.TemporaryDirectory() as directory:
            text = ("export function $canInitialize() { "
                    "const a = 1;const b = 2; }\n")
            state = write_state(directory, "state.ts", text)
            manifest_path = Path(directory) / "manifest.json"
            result = run_probe("inventory", "--state", str(state),
                               "--manifest", str(manifest_path))
            self.assertEqual(result.returncode, 0, msg=result.stderr)
            manifest = json.loads(manifest_path.read_text())
            self.assertEqual(manifest["initializer"]["statement_count"], 2)
            first, second = manifest["statements"]
            self.assertEqual(second["start"], first["end"])


class ProbeErrorTests(unittest.TestCase):
    def assert_probe_error(self, result, code, *absent):
        self.assertEqual(result.returncode, 2, msg=result.stderr)
        lines = result.stderr.strip().splitlines()
        self.assertEqual(len(lines), 1, msg=result.stderr)
        self.assertTrue(
            lines[0].startswith(f"startup-probe-error {code} "),
            msg=lines[0])
        self.assertGreater(len(lines[0].split(" ", 2)), 2)
        for path in absent:
            self.assertFalse(Path(path).exists(), msg=str(path))

    @needs_node
    def test_error_parse_error(self):
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(
                directory, "state.txt",
                "export function $canInitialize() { const a = 1; }\n")
            manifest_path = Path(directory) / "manifest.json"
            result = run_probe("inventory", "--state", str(state),
                               "--manifest", str(manifest_path))
            self.assert_probe_error(result, "parse_error", manifest_path)

    @needs_node
    def test_error_has_diagnostics(self):
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(
                directory, "state.ts",
                "export function $canInitialize() { const a = ; }\n")
            manifest_path = Path(directory) / "manifest.json"
            result = run_probe("inventory", "--state", str(state),
                               "--manifest", str(manifest_path))
            self.assert_probe_error(result, "has_diagnostics", manifest_path)

    @needs_node
    def test_error_initializer_missing(self):
        cases = {
            "no_initializer":
                "export function foo() { const a = 1; }\n",
            "empty_body":
                "export function $canInitialize() {}\n",
            "overload_without_body":
                "export function $canInitialize(): void;\n",
        }
        for name, text in cases.items():
            with self.subTest(name=name):
                with tempfile.TemporaryDirectory() as directory:
                    state = write_state(directory, "state.ts", text)
                    manifest_path = Path(directory) / "manifest.json"
                    result = run_probe(
                        "inventory", "--state", str(state),
                        "--manifest", str(manifest_path))
                    self.assert_probe_error(
                        result, "initializer_missing", manifest_path)

    @needs_node
    def test_error_initializer_ambiguous(self):
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(
                directory, "state.ts",
                "export function $canInitialize() { const a = 1; }\n"
                "export function $canInitialize() { const b = 2; }\n")
            manifest_path = Path(directory) / "manifest.json"
            result = run_probe("inventory", "--state", str(state),
                               "--manifest", str(manifest_path))
            self.assert_probe_error(
                result, "initializer_ambiguous", manifest_path)

    @needs_node
    def test_error_unsupported_statement(self):
        cases = {
            "let_binding":
                "export function $canInitialize() { let a = 1; }\n",
            "var_binding":
                "export function $canInitialize() { var a = 1; }\n",
            "control_flow":
                ("export function $canInitialize() { "
                 "if (true) { const a = 1; } }\n"),
            "destructuring":
                ("export function $canInitialize() { "
                 "const {a} = {a: 1}; }\n"),
        }
        for name, text in cases.items():
            with self.subTest(name=name):
                with tempfile.TemporaryDirectory() as directory:
                    state = write_state(directory, "state.ts", text)
                    manifest_path = Path(directory) / "manifest.json"
                    result = run_probe(
                        "inventory", "--state", str(state),
                        "--manifest", str(manifest_path))
                    self.assert_probe_error(
                        result, "unsupported_statement", manifest_path)

    @needs_node
    def test_error_async_expression(self):
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(
                directory, "state.ts",
                ("export function $canInitialize() { "
                 "const a = await foo(); }\n"))
            manifest_path = Path(directory) / "manifest.json"
            result = run_probe("inventory", "--state", str(state),
                               "--manifest", str(manifest_path))
            self.assert_probe_error(result, "async_expression", manifest_path)

    @needs_node
    def test_error_missing_semicolon(self):
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(
                directory, "state.ts",
                "export function $canInitialize() { const a = 1 }\n")
            manifest_path = Path(directory) / "manifest.json"
            result = run_probe("inventory", "--state", str(state),
                               "--manifest", str(manifest_path))
            self.assert_probe_error(
                result, "missing_semicolon", manifest_path)

    @needs_node
    def test_error_already_instrumented(self):
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(
                directory, "state.ts",
                ("export function $canInitialize() { const a = 1; }\n"
                 "const marker = \"__canStartupMark\";\n"))
            manifest_path = Path(directory) / "manifest.json"
            out_path = Path(directory) / "out.ts"
            result = run_probe(
                "instrument", "--state", str(state), "--out", str(out_path),
                "--manifest", str(manifest_path),
                "--probe-specifier", "./startup-probe.ts")
            self.assert_probe_error(
                result, "already_instrumented", manifest_path, out_path)

    @needs_node
    def test_error_io_error(self):
        with tempfile.TemporaryDirectory() as directory:
            missing = Path(directory) / "does-not-exist.ts"
            manifest_path = Path(directory) / "manifest.json"
            result = run_probe("inventory", "--state", str(missing),
                               "--manifest", str(manifest_path))
            self.assert_probe_error(result, "io_error", manifest_path)
            state = write_state(
                directory, "state.ts",
                "export function $canInitialize() { const a = 1; }\n")
            unwritable = (Path(directory) / "missing-dir"
                          / "manifest.json")
            result = run_probe("inventory", "--state", str(state),
                               "--manifest", str(unwritable))
            self.assert_probe_error(result, "io_error", unwritable)
            out_missing = Path(directory) / "missing-dir" / "out.ts"
            manifest2 = Path(directory) / "manifest2.json"
            result = run_probe(
                "instrument", "--state", str(state),
                "--out", str(out_missing),
                "--manifest", str(manifest2),
                "--probe-specifier", "./startup-probe.ts")
            self.assertEqual(result.returncode, 2, msg=result.stderr)

    @needs_node
    def test_error_writes_nothing_preserves_existing(self):
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(
                directory, "state.ts",
                "export function $canInitialize() { const a = ; }\n")
            manifest_path = Path(directory) / "manifest.json"
            manifest_path.write_text("sentinel")
            out_path = Path(directory) / "out.ts"
            result = run_probe(
                "instrument", "--state", str(state), "--out", str(out_path),
                "--manifest", str(manifest_path),
                "--probe-specifier", "./startup-probe.ts")
            self.assert_probe_error(result, "has_diagnostics", out_path)
            self.assertEqual(manifest_path.read_text(), "sentinel")

    @needs_node
    def test_usage_errors_exit_two_without_output(self):
        with tempfile.TemporaryDirectory() as directory:
            manifest_path = Path(directory) / "manifest.json"
            result = run_probe("bogus-command")
            self.assertEqual(result.returncode, 2, msg=result.stderr)
            self.assertFalse(manifest_path.exists())
            result = run_probe("inventory", "--state",
                               str(Path(directory) / "state.ts"))
            self.assertEqual(result.returncode, 2, msg=result.stderr)
            self.assertFalse(manifest_path.exists())


def measured(ms):
    return {"status": "measured", "ms": ms}


def skipped(status="skipped_no_metadata"):
    return {"status": status, "ms": None}


def stages_record(**overrides):
    record = {"import": measured(1.0),
              "diagnostics_import": skipped(),
              "configure": skipped(),
              "initialize": measured(2.0)}
    record.update(overrides)
    return record


class ValidateEventsTests(unittest.TestCase):
    def test_valid_returns_inclusive_costs(self):
        events = [[0, 0, 1.0], [0, 1, 2.5],
                  [1, 0, 3.0], [1, 1, 4.0]]
        self.assertEqual(startup.validate_events(events, 2), [1.5, 1.0])

    def test_valid_allows_zero_cost_and_int_ms(self):
        events = [[0, 0, 5], [0, 1, 5]]
        self.assertEqual(startup.validate_events(events, 1), [0.0])

    def test_invalid_count(self):
        good = [[0, 0, 1.0], [0, 1, 2.0]]
        for events in (good[:1], good + [[1, 0, 3.0]], [], None, {}, "x"):
            with self.subTest(events=events):
                with self.assertRaisesRegex(
                        ValueError, r"startup events invalid: count"):
                    startup.validate_events(events, 1)

    def test_invalid_index(self):
        cases = [
            [[1, 0, 1.0], [0, 1, 2.0]],
            [[-1, 0, 1.0], [0, 1, 2.0]],
            [["0", 0, 1.0], [0, 1, 2.0]],
            [[True, 0, 1.0], [0, 1, 2.0]],
            [[[0], 0, 1.0], [0, 1, 2.0]],
            [[0, 0], [0, 1, 2.0]],
            ["not-a-tuple", [0, 1, 2.0]],
        ]
        for events in cases:
            with self.subTest(events=events):
                with self.assertRaisesRegex(
                        ValueError, r"startup events invalid: index"):
                    startup.validate_events(events, 1)

    def test_invalid_phase(self):
        for phase in (2, -1, True, "0", None, 0.5):
            with self.subTest(phase=phase):
                with self.assertRaisesRegex(
                        ValueError, r"startup events invalid: phase"):
                    startup.validate_events(
                        [[0, phase, 1.0], [0, 1, 2.0]], 1)

    def test_invalid_order(self):
        cases = [
            [[0, 1, 1.0], [0, 0, 2.0]],
            [[1, 0, 1.0], [1, 1, 2.0], [0, 0, 3.0], [0, 1, 4.0]],
            [[0, 0, 1.0], [1, 1, 2.0], [1, 0, 3.0], [0, 1, 4.0]],
        ]
        for events in cases:
            with self.subTest(events=events):
                with self.assertRaisesRegex(
                        ValueError, r"startup events invalid: order"):
                    startup.validate_events(events, len(events) // 2)

    def test_invalid_time(self):
        with self.assertRaisesRegex(
                ValueError, r"startup events invalid: time"):
            startup.validate_events([[0, 0, "1"], [0, 1, 2.0]], 1)
        with self.assertRaisesRegex(
                ValueError, r"startup events invalid: time"):
            startup.validate_events([[0, 0, True], [0, 1, 2.0]], 1)
        with self.assertRaisesRegex(
                ValueError, r"startup events invalid: time"):
            startup.validate_events([[0, 0, 5.0], [0, 1, 3.0]], 1)


class AcceptLaunchTests(unittest.TestCase):
    def test_accept_ordinary(self):
        stdout = json.dumps({"status": "ok",
                             "stages": stages_record()}) + "\n"
        accepted = startup.accept_launch(0, stdout, "ordinary", 2)
        self.assertEqual(accepted["stages"]["initialize"]["ms"], 2.0)
        self.assertNotIn("statement_ms", accepted)
        self.assertNotIn("events", accepted)

    def test_accept_diagnostic(self):
        events = [[0, 0, 1.0], [0, 1, 2.0]]
        stdout = json.dumps({"status": "ok", "stages": stages_record(),
                             "events": events}) + "\n"
        accepted = startup.accept_launch(0, stdout, "diagnostic", 1)
        self.assertEqual(accepted["statement_ms"], [1.0])
        self.assertEqual(accepted["events"], events)

    def test_reject_exit(self):
        stdout = json.dumps({"status": "ok",
                             "stages": stages_record()}) + "\n"
        with self.assertRaisesRegex(ValueError, r"launch rejected: exit"):
            startup.accept_launch(1, stdout, "ordinary", 1)

    def test_reject_output_line_count(self):
        for stdout in ("", "\n",
                       "{}\n{}\n",
                       json.dumps({"status": "ok"}) + "\n\n"):
            with self.subTest(stdout=stdout):
                with self.assertRaisesRegex(
                        ValueError, r"launch rejected: output"):
                    startup.accept_launch(0, stdout, "ordinary", 1)

    def test_reject_json(self):
        with self.assertRaisesRegex(ValueError, r"launch rejected: json"):
            startup.accept_launch(0, "not json\n", "ordinary", 1)

    def test_reject_status(self):
        for record in ({"status": "failed", "stages": stages_record()},
                       {"stages": stages_record()},
                       ["ok"],
                       "ok"):
            with self.subTest(record=record):
                with self.assertRaisesRegex(
                        ValueError, r"launch rejected: status"):
                    startup.accept_launch(
                        0, json.dumps(record) + "\n", "ordinary", 1)

    def test_reject_stages_malformed(self):
        base = stages_record()
        missing = {key: base[key] for key in base if key != "import"}
        extra = dict(base)
        extra["extra"] = measured(1.0)
        cases = {
            "missing_stages": {"status": "ok"},
            "missing_key": {"status": "ok", "stages": missing},
            "extra_key": {"status": "ok", "stages": extra},
            "bad_status": {"status": "ok", "stages": stages_record(
                **{"import": {"status": "bogus", "ms": None}})},
            "measured_without_ms": {"status": "ok", "stages": stages_record(
                **{"import": {"status": "measured"}})},
            "measured_bool_ms": {"status": "ok", "stages": stages_record(
                **{"import": {"status": "measured", "ms": True}})},
            "skipped_with_ms": {"status": "ok", "stages": stages_record(
                **{"configure": {"status": "skipped_no_metadata",
                                "ms": 1.0}})},
            "stages_not_dict": {"status": "ok", "stages": []},
        }
        for name, record in cases.items():
            with self.subTest(name=name):
                with self.assertRaisesRegex(
                        ValueError, r"launch rejected: output"):
                    startup.accept_launch(
                        0, json.dumps(record) + "\n", "ordinary", 1)

    def test_reject_unexpected_events(self):
        stdout = json.dumps({"status": "ok", "stages": stages_record(),
                             "events": []}) + "\n"
        with self.assertRaisesRegex(
                ValueError, r"launch rejected: unexpected_events"):
            startup.accept_launch(0, stdout, "ordinary", 0)

    def test_reject_diagnostic_invalid_events(self):
        cases = {
            "count": [[0, 0, 1.0]],
            "order": [[0, 1, 1.0], [0, 0, 2.0]],
            "index": [[9, 0, 1.0], [9, 1, 2.0]],
            "phase": [[0, 0, 1.0], [0, 7, 2.0]],
            "time": [[0, 0, 5.0], [0, 1, 1.0]],
        }
        for code, events in cases.items():
            with self.subTest(code=code):
                stdout = json.dumps({"status": "ok",
                                     "stages": stages_record(),
                                     "events": events}) + "\n"
                with self.assertRaisesRegex(ValueError, code):
                    startup.accept_launch(0, stdout, "diagnostic", 1)

    def test_reject_diagnostic_missing_events(self):
        stdout = json.dumps({"status": "ok",
                             "stages": stages_record()}) + "\n"
        with self.assertRaisesRegex(ValueError, r"count"):
            startup.accept_launch(0, stdout, "diagnostic", 1)


def manifest_entry(index, start, end, text, kind="VariableStatement",
                   binding="a", factories=None):
    span = text[start:end]
    return {"index": index, "start": start, "end": end,
            "bytes": len(span.encode()),
            "sha256": hashlib.sha256(span.encode()).hexdigest(),
            "kind": kind, "binding": binding,
            "factories": [] if factories is None else factories}


def good_manifest(text):
    first = text.index(";") + 1
    second_start = first + 1
    second = text.index(";", second_start) + 1
    return {
        "schema": "can.startup-probe-manifest/1",
        "state_path": "/tmp/state.ts",
        "state_sha256": hashlib.sha256(text.encode()).hexdigest(),
        "state_bytes": len(text.encode()),
        "initializer": {"name": "$canInitialize", "start": 0,
                        "end": second, "statement_count": 2},
        "probe_specifier": None, "instrumented_sha256": None,
        "statements": [
            manifest_entry(0, 0, first, text),
            manifest_entry(1, second_start, second, text,
                           kind="ExpressionStatement",
                           binding="call:go", factories=["go"]),
        ],
    }


class ManifestCoverageTests(unittest.TestCase):
    TEXT = "const a = 1;\ngo(a);\n"

    def test_valid_returns_true(self):
        self.assertTrue(startup.check_manifest_coverage(
            good_manifest(self.TEXT), self.TEXT))

    def test_structural_problems_map_to_count(self):
        text = self.TEXT
        good = good_manifest(text)
        cases = {
            "not_dict": ([], text),
            "bad_schema": ({**good, "schema": "other"}, text),
            "missing_initializer": (
                {key: good[key] for key in good
                 if key != "initializer"}, text),
            "missing_statements": (
                {key: good[key] for key in good
                 if key != "statements"}, text),
            "count_mismatch": (
                {**good, "initializer": {"name": "$canInitialize",
                                        "start": 0, "end": 1,
                                        "statement_count": 5}}, text),
            "zero_count": (
                {**good, "initializer": {"name": "$canInitialize",
                                        "start": 0, "end": 1,
                                        "statement_count": 0},
                 "statements": []}, text),
        }
        for name, (manifest, state) in cases.items():
            with self.subTest(name=name):
                with self.assertRaisesRegex(
                        ValueError, r"manifest coverage invalid: count"):
                    startup.check_manifest_coverage(manifest, state)

    def test_invalid_order(self):
        text = self.TEXT
        bad_index = good_manifest(text)
        bad_index["statements"][1]["index"] = 5
        with self.assertRaisesRegex(
                ValueError, r"manifest coverage invalid: order"):
            startup.check_manifest_coverage(bad_index, text)
        first = text.index(";") + 1
        second = text.index(";", first) + 1
        reversed_manifest = good_manifest(text)
        reversed_manifest["statements"] = [
            manifest_entry(0, first, second, text),
            manifest_entry(1, 0, first, text),
        ]
        with self.assertRaisesRegex(
                ValueError, r"manifest coverage invalid: order"):
            startup.check_manifest_coverage(reversed_manifest, text)

    def test_invalid_span(self):
        text = self.TEXT
        first = text.index(";") + 1
        over = good_manifest(text)
        over["statements"][1]["end"] = len(text) + 10
        with self.assertRaisesRegex(
                ValueError, r"manifest coverage invalid: span"):
            startup.check_manifest_coverage(over, text)
        empty = good_manifest(text)
        empty["statements"][0]["start"] = 0
        empty["statements"][0]["end"] = 0
        with self.assertRaisesRegex(
                ValueError, r"manifest coverage invalid: span"):
            startup.check_manifest_coverage(empty, text)

    def test_invalid_kind(self):
        text = self.TEXT
        bad = good_manifest(text)
        bad["statements"][0]["kind"] = "IfStatement"
        with self.assertRaisesRegex(
                ValueError, r"manifest coverage invalid: kind"):
            startup.check_manifest_coverage(bad, text)

    def test_invalid_hash(self):
        text = self.TEXT
        bad_sha = good_manifest(text)
        bad_sha["statements"][0]["sha256"] = "0" * 64
        with self.assertRaisesRegex(
                ValueError, r"manifest coverage invalid: hash"):
            startup.check_manifest_coverage(bad_sha, text)
        bad_bytes = good_manifest(text)
        bad_bytes["statements"][1]["bytes"] += 1
        with self.assertRaisesRegex(
                ValueError, r"manifest coverage invalid: hash"):
            startup.check_manifest_coverage(bad_bytes, text)


class ArgsMathTests(unittest.TestCase):
    def test_parse_args_defaults_to_validate_only(self):
        args = startup.parse_args(["--repo", "/r", "--output", "/o"])
        self.assertFalse(args.measure)
        self.assertFalse(args.validate_only)
        self.assertFalse(args.trial_run)
        self.assertEqual(
            (args.trials, args.warmups, args.batches,
             args.child_timeout, args.sampling_bound),
            (6, 2, 7, 60, 300))
        self.assertEqual(args.scratch_parent, Path(tempfile.gettempdir()))
        self.assertIsNone(args.prepared_out)
        self.assertIsNone(args.trial_index)
        self.assertIsNone(args.prepared)
        self.assertIsNone(args.rows)

    def test_parse_args_flags(self):
        args = startup.parse_args(
            ["--repo", "/r", "--output", "/o", "--measure",
             "--trials", "3", "--warmups", "1", "--batches", "2",
             "--child-timeout", "9", "--sampling-bound", "11",
             "--scratch-parent", "/tmp/x"])
        self.assertTrue(args.measure)
        self.assertEqual((args.trials, args.warmups, args.batches),
                         (3, 1, 2))
        self.assertEqual(args.child_timeout, 9)
        self.assertEqual(args.sampling_bound, 11)
        self.assertEqual(args.scratch_parent, Path("/tmp/x"))
        trial = startup.parse_args(
            ["--repo", "/r", "--trial-run", "--trial-index", "4",
             "--prepared", "/p", "--rows", "/rows"])
        self.assertTrue(trial.trial_run)
        self.assertEqual(trial.trial_index, 4)
        prepared = startup.parse_args(
            ["--repo", "/r", "--output", "/o", "--validate-only",
             "--prepared-out", "/keep"])
        self.assertTrue(prepared.validate_only)
        self.assertEqual(prepared.prepared_out, Path("/keep"))

    def test_parse_args_requires_repo(self):
        with contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit):
                startup.parse_args(["--output", "/o"])

    def test_trial_order_rotation(self):
        self.assertEqual(
            startup.trial_order(0),
            ["ordinary-modules", "ordinary-bundle", "minimal",
             "diagnostic-modules"])
        self.assertEqual(
            startup.trial_order(1),
            ["ordinary-bundle", "minimal", "ordinary-modules",
             "diagnostic-modules"])
        self.assertEqual(
            startup.trial_order(2),
            ["minimal", "ordinary-modules", "ordinary-bundle",
             "diagnostic-modules"])
        for trial in range(6):
            order = startup.trial_order(trial)
            self.assertEqual(order[-1], "diagnostic-modules")
            self.assertEqual(order, startup.trial_order(trial + 3))
            self.assertEqual(set(order[:3]),
                             {"ordinary-modules", "ordinary-bundle",
                              "minimal"})

    def test_median(self):
        self.assertEqual(startup.median([3, 1, 2]), 2.0)
        self.assertEqual(startup.median([4, 1, 3, 2]), 2.5)
        self.assertEqual(startup.median([7]), 7.0)
        self.assertAlmostEqual(startup.median([0.1, 0.3, 0.2]), 0.2)

    def test_median_empty_refuses(self):
        with self.assertRaisesRegex(ValueError, "at least one sample"):
            startup.median([])

    def test_mad(self):
        self.assertEqual(startup.mad([1, 2, 3]), 1.0)
        self.assertEqual(startup.mad([5]), 0.0)
        self.assertEqual(startup.mad([1, 2, 3], center=2.0), 1.0)
        self.assertEqual(startup.mad([1, 2, 3], center=1.0), 1.0)

    def test_mad_empty_refuses(self):
        with self.assertRaisesRegex(ValueError, "at least one sample"):
            startup.mad([])

    def test_constants_frozen(self):
        self.assertEqual(startup.PROFILES,
                         ("ordinary-modules", "ordinary-bundle", "minimal",
                          "diagnostic-modules"))
        self.assertEqual(startup.ORDINARY_PROFILES,
                         ("ordinary-modules", "ordinary-bundle", "minimal"))
        self.assertEqual(startup.DIAGNOSTIC_PROFILE, "diagnostic-modules")
        self.assertEqual(startup.STAGES,
                         ("import", "diagnostics_import", "configure",
                          "initialize"))
        self.assertEqual(startup.STATUSES,
                         ("measured", "skipped_no_metadata",
                          "not_applicable_bundled"))
        self.assertEqual(startup.STATEMENT_MARK, "__canStartupMark")
        self.assertEqual(startup.EVENTS_KEY, "__canStartupEvents")
        self.assertEqual(startup.PROBE_SPECIFIER, "./startup-probe.ts")
        self.assertEqual(startup.SCHEMA, "can.startup-attribution/1")
        self.assertEqual(startup.PROBE_SCHEMA,
                         "can.startup-probe-manifest/1")
        self.assertEqual(startup.CAP_BYTES, 64 * 1024 * 1024)
        self.assertEqual(startup.CAP_FILES, 500)

    def test_driver_globals_patchable(self):
        self.assertTrue(hasattr(startup, "driver_command"))
        self.assertTrue(hasattr(startup, "driver_adapter"))
        previous_command, previous_adapter = (
            startup.driver_command, startup.driver_adapter)
        try:
            startup.driver_command = object()
            startup.driver_adapter = object()
            self.assertIsNotNone(startup.driver_command)
        finally:
            startup.driver_command = previous_command
            startup.driver_adapter = previous_adapter

    def test_bind_driver_loads_existing_helpers(self):
        previous_command, previous_adapter = (
            startup.driver_command, startup.driver_adapter)
        try:
            startup.bind_driver(REPO)
            self.assertTrue(callable(startup.driver_command))
            self.assertTrue(callable(startup.driver_adapter))
        finally:
            startup.driver_command = previous_command
            startup.driver_adapter = previous_adapter


class CapsTests(unittest.TestCase):
    def test_under_caps_passes(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "a.js").write_bytes(b"console.log(1);\n")
            (root / "sub").mkdir()
            (root / "sub" / "b.ts").write_bytes(b"export const x = 1;\n")
            total, compiled = startup.check_caps(root)
            self.assertEqual(compiled, 1)
            self.assertEqual(total, (root / "a.js").stat().st_size
                             + (root / "sub" / "b.ts").stat().st_size)

    def test_over_bytes_refuses(self):
        with tempfile.TemporaryDirectory() as directory:
            big = Path(directory) / "big.js"
            with open(big, "wb") as handle:
                handle.truncate(startup.CAP_BYTES + 1)
            with self.assertRaisesRegex(ValueError, "scratch cap exceeded"):
                startup.check_caps(directory)

    def test_exact_bytes_cap_passes(self):
        with tempfile.TemporaryDirectory() as directory:
            edge = Path(directory) / "edge.bin"
            with open(edge, "wb") as handle:
                handle.truncate(startup.CAP_BYTES)
            total, compiled = startup.check_caps(directory)
            self.assertEqual(total, startup.CAP_BYTES)
            self.assertEqual(compiled, 0)

    def test_over_files_refuses(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for index in range(startup.CAP_FILES + 1):
                (root / f"m{index}.js").write_bytes(b"x")
            with self.assertRaisesRegex(ValueError, "scratch cap exceeded"):
                startup.check_caps(root)

    def test_exact_file_cap_passes(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for index in range(startup.CAP_FILES):
                (root / f"m{index}.js").write_bytes(b"x")
            total, compiled = startup.check_caps(root)
            self.assertEqual(compiled, startup.CAP_FILES)

    def test_symlink_js_not_counted(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            real = root / "real.js"
            real.write_bytes(b"x")
            (root / "link.js").symlink_to(real)
            (root / "note.txt").write_bytes(b"y")
            total, compiled = startup.check_caps(root)
            self.assertEqual(compiled, 1)
            self.assertGreaterEqual(total, 2)


class ScratchTests(unittest.TestCase):
    def test_make_cleanup_roundtrip(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.assertTrue(scratch.is_dir())
            self.assertEqual(scratch.parent, Path(parent))
            self.assertTrue(scratch.name.startswith(startup.SCRATCH_PREFIX))
            (scratch / "work.txt").write_text("x")
            self.assertTrue(startup.cleanup_scratch(scratch))
            self.assertFalse(scratch.exists())

    def test_cleanup_none_and_missing_returns_false(self):
        self.assertFalse(startup.cleanup_scratch(None))
        with tempfile.TemporaryDirectory() as parent:
            missing = Path(parent) / "never-created"
            self.assertFalse(startup.cleanup_scratch(missing))

    def test_cleanup_foreign_refuses(self):
        with tempfile.TemporaryDirectory() as parent:
            foreign = Path(parent) / "foreign-dir"
            foreign.mkdir()
            (foreign / "keep.txt").write_text("x")
            self.addCleanup(shutil.rmtree, foreign, True)
            with self.assertRaisesRegex(ValueError, "scratch ownership invalid"):
                startup.cleanup_scratch(foreign)
            self.assertTrue((foreign / "keep.txt").exists())

    def test_owned_roundtrip_cleans(self):
        with tempfile.TemporaryDirectory() as parent:
            with startup.owned_scratch(parent) as scratch:
                self.assertTrue(scratch.is_dir())
                (scratch / "work.txt").write_text("x")
            self.assertFalse(scratch.exists())

    def test_owned_exception_cleans(self):
        with tempfile.TemporaryDirectory() as parent:
            with self.assertRaises(RuntimeError):
                with startup.owned_scratch(parent) as scratch:
                    (scratch / "work.txt").write_text("x")
                    raise RuntimeError("boom")
            self.assertFalse(scratch.exists())

    def test_owned_keyboard_interrupt_cleans(self):
        with tempfile.TemporaryDirectory() as parent:
            with self.assertRaises(KeyboardInterrupt):
                with startup.owned_scratch(parent) as scratch:
                    raise KeyboardInterrupt()
            self.assertFalse(scratch.exists())

    def test_owned_keep_preserves_on_success(self):
        with tempfile.TemporaryDirectory() as parent:
            with startup.owned_scratch(parent, keep=True) as scratch:
                (scratch / "kept.txt").write_text("x")
            self.assertTrue((scratch / "kept.txt").exists())
            self.assertTrue(startup.cleanup_scratch(scratch))
            self.assertFalse(scratch.exists())

    def test_owned_keep_cleans_on_failure(self):
        with tempfile.TemporaryDirectory() as parent:
            with self.assertRaises(RuntimeError):
                with startup.owned_scratch(parent, keep=True) as scratch:
                    raise RuntimeError("boom")
            self.assertFalse(scratch.exists())

    def test_owned_explicit_path(self):
        with tempfile.TemporaryDirectory() as parent:
            explicit = Path(parent) / "prepared-keep"
            with startup.owned_scratch(parent, path=explicit,
                                       keep=True) as scratch:
                self.assertEqual(scratch, explicit)
            self.assertTrue(explicit.is_dir())
            self.assertTrue(startup.cleanup_scratch(explicit))
            clash = Path(parent) / "clash"
            clash.mkdir()
            with self.assertRaisesRegex(ValueError, "already exists"):
                with startup.owned_scratch(parent, path=clash):
                    pass


def make_fake_runtime(root):
    runtime = Path(root) / "runtime"
    (runtime / "nested").mkdir(parents=True)
    (runtime / "a.ts").write_text("export const a = 1;\n")
    (runtime / "nested" / "b.ts").write_text("export const b = 2;\n")
    (runtime / "ignore.js").write_text("console.log(1);\n")
    return runtime


class LinkHelperTests(unittest.TestCase):
    def test_mirror_verify_replace_roundtrip(self):
        with tempfile.TemporaryDirectory() as directory:
            repo_runtime = make_fake_runtime(Path(directory) / "repo")
            inventory = Path(directory) / "inventory"
            mirrored = startup.mirror_runtime_inventory(
                repo_runtime, inventory)
            self.assertEqual(mirrored, 2)
            self.assertTrue((inventory / "a.ts").exists())
            self.assertTrue((inventory / "nested" / "b.ts").exists())
            self.assertFalse((inventory / "ignore.js").exists())
            for path in inventory.rglob("*.ts"):
                self.assertEqual(path.stat().st_size, 0)
            generated = Path(directory) / "generated"
            placeholders = generated / "runtime"
            startup.mirror_runtime_inventory(repo_runtime, placeholders)
            self.assertEqual(
                startup.verify_placeholders(placeholders, 2), 2)
            startup.replace_runtime_link(generated, repo_runtime)
            self.assertTrue(placeholders.is_symlink())
            self.assertEqual(placeholders.resolve(), repo_runtime.resolve())

    def test_mirror_empty_refuses(self):
        with tempfile.TemporaryDirectory() as directory:
            empty = Path(directory) / "runtime"
            empty.mkdir()
            (empty / "note.txt").write_text("x")
            with self.assertRaisesRegex(ValueError, "inventory empty"):
                startup.mirror_runtime_inventory(
                    empty, Path(directory) / "out")

    def test_verify_count_mismatch_refuses(self):
        with tempfile.TemporaryDirectory() as directory:
            repo_runtime = make_fake_runtime(Path(directory) / "repo")
            placeholders = Path(directory) / "placeholders"
            startup.mirror_runtime_inventory(repo_runtime, placeholders)
            with self.assertRaisesRegex(ValueError, "placeholder count"):
                startup.verify_placeholders(placeholders, 99)

    def test_verify_nonzero_refuses(self):
        with tempfile.TemporaryDirectory() as directory:
            repo_runtime = make_fake_runtime(Path(directory) / "repo")
            placeholders = Path(directory) / "placeholders"
            startup.mirror_runtime_inventory(repo_runtime, placeholders)
            (placeholders / "a.ts").write_text("compiled")
            with self.assertRaisesRegex(ValueError, "zero-byte"):
                startup.verify_placeholders(placeholders, 2)

    def test_replace_missing_refuses(self):
        with tempfile.TemporaryDirectory() as directory:
            repo_runtime = make_fake_runtime(Path(directory) / "repo")
            generated = Path(directory) / "generated"
            generated.mkdir()
            with self.assertRaisesRegex(ValueError, "placeholders missing"):
                startup.replace_runtime_link(generated, repo_runtime)
            link = generated / "runtime"
            link.symlink_to(repo_runtime, target_is_directory=True)
            with self.assertRaisesRegex(ValueError, "placeholders missing"):
                startup.replace_runtime_link(generated, repo_runtime)

    def test_install_node_modules_roundtrip(self):
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory) / "repo"
            (repo / "node_modules" / "pkg").mkdir(parents=True)
            work = Path(directory) / "work"
            work.mkdir()
            startup.install_node_modules_link(repo, work)
            target = work / "node_modules"
            self.assertTrue(target.is_symlink())
            self.assertEqual(target.resolve(),
                             (repo / "node_modules").resolve())
            startup.install_node_modules_link(repo, work)
            self.assertTrue(target.is_symlink())

    def test_install_node_modules_missing_refuses(self):
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory) / "repo"
            repo.mkdir()
            with self.assertRaises(FileNotFoundError):
                startup.install_node_modules_link(
                    repo, Path(directory) / "work")


class WriterTests(unittest.TestCase):
    def test_facade_content(self):
        with tempfile.TemporaryDirectory() as directory:
            dest = Path(directory) / "startup-facade.ts"
            startup.write_facade(dest)
            text = dest.read_text()
            self.assertIn(
                'export {$canInitialize as initialize}'
                ' from "./program/state.ts";', text)
            for oracle in ("doubled", "frequency", "sum", "generic_pass",
                           "captured_map"):
                self.assertIn(oracle, text)
            self.assertIn("verify", text)
            self.assertIn("./runtime/data.ts", text)
            self.assertIn("./runtime/completion.ts", text)
            self.assertIn("Object.isFrozen", text)
            self.assertIn("input", text)

    def test_minimal_content(self):
        with tempfile.TemporaryDirectory() as directory:
            dest = Path(directory) / "startup-minimal.ts"
            startup.write_minimal(dest)
            text = dest.read_text()
            self.assertIn("initialize", text)
            self.assertIn("verify", text)
            self.assertNotIn("import", text)

    def test_probe_content(self):
        with tempfile.TemporaryDirectory() as directory:
            dest = Path(directory) / "startup-probe.ts"
            startup.write_probe_module(dest)
            text = dest.read_text()
            self.assertIn("__canStartupMark", text)
            self.assertIn("__canStartupEvents", text)
            self.assertIn("performance.now()", text)
            self.assertIn("export", text)
            self.assertIn("globalThis", text)

    def test_launcher_content(self):
        with tempfile.TemporaryDirectory() as directory:
            dest = Path(directory) / "startup-launcher.js"
            startup.write_launcher(dest)
            text = dest.read_text()
            for line in text.splitlines():
                self.assertIsNone(
                    re.match(r"\s*import[\s{*\"]", line),
                    msg=f"static import: {line}")
            for flag in ("--root", "--mode", "--diagnostics",
                         "--diag-module", "--metadata-root", "--entry-url",
                         "--statements"):
                self.assertIn(flag, text)
            self.assertIn("import(", text)
            self.assertIn("root.initialize();", text)
            self.assertNotIn("await root.initialize", text)
            self.assertIn("await root.verify()", text)
            self.assertIn('JSON.stringify(record)', text)
            self.assertIn('"ok"', text)


class MainUsageTests(unittest.TestCase):
    def run_main(self, argv):
        with contextlib.redirect_stderr(io.StringIO()) as errors:
            code = startup.main(argv)
        return code, errors.getvalue()

    def test_conflicting_measure_validate(self):
        with tempfile.TemporaryDirectory() as directory:
            output = str(Path(directory) / "out.json")
            code, _ = self.run_main(
                ["--repo", str(REPO), "--output", output,
                 "--measure", "--validate-only"])
            self.assertEqual(code, 2)
            self.assertFalse(Path(output).exists())

    def test_prepared_out_with_measure(self):
        with tempfile.TemporaryDirectory() as directory:
            output = str(Path(directory) / "out.json")
            code, _ = self.run_main(
                ["--repo", str(REPO), "--output", output,
                 "--measure", "--prepared-out",
                 str(Path(directory) / "keep")])
            self.assertEqual(code, 2)
            self.assertFalse(Path(output).exists())

    def test_missing_output(self):
        code, _ = self.run_main(["--repo", str(REPO)])
        self.assertEqual(code, 2)

    def test_bad_ranges(self):
        with tempfile.TemporaryDirectory() as directory:
            output = str(Path(directory) / "out.json")
            for extra in (["--trials", "0"], ["--batches", "0"],
                          ["--warmups", "-1"], ["--child-timeout", "0"],
                          ["--sampling-bound", "0"]):
                with self.subTest(extra=extra):
                    code, _ = self.run_main(
                        ["--repo", str(REPO), "--output", output] + extra)
                    self.assertEqual(code, 2)

    def test_trial_run_validation(self):
        with tempfile.TemporaryDirectory() as directory:
            base = ["--repo", str(REPO), "--trial-run"]
            code, _ = self.run_main(base)
            self.assertEqual(code, 2)
            output = str(Path(directory) / "out.json")
            code, _ = self.run_main(
                base + ["--trial-index", "0", "--prepared",
                        str(directory), "--rows",
                        str(Path(directory) / "rows.json"),
                        "--output", output])
            self.assertEqual(code, 2)

    def test_bad_repo_driver(self):
        with tempfile.TemporaryDirectory() as directory:
            output = str(Path(directory) / "out.json")
            code, errors = self.run_main(
                ["--repo", directory, "--output", output])
            self.assertEqual(code, 2)
            self.assertIn("runtime driver", errors)
            self.assertFalse(Path(output).exists())


class ActualStateTests(unittest.TestCase):
    @unittest.skipUnless(os.environ.get("CAN_STARTUP_ACTUAL_STATE"),
                         "set CAN_STARTUP_ACTUAL_STATE to emitted "
                         "program/state.ts")
    @needs_node
    def test_actual_emitted_state_inventory(self):
        state = Path(os.environ["CAN_STARTUP_ACTUAL_STATE"]).resolve()
        expected = int(os.environ.get("CAN_STARTUP_ACTUAL_STATEMENTS",
                                      "57"))
        self.assertTrue(state.is_file(), msg=str(state))
        with tempfile.TemporaryDirectory() as directory:
            manifest_path = Path(directory) / "manifest.json"
            result = run_probe("inventory", "--state", str(state),
                               "--manifest", str(manifest_path),
                               timeout=180)
            self.assertEqual(result.returncode, 0, msg=result.stderr)
            manifest = json.loads(manifest_path.read_text())
            self.assertEqual(
                manifest["initializer"]["statement_count"], expected)
            self.assertTrue(startup.check_manifest_coverage(
                manifest, state.read_text()))


def make_row(trial, batch, profile, warmup, wall=1.0, stages=None,
             statement_ms=None):
    row = {"trial": trial, "batch": batch, "profile": profile,
           "warmup": warmup, "parent_wall_ms": wall,
           "stages": stages if stages is not None else stages_record()}
    if statement_ms is not None:
        row["statement_ms"] = statement_ms
    return row


def complete_membership_rows(trials=1, warmups=1, batches=2, full=False,
                             statement_count=2):
    rows = []
    for trial in range(trials):
        for profile in startup.PROFILES:
            for batch in range(warmups + batches):
                warmup = batch < warmups
                if full:
                    costs = ([0.5] * statement_count
                             if profile == startup.DIAGNOSTIC_PROFILE
                             else None)
                    rows.append(make_row(trial, batch, profile, warmup,
                                         wall=1.0 + batch,
                                         statement_ms=costs))
                else:
                    rows.append({"trial": trial, "batch": batch,
                                 "profile": profile, "warmup": warmup})
    return rows


class AcceptLaunchFiniteTests(unittest.TestCase):
    def launch_json(self, stages):
        return json.dumps({"status": "ok", "stages": stages}) + "\n"

    def test_reject_negative_ms(self):
        for stage in ("import", "initialize",
                      "diagnostics_import", "configure"):
            stages = stages_record()
            stages[stage] = measured(-1.0)
            with self.subTest(stage=stage):
                with self.assertRaisesRegex(
                        ValueError,
                        r"launch rejected: output.*finite milliseconds"):
                    startup.accept_launch(
                        0, self.launch_json(stages), "ordinary", 1)

    def test_reject_nan_ms(self):
        stages = stages_record(**{"import": measured(float("nan"))})
        with self.assertRaisesRegex(
                ValueError,
                r"launch rejected: output.*finite milliseconds"):
            startup.accept_launch(0, self.launch_json(stages),
                                  "ordinary", 1)

    def test_reject_infinite_ms(self):
        for value in (float("inf"), float("-inf")):
            with self.subTest(value=value):
                stages = stages_record(
                    **{"initialize": measured(value)})
                with self.assertRaisesRegex(
                        ValueError,
                        r"launch rejected: output.*finite milliseconds"):
                    startup.accept_launch(0, self.launch_json(stages),
                                          "ordinary", 1)

    def test_reject_measured_without_ms_says_finite(self):
        for item in ({"status": "measured"},
                     {"status": "measured", "ms": None}):
            with self.subTest(item=item):
                stages = stages_record(**{"import": dict(item)})
                with self.assertRaisesRegex(
                        ValueError,
                        r"launch rejected: output.*finite milliseconds"):
                    startup.accept_launch(0, self.launch_json(stages),
                                          "ordinary", 1)

    def test_reject_unmeasured_with_ms(self):
        stages = stages_record(
            **{"configure": {"status": "skipped_no_metadata",
                            "ms": 1.0}})
        with self.assertRaisesRegex(ValueError,
                                    r"launch rejected: output"):
            startup.accept_launch(0, self.launch_json(stages),
                                  "ordinary", 1,
                                  profile="ordinary-modules")

    def test_reject_profile_status_mismatch(self):
        bundled = {"status": "not_applicable_bundled", "ms": None}
        cases = [
            ("ordinary-bundle accepts only bundled",
             "ordinary-bundle", stages_record()),
            ("ordinary-bundle rejects measured pair",
             "ordinary-bundle",
             stages_record(diagnostics_import=measured(0.5),
                           configure=measured(0.6))),
            ("minimal accepts only skipped",
             "minimal",
             stages_record(diagnostics_import=dict(bundled),
                           configure=dict(bundled))),
            ("minimal rejects measured pair",
             "minimal",
             stages_record(diagnostics_import=measured(0.5),
                           configure=measured(0.6))),
            ("ordinary-modules rejects mixed pair",
             "ordinary-modules",
             stages_record(diagnostics_import=measured(0.5),
                           configure=skipped())),
            ("ordinary-modules rejects bundled pair",
             "ordinary-modules",
             stages_record(diagnostics_import=dict(bundled),
                           configure=dict(bundled))),
            ("diagnostic-modules rejects mixed pair",
             "diagnostic-modules",
             stages_record(diagnostics_import=skipped(),
                           configure=measured(0.6))),
            ("import must stay measured",
             "ordinary-modules",
             stages_record(**{"import": skipped()})),
            ("initialize must stay measured",
             "minimal",
             stages_record(**{"initialize": skipped()})),
        ]
        for name, profile, stages in cases:
            with self.subTest(name=name):
                with self.assertRaisesRegex(
                        ValueError,
                        rf"launch rejected: output.*profile {profile}"):
                    startup.accept_launch(0, self.launch_json(stages),
                                          "ordinary", 1, profile=profile)

    def test_reject_unknown_profile(self):
        with self.assertRaisesRegex(
                ValueError, r"launch rejected: output.*unknown profile"):
            startup.accept_launch(0, self.launch_json(stages_record()),
                                  "ordinary", 1, profile="bogus-profile")

    def test_accept_profile_valid(self):
        bundled = {"status": "not_applicable_bundled", "ms": None}
        bundle_stages = stages_record(
            diagnostics_import=dict(bundled),
            configure=dict(bundled))
        measured_pair = stages_record(
            diagnostics_import=measured(0.5),
            configure=measured(0.6))
        cases = [
            ("ordinary-modules", stages_record()),
            ("ordinary-modules", measured_pair),
            ("ordinary-bundle", bundle_stages),
            ("minimal", stages_record()),
            ("diagnostic-modules", stages_record()),
        ]
        for profile, stages in cases:
            with self.subTest(profile=profile):
                accepted = startup.accept_launch(
                    0, self.launch_json(stages), "ordinary", 1,
                    profile=profile)
                self.assertIn("stages", accepted)


class ValidateEventsFiniteTests(unittest.TestCase):
    def test_reject_nan_time(self):
        with self.assertRaisesRegex(
                ValueError, r"startup events invalid: time"):
            startup.validate_events(
                [[0, 0, float("nan")], [0, 1, 2.0]], 1)

    def test_reject_inf_time(self):
        for stamp in (float("inf"), float("-inf")):
            with self.subTest(stamp=stamp):
                with self.assertRaisesRegex(
                        ValueError, r"startup events invalid: time"):
                    startup.validate_events(
                        [[0, 0, 1.0], [0, 1, stamp]], 1)

    def test_reject_negative_time(self):
        with self.assertRaisesRegex(
                ValueError, r"startup events invalid: time"):
            startup.validate_events([[0, 0, -1.0], [0, 1, 0.0]], 1)

    def test_reject_globally_decreasing_across_statements(self):
        events = [[0, 0, 1.0], [0, 1, 3.0],
                  [1, 0, 2.0], [1, 1, 4.0]]
        with self.assertRaisesRegex(
                ValueError, r"startup events invalid: time"):
            startup.validate_events(events, 2)


class RowMembershipTests(unittest.TestCase):
    def test_accept_exact_membership(self):
        rows = complete_membership_rows(trials=1, warmups=1, batches=2)
        self.assertTrue(startup.check_row_membership(rows, 1, 1, 2))
        self.assertEqual(len(rows), 12)

    def test_omission_six_of_seven(self):
        rows = complete_membership_rows(trials=1, warmups=0, batches=7)
        self.assertEqual(len(rows), 28)
        del rows[6]
        with self.assertRaisesRegex(
                ValueError, r"row membership invalid: omission"):
            startup.check_row_membership(rows, 1, 0, 7)

    def test_duplicate_rejects(self):
        rows = complete_membership_rows(trials=1, warmups=1, batches=2)
        rows.append(dict(rows[0]))
        with self.assertRaisesRegex(
                ValueError, r"row membership invalid: duplicate"):
            startup.check_row_membership(rows, 1, 1, 2)

    def test_unknown_ids_reject(self):
        base = complete_membership_rows(trials=1, warmups=1, batches=2)
        bad_trial = [dict(item) for item in base]
        bad_trial[0] = dict(bad_trial[0])
        bad_trial[0]["trial"] = 99
        with self.assertRaisesRegex(
                ValueError, r"row membership invalid: unknown"):
            startup.check_row_membership(bad_trial, 1, 1, 2)
        bad_profile = [dict(item) for item in base]
        bad_profile[0] = dict(bad_profile[0])
        bad_profile[0]["profile"] = "bogus-profile"
        with self.assertRaisesRegex(
                ValueError, r"row membership invalid: unknown"):
            startup.check_row_membership(bad_profile, 1, 1, 2)
        bad_batch = [dict(item) for item in base]
        bad_batch[0] = dict(bad_batch[0])
        bad_batch[0]["batch"] = 99
        with self.assertRaisesRegex(
                ValueError, r"row membership invalid: unknown"):
            startup.check_row_membership(bad_batch, 1, 1, 2)

    def test_wrong_warmup_rejects(self):
        rows = complete_membership_rows(trials=1, warmups=1, batches=2)
        for item in rows:
            if item["batch"] == 0:
                item["warmup"] = False
                break
        with self.assertRaisesRegex(
                ValueError, r"row membership invalid: warmup"):
            startup.check_row_membership(rows, 1, 1, 2)

    def test_bool_trial_rejects(self):
        rows = complete_membership_rows(trials=1, warmups=1, batches=2)
        rows[0] = dict(rows[0])
        rows[0]["trial"] = True
        with self.assertRaisesRegex(
                ValueError, r"row membership invalid: unknown"):
            startup.check_row_membership(rows, 1, 1, 2)

    def test_non_list_rows_rejects(self):
        for bad in ("not-a-list", {"rows": []}, None):
            with self.subTest(bad=bad):
                with self.assertRaisesRegex(
                        ValueError, r"row membership invalid: count"):
                    startup.check_row_membership(bad, 1, 1, 2)


class TrialRowsTests(unittest.TestCase):
    def trial_rows(self, trial_index=0, warmups=1, batches=2):
        rows = []
        for profile in startup.PROFILES:
            for batch in range(warmups + batches):
                rows.append({"trial": trial_index, "batch": batch,
                             "profile": profile,
                             "warmup": batch < warmups})
        return rows

    def test_accept_exact_trial(self):
        rows = self.trial_rows(0, 1, 2)
        self.assertTrue(startup.check_trial_rows(rows, 0, 1, 2))

    def test_omission_six_of_seven(self):
        rows = self.trial_rows(0, 0, 7)
        del rows[3]
        with self.assertRaisesRegex(
                ValueError, r"row membership invalid: omission"):
            startup.check_trial_rows(rows, 0, 0, 7)

    def test_duplicate_rejects(self):
        rows = self.trial_rows(0, 1, 2)
        rows.append(dict(rows[0]))
        with self.assertRaisesRegex(
                ValueError, r"row membership invalid: duplicate"):
            startup.check_trial_rows(rows, 0, 1, 2)

    def test_unknown_rejects(self):
        wrong_trial = self.trial_rows(0, 1, 2)
        wrong_trial[0] = dict(wrong_trial[0])
        wrong_trial[0]["trial"] = 1
        with self.assertRaisesRegex(
                ValueError, r"row membership invalid: unknown"):
            startup.check_trial_rows(wrong_trial, 0, 1, 2)
        bad_profile = self.trial_rows(0, 1, 2)
        bad_profile[0] = dict(bad_profile[0])
        bad_profile[0]["profile"] = "nope"
        with self.assertRaisesRegex(
                ValueError, r"row membership invalid: unknown"):
            startup.check_trial_rows(bad_profile, 0, 1, 2)
        bad_batch = self.trial_rows(0, 1, 2)
        bad_batch[0] = dict(bad_batch[0])
        bad_batch[0]["batch"] = 99
        with self.assertRaisesRegex(
                ValueError, r"row membership invalid: unknown"):
            startup.check_trial_rows(bad_batch, 0, 1, 2)

    def test_wrong_warmup_rejects(self):
        rows = self.trial_rows(0, 1, 2)
        rows[0] = dict(rows[0])
        rows[0]["warmup"] = False
        with self.assertRaisesRegex(
                ValueError, r"row membership invalid: warmup"):
            startup.check_trial_rows(rows, 0, 1, 2)

    def test_bool_trial_rejects(self):
        rows = self.trial_rows(0, 1, 2)
        rows[0] = dict(rows[0])
        rows[0]["trial"] = True
        with self.assertRaisesRegex(
                ValueError, r"row membership invalid: unknown"):
            startup.check_trial_rows(rows, 0, 1, 2)

    def test_non_list_rejects(self):
        for bad in (None, "rows", {"trial": 0}):
            with self.subTest(bad=bad):
                with self.assertRaisesRegex(
                        ValueError, r"row membership invalid: count"):
                    startup.check_trial_rows(bad, 0, 1, 2)


class BuildReportTests(unittest.TestCase):
    TEXT = "const a = 1;\ngo(a);\n"

    def manifest(self):
        return good_manifest(self.TEXT)

    def full_rows(self, trials=1, warmups=1, batches=2):
        return complete_membership_rows(
            trials, warmups, batches, full=True, statement_count=2)

    def test_accept_exact_membership(self):
        manifest = self.manifest()
        rows = self.full_rows(1, 1, 2)
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "rows.json"
            path.write_text(json.dumps(rows))
            loaded = json.loads(path.read_text())
        report = startup.build_report(loaded, 1, manifest,
                                      warmups=1, batches=2)
        self.assertEqual(report["accepted_batches_per_profile_trial"], 2)
        self.assertEqual(set(report["profiles"]), set(startup.PROFILES))
        self.assertEqual(len(report["diagnostic_statements"]), 2)

    def test_reject_six_of_seven(self):
        manifest = self.manifest()
        rows = complete_membership_rows(1, 0, 7, full=True,
                                        statement_count=2)
        del rows[5]
        with self.assertRaisesRegex(startup._Fail, r"report rejected"):
            startup.build_report(rows, 1, manifest,
                                 warmups=0, batches=7)

    def test_reject_duplicate(self):
        manifest = self.manifest()
        rows = self.full_rows(1, 1, 2)
        rows.append(dict(rows[0]))
        with self.assertRaisesRegex(startup._Fail, r"report rejected"):
            startup.build_report(rows, 1, manifest,
                                 warmups=1, batches=2)


class ManifestSourceTests(unittest.TestCase):
    TEXT = "const a = 1;\ngo(a);\n"

    def test_coverage_rejects_wrong_sha(self):
        manifest = good_manifest(self.TEXT)
        manifest["state_sha256"] = "0" * 64
        with self.assertRaisesRegex(
                ValueError, r"manifest coverage invalid: source"):
            startup.check_manifest_coverage(manifest, self.TEXT)

    def test_coverage_rejects_wrong_bytes(self):
        manifest = good_manifest(self.TEXT)
        manifest["state_bytes"] += 1
        with self.assertRaisesRegex(
                ValueError, r"manifest coverage invalid: source"):
            startup.check_manifest_coverage(manifest, self.TEXT)

    def test_coverage_rejects_initializer_mismatch(self):
        for field, value in (("name", "foo"),
                             ("start", 999),
                             ("end", 999)):
            with self.subTest(field=field):
                manifest = good_manifest(self.TEXT)
                manifest["initializer"][field] = value
                with self.assertRaisesRegex(
                        ValueError,
                        r"manifest coverage invalid: initializer"):
                    startup.check_manifest_coverage(manifest, self.TEXT)

    def test_agreement_rejects_omitted_statement(self):
        authoritative = good_manifest(self.TEXT)
        candidate = json.loads(json.dumps(authoritative))
        candidate["statements"] = candidate["statements"][:-1]
        with self.assertRaisesRegex(
                ValueError, r"manifest agreement invalid: count"):
            startup.check_manifest_agreement(authoritative, candidate)

    def test_agreement_rejects_tampered_span(self):
        authoritative = good_manifest(self.TEXT)
        candidate = json.loads(json.dumps(authoritative))
        candidate["statements"][0]["end"] += 1
        with self.assertRaisesRegex(
                ValueError, r"manifest agreement invalid: span"):
            startup.check_manifest_agreement(authoritative, candidate)

    def test_agreement_rejects_tampered_hash(self):
        authoritative = good_manifest(self.TEXT)
        candidate = json.loads(json.dumps(authoritative))
        candidate["statements"][0]["sha256"] = "0" * 64
        with self.assertRaisesRegex(
                ValueError, r"manifest agreement invalid: hash"):
            startup.check_manifest_agreement(authoritative, candidate)
        candidate = json.loads(json.dumps(authoritative))
        candidate["statements"][1]["factories"] = ["evil"]
        with self.assertRaisesRegex(
                ValueError, r"manifest agreement invalid: hash"):
            startup.check_manifest_agreement(authoritative, candidate)

    def test_agreement_accepts_identical(self):
        authoritative = good_manifest(self.TEXT)
        candidate = json.loads(json.dumps(authoritative))
        self.assertTrue(
            startup.check_manifest_agreement(authoritative, candidate))


class AstralSpanTests(unittest.TestCase):
    @needs_node
    def test_probe_inventory_astral_coverage(self):
        text = ("// \U0001F600 astral before initializer\n"
                "export function $canInitialize() {\n"
                "  const a = 1;\n"
                "  const b = 2;\n"
                "}\n")
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(directory, "state.ts", text)
            manifest_path = Path(directory) / "manifest.json"
            result = run_probe("inventory", "--state", str(state),
                               "--manifest", str(manifest_path))
            self.assertEqual(result.returncode, 0, msg=result.stderr)
            manifest = json.loads(manifest_path.read_text())
            self.assertEqual(
                manifest["initializer"]["statement_count"], 2)
            self.assertTrue(
                startup.check_manifest_coverage(manifest, text))

    def test_utf16_to_python_index(self):
        text = "A\U0001F600B"
        self.assertEqual(
            startup._utf16_to_python_index(text, 0), 0)
        self.assertEqual(
            startup._utf16_to_python_index(text, 1), 1)
        self.assertEqual(
            startup._utf16_to_python_index(text, 3), 2)
        self.assertEqual(
            startup._utf16_to_python_index(text, 4), 3)
        for bad in (2, -1, 5, True, "1", None, 1.5):
            with self.subTest(offset=bad):
                with self.assertRaisesRegex(
                        ValueError,
                        r"manifest coverage invalid: span"):
                    startup._utf16_to_python_index(text, bad)
        ascii_text = "abc"
        for offset, expected in ((0, 0), (1, 1), (2, 2), (3, 3)):
            self.assertEqual(
                startup._utf16_to_python_index(ascii_text, offset),
                expected)


class ProbeSafeWriteTests(unittest.TestCase):
    def assert_one_line_error(self, result, code):
        self.assertEqual(result.returncode, 2, msg=result.stderr)
        lines = result.stderr.strip().splitlines()
        self.assertEqual(len(lines), 1, msg=result.stderr)
        self.assertTrue(
            lines[0].startswith(f"startup-probe-error {code} "),
            msg=lines[0])

    @needs_node
    def test_output_exists_inventory_preserves_manifest(self):
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(directory, "state.ts", FULL_STATE)
            manifest_path = Path(directory) / "manifest.json"
            sentinel = b"sentinel-manifest-bytes"
            manifest_path.write_bytes(sentinel)
            result = run_probe("inventory", "--state", str(state),
                               "--manifest", str(manifest_path))
            self.assert_one_line_error(result, "output_exists")
            self.assertEqual(manifest_path.read_bytes(), sentinel)

    @needs_node
    def test_output_exists_instrument_preserves_out(self):
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(directory, "state.ts", FULL_STATE)
            out_path = Path(directory) / "out.ts"
            manifest_path = Path(directory) / "manifest.json"
            sentinel = b"sentinel-out-bytes"
            out_path.write_bytes(sentinel)
            result = run_probe(
                "instrument", "--state", str(state),
                "--out", str(out_path),
                "--manifest", str(manifest_path),
                "--probe-specifier", "./startup-probe.ts")
            self.assert_one_line_error(result, "output_exists")
            self.assertEqual(out_path.read_bytes(), sentinel)
            self.assertFalse(manifest_path.exists())

    @needs_node
    def test_output_exists_instrument_preserves_manifest(self):
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(directory, "state.ts", FULL_STATE)
            out_path = Path(directory) / "out.ts"
            manifest_path = Path(directory) / "manifest.json"
            sentinel = b"sentinel-manifest-2"
            manifest_path.write_bytes(sentinel)
            result = run_probe(
                "instrument", "--state", str(state),
                "--out", str(out_path),
                "--manifest", str(manifest_path),
                "--probe-specifier", "./startup-probe.ts")
            self.assert_one_line_error(result, "output_exists")
            self.assertEqual(manifest_path.read_bytes(), sentinel)
            self.assertFalse(out_path.exists())

    @needs_node
    def test_second_write_failure_preserves_first_output(self):
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(directory, "state.ts", FULL_STATE)
            out_path = Path(directory) / "out.ts"
            sentinel = b"first-output-sentinel"
            out_path.write_bytes(sentinel)
            manifest_path = (Path(directory) / "missing-dir"
                             / "manifest.json")
            result = run_probe(
                "instrument", "--state", str(state),
                "--out", str(out_path),
                "--manifest", str(manifest_path),
                "--probe-specifier", "./startup-probe.ts")
            self.assert_one_line_error(result, "output_exists")
            self.assertEqual(out_path.read_bytes(), sentinel)
            self.assertFalse(manifest_path.exists())

    @needs_node
    def test_second_write_io_failure_rolls_back_new_out(self):
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(directory, "state.ts", FULL_STATE)
            out_path = Path(directory) / "out.ts"
            manifest_path = (Path(directory) / "missing-dir"
                             / "manifest.json")
            result = run_probe(
                "instrument", "--state", str(state),
                "--out", str(out_path),
                "--manifest", str(manifest_path),
                "--probe-specifier", "./startup-probe.ts")
            self.assert_one_line_error(result, "io_error")
            self.assertFalse(out_path.exists())
            self.assertFalse(manifest_path.exists())

    @needs_node
    def test_path_alias_out_equals_state(self):
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(directory, "state.ts", FULL_STATE)
            before = state.read_bytes()
            manifest_path = Path(directory) / "manifest.json"
            result = run_probe(
                "instrument", "--state", str(state),
                "--out", str(state),
                "--manifest", str(manifest_path),
                "--probe-specifier", "./startup-probe.ts")
            self.assert_one_line_error(result, "path_alias")
            self.assertEqual(state.read_bytes(), before)
            self.assertFalse(manifest_path.exists())

    @needs_node
    def test_path_alias_manifest_equals_state(self):
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(directory, "state.ts", FULL_STATE)
            before = state.read_bytes()
            result = run_probe("inventory", "--state", str(state),
                               "--manifest", str(state))
            self.assert_one_line_error(result, "path_alias")
            self.assertEqual(state.read_bytes(), before)

    @needs_node
    def test_path_alias_manifest_equals_state_instrument(self):
        with tempfile.TemporaryDirectory() as directory:
            state = write_state(directory, "state.ts", FULL_STATE)
            before = state.read_bytes()
            out_path = Path(directory) / "out.ts"
            result = run_probe(
                "instrument", "--state", str(state),
                "--out", str(out_path),
                "--manifest", str(state),
                "--probe-specifier", "./startup-probe.ts")
            self.assert_one_line_error(result, "path_alias")
            self.assertEqual(state.read_bytes(), before)
            self.assertFalse(out_path.exists())


class WriteOutputTests(unittest.TestCase):
    def test_write_output_refuses_preexisting(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "out.json"
            sentinel = '{"sentinel": true}\n'
            path.write_text(sentinel)
            with self.assertRaisesRegex(
                    ValueError, r"output refuses preexisting path"):
                startup._write_output(path, {"x": 1})
            self.assertEqual(path.read_text(), sentinel)

    def test_write_output_atomic_new_path(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "nested" / "dir" / "out.json"
            record = {"alpha": 1, "beta": [1, 2, 3]}
            startup._write_output(path, record)
            self.assertEqual(json.loads(path.read_text()), record)
            leftovers = list(Path(directory).rglob("*.tmp-*"))
            self.assertEqual(leftovers, [])
            with self.assertRaisesRegex(
                    ValueError, r"output refuses preexisting path"):
                startup._write_output(path, {"other": 2})
            self.assertEqual(json.loads(path.read_text()), record)


class OwnershipTests(unittest.TestCase):
    def test_cleanup_unregistered_prefix_preserved(self):
        with tempfile.TemporaryDirectory() as parent:
            target = Path(parent) / (
                startup.SCRATCH_PREFIX + "unregistered")
            target.mkdir()
            (target / "keep.txt").write_text("x")
            self.addCleanup(shutil.rmtree, target, True)
            with self.assertRaisesRegex(
                    ValueError, r"scratch ownership invalid"):
                startup.cleanup_scratch(target)
            self.assertTrue((target / "keep.txt").exists())

    def test_root_replacement_refuses(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup._ALLOCATED.discard, str(scratch))
            self.addCleanup(shutil.rmtree, scratch, True)
            (scratch / "keep.txt").write_text("x")
            marker = scratch / startup.OWNER_MARKER
            meta = json.loads(marker.read_text())
            meta["root_identity"] = [0, 0]
            marker.write_text(json.dumps(meta))
            with self.assertRaisesRegex(
                    ValueError,
                    r"scratch ownership invalid: replaced"):
                startup.cleanup_scratch(scratch)
            self.assertTrue((scratch / "keep.txt").exists())
            self.assertTrue(marker.exists())

    def test_marker_replacement_refuses(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup._ALLOCATED.discard, str(scratch))
            self.addCleanup(shutil.rmtree, scratch, True)
            marker = scratch / startup.OWNER_MARKER
            meta = json.loads(marker.read_text())
            meta["marker_identity"] = [0, 0]
            marker.write_text(json.dumps(meta))
            with self.assertRaisesRegex(
                    ValueError,
                    r"scratch ownership invalid: replaced"):
                startup.cleanup_scratch(scratch)
            self.assertTrue(scratch.exists())

    @unittest.skipUnless(shutil.which("sleep"),
                         "sleep is required for ownership liveness")
    def test_forged_live_foreign_pid_active_owner(self):
        with tempfile.TemporaryDirectory() as parent:
            target = Path(parent) / (
                startup.SCRATCH_PREFIX + "forged-live")
            target.mkdir()
            (target / "keep.txt").write_text("x")
            proc = subprocess.Popen([shutil.which("sleep"), "30"],
                                    stdout=subprocess.DEVNULL,
                                    stderr=subprocess.DEVNULL)

            def kill_sleep():
                try:
                    proc.kill()
                except OSError:
                    pass
                try:
                    proc.wait(timeout=2)
                except Exception:
                    pass

            self.addCleanup(kill_sleep)
            marker = target / startup.OWNER_MARKER
            marker.write_text("{}")
            root_info = os.stat(target, follow_symlinks=False)
            marker_info = os.stat(marker, follow_symlinks=False)
            meta = {
                "kind": startup.OWNER_KIND,
                "schema_version": startup.OWNER_SCHEMA,
                "root": str(target),
                "pid": proc.pid,
                "process_start": startup._process_start(proc.pid),
                "created_at": 0,
                "root_identity": [root_info.st_dev, root_info.st_ino],
                "marker_identity": [marker_info.st_dev,
                                    marker_info.st_ino],
                "process_group": None,
                "launched_children": False,
                "transferred": False,
                "keeper_pid": None,
                "state": "active",
            }
            marker.write_text(json.dumps(meta))
            self.assertTrue(startup._pid_alive(proc.pid))
            with self.assertRaisesRegex(
                    ValueError,
                    r"scratch ownership invalid: active_owner"):
                startup.cleanup_scratch(target)
            self.assertTrue((target / "keep.txt").exists())
            self.assertTrue(target.exists())

    @unittest.skipUnless(shutil.which("sleep"),
                         "sleep is required for ownership liveness")
    def test_forged_dead_pid_recover_cleans(self):
        with tempfile.TemporaryDirectory() as parent:
            target = Path(parent) / (
                startup.SCRATCH_PREFIX + "forged-dead")
            target.mkdir()
            (target / "keep.txt").write_text("x")
            sleeper = subprocess.Popen(
                [shutil.which("sleep"), "0.01"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL)
            dead_pid = sleeper.pid
            sleeper.wait()
            self.assertFalse(startup._pid_alive(dead_pid))
            marker = target / startup.OWNER_MARKER
            marker.write_text("{}")
            root_info = os.stat(target, follow_symlinks=False)
            marker_info = os.stat(marker, follow_symlinks=False)
            meta = {
                "kind": startup.OWNER_KIND,
                "schema_version": startup.OWNER_SCHEMA,
                "root": str(target),
                "pid": dead_pid,
                "process_start": None,
                "created_at": 0,
                "root_identity": [root_info.st_dev, root_info.st_ino],
                "marker_identity": [marker_info.st_dev,
                                    marker_info.st_ino],
                "process_group": None,
                "launched_children": False,
                "transferred": False,
                "keeper_pid": None,
                "state": "active",
            }
            marker.write_text(json.dumps(meta))
            self.assertTrue(startup.recover_scratch(target))
            self.assertFalse(target.exists())

    def test_cleanup_failure_chmod_retains(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            (scratch / "keep.txt").write_text("x")
            marker = scratch / startup.OWNER_MARKER

            def restore_and_cleanup():
                try:
                    os.chmod(scratch, 0o700)
                except OSError:
                    pass
                try:
                    startup.cleanup_scratch(scratch)
                except ValueError:
                    try:
                        shutil.rmtree(scratch, ignore_errors=True)
                    except OSError:
                        pass
                startup._ALLOCATED.discard(str(scratch))

            self.addCleanup(restore_and_cleanup)
            os.chmod(scratch, 0o500)
            with self.assertRaisesRegex(
                    ValueError, r"scratch cleanup failed"):
                startup.cleanup_scratch(scratch)
            self.assertTrue(scratch.exists())
            self.assertTrue(marker.exists())
            self.assertIn(str(scratch), startup._ALLOCATED)
            os.chmod(scratch, 0o700)
            self.assertTrue(startup.cleanup_scratch(scratch))
            self.assertFalse(scratch.exists())

    def test_keep_transfer_retained_then_retired(self):
        with tempfile.TemporaryDirectory() as parent:
            keeper = None
            with startup.owned_scratch(parent, keep=True) as scratch:
                (scratch / "kept.txt").write_text("x")
                keeper = scratch
            self.addCleanup(startup._ALLOCATED.discard, str(keeper))
            self.addCleanup(shutil.rmtree, keeper, True)
            self.assertTrue((keeper / "kept.txt").exists())
            meta = json.loads(
                (keeper / startup.OWNER_MARKER).read_text())
            self.assertEqual(meta["state"], "retained")
            self.assertTrue(meta["transferred"])
            self.assertEqual(meta["keeper_pid"], os.getpid())
            self.assertTrue(startup.cleanup_scratch(keeper))
            self.assertFalse(keeper.exists())


class LifecycleTests(unittest.TestCase):
    @unittest.skipUnless(shutil.which("sh") and shutil.which("sleep"),
                         "sh and sleep are required")
    def test_run_supervised_timeout_retires_controller_grandchild(self):
        with tempfile.TemporaryDirectory() as cwd:
            pidfile = Path(cwd) / "grandchild.pid"
            controller = {}

            def on_start(pid):
                controller["pid"] = pid

            def kill_stragglers():
                try:
                    if "pid" in controller:
                        try:
                            startup._kill_process_group(
                                controller["pid"], term_grace=0.2,
                                kill_grace=0.2)
                        except ValueError:
                            pass
                except Exception:
                    pass
                try:
                    if pidfile.exists():
                        try:
                            grand = int(pidfile.read_text().strip())
                            try:
                                os.kill(grand, signal.SIGKILL)
                            except OSError:
                                pass
                        except (ValueError, OSError):
                            pass
                except OSError:
                    pass

            self.addCleanup(kill_stragglers)
            with self.assertRaises(subprocess.TimeoutExpired):
                startup.run_supervised(
                    ["sh", "-c",
                     f"sleep 30 & echo $! > '{pidfile}'; wait"],
                    cwd=cwd, timeout=1.0, term_grace=0.2,
                    kill_grace=0.2, on_start=on_start)
            self.assertIn("pid", controller)
            self.assertFalse(startup._pid_alive(controller["pid"]))
            self.assertTrue(pidfile.exists(),
                            msg="controller did not record grandchild")
            grandchild = int(pidfile.read_text().strip())
            self.assertGreater(grandchild, 1)
            self.assertFalse(startup._pid_alive(grandchild))

    @unittest.skipUnless(shutil.which("sh") and shutil.which("sleep"),
                         "sh and sleep are required")
    def test_kill_process_group_cooperative(self):
        with tempfile.TemporaryDirectory() as cwd:
            pidfile = Path(cwd) / "gp.pid"
            proc = subprocess.Popen(
                ["sh", "-c",
                 f"sleep 30 & echo $! > '{pidfile}'; wait"],
                start_new_session=True,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL)

            def cleanup():
                try:
                    proc.kill()
                except OSError:
                    pass
                try:
                    proc.wait(timeout=2)
                except Exception:
                    pass
                try:
                    if pidfile.exists():
                        try:
                            grand = int(pidfile.read_text().strip())
                            try:
                                os.kill(grand, signal.SIGKILL)
                            except OSError:
                                pass
                        except (ValueError, OSError):
                            pass
                except OSError:
                    pass

            self.addCleanup(cleanup)
            deadline = time.monotonic() + 2.0
            while not pidfile.exists() and time.monotonic() < deadline:
                time.sleep(0.05)
            self.assertTrue(pidfile.exists(),
                            msg="grandchild pid not recorded")
            grandchild = int(pidfile.read_text().strip())
            self.assertTrue(startup._pid_alive(proc.pid))
            self.assertTrue(startup._pid_alive(grandchild))
            result = startup._kill_process_group(
                proc.pid, term_grace=1.0, kill_grace=0.5)
            self.assertTrue(result["termed"])
            self.assertFalse(result["killed"])
            self.assertTrue(result["retired"])
            proc.wait(timeout=2)
            self.assertFalse(startup._pid_alive(grandchild))

    @unittest.skipUnless(shutil.which("sh") and shutil.which("sleep"),
                         "sh and sleep are required")
    def test_kill_process_group_sigterm_ignoring(self):
        with tempfile.TemporaryDirectory() as cwd:
            pidfile = Path(cwd) / "gp.pid"
            proc = subprocess.Popen(
                ["sh", "-c",
                 "trap '' TERM; (trap '' TERM; exec sleep 30) & "
                 f"echo $! > '{pidfile}'; wait"],
                start_new_session=True,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL)

            def cleanup():
                try:
                    proc.kill()
                except OSError:
                    pass
                try:
                    proc.wait(timeout=2)
                except Exception:
                    pass
                try:
                    if pidfile.exists():
                        try:
                            grand = int(pidfile.read_text().strip())
                            try:
                                os.kill(grand, signal.SIGKILL)
                            except OSError:
                                pass
                        except (ValueError, OSError):
                            pass
                except OSError:
                    pass

            self.addCleanup(cleanup)
            deadline = time.monotonic() + 2.0
            while not pidfile.exists() and time.monotonic() < deadline:
                time.sleep(0.05)
            self.assertTrue(pidfile.exists(),
                            msg="grandchild pid not recorded")
            grandchild = int(pidfile.read_text().strip())
            result = startup._kill_process_group(
                proc.pid, term_grace=0.3, kill_grace=1.0)
            self.assertFalse(result["termed"])
            self.assertTrue(result["killed"])
            self.assertTrue(result["retired"])
            proc.wait(timeout=2)
            self.assertFalse(startup._pid_alive(grandchild))

    def test_run_supervised_success(self):
        with tempfile.TemporaryDirectory() as cwd:
            completed = startup.run_supervised(
                [sys.executable, "-c", "print('hello-supervised')"],
                cwd=cwd, timeout=5.0, term_grace=0.2,
                kill_grace=0.2)
            self.assertEqual(completed.returncode, 0)
            self.assertIn("hello-supervised", completed.stdout)

    def test_run_supervised_nonzero(self):
        with tempfile.TemporaryDirectory() as cwd:
            with self.assertRaises(subprocess.CalledProcessError) as ctx:
                startup.run_supervised(
                    [sys.executable, "-c",
                     "import sys; sys.exit(3)"],
                    cwd=cwd, timeout=5.0, term_grace=0.2,
                    kill_grace=0.2)
            self.assertEqual(ctx.exception.returncode, 3)


class IdentityTests(unittest.TestCase):
    def test_collect_tool_versions_keys(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            try:
                tracker = startup._phase_tracker(scratch)
                versions = startup.collect_tool_versions(tracker)
            finally:
                startup.cleanup_scratch(str(scratch))
        self.assertEqual(set(versions),
                         {"bun", "go", "node", "typescript"})
        for key, value in versions.items():
            with self.subTest(tool=key):
                self.assertTrue(
                    value is None or (
                        isinstance(value, str)
                        and len(value.strip()) > 0),
                    msg=f"{key}={value!r}")

    def test_collect_prepared_identities_drift(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup.cleanup_scratch, str(scratch))
            repo = Path(parent) / "repo"
            (scratch / "generated-js").mkdir(parents=True)
            (scratch / "generated-js" / "a.js").write_text(
                "console.log(1);\n")
            (scratch / "generated-js-diag").mkdir(parents=True)
            (scratch / "generated-js-diag" / "b.js").write_text(
                "console.log(2);\n")
            (scratch / "startup-ordinary-bundle.js").write_text(
                "/* bundle */\n")
            (scratch / "generated" / "program").mkdir(parents=True)
            (scratch / "generated" / "program" / "state.ts").write_text(
                "export const x = 1;\n")
            (scratch / "generated" / "startup-facade.ts").write_text(
                "facade\n")
            (scratch / "generated" / "startup-minimal.ts").write_text(
                "minimal\n")
            (scratch / "instrument-src" / "program").mkdir(parents=True)
            (scratch / "instrument-src" / "program"
             / "startup-probe.ts").write_text("probe\n")
            (scratch / "startup-launcher.js").write_text("launcher\n")
            real_modules = Path(parent) / "real-node-modules"
            real_modules.mkdir()
            (scratch / "node_modules").symlink_to(
                real_modules, target_is_directory=True)
            (repo / "tools" / "performance" / "drivers").mkdir(
                parents=True)
            (repo / "tools" / "performance" / "drivers"
             / "runtime.py").write_text("# runtime\n")
            (repo / "tools" / "performance" / "drivers"
             / "runtime-transpile.ts").write_text("// transpile\n")
            (repo / "tools" / "performance"
             / "startup-attribution.py").write_text("# driver\n")
            (repo / "tools" / "performance"
             / "startup-attribution.ts").write_text("// probe\n")
            (repo / "package.json").write_text('{"name": "x"}\n')
            (repo / "bun.lock").write_text("lock\n")
            tracker = startup._phase_tracker(scratch)
            before = startup.collect_prepared_identities(scratch, repo, tracker)
            identical = json.loads(json.dumps(before))
            self.assertTrue(
                startup.check_identity_drift(before, identical))
            tampered = json.loads(json.dumps(before))
            tampered["prepared"]["bundle"]["bytes"] = 999999
            with self.assertRaisesRegex(
                    ValueError, r"prepared identity drift"):
                startup.check_identity_drift(before, tampered)
            (scratch / "generated-js" / "a.js").write_text(
                "console.log(999);\n")
            after = startup.collect_prepared_identities(scratch, repo, tracker)
            with self.assertRaisesRegex(
                    ValueError, r"prepared identity drift"):
                startup.check_identity_drift(before, after)


# G28g regression helpers (frozen G28e contracts). Short-lived
# fixtures only, short graces, immediate test-owned cleanup.


def g28g_kill_stragglers(pid=None, pidfile=None):
    """Best-effort retirement for fixtures, including pre-fix leaks."""
    if pid is not None:
        try:
            startup._kill_process_group(pid, term_grace=0.2,
                                        kill_grace=0.2)
        except (ValueError, OSError):
            pass
    if pidfile is not None:
        try:
            if not pidfile.exists():
                return
            grand = int(pidfile.read_text().strip())
        except (ValueError, OSError):
            return
        try:
            os.kill(grand, signal.SIGKILL)
        except OSError:
            pass


class g28g_unverified_retirement:
    """Force the unverified branch while still really retiring groups.

    Wraps the real group killer so descendants are actually reaped
    (no leaked processes) but run_supervised observes the R5 failure
    report: RetirementFailed instead of a clean handoff.
    """

    def __enter__(self):
        real_kill = startup._kill_process_group

        def fake_kill(pgid, term_grace=2.0, kill_grace=1.0,
                      leader=None):
            try:
                real_kill(pgid, term_grace=term_grace,
                          kill_grace=kill_grace)
            except (ValueError, OSError):
                pass
            return {"pgid": pgid, "termed": False, "killed": True,
                    "retired": False, "refused": None}

        self.real_kill = real_kill
        startup._kill_process_group = fake_kill
        return self

    def __exit__(self, exc_type, exc, tb):
        startup._kill_process_group = self.real_kill
        return False


def g28g_complete_identity():
    """One complete synthetic prepared-identity record (G28e shape)."""
    executable = {"realpath": "/usr/bin/tool", "version": "1.0",
                  "dev": 16777233, "ino": 12345, "size": 67890}
    return {
        "tool_versions": {"bun": "1.4.2", "go": "go1.27.1",
                          "node": "v24.21.0", "typescript": "7.0.2"},
        "builtins": {"bun_revision": "abc123"},
        "executables": {"bun": dict(executable),
                        "go": dict(executable),
                        "node": dict(executable)},
        "drivers": {"drivers/runtime.py": "a" * 64,
                    "drivers/runtime-transpile.ts": "b" * 64,
                    "startup-attribution.py": "c" * 64,
                    "startup-attribution.ts": "d" * 64},
        "sources": {"startup-facade.ts": "e" * 64,
                    "startup-minimal.ts": "f" * 64,
                    "startup-probe.ts": "g" * 64,
                    "startup-launcher.js": "h" * 64,
                    "state.ts": {"sha256": "i" * 64,
                                 "bytes": 187723}},
        "inputs": {
            "fixture": [{"path": "main.can", "bytes": 12,
                         "sha256": "j" * 64}],
            "emitter": {
                "packages": [{"package": "can/compiler",
                              "files": [{"path": "compiler/x.go",
                                         "bytes": 10,
                                         "sha256": "k" * 64}]}],
                "go_mod": "l" * 64,
                "binary": {"sha256": "m" * 64, "bytes": 4455667},
            },
        },
        "runtime_ts": {"files": [{"path": "data.ts", "bytes": 100,
                                  "sha256": "n" * 64}]},
        "emitted": {"adapter": "o" * 64,
                    "packages": [{"path": "p.ts", "bytes": 50,
                                  "sha256": "p" * 64}]},
        "prepared": {
            "ordinary": {"files": [{"path": "startup-facade.js",
                                    "bytes": 200,
                                    "sha256": "q" * 64}]},
            "diagnostic": {"files": [{"path": "startup-facade.js",
                                      "bytes": 210,
                                      "sha256": "r" * 64}]},
            "bundle": {"sha256": "s" * 64, "bytes": 907741},
        },
        "resolved": {
            "computed": True,
            "roots": ["/tmp/scratch/generated-js/startup-facade.js"],
            "reachable_files": 119,
            "reachable_bare": [
                {"specifier": "node:fs",
                 "importers": ["/tmp/scratch/a.js"],
                 "resolution": {"status": "builtin",
                               "runtime": {"bun_revision": "abc123"}}},
                {"specifier": "node:path",
                 "importers": ["/tmp/scratch/a.js",
                               "/tmp/scratch/b.js"],
                 "resolution": {"status": "builtin",
                               "runtime": {"bun_revision": "abc123"}}},
            ],
            "unresolved": [],
            "truncated": False,
            "unreachable_bare": [],
        },
        "dependencies": {
            "node_modules": {"link": "../node_modules",
                             "realpath": "/repo/node_modules"},
            "package_json": "u" * 64,
            "bun_lock": "v" * 64,
        },
    }


def g28g_profile_stages(profile):
    """Canned child stages satisfying the frozen profile rules."""
    if profile == "ordinary-bundle":
        bundled = {"status": "not_applicable_bundled", "ms": None}
        return stages_record(diagnostics_import=dict(bundled),
                             configure=dict(bundled))
    return stages_record()


def g28g_rewrite_marker(scratch, **overrides):
    """Rewrite marker content in place (dev/ino identity preserved)."""
    marker = Path(scratch) / startup.OWNER_MARKER
    meta = json.loads(marker.read_text())
    meta.update(overrides)
    marker.write_text(json.dumps(meta))
    return meta


def g28g_dead_pid():
    """A surely-dead pid (short sleep child, reaped)."""
    proc = subprocess.Popen(
        [shutil.which("sleep"), "0.01"],
        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    dead = proc.pid
    proc.wait()
    return dead


class SupervisedSuccessDescendantTests(unittest.TestCase):
    """G28e success-with-leftover: R5 probe left a live descendant."""

    @unittest.skipUnless(shutil.which("sh") and shutil.which("sleep"),
                         "sh and sleep are required")
    def test_success_exit_zero_retires_live_descendant(self):
        with tempfile.TemporaryDirectory() as cwd:
            pidfile = Path(cwd) / "leftover.pid"
            controller = {}

            def on_start(pid):
                controller["pid"] = pid

            def cleanup():
                g28g_kill_stragglers(controller.get("pid"), pidfile)

            self.addCleanup(cleanup)
            completed = startup.run_supervised(
                ["sh", "-c",
                 f"sleep 30 & echo $! > '{pidfile}'"],
                cwd=cwd, timeout=5.0, term_grace=0.3,
                kill_grace=0.5, drain_timeout=1.0,
                on_start=on_start)
            self.assertEqual(completed.returncode, 0)
            self.assertIn("pid", controller)
            self.assertTrue(pidfile.exists(),
                            msg="controller did not record descendant")
            descendant = int(pidfile.read_text().strip())
            self.assertGreater(descendant, 1)
            self.assertFalse(startup._pid_alive(descendant),
                             msg="success left a live descendant")
            self.assertFalse(startup._pid_alive(controller["pid"],
                                                group=True),
                             msg="success left a live worker group")

    @unittest.skipUnless(shutil.which("sh") and shutil.which("sleep"),
                         "sh and sleep are required")
    def test_success_unverified_retirement_raises_with_code(self):
        with tempfile.TemporaryDirectory() as cwd:
            pidfile = Path(cwd) / "leftover.pid"
            controller = {}

            def on_start(pid):
                controller["pid"] = pid

            def cleanup():
                g28g_kill_stragglers(controller.get("pid"), pidfile)

            self.addCleanup(cleanup)
            with g28g_unverified_retirement():
                with self.assertRaises(
                        startup.RetirementFailed) as ctx:
                    startup.run_supervised(
                        ["sh", "-c",
                         f"sleep 30 & echo $! > '{pidfile}'"],
                        cwd=cwd, timeout=5.0, term_grace=0.2,
                        kill_grace=0.2, drain_timeout=1.0,
                        on_start=on_start)
            error = ctx.exception
            self.assertTrue(str(error).startswith(
                "phase retirement failed: "))
            self.assertEqual(error.returncode, 0)
            self.assertFalse(error.timed_out)
            self.assertFalse(error.interrupted)
            self.assertIsNotNone(error.retirement)
            self.assertFalse(error.retirement["retired"])
            self.assertIn("pipes_drained", error.retirement)
            self.assertIn("refused", error.retirement)
            self.assertIn("sh", error.args[0])
            descendant = int(pidfile.read_text().strip())
            self.assertFalse(startup._pid_alive(descendant),
                             msg="fixture leaked its descendant")

    def test_retirement_failed_attributes(self):
        error = startup.RetirementFailed("Detail words")
        self.assertEqual(str(error),
                         "phase retirement failed: Detail words")
        self.assertEqual(error.detail, "Detail words")
        self.assertIsNone(error.args)
        self.assertIsNone(error.returncode)
        self.assertFalse(error.timed_out)
        self.assertFalse(error.interrupted)
        self.assertIsNone(error.retirement)
        full = startup.RetirementFailed(
            "x", args=["a"], returncode=3, timed_out=True,
            interrupted=True, retirement={"retired": False})
        self.assertEqual(full.args, ["a"])
        self.assertEqual(full.returncode, 3)
        self.assertTrue(full.timed_out)
        self.assertTrue(full.interrupted)
        self.assertEqual(full.retirement, {"retired": False})


class SupervisedExitBranchTests(unittest.TestCase):
    """G28e exit-specific contracts: nonzero/timeout/signal/on_start."""

    @unittest.skipUnless(shutil.which("sh") and shutil.which("sleep"),
                         "sh and sleep are required")
    def test_nonzero_with_descendant_retires_then_raises_code(self):
        with tempfile.TemporaryDirectory() as cwd:
            pidfile = Path(cwd) / "failing.pid"
            controller = {}

            def on_start(pid):
                controller["pid"] = pid

            def cleanup():
                g28g_kill_stragglers(controller.get("pid"), pidfile)

            self.addCleanup(cleanup)
            with self.assertRaises(
                    subprocess.CalledProcessError) as ctx:
                startup.run_supervised(
                    ["sh", "-c",
                     f"sleep 30 & echo $! > '{pidfile}'; exit 3"],
                    cwd=cwd, timeout=5.0, term_grace=0.3,
                    kill_grace=0.5, drain_timeout=1.0,
                    on_start=on_start)
            self.assertEqual(ctx.exception.returncode, 3)
            descendant = int(pidfile.read_text().strip())
            self.assertFalse(startup._pid_alive(descendant),
                             msg="nonzero exit leaked its descendant")
            self.assertFalse(startup._pid_alive(controller["pid"],
                                                group=True))

    def test_nonzero_unverified_raises_retirement_failed(self):
        with tempfile.TemporaryDirectory() as cwd:
            with g28g_unverified_retirement():
                with self.assertRaises(
                        startup.RetirementFailed) as ctx:
                    startup.run_supervised(
                        [sys.executable, "-c",
                         "import sys; sys.exit(3)"],
                        cwd=cwd, timeout=5.0, term_grace=0.2,
                        kill_grace=0.2, drain_timeout=1.0)
            error = ctx.exception
            self.assertTrue(str(error).startswith(
                "phase retirement failed: "))
            self.assertEqual(error.returncode, 3)
            self.assertFalse(error.timed_out)
            self.assertFalse(error.interrupted)

    @unittest.skipUnless(shutil.which("sleep"),
                         "sleep is required")
    def test_timeout_unverified_raises_timed_out(self):
        with tempfile.TemporaryDirectory() as cwd:
            controller = {}

            def on_start(pid):
                controller["pid"] = pid

            def cleanup():
                g28g_kill_stragglers(controller.get("pid"))

            self.addCleanup(cleanup)
            with g28g_unverified_retirement():
                with self.assertRaises(
                        startup.RetirementFailed) as ctx:
                    startup.run_supervised(
                        [shutil.which("sleep"), "5"],
                        cwd=cwd, timeout=0.3, term_grace=0.2,
                        kill_grace=0.2, drain_timeout=1.0,
                        on_start=on_start)
            error = ctx.exception
            self.assertTrue(str(error).startswith(
                "phase retirement failed: "))
            self.assertTrue(error.timed_out)
            self.assertFalse(error.interrupted)
            self.assertIsNone(error.returncode)
            self.assertIn("sleep", error.args[0])
            self.assertFalse(startup._pid_alive(controller["pid"],
                                                group=True),
                             msg="fixture leaked its group")

    @unittest.skipUnless(shutil.which("sh") and shutil.which("sleep"),
                         "sh and sleep are required")
    def test_external_sigterm_verified_reraises_original(self):
        with tempfile.TemporaryDirectory() as cwd:
            pidfile = Path(cwd) / "victim.pid"
            controller = {}

            def on_start(pid):
                controller["pid"] = pid

            def cleanup():
                g28g_kill_stragglers(controller.get("pid"), pidfile)

            self.addCleanup(cleanup)
            with startup._interrupts():
                timer = threading.Timer(
                    0.3, os.kill,
                    args=(os.getpid(), signal.SIGTERM))
                timer.daemon = True
                timer.start()
                try:
                    with self.assertRaises(startup._Interrupted):
                        startup.run_supervised(
                            ["sh", "-c",
                             f"sleep 30 & echo $! > '{pidfile}'; wait"],
                            cwd=cwd, timeout=10.0, term_grace=0.2,
                            kill_grace=0.3, drain_timeout=1.0,
                            on_start=on_start)
                finally:
                    timer.cancel()
            self.assertTrue(pidfile.exists(),
                            msg="controller did not record descendant")
            descendant = int(pidfile.read_text().strip())
            self.assertFalse(startup._pid_alive(descendant),
                             msg="signal path leaked its descendant")
            self.assertFalse(startup._pid_alive(controller["pid"],
                                                group=True))

    @unittest.skipUnless(shutil.which("sh") and shutil.which("sleep"),
                         "sh and sleep are required")
    def test_external_sigterm_unverified_raises_interrupted(self):
        with tempfile.TemporaryDirectory() as cwd:
            pidfile = Path(cwd) / "victim.pid"
            controller = {}

            def on_start(pid):
                controller["pid"] = pid

            def cleanup():
                g28g_kill_stragglers(controller.get("pid"), pidfile)

            self.addCleanup(cleanup)
            with startup._interrupts():
                timer = threading.Timer(
                    0.3, os.kill,
                    args=(os.getpid(), signal.SIGTERM))
                timer.daemon = True
                timer.start()
                try:
                    with g28g_unverified_retirement():
                        with self.assertRaises(
                                startup.RetirementFailed) as ctx:
                            startup.run_supervised(
                                ["sh", "-c",
                                 f"sleep 30 & echo $! > '{pidfile}'"
                                 "; wait"],
                                cwd=cwd, timeout=10.0,
                                term_grace=0.2, kill_grace=0.3,
                                drain_timeout=1.0,
                                on_start=on_start)
                finally:
                    timer.cancel()
            error = ctx.exception
            self.assertTrue(str(error).startswith(
                "phase retirement failed: "))
            self.assertTrue(error.interrupted)
            self.assertFalse(error.timed_out)
            self.assertIsInstance(error.__cause__, startup._Interrupted)

    @unittest.skipUnless(shutil.which("sh") and shutil.which("sleep"),
                         "sh and sleep are required")
    def test_on_start_failure_retires_group_then_raises(self):
        with tempfile.TemporaryDirectory() as cwd:
            pidfile = Path(cwd) / "victim.pid"
            seen = {}

            def boom(pid):
                seen["pid"] = pid
                raise RuntimeError("boom-on-start")

            def cleanup():
                g28g_kill_stragglers(seen.get("pid"), pidfile)

            self.addCleanup(cleanup)
            with self.assertRaisesRegex(RuntimeError, "boom-on-start"):
                startup.run_supervised(
                    ["sh", "-c",
                     f"sleep 30 & echo $! > '{pidfile}'; wait"],
                    cwd=cwd, timeout=5.0, term_grace=0.2,
                    kill_grace=0.3, drain_timeout=1.0,
                    on_start=boom)
            self.assertIn("pid", seen)
            self.assertFalse(startup._pid_alive(seen["pid"],
                                                group=True),
                             msg="on_start failure leaked its group")


class SupervisedCaptureTests(unittest.TestCase):
    """G28j pipe-capacity capture: concurrent drain, exact bytes, overflow."""

    def test_stdout_1mib_captured_exactly(self):
        with tempfile.TemporaryDirectory() as cwd:
            completed = startup.run_supervised(
                [sys.executable, "-c",
                 "import sys; sys.stdout.write('x' * 1048576)"],
                cwd=cwd, timeout=10.0, term_grace=0.2,
                kill_grace=0.2, drain_timeout=2.0)
            self.assertEqual(completed.returncode, 0)
            self.assertEqual(len(completed.stdout), 1048576)
            self.assertEqual(completed.stdout, "x" * 1048576)
            self.assertEqual(completed.stderr, "")

    def test_stderr_1mib_captured_exactly(self):
        with tempfile.TemporaryDirectory() as cwd:
            completed = startup.run_supervised(
                [sys.executable, "-c",
                 "import sys; sys.stderr.write('e' * 1048576)"],
                cwd=cwd, timeout=10.0, term_grace=0.2,
                kill_grace=0.2, drain_timeout=2.0)
            self.assertEqual(completed.returncode, 0)
            self.assertEqual(completed.stderr, "e" * 1048576)
            self.assertEqual(completed.stdout, "")

    def test_mixed_large_output_both_pipes_exact(self):
        with tempfile.TemporaryDirectory() as cwd:
            completed = startup.run_supervised(
                [sys.executable, "-c",
                 ("import sys\n"
                  "for _ in range(64):\n"
                  "    sys.stdout.write('o' * 8192)\n"
                  "    sys.stderr.write('r' * 8192)\n")],
                cwd=cwd, timeout=10.0, term_grace=0.2,
                kill_grace=0.2, drain_timeout=2.0)
            self.assertEqual(completed.returncode, 0)
            self.assertEqual(completed.stdout, "o" * 524288)
            self.assertEqual(completed.stderr, "r" * 524288)

    def test_nonzero_large_output_raises_with_output(self):
        with tempfile.TemporaryDirectory() as cwd:
            with self.assertRaises(
                    subprocess.CalledProcessError) as ctx:
                startup.run_supervised(
                    [sys.executable, "-c",
                     ("import sys; sys.stdout.write('v' * 204800);"
                      " sys.exit(3)")],
                    cwd=cwd, timeout=10.0, term_grace=0.2,
                    kill_grace=0.2, drain_timeout=2.0)
            self.assertEqual(ctx.exception.returncode, 3)
            self.assertEqual(ctx.exception.output, "v" * 204800)

    @unittest.skipUnless(shutil.which("sh") and shutil.which("sleep"),
                         "sh and sleep are required")
    def test_leader_exit_output_with_descendant_held_pipe(self):
        with tempfile.TemporaryDirectory() as cwd:
            pidfile = Path(cwd) / "leftover.pid"
            controller = {}

            def on_start(pid):
                controller["pid"] = pid

            def cleanup():
                g28g_kill_stragglers(controller.get("pid"), pidfile)

            self.addCleanup(cleanup)
            completed = startup.run_supervised(
                ["sh", "-c",
                 f"sleep 30 & echo $! > '{pidfile}';"
                 f" {sys.executable} -c \"print('p' * 102400)\""],
                cwd=cwd, timeout=10.0, term_grace=0.3,
                kill_grace=0.5, drain_timeout=2.0,
                on_start=on_start)
            self.assertEqual(completed.returncode, 0)
            self.assertEqual(completed.stdout, "p" * 102400 + "\n")
            descendant = int(pidfile.read_text().strip())
            self.assertFalse(startup._pid_alive(descendant),
                             msg="capture path leaked its descendant")
            self.assertFalse(startup._pid_alive(controller["pid"],
                                                group=True))

    def test_capture_overflow_fails_explicitly(self):
        with tempfile.TemporaryDirectory() as cwd:
            controller = {}

            def on_start(pid):
                controller["pid"] = pid

            def cleanup():
                g28g_kill_stragglers(controller.get("pid"))

            self.addCleanup(cleanup)
            with self.assertRaisesRegex(ValueError,
                                        r"output overflow"):
                startup.run_supervised(
                    [sys.executable, "-c",
                     "import sys; sys.stdout.write('y' * 5242880)"],
                    cwd=cwd, timeout=10.0, term_grace=0.2,
                    kill_grace=0.3, drain_timeout=2.0,
                    on_start=on_start)
            self.assertFalse(startup._pid_alive(controller["pid"],
                                                group=True),
                             msg="overflow path leaked its group")

    def test_timeout_preserves_full_partial_output(self):
        with tempfile.TemporaryDirectory() as cwd:
            with self.assertRaises(
                    subprocess.TimeoutExpired) as ctx:
                startup.run_supervised(
                    [sys.executable, "-c",
                     ("import sys, time; sys.stdout.write('t' * 102400);"
                      " sys.stdout.flush(); time.sleep(30)")],
                    cwd=cwd, timeout=1.0, term_grace=0.2,
                    kill_grace=0.3, drain_timeout=2.0)
            self.assertEqual(ctx.exception.output, "t" * 102400)


class PhaseTrackerCustodyTests(unittest.TestCase):
    """G28e no-premature-clearing: marker keeps group until verified."""

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_track_records_group_and_leader(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup._ALLOCATED.discard, str(scratch))
            self.addCleanup(shutil.rmtree, scratch, True)
            proc = subprocess.Popen(
                [shutil.which("sleep"), "30"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL, start_new_session=True)

            def kill_sleep():
                try:
                    proc.terminate()
                except OSError:
                    pass
                try:
                    proc.wait(timeout=2)
                except Exception:
                    pass

            self.addCleanup(kill_sleep)
            tracker = startup._phase_tracker(scratch)
            tracker.track(proc.pid)
            meta = json.loads(
                (scratch / startup.OWNER_MARKER).read_text())
            self.assertEqual(meta["process_group"], proc.pid)
            self.assertTrue(meta["launched_children"])
            self.assertEqual(meta["group_leader"]["pid"], proc.pid)
            self.assertEqual(tracker.pgid, proc.pid)
            tracker.clear_verified()
            meta = json.loads(
                (scratch / startup.OWNER_MARKER).read_text())
            self.assertIsNone(meta["process_group"])
            self.assertIsNone(tracker.pgid)
            proc.terminate()
            proc.wait(timeout=2)
            self.assertTrue(startup.cleanup_scratch(scratch))

    def test_retire_pending_idle_returns_none(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup._ALLOCATED.discard, str(scratch))
            self.addCleanup(shutil.rmtree, scratch, True)
            tracker = startup._phase_tracker(scratch)
            self.assertIsNone(tracker.retire_pending())
            self.assertTrue(startup.cleanup_scratch(scratch))

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_retire_pending_retires_live_group_and_clears(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup._ALLOCATED.discard, str(scratch))
            self.addCleanup(shutil.rmtree, scratch, True)
            proc = subprocess.Popen(
                [shutil.which("sleep"), "30"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL, start_new_session=True)

            def kill_sleep():
                try:
                    proc.kill()
                except OSError:
                    pass
                try:
                    proc.wait(timeout=2)
                except Exception:
                    pass

            self.addCleanup(kill_sleep)
            tracker = startup._phase_tracker(scratch)
            tracker.track(proc.pid)
            record = tracker.retire_pending()
            self.assertTrue(record["retired"])
            self.assertEqual(record["pgid"], proc.pid)
            self.assertIsNone(tracker.pgid)
            meta = json.loads(
                (scratch / startup.OWNER_MARKER).read_text())
            self.assertIsNone(meta["process_group"])
            proc.wait(timeout=2)
            self.assertTrue(startup.cleanup_scratch(scratch))

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_retire_pending_failure_keeps_marker(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup._ALLOCATED.discard, str(scratch))
            self.addCleanup(shutil.rmtree, scratch, True)
            proc = subprocess.Popen(
                [shutil.which("sleep"), "30"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL, start_new_session=True)

            def kill_sleep():
                try:
                    proc.kill()
                except OSError:
                    pass
                try:
                    proc.wait(timeout=2)
                except Exception:
                    pass

            self.addCleanup(kill_sleep)
            tracker = startup._phase_tracker(scratch)
            tracker.track(proc.pid)
            with g28g_unverified_retirement():
                record = tracker.retire_pending()
            self.assertFalse(record["retired"])
            self.assertEqual(tracker.pgid, proc.pid)
            meta = json.loads(
                (scratch / startup.OWNER_MARKER).read_text())
            self.assertEqual(meta["process_group"], proc.pid)
            custody = startup._retained_custody(scratch, "test reason")
            self.assertEqual(custody["pgid"], proc.pid)
            self.assertEqual(custody["owner_pid"], os.getpid())
            self.assertEqual(custody["leader"]["pid"], proc.pid)
            self.assertIn("root_identity", custody)
            self.assertEqual(custody["reason"], "test reason")
            proc.wait(timeout=2)
            self.assertTrue(startup.cleanup_scratch(scratch))


class RegistrationRollbackTests(unittest.TestCase):
    """G28e registration rollback: exact-inode removal, never foreign."""

    def test_rollback_match_empty_removes(self):
        with tempfile.TemporaryDirectory() as parent:
            target = Path(parent) / "fresh-dir"
            target.mkdir()
            identity = startup._fs_identity(
                os.stat(target, follow_symlinks=False))
            self.assertIsNone(
                startup._rollback_new_directory(target, identity))
            self.assertFalse(os.path.lexists(target))

    def test_rollback_foreign_identity_refuses(self):
        with tempfile.TemporaryDirectory() as parent:
            target = Path(parent) / "victim"
            target.mkdir()
            (target / "keep.txt").write_text("x")
            self.addCleanup(shutil.rmtree, target, True)
            with self.assertRaisesRegex(ValueError, r"rollback failed"):
                startup._rollback_new_directory(target, [0, 0])
            self.assertTrue((target / "keep.txt").exists())

    def test_rollback_replaced_directory_refuses(self):
        with tempfile.TemporaryDirectory() as parent:
            target = Path(parent) / "replaced"
            target.mkdir()
            identity = startup._fs_identity(
                os.stat(target, follow_symlinks=False))
            shutil.rmtree(target)
            target.mkdir()
            self.addCleanup(shutil.rmtree, target, True)
            current = startup._fs_identity(
                os.stat(target, follow_symlinks=False))
            if current == identity:
                self.skipTest("filesystem reused the directory inode")
            with self.assertRaisesRegex(ValueError, r"rollback failed"):
                startup._rollback_new_directory(target, identity)
            self.assertTrue(target.is_dir())

    def test_rollback_nonempty_refuses(self):
        with tempfile.TemporaryDirectory() as parent:
            target = Path(parent) / "busy"
            target.mkdir()
            (target / "child.txt").write_text("x")
            self.addCleanup(shutil.rmtree, target, True)
            identity = startup._fs_identity(
                os.stat(target, follow_symlinks=False))
            with self.assertRaisesRegex(ValueError, r"rollback failed"):
                startup._rollback_new_directory(target, identity)
            self.assertTrue((target / "child.txt").exists())

    def test_rollback_file_and_missing_refuse(self):
        with tempfile.TemporaryDirectory() as parent:
            leaf = Path(parent) / "leaf.txt"
            leaf.write_text("x")
            identity = startup._fs_identity(
                os.stat(leaf, follow_symlinks=False))
            with self.assertRaisesRegex(ValueError, r"rollback failed"):
                startup._rollback_new_directory(leaf, identity)
            self.assertTrue(leaf.exists())
            missing = Path(parent) / "never-created"
            with self.assertRaisesRegex(ValueError, r"rollback failed"):
                startup._rollback_new_directory(missing, [1, 2])

    def test_make_scratch_marker_failure_rolls_back(self):
        with tempfile.TemporaryDirectory() as parent:
            before = sorted(os.listdir(parent))
            allocated_before = set(startup._ALLOCATED)
            real_write = startup._write_owner_marker

            def boom(root):
                raise ValueError("marker boom")

            startup._write_owner_marker = boom
            try:
                with self.assertRaises(ValueError) as ctx:
                    startup.make_scratch(parent)
            finally:
                startup._write_owner_marker = real_write
            message = str(ctx.exception)
            self.assertIn("scratch registration failed", message)
            self.assertIn("rolled back", message)
            self.assertEqual(sorted(os.listdir(parent)), before)
            self.assertEqual(set(startup._ALLOCATED), allocated_before)

    def test_explicit_path_marker_failure_rolls_back(self):
        with tempfile.TemporaryDirectory() as parent:
            explicit = Path(parent) / "explicit-keep"
            real_write = startup._write_owner_marker

            def boom(root):
                raise ValueError("marker boom")

            startup._write_owner_marker = boom
            try:
                with self.assertRaises(ValueError) as ctx:
                    with startup.owned_scratch(parent,
                                               path=explicit):
                        pass
            finally:
                startup._write_owner_marker = real_write
            message = str(ctx.exception)
            self.assertIn("scratch registration failed", message)
            self.assertIn("rolled back", message)
            self.assertFalse(os.path.lexists(explicit))

    def test_registration_rollback_failure_reports_pending(self):
        with tempfile.TemporaryDirectory() as parent:
            explicit = Path(parent) / "pending-keep"
            real_write = startup._write_owner_marker
            real_rollback = startup._rollback_new_directory

            def boom(root):
                raise ValueError("marker boom")

            def boom_rollback(path, identity):
                raise ValueError("rollback boom")

            startup._write_owner_marker = boom
            startup._rollback_new_directory = boom_rollback
            try:
                with self.assertRaises(ValueError) as ctx:
                    with startup.owned_scratch(parent,
                                               path=explicit):
                        pass
            finally:
                startup._write_owner_marker = real_write
                startup._rollback_new_directory = real_rollback
            message = str(ctx.exception)
            self.assertIn("scratch registration failed", message)
            self.assertIn("cleanup pending", message)
            self.assertTrue(explicit.is_dir())
            shutil.rmtree(explicit)


class RegistrationFailureTests(unittest.TestCase):
    """G28i registration failures: OSError/interrupt rollback, short writes."""

    def test_make_scratch_os_write_enospc_rolls_back(self):
        with tempfile.TemporaryDirectory() as parent:
            before = sorted(os.listdir(parent))
            allocated_before = set(startup._ALLOCATED)
            real_write = os.write

            def enospc(fd, data):
                raise OSError(errno.ENOSPC,
                              "injected disk-full marker write")

            os.write = enospc
            try:
                with self.assertRaises(ValueError) as ctx:
                    startup.make_scratch(parent)
            finally:
                os.write = real_write
            message = str(ctx.exception)
            self.assertIn("scratch registration failed", message)
            self.assertIn("rolled back", message)
            self.assertEqual(sorted(os.listdir(parent)), before)
            self.assertEqual(set(startup._ALLOCATED), allocated_before)

    def test_make_scratch_fsync_failure_rolls_back(self):
        with tempfile.TemporaryDirectory() as parent:
            before = sorted(os.listdir(parent))
            real_fsync = os.fsync

            def boom(fd):
                raise OSError(errno.EIO, "injected fsync failure")

            os.fsync = boom
            try:
                with self.assertRaises(ValueError) as ctx:
                    startup.make_scratch(parent)
            finally:
                os.fsync = real_fsync
            message = str(ctx.exception)
            self.assertIn("scratch registration failed", message)
            self.assertIn("rolled back", message)
            self.assertEqual(sorted(os.listdir(parent)), before)

    def test_explicit_path_oserror_rolls_back(self):
        with tempfile.TemporaryDirectory() as parent:
            explicit = Path(parent) / "explicit-enospc"
            real_write = os.write

            def enospc(fd, data):
                raise OSError(errno.ENOSPC,
                              "injected disk-full marker write")

            os.write = enospc
            try:
                with self.assertRaises(ValueError) as ctx:
                    with startup.owned_scratch(parent,
                                               path=explicit):
                        pass
            finally:
                os.write = real_write
            message = str(ctx.exception)
            self.assertIn("scratch registration failed", message)
            self.assertIn("rolled back", message)
            self.assertFalse(os.path.lexists(explicit))

    def test_interrupted_registration_rolls_back_and_reraises(self):
        with tempfile.TemporaryDirectory() as parent:
            before = sorted(os.listdir(parent))
            real_write = os.write

            def die(fd, data):
                raise KeyboardInterrupt("injected")

            os.write = die
            try:
                with self.assertRaises(KeyboardInterrupt):
                    startup.make_scratch(parent)
            finally:
                os.write = real_write
            self.assertEqual(sorted(os.listdir(parent)), before)

    def test_interrupted_registration_rollback_failure_reports_pending(self):
        with tempfile.TemporaryDirectory() as parent:
            explicit = Path(parent) / "pending-interrupt"
            real_write_marker = startup._write_owner_marker

            def die(root):
                (Path(root) / "debris.txt").write_text("debris")
                raise KeyboardInterrupt("injected")

            startup._write_owner_marker = die
            try:
                with self.assertRaises(ValueError) as ctx:
                    with startup.owned_scratch(parent,
                                               path=explicit):
                        pass
            finally:
                startup._write_owner_marker = real_write_marker
            message = str(ctx.exception)
            self.assertIn("scratch registration failed", message)
            self.assertIn("cleanup pending", message)
            self.assertTrue((explicit / "debris.txt").exists())
            shutil.rmtree(explicit)

    def test_short_marker_writes_complete(self):
        with tempfile.TemporaryDirectory() as parent:
            real_write = os.write

            def short(fd, data):
                return real_write(fd, bytes(data[:1]))

            os.write = short
            try:
                path = startup.make_scratch(parent)
            finally:
                os.write = real_write
            try:
                meta = startup._read_owner(path, allow_live_group=True)
                self.assertEqual(meta["kind"], startup.OWNER_KIND)
                self.assertEqual(meta["root"], str(path))
            finally:
                self.assertTrue(startup.cleanup_scratch(str(path)))
            self.assertFalse(os.path.lexists(path))


def g28m_fresh_root(parent, before, prefix="can-startup-attr-"):
    """Locate the single newly allocated scratch root under parent."""
    fresh = [name for name in os.listdir(parent)
             if name not in before and name.startswith(prefix)]
    if len(fresh) != 1:
        raise AssertionError(f"expected one fresh root, found {fresh}")
    return Path(parent) / fresh[0]


class ReplacementPreservationTests(unittest.TestCase):
    """G28m failed registration: replaced marker/root survive, pending told."""

    def test_write_failure_preserves_replaced_marker(self):
        with tempfile.TemporaryDirectory() as parent:
            before = sorted(os.listdir(parent))
            proof = {}
            real_write = os.write

            def fail_after_replace(fd, data):
                os.write = real_write
                try:
                    root = g28m_fresh_root(parent, before)
                    marker = root / startup.OWNER_MARKER
                    os.unlink(marker)
                    marker.write_bytes(
                        b'{"kind": "foreign-replacement"}')
                    info = os.stat(marker, follow_symlinks=False)
                    proof["identity"] = (info.st_dev, info.st_ino)
                finally:
                    os.write = fail_after_replace
                raise OSError(errno.ENOSPC,
                              "injected failure after marker replacement")

            os.write = fail_after_replace
            try:
                with self.assertRaises(ValueError) as ctx:
                    startup.make_scratch(parent)
            finally:
                os.write = real_write
            message = str(ctx.exception)
            self.assertIn("cleanup pending", message)
            self.assertNotIn("rolled back", message)
            root = g28m_fresh_root(parent, before)
            marker = root / startup.OWNER_MARKER
            self.assertTrue(root.is_dir())
            info = os.stat(marker, follow_symlinks=False)
            self.assertEqual((info.st_dev, info.st_ino),
                             proof["identity"])
            self.assertEqual(marker.read_bytes(),
                             b'{"kind": "foreign-replacement"}')
            shutil.rmtree(root)

    def test_fsync_failure_preserves_replaced_marker(self):
        with tempfile.TemporaryDirectory() as parent:
            before = sorted(os.listdir(parent))
            proof = {}
            real_fsync = os.fsync

            def fail_after_replace(fd):
                root = g28m_fresh_root(parent, before)
                marker = root / startup.OWNER_MARKER
                os.unlink(marker)
                marker.write_bytes(b'{"kind": "foreign-replacement"}')
                info = os.stat(marker, follow_symlinks=False)
                proof["identity"] = (info.st_dev, info.st_ino)
                raise OSError(errno.EIO, "injected fsync failure")

            os.fsync = fail_after_replace
            try:
                with self.assertRaises(ValueError) as ctx:
                    startup.make_scratch(parent)
            finally:
                os.fsync = real_fsync
            message = str(ctx.exception)
            self.assertIn("cleanup pending", message)
            self.assertNotIn("rolled back", message)
            root = g28m_fresh_root(parent, before)
            marker = root / startup.OWNER_MARKER
            info = os.stat(marker, follow_symlinks=False)
            self.assertEqual((info.st_dev, info.st_ino),
                             proof["identity"])
            shutil.rmtree(root)

    def test_interrupted_registration_preserves_replacement(self):
        with tempfile.TemporaryDirectory() as parent:
            before = sorted(os.listdir(parent))
            real_fsync = os.fsync

            def die_after_replace(fd):
                root = g28m_fresh_root(parent, before)
                marker = root / startup.OWNER_MARKER
                os.unlink(marker)
                marker.write_bytes(b'{"kind": "foreign-replacement"}')
                raise KeyboardInterrupt("injected")

            os.fsync = die_after_replace
            try:
                with self.assertRaises(ValueError) as ctx:
                    startup.make_scratch(parent)
            finally:
                os.fsync = real_fsync
            message = str(ctx.exception)
            self.assertIn("cleanup pending", message)
            root = g28m_fresh_root(parent, before)
            marker = root / startup.OWNER_MARKER
            self.assertEqual(marker.read_bytes(),
                             b'{"kind": "foreign-replacement"}')
            shutil.rmtree(root)

    def test_explicit_path_replacement_preserved(self):
        with tempfile.TemporaryDirectory() as parent:
            explicit = Path(parent) / "explicit-replaced"
            real_write = os.write

            def fail_after_replace(fd, data):
                os.write = real_write
                try:
                    marker = explicit / startup.OWNER_MARKER
                    os.unlink(marker)
                    marker.write_bytes(
                        b'{"kind": "foreign-replacement"}')
                finally:
                    os.write = fail_after_replace
                raise OSError(errno.ENOSPC,
                              "injected failure after marker replacement")

            os.write = fail_after_replace
            try:
                with self.assertRaises(ValueError) as ctx:
                    with startup.owned_scratch(parent,
                                               path=explicit):
                        pass
            finally:
                os.write = real_write
            message = str(ctx.exception)
            self.assertIn("cleanup pending", message)
            self.assertNotIn("rolled back", message)
            marker = explicit / startup.OWNER_MARKER
            self.assertEqual(marker.read_bytes(),
                             b'{"kind": "foreign-replacement"}')
            shutil.rmtree(explicit)

    def test_root_replacement_preserved(self):
        with tempfile.TemporaryDirectory() as parent:
            before = sorted(os.listdir(parent))
            real_write = os.write

            def fail_after_replace(fd, data):
                os.write = real_write
                try:
                    root = g28m_fresh_root(parent, before)
                    detached = Path(parent) / "detached-original"
                    os.rename(root, detached)
                    root.mkdir()
                    (root / startup.OWNER_MARKER).write_bytes(
                        b'{"kind": "foreign-root-marker"}')
                    (root / "sentinel.txt").write_text("foreign")
                finally:
                    os.write = fail_after_replace
                raise OSError(errno.ENOSPC,
                              "injected failure after root replacement")

            os.write = fail_after_replace
            try:
                with self.assertRaises(ValueError) as ctx:
                    startup.make_scratch(parent)
            finally:
                os.write = real_write
            message = str(ctx.exception)
            self.assertIn("cleanup pending", message)
            root = g28m_fresh_root(parent, before)
            self.assertEqual((root / startup.OWNER_MARKER).read_bytes(),
                             b'{"kind": "foreign-root-marker"}')
            self.assertEqual((root / "sentinel.txt").read_text(),
                             "foreign")
            shutil.rmtree(root)
            shutil.rmtree(Path(parent) / "detached-original",
                          ignore_errors=True)


class FailedRetirementCustodyTests(unittest.TestCase):
    """G28e failed retirement: marker kept, guarded recovery only."""

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_live_recorded_group_blocks_cleanup(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup._ALLOCATED.discard, str(scratch))
            self.addCleanup(shutil.rmtree, scratch, True)
            proc = subprocess.Popen(
                [shutil.which("sleep"), "30"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL, start_new_session=True)

            def kill_sleep():
                try:
                    proc.kill()
                except OSError:
                    pass
                try:
                    proc.wait(timeout=2)
                except Exception:
                    pass

            self.addCleanup(kill_sleep)
            tracker = startup._phase_tracker(scratch)
            tracker.track(proc.pid)
            with self.assertRaisesRegex(
                    ValueError,
                    r"scratch ownership invalid: active_group"):
                startup.cleanup_scratch(scratch)
            self.assertTrue(
                (scratch / startup.OWNER_MARKER).exists())
            self.assertIn(str(scratch), startup._ALLOCATED)
            with self.assertRaisesRegex(
                    ValueError,
                    r"scratch ownership invalid: active_owner"):
                startup.recover_scratch(scratch)
            record = tracker.retire_pending()
            self.assertTrue(record["retired"])
            proc.wait(timeout=2)
            self.assertTrue(startup.cleanup_scratch(scratch))
            self.assertFalse(scratch.exists())

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_recover_dead_owner_live_group_with_leader(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup._ALLOCATED.discard, str(scratch))
            self.addCleanup(shutil.rmtree, scratch, True)
            proc = subprocess.Popen(
                [shutil.which("sleep"), "30"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL, start_new_session=True)

            def kill_sleep():
                try:
                    proc.kill()
                except OSError:
                    pass
                try:
                    proc.wait(timeout=2)
                except Exception:
                    pass

            self.addCleanup(kill_sleep)
            dead = g28g_dead_pid()
            g28g_rewrite_marker(
                scratch, pid=dead, process_start=None,
                process_group=proc.pid,
                group_leader={"pid": proc.pid,
                              "process_start":
                                  startup._process_start(proc.pid)})
            self.assertTrue(startup._pid_alive(proc.pid))
            self.assertTrue(startup.recover_scratch(scratch))
            self.assertFalse(scratch.exists())
            proc.wait(timeout=2)
            self.assertFalse(startup._pid_alive(proc.pid))

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_stale_pgid_retires_without_signal(self):
        dead = g28g_dead_pid()
        if startup._pid_alive(dead, group=True):
            self.skipTest(f"pid {dead} was reused")
        result = startup._kill_process_group(dead, term_grace=0.2,
                                             kill_grace=0.1)
        self.assertTrue(result["retired"])
        self.assertIsNone(result["refused"])

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_leader_mismatch_refuses_without_signalling(self):
        proc = subprocess.Popen(
            [shutil.which("sleep"), "30"],
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
            start_new_session=True)

        def kill_sleep():
            try:
                proc.kill()
            except OSError:
                pass
            try:
                proc.wait(timeout=2)
            except Exception:
                pass

        self.addCleanup(kill_sleep)
        self.assertTrue(startup._pid_alive(proc.pid))
        foreign = {"pid": os.getpid(),
                   "process_start":
                       startup._process_start(os.getpid())}
        result = startup._kill_process_group(
            proc.pid, term_grace=0.2, kill_grace=0.1,
            leader=foreign)
        self.assertFalse(result["retired"])
        self.assertIn("owner-mismatch", result["refused"])
        self.assertTrue(startup._pid_alive(proc.pid),
                        msg="mismatched kill signalled a live group")
        dead = g28g_dead_pid()
        result = startup._kill_process_group(
            proc.pid, term_grace=0.2, kill_grace=0.1,
            leader={"pid": dead, "process_start": None})
        self.assertFalse(result["retired"])
        self.assertIn("owner-mismatch", result["refused"])
        self.assertTrue(startup._pid_alive(proc.pid),
                        msg="stale-leader kill signalled a live group")
        result = startup._kill_process_group(
            proc.pid, term_grace=0.2, kill_grace=0.1,
            leader="not-a-record")
        self.assertFalse(result["retired"])
        self.assertIn("owner-mismatch", result["refused"])
        self.assertTrue(startup._pid_alive(proc.pid),
                        msg="malformed-leader kill signalled a group")
        proc.terminate()
        proc.wait(timeout=2)

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_recover_stale_leader_refuses_and_retains(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup._ALLOCATED.discard, str(scratch))
            self.addCleanup(shutil.rmtree, scratch, True)
            proc = subprocess.Popen(
                [shutil.which("sleep"), "30"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL, start_new_session=True)

            def kill_sleep():
                try:
                    proc.kill()
                except OSError:
                    pass
                try:
                    proc.wait(timeout=2)
                except Exception:
                    pass

            self.addCleanup(kill_sleep)
            dead = g28g_dead_pid()
            g28g_rewrite_marker(
                scratch, pid=dead, process_start=None,
                process_group=proc.pid,
                group_leader={"pid": os.getpid(),
                              "process_start":
                                  startup._process_start(os.getpid())})
            with self.assertRaisesRegex(
                    ValueError, r"scratch cleanup failed.*owner-mismatch"):
                startup.recover_scratch(scratch)
            self.assertTrue(scratch.exists())
            self.assertTrue(startup._pid_alive(proc.pid))
            proc.terminate()
            proc.wait(timeout=2)
            self.assertTrue(startup.recover_scratch(scratch))
            self.assertFalse(scratch.exists())


class KeeperCustodyTests(unittest.TestCase):
    """G28e keeper custody: live consumer holds, exited never protects."""

    def test_take_release_idempotent(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup._ALLOCATED.discard, str(scratch))
            self.addCleanup(shutil.rmtree, scratch, True)
            first = startup.take_custody(scratch)
            self.assertEqual(first["keeper"]["pid"], os.getpid())
            self.assertIn("process_start", first["keeper"])
            self.assertIn("asserted_at", first["keeper"])
            second = startup.take_custody(scratch)
            self.assertEqual(second["keeper"]["pid"], os.getpid())
            released = startup.release_custody(scratch)
            self.assertIsNone(released["keeper"])
            with self.assertRaisesRegex(
                    ValueError,
                    r"scratch ownership invalid: active_keeper"):
                startup.release_custody(scratch)
            startup.take_custody(scratch)
            startup.release_custody(scratch)
            self.assertTrue(startup.cleanup_scratch(scratch))

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_live_foreign_keeper_blocks_cleanup_and_recover(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup._ALLOCATED.discard, str(scratch))
            self.addCleanup(shutil.rmtree, scratch, True)
            proc = subprocess.Popen(
                [shutil.which("sleep"), "30"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL)

            def kill_sleep():
                try:
                    proc.kill()
                except OSError:
                    pass
                try:
                    proc.wait(timeout=2)
                except Exception:
                    pass

            self.addCleanup(kill_sleep)
            g28g_rewrite_marker(
                scratch, keeper={"pid": proc.pid,
                                 "process_start":
                                     startup._process_start(proc.pid),
                                 "asserted_at": 0})
            with self.assertRaisesRegex(
                    ValueError,
                    r"scratch ownership invalid: active_keeper"):
                startup.cleanup_scratch(scratch)
            with self.assertRaisesRegex(
                    ValueError,
                    r"scratch ownership invalid: active_keeper"):
                startup.recover_scratch(scratch)
            self.assertTrue(scratch.exists())
            proc.terminate()
            proc.wait(timeout=2)
            self.assertTrue(startup.cleanup_scratch(scratch))
            self.assertFalse(scratch.exists())

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_dead_keeper_never_blocks_recovery(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup._ALLOCATED.discard, str(scratch))
            self.addCleanup(shutil.rmtree, scratch, True)
            dead = g28g_dead_pid()
            dead_keeper = g28g_dead_pid()
            g28g_rewrite_marker(
                scratch, pid=dead, process_start=None,
                keeper={"pid": dead_keeper, "process_start": None,
                        "asserted_at": 0})
            self.assertTrue(startup.recover_scratch(scratch))
            self.assertFalse(scratch.exists())

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_take_refuses_live_foreign_owner_and_keeper(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup._ALLOCATED.discard, str(scratch))
            self.addCleanup(shutil.rmtree, scratch, True)
            proc = subprocess.Popen(
                [shutil.which("sleep"), "30"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL)

            def kill_sleep():
                try:
                    proc.kill()
                except OSError:
                    pass
                try:
                    proc.wait(timeout=2)
                except Exception:
                    pass

            self.addCleanup(kill_sleep)
            start = startup._process_start(proc.pid)
            g28g_rewrite_marker(scratch, pid=proc.pid,
                                process_start=start)
            with self.assertRaisesRegex(
                    ValueError,
                    r"scratch ownership invalid: active_owner"):
                startup.take_custody(scratch)
            dead = g28g_dead_pid()
            g28g_rewrite_marker(
                scratch, pid=dead, process_start=None,
                keeper={"pid": proc.pid, "process_start": start,
                        "asserted_at": 0})
            with self.assertRaisesRegex(
                    ValueError,
                    r"scratch ownership invalid: active_keeper"):
                startup.take_custody(scratch)
            proc.terminate()
            proc.wait(timeout=2)
            taken = startup.take_custody(scratch)
            self.assertEqual(taken["keeper"]["pid"], os.getpid())
            startup.release_custody(scratch)
            self.assertTrue(startup.recover_scratch(scratch))

    def test_schema_one_marker_read_compat(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup._ALLOCATED.discard, str(scratch))
            self.addCleanup(shutil.rmtree, scratch, True)
            marker = scratch / startup.OWNER_MARKER
            meta = json.loads(marker.read_text())
            meta["schema_version"] = 1
            del meta["keeper"]
            del meta["group_leader"]
            marker.write_text(json.dumps(meta))
            read = startup._read_owner(scratch)
            self.assertIsNone(read["keeper"])
            self.assertIsNone(read["group_leader"])
            taken = startup.take_custody(scratch)
            self.assertEqual(taken["keeper"]["pid"], os.getpid())
            startup.release_custody(scratch)
            self.assertTrue(startup.cleanup_scratch(scratch))

    def test_keep_transfer_sets_live_keeper(self):
        with tempfile.TemporaryDirectory() as parent:
            keeper = None
            with startup.owned_scratch(parent, keep=True) as scratch:
                (scratch / "kept.txt").write_text("x")
                keeper = scratch
            self.addCleanup(startup._ALLOCATED.discard, str(keeper))
            self.addCleanup(shutil.rmtree, keeper, True)
            meta = json.loads(
                (keeper / startup.OWNER_MARKER).read_text())
            self.assertEqual(meta["state"], "retained")
            self.assertTrue(meta["transferred"])
            self.assertEqual(meta["keeper"]["pid"], os.getpid())
            startup.release_custody(keeper)
            self.assertTrue(startup.cleanup_scratch(keeper))
            self.assertFalse(keeper.exists())


class IdentityCompleteTests(unittest.TestCase):
    """G28e complete identities: accept whole, reject gaps at run level."""

    def test_accept_complete_synthetic_record(self):
        self.assertTrue(startup.check_identity_complete(
            g28g_complete_identity()))

    def test_reject_missing_tooling(self):
        def drop_versions(inv):
            inv.pop("tool_versions")

        def blank_bun(inv):
            inv["tool_versions"]["bun"] = " "

        def null_typescript(inv):
            inv["tool_versions"]["typescript"] = None

        def drop_revision(inv):
            inv["builtins"].pop("bun_revision")

        def drop_go_size(inv):
            inv["executables"]["go"].pop("size")

        def null_node(inv):
            inv["executables"]["node"] = None

        cases = {"tool_versions missing": drop_versions,
                 "bun blank": blank_bun,
                 "typescript null": null_typescript,
                 "bun_revision missing": drop_revision,
                 "go executable incomplete": drop_go_size,
                 "node executable null": null_node}
        for name, mutate in cases.items():
            with self.subTest(name=name):
                inv = g28g_complete_identity()
                mutate(inv)
                with self.assertRaisesRegex(
                        ValueError, r"identity incomplete"):
                    startup.check_identity_complete(inv)

    def test_reject_missing_drivers_sources(self):
        def drop_driver(inv):
            inv["drivers"].pop("drivers/runtime.py")

        def null_probe(inv):
            inv["drivers"]["startup-attribution.ts"] = None

        def drop_facade(inv):
            inv["sources"].pop("startup-facade.ts")

        def null_state(inv):
            inv["sources"]["state.ts"] = None

        def drop_state_bytes(inv):
            inv["sources"]["state.ts"].pop("bytes")

        cases = {"driver missing": drop_driver,
                 "probe driver null": null_probe,
                 "facade missing": drop_facade,
                 "state null": null_state,
                 "state bytes missing": drop_state_bytes}
        for name, mutate in cases.items():
            with self.subTest(name=name):
                inv = g28g_complete_identity()
                mutate(inv)
                with self.assertRaisesRegex(
                        ValueError, r"identity incomplete"):
                    startup.check_identity_complete(inv)

    def test_reject_missing_inputs(self):
        def empty_fixture(inv):
            inv["inputs"]["fixture"] = []

        def empty_packages(inv):
            inv["inputs"]["emitter"]["packages"] = []

        def fileless_package(inv):
            inv["inputs"]["emitter"]["packages"] = [
                {"package": "x", "files": []}]

        def drop_go_mod(inv):
            inv["inputs"]["emitter"].pop("go_mod")

        def null_binary(inv):
            inv["inputs"]["emitter"]["binary"] = None

        def empty_runtime_ts(inv):
            inv["runtime_ts"]["files"] = []

        def drop_adapter(inv):
            inv["emitted"].pop("adapter")

        def empty_emitted(inv):
            inv["emitted"]["packages"] = []

        cases = {"fixture empty": empty_fixture,
                 "emitter packages empty": empty_packages,
                 "emitter package fileless": fileless_package,
                 "go_mod missing": drop_go_mod,
                 "binary null": null_binary,
                 "runtime_ts empty": empty_runtime_ts,
                 "adapter missing": drop_adapter,
                 "emitted packages empty": empty_emitted}
        for name, mutate in cases.items():
            with self.subTest(name=name):
                inv = g28g_complete_identity()
                mutate(inv)
                with self.assertRaisesRegex(
                        ValueError, r"identity incomplete"):
                    startup.check_identity_complete(inv)

    def test_reject_missing_prepared(self):
        def empty_ordinary(inv):
            inv["prepared"]["ordinary"]["files"] = []

        def null_diagnostic(inv):
            inv["prepared"]["diagnostic"]["files"] = None

        def null_bundle(inv):
            inv["prepared"]["bundle"] = None

        def drop_bundle_bytes(inv):
            inv["prepared"]["bundle"].pop("bytes")

        cases = {"ordinary empty": empty_ordinary,
                 "diagnostic null": null_diagnostic,
                 "bundle null": null_bundle,
                 "bundle bytes missing": drop_bundle_bytes}
        for name, mutate in cases.items():
            with self.subTest(name=name):
                inv = g28g_complete_identity()
                mutate(inv)
                with self.assertRaisesRegex(
                        ValueError, r"identity incomplete"):
                    startup.check_identity_complete(inv)

    def test_reject_missing_resolved(self):
        def uncomputed(inv):
            inv["resolved"]["computed"] = False

        def empty_roots(inv):
            inv["resolved"]["roots"] = []

        def zero_files(inv):
            inv["resolved"]["reachable_files"] = 0

        def truncated(inv):
            inv["resolved"]["truncated"] = True

        def holding_unresolved(inv):
            inv["resolved"]["unresolved"] = [
                {"specifier": "x", "reason": "y"}]

        def builtin_no_runtime(inv):
            inv["resolved"]["reachable_bare"][0]["resolution"] = {
                "status": "builtin"}

        def file_status_rejected(inv):
            inv["resolved"]["reachable_bare"][1]["resolution"] = {
                "status": "file",
                "path": "/tmp/node_modules/can-pkg/i.js",
                "realpath": "/tmp/node_modules/can-pkg/i.js",
                "sha256": "t" * 64, "bytes": 33}

        def unsupported_contexts_rejected(inv):
            inv["resolved"]["reachable_bare"][1]["resolution"] = {
                "status": "unsupported",
                "reason": "reachable file dependency lacks "
                          "importer-specific ESM closure",
                "contexts": [
                    {"importer": "/tmp/scratch/a.js",
                     "builtin": False,
                     "resolved": "/tmp/a/node_modules/p/i.js",
                     "error": None},
                    {"importer": "/tmp/scratch/b.js",
                     "builtin": False,
                     "resolved": "/tmp/b/node_modules/p/i.js",
                     "error": None}]}

        def failed_entry(inv):
            inv["resolved"]["reachable_bare"][0]["resolution"] = {
                "status": "failed", "error": "nope"}

        def ambiguous_entry(inv):
            inv["resolved"]["reachable_bare"][1]["resolution"] = {
                "status": "ambiguous", "resolved": "/tmp/x"}

        def missing_resolution(inv):
            inv["resolved"]["reachable_bare"][0].pop("resolution")

        cases = {"computed false": uncomputed,
                 "roots empty": empty_roots,
                 "reachable_files zero": zero_files,
                 "walk truncated": truncated,
                 "unresolved holding": holding_unresolved,
                 "builtin lacks runtime": builtin_no_runtime,
                 "complete-looking file rejected": file_status_rejected,
                 "unsupported contexts rejected":
                     unsupported_contexts_rejected,
                 "required failed": failed_entry,
                 "required ambiguous": ambiguous_entry,
                 "resolution missing": missing_resolution}
        for name, mutate in cases.items():
            with self.subTest(name=name):
                inv = g28g_complete_identity()
                mutate(inv)
                with self.assertRaisesRegex(
                        ValueError, r"identity incomplete"):
                    startup.check_identity_complete(inv)

    def test_reject_missing_dependencies(self):
        def null_modules(inv):
            inv["dependencies"]["node_modules"] = None

        def drop_package(inv):
            inv["dependencies"].pop("package_json")

        def blank_lock(inv):
            inv["dependencies"]["bun_lock"] = ""

        cases = {"node_modules null": null_modules,
                 "package_json missing": drop_package,
                 "bun_lock blank": blank_lock}
        for name, mutate in cases.items():
            with self.subTest(name=name):
                inv = g28g_complete_identity()
                mutate(inv)
                with self.assertRaisesRegex(
                        ValueError, r"identity incomplete"):
                    startup.check_identity_complete(inv)

    def test_reject_non_mapping(self):
        for bad in ([], None, "x"):
            with self.subTest(bad=bad):
                with self.assertRaisesRegex(
                        ValueError, r"identity incomplete"):
                    startup.check_identity_complete(bad)

    def test_empty_equal_dicts_rejected_at_run_level(self):
        self.assertTrue(startup.check_identity_drift({}, {}))
        with self.assertRaisesRegex(
                ValueError, r"identity incomplete"):
            startup.check_identity_complete({})
        with self.assertRaisesRegex(
                ValueError, r"identity incomplete"):
            startup.check_identity_complete({"resolved": {}})

    def test_drift_rejects_nested_identity_change(self):
        before = g28g_complete_identity()
        identical = json.loads(json.dumps(before))
        self.assertTrue(
            startup.check_identity_drift(before, identical))
        with self.subTest(name="binary bytes"):
            after = json.loads(json.dumps(before))
            after["inputs"]["emitter"]["binary"]["bytes"] = 1
            with self.assertRaisesRegex(
                    ValueError, r"prepared identity drift"):
                startup.check_identity_drift(before, after)
        with self.subTest(name="runtime revision"):
            after = json.loads(json.dumps(before))
            after["resolved"]["reachable_bare"][1][
                "resolution"]["runtime"]["bun_revision"] = "zzz"
            with self.assertRaisesRegex(
                    ValueError, r"prepared identity drift"):
                startup.check_identity_drift(before, after)
        with self.subTest(name="reachable count"):
            after = json.loads(json.dumps(before))
            after["resolved"]["reachable_files"] = 5
            with self.assertRaisesRegex(
                    ValueError, r"prepared identity drift"):
                startup.check_identity_drift(before, after)


class ResolutionImporterTests(unittest.TestCase):
    """G28e importer-context resolution: real bun, paired importers."""

    def test_classify_pairs_importers_with_statuses(self):
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory) / "pkg.js"
            target.write_bytes(b"export const x = 1;\n")
            runtime = {"bun": "1.4.2", "bun_revision": "abc123"}
            pairs = [("node:fs", ["a.js", "b.js"]),
                     ("can-pkg", ["c.js"]),
                     ("missing-pkg", ["d.js"]),
                     ("dir-pkg", ["e.js"])]
            answers = [{"ok": True, "resolved": "node:fs"},
                       {"ok": True, "resolved": str(target)},
                       {"ok": False, "error": "Cannot find module"},
                       {"ok": True, "resolved": directory}]
            entries = startup._classify_resolutions(
                pairs, answers, runtime)
            self.assertEqual(
                [entry["specifier"] for entry in entries],
                ["node:fs", "can-pkg", "missing-pkg", "dir-pkg"])
            self.assertEqual(entries[0]["importers"], ["a.js", "b.js"])
            self.assertEqual(entries[0]["resolution"]["status"],
                             "builtin")
            self.assertEqual(entries[1]["importers"], ["c.js"])
            self.assertEqual(entries[1]["resolution"]["status"], "file")
            self.assertEqual(entries[1]["resolution"]["sha256"],
                             hashlib.sha256(
                                 target.read_bytes()).hexdigest())
            self.assertEqual(entries[1]["resolution"]["bytes"],
                             target.stat().st_size)
            self.assertEqual(entries[2]["resolution"]["status"],
                             "failed")
            self.assertIn("Cannot find",
                          entries[2]["resolution"]["error"])
            self.assertEqual(entries[3]["resolution"]["status"],
                             "ambiguous")
            fallback = startup._classify_resolutions(
                [("fs", ["f.js"])], [{"ok": False, "error": "nope"}],
                runtime, builtins={"fs"})
            self.assertEqual(fallback[0]["resolution"]["status"],
                             "builtin")
            self.assertEqual(fallback[0]["resolution"]["via"],
                             "runtime builtinModules")

    @unittest.skipUnless(shutil.which("bun"), "bun is required")
    def test_bun_resolve_respects_importer_directory(self):
        with tempfile.TemporaryDirectory() as directory:
            scratch = startup.make_scratch(directory)
            self.addCleanup(startup.cleanup_scratch, str(scratch))
            tracker = startup._phase_tracker(scratch)
            visible = scratch / "sub"
            hidden = Path(directory) / "elsewhere"
            package = (scratch / "node_modules"
                       / "can-g28g-probe-pkg")
            package.mkdir(parents=True)
            (package / "package.json").write_text(
                json.dumps({"name": "can-g28g-probe-pkg",
                            "main": "index.js"}))
            (package / "index.js").write_text(
                "export const x = 1;\n")
            visible.mkdir(parents=True)
            inner = visible / "importer.js"
            inner.write_text("import 'can-g28g-probe-pkg';\n")
            hidden.mkdir(parents=True)
            outer = hidden / "importer.js"
            outer.write_text("import 'can-g28g-probe-pkg';\n")
            answers = startup._resolve_bare_with_bun(
                scratch, [("can-g28g-probe-pkg", str(inner)),
                          ("can-g28g-probe-pkg", str(outer))],
                tracker)
            self.assertIsNotNone(answers,
                                 msg="bun resolve probe failed")
            self.assertEqual(len(answers), 2)
            self.assertTrue(answers[0]["ok"])
            self.assertIn("can-g28g-probe-pkg", answers[0]["resolved"])
            self.assertFalse(
                answers[1]["ok"],
                msg="wrong importer must not resolve the package")


def g28k_write_closure_tree(scratch, style):
    """Synthetic prepared tree with two importer contexts (a/ and b/)."""
    scratch = Path(scratch)
    ordinary = scratch / "generated-js"
    (ordinary / "a").mkdir(parents=True)
    (ordinary / "b").mkdir(parents=True)
    (ordinary / "startup-facade.js").write_text(
        'import "./a/importer.js";\nimport "./b/importer.js";\n')
    (ordinary / "startup-minimal.js").write_text(
        "export const x = 1;\n")
    if style == "file":
        for context in ("a", "b"):
            (ordinary / context / "importer.js").write_text(
                'import("probe-dep");\n')
            package = (ordinary / context / "node_modules"
                       / "probe-dep")
            package.mkdir(parents=True)
            (package / "package.json").write_text(
                json.dumps({"name": "probe-dep",
                            "main": "index.js"}))
            (package / "index.js").write_text(
                f"export const owner = '{context}';\n")
    else:
        for context in ("a", "b"):
            (ordinary / context / "importer.js").write_text(
                'import("node:fs");\n')
    diagnostic = scratch / "generated-js-diag"
    diagnostic.mkdir(parents=True)
    (diagnostic / "startup-facade.js").write_text(
        "export const x = 1;\n")
    generated = scratch / "generated"
    (generated / "program").mkdir(parents=True)
    (generated / "startup-facade.ts").write_text("facade\n")
    (generated / "startup-minimal.ts").write_text("minimal\n")
    (generated / "program" / "state.ts").write_text(
        "export const x = 1;\n")
    (generated / "generated-adapter.ts").write_text("adapter\n")
    (generated / "packages").mkdir(parents=True)
    (generated / "packages" / "p.ts").write_text("package\n")
    probe = scratch / "instrument-src" / "program"
    probe.mkdir(parents=True)
    (probe / "startup-probe.ts").write_text("probe\n")
    (scratch / "startup-launcher.js").write_text("launcher\n")
    (scratch / "startup-ordinary-bundle.js").write_text(
        "export const x = 1;\n")
    (scratch / "perfemit").write_bytes(b"fake-elf")
    real_modules = scratch.parent / "real-modules"
    real_modules.mkdir(exist_ok=True)
    (scratch / "node_modules").symlink_to(
        real_modules, target_is_directory=True)
    return ordinary


class ReachableClosureTests(unittest.TestCase):
    """G28k reachable closure: per-importer contexts, builtin-only complete."""

    def test_classify_reachable_all_builtin(self):
        runtime = {"bun": "1.4.2", "bun_revision": "abc123"}
        entry = startup._classify_reachable(
            "node:fs", ["a.js", "b.js"],
            [{"ok": True, "resolved": "node:fs"},
             {"ok": True, "resolved": "node:fs"}],
            runtime, {"fs", "path"})
        self.assertEqual(entry["specifier"], "node:fs")
        self.assertEqual(entry["importers"], ["a.js", "b.js"])
        self.assertEqual(entry["resolution"]["status"], "builtin")
        self.assertEqual(entry["resolution"]["runtime"], runtime)

    def test_classify_reachable_mixed_contexts_unsupported(self):
        runtime = {"bun": "1.4.2", "bun_revision": "abc123"}
        entry = startup._classify_reachable(
            "probe-dep", ["a/importer.js", "b/importer.js"],
            [{"ok": True, "resolved": "/x/a/node_modules/probe-dep/i.js"},
             {"ok": False, "error": "Cannot find module"}],
            runtime, set())
        self.assertEqual(entry["specifier"], "probe-dep")
        self.assertEqual(entry["resolution"]["status"],
                         "unsupported")
        contexts = entry["resolution"]["contexts"]
        self.assertEqual(len(contexts), 2)
        self.assertEqual(contexts[0]["importer"], "a/importer.js")
        self.assertIn("a/node_modules", contexts[0]["resolved"])
        self.assertIn("Cannot find", contexts[1]["error"])

    @unittest.skipUnless(shutil.which("bun") and shutil.which("go"),
                         "bun and go are required")
    def test_two_importer_file_contexts_rejected(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup.cleanup_scratch, str(scratch))
            tracker = startup._phase_tracker(scratch)
            ordinary = g28k_write_closure_tree(scratch, "file")
            inv = startup.collect_prepared_identities(scratch, REPO, tracker)
            specs = {entry["specifier"]
                     for entry in inv["resolved"]["reachable_bare"]}
            self.assertIn("probe-dep", specs)
            entry = [entry for entry in
                     inv["resolved"]["reachable_bare"]
                     if entry["specifier"] == "probe-dep"][0]
            self.assertEqual(entry["importers"],
                             [str((ordinary / "a" / "importer.js").resolve()),
                              str((ordinary / "b" / "importer.js").resolve())])
            self.assertEqual(entry["resolution"]["status"],
                             "unsupported")
            paths = sorted(context["resolved"] for context in
                           entry["resolution"]["contexts"])
            self.assertEqual(len(paths), 2)
            self.assertIn("node_modules", paths[0])
            self.assertNotEqual(paths[0], paths[1],
                                msg="distinct contexts collapsed")
            with self.assertRaises(ValueError) as ctx:
                startup.check_identity_complete(inv)
            message = str(ctx.exception)
            self.assertIn("identity incomplete", message)
            self.assertIn("probe-dep", message)

    @unittest.skipUnless(shutil.which("bun") and shutil.which("go"),
                         "bun and go are required")
    def test_builtin_only_tree_passes_full_completeness(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup.cleanup_scratch, str(scratch))
            tracker = startup._phase_tracker(scratch)
            ordinary = g28k_write_closure_tree(scratch, "builtin")
            inv = startup.collect_prepared_identities(scratch, REPO, tracker)
            entry = [entry for entry in
                     inv["resolved"]["reachable_bare"]
                     if entry["specifier"] == "node:fs"][0]
            self.assertEqual(entry["resolution"]["status"], "builtin")
            self.assertEqual(entry["importers"],
                             [str((ordinary / "a" / "importer.js").resolve()),
                              str((ordinary / "b" / "importer.js").resolve())])
            self.assertTrue(startup.check_identity_complete(inv))

    @unittest.skipUnless(shutil.which("bun") and shutil.which("go"),
                         "bun and go are required")
    def test_reachable_importers_exclude_syntactic_orphan(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup.cleanup_scratch, str(scratch))
            tracker = startup._phase_tracker(scratch)
            ordinary = g28k_write_closure_tree(scratch, "file")
            (ordinary / "orphan.js").write_text(
                'import("probe-dep");\nimport("orphan-dep");\n')
            inv = startup.collect_prepared_identities(scratch, REPO, tracker)
            specs = {entry["specifier"]
                     for entry in inv["resolved"]["reachable_bare"]}
            self.assertIn("probe-dep", specs)
            self.assertNotIn("orphan-dep", specs)
            entry = [entry for entry in
                     inv["resolved"]["reachable_bare"]
                     if entry["specifier"] == "probe-dep"][0]
            self.assertEqual(entry["importers"],
                             [str((ordinary / "a" / "importer.js").resolve()),
                              str((ordinary / "b" / "importer.js").resolve())])
            loose = {entry["specifier"] for entry in
                     inv["resolved"]["unreachable_bare"]}
            self.assertIn("orphan-dep", loose)


class MetadataCustodyTests(unittest.TestCase):
    """G28n metadata custody: durable group/leader, verified-only clearing."""

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_live_child_marker_custody(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup.cleanup_scratch, str(scratch))
            tracker = startup._phase_tracker(scratch)
            outcome = {}

            def run():
                try:
                    startup._tracked_command(
                        tracker, [shutil.which("sleep"), "30"],
                        str(parent), timeout=30)
                except BaseException as error:
                    outcome["error"] = error
                else:
                    outcome["error"] = None

            thread = threading.Thread(target=run, daemon=True)
            thread.start()
            try:
                pgid, leader = None, None
                deadline = time.monotonic() + 10
                while time.monotonic() < deadline:
                    meta = startup._read_owner(
                        scratch, allow_live_group=True)
                    if meta.get("process_group") is not None:
                        pgid = meta["process_group"]
                        leader = meta["group_leader"]
                        break
                    time.sleep(0.02)
                self.assertIsNotNone(
                    pgid, msg="tracked group never registered")
                self.assertTrue(
                    startup._pid_alive(pgid, group=True),
                    msg="group not alive at observation")
                self.assertEqual(leader["pid"], pgid)
                self.assertEqual(leader["process_start"],
                                 startup._process_start(pgid))
                record = tracker.retire_pending()
                self.assertTrue(record["retired"])
            finally:
                thread.join(timeout=15)
            self.assertFalse(thread.is_alive())
            self.assertIsInstance(outcome["error"],
                                  subprocess.CalledProcessError)
            meta = startup._read_owner(scratch)
            self.assertIsNone(meta.get("process_group"))
            self.assertFalse(startup._pid_alive(pgid, group=True))

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_busy_tracker_refuses_second_command(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup.cleanup_scratch, str(scratch))
            tracker = startup._phase_tracker(scratch)
            outcome = {}

            def run():
                try:
                    startup._tracked_command(
                        tracker, [shutil.which("sleep"), "30"],
                        str(parent), timeout=30)
                except BaseException as error:
                    outcome["error"] = error

            thread = threading.Thread(target=run, daemon=True)
            thread.start()
            try:
                deadline = time.monotonic() + 10
                while time.monotonic() < deadline:
                    meta = startup._read_owner(
                        scratch, allow_live_group=True)
                    if meta.get("process_group") is not None:
                        break
                    time.sleep(0.02)
                else:
                    self.fail("tracked group never registered")
                with self.assertRaisesRegex(ValueError,
                                            "another phase group"):
                    startup._tracked_command(
                        tracker, [sys.executable, "-c", "pass"],
                        str(parent), timeout=10)
                record = tracker.retire_pending()
                self.assertTrue(record["retired"])
            finally:
                thread.join(timeout=15)
            self.assertFalse(thread.is_alive())
            self.assertIsInstance(outcome["error"],
                                  subprocess.CalledProcessError)

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_timeout_clears_verified_marker(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            try:
                tracker = startup._phase_tracker(scratch)
                with self.assertRaises(subprocess.TimeoutExpired):
                    startup._tracked_command(
                        tracker, [shutil.which("sleep"), "30"],
                        str(parent), timeout=0.5)
                meta = startup._read_owner(scratch)
                self.assertIsNone(meta.get("process_group"))
                self.assertIsNone(tracker.pgid)
            finally:
                self.assertTrue(startup.cleanup_scratch(str(scratch)))

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_interruption_clears_and_reraises(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            try:
                tracker = startup._phase_tracker(scratch)
                with startup._interrupts():
                    timer = threading.Timer(
                        0.3, os.kill,
                        args=(os.getpid(), signal.SIGTERM))
                    timer.daemon = True
                    timer.start()
                    try:
                        with self.assertRaises(startup._Interrupted):
                            startup._tracked_command(
                                tracker,
                                [shutil.which("sleep"), "30"],
                                str(parent), timeout=10.0)
                    finally:
                        timer.cancel()
                meta = startup._read_owner(scratch)
                self.assertIsNone(meta.get("process_group"))
                self.assertIsNone(tracker.pgid)
            finally:
                self.assertTrue(startup.cleanup_scratch(str(scratch)))

    def test_nonzero_clears_and_propagates(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            try:
                tracker = startup._phase_tracker(scratch)
                with self.assertRaises(
                        subprocess.CalledProcessError) as ctx:
                    startup._tracked_command(
                        tracker, [sys.executable, "-c",
                                  "import sys; sys.exit(3)"],
                        str(parent), timeout=10)
                self.assertEqual(ctx.exception.returncode, 3)
                meta = startup._read_owner(scratch)
                self.assertIsNone(meta.get("process_group"))
            finally:
                self.assertTrue(startup.cleanup_scratch(str(scratch)))

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_failed_retirement_retains_marker(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup.cleanup_scratch, str(scratch))
            tracker = startup._phase_tracker(scratch)
            with g28g_unverified_retirement():
                with self.assertRaises(
                        startup.RetirementFailed) as ctx:
                    startup._tracked_command(
                        tracker, [shutil.which("sleep"), "5"],
                        str(parent), timeout=0.3)
            self.assertTrue(ctx.exception.timed_out)
            self.assertIsNotNone(tracker.pgid)
            meta = startup._read_owner(scratch, allow_live_group=True)
            self.assertEqual(meta.get("process_group"), tracker.pgid)
            self.assertEqual(meta["group_leader"]["pid"],
                             tracker.pgid)
            self.assertFalse(startup._pid_alive(tracker.pgid,
                                                group=True),
                             msg="fixture leaked its group")
            record = tracker.retire_pending()
            self.assertTrue(record["retired"])
            self.assertIsNone(tracker.pgid)
            meta = startup._read_owner(scratch)
            self.assertIsNone(meta.get("process_group"))

    @unittest.skipUnless(shutil.which("sleep"), "sleep is required")
    def test_registration_failure_propagates(self):
        with tempfile.TemporaryDirectory() as parent:
            bare = Path(parent) / "unregistered"
            bare.mkdir()
            tracker = startup._phase_tracker(bare)
            real_track = tracker.track
            seen = []

            def recording(pgid):
                seen.append(pgid)
                return real_track(pgid)

            tracker.track = recording
            with self.assertRaisesRegex(
                    ValueError, r"scratch ownership invalid"):
                startup._tracked_command(
                    tracker, [shutil.which("sleep"), "30"],
                    str(parent), timeout=10)
            self.assertEqual(len(seen), 1)
            self.assertFalse(startup._pid_alive(seen[0], group=True),
                             msg="registration failure leaked its group")
            self.assertIsNone(tracker.pgid)
            self.assertFalse((bare / startup.OWNER_MARKER).exists())

    @unittest.skipUnless(shutil.which("bun") and shutil.which("go"),
                         "bun and go are required")
    def test_collect_registers_each_metadata_family(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup.cleanup_scratch, str(scratch))
            tracker = startup._phase_tracker(scratch)
            real_track = tracker.track
            seen = []

            def recording(pgid):
                seen.append(pgid)
                return real_track(pgid)

            tracker.track = recording
            g28k_write_closure_tree(scratch, "builtin")
            inv = startup.collect_prepared_identities(
                scratch, REPO, tracker)
            self.assertGreaterEqual(len(seen), 8)
            for pgid in seen:
                self.assertFalse(startup._pid_alive(pgid, group=True),
                                 msg=f"metadata group {pgid} leaked")
            meta = startup._read_owner(scratch)
            self.assertIsNone(meta.get("process_group"))
            for key in ("bun", "go", "node"):
                self.assertTrue(inv["tool_versions"][key].strip())
            self.assertTrue(inv["builtins"]["bun_revision"].strip())
            self.assertTrue(inv["inputs"]["emitter"]["packages"])
            specs = {entry["specifier"] for entry in
                     inv["resolved"]["reachable_bare"]}
            self.assertIn("node:fs", specs)
            self.assertTrue(startup.check_identity_complete(inv))

    def test_emitter_failure_reports_error_string(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            try:
                tracker = startup._phase_tracker(scratch)
                repo = Path(parent) / "repo"
                repo.mkdir()
                (repo / "go.mod").write_text(
                    "module example.com/nolayout\n")
                result = startup._emitter_inputs(repo, tracker)
                self.assertEqual(result["packages"], [])
                self.assertIn("go list", result["error"])
                meta = startup._read_owner(scratch)
                self.assertIsNone(meta.get("process_group"))
            finally:
                self.assertTrue(startup.cleanup_scratch(str(scratch)))

    @unittest.skipUnless(shutil.which("bun") and shutil.which("go"),
                         "bun and go are required")
    def test_collect_propagates_failed_retirement(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = startup.make_scratch(parent)
            self.addCleanup(startup.cleanup_scratch, str(scratch))
            tracker = startup._phase_tracker(scratch)
            g28k_write_closure_tree(scratch, "builtin")
            with g28g_unverified_retirement():
                with self.assertRaises(
                        startup.RetirementFailed):
                    startup.collect_prepared_identities(
                        scratch, REPO, tracker)
            self.assertIsNotNone(tracker.pgid)
            meta = startup._read_owner(scratch, allow_live_group=True)
            self.assertEqual(meta.get("process_group"), tracker.pgid)
            record = tracker.retire_pending()
            self.assertTrue(record["retired"])
            meta = startup._read_owner(scratch)
            self.assertIsNone(meta.get("process_group"))


class BoundsRecordTests(unittest.TestCase):
    """G28e bounds separation: preparation vs sampling vs overall."""

    def test_validate_bounds_stamp_preparation_only(self):
        with tempfile.TemporaryDirectory() as parent:
            output = Path(parent) / "out.json"
            args = startup.parse_args(
                ["--repo", str(REPO), "--output", str(output),
                 "--validate-only", "--scratch-parent", parent])
            prepared = {"manifest": {"probe": "p"},
                        "prepared": {"statements": 1},
                        "checks": [{"name": "c", "status": "pass"}],
                        "timings": {"emit_ms": 1.0},
                        "scratch_bytes": 10, "scratch_files": 1}
            real_worker = startup.run_prepare_worker
            real_collect = startup.collect_prepared_identities
            startup.run_prepare_worker = (
                lambda *argv, **kwargs: prepared)
            startup.collect_prepared_identities = (
                lambda *argv, **kwargs: g28g_complete_identity())
            try:
                code = startup.run_validate(args, REPO)
            finally:
                startup.run_prepare_worker = real_worker
                startup.collect_prepared_identities = real_collect
            self.assertEqual(code, 0)
            record = json.loads(output.read_text())
            self.assertEqual(record["status"], "complete")
            self.assertEqual(record["mode"], "validate-only")
            bounds = record["bounds"]
            self.assertEqual(set(bounds),
                             {"overall", "preparation", "sampling"})
            self.assertIsNone(bounds["sampling"])
            for label in ("overall", "preparation"):
                self.assertIn("start", bounds[label])
                self.assertIn("end", bounds[label])
                self.assertLessEqual(bounds[label]["start"],
                                     bounds[label]["end"])
            self.assertEqual(record["timestamps"], bounds["overall"])
            self.assertLessEqual(bounds["overall"]["start"],
                                 bounds["preparation"]["start"])
            self.assertLessEqual(bounds["preparation"]["end"],
                                 bounds["overall"]["end"])
            self.assertTrue(record["scratch"]["retired"])
            leftovers = [item for item in os.listdir(parent)
                         if item != "out.json"]
            self.assertEqual(leftovers, [])

    def test_validate_failure_still_stamps_preparation(self):
        with tempfile.TemporaryDirectory() as parent:
            output = Path(parent) / "out.json"
            args = startup.parse_args(
                ["--repo", str(REPO), "--output", str(output),
                 "--validate-only", "--scratch-parent", parent])
            real_worker = startup.run_prepare_worker

            def failing_worker(*argv, **kwargs):
                raise startup._Fail("boom-prepare")

            startup.run_prepare_worker = failing_worker
            try:
                code = startup.run_validate(args, REPO)
            finally:
                startup.run_prepare_worker = real_worker
            self.assertEqual(code, 1)
            record = json.loads(output.read_text())
            self.assertEqual(record["status"], "failed")
            self.assertIn("boom-prepare", record["reason"])
            bounds = record["bounds"]
            self.assertIsNone(bounds["sampling"])
            self.assertIn("start", bounds["preparation"])
            self.assertIn("end", bounds["preparation"])
            self.assertLessEqual(bounds["preparation"]["start"],
                                 bounds["preparation"]["end"])
            self.assertTrue(record["scratch"]["retired"])
            leftovers = [item for item in os.listdir(parent)
                         if item != "out.json"]
            self.assertEqual(leftovers, [])

    def test_measure_bounds_separate_sampling(self):
        with tempfile.TemporaryDirectory() as parent:
            output = Path(parent) / "out.json"
            args = startup.parse_args(
                ["--repo", str(REPO), "--output", str(output),
                 "--measure", "--trials", "1", "--warmups", "0",
                 "--batches", "1", "--sampling-bound", "60",
                 "--scratch-parent", parent])
            probe = {"initializer": {"statement_count": 1},
                     "statements": [{"binding": "x"}]}
            prepared = {"manifest": {"probe": probe},
                        "prepared": {"statements": 1},
                        "checks": [{"name": "c", "status": "pass"}],
                        "timings": {"emit_ms": 1.0},
                        "scratch_bytes": 10, "scratch_files": 1}
            real_worker = startup.run_prepare_worker
            real_collect = startup.collect_prepared_identities
            real_supervised = startup.run_supervised

            def fake_supervised(argv, cwd, timeout, **kwargs):
                trial = int(argv[argv.index("--trial-index") + 1])
                rows_path = Path(argv[argv.index("--rows") + 1])
                rows = []
                for profile in startup.PROFILES:
                    costs = ([0.5]
                             if profile == startup.DIAGNOSTIC_PROFILE
                             else None)
                    rows.append(make_row(
                        trial, 0, profile, False, wall=2.5,
                        stages=g28g_profile_stages(profile),
                        statement_ms=costs))
                rows_path.write_text(json.dumps(
                    {"trial_index": trial, "rows": rows}))
                on_start = kwargs.get("on_start")
                if on_start is not None:
                    on_start(os.getpid())
                return subprocess.CompletedProcess(argv, 0, "", "")

            startup.run_prepare_worker = (
                lambda *argv, **kwargs: prepared)
            startup.collect_prepared_identities = (
                lambda *argv, **kwargs: g28g_complete_identity())
            startup.run_supervised = fake_supervised
            try:
                code = startup.run_measure(args, REPO)
            finally:
                startup.run_prepare_worker = real_worker
                startup.collect_prepared_identities = real_collect
                startup.run_supervised = real_supervised
            self.assertEqual(code, 0)
            record = json.loads(output.read_text())
            self.assertEqual(record["status"], "complete")
            self.assertEqual(record["mode"], "measure")
            bounds = record["bounds"]
            self.assertEqual(set(bounds),
                             {"overall", "preparation", "sampling"})
            for label in ("overall", "preparation", "sampling"):
                self.assertIn("start", bounds[label])
                self.assertIn("end", bounds[label])
                self.assertLessEqual(bounds[label]["start"],
                                     bounds[label]["end"])
            self.assertLessEqual(bounds["preparation"]["end"],
                                 bounds["sampling"]["start"])
            self.assertLessEqual(bounds["overall"]["start"],
                                 bounds["preparation"]["start"])
            self.assertLessEqual(bounds["sampling"]["end"],
                                 bounds["overall"]["end"])
            self.assertEqual(record["profile_orders"],
                             [startup.trial_order(0)])
            self.assertEqual(len(record["rows"]), 4)
            self.assertEqual(
                record["report"]["accepted_batches_per_profile_trial"],
                1)
            wall = record["report"]["profiles"][
                "ordinary-modules"]["parent_wall_ms"]["median"]
            self.assertEqual(wall, 2.5)
            leftovers = [item for item in os.listdir(parent)
                         if item != "out.json"]
            self.assertEqual(leftovers, [])

    def test_trial_rows_timer_wraps_driver_spawn(self):
        with tempfile.TemporaryDirectory() as parent:
            scratch = Path(parent) / "scratch"
            scratch.mkdir()
            profiles = {}
            for profile in startup.PROFILES:
                mode = ("diagnostic"
                        if profile == startup.DIAGNOSTIC_PROFILE
                        else "ordinary")
                profiles[profile] = {"mode": mode,
                                     "argv": ["--profile", profile,
                                              "--mode", mode]}
            (scratch / "prepared.json").write_text(json.dumps(
                {"statements": 1, "profiles": profiles}))
            real_driver = startup.driver_command

            def fake_driver(argv, cwd, timeout):
                profile = next(item for item in argv
                               if item in startup.PROFILES)
                mode = profiles[profile]["mode"]
                record = {"status": "ok",
                          "stages": g28g_profile_stages(profile)}
                if mode == "diagnostic":
                    record["events"] = [[0, 0, 1.0], [0, 1, 2.0]]
                return subprocess.CompletedProcess(
                    argv, 0, json.dumps(record) + "\n", "")

            startup.driver_command = fake_driver
            try:
                rows = startup.run_trial_rows(
                    scratch, 0, warmups=0, batches=1,
                    child_timeout=5)
            finally:
                startup.driver_command = real_driver
            self.assertEqual([row["profile"] for row in rows],
                             startup.trial_order(0))
            for row in rows:
                wall = row["parent_wall_ms"]
                self.assertIsInstance(wall, float)
                self.assertGreaterEqual(wall, 0.0)
                self.assertEqual(
                    row["stages"],
                    g28g_profile_stages(row["profile"]))
                self.assertFalse(row["warmup"])
                self.assertEqual((row["trial"], row["batch"]),
                                 (0, 0))
            diagnostic = [row for row in rows
                          if row["profile"]
                          == startup.DIAGNOSTIC_PROFILE][0]
            self.assertEqual(diagnostic["statement_ms"], [1.0])


if __name__ == "__main__":
    unittest.main()
