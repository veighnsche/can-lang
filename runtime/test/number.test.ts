import {test,expect} from "bun:test";
import {createHash} from "node:crypto";
import {catalogue} from "../catalogue.ts";
import {createDomainRuntime,domainFailureDiagnostics,type FailureShape} from "../domain.ts";
import {createNumbers} from "../number.ts";
import {value,type Completion} from "../completion.ts";
const identity=(kind:string,declaration:string)=>createHash("sha256").update("can-concrete-type-v1\0"+JSON.stringify([kind,declaration])).digest("hex");
const scalar=(name:string):FailureShape=>({identity:identity("primitive",name),kind:"primitive",declaration:name,arguments:[],fields:[],leaves:[],inputs:[],errors:[]});
const text=scalar("str"),integer=scalar("int");
const declarations=catalogue.errors.filter(error=>[1000,1001,1002,1003].includes(error.id));
const errors=declarations.map(error=>({identity:identity("error",error.identity),kind:"error",declaration:error.identity,arguments:[],fields:error.fields.map(field=>({name:field.name,type:field.type==="int"?integer.identity:text.identity})),leaves:[],inputs:[],errors:[]}));
const domain=createDomainRuntime({declarations:declarations.map(error=>({identity:error.identity,name:error.name,id:error.id,parameters:0})),shapes:[text,integer,...errors]});
const api=createNumbers(domain,{inexact:errors[0].identity,invalidNumber:errors[1].identity,invalidTextBool:errors[2].identity,invalidIntBool:errors[3].identity});
function invalid(result:Completion,id:number,payload:Record<string,unknown>){
 expect(result.kind).toBe("domain");if(result.kind!=="domain")throw Error("expected domain failure");
 const details=domainFailureDiagnostics(result.value);expect(details.declaration.id).toBe(id);expect(details.payload).toMatchObject(payload);
}

test("native formatting preserves bigint digits and IEEE spellings without locale formatting",async()=>{
 for(const input of [0n,-1n,9007199254740993n,10n**400n])expect(value(await api.fromInt(input))).toBe(String(input));
 for(const input of [NaN,Infinity,-Infinity,-0,0,Number.MIN_VALUE,Number.MAX_VALUE,0.1,1e21])expect(value(await api.fromFloat(input))).toBe(String(input));
 expect(value(await api.fromFloat(-0))).toBe("0");
 for(const input of [true,false])expect(value(await api.fromBool(input))).toBe(String(input));
});

test("exact conversion uses native represented values rather than the safe-integer cutoff",async()=>{
 for(const input of [0n,-1n,9007199254740992n,2n**80n,-(2n**100n)])expect(Object.is(value(await api.intToFloat(input)),Number(input))).toBe(true);
 for(const input of [9007199254740993n,-9007199254740993n,10n**400n])invalid(await api.intToFloat(input),1000,{reason:"not exactly representable as float"});
 for(const input of [0,-0,1,-1,2**80,Number.MAX_VALUE])expect(value(await api.floatToInt(input))).toBe(BigInt(input));
 for(const input of [NaN,Infinity,-Infinity,0.5,-0.5,Number.MIN_VALUE])invalid(await api.floatToInt(input),1000,{reason:"not a finite integer"});
});

test("Math and Number predicates agree with qualified native operations on boundaries",async()=>{
 const inputs=[NaN,Infinity,-Infinity,0,-0,0.5,-0.5,1.5,-1.5,2.5,-2.5,Number.MIN_VALUE,-Number.MIN_VALUE,Number.MAX_VALUE,2**53];
 for(const operation of ["floor","ceil","trunc","round"] as const)for(const input of inputs)expect(Object.is(value(await api[operation](input)),Math[operation](input))).toBe(true);
 for(const input of inputs){expect(value(await api.isFinite(input))).toBe(Number.isFinite(input));expect(value(await api.isNaN(input))).toBe(Number.isNaN(input))}
 expect(Object.is(value(await api.round(-0.5)),-0)).toBe(true);
});

test("integer parsing requires the entire exact decimal grammar",async()=>{
 for(const input of ["0","-0","1","-1","9007199254740993","9".repeat(400)])expect(value(await api.toInt(input))).toBe(BigInt(input));
 for(const input of [""," "," 1","1 ","1\n","1\r\n","+1","01","-01","1.0","1e2","0x10","0b10","1_0","NaN","Infinity","−1","１","1\u0000"]){invalid(await api.toInt(input),1001,{input})}
});

test("float parsing admits only decimal source numbers and finite native results",async()=>{
 for(const input of ["0","-0","1","-1","1.0","01.0","00e1","1e3","1E+3","-1e-999","1.25e-2","9007199254740993","5e-324"]){expect(Object.is(value(await api.toFloat(input)),Number(input))).toBe(true)}
 for(const input of [""," "," 1","1 ","1\n","1\r\n","+1","01","-01",".5","1.","1.e2","1e","1e+","0x10","1_0","NaN","Infinity","-Infinity","1e309","１","1\u0000"]){invalid(await api.toFloat(input),1001,{input})}
});

test("boolean conversions accept exact spellings and only the integer range zero through one",async()=>{
 for(const [input,expected] of [[false,0n],[true,1n]] as const){expect(value(await api.boolToInt(input))).toBe(expected);expect(value(await api.intToBool(expected))).toBe(input);expect(value(await api.toBool(String(input)))).toBe(input)}
 for(const input of [-1n,2n,2n**100n])invalid(await api.intToBool(input),1003,{value:input});
 for(const input of ["True","FALSE"," true","false\n","1","0",""])invalid(await api.toBool(input),1002,{input});
});
