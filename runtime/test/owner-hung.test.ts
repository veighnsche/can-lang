import {test,expect} from "bun:test";
import {spawn} from "node:child_process";
import {mkdtempSync,writeFileSync,rmSync} from "node:fs";
import {tmpdir} from "node:os";
import {join} from "node:path";
import {fileURLToPath} from "node:url";

for(const leased of [false,true])test(`external supervisor kills a nonsettling owner (leased=${leased})`,async()=>{
 const directory=mkdtempSync(join(tmpdir(),"can-owner-supervisor-"));
 const runtime=fileURLToPath(new URL("../",import.meta.url));
 const file=join(directory,"hung.ts");
 writeFileSync(file,`import {runOwnedRoot,launchOwned,registerResource,closeResource} from ${JSON.stringify(join(runtime,"owner.ts"))};
import {success,failure} from ${JSON.stringify(join(runtime,"completion.ts"))};
import {resourceStateFailure} from ${JSON.stringify(join(runtime,"failure.ts"))};
await runOwnedRoot(async()=>{
 const resource=${leased?'registerResource("pool",{},()=>{console.log("native-close");return success(undefined);})':'undefined'};
 const group=launchOwned([{captures:resource?[resource]:[],run:()=>new Promise(()=>{})}]);group.publish([]);
 ${leased?'await closeResource(resource,"pool",{milliseconds:0,failure:()=>failure(resourceStateFailure(undefined,{source:"test",start:0,end:0,invocation:[]}))});':''}
 console.log("owned-pending");return success(undefined);
});console.log("drained");`);
 const child=spawn(process.execPath,["--no-install",file],{stdio:["ignore","pipe","pipe"]});
 let output="",errors="",killTimer:ReturnType<typeof setTimeout>|undefined;
 const deadline=setTimeout(()=>child.kill("SIGKILL"),5000);
 child.stdout.on("data",chunk=>{output+=chunk;if(output.includes("owned-pending")&&!killTimer)killTimer=setTimeout(()=>child.kill("SIGKILL"),100);});
 child.stderr.on("data",chunk=>{errors+=chunk;});
 try{
  const status=await new Promise<{code:number|null;signal:NodeJS.Signals|null}>((resolve,reject)=>{child.once("error",reject);child.once("exit",(code,signal)=>resolve({code,signal}));});
  expect(output).toContain("owned-pending");expect(output).not.toContain("drained");expect(output).not.toContain("native-close");expect(errors).toBe("");expect(status.signal).toBe("SIGKILL");
 }finally{clearTimeout(deadline);if(killTimer)clearTimeout(killTimer);child.kill("SIGKILL");rmSync(directory,{recursive:true,force:true});}
},10000);
