package catalogue

import (
	"reflect"
	"testing"
)

func TestCallableAndArmDataSpecialization(t *testing.T) {
	for _, typ := range []string{
		"callable int () emits []",
		"callable void (str) emits [http::timeout]",
		"choice_arm<int> emits []",
		"choice_arm<void> emits [http::timeout]",
		"option::value<callable int (str) emits [http::timeout]>",
		"option::value<choice_arm<int> emits [http::timeout]>",
		"callable callable int (str) emits [] (bool) emits [http::timeout]",
	} {
		t.Run(typ, func(t *testing.T) {
			c := Builtin()
			spec, err := c.Resolve("append", GeneratedTargetID, GeneratedRevision, map[string]string{"T": typ}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if spec.Operation.Inputs[1].Type != typ || spec.Operation.Result != typ+"[]" {
				t.Fatalf("contract lost: %+v", spec.Operation)
			}
		})
	}
}

func TestStructuralSourceBridgeAndNestedSubstitution(t *testing.T) {
	text := "callable choice_arm<option::value<int>> emits [codec::invalid_data] (callable void (str) emits [http::timeout]) emits [http::status_error]"
	contract, err := parseConcreteType(text)
	if err != nil {
		t.Fatal(err)
	}
	if contract.result.name != "choice_arm" || contract.result.result.name != "option::value" || contract.inputs[0].inputs[0].name != "str" || contract.inputs[0].emits[0].name != "http::timeout" || contract.emits[0].name != "http::status_error" {
		t.Fatalf("structure lost: %+v", contract)
	}
	again, err := parseConcreteType(contract.String())
	if err != nil || !reflect.DeepEqual(contract, again) {
		t.Fatalf("structural round-trip failed: %v", err)
	}
	value := substitute("option::value<T[]>", map[string]string{"T": text})
	wrapped, err := parseConcreteType(value)
	if err != nil || !reflect.DeepEqual(wrapped.arguments[0].arguments[0], contract) {
		t.Fatalf("nested substitution lost bounds: %s: %v", value, err)
	}
}

func TestFunctionalDataRetainsWireAndKeyExclusions(t *testing.T) {
	c := Builtin()
	for _, typ := range []string{"callable int () emits []", "choice_arm<int> emits []", "option::value<callable int () emits []>", "option::value<choice_arm<int> emits []>", "callable int () emits [][]"} {
		for _, constraint := range []string{"wire", "map_key", "sort_key", "sql_scalar", "form", "sql_row", "sql_parameters"} {
			if c.admits(typ, constraint, func(string, string) bool { return true }) {
				t.Fatalf("admitted %s as %s", typ, constraint)
			}
		}
		if _, err := c.Resolve("codec::encode_json", GeneratedTargetID, GeneratedRevision, map[string]string{"T": typ}, nil, func(string, string) bool { return true }); err == nil {
			t.Fatalf("wire admitted %s", typ)
		}
		if _, err := c.Resolve("collections::empty_map", GeneratedTargetID, GeneratedRevision, map[string]string{"K": typ, "V": "int"}, nil, func(string, string) bool { return true }); err == nil {
			t.Fatalf("key admitted %s", typ)
		}
	}
}

func TestFunctionalDataRequiresValidComponentsAndErrorKinds(t *testing.T) {
	c := Builtin()
	for _, typ := range []string{
		"callable int ()", "choice_arm<int>", "callable int (void) emits []",
		"callable void[] () emits []", "choice_arm<void[]> emits []",
		"callable int () emits [int]", "callable int () emits [http::header]",
		"callable int () emits [http::invented]", "callable int () emits [http::timeout<int>]",
		"callable int () emits [http::timeout[]]", "callable int () emits [callable int () emits []]",
		"callable http::invented () emits []", "callable int () emits [] trailing",
	} {
		if c.admits(typ, "data", func(string, string) bool { return true }) {
			t.Fatalf("accepted malformed/closed contract %s", typ)
		}
	}
	seen := map[string]string{}
	proof := func(name, constraint string) bool {
		seen[name] = constraint
		return name == "app::payload" && constraint == "data" || name == "app::failed" && constraint == "error"
	}
	typ := "callable app::payload (app::payload) emits [app::failed]"
	if _, err := c.Resolve("append", GeneratedTargetID, GeneratedRevision, map[string]string{"T": typ}, nil, proof); err != nil {
		t.Fatal(err)
	}
	if seen["app::payload"] != "data" || seen["app::failed"] != "error" {
		t.Fatal(seen)
	}
	if c.admits(typ, "data", nil) {
		t.Fatal("unproven project contract admitted")
	}
}

func TestCallbackSpecializationPreservesNestedFunctionalBounds(t *testing.T) {
	c := Builtin()
	typ := "callable int (str,bool) emits [http::timeout]"
	args := map[string]string{"T": typ, "U": "choice_arm<int> emits [http::status_error]"}
	callbacks := map[string]CallbackContract{"callback": {Inputs: []string{"callable int (str, bool) emits [http::timeout]"}, Result: args["U"]}}
	spec, err := c.Resolve("array.map", GeneratedTargetID, GeneratedRevision, args, callbacks, nil)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Operation.Callbacks[0].Inputs[0] != typ || spec.Operation.Callbacks[0].Result != args["U"] || spec.Operation.Result != args["U"]+"[]" {
		t.Fatalf("lost functional contract: %+v", spec)
	}
	callbacks["callback"] = CallbackContract{Inputs: []string{"callable int (str,bool) emits []"}, Result: args["U"]}
	if _, err := c.Resolve("array.map", GeneratedTargetID, GeneratedRevision, args, callbacks, nil); err == nil {
		t.Fatal("nested callable error bound disappeared")
	}
}
