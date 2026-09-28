#!/usr/bin/env python3
"""Opt-in generated startup attribution. Default is validation only; timing
requires explicit --measure. See docs/performance/generated-startup-plan.md
and the frozen G25 contract in generated-startup-tasks.md.

Preparation (one perfemit build, zero-byte runtime inventory replaced by a
checkout link before execution, prepared JS module graph + Bun ESM bundle +
separately labelled instrumented module graph) is recorded separately from
timing. Each measured child is a fresh Bun process with a fresh private fd-3
snapshot supplied through the existing drivers/runtime.py command helper.
"""
import argparse
import hashlib
import importlib.util
import json
import math
import os
import selectors
import shutil
import signal
import stat
import subprocess
import sys
import tempfile
import time
from datetime import datetime, timezone
from pathlib import Path

SCHEMA = "can.startup-attribution/1"
PROBE_SCHEMA = "can.startup-probe-manifest/1"
ORDINARY_PROFILES = ("ordinary-modules", "ordinary-bundle", "minimal")
DIAGNOSTIC_PROFILE = "diagnostic-modules"
PROFILES = ORDINARY_PROFILES + (DIAGNOSTIC_PROFILE,)
STAGES = ("import", "diagnostics_import", "configure", "initialize")
STATUSES = ("measured", "skipped_no_metadata", "not_applicable_bundled")
STATEMENT_MARK = "__canStartupMark"
EVENTS_KEY = "__canStartupEvents"
PROBE_SPECIFIER = "./startup-probe.ts"
SCRATCH_PREFIX = "can-startup-attr-"
CAP_BYTES = 64 * 1024 * 1024
CAP_FILES = 500
OWNER_MARKER = ".startup-owner.json"
OWNER_KIND = "can.startup-scratch"
OWNER_SCHEMA = 2
OWNER_SCHEMAS = (1, 2)

driver_command = None
driver_adapter = None
_ALLOCATED = set()


class _Fail(Exception):
    pass


class _Interrupted(Exception):
    pass


class RetirementFailed(Exception):
    @property
    def args(self):
        return self._command_args

    @args.setter
    def args(self, value):
        self._command_args = value

    def __init__(self, detail, args=None, returncode=None, timed_out=False,
                 interrupted=False, retirement=None):
        super().__init__(f"phase retirement failed: {detail}")
        self._command_args = args
        self.detail = detail
        self.returncode = returncode
        self.timed_out = timed_out
        self.interrupted = interrupted
        self.retirement = retirement

    def __str__(self):
        return f"phase retirement failed: {self.detail}"


def bind_driver(repo):
    global driver_command, driver_adapter
    path = Path(repo) / "tools" / "performance" / "drivers" / "runtime.py"
    spec = importlib.util.spec_from_file_location("startup_attr_runtime_driver", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    driver_command = module.command
    driver_adapter = module.adapter


def parse_args(argv):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, required=True)
    parser.add_argument("--output", type=Path)
    parser.add_argument("--validate-only", action="store_true")
    parser.add_argument("--measure", action="store_true")
    parser.add_argument("--scratch-parent", type=Path, default=Path(tempfile.gettempdir()))
    parser.add_argument("--trials", type=int, default=6)
    parser.add_argument("--warmups", type=int, default=2)
    parser.add_argument("--batches", type=int, default=7)
    parser.add_argument("--child-timeout", type=int, default=60)
    parser.add_argument("--sampling-bound", type=int, default=300)
    parser.add_argument("--prepare-bound", type=int, default=600)
    parser.add_argument("--prepared-out", type=Path)
    parser.add_argument("--trial-run", action="store_true")
    parser.add_argument("--prepare-run", action="store_true")
    parser.add_argument("--trial-index", type=int)
    parser.add_argument("--prepared", type=Path)
    parser.add_argument("--rows", type=Path)
    return parser.parse_args(argv)


def median(values):
    ordered = sorted(values)
    if not ordered:
        raise ValueError("median requires at least one sample")
    middle = len(ordered) // 2
    if len(ordered) % 2:
        return float(ordered[middle])
    return (ordered[middle - 1] + ordered[middle]) / 2.0


def mad(values, center=None):
    if not values:
        raise ValueError("mad requires at least one sample")
    middle = median(values) if center is None else center
    return median([abs(value - middle) for value in values])


def trial_order(trial_index):
    shift = trial_index % len(ORDINARY_PROFILES)
    rotated = [ORDINARY_PROFILES[(shift + i) % len(ORDINARY_PROFILES)] for i in range(len(ORDINARY_PROFILES))]
    return rotated + [DIAGNOSTIC_PROFILE]


def _require_finite_ms(value, what):
    if (isinstance(value, bool) or not isinstance(value, (int, float))
            or not math.isfinite(value) or value < 0):
        raise ValueError(f"launch rejected: output {what} is not finite milliseconds")
    return float(value)


def validate_events(events, statement_count):
    if not isinstance(events, list) or len(events) != 2 * statement_count:
        found = len(events) if isinstance(events, list) else type(events).__name__
        raise ValueError(f"startup events invalid: count expected {2 * statement_count} found {found}")
    for position, event in enumerate(events):
        if (not isinstance(event, (list, tuple)) or len(event) != 3
                or isinstance(event[0], bool) or not isinstance(event[0], int)
                or not 0 <= event[0] < statement_count):
            raise ValueError(f"startup events invalid: index entry {position} has unknown statement")
        if isinstance(event[1], bool) or event[1] not in (0, 1):
            raise ValueError(f"startup events invalid: phase entry {position} is not 0/1")
        stamp = event[2]
        if isinstance(stamp, bool) or not isinstance(stamp, (int, float)):
            raise ValueError(f"startup events invalid: time entry {position} is not numeric")
        if not math.isfinite(stamp) or stamp < 0:
            raise ValueError(f"startup events invalid: time entry {position} is not finite nonnegative milliseconds")
    for index in range(statement_count):
        before, after = events[2 * index], events[2 * index + 1]
        if (before[0], before[1]) != (index, 0) or (after[0], after[1]) != (index, 1):
            raise ValueError(f"startup events invalid: order statement {index} marks out of order")
        if after[2] < before[2]:
            raise ValueError(f"startup events invalid: time statement {index} went backwards")
    for position in range(len(events) - 1):
        if events[position + 1][2] < events[position][2]:
            raise ValueError(f"startup events invalid: time event {position + 1} went backwards globally")
    return [float(events[2 * index + 1][2] - events[2 * index][2]) for index in range(statement_count)]


def _check_profile_stages(stages, profile):
    diag = (stages["diagnostics_import"]["status"], stages["configure"]["status"])
    if profile == "ordinary-bundle":
        if diag != ("not_applicable_bundled", "not_applicable_bundled"):
            raise ValueError(f"launch rejected: output profile {profile} diagnostics must be not_applicable_bundled")
    elif profile == "minimal":
        if diag != ("skipped_no_metadata", "skipped_no_metadata"):
            raise ValueError(f"launch rejected: output profile {profile} diagnostics must be skipped_no_metadata")
    elif profile in ("ordinary-modules", DIAGNOSTIC_PROFILE):
        if diag not in (("measured", "measured"), ("skipped_no_metadata", "skipped_no_metadata")):
            raise ValueError(f"launch rejected: output profile {profile} diagnostics must be a consistent measured/skipped pair")
    else:
        raise ValueError(f"launch rejected: output unknown profile {profile}")
    for name in ("import", "initialize"):
        if stages[name]["status"] != "measured":
            raise ValueError(f"launch rejected: output profile {profile} stage {name} must be measured")


def accept_launch(returncode, stdout, mode, statement_count, profile=None):
    if returncode != 0:
        raise ValueError(f"launch rejected: exit child failed with code {returncode}")
    lines = stdout.strip().splitlines()
    if len(lines) != 1:
        raise ValueError(f"launch rejected: output expected one JSON line found {len(lines)}")
    try:
        record = json.loads(lines[0])
    except json.JSONDecodeError as error:
        raise ValueError(f"launch rejected: json invalid child JSON: {error}")
    if not isinstance(record, dict) or record.get("status") != "ok":
        raise ValueError("launch rejected: status child did not report ok")
    stages = record.get("stages")
    if not isinstance(stages, dict) or set(stages) != set(STAGES):
        raise ValueError("launch rejected: output child stages malformed")
    for name in STAGES:
        item = stages[name]
        if not isinstance(item, dict) or item.get("status") not in STATUSES:
            raise ValueError("launch rejected: output child stages malformed")
        if item["status"] == "measured":
            _require_finite_ms(item.get("ms"), f"stage {name}")
        elif item.get("ms") is not None:
            raise ValueError("launch rejected: output child stages malformed")
    if profile is not None:
        _check_profile_stages(stages, profile)
    if mode == "ordinary":
        if "events" in record:
            raise ValueError("launch rejected: unexpected_events ordinary launch emitted probe events")
        return {"stages": stages}
    statement_ms = validate_events(record.get("events"), statement_count)
    return {"stages": stages, "statement_ms": statement_ms, "events": record["events"]}


def _utf16_index_map(text):
    """map[u] is the Python index for UTF-16 offset u, or None mid-surrogate."""
    mapping = []
    index = 0
    for char in text:
        mapping.append(index)
        if ord(char) > 0xFFFF:
            mapping.append(None)
        index += 1
    mapping.append(index)
    return mapping


def _utf16_to_python_index(text, offset):
    if isinstance(offset, bool) or not isinstance(offset, int):
        raise ValueError(f"manifest coverage invalid: span offset {offset!r} is not an integer")
    mapping = _utf16_index_map(text)
    if not 0 <= offset < len(mapping) or mapping[offset] is None:
        raise ValueError(f"manifest coverage invalid: span offset {offset} out of bounds")
    return mapping[offset]


def check_manifest_coverage(manifest, state_text):
    def invalid(code, detail):
        raise ValueError(f"manifest coverage invalid: {code} {detail}")
    if not isinstance(manifest, dict) or manifest.get("schema") != PROBE_SCHEMA:
        invalid("count", "not a startup probe manifest")
    initializer = manifest.get("initializer")
    statements = manifest.get("statements")
    if not isinstance(initializer, dict) or not isinstance(statements, list):
        invalid("count", "manifest lacks initializer/statements")
    expected = initializer.get("statement_count")
    if expected != len(statements) or expected < 1:
        invalid("count", f"statement_count {expected} != {len(statements)}")
    state_bytes = state_text.encode("utf8")
    if (manifest.get("state_sha256") != hashlib.sha256(state_bytes).hexdigest()
            or manifest.get("state_bytes") != len(state_bytes)):
        invalid("source", "whole-source identity does not match state text")
    units = _utf16_index_map(state_text)
    utf16_length = len(units) - 1
    previous_end = -1
    for position, item in enumerate(statements):
        if not isinstance(item, dict) or item.get("index") != position:
            invalid("order", f"entry {position} index mismatch")
        start, end = item.get("start"), item.get("end")
        if isinstance(start, bool) or isinstance(end, bool) or not isinstance(start, int) \
                or not isinstance(end, int) or start < previous_end:
            invalid("order", f"entry {position} starts out of order")
        if not 0 <= start < end <= utf16_length or start < previous_end \
                or units[start] is None or units[end] is None:
            invalid("span", f"entry {position} span out of bounds")
        if item.get("kind") not in ("VariableStatement", "ExpressionStatement"):
            invalid("kind", f"entry {position} kind {item.get('kind')}")
        span = state_text[units[start]:units[end]]
        if hashlib.sha256(span.encode("utf8")).hexdigest() != item.get("sha256") \
                or len(span.encode("utf8")) != item.get("bytes"):
            invalid("hash", f"entry {position} bytes do not match manifest")
        previous_end = end
    if (initializer.get("name") != "$canInitialize"
            or initializer.get("start") != statements[0].get("start")
            or initializer.get("end") != statements[-1].get("end")):
        invalid("initializer", "initializer extent does not match statement coverage")
    return True


def check_manifest_agreement(authoritative, candidate):
    def invalid(code, detail):
        raise ValueError(f"manifest agreement invalid: {code} {detail}")
    for manifest in (authoritative, candidate):
        if (not isinstance(manifest, dict) or manifest.get("schema") != PROBE_SCHEMA
                or not isinstance(manifest.get("initializer"), dict)
                or not isinstance(manifest.get("statements"), list)):
            invalid("count", "not a startup probe manifest")
    if (authoritative.get("state_sha256") != candidate.get("state_sha256")
            or authoritative.get("state_bytes") != candidate.get("state_bytes")):
        invalid("source", "state identity differs between manifests")
    if authoritative.get("initializer") != candidate.get("initializer"):
        initializer = candidate.get("initializer")
        if initializer.get("statement_count") != len(candidate.get("statements")):
            invalid("count", "candidate statement_count does not match its statements")
        invalid("initializer", "initializer extent differs between manifests")
    expected = authoritative.get("statements")
    found = candidate.get("statements")
    if len(found) != len(expected):
        invalid("count", f"statement count {len(found)} != authoritative {len(expected)}")
    for position, (want, got) in enumerate(zip(expected, found)):
        if not isinstance(got, dict) or got.get("index") != position \
                or got.get("start") != want.get("start") or got.get("end") != want.get("end"):
            invalid("span", f"entry {position} span differs from authoritative inventory")
        if (got.get("bytes") != want.get("bytes") or got.get("sha256") != want.get("sha256")
                or got.get("kind") != want.get("kind") or got.get("binding") != want.get("binding")
                or list(got.get("factories", [])) != list(want.get("factories", []))):
            invalid("hash", f"entry {position} content differs from authoritative inventory")
    return True


def check_caps(scratch):
    scratch = Path(scratch)
    total = 0
    compiled = 0
    for root, _, files in os.walk(scratch):
        for name in files:
            path = Path(root) / name
            try:
                info = path.lstat()
            except FileNotFoundError:
                continue
            total += info.st_size
            if path.suffix == ".js" and not path.is_symlink():
                compiled += 1
    if total > CAP_BYTES:
        raise ValueError(f"scratch cap exceeded: bytes {total} > {CAP_BYTES}")
    if compiled > CAP_FILES:
        raise ValueError(f"scratch cap exceeded: files {compiled} > {CAP_FILES}")
    return total, compiled


