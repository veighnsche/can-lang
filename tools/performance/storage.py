"""Owned scratch lifetime; compact evidence remains usable without execution trees."""

import contextlib
import fcntl
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import time

MARKER = '.performance-owner.json'
SCRATCH = ('source', 'work', 'go-cache')
KEEP_SECONDS = 7 * 24 * 60 * 60


@contextlib.contextmanager
def interruption_signals():
    """Translate termination into the same cleanup path as Ctrl-C."""
    previous = signal.getsignal(signal.SIGTERM)
    def interrupt(signum, frame):
        raise KeyboardInterrupt('Received SIGTERM')
    signal.signal(signal.SIGTERM, interrupt)
    try:
        yield
    finally:
        signal.signal(signal.SIGTERM, previous)


@contextlib.contextmanager
def defer_interrupts(raise_after=False):
    """Keep teardown atomic with respect to repeated Ctrl-C/termination."""
    pending = []
    previous = {sig: signal.getsignal(sig) for sig in (signal.SIGINT, signal.SIGTERM)}
    def deferred(signum, frame):
        pending.append(signum)
    for sig in previous:
        signal.signal(sig, deferred)
    try:
        yield
    finally:
        for sig, handler in previous.items():
            signal.signal(sig, handler)
    if pending and raise_after:
        raise KeyboardInterrupt('Termination received during child startup')


def identity(info):
    return [info.st_dev, info.st_ino]


def pid_alive(pid, group=False):
    if not isinstance(pid, int) or isinstance(pid, bool) or pid <= 1:
        raise ValueError('Invalid recorded process identity')
    try:
        (os.killpg if group else os.kill)(pid, 0)
    except ProcessLookupError:
        return False
    except PermissionError:
        return True
    return True


def verify_root(root, root_fd, metadata, marker_fd):
    if (identity(os.fstat(root_fd)) != metadata.get('root_identity')
        or identity(os.stat(root, follow_symlinks=False)) != metadata.get('root_identity')
        or identity(os.fstat(marker_fd)) != metadata.get('marker_identity')
        or identity(os.stat(MARKER, dir_fd=root_fd, follow_symlinks=False)) != metadata.get('marker_identity')):
        raise ValueError('Run root or ownership marker was replaced; refusing cleanup')


def open_references(root):
    """Check same-user open files/cwd/executables, including detached children."""
    import subprocess
    executable = shutil.which('lsof')
    if executable is None:
        raise ValueError('Open-reference inspection unavailable; retaining scratch')
    result = subprocess.run([executable, '-nP', '-Fpn', '-u', str(os.getuid())],
                            capture_output=True, text=True, timeout=15)
    if result.returncode not in (0, 1) or result.stderr.strip() or not result.stdout.startswith('p'):
        raise ValueError('Open-reference inspection failed; retaining scratch')
    prefix = str(root) + os.sep
    process = None
    references = []
    for line in result.stdout.splitlines():
        if line.startswith('p'):
            process = int(line[1:])
        elif line.startswith('n') and process is not None and process != os.getpid():
            path = line[1:]
            if path == str(root) or path.startswith(prefix):
                references.append({'pid': process, 'path': path})
    return references


