import {denyLiveBoundary,type AssertionContext} from "../assert/context.ts";
import {success,failure,type Completion} from "../completion.ts";
import {record} from "../data.ts";
import {createDomainRuntime} from "../domain.ts";
import {expectedIOFailure} from "./io.ts";
const origin=Object.freeze({source:"can:log",start:0,end:0,invocation:Object.freeze([])});
type Native=Readonly<{encode:(value:Readonly<{level:string;message:string}>)=>string;write:(line:string)=>Promise<number>}>;
const native:Native=Object.freeze({encode:JSON.stringify,write:line=>Bun.write(Bun.stderr,line)});
export function createLog(domain:ReturnType<typeof createDomainRuntime>,writeFailed:string,host:Native=native){
 async function write(level:"info"|"error",message:string,context?:AssertionContext):Promise<Completion<void>>{
  denyLiveBoundary(context,origin);
  // Private type validation precedes serialization; proxies/getters are never inspected.
  if(typeof message!=="string")throw new TypeError("invalid log text");
  const failed=(cause?:unknown)=>failure(domain.create(writeFailed,record(writeFailed,[["level",level]]),origin,cause));
  let line:string;
  try{line=host.encode({level,message})+"\n";}catch(cause){if(!(cause instanceof TypeError)&&!(cause instanceof RangeError))throw cause;return failed(cause);}
  try{await host.write(line);return success(undefined);}catch(cause){if(!expectedIOFailure(cause))throw cause;return failed(cause);}
 }
 return Object.freeze({
  async writeInfo(message:string,context?:AssertionContext){return write("info",message,context);},
  async writeError(message:string,context?:AssertionContext){return write("error",message,context);}
 });
}
