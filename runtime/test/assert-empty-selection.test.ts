import {test,expect} from "bun:test";
import {runAssertion} from '../assert/runner.ts';
import {callContext} from '../assert/context.ts';
import {withFixture} from '../assert/fixtures.ts';
import {settle,type Mode} from '../coordination.ts';
import {success} from '../completion.ts';
const root={package:'p',declaration:'p::main',name:'sample'};
const origin={source:'review',start:0,end:0,invocation:[]};
for(const inner of ['all','settled','any'] as Mode[])test(`empty ${inner} selection preserves lexical fixture order`,async()=>{
 const events:string[]=[];
 const rows=[0,1].map(i=>({selector:'sample',arguments:async()=>success([i]),expected:async()=>{events.push('row'+i);return success(i)}}));
 const result=await runAssertion({root,expected:async()=>success(0),actual:async ctx=>{
  await settle('all',[0,1].map(i=>({captures:[],run:async child=>{
   if(i===0){await settle(inner,[],child,'p::empty#0',[]);events.push('empty selected')}
   return callContext(child,'p::leaf#0',leaf=>withFixture(leaf,'table',rows,[i],()=>success(-1),origin));
  }})),ctx,'p::main#0',[[0],[1]]);return success(0);
 }});
 expect(result.passed).toBe(true);
 expect(events).toEqual(['empty selected','row0','row1']);
});
