import {test,expect} from "bun:test";
import {encodeQuestions,decodeAnswers,validateQuestions,QuestionIssue,AnswerIssue,type NoulDescriptor} from "../ai/typesafe.ts";
import {CodecIssue} from "../codec/budget.ts";
import {record} from "../data.ts";
import {ownBytes,copyBytes} from "../bytes.ts";
import type {Schema} from "../codec/json.ts";
const origin={source:"test:typesafe",start:0,end:0,invocation:[]};
const bytes=(s:string)=>ownBytes(new TextEncoder().encode(s));
const descriptor:NoulDescriptor={kind:"noul",instructions:"Evaluate `amount`",trueDescription:"Yes",falseDescription:"No",minimum:0.5};
const stateSchema:Schema={root:"state",nodes:[{identity:"state",kind:"record",name:"state",fields:[{name:"amount",type:"int"},{name:"zero",type:"float"}]},{identity:"int",kind:"primitive",name:"int"},{identity:"float",kind:"primitive",name:"float"}]};
const state=record("state",[["amount",9007199254740993n],["zero",-0]]);
const decode=(text:string,count=1)=>decodeAnswers(bytes(text),new Array(count).fill(descriptor),8192).map(answer=>{if(answer.kind!=="noul")throw Error("wrong answer kind");return answer.probability;});
function answerIssue(text:string,count:number,question:string,reason:AnswerIssue["reason"]){
 try{decode(text,count);throw Error("invalid answer accepted");}catch(e){expect(e).toBeInstanceOf(AnswerIssue);expect((e as AnswerIssue).question).toBe(question);expect((e as AnswerIssue).reason).toBe(reason);}
}

test("Noul wire request uses the real exact codec and local policy stays local",()=>{
 const encoded=encodeQuestions("jev-latest",stateSchema,state,[descriptor,{...descriptor,minimum:1}],8192);
 expect(new TextDecoder().decode(copyBytes(encoded,origin))).toBe('{"model":"jev-latest","state":{"amount":9007199254740993,"zero":-0},"questions":{"q0":{"type":"noul","instructions":"Evaluate `amount`","criteria":{"true":"Yes","false":"No"}},"q1":{"type":"noul","instructions":"Evaluate `amount`","criteria":{"true":"Yes","false":"No"}}}}');
 expect(()=>encodeQuestions("jev-latest",stateSchema,state,[descriptor],16)).toThrow(CodecIssue);
});

test("every Noul descriptor is checked, including unused registrations",()=>{
 for(const [change,reason] of [
  [{instructions:" \n"},"instructions"],[{instructions:"\ud800"},"instructions"],
  [{trueDescription:"\t"},"criterion"],[{falseDescription:"\udfff"},"criterion"],
  [{minimum:NaN},"threshold"],[{minimum:Infinity},"threshold"],[{minimum:-0.1},"threshold"],[{minimum:1.1},"threshold"],
 ] as const){
  try{validateQuestions([descriptor,{...descriptor,...change}]);throw Error("accepted descriptor");}catch(e){expect(e).toBeInstanceOf(QuestionIssue);expect((e as QuestionIssue).question).toBe("q1");expect((e as QuestionIssue).reason).toBe(reason);}
 }
 for(const count of [0,257])expect(()=>validateQuestions(new Array(count).fill(descriptor))).toThrow(QuestionIssue);
 validateQuestions(new Array(256).fill(descriptor));
 validateQuestions([{...descriptor,minimum:0},{...descriptor,minimum:1}]);
});

test("Noul accepts resolved model names, auxiliary metadata and both probability extremes",()=>{
 const p=decode('{"model":"resolved-model","usage":{"extra":null},"answers":{"q1":{"type":"noul","noul":1,"metadata":null},"q0":{"type":"noul","noul":-0}}}',2);
 expect(Object.is(p[0],-0)).toBe(true);expect(p[1]).toBe(1);expect(Object.isFrozen(decodeAnswers(bytes('{"model":"x","answers":{"q0":{"type":"noul","noul":0.5}}}'),[descriptor],8192))).toBe(true);
 expect(decode('{"model":"resolved","answers":{"q0":{"type":"noul","noul":0.5}}}')).toEqual([0.5]);
});

test("Noul validates the envelope, then IDs, then all answers in registration order",()=>{
 for(const text of ['null','[]','{}','{"model":""}','{"model":"x","answers":null}'])answerIssue(text,1,"","envelope");
 answerIssue('{"model":"x","answers":{"q0":{"type":"bad"},"extra":null}}',1,"","question_ids");
 answerIssue('{"model":"x","answers":{"q1":{"type":"noul","noul":0.5}}}',1,"","question_ids");
 answerIssue('{"model":"x","answers":{"q0":{"type":"choice","noul":0.5}}}',1,"q0","answer_type");
 answerIssue('{"model":"x","answers":{"q0":{"type":"noul"}}}',1,"q0","missing_value");
 for(const value of ['null','"0.5"','true','-0.1','1.1','1e999'])answerIssue('{"model":"x","answers":{"q0":{"type":"noul","noul":'+value+'}}}',1,"q0","probability");
 answerIssue('{"model":"x","answers":{"q1":{"type":"bad"},"q0":{"type":"noul"}}}',2,"q0","missing_value");
 answerIssue('{"model":"x","answers":{"q0":{"type":"noul","noul":0.5},"q1":{"type":"noul","noul":2}}}',2,"q1","probability");
});

test("provider envelopes share bounded UTF8, syntax and duplicate-member admission",()=>{
 for(const [text,reason] of [
  ['{"model":"x","answers":{"q0":{"type":"noul","noul":0,"noul":1}}}',"duplicate_member"],
  ['{"model":"x","model":"y",}',"invalid_json"],
  ['\ufeff{}',"invalid_json"],
  ['{"model":"\\ud800","answers":{}}',"unicode_scalar"],
  ['['.repeat(65)+'0'+']'.repeat(65),"depth_limit"],
 ] as const){try{decode(text);throw Error("accepted malformed document");}catch(e){expect(e).toBeInstanceOf(CodecIssue);expect((e as CodecIssue).reason).toBe(reason);}}
 expect(()=>decodeAnswers(ownBytes(new Uint8Array([255])),[descriptor],8192)).toThrow(CodecIssue);
});
