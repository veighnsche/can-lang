import {record} from "../data.ts";
import {copyBytes,ownBytes} from "../bytes.ts";
import {invoke,success,failure,type Completion,type AssertionContext} from "../completion.ts";
import {createDomainRuntime} from "../domain.ts";
import type {FailureOrigin} from "../failure.ts";
import {denyLiveBoundary} from "../assert/context.ts";
import {providerHTTP} from "../assert/provider.ts";
import {createTransport,type HTTPTypes} from "../transport/http.ts";
import type {Connection} from "../transport/request.ts";
import {encodeJSON,type Schema,type SchemaNode} from "../codec/json.ts";
import {parseDocument} from "../codec/document.ts";
import {CodecIssue,reject} from "../codec/budget.ts";

export type NoulDescriptor=Readonly<{instructions:string;trueDescription:string;falseDescription:string;minimum:number}>;
export class QuestionIssue extends Error {
 constructor(readonly question:string,readonly reason:"instructions"|"criterion"|"threshold"|"batch_count") {super("invalid AI question");}
}
export class AnswerIssue extends Error {
 constructor(readonly question:string,readonly reason:"envelope"|"question_ids"|"answer_type"|"missing_value"|"probability") {super("invalid AI answer");}
}

// Descriptor evaluation belongs to generated code. Validate the complete batch
// before encoding or giving the transport an opportunity to read credentials.
export function validateNoulQuestions(questions:readonly NoulDescriptor[]):void {
 if(questions.length<1||questions.length>256)throw new QuestionIssue("","batch_count");
 for(const [index,question] of questions.entries()){
  const id=`q${index}`;
  for(const [value,reason] of [[question.instructions,"instructions"],[question.trueDescription,"criterion"],[question.falseDescription,"criterion"]] as const){
   if(typeof value!=="string"||!value.isWellFormed()||value.trim()==="")throw new QuestionIssue(id,reason);
  }
  if(typeof question.minimum!=="number"||!Number.isFinite(question.minimum)||question.minimum<0||question.minimum>1)throw new QuestionIssue(id,"threshold");
 }
}

// The protocol wrapper uses the same schema-driven formatter as user state.
// Bigints never pass through Number or a second JSON serialization policy.
export function encodeNoulRequest(model:string,stateSchema:Schema,state:unknown,questions:readonly NoulDescriptor[],bytes:number){
 validateNoulQuestions(questions);
 const prefix="can:typesafe-private/";
 const id=(name:string)=>prefix+name;
 if(stateSchema.nodes.some(node=>node.identity.startsWith(prefix)))throw new TypeError("protocol schema identity collision");
 const fields=(entries:readonly (readonly [string,string])[])=>entries.map(([name,type])=>({name,type:id(type)}));
 const nodes:SchemaNode[]=[
  {identity:id("str"),kind:"primitive",name:"str"},
  {identity:id("criteria"),kind:"record",name:"criteria",fields:fields([["true","str"],["false","str"]])},
  {identity:id("noul"),kind:"record",name:"noul",fields:fields([["type","str"],["instructions","str"],["criteria","criteria"]])},
  {identity:id("questions"),kind:"record",name:"questions",fields:questions.map((_,i)=>({name:`q${i}`,type:id("noul")}))},
  {identity:id("request"),kind:"record",name:"request",fields:[{name:"model",type:id("str")},{name:"state",type:stateSchema.root},{name:"questions",type:id("questions")}]},
 ];
 const descriptors=questions.map((question,i)=>[`q${i}`,record(id("noul"),[["type","noul"],["instructions",question.instructions],["criteria",record(id("criteria"),[["true",question.trueDescription],["false",question.falseDescription]])]])] as const);
 return encodeJSON({root:id("request"),nodes:[...stateSchema.nodes,...nodes]},record(id("request"),[["model",model],["state",state],["questions",record(id("questions"),descriptors)]]),bytes);
}

function object(value:unknown):value is Record<string,unknown>{return value!==null&&typeof value==="object"&&!Array.isArray(value);}
export function decodeNoulAnswers(input:unknown,count:number,bytes:number):readonly number[]{
 const {parsed}=parseDocument(input,bytes);
 if(!object(parsed)||typeof parsed.model!=="string"||parsed.model.length===0||!object(parsed.answers))throw new AnswerIssue("","envelope");
 if(!parsed.model.isWellFormed())reject("/model","unicode_scalar");
 const answers=parsed.answers,ids=Object.keys(answers);
 // Diagnose the whole ID set before inspecting any individual answer.
 if(ids.length!==count||!Array.from({length:count},(_,i)=>`q${i}`).every(id=>Object.hasOwn(answers,id)))throw new AnswerIssue("","question_ids");
 const probabilities:number[]=[];
 for(let i=0;i<count;i++){
  const id=`q${i}`,answer=answers[id];
  if(!object(answer)||answer.type!=="noul")throw new AnswerIssue(id,"answer_type");
  if(!Object.hasOwn(answer,"noul"))throw new AnswerIssue(id,"missing_value");
  const p=answer.noul;
  if(typeof p!=="number"||!Number.isFinite(p)||p<0||p>1)throw new AnswerIssue(id,"probability");
  probabilities.push(p);
 }
 return Object.freeze(probabilities);
}

export type AITypes=HTTPTypes&Readonly<{invalidData:string;invalidQuestion:string;invalidAnswer:string}>;
export function createTypeSafe(domain:ReturnType<typeof createDomainRuntime>,types:AITypes,readEnvironment:(name:string)=>string|undefined){
 const transport=createTransport(domain,types,readEnvironment);
 function issue(cause:unknown,origin:FailureOrigin,requestLimit?:number):Completion<never>{
  let identity:string,fields:readonly (readonly [string,unknown])[];
  if(cause instanceof QuestionIssue){identity=types.invalidQuestion;fields=[["reason",cause.reason]];}
  else if(cause instanceof AnswerIssue){identity=types.invalidAnswer;fields=[["question",cause.question],["reason",cause.reason]];}
  else if(cause instanceof CodecIssue){
   if(cause.reason==="byte_limit"&&requestLimit!==undefined){identity=types.limit;fields=[["limit",BigInt(requestLimit)]];}
   else{identity=types.invalidData;fields=[["path",cause.path],["reason",cause.reason]];}
  }else throw cause;
  return failure(domain.create(identity,record(identity,fields),origin));
 }
 return Object.freeze({async noul(connection:Connection,model:string,stateSchema:Schema,state:unknown,questions:readonly NoulDescriptor[],origin:FailureOrigin,context?:AssertionContext):Promise<Completion<readonly number[]>>{
  return invoke(async()=>{
   let body:Uint8Array;
   try{body=copyBytes(encodeNoulRequest(model,stateSchema,state,questions,connection.maxBodyBytes),origin);}
   catch(cause){return issue(cause,origin,connection.maxBodyBytes);}
   const exchange=providerHTTP(context,origin);
   if(exchange===undefined)denyLiveBoundary(context,origin);
   return transport.request(connection,{path:"",method:"POST",query:[],headers:[{name:"content_type",value:"application/json"},{name:"accept",value:"application/json"}],body,exchange},bytes=>{
    try{return success(decodeNoulAnswers(ownBytes(bytes),questions.length,connection.maxBodyBytes));}
    catch(cause){return issue(cause,origin);}
   },origin);
  },origin);
 }});
}
