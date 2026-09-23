import {processErrorCode} from "./errors.ts";
// Group termination for detached children. Every Can child leads its own
// process group, so a negative pid reaches the child plus descendants that
// stayed in the group. ESRCH means the group is already gone: success.
export const sleep=(milliseconds:number)=>new Promise<void>(resolve=>{setTimeout(resolve,milliseconds);});
export function exitedOf(exitCode:number|null,signalCode:string|null):boolean{
 return exitCode!==null||signalCode!==null;
}
function signalGroup(pid:number,signal:"SIGTERM"|"SIGKILL"):void{
 if(!(pid>0))return;
 try{process.kill(-pid,signal);}
 catch(cause){
  const code=processErrorCode(cause);
  // ESRCH means the group is gone. EPERM on macOS means its members are
  // unreaped zombies that cannot take signals; live own-user members would
  // have accepted delivery instead. Either way nothing remains to signal.
  if(code==="ESRCH"||code==="EPERM")return;
  throw cause;
 }
}
export async function terminateGroup(pid:number,graceMs:number,isExited:()=>boolean):Promise<void>{
 signalGroup(pid,"SIGTERM");
 if(graceMs>0){await sleep(graceMs);if(isExited())return;}
 // Either no grace was granted or the group ignored SIGTERM: SIGKILL ends it.
 signalGroup(pid,"SIGKILL");
}
