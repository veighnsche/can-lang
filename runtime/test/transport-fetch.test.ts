import {test,expect} from "bun:test";
import {performRequest} from "../transport/fetch.ts";
import {transportProblem} from "../transport/deadline.ts";
import {runOwnedRoot} from "../owner.ts";
import {success,value} from "../completion.ts";
import type {Connection} from "../transport/request.ts";
const base={timeoutMilliseconds:1000,maxBodyBytes:100,headers:[]};
const request={path:"/",method:"GET" as const,query:[],headers:[]};
const decode=(bytes:Uint8Array)=>success(new TextDecoder().decode(bytes));
async function root<T>(body:()=>Promise<T>):Promise<T>{return value((await runOwnedRoot(async()=>success(await body()))).completion);}

test("native fetch makes one attempt, preserves request policy, and has no cookie jar",async()=>{
 const seen:{url:string;cookie:string|null;auth:string|null}[]=[];
 const server=Bun.serve({hostname:"127.0.0.1",port:0,fetch(req){seen.push({url:req.url,cookie:req.headers.get("cookie"),auth:req.headers.get("authorization")});return new Response("ok",{headers:{"set-cookie":"session=secret"}});}});
 const connection:Connection={...base,endpoint:server.url.href,bearerEnvironment:"TOKEN"};let reads=0;
 try{
  await root(async()=>{
   expect(value(await performRequest(connection,{...request,query:[{name:"q",value:["one","two"]}]},()=>{reads++;return "test-token";},decode))).toBe("ok");
   expect(value(await performRequest(connection,request,()=>{reads++;return "test-token";},decode))).toBe("ok");return undefined;
  });
  expect(reads).toBe(2);expect(seen.length).toBe(2);expect(seen[0].url).toEndWith("/?q=one&q=two");expect(seen.every(r=>r.cookie===null&&r.auth==="Bearer test-token")).toBe(true);
 }finally{server.stop(true);}
});
test("manual redirect returns status without entering decoder or redirect target",async()=>{
 let attempts=0,decoded=false;
 const server=Bun.serve({hostname:"127.0.0.1",port:0,fetch(){attempts++;return new Response("redirect",{status:302,headers:{location:"/next"}});}});
 try{await root(async()=>{
  try{await performRequest({...base,endpoint:server.url.href},request,()=>undefined,()=>{decoded=true;return success(undefined);});throw new Error("accepted redirect");}
  catch(cause){expect(transportProblem(cause)).toMatchObject({kind:"status",status:302});}
  expect(value(await performRequest({...base,endpoint:server.url.href},{...request,envelope:true},()=>undefined,(_,meta)=>success(meta.status)))).toBe(302);
  return undefined;
 });expect(attempts).toBe(2);expect(decoded).toBe(false);}finally{server.stop(true);}
});
test("stalled native response times out, aborts body, and never enters decode",async()=>{
 let attempts=0,decoded=false;
 const server=Bun.serve({hostname:"127.0.0.1",port:0,fetch(){attempts++;return new Response(new ReadableStream({start(c){c.enqueue(new Uint8Array([1]));}}));}});
 try{await root(async()=>{
  try{await performRequest({...base,endpoint:server.url.href,timeoutMilliseconds:30},request,()=>undefined,()=>{decoded=true;return success(undefined);});throw new Error("accepted stall");}
  catch(cause){expect(transportProblem(cause)).toEqual({kind:"timeout"});}return undefined;
 });expect(attempts).toBe(1);expect(decoded).toBe(false);}finally{server.stop(true);}
});
test("response decode past the total deadline cannot enter the handler",async()=>{
 let handled=false;
 const server=Bun.serve({hostname:"127.0.0.1",port:0,fetch(){return new Response("ok");}});
 try{await root(async()=>{
  try{await performRequest({...base,endpoint:server.url.href,timeoutMilliseconds:30},request,()=>undefined,()=>{const start=performance.now();while(performance.now()-start<40){}return success("late");});handled=true;}
  catch(cause){expect(transportProblem(cause)).toEqual({kind:"timeout"});}return undefined;
 });expect(handled).toBe(false);}finally{server.stop(true);}
});

test("owner cancellation has its exact transport phase",async()=>{
 const controller=new AbortController();controller.abort();let entered=false;
 await root(async()=>{
  try{await performRequest({...base,endpoint:"http://127.0.0.1:1"},{...request,ownerSignal:controller.signal},()=>undefined,()=>{entered=true;return success(undefined);});throw new Error("accepted cancelled request");}
  catch(cause){expect(transportProblem(cause)).toEqual({kind:"transport",phase:"cancelled"});}return undefined;
 });expect(entered).toBe(false);
});

test("delivered decompressed bytes are bounded and outgoing overflow prevents auth and launch",async()=>{
 let attempts=0,reads=0,decoded=false;
 const server=Bun.serve({hostname:"127.0.0.1",port:0,fetch(){attempts++;return new Response(Bun.gzipSync(new TextEncoder().encode("x".repeat(1000))),{headers:{"content-encoding":"gzip"}});}});
 try{await root(async()=>{
  const connection={...base,endpoint:server.url.href,maxBodyBytes:50,bearerEnvironment:"TOKEN"};
  try{await performRequest(connection,{...request,method:"POST",body:new Uint8Array(51)},()=>{reads++;return "secret";},decode);throw new Error("accepted outgoing overflow");}
  catch(cause){expect(transportProblem(cause)).toEqual({kind:"limit",limit:50});}
  expect(reads).toBe(0);expect(attempts).toBe(0);
  try{await performRequest(connection,request,()=>{reads++;return "secret";},()=>{decoded=true;return success(undefined);});throw new Error("accepted decoded overflow");}
  catch(cause){expect(transportProblem(cause)).toEqual({kind:"limit",limit:50});}
  return undefined;
 });expect(attempts).toBe(1);expect(reads).toBe(1);expect(decoded).toBe(false);}finally{server.stop(true);}
});

test("decoded then-named data stays boxed across transport promises",async()=>{
 let assimilated=false;
 const data=Object.freeze({then:()=>{assimilated=true;}});
 const server=Bun.serve({hostname:"127.0.0.1",port:0,fetch(){return new Response("ok");}});
 try{await root(async()=>{
  const result=await performRequest({...base,endpoint:server.url.href},request,()=>undefined,()=>success(data));
  expect(value(result)).toBe(data);return undefined;
 });expect(assimilated).toBe(false);}finally{server.stop(true);}
});
