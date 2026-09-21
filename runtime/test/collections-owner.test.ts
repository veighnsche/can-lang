import {test,expect} from "bun:test";
import {createHash} from "node:crypto";
import {catalogue} from "../catalogue.ts";
import {createDomainRuntime,domainFailureDiagnostics,type FailureShape} from "../domain.ts";
import {createMap,isMap} from "../collections/map.ts";
import {createSet,isSet,type ImmutableSet} from "../collections/set.ts";
import {record} from "../data.ts";
import {value,type Completion} from "../completion.ts";
const declarations=catalogue.errors.filter(e=>[1007,1008].includes(e.id));
const errors:FailureShape[]=declarations.map(e=>({identity:createHash("sha256").update("can-concrete-type-v1\0"+JSON.stringify(["error",e.identity])).digest("hex"),kind:"error",declaration:e.identity,arguments:[],fields:[],leaves:[],inputs:[],errors:[]}));
const domain=createDomainRuntime({declarations:declarations.map(e=>({identity:e.identity,name:e.name,id:e.id,parameters:0})),shapes:errors});
const maps=createMap<bigint,object>(domain,{map:"map-int-object",entry:"entry-int-object",absent:errors[0].identity,exists:errors[1].identity},"int");
function invalid(result:Completion,id:number){expect(result.kind).toBe("domain");if(result.kind!=="domain")throw Error("expected domain");expect(domainFailureDiagnostics(result.value).declaration.id).toBe(id);}
import {runOwnedRoot,registerResource,useResource,closeResource,resourceStatus,registerCallableCaptures} from "../owner.ts";
import {settle} from "../coordination.ts";
import {standardFailureKind} from "../failure.ts";
import {success,invoke} from "../completion.ts";
const origin={source:"test:map-owner",start:0,end:0,invocation:[]};
test("map-held resources lease before launch and remain usable by a losing participant",async()=>{
 for(const nested of [false,true]){
  const gate=Promise.withResolvers<void>(),selected=Promise.withResolvers<void>();let token:any,loser:Completion|undefined,constructionLeases=-1,captureLeases=-1;const events:string[]=[],diagnostics:unknown[]=[];
  const root=runOwnedRoot(async()=>{
   token=registerResource("test",{},()=>{events.push("close");return success(undefined)});
   let payload:object=token;
   if(nested){const callable=registerCallableCaptures(()=>{},[record("holder",[["resource",token]])]);payload=value(await maps.insert(value(await maps.empty()),2n,Object.freeze([callable,token])));}
   const map=value(await maps.insert(value(await maps.empty()),1n,payload));
   constructionLeases=resourceStatus(token).leases;
   await settle("race",[{captures:[map,map],run:async()=>{await gate.promise;const stored=nested?token:value(await maps.get(map,1n));loser=await invoke(()=>useResource(stored,"test",()=>{events.push("use");return success(undefined)}),origin);return loser;}},{captures:[],run:()=>success(undefined)}]);
   captureLeases=resourceStatus(token).leases;
   const close=closeResource(token,"test");await Promise.resolve();selected.resolve();await close;return success(undefined);
  },d=>{diagnostics.push(d)});
  await selected.promise;const pendingState=resourceStatus(token).state,pendingEvents=[...events];gate.resolve();
  const completed=await root;
  expect(constructionLeases).toBe(0);expect(captureLeases).toBe(1);expect(pendingState).toBe("closing");expect(pendingEvents).toEqual([]);
  expect(completed.completion.kind).toBe("ok");expect(loser?.kind).toBe("ok");expect(events).toEqual(["use","close"]);expect(diagnostics).toEqual([]);
 }
});
test("closed map-held resource rejects all participant preparation and rolls back prior leases",async()=>{
 const root=await runOwnedRoot(async()=>{
  const open=registerResource("open",{},()=>success(undefined)),closed=registerResource("closed",{},()=>success(undefined));
  await closeResource(closed,"closed");
  // Construction carries containment evidence even when resource state is closed.
  const map=value(await maps.insert(value(await maps.empty()),1n,closed));let starts=0;
  const result=await invoke(async()=>{await settle("all",[{captures:[open],run:()=>{starts++;return success(undefined)}},{captures:[map],run:()=>{starts++;return success(undefined)}}]);return success(undefined)},origin);
  expect(result.kind).toBe("standard");if(result.kind==="standard")expect(standardFailureKind(result.value)).toBe("resource_state");expect(starts).toBe(0);expect(resourceStatus(open).leases).toBe(0);await closeResource(open,"open");return success(undefined);
 });expect(root.completion.kind).toBe("ok");
});
test("replace and remove carry only new contents while older map aliases retain theirs",async()=>{
 const root=await runOwnedRoot(async()=>{
  const first=registerResource("first",{},()=>success(undefined)),second=registerResource("second",{},()=>success(undefined));
  const old=value(await maps.insert(value(await maps.empty()),1n,first)),replaced=value(await maps.replace(old,1n,second)),removed=value(await maps.remove(old,1n));
  await closeResource(first,"first");let starts=0;
  const okay=await settle("all",[{captures:[replaced,removed],run:()=>{starts++;expect(resourceStatus(second).leases).toBe(1);return success(undefined)}}]);
  expect(starts).toBe(1);expect(resourceStatus(second).leases).toBe(0);
  const bad=await invoke(async()=>{await settle("all",[{captures:[old],run:()=>{starts++;return success(undefined)}}]);return success(undefined)},origin);
  expect(bad.kind).toBe("standard");expect(starts).toBe(1);await closeResource(second,"second");return success(undefined);
 });expect(root.completion.kind).toBe("ok");
});
test("opaque containment traversal does not invoke proxies or getters",async()=>{
 let traps=0;const proxy=new Proxy({}, {get(){traps++;throw Error("trap")},ownKeys(){traps++;throw Error("trap")},getOwnPropertyDescriptor(){traps++;throw Error("trap")}});
 const root=await runOwnedRoot(async()=>{
  const map=value(await maps.insert(value(await maps.empty()),1n,proxy));
  const proxiedMap=new Proxy(map,{get(){traps++;throw Error("trap")},ownKeys(){traps++;throw Error("trap")}});
  await settle("all",[{captures:[map,proxiedMap],run:()=>success(undefined)}]);return success(undefined);
 });expect(root.completion.kind).toBe("ok");expect(traps).toBe(0);
});
