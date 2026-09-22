import {test,expect} from "bun:test";
import {createHash} from "node:crypto";
import {catalogue} from "../catalogue.ts";
import {createDomainRuntime,domainFailureDiagnostics,isDomainFailure,type FailureShape} from "../domain.ts";
import {createTypeSafe,type AITypes,type NoulDescriptor} from "../ai/typesafe.ts";
import {runOwnedRoot} from "../owner.ts";
import {success,failure,type Completion} from "../completion.ts";
import {record,recordIdentity,dataProperty} from "../data.ts";
import {assertionContext,contextReport} from "../assert/context.ts";
import {provideHTTP,type RawHTTPFixture} from "../assert/provider.ts";
import {runAssertion} from "../assert/runner.ts";
const hash=(key:unknown)=>createHash("sha256").update("can-concrete-type-v1\0"+JSON.stringify(key)).digest("hex");
const shape=(kind:string,declaration:string,fields:{name:string;type:string}[]=[]):FailureShape=>({identity:hash([kind,declaration]),kind,declaration,fields,arguments:[],leaves:[],inputs:[],errors:[]});
const str=shape("primitive","str"),int=shape("primitive","int");
const header=shape("record","can.std.http@1::header",[{name:"name",type:str.identity},{name:"value",type:str.identity}]);
const headers:FailureShape={...shape("array",""),identity:hash(["array","",header.identity]),element:header.identity};
const declarations=catalogue.errors.filter(e=>e.id>=1100&&e.id<=1106||e.id===1110||e.id===1120||e.id===1121);
const detailIdentity=hash(["variant","can.std.http@1::failure_detail"]);
const errors=declarations.map(d=>shape("error",d.identity,d.fields.map(f=>({name:f.name,type:f.type==="str"?str.identity:f.type==="int"?int.identity:f.type==="http::failure_detail"?detailIdentity:headers.identity}))));
const detail:FailureShape={identity:detailIdentity,kind:"variant",declaration:"can.std.http@1::failure_detail",fields:[],arguments:[],leaves:errors.filter((_,i)=>declarations[i].id!==1106&&declarations[i].id<1120).map(e=>e.identity),inputs:[],errors:[]};
const domain=createDomainRuntime({declarations:declarations.map(d=>({...d,parameters:0})),shapes:[str,int,header,headers,...errors,detail]});
const types=Object.fromEntries(["invalid","credential","transport","timeout","limit","status"].map((name,i)=>[name,errors[i].identity])) as unknown as AITypes;

const aiTypes={...types,header:header.identity,invalidData:errors[7].identity,invalidQuestion:errors[8].identity,invalidAnswer:errors[9].identity,failed:errors[6].identity};
const origin={source:"test:typesafe-transport",start:0,end:0,invocation:[]};
const schema={root:"state",nodes:[{identity:"state",kind:"record",name:"state",fields:[{name:"amount",type:"int"}]},{identity:"int",kind:"primitive",name:"int"}]};
const state=record("state",[["amount",9007199254740993n]]);
const question:NoulDescriptor={kind:"noul",instructions:"Check amount",trueDescription:"Yes",falseDescription:"No",minimum:0.5};
const connection={endpoint:"http://127.0.0.1:1/systemone",timeoutMilliseconds:1000,maxBodyBytes:8192,headers:[],bearerEnvironment:"TOKEN"};
function check(result:Completion,id:number,payload:object){expect(result.kind).toBe("domain");if(result.kind!=="domain")throw Error("expected domain");const d=domainFailureDiagnostics(result.value);expect(d.declaration.id).toBe(id);expect(d.payload).toMatchObject(payload);}
function checkDetail(result:Completion,leaf:string,payload:object){expect(result.kind).toBe("domain");if(result.kind!=="domain")throw Error("expected domain");const d=domainFailureDiagnostics(result.value);expect(d.declaration.id).toBe(1106);expect(d.provenance.boundary).toBe("native");const detail=dataProperty(d.payload,"detail");expect(recordIdentity(detail)).toBe(leaf);expect(detail).toMatchObject(payload);if(!isDomainFailure(d.cause))throw new Error("expected private original cause");}

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
  check(await api.ask(c,"jev-latest",schema,state,[question,{...question,instructions:""}],origin,"test:ai/judge"),1120,{reason:"instructions"});
  checkDetail(await api.ask({...c,maxBodyBytes:10},"jev-latest",schema,state,[question],origin,"test:ai/judge"),errors[4].identity,{limit:10n});
  checkDetail(await api.ask(c,"jev-latest",schema,record("state",[["amount",1]]),[question],origin,"test:ai/judge"),errors[7].identity,{path:"/state/amount",reason:"type"});
  expect(reads).toBe(0);expect(requests).toBe(0);
  const result=await api.ask(c,"jev-latest",schema,state,[question,question],origin,"test:ai/judge");
  expect(result.kind).toBe("ok");if(result.kind!=="ok")throw Error("expected answers");
  expect(result.value).toEqual([{kind:"noul",probability:0.5},{kind:"noul",probability:0.25}]);expect(reads).toBe(1);expect(requests).toBe(1);
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
  checkDetail(await api.ask(c,"jev-latest",schema,state,[question],origin,"test:ai/judge"),errors[7].identity,{reason:"invalid_json"});
  reply='{"model":"resolved","answers":{}}';
  check(await api.ask(c,"jev-latest",schema,state,[question],origin,"test:ai/judge"),1121,{question:"",reason:"question_ids"});
  reply='{"model":"resolved","answers":{"q0":{"type":"noul","noul":0.5},"q1":{"type":"noul","noul":2}}}';
  check(await api.ask(c,"jev-latest",schema,state,[question,question],origin,"test:ai/judge"),1121,{question:"q1",reason:"probability"});
  for(const code of [401,422,429,529]){status=code;checkDetail(await api.ask(c,"jev-latest",schema,state,[question],origin,"test:ai/judge"),errors[5].identity,{status:BigInt(code)});}
  expect(requests).toBe(7);
  return success(undefined);
 });expect(root.completion.kind).toBe("ok");}finally{server.stop(true);}
});

