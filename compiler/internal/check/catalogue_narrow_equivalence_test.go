package check

import (
	"reflect"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
)

// Reference implementations below are verbatim copies of the pre-narrowing
// whole-inventory scans. Each test compares the migrated helper against its
// reference exhaustively over the embedded inventory plus negative cases.

func refStreamOperation(inv catalogue.Inventory, identity string) *catalogue.Operation {
	switch identity {
	case "can.std.stream@1::read_many", "can.std.stream@1::close_reader", "can.std.stream@1::cancel_reader":
	default:
		return nil
	}
	for _, op := range inv.Operations {
		if op.Identity == identity {
			out := op
			return &out
		}
	}
	return nil
}

func refBrowserStateOperation(inv catalogue.Inventory, identity string) *catalogue.Operation {
	if !strings.HasPrefix(identity, "can.std.browser@1::") {
		return nil
	}
	if identity != browserCreateState && identity != browserReadState && identity != browserReplaceState {
		return nil
	}
	for _, op := range inv.Operations {
		if op.Identity == identity && op.Lowering.Task == "T22" {
			operation := op
			return &operation
		}
	}
	return nil
}

func refSQLOperation(inv catalogue.Inventory, identity string) *catalogue.Operation {
	for _, op := range inv.Operations {
		if op.Identity == identity {
			out := op
			return &out
		}
	}
	return nil
}

func checkHelperEquivalence(t *testing.T, name string, ref, got *catalogue.Operation, identity string) {
	t.Helper()
	if ref == nil {
		if got != nil {
			t.Fatalf("%s(%q) admitted %s; reference declines", name, identity, got.Identity)
		}
		return
	}
	if got == nil {
		t.Fatalf("%s(%q) declined; reference admits", name, identity)
	}
	if !reflect.DeepEqual(*got, *ref) {
		t.Fatalf("%s(%q) metadata differs from reference scan", name, identity)
	}
}

func TestNarrowHelperEquivalence(t *testing.T) {
	inv := catalogue.Builtin().Inventory()
	probes := make([]string, 0, len(inv.Operations)+8)
	for _, op := range inv.Operations {
		probes = append(probes, op.Identity)
	}
	probes = append(probes,
		"",
		"can.std.stream@1::does_not_exist",
		"can.std.browser@1::does_not_exist",
		"can.std.sql@1::does_not_exist",
		"can.std.stream@2::read_many",
		"can.std.browser@1::create_state ",
		"can.std.sql@1::query_one ",
		"CAN.STD.SQL@1::QUERY_ONE",
	)
	for _, identity := range probes {
		checkHelperEquivalence(t, "streamOperation", refStreamOperation(inv, identity), streamOperation(identity), identity)
		checkHelperEquivalence(t, "browserStateOperation", refBrowserStateOperation(inv, identity), browserStateOperation(identity), identity)
		checkHelperEquivalence(t, "sqlOperation", refSQLOperation(inv, identity), sqlOperation(identity), identity)
	}
	// Every admitted SQL identity keeps its original task: the helper
	// admits any found identity and invents no namespace restriction.
	for _, op := range inv.Operations {
		if got := sqlOperation(op.Identity); got == nil || got.Lowering.Task != op.Lowering.Task {
			t.Fatalf("sqlOperation(%q) task drift", op.Identity)
		}
	}
}

// TestNarrowHelperMutationIsolation pins that migrated helpers still serve
// deep defensive copies: nested mutation must not leak into later lookups.
func TestNarrowHelperMutationIsolation(t *testing.T) {
	inv := catalogue.Builtin().Inventory()
	cases := map[string]func(string) *catalogue.Operation{
		"streamOperation":       streamOperation,
		"browserStateOperation": browserStateOperation,
		"sqlOperation":          sqlOperation,
	}
	identities := map[string]string{
		"streamOperation":       "can.std.stream@1::read_many",
		"browserStateOperation": browserCreateState,
		"sqlOperation":          "can.std.sql@1::query_one",
	}
	for name, helper := range cases {
		identity := identities[name]
		var original catalogue.Operation
		found := false
		for _, op := range inv.Operations {
			if op.Identity == identity {
				original, found = op, true
				break
			}
		}
		if !found {
			t.Fatalf("%s oracle identity %q missing from inventory", name, identity)
		}
		mutated := helper(identity)
		if mutated == nil {
			t.Fatalf("%s(%q) declined", name, identity)
		}
		mutated.Result = "mutated::type"
		if len(mutated.Inputs) > 0 {
			mutated.Inputs[0].Type = "mutated::nested"
		}
		if again := helper(identity); again == nil || !reflect.DeepEqual(*again, original) {
			t.Fatalf("%s(%q) result aliases embedded inventory", name, identity)
		}
	}
}

