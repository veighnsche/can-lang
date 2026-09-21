import {test,expect} from "bun:test";
import {decodeResponse,encodeResponseRequest,ResponseIssue,type ResponseFormat} from "../ai/responses.ts";
import {CodecIssue} from "../codec/budget.ts";
import {ownBytes,copyBytes} from "../bytes.ts";
import {record} from "../data.ts";
import type {Schema} from "../codec/json.ts";
const origin={source:"test",start:0,end:0,invocation:[]};
const bytes=(value:unknown)=>ownBytes(new TextEncoder().encode(JSON.stringify(value)));
const schema:Schema={root:"state",nodes:[{identity:"state",kind:"record",name:"state",fields:[{name:"count",type:"int"}]},{identity:"int",kind:"primitive",name:"int"}]};
const format:ResponseFormat={name:"can_0123456789abcdef0123456789abcdef",schema:JSON.stringify({type:"object",properties:{count:{type:"integer"}},required:["count"],additionalProperties:false}),codec:schema};
const message=(content:unknown[])=>({type:"message",role:"assistant",status:"completed",content});
const part=(text:string)=>({type:"output_text",text});
const envelope=(output:unknown[]= [message([part("hello")])],extra:object={})=>({id:"response-1",object:"response",status:"completed",output,...extra});
function issue(value:unknown,kind:ResponseIssue["kind"],reason:string){try{decodeResponse(bytes(value),undefined,65536);throw Error("accepted");}catch(error){expect(error).toBeInstanceOf(ResponseIssue);expect((error as ResponseIssue).kind).toBe(kind);expect((error as ResponseIssue).reason).toBe(reason);}}
test("stateless request uses exact state codec and closed format",()=>{
 for(const output of [undefined,format]){
  const encoded=encodeResponseRequest("explicit-model"," instructions ",schema,record("state",[["count",9007199254740993n]]),2048,output,65536);
  expect(JSON.parse(new TextDecoder().decode(copyBytes(encoded,origin)))).toEqual({model:"explicit-model",instructions:" instructions ",input:'{"count":9007199254740993}',max_output_tokens:2048,store:false,stream:false,background:false,truncation:"disabled",text:{format:output?{type:"json_schema",name:format.name,strict:true,schema:JSON.parse(format.schema)}:{type:"text"}}});
 }
 expect(()=>encodeResponseRequest("m"," \n ",schema,record("state",[["count",1n]]),1,undefined,65536)).toThrow(ResponseIssue);
 expect(()=>encodeResponseRequest("m","\ud800",schema,record("state",[["count",1n]]),1,undefined,65536)).toThrow(CodecIssue);
});
test("text is concatenated exactly, including empty parts and reasoning",()=>{
 expect(decodeResponse(bytes(envelope([{type:"reasoning",id:"r",summary:[]},message([part("  a"),part(""),part("\nb  ")])])),undefined,65536)).toBe("  a\nb  ");
 expect(decodeResponse(bytes(envelope([message([part("")])])),undefined,65536)).toBe("");
});
test("generation envelope errors and status precedence",()=>{
 for(const bad of [null,{},envelope([],{id:""}),envelope([],{object:"other"}),envelope([],{status:"unknown"}),envelope([],{output:null}),envelope([],{status:"incomplete",id:7})])issue(bad,"invalid","envelope");
 issue(envelope([],{status:"incomplete",incomplete_details:{reason:"max_output_tokens"}}),"truncated","");
 issue(envelope([],{status:"incomplete",incomplete_details:{reason:"content_filter"}}),"refused","content_filter");
 for(const details of [undefined,null,{}, {reason:"unknown"}])issue(envelope([],{status:"incomplete",incomplete_details:details}),"invalid","incomplete_reason");
 for(const [status,reason] of [["failed","provider_failed"],["cancelled","provider_cancelled"],["queued","nonterminal_response"],["in_progress","nonterminal_response"]])issue(envelope([],{status}),"invalid",reason!);
 issue(envelope([],{error:{message:"private"}}),"invalid","provider_failed");
 for(const output of [[],[message([])],[message([part("x")]),message([part("y")])],[{...message([part("x")]),role:"user"}],[{type:"reasoning",id:"",summary:[]}]])issue(envelope(output),"invalid","output_shape");
 for(const item of [null,{type:"function_call"},{type:"web_search_call"}])issue(envelope([item]),"invalid","unsupported_output");
});
test("refusal validates all siblings and discards private prose",()=>{
 const refusal={type:"refusal",refusal:"private refusal"};
 issue(envelope([message([refusal,part("partial")])]),"refused","provider_refusal");
 for(const bad of [{type:"other"},{type:"output_text",text:3},{type:"refusal",refusal:null},null]){
  issue(envelope([message([refusal,bad])]),"invalid","output_shape");
  issue(envelope([message([bad,refusal])]),"invalid","output_shape");
 }
});
test("structured output retains exact tokens and rejects repairs",()=>{
 for(const token of ["9007199254740993","9007199254740993.0","90071992547409930e-1"]){
  expect(decodeResponse(bytes(envelope([message([part('{"count":'),part(token+"}")])])),format,65536)).toEqual(record("state",[["count",9007199254740993n]]));
 }
 for(const text of ['{"count":1.5}','{"count":null}','{"count":"1"}','{"count":1,"extra":0}','{"count":1,"count":2}','```json\n{"count":1}\n```','prefix {"count":1}','{}','{"count":1} trailing'])expect(()=>decodeResponse(bytes(envelope([message([part(text)])])),format,65536)).toThrow(CodecIssue);
 for(const selected of [undefined,format])expect(()=>decodeResponse(bytes(envelope([message([part("\ud800")])])),selected,65536)).toThrow(CodecIssue);
 expect(()=>decodeResponse(bytes(envelope()),undefined,2)).toThrow(CodecIssue);
 expect(()=>decodeResponse(ownBytes(new Uint8Array([255])),undefined,65536)).toThrow(CodecIssue);
});
