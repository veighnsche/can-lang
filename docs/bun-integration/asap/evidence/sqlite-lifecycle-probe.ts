// Further native evidence only; no compiler or production runtime imports.
import {mkdtemp, rm} from "node:fs/promises";
import {tmpdir} from "node:os";
import {join} from "node:path";
const directory=await mkdtemp(join(tmpdir(),"can-sqlite-asap-"));
const filename=join(directory,"test.sqlite");
const report:Record<string,unknown>={version:Bun.version,revision:Bun.revision};
let db=new Bun.SQL({adapter:"sqlite",filename,safeIntegers:true});
try {
  await db`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)`;
  await db.begin(async tx=>{
    await tx`INSERT INTO items VALUES (${9007199254740993n}, ${"persisted"})`;
    await Bun.sleep(1);
    await tx`INSERT INTO items VALUES (${2n}, ${"after await"})`;
  });
  report.commit=await db`SELECT id,name FROM items ORDER BY id`;
  let failed=false;
  try {await db.begin(async tx=>{await tx`INSERT INTO items VALUES (${3n}, ${"must roll back"})`;await Bun.sleep(1);throw new Error("deliberate rollback");});}
  catch(e){failed=e.message==="deliberate rollback";}
  report.rollback={originalError:failed,rows:await db`SELECT id FROM items WHERE id = ${3n}`};
  await db.close();
  db=new Bun.SQL({adapter:"sqlite",filename,safeIntegers:true});
  report.reopened=await db`SELECT id,name FROM items ORDER BY id`;
} catch(e) {report.failure={name:e.name,message:e.message};process.exitCode=1;}
finally {await db.close();await rm(directory,{recursive:true,force:true});}
console.log(JSON.stringify(report,(_,v)=>typeof v==="bigint"?{bigint:v.toString()}:v,2));
