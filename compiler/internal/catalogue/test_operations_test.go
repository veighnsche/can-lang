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
