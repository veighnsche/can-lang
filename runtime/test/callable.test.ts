import {test,expect} from "bun:test";
import {ownCallable,callableReceipt} from "../callable.ts";
import {success,value} from "../completion.ts";
import {record} from "../data.ts";

test("native callable receipts retain immutable aliases and protected results",async()=>{
 let assimilated=0;
 const data=record("receipt",[["then",()=>{assimilated++;}]]);
 const target=async()=>success(data);
 const first=ownCallable("owner/site","target",[data],async()=>target());
 const second=ownCallable("owner/site","target",[data],async()=>target());
 expect(first).not.toBe(second);expect(Object.isFrozen(first)).toBe(true);
 expect(callableReceipt(first)?.captures[0]).toBe(data);
 expect(Object.isFrozen(callableReceipt(first))).toBe(true);
 expect(Object.isFrozen(callableReceipt(first)?.captures)).toBe(true);
 expect(value(await first())).toBe(data);expect(assimilated).toBe(0);
 expect(callableReceipt(()=>{})).toBeUndefined();expect(callableReceipt({site:"owner/site"})).toBeUndefined();
 expect(()=>ownCallable("owner/site","target",[],first)).toThrow("invalid callable construction");
});
