// Symmetric crypto, signatures, and hashes. HMAC takes plain byte keys
// (no handle indirection for a pure function); AES-GCM and Ed25519 take
// opaque key handles whose algorithm and usage are checked before any
// native call, so an encryption-only key can never sign and a signing
// key can never decrypt. AES-GCM fixes a 12-byte nonce and a 128-bit
// tag: native accepts other nonce lengths, so the adapter enforces 12.
// Decrypt failure (tampered ciphertext, key, nonce, or associated data)
// collapses to one fieldless decrypt_failed: distinguishing the cause
// would build an oracle. The sealed convenience mints its nonce from
// native randomness and returns it alongside the ciphertext; generation
// is randomness, not a guarantee of caller nonce discipline.
import {success,failure,caught,type Completion,type AssertionContext} from "../../completion.ts";
import {denyLiveBoundary} from "../../assert/context.ts";
import {record} from "../../data.ts";
import {ownBytes,copyBytes} from "../../bytes.ts";
import {createDomainRuntime} from "../../domain.ts";
import {projectKey,bufferView,type KeyAlgorithm} from "./keys.ts";
const origin=Object.freeze({source:"can:crypto",start:0,end:0,invocation:Object.freeze([])});
export async function sha256(input:unknown,_context?:AssertionContext):Promise<Completion<unknown>>{
 const bytes=copyBytes(input,origin);
 return success(ownBytes(new Bun.CryptoHasher("sha256").update(bytes).digest()));
}
type KeyUse={key:CryptoKey}|{reject:string};
const use=(handle:unknown,algorithm:KeyAlgorithm,usage:string):KeyUse=>{
 const projected=projectKey(handle);
 if(projected===undefined)throw new TypeError("invalid compiler crypto key");
 if(projected.algorithm!==algorithm||!projected.usages.includes(usage))return{reject:projected.algorithm};
 return{key:projected.key};
};
export function createCryptoPrimitives(domain:ReturnType<typeof createDomainRuntime>,ids:{keyMisuse:string;invalidKey:string;invalidNonce:string;decryptFailed:string;sealed:string}){
 const misuse=(operation:string,algorithm:string)=>failure(domain.create(ids.keyMisuse,record(ids.keyMisuse,[["operation",operation],["algorithm",algorithm]]),origin));
 const badNonce=(length:bigint)=>failure(domain.create(ids.invalidNonce,record(ids.invalidNonce,[["length",length]]),origin));
 const corrupt=()=>failure(domain.create(ids.decryptFailed,record(ids.decryptFailed,[]),origin));
 return Object.freeze({
  async hmacSha256(key:unknown,message:unknown,context?:AssertionContext):Promise<Completion<unknown>>{
   const keyBytes=copyBytes(key,origin),messageBytes=copyBytes(message,origin);
   if(keyBytes.length===0)return failure(domain.create(ids.invalidKey,record(ids.invalidKey,[["reason","empty_hmac_key"]]),origin));
   try{
    const native=await crypto.subtle.importKey("raw",bufferView(keyBytes),{name:"HMAC",hash:"SHA-256"},false,["sign"]);
    return success(ownBytes(new Uint8Array(await crypto.subtle.sign("HMAC",native,bufferView(messageBytes)))));
   }catch(cause){return caught(cause,origin);}
  },
  async encryptAesGcm(handle:unknown,nonce:unknown,plaintext:unknown,associated:unknown,context?:AssertionContext):Promise<Completion<unknown>>{
   const got=use(handle,"AES-GCM","encrypt");
   if("reject" in got)return misuse("encrypt",got.reject);
   const nonceBytes=copyBytes(nonce,origin),plainBytes=copyBytes(plaintext,origin),aadBytes=copyBytes(associated,origin);
   if(nonceBytes.length!==12)return badNonce(BigInt(nonceBytes.length));
   try{
    const cipher=await crypto.subtle.encrypt({name:"AES-GCM",iv:bufferView(nonceBytes),additionalData:bufferView(aadBytes),tagLength:128},got.key,bufferView(plainBytes));
    return success(ownBytes(new Uint8Array(cipher)));
   }catch(cause){return caught(cause,origin);}
  },
  async encryptAesGcmSealed(handle:unknown,plaintext:unknown,associated:unknown,context?:AssertionContext):Promise<Completion<unknown>>{
   denyLiveBoundary(context,origin);
   const got=use(handle,"AES-GCM","encrypt");
   if("reject" in got)return misuse("encrypt",got.reject);
   const plainBytes=copyBytes(plaintext,origin),aadBytes=copyBytes(associated,origin);
   const nonceBytes=crypto.getRandomValues(new Uint8Array(12));
   try{
    const cipher=await crypto.subtle.encrypt({name:"AES-GCM",iv:nonceBytes,additionalData:bufferView(aadBytes),tagLength:128},got.key,bufferView(plainBytes));
    return success(record(ids.sealed,[["nonce",ownBytes(nonceBytes)],["ciphertext",ownBytes(new Uint8Array(cipher))]]));
   }catch(cause){return caught(cause,origin);}
  },
  async decryptAesGcm(handle:unknown,nonce:unknown,ciphertext:unknown,associated:unknown,context?:AssertionContext):Promise<Completion<unknown>>{
   const got=use(handle,"AES-GCM","decrypt");
   if("reject" in got)return misuse("decrypt",got.reject);
   const nonceBytes=copyBytes(nonce,origin),cipherBytes=copyBytes(ciphertext,origin),aadBytes=copyBytes(associated,origin);
   if(nonceBytes.length!==12)return badNonce(BigInt(nonceBytes.length));
   try{
    const plain=await crypto.subtle.decrypt({name:"AES-GCM",iv:bufferView(nonceBytes),additionalData:bufferView(aadBytes),tagLength:128},got.key,bufferView(cipherBytes));
    return success(ownBytes(new Uint8Array(plain)));
   }catch(cause){
    if(cause instanceof Error&&cause.name==="OperationError")return corrupt();
    return caught(cause,origin);
   }
  },
  async signEd25519(handle:unknown,message:unknown,context?:AssertionContext):Promise<Completion<unknown>>{
   const got=use(handle,"Ed25519","sign");
   if("reject" in got)return misuse("sign",got.reject);
   const messageBytes=copyBytes(message,origin);
   try{
    return success(ownBytes(new Uint8Array(await crypto.subtle.sign("Ed25519",got.key,bufferView(messageBytes)))));
   }catch(cause){return caught(cause,origin);}
  },
  async verifyEd25519(handle:unknown,message:unknown,signature:unknown,context?:AssertionContext):Promise<Completion<boolean>>{
   const got=use(handle,"Ed25519","verify");
   if("reject" in got)return misuse("verify",got.reject);
   const messageBytes=copyBytes(message,origin),signatureBytes=copyBytes(signature,origin);
   try{
    return success(await crypto.subtle.verify("Ed25519",got.key,bufferView(signatureBytes),bufferView(messageBytes)));
   }catch(cause){return caught(cause,origin);}
  },
 });
}
