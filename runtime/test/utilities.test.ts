import {test,expect} from "bun:test";
import {createHash} from "node:crypto";
import {catalogue} from "../catalogue.ts";
import {createDomainRuntime,domainFailureDiagnostics,type FailureShape} from "../domain.ts";
import {createClock} from "../platform/clock.ts";
import {createRandom} from "../platform/random.ts";
import {sha256} from "../platform/crypto/primitives.ts";
import {createLog} from "../platform/log.ts";
import {copyBytes,ownBytes,byteLength} from "../bytes.ts";
import {success,value,invoke,type Completion,type AssertionContext} from "../completion.ts";
import {runEntry} from "../entry.ts";
import {runAssertion} from "../assert/runner.ts";
const hash=(kind:string,name:string)=>createHash("sha256").update("can-concrete-type-v1\0"+JSON.stringify([kind,name])).digest("hex");
const shape=(kind:string,declaration:string,fields:{name:string;type:string}[]=[]):FailureShape=>({identity:hash(kind,declaration),kind,declaration,fields,arguments:[],leaves:[],inputs:[],errors:[]});
const str=shape("primitive","str"),int=shape("primitive","int");
const declarations=catalogue.errors.filter(e=>[1260,1261,1263].includes(e.id));
const errors=declarations.map(e=>shape("error",e.identity,e.fields.map(f=>({name:f.name,type:f.type==="int"?int.identity:str.identity}))));
const domain=createDomainRuntime({declarations:declarations.map(e=>({...e,parameters:0})),shapes:[str,int,...errors]});
const clock=createClock(domain,errors[0]!.identity),random=createRandom(domain,errors[1]!.identity);
const origin={source:"test",start:0,end:0,invocation:[]};
function check(result:Completion,id:number,payload:object){expect(result.kind).toBe("domain");if(result.kind!=="domain")throw Error("expected domain");const d=domainFailureDiagnostics(result.value);expect(d.declaration.id).toBe(id);expect(d.payload).toMatchObject(payload);}
test("native clock units, monotonic observations and awaited sleep",async()=>{
 const before=BigInt(Date.now()),wall=value(await clock.wallMillis());expect(wall>=before&&wall<=BigInt(Date.now())).toBe(true);
 const start=value(await clock.monotonicMillis());expect(Number.isFinite(start)).toBe(true);await clock.sleepMillis(10n);const end=value(await clock.monotonicMillis());expect(end>=start).toBe(true);expect(end-start).toBeGreaterThanOrEqual(5);
 expect(await clock.sleepMillis(0n)).toEqual(success(undefined));
 for(const n of [-1n,2147483648n,10n**100n])check(await clock.sleepMillis(n),1260,{milliseconds:n});
});
test("secure random bounds, immutable bytes and UUID version",async()=>{
 for(const n of [0n,1n,65536n]){const bytes=value(await random.secureBytes(n));expect(byteLength(bytes)).toBe(n);expect(Object.isFrozen(bytes)).toBe(true);const first=copyBytes(bytes,origin),second=copyBytes(bytes,origin);first.fill(0);expect(copyBytes(bytes,origin)).toEqual(second);}
 for(const n of [-1n,65537n,10n**100n])check(await random.secureBytes(n),1261,{length:n});
 for(let i=0;i<10;i++)expect(value(await random.uuidV4())).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
});
test("SHA-256 known vectors and independent binary oracle",async()=>{
 for(const [input,expected] of [["","e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"],["abc","ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"]]){
  const result=value(await sha256(ownBytes(new TextEncoder().encode(input))));expect(Buffer.from(copyBytes(result,origin)).toString("hex")).toBe(expected);
 }
 const binary=new Uint8Array([0,255,128,1]);expect(Buffer.from(copyBytes(value(await sha256(ownBytes(binary))),origin)).toString("hex")).toBe(createHash("sha256").update(binary).digest("hex"));
 let traps=0;const forged=new Proxy({},{get(){traps++;throw Error("private");}});expect((await invoke(()=>sha256(forged),origin)).kind).toBe("standard");expect(traps).toBe(0);
});
test("logs contain only native JSON string fields and await write",async()=>{
 const lines:string[]=[];let release!:()=>void;const pending=new Promise<void>(resolve=>{release=resolve;});
 const log=createLog(domain,errors[2]!.identity,{encode:JSON.stringify,write:async line=>{await pending;lines.push(line);return line.length;}});
 let done=false;const result=log.writeInfo('secret\n"\\\ud800').then(v=>{done=true;return v;});await Promise.resolve();expect(done).toBe(false);release();expect(await result).toEqual(success(undefined));
 expect(lines).toEqual([JSON.stringify({level:"info",message:'secret\n"\\\ud800'})+"\n"]);
 expect(await log.writeError("error text")).toEqual(success(undefined));expect(JSON.parse(lines[1]!)).toEqual({level:"error",message:"error text"});
});
test("log serialization and expected I/O failures disclose only level",async()=>{
 for(const serialization of [true,false]){
  const log=createLog(domain,errors[2]!.identity,{encode:()=>{if(serialization)throw new RangeError("private serialization");return "{}";},write:async()=>{throw Object.assign(new Error("private native path"),{code:"EPIPE"});}});
  check(await log.writeError("private message"),1263,{level:"error"});
  const reports:string[]=[];expect(await runEntry(()=>{},()=>log.writeError("private message"),[],line=>{reports.push(line);})).toBe(1);expect(reports).toHaveLength(1);expect(reports[0]).toContain('"id":1263');expect(reports[0]).not.toContain("private");
 }
 const log=createLog(domain,errors[2]!.identity,{encode:JSON.stringify,write:async()=>{throw new Error("defect");}});expect((await invoke(()=>log.writeInfo("private"),origin)).kind).toBe("standard");
 let traps=0;const forged=new Proxy({},{get(){traps++;throw Error("private");}});expect((await invoke(()=>log.writeInfo(forged as string),origin)).kind).toBe("standard");expect(traps).toBe(0);
});
test("host utilities require fixtures while hashing runs in ordinary assertions",async()=>{
 const log=createLog(domain,errors[2]!.identity,{encode:()=>{throw Error("ambient");},write:async()=>{throw Error("ambient");}});
 const calls:((context:AssertionContext)=>Promise<Completion>)[]=[c=>clock.wallMillis(c),c=>clock.monotonicMillis(c),c=>clock.sleepMillis(0n,c),c=>random.secureBytes(0n,c),c=>random.uuidV4(c),c=>log.writeInfo("secret",c),c=>log.writeError("secret",c)];
 for(const actual of calls){const report=await runAssertion({root:{package:"test",declaration:"host",name:"unsupplied"},actual,expected:async()=>success(undefined)});expect(report.passed).toBe(false);expect(JSON.stringify(report)).not.toContain("secret");}
 const input=ownBytes(new Uint8Array());const expected="e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855";const report=await runAssertion({root:{package:"test",declaration:"hash",name:"real"},actual:async c=>success(Buffer.from(copyBytes(value(await sha256(input,c)),origin)).toString("hex")),expected:async()=>success(expected)});expect(report.passed).toBe(true);expect(report.evidence).toEqual(["real-can"]);
});
