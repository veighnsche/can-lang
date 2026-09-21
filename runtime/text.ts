import {success,failure,type Completion,type AssertionContext} from "./completion.ts";
import {array,record} from "./data.ts";
import {createDomainRuntime} from "./domain.ts";
import {slice} from "./primitive.ts";
const origin=Object.freeze({source:"can:text",start:0,end:0,invocation:Object.freeze([] as string[])});
export function createText(domain:ReturnType<typeof createDomainRuntime>,identities:Readonly<{emptySeparator:string;emptyPattern:string;invalidUnicode:string}>){
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
 });
}