// refNominalMethod is the pre-narrowing nominal catalogue-method scan from
// method(), reduced to its catalogue interaction: receiver-identity match,
// local-name match and first-match order.
func refNominalMethod(inv catalogue.Inventory, receiverIdentity, local string) *catalogue.Operation {
	receiverIdentities := map[string]string{}
	for _, typ := range inv.Types {
		receiverIdentities[typ.Name] = typ.Identity
	}
	for _, op := range inv.Operations {
		if op.Kind != "method" || op.Receiver == "" {
			continue
		}
		receiverID, ok := receiverIdentities[op.Receiver]
		if !ok || receiverID != receiverIdentity {
			continue
		}
		name := op.Name
		if i := strings.LastIndex(name, "::"); i >= 0 {
			name = name[i+2:]
		} else if i := strings.LastIndex(name, "."); i >= 0 {
			name = name[i+1:]
		}
		if name != local {
			continue
		}
		out := op
		return &out
	}
	return nil
}

func TestNominalMethodLookupEquivalence(t *testing.T) {
	inv := catalogue.Builtin().Inventory()
	locals := map[string]bool{"": true, "missing_method": true, "Map": true}
	receivers := map[string]bool{"": true, "can.project.unknown@1::thing": true, "can.std.text@1::from_int": true}
	for _, typ := range inv.Types {
		receivers[typ.Identity] = true
	}
	for _, op := range inv.Operations {
		if op.Kind != "method" || op.Receiver == "" {
			continue
		}
		name := op.Name
		if i := strings.LastIndex(name, "::"); i >= 0 {
			name = name[i+2:]
		} else if i := strings.LastIndex(name, "."); i >= 0 {
			name = name[i+1:]
		}
		locals[name] = true
	}
	matched := 0
	for receiver := range receivers {
		methods := catalogue.Builtin().ReceiverMethods(receiver)
		for local := range locals {
			ref := refNominalMethod(inv, receiver, local)
			var got *catalogue.Operation
			for _, op := range methods {
				name := op.Name
				if i := strings.LastIndex(name, "::"); i >= 0 {
					name = name[i+2:]
				} else if i := strings.LastIndex(name, "."); i >= 0 {
					name = name[i+1:]
				}
				if name == local {
					out := op
					got = &out
					break
				}
			}
			if ref == nil {
				if got != nil {
					t.Fatalf("ReceiverMethods(%q) first-match admitted %s for %q", receiver, got.Name, local)
				}
				continue
			}
			matched++
			if got == nil {
				t.Fatalf("ReceiverMethods(%q) declined %q; reference admits %s", receiver, local, ref.Name)
			}
			if !reflect.DeepEqual(*got, *ref) {
				t.Fatalf("ReceiverMethods(%q) metadata differs for %q", receiver, local)
			}
		}
	}
	if matched == 0 {
		t.Fatal("no nominal catalogue methods exercised")
	}
}

