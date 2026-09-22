import {providerHTTP} from "../assert/provider.ts";
import {denyLiveBoundary} from "../assert/context.ts";
import {invoke,success,failure,type Completion,type AssertionContext} from "../completion.ts";
import {array,record} from "../data.ts";
import {ownBytes,copyBytes} from "../bytes.ts";
import {createDomainRuntime} from "../domain.ts";
import type {FailureOrigin} from "../failure.ts";
import {encodeJSON,decodeJSON,type Schema} from "../codec/json.ts";
import {CodecIssue,reject} from "../codec/budget.ts";
import {createTransport,type HTTPTypes} from "./http.ts";
import {createNormalizer} from "./normalize.ts";
import type {Connection} from "./request.ts";
import type {NativeRequest} from "./fetch.ts";
import {responseMedia} from "./media.ts";
export type FetchMode="json"|"text"|"bytes";
export type FetchBody=Readonly<{mode:FetchMode;value:unknown;schema?:Schema}>;
export type FetchResult=Readonly<{mode:FetchMode;schema?:Schema;envelope?:string}>;
export function createNamedFetch(domain:ReturnType<typeof createDomainRuntime>,types:HTTPTypes&Readonly<{invalidData:string;failed:string}>,readEnvironment:(name:string)=>string|undefined){
 const transport=createTransport(domain,types,readEnvironment);
 const normalize=createNormalizer(domain,{failed:types.failed,leaves:[types.invalid,types.credential,types.transport,types.timeout,types.limit,types.status,types.invalidData]});
 return Object.freeze({async request<T>(connection:Connection,request:Omit<NativeRequest,"body"|"bodyEncoding"|"envelope">,body:FetchBody|undefined,result:FetchResult,origin:FailureOrigin,operation:string,context?:AssertionContext):Promise<Completion<T>>{
  const native={boundary:"native",operation} as const;
  function invalid(cause:CodecIssue):Completion<never>{return failure(domain.create(types.invalidData,record(types.invalidData,[["path",cause.path],["reason",cause.reason]]),origin,undefined,native));}
  function limit():Completion<never>{return failure(domain.create(types.limit,record(types.limit,[["limit",BigInt(connection.maxBodyBytes)]]),origin,undefined,native));}
  return normalize.map(await invoke(async()=>{
   let encoded:Uint8Array|undefined;
   if(body){
    if(request.method==="GET"||request.method==="HEAD")throw new TypeError("checked fetch method cannot have body");
    try{
     if(body.mode==="json")encoded=copyBytes(encodeJSON(body.schema!,body.value,connection.maxBodyBytes),origin);
     else if(body.mode==="bytes")encoded=copyBytes(body.value,origin);
     else {if(typeof body.value!=="string")throw new TypeError("checked text body must be string");if(body.value.length>connection.maxBodyBytes)return limit();if(!body.value.isWellFormed())reject("","unicode_scalar");encoded=new TextEncoder().encode(body.value);}
    }catch(cause){if(cause instanceof CodecIssue)return cause.reason==="byte_limit"?limit():invalid(cause);throw cause;}
    if(encoded.byteLength>connection.maxBodyBytes)return limit();
   }
   const exchange=request.exchange??providerHTTP(context,origin);if(exchange===undefined)denyLiveBoundary(context,origin);
   return transport.request(connection,{...request,exchange,body:encoded,bodyEncoding:body?.mode,envelope:result.envelope!==undefined},(bytes,metadata)=>{
    let value:unknown;
    try{
     if(result.mode==="bytes")value=ownBytes(bytes);
     else {
      responseMedia(metadata.headers.find(header=>header.name==="content-type")?.value,result.mode==="json");
      if(result.mode==="json")value=decodeJSON(result.schema!,ownBytes(bytes),connection.maxBodyBytes);
      else {try{value=new TextDecoder("utf-8",{fatal:true,ignoreBOM:true}).decode(bytes);}catch(cause){if(cause instanceof TypeError)reject("","utf8");throw cause;}}
     }
    }catch(cause){if(cause instanceof CodecIssue)return invalid(cause);throw cause;}
    if(result.envelope!==undefined)value=record(result.envelope,[["status",BigInt(metadata.status)],["headers",array(metadata.headers.map(header=>record(types.header,[["name",header.name],["value",header.value]])))],["body",value]]);
    return success(value as T);
   },origin,operation);
  },origin),operation);
 }});
}
