import {test,expect} from "bun:test";
import {performRequest} from "../transport/fetch.ts";
import {transportProblem,transportFault,Deadline} from "../transport/deadline.ts";
import {nativeOperation,nativeCompletion} from "../transport/owned.ts";
import {runOwnedRoot,launchOwned,registerResource,closeResource,useResource,type OwnerDiagnostic} from "../owner.ts";
import {success,failure} from "../completion.ts";
function deferred<T>(){let resolve!:(v:T)=>void;let reject!:(v:unknown)=>void;const promise=new Promise<T>((r,j)=>{resolve=r;reject=j;});return {promise,resolve,reject};}

test("late decoder defect is reported while timeout remains selected",async()=>{
 const decodeGate=deferred<never>(),timed=deferred<void>();const diagnostics:OwnerDiagnostic[]=[];
 const server=Bun.serve({hostname:"127.0.0.1",port:0,fetch(){return new Response("ok");}});
 try{
  const pending=runOwnedRoot(async()=>{
   try{await performRequest({endpoint:server.url.href,timeoutMilliseconds:25,maxBodyBytes:100,headers:[]},{path:"/",method:"GET",headers:[],query:[]},()=>undefined,()=>decodeGate.promise);throw new Error("did not time out");}
   catch(cause){expect(transportProblem(cause)).toEqual({kind:"timeout"});}
   timed.resolve();return success(undefined);
  },d=>{diagnostics.push(d);});
  await timed.promise;decodeGate.reject(new TypeError("late decoder defect"));
  expect((await pending).completion.kind).toBe("ok");
  expect(diagnostics).toMatchObject([{phase:"late",category:"native_exception"}]);
 }finally{server.stop(true);}
});

test("native continuation retains captured resource after parent timeout",async()=>{
 const proceed=deferred<void>();let closed=false,used=false,prematurelyClosed=false;const diagnostics:OwnerDiagnostic[]=[];
 const pending=runOwnedRoot(async()=>{
  const resource=registerResource("connection",{},()=>{closed=true;return success(undefined);},{scopeManaged:true});
  const group=launchOwned([{captures:[resource],run:async()=>{
   const deadline=new Deadline(10);
   try{await nativeOperation(async()=>{await proceed.promise;const result=await useResource(resource,"connection",()=>{used=true;return success(undefined);});if(result.kind!=="ok")throw result.value;return 1;},[resource],deadline);}
   catch(cause){expect(transportProblem(cause)).toEqual({kind:"timeout"});}finally{deadline.dispose();}
   return success(undefined);
  }}]);group.publish([0]);await group.promises[0];
  const close=closeResource(resource,"connection");await Promise.resolve();await Promise.resolve();
  prematurelyClosed=closed;proceed.resolve();await close;
  return success(undefined);
 },d=>{diagnostics.push(d);});
 const result=await pending;
 expect(result.completion.kind).toBe("ok");expect(prematurelyClosed).toBe(false);expect(used).toBe(true);
});


test("late classified transport failure does not produce a standard diagnostic",async()=>{
 const gate=deferred<void>(),timed=deferred<void>();const diagnostics:OwnerDiagnostic[]=[];
 const pending=runOwnedRoot(async()=>{
  const deadline=new Deadline(5);
  try{await nativeOperation(async()=>{await gate.promise;throw transportFault({kind:"transport",phase:"cancelled"});},[],deadline);}
  catch(cause){expect(transportProblem(cause)).toEqual({kind:"timeout"});}finally{deadline.dispose();}
  timed.resolve();return success(undefined);
 },d=>{diagnostics.push(d);});
 await timed.promise;gate.resolve();expect((await pending).completion.kind).toBe("ok");expect(diagnostics).toEqual([]);
});

test("synchronous late standard failure preserves timeout and reports its occurrence",async()=>{
 const diagnostics:OwnerDiagnostic[]=[];
 const result=await runOwnedRoot(async()=>{
  const deadline=new Deadline(2);
  try{await nativeOperation(async()=>{const start=performance.now();while(performance.now()-start<5){}throw new TypeError("late synchronous fault");},[],deadline);}
  catch(cause){expect(transportProblem(cause)).toEqual({kind:"timeout"});}finally{deadline.dispose();}
  return success(undefined);
 },d=>{diagnostics.push(d);});
 expect(result.completion.kind).toBe("ok");expect(diagnostics).toMatchObject([{phase:"late",category:"native_exception"}]);
});


test("late boxed standard decoder completion remains visible to ownership",async()=>{
 const {captureStandard}=await import("../failure.ts");
 const gate=deferred<void>(),timed=deferred<void>();const diagnostics:OwnerDiagnostic[]=[];
 const pending=runOwnedRoot(async()=>{
  const deadline=new Deadline(5);
  try{await nativeCompletion(async()=>{await gate.promise;return failure(captureStandard(new Error("boxed late fault"),{source:"test:late",start:0,end:0,invocation:[]}));},[],deadline);}
  catch(cause){expect(transportProblem(cause)).toEqual({kind:"timeout"});}finally{deadline.dispose();}
  timed.resolve();return success(undefined);
 },d=>{diagnostics.push(d);});
 await timed.promise;gate.resolve();expect((await pending).completion.kind).toBe("ok");expect(diagnostics).toMatchObject([{phase:"late",category:"native_exception"}]);
});
