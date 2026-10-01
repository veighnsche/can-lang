package emit

import "fmt"

// cOperationBindings maps the NT-I02 C-ingress operations to the single
// $canC contribution: pure local seam checks over the ported K04 checker,
// with owner admission before any compute. The adapter shares the
// $canTest.owner table, wired once in the shared state module; every method
// is async and takes the trailing call context.
func cOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.c@1::parse_module": "$canC.parseModule",
		"can.std.c@1::check_module": "$canC.checkModule",
	}
	return bindingContribution{domain: "c", functions: functions}
}

// cStateImports lists the C-ingress adapter factory module the shared state
// module needs.
func (assembly *programAssembly) cStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/test-support/slices/i02/c.ts", Names: []ImportName{{"createCIngress", "$canCreateCIngress"}}},
	}
}

// cStateValueImportNames lists the C-ingress factory value authored and
// assertion modules import from the state module.
func cStateValueImportNames() []ImportName {
	return []ImportName{{"$canC", "$canC"}}
}

// declareCState emits the C-ingress factory binding.
func (builder *stateBuilder) declareCState() {
	builder.out.WriteString("export let $canC:ReturnType<typeof $canCreateCIngress>;\n")
}

// initializeCState constructs the C-ingress adapter inside the shared
// initializer, after the test factory exists: every op takes the owner
// first and the adapter admits the grant before any compute, so a foreign
// or released owner fails stale_handle without touching the seam checker.
// Result records reuse the nominal identities the catalogue parsed types
// reference.
func (builder *stateBuilder) initializeCState() {
	fmt.Fprintf(&builder.out, "$canC=$canCreateCIngress($canDomain,{staleHandle:%s,some:%s,none:%s,parsedModule:%s,parsedEdge:%s,parsedName:%s},$canTest.owner);\n",
		quote(builder.numberIDs["can.std.test@1::stale_handle"]),
		quote(builder.optionIDs["can.std.option@1::some"]),
		quote(builder.optionIDs["can.std.option@1::none"]),
		quote(builder.numberIDs["can.std.c@1::parsed_module"]),
		quote(builder.numberIDs["can.std.c@1::parsed_edge"]),
		quote(builder.numberIDs["can.std.c@1::parsed_name"]))
}
