import {test,expect} from "bun:test";
import {GenMapping,addMapping,toEncodedMap} from "./vendor/source-maps.mjs";
import {validateMaps} from "./source-map-validation.ts";
const span={start:8,end:15,line:3,column:5,endLine:3,endColumn:12,operation:"call"};
const index={schemaVersion:1,kind:"can.source-index",sources:[{id:"can.project.root/app/main.can",path:"main.can",spans:{"call:8:15":span}}],modules:[{path:"packages/test.ts",segments:[{line:2,column:3,source:"can.project.root/app/main.can",name:"call:8:15"}]}]};
function fixture(){const map=new GenMapping({file:"test.ts"});addMapping(map,{generated:{line:2,column:3},source:index.sources[0].id,original:{line:3,column:5},name:"call:8:15"});return {"packages/test.ts":toEncodedMap(map)};}
test("upstream source-map validation refuses missing, corrupt and unknown-source maps",()=>{
 expect(()=>validateMaps(index,fixture())).not.toThrow();
 expect(()=>validateMaps(index,{})).toThrow();
 for (const mutate of [
  (m:any)=>{m.mappings="?"}, (m:any)=>{m.mappings+=";"},
  (m:any)=>{m.sources[0]="/private/machine/secret.can"},
  (m:any)=>{m.sourcesContent[0]="bearer-secret"},
  (m:any)=>{m.names[0]="unknown:1:2"},
  (m:any)=>{m.file="other.ts"}, (m:any)=>{m.sourceRoot="/private"},
 ]){const maps=fixture();mutate(maps["packages/test.ts"]);expect(()=>validateMaps(index,maps)).toThrow();}
 const bad=structuredClone(index);bad.modules[0].segments[0].source="unknown";
 expect(()=>validateMaps(bad,fixture())).toThrow();
 const badOrder=structuredClone(index);badOrder.modules[0].segments.push(badOrder.modules[0].segments[0]);
 expect(()=>validateMaps(badOrder,fixture())).toThrow();
});
