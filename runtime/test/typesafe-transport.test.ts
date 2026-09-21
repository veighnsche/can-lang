import {test,expect} from "bun:test";
import {createHash} from "node:crypto";
import {catalogue} from "../catalogue.ts";
import {createDomainRuntime,domainFailureDiagnostics,type FailureShape} from "../domain.ts";
import {createTypeSafe,type AITypes,type NoulDescriptor} from "../ai/typesafe.ts";
import {runOwnedRoot} from "../owner.ts";
import {success,failure,type Completion} from "../completion.ts";
import {record} from "../data.ts";
import {assertionContext,contextReport} from "../assert/context.ts";
import {provideHTTP,type RawHTTPFixture} from "../assert/provider.ts";
import {runAssertion} from "../assert/runner.ts";
const hash=(key:unknown)=>createHash("sha256").update("can-concrete-type-v1\0"+JSON.stringify(key)).digest("hex");
const shape=(kind:string,declaration:string,fields:{name:string;type:string}[]=[]):FailureShape=>({identity:hash([kind,declaration]),kind,declaration,fields,arguments:[],leaves:[],inputs:[],errors:[]});
const str=shape("primitive","str"),int=shape("primitive","int");
const header=shape("record","can.std.http@1::header",[{name:"name",type:str.identity},{name:"value",type:str.identity}]);
const headers:FailureShape={...shape("array",""),identity:hash(["array","",header.identity]),element:header.identity};
const declarations=catalogue.errors.filter(e=>e.id>=1100&&e.id<=1105||e.id===1110||e.id===1120||e.id===1121);
const errors=declarations.map(d=>shape("error",d.identity,d.fields.map(f=>({name:f.name,type:f.type==="str"?str.identity:f.type==="int"?int.identity:headers.identity}))));
const domain=createDomainRuntime({declarations:declarations.map(d=>({...d,parameters:0})),shapes:[str,int,header,headers,...errors]});
const types=Object.fromEntries(["invalid","credential","transport","timeout","limit","status"].map((name,i)=>[name,errors[i].identity])) as unknown as AITypes;

const aiTypes={...types,header:header.identity,invalidData:errors[6].identity,invalidQuestion:errors[7].identity,invalidAnswer:errors[8].identity};
const origin={source:"test:typesafe-transport",start:0,end:0,invocation:[]};
const schema={root:"state",nodes:[{identity:"state",kind:"record",name:"state",fields:[{name:"amount",type:"int"}]},{identity:"int",kind:"primitive",name:"int"}]};
const state=record("state",[["amount",9007199254740993n]]);
const question:NoulDescriptor={instructions:"Check amount",trueDescription:"Yes",falseDescription:"No",minimum:0.5};
const connection={endpoint:"http://127.0.0.1:1/systemone",timeoutMilliseconds:1000,maxBodyBytes:8192,headers:[],bearerEnvironment:"TOKEN"};
function check(result:Completion,id:number,payload:object){expect(result.kind).toBe("domain");if(result.kind!=="domain")throw Error("expected domain");const d=domainFailureDiagnostics(result.value);expect(d.declaration.id).toBe(id);expect(d.payload).toMatchObject(payload);}

test("Noul sends one exact POST and reads credentials only after all input admission",async()=>{
 let reads=0,requests=0,body="",authorization="",contentType="",accept="";
 const api=createTypeSafe(domain,aiTypes,()=>{reads++;return "fixture-secret";});
 const server=Bun.serve({hostname:"127.0.0.1",port:0,async fetch(request){
  requests++;expect(request.method).toBe("POST");expect(new URL(request.url).pathname).toBe("/systemone");
  body=await request.text();authorization=request.headers.get("authorization")!;contentType=request.headers.get("content-type")!;accept=request.headers.get("accept")!;
  return Response.json({model:"resolved",answers:{q0:{type:"noul",noul:0.5},q1:{type:"noul",noul:0.25}}});
 }});
 try{const root=await runOwnedRoot(async()=>{
  const c={...connection,endpoint:new URL("/systemone",server.url).href};
  check(await api.noul(c,"jev-latest",schema,state,[question,{...question,instructions:""}],origin),1120,{reason:"instructions"});
  check(await api.noul({...c,maxBodyBytes:10},"jev-latest",schema,state,[question],origin),1104,{limit:10n});
  check(await api.noul(c,"jev-latest",schema,record("state",[["amount",1]]),[question],origin),1110,{path:"/state/amount",reason:"type"});
  expect(reads).toBe(0);expect(requests).toBe(0);
  const result=await api.noul(c,"jev-latest",schema,state,[question,question],origin);
  expect(result.kind).toBe("ok");if(result.kind!=="ok")throw Error("expected answers");
  expect(result.value).toEqual([0.5,0.25]);expect(reads).toBe(1);expect(requests).toBe(1);
  expect(authorization).toBe("Bearer fixture-secret");expect(contentType).toBe("application/json");expect(accept).toBe("application/json");
  expect(body).toBe('{"model":"jev-latest","state":{"amount":9007199254740993},"questions":{"q0":{"type":"noul","instructions":"Check amount","criteria":{"true":"Yes","false":"No"}},"q1":{"type":"noul","instructions":"Check amount","criteria":{"true":"Yes","false":"No"}}}}');
  return success(undefined);
 });expect(root.completion.kind).toBe("ok");}finally{server.stop(true);}
});

