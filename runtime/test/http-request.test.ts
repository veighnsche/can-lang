import {test,expect} from "bun:test";
import {createHash} from "node:crypto";
import {catalogue} from "../catalogue.ts";
import {createDomainRuntime,domainFailureDiagnostics,type FailureShape} from "../domain.ts";
import {createRequests,createResponses,snapshotRequest,snapshotRequestLazy,abandonRequest,normalizedPath,isRequest,isHTTPValue,nativeResponse} from "../platform/http.ts";
import {copyBytes,ownBytes} from "../bytes.ts";
import {value,invoke,success,failure,type Completion} from "../completion.ts";
import {record,array,dataProperty} from "../data.ts";
import {createRouter,dispatch,routeKind,isRouterValue} from "../platform/router.ts";
import {createStreamReads} from "../transport/stream/readable.ts";
import {createStreamWrites,registerSink} from "../transport/stream/writable.ts";
import {runOwnedRoot} from "../owner.ts";
const hash=(kind:string,name:string)=>createHash("sha256").update("can-concrete-type-v1\0"+JSON.stringify([kind,name])).digest("hex");
const shape=(kind:string,declaration:string,fields:{name:string;type:string}[]=[]):FailureShape=>({identity:hash(kind,declaration),kind,declaration,fields,arguments:[],leaves:[],inputs:[],errors:[]});
const str=shape("primitive","str"),int=shape("primitive","int");
const declarations=catalogue.errors.filter(e=>[1100,1104,1110,1230,1231,1232,1305,1316,1317,1318,1319].includes(e.id));
const errors=declarations.map(e=>shape("error",e.identity,e.fields.map(f=>({name:f.name,type:f.type==="int"?int.identity:str.identity}))));
const domain=createDomainRuntime({declarations:declarations.map(e=>({...e,parameters:0})),shapes:[str,int,...errors]});
const api=createRequests(domain,{invalid:errors[0]!.identity,limit:errors[1]!.identity,invalidData:errors[2]!.identity,header:"header",close:errors[10]!.identity,writeFailed:errors[8]!.identity});
const reads=createStreamReads(domain,{readFailed:errors[7]!.identity,cancelled:errors[9]!.identity,closeFailed:errors[10]!.identity,limitExceeded:errors[6]!.identity});
const writes=createStreamWrites(domain,{writeFailed:errors[8]!.identity,closeFailed:errors[10]!.identity});
const responses=createResponses(domain,{invalid:errors[0]!.identity,invalidData:errors[2]!.identity,close:errors[10]!.identity,writeFailed:errors[8]!.identity,limit:errors[1]!.identity});
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
 const cookies=await snapshot("http://localhost/",{headers:[["x-multi","1"],["set-cookie","a=1"],["set-cookie","b=2"],["x-multi","2"]]});
 const seen=value(await api.headers(cookies));
 expect(seen.filter(h=>dataProperty(h,"name")==="set-cookie").map(h=>dataProperty(h,"value"))).toEqual(["a=1","b=2"]);
 expect(seen.filter(h=>dataProperty(h,"name")==="x-multi").map(h=>dataProperty(h,"value"))).toEqual(["1, 2"]);
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
 for(const name of ["content-type","Content-Length","X-Content-Type-Options","Content-Security-Policy","content-security-policy-report-only","HX-Redirect","hx-location","connection","keep-alive","proxy-authenticate","proxy-authorization","te","trailer","transfer-encoding","upgrade","bad name",""]){check(await responses.makeHeaders(header(name,"x")),1100,{reason:"invalid_header"});}
 for(const content of ["a\nb","a\rb","a\0b","\ud800","😀"])check(await responses.makeHeaders(header("x-test",content)),1100,{reason:"invalid_header"});
 const headers=value(await responses.makeHeaders(header("X-Test"," value "))),status=value(await responses.ok());
 const response=nativeResponse(value(await responses.text(status,headers,"body")));expect(response.headers.get("x-test")).toBe("value");expect(response.headers.get("content-type")).toBe("text/plain; charset=utf-8");
 const jar=value(await responses.makeHeaders(array([record("header",[["name","set-cookie"],["value","a=1"]]),record("header",[["name","set-cookie"],["value","b=2"]])])));
 expect(nativeResponse(value(await responses.text(status,jar,"body"))).headers.getSetCookie()).toEqual(["a=1","b=2"]);
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
test("method table routes all seven methods with sorted Allow fallback",async()=>{
 const status=value(await responses.ok()),headers=value(await responses.emptyHeaders());
 const callback=async(request:unknown)=>responses.text(status,headers,value(await api.method(request)));
 const router=value(await routing.make(array([value(await routing.get("/m",callback)),value(await routing.post("/m",callback)),value(await routing.put("/m",callback)),value(await routing.patch("/m",callback)),value(await routing.delete("/m",callback)),value(await routing.options("/m",callback)),value(await routing.head("/m",callback))])));
 for(const method of ["GET","POST","PUT","PATCH","DELETE","OPTIONS"] as const){const response=value(await dispatch(router,await snapshot("http://localhost/m",{method})));expect(response.status).toBe(200);expect(await response.text()).toBe(method);}
 {const response=value(await dispatch(router,await snapshot("http://localhost/m",{method:"HEAD"})));expect(response.status).toBe(200);expect(await response.text()).toBe("");expect(response.headers.get("content-length")).toBe("4");}
 const trace=value(await dispatch(router,await snapshot("http://localhost/m",{method:"TRACE"})));
 expect(trace.status).toBe(405);expect(trace.headers.get("allow")).toBe("DELETE, GET, HEAD, OPTIONS, PATCH, POST, PUT");
 check(await routing.make(array([value(await routing.put("/m",callback)),value(await routing.put("/m",callback))])),1231,{method:"PUT",path:"/m"});
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
test("lazy bodies defer bytes until first access then buffer and cache",async()=>{
 const stream=new ReadableStream<Uint8Array>({pull(c){c.enqueue(new TextEncoder().encode("chunk"));c.close();}});
 const native=new Request("http://localhost/",{method:"POST",body:stream});
 const result=await snapshotRequestLazy(native,1024);if(result.kind!=="request")throw Error("rejected request");
 expect(native.bodyUsed).toBe(false);
 const first=value(await api.body(result.value,100n));expect(native.bodyUsed).toBe(true);
 expect(new TextDecoder().decode(copyBytes(first,origin))).toBe("chunk");
 expect(value(await api.body(result.value,100n))).toBe(first);
 check(await api.body(result.value,4n),1104,{limit:4n});
});
test("live buffered access reports ingress cap and read failures in handler",async()=>{
 const big=await snapshotRequestLazy(new Request("http://localhost/",{method:"POST",body:"x".repeat(17)}),16);
 if(big.kind!=="request")throw Error("rejected request");
 check(await api.body(big.value,100n),1104,{limit:16n});
 const failed=new ReadableStream<Uint8Array>({pull(){throw Error("wire down");}});
 const broken=await snapshotRequestLazy(new Request("http://localhost/",{method:"POST",body:failed}),16);
 if(broken.kind!=="request")throw Error("rejected request");
 check(await api.body(broken.value,100n),1100,{reason:"body_read"});
});
test("body readers are one-shot and poison buffered reads once live",async()=>{
 const owned=await runOwnedRoot(async ()=>{
  const lazy=await snapshotRequestLazy(new Request("http://localhost/",{method:"POST",body:"hello"}),64);
  if(lazy.kind!=="request")throw Error("rejected request");
  const reader=value(await api.bodyStream(lazy.value,3n));
  const first=value(await reads.readMany(reader,2n));
  expect(first.map(entry=>new TextDecoder().decode(copyBytes(entry,origin)))).toEqual(["hel","lo"]);
  check(await api.bodyStream(lazy.value,3n),1100,{reason:"body_consumed"});
  check(await api.body(lazy.value,100n),1100,{reason:"body_consumed"});
  expect((await reads.closeReader(reader)).kind).toBe("ok");
  const early=await snapshotRequestLazy(new Request("http://localhost/",{method:"POST",body:"hello"}),64);
  if(early.kind!=="request")throw Error("rejected request");
  expect(new TextDecoder().decode(copyBytes(value(await api.body(early.value,100n)),origin))).toBe("hello");
  const replay=value(await api.bodyStream(early.value,10n));
  const chunks=value(await reads.readMany(replay,5n));
  expect(chunks.map(entry=>new TextDecoder().decode(copyBytes(entry,origin))).join("")).toBe("hello");
  expect((await reads.closeReader(replay)).kind).toBe("ok");
  const buffered=await snapshot("http://localhost/",{method:"POST",body:"hello"});
  const memory=value(await api.bodyStream(buffered,10n));
  expect((await reads.closeReader(memory)).kind).toBe("ok");
  check(await api.bodyStream(buffered,10n),1100,{reason:"body_consumed"});
  expect(value(await api.body(buffered,100n))).toBe(value(await api.body(buffered,100n)));
  check(await api.bodyStream(buffered,0n),1104,{limit:0n});
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
});
test("stream marking routes bodies around the eager pre-read",async()=>{
 const status=value(await responses.ok()),headers=value(await responses.emptyHeaders());
 const callback=async()=>responses.text(status,headers,"ok");
 const plain=value(await routing.post("/u",callback)),marked=value(await routing.stream(plain));
 const router=value(await routing.make(array([marked,value(await routing.get("/u",callback))])));
 expect(routeKind(router,"POST","/u")).toBe("stream");
 expect(routeKind(router,"GET","/u")).toBe("buffered");
 expect(routeKind(router,"PUT","/u")).toBe(undefined);
 expect(routeKind(router,"POST","/missing")).toBe(undefined);
 check(await routing.make(array([plain,marked])),1231,{method:"POST",path:"/u"});
 let traps=0;const forged=new Proxy({},{get(){traps++;throw Error("secret");}});
 expect((await invoke(()=>routing.stream(forged),origin)).kind).toBe("standard");expect(traps).toBe(0);
});
test("dispatch abandons unread live bodies",async()=>{
 let cancelled=0;
 const stream=new ReadableStream<Uint8Array>({start(c){c.enqueue(new TextEncoder().encode("unread"));},cancel(){cancelled++;}});
 const status=value(await responses.ok()),headers=value(await responses.emptyHeaders());
 const router=value(await routing.make(array([value(await routing.stream(value(await routing.post("/u",async()=>responses.text(status,headers,"ok")))))])));
 const lazy=await snapshotRequestLazy(new Request("http://localhost/u",{method:"POST",body:stream}),64);
 if(lazy.kind!=="request")throw Error("rejected request");
 const response=value(await dispatch(router,lazy.value));
 expect(response.status).toBe(200);expect(cancelled).toBe(1);
});
test("unclosed request handles drain cleanly by design",async()=>{
 const owned=await runOwnedRoot(async ()=>{
  const lazy=await snapshotRequestLazy(new Request("http://localhost/",{method:"POST",body:"hello"}),64);
  if(lazy.kind!=="request")throw Error("rejected request");
  value(await api.bodyStream(lazy.value,4n));
  const pending=value(await responses.stream(value(await responses.ok()),value(await responses.emptyHeaders())));
  value(await responses.writer(pending));
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
});
test("body readers outside ownership fail resource_state",async()=>{
 const lazy=await snapshotRequestLazy(new Request("http://localhost/",{method:"POST",body:"hello"}),64);
 if(lazy.kind!=="request")throw Error("rejected request");
 expect((await invoke(()=>api.bodyStream(lazy.value,4n),origin)).kind).toBe("standard");
});
test("stream responses produce through short writes into a bounded queue",async()=>{
 const owned=await runOwnedRoot(async ()=>{
  const status=value(await responses.ok()),headers=value(await responses.emptyHeaders());
  const pending=value(await responses.stream(status,headers));
  const out=value(await responses.writer(pending));
  check(await responses.writer(pending),1100,{reason:"writer_taken"});
  const first=value(await writes.writeSome(out,ownBytes(new TextEncoder().encode("ab"))));
  const second=value(await writes.writeSome(out,ownBytes(new TextEncoder().encode("c"))));
  expect((await writes.closeWriter(out)).kind).toBe("ok");
  const served=nativeResponse(pending);
  return success({first,second,type:served.headers.get("content-type"),body:await served.text()});
 });
 expect(owned.cleanupFailed).toBe(false);
 expect(value(owned.completion)).toEqual({first:2n,second:1n,type:"application/octet-stream",body:"abc"});
});
test("the produced queue caps at one megabyte with short writes",async()=>{
 const owned=await runOwnedRoot(async ()=>{
  const pending=value(await responses.stream(value(await responses.ok()),value(await responses.emptyHeaders())));
  const out=value(await responses.writer(pending));
  const accepted=value(await writes.writeSome(out,ownBytes(new Uint8Array(2*1048576))));
  const short=value(await writes.writeSome(out,ownBytes(new Uint8Array([1]))));
  const reader=nativeResponse(pending).body!.getReader();
  const drained=await reader.read();
  reader.releaseLock();
  const reopened=value(await writes.writeSome(out,ownBytes(new Uint8Array([1,2,3]))));
  expect((await writes.closeWriter(out)).kind).toBe("ok");
  return success({accepted,short,drainedBytes:drained.done?0:drained.value.byteLength,reopened});
 });
 expect(owned.cleanupFailed).toBe(false);
 expect(value(owned.completion)).toEqual({accepted:1048576n,short:0n,drainedBytes:1048576,reopened:3n});
});
test("pending responses without writers serve empty and convert once",async()=>{
 const status=value(await responses.ok()),headers=value(await responses.emptyHeaders());
 const plain=value(await responses.text(status,headers,"x"));
 check(await responses.writer(plain),1100,{reason:"not_streaming"});
 const untaken=value(await responses.stream(status,headers));
 const owned=await runOwnedRoot(async ()=>{
  const once=value(await responses.stream(status,headers));
  const out=value(await responses.writer(once));
  expect((await writes.closeWriter(out)).kind).toBe("ok");
  const first=nativeResponse(once);
  let reused="no-throw";
  try{nativeResponse(once);}catch{reused="thrown";}
  const leaked=value(await responses.stream(status,headers));
  const writer=value(await responses.writer(leaked));
  const accepted=value(await writes.writeSome(writer,ownBytes(new TextEncoder().encode("late"))));
  const converted=nativeResponse(leaked);
  return success({first:await first.text(),reused,accepted,converted});
 });
 expect(owned.cleanupFailed).toBe(false);
 const done=value(owned.completion);
 expect(done.first).toBe("");expect(done.reused).toBe("thrown");expect(done.accepted).toBe(4n);
 expect(await done.converted.text()).toBe("late");
 expect(await nativeResponse(untaken).text()).toBe("");
});
test("HEAD suppresses stream bodies without entity length",async()=>{
 const owned=await runOwnedRoot(async ()=>{
  const pending=value(await responses.stream(value(await responses.ok()),value(await responses.emptyHeaders())));
  const out=value(await responses.writer(pending));
  await writes.closeWriter(out);
  const head=nativeResponse(pending,true);
  return success({body:await head.text(),length:head.headers.get("content-length")});
 });
 expect(value(owned.completion)).toEqual({body:"",length:null});
});
test("SSE frames events with validated fields",async()=>{
 const owned=await runOwnedRoot(async ()=>{
  const pending=value(await responses.sse(value(await responses.ok()),value(await responses.emptyHeaders())));
  const out=value(await responses.writer(pending));
  const event=(data:string,event:string,id:string,retry:string)=>record("sse_event",[["data",data],["event",event],["id",id],["retry",retry]]);
  expect(value(await responses.sseSend(out,event("tick","beat","7","")))).toBe(30n);
  expect(value(await responses.sseSend(out,event("a\nb\r\nc\rd","","","")))).toBe(33n);
  expect(value(await responses.sseSend(out,event("","","","100")))).toBe(12n);
  expect(value(await responses.sseSend(out,event("","","","")))).toBe(1n);
  expect(value(await responses.sseComment(out,"still here"))).toBe(14n);
  check(await responses.sseSend(out,event("x","bad\nevent","","")),1100,{reason:"sse_event"});
  check(await responses.sseSend(out,event("x","","bad\rid","")),1100,{reason:"sse_id"});
  check(await responses.sseSend(out,event("x","","","now")),1100,{reason:"sse_retry"});
  check(await responses.sseComment(out,"bad\ncomment"),1100,{reason:"sse_comment"});
  return success(pending);
 });
 const served=nativeResponse(value(owned.completion));
 expect(owned.cleanupFailed).toBe(false);
 expect(served.headers.get("content-type")).toBe("text/event-stream; charset=utf-8");
 expect(await served.text()).toBe("event: beat\nid: 7\ndata: tick\n\n"+"data: a\ndata: b\ndata: c\ndata: d\n\n"+"retry: 100\n\n"+"\n"+": still here\n\n");
});
test("SSE frames are atomic against the queue bound",async()=>{
 const owned=await runOwnedRoot(async ()=>{
  const pending=value(await responses.sse(value(await responses.ok()),value(await responses.emptyHeaders())));
  const out=value(await responses.writer(pending));
  const huge="x".repeat(1048576);
  check(await responses.sseSend(out,record("sse_event",[["data",huge],["event",""],["id",""],["retry",""]])),1104,{limit:1048576n});
  expect(value(await responses.sseSend(out,record("sse_event",[["data","ok"],["event",""],["id",""],["retry",""]])))).toBe(10n);
  const file=registerSink({write:()=>0,flush:()=>undefined,end:()=>undefined},(identity,fields,cause)=>failure(domain.create(identity,record(identity,fields),origin,cause)),errors[10]!.identity);
  check(await responses.sseSend(file,record("sse_event",[["data","x"],["event",""],["id",""],["retry",""]])),1100,{reason:"not_streaming"});
  expect((await writes.closeWriter(file)).kind).toBe("ok");
  return success(pending);
 });
 expect(owned.cleanupFailed).toBe(false);
 expect(await nativeResponse(value(owned.completion)).text()).toBe("data: ok\n\n");
});
test("stream responses serve produced queues over loopback",async()=>{
 const seen={status:0,body:""};
 const owned=await runOwnedRoot(async ()=>{
  const status=value(await responses.ok()),headers=value(await responses.emptyHeaders());
  const callback=async()=>{
   const pending=value(await responses.stream(status,headers));
   const out=value(await responses.writer(pending));
   value(await writes.writeSome(out,ownBytes(new TextEncoder().encode("one;"))));
   value(await writes.writeSome(out,ownBytes(new TextEncoder().encode("two"))));
   value(await writes.closeWriter(out));
   return success(pending);
  };
  const router=value(await routing.make(array([value(await routing.get("/s",callback))])));
  const server=Bun.serve({hostname:"127.0.0.1",port:0,async fetch(native){
   const snap=await snapshotRequest(native,65536);
   if(snap.kind!=="request")return new Response(null,{status:snap.status});
   return value(await dispatch(router,snap.value));
  }});
  try{
   const res=await fetch(new URL("/s",server.url));
   seen.status=res.status;seen.body=await res.text();
  }finally{await server.stop(true);}
  return success(undefined);
 });
 expect(owned.cleanupFailed).toBe(false);expect(owned.completion.kind).toBe("ok");
 expect(seen).toEqual({status:200,body:"one;two"});
});
