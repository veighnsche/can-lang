package emit

import "fmt"

// testOperationBindings maps the NT-P28 test operations to the single
// $canTest contribution: grant admission and owner release on
// $canTest.owner, channel open/send/receive/close/pending on
// $canTest.transport, workspace open/mkdir/write/read/close on
// $canTest.workspace, keyed tool dispatch on $canTest.tools, and
// evidence open/append/seal on $canTest.evidence. All adapters share
// one owner table, wired once in the shared state module; the JSON
// value codec frames envelopes.
func testOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.test@1::grant_admit":          "$canTest.owner.admitGrant",
		"can.std.test@1::grant_release":        "$canTest.owner.releaseGrant",
		"can.std.test@1::channel_open":         "$canTest.transport.openChannel",
		"can.std.test@1::channel_send":         "$canTest.transport.sendEnvelope",
		"can.std.test@1::channel_recv":         "$canTest.transport.recvEnvelope",
		"can.std.test@1::channel_close":        "$canTest.transport.closeChannel",
		"can.std.test@1::channel_pending":      "$canTest.transport.pendingDepth",
		"can.std.test@1::workspace_open":       "$canTest.workspace.openWorkspace",
		"can.std.test@1::workspace_mkdir":      "$canTest.workspace.workspaceMkdir",
		"can.std.test@1::workspace_write_text": "$canTest.workspace.workspaceWriteText",
		"can.std.test@1::workspace_read_text":  "$canTest.workspace.workspaceReadText",
		"can.std.test@1::workspace_close":      "$canTest.workspace.closeWorkspace",
		"can.std.test@1::run_tool":             "$canTest.tools.runTool",
		"can.std.test@1::evidence_open":        "$canTest.evidence.openEvidence",
		"can.std.test@1::evidence_append":      "$canTest.evidence.appendEvidence",
		"can.std.test@1::evidence_seal":        "$canTest.evidence.sealEvidence",
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
// frames against the json_object nominal identity. The tool table starts
// empty: owner-side tool registration arrives with the N-link slice, so
// run_tool rejects every key as unknown_tool until then.
func (builder *stateBuilder) initializeTestState() {
	fmt.Fprintf(&builder.out, "$canTest=$canCreateTestSupport($canDomain,{invalidGrant:%s,staleHandle:%s,closedHandle:%s,transportFailure:%s,channelFull:%s,detachedTransport:%s,channelEmpty:%s,invalidKind:%s,unknownTool:%s,invalidPath:%s,invalidName:%s,notFound:%s,alreadyExists:%s,denied:%s,unexpectedKind:%s,limitExceeded:%s,invalidData:%s,ioError:%s,spawnFailed:%s,timeout:%s,outputLimit:%s,invalidConfig:%s,processIoError:%s,toolResult:%s,receipt:%s},$canJSONValue,%s,new Map());\n", quote(builder.numberIDs["can.std.test@1::invalid_grant"]), quote(builder.numberIDs["can.std.test@1::stale_handle"]), quote(builder.numberIDs["can.std.test@1::closed_handle"]), quote(builder.numberIDs["can.std.test@1::transport_failure"]), quote(builder.numberIDs["can.std.test@1::channel_full"]), quote(builder.numberIDs["can.std.test@1::detached_transport"]), quote(builder.numberIDs["can.std.test@1::channel_empty"]), quote(builder.numberIDs["can.std.test@1::invalid_kind"]), quote(builder.numberIDs["can.std.test@1::unknown_tool"]), quote(builder.numberIDs["can.std.test@1::invalid_path"]), quote(builder.numberIDs["can.std.test@1::invalid_name"]), quote(builder.numberIDs["can.std.files@1::not_found"]), quote(builder.numberIDs["can.std.files@1::already_exists"]), quote(builder.numberIDs["can.std.files@1::denied"]), quote(builder.numberIDs["can.std.files@1::unexpected_kind"]), quote(builder.numberIDs["can.std.files@1::limit_exceeded"]), quote(builder.numberIDs["can.std.codec@1::invalid_data"]), quote(builder.numberIDs["can.std.files@1::io_error"]), quote(builder.numberIDs["can.std.process@1::spawn_failed"]), quote(builder.numberIDs["can.std.process@1::timeout"]), quote(builder.numberIDs["can.std.process@1::output_limit"]), quote(builder.numberIDs["can.std.process@1::invalid_config"]), quote(builder.numberIDs["can.std.process@1::io_error"]), quote(builder.numberIDs["can.std.test@1::tool_result"]), quote(builder.numberIDs["can.std.test@1::evidence_receipt"]), quote(builder.numberIDs["can.std.codec@1::json_object"]))
}
