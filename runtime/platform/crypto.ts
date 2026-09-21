import {success,type Completion,type AssertionContext} from "../completion.ts";
import {copyBytes,ownBytes,type Bytes} from "../bytes.ts";
const origin=Object.freeze({source:"can:crypto",start:0,end:0,invocation:Object.freeze([])});
export async function sha256(input:unknown,_context?:AssertionContext):Promise<Completion<Bytes>>{
 const bytes=copyBytes(input,origin);
 return success(ownBytes(new Bun.CryptoHasher("sha256").update(bytes).digest()));
}
