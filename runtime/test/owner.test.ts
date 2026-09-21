import {test,expect} from "bun:test";
import {runOwnedRoot,launchOwned,registerResource,useResource,closeResource,resourceStatus,withScope,guardCallback,registerCallableCaptures,type Resource,type OwnerDiagnostic} from "../owner.ts";
import {success,failure,invoke,value,type Completion} from "../completion.ts";
import {resourceStateFailure,standardFailureKind,captureStandard} from "../failure.ts";
const origin={source:"test:owner",start:0,end:0,invocation:[]};
function deferred<T>(){let resolve!:(value:T)=>void,reject!:(cause:unknown)=>void;const promise=new Promise<T>((a,b)=>{resolve=a;reject=b;});return {promise,resolve,reject};}
const empty=()=>success(undefined);

test("published groups inspect each losing outcome once while preserving occurrence dedup",async()=>{
 const count=256,ready=deferred<void>(),release=deferred<Completion>();
 const occurrence=captureStandard(new Error("shared losing occurrence"),origin),failed=failure(occurrence);
 const diagnostics:OwnerDiagnostic[]=[];
 const original=Set.prototype.has;let inspections=0;
 // Count selected-index queries, not elapsed time. The selected set contains
 // index zero; object-valued ownership/diagnostic sets are excluded.
 Set.prototype.has=function(value:unknown){
  if(typeof value==="number"&&this.size===1&&original.call(this,0))inspections++;
  return original.call(this,value);
 };
 try{
  const root=runOwnedRoot(async()=>{
   const group=launchOwned([
    {captures:[],run:empty},
    ...Array.from({length:count},()=>({captures:[],run:()=>release.promise})),
   ]);
   const selected=await group.promises[0];group.publish([0]);ready.resolve();return selected;
  },diagnostic=>{diagnostics.push(diagnostic)});
  await ready.promise;release.resolve(failed);
  const result=await root;
  expect(result.completion.kind).toBe("ok");expect(result.cleanupFailed).toBe(false);
  expect(inspections).toBe(count);expect(diagnostics).toHaveLength(1);
  expect(diagnostics[0].message).toBe("Error: shared losing occurrence");
 }finally{Set.prototype.has=original;}
});

test("late standard failures stay observed without replacing the selected success",async()=>{
 const winner=deferred<Completion>(),loser=deferred<Completion>(),selected=deferred<void>();
 const diagnostics:OwnerDiagnostic[]=[];let done=false;
 const root=runOwnedRoot(async()=>{
  const group=launchOwned([{run:()=>winner.promise,captures:[]},{run:()=>loser.promise,captures:[]}]);
  const result=await group.promises[0];group.publish([0]);selected.resolve();return result;
 },d=>{diagnostics.push(d);}).then(result=>{done=true;return result;});
 winner.resolve(success(7n));await selected.promise;await Promise.resolve();expect(done).toBe(false);
 const cause=Object.assign(new Error("late boom"),{privatePayload:"private secret"});loser.reject(cause);
 const result=await root;expect(value(result.completion)).toBe(7n);expect(result.cleanupFailed).toBe(false);
 expect(diagnostics.length).toBe(1);expect(diagnostics[0].message).toBe("Error: late boom");expect(diagnostics[0].phase).toBe("late");expect(diagnostics[0].category).toBe("native_exception");expect(JSON.stringify(diagnostics)).not.toContain("private secret");
});

