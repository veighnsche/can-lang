// File stream producers. Acquisition failures (missing path, denied,
// directory opened as a file, bad bounds) surface at open; anything that
// changes afterwards fails the affected read or write instead. Readers
// are lazy: open stats once for acquisition errors, then pulls stream
// through the shared reader. Writers create or truncate at open.
import {stat,writeFile} from "node:fs/promises";
import {dirname} from "node:path";
import {denyLiveBoundary,type AssertionContext} from "../../assert/context.ts";
import {failure,success,type Completion} from "../../completion.ts";
import {record} from "../../data.ts";
import {createDomainRuntime} from "../../domain.ts";
import type {FailureOrigin} from "../../failure.ts";
import {openByteCell,openLineCell} from "../../transport/stream/readable.ts";
import {registerSink} from "../../transport/stream/writable.ts";
import {registerReader,type Fail} from "../../transport/stream/lifecycle.ts";
import {classifyFileError,validatePath} from "./errors.ts";
const origin:FailureOrigin=Object.freeze({source:"can:files-stream",start:0,end:0,invocation:Object.freeze([])});
type Contracts=Readonly<{notFound:string;denied:string;invalidPath:string;unexpectedKind:string;limitExceeded:string;ioError:string;closeFailed:string}>;
export function createFileStreams(domain:ReturnType<typeof createDomainRuntime>,types:Contracts){
  const fail:Fail=(identity,fields,cause)=>failure(domain.create(identity,record(identity,fields),origin,cause));
  function failureFor(cause:unknown,path:string,operation:string):Completion<never>{
    const classified=classifyFileError(cause);
    if(classified===undefined)throw cause;
    switch(classified.kind){
      case "not_found":return fail(types.notFound,[["path",path]],cause);
      case "denied":return fail(types.denied,[["path",path],["operation",operation]],cause);
      case "unexpected_kind":return fail(types.unexpectedKind,[["path",path],["operation",operation]],cause);
      case "not_directory":
      case "name_too_long":return fail(types.invalidPath,[["path",path],["reason",classified.kind]],cause);
      default:return fail(types.ioError,[["path",path],["operation",operation]],cause);
    }
  }
  async function openReadable(path:string):Promise<Completion<undefined>>{
    try{
      const info=await stat(path);
      if(!info.isFile())return fail(types.unexpectedKind,[["path",path],["operation","open"]]);
      return success(undefined);
    }catch(cause){return failureFor(cause,path,"open");}
  }
  return Object.freeze({
    async readStream(path:string,maxChunk:bigint,context?:AssertionContext):Promise<Completion<object>>{
      denyLiveBoundary(context,origin);
      const invalid=validatePath(path);
      if(invalid!==undefined)return fail(types.invalidPath,[["path",path],["reason",invalid]]);
      if(typeof maxChunk!=="bigint"||maxChunk<1n)return fail(types.limitExceeded,[["limit",typeof maxChunk==="bigint"?maxChunk:-1n]]);
      const ready=await openReadable(path);
      if(ready.kind!=="ok")return ready;
      return success(registerReader(openByteCell(Bun.file(path).stream(),maxChunk),fail,types.closeFailed));
    },
    async readLinesStream(path:string,maxLine:bigint,context?:AssertionContext):Promise<Completion<object>>{
      denyLiveBoundary(context,origin);
      const invalid=validatePath(path);
      if(invalid!==undefined)return fail(types.invalidPath,[["path",path],["reason",invalid]]);
      if(typeof maxLine!=="bigint"||maxLine<1n)return fail(types.limitExceeded,[["limit",typeof maxLine==="bigint"?maxLine:-1n]]);
      const ready=await openReadable(path);
      if(ready.kind!=="ok")return ready;
      return success(registerReader(openLineCell(Bun.file(path).stream(),maxLine),fail,types.closeFailed));
    },
    async writeStream(path:string,context?:AssertionContext):Promise<Completion<object>>{
      denyLiveBoundary(context,origin);
      const invalid=validatePath(path);
      if(invalid!==undefined)return fail(types.invalidPath,[["path",path],["reason",invalid]]);
      try{
        const info=await stat(path).catch(cause=>{
          const classified=classifyFileError(cause);
          if(classified?.kind==="not_found")return undefined;
          throw cause;
        });
        if(info!==undefined&&!info.isFile())return fail(types.unexpectedKind,[["path",path],["operation","open"]]);
        if(info===undefined)await stat(dirname(path));
        await writeFile(path,new Uint8Array(0));
      }catch(cause){return failureFor(cause,path,"open");}
      return success(registerSink(Bun.file(path).writer(),fail,types.closeFailed));
    },
  });
}
