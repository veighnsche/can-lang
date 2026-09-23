import {lstat,access,constants} from "node:fs/promises";
import {denyLiveBoundary,type AssertionContext} from "../../assert/context.ts";
import {success,failure,type Completion} from "../../completion.ts";
import {record,dataProperty,dataArray} from "../../data.ts";
import {ownBytes,copyBytes,type Bytes} from "../../bytes.ts";
import {createDomainRuntime} from "../../domain.ts";
import {registerResource,closeResource} from "../../owner.ts";
import type {FailureOrigin} from "../../failure.ts";
import {classifySpawnError,validateExecutable,validateArg,parseEnvEntry,maxDurationMs} from "./errors.ts";
import {drainBounded,deliverStdin,type ChildProcess,type DrainOutcome} from "./io.ts";
import {terminateGroup,exitedOf} from "./wait.ts";
const origin:FailureOrigin=Object.freeze({source:"can:process",start:0,end:0,invocation:Object.freeze([])});
type Contracts=Readonly<{notFound:string;denied:string;spawnFailed:string;timeout:string;outputLimit:string;nonzero:string;invalidConfig:string;ioError:string;result:string}>;
type RunConfig=Readonly<{argv:readonly string[];cwd:string;env:Readonly<Record<string,string>>;stdin:Uint8Array;stdoutLimit:bigint;stderrLimit:bigint;deadlineMs:bigint;graceMs:bigint}>;
export function createProcesses(domain:ReturnType<typeof createDomainRuntime>,types:Contracts){
 const fail=(identity:string,fields:readonly(readonly[string,unknown])[],cause?:unknown)=>failure(domain.create(identity,record(identity,fields),origin,cause));
 const invalid=(field:string,reason:string)=>fail(types.invalidConfig,[["field",field],["reason",reason]]);
 function configure(executable:string,args:readonly unknown[],options:unknown):RunConfig|Completion<never>{
  const badExe=validateExecutable(executable);
  if(badExe!==undefined)return invalid("executable",badExe);
  const argv:string[]=[];
  for(const arg of dataArray(args)){
   if(validateArg(arg as string)!==undefined)return invalid("args","nul_byte");
   argv.push(arg as string);
  }
  const cwd=dataProperty(options,"cwd")as string;
  if(cwd!==""&&(cwd.includes("\0")))return invalid("cwd","nul_byte");
  const inherit=dataProperty(options,"inherit_env")as boolean;
  const extra:Record<string,string>={};
  for(const line of dataArray(dataProperty(options,"env"))){
   const parsed=parseEnvEntry(line as string);
   if("reason" in parsed)return invalid("env",parsed.reason);
   extra[parsed.name]=parsed.value;
  }
  const stdin=copyBytes(dataProperty(options,"stdin")as Bytes,origin);
  const stdoutLimit=dataProperty(options,"stdout_limit")as bigint;
  const stderrLimit=dataProperty(options,"stderr_limit")as bigint;
  if(stdoutLimit<0n)return invalid("stdout_limit","negative");
  if(stderrLimit<0n)return invalid("stderr_limit","negative");
  const deadlineMs=dataProperty(options,"deadline_ms")as bigint;
  const graceMs=dataProperty(options,"grace_ms")as bigint;
  if(deadlineMs<0n||graceMs<0n)return invalid(deadlineMs<0n?"deadline_ms":"grace_ms","negative");
  if(deadlineMs>BigInt(maxDurationMs)||graceMs>BigInt(maxDurationMs))return invalid(deadlineMs>BigInt(maxDurationMs)?"deadline_ms":"grace_ms","too_large");
  const env:Record<string,string>={};
  if(inherit)for(const [key,value] of Object.entries(Bun.env))if(value!==undefined)env[key]=value;
  for(const [key,value] of Object.entries(extra))env[key]=value;
  return {argv,cwd,env,stdin,stdoutLimit,stderrLimit,deadlineMs,graceMs};
 }
 async function spawnFailure(cause:unknown,executable:string,cwd:string):Promise<Completion<never>>{
  const classified=classifySpawnError(cause);
  if(classified===undefined)throw cause;
  if(classified.kind==="denied"){
   // Attribute best-effort: an inaccessible explicit cwd denies the spawn.
   if(cwd!=="")try{await access(cwd,constants.X_OK);}catch{return fail(types.denied,[["path",cwd],["operation","spawn"]],cause);}
   return fail(types.denied,[["path",executable],["operation","spawn"]],cause);
  }
  if(classified.kind==="not_found"){
   if(cwd!=="")try{await lstat(cwd);}catch(nested){
    if(classifySpawnError(nested)?.kind==="not_found")return fail(types.notFound,[["path",cwd]],cause);
   }
   return fail(types.notFound,[["path",executable]],cause);
  }
  return fail(types.spawnFailed,[["executable",executable]],cause);
 }
 type Settle=Readonly<{kind:"exited"}|{kind:"timeout"}|{kind:"stream_failed";stream:string}|{kind:"overflow";stream:string;limit:bigint}>;
 async function execute(executable:string,config:RunConfig):Promise<Completion<unknown>>{
  let proc:ChildProcess;
  try{
   proc=Bun.spawn([executable,...config.argv],{stdin:"pipe",stdout:"pipe",stderr:"pipe",cwd:config.cwd===""?undefined:config.cwd,env:config.env,detached:true});
  }catch(cause){return spawnFailure(cause,executable,config.cwd);}
  let settled=false;
  const isExited=()=>settled||exitedOf(proc.exitCode,proc.signalCode);
  const token=registerResource("process",{pid:proc.pid},async():Promise<Completion<void>>=>{
   if(settled)return success(undefined);
   try{await terminateGroup(proc.pid,Number(config.graceMs),isExited);}
   catch(cause){return failure(domain.create(types.ioError,record(types.ioError,[["operation","kill"]]),origin,cause));}
   return success(undefined);
   // Idempotent: scope drain may start the close while the run is settling;
   // the run's own close then joins the in-flight close instead of failing.
  },{scopeManaged:true,idempotent:true,shutdownMilliseconds:Number(config.graceMs)+5000});
  try{
   try{deliverStdin(proc,config.stdin);}
   catch(cause){
    await terminateGroup(proc.pid,Number(config.graceMs),isExited);
    try{await proc.exited;}catch{}
    return fail(types.ioError,[["operation","stdin"]],cause);
   }
   const stdoutDrain=drainBounded(proc.stdout,config.stdoutLimit,fail,types.ioError);
   const stderrDrain=drainBounded(proc.stderr,config.stderrLimit,fail,types.ioError);
   let decide!:(settle:Settle)=>void;
   const decided=new Promise<Settle>(resolve=>{decide=resolve;});
   let done=false;
   const finish=(settle:Settle)=>{if(!done){done=true;decide(settle);}};
   proc.exited.then(()=>finish({kind:"exited"}),()=>finish({kind:"exited"}));
   let timer:ReturnType<typeof setTimeout>|undefined;
   if(config.deadlineMs>0n)timer=setTimeout(()=>finish({kind:"timeout"}),Number(config.deadlineMs));
   stdoutDrain.then(outcome=>{if(outcome.overflow)finish({kind:"overflow",stream:"stdout",limit:config.stdoutLimit});},()=>finish({kind:"stream_failed",stream:"stdout"}));
   stderrDrain.then(outcome=>{if(outcome.overflow)finish({kind:"overflow",stream:"stderr",limit:config.stderrLimit});},()=>finish({kind:"stream_failed",stream:"stderr"}));
   try{
    const outcome=await decided;
    if(outcome.kind!=="exited"){
     // Termination is best-effort; the declared outcome below stands even
     // when cleanup itself fails.
     try{await terminateGroup(proc.pid,Number(config.graceMs),isExited);}catch{}
     try{await proc.exited;}catch{}
     // Drains observe EOF after termination; settle them before reporting.
     await Promise.allSettled([stdoutDrain,stderrDrain]);
    }
    if(outcome.kind==="stream_failed")return fail(types.ioError,[["operation",outcome.stream]]);
    const drains:readonly [DrainOutcome,DrainOutcome]=await Promise.all([stdoutDrain,stderrDrain]);
    if(outcome.kind==="timeout")return fail(types.timeout,[["deadline_ms",config.deadlineMs]]);
    if(outcome.kind==="overflow"||drains[0].overflow)return fail(types.outputLimit,[["stream","stdout"],["limit",config.stdoutLimit]]);
    if(drains[1].overflow)return fail(types.outputLimit,[["stream","stderr"],["limit",config.stderrLimit]]);
    if(proc.exitCode===null&&proc.signalCode===null)return fail(types.ioError,[["operation","wait"]]);
    const out=drains[0].overflow===false?drains[0].data:new Uint8Array();
    const err=drains[1].overflow===false?drains[1].data:new Uint8Array();
    return success(record(types.result,[["stdout",ownBytes(out)],["stderr",ownBytes(err)],["code",BigInt(proc.exitCode??-1)],["signal",proc.signalCode??""]]));
   }finally{if(timer!==undefined)clearTimeout(timer);}
  }finally{
   settled=true;
   // Scope drain may own the close already; close failures are owner
   // diagnostics, never a replacement for the run outcome.
   try{await closeResource(token,"process");}catch{}
  }
 }
 return Object.freeze({
  async run(executable:string,args:readonly unknown[],options:unknown,context?:AssertionContext):Promise<Completion<unknown>>{
   denyLiveBoundary(context,origin);
   const config=configure(executable,args,options);
   if(typeof (config as Completion<never>).kind==="string")return config as Completion<never>;
   return execute(executable,config as RunConfig);
  },
  async requireSuccess(value:unknown,_context?:AssertionContext):Promise<Completion<unknown>>{
   const code=dataProperty(value,"code")as bigint,signal=dataProperty(value,"signal")as string;
   if(code===0n&&signal==="")return success(value);
   return fail(types.nonzero,[["code",code],["signal",signal]]);
  },
  async which(name:string,context?:AssertionContext):Promise<Completion<string>>{
   denyLiveBoundary(context,origin);
   const bad=validateExecutable(name);
   if(bad!==undefined)return invalid("name",bad);
   const resolved=Bun.which(name);
   if(resolved===null)return fail(types.notFound,[["path",name]]);
   return success(resolved);
  }
 });
}
