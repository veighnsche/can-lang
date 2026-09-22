#!/usr/bin/env python3
"""Explicit pinned-Bun native probes; no production source edits or downloads."""
import argparse,datetime,hashlib,json,pathlib,subprocess,tempfile
p=pathlib.Path(__file__).resolve().parent
root=p.parents[3]
args=argparse.ArgumentParser();args.add_argument('--bun',required=True,type=pathlib.Path);opt=args.parse_args()
bun=opt.bun.resolve();target=json.loads((root/'distribution/target.json').read_text())
digest=lambda path:hashlib.sha256(path.read_bytes()).hexdigest()
actual=digest(bun)
if actual!=target['runtime']['sha256']:raise SystemExit('Runtime SHA mismatch; refusing unqualified binary')
report={'observedUTC':datetime.datetime.now(datetime.timezone.utc).isoformat(),'bunSHA256':actual,'target':target['targetId'],'runs':{}}
with tempfile.TemporaryDirectory(prefix='can-asap-native-') as cwd:
 for script,stem,seconds in [('native-probe.ts','native-probe',30),('sqlite-lifecycle-probe.ts','sqlite-lifecycle',20)]:
  result=subprocess.run([str(bun),'--no-install',str(p/script)],cwd=cwd,env={'PATH':'/usr/bin:/bin','HOME':cwd},capture_output=True,text=True,timeout=seconds)
  (p/(stem+'-results.json')).write_text(result.stdout);(p/(stem+'-stderr.txt')).write_text(result.stderr)
  if result.returncode:raise SystemExit(f'{script} exited {result.returncode}: inspect saved output')
  data=json.loads(result.stdout)
  if data['version']!=target['runtime']['version'] or data['revision']!=target['runtime']['revision']:raise SystemExit('Runtime identity mismatch')
  report['runs'][script]={'exitCode':result.returncode,'timeoutSeconds':seconds,'scriptSHA256':digest(p/script),'resultSHA256':digest(p/(stem+'-results.json'))}
(p/'probe-run-metadata.json').write_text(json.dumps(report,indent=2)+'\n')
print(json.dumps(report,indent=2))
