import {record} from "../data.ts";
import {copyBytes,ownBytes} from "../bytes.ts";
import {invoke,success,failure,type Completion,type AssertionContext} from "../completion.ts";
import {createDomainRuntime} from "../domain.ts";
import type {FailureOrigin} from "../failure.ts";
import {denyLiveBoundary} from "../assert/context.ts";
import {providerHTTP} from "../assert/provider.ts";
import {createTransport,type HTTPTypes} from "../transport/http.ts";
import type {Connection} from "../transport/request.ts";
import type {Schema} from "../codec/json.ts";
import {CodecIssue} from "../codec/budget.ts";

import {encodeQuestions,decodeAnswers,QuestionIssue,AnswerIssue,type QuestionDescriptor,type Answer} from "./questions.ts";
export {encodeQuestions,decodeAnswers,validateQuestions,QuestionIssue,AnswerIssue,type QuestionDescriptor,type NoulDescriptor,type Answer} from "./questions.ts";

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
 return Object.freeze({async ask(connection:Connection,model:string,stateSchema:Schema,state:unknown,questions:readonly QuestionDescriptor[],origin:FailureOrigin,context?:AssertionContext):Promise<Completion<readonly Answer[]>>{
  return invoke(async()=>{
   let body:Uint8Array;
   try{body=copyBytes(encodeQuestions(model,stateSchema,state,questions,connection.maxBodyBytes),origin);}
   catch(cause){return issue(cause,origin,connection.maxBodyBytes);}
   const exchange=providerHTTP(context,origin);
   if(exchange===undefined)denyLiveBoundary(context,origin);
   return transport.request(connection,{path:"",method:"POST",query:[],headers:[{name:"content_type",value:"application/json"},{name:"accept",value:"application/json"}],body,exchange},bytes=>{
    try{return success(decodeAnswers(ownBytes(bytes),questions,connection.maxBodyBytes));}
    catch(cause){return issue(cause,origin);}
   },origin);
  },origin);
 }});
}
