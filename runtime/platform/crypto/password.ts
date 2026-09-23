// Argon2id password hashing over the async Bun.password API, so hashing
// never blocks the event loop synchronously. Three fixed cost presets
// are the whole policy: 0 fast (8 MiB, 1 pass), 1 balanced (64 MiB,
// 2 passes, the native default), 2 secure (256 MiB, 3 passes). Anything
// else is cost_rejected; there are no custom costs to smuggle a weak or
// crushing parameterization through. Verification recognizes the
// recorded parameters from the encoded hash itself, but only inside the
// qualified envelope: v=19, memory 8 KiB..1 GiB, time 1..32, parallelism
// 1..4, 32-byte salt and hash. Anything outside that shape is
// invalid_hash, never a wrong-password false and never a native throw:
// native verify answers "" with false but throws on other malformed
// input, so the adapter gates shape before calling native.
import {success,failure,caught,type Completion,type AssertionContext} from "../../completion.ts";
import {denyLiveBoundary} from "../../assert/context.ts";
import {record} from "../../data.ts";
import {createDomainRuntime} from "../../domain.ts";
const origin=Object.freeze({source:"can:password",start:0,end:0,invocation:Object.freeze([])});
const presets=Object.freeze([
 Object.freeze({memoryCost:8192,timeCost:1,parallelism:1}),
 Object.freeze({memoryCost:65536,timeCost:2,parallelism:1}),
 Object.freeze({memoryCost:262144,timeCost:3,parallelism:1}),
]);
const phc=/^\$argon2id\$v=19\$m=(\d{1,7}),t=(\d{1,2}),p=(\d)\$[A-Za-z0-9+/]{43}\$[A-Za-z0-9+/]{43}$/;
const qualified=(encoded:string):boolean=>{
 const match=phc.exec(encoded);
 if(match===null)return false;
 const memory=Number(match[1]),time=Number(match[2]),parallelism=Number(match[3]);
 return memory>=8&&memory<=1048576&&time>=1&&time<=32&&parallelism>=1&&parallelism<=4;
};
export function createPasswords(domain:ReturnType<typeof createDomainRuntime>,ids:{costRejected:string;invalidHash:string}){
 const cost=(profile:bigint)=>failure(domain.create(ids.costRejected,record(ids.costRejected,[["profile",profile]]),origin));
 const malformed=()=>failure(domain.create(ids.invalidHash,record(ids.invalidHash,[["reason","format"]]),origin));
 return Object.freeze({
  async hash(password:unknown,profile:unknown,context?:AssertionContext):Promise<Completion<string>>{
   denyLiveBoundary(context,origin);
   if(typeof password!=="string")throw new TypeError("invalid compiler password input");
   if(typeof profile!=="bigint")throw new TypeError("invalid compiler cost profile");
   if(profile<0n||profile>2n)return cost(profile);
   try{
    const options=presets[Number(profile)]!;
    // The runtime accepts parallelism (probed 1/2/4) but bun-types omits
    // it from Argon2Algorithm; the assertion pins the call to the declared
    // parameter type without dropping the probed option.
    return success(await Bun.password.hash(password,{algorithm:"argon2id",memoryCost:options.memoryCost,timeCost:options.timeCost,parallelism:options.parallelism} as Parameters<typeof Bun.password.hash>[1]));
   }catch(cause){return caught(cause,origin);}
  },
  async verify(password:unknown,encoded:unknown,context?:AssertionContext):Promise<Completion<boolean>>{
   if(typeof password!=="string"||typeof encoded!=="string")throw new TypeError("invalid compiler password input");
   if(!qualified(encoded))return malformed();
   try{
    return success(await Bun.password.verify(password,encoded));
   }catch(cause){return caught(cause,origin);}
  },
 });
}
