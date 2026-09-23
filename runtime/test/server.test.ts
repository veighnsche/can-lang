import {test,expect} from "bun:test";
import {spawn} from "node:child_process";
import {mkdtempSync,writeFileSync,readFileSync,rmSync} from "node:fs";
import {tmpdir} from "node:os";
import {join} from "node:path";
import {fileURLToPath} from "node:url";
import {createHash} from "node:crypto";
import {catalogue} from "../catalogue.ts";
import {createDomainRuntime,domainFailureDiagnostics,type FailureShape} from "../domain.ts";
import {standardFailureDiagnostics} from "../failure.ts";
import {success,value,invoke,type Completion} from "../completion.ts";
import {runOwnedRoot,launchOwned,resourceStatus} from "../owner.ts";
import {assertionContext} from "../assert/context.ts";
import {ownBytes} from "../bytes.ts";
import {createServer,isServerValue} from "../platform/server.ts";
import {createRouter} from "../platform/router.ts";
import {createRequests,createResponses} from "../platform/http.ts";
import {createStreamReads} from "../transport/stream/readable.ts";
import {copyBytes} from "../bytes.ts";

const origin={source:"can:test",start:0,end:0,invocation:[]};
const identity=(kind:string,declaration:string)=>createHash("sha256").update("can-concrete-type-v1\0"+JSON.stringify([kind,declaration])).digest("hex");
const textShape:FailureShape={identity:identity("primitive","str"),kind:"primitive",declaration:"str",arguments:[],fields:[],leaves:[],inputs:[],errors:[]};
const fieldNames:Record<string,string[]>={"http::invalid_server_config":["reason"],"http::bind_failed":["address"],"http::shutdown_failed":["phase"],"http::invalid_route":["reason"],"http::duplicate_route":["method","path"],"http::ambiguous_route":["first","second"],"http::invalid_request":["reason"],"http::body_limit":["limit"],"codec::invalid_data":["path","reason"],"stream::read_failed":["reason"],"stream::write_failed":["reason"],"stream::cancelled":["reason"],"stream::close_failed":["reason"],"files::limit_exceeded":["limit"]};
const declarations=catalogue.errors.filter(e=>fieldNames[e.name]!==undefined).map(e=>({identity:e.identity,name:e.name,id:e.id,parameters:0}));
const intShape:FailureShape={identity:identity("primitive","int"),kind:"primitive",declaration:"int",arguments:[],fields:[],leaves:[],inputs:[],errors:[]};
const fieldKinds=new Map(catalogue.errors.flatMap(e=>e.fields.map(f=>[e.name+":"+f.name,f.type])));
const errorShapes:FailureShape[]=declarations.map(e=>({identity:identity("error",e.identity),kind:"error",declaration:e.identity,arguments:[],fields:fieldNames[e.name]!.map(name=>({name,type:fieldKinds.get(e.name+":"+name)==="int"?intShape.identity:textShape.identity})),leaves:[],inputs:[],errors:[]}));
const domain=createDomainRuntime({declarations,shapes:[textShape,intShape,...errorShapes]});
const id=(declaration:string)=>identity("error",declaration);
const server=createServer(domain,{invalidConfig:id("can.std.http@1::invalid_server_config"),bindFailed:id("can.std.http@1::bind_failed"),shutdownFailed:id("can.std.http@1::shutdown_failed")});
const router=createRouter(domain,{invalid:id("can.std.http@1::invalid_route"),duplicate:id("can.std.http@1::duplicate_route"),ambiguous:id("can.std.http@1::ambiguous_route")});
const responses=createResponses(domain,{invalid:id("can.std.http@1::invalid_request"),invalidData:id("can.std.codec@1::invalid_data"),close:id("can.std.stream@1::close_failed"),writeFailed:id("can.std.stream@1::write_failed"),limit:id("can.std.http@1::body_limit")});
const requests=createRequests(domain,{invalid:id("can.std.http@1::invalid_request"),limit:id("can.std.http@1::body_limit"),invalidData:id("can.std.codec@1::invalid_data"),header:"header",close:id("can.std.stream@1::close_failed"),writeFailed:id("can.std.stream@1::write_failed"),multipartForm:"unused",multipartField:"unused",multipartFile:"unused"});
const reads=createStreamReads(domain,{readFailed:id("can.std.stream@1::read_failed"),cancelled:id("can.std.stream@1::cancelled"),closeFailed:id("can.std.stream@1::close_failed"),limitExceeded:id("can.std.files@1::limit_exceeded")});
const textOf=(item:unknown)=>new TextDecoder().decode(copyBytes(item,origin));

