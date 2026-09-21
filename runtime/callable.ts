import {contextIdentity,type AssertionContext} from "./assert/context.ts";
import {callableIdentity,initializationCallableIdentity,type CallableIdentity} from "./assert/identity.ts";
import {registerCallableCaptures} from "./owner.ts";
// Native closures own execution and captures. This private receipt preserves
// creation-site/target/capture evidence for fixture identity; it is never a Can
// data projection or a diagnostic serialization surface.
type Receipt = Readonly<{identity:CallableIdentity;site:string;target:string;captures:readonly unknown[]}>;
const receipts=new WeakMap<Function,Receipt>();
export function ownCallable<T extends Function>(site:string,target:string,captures:readonly unknown[],value:T,resourceIndices:readonly number[]=captures.map((_,i)=>i),context?:AssertionContext):T {
 if (!site || !target || typeof value!=="function" || receipts.has(value)) throw new TypeError("invalid callable construction");
 if(new Set(resourceIndices).size!==resourceIndices.length || resourceIndices.some(i=>!Number.isInteger(i)||i<0||i>=captures.length))throw new TypeError("invalid callable resource capture");
 const guarded=registerCallableCaptures(value,resourceIndices.map(i=>captures[i]));
 const identity=context===undefined?initializationCallableIdentity(site,captures):callableIdentity(contextIdentity(context),site,captures);
 receipts.set(guarded,Object.freeze({identity,site,target,captures:Object.freeze([...captures])}));
 return Object.freeze(guarded);
}
export function callableReceipt(value:unknown):Receipt|undefined {
 return typeof value==="function" ? receipts.get(value) : undefined;
}

export function callableInstance(value:unknown):CallableIdentity|undefined{return callableReceipt(value)?.identity;}
