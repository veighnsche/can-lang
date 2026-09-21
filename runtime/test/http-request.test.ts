import {test,expect} from "bun:test";
import {createHash} from "node:crypto";
import {catalogue} from "../catalogue.ts";
import {createDomainRuntime,domainFailureDiagnostics,type FailureShape} from "../domain.ts";
import {createRequests,createResponses,snapshotRequest,isRequest,isHTTPValue,nativeResponse} from "../platform/http.ts";
import {copyBytes,ownBytes} from "../bytes.ts";
import {value,invoke,type Completion} from "../completion.ts";
import {record,array,dataProperty} from "../data.ts";
import {createRouter,dispatch,isRouterValue} from "../platform/router.ts";
const hash=(kind:string,name:string)=>createHash("sha256").update("can-concrete-type-v1\0"+JSON.stringify([kind,name])).digest("hex");
const shape=(kind:string,declaration:string,fields:{name:string;type:string}[]=[]):FailureShape=>({identity:hash(kind,declaration),kind,declaration,fields,arguments:[],leaves:[],inputs:[],errors:[]});
const str=shape("primitive","str"),int=shape("primitive","int");
const declarations=catalogue.errors.filter(e=>[1100,1104,1110,1230,1231,1232].includes(e.id));
const errors=declarations.map(e=>shape("error",e.identity,e.fields.map(f=>({name:f.name,type:f.type==="int"?int.identity:str.identity}))));
const domain=createDomainRuntime({declarations:declarations.map(e=>({...e,parameters:0})),shapes:[str,int,...errors]});
const api=createRequests(domain,{invalid:errors[0]!.identity,limit:errors[1]!.identity,invalidData:errors[2]!.identity,header:"header"});
const responses=createResponses(domain,{invalid:errors[0]!.identity,invalidData:errors[2]!.identity});
const routing=createRouter(domain,{invalid:errors[3]!.identity,duplicate:errors[4]!.identity,ambiguous:errors[5]!.identity});
const origin={source:"test",start:0,end:0,invocation:[]};
async function snapshot(url:string,init?:RequestInit,limit=1024){const result=await snapshotRequest(new Request(url,init),limit);if(result.kind!=="request")throw Error("rejected request");return result.value;}
function check(result:Completion,id:number,payload:object){expect(result.kind).toBe("domain");if(result.kind!=="domain")throw Error("expected domain");const d=domainFailureDiagnostics(result.value);expect(d.declaration.id).toBe(id);expect(d.payload).toMatchObject(payload);}
test("request URL normalization retains decoded path and query distinctions",async()=>{
 const request=await snapshot("http://localhost/a/../caf%C3%A9?x=+&x=%2B&empty=&escaped=%26%3D",{headers:{"X-Test":"value"}});
 expect(value(await api.method(request))).toBe("GET");expect(value(await api.path(request))).toBe("/café");
 expect(value(await api.queryAll(request,"x"))).toEqual([" ","+"]);
 expect(value(await api.queryOne(request,"empty"))).toBe("");expect(value(await api.queryOne(request,"escaped"))).toBe("&=");
 expect(value(await api.queryAll(request,"absent"))).toEqual([]);
 check(await api.queryOne(request,"absent"),1100,{reason:"query_missing"});check(await api.queryOne(request,"x"),1100,{reason:"query_repeated"});
 expect(value(await api.headers(request))).toEqual([record("header",[["name","x-test"],["value","value"]])]);
 expect(Object.isFrozen(request)).toBe(true);expect(isRequest(request)).toBe(true);
});
test("invalid query encodings fail without native replacement fallback",async()=>{
 for(const query of ["x=%","x=%GG","x=%FF","%ED%A0%80=x"]){const request=await snapshot("http://localhost/?"+query);check(await api.queryAll(request,"x"),1100,{reason:"query_value"});}
 const request=await snapshot("http://localhost/");check(await api.queryOne(request,"\ud800"),1100,{reason:"query_name"});
 for(const path of ["/%","/%FF","/%00","/%5c"]){expect(await snapshotRequest(new Request("http://localhost"+path),32)).toEqual({kind:"rejected",status:400});}
});
test("body snapshot is detached, cached and independently bounded on each call",async()=>{
 const source=new Uint8Array([1,2,3]);const native=new Request("http://localhost/",{method:"POST",body:source});
 const result=await snapshotRequest(native,3);expect(native.bodyUsed).toBe(true);if(result.kind!=="request")throw Error("rejected");
 source.fill(9);const first=value(await api.body(result.value,3n));expect(value(await api.body(result.value,100n))).toBe(first);
 const copy=copyBytes(first,origin);expect(copy).toEqual(new Uint8Array([1,2,3]));copy.fill(0);expect(copyBytes(first,origin)).toEqual(new Uint8Array([1,2,3]));
 for(const limit of [-1n,0n,2n])check(await api.body(result.value,limit),1104,{limit});
 const empty=await snapshot("http://localhost/");expect(copyBytes(value(await api.body(empty,0n)),origin)).toHaveLength(0);
});
test("native read failure and oversize reject before producing a callback request",async()=>{
 let cancelled=0;const stream=new ReadableStream<Uint8Array>({start(c){c.enqueue(new Uint8Array([1,2]));c.enqueue(new Uint8Array([3,4]));},cancel(){cancelled++;}});
 expect(await snapshotRequest(new Request("http://localhost/",{method:"POST",body:stream}),3)).toEqual({kind:"rejected",status:413});expect(cancelled).toBe(1);
 const failed=new ReadableStream<Uint8Array>({pull(){throw Error("private body failure");}});
 expect(await snapshotRequest(new Request("http://localhost/",{method:"POST",body:failed}),3)).toEqual({kind:"rejected",status:400});
});
test("JSON preserves exact integers and uses the caller's bounded expansion budget",async()=>{
 const schema={root:"int",nodes:[{identity:"int",kind:"primitive",name:"int"}]};
 for(const [text,expected] of [["9007199254740993",9007199254740993n],["1e8",100000000n]] as const){
  const request=await snapshot("http://localhost/",{method:"POST",body:text,headers:{"content-type":"application/json; charset=utf-8"}});
  expect(value(await api.json<bigint>(schema,request,32n))).toBe(expected);expect(value(await api.json<bigint>(schema,request,32n))).toBe(expected);
 }
 const expanded=await snapshot("http://localhost/",{method:"POST",body:"1e8",headers:{"content-type":"application/json"}});
 check(await api.json(schema,expanded,3n),1110,{path:"",reason:"byte_limit"});
 check(await api.json(schema,expanded,2n),1104,{limit:2n});
});
test("form decoding reads the same immutable body and enforces its media contract",async()=>{
 const schema={root:"form",fields:[{name:"name",kind:"str" as const},{name:"tags",kind:"array" as const}]};
 const text="name=a&tags=x&tags=y",request=await snapshot("http://localhost/",{method:"POST",body:text,headers:{"content-type":"application/x-www-form-urlencoded; charset=UTF-8"}});
 const expected=record("form",[["name","a"],["tags",array(["x","y"])]]);
 expect(value(await api.form<typeof expected>(schema,request,BigInt(text.length)))).toEqual(expected);expect(value(await api.form<typeof expected>(schema,request,100n))).toEqual(expected);
 expect(new TextDecoder().decode(copyBytes(value(await api.body(request,100n)),origin))).toBe(text);
 for(const type of ["text/plain","multipart/form-data","application/x-www-form-urlencoded; charset=latin1","application/x-www-form-urlencoded; charset=utf-8; charset=utf-8"]){const wrong=await snapshot("http://localhost/",{method:"POST",body:text,headers:{"content-type":type}});check(await api.form(schema,wrong,100n),1100,{reason:"unsupported_media_type"});}
 for(const [body,id,reason] of [["name=%",1100,"invalid_form_encoding"],["name=%FF",1110,"utf8"],["tags=x",1100,"form_missing"],["name=a&name=b",1100,"form_repeated"]] as const){const bad=await snapshot("http://localhost/",{method:"POST",body,headers:{"content-type":"application/x-www-form-urlencoded"}});check(await api.form(schema,bad,100n),id,{reason});}
});
test("duplicate headers pin native coalescing and exact media precedence",async()=>{
 const dup=await snapshot("http://localhost/",{method:"POST",body:"{}",headers:[["content-type","application/json"],["content-type","text/plain"]]});
 const schema={root:"int",nodes:[{identity:"int",kind:"primitive",name:"int"}]};
 check(await api.json(schema,dup,100n),1100,{reason:"unsupported_media_type"});
 const multi=await snapshot("http://localhost/",{headers:[["x-multi","1"],["x-multi","2"]]});
 const headers=value(await api.headers(multi));expect(headers).toHaveLength(1);
 const first=headers[0]!;expect(dataProperty(first,"name")).toBe("x-multi");expect(dataProperty(first,"value")).toBe("1, 2");
 const wrong=await snapshot("http://localhost/",{method:"POST",body:"[1,\"x\"]",headers:{"content-type":"application/json"}});
 check(await api.json({root:"ints",nodes:[{identity:"ints",kind:"array",name:"",element:"int"},{identity:"int",kind:"primitive",name:"int"}]},wrong,100n),1110,{path:"/1",reason:"type"});
});
test("forged request handles fail without invoking user traps",async()=>{
 let traps=0;const forged=new Proxy({},{get(){traps++;throw Error("secret");},getPrototypeOf(){traps++;throw Error("secret");}});
 expect(isRequest(forged)).toBe(false);expect((await invoke(()=>api.path(forged),origin)).kind).toBe("standard");expect(traps).toBe(0);
});
test("response status boundaries keep bodyless statuses separate",async()=>{
 const headers=value(await responses.emptyHeaders());
 for(const code of [-1n,0n,199n,600n,10n**100n]){check(await responses.makeStatus(code),1100,{reason:"invalid_status"});check(await responses.makeBodyStatus(code),1100,{reason:"invalid_status"});}
 for(const code of [200n,204n,205n,304n,599n]){const status=value(await responses.makeStatus(code));const response=nativeResponse(value(await responses.empty(status,headers)));expect(response.status).toBe(Number(code));expect(await response.text()).toBe("");expect(response.headers.get("x-content-type-options")).toBe("nosniff");}
 for(const code of [204n,205n,304n])check(await responses.makeBodyStatus(code),1100,{reason:"invalid_status"});
 for(const [get,code] of [[responses.ok,200],[responses.unprocessable,422],[responses.internal,500],[responses.unavailable,503]] as const){const status=value(await get());expect(isHTTPValue("body_status",status)).toBe(true);expect(isHTTPValue("status",status)).toBe(false);expect(nativeResponse(value(await responses.text(status,headers,"ok"))).status).toBe(code);}
});
test("native header validation rejects fixed sink overrides and hop-by-hop fields",async()=>{
 const header=(name:string,value:string)=>array([record("header",[["name",name],["value",value]])]);
 for(const name of ["content-type","Content-Length","X-Content-Type-Options","connection","keep-alive","proxy-authenticate","proxy-authorization","te","trailer","transfer-encoding","upgrade","bad name",""]){check(await responses.makeHeaders(header(name,"x")),1100,{reason:"invalid_header"});}
 for(const content of ["a\nb","a\rb","a\0b","\ud800","😀"])check(await responses.makeHeaders(header("x-test",content)),1100,{reason:"invalid_header"});
 const headers=value(await responses.makeHeaders(header("X-Test"," value "))),status=value(await responses.ok());
 const response=nativeResponse(value(await responses.text(status,headers,"body")));expect(response.headers.get("x-test")).toBe("value");expect(response.headers.get("content-type")).toBe("text/plain; charset=utf-8");
});
test("immutable response reuse makes independent native bodies with fixed encodings",async()=>{
 const status=value(await responses.ok()),headers=value(await responses.emptyHeaders());
 const text=value(await responses.text(status,headers,"a\ud800b"));expect(Object.isFrozen(text)).toBe(true);
 const first=nativeResponse(text),second=nativeResponse(text);expect(first).not.toBe(second);expect(await first.text()).toBe("a�b");expect(await second.text()).toBe("a�b");
 const body=ownBytes(new Uint8Array([0,255,128]));const binary=nativeResponse(value(await responses.bytes(status,headers,body)));expect(binary.headers.get("content-type")).toBe("application/octet-stream");expect(new Uint8Array(await binary.arrayBuffer())).toEqual(new Uint8Array([0,255,128]));
 const schema={root:"int",nodes:[{identity:"int",kind:"primitive",name:"int"}]};const json=nativeResponse(value(await responses.json(schema,status,headers,9007199254740993n)));expect(json.headers.get("content-type")).toBe("application/json; charset=utf-8");expect(await json.text()).toBe("9007199254740993");
 check(await responses.json({root:"str",nodes:[{identity:"str",kind:"primitive",name:"str"}]},status,headers,"\ud800"),1110,{path:"",reason:"unicode_scalar"});
});
test("response sinks reject forged and wrong-kind opaque handles without traps",async()=>{
 let traps=0;const forged=new Proxy({},{get(){traps++;throw Error("secret");}});expect(isHTTPValue("server_response",forged)).toBe(false);expect(()=>nativeResponse(forged)).toThrow();
 const status=value(await responses.ok()),headers=value(await responses.emptyHeaders());
 expect((await invoke(()=>responses.html(status,headers,forged),origin)).kind).toBe("standard");
 expect((await invoke(()=>responses.text(headers,status,"wrong"),origin)).kind).toBe("standard");expect(traps).toBe(0);
});
test("router rejects duplicates, normalized aliases and unsupported path patterns",async()=>{
 const status=value(await responses.ok()),headers=value(await responses.emptyHeaders());const callback=async()=>responses.text(status,headers,"ok");
 const first=value(await routing.get("/x",callback));check(await routing.make(array([first,first])),1231,{method:"GET",path:"/x"});
 for(const alias of ["/%78","/a/../x"]){const second=value(await routing.get(alias,callback));check(await routing.make(array([first,second])),1232,{first:"/x",second:alias});}
 for(const path of ["","x","//elsewhere/x","/x?q=1","/x#hash","/x/*","/x/:id","/x\\y","/%","/%FF","/\ud800","/x\ny"]){check(await routing.get(path,callback),1230,{reason:"path"});}
 expect((await routing.make(array([first,value(await routing.post("/x",callback))]))).kind).toBe("ok");
 let traps=0;const forged=new Proxy({},{get(){traps++;throw Error("secret");}});expect(isRouterValue("route",forged)).toBe(false);expect((await invoke(()=>routing.make(array([forged])),origin)).kind).toBe("standard");expect(traps).toBe(0);
});
test("exact dispatch distinguishes 404 and 405 and never aliases HEAD to GET",async()=>{
 const status=value(await responses.ok()),headers=value(await responses.emptyHeaders());let calls=0;
 const callback=async()=>{calls++;return responses.text(status,headers,"ok");};
 const router=value(await routing.make(array([value(await routing.get("/x",callback)),value(await routing.post("/x",callback))])));
 for(const [path,method,code,allow] of [["/missing","GET",404,null],["/x/","GET",404,null],["/x","HEAD",405,"GET, POST"],["/x","PUT",405,"GET, POST"],["/x","OPTIONS",405,"GET, POST"]] as const){const response=value(await dispatch(router,await snapshot("http://localhost"+path,{method})));expect(response.status).toBe(code);expect(response.headers.get("allow")).toBe(allow);}
 expect(calls).toBe(0);
 for(const method of ["GET","POST"]){const response=value(await dispatch(router,await snapshot("http://localhost/%78?q=ignored",{method})));expect(response.status).toBe(200);expect(await response.text()).toBe("ok");}expect(calls).toBe(2);
});
test("real loopback dispatch awaits completion and snapshots bodies before callback",async()=>{
 const status=value(await responses.ok()),headers=value(await responses.emptyHeaders());let calls=0,entered!:()=>void,release!:()=>void;
 const started=new Promise<void>(resolve=>{entered=resolve;}),gate=new Promise<void>(resolve=>{release=resolve;});
 const callback=async(request:unknown)=>{calls++;const body=value(await api.body(request,16n));expect(value(await api.body(request,16n))).toBe(body);entered();await gate;return responses.text(status,headers,value(await api.path(request))+":"+value(await api.queryOne(request,"q"))+":"+new TextDecoder().decode(copyBytes(body,origin)));};
 const router=value(await routing.make(array([value(await routing.post("/café",callback))])));
 const server=Bun.serve({hostname:"127.0.0.1",port:0,async fetch(request){const result=await snapshotRequest(request,16);if(result.kind==="rejected")return new Response(null,{status:result.status});return value(await dispatch(router,result.value));}});
 try{
  let settled=false;const pending=fetch(new URL("/a/../caf%C3%A9?q=%2B",server.url),{method:"POST",body:"same"}).then(response=>{settled=true;return response;});
  await started;expect(settled).toBe(false);release();const result=await pending;expect(await result.text()).toBe("/café:+:same");
  const large=await fetch(new URL("/caf%C3%A9?q=x",server.url),{method:"POST",body:"x".repeat(17)});expect(large.status).toBe(413);expect(calls).toBe(1);
  const head=await fetch(new URL("/caf%C3%A9",server.url),{method:"HEAD"});expect(head.status).toBe(405);expect(head.headers.get("allow")).toBe("POST");expect(calls).toBe(1);
 }finally{release();await server.stop(true);}
});
