import {test,expect} from "bun:test";
import {createHash} from "node:crypto";
import {catalogue} from "../catalogue.ts";
import {createDomainRuntime,domainFailureDiagnostics,type FailureShape} from "../domain.ts";
import {type HTTPTypes} from "../transport/http.ts";
import {runOwnedRoot} from "../owner.ts";
import {success,type Completion} from "../completion.ts";
const hash=(key:unknown)=>createHash("sha256").update("can-concrete-type-v1\0"+JSON.stringify(key)).digest("hex");
const shape=(kind:string,declaration:string,fields:{name:string;type:string}[]=[]):FailureShape=>({identity:hash([kind,declaration]),kind,declaration,fields,arguments:[],leaves:[],inputs:[],errors:[]});
const str=shape("primitive","str"),int=shape("primitive","int");
const header=shape("record","can.std.http@1::header",[{name:"name",type:str.identity},{name:"value",type:str.identity}]);
const headers:FailureShape={...shape("array",""),identity:hash(["array","",header.identity]),element:header.identity};
const declarations=catalogue.errors.filter(e=>e.id>=1100&&e.id<=1105||e.id===1110);
const errors=declarations.map(d=>shape("error",d.identity,d.fields.map(f=>({name:f.name,type:f.type==="str"?str.identity:f.type==="int"?int.identity:headers.identity}))));
const domain=createDomainRuntime({declarations:declarations.map(d=>({...d,parameters:0})),shapes:[str,int,header,headers,...errors]});
const types=Object.fromEntries(["invalid","credential","transport","timeout","limit","status"].map((name,i)=>[name,errors[i].identity])) as unknown as HTTPTypes;

