import {array,record} from "../data.ts";
import {copyBytes,ownBytes} from "../bytes.ts";
import {encodeJSON,decodeJSON,type Schema,type SchemaNode} from "../codec/json.ts";
import {parseDocument} from "../codec/document.ts";
import {CodecIssue,reject} from "../codec/budget.ts";
import {invoke,success,failure,type Completion,type AssertionContext} from "../completion.ts";
import {createDomainRuntime} from "../domain.ts";
import type {FailureOrigin} from "../failure.ts";
import {denyLiveBoundary} from "../assert/context.ts";
import {providerHTTP} from "../assert/provider.ts";
import {createTransport,type HTTPTypes} from "../transport/http.ts";
import type {Connection} from "../transport/request.ts";

export type ResponseFormat=Readonly<{name:string;schema:string;codec:Schema}>;
export type ResponseTypes=HTTPTypes&Readonly<{invalidData:string;refused:string;truncated:string;invalidResponse:string}>;
export class ResponseIssue extends Error{
 constructor(readonly kind:"invalid"|"refused"|"truncated"|"instructions",readonly reason:string=""){super("invalid generation response");}
}
const object=(value:unknown):value is Record<string,unknown>=>value!==null&&typeof value==="object"&&!Array.isArray(value);
const text=(value:unknown):value is string=>typeof value==="string"&&value.length>0;
const origin=Object.freeze({source:"can:responses",start:0,end:0,invocation:Object.freeze([])});

// Only compiler-owned protocol structure enters this projection. Authored state
// is already an exact codec-encoded string, never a second untyped value graph.
function encodeProtocol(value:unknown,bytes:number){
 const prefix="can:responses-private/";
 const nodes:SchemaNode[]=[{identity:prefix+"str",kind:"primitive",name:"str"},{identity:prefix+"bool",kind:"primitive",name:"bool"},{identity:prefix+"int",kind:"primitive",name:"int"},{identity:prefix+"strings",kind:"array",name:"strings",element:prefix+"str"}];
 let serial=0;
 function project(value:unknown):Readonly<{type:string;value:unknown}>{
  if(typeof value==="string")return {type:prefix+"str",value};
  if(typeof value==="boolean")return {type:prefix+"bool",value};
  if(typeof value==="bigint")return {type:prefix+"int",value};
  if(Array.isArray(value)&&value.every(item=>typeof item==="string"))return {type:prefix+"strings",value:array(value)};
  if(!object(value))throw new TypeError("unsupported compiler protocol value");
  const identity=prefix+serial++,fields=Object.entries(value).map(([name,child])=>({name,...project(child)}));
  nodes.push({identity,kind:"record",name:"protocol",fields:fields.map(({name,type})=>({name,type}))});
  return {type:identity,value:record(identity,fields.map(({name,value})=>[name,value] as const))};
 }
 const root=project(value);return encodeJSON({root:root.type,nodes},root.value,bytes);
}
export function encodeResponseRequest(model:string,instructions:string,stateSchema:Schema,state:unknown,maxOutputTokens:number,format:ResponseFormat|undefined,bytes:number){
 if(!instructions.isWellFormed())reject("/instructions","unicode_scalar");
 if(instructions.trim()==="")throw new ResponseIssue("instructions","instructions");
 const input=new TextDecoder().decode(copyBytes(encodeJSON(stateSchema,state,bytes),origin));
 const outputFormat=format===undefined?{type:"text"}:{type:"json_schema",name:format.name,strict:true,schema:JSON.parse(format.schema)};
 return encodeProtocol({model,instructions,input,max_output_tokens:BigInt(maxOutputTokens),store:false,stream:false,background:false,truncation:"disabled",text:{format:outputFormat}},bytes);
}

