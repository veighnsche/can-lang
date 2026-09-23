import {denyLiveBoundary,type AssertionContext} from "../../assert/context.ts";
import {success,failure,type Completion} from "../../completion.ts";
import {record} from "../../data.ts";
import {ownBytes,copyBytes,type Bytes} from "../../bytes.ts";
import {createDomainRuntime} from "../../domain.ts";
import type {FailureOrigin} from "../../failure.ts";
import {classifyFileError,validatePath} from "./errors.ts";
const origin:FailureOrigin=Object.freeze({source:"can:files-read",start:0,end:0,invocation:Object.freeze([])});
type Contracts=Readonly<{notFound:string;denied:string;invalidPath:string;unexpectedKind:string;limitExceeded:string;invalidData:string;ioError:string}>;
export function createFileReads(domain:ReturnType<typeof createDomainRuntime>,types:Contracts){
 const fail=(identity:string,fields:readonly(readonly[string,unknown])[],cause?:unknown)=>failure(domain.create(identity,record(identity,fields),origin,cause));
 function failureFor(cause:unknown,path:string,operation:string):Completion<never>{
  const classified=classifyFileError(cause);
  if(classified===undefined)throw cause;
  switch(classified.kind){
   case "not_found":return fail(types.notFound,[["path",path]],cause);
   case "denied":return fail(types.denied,[["path",path],["operation",operation]],cause);
   case "unexpected_kind":return fail(types.unexpectedKind,[["path",path],["operation",operation]],cause);
   case "not_directory":return fail(types.invalidPath,[["path",path],["reason","not_directory"]],cause);
   case "name_too_long":return fail(types.invalidPath,[["path",path],["reason","name_too_long"]],cause);
   default:return fail(types.ioError,[["path",path],["operation",operation]],cause);
  }
 }
 async function read(path:string,limit:bigint,operation:string):Promise<Completion<Bytes>>{
  const invalid=validatePath(path);
  if(invalid!==undefined)return fail(types.invalidPath,[["path",path],["reason",invalid]]);
  if(limit<0n)return fail(types.limitExceeded,[["limit",limit]]);
  let reader:ReadableStreamDefaultReader<Uint8Array>|undefined;
  let complete=false;
  try{
   reader=Bun.file(path).stream().getReader();
   const chunks:Uint8Array[]=[];let size=0n;
   for(;;){
    const next=await reader.read();if(next.done){complete=true;break;}
    const length=BigInt(next.value.byteLength);
    // Count before retaining so a growing file cannot evade the cap.
    if(length>limit-size)return fail(types.limitExceeded,[["limit",limit]]);
    size+=length;
    // The stream owns its views and may reuse them after the next read.
    if(length!==0n)chunks.push(new Uint8Array(next.value));
   }
   const result=new Uint8Array(Number(size));let offset=0;
   for(const chunk of chunks){result.set(chunk,offset);offset+=chunk.byteLength;}
   return success(ownBytes(result));
  }catch(cause){return failureFor(cause,path,operation);}
  finally{
   if(reader){
    // Observe cancellation rejection without replacing the original outcome.
    if(!complete)try{await reader.cancel();}catch{}
    reader.releaseLock();
   }
  }
 }
 return Object.freeze({
  async readBytes(path:string,maxBytes:bigint,context?:AssertionContext):Promise<Completion<Bytes>>{
   denyLiveBoundary(context,origin);return read(path,maxBytes,"read_bytes");
  },
  async readText(path:string,maxBytes:bigint,context?:AssertionContext):Promise<Completion<string>>{
   denyLiveBoundary(context,origin);
   const result=await read(path,maxBytes,"read_text");
   if(result.kind!=="ok")return result;
   const data=copyBytes(result.value,origin);
   try{return success(new TextDecoder("utf-8",{fatal:true,ignoreBOM:true}).decode(data));}
   catch(cause){if(!(cause instanceof TypeError))throw cause;return fail(types.invalidData,[["path",path],["reason","utf8"]]);}
  }
 });
}
