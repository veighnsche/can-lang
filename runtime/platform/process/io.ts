// Bounded pipe drains and stdin delivery for owned child processes.
// Streams are counted before retaining; readers are always released.
export type ChildProcess=Readonly<{
 pid:number;
 stdin:Readonly<{write(chunk:Uint8Array):unknown;flush():unknown;end():unknown}>;
 stdout:ReadableStream<Uint8Array>;
 stderr:ReadableStream<Uint8Array>;
 readonly exitCode:number|null;
 readonly signalCode:string|null;
 readonly exited:Promise<number>;
}>;
export type DrainOutcome=Readonly<{overflow:false;data:Uint8Array}|{overflow:true}>;
export async function drainBounded(stream:ReadableStream<Uint8Array>,limit:bigint):Promise<DrainOutcome>{
 const reader=stream.getReader();let complete=false;
 try{
  const chunks:Uint8Array[]=[];let size=0n;
  for(;;){
   const next=await reader.read();if(next.done){complete=true;break;}
   const length=BigInt(next.value.byteLength);
   if(length>limit-size)return {overflow:true};
   size+=length;
   if(length!==0n)chunks.push(new Uint8Array(next.value));
  }
  const data=new Uint8Array(Number(size));let offset=0;
  for(const chunk of chunks){data.set(chunk,offset);offset+=chunk.byteLength;}
  return {overflow:false,data};
 }finally{
  if(!complete)try{await reader.cancel();}catch{}
  reader.releaseLock();
 }
}
export function deliverStdin(proc:ChildProcess,data:Uint8Array):void{
 if(data.byteLength!==0)proc.stdin.write(data);
 proc.stdin.flush();proc.stdin.end();
}
