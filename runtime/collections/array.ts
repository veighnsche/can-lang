import {invoke,success,failure,caught,isCompletion,value,type Completion} from "../completion.ts";
import {array,record} from "../data.ts";
import {nonfiniteSortKeyFailure,type FailureOrigin} from "../failure.ts";
import {callContext,type AssertionContext} from "../assert/context.ts";
import {callableInstance} from "../callable.ts";

export type Trace=Readonly<{site:string;origin:FailureOrigin;context?:AssertionContext}>;
export type Callback<A extends unknown[],R>=(...arguments_:[...A,AssertionContext?])=>Completion<R>|Promise<Completion<R>>;
export type SortKey=bigint|number|string|boolean;
// Callback payloads are never returned naked from an async function. The parent
// collection frame remains active between visits and while native algorithms
// settle, so fixture scheduling cannot mistake those gaps for quiescence.
function callback<A extends unknown[],R>(action:Callback<A,R>,arguments_:A,trace:Trace):Promise<Completion<R>>{
 return callContext(trace.context,trace.site,context=>invoke(()=>action(...arguments_,context),trace.origin),callableInstance(action));
}
async function boundary<T>(run:()=>Promise<Completion<T>>,trace:Trace):Promise<Completion<T>>{
 try{return await run()}catch(cause){
  if(isCompletion(cause)&&cause.kind!=="ok")return cause;
  return caught(cause,trace.origin);
 }
}
async function observations<T,U>(source:readonly T[],action:Callback<[T],U>,trace:Trace):Promise<Completion<U>[]> {
 // Iterating elements would let Array.fromAsync await a data-valued `then`.
 // Indices are native numbers; the callback receives its element synchronously.
 return Array.fromAsync(source.keys(),async index=>{
  const result=await callback(action,[source[index]],trace);
  if(result.kind!=="ok")throw result;
  return result;
 });
}
export function map<T,U>(source:readonly T[],action:Callback<[T],U>,trace:Trace):Promise<Completion<readonly U[]>>{
 return boundary(async()=>success(array((await observations(source,action,trace)).map(value))),trace);
}
export function filter<T>(source:readonly T[],action:Callback<[T],boolean>,trace:Trace):Promise<Completion<readonly T[]>>{
 return boundary(async()=>{
  const decisions=(await observations(source,action,trace)).map(value);
  return success(array(source.filter((_,index)=>decisions[index])));
 },trace);
}
export function fold<T,U>(source:readonly T[],initial:U,action:Callback<[U,T],U>,trace:Trace):Promise<Completion<U>>{
 return boundary(()=>source.reduce<Promise<Completion<U>>>((prior,item)=>prior.then(async accumulator=>{
  const result=await callback(action,[value(accumulator),item],trace);
  if(result.kind!=="ok")throw result;
  return result;
 }),Promise.resolve(success(initial))),trace);
}
export function forEach<T>(source:readonly T[],action:Callback<[T],void>,trace:Trace):Promise<Completion<void>>{
 return boundary(()=>source.reduce<Promise<Completion<void>>>((prior,item)=>prior.then(async()=>{
  const result=await callback(action,[item],trace);
  if(result.kind!=="ok")throw result;
  return result;
 }),Promise.resolve(success(undefined))),trace);
}
async function search<T>(source:readonly T[],action:Callback<[T],boolean>,stop:boolean,trace:Trace):Promise<Completion<number>>{
 // A bounded await adapter supplies the short-circuit behavior native sync
 // predicates cannot provide. It neither sorts nor filters the source itself.
 let index=0;
 for(const item of source.values()){
  const result=await callback(action,[item],trace);
  if(result.kind!=="ok")return result;
  if(result.value===stop)return success(index);
  index++;
 }
 return success(-1);
}
export function some<T>(source:readonly T[],action:Callback<[T],boolean>,trace:Trace):Promise<Completion<boolean>>{
 return boundary(async()=>{const result=await search(source,action,true,trace);return result.kind==="ok"?success(result.value!==-1):result},trace);
}
export function every<T>(source:readonly T[],action:Callback<[T],boolean>,trace:Trace):Promise<Completion<boolean>>{
 return boundary(async()=>{const result=await search(source,action,false,trace);return result.kind==="ok"?success(result.value===-1):result},trace);
}
export function find<T>(source:readonly T[],action:Callback<[T],boolean>,identities:Readonly<{none:string;some:string}>,trace:Trace):Promise<Completion>{
 return boundary(async()=>{
  const result=await search(source,action,true,trace);
  if(result.kind!=="ok")return result;
  return success(result.value===-1?record(identities.none,[]):record(identities.some,[["value",source[result.value]]]));
 },trace);
}
export function sortBy<T,K extends SortKey>(source:readonly T[],action:Callback<[T],K>,trace:Trace):Promise<Completion<readonly T[]>>{
 return boundary(async()=>{
  const keys=await Array.fromAsync(source.keys(),async index=>{
   const result=await callback(action,[source[index]],trace);
   if(result.kind!=="ok")throw result;
   if(typeof result.value==="number"&&!Number.isFinite(result.value))throw failure(nonfiniteSortKeyFailure(undefined,trace.origin));
   // Only the sealed compiler-admitted int/float/str/bool key type reaches here.
   return success(Object.freeze({key:result.value,index}));
  });
  const decorated=keys.map(value);
  const sorted=decorated.toSorted((a,b)=>a.key<b.key?-1:a.key>b.key?1:a.index-b.index);
  return success(array(sorted.map(entry=>source[entry.index])));
 },trace);
}

export function concat<T>(left:readonly T[],right:readonly T[]):readonly T[]{return array(left.concat(right));}
export function toReversed<T>(source:readonly T[]):readonly T[]{return array(source.toReversed());}
export function append<T>(source:readonly T[],item:T):readonly T[]{return array([...source,item]);}
