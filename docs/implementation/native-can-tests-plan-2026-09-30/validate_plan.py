"""Plan/ledger validation during implementation; does not execute Can or test harnesses.

Repaired 2026-10-01 (review.md "validator scope conflict"): the frozen
all-planned assertion is replaced with validation of the four defined
statuses (planned/active/blocked/complete). This script validates the
source/master ledger agreement WITHOUT overwriting tasks.json, retains
every structural/hash/coverage check, requires recorded evidence and
reachable owned-path commit provenance for complete tasks, and rejects a
complete task whose required prerequisites remain unfinished.
planning-baseline.json stays a frozen historical snapshot and is only
read, never written.

Source-retirement note: tracked-file hashes are checked for every file
WITHOUT a recorded retirement commit. No retirement fields exist yet;
the first Z02 retirement must extend retirement-map.json records and
both validators together. Until then, any tracked source change fails
loudly here.

Exit 0 writes validation.json and prints it; any failure raises ValueError
(or records the sibling failure) without writing validation.json.
"""
from collections import Counter
from datetime import datetime, timezone
from hashlib import sha256
import fnmatch
import json
from pathlib import Path
import re
import subprocess

HERE = Path(__file__).resolve().parent
REPO = HERE.parents[2]
LEDGER = REPO / 'docs/syntax-taste/preparation/native-can-tests-migration-ledger-2026-09-30'
STATUSES = {'planned', 'active', 'blocked', 'complete'}

def read(path):
    return json.loads(path.read_text())

def require(condition, message):
    if not condition:
        raise ValueError(message)

def git(args):
    return subprocess.run(['git', '-C', str(REPO)] + args, capture_output=True, text=True)

def commit_touches_owned(token, patterns):
    proc = git(['show', '--name-only', '--pretty=format:', '-m', token])
    require(proc.returncode == 0, 'cannot list commit files: ' + token)
    for path in [line for line in proc.stdout.splitlines() if line.strip()]:
        for pattern in patterns:
            if '*' in pattern:
                if fnmatch.fnmatchcase(path, pattern):
                    return True
            elif pattern.endswith('/'):
                if path.startswith(pattern):
                    return True
            elif path == pattern:
                return True
    return False

parts = [read(HERE / f) for f in ['foundation.json', 'capabilities.json', 'integration.json', 'migration-tasks.json']]
tasks = [t for part in parts for t in part['tasks']]
ids = [t['id'] for t in tasks]
require(len(ids) == len(set(ids)), 'duplicate task IDs')
by_id = {t['id']: t for t in tasks}
required = ['title', 'lane', 'start_after', 'accept_after', 'files', 'changes', 'acceptance', 'evidence', 'commit', 'status']
for t in tasks:
    require(all(k in t for k in required), 'missing task field: ' + t['id'])
    require(t['status'] in STATUSES, 'bad status: ' + t['id'] + ' = ' + repr(t.get('status')))
    require(bool(t['files']) and bool(t['acceptance']) and bool(t['evidence']), 'incomplete task card: ' + t['id'])
    for dep in t['start_after'] + t['accept_after']:
        require(dep in by_id, 'unknown prerequisite: ' + t['id'] + ' -> ' + dep)
seen, visiting, order = set(), set(), []
def visit(id):
    if id in seen:
        return
    require(id not in visiting, 'dependency cycle: ' + id)
    visiting.add(id)
    for dep in by_id[id]['start_after'] + by_id[id]['accept_after']:
        visit(dep)
    visiting.remove(id)
    seen.add(id)
    order.append(id)
for id in ids:
    visit(id)

# Source/master agreement is validated, never regenerated: truthful
# implementation state in the source files must equal the aggregate.
aggregate = read(HERE / 'tasks.json')
require(aggregate['tasks'] == tasks, 'tasks.json aggregate differs from source task files')
require(aggregate['topological_completion_order'] == order, 'tasks.json order differs from recomputed order')

