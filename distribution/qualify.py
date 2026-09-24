#!/usr/bin/env python3
"""Qualify a local upstream archive offline; never locate or acquire a runtime."""
import argparse
import hashlib
import json
from pathlib import Path
import platform
import shutil
import subprocess
import tempfile
import zipfile


def digest(data):
    return hashlib.sha256(data).hexdigest()


def machine_arch(machine):
    return {"x86_64": "amd64", "aarch64": "arm64"}.get(machine, machine)


def os_release():
    fields = {}
    for line in Path("/etc/os-release").read_text().splitlines():
        if "=" in line:
            key, value = line.split("=", 1)
            fields[key] = value.strip('"')
    return fields


def glibc_version():
    out = subprocess.run(["getconf", "GNU_LIBC_VERSION"], capture_output=True, text=True, timeout=10)
    if out.returncode or not out.stdout.strip():
        raise RuntimeError("GNU libc version unavailable")
    return out.stdout.strip()


def unshare_available():
    try:
        probe = subprocess.run(["unshare", "-n", "true"], capture_output=True, timeout=10)
        return probe.returncode == 0
    except (OSError, subprocess.SubprocessError):
        return False


def no_uplink_interfaces():
    """True when no non-loopback interface exists or is up. Only a smoke
    check behind an explicit caller isolation claim, never a substitute
    for the unshare namespace."""
    net = Path("/sys/class/net")
    if not net.is_dir():
        return False
    for iface in net.iterdir():
        if iface.name == "lo":
            continue
        state = iface / "operstate"
        if not state.is_file():
            continue
        try:
            if state.read_text().strip() not in ("down", "dormant"):
                return False
        except OSError:
            return False
    return True


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--archive", required=True, type=Path)
    parser.add_argument("--report", required=True, type=Path)
    parser.add_argument("--target-file", default="distribution/target.json",
                        help="pinned target manifest; resolved against the checkout when relative")
    parser.add_argument("--network-isolation", default="",
                        help="Linux only: external network-denial mechanism (e.g. 'docker run --network none'); "
                             "required when unshare is unavailable, and verified against live interfaces")
    args = parser.parse_args()
    source = Path(__file__).resolve().parent.parent
    target_path = Path(args.target_file)
    if not target_path.is_absolute():
        target_path = source / target_path
    manifest_bytes = target_path.read_bytes()
    target = json.loads(manifest_bytes)
    report_path = args.report.resolve()
    report_path.parent.mkdir(parents=True, exist_ok=True)
    report = {"schemaVersion": 1, "kind": "can.native-capability-report",
              "targetId": target["targetId"], "manifestSHA256": digest(manifest_bytes),
              "passed": False, "failures": []}
    try:
        wanted = target["runtime"]
        os_version = ""
        if wanted["platform"] == "linux":
            if platform.system() != "Linux" or machine_arch(platform.machine()) != wanted["architecture"]:
                raise RuntimeError("qualification requires native Linux amd64; no fallback")
            release = os_release()
            if release.get("ID") != "debian":
                raise RuntimeError(f"qualification requires Debian, found {release.get('ID')!r}")
            os_version = release.get("VERSION_ID", "")
            if not os_version or tuple(map(int, os_version.split("."))) < tuple(map(int, wanted["minimumOSVersion"].split("."))):
                raise RuntimeError("unsupported Debian version")
            libc = glibc_version()
        elif wanted["platform"] == "darwin":
            if args.network_isolation:
                raise RuntimeError("--network-isolation applies to the Linux target only")
            if platform.system() != "Darwin" or platform.machine() != wanted["architecture"]:
                raise RuntimeError("qualification requires native macOS arm64; no fallback")
            os_version = platform.mac_ver()[0]
            if tuple(map(int, os_version.split("."))) < tuple(map(int, wanted["minimumOSVersion"].split("."))):
                raise RuntimeError("unsupported macOS version")
            libc = ""
        else:
            raise RuntimeError(f"unsupported target platform {wanted['platform']!r}")
        archive = args.archive.read_bytes()
        if len(archive) != target["upstream"]["size"] or digest(archive) != target["upstream"]["sha256"]:
            raise RuntimeError("upstream archive size or SHA-256 mismatch")
        with zipfile.ZipFile(args.archive) as bundle:
            executable = bundle.read(target["upstream"]["member"])
        if digest(executable) != target["runtime"]["sha256"]:
            raise RuntimeError("extracted runtime SHA-256 mismatch")
        with tempfile.TemporaryDirectory(prefix="can-qualification-") as temporary:
            work = Path(temporary).resolve()
            root = work / target["targetId"]
            runtime = root / wanted["executable"]
            runtime.parent.mkdir(parents=True)
            runtime.write_bytes(executable)
            runtime.chmod(0o755)
            (root / "distribution").mkdir()
            (root / "distribution/target.json").write_bytes(manifest_bytes)
            suite = root / "tests/conformance"
            suite.mkdir(parents=True)
            for name in ("native.ts", "native.test.ts"):
                shutil.copyfile(source / "tests/conformance" / name, suite / name)
            home = work / "home"
            home.mkdir()
            cwd = work / "unrelated-cwd"
            cwd.mkdir()
            # No ambient Bun config, preload, package discovery, PATH runtime, or network.
            env = {"HOME": str(home), "PATH": "/nonexistent", "TMPDIR": str(work)}
            if wanted["platform"] == "darwin":
                network = "denied by macOS sandbox"
                prefix = ["/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", str(runtime)]
            elif unshare_available():
                network = "denied by Linux network namespace (unshare -n)"
                prefix = ["unshare", "-n", str(runtime)]
            elif args.network_isolation:
                if not no_uplink_interfaces():
                    raise RuntimeError("claimed network isolation is not in effect: a non-loopback interface is up")
                network = f"caller-provided: {args.network_isolation}"
                prefix = [str(runtime)]
            else:
                raise RuntimeError("Linux network denial unavailable: unshare -n failed and --network-isolation was not given")
            commands = [prefix + ["test", str(suite / "native.test.ts")],
                        prefix + [str(suite / "native.ts"), str(root), os_version]]
            test_run = subprocess.run(commands[0], cwd=cwd, env=env, capture_output=True, text=True, timeout=120)
            print(test_run.stderr, end="")
            result = subprocess.run(commands[1], cwd=cwd, env=env, capture_output=True, text=True, timeout=120)
            if not result.stdout.strip():
                raise RuntimeError("staged runtime produced no report: " + result.stderr)
            report = json.loads(result.stdout)
            report["execution"] = {"network": network, "path": env["PATH"],
                                   "cwd": str(cwd), "command": commands[1],
                                   "archiveSHA256": digest(archive), "negativeTestsPassed": test_run.returncode == 0}
            if libc:
                report["execution"]["glibc"] = libc
            if test_run.returncode or result.returncode:
                report["passed"] = False
                report["failures"].append("staged qualification or rejection tests failed")
    except Exception as error:
        report["passed"] = False
        report["failures"].append(str(error))
    report_path.write_text(json.dumps(report, indent=2) + "\n")
    print(f"Native qualification {'passed' if report['passed'] else 'FAILED'}: {report_path}")
    return 0 if report["passed"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
