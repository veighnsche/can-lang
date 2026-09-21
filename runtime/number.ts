// The closed numeric catalogue delegates arithmetic, conversion and formatting
// to the qualified runtime. These checks enforce Can's explicit boundaries.
import {success,failure,type Completion,type AssertionContext} from "./completion.ts";
import {record} from "./data.ts";
import {createDomainRuntime} from "./domain.ts";

export type NumberErrors=Readonly<{inexact:string;invalidNumber:string;invalidTextBool:string;invalidIntBool:string}>;
const origin=Object.freeze({source:"can:number",start:0,end:0,invocation:Object.freeze([] as string[])});
const integerText=/^-?(?:0|[1-9][0-9]*)$/u;
// Decimal source numeric grammar: decimal integers or fractional/exponent
// floats. Float tokens may have leading zeros; integer tokens may not.
const floatText=/^-?(?:0|[1-9][0-9]*|[0-9]+\.[0-9]+(?:[eE][+-]?[0-9]+)?|[0-9]+[eE][+-]?[0-9]+)$/u;
function fullMatch(pattern:RegExp,text:string):boolean{return pattern.exec(text)?.[0]===text;}
export function createNumbers(domain:ReturnType<typeof createDomainRuntime>,errors:NumberErrors){
 function invalid(identity:string,name:string,value:string|bigint):Completion<never>{
  return failure(domain.create(identity,record(identity,[[name,value]]),origin));
 }
 return Object.freeze({
  async fromInt(value:bigint,_context?:AssertionContext):Promise<Completion<string>>{return success(String(value));},
  async fromFloat(value:number,_context?:AssertionContext):Promise<Completion<string>>{return success(String(value));},
  async fromBool(value:boolean,_context?:AssertionContext):Promise<Completion<string>>{return success(String(value));},
  async intToFloat(value:bigint,_context?:AssertionContext):Promise<Completion<number>>{
   const converted=Number(value);
   if(!Number.isFinite(converted)||BigInt(converted)!==value)return invalid(errors.inexact,"reason","not exactly representable as float");
   return success(converted);
  },
  async floatToInt(value:number,_context?:AssertionContext):Promise<Completion<bigint>>{
   if(!Number.isFinite(value)||!Number.isInteger(value))return invalid(errors.inexact,"reason","not a finite integer");
   return success(BigInt(value));
  },
  async boolToInt(value:boolean,_context?:AssertionContext):Promise<Completion<bigint>>{return success(BigInt(Number(value)));},
  async intToBool(value:bigint,_context?:AssertionContext):Promise<Completion<boolean>>{
   if(value!==0n&&value!==1n)return invalid(errors.invalidIntBool,"value",value);
   return success(value===1n);
  },
  async floor(value:number,_context?:AssertionContext):Promise<Completion<number>>{return success(Math.floor(value));},
  async ceil(value:number,_context?:AssertionContext):Promise<Completion<number>>{return success(Math.ceil(value));},
  async trunc(value:number,_context?:AssertionContext):Promise<Completion<number>>{return success(Math.trunc(value));},
  async round(value:number,_context?:AssertionContext):Promise<Completion<number>>{return success(Math.round(value));},
  async isFinite(value:number,_context?:AssertionContext):Promise<Completion<boolean>>{return success(Number.isFinite(value));},
  async isNaN(value:number,_context?:AssertionContext):Promise<Completion<boolean>>{return success(Number.isNaN(value));},
  async toInt(value:string,_context?:AssertionContext):Promise<Completion<bigint>>{
   if(!fullMatch(integerText,value))return invalid(errors.invalidNumber,"input",value);
   return success(BigInt(value));
  },
  async toFloat(value:string,_context?:AssertionContext):Promise<Completion<number>>{
   if(!fullMatch(floatText,value))return invalid(errors.invalidNumber,"input",value);
   const converted=Number(value);
   if(!Number.isFinite(converted))return invalid(errors.invalidNumber,"input",value);
   return success(converted);
  },
  async toBool(value:string,_context?:AssertionContext):Promise<Completion<boolean>>{
   if(value!=="true"&&value!=="false")return invalid(errors.invalidTextBool,"input",value);
   return success(value==="true");
  },
 });
}
