import {failure,type Completion} from "../completion.ts";
import {record} from "../data.ts";
import {createDomainRuntime,domainFailureDiagnostics} from "../domain.ts";

// Boundary normalization maps a native infrastructure failure produced for
// the calling operation into a fresh http::request_failed occurrence whose
// Detail is the original typed leaf and whose private cause is the original
// occurrence. Anything else — emitted failures, other operations, standard
// outcomes, successes — passes through untouched. Non-leaf natives pass
// through rather than corrupting the finite Detail contract.
export type NormalizeTypes=Readonly<{failed:string;leaves:readonly string[]}>;
export function createNormalizer(domain:ReturnType<typeof createDomainRuntime>,types:NormalizeTypes){
 const leaves=new Set(types.leaves);
 return Object.freeze({map<T>(outcome:Completion<T>,operation:string):Completion<T>{
  if(outcome.kind!=="domain")return outcome;
  const original=domainFailureDiagnostics(outcome.value);
  if(original.provenance.boundary!=="native"||original.provenance.operation!==operation)return outcome;
  if(!leaves.has(original.typeIdentity))return outcome;
  return failure(domain.create(types.failed,record(types.failed,[["detail",original.payload]]),original.origin,outcome.value,{boundary:"native",operation}));
 }});
}
