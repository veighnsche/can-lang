import {denyLiveBoundary,type AssertionContext} from "../../assert/context.ts";
import {success,failure,type Completion} from "../../completion.ts";
import {array,record} from "../../data.ts";
import {createDomainRuntime} from "../../domain.ts";
import type {FailureOrigin} from "../../failure.ts";
import {classifyFileError,validatePath} from "./errors.ts";
const origin:FailureOrigin=Object.freeze({source:"can:files-glob",start:0,end:0,invocation:Object.freeze([])});
type Contracts=Readonly<{notFound:string;denied:string;invalidPath:string;limitExceeded:string;ioError:string}>;
export function createFileGlobs(domain:ReturnType<typeof createDomainRuntime>,types:Contracts){
 const fail=(identity:string,fields:readonly(readonly[string,unknown])[],cause?:unknown)=>failure(domain.create(identity,record(identity,fields),origin,cause));
 return Object.freeze({
  async glob(base:string,pattern:string,followSymlinks:boolean,maxEntries:bigint,context?:AssertionContext):Promise<Completion<readonly string[]>>{
   denyLiveBoundary(context,origin);
   const badBase=validatePath(base);
   if(badBase!==undefined)return fail(types.invalidPath,[["path",base],["reason",badBase]]);
   const badPattern=validatePath(pattern);
   if(badPattern!==undefined)return fail(types.invalidPath,[["path",pattern],["reason",badPattern]]);
   if(maxEntries<0n)return fail(types.limitExceeded,[["limit",maxEntries]]);
   try{
    const collected:string[]=[];
    const scanner=new Bun.Glob(pattern).scan({cwd:base,absolute:true,followSymlinks,onlyFiles:false});
    for await(const match of scanner){
     collected.push(match);
     // Enforce the cap during enumeration, never by silent truncation.
     if(BigInt(collected.length)>maxEntries)return fail(types.limitExceeded,[["limit",maxEntries]]);
    }
    collected.sort();
    return success(array(collected));
   }catch(cause){
    const classified=classifyFileError(cause);
    if(classified===undefined)throw cause;
    switch(classified.kind){
     case "not_found":return fail(types.notFound,[["path",base]],cause);
     case "denied":return fail(types.denied,[["path",base],["operation","glob"]],cause);
     case "not_directory":
     case "name_too_long":return fail(types.invalidPath,[["path",base],["reason",classified.kind==="not_directory"?"not_directory":"name_too_long"]],cause);
     default:return fail(types.ioError,[["path",base],["operation","glob"]],cause);
    }
   }
  }
 });
}
