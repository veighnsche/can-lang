import {test,expect} from "bun:test";
import {createHash} from "node:crypto";
import {catalogue} from "../catalogue.ts";
import {createDomainRuntime,domainFailureDiagnostics,type FailureShape} from "../domain.ts";
import {createText} from "../text.ts";
import {value,type Completion} from "../completion.ts";
const shape=(kind:string,declaration:string):FailureShape=>({identity:createHash("sha256").update("can-concrete-type-v1\0"+JSON.stringify([kind,declaration])).digest("hex"),kind,declaration,arguments:[],fields:[],leaves:[],inputs:[],errors:[]});
const text=shape("primitive","str"),declarations=catalogue.errors.filter(e=>[1004,1005,1006].includes(e.id));
const errors=declarations.map(e=>({...shape("error",e.identity),fields:e.fields.map(f=>({name:f.name,type:text.identity}))}));
const domain=createDomainRuntime({declarations:declarations.map(e=>({identity:e.identity,name:e.name,id:e.id,parameters:0})),shapes:[text,...errors]});
const api=createText(domain,{emptySeparator:errors[0].identity,emptyPattern:errors[1].identity,invalidUnicode:errors[2].identity});
function invalid(result:Completion,id:number){expect(result.kind).toBe("domain");if(result.kind!=="domain")throw Error("expected domain failure");expect(domainFailureDiagnostics(result.value).declaration.id).toBe(id)}

test("search, case and trim delegate native code-unit and Unicode behavior",async()=>{
 for(const input of ["","A😀B","İßΣς","\u00a0\ufeff hello \u2029","\ud800"]){
  for(const needle of ["","A","😀","\ud83d","z"]){
   expect(value(await api.includes(input,needle))).toBe(input.includes(needle));
   expect(value(await api.startsWith(input,needle))).toBe(input.startsWith(needle));
   expect(value(await api.endsWith(input,needle))).toBe(input.endsWith(needle));
  }
  expect(value(await api.toLowerCase(input))).toBe(input.toLowerCase());
  expect(value(await api.toUpperCase(input))).toBe(input.toUpperCase());
  expect(value(await api.trim(input))).toBe(input.trim());
 }
 expect(value(await api.slice("A😀B",1n,2n))).toBe("\ud83d");
 expect(value(await api.slice("A😀B",-(10n**50n),10n**50n))).toBe("A😀B");
});

test("literal replacement preserves dollar sequences and split retains every empty piece",async()=>{
 for(const replacement of ["$&","$`","$'","$$","$1","😀"]){expect(value(await api.replaceAll("aba","a",replacement))).toBe(replacement+"b"+replacement)}
 expect(value(await api.replaceAll("a.a",".","$&"))).toBe("a$&a");
 const pieces=value(await api.split(",a,,",","));expect(pieces).toEqual(["","a","",""]);expect(Object.isFrozen(pieces)).toBe(true);
 expect(value(await api.split("",","))).toEqual([""]);
 expect(value(await api.join(pieces,","))).toBe(",a,,");expect(value(await api.join([],","))).toBe("");
 invalid(await api.split("text",""),1004);invalid(await api.replaceAll("text","","x"),1005);
});

test("scalar and grapheme APIs deliberately differ from UTF-16 indexing",async()=>{
 const input="A😀e\u0301👩‍👩‍👧‍👦";
 const scalars=value(await api.scalars(input));expect(scalars).toEqual(Array.from(input,c=>BigInt(c.codePointAt(0)!)));
 expect(scalars.length).toBeLessThan(input.length);expect(Object.isFrozen(scalars)).toBe(true);
 expect(value(await api.fromScalars(scalars))).toBe(input);
 const segments=value(await api.graphemes(input));expect(segments).toEqual(["A","😀","e\u0301","👩‍👩‍👧‍👦"]);expect(Object.isFrozen(segments)).toBe(true);
 expect(value(await api.normalizeNFC("e\u0301"))).toBe("é");
 expect(value(await api.scalars(""))).toEqual([]);expect(value(await api.graphemes(""))).toEqual([]);expect(value(await api.fromScalars([]))).toBe("");
});

test("named Unicode operations reject unpaired surrogates and invalid scalar values",async()=>{
 for(const input of ["\ud800","\udfff","a\ud800b","\udc00\ud800"]){
  invalid(await api.scalars(input),1006);invalid(await api.graphemes(input),1006);invalid(await api.normalizeNFC(input),1006);
 }
 for(const scalar of [-1n,0xd800n,0xdfffn,0x110000n,10n**100n])invalid(await api.fromScalars([65n,scalar,66n]),1006);
 for(const scalar of [0n,0xd7ffn,0xe000n,0x10ffffn])expect(value(await api.fromScalars([scalar]))).toBe(String.fromCodePoint(Number(scalar)));
});

test("bulk fromCodePoint uses bounded native argument chunks",async()=>{
 const input=Object.freeze(Array.from({length:150000},(_,i)=>i%2===0?65n:0x1f600n));
 const result=value(await api.fromScalars(input));expect(result).toBe("A😀".repeat(75000));
 expect(input[1]).toBe(0x1f600n);
});
