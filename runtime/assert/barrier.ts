import {checkedCompletion,type Completion} from "../completion.ts";
import {compareInvocations,invocationPath,type InvocationIdentity} from "./identity.ts";

declare const barrierBrand:unique symbol;
declare const frameBrand:unique symbol;
export type Barrier=Readonly<{[barrierBrand]:true}>;
export type Frame=Readonly<{[frameBrand]:true}>;
type Phase="reserved"|"running"|"waiting"|"fixture"|"done";
type FrameState={barrier:State;identity:InvocationIdentity;phase:Phase};
type Request={frame:FrameState;deliver:()=>Completion|Promise<Completion>;resolve:(value:Completion)=>void;reject:(cause:unknown)=>void};
type State={root:InvocationIdentity;frames:Set<FrameState>;identities:WeakSet<object>;pending:Request[];queued:boolean;compare:(a:InvocationIdentity,b:InvocationIdentity)=>number};
const barriers=new WeakMap<object,State>(),frames=new WeakMap<object,FrameState>();
const token=<T>():T=>Object.freeze(Object.create(null)) as T;
function barrier(value:Barrier):State{
 const found=value!==null&&typeof value==="object"?barriers.get(value):undefined;
 if(!found)throw new TypeError("invalid assertion barrier");return found;
}
function frame(value:Frame):FrameState{
 const found=value!==null&&typeof value==="object"?frames.get(value):undefined;
 if(!found)throw new TypeError("invalid assertion frame");return found;
}
function changed(state:State):void{
 if(state.queued)return;state.queued=true;
 queueMicrotask(()=>{
  state.queued=false;
  // This is an explicit quiescence test, not an estimate based on elapsed ticks.
  if([...state.frames].some(value=>value.phase==="running"||value.phase==="reserved")||state.pending.length===0)return;
  try{
   state.pending.sort((a,b)=>{
    const order=state.compare(a.frame.identity,b.frame.identity);
    if(!Number.isFinite(order))throw new TypeError("invalid conformance schedule order");
    return order;
   });
  }catch(cause){
   for(const request of state.pending.splice(0)){request.frame.phase="running";request.reject(cause)}
   return;
  }
  const request=state.pending.shift()!;
  request.frame.phase="running";
  // Keep the frame active through native promise delivery and its continuation.
  // Generated boundaries must park or finish it before another release occurs.
  Promise.resolve().then(request.deliver).then(checkedCompletion).then(request.resolve,request.reject);
 });
}
// Alternate comparison is compiler-conformance input only. Generated authored
// assertions always use the canonical comparator and expose no scheduling syntax.
export function createBarrier(root:InvocationIdentity,compare=compareInvocations):Barrier{
 invocationPath(root);
 const value=token<Barrier>();
 barriers.set(value,{root,frames:new Set(),identities:new WeakSet(),pending:[],queued:false,compare});
 return value;
}
export function reserveFrame(owner:Barrier,identity:InvocationIdentity):Frame{
 const state=barrier(owner);compareInvocations(state.root,identity);
 if(state.identities.has(identity))throw new TypeError("invocation frame already reserved");
 state.identities.add(identity);
 const value=token<Frame>(),entry:FrameState={barrier:state,identity,phase:"reserved"};
 frames.set(value,entry);state.frames.add(entry);return value;
}
function transition(value:Frame,from:Phase,to:Phase):void{
 const entry=frame(value);
 if(entry.phase!==from)throw new TypeError("invalid assertion frame transition");
 entry.phase=to;
 if(to==="done")entry.barrier.frames.delete(entry);
 changed(entry.barrier);
}
export function startFrame(value:Frame):void{transition(value,"reserved","running");}
export function suspendFrame(value:Frame):void{transition(value,"running","waiting");}
export function resumeFrame(value:Frame):void{transition(value,"waiting","running");}
export function finishFrame(value:Frame):void{transition(value,"running","done");}
export function abandonFrame(value:Frame):void{transition(value,"reserved","done");}
export function fixtureEvent(value:Frame,deliver:()=>Completion|Promise<Completion>):Promise<Completion>{
 const entry=frame(value);
 if(entry.phase!=="running")throw new TypeError("fixture requested outside a running invocation");
 entry.phase="fixture";
 const pending=new Promise<Completion>((resolve,reject)=>entry.barrier.pending.push({frame:entry,deliver,resolve,reject}));
 changed(entry.barrier);return pending;
}
export function barrierState(owner:Barrier){
 const state=barrier(owner);
 return Object.freeze({pending:state.pending.length,frames:Object.freeze([...state.frames].map(value=>Object.freeze({path:invocationPath(value.identity),phase:value.phase})))});
}