def _fs_identity(info):
    return [info.st_dev, info.st_ino]


def _process_start(pid):
    try:
        result = subprocess.run(["ps", "-p", str(pid), "-o", "lstart="],
                                capture_output=True, text=True, timeout=10)
    except Exception:
        return None
    if result.returncode != 0:
        return None
    started = result.stdout.strip()
    return started or None


def _pid_alive(pid, group=False):
    if not isinstance(pid, int) or isinstance(pid, bool) or pid <= 1:
        raise ValueError("Invalid recorded process identity")
    try:
        (os.killpg if group else os.kill)(pid, 0)
    except ProcessLookupError:
        return False
    except PermissionError:
        return True
    return True


def _owner_is_live(meta):
    pid = meta.get("pid")
    try:
        alive = _pid_alive(pid)
    except ValueError:
        return False
    if not alive:
        return False
    recorded = meta.get("process_start")
    if recorded is None:
        return True
    current = _process_start(pid)
    if current is None:
        return True
    return current == recorded


def _write_all(fd, data):
    view = memoryview(data)
    while view:
        written = os.write(fd, view)
        if written <= 0:
            raise OSError("short marker write: no progress writing ownership marker")
        view = view[written:]


def _unlink_exact_marker(marker, root, root_identity, opened_identity):
    if opened_identity is None:
        return False
    try:
        current_root = os.stat(root, follow_symlinks=False)
    except OSError:
        return False
    if _fs_identity(current_root) != root_identity:
        return False
    try:
        current_marker = os.stat(marker, follow_symlinks=False)
    except OSError:
        return False
    if _fs_identity(current_marker) != opened_identity:
        return False
    try:
        os.unlink(marker)
    except OSError:
        return False
    return True


def _write_owner_marker(root):
    root = Path(root)
    try:
        root_info = os.stat(root, follow_symlinks=False)
    except OSError as error:
        raise ValueError(f"scratch ownership invalid: missing cannot stat scratch {root}: {error}")
    if not stat.S_ISDIR(root_info.st_mode) or stat.S_ISLNK(root_info.st_mode):
        raise ValueError(f"scratch ownership invalid: foreign not an owned directory: {root}")
    marker = root / OWNER_MARKER
    try:
        fd = os.open(marker, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, 0o600)
    except FileExistsError:
        raise ValueError(f"scratch ownership invalid: foreign marker already exists in {root}")
    except OSError as error:
        raise ValueError(f"scratch ownership invalid: missing cannot create marker in {root}: {error}")
    try:
        meta = {"kind": OWNER_KIND, "schema_version": OWNER_SCHEMA, "root": str(root),
                "pid": os.getpid(), "process_start": _process_start(os.getpid()),
                "created_at": time.time(), "root_identity": _fs_identity(root_info),
                "marker_identity": _fs_identity(os.fstat(fd)), "process_group": None,
                "group_leader": None, "launched_children": False, "transferred": False,
                "keeper_pid": None, "keeper": None, "state": "active"}
        _write_all(fd, json.dumps(meta).encode("utf8"))
        os.fsync(fd)
    except BaseException:
        try:
            opened_identity = _fs_identity(os.fstat(fd))
        except OSError:
            opened_identity = None
        try:
            os.close(fd)
        except OSError:
            pass
        _unlink_exact_marker(marker, root, _fs_identity(root_info), opened_identity)
        raise
    os.close(fd)
    return meta


def _read_owner(root, allow_live_group=False):
    def invalid(code, detail):
        raise ValueError(f"scratch ownership invalid: {code} {detail}")
    root = Path(root)
    try:
        root_info = os.stat(root, follow_symlinks=False)
    except OSError:
        invalid("missing", f"no scratch at {root}")
    if not stat.S_ISDIR(root_info.st_mode) or stat.S_ISLNK(root_info.st_mode):
        invalid("foreign", f"not an owned directory: {root}")
    marker = root / OWNER_MARKER
    try:
        fd = os.open(marker, os.O_RDONLY | os.O_NOFOLLOW)
    except FileNotFoundError:
        invalid("missing", f"no ownership marker in {root}")
    except OSError as error:
        invalid("replaced", f"cannot open ownership marker in {root}: {error}")
    try:
        marker_info = os.fstat(fd)
        with os.fdopen(fd, "r") as handle:
            raw = handle.read()
    except OSError as error:
        invalid("replaced", f"cannot read ownership marker in {root}: {error}")
    try:
        meta = json.loads(raw)
    except json.JSONDecodeError:
        invalid("replaced", f"ownership marker unreadable in {root}")
    if (not isinstance(meta, dict) or meta.get("kind") != OWNER_KIND
            or meta.get("schema_version") not in OWNER_SCHEMAS
            or meta.get("root") != str(root)):
        invalid("foreign", f"unrecognized ownership in {root}")
    if "keeper" not in meta:
        meta["keeper"] = None
    if "group_leader" not in meta:
        meta["group_leader"] = None
    if meta["keeper"] is not None and not isinstance(meta["keeper"], dict):
        invalid("replaced", f"keeper record corrupt in {root}")
    try:
        marker_current = os.stat(marker, follow_symlinks=False)
    except OSError:
        invalid("replaced", f"ownership marker vanished in {root}")
    if (_fs_identity(root_info) != meta.get("root_identity")
            or _fs_identity(marker_info) != meta.get("marker_identity")
            or _fs_identity(marker_current) != meta.get("marker_identity")):
        invalid("replaced", f"scratch root or marker was replaced: {root}")
    if _owner_is_live(meta) and meta.get("pid") != os.getpid():
        invalid("active_owner", f"owner pid {meta.get('pid')} still running for {root}")
    group = meta.get("process_group")
    if group is not None and not allow_live_group and _pid_alive(group, group=True):
        invalid("active_group", f"child group {group} still running for {root}")
    keeper = meta.get("keeper")
    if (isinstance(keeper, dict) and _owner_is_live(keeper)
            and keeper.get("pid") != os.getpid()):
        invalid("active_keeper", f"keeper pid {keeper.get('pid')} still holds {root}")
    return meta


def _update_owner(root, mutate):
    root = Path(root)
    meta = _read_owner(root, allow_live_group=True)
    if meta.get("pid") != os.getpid() or not _owner_is_live(meta):
        raise ValueError(f"scratch ownership invalid: active_owner cannot update marker owned by {meta.get('pid')}")
    marker = root / OWNER_MARKER
    try:
        fd = os.open(marker, os.O_RDWR | os.O_NOFOLLOW)
    except OSError as error:
        raise ValueError(f"scratch ownership invalid: replaced cannot open marker in {root}: {error}")
    try:
        if _fs_identity(os.fstat(fd)) != meta.get("marker_identity"):
            raise ValueError(f"scratch ownership invalid: replaced marker was replaced in {root}")
        mutate(meta)
        with os.fdopen(fd, "r+") as handle:
            handle.seek(0)
            handle.write(json.dumps(meta))
            handle.truncate()
            handle.flush()
            os.fsync(handle.fileno())
    except ValueError:
        try:
            os.close(fd)
        except OSError:
            pass
        raise
    return meta


def _set_process_group(root, pgid, leader=None):
    def mutate(meta):
        if pgid is not None:
            meta["launched_children"] = True
        meta["process_group"] = pgid
        meta["group_leader"] = leader
    return _update_owner(root, mutate)


def take_custody(root):
    root = Path(root)
    meta = _read_owner(root, allow_live_group=True)
    marker = root / OWNER_MARKER
    try:
        fd = os.open(marker, os.O_RDWR | os.O_NOFOLLOW)
    except OSError as error:
        raise ValueError(f"scratch ownership invalid: replaced cannot open marker in {root}: {error}")
    try:
        if _fs_identity(os.fstat(fd)) != meta.get("marker_identity"):
            raise ValueError(f"scratch ownership invalid: replaced marker was replaced in {root}")
        meta["keeper"] = {"pid": os.getpid(), "process_start": _process_start(os.getpid()),
                          "asserted_at": time.time()}
        with os.fdopen(fd, "r+") as handle:
            handle.seek(0)
            handle.write(json.dumps(meta))
            handle.truncate()
            handle.flush()
            os.fsync(handle.fileno())
    except ValueError:
        try:
            os.close(fd)
        except OSError:
            pass
        raise
    return meta


def release_custody(root):
    root = Path(root)
    meta = _read_owner(root, allow_live_group=True)
    keeper = meta.get("keeper")
    if not isinstance(keeper, dict) or keeper.get("pid") != os.getpid():
        raise ValueError(f"scratch ownership invalid: active_keeper custody not held by this process for {root}")
    marker = root / OWNER_MARKER
    try:
        fd = os.open(marker, os.O_RDWR | os.O_NOFOLLOW)
    except OSError as error:
        raise ValueError(f"scratch ownership invalid: replaced cannot open marker in {root}: {error}")
    try:
        if _fs_identity(os.fstat(fd)) != meta.get("marker_identity"):
            raise ValueError(f"scratch ownership invalid: replaced marker was replaced in {root}")
        meta["keeper"] = None
        with os.fdopen(fd, "r+") as handle:
            handle.seek(0)
            handle.write(json.dumps(meta))
            handle.truncate()
            handle.flush()
            os.fsync(handle.fileno())
    except ValueError:
        try:
            os.close(fd)
        except OSError:
            pass
        raise
    return meta


def _delete_owned_tree(root, meta):
    root = Path(root)
    try:
        current = os.stat(root, follow_symlinks=False)
    except OSError as error:
        raise ValueError(f"scratch cleanup failed: {root}: cannot verify root before delete: {error}")
    if _fs_identity(current) != meta.get("root_identity"):
        raise ValueError(f"scratch ownership invalid: replaced scratch root was replaced: {root}")
    try:
        shutil.rmtree(root, ignore_errors=False)
    except OSError as error:
        raise ValueError(f"scratch cleanup failed: {root}: {error}")
    if os.path.lexists(root):
        raise ValueError(f"scratch cleanup failed: {root}: still present after delete")


def _rollback_new_directory(path, expected_identity):
    path = Path(path)
    try:
        current = os.stat(path, follow_symlinks=False)
    except OSError as error:
        raise ValueError(f"rollback failed: cannot stat {path}: {error}")
    if _fs_identity(current) != expected_identity or not stat.S_ISDIR(current.st_mode):
        raise ValueError(f"rollback failed: {path} was replaced, refusing to touch it")
    try:
        with os.scandir(path) as entries:
            if any(True for _ in entries):
                raise ValueError(f"rollback failed: {path} is not empty, refusing to touch it")
    except OSError as error:
        raise ValueError(f"rollback failed: cannot inspect {path}: {error}")
    try:
        os.rmdir(path)
    except OSError as error:
        raise ValueError(f"rollback failed: cannot remove {path}: {error}")


def _register_or_rollback(path):
    path = Path(path)
    try:
        identity = _fs_identity(os.stat(path, follow_symlinks=False))
    except OSError as error:
        raise ValueError(f"scratch registration failed: cannot stat new allocation {path}: {error}")
    try:
        return _write_owner_marker(path)
    except (KeyboardInterrupt, SystemExit) as interrupted:
        try:
            _rollback_new_directory(path, identity)
        except ValueError as rollback_error:
            raise ValueError(f"scratch registration failed: interrupted; {rollback_error} "
                             f"(cleanup pending: {path})") from interrupted
        raise
    except BaseException as marker_error:
        try:
            _rollback_new_directory(path, identity)
        except ValueError as rollback_error:
            raise ValueError(f"scratch registration failed: {marker_error}; {rollback_error} "
                             f"(cleanup pending: {path})") from marker_error
        raise ValueError(f"scratch registration failed: {marker_error}; rolled back {path}") from marker_error


def make_scratch(parent):
    parent = Path(parent)
    parent.mkdir(parents=True, exist_ok=True)
    path = Path(tempfile.mkdtemp(prefix=SCRATCH_PREFIX, dir=str(parent)))
    _register_or_rollback(path)
    _ALLOCATED.add(str(path))
    return path


def cleanup_scratch(path):
    if path is None:
        return False
    path = Path(path)
    if not os.path.lexists(path):
        if str(path) in _ALLOCATED:
            _ALLOCATED.discard(str(path))
            return True
        return False
    meta = _read_owner(path)
    _delete_owned_tree(path, meta)
    _ALLOCATED.discard(str(path))
    return True


def recover_scratch(path):
    if path is None:
        return False
    path = Path(path)
    if not os.path.lexists(path):
        if str(path) in _ALLOCATED:
            _ALLOCATED.discard(str(path))
            return True
        return False
    meta = _read_owner(path, allow_live_group=True)
    if _owner_is_live(meta):
        raise ValueError(f"scratch ownership invalid: active_owner owner pid {meta.get('pid')} still running for {path}")
    group = meta.get("process_group")
    if group is not None and _pid_alive(group, group=True):
        retired = _kill_process_group(group, leader=meta.get("group_leader"))
        if not retired["retired"]:
            reason = retired.get("refused") or f"abandoned group {group} survives retirement"
            raise ValueError(f"scratch cleanup failed: {path}: {reason}")
    _delete_owned_tree(path, meta)
    _ALLOCATED.discard(str(path))
    return True


