import { byteLength, copyBytes } from "../bytes.ts";
import { reject, standaloneBytes } from "./budget.ts";
import { scanJSON } from "./duplicates.ts";

const origin = Object.freeze({source:"can:codec",start:0,end:0,invocation:Object.freeze([])});

// Private maintained protocol adapters and typed decoding share this bounded
// native parser. Its output is never an authored Can untyped-JSON value.
export function parseDocument(input:unknown,bytes=standaloneBytes) {
  if(byteLength(input)>BigInt(bytes))reject("","byte_limit");
  let text:string;
  try{text=new TextDecoder("utf-8",{fatal:true,ignoreBOM:true}).decode(copyBytes(input,origin));}
  catch(cause){if(cause instanceof TypeError)reject("","utf8");throw cause;}
  // BOM preservation makes a leading BOM fail native JSON syntax, as required.
  const duplicate=scanJSON(text);
  const tokens=new WeakMap<object,Map<string,string>>();
  let rootHolder:object|undefined;
  let parsed:unknown;
  try{
    parsed=JSON.parse(text,function(this:object,key:string,value:unknown,context?:{source?:string}){
      if(context?.source!==undefined){let holder=tokens.get(this);if(!holder){holder=new Map();tokens.set(this,holder);}holder.set(key,context.source);}
      if(key==="")rootHolder=this;
      return value;
    });
  }catch(cause){if(cause instanceof SyntaxError)reject("","invalid_json");throw cause;}
  if(duplicate!==undefined)reject(duplicate,"duplicate_member");
  return {parsed,rootHolder:rootHolder!,tokens};
}
