import {writeFile,copyFile,rename,lstat,stat,constants} from "node:fs/promises";
import {dirname} from "node:path";
import {denyLiveBoundary,type AssertionContext} from "../../assert/context.ts";
import {success,failure,type Completion} from "../../completion.ts";
import {record} from "../../data.ts";
import {copyBytes,type Bytes} from "../../bytes.ts";
import {createDomainRuntime} from "../../domain.ts";
import type {FailureOrigin} from "../../failure.ts";
import {classifyFileError,validatePath} from "./errors.ts";
const origin:FailureOrigin=Object.freeze({source:"can:files-write",start:0,end:0,invocation:Object.freeze([])});
const encoder=new TextEncoder();
type Contracts=Readonly<{notFound:string;alreadyExists:string;denied:string;invalidPath:string;unexpectedKind:string;notEmpty:string;crossDevice:string;ioError:string}>;
export function createFileWrites(domain:ReturnType<typeof createDomainRuntime>,types:Contracts){
 const fail=(identity:string,fields:readonly(readonly[string,unknown])[],cause?:unknown)=>failure(domain.create(identity,record(identity,fields),origin,cause));
 function failureFor(cause:unknown,path:string,operation:string):Completion<never>{
  const classified=classifyFileError(cause);
  if(classified===undefined)throw cause;
  switch(classified.kind){
   case "not_found":return fail(types.notFound,[["path",path]],cause);
   case "already_exists":return fail(types.alreadyExists,[["path",path]],cause);
   case "denied":return fail(types.denied,[["path",path],["operation",operation]],cause);
   case "unexpected_kind":return fail(types.unexpectedKind,[["path",path],["operation",operation]],cause);
   case "not_empty":return fail(types.notEmpty,[["path",path]],cause);
   case "not_directory":return fail(types.invalidPath,[["path",path],["reason","not_directory"]],cause);
   case "name_too_long":return fail(types.invalidPath,[["path",path],["reason","name_too_long"]],cause);
   default:return fail(types.ioError,[["path",path],["operation",operation]],cause);
  }
 }
 function invalid(path:string):Completion<never>|undefined{
  const reason=validatePath(path);
  if(reason===undefined)return undefined;
  return fail(types.invalidPath,[["path",path],["reason",reason]]);
 }
 // Best-effort attribution for two-path operations: when the destination
 // parent is itself missing, the destination is the absent side.
 async function missingSide(source:string,destination:string):Promise<string>{
  try{await lstat(dirname(destination));}catch(cause){
   if(classifyFileError(cause)?.kind==="not_found")return destination;
  }
  return source;
 }
 async function copyOrMove(kind:"copy"|"move",source:string,destination:string,overwrite:boolean):Promise<Completion<undefined>>{
  const bad=invalid(source)??invalid(destination);
  if(bad!==undefined)return bad;
  if(kind==="move"&&!overwrite){
   try{await lstat(destination);return fail(types.alreadyExists,[["path",destination]]);}
   catch(cause){if(classifyFileError(cause)?.kind!=="not_found")return failureFor(cause,destination,"move");}
  }
  if(kind==="copy"){
   // copyFile rejects directories with platform-specific codes (EISDIR on
   // Linux, ENOTSUP on macOS); detect the kind directly for one contract.
   try{
    if((await stat(source)).isDirectory())return fail(types.unexpectedKind,[["path",source],["operation","copy"]]);
   }catch(cause){
    const classified=classifyFileError(cause);
    if(classified===undefined)throw cause;
    if(classified.kind==="not_found")return fail(types.notFound,[["path",source]],cause);
    return failureFor(cause,source,"copy");
   }
  }
  try{
   if(kind==="copy")await copyFile(source,destination,overwrite?undefined:constants.COPYFILE_EXCL);
   else await rename(source,destination);
   return success(undefined);
  }catch(cause){
   const classified=classifyFileError(cause);
   if(classified===undefined)throw cause;
   if(classified.kind==="not_found")return fail(types.notFound,[["path",await missingSide(source,destination)]],cause);
   if(classified.kind==="already_exists")return fail(types.alreadyExists,[["path",destination]],cause);
   if(classified.kind==="cross_device")return fail(types.crossDevice,[["source",source],["destination",destination]],cause);
   return failureFor(cause,kind==="copy"?source:destination,kind);
  }
 }
 return Object.freeze({
  async writeBytes(path:string,value:Bytes,overwrite:boolean,context?:AssertionContext):Promise<Completion<undefined>>{
   denyLiveBoundary(context,origin);
   const bad=invalid(path);
   if(bad!==undefined)return bad;
   const data=copyBytes(value,origin);
   try{await writeFile(path,data,{flag:overwrite?"w":"wx"});return success(undefined);}
   catch(cause){return failureFor(cause,path,"write_bytes");}
  },
  async writeText(path:string,value:string,overwrite:boolean,context?:AssertionContext):Promise<Completion<undefined>>{
   denyLiveBoundary(context,origin);
   const bad=invalid(path);
   if(bad!==undefined)return bad;
   try{await writeFile(path,encoder.encode(value),{flag:overwrite?"w":"wx"});return success(undefined);}
   catch(cause){return failureFor(cause,path,"write_text");}
  },
  async copy(source:string,destination:string,overwrite:boolean,context?:AssertionContext):Promise<Completion<undefined>>{
   denyLiveBoundary(context,origin);return copyOrMove("copy",source,destination,overwrite);
  },
  async move(source:string,destination:string,overwrite:boolean,context?:AssertionContext):Promise<Completion<undefined>>{
   denyLiveBoundary(context,origin);return copyOrMove("move",source,destination,overwrite);
  }
 });
}
