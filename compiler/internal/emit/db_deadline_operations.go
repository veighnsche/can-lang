package emit

import "fmt"

// dbDeadlineOperationBindings maps the NT-I16 deadline operations
// to the single $canDbDeadline contribution: owner-admitted calls
// over one adapter-held K26 service with verbatim db_fault
// mapping. The adapter shares the $canTest.owner table, wired
// once in the shared state module; every method is async and
// takes the trailing call context. The catalogue snake_case
// names map to the adapter camelCase calls; release/releaseAck
// become deadlineRelease/deadlineReleaseAck because the I14
// family already owns db::release and db::release_ack.
func dbDeadlineOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.db@1::dispatch":                 "$canDbDeadline.dispatch",
		"can.std.db@1::dispatch_facts":           "$canDbDeadline.dispatchFacts",
		"can.std.db@1::observe_deadline":         "$canDbDeadline.observeDeadline",
		"can.std.db@1::deadline_facts":           "$canDbDeadline.deadlineFacts",
		"can.std.db@1::record_driver_settlement": "$canDbDeadline.recordDriverSettlement",
		"can.std.db@1::driver_facts":             "$canDbDeadline.driverFacts",
		"can.std.db@1::record_server_ack":        "$canDbDeadline.recordServerAck",
		"can.std.db@1::server_facts":             "$canDbDeadline.serverFacts",
		"can.std.db@1::acquire_cancel_grant":     "$canDbDeadline.acquireCancelGrant",
		"can.std.db@1::request_cancel":           "$canDbDeadline.requestCancel",
		"can.std.db@1::cancel_facts":             "$canDbDeadline.cancelFacts",
		"can.std.db@1::quiesce_engine":           "$canDbDeadline.quiesceEngine",
		"can.std.db@1::fence":                    "$canDbDeadline.fence",
		"can.std.db@1::fence_facts":              "$canDbDeadline.fenceFacts",
		"can.std.db@1::lease_facts":              "$canDbDeadline.leaseFacts",
		"can.std.db@1::deadline_release":         "$canDbDeadline.deadlineRelease",
		"can.std.db@1::deadline_release_ack":     "$canDbDeadline.deadlineReleaseAck",
	}
	return bindingContribution{domain: "dbdeadline", functions: functions}
}

// dbDeadlineStateImports lists the deadline-observer adapter
// factory module the shared state module needs.
func (assembly *programAssembly) dbDeadlineStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/test-support/slices/i16/db_deadline.ts", Names: []ImportName{{"createDbDeadline", "$canCreateDbDeadline"}}},
	}
}

// dbDeadlineStateValueImportNames lists the deadline-observer
// factory value authored and assertion modules import from the
// state module.
func dbDeadlineStateValueImportNames() []ImportName {
	return []ImportName{{"$canDbDeadline", "$canDbDeadline"}}
}

// declareDbDeadlineState emits the deadline-observer factory binding.
func (builder *stateBuilder) declareDbDeadlineState() {
	builder.out.WriteString("export let $canDbDeadline:ReturnType<typeof $canCreateDbDeadline>;\n")
}

// initializeDbDeadlineState constructs the deadline-observer
// adapter inside the shared initializer, after the test factory
// exists: every op takes the owner first and the adapter admits
// the grant before any compute, so a foreign or released owner
// fails stale_handle without touching the service. Result
// records reuse the nominal identities the catalogue fact types
// reference.
func (builder *stateBuilder) initializeDbDeadlineState() {
	fmt.Fprintf(&builder.out, "$canDbDeadline=$canCreateDbDeadline($canDomain,{staleHandle:%s,dbFault:%s,dispatchedWork:%s,dispatchRecord:%s,deadlineRecord:%s,driverSettlementFacts:%s,serverAckFacts:%s,cancelGrant:%s,cancelGrantFacts:%s,cancelRecord:%s,fenceRecord:%s,leaseRecord:%s,deadlineReleaseRecord:%s},$canTest.owner);\n",
		quote(builder.numberIDs["can.std.test@1::stale_handle"]),
		quote(builder.numberIDs["can.std.db@1::db_fault"]),
		quote(builder.numberIDs["can.std.db@1::dispatched_work"]),
		quote(builder.numberIDs["can.std.db@1::dispatch_record"]),
		quote(builder.numberIDs["can.std.db@1::deadline_record"]),
		quote(builder.numberIDs["can.std.db@1::driver_settlement_facts"]),
		quote(builder.numberIDs["can.std.db@1::server_ack_facts"]),
		quote(builder.numberIDs["can.std.db@1::cancel_grant"]),
		quote(builder.numberIDs["can.std.db@1::cancel_grant_facts"]),
		quote(builder.numberIDs["can.std.db@1::cancel_record"]),
		quote(builder.numberIDs["can.std.db@1::fence_record"]),
		quote(builder.numberIDs["can.std.db@1::lease_record"]),
		quote(builder.numberIDs["can.std.db@1::deadline_release_record"]))
}
