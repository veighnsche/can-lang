import {invoke,failure,type Completion} from "../completion.ts";
import {array,record} from "../data.ts";
import {createDomainRuntime} from "../domain.ts";
import type {FailureOrigin} from "../failure.ts";
import {transportProblem} from "./deadline.ts";
import {performRequest,type NativeRequest,type ResponseMetadata} from "./fetch.ts";
import type {Connection} from "./request.ts";

export type HTTPTypes=Readonly<{invalid:string;credential:string;transport:string;timeout:string;limit:string;status:string;header:string}>;
export function createTransport(domain:ReturnType<typeof createDomainRuntime>,types:HTTPTypes,readEnvironment:(name:string)=>string|undefined){
 return Object.freeze({async request<T>(connection:Connection,request:NativeRequest,decode:(bytes:Uint8Array,metadata:ResponseMetadata)=>Completion<T>|Promise<Completion<T>>,origin:FailureOrigin,operation:string):Promise<Completion<T>>{
  return invoke(async()=>{
   try{return await performRequest(connection,request,readEnvironment,decode);}
   catch(cause){
    const problem=transportProblem(cause);if(!problem)throw cause;
    let identity:string,fields:[string,unknown][];
    switch(problem.kind){
     case "invalid":identity=types.invalid;fields=[["reason",problem.reason]];break;
     case "credential":identity=types.credential;fields=[["variable",connection.bearerEnvironment!]];break;
     case "transport":identity=types.transport;fields=[["phase",problem.phase]];break;
     case "timeout":identity=types.timeout;fields=[["timeout_ms",BigInt(connection.timeoutMilliseconds)]];break;
     case "limit":identity=types.limit;fields=[["limit",BigInt(problem.limit)]];break;
     case "status":identity=types.status;fields=[["status",BigInt(problem.status)],["headers",array(problem.headers.map(h=>record(types.header,[["name",h.name],["value",h.value]])))]];break;
    }
    return failure(domain.create(identity,record(identity,fields),origin,undefined,{boundary:"native",operation}));
   }
  },origin);
 }});
}