test("assertion context refuses a live Noul boundary before authentication",async()=>{
 let reads=0;const api=createTypeSafe(domain,aiTypes,()=>{reads++;return "secret";});
 const context=assertionContext({package:"app",declaration:"judge",name:"unprovided"});
 const root=await runOwnedRoot(()=>api.ask(connection,"jev-latest",schema,state,[question],origin,"test:ai/judge",context));
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
   const result=await api.ask(c,"jev-latest",schema,state,[question],origin,"test:ai/judge",context);if(result.kind!=="ok")return result;return success(result.value.map(answer=>{if(answer.kind!=="noul")throw Error("wrong answer kind");return answer.probability;}));
  },expected:async()=>malformed?failure(domain.create(aiTypes.invalidAnswer,record(aiTypes.invalidAnswer,[["question",""],["reason","question_ids"]]),origin)):success([0.5])});
  expect(report.passed).toBe(true);expect(report.evidence).toEqual(["raw-provider-fixture","real-can"]);
 }
 const mismatch=await runAssertion({root:{package:"conformance",declaration:"noul",name:"wrong-wire-input"},actual:async context=>{
  provideHTTP(context,[{...row,request:{...row.request,body:encode("{}")}}]);
  const result=await api.ask(c,"jev-latest",schema,state,[question],origin,"test:ai/judge",context);if(result.kind!=="ok")return result;return success(result.value.map(answer=>{if(answer.kind!=="noul")throw Error("wrong answer kind");return answer.probability;}));
 },expected:async()=>success([0.5])});
 expect(mismatch.passed).toBe(false);expect(mismatch.violations).toEqual(["argument mismatch"]);expect(mismatch.evidence).not.toContain("raw-provider-fixture");
 const leftover=await runAssertion({root:{package:"conformance",declaration:"noul",name:"unused-wire-input"},actual:async context=>{provideHTTP(context,[row]);return success(0);},expected:async()=>success(0)});
 expect(leftover.passed).toBe(false);expect(leftover.violations).toEqual(["unused fixture"]);
});


test("invalid raw HTTP configuration stays a harness failure after handling",async()=>{
 const body=new Uint8Array([123]);
 for(const response of [
  {status:999,headers:[],body},
  {status:200,headers:[["bad\nname","value"]],body},
  ...[204,205,304].map(status=>({status,headers:[],body})),
 ]){
  let registered=false;
  const report=await runAssertion({root:{package:"conformance",declaration:"noul",name:"bad-configuration"},actual:async context=>{
   try{provideHTTP(context,[{request:{method:"POST",url:"https://fixture.invalid/",headers:[],body},response:response as unknown as RawHTTPFixture["response"]}]);registered=true;}catch{/* Authored recovery cannot clear the harness violation. */}
   return success(0);
  },expected:async()=>success(0)});
  expect(registered).toBe(false);
  expect(report.passed).toBe(false);
  expect(report.violations).toEqual(["malformed fixture"]);
  expect(report.evidence).not.toContain("raw-provider-fixture");
 }
});