# Complete-task truthfulness: recorded evidence, reachable commits touching
# owned paths, and finished prerequisites.
complete_ids = [t['id'] for t in tasks if t['status'] == 'complete']
for t in tasks:
    if t['status'] != 'complete':
        continue
    evidence_path = HERE / 'evidence' / (t['id'] + '.json')
    require(evidence_path.is_file(), 'complete task lacks evidence file: ' + t['id'])
    evidence = read(evidence_path)
    commits = evidence.get('commits', [])
    require(bool(commits), 'complete task lacks recorded commits: ' + t['id'])
    for entry in commits:
        token = entry.split()[0]
        require(re.fullmatch(r'[0-9a-f]{7,40}', token) is not None, 'complete task has unparsable commit ref: ' + t['id'] + ' <- ' + entry)
        proc = git(['cat-file', '-e', token + '^{commit}'])
        require(proc.returncode == 0, 'complete task cites unreachable commit: ' + t['id'] + ' <- ' + entry)
        require(commit_touches_owned(token, t['files']), 'complete-task commit touches no owned path: ' + t['id'] + ' <- ' + entry)
    for dep in t['start_after'] + t['accept_after']:
        require(by_id[dep]['status'] == 'complete', 'complete task has unfinished prerequisite: ' + t['id'] + ' -> ' + dep + ' (' + by_id[dep]['status'] + ')')

rows = [r for f in ['core', 'browser', 'sql-history', 'lifecycle', 'external-callers'] for r in read(LEDGER / (f + '.json'))['rows']]
row_ids = {r['id'] for r in rows}
require(len(rows) == len(row_ids) == 292, 'source ledger mismatch')
migrations = [t for t in tasks if t['id'].startswith('M')]
assigned = [id for t in migrations for id in t['row_ids']]
require(Counter(assigned) == Counter(row_ids), 'migration rows omitted/duplicated')
require(len({t['files'][0] for t in migrations}) == 45, 'migration ownership overlap')
coverage = read(HERE / 'coverage-map.json')
cm = coverage['row_assignments']
require(Counter(r['row_id'] for r in cm) == Counter(row_ids), 'coverage map mismatch')
original = {r['id']: r for r in rows}
for r in cm:
    require(r['task_id'] in by_id, 'unknown migration owner')
    for field in ['source', 'protects', 'observes', 'variants', 'disposition']:
        require(r[field] == original[r['row_id']][field], 'changed source obligation: ' + r['row_id'] + '/' + field)
    require(bool(r.get('substeps')), 'row lacks execution substeps: ' + r['row_id'])
    for dep in r.get('start_after', []) + r.get('accept_after', []) + r.get('qualification_gates', []) + r.get('if_retained_accept_after', []):
        require(dep in by_id, 'unknown row gate: ' + r['row_id'] + '/' + dep)

source_index = read(LEDGER / 'source-index.json')['files']
require(len(source_index) == 224, 'source file count mismatch')
for f in source_index:
    p = REPO / f['path']
    require(p.is_file() and sha256(p.read_bytes()).hexdigest() == f['sha256'], 'source drift: ' + f['path'])
delegates = read(LEDGER / 'delegated-oracles.json')['suites']
require(len(delegates) == 30, 'delegate count mismatch')
for d in delegates:
    require(sha256((REPO / d['source']).read_bytes()).hexdigest() == d['sha256'], 'delegate drift: ' + d['source'])
assigned_delegates = coverage['delegated_oracle_assignments']
require(Counter(d['id'] for d in assigned_delegates) == Counter(d['id'] for d in delegates), 'delegate owner mismatch')
for d in assigned_delegates:
    require(d['owner_group'] in by_id, 'missing delegate owner')
    original_delegate = next(x for x in delegates if x['id'] == d['id'])
    require(d['source_sha256'] == original_delegate['sha256'], 'delegate hash omitted/changed')
    require(set(d['parent_rows']) == set(original_delegate['parent_rows']), 'delegate parent mapping changed')
retirement = read(HERE / 'retirement-map.json')
require(len(retirement['tracked_test_files']) == 224, 'retirement file count mismatch')
require(len(retirement['external_callers']) == 16, 'external caller count mismatch')
require({f['path'] for f in retirement['tracked_test_files']} == {f['path'] for f in source_index}, 'retirement paths differ')
for f in retirement['tracked_test_files']:
    require(f['owner'] == 'integrator', 'shared-file retirement owner mismatch')