const origin={source:"test:http",start:0,end:0,invocation:[]};
const connection={endpoint:"http://127.0.0.1:1/",timeoutMilliseconds:100,maxBodyBytes:3,headers:[]};
const request={path:"/",method:"GET" as const,query:[],headers:[]};
function check(result:Completion,id:number,payload:object){expect(result.kind).toBe("domain");if(result.kind!=="domain")throw new Error("expected domain");const d=domainFailureDiagnostics(result.value);expect(d.declaration.id).toBe(id);expect(d.payload).toMatchObject(payload);}
import {createNamedFetch} from "../transport/named.ts";
import {ownBytes,copyBytes} from "../bytes.ts";
import {record} from "../data.ts";
import {value} from "../completion.ts";
import {responseMedia,jsonRequestMedia} from "../transport/media.ts";
const responseSchema={root:"sample",nodes:[{identity:"sample",kind:"record",name:"sample",fields:[{name:"count",type:"int"}]},{identity:"int",kind:"primitive",name:"int"}]};
const ids={...types,header:header.identity,invalidData:errors[6].identity};
test("named fetch validates strict media types and charset without ambiguous parameters",()=>{
 for(const media of ["application/json","Application/JSON; Charset=\"UTF-8\"","application/problem+json; profile=\"a;b\""]){expect(()=>responseMedia(media,true)).not.toThrow();}
 for(const media of [undefined,"text/plain","application/json, application/json","application/json;","application/json; charset=utf-8; charset=utf-8","application/json; x", "application/json; charset=\"unterminated"]){expect(()=>responseMedia(media,true)).toThrow();}
 expect(()=>responseMedia("text/plain; charset=latin1",false)).toThrow("charset");
 expect(()=>responseMedia(undefined,false)).not.toThrow();
 expect(jsonRequestMedia("application/json; charset=UTF-8")).toBe(true);
 for(const media of ["application/problem+json","application/json; profile=x","application/json; charset=latin1"]){expect(jsonRequestMedia(media)).toBe(false);}
});
test("named fetch loopback decodes exact JSON, text BOM, bytes and immutable envelopes",async()=>{
 const server=Bun.serve({hostname:"127.0.0.1",port:0,fetch(req){
  const path=new URL(req.url).pathname;
  if(path==="/json")return new Response('{"count":9007199254740993}',{headers:{"content-type":"application/problem+json"}});
  if(path==="/text")return new Response(new Uint8Array([239,187,191,65]),{headers:{"content-type":"text/plain; charset=UTF-8"}});
  const headers=new Headers([["content-type","application/octet-stream"],["set-cookie","a=1"],["set-cookie","b=2"],["x-repeat","a"],["x-repeat","b"]]);
  return new Response(new Uint8Array([0,255,1]),{status:418,headers});
 }});
 try{const root=await runOwnedRoot(async()=>{
  const api=createNamedFetch(domain,ids,()=>undefined),c={...connection,endpoint:server.url.href,maxBodyBytes:1024};
  const json=value(await api.request<any>(c,{...request,path:"/json"},undefined,{mode:"json",schema:responseSchema},origin));expect(json.count).toBe(9007199254740993n);expect(Object.isFrozen(json)).toBe(true);
  expect(value(await api.request<string>(c,{...request,path:"/text"},undefined,{mode:"text"},origin))).toBe("\ufeffA");
  const envelope=value(await api.request<any>(c,request,undefined,{mode:"bytes",envelope:"response-bytes"},origin));
  expect(envelope.status).toBe(418n);expect([...copyBytes(envelope.body,origin)]).toEqual([0,255,1]);expect(Object.isFrozen(envelope)).toBe(true);expect(Object.isFrozen(envelope.headers)).toBe(true);
  expect(envelope.headers.filter((h:any)=>h.name==="set-cookie").map((h:any)=>h.value)).toEqual(["a=1","b=2"]);
  expect(envelope.headers.find((h:any)=>h.name==="x-repeat").value).toBe("a, b");
  check(await api.request(c,request,undefined,{mode:"bytes"},origin),1105,{status:418n});
  return success(undefined);
 });expect(root.completion.kind).toBe("ok");}finally{server.stop(true);}
});
test("named fetch encodes explicit bodies and captures credentials once before launch",async()=>{
 const received:{method:string;body:number[];type:string|null;auth:string|null;query:string[]}[]=[];
 const server=Bun.serve({hostname:"127.0.0.1",port:0,async fetch(req){received.push({method:req.method,body:[...new Uint8Array(await req.arrayBuffer())],type:req.headers.get("content-type"),auth:req.headers.get("authorization"),query:new URL(req.url).searchParams.getAll("q")});return new Response("ok");}});
 try{const root=await runOwnedRoot(async()=>{
  let reads=0;const api=createNamedFetch(domain,ids,()=>{reads++;return "token"}),c={...connection,endpoint:server.url.href,maxBodyBytes:1024,bearerEnvironment:"TOKEN"};
  for(const body of [{mode:"json" as const,value:record("sample",[["count",9007199254740993n]]),schema:responseSchema},{mode:"text" as const,value:"😀"},{mode:"bytes" as const,value:ownBytes(new Uint8Array([0,255]))}])expect(value(await api.request<string>(c,{...request,method:"POST",query:[{name:"q",value:["a b","+"]}]},body,{mode:"text"},origin))).toBe("ok");
  expect(reads).toBe(3);expect(received.map(r=>r.type)).toEqual(["application/json","text/plain; charset=utf-8","application/octet-stream"]);
  expect(new TextDecoder().decode(new Uint8Array(received[0].body))).toBe('{"count":9007199254740993}');expect(received[1].body).toEqual([...new TextEncoder().encode("😀")]);expect(received[2].body).toEqual([0,255]);
  expect(received.every(r=>r.auth==="Bearer token"&&r.query.join("|")==="a b|+")).toBe(true);
  return success(undefined);
 });expect(root.completion.kind).toBe("ok");}finally{server.stop(true);}
});
test("named fetch maps invalid bodies, media and malformed replies to exact errors",async()=>{
 let launches=0;
 const server=Bun.serve({hostname:"127.0.0.1",port:0,fetch(req){launches++;const path=new URL(req.url).pathname;
  if(path==="/utf8")return new Response(new Uint8Array([255]),{headers:{"content-type":"text/plain"}});
  if(path==="/charset")return new Response("x",{headers:{"content-type":"text/plain; charset=latin1"}});
  if(path==="/duplicate")return new Response('{"count":1,"count":2}',{headers:{"content-type":"application/json"}});
  return new Response("",{headers:{"content-type":path==="/empty"?"application/json":"text/plain"}});
 }});
 try{const root=await runOwnedRoot(async()=>{
  const api=createNamedFetch(domain,ids,()=>undefined),c={...connection,endpoint:server.url.href,maxBodyBytes:1024};
  for(const [path,mode,reason] of [["/utf8","text","utf8"],["/charset","text","charset"],["/empty","json","invalid_json"],["/wrong","json","media_type"],["/duplicate","json","duplicate_member"]] as const)check(await api.request(c,{...request,path},undefined,{mode,schema:responseSchema},origin),1110,{reason});
  const before=launches;
  check(await api.request(c,{...request,method:"POST",headers:[{name:"content_type",value:"text/plain"}]},{mode:"json",value:record("sample",[["count",1n]]),schema:responseSchema},{mode:"text"},origin),1100,{reason:"content_type"});
  check(await api.request(c,{...request,method:"POST"},{mode:"text",value:"\ud800"},{mode:"text"},origin),1110,{reason:"unicode_scalar"});
  check(await api.request({...c,maxBodyBytes:1},{...request,method:"POST"},{mode:"text",value:"é"},{mode:"text"},origin),1104,{limit:1n});
  expect(launches).toBe(before);
  return success(undefined);
 });expect(root.completion.kind).toBe("ok");}finally{server.stop(true);}
});
import {assertionContext,contextReport} from "../assert/context.ts";
import {provideHTTP} from "../assert/provider.ts";
import {runAssertion} from "../assert/runner.ts";
test("named fetch raw fixtures handle bodyless requests and unprovided assertions deny authentication",async()=>{
 let reads=0;const api=createNamedFetch(domain,ids,()=>{reads++;return "token"});
 const context=assertionContext({package:"test",declaration:"fetch",name:"unprovided"});
 const root=await runOwnedRoot(()=>api.request({...connection,bearerEnvironment:"TOKEN"},request,undefined,{mode:"text"},origin,context));
 expect(root.completion.kind).toBe("standard");expect(reads).toBe(0);expect(contextReport(context).violations).toEqual(["missing fixture"]);
 const c={...connection,endpoint:"https://fixture.invalid/",maxBodyBytes:1024};
 for(const status of [200,299,300,418,599]){
  const report=await runAssertion({root:{package:"test",declaration:"fetch",name:"raw-"+status},actual:async context=>{
   provideHTTP(context,[{request:{method:"GET",url:c.endpoint,headers:[],body:new Uint8Array()},response:{status,headers:[["content-type","text/plain"]],body:new TextEncoder().encode("ok")}}]);
   const result=await api.request<any>(c,request,undefined,{mode:"text",envelope:"text-response"},origin,context);
   if(result.kind!=="ok")return result;return success([result.value.status,result.value.body]);
  },expected:async()=>success([BigInt(status),"ok"])});
  expect(report.passed).toBe(true);expect(report.evidence).toContain("raw-provider-fixture");
 }
});
test("named fetch body Content-Type defaults and overrides preserve explicit encodings",async()=>{
 const api=createNamedFetch(domain,ids,()=>undefined);const c={...connection,maxBodyBytes:1024};let last:RequestInit|undefined;
 const exchange=async(_url:URL,init:RequestInit)=>{last=init;return new Response(new Uint8Array([255]),{headers:{"content-type":"bad; charset=latin1"}})};
 const root=await runOwnedRoot(async()=>{
  for(const body of [{mode:"text" as const,value:"é"},{mode:"bytes" as const,value:ownBytes(new Uint8Array([255]))}]){
   const result=await api.request(c,{...request,method:"POST",headers:[{name:"content_type",value:"custom/example; charset=latin1"}],exchange},body,{mode:"bytes"},origin);
   expect(result.kind).toBe("ok");expect(new Headers(last!.headers).get("content-type")).toBe("custom/example; charset=latin1");
   expect([...(last!.body as Uint8Array)]).toEqual(body.mode==="text"?[195,169]:[255]);
  }
  const body={mode:"json" as const,value:record("sample",[["count",1n]]),schema:responseSchema};
  expect((await api.request({...c,headers:[{name:"content_type",value:"bad/type"}]},{...request,method:"POST",headers:[{name:"content_type",value:[]}],exchange},body,{mode:"bytes"},origin)).kind).toBe("ok");
  expect(new Headers(last!.headers).get("content-type")).toBe("application/json");
  check(await api.request({...c,maxBodyBytes:1},{...request,method:"POST",exchange},body,{mode:"bytes"},origin),1104,{limit:1n});
  return success(undefined);
 });expect(root.completion.kind).toBe("ok");
 for(const bad of ["application/json; x=\"a\u0001b\"","application/json\u00a0; charset=utf-8"])expect(()=>responseMedia(bad,true)).toThrow("media_type");
});
