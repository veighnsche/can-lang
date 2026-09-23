import {success,failure,type AssertionContext} from "../completion.ts";
import {record,dataArray} from "../data.ts";
import {createDomainRuntime} from "../domain.ts";
import {resourceStateFailure} from "../failure.ts";
const origin=Object.freeze({source:"can:html",start:0,end:0,invocation:Object.freeze([])});
type Node=Readonly<{html:string;tag:string;head:boolean;anchor:boolean;form:boolean}>;
type Attribute=Readonly<{name:string;value:string;kind:"text"|"url"|"htmx"}>;
type Address=Readonly<{value:string;local:boolean}>;
const nodes=new WeakMap<object,Node>(),safe=new WeakMap<object,string>(),tags=new WeakMap<object,string>(),attributes=new WeakMap<object,Attribute>(),urls=new WeakMap<object,Address>(),targets=new WeakMap<object,string>();
function token<T>(map:WeakMap<object,T>,data:T):unknown{const value=Object.freeze(Object.create(null));map.set(value,data);return value;}
function read<T>(map:WeakMap<object,T>,value:unknown):T{if(value===null||(typeof value!=="object"&&typeof value!=="function")||!map.has(value))throw resourceStateFailure(undefined,origin);return map.get(value)!;}
export function renderSafe(value:unknown):string{return read(safe,value);}
export function isHTMLValue(kind:string|undefined,value:unknown):boolean{const map=kind==="node"?nodes:kind==="safe"?safe:kind==="url"?urls:kind==="tag"?tags:kind==="attribute"?attributes:kind==="target"?targets:undefined;return map!==undefined&&value!==null&&(typeof value==="object"||typeof value==="function")&&map.has(value);}
function string(value:string):string{if(typeof value!=="string")throw new TypeError("invalid HTML string");return value;}
const lower=(value:string)=>string(value).replace(/[A-Z]/g,c=>c.toLowerCase());
const authorTags=new Set("main header footer nav section article aside h1 h2 h3 h4 h5 h6 p div span ul ol li a form label input textarea select option button table thead tbody tr th td dl dt dd strong em small br hr code pre blockquote img del".split(" "));
const voidTags=new Set(["input","br","hr","img"]);
const globals=new Set("id class title lang dir hidden tabindex role".split(" "));
const applicability:Readonly<Record<string,readonly string[]>>=Object.freeze({name:["form","input","textarea","select","button"],value:["input","option","button","li"],type:["input","button","a","ol"],placeholder:["input","textarea"],autocomplete:["form","input","textarea","select"],for:["label"],method:["form"],rel:["a","form"],checked:["input"],selected:["option"],disabled:["input","textarea","select","option","button"],required:["input","textarea","select"],multiple:["input","select"],rows:["textarea"],cols:["textarea"],scope:["th"],colspan:["td","th"],rowspan:["td","th"],alt:["img"],align:["td","th"],start:["ol"]});
const inputTypes=new Set("hidden text search tel url email password date month week time datetime-local number range color checkbox radio file submit image reset button".split(" "));
const relations=new Set("alternate author bookmark external help license next nofollow noopener noreferrer opener prev privacy-policy search tag terms-of-service".split(" "));
const autocompleteFields=new Set("name honorific-prefix given-name additional-name family-name honorific-suffix nickname username new-password current-password one-time-code organization-title organization street-address address-line1 address-line2 address-line3 address-level4 address-level3 address-level2 address-level1 country country-name postal-code cc-name cc-given-name cc-additional-name cc-family-name cc-number cc-exp cc-exp-month cc-exp-year cc-csc cc-type transaction-currency transaction-amount language bday bday-day bday-month bday-year sex url photo tel tel-country-code tel-national tel-area-code tel-local tel-local-prefix tel-local-suffix tel-extension email impp".split(" "));
function autocomplete(value:string):boolean{
 const parts=lower(value).replace(/^[\t\n\f\r ]+|[\t\n\f\r ]+$/g,"").split(/[\t\n\f\r ]+/);if(parts.length===1&&(parts[0]==="on"||parts[0]==="off"))return true;
 if(parts[0]?.startsWith("section-")&&parts[0].length>8)parts.shift();
 if(parts[0]==="shipping"||parts[0]==="billing")parts.shift();
 if(["home","work","mobile","fax","pager"].includes(parts[0]??"")){parts.shift();if(!/^(tel(?:-[a-z-]+)?|email|impp)$/.test(parts[0]??""))return false;}
 if(!autocompleteFields.has(parts.shift()??""))return false;
 if(parts[0]==="webauthn")parts.shift();return parts.length===0;
}
function validValue(name:string,value:string,tag?:string):boolean{
 const v=lower(value);
 if(name==="dir")return ["ltr","rtl","auto"].includes(v);
 if(name==="hidden")return ["","hidden","until-found"].includes(v);
 if(["checked","selected","disabled","required","multiple"].includes(name))return v===""||v===name;
 if(name==="method")return ["get","post","dialog"].includes(v);
 if(name==="scope")return ["row","col","rowgroup","colgroup"].includes(v);
 if(name==="align")return ["left","center","right"].includes(v);
 if(name==="autocomplete")return tag==="form"?["on","off"].includes(v):autocomplete(value);
 if(name==="rel"){const parts=v.replace(/^[\t\n\f\r ]+|[\t\n\f\r ]+$/g,"").split(/[\t\n\f\r ]+/);return parts.length>0&&parts.every(p=>relations.has(p)&&!(tag==="form"&&["alternate","author","bookmark","privacy-policy","tag","terms-of-service"].includes(p)));}
 if(name==="type"&&tag!==undefined){if(tag==="input")return inputTypes.has(v);if(tag==="button")return ["submit","reset","button"].includes(v);if(tag==="ol")return ["1","a","A","i","I"].includes(value);}
 if(["rows","cols","colspan","rowspan"].includes(name)){if(!/^[0-9]+$/.test(value))return false;const n=BigInt(value);return name==="rowspan"?n<=65534n:n>=1n&&(name!=="colspan"||n<=1000n);}
 if(name==="tabindex"||name==="start"||(name==="value"&&tag==="li"))return /^-?[0-9]+$/.test(value);
 return true;
}
const attr=(name:string,value:string,kind:Attribute["kind"]="htmx")=>token(attributes,Object.freeze({name,value,kind}));
const node=(html:string,tag="",head=false,anchor=false,form=false)=>token(nodes,Object.freeze({html,tag,head,anchor,form}));
const children=(input:readonly unknown[])=>dataArray(input).map(v=>read(nodes,v));
const serialize=(a:Attribute)=>` ${a.name}="${Bun.escapeHTML(a.value)}"`;
const selectorID=(value:string)=>/^[A-Za-z_][A-Za-z0-9_-]*$/.test(string(value));
type Contracts=Readonly<{structure:string;url:string;target:string;interval:string}>;
export function createHTML(domain:ReturnType<typeof createDomainRuntime>,types:Contracts,declared:readonly string[]=[]){
 const bad=(identity:string,reason:string)=>failure(domain.create(identity,record(identity,[["reason",reason]]),origin));
 const structure=(reason:string)=>bad(types.structure,reason);
 const interval=(n:bigint)=>failure(domain.create(types.interval,record(types.interval,[["milliseconds",n]]),origin));
 const local=(input:unknown,name:string)=>{const url=read(urls,input);return url.local?success(attr(name,url.value)):bad(types.url,"same_origin");};
 const declaredURLs=new Set(declared);
 return Object.freeze({
  async makeTag(name:string,_context?:AssertionContext){name=lower(name);return authorTags.has(name)?success(token(tags,name)):structure("tag");},
  async text(value:string,_context?:AssertionContext){return success(node(Bun.escapeHTML(string(value))));},
  async textFragment(value:string,_context?:AssertionContext){return success(token(safe,Bun.escapeHTML(string(value))));},
  async parseURL(value:string,_context?:AssertionContext){
   string(value);if(!value.isWellFormed()||/[\x00-\x1f\x7f\\]/.test(value)||value.startsWith("//"))return bad(types.url,"syntax");
   const local=value.startsWith("/");if(!local&&!/^https:\/\//i.test(value))return bad(types.url,"scheme");
   let parsed:URL;try{parsed=new URL(value,"https://can.invalid/");}catch(cause){if(!(cause instanceof TypeError))throw cause;return bad(types.url,"syntax");}
   if(parsed.protocol!=="https:"||parsed.username!==""||parsed.password!=="")return bad(types.url,"authority");
   const output=local?parsed.pathname+parsed.search+parsed.hash:parsed.href;
   if(local&&(parsed.origin!=="https://can.invalid"||output.startsWith("//")))return bad(types.url,"same_origin");
   return success(token(urls,Object.freeze({value:output,local})));
  },
  async textAttribute(name:string,value:string,_context?:AssertionContext){
   name=lower(name);string(value);
   if(!globals.has(name)&&!/^aria-[a-z][a-z0-9-]*$/.test(name)&&!Object.hasOwn(applicability,name))return structure("attribute");
   if(!validValue(name,value))return structure("attribute_value");return success(attr(name,value,"text"));
  },
  async urlAttribute(name:string,url:unknown,_context?:AssertionContext){name=lower(name);const value=read(urls,url);return ["href","action","formaction","src"].includes(name)?success(attr(name,value.value,"url")):structure("attribute");},
  async element(tag:unknown,inputAttributes:readonly unknown[],inputChildren:readonly unknown[],_context?:AssertionContext){
   const name=read(tags,tag),attrs=dataArray(inputAttributes).map(v=>read(attributes,v)),kids=children(inputChildren),seen=new Set<string>();
   for(const a of attrs){
    if(seen.has(a.name))return structure("duplicate_attribute");seen.add(a.name);
    if(a.kind==="text"&&Object.hasOwn(applicability,a.name)&&!applicability[a.name]!.includes(name))return structure("attribute_tag");
    if(a.kind==="text"&&!validValue(a.name,a.value,name))return structure("attribute_value");
    if(a.kind==="url"&&({href:"a",action:"form",formaction:"button",src:"img"} as Record<string,string>)[a.name]!==name)return structure("attribute_tag");
   }
   if(kids.some(k=>k.head))return structure("head_context");
   if(voidTags.has(name)&&kids.length!==0)return structure("void_children");
   const required:Readonly<Record<string,readonly string[]>>={ul:["li"],ol:["li"],select:["option"],thead:["tr"],tbody:["tr"],tr:["th","td"]};
   if(Object.hasOwn(required,name)&&kids.some(k=>!required[name]!.includes(k.tag)))return structure("children");
   if(name==="table"&&!((kids.length===1&&kids[0]!.tag==="tbody")||(kids.length===2&&kids[0]!.tag==="thead"&&kids[1]!.tag==="tbody")))return structure("children");
   if(name==="dl"&&(kids.length%2!==0||kids.some((k,i)=>k.tag!==(i%2===0?"dt":"dd"))))return structure("children");
   if(name==="a"&&kids.some(k=>k.anchor)||name==="form"&&kids.some(k=>k.form))return structure("nested_element");
   const output=`<${name}${attrs.map(serialize).join("")}>`+(voidTags.has(name)?"":kids.map(k=>k.html).join("")+`</${name}>`);
   return success(node(output,name,false,name==="a"||kids.some(k=>k.anchor),name==="form"||kids.some(k=>k.form)));
  },
  async fragment(input:readonly unknown[],_context?:AssertionContext){const kids=children(input);return kids.some(k=>k.head)?structure("head_context"):success(token(safe,kids.map(k=>k.html).join("")));},
  async stylesheet(input:unknown,_context?:AssertionContext){const url=read(urls,input);return success(node(`<link rel="stylesheet" href="${Bun.escapeHTML(url.value)}">`,"link",true));},
  async metaViewport(_context?:AssertionContext){return success(node('<meta name="viewport" content="width=device-width, initial-scale=1">',"meta",true));},
  async document(title:string,head:readonly unknown[],body:readonly unknown[],_context?:AssertionContext){
   const h=children(head),b=children(body);if(h.some(n=>!n.head)||b.some(n=>n.head))return structure("document_context");
   return success(token(safe,`<!doctype html><html><head><title>${Bun.escapeHTML(string(title))}</title>${h.map(n=>n.html).join("")}</head><body>${b.map(n=>n.html).join("")}</body></html>`));
  },
  async get(url:unknown,_context?:AssertionContext){return local(url,"hx-get");},
  async post(url:unknown,_context?:AssertionContext){return local(url,"hx-post");},
  async targetID(id:string,_context?:AssertionContext){return selectorID(id)?success(token(targets,"#"+id)):bad(types.target,"id");},
  async targetAttribute(target:unknown,_context?:AssertionContext){return success(attr("hx-target",read(targets,target)));},
  async indicatorID(id:string,_context?:AssertionContext){return selectorID(id)?success(attr("hx-indicator","#"+id)):bad(types.target,"id");},
  async swapInner(_context?:AssertionContext){return success(attr("hx-swap","innerHTML"));},
  async swapOuter(_context?:AssertionContext){return success(attr("hx-swap","outerHTML"));},
  async triggerChange(_context?:AssertionContext){return success(attr("hx-trigger","change"));},
  async triggerInputChanged(delay:bigint,_context?:AssertionContext){return delay<0n||delay>60000n?interval(delay):success(attr("hx-trigger",`input changed delay:${delay}ms`));},
  async triggerEvery(period:bigint,_context?:AssertionContext){return period<1000n||period>3600000n?interval(period):success(attr("hx-trigger",`every ${period}ms`));},
  async disableThis(_context?:AssertionContext){return success(attr("hx-disable","this"));},
  async declareAsset(url:string,_context?:AssertionContext){
   if(typeof url!=="string"||!declaredURLs.has(url))throw new TypeError("undeclared asset url");
   return success(token(urls,Object.freeze({value:url,local:true})));
  },
  async rejectAsset(reason:string,_context?:AssertionContext){if(reason!=="missing"&&reason!=="unowned")throw new TypeError("invalid asset reason");return bad(types.url,reason);},
  async runtimeHead(_context?:AssertionContext){
   // htmx 4 swaps every status except noSwap entries. The compiler-owned
   // policy keeps 204/304 quiet and every 4xx/5xx except 422 out of swaps,
   // with same-origin fetch pinned explicitly.
   const noSwap=[204,304];for(let code=400;code<600;code++)if(code!==422)noSwap.push(code);
   const config=JSON.stringify({mode:"same-origin",noSwap});
   return success(node(`<meta name="htmx-config" content="${Bun.escapeHTML(config)}"><script defer src="/__can/assets/htmx-4.0.0.min.js" integrity="sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc"></script>`,"runtime",true));
  }
 });
}
