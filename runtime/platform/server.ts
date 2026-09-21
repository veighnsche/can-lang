import {success,failure,type Completion,type AssertionContext} from "../completion.ts";
import {denyLiveBoundary} from "../assert/context.ts";
import {record} from "../data.ts";
import {createDomainRuntime} from "../domain.ts";
import {resourceStateFailure} from "../failure.ts";
import {registerResource,useResource,closeResource,guardCallback,withScope,type Resource,type Scope} from "../owner.ts";
import {snapshotRequest} from "./http.ts";
import {dispatch,isRouterValue} from "./router.ts";
const origin=Object.freeze({source:"can:server",start:0,end:0,invocation:Object.freeze([])});
type Config=Readonly<{host:string;port:bigint;bodyLimit:number;shutdownMs:number}>;
type Native=Readonly<{server:Readonly<{stop:(closeActiveConnections?:boolean)=>void}>;scope:Scope;router:unknown;bodyLimit:number;shutdownMs:number;settled:Promise<Completion<void>>;settle:(completion:Completion<void>)=>void}>;
const configs=new WeakMap<object,Config>(),servers=new WeakMap<object,Native>();
const object=(value:unknown):value is object=>value!==null&&(typeof value==="object"||typeof value==="function");
export function isServerValue(kind:string|undefined,value:unknown):boolean{
 if(kind==="server_config")return object(value)&&configs.has(value);
 if(kind==="server")return object(value)&&servers.has(value);
 return false;
}
function fixed(status:400|413|500):Response{
 const headers=new Headers({"content-type":"text/plain; charset=utf-8","x-content-type-options":"nosniff"});
 return new Response(status===400?"Bad Request":status===413?"Payload Too Large":"Internal Server Error",{status,headers});
}
export function createServer(domain:ReturnType<typeof createDomainRuntime>,types:Readonly<{invalidConfig:string;bindFailed:string;shutdownFailed:string}>){
 const invalid=(reason:string)=>failure(domain.create(types.invalidConfig,record(types.invalidConfig,[["reason",reason]]),origin));
 const shutdown=(phase:string)=>failure(domain.create(types.shutdownFailed,record(types.shutdownFailed,[["phase",phase]]),origin));
 function readConfig(value:unknown):Config{if(!object(value)||!configs.has(value))throw resourceStateFailure(undefined,origin);return configs.get(value)!;}
 function readServer(value:unknown):Native{if(!object(value)||!servers.has(value))throw resourceStateFailure(undefined,origin);return servers.get(value)!;}
 return Object.freeze({
  async makeConfig(host:unknown,port:unknown,bodyLimit:unknown,shutdownMs:unknown,_context?:AssertionContext):Promise<Completion<unknown>>{
   if(typeof host!=="string")throw new TypeError("invalid compiler host");
   if(typeof port!=="bigint"||typeof bodyLimit!=="bigint"||typeof shutdownMs!=="bigint")throw new TypeError("invalid compiler config integer");
   if(bodyLimit<1n||bodyLimit>67108864n)return invalid("body_limit");
   if(shutdownMs<0n||shutdownMs>2147483647n)return invalid("shutdown_ms");
   const token=Object.freeze(Object.create(null));
   configs.set(token,Object.freeze({host,port,bodyLimit:Number(bodyLimit),shutdownMs:Number(shutdownMs)}));
   return success(token);
  },
  async start(config:unknown,router:unknown,context?:AssertionContext):Promise<Completion<unknown>>{
   denyLiveBoundary(context,origin);
   const {host,port,bodyLimit,shutdownMs}=readConfig(config);
   if(!isRouterValue("router",router))throw resourceStateFailure(undefined,origin);
   const address=host+":"+port.toString();
   const bind=()=>failure(domain.create(types.bindFailed,record(types.bindFailed,[["address",address]]),origin));
   let release!:()=>void;
   const lifetime=new Promise<void>(resolve=>{release=resolve;});
   // The live server resource belongs to the caller's scope so stop and wait
   // validate from user code; only fetch tasks live in the drainable child.
   let token!:Resource;
   let ready!:(value:Readonly<{server?:Native["server"];scope?:Scope;state?:boolean}>)=>void;
   const decided=new Promise<Readonly<{server?:Native["server"];scope?:Scope;state?:boolean}>>(resolve=>{ready=resolve;});
   const scoped=withScope(async (scope):Promise<Completion<undefined>>=>{
    const guarded=guardCallback(scope,async (native:Request):Promise<Completion<Response>>=>{
     return useResource(token,"server",async ()=>{
      const snapshot=await snapshotRequest(native,bodyLimit);
      if(snapshot.kind==="rejected")return success(fixed(snapshot.status));
      return dispatch(router,snapshot.value,context);
     });
    });
    try{
     const server=Bun.serve({hostname:host,port:Number(port),fetch:async (native:Request):Promise<Response>=>{
      try{const completed=await guarded(native);return completed.kind==="ok"?completed.value:fixed(500);}
      catch{return fixed(500);}
     },error:()=>fixed(500)});
     ready({server,scope});
     await lifetime;
    }catch{ready({});return success(undefined);}
    return success(undefined);
   });
   // withScope resolves its boxed outcome; a rejection here only reports an
   // entry violation (no owner context or a closing scope) without hanging.
   void scoped.then(()=>{},()=>{ready({state:true});});
   const outcome=await decided;
   if(outcome.state)throw resourceStateFailure(undefined,origin);
   if(outcome.server===undefined||outcome.scope===undefined)return bind();
   const server=outcome.server;
   let settle!:(completion:Completion<void>)=>void;
   const settled=new Promise<Completion<void>>(resolve=>{settle=resolve;});
   const native:Native=Object.freeze({server,scope:outcome.scope,router,bodyLimit,shutdownMs,settled,settle});
   token=registerResource("server",native,async ():Promise<Completion<void>>=>{
    let completion:Completion<void>;
    try{server.stop(false);completion=success(undefined);}
    catch{completion=shutdown("stop");}
    settle(completion);release();return completion;
   },{scopeManaged:true,shutdownMilliseconds:shutdownMs});
   servers.set(token,native);
   return success(token);
  },
  async stop(server:unknown,context?:AssertionContext):Promise<Completion<undefined>>{
   denyLiveBoundary(context,origin);
   const native=readServer(server);
   const completion=await closeResource(server,"server",{milliseconds:native.shutdownMs,failure:()=>shutdown("deadline")});
   return completion.kind==="ok"?success(undefined):completion;
  },
  async wait(server:unknown,context?:AssertionContext):Promise<Completion<undefined>>{
   denyLiveBoundary(context,origin);
   const native=readServer(server);
   // Signal delivery is context-free, so each firing re-enters ownership
   // through the server scope; a closed scope means the close already ran.
   const onSignal=()=>{
    try{const initiate=guardCallback(native.scope,async ()=>closeResource(server,"server"));void initiate().then(()=>{},()=>{});}
    catch{}
   };
   process.on("SIGINT",onSignal);process.on("SIGTERM",onSignal);
   try{const completion=await native.settled;return completion.kind==="ok"?success(undefined):completion;}
   finally{process.off("SIGINT",onSignal);process.off("SIGTERM",onSignal);}
  }
 });
}
