package catalogue

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// TestNarrowLookupExhaustiveEquality pins every K1 lookup to the former
// whole-inventory-scan oracle: each returned value must be identical to
// the matching inventory element, receiver methods must keep inventory
// order, and CurrentOperation must equal an explicitly pinned Operation.
func TestNarrowLookupExhaustiveEquality(t *testing.T) {
	c := Builtin()
	inv := c.Inventory()
	if len(inv.Operations) == 0 || len(inv.Types) == 0 || len(inv.Errors) == 0 {
		t.Fatal("embedded inventory is missing declarations")
	}
	for _, want := range inv.Operations {
		got, ok := c.OperationByIdentity(want.Identity)
		if !ok {
			t.Fatalf("OperationByIdentity(%q) declined", want.Identity)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("OperationByIdentity(%q) differs from inventory oracle", want.Identity)
		}
		current, err := c.CurrentOperation(want.Name)
		if err != nil {
			t.Fatalf("CurrentOperation(%q): %v", want.Name, err)
		}
		if !reflect.DeepEqual(current, want) {
			t.Fatalf("CurrentOperation(%q) differs from inventory oracle", want.Name)
		}
		pinned, err := c.Operation(want.Name, inv.TargetID, inv.Revision)
		if err != nil || !reflect.DeepEqual(pinned, current) {
			t.Fatalf("CurrentOperation(%q) differs from pinned Operation", want.Name)
		}
	}
	for _, want := range inv.Types {
		got, ok := c.TypeByIdentity(want.Identity)
		if !ok {
			t.Fatalf("TypeByIdentity(%q) declined", want.Identity)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("TypeByIdentity(%q) differs from inventory oracle", want.Identity)
		}
	}
	for _, want := range inv.Errors {
		got, ok := c.ErrorByIdentity(want.Identity)
		if !ok {
			t.Fatalf("ErrorByIdentity(%q) declined", want.Identity)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ErrorByIdentity(%q) differs from inventory oracle", want.Identity)
		}
	}
	if got := c.Errors(); !reflect.DeepEqual(got, inv.Errors) {
		t.Fatal("Errors() differs from inventory oracle")
	}
	// Receiver methods must match the old scan: kind=method operations
	// whose receiver names a catalogue type, in inventory order.
	receiverIdentities := map[string]string{}
	for _, typ := range inv.Types {
		receiverIdentities[typ.Name] = typ.Identity
	}
	covered := 0
	for _, typ := range inv.Types {
		var want []Operation
		for _, op := range inv.Operations {
			if op.Kind != "method" || op.Receiver == "" {
				continue
			}
			if receiverIdentities[op.Receiver] != typ.Identity {
				continue
			}
			want = append(want, op)
		}
		got := c.ReceiverMethods(typ.Identity)
		if len(want) == 0 {
			if len(got) != 0 {
				t.Fatalf("ReceiverMethods(%q) admitted %d operations", typ.Identity, len(got))
			}
			continue
		}
		covered++
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ReceiverMethods(%q) differs from inventory oracle", typ.Identity)
		}
	}
	if covered == 0 {
		t.Fatal("no receiver methods in embedded inventory")
	}
}

// TestNarrowLookupRejections pins negative behavior: absent and near
// identities decline, explicit pins still reject, and CurrentOperation
// reports unknown names exactly like Operation.
func TestNarrowLookupRejections(t *testing.T) {
	c := Builtin()
	inv := c.Inventory()
	near := []string{
		"",
		"can.std.text@1::from_int ",
		" can.std.text@1::from_int",
		"CAN.STD.TEXT@1::FROM_INT",
		"can.std.text@2::from_int",
		"can.std.text@1::from_in",
		"can.std.text@1::from_intx",
		"can.std.text@1:from_int",
		"can.std.text@1",
		"can.std.text@10::from_int",
		"can.project.text@1::from_int",
		"can.intrinsic.str@1::includes ",
		"can.std.collections@1::does_not_exist",
	}
	for _, identity := range near {
		if op, ok := c.OperationByIdentity(identity); ok {
			t.Fatalf("OperationByIdentity(%q) admitted %s", identity, op.Name)
		}
		if _, ok := c.TypeByIdentity(identity); ok {
			t.Fatalf("TypeByIdentity(%q) admitted", identity)
		}
		if _, ok := c.ErrorByIdentity(identity); ok {
			t.Fatalf("ErrorByIdentity(%q) admitted", identity)
		}
		if got := c.ReceiverMethods(identity); len(got) != 0 {
			t.Fatalf("ReceiverMethods(%q) admitted %d operations", identity, len(got))
		}
	}
	// Known names with wrong pins still reject through Operation, while
	// CurrentOperation derives this instance's pins.
	name := inv.Operations[0].Name
	if _, err := c.Operation(name, "wrong-target", inv.Revision); err == nil {
		t.Fatal("accepted wrong explicit target")
	}
	if _, err := c.Operation(name, inv.TargetID, inv.Revision+1); err == nil {
		t.Fatal("accepted wrong explicit revision")
	}
	if _, err := c.CurrentOperation(name); err != nil {
		t.Fatalf("CurrentOperation(%q): %v", name, err)
	}
	_, wantErr := c.Operation("no::such_operation", inv.TargetID, inv.Revision)
	_, gotErr := c.CurrentOperation("no::such_operation")
	if wantErr == nil || gotErr == nil || wantErr.Error() != gotErr.Error() {
		t.Fatalf("CurrentOperation unknown-name error differs: %v vs %v", gotErr, wantErr)
	}
}