class owned_scratch:
    def __init__(self, parent=None, path=None, keep=False):
        self.parent = Path(parent) if parent is not None else Path(tempfile.gettempdir())
        self.explicit = Path(path) if path is not None else None
        self.keep = keep
        self.path = None

    def __enter__(self):
        if self.explicit is not None:
            if os.path.lexists(self.explicit):
                raise ValueError(f"scratch path already exists: {self.explicit}")
            self.explicit.mkdir(parents=True)
            _register_or_rollback(self.explicit)
            self.path = self.explicit
        else:
            self.path = make_scratch(self.parent)
        _ALLOCATED.add(str(self.path))
        return self.path

    def __exit__(self, exc_type, exc, tb):
        if self.keep and exc_type is None:
            def mutate(meta):
                meta["transferred"] = True
                meta["keeper_pid"] = os.getpid()
                meta["keeper"] = {"pid": os.getpid(), "process_start": _process_start(os.getpid()),
                                  "asserted_at": time.time()}
                meta["state"] = "retained"
            _update_owner(self.path, mutate)
            return False
        cleanup_scratch(self.path)
        return False


def _reap_group_children(pgid):
    reaped = 0
    while True:
        try:
            pid, _ = os.waitpid(-pgid, os.WNOHANG)
        except ChildProcessError:
            return reaped
        except OSError:
            return reaped
        if pid == 0:
            return reaped
        reaped += 1


def _group_live(pgid):
    _reap_group_children(pgid)
    return _pid_alive(pgid, group=True)


def _process_group_of(pid):
    try:
        result = subprocess.run(["ps", "-p", str(pid), "-o", "pgid="],
                                capture_output=True, text=True, timeout=10)
    except Exception:
        return None
    if result.returncode != 0:
        return None
    try:
        return int(result.stdout.strip().split()[0])
    except (IndexError, ValueError):
        return None


def _verify_group_owner(pgid, leader):
    if not isinstance(leader, dict):
        return False, "leader record malformed"
    pid, start = leader.get("pid"), leader.get("process_start")
    try:
        alive = _pid_alive(pid)
    except ValueError:
        alive = False
    if alive:
        if start is not None:
            current = _process_start(pid)
            if current is None or current != start:
                return False, "leader pid reused"
        if _process_group_of(pid) != pgid:
            return False, "leader not in group"
        return True, "live leader"
    if _reap_group_children(pgid) > 0:
        return True, "reaped own member"
    return False, "leader dead, no continuity proof"


def _kill_process_group(pgid, term_grace=2.0, kill_grace=1.0, leader=None):
    if isinstance(pgid, bool) or not isinstance(pgid, int) or pgid <= 1:
        raise ValueError(f"invalid process group: {pgid!r}")
    if pgid == os.getpgrp():
        raise ValueError(f"refusing to signal own process group: {pgid}")
    if not _group_live(pgid):
        return {"pgid": pgid, "termed": True, "killed": False, "retired": True,
                "refused": None}
    if leader is not None:
        verified, reason = _verify_group_owner(pgid, leader)
        if not verified:
            return {"pgid": pgid, "termed": False, "killed": False, "retired": False,
                    "refused": f"owner-mismatch: {reason}"}
    try:
        os.killpg(pgid, signal.SIGTERM)
    except ProcessLookupError:
        return {"pgid": pgid, "termed": True, "killed": False, "retired": True,
                "refused": None}
    except PermissionError:
        return {"pgid": pgid, "termed": False, "killed": False, "retired": False,
                "refused": None}
    deadline = time.monotonic() + max(0.0, term_grace)
    while time.monotonic() < deadline:
        if not _group_live(pgid):
            return {"pgid": pgid, "termed": True, "killed": False, "retired": True,
                    "refused": None}
        time.sleep(0.05)
    if not _group_live(pgid):
        return {"pgid": pgid, "termed": True, "killed": False, "retired": True,
                "refused": None}
    try:
        os.killpg(pgid, signal.SIGKILL)
    except ProcessLookupError:
        return {"pgid": pgid, "termed": True, "killed": False, "retired": True,
                "refused": None}
    except PermissionError:
        return {"pgid": pgid, "termed": False, "killed": False, "retired": False,
                "refused": None}
    deadline = time.monotonic() + max(0.0, kill_grace)
    while time.monotonic() < deadline:
        if not _group_live(pgid):
            return {"pgid": pgid, "termed": False, "killed": True, "retired": True,
                    "refused": None}
        time.sleep(0.05)
    retired = not _group_live(pgid)
    return {"pgid": pgid, "termed": False, "killed": True, "retired": retired,
            "refused": None}


def _retire_and_drain(proc, term_grace, kill_grace, drain_timeout):
    killed = _kill_process_group(proc.pid, term_grace, kill_grace)
    try:
        out, err = proc.communicate(timeout=max(0.0, drain_timeout))
        drained = True
    except subprocess.TimeoutExpired as expired:
        out, err = expired.output or "", expired.stderr or ""
        for pipe in (proc.stdout, proc.stderr):
            try:
                if pipe is not None:
                    pipe.close()
            except OSError:
                pass
        drained = False
    retired = killed["retired"] and drained and not _group_live(proc.pid)
    retirement = {"pgid": proc.pid, "termed": killed["termed"], "killed": killed["killed"],
                  "retired": retired, "pipes_drained": drained, "refused": killed["refused"]}
    return retirement, out or "", err or ""


CAPTURE_LIMIT = 4 * 1024 * 1024


class _CaptureOverflow(Exception):
    pass


def _open_pipe_selector(proc):
    selector = selectors.DefaultSelector()
    for pipe, tag in ((proc.stdout, "stdout"), (proc.stderr, "stderr")):
        fd = pipe.fileno()
        os.set_blocking(fd, False)
        selector.register(fd, selectors.EVENT_READ, tag)
    return selector


def _consume_ready(selector, buffers, total, timeout, limit=CAPTURE_LIMIT):
    for key, _ in selector.select(timeout=timeout):
        try:
            chunk = os.read(key.fd, 65536)
        except OSError:
            chunk = b""
        if not chunk:
            try:
                selector.unregister(key.fd)
            except KeyError:
                pass
            continue
        buffers[key.data].extend(chunk)
        total += len(chunk)
        if limit is not None and total > limit:
            raise _CaptureOverflow(total)
    return total


def _close_pipes(proc, selector):
    for key in list(selector.get_map().values()):
        try:
            selector.unregister(key.fd)
        except KeyError:
            pass
    for pipe in (proc.stdout, proc.stderr):
        try:
            if pipe is not None:
                pipe.close()
        except OSError:
            pass


def _reap_bounded(proc, timeout):
    try:
        proc.wait(timeout=max(0.0, timeout))
        return True
    except subprocess.TimeoutExpired:
        return False


def _finish_supervised(proc, selector, buffers, total,
                       term_grace, kill_grace, drain_timeout):
    killed = _kill_process_group(proc.pid, term_grace, kill_grace)
    drained, overflowed = True, False
    if selector.get_map():
        end = time.monotonic() + max(0.0, drain_timeout)
        try:
            while selector.get_map():
                remaining = end - time.monotonic()
                if remaining <= 0:
                    drained = False
                    break
                total = _consume_ready(selector, buffers, total,
                                       min(remaining, 0.05))
        except _CaptureOverflow:
            drained, overflowed = False, True
    reaped = _reap_bounded(proc, max(1.0, term_grace + kill_grace))
    _close_pipes(proc, selector)
    group_dead = not _group_live(proc.pid)
    if overflowed:
        retired = killed["retired"] and reaped and group_dead
        return ({"pgid": proc.pid, "termed": killed["termed"],
                 "killed": killed["killed"], "retired": retired,
                 "pipes_drained": False, "refused": killed["refused"]},
                "", "", True)
    retired = killed["retired"] and drained and reaped and group_dead
    out = bytes(buffers["stdout"]).decode("utf-8")
    err = bytes(buffers["stderr"]).decode("utf-8")
    return ({"pgid": proc.pid, "termed": killed["termed"],
             "killed": killed["killed"], "retired": retired,
             "pipes_drained": drained, "refused": killed["refused"]},
            out, err, False)


def run_supervised(args, cwd, timeout, term_grace=2.0, kill_grace=1.0,
                   drain_timeout=5.0, on_start=None):
    argv = [str(item) for item in args]
    proc = subprocess.Popen(argv, cwd=str(cwd), stdout=subprocess.PIPE,
                            stderr=subprocess.PIPE, text=True, start_new_session=True)
    if on_start is not None:
        try:
            on_start(proc.pid)
        except BaseException:
            _retire_and_drain(proc, term_grace, kill_grace, drain_timeout)
            raise
    buffers = {"stdout": bytearray(), "stderr": bytearray()}
    total = 0
    selector = _open_pipe_selector(proc)
    deadline = time.monotonic() + timeout
    try:
        while True:
            if proc.poll() is not None:
                outcome = "exit"
                break
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                outcome = "timeout"
                break
            total = _consume_ready(selector, buffers, total,
                                   min(remaining, 0.05))
    except _CaptureOverflow:
        outcome = "overflow"
    except BaseException as signaled:
        retirement, _, _, _ = _finish_supervised(
            proc, selector, buffers, total,
            term_grace, kill_grace, drain_timeout)
        if retirement["retired"]:
            raise
        raise RetirementFailed(f"signal with unverified retirement: {retirement}",
                               args=argv, interrupted=True,
                               retirement=retirement) from signaled
    if outcome == "overflow":
        retirement, _, _, _ = _finish_supervised(
            proc, selector, buffers, total,
            term_grace, kill_grace, drain_timeout)
        if retirement["retired"]:
            raise ValueError(f"command output overflow: captured more than "
                             f"{CAPTURE_LIMIT} combined bytes from {argv}")
        raise RetirementFailed(f"output overflow with unverified retirement: {retirement}",
                               args=argv, retirement=retirement)
    if outcome == "timeout":
        retirement, out, err, overflowed = _finish_supervised(
            proc, selector, buffers, total,
            term_grace, kill_grace, drain_timeout)
        if overflowed:
            if retirement["retired"]:
                raise ValueError(f"command output overflow: captured more than "
                                 f"{CAPTURE_LIMIT} combined bytes from {argv}")
            raise RetirementFailed(f"output overflow with unverified retirement: {retirement}",
                                   args=argv, timed_out=True,
                                   retirement=retirement)
        if retirement["retired"]:
            raise subprocess.TimeoutExpired(proc.args, timeout, output=out, stderr=err)
        raise RetirementFailed(f"timeout with unverified retirement: {retirement}",
                               args=argv, timed_out=True, retirement=retirement)
    retirement, out, err, overflowed = _finish_supervised(
        proc, selector, buffers, total,
        term_grace, kill_grace, drain_timeout)
    if overflowed:
        if retirement["retired"]:
            raise ValueError(f"command output overflow: captured more than "
                             f"{CAPTURE_LIMIT} combined bytes from {argv}")
        raise RetirementFailed(f"output overflow with unverified retirement: {retirement}",
                               args=argv, returncode=proc.returncode,
                               retirement=retirement)
    if proc.returncode == 0 and retirement["retired"]:
        return subprocess.CompletedProcess(proc.args, proc.returncode, out, err)
    if proc.returncode != 0 and retirement["retired"]:
        raise subprocess.CalledProcessError(proc.returncode, proc.args, output=out, stderr=err)
    raise RetirementFailed(f"exit {proc.returncode} with unverified retirement: {retirement}",
                           args=argv, returncode=proc.returncode, retirement=retirement)


class _phase_tracker:
    def __init__(self, scratch):
        self.scratch = Path(scratch)
        self.pgid = None
        self.leader = None

    def track(self, pgid):
        self.pgid = pgid
        self.leader = {"pid": pgid, "process_start": _process_start(pgid)}
        _set_process_group(self.scratch, pgid, self.leader)

    def clear_verified(self):
        self.pgid = None
        self.leader = None
        _set_process_group(self.scratch, None)

    def retire_pending(self):
        if self.pgid is None:
            return None
        record = _kill_process_group(self.pgid)
        if record["retired"]:
            record["pgid"] = self.pgid
            self.pgid = None
            self.leader = None
            try:
                _set_process_group(self.scratch, None)
            except (ValueError, OSError):
                pass
        return record


def _retained_custody(scratch, reason):
    scratch = Path(scratch)
    try:
        meta = _read_owner(scratch, allow_live_group=True)
    except ValueError as error:
        return {"root": str(scratch), "reason": reason, "owner_error": str(error)}
    return {"root": str(scratch), "root_identity": meta.get("root_identity"),
            "owner_pid": meta.get("pid"), "owner_start": meta.get("process_start"),
            "pgid": meta.get("process_group"), "leader": meta.get("group_leader"),
            "reason": reason}


def _scratch_outcome(owned):
    path = owned.path if owned is not None else None
    if path is None:
        return False
    return not os.path.lexists(path)


def mirror_runtime_inventory(repo_runtime, dest):
    repo_runtime, dest = Path(repo_runtime), Path(dest)
    sources = sorted(repo_runtime.rglob("*.ts"))
    if not sources:
        raise ValueError("runtime inventory empty")
    for source in sources:
        target = dest / source.relative_to(repo_runtime)
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(b"")
    return len(sources)


def verify_placeholders(generated_runtime, expected_count):
    found = sorted(Path(generated_runtime).rglob("*.ts"))
    if len(found) != expected_count:
        raise ValueError(f"placeholder count {len(found)} != inventory {expected_count}")
    for path in found:
        if path.stat().st_size != 0:
            raise ValueError(f"placeholder not zero-byte: {path}")
    return len(found)


def replace_runtime_link(generated, repo_runtime):
    target = Path(generated) / "runtime"
    if not target.is_dir() or target.is_symlink():
        raise ValueError("runtime placeholders missing")
    shutil.rmtree(target)
    target.symlink_to(Path(repo_runtime), target_is_directory=True)
    if not target.is_symlink():
        raise ValueError("runtime link not created")


def install_node_modules_link(repo, work):
    modules = Path(repo) / "node_modules"
    if not modules.is_dir():
        raise FileNotFoundError("existing node_modules required; preparation never installs dependencies")
    target = Path(work) / "node_modules"
    if not target.exists():
        target.symlink_to(modules, target_is_directory=True)


