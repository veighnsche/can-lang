import {test,expect} from "bun:test";
import {decodeForm,strictParameters,FormIssue,type FormSchema} from "../platform/form.ts";
import {CodecIssue} from "../codec/budget.ts";
import {ownBytes} from "../bytes.ts";
import {record,array} from "../data.ts";
const schema:FormSchema={root:"form",fields:[{name:"name",kind:"str"},{name:"aliases",kind:"array"},{name:"note",kind:"optional",some:"some",none:"none"}]};
const bytes=(text:string)=>ownBytes(new TextEncoder().encode(text));
function decode(text:string){return decodeForm(schema,bytes(text),new TextEncoder().encode(text).length);}
test("form records preserve repeated values, empty presence, Unicode and plus semantics",()=>{
 expect(decode("name=a+b&aliases=x&aliases=%2B&note=" )).toEqual(record("form",[["name","a b"],["aliases",array(["x","+"])],["note",record("some",[["value",""]])]]));
 expect(decode("name=%F0%9F%98%80" )).toEqual(record("form",[["name","😀"],["aliases",array([])],["note",record("none",[])]]));
 expect(decodeForm({root:"empty",fields:[]},bytes(""),0)).toEqual(record("empty",[]));
 // A caller's body limit constrains the form wire bytes, not an invented JSON encoding.
 expect(decode("name=x")).toEqual(record("form",[["name","x"],["aliases",array([])],["note",record("none",[])]]));
});
test("missing/repeated scalar and optional fields have finite form reasons",()=>{
 for(const [input,reason] of [["aliases=a","form_missing"],["name=a&name=b","form_repeated"],["name=a&note=x&note=y","form_repeated"]] as const){
  try{decode(input!);throw Error("accepted");}catch(e){expect(e).toBeInstanceOf(FormIssue);expect((e as FormIssue).reason).toBe(reason);}
 }
});
test("malformed percent syntax and invalid encoded UTF-8 are distinct",()=>{
 for(const value of ["%","%0","%GG","abc%2x"])expect(()=>decode("name="+value)).toThrow(FormIssue);
 for(const value of ["%FF","%C0%AF","%ED%A0%80","%F4%90%80%80","%E2%82"]){try{decode("name="+value);throw Error("accepted");}catch(e){expect(e).toBeInstanceOf(CodecIssue);expect((e as CodecIssue).reason).toBe("utf8");expect((e as CodecIssue).path).toBe("/name");}}
 expect(()=>decodeForm(schema,ownBytes(new Uint8Array([255])),1)).toThrow(CodecIssue);
});
test("unknown fields, hostile names, and body limits cannot alter shape",()=>{
 for(const name of ["extra","__proto__","constructor","toString"]){try{decode("name=x&"+name+"=bad");throw Error("accepted");}catch(e){expect(e).toBeInstanceOf(CodecIssue);expect((e as CodecIssue).reason).toBe("type");}}
 expect(()=>decodeForm(schema,bytes("name=x"),5)).toThrow(CodecIssue);
 expect(Array.from(strictParameters("a=%26%3D&x=+&x=%2b"))).toEqual([["a","&="],["x"," "],["x","+"]]);
});
