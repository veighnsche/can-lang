import { runAssertionRoot as $canRunAssertionRoot } from "./runtime/assert/runner.ts";
import { $canInitialize as $canInitialize } from "./program/state.ts";
import { configureDiagnostics as $canConfigureDiagnostics } from "./runtime/diagnostics.ts";
import { $canCase as $canCase0 } from "./assertions/6abe8e13774b07642245601f17e48741d3a9c7cf581b258e1238c7911615f0fb.ts";
import { $canCase as $canCase1 } from "./assertions/a269b191d6bb1d4e6f6b2a3809d8618fdec1e58419d8be9788ec606a36bf8b7b.ts";
import { $canCase as $canCase2 } from "./assertions/436b2b5b2e09b8d954ce598c8ed41ee25dc26ae6c95010e13259f35dc257b121.ts";
process.exitCode = await $canRunAssertionRoot([$canCase0,$canCase1,$canCase2], () => {$canConfigureDiagnostics(import.meta.url); $canInitialize();}, process.argv.slice(2));
