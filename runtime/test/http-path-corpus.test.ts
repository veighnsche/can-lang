import {test,expect} from "bun:test";
import {createHash} from "node:crypto";
import {catalogue} from "../catalogue.ts";
import {createDomainRuntime,type FailureShape} from "../domain.ts";
import {createResponses} from "../platform/http.ts";
import {createRouter} from "../platform/router.ts";
import {success} from "../completion.ts";
const hash=(kind:string,name:string)=>createHash("sha256").update("can-concrete-type-v1\0"+JSON.stringify([kind,name])).digest("hex");
const shape=(kind:string,declaration:string,fields:{name:string;type:string}[]=[]):FailureShape=>({identity:hash(kind,declaration),kind,declaration,fields,arguments:[],leaves:[],inputs:[],errors:[]});
const str=shape("primitive","str"),int=shape("primitive","int");
const declarations=catalogue.errors.filter(e=>[1100,1230,1231,1232].includes(e.id));
const errors=declarations.map(e=>shape("error",e.identity,e.fields.map(f=>({name:f.name,type:f.type==="int"?int.identity:str.identity}))));
const domain=createDomainRuntime({declarations:declarations.map(e=>({...e,parameters:0})),shapes:[str,int,...errors]});
const responses=createResponses(domain,{invalid:errors[0]!.identity,invalidData:errors[0]!.identity,close:"unused"});
const routing=createRouter(domain,{invalid:errors[1]!.identity,duplicate:errors[2]!.identity,ambiguous:errors[3]!.identity});
test("route path corpus matches checked static validation",async()=>{
 const corpus=await Bun.file(new URL("./http-paths.json",import.meta.url)).json() as {accept:string[];reject:string[]};
 const status=await responses.ok(),headers=await responses.emptyHeaders();
 if(status.kind!=="ok"||headers.kind!=="ok")throw Error("setup");
 const callback=async()=>responses.text(status.value,headers.value,"ok");
 expect(corpus.accept.length).toBeGreaterThan(0);expect(corpus.reject.length).toBeGreaterThan(0);
 for(const path of corpus.accept)expect((await routing.get(path,callback)).kind).toBe("ok");
 for(const path of corpus.reject){const made=await routing.get(path,callback);expect(made.kind).not.toBe("ok");}
 expect((await routing.get("/\ud800",callback)).kind).not.toBe("ok");
});