function domainOutcome(completion:Completion<unknown>,name:string):Record<string,string>{
 expect(completion.kind).toBe("domain");if(completion.kind!=="domain")throw new Error("wrong outcome");
 const details=domainFailureDiagnostics(completion.value);
 expect(details.declaration.name).toBe(name);
 return details.payload as Record<string,string>;
}
async function standardKind(call:()=>Promise<Completion<unknown>>):Promise<string>{
 const completion=await invoke(call,origin);
 expect(completion.kind).toBe("standard");if(completion.kind!=="standard")throw new Error("wrong outcome");
 return standardFailureDiagnostics(completion.value).kind;
}
async function textResult(body:string):Promise<Completion<unknown>>{
 return responses.text(value(await responses.ok()),value(await responses.emptyHeaders()),body);
}
async function startOn(port:number,handler:(request?:unknown)=>Promise<Completion<unknown>>,options?:{shutdownMs?:bigint;bodyLimit?:bigint}):Promise<{token:unknown;config:unknown}>{
 const config=value(await server.makeConfig("127.0.0.1",BigInt(port),options?.bodyLimit??1048576n,options?.shutdownMs??5000n));
 const get=value(await router.get("/x",handler));
 const post=value(await router.post("/x",handler));
 const table=value(await router.make([get,post]));
 const started=await server.start(config,table);
 if(started.kind!=="ok")throw new Error("test server failed to start");
 return {token:value(started),config};
}
async function withoutUnhandled<T>(run:()=>Promise<T>):Promise<T>{
 const seen:unknown[]=[];
 const listener=(reason:unknown)=>{seen.push(reason);};
 process.on("unhandledRejection",listener);
 try{const out=await run();await Bun.sleep(20);expect(seen).toEqual([]);return out;}
 finally{process.off("unhandledRejection",listener);}
}

test("config validation rejects out-of-range numbers with field reasons",async ()=>{
 expect(domainOutcome(await server.makeConfig("127.0.0.1",8080n,0n,1000n),"http::invalid_server_config")).toMatchObject({reason:"body_limit"});
 expect(domainOutcome(await server.makeConfig("127.0.0.1",8080n,67108865n,1000n),"http::invalid_server_config")).toMatchObject({reason:"body_limit"});
 expect(domainOutcome(await server.makeConfig("127.0.0.1",8080n,1024n,-1n),"http::invalid_server_config")).toMatchObject({reason:"shutdown_ms"});
 expect(domainOutcome(await server.makeConfig("127.0.0.1",8080n,1024n,2147483648n),"http::invalid_server_config")).toMatchObject({reason:"shutdown_ms"});
 for(const [limit,shutdown] of [[1n,0n],[67108864n,2147483647n],[1024n,5000n]] as const){
  const made=await server.makeConfig("127.0.0.1",0n,limit,shutdown);
  expect(made.kind).toBe("ok");
  expect(Object.isFrozen(value(made))).toBe(true);
  expect(isServerValue("server_config",value(made))).toBe(true);
  expect(isServerValue("server",value(made))).toBe(false);
 }
});

test("config leaves address authority to Bun",async ()=>{
 for(const [host,port] of [["",8080n],["127.0.0.1",70000n],["127.0.0.1",-1n],["a\0b",8080n],["127.0.0.1",2n**100n]] as const){
  expect((await server.makeConfig(host,port,1024n,1000n)).kind).toBe("ok");
 }
});

