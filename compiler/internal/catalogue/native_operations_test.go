package catalogue

import (
	"reflect"
	"testing"
)

// TestNativeCatalogueContract pins the NT-I01 native-values surface: 11
// operations mirroring the K01/K02/K03/K06 test services over 5 opaque
// handles, constructible input/result records, and merge-defined contracts
// for the service methods that do not exist yet (describe/invoke/settle
// results, gate/fault shapes). Closed sets are inline str inputs enforced
// by NT-I01 literal allowlists, never distinct types. The emitter binds the
// ambient test owner as owner_grant; service errors surface as standard
// occurrences, so no op declares emits.
func TestNativeCatalogueContract(t *testing.T) {
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
	for _, name := range []string{"native::session", "native::value_handle", "native::pending_action", "native::gate", "native::fault"} {
		handle, ok := c.Type(name)
		if !ok || handle.Kind != "opaque" || handle.Constructible || len(handle.Projections) != 0 {
			t.Fatalf("%s marker differs: %+v", name, handle)
		}
		if err := c.CheckConstructor(name); err == nil {
			t.Fatalf("%s admits a public constructor", name)
		}
	}
	if err := c.CheckProjectPackage("native"); err == nil {
		t.Fatal("native package is not distribution-owned")
	}
	// Limits use the TS wire keys verbatim (mixed snake/camel); checkLimits
	// rejects unknown fields, so a normalized spelling would fail at runtime.
	limits, ok := c.Type("native::limits")
	if !ok {
		t.Fatal("native::limits missing")
	}
	fields := map[string]string{}
	for _, field := range limits.Fields {
		fields[field.Name] = field.Type
	}
	for _, name := range []string{"max_sessions", "max_handles_per_session", "maxPendingActions", "maxObserveEntries", "maxObserveBytes", "maxGatesPerSession", "maxFaultsPerSession"} {
		if fields[name] != "int" {
			t.Fatalf("native::limits field %s differs: %+v", name, limits.Fields)
		}
	}
	literal, ok := c.Type("native::literal_value")
	if !ok || literal.Kind != "variant" || !reflect.DeepEqual(literal.Leaves, []string{"native::literal_int", "native::literal_text", "native::literal_bool"}) {
		t.Fatalf("native::literal_value differs: %+v", literal)
	}
	payload, ok := c.Type("native::inert_literal")
	if !ok {
		t.Fatal("native::inert_literal missing")
	}
	payloadFields := map[string]string{}
	for _, field := range payload.Fields {
		payloadFields[field.Name] = field.Type
	}
	if payloadFields["tag"] != "str" || payloadFields["entries"] != "option::value<native::entry[]>" || payloadFields["descriptor"] != "option::value<str>" {
		t.Fatalf("native::inert_literal fields differ: %+v", payload.Fields)
	}
	type descriptor struct {
		result string
		inputs []string
		refs   []string
	}
	want := map[string]descriptor{
		"native::open":          {"native::session", []string{"test::owner", "str", "str", "str", "native::limits"}, []string{"N1"}},
		"native::describe":      {"native::descriptor_or_unknown", []string{"native::session", "str"}, []string{"N1"}},
		"native::make":          {"native::value_handle", []string{"native::session", "str", "native::inert_literal"}, []string{"N1"}},
		"native::invoke":        {"native::handle_or_pending_action", []string{"native::session", "str", "option::value<native::value_handle>", "native::value_handle[]"}, []string{"N1", "N2"}},
		"native::settle":        {"native::settlement_or_pending", []string{"native::pending_action", "native::deadline"}, []string{"N1"}},
		"native::observe":       {"native::inert_facts", []string{"native::session", "native::value_handle[]", "str", "native::observe_bounds"}, []string{"N1"}},
		"native::allocate_gate": {"native::gate", []string{"native::session", "native::gate_spec", "native::bounded_action_ref"}, []string{"N3"}},
		"native::release":       {"native::release_facts", []string{"native::gate"}, []string{"N3"}},
		"native::install_fault": {"native::fault", []string{"native::session", "str", "str"}, []string{"N1", "N3"}},
		"native::restore":       {"native::restore_outcome", []string{"native::fault"}, []string{"N1", "N3"}},
		"native::close":         {"native::close_receipt", []string{"native::session", "native::deadline"}, []string{"N1", "N2", "N3"}},
	}
	if len(want) != 11 {
		t.Fatalf("contract pins %d operations, want 11", len(want))
	}
	for name, descriptor := range want {
		op := operation(name)
		if op.Result != descriptor.result || len(op.Inputs) != len(descriptor.inputs) {
			t.Fatalf("%s descriptor differs: %+v", name, op)
		}
		for i, input := range descriptor.inputs {
			if op.Inputs[i].Type != input {
				t.Fatalf("%s input %d differs: %+v", name, i, op.Inputs)
			}
		}
		if len(op.Emits) != 0 {
			t.Fatalf("%s declares emits: %+v", name, op.Emits)
		}
		if op.Lowering.Task != "NT-I01" || !reflect.DeepEqual(op.Refs, descriptor.refs) || op.Assertion != "supplied" {
			t.Fatalf("%s traceability differs: task=%q refs=%v assertion=%q", name, op.Lowering.Task, op.Refs, op.Assertion)
		}
		if len(op.Lowering.Native) == 0 || op.Lowering.Adapter == "" {
			t.Fatalf("%s names no native recipe", name)
		}
	}
}
