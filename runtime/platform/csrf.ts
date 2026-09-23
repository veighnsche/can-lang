// Session-bound CSRF tokens over Bun.CSRF with explicit secrets. Every
// call takes a caller-supplied secret plus a nonempty session id with
// fixed base64url/sha256; the native thread-local default is never
// used. Token faults (malformed, expired, wrong session or secret)
// answer false; only invalid configuration fails a domain error, and
// no secret or token material ever enters a diagnostic.
import {success,failure,caught,type Completion,type AssertionContext} from "../completion.ts";
import {denyLiveBoundary} from "../assert/context.ts";
import {record} from "../data.ts";
import {createDomainRuntime} from "../domain.ts";
const origin=Object.freeze({source:"can:csrf",start:0,end:0,invocation:Object.freeze([])});
const DURATION_LIMIT=9007199254740991n;
export function createCSRF(domain:ReturnType<typeof createDomainRuntime>,ids:Readonly<{invalid:string}>){
  const invalid=(reason:string)=>failure(domain.create(ids.invalid,record(ids.invalid,[["reason",reason]]),origin));
  const duration=(value:unknown):number|ReturnType<typeof invalid>=>{
    if(typeof value!=="bigint"||value<0n||value>DURATION_LIMIT)return invalid("duration");
    return Number(value);
  };
  const credential=(secret:unknown,session:unknown):ReturnType<typeof invalid>|undefined=>{
    if(typeof secret!=="string"||typeof session!=="string")throw new TypeError("invalid compiler csrf credential");
    if(secret==="")return invalid("secret");
    if(session==="")return invalid("session");
    return undefined;
  };
  return Object.freeze({
    async generate(secret:unknown,session:unknown,expiresInMs:unknown,context?:AssertionContext):Promise<Completion<string>>{
      denyLiveBoundary(context,origin);
      const bad=credential(secret,session);
      if(bad!==undefined)return bad;
      const expires=duration(expiresInMs);
      if(typeof expires!=="number")return expires;
      try{
        return success(Bun.CSRF.generate(secret as string,{sessionId:session as string,expiresIn:expires}));
      }catch(cause){return caught(cause,origin);}
    },
    async verify(secret:unknown,session:unknown,token:unknown,maxAgeMs:unknown,_context?:AssertionContext):Promise<Completion<boolean>>{
      const bad=credential(secret,session);
      if(bad!==undefined)return bad;
      if(typeof token!=="string")throw new TypeError("invalid compiler csrf token");
      const age=duration(maxAgeMs);
      if(typeof age!=="number")return age;
      try{
        return success(Bun.CSRF.verify(token,{secret:secret as string,sessionId:session as string,maxAge:age}));
      }catch(cause){return caught(cause,origin);}
    },
  });
}
