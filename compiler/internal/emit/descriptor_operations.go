package emit

import "fmt"

// descriptorOperationBindings maps the NT-I11 descriptor-delivery operations
// to the single $canDescriptor contribution: N-owner envelope calls over the
// injected dispatch. The adapter shares the $canTest.owner table, wired once
// in the shared state module; every method is async and takes the trailing
// call context. The catalogue launch_child rename (launch is the opaque
// launch type) maps to the adapter launchChild envelope call.
func descriptorOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.descriptor@1::launch_child":   "$canDescriptor.launchChild",
		"can.std.descriptor@1::collect_status": "$canDescriptor.collectStatus",
		"can.std.descriptor@1::wait_child":     "$canDescriptor.waitChild",
		"can.std.descriptor@1::kill_child":     "$canDescriptor.killChild",
		"can.std.descriptor@1::release_launch": "$canDescriptor.releaseLaunch",
		"can.std.descriptor@1::read_facts":     "$canDescriptor.readFacts",
		"can.std.descriptor@1::expected_ack":   "$canDescriptor.expectedAck",
	}
	return bindingContribution{domain: "descriptor", functions: functions}
}

// descriptorStateImports lists the descriptor-delivery adapter factory module
// the shared state module needs.
func (assembly *programAssembly) descriptorStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/test-support/slices/i11/descriptor.ts", Names: []ImportName{{"createDescriptorDelivery", "$canCreateDescriptorDelivery"}}},
	}
}

// descriptorStateValueImportNames lists the descriptor-delivery factory value
// authored and assertion modules import from the state module.
func descriptorStateValueImportNames() []ImportName {
	return []ImportName{{"$canDescriptor", "$canDescriptor"}}
}

// declareDescriptorState emits the descriptor-delivery factory binding.
func (builder *stateBuilder) declareDescriptorState() {
	builder.out.WriteString("export let $canDescriptor:ReturnType<typeof $canCreateDescriptorDelivery>;\n")
}

// initializeDescriptorState constructs the descriptor-delivery adapter inside
// the shared initializer, after the test factory exists: every op takes the
// owner first and the adapter admits the grant before any dispatch, so a
// foreign or released owner fails stale_handle without touching the N
// transport. Result records reuse the nominal identities the catalogue
// facts types reference.
func (builder *stateBuilder) initializeDescriptorState() {
	fmt.Fprintf(&builder.out, "$canDescriptor=$canCreateDescriptorDelivery($canDomain,{staleHandle:%s,descriptorFault:%s,some:%s,none:%s,exit:%s,facts:%s,leaseReport:%s,report:%s},$canTest.owner);\n",
		quote(builder.numberIDs["can.std.test@1::stale_handle"]),
		quote(builder.numberIDs["can.std.descriptor@1::descriptor_fault"]),
		quote(builder.optionIDs["can.std.option@1::some"]),
		quote(builder.optionIDs["can.std.option@1::none"]),
		quote(builder.numberIDs["can.std.descriptor@1::exit"]),
		quote(builder.numberIDs["can.std.descriptor@1::facts"]),
		quote(builder.numberIDs["can.std.descriptor@1::lease_report"]),
		quote(builder.numberIDs["can.std.descriptor@1::report"]))
}