FACADE_SOURCE = """export {$canInitialize as initialize} from "./program/state.ts";
import {doubled, frequency, sum, generic_pass, captured_map} from "./generated-adapter.ts";
import {array} from "./runtime/data.ts";
import {value} from "./runtime/completion.ts";
export async function verify(): Promise<void> {
  const input = array([1n, 2n]);
  const doubledResult = value(await doubled(input));
  if (doubledResult.length !== 2 || doubledResult[1] !== 4n || !Object.isFrozen(doubledResult)) throw new Error("startup oracle doubled");
  if (input.length !== 2 || input[0] !== 1n || !Object.isFrozen(input)) throw new Error("startup oracle input unchanged");
  const occurrences = value(await frequency(array(["w0", "w1", "w0"]), "w0"));
  if (occurrences !== 2n) throw new Error("startup oracle frequency");
  const total = value(await sum(input));
  if (total !== 3n) throw new Error("startup oracle fold");
  const identity = value(await generic_pass(input));
  if (identity.length !== 2 || identity[0] !== 1n || !Object.isFrozen(identity)) throw new Error("startup oracle generic");
  const shifted = value(await captured_map(input, 4n));
  if (shifted.length !== 2 || shifted[1] !== 6n || !Object.isFrozen(shifted)) throw new Error("startup oracle captured");
}
"""

MINIMAL_SOURCE = """export function initialize(): void {}
export async function verify(): Promise<void> {}
"""

PROBE_SOURCE = """const events: Array<[number, number, number]> = (((globalThis as unknown as Record<string, unknown>).__canStartupEvents ??= []) as Array<[number, number, number]>);
export function __canStartupMark(index: number, phase: number): void {
  events.push([index, phase, performance.now()]);
}
"""

LAUNCHER_SOURCE = """// Plain launcher: no static imports, so no Can runtime preloads before the timer.
const args = process.argv.slice(2);
const get = (name) => {
  const i = args.indexOf(name);
  if (i < 0 || i + 1 >= args.length) throw new Error("missing launcher arg " + name);
  return args[i + 1];
};
const rootURL = get("--root");
const mode = get("--mode");
const diagnostics = get("--diagnostics");
const metadataRoot = get("--metadata-root");
const statements = Number(get("--statements"));
if ((mode !== "ordinary" && mode !== "diagnostic") || (diagnostics !== "skip" && diagnostics !== "run")) throw new Error("bad launcher mode");
if (!Number.isSafeInteger(statements) || statements < 1) throw new Error("bad launcher statements");
const stages = {};
const importStart = performance.now();
const root = await import(rootURL);
stages.import = {status: "measured", ms: performance.now() - importStart};
if (diagnostics === "run") {
  const diagStart = performance.now();
  const diag = await import(get("--diag-module"));
  stages.diagnostics_import = {status: "measured", ms: performance.now() - diagStart};
  const configureStart = performance.now();
  diag.configureDiagnostics(get("--entry-url"));
  stages.configure = {status: "measured", ms: performance.now() - configureStart};
} else {
  const skipped = get("--diag-na");
  if (skipped !== "skipped_no_metadata" && skipped !== "not_applicable_bundled") throw new Error("bad diag-na status");
  stages.diagnostics_import = {status: skipped, ms: null};
  stages.configure = {status: skipped, ms: null};
}
const initStart = performance.now();
root.initialize();
stages.initialize = {status: "measured", ms: performance.now() - initStart};
await root.verify();
const record = {status: "ok", metadata_root: metadataRoot, stages};
if (mode === "diagnostic") {
  const events = globalThis.__canStartupEvents;
  if (!Array.isArray(events) || events.length !== 2 * statements) throw new Error("incomplete probe events");
  record.events = events;
} else if ("__canStartupEvents" in globalThis) {
  throw new Error("unexpected probe events");
}
console.log(JSON.stringify(record));
"""


def write_facade(path):
    Path(path).write_text(FACADE_SOURCE)


def write_minimal(path):
    Path(path).write_text(MINIMAL_SOURCE)


def write_probe_module(path):
    Path(path).write_text(PROBE_SOURCE)


def write_launcher(path):
    Path(path).write_text(LAUNCHER_SOURCE)


def _sha256_file(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def _utcnow():
    return datetime.now(timezone.utc).isoformat()


def _git_commit(repo):
    try:
        result = subprocess.run(["git", "rev-parse", "HEAD"], cwd=str(repo), capture_output=True,
                                text=True, timeout=30, check=True)
        return result.stdout.strip()
    except Exception:
        return "unknown"


def _describe_child_error(error):
    detail = str(error)
    if isinstance(error, subprocess.CalledProcessError):
        detail += "\n" + (error.stderr or "")[-2000:]
    return detail[-2200:]


class _interrupts:
    def __enter__(self):
        self.previous = {}
        for number in (signal.SIGINT, signal.SIGTERM):
            try:
                self.previous[number] = signal.getsignal(number)
                signal.signal(number, self._raise)
            except (OSError, ValueError):
                continue
        return self

    def _raise(self, signum, frame):
        raise _Interrupted(f"signal {signum}")

    def __exit__(self, exc_type, exc, tb):
        for number, handler in self.previous.items():
            try:
                signal.signal(number, handler)
            except (OSError, ValueError):
                continue
        return False


def _run_probe(repo, state, out, manifest):
    node = shutil.which("node")
    if node is None:
        raise _Fail("node is required for the AST probe but not on PATH")
    probe = Path(repo) / "tools" / "performance" / "startup-attribution.ts"
    args = [node, "--no-warnings", str(probe), "instrument", "--state", str(state),
            "--out", str(out), "--manifest", str(manifest), "--probe-specifier", PROBE_SPECIFIER]
    try:
        driver_command(args, str(repo), timeout=180)
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired) as error:
        raise _Fail(f"AST probe failed: {_describe_child_error(error)}")


def _run_inventory(repo, state, manifest):
    node = shutil.which("node")
    if node is None:
        raise _Fail("node is required for the AST probe but not on PATH")
    probe = Path(repo) / "tools" / "performance" / "startup-attribution.ts"
    args = [node, "--no-warnings", str(probe), "inventory", "--state", str(state),
            "--manifest", str(manifest)]
    try:
        driver_command(args, str(repo), timeout=180)
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired) as error:
        raise _Fail(f"AST inventory failed: {_describe_child_error(error)}")


def prepare(repo, scratch, child_timeout=60):
    repo, scratch = Path(repo), Path(scratch)
    for tool in ("bun", "go"):
        if shutil.which(tool) is None:
            raise _Fail(f"{tool} must already be installed")
    checks = []
    timings = {}
    started = time.perf_counter()
    perfemit = scratch / "perfemit"
    previous_gomaxprocs = os.environ.get("GOMAXPROCS")
    os.environ["GOMAXPROCS"] = "2"
    try:
        driver_command(["go", "build", "-p=2", "-o", str(perfemit), "./compiler/perfemit"],
                       str(repo), timeout=300)
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired) as error:
        raise _Fail(f"perfemit build failed: {_describe_child_error(error)}")
    finally:
        if previous_gomaxprocs is None:
            os.environ.pop("GOMAXPROCS", None)
        else:
            os.environ["GOMAXPROCS"] = previous_gomaxprocs
    timings["perfemit_ms"] = (time.perf_counter() - started) * 1000.0
    checks.append({"name": "perfemit-build", "status": "pass",
                   "detail": f"{perfemit.stat().st_size} bytes"})
    generated = scratch / "generated"
    inventory = scratch / "runtime-inventory"
    mirrored = mirror_runtime_inventory(repo / "runtime", inventory)
    started = time.perf_counter()
    try:
        emitted = driver_command([str(perfemit), str(repo / "tools" / "performance" / "fixtures" / "runtime"),
                                  str(generated), str(inventory)], str(repo), timeout=300)
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired) as error:
        raise _Fail(f"emission failed: {_describe_child_error(error)}")
    timings["emit_ms"] = (time.perf_counter() - started) * 1000.0
    checks.append({"name": "emission", "status": "pass", "detail": emitted.stdout.strip()})
    try:
        verify_placeholders(generated / "runtime", mirrored)
    except ValueError as error:
        raise _Fail(f"placeholder verification failed: {error}")
    checks.append({"name": "placeholders", "status": "pass", "detail": f"{mirrored} zero-byte files"})
    replace_runtime_link(generated, repo / "runtime")
    install_node_modules_link(repo, scratch)
    checks.append({"name": "links", "status": "pass", "detail": "runtime + node_modules links"})
    driver_adapter(generated)
    bindings = json.loads((generated / "bindings.json").read_text())
    checks.append({"name": "bindings", "status": "pass", "detail": f"{len(bindings)} bindings"})
    write_facade(generated / "startup-facade.ts")
    write_minimal(generated / "startup-minimal.ts")
    ordinary = scratch / "generated-js"
    transpile_script = str(repo / "tools" / "performance" / "drivers" / "runtime-transpile.ts")
    started = time.perf_counter()
    try:
        transpile = driver_command(["bun", transpile_script, str(generated), str(ordinary)],
                                   str(scratch), timeout=300)
        # The existing transform does not follow the checkout runtime link, so
        # the referenced runtime inputs get their own identical transform pass.
        transpile_runtime = driver_command(["bun", transpile_script, str(generated / "runtime"),
                                            str(ordinary / "runtime")], str(scratch), timeout=300)
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired) as error:
        raise _Fail(f"transpile failed: {_describe_child_error(error)}")
    timings["transpile_ms"] = (time.perf_counter() - started) * 1000.0
    try:
        transpile_record = json.loads(transpile.stdout)
        transpile_record_runtime = json.loads(transpile_runtime.stdout)
    except json.JSONDecodeError:
        raise _Fail("transpile record invalid")
    transpile_record = {"javascript_modules": transpile_record.get("javascript_modules", 0)
                        + transpile_record_runtime.get("javascript_modules", 0),
                        "transformation": transpile_record.get("transformation"),
                        "passes": [{"root": "generated", "javascript_modules": transpile_record.get("javascript_modules")},
                                   {"root": "generated/runtime", "javascript_modules": transpile_record_runtime.get("javascript_modules")}]}
    try:
        _write_output(scratch / "transpilation.json", transpile_record)
    except (ValueError, OSError) as error:
        raise _Fail(f"cannot write transpile record: {error}")
    checks.append({"name": "transpile", "status": "pass",
                   "detail": f"{transpile_record.get('javascript_modules')} modules"})
    probe_manifest_path = scratch / "probe-manifest.json"
    inventory_manifest_path = scratch / "inventory-manifest.json"
    instrumented_src = scratch / "instrument-src"
    instrumented_state_ts = instrumented_src / "program" / "state.ts"
    instrumented_state_ts.parent.mkdir(parents=True, exist_ok=True)
    started = time.perf_counter()
    state_text = (generated / "program" / "state.ts").read_text()
    _run_inventory(repo, generated / "program" / "state.ts", inventory_manifest_path)
    _run_probe(repo, generated / "program" / "state.ts", instrumented_state_ts, probe_manifest_path)
    probe_manifest = json.loads(probe_manifest_path.read_text())
    inventory_manifest = json.loads(inventory_manifest_path.read_text())
    try:
        check_manifest_agreement(inventory_manifest, probe_manifest)
        check_manifest_coverage(probe_manifest, state_text)
    except ValueError as error:
        raise _Fail(f"probe coverage failed: {error}")
    write_probe_module(instrumented_src / "program" / "startup-probe.ts")
    instrument_js = scratch / "instrument-js"
    try:
        driver_command(["bun", str(repo / "tools" / "performance" / "drivers" / "runtime-transpile.ts"),
                        str(instrumented_src), str(instrument_js)], str(scratch), timeout=300)
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired) as error:
        raise _Fail(f"instrument transpile failed: {_describe_child_error(error)}")
    timings["instrument_ms"] = (time.perf_counter() - started) * 1000.0
    statement_count = probe_manifest["initializer"]["statement_count"]
    checks.append({"name": "instrument", "status": "pass",
                   "detail": f"{statement_count} statements, coverage exact"})
    diag = scratch / "generated-js-diag"
    (diag / "program").mkdir(parents=True, exist_ok=True)
    (diag / "packages").mkdir(parents=True, exist_ok=True)
    shutil.copy2(ordinary / "startup-facade.js", diag / "startup-facade.js")
    shutil.copy2(ordinary / "generated-adapter.js", diag / "generated-adapter.js")
    for package in sorted((ordinary / "packages").rglob("*.js")):
        target = diag / "packages" / package.relative_to(ordinary / "packages")
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(package, target)
    shutil.copy2(instrument_js / "program" / "state.js", diag / "program" / "state.js")
    shutil.copy2(instrument_js / "program" / "startup-probe.js", diag / "program" / "startup-probe.js")
    (diag / "runtime").symlink_to(ordinary / "runtime", target_is_directory=True)
    if _sha256_file(diag / "program" / "state.js") == _sha256_file(ordinary / "program" / "state.js"):
        raise _Fail("instrumented state identical to ordinary state")
    checks.append({"name": "diagnostic-graph", "status": "pass",
                   "detail": "owned state+probe, shared runtime link"})
    bundle = scratch / "startup-ordinary-bundle.js"
    started = time.perf_counter()
    try:
        driver_command(["bun", "build", str(ordinary / "startup-facade.js"), "--target=bun",
                        "--format=esm", "--outfile", str(bundle)], str(scratch), timeout=180)
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired) as error:
        raise _Fail(f"bundle failed: {_describe_child_error(error)}")
    timings["bundle_ms"] = (time.perf_counter() - started) * 1000.0
    bundle_sha = _sha256_file(bundle)
    checks.append({"name": "bundle", "status": "pass",
                   "detail": f"{bundle.stat().st_size} bytes sha {bundle_sha[:12]}"})
    launcher = scratch / "startup-launcher.js"
    write_launcher(launcher)
    metadata_root = generated
    diagnostics_applicable = (generated / "diagnostics" / "source-index.json").exists()
    entry_url = (generated / "entry.ts").resolve().as_uri()
    diag_module = {"ordinary-modules": (ordinary / "runtime" / "diagnostics.js").as_uri(),
                   DIAGNOSTIC_PROFILE: (diag / "runtime" / "diagnostics.js").as_uri()}
    profiles = {}
    roots = {"ordinary-modules": ordinary / "startup-facade.js", "ordinary-bundle": bundle,
             "minimal": ordinary / "startup-minimal.js", DIAGNOSTIC_PROFILE: diag / "startup-facade.js"}
    for profile, root in roots.items():
        if profile == "ordinary-bundle":
            diag_mode, diag_na = "skip", "not_applicable_bundled"
        elif profile == "minimal":
            diag_mode, diag_na = "skip", "skipped_no_metadata"
        elif diagnostics_applicable:
            diag_mode, diag_na = "run", "measured"
        else:
            diag_mode, diag_na = "skip", "skipped_no_metadata"
        profiles[profile] = {"mode": "diagnostic" if profile == DIAGNOSTIC_PROFILE else "ordinary",
                             "argv": [str(launcher), "--root", root.resolve().as_uri(),
                                      "--mode", "diagnostic" if profile == DIAGNOSTIC_PROFILE else "ordinary",
                                      "--diagnostics", diag_mode,
                                      "--diag-module", diag_module.get(profile, diag_module["ordinary-modules"]),
                                      "--diag-na", "measured" if diag_mode == "run" else diag_na,
                                      "--metadata-root", str(metadata_root),
                                      "--entry-url", entry_url, "--statements", str(statement_count)]}
    prepared = {"launcher": str(launcher), "statements": statement_count,
                "metadata_root": str(metadata_root), "entry_url": entry_url,
                "diagnostics_applicable": diagnostics_applicable, "profiles": profiles}
    try:
        _write_output(scratch / "prepared.json", prepared)
        _write_output(scratch / "manifest.json", probe_manifest)
    except (ValueError, OSError) as error:
        raise _Fail(f"cannot write prepared record: {error}")
    for profile in PROFILES:
        spec = profiles[profile]
        try:
            result = driver_command(["bun"] + spec["argv"], str(scratch), timeout=child_timeout)
        except (subprocess.CalledProcessError, subprocess.TimeoutExpired) as error:
            raise _Fail(f"preflight {profile} failed: {_describe_child_error(error)}")
        try:
            accepted = accept_launch(0, result.stdout, spec["mode"], statement_count, profile=profile)
        except ValueError as error:
            raise _Fail(f"preflight {profile} rejected: {error}")
        checks.append({"name": f"preflight:{profile}", "status": "pass",
                       "detail": f"stages {[k for k, v in accepted['stages'].items() if v['status'] == 'measured']}"})
    try:
        scratch_bytes, scratch_files = check_caps(scratch)
    except ValueError as error:
        raise _Fail(f"scratch cap check failed: {error}")
    manifest = {"probe": probe_manifest,
                "preparation": {"perfemit_sha256": _sha256_file(perfemit), "bindings": len(bindings),
                                "transpilation": transpile_record, "bundle_sha256": bundle_sha,
                                "bundle_bytes": bundle.stat().st_size,
                                "diagnostics_applicable": diagnostics_applicable,
                                "metadata_root": str(metadata_root), "entry_url": entry_url,
                                "roots": {name: spec["argv"][2] for name, spec in profiles.items()}}}
    return {"manifest": manifest, "prepared": prepared, "checks": checks, "timings": timings,
            "scratch_bytes": scratch_bytes, "scratch_files": scratch_files}


