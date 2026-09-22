"""Run isolated CLI/LSP probes with external process-group deadlines."""
import json, os, pathlib, signal, subprocess, time, sys, shutil
HERE=pathlib.Path(__file__).resolve().parent
launcher=pathlib.Path(sys.argv[1]).resolve()
work=pathlib.Path(sys.argv[2]).resolve()
work.mkdir(parents=True, exist_ok=True)
for source in (HERE/'projects').iterdir():
 if source.is_dir() and not (work/source.name).exists():
  shutil.copytree(source,work/source.name)
results=[]
def current(name):
 p=work/name/'dist/current.json'
 return json.loads(p.read_text()) if p.exists() else None
def run(label,args,timeout=15,input_bytes=None):
 start=time.monotonic()
 p=subprocess.Popen([str(launcher),*args],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,start_new_session=True)
 killed=False
 try: out,err=p.communicate(input_bytes,timeout=timeout)
 except subprocess.TimeoutExpired:
  killed=True;os.killpg(p.pid,signal.SIGKILL);out,err=p.communicate(timeout=3)
 row={'label':label,'argv':args,'external_limit_seconds':timeout,'externally_terminated':killed,'exit_code':p.returncode,'elapsed_seconds':round(time.monotonic()-start,3),'stdout':out.decode(errors='replace'),'stderr':err.decode(errors='replace')}
 results.append(row);(HERE/'cli-results.json').write_text(json.dumps(results,indent=2)+'\n')
 print(json.dumps({k:row[k] for k in ['label','exit_code','externally_terminated','elapsed_seconds','stderr']}),flush=True)
 return row
for name in ['passing','failing-assertion','pending-assertion','cpu-assertion','diagnostic-semantic','diagnostic-resolve','diagnostic-parse','resource-direct','resource-record','resource-array','resource-closure','resource-transaction-record','resource-transaction-array']:
 row=run(name+' build',['build',str(work/name)])
 row['current_after_build']=current(name)
for name in ['passing','failing-assertion','pending-assertion','cpu-assertion','resource-direct','resource-record','resource-array','resource-closure','resource-transaction-record','resource-transaction-array']:
 before=current(name)
 row=run(name+' assert',['assert',str(work/name)],timeout=4 if name in ['pending-assertion','cpu-assertion'] else 15)
 row['current_before_assert']=before;row['current_after_assert']=current(name)
# Batch a real LSP session per source file; EOF ends the server after didOpen.
def frame(obj):
 b=json.dumps(obj).encode();return b'Content-Length: '+str(len(b)).encode()+b'\r\n\r\n'+b
for name in ['passing','diagnostic-semantic','diagnostic-resolve','diagnostic-parse']:
 source=work/name/'src/main.can'
 msg=frame({'jsonrpc':'2.0','id':1,'method':'initialize','params':{'rootUri':(work/name).as_uri(),'capabilities':{}}})+frame({'jsonrpc':'2.0','method':'textDocument/didOpen','params':{'textDocument':{'uri':source.as_uri(),'languageId':'can','version':1,'text':source.read_text()}}})
 row=run(name+' lsp',['lsp','--stdio'],input_bytes=msg)
 raw=row['stdout'];messages=[]
 while raw:
  header,raw=raw.split('\r\n\r\n',1);n=int(header.split(':',1)[1]);messages.append(json.loads(raw[:n]));raw=raw[n:]
 row['messages']=messages
 row['source_lines']=source.read_text().splitlines()
(HERE/'cli-results.json').write_text(json.dumps(results,indent=2)+'\n')
