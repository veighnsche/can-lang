import {jsonRequestMedia} from "./media.ts";
import {transportFault,type RequestReason} from "./deadline.ts";
export type Entries=readonly Readonly<{name:string;value:string|readonly string[]}>[];
export type Connection=Readonly<{endpoint:string;timeoutMilliseconds:number;maxBodyBytes:number;bearerEnvironment?:string;headers:Entries}>;
const endpoints=new WeakMap<Connection,URL>();
const forbidden=new Set(["host","content-length","transfer-encoding","connection","upgrade","proxy-authorization","proxy-connection","keep-alive","te","trailer"]);
function invalid(reason:RequestReason):never{throw transportFault({kind:"invalid",reason});}
const scalar=(s:string)=>s.isWellFormed();
export function headerName(source:string):string{
 if(!/^[A-Za-z_][A-Za-z_0-9]*$/.test(source))invalid("header_name");
 const name=source.replaceAll("_","-").toLowerCase();if(forbidden.has(name))invalid("header_name");return name;
}
function applyHeaders(headers:Headers,entries:Entries,bearer:boolean):void{
 const seen=new Set<string>();
 for(const entry of entries){
  const name=headerName(entry.name);if(seen.has(name)||bearer&&name==="authorization")invalid("header_name");seen.add(name);
  headers.delete(name);
  for(const value of typeof entry.value==="string"?[entry.value]:entry.value){
   if(!scalar(value))invalid("header_value");
   try{headers.append(name,value);}catch(cause){if(cause instanceof TypeError)invalid("header_value");throw cause;}
  }
 }
}
export function endpointURL(endpoint:string):URL{
 if(!scalar(endpoint)||!endpoint)invalid("url");let url:URL;
 try{url=new URL(endpoint);}catch(cause){if(cause instanceof TypeError)invalid("url");throw cause;}
 if(url.protocol!=="http:"&&url.protocol!=="https:")invalid("url");
 if(url.username||url.password)invalid("userinfo");if(url.href.includes("#"))invalid("fragment");if(url.href.includes("?"))invalid("path_query");return url;
}
export function prepareRequest(connection:Connection,path:string,query:Entries,entries:Entries,readEnvironment:(name:string)=>string|undefined,bodyEncoding?:"json"|"text"|"bytes"):Readonly<{url:URL;headers:Headers}>{
 let endpoint=endpoints.get(connection);
 if(!endpoint){endpoint=endpointURL(connection.endpoint);endpoints.set(connection,endpoint);}
 if(!scalar(path))invalid("url");let url:URL;
 try{url=new URL(path,endpoint);}catch(cause){if(cause instanceof TypeError)invalid("url");throw cause;}
 if(url.protocol!=="http:"&&url.protocol!=="https:")invalid("url");
 if(url.username||url.password)invalid("userinfo");if(url.href.includes("#"))invalid("fragment");if(url.href.includes("?"))invalid("path_query");
 if(url.origin!==endpoint.origin)invalid("origin");
 const search=new URLSearchParams();
 for(const entry of query){
  if(!/^[A-Za-z_][A-Za-z_0-9]*$/.test(entry.name))invalid("query_value");
  for(const value of typeof entry.value==="string"?[entry.value]:entry.value){if(!scalar(value))invalid("query_value");search.append(entry.name,value);}
 }
 url.search=search.toString();const headers=new Headers();
 applyHeaders(headers,connection.headers,connection.bearerEnvironment!==undefined);
 applyHeaders(headers,entries,connection.bearerEnvironment!==undefined);
 if(bodyEncoding!==undefined){
  const contentType=headers.get("content-type");
  if(bodyEncoding==="json"&&contentType!==null&&!jsonRequestMedia(contentType))invalid("content_type");
  if(contentType===null)headers.set("content-type",bodyEncoding==="json"?"application/json":bodyEncoding==="text"?"text/plain; charset=utf-8":"application/octet-stream");
 }
 if(connection.bearerEnvironment!==undefined){
  const credential=readEnvironment(connection.bearerEnvironment);
  if(credential===undefined||credential==="")throw transportFault({kind:"credential"});
  if(!scalar(credential)||/[\r\n\0]/.test(credential))invalid("credential_value");
  try{headers.set("authorization",`Bearer ${credential}`);}catch(cause){if(cause instanceof TypeError)invalid("credential_value");throw cause;}
 }
 return Object.freeze({url,headers});
}
export function headerSnapshot(headers:Headers):readonly Readonly<{name:string;value:string}>[]{
 const result:{name:string;value:string}[]=[];
 for(const name of [...new Set(headers.keys())].sort()){
  for(const value of name==="set-cookie"?headers.getSetCookie():[headers.get(name)!])result.push(Object.freeze({name,value}));
 }
 return Object.freeze(result);
}