test("closing preserves existing-owner subleases and deadlines never force close",async()=>{
 const proceed=deferred<void>(),finished=deferred<void>();const events:string[]=[];let token!:Resource;
 const result=await runOwnedRoot(async()=>{
  token=registerResource("test.pool",{value:9},()=>{events.push("native-close");return empty();});
  const group=launchOwned([{captures:[token],run:async()=>{await proceed.promise;
   const used=await useResource(token,"test.pool",native=>{events.push("sublease");return success((native as {value:number}).value);});
   finished.resolve();return used;
  }}]);
  group.publish([]);
  const closing=closeResource(token,"test.pool",{milliseconds:0,failure:()=>failure(resourceStateFailure(undefined,origin))});
  const timed=await closing;expect(timed.kind).toBe("standard");expect(resourceStatus(token)).toMatchObject({state:"closing",leases:1});expect(events).toEqual([]);
  const refused=await invoke(()=>{launchOwned([{captures:[token],run:empty}]);return empty();},origin);
  expect(refused.kind).toBe("standard");expect(resourceStatus(token).leases).toBe(1);
  proceed.resolve();await finished.promise;return empty();
 });
 expect(events).toEqual(["sublease","native-close"]);expect(resourceStatus(token).state).toBe("closed");expect(result.cleanupFailed).toBe(true);
});

test("reverse automatic cleanup reports omissions even when native closes succeed",async()=>{
 const events:string[]=[];const diagnostics:OwnerDiagnostic[]=[];
 const result=await runOwnedRoot(()=>{
  registerResource("first",{},()=>{events.push("first");return empty();});
  registerResource("second",{},()=>{events.push("second");return empty();});
  return empty();
 },d=>{diagnostics.push(d);});
 expect(events).toEqual(["second","first"]);expect(result.cleanupFailed).toBe(true);expect(diagnostics.map(d=>d.category)).toEqual(["cleanup","cleanup"]);
});

test("scoped handles reject escapes and host callbacks block user code",async()=>{
 let token!:Resource;let callback!:()=>Promise<Completion>;let guarded!:()=>Promise<Completion>;let executions=0;
 const result=await runOwnedRoot(async()=>{
  const scoped=await withScope(async scope=>{
   token=registerResource("transaction",{},empty,{scopeManaged:true});
   callback=registerCallableCaptures(async()=>useResource(token,"transaction",()=>{executions++;return empty();}),[token]);
   guarded=guardCallback(scope,async()=>{executions++;return empty();});
   return empty();
  });expect(scoped.kind).toBe("ok");expect(resourceStatus(token).state).toBe("closed");
  for(const run of [callback,guarded,()=>useResource(token,"transaction",empty)]){
   const refused=await invoke(run,origin);expect(refused.kind).toBe("standard");
   if(refused.kind==="standard")expect(standardFailureKind(refused.value)).toBe("resource_state");
  }
  return empty();
 });
 expect(executions).toBe(0);expect(result.cleanupFailed).toBe(false);
});

test("resource preparation fails before every participant and rolls back leases",async()=>{
 let launched=0;
 const result=await runOwnedRoot(async()=>{
  const token=registerResource("pool",{},empty);
  const expired=registerResource("pool",{},empty);await closeResource(expired,"pool");
  const refused=await invoke(()=>{launchOwned([{captures:[token],run:()=>{launched++;return empty();}},{captures:[expired],run:empty}]);return empty();},origin);
  expect(refused.kind).toBe("standard");expect(resourceStatus(token).leases).toBe(0);
  await closeResource(token,"pool");return empty();
 });expect(launched).toBe(0);expect(result.cleanupFailed).toBe(false);
});

test("interleaved roots preserve native async context and reject cross-root handles",async()=>{
 const ready=deferred<Resource>(),release=deferred<void>();let a!:Resource;
 const first=runOwnedRoot(async()=>{
  a=registerResource("pool",{root:"a"},empty);ready.resolve(a);await release.promise;
  const read=await useResource(a,"pool",native=>success((native as {root:string}).root));expect(value(read)).toBe("a");
  await closeResource(a,"pool");return success("first");
 });
 const second=runOwnedRoot(async()=>{
  const foreign=await ready.promise;
  const refused=await invoke(()=>useResource(foreign,"pool",empty),origin);expect(refused.kind).toBe("standard");
  const b=registerResource("pool",{root:"b"},empty);await Promise.resolve();
  expect(value(await useResource(b,"pool",native=>success((native as {root:string}).root)))).toBe("b");
  await closeResource(b,"pool");release.resolve();return success("second");
 });
 const results=await Promise.all([first,second]);expect(results.map(result=>value(result.completion))).toEqual(["first","second"]);expect(results.every(result=>!result.cleanupFailed)).toBe(true);
});

