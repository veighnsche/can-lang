// Owned reader and consumption. Exactly one native pull per delivered
// batch step: no prefetch queue sits between the caller and the source,
// so read demand is the only read-side backpressure signal. Bytes cross
// into immutable values through copying constructors only; no reusable
// native chunk alias reaches Can state. Item kinds: bytes items are
// source chunks split at max_chunk; text items are \n-framed lines with
// one trailing \r stripped (CRLF tolerance) and a final unterminated
// segment still delivered as a line. Decoding is fatal UTF-8, matching
// whole-file reads; malformed input fails the read, never substitutes.
// A read interrupted by cancel reports cancelled and delivers nothing:
// partial items never cross together with a terminal failure.
import {denyLiveBoundary,type AssertionContext} from "../../assert/context.ts";
import {failure,success,type Completion} from "../../completion.ts";
import {array,record} from "../../data.ts";
import {ownBytes} from "../../bytes.ts";
import {createDomainRuntime} from "../../domain.ts";
import type {FailureOrigin} from "../../failure.ts";
import {validMaxItems,overCap} from "./budget.ts";
import {READER_KIND,registerReader,useReader,closeHandle,cancelHandle,type Fail,type ReaderCell,type ByteSource} from "./lifecycle.ts";
const origin:FailureOrigin=Object.freeze({source:"can:stream-read",start:0,end:0,invocation:Object.freeze([])});
const CHUNK_CLAMP=2n**31n;
type Contracts=Readonly<{readFailed:string;cancelled:string;closeFailed:string;limitExceeded:string}>;
function reasonFor(cause:unknown):string{
  // A dropped connection aborts pending reads; file and process sources
  // never raise AbortError, so the mapping stays source-faithful.
  if(typeof cause==="object"&&cause!==null){
    if((cause as {name?:unknown}).name==="AbortError")return "aborted";
    if(typeof (cause as {code?:unknown}).code==="string")return (cause as {code:string}).code;
  }
  return "io_error";
}
function utf8Length(value:string):bigint{
  return BigInt(new TextEncoder().encode(value).byteLength);
}
async function pullBytes(cell:ReaderCell,maxItems:bigint,fail:Fail,contracts:Contracts):Promise<Completion<readonly unknown[]>>{
  const items:unknown[]=[];
  const cap=cell.maxChunk>CHUNK_CLAMP?Number(CHUNK_CLAMP):Number(cell.maxChunk);
  for(;;){
    while(cell.carry!==undefined&&cell.carry.byteLength>0&&BigInt(items.length)<maxItems){
      const take=cell.carry.subarray(0,Math.min(cap,cell.carry.byteLength));
      const rest=cell.carry.byteLength-take.byteLength;
      cell.carry=rest>0?cell.carry.subarray(take.byteLength):undefined;
      items.push(ownBytes(take));
    }
    if(BigInt(items.length)>=maxItems||cell.ended===true)return success(array(items));
    let next;
    try{next=await cell.reader.read();}
    catch(cause){cell.errored=true;cell.carry=undefined;return fail(contracts.readFailed,[["reason",reasonFor(cause)]],cause);}
    if(next.done){
      cell.ended=true;
      if(cell.cancelled!==undefined)return fail(contracts.cancelled,[["reason",cell.cancelled]]);
      continue;
    }
    if(next.value.byteLength===0)continue;
    const view=new Uint8Array(next.value);
    cell.carry=cell.carry===undefined?view:concat(cell.carry,view);
  }
}
function concat(head:Uint8Array,tail:Uint8Array):Uint8Array{
  const merged=new Uint8Array(head.byteLength+tail.byteLength);
  merged.set(head,0);merged.set(tail,head.byteLength);
  return merged;
}
async function pullLines(cell:ReaderCell,maxItems:bigint,fail:Fail,contracts:Contracts):Promise<Completion<readonly unknown[]>>{
  const framing=cell.framing!;
  const items:unknown[]=[];
  for(;;){
    const newline=framing.carry.indexOf("\n");
    if(newline>=0){
      let line=framing.carry.slice(0,newline);
      framing.carry=framing.carry.slice(newline+1);
      if(line.endsWith("\r"))line=line.slice(0,-1);
      if(utf8Length(line)>cell.maxLine){cell.errored=true;return fail(contracts.limitExceeded,[["limit",cell.maxLine]]);}
      framing.pendingBytes=utf8Length(framing.carry);
      items.push(line);
      if(BigInt(items.length)>=maxItems)return success(array(items));
      continue;
    }
    if(cell.ended===true){
      if(framing.carry.length>0||framing.pendingBytes>0n){
        let tail: string;
        try{tail=framing.decoder.decode();}
        catch(cause){cell.errored=true;return fail(contracts.readFailed,[["reason","utf8"]],cause);}
        const line=framing.carry+tail;
        framing.carry="";framing.pendingBytes=0n;
        const clean=line.endsWith("\r")?line.slice(0,-1):line;
        if(utf8Length(clean)>cell.maxLine){cell.errored=true;return fail(contracts.limitExceeded,[["limit",cell.maxLine]]);}
        items.push(clean);
      }
      return success(array(items));
    }
    if(framing.pendingBytes>cell.maxLine){cell.errored=true;return fail(contracts.limitExceeded,[["limit",cell.maxLine]]);}
    let next;
    try{next=await cell.reader.read();}
    catch(cause){cell.errored=true;return fail(contracts.readFailed,[["reason",reasonFor(cause)]],cause);}
    if(next.done){
      cell.ended=true;
      if(cell.cancelled!==undefined)return fail(contracts.cancelled,[["reason",cell.cancelled]]);
      continue;
    }
    if(next.value.byteLength===0)continue;
    let text: string;
    try{text=framing.decoder.decode(next.value,{stream:true});}
    catch(cause){cell.errored=true;return fail(contracts.readFailed,[["reason","utf8"]],cause);}
    framing.carry+=text;
    framing.pendingBytes+=BigInt(next.value.byteLength);
  }
}
export function createStreamReads(domain:ReturnType<typeof createDomainRuntime>,types:Contracts){
  const fail:Fail=(identity,fields,cause)=>failure(domain.create(identity,record(identity,fields),origin,cause));
  return Object.freeze({
    async readMany(token:unknown,maxItems:bigint,context?:AssertionContext):Promise<Completion<readonly unknown[]>>{
      denyLiveBoundary(context,origin);
      if(!validMaxItems(maxItems))return fail(types.limitExceeded,[["limit",typeof maxItems==="bigint"?maxItems:-1n]]);
      return useReader(token,async cell=>{
        if(cell.item==="text"){
          if(cell.framing===undefined)throw new TypeError("invalid text reader cell");
          return pullLines(cell,maxItems,fail,types);
        }
        return pullBytes(cell,maxItems,fail,types);
      });
    },
    async closeReader(token:unknown,context?:AssertionContext):Promise<Completion<void>>{
      denyLiveBoundary(context,origin);
      return closeHandle(token,READER_KIND,fail,types.closeFailed);
    },
    async cancelReader(token:unknown,reason:string,context?:AssertionContext):Promise<Completion<void>>{
      denyLiveBoundary(context,origin);
      return cancelHandle(token,READER_KIND,reason,fail,types.closeFailed,async cell=>{
        if("reader" in cell)await cell.reader.cancel();
      });
    },
  });
}
// Internal bounded drain for producers that consume a whole pipe without
// exposing the reader (process stdout/stderr). Same pull/copy/release
// discipline as readMany; overflow short-circuits before retaining and
// native read failures reject so the caller keeps its failure contract.
export async function drainStream(stream:ByteSource,byteCap:bigint,fail:Fail,terminalFailed:string):Promise<Readonly<{overflow:boolean;data:Uint8Array}>>{
  const cell:ReaderCell={reader:stream.getReader(),item:"bytes",maxChunk:CHUNK_CLAMP,maxLine:CHUNK_CLAMP};
  // Internal handle: scope drain may win the terminal race against the
  // explicit close below, and that auto-cleanup is by design, not a leak.
  const token=registerReader(cell,fail,terminalFailed,{idempotent:true,scopeManaged:true});
  const chunks:Uint8Array[]=[];let size=0n;let complete=false;
  try{
    for(;;){
      const next=await cell.reader.read();
      if(next.done){complete=true;break;}
      const length=BigInt(next.value.byteLength);
      if(overCap(size,length,byteCap))return {overflow:true,data:new Uint8Array(0)};
      size+=length;
      if(length!==0n)chunks.push(new Uint8Array(next.value));
    }
    const data=new Uint8Array(Number(size));let offset=0;
    for(const chunk of chunks){data.set(chunk,offset);offset+=chunk.byteLength;}
    return {overflow:false,data};
  }finally{
    cell.ended=complete;
    await closeHandle(token,READER_KIND,fail,terminalFailed);
  }
}
export function openByteCell(stream:ByteSource,maxChunk:bigint):ReaderCell{
  return {reader:stream.getReader(),item:"bytes",maxChunk,maxLine:CHUNK_CLAMP};
}
export function openLineCell(stream:ByteSource,maxLine:bigint):ReaderCell{
  return {reader:stream.getReader(),item:"text",maxChunk:CHUNK_CLAMP,maxLine,framing:{decoder:new TextDecoder("utf-8",{fatal:true,ignoreBOM:true}),carry:"",pendingBytes:0n}};
}
