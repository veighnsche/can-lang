from pathlib import Path
import json,os,shutil,subprocess
repo=Path('/Users/vince/Projects/can-lang');ev=repo/'docs/syntax-taste/evidence/2026-09-24/post-upgrade-review-961f921';work=Path('/private/tmp/can-upgrade-audit-probe');work.mkdir(exist_ok=True)
runner=repo/'compiler/.upgrade-review-audit';runner.mkdir(exist_ok=False)
go='''package main
import("encoding/json";"crypto/sha256";"encoding/hex";"os";"path/filepath";"strings";"github.com/veighnsche/can-lang/compiler/internal/project";"github.com/veighnsche/can-lang/compiler/internal/check";"github.com/veighnsche/can-lang/compiler/internal/emit";"github.com/veighnsche/can-lang/compiler/internal/browser";"github.com/veighnsche/can-lang/compiler/internal/ir")
func main(){r:=map[string]any{};defer func(){json.NewEncoder(os.Stdout).Encode(r)}();g,e:=project.Load(os.Args[1]);if e!=nil{r["loadError"]=e.Error();return};p,e:=check.CheckProgram(g);if e!=nil{r["checkError"]=e.Error();return};r["checked"]=true;var deps []ir.Artifact;filepath.WalkDir(os.Args[2],func(path string,d os.DirEntry,e error)error{if e!=nil{return e};if !d.IsDir()&&strings.HasSuffix(path,".ts"){rel,_:=filepath.Rel(os.Args[2],path);deps=append(deps,ir.Artifact{Path:"runtime/"+filepath.ToSlash(rel),Runtime:true})};return nil});a,e:=emit.BrowserModules(p,"runtime",deps);if e!=nil{r["emitError"]=e.Error();return};r["emitted"]=true;files:=map[string]string{};for _,v:=range a{if v.Runtime||!strings.HasSuffix(v.Path,".ts"){continue};sum:=sha256.Sum256(v.Bytes);files[v.Path]=hex.EncodeToString(sum[:])};b,e:=browser.AssetBytes(files);if e!=nil{r["assetError"]=e.Error();return};a=append(a,ir.Artifact{Path:browser.AssetPath,Bytes:b});if e=browser.AuditArtifacts(a);e!=nil{r["auditError"]=e.Error()}else{r["auditAccepted"]=true};for _,v:=range a{if v.Runtime{continue};path:=filepath.Join(os.Args[1],"generated",v.Path);os.MkdirAll(filepath.Dir(path),0700);os.WriteFile(path,v.Bytes,0600)}}
'''
(runner/'main.go').write_text(go);(ev/'browser-audit-runner.go.txt').write_text(go)
base='''package app
    provides []
    uses []
fn str description
    emits []
    asserts
        sample: => ok EXPECT
    ok EXPR
fn void main
    emits []
    given
        str[] args
    asserts
        empty: [] => ok
    ok
'''
cases={'ordinary':('"hello"','"hello"'),'literal-host-text':('"Bun."','"Bun."'),'equivalent-concatenation':('"Bun."','"Bu" + "n."'),'literal-node-text':('"node: introduction"','"node: introduction"')}
results=[]
try:
 subprocess.run(['go','build','-o',str(work/'runner'),str(runner)],cwd=repo,env=dict(os.environ,GOCACHE='/tmp/can-upgrade-review-go-cache'),check=True)
 for name,(expected,expr) in cases.items():
  root=work/name;(root/'src').mkdir(parents=True,exist_ok=True)
  saved=ev/'browser-audit-cases'/name;(saved/'src').mkdir(parents=True,exist_ok=True)
  for rel,body in {'can.project.json':'{"source_root":"src","error_registry":"can.errors.json"}','can.errors.json':'{"active":[],"retired":[]}','src/main.can':base.replace('EXPECT',expected).replace('EXPR',expr)}.items():
   (root/rel).write_text(body);(saved/rel).write_text(body)
  p=subprocess.run([str(work/'runner'),str(root),str(repo/'runtime')],text=True,capture_output=True,check=True)
  result={'case':name,**json.loads(p.stdout)};results.append(result);print(json.dumps(result),flush=True)
 (ev/'browser-audit-results.json').write_text(json.dumps(results,indent=2)+'\n')
finally:shutil.rmtree(runner)