test("scoped shutdown drains losers before releasing the scoped native resource",async()=>{
 const ready=deferred<void>(),gate=deferred<void>();let closed=false,settled=false;
 const pending=runOwnedRoot(async()=>withScope(async()=>{
  const token=registerResource("transaction",{},()=>{closed=true;return empty();},{scopeManaged:true});
  const group=launchOwned([{captures:[token],run:async()=>{await gate.promise;return useResource(token,"transaction",empty);}}]);
  group.publish([]);ready.resolve();return empty();
 })).then(result=>{settled=true;return result;});
 await ready.promise;await Promise.resolve();expect(closed).toBe(false);expect(settled).toBe(false);
 gate.resolve();const result=await pending;expect(result.completion.kind).toBe("ok");expect(closed).toBe(true);expect(result.cleanupFailed).toBe(false);
});

test("wrong-kind and forged resources reject without touching native state",async()=>{
 let touched=0,traps=0;
 const root=await runOwnedRoot(async()=>{
  const token=registerResource("pool",{},empty);
  const forged=new Proxy({}, {get(){traps++;throw Error("trap");},getPrototypeOf(){traps++;throw Error("trap");}});
  for(const [value,kind] of [[token,"transaction"],[forged,"pool"]] as const){
   const refused=await invoke(()=>useResource(value,kind,()=>{touched++;return empty();}),origin);expect(refused.kind).toBe("standard");
  }
  await closeResource(token,"pool");
  expect((await invoke(()=>closeResource(token,"pool"),origin)).kind).toBe("standard");
  return empty();
 });expect(root.cleanupFailed).toBe(false);expect(touched).toBe(0);expect(traps).toBe(0);
});

test("automatic close faults propagate cleanup from a managed scope",async()=>{
 const root=await runOwnedRoot(()=>withScope(async()=>{
  registerResource("transaction",{},()=>{throw new Error("private native close fault");},{scopeManaged:true});
  return empty();
 }));
 expect(root.cleanupFailed).toBe(true);expect(root.completion.kind).toBe("standard");
 if(root.completion.kind==="standard")expect(standardFailureKind(root.completion.value)).toBe("cleanup");
});

test("resources opened by a loser retain its lease across registered shutdown expiry",async()=>{
 const create=deferred<void>(),use=deferred<void>(),expired=deferred<void>();let token!:Resource;let closed=false;
 const root=runOwnedRoot(()=>{
  const group=launchOwned([{captures:[],run:async()=>{
   await create.promise;
   token=registerResource("pool",{},()=>{closed=true;return empty();},{shutdownMilliseconds:0});
   await use.promise;return useResource(token,"pool",empty);
  }}]);group.publish([]);return empty();
 },diagnostic=>{if(diagnostic.phase==="cleanup")expired.resolve();});
 create.resolve();await expired.promise;expect(resourceStatus(token)).toMatchObject({state:"closing",leases:1});expect(closed).toBe(false);
 use.resolve();const result=await root;expect(result.cleanupFailed).toBe(true);expect(closed).toBe(true);
});

test("a participant-created managed child scope releases its initial lease at callback exit",async()=>{
 let closes=0;
 const result=await runOwnedRoot(async()=>{
  const group=launchOwned([{captures:[],run:()=>withScope(async()=>{
   registerResource("transaction",{},()=>{closes++;return empty();},{scopeManaged:true});return empty();
  })}]);const completed=await group.promises[0];group.publish([0]);return completed;
 });
 expect(closes).toBe(1);expect(result.cleanupFailed).toBe(false);expect(result.completion.kind).toBe("ok");
});

test("a participant can explicitly close its own newly opened resource",async()=>{
 let closed=0;
 const result=await runOwnedRoot(async()=>{
  const group=launchOwned([{captures:[],run:async()=>{
   const token=registerResource("pool",{},()=>{closed++;return empty();});
   await useResource(token,"pool",empty);return closeResource(token,"pool");
  }}]);const outcome=await group.promises[0];group.publish([0]);return outcome;
 });expect(result.cleanupFailed).toBe(false);expect(closed).toBe(1);
});