// TestScalarStringMethodLookupEquivalence pins the I24 scalar path:
// CurrentOperation("str."+local) plus the task check must match the old
// name/task scan for every operation-derived local name and negatives.
func TestScalarStringMethodLookupEquivalence(t *testing.T) {
	inv := catalogue.Builtin().Inventory()
	locals := map[string]bool{"": true, "missing": true, "Includes": true, "includes ": true}
	for _, op := range inv.Operations {
		if local, ok := strings.CutPrefix(op.Name, "str."); ok {
			locals[local] = true
		}
		locals[op.Name] = true
	}
	matched := 0
	for local := range locals {
		var ref *catalogue.Operation
		for _, op := range inv.Operations {
			if op.Name == "str."+local && op.Lowering.Task == "I24" {
				out := op
				ref = &out
				break
			}
		}
		op, err := catalogue.Builtin().CurrentOperation("str." + local)
		var got *catalogue.Operation
		if err == nil && op.Lowering.Task == "I24" {
			got = &op
		}
		if ref == nil {
			if got != nil {
				t.Fatalf("scalar lookup admitted str.%s", local)
			}
			continue
		}
		matched++
		if got == nil {
			t.Fatalf("scalar lookup declined str.%s", local)
		}
		if !reflect.DeepEqual(*got, *ref) {
			t.Fatalf("scalar lookup metadata differs for str.%s", local)
		}
	}
	if matched == 0 {
		t.Fatal("no scalar string methods exercised")
	}
}

// refConstructorMetadata is the pre-narrowing inferConstructor catalogue
// scan: record-kind constructible types first, then error declarations.
func refConstructorMetadata(inv catalogue.Inventory, identity string) ([]catalogue.Field, bool) {
	var metadata []catalogue.Field
	found := false
	for _, declaration := range inv.Types {
		if declaration.Identity == identity && declaration.Kind == "record" && declaration.Constructible {
			metadata = declaration.Fields
			found = true
			break
		}
	}
	for _, declaration := range inv.Errors {
		if declaration.Identity == identity {
			metadata = declaration.Fields
			found = true
			break
		}
	}
	return metadata, found
}

func TestConstructorMetadataLookupEquivalence(t *testing.T) {
	inv := catalogue.Builtin().Inventory()
	probes := []string{"", "can.project.unknown@1::thing", "can.std.text@1::does_not_exist"}
	for _, typ := range inv.Types {
		probes = append(probes, typ.Identity)
	}
	for _, decl := range inv.Errors {
		probes = append(probes, decl.Identity)
	}
	matched := 0
	for _, identity := range probes {
		wantFields, wantFound := refConstructorMetadata(inv, identity)
		var gotFields []catalogue.Field
		gotFound := false
		if decl, ok := catalogue.Builtin().TypeByIdentity(identity); ok && decl.Kind == "record" && decl.Constructible {
			gotFields, gotFound = decl.Fields, true
		} else if decl, ok := catalogue.Builtin().ErrorByIdentity(identity); ok {
			gotFields, gotFound = decl.Fields, true
		}
		if wantFound != gotFound {
			t.Fatalf("constructor metadata found=%v, reference %v for %q", gotFound, wantFound, identity)
		}
		if wantFound {
			matched++
			if !reflect.DeepEqual(gotFields, wantFields) {
				t.Fatalf("constructor metadata differs for %q", identity)
			}
			// Opaque and variant catalogue types must never yield
			// constructor fields through this path.
			if decl, ok := catalogue.Builtin().TypeByIdentity(identity); ok && decl.Kind != "record" {
				t.Fatalf("constructor metadata admitted %s kind %s", identity, decl.Kind)
			}
		}
	}
	if matched == 0 {
		t.Fatal("no constructor metadata exercised")
	}
}

// TestArrayPinLookupEquivalence pins that same-instance current-operation
// access returns exactly the metadata the old inventory-pin path served
// for every array operation.
func TestArrayPinLookupEquivalence(t *testing.T) {
	inv := catalogue.Builtin().Inventory()
	matched := 0
	for _, want := range inv.Operations {
		if want.Lowering.Task != "I21" {
			continue
		}
		matched++
		pinned, err := catalogue.Builtin().Operation(want.Name, inv.TargetID, inv.Revision)
		if err != nil {
			t.Fatalf("pinned Operation(%q): %v", want.Name, err)
		}
		current, err := catalogue.Builtin().CurrentOperation(want.Name)
		if err != nil || !reflect.DeepEqual(current, pinned) {
			t.Fatalf("CurrentOperation(%q) differs from pinned path", want.Name)
		}
	}
	if matched == 0 {
		t.Fatal("no I21 array operations in embedded inventory")
	}
}
