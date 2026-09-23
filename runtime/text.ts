import {success,failure,type Completion,type AssertionContext} from "./completion.ts";
import {array,record} from "./data.ts";
import {createDomainRuntime} from "./domain.ts";
import {slice} from "./primitive.ts";
const origin=Object.freeze({source:"can:text",start:0,end:0,invocation:Object.freeze([] as string[])});
type Compiled={source:string;flags:string};
const patterns=new WeakMap<object,Compiled>();
const object=(value:unknown):value is object=>value!==null&&(typeof value==="object"||typeof value==="function");
export function isTextRegexValue(kind:string|undefined,value:unknown):boolean{
 return kind==="regex"&&object(value)&&patterns.has(value);
}
const admitted=/^[imsuv]*$/;
// A result-count limit does not bound matching time: catastrophic
// patterns still burn CPU per match. Only the count is capped here;
// regex deadlines are not promised.
const MAX_MATCHES=10000n;
export function createText(domain:ReturnType<typeof createDomainRuntime>,identities:Readonly<{emptySeparator:string;emptyPattern:string;invalidUnicode:string;invalidRegex:string;invalidLimit:string;match:string}>){
 const segmenter=new Intl.Segmenter("und",{granularity:"grapheme"});
 function invalid(reason:string):Completion<never>{return failure(domain.create(identities.invalidUnicode,record(identities.invalidUnicode,[["reason",reason]]),origin));}
 function empty(identity:string):Completion<never>{return failure(domain.create(identity,record(identity,[]),origin));}
 return Object.freeze({
  async includes(text:string,value:string,_context?:AssertionContext){return success(text.includes(value));},
  async startsWith(text:string,value:string,_context?:AssertionContext){return success(text.startsWith(value));},
  async endsWith(text:string,value:string,_context?:AssertionContext){return success(text.endsWith(value));},
  async toLowerCase(text:string,_context?:AssertionContext){return success(text.toLowerCase());},
  async toUpperCase(text:string,_context?:AssertionContext){return success(text.toUpperCase());},
  async trim(text:string,_context?:AssertionContext){return success(text.trim());},
  async slice(text:string,start:bigint,end:bigint,_context?:AssertionContext){return success(slice(text,start,end));},
  async split(text:string,separator:string,_context?:AssertionContext):Promise<Completion<readonly string[]>>{
   if(separator==="")return empty(identities.emptySeparator);
   return success(array(text.split(separator)));
  },
  async replaceAll(text:string,search:string,replacement:string,_context?:AssertionContext):Promise<Completion<string>>{
   if(search==="")return empty(identities.emptyPattern);
   return success(text.replaceAll(search,()=>replacement));
  },
  async join(items:readonly string[],separator:string,_context?:AssertionContext){return success(items.join(separator));},
  async scalars(text:string,_context?:AssertionContext):Promise<Completion<readonly bigint[]>>{
   if(!text.isWellFormed())return invalid("unpaired surrogate");
   return success(array(Array.from(text,scalar=>BigInt(scalar.codePointAt(0)!))));
  },
  async fromScalars(values:readonly bigint[],_context?:AssertionContext):Promise<Completion<string>>{
   for(const value of values)if(value<0n||value>0x10ffffn||(value>=0xd800n&&value<=0xdfffn))return invalid("invalid scalar value");
   const chunks:string[]=[];
   // Bound native argument counts without recreating Unicode encoding.
   for(let start=0;start<values.length;start+=4096)chunks.push(String.fromCodePoint(...values.slice(start,start+4096).map(Number)));
   return success(chunks.join(""));
  },
  async graphemes(text:string,_context?:AssertionContext):Promise<Completion<readonly string[]>>{
   if(!text.isWellFormed())return invalid("unpaired surrogate");
   return success(array(Array.from(segmenter.segment(text),part=>part.segment)));
  },
  async normalizeNFC(text:string,_context?:AssertionContext):Promise<Completion<string>>{
   if(!text.isWellFormed())return invalid("unpaired surrogate");
   return success(text.normalize("NFC"));
  },
  async compileRegex(pattern:string,flags:string,_context?:AssertionContext):Promise<Completion<object>>{
   if(typeof pattern!=="string"||typeof flags!=="string")throw new TypeError("invalid compiler regex input");
   // Global, sticky, and indices flags are not admitted: scanning always
   // finds every match in order, and offsets come from match positions.
   if(!admitted.test(flags))return failure(domain.create(identities.invalidRegex,record(identities.invalidRegex,[["reason","flags"]]),origin));
   try{new RegExp(pattern,flags);}catch{return failure(domain.create(identities.invalidRegex,record(identities.invalidRegex,[["reason","syntax"]]),origin));}
   const handle=Object.freeze(Object.create(null));
   patterns.set(handle,{source:pattern,flags});
   return success(handle);
  },
  async findMatches(handle:unknown,text:string,limit:bigint,_context?:AssertionContext):Promise<Completion<readonly unknown[]>>{
   const compiled=object(handle)?patterns.get(handle):undefined;
   if(compiled===undefined)throw new TypeError("invalid compiler regex handle");
   if(typeof text!=="string")throw new TypeError("invalid compiler regex input");
   if(limit<0n||limit>MAX_MATCHES)return failure(domain.create(identities.invalidLimit,record(identities.invalidLimit,[["limit",limit]]),origin));
   // Fresh global scan per call: no shared lastIndex can leak between
   // matches calls. Start/end are UTF-16 code units, consistent with
   // str.slice and str.length. Absent captures read as "".
   const scanned=new RegExp(compiled.source,compiled.flags+"g");
   const out:unknown[]=[];
   let match:RegExpExecArray|null;
   while(BigInt(out.length)<limit&&(match=scanned.exec(text))!==null){
    out.push(record(identities.match,[["text",match[0]],["start",BigInt(match.index)],["end",BigInt(match.index+match[0].length)],["groups",array(match.slice(1).map(group=>group??""))]]));
    if(match[0]==="")scanned.lastIndex++;
   }
   return success(array(out));
  },
 });
}
