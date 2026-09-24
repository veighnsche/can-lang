"""Run two isolated rejection probes using a bundled canlc path."""

from pathlib import Path
import shutil
import subprocess
import sys
import tempfile


root = Path(__file__).resolve().parent
compiler = Path(sys.argv[1]).resolve()
cases = {
    "missing_arm": (
        "        domain::b_failure as failure => domain::a_failure(failure.reason)\n",
        "",
    ),
    "wrong_operation": (
        "helper::audit_fixed(callable a_action, callable double, callable positive)",
        "helper::audit_fixed(callable a_action, callable double, callable double)",
    ),
}
with tempfile.TemporaryDirectory(prefix="can-generic-helper-negative-", dir="/private/tmp") as scratch:
    for name, (old, new) in cases.items():
        target = Path(scratch) / name
        target.mkdir()
        shutil.copytree(root / "src", target / "src")
        for file in ("can.project.json", "can.errors.json"):
            shutil.copy(root / file, target / file)
        source = target / "src/main/main.can"
        original = source.read_text()
        assert original.count(old) >= 1
        source.write_text(original.replace(old, new, 1))
        run = subprocess.run(
            [str(compiler), "build", str(target)], capture_output=True, text=True
        )
        print(f"{name}: exit {run.returncode}")
        print((run.stderr or run.stdout).strip())
        assert run.returncode != 0
