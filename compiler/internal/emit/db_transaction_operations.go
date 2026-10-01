package emit

import "fmt"

// dbTransactionOperationBindings maps the NT-I14 transaction
// operations to the single $canDbTxn contribution: owner-admitted
// calls over one adapter-held K24 service with verbatim db_fault
// mapping. The adapter shares the $canTest.owner table, wired
// once in the shared state module; every method is async and
// takes the trailing call context. The catalogue snake_case
// names map to the adapter camelCase calls.
func dbTransactionOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.db@1::enter_callback":         "$canDbTxn.enterCallback",
		"can.std.db@1::actor_facts":            "$canDbTxn.actorFacts",
		"can.std.db@1::record_last_insert_id":  "$canDbTxn.recordLastInsertId",
		"can.std.db@1::last_insert_id_facts":   "$canDbTxn.lastInsertIdFacts",
		"can.std.db@1::compare_last_insert_id": "$canDbTxn.compareLastInsertId",
		"can.std.db@1::record_settlement":      "$canDbTxn.recordSettlement",
		"can.std.db@1::settlement_facts":       "$canDbTxn.settlementFacts",
		"can.std.db@1::release":                "$canDbTxn.release",
		"can.std.db@1::release_ack":            "$canDbTxn.releaseAck",
	}
	return bindingContribution{domain: "dbtxn", functions: functions}
}

// dbTransactionStateImports lists the transaction-observer adapter
// factory module the shared state module needs.
func (assembly *programAssembly) dbTransactionStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/test-support/slices/i14/db_transaction.ts", Names: []ImportName{{"createDbTransaction", "$canCreateDbTransaction"}}},
	}
}

// dbTransactionStateValueImportNames lists the transaction-observer
// factory value authored and assertion modules import from the
// state module.
func dbTransactionStateValueImportNames() []ImportName {
	return []ImportName{{"$canDbTxn", "$canDbTxn"}}
}

// declareDbTransactionState emits the transaction-observer factory
// binding.
func (builder *stateBuilder) declareDbTransactionState() {
	builder.out.WriteString("export let $canDbTxn:ReturnType<typeof $canCreateDbTransaction>;\n")
}

// initializeDbTransactionState constructs the transaction-observer
// adapter inside the shared initializer, after the test factory
// exists: every op takes the owner first and the adapter admits
// the grant before any compute, so a foreign or released owner
// fails stale_handle without touching the service. Result records
// reuse the nominal identities the catalogue fact types
// reference.
func (builder *stateBuilder) initializeDbTransactionState() {
	fmt.Fprintf(&builder.out, "$canDbTxn=$canCreateDbTransaction($canDomain,{staleHandle:%s,dbFault:%s,numberCell:%s,textCell:%s,bytesCell:%s,nullCell:%s,callbackEntryFacts:%s,callbackEntry:%s,actorIdentityFacts:%s,lastInsertIdRecord:%s,identityComparison:%s,settlementRecord:%s,releaseRecord:%s},$canTest.owner);\n",
		quote(builder.numberIDs["can.std.test@1::stale_handle"]),
		quote(builder.numberIDs["can.std.db@1::db_fault"]),
		quote(builder.numberIDs["can.std.db@1::number_cell"]),
		quote(builder.numberIDs["can.std.db@1::text_cell"]),
		quote(builder.numberIDs["can.std.db@1::bytes_cell"]),
		quote(builder.numberIDs["can.std.db@1::null_cell"]),
		quote(builder.numberIDs["can.std.db@1::callback_entry_facts"]),
		quote(builder.numberIDs["can.std.db@1::callback_entry"]),
		quote(builder.numberIDs["can.std.db@1::actor_identity_facts"]),
		quote(builder.numberIDs["can.std.db@1::last_insert_id_record"]),
		quote(builder.numberIDs["can.std.db@1::identity_comparison"]),
		quote(builder.numberIDs["can.std.db@1::settlement_record"]),
		quote(builder.numberIDs["can.std.db@1::release_record"]))
}
