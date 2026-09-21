import {contextOwner,fixtureIndex,registerFixtureTable,rawProviderEvidence,violation,type AssertionContext} from "./context.ts";
import type {FailureOrigin} from "../failure.ts";

// Compiler conformance data only: no authored Can constructor or public input
// accepts these rows. Bytes are copied at registration and at consumption.
export type RawHTTPFixture=Readonly<{
 request:Readonly<{method:string;url:string;headers:readonly (readonly [string,string])[];body:Uint8Array}>;
 response:Readonly<{status:number;headers:readonly (readonly [string,string])[];body:Uint8Array}>;
}>;
export type HTTPExchange=(url:URL,init:RequestInit)=>Promise<Response>;
const fixtures=new WeakMap<object,readonly RawHTTPFixture[]>();
const origin:FailureOrigin=Object.freeze({source:"can:raw-provider",start:0,end:0,invocation:Object.freeze([])});
export function provideHTTP(context:AssertionContext,rows:readonly RawHTTPFixture[]):void{
 if(fixtures.has(contextOwner(context)))throw violation(context,"malformed fixture",origin);
 const headers=(entries:readonly (readonly [string,string])[])=>Object.freeze(entries.map(([name,value])=>Object.freeze([name,value] as const)));
 let copied:readonly RawHTTPFixture[];
 try {
 copied=Object.freeze(rows.map(row=>Object.freeze({
  request:Object.freeze({...row.request,headers:headers(row.request.headers),body:new Uint8Array(row.request.body)}),
  response:Object.freeze({...row.response,headers:headers(row.response.headers),body:new Uint8Array(row.response.body)}),
 })));
 for(const row of copied){
  new Headers(row.request.headers.map(([name,value])=>[name,value]));
  fixtureResponse(row.response);
 }
 }catch{throw violation(context,"malformed fixture",origin);}
 fixtures.set(contextOwner(context),copied);
 registerFixtureTable(context,"can:raw-http",rows.length,origin);
}
export function providerHTTP(context:AssertionContext|undefined,where:FailureOrigin):HTTPExchange|undefined{
 if(context===undefined)return undefined;
 const rows=fixtures.get(contextOwner(context));if(rows===undefined)return undefined;
 return async(url,init)=>{
  const row=rows[fixtureIndex(context,"can:raw-http",rows.length,where)];
  const actualHeaders=Array.from(new Headers(init.headers).entries());
  const expectedHeaders=Array.from(new Headers(row.request.headers.map(([name,value])=>[name,value])).entries());
  const body=init.body===undefined?new Uint8Array():init.body;
  if(init.method!==row.request.method||url.href!==row.request.url||JSON.stringify(actualHeaders)!==JSON.stringify(expectedHeaders)||!(body instanceof Uint8Array)||body.length!==row.request.body.length||!body.every((byte,i)=>byte===row.request.body[i]))throw violation(context,"argument mismatch",where);
  const response=fixtureResponse(row.response);
  rawProviderEvidence(context);
  return response;
 };
}

function fixtureResponse(response:RawHTTPFixture["response"]):Response{
 if(!Number.isInteger(response.status)||response.status<200||response.status>599)throw new TypeError("invalid fixture status");
 if([204,205,304].includes(response.status)&&response.body.length!==0)throw new TypeError("invalid fixture body");
 return new Response([204,205,304].includes(response.status)&&response.body.length===0?null:new Uint8Array(response.body),{status:response.status,headers:response.headers.map(([name,value])=>[name,value])});
}
