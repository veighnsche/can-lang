// Named Can runtime checks. require evaluates its two arguments once each,
// in order, then takes a native boolean branch: true completes void, false
// produces checks::failed with the exact authored reason. The call-site span
// and invocation path travel in private occurrence metadata, never in the
// payload.
import {success,failure,type Completion,type AssertionContext} from "./completion.ts";
import {record} from "./data.ts";
import {createDomainRuntime} from "./domain.ts";
import type {FailureOrigin} from "./failure.ts";

export type ChecksErrors=Readonly<{failed:string}>;
export function createChecks(domain:ReturnType<typeof createDomainRuntime>,errors:ChecksErrors){
 return Object.freeze({
  async require(condition:boolean,reason:string,origin:FailureOrigin,_context?:AssertionContext):Promise<Completion<void>>{
   if(condition)return success(undefined);
   return failure(domain.create(errors.failed,record(errors.failed,[["reason",reason]]),origin));
  },
 });
}