test("idempotent reentrant close shares the installed native close attempt",async()=>{
 let calls=0;let reentered:Promise<Completion<void>>|undefined;
 const result=await runOwnedRoot(async()=>{
  let token!:Resource;
  token=registerResource("pool",{},()=>{calls++;if(calls===1)reentered=closeResource(token,"pool");return empty();},{idempotent:true});
  expect((await closeResource(token,"pool")).kind).toBe("ok");expect((await reentered!).kind).toBe("ok");return empty();
 });expect(calls).toBe(1);expect(result.cleanupFailed).toBe(false);
});

test("host callback wrappers preserve nested capture evidence for participants",async()=>{
 const gate=deferred<void>();let token!:Resource,closed=false;
 const result=await runOwnedRoot(()=>withScope(async scope=>{
  token=registerResource("pool",{},()=>{closed=true;return empty();},{scopeManaged:true});
  const callback=guardCallback(scope,registerCallableCaptures(async()=>{await gate.promise;return useResource(token,"pool",empty);},[token]));
  const group=launchOwned([{captures:[callback],run:callback}]);group.publish([]);
  expect(resourceStatus(token).leases).toBe(1);
  const closing=closeResource(token,"pool");await Promise.resolve();expect(closed).toBe(false);
  gate.resolve();await closing;return empty();
 }));expect(result.cleanupFailed).toBe(false);expect(closed).toBe(true);
});

test("ordinary callable receipts do not add self-deadlocking close leases",async()=>{
 let closes=0;
 const result=await runOwnedRoot(async()=>{
  const token=registerResource("pool",{},()=>{closes++;return empty();});
  const callback=registerCallableCaptures(()=>closeResource(token,"pool"),[token]);
  expect(resourceStatus(token).leases).toBe(0);return callback();
 });expect(closes).toBe(1);expect(result.cleanupFailed).toBe(false);
});


test("selected standard outcomes are not late and repeated losing occurrences report once",async()=>{
 const occurrence=captureStandard(new Error("one occurrence"),origin);const diagnostics:OwnerDiagnostic[]=[];
 const result=await runOwnedRoot(async()=>{
  const selected=launchOwned([{captures:[],run:()=>failure(occurrence)}]);await selected.promises[0];selected.publish([0]);
  const losers=launchOwned([{captures:[],run:()=>failure(occurrence)},{captures:[],run:()=>failure(occurrence)}]);losers.publish([]);
  return empty();
 },diagnostic=>{diagnostics.push(diagnostic);});
 expect(result.cleanupFailed).toBe(false);expect(diagnostics.length).toBe(1);expect(diagnostics[0].message).toBe("Error: one occurrence");
});

test("native callbacks bind their operation scope and drain without ambient context",async()=>{
 const ready=deferred<void>(),bodyExit=deferred<void>(),finish=deferred<void>();
 let callback!:()=>Promise<Completion>;let token!:Resource;let done=false;let entered=0;
 const root=runOwnedRoot(()=>withScope(async scope=>{
  callback=guardCallback(scope,async()=>{
   entered++;token=registerResource("callback",{},empty,{scopeManaged:true});
   await finish.promise;return useResource(token,"callback",empty);
  });ready.resolve();await bodyExit.promise;return empty();
 })).then(result=>{done=true;return result;});
 await ready.promise;
 const pending=callback(); // This test continuation was created outside the root.
 expect(entered).toBe(1);bodyExit.resolve();await Promise.resolve();await Promise.resolve();
 expect(done).toBe(false);expect(resourceStatus(token).leases).toBe(1);
 finish.resolve();expect((await pending).kind).toBe("ok");
 expect((await root).cleanupFailed).toBe(false);expect(resourceStatus(token).state).toBe("closed");
 const refused=await invoke(callback,origin);expect(refused.kind).toBe("standard");expect(entered).toBe(1);
});
