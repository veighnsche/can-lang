// Opaque WebCrypto key handles. Each handle is an empty frozen token; the
// registry below is the only map from handle to key material, so raw keys
// never appear in fixture reports, logs, or generic object projections.
// AES keys stay native-nonextractable (no Can operation exports them).
// Ed25519 pairs mint two single-usage handles: sign-only private and
// verify-only public. The pair generates native-extractable because the
// public half must stay distributable; the private half never leaves
// because no operation accepts a sign-capable handle for export.
import {success,failure,caught,type Completion,type AssertionContext} from "../../completion.ts";
import {denyLiveBoundary} from "../../assert/context.ts";
import {record} from "../../data.ts";
import {ownBytes,copyBytes} from "../../bytes.ts";
import {createDomainRuntime} from "../../domain.ts";
const origin=Object.freeze({source:"can:crypto-keys",start:0,end:0,invocation:Object.freeze([])});
export type KeyAlgorithm="AES-GCM"|"Ed25519";
type KeyState={algorithm:KeyAlgorithm;usages:ReadonlyArray<string>;key:CryptoKey};
const keys=new WeakMap<object,KeyState>();
const object=(value:unknown):value is object=>value!==null&&(typeof value==="object"||typeof value==="function");
export function isCryptoKeyValue(kind:string|undefined,value:unknown):boolean{
 return kind==="key"&&object(value)&&keys.has(value);
}
// projectKey exposes the registry entry to sibling crypto modules so
// every operation checks algorithm and usage before touching native.
export function projectKey(value:unknown):KeyState|undefined{
 if(!object(value))return undefined;
 return keys.get(value);
}
// copyBytes always hands back a fresh ArrayBuffer-backed copy (a slice of
// a fresh allocation), which is exactly what the DOM BufferSource typing
// requires; the assertion records that without copying a second time.
export function bufferView(bytes:Uint8Array):Uint8Array<ArrayBuffer>{
 return bytes as Uint8Array<ArrayBuffer>;
}
const mint=(algorithm:KeyAlgorithm,usages:ReadonlyArray<string>,key:CryptoKey):object=>{
 const handle=Object.freeze(Object.create(null));
 keys.set(handle,{algorithm,usages,key});
 return handle;
};
export function createCryptoKeys(domain:ReturnType<typeof createDomainRuntime>,ids:{keyMisuse:string;invalidKey:string;keypair:string}){
 const misuse=(operation:string,algorithm:string)=>failure(domain.create(ids.keyMisuse,record(ids.keyMisuse,[["operation",operation],["algorithm",algorithm]]),origin));
 const invalid=(reason:string)=>failure(domain.create(ids.invalidKey,record(ids.invalidKey,[["reason",reason]]),origin));
 return Object.freeze({
  async generateAESKey(context?:AssertionContext):Promise<Completion<object>>{
   denyLiveBoundary(context,origin);
   try{
    const key=await crypto.subtle.generateKey({name:"AES-GCM",length:256},false,["encrypt","decrypt"]);
    return success(mint("AES-GCM",["encrypt","decrypt"],key as CryptoKey));
   }catch(cause){return caught(cause,origin);}
  },
  async generateEd25519Keypair(context?:AssertionContext):Promise<Completion<unknown>>{
   denyLiveBoundary(context,origin);
   try{
    const pair=await crypto.subtle.generateKey({name:"Ed25519"},true,["sign","verify"]) as CryptoKeyPair;
    return success(record(ids.keypair,[["private_key",mint("Ed25519",["sign"],pair.privateKey)],["public_key",mint("Ed25519",["verify"],pair.publicKey)]]));
   }catch(cause){return caught(cause,origin);}
  },
  async importEd25519Public(input:unknown,context?:AssertionContext):Promise<Completion<object>>{
   const bytes=copyBytes(input,origin);
   if(bytes.length!==32)return invalid("bad_public_length");
   try{
    const key=await crypto.subtle.importKey("raw",bufferView(bytes),{name:"Ed25519"},true,["verify"]);
    return success(mint("Ed25519",["verify"],key));
   }catch(cause){return caught(cause,origin);}
  },
  async exportEd25519Public(handle:unknown,context?:AssertionContext):Promise<Completion<unknown>>{
   const projected=projectKey(handle);
   if(projected===undefined)throw new TypeError("invalid compiler crypto key");
   // Sign-capable handles never export, even though the native pair
   // generates extractable: distribution is a verify-key privilege.
   if(projected.algorithm!=="Ed25519"||!projected.usages.includes("verify")||projected.usages.includes("sign"))return misuse("export",projected.algorithm);
   try{
    return success(ownBytes(new Uint8Array(await crypto.subtle.exportKey("raw",projected.key))));
   }catch(cause){return caught(cause,origin);}
  },
 });
}