test("ephemeral ports bind distinct servers",async ()=>{
 const owned=await runOwnedRoot(async ()=>{
  const first=await startOn(0,async ()=>textResult("a"));
  const second=await startOn(0,async ()=>textResult("b"));
  expect(isServerValue("server",first.token)).toBe(true);
  expect(first.token).not.toBe(second.token);
  expect((await server.stop(first.token)).kind).toBe("ok");
  expect((await server.stop(second.token)).kind).toBe("ok");
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
});

test("bind failures echo the attempted address",async ()=>{
 const owned=await runOwnedRoot(async ()=>{
  const first=await startOn(18341,async ()=>textResult("one"));
  const config=value(await server.makeConfig("127.0.0.1",18341n,1024n,1000n));
  const route=value(await router.get("/x",async ()=>textResult("two")));
  const table=value(await router.make([route]));
  expect(domainOutcome(await server.start(config,table),"http::bind_failed")).toMatchObject({address:"127.0.0.1:18341"});
  const badPort=value(await server.makeConfig("127.0.0.1",70000n,1024n,1000n));
  expect(domainOutcome(await server.start(badPort,table),"http::bind_failed")).toMatchObject({address:"127.0.0.1:70000"});
  const huge=value(await server.makeConfig("127.0.0.1",2n**100n,1024n,1000n));
  expect(domainOutcome(await server.start(huge,table),"http::bind_failed")).toMatchObject({address:"127.0.0.1:"+(2n**100n).toString()});
  const badHost=value(await server.makeConfig("a\0b",18342n,1024n,1000n));
  expect(domainOutcome(await server.start(badHost,table),"http::bind_failed")).toMatchObject({address:"a\0b:18342"});
  expect((await server.stop(first.token)).kind).toBe("ok");
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
});

test("start rejects foreign routers without binding",async ()=>{
 const owned=await runOwnedRoot(async ()=>{
  const config=value(await server.makeConfig("127.0.0.1",18343n,1024n,1000n));
  expect(await standardKind(()=>server.start(config,{}))).toBe("resource_state");
  expect(await standardKind(()=>server.start(config,config))).toBe("resource_state");
  const route=value(await router.get("/x",async ()=>textResult("recovered")));
  const table=value(await router.make([route]));
  const token=value(await server.start(config,table));
  const response=await fetch("http://127.0.0.1:18343/x");
  expect(response.status).toBe(200);expect(await response.text()).toBe("recovered");
  expect(response.headers.get("content-security-policy")).toContain("script-src 'self'");
  expect(response.headers.get("content-security-policy")).not.toContain("unsafe-eval");
  expect((await server.stop(token)).kind).toBe("ok");
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
});

test("serves routed requests with keep-alive reuse",async ()=>{
 const owned=await runOwnedRoot(async ()=>{
  const config=value(await server.makeConfig("127.0.0.1",18344n,1048576n,5000n));
  const get=value(await router.get("/x",async ()=>textResult("got")));
  const post=value(await router.post("/x",async ()=>textResult("posted")));
  const table=value(await router.make([get,post]));
  const token=value(await server.start(config,table));
  for(let round=0;round<3;round++){
   const got=await fetch("http://127.0.0.1:18344/x");
   expect(got.status).toBe(200);expect(await got.text()).toBe("got");
   const posted=await fetch("http://127.0.0.1:18344/x",{method:"POST",body:"payload"});
   expect(posted.status).toBe(200);expect(await posted.text()).toBe("posted");
  }
  const missing=await fetch("http://127.0.0.1:18344/absent");
  expect(missing.status).toBe(404);expect(await missing.text()).toBe("Not Found");
  expect(missing.headers.get("x-content-type-options")).toBe("nosniff");
  const wrong=await fetch("http://127.0.0.1:18344/x",{method:"PUT"});
  expect(wrong.status).toBe(405);expect(wrong.headers.get("allow")).toBe("GET, POST");
  expect(resourceStatus(token)).toMatchObject({kind:"server",state:"open",leases:0});
  expect((await server.stop(token)).kind).toBe("ok");
  expect(resourceStatus(token)).toMatchObject({kind:"server",state:"closed",leases:0});
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
});

test("rejects over-limit bodies with 413",async ()=>{
 const owned=await runOwnedRoot(async ()=>{
  const {token}=await startOn(18345,async ()=>textResult("small"),{bodyLimit:8n});
  const tooLarge=await fetch("http://127.0.0.1:18345/x",{method:"POST",body:"nine-bytes"});
  expect(tooLarge.status).toBe(413);expect(await tooLarge.text()).toBe("Payload Too Large");
  expect(tooLarge.headers.get("x-content-type-options")).toBe("nosniff");
  const exact=await fetch("http://127.0.0.1:18345/x",{method:"POST",body:"eight123"});
  expect(exact.status).toBe(200);expect(await exact.text()).toBe("small");
  expect(resourceStatus(token)).toMatchObject({leases:0});
  expect((await server.stop(token)).kind).toBe("ok");
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
});

test("dispatch failures become sanitized 500s and release leases",async ()=>{
 const secret="plum-falcon-credentials-9f2";
 const owned=await runOwnedRoot(async ()=>{
  const config=value(await server.makeConfig("127.0.0.1",18346n,1048576n,5000n));
  const thrown=value(await router.get("/boom",async ()=>{throw new Error(secret);}));
  const failed=value(await router.get("/fail",async ()=>responses.makeBodyStatus(204n)));
  const table=value(await router.make([thrown,failed]));
  const token=value(await server.start(config,table));
  const first=await fetch("http://127.0.0.1:18346/boom");
  expect(first.status).toBe(500);
  const body=await first.text();
  expect(body).toBe("Internal Server Error");expect(body).not.toContain("plum-falcon");
  expect(first.headers.get("content-type")).toBe("text/plain; charset=utf-8");
  expect(first.headers.get("x-content-type-options")).toBe("nosniff");
  const second=await fetch("http://127.0.0.1:18346/fail");
  expect(second.status).toBe(500);expect(await second.text()).toBe("Internal Server Error");
  expect(resourceStatus(token)).toMatchObject({state:"open",leases:0});
  expect((await server.stop(token)).kind).toBe("ok");
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
});

test("stop drains delayed handlers before settling",async ()=>{
 const owned=await runOwnedRoot(async ()=>{
  const {token}=await startOn(18347,async ()=>{await Bun.sleep(150);return textResult("slow");});
  const pending=fetch("http://127.0.0.1:18347/x");
  await Bun.sleep(30);
  expect(resourceStatus(token)).toMatchObject({state:"open"});
  expect(resourceStatus(token).leases).toBeGreaterThanOrEqual(1);
  const begin=Date.now();
  expect((await server.stop(token)).kind).toBe("ok");
  expect(Date.now()-begin).toBeGreaterThanOrEqual(100);
  const response=await pending;
  expect(response.status).toBe(200);expect(await response.text()).toBe("slow");
  expect(resourceStatus(token)).toMatchObject({state:"closed",leases:0});
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
});

test("timed-out caller keeps work owned while wait observes settlement",async ()=>{
 await withoutUnhandled(async ()=>{
  // The expired caller bound reports a cleanup failure while the close
  // continues; the later drain still succeeds and the waiter observes ok.
  const diagnostics:unknown[]=[];
  const owned=await runOwnedRoot(async ()=>{
   let entered!:()=>void;const gate=new Promise<void>(resolve=>{entered=resolve;});
   let release!:()=>void;const held=new Promise<void>(resolve=>{release=resolve;});
   const {token}=await startOn(18348,async ()=>{entered();await held;return textResult("released");},{shutdownMs:30n});
   const pending=fetch("http://127.0.0.1:18348/x");
   await gate;
   const stopped=await server.stop(token);
   expect(domainOutcome(stopped,"http::shutdown_failed")).toMatchObject({phase:"deadline"});
   expect(resourceStatus(token)).toMatchObject({state:"closing"});
   const waiting=server.wait(token);
   const early=await Promise.race([waiting.then(()=> "settled"),Bun.sleep(100).then(()=> "pending")]);
   expect(early).toBe("pending");
   release();
   const response=await pending;
   expect(response.status).toBe(200);expect(await response.text()).toBe("released");
   expect((await waiting).kind).toBe("ok");
   expect(resourceStatus(token)).toMatchObject({state:"closed",leases:0});
   return success(undefined);
  },diagnostic=>{diagnostics.push(diagnostic);});
  expect(owned.cleanupFailed).toBe(true);expect(owned.completion.kind).toBe("ok");
  expect(diagnostics.length).toBe(1);expect(diagnostics[0]).toMatchObject({phase:"cleanup"});
 });
});

test("simultaneous stop callers split ok and resource_state",async ()=>{
 const owned=await runOwnedRoot(async ()=>{
  const {token}=await startOn(18349,async ()=>textResult("x"));
  const [first,second]=await Promise.all([invoke(()=>server.stop(token),origin),invoke(()=>server.stop(token),origin)]);
  const kinds=[first.kind,second.kind].sort();
  expect(kinds).toEqual(["ok","standard"]);
  const failed=first.kind==="standard"?first:second;
  if(failed.kind!=="standard")throw new Error("wrong outcome");
  expect(standardFailureDiagnostics(failed.value).kind).toBe("resource_state");
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
});

test("repeated and stale stops report resource_state while wait observes",async ()=>{
 const owned=await runOwnedRoot(async ()=>{
  const {token,config}=await startOn(18350,async ()=>textResult("x"));
  expect((await server.stop(token)).kind).toBe("ok");
  expect(await standardKind(()=>server.stop(token))).toBe("resource_state");
  expect(await standardKind(()=>server.stop({}))).toBe("resource_state");
  expect(await standardKind(()=>server.stop(config))).toBe("resource_state");
  expect(await standardKind(()=>server.wait({}))).toBe("resource_state");
  expect((await server.wait(token)).kind).toBe("ok");
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
});

test("early-winning coordination keeps a late server waiter",async ()=>{
 await withoutUnhandled(async ()=>{
  const owned=await runOwnedRoot(async ()=>{
   const {token}=await startOn(18351,async ()=>textResult("x"));
   // server_wait only observes settlement, so the waiter retains no lease
   // that could deadlock the close it is waiting for.
   const group=launchOwned([
    {captures:[],run:async ()=>{await Bun.sleep(10);return success("fast");}},
    {captures:[],run:()=>server.wait(token)}
   ]);
   expect((await group.promises[0]).kind).toBe("ok");
   group.publish([0]);
   expect((await server.stop(token)).kind).toBe("ok");
   const late=await group.promises[1];
   expect(late.kind).toBe("ok");if(late.kind!=="ok")throw new Error("wrong outcome");
   expect(value(late)).toBe(undefined);
   return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
 });
});

test("wait installs no lasting signal listeners",async ()=>{
 const owned=await runOwnedRoot(async ()=>{
  const before=[process.listenerCount("SIGINT"),process.listenerCount("SIGTERM")];
  const {token}=await startOn(18352,async ()=>textResult("x"));
  const waiting=server.wait(token);
  expect(process.listenerCount("SIGINT")).toBe(before[0]+1);
  expect(process.listenerCount("SIGTERM")).toBe(before[1]+1);
  expect((await server.stop(token)).kind).toBe("ok");
  expect((await waiting).kind).toBe("ok");
  expect([process.listenerCount("SIGINT"),process.listenerCount("SIGTERM")]).toEqual(before);
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
});

test("assertions deny live boundaries but run config validation",async ()=>{
 const context=assertionContext({package:"can.test",declaration:"server",name:"boundary"});
 const made=value(await server.makeConfig("127.0.0.1",18353n,1024n,1000n,context));
 expect(isServerValue("server_config",made)).toBe(true);
 expect(await standardKind(()=>server.start(made,{},context))).toBe("assertion");
 expect(await standardKind(()=>server.stop({},context))).toBe("assertion");
 expect(await standardKind(()=>server.wait({},context))).toBe("assertion");
 const owned=await runOwnedRoot(async ()=>{
  const route=value(await router.get("/x",async ()=>textResult("live")));
  const table=value(await router.make([route]));
  const token=value(await server.start(made,table));
  const response=await fetch("http://127.0.0.1:18353/x");
  expect(response.status).toBe(200);
  expect((await server.stop(token)).kind).toBe("ok");
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
});

test("unclosed idle servers auto-close at drain without failure",async ()=>{
 const owned=await runOwnedRoot(async ()=>{
  await startOn(18354,async ()=>textResult("x"));
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
});

const childRuntime=fileURLToPath(new URL("../",import.meta.url));
function childPrelude(errors:readonly string[]):string{
 return `import {createHash} from "node:crypto";
import {catalogue} from ${JSON.stringify(join(childRuntime,"catalogue.ts"))};
import {createDomainRuntime,domainFailureDiagnostics} from ${JSON.stringify(join(childRuntime,"domain.ts"))};
import {success,value} from ${JSON.stringify(join(childRuntime,"completion.ts"))};
import {runOwnedRoot} from ${JSON.stringify(join(childRuntime,"owner.ts"))};
import {createServer} from ${JSON.stringify(join(childRuntime,"platform/server.ts"))};
import {createRouter} from ${JSON.stringify(join(childRuntime,"platform/router.ts"))};
import {createResponses} from ${JSON.stringify(join(childRuntime,"platform/http.ts"))};
const identity=(kind,declaration)=>createHash("sha256").update("can-concrete-type-v1\\0"+JSON.stringify([kind,declaration])).digest("hex");
const textShape={identity:identity("primitive","str"),kind:"primitive",declaration:"str",arguments:[],fields:[],leaves:[],inputs:[],errors:[]};
const wanted=new Set(${JSON.stringify(errors)});
const declarations=catalogue.errors.filter(e=>wanted.has(e.name)).map(e=>({identity:e.identity,name:e.name,id:e.id,parameters:0}));
const lookup=Object.fromEntries(catalogue.errors.filter(e=>wanted.has(e.name)).map(e=>[e.name,e.fields.map(f=>f.name)]));
const shapes=declarations.map(e=>({identity:identity("error",e.identity),kind:"error",declaration:e.identity,arguments:[],fields:lookup[e.name].map(name=>({name,type:textShape.identity})),leaves:[],inputs:[],errors:[]}));
const domain=createDomainRuntime({declarations,shapes:[textShape,...shapes]});
const id=declaration=>identity("error",declaration);
const server=createServer(domain,{invalidConfig:id("can.std.http@1::invalid_server_config"),bindFailed:id("can.std.http@1::bind_failed"),shutdownFailed:id("can.std.http@1::shutdown_failed")});
const router=createRouter(domain,{invalid:id("can.std.http@1::invalid_route"),duplicate:id("can.std.http@1::duplicate_route"),ambiguous:id("can.std.http@1::ambiguous_route")});
const responses=createResponses(domain,{invalid:id("can.std.http@1::invalid_request"),invalidData:id("can.std.codec@1::invalid_data")});
`;
}
async function runChild(name:string,script:string,interact:(child:ReturnType<typeof spawn>,output:()=>string)=>Promise<void>):Promise<{output:string;errors:string;code:number|null;signal:NodeJS.Signals|null}>{
 const directory=mkdtempSync(join(tmpdir(),"can-server-"+name+"-"));
 const file=join(directory,name+".ts");
 writeFileSync(file,script);
 const child=spawn(process.execPath,["--no-install",file],{stdio:["ignore","pipe","pipe"]});
 let output="",errors="";
 child.stdout.on("data",chunk=>{output+=chunk;});
 child.stderr.on("data",chunk=>{errors+=chunk;});
 const deadline=setTimeout(()=>child.kill("SIGKILL"),10000);
 try{
  await interact(child,()=>output);
  const status=await new Promise<{code:number|null;signal:NodeJS.Signals|null}>((resolve,reject)=>{child.once("error",reject);child.once("exit",(code,signal)=>resolve({code,signal}));});
  return {output,errors,...status};
 }finally{clearTimeout(deadline);child.kill("SIGKILL");rmSync(directory,{recursive:true,force:true});}
}
async function waitFor(output:()=>string,marker:string):Promise<void>{
 const begin=Date.now();
 while(!output().includes(marker)){if(Date.now()-begin>8000)throw new Error("missing child marker "+marker);await Bun.sleep(20);}
}

for(const [signal,port] of [["SIGINT",18355],["SIGTERM",18356]] as const)test(`signals settle wait gracefully in a child (${signal})`,async ()=>{
 const script=childPrelude(["http::invalid_server_config","http::bind_failed","http::shutdown_failed","http::invalid_route","http::duplicate_route","http::ambiguous_route","http::invalid_request","codec::invalid_data"])+`
await runOwnedRoot(async ()=>{
 const config=value(await server.makeConfig("127.0.0.1",${port}n,1048576n,5000n));
 const route=value(await router.get("/ping",async ()=>responses.text(value(await responses.ok()),value(await responses.emptyHeaders()),"pong")));
 const table=value(await router.make([route]));
 const token=value(await server.start(config,table));
 console.log("listening");
 const waited=await server.wait(token);
 console.log("settled:"+waited.kind);
 return success(undefined);
});`;
 const result=await runChild("signal",script,async (child,output)=>{
  await waitFor(output,"listening");
  const response=await fetch("http://127.0.0.1:"+String(port)+"/ping");
  expect(response.status).toBe(200);expect(await response.text()).toBe("pong");
  child.kill(signal);
 });
 expect(result.output).toContain("listening");expect(result.output).toContain("settled:ok");
 expect(result.errors).toBe("");expect(result.signal).toBe(null);expect(result.code).toBe(0);
},15000);

test("external supervisor bounds nonsettling work",async ()=>{
 const script=childPrelude(["http::invalid_server_config","http::bind_failed","http::shutdown_failed","http::invalid_route","http::duplicate_route","http::ambiguous_route","http::invalid_request","codec::invalid_data"])+`
await runOwnedRoot(async ()=>{
 let entered;const gate=new Promise(resolve=>{entered=resolve;});
 const never=new Promise(()=>{});
 const config=value(await server.makeConfig("127.0.0.1",18357n,1048576n,50n));
 const route=value(await router.get("/x",async ()=>{entered();await never;return responses.text(value(await responses.ok()),value(await responses.emptyHeaders()),"late");}));
 const table=value(await router.make([route]));
 const token=value(await server.start(config,table));
 console.log("listening");
 const pending=fetch("http://127.0.0.1:18357/x");
 await gate;
 const stopped=await server.stop(token);
 console.log("stop:"+stopped.kind+":"+JSON.stringify(domainFailureDiagnostics(stopped.value).payload));
 const raced=await Promise.race([server.wait(token).then(()=>"settled"),Bun.sleep(200).then(()=> "wait-pending")]);
 console.log(raced);
 await pending.catch(()=>{});
 await server.wait(token);
 console.log("settled-late");
 return success(undefined);
});`;
 const result=await runChild("hung",script,async (child,output)=>{
  await waitFor(output,"wait-pending");
  await Bun.sleep(100);
  child.kill("SIGKILL");
 });
 expect(result.output).toContain("listening");
 expect(result.output).toContain("stop:domain");expect(result.output).toContain("deadline");
 expect(result.output).toContain("wait-pending");expect(result.output).not.toContain("settled-late");
 expect(result.errors).toBe("");expect(result.signal).toBe("SIGKILL");
},15000);

const tlsCert="-----BEGIN CERTIFICATE-----\nAA==\n-----END CERTIFICATE-----\n";
const tlsKey="-----BEGIN PRIVATE KEY-----\nAA==\n-----END PRIVATE KEY-----\n";
const tlsBytes=(text:string)=>ownBytes(new TextEncoder().encode(text));

test("TLS config validates PEM structure before serving",async ()=>{
 const made=value(await server.makeTlsConfig(tlsBytes(tlsCert),tlsBytes(tlsKey)));
 expect(isServerValue("tls_config",made)).toBe(true);
 expect(isServerValue("server_config",made)).toBe(false);
 expect((await server.makeTlsConfig(tlsBytes(tlsCert+tlsCert),tlsBytes(tlsKey))).kind).toBe("ok");
 for(const key of ["-----BEGIN RSA PRIVATE KEY-----\nAA==\n-----END RSA PRIVATE KEY-----\n","-----BEGIN EC PRIVATE KEY-----\nAA==\n-----END EC PRIVATE KEY-----\n"])expect((await server.makeTlsConfig(tlsBytes(tlsCert),tlsBytes(key))).kind).toBe("ok");
 expect(domainOutcome(await server.makeTlsConfig(tlsBytes("garbage"),tlsBytes(tlsKey)),"http::invalid_server_config")).toMatchObject({reason:"tls_cert"});
 expect(domainOutcome(await server.makeTlsConfig(tlsBytes(tlsCert),tlsBytes("garbage")),"http::invalid_server_config")).toMatchObject({reason:"tls_key"});
 expect(domainOutcome(await server.makeTlsConfig(tlsBytes(tlsKey),tlsBytes(tlsKey)),"http::invalid_server_config")).toMatchObject({reason:"tls_cert"});
 expect(domainOutcome(await server.makeTlsConfig(tlsBytes(tlsCert),tlsBytes(tlsCert)),"http::invalid_server_config")).toMatchObject({reason:"tls_key"});
 expect(domainOutcome(await server.makeTlsConfig(tlsBytes(""),tlsBytes(tlsKey)),"http::invalid_server_config")).toMatchObject({reason:"tls_cert"});
 expect(domainOutcome(await server.makeTlsConfig(tlsBytes(tlsCert),tlsBytes(tlsKey+tlsKey)),"http::invalid_server_config")).toMatchObject({reason:"tls_key"});
 expect(domainOutcome(await server.makeTlsConfig(tlsBytes("-----BEGIN CERTIFICATE-----\nAA==\n-----END X-----\n"),tlsBytes(tlsKey)),"http::invalid_server_config")).toMatchObject({reason:"tls_cert"});
 expect(domainOutcome(await server.makeTlsConfig(tlsBytes(tlsCert),ownBytes(new Uint8Array([255,254]))),"http::invalid_server_config")).toMatchObject({reason:"tls_key"});
 expect(domainOutcome(await server.makeTlsConfig(tlsBytes(tlsCert),tlsBytes("x".repeat(1048577))),"http::invalid_server_config")).toMatchObject({reason:"tls_key"});
});

function localChain(dir:string):{cert:Uint8Array;key:Uint8Array}{
 const cnf=join(dir,"san.cnf");
 writeFileSync(cnf,"[req]\ndistinguished_name=dn\nreq_extensions=v3\n[dn]\n[v3]\nsubjectAltName=IP:127.0.0.1\n");
 const keyPath=join(dir,"key.pem"),certPath=join(dir,"cert.pem");
 const generated=Bun.spawnSync(["openssl","req","-x509","-newkey","rsa:2048","-keyout",keyPath,"-out",certPath,"-days","2","-nodes","-subj","/CN=127.0.0.1","-config",cnf,"-extensions","v3"],{stdout:"ignore",stderr:"pipe"});
 if(generated.exitCode!==0)throw new Error("openssl chain failed: "+generated.stderr.toString());
 return {cert:new Uint8Array(readFileSync(certPath)),key:new Uint8Array(readFileSync(keyPath))};
}

test("TLS serves a generated chain and rejects untrusted clients",async ()=>{
 const dir=mkdtempSync(join(tmpdir(),"can-tls-"));
 try{
  const {cert,key}=localChain(dir);
  const owned=await runOwnedRoot(async ()=>{
   const config=value(await server.makeConfig("127.0.0.1",18358n,1048576n,5000n));
   const tls=value(await server.makeTlsConfig(ownBytes(cert),ownBytes(key)));
   const route=value(await router.get("/x",async ()=>textResult("secure")));
   const table=value(await router.make([route]));
   const token=value(await server.startTls(config,table,tls));
   const trusted=await fetch("https://127.0.0.1:18358/x",{tls:{ca:cert}} as RequestInit);
   expect(trusted.status).toBe(200);expect(await trusted.text()).toBe("secure");
   await expect(fetch("https://127.0.0.1:18358/x")).rejects.toThrow();
   expect((await server.stop(token)).kind).toBe("ok");
   expect((await server.wait(token)).kind).toBe("ok");
   return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
 }finally{rmSync(dir,{recursive:true,force:true});}
});

test("TLS start maps unparseable DER to bind failures",async ()=>{
 const owned=await runOwnedRoot(async ()=>{
  const config=value(await server.makeConfig("127.0.0.1",18359n,1048576n,5000n));
  const tls=value(await server.makeTlsConfig(tlsBytes(tlsCert),tlsBytes(tlsKey)));
  const route=value(await router.get("/x",async ()=>textResult("never")));
  const table=value(await router.make([route]));
  expect(domainOutcome(await server.startTls(config,table,tls),"http::bind_failed")).toMatchObject({address:"127.0.0.1:18359"});
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
});

test("stream routes read incrementally under per-request scope",async ()=>{
 const seen:{status:number;head:string;leases:number}={status:0,head:"",leases:-1};
 const owned=await runOwnedRoot(async ()=>{
  const config=value(await server.makeConfig("127.0.0.1",18360n,1048576n,5000n));
  const upload=value(await router.stream(value(await router.post("/u",async request=>{
   const reader=value(await requests.bodyStream(request,4n));
   const first=value(await reads.readMany(reader,1n));
   const head=textOf(first[0]);
   await reads.closeReader(reader);
   return textResult(head);
  }))));
  const skip=value(await router.stream(value(await router.post("/skip",async ()=>textResult("skipped")))));
  const table=value(await router.make([upload,skip]));
  const token=value(await server.start(config,table));
  const streamed=await fetch("http://127.0.0.1:18360/u",{method:"POST",body:"hello-world"});
  seen.status=streamed.status;seen.head=await streamed.text();
  const ignored=await fetch("http://127.0.0.1:18360/skip",{method:"POST",body:"x".repeat(1000)});
  if(ignored.status!==200||await ignored.text()!=="skipped")throw new Error("abandoned body broke the route");
  const again=await fetch("http://127.0.0.1:18360/u",{method:"POST",body:"abcd"});
  if(again.status!==200||await again.text()!=="abcd")throw new Error("connection reuse broke after abandon");
  seen.leases=resourceStatus(token).leases;
  expect((await server.stop(token)).kind).toBe("ok");
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
 expect(seen).toEqual({status:200,head:"hell",leases:0});
});

test("dropped connections abort live reads",async ()=>{
 let observed="";
 const owned=await runOwnedRoot(async ()=>{
  let reached!:()=>void;const pending=new Promise<void>(resolve=>{reached=resolve;});
  const config=value(await server.makeConfig("127.0.0.1",18361n,1048576n,5000n));
  const upload=value(await router.stream(value(await router.post("/u",async request=>{
   const reader=value(await requests.bodyStream(request,4n));
   const first=value(await reads.readMany(reader,1n));
   observed+=textOf(first[0])+":";
   reached();
   const second=await reads.readMany(reader,1n);
   if(second.kind==="domain"){const details=domainFailureDiagnostics(second.value);observed+="failed:"+details.declaration.id+":"+JSON.stringify(details.payload);}
   else observed+="unexpected:"+second.kind;
   await reads.closeReader(reader);
   return textResult("done");
  }))));
  const table=value(await router.make([upload]));
  const token=value(await server.start(config,table));
  const controller=new AbortController();
  const body=new ReadableStream({start(c){c.enqueue(new TextEncoder().encode("ab"));}});
  const settled=fetch("http://127.0.0.1:18361/u",{method:"POST",body,signal:controller.signal,duplex:"half"}).then(response=>response.text(),()=>"client-gone");
  await pending;
  await Bun.sleep(50);
  controller.abort("client-gone");
  await settled;
  expect((await server.stop(token)).kind).toBe("ok");
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
 expect(observed).toBe("ab:failed:1316:{\"reason\":\"aborted\"}");
});
