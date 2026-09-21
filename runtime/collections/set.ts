import {success,type AssertionContext} from "../completion.ts";
import {resourceStateFailure} from "../failure.ts";
export type Key = bigint | boolean | string;
export type KeyKind = "int" | "bool" | "str";
declare const brand: unique symbol;
export type ImmutableSet<K extends Key = Key> = Readonly<{readonly [brand]: K}>;
const storage = new WeakMap<object,{identity:string;values:Set<Key>}>();
const origin=Object.freeze({source:"can:collections:set",start:0,end:0,invocation:Object.freeze([] as string[])});
export function isSet(identity:string,value:unknown):value is object {
 return value!==null&&typeof value==="object"&&storage.get(value)?.identity===identity;
}
export function checkKey(kind:KeyKind,key:unknown):asserts key is Key {
 if(typeof key!==({int:"bigint",bool:"boolean",str:"string"} as const)[kind])throw resourceStateFailure(undefined,origin);
}
export function createSet<K extends Key>(identity:string,keyKind:KeyKind){
 function own(values:Set<Key>):ImmutableSet<K>{const token=Object.freeze(Object.create(null));storage.set(token,{identity,values});return token;}
 function backing(value:unknown):Set<Key>{if(!isSet(identity,value))throw resourceStateFailure(undefined,origin);return storage.get(value)!.values;}
 return Object.freeze({
  async empty(_context?:AssertionContext){return success(own(new Set()));},
  async contains(set:unknown,key:K,_context?:AssertionContext){const source=backing(set);checkKey(keyKind,key);return success(source.has(key));},
  async add(set:unknown,key:K,_context?:AssertionContext){const source=backing(set);checkKey(keyKind,key);return success(own(new Set(source).add(key)));},
  async union(left:unknown,right:unknown,_context?:AssertionContext){return success(own(backing(left).union(backing(right))));},
  async intersection(left:unknown,right:unknown,_context?:AssertionContext){
   const a=backing(left),b=backing(right);
   return success(own(new Set(Array.from(a).filter(key=>b.has(key)))));
  },
  async difference(left:unknown,right:unknown,_context?:AssertionContext){return success(own(backing(left).difference(backing(right))));},
 });
}