export function decodeResponse(input:unknown,format:ResponseFormat|undefined,bytes:number):unknown{
 const {parsed}=parseDocument(input,bytes);
 const invalid=(reason:string):never=>{throw new ResponseIssue("invalid",reason);};
 if(!object(parsed)||!text(parsed.id)||parsed.object!=="response"||typeof parsed.status!=="string"||!["completed","incomplete","failed","cancelled","queued","in_progress"].includes(parsed.status)||!Array.isArray(parsed.output))return invalid("envelope");
 if(!parsed.id.isWellFormed())reject("/id","unicode_scalar");
 if(parsed.status==="incomplete"){
  const reason=object(parsed.incomplete_details)?parsed.incomplete_details.reason:undefined;
  if(reason==="max_output_tokens")throw new ResponseIssue("truncated");
  if(reason==="content_filter")throw new ResponseIssue("refused","content_filter");
  return invalid("incomplete_reason");
 }
 if(parsed.status==="failed")return invalid("provider_failed");
 if(parsed.status==="cancelled")return invalid("provider_cancelled");
 if(parsed.status==="queued"||parsed.status==="in_progress")return invalid("nonterminal_response");
 if(parsed.error!==undefined&&parsed.error!==null)return invalid("provider_failed");
 let message:Record<string,unknown>|undefined;
 for(const item of parsed.output){
  if(!object(item))return invalid("unsupported_output");
  if(item.type==="reasoning"){
   if(!text(item.id)||!Array.isArray(item.summary))return invalid("output_shape");
   continue;
  }
  if(item.type!=="message")return invalid("unsupported_output");
  if(message!==undefined||item.role!=="assistant"||item.status!=="completed"||!Array.isArray(item.content)||item.content.length===0)return invalid("output_shape");
  message=item;
 }
 if(message===undefined)return invalid("output_shape");
 const parts:string[]=[];let refused=false;
 for(const part of message.content as unknown[]){
  if(!object(part))return invalid("output_shape");
  if(part.type==="refusal"){
   if(typeof part.refusal!=="string")return invalid("output_shape");refused=true;
  }else if(part.type==="output_text"){
   if(typeof part.text!=="string")return invalid("output_shape");parts.push(part.text);
  }else return invalid("output_shape");
 }
 if(refused)throw new ResponseIssue("refused","provider_refusal");
 if(parts.length===0)return invalid("output_shape");
 const output=parts.join("");
 if(!output.isWellFormed())reject("","unicode_scalar");
 if(format!==undefined)return decodeJSON(format.codec,ownBytes(new TextEncoder().encode(output)),bytes);
 return output;
}

export function createResponses(domain:ReturnType<typeof createDomainRuntime>,types:ResponseTypes,readEnvironment:(name:string)=>string|undefined){
 const transport=createTransport(domain,types,readEnvironment);
 function issue(cause:unknown,where:FailureOrigin,operation:string,requestLimit?:number):Completion<never>{
  let identity:string,fields:readonly (readonly [string,unknown])[];
  let boundary:"native"|"emitted"="emitted";
  if(cause instanceof CodecIssue){
   boundary="native";
   if(cause.reason==="byte_limit"&&requestLimit!==undefined){identity=types.limit;fields=[["limit",BigInt(requestLimit)]];}
   else{identity=types.invalidData;fields=[["path",cause.path],["reason",cause.reason]];}
  }else if(cause instanceof ResponseIssue){
   identity=cause.kind==="instructions"?types.invalid:cause.kind==="invalid"?types.invalidResponse:cause.kind==="refused"?types.refused:types.truncated;
   fields=cause.kind==="truncated"?[]:[["reason",cause.reason]];
  }else throw cause;
  return failure(domain.create(identity,record(identity,fields),where,undefined,{boundary,operation}));
 }
 return Object.freeze({async generate<T>(connection:Connection,model:string,maxOutputTokens:number,instructions:string,stateSchema:Schema,state:unknown,format:ResponseFormat|undefined,where:FailureOrigin,operation:string,context?:AssertionContext):Promise<Completion<T>>{
  return invoke(async()=>{
   let body:Uint8Array;
   try{body=copyBytes(encodeResponseRequest(model,instructions,stateSchema,state,maxOutputTokens,format,connection.maxBodyBytes),where);}catch(cause){return issue(cause,where,operation,connection.maxBodyBytes);}
   const exchange=providerHTTP(context,where);if(exchange===undefined)denyLiveBoundary(context,where);
   return transport.request(connection,{path:"",method:"POST",query:[],headers:[{name:"content_type",value:"application/json"},{name:"accept",value:"application/json"}],body,exchange},bytes=>{
    try{return success(decodeResponse(ownBytes(bytes),format,connection.maxBodyBytes) as T);}catch(cause){return issue(cause,where,operation);}
   },where,operation);
  },where);
 }});
}
