package catalogue

import (
	"reflect"
	"testing"
)

// TestOwnerTransportCatalogueContract pins the NT-P28 base owner/channel
// surface: N-issued grant admission, owner-bound channel open, framed JSON
// envelope send/receive/close/pending. Handles are opaque worker-local
// bindings, never the grants or bytes; the catalogue carries the shapes
// and the finite failure vocabulary, while issuance is verified by N at
// dispatch and completion discipline lives in the emitter.
func TestOwnerTransportCatalogueContract(t *testing.T) {
	c := Builtin()
	target, revision := c.Inventory().TargetID, c.Inventory().Revision
	operation := func(name string) Operation {
		t.Helper()
		op, err := c.Operation(name, target, revision)
		if err != nil {
			t.Fatal(err)
		}
		return op
	}
	owner, ok := c.Type("test::owner")
	if !ok || owner.Kind != "opaque" || owner.Constructible || len(owner.Projections) != 0 {
		t.Fatalf("test::owner marker differs: %+v", owner)
	}
	channel, ok := c.Type("test::channel")
	if !ok || channel.Kind != "opaque" || channel.Constructible || len(channel.Projections) != 1 ||
		channel.Projections[0].Name != "kind" || channel.Projections[0].Type != "str" {
		t.Fatalf("test::channel marker differs: %+v", channel)
	}
	if err := c.CheckConstructor("test::owner"); err == nil {
		t.Fatal("test::owner admits a public constructor")
	}
	if err := c.CheckConstructor("test::channel"); err == nil {
		t.Fatal("test::channel admits a public constructor")
	}
	if err := c.CheckProjectPackage("test"); err == nil {
		t.Fatal("test package is not distribution-owned")
	}
	admit := operation("test::grant_admit")
	if admit.Result != "test::owner" || len(admit.Inputs) != 1 || admit.Inputs[0].Type != "str" {
		t.Fatalf("test::grant_admit descriptor differs: %+v", admit)
	}
	if !reflect.DeepEqual(admit.Emits, []string{"test::invalid_grant"}) {
		t.Fatalf("test::grant_admit bound differs: %+v", admit.Emits)
	}
	release := operation("test::grant_release")
	if release.Result != "void" || len(release.Inputs) != 1 || release.Inputs[0].Type != "test::owner" || len(release.Emits) != 0 {
		t.Fatalf("test::grant_release descriptor differs: %+v", release)
	}
	open := operation("test::channel_open")
	if open.Result != "test::channel" || len(open.Inputs) != 2 || open.Inputs[0].Type != "test::owner" || open.Inputs[1].Type != "str" {
		t.Fatalf("test::channel_open descriptor differs: %+v", open)
	}
	if !reflect.DeepEqual(open.Emits, []string{"test::stale_handle", "test::invalid_kind"}) {
		t.Fatalf("test::channel_open bound differs: %+v", open.Emits)
	}
	send := operation("test::channel_send")
	if send.Result != "int" || len(send.Inputs) != 2 || send.Inputs[0].Type != "test::channel" || send.Inputs[1].Type != "codec::json_object" {
		t.Fatalf("test::channel_send descriptor differs: %+v", send)
	}
	// test::wrong_owner stays vocabulary-only until the N-link slice wires
	// live issuance verification; the base adapter cannot verify issuance.
	if !reflect.DeepEqual(send.Emits, []string{"test::stale_handle", "test::closed_handle", "test::channel_full", "test::detached_transport", "test::transport_failure"}) {
		t.Fatalf("test::channel_send bound differs: %+v", send.Emits)
	}
	recv := operation("test::channel_recv")
	if recv.Result != "codec::json_object" || len(recv.Inputs) != 1 || recv.Inputs[0].Type != "test::channel" {
		t.Fatalf("test::channel_recv descriptor differs: %+v", recv)
	}
	if !reflect.DeepEqual(recv.Emits, []string{"test::stale_handle", "test::closed_handle", "test::detached_transport", "test::channel_empty", "test::transport_failure"}) {
		t.Fatalf("test::channel_recv bound differs: %+v", recv.Emits)
	}
	closeOp := operation("test::channel_close")
	if closeOp.Result != "void" || len(closeOp.Inputs) != 1 || closeOp.Inputs[0].Type != "test::channel" || len(closeOp.Emits) != 0 {
		t.Fatalf("test::channel_close descriptor differs: %+v", closeOp)
	}
	pending := operation("test::channel_pending")
	if pending.Result != "int" || len(pending.Inputs) != 1 || pending.Inputs[0].Type != "test::channel" {
		t.Fatalf("test::channel_pending descriptor differs: %+v", pending)
	}
	if !reflect.DeepEqual(pending.Emits, []string{"test::stale_handle"}) {
		t.Fatalf("test::channel_pending bound differs: %+v", pending.Emits)
	}
	for _, name := range []string{"test::grant_admit", "test::grant_release", "test::channel_open", "test::channel_send", "test::channel_recv", "test::channel_close", "test::channel_pending"} {
		op := operation(name)
		if op.Lowering.Task != "NT-P28" || !reflect.DeepEqual(op.Refs, []string{"NT-P28"}) || op.Assertion != "real" {
			t.Fatalf("%s traceability differs: task=%q refs=%v assertion=%q", name, op.Lowering.Task, op.Refs, op.Assertion)
		}
		if len(op.Lowering.Native) == 0 || op.Lowering.Adapter == "" {
			t.Fatalf("%s names no native recipe", name)
		}
	}
}

