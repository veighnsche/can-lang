import {test,expect} from "bun:test";
import {createHash} from "node:crypto";
import {catalogue} from "../catalogue.ts";
import {createDomainRuntime,domainFailureDiagnostics,type FailureShape} from "../domain.ts";
import {createHTML,renderSafe,isHTMLValue} from "../platform/html.ts";
import {value,invoke,type Completion} from "../completion.ts";
const hash=(kind:string,name:string)=>createHash("sha256").update("can-concrete-type-v1\0"+JSON.stringify([kind,name])).digest("hex");
const shape=(kind:string,declaration:string,fields:{name:string;type:string}[]=[]):FailureShape=>({identity:hash(kind,declaration),kind,declaration,fields,arguments:[],leaves:[],inputs:[],errors:[]});
const str=shape("primitive","str"),int=shape("primitive","int");
const declarations=catalogue.errors.filter(e=>[1220,1221,1222,1223].includes(e.id));
const errors=declarations.map(e=>shape("error",e.identity,e.fields.map(f=>({name:f.name,type:f.type==="int"?int.identity:str.identity}))));
const domain=createDomainRuntime({declarations:declarations.map(e=>({...e,parameters:0})),shapes:[str,int,...errors]});
const html=createHTML(domain,{structure:errors[0]!.identity,url:errors[1]!.identity,target:errors[2]!.identity,interval:errors[3]!.identity});
const origin={source:"test",start:0,end:0,invocation:[]};
function check(result:Completion,id:number){expect(result.kind).toBe("domain");if(result.kind!=="domain")throw Error();expect(domainFailureDiagnostics(result.value).declaration.id).toBe(id);}
const tag=async(name:string)=>value(await html.makeTag(name));
const text=async(s:string)=>value(await html.text(s));
const element=async(name:string,kids:unknown[]=[],attrs:unknown[]=[])=>value(await html.element(await tag(name),attrs,kids));
const render=async(nodes:unknown[])=>renderSafe(value(await html.fragment(nodes)));
test("native escaping keeps hostile text and attribute data inert",async()=>{
 const hostile='<script>alert("x")</script>&\' onload="bad"';
 const node=await element("div",[await text(hostile)],[value(await html.textAttribute("TITLE",hostile))]);
 expect(await render([node])).toBe(`<div title="${Bun.escapeHTML(hostile)}">${Bun.escapeHTML(hostile)}</div>`);
 expect(renderSafe(value(await html.textFragment(hostile)))).toBe(Bun.escapeHTML(hostile));
 expect(Object.isFrozen(node)).toBe(true);expect(Reflect.ownKeys(node as object)).toEqual([]);
 for(const input of ["", "a\0b", "\ud800", "&amp;"])expect(await render([await text(input)])).toBe(Bun.escapeHTML(input));
});
test("closed tags, attributes, enums and tag applicability",async()=>{
 for(const name of ["script","style","iframe","object","embed","meta","body","title","svg","x-tag"])check(await html.makeTag(name),1220);
 for(const name of ["onclick","ONLOAD","style","srcdoc","href","src","action","formaction","hx-get","data-hx-get","hx-on:click","data-test"])check(await html.textAttribute(name,"value"),1220);
 for(const [name,val] of [["dir","sideways"],["hidden","false"],["checked","true"],["method","put"],["scope","all"],["align","justify"],["start","NaN"],["autocomplete","unknown"],["rel","javascript"],["rows","0"],["colspan","1001"],["rowspan","65535"]])check(await html.textAttribute(name!,val!),1220);
 for(const [name,val] of [["dir","rtl"],["hidden","until-found"],["required",""],["autocomplete","section-login username webauthn"],["rel","nofollow noopener"],["rowspan","0"],["aria-label","anything"]])expect((await html.textAttribute(name!,val!)).kind).toBe("ok");
 const checked=value(await html.textAttribute("checked","checked"));check(await html.element(await tag("div"),[checked],[]),1220);
 check(await html.element(await tag("button"),[value(await html.textAttribute("type","password"))],[]),1220);
 expect((await html.element(await tag("input"),[value(await html.textAttribute("type","password"))],[])).kind).toBe("ok");
 check(await html.element(await tag("div"),[value(await html.textAttribute("id","a")),value(await html.textAttribute("ID","b"))],[]),1220);
});
test("child categories, table order, pairs and descendant nesting",async()=>{
 for(const name of ["input","br","hr","img"])check(await html.element(await tag(name),[],[await text("x")]),1220);
 for(const [parent,child] of [["ul","li"],["ol","li"],["select","option"],["thead","tr"],["tbody","tr"],["tr","td"]]){
  const node=await element(child!);expect((await html.element(await tag(parent!),[],[node])).kind).toBe("ok");check(await html.element(await tag(parent!),[],[await text(" ")]),1220);
 }
 const tbody=await element("tbody"),thead=await element("thead");expect((await html.element(await tag("table"),[],[thead,tbody])).kind).toBe("ok");
 for(const kids of [[],[thead],[tbody,thead],[tbody,tbody]])check(await html.element(await tag("table"),[],kids),1220);
 const dt=await element("dt"),dd=await element("dd");expect((await html.element(await tag("dl"),[],[dt,dd])).kind).toBe("ok");check(await html.element(await tag("dl"),[],[dd,dt]),1220);
 for(const name of ["a","form"]){const nested=await element("div",[await element(name)]);check(await html.element(await tag(name),[],[nested]),1220);}
});
test("URL parsing rejects origin and script ambiguity",async()=>{
 for(const input of ["/\ud800","https://example.com/\udfff","//evil.test","/\\evil.test","javascript:alert(1)","data:text/html,x","http://a.test"," https://a.test","https://a:b@a.test","/x\ny","/.//evil.test","/%2e//evil.test","https://"] )check(await html.parseURL(input),1221);
 for(const input of ["/","/a b?q=x&y=z#f","/a/../b","https://example.com/path?x=1&y=2"]){const url=value(await html.parseURL(input));const attr=value(await html.urlAttribute("href",url));expect((await html.element(await tag("a"),[attr],[])).kind).toBe("ok");check(await html.element(await tag("div"),[attr],[]),1220);}
 {const url=value(await html.parseURL("https://example.com/i.png"));const attr=value(await html.urlAttribute("src",url));expect((await html.element(await tag("img"),[attr],[])).kind).toBe("ok");check(await html.element(await tag("a"),[attr],[]),1220);check(await html.urlAttribute("srcset",url),1220);}
 const outside=value(await html.parseURL("https://example.com/"));check(await html.get(outside),1221);check(await html.post(outside),1221);
 const local=value(await html.parseURL("/search?q=a&x=b"));expect(await render([await element("div",[],[value(await html.get(local))])])).toBe('<div hx-get="/search?q=a&amp;x=b"></div>');
});
test("head-only nodes, native title escaping and pinned HTMX policy",async()=>{
 const runtime=value(await html.runtimeHead()),viewport=value(await html.metaViewport()),css=value(await html.stylesheet(value(await html.parseURL("/style.css"))));
 for(const n of [runtime,viewport,css]){check(await html.fragment([n]),1220);check(await html.element(await tag("div"),[],[n]),1220);check(await html.document("x",[],[n]),1220);}
 check(await html.document("x",[await text("x")],[]),1220);
 const output=renderSafe(value(await html.document("<title>",[viewport,css,runtime],[await text("<body>")])));
 expect(output.startsWith("<!doctype html><html><head><title>&lt;title&gt;</title>")).toBe(true);expect(output.endsWith("<body>&lt;body&gt;</body></html>")).toBe(true);
 expect(output).toContain('/__can/assets/htmx-4.0.0.min.js');expect(output).toContain('mode&quot;:&quot;same-origin');expect(output).toContain('noSwap');expect(output).toContain('sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc');
});
test("typed HTMX selectors and finite intervals cannot inject trigger code",async()=>{
 for(const id of ["","#id","x y","x,body","x]","1id","x\n","x:has(*)"]) {check(await html.targetID(id),1222);check(await html.indicatorID(id),1222);}
 const target=value(await html.targetID("results_1"));const attrs=[value(await html.targetAttribute(target)),value(await html.indicatorID("busy")),value(await html.swapInner()),value(await html.disableThis()),value(await html.triggerInputChanged(60000n))];
 expect(await render([await element("input",[],attrs)])).toBe('<input hx-target="#results_1" hx-indicator="#busy" hx-swap="innerHTML" hx-disable="this" hx-trigger="input changed delay:60000ms">');
 for(const n of [-1n,60001n,10n**50n])check(await html.triggerInputChanged(n),1223);
 for(const n of [0n,999n,3600001n])check(await html.triggerEvery(n),1223);
 for(const n of [1000n,3600000n])expect((await html.triggerEvery(n)).kind).toBe("ok");
});
test("forged opaque handles fail without inspecting proxies",async()=>{
 let traps=0;const forged=new Proxy({},{get(){traps++;throw Error("private");},getPrototypeOf(){traps++;throw Error("private");}});
 const calls=[()=>html.element(forged,[],[]),()=>html.fragment([forged]),()=>html.urlAttribute("href",forged),()=>html.targetAttribute(forged),()=>html.stylesheet(forged),()=>html.get(forged)];
 for(const call of calls)expect((await invoke(call,origin)).kind).toBe("standard");expect(()=>renderSafe(forged)).toThrow();expect(isHTMLValue("safe",forged)).toBe(false);expect(traps).toBe(0);
});
test("the complete author tag inventory and each tag-checked attribute are admitted",async()=>{
 for(const name of "main header footer nav section article aside h1 h2 h3 h4 h5 h6 p div span ul ol li a form label input textarea select option button table thead tbody tr th td dl dt dd strong em small br hr code pre blockquote img del".split(" "))expect((await html.makeTag(name)).kind).toBe("ok");
 for(const [name,val,tagName] of [["name","field","input"],["value","abc","input"],["type","submit","button"],["placeholder","hint","textarea"],["autocomplete","email","input"],["for","field","label"],["method","post","form"],["rel","noopener","a"],["checked","","input"],["selected","selected","option"],["disabled","","button"],["required","required","select"],["multiple","","select"],["rows","2","textarea"],["cols","20","textarea"],["scope","row","th"],["colspan","2","td"],["rowspan","0","td"],["alt","text","img"],["align","left","th"],["align","center","td"],["align","right","td"],["start","5","ol"],["start","-1","ol"]]){
  const a=value(await html.textAttribute(name!,val!));expect((await html.element(await tag(tagName!),[a],[])).kind).toBe("ok");check(await html.element(await tag("aside"),[a],[]),1220);
 }
});
