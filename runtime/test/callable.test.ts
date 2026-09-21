import {test,expect} from "bun:test";
import {ownCallable,callableReceipt} from "../callable.ts";
import {success,value} from "../completion.ts";
import {record} from "../data.ts";

test("native callable receipts retain immutable aliases and protected results",async()=>{
 let assimilated=0;
 const data=record("receipt",[["then",()=>{assimilated++;}]]);
 const target=async()=>success(data);
 const first=ownCallable("owner#0","target",[data],async()=>target());
 const second=ownCallable("owner#0","target",[data],async()=>target());
 expect(first).not.toBe(second);expect(Object.isFrozen(first)).toBe(true);
 expect(callableReceipt(first)?.captures[0]).toBe(data);
 expect(Object.isFrozen(callableReceipt(first))).toBe(true);
 expect(Object.isFrozen(callableReceipt(first)?.captures)).toBe(true);
 expect(value(await first())).toBe(data);expect(assimilated).toBe(0);
 expect(callableReceipt(()=>{})).toBeUndefined();expect(callableReceipt({site:"owner#0"})).toBeUndefined();
 expect(()=>ownCallable("owner#0","target",[],first)).toThrow("invalid callable construction");
});

test("creation receipts distinguish frozen captures and preserve initialization identity across roots",async()=>{
 const {assertionContext,callContext,contextIdentity,finishAssertionExecution,closeContext}=await import("../assert/context.ts");
 const {invocationPath}=await import("../assert/identity.ts");
 const {callableInstance}=await import("../callable.ts");
 const makeRoot=(name:string)=>assertionContext({package:"p",declaration:"p::main",name});
 const root=makeRoot("first"),other=makeRoot("second");
 const initialized=ownCallable("p::initialized#0","target",["private setup"],async()=>success(undefined));
 const captures=["private near"];
 const first=ownCallable("p::factory#0","target",captures,async()=>success(undefined),[],root);
 captures[0]="changed";
 const second=ownCallable("p::factory#0","target",["different near"],async()=>success(undefined),[],root);
 const path=async(context:typeof root,fn:typeof first)=>callContext(context,"p::invoke#0",child=>invocationPath(contextIdentity(child!)),callableInstance(fn));
 const a=await path(root,first),b=await path(root,second);
 expect(a.segments[0].callable?.occurrence).toBe(0);expect(b.segments[0].callable?.occurrence).toBe(1);
 expect(a.segments[0].callable?.instance).not.toBe(b.segments[0].callable?.instance);
 expect(callableReceipt(first)?.captures).toEqual(["private near"]);
 expect(JSON.stringify(a)).not.toContain("private near");
 const x=await path(root,initialized),y=await path(other,initialized);
 expect(x.segments[0].callable).toEqual(y.segments[0].callable);
 expect(x.root).not.toEqual(y.root);
 await expect(path(other,first)).rejects.toThrow("another assertion root");
 for(const context of [root,other]){finishAssertionExecution(context);closeContext(context)}
});
