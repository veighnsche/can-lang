"""Focused fixture binding and launcher-descriptor regression checks."""
import importlib.util
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("runtime_driver", Path(__file__).with_name("runtime.py"))
runtime = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runtime)


class RuntimeDriverTests(unittest.TestCase):
    def test_adapter_finds_origin_bindings(self):
        with tempfile.TemporaryDirectory() as directory:
            generated = Path(directory)
            module = generated / "packages" / "sample.ts"
            module.parent.mkdir()
            module.write_text('export async function $canFunction8(input) {\nlet $canOrigin = "gallery::doubled";\n}\nexport async function $canFunction21(input) {\nlet $canOrigin = "gallery::frequency";\n}\n')
            runtime.adapter(generated, ("doubled", "frequency"))
            text = (generated / "generated-adapter.ts").read_text()
            self.assertIn('$canFunction8 as doubled', text)
            self.assertIn('$canFunction21 as frequency', text)
            self.assertIn('./packages/sample.ts', text)

    def test_adapter_records_binding_provenance(self):
        import hashlib
        import json
        with tempfile.TemporaryDirectory() as directory:
            generated = Path(directory)
            module = generated / "packages" / "sample.ts"
            module.parent.mkdir()
            module.write_text('export async function $canFunction8(input) {\nlet $canOrigin = "gallery::doubled";\n}\n')
            runtime.adapter(generated, ("doubled",))
            inventory = json.loads((generated / "bindings.json").read_text())
            self.assertEqual(inventory[0]["module_sha256"], hashlib.sha256(module.read_bytes()).hexdigest())
            self.assertEqual(inventory[0]["module_bytes"], module.stat().st_size)

    def test_adapter_rejects_ambiguous_binding(self):
        with tempfile.TemporaryDirectory() as directory:
            generated = Path(directory)
            (generated / "packages").mkdir()
            for name in ("one", "two"):
                (generated / "packages" / (name + ".ts")).write_text('export async function $canFunction8(input) {\nlet $canOrigin = "gallery::doubled";\n}\n')
            with self.assertRaisesRegex(ValueError, "ambiguous generated binding"):
                runtime.adapter(generated, ("doubled",))

    def test_adapter_rejects_missing_binding(self):
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaisesRegex(ValueError, "requested workload bindings"):
                runtime.adapter(Path(directory))

    def test_bun_snapshot_preserves_parent_fd_three(self):
        # Preserve the test runner's own descriptor too; exercise an existing fd3.
        try:
            previous = os.dup(3)
        except OSError:
            previous = None
        with tempfile.TemporaryFile() as fixture:
            fixture.write(b"parent-state")
            fixture.seek(0)
            os.dup2(fixture.fileno(), 3)
            try:
                def child(args, **options):
                    self.assertEqual(options['pass_fds'], (3,))
                    self.assertEqual(os.read(3, 32), b"{}")
                    return object()
                with patch.object(runtime.subprocess, "run", side_effect=child):
                    runtime.command(["bun", "fixture.ts"], Path("/tmp"))
                self.assertEqual(os.read(3, 32), b"parent-state")
            finally:
                if previous is not None:
                    os.dup2(previous, 3)
                    os.close(previous)
                elif fixture.fileno() != 3:
                    os.close(3)


if __name__ == "__main__":
    unittest.main()
