import {ownBytes,copyBytes,type Bytes} from "../bytes.ts";
import {CodecIssue} from "../codec/budget.ts";
import {mediaType} from "../transport/media.ts";
const origin=Object.freeze({source:"can:multipart",start:0,end:0,invocation:Object.freeze([])});
// Bounded multipart/form-data parser over an already-budgeted body. The
// body arrives whole (buffered ingress or first buffered access), so no new
// byte budget is needed; every part is a slice of those same bytes. Framing
// is strict CRLF, parts are flat (nested multipart is rejected, not
// recursed), and text decodes UTF-8 fatally while file bytes pass through.
export class MultipartIssue extends Error {constructor(readonly reason:"multipart_boundary"|"multipart_frame"|"multipart_nested"){super("invalid multipart");}}
export type MultipartField=Readonly<{name:string;value:string}>;
export type MultipartFile=Readonly<{name:string;filename:string;contentType:string;content:Bytes}>;
export type MultipartDocument=Readonly<{fields:readonly MultipartField[];files:readonly MultipartFile[]}>;
const text=new TextDecoder("utf-8",{fatal:true});
const encode=new TextEncoder();
function linesOf(body:Uint8Array):Uint8Array[]{
 // Split on CRLF only; lone bytes stay inside content lines.
 const lines:Uint8Array[]=[];let start=0;
 for(let i=0;i+1<body.byteLength;i++){
  if(body[i]===13&&body[i+1]===10){lines.push(body.subarray(start,i));start=i+2;i++;}
 }
 lines.push(body.subarray(start));
 return lines;
}
function asciiLine(line:Uint8Array):string{
 try{return text.decode(line);}catch{throw new MultipartIssue("multipart_frame");}
}
function unquote(value:string):string{
 if(!value.startsWith('"'))return value;
 if(value.length<2||!value.endsWith('"'))throw new MultipartIssue("multipart_frame");
 let out="";
 for(let i=1;i+1<value.length;i++){
  const char=value[i]!;
  if(char==="\\"){i++;if(i+1>=value.length)throw new MultipartIssue("multipart_frame");out+=value[i]!;}
  else{if(char==="\r"||char==="\n")throw new MultipartIssue("multipart_frame");out+=char;}
 }
 return out;
}
function dispositionOf(header:string):Readonly<{name:string;filename:string|undefined}>{
 const parts=header.split(";"),kind=parts[0]!.trim().toLowerCase();
 if(kind!=="form-data")throw new MultipartIssue("multipart_frame");
 let name:string|undefined,filename:string|undefined;
 for(const part of parts.slice(1)){
  const cut=part.indexOf("=");
  if(cut<0)throw new MultipartIssue("multipart_frame");
  const key=part.slice(0,cut).trim().toLowerCase(),value=unquote(part.slice(cut+1).trim());
  if(key==="name"){if(name!==undefined)throw new MultipartIssue("multipart_frame");name=value;}
  else if(key==="filename"){if(filename!==undefined)throw new MultipartIssue("multipart_frame");filename=value;}
 }
 if(name===undefined)throw new MultipartIssue("multipart_frame");
 return {name,filename};
}
export function decodeMultipart(input:unknown,boundary:string):MultipartDocument{
 if(!/^[A-Za-z0-9'()+_,\-./:=?]+$/.test(boundary)||boundary.length>70)throw new MultipartIssue("multipart_boundary");
 const body=new Uint8Array(copyBytes(input,origin));
 const lines=linesOf(body);
 const delimiter=encode.encode("--"+boundary),close=encode.encode("--"+boundary+"--");
 const same=(line:Uint8Array,marker:Uint8Array)=>line.byteLength===marker.byteLength&&line.every((value,at)=>value===marker[at]);
 const closedBy=(line:Uint8Array)=>same(line,close)||(line.byteLength>close.byteLength&&close.every((value,at)=>value===line[at])&&line.subarray(close.byteLength).every(value=>value===32||value===9));
 const fields:MultipartField[]=[],files:MultipartFile[]=[];
 let at=0;
 for(;;){
  if(at>=lines.length)throw new MultipartIssue("multipart_frame");
  const line=lines[at]!;
  // A premature close with no parts is an empty form; anything else before
  // the first delimiter is preamble and ignored. Delimiters compare as
  // bytes so binary content never fails header decoding.
  if(same(line,delimiter))break;
  if(closedBy(line))return {fields,files};
  at++;
 }
 at++;
 for(;;){
  const headerLines:string[]=[];
  for(;;){
   if(at>=lines.length)throw new MultipartIssue("multipart_frame");
   const line=asciiLine(lines[at]!);at++;
   if(line==="")break;
   if(line.startsWith(" ")||line.startsWith("\t"))throw new MultipartIssue("multipart_frame");
   headerLines.push(line);
  }
  let disposition:string|undefined,contentType:string|undefined;
  for(const header of headerLines){
   const cut=header.indexOf(":");
   if(cut<=0)throw new MultipartIssue("multipart_frame");
   const key=header.slice(0,cut).trim().toLowerCase(),value=header.slice(cut+1).trim();
   if(key==="content-disposition"){if(disposition!==undefined)throw new MultipartIssue("multipart_frame");disposition=value;}
   else if(key==="content-type"){if(contentType!==undefined)throw new MultipartIssue("multipart_frame");contentType=value;}
  }
  if(disposition===undefined)throw new MultipartIssue("multipart_frame");
  if(contentType!==undefined&&contentType.toLowerCase().startsWith("multipart/"))throw new MultipartIssue("multipart_nested");
  // Part content-types validate strict but store raw; parsing them is the
  // consumer's job.
  if(contentType!==undefined){try{mediaType(contentType);}catch(cause){if(cause instanceof CodecIssue)throw new MultipartIssue("multipart_frame");throw cause;}}
  const content:Uint8Array[]=[];let closed=false;
  for(;;){
   if(at>=lines.length)throw new MultipartIssue("multipart_frame");
   const raw=lines[at]!;at++;
   if(same(raw,delimiter))break;
   if(closedBy(raw)){closed=true;break;}
   content.push(raw);
  }
  const size=content.reduce((sum,line)=>sum+line.byteLength,0)+(content.length===0?0:(content.length-1)*2);
  const joined=new Uint8Array(size);let offset=0;
  for(const line of content){if(offset!==0){joined[offset]=13;joined[offset+1]=10;offset+=2;}joined.set(line,offset);offset+=line.byteLength;}
  const {name,filename}=dispositionOf(disposition);
  if(filename===undefined){
   let value:string;
   try{value=text.decode(joined);}catch{throw new CodecIssue("multipart","utf8");}
   fields.push({name,value});
  }else files.push({name,filename,contentType:contentType??"application/octet-stream",content:ownBytes(joined)});
  if(closed)return {fields,files};
 }
}
