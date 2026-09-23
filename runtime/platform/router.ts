import {success,failure,invoke,type Completion,type AssertionContext} from "../completion.ts";
import {record,dataArray} from "../data.ts";
import {createDomainRuntime} from "../domain.ts";
import {resourceStateFailure} from "../failure.ts";
import {normalizedPath,requestSnapshot,abandonRequest,nativeResponse,isUpgraded,UPGRADED_RESPONSE} from "./http.ts";
const origin=Object.freeze({source:"can:router",start:0,end:0,invocation:Object.freeze([])});
export type MountedCallback=(request:unknown,context?:AssertionContext)=>Promise<Completion<unknown>>;
type Method="GET"|"POST"|"PUT"|"PATCH"|"DELETE"|"OPTIONS"|"HEAD";
type Route=Readonly<{method:Method;source:string;path:string;stream:boolean;callback:MountedCallback}>;
type Router=ReadonlyMap<string,ReadonlyMap<string,Route>>;
const routes=new WeakMap<object,Route>(),routers=new WeakMap<object,Router>();
const object=(value:unknown):value is object=>value!==null&&(typeof value==="object"||typeof value==="function");
function read<T>(map:WeakMap<object,T>,value:unknown):T{if(!object(value)||!map.has(value))throw resourceStateFailure(undefined,origin);return map.get(value)!;}
function opaque<T>(map:WeakMap<object,T>,data:T):unknown{const token=Object.freeze(Object.create(null));map.set(token,data);return token;}
export function isRouterValue(kind:string|undefined,value:unknown):boolean{const map=kind==="route"?routes:kind==="router"?routers:undefined;return map!==undefined&&object(value)&&map.has(value);}
function fixed(status:number,allow?:string):Response{
 const headers=new Headers({"content-type":"text/plain; charset=utf-8","x-content-type-options":"nosniff"});if(allow)headers.set("allow",allow);
 return new Response(status===404?"Not Found":"Method Not Allowed",{status,headers});
}
export async function dispatch(router:unknown,request:unknown,context?:AssertionContext):Promise<Completion<Response>>{
 const table=read(routers,router),snapshot=requestSnapshot(request),methods=table.get(snapshot.path);
 try{
  if(!methods)return success(fixed(404));
  const route=methods.get(snapshot.method);if(!route)return success(fixed(405,[...methods.keys()].sort().join(", ")));
  const completed=await invoke(()=>route.callback(request,context),origin);if(completed.kind!=="ok")return completed;
  // Upgraded requests hold a live socket; the server answers no HTTP reply.
  if(isUpgraded(request))return success(UPGRADED_RESPONSE);
  return success(nativeResponse(completed.value,snapshot.method==="HEAD"));
 }finally{await abandonRequest(request);}
}
// Ingress peek: the server consults the table before snapshotting so
// stream-marked routes skip the eager bounded pre-read. Anything the
// peek cannot prove falls back to buffered ingress downstream.
export function routeKind(router:unknown,method:string,path:string):"stream"|"buffered"|undefined{
 const table=read(routers,router),methods=table.get(path),route=methods?.get(method);
 return route===undefined?undefined:route.stream?"stream":"buffered";
}
export function createRouter(domain:ReturnType<typeof createDomainRuntime>,types:Readonly<{invalid:string;duplicate:string;ambiguous:string}>){
 const error=(type:string,fields:readonly(readonly[string,unknown])[])=>failure(domain.create(type,record(type,fields),origin));
 function route(method:Method,source:string,callback:MountedCallback):Completion<unknown>{
  // Source checking additionally requires a static path and named exact callback.
  if(!source.isWellFormed()||!source.startsWith("/")||source.startsWith("//")||/[\x00-\x20\x7f\\?#*]/.test(source)||source.split("/").some(segment=>segment.startsWith(":")))return error(types.invalid,[["reason","path"]]);
  let path:string;try{path=normalizedPath(new URL(source,"http://can.invalid"));}catch(cause){if(!(cause instanceof URIError)&&!(cause instanceof TypeError))throw cause;return error(types.invalid,[["reason","path"]]);}
  if(path==="/__can"||path.startsWith("/__can/"))return error(types.invalid,[["reason","path"]]);
  if(typeof callback!=="function")throw new TypeError("invalid mounted callback");
  return success(opaque(routes,Object.freeze({method,source,path,stream:false,callback})));
 }
 return Object.freeze({
  async get(path:string,callback:MountedCallback,_context?:AssertionContext):Promise<Completion<unknown>>{return route("GET",path,callback);},
  async post(path:string,callback:MountedCallback,_context?:AssertionContext):Promise<Completion<unknown>>{return route("POST",path,callback);},
  async put(path:string,callback:MountedCallback,_context?:AssertionContext):Promise<Completion<unknown>>{return route("PUT",path,callback);},
  async patch(path:string,callback:MountedCallback,_context?:AssertionContext):Promise<Completion<unknown>>{return route("PATCH",path,callback);},
  async delete(path:string,callback:MountedCallback,_context?:AssertionContext):Promise<Completion<unknown>>{return route("DELETE",path,callback);},
  async options(path:string,callback:MountedCallback,_context?:AssertionContext):Promise<Completion<unknown>>{return route("OPTIONS",path,callback);},
  async head(path:string,callback:MountedCallback,_context?:AssertionContext):Promise<Completion<unknown>>{return route("HEAD",path,callback);},
  async stream(input:unknown,_context?:AssertionContext):Promise<Completion<unknown>>{
   const route=read(routes,input);
   return success(opaque(routes,Object.freeze({...route,stream:true})));
  },
  async make(input:unknown,_context?:AssertionContext):Promise<Completion<unknown>>{
   const table=new Map<string,Map<string,Route>>();
   for(const token of dataArray(input)){
    const route=read(routes,token);let methods=table.get(route.path);if(!methods){methods=new Map();table.set(route.path,methods);}
    const previous=methods.get(route.method);
    if(previous)return previous.source===route.source?error(types.duplicate,[["method",route.method],["path",route.path]]):error(types.ambiguous,[["first",previous.source],["second",route.source]]);
    methods.set(route.method,route);
   }
   return success(opaque(routers,table));
  }
 });
}
