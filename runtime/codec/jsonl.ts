import { byteLength, copyBytes } from "../bytes.ts";
import { array } from "../data.ts";
import { Budget, CodecIssue, childPath, reject, standaloneBytes } from "./budget.ts";
import { parseJSONText } from "./document.ts";
import { exactInt, projectValue, type Schema } from "./project.ts";

const origin = Object.freeze({source:"can:codec",start:0,end:0,invocation:Object.freeze([])});

// Strict line framing over bytes. Newline (0x0A) never appears inside a
// UTF-8 multibyte sequence, so byte splits are character-safe; each line
// decodes separately with fatal UTF-8. Only the incomplete suffix is
// retained, never prior records. Blank lines are skipped, a trailing CR is
// stripped per line, and the final line needs no newline. Records spanning
// lines are rejected: each yielded line must parse as complete JSON.
export type JsonlFramer = {
  push(chunk:Uint8Array):Uint8Array[];
  finish():Uint8Array|undefined;
};
export function createJsonlFramer(bytes=standaloneBytes):JsonlFramer {
  let suffix=new Uint8Array(0), yielded=0;
  const take=(line:Uint8Array):Uint8Array=>
    line.length>0&&line[line.length-1]===0x0d?line.slice(0,line.length-1):line;
  return {
    // Cells are reader-sized (tens of KB) with complete lines flushed per
    // push, so retained copies stay linear overall. The retained suffix is
    // the incomplete record: once it exceeds the byte cap the record can
    // never fit and fails here instead of growing without bound.
    push(chunk:Uint8Array):Uint8Array[] {
      const data=new Uint8Array(suffix.length+chunk.length);
      data.set(suffix,0);data.set(chunk,suffix.length);
      const lines:Uint8Array[]=[];
      let start=0;
      for(let i=0;i<data.length;i++){
        if(data[i]!==0x0a)continue;
        const line=take(data.slice(start,i));
        if(line.length>0){lines.push(line);yielded++;}
        start=i+1;
      }
      suffix=data.slice(start);
      if(suffix.length>bytes)reject(childPath("",yielded),"byte_limit");
      return lines;
    },
    finish():Uint8Array|undefined {
      if(suffix.length===0)return undefined;
      const line=take(suffix);
      suffix=new Uint8Array(0);
      return line.length>0?line:undefined;
    },
  };
}

// projectJsonlRecord decodes one framed line through the exact per-record
// JSON path and projects it against the row schema. Paths prefix the
// record index, so a member fault reads /<index>/<member>.
export function projectJsonlRecord(schema:Schema,line:Uint8Array,index:number,budget:Budget):unknown {
  const path=childPath("",index);
  let text:string;
  try{text=new TextDecoder("utf-8",{fatal:true,ignoreBOM:true}).decode(line);}
  catch(cause){if(cause instanceof TypeError)reject(path,"utf8");throw cause;}
  const {parsed,rootHolder,tokens}=prefixRecord(text,path);
  return projectIndexed(schema,parsed,rootHolder,tokens,path,budget);
}

function prefixRecord(text:string,path:string) {
  try{return parseJSONText(text);}
  catch(cause){
    if(cause instanceof CodecIssue)throw new CodecIssue(path+cause.path,cause.reason);
    throw cause;
  }
}

function projectIndexed(schema:Schema,parsed:unknown,holder:object,tokens:WeakMap<object,Map<string,string>>,path:string,budget:Budget):unknown {
  try{return projectValue(schema,parsed,holder,budget,exactInt(tokens));}
  catch(cause){
    if(cause instanceof CodecIssue)throw new CodecIssue(path+cause.path,cause.reason);
    throw cause;
  }
}

// decodeJsonlRecords projects a whole bounded payload. Record faults carry
// the record index; the shared budget bounds nodes and integers across
// every record.
export function decodeJsonlRecords(schema:Schema,input:unknown,bytes=standaloneBytes):unknown {
  if(byteLength(input)>BigInt(bytes))reject("","byte_limit");
  const raw=copyBytes(input,origin);
  const budget=new Budget(bytes);
  const framer=createJsonlFramer();
  const records:unknown[]=[];
  let index=0;
  for(const line of framer.push(raw))records.push(projectJsonlRecord(schema,line,index++,budget));
  const tail=framer.finish();
  if(tail!==undefined)records.push(projectJsonlRecord(schema,tail,index++,budget));
  return array(records);
}