def run_trial_rows(scratch, trial_index, warmups, batches, child_timeout):
    scratch = Path(scratch)
    prepared = json.loads((scratch / "prepared.json").read_text())
    statements = prepared["statements"]
    rows = []
    for profile in trial_order(trial_index):
        spec = prepared["profiles"][profile]
        for batch in range(warmups + batches):
            started = time.perf_counter_ns()
            try:
                result = driver_command(["bun"] + spec["argv"], str(scratch), timeout=child_timeout)
            except subprocess.TimeoutExpired as error:
                raise _Fail(f"trial {trial_index} {profile} batch {batch} child timeout")
            except subprocess.CalledProcessError as error:
                raise _Fail(f"trial {trial_index} {profile} batch {batch} exit {error.returncode}: "
                            f"{(error.stderr or '')[-500:]}")
            parent_wall_ms = (time.perf_counter_ns() - started) / 1e6
            if not math.isfinite(parent_wall_ms) or parent_wall_ms < 0:
                raise _Fail(f"trial {trial_index} {profile} batch {batch} parent wall clock invalid")
            try:
                accepted = accept_launch(0, result.stdout, spec["mode"], statements, profile=profile)
            except ValueError as error:
                raise _Fail(f"trial {trial_index} {profile} batch {batch} rejected: {error}")
            row = {"trial": trial_index, "batch": batch, "profile": profile,
                   "warmup": batch < warmups, "parent_wall_ms": parent_wall_ms,
                   "stages": accepted["stages"]}
            if spec["mode"] == "diagnostic":
                row["statement_ms"] = accepted["statement_ms"]
                row["events"] = accepted["events"]
            rows.append(row)
    return rows


def collect_activity():
    try:
        result = subprocess.run(["ps", "-eo", "pid,ppid,pcpu,pmem,comm"], capture_output=True,
                                text=True, timeout=30, check=True)
    except Exception as error:
        return {"at": _utcnow(), "error": f"{type(error).__name__}"}
    lines = result.stdout.strip().splitlines()[1:]
    entries = []
    for line in lines:
        parts = line.split(None, 4)
        if len(parts) != 5:
            continue
        try:
            entries.append({"pid": int(parts[0]), "ppid": int(parts[1]), "pcpu": float(parts[2]),
                            "pmem": float(parts[3]), "comm": parts[4][:64]})
        except ValueError:
            continue
    entries.sort(key=lambda item: item["pcpu"], reverse=True)
    return {"at": _utcnow(), "processes": len(entries), "top5": entries[:5]}


def check_row_membership(rows, trials, warmups, batches):
    def invalid(code, detail):
        raise ValueError(f"row membership invalid: {code} {detail}")
    if not isinstance(rows, list):
        invalid("count", f"rows is {type(rows).__name__}, not a list")
    if isinstance(trials, bool) or not isinstance(trials, int) or trials < 1:
        invalid("count", f"trials {trials!r} malformed")
    if isinstance(batches, bool) or not isinstance(batches, int) or batches < 1:
        invalid("count", f"batches {batches!r} malformed")
    if isinstance(warmups, bool) or not isinstance(warmups, int) or warmups < 0:
        invalid("count", f"warmups {warmups!r} malformed")
    seen = set()
    for position, row in enumerate(rows):
        if not isinstance(row, dict):
            invalid("unknown", f"entry {position} is not a row")
        trial, batch, profile, warmup = (row.get("trial"), row.get("batch"),
                                         row.get("profile"), row.get("warmup"))
        if isinstance(trial, bool) or not isinstance(trial, int) or not 0 <= trial < trials:
            invalid("unknown", f"entry {position} trial {trial!r} outside 0..{trials - 1}")
        if profile not in PROFILES:
            invalid("unknown", f"entry {position} profile {profile!r} unknown")
        total = warmups + batches
        if isinstance(batch, bool) or not isinstance(batch, int) or not 0 <= batch < total:
            invalid("unknown", f"entry {position} batch {batch!r} outside 0..{total - 1}")
        if not isinstance(warmup, bool) or warmup != (batch < warmups):
            invalid("warmup", f"entry {position} warmup {warmup!r} mismatches batch {batch}")
        key = (trial, profile, batch)
        if key in seen:
            invalid("duplicate", f"entry {position} repeats trial {trial} profile {profile} batch {batch}")
        seen.add(key)
    expected = {(trial, profile, batch) for trial in range(trials) for profile in PROFILES
                for batch in range(warmups + batches)}
    missing = expected - seen
    if missing:
        sample = sorted(missing)[0]
        invalid("omission", f"{len(missing)} rows missing, e.g. trial {sample[0]} profile {sample[1]} batch {sample[2]}")
    return True


def check_trial_rows(rows, trial_index, warmups, batches):
    def invalid(code, detail):
        raise ValueError(f"row membership invalid: {code} {detail}")
    if not isinstance(rows, list):
        invalid("count", f"rows is {type(rows).__name__}, not a list")
    seen = set()
    for position, row in enumerate(rows):
        if not isinstance(row, dict):
            invalid("unknown", f"entry {position} is not a row")
        trial, batch, profile, warmup = (row.get("trial"), row.get("batch"),
                                         row.get("profile"), row.get("warmup"))
        if trial != trial_index or isinstance(trial, bool) or not isinstance(trial, int):
            invalid("unknown", f"entry {position} trial {trial!r} != controller trial {trial_index}")
        if profile not in PROFILES:
            invalid("unknown", f"entry {position} profile {profile!r} unknown")
        total = warmups + batches
        if isinstance(batch, bool) or not isinstance(batch, int) or not 0 <= batch < total:
            invalid("unknown", f"entry {position} batch {batch!r} outside 0..{total - 1}")
        if not isinstance(warmup, bool) or warmup != (batch < warmups):
            invalid("warmup", f"entry {position} warmup {warmup!r} mismatches batch {batch}")
        key = (profile, batch)
        if key in seen:
            invalid("duplicate", f"entry {position} repeats profile {profile} batch {batch}")
        seen.add(key)
    expected = {(profile, batch) for profile in PROFILES for batch in range(warmups + batches)}
    missing = expected - seen
    if missing:
        sample = sorted(missing)[0]
        invalid("omission", f"{len(missing)} rows missing, e.g. profile {sample[0]} batch {sample[1]}")
    return True


def build_report(rows, trials, probe_manifest, warmups=2, batches=7):
    try:
        check_row_membership(rows, trials, warmups, batches)
    except ValueError as error:
        raise _Fail(f"report rejected: {error}")
    statement_count = probe_manifest["initializer"]["statement_count"]
    for row in rows:
        wall = row.get("parent_wall_ms")
        if isinstance(wall, bool) or not isinstance(wall, (int, float)) \
                or not math.isfinite(wall) or wall < 0:
            raise _Fail(f"report rejected: trial {row.get('trial')} profile {row.get('profile')} "
                        "parent wall is not finite milliseconds")
        for name in STAGES:
            item = row["stages"][name]
            if item["status"] == "measured":
                value = item.get("ms")
                if isinstance(value, bool) or not isinstance(value, (int, float)) \
                        or not math.isfinite(value) or value < 0:
                    raise _Fail(f"report rejected: trial {row.get('trial')} profile {row.get('profile')} "
                                f"stage {name} is not finite milliseconds")
            elif item.get("ms") is not None:
                raise _Fail(f"report rejected: trial {row.get('trial')} profile {row.get('profile')} "
                            f"stage {name} unmeasured but carries ms")
        if row["profile"] == DIAGNOSTIC_PROFILE:
            costs = row.get("statement_ms")
            if not isinstance(costs, list) or len(costs) != statement_count:
                raise _Fail(f"report rejected: trial {row.get('trial')} diagnostic statement_ms incomplete")
            for cost in costs:
                if isinstance(cost, bool) or not isinstance(cost, (int, float)) \
                        or not math.isfinite(cost) or cost < 0:
                    raise _Fail("report rejected: diagnostic statement cost is not finite milliseconds")
    accepted = [row for row in rows if not row["warmup"]]
    if not accepted:
        raise _Fail("report has no accepted rows")
    report = {"unit": "ms", "trials": trials, "accepted_batches_per_profile_trial": None,
              "profiles": {}, "diagnostic_statements": []}
    by_trial_profile = {}
    for row in accepted:
        by_trial_profile.setdefault((row["trial"], row["profile"]), []).append(row)
    for profile in PROFILES:
        per_trial = {}
        for trial in range(trials):
            group = by_trial_profile.get((trial, profile), [])
            if not group:
                raise _Fail(f"report missing trial {trial} profile {profile}")
            for stage in STAGES + ("parent_wall_ms",):
                samples = [row["stages"][stage]["ms"] if stage in STAGES else row["parent_wall_ms"]
                             for row in group]
                samples = [value for value in samples if value is not None]
                if samples:
                    per_trial.setdefault(stage, []).append(median(samples))
        summary = {}
        for stage, medians in per_trial.items():
            middle = median(medians)
            summary[stage] = {"median": middle, "min": min(medians), "max": max(medians),
                              "mad": mad(medians, middle), "measured_trials": len(medians)}
        for stage in STAGES:
            if stage not in summary:
                status = next(row["stages"][stage]["status"] for row in accepted
                              if row["profile"] == profile)
                summary[stage] = {"median": None, "min": None, "max": None, "mad": None,
                                  "measured_trials": 0, "status": status}
        summary["parent_wall_ms"] = summary.pop("parent_wall_ms")
        report["profiles"][profile] = summary
    bindings = [item["binding"] for item in probe_manifest["statements"]]
    for index, binding in enumerate(bindings):
        per_trial_medians = []
        for trial in range(trials):
            group = [row["statement_ms"][index] for row in accepted
                     if row["trial"] == trial and row["profile"] == DIAGNOSTIC_PROFILE]
            if group:
                per_trial_medians.append(median(group))
        if per_trial_medians:
            middle = median(per_trial_medians)
            report["diagnostic_statements"].append(
                {"index": index, "binding": binding, "median_ms": middle,
                 "min_ms": min(per_trial_medians), "max_ms": max(per_trial_medians),
                 "mad_ms": mad(per_trial_medians, middle)})
    accepted_batches = min(len(group) for group in by_trial_profile.values())
    if accepted_batches != batches:
        raise _Fail(f"report rejected: accepted batches per profile/trial {accepted_batches} != {batches}")
    report["accepted_batches_per_profile_trial"] = accepted_batches
    return report