// TestWorkspaceProcessEvidenceCatalogueContract pins the NT-P28 S1c
// surface: owner-scoped workspace directories with relative-path file
// operations, keyed tool dispatch (no paths in calls), and sealed
// evidence bundles with closed-vocabulary receipt kinds. Handles stay
// opaque and non-constructible; results and receipts are transparent
// records mirroring process::result.
func TestWorkspaceProcessEvidenceCatalogueContract(t *testing.T) {
	c := Builtin()
	target, revision := c.Inventory().TargetID, c.Inventory().Revision
	operation := func(name string) Operation {
		t.Helper()
		op, err := c.Operation(name, target, revision)
		if err != nil {
			t.Fatal(err)
		}
		return op
	}
	workspace, ok := c.Type("test::workspace")
	if !ok || workspace.Kind != "opaque" || workspace.Constructible || len(workspace.Projections) != 0 {
		t.Fatalf("test::workspace marker differs: %+v", workspace)
	}
	evidence, ok := c.Type("test::evidence")
	if !ok || evidence.Kind != "opaque" || evidence.Constructible || len(evidence.Projections) != 0 {
		t.Fatalf("test::evidence marker differs: %+v", evidence)
	}
	if err := c.CheckConstructor("test::workspace"); err == nil {
		t.Fatal("test::workspace admits a public constructor")
	}
	if err := c.CheckConstructor("test::evidence"); err == nil {
		t.Fatal("test::evidence admits a public constructor")
	}
	toolResult, ok := c.Type("test::tool_result")
	if !ok || toolResult.Kind != "record" || !toolResult.Constructible {
		t.Fatalf("test::tool_result marker differs: %+v", toolResult)
	}
	receipt, ok := c.Type("test::evidence_receipt")
	if !ok || receipt.Kind != "record" || !receipt.Constructible {
		t.Fatalf("test::evidence_receipt marker differs: %+v", receipt)
	}
	wsOpen := operation("test::workspace_open")
	if wsOpen.Result != "test::workspace" || len(wsOpen.Inputs) != 1 || wsOpen.Inputs[0].Type != "test::owner" {
		t.Fatalf("test::workspace_open descriptor differs: %+v", wsOpen)
	}
	if !reflect.DeepEqual(wsOpen.Emits, []string{"test::stale_handle"}) {
		t.Fatalf("test::workspace_open bound differs: %+v", wsOpen.Emits)
	}
	mkdir := operation("test::workspace_mkdir")
	if mkdir.Result != "void" || len(mkdir.Inputs) != 3 || mkdir.Inputs[0].Type != "test::workspace" || mkdir.Inputs[1].Type != "str" || mkdir.Inputs[2].Type != "bool" {
		t.Fatalf("test::workspace_mkdir descriptor differs: %+v", mkdir)
	}
	if !reflect.DeepEqual(mkdir.Emits, []string{"test::stale_handle", "test::closed_handle", "test::invalid_path", "files::not_found", "files::already_exists", "files::denied", "files::io_error"}) {
		t.Fatalf("test::workspace_mkdir bound differs: %+v", mkdir.Emits)
	}
	write := operation("test::workspace_write_text")
	if write.Result != "void" || len(write.Inputs) != 4 || write.Inputs[0].Type != "test::workspace" || write.Inputs[1].Type != "str" || write.Inputs[2].Type != "str" || write.Inputs[3].Type != "bool" {
		t.Fatalf("test::workspace_write_text descriptor differs: %+v", write)
	}
	if !reflect.DeepEqual(write.Emits, []string{"test::stale_handle", "test::closed_handle", "test::invalid_path", "files::not_found", "files::already_exists", "files::denied", "files::io_error"}) {
		t.Fatalf("test::workspace_write_text bound differs: %+v", write.Emits)
	}
	read := operation("test::workspace_read_text")
	if read.Result != "str" || len(read.Inputs) != 3 || read.Inputs[0].Type != "test::workspace" || read.Inputs[1].Type != "str" || read.Inputs[2].Type != "int" {
		t.Fatalf("test::workspace_read_text descriptor differs: %+v", read)
	}
	if !reflect.DeepEqual(read.Emits, []string{"test::stale_handle", "test::closed_handle", "test::invalid_path", "files::not_found", "files::denied", "files::unexpected_kind", "files::limit_exceeded", "codec::invalid_data", "files::io_error"}) {
		t.Fatalf("test::workspace_read_text bound differs: %+v", read.Emits)
	}
	wsClose := operation("test::workspace_close")
	if wsClose.Result != "void" || len(wsClose.Inputs) != 1 || wsClose.Inputs[0].Type != "test::workspace" || len(wsClose.Emits) != 0 {
		t.Fatalf("test::workspace_close descriptor differs: %+v", wsClose)
	}
	runTool := operation("test::run_tool")
	if runTool.Result != "test::tool_result" || len(runTool.Inputs) != 4 || runTool.Inputs[0].Type != "test::owner" || runTool.Inputs[1].Type != "str" || runTool.Inputs[2].Type != "str[]" || runTool.Inputs[3].Type != "process::options" {
		t.Fatalf("test::run_tool descriptor differs: %+v", runTool)
	}
	if !reflect.DeepEqual(runTool.Emits, []string{"test::stale_handle", "test::unknown_tool", "process::spawn_failed", "process::timeout", "process::output_limit", "process::io_error", "process::invalid_config", "files::not_found", "files::denied"}) {
		t.Fatalf("test::run_tool bound differs: %+v", runTool.Emits)
	}
	evOpen := operation("test::evidence_open")
	if evOpen.Result != "test::evidence" || len(evOpen.Inputs) != 1 || evOpen.Inputs[0].Type != "test::owner" {
		t.Fatalf("test::evidence_open descriptor differs: %+v", evOpen)
	}
	if !reflect.DeepEqual(evOpen.Emits, []string{"test::stale_handle"}) {
		t.Fatalf("test::evidence_open bound differs: %+v", evOpen.Emits)
	}
	append := operation("test::evidence_append")
	if append.Result != "void" || len(append.Inputs) != 3 || append.Inputs[0].Type != "test::evidence" || append.Inputs[1].Type != "str" || append.Inputs[2].Type != "bytes::buffer" {
		t.Fatalf("test::evidence_append descriptor differs: %+v", append)
	}
	if !reflect.DeepEqual(append.Emits, []string{"test::stale_handle", "test::closed_handle", "test::invalid_name"}) {
		t.Fatalf("test::evidence_append bound differs: %+v", append.Emits)
	}
	seal := operation("test::evidence_seal")
	if seal.Result != "test::evidence_receipt" || len(seal.Inputs) != 2 || seal.Inputs[0].Type != "test::evidence" || seal.Inputs[1].Type != "str" {
		t.Fatalf("test::evidence_seal descriptor differs: %+v", seal)
	}
	if !reflect.DeepEqual(seal.Emits, []string{"test::stale_handle", "test::closed_handle", "test::invalid_kind"}) {
		t.Fatalf("test::evidence_seal bound differs: %+v", seal.Emits)
	}
	for _, name := range []string{"test::workspace_open", "test::workspace_mkdir", "test::workspace_write_text", "test::workspace_read_text", "test::workspace_close", "test::run_tool", "test::evidence_open", "test::evidence_append", "test::evidence_seal"} {
		op := operation(name)
		if op.Lowering.Task != "NT-P28" || !reflect.DeepEqual(op.Refs, []string{"NT-P28"}) || op.Assertion != "real" {
			t.Fatalf("%s traceability differs: task=%q refs=%v assertion=%q", name, op.Lowering.Task, op.Refs, op.Assertion)
		}
		if len(op.Lowering.Native) == 0 || op.Lowering.Adapter == "" {
			t.Fatalf("%s names no native recipe", name)
		}
	}
}
