package emit

import "fmt"

// storeOperationBindings maps the NT-I17 object-store
// operations to the single $canStore contribution:
// owner-admitted calls over one adapter-held K27 service with
// verbatim store_fault mapping. The adapter shares the
// $canTest.owner table, wired once in the shared state module;
// every method is async and takes the trailing call context.
// The catalogue snake_case names map to the adapter camelCase
// calls; session_token_for_test passes through verbatim like
// db::connection_token_for_test.
func storeOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.store@1::open_prefix":            "$canStore.openPrefix",
		"can.std.store@1::receipt":                "$canStore.receipt",
		"can.std.store@1::close_prefix":           "$canStore.closePrefix",
		"can.std.store@1::open_session":           "$canStore.openSession",
		"can.std.store@1::session_token_for_test": "$canStore.sessionTokenForTest",
		"can.std.store@1::close_session":          "$canStore.closeSession",
		"can.std.store@1::put":                    "$canStore.put",
		"can.std.store@1::settle_write":           "$canStore.settleWrite",
		"can.std.store@1::pending":                "$canStore.pending",
		"can.std.store@1::get":                    "$canStore.get",
		"can.std.store@1::delete":                 "$canStore.delete",
		"can.std.store@1::compare_bytes":          "$canStore.compareBytes",
		"can.std.store@1::list":                   "$canStore.list",
		"can.std.store@1::seal_cleanup":           "$canStore.sealCleanup",
		"can.std.store@1::shared_digest":          "$canStore.sharedDigest",
	}
	return bindingContribution{domain: "store", functions: functions}
}

// storeStateImports lists the object-store adapter factory
// module the shared state module needs.
func (assembly *programAssembly) storeStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/test-support/slices/i17/store.ts", Names: []ImportName{{"createStore", "$canCreateStore"}}},
	}
}

// storeStateValueImportNames lists the object-store factory
// value authored and assertion modules import from the state
// module.
func storeStateValueImportNames() []ImportName {
	return []ImportName{{"$canStore", "$canStore"}}
}

// declareStoreState emits the object-store factory binding.
func (builder *stateBuilder) declareStoreState() {
	builder.out.WriteString("export let $canStore:ReturnType<typeof $canCreateStore>;\n")
}

// initializeStoreState constructs the object-store adapter
// inside the shared initializer, after the test factory
// exists: every op takes the owner first and the adapter admits
// the grant before any compute, so a foreign or released owner
// fails stale_handle without touching the service. Result
// records reuse the nominal identities the catalogue fact types
// reference.
func (builder *stateBuilder) initializeStoreState() {
	fmt.Fprintf(&builder.out, "$canStore=$canCreateStore($canDomain,{staleHandle:%s,storeFault:%s,optionSome:%s,optionNone:%s,prefixReceipt:%s,sessionFacts:%s,objectFacts:%s,writeFacts:%s,pageFacts:%s,pendingFacts:%s,storedObject:%s,cleanupReceipt:%s},$canTest.owner);\n",
		quote(builder.numberIDs["can.std.test@1::stale_handle"]),
		quote(builder.numberIDs["can.std.store@1::store_fault"]),
		quote(builder.optionIDs["can.std.option@1::some"]),
		quote(builder.optionIDs["can.std.option@1::none"]),
		quote(builder.numberIDs["can.std.store@1::prefix_receipt"]),
		quote(builder.numberIDs["can.std.store@1::session_facts"]),
		quote(builder.numberIDs["can.std.store@1::object_facts"]),
		quote(builder.numberIDs["can.std.store@1::write_facts"]),
		quote(builder.numberIDs["can.std.store@1::page_facts"]),
		quote(builder.numberIDs["can.std.store@1::pending_facts"]),
		quote(builder.numberIDs["can.std.store@1::stored_object"]),
		quote(builder.numberIDs["can.std.store@1::cleanup_receipt"]))
}
