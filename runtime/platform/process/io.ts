// Bounded pipe drains and stdin delivery for owned child processes.
// Drains run through the shared stream lifecycle (open/pull/close with
// counted-before-retaining caps); stream failures still reject so the
// spawn contract keeps its stream_failed outcome.
import {drainStream} from "../../transport/stream/readable.ts";
import type {Fail} from "../../transport/stream/lifecycle.ts";
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
export async function drainBounded(stream:ReadableStream<Uint8Array>,limit:bigint,fail:Fail,terminalFailed:string):Promise<DrainOutcome>{
  const outcome=await drainStream(stream,limit,fail,terminalFailed);
  if(outcome.overflow)return {overflow:true};
  return {overflow:false,data:outcome.data};
}
export function deliverStdin(proc:ChildProcess,data:Uint8Array):void{
  if(data.byteLength!==0)proc.stdin.write(data);
  proc.stdin.flush();proc.stdin.end();
}
