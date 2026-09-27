"""Render saved performance evidence with the bundled Typst template."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile


COMPILE_TIMEOUT_SECONDS = 30


def _validate(data):
    if not isinstance(data, dict) or data.get("kind") != "can.performance-report" or data.get("schema_version") != 1:
        raise ValueError("PDF export requires a can.performance-report with schema_version 1")
    for field in ("manifest", "cases", "rankings"):
        if not isinstance(data.get(field), dict):
            raise ValueError(f"Performance report {field} must be an object")
    if any(not isinstance(case, dict) for case in data["cases"].values()):
        raise ValueError("Performance report cases must contain objects")
    if any(not isinstance(board, dict) for board in data["rankings"].values()):
        raise ValueError("Performance report rankings must contain objects")


def write_pdf(data, output):
    """Compile evidence locally and publish a complete PDF without replacing files."""
    _validate(data)
    output = Path(output).expanduser()
    if output.suffix.lower() != ".pdf":
        raise ValueError("PDF output must have a .pdf extension")
    if os.path.lexists(output):
        raise FileExistsError(f"PDF output already exists: {output}")
    compiler = shutil.which("typst")
    if compiler is None:
        raise RuntimeError("PDF export requires an installed typst executable on PATH")
    # Serialize before allocating work; reject non-JSON values and nonfinite numbers.
    serialized = json.dumps(data, ensure_ascii=False, allow_nan=False)
    with tempfile.TemporaryDirectory(prefix="can-performance-pdf-") as temporary:
        root = Path(temporary)
        shutil.copyfile(Path(__file__).with_name("report.typ"), root / "report.typ")
        (root / "report.json").write_text(serialized, encoding="utf-8")
        compiled = root / "report.pdf"
        try:
            result = subprocess.run(
                [compiler, "compile", "--jobs", "1", "--ignore-system-fonts", "--root", str(root),
                 str(root / "report.typ"), str(compiled)],
                cwd=root, capture_output=True, text=True, timeout=COMPILE_TIMEOUT_SECONDS,
            )
        except subprocess.TimeoutExpired as error:
            raise RuntimeError(f"Typst PDF compilation exceeded {COMPILE_TIMEOUT_SECONDS} seconds") from error
        except OSError as error:
            raise RuntimeError(f"Could not run Typst: {error}") from error
        if result.returncode:
            detail = (result.stderr or result.stdout or "no diagnostic output").strip()[:2000]
            raise RuntimeError(f"Typst PDF compilation failed: {detail}")
        if not compiled.is_file():
            raise RuntimeError("Typst did not produce a PDF")
        with compiled.open("rb") as source:
            if source.read(5) != b"%PDF-":
                raise RuntimeError("Typst output is not a PDF")
            source.seek(0)
            # A sibling ensures the exclusive hard link stays on the same filesystem.
            # Both temporary contexts reclaim work on every handled failure.
            with tempfile.NamedTemporaryFile(prefix=".can-performance-pdf-", suffix=".tmp", dir=output.parent) as staging:
                shutil.copyfileobj(source, staging)
                staging.flush()
                os.link(staging.name, output)
