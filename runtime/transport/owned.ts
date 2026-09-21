import {launchNative,type OwnedGroup} from "../owner.ts";
import {success,value,type Completion} from "../completion.ts";
import {transportProblem,type Deadline} from "./deadline.ts";

// Register before invoking native work. Pass every retained Can handle in
// captures. A bounded caller must pass its deadline here so selection and late
// standard diagnostics share the same ownership decision.
export async function nativeOperation<T>(operation:()=>Promise<T>,captures:readonly unknown[]=[],deadline?:Deadline):Promise<T>{
 const group=launchNative({captures,run:async()=>{
  try{return success({ok:true as const,value:await operation()});}
  catch(cause){
   // Classified transport outcomes (including the expected abort after timeout)
   // are domain outcomes. Unexpected faults retain their standard occurrence.
   if(transportProblem(cause))return success({ok:false as const,cause});
   throw cause;
  }
 }});
 const completion=await selected(group,deadline);
 const outcome=value(completion) as {ok:true;value:T}|{ok:false;cause:unknown};
 if(!outcome.ok)throw outcome.cause;return outcome.value;
}
export function cleanupOperation(operation:()=>Promise<unknown>,captures:readonly unknown[]=[]):void{
 const group=launchNative({captures,run:async()=>{await operation();return success(undefined);}});
 group.publish([]);
}

async function selected(group:OwnedGroup,deadline?:Deadline):Promise<Completion>{
 let completion:Completion;
 try{completion=deadline?await deadline.wait(group.promises[0]):await group.promises[0];}
 catch(cause){group.publish([]);throw cause;}
 group.publish([0]);return completion;
}
// Decoders already use protected completions. Keep their standard/domain kind
// visible to ownership instead of nesting it in a successful native-value box.
export async function nativeCompletion<T>(operation:()=>Completion<T>|Promise<Completion<T>>,captures:readonly unknown[]=[],deadline?:Deadline):Promise<Completion<T>>{
 const group=launchNative({captures,run:operation});
 return await selected(group,deadline) as Completion<T>;
}
