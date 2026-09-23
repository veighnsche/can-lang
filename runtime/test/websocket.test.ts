import {test,expect} from "bun:test";
import {createHash} from "node:crypto";
import {createServer as createNetServer} from "node:net";
import {tmpdir} from "node:os";
import {join} from "node:path";
import {rm} from "node:fs/promises";
import {catalogue} from "../catalogue.ts";
import {createDomainRuntime,domainFailureDiagnostics,type FailureShape} from "../domain.ts";
import {createWebSockets,isWebSocketValue} from "../platform/websocket.ts";
import {createRequests,createResponses} from "../platform/http.ts";
import {createRouter} from "../platform/router.ts";
import {createServer} from "../platform/server.ts";
import {createStreamReads} from "../transport/stream/readable.ts";
import {copyBytes,ownBytes} from "../bytes.ts";
import {value,success} from "../completion.ts";
import {record,array,dataArray,dataProperty,recordIdentity} from "../data.ts";
import {isStandardFailure,standardFailureKind,standardFailureMessage,standardFailureDiagnostics} from "../failure.ts";
import {runOwnedRoot} from "../owner.ts";
import type {Completion} from "../completion.ts";
const hash=(kind:string,name:string)=>createHash("sha256").update("can-concrete-type-v1\0"+JSON.stringify([kind,name])).digest("hex");
const shape=(kind:string,declaration:string,fields:{name:string;type:string}[]=[]):FailureShape=>({identity:hash(kind,declaration),kind,declaration,fields,arguments:[],leaves:[],inputs:[],errors:[]});
const str=shape("primitive","str"),int=shape("primitive","int");
const httpDecls=catalogue.errors.filter(e=>[1100,1104,1110,1230,1231,1232,1233,1234,1235,1305,1316,1317,1318,1319,1333,1334,1335,1336,1337,1338,1339,1340].includes(e.id));
const declarations=httpDecls.map(e=>({...e,parameters:0}));
const errors=httpDecls.map(e=>shape("error",e.identity,e.fields.map(f=>({name:f.name,type:f.type==="int"?int.identity:str.identity}))));
const domain=createDomainRuntime({declarations,shapes:[str,int,...errors]});
const identity=(name:string)=>errors.find(e=>e.declaration===name)!.identity;
const failOf=(result:Completion)=>{expect(result.kind).toBe("domain");if(result.kind!=="domain")throw Error("expected domain");return domainFailureDiagnostics(result.value);};
function check(result:Completion,id:number,payload:object){const d=failOf(result);expect(d.declaration.id).toBe(id);expect(d.payload).toMatchObject(payload);}
function standardOf(thrown:unknown){expect(isStandardFailure(thrown)).toBe(true);return standardFailureKind(thrown as never);}
const TEXT="can.std.ws@1::text",BINARY="can.std.ws@1::binary",DRAIN="can.std.ws@1::drain",CLOSE="can.std.ws@1::closed",CONNECTION="can.std.ws@1::connection";
const ws=createWebSockets(domain,{connectFailed:identity("can.std.ws@1::connect_failed"),upgradeFailed:identity("can.std.ws@1::upgrade_failed"),unsupportedProtocol:identity("can.std.ws@1::unsupported_protocol"),sendFailed:identity("can.std.ws@1::send_failed"),invalidClose:identity("can.std.ws@1::invalid_close"),limitExceeded:identity("can.std.ws@1::limit_exceeded"),invalidUrl:identity("can.std.ws@1::invalid_url"),invalidProtocol:identity("can.std.ws@1::invalid_protocol"),closeFailed:identity("can.std.stream@1::close_failed"),text:TEXT,binary:BINARY,drain:DRAIN,close:CLOSE,connection:CONNECTION});
const reads=createStreamReads(domain,{readFailed:identity("can.std.stream@1::read_failed"),cancelled:identity("can.std.stream@1::cancelled"),closeFailed:identity("can.std.stream@1::close_failed"),limitExceeded:identity("can.std.files@1::limit_exceeded")});
const typeId=(name:string)=>catalogue.types.find(t=>t.name===name)!.identity;
const reqApi=createRequests(domain,{invalid:identity("can.std.http@1::invalid_request"),limit:identity("can.std.http@1::body_limit"),invalidData:identity("can.std.codec@1::invalid_data"),header:"can.std.http@1::header",close:identity("can.std.stream@1::close_failed"),writeFailed:identity("can.std.stream@1::write_failed"),multipartForm:typeId("http::multipart_form"),multipartField:typeId("http::multipart_field"),multipartFile:typeId("http::multipart_file")});
const responses=createResponses(domain,{invalid:identity("can.std.http@1::invalid_request"),invalidData:identity("can.std.codec@1::invalid_data"),close:identity("can.std.stream@1::close_failed"),writeFailed:identity("can.std.stream@1::write_failed"),limit:identity("can.std.http@1::body_limit")});
const routing=createRouter(domain,{invalid:identity("can.std.http@1::invalid_route"),duplicate:identity("can.std.http@1::duplicate_route"),ambiguous:identity("can.std.http@1::ambiguous_route")});
const servers=createServer(domain,{invalidConfig:identity("can.std.http@1::invalid_server_config"),bindFailed:identity("can.std.http@1::bind_failed"),shutdownFailed:identity("can.std.http@1::shutdown_failed")});
const origin={source:"test",start:0,end:0,invocation:[]};
// runOwnedRoot captures body throws into its completion instead of
// rethrowing, so every test must assert the completion it returns;
// ignoring it lets a mid-body throw pass vacuously.
async function owned(body:()=>Promise<void>):Promise<void>{
 const out=await runOwnedRoot(async()=>{await body();return success(undefined);});
 if(out.completion.kind!=="ok"||out.cleanupFailed){
  const cause=out.completion.kind==="standard"&&isStandardFailure(out.completion.value)?standardFailureDiagnostics(out.completion.value as never).cause as {message?:unknown;expected?:unknown;actual?:unknown;diff?:unknown}:undefined;
  const detail=out.completion.kind==="domain"?JSON.stringify(domainFailureDiagnostics(out.completion.value).payload):cause!==undefined?JSON.stringify({message:String(cause.message),expected:cause.expected,actual:cause.actual,diff:cause.diff}).slice(0,500):out.completion.kind;
  throw Error(`owned body failed: ${detail} cleanupFailed=${out.cleanupFailed}`);
 }
}
async function textResponse(status:bigint,body:string){return value(await responses.text(value(await responses.makeBodyStatus(status)),value(await responses.emptyHeaders()),body));}
async function untilDone(flag:{done?:boolean}):Promise<void>{
 for(let n=0;n<500&&!flag.done;n++)await new Promise(resolve=>setTimeout(resolve,10));
 expect(flag.done).toBe(true);
}
async function serveGet(path:string,callback:(request:unknown)=>Promise<Completion<unknown>>):Promise<{port:number;stop:()=>Promise<Completion<unknown>>}>{
 const port=await freePort();
 const config=value(await servers.makeConfig("127.0.0.1",BigInt(port),65536n,5000n));
 const built=value(await routing.make(array([value(await routing.get(path,callback))])));
 const token=value(await servers.start(config,built));
 return {port,stop:()=>servers.stop(token)};
}
const connectionOf=(completed:Completion)=>{const c=value(completed);expect(recordIdentity(c)).toBe(CONNECTION);return {session:dataProperty(c,"session"),events:dataProperty(c,"events"),protocol:dataProperty(c,"protocol")};};
const fieldOf=(item:unknown,name:string)=>{try{return dataProperty(item,name);}catch{return undefined;}};
const eventOf=(item:unknown)=>({kind:recordIdentity(item),text:fieldOf(item,"text"),data:fieldOf(item,"data"),code:fieldOf(item,"code"),reason:fieldOf(item,"reason")});
const strArray=(items:readonly string[])=>array(items);
async function freePort():Promise<number>{
 return new Promise((resolve,reject)=>{const probe=createNetServer();probe.on("error",reject);probe.listen(0,"127.0.0.1",()=>{const address=probe.address();if(typeof address==="object"&&address!==null){const port=address.port;probe.close(()=>resolve(port));}else{probe.close(()=>reject(Error("no port")));}});});
}
async function echoServer(hooks?:{onMessage?:(socket:{send:(m:string|Uint8Array)=>number},message:string|Buffer)=>void}):Promise<{port:number;close:()=>Promise<void>}>{
 const server=Bun.serve({hostname:"127.0.0.1",port:0,fetch(request,s){if(s.upgrade(request))return;return new Response("no-upgrade",{status:400});},websocket:{message(socket,message){if(hooks?.onMessage)hooks.onMessage(socket,message);else socket.send(message);}}});
 return {port:server.port,close:async()=>{await server.stop(true);}};
}
test("client echo carries text, binary, protocol and graceful close both ways",async()=>{
 const route=await echoServer();
 try{
  await owned(async()=>{
   const client=connectionOf(await ws.connect(`ws://127.0.0.1:${route.port}/`,strArray(["proto-a"]),65536n,64n,1048576n,5000n,false));
   expect(client.protocol).toBe("proto-a");
   expect(value(await ws.sendText(client.session,"hello"))).toBe(5n);
   const first=dataArray(value(await reads.readMany(client.events,1n)));
   expect(first).toHaveLength(1);expect(eventOf(first[0])).toMatchObject({kind:TEXT,text:"hello"});
   expect(value(await ws.sendBytes(client.session,ownBytes(new Uint8Array([1,2,3]))))).toBe(3n);
   const second=dataArray(value(await reads.readMany(client.events,1n)));
   expect(second).toHaveLength(1);
   const binary=eventOf(second[0]);expect(binary.kind).toBe(BINARY);
   expect(Array.from(copyBytes(binary.data,origin))).toEqual([1,2,3]);
   // Binary crosses through a copy: mutating the source cannot alias in.
   const source=new Uint8Array([9,9]);const held=ownBytes(source);source.fill(0);
   expect(value(await ws.sendBytes(client.session,held))).toBe(2n);
   const third=dataArray(value(await reads.readMany(client.events,1n)));
   expect(Array.from(copyBytes(eventOf(third[0]).data,origin))).toEqual([9,9]);
   expect(value(await ws.close(client.session,1000n,"client-bye"))).toBe(undefined);
   const last=dataArray(value(await reads.readMany(client.events,4n)));
   expect(last).toHaveLength(1);expect(eventOf(last[0])).toMatchObject({kind:CLOSE,code:1000n,reason:"client-bye"});
   expect(dataArray(value(await reads.readMany(client.events,4n)))).toHaveLength(0);
   expect(await reads.closeReader(client.events)).toMatchObject({kind:"ok"});
  });
 }finally{await route.close();}
});
test("connect validates URL, protocols and every bound before dialing",async()=>{
 await owned(async()=>{
  check(await ws.connect("not a url",strArray([]),65536n,64n,1048576n,5000n,false),1339,{reason:"unparseable"});
  check(await ws.connect("http://127.0.0.1:9/",strArray([]),65536n,64n,1048576n,5000n,false),1339,{reason:"scheme"});
  for(const bad of ["a,b","","has space","quo\"te"])check(await ws.connect("ws://127.0.0.1:9/",strArray([bad]),65536n,64n,1048576n,5000n,false),1340,{protocol:bad});
  for(const [args,limit] of [[[0n,64n,1048576n,5000n],0n],[[65536n,0n,1048576n,5000n],0n],[[65536n,64n,0n,5000n],0n],[[65536n,64n,1048576n,0n],0n],[[67108865n,64n,1048576n,5000n],67108865n],[[65536n,1025n,1048576n,5000n],1025n],[[65536n,64n,67108865n,5000n],67108865n],[[65536n,64n,1048576n,2147483648n],2147483648n]] as const)check(await ws.connect("ws://127.0.0.1:9/",strArray([]),...args,false),1338,{limit});
 });
});
test("connect failures distinguish refused, rejected, tls and timeout",async()=>{
 const port=await freePort();
 await owned(async()=>{
  check(await ws.connect(`ws://127.0.0.1:${port}/`,strArray([]),65536n,64n,1048576n,5000n,false),1333,{reason:"unreachable"});
 });
 const plain=Bun.serve({hostname:"127.0.0.1",port:0,fetch:()=>new Response("denied",{status:401})});
 try{
  await owned(async()=>{
   check(await ws.connect(`ws://127.0.0.1:${plain.port}/`,strArray([]),65536n,64n,1048576n,5000n,false),1333,{reason:"rejected"});
  });
 }finally{await plain.stop(true);}
 const cert=join(tmpdir(),"can-ws-test-cert.pem"),key=join(tmpdir(),"can-ws-test-key.pem");
 const made=Bun.spawnSync(["openssl","req","-x509","-newkey","rsa:2048","-keyout",key,"-out",cert,"-days","1","-nodes","-subj","/CN=127.0.0.1"]);
 expect(made.exitCode).toBe(0);
 const tls=Bun.serve({hostname:"127.0.0.1",port:0,tls:{cert:Bun.file(cert),key:Bun.file(key)},fetch(request,s){if(s.upgrade(request))return;return new Response("x",{status:400});},websocket:{message(socket,message){socket.send(message);}}});
 try{
  await owned(async()=>{
   check(await ws.connect(`wss://127.0.0.1:${tls.port}/`,strArray([]),65536n,64n,1048576n,5000n,false),1333,{reason:"tls"});
   const safe=connectionOf(await ws.connect(`wss://127.0.0.1:${tls.port}/`,strArray(["proto-a"]),65536n,64n,1048576n,5000n,true));
   expect(safe.protocol).toBe("proto-a");
   expect(value(await ws.sendText(safe.session,"tls-echo"))).toBe(8n);
   const got=dataArray(value(await reads.readMany(safe.events,1n)));
   expect(eventOf(got[0])).toMatchObject({kind:TEXT,text:"tls-echo"});
   expect(value(await ws.close(safe.session,1000n,""))).toBe(undefined);
   expect(dataArray(value(await reads.readMany(safe.events,2n)))).toHaveLength(1);
   expect(await reads.closeReader(safe.events)).toMatchObject({kind:"ok"});
  });
 }finally{await tls.stop(true);await rm(cert,{force:true});await rm(key,{force:true});}
 const silent=createNetServer(socket=>{socket.on("data",()=>{});});
 await new Promise<void>(resolve=>silent.listen(0,"127.0.0.1",()=>resolve()));
 try{
  const address=silent.address();if(typeof address!=="object"||address===null)throw Error("no silent port");
  await owned(async()=>{
   check(await ws.connect(`ws://127.0.0.1:${address.port}/`,strArray([]),65536n,64n,1048576n,200n,false),1333,{reason:"timeout"});
  });
 }finally{await new Promise<void>(resolve=>silent.close(()=>resolve()));}
});
test("sends enforce size, buffer and peer-close bounds with accepted counts",async()=>{
 const route=await echoServer();
 try{
  await owned(async()=>{
   const client=connectionOf(await ws.connect(`ws://127.0.0.1:${route.port}/`,strArray([]),8388608n,64n,8388608n,5000n,false));
   expect(value(await ws.sendText(client.session,"ok"))).toBe(2n);
   check(await ws.sendText(client.session,"x".repeat(8388609)),1336,{reason:"too_large"});
   check(await ws.sendBytes(client.session,ownBytes(new Uint8Array(8388609))),1336,{reason:"too_large"});
   // max_send_bytes 8 MiB: each 8 MiB send far exceeds a cold socket's
   // kernel take, so the buffer must still hold bytes when the next send
   // is attempted and one attempt in five is certain to observe block.
   let successes=0,blocked=false;
   for(let n=0;n<5&&!blocked;n++){
    const attempt=await ws.sendBytes(client.session,ownBytes(new Uint8Array(8388608)));
    if(attempt.kind==="ok"){expect(value(attempt)).toBe(8388608n);successes++;continue;}
    check(attempt,1336,{reason:"blocked"});blocked=true;
   }
   expect(blocked).toBe(true);
   const batch=dataArray(value(await reads.readMany(client.events,BigInt(successes+1))));
   expect(eventOf(batch[0])).toMatchObject({kind:TEXT,text:"ok"});
   for(let n=1;n<batch.length;n++)expect(copyBytes(eventOf(batch[n]).data,origin)).toHaveLength(8388608);
   expect(value(await ws.close(client.session,1000n,"done"))).toBe(undefined);
   expect(dataArray(value(await reads.readMany(client.events,2n)))).toHaveLength(1);
   expect(await reads.closeReader(client.events)).toMatchObject({kind:"ok"});
   // A locally closed session is terminal: sends observe resource-state.
   let thrown:unknown;try{await ws.sendText(client.session,"late");}catch(cause){thrown=cause;}
   expect(standardOf(thrown)).toBe("resource_state");
   thrown=undefined;try{await ws.close(client.session,1000n,"again");}catch(cause){thrown=cause;}
   expect(standardOf(thrown)).toBe("resource_state");
  });
 }finally{await route.close();}
});
test("close validates codes and reasons on the client before any frame",async()=>{
 const route=await echoServer();
 try{
  await owned(async()=>{
   const client=connectionOf(await ws.connect(`ws://127.0.0.1:${route.port}/`,strArray([]),65536n,64n,1048576n,5000n,false));
   for(const code of [999n,1004n,1005n,1006n,1015n,2000n,2999n,5000n])check(await ws.close(client.session,code,""),1337,{reason:"code"});
   check(await ws.close(client.session,1000n,"r".repeat(124)),1337,{reason:"reason"});
   check(await ws.close(client.session,1000n,"\ud800"),1337,{reason:"reason"});
   // Rejected closes leave the session usable; boundary values pass.
   expect(value(await ws.sendText(client.session,"alive"))).toBe(5n);
   expect(eventOf(dataArray(value(await reads.readMany(client.events,1n)))[0])).toMatchObject({kind:TEXT,text:"alive"});
   expect(value(await ws.close(client.session,3000n,"r".repeat(123)))).toBe(undefined);
   const last=dataArray(value(await reads.readMany(client.events,2n)));
   expect(eventOf(last[0])).toMatchObject({kind:CLOSE,code:3000n});
   expect(await reads.closeReader(client.events)).toMatchObject({kind:"ok"});
  });
 }finally{await route.close();}
});
test("oversized inbound messages fail the read terminally with cleanup intact",async()=>{
 const route=await echoServer({onMessage:(socket,message)=>{socket.send("way-too-long");}});
 try{
  await owned(async()=>{
   const client=connectionOf(await ws.connect(`ws://127.0.0.1:${route.port}/`,strArray([]),4n,64n,1048576n,5000n,false));
   expect(value(await ws.sendText(client.session,"go"))).toBe(2n);
   check(await reads.readMany(client.events,4n),1316,{reason:"message_too_large"});
   // Terminal read failure: close still succeeds, use observes state.
   expect(await reads.closeReader(client.events)).toMatchObject({kind:"ok"});
   let thrown:unknown;try{await reads.readMany(client.events,1n);}catch(cause){thrown=cause;}
   expect(standardOf(thrown)).toBe("resource_state");
   expect(value(await ws.close(client.session,1000n,"bye"))).toBe(undefined);
  });
 }finally{await route.close();}
});
test("an unread queue overrun fails instead of growing without bound",async()=>{
 const route=await echoServer({onMessage:(socket,message)=>{for(let n=0;n<6;n++)socket.send("filler-"+n);}});
 try{
  await owned(async()=>{
   const client=connectionOf(await ws.connect(`ws://127.0.0.1:${route.port}/`,strArray([]),65536n,2n,1048576n,5000n,false));
   expect(value(await ws.sendText(client.session,"burst"))).toBe(5n);
   // Six replies race the two-slot queue with no read pending: the pump
   // fails instead of retaining them all.
   await new Promise(resolve=>setTimeout(resolve,150));
   check(await reads.readMany(client.events,8n),1316,{reason:"queue_overrun"});
   expect(await reads.closeReader(client.events)).toMatchObject({kind:"ok"});
   expect(value(await ws.close(client.session,1000n,"bye"))).toBe(undefined);
  });
 }finally{await route.close();}
});
test("cancel interrupts a pending event read and two sessions stay apart",async()=>{
 let next=0;const seen=new Map<object,number>();
 const server=Bun.serve({hostname:"127.0.0.1",port:0,fetch(request,s){if(s.upgrade(request,{data:{tag:++next}}))return;return new Response("x",{status:400});},websocket:{message(socket,message){const tag=(socket.data as {tag:number}).tag;seen.set(socket,(seen.get(socket)??0)+1);socket.send(tag+":"+seen.get(socket)+":"+message);}}});
 try{
  await owned(async()=>{
   const first=connectionOf(await ws.connect(`ws://127.0.0.1:${server.port}/`,strArray([]),65536n,64n,1048576n,5000n,false));
   const second=connectionOf(await ws.connect(`ws://127.0.0.1:${server.port}/`,strArray([]),65536n,64n,1048576n,5000n,false));
   const pending=reads.readMany(first.events,1n);
   expect(await reads.cancelReader(first.events,"slow-consumer")).toMatchObject({kind:"ok"});
   check(await pending,1318,{reason:"slow-consumer"});
   expect(value(await ws.sendText(second.session,"ping"))).toBe(4n);
   const got=dataArray(value(await reads.readMany(second.events,1n)));
   expect(eventOf(got[0]).text).toMatch(/^2:1:ping$/);
   expect(value(await ws.sendText(second.session,"again"))).toBe(5n);
   const more=dataArray(value(await reads.readMany(second.events,1n)));
   expect(eventOf(more[0]).text).toMatch(/^2:2:again$/);
   expect(value(await ws.close(first.session,1000n,"one"))).toBe(undefined);
   expect(value(await ws.close(second.session,1000n,"two"))).toBe(undefined);
   const tail=dataArray(value(await reads.readMany(second.events,2n)));
   expect(eventOf(tail[0])).toMatchObject({kind:CLOSE,code:1000n,reason:"two"});
   expect(await reads.closeReader(second.events)).toMatchObject({kind:"ok"});
   let thrown:unknown;try{await reads.closeReader(first.events);}catch(cause){thrown=cause;}
   expect(standardOf(thrown)).toBe("resource_state");
  });
 }finally{await server.stop(true);}
});
test("accept upgrades inside the handler with an explicit protocol",async()=>{
 const observed:{protocol?:unknown;done?:boolean}={};
 await owned(async()=>{
  const route=await serveGet("/ws",async request=>{
   const conn=connectionOf(await ws.accept(request,"chat",65536n,64n,1048576n));
   observed.protocol=conn.protocol;
   for(;;){
    // Batch-fill semantics are uniform: trickling events read one at a
    // time, and the empty batch after the close event ends the loop.
    const batch=dataArray(value(await reads.readMany(conn.events,1n)));
    if(batch.length===0)break;
    for(const item of batch){
     const event=eventOf(item);
     if(event.kind===TEXT)value(await ws.sendText(conn.session,"echo:"+event.text));
     else if(event.kind===BINARY)value(await ws.sendBytes(conn.session,event.data));
    }
   }
   value(await reads.closeReader(conn.events));
   value(await ws.close(conn.session,1000n,"server-bye"));
   observed.done=true;
   return success(undefined);
  });
  try{
   const client=connectionOf(await ws.connect(`ws://127.0.0.1:${route.port}/ws`,strArray(["chat"]),65536n,64n,1048576n,5000n,false));
   expect(client.protocol).toBe("chat");expect(observed.protocol).toBe("chat");
   expect(value(await ws.sendText(client.session,"a"))).toBe(1n);
   expect(eventOf(dataArray(value(await reads.readMany(client.events,1n)))[0])).toMatchObject({kind:TEXT,text:"echo:a"});
   expect(value(await ws.sendBytes(client.session,ownBytes(new Uint8Array([7]))))).toBe(1n);
   const back=dataArray(value(await reads.readMany(client.events,1n)));
   expect(Array.from(copyBytes(eventOf(back[0]).data,origin))).toEqual([7]);
   expect(value(await ws.close(client.session,1000n,"client-bye"))).toBe(undefined);
   const tail=dataArray(value(await reads.readMany(client.events,2n)));
   // The initiator observes its own frame; the peer observed the wire one.
   expect(eventOf(tail[0])).toMatchObject({kind:CLOSE,code:1000n,reason:"client-bye"});
   expect(await reads.closeReader(client.events)).toMatchObject({kind:"ok"});
  await untilDone(observed);
  }finally{value(await route.stop());}
 });
});
test("a denied handshake stays ordinary HTTP with no socket opened",async()=>{
 await owned(async()=>{
  const route=await serveGet("/ws",async request=>{
   const headers=dataArray(value(await reqApi.headers(request)));
   const authed=headers.some(entry=>dataProperty(entry,"name")==="x-auth"&&dataProperty(entry,"value")==="yes");
   if(!authed)return success(await textResponse(401n,"denied"));
   // Unreached: the adapter client sends no custom headers, so every
   // handshake here stays a denied HTTP exchange by construction.
   return success(await textResponse(500n,"unexpected auth"));
  });
  try{
   const denied=await fetch(`http://127.0.0.1:${route.port}/ws`);
   expect(denied.status).toBe(401);expect(await denied.text()).toBe("denied");
   check(await ws.connect(`ws://127.0.0.1:${route.port}/ws`,strArray([]),65536n,64n,1048576n,5000n,false),1333,{reason:"rejected"});
  }finally{value(await route.stop());}
 });
});
test("accept validates selection, claims once and refuses plain HTTP",async()=>{
 const seen:{unsupported?:unknown;second?:unknown;done?:boolean}={};
 await owned(async()=>{
  const route=await serveGet("/ws",async request=>{
   const bad=await ws.accept(request,"nope",65536n,64n,1048576n);
   seen.unsupported=bad.kind==="domain"?domainFailureDiagnostics(bad.value).payload:bad.kind;
   const conn=connectionOf(await ws.accept(request,"",65536n,64n,1048576n));
   const again=await ws.accept(request,"",65536n,64n,1048576n);
   seen.second=again.kind==="domain"?domainFailureDiagnostics(again.value).payload:again.kind;
   const batch=dataArray(value(await reads.readMany(conn.events,1n)));
   if(eventOf(batch[0]).kind===TEXT)value(await ws.sendText(conn.session,"up"));
   const rest=dataArray(value(await reads.readMany(conn.events,2n)));
   expect(eventOf(rest[0]).kind).toBe(CLOSE);
   value(await reads.closeReader(conn.events));
   value(await ws.close(conn.session,1000n,"bye"));
   seen.done=true;
   return success(undefined);
  });
  try{
   const plain=await fetch(`http://127.0.0.1:${route.port}/ws`);
   expect(plain.status).toBe(500);
   const client=connectionOf(await ws.connect(`ws://127.0.0.1:${route.port}/ws`,strArray(["a"]),65536n,64n,1048576n,5000n,false));
   expect(client.protocol).toBe("a");
   expect(value(await ws.sendText(client.session,"hi"))).toBe(2n);
   expect(eventOf(dataArray(value(await reads.readMany(client.events,1n)))[0])).toMatchObject({kind:TEXT,text:"up"});
   expect(value(await ws.close(client.session,1000n,"done"))).toBe(undefined);
   expect(dataArray(value(await reads.readMany(client.events,2n)))).toHaveLength(1);
   expect(await reads.closeReader(client.events)).toMatchObject({kind:"ok"});
   expect(seen.unsupported).toMatchObject({protocol:"nope"});
   expect(seen.second).toMatchObject({reason:"already_upgraded"});
  await untilDone(seen);
  }finally{value(await route.stop());}
 });
});
test("server sends report accepted bytes and close validates where native does not",async()=>{
 const observed:{empty?:unknown;multi?:unknown;done?:boolean}={};
 await owned(async()=>{
  const route=await serveGet("/ws",async request=>{
   const conn=connectionOf(await ws.accept(request,"",65536n,64n,1048576n));
   observed.empty=value(await ws.sendText(conn.session,""));
   observed.multi=value(await ws.sendText(conn.session,"héllo€"));
   check(await ws.close(conn.session,999n,"bad"),1337,{reason:"code"});
   check(await ws.close(conn.session,1000n,"r".repeat(200)),1337,{reason:"reason"});
   const rest=dataArray(value(await reads.readMany(conn.events,2n)));
   expect(eventOf(rest[0]).kind).toBe(CLOSE);
   value(await reads.closeReader(conn.events));
   value(await ws.close(conn.session,1000n,"bye"));
   observed.done=true;
   return success(undefined);
  });
  try{
   const client=connectionOf(await ws.connect(`ws://127.0.0.1:${route.port}/ws`,strArray([]),65536n,64n,1048576n,5000n,false));
   const got=dataArray(value(await reads.readMany(client.events,2n)));
   expect(got.map(item=>eventOf(item).text)).toEqual(["","héllo€"]);
   expect(observed.empty).toBe(0n);expect(observed.multi).toBe(9n);
   expect(value(await ws.close(client.session,1000n,"done"))).toBe(undefined);
   expect(dataArray(value(await reads.readMany(client.events,2n)))).toHaveLength(1);
   expect(await reads.closeReader(client.events)).toMatchObject({kind:"ok"});
   await untilDone(observed);
  }finally{value(await route.stop());}
 });
});
test("saturated server sends fail fast and drain resumes them",async()=>{
 const observed:{blocked?:boolean;drained?:boolean;sent?:number;done?:boolean}={blocked:false,drained:false,sent:0};
 await owned(async()=>{
  const route=await serveGet("/ws",async request=>{
   const conn=connectionOf(await ws.accept(request,"",1048576n,64n,1048576n));
   const chunk=ownBytes(new Uint8Array(65536));
   for(let n=0;n<300&&!observed.blocked;n++){
    const attempt=await ws.sendBytes(conn.session,chunk);
    if(attempt.kind==="ok"){observed.sent!++;continue;}
    check(attempt,1336,{reason:"blocked"});observed.blocked=true;
   }
   expect(observed.blocked).toBe(true);
   for(;;){
    const batch=dataArray(value(await reads.readMany(conn.events,1n)));
    if(batch.length===0)break;
    if(eventOf(batch[0]).kind===DRAIN){observed.drained=true;break;}
   }
   expect(observed.drained).toBe(true);
   let resumed=false;
   for(let n=0;n<50&&!resumed;n++)resumed=(await ws.sendBytes(conn.session,chunk)).kind==="ok";
   expect(resumed).toBe(true);
   // A text marker ends the flood so the client needs no racy count.
   let marked=false;
   for(let n=0;n<200&&!marked;n++){
    marked=(await ws.sendText(conn.session,"done")).kind==="ok";
    if(!marked)await reads.readMany(conn.events,1n).catch(()=>[]);
   }
   expect(marked).toBe(true);
   const rest=dataArray(value(await reads.readMany(conn.events,2n)));
   expect(eventOf(rest[0]).kind).toBe(CLOSE);
   value(await reads.closeReader(conn.events));
   value(await ws.close(conn.session,1000n,"bye"));
   observed.done=true;
   return success(undefined);
  });
  try{
   const client=connectionOf(await ws.connect(`ws://127.0.0.1:${route.port}/ws`,strArray([]),1048576n,1024n,1048576n,5000n,false));
   let received=0,marked=false;
   while(!marked){
    for(const item of dataArray(value(await reads.readMany(client.events,1n)))){
     if(eventOf(item).kind===TEXT&&eventOf(item).text==="done")marked=true;else received++;
    }
   }
   expect(received).toBeGreaterThan(0);
   expect(value(await ws.close(client.session,1000n,"done"))).toBe(undefined);
   expect(dataArray(value(await reads.readMany(client.events,2n)))).toHaveLength(1);
   expect(await reads.closeReader(client.events)).toMatchObject({kind:"ok"});
   await untilDone(observed);
  }finally{value(await route.stop());}
 });
});
test("server stop fails active reads and prompts peers with 1001",async()=>{
 const observed:{stopped?:unknown;done?:boolean}={};
 await owned(async()=>{
  const route=await serveGet("/ws",async request=>{
   const conn=connectionOf(await ws.accept(request,"",65536n,64n,1048576n));
   const first=dataArray(value(await reads.readMany(conn.events,1n)));
   value(await ws.sendText(conn.session,"echo:"+eventOf(first[0]).text));
   const interrupted=await reads.readMany(conn.events,1n);
   observed.stopped=interrupted.kind==="domain"?domainFailureDiagnostics(interrupted.value).payload:interrupted.kind;
   value(await reads.closeReader(conn.events));
   value(await ws.close(conn.session,1000n,"bye"));
   observed.done=true;
   return success(undefined);
  });
  try{
   const client=connectionOf(await ws.connect(`ws://127.0.0.1:${route.port}/ws`,strArray([]),65536n,64n,1048576n,5000n,false));
   expect(value(await ws.sendText(client.session,"hi"))).toBe(2n);
   expect(eventOf(dataArray(value(await reads.readMany(client.events,1n)))[0])).toMatchObject({kind:TEXT,text:"echo:hi"});
   value(await route.stop());
   // Stop resolves after the closer runs, not after the handler observes
   // it, so wait for the handler's own record with a bounded poll.
   for(let n=0;n<300&&observed.stopped===undefined;n++)await new Promise(resolve=>setTimeout(resolve,10));
   expect(observed.stopped).toMatchObject({reason:"server_stopped"});
   const tail=dataArray(value(await reads.readMany(client.events,2n)));
   expect(eventOf(tail[0])).toMatchObject({kind:CLOSE,code:1001n,reason:"shutdown"});
   expect(value(await ws.close(client.session,1000n,"bye"))).toBe(undefined);
   expect(await reads.closeReader(client.events)).toMatchObject({kind:"ok"});
   await untilDone(observed);
  }finally{try{value(await route.stop());}catch{}}
 });
});
test("an abrupt peer surfaces as a 1006 close with an empty reason",async()=>{
 const observed:{code?:unknown;reason?:unknown}={};
 await owned(async()=>{
  const route=await serveGet("/ws",async request=>{
   const conn=connectionOf(await ws.accept(request,"",65536n,64n,1048576n));
   const batch=dataArray(value(await reads.readMany(conn.events,1n)));
   const event=eventOf(batch[0]);
   observed.code=event.code;observed.reason=event.reason;
   expect(event.kind).toBe(CLOSE);
   value(await reads.closeReader(conn.events));
   value(await ws.close(conn.session,1000n,"bye"));
   return success(undefined);
  });
  try{
   const net=await import("node:net");
   await new Promise<void>((resolve,reject)=>{
    const timer=setTimeout(()=>reject(Error("abrupt deadline")),5000);
    const raw=net.connect(route.port,"127.0.0.1",()=>{
     raw.write("GET /ws HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n");
    });
    let dropped=false;
    raw.on("data",()=>{if(!dropped){dropped=true;raw.destroy();clearTimeout(timer);setTimeout(resolve,300);}});
    raw.on("error",reject);
   });
   expect(observed.code).toBe(1006n);expect(observed.reason).toBe("");
  }finally{value(await route.stop());}
 });
});
test("session handles admit only their own kind",async()=>{
 const route=await echoServer();
 try{
  await owned(async()=>{
   const client=connectionOf(await ws.connect(`ws://127.0.0.1:${route.port}/`,strArray([]),65536n,64n,1048576n,5000n,false));
   expect(isWebSocketValue("ws-session",client.session)).toBe(true);
   expect(isWebSocketValue("ws-session",client.events)).toBe(false);
   expect(isWebSocketValue("ws-session",undefined)).toBe(false);
   expect(isWebSocketValue("stream-reader",client.session)).toBe(false);
   expect(isWebSocketValue(undefined,client.session)).toBe(false);
   expect(value(await ws.close(client.session,1000n,"bye"))).toBe(undefined);
   expect(dataArray(value(await reads.readMany(client.events,2n)))).toHaveLength(1);
   expect(await reads.closeReader(client.events)).toMatchObject({kind:"ok"});
  });
 }finally{await route.close();}
});
