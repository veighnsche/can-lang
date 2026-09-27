"""Small storage lifecycle checks; no production workloads or toolchain builds."""
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

import storage


class StorageTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        child = subprocess.Popen([sys.executable, '-c', 'pass'])
        child.wait(timeout=3)
        cls.dead_pid = child.pid

    def setUp(self):
        self.references = patch.object(storage, 'open_references', return_value=[])
        self.references.start()
        self.addCleanup(self.references.stop)

    def abandon(self, owner):
        owner.metadata['pid'] = self.dead_pid
        owner.save()
        owner.close()

    def dead_owner_marker(self, root):
        marker = root / storage.MARKER
        metadata = json.loads(marker.read_text())
        metadata['pid'] = self.dead_pid
        marker.write_text(json.dumps(metadata))

    def make_run(self, parent, keep=False):
        root = parent / 'run'
        root.mkdir()
        owner = storage.Scratch(root, keep)
        for name in storage.SCRATCH:
            (root / name).mkdir()
            (root / name / 'payload').write_text('scratch')
        (root / 'raw').mkdir()
        (root / 'raw' / 'trial.json').write_text('{}')
        return root, owner

    def test_default_deletes_only_scratch_and_preserves_evidence(self):
        with tempfile.TemporaryDirectory() as temp:
            root, owner = self.make_run(Path(temp))
            owner.finish()
            self.assertTrue((root / 'raw' / 'trial.json').exists())
            self.assertTrue(all(not (root / name).exists() for name in storage.SCRATCH))

    def test_reaper_skips_active_and_reclaims_expired_retention(self):
        with tempfile.TemporaryDirectory() as temp:
            parent = Path(temp)
            root, owner = self.make_run(parent, keep=True)
            self.assertEqual(storage.reap(parent)[0]['status'], 'active')
            expires = owner.metadata['keep_until']
            owner.finish()
            self.dead_owner_marker(root)
            self.assertEqual(storage.reap(parent, now=expires - 1)[0]['status'], 'retained')
            self.assertEqual(storage.reap(parent, now=expires + 1)[0]['status'], 'cleaned')
            self.assertFalse((root / 'source').exists())
            self.assertTrue((root / 'raw').exists())

    def test_abandoned_owner_reaped_but_unmarked_and_symlink_untouched(self):
        with tempfile.TemporaryDirectory() as temp:
            parent = Path(temp)
            root, owner = self.make_run(parent)
            self.abandon(owner)  # Simulate a genuinely dead owner with released OS lock.
            foreign = parent / 'foreign'
            foreign.mkdir()
            (foreign / 'source').mkdir()
            (parent / 'link').symlink_to(root, target_is_directory=True)
            self.assertEqual(storage.reap(parent)[0]['status'], 'cleaned')
            self.assertTrue((foreign / 'source').exists())
            self.assertTrue((parent / 'link').is_symlink())

    def test_substituted_top_level_symlink_refuses_cleanup(self):
        with tempfile.TemporaryDirectory() as temp:
            root, owner = self.make_run(Path(temp))
            (root / 'source' / 'payload').unlink()
            (root / 'source').rmdir()
            external = Path(temp) / 'external'
            external.mkdir()
            (external / 'payload').write_text('preserve')
            (root / 'source').symlink_to(external, target_is_directory=True)
            with self.assertRaisesRegex(ValueError, 'symlink'):
                owner.finish()
            self.assertEqual((external / 'payload').read_text(), 'preserve')

    def test_nested_symlink_does_not_delete_external_data(self):
        with tempfile.TemporaryDirectory() as temp:
            parent = Path(temp)
            root, owner = self.make_run(parent)
            external = parent / 'external'
            external.mkdir()
            (external / 'data').write_text('preserve')
            (root / 'source' / 'link').symlink_to(external, target_is_directory=True)
            owner.finish()
            self.assertEqual((external / 'data').read_text(), 'preserve')

    def test_live_orphan_group_is_not_reclaimed(self):
        with tempfile.TemporaryDirectory() as temp:
            parent = Path(temp)
            root, owner = self.make_run(parent)
            child = subprocess.Popen([sys.executable, '-c', 'import time; time.sleep(10)'], start_new_session=True)
            try:
                owner.child(child.pid)
                self.abandon(owner)
                self.assertEqual(storage.reap(parent)[0]['status'], 'child-still-running')
                self.assertTrue((root / 'source').exists())
            finally:
                child.terminate()
                child.wait(timeout=3)
            self.assertEqual(storage.reap(parent)[0]['status'], 'cleaned')

    def test_failed_group_shutdown_keeps_marker_and_scratch(self):
        import isolation
        with tempfile.TemporaryDirectory() as temp:
            root, owner = self.make_run(Path(temp))
            children = []
            real_popen = subprocess.Popen
            def launch(*args, **kwargs):
                process = real_popen(*args, **kwargs)
                children.append(process)
                return process
            try:
                with patch.object(isolation, 'PROCESS_OBSERVER', owner.child), patch.object(isolation.subprocess, 'Popen', side_effect=launch), patch.object(isolation, 'stop_group', side_effect=TimeoutError('injected shutdown failure')):
                    with self.assertRaisesRegex(TimeoutError, 'injected'):
                        isolation.execute([sys.executable, '-c', 'import time;time.sleep(10)'], root,
                                          os.environ.copy(), root / 'out', root / 'err', .03, root / 'events', False)
                with self.assertRaisesRegex(ValueError, 'live'):
                    owner.finish()
                self.assertEqual(json.loads((root / storage.MARKER).read_text())['process_group'], children[0].pid)
                self.assertTrue((root / 'work').exists())
            finally:
                for child in children:
                    os.killpg(child.pid, signal.SIGKILL)
                    child.wait(timeout=3)

    def test_signal_during_popen_registration_still_stops_child(self):
        import isolation
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            observed = []
            children = []
            real_popen = subprocess.Popen
            def launch(*args, **kwargs):
                child = real_popen(*args, **kwargs)
                children.append(child)
                os.kill(os.getpid(), signal.SIGTERM)
                return child
            with patch.object(isolation, 'PROCESS_OBSERVER', observed.append), patch.object(isolation.subprocess, 'Popen', side_effect=launch), self.assertRaises(KeyboardInterrupt):
                isolation.execute([sys.executable, '-c', 'import time;time.sleep(10)'], root,
                                  os.environ.copy(), root / 'out', root / 'err', 1, root / 'events', False)
            self.assertEqual(observed, [children[0].pid, None])
            self.assertIsNotNone(children[0].poll())

    def test_alive_owner_with_unlocked_marker_is_not_reclaimed(self):
        with tempfile.TemporaryDirectory() as temp:
            root, owner = self.make_run(Path(temp))
            owner.close()
            self.assertEqual(storage.reap(Path(temp))[0]['status'], 'owner-still-running')
            self.assertTrue((root / 'source').exists())

    def test_replaced_marker_refuses_reaping(self):
        with tempfile.TemporaryDirectory() as temp:
            root, owner = self.make_run(Path(temp))
            self.abandon(owner)
            marker = root / storage.MARKER
            replacement = root / 'replacement'
            replacement.write_bytes(marker.read_bytes())
            replacement.replace(marker)
            result = storage.reap(Path(temp))
            self.assertEqual(result[0]['status'], 'refused')
            self.assertTrue((root / 'source').exists())

    def test_replaced_root_never_deletes_foreign_tree(self):
        with tempfile.TemporaryDirectory() as temp:
            root, owner = self.make_run(Path(temp))
            moved = root.with_name('original')
            root.rename(moved)
            root.mkdir()
            (root / 'source').mkdir()
            (root / 'source' / 'foreign').write_text('preserve')
            with self.assertRaisesRegex(ValueError, 'replaced'):
                owner.finish()
            self.assertEqual((root / 'source' / 'foreign').read_text(), 'preserve')
            self.assertTrue((moved / 'source').exists())

    def test_live_group_preserved_when_finish_is_attempted(self):
        with tempfile.TemporaryDirectory() as temp:
            root, owner = self.make_run(Path(temp))
            child = subprocess.Popen([sys.executable, '-c', 'import time;time.sleep(10)'], start_new_session=True)
            try:
                owner.child(child.pid)
                with self.assertRaisesRegex(ValueError, 'live'):
                    owner.finish()
                self.assertEqual(json.loads((root / storage.MARKER).read_text())['process_group'], child.pid)
                self.assertTrue((root / 'source').exists())
            finally:
                child.terminate()
                child.wait(timeout=3)

    def test_detached_open_reference_prevents_cleanup(self):
        with tempfile.TemporaryDirectory() as temp:
            root, owner = self.make_run(Path(temp))
            owner.metadata['launched_children'] = True
            owner.save()
            with patch.object(storage, 'open_references', return_value=[{'pid': 12345, 'path': str(root / 'work')}]), self.assertRaisesRegex(ValueError, 'references'):
                owner.finish()
            self.assertTrue((root / 'work').exists())

    def test_repeated_signals_cannot_interrupt_teardown(self):
        with storage.defer_interrupts():
            os.kill(os.getpid(), signal.SIGTERM)
            os.kill(os.getpid(), signal.SIGINT)

    def test_sigterm_uses_interrupt_cleanup_and_restores_handler(self):
        before = signal.getsignal(signal.SIGTERM)
        with self.assertRaises(KeyboardInterrupt):
            with storage.interruption_signals():
                os.kill(os.getpid(), signal.SIGTERM)
        self.assertEqual(signal.getsignal(signal.SIGTERM), before)

    def test_raw_replacement_preserves_foreign_directory_and_symlink(self):
        import zipfile
        for symlink in (False, True):
            with self.subTest(symlink=symlink), tempfile.TemporaryDirectory() as temp:
                root, owner = self.make_run(Path(temp))
                (root / 'manifest.json').write_text('{}')
                owner.finish()
                foreign = Path(temp) / 'foreign'
                foreign.mkdir()
                (foreign / 'trial.json').write_text('foreign')
                real_testzip = zipfile.ZipFile.testzip
                def replace_raw(archive):
                    result = real_testzip(archive)
                    (root / 'raw').rename(root / 'original-raw')
                    if symlink:
                        (root / 'raw').symlink_to(foreign, target_is_directory=True)
                    else:
                        foreign.rename(root / 'raw')
                    return result
                with patch.object(zipfile.ZipFile, 'testzip', side_effect=replace_raw, autospec=True), self.assertRaisesRegex(ValueError, 'replaced'):
                    storage.archive_evidence(root)
                self.assertEqual((root / 'raw/trial.json').read_text(), 'foreign')
                self.assertEqual((root / 'original-raw/trial.json').read_text(), '{}')
                self.assertTrue((root / 'manifest.json').exists())
                self.assertFalse((root / 'evidence.zip').exists())

    def test_raw_nested_directory_is_refused(self):
        with tempfile.TemporaryDirectory() as temp:
            root, owner = self.make_run(Path(temp))
            (root / 'manifest.json').write_text('{}')
            (root / 'raw/nested').mkdir()
            (root / 'raw/nested/file').write_text('preserve')
            owner.finish()
            with self.assertRaisesRegex(ValueError, 'flat'):
                storage.archive_evidence(root)
            self.assertEqual((root / 'raw/nested/file').read_text(), 'preserve')

    def test_failed_archive_verification_preserves_loose_evidence(self):
        import zipfile
        with tempfile.TemporaryDirectory() as temp:
            root, owner = self.make_run(Path(temp))
            (root / 'manifest.json').write_text('{}')
            owner.finish()
            with patch.object(zipfile.ZipFile, 'testzip', return_value='corrupt'), self.assertRaisesRegex(ValueError, 'verification'):
                storage.archive_evidence(root)
            self.assertEqual((root / 'raw/trial.json').read_text(), '{}')
            self.assertTrue((root / 'manifest.json').exists())
            self.assertFalse((root / 'evidence.zip').exists())

    def test_archive_preserves_raw_records_without_execution_trees(self):
        with tempfile.TemporaryDirectory() as temp:
            root, owner = self.make_run(Path(temp))
            (root / "manifest.json").write_text("{\"status\":\"complete\"}")
            owner.finish()
            storage.archive_evidence(root)
            self.assertEqual(storage.read_evidence(root, "raw/trial.json"), "{}")
            self.assertEqual(storage.read_evidence(root / "evidence.zip", "manifest.json"), "{\"status\":\"complete\"}")
            self.assertFalse((root / "raw").exists())
            self.assertFalse((root / "manifest.json").exists())
            archive = (root / "evidence.zip").read_bytes()
            storage.archive_evidence(root)
            self.assertEqual((root / "evidence.zip").read_bytes(), archive)
            (root / "manifest.json").write_text("foreign")
            with self.assertRaisesRegex(ValueError, "refusing replacement"):
                storage.archive_evidence(root)
            self.assertEqual((root / "evidence.zip").read_bytes(), archive)


if __name__ == '__main__':
    unittest.main()
