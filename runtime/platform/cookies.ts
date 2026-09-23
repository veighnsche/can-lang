// Typed request-cookie parsing and Set-Cookie serialization over
// Bun.CookieMap and Bun.Cookie. Parsing keeps every pair in wire order
// with first-wins lookup; construction validates each field natively
// and wraps an opaque handle, so serialization cannot fail. The
// adapter claims no prefix or SameSite policy: whatever the native
// constructor accepts serializes, and browser treatment is documented
// in the contract instead of enforced here.
import {success,failure,type Completion,type AssertionContext} from "../completion.ts";
import {record,recordIdentity,dataProperty,dataArray} from "../data.ts";
import {createDomainRuntime} from "../domain.ts";
const origin=Object.freeze({source:"can:cookies",start:0,end:0,invocation:Object.freeze([])});
export const COOKIE_KIND="cookie";
const jars=new WeakMap<object,Bun.Cookie>();
const object=(value:unknown):value is object=>value!==null&&(typeof value==="object"||typeof value==="function");
export function isCookieValue(kind:string|undefined,value:unknown):boolean{
  return kind===COOKIE_KIND&&object(value)&&jars.has(value);
}
const DATE_LIMIT=8640000000000000n;
type Ids=Readonly<{invalid:string;collection:string;pair:string;some:string;none:string;samesiteStrict:string;samesiteLax:string;samesiteNone:string}>;
export function createCookies(domain:ReturnType<typeof createDomainRuntime>,ids:Ids){
  const invalid=(reason:string)=>failure(domain.create(ids.invalid,record(ids.invalid,[["reason",reason]]),origin));
  const mint=(jar:Bun.Cookie):object=>{
    const handle=Object.freeze(Object.create(null));
    jars.set(handle,jar);
    return handle;
  };
  const project=(value:unknown):Bun.Cookie|undefined=>object(value)?jars.get(value):undefined;
  const option=(value:unknown):unknown=>{
    const identity=recordIdentity(value);
    if(identity===ids.some)return dataProperty(value,"value");
    if(identity===ids.none)return undefined;
    throw new TypeError("invalid compiler cookie option");
  };
  const sameSite=(value:unknown):("strict"|"lax"|"none")=>{
    const identity=recordIdentity(value);
    if(identity===ids.samesiteStrict)return "strict";
    if(identity===ids.samesiteLax)return "lax";
    if(identity===ids.samesiteNone)return "none";
    throw new TypeError("invalid compiler cookie same-site");
  };
  // Each field validates through its own minimal native construction so
  // a TypeError attributes to exactly one reason; the final build then
  // cannot throw for any validated part.
  const checkName=(name:string):ReturnType<typeof invalid>|undefined=>{
    try{new Bun.Cookie(name,"");}
    catch{return invalid("name");}
    return undefined;
  };
  const checkPath=(path:string):ReturnType<typeof invalid>|undefined=>{
    try{new Bun.Cookie("can-cookie-probe","",{path});}
    catch{return invalid("path");}
    return undefined;
  };
  const checkDomain=(domainName:string):ReturnType<typeof invalid>|undefined=>{
    try{new Bun.Cookie("can-cookie-probe","",{domain:domainName});}
    catch{return invalid("domain");}
    return undefined;
  };
  return Object.freeze({
    async parse(header:unknown,_context?:AssertionContext):Promise<Completion<unknown>>{
      if(typeof header!=="string")throw new TypeError("invalid compiler cookie header");
      const pairs:unknown[]=[];
      for(const entry of new Bun.CookieMap(header) as unknown as Iterable<readonly [unknown,unknown]>){
        const text=typeof entry[1]==="string"?entry[1]:(entry[1] as {value?:unknown} | null)?.value;
        if(typeof entry[0]!=="string"||typeof text!=="string")throw new TypeError("invalid native cookie pair");
        pairs.push(record(ids.pair,[["name",entry[0]],["value",text]]));
      }
      return success(record(ids.collection,[["pairs",Object.freeze(pairs)]]));
    },
    async get(collection:unknown,name:unknown,_context?:AssertionContext):Promise<Completion<unknown>>{
      if(typeof name!=="string")throw new TypeError("invalid compiler cookie name");
      for(const entry of dataArray(dataProperty(collection,"pairs"))){
        if(dataProperty(entry,"name")===name)return success(record(ids.some,[["value",dataProperty(entry,"value")]]));
      }
      return success(record(ids.none,[]));
    },
    async make(name:unknown,value:unknown,attributes:unknown,_context?:AssertionContext):Promise<Completion<unknown>>{
      if(typeof name!=="string"||typeof value!=="string")throw new TypeError("invalid compiler cookie");
      const path=dataProperty(attributes,"path"),domainName=option(dataProperty(attributes,"domain"));
      const secure=dataProperty(attributes,"secure"),httpOnly=dataProperty(attributes,"http_only");
      if(typeof path!=="string"||(domainName!==undefined&&typeof domainName!=="string")||typeof secure!=="boolean"||typeof httpOnly!=="boolean")throw new TypeError("invalid compiler cookie attributes");
      const site=sameSite(dataProperty(attributes,"same_site"));
      const maxAge=option(dataProperty(attributes,"max_age")),expiresMs=option(dataProperty(attributes,"expires_ms"));
      if((maxAge!==undefined&&typeof maxAge!=="bigint")||(expiresMs!==undefined&&typeof expiresMs!=="bigint"))throw new TypeError("invalid compiler cookie attributes");
      if(maxAge!==undefined&&!Number.isSafeInteger(Number(maxAge)))return invalid("max_age");
      if(expiresMs!==undefined&&(expiresMs>DATE_LIMIT||expiresMs<-DATE_LIMIT))return invalid("expires");
      const badName=checkName(name);
      if(badName!==undefined)return badName;
      const badPath=checkPath(path);
      if(badPath!==undefined)return badPath;
      if(domainName!==undefined){
        const badDomain=checkDomain(domainName);
        if(badDomain!==undefined)return badDomain;
      }
      return success(mint(new Bun.Cookie(name,value,{path,...(domainName===undefined?{}:{domain:domainName}),secure,httpOnly,sameSite:site,...(maxAge===undefined?{}:{maxAge:Number(maxAge)}),...(expiresMs===undefined?{}:{expires:new Date(Number(expiresMs))})})));
    },
    async serialize(cookie:unknown,_context?:AssertionContext):Promise<Completion<string>>{
      const jar=project(cookie);
      if(jar===undefined)throw new TypeError("invalid compiler cookie handle");
      return success(jar.toString());
    },
    async remove(name:unknown,path:unknown,domainName:unknown,_context?:AssertionContext):Promise<Completion<unknown>>{
      if(typeof name!=="string"||typeof path!=="string")throw new TypeError("invalid compiler cookie tombstone");
      const domainValue=option(domainName);
      if(domainValue!==undefined&&typeof domainValue!=="string")throw new TypeError("invalid compiler cookie tombstone");
      const badName=checkName(name);
      if(badName!==undefined)return badName;
      const badPath=checkPath(path);
      if(badPath!==undefined)return badPath;
      if(domainValue!==undefined){
        const badDomain=checkDomain(domainValue);
        if(badDomain!==undefined)return badDomain;
      }
      return success(mint(new Bun.Cookie(name,"",{path,...(domainValue===undefined?{}:{domain:domainValue}),expires:new Date(0)})));
    },
  });
}
