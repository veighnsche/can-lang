import {callableInstance as $canCallableInstance} from "/Users/vince/Projects/can-lang/runtime/callable.ts";
import {callContext as $canCallContext} from "/Users/vince/Projects/can-lang/runtime/assert/context.ts";
import { success as $canSuccess, failure as $canFailure, value as $canValue, invoke as $canInvoke, caught as $canCaught, errorType as $canErrorType, errorPayload as $canErrorPayload, type Completion as $canCompletion, type AssertionContext as $canAssertionContext } from "/Users/vince/Projects/can-lang/runtime/completion.ts";
type $canType4470002cfedcccd19766ce54cfeac77f9c59cdabb22994f8a98fc52a2e8fef1f = boolean;
type $canType9353abd7132d355d7bfdef4cbb5d174ed0ecc613e3f7f32c4f8937c9bc76de71 = bigint;
type $canTypeb8e91cd161472ff64c8c68001eda803adc101e4ed35bdf62e579827cd88e47de = (arg0: $canType9353abd7132d355d7bfdef4cbb5d174ed0ecc613e3f7f32c4f8937c9bc76de71, $canContext?: $canAssertionContext) => Promise<$canCompletion<$canType9353abd7132d355d7bfdef4cbb5d174ed0ecc613e3f7f32c4f8937c9bc76de71>>;
async function countdown($canArg0: $canType9353abd7132d355d7bfdef4cbb5d174ed0ecc613e3f7f32c4f8937c9bc76de71, $canContext?: $canAssertionContext): Promise<$canCompletion<$canType9353abd7132d355d7bfdef4cbb5d174ed0ecc613e3f7f32c4f8937c9bc76de71>> {
try {
const $canExpr1 = ($canArg0) as unknown as $canType9353abd7132d355d7bfdef4cbb5d174ed0ecc613e3f7f32c4f8937c9bc76de71;
let $canExpr2 = true;
const $canExpr3 = 0n;
$canExpr2 = ($canExpr1 === $canExpr3);
if (($canExpr2 === false)) {
let $canRegion1: $canCompletion<unknown>;
let $canRegion3!: $canType9353abd7132d355d7bfdef4cbb5d174ed0ecc613e3f7f32c4f8937c9bc76de71;
$canRegion2: { try {
const $canExpr4 = ($canArg0) as unknown as $canType9353abd7132d355d7bfdef4cbb5d174ed0ecc613e3f7f32c4f8937c9bc76de71;
const $canExpr5 = 1n;
const $canExpr6 = ($canExpr4) - ($canExpr5);
const $canRegion4 = $canExpr6;
const $canExpr7 = ($canRegion4) as unknown as $canType9353abd7132d355d7bfdef4cbb5d174ed0ecc613e3f7f32c4f8937c9bc76de71;
$canRegion1 = await $canInvoke(() => $canCallContext($canContext,"can.project.root/app::countdown#0",($canContext) => countdown($canExpr7, $canContext),$canCallableInstance(countdown)), {source:"/private/var/folders/5x/bpszcvmj1tddmjp35y21dt680000gn/T/TestCoreBlindRecursionProbe2948454640/001/src/main.can",start:193,end:217,invocation:["can.project.root/app::countdown"]});
if ($canRegion1.kind !== 'ok') break $canRegion2;
$canRegion3 = $canValue($canRegion1) as $canType9353abd7132d355d7bfdef4cbb5d174ed0ecc613e3f7f32c4f8937c9bc76de71;
} catch ($canCause) { $canRegion1 = $canCaught($canCause, {source:"/private/var/folders/5x/bpszcvmj1tddmjp35y21dt680000gn/T/TestCoreBlindRecursionProbe2948454640/001/src/main.can",start:188,end:217,invocation:["can.project.root/app::countdown"]}); } }
return $canRegion1 as $canCompletion<$canType9353abd7132d355d7bfdef4cbb5d174ed0ecc613e3f7f32c4f8937c9bc76de71>;
}
else if (($canExpr2 === true)) {
const $canExpr8 = ($canArg0) as unknown as $canType9353abd7132d355d7bfdef4cbb5d174ed0ecc613e3f7f32c4f8937c9bc76de71;
return $canSuccess($canExpr8);
}
else { throw new Error('checked match was not exhaustive'); }
} catch ($canCause) { return $canCaught($canCause, {source:"/private/var/folders/5x/bpszcvmj1tddmjp35y21dt680000gn/T/TestCoreBlindRecursionProbe2948454640/001/src/main.can",start:144,end:247,invocation:["can.project.root/app::countdown"]}); }
}

for(const n of [100n,1000n,10000n]) console.log(String(n), (await countdown(n)).kind);
import {standardFailureKind, standardFailureMessage} from '/Users/vince/Projects/can-lang/runtime/failure.ts';
const result=await countdown(10000n);if(result.kind==='standard')console.log(standardFailureKind(result.value), standardFailureMessage(result.value));
