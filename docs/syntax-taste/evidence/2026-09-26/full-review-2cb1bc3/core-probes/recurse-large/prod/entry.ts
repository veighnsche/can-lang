import { runEntry as $canRunEntry } from "./runtime/entry.ts";
import { $canInitialize as $canInitialize } from "./program/state.ts";
import { configureDiagnostics as $canConfigureDiagnostics } from "./runtime/diagnostics.ts";
import { $canFunction1 as $canMain } from "./packages/p-df46aedee861e25239c787824bf1a5dd52302eaef1c965db358166e0dae5b4f1/s-a38a111be501eff38638f21703b1642f7a5cdd7f8c4da52c8d4620aa7473e881.ts";
process.exitCode = await $canRunEntry(() => {$canConfigureDiagnostics(import.meta.url); $canInitialize();}, $canMain, process.argv.slice(2));
