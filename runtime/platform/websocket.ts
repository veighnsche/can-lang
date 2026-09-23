// WebSocket client and server sessions over the B1-05 pull core. Inbound
// flow is a stream::reader over ws::event cells: text, binary, drain
// (server only, coalesced, never overruns) and close. Neither side emits
// open: connect resolves after the handshake and accept implies a live
// socket, with the negotiated protocol carried on the connection record.
// Failures fail the pending read as read_failed; error hooks without a
// close carry no provenance on this target, so close owns every terminal.
import {success,failure,type Completion,type AssertionContext} from "../completion.ts";
import {denyLiveBoundary} from "../assert/context.ts";
import {record,dataArray} from "../data.ts";
import {ownBytes,copyBytes,byteLength,isBytes} from "../bytes.ts";
import {createDomainRuntime} from "../domain.ts";
import {resourceStateFailure} from "../failure.ts";
import {registerResource,useResource,closeResource,resourceStatus} from "../owner.ts";
import {openEventCell} from "../transport/stream/readable.ts";
import {registerReader,type EventPump,type EventTake} from "../transport/stream/lifecycle.ts";
import {claimUpgrade,releaseUpgrade,offeredProtocols,isRequest} from "./http.ts";
const origin=Object.freeze({source:"can:websocket",start:0,end:0,invocation:Object.freeze([])});
export const SESSION_KIND="ws-session";
const MESSAGE_CAP=67108864n,QUEUE_CAP=1024n,SEND_CAP=67108864n,DEADLINE_CAP=2147483647n;
type EventIds=Readonly<{text:string;binary:string;drain:string;close:string;connection:string}>;
type ClientSocket=Readonly<{readyState:number;bufferedAmount:number;protocol:string;binaryType:string;send(payload:string|Uint8Array):void;close(code?:number,reason?:string):void}>;
type ServerSocket=Readonly<{readyState:number;data:unknown;send(payload:string|Uint8Array):number;close(code?:number,reason?:string):void}>;
type Session=Readonly<{id:number;kind:"client"|"server";maxMessage:bigint;maxSend:bigint;ids:EventIds;push:(value:unknown)=>void;pushDrain:(value:unknown)=>void;pushClose:(value:unknown)=>void;fail:(reason:string)=>void}>&{socket:ClientSocket|ServerSocket|undefined;server?:object;cleanClose:boolean};
// Server hooks are installed once per Bun.serve and select their session
// through typed data; client hooks close over their own session instead.
let nextSession=0;
const sessions=new Map<number,Session>(),serverSessions=new Map<object,Set<number>>();
function unregister(session:Session):void{
  sessions.delete(session.id);
  const owned=session.server===undefined?undefined:serverSessions.get(session.server);
  if(owned!==undefined){owned.delete(session.id);if(owned.size===0&&session.server!==undefined)serverSessions.delete(session.server);}
}
function sessionIdOf(socket:unknown):number|undefined{
  if(typeof socket!=="object"||socket===null)return undefined;
  const data=(socket as {data?:unknown}).data;
  if(typeof data!=="object"||data===null)return undefined;
  const id=(data as {session?:unknown}).session;
  return typeof id==="number"?id:undefined;
}
type QueueEntry=Readonly<{value:unknown;drain:boolean}>;
function createPump(maxQueued:number):Readonly<{pump:EventPump;push:(value:unknown)=>void;pushDrain:(value:unknown)=>void;pushClose:(value:unknown)=>void;fail:(reason:string)=>void}>{
  const queue:QueueEntry[]=[],waiters:((take:EventTake)=>void)[]=[];
  let terminal:EventTake|undefined,interrupted=false,disposed=false,drainPending=false;
  const wake=(take:EventTake):void=>{let waiter;while((waiter=waiters.shift())!==undefined)waiter(take);};
  const settle=(take:Extract<EventTake,{kind:"end"}|{kind:"failed"}>):void=>{
    if(terminal!==undefined)return;
    terminal=take;if(take.kind==="failed")queue.length=0;
    wake(take);
  };
  return {
    pump:Object.freeze({
      take:():Promise<EventTake>=>{
        const entry=queue.shift();
        if(entry!==undefined){if(entry.drain)drainPending=false;return Promise.resolve({kind:"event",value:entry.value});}
        if(terminal!==undefined)return Promise.resolve(terminal);
        // Disposed takes end: release runs only after every lease drains,
        // so reaching here means abandonment, never a live wait.
        if(disposed)return Promise.resolve({kind:"end"});
        if(interrupted)return Promise.resolve({kind:"interrupted"});
        return new Promise<EventTake>(resolve=>{waiters.push(resolve);});
      },
      interrupt:():void=>{interrupted=true;wake({kind:"interrupted"});},
      dispose:():void=>{disposed=true;queue.length=0;drainPending=false;},
    }),
    push:(value:unknown):void=>{
      if(terminal!==undefined||disposed)return;
      const waiter=waiters.shift();
      if(waiter!==undefined){waiter({kind:"event",value});return;}
      if(queue.length>=maxQueued){settle({kind:"failed",reason:"queue_overrun"});return;}
      queue.push({value,drain:false});
    },
    // Drain is coalesced and bypasses the data cap: at most one is ever
    // outstanding, and losing it would strand a blocked sender whose retry
    // signal never arrives.
    pushDrain:(value:unknown):void=>{
      if(terminal!==undefined||disposed||drainPending)return;
      const waiter=waiters.shift();
      if(waiter!==undefined){waiter({kind:"event",value});return;}
      drainPending=true;queue.push({value,drain:true});
    },
    // Close bypasses the data cap so a full queue still delivers its
    // terminal; queued data reads first, then the close, then the end.
    pushClose:(value:unknown):void=>{
      if(terminal!==undefined||disposed)return;
      const waiter=waiters.shift();
      if(waiter!==undefined){waiter({kind:"event",value});settle({kind:"end"});return;}
      queue.push({value,drain:false});settle({kind:"end"});
    },
    fail:(reason:string):void=>{if(!disposed)settle({kind:"failed",reason});},
  };
}
function utf8Bytes(text:string):bigint{return BigInt(new TextEncoder().encode(text).byteLength);}
const tokenPattern=/^[!#$%&'*+\-.^_`|~0-9A-Za-z]+$/;
function validProtocol(value:unknown):value is string{return typeof value==="string"&&value!==""&&tokenPattern.test(value);}
function validCode(value:bigint):boolean{
  if(value<1000n||value>4999n)return false;
  if(value>=1000n&&value<=1014n)return value!==1004n&&value!==1005n&&value!==1006n;
  return value>=3000n;
}
function messageIn(session:Session,message:unknown):void{
  if(typeof message==="string"){
    if(utf8Bytes(message)>session.maxMessage){session.fail("message_too_large");return;}
    session.push(record(session.ids.text,[["text",message]]));
    return;
  }
  const bytes=message instanceof ArrayBuffer?new Uint8Array(message):ArrayBuffer.isView(message)?new Uint8Array(message.buffer,message.byteOffset,message.byteLength):undefined;
  if(bytes===undefined){session.fail("message_type");return;}
  if(BigInt(bytes.byteLength)>session.maxMessage){session.fail("message_too_large");return;}
  session.push(record(session.ids.binary,[["data",ownBytes(bytes)]]));
}
function closeIn(session:Session,code:unknown,reason:unknown):void{
  const value=typeof code==="number"&&Number.isSafeInteger(code)?BigInt(code):-1n;
  session.pushClose(record(session.ids.close,[["code",value],["reason",typeof reason==="string"?reason:""]]));
}
function lookup(socket:unknown):Session|undefined{
  const id=sessionIdOf(socket);
  return id===undefined?undefined:sessions.get(id);
}
// Hook entry points for server.ts. They never throw into Bun and never
// enter Can: events enqueue for pull, and unknown or late sockets drop.
export function serverSocketOpen(socket:unknown):void{
  try{const session=lookup(socket);if(session!==undefined)session.socket=socket as ServerSocket;}catch{}
}
export function serverSocketMessage(socket:unknown,message:unknown):void{
  try{const session=lookup(socket);if(session!==undefined)messageIn(session,message);}catch{}
}
export function serverSocketClose(socket:unknown,code:unknown,reason:unknown):void{
  try{
    const session=lookup(socket);if(session===undefined)return;
    unregister(session);closeIn(session,code,reason);
  }catch{}
}
export function serverSocketDrain(socket:unknown):void{
  try{const session=lookup(socket);if(session!==undefined)session.pushDrain(record(session.ids.drain,[]));}catch{}
}
// Server stop prompts every live session with a 1001 frame and fails its
// pump: active reads observe server_stopped deterministically instead of
// hanging on peers that never echo.
export function closeServerSessions(server:object):void{
  const owned=serverSessions.get(server);
  if(owned===undefined)return;
  for(const id of [...owned]){
    const session=sessions.get(id);
    if(session===undefined){owned.delete(id);continue;}
    unregister(session);
    try{session.socket?.close(1001,"shutdown");}catch{}
    session.fail("server_stopped");
  }
  if(owned.size===0)serverSessions.delete(server);
}
export function isWebSocketValue(kind:string|undefined,value:unknown):boolean{
  if(kind!==SESSION_KIND)return false;
  try{return resourceStatus(value).kind===SESSION_KIND;}catch{return false;}
}
type Types=Readonly<{connectFailed:string;upgradeFailed:string;unsupportedProtocol:string;sendFailed:string;invalidClose:string;limitExceeded:string;invalidUrl:string;invalidProtocol:string;closeFailed:string}&EventIds>;
export function createWebSockets(domain:ReturnType<typeof createDomainRuntime>,types:Types){
  const fail=(identity:string,fields:readonly(readonly[string,unknown])[],cause?:unknown)=>failure(domain.create(identity,record(identity,fields),origin,cause));
  const overLimit=(value:bigint)=>fail(types.limitExceeded,[["limit",value]]);
  function checkCaps(maxMessage:unknown,maxQueued:unknown,maxSend:unknown):Readonly<{maxMessage:bigint;maxQueued:bigint;maxSend:bigint}|Completion<never>>{
    if(typeof maxMessage!=="bigint"||maxMessage<1n||maxMessage>MESSAGE_CAP)return overLimit(typeof maxMessage==="bigint"?maxMessage:-1n);
    if(typeof maxQueued!=="bigint"||maxQueued<1n||maxQueued>QUEUE_CAP)return overLimit(typeof maxQueued==="bigint"?maxQueued:-1n);
    if(typeof maxSend!=="bigint"||maxSend<1n||maxSend>SEND_CAP)return overLimit(typeof maxSend==="bigint"?maxSend:-1n);
    return {maxMessage,maxQueued,maxSend};
  }
  function register(caps:Readonly<{maxMessage:bigint;maxQueued:bigint;maxSend:bigint}>,kind:"client"|"server",server?:object):Readonly<{session:Session;sessionToken:object;readerToken:object}>{
    const pump=createPump(Number(caps.maxQueued));
    const session:Session={id:++nextSession,kind,maxMessage:caps.maxMessage,maxSend:caps.maxSend,ids:types,socket:undefined,server,cleanClose:false,push:pump.push,pushDrain:pump.pushDrain,pushClose:pump.pushClose,fail:pump.fail};
    sessions.set(session.id,session);
    if(server!==undefined){let owned=serverSessions.get(server);if(owned===undefined){owned=new Set();serverSessions.set(server,owned);}owned.add(session.id);}
    const readerToken=registerReader(openEventCell(pump.pump),(identity,fields,cause)=>fail(identity,fields,cause),types.closeFailed,{scopeManaged:true});
    const sessionToken=registerResource(SESSION_KIND,session,async():Promise<Completion<void>>=>{
      // Explicit close keeps the registry so the peer echo still lands;
      // abandonment and teardown unregister because no echo is awaited.
      const clean=session.cleanClose;session.cleanClose=false;
      try{session.socket?.close();}catch{}
      if(!clean)unregister(session);
      return success(undefined);
    });
    return {session,sessionToken,readerToken};
  }
  async function teardown(request:unknown,held:Readonly<{session:Session;sessionToken:object;readerToken:object}>):Promise<void>{
    try{await closeResource(held.readerToken,"stream-reader");}catch{}
    try{await closeResource(held.sessionToken,SESSION_KIND);}catch{}
    releaseUpgrade(request);
  }
  function sendOn(session:Session,payload:string|Uint8Array,bytes:bigint):Completion<bigint>{
    const socket=session.socket;
    if(socket===undefined||socket.readyState!==1)return fail(types.sendFailed,[["reason","closed"]]);
    if(session.kind==="client"&&(socket as ClientSocket).bufferedAmount+Number(bytes)>Number(session.maxSend))return fail(types.sendFailed,[["reason","blocked"]]);
    try{
      if(session.kind==="server"){
        const accepted=(socket as ServerSocket).send(payload);
        if(accepted===-1)return fail(types.sendFailed,[["reason","blocked"]]);
        return success(BigInt(accepted));
      }
      (socket as ClientSocket).send(payload);
      return success(bytes);
    }catch{return fail(types.sendFailed,[["reason","closed"]]);}
  }
  return Object.freeze({
    async connect(url:unknown,protocols:unknown,maxMessage:unknown,maxQueued:unknown,maxSend:unknown,deadlineMs:unknown,insecureTls:unknown,context?:AssertionContext):Promise<Completion<unknown>>{
      denyLiveBoundary(context,origin);
      if(typeof url!=="string")throw new TypeError("invalid compiler url");
      let parsed:URL;
      try{parsed=new URL(url);}catch{return fail(types.invalidUrl,[["reason","unparseable"]]);}
      if(parsed.protocol!=="ws:"&&parsed.protocol!=="wss:")return fail(types.invalidUrl,[["reason","scheme"]]);
      const offered:string[]=[];
      for(const entry of dataArray(protocols)){
        if(!validProtocol(entry))return fail(types.invalidProtocol,[["protocol",typeof entry==="string"?entry:""]]);
        offered.push(entry);
      }
      const caps=checkCaps(maxMessage,maxQueued,maxSend);if("kind" in caps)return caps;
      if(typeof deadlineMs!=="bigint"||deadlineMs<1n||deadlineMs>DEADLINE_CAP)return overLimit(typeof deadlineMs==="bigint"?deadlineMs:-1n);
      if(typeof insecureTls!=="boolean")throw new TypeError("invalid compiler insecure flag");
      const insecure=insecureTls&&parsed.protocol==="wss:";
      let socket:WebSocket;
      try{
        if(insecure)socket=new WebSocket(parsed.href,{...(offered.length===0?{}:{protocols:offered}),tls:{rejectUnauthorized:false}} as unknown as string[]);
        else socket=offered.length===0?new WebSocket(parsed.href):new WebSocket(parsed.href,offered);
      }catch{return fail(types.invalidUrl,[["reason","unparseable"]]);}
      try{socket.binaryType="arraybuffer";}catch{try{socket.close();}catch{}return fail(types.connectFailed,[["reason","unreachable"]]);}
      const opened=await new Promise<Readonly<{kind:"open"}|{kind:"failed";reason:string}>>(resolve=>{
        const timer=setTimeout(()=>{try{socket.close();}catch{}resolve({kind:"failed",reason:"timeout"});},Number(deadlineMs));
        socket.onopen=()=>{clearTimeout(timer);resolve({kind:"open"});};
        socket.onerror=()=>{};
        socket.onclose=(event:CloseEvent)=>{
          clearTimeout(timer);
          const code=event.code;
          resolve({kind:"failed",reason:code===1002?"rejected":code===1006?"unreachable":code===1015?"tls":"closed"});
        };
      });
      if(opened.kind==="failed")return fail(types.connectFailed,[["reason",opened.reason]]);
      const held=register(caps,"client");
      held.session.socket=socket as unknown as ClientSocket;
      socket.onmessage=(event:MessageEvent)=>{try{messageIn(held.session,event.data);}catch{}};
      socket.onclose=(event:CloseEvent)=>{try{closeIn(held.session,event.code,event.reason);}catch{}};
      socket.onerror=()=>{};
      return success(record(types.connection,[["session",held.sessionToken],["events",held.readerToken],["protocol",socket.protocol]]));
    },
    async accept(request:unknown,protocol:unknown,maxMessage:unknown,maxQueued:unknown,maxSend:unknown,context?:AssertionContext):Promise<Completion<unknown>>{
      denyLiveBoundary(context,origin);
      if(!isRequest(request))throw resourceStateFailure(undefined,origin);
      if(typeof protocol!=="string")throw new TypeError("invalid compiler protocol");
      const caps=checkCaps(maxMessage,maxQueued,maxSend);if("kind" in caps)return caps;
      if(protocol!==""&&!validProtocol(protocol))return fail(types.invalidProtocol,[["protocol",protocol]]);
      if(protocol!==""&&!offeredProtocols(request).includes(protocol))return fail(types.unsupportedProtocol,[["protocol",protocol]]);
      const claimed=claimUpgrade(request);
      if(claimed===undefined)throw resourceStateFailure(undefined,origin);
      if(claimed==="upgraded")return fail(types.upgradeFailed,[["reason","already_upgraded"]]);
      const held=register(caps,"server",claimed.server as object);
      let upgraded=false;
      try{
        upgraded=claimed.server.upgrade(claimed.native,protocol===""?{data:{session:held.session.id}}:{headers:{"Sec-WebSocket-Protocol":protocol},data:{session:held.session.id}});
      }catch{upgraded=false;}
      if(!upgraded){await teardown(request,held);return fail(types.upgradeFailed,[["reason","refused"]]);}
      return success(record(types.connection,[["session",held.sessionToken],["events",held.readerToken],["protocol",protocol]]));
    },
    async sendText(session:unknown,text:unknown,context?:AssertionContext):Promise<Completion<bigint>>{
      denyLiveBoundary(context,origin);
      if(typeof text!=="string")throw new TypeError("invalid compiler text");
      return useResource(session,SESSION_KIND,async native=>{
        const current=native as Session;
        const bytes=utf8Bytes(text);
        if(bytes>current.maxMessage)return fail(types.sendFailed,[["reason","too_large"]]);
        return sendOn(current,text,bytes);
      });
    },
    async sendBytes(session:unknown,body:unknown,context?:AssertionContext):Promise<Completion<bigint>>{
      denyLiveBoundary(context,origin);
      if(!isBytes(body))throw new TypeError("invalid compiler bytes");
      return useResource(session,SESSION_KIND,async native=>{
        const current=native as Session;
        const bytes=byteLength(body);
        if(bytes>current.maxMessage)return fail(types.sendFailed,[["reason","too_large"]]);
        return sendOn(current,new Uint8Array(copyBytes(body,origin)),bytes);
      });
    },
    async close(session:unknown,code:unknown,reason:unknown,context?:AssertionContext):Promise<Completion<undefined>>{
      denyLiveBoundary(context,origin);
      if(typeof code!=="bigint"||typeof reason!=="string")throw new TypeError("invalid compiler close");
      if(!validCode(code))return fail(types.invalidClose,[["reason","code"]]);
      if(!reason.isWellFormed()||utf8Bytes(reason)>123n)return fail(types.invalidClose,[["reason","reason"]]);
      // The frame goes out under a lease, then the handle closes outside
      // it: closing inside useResource would wait on its own lease.
      await useResource(session,SESSION_KIND,async native=>{
        const current=native as Session;
        try{current.socket?.close(Number(code),reason);}catch{}
        current.cleanClose=true;
        return success(undefined);
      });
      const completion=await closeResource(session,SESSION_KIND);
      return completion.kind==="ok"?success(undefined):completion;
    },
  });
}
