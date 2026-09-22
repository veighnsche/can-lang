"""LF04 evidence: supervised per-root assertion execution.

Cases on fresh copies of frozen gap projects (originals untouched):
 1. passing assert -> exit 0, suite report, empty stderr.
 2. failing-assertion assert -> exit 1, outcome mismatch, empty stderr.
 3. pending-assertion assert (default 5000ms) -> exit 1 with BOUNDED timeout
    report (no external kill needed; external limit 30s guards the harness).
 4. cpu-assertion assert -> exit 1 on its own (negative control, not a hang).
 5. passing assert --assert-timeout-ms 1 -> bounded timeout/late outcomes.
 6. selector run -> scope partial.
 7. invalid budgets (0, -5, abc, 600001, unset form) -> exit 2 usage.
 8. boundary --assert-timeout-ms 600000 accepted (flag validation only).
Writes results ONLY to the scratch out path.
"""
import json, os, pathlib, shutil, signal, subprocess, sys, time
HERE = pathlib.Path('/Users/vince/Projects/can-lang/docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects')
launcher = pathlib.Path(sys.argv[1]).resolve()
work = pathlib.Path(sys.argv[2]).resolve()
out = pathlib.Path(sys.argv[3]).resolve()
work.mkdir(parents=True, exist_ok=True)
results = []
def run(label, args, timeout=30):
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
    row = {'label': label, 'argv': args, 'exit_code': p.returncode,
           'externally_terminated': killed,
           'elapsed_seconds': round(time.monotonic() - start, 3),
           'stdout': out_b.decode(errors='replace')[-3000:],
           'stderr': err.decode(errors='replace')[-1500:]}
    results.append(row)
    print(json.dumps({k: row[k] for k in ['label', 'exit_code', 'externally_terminated', 'elapsed_seconds']}), flush=True)
    return row
for name in ['passing', 'failing-assertion', 'pending-assertion', 'cpu-assertion']:
    dst = work / name
    if dst.exists():
        shutil.rmtree(dst)
    shutil.copytree(HERE / name, dst)
run('passing assert', ['assert', str(work / 'passing')])
run('failing assert', ['assert', str(work / 'failing-assertion')])
run('pending assert default budget', ['assert', str(work / 'pending-assertion')], timeout=30)
run('cpu assert', ['assert', str(work / 'cpu-assertion')])
run('passing assert tiny budget', ['assert', '--assert-timeout-ms', '1', str(work / 'passing')])
run('selector partial', ['assert', str(work / 'passing'), 'can.project.root/app', 'can.project.root/app::answer', 'exact'])
for bad in ['0', '-5', 'abc', '600001', '1.5', 'unlimited']:
    run('invalid budget ' + bad, ['assert', '--assert-timeout-ms', bad, str(work / 'passing')])
run('missing budget value', ['assert', '--assert-timeout-ms'])
run('boundary max budget accepted', ['assert', '--assert-timeout-ms', '600000', str(work / 'passing')], timeout=60)
out.write_text(json.dumps(results, indent=2) + '\n')
print('wrote ' + str(out))