def _tracked_command(tracker, args, cwd, timeout=30, term_grace=0.5,
                   kill_grace=0.5, drain_timeout=2.0):
    if tracker.pgid is not None:
        raise ValueError("metadata command refused: another phase group is tracked")
    try:
        completed = run_supervised(args, cwd, timeout=timeout, term_grace=term_grace,
                                   kill_grace=kill_grace, drain_timeout=drain_timeout,
                                   on_start=tracker.track)
    except RetirementFailed:
        raise
    except BaseException:
        tracker.clear_verified()
        raise
    tracker.clear_verified()
    return completed


def _tracked_probe(tracker, args, cwd, timeout=30):
    try:
        completed = _tracked_command(tracker, args, cwd, timeout=timeout)
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired, OSError):
        return None
    except ValueError as error:
        if str(error).startswith("command output overflow"):
            return None
        raise
    text = (completed.stdout or "").strip()
    return text.splitlines()[0][:128] if text else None


def collect_tool_versions(tracker):
    versions = {}
    for key, argv in (("bun", ["bun", "--version"]), ("go", ["go", "version"]),
                     ("node", ["node", "--version"])):
        versions[key] = _tracked_probe(tracker, argv, Path.cwd())
    package = Path(__file__).resolve().parents[2] / "node_modules" / "typescript" / "package.json"
    try:
        versions["typescript"] = json.loads(package.read_text()).get("version")
    except Exception:
        versions["typescript"] = None
    return versions


def _executable_identity(name, version):
    path = shutil.which(name)
    if path is None:
        return None
    try:
        real = str(Path(path).resolve())
        info = os.stat(real)
        if not stat.S_ISREG(info.st_mode):
            return None
        return {"realpath": real, "version": version, "dev": info.st_dev,
                "ino": info.st_ino, "size": info.st_size}
    except OSError:
        return None


def _hash_tree(root, suffixes=None):
    files = []
    root = Path(root)
    if root.is_dir() and not root.is_symlink():
        for path in sorted(root.rglob("*")):
            if path.is_symlink() or not path.is_file():
                continue
            if suffixes is not None and path.suffix not in suffixes:
                continue
            try:
                data = path.read_bytes()
            except OSError:
                continue
            files.append({"path": path.relative_to(root).as_posix(), "bytes": len(data),
                          "sha256": hashlib.sha256(data).hexdigest()})
    return files


def _module_path(repo):
    try:
        for line in (Path(repo) / "go.mod").read_text().splitlines():
            words = line.split()
            if len(words) >= 2 and words[0] == "module":
                return words[1]
    except OSError:
        pass
    return None


def _emitter_inputs(repo, tracker):
    repo = Path(repo)
    empty = {"packages": [], "error": None}
    module = _module_path(repo)
    if module is None:
        empty["error"] = "go.mod module path unreadable"
        return empty
    try:
        completed = _tracked_command(tracker, ["go", "list", "-deps", "./compiler/perfemit"],
                                     str(repo), timeout=120)
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired, OSError) as error:
        empty["error"] = f"go list -deps failed: {error}"
        return empty
    except ValueError as error:
        if not str(error).startswith("command output overflow"):
            raise
        empty["error"] = f"go list -deps failed: {error}"
        return empty
    owned = sorted(line.strip() for line in (completed.stdout or "").splitlines()
                   if line.strip().startswith(module + "/") or line.strip() == module)
    if not owned:
        empty["error"] = "no module-owned emitter dependencies"
        return empty
    try:
        completed = _tracked_command(
            tracker,
            ["go", "list", "-f", "{{.Dir}}|{{join .GoFiles \",\"}}|{{join .CgoFiles \",\"}}"]
            + owned, str(repo), timeout=120)
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired, OSError) as error:
        empty["error"] = f"go list files failed: {error}"
        return empty
    except ValueError as error:
        if not str(error).startswith("command output overflow"):
            raise
        empty["error"] = f"go list files failed: {error}"
        return empty
    packages = []
    for package, line in zip(owned, (completed.stdout or "").splitlines()):
        parts = line.split("|")
        if len(parts) != 3:
            continue
        directory, go_files, cgo_files = parts[0], parts[1], parts[2]
        files = []
        for name in [item for item in (go_files.split(",") + cgo_files.split(",")) if item]:
            target = Path(directory) / name
            try:
                if target.is_symlink() or not target.is_file():
                    continue
                data = target.read_bytes()
            except OSError:
                continue
            try:
                rel = target.relative_to(repo).as_posix()
            except ValueError:
                rel = str(target)
            files.append({"path": rel, "bytes": len(data),
                          "sha256": hashlib.sha256(data).hexdigest()})
        packages.append({"package": package, "files": sorted(files, key=lambda item: item["path"])})
    return {"packages": packages, "error": None if packages else "no emitter files resolved"}


def _inventory_js(root):
    files, links = [], []
    root = Path(root)
    if root.is_dir() and not root.is_symlink():
        for path in sorted(root.rglob("*")):
            if path.is_symlink():
                links.append({"path": path.relative_to(root).as_posix(), "target": os.readlink(path)[:512]})
            elif path.is_file() and path.suffix == ".js":
                data = path.read_bytes()
                files.append({"path": path.relative_to(root).as_posix(), "bytes": len(data),
                              "sha256": hashlib.sha256(data).hexdigest()})
    return {"files": files, "symlinks": sorted(links, key=lambda item: item["path"])}


def _sha256_or_none(path):
    try:
        target = Path(path)
        if target.is_symlink() or not target.is_file():
            return None
        return hashlib.sha256(target.read_bytes()).hexdigest()
    except OSError:
        return None


def _bare_with_importers(roots):
    import re
    found = {}
    pattern = re.compile(r"""(?:from\s+|import\s*\(\s*)["']([^"'./][^"']*)["']""")
    for root in roots:
        root = Path(root)
        if not root.is_dir() or root.is_symlink():
            continue
        for path in root.rglob("*.js"):
            if path.is_symlink() or not path.is_file():
                continue
            try:
                text = path.read_text()
            except OSError:
                continue
            for spec in pattern.findall(text):
                found.setdefault(spec[:256], set()).add(str(path))
    return {spec: sorted(files) for spec, files in sorted(found.items())}


_IMPORT_PATTERN = None


def _import_pattern():
    global _IMPORT_PATTERN
    if _IMPORT_PATTERN is None:
        import re
        _IMPORT_PATTERN = re.compile(
            r"""(?:from\s+|import\s*\(\s*|import\s+|require\s*\(\s*)["']([^"']+)["']""")
    return _IMPORT_PATTERN


def _walk_reachable(roots, boundary, bundle_file=None):
    roots = [Path(item) for item in roots]
    boundary = Path(boundary)
    reachable, bare, unresolved, truncated = [], {}, [], False
    queue = [item for item in roots if item.is_file() and not item.is_symlink()]
    seen = set()
    pattern = _import_pattern()
    while queue:
        if len(seen) >= 2000:
            truncated = True
            break
        current = queue.pop(0)
        key = str(current)
        if key in seen:
            continue
        seen.add(key)
        reachable.append(key)
        try:
            text = current.read_text()
        except OSError:
            continue
        for spec in pattern.findall(text):
            if spec.startswith("."):
                target = (current.parent / spec).resolve()
                try:
                    target.relative_to(boundary.resolve())
                except ValueError:
                    unresolved.append({"importer": key, "specifier": spec[:256],
                                       "reason": "escapes prepared roots"})
                    continue
                if target.is_file() and not target.is_symlink():
                    if str(target) not in seen:
                        queue.append(target)
                else:
                    unresolved.append({"importer": key, "specifier": spec[:256],
                                       "reason": "relative target missing"})
            elif spec.startswith("/") or "://" in spec:
                unresolved.append({"importer": key, "specifier": spec[:256],
                                   "reason": "absolute or remote import"})
            else:
                bare.setdefault(spec[:256], set()).add(key)
    if bundle_file is not None and Path(bundle_file).is_file() \
            and not Path(bundle_file).is_symlink():
        try:
            text = Path(bundle_file).read_text()
        except OSError:
            text = ""
        for spec in _import_pattern().findall(text):
            if not spec.startswith(".") and not spec.startswith("/") and "://" not in spec:
                bare.setdefault(spec[:256], set()).add(str(bundle_file))
    return {"roots": [str(item) for item in roots], "reachable_files": len(reachable),
            "bare": {spec: sorted(importers) for spec, importers in sorted(bare.items())},
            "unresolved": unresolved[:200], "truncated": truncated}


def _resolve_bare_with_bun(scratch, pairs, tracker):
    script = ("const {createRequire} = require(\"module\");"
              "const pairs = JSON.parse(process.argv[1]);"
              "const out = [];"
              "for (const [spec, importer] of pairs) {"
              "  try { out.push({ok: true, resolved: createRequire(importer).resolve(spec)}); }"
              "  catch (error) { out.push({ok: false, error: String(error).split(\"\\n\")[0].slice(0, 200)}); }"
              "}"
              "console.log(JSON.stringify(out));")
    try:
        completed = _tracked_command(tracker, ["bun", "-e", script, json.dumps(pairs)],
                                     str(scratch), timeout=60)
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired, OSError):
        return None
    except ValueError as error:
        if not str(error).startswith("command output overflow"):
            raise
        return None
    try:
        return json.loads((completed.stdout or "").strip().splitlines()[-1])
    except (IndexError, json.JSONDecodeError):
        return None


def _builtin_modules(scratch, tracker):
    script = "console.log(JSON.stringify(require(\"module\").builtinModules));"
    try:
        completed = _tracked_command(tracker, ["bun", "-e", script], str(scratch),
                                     timeout=60)
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired, OSError):
        return None
    except ValueError as error:
        if not str(error).startswith("command output overflow"):
            raise
        return None
    try:
        names = json.loads((completed.stdout or "").strip().splitlines()[-1])
    except (IndexError, json.JSONDecodeError):
        return None
    if not isinstance(names, list) or not all(isinstance(item, str) for item in names):
        return None
    return set(names)


def _is_builtin_context(spec, resolved, builtins):
    echo = isinstance(resolved, str) and (
        resolved == "bun" or resolved.startswith("bun:") or resolved.startswith("node:"))
    return (spec == "bun" or spec.startswith("bun:") or spec.startswith("node:")
            or echo or (builtins is not None and spec in builtins))


def _classify_reachable(spec, importers, answers, runtime, builtins):
    contexts = []
    for importer, answer in zip(importers, answers):
        ok = isinstance(answer, dict) and answer.get("ok")
        resolved = answer.get("resolved", "") if isinstance(answer, dict) else ""
        contexts.append({"importer": importer,
                         "builtin": bool(_is_builtin_context(spec, resolved, builtins)),
                         "resolved": str(resolved)[:512] if ok else None,
                         "error": None if ok else (answer or {}).get("error", "resolver probe failed")})
    if contexts and all(item["builtin"] for item in contexts):
        return {"specifier": spec, "importers": list(importers),
                "resolution": {"status": "builtin", "runtime": runtime}}
    return {"specifier": spec, "importers": list(importers),
            "resolution": {"status": "unsupported",
                           "reason": "reachable file dependency lacks importer-specific ESM closure",
                           "contexts": contexts}}


def _classify_resolutions(pairs, answers, runtime, builtins=None):
    entries = []
    for position, (spec, importers) in enumerate(pairs):
        answer = answers[position] if isinstance(answers, list) and position < len(answers) else None
        if not isinstance(answer, dict) or not answer.get("ok"):
            if builtins is not None and spec in builtins:
                entries.append({"specifier": spec, "importers": importers,
                                "resolution": {"status": "builtin", "runtime": runtime,
                                               "via": "runtime builtinModules"}})
                continue
            entries.append({"specifier": spec, "importers": importers,
                            "resolution": {"status": "failed",
                                           "error": (answer or {}).get("error", "resolver probe failed")}})
            continue
        resolved = answer.get("resolved", "")
        echo_builtin = isinstance(resolved, str) and (
            resolved == "bun" or resolved.startswith("bun:") or resolved.startswith("node:"))
        if spec == "bun" or spec.startswith("bun:") or spec.startswith("node:") or echo_builtin \
                or (builtins is not None and spec in builtins):
            entries.append({"specifier": spec, "importers": importers,
                            "resolution": {"status": "builtin", "runtime": runtime}})
            continue
        target = Path(resolved) if isinstance(resolved, str) else None
        if target is not None and target.is_absolute() and target.is_file() \
                and not target.is_symlink():
            try:
                data = target.read_bytes()
            except OSError:
                data = None
            if data is not None:
                entries.append({"specifier": spec, "importers": importers,
                                "resolution": {"status": "file", "path": str(target),
                                               "realpath": str(target.resolve()),
                                               "sha256": hashlib.sha256(data).hexdigest(),
                                               "bytes": len(data)}})
                continue
        entries.append({"specifier": spec, "importers": importers,
                        "resolution": {"status": "ambiguous",
                                       "resolved": str(resolved)[:512]}})
    return entries


