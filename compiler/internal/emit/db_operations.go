package emit

import "fmt"

// dbOperationBindings maps the NT-I13 database-observer operations
// to the single $canDb contribution: owner-admitted calls over one
// adapter-held K22 core plus one K23 RETURNING observer with
// verbatim db_fault mapping. The adapter shares the
// $canTest.owner table, wired once in the shared state module;
// every method is async and takes the trailing call context. The
// catalogue snake_case names map to the adapter camelCase calls.
func dbOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.db@1::open_namespace":            "$canDb.openNamespace",
		"can.std.db@1::receipt":                   "$canDb.receipt",
		"can.std.db@1::close_namespace":           "$canDb.closeNamespace",
		"can.std.db@1::seed":                      "$canDb.seed",
		"can.std.db@1::pin_connection":            "$canDb.pinConnection",
		"can.std.db@1::connection_token_for_test": "$canDb.connectionTokenForTest",
		"can.std.db@1::unpin_connection":          "$canDb.unpinConnection",
		"can.std.db@1::begin_read":                "$canDb.beginRead",
		"can.std.db@1::fetch":                     "$canDb.fetch",
		"can.std.db@1::end_read":                  "$canDb.endRead",
		"can.std.db@1::compare_row":               "$canDb.compareRow",
		"can.std.db@1::record_compile":            "$canDb.recordCompile",
		"can.std.db@1::compile_record":            "$canDb.compileRecord",
		"can.std.db@1::record_returning":          "$canDb.recordReturning",
		"can.std.db@1::payload_facts":             "$canDb.payloadFacts",
		"can.std.db@1::record_final_rows":         "$canDb.recordFinalRows",
		"can.std.db@1::final_facts":               "$canDb.finalFacts",
		"can.std.db@1::compare_payload_to_final":  "$canDb.comparePayloadToFinal",
		"can.std.db@1::compare_claimed_row":       "$canDb.compareClaimedRow",
		"can.std.db@1::credit_verdict":            "$canDb.creditVerdict",
	}
	return bindingContribution{domain: "db", functions: functions}
}

// dbStateImports lists the database-observer adapter factory module
// the shared state module needs.
func (assembly *programAssembly) dbStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/test-support/slices/i13/db.ts", Names: []ImportName{{"createDbObserver", "$canCreateDbObserver"}}},
	}
}

// dbStateValueImportNames lists the database-observer factory value
// authored and assertion modules import from the state module.
func dbStateValueImportNames() []ImportName {
	return []ImportName{{"$canDb", "$canDb"}}
}

// declareDbState emits the database-observer factory binding.
func (builder *stateBuilder) declareDbState() {
	builder.out.WriteString("export let $canDb:ReturnType<typeof $canCreateDbObserver>;\n")
}

// initializeDbState constructs the database-observer adapter inside
// the shared initializer, after the test factory exists: every op
// takes the owner first and the adapter admits the grant before any
// compute, so a foreign or released owner fails stale_handle
// without touching either service. Result records reuse the nominal
// identities the catalogue fact types reference.
func (builder *stateBuilder) initializeDbState() {
	fmt.Fprintf(&builder.out, "$canDb=$canCreateDbObserver($canDomain,{staleHandle:%s,dbFault:%s,numberCell:%s,textCell:%s,bytesCell:%s,nullCell:%s,namespaceReceipt:%s,connectionFacts:%s,rowFacts:%s,readFacts:%s,seedFacts:%s,conversationFacts:%s,rowComparison:%s,compileFacts:%s,returningRowFacts:%s,returningPayloadFacts:%s,finalRowsFacts:%s,returningComparison:%s,returningCreditVerdict:%s},$canTest.owner);\n",
		quote(builder.numberIDs["can.std.test@1::stale_handle"]),
		quote(builder.numberIDs["can.std.db@1::db_fault"]),
		quote(builder.numberIDs["can.std.db@1::number_cell"]),
		quote(builder.numberIDs["can.std.db@1::text_cell"]),
		quote(builder.numberIDs["can.std.db@1::bytes_cell"]),
		quote(builder.numberIDs["can.std.db@1::null_cell"]),
		quote(builder.numberIDs["can.std.db@1::namespace_receipt"]),
		quote(builder.numberIDs["can.std.db@1::connection_facts"]),
		quote(builder.numberIDs["can.std.db@1::row_facts"]),
		quote(builder.numberIDs["can.std.db@1::read_facts"]),
		quote(builder.numberIDs["can.std.db@1::seed_facts"]),
		quote(builder.numberIDs["can.std.db@1::conversation_facts"]),
		quote(builder.numberIDs["can.std.db@1::row_comparison"]),
		quote(builder.numberIDs["can.std.db@1::compile_facts"]),
		quote(builder.numberIDs["can.std.db@1::returning_row_facts"]),
		quote(builder.numberIDs["can.std.db@1::returning_payload_facts"]),
		quote(builder.numberIDs["can.std.db@1::final_rows_facts"]),
		quote(builder.numberIDs["can.std.db@1::returning_comparison"]),
		quote(builder.numberIDs["can.std.db@1::returning_credit_verdict"]))
}
