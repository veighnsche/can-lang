"""Cooperative exclusion and observable quiet-host checks, not CPU reservation."""

import contextlib
import fcntl
import json
import os
from pathlib import Path
import re
import shlex
import signal
import subprocess
import sys
import time

LOCK = Path("/tmp/can-lang-perf-audit-2026-09-27.lock")


def log(path, event, **fields):
    with path.open("a") as handle:
        handle.write(json.dumps({"time": time.time(), "event": event, **fields}, allow_nan=False) + "\n")


@contextlib.contextmanager
def exclusive(timeout, evidence):
    started = time.monotonic()
    with LOCK.open("a") as handle:
        while True:
            try:
                fcntl.flock(handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
                break
            except BlockingIOError:
                if time.monotonic() - started >= timeout:
                    raise TimeoutError("Another participating build/benchmark holds the shared lock")
                time.sleep(min(0.25, timeout))
        log(evidence, "lock-acquired", wait_seconds=time.monotonic() - started)
        try:
            yield
        finally:
            fcntl.flock(handle, fcntl.LOCK_UN)
            log(evidence, "lock-released")


def parse_processes(output):
    result = {}
    for line in output.splitlines():
        parts = line.strip().split(None, 2)
        if len(parts) == 3:
            result[int(parts[0])] = (int(parts[1]), parts[2])
    return result


def related(rows, parent, child=None):
    ignored = set()
    while parent and parent not in ignored:
        ignored.add(parent)
        parent = rows.get(parent, (0, ""))[0]
    if child is not None:
        descendants = {child}
        while True:
            children = {pid for pid, (ppid, _) in rows.items() if ppid in descendants}
            if children <= descendants:
                break
            descendants |= children
        ignored |= descendants
    return ignored


def kind(command):
    try:
        words = shlex.split(command)
    except ValueError:
        words = command.split()
    if not words:
        return None
    executable, args = Path(words[0]).name.lower(), words[1:]
    if executable == "canlc":
        return None if args[:1] == ["lsp"] else "can-compiler"
    if executable in {"rustc", "clang", "clang++", "gcc", "g++", "cc1", "compile", "link"}:
        return "compiler"
    if executable in {"make", "ninja", "pytest", "jest", "vitest", "xcodebuild"} or executable.endswith(".test"):
        return "test/build"
    if executable in {"go", "cargo"} and any(x in {"build", "test", "run", "bench", "check"} for x in args[:2]):
        return "test/build"
    if executable.startswith("python") or executable in {"bun", "node", "npm", "npx", "pnpm", "yarn", "deno"}:
        for arg in args[:3]:
            if arg in {"-c", "-e", "--eval"}:
                break
            filename = Path(arg).name.lower()
            # Waiting coordinators must not deadlock the quiet gate. Their
            # actual build/test children remain visible to process inspection.
            if filename in {"perf.py", "reproduce.py", "locked.py"}:
                return None
            if "/tools/performance/" in arg:
                return "performance-driver"
            if re.search(r"(^|[._:-])(test|tests|bench|benchmark|perf|performance|check|build|pytest)([._:-]|$)", filename):
                return "test/build/benchmark"
    return None


def competing(child=None):
    output = subprocess.check_output(["ps", "-axo", "pid=,ppid=,args="], text=True, stderr=subprocess.PIPE, timeout=10)
    rows = parse_processes(output)
    if not rows or os.getpid() not in rows:
        raise RuntimeError("Process inspection is unavailable; isolation cannot be verified")
    ignore = related(rows, os.getpid(), child)
    return [{"pid": pid, "kind": category} for pid, (_, command) in rows.items()
            if pid not in ignore and (category := kind(command))]


def idle_percent():
    if sys.platform == "darwin":
        output = subprocess.check_output(["top", "-l", "2", "-n", "0", "-s", "1"], text=True, stderr=subprocess.PIPE, timeout=15)
        matches = re.findall(r"CPU usage:.*?([\d.]+)% idle", output)
        if not matches:
            raise RuntimeError("Cannot observe CPU idle time")
        return float(matches[-1])
    if sys.platform.startswith("linux"):
        def sample():
            fields = list(map(int, Path("/proc/stat").read_text().splitlines()[0].split()[1:]))
            return sum(fields[:8]), fields[3] + fields[4]
        before, idle_before = sample()
        time.sleep(1)
        after, idle_after = sample()
        if after <= before:
            raise RuntimeError("Invalid CPU observation interval")
        return 100 * (idle_after - idle_before) / (after - before)
    raise RuntimeError("Quiet-host observation supports macOS and Linux")


def quiet_window(seconds, minimum_idle, timeout, evidence):
    started, quiet_since, announced = time.monotonic(), None, -float("inf")
    while time.monotonic() - started < timeout:
        jobs, idle = competing(), idle_percent()
        now = time.monotonic()
        quiet_since = (quiet_since if quiet_since is not None else now) if not jobs and idle >= minimum_idle else None
        duration = now - quiet_since if quiet_since is not None else 0
        log(evidence, "quiet-sample", idle_percent=idle, competing=jobs, continuous_seconds=duration)
        if now - announced >= 20:
            print(f"Quiet gate: {idle:.0f}% idle, {len(jobs)} competing jobs, {duration:.0f}/{seconds:g}s", flush=True)
            announced = now
        if quiet_since is not None and duration >= seconds:
            log(evidence, "quiet-ready")
            return
        time.sleep(min(1, max(0, timeout - (time.monotonic() - started))))
    raise TimeoutError("No sustained quiet window; no accepted measurement started for this suite")


def stop_group(process):
    try:
        os.killpg(process.pid, signal.SIGTERM)
    except ProcessLookupError:
        return
    try:
        process.wait(timeout=3)
    except subprocess.TimeoutExpired:
        pass
    # Children may outlive a parent that already exited.
    try:
        os.killpg(process.pid, signal.SIGKILL)
    except ProcessLookupError:
        pass


def execute(command, cwd, environment, stdout, stderr, timeout, evidence, monitor):
    started = time.monotonic()
    with stdout.open("w") as out, stderr.open("w") as err:
        process = subprocess.Popen(command, cwd=cwd, env=environment, stdout=out, stderr=err, start_new_session=True)
        try:
            next_check = started + 2
            while process.poll() is None:
                now = time.monotonic()
                if now - started >= timeout:
                    raise TimeoutError("Driver exceeded its execution limit")
                if monitor and now >= next_check:
                    jobs = competing(process.pid)
                    log(evidence, "monitor", competing=jobs)
                    if jobs:
                        raise RuntimeError("Competing work appeared; trial is invalid")
                    next_check = now + 2
                time.sleep(0.1)
            if monitor:
                jobs = competing(process.pid)
                log(evidence, "monitor-final", competing=jobs)
                if jobs:
                    raise RuntimeError("Competing work was observed at trial completion")
            return process.returncode, time.monotonic() - started
        finally:
            stop_group(process)
