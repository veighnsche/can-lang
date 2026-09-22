import {test,expect} from "bun:test";
import {createHash} from "node:crypto";
import {catalogue} from "../catalogue.ts";
import {createDomainRuntime,domainFailureDiagnostics,isDomainFailure,type FailureShape} from "../domain.ts";
import {createNormalizer} from "../transport/normalize.ts";
import {success,failure,caught,type Completion} from "../completion.ts";
import {record,recordIdentity,dataProperty,array} from "../data.ts";
const hash=(key:unknown)=>createHash("sha256").update("can-concrete-type-v1\0"+JSON.stringify(key)).digest("hex");
const shape=(kind:string,declaration:string,fields:{name:string;type:string}[]=[]):FailureShape=>({identity:hash([kind,declaration]),kind,declaration,fields,arguments:[],leaves:[],inputs:[],errors:[]});
const str=shape("primitive","str"),int=shape("primitive","int");
const header=shape("record","can.std.http@1::header",[{name:"name",type:str.identity},{name:"value",type:str.identity}]);
const headers:FailureShape={...shape("array",""),identity:hash(["array","",header.identity]),element:header.identity};
const declarations=catalogue.errors.filter(e=>e.id>=1100&&e.id<=1106||e.id===1110);
const detailIdentity=hash(["variant","can.std.http@1::failure_detail"]);
const errors=declarations.map(d=>shape("error",d.identity,d.fields.map(f=>({name:f.name,type:f.type==="str"?str.identity:f.type==="int"?int.identity:f.type==="http::failure_detail"?detailIdentity:headers.identity}))));
const detail:FailureShape={identity:detailIdentity,kind:"variant",declaration:"can.std.http@1::failure_detail",fields:[],arguments:[],leaves:errors.filter((_,i)=>declarations[i].id!==1106).map(e=>e.identity),inputs:[],errors:[]};
const domain=createDomainRuntime({declarations:declarations.map(d=>({...d,parameters:0})),shapes:[str,int,header,headers,...errors,detail]});
const byId=(id:number)=>errors[declarations.findIndex(d=>d.id===id)].identity;
const origin={source:"test:normalize",start:0,end:0,invocation:[]};
const operation="test:normalize/fetch";
const normalize=createNormalizer(domain,{failed:byId(1106),leaves:[1100,1101,1102,1103,1104,1105,1110].map(byId)});
const leafPayloads:ReadonlyArray<readonly [number,ReadonlyArray<readonly [string,unknown]>]>= [
 [1100,[["reason","url"]]],
 [1101,[["variable","TOKEN"]]],
 [1102,[["phase","connect"]]],
 [1103,[["timeout_ms",5n]]],
 [1104,[["limit",3n]]],
 [1105,[["status",418n],["headers",array([record(header.identity,[["name","content-type"],["value","text/plain"]])])]]],
 [1110,[["path","/0"],["reason","invalid_json"]]],
];

test("all seven native leaves map to request_failed with exact Detail and private cause",()=>{
 for(const [id,fields] of leafPayloads){
  const token=domain.create(byId(id),record(byId(id),fields),origin,undefined,{boundary:"native",operation});
  const mapped=normalize.map(failure(token),operation);
  expect(mapped.kind).toBe("domain");if(mapped.kind!=="domain")throw Error("expected domain failure");
  const details=domainFailureDiagnostics(mapped.value);
  expect(details.declaration.id).toBe(1106);
  expect(details.provenance).toEqual({boundary:"native",operation});
  expect(details.occurrenceID).not.toBe(domainFailureDiagnostics(token).occurrenceID);
  const leaf=dataProperty(details.payload,"detail");
  expect(recordIdentity(leaf)).toBe(byId(id));
  const stable=(v:unknown)=>JSON.stringify(v,(_,x)=>typeof x==="bigint"?`#${x}`:x);
  expect(stable(leaf)).toBe(stable(domainFailureDiagnostics(token).payload));
  expect(leaf).toBe(domainFailureDiagnostics(token).payload);
  if(!isDomainFailure(details.cause))throw Error("expected private original cause");
  expect(details.cause).toBe(token);
 }
});

test("emitted, foreign-operation, non-leaf, standard and success outcomes pass through",()=>{
 const emitted=domain.create(byId(1110),record(byId(1110),[["path",""],["reason","authored"]]),origin);
 expect(normalize.map(failure(emitted),operation)).toMatchObject({kind:"domain"});
 const foreign=domain.create(byId(1102),record(byId(1102),[["phase","connect"]]),origin,undefined,{boundary:"native",operation:"test:normalize/other"});
 const foreignMapped=normalize.map(failure(foreign),operation);
 expect(foreignMapped.kind).toBe("domain");if(foreignMapped.kind!=="domain")throw Error("expected domain");
 expect(domainFailureDiagnostics((foreignMapped as {value:unknown}).value).declaration.id).toBe(1102);
 const narrow=createNormalizer(domain,{failed:byId(1106),leaves:[byId(1100)]});
 const nonLeaf=domain.create(byId(1102),record(byId(1102),[["phase","connect"]]),origin,undefined,{boundary:"native",operation});
 const nonLeafMapped=narrow.map(failure(nonLeaf),operation);
 expect(nonLeafMapped.kind).toBe("domain");if(nonLeafMapped.kind!=="domain")throw Error("expected domain");
 expect(domainFailureDiagnostics((nonLeafMapped as {value:unknown}).value).declaration.id).toBe(1102);
 const ok=success(1n);
 expect(normalize.map(ok,operation)).toBe(ok);
 const standard=caught(Error("boom"),origin);
 expect(normalize.map(standard,operation)).toBe(standard);
});