def collect_prepared_identities(scratch, repo, tracker):
    scratch, repo = Path(scratch), Path(repo)
    if Path(tracker.scratch) != scratch:
        raise ValueError("identity collection refused: tracker bound to a different scratch")
    ordinary = scratch / "generated-js"
    diagnostic = scratch / "generated-js-diag"
    bundle = scratch / "startup-ordinary-bundle.js"
    state_ts = scratch / "generated" / "program" / "state.ts"
    try:
        state_data = state_ts.read_bytes() if state_ts.is_file() and not state_ts.is_symlink() else None
    except OSError:
        state_data = None
    modules = scratch / "node_modules"
    if modules.is_symlink():
        try:
            node_target = {"link": os.readlink(modules)[:512], "realpath": str(modules.resolve())}
        except OSError:
            node_target = {"link": None, "realpath": None}
    else:
        node_target = None
    versions = collect_tool_versions(tracker)
    bun_revision = _tracked_probe(tracker, ["bun", "--revision"], scratch)
    runtime_id = {"bun": versions.get("bun"), "bun_revision": bun_revision}
    perfemit = scratch / "perfemit"
    try:
        perfemit_data = perfemit.read_bytes() if perfemit.is_file() and not perfemit.is_symlink() else None
    except OSError:
        perfemit_data = None
    walk = _walk_reachable([ordinary / "startup-facade.js", ordinary / "startup-minimal.js",
                            diagnostic / "startup-facade.js"], scratch, bundle_file=bundle)
    scanned = _bare_with_importers([ordinary, diagnostic])
    for spec, importers in walk["bare"].items():
        scanned.setdefault(spec, importers)
    reachable_pairs = [(spec, importer) for spec, importers in sorted(walk["bare"].items())
                       for importer in importers]
    extra_pairs = [(spec, scanned[spec][0]) for spec in sorted(scanned)
                   if spec not in walk["bare"]][:500]
    resolve_pairs = reachable_pairs + extra_pairs
    answers = _resolve_bare_with_bun(scratch, resolve_pairs, tracker) if resolve_pairs else []
    builtins = _builtin_modules(scratch, tracker) if resolve_pairs else set()
    ordered = answers if isinstance(answers, list) else []
    positions = {}
    for index in range(len(reachable_pairs)):
        positions.setdefault(reachable_pairs[index][0], []).append(index)
    reachable = [_classify_reachable(
        spec, walk["bare"][spec],
        [ordered[i] if i < len(ordered) else None for i in positions[spec]],
        runtime_id, builtins) for spec in sorted(walk["bare"])]
    extra_classified = _classify_resolutions(
        [(spec, scanned[spec]) for spec, _ in extra_pairs],
        ordered[len(reachable_pairs):], runtime_id, builtins=builtins)
    unreachable = [{"specifier": entry["specifier"], "files": entry["importers"],
                    "resolution": entry["resolution"]} for entry in extra_classified]
    return {
        "tool_versions": versions,
        "builtins": {"bun_revision": bun_revision},
        "executables": {
            "bun": _executable_identity("bun", versions.get("bun")),
            "go": _executable_identity("go", versions.get("go")),
            "node": _executable_identity("node", versions.get("node")),
        },
        "drivers": {
            "drivers/runtime.py": _sha256_or_none(repo / "tools" / "performance" / "drivers" / "runtime.py"),
            "drivers/runtime-transpile.ts": _sha256_or_none(repo / "tools" / "performance" / "drivers" / "runtime-transpile.ts"),
            "startup-attribution.py": _sha256_or_none(repo / "tools" / "performance" / "startup-attribution.py"),
            "startup-attribution.ts": _sha256_or_none(repo / "tools" / "performance" / "startup-attribution.ts"),
        },
        "sources": {
            "startup-facade.ts": _sha256_or_none(scratch / "generated" / "startup-facade.ts"),
            "startup-minimal.ts": _sha256_or_none(scratch / "generated" / "startup-minimal.ts"),
            "startup-probe.ts": _sha256_or_none(scratch / "instrument-src" / "program" / "startup-probe.ts"),
            "startup-launcher.js": _sha256_or_none(scratch / "startup-launcher.js"),
            "state.ts": ({"sha256": hashlib.sha256(state_data).hexdigest(), "bytes": len(state_data)}
                         if state_data is not None else None),
        },
        "inputs": {
            "scope": "conservative: fixture tree, module-owned emitter dependency files, build flags; stdlib identified by go version, output by built binary hash",
            "fixture": _hash_tree(repo / "tools" / "performance" / "fixtures" / "runtime"),
            "emitter": dict(_emitter_inputs(repo, tracker),
                            go_mod=_sha256_or_none(repo / "go.mod"),
                            binary=({"sha256": hashlib.sha256(perfemit_data).hexdigest(),
                                     "bytes": len(perfemit_data)}
                                    if perfemit_data is not None else None)),
            "build": {"gomaxprocs": "2",
                      "args": ["go", "build", "-p=2", "-o", "perfemit", "./compiler/perfemit"]},
        },
        "runtime_ts": {
            "scope": "conservative: whole checkout runtime TS, includes test-only inputs linked but unreachable from startup roots",
            "files": _hash_tree(repo / "runtime", (".ts",)),
        },
        "emitted": {
            "adapter": _sha256_or_none(scratch / "generated" / "generated-adapter.ts"),
            "packages": _hash_tree(scratch / "generated" / "packages", (".ts",)),
        },
        "prepared": {
            "ordinary": _inventory_js(ordinary),
            "diagnostic": _inventory_js(diagnostic),
            "bundle": ({"sha256": hashlib.sha256(bundle.read_bytes()).hexdigest(),
                        "bytes": bundle.stat().st_size}
                       if bundle.is_file() and not bundle.is_symlink() else None),
        },
        "resolved": {
            "computed": True,
            "scope": "builtin-only reachable closure; reachable file dependencies lack importer-specific ESM closure and fail completeness",
            "roots": walk["roots"],
            "reachable_files": walk["reachable_files"],
            "reachable_bare": reachable,
            "unresolved": walk["unresolved"],
            "truncated": walk["truncated"],
            "unreachable_bare": unreachable,
        },
        "dependencies": {
            "node_modules": node_target,
            "package_json": _sha256_or_none(repo / "package.json"),
            "bun_lock": _sha256_or_none(repo / "bun.lock"),
            "bare_specifiers": sorted(scanned),
        },
    }


def check_identity_complete(inv):
    def incomplete(what):
        raise ValueError(f"identity incomplete: {what}")
    if not isinstance(inv, dict):
        incomplete("inventory is not a mapping")
    versions = inv.get("tool_versions")
    if not isinstance(versions, dict):
        incomplete("tool_versions missing")
    for key in ("bun", "go", "node", "typescript"):
        value = versions.get(key)
        if not isinstance(value, str) or not value.strip():
            incomplete(f"tool_versions.{key} missing")
    builtins = inv.get("builtins") or {}
    if not isinstance(builtins.get("bun_revision"), str) or not builtins["bun_revision"].strip():
        incomplete("builtins.bun_revision missing")
    for key in ("bun", "go", "node"):
        entry = (inv.get("executables") or {}).get(key)
        if not isinstance(entry, dict) or not entry.get("realpath") or not entry.get("version") \
                or not isinstance(entry.get("dev"), int) or not isinstance(entry.get("ino"), int) \
                or not isinstance(entry.get("size"), int):
            incomplete(f"executables.{key} incomplete")
    drivers = inv.get("drivers") or {}
    for key in ("drivers/runtime.py", "drivers/runtime-transpile.ts",
                "startup-attribution.py", "startup-attribution.ts"):
        if not isinstance(drivers.get(key), str) or not drivers[key].strip():
            incomplete(f"drivers.{key} missing")
    sources = inv.get("sources") or {}
    for key in ("startup-facade.ts", "startup-minimal.ts", "startup-probe.ts",
                "startup-launcher.js"):
        if not isinstance(sources.get(key), str) or not sources[key].strip():
            incomplete(f"sources.{key} missing")
    state = sources.get("state.ts") or {}
    if not isinstance(state.get("sha256"), str) or not state["sha256"].strip() \
            or not isinstance(state.get("bytes"), int):
        incomplete("sources.state.ts incomplete")
    inputs = inv.get("inputs") or {}
    if not isinstance(inputs.get("fixture"), list) or not inputs["fixture"]:
        incomplete("inputs.fixture empty")
    emitter = inputs.get("emitter") or {}
    packages = emitter.get("packages")
    if not isinstance(packages, list) or not packages \
            or any(not isinstance(item, dict) or not item.get("files") for item in packages):
        incomplete("inputs.emitter.packages incomplete")
    if not isinstance(emitter.get("go_mod"), str) or not emitter["go_mod"].strip():
        incomplete("inputs.emitter.go_mod missing")
    binary = emitter.get("binary") or {}
    if not isinstance(binary.get("sha256"), str) or not binary["sha256"].strip() \
            or not isinstance(binary.get("bytes"), int):
        incomplete("inputs.emitter.binary incomplete")
    runtime_ts = inv.get("runtime_ts") or {}
    if not isinstance(runtime_ts.get("files"), list) or not runtime_ts["files"]:
        incomplete("runtime_ts.files empty")
    emitted = inv.get("emitted") or {}
    if not isinstance(emitted.get("adapter"), str) or not emitted["adapter"].strip():
        incomplete("emitted.adapter missing")
    if not isinstance(emitted.get("packages"), list) or not emitted["packages"]:
        incomplete("emitted.packages empty")
    prepared = inv.get("prepared") or {}
    for key in ("ordinary", "diagnostic"):
        graph = prepared.get(key) or {}
        if not isinstance(graph.get("files"), list) or not graph["files"]:
            incomplete(f"prepared.{key}.files empty")
    bundle = prepared.get("bundle") or {}
    if not isinstance(bundle.get("sha256"), str) or not bundle["sha256"].strip() \
            or not isinstance(bundle.get("bytes"), int):
        incomplete("prepared.bundle incomplete")
    resolved = inv.get("resolved") or {}
    if resolved.get("computed") is not True:
        incomplete("resolved.computed not true")
    if not isinstance(resolved.get("roots"), list) or not resolved["roots"]:
        incomplete("resolved.roots empty")
    if not isinstance(resolved.get("reachable_files"), int) or resolved["reachable_files"] < 1:
        incomplete("resolved.reachable_files empty")
    if resolved.get("truncated"):
        incomplete("resolved walk truncated")
    if resolved.get("unresolved"):
        first = resolved["unresolved"][0]
        incomplete(f"resolved.unresolved holds {first.get('specifier')} ({first.get('reason')})")
    for entry in resolved.get("reachable_bare") or []:
        resolution = entry.get("resolution") or {}
        status = resolution.get("status")
        if status == "builtin":
            if not isinstance((resolution.get("runtime") or {}).get("bun_revision"), str):
                incomplete(f"resolved builtin {entry.get('specifier')} lacks runtime identity")
        elif status in ("file", "unsupported"):
            incomplete(f"reachable file dependency lacks importer-specific closure: {entry.get('specifier')}")
        else:
            incomplete(f"required resolution failed: {entry.get('specifier')}")
    dependencies = inv.get("dependencies") or {}
    if not isinstance(dependencies.get("node_modules"), dict):
        incomplete("dependencies.node_modules missing")
    if not isinstance(dependencies.get("package_json"), str) \
            or not dependencies["package_json"].strip():
        incomplete("dependencies.package_json missing")
    if not isinstance(dependencies.get("bun_lock"), str) \
            or not dependencies["bun_lock"].strip():
        incomplete("dependencies.bun_lock missing")
    return True


def _first_difference(before, after, trail="identities"):
    if type(before) is not type(after):
        return f"{trail} type {type(before).__name__} != {type(after).__name__}"
    if isinstance(before, dict):
        if set(before) != set(after):
            return f"{trail} keys differ"
        for key in before:
            found = _first_difference(before[key], after[key], f"{trail}.{key}")
            if found:
                return found
        return None
    if isinstance(before, list):
        if len(before) != len(after):
            return f"{trail} length {len(before)} != {len(after)}"
        for index, (want, got) in enumerate(zip(before, after)):
            found = _first_difference(want, got, f"{trail}[{index}]")
            if found:
                return found
        return None
    if before != after:
        return f"{trail} {before!r} != {after!r}"
    return None


def check_identity_drift(before, after):
    if not isinstance(before, dict) or not isinstance(after, dict):
        raise ValueError("prepared identity drift: inventory is not a mapping")
    difference = _first_difference(before, after)
    if difference is not None:
        raise ValueError(f"prepared identity drift: {difference}")
    return True


def _write_output(path, record):
    path = Path(path)
    if os.path.lexists(path):
        raise ValueError(f"output refuses preexisting path: {path}")
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_name(f"{path.name}.tmp-{os.getpid()}")
    try:
        tmp.write_text(json.dumps(record, indent=2) + "\n")
        os.replace(tmp, path)
    except BaseException:
        try:
            tmp.unlink()
        except OSError:
            pass
        raise


def _record_failure(record, reason, retired=False, cleanup_error=None):
    scratch = {"retired": retired}
    if cleanup_error is not None:
        scratch["cleanup_error"] = cleanup_error
    record.update({"status": "failed", "reason": reason, "scratch": scratch})


def run_prepare_process(args, repo):
    scratch = Path(args.prepared)
    try:
        built = prepare(repo, scratch, args.child_timeout)
    except _Fail as error:
        try:
            _write_output(scratch / "preparation-result.json",
                          {"status": "failed", "error": str(error)})
        except (ValueError, OSError) as write_error:
            sys.stderr.write(f"startup preparation cannot write result: {write_error}\n")
        sys.stderr.write(f"startup preparation failed: {error}\n")
        return 1
    except Exception as error:
        sys.stderr.write(f"startup preparation internal {type(error).__name__}: {error}\n")
        return 1
    record = {"status": "complete", "manifest": built["manifest"], "prepared": built["prepared"],
              "checks": built["checks"], "timings": built["timings"],
              "scratch_bytes": built["scratch_bytes"], "scratch_files": built["scratch_files"]}
    try:
        _write_output(scratch / "preparation-result.json", record)
    except (ValueError, OSError) as error:
        sys.stderr.write(f"startup preparation cannot write result: {error}\n")
        return 1
    return 0


def _read_worker_error(scratch):
    try:
        result = json.loads((Path(scratch) / "preparation-result.json").read_text())
    except (OSError, json.JSONDecodeError):
        return None
    error = result.get("error")
    return error if isinstance(error, str) and error else None


