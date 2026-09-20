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


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--archive', required=True, type=Path)
    parser.add_argument('--report', required=True, type=Path)
    args = parser.parse_args()
    source = Path(__file__).resolve().parent.parent
    manifest_bytes = (source / 'distribution/target.json').read_bytes()
    target = json.loads(manifest_bytes)
    report_path = args.report.resolve()
    report_path.parent.mkdir(parents=True, exist_ok=True)
    report = {'schemaVersion': 1, 'kind': 'can.native-capability-report',
              'targetId': target['targetId'], 'manifestSHA256': digest(manifest_bytes),
              'passed': False, 'failures': []}
    try:
        if platform.system() != 'Darwin' or platform.machine() != target['runtime']['architecture']:
            raise RuntimeError('qualification requires native macOS arm64; no fallback')
        os_version = platform.mac_ver()[0]
        if tuple(map(int, os_version.split('.'))) < tuple(map(int, target['runtime']['minimumOSVersion'].split('.'))):
            raise RuntimeError('unsupported macOS version')
        archive = args.archive.read_bytes()
        if len(archive) != target['upstream']['size'] or digest(archive) != target['upstream']['sha256']:
            raise RuntimeError('upstream archive size or SHA-256 mismatch')
        with zipfile.ZipFile(args.archive) as bundle:
            executable = bundle.read(target['upstream']['member'])
        if digest(executable) != target['runtime']['sha256']:
            raise RuntimeError('extracted runtime SHA-256 mismatch')
        with tempfile.TemporaryDirectory(prefix='can-qualification-') as temporary:
            work = Path(temporary).resolve()
            root = work / target['targetId']
            runtime = root / target['runtime']['executable']
            runtime.parent.mkdir(parents=True)
            runtime.write_bytes(executable)
            runtime.chmod(0o755)
            (root / 'distribution').mkdir()
            (root / 'distribution/target.json').write_bytes(manifest_bytes)
            suite = root / 'tests/conformance'
            suite.mkdir(parents=True)
            for name in ('native.ts', 'native.test.ts'):
                shutil.copyfile(source / 'tests/conformance' / name, suite / name)
            home = work / 'home'
            home.mkdir()
            cwd = work / 'unrelated-cwd'
            cwd.mkdir()
            # No ambient Bun config, preload, package discovery, PATH runtime, or network.
            env = {'HOME': str(home), 'PATH': '/nonexistent', 'TMPDIR': str(work)}
            prefix = ['/usr/bin/sandbox-exec', '-p', '(version 1)(allow default)(deny network*)', str(runtime)]
            commands = [prefix + ['test', str(suite / 'native.test.ts')],
                        prefix + [str(suite / 'native.ts'), str(root), os_version]]
            test_run = subprocess.run(commands[0], cwd=cwd, env=env, capture_output=True, text=True, timeout=60)
            print(test_run.stderr, end='')
            result = subprocess.run(commands[1], cwd=cwd, env=env, capture_output=True, text=True, timeout=60)
            if not result.stdout.strip():
                raise RuntimeError('staged runtime produced no report: ' + result.stderr)
            report = json.loads(result.stdout)
            report['execution'] = {'network': 'denied by macOS sandbox', 'path': env['PATH'],
                                   'cwd': str(cwd), 'command': commands[1],
                                   'archiveSHA256': digest(archive), 'negativeTestsPassed': test_run.returncode == 0}
            if test_run.returncode or result.returncode:
                report['passed'] = False
                report['failures'].append('staged qualification or rejection tests failed')
    except Exception as error:
        report['passed'] = False
        report['failures'].append(str(error))
    report_path.write_text(json.dumps(report, indent=2) + '\n')
    print(f"Native qualification {'passed' if report['passed'] else 'FAILED'}: {report_path}")
    return 0 if report['passed'] else 1


if __name__ == '__main__':
    raise SystemExit(main())
