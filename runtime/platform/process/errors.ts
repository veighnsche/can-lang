import {types as nativeTypes} from "node:util";
// Durations follow the clock-sleep clamp: setTimeout cannot exceed 2^31-1.
export const maxDurationMs=2147483647;
export function processErrorCode(cause:unknown):string|undefined{
 if(cause===null||typeof cause!=="object"||nativeTypes.isProxy(cause)||!nativeTypes.isNativeError(cause))return undefined;
 const code=Object.getOwnPropertyDescriptor(cause,"code");
 if(code===undefined||!("value" in code)||typeof code.value!=="string")return undefined;
 return code.value;
}
export type SpawnProblem="not_found"|"denied"|"failed";
const denied=new Set(["EACCES","EPERM","EROFS"]);
const spawnFailed=new Set(["ENAMETOOLONG","ELOOP","ENOMEM","ETXTBSY","ENOEXEC","E2BIG","EINVAL","EBUSY","EAGAIN","EMFILE","ENFILE"]);
export function classifySpawnError(cause:unknown):{kind:SpawnProblem;code:string}|undefined{
 const code=processErrorCode(cause);
 if(code===undefined)return undefined;
 if(code==="ENOENT")return {kind:"not_found",code};
 if(denied.has(code))return {kind:"denied",code};
 if(spawnFailed.has(code))return {kind:"failed",code};
 return undefined;
}
export function validateExecutable(executable:string):string|undefined{
 if(executable==="")return "empty";
 if(executable.includes("\0"))return "nul_byte";
 return undefined;
}
// Empty arguments are legal for exec; only NUL bytes are rejected.
export function validateArg(arg:string):string|undefined{
 if(arg.includes("\0"))return "nul_byte";
 return undefined;
}
const envName=/^[A-Za-z_][A-Za-z0-9_]*$/;
export function parseEnvEntry(entry:string):{name:string;value:string}|{reason:string}{
 const cut=entry.indexOf("=");
 if(cut<=0)return {reason:"entry"};
 const name=entry.slice(0,cut),value=entry.slice(cut+1);
 if(!envName.test(name))return {reason:"name"};
 if(value.includes("\0"))return {reason:"nul_byte"};
 return {name,value};
}