hard_gates = {g: 'Q' + g for g in ['N1', 'N2', 'N3', 'B1', 'B2', 'B3', 'B4', 'B5', 'F1', 'F2', 'D1', 'D2', 'D3', 'D4']}
require(all(q in by_id for q in hard_gates.values()), 'missing integrated hard gate')
baseline = read(HERE / 'planning-baseline.json')
require(all(x == 'unrun' for x in baseline['integrated_gates'].values()), 'planning baseline gates changed')
require(baseline['head'] == '4cf8f91d97f45cf937c3464de54bc6275e70d49f', 'planning baseline head changed')
links = []
for p in [HERE.parent / 'native-can-tests-plan-2026-09-30.md'] + sorted(HERE.glob('*.md')):
    for target in re.findall(r'\[[^\]]*\]\(([^)]+)\)', p.read_text()):
        target = target.strip('<>')
        if '://' in target or target.startswith('#'):
            continue
        target = re.sub(r':\d+$', '', target.split('#')[0])
        if target:
            linked = (p.parent / target).resolve()
            require(linked.exists() or linked == HERE / 'validation.json' or linked == HERE / 'tasks.json', 'broken plan link: ' + str(p) + ' -> ' + target)
            links.append(target)

sibling = subprocess.run(['python3', str(HERE / 'tools' / 'check-implementation.py')], capture_output=True, text=True, cwd=str(HERE))
try:
    sibling_summary = json.loads(sibling.stdout) if sibling.stdout.strip() else None
except ValueError:
    sibling_summary = None
sibling_result = {'command': 'python3 tools/check-implementation.py', 'exit_code': sibling.returncode}
if sibling_summary is not None:
    sibling_result['summary'] = sibling_summary
else:
    sibling_result['error'] = (sibling.stdout + sibling.stderr)[-2000:]
require(sibling.returncode == 0, 'sibling implementation validator failed')

head = git(['rev-parse', 'HEAD'])
require(head.returncode == 0, 'cannot read HEAD')
result = {
    'validated_utc': datetime.now(timezone.utc).isoformat(),
    'scope': 'static plan/ledger validation during implementation; no qualification',
    'audited_head': head.stdout.strip(),
    'task_count': len(tasks),
    'task_counts': {prefix: sum(t['id'].startswith(prefix) for t in tasks) for prefix in ['P', 'K', 'I', 'Q', 'M', 'Z']},
    'status_counts': dict(Counter(t['status'] for t in tasks)),
    'unique_ids': True,
    'master_equals_source_views': True,
    'all_prerequisites_resolved': True,
    'acyclic_completion_graph': True,
    'complete_tasks_evidence_verified': len(complete_ids),
    'complete_task_commits_reachable_and_owned': True,
    'complete_prerequisites_satisfied': True,
    'ledger_rows_exactly_once': len(rows),
    'distinct_migration_prefixes': len(migrations),
    'unchanged_tracked_test_files': len(source_index),
    'unchanged_delegated_oracle_suites': len(delegates),
    'external_caller_records': len(retirement['external_callers']),
    'local_links_checked': len(links),
    'all_14_integrated_gates': 'unrun; explicit tasks exist',
    'validator_results': [sibling_result, {'command': 'python3 validate_plan.py', 'exit_code': 0}],
    'both_requested_validators_pass': True,
    'audit_round': {
        'implementation_executed': False,
        'tests_or_builds_executed': False,
        'browser_or_service_launched': False,
        'performance_measurements': False,
        'harnesses_deleted': 0,
        'runtime_coverage_credited': 0,
        'temporary_workspaces_allocated': 0,
        'cleanup_failures': [],
    },
    'evidence_limit': 'Static ledger validation only. Complete-task evidence files and commit provenance are checked for presence and reachability, not re-executed or requalified.',
}
(HERE / 'validation.json').write_text(json.dumps(result, indent=2) + '\n')
print(json.dumps(result, indent=2))
