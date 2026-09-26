import { $canInitialize as $canInitialize } from "./program/state.ts";
import { $canFunction1 as $canMain } from "./packages/p-df46aedee861e25239c787824bf1a5dd52302eaef1c965db358166e0dae5b4f1/s-a38a111be501eff38638f21703b1642f7a5cdd7f8c4da52c8d4620aa7473e881.ts";
import {standardFailureDiagnostics} from "./runtime/failure.ts";
$canInitialize();
const result = await $canMain([]);
if (result.kind === "standard") { const detail=standardFailureDiagnostics(result.value); console.log(result.kind, detail.kind, detail.cause); process.exitCode=1; }
else { console.log(result.kind); process.exitCode=result.kind === "ok" ? 0 : 1; }
