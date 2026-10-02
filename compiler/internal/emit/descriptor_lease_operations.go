package emit

import "fmt"

// descriptorLeaseOperationBindings maps the NT-I12 inherited
// generation-lease operation to the single $canLease contribution:
// an N-owner envelope call over the injected dispatch. The adapter
// shares the $canTest.owner table, wired once in the shared state
// module; the method is async and takes the trailing call context.
// The probe observes live external owner lease state, so it
// follows the I11 same-package envelope precedent rather than an
// adapter-held service.
func descriptorLeaseOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.descriptor@1::probe_lease": "$canLease.probeLease",
	}
	return bindingContribution{domain: "lease", functions: functions}
}

// descriptorLeaseStateImports lists the lease-probe adapter factory
// module the shared state module needs.
func (assembly *programAssembly) descriptorLeaseStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/test-support/slices/i12/lease.ts", Names: []ImportName{{"createLeaseProbe", "$canCreateLeaseProbe"}}},
	}
}

// descriptorLeaseStateValueImportNames lists the lease-probe factory
// value authored and assertion modules import from the state module.
func descriptorLeaseStateValueImportNames() []ImportName {
	return []ImportName{{"$canLease", "$canLease"}}
}

// declareDescriptorLeaseState emits the lease-probe factory binding.
func (builder *stateBuilder) declareDescriptorLeaseState() {
	builder.out.WriteString("export let $canLease:ReturnType<typeof $canCreateLeaseProbe>;\n")
}

// initializeDescriptorLeaseState constructs the lease-probe adapter
// inside the shared initializer, after the test factory exists:
// the op takes the owner first and the adapter admits the grant
// before any dispatch, so a foreign or released owner fails
// stale_handle without touching the N transport. The probe result
// reuses the nominal lease_report identity the catalogue type
// references.
func (builder *stateBuilder) initializeDescriptorLeaseState() {
	fmt.Fprintf(&builder.out, "$canLease=$canCreateLeaseProbe($canDomain,{staleHandle:%s,descriptorFault:%s,leaseReport:%s},$canTest.owner);\n",
		quote(builder.numberIDs["can.std.test@1::stale_handle"]),
		quote(builder.numberIDs["can.std.descriptor@1::descriptor_fault"]),
		quote(builder.numberIDs["can.std.descriptor@1::lease_report"]))
}
