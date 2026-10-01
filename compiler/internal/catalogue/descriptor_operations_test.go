package catalogue

import (
	"reflect"
	"testing"
)

// TestDescriptorCatalogueContract pins the NT-I11 descriptor-delivery
// surface: 7 owner-first operations mirroring the F1 owner vocabulary
// (Spawn/CollectStatus/Wait/Kill/Release/Facts/ExpectedAck) over 1 opaque
// launch handle, owner-mirror records, K17 schema-verbatim wire records,
// and the descriptor_fault error. Live behavior stays behind the N
// dispatch (stubbed until P14); rows for live ops are deferred, and the
// Can surface covers the pure contract only.
func TestDescriptorCatalogueContract(t *testing.T) {
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
	launch, ok := c.Type("descriptor::launch")
	if !ok || launch.Kind != "opaque" || launch.Constructible || len(launch.Projections) != 0 {
		t.Fatalf("descriptor::launch marker differs: %+v", launch)
	}
	if err := c.CheckConstructor("descriptor::launch"); err == nil {
		t.Fatal("descriptor::launch admits a public constructor")
	}
	if err := c.CheckProjectPackage("descriptor"); err == nil {
		t.Fatal("descriptor package is not distribution-owned")
	}
	fieldsOf := func(name string) map[string]string {
		t.Helper()
		typ, ok := c.Type(name)
		if !ok {
			t.Fatalf("%s missing", name)
		}
		fields := map[string]string{}
		for _, field := range typ.Fields {
			fields[field.Name] = field.Type
		}
		return fields
	}
	spec := fieldsOf("descriptor::spec")
	if spec["executable"] != "str" || spec["args"] != "str[]" || spec["env"] != "descriptor::env_entry[]" || spec["with_lease"] != "bool" || spec["detached"] != "bool" || spec["dir"] != "str" {
		t.Fatalf("descriptor::spec fields differ: %+v", spec)
	}
	facts := fieldsOf("descriptor::facts")
	if facts["offered"] != "bool" || facts["offered_bytes"] != "int" || facts["writer_done"] != "bool" || facts["writer_err"] != "str" || facts["accepted"] != "bool" || facts["malformed"] != "bool" || facts["eof"] != "bool" || facts["reaped"] != "bool" || facts["child_exit"] != "option::value<descriptor::exit>" {
		t.Fatalf("descriptor::facts fields differ: %+v", facts)
	}
	fd := fieldsOf("descriptor::fd_entry")
	for _, name := range []string{"fd", "owner", "direction", "byte_state", "eof_state", "lifetime", "bytes_offered", "bytes_accepted"} {
		if _, ok := fd[name]; !ok {
			t.Fatalf("descriptor::fd_entry lacks %s: %+v", name, fd)
		}
	}
	if fd["fd"] != "int" || fd["bytes_offered"] != "int" || fd["bytes_accepted"] != "int" {
		t.Fatalf("descriptor::fd_entry count fields differ: %+v", fd)
	}
	env := fieldsOf("descriptor::environment")
	for _, name := range []string{"schema_version", "run_id", "launch_id", "entry", "argv", "env_names", "env_digest", "log_sink"} {
		if _, ok := env[name]; !ok {
			t.Fatalf("descriptor::environment lacks %s: %+v", name, env)
		}
	}
	if env["argv"] != "str[]" || env["env_names"] != "str[]" {
		t.Fatalf("descriptor::environment list fields differ: %+v", env)
	}
	type descriptor struct {
		result string
		inputs []string
		emits  []string
	}
	want := map[string]descriptor{
		"descriptor::launch_child":   {"descriptor::launch", []string{"test::owner", "str", "descriptor::spec"}, []string{"test::stale_handle", "descriptor::descriptor_fault"}},
		"descriptor::collect_status": {"void", []string{"test::owner", "descriptor::launch", "int", "int"}, []string{"test::stale_handle", "descriptor::descriptor_fault"}},
		"descriptor::wait_child":     {"descriptor::exit", []string{"test::owner", "descriptor::launch", "int"}, []string{"test::stale_handle", "descriptor::descriptor_fault"}},
		"descriptor::kill_child":     {"void", []string{"test::owner", "descriptor::launch"}, []string{"test::stale_handle"}},
		"descriptor::release_launch": {"descriptor::report", []string{"test::owner", "descriptor::launch"}, []string{"test::stale_handle", "descriptor::descriptor_fault"}},
		"descriptor::read_facts":     {"descriptor::facts", []string{"test::owner", "descriptor::launch"}, []string{"test::stale_handle"}},
		"descriptor::expected_ack":   {"int[]", []string{"test::owner", "descriptor::launch"}, []string{"test::stale_handle"}},
	}
	if len(want) != 7 {
		t.Fatalf("contract pins %d operations, want 7", len(want))
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
		if op.Inputs[0].Type != "test::owner" {
			t.Fatalf("%s is not owner-first: %+v", name, op.Inputs)
		}
		if !reflect.DeepEqual(op.Emits, descriptor.emits) {
			t.Fatalf("%s emits differ: %+v", name, op.Emits)
		}
		if op.Lowering.Task != "NT-I11" || !reflect.DeepEqual(op.Refs, []string{"NT-I11"}) || op.Assertion != "supplied" {
			t.Fatalf("%s traceability differs: task=%q refs=%v assertion=%q", name, op.Lowering.Task, op.Refs, op.Assertion)
		}
		if len(op.Lowering.Native) == 0 || op.Lowering.Adapter == "" {
			t.Fatalf("%s names no native recipe", name)
		}
	}
}
