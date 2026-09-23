package emit

import "fmt"

// processOperationBindings maps the B1-04 process operations to their
// state-module target.
func processOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.process@1::run":             "$canProcesses.run",
		"can.std.process@1::require_success": "$canProcesses.requireSuccess",
		"can.std.process@1::which":           "$canProcesses.which",
	}
	return bindingContribution{domain: "process", functions: functions}
}

// processStateImports lists the process factory module the shared state
// module needs.
func (assembly *programAssembly) processStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/platform/process/spawn.ts", Names: []ImportName{{"createProcesses", "$canCreateProcesses"}}},
	}
}

// processStateValueImportNames lists the process factory value authored and
// assertion modules import from the state module.
func processStateValueImportNames() []ImportName {
	return []ImportName{{"$canProcesses", "$canProcesses"}}
}

// declareProcessState emits the process factory binding.
func (builder *stateBuilder) declareProcessState() {
	builder.out.WriteString("export let $canProcesses:ReturnType<typeof $canCreateProcesses>;\n")
}

// initializeProcessState constructs the process factory inside the shared
// initializer, after the domain runtime exists.
func (builder *stateBuilder) initializeProcessState() {
	fmt.Fprintf(&builder.out, "$canProcesses=$canCreateProcesses($canDomain,{notFound:%s,denied:%s,spawnFailed:%s,timeout:%s,outputLimit:%s,nonzero:%s,invalidConfig:%s,ioError:%s,result:%s});\n", quote(builder.numberIDs["can.std.files@1::not_found"]), quote(builder.numberIDs["can.std.files@1::denied"]), quote(builder.numberIDs["can.std.process@1::spawn_failed"]), quote(builder.numberIDs["can.std.process@1::timeout"]), quote(builder.numberIDs["can.std.process@1::output_limit"]), quote(builder.numberIDs["can.std.process@1::nonzero"]), quote(builder.numberIDs["can.std.process@1::invalid_config"]), quote(builder.numberIDs["can.std.process@1::io_error"]), quote(builder.numberIDs["can.std.process@1::result"]))
}