def _validate_preparation_result(scratch, result):
    scratch = Path(scratch)
    if not isinstance(result, dict) or result.get("status") != "complete":
        raise _Fail(f"preparation worker reported failure: {result.get('error') if isinstance(result, dict) else result!r}")
    checks = result.get("checks")
    if not isinstance(checks, list) or not checks \
            or any(not isinstance(item, dict) or item.get("status") != "pass" for item in checks):
        raise _Fail("preparation worker checks incomplete")
    manifest = result.get("manifest")
    if not isinstance(manifest, dict) or not isinstance(manifest.get("probe"), dict):
        raise _Fail("preparation worker manifest missing")
    prepared = result.get("prepared")
    if (not isinstance(prepared, dict) or set(prepared.get("profiles", {})) != set(PROFILES)
            or not isinstance(prepared.get("statements"), int)):
        raise _Fail("preparation worker profiles incomplete")
    try:
        state_text = (scratch / "generated" / "program" / "state.ts").read_text()
        inventory = json.loads((scratch / "inventory-manifest.json").read_text())
        check_manifest_agreement(inventory, manifest["probe"])
        check_manifest_coverage(manifest["probe"], state_text)
    except (OSError, json.JSONDecodeError) as error:
        raise _Fail(f"preparation files unreadable: {error}")
    except ValueError as error:
        raise _Fail(f"preparation coverage recheck failed: {error}")
    return result


def run_prepare_worker(repo, scratch, child_timeout, prepare_bound, tracker):
    tool = str(Path(__file__).resolve())
    try:
        run_supervised([sys.executable, tool, "--repo", str(repo), "--prepare-run",
                        "--prepared", str(scratch), "--child-timeout", str(child_timeout)],
                       str(repo), timeout=prepare_bound, on_start=tracker.track)
    except subprocess.TimeoutExpired:
        raise _Fail("preparation bound exceeded")
    except subprocess.CalledProcessError as error:
        detail = _read_worker_error(scratch) or (error.stderr or "")[-800:]
        raise _Fail(f"preparation failed: {detail}")
    except RetirementFailed as error:
        raise _Fail("preparation retirement failed")
    tracker.clear_verified()
    try:
        result = json.loads((Path(scratch) / "preparation-result.json").read_text())
    except (OSError, json.JSONDecodeError) as error:
        raise _Fail(f"preparation result missing: {error}")
    return _validate_preparation_result(scratch, result)


def _failure_reason(error):
    parts, visited, node = [], set(), error
    while node is not None and id(node) not in visited and len(parts) < 4:
        visited.add(id(node))
        text = str(node)
        if text and all(text not in known and known not in text for known in parts):
            parts.append(text[:500])
        nxt = getattr(node, "__context__", None)
        if nxt is None:
            nxt = getattr(node, "__cause__", None)
        node = nxt
    return "; caused by: ".join(parts) if parts else repr(error)[:200]


def run_validate(args, repo):
    started = _utcnow()
    commit = _git_commit(repo)
    record = {"schema": SCHEMA, "mode": "validate-only", "status": "failed", "repo": str(repo),
              "commit": commit, "timestamps": {"start": started, "end": started},
              "bounds": {"overall": {"start": started, "end": started},
                         "preparation": None, "sampling": None}}
    keep_path = args.prepared_out
    owned = owned_scratch(args.scratch_parent, path=keep_path, keep=keep_path is not None)
    try:
        with _interrupts():
            with owned as scratch:
                tracker = _phase_tracker(scratch)
                prep_start = _utcnow()
                try:
                    prepared = run_prepare_worker(repo, scratch, args.child_timeout,
                                                  args.prepare_bound, tracker)
                finally:
                    record["bounds"]["preparation"] = {"start": prep_start, "end": _utcnow()}
                    pending = tracker.retire_pending()
                    if pending is not None:
                        record["group_retirement"] = pending
                try:
                    identities = collect_prepared_identities(scratch, repo, tracker)
                    check_identity_complete(identities)
                except ValueError as error:
                    raise _Fail(f"identity inventory incomplete: {error}")
                record.update({"status": "complete", "preparation": prepared["timings"],
                               "manifest": prepared["manifest"], "checks": prepared["checks"],
                               "identities": identities,
                               "scratch": {"bytes": prepared["scratch_bytes"],
                                           "compiled_files": prepared["scratch_files"],
                                           "retired": keep_path is None},
                               "prepared_out": str(keep_path) if keep_path else None})
                if keep_path is not None:
                    record["scratch"]["transferred"] = str(keep_path)
    except _Interrupted as error:
        _record_failure(record, f"interrupted: {error}", retired=_scratch_outcome(owned))
    except _Fail as error:
        _record_failure(record, _failure_reason(error), retired=_scratch_outcome(owned))
    except RetirementFailed as error:
        _record_failure(record, _failure_reason(error), retired=_scratch_outcome(owned),
                        cleanup_error=_failure_reason(error))
        if owned.path is not None and os.path.lexists(owned.path):
            record["retained_custody"] = _retained_custody(owned.path, str(error))
    except (ValueError, OSError) as error:
        message = _failure_reason(error)
        if message.startswith("scratch cleanup failed") or message.startswith("scratch ownership invalid") \
                or message.startswith("scratch registration failed") \
                or message.startswith("phase retirement failed"):
            _record_failure(record, message, retired=_scratch_outcome(owned), cleanup_error=message)
        else:
            _record_failure(record, f"internal {type(error).__name__}: {error}",
                            retired=_scratch_outcome(owned))
        if owned.path is not None and os.path.lexists(owned.path) and "retained_custody" not in record:
            record["retained_custody"] = _retained_custody(owned.path, message)
    except Exception as error:
        _record_failure(record, f"internal {type(error).__name__}: {error}",
                        retired=_scratch_outcome(owned))
    record["timestamps"]["end"] = _utcnow()
    record["bounds"]["overall"]["end"] = record["timestamps"]["end"]
    try:
        _write_output(args.output, record)
    except (ValueError, OSError) as error:
        sys.stderr.write(f"cannot write output record: {error}\n")
        return 1
    return 0 if record["status"] == "complete" else 1


def run_measure(args, repo):
    started = _utcnow()
    commit = _git_commit(repo)
    record = {"schema": SCHEMA, "mode": "measure", "status": "failed", "repo": str(repo),
              "commit": commit, "timestamps": {"start": started, "end": started},
              "bounds": {"overall": {"start": started, "end": started},
                         "preparation": None, "sampling": None},
              "trials": args.trials, "warmups": args.warmups, "batches": args.batches}
    owned = owned_scratch(args.scratch_parent)
    try:
        with _interrupts():
            with owned as scratch:
                tracker = _phase_tracker(scratch)
                prep_start = _utcnow()
                try:
                    prepared = run_prepare_worker(repo, scratch, args.child_timeout,
                                                  args.prepare_bound, tracker)
                finally:
                    record["bounds"]["preparation"] = {"start": prep_start, "end": _utcnow()}
                    pending = tracker.retire_pending()
                    if pending is not None:
                        record["group_retirement"] = pending
                try:
                    frozen = collect_prepared_identities(scratch, repo, tracker)
                    check_identity_complete(frozen)
                except ValueError as error:
                    raise _Fail(f"identity inventory incomplete: {error}")
                record.update({"preparation": prepared["timings"], "manifest": prepared["manifest"],
                               "checks": prepared["checks"]})
                record["identities"] = frozen
                activity_before = collect_activity()
                deadline = time.monotonic() + args.sampling_bound
                profile_orders = []
                all_rows = []
                tool = str(Path(__file__).resolve())
                sampling_start = _utcnow()
                try:
                    for trial in range(args.trials):
                        remaining = deadline - time.monotonic()
                        if remaining <= 0:
                            raise _Fail("sampling deadline exceeded")
                        profile_orders.append(trial_order(trial))
                        rows_path = scratch / f"rows-{trial}.json"
                        try:
                            run_supervised([sys.executable, tool, "--repo", str(repo), "--trial-run",
                                            "--trial-index", str(trial), "--prepared", str(scratch),
                                            "--rows", str(rows_path), "--warmups", str(args.warmups),
                                            "--batches", str(args.batches),
                                            "--child-timeout", str(args.child_timeout)],
                                           str(repo), timeout=remaining, on_start=tracker.track)
                        except subprocess.TimeoutExpired:
                            raise _Fail("sampling deadline exceeded")
                        except subprocess.CalledProcessError as error:
                            raise _Fail(f"trial {trial} failed: {(error.stderr or '')[-800:]}")
                        except RetirementFailed as error:
                            raise _Fail(f"trial {trial} retirement failed")
                        tracker.clear_verified()
                        try:
                            payload = json.loads(rows_path.read_text())
                        except (FileNotFoundError, json.JSONDecodeError) as error:
                            raise _Fail(f"trial {trial} rows missing: {error}")
                        if payload.get("trial_index") != trial:
                            raise _Fail(f"trial {trial} rows carry wrong trial index")
                        try:
                            check_trial_rows(payload.get("rows"), trial, args.warmups, args.batches)
                        except ValueError as error:
                            raise _Fail(f"trial {trial} rejected: {error}")
                        all_rows.extend(payload["rows"])
                finally:
                    record["bounds"]["sampling"] = {"start": sampling_start, "end": _utcnow()}
                    pending = tracker.retire_pending()
                    if pending is not None:
                        record["group_retirement"] = pending
                record["activity"] = {"before": activity_before, "after": collect_activity()}
                record["identities_after"] = collect_prepared_identities(scratch, repo, tracker)
                try:
                    check_identity_complete(record["identities_after"])
                    check_identity_drift(frozen, record["identities_after"])
                except ValueError as error:
                    raise _Fail(str(error))
                record["profile_orders"] = profile_orders
                record["rows"] = all_rows
                record["report"] = build_report(all_rows, args.trials, prepared["manifest"]["probe"],
                                                warmups=args.warmups, batches=args.batches)
                try:
                    scratch_bytes, scratch_files = check_caps(scratch)
                except ValueError as error:
                    raise _Fail(f"scratch cap check failed: {error}")
                record["scratch"] = {"bytes": scratch_bytes, "compiled_files": scratch_files,
                                     "retired": True}
                record["status"] = "complete"
    except _Interrupted as error:
        _record_failure(record, f"interrupted: {error}", retired=_scratch_outcome(owned))
    except _Fail as error:
        _record_failure(record, _failure_reason(error), retired=_scratch_outcome(owned))
    except RetirementFailed as error:
        _record_failure(record, _failure_reason(error), retired=_scratch_outcome(owned),
                        cleanup_error=_failure_reason(error))
        if owned.path is not None and os.path.lexists(owned.path):
            record["retained_custody"] = _retained_custody(owned.path, str(error))
    except (ValueError, OSError) as error:
        message = _failure_reason(error)
        if message.startswith("scratch cleanup failed") or message.startswith("scratch ownership invalid") \
                or message.startswith("scratch registration failed") \
                or message.startswith("phase retirement failed"):
            _record_failure(record, message, retired=_scratch_outcome(owned), cleanup_error=message)
        else:
            _record_failure(record, f"internal {type(error).__name__}: {error}",
                            retired=_scratch_outcome(owned))
        if owned.path is not None and os.path.lexists(owned.path) and "retained_custody" not in record:
            record["retained_custody"] = _retained_custody(owned.path, message)
    except Exception as error:
        _record_failure(record, f"internal {type(error).__name__}: {error}",
                        retired=_scratch_outcome(owned))
    record["timestamps"]["end"] = _utcnow()
    record["bounds"]["overall"]["end"] = record["timestamps"]["end"]
    try:
        _write_output(args.output, record)
    except (ValueError, OSError) as error:
        sys.stderr.write(f"cannot write output record: {error}\n")
        return 1
    return 0 if record["status"] == "complete" else 1


def run_trial_process(args):
    try:
        rows = run_trial_rows(args.prepared, args.trial_index, args.warmups, args.batches,
                              args.child_timeout)
        check_trial_rows(rows, args.trial_index, args.warmups, args.batches)
    except _Fail as error:
        sys.stderr.write(f"startup trial failed: {error}\n")
        return 1
    except ValueError as error:
        sys.stderr.write(f"startup trial rejected: {error}\n")
        return 1
    except Exception as error:
        sys.stderr.write(f"startup trial internal {type(error).__name__}: {error}\n")
        return 1
    try:
        _write_output(args.rows, {"trial_index": args.trial_index, "rows": rows})
    except (ValueError, OSError) as error:
        sys.stderr.write(f"startup trial cannot write rows: {error}\n")
        return 1
    return 0


def main(argv=None):
    args = parse_args(argv if argv is not None else sys.argv[1:])
    if args.trial_run or args.prepare_run:
        if args.measure or args.validate_only or args.prepared_out is not None \
                or args.output is not None or (args.trial_run and args.prepare_run):
            sys.stderr.write("usage: worker modes take only their own inputs\n")
            return 2
        if args.trial_run and (args.trial_index is None or args.prepared is None
                               or args.rows is None):
            sys.stderr.write("usage: --trial-run requires --trial-index, --prepared, --rows\n")
            return 2
        if args.prepare_run and args.prepared is None:
            sys.stderr.write("usage: --prepare-run requires --prepared\n")
            return 2
    else:
        if args.output is None:
            sys.stderr.write("usage: --output is required\n")
            return 2
        if args.measure and args.validate_only:
            sys.stderr.write("usage: --measure and --validate-only conflict\n")
            return 2
        if args.measure and args.prepared_out is not None:
            sys.stderr.write("usage: --prepared-out is validation-only\n")
            return 2
        if args.trials < 1 or args.batches < 1 or args.warmups < 0 or args.child_timeout <= 0 \
                or args.sampling_bound <= 0 or args.prepare_bound <= 0:
            sys.stderr.write("usage: trials/batches/timeouts out of range\n")
            return 2
    repo = args.repo.resolve()
    try:
        bind_driver(repo)
    except Exception as error:
        sys.stderr.write(f"cannot load existing runtime driver: {error}\n")
        return 2
    if driver_command is None or driver_adapter is None:
        sys.stderr.write("existing runtime driver lacks command/adapter\n")
        return 2
    if args.trial_run:
        return run_trial_process(args)
    if args.prepare_run:
        return run_prepare_process(args, repo)
    if args.measure:
        return run_measure(args, repo)
    return run_validate(args, repo)


if __name__ == "__main__":
    sys.exit(main())