// TestNarrowLookupMutationIsolation pins deep-copy ownership: mutating any
// returned nested value must not affect later lookups or the inventory.
func TestNarrowLookupMutationIsolation(t *testing.T) {
	c := Builtin()
	inv := c.Inventory()
	originalOp := inv.Operations[0]
	op, _ := c.OperationByIdentity(originalOp.Identity)
	op.Result = "mutated::type"
	if len(op.Inputs) > 0 {
		op.Inputs[0].Type = "mutated::nested"
	}
	if len(op.Parameters) > 0 {
		op.Parameters[0].Constraint = "mutated"
	}
	if len(op.Callbacks) > 0 {
		op.Callbacks[0].Result = "mutated::callback"
	}
	if len(op.Emits) > 0 {
		op.Emits[0] = "mutated::error"
	}
	op.Lowering.Task = "mutated"
	again, ok := c.OperationByIdentity(originalOp.Identity)
	if !ok || !reflect.DeepEqual(again, originalOp) {
		t.Fatal("OperationByIdentity result aliases embedded inventory")
	}
	originalType := inv.Types[0]
	typ, _ := c.TypeByIdentity(originalType.Identity)
	if len(typ.Fields) > 0 {
		typ.Fields[0].Type = "mutated::nested"
	}
	if len(typ.Parameters) > 0 {
		typ.Parameters[0].Constraint = "mutated"
	}
	if len(typ.Leaves) > 0 {
		typ.Leaves[0] = "mutated::leaf"
	}
	if len(typ.Projections) > 0 {
		typ.Projections[0].Type = "mutated::projection"
	}
	typ.Constructible = !typ.Constructible
	if again, ok := c.TypeByIdentity(originalType.Identity); !ok || !reflect.DeepEqual(again, originalType) {
		t.Fatal("TypeByIdentity result aliases embedded inventory")
	}
	originalErr := inv.Errors[0]
	decl, _ := c.ErrorByIdentity(originalErr.Identity)
	if len(decl.Fields) > 0 {
		decl.Fields[0].Type = "mutated::nested"
	}
	if again, ok := c.ErrorByIdentity(originalErr.Identity); !ok || !reflect.DeepEqual(again, originalErr) {
		t.Fatal("ErrorByIdentity result aliases embedded inventory")
	}
	errs := c.Errors()
	errs[0].Name = "mutated::error"
	if len(errs[0].Fields) > 0 {
		errs[0].Fields[0].Type = "mutated::nested"
	}
	if again := c.Errors(); !reflect.DeepEqual(again, inv.Errors) {
		t.Fatal("Errors() result aliases embedded inventory")
	}
	var receiver string
	for _, typ := range inv.Types {
		if len(c.ReceiverMethods(typ.Identity)) > 0 {
			receiver = typ.Identity
			break
		}
	}
	if receiver == "" {
		t.Fatal("no receiver methods for mutation isolation")
	}
	methods := c.ReceiverMethods(receiver)
	before := append([]Operation(nil), methods...)
	methods[0].Result = "mutated::type"
	if again := c.ReceiverMethods(receiver); !reflect.DeepEqual(again, before) {
		t.Fatal("ReceiverMethods result aliases embedded inventory")
	}
	if fresh := c.Inventory(); !reflect.DeepEqual(fresh, inv) {
		t.Fatal("narrow-lookup mutation leaked into inventory")
	}
}

// TestNarrowLookupLoadValidation pins that indexes are built by the
// private loader: a freshly loaded catalogue serves identical lookups,
// and an inventory with a retargeted identity still fails validation.
func TestNarrowLookupLoadValidation(t *testing.T) {
	fresh, err := load(source)
	if err != nil {
		t.Fatal(err)
	}
	c := Builtin()
	inv := c.Inventory()
	op, _ := fresh.OperationByIdentity(inv.Operations[0].Identity)
	want, _ := c.OperationByIdentity(inv.Operations[0].Identity)
	if !reflect.DeepEqual(op, want) {
		t.Fatal("freshly loaded catalogue serves different identity lookups")
	}
	if _, err := fresh.CurrentOperation(inv.Operations[0].Name); err != nil {
		t.Fatal(err)
	}
	bad := c.Inventory()
	bad.Operations[0].Identity = bad.Operations[1].Identity
	raw, err := json.Marshal(bad)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := load(raw); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
		t.Fatalf("retargeted identity accepted: %v", err)
	}
}
