import {reject,CodecIssue} from "../codec/budget.ts";
// Headers owns normalization. This narrow adapter rejects ambiguity and syntax
// that a permissive MIME parser might otherwise silently discard.
const token="[!#$%&'*+.^_`|~0-9A-Za-z-]+";
const parameter=new RegExp('^[ \t]*('+token+')[ \t]*=[ \t]*('+token+'|"(?:[^"\\\\\\r\\n]|\\\\[^\\r\\n])*")[ \t]*(;|$)');
export function mediaType(value:string):Readonly<{type:string;parameters:ReadonlyMap<string,string>}>{
 if(/[^\x09\x20-\x7e\x80-\xff]/.test(value))reject("","media_type");
 const head=new RegExp('^('+token+'/'+token+')[ \t]*(;|$)').exec(value);
 if(!head)reject("","media_type");
 let rest=value.slice(head[0].length);const params=new Map<string,string>();
 if(head[2]===";"&&!rest.trim())reject("","media_type");
 while(rest){const part=parameter.exec(rest);if(!part)reject("","media_type");const name=part[1].toLowerCase();if(params.has(name))reject("","media_type");let data=part[2];if(data.startsWith('"'))data=data.slice(1,-1).replace(/\\(.)/g,"$1");params.set(name,data);rest=rest.slice(part[0].length);if(part[3]===";"&&!rest.trim())reject("","media_type");}
 return {type:head[1].toLowerCase(),parameters:params};
}
export function responseMedia(value:string|undefined,json:boolean):void{
 if(value===undefined){if(json)reject("","media_type");return;}
 const parsed=mediaType(value);
 if(json&&parsed.type!=="application/json"&&!parsed.type.split("/")[1].endsWith("+json"))reject("","media_type");
 const charset=parsed.parameters.get("charset");if(charset!==undefined&&charset.toLowerCase()!=="utf-8")reject("","charset");
}
export function jsonRequestMedia(value:string):boolean{
 try{const parsed=mediaType(value);return parsed.type==="application/json"&&[...parsed.parameters].every(([key,value])=>key==="charset"&&value.toLowerCase()==="utf-8");}catch(cause){if(cause instanceof CodecIssue)return false;throw cause;}
}
