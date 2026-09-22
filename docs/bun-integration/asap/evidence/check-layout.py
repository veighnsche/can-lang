#!/usr/bin/env python3
"""B1 changed-file size guard. Responsibility boundaries also require review."""
import argparse,pathlib,subprocess,sys
args=argparse.ArgumentParser(description=__doc__)
args.add_argument('--base',required=True,help='Implementation base Git commit/ref, after unrelated work is integrated')
opt=args.parse_args()
root=pathlib.Path(__file__).resolve().parents[4]
def git(*arguments):
 return subprocess.check_output(['git',*arguments],cwd=root)
base=git('rev-parse','--verify',opt.base+'^{commit}').decode().strip()
scopes=['compiler/internal','runtime','tests/integration']
changed=set(git('diff','--name-only','-z',base,'--',*scopes).decode().split('\0'))
changed.update(git('ls-files','--others','--exclude-standard','-z','--',*scopes).decode().split('\0'))
exempt={'compiler/internal/catalogue/generated.go','runtime/catalogue.ts'}
failures=[];checked=0
for name in sorted(changed):
 if not name or name in exempt:continue
 path=root/name
 if not path.is_file() or path.suffix not in {'.go','.ts'}:continue
 if 'testdata' in path.parts or 'fixtures' in path.parts:continue
 checked+=1
 lines=len(path.read_text().splitlines())
 test=name.endswith(('_test.go','.test.ts'))
 limit=600 if test else 400
 previous=subprocess.run(['git','show',f'{base}:{name}'],cwd=root,capture_output=True)
 old=len(previous.stdout.splitlines()) if previous.returncode==0 else None
 allowed=max(limit,old or 0)
 if lines>allowed:failures.append(f'{name}: {lines} lines; allowed {allowed} (base {old if old is not None else "new"})')
 print(f'{name}: {lines} lines; base {old if old is not None else "new"}; limit {allowed}')
for failure in failures:print('FAIL '+failure,file=sys.stderr)
print(f'Checked {checked} changed source files; {len(failures)} size violations. Cohesion and protected-hub rules still require review.')
sys.exit(bool(failures))
