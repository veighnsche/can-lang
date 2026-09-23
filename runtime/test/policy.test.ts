import {test,expect} from "bun:test";
import {createHash} from "node:crypto";
import {catalogue} from "../catalogue.ts";
import {createDomainRuntime,domainFailureDiagnostics,type FailureShape} from "../domain.ts";
import {success,failure} from "../completion.ts";
import {record,array} from "../data.ts";
import {assertionContext,contextReport,closeContext} from "../assert/context.ts";
import {provideInjection,policyBase,policyKey,policyLeaf} from "../assert/policy.ts";
const hash=(key:unknown)=>createHash("sha256").update("can-concrete-type-v1\0"+JSON.stringify(key)).digest("hex");
const shape=(kind:string,declaration:string,fields:{name:string;type:string}[]=[]):FailureShape=>({identity:hash([kind,declaration]),kind,declaration,fields,arguments:[],leaves:[],inputs:[],errors:[]});
const str=shape("primitive","str"),int=shape("primitive","int");
const header=shape("record","can.std.http@1::header",[{name:"name",type:str.identity},{name:"value",type:str.identity}]);
const headers:FailureShape={...shape("array",""),identity:hash(["array","",header.identity]),element:header.identity};
const declarations=catalogue.errors.filter(e=>e.id>=1100&&e.id<=1106||e.id===1110||e.id===1121);
const detailIdentity=hash(["variant","can.std.http@1::failure_detail"]);
const errors=declarations.map(d=>shape("error",d.identity,d.fields.map(f=>({name:f.name,type:f.type==="str"?str.identity:f.type==="int"?int.identity:f.type==="http::failure_detail"?detailIdentity:headers.identity}))));
const detail:FailureShape={identity:detailIdentity,kind:"variant",declaration:"can.std.http@1::failure_detail",fields:[],arguments:[],leaves:errors.filter((_,i)=>declarations[i].id!==1106&&declarations[i].id<1120).map(e=>e.identity),inputs:[],errors:[]};
const domain=createDomainRuntime({declarations:declarations.map(d=>({...d,parameters:0})),shapes:[str,int,header,headers,...errors,detail]});
const ids={invalid:errors[0].identity,credential:errors[1].identity,transport:errors[2].identity,timeout:errors[3].identity,limit:errors[4].identity,status:errors[5].identity,failed:errors[6].identity,invalidData:errors[7].identity,invalidAnswer:errors[8].identity};
const origin={source:"test:policy",start:0,end:0,invocation:[]};
const operation="test:policy/cached";
const statusPayload=()=>record(ids.status,[["status",404n],["headers",array([])]]);
const answerPayload=()=>record(ids.invalidAnswer,[["question","q2"],["reason","out of range"]]);
const codecPayload=()=>record(ids.invalidData,[["path","body"],["reason","type"]]);

test("native injection skips the operation and selects by leaf identity",async()=>{
 const context=assertionContext({package:"app",declaration:"app::cached",name:"absent"});
 provideInjection(context,operation,"native",ids.status,statusPayload(),ids.failed,domain,origin);
 const invoked=await policyBase(context,operation,origin,async()=>{throw new Error("wrapped operation ran");});
 expect(invoked.kind).toBe("domain");
 if(invoked.kind!=="domain")throw new Error("expected domain");
 expect(domainFailureDiagnostics(invoked.value).typeIdentity).toBe(ids.failed);
 expect(policyKey(invoked,ids.failed)).toEqual({origin:"native",identity:ids.status});
 expect(policyLeaf(invoked)).toEqual(statusPayload());
 closeContext(context);
 const report=contextReport(context);
 expect(report.violations).toEqual([]);
 expect(report.evidence).toContain("policy-fixture");
 expect(report.evidence).not.toContain("raw-provider-fixture");
});

test("emitted injection is not normalized and keeps its own identity",async()=>{
 const context=assertionContext({package:"app",declaration:"app::guarded",name:"absorbed"});
 provideInjection(context,operation,"emitted",ids.invalidAnswer,answerPayload(),ids.failed,domain,origin);
 const invoked=await policyBase(context,operation,origin,async()=>{throw new Error("wrapped operation ran");});
 expect(invoked.kind).toBe("domain");
 if(invoked.kind!=="domain")throw new Error("expected domain");
 expect(domainFailureDiagnostics(invoked.value).typeIdentity).toBe(ids.invalidAnswer);
 expect(policyKey(invoked,ids.failed)).toEqual({origin:"emitted",identity:ids.invalidAnswer});
 closeContext(context);
 expect(contextReport(context).violations).toEqual([]);
});

test("same-name codec keys stay distinct across origins",async()=>{
 for(const [name,originTag] of [["native","native"],["emitted","emitted"]] as const){
  const context=assertionContext({package:"app",declaration:"app::guarded",name});
  provideInjection(context,operation,originTag,ids.invalidData,codecPayload(),ids.failed,domain,origin);
  const invoked=await policyBase(context,operation,origin,async()=>{throw new Error("wrapped operation ran");});
  expect(policyKey(invoked,ids.failed)).toEqual({origin:originTag,identity:ids.invalidData});
  closeContext(context);
  expect(contextReport(context).violations).toEqual([]);
 }
});

test("injection rejects unknown origins and double provision",()=>{
 const context=assertionContext({package:"app",declaration:"app::cached",name:"bad"});
 expect(()=>provideInjection(context,operation,"bogus",ids.status,statusPayload(),ids.failed,domain,origin)).toThrow();
 provideInjection(context,operation,"native",ids.status,statusPayload(),ids.failed,domain,origin);
 expect(()=>provideInjection(context,operation,"native",ids.status,statusPayload(),ids.failed,domain,origin)).toThrow();
 closeContext(context);
 const report=contextReport(context);
 expect(report.violations).toContain("malformed fixture");
 expect(report.violations).toContain("unused fixture");
});

test("policyBase invokes the operation without context or injection",async()=>{
 const direct=await policyBase(undefined,operation,origin,async()=>success(1n));
 expect(direct).toEqual(success(1n));
 const context=assertionContext({package:"app",declaration:"app::cached",name:"live"});
 const live=await policyBase(context,operation,origin,async()=>failure(domain.create(ids.failed,record(ids.failed,[["detail",statusPayload()]]),origin)));
 expect(live.kind).toBe("domain");
 closeContext(context);
 const report=contextReport(context);
 expect(report.violations).toEqual([]);
 expect(report.evidence).not.toContain("policy-fixture");
});