test("Noul malformed responses stay distinct from status failures without retry",async()=>{
 let requests=0,reply='{',status=200;
 const api=createTypeSafe(domain,aiTypes,()=>"fixture-secret");
 const server=Bun.serve({hostname:"127.0.0.1",port:0,fetch(){requests++;return new Response(reply,{status});}});
 try{const root=await runOwnedRoot(async()=>{
  const c={...connection,endpoint:server.url.href};
  check(await api.noul(c,"jev-latest",schema,state,[question],origin),1110,{reason:"invalid_json"});
  reply='{"model":"resolved","answers":{}}';
  check(await api.noul(c,"jev-latest",schema,state,[question],origin),1121,{question:"",reason:"question_ids"});
  reply='{"model":"resolved","answers":{"q0":{"type":"noul","noul":0.5},"q1":{"type":"noul","noul":2}}}';
  check(await api.noul(c,"jev-latest",schema,state,[question,question],origin),1121,{question:"q1",reason:"probability"});
  for(const code of [401,422,429,529]){status=code;check(await api.noul(c,"jev-latest",schema,state,[question],origin),1105,{status:BigInt(code)});}
  expect(requests).toBe(7);
  return success(undefined);
 });expect(root.completion.kind).toBe("ok");}finally{server.stop(true);}
});

test("assertion context refuses a live Noul boundary before authentication",async()=>{
 let reads=0;const api=createTypeSafe(domain,aiTypes,()=>{reads++;return "secret";});
 const context=assertionContext({package:"app",declaration:"judge",name:"unprovided"});
 const root=await runOwnedRoot(()=>api.noul(connection,"jev-latest",schema,state,[question],origin,context));
 expect(root.completion.kind).toBe("standard");expect(reads).toBe(0);expect(contextReport(context).violations).toEqual(["missing fixture"]);
});

test("raw provider assertions cross request encoding, native response parsing and error mapping",async()=>{
 const encode=(text:string)=>new TextEncoder().encode(text);
 const c={...connection,endpoint:"https://fixture.invalid/systemone",bearerEnvironment:undefined};
 const api=createTypeSafe(domain,aiTypes,()=>{throw Error("raw fixture read ambient credential");});
 const row:RawHTTPFixture={request:{method:"POST",url:c.endpoint,headers:[["accept","application/json"],["content-type","application/json"]],body:encode('{"model":"jev-latest","state":{"amount":9007199254740993},"questions":{"q0":{"type":"noul","instructions":"Check amount","criteria":{"true":"Yes","false":"No"}}}}')},response:{status:200,headers:[["content-type","application/json"]],body:encode('{"model":"resolved","answers":{"q0":{"type":"noul","noul":0.5}}}')}};
 for(const malformed of [false,true]){
  const report=await runAssertion({root:{package:"conformance",declaration:"noul",name:malformed?"raw-invalid":"raw-valid"},actual:async context=>{
   provideHTTP(context,[malformed?{...row,response:{...row.response,body:encode('{"model":"resolved","answers":{}}')}}:row]);
   return api.noul(c,"jev-latest",schema,state,[question],origin,context);
  },expected:async()=>malformed?failure(domain.create(aiTypes.invalidAnswer,record(aiTypes.invalidAnswer,[["question",""],["reason","question_ids"]]),origin)):success([0.5])});
  expect(report.passed).toBe(true);expect(report.evidence).toEqual(["raw-provider-fixture","real-can"]);
 }
 const mismatch=await runAssertion({root:{package:"conformance",declaration:"noul",name:"wrong-wire-input"},actual:async context=>{
  provideHTTP(context,[{...row,request:{...row.request,body:encode("{}")}}]);
  return api.noul(c,"jev-latest",schema,state,[question],origin,context);
 },expected:async()=>success([0.5])});
 expect(mismatch.passed).toBe(false);expect(mismatch.violations).toEqual(["argument mismatch"]);expect(mismatch.evidence).not.toContain("raw-provider-fixture");
 const leftover=await runAssertion({root:{package:"conformance",declaration:"noul",name:"unused-wire-input"},actual:async context=>{provideHTTP(context,[row]);return success(0);},expected:async()=>success(0)});
 expect(leftover.passed).toBe(false);expect(leftover.violations).toEqual(["unused fixture"]);
});
