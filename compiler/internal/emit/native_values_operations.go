package emit

import "fmt"

// nativeValuesOperationBindings maps the NT-I01 native operations to the
// single $canNative contribution: N-owner envelope calls over the injected
// dispatch. The adapter shares the $canTest.owner table, wired once in the
// shared state module; every method is async and takes the trailing call
// context. Catalogue renames map back to service logical names inside the
// adapter (allocate_gate to native.gate, install_fault to native.fault).
func nativeValuesOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.native@1::open":          "$canNative.open",
		"can.std.native@1::describe":      "$canNative.describe",
		"can.std.native@1::make":          "$canNative.make",
		"can.std.native@1::invoke":        "$canNative.invoke",
		"can.std.native@1::settle":        "$canNative.settle",
		"can.std.native@1::observe":       "$canNative.observe",
		"can.std.native@1::allocate_gate": "$canNative.allocateGate",
		"can.std.native@1::release":       "$canNative.release",
		"can.std.native@1::install_fault": "$canNative.installFault",
		"can.std.native@1::restore":       "$canNative.restore",
		"can.std.native@1::close":         "$canNative.close",
	}
	return bindingContribution{domain: "native", functions: functions}
}

// nativeValuesStateImports lists the native-values adapter factory module
// the shared state module needs.
func (assembly *programAssembly) nativeValuesStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/test-support/slices/i01/native.ts", Names: []ImportName{{"createNativeValues", "$canCreateNativeValues"}}},
	}
}

// nativeValuesStateValueImportNames lists the native-values factory value
// authored and assertion modules import from the state module.
func nativeValuesStateValueImportNames() []ImportName {
	return []ImportName{{"$canNative", "$canNative"}}
}

// declareNativeValuesState emits the native-values factory binding.
func (builder *stateBuilder) declareNativeValuesState() {
	builder.out.WriteString("export let $canNative:ReturnType<typeof $canCreateNativeValues>;\n")
}

// initializeNativeValuesState constructs the native-values adapter inside
// the shared initializer, after the test factory exists: open takes the
// owner first and the adapter caches the grant per session, so later
// sessionless calls (settle, release, restore) reuse their row grant and a
// foreign handle fails before any dispatch. Result records reuse the
// nominal identities the catalogue facts types reference.
func (builder *stateBuilder) initializeNativeValuesState() {
	fmt.Fprintf(&builder.out, "$canNative=$canCreateNativeValues($canDomain,{some:%s,none:%s,inertFacts:%s,observation:%s,observationResult:%s,taggedEntry:%s,taggedValue:%s,aliasGroup:%s,observeCounters:%s,descriptorOrUnknown:%s,handleOrPendingAction:%s,settlementOrPending:%s,releaseFacts:%s,restoreOutcome:%s,closeReceipt:%s},$canTest.owner);\n",
		quote(builder.optionIDs["can.std.option@1::some"]),
		quote(builder.optionIDs["can.std.option@1::none"]),
		quote(builder.numberIDs["can.std.native@1::inert_facts"]),
		quote(builder.numberIDs["can.std.native@1::observation"]),
		quote(builder.numberIDs["can.std.native@1::observation_result"]),
		quote(builder.numberIDs["can.std.native@1::tagged_entry"]),
		quote(builder.numberIDs["can.std.native@1::tagged_value"]),
		quote(builder.numberIDs["can.std.native@1::alias_group"]),
		quote(builder.numberIDs["can.std.native@1::observe_counters"]),
		quote(builder.numberIDs["can.std.native@1::descriptor_or_unknown"]),
		quote(builder.numberIDs["can.std.native@1::handle_or_pending_action"]),
		quote(builder.numberIDs["can.std.native@1::settlement_or_pending"]),
		quote(builder.numberIDs["can.std.native@1::release_facts"]),
		quote(builder.numberIDs["can.std.native@1::restore_outcome"]),
		quote(builder.numberIDs["can.std.native@1::close_receipt"]))
}
