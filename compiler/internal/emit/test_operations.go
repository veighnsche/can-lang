package emit

import "fmt"

// testOperationBindings maps the NT-P28 base test owner/channel operations
// to the single $canTest contribution: grant admission and owner release
// on $canTest.owner, channel open/send/receive/close/pending on
// $canTest.transport. Both adapters share one owner table, wired once in
// the shared state module; the JSON value codec frames envelopes.
func testOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.test@1::grant_admit":     "$canTest.owner.admitGrant",
		"can.std.test@1::grant_release":   "$canTest.owner.releaseGrant",
		"can.std.test@1::channel_open":    "$canTest.transport.openChannel",
		"can.std.test@1::channel_send":    "$canTest.transport.sendEnvelope",
		"can.std.test@1::channel_recv":    "$canTest.transport.recvEnvelope",
		"can.std.test@1::channel_close":   "$canTest.transport.closeChannel",
		"can.std.test@1::channel_pending": "$canTest.transport.pendingDepth",
	}
	return bindingContribution{domain: "test", functions: functions}
}

// testStateImports lists the test-support factory modules the shared state
// module needs. The composed createTestSupport factory is the only import:
// owner and channel adapters arrive through it.
func (assembly *programAssembly) testStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/test-support/transport.ts", Names: []ImportName{{"createTestSupport", "$canCreateTestSupport"}}},
	}
}

// testStateValueImportNames lists the test factory value authored and
// assertion modules import from the state module.
func testStateValueImportNames() []ImportName {
	return []ImportName{{"$canTest", "$canTest"}}
}

// declareTestState emits the test factory binding.
func (builder *stateBuilder) declareTestState() {
	builder.out.WriteString("export let $canTest:ReturnType<typeof $canCreateTestSupport>;\n")
}

// initializeTestState constructs the test factory inside the shared
// initializer, after the domain runtime and the JSON value codec exist:
// the transport frames envelopes through $canJSONValue and guards decoded
// frames against the json_object nominal identity.
func (builder *stateBuilder) initializeTestState() {
	fmt.Fprintf(&builder.out, "$canTest=$canCreateTestSupport($canDomain,{invalidGrant:%s,staleHandle:%s,closedHandle:%s,transportFailure:%s,channelFull:%s,detachedTransport:%s,channelEmpty:%s,invalidKind:%s},$canJSONValue,%s);\n", quote(builder.numberIDs["can.std.test@1::invalid_grant"]), quote(builder.numberIDs["can.std.test@1::stale_handle"]), quote(builder.numberIDs["can.std.test@1::closed_handle"]), quote(builder.numberIDs["can.std.test@1::transport_failure"]), quote(builder.numberIDs["can.std.test@1::channel_full"]), quote(builder.numberIDs["can.std.test@1::detached_transport"]), quote(builder.numberIDs["can.std.test@1::channel_empty"]), quote(builder.numberIDs["can.std.test@1::invalid_kind"]), quote(builder.numberIDs["can.std.codec@1::json_object"]))
}
