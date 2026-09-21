import {test,expect} from "bun:test";
import {success,type Completion} from "../completion.ts";
import {rootIdentity,invocationIdentity,participantIdentities,compareInvocations} from "../assert/identity.ts";
import {createBarrier,reserveFrame,startFrame,suspendFrame,resumeFrame,finishFrame,abandonFrame,fixtureEvent,barrierState,type Frame} from "../assert/barrier.ts";
const root=()=>rootIdentity({package:"p",declaration:"p::main",name:"sample"});
const tick=async()=>{for(let i=0;i<30;i++)await Promise.resolve()};

test("barrier waits for every reserved participant and orders shared requests by full path",async()=>{
 const identity=root(),owner=createBarrier(identity),participants=participantIdentities(identity,"p::main#0",[[0],[1]]);
 const a=reserveFrame(owner,participants[0]),b=reserveFrame(owner,participants[1]);
 const events:string[]=[];
 startFrame(b);
 const second=fixtureEvent(b,()=>{events.push("B");return success("B")}).then(value=>{finishFrame(b);return value});
 await tick();expect(events).toEqual([]);expect(barrierState(owner).pending).toBe(1);
 startFrame(a);await tick();expect(events).toEqual([]);
 const first=fixtureEvent(a,()=>{events.push("A");return success("A")}).then(value=>{finishFrame(a);return value});
 expect((await first).kind).toBe("ok");expect((await second).kind).toBe("ok");
 expect(events).toEqual(["A","B"]);expect(barrierState(owner).frames).toEqual([]);
});

test("arbitrarily delayed pure descendants keep fixture allocation behind a proven barrier",async()=>{
 const identity=root(),owner=createBarrier(identity),participants=participantIdentities(identity,"p::main#0",[[0],[1]]);
 const a=reserveFrame(owner,participants[0]),b=reserveFrame(owner,participants[1]);startFrame(a);startFrame(b);
 const events:string[]=[];
 const second=fixtureEvent(b,()=>{events.push("B");return success("B")}).then(value=>{finishFrame(b);return value});
 const child=reserveFrame(owner,invocationIdentity(participants[0],"p::pure#0"));suspendFrame(a);startFrame(child);
 for(let i=0;i<4;i++)await tick();expect(events).toEqual([]);
 finishFrame(child);resumeFrame(a);
 const first=fixtureEvent(a,()=>{events.push("A1");return success("A1")}).then(async()=>{
  const value=await fixtureEvent(a,()=>{events.push("A2");return success("A2")});finishFrame(a);return value;
 });
 await Promise.all([first,second]);expect(events).toEqual(["A1","A2","B"]);
});

test("native selection continuation stays active before a late fixture can release",async()=>{
 const identity=root(),owner=createBarrier(identity),participants=participantIdentities(identity,"p::main#0",[[0],[1]]);
 const a=reserveFrame(owner,participants[0]),b=reserveFrame(owner,participants[1]);startFrame(a);startFrame(b);
 const events:string[]=[];
 let gate!:Frame;
 const second=fixtureEvent(b,()=>{events.push("late B");return success("B")}).then(value=>{finishFrame(b);return value});
 const first=fixtureEvent(a,()=>success("A")).then(value=>{
  // Native coordination must make this transition atomically when its adapter
  // observes a completion that can settle the aggregate.
  gate=reserveFrame(owner,invocationIdentity(identity,"p::selection#0"));startFrame(gate);finishFrame(a);return value;
 });
 await first;await tick();expect(events).toEqual([]);
 events.push("native continuation");finishFrame(gate);await second;
 expect(events).toEqual(["native continuation","late B"]);
});

test("conformance can exercise reverse ordering without changing authored default",async()=>{
 const identity=root(),owner=createBarrier(identity,(a,b)=>-compareInvocations(a,b));
 const participants=participantIdentities(identity,"p::main#0",[[0],[1],[2]]),events:number[]=[];
 const ready=participants.map(value=>reserveFrame(owner,value));ready.forEach(startFrame);
 await Promise.all(ready.map((value,index)=>fixtureEvent(value,()=>{events.push(index);return success(index)}).then(result=>{finishFrame(value);return result})));
 expect(events).toEqual([2,1,0]);
});

test("barriers reject forged and duplicate identities and invalid transitions",async()=>{
 const identity=root(),owner=createBarrier(identity),value=reserveFrame(owner,identity);
 expect(()=>reserveFrame(owner,identity)).toThrow("already reserved");
 expect(()=>reserveFrame(owner,root())).toThrow("different assertion roots");
 expect(()=>startFrame({} as Frame)).toThrow("invalid assertion frame");
 expect(()=>fixtureEvent(value,()=>success(undefined))).toThrow("outside a running");
 startFrame(value);expect(()=>startFrame(value)).toThrow("transition");
 const malformed=fixtureEvent(value,()=>({kind:"ok",value:1} as Completion));
 await expect(malformed).rejects.toThrow();finishFrame(value);
 const abandoned=reserveFrame(owner,invocationIdentity(identity,"p::unused#0"));abandonFrame(abandoned);
 expect(barrierState(owner).frames).toEqual([]);
});

test("invalid conformance ordering rejects every pending request without stranding frames",async()=>{
 const identity=root(),owner=createBarrier(identity,()=>NaN);
 const ready=participantIdentities(identity,"p::main#0",[[0],[1]]).map(value=>reserveFrame(owner,value));
 ready.forEach(startFrame);
 const outcomes=await Promise.allSettled(ready.map(value=>fixtureEvent(value,()=>success(undefined)).finally(()=>finishFrame(value))));
 expect(outcomes.map(value=>value.status)).toEqual(["rejected","rejected"]);
 expect(barrierState(owner)).toEqual({pending:0,frames:[]});
 expect(()=>participantIdentities(identity,"p::main#1",[[1],[0]])).toThrow("flattened source order");
 expect(()=>participantIdentities(identity,"p::main#1",[[0],[0,0]])).toThrow("flattened source order");
});
