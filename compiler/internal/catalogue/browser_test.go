package catalogue

import (
	"reflect"
	"testing"
)

// TestBrowserCatalogueContract pins the T22 bounded browser catalogue: the
// opaque app/view/node/state handles, the event/snapshot data projections,
// the missing-root/disposed/rejected/stale failures, and the exact operation
// bounds. Static literal admission lives in the checker; the runtime
// re-checks every dynamic name.
func TestBrowserCatalogueContract(t *testing.T) {
	c := Builtin()
	for _, name := range []string{"browser::app", "browser::view", "browser::node"} {
		decl, ok := c.Type(name)
		if !ok || decl.Kind != "opaque" || decl.Identity != "can.std.browser@1::"+name[len("browser::"):] || len(decl.Parameters) != 0 {
			t.Fatalf("browser handle %s lost: %+v %v", name, decl, ok)
		}
		if err := c.CheckConstructor(name); err == nil {
			t.Fatalf("public constructor for %s", name)
		}
	}
	state, ok := c.Type("browser::state")
	if !ok || state.Kind != "opaque" || len(state.Parameters) != 1 || state.Parameters[0].Name != "T" || state.Parameters[0].Constraint != "data" {
		t.Fatalf("browser::state declaration lost: %+v %v", state, ok)
	}
	snapshot, ok := c.Type("browser::snapshot")
	if !ok || snapshot.Kind != "record" {
		t.Fatalf("browser::snapshot declaration lost: %+v %v", snapshot, ok)
	}
	if len(snapshot.Parameters) != 1 || len(snapshot.Fields) != 2 || snapshot.Fields[0].Name != "version" || snapshot.Fields[0].Type != "int" || snapshot.Fields[1].Name != "value" || snapshot.Fields[1].Type != "T" {
		t.Fatalf("browser::snapshot shape differs: %+v", snapshot.Fields)
	}
	event, ok := c.Type("browser::event")
	if !ok || event.Kind != "record" {
		t.Fatalf("browser::event declaration lost: %+v %v", event, ok)
	}
	wantFields := []Field{{Name: "kind", Type: "str"}, {Name: "target", Type: "str"}, {Name: "value", Type: "str"}, {Name: "key", Type: "str"}}
	if !reflect.DeepEqual(event.Fields, wantFields) {
		t.Fatalf("browser::event fields differ: %+v", event.Fields)
	}
	stale, ok := c.Error("browser::stale_version")
	if !ok || len(stale.Fields) != 2 || stale.Fields[0].Name != "expected" || stale.Fields[0].Type != "int" || stale.Fields[1].Name != "actual" || stale.Fields[1].Type != "int" {
		t.Fatalf("browser::stale_version payload differs: %+v %v", stale, ok)
	}
	missing, ok := c.Error("browser::missing_root")
	if !ok || len(missing.Fields) != 1 || missing.Fields[0].Name != "root" || missing.Fields[0].Type != "str" {
		t.Fatalf("browser::missing_root payload differs: %+v %v", missing, ok)
	}
	for _, name := range []string{"browser::disposed", "browser::rejected"} {
		if _, ok := c.Error(name); !ok {
			t.Fatalf("browser error %s missing", name)
		}
	}
	listen, err := c.Operation("browser::on_event", c.Inventory().TargetID, c.Inventory().Revision)
	if err != nil {
		t.Fatal(err)
	}
	if len(listen.Inputs) != 4 || listen.Inputs[3].Type != "$callback" || len(listen.Callbacks) != 1 {
		t.Fatalf("browser::on_event inputs differ: %+v", listen.Inputs)
	}
	callback := listen.Callbacks[0]
	if callback.Name != "callback" || !reflect.DeepEqual(callback.Inputs, []string{"browser::event"}) || callback.Result != "void" || callback.DeriveErrors || len(callback.Emits) != 0 {
		t.Fatalf("browser::on_event callback differs: %+v", callback)
	}
	if !reflect.DeepEqual(listen.Emits, []string{"browser::disposed", "browser::rejected"}) {
		t.Fatalf("browser::on_event bound differs: %+v", listen.Emits)
	}
	timer, err := c.Operation("browser::set_timeout", c.Inventory().TargetID, c.Inventory().Revision)
	if err != nil {
		t.Fatal(err)
	}
	if len(timer.Callbacks) != 1 || len(timer.Callbacks[0].Inputs) != 0 || timer.Callbacks[0].Result != "void" {
		t.Fatalf("browser::set_timeout callback differs: %+v", timer.Callbacks)
	}
	read, err := c.Operation("browser::read_state", c.Inventory().TargetID, c.Inventory().Revision)
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Parameters) != 1 || read.Result != "browser::snapshot<T>" || !reflect.DeepEqual(read.Emits, []string{"browser::disposed"}) {
		t.Fatalf("browser::read_state descriptor differs: %+v", read)
	}
	spec, err := c.Resolve("browser::read_state", c.Inventory().TargetID, c.Inventory().Revision, map[string]string{"T": "str"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Operation.Result != "browser::snapshot<str>" || len(spec.Emits) != 1 {
		t.Fatalf("browser::read_state specialization differs: %+v", spec.Operation)
	}
	for _, name := range []string{"browser::mount", "browser::root", "browser::open_view", "browser::dispose_view", "browser::dispose_app", "browser::create_element", "browser::create_text", "browser::set_text", "browser::set_attribute", "browser::remove_attribute", "browser::append_child", "browser::remove_node", "browser::focus", "browser::on_event", "browser::set_timeout", "browser::create_state", "browser::read_state", "browser::replace_state"} {
		op, err := c.Operation(name, c.Inventory().TargetID, c.Inventory().Revision)
		if err != nil {
			t.Fatalf("browser operation %s missing: %v", name, err)
		}
		if op.Lowering.Task != "T22" || op.Assertion != "real" {
			t.Fatalf("browser operation %s evidence contract differs: %+v", name, op.Lowering)
		}
	}
}
