from pathlib import Path
import json, shutil, subprocess, os, re

repo=Path('/Users/vince/Projects/can-lang')
evidence=repo/'docs/syntax-taste/evidence/2026-09-24/clean-room-full-review'
evidence.mkdir(parents=True,exist_ok=True)
workspace=Path('/private/tmp/can-clean-review-probes')
workspace.mkdir(parents=True,exist_ok=True)
runner=repo/'compiler/.clean-review-probe'
runner.mkdir(exist_ok=False)
go=r'''package main
import (
 "encoding/json"
 "os"
 "path/filepath"
 "strings"
 "github.com/veighnsche/can-lang/compiler/internal/check"
 "github.com/veighnsche/can-lang/compiler/internal/project"
 "github.com/veighnsche/can-lang/compiler/internal/emit"
 "github.com/veighnsche/can-lang/compiler/internal/ir"
)
func main() {
 result:=map[string]any{"accepted":false}
 graph,err:=project.Load(os.Args[1]); if err!=nil {result["error"]=err.Error();json.NewEncoder(os.Stdout).Encode(result);return}
 p,err:=check.CheckProgram(graph);if err!=nil {result["error"]=err.Error();json.NewEncoder(os.Stdout).Encode(result);return}
 result["accepted"]=true
 patterns:=map[string][]string{}
 for _,f:=range p.Functions { if f.Region.Body.Terminal.Match!=nil {for _,a:=range f.Region.Body.Terminal.Match.Arms {for _,pat:=range a.Patterns {patterns[f.Symbol.Name]=append(patterns[f.Symbol.Name],pat.Kind)}}}}
 result["patterns"]=patterns
 var deps []ir.Artifact
 err=filepath.WalkDir(os.Args[2],func(path string,d os.DirEntry,err error)error{if err!=nil||d.IsDir()||!strings.HasSuffix(path,".ts"){return err};rel,err:=filepath.Rel(os.Args[2],path);if err!=nil{return err};deps=append(deps,ir.Artifact{Path:filepath.ToSlash(filepath.Join("runtime",rel))});return nil})
 if err!=nil {panic(err)}
 artifacts,err:=emit.AssertionModules(p,"runtime",deps);if err!=nil{result["emitError"]=err.Error();json.NewEncoder(os.Stdout).Encode(result);return}
 dir:=filepath.Join(os.Args[1],"generated");os.MkdirAll(dir,0700)
 for _,a:=range artifacts {if len(a.Bytes)==0{continue};path:=filepath.Join(dir,a.Path);os.MkdirAll(filepath.Dir(path),0700);if err:=os.WriteFile(path,a.Bytes,0600);err!=nil{panic(err)}}
 if err:=os.Symlink(os.Args[2],filepath.Join(dir,"runtime"));err!=nil&&!os.IsExist(err){panic(err)}
 result["entry"]=filepath.Join(dir,"entry.ts")
 json.NewEncoder(os.Stdout).Encode(result)
}
'''
(runner/'main.go').write_text(go)
(evidence/'probe-runner.go.txt').write_text(go)
header='package app\n    provides []\n    uses []\n'
main='fn void main\n    emits []\n    given\n        str[] args\n    asserts\n        empty: [] => ok\n    ok\n'
patterns='''record paid
record pending
record declined
variant payment
    paid
    pending
    declined
fn bool accepted
    emits []
    given
        payment value
    asserts
        sample: paid() => ok true
        rejected: declined() => ok false
    match value
        paid => ok true
        pending => ok false
        declined => ok false
'''
variants='''record unit
variant tagged<item>
    unit
variant bridge
    unit
fn tagged<str> convert
    emits []
    given
        tagged<int> value
    asserts
        sample: unit() => ok unit()
BODY
'''
helper='''package helper
    provides [read]
    uses [text]
fn str read
    emits []
    asserts
        unit: => ok "fixture"
    match call text::from_int(7)
        when
            unit: 7 => ok "fixture"
            customer: 7 => ok "fixture"
        ok str result => ok result
'''
consumer='''package app
    provides []
    uses [helper]
fn str read_customer
    emits []
    asserts
        customer: => ok "fixture"
    ok call helper::read()
'''+main
cases={
 'pattern-correct':{'src/main.can':header+patterns+main},
 'pattern-typo':{'src/main.can':header+patterns.replace('        declined =>','        decliend =>')+main},
 'pattern-typo-new-leaf':{'src/main.can':header+patterns.replace('variant payment','record refunded\nvariant payment').replace('    declined\n','    declined\n    refunded\n').replace('        declined =>','        decliend =>')+main},
 'pattern-correct-new-leaf':{'src/main.can':header+patterns.replace('variant payment','record refunded\nvariant payment').replace('    declined\n','    declined\n    refunded\n')+main},
 'pattern-constructor-typo':{'src/main.can':header+patterns.replace('        declined =>','        decliend() =>')+main},
 'variant-direct':{'src/main.can':header+variants.replace('BODY','    ok value')+main},
 'variant-via-bridge':{'src/main.can':header+variants.replace('BODY','    bridge alias = value\n    ok alias')+main},
 'variant-via-leaf':{'src/main.can':header+variants.replace('BODY','    match value\n        unit => ok value')+main},
 'fixture-original':{'src/helper/main.can':helper,'src/app/main.can':consumer},
 'fixture-renamed-root':{'src/helper/main.can':helper,'src/app/main.can':consumer.replace('        customer:', '        renamed:')},
}
results=[]
env=dict(os.environ,GOCACHE='/tmp/can-saas-review-go-cache')
try:
 build=subprocess.run(['go','build','-o',str(workspace/'probe'),str(runner)],cwd=repo,env=env,text=True,capture_output=True,check=True)
 for name,files in cases.items():
  directory=workspace/name;directory.mkdir(exist_ok=True)
  files={'can.project.json':'{"source_root":"src","error_registry":"can.errors.json"}','can.errors.json':'{"active":[],"retired":[]}',**files}
  for rel,body in files.items():
   p=directory/rel;p.parent.mkdir(parents=True,exist_ok=True);p.write_text(body)
   saved=evidence/'cases'/name/rel;saved.parent.mkdir(parents=True,exist_ok=True);saved.write_text(body)
  checked=subprocess.run([str(workspace/'probe'),str(directory),str(repo/'runtime')],cwd=repo,text=True,capture_output=True,check=True)
  result={'case':name,**json.loads(checked.stdout)}
  if result.get('entry'):
   generated=Path(result['entry']).parent
   diagnostics=generated/'diagnostics';diagnostics.mkdir(exist_ok=True)
   (diagnostics/'source-index.json').write_text(json.dumps({'schemaVersion':1,'kind':'can.source-index','sources':[],'modules':[]}))
   envfile=directory/'empty-environment.json';envfile.write_text('{}')
   count=len(re.findall(r'import \{ \$canCase as ',Path(result['entry']).read_text()))
   runs=[]
   for index in range(count):
    ran=subprocess.run(['/bin/sh','-c','exec bun "$1" "$2" 3< "$3"','probe',result['entry'],f'root={index}',str(envfile)],cwd=directory,text=True,capture_output=True,timeout=30)
    runs.append({'rootIndex':index,'exit':ran.returncode,'stdout':ran.stdout,'stderr':ran.stderr})
   result['runtimeRuns']=runs
  results.append(result)
  print(json.dumps({k:v for k,v in result.items() if k not in ('runtimeRuns','entry')}|{'runs':[{'exit':r['exit'],'report':json.loads(r['stdout']) if r['stdout'] else r['stderr']} for r in result.get('runtimeRuns',[])]}),flush=True)
 (evidence/'probe-results.json').write_text(json.dumps(results,indent=2)+'\n')
finally:
 shutil.rmtree(runner)
