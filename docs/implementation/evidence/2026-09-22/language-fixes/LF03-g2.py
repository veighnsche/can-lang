"""LF02 G2 rerun under LF03: assert must never change production current.

Cases (each on a fresh copy of the frozen gap projects):
 1. passing: build -> record current -> assert -> current identical, exit 0.
 2. failing-assertion: build -> record current -> assert -> exit 1, current identical.
 3. pending-assertion: build -> record current -> assert (externally killed at 4s)
    -> current identical.
 4. no-prior-current: fresh failing-assertion copy, assert WITHOUT build ->
    exit 1, no dist/current.json created, staged test generation exists.
Writes results ONLY to the scratch out path.
"""
import json, os, pathlib, shutil, signal, subprocess, sys, time
HERE = pathlib.Path('/Users/vince/Projects/can-lang/docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects')
launcher = pathlib.Path(sys.argv[1]).resolve()
work = pathlib.Path(sys.argv[2]).resolve()
out = pathlib.Path(sys.argv[3]).resolve()
work.mkdir(parents=True, exist_ok=True)
results = []
def current_of(name):
    p = work / name / 'dist/current.json'
    return p.read_text() if p.exists() else None
def builds_of(name):
    d = work / name / 'dist/builds'
    return sorted(x.name for x in d.iterdir()) if d.exists() else []
def run(label, args, timeout=15):
    start = time.monotonic()
    p = subprocess.Popen([str(launcher), *args], stdout=subprocess.PIPE,
                         stderr=subprocess.PIPE, start_new_session=True)
    killed = False
    try:
        out_b, err = p.communicate(timeout=timeout)
    except subprocess.TimeoutExpired:
        killed = True
        os.killpg(p.pid, signal.SIGKILL)
        out_b, err = p.communicate(timeout=3)
    row = {'label': label, 'exit_code': p.returncode, 'externally_terminated': killed,
           'elapsed_seconds': round(time.monotonic() - start, 3),
           'stdout_tail': out_b.decode(errors='replace')[-600:],
           'stderr_tail': err.decode(errors='replace')[-600:]}
    results.append(row)
    print(json.dumps({k: row[k] for k in ['label', 'exit_code', 'externally_terminated']}), flush=True)
    return row
for name in ['passing', 'failing-assertion', 'pending-assertion']:
    dst = work / name
    if dst.exists():
        shutil.rmtree(dst)
    shutil.copytree(HERE / name, dst)
    run(name + ' build', ['build', str(dst)])
    before = current_of(name)
    row = run(name + ' assert', ['assert', str(dst)], timeout=4 if name == 'pending-assertion' else 15)
    after = current_of(name)
    row['current_before_assert'] = before
    row['current_after_assert'] = after
    row['current_unchanged'] = before is not None and before == after
    row['staged_builds'] = builds_of(name)
# No-prior-current case.
dst = work / 'noprior'
if dst.exists():
    shutil.rmtree(dst)
shutil.copytree(HERE / 'failing-assertion', dst)
row = run('noprior assert-without-build', ['assert', str(dst)])
row['current_after_assert'] = current_of('noprior')
row['no_current_created'] = current_of('noprior') is None
row['staged_builds'] = builds_of('noprior')
out.write_text(json.dumps(results, indent=2) + '\n')
print('wrote ' + str(out))