class Scratch:
    def __init__(self, root, keep=False):
        self.root = Path(root).absolute()
        self.root_fd = os.open(self.root, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
        try:
            fd = os.open(MARKER, os.O_CREAT | os.O_EXCL | os.O_RDWR | os.O_NOFOLLOW, 0o600, dir_fd=self.root_fd)
        except BaseException:
            os.close(self.root_fd)
            raise
        self.handle = os.fdopen(fd, 'r+')
        fcntl.flock(fd, fcntl.LOCK_EX)
        self.metadata = {'kind': 'can.performance-scratch', 'schema_version': 1,
                         'root': str(self.root), 'pid': os.getpid(), 'created_at': time.time(),
                         'root_identity': identity(os.fstat(self.root_fd)),
                         'marker_identity': identity(os.fstat(fd)),
                         'keep_until': time.time() + KEEP_SECONDS if keep else None,
                         'scratch': list(SCRATCH), 'process_group': None,
                         'launched_children': False, 'state': 'active'}
        self.save()

    def save(self):
        self.handle.seek(0)
        json.dump(self.metadata, self.handle)
        self.handle.truncate()
        self.handle.flush()
        os.fsync(self.handle.fileno())

    def child(self, pid):
        if pid is None:
            group = self.metadata['process_group']
            if group is not None and pid_alive(group, group=True):
                raise ValueError('Child group is still live; preserving ownership')
        else:
            self.metadata['launched_children'] = True
        self.metadata['process_group'] = pid
        self.save()

    def close(self):
        self.handle.close()
        os.close(self.root_fd)

    def finish(self):
        try:
            verify_root(self.root, self.root_fd, self.metadata, self.handle.fileno())
            group = self.metadata['process_group']
            if group is not None and pid_alive(group, group=True):
                raise ValueError('Child group is still live; retaining scratch')
            if self.metadata['keep_until'] is None:
                if self.metadata['launched_children'] and open_references(self.root):
                    raise ValueError('Open workspace references remain; retaining scratch')
                remove_scratch(self.root_fd)
                self.metadata['state'] = 'cleaned'
            else:
                self.metadata['state'] = 'retained'
            self.metadata['process_group'] = None
            self.save()
        finally:
            self.close()


def remove_scratch(root_fd):
    import stat
    if not shutil.rmtree.avoids_symlink_attacks:
        raise ValueError('Safe descriptor-relative deletion unavailable')
    # Descriptor-relative deletion cannot wander into a replacement root.
    for name in SCRATCH:
        try:
            info = os.stat(name, dir_fd=root_fd, follow_symlinks=False)
        except FileNotFoundError:
            continue
        if not stat.S_ISDIR(info.st_mode):
            raise ValueError('Scratch path is not a directory (possibly symlink): ' + name)
    for name in SCRATCH:
        try:
            shutil.rmtree(name, dir_fd=root_fd)
        except FileNotFoundError:
            pass


def reap(parent, now=None):
    """Reclaim only dead, marked immediate children; never remove evidence."""
    parent = Path(parent).absolute()
    if parent.is_symlink():
        raise ValueError('Reap root cannot be a symlink')
    if not parent.exists():
        return []
    now = time.time() if now is None else now
    results = []
    parent_fd = os.open(parent, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    try:
        for name in sorted(os.listdir(parent_fd)):
            root = parent / name
            try:
                root_fd = os.open(name, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=parent_fd)
            except OSError:
                continue
            try:
                try:
                    fd = os.open(MARKER, os.O_RDWR | os.O_NOFOLLOW, dir_fd=root_fd)
                except OSError:
                    continue
                with os.fdopen(fd, 'r+') as handle:
                    try:
                        fcntl.flock(handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
                    except BlockingIOError:
                        results.append({'run': str(root), 'status': 'active'})
                        continue
                    metadata = json.load(handle)
                    if (metadata.get('kind') != 'can.performance-scratch' or metadata.get('schema_version') != 1
                        or metadata.get('root') != str(root) or metadata.get('scratch') != list(SCRATCH)):
                        raise ValueError('Unrecognized run ownership')
                    verify_root(root, root_fd, metadata, handle.fileno())
                    if metadata.get('state') == 'cleaned':
                        continue
                    if pid_alive(metadata.get('pid')):
                        results.append({'run': str(root), 'status': 'owner-still-running'})
                        continue
                    group = metadata.get('process_group')
                    if group is not None and pid_alive(group, group=True):
                        results.append({'run': str(root), 'status': 'child-still-running'})
                        continue
                    expires = metadata.get('keep_until')
                    if expires is not None and expires > now:
                        results.append({'run': str(root), 'status': 'retained', 'expires_at': expires})
                        continue
                    if metadata.get('launched_children') and open_references(root):
                        results.append({'run': str(root), 'status': 'workspace-still-open'})
                        continue
                    verify_root(root, root_fd, metadata, handle.fileno())
                    remove_scratch(root_fd)
                    metadata.update(state='cleaned', process_group=None)
                    handle.seek(0)
                    json.dump(metadata, handle)
                    handle.truncate()
                    handle.flush()
                    results.append({'run': str(root), 'status': 'cleaned'})
            except (OSError, ValueError, TypeError, subprocess.SubprocessError) as exc:
                results.append({'run': str(root), 'status': 'refused', 'reason': str(exc)})
            finally:
                os.close(root_fd)
    finally:
        os.close(parent_fd)
    return results


def archive_evidence(root):
    """Verify an atomic archive before descriptor-relative loose-file removal."""
    import hashlib
    import stat
    import zipfile
    root = Path(root)
    root_fd = os.open(root, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    raw_fd = None
    root_identity = identity(os.fstat(root_fd))
    try:
        records = {}
        for name in ('manifest.json', 'summary.json', 'report.md', 'isolation.jsonl'):
            if name in os.listdir(root_fd):
                info = os.stat(name, dir_fd=root_fd, follow_symlinks=False)
                if not stat.S_ISREG(info.st_mode):
                    raise ValueError('Evidence is not a regular file: ' + name)
                records[name] = (root_fd, name, identity(info))
        raw_identity = None
        if 'raw' in os.listdir(root_fd):
            raw_fd = os.open('raw', os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=root_fd)
            raw_identity = identity(os.fstat(raw_fd))
            for name in os.listdir(raw_fd):
                info = os.stat(name, dir_fd=raw_fd, follow_symlinks=False)
                if not stat.S_ISREG(info.st_mode):
                    raise ValueError('Raw evidence must be flat regular files: ' + name)
                records['raw/' + name] = (raw_fd, name, identity(info))
        names = list(records)

        def verify_locations():
            if identity(os.stat(root, follow_symlinks=False)) != root_identity:
                raise ValueError('Evidence root was replaced; preserving loose files')
            if raw_fd is not None:
                if identity(os.stat('raw', dir_fd=root_fd, follow_symlinks=False)) != raw_identity:
                    raise ValueError('Raw evidence directory was replaced; preserving loose files')
                expected = {leaf for directory_fd, leaf, _ in records.values() if directory_fd == raw_fd}
                if set(os.listdir(raw_fd)) != expected:
                    raise ValueError('Raw evidence inventory changed; preserving loose files')
            for directory_fd, leaf, expected in records.values():
                if identity(os.stat(leaf, dir_fd=directory_fd, follow_symlinks=False)) != expected:
                    raise ValueError('Evidence record was replaced; preserving loose files')

        if 'evidence.zip' in os.listdir(root_fd):
            fd = os.open('evidence.zip', os.O_RDONLY | os.O_NOFOLLOW, dir_fd=root_fd)
            with os.fdopen(fd, 'rb') as handle, zipfile.ZipFile(handle) as existing:
                if existing.testzip() is not None or existing.namelist().count('manifest.json') != 1 or len(existing.namelist()) != len(set(existing.namelist())):
                    raise ValueError('Existing evidence archive is invalid')
            if names:
                raise ValueError('Evidence archive already exists; refusing replacement')
            return  # Safe retry after successful compaction.
        if 'manifest.json' not in names:
            raise ValueError('Cannot archive evidence without its manifest')
        temporary = '.evidence.zip.tmp'
        fd = os.open(temporary, os.O_CREAT | os.O_EXCL | os.O_RDWR | os.O_NOFOLLOW, 0o600, dir_fd=root_fd)
        hashes = {}
        try:
            with os.fdopen(fd, 'w+b') as output:
                with zipfile.ZipFile(output, 'w', compression=zipfile.ZIP_DEFLATED) as archive:
                    for name in sorted(names):
                        directory_fd, leaf, expected_identity = records[name]
                        source_fd = os.open(leaf, os.O_RDONLY | os.O_NOFOLLOW, dir_fd=directory_fd)
                        with os.fdopen(source_fd, 'rb') as source, archive.open(name, 'w') as destination:
                            if not stat.S_ISREG(os.fstat(source.fileno()).st_mode) or identity(os.fstat(source.fileno())) != expected_identity:
                                raise ValueError('Evidence is not a regular file: ' + name)
                            digest = hashlib.sha256()
                            while block := source.read(1024 * 1024):
                                digest.update(block)
                                destination.write(block)
                            hashes[name] = digest.hexdigest()
                output.flush()
                os.fsync(output.fileno())
                output.seek(0)
                with zipfile.ZipFile(output) as verification:
                    if verification.testzip() is not None or set(verification.namelist()) != set(names):
                        raise ValueError('Evidence archive verification failed')
                    for name, expected in hashes.items():
                        with verification.open(name) as entry:
                            digest = hashlib.sha256()
                            while block := entry.read(1024 * 1024):
                                digest.update(block)
                            if digest.hexdigest() != expected:
                                raise ValueError('Evidence archive checksum mismatch: ' + name)
            verify_locations()
            # Link exclusively: a competing archive cannot be overwritten.
            os.link(temporary, 'evidence.zip', src_dir_fd=root_fd, dst_dir_fd=root_fd, follow_symlinks=False)
            for name in names:
                verify_locations()
                directory_fd, leaf, _ = records[name]
                os.unlink(leaf, dir_fd=directory_fd)
                del records[name]
            if raw_fd is not None:
                verify_locations()
                os.rmdir('raw', dir_fd=root_fd)
        finally:
            os.unlink(temporary, dir_fd=root_fd)
    except (zipfile.BadZipFile, zipfile.LargeZipFile, RuntimeError, EOFError, NotImplementedError) as exc:
        raise ValueError('Invalid evidence archive') from exc
    finally:
        if raw_fd is not None:
            os.close(raw_fd)
        os.close(root_fd)


def read_evidence(root, name):
    """Read exactly one record without extracting archived paths."""
    import zipfile
    root = Path(root)
    archive_path = root if root.is_file() else root / 'evidence.zip'
    if archive_path.exists():
        try:
            with zipfile.ZipFile(archive_path) as archive:
                if archive.namelist().count(name) != 1:
                    raise ValueError(f'Missing or duplicate archive evidence: {name}')
                return archive.read(name).decode('utf-8')
        except (zipfile.BadZipFile, zipfile.LargeZipFile, RuntimeError, EOFError, UnicodeError, NotImplementedError) as exc:
            raise ValueError(f'Invalid evidence archive: {archive_path}') from exc
    return (root / name).read_text()
