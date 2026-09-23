import {success,failure,type Completion,type AssertionContext} from "../completion.ts";
import {denyLiveBoundary} from "../assert/context.ts";
import {record,array,dataArray,dataProperty} from "../data.ts";
import {ownBytes,copyBytes,byteLength,type Bytes} from "../bytes.ts";
import {createDomainRuntime} from "../domain.ts";
import {resourceStateFailure} from "../failure.ts";
import {CodecIssue} from "../codec/budget.ts";
import {decodeJSON,encodeJSON,type Schema} from "../codec/json.ts";
import {mediaType} from "../transport/media.ts";
import {strictParameters,decodeForm,FormIssue,type FormSchema} from "./form.ts";
import {renderSafe} from "./html.ts";
const origin=Object.freeze({source:"can:http-server",start:0,end:0,invocation:Object.freeze([])});
type Snapshot=Readonly<{method:string;path:string;query:readonly(readonly[string,string])[];queryInvalid:boolean;headers:readonly(readonly[string,string])[];body:Bytes}>;
const requests=new WeakMap<object,Snapshot>();
const object=(value:unknown):value is object=>value!==null&&(typeof value==="object"||typeof value==="function");
export function requestSnapshot(value:unknown):Snapshot{if(!object(value)||!requests.has(value))throw resourceStateFailure(undefined,origin);return requests.get(value)!;}
export function isRequest(value:unknown):boolean{return object(value)&&requests.has(value);}
export type SnapshotResult=Readonly<{kind:"request";value:unknown}|{kind:"rejected";status:400|413}>;
export function normalizedPath(url:URL):string{
 const path=decodeURIComponent(url.pathname);
 if(!path.startsWith("/")||/[\x00-\x1f\x7f\\]/.test(path))throw new URIError("invalid request path");
 return path;
}
// Private server ingress. Can receives only a detached immutable snapshot;
// native Request/Headers/stream objects never enter source-level values.
export async function snapshotRequest(request:Request,limit:number):Promise<SnapshotResult>{
 if(!Number.isSafeInteger(limit)||limit<1||limit>67108864)throw new TypeError("invalid server body budget");
 let url:URL,path:string;
 try{url=new URL(request.url);path=normalizedPath(url);}catch(cause){if(!(cause instanceof TypeError)&&!(cause instanceof URIError))throw cause;return {kind:"rejected",status:400};}
 let query:readonly(readonly[string,string])[]=[],queryInvalid=false;
 try{query=Object.freeze(Array.from(strictParameters(url.search.slice(1)),entry=>Object.freeze(entry)));}
 catch(cause){if(!(cause instanceof FormIssue)&&!(cause instanceof CodecIssue))throw cause;queryInvalid=true;}
 const headers=Object.freeze(Array.from(request.headers.entries(),entry=>Object.freeze(entry)));
 const chunks:Uint8Array[]=[];let size=0;
 if(request.body){const reader=request.body.getReader();let complete=false;
  try{for(;;){
   let next:Awaited<ReturnType<typeof reader.read>>;try{next=await reader.read();}catch{return {kind:"rejected",status:400};}
   if(next.done){complete=true;break;}
   if(next.value.byteLength>limit-size)return {kind:"rejected",status:413};
   size+=next.value.byteLength;if(next.value.byteLength!==0)chunks.push(new Uint8Array(next.value));
  }}finally{if(!complete)try{await reader.cancel();}catch{}reader.releaseLock();}
 }
 const data=new Uint8Array(size);let offset=0;for(const chunk of chunks){data.set(chunk,offset);offset+=chunk.byteLength;}
 const token=Object.freeze(Object.create(null));requests.set(token,Object.freeze({method:request.method,path,query,queryInvalid,headers,body:ownBytes(data)}));
 return {kind:"request",value:token};
}

