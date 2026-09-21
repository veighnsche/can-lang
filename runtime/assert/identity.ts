import {createHash} from "node:crypto";

export type RootIdentity=Readonly<{package:string;declaration:string;name:string}>;
export type CallablePath=Readonly<{site:string;occurrence:number;instance:string}>;
export type PathSegment=Readonly<{site:string;occurrence:number;participant:readonly number[];callable?:CallablePath}>;
export type InvocationPath=Readonly<{root:RootIdentity;segments:readonly PathSegment[]}>;
declare const invocationBrand:unique symbol;
declare const callableBrand:unique symbol;
export type InvocationIdentity=Readonly<{[invocationBrand]:true}>;
export type CallableIdentity=Readonly<{[callableBrand]:true}>;
type Frame={family:object;path:InvocationPath;participant:readonly number[];occurrences:Map<string,number>};
type Instance={family:object|null;path:CallablePath;captures:readonly unknown[]};
const frames=new WeakMap<object,Frame>(),instances=new WeakMap<object,Instance>();
const token=<T>():T=>Object.freeze(Object.create(null)) as T;
function frame(identity:InvocationIdentity):Frame{
 const value=identity!==null&&typeof identity==="object"?frames.get(identity):undefined;
 if(!value)throw new TypeError("invalid invocation identity");
 return value;
}
function instance(identity:CallableIdentity):Instance{
 const value=identity!==null&&typeof identity==="object"?instances.get(identity):undefined;
 if(!value)throw new TypeError("invalid callable identity");
 return value;
}
function siteParts(site:string):readonly [string,number]{
 if(typeof site!=="string")throw new TypeError("invalid lexical site");
 const split=site.lastIndexOf("#"),owner=site.slice(0,split),ordinal=site.slice(split+1);
 if(split<=0||!/^(0|[1-9][0-9]*)$/.test(ordinal)||!Number.isSafeInteger(Number(ordinal)))throw new TypeError("invalid lexical site");
 return [owner,Number(ordinal)];
}
function occurrence(parent:Frame,site:string):number{
 siteParts(site);
 const next=parent.occurrences.get(site)??0;
 if(!Number.isSafeInteger(next+1))throw new TypeError("invocation occurrence overflow");
 parent.occurrences.set(site,next+1);return next;
}
function make(parent:Frame,segment:PathSegment):InvocationIdentity{
 const identity=token<InvocationIdentity>();
 frames.set(identity,{family:parent.family,path:Object.freeze({root:parent.path.root,segments:Object.freeze([...parent.path.segments,segment])}),participant:segment.participant,occurrences:new Map()});
 return identity;
}
export function rootIdentity(root:RootIdentity):InvocationIdentity{
 if(!root||![root.package,root.declaration,root.name].every(value=>typeof value==="string"&&value.length>0))throw new TypeError("invalid assertion root identity");
 const identity=token<InvocationIdentity>();
 frames.set(identity,{family:Object.freeze({}),path:Object.freeze({root:Object.freeze({...root}),segments:Object.freeze([])}),participant:Object.freeze([]),occurrences:new Map()});
 return identity;
}
export function invocationIdentity(parentIdentity:InvocationIdentity,site:string,callable?:CallableIdentity):InvocationIdentity{
 const parent=frame(parentIdentity),receipt=callable===undefined?undefined:instance(callable);
 if(receipt&&receipt.family!==null&&receipt.family!==parent.family)throw new TypeError("callable belongs to another assertion root");
 const segment=Object.freeze({site,occurrence:occurrence(parent,site),participant:parent.participant,...(receipt?{callable:receipt.path}:{})});
 return make(parent,segment);
}
// One coordination occurrence reserves the complete flattened batch before any
// body starts. Direct entries use [writtenIndex]; spreads use [writtenIndex,i].
export function participantIdentities(parentIdentity:InvocationIdentity,site:string,positions:readonly (readonly number[])[]):readonly InvocationIdentity[]{
 const parent=frame(parentIdentity),seen=new Set<string>();let previous:readonly number[]|undefined;
 for(const position of positions){
  if((position.length!==1&&position.length!==2)||position.some(value=>!Number.isSafeInteger(value)||value<0))throw new TypeError("invalid participant path");
  const key=JSON.stringify(position);if(seen.has(key))throw new TypeError("duplicate participant path");seen.add(key);
  if(previous&&(compareNumbers(previous,position)>=0||previous[0]===position[0]&&(previous.length===1||position.length===1)))throw new TypeError("participant paths are not in flattened source order");
  previous=position;
 }
 const visit=occurrence(parent,site);
 return Object.freeze(positions.map(position=>make(parent,Object.freeze({site,occurrence:visit,participant:Object.freeze([...parent.participant,...position])}))));
}
export function callableIdentity(parentIdentity:InvocationIdentity,site:string,captures:readonly unknown[]):CallableIdentity{
 const parent=frame(parentIdentity),visit=occurrence(parent,site),identity=token<CallableIdentity>();
 // The digest identifies the creation path, not a serialization of private
 // receiver/near values. Frozen capture references stay in the private receipt.
 const key=createHash("sha256").update("can-callable-instance-v1\0"+JSON.stringify([parent.path,site,visit])).digest("hex");
 instances.set(identity,{family:parent.family,path:Object.freeze({site,occurrence:visit,instance:key}),captures:Object.freeze([...captures])});
 return identity;
}
export function invocationPath(identity:InvocationIdentity):InvocationPath{return frame(identity).path;}
export function callableCaptures(identity:CallableIdentity):readonly unknown[]{return instance(identity).captures;}
const compareText=(a:string,b:string)=>a<b?-1:a>b?1:0;
function compareNumbers(a:readonly number[],b:readonly number[]):number{
 for(let i=0;i<Math.min(a.length,b.length);i++)if(a[i]!==b[i])return a[i]<b[i]?-1:1;
 return Math.sign(a.length-b.length);
}
export function compareInvocations(left:InvocationIdentity,right:InvocationIdentity):number{
 const a=frame(left),b=frame(right);
 if(a.family!==b.family)throw new TypeError("cannot order different assertion roots");
 const x=a.path.segments,y=b.path.segments;
 for(let i=0;i<Math.min(x.length,y.length);i++){
  const [xd,xn]=siteParts(x[i].site),[yd,yn]=siteParts(y[i].site);
  const difference=compareText(xd,yd)||Math.sign(xn-yn)||Math.sign(x[i].occurrence-y[i].occurrence)||compareNumbers(x[i].participant,y[i].participant)||compareText(x[i].callable?.instance??"",y[i].callable?.instance??"");
  if(difference)return difference;
 }
 return Math.sign(x.length-y.length);
}

// Pure initialization precedes assertion roots. These immutable creation
// receipts may be embedded in multiple isolated roots; they own no queue state.
const initializationOccurrences=new Map<string,number>();
export function initializationCallableIdentity(site:string,captures:readonly unknown[]):CallableIdentity{
 siteParts(site);
 const visit=initializationOccurrences.get(site)??0;
 if(!Number.isSafeInteger(visit+1))throw new TypeError("callable occurrence overflow");
 initializationOccurrences.set(site,visit+1);
 const identity=token<CallableIdentity>();
 const key=createHash("sha256").update("can-initialization-callable-v1\0"+JSON.stringify([site,visit])).digest("hex");
 instances.set(identity,{family:null,path:Object.freeze({site,occurrence:visit,instance:key}),captures:Object.freeze([...captures])});
 return identity;
}
