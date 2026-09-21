import {launchNative} from "../owner.ts";
import {success,value} from "../completion.ts";

// Register before invoking native work. The caller may stop waiting, but root
// supervision retains the actual promise through native settlement.
export async function nativeOperation<T>(operation:()=>Promise<T>):Promise<T>{
 const group=launchNative({captures:[],run:async()=>{
  try{return success({ok:true as const,value:await operation()});}
  catch(cause){return success({ok:false as const,cause});}
 }});
 group.publish([0]);
 const outcome=value(await group.promises[0]) as {ok:true;value:T}|{ok:false;cause:unknown};
 if(!outcome.ok)throw outcome.cause;return outcome.value;
}
export function cleanupOperation(operation:()=>Promise<unknown>):void{
 const group=launchNative({captures:[],run:async()=>{await operation();return success(undefined);}});
 group.publish([]);
}