type ResponseSnapshot=Readonly<{status:number;headers:readonly(readonly[string,string])[];body:Bytes|null}>;
const statuses=new WeakMap<object,number>(),bodyStatuses=new WeakMap<object,number>();
const serverHeaders=new WeakMap<object,readonly(readonly[string,string])[]>(),responses=new WeakMap<object,ResponseSnapshot>();
function opaque<T>(map:WeakMap<object,T>,data:T):unknown{const token=Object.freeze(Object.create(null));map.set(token,data);return token;}
function read<T>(map:WeakMap<object,T>,value:unknown):T{if(!object(value)||!map.has(value))throw resourceStateFailure(undefined,origin);return map.get(value)!;}
export function isHTTPValue(kind:string|undefined,value:unknown):boolean{
 if(kind==="request")return isRequest(value);
 const map=kind==="status"?statuses:kind==="body_status"?bodyStatuses:kind==="server_headers"?serverHeaders:kind==="server_response"?responses:undefined;
 return map!==undefined&&object(value)&&map.has(value);
}
// Each dispatch converts its complete immutable value once. A later dispatch
// can reuse the Can value without reusing an already-consumed native body.
export function nativeResponse(value:unknown,head=false):Response{
 const response=read(responses,value),headers:[string,string][]=response.headers.map(([name,value])=>[name,value]);
 // HEAD suppresses every body while keeping the entity length the complete
 // value produced, so HEAD and GET agree on length by construction.
 if(head&&response.body!==null)return new Response(null,{status:response.status,headers:[...headers,["content-length",byteLength(response.body).toString()]]});
 return new Response(response.body===null?null:new Uint8Array(copyBytes(response.body,origin)),{status:response.status,headers});
}
const forbiddenResponseHeaders=new Set(["content-type","content-length","x-content-type-options","content-security-policy","content-security-policy-report-only","connection","keep-alive","proxy-authenticate","proxy-authorization","te","trailer","transfer-encoding","upgrade"]);
export function createResponses(domain:ReturnType<typeof createDomainRuntime>,types:Pick<Types,"invalid"|"invalidData">){
 const invalid=(reason:string)=>failure(domain.create(types.invalid,record(types.invalid,[["reason",reason]]),origin));
 function status(value:bigint,body:boolean):Completion<unknown>{if(value<200n||value>599n||body&&(value===204n||value===205n||value===304n))return invalid("invalid_status");return success(opaque(body?bodyStatuses:statuses,Number(value)));}
 function response(status:unknown,headers:unknown,body:Bytes|null,mime?:string):unknown{
  const code=read(body===null?statuses:bodyStatuses,status),entries=read(serverHeaders,headers);
  const native=new Headers(entries.map(([name,value])=>[name,value]));native.set("x-content-type-options","nosniff");if(mime)native.set("content-type",mime);
  return opaque(responses,Object.freeze({status:code,headers:Object.freeze(Array.from(native.entries(),entry=>Object.freeze(entry))),body}));
 }
 const textBytes=(text:string)=>ownBytes(new TextEncoder().encode(text));
 return Object.freeze({
  async makeStatus(value:bigint,_context?:AssertionContext):Promise<Completion<unknown>>{return status(value,false);},
  async makeBodyStatus(value:bigint,_context?:AssertionContext):Promise<Completion<unknown>>{return status(value,true);},
  async ok(_context?:AssertionContext):Promise<Completion<unknown>>{return status(200n,true);},
  async unprocessable(_context?:AssertionContext):Promise<Completion<unknown>>{return status(422n,true);},
  async internal(_context?:AssertionContext):Promise<Completion<unknown>>{return status(500n,true);},
  async unavailable(_context?:AssertionContext):Promise<Completion<unknown>>{return status(503n,true);},
  async emptyHeaders(_context?:AssertionContext):Promise<Completion<unknown>>{return success(opaque(serverHeaders,Object.freeze([])));},
  async makeHeaders(input:unknown,_context?:AssertionContext):Promise<Completion<unknown>>{
   const headers=new Headers();
   for(const entry of dataArray(input)){
    const name=dataProperty(entry,"name"),value=dataProperty(entry,"value");
    if(typeof name!=="string"||typeof value!=="string")throw new TypeError("invalid compiler header");
    if(forbiddenResponseHeaders.has(name.toLowerCase())||name.toLowerCase().startsWith("hx-")||!name.isWellFormed()||!value.isWellFormed())return invalid("invalid_header");
    try{headers.append(name,value);}catch(cause){if(cause instanceof TypeError)return invalid("invalid_header");throw cause;}
   }
   return success(opaque(serverHeaders,Object.freeze(Array.from(headers.entries(),entry=>Object.freeze(entry)))));
  },
  async empty(status:unknown,headers:unknown,_context?:AssertionContext):Promise<Completion<unknown>>{return success(response(status,headers,null));},
  async bytes(status:unknown,headers:unknown,body:unknown,_context?:AssertionContext):Promise<Completion<unknown>>{return success(response(status,headers,ownBytes(copyBytes(body,origin)),"application/octet-stream"));},
  async text(status:unknown,headers:unknown,body:string,_context?:AssertionContext):Promise<Completion<unknown>>{return success(response(status,headers,textBytes(body),"text/plain; charset=utf-8"));},
  async html(status:unknown,headers:unknown,body:unknown,_context?:AssertionContext):Promise<Completion<unknown>>{return success(response(status,headers,textBytes(renderSafe(body)),"text/html; charset=utf-8"));},
  async json<T>(schema:Schema,status:unknown,headers:unknown,body:T,_context?:AssertionContext):Promise<Completion<unknown>>{
   try{return success(response(status,headers,encodeJSON(schema,body),"application/json; charset=utf-8"));}
   catch(cause){if(cause instanceof CodecIssue)return failure(domain.create(types.invalidData,record(types.invalidData,[["path",cause.path],["reason",cause.reason]]),origin));throw cause;}
  }
 });
}
type Types=Readonly<{invalid:string;limit:string;invalidData:string;header:string}>;
export function createRequests<Header>(domain:ReturnType<typeof createDomainRuntime>,types:Types){
 const invalid=(reason:string)=>failure(domain.create(types.invalid,record(types.invalid,[["reason",reason]]),origin));
 const bounded=(snapshot:Snapshot,limit:bigint):Completion<Bytes>=>limit<0n||byteLength(snapshot.body)>limit?failure(domain.create(types.limit,record(types.limit,[["limit",limit]]),origin)):success(snapshot.body);
 const codecFailure=(cause:unknown):Completion<never>=>{
  if(cause instanceof FormIssue)return invalid(cause.reason);
  if(cause instanceof CodecIssue)return failure(domain.create(types.invalidData,record(types.invalidData,[["path",cause.path],["reason",cause.reason]]),origin));
  throw cause;
 };
 function media(snapshot:Snapshot,form:boolean):boolean{
  const value=snapshot.headers.find(([name])=>name==="content-type")?.[1];if(value===undefined)return false;
  try{const parsed=mediaType(value);return parsed.type===(form?"application/x-www-form-urlencoded":"application/json")&&[...parsed.parameters].every(([key,value])=>key==="charset"&&value.toLowerCase()==="utf-8");}catch(cause){if(cause instanceof CodecIssue)return false;throw cause;}
 }
 return Object.freeze({
  async method(request:unknown,context?:AssertionContext):Promise<Completion<string>>{denyLiveBoundary(context,origin);return success(requestSnapshot(request).method);},
  async path(request:unknown,context?:AssertionContext):Promise<Completion<string>>{denyLiveBoundary(context,origin);return success(requestSnapshot(request).path);},
  async headers(request:unknown,context?:AssertionContext):Promise<Completion<readonly Header[]>>{denyLiveBoundary(context,origin);return success(array(requestSnapshot(request).headers.map(([name,value])=>record(types.header,[["name",name],["value",value]]) as Header)));},
  async queryAll(request:unknown,name:string,context?:AssertionContext):Promise<Completion<readonly string[]>>{denyLiveBoundary(context,origin);const snapshot=requestSnapshot(request);if(!name.isWellFormed())return invalid("query_name");if(snapshot.queryInvalid)return invalid("query_value");return success(array(snapshot.query.filter(([key])=>key===name).map(([,value])=>value)));},
  async queryOne(request:unknown,name:string,context?:AssertionContext):Promise<Completion<string>>{denyLiveBoundary(context,origin);const snapshot=requestSnapshot(request);if(!name.isWellFormed())return invalid("query_name");if(snapshot.queryInvalid)return invalid("query_value");const values=snapshot.query.filter(([key])=>key===name);return values.length===0?invalid("query_missing"):values.length!==1?invalid("query_repeated"):success(values[0]![1]);},
  async body(request:unknown,limit:bigint,context?:AssertionContext):Promise<Completion<Bytes>>{denyLiveBoundary(context,origin);return bounded(requestSnapshot(request),limit);},
  async json<T>(schema:Schema,request:unknown,limit:bigint,context?:AssertionContext):Promise<Completion<T>>{denyLiveBoundary(context,origin);const snapshot=requestSnapshot(request),body=bounded(snapshot,limit);if(body.kind!=="ok")return body;if(!media(snapshot,false))return invalid("unsupported_media_type");try{return success(decodeJSON(schema,body.value,Math.max(1,Number(limit>67108864n?67108864n:limit))) as T);}catch(cause){return codecFailure(cause);}},
  async form<T>(schema:FormSchema,request:unknown,limit:bigint,context?:AssertionContext):Promise<Completion<T>>{denyLiveBoundary(context,origin);const snapshot=requestSnapshot(request),body=bounded(snapshot,limit);if(body.kind!=="ok")return body;if(!media(snapshot,true))return invalid("unsupported_media_type");try{return success(decodeForm(schema,body.value,Number(byteLength(body.value))) as T);}catch(cause){return codecFailure(cause);}}
 });
}
